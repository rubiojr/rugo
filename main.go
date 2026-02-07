package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/rubiojr/rugo/compiler"
	_ "github.com/rubiojr/rugo/modules/cli"
	_ "github.com/rubiojr/rugo/modules/color"
	_ "github.com/rubiojr/rugo/modules/conv"
	_ "github.com/rubiojr/rugo/modules/http"
	_ "github.com/rubiojr/rugo/modules/json"
	_ "github.com/rubiojr/rugo/modules/os"
	_ "github.com/rubiojr/rugo/modules/str"
	_ "github.com/rubiojr/rugo/modules/test"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

var version = "v0.1.5"

func main() {
	cmd := &cli.Command{
		Name:                   "rugo",
		Usage:                  "A Ruby-inspired language that compiles to Go",
		Version:                version,
		UseShortOptionHandling: true,
		// Allow `rugo script.rg` as shorthand for `rugo run script.rg`
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				arg := cmd.Args().First()
				if strings.HasSuffix(arg, ".rg") || isRugoScript(arg) {
					comp := &compiler.Compiler{}
					return comp.Run(arg, cmd.Args().Tail()...)
				}
			}
			return cli.DefaultShowRootCommandHelp(cmd)
		},
		Commands: []*cli.Command{
			{
				Name:            "run",
				Usage:           "Compile and run a .rg file",
				ArgsUsage:       "<file.rg> [args...]",
				SkipFlagParsing: true,
				Action:          runAction,
			},
			{
				Name:      "build",
				Usage:     "Compile a .rg file to a native binary",
				ArgsUsage: "<file.rg>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output binary name",
					},
				},
				Action: buildAction,
			},
			{
				Name:      "emit",
				Usage:     "Output the generated Go source code",
				ArgsUsage: "<file.rg>",
				Action:    emitAction,
			},
			{
				Name:      "test",
				Usage:     "Run .rt test files",
				ArgsUsage: "[file.rt | directory]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "filter",
						Aliases: []string{"f"},
						Usage:   "Run only tests matching this substring",
					},
					&cli.IntFlag{
						Name:    "jobs",
						Aliases: []string{"j"},
						Usage:   "Parallel test files",
						Value:   1,
					},
					&cli.BoolFlag{
						Name:    "no-color",
						Aliases: []string{"C"},
						Usage:   "Disable ANSI color output",
					},
				},
				Action: testAction,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return fmt.Errorf("usage: rugo run <file.rg> [args...]")
	}
	comp := &compiler.Compiler{}
	return comp.Run(cmd.Args().First(), cmd.Args().Tail()...)
}

func buildAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return fmt.Errorf("usage: rugo build [-o output] <file.rg>")
	}
	comp := &compiler.Compiler{}
	output := cmd.String("output")
	// Also check if -o was passed after the filename (urfave quirk)
	if output == "" {
		for i, arg := range os.Args {
			if (arg == "-o" || arg == "--output") && i+1 < len(os.Args) {
				output = os.Args[i+1]
			}
		}
	}
	return comp.Build(cmd.Args().First(), output)
}

func emitAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return fmt.Errorf("usage: rugo emit <file.rg>")
	}
	comp := &compiler.Compiler{}
	src, err := comp.Emit(cmd.Args().First())
	if err != nil {
		return err
	}
	fmt.Print(src)
	return nil
}

// isRugoScript checks if a file exists and looks like a rugo script.
func isRugoScript(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 64)
	n, _ := f.Read(buf)
	line := string(buf[:n])
	return strings.HasPrefix(line, "#!")
}

func testAction(ctx context.Context, cmd *cli.Command) error {
	target := "."
	if cmd.NArg() > 0 {
		target = cmd.Args().First()
	}

	// Set NO_COLOR if --no-color flag, non-interactive, or NO_COLOR already set.
	// RUGO_FORCE_COLOR is set by the parent process when it knows the terminal supports color
	// (child subprocesses have piped stderr so can't detect TTY themselves).
	if cmd.Bool("no-color") || os.Getenv("NO_COLOR") != "" {
		os.Setenv("NO_COLOR", "1")
	} else if !term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("RUGO_FORCE_COLOR") == "" {
		os.Setenv("NO_COLOR", "1")
	} else {
		os.Setenv("RUGO_FORCE_COLOR", "1")
	}

	// Collect .rt files
	var files []string
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("cannot access %s: %w", target, err)
	}
	if info.IsDir() {
		entries, err := os.ReadDir(target)
		if err != nil {
			return fmt.Errorf("reading directory %s: %w", target, err)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".rt") {
				files = append(files, filepath.Join(target, e.Name()))
			}
		}
	} else {
		files = []string{target}
	}

	if len(files) == 0 {
		return fmt.Errorf("no .rt test files found in %s", target)
	}

	// Single file: run directly (no subprocess overhead)
	if len(files) == 1 {
		fmt.Fprintf(os.Stderr, "=== %s ===\n", files[0])
		comp := &compiler.Compiler{}
		if err := comp.Run(files[0]); err != nil {
			os.Exit(1)
		}
		return nil
	}

	// Multiple files: check concurrency setting
	jobs := cmd.Int("jobs")
	if jobs < 1 {
		jobs = 1
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find rugo binary: %w", err)
	}

	type fileResult struct {
		output bytes.Buffer
		failed bool
		done   chan struct{}
	}

	results := make([]fileResult, len(files))
	for i := range results {
		results[i].done = make(chan struct{})
	}

	// Semaphore for bounded concurrency
	sem := make(chan struct{}, jobs)

	for i, f := range files {
		go func(i int, f string) {
			sem <- struct{}{}
			defer func() { <-sem }()
			defer close(results[i].done)
			cmd := exec.Command(self, "test", f)
			cmd.Stdout = &results[i].output
			cmd.Stderr = &results[i].output
			if err := cmd.Run(); err != nil {
				results[i].failed = true
			}
		}(i, f)
	}

	// Stream results in file order as they complete and accumulate totals
	anyFailed := false
	grandTests, grandPassed, grandFailed, grandSkipped := 0, 0, 0, 0
	summaryRe := regexp.MustCompile(`(\d+) tests, (\d+) passed, (\d+) failed, (\d+) skipped`)
	for i := range results {
		<-results[i].done
		out := results[i].output.Bytes()
		os.Stdout.Write(out)
		if results[i].failed {
			anyFailed = true
		}
		if m := summaryRe.FindSubmatch(out); m != nil {
			t, _ := strconv.Atoi(string(m[1]))
			p, _ := strconv.Atoi(string(m[2]))
			f, _ := strconv.Atoi(string(m[3]))
			s, _ := strconv.Atoi(string(m[4]))
			grandTests += t
			grandPassed += p
			grandFailed += f
			grandSkipped += s
		}
	}

	// Print grand total summary
	noColor := os.Getenv("NO_COLOR") != ""
	colorOK, colorFail, colorReset := "\033[32m", "\033[31m", "\033[0m"
	if noColor {
		colorOK, colorFail, colorReset = "", "", ""
	}
	if grandFailed > 0 {
		fmt.Fprintf(os.Stderr, "\n%d files, %d tests, %d passed, %s%d failed%s, %d skipped\n",
			len(files), grandTests, grandPassed, colorFail, grandFailed, colorReset, grandSkipped)
	} else {
		fmt.Fprintf(os.Stderr, "\n%d files, %d tests, %s%d passed%s, %d failed, %d skipped\n",
			len(files), grandTests, colorOK, grandPassed, colorReset, grandFailed, grandSkipped)
	}

	if anyFailed {
		os.Exit(1)
	}
	return nil
}
