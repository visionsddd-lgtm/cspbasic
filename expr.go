package main

import (
	"math"
	"strconv"
	"strings"
)

// --- Type-Aware 1984 Expression Evaluator (recursive descent) ---

type evalVal struct {
	s     string
	isStr bool
}

type exprParser struct {
	ctx  *execContext
	toks []Token
	pos  int
}

func (ctx *execContext) evalSubExpr(tokens []Token) string { return ctx.evalTypedSubExpr(tokens).s }

func (ctx *execContext) evalNum(tokens []Token) float64 {
	f, _ := toFloat(ctx.evalSubExpr(tokens))
	return f
}

func (ctx *execContext) evalTypedSubExpr(tokens []Token) evalVal {
	if len(tokens) == 0 {
		return evalVal{"", true}
	}
	p := &exprParser{ctx: ctx, toks: tokens, pos: 0}
	return p.parseLogicOr()
}

// evalPrintList evaluates a PRINT-style `a; b, c` list; newline is false when it ends in ';'.
func (ctx *execContext) evalPrintList(tokens []Token) (out string, newline bool) {
	var sb strings.Builder
	newline = true
	segStart, parenDepth := 0, 0
	for i, t := range tokens {
		if t.Type == TokLParen {
			parenDepth++
		} else if t.Type == TokRParen && parenDepth > 0 {
			parenDepth--
		} else if parenDepth == 0 && (t.Type == TokSemi || t.Type == TokComma) {
			if i > segStart {
				sb.WriteString(formatRecordDisplay(ctx.evalSubExpr(tokens[segStart:i])))
			}
			if t.Type == TokComma {
				sb.WriteString("\t")
			} else if i == len(tokens)-1 {
				newline = false
			}
			segStart = i + 1
		}
	}
	if segStart < len(tokens) {
		sb.WriteString(formatRecordDisplay(ctx.evalSubExpr(tokens[segStart:])))
	}
	return sb.String(), newline
}

func isApplyName(name string) bool {
	switch name {
	case "APPLY", "APPLY%", "APPLY$", "APPLY@", "FNAPPLY", "FNAPPLY%", "FNAPPLY$", "FNAPPLY@":
		return true
	}
	return false
}

func isComparisonOp(t TokenType) bool {
	return t == TokAssign || t == TokLT || t == TokGT || t == TokLE || t == TokGE || t == TokNE
}

func compareTypedValues(left evalVal, op string, right evalVal) bool {
	if !left.isStr && !right.isStr {
		lNum, _ := toFloat(left.s)
		rNum, _ := toFloat(right.s)
		switch op {
		case "=":
			return lNum == rNum
		case "<>":
			return lNum != rNum
		case "<":
			return lNum < rNum
		case ">":
			return lNum > rNum
		case "<=":
			return lNum <= rNum
		case ">=":
			return lNum >= rNum
		}
	}
	switch op {
	case "=":
		return left.s == right.s
	case "<>":
		return left.s != right.s
	case "<":
		return left.s < right.s
	case ">":
		return left.s > right.s
	case "<=":
		return left.s <= right.s
	case ">=":
		return left.s >= right.s
	}
	return false
}

func toIndices(args []string) []int {
	idxs := make([]int, len(args))
	for i, s := range args {
		f, _ := toFloat(s)
		idxs[i] = int(f)
	}
	return idxs
}

func (p *exprParser) parseLogicOr() evalVal {
	left := p.parseLogicAnd()
	for p.pos < len(p.toks) && p.toks[p.pos].Type == TokIdent {
		op := strings.ToUpper(p.toks[p.pos].Text)
		if op != "OR" && op != "XOR" && op != "EQV" && op != "IMP" {
			break
		}
		p.pos++
		right := p.parseLogicAnd()
		lF, _ := toFloat(left.s)
		rF, _ := toFloat(right.s)
		lI, rI := int16(lF), int16(rF)
		switch op {
		case "OR":
			left = evalVal{strconv.Itoa(int(lI | rI)), false}
		case "XOR":
			left = evalVal{strconv.Itoa(int(lI ^ rI)), false}
		case "EQV":
			left = evalVal{strconv.Itoa(int(^(lI ^ rI))), false}
		case "IMP":
			left = evalVal{strconv.Itoa(int((^lI) | rI)), false}
		}
	}
	return left
}

