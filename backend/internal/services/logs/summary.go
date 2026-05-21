package logs

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

// Bumped to 2 when bot/scanner classification fields were added. Older
// summaries on disk (version 1) are rejected by LoadSummary so the admin
// panel's "Forzar" action can rebuild them.
const summarySchemaVersion = 2

// Memory caps for the per-file maps. When a map exceeds the "max" size we
// prune it down to "keep" entries (highest counts win). This bounds peak RSS
// during BuildSummary at the cost of dropping low-frequency keys, which is
// the right trade-off for log analytics dominated by long tails of one-hit
// scanner paths and bot IPs.
const (
	urlEntriesMax     = 5000
	urlEntriesKeep    = 2000
	ipEntriesMax      = 2000
	ipEntriesKeep     = 1000
	uaEntriesMax      = 500
	uaEntriesKeep     = 200
	botUAEntriesMax   = 250
	botUAEntriesKeep  = 100
	scannerEntriesMax = 250
	scannerEntriesKeep = 100
)

// botUARegex matches User-Agent strings that self-identify as automated
// clients. Case-insensitive. The pattern is intentionally broad: false
// positives on "human" UAs that contain these substrings are extremely rare,
// while bots that don't match are caught by the scanner-path heuristic.
var botUARegex = regexp.MustCompile(`(?i)(bot|crawl|spider|slurp|bingpreview|duckduck|yandex|baidu|ahrefs|semrush|mj12|dotbot|petal|gptbot|claudebot|ccbot|amazonbot|facebookexternalhit|headlesschrome|python-requests|curl/|wget|go-http-client|httpclient|scrapy|nikto|nmap|masscan|zgrab|sqlmap)`)

// scannerPathRegex matches request paths typically probed by vulnerability
// scanners (WordPress login pages on a site that has no WordPress, env file
// dumps, exposed git directories, Exchange autodiscover, etc.). A hit on any
// of these paths is treated as scanner traffic regardless of the User-Agent.
var scannerPathRegex = regexp.MustCompile(`(?i)(/wp-(login|admin|content|includes)|/wordpress/|/\.env|/phpmyadmin|/xmlrpc|/\.git/|/actuator|/console|/admin\.php|/owa/|/autodiscover|/\.aws/|/\.ssh/|/server-status|/cgi-bin/|/vendor/phpunit)`)

// classifyEntry returns whether an entry is bot traffic, scanner traffic, or
// neither. An entry can be both (counted once on each axis).
func classifyEntry(e LogEntry) (isBot, isScanner bool) {
	if e.UserAgent != "" && botUARegex.MatchString(e.UserAgent) {
		isBot = true
	}
	if e.URL != "" && scannerPathRegex.MatchString(e.URL) {
		isScanner = true
	}
	return
}

// summaryDir is the on-disk location for *.summary.json.gz files. Set via
// SetSummaryDir at startup; defaults to empty (writes disabled).
var summaryDir = ""

// SetSummaryDir configures where summaries are stored. The directory will be
// created on first write. Empty value disables persistent caching (summaries
// are still computed but not saved/loaded).
func SetSummaryDir(dir string) { summaryDir = dir }

// SummaryDir returns the configured directory.
func SummaryDir() string { return summaryDir }

// FileSummary is the precomputed aggregation for a single log file.
type FileSummary struct {
	Version        int              `json:"version"`
	Site           string           `json:"site"`
	Source         string           `json:"source"`
	SourceMTime    time.Time        `json:"source_mtime"`
	MinTime        time.Time        `json:"min_time"`
	MaxTime        time.Time        `json:"max_time"`
	TotalHits      int64            `json:"total_hits"`
	BotHits        int64            `json:"bot_hits"`
	ScannerHits    int64            `json:"scanner_hits"`
	ByDay          map[string]int64 `json:"by_day"`
	ByURL          map[string]int64 `json:"by_url"`
	ByIP           map[string]int64 `json:"by_ip"`
	ByStatus       map[int]int64    `json:"by_status"`
	ByMethod       map[string]int64 `json:"by_method"`
	ByUA           map[string]int64 `json:"by_ua"`
	ByBotUA        map[string]int64 `json:"by_bot_ua"`
	ByScannerPath  map[string]int64 `json:"by_scanner_path"`
}

func newSummary(site, source string, mtime time.Time) *FileSummary {
	return &FileSummary{
		Version:       summarySchemaVersion,
		Site:          site,
		Source:        source,
		SourceMTime:   mtime,
		ByDay:         make(map[string]int64),
		ByURL:         make(map[string]int64),
		ByIP:          make(map[string]int64),
		ByStatus:      make(map[int]int64),
		ByMethod:      make(map[string]int64),
		ByUA:          make(map[string]int64),
		ByBotUA:       make(map[string]int64),
		ByScannerPath: make(map[string]int64),
	}
}

