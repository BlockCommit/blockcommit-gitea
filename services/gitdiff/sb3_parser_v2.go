// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitdiff

import (
	"fmt"
	"strings"
)

// ParserOptions configures the SB3 parser behavior
type ParserOptions struct {
	Tabs          string
	VariableStyle VariableStyle
	Locale        string
}

// VariableStyle determines when to add ::variables suffix
type VariableStyle int

const (
	VariableStyleNone     VariableStyle = iota // Never add ::variables
	VariableStyleAlways                       // Always add ::variables
	VariableStyleAsNeeded                     // Add ::variables only if needed
)

// Inputtable represents something that can be used as an input
type Inputtable interface {
	ToScratchblocks(options ParserOptions) string
}

// Connectable represents something that can be connected in a script
type Connectable interface {
	ToScratchblocks(options ParserOptions) string
	GetOpcode() string
	GetID() string
	GetNext() *string
	SetNext(next *string)
}

// Block represents a regular stack/hat/cap block
type Block struct {
	ID          string
	Opcode      string
	Inputs      map[string]Inputtable
	Fields      map[string][]string
	Next        *string
	Parent      *string
	Shadow      bool
	TopLevel    bool
	X, Y        *float64
	Mutation    any
	Inputtables map[string][]any // Original input data for parsing
}

// ToScratchblocks converts a Block to ScratchBlocks syntax
func (b *Block) ToScratchblocks(options ParserOptions) string {
	// Use old handler with preprocessed inputs to fix variable/list formatting
	if fn, exists := BlockOpcodeMap[b.Opcode]; exists {
		// Preprocess inputs to fix variable/list formatting
		preprocessedInputs := b.preprocessInputs()
		
		// Convert Block to RawBlock for compatibility with existing parser
		rawBlock := RawBlock{
			Opcode:   b.Opcode,
			Next:     b.Next,
			Parent:   b.Parent,
			Shadow:   b.Shadow,
			TopLevel: b.TopLevel,
			X:        b.X,
			Y:        b.Y,
			Mutation: b.Mutation,
			Fields:   b.Fields,
			Inputs:   preprocessedInputs,
		}
		return fn(&rawBlock, &ScratchBlocksParser{locale: options.Locale})
	}
	return fmt.Sprintf("[%s]", strings.ReplaceAll(b.Opcode, "_", " "))
}

// preprocessInputs pre-processes inputs to fix variable/list formatting and remove array prefixes
func (b *Block) preprocessInputs() map[string][]any {
	preprocessed := make(map[string][]any)
	
	for inputName, inputtable := range b.Inputs {
		// Convert Inputtable to the format expected by old parser
		if inp, ok := inputtable.(*Input); ok {
			// Remove any array prefix like "[4 10]" or "[10 xxx]"
			cleanValue := removeArrayPrefix(inp.Value)
			preprocessed[inputName] = []any{float64(1), cleanValue}
		} else if variable, ok := inputtable.(*Variable); ok {
			preprocessed[inputName] = []any{float64(10), variable.Name}
		} else if menu, ok := inputtable.(*Menu); ok {
			if menu.Type == "list" {
				preprocessed[inputName] = []any{float64(11), menu.Value}
			} else if menu.Type == "broadcast" {
				preprocessed[inputName] = []any{float64(12), menu.Value}
			} else {
				preprocessed[inputName] = []any{float64(2), menu.Value}
			}
		} else if nbp, ok := inputtable.(*NestedBlockPlaceholder); ok {
			if nbp.Resolved && nbp.Block != nil {
				// Get the actual block content for input
				blockContent := nbp.Block.ToScratchblocks(ParserOptions{})
				// Remove any array prefix from the block content
				cleanContent := removeArrayPrefix(blockContent)
				preprocessed[inputName] = []any{float64(1), cleanContent}
			} else {
				// Unresolved placeholder - return empty to avoid showing [block: xxx]
				preprocessed[inputName] = []any{float64(1), ""}
			}
		} else if reporter, ok := inputtable.(*ReporterBlock); ok {
			// Get reporter content directly
			content := reporter.ToScratchblocks(ParserOptions{})
			// Remove any array prefix
			cleanContent := removeArrayPrefix(content)
			preprocessed[inputName] = []any{float64(1), cleanContent}
		} else if boolean, ok := inputtable.(*BooleanBlock); ok {
			// Get boolean content directly
			content := boolean.ToScratchblocks(ParserOptions{})
			// Remove any array prefix  
			cleanContent := removeArrayPrefix(content)
			preprocessed[inputName] = []any{float64(1), cleanContent}
		} else {
			// Unknown type, try to convert to string
			content := inputtable.ToScratchblocks(ParserOptions{})
			cleanContent := removeArrayPrefix(content)
			preprocessed[inputName] = []any{float64(1), cleanContent}
		}
	}
	
	return preprocessed
}

