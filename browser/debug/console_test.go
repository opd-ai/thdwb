package debug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsoleCollectorAddAndEntries(t *testing.T) {
	cc := NewConsoleCollector(10)

	cc.Add("log", "https://example.com", "hello", "world")
	cc.Add("error", "ext1", "something failed")
	cc.Add("warn", "https://example.org", "deprecated")

	entries := cc.Entries()
	require.Len(t, entries, 3)
	assert.Equal(t, "log", entries[0].Level)
	assert.Equal(t, "https://example.com", entries[0].Origin)
	assert.Equal(t, "hello", entries[0].Message)
	assert.Equal(t, []interface{}{"world"}, entries[0].Args)
	assert.Equal(t, "error", entries[1].Level)
	assert.Equal(t, "ext1", entries[1].Origin)
	assert.Equal(t, "something failed", entries[1].Message)
	assert.Equal(t, "warn", entries[2].Level)
}

func TestConsoleCollectorMaxEntries(t *testing.T) {
	cc := NewConsoleCollector(3)

	cc.Add("log", "a", "one")
	cc.Add("log", "a", "two")
	cc.Add("log", "a", "three")
	cc.Add("log", "a", "four") // should evict "one"

	entries := cc.Entries()
	require.Len(t, entries, 3)
	assert.Equal(t, "two", entries[0].Message)
	assert.Equal(t, "three", entries[1].Message)
	assert.Equal(t, "four", entries[2].Message)
}

func TestConsoleCollectorErrorEntries(t *testing.T) {
	cc := NewConsoleCollector(10)

	cc.Add("log", "a", "ok")
	cc.Add("error", "a", "err1")
	cc.Add("warn", "a", "warn")
	cc.Add("error", "a", "err2")

	errors := cc.ErrorEntries()
	require.Len(t, errors, 2)
	assert.Equal(t, "err1", errors[0].Message)
	assert.Equal(t, "err2", errors[1].Message)
}

func TestConsoleCollectorClearAndCount(t *testing.T) {
	cc := NewConsoleCollector(10)

	cc.Add("log", "a", "one")
	assert.Equal(t, 1, cc.Count())

	cc.Clear()
	assert.Equal(t, 0, cc.Count())
	assert.Empty(t, cc.Entries())
}

func TestConsoleEntryFields(t *testing.T) {
	e := &ConsoleEntry{
		Timestamp:  time.Now(),
		Level:      "error",
		Origin:     "https://example.com",
		Message:    "failed to load",
		Args:       []interface{}{42, true},
	}
	assert.Equal(t, "error", e.Level)
	assert.Equal(t, "https://example.com", e.Origin)
	assert.Equal(t, "failed to load", e.Message)
	assert.Equal(t, []interface{}{42, true}, e.Args)
}
