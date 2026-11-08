package extensions

import (
	"testing"
	"time"

	goja "github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

// TestScheduleAlarmDueNowReentrantNoDeadlock (audit C-28) reproduces the
// chrome.alarms chaining deadlock: an onAlarm listener that re-arms itself
// with chrome.alarms.create({when: <now|soon>}). ScheduleAlarm used to fire a
// due alarm synchronously while holding the plain alarmMu, so the listener's
// re-entrant CreateAlarm/ScheduleAlarm blocked forever on the same goroutine
// (probe: DEADLOCK CONFIRMED). The fire must always happen on a timer
// goroutine, never under alarmMu.
func TestScheduleAlarmDueNowReentrantNoDeadlock(t *testing.T) {
	m := NewManager(t.TempDir())
	defer m.Stop()

	vm := goja.New()
	rearms := make(chan struct{}, 4)

	api := NewChromeAPI(&Extension{ID: "ext-rearm"}, vm, m.messageRouter)
	// Register a JS onAlarm listener that re-enters ScheduleAlarm, exactly
	// like chrome.alarms.create({when: Date.now()}) from a listener.
	require.NoError(t, vm.Set("rearm", func(call goja.FunctionCall) goja.Value {
		rearms <- struct{}{}
		// Re-entrant schedule on the timer goroutine: must not deadlock.
		m.ScheduleAlarm("ext-rearm", "chain", float64(time.Now().UnixMilli()))
		return goja.Undefined()
	}))
	v, err := vm.RunString(`(function() { rearm(); })`)
	require.NoError(t, err)
	cb, ok := goja.AssertFunction(v)
	require.True(t, ok)
	api.mu.Lock()
	api.alarmEventListeners["onAlarm"] = append(api.alarmEventListeners["onAlarm"], cb)
	api.mu.Unlock()

	m.mu.Lock()
	m.chromeAPIs["ext-rearm"] = api
	m.mu.Unlock()

	// Due now: the pre-fix shape fired FireAlarm inline under alarmMu and the
	// re-entrant ScheduleAlarm self-deadlocked.
	done := make(chan struct{})
	go func() {
		m.ScheduleAlarm("ext-rearm", "start", float64(time.Now().UnixMilli()))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("ScheduleAlarm(self-re-arming onAlarm listener) deadlocked (audit C-28)")
	}

	// The chain must actually fire on the timer goroutine.
	select {
	case <-rearms:
	case <-time.After(3 * time.Second):
		t.Fatal("due alarm never fired on the timer goroutine")
	}
}

// TestScheduleAlarmDueNowFiresOnTimerGoroutine checks the fire itself is
// asynchronous: ScheduleAlarm returns before the listener runs.
func TestScheduleAlarmDueNowFiresOnTimerGoroutine(t *testing.T) {
	m := NewManager(t.TempDir())
	defer m.Stop()

	vm := goja.New()
	fired := make(chan struct{}, 1)
	require.NoError(t, vm.Set("markFired", func(call goja.FunctionCall) goja.Value {
		fired <- struct{}{}
		return goja.Undefined()
	}))

	api := NewChromeAPI(&Extension{ID: "ext-async"}, vm, m.messageRouter)
	v, err := vm.RunString(`(function() { markFired(); })`)
	require.NoError(t, err)
	cb, ok := goja.AssertFunction(v)
	require.True(t, ok)
	api.mu.Lock()
	api.alarmEventListeners["onAlarm"] = append(api.alarmEventListeners["onAlarm"], cb)
	api.mu.Unlock()
	m.mu.Lock()
	m.chromeAPIs["ext-async"] = api
	m.mu.Unlock()

	start := time.Now()
	m.ScheduleAlarm("ext-async", "a", float64(time.Now().UnixMilli()))
	require.True(t, time.Since(start) < 50*time.Millisecond,
		"ScheduleAlarm must return without running the listener inline")

	select {
	case <-fired:
	case <-time.After(3 * time.Second):
		t.Fatal("alarm did not fire on the timer goroutine")
	}
}
