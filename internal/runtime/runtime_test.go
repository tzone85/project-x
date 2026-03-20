package runtime

import "testing"

func TestAgentStatus_String(t *testing.T) {
	tests := []struct {
		status AgentStatus
		want   string
	}{
		{StatusWorking, "working"},
		{StatusDone, "done"},
		{StatusTerminated, "terminated"},
		{StatusPermissionPrompt, "permission_prompt"},
		{StatusPlanMode, "plan_mode"},
		{StatusStuck, "stuck"},
		{StatusIdle, "idle"},
		{AgentStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.want {
				t.Errorf("AgentStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "hello", "'hello'"},
		{"with spaces", "hello world", "'hello world'"},
		{"with single quote", "it's", "'it'\\''s'"},
		{"empty", "", "''"},
		{"with special chars", "a && b | c", "'a && b | c'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellQuote(tt.input)
			if got != tt.want {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
