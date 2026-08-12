// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web API for providing JSON-formatted event data.

package web

import (
	"encoding/json"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/playoff"
	"github.com/Team254/cheesy-arena-lite/websocket"
	"io"
	"net/http"
)

type MatchResultWithSummary struct {
	model.MatchResult
	RedSummary  *game.ScoreSummary
	BlueSummary *game.ScoreSummary
}

type MatchWithResult struct {
	model.Match
	Result *MatchResultWithSummary
}

type RankingWithNickname struct {
	game.Ranking
	Nickname     string
	TeleopPoints int
}

type allianceMatchup struct {
	Id                 string
	RedAllianceSource  string
	BlueAllianceSource string
	RedAlliance        *model.Alliance
	BlueAlliance       *model.Alliance
	IsActive           bool
	SeriesLeader       string
	SeriesStatus       string
	IsComplete         bool
}

// Counts of each scoring element, along with the point totals they add up to.
type apiAllianceScore struct {
	FactoryParks       int `json:"factoryParks"`
	AutoCrops          int `json:"autoCrops"`
	SilosDumped        int `json:"silosDumped"`
	TeleopCrops        int `json:"teleopCrops"`
	CityLimitsProducts int `json:"cityLimitsProducts"`
	CityCenterProducts int `json:"cityCenterProducts"`
	BarnParks          int `json:"barnParks"`
	BarnHangs          int `json:"barnHangs"`
	MinorPenalties     int `json:"minorPenalties"`
	MajorPenalties     int `json:"majorPenalties"`
	AutoPoints         int `json:"autoPoints"`
	TeleopPoints       int `json:"teleopPoints"`
	EndgamePoints      int `json:"endgamePoints"`
	PenaltyPoints      int `json:"penaltyPoints"`
}

type apiScore struct {
	Red  apiAllianceScore `json:"red"`
	Blue apiAllianceScore `json:"blue"`
}

type apiAllianceScorePatch struct {
	FactoryParks       *int `json:"factoryParks"`
	AutoCrops          *int `json:"autoCrops"`
	SilosDumped        *int `json:"silosDumped"`
	TeleopCrops        *int `json:"teleopCrops"`
	CityLimitsProducts *int `json:"cityLimitsProducts"`
	CityCenterProducts *int `json:"cityCenterProducts"`
	BarnParks          *int `json:"barnParks"`
	BarnHangs          *int `json:"barnHangs"`
	MinorPenalties     *int `json:"minorPenalties"`
	MajorPenalties     *int `json:"majorPenalties"`
}

type apiScorePatch struct {
	Red  apiAllianceScorePatch `json:"red"`
	Blue apiAllianceScorePatch `json:"blue"`
}

func newApiAllianceScore(score *game.Score) apiAllianceScore {
	return apiAllianceScore{
		FactoryParks:       score.FactoryParks,
		AutoCrops:          score.AutoCrops,
		SilosDumped:        score.SilosDumped,
		TeleopCrops:        score.TeleopCrops,
		CityLimitsProducts: score.CityLimitsProducts,
		CityCenterProducts: score.CityCenterProducts,
		BarnParks:          score.BarnParks,
		BarnHangs:          score.BarnHangs,
		MinorPenalties:     score.MinorPenalties,
		MajorPenalties:     score.MajorPenalties,
		AutoPoints:         score.AutoPoints(),
		TeleopPoints:       score.TeleopPoints(),
		EndgamePoints:      score.EndgamePoints(),
		PenaltyPoints:      score.PenaltyPoints(),
	}
}

// Overwrites the score with the given element counts, ignoring any negative values.
func applyApiAllianceScore(score *game.Score, apiScore apiAllianceScore) {
	score.FactoryParks = max(apiScore.FactoryParks, 0)
	score.AutoCrops = max(apiScore.AutoCrops, 0)
	score.SilosDumped = max(apiScore.SilosDumped, 0)
	score.TeleopCrops = max(apiScore.TeleopCrops, 0)
	score.CityLimitsProducts = max(apiScore.CityLimitsProducts, 0)
	score.CityCenterProducts = max(apiScore.CityCenterProducts, 0)
	score.BarnParks = max(apiScore.BarnParks, 0)
	score.BarnHangs = max(apiScore.BarnHangs, 0)
	score.MinorPenalties = max(apiScore.MinorPenalties, 0)
	score.MajorPenalties = max(apiScore.MajorPenalties, 0)
}

func (web *Web) currentApiScore() apiScore {
	return apiScore{
		Red:  newApiAllianceScore(&web.arena.RedRealtimeScore.CurrentScore),
		Blue: newApiAllianceScore(&web.arena.BlueRealtimeScore.CurrentScore),
	}
}

