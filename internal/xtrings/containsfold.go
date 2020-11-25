package xtrings

import (
	"strings"
)

func ContainsFold(s, substr string) bool {
	fc := rune(substr[0])
	for i, c := range s {
		end := i + len(substr)
		if end > len(s) {
			end = len(s)
		}

		if c == fc && strings.EqualFold(s[i:end], substr) {
			return true
		}
	}
	return false
}
