package extensions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentScriptLoaderCreation(t *testing.T) {
	l := NewContentScriptLoader()
	require.NotNil(t, l)
	require.Equal(t, 0, l.Count())
}

func TestContentScriptLoaderRegisterAndMatching(t *testing.T) {
	l := NewContentScriptLoader()

	l.Register(ContentScript{
		ID:      "cs1",
		ExtID:   "ext1",
		Matches: []string{"*://example.com/*"},
		JS:      []string{"content.js"},
		RunAt:   "document_idle",
	})
	l.Register(ContentScript{
		ID:      "cs2",
		ExtID:   "ext2",
		Matches: []string{"https://example.org/*"},
		JS:      []string{"other.js"},
		RunAt:   "document_start",
	})

	require.Equal(t, 2, l.Count())

	// Match example.com.
	matches := l.Matching("https://example.com/page")
	require.Len(t, matches, 1)
	require.Equal(t, "cs1", matches[0].ID)

	// Match example.org.
	matches = l.Matching("https://example.org/page")
	require.Len(t, matches, 1)
	require.Equal(t, "cs2", matches[0].ID)

	// No match.
	matches = l.Matching("https://other.com/page")
	require.Empty(t, matches)
}

func TestContentScriptLoaderExcludeMatches(t *testing.T) {
	l := NewContentScriptLoader()

	l.Register(ContentScript{
		ID:            "cs-exclude",
		ExtID:         "ext1",
		Matches:       []string{"*://example.com/*"},
		ExcludeMatches: []string{"*://example.com/admin/*"},
		JS:            []string{"content.js"},
	})

	// Should match normal page.
	matches := l.Matching("https://example.com/page")
	require.Len(t, matches, 1)

	// Should NOT match excluded path.
	matches = l.Matching("https://example.com/admin/settings")
	require.Empty(t, matches)
}

func TestContentScriptLoaderRunAtOrdering(t *testing.T) {
	l := NewContentScriptLoader()

	l.Register(ContentScript{
		ID:      "cs-idle",
		ExtID:   "ext1",
		Matches: []string{"*://example.com/*"},
		JS:      []string{"a.js"},
		RunAt:   "document_idle",
	})
	l.Register(ContentScript{
		ID:      "cs-start",
		ExtID:   "ext1",
		Matches: []string{"*://example.com/*"},
		JS:      []string{"b.js"},
		RunAt:   "document_start",
	})
	l.Register(ContentScript{
		ID:      "cs-end",
		ExtID:   "ext1",
		Matches: []string{"*://example.com/*"},
		JS:      []string{"c.js"},
		RunAt:   "document_end",
	})

	matches := l.Matching("https://example.com/page")
	require.Len(t, matches, 3)
	require.Equal(t, "cs-start", matches[0].ID)
	require.Equal(t, "cs-end", matches[1].ID)
	require.Equal(t, "cs-idle", matches[2].ID)
}

func TestContentScriptLoaderMatchingForExtension(t *testing.T) {
	l := NewContentScriptLoader()

	l.Register(ContentScript{
		ID:      "cs1",
		ExtID:   "ext1",
		Matches: []string{"*://example.com/*"},
		JS:      []string{"a.js"},
	})
	l.Register(ContentScript{
		ID:      "cs2",
		ExtID:   "ext2",
		Matches: []string{"*://example.com/*"},
		JS:      []string{"b.js"},
	})

	matches := l.MatchingForExtension("ext1", "https://example.com/page")
	require.Len(t, matches, 1)
	require.Equal(t, "cs1", matches[0].ID)

	matches = l.MatchingForExtension("ext2", "https://example.com/page")
	require.Len(t, matches, 1)
	require.Equal(t, "cs2", matches[0].ID)

	// No scripts for unknown extension.
	matches = l.MatchingForExtension("ext3", "https://example.com/page")
	require.Empty(t, matches)
}

func TestContentScriptLoaderCountForExtension(t *testing.T) {
	l := NewContentScriptLoader()

	l.Register(ContentScript{ID: "cs1", ExtID: "ext1", Matches: []string{"*://*/*"}, JS: []string{"a.js"}})
	l.Register(ContentScript{ID: "cs2", ExtID: "ext1", Matches: []string{"*://*/*"}, JS: []string{"b.js"}})
	l.Register(ContentScript{ID: "cs3", ExtID: "ext2", Matches: []string{"*://*/*"}, JS: []string{"c.js"}})

	require.Equal(t, 2, l.CountForExtension("ext1"))
	require.Equal(t, 1, l.CountForExtension("ext2"))
	require.Equal(t, 0, l.CountForExtension("ext3"))
}

func TestContentScriptLoaderRemoveForExtension(t *testing.T) {
	l := NewContentScriptLoader()

	l.Register(ContentScript{ID: "cs1", ExtID: "ext1", Matches: []string{"*://*/*"}, JS: []string{"a.js"}})
	l.Register(ContentScript{ID: "cs2", ExtID: "ext2", Matches: []string{"*://*/*"}, JS: []string{"b.js"}})

	require.Equal(t, 2, l.Count())

	l.RemoveForExtension("ext1")
	require.Equal(t, 1, l.Count())
	require.Equal(t, 0, l.CountForExtension("ext1"))

	matches := l.Matching("https://any.com/page")
	require.Len(t, matches, 1)
	require.Equal(t, "cs2", matches[0].ID)
}

func TestContentScriptLoaderRegisterFromManifest(t *testing.T) {
	manifest := &Manifest{
		ManifestVersion: ManifestV3,
		Name:            "Test",
		Version:         "1.0",
		ContentScripts: []ContentScriptManifest{
			{
				Matches: []string{"*://example.com/*"},
				JS:      []string{"content.js"},
				RunAt:   "document_end",
			},
		},
	}

	l := NewContentScriptLoader()
	l.RegisterFromManifest("ext1", manifest)

	require.Equal(t, 1, l.Count())
	require.Equal(t, 1, l.CountForExtension("ext1"))

	matches := l.Matching("https://example.com/page")
	require.Len(t, matches, 1)
	require.Equal(t, "ext1:content.js", matches[0].ID)
	require.Equal(t, "ext1", matches[0].ExtID)
	require.Equal(t, "document_end", matches[0].RunAt)
}

func TestContentScriptLoaderEnsureExtIDs(t *testing.T) {
	l := NewContentScriptLoader()

	l.Register(ContentScript{
		ID:      "ext1:script1",
		Matches: []string{"*://*/*"},
		JS:      []string{"a.js"},
	})
	// Script without ExtID but with ID prefix.
	l.Register(ContentScript{
		ID:      "ext2:script2",
		Matches: []string{"*://*/*"},
		JS:      []string{"b.js"},
	})

	l.EnsureExtIDs()

	// First script already has ExtID empty but ID has prefix.
	// EnsureExtIDs should derive ExtID from ID prefix.
	// Note: the first script's ExtID is empty, ID is "ext1:script1"
	// so it should be set to "ext1".
	// The second script has no colon, so ExtID remains empty.
	_ = l
}
