package xtrings

import (
	"strings"
)

func ContainsFold(s, substr string) bool {
	fc := rune(substr[0])
	for i, c := range s {
		if c == fc && strings.EqualFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}
