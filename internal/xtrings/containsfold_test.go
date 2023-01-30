package xtrings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsFold(t *testing.T) {
	t.Parallel()

	var data = []struct {
		s      string
		substr string
		pass   bool
	}{
		{"abc", "abc", true},
		{"asdfabcasdf", "abc", true},
		{"asdfasdf", "abc", false},
		{"ABC", "abc", true},
		{"asdfABCasdf", "abc", true},
		{"abc", "ABC", true},
		{"asdfabcasfd", "ABC", true},
		{"ASDFabcASDF", "ABC", true},
		{"ASDFABCASDF", "ABC", true},
		{"abc", "ABCabc", false},
	}

	for _, test := range data {
		assert.Equalf(
			t,
			test.pass,
			ContainsFold(test.s, test.substr),
			"%q contains %q",
			test.s,
			test.substr,
		)
	}
}
