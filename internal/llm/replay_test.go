package llm

import (
	"context"
	"testing"
)

func TestReplayClient_EmptyResponses(t *testing.T) {
	client := NewReplayClient()
	resp, err := client.Complete(context.Background(), CompletionRequest{})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("Content = %q, want empty", resp.Content)
	}
}

func TestReplayClient_SingleResponse(t *testing.T) {
	client := NewReplayClient(CompletionResponse{
		Content: "hello",
		Model:   "test-model",
	})

	resp, err := client.Complete(context.Background(), CompletionRequest{})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("Content = %q, want %q", resp.Content, "hello")
	}
	if resp.Model != "test-model" {
		t.Errorf("Model = %q, want %q", resp.Model, "test-model")
	}
}

func TestReplayClient_CyclesResponses(t *testing.T) {
	client := NewReplayClient(
		CompletionResponse{Content: "a"},
		CompletionResponse{Content: "b"},
		CompletionResponse{Content: "c"},
	)

	expected := []string{"a", "b", "c", "a", "b"}
	for i, want := range expected {
		resp, err := client.Complete(context.Background(), CompletionRequest{})
		if err != nil {
			t.Fatalf("Complete[%d]: %v", i, err)
		}
		if resp.Content != want {
			t.Errorf("Complete[%d] = %q, want %q", i, resp.Content, want)
		}
	}
}

func TestReplayClient_IgnoresRequest(t *testing.T) {
	client := NewReplayClient(CompletionResponse{Content: "fixed"})

	req := CompletionRequest{
		System:    "you are a test",
		Messages:  []Message{{Role: RoleUser, Content: "ignored"}},
		Model:     "gpt-4",
		MaxTokens: 100,
	}

	resp, err := client.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "fixed" {
		t.Errorf("Content = %q, want %q", resp.Content, "fixed")
	}
}

func TestReplayClient_DoesNotMutateInput(t *testing.T) {
	original := []CompletionResponse{
		{Content: "a"},
		{Content: "b"},
	}
	client := NewReplayClient(original...)

	// Mutate original after creation.
	original[0].Content = "mutated"

	resp, _ := client.Complete(context.Background(), CompletionRequest{})
	if resp.Content != "a" {
		t.Errorf("Content = %q, want %q (should not reflect external mutation)", resp.Content, "a")
	}
}
