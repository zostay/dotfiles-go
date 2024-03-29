package esv_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zostay/dotfiles-go/internal/esv"
)

var parseVerseRefInputs = []struct {
	input        string
	justVerse    bool
	usesWildcard bool
	expect       esv.VerseRef
}{
	{"", false, false, nil},
	{"", true, false, nil},
	{"1:2", false, false, esv.NewChapterVerse(1, 2)},
	{"3:16", false, false, esv.NewChapterVerse(3, 16)},
	{"102:*", false, true, esv.NewChapterVerse(102, esv.Final)},
	{"*:1", false, true, nil},
	{"*:*", false, true, esv.NewChapterVerse(esv.Final, esv.Final)},
	{"*", true, true, esv.NewJustVerse(esv.Final)},
	{"*:1", false, true, nil},
}

func TestParseVerseRef(t *testing.T) {
	t.Parallel()

	// no options
	for _, input := range parseVerseRefInputs {
		v, err := esv.ParseVerseRef(input.input)
		if input.expect == nil {
			t.Run("no options with bad input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else if input.usesWildcard {
			t.Run("no options with wildcard input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else {
			t.Run("no options with good input: "+input.input, func(t *testing.T) {
				assert.NoError(t, err)
				assert.Equal(t, input.expect, v)
			})
		}
	}

	// AllowWildcard
	for _, input := range parseVerseRefInputs {
		v, err := esv.ParseVerseRef(input.input, esv.AllowWildcard)
		if input.expect == nil {
			t.Run("AllowWildcard with bad input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else {
			t.Run("AllowWildcard with good input: "+input.input, func(t *testing.T) {
				assert.NoError(t, err)
				assert.Equal(t, input.expect, v)
			})
		}
	}

	// ExpectChapterAndVerse
	for _, input := range parseVerseRefInputs {
		v, err := esv.ParseVerseRef(input.input, esv.ExpectChapterAndVerse)
		if input.expect == nil {
			t.Run("ExpectChapterAndVerse with bad input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else if input.justVerse {
			t.Run("ExpectChapterAndVerse with just verse input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else if input.usesWildcard {
			t.Run("ExpectChapterAndVerse with wildcard input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else {
			t.Run("ExpectChapterAndVerse with good input: "+input.input, func(t *testing.T) {
				assert.NoError(t, err)
				assert.Equal(t, input.expect, v)
			})
		}
	}

	// ExpectJustVerse
	for _, input := range parseVerseRefInputs {
		v, err := esv.ParseVerseRef(input.input, esv.ExpectJustVerse)
		if input.expect == nil {
			t.Run("ExpectJustVerse with bad input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else if !input.justVerse {
			t.Run("ExpectJustVerse with chapter and verse input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else if input.usesWildcard {
			t.Run("ExpectJustVerse with wildcard input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else {
			t.Run("ExpectJustVerse with good input: "+input.input, func(t *testing.T) {
				assert.NoError(t, err)
				assert.Equal(t, input.expect, v)
			})
		}
	}

	// AllowWildcard, ExpectChapterAndVerse
	for _, input := range parseVerseRefInputs {
		v, err := esv.ParseVerseRef(input.input, esv.AllowWildcard, esv.ExpectChapterAndVerse)
		if input.expect == nil {
			t.Run("AllowWildcard+ExpectChapterAndVerse with bad input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else if input.justVerse {
			t.Run("AllowWildcard+ExpectChapterAndVerse with just verse input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else {
			t.Run("AllowWildcard+ExpectChapterAndVerse with good input: "+input.input, func(t *testing.T) {
				assert.NoError(t, err)
				assert.Equal(t, input.expect, v)
			})
		}
	}

	// AllowWildcard, ExpectJustVerse
	for _, input := range parseVerseRefInputs {
		v, err := esv.ParseVerseRef(input.input, esv.AllowWildcard, esv.ExpectJustVerse)
		if input.expect == nil {
			t.Run("AllowWildcard+ExpectJustVerse with bad input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else if !input.justVerse {
			t.Run("AllowWildcard+ExpectJustVerse with chapter and verse input: "+input.input, func(t *testing.T) {
				assert.Error(t, err)
				assert.Nil(t, v)
			})
		} else {
			t.Run("AllowWildcard+ExpectJustVerse with good input: "+input.input, func(t *testing.T) {
				assert.NoError(t, err)
				assert.Equal(t, input.expect, v)
			})
		}
	}
}
