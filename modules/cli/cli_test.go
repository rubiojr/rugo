package climod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rubiojr/rugo/modules"
)

func TestModuleRegistration(t *testing.T) {
	m, ok := modules.Get("cli")
	require.True(t, ok, "cli module should be registered")
	assert.Equal(t, "cli", m.Name)
	assert.Equal(t, "CLI", m.Type)
	assert.Equal(t, "run", m.DispatchEntry)
	assert.NotEmpty(t, m.Runtime)

	funcNames := make(map[string]bool)
	for _, f := range m.Funcs {
		funcNames[f.Name] = true
	}
	for _, name := range []string{"name", "version", "about", "cmd", "flag", "bool_flag", "run", "parse", "command", "get", "args", "help"} {
		assert.True(t, funcNames[name], "missing function: %s", name)
	}
}

func newCLI() *CLI {
	return &CLI{}
}

func TestMetadata(t *testing.T) {
	c := newCLI()
	c.Name("myapp")
	c.Version("1.0.0")
	c.About("test app")

	assert.Equal(t, "myapp", c.appName)
	assert.Equal(t, "1.0.0", c.appVersion)
	assert.Equal(t, "test app", c.appAbout)
}

func TestCmdRegistration(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")
	c.Cmd("goodbye", "Say goodbye")

	require.Len(t, c.commands, 2)
	assert.Equal(t, "hello", c.commands[0].Name)
	assert.Equal(t, "goodbye", c.commands[1].Name)
}

func TestFlagRegistration(t *testing.T) {
	c := newCLI()
	c.Flag("hello", "name", "n", "Name to greet", "World")
	c.BoolFlag("hello", "verbose", "v", "Verbose output")

	require.Len(t, c.flags, 2)
	assert.Equal(t, "name", c.flags[0].Long)
	assert.Equal(t, "n", c.flags[0].Short)
	assert.Equal(t, "World", c.flags[0].Default)
	assert.False(t, c.flags[0].IsBool)
	assert.True(t, c.flags[1].IsBool)
}

func TestParseSimpleCommand(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")
	c.parseArgs([]string{"hello"})

	assert.Equal(t, "hello", c.matched)
	assert.Empty(t, c.remaining)
}

func TestParseCommandWithStringFlag(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")
	c.Flag("hello", "name", "n", "Name", "World")
	c.parseArgs([]string{"hello", "-n", "Rugo"})

	assert.Equal(t, "hello", c.matched)
	assert.Equal(t, "Rugo", c.values["name"])
}

func TestParseCommandWithLongFlag(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")
	c.Flag("hello", "name", "n", "Name", "World")
	c.parseArgs([]string{"hello", "--name", "Developer"})

	assert.Equal(t, "Developer", c.values["name"])
}

func TestParseBoolFlag(t *testing.T) {
	c := newCLI()
	c.Cmd("serve", "Start server")
	c.BoolFlag("serve", "verbose", "v", "Verbose")
	c.parseArgs([]string{"serve", "-v"})

	assert.Equal(t, "true", c.values["verbose"])
}

func TestParseBoolFlagDefault(t *testing.T) {
	c := newCLI()
	c.Cmd("serve", "Start server")
	c.BoolFlag("serve", "verbose", "v", "Verbose")
	c.parseArgs([]string{"serve"})

	_, exists := c.values["verbose"]
	assert.False(t, exists, "bool flag should not have a default value")
}

func TestParseDefaultValue(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")
	c.Flag("hello", "name", "n", "Name", "World")
	c.parseArgs([]string{"hello"})

	assert.Equal(t, "World", c.values["name"])
}

func TestParsePositionalArgs(t *testing.T) {
	c := newCLI()
	c.Cmd("add", "Add item")
	c.parseArgs([]string{"add", "foo", "bar", "baz"})

	assert.Equal(t, "add", c.matched)
	require.Len(t, c.remaining, 3)
	assert.Equal(t, "foo", c.remaining[0])
	assert.Equal(t, "bar", c.remaining[1])
	assert.Equal(t, "baz", c.remaining[2])
}

func TestParseMixedFlagsAndArgs(t *testing.T) {
	c := newCLI()
	c.Cmd("add", "Add item")
	c.Flag("add", "priority", "p", "Priority", "normal")
	c.parseArgs([]string{"add", "-p", "high", "my-task"})

	assert.Equal(t, "high", c.values["priority"])
	require.Len(t, c.remaining, 1)
	assert.Equal(t, "my-task", c.remaining[0])
}

