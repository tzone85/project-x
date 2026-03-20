package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newConfigShowCmd(), newConfigValidateCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the current configuration as YAML",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := yaml.Marshal(app.config)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}
			fmt.Print(string(data))
			return nil
		},
	}
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the current configuration and report any errors",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.config.Validate(); err != nil {
				return fmt.Errorf("config validation failed: %w", err)
			}
			fmt.Println("Configuration is valid.")
			return nil
		},
	}
}
