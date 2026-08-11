XRP Iron Acres Arena
============
A field management system for the XRP **"Hotwire Iron Acres"** game, forked from
[Cheesy Arena Lite](https://github.com/Team254/cheesy-arena-lite).

Matches are **2 versus 2**. Robots are XRPs driven directly from each team's own controller, so there is no FRC
control system, no field network, and no driver station software involved — the field crew marks each station
**Ready** once a robot is staged in its Barn, and the arena runs the match clock, sounds, and scoring around that.

The rules this implements are in [`Hotwire Iron Acres Manual.md`](Hotwire%20Iron%20Acres%20Manual.md), which is the
source of truth for scoring values, penalties, and match timing.

## License
Teams may use this freely for practice, scrimmages, and off-season events. See [LICENSE](LICENSE) for more details.

## Running

1. Download [Go](https://golang.org/dl/) (see `go.mod` for the required version)
1. Clone this repository and navigate to its directory in a terminal
1. Compile the code with `go build`
1. Run the `cheesy-arena-lite` or `cheesy-arena-lite.exe` binary
1. Navigate to http://localhost:8080 in your browser (Google Chrome recommended)

Any IP address works; nothing on the field needs to reach the server except the browsers running the panels and
displays.

## Match flow

| Phase | Length | Notes |
| --- | --- | --- |
| Autonomous | 0:10 | Drivers press one button, then hands off (G02) |
| Pause | 0:06 | "Drivers, pick up your controllers. Three. Two. One." plays here |
| Driver Controlled | 2:30 | Endgame warning sounds with 0:30 left |

The pause exists to fit the countdown cue; it is adjustable on the Settings page, but must stay at least as long as
`static/audio/countdown.wav` or the teleop horn will cut the announcement off. `game.countdownDurationSec` records the
clip length and `TestCountdownFitsInPause` guards the relationship.

> `static/audio/countdown.wav` is currently a text-to-speech placeholder. Replace it with a real recording before an
> event, then update `countdownDurationSec` and `PauseDurationSec` to match.

## Scoring

Scores are entered as **element counts**, not point totals; the server applies the point values from manual §5.2.

| Element | Points | Period |
| --- | --- | --- |
| Parked in Factory | 5 | Auto |
| Crops in Factory | 7 | Auto and Teleop |
| Silo Dumped | 5 | Auto |
| Product in City Limits | 10 | Teleop |
| Product in City Center | 15 | Teleop |
| Parked in Barn | 5 | Endgame |
| Hanging from Barn | 25 | Endgame |

Penalties are recorded on the **offending** alliance and award points to the opponent: Minor 10, Major 25 (T03).

## Interfaces

- **Scoring panel** (`/panels/scoring`) — ± counters per element with live per-period subtotals
- **Referee panel** (`/panels/referee`) — cards plus Minor/Major penalty counters per alliance
- **Match play** (`/match_play`) — match control and per-station Ready/Bypass
- **Field monitor** (`/displays/field_monitor`) — staging and readiness view
- **Match review** (`/match_review`) — post-match editing of element counts and penalties

## Differences from upstream Cheesy Arena Lite

The FRC control-system plumbing was removed rather than left dormant: no `plc/`, `network/`, driver station
connections, team match logs, TBA/Nexus publishing, or WPA key provisioning. `partner/blackmagic.go` and
`partner/companion.go` were kept, since AV automation is game-agnostic. Alliances are two robots on the field with a
`TeamIds` list that may still hold backups.

## Development

See [AGENTS.md](AGENTS.md) for build, test, and style conventions.
