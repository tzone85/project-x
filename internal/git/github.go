package git

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// GitHub provides GitHub operations via the gh CLI.
type GitHub struct {
	runner CommandRunner
	dir    string
}

// NewGitHub creates a GitHub operations helper.
func NewGitHub(runner CommandRunner, dir string) *GitHub {
	return &GitHub{runner: runner, dir: dir}
}

// gh runs a gh subcommand in the repo directory.
func (g *GitHub) gh(ctx context.Context, args ...string) (string, error) {
	return g.runner.Run(ctx, g.dir, "gh", args...)
}

// CreatePR creates a pull request and returns the PR URL.
func (g *GitHub) CreatePR(ctx context.Context, opts PRCreateOptions) (string, error) {
	args := []string{"pr", "create",
		"--title", opts.Title,
		"--body", opts.Body,
	}
	if opts.Base != "" {
		args = append(args, "--base", opts.Base)
	}
	if opts.Head != "" {
		args = append(args, "--head", opts.Head)
	}
	if opts.Draft {
		args = append(args, "--draft")
	}
	return g.gh(ctx, args...)
}

// MergePR merges a pull request by number.
func (g *GitHub) MergePR(ctx context.Context, number int, opts PRMergeOptions) error {
	args := []string{"pr", "merge", strconv.Itoa(number)}

	switch opts.Method {
	case "squash":
		args = append(args, "--squash")
	case "rebase":
		args = append(args, "--rebase")
	default:
		args = append(args, "--merge")
	}

	if opts.DeleteBranch {
		args = append(args, "--delete-branch")
	}
	if opts.AutoMerge {
		args = append(args, "--auto")
	}
	if opts.CommitSubject != "" {
		args = append(args, "--subject", opts.CommitSubject)
	}

	_, err := g.gh(ctx, args...)
	return err
}

// ListPRs returns open pull requests. Optional filters can be passed
// as gh pr list flags (e.g., "--state", "closed").
func (g *GitHub) ListPRs(ctx context.Context, filters ...string) ([]PRInfo, error) {
	args := []string{"pr", "list", "--json",
		"number,title,state,url,headRefName,baseRefName,author",
	}
	args = append(args, filters...)

	out, err := g.gh(ctx, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var raw []struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		State       string `json:"state"`
		URL         string `json:"url"`
		HeadRefName string `json:"headRefName"`
		BaseRefName string `json:"baseRefName"`
		Author      struct {
			Login string `json:"login"`
		} `json:"author"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, fmt.Errorf("parse PR list: %w", err)
	}

	prs := make([]PRInfo, len(raw))
	for i, r := range raw {
		prs[i] = PRInfo{
			Number: r.Number,
			Title:  r.Title,
			State:  r.State,
			URL:    r.URL,
			Branch: r.HeadRefName,
			Base:   r.BaseRefName,
			Author: r.Author.Login,
		}
	}
	return prs, nil
}

// GetPR returns info about a specific PR by number.
func (g *GitHub) GetPR(ctx context.Context, number int) (PRInfo, error) {
	out, err := g.gh(ctx, "pr", "view", strconv.Itoa(number), "--json",
		"number,title,state,url,headRefName,baseRefName,author")
	if err != nil {
		return PRInfo{}, err
	}

	var raw struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		State       string `json:"state"`
		URL         string `json:"url"`
		HeadRefName string `json:"headRefName"`
		BaseRefName string `json:"baseRefName"`
		Author      struct {
			Login string `json:"login"`
		} `json:"author"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return PRInfo{}, fmt.Errorf("parse PR: %w", err)
	}

	return PRInfo{
		Number: raw.Number,
		Title:  raw.Title,
		State:  raw.State,
		URL:    raw.URL,
		Branch: raw.HeadRefName,
		Base:   raw.BaseRefName,
		Author: raw.Author.Login,
	}, nil
}

// AddReviewers adds reviewers to a pull request.
func (g *GitHub) AddReviewers(ctx context.Context, number int, reviewers []string) error {
	args := []string{"pr", "edit", strconv.Itoa(number),
		"--add-reviewer", strings.Join(reviewers, ","),
	}
	_, err := g.gh(ctx, args...)
	return err
}
