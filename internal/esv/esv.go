package esv

import (
	"fmt"
	"math/rand"

	"github.com/zostay/go-esv-api/pkg/esv"

	"github.com/zostay/dotfiles-go/internal/keeper"
)

func RandomBook() Book {
	return Books[rand.Int()%len(Books)] //nolint:gosec // weak random is fine here
}

func randomBook() *Book {
	return &Books[rand.Int()%len(Books)] //nolint:gosec // weak random is fine here
}

func RandomPassage(b Book) []VerseRef {
	x := rand.Int() % len(b.verses) //nolint:gosec // weak random is fine here
	o := rand.Int() % 30            //nolint:gosec // weak random is fine here
	y := x + o
	if y >= len(b.verses) {
		y = len(b.verses) - 1
	}

	return b.verses[x:y]
}

func randomPassage(b *Book) []VerseRef {
	x := rand.Int() % len(b.verses) //nolint:gosec // weak random is fine here
	o := rand.Int() % 30            //nolint:gosec // weak random is fine here
	y := x + o
	if y >= len(b.verses) {
		y = len(b.verses) - 1
	}

	return b.verses[x:y]
}

func randomPassageFromExtract(b *BookExtract) []VerseRef {
	x := rand.Int() % len(b.Verses()) //nolint:gosec // weak random is fine here
	o := rand.Int() % 30              //nolint:gosec // weak random is fine here
	y := x + o
	if y >= len(b.verses) {
		y = len(b.verses) - 1
	}

	return b.verses[x:y]
}

type randomOpts struct {
	category string
	book     string
}

type RandomReferenceOption func(*randomOpts)

func FromBook(name string) RandomReferenceOption {
	return func(o *randomOpts) {
		o.book = name
	}
}

func FromCategory(name string) RandomReferenceOption {
	return func(o *randomOpts) {
		o.category = name
	}
}

// Random pulls a random reference from the Bible and returns it. You can use the
// options to help narrow down where the passages are selected from.
func Random(opt ...RandomReferenceOption) (string, error) {
	o := &randomOpts{}
	for _, f := range opt {
		f(o)
	}

	var (
		b  *Book
		vs []VerseRef
	)
	if o.category != "" {
		exs, err := LookupCategory(o.category)
		if err != nil {
			return "", err
		}

		// lazy way to weight the books by the number of verses they have
		bag := make([]*BookExtract, 0, len(exs))
		for i := range exs {
			for range exs[i].Verses() {
				bag = append(bag, &exs[i])
			}
		}

		be := bag[rand.Int()%len(bag)] //nolint:gosec // weak random is fine here
		b = be.Book
		vs = randomPassageFromExtract(be)
	} else {
		if o.book != "" {
			ex, err := LookupBook(o.book)
			if err != nil {
				return "", err
			}

			b = ex.Book
		} else {
			b = randomBook()
		}

		vs = randomPassage(b)
	}

	v1, v2 := vs[0], vs[len(vs)-1]

	if len(vs) > 1 {
		return fmt.Sprintf("%s %s-%s", b.name, v1.Ref(), v2.Ref()), nil
	}

	return fmt.Sprintf("%s %s", b.name, v1.Ref()), nil
}

// RandomReference pulls a random reference from the Bible and returns it.
// Deprecated: Use Random() instead.
func RandomReference() string {
	b := RandomBook()
	vs := RandomPassage(b)

	v1 := vs[0]
	v2 := vs[len(vs)-1]

	if len(vs) > 1 {
		return fmt.Sprintf("%s %s-%s", b.name, v1.Ref(), v2.Ref())
	} else {
		return fmt.Sprintf("%s %s", b.name, v1.Ref())
	}
}

func RandomVerse(opt ...RandomReferenceOption) (string, error) {
	ref, err := Random(opt...)
	if err != nil {
		return "", err
	}
	return GetVerse(ref)
}

func RandomVerseHTML(opt ...RandomReferenceOption) (string, error) {
	ref, err := Random(opt...)
	if err != nil {
		return "", err
	}
	return GetVerseHTML(ref)
}

func GetVerse(ref string) (string, error) {
	token, err := keeper.GetSecret("ESV_API_TOKEN")
	if err != nil {
		return "", err
	}

	c := esv.New(token.Password())
	tr, err := c.PassageText(ref,
		esv.WithIncludeVerseNumbers(false),
		esv.WithIncludeHeadings(false),
		esv.WithIncludeFootnotes(false),
	)
	if err != nil {
		return "", err
	}

	return tr.Passages[0], nil
}

func GetVerseHTML(ref string) (string, error) {
	token, err := keeper.GetSecret("ESV_API_TOKEN")
	if err != nil {
		return "", err
	}

	c := esv.New(token.Password())
	tr, err := c.PassageHtml(ref,
		esv.WithIncludeVerseNumbers(false),
		esv.WithIncludeHeadings(false),
		esv.WithIncludeFootnotes(false),
	)
	if err != nil {
		return "", err
	}

	return tr.Passages[0], nil
}
