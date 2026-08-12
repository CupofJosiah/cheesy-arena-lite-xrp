// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Game-specific fields by which teams are ranked and the logic for sorting rankings.

package game

import "math/rand"

type RankingFields struct {
	RankingPoints     int
	MatchPoints       int
	AutoPoints        int
	PostMatchPoints   int
	Wins              int
	Losses            int
	Ties              int
	Disqualifications int
	Played            int
	Random            float64
}

type Ranking struct {
	TeamId       TeamId `db:"id,manual"`
	Rank         int
	PreviousRank int
	RankingFields
}

type Rankings []Ranking

var RankingRandomFloat64 = rand.Float64

func (fields *RankingFields) AddScoreSummary(ownScore *ScoreSummary, opponentScore *ScoreSummary, disqualified bool) {
	fields.Played += 1

	// Store a random value to be used as the last tiebreaker if necessary.
	fields.Random = RankingRandomFloat64()

	if disqualified {
		fields.Disqualifications += 1
		return
	}

	fields.RankingPoints += ownScore.WinRankingPoints
	switch ownScore.WinRankingPoints {
	case 2:
		fields.Wins += 1
	case 1:
		fields.Ties += 1
	default:
		fields.Losses += 1
	}

	fields.MatchPoints += ownScore.MatchPoints
	fields.AutoPoints += ownScore.AutoPoints
	fields.PostMatchPoints += ownScore.PostMatchPoints
}

func (fields RankingFields) TeleopPoints() int {
	return fields.MatchPoints - fields.AutoPoints - fields.PostMatchPoints
}

// Helper function to implement the required interface for Sort.
func (rankings Rankings) Len() int {
	return len(rankings)
}

// Helper function to implement the required interface for Sort.
func (rankings Rankings) Less(i, j int) bool {
	a := rankings[i]
	b := rankings[j]

	// Use cross-multiplication to keep it in integer math.
	if a.RankingPoints*b.Played == b.RankingPoints*a.Played {
		if a.MatchPoints*b.Played == b.MatchPoints*a.Played {
			if a.AutoPoints*b.Played == b.AutoPoints*a.Played {
				if a.PostMatchPoints*b.Played == b.PostMatchPoints*a.Played {
					return a.Random > b.Random
				}
				return a.PostMatchPoints*b.Played > b.PostMatchPoints*a.Played
			}
			return a.AutoPoints*b.Played > b.AutoPoints*a.Played
		}
		return a.MatchPoints*b.Played > b.MatchPoints*a.Played
	}
	return a.RankingPoints*b.Played > b.RankingPoints*a.Played
}

// Helper function to implement the required interface for Sort.
func (rankings Rankings) Swap(i, j int) {
	rankings[i], rankings[j] = rankings[j], rankings[i]
}
