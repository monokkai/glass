package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

type hotPath struct {
	Frame string `json:"frame"`
	Count int    `json:"count"`
	Stuck bool   `json:"stuck"`
}

type snapshot struct {
	PID        int       `json:"pid"`
	Goroutines int       `json:"goroutines"`
	HeapAlloc  uint64    `json:"heap_alloc"`
	HeapSys    uint64    `json:"heap_sys"`
	HotPaths   []hotPath `json:"hot_paths"`
	Errors     []struct {
		Message string    `json:"message"`
		At      time.Time `json:"at"`
	} `json:"errors"`
}

func main() {
	if len(os.Args) < 3 || os.Args[1] != "attach" {
		fmt.Fprintln(os.Stderr, "usage: glass attach <pid>")
		os.Exit(1)
	}

	pid := os.Args[2]
	sockPath := filepath.Join("/tmp/glass", pid+".sock")

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not attach to pid %s: %v\n", pid, err)
		fmt.Fprintln(os.Stderr, "is that process importing github.com/you/glass?")
		os.Exit(1)
	}
	defer conn.Close()

	dec := json.NewDecoder(bufio.NewReader(conn))
	first := true
	for {
		var snap snapshot
		if err := dec.Decode(&snap); err != nil {
			fmt.Fprintln(os.Stderr, "\nconnection closed:", err)
			return
		}
		render(snap, first)
		first = false
	}
}

const maxHotPaths = 5

const linesRendered = 3 + 1 + 1 + maxHotPaths + 1 + 1 + 1

func render(s snapshot, first bool) {
	if !first {
		fmt.Printf("\033[%dA", linesRendered)
	}
	const clearLine = "\033[2K\r"

	fmt.Printf("%sglass · pid %d\n", clearLine, s.PID)
	fmt.Printf("%sgoroutines: %d\n", clearLine, s.Goroutines)
	fmt.Printf("%sheap: %s / %s\n", clearLine, humanBytes(s.HeapAlloc), humanBytes(s.HeapSys))
	fmt.Printf("%s\n", clearLine)

	fmt.Printf("%shot paths:\n", clearLine)
	for i := 0; i < maxHotPaths; i++ {
		if i >= len(s.HotPaths) {
			fmt.Printf("%s\n", clearLine)
			continue
		}
		hp := s.HotPaths[i]
		marker := " "
		suffix := ""
		if hp.Stuck {
			marker = "⚠"
			suffix = " [stuck]"
		}
		fmt.Printf("%s  %s %-3d %s%s\n", clearLine, marker, hp.Count, hp.Frame, suffix)
	}

	fmt.Printf("%s\n", clearLine)
	fmt.Printf("%slast error:\n", clearLine)
	if len(s.Errors) == 0 {
		fmt.Printf("%s  (none)\n", clearLine)
	} else {
		last := s.Errors[len(s.Errors)-1]
		fmt.Printf("%s  %s\n", clearLine, last.Message)
	}
}

func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
