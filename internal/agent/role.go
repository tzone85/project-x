// Package agent defines agent roles, prompt generation, scoring, and state management
// for the px orchestration system.
package agent

import "fmt"

// Role represents an agent's role in the development pipeline.
type Role string

const (
	RoleTechLead     Role = "tech_lead"
	RoleSenior       Role = "senior"
	RoleIntermediate Role = "intermediate"
	RoleJunior       Role = "junior"
	RoleQA           Role = "qa"
	RoleFeatureTest  Role = "feature_test"
	RoleAuditor      Role = "auditor"
)

// Capabilities defines what actions an agent role can perform.
type Capabilities struct {
	CanPlan      bool
	CanReview    bool
	CanImplement bool
	CanTest      bool
	CanAudit     bool
	CanMerge     bool
}

// RoleDefinition describes a role's properties, capabilities, and constraints.
type RoleDefinition struct {
	Name          string
	Description   string
	Role          Role
	Capabilities  Capabilities
	MaxComplexity int
}

// roleRegistry maps each role to its definition.
var roleRegistry = map[Role]RoleDefinition{
	RoleTechLead: {
		Name:        "Tech Lead",
		Description: "Leads technical decisions, reviews architecture, and approves merges",
		Role:        RoleTechLead,
		Capabilities: Capabilities{
			CanPlan:      true,
			CanReview:    true,
			CanImplement: true,
			CanTest:      true,
			CanAudit:     true,
			CanMerge:     true,
		},
		MaxComplexity: 8,
	},
	RoleSenior: {
		Name:        "Senior Developer",
		Description: "Handles complex implementation, reviews code, and mentors junior agents",
		Role:        RoleSenior,
		Capabilities: Capabilities{
			CanPlan:      true,
			CanReview:    true,
			CanImplement: true,
			CanTest:      true,
			CanAudit:     false,
			CanMerge:     true,
		},
		MaxComplexity: 8,
	},
	RoleIntermediate: {
		Name:        "Intermediate Developer",
		Description: "Implements moderate-complexity stories with standard patterns",
		Role:        RoleIntermediate,
		Capabilities: Capabilities{
			CanPlan:      false,
			CanReview:    false,
			CanImplement: true,
			CanTest:      true,
			CanAudit:     false,
			CanMerge:     false,
		},
		MaxComplexity: 5,
	},
	RoleJunior: {
		Name:        "Junior Developer",
		Description: "Implements simple, well-defined stories following established patterns",
		Role:        RoleJunior,
		Capabilities: Capabilities{
			CanPlan:      false,
			CanReview:    false,
			CanImplement: true,
			CanTest:      true,
			CanAudit:     false,
			CanMerge:     false,
		},
		MaxComplexity: 3,
	},
	RoleQA: {
		Name:        "QA Engineer",
		Description: "Runs test suites, validates acceptance criteria, and reports defects",
		Role:        RoleQA,
		Capabilities: Capabilities{
			CanPlan:      false,
			CanReview:    false,
			CanImplement: false,
			CanTest:      true,
			CanAudit:     false,
			CanMerge:     false,
		},
		MaxComplexity: 8,
	},
	RoleFeatureTest: {
		Name:        "Feature Tester",
		Description: "Tests specific feature branches against acceptance criteria",
		Role:        RoleFeatureTest,
		Capabilities: Capabilities{
			CanPlan:      false,
			CanReview:    false,
			CanImplement: false,
			CanTest:      true,
			CanAudit:     false,
			CanMerge:     false,
		},
		MaxComplexity: 8,
	},
	RoleAuditor: {
		Name:        "Code Auditor",
		Description: "Reviews code for security, compliance, and quality standards",
		Role:        RoleAuditor,
		Capabilities: Capabilities{
			CanPlan:      false,
			CanReview:    true,
			CanImplement: false,
			CanTest:      false,
			CanAudit:     true,
			CanMerge:     false,
		},
		MaxComplexity: 8,
	},
}

// GetRoleDefinition returns the definition for a given role.
// Returns false if the role is not registered.
func GetRoleDefinition(r Role) (RoleDefinition, bool) {
	def, ok := roleRegistry[r]
	return def, ok
}

// AllRoles returns all registered roles in a stable order.
func AllRoles() []Role {
	return []Role{
		RoleTechLead,
		RoleSenior,
		RoleIntermediate,
		RoleJunior,
		RoleQA,
		RoleFeatureTest,
		RoleAuditor,
	}
}

// ParseRole converts a string to a Role, returning an error if the string
// does not match a known role.
func ParseRole(s string) (Role, error) {
	r := Role(s)
	if _, ok := roleRegistry[r]; !ok {
		return "", fmt.Errorf("unknown role: %q", s)
	}
	return r, nil
}
