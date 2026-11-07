package hotdog

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout runs fn and returns everything it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()

	w.Close()
	os.Stdout = old
	out := <-done
	r.Close()
	return out
}

// TestMutationObserverErrorIsLogged (audit C-10) verifies that a throwing
// MutationObserver callback is reported instead of silently discarded. The old
// code called `_, _ = mo.callback(...)`, so both the returned error and any
// panic were dropped on the floor -- a broken observer was 100% invisible.
func TestMutationObserverErrorIsLogged(t *testing.T) {
	wc := newBindingTestContext(t, `<html><body><div id="t"></div></body></html>`)

	out := captureStdout(t, func() {
		_, err := wc.RunJS(`
			var mo = new MutationObserver(function(records) {
				throw new Error("observer blew up");
			});
			mo.observe(document.getElementById("t"));
			document.getElementById("t").setAttribute("data-x", "1");
		`)
		require.NoError(t, err)
	})

	// runJSListener logs the thrown error to the page console; the old
	// `_, _ =` path logged nothing at all.
	assert.Contains(t, out, "console.error",
		"a throwing observer must be reported, not silently swallowed")
	assert.Contains(t, out, "observer blew up")
}

// TestMutationObserverErrorDoesNotBreakOtherObservers proves one failing
// observer does not prevent later mutations or other observers from running.
func TestMutationObserverErrorDoesNotBreakOtherObservers(t *testing.T) {
	wc := newBindingTestContext(t, `<html><body><div id="t"></div></body></html>`)

	_, err := wc.RunJS(`
		var good = 0;
		var bad = new MutationObserver(function() { throw new Error("kaboom"); });
		bad.observe(document.getElementById("t"));
		var ok = new MutationObserver(function() { good = good + 1; });
		ok.observe(document.getElementById("t"));
	`)
	require.NoError(t, err)

	out := captureStdout(t, func() {
		_, err := wc.RunJS(`document.getElementById("t").setAttribute("data-x", "2");`)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "kaboom")

	v, err := wc.RunJS(`good;`)
	require.NoError(t, err)
	assert.EqualValues(t, 1, v.ToInteger(), "healthy observer must still fire")
}
