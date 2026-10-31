package multiprocess

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtensionProcessInitialState(t *testing.T) {
	p := NewExtensionProcess("ext1")
	require.Equal(t, "ext1", p.ID())
	require.Equal(t, ProcessStopped, p.State())
	require.False(t, p.IsRunning())
	require.Equal(t, 0, p.RestartCount())
}

func TestProcessManagerStartStop(t *testing.T) {
	pm := NewProcessManager()
	require.Equal(t, 0, pm.Count())
	require.Equal(t, 0, pm.RunningCount())
	require.False(t, pm.IsRunning("ext1"))

	// GetOrCreate returns the same process for the same ID.
	p1 := pm.GetOrCreate("ext1")
	p2 := pm.GetOrCreate("ext1")
	require.Same(t, p1, p2)

	require.Equal(t, 1, pm.Count())
}
