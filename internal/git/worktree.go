package git

import "fmt"

// CreateWorktree creates a new git worktree with a new branch at the given path.
func CreateWorktree(runner CommandRunner, repoDir, worktreePath, branch string) error {
	_, err := runner.Run(repoDir, "git", "worktree", "add", "-b", branch, worktreePath)
	if err != nil {
		return fmt.Errorf("creating worktree: %w", err)
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
