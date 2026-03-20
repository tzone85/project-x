package git

import (
	"errors"
	"fmt"
	"testing"
)

func TestCommandError_Error_WithStderr(t *testing.T) {
	err := &CommandError{
		Command: "git push",
		Stderr:  "permission denied",
		Err:     fmt.Errorf("exit status 1"),
	}
	expected := "git command failed: git push: permission denied"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestCommandError_Error_WithoutStderr(t *testing.T) {
	inner := fmt.Errorf("exit status 1")
	err := &CommandError{
		Command: "git push",
		Err:     inner,
	}
	expected := "git command failed: git push: exit status 1"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestCommandError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &CommandError{
		Command: "git push",
		Err:     inner,
	}
	if !errors.Is(err, inner) {
		t.Error("expected Unwrap to return inner error")
	}
}

func TestConflictError_Error(t *testing.T) {
	err := &ConflictError{
		Operation: "merge",
		Files:     []string{"a.go", "b.go"},
	}
	expected := "merge conflict in 2 files"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestNotFoundError_Error(t *testing.T) {
	err := &NotFoundError{
		Resource: "branch",
		Name:     "feature-x",
	}
	expected := "branch not found: feature-x"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
