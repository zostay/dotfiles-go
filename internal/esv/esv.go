package esv

import (
	"fmt"
	"math/rand"

	"github.com/zostay/go-esv-api/pkg/esv"

	"github.com/zostay/dotfiles-go/pkg/secrets"
)

func RandomBook() Book {
	return Books[rand.Int()%len(Books)]
}

func RandomPassage(b Book) []VerseRef {
	x := rand.Int() % len(b.verses)
	o := rand.Int() % 30
	y := x + o
	if y >= len(b.verses) {
		y = len(b.verses) - 1
	}

	return b.verses[x:y]
}

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

func RandomVerse() (string, error) {
	ref := RandomReference()
	return GetVerse(ref)
}

func GetVerse(ref string) (string, error) {
	s, err := secrets.Insecure()
	if err != nil {
		return "", err
	}

	token, err := s.GetSecret("ESV_API_TOKEN")
	if err != nil {
		return "", err
	}

	c := esv.New(token.Value)
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
