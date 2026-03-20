package modelswitch

import "strings"

// Scope identifies where a model switch is happening.
type Scope string

const (
	ScopeLLM     Scope = "llm"
	ScopeRuntime Scope = "runtime"
)

// Request describes a proposed switch from Claude to a fallback model/runtime.
type Request struct {
	Scope           Scope
	Operation       string
	StoryID         string
	StoryTitle      string
	CurrentProvider string
	CurrentRuntime  string
	TargetProvider  string
	TargetRuntime   string
	TargetModel     string
	Reason          string
	Note            string
}

// Approver decides whether a fallback switch should proceed.
type Approver interface {
	ApproveSwitch(req Request) (bool, error)
}

// DetectClaudeExhaustion normalizes account-limit/balance errors into a reason
// string suitable for user-facing approval prompts.
func DetectClaudeExhaustion(text string) (string, bool) {
	lower := strings.ToLower(text)

	switch {
	case containsAny(lower, "credit balance", "insufficient credits", "billing"):
		return "Anthropic balance is too low", true
	case containsAny(lower, "out of extra usage", "extra usage"):
		return "Claude usage limits have been reached", true
	case containsAny(lower, "usage limit", "limit reached", "subscription limit"):
		return "Claude usage limits have been reached", true
	case containsAny(lower, "quota exceeded", "quota has been exceeded"):
		return "Claude quota has been exceeded", true
	case containsAny(lower, "try again at", "try again after") && containsAny(lower, "claude", "anthropic"):
		return "Claude account limits are temporarily exhausted", true
	default:
		return "", false
	}
}

func containsAny(s string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(s, part) {
			return true
		}
	}
	return false
}
