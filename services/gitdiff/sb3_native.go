// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitdiff

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Sb3NativeDiff provides native Go implementation of sb3-diff functionality
// This avoids external command dependencies and improves performance

// RawProject represents the raw project.json structure from sb3 files
type RawProject struct {
	Targets    []RawTarget `json:"targets"`
	Monitors   []any       `json:"monitors"`
	Extensions []string    `json:"extensions"`
	Meta       struct {
		Semver string `json:"semver"`
		VM     string `json:"vm"`
		Agent  string `json:"agent"`
	} `json:"meta"`
}

// RawTarget represents a target in the Scratch project
type RawTarget struct {
	IsStage    bool                `json:"isStage"`
	Name       string              `json:"name"`
	Variables  map[string][]any    `json:"variables"`
	Lists      map[string][]any    `json:"lists"`
	Broadcasts map[string]string   `json:"broadcasts"`
	Blocks     map[string]RawBlock `json:"blocks"`
	Comments   map[string]any      `json:"comments"`
	Costumes   []any               `json:"costumes"`
	Sounds     []any               `json:"sounds"`
	Volume     float64             `json:"volume"`
	LayerOrder int                 `json:"layerOrder"`
	Visible    *bool               `json:"visible,omitempty"`
	X          *float64            `json:"x,omitempty"`
	Y          *float64            `json:"y,omitempty"`
	Size       *float64            `json:"size,omitempty"`
	Direction  *float64            `json:"direction,omitempty"`
}

// RawBlock represents a block in Scratch
type RawBlock struct {
	Opcode   string              `json:"opcode"`
	Next     *string             `json:"next,omitempty"`
	Parent   *string             `json:"parent,omitempty"`
	Inputs   map[string][]any    `json:"inputs"`
	Fields   map[string][]string `json:"fields"`
	Shadow   bool                `json:"shadow"`
	TopLevel bool                `json:"topLevel"`
	X        *float64            `json:"x,omitempty"`
	Y        *float64            `json:"y,omitempty"`
	Mutation any                 `json:"mutation,omitempty"`
}

// ScriptText represents a script in text format for diffing
type ScriptText struct {
	Text        string      `json:"text"`
	Fingerprint string      `json:"fingerprint"`
	Blocks      []BlockText `json:"blocks"`
	Original    any         `json:"original"`
	TopID       string      `json:"topId"`
}

// BlockText represents a block in text format
type BlockText struct {
	Text        string `json:"text"`
	Fingerprint string `json:"fingerprint"`
	Depth       int    `json:"depth"`
	Original    any    `json:"original"`
}

// CreateSb3DiffNative creates a semantic diff for Scratch .sb3 project files using native Go implementation
func CreateSb3DiffNative(diffFile *DiffFile, baseReader, headReader io.Reader) (*Sb3DiffResult, error) {
	// Parse old project
	var oldProject *RawProject
	var err error
	if baseReader != nil {
		oldProject, err = parseSb3Project(baseReader)
		if err != nil {
			return &Sb3DiffResult{Error: fmt.Sprintf("failed to parse base file: %v", err)}, nil
		}
	}

	// Parse new project
	var newProject *RawProject
	if headReader != nil {
		newProject, err = parseSb3Project(headReader)
		if err != nil {
			return &Sb3DiffResult{Error: fmt.Sprintf("failed to parse head file: %v", err)}, nil
		}
	}

	// Handle single file scenarios
	if oldProject == nil && newProject != nil {
		// File was added - create empty base project for comparison
		oldProject = createEmptyProject()
	} else if oldProject != nil && newProject == nil {
		// File was deleted - create empty head project for comparison
		newProject = createEmptyProject()
	}

	// Generate diff
	diff, err := generateNativeDiff(oldProject, newProject)
	if err != nil {
		return &Sb3DiffResult{Error: fmt.Sprintf("failed to generate diff: %v", err)}, nil
	}

	return &Sb3DiffResult{Diff: diff}, nil
}

