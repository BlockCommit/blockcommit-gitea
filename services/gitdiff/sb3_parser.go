// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitdiff

import (
	"fmt"
	"strings"
)

// ScratchBlocksParser converts Scratch blocks to ScratchBlocks syntax
// Based on parse-sb3-blocks library logic
type ScratchBlocksParser struct {
	locale string
}

// NewScratchBlocksParser creates a new ScratchBlocks parser
func NewScratchBlocksParser(locale string) *ScratchBlocksParser {
	if locale == "" {
		locale = "en"
	}
	return &ScratchBlocksParser{locale: locale}
}

// BlockOpcodeMap maps Scratch opcodes to ScratchBlocks syntax
var BlockOpcodeMap = map[string]func(*RawBlock, *ScratchBlocksParser) string{
	// Events
	"event_whenflagclicked": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "when flag clicked"
	},
	"event_whenkeypressed": func(b *RawBlock, p *ScratchBlocksParser) string {
		key := p.getFieldValue(b, "KEY_OPTION", "space")
		return fmt.Sprintf("when [%s v] key pressed", key)
	},
	"event_whenbackdropswitchesto": func(b *RawBlock, p *ScratchBlocksParser) string {
		backdrop := p.getFieldValue(b, "BACKDROP", "next backdrop v")
		return fmt.Sprintf("when backdrop switches to [%s]", backdrop)
	},
	"event_whenthisspriteclicked": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "when this sprite clicked"
	},
	"event_whenbroadcastreceived": func(b *RawBlock, p *ScratchBlocksParser) string {
		msg := p.getFieldValue(b, "BROADCAST_OPTION", "message1 v")
		return fmt.Sprintf("when I receive [%s]", msg)
	},
	"event_whengreaterthan": func(b *RawBlock, p *ScratchBlocksParser) string {
		operand := p.getInputValue(b, "OPERAND", "0")
		option := p.getFieldValue(b, "OPTION", "timer")
		return fmt.Sprintf("when [%s v] > (%s)", option, operand)
	},
	"event_broadcast": func(b *RawBlock, p *ScratchBlocksParser) string {
		msg := p.getInputValue(b, "BROADCAST", "message1")
		return fmt.Sprintf("broadcast [%s]", msg)
	},
	"event_broadcastandwait": func(b *RawBlock, p *ScratchBlocksParser) string {
		msg := p.getInputValue(b, "BROADCAST", "message1")
		return fmt.Sprintf("broadcast [%s] and wait", msg)
	},

	// Motion
	"motion_movesteps": func(b *RawBlock, p *ScratchBlocksParser) string {
		steps := p.getInputValue(b, "STEPS", "10")
		return fmt.Sprintf("move (%s) steps", steps)
	},
	"motion_turnright": func(b *RawBlock, p *ScratchBlocksParser) string {
		deg := p.getInputValue(b, "DEGREES", "15")
		return fmt.Sprintf("turn cw (%s) degrees", deg)
	},
	"motion_turnleft": func(b *RawBlock, p *ScratchBlocksParser) string {
		deg := p.getInputValue(b, "DEGREES", "15")
		return fmt.Sprintf("turn ccw (%s) degrees", deg)
	},
	"motion_gotoxy": func(b *RawBlock, p *ScratchBlocksParser) string {
		x := p.getInputValue(b, "X", "0")
		y := p.getInputValue(b, "Y", "0")
		return fmt.Sprintf("go to x: (%s) y: (%s)", x, y)
	},
	"motion_goto": func(b *RawBlock, p *ScratchBlocksParser) string {
		to := p.getInputValue(b, "TO", "_mouse_")
		return fmt.Sprintf("go to [%s v]", to)
	},
	"motion_glideto": func(b *RawBlock, p *ScratchBlocksParser) string {
		secs := p.getInputValue(b, "SECS", "1")
		x := p.getInputValue(b, "X", "0")
		y := p.getInputValue(b, "Y", "0")
		return fmt.Sprintf("glide (%s) secs to x: (%s) y: (%s)", secs, x, y)
	},
	"motion_pointindirection": func(b *RawBlock, p *ScratchBlocksParser) string {
		dir := p.getInputValue(b, "DIRECTION", "90")
		return fmt.Sprintf("point in direction (%s)", dir)
	},
	"motion_pointtowards": func(b *RawBlock, p *ScratchBlocksParser) string {
		towards := p.getInputValue(b, "TOWARDS", "_mouse_")
		return fmt.Sprintf("point towards [%s v]", towards)
	},
	"motion_changexby": func(b *RawBlock, p *ScratchBlocksParser) string {
		dx := p.getInputValue(b, "DX", "10")
		return fmt.Sprintf("change x by (%s)", dx)
	},
	"motion_setx": func(b *RawBlock, p *ScratchBlocksParser) string {
		x := p.getInputValue(b, "X", "0")
		return fmt.Sprintf("set x to (%s)", x)
	},
	"motion_changeyby": func(b *RawBlock, p *ScratchBlocksParser) string {
		dy := p.getInputValue(b, "DY", "10")
		return fmt.Sprintf("change y by (%s)", dy)
	},
	"motion_sety": func(b *RawBlock, p *ScratchBlocksParser) string {
		y := p.getInputValue(b, "Y", "0")
		return fmt.Sprintf("set y to (%s)", y)
	},
	"motion_ifonedgebounce": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "if on edge, bounce"
	},
	"motion_setrotationstyle": func(b *RawBlock, p *ScratchBlocksParser) string {
		style := p.getInputValue(b, "STYLE", "left-right")
		return fmt.Sprintf("set rotation style [%s v]", style)
	},
	"motion_xposition": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "x position"
	},
	"motion_yposition": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "y position"
	},
	"motion_direction": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "direction"
	},

	// Looks
	"looks_say": func(b *RawBlock, p *ScratchBlocksParser) string {
		msg := p.getInputValue(b, "MESSAGE", "Hello!")
		// If the message is already a reporter (starts with '(' or '<'), use it directly
		if strings.HasPrefix(msg, "(") || strings.HasPrefix(msg, "<") {
			return fmt.Sprintf("say %s", msg)
		}
		return fmt.Sprintf("say [%s]", msg)
	},
	"looks_sayforsecs": func(b *RawBlock, p *ScratchBlocksParser) string {
		msg := p.getInputValue(b, "MESSAGE", "Hello!")
		secs := p.getInputValue(b, "SECS", "2")
		// If the message is already a reporter (starts with '(' or '<'), use it directly
		if strings.HasPrefix(msg, "(") || strings.HasPrefix(msg, "<") {
			return fmt.Sprintf("say %s for (%s) seconds", msg, secs)
		}
		return fmt.Sprintf("say [%s] for (%s) seconds", msg, secs)
	},
	"looks_think": func(b *RawBlock, p *ScratchBlocksParser) string {
		msg := p.getInputValue(b, "MESSAGE", "Hmm...")
		// If the message is already a reporter (starts with '(' or '<'), use it directly
		if strings.HasPrefix(msg, "(") || strings.HasPrefix(msg, "<") {
			return fmt.Sprintf("think %s", msg)
		}
		return fmt.Sprintf("think [%s]", msg)
	},
	"looks_thinkforsecs": func(b *RawBlock, p *ScratchBlocksParser) string {
		msg := p.getInputValue(b, "MESSAGE", "Hmm...")
		secs := p.getInputValue(b, "SECS", "2")
		// If the message is already a reporter (starts with '(' or '<'), use it directly
		if strings.HasPrefix(msg, "(") || strings.HasPrefix(msg, "<") {
			return fmt.Sprintf("think %s for (%s) seconds", msg, secs)
		}
		return fmt.Sprintf("think [%s] for (%s) seconds", msg, secs)
	},
	"looks_show": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "show"
	},
	"looks_hide": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "hide"
	},
	"looks_switchcostumeto": func(b *RawBlock, p *ScratchBlocksParser) string {
		costume := p.getInputValue(b, "COSTUME", "costume1")
		return fmt.Sprintf("switch costume to [%s v]", costume)
	},
	"looks_nextcostume": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "next costume"
	},
	"looks_switchbackdropto": func(b *RawBlock, p *ScratchBlocksParser) string {
		backdrop := p.getInputValue(b, "BACKDROP", "backdrop1")
		return fmt.Sprintf("switch backdrop to [%s v]", backdrop)
	},
	"looks_nextbackdrop": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "next backdrop"
	},
	"looks_changesizeby": func(b *RawBlock, p *ScratchBlocksParser) string {
		change := p.getInputValue(b, "CHANGE", "10")
		return fmt.Sprintf("change size by (%s)", change)
	},
	"looks_setsizeto": func(b *RawBlock, p *ScratchBlocksParser) string {
		size := p.getInputValue(b, "SIZE", "100")
		return fmt.Sprintf("set size to (%s)%%", size)
	},
	"looks_changeeffectby": func(b *RawBlock, p *ScratchBlocksParser) string {
		effect := p.getInputValue(b, "EFFECT", "color")
		change := p.getInputValue(b, "CHANGE", "25")
		return fmt.Sprintf("change [%s v] effect by (%s)", effect, change)
	},
	"looks_seteffectto": func(b *RawBlock, p *ScratchBlocksParser) string {
		effect := p.getInputValue(b, "EFFECT", "color")
		value := p.getInputValue(b, "VALUE", "0")
		return fmt.Sprintf("set [%s v] effect to (%s)", effect, value)
	},
	"looks_cleargraphiceffects": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "clear graphic effects"
	},
	"looks_gotofrontback": func(b *RawBlock, p *ScratchBlocksParser) string {
		position := p.getInputValue(b, "FRONT_BACK", "front")
		return fmt.Sprintf("go to [%s v] layer", position)
	},
	"looks_goforwardbackwardlayers": func(b *RawBlock, p *ScratchBlocksParser) string {
		direction := p.getInputValue(b, "FORWARD_BACKWARD", "forward")
		num := p.getInputValue(b, "NUM", "1")
		return fmt.Sprintf("go [%s v] (%s) layers", direction, num)
	},
	"looks_costumenumbername": func(b *RawBlock, p *ScratchBlocksParser) string {
		which := p.getInputValue(b, "NUMBER_NAME", "number")
		return fmt.Sprintf("costume [%s v]", which)
	},
	"looks_backdropnumbername": func(b *RawBlock, p *ScratchBlocksParser) string {
		which := p.getInputValue(b, "NUMBER_NAME", "number")
		return fmt.Sprintf("backdrop [%s v]", which)
	},
	"looks_size": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "size"
	},

	// Sound
	"sound_play": func(b *RawBlock, p *ScratchBlocksParser) string {
		sound := p.getInputValue(b, "SOUND_MENU", "pop")
		return fmt.Sprintf("start sound [%s v]", sound)
	},
	"sound_playuntildone": func(b *RawBlock, p *ScratchBlocksParser) string {
		sound := p.getInputValue(b, "SOUND_MENU", "pop")
		return fmt.Sprintf("play sound [%s v] until done", sound)
	},
	"sound_stop": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "stop all sounds"
	},
	"sound_changevolumeby": func(b *RawBlock, p *ScratchBlocksParser) string {
		volume := p.getInputValue(b, "VOLUME", "-10")
		return fmt.Sprintf("change volume by (%s)", volume)
	},
	"sound_setvolumeto": func(b *RawBlock, p *ScratchBlocksParser) string {
		volume := p.getInputValue(b, "VOLUME", "100")
		return fmt.Sprintf("set volume to (%s)%%", volume)
	},
	"sound_volume": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "volume"
	},

	// Control
	"control_wait": func(b *RawBlock, p *ScratchBlocksParser) string {
		duration := p.getInputValue(b, "DURATION", "1")
		return fmt.Sprintf("wait (%s) seconds", duration)
	},
	"control_repeat": func(b *RawBlock, p *ScratchBlocksParser) string {
		times := p.getInputValue(b, "TIMES", "10")
		return fmt.Sprintf("repeat (%s)", times)
	},
	"control_forever": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "forever"
	},
	"control_if": func(b *RawBlock, p *ScratchBlocksParser) string {
		condition := p.getInputValue(b, "CONDITION", "")
		return fmt.Sprintf("if <%s> then", condition)
	},
	"control_if_else": func(b *RawBlock, p *ScratchBlocksParser) string {
		condition := p.getInputValue(b, "CONDITION", "")
		return fmt.Sprintf("if <%s> then", condition)
	},
	"control_stop": func(b *RawBlock, p *ScratchBlocksParser) string {
		option := p.getInputValue(b, "STOP_OPTION", "all")
		return fmt.Sprintf("stop [%s v]", option)
	},
	"control_wait_until": func(b *RawBlock, p *ScratchBlocksParser) string {
		condition := p.getInputValue(b, "CONDITION", "")
		return fmt.Sprintf("wait until <%s>", condition)
	},
	"control_repeat_until": func(b *RawBlock, p *ScratchBlocksParser) string {
		condition := p.getInputValue(b, "CONDITION", "")
		return fmt.Sprintf("repeat until <%s>", condition)
	},
	"control_all": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "stop all"
	},
	"control_start_as_clone": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "when I start as a clone"
	},
	"control_create_clone_of": func(b *RawBlock, p *ScratchBlocksParser) string {
		clone := p.getInputValue(b, "CLONE_OPTION", "_myself_")
		return fmt.Sprintf("create clone of [%s v]", clone)
	},
	"control_delete_this_clone": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "delete this clone"
	},

	// Sensing
	"sensing_touchingobject": func(b *RawBlock, p *ScratchBlocksParser) string {
		object := p.getInputValue(b, "TOUCHINGOBJECTMENU", "_mouse_")
		return fmt.Sprintf("touching [%s v]?", object)
	},
	"sensing_touchingcolor": func(b *RawBlock, p *ScratchBlocksParser) string {
		color := p.getInputValue(b, "COLOR", "")
		return fmt.Sprintf("touching color (%s)?", color)
	},
	"sensing_distanceto": func(b *RawBlock, p *ScratchBlocksParser) string {
		object := p.getInputValue(b, "DISTANCETOMENU", "_mouse_")
		return fmt.Sprintf("distance to [%s v]", object)
	},
	"sensing_askandwait": func(b *RawBlock, p *ScratchBlocksParser) string {
		question := p.getInputValue(b, "QUESTION", "What's your name?")
		return fmt.Sprintf("ask [%s] and wait", question)
	},
	"sensing_setdragmode": func(b *RawBlock, p *ScratchBlocksParser) string {
		mode := p.getInputValue(b, "DRAG_MODE", "draggable")
		return fmt.Sprintf("set drag mode to [%s v]", mode)
	},
	"sensing_ask": func(b *RawBlock, p *ScratchBlocksParser) string {
		question := p.getInputValue(b, "QUESTION", "What's your name?")
		return fmt.Sprintf("ask [%s] and wait", question)
	},
	"sensing_answer": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "answer"
	},
	"sensing_keypressed": func(b *RawBlock, p *ScratchBlocksParser) string {
		key := p.getFieldValue(b, "KEY_OPTION", "space")
		return fmt.Sprintf("key [%s v] pressed?", key)
	},
	"sensing_mousedown": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "mouse down?"
	},
	"sensing_mousex": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "mouse x"
	},
	"sensing_mousey": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "mouse y"
	},
	"sensing_loudness": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "loudness"
	},
	"sensing_timer": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "timer"
	},
	"sensing_resettimer": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "reset timer"
	},
	"sensing_current": func(b *RawBlock, p *ScratchBlocksParser) string {
		menu := p.getInputValue(b, "CURRENTMENU", "YEAR")
		return fmt.Sprintf("[%s v]", menu)
	},
	"sensing_dayssince2000": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "days since 2000"
	},
	"sensing_userid": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "user id"
	},
	"sensing_username": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "username"
	},

	// Operators
	"operator_add": func(b *RawBlock, p *ScratchBlocksParser) string {
		num1 := p.getInputValue(b, "NUM1", "")
		num2 := p.getInputValue(b, "NUM2", "")
		return fmt.Sprintf("(%s) + (%s)", num1, num2)
	},
	"operator_subtract": func(b *RawBlock, p *ScratchBlocksParser) string {
		num1 := p.getInputValue(b, "NUM1", "")
		num2 := p.getInputValue(b, "NUM2", "")
		return fmt.Sprintf("(%s) - (%s)", num1, num2)
	},
	"operator_multiply": func(b *RawBlock, p *ScratchBlocksParser) string {
		num1 := p.getInputValue(b, "NUM1", "")
		num2 := p.getInputValue(b, "NUM2", "")
		return fmt.Sprintf("(%s) * (%s)", num1, num2)
	},
	"operator_divide": func(b *RawBlock, p *ScratchBlocksParser) string {
		num1 := p.getInputValue(b, "NUM1", "")
		num2 := p.getInputValue(b, "NUM2", "")
		return fmt.Sprintf("(%s) / (%s)", num1, num2)
	},
	"operator_random": func(b *RawBlock, p *ScratchBlocksParser) string {
		from := p.getInputValue(b, "FROM", "1")
		to := p.getInputValue(b, "TO", "10")
		return fmt.Sprintf("pick random (%s) to (%s)", from, to)
	},
	"operator_gt": func(b *RawBlock, p *ScratchBlocksParser) string {
		op1 := p.getInputValue(b, "OPERAND1", "")
		op2 := p.getInputValue(b, "OPERAND2", "")
		return fmt.Sprintf("(%s) > (%s)", op1, op2)
	},
	"operator_lt": func(b *RawBlock, p *ScratchBlocksParser) string {
		op1 := p.getInputValue(b, "OPERAND1", "")
		op2 := p.getInputValue(b, "OPERAND2", "")
		return fmt.Sprintf("(%s) < (%s)", op1, op2)
	},
	"operator_equals": func(b *RawBlock, p *ScratchBlocksParser) string {
		op1 := p.getInputValue(b, "OPERAND1", "")
		op2 := p.getInputValue(b, "OPERAND2", "")
		return fmt.Sprintf("(%s) = (%s)", op1, op2)
	},
	"operator_and": func(b *RawBlock, p *ScratchBlocksParser) string {
		op1 := p.getInputValue(b, "OPERAND1", "")
		op2 := p.getInputValue(b, "OPERAND2", "")
		return fmt.Sprintf("<%s> and <%s>", op1, op2)
	},
	"operator_or": func(b *RawBlock, p *ScratchBlocksParser) string {
		op1 := p.getInputValue(b, "OPERAND1", "")
		op2 := p.getInputValue(b, "OPERAND2", "")
		return fmt.Sprintf("<%s> or <%s>", op1, op2)
	},
	"operator_not": func(b *RawBlock, p *ScratchBlocksParser) string {
		op := p.getInputValue(b, "OPERAND", "")
		return fmt.Sprintf("not <%s>", op)
	},
	"operator_join": func(b *RawBlock, p *ScratchBlocksParser) string {
		str1 := p.getInputValue(b, "STRING1", "")
		str2 := p.getInputValue(b, "STRING2", "")
		return fmt.Sprintf("join [%s] [%s]", str1, str2)
	},
	"operator_letter": func(b *RawBlock, p *ScratchBlocksParser) string {
		letter := p.getInputValue(b, "LETTER", "1")
		str := p.getInputValue(b, "STRING", "")
		return fmt.Sprintf("letter (%s) of [%s]", letter, str)
	},
	"operator_length": func(b *RawBlock, p *ScratchBlocksParser) string {
		str := p.getInputValue(b, "STRING", "")
		return fmt.Sprintf("length of [%s]", str)
	},
	"operator_contains": func(b *RawBlock, p *ScratchBlocksParser) string {
		str1 := p.getInputValue(b, "STRING1", "")
		str2 := p.getInputValue(b, "STRING2", "")
		return fmt.Sprintf("[%s] contains [%s]?", str1, str2)
	},
	"operator_mod": func(b *RawBlock, p *ScratchBlocksParser) string {
		num1 := p.getInputValue(b, "NUM1", "")
		num2 := p.getInputValue(b, "NUM2", "")
		return fmt.Sprintf("(%s) mod (%s)", num1, num2)
	},
	"operator_round": func(b *RawBlock, p *ScratchBlocksParser) string {
		num := p.getInputValue(b, "NUM", "")
		return fmt.Sprintf("round (%s)", num)
	},
	"operator_mathop": func(b *RawBlock, p *ScratchBlocksParser) string {
		op := p.getInputValue(b, "OPERATOR", "")
		num := p.getInputValue(b, "NUM", "")
		return fmt.Sprintf("[%s v] of (%s)", op, num)
	},

	// Variables
	"data_setvariableto": func(b *RawBlock, p *ScratchBlocksParser) string {
		varName := p.getVariableName(b)
		value := p.getInputValue(b, "VALUE", "0")
		return fmt.Sprintf("set [%s v] to [%s]", varName, value)
	},
	"data_changevariableby": func(b *RawBlock, p *ScratchBlocksParser) string {
		varName := p.getVariableName(b)
		value := p.getInputValue(b, "VALUE", "1")
		return fmt.Sprintf("change [%s v] by (%s)", varName, value)
	},
	"data_showvariable": func(b *RawBlock, p *ScratchBlocksParser) string {
		varName := p.getVariableName(b)
		return fmt.Sprintf("show variable [%s v]", varName)
	},
	"data_hidevariable": func(b *RawBlock, p *ScratchBlocksParser) string {
		varName := p.getVariableName(b)
		return fmt.Sprintf("hide variable [%s v]", varName)
	},

	// Lists
	"data_addtolist": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		item := p.getInputValue(b, "ITEM", "thing")
		return fmt.Sprintf("add [%s] to [%s v]", item, listName)
	},
	"data_deleteoflist": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		index := p.getInputValue(b, "INDEX", "1")
		return fmt.Sprintf("delete (%s) of [%s v]", index, listName)
	},
	"data_deletealloflist": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		return fmt.Sprintf("delete all of [%s v]", listName)
	},
	"data_inserttolist": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		item := p.getInputValue(b, "ITEM", "thing")
		index := p.getInputValue(b, "INDEX", "1")
		return fmt.Sprintf("insert [%s] at (%s) of [%s v]", item, index, listName)
	},
	"data_replaceitemoflist": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		index := p.getInputValue(b, "INDEX", "1")
		item := p.getInputValue(b, "ITEM", "thing")
		return fmt.Sprintf("replace item (%s) of [%s v] with [%s]", index, listName, item)
	},
	"data_itemoflist": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		index := p.getInputValue(b, "INDEX", "1")
		return fmt.Sprintf("item (%s) of [%s v]", index, listName)
	},
	"data_itemnumoflist": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		item := p.getInputValue(b, "ITEM", "thing")
		return fmt.Sprintf("index of [%s] in [%s v]", item, listName)
	},
	"data_lengthoflist": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		return fmt.Sprintf("length of [%s v]", listName)
	},
	"data_listcontainsitem": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		item := p.getInputValue(b, "ITEM", "thing")
		return fmt.Sprintf("[%s v] contains [%s]?", listName, item)
	},
	"data_showlist": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		return fmt.Sprintf("show list [%s v]", listName)
	},
	"data_hidelist": func(b *RawBlock, p *ScratchBlocksParser) string {
		listName := p.getListName(b)
		return fmt.Sprintf("hide list [%s v]", listName)
	},

	// Procedures (Custom blocks)
	"procedures_definition": func(b *RawBlock, p *ScratchBlocksParser) string {
		// For custom blocks, we need to extract the procedure code from mutation
		if b.Mutation != nil {
			if mutation, ok := b.Mutation.(map[string]any); ok {
				if proccode, ok := mutation["proccode"].(string); ok {
					return fmt.Sprintf("define %s", proccode)
				}
			}
		}
		return "define"
	},
	"procedures_call": func(b *RawBlock, p *ScratchBlocksParser) string {
		if b.Mutation != nil {
			if mutation, ok := b.Mutation.(map[string]any); ok {
				if proccode, ok := mutation["proccode"].(string); ok {
					// Extract argument IDs from mutation in the correct order
					var argumentNames []string

					// argumentids can be either a string (comma-separated) or an array
					if argumentIDs, ok := mutation["argumentids"].([]any); ok {
						for _, id := range argumentIDs {
							if idStr, ok := id.(string); ok {
								argumentNames = append(argumentNames, idStr)
							}
						}
					} else if argumentIDs, ok := mutation["argumentids"].(string); ok {
						argumentNames = strings.Split(argumentIDs, ",")
					}

					// Get argument values in the correct order
					var args []string
					for _, argName := range argumentNames {
						// Input key format: argument_reporter_custom_paramN
						inputKey := fmt.Sprintf("argument_reporter_custom_%s", argName)
						value := p.getInputValue(b, inputKey, "")
						args = append(args, value)
					}

					// Replace placeholders in proccode with actual values
					// Placeholders: %s (string), %b (boolean), %n (number)
					if len(args) > 0 {
						result := proccode
						for _, arg := range args {
							// Replace first occurrence of any placeholder
							result = strings.Replace(result, "%s", arg, 1)
							result = strings.Replace(result, "%b", arg, 1)
							result = strings.Replace(result, "%n", arg, 1)
						}
						return result
					}
					return proccode
				}
			}
		}
		return "custom block"
	},

	// Additional sound blocks
	"sound_seteffectto": func(b *RawBlock, p *ScratchBlocksParser) string {
		effect := p.getInputValue(b, "EFFECT", "volume")
		value := p.getInputValue(b, "VALUE", "100")
		return fmt.Sprintf("set [%s v] effect to (%s)", effect, value)
	},
	"sound_cleareffects": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "clear sound effects"
	},

	// Additional sensing blocks
	"sensing_of": func(b *RawBlock, p *ScratchBlocksParser) string {
		property := p.getInputValue(b, "PROPERTY", "x position")
		object := p.getInputValue(b, "OBJECT", "_stage_")
		return fmt.Sprintf("(%s of [%s v])", property, object)
	},
	"sensing_coloristouchingcolor": func(b *RawBlock, p *ScratchBlocksParser) string {
		color1 := p.getInputValue(b, "COLOR", "")
		color2 := p.getInputValue(b, "COLOR2", "")
		return fmt.Sprintf("color (%s) is touching (%s)?", color1, color2)
	},

	// Additional pen blocks (if supported)
	"pen_penDown": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "pen down"
	},
	"pen_penUp": func(b *RawBlock, p *ScratchBlocksParser) string {
		return "pen up"
	},
	"pen_setPenColorToColor": func(b *RawBlock, p *ScratchBlocksParser) string {
		color := p.getInputValue(b, "COLOR", "")
		return fmt.Sprintf("set pen color to (%s)", color)
	},
	"pen_changePenSizeBy": func(b *RawBlock, p *ScratchBlocksParser) string {
		size := p.getInputValue(b, "SIZE", "1")
		return fmt.Sprintf("change pen size by (%s)", size)
	},
	"pen_setPenSizeTo": func(b *RawBlock, p *ScratchBlocksParser) string {
		size := p.getInputValue(b, "SIZE", "1")
		return fmt.Sprintf("set pen size to (%s)", size)
	},
}

