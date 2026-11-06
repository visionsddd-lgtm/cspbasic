package main

import (
	"strings"
	"sync"
)

// dataStore is the shared DATA pool consumed by READ and repositioned by RESTORE.
type dataStore struct {
	mu          sync.Mutex
	pool        []string
	lineOffsets map[int]int // DATA line number -> first pool index
	ptr         int
}

func (d *dataStore) load(p *program) {
	d.pool = nil
	d.lineOffsets = make(map[int]int)
	for _, lineNo := range p.order {
		toks := p.lines[lineNo]
		if len(toks) > 0 && strings.EqualFold(toks[0].Text, "DATA") {
			d.lineOffsets[lineNo] = len(d.pool)
			d.pool = append(d.pool, parseDataTokens(toks[1:])...)
		}
	}
}

func (d *dataStore) next() (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ptr >= len(d.pool) {
		return "", false
	}
	val := d.pool[d.ptr]
	d.ptr++
	return val, true
}

func (d *dataStore) restore(order []int, targetLine int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if targetLine <= 0 {
		d.ptr = 0
		return
	}
	for _, lineNo := range order {
		if lineNo >= targetLine {
			if offset, ok := d.lineOffsets[lineNo]; ok {
				d.ptr = offset
				return
			}
		}
	}
	d.ptr = len(d.pool)
}

func (d *dataStore) remaining() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return max(0, len(d.pool)-d.ptr)
}

func parseDataTokens(toks []Token) []string {
	var out []string
	var cur strings.Builder
	for _, t := range toks {
		if t.Type == TokComma {
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteString(t.Text)
	}
	if cur.Len() > 0 || len(toks) > 0 {
		out = append(out, strings.TrimSpace(cur.String()))
	}
	return out
}
