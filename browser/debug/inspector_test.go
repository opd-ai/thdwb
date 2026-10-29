package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectorRegisterAndGet(t *testing.T) {
	i := NewInspector()

	info := &ExtensionInfo{
		ID:          "ext1",
		Name:        "Test Ext",
		Version:     "1.0",
		Enabled:     true,
		Permissions: []string{"tabs", "storage"},
		MemoryMB:    12.5,
	}
	i.RegisterExtension(info)

	got := i.GetExtension("ext1")
	require.NotNil(t, got)
	assert.Same(t, info, got)
	assert.Equal(t, "Test Ext", got.Name)
	assert.Equal(t, true, got.Enabled)
	assert.Equal(t, []string{"tabs", "storage"}, got.Permissions)
	assert.Equal(t, 12.5, got.MemoryMB)

	assert.Nil(t, i.GetExtension("missing"))
}

func TestInspectorUnregister(t *testing.T) {
	i := NewInspector()
	i.RegisterExtension(&ExtensionInfo{ID: "ext1", Name: "One"})
	i.RegisterExtension(&ExtensionInfo{ID: "ext2", Name: "Two"})

	i.UnregisterExtension("ext1")
	assert.Nil(t, i.GetExtension("ext1"))
	require.NotNil(t, i.GetExtension("ext2"))
	assert.Equal(t, 1, i.Count())
}

func TestInspectorAllExtensions(t *testing.T) {
	i := NewInspector()
	i.RegisterExtension(&ExtensionInfo{ID: "ext1", Name: "One", Enabled: true})
	i.RegisterExtension(&ExtensionInfo{ID: "ext2", Name: "Two", Enabled: false})
	i.RegisterExtension(&ExtensionInfo{ID: "ext3", Name: "Three", Enabled: true})

	all := i.AllExtensions()
	require.Len(t, all, 3)
	allNames := map[string]bool{}
	for _, e := range all {
		allNames[e.Name] = true
	}
	assert.True(t, allNames["One"])
	assert.True(t, allNames["Two"])
	assert.True(t, allNames["Three"])

	enabled := i.EnabledExtensions()
	require.Len(t, enabled, 2)
	enabledNames := map[string]bool{}
	for _, e := range enabled {
		enabledNames[e.Name] = true
	}
	assert.True(t, enabledNames["One"])
	assert.True(t, enabledNames["Three"])
	assert.False(t, enabledNames["Two"])
}

func TestInspectorCount(t *testing.T) {
	i := NewInspector()
	assert.Equal(t, 0, i.Count())

	i.RegisterExtension(&ExtensionInfo{ID: "ext1", Name: "One"})
	assert.Equal(t, 1, i.Count())

	i.RegisterExtension(&ExtensionInfo{ID: "ext2", Name: "Two"})
	assert.Equal(t, 2, i.Count())

	i.UnregisterExtension("ext1")
	assert.Equal(t, 1, i.Count())
}
