// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package field

import (
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/playoff"
	"github.com/Team254/cheesy-arena-lite/tournament"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAssignTeam(t *testing.T) {
	arena := setupTestArena(t)

	team := model.Team{Id: 254}
	err := arena.Database.CreateTeam(&team)
	assert.Nil(t, err)
	err = arena.Database.CreateTeam(&model.Team{Id: 1114})
	assert.Nil(t, err)

	err = arena.assignTeam(254, "B1")
	assert.Nil(t, err)
	assert.Equal(t, team, *arena.AllianceStations["B1"].Team)

	// Nothing should happen if the same team is assigned to the same station.
	arena.AllianceStations["B1"].Ready = true
	err = arena.assignTeam(254, "B1")
	assert.Nil(t, err)
	assert.Equal(t, team, *arena.AllianceStations["B1"].Team)
	assert.True(t, arena.AllianceStations["B1"].Ready)

	// Test reassignment to another team, which must clear the station's readiness.
	err = arena.assignTeam(1114, "B1")
	assert.Nil(t, err)
	assert.Equal(t, 1114, arena.AllianceStations["B1"].Team.Id)
	assert.False(t, arena.AllianceStations["B1"].Ready)

	// Check assigning zero as the team number.
	err = arena.assignTeam(0, "R2")
	assert.Nil(t, err)
	assert.Nil(t, arena.AllianceStations["R2"].Team)

	// Check assigning to a non-existent station.
	err = arena.assignTeam(254, "R3")
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Invalid alliance station")
	}
}

func TestArenaCheckCanStartMatch(t *testing.T) {
	arena := setupTestArena(t)

	// A match cannot start until every station is either staged or bypassed.
	err := arena.checkCanStartMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot start match until all robots are ready or bypassed")
	}
	arena.AllianceStations["R1"].Bypass = true
	arena.AllianceStations["R2"].Bypass = true
	arena.AllianceStations["B1"].Bypass = true
	err = arena.checkCanStartMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot start match until all robots are ready or bypassed")
	}

	// Marking the last station ready, rather than bypassing it, is also sufficient.
	assert.Nil(t, arena.SetStationReady("B2", true))
	assert.Nil(t, arena.checkCanStartMatch())

	// An emergency stop blocks the start even when the station is otherwise ready.
	arena.AllianceStations["B2"].EStop = true
	err = arena.checkCanStartMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot start match while an emergency stop is active")
	}
	arena.AllianceStations["B2"].EStop = false
	assert.Nil(t, arena.checkCanStartMatch())

	// Setting readiness on an unknown station is an error.
	if err = arena.SetStationReady("R3", true); assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Invalid alliance station")
	}
}

func TestArenaMatchFlow(t *testing.T) {
	arena := setupTestArena(t)

	arena.Database.CreateTeam(&model.Team{Id: 254})
	assert.Nil(t, arena.assignTeam(254, "B2"))
	arena.Database.CreateTeam(&model.Team{Id: 1678})
	assert.Nil(t, arena.assignTeam(1678, "R2"))

	assert.Equal(t, PreMatch, arena.MatchState)
	arena.Update()
	assert.Equal(t, PreMatch, arena.MatchState)

	// Check match start, autonomous and transition to teleop.
	arena.AllianceStations["R1"].Bypass = true
	arena.AllianceStations["B1"].Bypass = true
	assert.Nil(t, arena.SetStationReady("R2", true))
	assert.Nil(t, arena.SetStationReady("B2", true))
	assert.Nil(t, arena.StartMatch())
	arena.Update()
	assert.Equal(t, AutoPeriod, arena.MatchState)
	arena.Update()
	assert.Equal(t, AutoPeriod, arena.MatchState)

	arena.MatchStartTime = time.Now().Add(
		-time.Duration(game.MatchTiming.AutoDurationSec) * time.Second,
	)
	arena.Update()
	assert.Equal(t, PausePeriod, arena.MatchState)
	arena.Update()
	assert.Equal(t, PausePeriod, arena.MatchState)

	arena.MatchStartTime = time.Now().Add(
		-time.Duration(
			game.MatchTiming.AutoDurationSec+game.MatchTiming.PauseDurationSec,
		) * time.Second,
	)
	arena.Update()
	assert.Equal(t, TeleopPeriod, arena.MatchState)
	arena.Update()
	assert.Equal(t, TeleopPeriod, arena.MatchState)

	// Check match end.
	arena.MatchStartTime = time.Now().Add(
		-time.Duration(
			game.MatchTiming.AutoDurationSec+game.MatchTiming.PauseDurationSec+game.MatchTiming.TeleopDurationSec,
		) * time.Second,
	)
	arena.Update()
	assert.Equal(t, PostMatch, arena.MatchState)
	arena.Update()
	assert.Equal(t, PostMatch, arena.MatchState)

	// Resetting the match must clear the per-station flags.
	arena.AllianceStations["R1"].Bypass = true
	arena.AllianceStations["B2"].EStop = true
	arena.ResetMatch()
	arena.Update()
	assert.Equal(t, PreMatch, arena.MatchState)
	assert.False(t, arena.AllianceStations["R1"].Bypass)
	assert.False(t, arena.AllianceStations["B2"].EStop)
	assert.False(t, arena.AllianceStations["R2"].Ready)
}

