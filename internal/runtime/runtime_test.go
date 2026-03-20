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

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := tc.status.String()
			if got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestCostTier_Values(t *testing.T) {
	if CostTierSubscription >= CostTierAPI {
		t.Error("subscription tier should sort before API tier")
	}
}
