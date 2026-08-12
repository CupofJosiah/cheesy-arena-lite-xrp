// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web routes for configuring the team list.

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"net/http"
	"strconv"
	"strings"
)

// The message shown when the team list can no longer be edited.
const teamListLockedMessage = "You can't modify the team list once the qualification schedule has been " +
	"generated. If you need to change the team list, clear all other data first on the Settings page."

// Global var to hold the team import progress percentage.
var progressPercentage float64 = 5

// Shows the team list.
func (web *Web) teamsGetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	web.renderTeams(w, r, "")
}

// Adds teams to the team list.
func (web *Web) teamsPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	if !web.canModifyTeamList() {
		web.renderTeams(w, r, teamListLockedMessage)
		return
	}

	// Parse and validate the whole list before creating anything, so that a typo doesn't leave a half-imported team
	// list behind.
	var teamIds []game.TeamId
	seenTeamIds := make(map[game.TeamId]struct{})
	for _, teamIdString := range strings.Split(r.PostFormValue("teamNumbers"), "\n") {
		if strings.TrimSpace(teamIdString) == "" {
			continue
		}
		teamId, err := game.ParseTeamId(teamIdString)
		if err != nil {
			progressPercentage = 100
			web.renderTeams(w, r, capitalizeFirst(err.Error())+".")
			return
		}
		if _, ok := seenTeamIds[teamId]; ok {
			progressPercentage = 100
			web.renderTeams(w, r, fmt.Sprintf("Team %s appears more than once in the list.", teamId))
			return
		}
		existingTeam, err := web.arena.Database.GetTeamById(teamId)
		if err != nil {
			handleWebErr(w, err)
			return
		}
		if existingTeam != nil {
			progressPercentage = 100
			web.renderTeams(w, r, fmt.Sprintf("Team %s is already in the team list.", teamId))
			return
		}
		seenTeamIds[teamId] = struct{}{}
		teamIds = append(teamIds, teamId)
	}

	progressPercentage = 5
	progressIncrement := 95.0 / float64(len(teamIds))
	for _, teamId := range teamIds {
		team := model.Team{Id: teamId}
		if err := web.arena.Database.CreateTeam(&team); err != nil {
			progressPercentage = 100
			handleWebErr(w, err)
			return
		}

		progressPercentage += progressIncrement
	}
	progressPercentage = 100

	http.Redirect(w, r, "/setup/teams", 303)
}

// Clears the team list.
func (web *Web) teamsClearHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	if !web.canModifyTeamList() {
		web.renderTeams(w, r, teamListLockedMessage)
		return
	}

	err := web.arena.Database.TruncateTeams()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	http.Redirect(w, r, "/setup/teams", 303)
}

// Shows the page to edit a team's fields.
func (web *Web) teamEditGetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	teamId := game.TeamIdFromString(r.PathValue("id"))
	team, err := web.arena.Database.GetTeamById(teamId)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if team == nil {
		http.Error(w, fmt.Sprintf("Error: No such team: %s", teamId), 400)
		return
	}

	template, err := web.parseFiles("templates/edit_team.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
		*model.Team
	}{web.arena.EventSettings, team}
	err = template.ExecuteTemplate(w, "base", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Updates a team's fields.
func (web *Web) teamEditPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	teamId := game.TeamIdFromString(r.PathValue("id"))
	team, err := web.arena.Database.GetTeamById(teamId)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if team == nil {
		http.Error(w, fmt.Sprintf("Error: No such team: %s", teamId), 400)
		return
	}

	team.Name = r.PostFormValue("name")
	team.Nickname = r.PostFormValue("nickname")
	team.City = r.PostFormValue("city")
	team.SchoolName = r.PostFormValue("schoolName")
	team.StateProv = r.PostFormValue("stateProv")
	team.Country = r.PostFormValue("country")
	team.RookieYear, _ = strconv.Atoi(r.PostFormValue("rookieYear"))
	team.RobotName = r.PostFormValue("robotName")
	team.Accomplishments = r.PostFormValue("accomplishments")
	team.HasConnected = r.PostFormValue("hasConnected") == "on"
	err = web.arena.Database.UpdateTeam(team)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	http.Redirect(w, r, "/setup/teams", 303)
}

// Removes a team from the team list.
func (web *Web) teamDeletePostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	if !web.canModifyTeamList() {
		web.renderTeams(w, r, teamListLockedMessage)
		return
	}

	teamId := game.TeamIdFromString(r.PathValue("id"))
	team, err := web.arena.Database.GetTeamById(teamId)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if team == nil {
		http.Error(w, fmt.Sprintf("Error: No such team: %s", teamId), 400)
		return
	}
	err = web.arena.Database.DeleteTeam(team.Id)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	http.Redirect(w, r, "/setup/teams", 303)
}

// Returns the current team import progress.
func (web *Web) teamsUpdateProgressBarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write([]byte(fmt.Sprintf("%.0f", progressPercentage))); err != nil {
		handleWebErr(w, err)
		return
	}
}

func (web *Web) renderTeams(w http.ResponseWriter, r *http.Request, errorMessage string) {
	teams, err := web.arena.Database.GetAllTeams()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	template, err := web.parseFiles("templates/setup_teams.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
		Teams        []model.Team
		ErrorMessage string
	}{web.arena.EventSettings, teams, errorMessage}
	err = template.ExecuteTemplate(w, "base", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Returns the given string with its first letter capitalized, for rendering lowercase error text as a sentence.
func capitalizeFirst(text string) string {
	if text == "" {
		return text
	}
	return strings.ToUpper(text[0:1]) + text[1:]
}

// Returns true if it is safe to change the team list (i.e. no matches/results exist yet).
func (web *Web) canModifyTeamList() bool {
	matches, err := web.arena.Database.GetMatchesByType(model.Qualification, true)
	if err != nil || len(matches) > 0 {
		return false
	}
	return true
}
