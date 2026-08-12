// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Functions for controlling the arena and match play.

package field

import (
	"fmt"
	"github.com/Team254/cheesy-arena-lite/game"
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/Team254/cheesy-arena-lite/partner"
	"github.com/Team254/cheesy-arena-lite/playoff"
	"log"
	"strings"
	"time"
)

const (
	arenaLoopPeriodMs      = 10
	arenaLoopWarningMs     = 5
	periodicTaskPeriodSec  = 15
	matchEndScoreDwellSec  = 3
	postTimeoutSec         = 4
	showNextMatchDelaySec  = 5
	scheduledBreakDelaySec = 5
	earlyLateThresholdMin  = 2.5
	MaxMatchGapMin         = 20
)

// AllianceStationIds enumerates the driver station positions in a fixed order, red alliance first.
var AllianceStationIds = []string{"R1", "R2", "B1", "B2"}

// RedAllianceStationIds and BlueAllianceStationIds are the per-alliance subsets of AllianceStationIds.
var (
	RedAllianceStationIds  = []string{"R1", "R2"}
	BlueAllianceStationIds = []string{"B1", "B2"}
)

// Progression of match states.
type MatchState int

const (
	PreMatch MatchState = iota
	StartMatch
	AutoPeriod
	PausePeriod
	TeleopPeriod
	PostMatch
	TimeoutActive
	PostTimeout
)

type Arena struct {
	Database         *model.Database
	EventSettings    *model.EventSettings
	BlackmagicClient *partner.BlackmagicClient
	CompanionClient  *partner.CompanionClient
	AllianceStations map[string]*AllianceStation
	Displays         map[string]*Display
	TeamSigns        *TeamSigns
	ScoringPanelRegistry
	ArenaNotifiers
	MatchState
	lastMatchState                    MatchState
	CurrentMatch                      *model.Match
	MatchStartTime                    time.Time
	LastMatchTimeSec                  float64
	RedRealtimeScore                  *RealtimeScore
	BlueRealtimeScore                 *RealtimeScore
	lastPeriodicTaskTime              time.Time
	EventStatus                       EventStatus
	FieldVolunteers                   bool
	FieldReset                        bool
	AudienceDisplayMode               string
	SavedMatch                        *model.Match
	SavedMatchResult                  *model.MatchResult
	SavedRankings                     game.Rankings
	AllianceStationDisplayMode        string
	AllianceSelectionAlliances        []model.Alliance
	AllianceSelectionRankedTeams      []model.AllianceSelectionRankedTeam
	AllianceSelectionShowTimer        bool
	AllianceSelectionTimeRemainingSec int
	PlayoffTournament                 *playoff.PlayoffTournament
	LowerThird                        *model.LowerThird
	ShowLowerThird                    bool
	MuteMatchSounds                   bool
	matchAborted                      bool
	soundsPlayed                      map[*game.MatchSound]struct{}
	breakDescription                  string
	breakNextMatchName                string
}

// AllianceStation represents one driver station position on the field. XRP robots are driven directly from each
// team's own controller rather than through the field, so the arena only tracks who is assigned where and whether
// the position is ready for a match to start.
type AllianceStation struct {
	// Astop and Estop are set by the referee to stop a robot for safety or field damage, per rule S03.
	AStop  bool
	EStop  bool
	Bypass bool
	Ready  bool
	Team   *model.Team
}

