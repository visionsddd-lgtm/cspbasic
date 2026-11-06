package main

import "strings"

// execLocal handles the LOCAL statement, marking variables as local and removing them from the shared scope.
func (ctx *execContext) execLocal(args []Token) {
	frame := ctx.topFrame()
	for _, t := range args {
		if t.Type == TokIdent {
			key := strings.ToUpper(t.Text)
			frame.localScope[key] = true
			delete(ctx.shared, key)
			ctx.setVar(key, defaultBySigil(key))
		}
	}
}
// execShared handles the SHARED statement, marking variables as shared and removing them from the local scope.
func (ctx *execContext) execShared(args []Token) {
	frame := ctx.topFrame()
	for _, t := range args {
		if t.Type == TokIdent {
			key := strings.ToUpper(t.Text)
			ctx.shared[key] = true
			delete(frame.localScope, key)
		}
	}
}
// execSwap handles the SWAP statement, exchanging the values of two variables.
func (ctx *execContext) execSwap(args []Token) {
	set1, get1, _, rest, ok := ctx.parseLValue(args)
	if !ok || len(rest) < 2 || rest[0].Type != TokComma {
		return
	}
	if set2, get2, _, _, ok2 := ctx.parseLValue(rest[1:]); ok2 {
		v1, v2 := get1(), get2()
		set1(v2)
		set2(v1)
	}
}

// execDim handles DIM TEMPLATE T@(fields...), DIM A(n, ...) [AS T@] and DIM V AS T@.
func (ctx *execContext) execDim(args []Token) {
	if len(args) >= 4 && args[0].Type == TokIdent && strings.EqualFold(args[0].Text, "TEMPLATE") {
		tName := strings.ToUpper(args[1].Text)
		if args[2].Type == TokLParen {
			var fields []string
			for i := 3; i < len(args); i++ {
				if args[i].Type == TokRParen {
					break
				}
				if args[i].Type == TokIdent {
					fields = append(fields, strings.ToUpper(args[i].Text))
				}
			}
			if len(fields) > 0 {
				ctx.vm.templates.Store(tName, fields)
				ctx.vm.varTypes.Store(tName, tName)
			}
		}
		return
	}

	for i := 0; i < len(args); i++ {
		if args[i].Type == TokComma {
			continue
		}
		if args[i].Type == TokIdent {
			varName := strings.ToUpper(args[i].Text)
			if idxs, nextPos, pOk := ctx.parseParenIndices(args, i+1); pOk {
				ctx.vm.arrays.Dim(varName, idxs)
				i = nextPos - 1
				if nextPos+1 < len(args) && strings.EqualFold(args[nextPos].Text, "AS") {
					ctx.vm.varTypes.Store(varName, strings.ToUpper(args[nextPos+1].Text))
					i = nextPos + 1
				}
			} else if i+2 < len(args) && strings.EqualFold(args[i+1].Text, "AS") {
				ctx.vm.varTypes.Store(varName, strings.ToUpper(args[i+2].Text))
				i += 2
			}
		}
	}
}

// execErase handles the ERASE statement, removing arrays from the VM.
func (ctx *execContext) execErase(args []Token) {
	for _, t := range args {
		if t.Type == TokIdent {
			ctx.vm.arrays.Delete(t.Text)
		}
	}
}

// execRead handles the READ statement, filling each target from the DATA pool; whole records consume one item per field.
func (ctx *execContext) execRead(args []Token) {
	readInto := func(set func(string)) {
		if val, hasData := ctx.vm.data.next(); !hasData {
			ctx.setVar("DATA_EOF$", "1")
		} else {
			ctx.setVar("DATA_EOF$", "0")
			set(val)
		}
	}
	rem := args
	for len(rem) > 0 {
		if rem[0].Type == TokComma {
			rem = rem[1:]
			continue
		}
		setter, _, fieldSetters, rest, ok := ctx.parseLValue(rem)
		if !ok {
			break
		}
		if len(fieldSetters) > 0 {
			for _, fs := range fieldSetters {
				readInto(fs)
			}
		} else {
			readInto(setter)
		}
		rem = rest
	}
}

// execRestore handles the RESTORE statement, resetting the DATA pointer to the specified line or the beginning.
func (ctx *execContext) execRestore(args []Token) {
	targetLine := 0
	if len(args) >= 1 {
		targetLine = int(ctx.evalNum(args))
	}
	ctx.vm.data.restore(ctx.vm.prog.order, targetLine)
	ctx.setVar("DATA_EOF$", "0")
}
