// Package xtrings contains some specialized string tools for use in my
// dotfiles.
package xtrings

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ContainsFold looks for substrings in a case insensitive way. It attempts to
// be fast.
func ContainsFold(s, substr string) bool {
	//fmt.Printf("%q contains %q ??\n", s, substr)
	// This code borrows bits from strings.EqualFold()

	// All strings contain an empty substring
	if substr == "" {
		return true
	}

	// If the substr wasn't empty, then it won't be found in an empty string
	if s == "" {
		return false
	}

	// If the substr is larger than the string, we ain't matchin' that either.
	if len(s) < len(substr) {
		return false
	}

	// Find a char that we can use to identify the first char we're looking for
	// in the string.
	var ofc rune
	if substr[0] < utf8.RuneSelf {
		ofc, substr = rune(substr[0]), substr[1:]
	} else {
		f, size := utf8.DecodeRuneInString(substr)
		ofc, substr = f, substr[size:]
	}

	sl := len(substr)

	// Hunt until there's nothing left to search
	for s != "" {
		fc := ofc

		// Grabe the first char of what's remaining to compare it to the first
		// char of what we're looking for
		var tc rune
		if s[0] < utf8.RuneSelf {
			tc, s = rune(s[0]), s[1:]
		} else {
			t, size := utf8.DecodeRuneInString(s)
			tc, s = t, s[size:]
		}

		// Set this to true, if it's a match
		try := false

		// If the runes are equal, we have a match
		if fc == tc {
			//fmt.Printf("%c == %c\n", fc, tc)
			try = true
		}

		// They might be fold-equal, but they aren't identical. If they're
		// equal, the math we're about to do is simpler if we identify one as
		// the lower codepoint. We'll make the character we're searching with
		// the lower codepoint. (If they aren't equal, it won't matter what we
		// do here. The downside is that we're going to iterate through folded
		// chars on one of these and one might have more fold chars than the
		// other. Oh well.)
		if !try && tc < fc {
			tc, fc = fc, tc
		}

		// ASCII? Take a short cut
		if !try && tc < utf8.RuneSelf {
			if 'A' <= fc && fc <= 'Z' && tc == fc+'a'-'A' {
				//fmt.Printf("%c ~= %c (ASCII shortcut)\n", fc, tc)
				try = true
			} else {
				// If it's ASCII and we didn't match, we're not going to, so move on
				// to the next char.
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
		if !try {
			f := unicode.SimpleFold(fc)
			for f != fc && f < tc {
				f = unicode.SimpleFold(f)
			}

			// If we ended up with equal rather than greater than, this will pass.
			if f == tc {
				try = true
			}
		}

		// We've flagged it as a try, so let's give it a shot using EqualFold on
		// the substring of the remainder.
		if try && strings.EqualFold(s[:sl], substr) {
			//fmt.Printf("%q ~= %q\n", s[:sl], substr)
			return true
			//} else if try {
			//	fmt.Printf("%q !~= %q\n", s[:sl], substr)
			//} else {
			//	fmt.Printf("HERE?\n")
		}
	}

	// We ran out of string to test: no joy.
	return false
}
