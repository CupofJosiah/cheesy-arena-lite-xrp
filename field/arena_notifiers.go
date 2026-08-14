// Copyright 2018 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Contains configuration of the publish-subscribe notifiers that allow the arena to push updates to websocket clients.

package field

import (
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/playoff"
	"github.com/Team254/cheesy-arena-lite/websocket"
	"log"
)

type ArenaNotifiers struct {
	AllianceSelectionNotifier          *websocket.Notifier
	AllianceStationDisplayModeNotifier *websocket.Notifier
	ArenaStatusNotifier                *websocket.Notifier
	AudienceDisplayModeNotifier        *websocket.Notifier
	DisplayConfigurationNotifier       *websocket.Notifier
	EventStatusNotifier                *websocket.Notifier
	LowerThirdNotifier                 *websocket.Notifier
	MatchLoadNotifier                  *websocket.Notifier
	MatchTimeNotifier                  *websocket.Notifier
	MatchTimingNotifier                *websocket.Notifier
	PlaySoundNotifier                  *websocket.Notifier
	RealtimeScoreNotifier              *websocket.Notifier
	ReloadDisplaysNotifier             *websocket.Notifier
	ScorePostedNotifier                *websocket.Notifier
	ScoringStatusNotifier              *websocket.Notifier
}

type MatchTimeMessage struct {
	MatchState
	MatchTimeSec int
}

type audienceAllianceScoreFields struct {
	Score        *game.Score
	ScoreSummary *game.ScoreSummary
}

// Instantiates notifiers and configures their message producing methods.
func (arena *Arena) configureNotifiers() {
	arena.AllianceSelectionNotifier = websocket.NewNotifier("allianceSelection", arena.generateAllianceSelectionMessage)
	arena.AllianceStationDisplayModeNotifier = websocket.NewNotifier(
		"allianceStationDisplayMode", arena.generateAllianceStationDisplayModeMessage,
	)
	arena.ArenaStatusNotifier = websocket.NewNotifier("arenaStatus", arena.generateArenaStatusMessage)
	arena.AudienceDisplayModeNotifier = websocket.NewNotifier(
		"audienceDisplayMode", arena.generateAudienceDisplayModeMessage,
	)
	arena.DisplayConfigurationNotifier = websocket.NewNotifier(
		"displayConfiguration", arena.generateDisplayConfigurationMessage,
	)
	arena.EventStatusNotifier = websocket.NewNotifier("eventStatus", arena.generateEventStatusMessage)
	arena.LowerThirdNotifier = websocket.NewNotifier("lowerThird", arena.generateLowerThirdMessage)
	arena.MatchLoadNotifier = websocket.NewNotifier("matchLoad", arena.GenerateMatchLoadMessage)
	arena.MatchTimeNotifier = websocket.NewNotifier("matchTime", arena.generateMatchTimeMessage)
	arena.MatchTimingNotifier = websocket.NewNotifier("matchTiming", arena.generateMatchTimingMessage)
	arena.PlaySoundNotifier = websocket.NewNotifier("playSound", nil)
	arena.RealtimeScoreNotifier = websocket.NewNotifier("realtimeScore", arena.generateRealtimeScoreMessage)
	arena.ReloadDisplaysNotifier = websocket.NewNotifier("reload", nil)
	arena.ScorePostedNotifier = websocket.NewNotifier("scorePosted", arena.GenerateScorePostedMessage)
	arena.ScoringStatusNotifier = websocket.NewNotifier("scoringStatus", arena.generateScoringStatusMessage)
}

func (arena *Arena) generateAllianceSelectionMessage() any {
	return &struct {
		Alliances        []model.Alliance
		ShowTimer        bool
		TimeRemainingSec int
		RankedTeams      []model.AllianceSelectionRankedTeam
	}{
		arena.AllianceSelectionAlliances,
		arena.AllianceSelectionShowTimer,
		arena.AllianceSelectionTimeRemainingSec,
		arena.AllianceSelectionRankedTeams,
	}
}

func (arena *Arena) generateAllianceStationDisplayModeMessage() any {
	return arena.AllianceStationDisplayMode
}

func (arena *Arena) generateArenaStatusMessage() any {
	return &struct {
		MatchId          int
		AllianceStations map[string]*AllianceStation
		MatchState
		CanStartMatch bool
	}{
		arena.CurrentMatch.Id,
		arena.AllianceStations,
		arena.MatchState,
		arena.checkCanStartMatch() == nil,
	}
}

func (arena *Arena) generateAudienceDisplayModeMessage() any {
	return arena.AudienceDisplayMode
}

func (arena *Arena) generateDisplayConfigurationMessage() any {
	// Notify() for this notifier must always called from a method that has a lock on the display mutex.
	// Make a copy of the map to avoid potential data races; otherwise the same map would get iterated through as it is
	// serialized to JSON, outside the mutex lock.
	displaysCopy := make(map[string]Display)
	for displayId, display := range arena.Displays {
		displaysCopy[displayId] = *display
	}
	return displaysCopy
}

