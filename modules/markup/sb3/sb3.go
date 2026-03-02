// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package sb3

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"strings"

	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/setting"
)

func init() {
	markup.RegisterRenderer(Renderer{})
}

type Renderer struct{}

// SB3 project structure
type SB3Project struct {
	Targets []struct {
		Name     string         `json:"name"`
		Blocks   map[string]any `json:"blocks"`
		Costumes []struct {
			Name string `json:"name"`
		} `json:"costumes"`
		Sounds []struct {
			Name string `json:"name"`
		} `json:"sounds"`
	} `json:"targets"`
}

func (Renderer) Name() string {
	return "sb3"
}

func (Renderer) FileNamePatterns() []string {
	return []string{"*.sb3"}
}

func (Renderer) SanitizerRules() []setting.MarkupSanitizerRule {
	return []setting.MarkupSanitizerRule{
		{Element: "div", AllowAttr: "class", Regexp: `^sb3-preview$`},
		{Element: "div", AllowAttr: "class", Regexp: `^sb3-sprite$`},
		{Element: "div", AllowAttr: "class", Regexp: `^sb3-script$`},
		{Element: "pre", AllowAttr: "class", Regexp: `^blocks$`},
		{Element: "svg", AllowAttr: "class", Regexp: `^sb3-blocks$`},
	}
}

// extractAndParseSB3 extracts project.json from a .sb3 file
func extractAndParseSB3(input io.Reader) (*SB3Project, error) {
	// Read the SB3 file content
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// Register decompressor for deflate (standard zip compression)
	zip.RegisterDecompressor(8, func(r io.Reader) io.ReadCloser {
		return flate.NewReader(r)
	})

	// Open the zip file
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	// Find project.json
	for _, file := range zipReader.File {
		if file.Name == "project.json" {
			// Open the file
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			// Read and parse JSON
			projectData, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			var project SB3Project
			if err := json.Unmarshal(projectData, &project); err != nil {
				return nil, err
			}

			return &project, nil
		}
	}

	return nil, nil
}

// convertBlocksToScratchBlocks converts SB3 blocks to ScratchBlocks text format
func convertBlocksToScratchBlocks(blocks map[string]any, blockID string, indent int) string {
	result := ""

	// Find the block with the given ID
	block, exists := blocks[blockID]
	if !exists {
		return result
	}

	blockMap, ok := block.(map[string]any)
	if !ok {
		return result
	}

	// Get opcode
	opcode, _ := blockMap["opcode"].(string)

	// Generate indentation
	indentStr := strings.Repeat("    ", indent)

	// Convert opcode to scratchblocks syntax
	result += indentStr + convertOpcodeToScratchBlocks(opcode, blockMap, blocks) + "\n"

	// Handle next block (stack)
	if next, ok := blockMap["next"].([]any); ok && len(next) > 0 {
		if nextID, ok := next[0].(string); ok {
			result += convertBlocksToScratchBlocks(blocks, nextID, indent)
		}
	}

	return result
}

// convertOpcodeToScratchBlocks converts a single block opcode to ScratchBlocks syntax
func convertOpcodeToScratchBlocks(opcode string, block map[string]any, allBlocks map[string]any) string {
	// Handle common opcodes
	switch opcode {
	case "event_whenflagclicked":
		return "when flag clicked"
	case "event_whenkeypressed":
		if inputs, ok := block["inputs"].(map[string]any); ok {
			if keyOption, ok := inputs["KEY_OPTION"].([]any); ok && len(keyOption) > 1 {
				if keyVal, ok := keyOption[1].(string); ok {
					return "when [" + keyVal + " v] key pressed"
				}
			}
		}
		return "when [space v] key pressed"
	case "looks_say":
		if inputs, ok := block["inputs"].(map[string]any); ok {
			if message, ok := inputs["MESSAGE"].([]any); ok && len(message) > 1 {
				if msgVal, ok := message[1].(string); ok {
					return "say [" + msgVal + "] for (2) secs"
				}
			}
		}
		return "say [Hello!]"
	case "motion_movesteps":
		if inputs, ok := block["inputs"].(map[string]any); ok {
			if steps, ok := inputs["STEPS"].([]any); ok && len(steps) > 1 {
				if stepsVal, ok := steps[1].(float64); ok {
					return "move (" + formatNumber(stepsVal) + ") steps"
				}
			}
		}
		return "move (10) steps"
	case "control_forever":
		body := getNestedBlocks(block, allBlocks)
		return "forever\n" + body + "end"
	case "control_repeat":
		if inputs, ok := block["inputs"].(map[string]any); ok {
			if times, ok := inputs["TIMES"].([]any); ok && len(times) > 1 {
				if timesVal, ok := times[1].(float64); ok {
					body := getNestedBlocks(block, allBlocks)
					return "repeat (" + formatNumber(timesVal) + ")\n" + body + "end"
				}
			}
		}
		return "repeat (10)\nend"
	case "control_if":
		if inputs, ok := block["inputs"].(map[string]any); ok {
			if condition, ok := inputs["CONDITION"].([]any); ok && len(condition) > 1 {
				body := getNestedBlocks(block, allBlocks)
				return "if <> then\n" + body + "end"
			}
		}
		return "if <> then\nend"
	case "operator_add":
		if inputs, ok := block["inputs"].(map[string]any); ok {
			if num1, ok := inputs["NUM1"].([]any); ok && len(num1) > 1 {
				if num2, ok := inputs["NUM2"].([]any); ok && len(num2) > 1 {
					return "(" + getLiteralValue(num1[1]) + " + " + getLiteralValue(num2[1]) + ")"
				}
			}
		}
		return "((1) + (1))"
	}

	// Default: try to convert opcode to readable format
	return "[" + opcode + "]"
}

