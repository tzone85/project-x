package state

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestProjector_EnqueueAndProcess(t *testing.T) {
	ps := newTestProjectionStore(t)

	projector := NewProjector(ps, WithChannelSize(10), WithBatchSize(5), WithFlushInterval(50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	projector.Start(ctx)

	// Enqueue a requirement event
	event, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Test Req", Description: "Desc", Source: "test",
	})
	if !projector.Enqueue(event) {
		t.Fatal("enqueue failed")
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify projected
	req, err := ps.GetRequirement("req-1")
	if err != nil {
		t.Fatalf("GetRequirement: %v", err)
	}
	if req == nil {
		t.Fatal("requirement not projected")
	}
	if req.Title != "Test Req" {
		t.Errorf("expected title 'Test Req', got %s", req.Title)
	}

	projector.Stop()
}

func TestProjector_BatchProcessing(t *testing.T) {
	ps := newTestProjectionStore(t)

	projector := NewProjector(ps,
		WithChannelSize(100),
		WithBatchSize(10),
		WithFlushInterval(1*time.Second), // long interval to force batch-size trigger
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	projector.Start(ctx)

	// Enqueue 10 events (should trigger a batch flush)
	for i := 0; i < 10; i++ {
		event, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
			RequirementID: "req-" + string(rune('a'+i)),
			Title:         "Req",
			Description:   "", Source: "test",
		})
		projector.Enqueue(event)
	}

	// Wait for batch to process
	time.Sleep(200 * time.Millisecond)

	reqs, err := ps.ListRequirements(DefaultPageParams())
	if err != nil {
		t.Fatalf("ListRequirements: %v", err)
	}
	if len(reqs) != 10 {
		t.Errorf("expected 10 requirements, got %d", len(reqs))
	}

	projector.Stop()
}

func TestProjector_DrainOnStop(t *testing.T) {
	ps := newTestProjectionStore(t)

	projector := NewProjector(ps,
		WithChannelSize(100),
		WithBatchSize(50),
		WithFlushInterval(10*time.Second), // Very long — only drains on stop
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	projector.Start(ctx)

	// Enqueue events
	for i := 0; i < 5; i++ {
		event, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
			RequirementID: "req-" + string(rune('a'+i)),
			Title:         "Req",
			Description:   "", Source: "test",
		})
		projector.Enqueue(event)
	}

	// Stop should drain
	projector.Stop()

	reqs, err := ps.ListRequirements(DefaultPageParams())
	if err != nil {
		t.Fatalf("ListRequirements: %v", err)
	}
	if len(reqs) != 5 {
		t.Errorf("expected 5 requirements after drain, got %d", len(reqs))
	}
}

func TestProjector_ContextCancellation(t *testing.T) {
	ps := newTestProjectionStore(t)

	projector := NewProjector(ps,
		WithChannelSize(100),
		WithBatchSize(50),
		WithFlushInterval(10*time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())
	projector.Start(ctx)

	// Enqueue events
	for i := 0; i < 3; i++ {
		event, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
			RequirementID: "req-" + string(rune('a'+i)),
			Title:         "Req",
			Description:   "", Source: "test",
		})
		projector.Enqueue(event)
	}

	// Give events time to be sent
	time.Sleep(50 * time.Millisecond)

	// Cancel context should drain
	cancel()
	projector.Stop()

	reqs, err := ps.ListRequirements(DefaultPageParams())
	if err != nil {
		t.Fatalf("ListRequirements: %v", err)
	}
	if len(reqs) != 3 {
		t.Errorf("expected 3 requirements after ctx cancel drain, got %d", len(reqs))
	}
}

func TestProjector_Pending(t *testing.T) {
	ps := newTestProjectionStore(t)

	projector := NewProjector(ps,
		WithChannelSize(100),
		WithBatchSize(50),
		WithFlushInterval(10*time.Second),
	)
	// Don't start — events stay in channel

	event, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Req", Description: "", Source: "test",
	})
	projector.Enqueue(event)

	if projector.Pending() != 1 {
		t.Errorf("expected 1 pending, got %d", projector.Pending())
	}
}

func TestProjector_FullChannelDropsEvent(t *testing.T) {
	ps := newTestProjectionStore(t)

	projector := NewProjector(ps, WithChannelSize(1))
	// Don't start — channel will fill up

	event1, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Req", Description: "", Source: "test",
	})
	event2, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-2", Title: "Req", Description: "", Source: "test",
	})

	if !projector.Enqueue(event1) {
		t.Error("first enqueue should succeed")
	}

	// Second enqueue should fail (channel full)
	if projector.Enqueue(event2) {
		t.Error("second enqueue should fail when channel is full")
	}
}