func (p *exprParser) parseLogicAnd() evalVal {
	left := p.parseLogicNot()
	for p.pos < len(p.toks) && p.toks[p.pos].Type == TokIdent && strings.EqualFold(p.toks[p.pos].Text, "AND") {
		p.pos++
		right := p.parseLogicNot()
		lF, _ := toFloat(left.s)
		rF, _ := toFloat(right.s)
		left = evalVal{strconv.Itoa(int(int16(lF) & int16(rF))), false}
	}
	return left
}

func (p *exprParser) parseLogicNot() evalVal {
	if p.pos < len(p.toks) && p.toks[p.pos].Type == TokIdent && strings.EqualFold(p.toks[p.pos].Text, "NOT") {
		p.pos++
		if isTruthy(p.parseLogicNot().s) {
			return evalVal{"0", false}
		}
		return evalVal{"-1", false}
	}
	return p.parseComparison()
}

func (p *exprParser) parseComparison() evalVal {
	left := p.parseAddSub()
	for p.pos < len(p.toks) && isComparisonOp(p.toks[p.pos].Type) {
		op := p.toks[p.pos].Text
		p.pos++
		right := p.parseAddSub()
		if compareTypedValues(left, op, right) {
			left = evalVal{"-1", false}
		} else {
			left = evalVal{"0", false}
		}
	}
	return left
}

func (p *exprParser) parseAddSub() evalVal {
	left := p.parseMulDiv()
	for p.pos < len(p.toks) {
		op := p.toks[p.pos].Type
		if op != TokPlus && op != TokMinus {
			break
		}
		p.pos++
		right := p.parseMulDiv()
		if op == TokPlus {
			if left.isStr || right.isStr {
				left = evalVal{left.s + right.s, true}
			} else {
				lNum, _ := toFloat(left.s)
				rNum, _ := toFloat(right.s)
				left = evalVal{formatNum(lNum + rNum), false}
			}
		} else {
			lNum, _ := toFloat(left.s)
			rNum, _ := toFloat(right.s)
			left = evalVal{formatNum(lNum - rNum), false}
		}
	}
	return left
}

func (p *exprParser) parseMulDiv() evalVal {
	left := p.parsePower()
	for p.pos < len(p.toks) {
		op := p.toks[p.pos].Type
		if op != TokStar && op != TokSlash && op != TokIDiv && op != TokMod {
			break
		}
		p.pos++
		right := p.parsePower()
		lNum, _ := toFloat(left.s)
		rNum, _ := toFloat(right.s)
		switch op {
		case TokStar:
			left = evalVal{formatNum(lNum * rNum), false}
		case TokSlash:
			if rNum == 0 {
				left = evalVal{"0", false}
			} else {
				left = evalVal{formatNum(lNum / rNum), false}
			}
		case TokIDiv:
			if int64(rNum) == 0 {
				left = evalVal{"0", false}
			} else {
				left = evalVal{strconv.FormatInt(int64(lNum)/int64(rNum), 10), false}
			}
		case TokMod:
			if int64(rNum) == 0 {
				left = evalVal{"0", false}
			} else {
				left = evalVal{strconv.FormatInt(int64(lNum)%int64(rNum), 10), false}
			}
		}
	}
	return left
}

func (p *exprParser) parsePower() evalVal {
	left := p.parseUnary()
	if p.pos < len(p.toks) && p.toks[p.pos].Type == TokCaret {
		p.pos++
		right := p.parsePower()
		lNum, _ := toFloat(left.s)
		rNum, _ := toFloat(right.s)
		return evalVal{formatNum(math.Pow(lNum, rNum)), false}
	}
	return left
}

func (p *exprParser) parseUnary() evalVal {
	if p.pos < len(p.toks) {
		switch p.toks[p.pos].Type {
		case TokMinus:
			p.pos++
			val, _ := toFloat(p.parseUnary().s)
			return evalVal{formatNum(-val), false}
		case TokPlus:
			p.pos++
			return p.parseUnary()
		case TokHash:
			p.pos++
			return evalVal{"#" + p.parsePrimary().s, true}
		}
	}
	return p.parsePrimary()
}

