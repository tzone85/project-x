package llm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CodexCLIClient implements Client by invoking `codex exec` non-interactively.
// It is used as a fallback when API-based OpenAI access is unavailable but the
// user has a working Codex CLI session.
type CodexCLIClient struct {
	cliPath string
}

// NewCodexCLIClient creates a client using the default "codex" binary.
func NewCodexCLIClient() *CodexCLIClient {
	return &CodexCLIClient{cliPath: "codex"}
}

// NewCodexCLIClientWithPath creates a client using an explicit codex path.
func NewCodexCLIClientWithPath(cliPath string) *CodexCLIClient {
	return &CodexCLIClient{cliPath: cliPath}
}

var _ Client = (*CodexCLIClient)(nil)

// Complete runs `codex exec`, writing the final model message to a temp file.
func (c *CodexCLIClient) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	outputFile, err := os.CreateTemp("", "px-codex-last-message-*.txt")
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("create codex output file: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	prompt := buildCLIPrompt(req)
	args := c.buildArgs(req, outputPath)

	cmd := exec.CommandContext(ctx, c.cliPath, args...)
	cmd.Stdin = strings.NewReader(prompt)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return CompletionResponse{}, classifyCodexCLIError(err, out)
	}

	data, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		return CompletionResponse{}, fmt.Errorf("read codex output: %w", readErr)
	}

	return CompletionResponse{
		Content: strings.TrimSpace(string(data)),
		Model:   req.Model,
	}, nil
}

func (c *CodexCLIClient) buildArgs(req CompletionRequest, outputPath string) []string {
	args := []string{
		"exec",
		"--skip-git-repo-check",
		"--sandbox",
		"read-only",
		"--color",
		"never",
		"--output-last-message",
		outputPath,
	}

	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	// Use stdin for the prompt.
	args = append(args, "-")
	return args
}

func classifyCodexCLIError(originalErr error, output []byte) error {
	text := strings.ToLower(string(output))

	switch {
	case containsAny(text, "exceeded your current quota", "check your plan and billing", "billing"):
		return &APIError{
			StatusCode: 429,
			Message:    strings.TrimSpace(string(output)),
			Retryable:  false,
		}
	case containsAny(text, "authentication", "login", "unauthorized", "not logged in"):
		return &APIError{
			StatusCode: 401,
			Message:    strings.TrimSpace(string(output)),
			Retryable:  false,
		}
	case containsAny(text, "rate limit", "too many requests"):
		return &APIError{
			StatusCode: 429,
			Message:    strings.TrimSpace(string(output)),
			Retryable:  true,
		}
	}

	return fmt.Errorf("codex CLI failed: %w: %s", originalErr, strings.TrimSpace(string(output)))
}
