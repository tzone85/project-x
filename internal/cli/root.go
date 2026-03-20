package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	version = "dev"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "px",
		Short: "Project X — AI agent orchestration for the full SDLC",
		Long:  "Orchestrate autonomous AI agents from requirements to merged PRs.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./px.yaml)")
	cmd.AddCommand(newVersionCmd())
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("px %s\n", version)
		},
	}
}

func Execute() error {
	return NewRootCmd().Execute()
}