// removeArrayPrefix removes SB3 internal array prefix like "[4 10]" or "[10 xxx]" and returns the clean value
// It preserves valid scratchblocks syntax like (x position), <>, [text]
func removeArrayPrefix(value string) string {
	value = strings.TrimSpace(value)
	
	// If it's already valid scratchblocks syntax, return as-is
	// Valid syntax: (...) for reporters, <...> for booleans, [...] for text
	if strings.HasPrefix(value, "(") || strings.HasPrefix(value, "<") {
		return value
	}
	
	// If it starts with [ but is a complete scratchblocks syntax (ends with ]), check if it has valid structure
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		// If it's a dropdown (has " v" before the closing bracket), preserve it
		if strings.Contains(value, " v]") {
			return value
		}
		// If it doesn't have spaces or has only one element, it's likely valid scratchblocks
		if !strings.Contains(value, " ") || len(strings.Split(value, " ")) <= 1 {
			return value
		}
	}
	
	// Now handle SB3 internal array format: [type value]
	// The format is always: [number_type value]
	// e.g., [4 10] for number 10, [10 xxx] for variable xxx
	if strings.HasPrefix(value, "[") && strings.Contains(value, " ") {
		// Split by space to get type and value
		parts := strings.SplitN(value, " ", 2)
		if len(parts) == 2 {
			// The first part should be the type number
			typePart := strings.TrimPrefix(parts[0], "[")
			// Check if it's a number (SB3 type indicator)
			if _, err := fmt.Sscanf(typePart, "%f", new(float64)); err == nil {
				// It's a valid SB3 type, extract the value part
				valuePart := parts[1]
				// Remove trailing ]
				actualValue := strings.TrimSuffix(valuePart, "]")
				return actualValue
			}
		}
	}
	
	// If no SB3 array prefix found, return as-is
	return value
}

func (b *Block) GetOpcode() string { return b.Opcode }
func (b *Block) GetID() string     { return b.ID }
func (b *Block) GetNext() *string { return b.Next }
func (b *Block) SetNext(next *string) {
	b.Next = next
}

// CBlock represents a C-block (conditional block like if, repeat, etc.)
type CBlock struct {
	*Block
	Substack string // ID of the substack
}

// ToScratchblocks converts a CBlock to ScratchBlocks syntax
func (cb *CBlock) ToScratchblocks(options ParserOptions) string {
	// Get the base block text
	baseText := cb.Block.ToScratchblocks(options)

	// Check if there's a substack to append
	if cb.Substack != "" {
		// The substack will be added separately by the parser
		return baseText
	}

	return baseText
}

// EBlock represents an E-block (if-else block)
type EBlock struct {
	*CBlock
	ElseSubstack string // ID of the else substack
}

// ToScratchblocks converts an EBlock to ScratchBlocks syntax
func (eb *EBlock) ToScratchblocks(options ParserOptions) string {
	return eb.CBlock.ToScratchblocks(options)
}

// Definition represents a custom block definition
type Definition struct {
	*Block
	Proccode string // Custom block name
}

// ToScratchblocks converts a Definition to ScratchBlocks syntax
func (d *Definition) ToScratchblocks(options ParserOptions) string {
	proccode := d.Proccode
	if proccode == "" {
		// Try to get from mutation
		if mutation, ok := d.Mutation.(map[string]any); ok {
			if pc, ok := mutation["proccode"].(string); ok {
				proccode = pc
			}
		}
	}
	return fmt.Sprintf("define %s", proccode)
}

