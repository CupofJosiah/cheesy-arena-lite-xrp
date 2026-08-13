// Copyright 2022 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Model representing the calculated totals of a match score.

package game

type ScoreSummary struct {
	AutoPoints       int
	TeleopPoints     int
	PostMatchPoints  int
	BonusPoints      int
	MatchPoints      int
	FoulPoints       int
	Score            int
	WinRankingPoints int
	PlayoffDq        bool
}

type MatchStatus int

const (
	MatchScheduled MatchStatus = iota
	MatchHidden
	RedWonMatch
	BlueWonMatch
	TieMatch
)

func (t MatchStatus) Get() MatchStatus {
	return t
}

// Determines the winner of the match given the score summaries for both alliances.
func DetermineMatchStatus(
	redScoreSummary, blueScoreSummary *ScoreSummary,
	applyPlayoffTiebreakers bool,
) (MatchStatus, string) {
	if redScoreSummary.PlayoffDq != blueScoreSummary.PlayoffDq {
		if redScoreSummary.PlayoffDq {
			return BlueWonMatch, ""
		}
		return RedWonMatch, ""
	}

	if status := comparePoints(redScoreSummary.Score, blueScoreSummary.Score); status != TieMatch {
		return status, ""
	}

	if applyPlayoffTiebreakers {
		return TieMatch, "TRUE TIE"
	}
	return TieMatch, ""
}

// Helper method to compare the red and blue alliance point totals and return the appropriate MatchStatus.
func comparePoints(redPoints, bluePoints int) MatchStatus {
	if redPoints > bluePoints {
		return RedWonMatch
	}
	if redPoints < bluePoints {
		return BlueWonMatch
	}
	return TieMatch
}
