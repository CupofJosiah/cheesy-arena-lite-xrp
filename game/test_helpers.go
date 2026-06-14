// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Helper methods for use in tests in this package and others.

package game

func TestScore1() *Score {
	return &Score{
		AutoPoints:        18,
		TeleopPoints:      90,
		PostMatchPoints:   30,
		FoulPointsAgainst: 24,
	}
}

func TestScore2() *Score {
	return &Score{
		AutoPoints:        35,
		TeleopPoints:      148,
		PostMatchPoints:   60,
		FoulPointsAgainst: 0,
	}
}

func TestRanking1() *Ranking {
	return &Ranking{
		TeamId:       254,
		Rank:         1,
		PreviousRank: 0,
		RankingFields: RankingFields{
			RankingPoints:     20,
			MatchPoints:       625,
			AutoPoints:        90,
			PostMatchPoints:   154,
			Wins:              3,
			Losses:            2,
			Ties:              1,
			Disqualifications: 0,
			Played:            10,
			Random:            0.254,
		},
	}
}

func TestRanking2() *Ranking {
	return &Ranking{
		TeamId:       1114,
		Rank:         2,
		PreviousRank: 1,
		RankingFields: RankingFields{
			RankingPoints:     18,
			MatchPoints:       700,
			AutoPoints:        125,
			PostMatchPoints:   90,
			Wins:              1,
			Losses:            3,
			Ties:              2,
			Disqualifications: 0,
			Played:            10,
			Random:            0.1114,
		},
	}
}
