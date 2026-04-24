package admin

import "strings"

const maxQuestionLabelLen = 32

func normalizeLabel(l *string) *string {
	if l == nil {
		return nil
	}
	s := strings.TrimSpace(*l)
	if s == "" {
		return nil
	}
	if len([]rune(s)) > maxQuestionLabelLen {
		s = string([]rune(s)[:maxQuestionLabelLen])
	}
	return &s
}