func TestParseDoubleDash(t *testing.T) {
	c := newCLI()
	c.Cmd("run", "Run cmd")
	c.parseArgs([]string{"run", "--", "--not-a-flag", "arg"})

	assert.Equal(t, "run", c.matched)
	require.Len(t, c.remaining, 2)
	assert.Equal(t, "--not-a-flag", c.remaining[0])
}

func TestParseNoCommand(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")
	c.parseArgs([]string{})

	assert.Equal(t, "", c.matched)
}

func TestMatchCommandSingleWord(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")
	c.Cmd("goodbye", "Say goodbye")

	match, consumed := c.matchCommand([]string{"hello", "extra"})
	assert.Equal(t, "hello", match)
	assert.Equal(t, 1, consumed)
}

func TestMatchCommandMultiWord(t *testing.T) {
	c := newCLI()
	c.Cmd("db migrate", "Run migrations")
	c.Cmd("db seed", "Seed data")

	match, consumed := c.matchCommand([]string{"db", "migrate", "extra"})
	assert.Equal(t, "db migrate", match)
	assert.Equal(t, 2, consumed)
}

func TestMatchCommandGreedy(t *testing.T) {
	c := newCLI()
	c.Cmd("db", "Database operations")
	c.Cmd("db migrate", "Run migrations")

	match, consumed := c.matchCommand([]string{"db", "migrate"})
	assert.Equal(t, "db migrate", match)
	assert.Equal(t, 2, consumed, "should prefer longest match")
}

func TestMatchCommandNoMatch(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")

	match, consumed := c.matchCommand([]string{"unknown"})
	assert.Equal(t, "", match)
	assert.Equal(t, 0, consumed)
}

func TestGetStringFlag(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")
	c.Flag("hello", "name", "n", "Name", "World")
	c.parseArgs([]string{"hello", "-n", "Test"})

	assert.Equal(t, "Test", c.Get("name"))
}

func TestGetBoolFlag(t *testing.T) {
	c := newCLI()
	c.Cmd("serve", "Start")
	c.BoolFlag("serve", "verbose", "v", "Verbose")

	c.parseArgs([]string{"serve", "-v"})
	assert.Equal(t, true, c.Get("verbose"))
}

func TestGetBoolFlagNotSet(t *testing.T) {
	c := newCLI()
	c.Cmd("serve", "Start")
	c.BoolFlag("serve", "verbose", "v", "Verbose")

	c.parseArgs([]string{"serve"})
	assert.Equal(t, false, c.Get("verbose"))
}

func TestGetUnknownFlag(t *testing.T) {
	c := newCLI()
	c.parseArgs([]string{})
	assert.Nil(t, c.Get("nonexistent"))
}

func TestCommandReturnsMatched(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")
	c.parseArgs([]string{"hello"})

	assert.Equal(t, "hello", c.Command())
}

func TestArgsReturnsRemaining(t *testing.T) {
	c := newCLI()
	c.Cmd("echo", "Echo")
	c.parseArgs([]string{"echo", "a", "b"})

	args, ok := c.Args().([]interface{})
	require.True(t, ok)
	assert.Len(t, args, 2)
}

func TestParseCalledOnce(t *testing.T) {
	c := newCLI()
	c.Cmd("hello", "Say hello")
	c.parseArgs([]string{"hello"})
	assert.Equal(t, "hello", c.matched)

	// Second parse should be a no-op
	c.parseArgs([]string{"goodbye"})
	assert.Equal(t, "hello", c.matched, "second parse should be ignored")
}

func TestSpaceSeparatedSubcommandParsing(t *testing.T) {
	c := newCLI()
	c.Cmd("db migrate", "Run migrations")
	c.Flag("db migrate", "dry-run", "d", "Dry run", "false")
	c.parseArgs([]string{"db", "migrate", "--dry-run", "true"})

	assert.Equal(t, "db migrate", c.matched)
	assert.Equal(t, "true", c.values["dry-run"])
}

func TestColonSubcommandParsing(t *testing.T) {
	c := newCLI()
	c.Cmd("db:migrate", "Run migrations")
	c.parseArgs([]string{"db:migrate"})

	assert.Equal(t, "db:migrate", c.matched)
}
