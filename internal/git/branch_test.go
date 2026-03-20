package git

import (
	"context"
	"fmt"
	"testing"
)

func TestBranch_Create(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git branch feature", "", nil)

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	err := b.Create(context.Background(), "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("git branch feature") {
		t.Error("expected branch create called")
	}
}

func TestBranch_CreateFrom(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git branch feature main", "", nil)

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	err := b.CreateFrom(context.Background(), "feature", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBranch_Checkout(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git checkout main", "", nil)

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	err := b.Checkout(context.Background(), "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBranch_CheckoutNew(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git checkout -b feature", "", nil)

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	err := b.CheckoutNew(context.Background(), "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBranch_Delete(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git branch -d old", "", nil)

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	err := b.Delete(context.Background(), "old", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBranch_Delete_Force(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git branch -D old", "", nil)

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	err := b.Delete(context.Background(), "old", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("-D") {
		t.Error("expected -D flag for force delete")
	}
}

func TestBranch_List(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git branch --format=%(refname:short)", "main\nfeature\ndev", nil)

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	branches, err := b.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 3 {
		t.Fatalf("expected 3 branches, got %d", len(branches))
	}
	if branches[0] != "main" {
		t.Errorf("expected first branch 'main', got %s", branches[0])
	}
}

func TestBranch_List_Empty(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git branch --format=%(refname:short)", "", nil)

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	branches, err := b.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branches != nil {
		t.Errorf("expected nil for empty list, got %v", branches)
	}
}

func TestBranch_Current(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git rev-parse --abbrev-ref HEAD", "main", nil)

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	name, err := b.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "main" {
		t.Errorf("expected 'main', got %s", name)
	}
}

func TestBranch_Exists_True(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git rev-parse --verify refs/heads/main", "abc123", nil)

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	exists, err := b.Exists(context.Background(), "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected branch to exist")
	}
}

func TestBranch_Exists_False(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git rev-parse --verify refs/heads/nonexistent", "", &CommandError{
		Command: "git rev-parse",
		Err:     fmt.Errorf("not found"),
	})

	ops := NewOps(mock, "/repo")
	b := NewBranch(ops)
	exists, err := b.Exists(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected branch not to exist")
	}
}
