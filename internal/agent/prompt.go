package agent

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// TechStack holds detected project technology details injected into prompts.
type TechStack struct {
	Language    string
	Framework   string
	TestRunner  string
	BuildTool   string
	Linter      string
	PackageJSON bool
}

// String returns a human-readable summary of the tech stack.
func (ts TechStack) String() string {
	var parts []string
	if ts.Language != "" {
		parts = append(parts, "Language: "+ts.Language)
	}
	if ts.Framework != "" {
		parts = append(parts, "Framework: "+ts.Framework)
	}
	if ts.TestRunner != "" {
		parts = append(parts, "Test Runner: "+ts.TestRunner)
	}
	if ts.BuildTool != "" {
		parts = append(parts, "Build Tool: "+ts.BuildTool)
	}
	if ts.Linter != "" {
		parts = append(parts, "Linter: "+ts.Linter)
	}
	return strings.Join(parts, ", ")
}

// StoryContext holds all context needed to generate a story-specific prompt.
type StoryContext struct {
	StoryID            string
	Title              string
	Description        string
	AcceptanceCriteria []string
	OwnedFiles         []string
	Complexity         int
	TechStack          *TechStack
}

// systemPromptTemplate is the template for generating role-based system prompts.
var systemPromptTemplate = template.Must(template.New("system").Parse(`You are a {{.Name}} in an AI agent orchestration system.

{{.Description}}

Your capabilities:
{{- if .Capabilities.CanPlan}}
- You can plan and decompose requirements into stories
{{- end}}
{{- if .Capabilities.CanReview}}
- You can review code for quality, correctness, and standards
{{- end}}
{{- if .Capabilities.CanImplement}}
- You can implement code changes and create new files
{{- end}}
{{- if .Capabilities.CanTest}}
- You can write and run tests
{{- end}}
{{- if .Capabilities.CanAudit}}
- You can audit code for security and compliance
{{- end}}
{{- if .Capabilities.CanMerge}}
- You can approve and merge pull requests
{{- end}}

Constraints:
- Maximum story complexity you can handle: {{.MaxComplexity}}
- Follow established patterns in the codebase
- Write tests for all changes
- Keep changes focused and minimal
`))

// storyPromptTemplate is the template for generating story-specific prompts.
var storyPromptTemplate = template.Must(template.New("story").Parse(`## Story: {{.Story.StoryID}}

**Title:** {{.Story.Title}}

{{- if .Story.Description}}

**Description:** {{.Story.Description}}
{{- end}}

{{- if .Story.AcceptanceCriteria}}

**Acceptance Criteria:**
{{- range .Story.AcceptanceCriteria}}
- [ ] {{.}}
{{- end}}
{{- end}}

{{- if .Story.OwnedFiles}}

**Files to modify:**
{{- range .Story.OwnedFiles}}
- {{.}}
{{- end}}
{{- end}}

**Complexity:** {{.Story.Complexity}}/8

{{- if .Story.TechStack}}

**Tech Stack:** {{.Story.TechStack}}
{{- end}}

As a {{.RoleName}}, implement this story following the project's established patterns.
`))

// GenerateSystemPrompt creates a role-appropriate system prompt for an agent.
func GenerateSystemPrompt(role Role) (string, error) {
	def, ok := GetRoleDefinition(role)
	if !ok {
		return "", fmt.Errorf("unknown role: %q", role)
	}

	var buf bytes.Buffer
	if err := systemPromptTemplate.Execute(&buf, def); err != nil {
		return "", fmt.Errorf("executing system prompt template: %w", err)
	}

	return buf.String(), nil
}

// storyPromptData bundles story context with role info for the template.
type storyPromptData struct {
	Story    StoryContext
	RoleName string
}

// GenerateStoryPrompt creates a story-specific prompt with role context,
// acceptance criteria, tech stack info, and file ownership.
func GenerateStoryPrompt(role Role, ctx StoryContext) (string, error) {
	def, ok := GetRoleDefinition(role)
	if !ok {
		return "", fmt.Errorf("unknown role: %q", role)
	}

	data := storyPromptData{
		Story:    ctx,
		RoleName: def.Name,
	}

	var buf bytes.Buffer
	if err := storyPromptTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing story prompt template: %w", err)
	}

	return buf.String(), nil
}
