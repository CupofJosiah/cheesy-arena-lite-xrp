// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"github.com/Team254/cheesy-arena-lite/field"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/tournament"
	"github.com/Team254/cheesy-arena-lite/websocket"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMatchPlay(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/match_play")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Are you sure you want to discard the results for this match?")
	assert.Contains(t, recorder.Body.String(), "Scoring")
}

func TestCommitMatchGenericScores(t *testing.T) {
	web := setupTestWeb(t)

	match := &model.Match{Type: model.Qualification, Red1: 101, Red2: 102, Red3: 103, Blue1: 104, Blue2: 105, Blue3: 106}
	assert.Nil(t, web.arena.Database.CreateMatch(match))
	matchResult := &model.MatchResult{
		MatchId: match.Id,
		RedScore: &game.Score{
			AutoPoints:        3,
			TeleopPoints:      7,
			PostMatchPoints:   2,
			FoulPointsAgainst: 1,
		},
		BlueScore: &game.Score{
			AutoPoints:        4,
			TeleopPoints:      5,
			PostMatchPoints:   1,
			FoulPointsAgainst: 6,
		},
		RedCards:  map[string]string{},
		BlueCards: map[string]string{},
	}

	err := web.commitMatchScore(match, matchResult, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, matchResult.PlayNumber)
	match, _ = web.arena.Database.GetMatchById(match.Id)
	assert.Equal(t, game.RedWonMatch, match.Status)

	storedResult, err := web.arena.Database.GetMatchResultForMatch(match.Id)
	assert.Nil(t, err)
	assert.Equal(t, matchResult, storedResult)
	assert.Equal(t, 18, storedResult.RedScoreSummary().Score)
	assert.Equal(t, 11, storedResult.BlueScoreSummary().Score)
}

func TestCommitPlayoffDq(t *testing.T) {
	web := setupTestWeb(t)

	tournament.CreateTestAlliances(web.arena.Database, 2)
	web.arena.EventSettings.PlayoffType = model.SingleEliminationPlayoff
	web.arena.EventSettings.NumPlayoffAlliances = 2
	web.arena.CreatePlayoffTournament()
	assert.Nil(t, web.arena.CreatePlayoffMatches(time.Now()))
	matches, err := web.arena.Database.GetMatchesByType(model.Playoff, false)
	assert.Nil(t, err)
	match := &matches[0]

	matchResult := model.NewMatchResult()
	matchResult.MatchId = match.Id
	matchResult.MatchType = match.Type
	matchResult.RedScore.AutoPoints = 20
	matchResult.BlueScore.AutoPoints = 1
	matchResult.RedCards = map[string]string{"1": "dq"}

	assert.Nil(t, web.commitMatchScore(match, matchResult, true))
	match, _ = web.arena.Database.GetMatchById(match.Id)
	assert.Equal(t, game.BlueWonMatch, match.Status)
	assert.True(t, matchResult.RedScore.PlayoffDq)
	assert.False(t, matchResult.BlueScore.PlayoffDq)
	assert.Equal(t, 0, matchResult.RedScoreSummary().Score)
}

func TestMatchPlayWebsocketCommitCurrentGenericScore(t *testing.T) {
	web := setupTestWeb(t)
	web.arena.CurrentMatch = &model.Match{Type: model.Test}
	web.arena.MatchState = field.PostMatch
	web.arena.RedRealtimeScore.CurrentScore = game.Score{AutoPoints: 1, TeleopPoints: 2, PostMatchPoints: 3}
	web.arena.BlueRealtimeScore.CurrentScore = game.Score{AutoPoints: 4, TeleopPoints: 5, PostMatchPoints: 6}

	server, wsUrl := web.startTestServer()
	defer server.Close()
	conn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/match_play/websocket", nil)
	assert.Nil(t, err)
	defer conn.Close()
	ws := websocket.NewTestWebsocket(conn)
	readWebsocketMultiple(t, ws, 10)

	ws.Write("commitAndPost", nil)
	messages := readWebsocketMultiple(t, ws, 6)
	assert.NotNil(t, messages["scorePosted"])
	assert.Equal(t, 6, web.arena.SavedMatchResult.RedScoreSummary().Score)
	assert.Equal(t, 15, web.arena.SavedMatchResult.BlueScoreSummary().Score)
}
