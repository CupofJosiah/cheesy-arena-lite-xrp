// Copyright 2018 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side logic for the field testing page.

var websocket;

// Sends a websocket message to play the given sound on the audience display.
var playSound = function (sound) {
  websocket.send("playSound", sound);
};

$(function () {
  websocket = new CheesyWebsocket("/setup/field_testing/websocket", {});
});
