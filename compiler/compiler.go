package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rubiojr/rugo/compiler/gobridge"
	"github.com/rubiojr/rugo/modules"
	"github.com/rubiojr/rugo/parser"
)

// Compiler orchestrates the full compilation pipeline.
type Compiler struct {
	// BaseDir is the directory of the main source file (for resolving requires).
	BaseDir string
	// loaded tracks already-loaded files to prevent duplicate requires.
	loaded map[string]bool
	// imports tracks which Rugo stdlib modules have been imported via use.
	imports map[string]bool
	// goImports tracks Go stdlib bridge packages imported via import.
	goImports map[string]string // package path → alias (empty = default)
	// nsFuncs tracks namespace+function pairs to detect duplicates.
	nsFuncs map[string]string // "ns.func" → source file
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
	if c.imports == nil {
		c.imports = make(map[string]bool)
	}
	if c.goImports == nil {
		c.goImports = make(map[string]string)
	}
	if c.nsFuncs == nil {
		c.nsFuncs = make(map[string]string)
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
	goSrc, err := generate(resolved, filename)
	if err != nil {
		return nil, fmt.Errorf("code generation: %w", err)
	}

	return &CompileResult{GoSource: goSrc, Program: resolved, SourceFile: filename}, nil
}

// goModContent generates a go.mod with require lines for any external
// dependencies declared by the imported modules.
func goModContent(prog *Program) string {
	var modNames []string
	for _, stmt := range prog.Statements {
		if imp, ok := stmt.(*UseStmt); ok {
			modNames = append(modNames, imp.Module)
		}
	}

	goMod := "module rugo_program\n\ngo 1.22\n"
	if deps := modules.CollectGoDeps(modNames); len(deps) > 0 {
		goMod += "\nrequire (\n"
		for _, dep := range deps {
			goMod += "\t" + dep + "\n"
		}
		goMod += ")\n"
	}
	return goMod
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

	goMod := goModContent(result.Program)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		return fmt.Errorf("writing go.mod: %w", err)
	}

	// Build to a binary first, then run it
	binFile := filepath.Join(tmpDir, "rugo_program")
	buildCmd := exec.Command("go", "build", "-mod=mod", "-ldflags=-s -w", "-o", binFile, ".")
	buildCmd.Dir = tmpDir
	buildCmd.Env = appendGoNoSumCheck(os.Environ())
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
	goMod := goModContent(result.Program)
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

	cmd := exec.Command("go", "build", "-mod=mod", "-ldflags=-s -w", "-o", absOutput, ".")
	cmd.Dir = tmpDir
	cmd.Env = appendGoNoSumCheck(os.Environ())
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

	// Expand heredocs before comment stripping (bodies may contain #).
	cleaned, err := expandHeredocs(string(src))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}

	// Strip comments
	cleaned, err = stripComments(cleaned)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}

	// Scan for user-defined function names (quick pass for def lines)
	userFuncs := scanFuncDefs(cleaned)

	// preprocess: paren-free calls + shell fallback
	var lineMap []int
	cleaned, lineMap, err = preprocess(cleaned, userFuncs)
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

	prog, err := walkWithLineMap(p, ast, lineMap)
	if err != nil {
		return nil, fmt.Errorf("walking AST for %s: %w", filename, err)
	}

	return prog, nil
}

