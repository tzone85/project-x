package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestConflict_Detect_NoConflicts(t *testing.T) {
	dir := t.TempDir()
	// Create a .git dir without conflict markers
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	mock := NewMockRunner()
	mock.Stub("git diff --name-only --diff-filter=U", "", nil)

	ops := NewOps(mock, dir)
	c := NewConflict(ops)
	info, err := c.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.InProgress {
		t.Error("expected no conflict in progress")
	}
	if len(info.ConflictFiles) != 0 {
		t.Errorf("expected no conflict files, got %v", info.ConflictFiles)
	}
}

func TestConflict_Detect_MergeConflict(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create MERGE_HEAD to indicate merge in progress
	if err := os.WriteFile(filepath.Join(gitDir, "MERGE_HEAD"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock := NewMockRunner()
	mock.Stub("git diff --name-only --diff-filter=U", "file1.go\nfile2.go", nil)

	ops := NewOps(mock, dir)
	c := NewConflict(ops)
	info, err := c.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.InProgress {
		t.Error("expected conflict in progress")
	}
	if info.Type != "merge" {
		t.Errorf("expected type 'merge', got %s", info.Type)
	}
	if len(info.ConflictFiles) != 2 {
		t.Errorf("expected 2 conflict files, got %d", len(info.ConflictFiles))
	}
}

func TestConflict_Detect_RebaseConflict(t *testing.T) {
	dir := t.TempDir()
	rebaseDir := filepath.Join(dir, ".git", "rebase-merge")
	if err := os.MkdirAll(rebaseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	mock := NewMockRunner()
	mock.Stub("git diff --name-only --diff-filter=U", "file1.go", nil)

	ops := NewOps(mock, dir)
	c := NewConflict(ops)
	info, err := c.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Type != "rebase" {
		t.Errorf("expected type 'rebase', got %s", info.Type)
	}
}

func TestConflict_Detect_RebaseApply(t *testing.T) {
	dir := t.TempDir()
	rebaseDir := filepath.Join(dir, ".git", "rebase-apply")
	if err := os.MkdirAll(rebaseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	mock := NewMockRunner()
	mock.Stub("git diff --name-only --diff-filter=U", "", nil)

	ops := NewOps(mock, dir)
	c := NewConflict(ops)
	info, err := c.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Type != "rebase" {
		t.Errorf("expected type 'rebase', got %s", info.Type)
	}
}

func TestConflict_Detect_CherryPickConflict(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "CHERRY_PICK_HEAD"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock := NewMockRunner()
	mock.Stub("git diff --name-only --diff-filter=U", "", nil)

	ops := NewOps(mock, dir)
	c := NewConflict(ops)
	info, err := c.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Type != "cherry-pick" {
		t.Errorf("expected type 'cherry-pick', got %s", info.Type)
	}
}

func TestConflict_HasConflicts(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git diff --name-only --diff-filter=U", "conflict.go", nil)

	ops := NewOps(mock, "/repo")
	c := NewConflict(ops)
	has, err := c.HasConflicts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Error("expected conflicts")
	}
}

func TestConflict_HasConflicts_None(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git diff --name-only --diff-filter=U", "", nil)

	ops := NewOps(mock, "/repo")
	c := NewConflict(ops)
	has, err := c.HasConflicts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Error("expected no conflicts")
	}
}

func TestConflict_Detect_UnmergedFilesOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// No conflict state files, but git reports unmerged files
	mock := NewMockRunner()
	mock.Stub("git diff --name-only --diff-filter=U", "file.go", nil)

	ops := NewOps(mock, dir)
	c := NewConflict(ops)
	info, err := c.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.InProgress {
		t.Error("expected InProgress when unmerged files exist")
	}
}