// ProcedureCall represents a call to a custom block
type ProcedureCall struct {
	*Block
	Proccode string
}

// ToScratchblocks converts a ProcedureCall to ScratchBlocks syntax
func (pc *ProcedureCall) ToScratchblocks(options ParserOptions) string {
	proccode := pc.Proccode
	if proccode == "" {
		// Try to get from mutation
		if mutation, ok := pc.Mutation.(map[string]any); ok {
			if pc, ok := mutation["proccode"].(string); ok {
				proccode = pc
			}
		}
	}

	// Add arguments
	var args []string
	if pc.Inputs != nil {
		for _, input := range pc.Inputs {
			args = append(args, input.ToScratchblocks(options))
		}
	}

	if len(args) > 0 {
		return fmt.Sprintf("%s %s", proccode, strings.Join(args, " "))
	}
	return proccode
}

// Stack represents a stack of blocks (used for arguments to C/E blocks)
type Stack struct {
	Blocks []Connectable
}

// ToScratchblocks converts a Stack to ScratchBlocks syntax
func (s *Stack) ToScratchblocks(options ParserOptions) string {
	if len(s.Blocks) == 0 {
		return ""
	}

	var lines []string
	for _, block := range s.Blocks {
		lines = append(lines, options.Tabs+block.ToScratchblocks(options))
	}
	return strings.Join(lines, "\n")
}

// Menu represents a menu option (field menu or menu block)
type Menu struct {
	Value string
	Type  string // "field" or "menu"
}

// ToScratchblocks converts a Menu to ScratchBlocks syntax
func (m *Menu) ToScratchblocks(options ParserOptions) string {
	if m.Type == "field" {
		return fmt.Sprintf("[%s v]", m.Value)
	}
	return fmt.Sprintf("(%s)", m.Value)
}

// Variable represents a variable (variable reporter or custom block argument)
type Variable struct {
	Name  string
	Value string
}

// ToScratchblocks converts a Variable to ScratchBlocks syntax
func (v *Variable) ToScratchblocks(options ParserOptions) string {
	suffix := ""
	if options.VariableStyle == VariableStyleAlways {
		suffix = " ::variables"
	}
	return fmt.Sprintf("(%s%s)", v.Name, suffix)
}

// Icon represents an icon (greenFlag, turnLeft, turnRight)
type Icon struct {
	Type string
}

// ToScratchblocks converts an Icon to ScratchBlocks syntax
func (i *Icon) ToScratchblocks(options ParserOptions) string {
	switch i.Type {
	case "greenFlag":
		return "⚑"
	case "turnLeft":
		return "↺"
	case "turnRight":
		return "↻"
	default:
		return i.Type
	}
}

// BooleanBlock represents a boolean reporter block
type BooleanBlock struct {
	*Block
}

// ToScratchblocks converts a BooleanBlock to ScratchBlocks syntax
func (bb *BooleanBlock) ToScratchblocks(options ParserOptions) string {
	// Use the parsed inputs to build boolean expression
	return bb.buildBooleanExpression(options)
}

// buildBooleanExpression builds a boolean expression from the block's inputs
func (bb *BooleanBlock) buildBooleanExpression(options ParserOptions) string {
	// Try to use the opcode-specific formatting from BlockOpcodeMap
	if handler, exists := booleanHandlers[bb.Opcode]; exists {
		return handler(bb, options)
	}
	
	// Default: try to use the first input as the boolean expression
	if len(bb.Inputs) > 0 {
		// Get the first input and convert it
		firstInput := bb.Inputs["CONDITION"]
		if firstInput != nil {
			inputText := firstInput.ToScratchblocks(options)
			return fmt.Sprintf("<%s>", inputText)
		}
	}
	
	return "<>"
}

