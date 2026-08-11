// Copyright 2019 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Audience sound timings.

package game

// Length of static/audio/countdown.wav, rounded up. The pause between autonomous and the driver-controlled period
// must be at least this long or the teleop horn will cut the countdown off. Update this if the clip is re-recorded.
const countdownDurationSec = 5.5

type MatchSound struct {
	Name          string
	FileExtension string
	MatchTimeSec  float64
}

// List of sounds and how many seconds into the match they are played. A negative time indicates that the sound can only
// be triggered explicitly.
var MatchSounds []*MatchSound

// UniqueMatchSounds returns the first occurrence of each sound name while preserving input order.
func UniqueMatchSounds() []*MatchSound {
	seen := make(map[string]struct{}, len(MatchSounds))
	uniqueSounds := make([]*MatchSound, 0, len(MatchSounds))
	for _, sound := range MatchSounds {
		if sound == nil {
			continue
		}
		if _, ok := seen[sound.Name]; ok {
			continue
		}

		seen[sound.Name] = struct{}{}
		uniqueSounds = append(uniqueSounds, sound)
	}

	return uniqueSounds
}

func UpdateMatchSounds() {
	matchEndSec := MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec + MatchTiming.TeleopDurationSec
	warningSec := matchEndSec - MatchTiming.WarningSoundTimeSec
	if warningSec < 0 {
		warningSec = 0
	}

	MatchSounds = []*MatchSound{
		{
			"start",
			"wav",
			0,
		},
		{
			// "Drivers, pick up your controllers. Three. Two. One." Doubles as the end-of-autonomous signal, since
			// G02 keeps hands off the controls only until the driver-controlled period begins. The clip runs about
			// 5.4 seconds, so PauseDurationSec must be at least 6 for it to finish before the teleop horn.
			"countdown",
			"wav",
			float64(MatchTiming.AutoDurationSec),
		},
		{
			"resume",
			"wav",
			float64(MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec),
		},
		{
			"warning",
			"wav",
			float64(warningSec),
		},
		{
			"end",
			"wav",
			float64(matchEndSec),
		},
		{
			"abort",
			"wav",
			-1,
		},
		{
			"match_result",
			"wav",
			-1,
		},
		{
			"pick_clock",
			"wav",
			-1,
		},
		{
			"pick_clock_expired",
			"wav",
			-1,
		},
		{
			"field_reset",
			"wav",
			-1,
		},
	}
}
