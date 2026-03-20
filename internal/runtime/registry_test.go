package runtime

import (
	"testing"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	rt := NewClaudeCodeRuntime(false)
	reg.Register("claude-code", rt)

	got, err := reg.Get("claude-code")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name() != "claude-code" {
		t.Errorf("expected claude-code, got %s", got.Name())
	}
}

func TestRegistry_GetUnknown(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown runtime")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register("b-runtime", &CodexRuntime{})
	reg.Register("a-runtime", NewClaudeCodeRuntime(false))

	names := reg.List()
	if len(names) != 2 {
		t.Fatalf("expected 2 runtimes, got %d", len(names))
	}
	// List should return sorted names.
	if names[0] != "a-runtime" {
		t.Errorf("expected first element 'a-runtime', got %s", names[0])
	}
	if names[1] != "b-runtime" {
		t.Errorf("expected second element 'b-runtime', got %s", names[1])
	}
}

func TestRegistry_ListEmpty(t *testing.T) {
	reg := NewRegistry()
	names := reg.List()
	if len(names) != 0 {
		t.Errorf("expected 0 runtimes, got %d", len(names))
	}
}

func TestRegistry_RegisterOverwrite(t *testing.T) {
	reg := NewRegistry()
	rt1 := NewClaudeCodeRuntime(false)
	rt2 := NewClaudeCodeRuntime(true)

	reg.Register("claude-code", rt1)
	reg.Register("claude-code", rt2)

	got, err := reg.Get("claude-code")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// Should return the most recently registered runtime.
	caps := got.Capabilities()
	if !caps.SupportsGodmode {
		t.Error("expected overwritten runtime with godmode=true")
	}
}
