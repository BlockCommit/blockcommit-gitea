// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitdiff

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Sb3DiffType represents the type of a Scratch project change.
type Sb3DiffType string

// Sb3DiffType possible values.
const (
	Sb3DiffTypeAdd    Sb3DiffType = "add"
	Sb3DiffTypeDelete Sb3DiffType = "delete"
	Sb3DiffTypeModify Sb3DiffType = "modify"
)

// Sb3DiffItemType represents the type of item changed in a Scratch project.
type Sb3DiffItemType string

// Sb3DiffItemType possible values.
const (
	Sb3DiffItemTarget   Sb3DiffItemType = "target"
	Sb3DiffItemScript   Sb3DiffItemType = "script"
	Sb3DiffItemVariable Sb3DiffItemType = "variable"
	Sb3DiffItemList     Sb3DiffItemType = "list"
	Sb3DiffItemCostume  Sb3DiffItemType = "costume"
	Sb3DiffItemSound    Sb3DiffItemType = "sound"
	Sb3DiffItemBlock    Sb3DiffItemType = "block"
)

// Sb3DiffLocation represents the location of a change in a Scratch project.
type Sb3DiffLocation struct {
	TargetName  string `json:"targetName"`
	ScriptIndex *int   `json:"scriptIndex,omitempty"`
	BlockIndex  *int   `json:"blockIndex,omitempty"`
	BlockPath   string `json:"blockPath,omitempty"`
}

// Sb3DiffItem represents a single change item in a Scratch project diff.
type Sb3DiffItem struct {
	Type        Sb3DiffItemType  `json:"type"`
	Location    Sb3DiffLocation  `json:"location"`
	Old         *json.RawMessage `json:"old,omitempty"`
	New         *json.RawMessage `json:"new,omitempty"`
	Fingerprint string           `json:"fingerprint,omitempty"`
}

// GetFormattedOld returns a formatted, readable version of the old value.
func (i *Sb3DiffItem) GetFormattedOld() string {
	if i.Old == nil {
		return ""
	}
	var data any
	if err := json.Unmarshal(*i.Old, &data); err != nil {
		return string(*i.Old)
	}
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return string(*i.Old)
	}
	return string(formatted)
}

// GetFormattedNew returns a formatted, readable version of the new value.
func (i *Sb3DiffItem) GetFormattedNew() string {
	if i.New == nil {
		return ""
	}
	var data any
	if err := json.Unmarshal(*i.New, &data); err != nil {
		return string(*i.New)
	}
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return string(*i.New)
	}
	return string(formatted)
}

// GetDisplayLabel returns a human-readable label for the item type.
func (i *Sb3DiffItem) GetDisplayLabel() string {
	switch i.Type {
	case Sb3DiffItemTarget:
		return "角色"
	case Sb3DiffItemVariable:
		return "变量"
	case Sb3DiffItemList:
		return "列表"
	case Sb3DiffItemCostume:
		return "造型"
	case Sb3DiffItemSound:
		return "声音"
	case Sb3DiffItemScript:
		return "脚本"
	case Sb3DiffItemBlock:
		return "积木"
	default:
		return string(i.Type)
	}
}

// GetScratchBlocksOld returns the old value in ScratchBlocks syntax format for visualization
func (i *Sb3DiffItem) GetScratchBlocksOld() string {
	if i.Old == nil {
		return ""
	}

	// For script type, try to parse as ScriptText to get ScratchBlocks syntax
	if i.Type == Sb3DiffItemScript {
		var script ScriptText
		if err := json.Unmarshal(*i.Old, &script); err == nil {
			// Return the scratchblocks syntax
			return script.Text
		}
	}

	// For block type, try to parse as RawBlock and convert to scratchblocks
	if i.Type == Sb3DiffItemBlock {
		var rawBlock RawBlock
		if err := json.Unmarshal(*i.Old, &rawBlock); err == nil {
			parser := NewScratchBlocksParser("en")
			return parser.ConvertBlock(&rawBlock)
		}
	}

	// For other types, try to format as readable text
	return i.formatReadableText(i.Old, i.Type)
}

