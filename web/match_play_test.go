// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"encoding/json"
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

func TestCommitMatchScores(t *testing.T) {
	web := setupTestWeb(t)

	match := &model.Match{Type: model.Qualification, Red1: "101", Red2: "102", Blue1: "104", Blue2: "105"}
	assert.Nil(t, web.arena.Database.CreateMatch(match))
	matchResult := &model.MatchResult{
		MatchId: match.Id,
		RedScore: &game.Score{
			// Auto 24, teleop 46, endgame 30; concedes 10 penalty points to blue.
			FactoryParks:       1,
			AutoCrops:          2,
			SilosDumped:        1,
			TeleopCrops:        3,
			CityLimitsProducts: 1,
			CityCenterProducts: 1,
			BarnParks:          1,
			BarnHangs:          1,
			MinorPenalties:     1,
		},
		BlueScore: &game.Score{
			// Auto 17, teleop 34, endgame 10; concedes 25 penalty points to red.
			FactoryParks:       2,
			AutoCrops:          1,
			TeleopCrops:        2,
			CityLimitsProducts: 2,
			BarnParks:          2,
			MajorPenalties:     1,
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
	assert.Equal(t, 125, storedResult.RedScoreSummary().Score)
	assert.Equal(t, 71, storedResult.BlueScoreSummary().Score)
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
	matchResult.RedScore.FactoryParks = 4
	matchResult.BlueScore.AutoCrops = 1
	matchResult.RedCards = map[string]string{"1": "dq"}

	assert.Nil(t, web.commitMatchScore(match, matchResult, true))
	match, _ = web.arena.Database.GetMatchById(match.Id)
	assert.Equal(t, game.BlueWonMatch, match.Status)
	assert.True(t, matchResult.RedScore.PlayoffDq)
	assert.False(t, matchResult.BlueScore.PlayoffDq)
	assert.Equal(t, 0, matchResult.RedScoreSummary().Score)
}

func TestMatchPlayWebsocketCommitCurrentScore(t *testing.T) {
	web := setupTestWeb(t)
	web.arena.CurrentMatch = &model.Match{Type: model.Test}
	web.arena.MatchState = field.PostMatch
	// Red scores 5 + 7 + 5 = 17; blue scores 7 + 10 + 25 = 42.
	web.arena.RedRealtimeScore.CurrentScore = game.Score{FactoryParks: 1, TeleopCrops: 1, BarnParks: 1}
	web.arena.BlueRealtimeScore.CurrentScore = game.Score{AutoCrops: 1, CityLimitsProducts: 1, BarnHangs: 1}

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
	assert.Equal(t, 17, web.arena.SavedMatchResult.RedScoreSummary().Score)
	assert.Equal(t, 42, web.arena.SavedMatchResult.BlueScoreSummary().Score)
}

// Verifies that the posted-score announcement flags a new event high score, which is what raises the badge on the
// audience display's final score screen.
func TestCommitMatchScoreHighScore(t *testing.T) {
	web := setupTestWeb(t)

	type highScoreFlags struct {
		RedHighScore  bool
		BlueHighScore bool
	}
	// BonusPoints maps straight onto the final score, giving exact control over the numbers.
	post := func(matchType model.MatchType, redScore, blueScore int) highScoreFlags {
		match := &model.Match{Type: matchType, Red1: "101", Red2: "102", Blue1: "104", Blue2: "105"}
		assert.Nil(t, web.arena.Database.CreateMatch(match))
		matchResult := &model.MatchResult{
			MatchId:   match.Id,
			MatchType: matchType,
			RedScore:  &game.Score{BonusPoints: redScore},
			BlueScore: &game.Score{BonusPoints: blueScore},
			RedCards:  map[string]string{},
			BlueCards: map[string]string{},
		}
		assert.Nil(t, web.commitMatchScore(match, matchResult, false))

		// The message is an anonymous struct, so read the flags back out through JSON the way a client would.
		messageJson, err := json.Marshal(web.arena.GenerateScorePostedMessage())
		assert.Nil(t, err)
		var flags highScoreFlags
		assert.Nil(t, json.Unmarshal(messageJson, &flags))
		return flags
	}

	// The first match of the event has nothing to beat, so any score at all must not claim a record.
	assert.Equal(t, highScoreFlags{}, post(model.Qualification, 120, 90))

	// A score that beats the standing 120 is a new high score, and only the top score in the match claims it even
	// though both alliances beat the old record.
	assert.Equal(t, highScoreFlags{RedHighScore: true}, post(model.Qualification, 200, 150))

	// Nothing beats the standing 200.
	assert.Equal(t, highScoreFlags{}, post(model.Qualification, 199, 10))

	// Equalling the standing record is not beating it.
	assert.Equal(t, highScoreFlags{}, post(model.Qualification, 200, 10))

	// The losing alliance can still set the high score.
	assert.Equal(t, highScoreFlags{BlueHighScore: true}, post(model.Qualification, 10, 250))

	// A tie at the top flags both alliances.
	assert.Equal(t, highScoreFlags{RedHighScore: true, BlueHighScore: true}, post(model.Qualification, 300, 300))

	// Practice matches are not comparable with qualification matches and never claim a record.
	assert.Equal(t, highScoreFlags{}, post(model.Practice, 900, 800))

	// Regenerating the same announcement, as happens for every display that connects, must give the same answer even
	// though the match being announced is already committed to the database by then.
	messageJson, err := json.Marshal(web.arena.GenerateScorePostedMessage())
	assert.Nil(t, err)
	var flags highScoreFlags
	assert.Nil(t, json.Unmarshal(messageJson, &flags))
	assert.Equal(t, highScoreFlags{}, flags)

	// Neither do playoff matches. Committing one needs alliance records that this test has no reason to build, so
	// announce it directly instead; the branch under test is the match type check.
	web.arena.SavedMatch = &model.Match{Type: model.Playoff, Red1: "101", Red2: "102", Blue1: "104", Blue2: "105"}
	web.arena.SavedMatchResult = &model.MatchResult{
		MatchType: model.Playoff,
		RedScore:  &game.Score{BonusPoints: 900},
		BlueScore: &game.Score{BonusPoints: 800},
		RedCards:  map[string]string{},
		BlueCards: map[string]string{},
	}
	messageJson, err = json.Marshal(web.arena.GenerateScorePostedMessage())
	assert.Nil(t, err)
	assert.Nil(t, json.Unmarshal(messageJson, &flags))
	assert.Equal(t, highScoreFlags{}, flags)

	// And the qualification record still stands afterwards.
	assert.Equal(t, highScoreFlags{RedHighScore: true}, post(model.Qualification, 301, 5))
}
