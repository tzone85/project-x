package planner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tzone85/project-x/internal/config"
	"github.com/tzone85/project-x/internal/llm"
)

// BuildDecompositionMessages creates the LLM messages for Pass 1 (decomposition).
func BuildDecompositionMessages(req Requirement, techStack TechStack, cfg config.PlanningConfig) []llm.Message {
	system := buildDecompositionSystem(techStack, cfg)
	user := buildDecompositionUser(req)

	return []llm.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}
}

// BuildValidationMessages creates the LLM messages for Pass 2 (validation).
func BuildValidationMessages(req Requirement, stories []Story, cfg config.PlanningConfig) []llm.Message {
	system := buildValidationSystem(cfg)
	user := buildValidationUser(req, stories)

	return []llm.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}
}

// BuildRefinementMessages creates messages for re-decomposition after critique.
func BuildRefinementMessages(req Requirement, techStack TechStack, cfg config.PlanningConfig, critique string) []llm.Message {
	system := buildDecompositionSystem(techStack, cfg)
	user := buildDecompositionUser(req)
	feedback := fmt.Sprintf(
		"The previous decomposition was reviewed and found issues:\n\n%s\n\n"+
			"Please produce an improved decomposition that addresses these concerns. "+
			"Respond with the same JSON array format.",
		critique,
	)

	return []llm.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
		{Role: "assistant", Content: "I'll decompose this requirement into stories."},
		{Role: "user", Content: feedback},
	}
}

func buildDecompositionSystem(ts TechStack, cfg config.PlanningConfig) string {
	var b strings.Builder

	b.WriteString("You are a Tech Lead decomposing a software requirement into implementation stories.\n\n")
	b.WriteString("## Constraints\n")
	fmt.Fprintf(&b, "- Maximum story complexity: %d (1-8 scale)\n", cfg.MaxStoryComplexity)
	fmt.Fprintf(&b, "- Maximum stories per requirement: %d\n", cfg.MaxStoriesPerRequirement)
	b.WriteString("- Each story must have: title, description, acceptance_criteria, owned_files, complexity, depends_on\n")

	if cfg.EnforceFileOwnership {
		b.WriteString("- No two stories may own the same file\n")
	}

	b.WriteString("- depends_on is an array of story IDs this story depends on (empty if none)\n")
	b.WriteString("- Story IDs should be sequential: STR-1, STR-2, etc.\n\n")

	if ts.Language != "" {
		b.WriteString("## Tech Stack\n")
		writeIfSet(&b, "Language", ts.Language)
		writeIfSet(&b, "Framework", ts.Framework)
		writeIfSet(&b, "Test Runner", ts.TestRunner)
		writeIfSet(&b, "Linter", ts.Linter)
		writeIfSet(&b, "Build Tool", ts.BuildTool)
		writeIfSet(&b, "Directory Layout", ts.DirectoryLayout)
		if len(ts.TestPatterns) > 0 {
			fmt.Fprintf(&b, "- Test Patterns: %s\n", strings.Join(ts.TestPatterns, ", "))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Output Format\n")
	b.WriteString("Respond with ONLY a JSON array of story objects. No markdown, no explanation.\n")
	b.WriteString("Example:\n")
	b.WriteString(`[{"id":"STR-1","title":"...","description":"...","acceptance_criteria":["..."],"owned_files":["..."],"complexity":3,"depends_on":[]}]`)
	b.WriteString("\n")

	return b.String()
}

func buildDecompositionUser(req Requirement) string {
	return fmt.Sprintf(
		"Decompose this requirement into implementation stories:\n\n"+
			"**Requirement ID:** %s\n"+
			"**Title:** %s\n\n"+
			"%s",
		req.ID, req.Title, req.Description,
	)
}

func buildValidationSystem(cfg config.PlanningConfig) string {
	var b strings.Builder

	b.WriteString("You are a QA reviewer validating a story decomposition plan.\n\n")
	b.WriteString("## Quality Criteria\n")
	fmt.Fprintf(&b, "- Required fields: %s\n", strings.Join(cfg.RequiredFields, ", "))
	fmt.Fprintf(&b, "- Max complexity per story: %d\n", cfg.MaxStoryComplexity)
	fmt.Fprintf(&b, "- Max stories: %d\n", cfg.MaxStoriesPerRequirement)
	if cfg.EnforceFileOwnership {
		b.WriteString("- No file owned by multiple stories\n")
	}
	b.WriteString("- Dependencies must reference valid story IDs\n")
	b.WriteString("- No circular dependencies\n")
	b.WriteString("- Acceptance criteria must be specific and testable\n\n")

	b.WriteString("## Output Format\n")
	b.WriteString("Respond with ONLY a JSON object:\n")
	b.WriteString(`{"valid":true/false,"issues":[{"story_id":"...","field":"...","message":"...","severity":"error|warning"}],"critique":"summary of issues if any"}`)
	b.WriteString("\n")

	return b.String()
}

func buildValidationUser(req Requirement, stories []Story) string {
	storiesJSON, _ := json.MarshalIndent(stories, "", "  ")
	return fmt.Sprintf(
		"Validate this decomposition plan for requirement %q:\n\n%s",
		req.ID, string(storiesJSON),
	)
}

func writeIfSet(b *strings.Builder, label, value string) {
	if value != "" {
		fmt.Fprintf(b, "- %s: %s\n", label, value)
	}
}
