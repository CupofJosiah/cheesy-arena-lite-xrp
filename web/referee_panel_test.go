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
	assert.Contains(t, recorder.Body.String(), "Foul Points Against")
	assert.Contains(t, recorder.Body.String(), "Commit & Post")
	assert.Contains(t, recorder.Body.String(), "Scores not committed")
	assert.NotContains(t, recorder.Body.String(), `id="redFoulPointsAgainst" type="number" min="0" step="1" value="0" disabled`)
	assert.NotContains(t, recorder.Body.String(), `id="blueFoulPointsAgainst" type="number" min="0" step="1" value="0" disabled`)
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

	ws.Write("foulPoints", struct {
		RedFoulPointsAgainst  int
		BlueFoulPointsAgainst int
	}{5, 7})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, 5, web.arena.RedRealtimeScore.CurrentScore.FoulPointsAgainst)
	assert.Equal(t, 7, web.arena.BlueRealtimeScore.CurrentScore.FoulPointsAgainst)

	ws.Write("foulPointsAgainst", struct {
		Alliance string
		Points   int
	}{"blue", 11})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, 11, web.arena.BlueRealtimeScore.CurrentScore.FoulPointsAgainst)

	ws.Write("card", struct {
		Alliance string
		TeamId   int
		Card     string
	}{"red", 256, "yellow"})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, "yellow", web.arena.RedRealtimeScore.Cards["256"])

	web.arena.CurrentMatch.Type = model.Playoff
	web.arena.CurrentMatch.Blue1 = 1679
	web.arena.CurrentMatch.Blue2 = 1680
	web.arena.CurrentMatch.Blue3 = 1681
	ws.Write("card", struct {
		Alliance string
		TeamId   int
		Card     string
	}{"blue", 1680, "red"})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, "red", web.arena.BlueRealtimeScore.Cards["1679"])
	assert.Equal(t, "red", web.arena.BlueRealtimeScore.Cards["1680"])
	assert.Equal(t, "red", web.arena.BlueRealtimeScore.Cards["1681"])

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
	web.arena.RedRealtimeScore.CurrentScore = game.Score{AutoPoints: 1, TeleopPoints: 2, PostMatchPoints: 3}
	web.arena.BlueRealtimeScore.CurrentScore = game.Score{AutoPoints: 4, TeleopPoints: 5, PostMatchPoints: 6}

	server, wsUrl := web.startTestServer()
	defer server.Close()
	conn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/referee/websocket", nil)
	assert.Nil(t, err)
	defer conn.Close()
	ws := websocket.NewTestWebsocket(conn)
	readWebsocketMultiple(t, ws, 4)

	ws.Write("commitAndPost", nil)
	readWebsocketType(t, ws, "scoringStatus")
	assert.Equal(t, 6, web.arena.SavedMatchResult.RedScoreSummary().Score)
	assert.Equal(t, 15, web.arena.SavedMatchResult.BlueScoreSummary().Score)
	assert.True(t, web.arena.SavedMatchResult.RedScore.Equals(&game.Score{AutoPoints: 1, TeleopPoints: 2, PostMatchPoints: 3}))
	assert.True(t, web.arena.SavedMatchResult.BlueScore.Equals(&game.Score{AutoPoints: 4, TeleopPoints: 5, PostMatchPoints: 6}))
}
