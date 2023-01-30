package mail

import (
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func withColor(f func()) {
	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()
	f()
}

func TestColorPalette_Fcolor(t *testing.T) { //nolint:paralleltest // changes globals
	withColor(func() {
		buf := &strings.Builder{}
		cp.Fcolor(buf, "reading", "one", "dropping", "two")
		const expect = "\x1b[95mone\x1b[0m\x1b[93mtwo\x1b[0m"
		assert.Equal(t, expect, buf.String())
	})
}

func TestColorPalette_Fprintf(t *testing.T) { //nolint:paralleltest // changes globals
	withColor(func() {
		buf := &strings.Builder{}
		cp.Fprintf("forwarding", buf, "test %d", 42)
		const expect = "\x1b[93mtest 42\x1b[0m"
		assert.Equal(t, expect, buf.String())
	})
}

func TestColorPalette_Sprintf(t *testing.T) { //nolint:paralleltest // changes globals
	withColor(func() {
		s := cp.Sprintf("fail", "test %d", 42)
		const expect = "\x1b[91mtest 42\x1b[0m"
		assert.Equal(t, expect, s)
	})
}

func TestColorPalette_Join(t *testing.T) { //nolint:paralleltest // changes globals
	withColor(func() {
		s := cp.Join("label", []string{"one", "two", "three"}, ",")
		const expect = "one\x1b[34m,\x1b[0mtwo\x1b[34m,\x1b[0mthree"
		assert.Equal(t, expect, s)
	})
}
