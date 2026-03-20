package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/tzone85/project-x/internal/llm"
)

// mockClient implements llm.Client for testing.
type mockClient struct {
	responses []llm.CompletionResponse
	calls     [][]llm.Message
	callIndex int
}

func (m *mockClient) Complete(_ context.Context, msgs []llm.Message) (llm.CompletionResponse, error) {
	m.calls = append(m.calls, msgs)
	if m.callIndex >= len(m.responses) {
		return llm.CompletionResponse{}, fmt.Errorf("no more mock responses (call %d)", m.callIndex)
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testTechStack() TechStack {
	return TechStack{
		Language:  "Go",
		Framework: "Cobra",
		TestRunner: "go test",
		Linter:    "golangci-lint",
		BuildTool: "go build",
	}
}

func makeStoriesJSON(stories []Story) string {
	data, _ := json.Marshal(stories)
	return string(data)
}

func makeValidationJSON(result ValidationResult) string {
	data, _ := json.Marshal(result)
	return string(data)
}

func TestPlanSimpleRequirementSkipsLLMValidation(t *testing.T) {
	// Less than 3 stories → skip LLM validation
	stories := []Story{validStory("STR-1"), validStory("STR-2")}

	client := &mockClient{
		responses: []llm.CompletionResponse{
			{Content: makeStoriesJSON(stories), InputTokens: 100, OutputTokens: 50, CostUSD: 0.01},
		},
	}

	p := New(client, defaultPlanningConfig(), testLogger())
	req := Requirement{ID: "REQ-1", Title: "Simple feature", Description: "Do something simple"}

	result, err := p.Plan(context.Background(), req, testTechStack())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Stories) != 2 {
		t.Errorf("expected 2 stories, got %d", len(result.Stories))
	}
	if result.Rounds != 1 {
		t.Errorf("expected 1 round, got %d", result.Rounds)
	}
	// Only 1 LLM call (decomposition only, no validation)
	if len(client.calls) != 1 {
		t.Errorf("expected 1 LLM call for simple requirement, got %d", len(client.calls))
	}
	if result.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", result.InputTokens)
	}
}

func TestPlanWithValidDecomposition(t *testing.T) {
	// 3+ stories → triggers LLM validation
	stories := []Story{validStory("STR-1"), validStory("STR-2"), validStory("STR-3")}

	validResult := ValidationResult{Valid: true, Issues: nil, Critique: ""}

	client := &mockClient{
		responses: []llm.CompletionResponse{
			{Content: makeStoriesJSON(stories), InputTokens: 200, OutputTokens: 100},
			{Content: makeValidationJSON(validResult), InputTokens: 150, OutputTokens: 50},
		},
	}

	p := New(client, defaultPlanningConfig(), testLogger())
	req := Requirement{ID: "REQ-1", Title: "Feature", Description: "Complex feature"}

	result, err := p.Plan(context.Background(), req, testTechStack())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Stories) != 3 {
		t.Errorf("expected 3 stories, got %d", len(result.Stories))
	}
	if result.Rounds != 1 {
		t.Errorf("expected 1 round (no refinement needed), got %d", result.Rounds)
	}
	// 2 LLM calls: decompose + validate
	if len(client.calls) != 2 {
		t.Errorf("expected 2 LLM calls, got %d", len(client.calls))
	}
	if result.InputTokens != 350 {
		t.Errorf("expected 350 total input tokens, got %d", result.InputTokens)
	}
}

func TestPlanWithRefinement(t *testing.T) {
	badStories := []Story{validStory("STR-1"), validStory("STR-2"), validStory("STR-3")}
	goodStories := []Story{validStory("STR-1"), validStory("STR-2"), validStory("STR-3")}

	failResult := ValidationResult{
		Valid:    false,
		Issues:   []ValidationIssue{{StoryID: "STR-1", Field: "title", Message: "too vague", Severity: "error"}},
		Critique: "Story titles need more detail",
	}
	passResult := ValidationResult{Valid: true, Issues: nil}

	client := &mockClient{
		responses: []llm.CompletionResponse{
			{Content: makeStoriesJSON(badStories), InputTokens: 200, OutputTokens: 100},   // decompose
			{Content: makeValidationJSON(failResult), InputTokens: 150, OutputTokens: 50},  // validate (fail)
			{Content: makeStoriesJSON(goodStories), InputTokens: 220, OutputTokens: 110},   // re-decompose
			{Content: makeValidationJSON(passResult), InputTokens: 160, OutputTokens: 60},  // validate (pass)
		},
	}

	p := New(client, defaultPlanningConfig(), testLogger())
	req := Requirement{ID: "REQ-1", Title: "Feature", Description: "Complex feature"}

	result, err := p.Plan(context.Background(), req, testTechStack())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Rounds != 2 {
		t.Errorf("expected 2 rounds, got %d", result.Rounds)
	}
	// 4 LLM calls: decompose + validate + re-decompose + validate
	if len(client.calls) != 4 {
		t.Errorf("expected 4 LLM calls, got %d", len(client.calls))
	}
	if result.Validation.Valid != true {
		t.Error("expected final validation to be valid after refinement")
	}
}

