// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/field"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
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
	assert.Contains(t, recorder.Body.String(), "Autonomous")
	assert.Contains(t, recorder.Body.String(), "Driver Controlled")
	assert.Contains(t, recorder.Body.String(), "Endgame")
	assert.Contains(t, recorder.Body.String(), "Bonus Points")
	assert.Contains(t, recorder.Body.String(), `id="red-FactoryParks"`)
	assert.Contains(t, recorder.Body.String(), `id="blue-CityCenterProducts"`)
	assert.Contains(t, recorder.Body.String(), `id="red-BarnHangs"`)
	assert.Contains(t, recorder.Body.String(), `id="blue-BonusPoints"`)
	// The increment is a literal in the template so that saving the file cannot break a running server, so guard here
	// that it has not drifted from the constant the rest of the system uses.
	assert.Contains(
		t, recorder.Body.String(), fmt.Sprintf("adjustScore('red', 'BonusPoints', -%d);", game.BonusPointIncrement),
	)
	assert.Contains(
		t, recorder.Body.String(), fmt.Sprintf("adjustScore('red', 'BonusPoints', %d);", game.BonusPointIncrement),
	)
	assert.Contains(t, recorder.Body.String(), fmt.Sprintf("%d points per tap", game.BonusPointIncrement))
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
		Red: scoringPanelAllianceScore{
			FactoryParks: 1, AutoCrops: 2, SilosDumped: 1, TeleopCrops: 3, CityLimitsProducts: 1,
			CityCenterProducts: 1, BarnParks: 1, BarnHangs: 1, BonusPoints: 20,
		},
		Blue: scoringPanelAllianceScore{
			FactoryParks: 2, AutoCrops: 1, TeleopCrops: 2, CityLimitsProducts: 2, BarnParks: 2,
			// Negative element counts are clamped to zero rather than subtracting points, but a bonus is an
			// adjustment and is kept as entered.
			BarnHangs: -3, BonusPoints: -10,
		},
	})
	readWebsocketType(t, ws, "realtimeScore")
	redScore := &web.arena.RedRealtimeScore.CurrentScore
	blueScore := &web.arena.BlueRealtimeScore.CurrentScore
	assert.Equal(t, 1, redScore.FactoryParks)
	assert.Equal(t, 2, redScore.AutoCrops)
	assert.Equal(t, 1, redScore.SilosDumped)
	assert.Equal(t, 3, redScore.TeleopCrops)
	assert.Equal(t, 1, redScore.CityLimitsProducts)
	assert.Equal(t, 1, redScore.CityCenterProducts)
	assert.Equal(t, 1, redScore.BarnParks)
	assert.Equal(t, 1, redScore.BarnHangs)
	assert.Equal(t, 24, redScore.AutoPoints())
	assert.Equal(t, 46, redScore.TeleopPoints())
	assert.Equal(t, 30, redScore.EndgamePoints())
	assert.Equal(t, 20, redScore.BonusPoints)
	assert.Equal(t, 0, blueScore.BarnHangs)
	assert.Equal(t, -10, blueScore.BonusPoints)
	assert.Equal(t, 17, blueScore.AutoPoints())
	assert.Equal(t, 34, blueScore.TeleopPoints())
	assert.Equal(t, 10, blueScore.EndgamePoints())

	// The scoring panel does not own penalties, so committing a score must leave them untouched.
	assert.Equal(t, 0, redScore.MinorPenalties)
	assert.Equal(t, 0, redScore.MajorPenalties)

	ws.Write("commitMatch", nil)
	readWebsocketType(t, ws, "error")
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("scoring"))

	web.arena.MatchState = field.PostMatch
	ws.Write("commitMatch", nil)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("scoring"))
}

// A bonus entered on the panel has to survive the whole path to the posted score: the websocket message, the arena's
// realtime score, the committed match result, and the summary the displays read.
func TestScoringPanelBonusPointsReachFinalScore(t *testing.T) {
	web := setupTestWeb(t)

	match := &model.Match{Type: model.Qualification, Red1: "101", Red2: "102", Blue1: "104", Blue2: "105"}
	assert.Nil(t, web.arena.Database.CreateMatch(match))
	assert.Nil(t, web.arena.LoadMatch(match))

	server, wsUrl := web.startTestServer()
	defer server.Close()
	conn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/websocket", nil)
	assert.Nil(t, err)
	defer conn.Close()
	ws := websocket.NewTestWebsocket(conn)

	readWebsocketType(t, ws, "resetLocalState")
	readWebsocketType(t, ws, "matchLoad")
	readWebsocketType(t, ws, "matchTime")
	readWebsocketType(t, ws, "realtimeScore")

	// Both alliances score one crop, so the 30-point bonus is the only thing separating them.
	ws.Write("score", scoringPanelScoreMessage{
		Red:  scoringPanelAllianceScore{AutoCrops: 1, BonusPoints: 30},
		Blue: scoringPanelAllianceScore{AutoCrops: 1},
	})
	readWebsocketType(t, ws, "realtimeScore")
	assert.Equal(t, 30, web.arena.RedRealtimeScore.CurrentScore.BonusPoints)

	web.arena.MatchState = field.PostMatch
	assert.Nil(t, web.commitCurrentMatchScore())

	storedResult, err := web.arena.Database.GetMatchResultForMatch(match.Id)
	assert.Nil(t, err)
	assert.Equal(t, 30, storedResult.RedScore.BonusPoints)
	assert.Equal(t, 37, storedResult.RedScoreSummary().Score)
	assert.Equal(t, 30, storedResult.RedScoreSummary().BonusPoints)
	assert.Equal(t, 7, storedResult.BlueScoreSummary().Score)

	storedMatch, err := web.arena.Database.GetMatchById(match.Id)
	assert.Nil(t, err)
	assert.Equal(t, game.RedWonMatch, storedMatch.Status)
}
