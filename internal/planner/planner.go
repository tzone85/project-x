package planner

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tzone85/project-x/internal/config"
	"github.com/tzone85/project-x/internal/llm"
)

const maxRefinementRounds = 2

// Planner decomposes requirements into validated stories using a two-pass approach.
type Planner struct {
	client    llm.Client
	cfg       config.PlanningConfig
	validator StoryValidator
	logger    *slog.Logger
}

// New creates a Planner with the given LLM client and planning config.
func New(client llm.Client, cfg config.PlanningConfig, logger *slog.Logger) *Planner {
	return &Planner{
		client:    client,
		cfg:       cfg,
		validator: NewStoryValidator(cfg),
		logger:    logger,
	}
}

// Plan executes the two-pass planning process for a requirement.
// Pass 1: Decomposition via LLM.
// Pass 2: Validation via separate LLM call + structural validation.
// If validation fails, critique is fed back for a second attempt (max 2 rounds).
// Simple requirements (<3 stories) skip LLM validation to save cost.
func (p *Planner) Plan(ctx context.Context, req Requirement, techStack TechStack) (PlanResult, error) {
	result := PlanResult{RequirementID: req.ID}

	stories, resp, err := p.decompose(ctx, req, techStack, "")
	if err != nil {
		return result, fmt.Errorf("decomposition failed: %w", err)
	}
	result.addTokens(resp)
	result.Rounds = 1

	// Structural validation always runs
	structResult := p.validator.Validate(stories)

	// Skip LLM validation for simple requirements (<3 stories)
	if len(stories) < 3 {
		p.logger.Info("skipping LLM validation for simple requirement",
			"requirement_id", req.ID, "story_count", len(stories))
		result.Stories = stories
		result.Validation = structResult
		result.QualityWarnings = issueMessages(structResult.Issues, "warning")
		return result, nil
	}

	// Pass 2: LLM validation
	llmValidation, valResp, err := p.validate(ctx, req, stories)
	if err != nil {
		// LLM validation failure is non-fatal: use structural result
		p.logger.Warn("LLM validation failed, using structural validation only",
			"requirement_id", req.ID, "error", err)
		result.Stories = stories
		result.Validation = structResult
		result.QualityWarnings = append(
			issueMessages(structResult.Issues, "warning"),
			"LLM validation was skipped due to error",
		)
		return result, nil
	}
	result.addTokens(valResp)

	// Merge structural + LLM issues
	combined := mergeValidation(structResult, llmValidation)

	if combined.Valid {
		result.Stories = stories
		result.Validation = combined
		result.QualityWarnings = issueMessages(combined.Issues, "warning")
		return result, nil
	}

	// Refinement loop: feed critique back, max 2 total rounds
	for round := 2; round <= maxRefinementRounds; round++ {
		critique := combined.Critique
		if critique == "" {
			critique = formatIssuesAsCritique(combined.Issues)
		}

		p.logger.Info("refining decomposition",
			"requirement_id", req.ID, "round", round, "critique_len", len(critique))

		stories, resp, err = p.decompose(ctx, req, techStack, critique)
		if err != nil {
			result.QualityWarnings = append(result.QualityWarnings,
				fmt.Sprintf("refinement round %d failed: %v", round, err))
			break
		}
		result.addTokens(resp)
		result.Rounds = round

		structResult = p.validator.Validate(stories)

		llmValidation, valResp, err = p.validate(ctx, req, stories)
		if err != nil {
			combined = structResult
		} else {
			result.addTokens(valResp)
			combined = mergeValidation(structResult, llmValidation)
		}

		if combined.Valid {
			break
		}
	}

	result.Stories = stories
	result.Validation = combined
	result.QualityWarnings = issueMessages(combined.Issues, "warning")

	if !combined.Valid {
		result.QualityWarnings = append(result.QualityWarnings,
			"plan proceeded with remaining quality issues after max refinement rounds")
	}

	return result, nil
}

func (p *Planner) decompose(ctx context.Context, req Requirement, ts TechStack, critique string) ([]Story, llm.CompletionResponse, error) {
	var msgs []llm.Message
	if critique == "" {
		msgs = BuildDecompositionMessages(req, ts, p.cfg)
	} else {
		msgs = BuildRefinementMessages(req, ts, p.cfg, critique)
	}

	resp, err := p.client.Complete(ctx, msgs)
	if err != nil {
		return nil, resp, fmt.Errorf("LLM decomposition call: %w", err)
	}

	stories, err := ParseStories(resp.Content)
	if err != nil {
		return nil, resp, fmt.Errorf("parsing decomposition response: %w", err)
	}

	return stories, resp, nil
}

func (p *Planner) validate(ctx context.Context, req Requirement, stories []Story) (ValidationResult, llm.CompletionResponse, error) {
	msgs := BuildValidationMessages(req, stories, p.cfg)

	resp, err := p.client.Complete(ctx, msgs)
	if err != nil {
		return ValidationResult{}, resp, fmt.Errorf("LLM validation call: %w", err)
	}

	result, err := ParseValidation(resp.Content)
	if err != nil {
		return ValidationResult{}, resp, fmt.Errorf("parsing validation response: %w", err)
	}

	return result, resp, nil
}

func (r *PlanResult) addTokens(resp llm.CompletionResponse) {
	r.InputTokens += resp.InputTokens
	r.OutputTokens += resp.OutputTokens
	r.CostUSD += resp.CostUSD
}

func mergeValidation(structural, llmResult ValidationResult) ValidationResult {
	allIssues := append(structural.Issues, llmResult.Issues...)
	hasErrors := !structural.Valid || !llmResult.Valid

	return ValidationResult{
		Valid:    !hasErrors,
		Issues:   allIssues,
		Critique: llmResult.Critique,
	}
}

func issueMessages(issues []ValidationIssue, severity string) []string {
	var msgs []string
	for _, issue := range issues {
		if issue.Severity == severity {
			msgs = append(msgs, issue.Message)
		}
	}
	return msgs
}

func formatIssuesAsCritique(issues []ValidationIssue) string {
	var parts []string
	for _, issue := range issues {
		if issue.Severity == "error" {
			prefix := ""
			if issue.StoryID != "" {
				prefix = fmt.Sprintf("[%s] ", issue.StoryID)
			}
			parts = append(parts, fmt.Sprintf("- %s%s: %s", prefix, issue.Field, issue.Message))
		}
	}
	if len(parts) == 0 {
		return "Validation found issues. Please improve the decomposition."
	}
	return fmt.Sprintf("Issues found:\n%s", joinLines(parts))
}

func joinLines(parts []string) string {
	result := ""
	for _, p := range parts {
		result += p + "\n"
	}
	return result
}
