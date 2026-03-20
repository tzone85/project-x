package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tzone85/project-x/internal/state"
)

func newGCCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gc",
		Short: "Garbage collect old worktrees and branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGC()
		},
	}
}

// runGC removes worktrees under the state dir for archived or completed requirements.
func runGC() error {
	reqs, err := app.projStore.ListRequirements(state.ReqFilter{})
	if err != nil {
		return fmt.Errorf("list requirements: %w", err)
	}

	worktreesDir := filepath.Join(app.stateDir, "worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No worktrees found.")
			fmt.Println("Garbage collection complete.")
			return nil
		}
		return fmt.Errorf("read worktrees dir: %w", err)
	}

	// Build set of active requirement IDs (non-archived, non-completed).
	activeReqIDs := make(map[string]bool, len(reqs))
	for _, req := range reqs {
		if req.Status != "archived" && req.Status != "completed" {
			activeReqIDs[req.ID] = true
		}
	}

	removed := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if activeReqIDs[entry.Name()] {
			continue
		}
		target := filepath.Join(worktreesDir, entry.Name())
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("remove worktree %s: %w", target, err)
		}
		fmt.Printf("Removed worktree: %s\n", target)
		removed++
	}

	if removed == 0 {
		fmt.Println("Nothing to collect.")
	}
	fmt.Println("Garbage collection complete.")
	return nil
}
