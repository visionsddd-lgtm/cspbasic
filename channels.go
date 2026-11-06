package main

import (
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
)

// --- Unified Thread-Safe Channel & Disk File Descriptor ---

type BasicChan struct {
	ch       chan string
	closed   atomic.Bool
	once     sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}
	isFileIO bool
}

func newMemChan(bufSz int) *BasicChan { return &BasicChan{ch: make(chan string, max(0, bufSz))} }

func (bc *BasicChan) Close() {
	bc.once.Do(func() {
		bc.closed.Store(true)
		if bc.stopCh != nil {
			close(bc.stopCh)
		}
		close(bc.ch)
	})
	if bc.doneCh != nil {
		<-bc.doneCh
	}
}

func (bc *BasicChan) IsEOF() bool { return bc.closed.Load() && len(bc.ch) == 0 }

type ChanMap struct{ m sync.Map }

func NewChanMap() *ChanMap { return &ChanMap{} }

func normalizeChanKey(key string) string {
	k := strings.ToUpper(strings.TrimSpace(key))
	if k != "" && !strings.HasPrefix(k, "#") && unicode.IsDigit(rune(k[0])) {
		return "#" + k
	}
	return k
}

func (c *ChanMap) Load(key string) (*BasicChan, bool) {
	v, ok := c.m.Load(normalizeChanKey(key))
	if !ok {
		return nil, false
	}
	return v.(*BasicChan), true
}

func (c *ChanMap) Store(key string, bufSz int) { c.StoreChan(key, newMemChan(bufSz)) }

func (c *ChanMap) StoreChan(key string, bc *BasicChan) {
	norm := normalizeChanKey(key)
	if old, ok := c.m.Load(norm); ok {
		old.(*BasicChan).Close()
	}
	c.m.Store(norm, bc)
}

func (c *ChanMap) CloseAll() {
	c.m.Range(func(_, v any) bool { v.(*BasicChan).Close(); return true })
}

// parseChanArg splits `#n, rest` or `name, rest` into a channel key and the remaining tokens.
func parseChanArg(args []Token) (chKey string, rest []Token, ok bool) {
	if len(args) >= 2 && args[0].Type == TokHash {
		if len(args) == 2 {
			return "#" + args[1].Text, nil, true
		}
		if args[2].Type == TokComma || args[2].Type == TokSemi {
			return "#" + args[1].Text, args[3:], true
		}
	}
	if len(args) >= 1 && (args[0].Type == TokIdent || args[0].Type == TokNumber) {
		if len(args) == 1 {
			return normalizeChanKey(args[0].Text), nil, true
		}
		if args[1].Type == TokComma || args[1].Type == TokSemi {
			return normalizeChanKey(args[0].Text), args[2:], true
		}
	}
	return "", nil, false
}
