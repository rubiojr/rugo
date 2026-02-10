package compiler

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rubiojr/rugo/compiler/gobridge"
	"github.com/rubiojr/rugo/modules"
	"github.com/rubiojr/rugo/parser"
	"github.com/rubiojr/rugo/remote"
	"modernc.org/scanner"
)

// Compiler orchestrates the full compilation pipeline.
type Compiler struct {
	// BaseDir is the directory of the main source file (for resolving requires).
	BaseDir string
	// TestMode enables test harness generation (rats blocks are included).
	// When false (default), rats blocks are silently skipped during codegen.
	TestMode bool
	// ModuleDir overrides the default module cache directory (~/.rugo/modules).
	// Used for testing. When empty, the default is used.
	ModuleDir string
	// Frozen errors if the lock file is stale or a new dependency is resolved.
	Frozen bool
	// resolver handles remote module fetching, caching, and lock file state.
	resolver *remote.Resolver
	// loaded tracks already-loaded files and the namespace they were loaded under.
	loaded map[string]string // abs path → namespace
	// imports tracks which Rugo stdlib modules have been imported via use.
	imports map[string]bool
	// goImports tracks Go stdlib bridge packages imported via import.
	goImports map[string]string // package path → alias (empty = default)
	// nsFuncs tracks namespace+function pairs to detect duplicates.
	nsFuncs map[string]string // "ns.func" → source file
	// sourcePrefix is prepended to the main file source before parsing.
	// Used to auto-inject require statements for test helpers.
	sourcePrefix string
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
		c.loaded = make(map[string]string)
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

	// Initialize remote resolver.
	if c.resolver == nil {
		c.resolver = &remote.Resolver{ModuleDir: c.ModuleDir, Frozen: c.Frozen}
		if err := c.resolver.InitLockFromDir(c.BaseDir); err != nil {
			return nil, err
		}
	}

	// In test mode, auto-require .rg files from helpers/ dir next to the test file.
	if c.TestMode {
		c.sourcePrefix = c.discoverHelpers()
	}

	prog, err := c.parseFile(absPath, displayPath(absPath))
	c.sourcePrefix = "" // Only apply to the main file, not requires
	if err != nil {
		return nil, err
	}

	// Resolve requires recursively
	resolved, err := c.resolveRequires(prog)
	if err != nil {
		return nil, err
	}

	// Write lock file if modified during compilation.
	if err := c.resolver.WriteLockIfDirty(); err != nil {
		return nil, err
	}

	// Generate Go source
	goSrc, err := generate(resolved, filename, c.TestMode)
	if err != nil {
		return nil, err
	}

	return &CompileResult{GoSource: goSrc, Program: resolved, SourceFile: filename}, nil
}

