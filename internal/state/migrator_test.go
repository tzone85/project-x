package state

import (
	"database/sql"
	"io/fs"
	"testing"
	"testing/fstest"

	_ "github.com/mattn/go-sqlite3"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrator_RunsMigrations(t *testing.T) {
	db := newTestDB(t)

	migrations := fstest.MapFS{
		"001_create_users.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT);"),
		},
		"002_create_posts.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE posts (id TEXT PRIMARY KEY, user_id TEXT, body TEXT);"),
		},
	}

	m := NewMigrator(db, migrations)
	if err := m.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verify tables exist
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&name)
	if err != nil {
		t.Fatalf("users table not found: %v", err)
	}

	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='posts'").Scan(&name)
	if err != nil {
		t.Fatalf("posts table not found: %v", err)
	}

	// Verify versions tracked
	versions, err := m.AppliedVersions()
	if err != nil {
		t.Fatalf("AppliedVersions: %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}
}

func TestMigrator_SkipsAppliedMigrations(t *testing.T) {
	db := newTestDB(t)

	migrations := fstest.MapFS{
		"001_create_users.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE users (id TEXT PRIMARY KEY);"),
		},
	}

	m := NewMigrator(db, migrations)
	if err := m.Migrate(); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}

	// Run again — should be idempotent
	if err := m.Migrate(); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	versions, err := m.AppliedVersions()
	if err != nil {
		t.Fatalf("AppliedVersions: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("expected 1 version, got %d", len(versions))
	}
}

func TestMigrator_RunsInOrder(t *testing.T) {
	db := newTestDB(t)

	// 002 depends on 001 (references the table)
	migrations := fstest.MapFS{
		"001_create_items.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE items (id TEXT PRIMARY KEY);"),
		},
		"002_add_column.sql": &fstest.MapFile{
			Data: []byte("ALTER TABLE items ADD COLUMN name TEXT;"),
		},
	}

	m := NewMigrator(db, migrations)
	if err := m.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verify the column exists
	_, err := db.Exec("INSERT INTO items (id, name) VALUES ('1', 'test')")
	if err != nil {
		t.Fatalf("insert failed — column likely missing: %v", err)
	}
}

func TestMigrator_IgnoresNonSQLFiles(t *testing.T) {
	db := newTestDB(t)

	migrations := fstest.MapFS{
		"001_init.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE test (id TEXT PRIMARY KEY);"),
		},
		"README.md": &fstest.MapFile{
			Data: []byte("# Migrations"),
		},
		"embed.go": &fstest.MapFile{
			Data: []byte("package migrations"),
		},
	}

	m := NewMigrator(db, migrations)
	if err := m.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	versions, err := m.AppliedVersions()
	if err != nil {
		t.Fatalf("AppliedVersions: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("expected 1 version (only .sql), got %d", len(versions))
	}
}

func TestMigrator_FailedMigrationRollsBack(t *testing.T) {
	db := newTestDB(t)

	migrations := fstest.MapFS{
		"001_good.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE good (id TEXT PRIMARY KEY);"),
		},
		"002_bad.sql": &fstest.MapFile{
			Data: []byte("INVALID SQL STATEMENT;"),
		},
	}

	m := NewMigrator(db, migrations)
	err := m.Migrate()
	if err == nil {
		t.Fatal("expected error from bad migration")
	}

	// First migration should have been applied
	versions, err := m.AppliedVersions()
	if err != nil {
		t.Fatalf("AppliedVersions: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("expected 1 version (only good), got %d", len(versions))
	}
}

func TestMigrator_EmptyFS(t *testing.T) {
	db := newTestDB(t)
	m := NewMigrator(db, fstest.MapFS{})
	if err := m.Migrate(); err != nil {
		t.Fatalf("Migrate with empty FS: %v", err)
	}
}

func TestMigrator_WithRealMigrations(t *testing.T) {
	db := newTestDB(t)

	// Use the actual embedded migrations
	realFS, err := fs.Sub(testMigrationsFS(), ".")
	if err != nil {
		t.Fatalf("fs.Sub: %v", err)
	}

	m := NewMigrator(db, realFS)
	if err := m.Migrate(); err != nil {
		t.Fatalf("Migrate with real migrations: %v", err)
	}

	// Verify key tables exist
	tables := []string{"requirements", "stories", "agents", "escalations",
		"token_usage", "session_health", "pipeline_runs", "events"}

	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}

	// Verify indexes
	indexes := []string{
		"idx_stories_req_id", "idx_stories_status", "idx_stories_req_status",
		"idx_agents_status", "idx_escalations_story_id",
		"idx_token_usage_story_id", "idx_token_usage_req_id", "idx_token_usage_date",
		"idx_pipeline_runs_story_id", "idx_events_type", "idx_events_created_at",
	}

	for _, idx := range indexes {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&name)
		if err != nil {
			t.Errorf("index %s not found: %v", idx, err)
		}
	}
}