// parseSb3Project parses an sb3 file and extracts project.json
func parseSb3Project(reader io.Reader) (*RawProject, error) {
	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read sb3 file: %w", err)
	}

	// Open as zip
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open sb3 file as zip: %w", err)
	}

	// Find project.json
	var projectJsonFile *zip.File
	for _, f := range zipReader.File {
		if f.Name == "project.json" {
			projectJsonFile = f
			break
		}
	}

	if projectJsonFile == nil {
		return nil, fmt.Errorf("project.json not found in sb3 file")
	}

	// Read project.json
	rc, err := projectJsonFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open project.json: %w", err)
	}
	defer rc.Close()

	projectData, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read project.json: %w", err)
	}

	// Parse JSON
	var project RawProject
	if err := json.Unmarshal(projectData, &project); err != nil {
		return nil, fmt.Errorf("failed to parse project.json: %w", err)
	}

	return &project, nil
}

// createEmptyProject creates an empty Scratch project for comparison
func createEmptyProject() *RawProject {
	return &RawProject{
		Targets: []RawTarget{
			{
				IsStage:    true,
				Name:       "Stage",
				Variables:  make(map[string][]any),
				Lists:      make(map[string][]any),
				Broadcasts: make(map[string]string),
				Blocks:     make(map[string]RawBlock),
				Comments:   make(map[string]any),
				Costumes:   []any{},
				Sounds:     []any{},
				Volume:     100,
				LayerOrder: 0,
			},
		},
		Monitors:   []any{},
		Extensions: []string{},
	}
}

// generateNativeDiff generates a diff between two Scratch projects
func generateNativeDiff(oldProject, newProject *RawProject) (*Sb3Diff, error) {
	diff := &Sb3Diff{
		Summary: Sb3DiffSummary{},
		Items:   []Sb3DiffItem{},
	}

	// Create target maps
	oldTargets := make(map[string]RawTarget)
	newTargets := make(map[string]RawTarget)

	for _, t := range oldProject.Targets {
		oldTargets[t.Name] = t
	}
	for _, t := range newProject.Targets {
		newTargets[t.Name] = t
	}

	// Find all target names
	allTargetNames := make(map[string]bool)
	for name := range oldTargets {
		allTargetNames[name] = true
	}
	for name := range newTargets {
		allTargetNames[name] = true
	}

	// Compare each target
	for targetName := range allTargetNames {
		oldTarget, oldExists := oldTargets[targetName]
		newTarget, newExists := newTargets[targetName]

		if oldExists && newExists {
			// Target exists in both - compare contents
			compareTargets(diff, targetName, &oldTarget, &newTarget)
		} else if oldExists && !newExists {
			// Target was deleted
			diff.Summary.TargetsDeleted++
			oldJSON := marshalJSON(oldTarget)
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemTarget,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				Old: &oldJSON,
			})
		} else if !oldExists && newExists {
			// Target was added
			diff.Summary.TargetsAdded++
			newJSON := marshalJSON(newTarget)
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemTarget,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				New: &newJSON,
			})
		}
	}

	return diff, nil
}

// compareTargets compares two targets and adds diff items
func compareTargets(diff *Sb3Diff, targetName string, oldTarget, newTarget *RawTarget) {
	// Compare variables
	compareVariables(diff, targetName, oldTarget.Variables, newTarget.Variables)

	// Compare lists
	compareLists(diff, targetName, oldTarget.Lists, newTarget.Lists)

	// Compare costumes
	compareCostumes(diff, targetName, oldTarget.Costumes, newTarget.Costumes)

	// Compare sounds
	compareSounds(diff, targetName, oldTarget.Sounds, newTarget.Sounds)

	// Compare scripts
	compareScripts(diff, targetName, oldTarget.Blocks, newTarget.Blocks)
}

