# XRP / Iron Acres Adaptation — Handoff

Status as of this commit. Converting Cheesy Arena Lite (FRC, 3v3, generic scoring) into a field
management system for the **XRP "Hotwire Iron Acres"** game (2v2, element-level scoring, no FRC
control system).

Source of truth for game rules: `Hotwire Iron Acres Manual.md` in the repo root.

## Current state

**`go build ./...` passes.** All production code compiles.

**`go test ./...` — 7 of 8 packages pass:**

| Package | Status |
| --- | --- |
| `game` | pass |
| `model` | pass |
| `field` | pass |
| `tournament` | pass |
| `playoff` | pass |
| `partner` | pass |
| `websocket` | pass |
| `web` | **build failure — this is the remaining work** |

`go fmt ./...` has been run.

> **Note on toolchain:** Go was not installed on this machine. A portable Go 1.26.5 was downloaded to
> the session scratchpad to build and test. Install Go properly (`go.mod` requires 1.26) before
> continuing, or re-download the toolchain.

## Scoping decisions already made

These were confirmed and are already implemented; don't relitigate them:

1. **FRC control-system plumbing was stripped**, not left dormant.
2. **2v2 is hardcoded**, not a setting.
3. **Element-level scoring panel**, with Minor/Major penalty entry on the referee panel.
4. **Ranking criteria left unchanged** (still W/L ranking points) — deferred, see Open Questions.

## What changed

### Game model — `game/`

- `game/score.go` rewritten. `Score` now holds **element counts**, not point totals:
  `FactoryParks`, `AutoCrops`, `SilosDumped`, `TeleopCrops`, `CityLimitsProducts`,
  `CityCenterProducts`, `BarnParks`, `BarnHangs`, `MinorPenalties`, `MajorPenalties`, `PlayoffDq`.
- Point values are constants from manual §5.2 / T03:
  park 5, crop 7, silo 5, city limits 10, city center 15, barn park 5, hang 25, minor 10, major 25.
