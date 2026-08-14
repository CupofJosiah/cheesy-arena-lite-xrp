// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSetupSchedule(t *testing.T) {
	web := setupTestWeb(t)

	for i := 0; i < 12; i++ {
		web.arena.Database.CreateTeam(&model.Team{Id: teamId(i + 101)})
	}
	web.arena.Database.CreateMatch(&model.Match{Type: model.Practice, ShortName: "P1"})

	// Check the default setting values.
	recorder := web.getHttpResponse("/setup/schedule?matchType=practice")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "addBlock();")
	// The page's match count estimates are computed in JavaScript, so the team count per match has to be handed to it
	// rather than duplicated as a literal there.
	assert.Contains(t, recorder.Body.String(), fmt.Sprintf("var teamsPerMatch = %d;", game.TeamsPerMatch))
	// No schedule has been generated, so there is nothing to swap.
	assert.NotContains(t, recorder.Body.String(), "Swap Teams")

	// Submit a schedule for generation. 30 matches over 12 teams works out to 10 matches per team.
	postData := "numScheduleBlocks=3&startTime0=2014-01-01 09:00:00 AM&numMatches0=7&matchSpacingSec0=480&" +
		"startTime1=2014-01-02 09:56:00 AM&numMatches1=8&matchSpacingSec1=420&startTime2=2014-01-03 01:00:00 PM&" +
		"numMatches2=15&matchSpacingSec2=360&matchType=qualification"
	recorder = web.postHttpResponse("/setup/schedule/generate", postData)
	assert.Equal(t, 303, recorder.Code)
	recorder = web.getHttpResponse("/setup/schedule?matchType=qualification")
	assert.Contains(t, recorder.Body.String(), "2014-01-01 09:48:00") // Last match of first block.
	assert.Contains(t, recorder.Body.String(), "2014-01-02 10:45:00") // Last match of second block.
	assert.Contains(t, recorder.Body.String(), "2014-01-03 14:24:00") // Last match of third block.

	// Save schedule and check that it was persisted.
	recorder = web.postHttpResponse("/setup/schedule/save?matchType=qualification", "")
	assert.Equal(t, 303, recorder.Code)
	matches, err := web.arena.Database.GetMatchesByType(model.Qualification, true)
	assert.Nil(t, err)
	if !assert.Equal(t, 30, len(matches)) {
		return
	}
	location, _ := time.LoadLocation("Local")
	assert.Equal(t, time.Date(2014, 1, 1, 9, 0, 0, 0, location).Unix(), matches[0].Time.Unix())
	assert.Equal(t, time.Date(2014, 1, 2, 9, 56, 0, 0, location).Unix(), matches[7].Time.Unix())
	assert.Equal(t, time.Date(2014, 1, 3, 13, 0, 0, 0, location).Unix(), matches[15].Time.Unix())
}

// Verifies that a schedule can be generated and reported on for teams whose numbers contain letters.
func TestSetupScheduleWithLetteredTeamNumbers(t *testing.T) {
	web := setupTestWeb(t)

	var expectedTeamIds []game.TeamId
	for i := 0; i < 12; i++ {
		// Alternate between "1A", "1B", "2A", ... so that both halves of the ID vary.
		id := game.TeamId(fmt.Sprintf("%d%c", i/2+1, 'A'+i%2))
		expectedTeamIds = append(expectedTeamIds, id)
		assert.Nil(t, web.arena.Database.CreateTeam(&model.Team{Id: id}))
	}

	postData := "numScheduleBlocks=1&startTime0=2014-01-01 09:00:00 AM&numMatches0=30&matchSpacingSec0=360&" +
		"matchType=qualification"
	recorder := web.postHttpResponse("/setup/schedule/generate", postData)
	assert.Equal(t, 303, recorder.Code)
	recorder = web.postHttpResponse("/setup/schedule/save?matchType=qualification", "")
	assert.Equal(t, 303, recorder.Code)

	matches, err := web.arena.Database.GetMatchesByType(model.Qualification, true)
	assert.Nil(t, err)
	if !assert.Equal(t, 30, len(matches)) {
		return
	}

	// Every scheduled team must be one of the lettered teams that were created.
	scheduled := make(map[game.TeamId]int)
	for _, match := range matches {
		for _, id := range match.TeamIds() {
			assert.Contains(t, expectedTeamIds, id)
			scheduled[id]++
		}
	}
	assert.Equal(t, 12, len(scheduled))

	// The schedule reports must render the lettered numbers.
	recorder = web.getHttpResponse("/reports/csv/schedule/qualification")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "6B")
	recorder = web.getHttpResponse("/reports/pdf/schedule/qualification")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/pdf", recorder.Header().Get("Content-Type"))
}

