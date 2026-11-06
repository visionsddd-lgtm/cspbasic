package main

import "strings"

// parseParenIndices evaluates `(i, j, ...)` at startPos, returning the indices and the position after ')'.
func (ctx *execContext) parseParenIndices(toks []Token, startPos int) ([]int, int, bool) {
	if startPos >= len(toks) || toks[startPos].Type != TokLParen {
		return nil, startPos, false
	}
	depth, closeIdx, segStart := 0, -1, startPos+1
	var subExprs [][]Token
	for i := startPos; i < len(toks); i++ {
		switch toks[i].Type {
		case TokLParen:
			depth++
		case TokRParen:
			depth--
			if depth == 0 {
				if i > segStart {
					subExprs = append(subExprs, toks[segStart:i])
				}
				closeIdx = i
			}
		case TokComma:
			if depth == 1 {
				subExprs = append(subExprs, toks[segStart:i])
				segStart = i + 1
			}
		}
		if closeIdx != -1 {
			break
		}
	}
	if closeIdx == -1 || len(subExprs) == 0 {
		return nil, startPos, false
	}
	idxs := make([]int, len(subExprs))
	for i, se := range subExprs {
		idxs[i] = int(ctx.evalNum(se))
	}
	return idxs, closeIdx + 1, true
}

// parseLValue resolves an assignable target (scalar, A(i), A(i).F, R@.F, R@.F(i), or record).
// fieldSetters is non-nil for whole records so READ can fill them field by field.
func (ctx *execContext) parseLValue(toks []Token) (setter func(string), getter func() string, fieldSetters []func(string), rest []Token, ok bool) {
	if len(toks) == 0 || toks[0].Type != TokIdent {
		return nil, nil, nil, nil, false
	}
	name := strings.ToUpper(toks[0].Text)

	if idxs, nextPos, pOk := ctx.parseParenIndices(toks, 1); pOk {
		arr := ctx.vm.arrays.LoadOrAutoDim(name, len(idxs))
		if nextPos+1 < len(toks) && toks[nextPos].Type == TokDot && toks[nextPos+1].Type == TokIdent {
			field := strings.ToUpper(toks[nextPos+1].Text)
			return func(v string) { arr.SetField(field, idxs, v) },
				func() string { return arr.GetField(field, idxs) },
				nil, toks[nextPos+2:], true
		}
		fields, isRec := ctx.vm.getTemplateFields(name)
		if isRec || strings.HasSuffix(name, "@") {
			var fSetters []func(string)
			for _, fName := range fields {
				fn := fName
				fSetters = append(fSetters, func(v string) { arr.SetField(fn, idxs, v) })
			}
			return func(packed string) { arr.UnpackRecord(idxs, packed, fields) },
				func() string { return arr.PackRecord(idxs, fields) },
				fSetters, toks[nextPos:], true
		}
		return func(v string) { arr.Set(name, idxs, v) },
			func() string { return arr.Get(idxs) },
			nil, toks[nextPos:], true
	}

	if len(toks) >= 3 && toks[1].Type == TokDot && toks[2].Type == TokIdent {
		field := strings.ToUpper(toks[2].Text)
		if idxs, nextPos, pOk := ctx.parseParenIndices(toks, 3); pOk {
			arr := ctx.vm.arrays.LoadOrAutoDim(name, len(idxs))
			return func(v string) { arr.SetField(field, idxs, v) },
				func() string { return arr.GetField(field, idxs) },
				nil, toks[nextPos:], true
		}
		compKey := name + "." + field
		return func(v string) { ctx.setVar(compKey, v) },
			func() string { return ctx.getVar(compKey) },
			nil, toks[3:], true
	}

	if fields, isRec := ctx.vm.getTemplateFields(name); isRec {
		fSetters := make([]func(string), len(fields))
		for i, fName := range fields {
			compKey := name + "." + fName
			fSetters[i] = func(v string) { ctx.setVar(compKey, v) }
		}
		return func(packed string) { ctx.setVar(name, packed) },
			func() string { return ctx.getVar(name) },
			fSetters, toks[1:], true
	}

	return func(v string) { ctx.setVar(name, v) },
		func() string { return ctx.getVar(name) },
		nil, toks[1:], true
}

// tryAssign executes `target = expr` and reports whether toks had that shape.
func (ctx *execContext) tryAssign(toks []Token) bool {
	setter, _, _, rest, ok := ctx.parseLValue(toks)
	if !ok || len(rest) < 2 || rest[0].Type != TokAssign {
		return false
	}
	setter(ctx.evalSubExpr(rest[1:]))
	return true
}