- Points are now **methods**: `AutoPoints()`, `TeleopPoints()`, `EndgamePoints()`, `PenaltyPoints()`.
  (They used to be struct fields — that's the source of most remaining test breakage.)
- Penalties are recorded on the **offending** alliance and award points to the opponent, matching the
  old `FoulPointsAgainst` direction.
- `ScoreSummary` field names are **unchanged** (`AutoPoints`/`TeleopPoints`/`PostMatchPoints`/…) so
  ranking and display code kept working. `PostMatchPoints` means endgame.
- `game/alliance.go` (new): `TeamsPerAlliance = 2`, `TeamsPerMatch = 4`.
- `game/match_timing.go`: now `{10, 3, 150, 30, 0}` — auto 0:10, pause 0:03, teleop 2:30, endgame
  warning at 0:30. Total match 2:40 per manual §4.1.

### 2v2 conversion

- `model.Match` lost `Red3`/`Blue3` and their surrogate flags. Added helpers `TeamIds()`
  (`[4]int`) and `TeamIdsByStation()` (`map[string]int`).
- `model.Alliance.Lineup` is now `[game.TeamsPerAlliance]int`.
- `field.AllianceStationIds` = `["R1","R2","B1","B2"]`, plus per-alliance subsets.
- `arena.SubstituteTeams` takes 4 team numbers.
- Schedule loader (`tournament/schedule.go`) now reads **`<teams>_<matchesPerTeam>_2.csv`**.
  `TeamsPerMatch = 4`; each row is 8 fields (team + surrogate flag × 4).
- Playoff lineups, judging schedule, rankings, reports, and all templates/JS updated to 2 per alliance.

### FRC plumbing removed

Deleted outright: `plc/`, `network/`, `partner/tba.go`, `partner/nexus.go`,
`field/driver_station_connection.go`, `field/team_match_log.go`, `web/match_logs.go`,
`templates/match_logs.html`, `templates/view_match_log.html`, and their tests.
`partner/blackmagic.go` and `partner/companion.go` were **kept** (generic AV automation).

`AllianceStation` is now just `{AStop, EStop, Bypass, Ready, Team}`. There is no robot link state —
XRP robots are driven directly from each team's own controller. The field crew marks a station
**Ready** once the robot is staged in its Barn; `checkCanStartMatch` requires every station to be
Ready or Bypassed. New websocket command `toggleReady`; new arena method `SetStationReady`.

`EventSettings` lost all TBA/Nexus/AP/switch/SCC/PLC/lite-UDP fields and the third team-sign IDs.
`go.mod` no longer depends on `goburrow/modbus` or `golang.org/x/crypto`.

### UI

- **Scoring panel** (`templates/scoring_panel.html`, `static/js/scoring_panel.js`): rebuilt as
  ±counters per element matching the Appendix B scorecard, with live per-period subtotals. The
  server computes points; the JS point values are only for optimistic display.
- **Referee panel** (`templates/referee_panel.html`, `static/js/referee_panel.js`): foul-points
  number boxes replaced with Minor/Major penalty counters per alliance, showing points awarded to the
  opponent. Websocket commands are now `penalties` (absolute) and `addPenalty` (delta).
- **Match review editor** (`templates/edit_match_result.html`, `static/js/match_review.js`): element
  inputs grouped by period plus penalties.
- **Field monitor**: rebuilt as a staging/readiness view (was DS/radio/battery telemetry).
- **Match play**: DS/Radio/Robot status columns replaced with one Ready toggle; network and PLC
  status blocks removed.
- **Field testing page**: PLC I/O tables removed, sounds only.
- **Settings page**: Team Info Download, Networking, SCC, PLC, DS Lite Mode, and Nexus sections removed.

## Remaining work

### 1. Fix the `web` test package (the only thing failing)

12 test files, ~72 references to fix. Mechanical, following patterns already applied elsewhere:

| File | What to fix |
| --- | --- |
| `alliance_selection_test.go` | `Lineup[2]` out of bounds; `TbaClient`, `TbaPublishingEnabled` gone |
| `announcer_display_test.go` | `Blue3` in match literal |
| `api_test.go` | `Red3`/`Blue3`; API score JSON is now element counts |
| `field_monitor_display_test.go` | `SubstituteTeams` now takes 4 args |
| `match_play_test.go` | `Red3`/`Blue3`; `game.Score{AutoPoints: …}` → element fields |
| `match_review_test.go` | score JSON strings need the new element field names |
| `referee_panel_test.go` | `FoulPointsAgainst` → `MinorPenalties`/`MajorPenalties`; `foulPoints` message → `penalties`; 2 teams per alliance |
| `reports_test.go` | `Red3`/`Blue3` |
| `scoring_panel_test.go` | score message shape is now `{Red: {...}, Blue: {...}}` with element counts |
| `setup_field_testing_test.go` | PLC assertions — most of this test should be deleted |
| `setup_settings_test.go` | `TbaPublishingEnabled` gone |
| `setup_teams_test.go` | WPA/network-security assertions |

Reference patterns: `field/arena_test.go` (readiness model), `game/score_test.go` (element scoring),
`tournament/schedule_test.go` (2v2 schedules).

### 2. "Drivers pick up your controllers, 3-2-1" countdown — **NOT STARTED**

Requested mid-session; the timing model is in place (3s pause between auto and teleop) but the audio
cue was never added. Plan:

- Sounds are declared in `game/match_sounds.go` (`UpdateMatchSounds`) and rendered as
  `<audio id="sound-NAME" src="/static/audio/NAME.wav">` in `templates/audience_display.html`.
  A new cue therefore needs a real WAV in `static/audio/`.
- Recommended approach: **one** `countdown.wav` containing the whole phrase
  ("Drivers, pick up your controllers. Three. Two. One.") played at auto end, rather than four
  separately-timed cues — avoids drift.
- Then set `PauseDurationSec` to match the clip length (currently 3s; the phrase likely needs ~6s)
  and add the sound entry between the auto-end `end` cue and the teleop `resume` cue.
- A placeholder WAV can be generated on Windows with `System.Speech`:
  `Add-Type -AssemblyName System.Speech; $s = New-Object System.Speech.Synthesis.SpeechSynthesizer; $s.SetOutputToWaveFile("static/audio/countdown.wav"); $s.Speak("Drivers, pick up your controllers. Three. Two. One."); $s.Dispose()`
  Replace with a real recording before an event.

### 3. Verify in a browser

Nothing has been run against a live server. The Go code compiles and unit tests pass, but **no page
has been visually confirmed**. Worth checking at minimum: scoring panel, referee panel, match play,
audience display, match review editor.

### 4. Cosmetic / stale

- `static/css/scoring_panel.css` and `referee_panel.css` were **not** updated for the new counter
  markup (`.score-counter`, `.counter-button`, `.penalty-counter`, `.ready-status`, …). Expect the
  new panels to look unstyled until CSS is added.
- `README.md` still describes FRC driver stations and the 10.0.100.5 convention.
- `AGENTS.md` still describes the upstream-porting workflow against `../cheesy-arena`. That workflow
  is now largely obsolete — this fork has diverged hard.
- `switch_config.txt`, `fix_avatar_colors_for_overlay`, `tunnel*` are leftover FRC/ops files.

## Open questions

1. **Ranking criteria.** Deferred during this session. Manual T02 says each team receives their
   alliance's full score including penalty points, but states no ranking rule. Currently still
   FRC-style: 2 RP win / 1 tie, sorted by avg RP → avg match points → auto → endgame. If you want
   ranking by average score instead, change `game/ranking_fields.go` `Less()`.

2. **Schedule coverage is 12 teams only.** `schedules/*_2.csv` exist for 12 teams at 1–14 matches per
   team. Any other team count fails with "No schedule template exists for N teams and M matches".
   Generate more if you expect a different team count.

3. **Old 3v3 schedules are now dead files and are a footgun.** ~1300 `<teams>_<matches>.csv` files are
   unreachable (the loader only opens `*_2.csv`). Worse, files like `schedules/100_2.csv` *look* like
   2v2 files under the new convention but are actually 100 teams × 2 matches, 3v3. They are never
   loaded — the loader always appends `_2`, so there's no correctness bug — but consider deleting
   them to avoid confusion. **Not deleted without your say-so.**

4. **Manual inconsistency.** §4.1 gives a 2:40 total (0:10 + 2:30); the Glossary says "2 minute 45
   second contest". Implemented §4.1. Worth correcting the glossary.

5. **G01/G02 "no Auto points" penalties are not modelled.** The referee must manually zero the
   offending alliance's auto element counts. Could be automated if desired.
