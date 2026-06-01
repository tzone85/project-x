package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetMetrics_EmptyStore(t *testing.T) {
	h := setupTestHandlers(t)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	h.GetMetrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	got := rec.Body.String()
	wantPrefixes := []string{
		"# HELP px_build_info",
		"# TYPE px_build_info gauge",
		"px_build_info{version=",
		"# HELP px_uptime_seconds",
		"px_uptime_seconds ",
		"# HELP px_cost_usd_total",
		"px_cost_usd_total 0.000000",
	}
	for _, prefix := range wantPrefixes {
		if !strings.Contains(got, prefix) {
			t.Errorf("missing %q in output:\n%s", prefix, got)
		}
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
}

func TestGetMetrics_WithCostRows(t *testing.T) {
	h := setupTestHandlers(t)
	h.dailyLimitUSD = 50.0
	db := h.projStore.DB()
	_, err := db.Exec(`INSERT INTO token_usage (id, req_id, model, input_tokens, output_tokens, cost_usd)
		VALUES ('t1','r1','gpt',100,50,1.25),('t2','r1','gpt',200,80,2.50)`)
	if err != nil {
		t.Fatalf("seed cost: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.GetMetrics(rec, req)

	got := rec.Body.String()
	if !strings.Contains(got, "px_cost_usd_total 3.750000") {
		t.Errorf("expected px_cost_usd_total 3.750000, got:\n%s", got)
	}
	if !strings.Contains(got, "px_cost_daily_limit_usd 50.000000") {
		t.Errorf("expected px_cost_daily_limit_usd 50.000000, got:\n%s", got)
	}
}
