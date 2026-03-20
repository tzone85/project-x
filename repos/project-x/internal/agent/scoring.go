package agent

// RoleForComplexity returns the best-fit role for a story of the given complexity.
// Prefers less expensive roles (junior → intermediate → senior) that can handle
// the complexity level.
func RoleForComplexity(complexity int, roles map[Role]RoleDefinition) Role {
	// Priority order: least capable (cheapest) that can handle the complexity
	priority := []Role{
		RoleJunior,
		RoleIntermediate,
		RoleSenior,
	}

	for _, role := range priority {
		rd, ok := roles[role]
		if ok && rd.CanHandle(complexity) {
			return role
		}
	}

	// Fallback to senior for anything above all thresholds
	return RoleSenior
}
