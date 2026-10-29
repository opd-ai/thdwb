// Package errors provides error logging and reporting for the thdwb browser:
// JS errors, permission denials, CSP violations, and other runtime errors.
package errors

import (
	"sync"
	"time"
)

// ErrorSeverity classifies the severity of a logged error.
type ErrorSeverity int

const (
	// SeverityInfo is an informational event (not an error).
	SeverityInfo ErrorSeverity = iota
	// SeverityWarning is a non-fatal warning.
	SeverityWarning
	// SeverityError is a recoverable error.
	SeverityError
	// SeverityCritical is a fatal or unrecoverable error.
	SeverityCritical
)

// String returns a human-readable severity label.
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ErrorEntry is a single logged error or event.
type ErrorEntry struct {
	Timestamp  time.Time
	Severity   ErrorSeverity
	Category   string // "js", "csp", "permission", "network", "extension", "other"
	Source     string // URL, extension ID, or component that produced the error
	Message    string
	Details    string
	StackTrace string
}

// Logger collects and reports errors.
type Logger struct {
	mu     sync.Mutex
	entries []*ErrorEntry
	max    int
}

// NewLogger creates a Logger with the given max entries.
func NewLogger(max int) *Logger {
	return &Logger{max: max}
}

// Log records an error entry.
func (l *Logger) Log(severity ErrorSeverity, category, source, message, details string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = append(l.entries, &ErrorEntry{
		Timestamp: time.Now(),
		Severity:  severity,
		Category:  category,
		Source:    source,
		Message:   message,
		Details:   details,
	})

	if len(l.entries) > l.max {
		l.entries = l.entries[len(l.entries)-l.max:]
	}
}

// LogJS records a JavaScript error.
func (l *Logger) LogJS(source, message, stackTrace string) {
	l.Log(SeverityError, "js", source, message, stackTrace)
}

// LogCSP records a CSP violation.
func (l *Logger) LogCSP(source, message, details string) {
	l.Log(SeverityWarning, "csp", source, message, details)
}

// LogPermission records a permission denial.
func (l *Logger) LogPermission(source, permission, details string) {
	l.Log(SeverityWarning, "permission", source, "permission denied: "+permission, details)
}

// LogExtension records an extension error.
func (l *Logger) LogExtension(extID, message, details string) {
	l.Log(SeverityError, "extension", extID, message, details)
}

// LogCritical records a critical error.
func (l *Logger) LogCritical(category, source, message, details string) {
	l.Log(SeverityCritical, category, source, message, details)
}

// Entries returns a copy of all logged entries.
func (l *Logger) Entries() []*ErrorEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]*ErrorEntry, len(l.entries))
	copy(out, l.entries)
	return out
}

// EntriesByCategory returns entries filtered by category.
func (l *Logger) EntriesByCategory(category string) []*ErrorEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	var out []*ErrorEntry
	for _, e := range l.entries {
		if e.Category == category {
			out = append(out, e)
		}
	}
	return out
}

// EntriesBySeverity returns entries filtered by minimum severity.
func (l *Logger) EntriesBySeverity(min ErrorSeverity) []*ErrorEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	var out []*ErrorEntry
	for _, e := range l.entries {
		if e.Severity >= min {
			out = append(out, e)
		}
	}
	return out
}

// Count returns the number of logged entries.
func (l *Logger) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// Clear removes all entries.
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = nil
}

// ErrorCounts returns a count of entries by category.
func (l *Logger) ErrorCounts() map[string]int {
	l.mu.Lock()
	defer l.mu.Unlock()

	counts := make(map[string]int)
	for _, e := range l.entries {
		counts[e.Category]++
	}
	return counts
}
