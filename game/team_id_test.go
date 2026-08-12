// Copyright 2026 Team 254. All Rights Reserved.

package game

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestParseTeamId(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected TeamId
	}{
		{"254", "254"},
		{" 254 ", "254"},
		{"12a", "12A"},
		{"12A", "12A"},
		{"A", "A"},
		{"123456", "123456"},
	} {
		teamId, err := ParseTeamId(test.input)
		if assert.Nil(t, err, test.input) {
			assert.Equal(t, test.expected, teamId, test.input)
		}
	}

	for _, test := range []struct {
		input   string
		message string
	}{
		{"", "cannot be blank"},
		{"   ", "cannot be blank"},
		{"1234567", "longer than 6 characters"},
		{"12-A", "only letters and digits"},
		{"12 A", "only letters and digits"},
		{"../etc", "only letters and digits"},
		{"12.A", "only letters and digits"},
	} {
		teamId, err := ParseTeamId(test.input)
		if assert.NotNil(t, err, test.input) {
			assert.Contains(t, err.Error(), test.message, test.input)
		}
		assert.Equal(t, TeamId(""), teamId, test.input)
	}
}

func TestTeamIdFromString(t *testing.T) {
	assert.Equal(t, TeamId("12A"), TeamIdFromString("12a"))
	assert.Equal(t, TeamId(""), TeamIdFromString(""))
	assert.Equal(t, TeamId(""), TeamIdFromString("not a team"))
}

func TestTeamIdJson(t *testing.T) {
	// Team IDs are written as JSON strings.
	data, err := json.Marshal(TeamId("12A"))
	assert.Nil(t, err)
	assert.Equal(t, `"12A"`, string(data))

	var teamId TeamId
	assert.Nil(t, json.Unmarshal([]byte(`"12A"`), &teamId))
	assert.Equal(t, TeamId("12A"), teamId)

	// A bare number is also accepted, for databases written before team IDs allowed letters.
	assert.Nil(t, json.Unmarshal([]byte("254"), &teamId))
	assert.Equal(t, TeamId("254"), teamId)

	// Zero was the old representation of "no team".
	assert.Nil(t, json.Unmarshal([]byte("0"), &teamId))
	assert.Equal(t, TeamId(""), teamId)

	assert.NotNil(t, json.Unmarshal([]byte("true"), &teamId))

	// Structs containing team IDs round-trip through both representations.
	type lineup struct {
		Red1 TeamId
		Red2 TeamId
	}
	var oldLineup lineup
	assert.Nil(t, json.Unmarshal([]byte(`{"Red1":254,"Red2":0}`), &oldLineup))
	assert.Equal(t, lineup{Red1: "254", Red2: ""}, oldLineup)
}

func TestLessTeamId(t *testing.T) {
	teamIds := []TeamId{"10", "9B", "9A", "2", "100A", "100", "A", "1", "9", "01"}
	sort.Slice(teamIds, func(i, j int) bool { return LessTeamId(teamIds[i], teamIds[j]) })

	// Runs of digits sort numerically, so 9 comes before 10; a bare number sorts before the same number with a
	// suffix; leading zeroes don't affect the ordering except as a tiebreak.
	assert.Equal(
		t,
		[]TeamId{"1", "01", "2", "9", "9A", "9B", "10", "100", "100A", "A"},
		teamIds,
	)

	assert.False(t, LessTeamId("254", "254"))
	assert.True(t, LessTeamId("", "1"))
	assert.False(t, LessTeamId("1", ""))
}
