package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive <req-id>",
		Short: "Archive a completed requirement",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqID := args[0]

			if err := app.projStore.ArchiveRequirement(reqID); err != nil {
				return fmt.Errorf("archive requirement %s: %w", reqID, err)
			}

			if err := app.projStore.ArchiveStoriesByReq(reqID); err != nil {
				return fmt.Errorf("archive stories for requirement %s: %w", reqID, err)
			}

			fmt.Printf("Archived requirement %s and its stories.\n", reqID)
			return nil
		},
	}
}
