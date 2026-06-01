---
title: Web dashboard and API
tags: [px-dispatch, web, dashboard]
---

# Web dashboard and API

`px dashboard --web` boots an HTTP server (default :7890) that exposes:

| Endpoint | Returns |
|---|---|
| `GET /api/about` | Project metadata — name, tagline, version, description, features, pipeline stages, runtimes |
| `GET /api/health` | `{status: "ok", uptime: 12345}` |
| `GET /api/requirements` | Non-archived requirements |
| `GET /api/stories?req_id=…&status=…&limit=…&offset=…` | Stories |
| `GET /api/agents?status=…` | Agents |
| `GET /api/events?type=…&limit=…` | Recent events |
| `GET /api/escalations` | Open + resolved escalations |
| `GET /api/cost?req_id=…&story_id=…` | Daily / req / story cost in USD |
| `GET /api/stream` | Server-Sent Events stream |
| `GET /metrics` | Prometheus text — build info, uptime, status counts, cost totals |
| `GET /` | embedded SPA |

The SPA is bundled with `//go:embed`, so `./px dashboard --web` works on a
clean host — no separate npm build to deploy.

## SSE event stream

`GET /api/stream` reads the same projector channel that `px.db` consumes.
The SPA subscribes once on load and re-renders only the panels whose
underlying data changed. No polling.

## The About endpoint as the project's source of truth

`internal/web/handlers.go::projectAbout` is the canonical project
description. Anything that needs to know "what is this thing" — SPA About
tab, GitHub repo description, README — reads from this single struct.
`TestGetAbout` keeps the README and the struct in sync by failing when they
drift.

## TUI vs Web

`px dashboard` (no flag) launches the TUI via Bubbletea. Six panels:
pipeline kanban, agents, activity stream, escalations, cost, logs. Same
data, different shell.

Pick **web** when you want a watch window or to share the URL. Pick **TUI**
when you're already in a terminal flow or remote-tunnelled.

## Metrics endpoint (`GET /metrics`)

Returns plain-text Prometheus exposition format — no extra dependency
needed at the scraper side. Use this for local Grafana dashboards or to
trip alerts when the orchestrator is running unattended.

Series exposed:

- `px_build_info{version="…"}` — gauge, always `1`; carry-tag for build version
- `px_uptime_seconds` — gauge, seconds since process start
- `px_requirements_total{status="…"}` — gauge, per-status row count
- `px_stories_total{status="…"}` — gauge
- `px_agents_total{status="…"}` — gauge
- `px_escalations_total` — gauge
- `px_cost_usd_total` — counter, sum of `token_usage.cost_usd`
- `px_cost_usd_today` — gauge, UTC-day window of the above
- `px_cost_daily_limit_usd` — gauge, when `budget.max_cost_per_day_usd` is set

The handler reads directly from the SQLite projections; if no `px.db` is
attached (e.g. unit-test wiring) only the build/uptime series are emitted.

## Server lifecycle

`runWebDashboard(ctx, port, bind)` honours context cancellation. SIGINT in
the calling shell propagates through `signal.Notify` and the server's
`Shutdown` is called with the same context. Returns within a second of the
signal under normal load.
