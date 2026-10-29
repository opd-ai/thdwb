package telemetry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	assert.Equal(t, NotAsked, c.Enabled)
	assert.Equal(t, "https://telemetry.thdwb.dev/submit", c.Endpoint)
	assert.Equal(t, 15*time.Minute, c.Interval)
	require.NotNil(t, c.HTTPClient)
}

func TestTelemetryOptInOut(t *testing.T) {
	t.Run("opt in", func(t *testing.T) {
		tel := &Telemetry{config: DefaultConfig()}
		assert.True(t, tel.IsNotAsked())
		assert.False(t, tel.IsEnabled())

		tel.OptIn()
		assert.True(t, tel.IsEnabled())
		assert.False(t, tel.IsNotAsked())
	})

	t.Run("opt out", func(t *testing.T) {
		tel := &Telemetry{config: DefaultConfig()}
		tel.OptIn()
		tel.OptOut()
		assert.False(t, tel.IsEnabled())
	})
}

func TestTelemetryRecordDisabled(t *testing.T) {
	tel := &Telemetry{config: DefaultConfig()} // NotAsked by default
	tel.Record("test_metric", 1.0, nil)
	assert.Equal(t, 0, tel.PendingCount())
}

func TestTelemetryRecordEnabled(t *testing.T) {
	tel := &Telemetry{config: DefaultConfig()}
	tel.OptIn()
	tel.Record("page_load", 1.5, map[string]string{"url": "example.com"})
	assert.Equal(t, 1, tel.PendingCount())
}

func TestTelemetryRecordPageLoad(t *testing.T) {
	tel := &Telemetry{config: DefaultConfig()}
	tel.OptIn()
	tel.RecordPageLoad(2*time.Second, map[string]string{"url": "example.com"})
	require.Equal(t, 1, tel.PendingCount())
}

func TestTelemetryRecordExtensionStart(t *testing.T) {
	tel := &Telemetry{config: DefaultConfig()}
	tel.OptIn()
	tel.RecordExtensionStart("ext-1", 500*time.Millisecond)
	require.Equal(t, 1, tel.PendingCount())
}

func TestTelemetryRecordSessionLength(t *testing.T) {
	tel := &Telemetry{config: DefaultConfig()}
	tel.OptIn()
	tel.RecordSessionLength(30 * time.Minute)
	require.Equal(t, 1, tel.PendingCount())
}

func TestStoreSaveLoadEnabled(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	require.NoError(t, store.SaveEnabled(true))
	// Verify file was written.
	data, err := os.ReadFile(filepath.Join(dir, "enabled"))
	require.NoError(t, err)
	assert.Equal(t, "enabled", string(data))

	require.NoError(t, store.SaveEnabled(false))
	data, err = os.ReadFile(filepath.Join(dir, "enabled"))
	require.NoError(t, err)
	assert.Equal(t, "disabled", string(data))
}

func TestStoreSaveLoadMetrics(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	metrics := []Metric{
		{Name: "page_load", Value: 1.5, Timestamp: time.Now()},
		{Name: "session_length", Value: 1800.0},
	}
	require.NoError(t, store.SaveMetrics(metrics))

	loaded, err := store.Load()
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "page_load", loaded[0].Name)
	assert.Equal(t, 1.5, loaded[0].Value)
	assert.Equal(t, "session_length", loaded[1].Name)
	assert.Equal(t, 1800.0, loaded[1].Value)
}

func TestStoreLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestNewTelemetryWithStore(t *testing.T) {
	dir := t.TempDir()
	tel, err := New(DefaultConfig(), dir)
	require.NoError(t, err)
	require.NotNil(t, tel)
	assert.NotNil(t, tel.store)
}

func TestSubmitDisabled(t *testing.T) {
	tel := &Telemetry{config: DefaultConfig()} // NotAsked
	err := tel.Submit()
	assert.NoError(t, err)
	assert.Equal(t, 0, tel.PendingCount())
}
