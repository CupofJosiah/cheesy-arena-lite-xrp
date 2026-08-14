// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web routes for generating practice and qualification schedules.

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/tournament"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// Global vars to hold schedules that are in the process of being generated.
var cachedMatches = make(map[model.MatchType][]model.Match)
var cachedScheduledTeams = make(map[model.MatchType][]ScheduledTeam)

// Summarizes where a team lands in a schedule that has been generated but not yet saved, to give the operator enough
// information to decide whether to exchange it with another team before committing the schedule.
type ScheduledTeam struct {
	Id         game.TeamId
	FirstMatch string
	NumMatches int
}

// Shows the schedule editing page.
func (web *Web) scheduleGetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	matchTypeString := getMatchType(r)
	matchType, _ := model.MatchTypeFromString(matchTypeString)
	if matchType != model.Practice && matchType != model.Qualification {
		http.Redirect(w, r, "/setup/schedule?matchType=practice", 302)
		return
	}

	web.renderSchedule(w, r, "")
}

// Generates the schedule, presents it for review without saving it, and saves the schedule blocks to the database.
func (web *Web) scheduleGeneratePostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	matchTypeString := getMatchType(r)
	matchType, err := model.MatchTypeFromString(matchTypeString)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	scheduleBlocks, err := getScheduleBlocks(r)
	// Save blocks even if there is an error, so that any good ones are not discarded.
	deleteBlocksErr := web.arena.Database.DeleteScheduleBlocksByMatchType(matchType)
	if deleteBlocksErr != nil {
		handleWebErr(w, err)
		return
	}
	for _, block := range scheduleBlocks {
		block.MatchType = matchType
		createBlockErr := web.arena.Database.CreateScheduleBlock(&block)
		if createBlockErr != nil {
			handleWebErr(w, err)
			return
		}
	}
	if err != nil {
		web.renderSchedule(w, r, "Incomplete or invalid schedule block parameters specified.")
		return
	}

	// Build the schedule.
	teams, err := web.arena.Database.GetAllTeams()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if len(teams) == 0 {
		web.renderSchedule(
			w,
			r,
			"No team list is configured. Set up the list of teams at the event before generating the schedule.",
		)
		return
	}
	if len(teams) < 6 {
		web.renderSchedule(
			w,
			r,
			fmt.Sprintf("There are only %d teams. There must be at least 6 teams to generate a schedule.", len(teams)),
		)
		return
	}

	matches, err := tournament.BuildRandomSchedule(teams, scheduleBlocks, matchType)
	if err != nil {
		web.renderSchedule(w, r, fmt.Sprintf("Error generating schedule: %s.", err.Error()))
		return
	}
	cachedMatches[matchType] = matches
	cachedScheduledTeams[matchType] = buildScheduledTeams(matches)

	http.Redirect(w, r, "/setup/schedule?matchType="+matchTypeString, 303)
}

// Exchanges two teams throughout the generated but not yet saved schedule.
func (web *Web) scheduleSwapPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	matchTypeString := getMatchType(r)
	matchType, err := model.MatchTypeFromString(matchTypeString)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	matches := cachedMatches[matchType]
	if len(matches) == 0 {
		web.renderSchedule(w, r, "Can't swap teams because no schedule has been generated. Generate one first.")
		return
	}

	team1 := game.TeamIdFromString(r.PostFormValue("team1"))
	team2 := game.TeamIdFromString(r.PostFormValue("team2"))
	if team1 == team2 {
		web.renderSchedule(w, r, "Select two different teams to swap.")
		return
	}
	// Both teams must already appear in the schedule. Otherwise the swap would drop a team out of it rather than
	// exchange two, which can happen if the team list was edited after the schedule was generated.
	for _, teamId := range []game.TeamId{team1, team2} {
		if !isTeamScheduled(matches, teamId) {
			web.renderSchedule(
				w,
				r,
				fmt.Sprintf(
					"Team %q is not in the generated schedule. Regenerate the schedule to pick up team list changes.",
					teamId,
				),
			)
			return
		}
	}

	// Exchanging the two teams everywhere they appear is equivalent to exchanging their positions in the underlying
	// schedule template, because the surrogate flags belong to the match slots rather than to the teams filling them.
	// That is what keeps the template's balance of partners, opponents and match spacing intact; every property the
	// schedule had before the swap it still has afterwards, with the two teams' schedules traded whole.
	swapTeams := func(teamId *game.TeamId) {
		if *teamId == team1 {
			*teamId = team2
		} else if *teamId == team2 {
			*teamId = team1
		}
	}
	for i := range matches {
		swapTeams(&matches[i].Red1)
		swapTeams(&matches[i].Red2)
		swapTeams(&matches[i].Blue1)
		swapTeams(&matches[i].Blue2)
	}
	cachedScheduledTeams[matchType] = buildScheduledTeams(matches)

	http.Redirect(w, r, "/setup/schedule?matchType="+matchTypeString, 303)
}