func (web *Web) scoresApiHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
	case http.MethodPut:
		var score apiScore
		if err := json.NewDecoder(r.Body).Decode(&score); err != nil {
			handleWebErr(w, err)
			return
		}
		applyApiAllianceScore(&web.arena.RedRealtimeScore.CurrentScore, score.Red)
		applyApiAllianceScore(&web.arena.BlueRealtimeScore.CurrentScore, score.Blue)
		web.arena.RealtimeScoreNotifier.Notify()
	case http.MethodPatch:
		var patch apiScorePatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			handleWebErr(w, err)
			return
		}
		applyScorePatch(&web.arena.RedRealtimeScore.CurrentScore, patch.Red)
		applyScorePatch(&web.arena.BlueRealtimeScore.CurrentScore, patch.Blue)
		web.arena.RealtimeScoreNotifier.Notify()
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(web.currentApiScore()); err != nil {
		handleWebErr(w, err)
		return
	}
}

// Adds the given deltas to the score, clamping each element at zero.
func applyScorePatch(score *game.Score, patch apiAllianceScorePatch) {
	for _, field := range []struct {
		delta *int
		value *int
	}{
		{patch.FactoryParks, &score.FactoryParks},
		{patch.AutoCrops, &score.AutoCrops},
		{patch.SilosDumped, &score.SilosDumped},
		{patch.TeleopCrops, &score.TeleopCrops},
		{patch.CityLimitsProducts, &score.CityLimitsProducts},
		{patch.CityCenterProducts, &score.CityCenterProducts},
		{patch.BarnParks, &score.BarnParks},
		{patch.BarnHangs, &score.BarnHangs},
		{patch.MinorPenalties, &score.MinorPenalties},
		{patch.MajorPenalties, &score.MajorPenalties},
	} {
		if field.delta != nil {
			*field.value = max(*field.value+*field.delta, 0)
		}
	}
}