// discoverHelpers finds .rg files in a helpers/ directory next to the test file
// and returns require statements to prepend to the source.
func (c *Compiler) discoverHelpers() string {
	helpersDir := filepath.Join(c.BaseDir, "helpers")
	entries, err := os.ReadDir(helpersDir)
	if err != nil {
		return ""
	}
	var lines []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".rg") {
			continue
		}
		base := strings.TrimSuffix(name, ".rg")
		lines = append(lines, fmt.Sprintf("require \"helpers/%s\" as \"%s\"", base, base))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
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
	var buildStderr bytes.Buffer
	buildCmd.Stderr = &buildStderr
	if err := buildCmd.Run(); err != nil {
		return translateBuildError(buildStderr.String(), result.SourceFile)
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
	var cmdStderr bytes.Buffer
	cmd.Stderr = &cmdStderr
	if err := cmd.Run(); err != nil {
		return translateBuildError(cmdStderr.String(), result.SourceFile)
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

func (c *Compiler) parseFile(filename, displayName string) (*Program, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", displayName, err)
	}

	// Expand heredocs before comment stripping (bodies may contain #).
	cleaned, err := expandHeredocs(c.sourcePrefix + string(src))
	if err != nil {
		return nil, fmt.Errorf("%s:%w", displayName, err)
	}

	// Strip comments
	cleaned, err = stripComments(cleaned)
	if err != nil {
		return nil, fmt.Errorf("%s:%w", displayName, err)
	}

	// Expand struct definitions and method definitions before other preprocessing
	var structLineMap []int
	cleaned, structLineMap = expandStructDefs(cleaned)

	// Scan for user-defined function names (quick pass for def lines)
	userFuncs := scanFuncDefs(cleaned)

	// preprocess: paren-free calls + shell fallback
	var lineMap []int
	cleaned, lineMap, err = preprocess(cleaned, userFuncs)
	if err != nil {
		return nil, fmt.Errorf("%s:%w", displayName, err)
	}

	// Compose struct line map with preprocess line map for accurate source locations
	if structLineMap != nil && lineMap != nil {
		for i, ppLine := range lineMap {
			if ppLine > 0 && ppLine <= len(structLineMap) {
				lineMap[i] = structLineMap[ppLine-1]
			}
		}
	} else if structLineMap != nil {
		lineMap = structLineMap
	}

	// Ensure source ends with newline
	if !strings.HasSuffix(cleaned, "\n") {
		cleaned += "\n"
	}

	p := &parser.Parser{}
	ast, err := p.Parse(displayName, []byte(cleaned))
	if err != nil {
		return nil, firstParseError(err)
	}

	prog, err := walkWithLineMap(p, ast, lineMap)
	if err != nil {
		return nil, fmt.Errorf("%s: internal compiler error: %w (please report this bug)", displayName, err)
	}

	prog.SourceFile = displayName
	return prog, nil
}

func (c *Compiler) resolveRequires(prog *Program) (*Program, error) {
	// Validate that import/require only appear at top level
	if err := validateTopLevelOnly(prog.Statements, prog.SourceFile); err != nil {
		return nil, err
	}

	var resolved []Statement

	for _, s := range prog.Statements {
		// Validate and deduplicate use statements (Rugo stdlib modules)
		if use, ok := s.(*UseStmt); ok {
			if use.Module == "" {
				return nil, fmt.Errorf("%s:%d: empty module name in use statement", prog.SourceFile, s.StmtLine())
			}
			if !modules.IsModule(use.Module) {
				if suggestion := closestMatch(use.Module, modules.Names()); suggestion != "" {
					return nil, fmt.Errorf("%s:%d: unknown module %q — did you mean %q?", prog.SourceFile, s.StmtLine(), use.Module, suggestion)
				}
				return nil, fmt.Errorf("%s:%d: unknown module %q (available: %s)", prog.SourceFile, s.StmtLine(), use.Module, strings.Join(modules.Names(), ", "))
			}
			if !c.imports[use.Module] {
				c.imports[use.Module] = true
				resolved = append(resolved, s)
			}
			continue
		}

		// Validate and deduplicate import statements (Go stdlib bridge)
		if imp, ok := s.(*ImportStmt); ok {
			if imp.Package == "" {
				return nil, fmt.Errorf("%s:%d: empty package name in import statement", prog.SourceFile, s.StmtLine())
			}
			ns := goBridgeNamespace(imp)
			if !gobridge.IsPackage(imp.Package) {
				if suggestion := closestMatch(imp.Package, gobridge.PackageNames()); suggestion != "" {
					return nil, fmt.Errorf("%s:%d: unknown package %q — did you mean %q?", prog.SourceFile, s.StmtLine(), imp.Package, suggestion)
				}
				return nil, fmt.Errorf("%s:%d: unknown package %q (available: %s)", prog.SourceFile, s.StmtLine(), imp.Package, strings.Join(gobridge.PackageNames(), ", "))
			}
			// Check for namespace conflicts with Rugo modules
			if c.imports[ns] {
				return nil, fmt.Errorf("%s:%d: import namespace %q conflicts with a use'd Rugo module; add an alias: import %q as <alias>", prog.SourceFile, s.StmtLine(), ns, imp.Package)
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

		// Validate require path
		if req.Path == "" {
			return nil, fmt.Errorf("%s:%d: empty path in require statement", prog.SourceFile, s.StmtLine())
		}

		// Validate alias
		if req.Alias != "" {
			if err := validateNamespace(req.Alias); err != nil {
				return nil, fmt.Errorf("%s:%d: invalid require alias %q: %s", prog.SourceFile, s.StmtLine(), req.Alias, err)
			}
		}

		// Handle "with" clause: load specific sub-modules from a directory
		if len(req.With) > 0 {
			var baseDir string
			if remote.IsRemoteRequire(req.Path) {
				var err error
				baseDir, err = c.resolver.FetchRepo(req.Path)
				if err != nil {
					return nil, fmt.Errorf("%s:%d: %w", prog.SourceFile, s.StmtLine(), err)
				}
			} else {
				// Local path: resolve relative to the calling file's directory
				localDir := req.Path
				if !filepath.IsAbs(localDir) {
					localDir = filepath.Join(c.BaseDir, localDir)
				}
				info, err := os.Stat(localDir)
				if err != nil || !info.IsDir() {
					return nil, fmt.Errorf("%s:%d: require with 'with' requires a directory, but %q is not a directory", prog.SourceFile, req.StmtLine(), req.Path)
				}
				baseDir = localDir
			}

			for _, modName := range req.With {
				modFile := filepath.Join(baseDir, modName+".rg")
				if !fileExists(modFile) {
					// Fallback: look in lib/ subdirectory
					modFile = filepath.Join(baseDir, "lib", modName+".rg")
				}
				if !fileExists(modFile) {
					return nil, fmt.Errorf("%s:%d: module %q not found in %s (no %s.rg)", prog.SourceFile, req.StmtLine(), modName, req.Path, modName)
				}

				absPath, err := filepath.Abs(modFile)
				if err != nil {
					return nil, fmt.Errorf("resolving require path %s/%s: %w", req.Path, modName, err)
				}

				if _, alreadyLoaded := c.loaded[absPath]; alreadyLoaded {
					continue
				}
				c.loaded[absPath] = modName

				reqProg, err := c.parseFile(absPath, displayPath(absPath))
				if err != nil {
					return nil, fmt.Errorf("in require %q (with %s): %w", req.Path, modName, err)
				}

				oldBase := c.BaseDir
				c.BaseDir = filepath.Dir(absPath)
				reqProg, err = c.resolveRequires(reqProg)
				c.BaseDir = oldBase
				if err != nil {
					return nil, err
				}

				ns := modName

				if c.imports[ns] {
					return nil, fmt.Errorf("%s:%d: require namespace %q (from with) conflicts with use'd stdlib module", prog.SourceFile, req.StmtLine(), ns)
				}
				for pkg, alias := range c.goImports {
					bridgeNS := alias
					if bridgeNS == "" {
						bridgeNS = gobridge.DefaultNS(pkg)
					}
					if ns == bridgeNS {
						return nil, fmt.Errorf("%s:%d: require namespace %q (from with) conflicts with imported Go bridge package %q", prog.SourceFile, req.StmtLine(), ns, pkg)
					}
				}

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
						if st.Namespace != "" {
							resolved = append(resolved, st)
							continue
						}
						nsKey := ns + "." + st.Name
						if src, exists := c.nsFuncs[nsKey]; exists {
							return nil, fmt.Errorf("function %q in namespace %q already defined (from %s)", st.Name, ns, src)
						}
						c.nsFuncs[nsKey] = req.Path + "/" + modName
						st.Namespace = ns
						resolved = append(resolved, st)
					case *AssignStmt:
						if st.Namespace != "" {
							resolved = append(resolved, st)
							continue
						}
						st.Namespace = ns
						resolved = append(resolved, st)
					}
				}
			}
			continue
		}

		var absPath string

		if remote.IsRemoteRequire(req.Path) {
			// Remote require: fetch from git and resolve entry point
			localPath, err := c.resolver.ResolveModule(req.Path)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %w", prog.SourceFile, s.StmtLine(), err)
			}
			absPath = localPath
		} else {
			// Local require: resolve relative to calling file
			reqPath := req.Path
			if !filepath.IsAbs(reqPath) {
				reqPath = filepath.Join(c.BaseDir, reqPath)
			}
			// Try as a file first (append .rg if needed), then as a directory
			if !strings.HasSuffix(reqPath, ".rg") {
				if fileExists(reqPath + ".rg") {
					reqPath += ".rg"
				} else if info, err := os.Stat(reqPath); err == nil && info.IsDir() {
					entryPoint, err := FindLocalEntryPoint(reqPath)
					if err != nil {
						return nil, fmt.Errorf("%s:%d: %w", prog.SourceFile, req.StmtLine(), err)
					}
					reqPath = entryPoint
				} else {
					reqPath += ".rg"
				}
			}
			var err error
			absPath, err = filepath.Abs(reqPath)
			if err != nil {
				return nil, fmt.Errorf("resolving require path %s: %w", req.Path, err)
			}
		}

		// Determine namespace early (needed for dedup check)
		ns := req.Alias
		if ns == "" {
			if remote.IsRemoteRequire(req.Path) {
				ns, _ = remote.DefaultNamespace(req.Path)
			} else {
				base := filepath.Base(req.Path)
				ns = strings.TrimSuffix(base, filepath.Ext(base))
			}
		}

		if prevNS, alreadyLoaded := c.loaded[absPath]; alreadyLoaded {
			if ns != prevNS {
				return nil, fmt.Errorf("%s:%d: %q already required as %q — cannot re-require with a different namespace %q", prog.SourceFile, req.StmtLine(), req.Path, prevNS, ns)
			}
			continue // Already loaded with same namespace
		}
		c.loaded[absPath] = ns

		reqProg, err := c.parseFile(absPath, displayPath(absPath))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("cannot find required file %q (looked for %s)", req.Path, displayPath(absPath))
			}
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

		// Reject require namespace that conflicts with a use'd Rugo module
		if c.imports[ns] {
			return nil, fmt.Errorf("%s:%d: require namespace %q conflicts with use'd stdlib module", prog.SourceFile, req.StmtLine(), ns)
		}
		// Reject require namespace that conflicts with an import'd Go bridge
		for pkg, alias := range c.goImports {
			bridgeNS := alias
			if bridgeNS == "" {
				bridgeNS = gobridge.DefaultNS(pkg)
			}
			if ns == bridgeNS {
				return nil, fmt.Errorf("%s:%d: require namespace %q conflicts with imported Go bridge package %q", prog.SourceFile, req.StmtLine(), ns, pkg)
			}
		}

		// Include use/import statements and function definitions from required files.
		// Functions/assignments already namespaced by a deeper require are passed through.
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
				if st.Namespace != "" {
					// Already namespaced from a nested require — pass through
					resolved = append(resolved, st)
					continue
				}
				// Detect duplicate function in same namespace
				nsKey := ns + "." + st.Name
				if src, exists := c.nsFuncs[nsKey]; exists {
					return nil, fmt.Errorf("function %q in namespace %q already defined (from %s)", st.Name, ns, src)
				}
				c.nsFuncs[nsKey] = req.Path
				st.Namespace = ns
				resolved = append(resolved, st)
			case *AssignStmt:
				if st.Namespace != "" {
					resolved = append(resolved, st)
					continue
				}
				// Top-level assignments (constants) from required files
				st.Namespace = ns
				resolved = append(resolved, st)
			}
		}
	}

	return &Program{Statements: resolved}, nil
}

