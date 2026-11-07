package sauce

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFetchContextCancellation locks in audit R-6: requests built through the
// Ctx variants carry the caller's context, so cancelling it aborts the
// in-flight fetch instead of leaving the goroutine pinned to a stalled server.
func TestFetchContextCancellation(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // stall until the test decides to unblock
		_, _ = w.Write([]byte("late"))
	}))
	defer srv.Close()
	defer close(release)

	u, err := url.Parse(srv.URL + "/stalled")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan *Resource, 1)
	go func() {
		done <- fetchExternalPageWithOptionsCtx(ctx, nil, u, "GET", nil, nil)
	}()

	// Give the request a moment to actually reach the stalled handler.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case res := <-done:
		assert.Equal(t, -1, res.Code, "cancelled fetch must surface a transport error")
		assert.NotEmpty(t, res.Body, "error page should explain the failure")
	case <-time.After(3 * time.Second):
		t.Fatal("fetch did not abort after context cancellation")
	}
}

// TestFetchExternalPageWithOptionsCtxSuccess verifies the ctx path still
// returns normal responses when the context stays alive.
func TestFetchExternalPageWithOptionsCtxSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ctx-ok"))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL + "/ctx-success-1")
	require.NoError(t, err)
	res := FetchExternalPageWithOptionsCtx(context.Background(), u, "GET", nil, nil)
	assert.Equal(t, 200, res.Code)
	assert.Equal(t, "ctx-ok", res.Body)
}

// TestMakeCORSRequestWithClientCtxPassesContext verifies the CORS wrapper
// threads ctx into the underlying request (a pre-cancelled context fails
// before any network I/O).
func TestMakeCORSRequestWithClientCtxPassesContext(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	u, err := url.Parse(srv.URL + "/cors-cancelled")
	require.NoError(t, err)
	res := MakeCORSRequestWithClientCtx(ctx, nil, u, "https://initiator.test", "GET", nil, nil)
	assert.Equal(t, -1, res.Code, "pre-cancelled context must not reach the server")
	assert.False(t, hit, "server should never have been contacted")
}

// TestDefaultClientHasTimeout pins the global client deadline (audit R-6):
// the shared client used by GetImage and the deprecated no-client entry
// points must no longer wait forever on a stalled peer.
func TestDefaultClientHasTimeout(t *testing.T) {
	if defaultClient().Timeout != defaultClientTimeout {
		t.Fatalf("global client timeout = %v, want %v", defaultClient().Timeout, defaultClientTimeout)
	}
}
