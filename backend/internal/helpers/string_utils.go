package helpers

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
