package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Migrations run automatically when SQLiteStore opens in PersistentPreRunE.
			// This command confirms they are applied.
			fmt.Printf("Database is up to date at %s\n", app.stateDir)
			return nil
		},
	}
}
