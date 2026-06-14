// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package game

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddScoreSummary(t *testing.T) {
	randomizer := rand.New(rand.NewSource(0))
	RankingRandomFloat64 = randomizer.Float64
	redSummary := &ScoreSummary{
		AutoPoints:       30,
		TeleopPoints:     18,
		PostMatchPoints:  19,
		MatchPoints:      67,
		Score:            67,
		WinRankingPoints: 0,
	}
	blueSummary := &ScoreSummary{
		AutoPoints:       16,
		TeleopPoints:     31,
		PostMatchPoints:  14,
		MatchPoints:      61,
		Score:            81,
		WinRankingPoints: 2,
	}
	tieSummary := &ScoreSummary{
		AutoPoints:       10,
		TeleopPoints:     20,
		PostMatchPoints:  30,
		MatchPoints:      60,
		Score:            60,
		WinRankingPoints: 1,
	}
	rankingFields := RankingFields{}

	rankingFields.AddScoreSummary(redSummary, blueSummary, false)
	assert.Equal(t, RankingFields{0, 67, 30, 19, 0, 1, 0, 0, 1, 0.9451961492941164}, rankingFields)

	rankingFields.AddScoreSummary(blueSummary, redSummary, false)
	assert.Equal(t, RankingFields{2, 128, 46, 33, 1, 1, 0, 0, 2, 0.24496508529377975}, rankingFields)

	rankingFields.AddScoreSummary(tieSummary, tieSummary, false)
	assert.Equal(t, RankingFields{3, 188, 56, 63, 1, 1, 1, 0, 3, 0.6559562651954052}, rankingFields)

	rankingFields.AddScoreSummary(blueSummary, redSummary, true)
	assert.Equal(t, RankingFields{3, 188, 56, 63, 1, 1, 1, 1, 4, 0.05434383959970039}, rankingFields)
}

func TestSortRankings(t *testing.T) {
	rankings := make(Rankings, 10)
	rankings[0] = Ranking{TeamId: 1, RankingFields: RankingFields{RankingPoints: 50, MatchPoints: 50, AutoPoints: 50, PostMatchPoints: 50, Played: 10, Random: 0.49}}
	rankings[1] = Ranking{TeamId: 2, RankingFields: RankingFields{RankingPoints: 50, MatchPoints: 50, AutoPoints: 50, PostMatchPoints: 50, Played: 10, Random: 0.51}}
	rankings[2] = Ranking{TeamId: 3, RankingFields: RankingFields{RankingPoints: 50, MatchPoints: 50, AutoPoints: 50, PostMatchPoints: 49, Played: 10, Random: 0.50}}
	rankings[3] = Ranking{TeamId: 4, RankingFields: RankingFields{RankingPoints: 50, MatchPoints: 50, AutoPoints: 50, PostMatchPoints: 51, Played: 10, Random: 0.50}}
	rankings[4] = Ranking{TeamId: 5, RankingFields: RankingFields{RankingPoints: 50, MatchPoints: 50, AutoPoints: 49, PostMatchPoints: 50, Played: 10, Random: 0.50}}
	rankings[5] = Ranking{TeamId: 6, RankingFields: RankingFields{RankingPoints: 50, MatchPoints: 50, AutoPoints: 51, PostMatchPoints: 50, Played: 10, Random: 0.50}}
	rankings[6] = Ranking{TeamId: 7, RankingFields: RankingFields{RankingPoints: 50, MatchPoints: 49, AutoPoints: 50, PostMatchPoints: 50, Played: 10, Random: 0.50}}
	rankings[7] = Ranking{TeamId: 8, RankingFields: RankingFields{RankingPoints: 50, MatchPoints: 51, AutoPoints: 50, PostMatchPoints: 50, Played: 10, Random: 0.50}}
	rankings[8] = Ranking{TeamId: 9, RankingFields: RankingFields{RankingPoints: 49, MatchPoints: 50, AutoPoints: 50, PostMatchPoints: 50, Played: 10, Random: 0.50}}
	rankings[9] = Ranking{TeamId: 10, RankingFields: RankingFields{RankingPoints: 51, MatchPoints: 50, AutoPoints: 50, PostMatchPoints: 50, Played: 10, Random: 0.50}}
	sort.Sort(rankings)
	assert.Equal(t, []int{10, 8, 6, 4, 2, 1, 3, 5, 7, 9}, rankingTeamIds(rankings))

	rankings = make(Rankings, 3)
	rankings[0] = Ranking{TeamId: 1, RankingFields: RankingFields{RankingPoints: 10, MatchPoints: 25, AutoPoints: 25, PostMatchPoints: 25, Played: 5, Random: 0.49}}
	rankings[1] = Ranking{TeamId: 2, RankingFields: RankingFields{RankingPoints: 19, MatchPoints: 50, AutoPoints: 50, PostMatchPoints: 50, Played: 9, Random: 0.51}}
	rankings[2] = Ranking{TeamId: 3, RankingFields: RankingFields{RankingPoints: 20, MatchPoints: 50, AutoPoints: 50, PostMatchPoints: 50, Played: 10, Random: 0.51}}
	sort.Sort(rankings)
	assert.Equal(t, []int{2, 3, 1}, rankingTeamIds(rankings))
}

func rankingTeamIds(rankings Rankings) []int {
	teamIds := make([]int, len(rankings))
	for i, ranking := range rankings {
		teamIds[i] = ranking.TeamId
	}
	return teamIds
}
