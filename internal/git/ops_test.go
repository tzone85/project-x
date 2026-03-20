package git

import (
	"context"
	"testing"
)

func TestParseStatusOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []StatusEntry
	}{
		{
			name:  "modified and new files",
			input: " M file1.go\n?? file2.go\nA  file3.go",
			expected: []StatusEntry{
				{Staging: ' ', Worktree: 'M', Path: "file1.go"},
				{Staging: '?', Worktree: '?', Path: "file2.go"},
				{Staging: 'A', Worktree: ' ', Path: "file3.go"},
			},
		},
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name:     "short line ignored",
			input:    "ab",
			expected: []StatusEntry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseStatusOutput(tt.input)
			if tt.expected == nil {
				if result != nil && len(result) != 0 {
					t.Errorf("expected nil/empty, got %v", result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d entries, got %d", len(tt.expected), len(result))
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("entry %d: expected %+v, got %+v", i, exp, result[i])
				}
			}
		})
	}
}

func TestParseLogOutput(t *testing.T) {
	input := "abc123|Alice|2026-01-01 12:00:00|Fix bug\ndef456|Bob|2026-01-02 13:00:00|Add feature"
	entries := parseLogOutput(input)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Hash != "abc123" {
		t.Errorf("expected hash abc123, got %s", entries[0].Hash)
	}
	if entries[0].Author != "Alice" {
		t.Errorf("expected author Alice, got %s", entries[0].Author)
	}
	if entries[1].Subject != "Add feature" {
		t.Errorf("expected subject 'Add feature', got %s", entries[1].Subject)
	}
}

func TestParseLogOutput_Empty(t *testing.T) {
	entries := parseLogOutput("")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseLogOutput_IncompleteLine(t *testing.T) {
	entries := parseLogOutput("abc|only two parts")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for incomplete line, got %d", len(entries))
	}
}

func TestParseDiffStat(t *testing.T) {
	input := " file1.go | 10 ++++------\n file2.go |  5 +++++\n 2 files changed, 7 insertions(+), 6 deletions(-)"
	stat := parseDiffStat(input)

	if stat.FilesChanged != 2 {
		t.Errorf("expected 2 files changed, got %d", stat.FilesChanged)
	}
	if stat.Insertions != 7 {
		t.Errorf("expected 7 insertions, got %d", stat.Insertions)
	}
	if stat.Deletions != 6 {
		t.Errorf("expected 6 deletions, got %d", stat.Deletions)
	}
}

func TestParseDiffStat_Empty(t *testing.T) {
	stat := parseDiffStat("")
	if stat.FilesChanged != 0 {
		t.Errorf("expected 0 files changed, got %d", stat.FilesChanged)
	}
}

func TestOps_Status(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git status --porcelain=v1", " M main.go\n?? new.go", nil)

	ops := NewOps(mock, "/repo")
	entries, err := ops.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Path != "main.go" {
		t.Errorf("expected main.go, got %s", entries[0].Path)
	}
}

func TestOps_Status_Clean(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git status --porcelain=v1", "", nil)

	ops := NewOps(mock, "/repo")
	entries, err := ops.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil for clean repo, got %v", entries)
	}
}

func TestOps_Log(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git log", "abc|Author|2026-01-01|Subject line", nil)

	ops := NewOps(mock, "/repo")
	entries, err := ops.Log(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestOps_Commit(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git commit", "1 file changed", nil)

	ops := NewOps(mock, "/repo")
	out, err := ops.Commit(context.Background(), "test commit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "1 file changed" {
		t.Errorf("expected '1 file changed', got %q", out)
	}
	if !mock.Called("git commit") {
		t.Error("expected commit to be called")
	}
}

func TestOps_CommitAll(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git commit", "ok", nil)

	ops := NewOps(mock, "/repo")
	_, err := ops.CommitAll(context.Background(), "all changes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("-a") {
		t.Error("expected -a flag")
	}
}

func TestOps_Push(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git push", "", nil)

	ops := NewOps(mock, "/repo")
	err := ops.Push(context.Background(), "origin", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("git push") {
		t.Error("expected push to be called")
	}
}

func TestOps_Pull(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git pull", "", nil)

	ops := NewOps(mock, "/repo")
	err := ops.Pull(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOps_Rebase(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git rebase", "", nil)

	ops := NewOps(mock, "/repo")
	err := ops.Rebase(context.Background(), "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("git rebase main") {
		t.Error("expected rebase main")
	}
}

func TestOps_Merge(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git merge", "", nil)

	ops := NewOps(mock, "/repo")
	err := ops.Merge(context.Background(), "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOps_IsClean_True(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git status --porcelain=v1", "", nil)

	ops := NewOps(mock, "/repo")
	clean, err := ops.IsClean(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !clean {
		t.Error("expected clean")
	}
}

func TestOps_IsClean_False(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git status --porcelain=v1", " M file.go", nil)

	ops := NewOps(mock, "/repo")
	clean, err := ops.IsClean(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clean {
		t.Error("expected not clean")
	}
}

func TestOps_Diff(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git diff --stat", " 2 files changed, 5 insertions(+), 3 deletions(-)", nil)

	ops := NewOps(mock, "/repo")
	stat, err := ops.Diff(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stat.FilesChanged != 2 {
		t.Errorf("expected 2 files changed, got %d", stat.FilesChanged)
	}
}

func TestOps_CurrentBranch(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git rev-parse --abbrev-ref HEAD", "main", nil)

	ops := NewOps(mock, "/repo")
	branch, err := ops.CurrentBranch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected main, got %s", branch)
	}
}

func TestOps_Rev(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git rev-parse HEAD", "abc123def", nil)

	ops := NewOps(mock, "/repo")
	hash, err := ops.Rev(context.Background(), "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "abc123def" {
		t.Errorf("expected abc123def, got %s", hash)
	}
}

func TestOps_Add(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git add", "", nil)

	ops := NewOps(mock, "/repo")
	err := ops.Add(context.Background(), "file1.go", "file2.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("git add file1.go file2.go") {
		t.Error("expected add with both files")
	}
}

func TestOps_RebaseAbort(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git rebase --abort", "", nil)

	ops := NewOps(mock, "/repo")
	err := ops.RebaseAbort(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOps_MergeAbort(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git merge --abort", "", nil)

	ops := NewOps(mock, "/repo")
	err := ops.MergeAbort(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOps_DiffRaw(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("git diff", "+added line\n-removed line", nil)

	ops := NewOps(mock, "/repo")
	out, err := ops.DiffRaw(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty diff output")
	}
}
