// Copyright 2026 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Shared helpers for audience and wall displays.

(function (window) {
  window.DisplayShared = {
    applyDisplaySides: function (urlParams) {
      const reversed = urlParams.get("reversed");
      const redSide = reversed === "true" ? "right" : "left";
      const blueSide = reversed === "true" ? "left" : "right";
      $(".reversible-left").attr("data-reversed", reversed);
      $(".reversible-right").attr("data-reversed", reversed);
      return {redSide: redSide, blueSide: blueSide};
    },

    getAvatarUrl: function (teamId) {
      return "/api/teams/" + teamId + "/avatar";
    },

    handleMatchLoad: function (data, redSide, blueSide) {
      const currentMatch = data.Match;
      $(`#${redSide}Team1`).text(currentMatch.Red1);
      $(`#${redSide}Team1`).attr("data-yellow-card", data.Teams["R1"]?.YellowCard);
      $(`#${redSide}Team2`).text(currentMatch.Red2);
      $(`#${redSide}Team2`).attr("data-yellow-card", data.Teams["R2"]?.YellowCard);
      $(`#${redSide}Team1Avatar`).attr("src", this.getAvatarUrl(currentMatch.Red1));
      $(`#${redSide}Team2Avatar`).attr("src", this.getAvatarUrl(currentMatch.Red2));
      $(`#${blueSide}Team1`).text(currentMatch.Blue1);
      $(`#${blueSide}Team1`).attr("data-yellow-card", data.Teams["B1"]?.YellowCard);
      $(`#${blueSide}Team2`).text(currentMatch.Blue2);
      $(`#${blueSide}Team2`).attr("data-yellow-card", data.Teams["B2"]?.YellowCard);
      $(`#${blueSide}Team1Avatar`).attr("src", this.getAvatarUrl(currentMatch.Blue1));
      $(`#${blueSide}Team2Avatar`).attr("src", this.getAvatarUrl(currentMatch.Blue2));

      if (currentMatch.Type === matchTypePlayoff) {
        $(`#${redSide}PlayoffAlliance`).text(currentMatch.PlayoffRedAlliance);
        $(`#${blueSide}PlayoffAlliance`).text(currentMatch.PlayoffBlueAlliance);
        $(".playoff-alliance").show();

        if (data.Matchup.NumWinsToAdvance > 1) {
          $(`#${redSide}PlayoffAllianceWins`).text(data.Matchup.RedAllianceWins);
          $(`#${blueSide}PlayoffAllianceWins`).text(data.Matchup.BlueAllianceWins);
          $("#playoffSeriesStatus").css("display", "flex");
        } else {
          $("#playoffSeriesStatus").hide();
        }
      } else {
        $(`#${redSide}PlayoffAlliance`).text("");
        $(`#${blueSide}PlayoffAlliance`).text("");
        $(".playoff-alliance").hide();
        $("#playoffSeriesStatus").hide();
      }

      let matchName = data.Match.LongName;
      if (data.Match.NameDetail !== "") {
        matchName += " &ndash; " + data.Match.NameDetail;
      }
      $("#matchName").html(matchName);
      const timeoutNextMatchName = data.BreakNextMatchName || "";
      const timeoutDetailOpacity = $("#timeoutBreakDescription").css("opacity");
      $("#timeoutNextMatch").toggle(timeoutNextMatchName !== "").css(
        "opacity", timeoutNextMatchName === "" ? 0 : timeoutDetailOpacity
      );
      $("#timeoutNextMatchName").text(timeoutNextMatchName);
      $("#timeoutBreakDescription").text(data.BreakDescription);
      return currentMatch;
    },

    handleMatchTime: function (data) {
      translateMatchTime(data, function (matchState, matchStateText, countdownSec) {
        $("#matchTime").text(getCountdownString(countdownSec));
      });
    },

    handleRealtimeScore: function (data, redSide, blueSide) {
      $(`#${redSide}ScoreNumber`).text(data.Red.ScoreSummary.Score - data.Red.ScoreSummary.PostMatchPoints);
      $(`#${blueSide}ScoreNumber`).text(data.Blue.ScoreSummary.Score - data.Blue.ScoreSummary.PostMatchPoints);
      $(`#${redSide}ScoreNumber`).closest(".score").attr("data-generic", true);
      $(`#${blueSide}ScoreNumber`).closest(".score").attr("data-generic", true);
      $(".score-fields").hide();
    },
  };
})(window);
