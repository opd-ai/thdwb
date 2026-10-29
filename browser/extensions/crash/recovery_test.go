package crash

import (
	"errors"
	"testing"

	goja "github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryManagerRecordCrash(t *testing.T) {
	rm := NewRecoveryManager(10)

	event := rm.RecordCrash("ext1", "panic", "nil pointer dereference")
	require.NotNil(t, event)
	assert.Equal(t, "ext1", event.ExtensionID)
	assert.Equal(t, "panic", event.Reason)
	assert.Equal(t, "nil pointer dereference", event.Details)
	assert.False(t, event.Recovered)
	assert.False(t, event.Timestamp.IsZero())
}

func TestRecoveryManagerMaxEvents(t *testing.T) {
	rm := NewRecoveryManager(3)

	rm.RecordCrash("ext1", "panic", "one")
	rm.RecordCrash("ext1", "panic", "two")
	rm.RecordCrash("ext1", "panic", "three")
	rm.RecordCrash("ext1", "panic", "four") // evicts "one"

	events := rm.Events()
	require.Len(t, events, 3)
	assert.Equal(t, "two", events[0].Details)
	assert.Equal(t, "three", events[1].Details)
	assert.Equal(t, "four", events[2].Details)
}

func TestRecoveryManagerMarkRecovered(t *testing.T) {
	rm := NewRecoveryManager(10)

	rm.RecordCrash("ext1", "panic", "crash1")
	rm.RecordCrash("ext2", "panic", "crash2")
	rm.RecordCrash("ext1", "panic", "crash3")

	rm.MarkRecovered("ext1")

	events := rm.EventsForExtension("ext1")
	require.Len(t, events, 2)
	assert.False(t, events[0].Recovered) // crash1 not recovered
	assert.True(t, events[1].Recovered)  // crash3 recovered

	// ext2 events should not be affected.
	ext2Events := rm.EventsForExtension("ext2")
	require.Len(t, ext2Events, 1)
	assert.False(t, ext2Events[0].Recovered)
}

func TestRecoveryManagerRecoverWithCallback(t *testing.T) {
	rm := NewRecoveryManager(10)
	restarted := false
	rm.SetRestartCallback(func(extID string) error {
		assert.Equal(t, "ext1", extID)
		restarted = true
		return nil
	})

	rm.RecordCrash("ext1", "panic", "crash")
	require.NoError(t, rm.Recover("ext1"))
	assert.True(t, restarted)

	events := rm.EventsForExtension("ext1")
	require.Len(t, events, 1)
	assert.True(t, events[0].Recovered)
}

func TestRecoveryManagerRecoverNoCallback(t *testing.T) {
	rm := NewRecoveryManager(10)
	err := rm.Recover("ext1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no restart callback set")
}

func TestRecoveryManagerRecoverCallbackFailure(t *testing.T) {
	rm := NewRecoveryManager(10)
	rm.SetRestartCallback(func(extID string) error {
		return errors.New("restart failed")
	})

	err := rm.Recover("ext1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restart failed")

	events := rm.EventsForExtension("ext1")
	require.Len(t, events, 1) // only the recovery_failed event
	assert.Equal(t, "recovery_failed", events[0].Reason)
	assert.False(t, events[0].Recovered)
}

func TestRecoveryManagerCountAndClear(t *testing.T) {
	rm := NewRecoveryManager(10)

	assert.Equal(t, 0, rm.Count())

	rm.RecordCrash("ext1", "panic", "crash")
	assert.Equal(t, 1, rm.Count())

	rm.Clear()
	assert.Equal(t, 0, rm.Count())
	assert.Empty(t, rm.Events())
}

func TestSafeRunCatchesPanic(t *testing.T) {
	rm := NewRecoveryManager(10)

	err := SafeRun(rm, "ext1", func() error {
		panic("boom")
	})
	assert.NoError(t, err) // panic is recovered, function returns normally

	events := rm.EventsForExtension("ext1")
	require.Len(t, events, 1)
	assert.Equal(t, "panic", events[0].Reason)
	assert.Contains(t, events[0].Details, "boom")
}

func TestSafeRunNoPanic(t *testing.T) {
	rm := NewRecoveryManager(10)

	err := SafeRun(rm, "ext1", func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, rm.Count())
}

func TestSafeRunNilRecoveryManager(t *testing.T) {
	// Should not panic even with nil RecoveryManager.
	err := SafeRun(nil, "ext1", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestSafeRunVM(t *testing.T) {
	rm := NewRecoveryManager(10)
	vm := goja.New()

	_, err := SafeRunVM(rm, "ext1", vm, `1 + 1`)
	assert.NoError(t, err)
	assert.Equal(t, 0, rm.Count())
}

func TestSafeRunVMCatchesPanic(t *testing.T) {
	rm := NewRecoveryManager(10)
	vm := goja.New()

	// goja's RunString returns an error for thrown exceptions, not a panic.
	_, err := SafeRunVM(rm, "ext1", vm, `throw new Error("script error")`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "script error")

	// No crash recorded because goja converts throws to errors, not panics.
	assert.Equal(t, 0, rm.Count())
}
