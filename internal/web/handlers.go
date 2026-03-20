package web

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/tzone85/project-x/internal/state"
)

// startTime records when the server process started, for the health endpoint.
var startTime = time.Now()

// Handlers provides HTTP handler methods for the web dashboard API.
// All handlers return JSON responses with proper Content-Type headers.
type Handlers struct {
	eventStore state.EventStore
	projStore  *state.SQLiteStore
	db         *sql.DB
}

// ListRequirements returns all non-archived requirements as JSON.
func (h *Handlers) ListRequirements(w http.ResponseWriter, r *http.Request) {
	reqs, err := h.projStore.ListRequirements(state.ReqFilter{ExcludeArchived: true})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, ensureSlice(reqs))
}

// ListStories returns stories filtered by optional query parameters:
// req_id, status, limit, offset.
func (h *Handlers) ListStories(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := state.StoryFilter{
		ReqID:  q.Get("req_id"),
		Status: q.Get("status"),
		Limit:  parseIntParam(q.Get("limit"), 0),
		Offset: parseIntParam(q.Get("offset"), 0),
	}

	stories, err := h.projStore.ListStories(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, ensureSlice(stories))
}

// ListAgents returns agents filtered by optional status query parameter.
func (h *Handlers) ListAgents(w http.ResponseWriter, r *http.Request) {
	filter := state.AgentFilter{
		Status: r.URL.Query().Get("status"),
	}

	agents, err := h.projStore.ListAgents(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, ensureSlice(agents))
}

// ListEvents returns events from the event store, filtered by optional
// query parameters: type, agent_id, story_id, limit.
func (h *Handlers) ListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := state.EventFilter{
		Type:    state.EventType(q.Get("type")),
		AgentID: q.Get("agent_id"),
		StoryID: q.Get("story_id"),
		Limit:   parseIntParam(q.Get("limit"), 0),
		After:   q.Get("after"),
	}

	events, err := h.eventStore.List(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, ensureSlice(events))
}

// costResponse is the JSON shape returned by the cost endpoint.
type costResponse struct {
	TodayUSD float64 `json:"today_usd"`
	ReqUSD   float64 `json:"req_usd,omitempty"`
	StoryUSD float64 `json:"story_usd,omitempty"`
}

// GetCost returns cost summary data. Supports optional query parameters:
// req_id (cost for a specific requirement), story_id (cost for a specific story).
func (h *Handlers) GetCost(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	today := time.Now().Format("2006-01-02")

	resp := costResponse{}

	// Daily cost.
	dailyCost, err := queryCostByDay(h.db, today)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.TodayUSD = dailyCost

	// Optional: cost by requirement.
	if reqID := q.Get("req_id"); reqID != "" {
		reqCost, err := queryCostByReq(h.db, reqID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.ReqUSD = reqCost
	}

	// Optional: cost by story.
	if storyID := q.Get("story_id"); storyID != "" {
		storyCost, err := queryCostByStory(h.db, storyID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.StoryUSD = storyCost
	}

	writeJSON(w, resp)
}

// healthResponse is the JSON shape returned by the health endpoint.
type healthResponse struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

// GetHealth returns the server health status.
func (h *Handlers) GetHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, healthResponse{
		Status: "ok",
		Uptime: time.Since(startTime).Round(time.Second).String(),
	})
}

// writeJSON encodes v as JSON and writes it to w with the correct Content-Type.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// parseIntParam parses a string to int, returning defaultVal on error.
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// ensureSlice returns an empty non-nil slice if the input is nil.
// This ensures JSON encoding produces [] instead of null.
func ensureSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// queryCostByDay returns the total cost for a given date from the token_usage table.
func queryCostByDay(db *sql.DB, date string) (float64, error) {
	var total float64
	err := db.QueryRow(
		"SELECT COALESCE(SUM(cost_usd), 0) FROM token_usage WHERE date(created_at) = ?",
		date,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// queryCostByReq returns the total cost for a requirement.
func queryCostByReq(db *sql.DB, reqID string) (float64, error) {
	var total float64
	err := db.QueryRow(
		"SELECT COALESCE(SUM(cost_usd), 0) FROM token_usage WHERE req_id = ?",
		reqID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// queryCostByStory returns the total cost for a story.
func queryCostByStory(db *sql.DB, storyID string) (float64, error) {
	var total float64
	err := db.QueryRow(
		"SELECT COALESCE(SUM(cost_usd), 0) FROM token_usage WHERE story_id = ?",
		storyID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}