// getInputValue extracts the value from an input
func (p *ScratchBlocksParser) getInputValue(block *RawBlock, inputName string, defaultValue string) string {
	input, exists := block.Inputs[inputName]
	if !exists || len(input) == 0 {
		return defaultValue
	}

	// Input format: [type, value]
	if len(input) >= 2 {
		inputType, typeOk := input[0].(float64)
		value := input[1]

		if !typeOk {
			// If type is not a number, try to return value directly
			if v, ok := value.(string); ok {
				return v
			}
			return defaultValue
		}

		// Handle different Scratch 3.0 input types
		// Type 1: Literal value (string or number)
		if inputType == 1 {
			// Check if value is an array format: [id, actualValue, ...]
			if arr, ok := value.([]any); ok && len(arr) >= 2 {
				// Format: [id, actualValue, ...]
				// Return the actual value
				if actualValue, ok := arr[1].(string); ok {
					return actualValue
				} else if actualValue, ok := arr[1].(float64); ok {
					return fmt.Sprintf("%.0f", actualValue)
				}
				return fmt.Sprintf("%v", arr[1])
			}
			// Regular value
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
		}

		// Type 2: Menu option - value is an ID, actual display value is in Fields
		if inputType == 2 {
			if menuID, ok := value.(string); ok {
				// Try to get the display value from Fields
				if fieldValue, exists := block.Fields[inputName]; exists && len(fieldValue) > 0 {
					return fieldValue[0]
				}
				// Fall back to the menu ID if field not found
				return menuID
			}
		}

		// Type 3: Nested block - return block ID (would need recursive parsing for full support)
		if inputType == 3 {
			if blockID, ok := value.(string); ok {
				// Try to get the field value corresponding to this block
				// For sensing blocks like "sensing_touchingobject", the field might be named differently
				// Check if there's a field with the same name as the input
				if fieldValue, exists := block.Fields[inputName]; exists && len(fieldValue) > 0 {
					return fieldValue[0]
				}
				// Return block ID as fallback
				return fmt.Sprintf("[block: %s]", blockID)
			}
		}

		// Type 4-13: References (variables, lists, costumes, sounds, etc.)
		// The value is usually the name directly, but could be an array format ["name", ...]
		if inputType >= 4 && inputType <= 13 {
			if v, ok := value.(string); ok {
				return v
			}
			// Handle array format: ["name", value, ...]
			if arr, ok := value.([]any); ok && len(arr) > 0 {
				if name, ok := arr[0].(string); ok {
					return name
				}
				// Fallback to string representation
				return fmt.Sprintf("%v", arr[0])
			}
		}

		// Fallback: try to convert to string
		if v, ok := value.(string); ok {
			return v
		}
		if v, ok := value.(float64); ok {
			return fmt.Sprintf("%.0f", v)
		}
		if arr, ok := value.([]any); ok && len(arr) > 0 {
			// Handle array values that weren't caught above
			return fmt.Sprintf("%v", arr[0])
		}
	}

	return defaultValue
}

