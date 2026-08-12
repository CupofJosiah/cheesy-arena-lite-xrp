// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web handlers for the referee interface.

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/field"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/websocket"
	"github.com/mitchellh/mapstructure"
	"io"
	"log"
	"net/http"
)

// Renders the referee interface for assigning fouls.
func (web *Web) refereePanelHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	template, err := web.parseFiles("templates/referee_panel.html", "templates/base.html")
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

// The websocket endpoint for the refereee interface client to send control commands and receive status updates.
func (web *Web) refereePanelWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	ws, err := websocket.NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer closeWebsocket(ws)

	// Subscribe the websocket to the notifiers whose messages will be passed on to the client, in a separate goroutine.
	go ws.HandleNotifiers(
		web.arena.MatchLoadNotifier,
		web.arena.MatchTimeNotifier,
		web.arena.RealtimeScoreNotifier,
		web.arena.ScoringStatusNotifier,
		web.arena.ReloadDisplaysNotifier,
	)

	// Loop, waiting for commands and responding to them, until the client closes the connection.
	for {
		messageType, data, err := ws.Read()
		if err != nil {
			if err == io.EOF {
				// Client has closed the connection; nothing to do here.
				return
			}
			log.Println(err)
			return
		}

		switch messageType {
		case "penalties":
			// Sets the absolute penalty counts for both alliances, as typed into the referee panel.
			args := struct {
				RedMinorPenalties  int
				RedMajorPenalties  int
				BlueMinorPenalties int
				BlueMajorPenalties int
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				writeWebsocketError(ws, err.Error())
				continue
			}
			redScore := &web.arena.RedRealtimeScore.CurrentScore
			blueScore := &web.arena.BlueRealtimeScore.CurrentScore
			redScore.MinorPenalties = max(args.RedMinorPenalties, 0)
			redScore.MajorPenalties = max(args.RedMajorPenalties, 0)
			blueScore.MinorPenalties = max(args.BlueMinorPenalties, 0)
			blueScore.MajorPenalties = max(args.BlueMajorPenalties, 0)
			web.arena.RealtimeScoreNotifier.Notify()
		case "addPenalty":
			// Adjusts one alliance's count of a single penalty tier, as tapped on the referee panel.
			args := struct {
				Alliance string
				Tier     string
				Delta    int
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				writeWebsocketError(ws, err.Error())
				continue
			}
			score := &web.arena.BlueRealtimeScore.CurrentScore
			if args.Alliance == "red" {
				score = &web.arena.RedRealtimeScore.CurrentScore
			}
			switch args.Tier {
			case "minor":
				score.MinorPenalties = max(score.MinorPenalties+args.Delta, 0)
			case "major":
				score.MajorPenalties = max(score.MajorPenalties+args.Delta, 0)
			default:
				writeWebsocketError(ws, fmt.Sprintf("Invalid penalty tier '%s'.", args.Tier))
				continue
			}
			web.arena.RealtimeScoreNotifier.Notify()
		case "card":
			args := struct {
				Alliance string
				TeamId   game.TeamId
				Card     string
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				writeWebsocketError(ws, err.Error())
				continue
			}

			// Set the card in the correct alliance's score.
			var cards map[string]string
			if args.Alliance == "red" {
				cards = web.arena.RedRealtimeScore.Cards
			} else {
				cards = web.arena.BlueRealtimeScore.Cards
			}
			if web.arena.CurrentMatch.Type == model.Playoff {
				// Cards apply to the whole alliance in playoffs.
				if args.Alliance == "red" {
					cards[string(web.arena.CurrentMatch.Red1)] = args.Card
					cards[string(web.arena.CurrentMatch.Red2)] = args.Card
				} else {
					cards[string(web.arena.CurrentMatch.Blue1)] = args.Card
					cards[string(web.arena.CurrentMatch.Blue2)] = args.Card
				}
			} else {
				cards[string(args.TeamId)] = args.Card
			}
			web.arena.RealtimeScoreNotifier.Notify()
		case "signalVolunteers":
			web.arena.SignalVolunteers()
		case "signalReset":
			web.arena.SignalReset()
		case "commitMatch":
			if web.arena.MatchState != field.PostMatch {
				// Don't allow committing the fouls until the match is over.
				continue
			}
			web.arena.RedRealtimeScore.FoulsCommitted = true
			web.arena.BlueRealtimeScore.FoulsCommitted = true
			web.arena.ScoringStatusNotifier.Notify()
		case "commitAndPost":
			if web.arena.MatchState != field.PostMatch {
				// Don't allow committing the match until it is over.
				continue
			}
			web.arena.RedRealtimeScore.FoulsCommitted = true
			web.arena.BlueRealtimeScore.FoulsCommitted = true
			web.arena.ScoringStatusNotifier.Notify()

			if err = web.commitPostAndLoadNextMatch(); err != nil {
				writeWebsocketError(ws, err.Error())
				continue
			}
		default:
			writeWebsocketError(ws, fmt.Sprintf("Invalid message type '%s'.", messageType))
		}
	}
}