// Creates the arena and sets it to its initial state.
func NewArena(dbPath string) (*Arena, error) {
	arena := new(Arena)
	arena.configureNotifiers()

	arena.AllianceStations = make(map[string]*AllianceStation)
	for _, station := range AllianceStationIds {
		arena.AllianceStations[station] = new(AllianceStation)
	}

	arena.Displays = make(map[string]*Display)

	arena.TeamSigns = NewTeamSigns()

	var err error
	arena.Database, err = model.OpenDatabase(dbPath)
	if err != nil {
		return nil, err
	}
	err = arena.LoadSettings()
	if err != nil {
		return nil, err
	}

	arena.ScoringPanelRegistry.initialize()

	// Load empty match as current.
	arena.MatchState = PreMatch
	arena.LoadTestMatch()
	arena.LastMatchTimeSec = 0
	arena.lastMatchState = -1

	// Initialize display parameters.
	arena.AudienceDisplayMode = "blank"
	arena.SavedMatch = &model.Match{}
	arena.SavedMatchResult = model.NewMatchResult()
	arena.AllianceStationDisplayMode = "match"

	return arena, nil
}

// Loads or reloads the event settings upon initial setup or change.
func (arena *Arena) LoadSettings() error {
	settings, err := arena.Database.GetEventSettings()
	if err != nil {
		return err
	}
	arena.EventSettings = settings

	// Initialize the components that depend on settings.
	arena.TeamSigns.Red1.SetId(settings.TeamSignRed1Id)
	arena.TeamSigns.Red2.SetId(settings.TeamSignRed2Id)
	arena.TeamSigns.RedTimer.SetId(settings.TeamSignRedTimerId)
	arena.TeamSigns.Blue1.SetId(settings.TeamSignBlue1Id)
	arena.TeamSigns.Blue2.SetId(settings.TeamSignBlue2Id)
	arena.TeamSigns.BlueTimer.SetId(settings.TeamSignBlueTimerId)
	arena.BlackmagicClient = partner.NewBlackmagicClient(settings.BlackmagicAddresses)

	// Initialize Companion client with event configurations
	companionEventConfigs := map[partner.CompanionEvent]partner.CompanionEventConfig{
		partner.EventMatchPreview: {
			Page:   settings.CompanionMatchPreviewPage,
			Row:    settings.CompanionMatchPreviewRow,
			Column: settings.CompanionMatchPreviewColumn,
		},
		partner.EventShowOverlay: {
			Page:   settings.CompanionSetAudiencePage,
			Row:    settings.CompanionSetAudienceRow,
			Column: settings.CompanionSetAudienceColumn,
		},
		partner.EventMatchStart: {
			Page:   settings.CompanionMatchStartPage,
			Row:    settings.CompanionMatchStartRow,
			Column: settings.CompanionMatchStartColumn,
		},
		partner.EventTeleopStart: {
			Page:   settings.CompanionTeleopStartPage,
			Row:    settings.CompanionTeleopStartRow,
			Column: settings.CompanionTeleopStartColumn,
		},
		partner.EventMatchEnd: {
			Page:   settings.CompanionMatchEndPage,
			Row:    settings.CompanionMatchEndRow,
			Column: settings.CompanionMatchEndColumn,
		},
		partner.EventShowFinalScore: {
			Page:   settings.CompanionPostResultPage,
			Row:    settings.CompanionPostResultRow,
			Column: settings.CompanionPostResultColumn,
		},
		partner.EventAllianceSelection: {
			Page:   settings.CompanionAllianceSelectionPage,
			Row:    settings.CompanionAllianceSelectionRow,
			Column: settings.CompanionAllianceSelectionColumn,
		},
		partner.EventMatchAbort: {
			Page:   settings.CompanionMatchAbortPage,
			Row:    settings.CompanionMatchAbortRow,
			Column: settings.CompanionMatchAbortColumn,
		},
	}
	arena.CompanionClient = partner.NewCompanionClient(
		settings.CompanionAddress,
		settings.CompanionPort,
		companionEventConfigs,
	)

	game.MatchTiming.AutoDurationSec = settings.AutoDurationSec
	game.MatchTiming.PauseDurationSec = settings.PauseDurationSec
	game.MatchTiming.TeleopDurationSec = settings.TeleopDurationSec
	game.MatchTiming.WarningSoundTimeSec = settings.WarningSoundTimeSec
	game.UpdateMatchSounds()
	arena.MatchTimingNotifier.Notify()

	// Reconstruct the playoff tournament in memory.
	if err = arena.CreatePlayoffTournament(); err != nil {
		return err
	}
	if err = arena.UpdatePlayoffTournament(); err != nil {
		return err
	}

	return nil
}