// displayPath returns a relative path for use in error messages.
// Falls back to the original path if relativization fails.
func displayPath(absPath string) string {
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, absPath); err == nil {
			return rel
		}
	}
	return absPath
}

// firstParseError extracts only the first error from a parser error list
// and reformats it for human readability.
func firstParseError(err error) error {
	if el, ok := err.(scanner.ErrList); ok && len(el) > 0 {
		e := el[0]
		msg := formatParseError(e)

		snippetLine := e.Pos.Line
		snippetCol := e.Pos.Column

		// For "rats/bench missing name", show the keyword line
		rawMsg := e.Err.Error()
		if strings.Contains(rawMsg, "expected [str_lit]") {
			if keyword := findBlockMissingName(e.Pos.Filename, e.Pos.Line); keyword != "" {
				if data, err := os.ReadFile(e.Pos.Filename); err == nil {
					srcLines := strings.Split(string(data), "\n")
					for i := e.Pos.Line - 1; i >= 0 && i >= e.Pos.Line-3; i-- {
						trimmed := strings.TrimSpace(srcLines[i])
						if trimmed == keyword || strings.HasPrefix(trimmed, keyword+" ") || strings.HasPrefix(trimmed, keyword+"\t") {
							snippetLine = i + 1
							raw := srcLines[i]
							indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
							snippetCol = indent + len(keyword) + 1
							break
						}
					}
				}
			}
		}

		if snippet := sourceSnippet(e.Pos.Filename, snippetLine, snippetCol); snippet != "" {
			msg += "\n" + snippet
		}
		return fmt.Errorf("%s", msg)
	}
	return err
}

