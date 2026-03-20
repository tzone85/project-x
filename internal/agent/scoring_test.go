package agent

import (
	"testing"
)

func TestMatchRoleForComplexity(t *testing.T) {
	tests := []struct {
		name       string
		complexity int
		wantRoles  []Role
		wantErr    bool
	}{
		{
			name:       "complexity_1_matches_junior_and_above",
			complexity: 1,
			wantRoles:  []Role{RoleJunior, RoleIntermediate, RoleSenior, RoleTechLead},
		},
		{
			name:       "complexity_3_matches_junior_and_above",
			complexity: 3,
			wantRoles:  []Role{RoleJunior, RoleIntermediate, RoleSenior, RoleTechLead},
		},
		{
			name:       "complexity_4_excludes_junior",
			complexity: 4,
			wantRoles:  []Role{RoleIntermediate, RoleSenior, RoleTechLead},
		},
		{
			name:       "complexity_5_matches_intermediate_and_above",
			complexity: 5,
			wantRoles:  []Role{RoleIntermediate, RoleSenior, RoleTechLead},
		},
		{
			name:       "complexity_6_excludes_intermediate",
			complexity: 6,
			wantRoles:  []Role{RoleSenior, RoleTechLead},
		},
		{
			name:       "complexity_8_only_senior_and_lead",
			complexity: 8,
			wantRoles:  []Role{RoleSenior, RoleTechLead},
		},
		{
			name:       "complexity_0_is_invalid",
			complexity: 0,
			wantErr:    true,
		},
		{
			name:       "negative_complexity_is_invalid",
			complexity: -1,
			wantErr:    true,
		},
		{
			name:       "complexity_9_exceeds_max",
			complexity: 9,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchRolesForComplexity(tt.complexity)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.wantRoles) {
				t.Fatalf("got %d roles, want %d: %v", len(got), len(tt.wantRoles), got)
			}
			for i, role := range got {
				if role != tt.wantRoles[i] {
					t.Errorf("role[%d]: got %q, want %q", i, role, tt.wantRoles[i])
				}
			}
		})
	}
}

func TestBestRoleForComplexity(t *testing.T) {
	tests := []struct {
		name       string
		complexity int
		want       Role
		wantErr    bool
	}{
		{"complexity_1_gets_junior", 1, RoleJunior, false},
		{"complexity_3_gets_junior", 3, RoleJunior, false},
		{"complexity_4_gets_intermediate", 4, RoleIntermediate, false},
		{"complexity_5_gets_intermediate", 5, RoleIntermediate, false},
		{"complexity_6_gets_senior", 6, RoleSenior, false},
		{"complexity_8_gets_senior", 8, RoleSenior, false},
		{"complexity_0_errors", 0, "", true},
		{"complexity_9_errors", 9, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BestRoleForComplexity(tt.complexity)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewPerformanceRecord(t *testing.T) {
	rec := NewPerformanceRecord("agent-1")

	if rec.AgentID != "agent-1" {
		t.Errorf("got AgentID %q, want %q", rec.AgentID, "agent-1")
	}
	if rec.StoriesCompleted != 0 {
		t.Errorf("expected 0 stories completed, got %d", rec.StoriesCompleted)
	}
	if rec.StoriesFailed != 0 {
		t.Errorf("expected 0 stories failed, got %d", rec.StoriesFailed)
	}
	if rec.TotalRetries != 0 {
		t.Errorf("expected 0 retries, got %d", rec.TotalRetries)
	}
}

func TestRecordCompletion(t *testing.T) {
	rec := NewPerformanceRecord("agent-1")
	rec = rec.RecordCompletion()

	if rec.StoriesCompleted != 1 {
		t.Errorf("got %d, want 1", rec.StoriesCompleted)
	}

	rec = rec.RecordCompletion()
	if rec.StoriesCompleted != 2 {
		t.Errorf("got %d, want 2", rec.StoriesCompleted)
	}
}

func TestRecordFailure(t *testing.T) {
	rec := NewPerformanceRecord("agent-1")
	rec = rec.RecordFailure()

	if rec.StoriesFailed != 1 {
		t.Errorf("got %d, want 1", rec.StoriesFailed)
	}
}

func TestRecordRetry(t *testing.T) {
	rec := NewPerformanceRecord("agent-1")
	rec = rec.RecordRetry()

	if rec.TotalRetries != 1 {
		t.Errorf("got %d, want 1", rec.TotalRetries)
	}
}

func TestSuccessRate(t *testing.T) {
	tests := []struct {
		name      string
		completed int
		failed    int
		want      float64
	}{
		{"no_stories", 0, 0, 0.0},
		{"all_success", 5, 0, 1.0},
		{"all_failed", 0, 5, 0.0},
		{"mixed", 3, 1, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := NewPerformanceRecord("agent-1")
			for i := 0; i < tt.completed; i++ {
				rec = rec.RecordCompletion()
			}
			for i := 0; i < tt.failed; i++ {
				rec = rec.RecordFailure()
			}

			got := rec.SuccessRate()
			if got != tt.want {
				t.Errorf("got %f, want %f", got, tt.want)
			}
		})
	}
}

func TestPerformanceRecordImmutability(t *testing.T) {
	original := NewPerformanceRecord("agent-1")
	updated := original.RecordCompletion()

	if original.StoriesCompleted != 0 {
		t.Error("original was mutated")
	}
	if updated.StoriesCompleted != 1 {
		t.Errorf("expected 1, got %d", updated.StoriesCompleted)
	}
}
