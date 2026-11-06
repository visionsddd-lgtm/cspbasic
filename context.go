package main

import "strings"

// execContext is per-fiber state: the main program and each SPAWNed worker get their own.
type execContext struct {
	vm          *BasicVM
	isMain      bool
	frames      []*stackFrame
	shared      map[string]bool
	errGotoLine int
	lastErrIdx  int // line of the last trapped error, for RESUME
	lastErrNext pc  // statement after it, for RESUME NEXT
}

type stackFrame struct {
	locals     map[string]string
	localScope map[string]bool
	forLoops   map[string]forFrame
	whileStack []whileFrame
	callStack  []pc
	retVal     string
	returned   bool
}

type forFrame struct {
	end  float64
	step float64
	body pc
}

type whileFrame struct {
	body     pc
	condToks []Token
}

func newExecContext(vm *BasicVM, isMain bool) *execContext {
	return &execContext{
		vm:     vm,
		isMain: isMain,
		frames: []*stackFrame{newStackFrame()},
		shared: make(map[string]bool),
	}
}

func newStackFrame() *stackFrame {
	return &stackFrame{
		locals:     make(map[string]string),
		localScope: make(map[string]bool),
		forLoops:   make(map[string]forFrame),
	}
}

func (ctx *execContext) topFrame() *stackFrame {
	return ctx.frames[len(ctx.frames)-1]
}

func (ctx *execContext) pushFrame() *stackFrame {
	frame := newStackFrame()
	ctx.frames = append(ctx.frames, frame)
	return frame
}

func (ctx *execContext) popFrame() { ctx.frames = ctx.frames[:len(ctx.frames)-1] }

// resolveTargetMap returns the local map owning baseKey, or isGlobal=true for vm.vars.
// Workers default to fiber-local root variables unless the name was declared SHARED.
func (ctx *execContext) resolveTargetMap(baseKey string) (m map[string]string, isGlobal bool) {
	for i := len(ctx.frames) - 1; i >= 1; i-- {
		if ctx.frames[i].localScope[baseKey] {
			return ctx.frames[i].locals, false
		}
	}
	root := ctx.frames[0]
	if root.localScope[baseKey] || (!ctx.isMain && !ctx.shared[baseKey]) {
		return root.locals, false
	}
	return nil, true
}

func (ctx *execContext) getVar(name string) string {
	key := strings.ToUpper(name)
	baseKey := baseVarName(key)
	m, isGlobal := ctx.resolveTargetMap(baseKey)

	if strings.HasSuffix(key, "@") && !strings.Contains(key, ".") {
		prefix := key + "."
		if !isGlobal {
			if parts := collectRecordFields(m, prefix); len(parts) > 0 {
				return packRecord(parts)
			}
			if val, ok := m[key]; ok && val != "" {
				return val
			}
			if ctx.topFrame().localScope[baseKey] {
				return ""
			}
		}
		// Fall back to globals so SPAWNed workers can read global @ records.
		ctx.vm.varsMu.RLock()
		parts := collectRecordFields(ctx.vm.vars, prefix)
		raw := ctx.vm.vars[key]
		ctx.vm.varsMu.RUnlock()
		if len(parts) > 0 {
			return packRecord(parts)
		}
		return raw
	}

	if !isGlobal {
		if val, ok := m[key]; ok {
			return val
		}
		if ctx.topFrame().localScope[baseKey] {
			return defaultBySigil(key)
		}
	}
	ctx.vm.varsMu.RLock()
	defer ctx.vm.varsMu.RUnlock()
	return ctx.vm.vars[key]
}

func (ctx *execContext) setVar(name, val string) {
	key := strings.ToUpper(name)
	val = coerceBySigil(key, val)
	baseKey := baseVarName(key)
	m, isGlobal := ctx.resolveTargetMap(baseKey)

	isRecVar := (strings.HasSuffix(key, "@") && !strings.Contains(key, ".")) || isPackedRecord(val)
	tmplFields, hasTmpl := ctx.vm.getTemplateFields(key)

	if isGlobal {
		ctx.vm.varsMu.Lock()
		if isRecVar || hasTmpl {
			unpackRecordIntoMap(ctx.vm.vars, key, val, tmplFields)
		}
		ctx.vm.vars[key] = val
		ctx.vm.varsMu.Unlock()
		return
	}

	if isRecVar || hasTmpl {
		unpackRecordIntoMap(m, key, val, tmplFields)
	}
	m[key] = val
}
