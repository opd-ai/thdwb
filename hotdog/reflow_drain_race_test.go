package hotdog

import (
	"sync"
	"testing"

	"github.com/danfragoso/thdwb/assets"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/require"
)

// TestDrainReflowQueueActiveRace (audit C-15) races SetActive (which writes
// wc.active under animationFrameMu) against DrainReflowQueue, which used to
// read wc.active while holding only reflowMu -- the wrong mutex (probe P24).
func TestDrainReflowQueueActiveRace(t *testing.T) {
	wc := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	require.NoError(t, wc.LoadHTML("<html><body><div id='x'></div></body></html>", "https://reflow.example"))

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				_ = wc.DrainReflowQueue()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 500; j++ {
			wc.SetActive(j%2 == 0)
		}
	}()
	wg.Wait()
	wc.SetActive(true)
}
