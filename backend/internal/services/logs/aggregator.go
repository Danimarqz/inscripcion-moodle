package logs

import (
	"fmt"
	"sort"
	"time"
)

// Stats is the JSON payload returned to the frontend.
type Stats struct {
	Range          Range          `json:"range"`
	TotalHits      int64          `json:"totalHits"`
	BotHits        int64          `json:"botHits"`
	ScannerHits    int64          `json:"scannerHits"`
	HumanHits      int64          `json:"humanHits"`
	UniqueIPs      int            `json:"uniqueIPs"`
	ByDay          []DayBucket    `json:"byDay"`
	ByMonth        []MonthBucket  `json:"byMonth"`
	TopURLs        []URLBucket    `json:"topUrls"`
	TopIPs         []IPBucket     `json:"topIps"`
	ByStatus       []StatusBucket `json:"byStatus"`
	Sites          []string       `json:"sites"`
	TopMethods     []MethodBucket `json:"topMethods"`
	TopUA          []UABucket     `json:"topUA"`
	TopBotUA       []UABucket     `json:"topBotUA"`
	TopScannerPath []URLBucket    `json:"topScannerPath"`
}

type UABucket struct {
	UserAgent string `json:"userAgent"`
	Hits      int64  `json:"hits"`
}

type Range struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	Site string `json:"site,omitempty"`
}

type DayBucket struct {
	Date string `json:"date"`
	Hits int64  `json:"hits"`
}

type MonthBucket struct {
	Month string `json:"month"`
	Hits  int64  `json:"hits"`
}

type URLBucket struct {
	URL  string `json:"url"`
	Hits int64  `json:"hits"`
}

type IPBucket struct {
	IP   string `json:"ip"`
	Hits int64  `json:"hits"`
}

type StatusBucket struct {
	Status int   `json:"status"`
	Hits   int64 `json:"hits"`
}

type MethodBucket struct {
	Method string `json:"method"`
	Hits   int64  `json:"hits"`
}

// Aggregator accumulates log entries and produces a Stats summary.
type Aggregator struct {
	totalHits     int64
	botHits       int64
	scannerHits   int64
	byDay         map[string]int64
	byMonth       map[string]int64
	byURL         map[string]int64
	byIP          map[string]int64
	byStatus      map[int]int64
	byMethod      map[string]int64
	byUA          map[string]int64
	byBotUA       map[string]int64
	byScannerPath map[string]int64
	sites         map[string]struct{}
}

func NewAggregator() *Aggregator {
	return &Aggregator{
		byDay:         make(map[string]int64),
		byMonth:       make(map[string]int64),
		byURL:         make(map[string]int64),
		byIP:          make(map[string]int64),
		byStatus:      make(map[int]int64),
		byMethod:      make(map[string]int64),
		byUA:          make(map[string]int64),
		byBotUA:       make(map[string]int64),
		byScannerPath: make(map[string]int64),
		sites:         make(map[string]struct{}),
	}
}

func (a *Aggregator) Add(e LogEntry) {
	a.totalHits++
	day := e.Time.UTC().Format("2006-01-02")
	month := e.Time.UTC().Format("2006-01")
	a.byDay[day]++
	a.byMonth[month]++
	if e.URL != "" {
		a.byURL[e.URL]++
	}
	if e.IPPrefix != "" {
		a.byIP[e.IPPrefix]++
	}
	a.byStatus[e.Status]++
	if e.Method != "" {
		a.byMethod[e.Method]++
	}
	if e.Site != "" {
		a.sites[e.Site] = struct{}{}
	}
}

