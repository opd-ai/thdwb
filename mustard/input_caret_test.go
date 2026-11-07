package mustard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSetValueClampsCursor guards the caret panic: cursorPosition is a negative
// offset from the end of the value, and SetValue used to leave it stale when the
// new value was shorter, so draw()'s value[len(value)+cursorPosition:] slice
// panicked with a negative index.
func TestSetValueClampsCursor(t *testing.T) {
	in := &InputWidget{}
	in.value = "http://example.com/some/long/path"
	in.cursorPosition = -10 // caret moved left 10 from the end

	in.SetValue("")
	assert.Equal(t, 0, in.cursorPosition, "caret must clamp to the shorter value")
	assert.Equal(t, 0, in.caretIndex())

	// A positive caret (past the end) is clamped to the end.
	in.SetValue("abc")
	in.cursorPosition = 5
	assert.Equal(t, 3, in.caretIndex(), "caretIndex must clamp above len(value)")

	// In-range caret is unchanged.
	in.value = "abcdef"
	in.cursorPosition = -2
	assert.Equal(t, 4, in.caretIndex())
}

// TestCaretIndexNeverPanics exercises the two draw() slice sites for a range of
// (value, cursor) combinations that previously produced out-of-range indices.
func TestCaretIndexNeverPanics(t *testing.T) {
	cases := []struct {
		value  string
		cursor int
	}{
		{"", -10},
		{"", 0},
		{"a", -5},
		{"abc", 3},
		{"hello", -2},
	}
	for _, c := range cases {
		in := &InputWidget{value: c.value, cursorPosition: c.cursor}
		idx := in.caretIndex()
		assert.GreaterOrEqual(t, idx, 0)
		assert.LessOrEqual(t, idx, len(in.value))
		// Both slice expressions used in draw() must be valid.
		_ = in.value[in.caretIndex():]
		_ = in.value[:in.caretIndex()]
	}
}
