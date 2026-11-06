package main

import (
	"sync"
	"sync/atomic"
)

// memory emulates a 1 MiB real-mode address space for PEEK/POKE with DEF SEG.
type memory struct {
	mu    sync.RWMutex
	seg   atomic.Uint32
	bytes [1 << 20]uint8
}

func (m *memory) addr(offset int) int {
	return (int(m.seg.Load())*16 + offset) & 0xFFFFF
}

func (m *memory) setSegment(seg int) { m.seg.Store(uint32(seg & 0xFFFF)) }

func (m *memory) peek(offset int) uint8 {
	a := m.addr(offset)
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bytes[a]
}

func (m *memory) poke(offset int, val uint8) {
	a := m.addr(offset)
	m.mu.Lock()
	m.bytes[a] = val
	m.mu.Unlock()
}
