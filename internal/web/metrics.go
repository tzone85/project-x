package web

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// GetMetrics serves operational metrics in the Prometheus text exposition
// format. Counts are computed on demand from the SQLite projections; the
// endpoint is intended for local scraping and small enough not to need a
// metrics registry.
func (h *Handlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	writeBuildInfo(w, h.version)
	writeUptime(w)

	if h.db == nil {
		return
	}

	writeStatusCounts(w, h.db, "px_requirements_total", "requirements")
	writeStatusCounts(w, h.db, "px_stories_total", "stories")
	writeStatusCounts(w, h.db, "px_agents_total", "agents")
	writeEscalationCounts(w, h.db)
	writeCostTotals(w, h.db, h.dailyLimitUSD)
}

func writeBuildInfo(w io.Writer, version string) {
	_, _ = fmt.Fprintln(w, "# HELP px_build_info Build information.")
	_, _ = fmt.Fprintln(w, "# TYPE px_build_info gauge")
	_, _ = fmt.Fprintf(w, "px_build_info{version=%q} 1\n", version)
}

func writeUptime(w io.Writer) {
	uptime := time.Since(startTime).Seconds()
	_, _ = fmt.Fprintln(w, "# HELP px_uptime_seconds Process uptime in seconds.")
	_, _ = fmt.Fprintln(w, "# TYPE px_uptime_seconds gauge")
	_, _ = fmt.Fprintf(w, "px_uptime_seconds %.3f\n", uptime)
}

// writeStatusCounts emits `<metric>{status="..."} N` rows for every status
// value present in the named table. Tables without a `status` column are
// skipped silently — the query simply returns no rows.
func writeStatusCounts(w io.Writer, db *sql.DB, metric, table string) {
	rows, err := db.Query(fmt.Sprintf(`SELECT status, COUNT(*) FROM %s GROUP BY status`, table))
	if err != nil {
		slog.Debug("metrics: count query failed", "table", table, "err", err)
		return
	}
	defer func() { _ = rows.Close() }()

	_, _ = fmt.Fprintf(w, "# HELP %s Count by status in table %s.\n", metric, table)
	_, _ = fmt.Fprintf(w, "# TYPE %s gauge\n", metric)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		_, _ = fmt.Fprintf(w, "%s{status=%q} %d\n", metric, status, count)
	}
}

func writeEscalationCounts(w io.Writer, db *sql.DB) {
	// Schema may carry escalation state under different column names; try the
	// common ones and fall back to a single open/closed split.
	rows, err := db.Query(`SELECT COUNT(*) FROM escalations`)
	if err != nil {
		slog.Debug("metrics: escalations query failed", "err", err)
		return
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return
	}
	var total int64
	if err := rows.Scan(&total); err != nil {
		return
	}
	_, _ = fmt.Fprintln(w, "# HELP px_escalations_total Total escalations recorded.")
	_, _ = fmt.Fprintln(w, "# TYPE px_escalations_total gauge")
	_, _ = fmt.Fprintf(w, "px_escalations_total %d\n", total)
}

func writeCostTotals(w io.Writer, db *sql.DB, dailyLimitUSD float64) {
	var totalUSD float64
	row := db.QueryRow(`SELECT COALESCE(SUM(cost_usd), 0) FROM token_usage`)
	if err := row.Scan(&totalUSD); err != nil {
		slog.Debug("metrics: total cost query failed", "err", err)
		return
	}
	_, _ = fmt.Fprintln(w, "# HELP px_cost_usd_total Sum of recorded LLM token cost in USD.")
	_, _ = fmt.Fprintln(w, "# TYPE px_cost_usd_total counter")
	_, _ = fmt.Fprintf(w, "px_cost_usd_total %.6f\n", totalUSD)

	var todayUSD float64
	row = db.QueryRow(
		`SELECT COALESCE(SUM(cost_usd), 0) FROM token_usage WHERE substr(created_at, 1, 10) = ?`,
		time.Now().UTC().Format("2006-01-02"),
	)
	if err := row.Scan(&todayUSD); err == nil {
		_, _ = fmt.Fprintln(w, "# HELP px_cost_usd_today Sum of recorded LLM token cost in USD for today (UTC).")
		_, _ = fmt.Fprintln(w, "# TYPE px_cost_usd_today gauge")
		_, _ = fmt.Fprintf(w, "px_cost_usd_today %.6f\n", todayUSD)
	}

	if dailyLimitUSD > 0 {
		_, _ = fmt.Fprintln(w, "# HELP px_cost_daily_limit_usd Configured daily cost budget in USD.")
		_, _ = fmt.Fprintln(w, "# TYPE px_cost_daily_limit_usd gauge")
		_, _ = fmt.Fprintf(w, "px_cost_daily_limit_usd %.6f\n", dailyLimitUSD)
	}
}
