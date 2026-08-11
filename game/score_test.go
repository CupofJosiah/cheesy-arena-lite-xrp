// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScorePointsByPeriod(t *testing.T) {
	score := &Score{
		FactoryParks:       2,
		AutoCrops:          3,
		SilosDumped:        4,
		TeleopCrops:        5,
		CityLimitsProducts: 6,
		CityCenterProducts: 7,
		BarnParks:          1,
		BarnHangs:          2,
		MinorPenalties:     3,
		MajorPenalties:     4,
	}

	// 2*5 + 3*7 + 4*5
	assert.Equal(t, 51, score.AutoPoints())

	// 5*7 + 6*10 + 7*15
	assert.Equal(t, 200, score.TeleopPoints())

	// 1*5 + 2*25
	assert.Equal(t, 55, score.EndgamePoints())

	// 3*10 + 4*25
	assert.Equal(t, 130, score.PenaltyPoints())
}

func TestScoreNilReceiverPoints(t *testing.T) {
	var score *Score
	assert.Equal(t, 0, score.AutoPoints())
	assert.Equal(t, 0, score.TeleopPoints())
	assert.Equal(t, 0, score.EndgamePoints())
	assert.Equal(t, 0, score.PenaltyPoints())
}

func TestScoreSummary(t *testing.T) {
	// Auto 12 (1 park + 1 crop), teleop 25 (10 + 15), endgame 30 (1 park + 1 hang), one minor penalty.
	redScore := &Score{
		FactoryParks:       1,
		AutoCrops:          1,
		CityLimitsProducts: 1,
		CityCenterProducts: 1,
		BarnParks:          1,
		BarnHangs:          1,
		MinorPenalties:     1,
	}
	// Auto 5, teleop 14 (2 crops), endgame 25 (1 hang), one major penalty.
	blueScore := &Score{
		FactoryParks:   1,
		TeleopCrops:    2,
		BarnHangs:      1,
		MajorPenalties: 1,
	}

	redSummary := redScore.Summarize(blueScore)
	assert.Equal(t, 12, redSummary.AutoPoints)
	assert.Equal(t, 25, redSummary.TeleopPoints)
	assert.Equal(t, 30, redSummary.PostMatchPoints)
	assert.Equal(t, 67, redSummary.MatchPoints)
	// Blue's major penalty awards 25 points to red.
	assert.Equal(t, 25, redSummary.FoulPoints)
	assert.Equal(t, 92, redSummary.Score)
	assert.Equal(t, 2, redSummary.WinRankingPoints)

	blueSummary := blueScore.Summarize(redScore)
	assert.Equal(t, 5, blueSummary.AutoPoints)
	assert.Equal(t, 14, blueSummary.TeleopPoints)
	assert.Equal(t, 25, blueSummary.PostMatchPoints)
	assert.Equal(t, 44, blueSummary.MatchPoints)
	// Red's minor penalty awards 10 points to blue.
	assert.Equal(t, 10, blueSummary.FoulPoints)
	assert.Equal(t, 54, blueSummary.Score)
	assert.Equal(t, 0, blueSummary.WinRankingPoints)
}

func TestScoreWinRankingPoints(t *testing.T) {
	assert.Equal(t, 2, (&Score{AutoCrops: 2}).Summarize(&Score{AutoCrops: 1}).WinRankingPoints)
	assert.Equal(t, 1, (&Score{AutoCrops: 2}).Summarize(&Score{AutoCrops: 2}).WinRankingPoints)
	assert.Equal(t, 0, (&Score{AutoCrops: 1}).Summarize(&Score{AutoCrops: 2}).WinRankingPoints)
}

// A penalty must be able to swing the win, since penalty points count toward the final score.
func TestScoreWinRankingPointsWithPenalties(t *testing.T) {
	// Red out-scores blue on match points by one crop, but concedes a major penalty worth more.
	redScore := &Score{AutoCrops: 2, MajorPenalties: 1}
	blueScore := &Score{AutoCrops: 1}

	assert.Equal(t, 14, redScore.Summarize(blueScore).Score)
	assert.Equal(t, 32, blueScore.Summarize(redScore).Score)
	assert.Equal(t, 0, redScore.Summarize(blueScore).WinRankingPoints)
	assert.Equal(t, 2, blueScore.Summarize(redScore).WinRankingPoints)
}

func TestScorePlayoffDisqualification(t *testing.T) {
	redScore := &Score{AutoCrops: 2, TeleopCrops: 3, BarnParks: 1, PlayoffDq: true}
	blueScore := &Score{AutoCrops: 1, TeleopCrops: 1}

	redSummary := redScore.Summarize(blueScore)
	assert.Equal(t, 0, redSummary.Score)
	assert.True(t, redSummary.PlayoffDq)

	blueSummary := blueScore.Summarize(redScore)
	assert.Equal(t, 14, blueSummary.Score)
	assert.False(t, blueSummary.PlayoffDq)
}

func TestScoreEquals(t *testing.T) {
	score1 := TestScore1()
	score2 := TestScore1()
	assert.True(t, score1.Equals(score2))

	// Changing any single field must break equality.
	for _, mutate := range []func(*Score){
		func(s *Score) { s.FactoryParks++ },
		func(s *Score) { s.AutoCrops++ },
		func(s *Score) { s.SilosDumped++ },
		func(s *Score) { s.TeleopCrops++ },
		func(s *Score) { s.CityLimitsProducts++ },
		func(s *Score) { s.CityCenterProducts++ },
		func(s *Score) { s.BarnParks++ },
		func(s *Score) { s.BarnHangs++ },
		func(s *Score) { s.MinorPenalties++ },
		func(s *Score) { s.MajorPenalties++ },
		func(s *Score) { s.PlayoffDq = !s.PlayoffDq },
	} {
		mutated := TestScore1()
		mutate(mutated)
		assert.False(t, score1.Equals(mutated))
	}
}

func TestScoreEqualsNil(t *testing.T) {
	var nilScore *Score
	assert.True(t, nilScore.Equals(nil))
	assert.False(t, nilScore.Equals(TestScore1()))
	assert.False(t, TestScore1().Equals(nil))
}
