package state

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	defaultChannelSize = 256
	defaultBatchSize   = 50
	defaultFlushInterval = 100 * time.Millisecond
)

// Projector is the async projection goroutine that drains events from a
// buffered channel and applies them to the ProjectionStore in batches.
// It decouples event emission from projection writes.
type Projector struct {
	store   *ProjectionStore
	eventCh chan Event
	logger  *slog.Logger

	batchSize     int
	flushInterval time.Duration

	wg   sync.WaitGroup
	done chan struct{}
}

// ProjectorOption configures the Projector.
type ProjectorOption func(*Projector)

// WithChannelSize sets the buffered channel size.
func WithChannelSize(size int) ProjectorOption {
	return func(p *Projector) {
		p.eventCh = make(chan Event, size)
	}
}

// WithBatchSize sets the maximum batch size for projection writes.
func WithBatchSize(size int) ProjectorOption {
	return func(p *Projector) {
		p.batchSize = size
	}
}

// WithFlushInterval sets the interval for flushing partial batches.
func WithFlushInterval(d time.Duration) ProjectorOption {
	return func(p *Projector) {
		p.flushInterval = d
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) ProjectorOption {
	return func(p *Projector) {
		p.logger = logger
	}
}

// NewProjector creates a new async projector. Call Start to begin processing.
func NewProjector(store *ProjectionStore, opts ...ProjectorOption) *Projector {
	p := &Projector{
		store:         store,
		eventCh:       make(chan Event, defaultChannelSize),
		logger:        slog.Default(),
		batchSize:     defaultBatchSize,
		flushInterval: defaultFlushInterval,
		done:          make(chan struct{}),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Enqueue sends an event to the projection channel. It does not block if
// the channel is full — it logs a warning and drops the event. In practice,
// the channel should be sized large enough that this never happens.
func (p *Projector) Enqueue(event Event) bool {
	select {
	case p.eventCh <- event:
		return true
	default:
		p.logger.Warn("projection channel full, event dropped",
			"event_id", event.ID,
			"event_type", event.Type,
		)
		return false
	}
}

// Start launches the background goroutine that drains and applies projections.
func (p *Projector) Start(ctx context.Context) {
	p.wg.Add(1)
	go p.run(ctx)
}

// Stop signals the projector to drain remaining events and stop.
// It blocks until all pending events have been processed.
func (p *Projector) Stop() {
	close(p.done)
	p.wg.Wait()
}

// Pending returns the number of events waiting to be projected.
func (p *Projector) Pending() int {
	return len(p.eventCh)
}

func (p *Projector) run(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	batch := make([]Event, 0, p.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		p.applyBatch(batch)
		batch = batch[:0]
	}

	for {
		select {
		case event, ok := <-p.eventCh:
			if !ok {
				flush()
				return
			}
			batch = append(batch, event)
			if len(batch) >= p.batchSize {
				flush()
			}

		case <-ticker.C:
			flush()

		case <-p.done:
			// Drain remaining events from the channel
			p.drain(batch)
			return

		case <-ctx.Done():
			// Drain remaining events from the channel
			p.drain(batch)
			return
		}
	}
}

func (p *Projector) drain(batch []Event) {
	// Process any events already in the batch
	p.applyBatch(batch)

	// Drain the channel
	for {
		select {
		case event, ok := <-p.eventCh:
			if !ok {
				return
			}
			p.applyBatch([]Event{event})
		default:
			return
		}
	}
}

func (p *Projector) applyBatch(events []Event) {
	for _, event := range events {
		if err := p.store.ApplyEvent(event); err != nil {
			p.logger.Error("failed to apply projection",
				"event_id", event.ID,
				"event_type", event.Type,
				"error", err,
			)
		}
	}
}
