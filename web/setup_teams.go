// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web routes for configuring the team list.

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/model"
	"net/http"
	"strconv"
	"strings"
)

// Global var to hold the team import progress percentage.
var progressPercentage float64 = 5

// Shows the team list.
func (web *Web) teamsGetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	web.renderTeams(w, r, false)
}

// Adds teams to the team list.
func (web *Web) teamsPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	if !web.canModifyTeamList() {
		web.renderTeams(w, r, true)
		return
	}

	var teamNumbers []int
	for _, teamNumberString := range strings.Split(r.PostFormValue("teamNumbers"), "\r\n") {
		teamNumber, err := strconv.Atoi(teamNumberString)
		if err == nil {
			teamNumbers = append(teamNumbers, teamNumber)
		}
	}

	progressPercentage = 5
	progressIncrement := 95.0 / float64(len(teamNumbers))
	for _, teamNumber := range teamNumbers {
		team := model.Team{Id: teamNumber}
		if err := web.arena.Database.CreateTeam(&team); err != nil {
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
		web.renderTeams(w, r, true)
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

	teamId, _ := strconv.Atoi(r.PathValue("id"))
	team, err := web.arena.Database.GetTeamById(teamId)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if team == nil {
		http.Error(w, fmt.Sprintf("Error: No such team: %d", teamId), 400)
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

	teamId, _ := strconv.Atoi(r.PathValue("id"))
	team, err := web.arena.Database.GetTeamById(teamId)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if team == nil {
		http.Error(w, fmt.Sprintf("Error: No such team: %d", teamId), 400)
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
		web.renderTeams(w, r, true)
		return
	}

	teamId, _ := strconv.Atoi(r.PathValue("id"))
	team, err := web.arena.Database.GetTeamById(teamId)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if team == nil {
		http.Error(w, fmt.Sprintf("Error: No such team: %d", teamId), 400)
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

func (web *Web) renderTeams(w http.ResponseWriter, r *http.Request, showErrorMessage bool) {
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
		Teams            []model.Team
		ShowErrorMessage bool
	}{web.arena.EventSettings, teams, showErrorMessage}
	err = template.ExecuteTemplate(w, "base", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Returns true if it is safe to change the team list (i.e. no matches/results exist yet).
func (web *Web) canModifyTeamList() bool {
	matches, err := web.arena.Database.GetMatchesByType(model.Qualification, true)
	if err != nil || len(matches) > 0 {
		return false
	}
	return true
}
