package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/inscripcion-moodle/go-backend/internal/cache"
	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/services/logs"
)

type LogsController struct {
	cache      *cache.Cache
	cacheTTL   time.Duration
	logDir     string
	logPattern string
}

func NewLogsController(rds *redis.Client, cfg *config.Config) *LogsController {
	return &LogsController{
		cache:      cache.New(rds),
		cacheTTL:   cfg.LogsCacheTTL,
		logDir:     cfg.ApacheLogDir,
		logPattern: cfg.ApacheLogPattern,
	}
}

// GetStats handles GET /api/logs/stats?from=&to=&host=&topN=
func (h *LogsController) GetStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, err := parseRangeBound(q.Get("from"), false)
	if err != nil {
		http.Error(w, "invalid 'from' parameter: "+err.Error(), http.StatusBadRequest)
		return
	}
	to, err := parseRangeBound(q.Get("to"), true)
	if err != nil {
		http.Error(w, "invalid 'to' parameter: "+err.Error(), http.StatusBadRequest)
		return
	}
	site := strings.TrimSpace(q.Get("site"))
	topN := 25
	if raw := q.Get("topN"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 200 {
			topN = n
		}
	}

	key := fmt.Sprintf("logs:stats:v2:%s:%s:%s:%d", q.Get("from"), q.Get("to"), site, topN)
	payload, err := h.cache.GetOrSet(r.Context(), key, h.cacheTTL, func() ([]byte, error) {
		stats, err := logs.Compute(h.logDir, h.logPattern, logs.ReadFilter{
			From: from,
			To:   to,
			Site: site,
		}, topN)
		if err != nil {
			return nil, err
		}
		return json.Marshal(stats)
	})
	if err != nil {
		http.Error(w, "failed to compute log stats: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

// GetSites handles GET /api/logs/sites
func (h *LogsController) GetSites(w http.ResponseWriter, r *http.Request) {
	key := "logs:sites:v1"
	payload, err := h.cache.GetOrSet(r.Context(), key, h.cacheTTL, func() ([]byte, error) {
		sites, err := logs.CollectSites(h.logDir, h.logPattern)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"sites": sites})
	})
	if err != nil {
		http.Error(w, "failed to collect sites: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

// parseRangeBound accepts "YYYY-MM" or "YYYY-MM-DD" (UTC). When inclusiveEnd
// is true, "YYYY-MM" expands to the last day of that month and "YYYY-MM-DD"
// expands to 23:59:59.
func parseRangeBound(raw string, inclusiveEnd bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		if inclusiveEnd {
			return t.Add(24*time.Hour - time.Second), nil
		}
		return t, nil
	}
	if t, err := time.Parse("2006-01", raw); err == nil {
		if inclusiveEnd {
			return t.AddDate(0, 1, 0).Add(-time.Second), nil
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("expected YYYY-MM or YYYY-MM-DD")
}
