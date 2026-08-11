# Repository Guidelines

## Project Structure & Module Organization
`main.go` is the entry point for the Go web server. Core domains live in top-level packages such as `field/`, `game/`, `partner/`, `playoff/`, `tournament/`, and `websocket/`. Web UI assets are in `web/`, `static/`, and `templates/`. Pre-generated schedules are in `schedules/`; the 2v2 loader reads `<teams>_<matchesPerTeam>_2.csv`. BoltDB data is stored in `db/` (and test fixtures in `*_test.db` files at the repo root).

This is a game-specific fork for XRP "Hotwire Iron Acres" (2v2, element-level scoring, no FRC control system). `Hotwire Iron Acres Manual.md` in the repo root is the source of truth for rules, point values, and match timing. See [README.md](README.md) for what was removed relative to upstream.

## Build, Test, and Development Commands
See `go.mod` for what version of Go to use.
1. `go build`
   Builds the `cheesy-arena` binary in the repo root.
1. `./cheesy-arena`
   Runs the server; open `http://localhost:8080` in a browser.
1. `go test ./...`
   Runs all Go tests across packages. Should be run after making any code changes to ensure nothing is broken.
1. `go fmt ./...`
   Formats all Go code in the repo. Should be run after making any code changes to ensure consistent style.

## Coding Style & Naming Conventions
Follow standard Go style: tabs for indentation, exported names in `CamelCase`, unexported in `camelCase`. Format code with `gofmt` before submitting changes. If you update a set of enum-style constants, run `go generate ./...` to refresh the generated enum string helpers. Keep package names short and domain-focused (matching existing directories like `field`, `game`, `partner`).

Committed Go files use CRLF line endings, which `gofmt` rewrites to LF. Running `go fmt ./...` therefore marks every file as modified in `git status` even though `git diff` reports no content change. Format only the files you actually edited, or restore the rest with `git checkout --` before committing.

Order imports alphabetically without any grouping or empty lines between them or special treatment of standard library vs third-party imports. Update any files that don't adhere to this standard if editing them for other reasons. Don't use goimports.

## Testing Guidelines
Tests are Go `*_test.go` files co-located with packages (for example `field/`, `game/`, `partner/`, `playoff/`). Use `go test ./...` for the full suite and `go test ./field -run TestName` to target specific areas. When adding new behavior, add or update tests in the same package and prefer table-driven tests for coverage.

## Commit & Pull Request Guidelines
Commit messages in this repo are short, imperative sentences (for example “Fix driver station TCP reads”) and often include an issue/PR number in parentheses (for example “... (#258)”). Keep to that style.

PRs should include:
1. A clear summary of the change.
1. Test notes (exact commands run, for example `go test ./...`).
1. UI screenshots when changing pages in `web/`, `static/`, or `templates/`.

## Configuration & Ops Notes
The server runs locally and uses BoltDB for data. There is no field network, PLC, or driver station integration; per-station readiness is set by hand from the match play page or field monitor.

Changes to scoring elements, point values, penalties, or match timing must trace back to `Hotwire Iron Acres Manual.md`. Point values live in `game/score.go` as constants and are applied server-side; the values in `static/js/scoring_panel.js` and the panel templates exist only for optimistic display and must be kept in sync.

## Upstream Porting Workflow
This fork has diverged substantially from Cheesy Arena Lite: 2v2 alliances, element-level scoring, and the removal of the FRC control-system plumbing all touch the same files upstream keeps changing. Do not run a blanket porting pass.

If you are asked to pull a specific upstream fix, use the sibling local checkout at `../cheesy-arena` (do not fetch from GitHub unless explicitly asked), and port that change by hand. Do not reintroduce game-mode branching, driver station or PLC code, TBA/Nexus publishing, WPA key provisioning, or a third team per alliance. `UPSTREAM.md` records the last upstream commit that was reviewed wholesale, which predates the XRP conversion.