// booleanHandlers provides opcode-specific boolean formatting
var booleanHandlers = map[string]func(*BooleanBlock, ParserOptions) string{
	"operator_and": func(b *BooleanBlock, options ParserOptions) string {
		if len(b.Inputs) >= 2 {
			op1 := b.Inputs["OPERAND1"]
			op2 := b.Inputs["OPERAND2"]
			if op1 != nil && op2 != nil {
				return fmt.Sprintf("<%s> and <%s>", op1.ToScratchblocks(options), op2.ToScratchblocks(options))
			}
			return "<>"
		}
		return "<>"
	},
	"operator_or": func(b *BooleanBlock, options ParserOptions) string {
		if len(b.Inputs) >= 2 {
			op1 := b.Inputs["OPERAND1"]
			op2 := b.Inputs["OPERAND2"]
			if op1 != nil && op2 != nil {
				return fmt.Sprintf("<%s> or <%s>", op1.ToScratchblocks(options), op2.ToScratchblocks(options))
			}
			return "<>"
		}
		return "<>"
	},
	"operator_not": func(b *BooleanBlock, options ParserOptions) string {
		if len(b.Inputs) > 0 {
			op := b.Inputs["OPERAND"]
			if op != nil {
				return fmt.Sprintf("not <%s>", op.ToScratchblocks(options))
			}
		}
		return "not <>"
	},
	"operator_join": func(b *BooleanBlock, options ParserOptions) string {
		if len(b.Inputs) >= 2 {
			str1 := b.Inputs["STRING1"]
			str2 := b.Inputs["STRING2"]
			if str1 != nil && str2 != nil {
				return fmt.Sprintf("[%s] [%s]", str1.ToScratchblocks(options), str2.ToScratchblocks(options))
			}
		}
		return "[ ] [ ]"
	},
	"operator_letter": func(b *BooleanBlock, options ParserOptions) string {
		if len(b.Inputs) >= 2 {
			letter := b.Inputs["LETTER"]
			str := b.Inputs["STRING"]
			if letter != nil && str != nil {
				return fmt.Sprintf("letter (%s) of [%s]", letter.ToScratchblocks(options), str.ToScratchblocks(options))
			}
		}
		return "letter () of [ ]"
	},
	"operator_length": func(b *BooleanBlock, options ParserOptions) string {
		if len(b.Inputs) > 0 {
			str := b.Inputs["STRING"]
			if str != nil {
				return fmt.Sprintf("length of [%s]", str.ToScratchblocks(options))
			}
		}
		return "length of [ ]"
	},
	"operator_contains": func(b *BooleanBlock, options ParserOptions) string {
		if len(b.Inputs) >= 2 {
			str := b.Inputs["STRING"]
			item := b.Inputs["ITEM"]
			if str != nil && item != nil {
				return fmt.Sprintf("[%s] contains [%s]?", str.ToScratchblocks(options), item.ToScratchblocks(options))
			}
		}
		return "[ ] contains [ ]?"
	},
	"sensing_touchingcolor": func(b *BooleanBlock, options ParserOptions) string {
		if len(b.Inputs) >= 2 {
			color := b.Inputs["COLOR"]
			object := b.Inputs["TOUCHINGOBJECTMENU"]
			if color != nil && object != nil {
				return fmt.Sprintf("touching [%s v]?", color.ToScratchblocks(options), object.ToScratchblocks(options))
			}
		}
		return "touching [ ]?"
	},
}

// ReporterBlock represents a reporter block
type ReporterBlock struct {
	*Block
}

// ToScratchblocks converts a ReporterBlock to ScratchBlocks syntax
func (rb *ReporterBlock) ToScratchblocks(options ParserOptions) string {
	// Use the parsed inputs to build reporter expression
	return rb.buildReporterExpression(options)
}

