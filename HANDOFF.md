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

> **Toolchain:** Go is not installed on this machine. A portable Go 1.26.5 was downloaded to a session scratchpad
> to build and test, and that scratchpad is temporary. **Install Go properly before continuing.**
>
> This is not a developer-convenience problem, it is an operational one. The server runs from the checked-in-adjacent
> `cheesy-arena-lite.exe`, which is gitignored and can only be produced by a Go toolchain. Templates, JS and CSS are
> read from disk at request time, so editing them changes a running server immediately — but anything in a `.go`
> file does nothing until that binary is rebuilt. A stale binary is therefore invisible: the UI for a new feature
> appears and even previews correctly client-side, while the server silently ignores it (`mapstructure` drops
> unknown fields from websocket messages without error). Bonus points landed in exactly this trap.
>
> `cheesy-arena-lite-prev.exe` is the last binary built before the bonus-points work, kept as a fallback.

## Alphanumeric team numbers

Team numbers can now contain letters (`12A`, `12B`), so several teams from the same school can be told apart. The
team ID is a `game.TeamId` string everywhere it used to be an `int`: `Team.Id`, the four station fields on `Match`,
`Alliance.TeamIds`/`Lineup`, `Ranking.TeamId`, `Award.TeamId`, `JudgingSlot.TeamId`, and every map keyed by team.
`""` replaces `0` as "no team".

- `game/team_id.go` holds the type, `ParseTeamId` (validate + uppercase, letters and digits only, max 6 characters),
  and `LessTeamId` (natural sort, so `9B` sorts before `10A`).
- The Bolt `table` wrapper is now generic over its ID type (`table[R, I]`); string-keyed tables must use manual IDs,
  since only ints have a sequence to draw from. Bucket keys are unchanged for numeric IDs.
- `game.TeamId` unmarshals from a JSON number as well as a string, so **existing databases still load**.
  `model/legacy_team_id_test.go` writes old-format records straight into the buckets and reads them back to guard
  this.
- The team import page validates the whole pasted list before creating anything and reports bad entries and
  duplicates instead of silently skipping them.
- `edit_match_result.html` had to quote the team numbers it writes into a JS array literal; these are
  `text/template`, so an unquoted `12A` would have been a syntax error. The `itoa` template helper was replaced by
  `teamKey`.

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

### Final score display and team avatars

The final score screen rendered four team rows per alliance (`{{range $i := seq 4}}`) because upstream put three
robots on the field plus an off-field team in row 4. The 2v2 conversion dropped the `setTeamInfo(..., 3, ...)` calls
from `audience_display.js` but left the template emitting the row, so row 3 was never populated **and** never
hidden — `setTeamInfo` is what calls `.toggle(hasTeam)`. Its two `<img src="">` tags rendered as broken-image icons
on every posted score. The off-field team moved to row 3 and the loop is now `seq 3`.

Team avatars were removed entirely. They came from TBA, which this fork does not have, so every lookup fell through
to a blank placeholder and left an empty 35px gutter in each team row. Gone: the `/api/teams/{teamId}/avatar` route
and handler, `static/img/avatars/`, `DisplayShared.getAvatarUrl`, the `.avatars`/`.avatar`/`.final-team-avatar` CSS,
and the markup on the audience, wall, and queueing displays. The queueing display's two `col-lg-1` avatar columns
were folded into the team columns, which went from `col-lg-2` to `col-lg-3` to keep the row at 12.

Ranking points and the per-team rank readout were then dropped from the final score screen. That removed the last
users of `playoff-hidden-field`, so the show/hide pair in `handleScorePosted` collapsed to a single
`$(".playoff-only-field").toggle(...)`, `setTeamInfo` lost its `rankings` parameter, and `static/img/rank-up.svg`
and `rank-down.svg` were deleted. `RedRankings`/`BlueRankings` stay in the `scorePosted` message — the announcer
display still reads them.

### Element counts on the match overlay