func TestArenaStateEnforcement(t *testing.T) {
	arena := setupTestArena(t)

	arena.AllianceStations["R1"].Bypass = true
	arena.AllianceStations["R2"].Bypass = true
	arena.AllianceStations["B1"].Bypass = true
	arena.AllianceStations["B2"].Bypass = true

	err := arena.LoadMatch(new(model.Match))
	assert.Nil(t, err)
	err = arena.AbortMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot abort match when")
	}
	err = arena.StartMatch()
	assert.Nil(t, err)
	err = arena.LoadMatch(new(model.Match))
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot load match while")
	}
	err = arena.StartMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot start match while")
	}
	err = arena.ResetMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot reset match while")
	}
	arena.MatchState = AutoPeriod
	err = arena.LoadMatch(new(model.Match))
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot load match while")
	}
	err = arena.StartMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot start match while")
	}
	err = arena.ResetMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot reset match while")
	}
	arena.MatchState = PausePeriod
	err = arena.LoadMatch(new(model.Match))
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot load match while")
	}
	err = arena.StartMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot start match while")
	}
	err = arena.ResetMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot reset match while")
	}
	arena.MatchState = TeleopPeriod
	err = arena.LoadMatch(new(model.Match))
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot load match while")
	}
	err = arena.StartMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot start match while")
	}
	err = arena.ResetMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot reset match while")
	}
	arena.MatchState = PostMatch
	err = arena.LoadMatch(new(model.Match))
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot load match while")
	}
	err = arena.StartMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot start match while")
	}
	err = arena.AbortMatch()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cannot abort match when")
	}

	err = arena.ResetMatch()
	assert.Nil(t, err)
	assert.Equal(t, PreMatch, arena.MatchState)
	err = arena.ResetMatch()
	assert.Nil(t, err)
	err = arena.LoadMatch(new(model.Match))
	assert.Nil(t, err)
}