// buildReporterExpression builds a reporter expression from the block's inputs
func (rb *ReporterBlock) buildReporterExpression(options ParserOptions) string {
	// First, check if this is actually a boolean block
	if isBooleanOpcode(rb.Opcode) {
		// Delegate to boolean block handling
		return (&BooleanBlock{Block: rb.Block}).ToScratchblocks(options)
	}
	
	// For regular reporter blocks, use the opcode-specific handler
	if handler, exists := BlockOpcodeMap[rb.Opcode]; exists {
		// Convert Block to RawBlock for compatibility with existing parser
		rawBlock := RawBlock{
			Opcode:   rb.Opcode,
			Next:     rb.Next,
			Parent:   rb.Parent,
			Shadow:   rb.Shadow,
			TopLevel: rb.TopLevel,
			X:        rb.X,
			Y:        rb.Y,
			Mutation: rb.Mutation,
			Fields:   rb.Fields,
			Inputs:   rb.Block.preprocessInputs(),
		}
		return handler(&rawBlock, &ScratchBlocksParser{locale: options.Locale})
	}
	
	// Fallback: return formatted block text
	text := rb.Block.ToScratchblocks(options)
	return fmt.Sprintf("(%s)", text)
}

// Input represents a literal input
type Input struct {
	Type  string // "number", "string", "color", "broadcast"
	Value string
}

// ToScratchblocks converts an Input to ScratchBlocks syntax
func (inp *Input) ToScratchblocks(options ParserOptions) string {
	switch inp.Type {
	case "number":
		// Numbers use parentheses: (10)
		// The value should already be clean from parseInput
		return fmt.Sprintf("(%s)", inp.Value)
	case "string":
		// Strings use square brackets: [text]
		// The value should already be clean from parseInput
		return fmt.Sprintf("[%s]", inp.Value)
	case "color":
		// Colors use square brackets: [hex]
		return fmt.Sprintf("[%s]", inp.Value)
	case "broadcast":
		// Broadcasts use dropdown format: [message v]
		return fmt.Sprintf("[%s v]", inp.Value)
	default:
		return inp.Value
	}
}
// NestedBlockPlaceholder represents a nested block that needs to be resolved
type NestedBlockPlaceholder struct {
	BlockID string
	Resolved bool
	Block    Inputtable
}

// ToScratchblocks converts a NestedBlockPlaceholder to ScratchBlocks syntax
func (nbp *NestedBlockPlaceholder) ToScratchblocks(options ParserOptions) string {
	if nbp.Resolved && nbp.Block != nil {
		return nbp.Block.ToScratchblocks(options)
	}
	// Fallback if not resolved - return empty string or simple placeholder
	return ""
}

// Resolve resolves the nested block placeholder by looking up the block
func (nbp *NestedBlockPlaceholder) Resolve(blocks map[string]Connectable) {
	if block, exists := blocks[nbp.BlockID]; exists {
		nbp.Block = block.(Inputtable)
		nbp.Resolved = true
	}
}

// SB3ParserV2 is the improved parser for SB3 files
type SB3ParserV2 struct {
	options ParserOptions
	blocks  map[string]Connectable
}

// NewSB3ParserV2 creates a new improved SB3 parser
func NewSB3ParserV2(options ParserOptions) *SB3ParserV2 {
	if options.Locale == "" {
		options.Locale = "en"
	}
	if options.Tabs == "" {
		options.Tabs = "    " // 4 spaces
	}
	if options.VariableStyle == 0 {
		options.VariableStyle = VariableStyleNone
	}
	return &SB3ParserV2{
		options: options,
		blocks:  make(map[string]Connectable),
	}
}

// ParseBlocks parses raw blocks into Connectable objects
func (p *SB3ParserV2) ParseBlocks(rawBlocks map[string]RawBlock) error {
	// First pass: parse all blocks
	for id, rawBlock := range rawBlocks {
		block := p.parseRawBlock(id, rawBlock)
		if block != nil {
			p.blocks[id] = block
		}
	}

	// Second pass: resolve all nested block placeholders
	p.resolveNestedBlocks()

	return nil
}

// resolveNestedBlocks resolves all nested block placeholders
func (p *SB3ParserV2) resolveNestedBlocks() {
	for _, block := range p.blocks {
		if b, ok := block.(*Block); ok {
			// Resolve inputs in this block
			for _, input := range b.Inputs {
				if nbp, ok := input.(*NestedBlockPlaceholder); ok {
					nbp.Resolve(p.blocks)
				}
			}
		}
	}
}

