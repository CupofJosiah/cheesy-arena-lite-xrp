// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"encoding/json"
	"fmt"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func TestMatchReview(t *testing.T) {
	web := setupTestWeb(t)

	match1 := model.Match{Type: model.Practice, ShortName: "P1", Status: game.RedWonMatch}
	match2 := model.Match{Type: model.Qualification, ShortName: "Q1", Status: game.BlueWonMatch}
	match3 := model.Match{Type: model.Playoff, ShortName: "SF1-1", Status: game.TieMatch}
	web.arena.Database.CreateMatch(&match1)
	web.arena.Database.CreateMatch(&match2)
	web.arena.Database.CreateMatch(&match3)

	recorder := web.getHttpResponse("/match_review")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), ">P1<")
	assert.Contains(t, recorder.Body.String(), ">Q1<")
	assert.Contains(t, recorder.Body.String(), ">SF1-1<")
	assert.Contains(t, recorder.Body.String(), "match-review-rps")
}

func TestMatchReviewEditExistingResult(t *testing.T) {
	web := setupTestWeb(t)

	match := model.Match{Type: model.Practice, LongName: "Practice 1", ShortName: "P1"}
	web.arena.Database.CreateMatch(&match)
	matchResult := model.NewMatchResult()
	matchResult.MatchId = match.Id
	matchResult.RedScore = &game.Score{AutoPoints: 3, TeleopPoints: 7, PostMatchPoints: 2}
	matchResult.BlueScore = &game.Score{AutoPoints: 4, TeleopPoints: 5, FoulPointsAgainst: 1}
	assert.Nil(t, web.arena.Database.CreateMatchResult(matchResult))

	recorder := web.getHttpResponse(fmt.Sprintf("/match_review/%d/edit", match.Id))
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), " Practice 1 ")
	assert.Contains(t, recorder.Body.String(), "Auto")
	assert.Contains(t, recorder.Body.String(), "Teleop")
	assert.Contains(t, recorder.Body.String(), "Endgame")
	assert.Contains(t, recorder.Body.String(), "Foul Points Against")
	assert.NotContains(t, recorder.Body.String(), "Tower")

	postBody := fmt.Sprintf(
		"matchResultJson=%s",
		url.QueryEscape(fmt.Sprintf(
			`{"MatchId":%d,"RedScore":{"AutoPoints":1,"TeleopPoints":2,"PostMatchPoints":3},`+
				`"BlueScore":{"AutoPoints":4,"TeleopPoints":5,"PostMatchPoints":6,"FoulPointsAgainst":7},`+
				`"RedCards":{"105":"yellow"},"BlueCards":{}}`,
			match.Id,
		)),
	)
	recorder = web.postHttpResponse(fmt.Sprintf("/match_review/%d/edit", match.Id), postBody)
	assert.Equal(t, 303, recorder.Code, recorder.Body.String())

	updatedResult, err := web.arena.Database.GetMatchResultForMatch(match.Id)
	assert.Nil(t, err)
	assert.Equal(t, 6, updatedResult.RedScoreSummary().MatchPoints)
	assert.Equal(t, 15, updatedResult.BlueScoreSummary().Score)
	assert.Equal(t, "yellow", updatedResult.RedCards["105"])
}

func TestMatchReviewEditCurrentMatch(t *testing.T) {
	web := setupTestWeb(t)

	match := model.Match{Type: model.Qualification, LongName: "Qualification 352", ShortName: "Q352"}
	web.arena.Database.CreateMatch(&match)
	web.arena.LoadMatch(&match)

	postBody := fmt.Sprintf(
		"matchResultJson=%s",
		url.QueryEscape(fmt.Sprintf(
			`{"MatchId":%d,"RedScore":{"AutoPoints":1,"TeleopPoints":2,"PostMatchPoints":3},`+
				`"BlueScore":{"AutoPoints":4,"TeleopPoints":5,"FoulPointsAgainst":6},`+
				`"RedCards":{"105":"yellow"},"BlueCards":{}}`,
			match.Id,
		)),
	)
	recorder := web.postHttpResponse("/match_review/current/edit", postBody)
	assert.Equal(t, 303, recorder.Code, recorder.Body.String())
	assert.Equal(t, "/match_play", recorder.Header().Get("Location"))

	match2, _ := web.arena.Database.GetMatchById(match.Id)
	assert.Equal(t, game.MatchScheduled, match2.Status)
	assert.Equal(t, game.Score{AutoPoints: 1, TeleopPoints: 2, PostMatchPoints: 3}, web.arena.RedRealtimeScore.CurrentScore)
	assert.Equal(t, game.Score{AutoPoints: 4, TeleopPoints: 5, FoulPointsAgainst: 6}, web.arena.BlueRealtimeScore.CurrentScore)
	assert.Equal(t, "yellow", web.arena.RedRealtimeScore.Cards["105"])
}

func TestMatchReviewSummary(t *testing.T) {
	web := setupTestWeb(t)

	match := model.Match{Type: model.Qualification, LongName: "Qualification 1", ShortName: "Q1"}
	web.arena.Database.CreateMatch(&match)

	postBody := fmt.Sprintf(
		`{"MatchId":%d,"RedScore":{"AutoPoints":1,"TeleopPoints":2,"PostMatchPoints":3},`+
			`"BlueScore":{"AutoPoints":4,"TeleopPoints":5,"PostMatchPoints":6,"FoulPointsAgainst":7},`+
			`"RedCards":{},"BlueCards":{}}`,
		match.Id,
	)
	recorder := web.postHttpResponse(fmt.Sprintf("/match_review/%d/summary", match.Id), postBody)
	assert.Equal(t, 200, recorder.Code, recorder.Body.String())
	assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])

	var response MatchReviewSummaryResponse
	assert.Nil(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, 6, response.RedSummary.MatchPoints)
	assert.Equal(t, 13, response.RedSummary.Score)
	assert.Equal(t, 15, response.BlueSummary.MatchPoints)
	assert.Equal(t, 15, response.BlueSummary.Score)

	matchResult, err := web.arena.Database.GetMatchResultForMatch(match.Id)
	assert.Nil(t, err)
	assert.Nil(t, matchResult)
}
