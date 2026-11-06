package main

import (
	"strings"
	"sync"
)

// --- Thread-Safe N-Dimensional BASIC Array with per-element record fields ---

type BasicArray struct {
	mu     sync.RWMutex
	dims   []int
	total  int
	data   []string
	fields map[string][]string
	isStr  bool
}

func newBasicArray(name string, dims []int) *BasicArray {
	total := 1
	cleanDims := make([]int, len(dims))
	for i, d := range dims {
		cleanDims[i] = max(0, d)
		total *= (cleanDims[i] + 1)
	}
	return &BasicArray{
		dims:   cleanDims,
		total:  total,
		data:   make([]string, total),
		fields: make(map[string][]string),
		isStr:  strings.HasSuffix(name, "$") || strings.HasSuffix(name, "@"),
	}
}

func (a *BasicArray) flatIndex(indices []int) (int, bool) {
	if len(indices) != len(a.dims) {
		return 0, false
	}
	offset, stride := 0, 1
	for i := len(a.dims) - 1; i >= 0; i-- {
		if indices[i] < 0 || indices[i] > a.dims[i] {
			return 0, false
		}
		offset += indices[i] * stride
		stride *= (a.dims[i] + 1)
	}
	return offset, true
}

func (a *BasicArray) Get(indices []int) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	off, ok := a.flatIndex(indices)
	if !ok || a.data[off] == "" {
		if a.isStr {
			return ""
		}
		return "0"
	}
	return a.data[off]
}

func (a *BasicArray) Set(name string, indices []int, val string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if off, ok := a.flatIndex(indices); ok {
		a.data[off] = coerceBySigil(name, val)
	}
}

func (a *BasicArray) GetField(field string, indices []int) string {
	fKey := strings.ToUpper(field)
	a.mu.RLock()
	defer a.mu.RUnlock()
	off, ok := a.flatIndex(indices)
	if !ok {
		return defaultBySigil(fKey)
	}
	slice, exists := a.fields[fKey]
	if !exists || slice[off] == "" {
		return defaultBySigil(fKey)
	}
	return slice[off]
}

func (a *BasicArray) SetField(field string, indices []int, val string) {
	fKey := strings.ToUpper(field)
	a.mu.Lock()
	defer a.mu.Unlock()
	off, ok := a.flatIndex(indices)
	if !ok {
		return
	}
	slice, exists := a.fields[fKey]
	if !exists {
		slice = make([]string, a.total)
		a.fields[fKey] = slice
	}
	slice[off] = coerceBySigil(fKey, val)
}

// PackRecord serializes all record fields at indices: template fields first, then any extras.
func (a *BasicArray) PackRecord(indices []int, tmplFields []string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	off, ok := a.flatIndex(indices)
	if !ok {
		return ""
	}
	seen := make(map[string]bool)
	var parts []string
	for _, f := range tmplFields {
		fKey := strings.ToUpper(f)
		seen[fKey] = true
		val := defaultBySigil(fKey)
		if slice, exists := a.fields[fKey]; exists && slice[off] != "" {
			val = slice[off]
		}
		parts = append(parts, fKey+"="+val)
	}
	for fKey, slice := range a.fields {
		if !seen[fKey] {
			val := slice[off]
			if val == "" {
				val = defaultBySigil(fKey)
			}
			parts = append(parts, fKey+"="+val)
		}
	}
	return packRecord(parts)
}

func (a *BasicArray) UnpackRecord(indices []int, packed string, tmplFields []string) {
	forEachPackedField(packed, tmplFields, func(field, val string) {
		a.SetField(field, indices, val)
	})
}

func (a *BasicArray) Reset(dims []int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	fresh := newBasicArray("", dims)
	a.dims, a.total, a.data, a.fields = fresh.dims, fresh.total, fresh.data, fresh.fields
}

func (a *BasicArray) UBound(dim int) int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if dim < 1 || dim > len(a.dims) {
		return -1
	}
	return a.dims[dim-1]
}

type ArrayMap struct{ m sync.Map }

func NewArrayMap() *ArrayMap { return &ArrayMap{} }

func (am *ArrayMap) Dim(name string, dims []int) {
	key := strings.ToUpper(name)
	if v, ok := am.m.Load(key); ok {
		v.(*BasicArray).Reset(dims)
		return
	}
	am.m.Store(key, newBasicArray(key, dims))
}

func (am *ArrayMap) LoadOrAutoDim(name string, numDims int) *BasicArray {
	key := strings.ToUpper(name)
	if v, ok := am.m.Load(key); ok {
		return v.(*BasicArray)
	}
	defaultDims := make([]int, max(1, numDims))
	for i := range defaultDims {
		defaultDims[i] = 10
	}
	actual, _ := am.m.LoadOrStore(key, newBasicArray(key, defaultDims))
	return actual.(*BasicArray)
}

func (am *ArrayMap) Delete(name string) { am.m.Delete(strings.ToUpper(name)) }
