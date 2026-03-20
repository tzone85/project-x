package state

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Compile-time check: FileStore must implement EventStore.
var _ EventStore = (*FileStore)(nil)

// FileStore implements EventStore as an append-only JSONL file.
// Events are loaded into memory on open for fast filtering.
// Thread-safe via sync.Mutex.
type FileStore struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	events []Event
}

// NewFileStore creates a new FileStore backed by the given JSONL file path.
// It creates parent directories if needed, opens (or creates) the file,
// and loads any existing events into memory.
func NewFileStore(path string) (*FileStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("filestore: create parent dirs: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("filestore: open file: %w", err)
	}

	events, err := loadEvents(path)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("filestore: load events: %w", err)
	}

	return &FileStore{
		path:   path,
		file:   file,
		events: events,
	}, nil
}

// loadEvents reads all JSONL lines from the file and deserializes them.
func loadEvents(path string) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)

	// Increase buffer for potentially large payloads.
	const maxLineSize = 1024 * 1024 // 1 MB
	scanner.Buffer(make([]byte, 0, maxLineSize), maxLineSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var evt Event
		if err := json.Unmarshal(line, &evt); err != nil {
			return nil, fmt.Errorf("unmarshal event line: %w", err)
		}
		events = append(events, evt)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	return events, nil
}

// Append writes an event as a JSON line to disk and adds it to the in-memory slice.
// The write is flushed/synced to ensure durability.
func (fs *FileStore) Append(evt Event) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("filestore: marshal event: %w", err)
	}

	line := append(data, '\n')
	if _, err := fs.file.Write(line); err != nil {
		return fmt.Errorf("filestore: write event: %w", err)
	}

	if err := fs.file.Sync(); err != nil {
		return fmt.Errorf("filestore: sync file: %w", err)
	}

	fs.events = append(fs.events, evt)
	return nil
}

// List returns events matching the given filter.
// When Limit is set, it returns the last N matching events.
// When After is set, only events with a timestamp strictly after the cursor are returned.
func (fs *FileStore) List(filter EventFilter) ([]Event, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var matched []Event
	for _, evt := range fs.events {
		if matchesFilter(evt, filter) {
			matched = append(matched, evt)
		}
	}

	if filter.Limit > 0 && len(matched) > filter.Limit {
		matched = matched[len(matched)-filter.Limit:]
	}

	return matched, nil
}

// Count returns the number of events matching the given filter.
func (fs *FileStore) Count(filter EventFilter) (int, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	count := 0
	for _, evt := range fs.events {
		if matchesFilter(evt, filter) {
			count++
		}
	}

	return count, nil
}

// All returns a copy of all events in the store.
func (fs *FileStore) All() ([]Event, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	result := make([]Event, len(fs.events))
	copy(result, fs.events)
	return result, nil
}

// Close closes the underlying file handle.
func (fs *FileStore) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.file != nil {
		return fs.file.Close()
	}
	return nil
}

// matchesFilter checks whether an event satisfies all non-zero fields of the filter.
func matchesFilter(evt Event, filter EventFilter) bool {
	if filter.Type != "" && evt.Type != filter.Type {
		return false
	}
	if filter.AgentID != "" && evt.AgentID != filter.AgentID {
		return false
	}
	if filter.StoryID != "" && evt.StoryID != filter.StoryID {
		return false
	}
	if filter.After != "" && evt.Timestamp <= filter.After {
		return false
	}
	return true
}
