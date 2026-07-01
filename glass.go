package glass

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/monokkai/glass/internal/stackparse"
)

const (
	socketDir     = "/tmp/glass"
	errRingCap    = 20
	maxHotPaths   = 5
	stuckDwell    = 10 * time.Second
	snapshotEvery = 500 * time.Millisecond
)

type trackedGoroutine struct {
	state    string
	topFrame string
	since    time.Time
}

var (
	trackMu sync.Mutex
	tracked = map[int]trackedGoroutine{}
)

func stuckGoroutineIDs(gs []stackparse.Goroutine) map[int]bool {
	now := time.Now()
	next := make(map[int]trackedGoroutine, len(gs))
	stuck := map[int]bool{}

	trackMu.Lock()
	for _, g := range gs {
		since := now
		if prev, ok := tracked[g.ID]; ok && prev.state == g.State && prev.topFrame == g.TopFrame {
			since = prev.since
		}
		next[g.ID] = trackedGoroutine{state: g.State, topFrame: g.TopFrame, since: since}

		switch g.State {
		case "running", "runnable", "syscall", "sleep":
		default:
			if now.Sub(since) >= stuckDwell {
				stuck[g.ID] = true
			}
		}
	}
	tracked = next
	trackMu.Unlock()

	return stuck
}

type ErrorRecord struct {
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

type Snapshot struct {
	PID        int                  `json:"pid"`
	Goroutines int                  `json:"goroutines"`
	HeapAlloc  uint64               `json:"heap_alloc"`
	HeapSys    uint64               `json:"heap_sys"`
	HotPaths   []stackparse.HotPath `json:"hot_paths"`
	Errors     []ErrorRecord        `json:"errors"`
	Timestamp  time.Time            `json:"timestamp"`
}

var (
	errMu   sync.Mutex
	errRing = make([]ErrorRecord, 0, errRingCap)
)

func init() {
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		return
	}

	sockPath := filepath.Join(socketDir, strconv.Itoa(os.Getpid())+".sock")
	_ = os.Remove(sockPath)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return
	}

	go acceptLoop(ln)
}

func acceptLoop(ln net.Listener) {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go serveConn(conn)
	}
}

func serveConn(conn net.Conn) {
	defer conn.Close()

	enc := json.NewEncoder(conn)
	ticker := time.NewTicker(snapshotEvery)
	defer ticker.Stop()

	for range ticker.C {
		if err := enc.Encode(currentSnapshot()); err != nil {
			return
		}
	}
}

func currentSnapshot() Snapshot {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	dump := captureStack()
	goroutines := stackparse.Parse(dump)
	stuckIDs := stuckGoroutineIDs(goroutines)
	hotPaths := stackparse.Group(goroutines, stuckIDs)
	if len(hotPaths) > maxHotPaths {
		hotPaths = hotPaths[:maxHotPaths]
	}

	errMu.Lock()
	errsCopy := make([]ErrorRecord, len(errRing))
	copy(errsCopy, errRing)
	errMu.Unlock()

	return Snapshot{
		PID:        os.Getpid(),
		Goroutines: len(goroutines),
		HeapAlloc:  ms.HeapAlloc,
		HeapSys:    ms.HeapSys,
		HotPaths:   hotPaths,
		Errors:     errsCopy,
		Timestamp:  time.Now(),
	}
}

func captureStack() string {
	buf := make([]byte, 64*1024)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

func RecordError(err error) {
	if err == nil {
		return
	}
	errMu.Lock()
	defer errMu.Unlock()

	errRing = append(errRing, ErrorRecord{Message: err.Error(), At: time.Now()})
	if len(errRing) > errRingCap {
		errRing = errRing[len(errRing)-errRingCap:]
	}
}
