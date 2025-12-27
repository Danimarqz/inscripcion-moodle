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
