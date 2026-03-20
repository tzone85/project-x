package llm

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestBuildCLIPrompt_SystemAndMessages(t *testing.T) {
	req := CompletionRequest{
		System: "You are a helpful assistant.",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
			{Role: RoleAssistant, Content: "Hi there"},
			{Role: RoleUser, Content: "How are you?"},
		},
	}
	prompt := buildCLIPrompt(req)
	if !strings.Contains(prompt, "You are a helpful assistant.") {
		t.Error("missing system prompt")
	}
	if !strings.Contains(prompt, "Hello") {
		t.Error("missing first user message")
	}
	if !strings.Contains(prompt, "How are you?") {
		t.Error("missing second user message")
	}
	// Assistant messages should NOT be included (CLI is single-turn)
	if strings.Contains(prompt, "Hi there") {
		t.Error("assistant message should not be in CLI prompt")
	}
}

func TestBuildCLIPrompt_NoSystem(t *testing.T) {
	req := CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}
	prompt := buildCLIPrompt(req)
	if !strings.Contains(prompt, "Hello") {
		t.Error("missing user message")
	}
}

func TestBuildCLIPrompt_EmptyRequest(t *testing.T) {
	req := CompletionRequest{}
	prompt := buildCLIPrompt(req)
	if prompt != "" {
		t.Errorf("expected empty prompt for empty request, got %q", prompt)
	}
}

func TestBuildCLIPrompt_SystemOnly(t *testing.T) {
	req := CompletionRequest{
		System: "Be concise.",
	}
	prompt := buildCLIPrompt(req)
	if !strings.Contains(prompt, "Be concise.") {
		t.Error("missing system prompt")
	}
}

func TestBuildCLIPrompt_MultipleUserMessages(t *testing.T) {
	req := CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "First"},
			{Role: RoleUser, Content: "Second"},
		},
	}
	prompt := buildCLIPrompt(req)
	if !strings.Contains(prompt, "First") {
		t.Error("missing first user message")
	}
	if !strings.Contains(prompt, "Second") {
		t.Error("missing second user message")
	}
}

func TestTrimCodeFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no fences", "hello world", "hello world"},
		{"json fences", "```json\n{\"key\": \"val\"}\n```", "{\"key\": \"val\"}"},
		{"plain fences", "```\nsome code\n```", "some code"},
		{"no closing fence", "```json\n{\"key\": \"val\"}", "{\"key\": \"val\"}"},
		{"empty", "", ""},
		{"only opening fence", "```\n", ""},
		{"fence with trailing whitespace", "```json\n{\"a\": 1}\n```\n", "{\"a\": 1}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimCodeFences(tt.input)
			if got != tt.want {
				t.Errorf("trimCodeFences(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestClassifyCLIError_BillingExhaustion(t *testing.T) {
	err := classifyCLIError(fmt.Errorf("exit 1"), []byte("Error: credit balance exhausted"))
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if apiErr.Retryable {
		t.Error("billing error should not be retryable")
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestClassifyCLIError_AuthFailure(t *testing.T) {
	err := classifyCLIError(fmt.Errorf("exit 1"), []byte("Error: authentication failed"))
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if apiErr.Retryable {
		t.Error("auth error should not be retryable")
	}
}

func TestClassifyCLIError_RateLimit(t *testing.T) {
	err := classifyCLIError(fmt.Errorf("exit 1"), []byte("Error: rate limit exceeded, too many requests"))
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if !apiErr.Retryable {
		t.Error("rate limit should be retryable")
	}
}

func TestClassifyCLIError_GenericError(t *testing.T) {
	err := classifyCLIError(fmt.Errorf("exit 1"), []byte("some unknown error"))
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Error("generic error should not be APIError")
	}
}

func TestClassifyCLIError_OverloadedError(t *testing.T) {
	err := classifyCLIError(fmt.Errorf("exit 1"), []byte("Error: server overloaded"))
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError")
	}
	if !apiErr.Retryable {
		t.Error("overloaded error should be retryable")
	}
	if apiErr.StatusCode != 529 {
		t.Errorf("expected status 529, got %d", apiErr.StatusCode)
	}
}

func TestNewClaudeCLIClient(t *testing.T) {
	c := NewClaudeCLIClient()
	if c.cliPath != "claude" {
		t.Errorf("expected default CLI path 'claude', got %q", c.cliPath)
	}
	if c.skipPerms {
		t.Error("skipPerms should default to false")
	}
}

func TestNewClaudeCLIClientWithPath(t *testing.T) {
	c := NewClaudeCLIClientWithPath("/usr/local/bin/claude")
	if c.cliPath != "/usr/local/bin/claude" {
		t.Errorf("expected custom CLI path, got %q", c.cliPath)
	}
}

func TestClaudeCLIClient_WithSkipPermissions(t *testing.T) {
	original := NewClaudeCLIClient()
	withSkip := original.WithSkipPermissions()

	// Original should be unchanged (immutability)
	if original.skipPerms {
		t.Error("original should not be modified")
	}
	if !withSkip.skipPerms {
		t.Error("copy should have skipPerms enabled")
	}
	if withSkip.cliPath != original.cliPath {
		t.Error("copy should preserve cliPath")
	}
}