// getFieldValue extracts the value from a field
func (p *ScratchBlocksParser) getFieldValue(block *RawBlock, fieldName string, defaultValue string) string {
	field, exists := block.Fields[fieldName]
	if !exists || len(field) == 0 {
		return defaultValue
	}
	
	// Field format: [value, nil] - value is already a string
	if len(field) >= 1 {
		return field[0]
	}
	
	return defaultValue
}

// getVariableName extracts the variable name from a variable block
func (p *ScratchBlocksParser) getVariableName(block *RawBlock) string {
	// Try to get from mutation first
	if block.Mutation != nil {
		if mutation, ok := block.Mutation.(map[string]any); ok {
			if varName, ok := mutation["variableName"].(string); ok {
				return varName
			}
		}
	}
	
	// Try to get from VARIABLE input
	return p.getInputValue(block, "VARIABLE", "variable")
}

// getListName extracts the list name from a list block
func (p *ScratchBlocksParser) getListName(block *RawBlock) string {
	// Try to get from mutation first
	if block.Mutation != nil {
		if mutation, ok := block.Mutation.(map[string]any); ok {
			if listName, ok := mutation["listName"].(string); ok {
				return listName
			}
		}
	}
	
	// Try to get from LIST input
	return p.getInputValue(block, "LIST", "list")
}

