-- 001_init.sql: Initial schema for Project X
-- All tables and indexes per Section 5.2

CREATE TABLE IF NOT EXISTS requirements (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source      TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'draft',
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS stories (
    id                  TEXT PRIMARY KEY,
    req_id              TEXT NOT NULL REFERENCES requirements(id),
    title               TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    acceptance_criteria TEXT NOT NULL DEFAULT '',
    owned_files         TEXT NOT NULL DEFAULT '[]', -- JSON array
    complexity          INTEGER NOT NULL DEFAULT 1,
    depends_on          TEXT NOT NULL DEFAULT '[]', -- JSON array
    status              TEXT NOT NULL DEFAULT 'planned',
    agent_id            TEXT,
    wave                INTEGER NOT NULL DEFAULT 0,
    created_at          DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at          DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS agents (
    id            TEXT PRIMARY KEY,
    role          TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'idle',
    current_story TEXT,
    session_name  TEXT,
    runtime       TEXT,
    created_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at    DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS escalations (
    id         TEXT PRIMARY KEY,
    story_id   TEXT NOT NULL REFERENCES stories(id),
    reason     TEXT NOT NULL,
    from_role  TEXT NOT NULL,
    to_role    TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'open',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    resolved_at DATETIME
);

CREATE TABLE IF NOT EXISTS token_usage (
    id            TEXT PRIMARY KEY,
    story_id      TEXT NOT NULL,
    req_id        TEXT NOT NULL,
    agent_id      TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL,
    input_tokens  INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd      REAL NOT NULL DEFAULT 0.0,
    stage         TEXT NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS session_health (
    session_name     TEXT PRIMARY KEY,
    status           TEXT NOT NULL DEFAULT 'unknown',
    pane_pid         INTEGER NOT NULL DEFAULT 0,
    last_output_hash TEXT NOT NULL DEFAULT '',
    recovery_attempts INTEGER NOT NULL DEFAULT 0,
    last_check_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at       DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS pipeline_runs (
    id         TEXT PRIMARY KEY,
    story_id   TEXT NOT NULL REFERENCES stories(id),
    stage      TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'running',
    attempt    INTEGER NOT NULL DEFAULT 1,
    error      TEXT NOT NULL DEFAULT '',
    started_at DATETIME NOT NULL DEFAULT (datetime('now')),
    ended_at   DATETIME,
    duration_ms INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS events (
    id         TEXT PRIMARY KEY,
    type       TEXT NOT NULL,
    payload    TEXT NOT NULL DEFAULT '{}', -- JSON
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- Indexes per Section 5.2
CREATE INDEX IF NOT EXISTS idx_stories_req_id ON stories(req_id);
CREATE INDEX IF NOT EXISTS idx_stories_status ON stories(status);
CREATE INDEX IF NOT EXISTS idx_stories_req_status ON stories(req_id, status);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_escalations_story_id ON escalations(story_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_story_id ON token_usage(story_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_req_id ON token_usage(req_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_date ON token_usage(created_at);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_story_id ON pipeline_runs(story_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
