// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"github.com/Team254/cheesy-arena-lite/field"
	"github.com/Team254/cheesy-arena-lite/websocket"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScoringPanel(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/panels/scoring")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Scoring Panel - Untitled Event - Cheesy Arena")
	assert.Contains(t, recorder.Body.String(), "Endgame")
	assert.NotContains(t, recorder.Body.String(), "Tower")
}

func TestScoringPanelWebsocket(t *testing.T) {
	web := setupTestWeb(t)

	server, wsUrl := web.startTestServer()
	defer server.Close()
	conn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/websocket", nil)
	assert.Nil(t, err)
	defer conn.Close()
	ws := websocket.NewTestWebsocket(conn)
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumPanels("scoring"))

	readWebsocketType(t, ws, "resetLocalState")
	readWebsocketType(t, ws, "matchLoad")
	readWebsocketType(t, ws, "matchTime")
	readWebsocketType(t, ws, "realtimeScore")

	ws.Write("score", scoringPanelScoreMessage{
		RedAuto: 3, RedTeleop: 7, RedPostMatch: 2,
		BlueAuto: 4, BlueTeleop: 5, BluePostMatch: 6,
	})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, 3, web.arena.RedRealtimeScore.CurrentScore.AutoPoints)
	assert.Equal(t, 7, web.arena.RedRealtimeScore.CurrentScore.TeleopPoints)
	assert.Equal(t, 2, web.arena.RedRealtimeScore.CurrentScore.PostMatchPoints)
	assert.Equal(t, 4, web.arena.BlueRealtimeScore.CurrentScore.AutoPoints)
	assert.Equal(t, 5, web.arena.BlueRealtimeScore.CurrentScore.TeleopPoints)
	assert.Equal(t, 6, web.arena.BlueRealtimeScore.CurrentScore.PostMatchPoints)

	ws.Write("commitMatch", nil)
	readWebsocketType(t, ws, "error")
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("scoring"))

	web.arena.MatchState = field.PostMatch
	ws.Write("commitMatch", nil)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("scoring"))
}