// sourceSnippet returns a Rust-style source code snippet for the given
// file, line, and column, with a caret pointing to the error position.
func sourceSnippet(filename string, line, col int) string {
	if filename == "" || line <= 0 {
		return ""
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if line > len(lines) {
		return ""
	}

	var sb strings.Builder
	lineNumWidth := len(fmt.Sprintf("%d", line+1))
	pad := strings.Repeat(" ", lineNumWidth)

	sb.WriteString(fmt.Sprintf("\n%s |\n", pad))

	// Show the error line
	srcLine := lines[line-1]
	sb.WriteString(fmt.Sprintf("%*d | %s\n", lineNumWidth, line, srcLine))

	// Show caret pointer
	if col > 0 {
		caretPad := strings.Repeat(" ", col-1)
		sb.WriteString(fmt.Sprintf("%s | %s^\n", pad, caretPad))
	}

	return sb.String()
}

// formatParseError rewrites a parser error message into a human-friendly format.
// Input format: `file:line:col: "token" [type]: expected [Sym1 Sym2 ...]`
// Output format: `file:line:col: unexpected <desc> — expected <friendly list>`
func formatParseError(e scanner.ErrWithPosition) string {
	msg := e.Err.Error()

	// Extract the expected set from the message
	// Format: `"token" [type]: expected [...]`
	prefix := fmt.Sprintf("%s: ", e.Pos)

	// Try to parse the structured error message
	if idx := strings.Index(msg, "expected ["); idx >= 0 {
		// Get the part before "expected" to extract token info
		beforeExpected := msg[:idx]
		expectedPart := msg[idx+len("expected ["):]
		expectedPart = strings.TrimSuffix(expectedPart, "]")

		// Parse token description from: "token" [type]:
		tokenDesc := parseTokenDescription(beforeExpected)

		// Special case: stray "end" with no matching block
		if strings.Contains(beforeExpected, `"end"`) && isStatementExpectedSet(expectedPart) {
			return prefix + "unexpected \"end\" — no matching block to close (def, if, while, for, etc.)"
		}

		// Special case: "or" without "try"
		if strings.Contains(beforeExpected, `"or"`) {
			return prefix + "unexpected \"or\" — did you mean \"try <expr> or <default>\"?"
		}

		// Special case: EOF with expected "end" — unclosed block
		if strings.Contains(beforeExpected, "[EOF]") && isEndExpectedSet(expectedPart) {
			blockType := detectUnclosedBlock(e.Pos.Filename)
			if blockType != "" {
				return prefix + "unexpected end of file — unclosed " + blockType
			}
			return prefix + "unexpected end of file — expected \"end\" (unclosed block)"
		}

		// Special case: expected str_lit after "rats" or "bench" — missing test name
		if strings.TrimSpace(expectedPart) == "str_lit" {
			if keyword := findBlockMissingName(e.Pos.Filename, e.Pos.Line); keyword != "" {
				return prefix + "\"" + keyword + "\" requires a name — e.g. " + keyword + " \"description\""
			}
		}

		// Parse and simplify the expected set
		friendly := simplifyExpectedSet(expectedPart)

		return prefix + "unexpected " + tokenDesc + " — expected " + friendly
	}

	return prefix + msg
}

// parseTokenDescription extracts a human-friendly token description
// from the parser error prefix like: `"puts" [ident]: `
func parseTokenDescription(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ":")
	s = strings.TrimSpace(s)

	// Parse: "token" [type]
	// Find the token value between quotes
	tokenVal := ""
	tokenType := ""
	if len(s) > 0 && s[0] == '"' {
		end := strings.Index(s[1:], "\"")
		if end >= 0 {
			tokenVal = s[1 : end+1]
			rest := strings.TrimSpace(s[end+2:])
			if len(rest) > 1 && rest[0] == '[' {
				tokenType = strings.Trim(rest, "[]")
			}
		}
	}

	// Map token types to friendly names
	switch tokenType {
	case "EOF":
		return "end of file"
	case "ident":
		// Hide internal preprocessor names from user-facing errors
		if tokenVal == "__shell__" || tokenVal == "__capture__" || tokenVal == "__pipe_shell__" {
			return "expression"
		}
		if tokenVal != "" {
			return "\"" + tokenVal + "\""
		}
		return "identifier"
	case "str_lit":
		if tokenVal != "" {
			return "string " + tokenVal
		}
		return "string"
	case "integer":
		if tokenVal != "" {
			return "number " + tokenVal
		}
		return "number"
	case "float_lit":
		if tokenVal != "" {
			return "number " + tokenVal
		}
		return "decimal number"
	case "raw_str_lit":
		return "raw string"
	default:
		// For keyword tokens like ["end"], ["if"], etc. — show quoted
		if tokenVal != "" {
			return "\"" + tokenVal + "\""
		}
		if tokenType != "" {
			return "\"" + tokenType + "\""
		}
		return "token"
	}
}

