// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScoreSummary(t *testing.T) {
	redScore := &Score{AutoPoints: 12, TeleopPoints: 34, PostMatchPoints: 5, FoulPointsAgainst: 7}
	blueScore := &Score{AutoPoints: 10, TeleopPoints: 30, PostMatchPoints: 4, FoulPointsAgainst: 3}

	redSummary := redScore.Summarize(blueScore)
	assert.Equal(t, 12, redSummary.AutoPoints)
	assert.Equal(t, 34, redSummary.TeleopPoints)
	assert.Equal(t, 5, redSummary.PostMatchPoints)
	assert.Equal(t, 51, redSummary.MatchPoints)
	assert.Equal(t, 3, redSummary.FoulPoints)
	assert.Equal(t, 54, redSummary.Score)
	assert.Equal(t, 2, redSummary.WinRankingPoints)

	blueSummary := blueScore.Summarize(redScore)
	assert.Equal(t, 10, blueSummary.AutoPoints)
	assert.Equal(t, 30, blueSummary.TeleopPoints)
	assert.Equal(t, 4, blueSummary.PostMatchPoints)
	assert.Equal(t, 44, blueSummary.MatchPoints)
	assert.Equal(t, 7, blueSummary.FoulPoints)
	assert.Equal(t, 51, blueSummary.Score)
	assert.Equal(t, 0, blueSummary.WinRankingPoints)
}

func TestScoreWinRankingPoints(t *testing.T) {
	assert.Equal(t, 2, (&Score{AutoPoints: 10}).Summarize(&Score{AutoPoints: 9}).WinRankingPoints)
	assert.Equal(t, 1, (&Score{AutoPoints: 10}).Summarize(&Score{AutoPoints: 10}).WinRankingPoints)
	assert.Equal(t, 0, (&Score{AutoPoints: 9}).Summarize(&Score{AutoPoints: 10}).WinRankingPoints)
}

func TestScorePlayoffDisqualification(t *testing.T) {
	redScore := &Score{AutoPoints: 12, TeleopPoints: 34, PostMatchPoints: 5, PlayoffDq: true}
	blueScore := &Score{AutoPoints: 10, TeleopPoints: 30, PostMatchPoints: 4}

	redSummary := redScore.Summarize(blueScore)
	assert.Equal(t, 0, redSummary.Score)
	assert.True(t, redSummary.PlayoffDq)

	blueSummary := blueScore.Summarize(redScore)
	assert.Equal(t, 44, blueSummary.Score)
	assert.False(t, blueSummary.PlayoffDq)
}

func TestScoreEquals(t *testing.T) {
	score1 := &Score{AutoPoints: 1, TeleopPoints: 2, PostMatchPoints: 3, FoulPointsAgainst: 4, PlayoffDq: true}
	score2 := &Score{AutoPoints: 1, TeleopPoints: 2, PostMatchPoints: 3, FoulPointsAgainst: 4, PlayoffDq: true}
	assert.True(t, score1.Equals(score2))

	score2.AutoPoints++
	assert.False(t, score1.Equals(score2))
	score2.AutoPoints--
	score2.TeleopPoints++
	assert.False(t, score1.Equals(score2))
	score2.TeleopPoints--
	score2.PostMatchPoints++
	assert.False(t, score1.Equals(score2))
	score2.PostMatchPoints--
	score2.FoulPointsAgainst++
	assert.False(t, score1.Equals(score2))
	score2.FoulPointsAgainst--
	score2.PlayoffDq = false
	assert.False(t, score1.Equals(score2))
}