// compareVariables compares variables between two targets
func compareVariables(diff *Sb3Diff, targetName string, oldVars, newVars map[string][]any) {
	allVarNames := make(map[string]bool)
	for name := range oldVars {
		allVarNames[name] = true
	}
	for name := range newVars {
		allVarNames[name] = true
	}

	for varName := range allVarNames {
		oldVal, oldExists := oldVars[varName]
		newVal, newExists := newVars[varName]

		if oldExists && newExists {
			// Variable exists in both - check if changed
			oldJSON := marshalJSON(oldVal)
			newJSON := marshalJSON(newVal)
			if !bytes.Equal(oldJSON, newJSON) {
				diff.Summary.VariablesModified++
				diff.Items = append(diff.Items, Sb3DiffItem{
					Type: Sb3DiffItemVariable,
					Location: Sb3DiffLocation{
						TargetName: targetName,
					},
					Old: &oldJSON,
					New: &newJSON,
				})
			}
		} else if oldExists && !newExists {
			// Variable was deleted
			oldJSON := marshalJSON(oldVal)
			diff.Summary.VariablesDeleted++
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemVariable,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				Old: &oldJSON,
			})
		} else if !oldExists && newExists {
			// Variable was added
			newJSON := marshalJSON(newVal)
			diff.Summary.VariablesAdded++
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemVariable,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				New: &newJSON,
			})
		}
	}
}

// compareLists compares lists between two targets
func compareLists(diff *Sb3Diff, targetName string, oldLists, newLists map[string][]any) {
	allListNames := make(map[string]bool)
	for name := range oldLists {
		allListNames[name] = true
	}
	for name := range newLists {
		allListNames[name] = true
	}

	for listName := range allListNames {
		oldVal, oldExists := oldLists[listName]
		newVal, newExists := newLists[listName]

		if oldExists && newExists {
			// List exists in both - check if changed
			oldJSON := marshalJSON(oldVal)
			newJSON := marshalJSON(newVal)
			if !bytes.Equal(oldJSON, newJSON) {
				diff.Summary.ListsModified++
				diff.Items = append(diff.Items, Sb3DiffItem{
					Type: Sb3DiffItemList,
					Location: Sb3DiffLocation{
						TargetName: targetName,
					},
					Old: &oldJSON,
					New: &newJSON,
				})
			}
		} else if oldExists && !newExists {
			// List was deleted
			oldJSON := marshalJSON(oldVal)
			diff.Summary.ListsDeleted++
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemList,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				Old: &oldJSON,
			})
		} else if !oldExists && newExists {
			// List was added
			newJSON := marshalJSON(newVal)
			diff.Summary.ListsAdded++
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemList,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				New: &newJSON,
			})
		}
	}
}

// compareCostumes compares costumes between two targets
func compareCostumes(diff *Sb3Diff, targetName string, oldCostumes, newCostumes []any) {
	oldMap := make(map[string]any)
	newMap := make(map[string]any)

	for _, c := range oldCostumes {
		if costumeMap, ok := c.(map[string]any); ok {
			if md5ext, ok := costumeMap["md5ext"].(string); ok {
				oldMap[md5ext] = c
			}
		}
	}
	for _, c := range newCostumes {
		if costumeMap, ok := c.(map[string]any); ok {
			if md5ext, ok := costumeMap["md5ext"].(string); ok {
				newMap[md5ext] = c
			}
		}
	}

	// Find added costumes
	for md5ext, costume := range newMap {
		if _, exists := oldMap[md5ext]; !exists {
			diff.Summary.CostumesAdded++
			costumeJSON := marshalJSON(costume)
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemCostume,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				New: &costumeJSON,
			})
		}
	}

	// Find deleted costumes
	for md5ext, costume := range oldMap {
		if _, exists := newMap[md5ext]; !exists {
			diff.Summary.CostumesDeleted++
			costumeJSON := marshalJSON(costume)
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemCostume,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				Old: &costumeJSON,
			})
		}
	}
}