`.score-fields`, the slot between the score number and the team column, was inherited from upstream but was empty
and explicitly hidden. It now carries live crop and product counts on both the audience and wall displays: crops
are `AutoCrops + TeleopCrops`, product is `CityLimitsProducts + CityCenterProducts`. Both come off the existing
`realtimeScore` message, which already sent the raw `Score` alongside the summary, so no notifier change was
needed. The code lives in `DisplayShared.setScoreFields` so the two displays cannot drift apart.

The icons are `static/img/crop.svg` and `static/img/product.svg` applied as CSS `mask-image` over a white
`background-color` rather than as `<img>` tags. The white is a property of the stylesheet, not of the asset, so
the fill inside the SVG cannot drift the color and the same file stays reusable elsewhere. On the right-hand
alliance `.score-field` flips to `row-reverse` so the icon sits outboard of the count on both sides.

Making room for them widened `scoreOut` from 250px to 385px on both displays: a 120px `.score-fields` column, the
180px score number, and an 85px empty gutter on the inboard edge. **The gutter is not optional.** `#matchCircle`
is 150px wide and straddles the centerline, so it covers the inboard 75px of each score panel; `.score` justifies
its contents outboard, which is what puts the slack where the circle needs it. Upstream's 250px was the same
arrangement without the counts (180 + 70). A first attempt at 280px dropped the gutter to nothing and the circle
sat on top of the score number.

### The logo in the match circle

`#matchCircle` holds the game logo and the match timer. Both are positioned against the circle explicitly rather
than stacked in normal flow, because in flow the logo's line box shifted the timer by however much leading the
logo's aspect ratio happened to produce. `#logo`'s `top` is the **centre** of the logo, not its upper edge: the
display scripts animate it between the circle's centre at rest (`logoDown`, read straight out of the CSS) and the
centre of the circle's upper half during a match (`logoUp`, 37px, duplicated in both display scripts), and the
logo stays centred at both ends whatever shape it is. `max-width`/`max-height` size it to the largest box that
still clears the circle's edge, so a wide banner and a square logo both fit.

**CSS cannot see transparent margins baked into the image.** A logo with padding renders small and floating high
no matter how the box is positioned, because the box is centred and the artwork is not. `static/img/game-logo.png`
is therefore kept cropped to its artwork — the Iron Acres logo arrived as 1193x709 with the art occupying only
y=190..507, which is exactly what that looks like on the audience display. Crop any replacement the same way
(`Image.open(p).crop(Image.open(p).getbbox())`).

`scoreOut` and `scoreFieldsOut` are duplicated as constants in `audience_display.js` and `wall_display.js` and
must stay in sync with `.score-number`'s width in `display_overlay_shared.css`. The overlay's top row is now
`70 + 385 + 385 + 70` = 910px wide, against 640px upstream — worth knowing, since the audience display is
composited over video.

### The winner celebration

When a result is posted the audience display sweeps the winning alliance's colour across the screen, slams in
`RED ALLIANCE` / `WINS` with the two scores, and then hands off to the existing final score screen. A tie shows
each alliance on its own half under `TIE` / `MATCH`.

**It is deliberately not a new audience display mode.** It is a prelude to the existing `score` screen, built
entirely in `templates/audience_display.html`, `static/css/audience_display.css` and `static/js/audience_display.js`
with no Go changes, so it can be deployed onto a running server by saving the files and refreshing the browser —
see the template/handler hazard below for why that mattered. Making it a real mode would mean a `winner` value in
`SetAudienceDisplayMode`, a server-timed hand-off to `score` (a client that self-advances leaves the server
thinking it is still on `winner`, which silently breaks `AutomateAudienceDisplay`'s `!= "score"` guard and the
match play radio buttons), and a new radio in `audience_display_radio_buttons.html`. That is the upgrade path if
an operator preview button is ever wanted; it is a rebuild, so not something to do mid-event.

