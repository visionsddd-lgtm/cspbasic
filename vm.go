package main

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"time"
)

// BasicVM holds state shared by every fiber; per-fiber state lives in execContext.
type BasicVM struct {
	prog program

	varsMu sync.RWMutex
	vars   map[string]string
	arrays *ArrayMap
	chans  *ChanMap

	templates   sync.Map // TemplateName -> []string
	varTypes    sync.Map // VarOrArrayName -> TemplateName
	funcs       sync.Map // FNNAME -> userFn
	lambdaCache sync.Map // RawLambdaString -> []Token

	data   dataStore
	mem    memory
	cursor cursor

	wg        sync.WaitGroup
	stdinRdr  *bufio.Reader
	startTime time.Time
}
// NewBasicVM creates and initializes a new BasicVM instance.
func NewBasicVM() *BasicVM {
	vm := &BasicVM{
		prog:      program{lines: make(map[int][]Token)},
		vars:      make(map[string]string),
		arrays:    NewArrayMap(),
		chans:     NewChanMap(),
		stdinRdr:  bufio.NewReader(os.Stdin),
		startTime: time.Now(),
	}
	vm.cursor.row.Store(1)
	vm.cursor.col.Store(1)
	return vm
}

// LoadProgram loads the given source code into the VM, registering any DEF FN functions.
func (vm *BasicVM) LoadProgram(src []byte) {
	vm.prog.load(src)
	vm.data.load(&vm.prog)
	for idx, lineNo := range vm.prog.order {
		toks := vm.prog.lines[lineNo]
		if len(toks) > 0 && strings.EqualFold(toks[0].Text, "DEF") {
			vm.registerDefFn(idx, toks[1:])
		}
	}
}

// RunFrom starts executing the program from the given index.
func (vm *BasicVM) RunFrom(startIdx int) {
	vm.runFiber(startIdx, true)
	vm.chans.CloseAll()
}

// runFiber executes the program starting from the given index in a new execution context.
func (vm *BasicVM) runFiber(startIdx int, isMain bool) {
	ctx := newExecContext(vm, isMain)
	ctx.run(linePC(startIdx), vm.prog.contains)
}

// spawn starts a new fiber executing the program from the given index.
func (vm *BasicVM) spawn(startIdx int) {
	vm.wg.Add(1)
	go func() {
		defer vm.wg.Done()
		vm.runFiber(startIdx, false)
	}()
}