// compareSounds compares sounds between two targets
func compareSounds(diff *Sb3Diff, targetName string, oldSounds, newSounds []any) {
	oldMap := make(map[string]any)
	newMap := make(map[string]any)

	for _, s := range oldSounds {
		if soundMap, ok := s.(map[string]any); ok {
			if md5ext, ok := soundMap["md5ext"].(string); ok {
				oldMap[md5ext] = s
			}
		}
	}
	for _, s := range newSounds {
		if soundMap, ok := s.(map[string]any); ok {
			if md5ext, ok := soundMap["md5ext"].(string); ok {
				newMap[md5ext] = s
			}
		}
	}

	// Find added sounds
	for md5ext, sound := range newMap {
		if _, exists := oldMap[md5ext]; !exists {
			diff.Summary.SoundsAdded++
			soundJSON := marshalJSON(sound)
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemSound,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				New: &soundJSON,
			})
		}
	}

	// Find deleted sounds
	for md5ext, sound := range oldMap {
		if _, exists := newMap[md5ext]; !exists {
			diff.Summary.SoundsDeleted++
			soundJSON := marshalJSON(sound)
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemSound,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				Old: &soundJSON,
			})
		}
	}
}

// compareScripts compares scripts between two targets
func compareScripts(diff *Sb3Diff, targetName string, oldBlocks, newBlocks map[string]RawBlock) {
	// Find top-level blocks
	oldTopBlocks := findTopLevelBlocks(oldBlocks)
	newTopBlocks := findTopLevelBlocks(newBlocks)

	// Parse scripts into text format
	oldScripts := parseScripts(oldBlocks, oldTopBlocks)
	newScripts := parseScripts(newBlocks, newTopBlocks)

	// Match scripts by fingerprint
	oldScriptsMap := make(map[string]*ScriptText)
	newScriptsMap := make(map[string]*ScriptText)

	for i := range oldScripts {
		fp := generateFingerprint(oldScripts[i].Text)
		oldScriptsMap[fp] = &oldScripts[i]
	}

	for i := range newScripts {
		fp := generateFingerprint(newScripts[i].Text)
		newScriptsMap[fp] = &newScripts[i]
	}

	// Find modified scripts
	for fp, oldScript := range oldScriptsMap {
		if newScript, exists := newScriptsMap[fp]; exists {
			// Scripts match by fingerprint - check if content changed
			if oldScript.Text != newScript.Text {
				diff.Summary.ScriptsModified++
				oldJSON := marshalJSON(oldScript)
				newJSON := marshalJSON(newScript)
				diff.Items = append(diff.Items, Sb3DiffItem{
					Type: Sb3DiffItemScript,
					Location: Sb3DiffLocation{
						TargetName:  targetName,
						ScriptIndex: &[]int{0}[0], // Will be updated
					},
					Old: &oldJSON,
					New: &newJSON,
				})
			}
			delete(newScriptsMap, fp)
		} else {
			// Script was deleted
			oldJSON := marshalJSON(oldScript)
			diff.Summary.ScriptsDeleted++
			diff.Items = append(diff.Items, Sb3DiffItem{
				Type: Sb3DiffItemScript,
				Location: Sb3DiffLocation{
					TargetName: targetName,
				},
				Old: &oldJSON,
			})
		}
	}

	// Find added scripts
	for _, newScript := range newScriptsMap {
		newJSON := marshalJSON(newScript)
		diff.Summary.ScriptsAdded++
		diff.Items = append(diff.Items, Sb3DiffItem{
			Type: Sb3DiffItemScript,
			Location: Sb3DiffLocation{
				TargetName: targetName,
			},
			New: &newJSON,
		})
	}
}

// findTopLevelBlocks finds all top-level blocks
func findTopLevelBlocks(blocks map[string]RawBlock) []string {
	var topLevelIDs []string
	for id, block := range blocks {
		if block.TopLevel && !block.Shadow {
			topLevelIDs = append(topLevelIDs, id)
		}
	}
	return topLevelIDs
}

// parseScripts parses scripts from blocks
func parseScripts(blocks map[string]RawBlock, topLevelIDs []string) []ScriptText {
	var scripts []ScriptText

	for _, topID := range topLevelIDs {
		script := parseScript(blocks, topID)
		scripts = append(scripts, script)
	}

	return scripts
}

