package esv

import "fmt"

func LookupBook(name string) (BookExtract, error) {
	for i := range Books {
		b := &Books[i]
		if b.name == name {
			return BookExtract{
				Book:  b,
				First: b.verses[0],
				Last:  b.verses[len(b.verses)-1],
			}, nil
		}
	}
	return BookExtract{}, fmt.Errorf("%w: %s", ErrNotFound, name)
}

func MustLookupBook(name string) BookExtract {
	b, err := LookupBook(name)
	if err != nil {
		panic(err)
	}
	return b
}

func LookupBookExtract(name, first, last string) (BookExtract, error) {
	b, err := LookupBook(name)
	if err != nil {
		return BookExtract{}, err
	}

	opt := &parseVerseOpts{
		expectedRefType: expectEither,
		allowWildcard:   false,
	}
	if b.justVerse {
		opt.expectedRefType = expectJustVerse
	} else {
		opt.expectedRefType = expectChapterAndVerse
	}

	fv, err := parseVerseRef(first, opt)
	if err != nil {
		return BookExtract{}, fmt.Errorf("unable to parse first ref for extract: %w", err)
	}

	opt.allowWildcard = true
	lv, err := parseVerseRef(last, opt)
	if err != nil {
		return BookExtract{}, fmt.Errorf("unable to parse last ref for extract: %w", err)
	}

	fvi := 0
	for i, verse := range b.verses {
		if verse.Equal(fv) {
			fvi = i
			break
		}
	}

	if fvi == 0 {
		return BookExtract{}, fmt.Errorf("%w: %s %s", ErrNotFound, name, first)
	}

	b.First = fv

	if lv.Wildcard() != WildcardNone {
		if b.justVerse || lv.Wildcard() == WildcardChapter {
			b.Last = b.verses[len(b.verses)-1]
			return b, nil
		}

		if lv.Wildcard() == WildcardVerse {
			b.Last = b.First
			for i := fvi; i < len(b.verses); i++ {
				if b.verses[i].(*ChapterVerse).chapter != fv.(*ChapterVerse).chapter {
					break
				}
				b.Last = b.verses[i]
			}
			return b, nil
		}
	}

	for i := fvi; i < len(b.verses); i++ {
		if b.verses[i].Equal(lv) {
			b.First = fv
			b.Last = lv
			return b, nil
		}
	}

	if !fv.Before(lv) {
		return BookExtract{}, fmt.Errorf("%w: %s is before %s", ErrReversed, last, first)
	}

	return BookExtract{}, fmt.Errorf("%w: %s %s not found", ErrNotFound, name, last)
}

func MustLookupBookExtract(name, first, last string) BookExtract {
	b, err := LookupBookExtract(name, first, last)
	if err != nil {
		panic(err)
	}
	return b
}

func LookupCategory(name string) ([]BookExtract, error) {
	exs, ok := Categories[name]
	if !ok {
		return nil, ErrNotFound
	}
	return exs, nil
}

func MustLookupCategory(name string) []BookExtract {
	b, err := LookupCategory(name)
	if err != nil {
		panic(err)
	}
	return b
}
