// Copyright 2020 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Model representing the instantaneous score of a match.

package game

type Score struct {
	AutoPoints        int
	TeleopPoints      int
	PostMatchPoints   int
	FoulPointsAgainst int
	PlayoffDq         bool
}

// Summarize calculates and returns the summary fields used for ranking and display.
func (score *Score) Summarize(opponentScore *Score) *ScoreSummary {
	if opponentScore == nil {
		opponentScore = new(Score)
	}

	summary := &ScoreSummary{PlayoffDq: score.PlayoffDq}

	if score.PlayoffDq {
		summary.FoulPoints = opponentScore.FoulPointsAgainst
		summary.WinRankingPoints = winRankingPoints(0, opponentScore.matchPoints())
		return summary
	}

	summary.AutoPoints = score.AutoPoints
	summary.TeleopPoints = score.TeleopPoints
	summary.PostMatchPoints = score.PostMatchPoints
	summary.MatchPoints = score.matchPoints()
	summary.FoulPoints = opponentScore.FoulPointsAgainst
	summary.Score = summary.MatchPoints + summary.FoulPoints
	summary.WinRankingPoints = winRankingPoints(summary.Score, opponentScore.matchPoints()+score.FoulPointsAgainst)

	return summary
}

func (score *Score) matchPoints() int {
	if score == nil || score.PlayoffDq {
		return 0
	}
	return score.AutoPoints + score.TeleopPoints + score.PostMatchPoints
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

	return score.AutoPoints == other.AutoPoints &&
		score.TeleopPoints == other.TeleopPoints &&
		score.PostMatchPoints == other.PostMatchPoints &&
		score.FoulPointsAgainst == other.FoulPointsAgainst &&
		score.PlayoffDq == other.PlayoffDq
}
