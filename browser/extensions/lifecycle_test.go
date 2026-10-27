package extensions

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLifecycleEnableDisable(t *testing.T) {
	lm := NewLifecycleManager()
	m := &Manifest{ManifestVersion: ManifestV3, Name: "T", Version: "1.0"}
	ext := NewExtension("e1", "/p", m)

	require.False(t, lm.IsEnabled("e1"))
	require.Equal(t, LifecycleDisabled, lm.State("e1"))

	require.NoError(t, lm.Enable(ext))
	require.True(t, lm.IsEnabled("e1"))
	require.Equal(t, LifecycleEnabled, lm.State("e1"))
	require.True(t, ext.IsEnabled())

	// Enable again should be a no-op.
	require.NoError(t, lm.Enable(ext))
	require.True(t, lm.IsEnabled("e1"))

	require.NoError(t, lm.Disable(ext))
	require.False(t, lm.IsEnabled("e1"))
	require.Equal(t, LifecycleDisabled, lm.State("e1"))
	require.False(t, ext.IsEnabled())
}

func TestLifecycleStartHookError(t *testing.T) {
	lm := NewLifecycleManager()
	startErr := errors.New("start failed")
	lm.SetHooks(func(ext *Extension) error { return startErr }, nil, nil)

	m := &Manifest{ManifestVersion: ManifestV3, Name: "T", Version: "1.0"}
	ext := NewExtension("e1", "/p", m)

	err := lm.Enable(ext)
	require.Error(t, err)
	require.Contains(t, err.Error(), "start failed")
	require.False(t, lm.IsEnabled("e1"))
}

func TestLifecycleStopHookError(t *testing.T) {
	lm := NewLifecycleManager()
	stopErr := errors.New("stop failed")
	lm.SetHooks(
		func(ext *Extension) error { return nil },
		func(ext *Extension) error { return stopErr },
		nil,
	)

	m := &Manifest{ManifestVersion: ManifestV3, Name: "T", Version: "1.0"}
	ext := NewExtension("e1", "/p", m)

	require.NoError(t, lm.Enable(ext))
	require.True(t, lm.IsEnabled("e1"))

	err := lm.Disable(ext)
	require.Error(t, err)
	require.Contains(t, err.Error(), "stop failed")
	// State should remain enabled since stop failed.
	require.True(t, lm.IsEnabled("e1"))
}

func TestLifecycleReload(t *testing.T) {
	lm := NewLifecycleManager()
	reloadCalls := 0
	lm.SetHooks(nil, nil, func(ext *Extension) error {
		reloadCalls++
		return nil
	})

	m := &Manifest{ManifestVersion: ManifestV3, Name: "T", Version: "1.0"}
	ext := NewExtension("e1", "/p", m)

	require.NoError(t, lm.Enable(ext))
	require.NoError(t, lm.Reload(ext))
	require.Equal(t, 1, reloadCalls)
	require.Equal(t, LifecycleEnabled, lm.State("e1"))
}

func TestLifecycleUninstall(t *testing.T) {
	lm := NewLifecycleManager()
	m := &Manifest{ManifestVersion: ManifestV3, Name: "T", Version: "1.0"}
	ext := NewExtension("e1", "/p", m)

	require.NoError(t, lm.Enable(ext))
	require.NoError(t, lm.Uninstall(ext))
	require.False(t, lm.IsEnabled("e1"))
	require.False(t, ext.IsEnabled())
}

func TestLifecycleEnabledIDs(t *testing.T) {
	lm := NewLifecycleManager()
	m1 := &Manifest{ManifestVersion: ManifestV3, Name: "T1", Version: "1.0"}
	m2 := &Manifest{ManifestVersion: ManifestV3, Name: "T2", Version: "1.0"}
	ext1 := NewExtension("e1", "/p1", m1)
	ext2 := NewExtension("e2", "/p2", m2)

	require.NoError(t, lm.Enable(ext1))
	require.NoError(t, lm.Enable(ext2))

	ids := lm.EnabledIDs()
	require.Contains(t, ids, "e1")
	require.Contains(t, ids, "e2")
	require.Len(t, ids, 2)

	require.NoError(t, lm.Disable(ext1))
	ids = lm.EnabledIDs()
	require.Contains(t, ids, "e2")
	require.NotContains(t, ids, "e1")
	require.Len(t, ids, 1)
}
