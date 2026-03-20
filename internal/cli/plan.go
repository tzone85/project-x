package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewPlanCmd creates the `px plan` command for requirement decomposition.
func NewPlanCmd() *cobra.Command {
	var (
		review bool
		refine bool
	)

	cmd := &cobra.Command{
		Use:   "plan [requirement-file]",
		Short: "Decompose a requirement into stories",
		Long: `Run the two-pass planner to decompose a requirement file into stories.

Usage:
  px plan <requirement-file>     # plan only
  px plan --review <req-id>      # inspect existing plan
  px plan --refine <req-id>      # re-plan with feedback`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if review && len(args) == 1 {
				return runPlanReview(args[0])
			}
			if refine && len(args) == 1 {
				return runPlanRefine(args[0])
			}
			if len(args) == 1 {
				return runPlan(args[0])
			}
			return fmt.Errorf("provide a requirement file or --review/--refine with a requirement ID")
		},
	}

	cmd.Flags().BoolVar(&review, "review", false, "inspect an existing plan by requirement ID")
	cmd.Flags().BoolVar(&refine, "refine", false, "re-plan a requirement with feedback")

	return cmd
}

func runPlan(requirementFile string) error {
	fmt.Printf("Planning from: %s\n", requirementFile)
	fmt.Println("Two-pass planner not yet wired to LLM client. Use this as a placeholder.")
	return nil
}

func runPlanReview(reqID string) error {
	fmt.Printf("Reviewing plan for requirement: %s\n", reqID)
	fmt.Println("Plan review not yet implemented.")
	return nil
}

func runPlanRefine(reqID string) error {
	fmt.Printf("Refining plan for requirement: %s\n", reqID)
	fmt.Println("Plan refinement not yet implemented.")
	return nil
}
