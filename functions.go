package main

import "strings"

// userFn is a DEF FN: single-line (body is an expression) or multi-line (program lines startIdx..endIdx).
type userFn struct {
	name     string
	params   []string
	body     []Token
	startIdx int
	endIdx   int
	isMulti  bool
}

// registerDefFn records `DEF FNX(params) = expr` or a multi-line DEF FN block; isMulti reports the latter.
func (vm *BasicVM) registerDefFn(idx int, args []Token) (isMulti bool) {
	if len(args) == 0 || strings.EqualFold(args[0].Text, "SEG") {
		return false
	}
	fnName, lpIdx := "", -1
	if strings.EqualFold(args[0].Text, "FN") && len(args) >= 2 && args[1].Type == TokIdent {
		fnName, lpIdx = "FN"+strings.ToUpper(args[1].Text), 2
	} else if strings.HasPrefix(strings.ToUpper(args[0].Text), "FN") {
		fnName, lpIdx = strings.ToUpper(args[0].Text), 1
	} else {
		return false
	}

	var params []string
	afterRP := lpIdx
	if lpIdx < len(args) && args[lpIdx].Type == TokLParen {
		for i := lpIdx + 1; i < len(args); i++ {
			if args[i].Type == TokRParen {
				afterRP = i + 1
				break
			}
			if args[i].Type == TokIdent {
				params = append(params, strings.ToUpper(args[i].Text))
			}
		}
	}
	if afterRP < len(args) && args[afterRP].Type == TokAssign {
		vm.funcs.Store(fnName, userFn{name: fnName, params: params, body: args[afterRP+1:], isMulti: false})
		return false
	}
	endIdx := vm.prog.findEndFn(idx)
	vm.funcs.Store(fnName, userFn{name: fnName, params: params, startIdx: idx + 1, endIdx: endIdx, isMulti: true})
	return true
}

func (ctx *execContext) callUserFn(fn userFn, fnArgs []string) string {
	frame := ctx.pushFrame()
	for _, param := range fn.params {
		frame.localScope[param] = true
	}
	for i, param := range fn.params {
		if i < len(fnArgs) {
			ctx.setVar(param, fnArgs[i])
		} else {
			ctx.setVar(param, defaultBySigil(param))
		}
	}

	var res string
	if !fn.isMulti {
		res = ctx.evalSubExpr(fn.body)
	} else {
		prog := &ctx.vm.prog
		ctx.run(linePC(fn.startIdx), func(pos pc) bool {
			return pos.idx >= 0 && pos.idx < fn.endIdx && pos.idx < len(prog.order) && !frame.returned
		})
		res = frame.retVal
		if res == "" {
			res = defaultBySigil(fn.name)
		}
	}

	ctx.popFrame()
	return coerceBySigil(fn.name, res)
}

// execDef handles DEF SEG and DEF FN; multi-line DEF FN bodies are skipped at definition time.
func (ctx *execContext) execDef(idx int, args []Token, cont pc) pc {
	if len(args) >= 1 && strings.EqualFold(args[0].Text, "SEG") {
		ctx.execDefSeg(args)
	} else if ctx.vm.registerDefFn(idx, args) {
		return linePC(ctx.vm.prog.findEndFn(idx) + 1)
	}
	return cont
}

// execFnStmt runs `FN RETURN expr` or a call used as a statement (FN X(...), APPLY(...)).
func (ctx *execContext) execFnStmt(toks []Token, cont pc) (pc, bool) {
	args := toks[1:]
	if strings.EqualFold(toks[0].Text, "FN") && len(args) >= 1 && strings.EqualFold(args[0].Text, "RETURN") {
		frame := ctx.topFrame()
		frame.retVal = ctx.evalSubExpr(args[1:])
		frame.returned = true
		return cont, true
	}
	_ = ctx.evalSubExpr(toks)
	return cont, false
}
