// Copyright 2023 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side logic for the referee interface.

var websocket;
let refCommitted = false;
let controlsAvailable = false;
let scoreIsReady = false;

// Penalty point values from T03 of the manual, used to preview the points awarded to the opponent.
const penaltyPointValues = {minor: 10, major: 25};

const penaltyCounts = {
  red: {minor: 0, major: 0},
  blue: {minor: 0, major: 0},
};

const sendPenalties = function () {
  websocket.send("penalties", {
    RedMinorPenalties: penaltyCounts.red.minor,
    RedMajorPenalties: penaltyCounts.red.major,
    BlueMinorPenalties: penaltyCounts.blue.minor,
    BlueMajorPenalties: penaltyCounts.blue.major,
  });
};

// Adjusts one alliance's count for a penalty tier. Penalties can be assessed at any time during a match.
var adjustPenalty = function (alliance, tier, delta) {
  penaltyCounts[alliance][tier] = Math.max(penaltyCounts[alliance][tier] + delta, 0);
  renderPenalties(alliance);
  websocket.send("addPenalty", {Alliance: alliance, Tier: tier, Delta: delta});
};

const renderPenalties = function (alliance) {
  const counts = penaltyCounts[alliance];
  $(`#${alliance}-minorPenalties`).text(counts.minor);
  $(`#${alliance}-majorPenalties`).text(counts.major);
  $(`#${alliance}-penaltyPoints`).text(
    counts.minor * penaltyPointValues.minor + counts.major * penaltyPointValues.major
  );
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
	sendPenalties();
	websocket.send("commitAndPost");
};

var handleMatchLoad = function (data) {
  $("#matchName").text(data.Match.LongName);

  setTeamCard("red", 1, data.Teams["R1"]);
  setTeamCard("red", 2, data.Teams["R2"]);
  setTeamCard("blue", 1, data.Teams["B1"]);
  setTeamCard("blue", 2, data.Teams["B2"]);
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

  penaltyCounts.red.minor = data.Red.Score.MinorPenalties || 0;
  penaltyCounts.red.major = data.Red.Score.MajorPenalties || 0;
  penaltyCounts.blue.minor = data.Blue.Score.MinorPenalties || 0;
  penaltyCounts.blue.major = data.Blue.Score.MajorPenalties || 0;
  renderPenalties("red");
  renderPenalties("blue");
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
  renderPenalties("red");
  renderPenalties("blue");

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
