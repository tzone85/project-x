CREATE TABLE IF NOT EXISTS token_usage (
    id TEXT PRIMARY KEY,
    req_id TEXT NOT NULL,
    story_id TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    cost_usd REAL NOT NULL DEFAULT 0.0,
    stage TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_token_usage_story_id ON token_usage(story_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_req_id ON token_usage(req_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_date ON token_usage(created_at);
