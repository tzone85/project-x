package state

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Migrator runs forward-only SQL migrations tracked in a schema_migrations table.
type Migrator struct {
	db        *sql.DB
	migrations fs.FS
}

// NewMigrator creates a migrator that reads SQL files from the given embedded FS
// and applies them against the given database connection.
func NewMigrator(db *sql.DB, migrations fs.FS) *Migrator {
	return &Migrator{db: db, migrations: migrations}
}

// Migrate runs all pending migrations in order. It creates the schema_migrations
// table if it does not exist. Each migration is run in a transaction.
func (m *Migrator) Migrate() error {
	if err := m.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	applied, err := m.appliedVersions()
	if err != nil {
		return fmt.Errorf("read applied versions: %w", err)
	}

	pending, err := m.pendingMigrations(applied)
	if err != nil {
		return fmt.Errorf("find pending migrations: %w", err)
	}

	for _, mig := range pending {
		if err := m.applyMigration(mig); err != nil {
			return fmt.Errorf("apply migration %s: %w", mig.name, err)
		}
	}

	return nil
}

// migration represents a single SQL migration file.
type migration struct {
	name    string
	content string
}

func (m *Migrator) ensureMigrationsTable() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)
	`)
	return err
}

func (m *Migrator) appliedVersions() (map[string]bool, error) {
	rows, err := m.db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

func (m *Migrator) pendingMigrations(applied map[string]bool) ([]migration, error) {
	entries, err := fs.ReadDir(m.migrations, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var pending []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if applied[entry.Name()] {
			continue
		}

		content, err := fs.ReadFile(m.migrations, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		pending = append(pending, migration{
			name:    entry.Name(),
			content: string(content),
		})
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].name < pending[j].name
	})

	return pending, nil
}

func (m *Migrator) applyMigration(mig migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(mig.content); err != nil {
		return fmt.Errorf("exec sql: %w", err)
	}

	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version) VALUES (?)",
		mig.name,
	); err != nil {
		return fmt.Errorf("record version: %w", err)
	}

	return tx.Commit()
}

// AppliedVersions returns the list of applied migration versions.
func (m *Migrator) AppliedVersions() ([]string, error) {
	if err := m.ensureMigrationsTable(); err != nil {
		return nil, err
	}

	applied, err := m.appliedVersions()
	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, len(applied))
	for v := range applied {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	return versions, nil
}
