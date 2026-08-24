package logs

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// siteFromFilename extracts a logical site label from a log filename.
// Examples:
//
//	moodle-access.log               → "moodle"
//	opositatcae-access.log-20260503.gz → "opositatcae"
//	simulador-access.log            → "simulador"
//	inscripcion_http_access.log     → "inscripcion"
//	inscripcion_https_access.log.5.gz → "inscripcion"
//	access_log                      → "default"
var siteSplitRegex = regexp.MustCompile(`[-_](https?_)?access`)

func siteFromFilename(path string) string {
	name := filepath.Base(path)
	if loc := siteSplitRegex.FindStringIndex(name); loc != nil {
		return name[:loc[0]]
	}
	return "default"
}

// rotationDateRegex extracts the YYYYMMDD suffix that logrotate appends to
// rotated files (e.g. "moodle-access.log-20260419.gz"). The current (active)
// log file has no such suffix.
var rotationDateRegex = regexp.MustCompile(`-(\d{8})\.gz$`)

// rotationDate returns the rotation date encoded in the filename, or the zero
// value if the file is the current (uncompressed) log.
func rotationDate(path string) time.Time {
	m := rotationDateRegex.FindStringSubmatch(filepath.Base(path))
	if m == nil {
		return time.Time{}
	}
	t, err := time.Parse("20060102", m[1])
	if err != nil {
		return time.Time{}
	}
	return t
}

// scanBufferMax is the largest single log line we will accept (1 MiB).
const scanBufferMax = 1024 * 1024

// ReadFilter narrows the lines that are returned to the caller. Empty values
// disable the corresponding check.
type ReadFilter struct {
	From time.Time
	To   time.Time
	Site string
}

// FileInfo describes a candidate log file.
type FileInfo struct {
	Path    string
	Site    string
	Rotated bool // true when the filename has a -YYYYMMDD.gz rotation suffix
}

// ListFiles globs the patterns under dir and returns matching files annotated
// with their derived site and rotation flag. Filter narrows by site only;
// date-range pruning lives in ListRelevantFiles.
func ListFiles(dir, pattern string, filter ReadFilter) ([]FileInfo, error) {
	if dir == "" {
		return nil, fmt.Errorf("logs: directory is required")
	}
	if pattern == "" {
		pattern = "access_log*"
	}
	seen := make(map[string]struct{})
	var matches []string
	for p := range strings.SplitSeq(pattern, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		found, err := filepath.Glob(filepath.Join(dir, p))
		if err != nil {
			return nil, fmt.Errorf("logs: glob %q: %w", p, err)
		}
		for _, m := range found {
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			matches = append(matches, m)
		}
	}
	sort.Strings(matches)

	out := make([]FileInfo, 0, len(matches))
	for _, path := range matches {
		site := siteFromFilename(path)
		if filter.Site != "" && !strings.EqualFold(filter.Site, site) {
			continue
		}
		rd := rotationDate(path)
		out = append(out, FileInfo{Path: path, Site: site, Rotated: !rd.IsZero()})
	}
	return out, nil
}

// ListRelevantFiles returns files that may contain entries inside
// [filter.From, filter.To]. The active (uncompressed) log is always included.
func ListRelevantFiles(dir, pattern string, filter ReadFilter) ([]FileInfo, error) {
	all, err := ListFiles(dir, pattern, filter)
	if err != nil {
		return nil, err
	}
	out := make([]FileInfo, 0, len(all))
	for _, fi := range all {
		if fi.Rotated {
			rd := rotationDate(fi.Path)
			if !filter.From.IsZero() && rd.Before(filter.From.AddDate(0, 0, -1)) {
				continue
			}
			if !filter.To.IsZero() && rd.After(filter.To.AddDate(0, 0, 14)) {
				continue
			}
		}
		out = append(out, fi)
	}
	return out, nil
}

// EachEntry walks every relevant log file and invokes fn for matching entries.
// Used by BuildSummary; the live query path now goes through summaries.
func EachEntry(dir, pattern string, filter ReadFilter, fn func(LogEntry)) error {
	files, err := ListRelevantFiles(dir, pattern, filter)
	if err != nil {
		return err
	}
	for _, fi := range files {
		if err := readOne(fi.Path, fi.Site, filter, fn); err != nil {
			return fmt.Errorf("logs: read %s: %w", fi.Path, err)
		}
	}
	return nil
}

func readOne(path, site string, filter ReadFilter, fn func(LogEntry)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		r = gz
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), scanBufferMax)
	for scanner.Scan() {
		entry, ok := ParseLine(scanner.Text())
		if !ok {
			continue
		}
		entry.Site = site
		if !filter.matches(entry) {
			continue
		}
		fn(entry)
	}
	return scanner.Err()
}

func (f ReadFilter) matches(e LogEntry) bool {
	if !f.From.IsZero() && e.Time.Before(f.From) {
		return false
	}
	if !f.To.IsZero() && e.Time.After(f.To) {
		return false
	}
	return true
}