// Constructs an empty playoff tournament in memory, based only on the number of alliances.
func (arena *Arena) CreatePlayoffTournament() error {
	var err error
	arena.PlayoffTournament, err = playoff.NewPlayoffTournament(
		arena.EventSettings.PlayoffType, arena.EventSettings.NumPlayoffAlliances,
	)
	return err
}

// Performs the one-time creation of all matches for the playoff tournament.
func (arena *Arena) CreatePlayoffMatches(startTime time.Time) error {
	return arena.PlayoffTournament.CreateMatchesAndBreaks(arena.Database, startTime)
}

// Traverses the playoff tournament rounds to assess winners and populate subsequent matches.
func (arena *Arena) UpdatePlayoffTournament() error {
	alliances, err := arena.Database.GetAllAlliances()
	if err != nil {
		return err
	}
	if len(alliances) > 0 {
		return arena.PlayoffTournament.UpdateMatches(arena.Database)
	}
	return nil
}

// Sets up the arena for the given match.
func (arena *Arena) LoadMatch(match *model.Match) error {
	if arena.MatchState != PreMatch && arena.MatchState != TimeoutActive {
		return fmt.Errorf("cannot load match while there is a match still in progress or with results pending")
	}

	arena.CurrentMatch = match

	for station, teamId := range match.TeamIdsByStation() {
		if err := arena.assignTeam(teamId, station); err != nil {
			return err
		}
	}

	// Reset the arena state and realtime scores.
	arena.soundsPlayed = make(map[*game.MatchSound]struct{})
	arena.RedRealtimeScore = NewRealtimeScore()
	arena.BlueRealtimeScore = NewRealtimeScore()
	arena.ScoringPanelRegistry.resetScoreCommitted()

	// Notify any listeners about the new match.
	arena.MatchLoadNotifier.Notify()
	arena.RealtimeScoreNotifier.Notify()
	arena.AllianceStationDisplayMode = "match"
	arena.AllianceStationDisplayModeNotifier.Notify()
	arena.ScoringStatusNotifier.Notify()

	return nil
}

// Sets a new test match containing no teams as the current match.
func (arena *Arena) LoadTestMatch() error {
	return arena.LoadMatch(&model.Match{Type: model.Test, ShortName: "T", LongName: "Test Match"})
}

// Loads the first unplayed match of the current match type.
func (arena *Arena) LoadNextMatch(startScheduledBreak bool) error {
	nextMatch, err := arena.getNextMatch(false)
	if err != nil {
		return err
	}
	if nextMatch == nil {
		return arena.LoadTestMatch()
	}
	err = arena.LoadMatch(nextMatch)
	if err != nil {
		return err
	}

	// Start the timeout timer if there is a scheduled break before this match.
	if startScheduledBreak {
		scheduledBreak, err := arena.Database.GetScheduledBreakByMatchTypeOrder(nextMatch.Type, nextMatch.TypeOrder)
		if err != nil {
			return err
		}
		if scheduledBreak != nil {
			go func() {
				time.Sleep(time.Second * scheduledBreakDelaySec)
				if err := arena.StartTimeout(scheduledBreak.Description, scheduledBreak.DurationSec); err != nil {
					log.Printf("Failed to start scheduled break timeout before match %d: %v", nextMatch.Id, err)
				}
			}()
		}
	}

	return nil
}

