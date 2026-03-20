package runtime

import "context"

// AgentStatus represents the detected status of an agent in a session.
type AgentStatus string

const (
	StatusIdle     AgentStatus = "idle"
	StatusRunning  AgentStatus = "running"
	StatusFinished AgentStatus = "finished"
	StatusErrored  AgentStatus = "errored"
	StatusUnknown  AgentStatus = "unknown"
)

// HealthStatus represents the health of a runtime session.
type HealthStatus string

const (
	HealthHealthy HealthStatus = "healthy"
	HealthStale   HealthStatus = "stale"
	HealthDead    HealthStatus = "dead"
	HealthMissing HealthStatus = "missing"
)

// SessionConfig holds parameters for spawning a new agent session.
type SessionConfig struct {
	SessionName string
	WorkDir     string
	Prompt      string
	Model       string
	Role        string
	StoryID     string
	ExtraArgs   []string
}

// RuntimeCapabilities describes what a runtime supports.
type RuntimeCapabilities struct {
	SupportedModels    []string
	SupportsGodmode    bool
	SupportsLogFile    bool
	SupportsJsonOutput bool
	MaxPromptLength    int
}

// Runtime is the interface that all agent runtimes must implement.
// Each runtime encapsulates a CLI tool (Claude Code, Codex, Gemini)
// and manages tmux sessions for that tool.
type Runtime interface {
	Name() string
	Version(ctx context.Context) (string, error)
	Spawn(ctx context.Context, cfg SessionConfig) error
	Kill(ctx context.Context, sessionName string) error
	DetectStatus(ctx context.Context, sessionName string) (AgentStatus, error)
	ReadOutput(ctx context.Context, sessionName string, lines int) (string, error)
	Health(ctx context.Context, sessionName string) (HealthStatus, error)
	SendInput(ctx context.Context, sessionName string, input string) error
	Capabilities() RuntimeCapabilities
}

// CostTier classifies runtimes by their cost model.
type CostTier int

const (
	// TierSubscription means the runtime is included in a subscription (e.g., Claude Code CLI).
	TierSubscription CostTier = iota
	// TierAPIBased means the runtime charges per API call.
	TierAPIBased
)

// RoutingPreference defines which runtime to prefer for a given role.
type RoutingPreference struct {
	Role     string
	Prefer   string
	Fallback string
}

// RoutingConfig holds routing strategy and role-based preferences.
type RoutingConfig struct {
	Strategy    string              // "cost_optimized" or "performance"
	Preferences []RoutingPreference
}

// DefaultRoutingConfig returns the spec-defined routing defaults.
func DefaultRoutingConfig() RoutingConfig {
	return RoutingConfig{
		Strategy: "cost_optimized",
		Preferences: []RoutingPreference{
			{Role: "junior", Prefer: "codex", Fallback: "claude-code"},
			{Role: "senior", Prefer: "claude-code", Fallback: "gemini"},
		},
	}
}
