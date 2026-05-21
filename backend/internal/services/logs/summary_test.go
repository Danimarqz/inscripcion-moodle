package logs

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const sampleLines = `203.0.113.10 - - [12/Apr/2026:10:00:00 +0000] "GET /a HTTP/1.1" 200 100 "-" "-"
203.0.113.10 - - [12/Apr/2026:10:01:00 +0000] "GET /a HTTP/1.1" 200 100 "-" "-"
198.51.100.5 - - [13/Apr/2026:11:00:00 +0000] "POST /b HTTP/1.1" 404 50 "-" "-"
198.51.100.5 - - [14/Apr/2026:00:00:00 +0000] "GET /a HTTP/1.1" 500 0 "-" "-"
`

func writeLogFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeGzLogFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBuildSummary_Counts(t *testing.T) {
	dir := t.TempDir()
	path := writeLogFile(t, dir, "moodle-access.log", sampleLines)
	s, err := BuildSummary(path, "moodle")
	if err != nil {
		t.Fatal(err)
	}
	if s.TotalHits != 4 {
		t.Errorf("total = %d, want 4", s.TotalHits)
	}
	if s.ByDay["2026-04-12"] != 2 {
		t.Errorf("byDay[2026-04-12] = %d, want 2", s.ByDay["2026-04-12"])
	}
	if s.ByURL["/a"] != 3 {
		t.Errorf("byURL[/a] = %d, want 3", s.ByURL["/a"])
	}
	if s.ByStatus[200] != 2 || s.ByStatus[404] != 1 || s.ByStatus[500] != 1 {
		t.Errorf("byStatus = %v", s.ByStatus)
	}
	if s.ByMethod["GET"] != 3 {
		t.Errorf("byMethod[GET] = %d, want 3", s.ByMethod["GET"])
	}
}

func TestSaveLoadSummary_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	cacheDir := t.TempDir()
	SetSummaryDir(cacheDir)
	defer SetSummaryDir("")

	path := writeLogFile(t, dir, "moodle-access.log", sampleLines)
	original, err := BuildSummary(path, "moodle")
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveSummary(path, original); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSummary(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TotalHits != original.TotalHits {
		t.Errorf("totalHits roundtrip mismatch: %d vs %d", loaded.TotalHits, original.TotalHits)
	}
	if loaded.ByURL["/a"] != original.ByURL["/a"] {
		t.Errorf("byURL roundtrip mismatch")
	}
}

func TestEnsureSummary_UsesCache(t *testing.T) {
	dir := t.TempDir()
	cacheDir := t.TempDir()
	SetSummaryDir(cacheDir)
	defer SetSummaryDir("")

	path := writeGzLogFile(t, dir, "moodle-access.log-20260419.gz", sampleLines)

	// First call: builds + saves.
	first, err := EnsureSummary(path, "moodle")
	if err != nil {
		t.Fatal(err)
	}
	if first.TotalHits != 4 {
		t.Fatalf("first total = %d", first.TotalHits)
	}

	// Tamper with the underlying log to detect re-parsing.
	if err := os.WriteFile(path, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second call: should load the cached summary, not re-parse.
	second, err := EnsureSummary(path, "moodle")
	if err != nil {
		t.Fatal(err)
	}
	if second.TotalHits != 4 {
		t.Errorf("cache miss: total = %d", second.TotalHits)
	}
}

func TestMergeSummaries_DateRangeFilter(t *testing.T) {
	a := newSummary("moodle", "a", time.Now())
	a.TotalHits = 100
	a.ByDay["2026-04-01"] = 40
	a.ByDay["2026-04-02"] = 60
	a.ByURL["/x"] = 100
	a.MinTime = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	a.MaxTime = time.Date(2026, 4, 2, 23, 59, 59, 0, time.UTC)

	b := newSummary("moodle", "b", time.Now())
	b.TotalHits = 50
	b.ByDay["2026-04-10"] = 50
	b.ByURL["/y"] = 50
	b.MinTime = time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	b.MaxTime = time.Date(2026, 4, 10, 23, 59, 59, 0, time.UTC)

	stats := MergeSummaries([]*FileSummary{a, b}, ReadFilter{
		From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 5, 23, 59, 59, 0, time.UTC),
	}, Range{}, 10)

	if stats.TotalHits != 100 {
		t.Errorf("total = %d, want 100 (b should be excluded by date)", stats.TotalHits)
	}
}
