package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Exec tests (use a real git repo) ---

func TestRunGitStatus(t *testing.T) {
	dir := initTestRepo(t)

	result, err := RunGit(context.Background(), dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Fresh repo with one commit should have clean status
	if result.Stdout != "" {
		t.Errorf("expected clean status, got %q", result.Stdout)
	}
}

func TestRunGitInvalidCommand(t *testing.T) {
	dir := initTestRepo(t)

	_, err := RunGit(context.Background(), dir, "not-a-command")
	if err == nil {
		t.Fatal("expected error for invalid git command")
	}
}

func TestRunGitContextCancellation(t *testing.T) {
	dir := initTestRepo(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RunGit(ctx, dir, "status")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// --- Ops tests ---

func TestOpsCurrentBranch(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)

	branch, err := ops.CurrentBranch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default branch in test repos is usually main or master
	if branch != "main" && branch != "master" {
		t.Errorf("branch = %q, want main or master", branch)
	}
}

func TestOpsCreateBranchAndCheckout(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)

	ctx := context.Background()
	err := ops.CreateBranch(ctx, "feature/test", "HEAD")
	if err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	branch, _ := ops.CurrentBranch(ctx)
	if branch != "feature/test" {
		t.Errorf("branch = %q, want feature/test", branch)
	}

	err = ops.CheckoutBranch(ctx, "main")
	if err != nil {
		t.Fatalf("CheckoutBranch: %v", err)
	}

	branch, _ = ops.CurrentBranch(ctx)
	if branch != "main" {
		t.Errorf("branch = %q, want main", branch)
	}
}

func TestOpsCommitAndLog(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)

	ctx := context.Background()

	// Create a file and commit
	writeFile(t, filepath.Join(dir, "test.txt"), "hello")
	err := ops.Commit(ctx, "add test file", []string{"test.txt"})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	log, err := ops.Log(ctx, 5)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if !strings.Contains(log, "add test file") {
		t.Errorf("log missing commit message, got %q", log)
	}
}

func TestOpsStatusAndDiff(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)

	ctx := context.Background()

	// Create untracked file
	writeFile(t, filepath.Join(dir, "new.txt"), "content")

	status, err := ops.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !strings.Contains(status, "new.txt") {
		t.Errorf("status missing new file, got %q", status)
	}

	// Stage and check staged diff
	RunGit(ctx, dir, "add", "new.txt")
	diff, err := ops.Diff(ctx, true)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(diff, "content") {
		t.Errorf("staged diff missing content, got %q", diff)
	}
}

func TestOpsHasConflicts(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)

	hasConflicts, err := ops.HasConflicts(context.Background())
	if err != nil {
		t.Fatalf("HasConflicts: %v", err)
	}
	if hasConflicts {
		t.Error("fresh repo should have no conflicts")
	}
}

func TestOpsDeleteBranch(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)

	ctx := context.Background()
	ops.CreateBranch(ctx, "to-delete", "HEAD")
	ops.CheckoutBranch(ctx, "main")

	err := ops.DeleteBranch(ctx, "to-delete")
	if err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}
}

func TestOpsWorktreeListOnMainRepo(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)

	worktrees, err := ops.ListWorktrees(context.Background())
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	// Main repo is always a worktree
	if len(worktrees) == 0 {
		t.Error("expected at least 1 worktree (main)")
	}
}

// --- TechStack tests ---

func TestDetectTechStackGo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test")
	os.MkdirAll(filepath.Join(dir, "internal"), 0755)
	os.MkdirAll(filepath.Join(dir, "cmd"), 0755)

	ts := DetectTechStack(context.Background(), dir)

	if ts.Language != "Go" {
		t.Errorf("language = %q, want Go", ts.Language)
	}
	if ts.TestRunner != "go test" {
		t.Errorf("test runner = %q, want 'go test'", ts.TestRunner)
	}
	if ts.DirLayout != "Go standard (cmd/ + internal/)" {
		t.Errorf("layout = %q, want Go standard", ts.DirLayout)
	}
}

func TestDetectTechStackJS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), "{}")
	writeFile(t, filepath.Join(dir, "next.config.js"), "")
	writeFile(t, filepath.Join(dir, "jest.config.js"), "")
	writeFile(t, filepath.Join(dir, ".eslintrc.json"), "{}")
	writeFile(t, filepath.Join(dir, "webpack.config.js"), "")
	os.MkdirAll(filepath.Join(dir, "src"), 0755)

	ts := DetectTechStack(context.Background(), dir)

	if ts.Language != "JavaScript/TypeScript" {
		t.Errorf("language = %q, want JavaScript/TypeScript", ts.Language)
	}
	if ts.Framework != "Next.js" {
		t.Errorf("framework = %q, want Next.js", ts.Framework)
	}
	if ts.TestRunner != "Jest" {
		t.Errorf("test runner = %q, want Jest", ts.TestRunner)
	}
	if ts.Linter != "ESLint" {
		t.Errorf("linter = %q, want ESLint", ts.Linter)
	}
	if ts.BuildTool != "Webpack" {
		t.Errorf("build tool = %q, want Webpack", ts.BuildTool)
	}
	if ts.DirLayout != "src/" {
		t.Errorf("layout = %q, want src/", ts.DirLayout)
	}
}

func TestDetectTechStackPython(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "")
	writeFile(t, filepath.Join(dir, "ruff.toml"), "")
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	os.MkdirAll(filepath.Join(dir, "tests"), 0755)

	ts := DetectTechStack(context.Background(), dir)

	if ts.Language != "Python" {
		t.Errorf("language = %q, want Python", ts.Language)
	}
	if ts.Linter != "Ruff" {
		t.Errorf("linter = %q, want Ruff", ts.Linter)
	}
	if ts.DirLayout != "src/ + tests/" {
		t.Errorf("layout = %q, want src/ + tests/", ts.DirLayout)
	}
}

