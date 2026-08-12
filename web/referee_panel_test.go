// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"github.com/Team254/cheesy-arena-lite/field"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/websocket"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRefereePanel(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/panels/referee")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Referee Panel - Untitled Event - Cheesy Arena")
	assert.Contains(t, recorder.Body.String(), "Penalties Committed")
	assert.Contains(t, recorder.Body.String(), `id="red-minorPenalties"`)
	assert.Contains(t, recorder.Body.String(), `id="red-majorPenalties"`)
	assert.Contains(t, recorder.Body.String(), `id="blue-minorPenalties"`)
	assert.Contains(t, recorder.Body.String(), `id="blue-majorPenalties"`)
	assert.Contains(t, recorder.Body.String(), "Commit & Post")
	assert.Contains(t, recorder.Body.String(), "Scores not committed")
	assert.NotContains(t, recorder.Body.String(), "Foul Points Against")
	assert.NotContains(t, recorder.Body.String(), "Tower")
}

func TestRefereePanelWebsocket(t *testing.T) {
	web := setupTestWeb(t)

	server, wsUrl := web.startTestServer()
	defer server.Close()
	conn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/referee/websocket", nil)
	assert.Nil(t, err)
	defer conn.Close()
	ws := websocket.NewTestWebsocket(conn)

	readWebsocketType(t, ws, "matchLoad")
	readWebsocketType(t, ws, "matchTime")
	readWebsocketType(t, ws, "realtimeScore")
	readWebsocketType(t, ws, "scoringStatus")

	ws.Write("penalties", struct {
		RedMinorPenalties  int
		RedMajorPenalties  int
		BlueMinorPenalties int
		BlueMajorPenalties int
	}{5, 1, 7, 2})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, 5, web.arena.RedRealtimeScore.CurrentScore.MinorPenalties)
	assert.Equal(t, 1, web.arena.RedRealtimeScore.CurrentScore.MajorPenalties)
	assert.Equal(t, 7, web.arena.BlueRealtimeScore.CurrentScore.MinorPenalties)
	assert.Equal(t, 2, web.arena.BlueRealtimeScore.CurrentScore.MajorPenalties)

	// Negative absolute counts are clamped to zero.
	ws.Write("penalties", struct {
		RedMinorPenalties  int
		RedMajorPenalties  int
		BlueMinorPenalties int
		BlueMajorPenalties int
	}{-1, 1, 7, 2})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, 0, web.arena.RedRealtimeScore.CurrentScore.MinorPenalties)

	ws.Write("addPenalty", struct {
		Alliance string
		Tier     string
		Delta    int
	}{"blue", "minor", 3})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, 10, web.arena.BlueRealtimeScore.CurrentScore.MinorPenalties)

	// Decrementing past zero is clamped.
	ws.Write("addPenalty", struct {
		Alliance string
		Tier     string
		Delta    int
	}{"blue", "major", -5})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, 0, web.arena.BlueRealtimeScore.CurrentScore.MajorPenalties)

	ws.Write("addPenalty", struct {
		Alliance string
		Tier     string
		Delta    int
	}{"red", "bogus", 1})
	assert.Contains(t, readWebsocketError(t, ws), "Invalid penalty tier 'bogus'.")

	ws.Write("card", struct {
		Alliance string
		TeamId   game.TeamId
		Card     string
	}{"red", "256", "yellow"})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, "yellow", web.arena.RedRealtimeScore.Cards["256"])

	web.arena.CurrentMatch.Type = model.Playoff
	web.arena.CurrentMatch.Blue1 = "1679"
	web.arena.CurrentMatch.Blue2 = "1680"
	ws.Write("card", struct {
		Alliance string
		TeamId   game.TeamId
		Card     string
	}{"blue", "1680", "red"})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, "red", web.arena.BlueRealtimeScore.Cards["1679"])
	assert.Equal(t, "red", web.arena.BlueRealtimeScore.Cards["1680"])

	assert.False(t, web.arena.RedRealtimeScore.FoulsCommitted)
	assert.False(t, web.arena.BlueRealtimeScore.FoulsCommitted)
	web.arena.MatchState = field.PostMatch
	ws.Write("commitMatch", nil)
	readWebsocketType(t, ws, "scoringStatus")
	assert.True(t, web.arena.RedRealtimeScore.FoulsCommitted)
	assert.True(t, web.arena.BlueRealtimeScore.FoulsCommitted)
}

func TestRefereePanelWebsocketCommitAndPost(t *testing.T) {
	web := setupTestWeb(t)
	web.arena.CurrentMatch = &model.Match{Type: model.Test}
	web.arena.MatchState = field.PostMatch
	// Red scores 5 + 7 + 5 = 17; blue scores 7 + 10 + 25 = 42.
	redScore := game.Score{FactoryParks: 1, TeleopCrops: 1, BarnParks: 1}
	blueScore := game.Score{AutoCrops: 1, CityLimitsProducts: 1, BarnHangs: 1}
	web.arena.RedRealtimeScore.CurrentScore = redScore
	web.arena.BlueRealtimeScore.CurrentScore = blueScore

	server, wsUrl := web.startTestServer()
	defer server.Close()
	conn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/referee/websocket", nil)
	assert.Nil(t, err)
	defer conn.Close()
	ws := websocket.NewTestWebsocket(conn)
	readWebsocketMultiple(t, ws, 4)

	ws.Write("commitAndPost", nil)
	readWebsocketType(t, ws, "scoringStatus")
	assert.Equal(t, 17, web.arena.SavedMatchResult.RedScoreSummary().Score)
	assert.Equal(t, 42, web.arena.SavedMatchResult.BlueScoreSummary().Score)
	assert.True(t, web.arena.SavedMatchResult.RedScore.Equals(&redScore))
	assert.True(t, web.arena.SavedMatchResult.BlueScore.Equals(&blueScore))
}
