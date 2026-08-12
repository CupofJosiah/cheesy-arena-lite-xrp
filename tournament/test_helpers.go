// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Helper methods for use in tests in this package and others.

package tournament

import (
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"strconv"
	"testing"
)

func CreateTestAlliances(database *model.Database, allianceCount int) {
	for i := 1; i <= allianceCount; i++ {
		alliance := model.Alliance{
			Id:      i,
			TeamIds: []game.TeamId{teamId(100*i + 1), teamId(100*i + 2), teamId(100*i + 3), teamId(100*i + 4)},
			Lineup:  [game.TeamsPerAlliance]game.TeamId{teamId(100*i + 2), teamId(100*i + 1)},
		}
		database.CreateAlliance(&alliance)
	}
}

func teamId(number int) game.TeamId {
	return game.TeamId(strconv.Itoa(number))
}

func setupTestDb(t *testing.T) *model.Database {
	return model.SetupTestDb(t)
}
