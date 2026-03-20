package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tzone85/project-x/internal/config"
	"github.com/tzone85/project-x/internal/state"

	_ "github.com/mattn/go-sqlite3"
)

// NewMigrateCmd creates the `px migrate` command that runs database migrations.
func NewMigrateCmd(cfgFn func() config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		Long:  "Applies any pending forward-only SQL migrations to the SQLite projection database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := cfgFn()
			return runMigrate(cfg.Workspace.DataDir)
		},
	}
}

func runMigrate(dataDir string) error {
	dbPath := filepath.Join(dataDir, "projections.db")
	store, err := state.NewProjectionStore(dbPath)
	if err != nil {
		return fmt.Errorf("open projection store: %w", err)
	}
	defer store.Close()

	fmt.Println("Migrations applied successfully.")
	return nil
}
