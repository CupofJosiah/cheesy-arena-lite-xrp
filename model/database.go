// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Functions for manipulating the per-event Bolt datastore.

package model

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/game"
	"go.etcd.io/bbolt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const backupsDir = "db/backups"

var BaseDir = "." // Mutable for testing

type Database struct {
	Path                string
	bolt                *bbolt.DB
	allianceTable       *table[Alliance, int]
	awardTable          *table[Award, int]
	eventSettingsTable  *table[EventSettings, int]
	judgingSlotTable    *table[JudgingSlot, int]
	lowerThirdTable     *table[LowerThird, int]
	matchTable          *table[Match, int]
	matchResultTable    *table[MatchResult, int]
	rankingTable        *table[game.Ranking, game.TeamId]
	scheduleBlockTable  *table[ScheduleBlock, int]
	scheduledBreakTable *table[ScheduledBreak, int]
	sponsorSlideTable   *table[SponsorSlide, int]
	teamTable           *table[Team, game.TeamId]
	userSessionTable    *table[UserSession, int]
}

// Opens the Bolt database at the given path, creating it if it doesn't exist.
func OpenDatabase(filename string) (*Database, error) {
	database := Database{Path: filename}
	var err error
	database.bolt, err = bbolt.Open(database.Path, 0644, &bbolt.Options{NoSync: true, Timeout: time.Second})
	if err != nil {
		return nil, err
	}

	// Register tables.
	if database.allianceTable, err = newTable[Alliance, int](&database); err != nil {
		return nil, err
	}
	if database.awardTable, err = newTable[Award, int](&database); err != nil {
		return nil, err
	}
	if database.eventSettingsTable, err = newTable[EventSettings, int](&database); err != nil {
		return nil, err
	}
	if database.judgingSlotTable, err = newTable[JudgingSlot, int](&database); err != nil {
		return nil, err
	}
	if database.lowerThirdTable, err = newTable[LowerThird, int](&database); err != nil {
		return nil, err
	}
	if database.matchTable, err = newTable[Match, int](&database); err != nil {
		return nil, err
	}
	if database.matchResultTable, err = newTable[MatchResult, int](&database); err != nil {
		return nil, err
	}
	if database.rankingTable, err = newTable[game.Ranking, game.TeamId](&database); err != nil {
		return nil, err
	}
	if database.scheduleBlockTable, err = newTable[ScheduleBlock, int](&database); err != nil {
		return nil, err
	}
	if database.scheduledBreakTable, err = newTable[ScheduledBreak, int](&database); err != nil {
		return nil, err
	}
	if database.sponsorSlideTable, err = newTable[SponsorSlide, int](&database); err != nil {
		return nil, err
	}
	if database.teamTable, err = newTable[Team, game.TeamId](&database); err != nil {
		return nil, err
	}
	if database.userSessionTable, err = newTable[UserSession, int](&database); err != nil {
		return nil, err
	}

	return &database, nil
}

func (database *Database) Close() error {
	return database.bolt.Close()
}

// Creates a copy of the current database and saves it to the backups directory.
func (database *Database) Backup(eventName, reason string) error {
	backupsPath := filepath.Join(BaseDir, backupsDir)
	err := os.MkdirAll(backupsPath, 0755)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf(
		"%s/%s_%s_%s.db",
		backupsPath,
		strings.Replace(eventName, " ", "_", -1),
		time.Now().Format("20060102150405"),
		reason,
	)

	dest, err := os.Create(filename)
	if err != nil {
		return err
	}

	err = database.WriteBackup(dest)
	closeErr := dest.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// Takes a snapshot of Bolt database and writes it to the given writer.
func (database *Database) WriteBackup(writer io.Writer) error {
	return database.bolt.View(
		func(tx *bbolt.Tx) error {
			_, err := tx.WriteTo(writer)
			return err
		},
	)
}
