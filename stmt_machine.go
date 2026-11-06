package main

import "fmt"
// execCls handles the CLS statement, clearing the screen and resetting the cursor position.
func (ctx *execContext) execCls() {
	fmt.Print("\x1b[2J\x1b[H")
	ctx.vm.cursor.row.Store(1)
	ctx.vm.cursor.col.Store(1)
}

// execLocate moves the cursor: LOCATE [row][, col]; omitted values keep their current position.
func (ctx *execContext) execLocate(args []Token) {
	cur := &ctx.vm.cursor
	parts := splitOnCommas(args)
	row, col := int(cur.row.Load()), int(cur.col.Load())
	if len(parts) >= 1 && len(parts[0]) > 0 {
		row = max(1, int(ctx.evalNum(parts[0])))
	}
	if len(parts) >= 2 && len(parts[1]) > 0 {
		col = max(1, int(ctx.evalNum(parts[1])))
	}
	cur.row.Store(int32(row))
	cur.col.Store(int32(col))
	fmt.Printf("\x1b[%d;%dH", row, col)
}

// execColor maps CGA COLOR fg[, bg] to ANSI SGR codes.
func (ctx *execContext) execColor(args []Token) {
	parts := splitOnCommas(args)
	if len(parts) < 1 || len(parts[0]) == 0 {
		return
	}
	fg := int(ctx.evalNum(parts[0])) & 15
	if len(parts) >= 2 && len(parts[1]) > 0 {
		bg := int(ctx.evalNum(parts[1])) & 7
		fmt.Printf("\x1b[%d;%dm", cgaFgANSI[fg], cgaBgANSI[bg])
	} else {
		fmt.Printf("\x1b[%dm", cgaFgANSI[fg])
	}
}

// execPoke handles POKE offset, value relative to the current DEF SEG.
func (ctx *execContext) execPoke(args []Token) {
	for i, t := range args {
		if t.Type == TokComma {
			offset := ctx.evalNum(args[:i])
			val := ctx.evalNum(args[i+1:])
			ctx.vm.mem.poke(int(offset), uint8(int(val)&0xFF))
			return
		}
	}
}

// execDefSeg handles DEF SEG [= segment]; bare DEF SEG resets to 0.
func (ctx *execContext) execDefSeg(args []Token) {
	if len(args) >= 3 && args[1].Type == TokAssign {
		ctx.vm.mem.setSegment(int(ctx.evalNum(args[2:])))
	} else {
		ctx.vm.mem.setSegment(0)
	}
}
