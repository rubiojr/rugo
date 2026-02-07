package colormod

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rubiojr/rugo/modules"
)

func TestModuleRegistration(t *testing.T) {
	m, ok := modules.Get("color")
	require.True(t, ok)
	assert.Equal(t, "Color", m.Type)
	assert.Len(t, m.Funcs, 19)
}

func TestForegroundColors(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := &Color{}

	assert.Equal(t, "\033[31mhello\033[0m", c.Red("hello"))
	assert.Equal(t, "\033[32mhello\033[0m", c.Green("hello"))
	assert.Equal(t, "\033[33mhello\033[0m", c.Yellow("hello"))
	assert.Equal(t, "\033[34mhello\033[0m", c.Blue("hello"))
	assert.Equal(t, "\033[35mhello\033[0m", c.Magenta("hello"))
	assert.Equal(t, "\033[36mhello\033[0m", c.Cyan("hello"))
	assert.Equal(t, "\033[37mhello\033[0m", c.White("hello"))
	assert.Equal(t, "\033[90mhello\033[0m", c.Gray("hello"))
}

func TestBackgroundColors(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := &Color{}

	assert.Equal(t, "\033[41mhello\033[0m", c.BgRed("hello"))
	assert.Equal(t, "\033[42mhello\033[0m", c.BgGreen("hello"))
	assert.Equal(t, "\033[43mhello\033[0m", c.BgYellow("hello"))
	assert.Equal(t, "\033[44mhello\033[0m", c.BgBlue("hello"))
	assert.Equal(t, "\033[45mhello\033[0m", c.BgMagenta("hello"))
	assert.Equal(t, "\033[46mhello\033[0m", c.BgCyan("hello"))
	assert.Equal(t, "\033[47mhello\033[0m", c.BgWhite("hello"))
	assert.Equal(t, "\033[100mhello\033[0m", c.BgGray("hello"))
}

func TestStyles(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := &Color{}

	assert.Equal(t, "\033[1mhello\033[0m", c.Bold("hello"))
	assert.Equal(t, "\033[2mhello\033[0m", c.Dim("hello"))
	assert.Equal(t, "\033[4mhello\033[0m", c.Underline("hello"))
}

func TestNoColor(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	c := &Color{}
	assert.Equal(t, "hello", c.Red("hello"))
	assert.Equal(t, "hello", c.BgBlue("hello"))
	assert.Equal(t, "hello", c.Bold("hello"))
}

func TestComposable(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := &Color{}

	result := c.Bold(c.Red("err").(string))
	assert.Equal(t, "\033[1m\033[31merr\033[0m\033[0m", result)
}

func TestEmptyString(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := &Color{}

	assert.Equal(t, "\033[31m\033[0m", c.Red(""))
}
