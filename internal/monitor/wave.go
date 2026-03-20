package monitor

import (
	"log/slog"

	"github.com/tzone85/project-x/internal/state"
)

// WaveChecker determines if all stories in a wave are complete
// and identifies stories ready for the next wave.
type WaveChecker struct {
	waves  WaveTracker
	logger *slog.Logger
}

// NewWaveChecker creates a WaveChecker.
func NewWaveChecker(waves WaveTracker, logger *slog.Logger) *WaveChecker {
	if logger == nil {
		logger = slog.Default()
	}
	return &WaveChecker{waves: waves, logger: logger}
}

// CheckCompletion returns true if all stories in the given wave for a requirement
// are done (status "merged" or "done").
func (wc *WaveChecker) CheckCompletion(reqID string, wave int) bool {
	stories, err := wc.waves.ListStoriesByRequirement(reqID, state.PageParams{Limit: 1000})
	if err != nil {
		wc.logger.Error("failed to list stories for wave check",
			"req_id", reqID, "error", err)
		return false
	}

	for _, s := range stories {
		if s.Wave == wave && s.Status != "merged" && s.Status != "done" {
			return false
		}
	}
	return true
}

// NextWaveStories returns stories in the next wave that are ready for dispatch.
func (wc *WaveChecker) NextWaveStories(reqID string, completedWave int) []state.Story {
	stories, err := wc.waves.ListStoriesByRequirement(reqID, state.PageParams{Limit: 1000})
	if err != nil {
		wc.logger.Error("failed to list stories for next wave",
			"req_id", reqID, "error", err)
		return nil
	}

	nextWave := completedWave + 1
	var ready []state.Story
	for _, s := range stories {
		if s.Wave == nextWave && s.Status == "planned" {
			ready = append(ready, s)
		}
	}
	return ready
}
