// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package model

import (
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetNonexistentMatchResult(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	match, err := db.GetMatchResultForMatch(1114)
	assert.Nil(t, err)
	assert.Nil(t, match)
}

func TestMatchResultCrud(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	matchResult := BuildTestMatchResult(254, 5)
	assert.Nil(t, db.CreateMatchResult(matchResult))
	matchResult2, err := db.GetMatchResultForMatch(254)
	assert.Nil(t, err)
	assert.Equal(t, matchResult, matchResult2)

	matchResult.BlueScore.BarnHangs = 12
	matchResult.BlueScore.MinorPenalties = 3
	assert.Nil(t, db.UpdateMatchResult(matchResult))
	matchResult2, err = db.GetMatchResultForMatch(254)
	assert.Nil(t, err)
	assert.Equal(t, matchResult, matchResult2)

	assert.Nil(t, db.DeleteMatchResult(matchResult.Id))
	matchResult2, err = db.GetMatchResultForMatch(254)
	assert.Nil(t, err)
	assert.Nil(t, matchResult2)
}

func TestTruncateMatchResults(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	matchResult := BuildTestMatchResult(254, 1)
	assert.Nil(t, db.CreateMatchResult(matchResult))
	assert.Nil(t, db.TruncateMatchResults())
	matchResult2, err := db.GetMatchResultForMatch(254)
	assert.Nil(t, err)
	assert.Nil(t, matchResult2)
}

func TestGetMatchResultForMatch(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	matchResult := BuildTestMatchResult(254, 2)
	assert.Nil(t, db.CreateMatchResult(matchResult))
	matchResult2 := BuildTestMatchResult(254, 5)
	assert.Nil(t, db.CreateMatchResult(matchResult2))
	matchResult3 := BuildTestMatchResult(254, 4)
	assert.Nil(t, db.CreateMatchResult(matchResult3))

	// Should return the match result with the highest play number (i.e. the most recent).
	matchResult4, err := db.GetMatchResultForMatch(254)
	assert.Nil(t, err)
	assert.Equal(t, matchResult2, matchResult4)
}

func TestCorrectPlayoffScoreResetsDqState(t *testing.T) {
	matchResult := NewMatchResult()
	matchResult.RedScore.PlayoffDq = true
	matchResult.BlueScore.PlayoffDq = true
	matchResult.RedCards = map[string]string{"1": "red"}
	matchResult.BlueCards = map[string]string{}

	matchResult.CorrectPlayoffScore()
	assert.Equal(t, true, matchResult.RedScore.PlayoffDq)
	assert.Equal(t, false, matchResult.BlueScore.PlayoffDq)

	matchResult.RedCards = map[string]string{}
	matchResult.BlueCards = map[string]string{"4": "dq"}

	matchResult.CorrectPlayoffScore()
	assert.Equal(t, false, matchResult.RedScore.PlayoffDq)
	assert.Equal(t, true, matchResult.BlueScore.PlayoffDq)
}

func TestGetHighestQualificationScore(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	// BonusPoints maps straight onto the final score, so it gives exact control over the numbers here. Match IDs are
	// assigned by the database, so the caller gets the assigned one back to refer to the match by.
	order := 0
	addMatch := func(matchType MatchType, status game.MatchStatus, redScore, blueScore int) int {
		order++
		match := Match{Type: matchType, TypeOrder: order, Status: status}
		assert.Nil(t, db.CreateMatch(&match))
		assert.Nil(
			t,
			db.CreateMatchResult(
				&MatchResult{
					MatchId:    match.Id,
					PlayNumber: 1,
					MatchType:  matchType,
					RedScore:   &game.Score{BonusPoints: redScore},
					BlueScore:  &game.Score{BonusPoints: blueScore},
				},
			),
		)
		return match.Id
	}

	// No matches played yet.
	highScore, matchesPlayed, err := db.GetHighestQualificationScore(0)
	assert.Nil(t, err)
	assert.Equal(t, 0, highScore)
	// The caller has to be able to tell "nothing has beaten this" apart from "there was nothing to beat".
	assert.Equal(t, 0, matchesPlayed)

	firstMatchId := addMatch(Qualification, game.RedWonMatch, 120, 90)
	secondMatchId := addMatch(Qualification, game.BlueWonMatch, 80, 150)
	highScore, matchesPlayed, err = db.GetHighestQualificationScore(0)
	assert.Nil(t, err)
	assert.Equal(t, 150, highScore, "the losing alliance's score still counts if it is the highest")
	assert.Equal(t, 2, matchesPlayed)

	// The excluded match must not be compared against itself, and does not count towards the total either.
	highScore, matchesPlayed, err = db.GetHighestQualificationScore(secondMatchId)
	assert.Nil(t, err)
	assert.Equal(t, 120, highScore)
	assert.Equal(t, 1, matchesPlayed)

	// Practice and playoff matches are not comparable with qualification matches.
	addMatch(Practice, game.RedWonMatch, 400, 0)
	addMatch(Playoff, game.RedWonMatch, 500, 0)
	highScore, matchesPlayed, err = db.GetHighestQualificationScore(0)
	assert.Nil(t, err)
	assert.Equal(t, 150, highScore)

	// A scheduled but unplayed match has a result of zero and must not be treated as played.
	addMatch(Qualification, game.MatchScheduled, 999, 999)
	highScore, matchesPlayed, err = db.GetHighestQualificationScore(0)
	assert.Nil(t, err)
	assert.Equal(t, 150, highScore)

	// A hidden match is excluded, the same as it is from the schedule.
	addMatch(Qualification, game.MatchHidden, 888, 888)
	highScore, matchesPlayed, err = db.GetHighestQualificationScore(0)
	assert.Nil(t, err)
	assert.Equal(t, 150, highScore)
	assert.Equal(t, 2, matchesPlayed, "only the two completed qualification matches count")

	// A re-scored match counts at its latest play number, not its first.
	assert.Nil(
		t,
		db.CreateMatchResult(
			&MatchResult{
				MatchId:    firstMatchId,
				PlayNumber: 2,
				MatchType:  Qualification,
				RedScore:   &game.Score{BonusPoints: 300},
				BlueScore:  &game.Score{BonusPoints: 0},
			},
		),
	)
	highScore, matchesPlayed, err = db.GetHighestQualificationScore(0)
	assert.Nil(t, err)
	assert.Equal(t, 300, highScore)
}
