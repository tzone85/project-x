package modelswitch

import "testing"

func TestDetectClaudeExhaustion(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantReason string
		wantOK     bool
	}{
		// Credit/billing patterns.
		{"credit balance", `{"error":"credit balance is zero"}`, "Anthropic balance is too low", true},
		{"insufficient credits", "Error: insufficient credits for request", "Anthropic balance is too low", true},
		{"billing issue", "billing account suspended", "Anthropic balance is too low", true},

		// Extra usage patterns.
		{"out of extra usage", `You're out of extra usage · resets 7pm`, "Claude usage limits have been reached", true},
		{"extra usage caps", "extra usage has been exceeded", "Claude usage limits have been reached", true},

		// Usage limit patterns.
		{"usage limit", "usage limit reached for this period", "Claude usage limits have been reached", true},
		{"limit reached", "API limit reached, please wait", "Claude usage limits have been reached", true},
		{"subscription limit", "subscription limit for claude-sonnet", "Claude usage limits have been reached", true},

		// Quota patterns.
		{"quota exceeded", "Error: quota exceeded for project", "Claude quota has been exceeded", true},
		{"quota has been exceeded", "Your quota has been exceeded", "Claude quota has been exceeded", true},

		// Temporary exhaustion.
		{"try again at claude", "try again at 5pm. Claude is busy", "Claude account limits are temporarily exhausted", true},
		{"try again after anthropic", "try again after 10 minutes, anthropic API", "Claude account limits are temporarily exhausted", true},

		// Case insensitivity.
		{"mixed case", "USAGE LIMIT reached", "Claude usage limits have been reached", true},
		{"upper case extra", "OUT OF EXTRA USAGE", "Claude usage limits have been reached", true},

		// Non-matching.
		{"empty string", "", "", false},
		{"unrelated error", "connection timeout", "", false},
		{"try again without claude", "try again at 5pm", "", false},
		{"random text", "the quick brown fox", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, ok := DetectClaudeExhaustion(tt.input)
			if ok != tt.wantOK {
				t.Errorf("DetectClaudeExhaustion(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if reason != tt.wantReason {
				t.Errorf("DetectClaudeExhaustion(%q) reason = %q, want %q", tt.input, reason, tt.wantReason)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		parts []string
		want  bool
	}{
		{"matches first", "hello world", []string{"hello", "foo"}, true},
		{"matches second", "hello world", []string{"foo", "world"}, true},
		{"no match", "hello world", []string{"foo", "bar"}, false},
		{"empty string", "", []string{"foo"}, false},
		{"empty parts", "hello", []string{}, false},
		{"exact match", "hello", []string{"hello"}, true},
		{"substring", "hello world", []string{"lo wo"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsAny(tt.s, tt.parts...)
			if got != tt.want {
				t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.parts, got, tt.want)
			}
		})
	}
}

func TestScopeConstants(t *testing.T) {
	if ScopeLLM != "llm" {
		t.Errorf("ScopeLLM = %q, want %q", ScopeLLM, "llm")
	}
	if ScopeRuntime != "runtime" {
		t.Errorf("ScopeRuntime = %q, want %q", ScopeRuntime, "runtime")
	}
}

func TestRequest_CanBeConstructed(t *testing.T) {
	req := Request{
		Scope:           ScopeLLM,
		Operation:       "plan",
		StoryID:         "STR-001",
		StoryTitle:      "Test Story",
		CurrentProvider: "anthropic",
		CurrentRuntime:  "claude-code",
		TargetProvider:  "openai",
		TargetRuntime:   "codex",
		TargetModel:     "gpt-4o",
		Reason:          "Claude exhausted",
		Note:            "Temporary switch",
	}

	if req.Scope != ScopeLLM {
		t.Errorf("Scope = %q, want %q", req.Scope, ScopeLLM)
	}
	if req.StoryID != "STR-001" {
		t.Errorf("StoryID = %q, want %q", req.StoryID, "STR-001")
	}
}
