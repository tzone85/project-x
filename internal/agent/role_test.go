package agent

import (
	"testing"
)

func TestAllRolesDefined(t *testing.T) {
	expected := []Role{
		RoleTechLead,
		RoleSenior,
		RoleIntermediate,
		RoleJunior,
		RoleQA,
		RoleFeatureTest,
		RoleAuditor,
	}

	for _, role := range expected {
		def, ok := GetRoleDefinition(role)
		if !ok {
			t.Errorf("role %s not found in registry", role)
			continue
		}
		if def.Name == "" {
			t.Errorf("role %s has empty name", role)
		}
		if def.Description == "" {
			t.Errorf("role %s has empty description", role)
		}
	}
}

func TestAllRolesReturnsSevenRoles(t *testing.T) {
	roles := AllRoles()
	if len(roles) != 7 {
		t.Errorf("expected 7 roles, got %d", len(roles))
	}
}

func TestRoleStringRepresentation(t *testing.T) {
	tests := []struct {
		role Role
		want string
	}{
		{RoleTechLead, "tech_lead"},
		{RoleSenior, "senior"},
		{RoleIntermediate, "intermediate"},
		{RoleJunior, "junior"},
		{RoleQA, "qa"},
		{RoleFeatureTest, "feature_test"},
		{RoleAuditor, "auditor"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := string(tt.role); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRoleValid(t *testing.T) {
	tests := []struct {
		input string
		want  Role
	}{
		{"tech_lead", RoleTechLead},
		{"senior", RoleSenior},
		{"intermediate", RoleIntermediate},
		{"junior", RoleJunior},
		{"qa", RoleQA},
		{"feature_test", RoleFeatureTest},
		{"auditor", RoleAuditor},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseRole(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRoleInvalid(t *testing.T) {
	_, err := ParseRole("invalid_role")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestRoleCapabilities(t *testing.T) {
	tests := []struct {
		role           Role
		canPlan        bool
		canReview      bool
		canImplement   bool
		canTest        bool
		canAudit       bool
		canMerge       bool
		maxComplexity  int
	}{
		{RoleTechLead, true, true, true, true, true, true, 8},
		{RoleSenior, true, true, true, true, false, true, 8},
		{RoleIntermediate, false, false, true, true, false, false, 5},
		{RoleJunior, false, false, true, true, false, false, 3},
		{RoleQA, false, false, false, true, false, false, 8},
		{RoleFeatureTest, false, false, false, true, false, false, 8},
		{RoleAuditor, false, true, false, false, true, false, 8},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			def, ok := GetRoleDefinition(tt.role)
			if !ok {
				t.Fatalf("role %s not found", tt.role)
			}

			if def.Capabilities.CanPlan != tt.canPlan {
				t.Errorf("CanPlan: got %v, want %v", def.Capabilities.CanPlan, tt.canPlan)
			}
			if def.Capabilities.CanReview != tt.canReview {
				t.Errorf("CanReview: got %v, want %v", def.Capabilities.CanReview, tt.canReview)
			}
			if def.Capabilities.CanImplement != tt.canImplement {
				t.Errorf("CanImplement: got %v, want %v", def.Capabilities.CanImplement, tt.canImplement)
			}
			if def.Capabilities.CanTest != tt.canTest {
				t.Errorf("CanTest: got %v, want %v", def.Capabilities.CanTest, tt.canTest)
			}
			if def.Capabilities.CanAudit != tt.canAudit {
				t.Errorf("CanAudit: got %v, want %v", def.Capabilities.CanAudit, tt.canAudit)
			}
			if def.Capabilities.CanMerge != tt.canMerge {
				t.Errorf("CanMerge: got %v, want %v", def.Capabilities.CanMerge, tt.canMerge)
			}
			if def.MaxComplexity != tt.maxComplexity {
				t.Errorf("MaxComplexity: got %d, want %d", def.MaxComplexity, tt.maxComplexity)
			}
		})
	}
}

func TestGetRoleDefinitionUnknown(t *testing.T) {
	_, ok := GetRoleDefinition(Role("nonexistent"))
	if ok {
		t.Fatal("expected ok=false for unknown role")
	}
}
