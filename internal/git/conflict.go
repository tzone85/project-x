package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// Conflict provides conflict detection and resolution helpers.
type Conflict struct {
	ops *Ops
}

// NewConflict creates a Conflict helper using the given Ops.
func NewConflict(ops *Ops) *Conflict {
	return &Conflict{ops: ops}
}

// Detect checks for merge/rebase conflicts and returns conflict info.
func (c *Conflict) Detect(ctx context.Context) (ConflictInfo, error) {
	info := ConflictInfo{}

	conflictType := c.detectConflictType()
	if conflictType != "" {
		info.InProgress = true
		info.Type = conflictType
	}

	files, err := c.conflictedFiles(ctx)
	if err != nil {
		return info, err
	}
	info.ConflictFiles = files

	if len(files) > 0 {
		info.InProgress = true
	}

	return info, nil
}

// HasConflicts returns true if there are unresolved conflicts.
func (c *Conflict) HasConflicts(ctx context.Context) (bool, error) {
	files, err := c.conflictedFiles(ctx)
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

// conflictedFiles returns files with unmerged status.
func (c *Conflict) conflictedFiles(ctx context.Context) ([]string, error) {
	out, err := c.ops.git(ctx, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// detectConflictType checks which type of operation is in progress by
// looking for git state directories.
func (c *Conflict) detectConflictType() string {
	gitDir := filepath.Join(c.ops.dir, ".git")

	if fileExists(filepath.Join(gitDir, "rebase-merge")) ||
		fileExists(filepath.Join(gitDir, "rebase-apply")) {
		return "rebase"
	}
	if fileExists(filepath.Join(gitDir, "MERGE_HEAD")) {
		return "merge"
	}
	if fileExists(filepath.Join(gitDir, "CHERRY_PICK_HEAD")) {
		return "cherry-pick"
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
