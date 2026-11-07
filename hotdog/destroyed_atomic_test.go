package hotdog

import (
	"sync"
	"testing"

	"github.com/danfragoso/thdwb/assets"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDestroyedFlagIsAtomic (audit C-17) races Destroy (which sets `destroyed`
// under animationFrameMu) against LoadHTML/LoadURL, which read the flag
// without holding any lock. The plain bool made that a data race; the field is
// now an atomic.Bool so a concurrent navigation check is well-defined.
func TestDestroyedFlagIsAtomic(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	require.NoError(t, wc.LoadHTML("<html><body></body></html>", "https://destroy.example"))

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 2000; j++ {
				// ":/bad" fails url.Parse immediately after the destroyed
				// check, so this goroutine touches ONLY the destroyed flag --
				// the exact unlocked read C-17 flagged.
				_ = wc.LoadURL(":/bad")
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		wc.Destroy()
	}()
	wg.Wait()
}

// TestLoadHTMLErrorsAfterDestroy keeps the behavioral contract: navigation on
// a destroyed window must be refused, not crash.
func TestLoadHTMLErrorsAfterDestroy(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	require.NoError(t, wc.LoadHTML("<html><body></body></html>", "https://destroy.example"))
	wc.Destroy()

	err := wc.LoadHTML("<html><body>second</body></html>", "https://destroy.example")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destroyed")

	err = wc.LoadURL("https://destroy.example/")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destroyed")

	assert.Equal(t, uint64(0), wc.ScheduleAnimationFrame(func() {}),
		"destroyed windows must not queue animation frames")
}
