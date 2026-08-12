// Copyright 2026 Team 254. All Rights Reserved.
//
// Type and helpers for team identifiers, which may contain letters as well as digits.

package game

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Identifies a team at the event. Team numbers are alphanumeric rather than purely numeric so that events can run
// multiple teams from the same school as e.g. "12A" and "12B". The empty string means "no team".
type TeamId string

// The maximum number of characters in a team ID, limited by the width of the team number displays.
const MaxTeamIdLength = 6

// Returns the team ID parsed from the given user-supplied string, normalized to uppercase. Returns an error if the
// string contains anything other than letters and digits, since team IDs are used verbatim in URLs and CSV exports.
func ParseTeamId(teamIdString string) (TeamId, error) {
	trimmed := strings.TrimSpace(teamIdString)
	if trimmed == "" {
		return "", fmt.Errorf("team number cannot be blank")
	}
	if len(trimmed) > MaxTeamIdLength {
		return "", fmt.Errorf("team number %q is longer than %d characters", trimmed, MaxTeamIdLength)
	}
	for _, character := range trimmed {
		isDigit := character >= '0' && character <= '9'
		isLetter := character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
		if !isDigit && !isLetter {
			return "", fmt.Errorf("team number %q must contain only letters and digits", trimmed)
		}
	}
	return TeamId(strings.ToUpper(trimmed)), nil
}

// Returns the team ID parsed from the given string, or the empty team ID if it is not valid. For use in contexts where
// a blank or malformed value should be treated as "no team" rather than as an error.
func TeamIdFromString(teamIdString string) TeamId {
	teamId, err := ParseTeamId(teamIdString)
	if err != nil {
		return ""
	}
	return teamId
}

func (teamId TeamId) String() string {
	return string(teamId)
}

// Reads a team ID from JSON, accepting a bare number as well as a string so that databases written before team IDs
// allowed letters can still be loaded.
func (teamId *TeamId) UnmarshalJSON(data []byte) error {
	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		*teamId = TeamId(stringValue)
		return nil
	}

	var intValue int
	if err := json.Unmarshal(data, &intValue); err != nil {
		return fmt.Errorf("team ID must be a string or a number; got %s", string(data))
	}
	if intValue == 0 {
		*teamId = ""
	} else {
		*teamId = TeamId(strconv.Itoa(intValue))
	}
	return nil
}

// Returns true if team ID a should sort before team ID b, comparing runs of digits numerically so that "9A" sorts
// before "10A".
func LessTeamId(a, b TeamId) bool {
	return compareTeamIds(a, b) < 0
}

// Returns a negative number if a sorts before b, a positive number if it sorts after, and zero if they are equal.
func compareTeamIds(a, b TeamId) int {
	aRunes, bRunes := []rune(string(a)), []rune(string(b))
	i, j := 0, 0
	for i < len(aRunes) && j < len(bRunes) {
		if isDigitRune(aRunes[i]) && isDigitRune(bRunes[j]) {
			// Compare the whole run of digits as a number, ignoring leading zeroes.
			aStart, bStart := i, j
			for i < len(aRunes) && isDigitRune(aRunes[i]) {
				i++
			}
			for j < len(bRunes) && isDigitRune(bRunes[j]) {
				j++
			}
			aDigits := strings.TrimLeft(string(aRunes[aStart:i]), "0")
			bDigits := strings.TrimLeft(string(bRunes[bStart:j]), "0")
			if len(aDigits) != len(bDigits) {
				return len(aDigits) - len(bDigits)
			}
			if aDigits != bDigits {
				return strings.Compare(aDigits, bDigits)
			}
			continue
		}

		if aRunes[i] != bRunes[j] {
			return int(aRunes[i]) - int(bRunes[j])
		}
		i++
		j++
	}
	return (len(aRunes) - i) - (len(bRunes) - j)
}

func isDigitRune(character rune) bool {
	return character >= '0' && character <= '9'
}