// parseScript parses a single script from a top-level block using improved V2 parser
func parseScript(blocks map[string]RawBlock, topID string) ScriptText {
	// Use the improved V2 parser
	parser := NewSB3ParserV2(ParserOptions{
		Locale:        "en",
		Tabs:          "    ",
		VariableStyle: VariableStyleNone,
	})

	// Parse all blocks
	if err := parser.ParseBlocks(blocks); err != nil {
		// Fall back to old parser if V2 fails
		return parseScriptLegacy(blocks, topID)
	}

	// Convert script to ScratchBlocks syntax
	scriptText := parser.ToScratchblocks(topID)
	
	// Debug: Check if the returned text has newlines
	// The parser should always add newlines between blocks
	if len(scriptText) > 0 && !strings.Contains(scriptText, "\n") {
		// This indicates a problem with the parser - it returned text without newlines
		// Force rebuild with newlines as fallback
		scriptText = rebuildScriptWithNewlines(blocks, topID, parser)
	}

	// Parse individual blocks for diff tracking using V2 parser
	var blocksText []BlockText
	currentID := topID
	depth := 0

	for currentID != "" {
		block, exists := blocks[currentID]
		if !exists {
			break
		}

		// Use V2 parser for individual block text to get proper variable/list formatting
		blockText := generateBlockTextV2(currentID, block, parser)
		blocksText = append(blocksText, BlockText{
			Text:        blockText,
			Fingerprint: generateFingerprint(blockText),
			Depth:       depth,
			Original:    block,
		})

		// Update depth for C-blocks
		if isCBlockOpcode(block.Opcode) {
			depth++
		} else if isEBlockOpcode(block.Opcode) {
			// E-blocks handle else substack
			depth++
		} else if block.Parent != nil {
			// Check if this block is inside a C-block
			if parentBlock, parentExists := blocks[*block.Parent]; parentExists {
				if isCBlockOpcode(parentBlock.Opcode) || isEBlockOpcode(parentBlock.Opcode) {
					depth = 1 // Inside a C-block
				}
			}
		}

		if block.Next != nil {
			currentID = *block.Next
		} else {
			currentID = ""
		}
	}

	return ScriptText{
		Text:        strings.TrimSpace(scriptText),
		Fingerprint: generateFingerprint(scriptText),
		Blocks:      blocksText,
		TopID:       topID,
	}
}

// generateBlockTextV2 generates block text using the V2 parser
func generateBlockTextV2(blockID string, block RawBlock, parser *SB3ParserV2) string {
	// Get the parsed block from the parser using block ID
	if connectable, exists := parser.blocks[blockID]; exists {
		return connectable.ToScratchblocks(parser.options)
	}
	
	// Fallback to legacy parser
	return generateBlockTextLegacy(block)
}

// parseScriptLegacy parses a script using the legacy parser (fallback)
func parseScriptLegacy(blocks map[string]RawBlock, topID string) ScriptText {
	var textBuilder strings.Builder
	var blocksText []BlockText

	currentID := topID
	depth := 0

	for currentID != "" {
		block, exists := blocks[currentID]
		if !exists {
			break
		}

		// Generate block text
		blockText := generateBlockTextLegacy(block)
		blocksText = append(blocksText, BlockText{
			Text:        blockText,
			Fingerprint: generateFingerprint(blockText),
			Depth:       depth,
			Original:    block,
		})

		// For sequential blocks, we don't increase depth
		textBuilder.WriteString(strings.Repeat("  ", depth))
		textBuilder.WriteString(blockText)
		textBuilder.WriteString("\n")

		if block.Next != nil {
			currentID = *block.Next
		} else {
			currentID = ""
		}
	}

	scriptText := textBuilder.String()
	return ScriptText{
		Text:        scriptText,
		Fingerprint: generateFingerprint(scriptText),
		Blocks:      blocksText,
		TopID:       topID,
	}
}

// generateBlockTextLegacy generates block text using the legacy parser
func generateBlockTextLegacy(block RawBlock) string {
	parser := NewScratchBlocksParser("en")
	return parser.ConvertBlock(&block)
}