// Returns the surrogate flags of a match in the same order as Match.TeamIds().
func surrogateFlags(match *model.Match) [game.TeamsPerMatch]bool {
	return [game.TeamsPerMatch]bool{
		match.Red1IsSurrogate, match.Red2IsSurrogate, match.Blue1IsSurrogate, match.Blue2IsSurrogate,
	}
}

// Verifies that two teams can be exchanged throughout a generated schedule before it is saved.
func TestSetupScheduleSwapTeams(t *testing.T) {
	web := setupTestWeb(t)

	for i := 0; i < 12; i++ {
		web.arena.Database.CreateTeam(&model.Team{Id: teamId(i + 101)})
	}

	postData := "numScheduleBlocks=1&startTime0=2014-01-01 09:00:00 AM&numMatches0=30&matchSpacingSec0=360&" +
		"matchType=qualification"
	recorder := web.postHttpResponse("/setup/schedule/generate", postData)
	assert.Equal(t, 303, recorder.Code)

	// Snapshot the assignment so that the swapped one can be compared against it slot by slot.
	before := make([][game.TeamsPerMatch]game.TeamId, len(cachedMatches[model.Qualification]))
	surrogatesBefore := make([][game.TeamsPerMatch]bool, len(cachedMatches[model.Qualification]))
	expectedSwaps := 0
	for i, match := range cachedMatches[model.Qualification] {
		before[i] = match.TeamIds()
		surrogatesBefore[i] = surrogateFlags(&match)
		for _, id := range before[i] {
			if id == "101" || id == "112" {
				expectedSwaps++
			}
		}
	}
	if !assert.Equal(t, 30, len(before)) {
		return
	}
	assert.NotZero(t, expectedSwaps)

	recorder = web.postHttpResponse("/setup/schedule/swap?matchType=qualification", "team1=101&team2=112")
	assert.Equal(t, 303, recorder.Code)

	// Every slot must hold exactly what it held before, except that the two teams have traded places.
	swaps := 0
	for i, match := range cachedMatches[model.Qualification] {
		for j, id := range match.TeamIds() {
			switch before[i][j] {
			case "101":
				assert.Equal(t, game.TeamId("112"), id)
				swaps++
			case "112":
				assert.Equal(t, game.TeamId("101"), id)
				swaps++
			default:
				assert.Equal(t, before[i][j], id)
			}
		}
	}
	assert.Equal(t, expectedSwaps, swaps)

	// The surrogate flags belong to the match slots rather than to the teams filling them, so they must be exactly
	// where they were. This is the invariant that makes a swap equivalent to reseeding the schedule template.
	for i, match := range cachedMatches[model.Qualification] {
		assert.Equal(t, surrogatesBefore[i], surrogateFlags(&match))
	}

	// Swapping the same pair back must restore the original assignment exactly.
	recorder = web.postHttpResponse("/setup/schedule/swap?matchType=qualification", "team1=112&team2=101")
	assert.Equal(t, 303, recorder.Code)
	for i, match := range cachedMatches[model.Qualification] {
		assert.Equal(t, before[i], match.TeamIds())
	}

	// The swap interface and the review table must both be rendered once a schedule exists.
	recorder = web.getHttpResponse("/setup/schedule?matchType=qualification")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Swap Teams")
	assert.Contains(t, recorder.Body.String(), `action="/setup/schedule/swap?matchType=Qualification"`)

	// The swap must survive being saved, since it is the cached schedule that gets persisted.
	recorder = web.postHttpResponse("/setup/schedule/swap?matchType=qualification", "team1=101&team2=112")
	assert.Equal(t, 303, recorder.Code)
	recorder = web.postHttpResponse("/setup/schedule/save?matchType=qualification", "")
	assert.Equal(t, 303, recorder.Code)
	matches, err := web.arena.Database.GetMatchesByType(model.Qualification, true)
	assert.Nil(t, err)
	if !assert.Equal(t, 30, len(matches)) {
		return
	}
	for i, match := range matches {
		assert.Equal(t, cachedMatches[model.Qualification][i].TeamIds(), match.TeamIds())
	}
}

