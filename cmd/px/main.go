// Package main is the entry point for the px CLI.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/tzone85/project-x/internal/cli"
	"github.com/tzone85/project-x/internal/config"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var (
		configPath string
		verbose    bool
	)

	cmd := &cobra.Command{
		Use:     "px",
		Short:   "AI agent orchestration CLI",
		Long:    "Project X (px) drives the full software development lifecycle — from natural-language requirements to merged PRs.",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			logLevel := cfg.Workspace.LogLevel
			if verbose {
				logLevel = "debug"
			}

			cleanup, err := config.SetupLogging(logLevel, cfg.Workspace.DataDir)
			if err != nil {
				return fmt.Errorf("setting up logging: %w", err)
			}

			// Store cleanup and config in the command context for subcommands.
			cmd.SetContext(withCleanup(cmd.Context(), cleanup))
			cmd.SetContext(withConfig(cmd.Context(), cfg))

			slog.Info("px starting", "version", version, "config", configPath)
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if cleanup, ok := cleanupFromContext(cmd.Context()); ok {
				cleanup()
			}
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", config.DefaultConfigPath(), "path to config file")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable debug logging")

	// Config accessor for subcommands that need it
	cfgFn := func() config.Config {
		cfg, _ := ConfigFromContext(cmd.Context())
		return cfg
	}

	cmd.AddCommand(versionCmd())
	cmd.AddCommand(cli.NewInitCmd())
	cmd.AddCommand(cli.NewMigrateCmd(cfgFn))
	cmd.AddCommand(cli.NewCostCmd(cfgFn))
	cmd.AddCommand(cli.NewStatusCmd(cfgFn))
	cmd.AddCommand(cli.NewPlanCmd())
	cmd.AddCommand(cli.NewResumeCmd())

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version info",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("px version %s\n", version)
		},
	}
}
