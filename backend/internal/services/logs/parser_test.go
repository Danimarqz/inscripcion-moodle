package logs

import (
	"strings"
	"testing"
)

func TestParseLine_Combined(t *testing.T) {
	line := `203.0.113.42 - - [24/Apr/2026:11:00:01 +0200] "GET /exam/foo HTTP/1.1" 200 1532 "https://example.com/" "Mozilla/5.0 (X11)"`
	entry, ok := ParseLine(line)
	if !ok {
		t.Fatalf("expected line to parse")
	}
	if !strings.HasPrefix(entry.IPPrefix, "anon-") || len(entry.IPPrefix) != len("anon-")+10 {
		t.Errorf("ip prefix = %q, want anon-<10 hex>", entry.IPPrefix)
	}
	again, _ := ParseLine(line)
	if again.IPPrefix != entry.IPPrefix {
		t.Errorf("hash not stable: %q vs %q", entry.IPPrefix, again.IPPrefix)
	}
	if entry.Method != "GET" || entry.URL != "/exam/foo" {
		t.Errorf("request parsed wrong: %q %q", entry.Method, entry.URL)
	}
	if entry.Status != 200 || entry.Bytes != 1532 {
		t.Errorf("status/bytes wrong: %d %d", entry.Status, entry.Bytes)
	}
	if entry.Time.Year() != 2026 || entry.Time.Month() != 4 || entry.Time.Day() != 24 {
		t.Errorf("time parsed wrong: %v", entry.Time)
	}
}

func TestParseLine_VhostPrefix(t *testing.T) {
	line := `exam.example.com 198.51.100.7 - - [01/May/2026:08:30:00 +0000] "POST /api/login HTTP/1.1" 401 0 "-" "curl/8.0"`
	entry, ok := ParseLine(line)
	if !ok {
		t.Fatalf("expected line to parse")
	}
	if entry.Host != "exam.example.com" {
		t.Errorf("host = %q, want exam.example.com", entry.Host)
	}
	if !strings.HasPrefix(entry.IPPrefix, "anon-") {
		t.Errorf("ip prefix = %q", entry.IPPrefix)
	}
	if entry.Status != 401 {
		t.Errorf("status = %d", entry.Status)
	}
}

func TestParseLine_IPv6(t *testing.T) {
	line := `2001:db8:abcd:1234::1 - - [01/May/2026:00:00:00 +0000] "GET / HTTP/1.1" 200 100 "-" "-"`
	entry, ok := ParseLine(line)
	if !ok {
		t.Fatalf("expected line to parse")
	}
	if !strings.HasPrefix(entry.IPPrefix, "anon-") {
		t.Errorf("ipv6 prefix = %q", entry.IPPrefix)
	}
}

func TestParseLine_DashBytes(t *testing.T) {
	line := `10.0.0.1 - - [01/May/2026:00:00:00 +0000] "GET / HTTP/1.1" 304 - "-" "-"`
	entry, ok := ParseLine(line)
	if !ok {
		t.Fatalf("expected line to parse")
	}
	if entry.Bytes != 0 {
		t.Errorf("bytes = %d, want 0", entry.Bytes)
	}
}

func TestParseLine_Malformed(t *testing.T) {
	if _, ok := ParseLine(""); ok {
		t.Error("blank line should not parse")
	}
	if _, ok := ParseLine("garbage line with no structure"); ok {
		t.Error("garbage should not parse")
	}
}
