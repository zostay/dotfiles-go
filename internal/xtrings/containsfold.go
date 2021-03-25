// Package xtrings contains some specialized string tools for use in my
// dotfiles.
package xtrings

import (
	"unicode"
	"unicode/utf8"
)

// ContainsFold looks for substrings in a case insensitive way. It attempts to
// be fast.
func ContainsFold(s, substr string) bool {
	//fmt.Printf("%q == %q ?\n", s, substr)

	// This code borrows bits from strings.EqualFold()

	// All strings contain an empty substring
	if substr == "" {
		return true
	}

	// If the substr wasn't empty, then it won't be found in an empty string
	if s == "" {
		return false
	}

	// Be sure we have a full list of decoded runes to work with from the substr
	subr := make([]rune, 0, len(substr))
	for substr != "" {
		var r rune
		if substr[0] < utf8.RuneSelf {
			r, substr = rune(substr[0]), substr[1:]
		} else {
			dr, size := utf8.DecodeRuneInString(substr)
			r, substr = dr, substr[size:]
		}

		subr = append(subr, r)
	}

	// Hunt until there's nothing left to search
	matching := 0
	for s != "" {
		// matched all? Happy!
		if matching >= len(subr) {
			return true
		}

		// Grab the first char of what's remaining to compare it to the first
		// char of what we're looking for
		var tc rune
		if s[0] < utf8.RuneSelf {
			tc, s = rune(s[0]), s[1:]
		} else {
			t, size := utf8.DecodeRuneInString(s)
			tc, s = t, s[size:]
		}

		//fmt.Printf("CHECK %c == %c ? (%d)\n", subr[matching], tc, matching)

		// If the runes are equal, we have a match
		if subr[matching] == tc {
			matching++
			continue
		}

		// They might be fold-equal, but they aren't identical. If they're
		// equal, the math we're about to do is simpler if we identify one as
		// the lower codepoint. We'll make the character we're searching with
		// the lower codepoint. (If they aren't equal, it won't matter what we
		// do here. The downside is that we're going to iterate through folded
		// chars on one of these and one might have more fold chars than the
		// other. Oh well.)
		sr := subr[matching]
		if tc < sr {
			tc, sr = sr, tc
		}

		// ASCII? Take a short cut
		if tc < utf8.RuneSelf {
			if 'A' <= sr && sr <= 'Z' && tc == sr+'a'-'A' {
				matching++
				continue
			}
		}

		// If there are no fold chars for this, the loop never iterates and we
		// have the same char as before. At which point, the comparison will
		// fail and we'll move on to the next char.
		//
		// If there are fold chars for this, it will iterate at least once and
		// maybe multiple times. It will iterate until it finds a fold char that
		// is equal to or greater than the char we're looking for (noting that
		// we may have swapped these items a moment ago, but fold equivalence
		// testing works either way.)
		r := unicode.SimpleFold(sr)
		for r != sr && r < tc {
			r = unicode.SimpleFold(r)
		}

		// If we ended up with equal rather than greater than, this will pass.
		if r == tc {
			matching++
			continue
		}

		// still here? no match
		matching = 0
	}

	return matching >= len(subr)
}
