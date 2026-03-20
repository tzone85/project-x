package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tzone85/project-x/internal/git"
	"github.com/tzone85/project-x/internal/llm"
)

// reviewResponse is the expected JSON structure from the LLM review.
type reviewResponse struct {
	Passed   bool     `json:"passed"`
	Summary  string   `json:"summary"`
	Comments []string `json:"comments"`
}

// ReviewStage sends the story diff to an LLM for code review. The LLM
// returns a structured verdict indicating whether the changes pass review.
type ReviewStage struct {
	runner    git.CommandRunner
	llmClient llm.Client
}

// NewReviewStage creates a ReviewStage with the given runner and LLM client.
func NewReviewStage(runner git.CommandRunner, client llm.Client) *ReviewStage {
	return &ReviewStage{runner: runner, llmClient: client}
}

// Name returns the stage identifier.
func (s *ReviewStage) Name() string { return "review" }

// Execute gets the diff and file tree, sends them to the LLM for review,
// and parses the structured response.
func (s *ReviewStage) Execute(ctx context.Context, sc StoryContext) (StageResult, error) {
	diff, err := s.getDiff(sc)
	if err != nil {
		return StageFailed, fmt.Errorf("getting diff for review: %w", err)
	}

	fileTree, err := s.runner.Run(sc.WorktreePath, "git", "ls-files")
	if err != nil {
		return StageFailed, fmt.Errorf("listing files: %w", err)
	}

	prompt := buildReviewPrompt(sc.StoryID, diff, fileTree)
	resp, err := s.llmClient.Complete(ctx, llm.CompletionRequest{
		System: "You are a code reviewer. Respond with JSON: {\"passed\": bool, \"summary\": string, \"comments\": [string]}",
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: prompt},
		},
	})
	if err != nil {
		if llm.IsFatalAPIError(err) {
			return StageFatal, fmt.Errorf("LLM review: %w", err)
		}
		return StageFailed, fmt.Errorf("LLM review: %w", err)
	}

	var review reviewResponse
	if err := json.Unmarshal([]byte(resp.Content), &review); err != nil {
		return StageFailed, fmt.Errorf("parsing review response: %w", err)
	}

	if !review.Passed {
		return StageFailed, fmt.Errorf("review failed: %s", review.Summary)
	}

	return StagePassed, nil
}

// getDiff retrieves the full diff against the base branch.
func (s *ReviewStage) getDiff(sc StoryContext) (string, error) {
	baseBranch := sc.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	mergeBase, err := s.runner.Run(sc.WorktreePath, "git", "merge-base", "HEAD", "origin/"+baseBranch)
	if err != nil {
		return "", fmt.Errorf("finding merge-base: %w", err)
	}

	diff, err := s.runner.Run(sc.WorktreePath, "git", "diff", mergeBase)
	if err != nil {
		return "", fmt.Errorf("running diff: %w", err)
	}

	return diff, nil
}

// buildReviewPrompt constructs the LLM prompt for code review.
func buildReviewPrompt(storyID, diff, fileTree string) string {
	return fmt.Sprintf(
		"Review the following changes for story %s.\n\n"+
			"## File Tree\n```\n%s\n```\n\n"+
			"## Diff\n```diff\n%s\n```\n\n"+
			"Respond with JSON: {\"passed\": bool, \"summary\": string, \"comments\": [string]}",
		storyID, fileTree, diff,
	)
}
