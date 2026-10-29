// Package debug provides debugging tools for the thdwb browser: a JS console
// log collector, extension inspector, and network request monitor.
package debug

import (
	"sync"
	"time"
)

// ConsoleEntry is a single console message captured from a page or extension.
type ConsoleEntry struct {
	Timestamp time.Time
	Level     string // "log", "error", "warn", "info"
	Origin    string // URL or extension ID that produced the message
	Message   string
	Args      []interface{}
}

// ConsoleCollector captures console messages for debugging.
type ConsoleCollector struct {
	mu   sync.Mutex
	entries []*ConsoleEntry
	max  int
}

// NewConsoleCollector creates a ConsoleCollector with the given max entries.
func NewConsoleCollector(max int) *ConsoleCollector {
	return &ConsoleCollector{max: max}
}

// Add records a console message.
func (cc *ConsoleCollector) Add(level, origin, message string, args ...interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.entries = append(cc.entries, &ConsoleEntry{
		Timestamp: time.Now(),
		Level:     level,
		Origin:    origin,
		Message:   message,
		Args:      args,
	})

	if len(cc.entries) > cc.max {
		cc.entries = cc.entries[len(cc.entries)-cc.max:]
	}
}

// Entries returns a copy of all captured console entries.
func (cc *ConsoleCollector) Entries() []*ConsoleEntry {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	out := make([]*ConsoleEntry, len(cc.entries))
	copy(out, cc.entries)
	return out
}

// ErrorEntries returns only error-level entries.
func (cc *ConsoleCollector) ErrorEntries() []*ConsoleEntry {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	var out []*ConsoleEntry
	for _, e := range cc.entries {
		if e.Level == "error" {
			out = append(out, e)
		}
	}
	return out
}

// Clear removes all entries.
func (cc *ConsoleCollector) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.entries = nil
}

// Count returns the number of captured entries.
func (cc *ConsoleCollector) Count() int {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return len(cc.entries)
}