func (arena *Arena) generateEventStatusMessage() any {
	return arena.EventStatus
}

func (arena *Arena) generateLowerThirdMessage() any {
	return &struct {
		LowerThird     *model.LowerThird
		ShowLowerThird bool
	}{arena.LowerThird, arena.ShowLowerThird}
}

func (arena *Arena) GenerateMatchLoadMessage() any {
	teams := make(map[string]*model.Team)
	var allTeamIds []game.TeamId
	for station, allianceStation := range arena.AllianceStations {
		teams[station] = allianceStation.Team
		if allianceStation.Team != nil {
			allTeamIds = append(allTeamIds, allianceStation.Team.Id)
		}
	}

	matchResult, err := arena.Database.GetMatchResultForMatch(arena.CurrentMatch.Id)
	if err != nil {
		log.Printf("Failed to get match result for match %d while generating match load message: %v",
			arena.CurrentMatch.Id, err)
	}
	isReplay := matchResult != nil

	var matchup *playoff.Matchup
	redOffFieldTeams := []*model.Team{}
	blueOffFieldTeams := []*model.Team{}
	if arena.CurrentMatch.Type == model.Playoff {
		matchGroup := arena.PlayoffTournament.MatchGroups()[arena.CurrentMatch.PlayoffMatchGroupId]
		matchup, _ = matchGroup.(*playoff.Matchup)
		redOffFieldTeamIds, blueOffFieldTeamIds, err := arena.Database.GetOffFieldTeamIds(arena.CurrentMatch)
		if err != nil {
			log.Printf("Failed to get off-field teams for match %d while generating match load message: %v",
				arena.CurrentMatch.Id, err)
		}
		for _, teamId := range redOffFieldTeamIds {
			team, err := arena.Database.GetTeamById(teamId)
			if err != nil {
				log.Printf("Failed to get red off-field team %s while generating match load message: %v", teamId, err)
			}
			redOffFieldTeams = append(redOffFieldTeams, team)
			allTeamIds = append(allTeamIds, teamId)
		}
		for _, teamId := range blueOffFieldTeamIds {
			team, err := arena.Database.GetTeamById(teamId)
			if err != nil {
				log.Printf("Failed to get blue off-field team %s while generating match load message: %v", teamId, err)
			}
			blueOffFieldTeams = append(blueOffFieldTeams, team)
			allTeamIds = append(allTeamIds, teamId)
		}
	}

	rankings := make(map[string]int)
	for _, teamId := range allTeamIds {
		ranking, err := arena.Database.GetRankingForTeam(teamId)
		if err != nil {
			log.Printf("Failed to get ranking for team %s while generating match load message: %v", teamId, err)
		}
		if ranking != nil {
			rankings[string(teamId)] = ranking.Rank
		}
	}

	var allowManualSubstitution = arena.CurrentMatch.ShouldAllowSubstitution()

	return &struct {
		Match              *model.Match
		AllowSubstitution  bool
		IsReplay           bool
		Teams              map[string]*model.Team
		Rankings           map[string]int
		Matchup            *playoff.Matchup
		RedOffFieldTeams   []*model.Team
		BlueOffFieldTeams  []*model.Team
		BreakDescription   string
		BreakNextMatchName string
	}{
		arena.CurrentMatch,
		allowManualSubstitution,
		isReplay,
		teams,
		rankings,
		matchup,
		redOffFieldTeams,
		blueOffFieldTeams,
		arena.breakDescription,
		arena.breakNextMatchName,
	}
}

func (arena *Arena) generateMatchTimeMessage() any {
	return MatchTimeMessage{arena.MatchState, int(arena.MatchTimeSec())}
}

func (arena *Arena) generateMatchTimingMessage() any {
	return &game.MatchTiming
}

func (arena *Arena) generateRealtimeScoreMessage() any {
	fields := struct {
		Red       *audienceAllianceScoreFields
		Blue      *audienceAllianceScoreFields
		RedCards  map[string]string
		BlueCards map[string]string
		MatchState
	}{
		getAudienceAllianceScoreFields(arena.RedRealtimeScore, arena.RedScoreSummary()),
		getAudienceAllianceScoreFields(arena.BlueRealtimeScore, arena.BlueScoreSummary()),
		arena.RedRealtimeScore.Cards,
		arena.BlueRealtimeScore.Cards,
		arena.MatchState,
	}
	return &fields
}

