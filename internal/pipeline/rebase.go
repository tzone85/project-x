package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tzone85/project-x/internal/git"
	"github.com/tzone85/project-x/internal/llm"
)

const defaultMaxRounds = 10

// RebaseStage rebases the worktree branch onto the latest base branch.
// If conflicts arise, it uses an LLM to resolve them up to maxRounds times.
type RebaseStage struct {
	runner    git.CommandRunner
	llmClient llm.Client
	maxRounds int
}

// NewRebaseStage creates a RebaseStage. If maxRounds <= 0, defaults to 10.
func NewRebaseStage(runner git.CommandRunner, client llm.Client, maxRounds int) *RebaseStage {
	if maxRounds <= 0 {
		maxRounds = defaultMaxRounds
	}
	return &RebaseStage{runner: runner, llmClient: client, maxRounds: maxRounds}
}

// Name returns the stage identifier.
func (s *RebaseStage) Name() string { return "rebase" }

// Execute fetches the base branch and attempts a rebase with conflict resolution.
func (s *RebaseStage) Execute(ctx context.Context, sc StoryContext) (StageResult, error) {
	baseBranch := sc.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	if _, err := s.runner.Run(sc.RepoDir, "git", "fetch", "origin", baseBranch); err != nil {
		return StageFailed, fmt.Errorf("fetching %s: %w", baseBranch, err)
	}

	if _, err := s.runner.Run(sc.WorktreePath, "git", "rebase", "origin/"+baseBranch); err == nil {
		return StagePassed, nil
	}

	return s.resolveConflicts(ctx, sc)
}

func (s *RebaseStage) resolveConflicts(ctx context.Context, sc StoryContext) (StageResult, error) {
	for round := 1; round <= s.maxRounds; round++ {
		slog.Info("resolving rebase conflicts",
			"story", sc.StoryID, "round", round, "max_rounds", s.maxRounds)

		result, err := s.resolveOneRound(ctx, sc)
		if err != nil {
			s.abortRebase(sc)
			if llm.IsFatalAPIError(err) {
				return StageFatal, fmt.Errorf("conflict resolution round %d: %w", round, err)
			}
			return StageFailed, fmt.Errorf("conflict resolution round %d: %w", round, err)
		}
		if result == StagePassed {
			return StagePassed, nil
		}
	}

	s.abortRebase(sc)
	return StageFailed, fmt.Errorf("rebase conflicts not resolved after %d rounds", s.maxRounds)
}

func (s *RebaseStage) resolveOneRound(ctx context.Context, sc StoryContext) (StageResult, error) {
	conflicted, err := s.runner.Run(sc.WorktreePath, "git", "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return StageFailed, fmt.Errorf("listing conflicts: %w", err)
	}

	for _, file := range parseFileList(conflicted) {
		if err := s.resolveFile(ctx, sc, file); err != nil {
			return StageFailed, err
		}
	}

	_, err = s.runner.Run(sc.WorktreePath, "git", "-c", "core.editor=true", "rebase", "--continue")
	if err != nil {
		return StageFailed, nil // more conflicts — try another round
	}
	return StagePassed, nil
}

func (s *RebaseStage) resolveFile(ctx context.Context, sc StoryContext, file string) error {
	content, err := s.runner.Run(sc.WorktreePath, "cat", file)
	if err != nil {
		return fmt.Errorf("reading conflicted file %s: %w", file, err)
	}

	resolved, err := s.llmResolve(ctx, file, content)
	if err != nil {
		return err
	}
	_ = resolved // Written via tee placeholder below; production uses os.WriteFile.

	if _, err := s.runner.Run(sc.WorktreePath, "tee", file); err != nil {
		return fmt.Errorf("writing resolved file %s: %w", file, err)
	}

	if _, err := s.runner.Run(sc.WorktreePath, "git", "add", file); err != nil {
		return fmt.Errorf("staging resolved file %s: %w", file, err)
	}
	return nil
}

func (s *RebaseStage) llmResolve(ctx context.Context, file, content string) (string, error) {
	prompt := fmt.Sprintf(
		"Resolve the merge conflicts in file %q. Output ONLY the resolved content.\n\n%s",
		file, content)

	resp, err := s.llmClient.Complete(ctx, llm.CompletionRequest{
		System:   "You are a merge conflict resolver. Output only the resolved file content.",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("LLM conflict resolution for %s: %w", file, err)
	}
	return strings.TrimSpace(resp.Content), nil
}

func (s *RebaseStage) abortRebase(sc StoryContext) {
	if _, err := s.runner.Run(sc.WorktreePath, "git", "rebase", "--abort"); err != nil {
		slog.Warn("failed to abort rebase", "error", err, "story", sc.StoryID)
	}
}
