package climod

import (
	"fmt"
	"os"
	"strings"
)

// --- cli module ---

type cliCommand struct {
	Name string
	Desc string
}

type cliFlag struct {
	Cmd     string
	Long    string
	Short   string
	Desc    string
	Default string
	IsBool  bool
}

type CLI struct {
	appName    string
	appVersion string
	appAbout   string
	commands   []cliCommand
	flags      []cliFlag
	parsed     bool
	matched    string
	values     map[string]string
	remaining  []interface{}
}

func (c *CLI) Name(name string) interface{} {
	c.appName = name
	return nil
}

func (c *CLI) Version(ver string) interface{} {
	c.appVersion = ver
	return nil
}

func (c *CLI) About(desc string) interface{} {
	c.appAbout = desc
	return nil
}

func (c *CLI) Cmd(name, desc string) interface{} {
	c.commands = append(c.commands, cliCommand{Name: name, Desc: desc})
	return nil
}

func (c *CLI) Flag(cmd, long, short, desc, defVal string) interface{} {
	c.flags = append(c.flags, cliFlag{
		Cmd: cmd, Long: long, Short: short, Desc: desc, Default: defVal,
	})
	return nil
}

func (c *CLI) BoolFlag(cmd, long, short, desc string) interface{} {
	c.flags = append(c.flags, cliFlag{
		Cmd: cmd, Long: long, Short: short, Desc: desc, IsBool: true,
	})
	return nil
}

func (c *CLI) Parse() interface{} {
	c.parseArgs(os.Args[1:])
	return nil
}

func (c *CLI) Run() interface{} {
	c.parseArgs(os.Args[1:])
	if c.matched == "" {
		c.printHelp()
		os.Exit(0)
	}
	handlerName := strings.NewReplacer(":", "_", "-", "_", " ", "_").Replace(c.matched)
	if fn, ok := rugo_cli_dispatch[handlerName]; ok {
		return fn(c.remaining)
	}
	fmt.Fprintf(os.Stderr, "error: no handler function defined for command %q (expected: def %s(args))\n", c.matched, handlerName)
	os.Exit(1)
	return nil
}

func (c *CLI) Command() interface{} {
	return c.matched
}

func (c *CLI) Get(name string) interface{} {
	if c.values == nil {
		return nil
	}
	// Check bool flags
	for _, f := range c.flags {
		if f.Long == name && f.IsBool {
			v, ok := c.values[name]
			if !ok {
				return false
			}
			return v == "true"
		}
	}
	if v, ok := c.values[name]; ok {
		return v
	}
	return nil
}

func (c *CLI) Args() interface{} {
	return c.remaining
}

func (c *CLI) Help() interface{} {
	c.printHelp()
	os.Exit(0)
	return nil
}

// parseArgs parses command-line arguments into matched command, flag values, and remaining args.
func (c *CLI) parseArgs(args []string) {
	if c.parsed {
		return
	}
	c.parsed = true
	c.values = make(map[string]string)
	c.remaining = nil

	// Apply defaults
	for _, f := range c.flags {
		if !f.IsBool && f.Default != "" {
			c.values[f.Long] = f.Default
		}
	}

	i := 0

	// Find command (first non-flag arg)
	if i < len(args) {
		arg := args[i]
		if arg == "--help" || arg == "-h" {
			if c.matched != "" {
				c.printCommandHelp(c.matched)
			} else {
				c.printHelp()
			}
			os.Exit(0)
		}
		if arg == "--version" || arg == "-V" {
			fmt.Printf("%s %s\n", c.appName, c.appVersion)
			os.Exit(0)
		}
		if arg == "--" {
			i++
		} else if strings.HasPrefix(arg, "-") {
			// Global flag or premature flag — skip for now, re-parse below
		} else if matched, consumed := c.matchCommand(args[i:]); matched != "" {
			// Match against known commands — try longest (multi-word) match first
			c.matched = matched
			i += consumed
		}
		// Unknown positional — treat as remaining
	}

	// Parse flags and remaining args
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			i++
			for i < len(args) {
				c.remaining = append(c.remaining, args[i])
				i++
			}
			break
		}
		if arg == "--help" || arg == "-h" {
			if c.matched != "" {
				c.printCommandHelp(c.matched)
			} else {
				c.printHelp()
			}
			os.Exit(0)
		}

		if strings.HasPrefix(arg, "--") {
			name := arg[2:]
			if f, ok := c.findFlag(c.matched, name, false); ok {
				if f.IsBool {
					c.values[f.Long] = "true"
				} else if i+1 < len(args) {
					i++
					c.values[f.Long] = args[i]
				} else {
					fmt.Fprintf(os.Stderr, "error: flag --%s requires a value\n", name)
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "error: unknown flag --%s\n", name)
				if c.matched != "" {
					c.printCommandHelp(c.matched)
				} else {
					c.printHelp()
				}
				os.Exit(1)
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			short := arg[1:]
			if f, ok := c.findFlag(c.matched, short, true); ok {
				if f.IsBool {
					c.values[f.Long] = "true"
				} else if i+1 < len(args) {
					i++
					c.values[f.Long] = args[i]
				} else {
					fmt.Fprintf(os.Stderr, "error: flag -%s requires a value\n", short)
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "error: unknown flag -%s\n", short)
				if c.matched != "" {
					c.printCommandHelp(c.matched)
				} else {
					c.printHelp()
				}
				os.Exit(1)
			}
		} else {
			c.remaining = append(c.remaining, arg)
		}
		i++
	}
}

