// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side logic for the generic scoring interface.

var websocket;
let committed = false;
let scoringAvailable = false;
let commitAvailable = false;

const parseScoreInput = function (selector) {
  const value = parseInt($(selector).val(), 10);
  return Number.isFinite(value) && value >= 0 ? value : 0;
};

const scoreMessage = function () {
  return {
    RedAuto: parseScoreInput("#redAuto"),
    RedTeleop: parseScoreInput("#redTeleop"),
    RedPostMatch: parseScoreInput("#redPostMatch"),
    BlueAuto: parseScoreInput("#blueAuto"),
    BlueTeleop: parseScoreInput("#blueTeleop"),
    BluePostMatch: parseScoreInput("#bluePostMatch"),
  };
};

const sendScore = function () {
  websocket.send("score", scoreMessage());
};

const setInputValue = function (selector, value) {
  const input = $(selector);
  if (!input.is(":focus")) {
    input.val(value || 0);
  }
};

const handleMatchLoad = function (data) {
  $("#matchName").text(data.Match.LongName);
};

const handleMatchTime = function (data) {
  switch (matchStates[data.MatchState]) {
    case "AUTO_PERIOD":
    case "PAUSE_PERIOD":
    case "TELEOP_PERIOD":
      scoringAvailable = true;
      commitAvailable = false;
      committed = false;
      break;
    case "POST_MATCH":
      if (!committed) {
        scoringAvailable = true;
        commitAvailable = true;
      }
      break;
    default:
      scoringAvailable = false;
      commitAvailable = false;
      committed = false;
  }
  updateUIMode();
};

const resetLocalState = function () {
  committed = false;
  updateUIMode();
};

const updateUIMode = function () {
  $("input[type=number]").prop("disabled", !scoringAvailable);
  $("#commit").prop("disabled", !commitAvailable);
};

const handleRealtimeScore = function (data) {
  setInputValue("#redAuto", data.Red.Score.AutoPoints);
  setInputValue("#redTeleop", data.Red.Score.TeleopPoints);
  setInputValue("#redPostMatch", data.Red.Score.PostMatchPoints);
  setInputValue("#blueAuto", data.Blue.Score.AutoPoints);
  setInputValue("#blueTeleop", data.Blue.Score.TeleopPoints);
  setInputValue("#bluePostMatch", data.Blue.Score.PostMatchPoints);
};

const commitMatchScore = function () {
  sendScore();
  websocket.send("commitMatch");

  committed = true;
  scoringAvailable = false;
  commitAvailable = false;
  updateUIMode();
};

$(function () {
  resetLocalState();
  $("input[type=number]").on("input", sendScore);

  websocket = new CheesyWebsocket("/panels/scoring/websocket", {
    matchLoad: function (event) {
      handleMatchLoad(event.data);
    },
    matchTime: function (event) {
      handleMatchTime(event.data);
    },
    realtimeScore: function (event) {
      handleRealtimeScore(event.data);
    },
    resetLocalState: function () {
      resetLocalState();
    },
  });
});
