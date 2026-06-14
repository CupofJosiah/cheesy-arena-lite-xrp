// Copyright 2022 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package game

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScoreSummaryDetermineMatchStatus(t *testing.T) {
	assertMatchStatus := func(
		expectedStatus MatchStatus,
		expectedTiebreaker string,
		redScoreSummary *ScoreSummary,
		blueScoreSummary *ScoreSummary,
		applyPlayoffTiebreakers bool,
	) {
		status, tiebreaker := DetermineMatchStatus(redScoreSummary, blueScoreSummary, applyPlayoffTiebreakers)
		assert.Equal(t, expectedStatus, status)
		assert.Equal(t, expectedTiebreaker, tiebreaker)
	}

	redScoreSummary := &ScoreSummary{Score: 10}
	blueScoreSummary := &ScoreSummary{Score: 10}
	assertMatchStatus(TieMatch, "", redScoreSummary, blueScoreSummary, false)
	assertMatchStatus(TieMatch, "TRUE TIE", redScoreSummary, blueScoreSummary, true)

	redScoreSummary.Score = 11
	assertMatchStatus(RedWonMatch, "", redScoreSummary, blueScoreSummary, false)
	assertMatchStatus(RedWonMatch, "", redScoreSummary, blueScoreSummary, true)

	blueScoreSummary.Score = 12
	assertMatchStatus(BlueWonMatch, "", redScoreSummary, blueScoreSummary, false)
	assertMatchStatus(BlueWonMatch, "", redScoreSummary, blueScoreSummary, true)

	redScoreSummary.Score = 12
	assertMatchStatus(TieMatch, "", redScoreSummary, blueScoreSummary, false)
	assertMatchStatus(TieMatch, "TRUE TIE", redScoreSummary, blueScoreSummary, true)

	redScoreSummary = &ScoreSummary{Score: 0, PlayoffDq: true}
	blueScoreSummary = &ScoreSummary{Score: 0}
	assertMatchStatus(BlueWonMatch, "", redScoreSummary, blueScoreSummary, false)
	assertMatchStatus(BlueWonMatch, "", redScoreSummary, blueScoreSummary, true)

	redScoreSummary = &ScoreSummary{Score: 0}
	blueScoreSummary = &ScoreSummary{Score: 0, PlayoffDq: true}
	assertMatchStatus(RedWonMatch, "", redScoreSummary, blueScoreSummary, false)
	assertMatchStatus(RedWonMatch, "", redScoreSummary, blueScoreSummary, true)

	redScoreSummary = &ScoreSummary{Score: 0, PlayoffDq: true}
	blueScoreSummary = &ScoreSummary{Score: 0, PlayoffDq: true}
	assertMatchStatus(TieMatch, "", redScoreSummary, blueScoreSummary, false)
	assertMatchStatus(TieMatch, "TRUE TIE", redScoreSummary, blueScoreSummary, true)
}
