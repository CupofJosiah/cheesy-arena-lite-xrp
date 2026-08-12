// Copyright 2026 Team 254. All Rights Reserved.
//
// Verifies that databases written before team IDs could contain letters are still readable.

package model

import (
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"
	"testing"
)

// Writes a record directly into a bucket, bypassing the table wrapper, so that the stored JSON can use the old
// numeric representation of team IDs.
func putLegacyRecord(t *testing.T, database *Database, bucketName, key, recordJson string) {
	assert.Nil(
		t,
		database.bolt.Update(
			func(tx *bbolt.Tx) error {
				return tx.Bucket([]byte(bucketName)).Put([]byte(key), []byte(recordJson))
			},
		),
	)
}

func TestReadLegacyNumericTeamIds(t *testing.T) {
	db := SetupTestDb(t)

	putLegacyRecord(t, db, "Team", "254", `{"Id":254,"Nickname":"The Cheesy Poofs","RookieYear":1999}`)
	team, err := db.GetTeamById("254")
	assert.Nil(t, err)
	if assert.NotNil(t, team) {
		assert.Equal(t, game.TeamId("254"), team.Id)
		assert.Equal(t, "The Cheesy Poofs", team.Nickname)
	}

	// A match written with numeric team IDs, including a zero for an empty station.
	putLegacyRecord(
		t,
		db,
		"Match",
		"1",
		`{"Id":1,"Type":2,"TypeOrder":1,"ShortName":"Q1","Red1":254,"Red2":1114,"Blue1":971,"Blue2":0}`,
	)
	match, err := db.GetMatchById(1)
	assert.Nil(t, err)
	if assert.NotNil(t, match) {
		assert.Equal(
			t, [game.TeamsPerMatch]game.TeamId{"254", "1114", "971", ""}, match.TeamIds(),
		)
	}

	// An alliance written with numeric team IDs.
	putLegacyRecord(t, db, "Alliance", "1", `{"Id":1,"TeamIds":[254,1114,971],"Lineup":[1114,254]}`)
	alliance, err := db.GetAllianceById(1)
	assert.Nil(t, err)
	if assert.NotNil(t, alliance) {
		assert.Equal(t, []game.TeamId{"254", "1114", "971"}, alliance.TeamIds)
		assert.Equal(t, [game.TeamsPerAlliance]game.TeamId{"1114", "254"}, alliance.Lineup)
	}

	// A ranking, whose bucket key was already the base-10 team number.
	putLegacyRecord(t, db, "Ranking", "254", `{"TeamId":254,"Rank":1,"RankingPoints":20}`)
	ranking, err := db.GetRankingForTeam("254")
	assert.Nil(t, err)
	if assert.NotNil(t, ranking) {
		assert.Equal(t, game.TeamId("254"), ranking.TeamId)
		assert.Equal(t, 1, ranking.Rank)
	}

	// An award with no team attached used to store zero.
	putLegacyRecord(t, db, "Award", "1", `{"Id":1,"Type":0,"AwardName":"Safety","TeamId":0,"PersonName":"Bob"}`)
	award, err := db.GetAwardById(1)
	assert.Nil(t, err)
	if assert.NotNil(t, award) {
		assert.Equal(t, game.TeamId(""), award.TeamId)
	}
}