// rebuildScriptWithNewlines rebuilds a script with proper newlines from the parser
func rebuildScriptWithNewlines(blocks map[string]RawBlock, topID string, parser *SB3ParserV2) string {
	var result strings.Builder
	currentID := topID
	indentLevel := 0

	for currentID != "" {
		block, exists := blocks[currentID]
		if !exists {
			break
		}

		// Get block text from parser
		if connectable, ok := parser.blocks[currentID]; ok {
			blockText := connectable.ToScratchblocks(parser.options)
			result.WriteString(strings.Repeat(parser.options.Tabs, indentLevel))
			result.WriteString(blockText)
			result.WriteString("\n")
		}

		// Handle C-blocks with substacks
		if isCBlockOpcode(block.Opcode) || isEBlockOpcode(block.Opcode) {
			substackID := parser.getSubstackID(block, "SUBSTACK")
			if substackID != "" {
				indentLevel++
				substackText := rebuildSubstackWithNewlines(blocks, substackID, parser, indentLevel)
				result.WriteString(substackText)
				indentLevel--
			}
			substackID2 := parser.getSubstackID(block, "SUBSTACK2")
			if substackID2 != "" {
				indentLevel++
				result.WriteString(strings.Repeat(parser.options.Tabs, indentLevel))
				result.WriteString("else\n")
				substackText := rebuildSubstackWithNewlines(blocks, substackID2, parser, indentLevel)
				result.WriteString(substackText)
				indentLevel--
			}
		}

		if block.Next != nil {
			currentID = *block.Next
		} else {
			currentID = ""
		}
	}

	return result.String()
}

// rebuildSubstackWithNewlines rebuilds a substack with proper newlines
func rebuildSubstackWithNewlines(blocks map[string]RawBlock, substackID string, parser *SB3ParserV2, indentLevel int) string {
	var result strings.Builder
	currentID := substackID

	for currentID != "" {
		if connectable, ok := parser.blocks[currentID]; ok {
			blockText := connectable.ToScratchblocks(parser.options)
			result.WriteString(strings.Repeat(parser.options.Tabs, indentLevel))
			result.WriteString(blockText)
			result.WriteString("\n")

			// Handle nested C-blocks
			if block, exists := blocks[currentID]; exists {
				if isCBlockOpcode(block.Opcode) || isEBlockOpcode(block.Opcode) {
					subSubstackID := parser.getSubstackID(block, "SUBSTACK")
					if subSubstackID != "" {
						indentLevel++
						substackText := rebuildSubstackWithNewlines(blocks, subSubstackID, parser, indentLevel)
						result.WriteString(substackText)
						indentLevel--
					}
					subSubstackID2 := parser.getSubstackID(block, "SUBSTACK2")
					if subSubstackID2 != "" {
						indentLevel++
						result.WriteString(strings.Repeat(parser.options.Tabs, indentLevel))
						result.WriteString("else\n")
						substackText := rebuildSubstackWithNewlines(blocks, subSubstackID2, parser, indentLevel)
						result.WriteString(substackText)
						indentLevel--
					}
				}
			}
		}

		if block, exists := blocks[currentID]; exists && block.Next != nil {
			currentID = *block.Next
		} else {
			currentID = ""
		}
	}

	return result.String()
}

// isCBlockOpcode checks if an opcode is for a C-block
func isCBlockOpcode(opcode string) bool {
	cBlocks := map[string]bool{
		"control_if":           true,
		"control_repeat":       true,
		"control_repeat_until": true,
		"control_forever":      true,
		"control_for_each":     true,
	}
	return cBlocks[opcode]
}

// isEBlockOpcode checks if an opcode is for an E-block (if-else)
func isEBlockOpcode(opcode string) bool {
	return opcode == "control_if_else"
}

// generateFingerprint generates a fingerprint from text
func generateFingerprint(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])[:16]
}

// marshalJSON marshals a value to JSON.RawMessage
func marshalJSON(val any) json.RawMessage {
	data, _ := json.Marshal(val)
	return data
}