// Assigns the given team to the given station, also substituting it into the match record.
func (arena *Arena) SubstituteTeams(red1, red2, blue1, blue2 game.TeamId) error {
	if !arena.CurrentMatch.ShouldAllowSubstitution() {
		return fmt.Errorf("Can't substitute teams for qualification matches.")
	}

	if err := arena.validateTeams(red1, red2, blue1, blue2); err != nil {
		return err
	}
	if err := arena.assignTeam(red1, "R1"); err != nil {
		return err
	}
	if err := arena.assignTeam(red2, "R2"); err != nil {
		return err
	}
	if err := arena.assignTeam(blue1, "B1"); err != nil {
		return err
	}
	if err := arena.assignTeam(blue2, "B2"); err != nil {
		return err
	}

	arena.CurrentMatch.Red1 = red1
	arena.CurrentMatch.Red2 = red2
	arena.CurrentMatch.Blue1 = blue1
	arena.CurrentMatch.Blue2 = blue2
	arena.MatchLoadNotifier.Notify()

	if arena.CurrentMatch.Type != model.Test {
		arena.Database.UpdateMatch(arena.CurrentMatch)
	}
	return nil
}

// Starts the match if all conditions are met.
func (arena *Arena) StartMatch() error {
	err := arena.checkCanStartMatch()
	if err == nil {
		// Save the match start time to the database for posterity.
		arena.CurrentMatch.StartedAt = time.Now()
		if arena.CurrentMatch.Type != model.Test {
			arena.Database.UpdateMatch(arena.CurrentMatch)
		}
		arena.updateCycleTime(arena.CurrentMatch.StartedAt)

		// Save the teams that have taken the field, so that the team list can show who has played at least once.
		for _, allianceStation := range arena.AllianceStations {
			if allianceStation.Team != nil && !allianceStation.Team.HasConnected && allianceStation.Ready {
				allianceStation.Team.HasConnected = true
				arena.Database.UpdateTeam(allianceStation.Team)
			}
		}

		arena.MatchState = StartMatch
	}
	return err
}

// Kills the current match or timeout if it is underway.
func (arena *Arena) AbortMatch() error {
	if arena.MatchState == PreMatch || arena.MatchState == PostMatch || arena.MatchState == PostTimeout {
		return fmt.Errorf("cannot abort match when it is not in progress")
	}

	if arena.MatchState == TimeoutActive {
		// Handle by advancing the timeout clock to the end and letting the regular logic deal with it.
		arena.MatchStartTime = time.Now().Add(-time.Second * time.Duration(game.MatchTiming.TimeoutDurationSec))
		return nil
	}

	arena.PlaySound("abort")
	arena.MatchState = PostMatch
	arena.matchAborted = true
	arena.SetAudienceDisplayMode("blank")
	go arena.BlackmagicClient.StopRecording()
	go arena.CompanionClient.SendEvent(partner.EventMatchAbort)
	return nil
}

// Clears out the match and resets the arena state unless there is a match underway.
func (arena *Arena) ResetMatch() error {
	if arena.MatchState != PostMatch && arena.MatchState != PreMatch && arena.MatchState != TimeoutActive {
		return fmt.Errorf("cannot reset match while it is in progress")
	}
	if arena.MatchState != TimeoutActive {
		arena.MatchState = PreMatch
	}
	arena.matchAborted = false
	for _, allianceStation := range arena.AllianceStations {
		allianceStation.Bypass = false
		allianceStation.Ready = false
		allianceStation.AStop = false
		allianceStation.EStop = false
	}
	arena.MuteMatchSounds = false
	return nil
}

// Starts a timeout of the given duration.
func (arena *Arena) StartTimeout(description string, durationSec int) error {
	return arena.startTimeout(description, arena.currentMatchDisplayName(), durationSec)
}

// Starts an ad-hoc timeout of the given duration.
func (arena *Arena) StartAdHocTimeout(description string, nextMatchName string, durationSec int) error {
	return arena.startTimeout(description, nextMatchName, durationSec)
}

// Sets the text shown on timeout displays.
func (arena *Arena) SetTimeoutDisplay(description string, nextMatchName string) {
	arena.breakDescription = description
	arena.breakNextMatchName = nextMatchName
	arena.MatchLoadNotifier.Notify()
}

