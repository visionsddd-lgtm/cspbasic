package main

import "sync/atomic"

// cursor tracks the 1-based text cursor for CSRLIN, POS and TAB.
type cursor struct {
	row atomic.Int32
	col atomic.Int32
}

var cgaFgANSI = [16]int{30, 34, 32, 36, 31, 35, 33, 37, 90, 94, 92, 96, 91, 95, 93, 97}
var cgaBgANSI = [8]int{40, 44, 42, 46, 41, 45, 43, 47}
