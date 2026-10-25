package profiler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProfilerRestartExistingProfile(t *testing.T) {
	p := CreateProfiler()
	p.Start("work")
	time.Sleep(10 * time.Millisecond)
	p.Start("work") // restart branch: updates startTime of existing profile
	assert.Len(t, p.GetAllProfiles(), 1)

	p.Stop("work")
	p.Stop("nonexistent") // no-op stop branch

	profile := p.GetProfile("work")
	assert.NotNil(t, profile)
	assert.Equal(t, "work", profile.GetName())
	assert.Greater(t, profile.GetElapsedTime(), time.Duration(0))
}

func TestProfilerGetMissingProfile(t *testing.T) {
	p := CreateProfiler()
	assert.Empty(t, p.GetAllProfiles())
	assert.Nil(t, p.GetProfile("no-such-profile"))
}
