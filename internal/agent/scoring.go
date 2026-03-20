package agent

import "fmt"

// maxAllowedComplexity is the upper bound for story complexity.
const maxAllowedComplexity = 8

// implementationRoles are roles that can be assigned implementation stories,
// ordered from least to most capable.
var implementationRoles = []Role{
	RoleJunior,
	RoleIntermediate,
	RoleSenior,
	RoleTechLead,
}

// MatchRolesForComplexity returns all implementation roles capable of handling
// a story of the given complexity. Roles are ordered from least to most capable.
func MatchRolesForComplexity(complexity int) ([]Role, error) {
	if complexity < 1 || complexity > maxAllowedComplexity {
		return nil, fmt.Errorf("complexity must be between 1 and %d, got %d", maxAllowedComplexity, complexity)
	}

	var matched []Role
	for _, role := range implementationRoles {
		def, ok := GetRoleDefinition(role)
		if !ok {
			continue
		}
		if complexity <= def.MaxComplexity {
			matched = append(matched, role)
		}
	}

	return matched, nil
}

// BestRoleForComplexity returns the least-capable role that can handle the
// given complexity. This optimizes cost by using the simplest agent possible.
func BestRoleForComplexity(complexity int) (Role, error) {
	roles, err := MatchRolesForComplexity(complexity)
	if err != nil {
		return "", err
	}
	if len(roles) == 0 {
		return "", fmt.Errorf("no role can handle complexity %d", complexity)
	}
	return roles[0], nil
}

// PerformanceRecord tracks an agent's performance metrics.
type PerformanceRecord struct {
	AgentID          string
	StoriesCompleted int
	StoriesFailed    int
	TotalRetries     int
}

// NewPerformanceRecord creates a fresh performance record for an agent.
func NewPerformanceRecord(agentID string) PerformanceRecord {
	return PerformanceRecord{AgentID: agentID}
}

// RecordCompletion returns a new record with an incremented completion count.
func (p PerformanceRecord) RecordCompletion() PerformanceRecord {
	return PerformanceRecord{
		AgentID:          p.AgentID,
		StoriesCompleted: p.StoriesCompleted + 1,
		StoriesFailed:    p.StoriesFailed,
		TotalRetries:     p.TotalRetries,
	}
}

// RecordFailure returns a new record with an incremented failure count.
func (p PerformanceRecord) RecordFailure() PerformanceRecord {
	return PerformanceRecord{
		AgentID:          p.AgentID,
		StoriesCompleted: p.StoriesCompleted,
		StoriesFailed:    p.StoriesFailed + 1,
		TotalRetries:     p.TotalRetries,
	}
}

// RecordRetry returns a new record with an incremented retry count.
func (p PerformanceRecord) RecordRetry() PerformanceRecord {
	return PerformanceRecord{
		AgentID:          p.AgentID,
		StoriesCompleted: p.StoriesCompleted,
		StoriesFailed:    p.StoriesFailed,
		TotalRetries:     p.TotalRetries + 1,
	}
}

// SuccessRate returns the ratio of completed stories to total stories attempted.
// Returns 0 if no stories have been attempted.
func (p PerformanceRecord) SuccessRate() float64 {
	total := p.StoriesCompleted + p.StoriesFailed
	if total == 0 {
		return 0.0
	}
	return float64(p.StoriesCompleted) / float64(total)
}
