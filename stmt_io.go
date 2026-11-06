package main

import (
	"fmt"
	"os"
	"strings"
)
// execPrint handles the PRINT statement, optionally directing output to a channel with #n.
func (ctx *execContext) execPrint(args []Token) {
	vm := ctx.vm
	if len(args) >= 2 && args[0].Type == TokHash {
		if chKey, rest, ok := parseChanArg(args); ok {
			if bc, exists := vm.chans.Load(chKey); exists && !bc.closed.Load() {
				bc.ch <- ctx.chanMessage(bc, rest)
			}
		}
		return
	}
	str, newline := ctx.evalPrintList(args)
	if newline {
		fmt.Println(str)
		vm.cursor.row.Add(1)
		vm.cursor.col.Store(1)
	} else {
		fmt.Print(str)
		vm.cursor.col.Add(int32(len([]rune(str))))
	}
}

// execInput handles the INPUT statement, reading `INPUT ["prompt";] target` from stdin, or CSV fields via `INPUT #n, a, b`.
func (ctx *execContext) execInput(args []Token) {
	if len(args) >= 2 && args[0].Type == TokHash {
		ctx.inputFromChan(args)
		return
	}
	prompt, targetStart := "", 0
	if len(args) >= 2 && args[0].Type == TokString && (args[1].Type == TokComma || args[1].Type == TokSemi) {
		prompt, targetStart = args[0].Text, 2
	}
	fmt.Print(prompt)
	line, err := ctx.vm.stdinRdr.ReadString('\n')
	if setter, _, _, _, ok := ctx.parseLValue(args[targetStart:]); ok {
		setter(strings.TrimSpace(line))
	}
	if err != nil && line == "" {
		ctx.setVar("EOF$", "1")
	} else {
		ctx.setVar("EOF$", "0")
	}
}

// inputFromChan handles input from a channel, assigning CSV fields to targets and pulling further lines when a line runs out.
func (ctx *execContext) inputFromChan(args []Token) {
	chKey, rest, ok := parseChanArg(args)
	if !ok || len(rest) < 1 {
		return
	}
	bc, exists := ctx.vm.chans.Load(chKey)
	if !exists {
		return
	}
	line, open := <-bc.ch
	if !open {
		ctx.setVar("EOF$", "1")
		return
	}
	ctx.setVar("EOF$", "0")
	fields := parseCSVLine(line)
	fieldIdx := 0
	rem := rest
	for len(rem) > 0 {
		if rem[0].Type == TokComma {
			rem = rem[1:]
			continue
		}
		setter, _, _, nextRem, lOk := ctx.parseLValue(rem)
		if !lOk {
			break
		}
		if fieldIdx < len(fields) {
			setter(fields[fieldIdx])
			fieldIdx++
		} else if nextLine, ok2 := <-bc.ch; ok2 {
			fields = parseCSVLine(nextLine)
			fieldIdx = 0
			if len(fields) > 0 {
				setter(fields[0])
				fieldIdx = 1
			}
		}
		rem = nextRem
	}
}

// execLine handles the LINE INPUT statement, optionally reading from a channel with #n.
func (ctx *execContext) execLine(args []Token) {
	if len(args) < 2 || !strings.EqualFold(args[0].Text, "INPUT") {
		return
	}
	inArgs := args[1:]
	if inArgs[0].Type != TokHash {
		ctx.execInput(inArgs)
		return
	}
	if chKey, rest, ok := parseChanArg(inArgs); ok && len(rest) >= 1 {
		if bc, exists := ctx.vm.chans.Load(chKey); exists {
			ctx.recvInto(bc, rest)
		}
	}
}

// execWrite handles the WRITE statement, emitting a CSV line with strings quoted: WRITE #n, a, b$.
func (ctx *execContext) execWrite(args []Token) {
	chKey, rest, ok := parseChanArg(args)
	if !ok {
		return
	}
	bc, exists := ctx.vm.chans.Load(chKey)
	if !exists || bc.closed.Load() {
		return
	}
	var parts []string
	appendField := func(toks []Token) {
		ev := ctx.evalTypedSubExpr(toks)
		if ev.isStr {
			parts = append(parts, `"`+ev.s+`"`)
		} else {
			parts = append(parts, ev.s)
		}
	}
	segStart, parenDepth := 0, 0
	for i, t := range rest {
		if t.Type == TokLParen {
			parenDepth++
		} else if t.Type == TokRParen && parenDepth > 0 {
			parenDepth--
		} else if parenDepth == 0 && (t.Type == TokComma || t.Type == TokSemi) {
			appendField(rest[segStart:i])
			segStart = i + 1
		}
	}
	if segStart < len(rest) {
		appendField(rest[segStart:])
	}
	line := strings.Join(parts, ",")
	if bc.isFileIO {
		line += "\r\n"
	}
	bc.ch <- line
}

// execOpen handles the OPEN statement, opening a file with OPEN file$ FOR INPUT|OUTPUT|APPEND AS #n.
func (ctx *execContext) execOpen(args []Token) error {
	forIdx, asIdx := -1, -1
	for i, t := range args {
		if t.Type == TokIdent {
			if strings.EqualFold(t.Text, "FOR") && forIdx == -1 {
				forIdx = i
			} else if strings.EqualFold(t.Text, "AS") && asIdx == -1 {
				asIdx = i
			}
		}
	}
	if forIdx <= 0 || asIdx <= forIdx+1 || asIdx+1 >= len(args) {
		return fmt.Errorf("invalid OPEN syntax")
	}
	filename := ctx.evalSubExpr(args[:forIdx])
	mode := strings.ToUpper(args[forIdx+1].Text)
	chKey := normalizeChanKey(ctx.evalSubExpr(args[asIdx+1:]))

	bc, err := openFileChan(filename, mode)
	if err != nil {
		return err
	}
	ctx.vm.chans.StoreChan(chKey, bc)
	if mode == "INPUT" {
		ctx.setVar("EOF$", "0")
	}
	return nil
}

// execClose handles the CLOSE statement, closing the specified channels or 
// all channels if no arguments are provided. It iterates over the listed channels, 
// or all of them with no arguments.
func (ctx *execContext) execClose(args []Token) {
	chans := ctx.vm.chans
	if len(args) == 0 {
		chans.CloseAll()
		return
	}
	for i := 0; i < len(args); i++ {
		if args[i].Type == TokComma {
			continue
		}
		if args[i].Type == TokHash && i+1 < len(args) {
			if bc, exists := chans.Load("#" + args[i+1].Text); exists {
				bc.Close()
			}
			i++
		} else if bc, exists := chans.Load(args[i].Text); exists {
			bc.Close()
		}
	}
}

// execName handles the NAME statement, renaming a file with NAME old$ AS new$.
func (ctx *execContext) execName(args []Token) error {
	asIdx := -1
	for i, t := range args {
		if t.Type == TokIdent && strings.EqualFold(t.Text, "AS") {
			asIdx = i
			break
		}
	}
	if asIdx > 0 && asIdx+1 < len(args) {
		return os.Rename(ctx.evalSubExpr(args[:asIdx]), ctx.evalSubExpr(args[asIdx+1:]))
	}
	return nil
}