func (arena *Arena) GenerateScorePostedMessage() any {
	redScoreSummary := arena.SavedMatchResult.RedScoreSummary()
	blueScoreSummary := arena.SavedMatchResult.BlueScoreSummary()
	_, tiebreakReason := game.DetermineMatchStatus(
		redScoreSummary, blueScoreSummary, arena.SavedMatch.UseTiebreakCriteria,
	)
	redRankingPoints := redScoreSummary.WinRankingPoints
	blueRankingPoints := blueScoreSummary.WinRankingPoints

	// For playoff matches, summarize the state of the series.
	var redWins, blueWins int
	var redDestination, blueDestination string
	redOffFieldTeamIds := []game.TeamId{}
	blueOffFieldTeamIds := []game.TeamId{}
	if arena.SavedMatch.Type == model.Playoff {
		matchGroup := arena.PlayoffTournament.MatchGroups()[arena.SavedMatch.PlayoffMatchGroupId]
		if matchup, ok := matchGroup.(*playoff.Matchup); ok {
			redWins = matchup.RedAllianceWins
			blueWins = matchup.BlueAllianceWins
			redDestination = matchup.RedAllianceDestination()
			blueDestination = matchup.BlueAllianceDestination()
		}
		var err error
		redOffFieldTeamIds, blueOffFieldTeamIds, err = arena.Database.GetOffFieldTeamIds(arena.SavedMatch)
		if err != nil {
			log.Printf("Failed to get off-field teams for match %d while generating score posted message: %v",
				arena.SavedMatch.Id, err)
		}
	}

	// Flag a new event high score, which is only meaningful for qualification matches since playoff alliances are not
	// comparable with them. Only the top score in the match is flagged, so that an alliance which beat the old record
	// but was still outscored does not also claim it; a tie at the top flags both.
	var redHighScore, blueHighScore bool
	if arena.SavedMatch.Type == model.Qualification {
		previousHighScore, previousMatchesPlayed, err := arena.Database.GetHighestQualificationScore(
			arena.SavedMatch.Id,
		)
		if err != nil {
			log.Printf("Failed to get the high score for match %d while generating score posted message: %v",
				arena.SavedMatch.Id, err)
		} else if previousMatchesPlayed > 0 {
			topScore := max(redScoreSummary.Score, blueScoreSummary.Score)
			if topScore > previousHighScore {
				redHighScore = redScoreSummary.Score == topScore
				blueHighScore = blueScoreSummary.Score == topScore
			}
		}
	}

	redRankings := map[game.TeamId]*game.Ranking{
		arena.SavedMatch.Red1: nil, arena.SavedMatch.Red2: nil,
	}
	blueRankings := map[game.TeamId]*game.Ranking{
		arena.SavedMatch.Blue1: nil, arena.SavedMatch.Blue2: nil,
	}
	for index, ranking := range arena.SavedRankings {
		if _, ok := redRankings[ranking.TeamId]; ok {
			redRankings[ranking.TeamId] = &arena.SavedRankings[index]
		}
		if _, ok := blueRankings[ranking.TeamId]; ok {
			blueRankings[ranking.TeamId] = &arena.SavedRankings[index]
		}
	}

	return &struct {
		Match               *model.Match
		RedScoreSummary     *game.ScoreSummary
		BlueScoreSummary    *game.ScoreSummary
		RedRankingPoints    int
		BlueRankingPoints   int
		RedCards            map[string]string
		BlueCards           map[string]string
		RedRankings         map[game.TeamId]*game.Ranking
		BlueRankings        map[game.TeamId]*game.Ranking
		RedOffFieldTeamIds  []game.TeamId
		BlueOffFieldTeamIds []game.TeamId
		RedWon              bool
		BlueWon             bool
		RedHighScore        bool
		BlueHighScore       bool
		TiebreakReason      string
		RedWins             int
		BlueWins            int
		RedDestination      string
		BlueDestination     string
	}{
		arena.SavedMatch,
		redScoreSummary,
		blueScoreSummary,
		redRankingPoints,
		blueRankingPoints,
		arena.SavedMatchResult.RedCards,
		arena.SavedMatchResult.BlueCards,
		redRankings,
		blueRankings,
		redOffFieldTeamIds,
		blueOffFieldTeamIds,
		arena.SavedMatch.Status == game.RedWonMatch,
		arena.SavedMatch.Status == game.BlueWonMatch,
		redHighScore,
		blueHighScore,
		tiebreakReason,
		redWins,
		blueWins,
		redDestination,
		blueDestination,
	}
}

func (arena *Arena) generateScoringStatusMessage() any {
	return &struct {
		RefereeScoreReady     bool
		ScoringReady          bool
		NumScoringPanels      int
		NumScoringPanelsReady int
	}{
		arena.RedRealtimeScore.FoulsCommitted && arena.BlueRealtimeScore.FoulsCommitted,
		arena.positionPostMatchScoreReady("scoring"),
		arena.ScoringPanelRegistry.GetNumPanels("scoring"),
		arena.GetNumScoreCommitted("scoring"),
	}
}

// Constructs the data object for one alliance sent to the audience display for the realtime scoring overlay.
func getAudienceAllianceScoreFields(
	allianceScore *RealtimeScore,
	allianceScoreSummary *game.ScoreSummary,
) *audienceAllianceScoreFields {
	fields := new(audienceAllianceScoreFields)
	fields.Score = &allianceScore.CurrentScore
	fields.ScoreSummary = allianceScoreSummary
	return fields
}
