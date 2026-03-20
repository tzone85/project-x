package git

import (
	"errors"
	"strings"
	"testing"
)

func TestCreatePR_ParsesURLAndNumber(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("https://github.com/owner/repo/pull/42", nil)

	result, err := CreatePR(mock, "/repo", "feature-branch", "Add feature", "Description body", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PRNumber != 42 {
		t.Errorf("expected PR number 42, got %d", result.PRNumber)
	}
	if result.PRURL != "https://github.com/owner/repo/pull/42" {
		t.Errorf("expected PR URL, got %q", result.PRURL)
	}

	cmd := mock.Commands[0]
	if cmd.Dir != "/repo" {
		t.Errorf("expected dir /repo, got %s", cmd.Dir)
	}
	if cmd.Name != "gh" {
		t.Errorf("expected gh command, got %s", cmd.Name)
	}

	argsStr := strings.Join(cmd.Args, " ")
	if !strings.Contains(argsStr, "--head feature-branch") {
		t.Errorf("expected --head flag, got %q", argsStr)
	}
	if !strings.Contains(argsStr, "--base main") {
		t.Errorf("expected --base flag, got %q", argsStr)
	}
	if !strings.Contains(argsStr, "--title Add feature") {
		t.Errorf("expected --title flag, got %q", argsStr)
	}
}

func TestCreatePR_ParsesHighPRNumber(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("https://github.com/owner/repo/pull/1234", nil)

	result, err := CreatePR(mock, "/repo", "branch", "Title", "Body", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PRNumber != 1234 {
		t.Errorf("expected PR number 1234, got %d", result.PRNumber)
	}
}

func TestCreatePR_GHFailure(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", errors.New("gh not found"))

	_, err := CreatePR(mock, "/repo", "branch", "Title", "Body", "main")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "gh not found") {
		t.Errorf("expected error to contain 'gh not found', got %q", err.Error())
	}
}

func TestCreatePR_InvalidURLFormat(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("not-a-valid-url", nil)

	_, err := CreatePR(mock, "/repo", "branch", "Title", "Body", "main")
	if err == nil {
		t.Fatal("expected error for invalid URL format, got nil")
	}
}

func TestMergePR_WithAutoMerge(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", nil)

	err := MergePR(mock, "/repo", 42, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := mock.Commands[0]
	if cmd.Name != "gh" {
		t.Errorf("expected gh command, got %s", cmd.Name)
	}

	argsStr := strings.Join(cmd.Args, " ")
	if !strings.Contains(argsStr, "pr merge 42") {
		t.Errorf("expected 'pr merge 42', got %q", argsStr)
	}
	if !strings.Contains(argsStr, "--squash") {
		t.Errorf("expected --squash, got %q", argsStr)
	}
	if !strings.Contains(argsStr, "--auto") {
		t.Errorf("expected --auto for autoMerge, got %q", argsStr)
	}
}

func TestMergePR_WithoutAutoMerge(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", nil)

	err := MergePR(mock, "/repo", 99, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := mock.Commands[0]
	argsStr := strings.Join(cmd.Args, " ")
	if !strings.Contains(argsStr, "pr merge 99") {
		t.Errorf("expected 'pr merge 99', got %q", argsStr)
	}
	if !strings.Contains(argsStr, "--squash") {
		t.Errorf("expected --squash, got %q", argsStr)
	}
	if strings.Contains(argsStr, "--auto") {
		t.Errorf("did not expect --auto, got %q", argsStr)
	}
}

func TestMergePR_PropagatesError(t *testing.T) {
	mock := &MockRunner{}
	mock.AddResponse("", errors.New("merge conflict"))

	err := MergePR(mock, "/repo", 42, true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "merge conflict") {
		t.Errorf("expected error to contain 'merge conflict', got %q", err.Error())
	}
}
