package main

import (
	"fmt"
	"strconv"
	"strings"
)

// execIf continues into the chosen branch's tokens; IF owns the rest of its line.
func (ctx *execContext) execIf(idx int, args []Token, cont pc) (pc, bool) {
	thenIdx, elseIdx, parenDepth, ifDepth := -1, -1, 0, 0
	for i, t := range args {
		if t.Type == TokLParen {
			parenDepth++
		} else if t.Type == TokRParen && parenDepth > 0 {
			parenDepth--
		} else if parenDepth == 0 && t.Type == TokIdent {
			w := strings.ToUpper(t.Text)
			if w == "IF" {
				ifDepth++
			} else if w == "THEN" && thenIdx == -1 && ifDepth == 0 {
				thenIdx = i
			} else if w == "ELSE" && thenIdx != -1 && ifDepth == 0 {
				elseIdx = i
				break
			} else if w == "ELSE" && ifDepth > 0 {
				ifDepth--
			}
		}
	}
	if thenIdx <= 0 {
		return cont, false
	}

	var branchToks []Token
	if isTruthy(ctx.evalSubExpr(args[:thenIdx])) {
		if elseIdx != -1 {
			branchToks = args[thenIdx+1 : elseIdx]
		} else {
			branchToks = args[thenIdx+1:]
		}
	} else if elseIdx != -1 {
		branchToks = args[elseIdx+1:]
	}
	if len(branchToks) == 0 {
		return cont, false
	}
	if target, ok := ctx.vm.prog.jumpTarget(branchToks); ok {
		return target, false
	}
	return pc{idx: idx, toks: branchToks}, false
}

func (ctx *execContext) execFor(args []Token, cont pc) {
	if len(args) < 5 || args[1].Type != TokAssign {
		return
	}
	varName := strings.ToUpper(args[0].Text)
	toIdx, stepIdx := -1, -1
	for i := 2; i < len(args); i++ {
		if args[i].Type == TokIdent && strings.EqualFold(args[i].Text, "TO") && toIdx == -1 {
			toIdx = i
		} else if args[i].Type == TokIdent && strings.EqualFold(args[i].Text, "STEP") && stepIdx == -1 {
			stepIdx = i
		}
	}
	if toIdx == -1 {
		return
	}
	startVal := ctx.evalNum(args[2:toIdx])
	var endVal, stepVal float64
	if stepIdx != -1 {
		endVal = ctx.evalNum(args[toIdx+1 : stepIdx])
		stepVal = ctx.evalNum(args[stepIdx+1:])
	} else {
		endVal = ctx.evalNum(args[toIdx+1:])
		stepVal = 1
	}
	ctx.setVar(varName, formatNum(startVal))
	ctx.topFrame().forLoops[varName] = forFrame{end: endVal, step: stepVal, body: cont}
}

func (ctx *execContext) execNext(args []Token, cont pc) pc {
	if len(args) < 1 {
		return cont
	}
	frame := ctx.topFrame()
	varName := strings.ToUpper(args[0].Text)
	fLoop, exists := frame.forLoops[varName]
	if !exists {
		return cont
	}
	val, _ := toFloat(ctx.getVar(varName))
	val += fLoop.step
	ctx.setVar(varName, formatNum(val))
	if (fLoop.step > 0 && val <= fLoop.end) || (fLoop.step < 0 && val >= fLoop.end) {
		return fLoop.body
	}
	delete(frame.forLoops, varName)
	return cont
}

func (ctx *execContext) execWhile(args []Token, cont pc) pc {
	if !isTruthy(ctx.evalSubExpr(args)) {
		return ctx.vm.prog.skipWhileBody(cont)
	}
	frame := ctx.topFrame()
	if len(frame.whileStack) == 0 || !samePC(frame.whileStack[len(frame.whileStack)-1].body, cont) {
		frame.whileStack = append(frame.whileStack, whileFrame{body: cont, condToks: args})
	}
	return cont
}

func (ctx *execContext) execWend(cont pc) pc {
	frame := ctx.topFrame()
	if len(frame.whileStack) > 0 {
		top := frame.whileStack[len(frame.whileStack)-1]
		if isTruthy(ctx.evalSubExpr(top.condToks)) {
			return top.body
		}
		frame.whileStack = frame.whileStack[:len(frame.whileStack)-1]
	}
	return cont
}

func (ctx *execContext) execGoto(args []Token) pc {
	return linePC(ctx.vm.prog.indexOf(int(ctx.evalNum(args))))
}

func (ctx *execContext) execGosub(args []Token, cont pc) pc {
	target := int(ctx.evalNum(args))
	frame := ctx.topFrame()
	frame.callStack = append(frame.callStack, cont)
	return linePC(ctx.vm.prog.indexOf(target))
}

// execReturn pops a GOSUB; with no GOSUB pending it returns from the fiber or FN.
func (ctx *execContext) execReturn(args []Token, cont pc) (pc, bool) {
	frame := ctx.topFrame()
	if len(frame.callStack) == 0 {
		if len(args) > 0 {
			frame.retVal = ctx.evalSubExpr(args)
		}
		frame.returned = true
		return cont, true
	}
	ret := frame.callStack[len(frame.callStack)-1]
	frame.callStack = frame.callStack[:len(frame.callStack)-1]
	return ret, false
}

// execOn handles ON ERROR GOTO n and ON expr GOTO/GOSUB a, b, ...
func (ctx *execContext) execOn(args []Token, cont pc) pc {
	if len(args) >= 3 && strings.EqualFold(args[0].Text, "ERROR") && strings.EqualFold(args[1].Text, "GOTO") {
		ctx.errGotoLine = int(ctx.evalNum(args[2:]))
		return cont
	}
	jumpType, kwIdx := "", -1
	for i, t := range args {
		if t.Type == TokIdent && (strings.EqualFold(t.Text, "GOTO") || strings.EqualFold(t.Text, "GOSUB")) {
			jumpType, kwIdx = strings.ToUpper(t.Text), i
			break
		}
	}
	if kwIdx <= 0 {
		return cont
	}
	choice := int(ctx.evalNum(args[:kwIdx]))
	var targets []int
	for _, t := range args[kwIdx+1:] {
		if t.Type == TokNumber {
			l, _ := strconv.Atoi(t.Text)
			targets = append(targets, l)
		}
	}
	if choice < 1 || choice > len(targets) {
		return cont
	}
	target := linePC(ctx.vm.prog.indexOf(targets[choice-1]))
	if jumpType == "GOSUB" {
		frame := ctx.topFrame()
		frame.callStack = append(frame.callStack, cont)
	}
	return target
}

func (ctx *execContext) execResume(args []Token) pc {
	if len(args) >= 1 && strings.EqualFold(args[0].Text, "NEXT") {
		return ctx.lastErrNext
	}
	if len(args) >= 1 {
		return linePC(ctx.vm.prog.indexOf(int(ctx.evalNum(args))))
	}
	return linePC(ctx.lastErrIdx)
}

// execEnd handles END, SYSTEM and the block terminators END SELECT / END FN / END DEF.
func (ctx *execContext) execEnd(args []Token, cont pc) (pc, bool) {
	if len(args) >= 1 {
		w := strings.ToUpper(args[0].Text)
		if w == "SELECT" {
			return cont, false
		}
		if w == "FN" || w == "DEF" {
			ctx.topFrame().returned = true
			return cont, true
		}
	}
	ctx.vm.chans.CloseAll()
	fmt.Print("\x1b[0m")
	return cont, true
}
