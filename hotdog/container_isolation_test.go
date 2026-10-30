package hotdog

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	sauce "github.com/danfragoso/thdwb/sauce"
)

// TestContainerLocalStorageIsolation verifies that localStorage is partitioned
// by container ID + origin, so data written in one container is not visible
// in another container for the same origin.
func TestContainerLocalStorageIsolation(t *testing.T) {
	manager, err := NewOriginStorageManager("")
	require.NoError(t, err)

	origin, err := url.Parse("https://example.com")
	require.NoError(t, err)

	// Write to container "c1"
	require.NoError(t, manager.SetItem("c1", origin, "theme", "dark"))
	// Write to container "c2"
	require.NoError(t, manager.SetItem("c2", origin, "theme", "light"))

	// Verify isolation
	value, ok := manager.GetItem("c1", origin, "theme")
	require.True(t, ok)
	require.Equal(t, "dark", value)

	value, ok = manager.GetItem("c2", origin, "theme")
	require.True(t, ok)
	require.Equal(t, "light", value)

	// Default container should be isolated too
	_, ok = manager.GetItem(DefaultContainerID, origin, "theme")
	require.False(t, ok, "default container should not see c1/c2 data")

	// Lengths should be per-container
	require.Equal(t, 1, manager.length("c1", origin))
	require.Equal(t, 1, manager.length("c2", origin))
	require.Equal(t, 0, manager.length(DefaultContainerID, origin))
}

// TestContainerSessionStorageIsolation verifies that sessionStorage is
// partitioned by container ID + origin.
func TestContainerSessionStorageIsolation(t *testing.T) {
	manager := NewSessionStorageManager()

	origin, err := url.Parse("https://example.com")
	require.NoError(t, err)

	// Write to container "c1"
	manager.SetItem("c1", origin, "sessionKey", "c1-value")
	// Write to container "c2"
	manager.SetItem("c2", origin, "sessionKey", "c2-value")

	// Verify isolation
	value, ok := manager.GetItem("c1", origin, "sessionKey")
	require.True(t, ok)
	require.Equal(t, "c1-value", value)

	value, ok = manager.GetItem("c2", origin, "sessionKey")
	require.True(t, ok)
	require.Equal(t, "c2-value", value)

	// Default container should be isolated
	_, ok = manager.GetItem(DefaultContainerID, origin, "sessionKey")
	require.False(t, ok)

	// Lengths should be per-container
	require.Equal(t, 1, manager.length("c1", origin))
	require.Equal(t, 1, manager.length("c2", origin))
	require.Equal(t, 0, manager.length(DefaultContainerID, origin))
}

// TestContainerCookieJarIsolation verifies that cookies are partitioned by
// container ID + origin, so cookies set in one container are not visible
// in another container for the same origin.
func TestContainerCookieJarIsolation(t *testing.T) {
	jar := sauce.NewContainerCookieJar()

	origin, err := url.Parse("https://example.com")
	require.NoError(t, err)

	// Set cookie in container "c1"
	jar.SetCookies("c1", origin, []*http.Cookie{
		{Name: "session", Value: "c1-value", Path: "/"},
	})
	jar.SetCookies("c2", origin, []*http.Cookie{
		{Name: "session", Value: "c2-value", Path: "/"},
	})

	// Verify isolation
	cookies := jar.Cookies("c1", origin)
	require.Len(t, cookies, 1)
	require.Equal(t, "c1-value", cookies[0].Value)

	cookies = jar.Cookies("c2", origin)
	require.Len(t, cookies, 1)
	require.Equal(t, "c2-value", cookies[0].Value)

	// Default container should be isolated
	cookies = jar.Cookies(DefaultContainerID, origin)
	require.Empty(t, cookies)
}

// TestContainerClearIsolation verifies that clearing one container does not
// affect another container's data.
func TestContainerClearIsolation(t *testing.T) {
	manager, err := NewOriginStorageManager("")
	require.NoError(t, err)

	origin, err := url.Parse("https://example.com")
	require.NoError(t, err)

	manager.SetItem("c1", origin, "key", "c1-value")
	manager.SetItem("c2", origin, "key", "c2-value")

	// Clear container c1
	require.NoError(t, manager.ClearOrigin("c1", origin))

	// c1 should be empty
	_, ok := manager.GetItem("c1", origin, "key")
	require.False(t, ok)

	// c2 should still have its value
	value, ok := manager.GetItem("c2", origin, "key")
	require.True(t, ok)
	require.Equal(t, "c2-value", value)
}

// TestContainerClearAllIsolation verifies ClearContainer removes all data
// for a container across all origins.
func TestContainerClearAllIsolation(t *testing.T) {
	manager, err := NewOriginStorageManager("")
	require.NoError(t, err)

	origin1, err := url.Parse("https://example.com")
	require.NoError(t, err)
	origin2, err := url.Parse("https://other.example")
	require.NoError(t, err)

	manager.SetItem("c1", origin1, "key1", "value1")
	manager.SetItem("c1", origin2, "key2", "value2")
	manager.SetItem("c2", origin1, "key1", "value1-c2")

	// Clear all of c1
	require.NoError(t, manager.ClearContainer("c1"))

	// c1 should be empty for both origins
	require.Equal(t, 0, manager.length("c1", origin1))
	require.Equal(t, 0, manager.length("c1", origin2))

	// c2 should be unaffected
	require.Equal(t, 1, manager.length("c2", origin1))
	value, ok := manager.GetItem("c2", origin1, "key1")
	require.True(t, ok)
	require.Equal(t, "value1-c2", value)
}
