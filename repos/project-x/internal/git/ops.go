package git

import (
	"context"
	"fmt"
	"strings"
)

// Ops provides high-level git operations on a repository directory.
type Ops struct {
	dir string
}

// NewOps creates a new Ops for the given repository directory.
func NewOps(dir string) *Ops {
	return &Ops{dir: dir}
}

// Status returns the current git status output.
func (o *Ops) Status(ctx context.Context) (string, error) {
	r, err := RunGit(ctx, o.dir, "status", "--porcelain")
	if err != nil {
		return "", err
	}
	return r.Stdout, nil
}

// Diff returns the diff output. If staged is true, shows staged changes only.
func (o *Ops) Diff(ctx context.Context, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	r, err := RunGit(ctx, o.dir, args...)
	if err != nil {
		return "", err
	}
	return r.Stdout, nil
}

// Log returns recent commit log entries.
func (o *Ops) Log(ctx context.Context, maxCount int) (string, error) {
	r, err := RunGit(ctx, o.dir, "log", "--oneline", fmt.Sprintf("-n%d", maxCount))
	if err != nil {
		return "", err
	}
	return r.Stdout, nil
}

// Commit creates a git commit with the given message. If files is non-empty,
// stages those files first.
func (o *Ops) Commit(ctx context.Context, message string, files []string) error {
	if len(files) > 0 {
		args := append([]string{"add"}, files...)
		if _, err := RunGit(ctx, o.dir, args...); err != nil {
			return fmt.Errorf("staging files: %w", err)
		}
	}
	_, err := RunGit(ctx, o.dir, "commit", "-m", message)
	return err
}

// Push pushes the current branch to the remote.
func (o *Ops) Push(ctx context.Context, remote, branch string) error {
	_, err := RunGit(ctx, o.dir, "push", remote, branch)
	return err
}

// PushWithUpstream pushes and sets the upstream tracking branch.
func (o *Ops) PushWithUpstream(ctx context.Context, remote, branch string) error {
	_, err := RunGit(ctx, o.dir, "push", "-u", remote, branch)
	return err
}

// Pull pulls from the remote.
func (o *Ops) Pull(ctx context.Context, remote, branch string) error {
	_, err := RunGit(ctx, o.dir, "pull", remote, branch)
	return err
}

// Rebase rebases the current branch onto the given target.
func (o *Ops) Rebase(ctx context.Context, target string) error {
	_, err := RunGit(ctx, o.dir, "rebase", target)
	return err
}

// Merge merges the given branch into the current branch.
func (o *Ops) Merge(ctx context.Context, branch string) error {
	_, err := RunGit(ctx, o.dir, "merge", branch)
	return err
}

// CurrentBranch returns the name of the current branch.
func (o *Ops) CurrentBranch(ctx context.Context) (string, error) {
	r, err := RunGit(ctx, o.dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return r.Stdout, nil
}

// CreateBranch creates and checks out a new branch from base.
func (o *Ops) CreateBranch(ctx context.Context, name, base string) error {
	_, err := RunGit(ctx, o.dir, "checkout", "-b", name, base)
	return err
}

// CheckoutBranch checks out an existing branch.
func (o *Ops) CheckoutBranch(ctx context.Context, name string) error {
	_, err := RunGit(ctx, o.dir, "checkout", name)
	return err
}

// DeleteBranch deletes a local branch.
func (o *Ops) DeleteBranch(ctx context.Context, name string) error {
	_, err := RunGit(ctx, o.dir, "branch", "-d", name)
	return err
}

// HasConflicts checks if there are unresolved merge conflicts.
func (o *Ops) HasConflicts(ctx context.Context) (bool, error) {
	r, err := RunGit(ctx, o.dir, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return false, err
	}
	return r.Stdout != "", nil
}

// --- Worktree operations ---

// CreateWorktree creates a new git worktree.
func (o *Ops) CreateWorktree(ctx context.Context, path, branch string) error {
	_, err := RunGit(ctx, o.dir, "worktree", "add", path, branch)
	return err
}

// RemoveWorktree removes a git worktree.
func (o *Ops) RemoveWorktree(ctx context.Context, path string) error {
	_, err := RunGit(ctx, o.dir, "worktree", "remove", path, "--force")
	return err
}

// ListWorktrees returns the list of worktree paths.
func (o *Ops) ListWorktrees(ctx context.Context) ([]string, error) {
	r, err := RunGit(ctx, o.dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, line := range strings.Split(r.Stdout, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			paths = append(paths, strings.TrimPrefix(line, "worktree "))
		}
	}
	return paths, nil
}

// --- GitHub CLI operations ---

// CreatePR creates a GitHub pull request and returns the PR URL.
func (o *Ops) CreatePR(ctx context.Context, title, body, base string) (string, error) {
	r, err := RunGH(ctx, o.dir, "pr", "create", "--title", title, "--body", body, "--base", base)
	if err != nil {
		return "", err
	}
	return r.Stdout, nil
}

// MergePR merges a GitHub pull request.
func (o *Ops) MergePR(ctx context.Context, prNumber string, method string) error {
	args := []string{"pr", "merge", prNumber, "--" + method}
	_, err := RunGH(ctx, o.dir, args...)
	return err
}

// ListPRs lists open pull requests.
func (o *Ops) ListPRs(ctx context.Context) (string, error) {
	r, err := RunGH(ctx, o.dir, "pr", "list")
	if err != nil {
		return "", err
	}
	return r.Stdout, nil
}
