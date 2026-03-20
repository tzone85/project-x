package agent

import (
	"fmt"
	"strings"
)

// TechStack describes the detected technology stack of a project.
type TechStack struct {
	Language   string
	Framework  string
	TestRunner string
	Linter     string
	BuildTool  string
	DirLayout  string
}

// StorySpec contains the information needed to generate a story prompt.
type StorySpec struct {
	ID                 string
	Title              string
	Description        string
	AcceptanceCriteria []string
	OwnedFiles         []string
	Complexity         int
	DependsOn          []string
}

// PromptBuilder generates system and story prompts for agents.
type PromptBuilder struct {
	roles map[Role]RoleDefinition
}

// NewPromptBuilder creates a new prompt builder with the given role definitions.
func NewPromptBuilder(roles map[Role]RoleDefinition) *PromptBuilder {
	return &PromptBuilder{roles: roles}
}

// SystemPrompt generates the system prompt for an agent with the given role.
func (pb *PromptBuilder) SystemPrompt(role Role, tech TechStack) string {
	rd, ok := pb.roles[role]
	if !ok {
		return fmt.Sprintf("You are an AI agent with role: %s.", role)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("You are a %s AI agent.\n\n", rd.DisplayName))
	sb.WriteString(fmt.Sprintf("## Role\n%s\n\n", rd.Description))

	if len(rd.Capabilities) > 0 {
		sb.WriteString("## Capabilities\n")
		for _, cap := range rd.Capabilities {
			sb.WriteString(fmt.Sprintf("- %s\n", cap))
		}
		sb.WriteString("\n")
	}

	if len(rd.Constraints) > 0 {
		sb.WriteString("## Constraints\n")
		for _, c := range rd.Constraints {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
		sb.WriteString("\n")
	}

	if tech.Language != "" {
		sb.WriteString("## Tech Stack\n")
		sb.WriteString(fmt.Sprintf("- Language: %s\n", tech.Language))
		if tech.Framework != "" {
			sb.WriteString(fmt.Sprintf("- Framework: %s\n", tech.Framework))
		}
		if tech.TestRunner != "" {
			sb.WriteString(fmt.Sprintf("- Test Runner: %s\n", tech.TestRunner))
		}
		if tech.Linter != "" {
			sb.WriteString(fmt.Sprintf("- Linter: %s\n", tech.Linter))
		}
		if tech.BuildTool != "" {
			sb.WriteString(fmt.Sprintf("- Build Tool: %s\n", tech.BuildTool))
		}
		if tech.DirLayout != "" {
			sb.WriteString(fmt.Sprintf("- Directory Layout: %s\n", tech.DirLayout))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// StoryPrompt generates the task prompt for a specific story.
func (pb *PromptBuilder) StoryPrompt(spec StorySpec) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Story: %s\n", spec.Title))
	sb.WriteString(fmt.Sprintf("**ID:** %s\n", spec.ID))
	sb.WriteString(fmt.Sprintf("**Complexity:** %d\n\n", spec.Complexity))

	if spec.Description != "" {
		sb.WriteString(fmt.Sprintf("## Description\n%s\n\n", spec.Description))
	}

	if len(spec.AcceptanceCriteria) > 0 {
		sb.WriteString("## Acceptance Criteria\n")
		for _, ac := range spec.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", ac))
		}
		sb.WriteString("\n")
	}

	if len(spec.OwnedFiles) > 0 {
		sb.WriteString("## Owned Files\n")
		for _, f := range spec.OwnedFiles {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	if len(spec.DependsOn) > 0 {
		sb.WriteString(fmt.Sprintf("## Dependencies\nThis story depends on: %s\n\n",
			strings.Join(spec.DependsOn, ", ")))
	}

	return sb.String()
}
