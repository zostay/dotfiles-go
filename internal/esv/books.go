package esv

import (
	"strconv"
)

type Book struct {
	name   string
	verses []VerseRef
}

type VerseRef interface {
	Ref() string
}

type ChapterVerse struct {
	chapter int
	verse   int
}

func (v *ChapterVerse) Ref() string {
	return strconv.Itoa(v.chapter) + ":" + strconv.Itoa(v.verse)
}

type JustVerse struct {
	verse int
}

func (v *JustVerse) Ref() string {
	return strconv.Itoa(v.verse)
}
