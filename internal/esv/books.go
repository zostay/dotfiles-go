package esv

import (
	"errors"
	"strconv"
)

var (
	ErrNotFound = errors.New("scripture reference not found")
	ErrMismatch = errors.New("scripture reference mismatch (chapter/verse vs. verse)")
	ErrReversed = errors.New("scripture reference is reversed")
)

const Final = -1

type Book struct {
	name      string
	justVerse bool
	verses    []VerseRef
}

func (b *Book) Name() string {
	return b.name
}

type WildcardType int

const (
	WildcardNone WildcardType = iota
	WildcardChapter
	WildcardVerse
)

type VerseRef interface {
	Ref() string
	Wildcard() WildcardType
	Before(VerseRef) bool
	Equal(VerseRef) bool
}

type ChapterVerse struct {
	chapter int
	verse   int
}

func NewChapterVerse(chapter, verse int) *ChapterVerse {
	return &ChapterVerse{
		chapter: chapter,
		verse:   verse,
	}
}

func (v *ChapterVerse) Ref() string {
	return strconv.Itoa(v.chapter) + ":" + strconv.Itoa(v.verse)
}

func (v *ChapterVerse) Wildcard() WildcardType {
	if v.chapter == -1 {
		return WildcardChapter
	}
	if v.verse == -1 {
		return WildcardVerse
	}
	return WildcardNone
}

func (v *ChapterVerse) Before(ov VerseRef) bool {
	return v.chapter < ov.(*ChapterVerse).chapter || (v.chapter == ov.(*ChapterVerse).chapter && v.verse < ov.(*ChapterVerse).verse)
}

func (v *ChapterVerse) Equal(ov VerseRef) bool {
	return v.chapter == ov.(*ChapterVerse).chapter && v.verse == ov.(*ChapterVerse).verse
}

type JustVerse struct {
	verse int
}

func NewJustVerse(verse int) *JustVerse {
	return &JustVerse{
		verse: verse,
	}
}

func (v *JustVerse) Ref() string {
	return strconv.Itoa(v.verse)
}

func (v *JustVerse) Wildcard() WildcardType {
	if v.verse == -1 {
		return WildcardVerse
	}
	return WildcardNone
}

func (v *JustVerse) Before(ov VerseRef) bool {
	return v.verse < ov.(*JustVerse).verse
}

func (v *JustVerse) Equal(ov VerseRef) bool {
	return v.verse == ov.(*JustVerse).verse
}

type BookExtract struct {
	*Book
	First VerseRef
	Last  VerseRef
}

func (e *BookExtract) FullRef() string {
	return e.Book.name + " " + e.First.Ref() + "-" + e.Last.Ref()
}

func (e *BookExtract) Verses() []VerseRef {
	verses := make([]VerseRef, 0, len(e.Book.verses))
	started := false
	for _, verse := range e.Book.verses {
		if verse == e.First {
			started = true
		}
		if started {
			verses = append(verses, verse)
		}
		if verse == e.Last {
			break
		}
	}

	return verses
}