// getNestedBlocks extracts nested blocks from a C block (like forever, repeat, if)
func getNestedBlocks(block map[string]any, allBlocks map[string]any) string {
	result := ""
	if inputs, ok := block["inputs"].(map[string]any); ok {
		if substack, ok := inputs["SUBSTACK"].([]any); ok && len(substack) > 0 {
			if substackID, ok := substack[0].(string); ok && substackID != "" {
				result += convertBlocksToScratchBlocks(allBlocks, substackID, 1)
			}
		}
	}
	return result
}

// formatNumber formats a number for scratchblocks
func formatNumber(num float64) string {
	return fmt.Sprintf("%.0f", num)
}

// getLiteralValue extracts a literal value from an input
func getLiteralValue(value any) string {
	switch v := value.(type) {
	case float64:
		return formatNumber(v)
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// Render implements markup.Renderer
func (r Renderer) Render(ctx *markup.RenderContext, input io.Reader, output io.Writer) error {
	project, err := extractAndParseSB3(input)
	if err != nil {
		return err
	}

	if project == nil {
		return nil
	}

	// Start HTML output
	if _, err := io.WriteString(output, `<div class="sb3-preview">`); err != nil {
		return err
	}

	// Iterate through all targets (sprites/stage)
	for _, target := range project.Targets {
		if len(target.Blocks) == 0 {
			continue
		}

		// Sprite header
		spriteName := html.EscapeString(target.Name)
		if spriteName == "" {
			spriteName = "Stage"
		}

		if _, err := io.WriteString(output, `<div class="sb3-sprite"><h4>`); err != nil {
			return err
		}
		if _, err := io.WriteString(output, spriteName); err != nil {
			return err
		}
		if _, err := io.WriteString(output, `</h4>`); err != nil {
			return err
		}

		// Find hat blocks (event blocks that start scripts)
		scriptCount := 0
		for blockID, block := range target.Blocks {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}

			opcode, _ := blockMap["opcode"].(string)

			// Check if this is a hat block (starts a script)
			if isHatBlock(opcode) && blockMap["next"] != nil {
				scriptCount++

				if _, err := io.WriteString(output, `<div class="sb3-script"><pre class="blocks">`); err != nil {
					return err
				}

				// Convert the script to ScratchBlocks format
				scratchBlocksText := convertBlocksToScratchBlocks(target.Blocks, blockID, 0)
				if _, err := io.WriteString(output, html.EscapeString(scratchBlocksText)); err != nil {
					return err
				}

				if _, err := io.WriteString(output, `</pre></div>`); err != nil {
					return err
				}
			}
		}

		if scriptCount == 0 {
			if _, err := io.WriteString(output, `<p class="tw-text-muted">No scripts found</p>`); err != nil {
				return err
			}
		}

		if _, err := io.WriteString(output, `</div>`); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(output, `</div>`); err != nil {
		return err
	}

	return nil
}

// isHatBlock checks if an opcode is a hat block (starts a script)
func isHatBlock(opcode string) bool {
	hatBlocks := map[string]bool{
		"event_whenflagclicked":        true,
		"event_whenkeypressed":         true,
		"event_whenbackdropswitchesto": true,
		"event_whengreaterthan":        true,
		"event_whenbroadcastreceived":  true,
		"control_start_as_clone":       true,
	}
	return hatBlocks[opcode]
}
