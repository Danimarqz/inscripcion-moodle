package helpers

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// IsLetter returns true if the input is a single uppercase letter (A-Z).
func IsLetter(value string) bool {
	return len(value) == 1 && value[0] >= 'A' && value[0] <= 'Z'
}

// IsDigits returns true if every rune in the string is between '0' and '9'.
func IsDigits(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return len(value) > 0
}

// ToInt converts the numeric string into an integer.
func ToInt(value string) int {
	result := 0
	for i := 0; i < len(value); i++ {
		result = result*10 + int(value[i]-'0')
	}
	return result
}

// NormalizeName removes accents, normalizes spaces and returns the uppercase version of the string.
func NormalizeName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	// Remove accents and normalize spacing, uppercase result for strict compares.
	decomposed := norm.NFD.String(trimmed)
	var b strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if unicode.IsSpace(r) {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// MatchMaskedDNI reports whether a masked DNI/NIE (e.g. "****2611*") matches a
// full one (e.g. "78942611X"). It compares position by position: every
// alphanumeric character in the mask must equal the full value at that index,
// while '*' (and any other non-alphanumeric) positions act as wildcards.
// Matching is therefore independent of WHICH characters the official source
// chose to reveal, instead of assuming a fixed center window.
func MatchMaskedDNI(masked, full string) bool {
	masked = strings.ToUpper(strings.TrimSpace(masked))
	full = strings.ToUpper(strings.TrimSpace(full))

	if masked == full {
		return true
	}
	if len(masked) != len(full) {
		// Source mask malformed (wrong number of '*'), so position-by-position
		// alignment is impossible. Fall back to checking the revealed characters
		// appear contiguously in the full DNI. Callers gate this with an exact
		// name+surname match, so false positives are negligible.
		// ponytail: contiguous block only; make it position-aware if a short
		// mask ever reveals non-contiguous chars (e.g. "3*2*5").
		revealed := make([]byte, 0, len(masked))
		for i := 0; i < len(masked); i++ {
			m := masked[i]
			if (m >= '0' && m <= '9') || (m >= 'A' && m <= 'Z') {
				revealed = append(revealed, m)
			}
		}
		return len(revealed) > 0 && strings.Contains(full, string(revealed))
	}

	for i := 0; i < len(masked); i++ {
		m := masked[i]
		isAlphaNum := (m >= '0' && m <= '9') || (m >= 'A' && m <= 'Z')
		if !isAlphaNum {
			continue
		}
		if m != full[i] {
			return false
		}
	}
	return true
}

// CreateSlug generates a lowercase, alphanumeric slug without spaces.
func CreateSlug(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	decomposed := norm.NFD.String(trimmed)
	var b strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if unicode.IsSpace(r) {
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