// matchCommand tries to match the longest multi-word command from the given args.
// Returns the matched command name and the number of args consumed.
func (c *CLI) matchCommand(args []string) (string, int) {
	bestMatch := ""
	bestLen := 0
	for _, cmd := range c.commands {
		parts := strings.Split(cmd.Name, " ")
		if len(parts) > len(args) {
			continue
		}
		match := true
		for j, p := range parts {
			if args[j] != p {
				match = false
				break
			}
		}
		if match && len(parts) > bestLen {
			bestMatch = cmd.Name
			bestLen = len(parts)
		}
	}
	return bestMatch, bestLen
}

func (c *CLI) findFlag(cmd, name string, isShort bool) (cliFlag, bool) {
	for _, f := range c.flags {
		if f.Cmd != cmd && f.Cmd != "" {
			continue
		}
		if isShort && f.Short == name {
			return f, true
		}
		if !isShort && f.Long == name {
			return f, true
		}
	}
	return cliFlag{}, false
}

func (c *CLI) printHelp() {
	if c.appName != "" {
		fmt.Fprintf(os.Stderr, "%s", c.appName)
		if c.appVersion != "" {
			fmt.Fprintf(os.Stderr, " %s", c.appVersion)
		}
		if c.appAbout != "" {
			fmt.Fprintf(os.Stderr, " — %s", c.appAbout)
		}
		fmt.Fprintln(os.Stderr)
	}

	fmt.Fprintln(os.Stderr, "\nUsage:")
	if c.appName != "" {
		fmt.Fprintf(os.Stderr, "  %s <command> [flags] [args...]\n", c.appName)
	} else {
		fmt.Fprintln(os.Stderr, "  <command> [flags] [args...]")
	}

	if len(c.commands) > 0 {
		fmt.Fprintln(os.Stderr, "\nCommands:")
		// Group by prefix for subcommands
		maxLen := 0
		for _, cmd := range c.commands {
			if len(cmd.Name) > maxLen {
				maxLen = len(cmd.Name)
			}
		}
		for _, cmd := range c.commands {
			fmt.Fprintf(os.Stderr, "  %-*s  %s\n", maxLen, cmd.Name, cmd.Desc)
		}
	}

	// Global flags
	globalFlags := c.flagsForCmd("")
	if len(globalFlags) > 0 {
		fmt.Fprintln(os.Stderr, "\nFlags:")
		c.printFlags(globalFlags)
	}

	fmt.Fprintln(os.Stderr, "\nGlobal flags:")
	fmt.Fprintln(os.Stderr, "  -h, --help      Show help")
	if c.appVersion != "" {
		fmt.Fprintln(os.Stderr, "  -V, --version   Show version")
	}
}

func (c *CLI) printCommandHelp(cmdName string) {
	// Find command description
	var desc string
	for _, cmd := range c.commands {
		if cmd.Name == cmdName {
			desc = cmd.Desc
			break
		}
	}
	if c.appName != "" {
		fmt.Fprintf(os.Stderr, "%s %s", c.appName, cmdName)
	} else {
		fmt.Fprintf(os.Stderr, "%s", cmdName)
	}
	if desc != "" {
		fmt.Fprintf(os.Stderr, " — %s", desc)
	}
	fmt.Fprintln(os.Stderr)

	cmdFlags := c.flagsForCmd(cmdName)
	globalFlags := c.flagsForCmd("")
	allFlags := append(cmdFlags, globalFlags...)
	if len(allFlags) > 0 {
		fmt.Fprintln(os.Stderr, "\nFlags:")
		c.printFlags(allFlags)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  -h, --help   Show help")
}

func (c *CLI) flagsForCmd(cmd string) []cliFlag {
	var result []cliFlag
	for _, f := range c.flags {
		if f.Cmd == cmd {
			result = append(result, f)
		}
	}
	return result
}

func (c *CLI) printFlags(flags []cliFlag) {
	for _, f := range flags {
		short := ""
		if f.Short != "" {
			short = "-" + f.Short + ", "
		} else {
			short = "    "
		}
		long := "--" + f.Long
		if f.IsBool {
			fmt.Fprintf(os.Stderr, "  %s%-14s %s\n", short, long, f.Desc)
		} else {
			flag := fmt.Sprintf("%s%s <value>", short, long)
			def := ""
			if f.Default != "" {
				def = fmt.Sprintf(" (default: %q)", f.Default)
			}
			fmt.Fprintf(os.Stderr, "  %-20s %s%s\n", flag, f.Desc, def)
		}
	}
}