func TestSetupScheduleSwapTeamsErrors(t *testing.T) {
	web := setupTestWeb(t)

	for i := 0; i < 12; i++ {
		web.arena.Database.CreateTeam(&model.Team{Id: teamId(i + 101)})
	}

	// No schedule has been generated yet.
	recorder := web.postHttpResponse("/setup/schedule/swap?matchType=qualification", "team1=101&team2=102")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "no schedule has been generated")

	postData := "numScheduleBlocks=1&startTime0=2014-01-01 09:00:00 AM&numMatches0=30&matchSpacingSec0=360&" +
		"matchType=qualification"
	recorder = web.postHttpResponse("/setup/schedule/generate", postData)
	assert.Equal(t, 303, recorder.Code)

	// The same team on both sides of the swap.
	recorder = web.postHttpResponse("/setup/schedule/swap?matchType=qualification", "team1=101&team2=101")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Select two different teams to swap.")

	// A team that is not in the generated schedule would be dropped into it rather than exchanged.
	recorder = web.postHttpResponse("/setup/schedule/swap?matchType=qualification", "team1=101&team2=254")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `Team "254" is not in the generated schedule.`)

	// The schedule must be untouched by any of the rejected swaps.
	scheduled := make(map[game.TeamId]int)
	for _, match := range cachedMatches[model.Qualification] {
		for _, id := range match.TeamIds() {
			scheduled[id]++
		}
	}
	assert.Equal(t, 12, len(scheduled))
	assert.Equal(t, 10, scheduled["101"])
}

func TestSetupScheduleErrors(t *testing.T) {
	web := setupTestWeb(t)

	// No teams.
	postData := "numScheduleBlocks=1&startTime0=2014-01-01 09:00:00 AM&numMatches0=7&matchSpacingSec0=480&" +
		"matchType=practice"
	recorder := web.postHttpResponse("/setup/schedule/generate", postData)
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "No team list is configured.")

	// Insufficient number of teams.
	for i := 0; i < 5; i++ {
		web.arena.Database.CreateTeam(&model.Team{Id: teamId(i + 101)})
	}
	postData = "numScheduleBlocks=1&startTime0=2014-01-01 09:00:00 AM&numMatches0=7&matchSpacingSec0=480&" +
		"matchType=practice"
	recorder = web.postHttpResponse("/setup/schedule/generate", postData)
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "There must be at least 6 teams to generate a schedule.")

	// More matches per team than schedules exist for. 700 matches over 6 teams is 466 matches per team.
	web.arena.Database.CreateTeam(&model.Team{Id: "118"})
	postData = "numScheduleBlocks=1&startTime0=2014-01-01 09:00:00 AM&numMatches0=700&matchSpacingSec0=480&" +
		"matchType=practice"
	recorder = web.postHttpResponse("/setup/schedule/generate", postData)
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "No schedule template exists for 6 teams and 466 matches")

	// Incomplete scheduling data received.
	postData = "numScheduleBlocks=1&startTime0=2014-01-01 09:00:00 AM&numMatches0=&matchSpacingSec0=480&" +
		"matchType=practice"
	recorder = web.postHttpResponse("/setup/schedule/generate", postData)
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Incomplete or invalid schedule block parameters specified.")

	// Previous schedule already exists. 30 matches over 12 teams works out to 10 matches per team.
	for i := 0; i < 6; i++ {
		web.arena.Database.CreateTeam(&model.Team{Id: teamId(i + 119)})
	}
	web.arena.Database.CreateMatch(&model.Match{Type: model.Practice, ShortName: "P1"})
	web.arena.Database.CreateMatch(&model.Match{Type: model.Practice, ShortName: "P2"})
	postData = "numScheduleBlocks=1&startTime0=2014-01-01 09:00:00 AM&numMatches0=30&matchSpacingSec0=480&" +
		"matchType=practice"
	recorder = web.postHttpResponse("/setup/schedule/generate", postData)
	assert.Equal(t, 303, recorder.Code)
	recorder = web.postHttpResponse("/setup/schedule/save", postData)
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "schedule of 2 Practice matches already exists")
}
