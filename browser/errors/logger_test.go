package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerLogAndEntries(t *testing.T) {
	l := NewLogger(10)

	l.Log(SeverityError, "js", "https://example.com", "TypeError: undefined is not a function", "at line 42")
	l.Log(SeverityWarning, "csp", "https://example.com", "script-src blocked", "inline script")

	entries := l.Entries()
	require.Len(t, entries, 2)
	assert.Equal(t, SeverityError, entries[0].Severity)
	assert.Equal(t, "js", entries[0].Category)
	assert.Equal(t, "https://example.com", entries[0].Source)
	assert.Equal(t, "TypeError: undefined is not a function", entries[0].Message)
	assert.Equal(t, "at line 42", entries[0].Details)
	assert.False(t, entries[0].Timestamp.IsZero())
	assert.Equal(t, SeverityWarning, entries[1].Severity)
	assert.Equal(t, "csp", entries[1].Category)
}

func TestLoggerMaxEntries(t *testing.T) {
	l := NewLogger(3)

	l.Log(SeverityInfo, "a", "s1", "m1", "d1")
	l.Log(SeverityInfo, "a", "s1", "m2", "d2")
	l.Log(SeverityInfo, "a", "s1", "m3", "d3")
	l.Log(SeverityInfo, "a", "s1", "m4", "d4") // evicts m1

	entries := l.Entries()
	require.Len(t, entries, 3)
	assert.Equal(t, "m2", entries[0].Message)
	assert.Equal(t, "m3", entries[1].Message)
	assert.Equal(t, "m4", entries[2].Message)
}

func TestLoggerLogJS(t *testing.T) {
	l := NewLogger(10)
	l.LogJS("https://example.com", "ReferenceError: x is not defined", "at foo (bar.js:10)")

	entries := l.EntriesByCategory("js")
	require.Len(t, entries, 1)
	assert.Equal(t, SeverityError, entries[0].Severity)
	assert.Equal(t, "https://example.com", entries[0].Source)
	assert.Equal(t, "ReferenceError: x is not defined", entries[0].Message)
	assert.Equal(t, "at foo (bar.js:10)", entries[0].Details)
}

func TestLoggerLogCSP(t *testing.T) {
	l := NewLogger(10)
	l.LogCSP("https://example.com", "script-src blocked 'unsafe-inline'", "policy: default-src 'self'")

	entries := l.EntriesByCategory("csp")
	require.Len(t, entries, 1)
	assert.Equal(t, SeverityWarning, entries[0].Severity)
	assert.Equal(t, "script-src blocked 'unsafe-inline'", entries[0].Message)
	assert.Equal(t, "policy: default-src 'self'", entries[0].Details)
}

func TestLoggerLogPermission(t *testing.T) {
	l := NewLogger(10)
	l.LogPermission("ext1", "tabs", "extension does not have tabs permission")

	entries := l.EntriesByCategory("permission")
	require.Len(t, entries, 1)
	assert.Equal(t, SeverityWarning, entries[0].Severity)
	assert.Equal(t, "ext1", entries[0].Source)
	assert.Equal(t, "permission denied: tabs", entries[0].Message)
	assert.Equal(t, "extension does not have tabs permission", entries[0].Details)
}

func TestLoggerLogExtension(t *testing.T) {
	l := NewLogger(10)
	l.LogExtension("ext1", "background script crashed", "panic: nil pointer")

	entries := l.EntriesByCategory("extension")
	require.Len(t, entries, 1)
	assert.Equal(t, SeverityError, entries[0].Severity)
	assert.Equal(t, "ext1", entries[0].Source)
	assert.Equal(t, "background script crashed", entries[0].Message)
	assert.Equal(t, "panic: nil pointer", entries[0].Details)
}

func TestLoggerLogCritical(t *testing.T) {
	l := NewLogger(10)
	l.LogCritical("network", "https://example.com", "TLS handshake failed", "certificate expired")

	entries := l.EntriesBySeverity(SeverityCritical)
	require.Len(t, entries, 1)
	assert.Equal(t, SeverityCritical, entries[0].Severity)
	assert.Equal(t, "network", entries[0].Category)
	assert.Equal(t, "TLS handshake failed", entries[0].Message)
}

func TestLoggerEntriesBySeverity(t *testing.T) {
	l := NewLogger(10)

	l.Log(SeverityInfo, "a", "s1", "info1", "d1")
	l.Log(SeverityWarning, "a", "s1", "warn1", "d2")
	l.Log(SeverityError, "a", "s1", "err1", "d3")
	l.Log(SeverityCritical, "a", "s1", "crit1", "d4")

	// Min severity = warning should include warning, error, critical.
	warn := l.EntriesBySeverity(SeverityWarning)
	require.Len(t, warn, 3)

	// Min severity = critical should include only critical.
	crit := l.EntriesBySeverity(SeverityCritical)
	require.Len(t, crit, 1)
	assert.Equal(t, "crit1", crit[0].Message)
}

func TestLoggerErrorCounts(t *testing.T) {
	l := NewLogger(10)

	l.Log(SeverityError, "js", "s1", "m1", "d1")
	l.Log(SeverityError, "js", "s1", "m2", "d2")
	l.Log(SeverityWarning, "csp", "s1", "m3", "d3")
	l.Log(SeverityError, "extension", "s1", "m4", "d4")

	counts := l.ErrorCounts()
	assert.Equal(t, 2, counts["js"])
	assert.Equal(t, 1, counts["csp"])
	assert.Equal(t, 1, counts["extension"])
}

func TestLoggerCountAndClear(t *testing.T) {
	l := NewLogger(10)

	assert.Equal(t, 0, l.Count())

	l.Log(SeverityInfo, "a", "s1", "m1", "d1")
	assert.Equal(t, 1, l.Count())

	l.Clear()
	assert.Equal(t, 0, l.Count())
	assert.Empty(t, l.Entries())
}

func TestErrorSeverityString(t *testing.T) {
	assert.Equal(t, "info", SeverityInfo.String())
	assert.Equal(t, "warning", SeverityWarning.String())
	assert.Equal(t, "error", SeverityError.String())
	assert.Equal(t, "critical", SeverityCritical.String())
}
