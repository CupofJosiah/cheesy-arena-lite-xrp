# XRP / Iron Acres Adaptation — Handoff

Cheesy Arena Lite converted into a field management system for the XRP **"Hotwire Iron Acres"** game (2v2,
element-level scoring, no FRC control system). Source of truth for rules: `Hotwire Iron Acres Manual.md`.

See `README.md` for how the system works and what was removed relative to upstream. This file tracks what is left.

## Current state

**`go build ./...` passes. `go test ./...` passes — all 8 packages.**

The server has been run and exercised end to end: teams imported, a 12-team / 30-match qualification schedule
generated and saved, a match driven through its full cycle, and scores entered from the scoring panel and read back
through the API. Every page linked from the navbar returns 200. Match cue sequence observed live:

```
  0.01s  start
 10.02s  countdown     <- "drivers, pick up your controllers, 3-2-1"
 16.01s  resume
```

> **Toolchain:** Go is not installed on this machine. A portable Go 1.26.5 was downloaded to the session scratchpad
> to build and test. Install Go properly before continuing.

## What was done in this session

Picking up from the previous handoff, whose remaining work was "fix the `web` test package".

### Production bugs found while fixing the tests

The test failures were not all test-side. Three templates still referenced things the conversion had deleted, and
Go templates fail at execution time, not at build time — so these were live 500s that no test could catch while the
`web` package did not compile:

- `templates/base.html` referenced `EventSettings.NetworkSecurityEnabled`. This is the shared navbar, so **every
  admin page returned 500.**
- `templates/edit_team.html` referenced the same field.
- `templates/setup_teams.html` referenced `EventSettings.TbaDownloadEnabled` and `NetworkSecurityEnabled`.
- `templates/match_review.html` indexed `RedTeams`/`BlueTeams` at index 2, which no longer exists in 2v2, so the
  match review page returned 500.

Also fixed: a dead `/match_logs` navbar link (route was deleted), and a missing `static/manifest/scoring.manifest`
that 404'd on every scoring panel load. The stale `red_scoring.manifest` / `blue_scoring.manifest` were removed.

### WPA key plumbing removed

The WPA key feature was half-wired: `teamEditPostHandler` no longer read the field, and its UI was only reachable
behind the `NetworkSecurityEnabled` setting that had been deleted. WPA keys provision FRC radios, which this fork
does not have, so it was removed the rest of the way, consistent with the earlier decision to strip FRC plumbing
rather than leave it dormant: `Team.WpaKey`, `teamsGenerateWpaKeysHandler`, `wpaKeysCsvReportHandler`, both routes,
and the template blocks. `Team` is JSON-encoded in BoltDB, so dropping the field is schema-safe.

### Web tests fixed

All 12 files from the previous handoff, plus four more that compiled but would have panicked or failed at runtime
(`alliance_station_display_test.go`, `audience_display_test.go`, `wall_display_test.go` set `AllianceStations["R3"]`,
which is now `nil`; `reports_test.go` expected Red3/Blue3 CSV columns).

`setup_schedule_test.go` used 38 teams, for which no 2v2 schedule template exists — see open question 1. It now uses
12 teams. The failure had been masking a `matches[0]` panic that aborted the test binary and silently skipped 18
tests.

### Countdown cue

`static/audio/countdown.wav` plays at the end of autonomous. It **replaces** the old auto-end buzzer rather than
layering on top of it: the manual has no auto-end stoppage, and G02 keeps hands off the controls only until the
driver-controlled period begins, so the announcement is the signal. `PauseDurationSec` went from 3 to 6 to fit the
5.4 s clip, making a match 2:46 rather than 2:40 — see open question 4.

`game.countdownDurationSec` records the clip length, `TestCountdownFitsInPause` guards that the pause is long
enough, and `TestArenaMatchSounds` walks a full match timeline asserting each cue fires once, in order.

**The clip is a text-to-speech placeholder.** Replace it with a real recording before an event, then update
`countdownDurationSec` and `PauseDurationSec`.

### CSS

`scoring_panel.css` and `referee_panel.css` now style the counter markup (`.score-counter`, `.counter-button`,
`.penalty-counter`, period headings, subtotal blocks), including the narrow-screen breakpoints for tablets. Both
panels were checked in a browser.

## Remaining work

1. **Replace `countdown.wav` with a real recording** (above).
2. **Schedule templates beyond 12 teams** — see open question 1. This is the one thing that will block a real event.
3. **Field monitor CSS uses stale FRC status names.** `.team-id[data-status=robot-linked]` is what "staged and
   ready" now maps to. It renders correctly; the names are just misleading.
4. **Leftover ops files** in the repo root: `switch_config.txt`, `fix_avatar_colors_for_overlay`, `tunnel`,
   `tunnel_nginx_config`, `coverage`. Not referenced by any code.

## Open questions

1. **Schedule coverage is 12 teams only.** `schedules/*_2.csv` exist for 12 teams at 1–14 matches per team; any
   other team count fails with "No schedule template exists for N teams and M matches". The legacy 3v3 set covers
   6–100 teams, so an equivalent 2v2 set needs generating. This was left alone because it is a decision about how
   many teams you actually expect — but note that the system currently cannot schedule any event that is not
   exactly 12 teams.

2. **~1300 dead 3v3 schedules are a footgun.** Files like `schedules/100_2.csv` look like 2v2 files under the new
   naming but are 100 teams × 2 matches, 3v3. The loader always appends `_2`, so they are unreachable and there is
   no correctness bug — but they will confuse anyone generating new templates. **Still not deleted without your
   say-so.**

3. **Ranking criteria unchanged.** Manual T02 says each team receives their alliance's full score including penalty
   points but states no ranking rule. Still FRC-style: 2 RP win / 1 tie, sorted by avg RP → avg match points → auto
   → endgame. To rank by average score instead, change `game/ranking_fields.go` `Less()`.

4. **Manual timing inconsistency.** §4.1 gives 2:40 (0:10 + 2:30, implying no pause); the Glossary says "2 minute
   45 second contest". The countdown cue requires a pause, and 6 s puts a match at 2:46 — closer to the Glossary
   than to §4.1. `PauseDurationSec` is adjustable on the Settings page. Worth reconciling the manual.

5. **G01/G02 "no Auto points" penalties are not modelled.** The referee must manually zero the offending alliance's
   auto element counts. Could be automated.

6. **`go vet ./...` reports pre-existing upstream issues** (unkeyed struct literals, `%s:%d` IPv6 address
   formatting). None were introduced by this work; none were fixed.
