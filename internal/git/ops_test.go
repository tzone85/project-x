package git

import (
	"errors"
	"strings"
	"testing"
)

// --- MockRunner ---

func TestMockRunner_RecordsCommands(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", nil)
	mock.AddResponse("abc123", nil)

	_, _ = mock.Run("/repo", "git", "status")
	_, _ = mock.Run("/repo", "git", "log", "--oneline")

	if len(mock.Commands) != 2 {
		t.Fatalf("expected 2 commands recorded, got %d", len(mock.Commands))
	}

	first := mock.Commands[0]
	if first.Dir != "/repo" || first.Name != "git" || strings.Join(first.Args, " ") != "status" {
		t.Errorf("first command mismatch: %+v", first)
	}

	second := mock.Commands[1]
	if second.Dir != "/repo" || second.Name != "git" || strings.Join(second.Args, " ") != "log --oneline" {
		t.Errorf("second command mismatch: %+v", second)
	}
}

func TestMockRunner_ReturnsConfiguredOutput(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("hello", nil)
	mock.AddResponse("", errors.New("fail"))

	out, err := mock.Run("/dir", "echo", "hello")
	if out != "hello" || err != nil {
		t.Errorf("expected (hello, nil), got (%q, %v)", out, err)
	}

	out, err = mock.Run("/dir", "fail-cmd")
	if out != "" || err == nil || err.Error() != "fail" {
		t.Errorf("expected ('', fail), got (%q, %v)", out, err)
	}
}

func TestMockRunner_PanicsWhenExhausted(t *testing.T) {
	mock := &MockRunner{}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when no responses configured")
		}
	}()

	_, _ = mock.Run("/dir", "git", "status")
}

// --- FetchBranch ---

func TestFetchBranch_CallsGitFetch(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", nil)

	err := FetchBranch(mock, "/repo", "feature-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(mock.Commands))
	}

	cmd := mock.Commands[0]
	if cmd.Dir != "/repo" {
		t.Errorf("expected dir /repo, got %s", cmd.Dir)
	}
	if cmd.Name != "git" {
		t.Errorf("expected git command, got %s", cmd.Name)
	}

	argsStr := strings.Join(cmd.Args, " ")
	if argsStr != "fetch origin feature-branch" {
		t.Errorf("expected 'fetch origin feature-branch', got %q", argsStr)
	}
}

func TestFetchBranch_PropagatesError(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", errors.New("network error"))

	err := FetchBranch(mock, "/repo", "main")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("expected error to contain 'network error', got %q", err.Error())
	}
}

// --- RebaseOnto ---

func TestRebaseOnto_CallsGitRebase(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", nil)

	err := RebaseOnto(mock, "/worktree", "origin/main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(mock.Commands))
	}

	cmd := mock.Commands[0]
	if cmd.Dir != "/worktree" {
		t.Errorf("expected dir /worktree, got %s", cmd.Dir)
	}

	argsStr := strings.Join(cmd.Args, " ")
	if argsStr != "rebase origin/main" {
		t.Errorf("expected 'rebase origin/main', got %q", argsStr)
	}
}

func TestRebaseOnto_PropagatesError(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", errors.New("conflict"))

	err := RebaseOnto(mock, "/worktree", "origin/main")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Errorf("expected error to contain 'conflict', got %q", err.Error())
	}
}

// --- Diff ---

func TestDiff_ReturnsChanges(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("abc123", nil) // merge-base
	mock.AddResponse("diff --git a/file.go b/file.go\n+added line", nil) // diff

	diff, err := Diff(mock, "/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "+added line") {
		t.Errorf("expected diff to contain '+added line', got %q", diff)
	}
}

func TestDiff_EmptyDiff(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("abc123", nil) // merge-base
	mock.AddResponse("", nil)       // diff (empty)

	diff, err := Diff(mock, "/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff, got %q", diff)
	}
}

func TestDiff_MergeBaseError(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", errors.New("no merge base"))

	_, err := Diff(mock, "/worktree")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- MergeBase ---

func TestMergeBase_ReturnsCommit(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("abc123def456", nil)

	commit, err := MergeBase(mock, "/worktree", "HEAD", "origin/main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if commit != "abc123def456" {
		t.Errorf("expected abc123def456, got %q", commit)
	}

	cmd := mock.Commands[0]
	argsStr := strings.Join(cmd.Args, " ")
	if argsStr != "merge-base HEAD origin/main" {
		t.Errorf("expected 'merge-base HEAD origin/main', got %q", argsStr)
	}
}

// --- DeleteRemoteBranch ---

func TestDeleteRemoteBranch_CallsGitPush(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", nil)

	err := DeleteRemoteBranch(mock, "/repo", "feature-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := mock.Commands[0]
	argsStr := strings.Join(cmd.Args, " ")
	if argsStr != "push origin --delete feature-branch" {
		t.Errorf("expected 'push origin --delete feature-branch', got %q", argsStr)
	}
}

func TestDeleteRemoteBranch_PropagatesError(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", errors.New("permission denied"))

	err := DeleteRemoteBranch(mock, "/repo", "main")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- DiffNameOnly ---

func TestDiffNameOnly_ReturnsFileList(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("file1.go\nfile2.go\nfile3_test.go", nil)

	files, err := DiffNameOnly(mock, "/worktree", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	if files[0] != "file1.go" || files[1] != "file2.go" || files[2] != "file3_test.go" {
		t.Errorf("unexpected files: %v", files)
	}

	cmd := mock.Commands[0]
	argsStr := strings.Join(cmd.Args, " ")
	if argsStr != "diff --name-only abc123" {
		t.Errorf("expected 'diff --name-only abc123', got %q", argsStr)
	}
}

func TestDiffNameOnly_EmptyDiff(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", nil)

	files, err := DiffNameOnly(mock, "/worktree", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d: %v", len(files), files)
	}
}