// simplifyExpectedSet takes a space-separated list of parser symbols
// and returns a human-friendly description.
func simplifyExpectedSet(raw string) string {
	parts := strings.Fields(raw)

	var terminals []string
	seen := make(map[string]bool)
	for _, p := range parts {
		// Skip non-terminal grammar symbols (PascalCase names)
		if isGrammarSymbol(p) {
			continue
		}
		// Clean up the terminal name
		friendly := friendlyTerminal(p)
		if friendly != "" && !seen[friendly] {
			seen[friendly] = true
			terminals = append(terminals, friendly)
		}
	}

	if len(terminals) == 0 {
		return "an expression or statement"
	}
	if len(terminals) == 1 {
		return terminals[0]
	}
	if len(terminals) == 2 {
		return terminals[0] + " or " + terminals[1]
	}
	if len(terminals) <= 5 {
		return strings.Join(terminals[:len(terminals)-1], ", ") + ", or " + terminals[len(terminals)-1]
	}
	// Too many — summarize
	return strings.Join(terminals[:4], ", ") + ", ..."
}

// isGrammarSymbol returns true for internal parser non-terminal names
// like HashLit, ArrayLit, ParallelExpr, etc. These are PascalCase
// identifiers that should not be shown to users.
func isGrammarSymbol(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Quoted strings like "end", "if" are terminals — not grammar symbols
	if s[0] == '"' || s[0] == '\'' {
		return false
	}
	// Token type names that should be translated, not hidden
	switch s {
	case "str_lit", "raw_str_lit", "integer", "ident", "float_lit", "comp_op":
		return false
	}
	// PascalCase identifiers (starts with uppercase) are grammar non-terminals
	if s[0] >= 'A' && s[0] <= 'Z' {
		return true
	}
	return false
}