func TestPlanMaxRefinementRounds(t *testing.T) {
	stories := []Story{validStory("STR-1"), validStory("STR-2"), validStory("STR-3")}
	failResult := ValidationResult{
		Valid:    false,
		Issues:   []ValidationIssue{{StoryID: "STR-1", Field: "title", Message: "bad", Severity: "error"}},
		Critique: "still bad",
	}

	client := &mockClient{
		responses: []llm.CompletionResponse{
			{Content: makeStoriesJSON(stories)},     // decompose round 1
			{Content: makeValidationJSON(failResult)}, // validate round 1 (fail)
			{Content: makeStoriesJSON(stories)},     // decompose round 2
			{Content: makeValidationJSON(failResult)}, // validate round 2 (still fail)
		},
	}

	p := New(client, defaultPlanningConfig(), testLogger())
	req := Requirement{ID: "REQ-1", Title: "Feature", Description: "Something"}

	result, err := p.Plan(context.Background(), req, testTechStack())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Rounds != 2 {
		t.Errorf("expected max 2 rounds, got %d", result.Rounds)
	}
	// Should proceed with quality warnings even though validation failed
	if len(result.QualityWarnings) == 0 {
		t.Error("expected quality warnings when max rounds reached with failures")
	}
	if len(result.Stories) != 3 {
		t.Errorf("expected stories even on validation failure, got %d", len(result.Stories))
	}
}

func TestPlanDecompositionFailure(t *testing.T) {
	client := &mockClient{
		responses: []llm.CompletionResponse{
			{Content: "not valid json"},
		},
	}

	p := New(client, defaultPlanningConfig(), testLogger())
	req := Requirement{ID: "REQ-1", Title: "Feature", Description: "Something"}

	_, err := p.Plan(context.Background(), req, testTechStack())
	if err == nil {
		t.Error("expected error for invalid decomposition response")
	}
}

func TestPlanLLMValidationFailureFallsBackToStructural(t *testing.T) {
	stories := []Story{validStory("STR-1"), validStory("STR-2"), validStory("STR-3")}

	client := &mockClient{
		responses: []llm.CompletionResponse{
			{Content: makeStoriesJSON(stories), InputTokens: 200, OutputTokens: 100},
			{Content: "not valid json"}, // LLM validation returns garbage
		},
	}

	p := New(client, defaultPlanningConfig(), testLogger())
	req := Requirement{ID: "REQ-1", Title: "Feature", Description: "Something"}

	result, err := p.Plan(context.Background(), req, testTechStack())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Stories) != 3 {
		t.Errorf("expected 3 stories despite validation failure, got %d", len(result.Stories))
	}
	// Should have warning about LLM validation skip
	hasWarning := false
	for _, w := range result.QualityWarnings {
		if w == "LLM validation was skipped due to error" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Errorf("expected LLM validation skip warning, got warnings: %v", result.QualityWarnings)
	}
}

func TestPlanTokenAccumulation(t *testing.T) {
	stories := []Story{validStory("STR-1"), validStory("STR-2"), validStory("STR-3")}
	validResult := ValidationResult{Valid: true}

	client := &mockClient{
		responses: []llm.CompletionResponse{
			{Content: makeStoriesJSON(stories), InputTokens: 100, OutputTokens: 50, CostUSD: 0.01},
			{Content: makeValidationJSON(validResult), InputTokens: 80, OutputTokens: 30, CostUSD: 0.005},
		},
	}

	p := New(client, defaultPlanningConfig(), testLogger())
	req := Requirement{ID: "REQ-1", Title: "Feature", Description: "Something"}

	result, err := p.Plan(context.Background(), req, testTechStack())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.InputTokens != 180 {
		t.Errorf("expected 180 input tokens, got %d", result.InputTokens)
	}
	if result.OutputTokens != 80 {
		t.Errorf("expected 80 output tokens, got %d", result.OutputTokens)
	}
	if result.CostUSD != 0.015 {
		t.Errorf("expected $0.015 cost, got $%f", result.CostUSD)
	}
}

func TestPlanStructuralValidationAlwaysRuns(t *testing.T) {
	// Stories with file ownership conflict should be caught even for simple requirements
	s1 := validStory("STR-1")
	s2 := validStory("STR-2")
	s2.OwnedFiles = s1.OwnedFiles // conflict

	client := &mockClient{
		responses: []llm.CompletionResponse{
			{Content: makeStoriesJSON([]Story{s1, s2})},
		},
	}

	p := New(client, defaultPlanningConfig(), testLogger())
	req := Requirement{ID: "REQ-1", Title: "Simple", Description: "Something"}

	result, err := p.Plan(context.Background(), req, testTechStack())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Validation.Valid {
		t.Error("expected structural validation to catch file ownership conflict")
	}
}

func TestPlanContextCancellation(t *testing.T) {
	client := &mockClient{
		responses: nil, // will cause error on call
	}

	p := New(client, defaultPlanningConfig(), testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	req := Requirement{ID: "REQ-1", Title: "Feature", Description: "Something"}
	_, err := p.Plan(ctx, req, testTechStack())
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
}
