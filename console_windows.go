package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

var (
	msvcrt    = syscall.NewLazyDLL("msvcrt.dll")
	procKbhit = msvcrt.NewProc("_kbhit")
	procGetch = msvcrt.NewProc("_getch")
	kernel32  = syscall.NewLazyDLL("kernel32.dll")
	procGetH  = kernel32.NewProc("GetStdHandle")
	procSetM  = kernel32.NewProc("SetConsoleMode")
)

// init enables ANSI/VT escape processing on stdout.
func init() {
	const stdOut = ^uintptr(10)
	h, _, _ := procGetH.Call(stdOut)
	if h != 0 && h != ^uintptr(0) {
		procSetM.Call(h, 0x0007)
	}
}

func pollInkey() string {
	hit, _, _ := procKbhit.Call()
	if hit == 0 {
		time.Sleep(5 * time.Millisecond)
		return ""
	}
	ch, _, _ := procGetch.Call()
	if ch == 3 {
		fmt.Println("\n^C Break")
		os.Exit(130)
	}
	if ch == 0 || ch == 0xE0 {
		scan, _, _ := procGetch.Call()
		return string([]byte{0, byte(scan)})
	}
	return string(byte(ch))
}
