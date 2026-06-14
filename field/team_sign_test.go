// Copyright 2024 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package field

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGenerateTeamSignAllianceScores(t *testing.T) {
	arena := setupTestArena(t)
	arena.RedRealtimeScore.CurrentScore = game.Score{AutoPoints: 4, TeleopPoints: 6, PostMatchPoints: 2}
	arena.BlueRealtimeScore.CurrentScore = game.Score{AutoPoints: 1, TeleopPoints: 3, FoulPointsAgainst: 5}

	assert.Equal(t, "R015-B004", generateTeamSignAllianceScores(arena, true))
	assert.Equal(t, "B004-R015", generateTeamSignAllianceScores(arena, false))
}

func TestGenerateInMatchTimerRearText(t *testing.T) {
	arena := setupTestArena(t)
	arena.RedRealtimeScore.CurrentScore = game.Score{AutoPoints: 10}
	arena.BlueRealtimeScore.CurrentScore = game.Score{FoulPointsAgainst: 2}

	assert.Equal(t, "1:23       R012-B000", generateInMatchTimerRearText(arena, true, "1:23"))
}

func TestTeamSignTimerUsesGenericTeleopCountdown(t *testing.T) {
	arena := setupTestArena(t)
	arena.MatchState = TeleopPeriod
	arena.MatchStartTime = time.Now().Add(
		-time.Duration(game.MatchTiming.AutoDurationSec+game.MatchTiming.PauseDurationSec+10) * time.Second,
	)
	signs := NewTeamSigns()
	signs.RedTimer.address = 50

	signs.Update(arena)

	remaining := game.MatchTiming.TeleopDurationSec - 10
	assert.Equal(t, formatCountdown(remaining), signs.RedTimer.frontText)
}

func TestTeamSignLogoUsesCurrentYear(t *testing.T) {
	arena := setupTestArena(t)
	arena.AllianceStationDisplayMode = "logo"

	frontText, frontColor, rearText := generateTimerTexts(arena, "00:00", "")

	assert.Equal(t, fmt.Sprintf("%5d", time.Now().Year()), frontText)
	assert.Equal(t, whiteColor, frontColor)
	assert.Equal(t, "", rearText)
}

func TestTeamSignPostMatchShowsNextTeam(t *testing.T) {
	arena := setupTestArena(t)
	arena.MatchState = PostMatch
	arena.AllianceStationDisplayMode = "match"
	arena.AllianceStations["R1"].Team = &model.Team{Id: 254}

	sign := TeamSign{address: 51, nextMatchTeamId: 1114}
	frontText, frontColor, rearText := sign.generateTeamNumberTexts(arena, "R1", true, "00:00", "")

	assert.Equal(t, "  254", frontText)
	assert.Equal(t, redColor, frontColor)
	assert.Equal(t, "Next Team Up: 1114", rearText)
}

func formatCountdown(seconds int) string {
	return time.Unix(int64(seconds), 0).UTC().Format("04:05")
}
