package mail_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zostay/dotfiles-go/internal/mail"
)

func TestActionsSummary_String(t *testing.T) {
	as := mail.ActionsSummary{
		"one":   42,
		"two":   13,
		"three": 100,
	}

	const expected = ` one   :  42
 three : 100
 two   :  13
------- -----
 Total : 155
`

	assert.Equal(t, expected, as.String())
}
