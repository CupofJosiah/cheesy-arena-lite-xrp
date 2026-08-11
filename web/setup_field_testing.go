// Copyright 2018 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web routes for testing the field sounds.

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/websocket"
	"io"
	"log"
	"net/http"
)

// Shows the Field Testing page.
func (web *Web) fieldTestingGetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	template, err := web.parseFiles("templates/setup_field_testing.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
		MatchSounds []*game.MatchSound
	}{
		web.arena.EventSettings,
		game.UniqueMatchSounds(),
	}
	err = template.ExecuteTemplate(w, "base", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// The websocket endpoint for sending realtime updates to the Field Testing page.
func (web *Web) fieldTestingWebsocketHandler(w http.ResponseWriter, r *http.Request) {
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
	go ws.HandleNotifiers(web.arena.ArenaStatusNotifier)

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
		case "playSound":
			sound, ok := data.(string)
			if !ok {
				writeWebsocketError(ws, fmt.Sprintf("Failed to parse '%s' message.", messageType))
				continue
			}
			web.arena.PlaySoundNotifier.NotifyWithMessage(sound)
		default:
			writeWebsocketError(ws, fmt.Sprintf("Invalid message type '%s'.", messageType))
			continue
		}
	}
}
