package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	filename := "program.bas"
	if len(os.Args) >= 2 {
		filename = os.Args[1]
	}
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()
	prog, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	vm := NewBasicVM()
	vm.LoadProgram(prog)
	vm.RunFrom(0)
}
