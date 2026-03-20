package state

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// EventStore is an append-only JSONL event log. It is the source of truth
// for the entire system. SQLite projections are derived materialized views
// that can be rebuilt by replaying this log.
type EventStore struct {
	mu   sync.Mutex
	path string
}

// NewEventStore creates an EventStore backed by the given file path.
// The file is created if it does not exist.
func NewEventStore(path string) (*EventStore, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open event store: %w", err)
	}
	f.Close()

	return &EventStore{path: path}, nil
}

// Append writes an event to the JSONL log. The event is validated against
// the payload registry before writing.
func (es *EventStore) Append(event Event) error {
	if err := ValidatePayload(event); err != nil {
		return fmt.Errorf("append: %w", err)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("append marshal: %w", err)
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	f, err := os.OpenFile(es.path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("append open: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append write: %w", err)
	}

	return nil
}

// Replay reads all events from the JSONL log and calls the handler for each.
// Events are replayed in order. If the handler returns an error, replay stops.
func (es *EventStore) Replay(handler func(Event) error) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	return es.replayUnlocked(handler)
}

// replayUnlocked reads events without holding the mutex (caller must hold it).
func (es *EventStore) replayUnlocked(handler func(Event) error) error {
	f, err := os.Open(es.path)
	if err != nil {
		return fmt.Errorf("replay open: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB max line

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			return fmt.Errorf("replay line %d: unmarshal: %w", lineNum, err)
		}

		if err := handler(event); err != nil {
			return fmt.Errorf("replay line %d: handler: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("replay scan: %w", err)
	}

	return nil
}

// Count returns the number of events in the log.
func (es *EventStore) Count() (int, error) {
	count := 0
	err := es.Replay(func(_ Event) error {
		count++
		return nil
	})
	return count, err
}

// Path returns the file path of the event store.
func (es *EventStore) Path() string {
	return es.path
}
