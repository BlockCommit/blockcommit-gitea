// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitdiff

import (
	"encoding/json"
	"io"
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

	// Try to parse as ScriptText to get ScratchBlocks syntax
	var script ScriptText
	if err := json.Unmarshal(*i.Old, &script); err == nil {
		// If it's a script, return the text which is already in ScratchBlocks format
		return script.Text
	}

	// For other types, return formatted JSON
	return i.GetFormattedOld()
}

// GetScratchBlocksNew returns the new value in ScratchBlocks syntax format for visualization
func (i *Sb3DiffItem) GetScratchBlocksNew() string {
	if i.New == nil {
		return ""
	}

	// Try to parse as ScriptText to get ScratchBlocks syntax
	var script ScriptText
	if err := json.Unmarshal(*i.New, &script); err == nil {
		// If it's a script, return the text which is already in ScratchBlocks format
		return script.Text
	}

	// For other types, return formatted JSON
	return i.GetFormattedNew()
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
