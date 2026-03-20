package cli

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const shutdownTimeout = 30 * time.Second

// ShutdownHook is a named cleanup function run during graceful shutdown.
type ShutdownHook struct {
	Name string
	Fn   func(ctx context.Context) error
}

// ShutdownManager handles graceful shutdown on SIGINT/SIGTERM.
type ShutdownManager struct {
	cancel context.CancelFunc
	logger *slog.Logger

	mu    sync.Mutex
	hooks []ShutdownHook
}

// NewShutdownManager creates a shutdown manager that cancels the given context
// on SIGINT/SIGTERM and runs registered hooks.
func NewShutdownManager(cancel context.CancelFunc, logger *slog.Logger) *ShutdownManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &ShutdownManager{
		cancel: cancel,
		logger: logger.With("component", "shutdown"),
	}
}

// Register adds a cleanup hook to run during shutdown.
// Hooks run in registration order.
func (sm *ShutdownManager) Register(hook ShutdownHook) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.hooks = append(sm.hooks, hook)
}

// Listen blocks until a shutdown signal is received, then runs hooks.
// Returns after all hooks complete or timeout.
func (sm *ShutdownManager) Listen() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	sm.logger.Info("shutdown signal received", "signal", sig.String())

	// 1. Cancel context (stops polling, no new pipeline stages)
	sm.cancel()

	// 2. Run hooks with timeout
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	sm.runHooks(ctx)

	sm.logger.Info("shutdown complete")
}

// Shutdown triggers shutdown programmatically (for testing).
func (sm *ShutdownManager) Shutdown() {
	sm.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	sm.runHooks(ctx)
}

func (sm *ShutdownManager) runHooks(ctx context.Context) {
	sm.mu.Lock()
	hooks := make([]ShutdownHook, len(sm.hooks))
	copy(hooks, sm.hooks)
	sm.mu.Unlock()

	for _, hook := range hooks {
		sm.logger.Info("running shutdown hook", "name", hook.Name)
		if err := hook.Fn(ctx); err != nil {
			sm.logger.Error("shutdown hook failed", "name", hook.Name, "error", err)
		}
	}
}