// BuildSummary parses the given log file end-to-end and produces a summary.
// The file is read with the same gzip-aware streaming logic as the live
// reader.
func BuildSummary(path, site string) (*FileSummary, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	s := newSummary(site, filepath.Base(path), info.ModTime().UTC())
	err = readOne(path, site, ReadFilter{}, func(e LogEntry) {
		s.TotalHits++
		day := e.Time.UTC().Format("2006-01-02")
		s.ByDay[day]++
		if e.URL != "" {
			s.ByURL[e.URL]++
			if len(s.ByURL) > urlEntriesMax {
				pruneTopN(s.ByURL, urlEntriesKeep)
			}
		}
		if e.IPPrefix != "" {
			s.ByIP[e.IPPrefix]++
			if len(s.ByIP) > ipEntriesMax {
				pruneTopN(s.ByIP, ipEntriesKeep)
			}
		}
		s.ByStatus[e.Status]++
		if e.Method != "" {
			s.ByMethod[e.Method]++
		}
		isBot, isScanner := classifyEntry(e)
		if e.UserAgent != "" {
			s.ByUA[e.UserAgent]++
			if len(s.ByUA) > uaEntriesMax {
				pruneTopN(s.ByUA, uaEntriesKeep)
			}
		}
		if isBot {
			s.BotHits++
			if e.UserAgent != "" {
				s.ByBotUA[e.UserAgent]++
				if len(s.ByBotUA) > botUAEntriesMax {
					pruneTopN(s.ByBotUA, botUAEntriesKeep)
				}
			}
		}
		if isScanner {
			s.ScannerHits++
			if e.URL != "" {
				s.ByScannerPath[e.URL]++
				if len(s.ByScannerPath) > scannerEntriesMax {
					pruneTopN(s.ByScannerPath, scannerEntriesKeep)
				}
			}
		}
		if s.MinTime.IsZero() || e.Time.Before(s.MinTime) {
			s.MinTime = e.Time
		}
		if e.Time.After(s.MaxTime) {
			s.MaxTime = e.Time
		}
	})
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	// Final prune so saved summaries never exceed the keep size.
	pruneTopN(s.ByURL, urlEntriesKeep)
	pruneTopN(s.ByIP, ipEntriesKeep)
	pruneTopN(s.ByUA, uaEntriesKeep)
	pruneTopN(s.ByBotUA, botUAEntriesKeep)
	pruneTopN(s.ByScannerPath, scannerEntriesKeep)
	return s, nil
}

// pruneTopN keeps the keep-highest entries in m and drops the rest. No-op if
// the map is already small enough. Counts of dropped entries are lost, which
// is acceptable for log analytics where the long tail is dominated by
// one-hit scanner traffic.
func pruneTopN(m map[string]int64, keep int) {
	if len(m) <= keep {
		return
	}
	counts := make([]int64, 0, len(m))
	for _, v := range m {
		counts = append(counts, v)
	}
	sort.Slice(counts, func(i, j int) bool { return counts[i] > counts[j] })
	threshold := counts[keep-1]
	for k, v := range m {
		if v < threshold {
			delete(m, k)
		}
	}
	// Above-threshold entries that share the threshold value can leave the
	// map slightly larger than `keep`. Drop ties until size matches.
	for k, v := range m {
		if len(m) <= keep {
			break
		}
		if v == threshold {
			delete(m, k)
		}
	}
}

// summaryPath maps a log file path to its summary file path under summaryDir.
// Path separators and characters that some filesystems disallow (e.g. ':' on
// Windows) are flattened so the summary dir stays flat and portable.
func summaryPath(logPath string) string {
	if summaryDir == "" {
		return ""
	}
	flat := strings.NewReplacer(
		string(filepath.Separator), "__",
		"/", "__",
		":", "_",
	).Replace(logPath)
	return filepath.Join(summaryDir, flat+".summary.json.gz")
}

// LoadSummary reads a previously persisted summary. Returns os.ErrNotExist if
// no cached summary is available.
func LoadSummary(logPath string) (*FileSummary, error) {
	sp := summaryPath(logPath)
	if sp == "" {
		return nil, os.ErrNotExist
	}
	f, err := os.Open(sp)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip open %s: %w", sp, err)
	}
	defer gz.Close()
	var s FileSummary
	if err := json.NewDecoder(gz).Decode(&s); err != nil {
		return nil, fmt.Errorf("decode %s: %w", sp, err)
	}
	if s.Version != summarySchemaVersion {
		return nil, fmt.Errorf("summary %s: schema version %d (want %d)", sp, s.Version, summarySchemaVersion)
	}
	return &s, nil
}

// SaveSummary writes the summary atomically (tmp + rename). Caller-supplied
// summaryDir must be writable.
func SaveSummary(logPath string, s *FileSummary) error {
	sp := summaryPath(logPath)
	if sp == "" {
		return errors.New("summary directory not configured")
	}
	if err := os.MkdirAll(filepath.Dir(sp), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(sp), err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(sp), filepath.Base(sp)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		// Best-effort cleanup if anything failed mid-way.
		_ = os.Remove(tmpName)
	}()
	gz := gzip.NewWriter(tmp)
	if err := json.NewEncoder(gz).Encode(s); err != nil {
		_ = gz.Close()
		_ = tmp.Close()
		return err
	}
	if err := gz.Close(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, sp); err != nil {
		return err
	}
	return nil
}

// ensureGroup deduplicates concurrent EnsureSummary calls for the same path.
var ensureGroup singleflight.Group

// EnsureSummary returns the summary for logPath, loading from disk when
// available and building+saving it otherwise. Only call this for ROTATED
// (immutable) files; the active log should call BuildSummary directly to
// avoid caching a moving target.
func EnsureSummary(logPath, site string) (*FileSummary, error) {
	v, err, _ := ensureGroup.Do(logPath, func() (any, error) {
		if s, err := LoadSummary(logPath); err == nil {
			return s, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			// If a cached summary exists but is corrupt or wrong version, try
			// to fall through to a rebuild rather than fail the whole query.
			fmt.Fprintf(io.Discard, "logs: discarding broken summary for %s: %v\n", logPath, err)
		}
		s, err := BuildSummary(logPath, site)
		if err != nil {
			return nil, err
		}
		if summaryDir != "" {
			if err := SaveSummary(logPath, s); err != nil {
				// Don't fail the request: in-memory summary is still good.
				fmt.Fprintf(io.Discard, "logs: save summary failed for %s: %v\n", logPath, err)
			}
		}
		return s, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*FileSummary), nil
}
