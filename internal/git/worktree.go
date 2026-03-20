package git

import (
	"context"
	"strings"
)

// Worktree provides git worktree management operations.
type Worktree struct {
	ops *Ops
}

// NewWorktree creates a Worktree manager using the given Ops.
func NewWorktree(ops *Ops) *Worktree {
	return &Worktree{ops: ops}
}

// Add creates a new worktree at the given path for the given branch.
func (w *Worktree) Add(ctx context.Context, path, branch string) error {
	_, err := w.ops.git(ctx, "worktree", "add", path, branch)
	return err
}

// AddNew creates a new worktree with a new branch.
func (w *Worktree) AddNew(ctx context.Context, path, newBranch string) error {
	_, err := w.ops.git(ctx, "worktree", "add", "-b", newBranch, path)
	return err
}

// Remove removes a worktree at the given path.
func (w *Worktree) Remove(ctx context.Context, path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}
	_, err := w.ops.git(ctx, args...)
	return err
}

// List returns all worktrees with their info.
func (w *Worktree) List(ctx context.Context) ([]WorktreeInfo, error) {
	out, err := w.ops.git(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return parseWorktreeOutput(out), nil
}

// Prune cleans up stale worktree references.
func (w *Worktree) Prune(ctx context.Context) error {
	_, err := w.ops.git(ctx, "worktree", "prune")
	return err
}

// parseWorktreeOutput parses git worktree list --porcelain output.
// Each worktree block is separated by a blank line.
// Format:
//
//	worktree /path/to/worktree
//	HEAD abc123
//	branch refs/heads/main
func parseWorktreeOutput(out string) []WorktreeInfo {
	blocks := strings.Split(out, "\n\n")
	trees := make([]WorktreeInfo, 0, len(blocks))

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		info := parseWorktreeBlock(block)
		trees = append(trees, info)
	}
	return trees
}

func parseWorktreeBlock(block string) WorktreeInfo {
	var info WorktreeInfo
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			info.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			info.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			info.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			info.Bare = true
		}
	}
	return info
}
