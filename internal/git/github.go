package git

import (
	"fmt"
	"strconv"
	"strings"
)

// PRResult holds the result of a PR creation or merge operation.
type PRResult struct {
	PRNumber int
	PRURL    string
	Merged   bool
}

// CreatePR creates a GitHub pull request using the gh CLI.
func CreatePR(runner CommandRunner, repoDir, branch, title, body, baseBranch string) (PRResult, error) {
	output, err := runner.Run(repoDir, "gh",
		"pr", "create",
		"--head", branch,
		"--base", baseBranch,
		"--title", title,
		"--body", body,
	)
	if err != nil {
		return PRResult{}, fmt.Errorf("creating PR: %w", err)
	}

	prURL := strings.TrimSpace(output)
	prNumber, err := parsePRNumber(prURL)
	if err != nil {
		return PRResult{}, fmt.Errorf("parsing PR URL %q: %w", prURL, err)
	}

	return PRResult{
		PRNumber: prNumber,
		PRURL:    prURL,
	}, nil
}

// MergePR merges a GitHub pull request using the gh CLI with squash strategy.
func MergePR(runner CommandRunner, repoDir string, prNumber int, autoMerge bool) error {
	args := []string{"pr", "merge", strconv.Itoa(prNumber), "--squash"}
	if autoMerge {
		args = append(args, "--auto")
	}

	_, err := runner.Run(repoDir, "gh", args...)
	if err != nil {
		return fmt.Errorf("merging PR #%d: %w", prNumber, err)
	}
	return nil
}

// parsePRNumber extracts the PR number from a GitHub PR URL.
// Expected format: https://github.com/owner/repo/pull/<number>
func parsePRNumber(url string) (int, error) {
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid PR URL format: %s", url)
	}

	lastPart := parts[len(parts)-1]
	secondLast := parts[len(parts)-2]

	if secondLast != "pull" {
		return 0, fmt.Errorf("invalid PR URL format (expected /pull/<number>): %s", url)
	}

	num, err := strconv.Atoi(lastPart)
	if err != nil {
		return 0, fmt.Errorf("invalid PR number in URL %s: %w", url, err)
	}
	return num, nil
}
