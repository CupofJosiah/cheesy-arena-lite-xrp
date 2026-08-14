// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Model and datastore CRUD methods for the results (score and fouls) from a match at an event.

package model

import (
	"github.com/Team254/cheesy-arena-lite/game"
)

type MatchResult struct {
	Id         int `db:"id"`
	MatchId    int
	PlayNumber int
	MatchType  MatchType
	RedScore   *game.Score
	BlueScore  *game.Score
	RedCards   map[string]string
	BlueCards  map[string]string
}

// Returns a new match result object with empty slices instead of nil.
func NewMatchResult() *MatchResult {
	matchResult := new(MatchResult)
	matchResult.EnsureInitialized()
	return matchResult
}

func (matchResult *MatchResult) EnsureInitialized() {
	if matchResult.RedScore == nil {
		matchResult.RedScore = new(game.Score)
	}
	if matchResult.BlueScore == nil {
		matchResult.BlueScore = new(game.Score)
	}
	if matchResult.RedCards == nil {
		matchResult.RedCards = make(map[string]string)
	}
	if matchResult.BlueCards == nil {
		matchResult.BlueCards = make(map[string]string)
	}
}

func (database *Database) CreateMatchResult(matchResult *MatchResult) error {
	return database.matchResultTable.create(matchResult)
}

func (database *Database) GetMatchResultForMatch(matchId int) (*MatchResult, error) {
	matchResults, err := database.matchResultTable.getAll()
	if err != nil {
		return nil, err
	}

	var mostRecentMatchResult *MatchResult
	for i, matchResult := range matchResults {
		if matchResult.MatchId == matchId &&
			(mostRecentMatchResult == nil || matchResult.PlayNumber > mostRecentMatchResult.PlayNumber) {
			mostRecentMatchResult = &matchResults[i]
		}
	}
	return mostRecentMatchResult, nil
}

// Returns the highest score achieved by either alliance in any completed qualification match other than the one with
// the given ID, along with how many such matches there are. Pass the ID of the match being posted so that its own
// score is not compared against itself; the score is already committed to the database by the time the result is
// announced, and the announcement is regenerated for every display that connects, so the comparison has to be stable
// rather than depend on when it is made. The count lets the caller tell "nothing has beaten this" apart from "there
// was nothing to beat", since otherwise any score at all sets a record in the first match of an event.
func (database *Database) GetHighestQualificationScore(excludeMatchId int) (int, int, error) {
	matches, err := database.GetMatchesByType(Qualification, false)
	if err != nil {
		return 0, 0, err
	}
	matchResults, err := database.matchResultTable.getAll()
	if err != nil {
		return 0, 0, err
	}

	// Index the latest result per match in one pass. Looking each one up individually would rescan the whole table
	// per match, and this runs on every posted score and every display that connects.
	latestResults := make(map[int]*MatchResult)
	for i, matchResult := range matchResults {
		latest, ok := latestResults[matchResult.MatchId]
		if !ok || matchResult.PlayNumber > latest.PlayNumber {
			latestResults[matchResult.MatchId] = &matchResults[i]
		}
	}

	highScore, matchesPlayed := 0, 0
	for _, match := range matches {
		if match.Id == excludeMatchId || !match.IsComplete() {
			continue
		}
		matchResult, ok := latestResults[match.Id]
		if !ok {
			continue
		}
		matchesPlayed++
		for _, score := range []int{matchResult.RedScoreSummary().Score, matchResult.BlueScoreSummary().Score} {
			if score > highScore {
				highScore = score
			}
		}
	}
	return highScore, matchesPlayed, nil
}

func (database *Database) UpdateMatchResult(matchResult *MatchResult) error {
	return database.matchResultTable.update(matchResult)
}

func (database *Database) DeleteMatchResult(id int) error {
	return database.matchResultTable.delete(id)
}

func (database *Database) TruncateMatchResults() error {
	return database.matchResultTable.truncate()
}

// Calculates and returns the summary fields used for ranking and display for the red alliance.
func (matchResult *MatchResult) RedScoreSummary() *game.ScoreSummary {
	matchResult.EnsureInitialized()
	return matchResult.RedScore.Summarize(matchResult.BlueScore)
}

// Calculates and returns the summary fields used for ranking and display for the blue alliance.
func (matchResult *MatchResult) BlueScoreSummary() *game.ScoreSummary {
	matchResult.EnsureInitialized()
	return matchResult.BlueScore.Summarize(matchResult.RedScore)
}

// Checks the score for disqualifications or a tie and adjusts it appropriately.
func (matchResult *MatchResult) CorrectPlayoffScore() {
	matchResult.EnsureInitialized()
	matchResult.RedScore.PlayoffDq = false
	matchResult.BlueScore.PlayoffDq = false
	for _, card := range matchResult.RedCards {
		if card == "red" || card == "dq" {
			matchResult.RedScore.PlayoffDq = true
		}
	}
	for _, card := range matchResult.BlueCards {
		if card == "red" || card == "dq" {
			matchResult.BlueScore.PlayoffDq = true
		}
	}
}