// Saves the generated schedule to the database.
func (web *Web) scheduleSavePostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	matchTypeString := getMatchType(r)
	matchType, err := model.MatchTypeFromString(matchTypeString)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	existingMatches, err := web.arena.Database.GetMatchesByType(matchType, true)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	if len(existingMatches) > 0 {
		web.renderSchedule(
			w,
			r,
			fmt.Sprintf(
				"Can't save schedule because a schedule of %d %s matches already exists. Clear it first on the "+
					"Settings page.",
				len(existingMatches),
				matchType,
			),
		)
		return
	}

	for _, match := range cachedMatches[matchType] {
		err = web.arena.Database.CreateMatch(&match)
		if err != nil {
			handleWebErr(w, err)
			return
		}
	}

	// Back up the database.
	err = web.arena.Database.Backup(web.arena.EventSettings.Name, "post_scheduling")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	http.Redirect(w, r, "/setup/schedule?matchType="+matchTypeString, 303)
}

func (web *Web) renderSchedule(w http.ResponseWriter, r *http.Request, errorMessage string) {
	matchTypeString := getMatchType(r)
	matchType, err := model.MatchTypeFromString(matchTypeString)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	scheduleBlocks, err := web.arena.Database.GetScheduleBlocksByMatchType(matchType)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	teams, err := web.arena.Database.GetAllTeams()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	template, err := web.parseFiles("templates/setup_schedule.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
		MatchType      model.MatchType
		ScheduleBlocks []model.ScheduleBlock
		NumTeams       int
		TeamsPerMatch  int
		Matches        []model.Match
		ScheduledTeams []ScheduledTeam
		ErrorMessage   string
	}{
		web.arena.EventSettings,
		matchType,
		scheduleBlocks,
		len(teams),
		game.TeamsPerMatch,
		cachedMatches[matchType],
		cachedScheduledTeams[matchType],
		errorMessage,
	}
	err = template.ExecuteTemplate(w, "base", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Converts the post form variables into a slice of schedule blocks.
func getScheduleBlocks(r *http.Request) ([]model.ScheduleBlock, error) {
	numScheduleBlocks, err := strconv.Atoi(r.PostFormValue("numScheduleBlocks"))
	if err != nil {
		return []model.ScheduleBlock{}, err
	}
	var returnErr error
	scheduleBlocks := make([]model.ScheduleBlock, numScheduleBlocks)
	location, _ := time.LoadLocation("Local")
	for i := 0; i < numScheduleBlocks; i++ {
		scheduleBlocks[i].StartTime, err = time.ParseInLocation(
			"2006-01-02 03:04:05 PM", r.PostFormValue(fmt.Sprintf("startTime%d", i)), location,
		)
		if err != nil {
			returnErr = err
		}
		scheduleBlocks[i].NumMatches, err = strconv.Atoi(r.PostFormValue(fmt.Sprintf("numMatches%d", i)))
		if err != nil {
			returnErr = err
		}
		scheduleBlocks[i].MatchSpacingSec, err = strconv.Atoi(r.PostFormValue(fmt.Sprintf("matchSpacingSec%d", i)))
		if err != nil {
			returnErr = err
		}
	}
	return scheduleBlocks, returnErr
}

// Returns the teams appearing in the given matches, sorted by team ID, summarizing where each one lands. Surrogate
// appearances are counted, since they are still matches the team has to show up for.
func buildScheduledTeams(matches []model.Match) []ScheduledTeam {
	teamsById := make(map[game.TeamId]*ScheduledTeam)
	for _, match := range matches {
		for _, teamId := range match.TeamIds() {
			if scheduledTeam, ok := teamsById[teamId]; ok {
				scheduledTeam.NumMatches++
			} else {
				teamsById[teamId] = &ScheduledTeam{Id: teamId, FirstMatch: match.ShortName, NumMatches: 1}
			}
		}
	}

	scheduledTeams := make([]ScheduledTeam, 0, len(teamsById))
	for _, scheduledTeam := range teamsById {
		scheduledTeams = append(scheduledTeams, *scheduledTeam)
	}
	sort.Slice(
		scheduledTeams, func(i, j int) bool {
			return game.LessTeamId(scheduledTeams[i].Id, scheduledTeams[j].Id)
		},
	)
	return scheduledTeams
}

// Returns true if the given team appears anywhere in the given matches.
func isTeamScheduled(matches []model.Match, teamId game.TeamId) bool {
	for _, match := range matches {
		for _, matchTeamId := range match.TeamIds() {
			if matchTeamId == teamId {
				return true
			}
		}
	}
	return false
}

func getMatchType(r *http.Request) string {
	if matchType, ok := r.URL.Query()["matchType"]; ok {
		return matchType[0]
	}
	return r.PostFormValue("matchType")
}