// friendlyTerminal converts a parser terminal symbol to a human-friendly string.
func friendlyTerminal(s string) string {
	// Token type names
	switch s {
	case "str_lit":
		return "a string"
	case "raw_str_lit":
		return "a string"
	case "integer":
		return "a number"
	case "float_lit":
		return "a number"
	case "ident":
		return "an identifier"
	case "comp_op":
		return "a comparison operator"
	}
	// Quoted keywords: "end", "if", etc.
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s // Already quoted and readable
	}
	// Single-char tokens in quotes: '(', ')', etc.
	if len(s) >= 3 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return "\"" + s[1:len(s)-1] + "\""
	}
	return ""
}

// isStatementExpectedSet returns true if the expected set looks like
// a full statement/expression set (contains "def", "if", "while", etc.).
// closestMatch finds the closest match to name from candidates
// using Levenshtein distance. Returns "" if no close match (distance > 2).
func closestMatch(name string, candidates []string) string {
	best := ""
	bestDist := 3 // threshold: max distance 2
	for _, c := range candidates {
		d := levenshtein(name, c)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	return best
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev = curr
	}
	return prev[lb]
}

func isStatementExpectedSet(raw string) bool {
	return strings.Contains(raw, `"def"`) && strings.Contains(raw, `"if"`) && strings.Contains(raw, `"while"`)
}

// isEndExpectedSet returns true when the expected set indicates a missing "end" keyword.
func isEndExpectedSet(raw string) bool {
	return strings.Contains(raw, `"end"`)
}

// findBlockMissingName checks if the error is caused by a "rats" or "bench"
// keyword that is missing its name string (either bare on its own line,
// or followed by a non-string token on the same line).
func findBlockMissingName(filename string, errorLine int) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	// Check current line and preceding lines
	for i := errorLine - 1; i >= 0 && i >= errorLine-3; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "rats" || strings.HasPrefix(trimmed, "rats ") || strings.HasPrefix(trimmed, "rats\t") {
			return "rats"
		}
		if trimmed == "bench" || strings.HasPrefix(trimmed, "bench ") || strings.HasPrefix(trimmed, "bench\t") {
			return "bench"
		}
	}
	return ""
}

