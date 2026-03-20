CREATE TABLE IF NOT EXISTS session_health (
    session_name TEXT PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'unknown',
    pane_pid INTEGER NOT NULL DEFAULT 0,
    last_output_hash TEXT NOT NULL DEFAULT '',
    last_healthy_at TIMESTAMP,
    recovery_attempts INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
