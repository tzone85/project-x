# Project X (px)

**AI agent orchestration for the full software development lifecycle.**

Hand off a requirement, walk away, come back to merged PRs.

## Features

- **Autonomous Agents**: Decompose requirements into stories, dispatch AI agents, auto-merge results
- **Cost Protection**: Budget enforcement with circuit breakers — never run away on API spend
- **Multi-Runtime**: Pluggable support for Claude Code, Codex, and Gemini CLIs
- **Smart Planning**: Two-pass planner with quality validation and dependency DAGs
- **Pipeline Stages**: Review, QA, rebase with LLM conflict resolution, auto-merge
- **Full Observability**: Scrollable TUI dashboard + browser-based web dashboard
- **Session Health**: Detect stale/dead tmux sessions, auto-recover
- **Event-Sourced**: Append-only event log + SQLite projections for auditability

## Quick Start

### Install

```bash
# From source
go install github.com/tzone85/project-x/cmd/px@latest

# Or build from source
git clone https://github.com/tzone85/project-x.git
cd project-x
make build
```

### Prerequisites

- Go 1.22+
- tmux
- git
- gh (GitHub CLI)
- At least one AI runtime: `claude` (Claude Code), `codex`, or `gemini`

### Usage

```bash
# 1. Initialize (creates ~/.px/ with config and database)
px migrate

# 2. Plan: decompose a requirement into stories
echo "Add user authentication with OAuth2" | px plan -

# 3. Review the plan
px plan --review <req-id>

# 4. Dispatch agents and monitor pipeline
px resume <req-id>

# 5. Watch progress
px dashboard          # TUI mode
px dashboard --web    # Browser mode (http://localhost:7890)

# 6. Check costs
px cost

# 7. Check status
px status
```

## Architecture

```
requirement -> planner -> stories + DAG -> dispatcher -> agents (parallel waves)
                                                            |
merged PRs <- merge <- rebase <- QA <- review <- agent completion
```

### Key Components

| Component | Package | Purpose |
|-----------|---------|---------|
| State | `internal/state` | Event-sourced store (JSONL + SQLite projections) |
| LLM | `internal/llm` | Client interface + Claude CLI, Anthropic, OpenAI |
| Cost | `internal/cost` | Ledger, circuit breaker, pricing |
| Planner | `internal/planner` | Two-pass requirement decomposition |
| Pipeline | `internal/pipeline` | 7 composable stages (review to merge) |
| Runtime | `internal/runtime` | Pluggable AI CLI runtimes |
| Monitor | `internal/monitor` | Dispatcher, executor, poller, watchdog |
| Dashboard | `internal/dashboard` | Bubbletea TUI |
| Web | `internal/web` | REST API + SSE + embedded SPA |

## Configuration

Copy and customize the example config:

```bash
cp px.config.example.yaml px.yaml
```

Key settings: budget limits, routing preferences, runtime definitions, pipeline retry policies.

See `px.config.example.yaml` for all options.

## License

Apache 2.0
