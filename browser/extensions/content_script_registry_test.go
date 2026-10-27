package extensions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentScriptRegistry(t *testing.T) {
	reg := NewContentScriptRegistry()

	require.Equal(t, 0, reg.Count())

	reg.Register(ContentScript{
		ID:      "cs1",
		Matches: []string{"*://example.com/*"},
		JS:      []string{"content.js"},
	})
	reg.Register(ContentScript{
		ID:      "cs2",
		Matches: []string{"https://example.org/*"},
		JS:      []string{"other.js"},
	})

	require.Equal(t, 2, reg.Count())

	// Match example.com.
	matches := reg.Matching("https://example.com/page")
	require.Len(t, matches, 1)
	require.Equal(t, "cs1", matches[0].ID)

	// Match example.org.
	matches = reg.Matching("https://example.org/page")
	require.Len(t, matches, 1)
	require.Equal(t, "cs2", matches[0].ID)

	// No match.
	matches = reg.Matching("https://other.com/page")
	require.Empty(t, matches)

	// All returns all scripts.
	all := reg.All()
	require.Len(t, all, 2)
}

func TestContentScriptRegistryExcludeMatches(t *testing.T) {
	reg := NewContentScriptRegistry()

	reg.Register(ContentScript{
		ID:            "cs-exclude",
		Matches:       []string{"*://example.com/*"},
		ExcludeMatches: []string{"*://example.com/admin/*"},
		JS:            []string{"content.js"},
	})

	// Should match normal page.
	matches := reg.Matching("https://example.com/page")
	require.Len(t, matches, 1)

	// Should NOT match excluded path.
	matches = reg.Matching("https://example.com/admin/settings")
	require.Empty(t, matches)
}

func TestContentScriptRegistryAllReturnsCopy(t *testing.T) {
	reg := NewContentScriptRegistry()
	reg.Register(ContentScript{ID: "cs1", Matches: []string{"*"}, JS: []string{"a.js"}})

	all := reg.All()
	all[0].ID = "modified"

	// Original should be unchanged.
	original := reg.All()
	require.Equal(t, "cs1", original[0].ID)
}
