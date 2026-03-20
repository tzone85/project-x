// Package planner implements the two-pass planning system for requirement
// decomposition into stories with quality validation.
package planner

// TechStack holds detected technology context for a repository.
type TechStack struct {
	Language       string   `json:"language"`
	Framework      string   `json:"framework"`
	TestRunner     string   `json:"test_runner"`
	Linter         string   `json:"linter"`
	BuildTool      string   `json:"build_tool"`
	DirectoryLayout string  `json:"directory_layout"`
	TestPatterns   []string `json:"test_patterns"`
}

// Story represents a single decomposed unit of work from a requirement.
type Story struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	OwnedFiles         []string `json:"owned_files"`
	Complexity         int      `json:"complexity"`
	DependsOn          []string `json:"depends_on"`
}

// Requirement holds the input for planning — the work to be decomposed.
type Requirement struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
}

// ValidationIssue describes a single quality problem found during validation.
type ValidationIssue struct {
	StoryID  string `json:"story_id"`
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error" or "warning"
}

// ValidationResult holds the outcome of a validation pass.
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Issues  []ValidationIssue `json:"issues"`
	Critique string           `json:"critique"`
}

// PlanResult holds the final output of the two-pass planning process.
type PlanResult struct {
	RequirementID   string           `json:"requirement_id"`
	Stories         []Story          `json:"stories"`
	Validation      ValidationResult `json:"validation"`
	Rounds          int              `json:"rounds"`
	QualityWarnings []string         `json:"quality_warnings"`
	InputTokens     int              `json:"input_tokens"`
	OutputTokens    int              `json:"output_tokens"`
	CostUSD         float64          `json:"cost_usd"`
}
