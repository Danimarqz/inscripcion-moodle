package logs

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ipHashSalt is set at process start from config; until then anonymizeIP uses
// an empty salt (still non-reversible, but correlatable between restarts only
// if the same data is fed). Production should always set IP_HASH_SALT.
var ipHashSalt = ""

// SetIPHashSalt configures the salt used when anonymizing client IPs. Call
// once at startup.
func SetIPHashSalt(salt string) { ipHashSalt = salt }

// LogEntry is a single parsed Apache combined log line.
type LogEntry struct {
	Time      time.Time
	IPPrefix  string
	Host      string // vhost from %v (may be empty if log uses plain combined)
	Site      string // logical site, derived from the source filename
	Method    string
	URL       string
	Status    int
	Bytes     int64
	Referer   string
	UserAgent string
}

// combinedLogRegex matches the Apache "combined" log format:
//
//	%h %l %u %t "%r" %>s %b "%{Referer}i" "%{User-agent}i"
//
// It also tolerates an optional vhost prefix (LogFormat "%v %h ..."), and an
// optional trailing duration field that some configurations append.
var combinedLogRegex = regexp.MustCompile(
	`^(?:(?P<vhost>\S+)\s+)?` +
		`(?P<ip>\S+)\s+\S+\s+\S+\s+` +
		`\[(?P<time>[^\]]+)\]\s+` +
		`"(?P<request>[^"]*)"\s+` +
		`(?P<status>\d{3})\s+` +
		`(?P<bytes>-|\d+)` +
		`(?:\s+"(?P<referer>[^"]*)"\s+"(?P<ua>[^"]*)")?`,
)

const apacheTimeLayout = "02/Jan/2006:15:04:05 -0700"

// ParseLine parses a single Apache combined-log line. Returns ok=false if the
// line does not match (blank lines, malformed entries, etc.).
func ParseLine(line string) (LogEntry, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return LogEntry{}, false
	}
	m := combinedLogRegex.FindStringSubmatch(line)
	if m == nil {
		return LogEntry{}, false
	}
	groups := make(map[string]string, len(combinedLogRegex.SubexpNames()))
	for i, name := range combinedLogRegex.SubexpNames() {
		if name != "" {
			groups[name] = m[i]
		}
	}

	t, err := time.Parse(apacheTimeLayout, groups["time"])
	if err != nil {
		return LogEntry{}, false
	}

	status, err := strconv.Atoi(groups["status"])
	if err != nil {
		return LogEntry{}, false
	}

	var bytes int64
	if b := groups["bytes"]; b != "" && b != "-" {
		bytes, _ = strconv.ParseInt(b, 10, 64)
	}

	method, url := splitRequest(groups["request"])

	return LogEntry{
		Time:      t,
		IPPrefix:  anonymizeIP(groups["ip"]),
		Host:      groups["vhost"],
		Method:    method,
		URL:       url,
		Status:    status,
		Bytes:     bytes,
		Referer:   groups["referer"],
		UserAgent: groups["ua"],
	}, true
}

func splitRequest(req string) (method, url string) {
	if req == "" {
		return "", ""
	}
	parts := strings.SplitN(req, " ", 3)
	if len(parts) < 2 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// anonymizeIP turns the client IP into an opaque, salted hash. The same IP
// always maps to the same hash (so top-N grouping works) but the original
// address cannot be recovered without the salt.
func anonymizeIP(raw string) string {
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(ipHashSalt + "\x00" + raw))
	return "anon-" + hex.EncodeToString(sum[:5])
}
