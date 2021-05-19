package mail

import (
	"io"
	"strings"

	"github.com/fatih/color"
)

type ColorPalette map[string]*color.Color

var (
	cp = ColorPalette{
		"base":   color.New(color.FgHiBlack),
		"meh":    color.New(color.FgHiBlack),
		"fail":   color.New(color.FgHiRed),
		"warn":   color.New(color.FgHiYellow),
		"pass":   color.New(color.FgHiGreen),
		"header": color.New(color.FgMagenta),
		"action": color.New(color.FgWhite),
		"label":  color.New(color.FgBlue),
		"value":  color.New(color.FgCyan),
		"file":   color.New(color.FgWhite),

		"reading":    color.New(color.FgHiMagenta),
		"labeling":   color.New(color.FgHiGreen),
		"forwarding": color.New(color.FgHiYellow),
		"moving":     color.New(color.FgHiCyan),
		"clearing":   color.New(color.FgHiBlue),
		"dropping":   color.New(color.FgHiYellow),
		"searching":  color.New(color.FgHiMagenta),
		"fixing":     color.New(color.FgHiCyan),
	}
)

func (cp ColorPalette) Join(color string, args []string, d string) string {
	if c, ok := cp[color]; ok {
		return strings.Join(args, c.Sprint(d))
	} else {
		panic("unknown color " + color)
	}
}

func (cp ColorPalette) Fcolor(out io.Writer, args ...string) {
	color := "base"
	for i, v := range args {
		if i%2 == 0 {
			color = v
		} else {
			if c, ok := cp[color]; ok {
				c.Fprint(out, v)
			} else {
				panic("unknown color " + color)
			}
		}
	}
}

func (cp ColorPalette) Scolor(args ...string) string {
	var out strings.Builder
	color := "base"
	for i, v := range args {
		if i%2 == 0 {
			color = v
		} else {
			if c, ok := cp[color]; ok {
				c.Fprint(&out, v)
			} else {
				panic("unknown color " + color)
			}
		}
	}
	return out.String()
}

func (cp ColorPalette) Fprintf(color string, out io.Writer, fmt string, args ...interface{}) {
	if c, ok := cp[color]; ok {
		c.Fprintf(out, fmt, args...)
		return
	}
	panic("unknown color " + color)
}

func (cp ColorPalette) Sprintf(color string, fmt string, args ...interface{}) string {
	if c, ok := cp[color]; ok {
		return c.Sprintf(fmt, args...)
	}
	panic("unknown color " + color)
}
