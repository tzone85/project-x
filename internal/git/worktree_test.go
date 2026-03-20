package git

import (
	"errors"
	"strings"
	"testing"
)

func TestCreateWorktree_CallsCorrectCommands(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", nil) // git worktree add

	err := CreateWorktree(mock, "/repo", "/tmp/worktree-abc", "px/story-123")
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

	argsStr := strings.Join(cmd.Args, " ")
	expected := "worktree add -b px/story-123 /tmp/worktree-abc"
	if argsStr != expected {
		t.Errorf("expected %q, got %q", expected, argsStr)
	}
}

func TestCreateWorktree_PropagatesError(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", errors.New("already exists"))

	err := CreateWorktree(mock, "/repo", "/tmp/worktree-abc", "px/story-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected error to contain 'already exists', got %q", err.Error())
	}
}

func TestRemoveWorktree_CallsCleanupCommands(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", nil) // git worktree remove
	mock.AddResponse("", nil) // git branch -D

	err := RemoveWorktree(mock, "/repo", "/tmp/worktree-abc", "px/story-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(mock.Commands))
	}

	// First command: worktree remove
	first := mock.Commands[0]
	if first.Dir != "/repo" {
		t.Errorf("expected dir /repo, got %s", first.Dir)
	}
	firstArgs := strings.Join(first.Args, " ")
	if firstArgs != "worktree remove /tmp/worktree-abc --force" {
		t.Errorf("expected 'worktree remove /tmp/worktree-abc --force', got %q", firstArgs)
	}

	// Second command: branch -D
	second := mock.Commands[1]
	if second.Dir != "/repo" {
		t.Errorf("expected dir /repo, got %s", second.Dir)
	}
	secondArgs := strings.Join(second.Args, " ")
	if secondArgs != "branch -D px/story-123" {
		t.Errorf("expected 'branch -D px/story-123', got %q", secondArgs)
	}
}

func TestRemoveWorktree_WorktreeRemoveError(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", errors.New("not a worktree"))

	err := RemoveWorktree(mock, "/repo", "/tmp/worktree-abc", "px/story-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not a worktree") {
		t.Errorf("expected error to contain 'not a worktree', got %q", err.Error())
	}
}

func TestRemoveWorktree_BranchDeleteError(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", nil)                        // worktree remove succeeds
	mock.AddResponse("", errors.New("branch in use")) // branch -D fails

	err := RemoveWorktree(mock, "/repo", "/tmp/worktree-abc", "px/story-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "branch in use") {
		t.Errorf("expected error to contain 'branch in use', got %q", err.Error())
	}
}
