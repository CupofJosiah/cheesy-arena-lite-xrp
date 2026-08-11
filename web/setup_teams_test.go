// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetupTeams(t *testing.T) {
	web := setupTestWeb(t)

	// Check that there are no teams to start.
	recorder := web.getHttpResponse("/setup/teams")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "0 teams")

	// Add some teams. Non-numeric lines are ignored.
	recorder = web.postHttpResponse("/setup/teams", "teamNumbers=254\r\nnotateam\r\n1114\r\n")
	assert.Equal(t, 303, recorder.Code)
	recorder = web.getHttpResponse("/setup/teams")
	assert.Contains(t, recorder.Body.String(), "2 teams")
	assert.Contains(t, recorder.Body.String(), "254")
	assert.Contains(t, recorder.Body.String(), "1114")

	// Add another team.
	recorder = web.postHttpResponse("/setup/teams", "teamNumbers=33")
	assert.Equal(t, 303, recorder.Code)
	recorder = web.getHttpResponse("/setup/teams")
	assert.Contains(t, recorder.Body.String(), "3 teams")
	assert.Contains(t, recorder.Body.String(), "33")

	// Edit a team.
	recorder = web.getHttpResponse("/setup/teams/254/edit")
	assert.Equal(t, 200, recorder.Code)
	recorder = web.postHttpResponse("/setup/teams/254/edit", "nickname=Teh Chezy Pofs&schoolName=Bellarmine")
	assert.Equal(t, 303, recorder.Code)
	recorder = web.getHttpResponse("/setup/teams")
	assert.Contains(t, recorder.Body.String(), "Teh Chezy Pofs")
	assert.Contains(t, recorder.Body.String(), "Bellarmine")

	// Delete a team.
	recorder = web.postHttpResponse("/setup/teams/1114/delete", "")
	assert.Equal(t, 303, recorder.Code)
	recorder = web.getHttpResponse("/setup/teams")
	assert.Contains(t, recorder.Body.String(), "2 teams")

	// Test clearing all teams.
	recorder = web.postHttpResponse("/setup/teams/clear", "")
	assert.Equal(t, 303, recorder.Code)
	recorder = web.getHttpResponse("/setup/teams")
	assert.Contains(t, recorder.Body.String(), "0 teams")
}

func TestSetupTeamsDisallowModification(t *testing.T) {
	web := setupTestWeb(t)

	web.arena.Database.CreateTeam(&model.Team{Id: 254, Nickname: "The Cheesy Poofs"})
	web.arena.Database.CreateMatch(&model.Match{Type: model.Qualification})

	// Disallow adding teams.
	recorder := web.postHttpResponse("/setup/teams", "teamNumbers=33")
	assert.Contains(t, recorder.Body.String(), "can't modify")
	assert.Contains(t, recorder.Body.String(), "1 teams")
	assert.Contains(t, recorder.Body.String(), "The Cheesy Poofs")

	// Disallow deleting team.
	recorder = web.postHttpResponse("/setup/teams/254/delete", "")
	assert.Contains(t, recorder.Body.String(), "can't modify")
	assert.Contains(t, recorder.Body.String(), "1 teams")
	assert.Contains(t, recorder.Body.String(), "The Cheesy Poofs")

	// Disallow clearing all teams.
	recorder = web.postHttpResponse("/setup/teams/clear", "")
	assert.Contains(t, recorder.Body.String(), "can't modify")
	assert.Contains(t, recorder.Body.String(), "1 teams")
	assert.Contains(t, recorder.Body.String(), "The Cheesy Poofs")

	// Allow editing a team.
	recorder = web.postHttpResponse("/setup/teams/254/edit", "nickname=Teh Chezy Pofs")
	assert.Equal(t, 303, recorder.Code)
	recorder = web.getHttpResponse("/setup/teams")
	assert.NotContains(t, recorder.Body.String(), "can't modify")
	assert.Contains(t, recorder.Body.String(), "1 teams")
	assert.Contains(t, recorder.Body.String(), "Teh Chezy Pofs")
}

func TestSetupTeamsBadReqest(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/setup/teams/254/edit")
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "No such team")
	recorder = web.postHttpResponse("/setup/teams/254/edit", "")
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "No such team")
	recorder = web.postHttpResponse("/setup/teams/254/delete", "")
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "No such team")
}

func TestSetupTeamsProgress(t *testing.T) {
	web := setupTestWeb(t)
	progressPercentage = 25.4

	recorder := web.getHttpResponse("/setup/teams/progress")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "25", recorder.Body.String())
}
