package testing

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/danfragoso/thdwb/browser/extensions"
	"github.com/stretchr/testify/require"
)

func TestHarnessCreateExtensionV3(t *testing.T) {
	h := NewHarness(t)
	defer h.Cleanup()

	ext := h.CreateExtensionV3("ext1", "Test Ext", "1.0", "tabs", "storage")
	require.NotNil(t, ext)
	require.Equal(t, "ext1", ext.ID)
	require.Equal(t, "Test Ext", ext.Name)
	require.Equal(t, "1.0", ext.Version)
	require.Contains(t, ext.Permissions, "tabs")
	require.Contains(t, ext.Permissions, "storage")

	h.AssertExtensionExists("ext1")
	h.AssertExtensionCount(1)
}

func TestHarnessEnableDisable(t *testing.T) {
	h := NewHarness(t)
	defer h.Cleanup()

	h.CreateExtensionV3("ext1", "Test Ext", "1.0")
	h.EnableExtension("ext1")
	h.AssertEnabled("ext1")

	h.DisableExtension("ext1")
	h.AssertDisabled("ext1")
}

func TestHarnessMultipleExtensions(t *testing.T) {
	h := NewHarness(t)
	defer h.Cleanup()

	h.CreateExtensionV3("ext1", "Ext One", "1.0")
	h.CreateExtensionV3("ext2", "Ext Two", "2.0")
	h.AssertExtensionCount(2)

	h.AssertExtensionExists("ext1")
	h.AssertExtensionExists("ext2")
	h.AssertExtensionNotExists("ext3")
}

func TestHarnessWaitFor(t *testing.T) {
	h := NewHarness(t)
	defer h.Cleanup()

	var called atomic.Bool
	go func() {
		time.Sleep(20 * time.Millisecond)
		called.Store(true)
	}()

	h.WaitFor(func() bool { return called.Load() }, 2*time.Second)
	require.True(t, called.Load())
}

func TestHarnessWaitForTimeout(t *testing.T) {
	h := NewHarness(t)
	defer h.Cleanup()

	// This should timeout since the condition never becomes true.
	err := h.WaitFor(func() bool { return false }, 50*time.Millisecond)
	require.Error(t, err)
}

func TestManagerAddExtension(t *testing.T) {
	m := extensions.NewManager("")

	manifest := &extensions.Manifest{
		ManifestVersion: extensions.ManifestV3,
		Name:            "Test",
		Version:         "1.0",
	}
	ext := extensions.NewExtension("ext1", "/tmp", manifest)

	require.NoError(t, m.AddExtension(ext))
	require.NotNil(t, m.Get("ext1"))
	require.Equal(t, 1, m.Count())

	// Adding duplicate should error.
	require.Error(t, m.AddExtension(ext))
}