func (arena *Arena) startTimeout(description string, nextMatchName string, durationSec int) error {
	if arena.MatchState != PreMatch {
		return fmt.Errorf("cannot start timeout while there is a match still in progress or with results pending")
	}

	game.MatchTiming.TimeoutDurationSec = durationSec
	game.UpdateMatchSounds()
	arena.soundsPlayed = make(map[*game.MatchSound]struct{})
	arena.MatchTimingNotifier.Notify()
	arena.breakDescription = description
	arena.breakNextMatchName = nextMatchName
	arena.MatchLoadNotifier.Notify()
	arena.MatchState = TimeoutActive
	arena.MatchStartTime = time.Now()
	arena.LastMatchTimeSec = -1
	arena.AllianceStationDisplayMode = "timeout"
	arena.AllianceStationDisplayModeNotifier.Notify()

	return nil
}

func (arena *Arena) currentMatchDisplayName() string {
	if arena.CurrentMatch == nil {
		return ""
	}
	name := arena.CurrentMatch.LongName
	if arena.CurrentMatch.NameDetail != "" {
		name += " - " + arena.CurrentMatch.NameDetail
	}
	return name
}

// Updates the audience display screen.
func (arena *Arena) SetAudienceDisplayMode(mode string) {
	if arena.AudienceDisplayMode != mode {
		arena.AudienceDisplayMode = mode
		arena.AudienceDisplayModeNotifier.Notify()
		if mode == "score" {
			arena.PlaySound("match_result")
			go arena.CompanionClient.SendEvent(partner.EventShowFinalScore)
		} else if mode == "allianceSelection" {
			go arena.CompanionClient.SendEvent(partner.EventAllianceSelection)
		} else if mode == "intro" {
			go arena.CompanionClient.SendEvent(partner.EventMatchPreview)
		} else if mode == "match" {
			go arena.CompanionClient.SendEvent(partner.EventShowOverlay)
		}
	}
}

// Updates the alliance station display screen.
func (arena *Arena) SetAllianceStationDisplayMode(mode string) {
	if arena.AllianceStationDisplayMode != mode {
		arena.AllianceStationDisplayMode = mode
		arena.AllianceStationDisplayModeNotifier.Notify()
	}
}

// Returns the fractional number of seconds since the start of the match.
func (arena *Arena) MatchTimeSec() float64 {
	if arena.MatchState == PreMatch || arena.MatchState == StartMatch || arena.MatchState == PostMatch {
		return 0
	} else {
		return time.Since(arena.MatchStartTime).Seconds()
	}
}

