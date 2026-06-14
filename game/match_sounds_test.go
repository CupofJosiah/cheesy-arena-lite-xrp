// Copyright 2026 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package game

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUniqueMatchSounds(t *testing.T) {
	UpdateMatchSounds()

	uniqueSounds := UniqueMatchSounds()

	assert.Equal(
		t,
		[]string{
			"start",
			"end",
			"resume",
			"warning",
			"abort",
			"match_result",
			"pick_clock",
			"pick_clock_expired",
			"field_reset",
		},
		matchSoundNames(uniqueSounds),
	)
	assert.Len(t, uniqueSounds, 9)
	assert.Same(t, MatchSounds[0], uniqueSounds[0])
	assert.Same(t, MatchSounds[1], uniqueSounds[1])
	assert.Same(t, MatchSounds[3], uniqueSounds[3])
}

func TestUpdateMatchSoundsGenericTiming(t *testing.T) {
	original := MatchTiming
	defer func() {
		MatchTiming = original
		UpdateMatchSounds()
	}()

	MatchTiming.AutoDurationSec = 20
	MatchTiming.PauseDurationSec = 3
	MatchTiming.TeleopDurationSec = 140
	MatchTiming.WarningSoundTimeSec = 30
	UpdateMatchSounds()

	assert.Equal(t, "start", MatchSounds[0].Name)
	assert.Equal(t, float64(0), MatchSounds[0].MatchTimeSec)
	assert.Equal(t, "end", MatchSounds[1].Name)
	assert.Equal(t, float64(20), MatchSounds[1].MatchTimeSec)
	assert.Equal(t, "resume", MatchSounds[2].Name)
	assert.Equal(t, float64(23), MatchSounds[2].MatchTimeSec)
	assert.Equal(t, "warning", MatchSounds[3].Name)
	assert.Equal(t, float64(133), MatchSounds[3].MatchTimeSec)
	assert.Equal(t, "end", MatchSounds[4].Name)
	assert.Equal(t, float64(163), MatchSounds[4].MatchTimeSec)
}

func matchSoundNames(matchSounds []*MatchSound) []string {
	names := make([]string, 0, len(matchSounds))
	for _, sound := range matchSounds {
		names = append(names, sound.Name)
	}
	return names
}
