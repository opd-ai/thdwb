package bun

import (
	"sync"
	"testing"
	"time"

	"github.com/danfragoso/thdwb/assets"
	gg "github.com/danfragoso/thdwb/gg"
	"github.com/danfragoso/thdwb/hotdog"
	"github.com/danfragoso/thdwb/ketchup"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/require"
)

// TestRenderDocumentSerializedAgainstJSMutation (audit R-7) reproduces the
// render-vs-script race the audit confirmed with -race: RenderDocument walks
// Children/RenderBox/Style and assigns RenderBox, while page JS mutates the
// same tree from another goroutine. Both sides must hold the window's
// jsRuntimeMu or the detector fires. With the shell's WithJSRLock wrapper the
// walk is serialized against RunJS and stays clean.
func TestRenderDocumentSerializedAgainstJSMutation(t *testing.T) {
	const page = `<!DOCTYPE html><html><head>
	<style>div { width: 10px; height: 10px; background-color: red; }</style>
	</head><body><div id="root"></div></body></html>`

	wc := hotdog.NewWindowContext(&hotdog.Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer wc.Destroy()
	doc := ketchup.ParseHTMLDocument(page, wc)
	require.NotNil(t, doc)
	wc.ActiveDocument = doc
	require.NoError(t, wc.InitJSRuntime())

	ctx := gg.NewContext(800, 600)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Page-JS mutators: RunJS takes jsRuntimeMu, like rAF/event callbacks do.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_, _ = wc.RunJS(`
					var r = document.getElementById("root");
					if (r) { r.appendChild(document.createElement("span")); }
				`)
			}
		}()
	}

	// Renders must take the same lock the mutators hold.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 30; i++ {
			done := make(chan struct{})
			go func() { // RunJS is reentrant-locked; render on its own goroutine
				wc.WithJSRLock(func() {
					_ = RenderDocument(ctx, doc, false)
				})
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("render under WithJSRLock deadlocked")
				close(stop)
				return
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
}
