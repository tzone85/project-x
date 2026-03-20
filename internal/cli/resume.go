package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewResumeCmd creates the `px resume` command that dispatches an approved plan.
func NewResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <req-id>",
		Short: "Dispatch an approved plan for execution",
		Long:  "Resume or start execution of a planned requirement by dispatching wave 1 stories to agent sessions.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResume(args[0])
		},
	}
}

func runResume(reqID string) error {
	fmt.Printf("Resuming requirement: %s\n", reqID)
	fmt.Println("Agent dispatch not yet wired to monitor/runtime. Use this as a placeholder.")
	return nil
}
