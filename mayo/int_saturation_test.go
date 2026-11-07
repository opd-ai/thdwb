package mayo

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMapIntValueSaturates (audit O-15) checks that an out-of-range integer
// saturates instead of collapsing to 0. z-index: 99999999999999 is meant to be
// "topmost"; strconv.Atoi returned an error and the old code mapped it to 0,
// putting the element below everything.
func TestMapIntValueSaturates(t *testing.T) {
	// Beyond int64: strconv.Atoi (the old code) returned a range error and the
	// helper collapsed it to 0, putting a "topmost" z-index BELOW everything.
	assert.Equal(t, math.MaxInt32, mapIntValue("99999999999999999999999"))
	assert.Equal(t, math.MinInt32, mapIntValue("-99999999999999999999999"))
	// Within int64 but above int32: now clamped to the CSS integer domain.
	assert.Equal(t, math.MaxInt32, mapIntValue("99999999999999"))
	assert.Equal(t, math.MaxInt32, mapIntValue("z-index: 99999999999999"))
	assert.Equal(t, math.MinInt32, mapIntValue("-99999999999999"))

	// Boundary values still parse exactly.
	assert.Equal(t, math.MaxInt32, mapIntValue("2147483647"))
	assert.Equal(t, math.MinInt32, mapIntValue("-2147483648"))
	assert.Equal(t, 5, mapIntValue("5"))

	// Non-numeric input still yields 0.
	assert.Equal(t, 0, mapIntValue("auto"))
	assert.Equal(t, 0, mapIntValue(""))
}