func TestDetectTechStackEmpty(t *testing.T) {
	dir := t.TempDir()

	ts := DetectTechStack(context.Background(), dir)
	if ts.Language != "" {
		t.Errorf("language = %q, want empty", ts.Language)
	}
}

func TestDetectTechStackFirstMatchWins(t *testing.T) {
	dir := t.TempDir()
	// Both go.mod and package.json present — Go should win (first in rules)
	writeFile(t, filepath.Join(dir, "go.mod"), "module test")
	writeFile(t, filepath.Join(dir, "package.json"), "{}")

	ts := DetectTechStack(context.Background(), dir)
	if ts.Language != "Go" {
		t.Errorf("language = %q, want Go (first match wins)", ts.Language)
	}
}

// --- Additional ops tests for coverage ---

func TestOpsMergeLocalBranches(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)
	ctx := context.Background()

	// Create a feature branch with a commit
	ops.CreateBranch(ctx, "feature", "main")
	writeFile(t, filepath.Join(dir, "feature.txt"), "feature")
	ops.Commit(ctx, "feature commit", []string{"feature.txt"})

	// Switch to main and merge
	ops.CheckoutBranch(ctx, "main")
	err := ops.Merge(ctx, "feature")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	log, _ := ops.Log(ctx, 5)
	if !strings.Contains(log, "feature commit") {
		t.Error("merge did not bring feature commit into main")
	}
}

func TestOpsRebaseLocalBranches(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)
	ctx := context.Background()

	// Create feature branch
	ops.CreateBranch(ctx, "feature", "main")
	writeFile(t, filepath.Join(dir, "feature.txt"), "feature")
	ops.Commit(ctx, "feature work", []string{"feature.txt"})

	// Add commit on main
	ops.CheckoutBranch(ctx, "main")
	writeFile(t, filepath.Join(dir, "main.txt"), "main")
	ops.Commit(ctx, "main work", []string{"main.txt"})

	// Rebase feature onto main
	ops.CheckoutBranch(ctx, "feature")
	err := ops.Rebase(ctx, "main")
	if err != nil {
		t.Fatalf("Rebase: %v", err)
	}
}

func TestOpsCreateAndRemoveWorktree(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)
	ctx := context.Background()

	// Create a branch for the worktree
	RunGit(ctx, dir, "branch", "wt-branch")

	wtPath := filepath.Join(t.TempDir(), "worktree")
	err := ops.CreateWorktree(ctx, wtPath, "wt-branch")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	worktrees, _ := ops.ListWorktrees(ctx)
	found := false
	for _, wt := range worktrees {
		if strings.Contains(wt, "worktree") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("worktree not listed in %v", worktrees)
	}

	err = ops.RemoveWorktree(ctx, wtPath)
	if err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}
}

func TestOpsUnstagedDiff(t *testing.T) {
	dir := initTestRepo(t)
	ops := NewOps(dir)
	ctx := context.Background()

	// Modify existing file
	writeFile(t, filepath.Join(dir, "README.md"), "# Modified")

	diff, err := ops.Diff(ctx, false)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(diff, "Modified") {
		t.Error("unstaged diff missing content")
	}
}

func TestOpsPushAndPull(t *testing.T) {
	// Create a bare remote repo
	remoteDir := t.TempDir()
	runCmd(context.Background(), remoteDir, "git", "init", "--bare")

	// Create local repo and add remote
	dir := initTestRepo(t)
	ops := NewOps(dir)
	ctx := context.Background()

	RunGit(ctx, dir, "remote", "add", "origin", remoteDir)
	err := ops.Push(ctx, "origin", "main")
	if err != nil {
		t.Fatalf("Push: %v", err)
	}

	// Make a change and push again
	writeFile(t, filepath.Join(dir, "pushed.txt"), "pushed")
	ops.Commit(ctx, "push test", []string{"pushed.txt"})
	err = ops.PushWithUpstream(ctx, "origin", "main")
	if err != nil {
		t.Fatalf("PushWithUpstream: %v", err)
	}

	// Clone into a second local and pull
	dir2 := t.TempDir()
	runCmd(ctx, dir2, "git", "clone", remoteDir, ".")
	ops2 := NewOps(dir2)

	// Add another commit to remote via first repo
	writeFile(t, filepath.Join(dir, "new.txt"), "new")
	ops.Commit(ctx, "new commit", []string{"new.txt"})
	ops.Push(ctx, "origin", "main")

	err = ops2.Pull(ctx, "origin", "main")
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
}

func TestDetectTechStackLibLayout(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0755)

	ts := DetectTechStack(context.Background(), dir)
	if ts.DirLayout != "lib/" {
		t.Errorf("layout = %q, want lib/", ts.DirLayout)
	}
}

// --- Helpers ---

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init", "--initial-branch=main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}

	for _, args := range cmds {
		result, err := runCmd(context.Background(), dir, args[0], args[1:]...)
		if err != nil {
			t.Fatalf("init repo %v: %v\n%s", args, err, result.Stderr)
		}
	}

	// Need at least one commit for branches to work
	writeFile(t, filepath.Join(dir, "README.md"), "# Test")
	runCmd(context.Background(), dir, "git", "add", ".")
	runCmd(context.Background(), dir, "git", "commit", "-m", "initial")

	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
