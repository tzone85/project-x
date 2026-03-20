package git

import "fmt"

// CommandError wraps a failed git/gh command with context.
type CommandError struct {
	Command string
	Args    []string
	Stderr  string
	Err     error
}

func (e *CommandError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("git command failed: %s: %s", e.Command, e.Stderr)
	}
	return fmt.Sprintf("git command failed: %s: %v", e.Command, e.Err)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

// ConflictError indicates a merge/rebase conflict.
type ConflictError struct {
	Operation string
	Files     []string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s conflict in %d files", e.Operation, len(e.Files))
}

// NotFoundError indicates a branch, worktree, or PR was not found.
type NotFoundError struct {
	Resource string
	Name     string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.Name)
}
