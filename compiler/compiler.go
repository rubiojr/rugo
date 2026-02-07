package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rubiojr/rugo/modules"
	"github.com/rubiojr/rugo/parser"
)

// Compiler orchestrates the full compilation pipeline.
type Compiler struct {
	// BaseDir is the directory of the main source file (for resolving requires).
	BaseDir string
	// loaded tracks already-loaded files to prevent duplicate requires.
	loaded map[string]bool
}

// CompileResult holds the output of a compilation.
type CompileResult struct {
	GoSource   string
	Program    *Program
	SourceFile string // original .rg filename
}

// Compile reads a .rg file, resolves requires, and produces Go source.
func (c *Compiler) Compile(filename string) (*CompileResult, error) {
	if c.loaded == nil {
		c.loaded = make(map[string]bool)
	}
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("resolving path %s: %w", filename, err)
	}
	c.BaseDir = filepath.Dir(absPath)

	prog, err := c.parseFile(absPath)
	if err != nil {
		return nil, err
	}

	// Resolve requires recursively
	resolved, err := c.resolveRequires(prog)
	if err != nil {
		return nil, err
	}

	// Generate Go source
	goSrc, err := Generate(resolved, filename)
	if err != nil {
		return nil, fmt.Errorf("code generation: %w", err)
	}

	return &CompileResult{GoSource: goSrc, Program: resolved, SourceFile: filename}, nil
}

// Run compiles and runs a .rg file, passing extraArgs to the compiled binary.
func (c *Compiler) Run(filename string, extraArgs ...string) error {
	result, err := c.Compile(filename)
	if err != nil {
		return err
	}

	// Write to temp directory with go module
	tmpDir, err := os.MkdirTemp("", "rugo-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(result.GoSource), 0644); err != nil {
		return fmt.Errorf("writing Go source: %w", err)
	}

	goMod := "module rugo_program\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		return fmt.Errorf("writing go.mod: %w", err)
	}

	// Build to a binary first, then run it
	binFile := filepath.Join(tmpDir, "rugo_program")
	buildCmd := exec.Command("go", "build", "-o", binFile, ".")
	buildCmd.Dir = tmpDir
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	cmd := exec.Command(binFile, extraArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.RemoveAll(tmpDir)
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

// Build compiles a .rg file to a native binary.
func (c *Compiler) Build(filename, output string) error {
	result, err := c.Compile(filename)
	if err != nil {
		return err
	}

	// Write to temp directory with a go module
	tmpDir, err := os.MkdirTemp("", "rugo-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(result.GoSource), 0644); err != nil {
		return fmt.Errorf("writing Go source: %w", err)
	}

	// Create a go.mod so go build works properly
	goMod := "module rugo_program\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		return fmt.Errorf("writing go.mod: %w", err)
	}

	if output == "" {
		base := filepath.Base(filename)
		output = strings.TrimSuffix(base, filepath.Ext(base))
	}

	absOutput, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}

	cmd := exec.Command("go", "build", "-o", absOutput, ".")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}
	return nil
}

// Emit compiles a .rg file and outputs the Go source.
func (c *Compiler) Emit(filename string) (string, error) {
	result, err := c.Compile(filename)
	if err != nil {
		return "", err
	}
	return result.GoSource, nil
}

func (c *Compiler) parseFile(filename string) (*Program, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}

	// Strip comments
	cleaned := StripComments(string(src))

	// Scan for user-defined function names (quick pass for def lines)
	userFuncs := ScanFuncDefs(cleaned)

	// Preprocess: paren-free calls + shell fallback
	var lineMap []int
	cleaned, lineMap, err = Preprocess(cleaned, userFuncs)
	if err != nil {
		return nil, fmt.Errorf("preprocessing %s: %w", filename, err)
	}

	// Ensure source ends with newline
	if !strings.HasSuffix(cleaned, "\n") {
		cleaned += "\n"
	}

	p := &parser.Parser{}
	ast, err := p.Parse(filename, []byte(cleaned))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filename, err)
	}

	prog, err := WalkWithLineMap(p, ast, lineMap)
	if err != nil {
		return nil, fmt.Errorf("walking AST for %s: %w", filename, err)
	}

	return prog, nil
}

func (c *Compiler) resolveRequires(prog *Program) (*Program, error) {
	var resolved []Statement

	for _, s := range prog.Statements {
		// Validate import statements
		if imp, ok := s.(*ImportStmt); ok {
			if !modules.IsModule(imp.Module) {
				return nil, fmt.Errorf("unknown stdlib module: %q (available: %s)", imp.Module, strings.Join(modules.Names(), ", "))
			}
			resolved = append(resolved, s)
			continue
		}

		req, ok := s.(*RequireStmt)
		if !ok {
			resolved = append(resolved, s)
			continue
		}

		// Resolve the require path
		reqPath := req.Path
		if !strings.HasSuffix(reqPath, ".rg") {
			reqPath += ".rg"
		}
		if !filepath.IsAbs(reqPath) {
			reqPath = filepath.Join(c.BaseDir, reqPath)
		}

		absPath, err := filepath.Abs(reqPath)
		if err != nil {
			return nil, fmt.Errorf("resolving require path %s: %w", req.Path, err)
		}

		if c.loaded[absPath] {
			continue // Already loaded
		}
		c.loaded[absPath] = true

		reqProg, err := c.parseFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("in require %q: %w", req.Path, err)
		}

		// Recursively resolve requires in the required file
		oldBase := c.BaseDir
		c.BaseDir = filepath.Dir(absPath)
		reqProg, err = c.resolveRequires(reqProg)
		c.BaseDir = oldBase
		if err != nil {
			return nil, err
		}

		// Determine namespace: alias or filename
		ns := req.Alias
		if ns == "" {
			base := filepath.Base(req.Path)
			ns = strings.TrimSuffix(base, filepath.Ext(base))
			if !strings.HasSuffix(req.Path, ".rg") {
				ns = req.Path
			}
		}

		// Include function definitions from required files, namespaced
		for _, rs := range reqProg.Statements {
			if fd, ok := rs.(*FuncDef); ok {
				fd.Namespace = ns
				resolved = append(resolved, fd)
			}
		}
	}

	return &Program{Statements: resolved}, nil
}