// Result builds the final Stats. topN limits the size of the URL / IP /
// method tables; pass <=0 to use the default of 25.
func (a *Aggregator) Result(rng Range, topN int) Stats {
	if topN <= 0 {
		topN = 25
	}

	stats := Stats{
		Range:          rng,
		TotalHits:      a.totalHits,
		BotHits:        a.botHits,
		ScannerHits:    a.scannerHits,
		HumanHits:      a.totalHits - a.botHits - a.scannerHits,
		UniqueIPs:      len(a.byIP),
		ByDay:          daySlice(a.byDay),
		ByMonth:        monthSlice(a.byMonth),
		TopURLs:        topURLs(a.byURL, topN),
		TopIPs:         topIPs(a.byIP, topN),
		ByStatus:       statusSlice(a.byStatus),
		Sites:          sortedKeys(a.sites),
		TopMethods:     topMethods(a.byMethod),
		TopUA:          topUA(a.byUA, topN),
		TopBotUA:       topUA(a.byBotUA, topN),
		TopScannerPath: topURLs(a.byScannerPath, topN),
	}
	if stats.HumanHits < 0 {
		stats.HumanHits = 0
	}
	return stats
}

func topUA(in map[string]int64, n int) []UABucket {
	out := make([]UABucket, 0, len(in))
	for ua, c := range in {
		out = append(out, UABucket{UserAgent: ua, Hits: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Hits != out[j].Hits {
			return out[i].Hits > out[j].Hits
		}
		return out[i].UserAgent < out[j].UserAgent
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func daySlice(in map[string]int64) []DayBucket {
	out := make([]DayBucket, 0, len(in))
	for d, n := range in {
		out = append(out, DayBucket{Date: d, Hits: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

func monthSlice(in map[string]int64) []MonthBucket {
	out := make([]MonthBucket, 0, len(in))
	for m, n := range in {
		out = append(out, MonthBucket{Month: m, Hits: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Month < out[j].Month })
	return out
}

func topURLs(in map[string]int64, n int) []URLBucket {
	out := make([]URLBucket, 0, len(in))
	for u, c := range in {
		out = append(out, URLBucket{URL: u, Hits: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Hits != out[j].Hits {
			return out[i].Hits > out[j].Hits
		}
		return out[i].URL < out[j].URL
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func topIPs(in map[string]int64, n int) []IPBucket {
	out := make([]IPBucket, 0, len(in))
	for ip, c := range in {
		out = append(out, IPBucket{IP: ip, Hits: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Hits != out[j].Hits {
			return out[i].Hits > out[j].Hits
		}
		return out[i].IP < out[j].IP
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func statusSlice(in map[int]int64) []StatusBucket {
	out := make([]StatusBucket, 0, len(in))
	for s, c := range in {
		out = append(out, StatusBucket{Status: s, Hits: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Status < out[j].Status })
	return out
}

func sortedKeys(in map[string]struct{}) []string {
	out := make([]string, 0, len(in))
	for h := range in {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

func topMethods(in map[string]int64) []MethodBucket {
	out := make([]MethodBucket, 0, len(in))
	for m, c := range in {
		out = append(out, MethodBucket{Method: m, Hits: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Hits > out[j].Hits })
	return out
}

// Compute lists relevant log files, ensures each one has a summary on disk
// (built and persisted on first access for rotated files; rebuilt every time
// for the active log), and merges the summaries into a Stats payload.
func Compute(dir, pattern string, filter ReadFilter, topN int) (Stats, error) {
	files, err := ListRelevantFiles(dir, pattern, filter)
	if err != nil {
		return Stats{}, err
	}
	summaries := make([]*FileSummary, 0, len(files))
	for _, fi := range files {
		var s *FileSummary
		var err error
		if fi.Rotated {
			s, err = EnsureSummary(fi.Path, fi.Site)
		} else {
			// Active log: never cached because it grows continuously.
			s, err = BuildSummary(fi.Path, fi.Site)
		}
		if err != nil {
			return Stats{}, fmt.Errorf("summary %s: %w", fi.Path, err)
		}
		summaries = append(summaries, s)
	}

	rng := Range{Site: filter.Site}
	if !filter.From.IsZero() {
		rng.From = filter.From.UTC().Format(time.RFC3339)
	}
	if !filter.To.IsZero() {
		rng.To = filter.To.UTC().Format(time.RFC3339)
	}
	return MergeSummaries(summaries, filter, rng, topN), nil
}

// MergeSummaries fuses per-file summaries into a single Stats result, applying
// a date-range filter at day granularity and trimming top-N tables.
func MergeSummaries(summaries []*FileSummary, filter ReadFilter, rng Range, topN int) Stats {
	agg := NewAggregator()
	fromDay := ""
	toDay := ""
	if !filter.From.IsZero() {
		fromDay = filter.From.UTC().Format("2006-01-02")
	}
	if !filter.To.IsZero() {
		toDay = filter.To.UTC().Format("2006-01-02")
	}

	for _, s := range summaries {
		if s == nil {
			continue
		}
		if s.Site != "" {
			agg.sites[s.Site] = struct{}{}
		}
		// If every day in this summary is in range we can short-circuit and
		// fold the precomputed totals. Otherwise we fall back to per-day
		// recomputation, which preserves byDay accuracy but loses URL/IP
		// granularity for partially-covered files. In practice the boundary
		// files only contribute a small fraction of hits, so this is fine.
		if s.MaxTime.IsZero() || (inRange(s.MinTime, fromDay, toDay) && inRange(s.MaxTime, fromDay, toDay)) {
			foldFull(agg, s)
		} else {
			foldByDay(agg, s, fromDay, toDay)
		}
	}
	return agg.Result(rng, topN)
}

func inRange(t time.Time, fromDay, toDay string) bool {
	if t.IsZero() {
		return true
	}
	d := t.UTC().Format("2006-01-02")
	if fromDay != "" && d < fromDay {
		return false
	}
	if toDay != "" && d > toDay {
		return false
	}
	return true
}

func foldFull(a *Aggregator, s *FileSummary) {
	a.totalHits += s.TotalHits
	a.botHits += s.BotHits
	a.scannerHits += s.ScannerHits
	for d, n := range s.ByDay {
		a.byDay[d] += n
	}
	// byMonth is derived from byDay so it stays in sync with the day-range
	// folding logic for boundary files.
	for d, n := range s.ByDay {
		if len(d) >= 7 {
			a.byMonth[d[:7]] += n
		}
	}
	for k, n := range s.ByURL {
		a.byURL[k] += n
	}
	for k, n := range s.ByIP {
		a.byIP[k] += n
	}
	for k, n := range s.ByStatus {
		a.byStatus[k] += n
	}
	for k, n := range s.ByMethod {
		a.byMethod[k] += n
	}
	for k, n := range s.ByUA {
		a.byUA[k] += n
	}
	for k, n := range s.ByBotUA {
		a.byBotUA[k] += n
	}
	for k, n := range s.ByScannerPath {
		a.byScannerPath[k] += n
	}
}

// foldByDay only contributes byDay for days inside the range. URLs / IPs etc.
// are skipped for boundary files because we lack per-day breakdown for them.
// The caller (MergeSummaries) reserves this branch for files that straddle
// the requested range, which are typically a small minority of the total.
func foldByDay(a *Aggregator, s *FileSummary, fromDay, toDay string) {
	for d, n := range s.ByDay {
		if fromDay != "" && d < fromDay {
			continue
		}
		if toDay != "" && d > toDay {
			continue
		}
		a.byDay[d] += n
		a.totalHits += n
	}
}

// CollectSites returns the distinct site labels (derived from filenames)
// available under dir/pattern. It does not parse log lines, so it's cheap.
func CollectSites(dir, pattern string) ([]string, error) {
	files, err := ListFiles(dir, pattern, ReadFilter{})
	if err != nil {
		return nil, err
	}
	sites := make(map[string]struct{}, len(files))
	for _, fi := range files {
		sites[fi.Site] = struct{}{}
	}
	return sortedKeys(sites), nil
}
