package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/rubiojr/rugo/compiler"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

// Execute runs the Rugo CLI with the given version string.
// Import modules via blank imports before calling this function
// so they register via init().
func Execute(version string) {
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
				Name:      "rats",
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
	targets := cmd.Args().Slice()
	if len(targets) == 0 {
		targets = []string{"."}
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
	for _, target := range targets {
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
			files = append(files, target)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no .rt test files found")
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

	ansi := `(?:\x1b\[[0-9;]*m)*`
	summaryRe := regexp.MustCompile(ansi + `(\d+) tests, ` + ansi + `(\d+) passed` + ansi + `, ` + ansi + `(\d+) failed` + ansi + `, (\d+) skipped`)

	type fileResult struct {
		output []byte
		failed bool
	}

	results := make([]fileResult, len(files))

	if jobs == 1 {
		// Sequential: run each file with live output
		for i, f := range files {
			c := exec.Command(self, "rats", f)
			var buf bytes.Buffer
			c.Stdout = io.MultiWriter(os.Stdout, &buf)
			c.Stderr = io.MultiWriter(os.Stderr, &buf)
			if err := c.Run(); err != nil {
				results[i].failed = true
			}
			results[i].output = buf.Bytes()
		}
	} else {
		// Parallel: buffer output per file, print in order
		type asyncResult struct {
			buf  bytes.Buffer
			done chan struct{}
		}
		async := make([]asyncResult, len(files))
		for i := range async {
			async[i].done = make(chan struct{})
		}
		work := make(chan int, len(files))
		for i := range files {
			work <- i
		}
		close(work)
		var wg sync.WaitGroup
		for range jobs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range work {
					c := exec.Command(self, "rats", files[i])
					c.Stdout = &async[i].buf
					c.Stderr = &async[i].buf
					if err := c.Run(); err != nil {
						results[i].failed = true
					}
					close(async[i].done)
				}
			}()
		}
		for i := range async {
			<-async[i].done
			out := async[i].buf.Bytes()
			os.Stdout.Write(out)
			results[i].output = out
		}
	}

	// Accumulate totals and print grand summary
	anyFailed := false
	grandTests, grandPassed, grandFailed, grandSkipped := 0, 0, 0, 0
	for _, r := range results {
		if r.failed {
			anyFailed = true
		}
		if m := summaryRe.FindSubmatch(r.output); m != nil {
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
