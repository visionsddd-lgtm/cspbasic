package main

import (
	"reflect"
	"strings"
	"time"
)

// SELECT ... CASE RECV/SEND/TIMEOUT/ELSE ... END SELECT maps onto a dynamic reflect.Select.

type selectBranch struct {
	kind     string
	bc       *BasicChan
	setter   func(string)
	thenToks []Token
	bodyIdx  int
}

func (ctx *execContext) execSelect(selectIdx int) (pc, bool) {
	prog := &ctx.vm.prog
	endIdx := prog.findEndSelect(selectIdx)

	for {
		branches, cases, allRecvClosed := ctx.armSelect(selectIdx, endIdx)
		if allRecvClosed {
			ctx.setVar("EOF$", "1")
			return linePC(endIdx + 1), false
		}
		if len(cases) == 0 {
			return linePC(endIdx + 1), false
		}

		chosen, recvVal, recvOK := reflect.Select(cases)
		br := branches[chosen]
		if br.kind == "RECV" {
			if !recvOK {
				continue // channel closed under us; re-arm without it
			}
			br.setter(recvVal.String())
			ctx.setVar("EOF$", "0")
		}
		if len(br.thenToks) == 0 {
			return linePC(br.bodyIdx), false
		}
		if target, ok := prog.jumpTarget(br.thenToks); ok {
			return target, false
		}
		// Run the inline THEN statements, then continue after END SELECT.
		return pc{idx: endIdx, toks: br.thenToks}, false
	}
}

// armSelect builds one select case per top-level CASE line. allRecvClosed is true when
// RECV cases exist but every one of their channels is exhausted.
func (ctx *execContext) armSelect(selectIdx, endIdx int) (branches []selectBranch, cases []reflect.SelectCase, allRecvClosed bool) {
	vm := ctx.vm
	totalRecv, openRecv, depth := 0, 0, 0

	for i := selectIdx + 1; i < endIdx; i++ {
		t := vm.prog.at(i)
		if len(t) == 0 {
			continue
		}
		if strings.EqualFold(t[0].Text, "SELECT") {
			depth++
			continue
		}
		if isEndSelectLine(t) {
			depth--
			continue
		}
		if depth != 0 || !strings.EqualFold(t[0].Text, "CASE") || len(t) < 2 {
			continue
		}

		head, inlineThen := splitCaseThen(t[1:])
		if len(head) == 0 {
			continue
		}

		switch strings.ToUpper(head[0].Text) {
		case "RECV":
			totalRecv++
			if chKey, rest, ok := parseChanArg(head[1:]); ok && len(rest) >= 1 {
				if bc, exists := vm.chans.Load(chKey); exists && !bc.IsEOF() {
					if setter, _, _, _, lOk := ctx.parseLValue(rest); lOk {
						openRecv++
						branches = append(branches, selectBranch{kind: "RECV", bc: bc, setter: setter, thenToks: inlineThen, bodyIdx: i + 1})
						cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(bc.ch)})
					}
				}
			}
		case "SEND":
			if chKey, rest, ok := parseChanArg(head[1:]); ok && len(rest) >= 1 {
				if bc, exists := vm.chans.Load(chKey); exists && !bc.closed.Load() {
					val := ctx.chanMessage(bc, rest)
					branches = append(branches, selectBranch{kind: "SEND", bc: bc, thenToks: inlineThen, bodyIdx: i + 1})
					cases = append(cases, reflect.SelectCase{Dir: reflect.SelectSend, Chan: reflect.ValueOf(bc.ch), Send: reflect.ValueOf(val)})
				}
			}
		case "TIMEOUT":
			sec := ctx.evalNum(head[1:])
			dur := time.Duration(max(0.0, sec) * float64(time.Second))
			branches = append(branches, selectBranch{kind: "TIMEOUT", thenToks: inlineThen, bodyIdx: i + 1})
			cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(dur))})
		case "ELSE", "DEFAULT":
			branches = append(branches, selectBranch{kind: "ELSE", thenToks: inlineThen, bodyIdx: i + 1})
			cases = append(cases, reflect.SelectCase{Dir: reflect.SelectDefault})
		}
	}
	return branches, cases, totalRecv > 0 && openRecv == 0
}

// splitCaseThen splits `RECV #1, x$ THEN stmt` into its head and inline THEN body.
func splitCaseThen(caseArgs []Token) (head, inlineThen []Token) {
	for k, tok := range caseArgs {
		if tok.Type == TokIdent && strings.EqualFold(tok.Text, "THEN") {
			return caseArgs[:k], caseArgs[k+1:]
		}
	}
	return caseArgs, nil
}
