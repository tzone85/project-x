package git

import (
	"context"
	"testing"
)

func TestWorktree_Add(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git worktree add /tmp/wt feature", "", nil)

	ops := NewOps(mock, "/repo")
	wt := NewWorktree(ops)
	err := wt.Add(context.Background(), "/tmp/wt", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorktree_AddNew(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git worktree add -b new-branch /tmp/wt", "", nil)

	ops := NewOps(mock, "/repo")
	wt := NewWorktree(ops)
	err := wt.AddNew(context.Background(), "/tmp/wt", "new-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorktree_Remove(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git worktree remove /tmp/wt", "", nil)

	ops := NewOps(mock, "/repo")
	wt := NewWorktree(ops)
	err := wt.Remove(context.Background(), "/tmp/wt", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorktree_Remove_Force(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git worktree remove /tmp/wt --force", "", nil)

	ops := NewOps(mock, "/repo")
	wt := NewWorktree(ops)
	err := wt.Remove(context.Background(), "/tmp/wt", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorktree_Prune(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git worktree prune", "", nil)

	ops := NewOps(mock, "/repo")
	wt := NewWorktree(ops)
	err := wt.Prune(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseWorktreeOutput(t *testing.T) {
	input := `worktree /home/user/repo
HEAD abc123def456
branch refs/heads/main

worktree /tmp/wt-feature
HEAD def789abc123
branch refs/heads/feature`

	trees := parseWorktreeOutput(input)
	if len(trees) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(trees))
	}

	if trees[0].Path != "/home/user/repo" {
		t.Errorf("expected path /home/user/repo, got %s", trees[0].Path)
	}
	if trees[0].Head != "abc123def456" {
		t.Errorf("expected HEAD abc123def456, got %s", trees[0].Head)
	}
	if trees[0].Branch != "main" {
		t.Errorf("expected branch main, got %s", trees[0].Branch)
	}

	if trees[1].Branch != "feature" {
		t.Errorf("expected branch feature, got %s", trees[1].Branch)
	}
}

func TestParseWorktreeOutput_Bare(t *testing.T) {
	input := `worktree /home/user/repo
HEAD abc123
bare`

	trees := parseWorktreeOutput(input)
	if len(trees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(trees))
	}
	if !trees[0].Bare {
		t.Error("expected bare to be true")
	}
}

func TestParseWorktreeOutput_Empty(t *testing.T) {
	trees := parseWorktreeOutput("")
	if len(trees) != 0 {
		t.Errorf("expected 0 worktrees, got %d", len(trees))
	}
}

func TestWorktree_List(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git worktree list --porcelain", "worktree /repo\nHEAD abc\nbranch refs/heads/main", nil)

	ops := NewOps(mock, "/repo")
	wt := NewWorktree(ops)
	trees, err := wt.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(trees))
	}
	if trees[0].Branch != "main" {
		t.Errorf("expected branch main, got %s", trees[0].Branch)
	}
}

func TestWorktree_List_Empty(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git worktree list --porcelain", "", nil)

	ops := NewOps(mock, "/repo")
	wt := NewWorktree(ops)
	trees, err := wt.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trees != nil {
		t.Errorf("expected nil, got %v", trees)
	}
}
