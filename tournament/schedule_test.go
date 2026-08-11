// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package tournament

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestNonExistentSchedule(t *testing.T) {
	teams := make([]model.Team, 5)
	scheduleBlocks := []model.ScheduleBlock{{0, model.Test, time.Unix(0, 0).UTC(), 2, 60}}
	_, err := BuildRandomSchedule(teams, scheduleBlocks, model.Test)
	expectedErr := "No schedule template exists for 5 teams and 1 matches"
	if assert.NotNil(t, err) {
		assert.Equal(t, expectedErr, err.Error())
	}
}

func TestMalformedSchedule(t *testing.T) {
	filename := fmt.Sprintf("%s/4_1_2.csv", filepath.Join(model.BaseDir, schedulesDir))
	scheduleFile, _ := os.Create(filename)
	defer os.Remove(filename)
	scheduleFile.WriteString("1,0,2,0,3,0,4,0\n4,0,3,0,2,0,1,0\n")
	scheduleFile.Close()
	teams := make([]model.Team, 4)
	scheduleBlocks := []model.ScheduleBlock{{0, model.Test, time.Unix(0, 0).UTC(), 1, 60}}
	_, err := BuildRandomSchedule(teams, scheduleBlocks, model.Test)
	expectedErr := "Schedule file contains 2 matches, expected 1"
	if assert.NotNil(t, err) {
		assert.Equal(t, expectedErr, err.Error())
	}

	os.Remove(filename)
	scheduleFile, _ = os.Create(filename)
	scheduleFile.WriteString("1,0,asdf,0,3,0,4,0\n")
	scheduleFile.Close()
	_, err = BuildRandomSchedule(teams, scheduleBlocks, model.Test)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "strconv.Atoi")
	}
}

// A schedule row that is too short must be reported rather than panicking.
func TestShortScheduleRow(t *testing.T) {
	filename := fmt.Sprintf("%s/4_1_2.csv", filepath.Join(model.BaseDir, schedulesDir))
	scheduleFile, _ := os.Create(filename)
	defer os.Remove(filename)
	scheduleFile.WriteString("1,0,2,0\n")
	scheduleFile.Close()
	teams := make([]model.Team, 4)
	scheduleBlocks := []model.ScheduleBlock{{0, model.Test, time.Unix(0, 0).UTC(), 1, 60}}
	_, err := BuildRandomSchedule(teams, scheduleBlocks, model.Test)
	if assert.NotNil(t, err) {
		assert.Equal(t, "Schedule file match 1 contains 4 fields, expected 8", err.Error())
	}
}

// Verifies the structure of a generated 2v2 schedule for each available number of matches per team.
func TestScheduleTeams(t *testing.T) {
	const numTeams = 12
	teams := make([]model.Team, numTeams)
	for i := 0; i < numTeams; i++ {
		teams[i].Id = i + 101
	}

	for _, matchesPerTeam := range []int{6, 7, 8, 9, 10} {
		randomizer := rand.New(rand.NewSource(0))
		schedulePerm = randomizer.Perm

		numMatches := numTeams * matchesPerTeam / TeamsPerMatch
		scheduleBlocks := []model.ScheduleBlock{
			{0, model.Qualification, time.Unix(0, 0).UTC(), numMatches, 60},
		}
		matches, err := BuildRandomSchedule(teams, scheduleBlocks, model.Qualification)
		if !assert.Nil(t, err) {
			continue
		}
		assert.Equal(t, numMatches, len(matches))

		matchCounts := make(map[int]int)
		for i, match := range matches {
			assert.Equal(t, model.Qualification, match.Type)
			assert.Equal(t, i+1, match.TypeOrder)
			assert.Equal(t, fmt.Sprintf("Q%d", i+1), match.ShortName)
			assert.Equal(t, fmt.Sprintf("Qualification %d", i+1), match.LongName)
			assert.Equal(t, "qm", match.TbaMatchKey.CompLevel)
			assert.Equal(t, i+1, match.TbaMatchKey.MatchNumber)
			assert.Equal(t, time.Unix(int64(i*60), 0).UTC(), match.Time)

			// A team must never appear twice in the same match.
			seen := make(map[int]bool)
			for _, teamId := range match.TeamIds() {
				assert.False(t, seen[teamId], "team %d appears twice in match %d", teamId, i+1)
				seen[teamId] = true
				matchCounts[teamId]++
			}
		}

		// Every team must be scheduled, and each must play the same number of matches.
		assert.Equal(t, numTeams, len(matchCounts))
		for _, team := range teams {
			assert.Equal(t, matchesPerTeam, matchCounts[team.Id], "team %d", team.Id)
		}
	}
}

func TestScheduleTiming(t *testing.T) {
	teams := make([]model.Team, 12)
	// 30 matches over three blocks works out to 10 qualification matches per team.
	scheduleBlocks := []model.ScheduleBlock{
		{0, model.Qualification, time.Unix(100, 0).UTC(), 10, 75},
		{0, model.Qualification, time.Unix(20000, 0).UTC(), 5, 1000},
		{0, model.Qualification, time.Unix(100000, 0).UTC(), 15, 29},
	}
	matches, err := BuildRandomSchedule(teams, scheduleBlocks, model.Qualification)
	assert.Nil(t, err)
	assert.Equal(t, time.Unix(100, 0).UTC(), matches[0].Time)
	assert.Equal(t, time.Unix(775, 0).UTC(), matches[9].Time)
	assert.Equal(t, time.Unix(20000, 0).UTC(), matches[10].Time)
	assert.Equal(t, time.Unix(24000, 0).UTC(), matches[14].Time)
	assert.Equal(t, time.Unix(100000, 0).UTC(), matches[15].Time)
	assert.Equal(t, time.Unix(100406, 0).UTC(), matches[29].Time)
}

// The bundled 2v2 schedules divide evenly and so contain no surrogates; use a synthetic file to cover the flags.
func TestScheduleSurrogates(t *testing.T) {
	randomizer := rand.New(rand.NewSource(0))
	schedulePerm = randomizer.Perm

	filename := fmt.Sprintf("%s/4_2_2.csv", filepath.Join(model.BaseDir, schedulesDir))
	scheduleFile, _ := os.Create(filename)
	defer os.Remove(filename)
	scheduleFile.WriteString("1,0,2,0,3,0,4,0\n1,1,2,0,3,1,4,0\n")
	scheduleFile.Close()

	teams := make([]model.Team, 4)
	for i := range teams {
		teams[i].Id = i + 101
	}
	scheduleBlocks := []model.ScheduleBlock{{0, model.Qualification, time.Unix(0, 0).UTC(), 2, 60}}
	matches, err := BuildRandomSchedule(teams, scheduleBlocks, model.Qualification)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(matches))

	assert.False(t, matches[0].Red1IsSurrogate)
	assert.False(t, matches[0].Red2IsSurrogate)
	assert.False(t, matches[0].Blue1IsSurrogate)
	assert.False(t, matches[0].Blue2IsSurrogate)

	assert.True(t, matches[1].Red1IsSurrogate)
	assert.False(t, matches[1].Red2IsSurrogate)
	assert.True(t, matches[1].Blue1IsSurrogate)
	assert.False(t, matches[1].Blue2IsSurrogate)
}
