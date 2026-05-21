package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/inscripcion-moodle/go-backend/internal/services/logs"
)

// adminLogsTracker is a process-wide singleton so both POST and GET handlers
// see the same warm-up state.
var adminLogsTracker = logs.NewWarmupTracker()

// startWarmup kicks off a warm-up if none is in progress. Returns 202 with
// the current snapshot, or 409 if one is already running.
func (h *AdminController) startWarmup(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force") == "true"
	dir := h.cfg.ApacheLogDir
	pattern := h.cfg.ApacheLogPattern
	conc := h.cfg.LogsWarmupConcurrency

	// Run with a fresh background context so closing the HTTP response does
	// not abort the warm-up.
	if err := adminLogsTracker.Start(context.Background(), dir, pattern, conc, force); err != nil {
		if errors.Is(err, logs.ErrAlreadyRunning) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(adminLogsTracker.Snapshot())
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(adminLogsTracker.Snapshot())
}

func (h *AdminController) warmupStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(adminLogsTracker.Snapshot())
}

func (h *AdminController) cancelWarmup(w http.ResponseWriter, _ *http.Request) {
	adminLogsTracker.Cancel()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(adminLogsTracker.Snapshot())
}
