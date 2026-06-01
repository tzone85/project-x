# Production Readiness — px-dispatch

Status: approved
Date: 2026-06-01
Owner: Thando Mini

## Goal

Take px-dispatch from "well-tested prototype" to "ship-able OSS tool" — installable via brew/go install/docker/GitHub release, with blocking quality gates, hardened operations, and a tagged `v0.1.0`.

## Baseline (as of branch)

- Build/vet clean. All tests pass with `-race`.
- Coverage 78–100 % per `internal/*` package.
- 58 lint findings (50 errcheck on unchecked `Close()`, 7 staticcheck nits, 1 unused var). CI lint is `continue-on-error`.
- Committed cruft: `px` binary (16 MB), `firebase-debug.log`, stale `coverage.out`.
- `mattn/go-sqlite3` (CGO) blocks pure cross-compile.
- GoReleaser config present but unused; no release workflow.
- No Docker image, no brew tap, no SECURITY.md.

## Non-goals

- New product features.
- OpenTelemetry tracing (premature for current single-user CLI).
- Multi-repo orchestration.
- Pure-Go agent for Windows tmux equivalent.

## Approach — four sequenced PRs

Each PR leaves the repo in a working, mergeable state.

### Phase 1 — Hygiene + quality gates

Goal: stop the bleeding. Future PRs cannot erode quality once gates are blocking.

- Delete tracked binaries and stale artifacts: `px`, `firebase-debug.log`, `coverage.out`.
- Extend `.gitignore` to keep them out (`px`, `*.log`, `coverage.out`, `dist/`).
- Fix the 58 lint findings:
  - errcheck on `Close()`: wrap with `_ = x.Close()` in test cleanup or check and log in production code (`internal/state/sqlite.go`, `internal/web/server.go`, `internal/web/sse.go`).
  - staticcheck nits: `internal/cli/plan_helpers_test.go`, `internal/dashboard/activity_test.go`, `internal/llm/claude_cli.go`, `internal/llm/errors.go`, `internal/monitor/executor.go`.
  - delete unused `panelBorderStyle` in `internal/dashboard/styles.go`.
- Flip `.github/workflows/ci.yml` lint step to blocking (remove `continue-on-error`).
- Add coverage gate: fail CI if total coverage drops below 75 % (current ≈85 %).
- Add `govulncheck` step in CI.
- Add `dependabot.yml` for `gomod` + `github-actions`.

### Phase 2 — CGO drop + release pipeline

Goal: installable everywhere.

- Replace `github.com/mattn/go-sqlite3` with `modernc.org/sqlite` (pure Go). Driver name `sqlite` (not `sqlite3`); update `sql.Open` call sites in `internal/state` and any migration code.
- Rewrite `.goreleaser.yml` for pure-Go cross-compile across `darwin/linux/windows × amd64/arm64`. Add SBOM (cyclonedx) and checksums.
- Add `.github/workflows/release.yml` triggered on `v*` tags. Calls GoReleaser, publishes to GitHub Releases, brew tap, ghcr.io.
- Brew tap: new repo `tzone85/homebrew-tap`; GoReleaser auto-publishes formula.
- Docker: multi-arch image `ghcr.io/tzone85/px-dispatch:<tag>` built in release workflow. Image is small (distroless or alpine + ca-certificates + tmux + git).
- `go install` continues to work — verified by smoke job in CI.

### Phase 3 — Ops hardening + metrics

Goal: long-running orchestration is observable and crash-resistant.

- `internal/logging`: add log-level config (debug/info/warn/error), file-output target (default `$PX_HOME/px.log`), structured JSON option.
- Request IDs (ULIDs) propagated through planner → dispatcher → pipeline stages → state events; surface in logs and `/api/events`.
- Add `/metrics` endpoint on the web server: counters/gauges for stories planned, stories merged, agent states, pipeline stage outcomes, cost totals, budget warnings. Plain Prometheus text format, no new deps if we hand-roll (or pull `prometheus/client_golang` if simpler).
- Panic recovery audit: every goroutine launched in `internal/monitor`, `internal/pipeline`, `internal/web` must have `defer recover` that logs + emits an event.
- Graceful shutdown audit: SIGINT/SIGTERM drains in-flight pipeline runs, flushes event store, kills tmux sessions cleanly.
- Event log retention: `px gc` learns to compact JSONL events older than configurable days into a `.archive` while preserving SQLite projection integrity.

### Phase 4 — OSS polish + Obsidian + v0.1.0

Goal: tag a release.

- Add `SECURITY.md` (private vulnerability disclosure via GitHub).
- Add `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1).
- Generate `CHANGELOG.md` from conventional commits using `git-cliff` or GoReleaser's changelog feature.
- Update `docs/obsidian/` lessons to reflect the new release flow, metrics endpoint, log levels, and CGO-free build. Re-run `make sync-vault`.
- Tag `v0.1.0`. Release workflow fires. Verify all four channels deliver.

## Risks

| Risk | Mitigation |
|------|------------|
| `modernc.org/sqlite` behavior diff vs `mattn/go-sqlite3` (locking, transactions) | Full test suite already exercises state pkg at 78 %; run e2e tests on the swap branch. |
| Brew tap repo doesn't exist yet | Create empty `tzone85/homebrew-tap` before Phase 2 release dry-run; GoReleaser writes formula automatically. |
| Blocking CI lint surfaces hidden errors on first PR | Phase 1 fixes the 58 known issues before flipping the switch. |
| Coverage gate at 75 % may flap on monitor/state | Per-package floor would be stricter but harder to reason about; total-coverage gate is the simplest contract. Tune up later. |
| `go.mod` says `go 1.26.1` — bleeding-edge runner availability | CI uses `go-version-file: go.mod`; tested by setup-go's automatic toolchain resolution. |

## Architecture decisions

- **Event sourcing stays.** No change to events.jsonl → SQLite projection model.
- **Pure-Go SQLite is non-negotiable.** Phase 2 is gated on this.
- **No new heavyweight deps in Phase 3.** Hand-roll metrics if `client_golang` adds more weight than value (decide during Phase 3 spike).
- **One module, one binary, one CLI.** Don't fragment for the release.

## Out of scope (deferred to v0.2.0+)

- Webhook notifications.
- Multi-repo orchestration.
- Agent reputation scoring.
- OTel tracing.
- VHS demo tape automation (manual regen remains documented in README).
