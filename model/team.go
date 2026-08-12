// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Model and datastore CRUD methods for a team at an event.

package model

import (
	"github.com/Team254/cheesy-arena-lite/game"
	"sort"
)

type Team struct {
	Id              game.TeamId `db:"id,manual"`
	Name            string
	Nickname        string
	City            string
	StateProv       string
	Country         string
	SchoolName      string
	RookieYear      int
	RobotName       string
	Accomplishments string
	YellowCard      bool
	HasConnected    bool
	FtaNotes        string
}

func (database *Database) CreateTeam(team *Team) error {
	return database.teamTable.create(team)
}

func (database *Database) GetTeamById(id game.TeamId) (*Team, error) {
	return database.teamTable.getById(id)
}

func (database *Database) UpdateTeam(team *Team) error {
	return database.teamTable.update(team)
}

func (database *Database) DeleteTeam(id game.TeamId) error {
	return database.teamTable.delete(id)
}

func (database *Database) TruncateTeams() error {
	return database.teamTable.truncate()
}

func (database *Database) GetAllTeams() ([]Team, error) {
	teams, err := database.teamTable.getAll()
	if err != nil {
		return nil, err
	}
	sort.Slice(
		teams, func(i, j int) bool {
			return game.LessTeamId(teams[i].Id, teams[j].Id)
		},
	)
	return teams, nil
}