// Performs a single iteration of checking inputs and timers and setting outputs accordingly to control the
// flow of a match.
func (arena *Arena) Update() {
	matchTimeSec := arena.MatchTimeSec()
	stateChanged := false
	switch arena.MatchState {
	case PreMatch:
		// Nothing to do; waiting for the match to be started.
	case StartMatch:
		arena.MatchStartTime = time.Now()
		arena.LastMatchTimeSec = -1
		arena.SetAudienceDisplayMode("match")
		arena.SetAllianceStationDisplayMode("match")
		go arena.BlackmagicClient.StartRecording()
		go arena.CompanionClient.SendEvent(partner.EventMatchStart)
		arena.MatchState = AutoPeriod
		stateChanged = true
		arena.FieldVolunteers = false
		arena.FieldReset = false
	case AutoPeriod:
		if matchTimeSec >= game.GetDurationToAutoEnd().Seconds() {
			arena.MatchState = PausePeriod
			stateChanged = true
		}
	case PausePeriod:
		if matchTimeSec >= game.GetDurationToTeleopStart().Seconds() {
			arena.MatchState = TeleopPeriod
			stateChanged = true
			go arena.CompanionClient.SendEvent(partner.EventTeleopStart)
		}
	case TeleopPeriod:
		if matchTimeSec >= game.GetDurationToTeleopEnd().Seconds() {
			arena.MatchState = PostMatch
			stateChanged = true
			go arena.BlackmagicClient.StopRecording()
			go arena.CompanionClient.SendEvent(partner.EventMatchEnd)
			go func() {
				// Leave the scores on the screen briefly at the end of the match.
				time.Sleep(time.Second * matchEndScoreDwellSec)
				arena.SetAudienceDisplayMode("blank")
			}()
			go func() {
				// Queue the next match on the team signs after a delay.
				time.Sleep(time.Second * showNextMatchDelaySec)
				arena.showNextMatchOnTeamSigns()
			}()
		}
	case TimeoutActive:
		if matchTimeSec >= float64(game.MatchTiming.TimeoutDurationSec) {
			arena.MatchState = PostTimeout
			go func() {
				// Leave the timer on the screen briefly at the end of the timeout period.
				time.Sleep(time.Second * matchEndScoreDwellSec)
				arena.SetAudienceDisplayMode("intro")
				arena.SetAllianceStationDisplayMode("match")
			}()
		}
	case PostTimeout:
		if matchTimeSec >= float64(game.MatchTiming.TimeoutDurationSec+postTimeoutSec) {
			arena.MatchState = PreMatch
		}
	}

	// Send a match tick notification if passing an integer second threshold or if the match state changed.
	if int(matchTimeSec) != int(arena.LastMatchTimeSec) || arena.MatchState != arena.lastMatchState {
		arena.MatchTimeNotifier.Notify()
	}

	if stateChanged {
		arena.ArenaStatusNotifier.Notify()
	}

	arena.handleSounds(matchTimeSec)
	arena.updateFieldReadyState()

	// Handle the team number / timer displays.
	arena.TeamSigns.Update(arena)

	arena.LastMatchTimeSec = matchTimeSec
	arena.lastMatchState = arena.MatchState
}

// Loops indefinitely to track and update the arena components.
func (arena *Arena) Run() {
	for {
		loopStartTime := time.Now()
		arena.Update()
		if time.Since(loopStartTime).Milliseconds() > arenaLoopWarningMs {
			log.Printf("Warning: Arena loop iteration took a long time: %dms", time.Since(loopStartTime).Milliseconds())
		}
		if time.Since(arena.lastPeriodicTaskTime).Seconds() >= periodicTaskPeriodSec {
			arena.lastPeriodicTaskTime = time.Now()
			go arena.runPeriodicTasks()
		}

		time.Sleep(time.Millisecond * arenaLoopPeriodMs)
	}
}

// Calculates the red alliance score summary for the given realtime snapshot.
func (arena *Arena) RedScoreSummary() *game.ScoreSummary {
	return arena.RedRealtimeScore.CurrentScore.Summarize(&arena.BlueRealtimeScore.CurrentScore)
}

// Calculates the blue alliance score summary for the given realtime snapshot.
func (arena *Arena) BlueScoreSummary() *game.ScoreSummary {
	return arena.BlueRealtimeScore.CurrentScore.Summarize(&arena.RedRealtimeScore.CurrentScore)
}

// Checks that the given teams are present in the database, allowing the empty team ID which indicates an empty spot.
func (arena *Arena) validateTeams(teamIds ...game.TeamId) error {
	for _, teamId := range teamIds {
		if teamId == "" {
			continue
		}
		team, err := arena.Database.GetTeamById(teamId)
		if err != nil {
			return err
		}
		if team == nil {
			return fmt.Errorf("Team %s is not present at the event.", teamId)
		}
	}
	return nil
}

