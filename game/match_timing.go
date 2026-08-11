// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Match period timing.

package game

import "time"

const (
	ScoringGracePeriodSec  = 3
	MotorsOnExtraPeriodSec = 2
)

var MatchTiming = struct {
	AutoDurationSec     int
	PauseDurationSec    int
	TeleopDurationSec   int
	WarningSoundTimeSec int
	TimeoutDurationSec  int
	// The pause is sized so the "drivers, pick up your controllers" countdown finishes before the teleop horn.
}{10, 6, 150, 30, 0}

func GetDurationToAutoEnd() time.Duration {
	return time.Duration(MatchTiming.AutoDurationSec) * time.Second
}

func GetDurationToTeleopStart() time.Duration {
	return time.Duration(
		MatchTiming.AutoDurationSec+MatchTiming.PauseDurationSec,
	) * time.Second
}

func GetDurationToTeleopEnd() time.Duration {
	return time.Duration(
		MatchTiming.AutoDurationSec+MatchTiming.PauseDurationSec+MatchTiming.TeleopDurationSec,
	) * time.Second
}
