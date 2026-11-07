package hotdog

import (
	"sync"
	"testing"

	"github.com/danfragoso/thdwb/assets"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInputStateConcurrentWrites (audit C-9) hammers SetInputState from several
// goroutines. inputState was a plain map, so concurrent writes triggered the
// unrecoverable "fatal error: concurrent map writes" runtime kill (probe P9).
// Under -race this must pass cleanly.
func TestInputStateConcurrentWrites(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	require.NoError(t, wc.LoadHTML("<html><body></body></html>", "https://input.example"))

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				wc.SetInputState("key", id*1000+j)
				_, _ = wc.GetInputState("key")
			}
		}(i)
	}
	wg.Wait()

	v, ok := wc.GetInputState("key")
	assert.True(t, ok)
	assert.NotNil(t, v)
}

// TestFocusedElementConcurrentAccess races the render goroutine's
// paintFocusRing read (via GetFocusedElement, bun/bun.go:138) against
// SetFocusedElement from input handling (probe P4).
func TestFocusedElementConcurrentAccess(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	require.NoError(t, wc.LoadHTML("<html><body><input id='a'><input id='b'></body></html>", "https://input.example"))

	a := &NodeDOM{Type: NodeTypeElement, Element: "input", WindowCtx: wc}
	b := &NodeDOM{Type: NodeTypeElement, Element: "input", WindowCtx: wc}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() { // render goroutine
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = wc.GetFocusedElement()
			_ = wc.focusedDOMNode()
		}
	}()
	wg.Add(1)
	go func() { // input goroutine
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			wc.SetFocusedElement(a)
			wc.SetFocusedElement(b)
			wc.SetFocusedElement(nil)
		}
		close(stop)
	}()
	wg.Wait()
}

// TestInputStateAfterDestroy makes sure the accessors do not panic on the map
// that Destroy nils out.
func TestInputStateAfterDestroy(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	require.NoError(t, wc.LoadHTML("<html><body></body></html>", "https://input.example"))
	wc.SetInputState("k", 1)
	wc.Destroy()

	assert.NotPanics(t, func() {
		wc.SetInputState("k", 2)
		v, ok := wc.GetInputState("k")
		assert.Nil(t, v)
		assert.False(t, ok)
	})
}
