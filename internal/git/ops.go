package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Ops provides core git operations for a repository.
type Ops struct {
	runner CommandRunner
	dir    string
}

// NewOps creates a new Ops instance for the given repository directory.
func NewOps(runner CommandRunner, dir string) *Ops {
	return &Ops{runner: runner, dir: dir}
}

// git runs a git subcommand in the repo directory.
func (o *Ops) git(ctx context.Context, args ...string) (string, error) {
	return o.runner.Run(ctx, o.dir, "git", args...)
}

// Status returns the working tree status as parsed entries.
func (o *Ops) Status(ctx context.Context) ([]StatusEntry, error) {
	out, err := o.git(ctx, "status", "--porcelain=v1")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return parseStatusOutput(out), nil
}

// Diff returns the diff stat for the working tree.
func (o *Ops) Diff(ctx context.Context, args ...string) (DiffStat, error) {
	cmdArgs := append([]string{"diff", "--stat"}, args...)
	out, err := o.git(ctx, cmdArgs...)
	if err != nil {
		return DiffStat{}, err
	}
	return parseDiffStat(out), nil
}

// DiffRaw returns the raw diff output.
func (o *Ops) DiffRaw(ctx context.Context, args ...string) (string, error) {
	cmdArgs := append([]string{"diff"}, args...)
	return o.git(ctx, cmdArgs...)
}

// Log returns the last n commit entries.
func (o *Ops) Log(ctx context.Context, n int, args ...string) ([]LogEntry, error) {
	cmdArgs := append([]string{
		"log",
		fmt.Sprintf("-n%d", n),
		"--format=%H|%an|%ai|%s",
	}, args...)
	out, err := o.git(ctx, cmdArgs...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return parseLogOutput(out), nil
}

// Commit creates a commit with the given message.
func (o *Ops) Commit(ctx context.Context, message string, args ...string) (string, error) {
	cmdArgs := append([]string{"commit", "-m", message}, args...)
	return o.git(ctx, cmdArgs...)
}

// CommitAll stages all changes and commits.
func (o *Ops) CommitAll(ctx context.Context, message string) (string, error) {
	return o.Commit(ctx, message, "-a")
}

// Add stages files for commit.
func (o *Ops) Add(ctx context.Context, paths ...string) error {
	args := append([]string{"add"}, paths...)
	_, err := o.git(ctx, args...)
	return err
}

// Push pushes to the remote.
func (o *Ops) Push(ctx context.Context, args ...string) error {
	cmdArgs := append([]string{"push"}, args...)
	_, err := o.git(ctx, cmdArgs...)
	return err
}

// Pull pulls from the remote.
func (o *Ops) Pull(ctx context.Context, args ...string) error {
	cmdArgs := append([]string{"pull"}, args...)
	_, err := o.git(ctx, cmdArgs...)
	return err
}

// Rebase rebases the current branch onto the given base.
func (o *Ops) Rebase(ctx context.Context, base string, args ...string) error {
	cmdArgs := append([]string{"rebase", base}, args...)
	_, err := o.git(ctx, cmdArgs...)
	return err
}

// RebaseAbort aborts an in-progress rebase.
func (o *Ops) RebaseAbort(ctx context.Context) error {
	_, err := o.git(ctx, "rebase", "--abort")
	return err
}

// Merge merges the given branch into the current branch.
func (o *Ops) Merge(ctx context.Context, branch string, args ...string) error {
	cmdArgs := append([]string{"merge", branch}, args...)
	_, err := o.git(ctx, cmdArgs...)
	return err
}

// MergeAbort aborts an in-progress merge.
func (o *Ops) MergeAbort(ctx context.Context) error {
	_, err := o.git(ctx, "merge", "--abort")
	return err
}

// CurrentBranch returns the name of the current branch.
func (o *Ops) CurrentBranch(ctx context.Context) (string, error) {
	return o.git(ctx, "rev-parse", "--abbrev-ref", "HEAD")
}

// Rev returns the commit hash for a given ref.
func (o *Ops) Rev(ctx context.Context, ref string) (string, error) {
	return o.git(ctx, "rev-parse", ref)
}

// IsClean returns true if the working tree has no uncommitted changes.
func (o *Ops) IsClean(ctx context.Context) (bool, error) {
	entries, err := o.Status(ctx)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// parseStatusOutput parses git status --porcelain=v1 output.
func parseStatusOutput(out string) []StatusEntry {
	lines := strings.Split(out, "\n")
	entries := make([]StatusEntry, 0, len(lines))
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		entries = append(entries, StatusEntry{
			Staging:  line[0],
			Worktree: line[1],
			Path:     line[3:],
		})
	}
	return entries
}

// parseLogOutput parses git log --format=%H|%an|%ai|%s output.
func parseLogOutput(out string) []LogEntry {
	lines := strings.Split(out, "\n")
	entries := make([]LogEntry, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		entries = append(entries, LogEntry{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    parts[2],
			Subject: parts[3],
		})
	}
	return entries
}

// parseDiffStat parses git diff --stat summary line.
func parseDiffStat(out string) DiffStat {
	stat := DiffStat{Raw: out}
	if out == "" {
		return stat
	}
	lines := strings.Split(out, "\n")
	summary := lines[len(lines)-1]
	// Parse "N files changed, N insertions(+), N deletions(-)"
	parts := strings.Split(summary, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) < 2 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		switch {
		case strings.Contains(part, "changed"):
			stat.FilesChanged = n
		case strings.Contains(part, "insertion"):
			stat.Insertions = n
		case strings.Contains(part, "deletion"):
			stat.Deletions = n
		}
	}
	return stat
}