func TestLoadNextMatch(t *testing.T) {
	arena := setupTestArena(t)

	arena.Database.CreateTeam(&model.Team{Id: 1114})
	practiceMatch1 := model.Match{Type: model.Practice, TypeOrder: 1}
	practiceMatch2 := model.Match{Type: model.Practice, TypeOrder: 2, Status: game.RedWonMatch}
	practiceMatch3 := model.Match{Type: model.Practice, TypeOrder: 3}
	arena.Database.CreateMatch(&practiceMatch1)
	arena.Database.CreateMatch(&practiceMatch2)
	arena.Database.CreateMatch(&practiceMatch3)
	qualificationMatch1 := model.Match{Type: model.Qualification, TypeOrder: 1, Status: game.BlueWonMatch}
	qualificationMatch2 := model.Match{Type: model.Qualification, TypeOrder: 2}
	arena.Database.CreateMatch(&qualificationMatch1)
	arena.Database.CreateMatch(&qualificationMatch2)

	// Test match should be followed by another, empty test match.
	assert.Equal(t, 0, arena.CurrentMatch.Id)
	err := arena.SubstituteTeams(1114, 0, 0, 0)
	assert.Nil(t, err)
	arena.CurrentMatch.Status = game.TieMatch
	err = arena.LoadNextMatch(false)
	assert.Nil(t, err)
	assert.Equal(t, 0, arena.CurrentMatch.Id)
	assert.Equal(t, 0, arena.CurrentMatch.Red1)
	assert.Equal(t, false, arena.CurrentMatch.IsComplete())

	// Other matches should be loaded by type until they're all complete.
	err = arena.LoadMatch(&practiceMatch2)
	assert.Nil(t, err)
	err = arena.LoadNextMatch(false)
	assert.Nil(t, err)
	assert.Equal(t, practiceMatch1.Id, arena.CurrentMatch.Id)
	practiceMatch1.Status = game.RedWonMatch
	arena.Database.UpdateMatch(&practiceMatch1)
	err = arena.LoadNextMatch(false)
	assert.Nil(t, err)
	assert.Equal(t, practiceMatch3.Id, arena.CurrentMatch.Id)
	practiceMatch3.Status = game.BlueWonMatch
	arena.Database.UpdateMatch(&practiceMatch3)
	err = arena.LoadNextMatch(false)
	assert.Nil(t, err)
	assert.Equal(t, 0, arena.CurrentMatch.Id)
	assert.Equal(t, model.Test, arena.CurrentMatch.Type)

	err = arena.LoadMatch(&qualificationMatch1)
	assert.Nil(t, err)
	err = arena.LoadNextMatch(false)
	assert.Nil(t, err)
	assert.Equal(t, qualificationMatch2.Id, arena.CurrentMatch.Id)
}

func TestSubstituteTeam(t *testing.T) {
	arena := setupTestArena(t)
	tournament.CreateTestAlliances(arena.Database, 2)
	arena.PlayoffTournament, _ = playoff.NewPlayoffTournament(
		arena.EventSettings.PlayoffType, arena.EventSettings.NumPlayoffAlliances,
	)

	arena.Database.CreateTeam(&model.Team{Id: 101})
	arena.Database.CreateTeam(&model.Team{Id: 102})
	arena.Database.CreateTeam(&model.Team{Id: 103})
	arena.Database.CreateTeam(&model.Team{Id: 104})
	arena.Database.CreateTeam(&model.Team{Id: 105})
	arena.Database.CreateTeam(&model.Team{Id: 106})
	arena.Database.CreateTeam(&model.Team{Id: 107})

	// Substitute teams into test match.
	err := arena.SubstituteTeams(0, 0, 101, 0)
	assert.Nil(t, err)
	assert.Equal(t, 101, arena.CurrentMatch.Blue1)
	assert.Equal(t, 101, arena.AllianceStations["B1"].Team.Id)
	err = arena.assignTeam(104, "R4")
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Invalid alliance station")
	}

	// Substitute teams into practice match.
	match := model.Match{Type: model.Practice, Red1: 101, Red2: 102, Blue1: 103, Blue2: 104}
	arena.Database.CreateMatch(&match)
	arena.LoadMatch(&match)
	err = arena.SubstituteTeams(107, 102, 103, 104)
	assert.Nil(t, err)
	assert.Equal(t, 107, arena.CurrentMatch.Red1)
	assert.Equal(t, 107, arena.AllianceStations["R1"].Team.Id)
	matchResult := model.NewMatchResult()
	matchResult.MatchId = arena.CurrentMatch.Id

	// Check that substitution is disallowed in qualification matches.
	match = model.Match{Type: model.Qualification, Red1: 101, Red2: 102, Blue1: 103, Blue2: 104}
	arena.Database.CreateMatch(&match)
	arena.LoadMatch(&match)
	err = arena.SubstituteTeams(107, 102, 103, 104)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Can't substitute teams for qualification matches.")
	}
	match = model.Match{Type: model.Playoff, Red1: 101, Red2: 102, Blue1: 103, Blue2: 104}
	arena.Database.CreateMatch(&match)
	arena.LoadMatch(&match)
	assert.Nil(t, arena.SubstituteTeams(107, 102, 103, 104))

	// Check that loading a nonexistent team fails.
	err = arena.SubstituteTeams(101, 102, 103, 108)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Team 108 is not present at the event.")
	}
}