// parseParenArgs consumes `(a, b, ...)`; for UBOUND the first arg is taken as a bare array name.
func (p *exprParser) parseParenArgs(specialUBound bool) []string {
	p.pos++
	var fnArgs []string
	if p.pos < len(p.toks) && p.toks[p.pos].Type != TokRParen {
		for {
			if specialUBound && len(fnArgs) == 0 && p.pos < len(p.toks) && p.toks[p.pos].Type == TokIdent {
				fnArgs = append(fnArgs, strings.ToUpper(p.toks[p.pos].Text))
				p.pos++
			} else {
				fnArgs = append(fnArgs, p.parseLogicOr().s)
			}
			if p.pos < len(p.toks) && p.toks[p.pos].Type == TokComma {
				p.pos++
				continue
			}
			break
		}
	}
	if p.pos < len(p.toks) && p.toks[p.pos].Type == TokRParen {
		p.pos++
	}
	return fnArgs
}

func (p *exprParser) parsePrimary() evalVal {
	if p.pos >= len(p.toks) {
		return evalVal{"0", false}
	}
	t := p.toks[p.pos]
	p.pos++
	switch t.Type {
	case TokLParen:
		val := p.parseLogicOr()
		if p.pos < len(p.toks) && p.toks[p.pos].Type == TokRParen {
			p.pos++
		}
		return val
	case TokNumber:
		return evalVal{t.Text, false}
	case TokString:
		return evalVal{t.Text, true}
	case TokIdent:
		return p.parseIdent(strings.ToUpper(t.Text))
	default:
		return evalVal{"", true}
	}
}

// parseIdent resolves a name to a record field, call (APPLY, FN, builtin), array element, or variable.
func (p *exprParser) parseIdent(name string) evalVal {
	ctx := p.ctx
	if name == "FN" && p.pos < len(p.toks) && p.toks[p.pos].Type == TokIdent {
		name = "FN" + strings.ToUpper(p.toks[p.pos].Text)
		p.pos++
	}
	isStrName := isStringOrRecName(name)

	if p.pos+1 < len(p.toks) && p.toks[p.pos].Type == TokDot && p.toks[p.pos+1].Type == TokIdent {
		field := strings.ToUpper(p.toks[p.pos+1].Text)
		fIsStr := isStringOrRecName(field)
		p.pos += 2
		if p.pos < len(p.toks) && p.toks[p.pos].Type == TokLParen {
			idxs := toIndices(p.parseParenArgs(false))
			return evalVal{ctx.vm.arrays.LoadOrAutoDim(name, len(idxs)).GetField(field, idxs), fIsStr}
		}
		val := ctx.getVar(name + "." + field)
		if val == "" && !fIsStr {
			return evalVal{"0", false}
		}
		return evalVal{val, fIsStr}
	}

	if p.pos < len(p.toks) && p.toks[p.pos].Type == TokLParen {
		fnArgs := p.parseParenArgs(name == "UBOUND")
		if p.pos+1 < len(p.toks) && p.toks[p.pos].Type == TokDot && p.toks[p.pos+1].Type == TokIdent {
			field := strings.ToUpper(p.toks[p.pos+1].Text)
			p.pos += 2
			idxs := toIndices(fnArgs)
			return evalVal{ctx.vm.arrays.LoadOrAutoDim(name, len(idxs)).GetField(field, idxs), isStringOrRecName(field)}
		}
		if isApplyName(name) {
			res, _ := ctx.execApply(name, fnArgs)
			return evalVal{res, isStrName || isPackedRecord(res)}
		}
		if fnAny, ok := ctx.vm.funcs.Load(name); ok {
			return evalVal{ctx.callUserFn(fnAny.(userFn), fnArgs), isStrName}
		}
		if res, isBuiltin := ctx.callBuiltin(name, fnArgs); isBuiltin {
			return evalVal{res, isStrName}
		}
		idxs := toIndices(fnArgs)
		arr := ctx.vm.arrays.LoadOrAutoDim(name, len(idxs))
		fields, isRec := ctx.vm.getTemplateFields(name)
		if isRec || strings.HasSuffix(name, "@") {
			return evalVal{arr.PackRecord(idxs, fields), true}
		}
		return evalVal{arr.Get(idxs), isStrName}
	}

	if fnAny, ok := ctx.vm.funcs.Load(name); ok {
		return evalVal{ctx.callUserFn(fnAny.(userFn), nil), isStrName}
	}
	if name == "INKEY$" || name == "TIMER" || name == "RND" || name == "DREM" || name == "CSRLIN" {
		res, _ := ctx.callBuiltin(name, nil)
		return evalVal{res, isStrName}
	}
	val := ctx.getVar(name)
	if val == "" && !isStrName {
		return evalVal{"0", false}
	}
	return evalVal{val, isStrName || isPackedRecord(val)}
}