func (c *Compiler) resolveRequires(prog *Program) (*Program, error) {
	// Validate that import/require only appear at top level
	if err := validateTopLevelOnly(prog.Statements); err != nil {
		return nil, err
	}

	var resolved []Statement

	for _, s := range prog.Statements {
		// Validate and deduplicate use statements (Rugo stdlib modules)
		if use, ok := s.(*UseStmt); ok {
			if !modules.IsModule(use.Module) {
				return nil, fmt.Errorf("unknown stdlib module: %q (available: %s)", use.Module, strings.Join(modules.Names(), ", "))
			}
			if !c.imports[use.Module] {
				c.imports[use.Module] = true
				resolved = append(resolved, s)
			}
			continue
		}

		// Validate and deduplicate import statements (Go stdlib bridge)
		if imp, ok := s.(*ImportStmt); ok {
			ns := goBridgeNamespace(imp)
			if !gobridge.IsPackage(imp.Package) {
				return nil, fmt.Errorf("unsupported Go bridge package: %q (available: %s)", imp.Package, strings.Join(gobridge.PackageNames(), ", "))
			}
			// Check for namespace conflicts with Rugo modules
			if c.imports[ns] {
				return nil, fmt.Errorf("import namespace %q conflicts with a use'd Rugo module; add an alias: import %q as <alias>", ns, imp.Package)
			}
			if _, exists := c.goImports[imp.Package]; !exists {
				c.goImports[imp.Package] = imp.Alias
				resolved = append(resolved, s)
			}
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

		// Reject require namespace that conflicts with a use'd Rugo module
		if c.imports[ns] {
			return nil, fmt.Errorf("require namespace %q conflicts with use'd stdlib module", ns)
		}
		// Reject require namespace that conflicts with an import'd Go bridge
		for pkg, alias := range c.goImports {
			bridgeNS := alias
			if bridgeNS == "" {
				bridgeNS = gobridge.DefaultNS(pkg)
			}
			if ns == bridgeNS {
				return nil, fmt.Errorf("require namespace %q conflicts with imported Go bridge package %q", ns, pkg)
			}
		}

		// Include use/import statements and function definitions from required files
		for _, rs := range reqProg.Statements {
			switch st := rs.(type) {
			case *UseStmt:
				c.imports[st.Module] = true
				resolved = append(resolved, st)
			case *ImportStmt:
				if _, exists := c.goImports[st.Package]; !exists {
					c.goImports[st.Package] = st.Alias
				}
				resolved = append(resolved, st)
			case *FuncDef:
				// Detect duplicate function in same namespace
				nsKey := ns + "." + st.Name
				if src, exists := c.nsFuncs[nsKey]; exists {
					return nil, fmt.Errorf("function %q in namespace %q already defined (from %s)", st.Name, ns, src)
				}
				c.nsFuncs[nsKey] = req.Path
				st.Namespace = ns
				resolved = append(resolved, st)
			}
		}
	}

	return &Program{Statements: resolved}, nil
}

// validateTopLevelOnly walks statement trees and returns an error if
// import or require statements appear inside function bodies or blocks.
func validateTopLevelOnly(stmts []Statement) error {
	for _, s := range stmts {
		switch st := s.(type) {
		case *FuncDef:
			if err := rejectNestedImports(st.Body); err != nil {
				return err
			}
		case *IfStmt:
			if err := rejectNestedImports(st.Body); err != nil {
				return err
			}
			for _, clause := range st.ElsifClauses {
				if err := rejectNestedImports(clause.Body); err != nil {
					return err
				}
			}
			if err := rejectNestedImports(st.ElseBody); err != nil {
				return err
			}
		case *WhileStmt:
			if err := rejectNestedImports(st.Body); err != nil {
				return err
			}
		case *ForStmt:
			if err := rejectNestedImports(st.Body); err != nil {
				return err
			}
		case *TestDef:
			if err := rejectNestedImports(st.Body); err != nil {
				return err
			}
		}
	}
	return nil
}

// rejectNestedImports checks a block body for use/import/require statements
// and returns an error if any are found.
func rejectNestedImports(stmts []Statement) error {
	for _, s := range stmts {
		switch s.(type) {
		case *UseStmt:
			return fmt.Errorf("line %d: use statements must be at the top level", s.StmtLine())
		case *ImportStmt:
			return fmt.Errorf("line %d: import statements must be at the top level", s.StmtLine())
		case *RequireStmt:
			return fmt.Errorf("line %d: require statements must be at the top level", s.StmtLine())
		}
		// Recurse into nested blocks
		switch st := s.(type) {
		case *FuncDef:
			if err := rejectNestedImports(st.Body); err != nil {
				return err
			}
		case *IfStmt:
			if err := rejectNestedImports(st.Body); err != nil {
				return err
			}
			for _, clause := range st.ElsifClauses {
				if err := rejectNestedImports(clause.Body); err != nil {
					return err
				}
			}
			if err := rejectNestedImports(st.ElseBody); err != nil {
				return err
			}
		case *WhileStmt:
			if err := rejectNestedImports(st.Body); err != nil {
				return err
			}
		case *ForStmt:
			if err := rejectNestedImports(st.Body); err != nil {
				return err
			}
		}
	}
	return nil
}

// appendGoNoSumCheck adds GONOSUMCHECK=* to the environment if not already set,
// allowing temporary build directories to resolve module dependencies without
// requiring a pre-populated go.sum.
func appendGoNoSumCheck(env []string) []string {
	for _, e := range env {
		if strings.HasPrefix(e, "GONOSUMCHECK=") {
			return env
		}
	}
	return append(env, "GONOSUMCHECK=*")
}

// goBridgeNamespace returns the Rugo namespace for a Go bridge import.
func goBridgeNamespace(imp *ImportStmt) string {
	if imp.Alias != "" {
		return imp.Alias
	}
	return gobridge.DefaultNS(imp.Package)
}
