package agent

import (
	"fmt"
	"strings"
)

// PromptContext holds the data needed to generate agent prompts.
type PromptContext struct {
	StoryID            string
	StoryTitle         string
	StoryDescription   string
	AcceptanceCriteria string
	RepoPath           string
	Complexity         int
	ReviewFeedback     string
	TechStack          string
}

// roleSystemDescriptions maps each role to its system prompt instruction text.
var roleSystemDescriptions = map[Role]string{
	RoleTechLead: "You are a tech lead reviewing and decomposing requirements. " +
		"Focus on architectural decisions, code organization, and ensuring " +
		"the implementation plan is sound before any code is written.",

	RoleSenior: "You are a senior developer. Write production-quality code " +
		"with comprehensive error handling, tests, and documentation. " +
		"Consider edge cases, performance, and maintainability.",

	RoleIntermediate: "You are a developer. Implement the feature as specified, " +
		"following established patterns in the codebase. Write clean, " +
		"well-tested code that meets the acceptance criteria.",

	RoleJunior: "You are a junior developer. Follow instructions precisely " +
		"and implement exactly what is specified. Ask for clarification " +
		"if requirements are ambiguous. Write tests for your code.",

	RoleQA: "You are a QA engineer. Run lint, build, and tests to verify " +
		"the implementation meets quality standards. Report any failures " +
		"with clear descriptions of what went wrong and how to fix it.",

	RoleSupervisor: "You are a supervisor reviewing agent work for quality " +
		"and completeness. Verify that acceptance criteria are met, code " +
		"quality standards are upheld, and no regressions are introduced.",
}

// SystemPrompt generates the system prompt for an agent based on its role.
func SystemPrompt(role Role, ctx PromptContext) string {
	description, ok := roleSystemDescriptions[role]
	if !ok {
		description = roleSystemDescriptions[RoleJunior]
	}

	var b strings.Builder
	b.WriteString(description)

	if ctx.TechStack != "" {
		b.WriteString("\n\nTech Stack:\n")
		b.WriteString(ctx.TechStack)
	}

	if ctx.RepoPath != "" {
		b.WriteString("\n\nRepository: ")
		b.WriteString(ctx.RepoPath)
	}

	return b.String()
}

// GoalPrompt generates the goal/task prompt with story details. Review
// feedback is included only when non-empty.
func GoalPrompt(role Role, ctx PromptContext) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("## Story: %s\n", ctx.StoryTitle))
	b.WriteString(fmt.Sprintf("ID: %s\n", ctx.StoryID))
	b.WriteString(fmt.Sprintf("Complexity: %d\n", ctx.Complexity))

	if ctx.StoryDescription != "" {
		b.WriteString(fmt.Sprintf("\n### Description\n%s\n", ctx.StoryDescription))
	}

	if ctx.AcceptanceCriteria != "" {
		b.WriteString(fmt.Sprintf("\n### Acceptance Criteria\n%s\n", ctx.AcceptanceCriteria))
	}

	if ctx.ReviewFeedback != "" {
		b.WriteString(fmt.Sprintf("\n### Review Feedback\n%s\n", ctx.ReviewFeedback))
	}

	return b.String()
}
