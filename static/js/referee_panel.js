// Copyright 2023 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side logic for the referee interface.

var websocket;
let refCommitted = false;
let controlsAvailable = false;
let scoreIsReady = false;

const parseFoulInput = function (selector) {
  const value = parseInt($(selector).val(), 10);
  return Number.isFinite(value) && value >= 0 ? value : 0;
};

const sendFoulPoints = function () {
  websocket.send("foulPoints", {
    RedFoulPointsAgainst: parseFoulInput("#redFoulPointsAgainst"),
    BlueFoulPointsAgainst: parseFoulInput("#blueFoulPointsAgainst"),
  });
};

const setInputValue = function (selector, value) {
  const input = $(selector);
  if (!input.is(":focus")) {
    input.val(value || 0);
  }
};

var cycleCard = function (cardButton) {
  if (!controlsAvailable || refCommitted) {
    return;
  }

  const currentCard = $(cardButton).attr("data-card");
  const hasOldYellowCard = $(cardButton).attr("data-old-yellow-card") === "true";
  let newCard = "";
  if (currentCard === "" && hasOldYellowCard) {
    newCard = "red";
  } else if (currentCard === "") {
    newCard = "yellow";
  } else if (currentCard === "yellow") {
    newCard = "red";
  }
  websocket.send(
    "card",
    {Alliance: $(cardButton).attr("data-alliance"), TeamId: parseInt($(cardButton).attr("data-team")), Card: newCard}
  );
  $(cardButton).attr("data-card", newCard);
};

var signalVolunteers = function () {
  websocket.send("signalVolunteers");
};

var signalReset = function () {
  websocket.send("signalReset");
};

var confirmCommit = function () {
	if (!controlsAvailable || refCommitted) {
		return;
	}
	if (scoreIsReady) {
		commitAndPost();
		return;
	}

	$("#confirmCommit").modal("show");
};

var commitAndPost = function () {
	sendFoulPoints();
	websocket.send("commitAndPost");
};

var handleMatchLoad = function (data) {
  $("#matchName").text(data.Match.LongName);

  setTeamCard("red", 1, data.Teams["R1"]);
  setTeamCard("red", 2, data.Teams["R2"]);
  setTeamCard("red", 3, data.Teams["R3"]);
  setTeamCard("blue", 1, data.Teams["B1"]);
  setTeamCard("blue", 2, data.Teams["B2"]);
  setTeamCard("blue", 3, data.Teams["B3"]);
};

const handleMatchTime = function (data) {
	controlsAvailable = matchStates[data.MatchState] === "POST_MATCH";
	if (!controlsAvailable) {
		refCommitted = false;
	}
	updateUIMode();
};

const handleRealtimeScore = function (data) {
  for (const [teamId, card] of Object.entries(Object.assign(data.RedCards, data.BlueCards))) {
    $(`[data-team="${teamId}"]`).attr("data-card", card);
  }

  setInputValue("#redFoulPointsAgainst", data.Red.Score.FoulPointsAgainst);
  setInputValue("#blueFoulPointsAgainst", data.Blue.Score.FoulPointsAgainst);
};

const handleScoringStatus = function (data) {
	refCommitted = data.RefereeScoreReady;
	scoreIsReady = data.ScoringReady;
	$("#scoringPanelStatus").text(`Scoring ${data.NumScoringPanelsReady}/${data.NumScoringPanels}`);
	$("#scoringPanelStatus").attr("data-present", data.NumScoringPanels > 0);
	$("#scoringPanelStatus").attr("data-ready", data.ScoringReady);
	$("#commitButton").toggleClass("disabled", !scoreIsReady);
	updateUIMode();
};

const updateUIMode = function () {
	$("input[type=number]").prop("disabled", false);
	$(".control-button").attr("data-enabled", controlsAvailable);
	$("#commitButton").attr("data-enabled", controlsAvailable && !refCommitted);
};

const setTeamCard = function (alliance, position, team) {
  const cardButton = $(`#${alliance}Team${position}Card`);
  if (team === null) {
    cardButton.text(0);
    cardButton.attr("data-team", 0);
    cardButton.attr("data-old-yellow-card", "");
  } else {
    cardButton.text(team.Id);
    cardButton.attr("data-team", team.Id);
    cardButton.attr("data-old-yellow-card", team.YellowCard);
  }
  cardButton.attr("data-card", "");
};

$(function () {
  $("input[type=number]").on("input", sendFoulPoints);

  websocket = new CheesyWebsocket("/panels/referee/websocket", {
    matchLoad: function (event) {
      handleMatchLoad(event.data);
    },
    matchTime: function (event) {
      handleMatchTime(event.data);
    },
    realtimeScore: function (event) {
      handleRealtimeScore(event.data);
    },
    scoringStatus: function (event) {
      handleScoringStatus(event.data);
    },
  });
});
