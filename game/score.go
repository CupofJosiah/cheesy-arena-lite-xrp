// Copyright 2020 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Model representing the instantaneous score of a match.

package game

// Point values for each scoring element, per section 5.2 of the Iron Acres manual.
const (
	FactoryParkPointValue       = 5
	CropPointValue              = 7
	SiloDumpedPointValue        = 5
	CityLimitsProductPointValue = 10
	CityCenterProductPointValue = 15
	BarnParkPointValue          = 5
	BarnHangPointValue          = 25
)

// Point values awarded to the non-offending alliance for each penalty tier, per T03.
const (
	MinorPenaltyPointValue = 10
	MajorPenaltyPointValue = 25
)

type Score struct {
	// Autonomous period.
	FactoryParks int
	AutoCrops    int
	SilosDumped  int

	// Driver-controlled period.
	TeleopCrops        int
	CityLimitsProducts int
	CityCenterProducts int

	// Endgame. Parking and hanging are evaluated at the end of the match regardless of when the robot arrives.
	BarnParks int
	BarnHangs int

	// Penalties committed by this alliance. Points for these are awarded to the opposing alliance.
	MinorPenalties int
	MajorPenalties int

	PlayoffDq bool
}

// Summarize calculates and returns the summary fields used for ranking and display.
func (score *Score) Summarize(opponentScore *Score) *ScoreSummary {
	if opponentScore == nil {
		opponentScore = new(Score)
	}

	summary := &ScoreSummary{PlayoffDq: score.PlayoffDq}

	if score.PlayoffDq {
		summary.FoulPoints = opponentScore.PenaltyPoints()
		summary.WinRankingPoints = winRankingPoints(0, opponentScore.matchPoints())
		return summary
	}

	summary.AutoPoints = score.AutoPoints()
	summary.TeleopPoints = score.TeleopPoints()
	summary.PostMatchPoints = score.EndgamePoints()
	summary.MatchPoints = score.matchPoints()
	summary.FoulPoints = opponentScore.PenaltyPoints()
	summary.Score = summary.MatchPoints + summary.FoulPoints
	summary.WinRankingPoints = winRankingPoints(summary.Score, opponentScore.matchPoints()+score.PenaltyPoints())

	return summary
}

// AutoPoints returns the points scored during the autonomous period.
func (score *Score) AutoPoints() int {
	if score == nil {
		return 0
	}
	return score.FactoryParks*FactoryParkPointValue +
		score.AutoCrops*CropPointValue +
		score.SilosDumped*SiloDumpedPointValue
}

// TeleopPoints returns the points scored during the driver-controlled period, excluding endgame.
func (score *Score) TeleopPoints() int {
	if score == nil {
		return 0
	}
	return score.TeleopCrops*CropPointValue +
		score.CityLimitsProducts*CityLimitsProductPointValue +
		score.CityCenterProducts*CityCenterProductPointValue
}

// EndgamePoints returns the points scored for parking and hanging at the end of the match.
func (score *Score) EndgamePoints() int {
	if score == nil {
		return 0
	}
	return score.BarnParks*BarnParkPointValue + score.BarnHangs*BarnHangPointValue
}

// PenaltyPoints returns the points that this alliance's penalties award to the opposing alliance.
func (score *Score) PenaltyPoints() int {
	if score == nil {
		return 0
	}
	return score.MinorPenalties*MinorPenaltyPointValue + score.MajorPenalties*MajorPenaltyPointValue
}

func (score *Score) matchPoints() int {
	if score == nil || score.PlayoffDq {
		return 0
	}
	return score.AutoPoints() + score.TeleopPoints() + score.EndgamePoints()
}

func winRankingPoints(score, opponentScore int) int {
	if score > opponentScore {
		return 2
	}
	if score == opponentScore {
		return 1
	}
	return 0
}

// Equals returns true if and only if all fields of the two scores are equal.
func (score *Score) Equals(other *Score) bool {
	if score == nil || other == nil {
		return score == other
	}

	return *score == *other
}