// hasInputs checks if a Connectable has inputs
func hasInputs(c Connectable) bool {
	switch v := c.(type) {
	case *Block:
		return len(v.Inputs) > 0
	case *CBlock:
		return len(v.Inputs) > 0 || v.Substack != ""
	case *EBlock:
		return len(v.Inputs) > 0 || v.Substack != "" || v.ElseSubstack != ""
	case *ReporterBlock:
		return len(v.Inputs) > 0
	case *BooleanBlock:
		return len(v.Inputs) > 0
	case *Definition:
		return len(v.Inputs) > 0
	case *ProcedureCall:
		return len(v.Inputs) > 0
	default:
		return false
	}
}

// parseRawBlock converts a RawBlock to a Connectable object
func (p *SB3ParserV2) parseRawBlock(id string, rawBlock RawBlock) Connectable {
	baseBlock := &Block{
		ID:          id,
		Opcode:      rawBlock.Opcode,
		Next:        rawBlock.Next,
		Parent:      rawBlock.Parent,
		Shadow:      rawBlock.Shadow,
		TopLevel:    rawBlock.TopLevel,
		X:           rawBlock.X,
		Y:           rawBlock.Y,
		Mutation:    rawBlock.Mutation,
		Fields:      rawBlock.Fields,
		Inputtables: rawBlock.Inputs,
	}

	// Parse inputs
	baseBlock.Inputs = make(map[string]Inputtable)
	for inputName, inputData := range rawBlock.Inputs {
		baseBlock.Inputs[inputName] = p.parseInput(inputData, rawBlock.Fields, inputName)
	}

	// Determine block type
	switch {
	case strings.HasPrefix(rawBlock.Opcode, "procedures_definition"):
		return &Definition{Block: baseBlock}
	case strings.HasPrefix(rawBlock.Opcode, "procedures_call"):
		return &ProcedureCall{Block: baseBlock}
	case rawBlock.Opcode == "control_if_else":
		return &EBlock{CBlock: &CBlock{
			Block:    baseBlock,
			Substack: p.getSubstackID(rawBlock, "SUBSTACK"),
		}, ElseSubstack: p.getSubstackID(rawBlock, "SUBSTACK2")}
	case isCBlockOpcodeV2(rawBlock.Opcode):
		return &CBlock{
			Block:    baseBlock,
			Substack: p.getSubstackID(rawBlock, "SUBSTACK"),
		}
	case isBooleanOpcode(rawBlock.Opcode):
		return &BooleanBlock{Block: baseBlock}
	case isReporterOpcode(rawBlock.Opcode):
		return &ReporterBlock{Block: baseBlock}
	default:
		return baseBlock
	}
}

// parseInput parses input data into an Inputtable
func (p *SB3ParserV2) parseInput(inputData []any, fields map[string][]string, inputName string) Inputtable {
	if len(inputData) < 2 {
		return &Input{Type: "string", Value: ""}
	}

	inputType, _ := inputData[0].(float64)
	value := inputData[1]

	switch int(inputType) {
	case 1: // Literal value
		return &Input{
			Type:  p.getLiteralType(value),
			Value: p.formatLiteralValue(value),
		}
	case 2: // Menu option
		if menuID, ok := value.(string); ok {
			// Try to get the display value from Fields
			// The field key is usually the same as the input name
			displayValue := menuID
			if fieldValues, exists := fields[inputName]; exists && len(fieldValues) > 0 {
				displayValue = fieldValues[0]
			}
			return &Menu{
				Value: displayValue,
				Type:  "field",
			}
		}
	case 3: // Nested block - try to parse it immediately if possible
		if blockID, ok := value.(string); ok {
			// Try to get already parsed block
			if block, exists := p.blocks[blockID]; exists {
				return block
			}
			// Create placeholder for later resolution
			return &NestedBlockPlaceholder{
				BlockID: blockID,
			}
		}
		case 4: // Positive number
			if num, ok := value.(float64); ok {
				return &Input{Type: "number", Value: fmt.Sprintf("%.0f", num)}
			}
		case 5: // Positive integer
			if num, ok := value.(int); ok {
				return &Input{Type: "number", Value: fmt.Sprintf("%d", num)}
			}
		case 6: // Positive angle
			if angle, ok := value.(float64); ok {
				return &Input{Type: "number", Value: fmt.Sprintf("%.0f", angle)}
			}
		case 7: // Color
			if color, ok := value.(string); ok {
				return &Input{Type: "color", Value: color}
			}
		case 8: // String
			if str, ok := value.(string); ok {
				return &Input{Type: "string", Value: str}
			}
		case 9: // Broadcast
			if msg, ok := value.(string); ok {
				return &Menu{Value: msg, Type: "broadcast"}
			}
		case 10: // Variable
			if varName, ok := value.(string); ok {
				return &Variable{
					Name:  varName,
					Value: varName,
				}
			}
		case 11: // List
			if listName, ok := value.(string); ok {
				return &Menu{
					Value: listName,
					Type:  "list",
				}
			}
	case 12: // Broadcast (another format)
		if msg, ok := value.(string); ok {
			return &Menu{
				Value: msg,
				Type:  "broadcast",
			}
		}
	case 13: // Note
		if note, ok := value.(string); ok {
			return &Input{Type: "string", Value: note}
		}
	}

	// Default: treat as string
	return &Input{
		Type:  "string",
		Value: p.formatValue(value),
	}
}

