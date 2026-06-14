package game

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMatchTimingDurations(t *testing.T) {
	original := MatchTiming
	defer func() {
		MatchTiming = original
	}()

	MatchTiming.AutoDurationSec = 12
	MatchTiming.PauseDurationSec = 3
	MatchTiming.TeleopDurationSec = 140

	assert.Equal(t, 12*time.Second, GetDurationToAutoEnd())
	assert.Equal(t, 15*time.Second, GetDurationToTeleopStart())
	assert.Equal(t, 155*time.Second, GetDurationToTeleopEnd())
}
