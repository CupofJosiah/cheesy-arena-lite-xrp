// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side methods for editing a generic match result.

const allianceResults = {};
let matchResult;

const SUMMARY_REFRESH_DELAY_MS = 150;
let summaryRefreshTimer;
let latestSummaryRequestId = 0;

$("form").submit(function () {
  updateAllResults();

  const matchResultJson = JSON.stringify(matchResult);
  $("input[name=matchResultJson]").remove();
  $("<input />").attr("type", "hidden").attr("name", "matchResultJson").attr("value", matchResultJson).appendTo("form");

  return true;
});

$("form").on("input change", "input, select", function () {
  scheduleScoreSummaryRefresh();
});

// Values stored on a match result, matching the fields of game.Score. BonusPoints is points rather than a count and
// may be negative; the form and parseFormInt both handle that already.
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
  "MinorPenalties",
  "MajorPenalties",
];

const renderResults = function (alliance) {
  const result = allianceResults[alliance];
  result.score = normalizeScore(result.score);
  result.cards = result.cards || {};

  scoreFields.forEach((field) => {
    getInputElement(alliance, field).val(result.score[field]);
  });
  renderCards(alliance);
};

const updateResults = function (alliance) {
  const result = allianceResults[alliance];
  const formData = {};
  $.each($("form").serializeArray(), function (k, v) {
    formData[v.name] = v.value;
  });

  scoreFields.forEach((field) => {
    result.score[field] = parseFormInt(formData[`${alliance}${field}`]);
  });

  result.cards = {};
  $.each(result.teams, function (i, team) {
    result.cards[team] = formData[`${alliance}Team${team}Card`];
  });
};

const updateAllResults = function () {
  updateResults("red");
  updateResults("blue");

  matchResult.RedScore = allianceResults["red"].score;
  matchResult.BlueScore = allianceResults["blue"].score;
  matchResult.RedCards = allianceResults["red"].cards;
  matchResult.BlueCards = allianceResults["blue"].cards;
};

const renderCards = function (alliance) {
  const result = allianceResults[alliance];
  $.each(result.cards, function (team, card) {
    getInputElement(alliance, `Team${team}Card`, card).prop("checked", true);
  });
};

const scheduleScoreSummaryRefresh = function () {
  window.clearTimeout(summaryRefreshTimer);
  summaryRefreshTimer = window.setTimeout(refreshScoreSummaries, SUMMARY_REFRESH_DELAY_MS);
};

const refreshScoreSummaries = function () {
  updateAllResults();
  const requestId = ++latestSummaryRequestId;

  $.ajax({
    url: `/match_review/${matchId}/summary`,
    method: "POST",
    contentType: "application/json",
    data: JSON.stringify(matchResult),
    success: function (data) {
      if (requestId !== latestSummaryRequestId) {
        return;
      }
      updateSummaryCard("red", data.RedSummary);
      updateSummaryCard("blue", data.BlueSummary);
    },
  });
};

const updateSummaryCard = function (alliance, summary) {
  const summaryCard = $(`#${alliance}Summary`);
  $.each(summary, function (field, value) {
    summaryCard.find(`[data-summary-field=${field}]`).html(value);
  });
};

const getInputElement = function (alliance, name, value) {
  let selector = `input[name=${alliance}${name}]`;
  if (value !== undefined) {
    selector += `[value=${value}]`;
  }
  return $(selector);
};

const normalizeScore = function (score) {
  score = score || {};
  scoreFields.forEach((field) => {
    score[field] = parseFormInt(score[field]);
  });
  score.PlayoffDq = !!score.PlayoffDq;
  return score;
};

const parseFormInt = function (value) {
  const parsed = parseInt(value, 10);
  if (isNaN(parsed)) {
    return 0;
  }
  return parsed;
};
