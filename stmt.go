package main

import (
	"fmt"
	"os"
	"strings"
)

// Statement dispatch. Handlers live in stmt_*.go grouped by concern. Each statement receives
// cont (the position after it); handlers that alter control flow return a different pc.

// step executes the statement at pos and returns where to continue.
func (ctx *execContext) step(pos pc) (pc, bool) {
	stmt, rest := splitStmt(ctx.vm.prog.tokensAt(pos))
	return ctx.execSingleStmt(pos.idx, stmt, afterStmt(pos.idx, rest))
}

// run steps from pos while within(pos) holds, stopping early when a statement terminates.
func (ctx *execContext) run(pos pc, within func(pc) bool) (pc, bool) {
	for within(pos) {
		next, term := ctx.step(pos)
		if term || ctx.topFrame().returned {
			return next, term
		}
		pos = next
	}
	return pos, false
}

// execSingleStmt executes a single statement at pos. It returns the next position to execute and a boolean indicating if execution should terminate.
func (ctx *execContext) execSingleStmt(idx int, toks []Token, cont pc) (next pc, terminate bool) {
	if len(toks) == 0 {
		return cont, false
	}
	if ctx.tryAssign(toks) {
		return cont, false
	}
	cmd := strings.ToUpper(toks[0].Text)
	args := toks[1:]

	if isApplyName(cmd) || strings.HasPrefix(cmd, "FN") {
		return ctx.execFnStmt(toks, cont)
	}

	var err error
	switch cmd {
	case "REM", "DATA", "ENDSELECT":

	// Control flow (stmt_control.go)
	case "IF":
		return ctx.execIf(idx, args, cont)
	case "FOR":
		ctx.execFor(args, cont)
	case "NEXT":
		return ctx.execNext(args, cont), false
	case "WHILE":
		return ctx.execWhile(args, cont), false
	case "WEND":
		return ctx.execWend(cont), false
	case "GOTO":
		return ctx.execGoto(args), false
	case "GOSUB":
		return ctx.execGosub(args, cont), false
	case "RETURN":
		return ctx.execReturn(args, cont)
	case "ON":
		return ctx.execOn(args, cont), false
	case "RESUME":
		return ctx.execResume(args), false
	case "END", "SYSTEM":
		return ctx.execEnd(args, cont)
	case "ENDFN", "ENDDEF":
		ctx.topFrame().returned = true
		return cont, true

	// Variables, arrays and DATA (stmt_vars.go, functions.go)
	case "LET":
		ctx.tryAssign(args)
	case "LOCAL":
		ctx.execLocal(args)
	case "SHARED", "GLOBAL":
		ctx.execShared(args)
	case "SWAP":
		ctx.execSwap(args)
	case "DIM", "REDIM":
		ctx.execDim(args)
	case "ERASE":
		ctx.execErase(args)
	case "READ":
		ctx.execRead(args)
	case "RESTORE":
		ctx.execRestore(args)
	case "DEF":
		return ctx.execDef(idx, args, cont), false

	// Console and file I/O (stmt_io.go)
	case "PRINT":
		ctx.execPrint(args)
	case "INPUT":
		ctx.execInput(args)
	case "LINE":
		ctx.execLine(args)
	case "WRITE":
		ctx.execWrite(args)
	case "OPEN":
		err = ctx.execOpen(args)
	case "CLOSE":
		ctx.execClose(args)
	case "KILL":
		err = os.Remove(ctx.evalSubExpr(args))
	case "NAME":
		err = ctx.execName(args)

	// Channels and fibers (stmt_csp.go, select.go)
	case "CHAN":
		ctx.execChan(args)
	case "SPAWN":
		ctx.execSpawn(args)
	case "SEND":
		ctx.execSend(args)
	case "RECV":
		ctx.execRecv(args)
	case "WAIT":
		ctx.vm.wg.Wait()
	case "SLEEP":
		ctx.execSleep(args)
	case "SELECT":
		return ctx.execSelect(idx)
	case "CASE":
		return linePC(ctx.vm.prog.findEndSelect(idx) + 1), false

	// Screen and memory emulation (stmt_machine.go)
	case "CLS":
		ctx.execCls()
	case "BEEP":
		fmt.Print("\a")
	case "LOCATE":
		ctx.execLocate(args)
	case "COLOR":
		ctx.execColor(args)
	case "POKE":
		ctx.execPoke(args)

	default:
		err = fmt.Errorf("syntax error: %s", cmd)
	}

	if err != nil {
		return ctx.triggerError(idx, cont, err), false
	}
	return cont, false
}

// triggerError jumps to the ON ERROR GOTO handler if set, otherwise reports and skips the rest of the line.
func (ctx *execContext) triggerError(idx int, cont pc, err error) pc {
	if ctx.errGotoLine > 0 {
		ctx.lastErrIdx = idx
		ctx.lastErrNext = cont
		ctx.setVar("ERR$", err.Error())
		return linePC(ctx.vm.prog.indexOf(ctx.errGotoLine))
	}
	fmt.Println("I/O Error:", err)
	return linePC(idx + 1)
}
// splitOnCommas splits a list of tokens into sublists separated by commas.
func splitOnCommas(args []Token) [][]Token {
	var parts [][]Token
	start := 0
	for i, t := range args {
		if t.Type == TokComma {
			parts = append(parts, args[start:i])
			start = i + 1
		}
	}
	return append(parts, args[start:])
}
