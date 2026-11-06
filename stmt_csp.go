package main

import "time"

// execChan creates (or replaces) an in-memory channel: CHAN #n, bufferSize.
func (ctx *execContext) execChan(args []Token) {
	if chKey, rest, ok := parseChanArg(args); ok && len(rest) >= 1 {
		ctx.vm.chans.Store(chKey, int(ctx.evalNum(rest)))
	}
}

// execSpawn starts a worker fiber at the given line; it runs until RETURN or END.
func (ctx *execContext) execSpawn(args []Token) {
	if len(args) >= 1 {
		ctx.vm.spawn(ctx.vm.prog.indexOf(int(ctx.evalNum(args))))
	}
}

func (ctx *execContext) execSend(args []Token) {
	if chKey, rest, ok := parseChanArg(args); ok && len(rest) >= 1 {
		if bc, exists := ctx.vm.chans.Load(chKey); exists && !bc.closed.Load() {
			bc.ch <- ctx.chanMessage(bc, rest)
		}
	}
}

func (ctx *execContext) execRecv(args []Token) {
	if chKey, rest, ok := parseChanArg(args); ok && len(rest) >= 1 {
		if bc, exists := ctx.vm.chans.Load(chKey); exists {
			ctx.recvInto(bc, rest)
		}
	}
}

func (ctx *execContext) execSleep(args []Token) {
	if len(args) >= 1 {
		time.Sleep(time.Duration(ctx.evalNum(args) * float64(time.Second)))
	}
}

// chanMessage evaluates a PRINT-style list for sending; file channels get CRLF line endings.
func (ctx *execContext) chanMessage(bc *BasicChan, toks []Token) string {
	val, newline := ctx.evalPrintList(toks)
	if bc.isFileIO && newline {
		val += "\r\n"
	}
	return val
}

// recvInto blocks for one message, assigns it to the target, and updates EOF$.
func (ctx *execContext) recvInto(bc *BasicChan, target []Token) {
	val, open := <-bc.ch
	if setter, _, _, _, ok := ctx.parseLValue(target); ok {
		setter(val)
	}
	if !open {
		ctx.setVar("EOF$", "1")
	} else {
		ctx.setVar("EOF$", "0")
	}
}
