package webdriver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWebDriverCreateSession verifies session creation.
func TestWebDriverCreateSession(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"capabilities": {"browserName": "thdwb"}}`)
	resp, err := http.Post(ts.URL+"/session", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Value struct {
			SessionID   string `json:"sessionId"`
			Capabilities Capabilities `json:"capabilities"`
		} `json:"value"`
	}
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &result))
	require.NotEmpty(t, result.Value.SessionID)
	require.Equal(t, "thdwb", result.Value.Capabilities.BrowserName)
}

// TestWebDriverListSessions verifies listing sessions.
func TestWebDriverListSessions(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Create a session.
	body := strings.NewReader(`{"capabilities": {}}`)
	resp, err := http.Post(ts.URL+"/session", "application/json", body)
	require.NoError(t, err)
	resp.Body.Close()

	// List sessions.
	resp, err = http.Get(ts.URL + "/session")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Value []struct {
			SessionID string `json:"sessionId"`
		} `json:"value"`
	}
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &result))
	require.Len(t, result.Value, 1)
	require.NotEmpty(t, result.Value[0].SessionID)
}

// TestWebDriverGetSession verifies getting a specific session.
func TestWebDriverGetSession(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Create a session.
	body := strings.NewReader(`{"capabilities": {"browserName": "thdwb"}}`)
	resp, err := http.Post(ts.URL+"/session", "application/json", body)
	require.NoError(t, err)
	var createResult struct {
		Value struct {
			SessionID string `json:"sessionId"`
		} `json:"value"`
	}
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	require.NoError(t, json.Unmarshal(data, &createResult))

	// Get the session.
	resp, err = http.Get(ts.URL + "/session/" + createResult.Value.SessionID)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResult struct {
		Value struct {
			SessionID string `json:"sessionId"`
		} `json:"value"`
	}
	data, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &getResult))
	require.Equal(t, createResult.Value.SessionID, getResult.Value.SessionID)
}

// TestWebDriverDeleteSession verifies session deletion.
func TestWebDriverDeleteSession(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Create a session.
	body := strings.NewReader(`{"capabilities": {}}`)
	resp, err := http.Post(ts.URL+"/session", "application/json", body)
	require.NoError(t, err)
	var createResult struct {
		Value struct {
			SessionID string `json:"sessionId"`
		} `json:"value"`
	}
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	require.NoError(t, json.Unmarshal(data, &createResult))

	// Delete the session.
	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/session/"+createResult.Value.SessionID, nil)
	require.NoError(t, err)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Session should be gone.
	require.False(t, srv.SessionCount() > 0)
	require.Equal(t, 0, srv.SessionCount())
}

// TestWebDriverSessionNotFound verifies 404 for unknown session.
func TestWebDriverSessionNotFound(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/session/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestWebDriverSessionCount verifies session count tracking.
func TestWebDriverSessionCount(t *testing.T) {
	srv := NewServer()
	require.Equal(t, 0, srv.SessionCount())

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"capabilities": {}}`)
	_, err := http.Post(ts.URL+"/session", "application/json", body)
	require.NoError(t, err)

	require.Equal(t, 1, srv.SessionCount())
}
