package extensions

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStopConcurrentWithSchedule (audit C-11) races Manager.Stop against
// ScheduleAlarm. Stop used to range alarmTimers after releasing alarmMu while
// ScheduleAlarm and the timer callbacks mutated the same map under the lock --
// a data race that escalates to a fatal "concurrent map iteration and map
// write" (probe P18). A stopped manager must also reject new schedules instead
// of panicking on the nil map.
//
// The seeded 2000 timers stretch Stop's iteration so the unlocked range (the
// pre-fix shape) reliably overlaps the concurrent writers under -race.
func TestStopConcurrentWithSchedule(t *testing.T) {
	m := NewManager(t.TempDir())
	for i := 0; i < 2000; i++ {
		m.ScheduleAlarm("seed", fmt.Sprintf("s%d", i), float64(time.Now().Add(time.Hour).UnixMilli()))
	}

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				k := fmt.Sprintf("n%d", j%64)
				m.ScheduleAlarm("ext", k, float64(time.Now().Add(time.Hour).UnixMilli()))
				m.CancelAlarm("ext", k)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond)
		m.Stop()
	}()
	wg.Wait()

	// Scheduling after Stop must be a no-op, not a nil-map panic.
	require.NotPanics(t, func() {
		m.ScheduleAlarm("ext", "late", float64(time.Now().Add(time.Hour).UnixMilli()))
		m.CancelAlarm("ext", "late")
	})
	m.Stop() // idempotent
}
