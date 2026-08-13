// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side logic for the Iron Acres scoring interface.

var websocket;
let committed = false;
let scoringAvailable = false;
let commitAvailable = false;

// Values entered on this panel, mirrored from the server on every realtime score update.
const scoreFields = [
  "FactoryParks",
  "AutoCrops",
  "SilosDumped",
  "TeleopCrops",
  "CityLimitsProducts",
  "CityCenterProducts",
  "BarnParks",
  "BarnHangs",
  "BonusPoints",
];

// The one field above that holds points rather than an element count, and so is neither multiplied by a point value
// nor clamped at zero.
const bonusField = "BonusPoints";

// Point values from section 5.2 of the manual, used only to preview totals before the server responds.
const pointValues = {
  FactoryParks: 5,
  AutoCrops: 7,
  SilosDumped: 5,
  TeleopCrops: 7,
  CityLimitsProducts: 10,
  CityCenterProducts: 15,
  BarnParks: 5,
  BarnHangs: 25,
};

const localScore = {
  red: emptyAllianceScore(),
  blue: emptyAllianceScore(),
};

function emptyAllianceScore() {
  const score = {};
  scoreFields.forEach((field) => (score[field] = 0));
  return score;
}

const scoreMessage = function () {
  return { Red: localScore.red, Blue: localScore.blue };
};

const sendScore = function () {
  websocket.send("score", scoreMessage());
};

// Adjusts one field and pushes the whole score to the server.
const adjustScore = function (alliance, field, delta) {
  if (!scoringAvailable) {
    return;
  }
  const value = localScore[alliance][field] + delta;
  localScore[alliance][field] = field === bonusField ? value : Math.max(value, 0);
  renderAlliance(alliance);
  sendScore();
};

const renderAlliance = function (alliance) {
  const score = localScore[alliance];
  scoreFields.forEach((field) => {
    $(`#${alliance}-${field}`).text(score[field]);
  });

  const autoPoints =
    score.FactoryParks * pointValues.FactoryParks +
    score.AutoCrops * pointValues.AutoCrops +
    score.SilosDumped * pointValues.SilosDumped;
  const teleopPoints =
    score.TeleopCrops * pointValues.TeleopCrops +
    score.CityLimitsProducts * pointValues.CityLimitsProducts +
    score.CityCenterProducts * pointValues.CityCenterProducts;
  const endgamePoints = score.BarnParks * pointValues.BarnParks + score.BarnHangs * pointValues.BarnHangs;
  const bonusPoints = score[bonusField];

  $(`#${alliance}-autoPoints`).text(autoPoints);
  $(`#${alliance}-teleopPoints`).text(teleopPoints);
  $(`#${alliance}-endgamePoints`).text(endgamePoints);
  $(`#${alliance}-bonusPoints`).text(bonusPoints);
  $(`#${alliance}-totalPoints`).text(autoPoints + teleopPoints + endgamePoints + bonusPoints);
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
  $(".counter-button").prop("disabled", !scoringAvailable);
  $("#commit").prop("disabled", !commitAvailable);
};

const handleRealtimeScore = function (data) {
  ["red", "blue"].forEach((alliance) => {
    const serverScore = alliance === "red" ? data.Red.Score : data.Blue.Score;
    scoreFields.forEach((field) => {
      localScore[alliance][field] = serverScore[field] || 0;
    });
    renderAlliance(alliance);
  });
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
  renderAlliance("red");
  renderAlliance("blue");

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
