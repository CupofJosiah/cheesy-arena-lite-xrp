// Copyright 2018 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side logic for the field monitor display.

let websocket;
let currentMatchId;
let redSide;
let blueSide;

const handleArenaStatus = function (data) {
  // If getting data for the wrong match (e.g. after a server restart), reload the page.
  if (currentMatchId == null) {
    currentMatchId = data.MatchId;
  } else if (currentMatchId !== data.MatchId) {
    location.reload();
  }

  $.each(data.AllianceStations, function (station, stationStatus) {
    // Select the DOM elements corresponding to the team station.
    let teamElementPrefix;
    if (station[0] === "R") {
      teamElementPrefix = "#" + redSide + "Team" + station[1];
    } else {
      teamElementPrefix = "#" + blueSide + "Team" + station[1];
    }
    const teamIdElement = $(teamElementPrefix + "Id");
    const teamNotesElement = $(teamElementPrefix + "Notes");
    const teamNotesTextElement = $(teamElementPrefix + "Notes div");
    const teamReadyElement = $(teamElementPrefix + "Ready");
    const teamBypassElement = $(teamElementPrefix + "Bypass");

    teamNotesTextElement.attr("data-station", station);

    if (stationStatus.Team) {
      // Set the team number and status.
      teamIdElement.text(stationStatus.Team.Id);
      let status = "no-link";
      if (stationStatus.Bypass) {
        status = "";
      } else if (stationStatus.Ready) {
        status = "robot-linked";
      }
      teamIdElement.attr("data-status", status);
      teamNotesTextElement.text(stationStatus.Team.FtaNotes);
      teamNotesElement.attr("data-status", status);
    } else {
      // No team is present in this position for this match; blank out the status.
      teamIdElement.text("");
      teamNotesTextElement.text("");
      teamNotesElement.attr("data-status", "");
    }

    teamReadyElement.attr("data-status-ok", stationStatus.Ready);
    teamReadyElement.text(stationStatus.Ready ? "READY" : "STAGE");

    if (stationStatus.EStop) {
      teamBypassElement.attr("data-status-ok", false);
      teamBypassElement.text("E-STP");
    } else if (stationStatus.AStop) {
      teamBypassElement.attr("data-status-ok", true);
      teamBypassElement.text("A-STP");
    } else if (stationStatus.Bypass) {
      teamBypassElement.attr("data-status-ok", false);
      teamBypassElement.text("BYP");
    } else {
      teamBypassElement.attr("data-status-ok", true);
      teamBypassElement.text("");
    }
  });
};

// Handles a websocket message to update the match time countdown.
const handleMatchTime = function (data) {
  translateMatchTime(data, function (matchState, matchStateText, countdownSec) {
    $("#matchState").text(matchStateText);
    $("#matchTime").text(countdownSec);
    if (matchStateText === "PRE-MATCH" || matchStateText === "POST-MATCH") {
      $(".ds-dependent").attr("data-preMatch", "true");
    } else {
      $(".ds-dependent").attr("data-preMatch", "false");
    }
  });
};

// Handles a websocket message to update the match score.
const handleRealtimeScore = function (data, reversed) {

  if (reversed === "true") {
    $("#rightScore").text(data.Red.ScoreSummary.Score);
    $("#leftScore").text(data.Blue.ScoreSummary.Score);
  } else {
    $("#rightScore").text(data.Blue.ScoreSummary.Score);
    $("#leftScore").text(data.Red.ScoreSummary.Score);

  }
};

// Handles a websocket message to update current match
const handleMatchLoad = function (data) {
  $("#matchName").text(data.Match.LongName);
};

// Handles a websocket message to update the event status message.
const handleEventStatus = function (data) {
  if (data.CycleTime === "") {
    $("#cycleTimeMessage").text("Last cycle time: Unknown");
  } else {
    $("#cycleTimeMessage").text("Last cycle time: " + data.CycleTime);
  }
  $("#earlyLateMessage").text(data.EarlyLateMessage);
};

// Makes the team notes section editable and handles saving edits to the server.
const editFtaNotes = function (element) {
  const teamNotesTextElement = $(element);
  const textArea = $("<textarea />");
  textArea.val(teamNotesTextElement.text());
  teamNotesTextElement.replaceWith(textArea);
  textArea.focus();
  textArea.blur(function () {
    textArea.replaceWith(teamNotesTextElement);
    if (textArea.val() !== teamNotesTextElement.text()) {
      websocket.send("updateTeamNotes", {station: teamNotesTextElement.attr("data-station"), notes: textArea.val()});
    }
  });
};

$(function () {
  // Read the configuration for this display from the URL query string.
  const urlParams = new URLSearchParams(window.location.search);
  const reversed = urlParams.get("reversed");
  if (reversed === "true") {
    redSide = "right";
    blueSide = "left";
  } else {
    redSide = "left";
    blueSide = "right";
  }

  //Read if display to be used in a Driver Station, ignore FTA flag if so.
  const driverStation = urlParams.get("ds");
  if (driverStation === "true") {
    $(".fta-dependent").attr("data-fta", "false");
    $(".ds-dependent").attr("data-ds", driverStation);
  } else {
    $(".fta-dependent").attr("data-fta", urlParams.get("fta"));
    $(".ds-dependent").attr("data-ds", driverStation);
  }

  $(".reversible-left").attr("data-reversed", reversed);
  $(".reversible-right").attr("data-reversed", reversed);


  // Set up the websocket back to the server.
  websocket = new CheesyWebsocket("/displays/field_monitor/websocket", {
    arenaStatus: function (event) {
      handleArenaStatus(event.data);
    },
    eventStatus: function (event) {
      handleEventStatus(event.data);
    },
    matchLoad: function (event) {
      handleMatchLoad(event.data);
    },
    matchTiming: function (event) {
      handleMatchTiming(event.data);
    },
    matchTime: function (event) {
      handleMatchTime(event.data);
    },
    realtimeScore: function (event) {
      handleRealtimeScore(event.data, reversed);
    },
  });
});
