package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Files are exposed to BASIC as BasicChans fed or drained by a background goroutine.

func openFileChan(filename, mode string) (*BasicChan, error) {
	switch mode {
	case "INPUT":
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		return newFileInputChan(f), nil
	case "OUTPUT":
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
		return newFileOutputChan(f), nil
	case "APPEND":
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		return newFileOutputChan(f), nil
	}
	return nil, fmt.Errorf("unsupported OPEN mode: %s", mode)
}

func newFileInputChan(f *os.File) *BasicChan {
	bc := &BasicChan{ch: make(chan string, 64), stopCh: make(chan struct{}), doneCh: make(chan struct{}), isFileIO: true}
	go func() {
		defer close(bc.doneCh)
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			select {
			case bc.ch <- sc.Text():
			case <-bc.stopCh:
				return
			}
		}
		bc.once.Do(func() { bc.closed.Store(true); close(bc.ch) })
	}()
	return bc
}

func newFileOutputChan(f *os.File) *BasicChan {
	bc := &BasicChan{ch: make(chan string, 64), stopCh: make(chan struct{}), doneCh: make(chan struct{}), isFileIO: true}
	go func() {
		defer close(bc.doneCh)
		defer f.Close()
		w := bufio.NewWriter(f)
		defer w.Flush()
		for msg := range bc.ch {
			_, _ = w.WriteString(msg)
		}
	}()
	return bc
}

func parseCSVLine(line string) []string {
	var fields []string
	var cur strings.Builder
	inQuotes := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '"' {
			inQuotes = !inQuotes
		} else if ch == ',' && !inQuotes {
			fields = append(fields, strings.TrimSpace(cur.String()))
			cur.Reset()
		} else {
			cur.WriteByte(ch)
		}
	}
	fields = append(fields, strings.TrimSpace(cur.String()))
	return fields
}
