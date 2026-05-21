package logs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// WarmupReport summarises a warm-up pass.
type WarmupReport struct {
	Total     int           `json:"total"`
	Built     int           `json:"built"`
	Cached    int           `json:"cached"`
	Failed    int           `json:"failed"`
	Errors    []string      `json:"errors,omitempty"`
	StartedAt time.Time     `json:"started_at"`
	EndedAt   time.Time     `json:"ended_at"`
	Duration  time.Duration `json:"duration"`
	InProgress bool         `json:"in_progress"`
	Cancelled  bool         `json:"cancelled,omitempty"`
}

// WarmupTracker holds the live state of an in-flight warm-up so the admin UI
// can poll progress.
type WarmupTracker struct {
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	report  WarmupReport
}

// NewWarmupTracker returns a fresh tracker.
func NewWarmupTracker() *WarmupTracker { return &WarmupTracker{} }

// Snapshot returns a copy of the current report. Safe to call concurrently.
func (w *WarmupTracker) Snapshot() WarmupReport {
	w.mu.Lock()
	defer w.mu.Unlock()
	r := w.report
	if len(w.report.Errors) > 0 {
		r.Errors = append([]string(nil), w.report.Errors...)
	}
	r.InProgress = w.running
	if w.running {
		r.Duration = time.Since(w.report.StartedAt)
	}
	return r
}

// Cancel signals an in-flight warm-up to stop. No-op if none is running.
func (w *WarmupTracker) Cancel() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
	}
}

// Start launches WarmAll in a goroutine. Returns ErrAlreadyRunning if a
// warm-up is already in progress. The caller's ctx is used as the parent.
func (w *WarmupTracker) Start(ctx context.Context, dir, pattern string, concurrency int, force bool) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return ErrAlreadyRunning
	}
	subCtx, cancel := context.WithCancel(ctx)
	w.running = true
	w.cancel = cancel
	w.report = WarmupReport{StartedAt: time.Now().UTC()}
	w.mu.Unlock()

	go func() {
		defer cancel()
		w.runLocked(subCtx, dir, pattern, concurrency, force)
	}()
	return nil
}

// ErrAlreadyRunning is returned when Start is called while a warm-up is
// already in progress.
var ErrAlreadyRunning = errors.New("warmup already running")

func (w *WarmupTracker) runLocked(ctx context.Context, dir, pattern string, concurrency int, force bool) {
	files, err := ListFiles(dir, pattern, ReadFilter{})
	if err != nil {
		w.finish(WarmupReport{Errors: []string{err.Error()}})
		return
	}

	// Only rotated files get cached; the active log is always re-parsed.
	rotated := make([]FileInfo, 0, len(files))
	for _, f := range files {
		if f.Rotated {
			rotated = append(rotated, f)
		}
	}

	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 8 {
		concurrency = 8
	}

	w.mu.Lock()
	w.report.Total = len(rotated)
	w.mu.Unlock()

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	cancelled := false

	for _, fi := range rotated {
		select {
		case <-ctx.Done():
			cancelled = true
		default:
		}
		if cancelled {
			break
		}
		fi := fi
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			cached := false
			var workErr error
			if !force {
				if _, err := LoadSummary(fi.Path); err == nil {
					cached = true
				}
			}
			if !cached {
				s, err := BuildSummary(fi.Path, fi.Site)
				if err != nil {
					workErr = err
				} else if summaryDir != "" {
					workErr = SaveSummary(fi.Path, s)
				}
			}

			w.mu.Lock()
			if cached {
				w.report.Cached++
			} else if workErr != nil {
				w.report.Failed++
				w.report.Errors = append(w.report.Errors,
					fmt.Sprintf("%s: %v", fi.Path, workErr))
			} else {
				w.report.Built++
			}
			w.mu.Unlock()
		}()
	}
	wg.Wait()

	w.mu.Lock()
	w.report.Cancelled = cancelled
	w.mu.Unlock()
	w.finish(WarmupReport{})
}

func (w *WarmupTracker) finish(_ WarmupReport) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.running = false
	w.cancel = nil
	w.report.EndedAt = time.Now().UTC()
	w.report.Duration = w.report.EndedAt.Sub(w.report.StartedAt)
}
