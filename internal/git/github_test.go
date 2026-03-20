package git

import (
	"context"
	"testing"
)

func TestGitHub_CreatePR(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr create", "https://github.com/org/repo/pull/42", nil)

	gh := NewGitHub(mock, "/repo")
	url, err := gh.CreatePR(context.Background(), PRCreateOptions{
		Title: "Add feature",
		Body:  "Description",
		Base:  "main",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://github.com/org/repo/pull/42" {
		t.Errorf("expected PR URL, got %s", url)
	}
	if !mock.Called("--title") {
		t.Error("expected --title flag")
	}
	if !mock.Called("--base") {
		t.Error("expected --base flag")
	}
}

func TestGitHub_CreatePR_Draft(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr create", "https://github.com/org/repo/pull/1", nil)

	gh := NewGitHub(mock, "/repo")
	_, err := gh.CreatePR(context.Background(), PRCreateOptions{
		Title: "WIP",
		Body:  "Work in progress",
		Draft: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("--draft") {
		t.Error("expected --draft flag")
	}
}

func TestGitHub_CreatePR_WithHead(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr create", "url", nil)

	gh := NewGitHub(mock, "/repo")
	_, err := gh.CreatePR(context.Background(), PRCreateOptions{
		Title: "PR",
		Body:  "body",
		Head:  "feature-branch",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("--head") {
		t.Error("expected --head flag")
	}
}

func TestGitHub_MergePR_Default(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr merge", "", nil)

	gh := NewGitHub(mock, "/repo")
	err := gh.MergePR(context.Background(), 42, PRMergeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("--merge") {
		t.Error("expected --merge flag as default")
	}
}

func TestGitHub_MergePR_Squash(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr merge", "", nil)

	gh := NewGitHub(mock, "/repo")
	err := gh.MergePR(context.Background(), 42, PRMergeOptions{
		Method:       "squash",
		DeleteBranch: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("--squash") {
		t.Error("expected --squash flag")
	}
	if !mock.Called("--delete-branch") {
		t.Error("expected --delete-branch flag")
	}
}

func TestGitHub_MergePR_Rebase(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr merge", "", nil)

	gh := NewGitHub(mock, "/repo")
	err := gh.MergePR(context.Background(), 1, PRMergeOptions{Method: "rebase"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("--rebase") {
		t.Error("expected --rebase flag")
	}
}

func TestGitHub_MergePR_AutoMerge(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr merge", "", nil)

	gh := NewGitHub(mock, "/repo")
	err := gh.MergePR(context.Background(), 1, PRMergeOptions{
		AutoMerge:     true,
		CommitSubject: "feat: merged",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("--auto") {
		t.Error("expected --auto flag")
	}
	if !mock.Called("--subject") {
		t.Error("expected --subject flag")
	}
}

func TestGitHub_ListPRs(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr list", `[{"number":1,"title":"PR 1","state":"OPEN","url":"https://github.com/org/repo/pull/1","headRefName":"feature","baseRefName":"main","author":{"login":"alice"}}]`, nil)

	gh := NewGitHub(mock, "/repo")
	prs, err := gh.ListPRs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("expected 1 PR, got %d", len(prs))
	}
	if prs[0].Number != 1 {
		t.Errorf("expected PR number 1, got %d", prs[0].Number)
	}
	if prs[0].Author != "alice" {
		t.Errorf("expected author alice, got %s", prs[0].Author)
	}
	if prs[0].Branch != "feature" {
		t.Errorf("expected branch feature, got %s", prs[0].Branch)
	}
}

func TestGitHub_ListPRs_Empty(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr list", "", nil)

	gh := NewGitHub(mock, "/repo")
	prs, err := gh.ListPRs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prs != nil {
		t.Errorf("expected nil, got %v", prs)
	}
}

func TestGitHub_ListPRs_WithFilters(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr list", "[]", nil)

	gh := NewGitHub(mock, "/repo")
	_, err := gh.ListPRs(context.Background(), "--state", "closed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("--state") {
		t.Error("expected filter flags passed")
	}
}

func TestGitHub_GetPR(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr view", `{"number":42,"title":"My PR","state":"OPEN","url":"https://github.com/org/repo/pull/42","headRefName":"feature","baseRefName":"main","author":{"login":"bob"}}`, nil)

	gh := NewGitHub(mock, "/repo")
	pr, err := gh.GetPR(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.Number != 42 {
		t.Errorf("expected number 42, got %d", pr.Number)
	}
	if pr.Title != "My PR" {
		t.Errorf("expected title 'My PR', got %s", pr.Title)
	}
	if pr.Author != "bob" {
		t.Errorf("expected author bob, got %s", pr.Author)
	}
}

func TestGitHub_AddReviewers(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr edit", "", nil)

	gh := NewGitHub(mock, "/repo")
	err := gh.AddReviewers(context.Background(), 42, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.Called("--add-reviewer") {
		t.Error("expected --add-reviewer flag")
	}
	if !mock.Called("alice,bob") {
		t.Error("expected reviewers joined with comma")
	}
}

func TestGitHub_ListPRs_InvalidJSON(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr list", "not json", nil)

	gh := NewGitHub(mock, "/repo")
	_, err := gh.ListPRs(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestGitHub_GetPR_InvalidJSON(t *testing.T) {
	mock := NewMockRunner()
	mock.Stub("gh pr view", "not json", nil)

	gh := NewGitHub(mock, "/repo")
	_, err := gh.GetPR(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