// getLiteralType determines the type of a literal value
func (p *SB3ParserV2) getLiteralType(value any) string {
	// Handle array format: ["type", actualValue, ...]
	if arr, ok := value.([]any); ok && len(arr) > 1 {
		// Look at the second element to determine type
		if _, ok := arr[1].(string); ok {
			return "string"
		} else if _, ok := arr[1].(float64); ok {
			return "number"
		} else if _, ok := arr[1].(int); ok {
			return "number"
		}
	}
	// Handle regular values
	switch value.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case int:
		return "number"
	case bool:
		return "string"
	default:
		return "string"
	}
}

// formatLiteralValue formats a literal value for display
func (p *SB3ParserV2) formatLiteralValue(value any) string {
	// Handle array format: ["type", actualValue, ...]
	if arr, ok := value.([]any); ok && len(arr) > 1 {
		// Format: ["4", 10] -> "10", ["10", "你好！"] -> "你好！"
		if actualValue, ok := arr[1].(string); ok {
			return actualValue
		} else if actualValue, ok := arr[1].(float64); ok {
			return fmt.Sprintf("%.0f", actualValue)
		}
		// Fallback to string representation
		return fmt.Sprintf("%v", arr[1])
	}
	// Handle regular values
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	}
	return fmt.Sprintf("%v", value)
}

// formatValue formats any value to string
func (p *SB3ParserV2) formatValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

// extractNameFromArray extracts the name from a variable/list array format
// Format: ["name", value, ...] or just "name"
func (p *SB3ParserV2) extractNameFromArray(value any) string {
	// If it's a string, return it directly
	if name, ok := value.(string); ok {
		return name
	}

	// If it's an array, extract the first element
	if arr, ok := value.([]any); ok && len(arr) > 0 {
		if name, ok := arr[0].(string); ok {
			return name
		}
	}

	// Fallback
	return fmt.Sprintf("%v", value)
}

// getSubstackID extracts a substack ID from inputs
func (p *SB3ParserV2) getSubstackID(rawBlock RawBlock, substackName string) string {
	if substackData, exists := rawBlock.Inputs[substackName]; exists && len(substackData) >= 2 {
		if substackID, ok := substackData[1].(string); ok {
			return substackID
		}
	}
	return ""
}

// isCBlockOpcodeV2 checks if an opcode is for a C-block (V2 version)
func isCBlockOpcodeV2(opcode string) bool {
	cBlocks := map[string]bool{
		"control_if":                  true,
		"control_repeat":              true,
		"control_repeat_until":        true,
		"control_forever":             true,
		"control_for_each":            true,
		"motion_gotoxy":               false, // Not a C-block
		"procedures_definition":       false, // Not a C-block
	}
	return cBlocks[opcode] || strings.HasPrefix(opcode, "control_")
}

