package cli

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tzone85/project-x/internal/config"
	"github.com/tzone85/project-x/internal/state"

	_ "github.com/mattn/go-sqlite3"
)

// NewStatusCmd creates the `px status` command that shows system overview.
func NewStatusCmd(cfgFn func() config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show system status overview",
		Long:  "Display counts of requirements, stories, agents, and recent events.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := cfgFn()
			return runStatus(cfg.Workspace.DataDir)
		},
	}
}

func runStatus(dataDir string) error {
	dbPath := filepath.Join(dataDir, "projections.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&mode=ro&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	queries := state.NewReadOnlyQueries(db)
	defer queries.Close()

	page := state.PageParams{Limit: 1000}

	reqs, err := queries.ListRequirements(page)
	if err != nil {
		return fmt.Errorf("list requirements: %w", err)
	}

	stories, err := queries.ListStories(page)
	if err != nil {
		return fmt.Errorf("list stories: %w", err)
	}

	agents, err := queries.ListAgents(page)
	if err != nil {
		return fmt.Errorf("list agents: %w", err)
	}

	// Count stories by status
	statusCounts := make(map[string]int)
	for _, s := range stories {
		statusCounts[s.Status]++
	}

	fmt.Printf("Requirements: %d\n", len(reqs))
	fmt.Printf("Stories:      %d\n", len(stories))
	for status, count := range statusCounts {
		fmt.Printf("  %-15s %d\n", status, count)
	}
	fmt.Printf("Agents:       %d\n", len(agents))

	activeAgents := 0
	for _, a := range agents {
		if a.Status == "active" || a.Status == "running" {
			activeAgents++
		}
	}
	fmt.Printf("  active:       %d\n", activeAgents)

	return nil
}