// ConvertBlock converts a single block to ScratchBlocks syntax
func (p *ScratchBlocksParser) ConvertBlock(block *RawBlock) string {
	// Look up opcode in map
	if fn, exists := BlockOpcodeMap[block.Opcode]; exists {
		return fn(block, p)
	}
	
	// Unknown opcode - return formatted representation
	return fmt.Sprintf("[%s]", strings.ReplaceAll(block.Opcode, "_", " "))
}

// ConvertScript converts a script chain to ScratchBlocks syntax
func (p *ScratchBlocksParser) ConvertScript(blocks map[string]RawBlock, topBlockID string) string {
	var result []string
	currentID := topBlockID
	
	for currentID != "" {
		block, exists := blocks[currentID]
		if !exists {
			break
		}
		
		// Convert this block
		blockText := p.ConvertBlock(&block)
		result = append(result, blockText)
		
		// Check if there's a next block
		if block.Next != nil {
			currentID = *block.Next
		} else {
			currentID = ""
		}
	}
	
	return strings.Join(result, "\n")
}

// GenerateScriptForDiff generates ScratchBlocks syntax for script diff items
func (p *ScratchBlocksParser) GenerateScriptForDiff(blocks map[string]RawBlock, topBlockID string) string {
	return p.ConvertScript(blocks, topBlockID)
}