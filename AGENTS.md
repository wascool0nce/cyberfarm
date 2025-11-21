# Repository Guidelines

## Project Structure & Module Organization
- `cmd/cyberfarm/main.go` configures the Ebiten window, builds the game service, and hands control to the engine.
- `internal/domain` defines entities (`GameState`, `Plot`), tunable constants, plant tables, and helper math such as `ExpandPlotCost`.
- `internal/usecase` hosts `Service`, the authoritative state machine for growth ticks, story events, scoring, and player actions.
- `internal/adapter/ebitengame` renders sprites, HUD, and menus via Ebiten; treat it as the only package that touches graphics or input.
- `BUILD.md` records prerequisites, default controls, and the same commands listed below—skim it before updating UX.

## Build, Test, and Development Commands
- `go run ./cmd/cyberfarm` launches the interactive farm; keep a terminal running for manual verification.
- `go build -o bin/cyberfarm ./cmd/cyberfarm` produces a reusable binary for demos or releases.
- `go test ./...` is fast even without suites today—run it (and add tests) whenever you touch logic.
- `go fmt ./...` and `go vet ./cmd/... ./internal/...` keep formatting and static checks aligned with upstream Go tooling.

## Coding Style & Naming Conventions
- Rely on `gofmt` (tabs + aligned structs) and idiomatic Go naming: exported types use PascalCase, unexported helpers use camelCase verbs.
- Keep business rules in `usecase`, pure data in `domain`, and UI plumbing in `adapter`; avoid cyclic imports by honoring that layering.
- Prefer explicit durations/costs from `domain/constants.go` instead of recoding literals inside loops or UI text.

## Testing Guidelines
- Add `_test.go` files beside the code they exercise—table-driven cases for scoring, growth timing, and event resolution pay off quickly.
- Until suites exist, perform a manual sanity pass: run the binary, unlock a seed, plant/water/harvest, resolve a story event, and reach the coin goal to confirm the HUD win banner.
- Record manual steps in the PR body so others can replay the scenario.

## Commit & Pull Request Guidelines
- Write imperative summaries (“Add carrot spoil warning”) and keep gameplay, UI, and balance tweaks in separate commits when possible.
- PRs should include intent, affected packages, manual/automated test notes, and screenshots or GIFs for any UI or HUD change.
- Call out new constants or tunables so reviewers can double-check balancing implications before merging.

## Architecture & Gameplay Notes
- The render loop asks `Service.MaybeTick()` once per frame; heavy work should stay outside Ebiten draw hooks to keep 60 FPS.
- Story events pause ticking—always clear `PausedForEvent` when resolving choices to avoid soft locks.