// Loads a team into an alliance station, cleaning up the previous team there if there is one.
func (arena *Arena) assignTeam(teamId game.TeamId, station string) error {
	// Reject invalid station values.
	allianceStation, ok := arena.AllianceStations[station]
	if !ok {
		return fmt.Errorf("Invalid alliance station '%s'.", station)
	}

	// Do nothing if the station is already assigned to the requested team.
	if allianceStation.Team != nil && allianceStation.Team.Id == teamId {
		return nil
	}

	allianceStation.AStop = false
	allianceStation.EStop = false
	allianceStation.Ready = false

	// Leave the station empty if the team number is blank.
	if teamId == "" {
		allianceStation.Team = nil
		return nil
	}

	// Load the team model. If it doesn't exist, enable anonymous operation.
	team, err := arena.Database.GetTeamById(teamId)
	if err != nil {
		return err
	}
	if team == nil {
		team = &model.Team{Id: teamId}
	}

	arena.AllianceStations[station].Team = team
	return nil
}

// Returns the next match of the same type that is currently loaded, or nil if there are no more matches.
func (arena *Arena) getNextMatch(excludeCurrent bool) (*model.Match, error) {
	if arena.CurrentMatch.Type == model.Test {
		return nil, nil
	}

	matches, err := arena.Database.GetMatchesByType(arena.CurrentMatch.Type, false)
	if err != nil {
		return nil, err
	}
	for _, match := range matches {
		if !match.IsComplete() && !(excludeCurrent && match.Id == arena.CurrentMatch.Id) {
			return &match, nil
		}
	}

	// There are no matches left of the same type.
	return nil, nil
}

// Shows the teams for the upcoming match on the team signs, so that they can queue while the current match is
// still being scored and committed.
func (arena *Arena) showNextMatchOnTeamSigns() {
	if arena.MatchState != PostMatch {
		// The next match has already been loaded; no need to do anything.
		return
	}

	nextMatch, err := arena.getNextMatch(true)
	if err != nil {
		log.Printf("Failed to look up the next match: %s", err.Error())
	}
	if nextMatch == nil {
		return
	}

	arena.TeamSigns.SetNextMatchTeams(nextMatch.TeamIds())
}

// Returns nil if the match can be started, and an error otherwise.
func (arena *Arena) checkCanStartMatch() error {
	if arena.MatchState != PreMatch {
		return fmt.Errorf("cannot start match while there is a match still in progress or with results pending")
	}

	return arena.checkAllianceStationsReady(AllianceStationIds...)
}

// Returns nil if every given station is ready for a match to start, and an error otherwise. A station is ready once
// the field crew has marked its robot as staged, or once it has been bypassed.
func (arena *Arena) checkAllianceStationsReady(stations ...string) error {
	for _, station := range stations {
		allianceStation := arena.AllianceStations[station]
		if allianceStation.EStop {
			return fmt.Errorf("cannot start match while an emergency stop is active")
		}
		if !allianceStation.Bypass && !allianceStation.Ready {
			return fmt.Errorf("cannot start match until all robots are ready or bypassed")
		}
	}

	return nil
}

// Returns the alliance station identifier for the given team, or the empty string if the team is not present
// in the current match.
func (arena *Arena) getAssignedAllianceStation(teamId game.TeamId) string {
	for station, allianceStation := range arena.AllianceStations {
		if allianceStation.Team != nil && allianceStation.Team.Id == teamId {
			return station
		}
	}

	return ""
}

// Records the moment at which every station first became ready, for cycle time reporting.
func (arena *Arena) updateFieldReadyState() {
	switch arena.MatchState {
	case PreMatch, TimeoutActive, PostTimeout:
		if arena.checkAllianceStationsReady(AllianceStationIds...) == nil {
			arena.FieldVolunteers = false
			arena.FieldReset = false
			if arena.CurrentMatch.FieldReadyAt.IsZero() {
				arena.CurrentMatch.FieldReadyAt = time.Now()
			}
		}
	}
}

// Marks a station's robot as staged and ready for the match to start, or clears that state.
func (arena *Arena) SetStationReady(station string, ready bool) error {
	allianceStation, ok := arena.AllianceStations[station]
	if !ok {
		return fmt.Errorf("Invalid alliance station '%s'.", station)
	}
	allianceStation.Ready = ready
	arena.ArenaStatusNotifier.Notify()
	return nil
}

