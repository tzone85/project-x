package git

import "fmt"

// CreateWorktree creates a new git worktree with a branch at the given path.
// If the branch already exists (from a prior failed attempt), it reuses it
// instead of failing. Also cleans up stale worktree entries.
func CreateWorktree(runner CommandRunner, repoDir, worktreePath, branch string) error {
	// Prune stale worktree entries first (handles leftover from crashes).
	runner.Run(repoDir, "git", "worktree", "prune")

	// Remove existing worktree directory if it exists.
	runner.Run(repoDir, "git", "worktree", "remove", worktreePath, "--force")

	// Try creating with a new branch first.
	_, err := runner.Run(repoDir, "git", "worktree", "add", "-b", branch, worktreePath)
	if err != nil {
		// Branch may already exist — delete it and retry.
		runner.Run(repoDir, "git", "branch", "-D", branch)
		_, err = runner.Run(repoDir, "git", "worktree", "add", "-b", branch, worktreePath)
		if err != nil {
			return fmt.Errorf("creating worktree: %w", err)
		}
	}
	return nil
}

// RemoveWorktree removes a worktree and deletes the associated local branch.
func RemoveWorktree(runner CommandRunner, repoDir, worktreePath, branch string) error {
	_, err := runner.Run(repoDir, "git", "worktree", "remove", worktreePath, "--force")
	if err != nil {
		return fmt.Errorf("removing worktree: %w", err)
	}

	_, err = runner.Run(repoDir, "git", "branch", "-D", branch)
	if err != nil {
		return fmt.Errorf("deleting branch: %w", err)
	}

	return nil
}
