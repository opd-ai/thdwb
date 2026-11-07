package extensions

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
)

// goroutineID returns the ID of the calling goroutine. It backs the reentrant
// VM lock: goja runtimes are not goroutine-safe, but extension event handlers
// legitimately re-enter the same VM from within a callback.
func goroutineID() uint64 {
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]
	// Format: "goroutine N ["
	buf = buf[len("goroutine "):]
	if i := bytes.IndexByte(buf, ' '); i >= 0 {
		buf = buf[:i]
	}
	id, err := strconv.ParseUint(string(buf), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// reentrantMutex is a mutex that the owning goroutine may lock repeatedly.
// Other goroutines block until the owner fully releases it.
//
// Extension background VMs need this because a single goja runtime must be
// driven by one goroutine at a time, yet a listener invoked by Fire* can call
// chrome.tabs.create (and similar), which fires another event on the same VM
// from the same goroutine. A plain sync.Mutex would deadlock on that re-entry.
type reentrantMutex struct {
	mu    sync.Mutex
	cond  *sync.Cond
	owner uint64
	depth int
}

func newReentrantMutex() *reentrantMutex {
	m := &reentrantMutex{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func (m *reentrantMutex) lock() {
	id := goroutineID()
	m.mu.Lock()
	for m.owner != 0 && m.owner != id {
		m.cond.Wait()
	}
	m.owner = id
	m.depth++
	m.mu.Unlock()
}

func (m *reentrantMutex) unlock() {
	id := goroutineID()
	m.mu.Lock()
	if m.owner != id {
		m.mu.Unlock()
		return
	}
	if m.depth > 0 {
		m.depth--
	}
	if m.depth == 0 {
		m.owner = 0
		m.cond.Broadcast()
	}
	m.mu.Unlock()
}