// Set the team signs to signal count mode, if not in a match.
func (arena *Arena) SignalVolunteers() {
	if arena.MatchState != PostMatch && arena.MatchState != PreMatch {
		// Don't signal volunteers during matches.
		return
	}
	arena.FieldVolunteers = true
	arena.FieldReset = false
	arena.AllianceStationDisplayMode = "signalCount"
	arena.AllianceStationDisplayModeNotifier.Notify()
}

// Set the team signs to field reset mode, if not in a match.
func (arena *Arena) SignalReset() {
	if arena.MatchState != PostMatch && arena.MatchState != PreMatch {
		// Don't signal reset during matches.
		return
	}
	if !arena.FieldReset {
		// Play the audio cue once, even if reset is signaled multiple times.
		arena.PlaySound("field_reset")
	}
	arena.FieldVolunteers = false
	arena.FieldReset = true
	arena.AllianceStationDisplayMode = "fieldReset"
	arena.AllianceStationDisplayModeNotifier.Notify()
}

func (arena *Arena) handleSounds(matchTimeSec float64) {
	if arena.MatchState == PreMatch || arena.MatchState == TimeoutActive || arena.MatchState == PostTimeout {
		// Only apply this logic during a match.
		return
	}

	for _, sound := range game.MatchSounds {
		if sound.MatchTimeSec < 0 {
			// Skip sounds with negative timestamps; they are meant to only be triggered explicitly.
			continue
		}
		if _, ok := arena.soundsPlayed[sound]; !ok {
			if matchTimeSec >= sound.MatchTimeSec && matchTimeSec-sound.MatchTimeSec < 1 {
				arena.PlaySound(sound.Name)
				arena.soundsPlayed[sound] = struct{}{}
			}
		}
	}
}

func (arena *Arena) PlaySound(name string) {
	if !arena.MuteMatchSounds {
		arena.PlaySoundNotifier.NotifyWithMessage(name)
	}
}

func (arena *Arena) positionPostMatchScoreReady(position string) bool {
	numPanels := arena.ScoringPanelRegistry.GetNumPanels(position)
	return numPanels > 0 && arena.ScoringPanelRegistry.GetNumScoreCommitted(position) >= numPanels
}

// Performs any actions that need to run at the interval specified by periodicTaskPeriodSec.
func (arena *Arena) runPeriodicTasks() {
	arena.updateEarlyLateMessage()
	arena.purgeDisconnectedDisplays()
}

// Handles audience display automation from after score post to next match intro.
func (arena *Arena) AutomateAudienceDisplay(postedMatch *model.Match) {
	// Show the score for 20 seconds before moving on.
	time.Sleep(20 * time.Second)
	if arena.AudienceDisplayMode != "score" {
		return
	}

	if arena.CurrentMatch.Type == model.Playoff {
		time.Sleep(10 * time.Second)
		isFinals := strings.Contains(postedMatch.LongName, "Final") || strings.Contains(postedMatch.LongName, "Overtime")
		if !isFinals {
			arena.SetAudienceDisplayMode("bracket")
			time.Sleep(20 * time.Second)
			if arena.AudienceDisplayMode != "bracket" {
				return
			}
		}
	}

	if arena.MatchState == TimeoutActive {
		arena.SetAudienceDisplayMode("timeout")
		arena.SetAllianceStationDisplayMode("timeout")
		return
	}

	if arena.CurrentMatch.Type == model.Test {
		// No next match loaded, show the score for longer and then go into awards mode.
		time.Sleep(40 * time.Second)
		if arena.AudienceDisplayMode != "score" {
			return
		}

		arena.SetAudienceDisplayMode("logoLuma")
		arena.SetAllianceStationDisplayMode("logo")
		return
	}

	arena.SetAudienceDisplayMode("intro")
}