// isBooleanOpcode checks if an opcode is for a boolean block
func isBooleanOpcode(opcode string) bool {
	return strings.HasSuffix(opcode, "_boolean") ||
		opcode == "operator_and" ||
		opcode == "operator_or" ||
		opcode == "operator_not" ||
		strings.HasPrefix(opcode, "sensing_") && strings.HasSuffix(opcode, "touching") ||
		strings.HasPrefix(opcode, "sensing_") && strings.HasSuffix(opcode, "touchingcolor")
}

// isReporterOpcode checks if an opcode is for a reporter block
func isReporterOpcode(opcode string) bool {
	return strings.HasSuffix(opcode, "_reporter") ||
		strings.HasPrefix(opcode, "operator_") ||
		strings.HasPrefix(opcode, "motion_") && !isBooleanOpcode(opcode) ||
		strings.HasPrefix(opcode, "looks_") && !isBooleanOpcode(opcode) ||
		strings.HasPrefix(opcode, "sensing_") && !isBooleanOpcode(opcode) ||
		strings.HasPrefix(opcode, "sound_") && !isBooleanOpcode(opcode) ||
		strings.HasPrefix(opcode, "pen_") && !isBooleanOpcode(opcode) ||
		opcode == "data_variable" ||
		opcode == "data_listcontents"
}

// ToScratchblocks converts a script to ScratchBlocks syntax
func (p *SB3ParserV2) ToScratchblocks(startBlockID string) string {
	_, exists := p.blocks[startBlockID]
	if !exists {
		return ""
	}

	var result strings.Builder
	currentID := startBlockID
	indentLevel := 0

	for currentID != "" {
		currentBlock, exists := p.blocks[currentID]
		if !exists {
			break
		}

		// Add indentation
		result.WriteString(strings.Repeat(p.options.Tabs, indentLevel))

		// Convert block to text
		blockText := currentBlock.ToScratchblocks(p.options)
		result.WriteString(blockText)
		result.WriteString("\n")

		// Handle C-blocks and their substacks
		if cb, ok := currentBlock.(*CBlock); ok && cb.Substack != "" {
			indentLevel++
			// Process substack
			substackResult := p.processSubstack(cb.Substack, indentLevel)
			result.WriteString(substackResult)
			indentLevel--

			// Handle E-block else substack
			if eb, ok := currentBlock.(*EBlock); ok && eb.ElseSubstack != "" {
				result.WriteString(strings.Repeat(p.options.Tabs, indentLevel))
				result.WriteString("else\n")
				indentLevel++
				elseResult := p.processSubstack(eb.ElseSubstack, indentLevel)
				result.WriteString(elseResult)
				indentLevel--
			}
		}

		// Move to next block
		if currentBlock.GetNext() != nil {
			currentID = *currentBlock.GetNext()
		} else {
			currentID = ""
		}
	}

	return result.String()
}

// processSubstack processes a substack (blocks inside C/E blocks)
func (p *SB3ParserV2) processSubstack(substackID string, indentLevel int) string {
	var result strings.Builder
	currentID := substackID

	for currentID != "" {
		connectable, exists := p.blocks[currentID]
		if !exists {
			break
		}

		// Add indentation
		result.WriteString(strings.Repeat(p.options.Tabs, indentLevel))

		// Convert block to text
		blockText := connectable.ToScratchblocks(p.options)
		result.WriteString(blockText)
		result.WriteString("\n")

		// Handle nested C-blocks
		if cb, ok := connectable.(*CBlock); ok && cb.Substack != "" {
			indentLevel++
			substackResult := p.processSubstack(cb.Substack, indentLevel)
			result.WriteString(substackResult)
			indentLevel--

			// Handle nested E-blocks
			if eb, ok := connectable.(*EBlock); ok && eb.ElseSubstack != "" {
				result.WriteString(strings.Repeat(p.options.Tabs, indentLevel))
				result.WriteString("else\n")
				indentLevel++
				elseResult := p.processSubstack(eb.ElseSubstack, indentLevel)
				result.WriteString(elseResult)
				indentLevel--
			}
		}

		// Move to next block
		if connectable.GetNext() != nil {
			currentID = *connectable.GetNext()
		} else {
			currentID = ""
		}
	}

	return result.String()
}