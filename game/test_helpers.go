// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Helper methods for use in tests in this package and others.

package game

// Auto 24, teleop 92, endgame 30 (match points 146); concedes 35 penalty points to the opponent.
func TestScore1() *Score {
	return &Score{
		FactoryParks:       1,
		AutoCrops:          2,
		SilosDumped:        1,
		TeleopCrops:        6,
		CityLimitsProducts: 2,
		CityCenterProducts: 2,
		BarnParks:          1,
		BarnHangs:          1,
		MinorPenalties:     1,
		MajorPenalties:     1,
	}
}

// Auto 41, teleop 133, endgame 50 (match points 224); concedes no penalty points.
func TestScore2() *Score {
	return &Score{
		FactoryParks:       2,
		AutoCrops:          3,
		SilosDumped:        2,
		TeleopCrops:        9,
		CityLimitsProducts: 1,
		CityCenterProducts: 4,
		BarnHangs:          2,
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