// detectUnclosedBlock reads the source file and returns a description of
// the last unmatched block-opening keyword (e.g., `"def" block`).
func detectUnclosedBlock(filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	type blockInfo struct {
		keyword string
		line    int
	}
	var stack []blockInfo
	blockOpeners := map[string]bool{
		"def": true, "if": true, "while": true, "for": true,
		"rats": true, "bench": true, "fn": true,
		"spawn": true, "parallel": true, "try": true,
	}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Get first word
		word := trimmed
		if idx := strings.IndexAny(trimmed, " \t("); idx >= 0 {
			word = trimmed[:idx]
		}
		if blockOpeners[word] {
			stack = append(stack, blockInfo{keyword: word, line: i + 1})
		} else if word == "end" && len(stack) > 0 {
			stack = stack[:len(stack)-1]
		}
	}
	if len(stack) > 0 {
		b := stack[len(stack)-1]
		return fmt.Sprintf("\"%s\" block (opened at line %d)", b.keyword, b.line)
	}
	return ""
}

// translateBuildError post-processes `go build` stderr to translate
// Go compiler errors into Rugo-friendly messages.
func translateBuildError(stderr, sourceFile string) error {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")

	var translated []string
	for _, line := range lines {
		// Skip "# rugo_program" package header
		if strings.HasPrefix(line, "# ") {
			continue
		}
		// Translate lines referencing generated Go files into Rugo-friendly messages
		if strings.Contains(line, "main.go:") && strings.HasPrefix(strings.TrimSpace(line), "./") {
			// Translate "undefined: rugons_X_Y" to "undefined function X.Y"
			if strings.Contains(line, "undefined: rugons_") {
				if idx := strings.Index(line, "undefined: rugons_"); idx >= 0 {
					goIdent := line[idx+len("undefined: rugons_"):]
					// rugons_<ns>_<func> → ns.func
					if us := strings.Index(goIdent, "_"); us >= 0 {
						ns := goIdent[:us]
						fn := goIdent[us+1:]
						line = fmt.Sprintf("%s: undefined function %s.%s (check that the function exists in the required module)", sourceFile, ns, fn)
					}
				}
			} else {
				continue
			}
		}
		// Translate Go terms to Rugo terms
		line = strings.ReplaceAll(line, "continue is not in a loop", "next is not in a loop")
		line = strings.ReplaceAll(line, "break is not in a loop, switch, or select", "break is not in a loop")
		// Translate "not an interface" type assertion error to friendly lambda error
		if strings.Contains(line, "is not an interface") {
			// Go error: "invalid operation: x (variable of type int) is not an interface"
			// Translate to: "cannot call x — not a function (did you mean to assign a fn...end lambda?)"
			if start := strings.Index(line, "invalid operation: "); start >= 0 {
				rest := line[start+len("invalid operation: "):]
				varName := rest
				if spaceIdx := strings.IndexAny(rest, " ("); spaceIdx >= 0 {
					varName = rest[:spaceIdx]
				}
				prefix := line[:start]
				line = prefix + "cannot call " + varName + " — not a function (did you mean to assign a fn...end lambda?)"
			}
		}
		// Strip rugofn_ prefix from function names
		line = strings.ReplaceAll(line, "rugofn_", "")
		// Translate remaining rugons_ prefixes in any other error context
		for strings.Contains(line, "rugons_") {
			idx := strings.Index(line, "rugons_")
			rest := line[idx+7:]
			if us := strings.Index(rest, "_"); us >= 0 {
				ns := rest[:us]
				fn := rest[us+1:]
				// Find end of identifier
				end := 0
				for end < len(fn) && (fn[end] == '_' || (fn[end] >= 'a' && fn[end] <= 'z') || (fn[end] >= 'A' && fn[end] <= 'Z') || (fn[end] >= '0' && fn[end] <= '9')) {
					end++
				}
				line = line[:idx] + ns + "." + fn[:end] + line[idx+7+us+1+end:]
			} else {
				break
			}
		}
		// Clean up temp dir path prefix in file references
		if idx := strings.Index(line, sourceFile); idx > 0 {
			line = line[idx:]
		} else if colonIdx := strings.Index(line, ":"); colonIdx > 0 {
			// Try to strip ../tmpdir/ prefix
			prefix := line[:colonIdx]
			if strings.HasPrefix(prefix, "../") || strings.HasPrefix(prefix, "./") {
				base := filepath.Base(prefix)
				if strings.HasSuffix(base, ".rg") {
					line = base + line[colonIdx:]
				}
			}
		}
		translated = append(translated, line)
	}

	if len(translated) == 0 {
		// All lines were internal Go errors — include raw stderr for debugging
		return fmt.Errorf("compilation failed:\n%s", strings.TrimSpace(stderr))
	}
	return fmt.Errorf("%s", strings.Join(translated, "\n"))
}

