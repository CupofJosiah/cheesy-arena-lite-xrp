// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package model

import (
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEventSettingsReadWrite(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	eventSettings, err := db.GetEventSettings()
	assert.Nil(t, err)
	assert.Equal(t, 1, eventSettings.Id)
	assert.Equal(t, "Untitled Event", eventSettings.Name)
	assert.Equal(t, DoubleEliminationPlayoff, eventSettings.PlayoffType)
	assert.Equal(t, 8, eventSettings.NumPlayoffAlliances)
	assert.Equal(t, "L", eventSettings.SelectionRound2Order)
	assert.Equal(t, "", eventSettings.SelectionRound3Order)
	assert.True(t, eventSettings.SelectionShowUnpickedTeams)
	assert.True(t, eventSettings.TbaDownloadEnabled)
	assert.False(t, eventSettings.TbaPublishingEnabled)
	assert.Equal(t, 36, eventSettings.ApChannel)
	assert.Equal(t, "configure terminal\ninterface range gigabitEthernet 1/2-4\nno shutdown\nexit\nexit\nexit", eventSettings.SCCUpCommands)
	assert.Equal(t, "configure terminal\ninterface range gigabitEthernet 1/2-4\nshutdown\nexit\nexit\nexit", eventSettings.SCCDownCommands)
	assert.Equal(t, game.MatchTiming.AutoDurationSec, eventSettings.AutoDurationSec)
	assert.Equal(t, game.MatchTiming.PauseDurationSec, eventSettings.PauseDurationSec)
	assert.Equal(t, game.MatchTiming.TeleopDurationSec, eventSettings.TeleopDurationSec)
	assert.Equal(t, game.MatchTiming.WarningSoundTimeSec, eventSettings.WarningSoundTimeSec)
	assert.Equal(t, "", eventSettings.CompanionAddress)
	assert.Equal(t, 0, eventSettings.CompanionPort)

	eventSettings.Name = "Chezy Champs"
	eventSettings.NumPlayoffAlliances = 6
	eventSettings.SelectionRound2Order = "F"
	eventSettings.SelectionRound3Order = "L"
	eventSettings.AutoDurationSec = 15
	eventSettings.PauseDurationSec = 5
	eventSettings.TeleopDurationSec = 120
	eventSettings.WarningSoundTimeSec = 20
	err = db.UpdateEventSettings(eventSettings)
	assert.Nil(t, err)
	eventSettings2, err := db.GetEventSettings()
	assert.Nil(t, err)
	assert.Equal(t, eventSettings, eventSettings2)
}
