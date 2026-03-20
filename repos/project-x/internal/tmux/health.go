package tmux

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"
)

// HealthStatus represents the health state of a tmux session.
type HealthStatus string

const (
	HealthHealthy HealthStatus = "healthy"
	HealthStale   HealthStatus = "stale"
	HealthDead    HealthStatus = "dead"
	HealthMissing HealthStatus = "missing"
)

// HealthResult contains the result of a health check.
type HealthResult struct {
	Status     HealthStatus
	SessionName string
	PanePID    string
	ExitCode   string
	LastCheck  time.Time
}

// HealthConfig configures health monitoring thresholds.
type HealthConfig struct {
	StaleThreshold      time.Duration
	MaxRecoveryAttempts int
	OnDead              string // "redispatch" or "pause"
	OnStale             string // "restart" or "kill"
}

// DefaultHealthConfig returns sensible defaults for health monitoring.
func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		StaleThreshold:      180 * time.Second,
		MaxRecoveryAttempts: 2,
		OnDead:              "redispatch",
		OnStale:             "restart",
	}
}

// HealthMonitor checks session health by tracking output changes.
type HealthMonitor struct {
	session *Session
	config  HealthConfig

	mu           sync.Mutex
	outputHashes map[string]string    // session → last output hash
	lastChanged  map[string]time.Time // session → last time output changed
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(session *Session, config HealthConfig) *HealthMonitor {
	return &HealthMonitor{
		session:      session,
		config:       config,
		outputHashes: make(map[string]string),
		lastChanged:  make(map[string]time.Time),
	}
}

// SessionHealth checks the health of a named tmux session.
func (hm *HealthMonitor) SessionHealth(ctx context.Context, name string) HealthResult {
	now := time.Now()
	result := HealthResult{
		SessionName: name,
		LastCheck:   now,
	}

	// Check session exists
	exists, err := hm.session.SessionExists(ctx, name)
	if err != nil || !exists {
		result.Status = HealthMissing
		return result
	}

	// Check pane process status
	paneInfo, err := hm.session.runner.Run(ctx, "list-panes", "-t", name,
		"-F", "#{pane_pid} #{pane_dead} #{pane_dead_status}")
	if err != nil {
		result.Status = HealthDead
		return result
	}

	parts := strings.Fields(paneInfo)
	if len(parts) >= 2 {
		result.PanePID = parts[0]
		isDead := parts[1]
		if isDead == "1" {
			result.Status = HealthDead
			if len(parts) >= 3 {
				result.ExitCode = parts[2]
			}
			return result
		}
	}

	// Check output flow
	output, err := hm.session.CaptureOutput(ctx, name)
	if err != nil {
		result.Status = HealthStale
		return result
	}

	hash := hashOutput(output)

	hm.mu.Lock()
	defer hm.mu.Unlock()

	prevHash, hasPrev := hm.outputHashes[name]
	hm.outputHashes[name] = hash

	if !hasPrev || hash != prevHash {
		hm.lastChanged[name] = now
		result.Status = HealthHealthy
		return result
	}

	// Output hasn't changed — check staleness
	lastChange, hasLastChange := hm.lastChanged[name]
	if !hasLastChange {
		hm.lastChanged[name] = now
		result.Status = HealthHealthy
		return result
	}

	if now.Sub(lastChange) > hm.config.StaleThreshold {
		result.Status = HealthStale
	} else {
		result.Status = HealthHealthy
	}

	return result
}

// RecoveryAttempts tracks recovery attempts per session.
type RecoveryTracker struct {
	mu       sync.Mutex
	attempts map[string]int
	maxRetry int
}

// NewRecoveryTracker creates a new recovery tracker.
func NewRecoveryTracker(maxAttempts int) *RecoveryTracker {
	return &RecoveryTracker{
		attempts: make(map[string]int),
		maxRetry: maxAttempts,
	}
}

// CanRecover returns true if the session hasn't exceeded max recovery attempts.
func (rt *RecoveryTracker) CanRecover(sessionName string) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.attempts[sessionName] < rt.maxRetry
}

// RecordAttempt increments the recovery attempt counter.
func (rt *RecoveryTracker) RecordAttempt(sessionName string) int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.attempts[sessionName]++
	return rt.attempts[sessionName]
}

// Reset clears the recovery counter for a session.
func (rt *RecoveryTracker) Reset(sessionName string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.attempts, sessionName)
}

func hashOutput(output string) string {
	h := sha256.Sum256([]byte(output))
	return fmt.Sprintf("%x", h[:8])
}