func TestArenaTimeout(t *testing.T) {
	arena := setupTestArena(t)

	// Test regular ending of timeout.
	timeoutDurationSec := 9
	assert.Nil(t, arena.StartTimeout("Break 1", timeoutDurationSec))
	assert.Equal(t, timeoutDurationSec, game.MatchTiming.TimeoutDurationSec)
	assert.Equal(t, TimeoutActive, arena.MatchState)
	assert.Equal(t, "Break 1", arena.breakDescription)
	assert.Equal(t, "Test Match", arena.breakNextMatchName)
	arena.MatchStartTime = time.Now().Add(-time.Duration(timeoutDurationSec) * time.Second)
	arena.Update()
	assert.Equal(t, PostTimeout, arena.MatchState)
	arena.MatchStartTime = time.Now().Add(-time.Duration(timeoutDurationSec+postTimeoutSec) * time.Second)
	arena.Update()
	assert.Equal(t, PreMatch, arena.MatchState)

	// Test ad-hoc timeout display text.
	timeoutDurationSec = 14
	assert.Nil(t, arena.StartAdHocTimeout("Repair Break", "", timeoutDurationSec))
	assert.Equal(t, "Repair Break", arena.breakDescription)
	assert.Equal(t, "", arena.breakNextMatchName)
	arena.SetTimeoutDisplay("Inspection Break", "Practice 1")
	assert.Equal(t, "Inspection Break", arena.breakDescription)
	assert.Equal(t, "Practice 1", arena.breakNextMatchName)
	arena.MatchStartTime = time.Now().Add(-time.Duration(timeoutDurationSec) * time.Second)
	arena.Update()
	assert.Equal(t, PostTimeout, arena.MatchState)
	arena.MatchStartTime = time.Now().Add(-time.Duration(timeoutDurationSec+postTimeoutSec) * time.Second)
	arena.Update()
	assert.Equal(t, PreMatch, arena.MatchState)

	// Test early cancellation of timeout.
	timeoutDurationSec = 28
	assert.Nil(t, arena.StartTimeout("Break 2", timeoutDurationSec))
	assert.Equal(t, "Break 2", arena.breakDescription)
	assert.Equal(t, TimeoutActive, arena.MatchState)
	assert.Equal(t, timeoutDurationSec, game.MatchTiming.TimeoutDurationSec)
	assert.Nil(t, arena.AbortMatch())
	arena.Update()
	assert.Equal(t, PostTimeout, arena.MatchState)
	arena.MatchStartTime = time.Now().Add(-time.Duration(timeoutDurationSec+postTimeoutSec) * time.Second)
	arena.Update()
	assert.Equal(t, PreMatch, arena.MatchState)

	// Test that timeout can't be started during a match.
	arena.AllianceStations["R1"].Bypass = true
	arena.AllianceStations["R2"].Bypass = true
	arena.AllianceStations["B1"].Bypass = true
	arena.AllianceStations["B2"].Bypass = true
	assert.Nil(t, arena.StartMatch())
	arena.Update()
	assert.NotNil(t, arena.StartTimeout("Timeout", 1))
	assert.NotEqual(t, TimeoutActive, arena.MatchState)
	assert.Equal(t, timeoutDurationSec, game.MatchTiming.TimeoutDurationSec)
	arena.MatchStartTime = time.Now().Add(
		-time.Duration(
			game.MatchTiming.AutoDurationSec+game.MatchTiming.PauseDurationSec+game.MatchTiming.TeleopDurationSec,
		) * time.Second,
	)
	for arena.MatchState != PostMatch {
		arena.Update()
		assert.NotNil(t, arena.StartTimeout("Timeout", 1))
	}

	// Test that a match can be loaded during a timeout.
	assert.Nil(t, arena.ResetMatch())
	assert.Nil(t, arena.LoadTestMatch())
	assert.Nil(t, arena.StartTimeout("Break 2", 10))
	assert.Equal(t, TimeoutActive, arena.MatchState)
	match := model.Match{
		Type: model.Playoff, ShortName: "F1", Red1: 1, Red2: 2, Blue1: 3, Blue2: 4,
	}
	assert.Nil(t, arena.Database.CreateMatch(&match))
	assert.Nil(t, arena.LoadMatch(&match))
	assert.Equal(t, TimeoutActive, arena.MatchState)
	assert.Equal(t, match, *arena.CurrentMatch)
}