// validateTopLevelOnly walks statement trees and returns an error if
// import or require statements appear inside function bodies or blocks.
func validateTopLevelOnly(stmts []Statement, sourceFile string) error {
	for _, s := range stmts {
		switch st := s.(type) {
		case *FuncDef:
			if err := rejectNestedImports(st.Body, sourceFile); err != nil {
				return err
			}
		case *IfStmt:
			if err := rejectNestedImports(st.Body, sourceFile); err != nil {
				return err
			}
			for _, clause := range st.ElsifClauses {
				if err := rejectNestedImports(clause.Body, sourceFile); err != nil {
					return err
				}
			}
			if err := rejectNestedImports(st.ElseBody, sourceFile); err != nil {
				return err
			}
		case *WhileStmt:
			if err := rejectNestedImports(st.Body, sourceFile); err != nil {
				return err
			}
		case *ForStmt:
			if err := rejectNestedImports(st.Body, sourceFile); err != nil {
				return err
			}
		case *TestDef:
			if err := rejectNestedImports(st.Body, sourceFile); err != nil {
				return err
			}
		}
	}
	return nil
}

// rejectNestedImports checks a block body for use/import/require statements
// and returns an error if any are found.
func rejectNestedImports(stmts []Statement, sourceFile string) error {
	for _, s := range stmts {
		switch s.(type) {
		case *UseStmt:
			return fmt.Errorf("%s:%d: use statements must be at the top level", sourceFile, s.StmtLine())
		case *ImportStmt:
			return fmt.Errorf("%s:%d: import statements must be at the top level", sourceFile, s.StmtLine())
		case *RequireStmt:
			return fmt.Errorf("%s:%d: require statements must be at the top level", sourceFile, s.StmtLine())
		}
		// Recurse into nested blocks
		switch st := s.(type) {
		case *FuncDef:
			if err := rejectNestedImports(st.Body, sourceFile); err != nil {
				return err
			}
		case *IfStmt:
			if err := rejectNestedImports(st.Body, sourceFile); err != nil {
				return err
			}
			for _, clause := range st.ElsifClauses {
				if err := rejectNestedImports(clause.Body, sourceFile); err != nil {
					return err
				}
			}
			if err := rejectNestedImports(st.ElseBody, sourceFile); err != nil {
				return err
			}
		case *WhileStmt:
			if err := rejectNestedImports(st.Body, sourceFile); err != nil {
				return err
			}
		case *ForStmt:
			if err := rejectNestedImports(st.Body, sourceFile); err != nil {
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

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// validateNamespace checks that a namespace name is a valid Rugo identifier
// and not a reserved keyword.
func validateNamespace(name string) error {
	if name == "" {
		return fmt.Errorf("cannot be empty")
	}
	// Must start with a letter or underscore
	if !((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z') || name[0] == '_') {
		return fmt.Errorf("must start with a letter or underscore")
	}
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return fmt.Errorf("contains invalid character %q", ch)
		}
	}
	if rugoKeywords[name] {
		return fmt.Errorf("%q is a reserved keyword", name)
	}
	return nil
}

// FindLocalEntryPoint resolves the entry point .rg file within a local directory.
// Resolution order: <dirname>.rg → main.rg → sole .rg file.
func FindLocalEntryPoint(dir string) (string, error) {
	name := filepath.Base(dir)

	// 1. <dirname>.rg
	candidate := filepath.Join(dir, name+".rg")
	if fileExists(candidate) {
		return candidate, nil
	}

	// 2. main.rg
	candidate = filepath.Join(dir, "main.rg")
	if fileExists(candidate) {
		return candidate, nil
	}

	// 3. Exactly one .rg file
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading directory %s: %w", dir, err)
	}
	var rgFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".rg") {
			rgFiles = append(rgFiles, filepath.Join(dir, e.Name()))
		}
	}
	if len(rgFiles) == 1 {
		return rgFiles[0], nil
	}

	if len(rgFiles) == 0 {
		return "", fmt.Errorf("no .rg files found in directory %q", dir)
	}
	return "", fmt.Errorf("cannot determine entry point for directory %q: found %d .rg files (add a %s.rg or main.rg, or use 'with' to select specific modules)", dir, len(rgFiles), name)
}