// GetScratchBlocksNew returns the new value in ScratchBlocks syntax format for visualization
func (i *Sb3DiffItem) GetScratchBlocksNew() string {
	if i.New == nil {
		return ""
	}

	// For script type, try to parse as ScriptText to get ScratchBlocks syntax
	if i.Type == Sb3DiffItemScript {
		var script ScriptText
		if err := json.Unmarshal(*i.New, &script); err == nil {
			// Return the scratchblocks syntax
			return script.Text
		}
	}

	// For block type, try to parse as RawBlock and convert to scratchblocks
	if i.Type == Sb3DiffItemBlock {
		var rawBlock RawBlock
		if err := json.Unmarshal(*i.New, &rawBlock); err == nil {
			parser := NewScratchBlocksParser("en")
			return parser.ConvertBlock(&rawBlock)
		}
	}

	// For other types, try to format as readable text
	return i.formatReadableText(i.New, i.Type)
}

// GetScratchBlocksOldSafe returns the old value with HTML-safe escaping that preserves newlines
func (i *Sb3DiffItem) GetScratchBlocksOldSafe() string {
	text := i.GetScratchBlocksOld()
	return escapeHTMLPreserveWhitespace(text)
}

// GetScratchBlocksNewSafe returns the new value with HTML-safe escaping that preserves newlines
func (i *Sb3DiffItem) GetScratchBlocksNewSafe() string {
	text := i.GetScratchBlocksNew()
	
	// Log for debugging - check if text contains newlines
	// This will help identify if the problem is in text generation or rendering
	hasNewlines := strings.Contains(text, "\n")
	if len(text) > 100 && !hasNewlines {
		// Long text without newlines might indicate a problem
		// For now, we'll return it as-is, but this could be improved
	}
	
	escaped := escapeHTMLPreserveWhitespace(text)
	return escaped
}