func TestSaveTeamHasConnected(t *testing.T) {
	arena := setupTestArena(t)

	arena.Database.CreateTeam(&model.Team{Id: 101})
	arena.Database.CreateTeam(&model.Team{Id: 102})
	arena.Database.CreateTeam(&model.Team{Id: 103})
	arena.Database.CreateTeam(&model.Team{Id: 104, City: "San Jose", HasConnected: true})
	match := model.Match{Red1: 101, Red2: 102, Blue1: 103, Blue2: 104}
	arena.Database.CreateMatch(&match)
	arena.LoadMatch(&match)

	// Only the stations that were staged count as having taken the field.
	arena.AllianceStations["R1"].Bypass = true
	assert.Nil(t, arena.SetStationReady("R2", true))
	arena.AllianceStations["B1"].Bypass = true
	assert.Nil(t, arena.SetStationReady("B2", true))
	arena.AllianceStations["B2"].Team.City = "Sand Hosay" // Change some other field to verify that it isn't saved.
	assert.Nil(t, arena.StartMatch())

	teams, _ := arena.Database.GetAllTeams()
	if assert.Equal(t, 4, len(teams)) {
		assert.False(t, teams[0].HasConnected)
		assert.True(t, teams[1].HasConnected)
		assert.False(t, teams[2].HasConnected)
		assert.True(t, teams[3].HasConnected)
		assert.Equal(t, "San Jose", teams[3].City)
	}
}

func TestSignalVolunteers(t *testing.T) {
	arena := setupTestArena(t)

	// Test that SignalVolunteers only works in PreMatch and PostMatch states.
	for _, state := range []MatchState{StartMatch, AutoPeriod, PausePeriod, TeleopPeriod, TimeoutActive, PostTimeout} {
		arena.MatchState = state
		arena.FieldVolunteers = false
		arena.SignalVolunteers()
		assert.False(t, arena.FieldVolunteers)
		assert.NotEqual(t, "signalCount", arena.AllianceStationDisplayMode)
	}

	// Test SignalVolunteers in PreMatch state.
	arena.MatchState = PreMatch
	arena.FieldReset = true
	arena.AllianceStationDisplayMode = "match"
	arena.SignalVolunteers()
	assert.True(t, arena.FieldVolunteers)
	assert.False(t, arena.FieldReset)
	assert.Equal(t, "signalCount", arena.AllianceStationDisplayMode)

	// Test SignalVolunteers in PostMatch state.
	arena.MatchState = PostMatch
	arena.FieldVolunteers = false
	arena.FieldReset = false
	arena.AllianceStationDisplayMode = "match"
	arena.SignalVolunteers()
	assert.True(t, arena.FieldVolunteers)
	assert.False(t, arena.FieldReset)
	assert.Equal(t, "signalCount", arena.AllianceStationDisplayMode)
}

func TestSignalReset(t *testing.T) {
	arena := setupTestArena(t)

	// Test that SignalReset only works in PreMatch and PostMatch states.
	for _, state := range []MatchState{StartMatch, AutoPeriod, PausePeriod, TeleopPeriod, TimeoutActive, PostTimeout} {
		arena.MatchState = state
		arena.FieldReset = false
		arena.FieldVolunteers = false
		arena.AllianceStationDisplayMode = "match"
		arena.SignalReset()
		assert.False(t, arena.FieldReset)
		assert.False(t, arena.FieldVolunteers)
		assert.NotEqual(t, "fieldReset", arena.AllianceStationDisplayMode)
	}

	// Test SignalReset in PreMatch state.
	arena.MatchState = PreMatch
	arena.FieldReset = false
	arena.FieldVolunteers = true
	arena.AllianceStationDisplayMode = "match"
	arena.SignalReset()
	assert.False(t, arena.FieldVolunteers)
	assert.True(t, arena.FieldReset)
	assert.Equal(t, "fieldReset", arena.AllianceStationDisplayMode)

	// Test SignalReset in PostMatch state.
	arena.MatchState = PostMatch
	arena.FieldReset = false
	arena.FieldVolunteers = true
	arena.AllianceStationDisplayMode = "match"
	arena.SignalReset()
	assert.False(t, arena.FieldVolunteers)
	assert.True(t, arena.FieldReset)
	assert.Equal(t, "fieldReset", arena.AllianceStationDisplayMode)
}
