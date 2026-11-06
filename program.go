package main

import (
	"sort"
	"strconv"
	"strings"
)

// program is the immutable, line-numbered token listing plus block-structure lookups.
type program struct {
	lines map[int][]Token
	order []int // sorted line numbers; "idx" elsewhere is a position in this slice
}
// load parses the source code into line-numbered tokens and sorts the line numbers.
func (p *program) load(src []byte) {
	for _, rawLine := range strings.Split(string(src), "\n") {
		toks := LexLine(strings.TrimSpace(rawLine))
		if len(toks) < 1 || toks[0].Type != TokNumber {
			continue
		}
		lineNum, _ := strconv.Atoi(toks[0].Text)
		p.lines[lineNum] = toks[1:]
		p.order = append(p.order, lineNum)
	}
	sort.Ints(p.order)
}

// pc is an execution position: line idx in order plus the statements still to run on it.
type pc struct {
	idx  int
	toks []Token // nil means the whole line
}

func linePC(idx int) pc { return pc{idx: idx} }

// afterStmt is where execution continues once the statement preceding rest finishes.
func afterStmt(idx int, rest []Token) pc {
	if len(rest) == 0 {
		return linePC(idx + 1)
	}
	return pc{idx: idx, toks: rest}
}

func samePC(a, b pc) bool {
	return a.idx == b.idx && len(a.toks) == len(b.toks) && (len(a.toks) == 0 || &a.toks[0] == &b.toks[0])
}

// splitStmt splits off the first statement at a top-level ':'; IF owns the rest of its line.
func splitStmt(toks []Token) (stmt, rest []Token) {
	if len(toks) == 0 || strings.EqualFold(toks[0].Text, "IF") {
		return toks, nil
	}
	parenDepth := 0
	for i, t := range toks {
		if t.Type == TokLParen {
			parenDepth++
		} else if t.Type == TokRParen && parenDepth > 0 {
			parenDepth--
		} else if parenDepth == 0 && t.Type == TokColon {
			return toks[:i], toks[i+1:]
		}
	}
	return toks, nil
}

func (p *program) at(idx int) []Token { return p.lines[p.order[idx]] }

func (p *program) contains(pos pc) bool { return pos.idx >= 0 && pos.idx < len(p.order) }

func (p *program) tokensAt(pos pc) []Token {
	if pos.toks != nil {
		return pos.toks
	}
	return p.at(pos.idx)
}

func (p *program) indexOf(lineNum int) int { return sort.SearchInts(p.order, lineNum) }

// jumpTarget reports whether toks is a bare line number (as in `THEN 100`).
func (p *program) jumpTarget(toks []Token) (pc, bool) {
	if len(toks) == 1 && toks[0].Type == TokNumber {
		lineNum, _ := strconv.Atoi(toks[0].Text)
		return linePC(p.indexOf(lineNum)), true
	}
	return pc{}, false
}

func (p *program) findEndSelect(fromIdx int) int {
	depth := 1
	for i := fromIdx + 1; i < len(p.order); i++ {
		t := p.at(i)
		if len(t) == 0 {
			continue
		}
		if strings.EqualFold(t[0].Text, "SELECT") {
			depth++
		} else if isEndSelectLine(t) {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(p.order) - 1
}

// skipWhileBody scans statements from the loop body for the matching WEND and returns the position after it.
func (p *program) skipWhileBody(body pc) pc {
	depth := 1
	for pos := body; p.contains(pos); {
		stmt, rest := splitStmt(p.tokensAt(pos))
		next := afterStmt(pos.idx, rest)
		if len(stmt) > 0 {
			switch strings.ToUpper(stmt[0].Text) {
			case "WHILE":
				depth++
			case "WEND":
				depth--
				if depth == 0 {
					return next
				}
			}
		}
		pos = next
	}
	return linePC(len(p.order))
}

func (p *program) findEndFn(fromIdx int) int {
	for i := fromIdx + 1; i < len(p.order); i++ {
		if isEndFnTokens(p.at(i)) {
			return i
		}
	}
	return len(p.order) - 1
}

func isEndSelectLine(toks []Token) bool {
	if len(toks) == 0 {
		return false
	}
	return strings.EqualFold(toks[0].Text, "ENDSELECT") ||
		(len(toks) >= 2 && strings.EqualFold(toks[0].Text, "END") && strings.EqualFold(toks[1].Text, "SELECT"))
}

func isEndFnTokens(toks []Token) bool {
	if len(toks) == 0 {
		return false
	}
	w0 := strings.ToUpper(toks[0].Text)
	if w0 == "ENDFN" || w0 == "ENDDEF" {
		return true
	}
	if len(toks) >= 2 && w0 == "END" {
		w1 := strings.ToUpper(toks[1].Text)
		return w1 == "FN" || w1 == "DEF"
	}
	return false
}