// escapeHTMLPreserveWhitespace escapes HTML special characters but preserves newlines and tabs
func escapeHTMLPreserveWhitespace(text string) string {
	var result strings.Builder
	for _, r := range text {
		switch r {
		case '<':
			result.WriteString("&lt;")
		case '>':
			result.WriteString("&gt;")
		case '&':
			result.WriteString("&amp;")
		case '"':
			result.WriteString("&quot;")
		case '\'':
			result.WriteString("&#39;")
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// GetDebugText returns a debug version of the text with visible newlines for troubleshooting
func (i *Sb3DiffItem) GetDebugText() string {
	text := i.GetScratchBlocksNew()
	// Replace newlines with visible markers for debugging
	debugText := strings.ReplaceAll(text, "\n", " [NEWLINE] ")
	return fmt.Sprintf("Length: %d, HasNewlines: %v, Text: %s", len(text), strings.Contains(text, "\n"), debugText)
}

// formatReadableText formats the value as readable text based on its type
func (i *Sb3DiffItem) formatReadableText(data *json.RawMessage, itemType Sb3DiffItemType) string {
	if data == nil {
		return ""
	}

	var rawData any
	if err := json.Unmarshal(*data, &rawData); err != nil {
		return string(*data)
	}

	switch itemType {
	case Sb3DiffItemTarget:
		// Format target info
		if target, ok := rawData.(map[string]any); ok {
			name := getStringValue(target, "name")
			return fmt.Sprintf("角色: %s", name)
		}
	case Sb3DiffItemVariable:
		// Format variable info - handle both map and array formats
		if variable, ok := rawData.(map[string]any); ok {
			name := getStringValue(variable, "name")
			value := fmt.Sprintf("%v", variable["value"])
			return fmt.Sprintf("变量: %s = %s", name, value)
		}
		// Handle array format: ["name", value, ...]
		if variable, ok := rawData.([]any); ok && len(variable) > 0 {
			name := fmt.Sprintf("%v", variable[0])
			return fmt.Sprintf("变量: %s", name)
		}
	case Sb3DiffItemList:
		// Format list info - handle both map and array formats
		if list, ok := rawData.(map[string]any); ok {
			name := getStringValue(list, "name")
			content := fmt.Sprintf("%v", list["content"])
			return fmt.Sprintf("列表: %s = %s", name, content)
		}
		// Handle array format: ["name", content, ...]
		if list, ok := rawData.([]any); ok && len(list) > 0 {
			name := fmt.Sprintf("%v", list[0])
			return fmt.Sprintf("列表: %s", name)
		}
	case Sb3DiffItemCostume:
		// Format costume info
		if costume, ok := rawData.(map[string]any); ok {
			name := getStringValue(costume, "name")
			return fmt.Sprintf("造型: %s", name)
		}
	case Sb3DiffItemSound:
		// Format sound info
		if sound, ok := rawData.(map[string]any); ok {
			name := getStringValue(sound, "name")
			return fmt.Sprintf("声音: %s", name)
		}
	}

	// Fallback: return formatted JSON
	formatted, err := json.MarshalIndent(rawData, "", "  ")
	if err != nil {
		return string(*data)
	}
	return string(formatted)
}

// getStringValue safely gets a string value from a map
func getStringValue(data map[string]any, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Sb3DiffSummary represents a summary of changes in a Scratch project diff.
type Sb3DiffSummary struct {
	TargetsAdded      int `json:"targetsAdded"`
	TargetsDeleted    int `json:"targetsDeleted"`
	ScriptsAdded      int `json:"scriptsAdded"`
	ScriptsDeleted    int `json:"scriptsDeleted"`
	ScriptsModified   int `json:"scriptsModified"`
	BlocksAdded       int `json:"blocksAdded"`
	BlocksDeleted     int `json:"blocksDeleted"`
	BlocksModified    int `json:"blocksModified"`
	VariablesAdded    int `json:"variablesAdded"`
	VariablesDeleted  int `json:"variablesDeleted"`
	VariablesModified int `json:"variablesModified"`
	ListsAdded        int `json:"listsAdded"`
	ListsDeleted      int `json:"listsDeleted"`
	ListsModified     int `json:"listsModified"`
	CostumesAdded     int `json:"costumesAdded"`
	CostumesDeleted   int `json:"costumesDeleted"`
	CostumesModified  int `json:"costumesModified"`
	SoundsAdded       int `json:"soundsAdded"`
	SoundsDeleted     int `json:"soundsDeleted"`
	SoundsModified    int `json:"soundsModified"`
}

// Sb3Diff represents a complete Scratch project diff.
type Sb3Diff struct {
	Summary Sb3DiffSummary `json:"summary"`
	Items   []Sb3DiffItem  `json:"items"`
}

// Sb3DiffResult represents the result of a Scratch project diff operation.
type Sb3DiffResult struct {
	Diff  *Sb3Diff `json:"diff"`
	Error string   `json:"error,omitempty"`
}

// CreateSb3Diff creates a semantic diff for Scratch .sb3 project files.
// Uses native Go implementation for better performance and no external dependencies.
func CreateSb3Diff(diffFile *DiffFile, baseReader, headReader io.Reader) (*Sb3DiffResult, error) {
	// Use native Go implementation
	return CreateSb3DiffNative(diffFile, baseReader, headReader)
}

// GetSb3DiffByType filters diff items by type for easier template rendering.
func (d *Sb3Diff) GetSb3DiffByType(itemType Sb3DiffItemType) []Sb3DiffItem {
	var items []Sb3DiffItem
	for _, item := range d.Items {
		if item.Type == itemType {
			items = append(items, item)
		}
	}
	return items
}

// GetSb3DiffByTarget groups diff items by target name.
func (d *Sb3Diff) GetSb3DiffByTarget() map[string][]Sb3DiffItem {
	result := make(map[string][]Sb3DiffItem)
	for _, item := range d.Items {
		targetName := item.Location.TargetName
		result[targetName] = append(result[targetName], item)
	}
	return result
}

// GetChangedTargets returns a list of target names that have changes.
func (d *Sb3Diff) GetChangedTargets() []string {
	targetSet := make(map[string]bool)
	for _, item := range d.Items {
		targetSet[item.Location.TargetName] = true
	}
	targets := make([]string, 0, len(targetSet))
	for target := range targetSet {
		targets = append(targets, target)
	}
	return targets
}

// HasChanges returns true if there are any changes in the diff.
func (d *Sb3Diff) HasChanges() bool {
	return len(d.Items) > 0
}

// GetTotalChanges returns the total number of changes across all types.
func (d *Sb3Diff) GetTotalChanges() int {
	return len(d.Items)
}
