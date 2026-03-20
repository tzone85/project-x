package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewSSEHub(t *testing.T) {
	hub := NewSSEHub()
	if hub == nil {
		t.Fatal("NewSSEHub returned nil")
	}
	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount = %d, want 0", hub.ClientCount())
	}
}

func TestSSEHub_Broadcast_NoClients(t *testing.T) {
	hub := NewSSEHub()
	// Should not panic with no clients.
	hub.Broadcast("test", `{"msg":"hello"}`)
}

func TestSSEHub_ClientCount_Empty(t *testing.T) {
	hub := NewSSEHub()
	if got := hub.ClientCount(); got != 0 {
		t.Errorf("ClientCount = %d, want 0", got)
	}
}

// flushRecorder wraps httptest.ResponseRecorder to implement http.Flusher.
type flushRecorder struct {
	*httptest.ResponseRecorder
	flushed int
}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
}

func (f *flushRecorder) Flush() {
	f.flushed++
}

func TestSSEHub_ServeHTTP_SendsConnectedEvent(t *testing.T) {
	hub := NewSSEHub()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/stream", nil).WithContext(ctx)
	w := newFlushRecorder()

	done := make(chan struct{})
	go func() {
		hub.ServeHTTP(w, req)
		close(done)
	}()

	// Give the handler time to start and send the connected event.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := w.Body.String()
	if !strings.Contains(body, `data: {"type":"connected"}`) {
		t.Errorf("expected connected event, got: %q", body)
	}

	// Verify SSE headers.
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
}

func TestSSEHub_ServeHTTP_StreamsMessages(t *testing.T) {
	hub := NewSSEHub()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/stream", nil).WithContext(ctx)
	w := newFlushRecorder()

	done := make(chan struct{})
	go func() {
		hub.ServeHTTP(w, req)
		close(done)
	}()

	// Wait for client to register.
	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Fatalf("ClientCount = %d, want 1", hub.ClientCount())
	}

	hub.Broadcast("update", `{"status":"working"}`)

	// Give broadcast time to propagate.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := w.Body.String()
	if !strings.Contains(body, `"type":"update"`) {
		t.Errorf("expected update event in body, got: %q", body)
	}
	if !strings.Contains(body, `"status":"working"`) {
		t.Errorf("expected status data in body, got: %q", body)
	}
}

func TestSSEHub_CleansUpOnDisconnect(t *testing.T) {
	hub := NewSSEHub()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/stream", nil).WithContext(ctx)
	w := newFlushRecorder()

	done := make(chan struct{})
	go func() {
		hub.ServeHTTP(w, req)
		close(done)
	}()

	// Wait for connection.
	time.Sleep(50 * time.Millisecond)
	if hub.ClientCount() != 1 {
		t.Fatalf("ClientCount after connect = %d, want 1", hub.ClientCount())
	}

	cancel()
	<-done

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount after disconnect = %d, want 0", hub.ClientCount())
	}
}

func TestSSEHub_Broadcast_ConcurrentSafe(t *testing.T) {
	hub := NewSSEHub()

	// Spawn multiple clients.
	const numClients = 5
	var wg sync.WaitGroup
	ctxs := make([]context.CancelFunc, numClients)

	for i := 0; i < numClients; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		ctxs[i] = cancel
		req := httptest.NewRequest("GET", "/api/stream", nil).WithContext(ctx)
		w := newFlushRecorder()

		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.ServeHTTP(w, req)
		}()
	}

	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != numClients {
		t.Fatalf("ClientCount = %d, want %d", hub.ClientCount(), numClients)
	}

	// Broadcast concurrently — should not race.
	var bcastWg sync.WaitGroup
	for i := 0; i < 10; i++ {
		bcastWg.Add(1)
		go func(n int) {
			defer bcastWg.Done()
			hub.Broadcast("tick", `{"n":`+string(rune('0'+n))+`}`)
		}(i)
	}
	bcastWg.Wait()

	// Disconnect all.
	for _, cancel := range ctxs {
		cancel()
	}
	wg.Wait()

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount after all disconnects = %d, want 0", hub.ClientCount())
	}
}

func TestSSEHub_ServeHTTP_NonFlusher(t *testing.T) {
	hub := NewSSEHub()

	// Use a writer that does NOT implement http.Flusher.
	req := httptest.NewRequest("GET", "/api/stream", nil)
	w := &nonFlusherWriter{header: http.Header{}}

	hub.ServeHTTP(w, req)

	if w.code != http.StatusInternalServerError {
		t.Errorf("expected 500 for non-flusher, got %d", w.code)
	}
}

// nonFlusherWriter is a ResponseWriter that doesn't implement Flusher.
type nonFlusherWriter struct {
	header http.Header
	code   int
	body   []byte
}

func (w *nonFlusherWriter) Header() http.Header         { return w.header }
func (w *nonFlusherWriter) Write(b []byte) (int, error)  { w.body = append(w.body, b...); return len(b), nil }
func (w *nonFlusherWriter) WriteHeader(code int)         { w.code = code }