It works because the server already sends the pieces in the right order: `commitMatchScore` fires
`ScorePostedNotifier` before `commitPostAndLoadNextMatch` calls `SetAudienceDisplayMode("score")`, so the display
knows who won before it is told to show the score. Three things follow from that:

- **A `scorePosted` message is not evidence that a result was just posted.** `websocket.HandleNotifiers` replays
  every notifier's last message to each client as it connects, and the display reconnects on its own after a
  network blip, so `handleScorePosted` compares the payload against the previous one and only arms the celebration
  when it has actually changed. Match ID alone is not enough — test matches reuse an ID.
- **The celebration is armed once and consumed once.** `pendingWinnerAnimation` is cleared as it plays, so
  navigating back to the final score later is quiet. Only `blank -> score` is hooked, which is the whole
  post-match path: there is no direct `match -> score` edge in the transition map, so it always routes via `blank`.
- **The blinds close underneath it.** The layer is opaque and covers the screen, so `assembleScoreScreen` runs
  concurrently with the animation rather than after it and the hand-off is just the layer fading out. Running them
  in sequence instead leaves the finished celebration frozen on screen for the ~2s the blinds take.

`--winner-duration` in the CSS is the single source of truth for the length; the script reads it back out of the
computed style, the same way `logoDown` and `scoreIn` are read out of the CSS. Keep it past the last keyframe.

Two details in the CSS look like mistakes and are not. The sweep halves are `calc(50% + 1px)` so that they overlap
rather than abut; at exactly `50%` a hairline of the page shows through down the centre of the screen. And the
winning alliance's score chip is inverted to a white fill with coloured digits, because it is otherwise the one
number on screen that is the same colour as the field behind it, leaving the loser's score reading first.

`TestAudienceDisplay` asserts the markup is present, because it is inert — nothing else would notice if it went
missing from the template.

### Bonus points

`Score.BonusPoints` is a manual adjustment the scorekeeper can apply to either alliance. It is the only field on
`game.Score` that holds points rather than a count of elements, and the only one that may be negative, so that an
award can be taken back or a deduction applied. Every path that clamps element counts at zero
(`scoringPanelAllianceScore.applyTo`, `applyApiAllianceScore`, `applyScorePatch`, `adjustScore` in
`scoring_panel.js`) special-cases it.

It is entered on the scoring panel under a **Bonus Points** heading, one tap per `game.BonusPointIncrement` (10),
and on the edit-match-result page as a number input with `step="10"` and no minimum. The increment is written into
`scoring_panel.html` as a literal, the same way the element point values are, and `TestScoringPanel` asserts it has
not drifted from the constant — see the hazard below for why it is not passed in from the handler.

> **Templates are read from disk on every request; handlers are not.** `web.parseFiles` re-parses the `.html` files
> per request, so editing a template changes what a *running* server serves immediately, with no rebuild. The Go
> code behind it is whatever was compiled into the running binary. A template that reads a field the running
> binary's handler does not supply fails at execution time, and the page dies mid-render with
> `can't evaluate field X`. Adding `.BonusPointIncrement` to `scoring_panel.html` took out the scoring panel on a
> live server this way, before anything had been rebuilt. Keep new template/Go couplings out of the panels, and
> **restart the server after pulling changes that touch `game.Score` or `game.ScoreSummary`** — the summary fields
> that `edit_match_result.html` and `announcer_display_score_posted.html` now read have the same requirement.

The bonus is part of `matchPoints()`, so it counts toward the win, the final score, and ranking match points. Two
consequences worth knowing:

- `RankingFields.TeleopPoints()` is a residual (`MatchPoints - AutoPoints - PostMatchPoints`), so a bonus lands in
  the teleop column of the rankings report. Rankings themselves are unaffected; teleop is not a sort criterion.
- The overlay shows `Score - PostMatchPoints` during a match, so a bonus appears on the audience display the moment
  it is entered.

`ScoreSummary.BonusPoints` carries it to the displays. The announcer's posted-score modal and the audience final
score screen both hide the bonus row when it is zero, which is every normal match.

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
