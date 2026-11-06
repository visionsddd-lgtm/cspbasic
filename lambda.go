package main

import (
	"fmt"
	"strings"
)

// --- Thread-Safe Anonymous Lambda Evaluator (FN APPLY / FN APPLY% / FN APPLY$ / FN APPLY@) ---

// Positional args bind to both names in each pair (arg 1 -> X and A, ...).
var lambdaPositionalAliases = [][2]string{
	{"X", "A"},
	{"Y", "B"},
	{"Z", "C"},
}
// isStatementKeyword reports whether a word is a BASIC statement keyword.
func isStatementKeyword(w string) bool {
	switch w {
	case "PRINT", "LET", "IF", "FOR", "NEXT", "WHILE", "WEND", "GOTO", "GOSUB", "RETURN", "FN",
		"SEND", "RECV", "CHAN", "SPAWN", "CLOSE", "SELECT", "CASE", "LOCAL", "SHARED", "GLOBAL",
		"POKE", "CLS", "LOCATE", "COLOR", "BEEP", "OPEN", "WRITE", "LINE", "INPUT", "READ",
		"RESTORE", "SWAP", "DIM", "REDIM", "ERASE", "KILL", "NAME", "SLEEP", "WAIT", "ON", "RESUME":
		return true
	}
	return false
}

// isLambdaStatement reports whether lambda code should be executed rather than evaluated.
func isLambdaStatement(toks []Token) bool {
	if len(toks) == 0 {
		return false
	}
	if toks[0].Type == TokIdent && isStatementKeyword(strings.ToUpper(toks[0].Text)) {
		return true
	}
	parenDepth := 0
	for _, t := range toks {
		if t.Type == TokLParen {
			parenDepth++
		} else if t.Type == TokRParen && parenDepth > 0 {
			parenDepth--
		} else if parenDepth == 0 && t.Type == TokColon {
			return true
		}
	}
	return false
}

// splitLambdaHeader parses the optional `(P1%, P2$) => body` header.
func splitLambdaHeader(code string) (params []string, body string) {
	arrowIdx := strings.Index(code, "=>")
	if arrowIdx == -1 {
		return nil, code
	}
	lhs := strings.TrimSpace(code[:arrowIdx])
	lhs = strings.TrimPrefix(strings.TrimSuffix(lhs, ")"), "(")
	for _, p := range strings.Split(lhs, ",") {
		pClean := strings.ToUpper(strings.TrimSpace(p))
		if pClean != "" {
			params = append(params, pClean)
		}
	}
	return params, strings.TrimSpace(code[arrowIdx+2:])
}
// bindAliasSet binds a base variable name to all its type-suffixed variants in the current frame and context.
func bindAliasSet(ctx *execContext, frame *stackFrame, base string, val string) {
	for _, suffix := range []string{"", "%", "$", "@"} {
		k := base + suffix
		frame.localScope[k] = true
		ctx.setVar(k, val)
	}
}
// lexLambdaCached lexes a lambda code string, caching the result for efficiency.
func (vm *BasicVM) lexLambdaCached(codeStr string) []Token {
	if cached, ok := vm.lambdaCache.Load(codeStr); ok {
		return cached.([]Token)
	}
	toks := LexLine(codeStr)
	vm.lambdaCache.Store(codeStr, toks)
	return toks
}

// execApply executes an anonymous formula or statement string inside an isolated stackFrame.
func (ctx *execContext) execApply(applyName string, args []string) (string, bool) {
	if len(args) == 0 {
		return "", true
	}
	explicitParams, codeStr := splitLambdaHeader(strings.TrimSpace(args[0]))
	params := args[1:]

	toks := ctx.vm.lexLambdaCached(codeStr)
	if len(toks) == 0 {
		return defaultBySigil(applyName), true
	}

	frame := ctx.pushFrame()
	for i, val := range params {
		if i < len(explicitParams) {
			ep := explicitParams[i]
			frame.localScope[baseVarName(ep)] = true
			ctx.setVar(ep, val)
		}
		if i < len(lambdaPositionalAliases) {
			bindAliasSet(ctx, frame, lambdaPositionalAliases[i][0], val)
			bindAliasSet(ctx, frame, lambdaPositionalAliases[i][1], val)
		}
		if i == 0 {
			bindAliasSet(ctx, frame, "OBJ", val)
			bindAliasSet(ctx, frame, "SELF", val)
		}
		bindAliasSet(ctx, frame, fmt.Sprintf("ARG%d", i+1), val)
	}

	var res string
	if isLambdaStatement(toks) {
		// Lambda code is detached from the listing: run until control leaves its own tokens.
		ctx.run(pc{toks: toks}, func(pos pc) bool { return pos.toks != nil })
		res = frame.retVal
	} else {
		res = ctx.evalSubExpr(toks)
	}

	ctx.popFrame()
	if res == "" {
		res = defaultBySigil(applyName)
	}
	return coerceBySigil(applyName, res), true
}
