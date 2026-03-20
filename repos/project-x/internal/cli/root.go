// Package cli implements the Cobra command tree for the px CLI.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose bool
	version = "dev"
)

// SetVersion sets the version string for the version command.
func SetVersion(v string) {
	version = v
}

// NewRootCmd creates the root px command with all subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "px",
		Short: "AI Agent Orchestration CLI",
		Long:  "Project X (px) drives the full software development lifecycle from natural-language requirements to merged PRs.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug output")

	root.AddCommand(
		newVersionCmd(),
		newPlanCmd(),
		newResumeCmd(),
		newCostCmd(),
		newDashboardCmd(),
		newMigrateCmd(),
	)

	return root
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
