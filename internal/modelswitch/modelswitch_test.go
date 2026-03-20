package modelswitch

import "testing"

func TestDetectClaudeExhaustion_ExtraUsageLimit(t *testing.T) {
	reason, ok := DetectClaudeExhaustion(`{"result":"You're out of extra usage · resets 7pm (Africa/Johannesburg)"}`)
	if !ok {
		t.Fatal("expected extra usage exhaustion to be detected")
	}
	if reason != "Claude usage limits have been reached" {
		t.Fatalf("unexpected reason: %s", reason)
	}
}
