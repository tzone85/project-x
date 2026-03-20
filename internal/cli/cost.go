package cli

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/tzone85/project-x/internal/config"
	"github.com/tzone85/project-x/internal/state"

	_ "github.com/mattn/go-sqlite3"
)

// NewCostCmd creates the `px cost` command that shows spending summaries.
func NewCostCmd(cfgFn func() config.Config) *cobra.Command {
	var (
		storyID string
		reqID   string
		daily   bool
	)

	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Show cost spending summary",
		Long:  "Display token spending by story, requirement, or today's total.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := cfgFn()
			return runCost(cfg.Workspace.DataDir, storyID, reqID, daily)
		},
	}

	cmd.Flags().StringVar(&storyID, "story", "", "show cost for a specific story ID")
	cmd.Flags().StringVar(&reqID, "requirement", "", "show cost for a specific requirement ID")
	cmd.Flags().BoolVar(&daily, "today", false, "show today's total cost")

	return cmd
}

func runCost(dataDir, storyID, reqID string, daily bool) error {
	dbPath := filepath.Join(dataDir, "projections.db")
	queries, err := openReadOnly(dbPath)
	if err != nil {
		return err
	}
	defer queries.Close()

	if storyID != "" {
		total, err := queries.GetStoryTotalCost(storyID)
		if err != nil {
			return fmt.Errorf("get story cost: %w", err)
		}
		fmt.Printf("Story %s: $%.4f\n", storyID, total)
		return nil
	}

	if reqID != "" {
		total, err := queries.GetRequirementTotalCost(reqID)
		if err != nil {
			return fmt.Errorf("get requirement cost: %w", err)
		}
		fmt.Printf("Requirement %s: $%.4f\n", reqID, total)
		return nil
	}

	if daily {
		total, err := queries.GetDailyTotalCost(time.Now())
		if err != nil {
			return fmt.Errorf("get daily cost: %w", err)
		}
		fmt.Printf("Today's total: $%.4f\n", total)
		return nil
	}

	// Default: show today's total
	total, err := queries.GetDailyTotalCost(time.Now())
	if err != nil {
		return fmt.Errorf("get daily cost: %w", err)
	}
	fmt.Printf("Today's total: $%.4f\n", total)
	return nil
}

func openReadOnly(dbPath string) (*state.ReadOnlyQueries, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&mode=ro&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open read-only db: %w", err)
	}
	return state.NewReadOnlyQueries(db), nil
}
