package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewInitCmd creates the `px init` command that scaffolds a new project workspace.
func NewInitCmd() *cobra.Command {
	var dataDir string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new px workspace",
		Long:  "Creates the data directory, default config file, and empty event log.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(dataDir)
		},
	}

	cmd.Flags().StringVar(&dataDir, "data-dir", defaultDataDir(), "workspace data directory")
	return cmd
}

func runInit(dataDir string) error {
	dirs := []string{
		dataDir,
		filepath.Join(dataDir, "logs"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	eventsPath := filepath.Join(dataDir, "events.jsonl")
	if _, err := os.Stat(eventsPath); os.IsNotExist(err) {
		f, err := os.Create(eventsPath)
		if err != nil {
			return fmt.Errorf("create events file: %w", err)
		}
		f.Close()
	}

	configPath := "px.config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(defaultConfigYAML), 0644); err != nil {
			return fmt.Errorf("create config file: %w", err)
		}
		fmt.Printf("Created %s\n", configPath)
	}

	fmt.Printf("Workspace initialized at %s\n", dataDir)
	return nil
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".px"
	}
	return filepath.Join(home, ".px")
}

const defaultConfigYAML = `# px configuration
# See docs/config-reference.md for full options

budget:
  max_cost_per_story_usd: 2.00
  max_cost_per_requirement_usd: 20.00
  max_cost_per_day_usd: 50.00
  warning_threshold_pct: 80
  hard_stop: true

sessions:
  stale_threshold_s: 180
  on_dead: redispatch
  on_stale: restart
  max_recovery_attempts: 2

pipeline:
  stages:
    review:
      max_retries: 2
      on_exhaust: escalate
    qa:
      max_retries: 3
      on_exhaust: pause_requirement
    rebase:
      max_retries: 2
      on_exhaust: pause_requirement
    merge:
      max_retries: 1
      on_exhaust: pause_requirement

routing:
  strategy: cost_optimized
  preferences:
    - role: junior
      prefer: codex
      fallback: claude-code
    - role: senior
      prefer: claude-code
      fallback: gemini
`
