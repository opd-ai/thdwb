package extensions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentScriptCSPCreation(t *testing.T) {
	csp := NewContentScriptCSP("")
	require.NotNil(t, csp)
	require.False(t, csp.Enabled())
}

func TestContentScriptCSPWithPolicy(t *testing.T) {
	csp := NewContentScriptCSP("script-src 'self' https://example.com; object-src 'none'")
	require.NotNil(t, csp)
	require.True(t, csp.Enabled())
}

func TestContentScriptCSPCheckScriptAllowed(t *testing.T) {
	csp := NewContentScriptCSP("script-src 'self' https://example.com")

	// 'self' is allowed (non-data/blob URL).
	err := csp.CheckScript("https://example.com/script.js")
	require.NoError(t, err)

	err = csp.CheckScript("https://example.com/app.js")
	require.NoError(t, err)
}

func TestContentScriptCSPCheckScriptBlocked(t *testing.T) {
	csp := NewContentScriptCSP("script-src https://example.com")

	// Different host is blocked.
	err := csp.CheckScript("https://evil.com/script.js")
	require.Error(t, err)
	require.Contains(t, err.Error(), "blocked")
}

func TestContentScriptCSPCheckScriptNone(t *testing.T) {
	csp := NewContentScriptCSP("script-src 'none'")

	err := csp.CheckScript("https://example.com/script.js")
	require.Error(t, err)
	require.Contains(t, err.Error(), "blocked")
}

func TestContentScriptCSPCheckScriptDefaultSrcFallback(t *testing.T) {
	csp := NewContentScriptCSP("default-src https://example.com")

	// No script-src, falls back to default-src.
	err := csp.CheckScript("https://example.com/script.js")
	require.NoError(t, err)

	err = csp.CheckScript("https://other.com/script.js")
	require.Error(t, err)
}

func TestContentScriptCSPCheckConnection(t *testing.T) {
	csp := NewContentScriptCSP("connect-src https://api.example.com")

	err := csp.CheckConnection("https://api.example.com/data")
	require.NoError(t, err)

	err = csp.CheckConnection("https://evil.com/data")
	require.Error(t, err)
}

func TestContentScriptCSPCheckObjectNone(t *testing.T) {
	csp := NewContentScriptCSP("object-src 'none'")

	err := csp.CheckObject("https://example.com/object.swf")
	require.Error(t, err)
	require.Contains(t, err.Error(), "blocked")
}

func TestContentScriptCSPNoPolicy(t *testing.T) {
	csp := NewContentScriptCSP("")

	require.False(t, csp.Enabled())

	// All checks should pass with no policy.
	require.NoError(t, csp.CheckScript("https://anything.com/script.js"))
	require.NoError(t, csp.CheckConnection("https://anything.com/data"))
	require.NoError(t, csp.CheckObject("https://anything.com/object.swf"))
}

func TestContentScriptCSPPolicy(t *testing.T) {
	csp := NewContentScriptCSP("script-src 'self'")
	policy := csp.Policy()
	require.NotNil(t, policy)
	require.Contains(t, policy.Directives, "script-src")
	require.Equal(t, []string{"'self'"}, policy.Directives["script-src"])
}

func TestContentScriptCSPWildcard(t *testing.T) {
	csp := NewContentScriptCSP("script-src *")

	// Wildcard allows everything.
	require.NoError(t, csp.CheckScript("https://evil.com/script.js"))
	require.NoError(t, csp.CheckScript("https://anything.com/app.js"))
}