// Generates a JSON dump of the matches and results.
func (web *Web) matchesApiHandler(w http.ResponseWriter, r *http.Request) {
	matchType, err := model.MatchTypeFromString(r.PathValue("type"))
	if err != nil {
		handleWebErr(w, err)
		return
	}

	matches, err := web.arena.Database.GetMatchesByType(matchType, false)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	matchesWithResults := make([]MatchWithResult, len(matches))
	for i, match := range matches {
		matchesWithResults[i].Match = match
		matchResult, err := web.arena.Database.GetMatchResultForMatch(match.Id)
		if err != nil {
			handleWebErr(w, err)
			return
		}
		var matchResultWithSummary *MatchResultWithSummary
		if matchResult != nil {
			matchResultWithSummary = &MatchResultWithSummary{MatchResult: *matchResult}
			matchResultWithSummary.RedSummary = matchResult.RedScoreSummary()
			matchResultWithSummary.BlueSummary = matchResult.BlueScoreSummary()
		}
		matchesWithResults[i].Result = matchResultWithSummary
	}

	jsonData, err := json.MarshalIndent(matchesWithResults, "", "  ")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	_, err = w.Write(jsonData)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Generates a JSON dump of the sponsor slides for use by the audience display.
func (web *Web) sponsorSlidesApiHandler(w http.ResponseWriter, r *http.Request) {
	sponsors, err := web.arena.Database.GetAllSponsorSlides()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	if sponsors == nil {
		// Go marshals an empty slice to null, so explicitly create it so that it appears as an empty JSON array.
		sponsors = make([]model.SponsorSlide, 0)
	}
	jsonData, err := json.MarshalIndent(sponsors, "", "  ")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	_, err = w.Write(jsonData)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Generates a JSON dump of the qualification rankings, primarily for use by the rankings display.
func (web *Web) rankingsApiHandler(w http.ResponseWriter, r *http.Request) {
	rankings, err := web.arena.Database.GetAllRankings()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	var rankingsWithNicknames []RankingWithNickname
	if rankings == nil {
		// Go marshals an empty slice to null, so explicitly create it so that it appears as an empty JSON array.
		rankingsWithNicknames = make([]RankingWithNickname, 0)
	} else {
		rankingsWithNicknames = make([]RankingWithNickname, len(rankings))
	}

	// Get team info so that nicknames can be displayed.
	teams, err := web.arena.Database.GetAllTeams()
	if err != nil {
		handleWebErr(w, err)
		return
	}
	teamNicknames := make(map[game.TeamId]string)
	for _, team := range teams {
		teamNicknames[team.Id] = team.Nickname
	}
	for i, ranking := range rankings {
		rankingsWithNicknames[i] = RankingWithNickname{ranking, teamNicknames[ranking.TeamId], ranking.TeleopPoints()}
	}

	// Get the last match scored so we can report that on the display.
	matches, err := web.arena.Database.GetMatchesByType(model.Qualification, false)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	var highestPlayedMatch model.Match
	for _, match := range matches {
		if match.IsComplete() {
			highestPlayedMatch = match
		}
	}

	data := struct {
		Rankings           []RankingWithNickname
		HighestPlayedMatch string
	}{rankingsWithNicknames, highestPlayedMatch.ShortName}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	_, err = w.Write(jsonData)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Generates a JSON dump of the alliances.
func (web *Web) alliancesApiHandler(w http.ResponseWriter, r *http.Request) {
	alliances, err := web.arena.Database.GetAllAlliances()
	if err != nil {
		handleWebErr(w, err)
		return
	}

	jsonData, err := json.MarshalIndent(alliances, "", "  ")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	_, err = w.Write(jsonData)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Websocket API for receiving arena status updates.
func (web *Web) arenaWebsocketApiHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer closeWebsocket(ws)

	// Subscribe the websocket to the notifiers whose messages will be passed on to the client.
	ws.HandleNotifiers(web.arena.MatchTimingNotifier, web.arena.MatchLoadNotifier, web.arena.MatchTimeNotifier)
}

func (web *Web) bracketSvgApiHandler(w http.ResponseWriter, r *http.Request) {
	var activeMatch *model.Match
	if activeMatchValue, ok := r.URL.Query()["activeMatch"]; ok {
		if activeMatchValue[0] == "current" {
			activeMatch = web.arena.CurrentMatch
		} else if activeMatchValue[0] == "saved" {
			activeMatch = web.arena.SavedMatch
		}
	}

	w.Header().Add("Content-Type", "image/svg+xml")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	if err := web.generateBracketSvg(w, activeMatch); err != nil {
		handleWebErr(w, err)
		return
	}
}

func (web *Web) generateBracketSvg(w io.Writer, activeMatch *model.Match) error {
	alliances, err := web.arena.Database.GetAllAlliances()
	if err != nil {
		return err
	}

	matchups := make(map[string]*allianceMatchup)
	if web.arena.PlayoffTournament != nil {
		for _, matchGroup := range web.arena.PlayoffTournament.MatchGroups() {
			matchup, ok := matchGroup.(*playoff.Matchup)
			if !ok {
				continue
			}
			allianceMatchup := allianceMatchup{
				Id:                 matchup.Id(),
				RedAllianceSource:  matchup.RedAllianceSourceDisplayName(),
				BlueAllianceSource: matchup.BlueAllianceSourceDisplayName(),
				IsComplete:         matchup.IsComplete(),
			}
			if matchup.RedAllianceId > 0 {
				if len(alliances) > 0 {
					allianceMatchup.RedAlliance = &alliances[matchup.RedAllianceId-1]
				} else {
					allianceMatchup.RedAlliance = &model.Alliance{Id: matchup.RedAllianceId}
				}
			}
			if matchup.BlueAllianceId > 0 {
				if len(alliances) > 0 {
					allianceMatchup.BlueAlliance = &alliances[matchup.BlueAllianceId-1]
				} else {
					allianceMatchup.BlueAlliance = &model.Alliance{Id: matchup.BlueAllianceId}
				}
			}
			if activeMatch != nil {
				allianceMatchup.IsActive = activeMatch.PlayoffMatchGroupId == matchup.Id()
			}
			allianceMatchup.SeriesLeader, allianceMatchup.SeriesStatus = matchup.StatusText()
			matchups[matchup.Id()] = &allianceMatchup
		}
	}

	bracketType := "double"
	numAlliances := web.arena.EventSettings.NumPlayoffAlliances
	if web.arena.EventSettings.PlayoffType == model.DoubleEliminationPlayoff && numAlliances == 4 {
		bracketType = "double4"
	} else if web.arena.EventSettings.PlayoffType == model.SingleEliminationPlayoff {
		if numAlliances > 8 {
			bracketType = "16"
		} else if numAlliances > 4 {
			bracketType = "8"
		} else if numAlliances > 2 {
			bracketType = "4"
		} else {
			bracketType = "2"
		}
	}

	template, err := web.parseFiles("templates/bracket.svg")
	if err != nil {
		return err
	}
	data := struct {
		BracketType string
		Matchups    map[string]*allianceMatchup
	}{bracketType, matchups}
	return template.ExecuteTemplate(w, "bracket", data)
}
