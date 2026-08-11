// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web handlers for scoring interface.

package web

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Team254/cheesy-arena-lite/field"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/websocket"
	"github.com/mitchellh/mapstructure"
)

// Element counts entered on the scoring panel for one alliance. Point values are applied server-side by game.Score.
type scoringPanelAllianceScore struct {
	FactoryParks       int
	AutoCrops          int
	SilosDumped        int
	TeleopCrops        int
	CityLimitsProducts int
	CityCenterProducts int
	BarnParks          int
	BarnHangs          int
}

type scoringPanelScoreMessage struct {
	Red  scoringPanelAllianceScore
	Blue scoringPanelAllianceScore
}

// Copies the entered element counts into the given score, ignoring any negative values.
func (allianceScore *scoringPanelAllianceScore) applyTo(score *game.Score) {
	score.FactoryParks = max(allianceScore.FactoryParks, 0)
	score.AutoCrops = max(allianceScore.AutoCrops, 0)
	score.SilosDumped = max(allianceScore.SilosDumped, 0)
	score.TeleopCrops = max(allianceScore.TeleopCrops, 0)
	score.CityLimitsProducts = max(allianceScore.CityLimitsProducts, 0)
	score.CityCenterProducts = max(allianceScore.CityCenterProducts, 0)
	score.BarnParks = max(allianceScore.BarnParks, 0)
	score.BarnHangs = max(allianceScore.BarnHangs, 0)
}

// Renders the scoring interface which enables input of scores in real-time.
func (web *Web) scoringPanelHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	template, err := web.parseFiles("templates/scoring_panel.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
	}{web.arena.EventSettings}
	err = template.ExecuteTemplate(w, "base_no_navbar", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// The websocket endpoint for the scoring interface client to send control commands and receive status updates.
func (web *Web) scoringPanelWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	ws, err := websocket.NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer closeWebsocket(ws)
	web.arena.ScoringPanelRegistry.RegisterPanel("scoring", ws)
	web.arena.ScoringStatusNotifier.Notify()
	defer web.arena.ScoringStatusNotifier.Notify()
	defer web.arena.ScoringPanelRegistry.UnregisterPanel("scoring", ws)

	writeWebsocketMessage(ws, "resetLocalState", nil)

	go ws.HandleNotifiers(
		web.arena.MatchLoadNotifier,
		web.arena.MatchTimeNotifier,
		web.arena.RealtimeScoreNotifier,
		web.arena.ReloadDisplaysNotifier,
	)

	for {
		command, data, err := ws.Read()
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Println(err)
			return
		}

		switch command {
		case "commitMatch":
			if web.arena.MatchState != field.PostMatch {
				writeWebsocketError(ws, "Cannot commit score: Match is not over.")
				continue
			}
			web.arena.ScoringPanelRegistry.SetScoreCommitted("scoring", ws)
			web.arena.ScoringStatusNotifier.Notify()
		case "score":
			var args scoringPanelScoreMessage
			if err = mapstructure.Decode(data, &args); err != nil {
				writeWebsocketError(ws, err.Error())
				continue
			}

			args.Red.applyTo(&web.arena.RedRealtimeScore.CurrentScore)
			args.Blue.applyTo(&web.arena.BlueRealtimeScore.CurrentScore)
			web.arena.RealtimeScoreNotifier.Notify()
		default:
			writeWebsocketError(ws, fmt.Sprintf("Invalid message type '%s'.", command))
		}
	}
}
