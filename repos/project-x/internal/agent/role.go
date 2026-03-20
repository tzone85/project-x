// Package agent defines agent roles, prompt generation, scoring,
// and state management for the agent subsystem.
package agent

// Role represents an agent's role in the development workflow.
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

// RoleDefinition describes the capabilities and constraints of a role.
type RoleDefinition struct {
	Role          Role
	DisplayName   string
	Description   string
	MaxComplexity int      // maximum story complexity this role can handle
	Models        []string // preferred model identifiers (provider/model)
	Capabilities  []string // what this role can do
	Constraints   []string // limitations or rules for this role
}

// AllRoles returns the ordered list of all defined roles.
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

// DefaultRoleDefinitions returns the built-in role definitions.
func DefaultRoleDefinitions() map[Role]RoleDefinition {
	return map[Role]RoleDefinition{
		RoleTechLead: {
			Role:          RoleTechLead,
			DisplayName:   "Tech Lead",
			Description:   "Leads technical planning, code review, and architecture decisions",
			MaxComplexity: 8,
			Models:        []string{"anthropic/claude-opus-4-20250514"},
			Capabilities:  []string{"planning", "review", "architecture", "escalation_handling"},
			Constraints:   []string{"does not implement stories directly"},
		},
		RoleSenior: {
			Role:          RoleSenior,
			DisplayName:   "Senior Developer",
			Description:   "Implements complex stories, handles escalations from intermediate agents",
			MaxComplexity: 8,
			Models:        []string{"anthropic/claude-sonnet-4-20250514"},
			Capabilities:  []string{"implementation", "debugging", "refactoring", "review"},
			Constraints:   []string{},
		},
		RoleIntermediate: {
			Role:          RoleIntermediate,
			DisplayName:   "Intermediate Developer",
			Description:   "Implements moderate complexity stories",
			MaxComplexity: 5,
			Models:        []string{"anthropic/claude-sonnet-4-20250514"},
			Capabilities:  []string{"implementation", "testing"},
			Constraints:   []string{"escalates complexity > 5 stories"},
		},
		RoleJunior: {
			Role:          RoleJunior,
			DisplayName:   "Junior Developer",
			Description:   "Implements simple stories with clear specifications",
			MaxComplexity: 3,
			Models:        []string{"openai/gpt-4o-mini"},
			Capabilities:  []string{"implementation"},
			Constraints:   []string{"requires detailed acceptance criteria", "escalates ambiguous stories"},
		},
		RoleQA: {
			Role:          RoleQA,
			DisplayName:   "QA Engineer",
			Description:   "Runs quality checks, linting, type-checking, and test suites",
			MaxComplexity: 8,
			Models:        []string{"anthropic/claude-sonnet-4-20250514"},
			Capabilities:  []string{"testing", "linting", "type_checking"},
			Constraints:   []string{"does not modify implementation code"},
		},
		RoleFeatureTest: {
			Role:          RoleFeatureTest,
			DisplayName:   "Feature Tester",
			Description:   "Writes and runs feature-level tests",
			MaxComplexity: 5,
			Models:        []string{"openai/gpt-4o-mini"},
			Capabilities:  []string{"test_writing", "test_execution"},
			Constraints:   []string{"test code only"},
		},
		RoleAuditor: {
			Role:          RoleAuditor,
			DisplayName:   "Code Auditor",
			Description:   "Reviews code for security, performance, and best practices",
			MaxComplexity: 8,
			Models:        []string{"anthropic/claude-opus-4-20250514"},
			Capabilities:  []string{"security_review", "performance_review", "best_practices"},
			Constraints:   []string{"read-only analysis, does not modify code"},
		},
	}
}

// CanHandle reports whether the role can handle a story of the given complexity.
func (rd RoleDefinition) CanHandle(complexity int) bool {
	return complexity <= rd.MaxComplexity
}
