package colormod

import "os"

// --- color module ---

type Color struct{}

func colorize(code, s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

// Foreground colors

func (*Color) Red(s string) interface{}     { return colorize("31", s) }
func (*Color) Green(s string) interface{}   { return colorize("32", s) }
func (*Color) Yellow(s string) interface{}  { return colorize("33", s) }
func (*Color) Blue(s string) interface{}    { return colorize("34", s) }
func (*Color) Magenta(s string) interface{} { return colorize("35", s) }
func (*Color) Cyan(s string) interface{}    { return colorize("36", s) }
func (*Color) White(s string) interface{}   { return colorize("37", s) }
func (*Color) Gray(s string) interface{}    { return colorize("90", s) }

// Background colors

func (*Color) BgRed(s string) interface{}     { return colorize("41", s) }
func (*Color) BgGreen(s string) interface{}   { return colorize("42", s) }
func (*Color) BgYellow(s string) interface{}  { return colorize("43", s) }
func (*Color) BgBlue(s string) interface{}    { return colorize("44", s) }
func (*Color) BgMagenta(s string) interface{} { return colorize("45", s) }
func (*Color) BgCyan(s string) interface{}    { return colorize("46", s) }
func (*Color) BgWhite(s string) interface{}   { return colorize("47", s) }
func (*Color) BgGray(s string) interface{}    { return colorize("100", s) }

// Styles

func (*Color) Bold(s string) interface{}      { return colorize("1", s) }
func (*Color) Dim(s string) interface{}       { return colorize("2", s) }
func (*Color) Underline(s string) interface{} { return colorize("4", s) }
