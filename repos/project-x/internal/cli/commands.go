package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version info",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("px %s\n", version)
		},
	}
}

func newPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan [requirement-file]",
		Short: "Decompose requirement into stories",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("requirement file or --review/--refine flag required")
			}
			cmd.Printf("Planning from: %s\n", args[0])
			return nil
		},
	}

	var reviewReqID string
	var refineReqID string

	cmd.Flags().StringVar(&reviewReqID, "review", "", "Display plan for inspection")
	cmd.Flags().StringVar(&refineReqID, "refine", "", "Re-plan with user feedback")

	return cmd
}

func newResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <req-id>",
		Short: "Dispatch approved plan and start monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("Resuming requirement: %s\n", args[0])
			return nil
		},
	}
}

func newCostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Show spending by story, requirement, day",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Cost report:")
			cmd.Println("  (no data yet)")
			return nil
		},
	}

	cmd.Flags().Bool("update-prices", false, "Fetch latest pricing (future)")

	return cmd
}

func newDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch dashboard",
		RunE: func(cmd *cobra.Command, _ []string) error {
			web, _ := cmd.Flags().GetBool("web")
			port, _ := cmd.Flags().GetInt("port")

			if web {
				cmd.Printf("Starting web dashboard on port %d\n", port)
			} else {
				cmd.Println("Starting TUI dashboard")
			}
			return nil
		},
	}

	cmd.Flags().Bool("web", false, "Launch browser dashboard")
	cmd.Flags().Int("port", 7890, "Web dashboard port")

	return cmd
}

func newMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Run pending database migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Running migrations...")
			return nil
		},
	}
}
