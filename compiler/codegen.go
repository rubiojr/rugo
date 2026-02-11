package compiler

import (
	"github.com/rubiojr/rugo/ast"
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/rubiojr/rugo/gobridge"
	"github.com/rubiojr/rugo/modules"
	"github.com/rubiojr/rugo/parser"
)

//go:embed templates/runtime_core_pre.go.tmpl
var runtimeCorePre string

//go:embed templates/runtime_core_post.go.tmpl
var runtimeCorePost string

//go:embed templates/runtime_spawn.go.tmpl
var runtimeSpawn string

// codeGen generates Go source code from a typed AST.
type codeGen struct {
	sb              strings.Builder
	indent          int
	declared        map[string]bool // track declared variables per scope
	scopes          []map[string]bool
	constScopes     []map[string]int // track constant bindings: name → line of first assignment
	inFunc          bool
	imports         map[string]bool   // Rugo stdlib modules imported (via use)
	goImports       map[string]string // Go bridge packages: path → alias
	namespaces      map[string]bool   // known require namespaces
	nsVarNames      map[string]bool   // namespaced var names: "ns.name" → true
	sourceFile      string            // original source filename for //line directives
	hasSpawn        bool              // whether spawn is used
	hasParallel     bool              // whether parallel is used
	hasBench        bool              // whether bench blocks are present
	usesTaskMethods bool              // whether .value/.done/.wait appear
	funcDefs        map[string]int    // user function name → param count
	handlerVars     map[string]bool   // top-level vars promoted to package-level for handler access
	testMode        bool              // include rats blocks in output
	typeInfo        *TypeInfo         // inferred type information (nil disables typed codegen)
	currentFunc     *ast.FuncDef          // current function being generated (for type lookups)
	varTypeScope    string            // override scope key for varType lookups (test/bench blocks)
	inSpawn         int               // nesting depth of spawn blocks (>0 means inside spawn)
}

// generate produces Go source code from a ast.Program AST.
func generate(prog *ast.Program, sourceFile string, testMode bool) (string, error) {
	// Run type inference before code generation.
	ti := Infer(prog)

	g := &codeGen{
		declared:    make(map[string]bool),
		scopes:      []map[string]bool{make(map[string]bool)},
		constScopes: []map[string]int{make(map[string]int)},
		imports:     make(map[string]bool),
		goImports:   make(map[string]string),
		namespaces:  make(map[string]bool),
		nsVarNames:  make(map[string]bool),
		handlerVars: make(map[string]bool),
		sourceFile:  sourceFile,
		funcDefs:    make(map[string]int),
		testMode:    testMode,
		typeInfo:    ti,
	}
	return g.generate(prog)
}

func (g *codeGen) generate(prog *ast.Program) (string, error) {
	// Collect imports and separate functions, tests, benchmarks, and top-level statements
	var funcs []*ast.FuncDef
	var tests []*ast.TestDef
	var benches []*ast.BenchDef
	var topStmts []ast.Statement
	var nsVars []*ast.AssignStmt // top-level assignments from require'd files (emitted as package-level vars)
	var setupFunc *ast.FuncDef
	var teardownFunc *ast.FuncDef
	var setupFileFunc *ast.FuncDef
	var teardownFileFunc *ast.FuncDef
	funcLines := make(map[string]int) // track first definition line per function
	for _, s := range prog.Statements {
		switch st := s.(type) {
		case *ast.FuncDef:
			// Detect duplicate function definitions
			key := st.Name
			if st.Namespace != "" {
				key = st.Namespace + "." + st.Name
			}
			if prevLine, exists := funcLines[key]; exists {
				return "", fmt.Errorf("%s:%d: function %q already defined at line %d", g.sourceFile, st.SourceLine, st.Name, prevLine)
			}
			funcLines[key] = st.SourceLine

			if st.Name == "setup" && st.Namespace == "" {
				setupFunc = st
			} else if st.Name == "teardown" && st.Namespace == "" {
				teardownFunc = st
			} else if st.Name == "setup_file" && st.Namespace == "" {
				setupFileFunc = st
			} else if st.Name == "teardown_file" && st.Namespace == "" {
				teardownFileFunc = st
			}
			funcs = append(funcs, st)
		case *ast.TestDef:
			if g.testMode {
				tests = append(tests, st)
			}
		case *ast.BenchDef:
			benches = append(benches, st)
		case *ast.RequireStmt:
			continue
		case *ast.UseStmt:
			g.imports[st.Module] = true
			continue
		case *ast.ImportStmt:
			g.goImports[st.Package] = st.Alias
			continue
		default:
			if assign, ok := s.(*ast.AssignStmt); ok && assign.Namespace != "" {
				nsVars = append(nsVars, assign)
			} else {
				topStmts = append(topStmts, s)
			}
		}
		_ = s
	}

	// Build function definition registry for argument count validation
	for _, f := range funcs {
		key := f.Name
		if f.Namespace != "" {
			key = f.Namespace + "." + f.Name
			g.namespaces[f.Namespace] = true
		}
		g.funcDefs[key] = len(f.Params)
	}

	// Register namespaces from require'd constants
	for _, nv := range nsVars {
		g.namespaces[nv.Namespace] = true
		g.nsVarNames[nv.Namespace+"."+nv.Target] = true
	}

	// Identify top-level variables referenced by user-defined functions.
	// All def functions are emitted as top-level Go functions but top-level
	// variables live inside main(). Promote referenced vars to package-level.
	// Exclude variables that the function also assigns to (local shadows).
	{
		// Collect top-level assignment targets
		topVarNames := make(map[string]bool)
		for _, s := range topStmts {
			if a, ok := s.(*ast.AssignStmt); ok && a.Namespace == "" {
				topVarNames[a.Target] = true
			}
		}
		// Collect idents referenced by any non-namespaced function,
		// excluding those that the function itself assigns to.
		for _, f := range funcs {
			if f.Namespace != "" {
				continue
			}
			// Collect variables assigned inside this function body
			localAssigns := make(map[string]bool)
			for _, s := range f.Body {
				if a, ok := s.(*ast.AssignStmt); ok {
					localAssigns[a.Target] = true
				}
			}
			refs := collectIdents(f.Body)
			for name := range refs {
				if topVarNames[name] && !localAssigns[name] {
					g.handlerVars[name] = true
				}
			}
		}
		// Check test bodies for references to top-level variables.
		// rats blocks are emitted as separate Go functions and cannot
		// access top-level variables — report a clear error early.
		for _, t := range tests {
			localAssigns := make(map[string]bool)
			for _, s := range t.Body {
				if a, ok := s.(*ast.AssignStmt); ok {
					localAssigns[a.Target] = true
				}
			}
			refs := collectIdents(t.Body)
			for name := range refs {
				if !topVarNames[name] || localAssigns[name] {
					continue
				}
				hint := "use an environment variable to share state with rats blocks"
				if name[0] >= 'a' && name[0] <= 'z' {
					hint = "variables are block-scoped; use a constant (UPPERCASE) or an environment variable instead"
				}
				return "", fmt.Errorf("%s:%d: '%s' is not available inside rats blocks (%s)", g.sourceFile, t.SourceLine, name, hint)
			}
		}
	}

	// Detect spawn/parallel/bench usage to gate runtime emission and imports
	g.hasSpawn = astUsesSpawn(prog)
	g.hasParallel = astUsesParallel(prog)
	g.hasBench = len(benches) > 0
	g.usesTaskMethods = astUsesTaskMethods(prog)
	needsSpawnRuntime := g.hasSpawn || g.usesTaskMethods
	needsSyncImport := needsSpawnRuntime || g.hasParallel
	needsTimeImport := needsSpawnRuntime || g.hasBench

	g.writeln("package main")
	g.writeln("")
	g.writeln("import (")
	g.indent++
	g.writeln(`"fmt"`)
	g.writeln(`"os"`)
	g.writeln(`"os/exec"`)
	g.writeln(`"runtime/debug"`)
	g.writeln(`"sort"`)
	g.writeln(`"strings"`)
	if needsSyncImport {
		g.writeln(`"sync"`)
	}
	if needsTimeImport {
		g.writeln(`"time"`)
	}
	baseImports := map[string]bool{
		"fmt": true, "os": true, "os/exec": true,
		"runtime/debug": true, "strings": true, "sort": true,
	}
	emittedImports := make(map[string]bool)
	for k := range baseImports {
		emittedImports[k] = true
	}
	if needsSyncImport {
		emittedImports["sync"] = true
	}
	if needsTimeImport {
		emittedImports["time"] = true
	}
	// Emit Go imports for Rugo stdlib modules (use)
	for _, name := range importedModuleNames(g.imports) {
		if m, ok := modules.Get(name); ok {
			for _, imp := range m.GoImports {
				// Extract bare path from potentially aliased import
				barePath := imp
				if strings.Contains(imp, `"`) {
					// aliased import: alias "path"
					parts := strings.Fields(imp)
					if len(parts) == 2 {
						barePath = strings.Trim(parts[1], `"`)
					}
				}
				if emittedImports[barePath] {
					continue
				}
				emittedImports[barePath] = true
				if strings.Contains(imp, `"`) {
					g.writef("%s\n", imp)
				} else {
					g.writef("\"%s\"\n", imp)
				}
			}
		}
	}
	// Emit Go imports for Go bridge packages (import)
	// Packages with NoGoImport are skipped (runtime-only helpers)
	for _, pkg := range sortedGoBridgeImports(g.goImports) {
		if bp := gobridge.GetPackage(pkg); bp != nil && bp.NoGoImport {
			// Emit extra imports needed by runtime helpers (e.g. maps needs sort)
			for _, extra := range bp.ExtraImports {
				if !emittedImports[extra] {
					g.writef("\"%s\"\n", extra)
					emittedImports[extra] = true
				}
			}
			continue
		}
		alias := g.goImports[pkg]
		if alias == "" && emittedImports[pkg] {
			continue // already imported without alias
		}
		if alias != "" {
			g.writef("%s \"%s\"\n", alias, pkg)
		} else {
			emittedImports[pkg] = true
			g.writef("\"%s\"\n", pkg)
		}
	}
	g.indent--
	g.writeln(")")
	g.writeln("")

	// Silence unused import warnings
	g.writeln("var _ = fmt.Sprintf")
	g.writeln("var _ = os.Exit")
	g.writeln("var _ = exec.Command")
	g.writeln("var _ = strings.NewReader")
	g.writeln("var _ = sort.Slice")
	g.writeln("var _ = debug.Stack")
	if needsSyncImport {
		g.writeln("var _ sync.Once")
	}
	if needsTimeImport {
		g.writeln("var _ = time.Now")
	}
	g.writeln("")

	// Runtime helpers
	g.writeRuntime()

	// Package-level variables from require'd files
	for _, nv := range nsVars {
		expr, err := g.exprString(nv.Value)
		if err != nil {
			return "", err
		}
		g.writef("var rugons_%s_%s interface{} = %s\n", nv.Namespace, nv.Target, expr)
	}
	if len(nsVars) > 0 {
		g.writeln("")
	}

	// Package-level variables for user-defined function access
	if len(g.handlerVars) > 0 {
		names := make([]string, 0, len(g.handlerVars))
		for name := range g.handlerVars {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			g.writef("var %s interface{}\n", name)
		}
		g.writeln("")
	}

	// User-defined functions
	for _, f := range funcs {
		if err := g.writeFunc(f); err != nil {
			return "", err
		}
		g.writeln("")
	}

	// Dispatch maps for modules that declare DispatchEntry
	dispatchHandlers := collectDispatchHandlers(prog.Statements, g.imports)
	g.writeDispatchMaps(funcs, dispatchHandlers)

	if len(tests) > 0 {
		return g.generateTestHarness(tests, topStmts, setupFunc, teardownFunc, setupFileFunc, teardownFileFunc)
	}

	if len(benches) > 0 {
		return g.generateBenchHarness(benches, topStmts)
	}

	// Main function
	g.writeln("func main() {")
	g.indent++
	g.writePanicHandler()
	g.pushScope()
	for _, s := range topStmts {
		if err := g.writeStmt(s); err != nil {
			return "", err
		}
	}
	g.popScope()
	g.indent--
	g.writeln("}")

	return g.sb.String(), nil
}

func (g *codeGen) generateTestHarness(tests []*ast.TestDef, topStmts []ast.Statement, setup, teardown, setupFile, teardownFile *ast.FuncDef) (string, error) {
	// Emit each test as a function
	for i, t := range tests {
		funcName := fmt.Sprintf("rugo_test_%d", i)
		g.writef("func %s() (passed bool, skipped bool, skipReason string) {\n", funcName)
		g.indent++
		g.writeln("defer func() {")
		g.indent++
		g.writeln("if r := recover(); r != nil {")
		g.indent++
		g.writeln(`if reason, ok := r.(rugoTestSkip); ok {`)
		g.indent++
		g.writeln("skipped = true")
		g.writeln("skipReason = string(reason)")
		g.writeln("return")
		g.indent--
		g.writeln("}")
		g.writeln(`failColor := "\033[31m"`)
		g.writeln(`failReset := "\033[0m"`)
		g.writeln(`if os.Getenv("NO_COLOR") != "" {`)
		g.indent++
		g.writeln(`failColor = ""`)
		g.writeln(`failReset = ""`)
		g.indent--
		g.writeln(`}`)
		g.writeln(`fmt.Fprintf(os.Stderr, "  %sFAIL%s: %v\n", failColor, failReset, r)`)
		g.writeln("passed = false")
		g.indent--
		g.writeln("}")
		g.indent--
		g.writeln("}()")
		g.pushScope()
		g.varTypeScope = fmt.Sprintf("__test_%p", t)
		for _, s := range t.Body {
			if err := g.writeStmt(s); err != nil {
				return "", err
			}
		}
		g.varTypeScope = ""
		g.popScope()
		g.writeln("passed = true")
		g.writeln("return")
		g.indent--
		g.writeln("}")
		g.writeln("")
	}

	// Main function: delegate to runtime test runner
	g.writeln("func main() {")
	g.indent++
	g.writePanicHandler()
	g.pushScope()

	// Run top-level setup code
	for _, s := range topStmts {
		if err := g.writeStmt(s); err != nil {
			return "", err
		}
	}

	// Build test cases and call the runtime runner
	g.writeln("rugo_test_runner([]rugoTestCase{")
	g.indent++
	for i, t := range tests {
		escapedName := strings.ReplaceAll(t.Name, `"`, `\"`)
		g.writef("{Name: \"%s\", Func: rugo_test_%d},\n", escapedName, i)
	}
	g.indent--

	setupArg := "nil"
	teardownArg := "nil"
	setupFileArg := "nil"
	teardownFileArg := "nil"
	if setup != nil {
		setupArg = "rugofn_setup"
	}
	if teardown != nil {
		teardownArg = "rugofn_teardown"
	}
	if setupFile != nil {
		setupFileArg = "rugofn_setup_file"
	}
	if teardownFile != nil {
		teardownFileArg = "rugofn_teardown_file"
	}
	g.writef("}, %s, %s, %s, %s, _test)\n", setupArg, teardownArg, setupFileArg, teardownFileArg)

	g.popScope()
	g.indent--
	g.writeln("}")

	return g.sb.String(), nil
}

func (g *codeGen) generateBenchHarness(benches []*ast.BenchDef, topStmts []ast.Statement) (string, error) {
	// Emit each benchmark as a function
	for i, b := range benches {
		funcName := fmt.Sprintf("rugo_bench_%d", i)
		g.writef("func %s() {\n", funcName)
		g.indent++
		g.pushScope()
		g.varTypeScope = fmt.Sprintf("__bench_%p", b)
		for _, s := range b.Body {
			if err := g.writeStmt(s); err != nil {
				return "", err
			}
		}
		g.varTypeScope = ""
		g.popScope()
		g.indent--
		g.writeln("}")
		g.writeln("")
	}

	// Main function: run benchmarks via runtime runner
	g.writeln("func main() {")
	g.indent++
	g.writePanicHandler()
	g.pushScope()

	// Run top-level setup code (imports, variable defs, helper functions)
	for _, s := range topStmts {
		if err := g.writeStmt(s); err != nil {
			return "", err
		}
	}

	// Build bench cases and call the runtime runner
	g.writeln("rugo_bench_runner([]rugoBenchCase{")
	g.indent++
	for i, b := range benches {
		escapedName := strings.ReplaceAll(b.Name, `"`, `\"`)
		g.writef("{Name: \"%s\", Func: rugo_bench_%d},\n", escapedName, i)
	}
	g.indent--
	g.writeln("})")

	g.popScope()
	g.indent--
	g.writeln("}")

	return g.sb.String(), nil
}

func (g *codeGen) writeRuntime() {
	g.sb.WriteString(runtimeCorePre)

	// Module runtimes (only for use'd modules)
	for _, name := range importedModuleNames(g.imports) {
		if m, ok := modules.Get(name); ok {
			g.sb.WriteString(m.FullRuntime())
		}
	}

	g.sb.WriteString(runtimeCorePost)

	if g.hasSpawn || g.usesTaskMethods {
		g.writeSpawnRuntime()
	}

	// Go bridge helpers (only if any Go packages are imported)
	if len(g.goImports) > 0 {
		g.writeGoBridgeRuntime()
	}
}

func (g *codeGen) writeSpawnRuntime() {
	g.sb.WriteString(runtimeSpawn)
}

// writeDispatchMaps generates typed dispatch maps for modules that declare DispatchEntry.
// Each map maps user-defined function names to their Go implementations.
// When a module provides DispatchTransform, only functions matching transformed
// handler names from the source are included. Otherwise all eligible functions are included.
func (g *codeGen) writeDispatchMaps(funcs []*ast.FuncDef, handlers map[string]bool) {
	for _, name := range importedModuleNames(g.imports) {
		m, ok := modules.Get(name)
		if !ok || m.DispatchEntry == "" {
			continue
		}
		// Build the set of transformed handler names for this module
		var resolved map[string]bool
		if m.DispatchTransform != nil {
			resolved = make(map[string]bool)
			for h := range handlers {
				resolved[m.DispatchTransform(h)] = true
			}
		}
		g.writef("var rugo_%s_dispatch = map[string]func(interface{}) interface{}{\n", m.Name)
		g.indent++
		for _, f := range funcs {
			if len(f.Params) != 1 {
				continue
			}
			if resolved != nil && !resolved[f.Name] {
				continue
			}
			// Use un-namespaced name as dispatch key, full Go name as value
			var goName string
			if f.Namespace != "" {
				goName = fmt.Sprintf("rugons_%s_%s", f.Namespace, f.Name)
			} else {
				goName = fmt.Sprintf("rugofn_%s", f.Name)
			}
			g.writef("%q: %s,\n", f.Name, goName)
		}
		g.indent--
		g.writeln("}")
		g.writeln("")
	}
}

// collectDispatchHandlers scans top-level statements for module method calls
// that register handler functions (e.g. web.get("/", "handler"), cli.cmd("greet", "fn"))
// and returns the set of handler function names referenced.
func collectDispatchHandlers(stmts []ast.Statement, imports map[string]bool) map[string]bool {
	handlers := make(map[string]bool)
	// Collect module names that have dispatch entries
	dispatchModules := make(map[string]bool)
	for name := range imports {
		if m, ok := modules.Get(name); ok && m.DispatchEntry != "" {
			dispatchModules[m.Name] = true
		}
	}
	if len(dispatchModules) == 0 {
		return handlers
	}
	for _, s := range stmts {
		collectDispatchHandlersFromStmt(s, dispatchModules, handlers)
	}
	return handlers
}

func collectDispatchHandlersFromStmt(s ast.Statement, dispatchModules map[string]bool, handlers map[string]bool) {
	switch st := s.(type) {
	case *ast.ExprStmt:
		collectDispatchHandlersFromExpr(st.Expression, dispatchModules, handlers)
	case *ast.IfStmt:
		for _, b := range st.Body {
			collectDispatchHandlersFromStmt(b, dispatchModules, handlers)
		}
		for _, ei := range st.ElsifClauses {
			for _, b := range ei.Body {
				collectDispatchHandlersFromStmt(b, dispatchModules, handlers)
			}
		}
		for _, b := range st.ElseBody {
			collectDispatchHandlersFromStmt(b, dispatchModules, handlers)
		}
	case *ast.ForStmt:
		for _, b := range st.Body {
			collectDispatchHandlersFromStmt(b, dispatchModules, handlers)
		}
	case *ast.WhileStmt:
		for _, b := range st.Body {
			collectDispatchHandlersFromStmt(b, dispatchModules, handlers)
		}
	case *ast.FuncDef:
		for _, b := range st.Body {
			collectDispatchHandlersFromStmt(b, dispatchModules, handlers)
		}
	}
}

func collectDispatchHandlersFromExpr(e ast.Expr, dispatchModules map[string]bool, handlers map[string]bool) {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return
	}
	// Check if this is a module.method() call on a dispatch module
	dot, ok := call.Func.(*ast.DotExpr)
	if !ok {
		return
	}
	ident, ok := dot.Object.(*ast.IdentExpr)
	if !ok || !dispatchModules[ident.Name] {
		return
	}
	// Extract string literal arguments as potential handler names
	for _, arg := range call.Args {
		if str, ok := arg.(*ast.StringLiteral); ok {
			handlers[str.Value] = true
		}
	}
}

func (g *codeGen) writeFunc(f *ast.FuncDef) error {
	// Use the function's original source file for //line directives if available.
	if f.SourceFile != "" {
		saved := g.sourceFile
		g.sourceFile = f.SourceFile
		defer func() { g.sourceFile = saved }()
	}

	// Check if this function has typed inference info.
	fti := g.funcTypeInfo(f)

	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		if fti != nil && fti.ParamTypes[i].IsTyped() {
			params[i] = p + " " + fti.ParamTypes[i].GoType()
		} else {
			params[i] = p + " interface{}"
		}
	}

	// Determine function name: namespaced or local
	var goName string
	if f.Namespace != "" {
		goName = fmt.Sprintf("rugons_%s_%s", f.Namespace, f.Name)
	} else {
		goName = fmt.Sprintf("rugofn_%s", f.Name)
	}

	retType := "interface{}"
	if fti != nil && fti.ReturnType.IsTyped() {
		retType = fti.ReturnType.GoType()
	}

	g.writef("func %s(%s) %s {\n", goName, strings.Join(params, ", "), retType)
	g.indent++
	g.pushScope()
	// Mark params as declared
	for _, p := range f.Params {
		g.declareVar(p)
	}
	g.currentFunc = f
	g.inFunc = true
	hasImplicitReturn := false
	for i, s := range f.Body {
		// Implicit return: last expression in function body becomes the return value.
		if i == len(f.Body)-1 {
			if es, ok := s.(*ast.ExprStmt); ok {
				g.emitLineDirective(es.SourceLine)
				expr, err := g.exprString(es.Expression)
				if err != nil {
					return err
				}
				g.writef("return %s\n", expr)
				hasImplicitReturn = true
				continue
			}
		}
		if err := g.writeStmt(s); err != nil {
			return err
		}
	}
	if !hasImplicitReturn {
		// Default return: typed zero value or nil.
		if fti != nil && fti.ReturnType.IsTyped() {
			g.writef("return %s\n", typedZero(fti.ReturnType))
		} else {
			g.writeln("return nil")
		}
	}
	g.inFunc = false
	g.currentFunc = nil
	g.popScope()
	g.indent--
	g.writeln("}")
	return nil
}

// funcTypeInfo returns the inferred type info for a function, or nil.
func (g *codeGen) funcTypeInfo(f *ast.FuncDef) *FuncTypeInfo {
	if g.typeInfo == nil {
		return nil
	}
	return g.typeInfo.FuncTypes[funcKey(f)]
}

// typedZero returns the zero value for a typed return.
func typedZero(t RugoType) string {
	switch t {
	case TypeInt:
		return "0"
	case TypeFloat:
		return "0.0"
	case TypeString:
		return `""`
	case TypeBool:
		return "false"
	default:
		return "nil"
	}
}

// emitLineDirective writes a //line directive for the original source file.
func (g *codeGen) emitLineDirective(line int) {
	if line > 0 && g.sourceFile != "" {
		g.sb.WriteString(fmt.Sprintf("//line %s:%d\n", g.sourceFile, line))
	}
}

func (g *codeGen) writeStmt(s ast.Statement) error {
	g.emitLineDirective(s.StmtLine())
	var err error
	switch st := s.(type) {
	case *ast.AssignStmt:
		err = g.writeAssign(st)
	case *ast.IndexAssignStmt:
		err = g.writeIndexAssign(st)
	case *ast.DotAssignStmt:
		err = g.writeDotAssign(st)
	case *ast.ExprStmt:
		err = g.writeExprStmt(st)
	case *ast.IfStmt:
		err = g.writeIf(st)
	case *ast.WhileStmt:
		err = g.writeWhile(st)
	case *ast.ForStmt:
		err = g.writeFor(st)
	case *ast.BreakStmt:
		g.writeln("break")
		return nil
	case *ast.NextStmt:
		g.writeln("continue")
		return nil
	case *ast.ReturnStmt:
		err = g.writeReturn(st)
	case *ast.FuncDef:
		err = fmt.Errorf("nested function definitions not supported")
	case *ast.RequireStmt:
		return nil
	case *ast.ImportStmt:
		return nil
	default:
		err = fmt.Errorf("unknown statement type: %T", s)
	}
	if err != nil {
		return g.stmtError(s, err)
	}
	return nil
}

// stmtError wraps a codegen error with file:line context from the statement.
func (g *codeGen) stmtError(s ast.Statement, err error) error {
	line := s.StmtLine()
	msg := err.Error()
	// Strip existing "line N: " prefix if present
	if strings.HasPrefix(msg, "line ") {
		if idx := strings.Index(msg, ": "); idx != -1 {
			msg = msg[idx+2:]
		}
	}
	if line > 0 && g.sourceFile != "" {
		return fmt.Errorf("%s:%d: %s", g.sourceFile, line, msg)
	}
	return err
}

func (g *codeGen) writeAssign(a *ast.AssignStmt) error {
	// Uppercase names are constants — reject reassignment
	if origLine, ok := g.constantLine(a.Target); ok {
		return fmt.Errorf("cannot reassign constant %s (first assigned at line %d)", a.Target, origLine)
	}

	exprType := g.exprType(a.Value)
	varType := g.varType(a.Target)

	// If the variable is dynamic but the expression is typed, box the value.
	expr, err := g.exprString(a.Value)
	if err != nil {
		return err
	}
	if !varType.IsTyped() && exprType.IsTyped() {
		expr = fmt.Sprintf("interface{}(%s)", expr)
	}

	if g.isDeclared(a.Target) || g.handlerVars[a.Target] {
		g.writef("%s = %s\n", a.Target, expr)
		// Track constants for handler vars on first use in main()
		if g.handlerVars[a.Target] && !g.isDeclared(a.Target) {
			g.declareVar(a.Target)
			if len(a.Target) > 0 && a.Target[0] >= 'A' && a.Target[0] <= 'Z' {
				g.declareConst(a.Target, a.SourceLine)
			}
		}
	} else {
		g.writef("%s := %s\n", a.Target, expr)
		g.declareVar(a.Target)
		if len(a.Target) > 0 && a.Target[0] >= 'A' && a.Target[0] <= 'Z' {
			g.declareConst(a.Target, a.SourceLine)
		}
	}
	// Suppress "declared but not used" by referencing with _
	g.writef("_ = %s\n", a.Target)
	return nil
}

func (g *codeGen) writeIndexAssign(ia *ast.IndexAssignStmt) error {
	obj, err := g.exprString(ia.Object)
	if err != nil {
		return err
	}
	idx, err := g.exprString(ia.Index)
	if err != nil {
		return err
	}
	val, err := g.exprString(ia.Value)
	if err != nil {
		return err
	}
	g.writef("rugo_index_set(%s, %s, %s)\n", obj, idx, val)
	return nil
}

func (g *codeGen) writeDotAssign(da *ast.DotAssignStmt) error {
	if da.Field == "__type__" {
		return fmt.Errorf("cannot assign to .__type__ — use type_of() for type introspection")
	}
	obj, err := g.exprString(da.Object)
	if err != nil {
		return err
	}
	val, err := g.exprString(da.Value)
	if err != nil {
		return err
	}
	g.writef("rugo_dot_set(%s, %q, %s)\n", obj, da.Field, val)
	return nil
}

func (g *codeGen) writeExprStmt(e *ast.ExprStmt) error {
	expr, err := g.exprString(e.Expression)
	if err != nil {
		return err
	}
	g.writef("_ = %s\n", expr)
	return nil
}

func (g *codeGen) writeIf(i *ast.IfStmt) error {
	// Pre-declare variables assigned in any branch so they're visible
	// after the if block (Ruby-like scoping: if/else doesn't create a new scope).
	var allBranches []ast.Statement
	allBranches = append(allBranches, i.Body...)
	for _, ec := range i.ElsifClauses {
		allBranches = append(allBranches, ec.Body...)
	}
	allBranches = append(allBranches, i.ElseBody...)
	for _, name := range collectAssignTargets(allBranches) {
		if !g.isDeclared(name) {
			varType := g.varType(name)
			if varType.IsTyped() {
				g.writef("var %s %s\n", name, varType.GoType())
			} else {
				g.writef("var %s interface{}\n", name)
			}
			g.declareVar(name)
		}
	}

	cond, err := g.exprString(i.Condition)
	if err != nil {
		return err
	}
	g.writef("if %s {\n", g.condExpr(cond, i.Condition))
	g.indent++
	for _, s := range i.Body {
		if err := g.writeStmt(s); err != nil {
			return err
		}
	}
	g.indent--
	for _, ec := range i.ElsifClauses {
		cond, err := g.exprString(ec.Condition)
		if err != nil {
			return err
		}
		g.writef("} else if %s {\n", g.condExpr(cond, ec.Condition))
		g.indent++
		for _, s := range ec.Body {
			if err := g.writeStmt(s); err != nil {
				return err
			}
		}
		g.indent--
	}
	if len(i.ElseBody) > 0 {
		g.writeln("} else {")
		g.indent++
		for _, s := range i.ElseBody {
			if err := g.writeStmt(s); err != nil {
				return err
			}
		}
		g.indent--
	}
	g.writeln("}")
	return nil
}

// collectAssignTargets returns variable names assigned in a list of statements,
// in order of first appearance. It recurses into nested if/else blocks.
func collectAssignTargets(stmts []ast.Statement) []string {
	var names []string
	seen := make(map[string]bool)
	var collect func([]ast.Statement)
	collect = func(stmts []ast.Statement) {
		for _, s := range stmts {
			switch st := s.(type) {
			case *ast.AssignStmt:
				if !seen[st.Target] {
					names = append(names, st.Target)
					seen[st.Target] = true
				}
			case *ast.IfStmt:
				collect(st.Body)
				for _, clause := range st.ElsifClauses {
					collect(clause.Body)
				}
				collect(st.ElseBody)
			}
		}
	}
	collect(stmts)
	return names
}

func (g *codeGen) writeWhile(w *ast.WhileStmt) error {
	cond, err := g.exprString(w.Condition)
	if err != nil {
		return err
	}
	g.writef("for %s {\n", g.condExpr(cond, w.Condition))
	g.indent++
	g.pushScope()
	for _, s := range w.Body {
		if err := g.writeStmt(s); err != nil {
			return err
		}
	}
	g.popScope()
	g.indent--
	g.writeln("}")
	return nil
}

func (g *codeGen) writeFor(f *ast.ForStmt) error {
	coll, err := g.exprString(f.Collection)
	if err != nil {
		return err
	}

	iterVar := f.Var
	idxVar := f.IndexVar

	// Declare the loop variable(s)
	if idxVar != "" {
		// Two-variable form: for key, val in hash / for idx, val in arr
		g.writef("for _, rugo_for_kv := range rugo_iterable(%s) {\n", coll)
		g.indent++
		g.pushScope()
		if iterVar == "_" {
			g.writef("_ = rugo_for_kv.Key\n")
		} else {
			g.writef("%s := rugo_for_kv.Key\n", iterVar)
			g.writef("_ = %s\n", iterVar)
			g.declareVar(iterVar)
		}
		if idxVar == "_" {
			g.writef("_ = rugo_for_kv.Val\n")
		} else {
			g.writef("%s := rugo_for_kv.Val\n", idxVar)
			g.writef("_ = %s\n", idxVar)
			g.declareVar(idxVar)
		}
	} else {
		// Single-variable form: for val in arr / for key in hash
		g.writef("for _, rugo_for_item := range rugo_iterable_default(%s) {\n", coll)
		g.indent++
		g.pushScope()
		g.writef("%s := rugo_for_item\n", iterVar)
		g.writef("_ = %s\n", iterVar)
		g.declareVar(iterVar)
	}

	for _, s := range f.Body {
		if err := g.writeStmt(s); err != nil {
			return err
		}
	}
	g.popScope()
	g.indent--
	g.writeln("}")
	return nil
}

func (g *codeGen) writeReturn(r *ast.ReturnStmt) error {
	// Inside a spawn block, return EXPR must assign to t.result and
	// use a bare return (the goroutine closure has no return value).
	if g.inSpawn > 0 {
		if r.Value != nil {
			expr, err := g.exprString(r.Value)
			if err != nil {
				return err
			}
			g.writef("t.result = %s\n", expr)
		}
		g.writeln("return")
		return nil
	}
	fti := g.currentFuncTypeInfo()
	if r.Value == nil {
		if fti != nil && fti.ReturnType.IsTyped() {
			g.writef("return %s\n", typedZero(fti.ReturnType))
		} else {
			g.writeln("return nil")
		}
	} else {
		expr, err := g.exprString(r.Value)
		if err != nil {
			return err
		}
		g.writef("return %s\n", expr)
	}
	return nil
}

func (g *codeGen) exprString(e ast.Expr) (string, error) {
	switch ex := e.(type) {
	case *ast.IntLiteral:
		if g.exprIsTyped(e) {
			return ex.Value, nil
		}
		return fmt.Sprintf("interface{}(%s)", ex.Value), nil
	case *ast.FloatLiteral:
		if g.exprIsTyped(e) {
			return ex.Value, nil
		}
		return fmt.Sprintf("interface{}(%s)", ex.Value), nil
	case *ast.BoolLiteral:
		if g.exprIsTyped(e) {
			if ex.Value {
				return "true", nil
			}
			return "false", nil
		}
		if ex.Value {
			return "interface{}(true)", nil
		}
		return "interface{}(false)", nil
	case *ast.NilLiteral:
		return "interface{}(nil)", nil
	case *ast.StringLiteral:
		if ex.Raw {
			escaped := goEscapeString(ex.Value)
			if g.exprIsTyped(e) {
				return fmt.Sprintf(`"%s"`, escaped), nil
			}
			return fmt.Sprintf(`interface{}("%s")`, escaped), nil
		}
		return g.stringLiteral(ex.Value, g.exprIsTyped(e))
	case *ast.IdentExpr:
		// Bare function name without parens: treat as zero-arg call (Ruby semantics).
		// Local variables shadow function names.
		if !g.isDeclared(ex.Name) {
			if expected, ok := g.funcDefs[ex.Name]; ok {
				if expected != 0 {
					return "", fmt.Errorf("function '%s' expects %d argument(s), called with 0", ex.Name, expected)
				}
				call := fmt.Sprintf("rugofn_%s()", ex.Name)
				if g.typeInfo != nil {
					if fti, ok := g.typeInfo.FuncTypes[ex.Name]; ok && fti.ReturnType.IsTyped() {
						return call, nil
					}
				}
				return fmt.Sprintf("interface{}(%s)", call), nil
			}
		}
		// Sibling constant reference within a namespace
		if g.currentFunc != nil && g.currentFunc.Namespace != "" && !g.isDeclared(ex.Name) {
			nsKey := g.currentFunc.Namespace + "." + ex.Name
			if g.nsVarNames[nsKey] {
				return fmt.Sprintf("rugons_%s_%s", g.currentFunc.Namespace, ex.Name), nil
			}
		}
		return ex.Name, nil
	case *ast.DotExpr:
		return g.dotExpr(ex)
	case *ast.BinaryExpr:
		return g.binaryExpr(ex)
	case *ast.UnaryExpr:
		return g.unaryExpr(ex)
	case *ast.CallExpr:
		return g.callExpr(ex)
	case *ast.IndexExpr:
		return g.indexExpr(ex)
	case *ast.SliceExpr:
		return g.sliceExpr(ex)
	case *ast.ArrayLiteral:
		return g.arrayLiteral(ex)
	case *ast.HashLiteral:
		return g.hashLiteral(ex)
	case *ast.TryExpr:
		return g.tryExpr(ex)
	case *ast.SpawnExpr:
		return g.spawnExpr(ex)
	case *ast.ParallelExpr:
		return g.parallelExpr(ex)
	case *ast.FnExpr:
		return g.fnExpr(ex)
	default:
		return "", fmt.Errorf("unknown expression type: %T", e)
	}
}

func (g *codeGen) stringLiteral(value string, typed bool) (string, error) {
	if ast.HasInterpolation(value) {
		format, exprStrs := ast.ProcessInterpolation(value)
		args := make([]string, len(exprStrs))
		for i, exprStr := range exprStrs {
			// Parse the interpolated expression through the rugo pipeline
			goExpr, err := g.compileInterpolatedExpr(exprStr)
			if err != nil {
				return "", fmt.Errorf("interpolation error in #{%s}: %w", exprStr, err)
			}
			args[i] = goExpr
		}
		escapedFmt := goEscapeString(format)
		if len(args) > 0 {
			return fmt.Sprintf(`fmt.Sprintf("%s", %s)`, escapedFmt, strings.Join(args, ", ")), nil
		}
		if typed {
			return fmt.Sprintf(`"%s"`, escapedFmt), nil
		}
		return fmt.Sprintf(`interface{}("%s")`, escapedFmt), nil
	}
	escaped := goEscapeString(value)
	if typed {
		return fmt.Sprintf(`"%s"`, escaped), nil
	}
	return fmt.Sprintf(`interface{}("%s")`, escaped), nil
}

// compileInterpolatedExpr parses a rugo expression string and generates Go code.
func (g *codeGen) compileInterpolatedExpr(exprStr string) (string, error) {
	// Wrap in a dummy statement so the parser can handle it
	src := exprStr + "\n"
	p := &parser.Parser{}
	// Parse as a full program with just this expression
	fullSrc := src
	flatAST, err := p.Parse("<interpolation>", []byte(fullSrc))
	if err != nil {
		return "", fmt.Errorf("parsing: %w", err)
	}
	prog, err := ast.Walk(p, flatAST)
	if err != nil {
		return "", fmt.Errorf("walking: %w", err)
	}
	if len(prog.Statements) == 0 {
		return `interface{}("")`, nil
	}
	// Extract the expression from the statement
	switch s := prog.Statements[0].(type) {
	case *ast.ExprStmt:
		return g.exprString(s.Expression)
	case *ast.AssignStmt:
		return g.exprString(s.Value)
	default:
		return "", fmt.Errorf("unexpected statement type in interpolation: %T", s)
	}
}

func goEscapeString(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '\\':
			sb.WriteString(`\\`)
		case ch == '"':
			sb.WriteString(`\"`)
		case ch == '\n':
			sb.WriteString(`\n`)
		case ch == '\r':
			sb.WriteString(`\r`)
		case ch == '\t':
			sb.WriteString(`\t`)
		case ch < 0x20 || ch == 0x7f:
			fmt.Fprintf(&sb, `\x%02x`, ch)
		default:
			sb.WriteByte(ch)
		}
	}
	return sb.String()
}

func (g *codeGen) binaryExpr(e *ast.BinaryExpr) (string, error) {
	leftType := g.exprType(e.Left)
	rightType := g.exprType(e.Right)

	left, err := g.exprString(e.Left)
	if err != nil {
		return "", err
	}
	right, err := g.exprString(e.Right)
	if err != nil {
		return "", err
	}

	// Typed native ops: emit direct Go operators when both sides are typed
	// AND will actually produce typed Go values (not interface{}).
	switch e.Op {
	case "+":
		if leftType == TypeInt && rightType == TypeInt && g.goTyped(e.Left) && g.goTyped(e.Right) {
			return fmt.Sprintf("(%s + %s)", left, right), nil
		}
		if leftType.IsNumeric() && rightType.IsNumeric() && leftType.IsTyped() && rightType.IsTyped() && g.goTyped(e.Left) && g.goTyped(e.Right) {
			return fmt.Sprintf("(%s + %s)", g.ensureFloat(left, leftType), g.ensureFloat(right, rightType)), nil
		}
		if leftType == TypeString && rightType == TypeString && g.goTyped(e.Left) && g.goTyped(e.Right) {
			return fmt.Sprintf("(%s + %s)", left, right), nil
		}
		return fmt.Sprintf("rugo_add(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case "-":
		if leftType == TypeInt && rightType == TypeInt && g.goTyped(e.Left) && g.goTyped(e.Right) {
			return fmt.Sprintf("(%s - %s)", left, right), nil
		}
		if leftType.IsNumeric() && rightType.IsNumeric() && leftType.IsTyped() && rightType.IsTyped() && g.goTyped(e.Left) && g.goTyped(e.Right) {
			return fmt.Sprintf("(%s - %s)", g.ensureFloat(left, leftType), g.ensureFloat(right, rightType)), nil
		}
		return fmt.Sprintf("rugo_sub(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case "*":
		if leftType == TypeInt && rightType == TypeInt && g.goTyped(e.Left) && g.goTyped(e.Right) {
			return fmt.Sprintf("(%s * %s)", left, right), nil
		}
		if leftType.IsNumeric() && rightType.IsNumeric() && leftType.IsTyped() && rightType.IsTyped() && g.goTyped(e.Left) && g.goTyped(e.Right) {
			return fmt.Sprintf("(%s * %s)", g.ensureFloat(left, leftType), g.ensureFloat(right, rightType)), nil
		}
		return fmt.Sprintf("rugo_mul(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case "/":
		if leftType == TypeInt && rightType == TypeInt && g.goTyped(e.Left) && g.goTyped(e.Right) {
			return fmt.Sprintf("(%s / %s)", left, right), nil
		}
		if leftType.IsNumeric() && rightType.IsNumeric() && leftType.IsTyped() && rightType.IsTyped() && g.goTyped(e.Left) && g.goTyped(e.Right) {
			return fmt.Sprintf("(%s / %s)", g.ensureFloat(left, leftType), g.ensureFloat(right, rightType)), nil
		}
		return fmt.Sprintf("rugo_div(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case "%":
		if leftType == TypeInt && rightType == TypeInt && g.goTyped(e.Left) && g.goTyped(e.Right) {
			return fmt.Sprintf("(%s %% %s)", left, right), nil
		}
		return fmt.Sprintf("rugo_mod(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case "==":
		if leftType == rightType && leftType.IsTyped() {
			return fmt.Sprintf("(%s == %s)", left, right), nil
		}
		return fmt.Sprintf("rugo_eq(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case "!=":
		if leftType == rightType && leftType.IsTyped() {
			return fmt.Sprintf("(%s != %s)", left, right), nil
		}
		return fmt.Sprintf("rugo_neq(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case "<":
		if leftType == rightType && leftType.IsTyped() && (leftType.IsNumeric() || leftType == TypeString) {
			return fmt.Sprintf("(%s < %s)", left, right), nil
		}
		return fmt.Sprintf("rugo_lt(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case ">":
		if leftType == rightType && leftType.IsTyped() && (leftType.IsNumeric() || leftType == TypeString) {
			return fmt.Sprintf("(%s > %s)", left, right), nil
		}
		return fmt.Sprintf("rugo_gt(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case "<=":
		if leftType == rightType && leftType.IsTyped() && (leftType.IsNumeric() || leftType == TypeString) {
			return fmt.Sprintf("(%s <= %s)", left, right), nil
		}
		return fmt.Sprintf("rugo_le(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case ">=":
		if leftType == rightType && leftType.IsTyped() && (leftType.IsNumeric() || leftType == TypeString) {
			return fmt.Sprintf("(%s >= %s)", left, right), nil
		}
		return fmt.Sprintf("rugo_ge(%s, %s)", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case "&&":
		if leftType == TypeBool && rightType == TypeBool {
			return fmt.Sprintf("(%s && %s)", left, right), nil
		}
		return fmt.Sprintf("interface{}(rugo_to_bool(%s) && rugo_to_bool(%s))", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	case "||":
		if leftType == TypeBool && rightType == TypeBool {
			return fmt.Sprintf("(%s || %s)", left, right), nil
		}
		return fmt.Sprintf("interface{}(rugo_to_bool(%s) || rugo_to_bool(%s))", g.boxed(left, leftType), g.boxed(right, rightType)), nil

	default:
		return "", fmt.Errorf("unknown operator: %s", e.Op)
	}
}

func (g *codeGen) unaryExpr(e *ast.UnaryExpr) (string, error) {
	operandType := g.exprType(e.Operand)
	operand, err := g.exprString(e.Operand)
	if err != nil {
		return "", err
	}
	switch e.Op {
	case "-":
		if operandType == TypeInt || operandType == TypeFloat {
			return fmt.Sprintf("(-%s)", operand), nil
		}
		return fmt.Sprintf("rugo_negate(%s)", g.boxed(operand, operandType)), nil
	case "!":
		if operandType == TypeBool {
			return fmt.Sprintf("(!%s)", operand), nil
		}
		return fmt.Sprintf("rugo_not(%s)", g.boxed(operand, operandType)), nil
	default:
		return "", fmt.Errorf("unknown unary operator: %s", e.Op)
	}
}

func (g *codeGen) dotExpr(e *ast.DotExpr) (string, error) {
	if e.Field == "__type__" {
		return "", fmt.Errorf("cannot access .__type__ directly — use type_of() instead")
	}
	// Rugo stdlib or namespace access without call
	if ns, ok := e.Object.(*ast.IdentExpr); ok {
		nsName := ns.Name
		// Local variables shadow namespaces for dot access
		if !g.isDeclared(nsName) {
			if g.imports[nsName] {
				if goFunc, ok := modules.LookupFunc(nsName, e.Field); ok {
					return fmt.Sprintf("interface{}(%s)", goFunc), nil
				}
			}
			// Go bridge function reference (without call)
			if pkg, ok := gobridge.PackageForNS(nsName, g.goImports); ok {
				if sig, ok := gobridge.Lookup(pkg, e.Field); ok {
					_ = sig
					return "", fmt.Errorf("Go bridge function %s.%s must be called with arguments", nsName, e.Field)
				}
			}
			// Known require namespace — function reference
			if g.namespaces[nsName] {
				return fmt.Sprintf("interface{}(rugons_%s_%s)", nsName, e.Field), nil
			}
		}
		// Not a known namespace or shadowed by variable — dot access (handles both hashes and tasks at runtime)
		g.usesTaskMethods = g.usesTaskMethods || taskMethodNames[e.Field]
		return fmt.Sprintf("rugo_dot_get(%s, %q)", nsName, e.Field), nil
	}
	obj, err := g.exprString(e.Object)
	if err != nil {
		return "", err
	}
	// Dot access on non-ident expressions (handles both hashes and tasks at runtime)
	g.usesTaskMethods = g.usesTaskMethods || taskMethodNames[e.Field]
	return fmt.Sprintf("rugo_dot_get(%s, %q)", obj, e.Field), nil
}

func (g *codeGen) callExpr(e *ast.CallExpr) (string, error) {
	args := make([]string, len(e.Args))
	for i, a := range e.Args {
		s, err := g.exprString(a)
		if err != nil {
			return "", err
		}
		args[i] = s
	}
	argStr := strings.Join(args, ", ")

	// Check for namespaced function calls: ns.func(args)
	if dot, ok := e.Func.(*ast.DotExpr); ok {
		if ns, ok := dot.Object.(*ast.IdentExpr); ok {
			nsName := ns.Name
			// Local variables shadow namespaces for dot calls
			if !g.isDeclared(nsName) {
				// Rugo stdlib module call
				if g.imports[nsName] {
					if goFunc, ok := modules.LookupFunc(nsName, dot.Field); ok {
						return fmt.Sprintf("%s(%s)", goFunc, argStr), nil
					}
					return "", fmt.Errorf("unknown function %s.%s in module %q", nsName, dot.Field, nsName)
				}
				// Go bridge call
				if pkg, ok := gobridge.PackageForNS(nsName, g.goImports); ok {
					if sig, ok := gobridge.Lookup(pkg, dot.Field); ok {
						if !sig.Variadic && len(e.Args) != len(sig.Params) {
							return "", argCountError(nsName+"."+dot.Field, len(e.Args), len(sig.Params))
						}
						return g.generateGoBridgeCall(pkg, sig, args, nsName+"."+dot.Field), nil
					}
					return "", fmt.Errorf("unknown function %s.%s in Go bridge package %q", nsName, dot.Field, pkg)
				}
				// Known require namespace
				if g.namespaces[nsName] {
					nsKey := nsName + "." + dot.Field
					if expected, ok := g.funcDefs[nsKey]; ok {
						if len(e.Args) != expected {
							return "", argCountError(nsName+"."+dot.Field, len(e.Args), expected)
						}
					}
					typedArgs := g.typedCallArgs(nsKey, args, e.Args)
					return fmt.Sprintf("rugons_%s_%s(%s)", nsName, dot.Field, typedArgs), nil
				}
			}
			// Not a known namespace or shadowed by variable — dispatch via generic DotCall
			return fmt.Sprintf("rugo_dot_call(%s, %q, %s)", nsName, dot.Field, argStr), nil
		}
		// Non-ident object: e.g. tasks[i].wait(n), q.push(val)
		obj, oerr := g.exprString(dot.Object)
		if oerr != nil {
			return "", oerr
		}
		return fmt.Sprintf("rugo_dot_call(%s, %q, %s)", obj, dot.Field, argStr), nil
	}

	// Check for built-in functions (globals)
	if ident, ok := e.Func.(*ast.IdentExpr); ok {
		switch ident.Name {
		case "puts":
			return fmt.Sprintf("rugo_puts(%s)", g.boxedArgs(args, e.Args)), nil
		case "print":
			return fmt.Sprintf("rugo_print(%s)", g.boxedArgs(args, e.Args)), nil
		case "__shell__":
			return fmt.Sprintf("rugo_shell(%s)", argStr), nil
		case "__capture__":
			return fmt.Sprintf("rugo_capture(%s)", argStr), nil
		case "__pipe_shell__":
			return fmt.Sprintf("rugo_pipe_shell(%s)", argStr), nil
		case "len":
			call := fmt.Sprintf("rugo_len(%s)", g.boxedArgs(args, e.Args))
			if g.exprType(e) == TypeInt {
				return call + ".(int)", nil
			}
			return call, nil
		case "append":
			return fmt.Sprintf("rugo_append(%s)", g.boxedArgs(args, e.Args)), nil
		case "raise":
			return fmt.Sprintf("rugo_raise(%s)", g.boxedArgs(args, e.Args)), nil
		case "exit":
			return fmt.Sprintf("rugo_exit(%s)", g.boxedArgs(args, e.Args)), nil
		case "type_of":
			if len(e.Args) != 1 {
				return "", fmt.Errorf("type_of expects 1 argument, got %d", len(e.Args))
			}
			return fmt.Sprintf("rugo_type_of(%s)", g.boxedArgs(args, e.Args)), nil
		default:
			// Sibling function call within a namespace: resolve unqualified
			// calls against the current function's namespace first.
			if g.currentFunc != nil && g.currentFunc.Namespace != "" {
				nsKey := g.currentFunc.Namespace + "." + ident.Name
				if expected, ok := g.funcDefs[nsKey]; ok {
					if len(e.Args) != expected {
						return "", argCountError(ident.Name, len(e.Args), expected)
					}
					typedArgs := g.typedCallArgs(nsKey, args, e.Args)
					return fmt.Sprintf("rugons_%s_%s(%s)", g.currentFunc.Namespace, ident.Name, typedArgs), nil
				}
			}
			// User-defined function — validate argument count
			if expected, ok := g.funcDefs[ident.Name]; ok {
				if len(e.Args) != expected {
					return "", argCountError(ident.Name, len(e.Args), expected)
				}
				// Generate typed call if function has typed params.
				typedArgs := g.typedCallArgs(ident.Name, args, e.Args)
				return fmt.Sprintf("rugofn_%s(%s)", ident.Name, typedArgs), nil
			}
			// Lambda variable call — dynamic dispatch via type assertion
			if g.isDeclared(ident.Name) {
				return fmt.Sprintf("%s.(func(...interface{}) interface{})(%s)", ident.Name, argStr), nil
			}
			// Generate typed call if function has typed params.
			typedArgs := g.typedCallArgs(ident.Name, args, e.Args)
			return fmt.Sprintf("rugofn_%s(%s)", ident.Name, typedArgs), nil
		}
	}

	// Dynamic call (shouldn't happen in v1 but handle gracefully)
	funcExpr, err := g.exprString(e.Func)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.(%s)(%s)", funcExpr, "func(...interface{}) interface{}", argStr), nil
}

func (g *codeGen) indexExpr(e *ast.IndexExpr) (string, error) {
	obj, err := g.exprString(e.Object)
	if err != nil {
		return "", err
	}
	idx, err := g.exprString(e.Index)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("rugo_index(%s, %s)", obj, idx), nil
}

func (g *codeGen) sliceExpr(e *ast.SliceExpr) (string, error) {
	obj, err := g.exprString(e.Object)
	if err != nil {
		return "", err
	}
	start, err := g.exprString(e.Start)
	if err != nil {
		return "", err
	}
	length, err := g.exprString(e.Length)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("rugo_slice(%s, %s, %s)", obj, start, length), nil
}

func (g *codeGen) arrayLiteral(e *ast.ArrayLiteral) (string, error) {
	elems := make([]string, len(e.Elements))
	for i, el := range e.Elements {
		s, err := g.exprString(el)
		if err != nil {
			return "", err
		}
		elems[i] = s
	}
	return fmt.Sprintf("interface{}([]interface{}{%s})", strings.Join(elems, ", ")), nil
}

func (g *codeGen) hashLiteral(e *ast.HashLiteral) (string, error) {
	pairs := make([]string, len(e.Pairs))
	for i, p := range e.Pairs {
		key, err := g.exprString(p.Key)
		if err != nil {
			return "", err
		}
		val, err := g.exprString(p.Value)
		if err != nil {
			return "", err
		}
		pairs[i] = fmt.Sprintf("%s: %s", key, val)
	}
	return fmt.Sprintf("interface{}(map[interface{}]interface{}{%s})", strings.Join(pairs, ", ")), nil
}

func (g *codeGen) tryExpr(e *ast.TryExpr) (string, error) {
	exprStr, err := g.exprString(e.Expr)
	if err != nil {
		return "", err
	}

	// Build the handler body as Go source in a temporary buffer.
	var handlerBuf strings.Builder
	handlerCode, cerr := g.captureOutput(func() error {
		g.pushScope()
		g.declareVar(e.ErrVar)

		for i, s := range e.Handler {
			isLast := i == len(e.Handler)-1
			if isLast {
				// Last statement: if it's a bare expression, assign to r (return value)
				if es, ok := s.(*ast.ExprStmt); ok {
					val, verr := g.exprString(es.Expression)
					if verr != nil {
						g.popScope()
						return verr
					}
					g.writef("r = %s\n", val)
					continue
				}
			}
			if werr := g.writeStmt(s); werr != nil {
				g.popScope()
				return werr
			}
		}

		g.popScope()
		return nil
	})
	if cerr != nil {
		return "", cerr
	}

	// Build the IIFE
	handlerBuf.WriteString("func() (r interface{}) {\n")
	handlerBuf.WriteString("\t\tdefer func() {\n")
	handlerBuf.WriteString("\t\t\tif e := recover(); e != nil {\n")
	handlerBuf.WriteString(fmt.Sprintf("\t\t\t\t%s := fmt.Sprint(e)\n", e.ErrVar))
	handlerBuf.WriteString(fmt.Sprintf("\t\t\t\t_ = %s\n", e.ErrVar))
	// Indent and write handler code
	for _, line := range strings.Split(handlerCode, "\n") {
		if line != "" {
			handlerBuf.WriteString("\t\t\t\t" + strings.TrimLeft(line, "\t") + "\n")
		}
	}
	handlerBuf.WriteString("\t\t\t}\n")
	handlerBuf.WriteString("\t\t}()\n")
	handlerBuf.WriteString(fmt.Sprintf("\t\treturn %s\n", exprStr))
	handlerBuf.WriteString("\t}()")

	return handlerBuf.String(), nil
}

func (g *codeGen) spawnExpr(e *ast.SpawnExpr) (string, error) {
	// Generate the body code in a temporary buffer.
	g.inSpawn++
	bodyCode, cerr := g.captureOutput(func() error {
		g.pushScope()
		for i, s := range e.Body {
			isLast := i == len(e.Body)-1
			if isLast {
				// Last statement: if it's a bare expression, assign to t.result
				if es, ok := s.(*ast.ExprStmt); ok {
					val, verr := g.exprString(es.Expression)
					if verr != nil {
						g.popScope()
						return verr
					}
					g.writef("t.result = %s\n", val)
					continue
				}
			}
			if werr := g.writeStmt(s); werr != nil {
				g.popScope()
				return werr
			}
		}
		g.popScope()
		return nil
	})
	g.inSpawn--
	if cerr != nil {
		return "", cerr
	}

	// Build the IIFE that creates a rugoTask and launches a goroutine
	var buf strings.Builder
	buf.WriteString("func() interface{} {\n")
	buf.WriteString("\t\tt := &rugoTask{done: make(chan struct{})}\n")
	buf.WriteString("\t\tgo func() {\n")
	buf.WriteString("\t\t\tdefer func() {\n")
	buf.WriteString("\t\t\t\tif e := recover(); e != nil {\n")
	buf.WriteString("\t\t\t\t\tt.err = fmt.Sprint(e)\n")
	buf.WriteString("\t\t\t\t}\n")
	buf.WriteString("\t\t\t\tclose(t.done)\n")
	buf.WriteString("\t\t\t}()\n")
	for _, line := range strings.Split(bodyCode, "\n") {
		if line != "" {
			buf.WriteString("\t\t\t" + strings.TrimLeft(line, "\t") + "\n")
		}
	}
	buf.WriteString("\t\t}()\n")
	buf.WriteString("\t\treturn interface{}(t)\n")
	buf.WriteString("\t}()")

	return buf.String(), nil
}

func (g *codeGen) fnExpr(e *ast.FnExpr) (string, error) {
	// Emit: func(_args ...interface{}) interface{} { p1 := _args[0]; ...; body; return nil }
	bodyCode, cerr := g.captureOutput(func() error {
		g.pushScope()
		for _, p := range e.Params {
			g.declareVar(p)
		}
		savedFunc := g.currentFunc
		savedInFunc := g.inFunc
		g.inFunc = true
		for i, s := range e.Body {
			isLast := i == len(e.Body)-1
			if isLast {
				// Last statement: if it's a bare expression, make it the return value
				if es, ok := s.(*ast.ExprStmt); ok {
					g.emitLineDirective(es.StmtLine())
					val, verr := g.exprString(es.Expression)
					if verr != nil {
						g.popScope()
						g.inFunc = savedInFunc
						g.currentFunc = savedFunc
						return verr
					}
					g.writef("return %s\n", val)
					continue
				}
			}
			if werr := g.writeStmt(s); werr != nil {
				g.popScope()
				g.inFunc = savedInFunc
				g.currentFunc = savedFunc
				return werr
			}
		}
		g.inFunc = savedInFunc
		g.currentFunc = savedFunc
		g.popScope()
		return nil
	})
	if cerr != nil {
		return "", cerr
	}

	var buf strings.Builder
	buf.WriteString("interface{}(func(_args ...interface{}) interface{} {\n")
	// Unpack parameters from variadic args
	for i, p := range e.Params {
		buf.WriteString(fmt.Sprintf("\t\tvar %s interface{}\n", p))
		buf.WriteString(fmt.Sprintf("\t\tif len(_args) > %d { %s = _args[%d] }\n", i, p, i))
		buf.WriteString(fmt.Sprintf("\t\t_ = %s\n", p))
	}
	for _, line := range strings.Split(bodyCode, "\n") {
		if line != "" {
			trimmed := strings.TrimLeft(line, "\t")
			// //line directives must start at column 1 (no indentation)
			if strings.HasPrefix(trimmed, "//line ") {
				buf.WriteString(trimmed + "\n")
			} else {
				buf.WriteString("\t\t" + trimmed + "\n")
			}
		}
	}
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t})")

	return buf.String(), nil
}

func (g *codeGen) parallelExpr(e *ast.ParallelExpr) (string, error) {
	// Each statement becomes a goroutine; collect results in an ordered array.
	n := len(e.Body)

	if n == 0 {
		return "interface{}([]interface{}{})", nil
	}

	// Generate each statement's Go expression
	type stmtCode struct {
		code   string
		isExpr bool
	}
	stmts := make([]stmtCode, n)
	for i, s := range e.Body {
		if es, ok := s.(*ast.ExprStmt); ok {
			code, err := g.exprString(es.Expression)
			if err != nil {
				return "", err
			}
			stmts[i] = stmtCode{code: code, isExpr: true}
		} else {
			// Non-expression statement: generate into a temp buffer
			code, err := g.captureOutput(func() error {
				g.pushScope()
				if err := g.writeStmt(s); err != nil {
					g.popScope()
					return err
				}
				g.popScope()
				return nil
			})
			if err != nil {
				return "", err
			}
			stmts[i] = stmtCode{code: code, isExpr: false}
		}
	}

	var buf strings.Builder
	buf.WriteString("func() interface{} {\n")
	buf.WriteString(fmt.Sprintf("\t\t_results := make([]interface{}, %d)\n", n))
	buf.WriteString("\t\tvar _wg sync.WaitGroup\n")
	buf.WriteString("\t\tvar _parErr string\n")
	buf.WriteString("\t\tvar _parOnce sync.Once\n")
	buf.WriteString(fmt.Sprintf("\t\t_wg.Add(%d)\n", n))

	for i, sc := range stmts {
		buf.WriteString("\t\tgo func() {\n")
		buf.WriteString("\t\t\tdefer _wg.Done()\n")
		buf.WriteString("\t\t\tdefer func() {\n")
		buf.WriteString("\t\t\t\tif e := recover(); e != nil {\n")
		buf.WriteString("\t\t\t\t\t_parOnce.Do(func() { _parErr = fmt.Sprint(e) })\n")
		buf.WriteString("\t\t\t\t}\n")
		buf.WriteString("\t\t\t}()\n")
		if sc.isExpr {
			buf.WriteString(fmt.Sprintf("\t\t\t_results[%d] = %s\n", i, sc.code))
		} else {
			for _, line := range strings.Split(sc.code, "\n") {
				if line != "" {
					buf.WriteString("\t\t\t" + strings.TrimLeft(line, "\t") + "\n")
				}
			}
		}
		buf.WriteString("\t\t}()\n")
	}

	buf.WriteString("\t\t_wg.Wait()\n")
	buf.WriteString("\t\tif _parErr != \"\" {\n")
	buf.WriteString("\t\t\tpanic(_parErr)\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t\tout := make([]interface{}, len(_results))\n")
	buf.WriteString("\t\tcopy(out, _results)\n")
	buf.WriteString("\t\treturn interface{}(out)\n")
	buf.WriteString("\t}()")

	return buf.String(), nil
}

// writePanicHandler emits the defer/recover block used in all main() functions.
func (g *codeGen) writePanicHandler() {
	g.writeln(`defer func() {`)
	g.indent++
	g.writeln(`if e := recover(); e != nil {`)
	g.indent++
	g.writeln(`if shellErr, ok := e.(rugoShellError); ok {`)
	g.indent++
	g.writeln(`os.Exit(shellErr.code)`)
	g.indent--
	g.writeln(`}`)
	g.writeln(`rugo_panic_handler(e)`)
	g.indent--
	g.writeln(`}`)
	g.indent--
	g.writeln(`}()`)
}

// captureOutput runs fn while writing to a temporary buffer,
// then restores the original buffer and returns the captured output.
func (g *codeGen) captureOutput(fn func() error) (string, error) {
	saved := g.sb
	g.sb = strings.Builder{}
	err := fn()
	result := g.sb.String()
	g.sb = saved
	return result, err
}

// collectIdents returns the set of identifier names referenced in statements.
func collectIdents(stmts []ast.Statement) map[string]bool {
	names := make(map[string]bool)
	for _, s := range stmts {
		collectIdentsFromStmt(s, names)
	}
	return names
}

func collectIdentsFromStmt(s ast.Statement, names map[string]bool) {
	switch st := s.(type) {
	case *ast.AssignStmt:
		collectIdentsFromExpr(st.Value, names)
	case *ast.IndexAssignStmt:
		collectIdentsFromExpr(st.Object, names)
		collectIdentsFromExpr(st.Index, names)
		collectIdentsFromExpr(st.Value, names)
	case *ast.DotAssignStmt:
		collectIdentsFromExpr(st.Object, names)
		collectIdentsFromExpr(st.Value, names)
	case *ast.ExprStmt:
		collectIdentsFromExpr(st.Expression, names)
	case *ast.IfStmt:
		collectIdentsFromExpr(st.Condition, names)
		for _, b := range st.Body {
			collectIdentsFromStmt(b, names)
		}
		for _, ei := range st.ElsifClauses {
			collectIdentsFromExpr(ei.Condition, names)
			for _, b := range ei.Body {
				collectIdentsFromStmt(b, names)
			}
		}
		for _, b := range st.ElseBody {
			collectIdentsFromStmt(b, names)
		}
	case *ast.WhileStmt:
		collectIdentsFromExpr(st.Condition, names)
		for _, b := range st.Body {
			collectIdentsFromStmt(b, names)
		}
	case *ast.ForStmt:
		collectIdentsFromExpr(st.Collection, names)
		for _, b := range st.Body {
			collectIdentsFromStmt(b, names)
		}
	case *ast.ReturnStmt:
		if st.Value != nil {
			collectIdentsFromExpr(st.Value, names)
		}
	}
}

func collectIdentsFromExpr(e ast.Expr, names map[string]bool) {
	switch ex := e.(type) {
	case *ast.IdentExpr:
		names[ex.Name] = true
	case *ast.BinaryExpr:
		collectIdentsFromExpr(ex.Left, names)
		collectIdentsFromExpr(ex.Right, names)
	case *ast.UnaryExpr:
		collectIdentsFromExpr(ex.Operand, names)
	case *ast.CallExpr:
		collectIdentsFromExpr(ex.Func, names)
		for _, a := range ex.Args {
			collectIdentsFromExpr(a, names)
		}
	case *ast.IndexExpr:
		collectIdentsFromExpr(ex.Object, names)
		collectIdentsFromExpr(ex.Index, names)
	case *ast.SliceExpr:
		collectIdentsFromExpr(ex.Object, names)
		collectIdentsFromExpr(ex.Start, names)
		collectIdentsFromExpr(ex.Length, names)
	case *ast.DotExpr:
		collectIdentsFromExpr(ex.Object, names)
	case *ast.ArrayLiteral:
		for _, el := range ex.Elements {
			collectIdentsFromExpr(el, names)
		}
	case *ast.HashLiteral:
		for _, p := range ex.Pairs {
			collectIdentsFromExpr(p.Key, names)
			collectIdentsFromExpr(p.Value, names)
		}
	case *ast.TryExpr:
		collectIdentsFromExpr(ex.Expr, names)
		for _, b := range ex.Handler {
			collectIdentsFromStmt(b, names)
		}
	case *ast.FnExpr:
		for _, b := range ex.Body {
			collectIdentsFromStmt(b, names)
		}
	case *ast.StringLiteral:
		if ast.HasInterpolation(ex.Value) {
			_, exprStrs := ast.ProcessInterpolation(ex.Value)
			for _, exprStr := range exprStrs {
				p := &parser.Parser{}
				flatAST, err := p.Parse("<ident-scan>", []byte(exprStr+"\n"))
				if err != nil {
					continue
				}
				prog, err := ast.Walk(p, flatAST)
				if err != nil {
					continue
				}
				for _, s := range prog.Statements {
					collectIdentsFromStmt(s, names)
				}
			}
		}
	}
}

// Scope management
func (g *codeGen) pushScope() {
	g.scopes = append(g.scopes, make(map[string]bool))
	g.constScopes = append(g.constScopes, make(map[string]int))
}

func (g *codeGen) popScope() {
	g.scopes = g.scopes[:len(g.scopes)-1]
	g.constScopes = g.constScopes[:len(g.constScopes)-1]
}

func (g *codeGen) declareVar(name string) {
	g.scopes[len(g.scopes)-1][name] = true
}

func (g *codeGen) isDeclared(name string) bool {
	for i := len(g.scopes) - 1; i >= 0; i-- {
		if g.scopes[i][name] {
			return true
		}
	}
	return false
}

func (g *codeGen) declareConst(name string, line int) {
	g.constScopes[len(g.constScopes)-1][name] = line
}

func (g *codeGen) constantLine(name string) (int, bool) {
	for i := len(g.constScopes) - 1; i >= 0; i-- {
		if line, ok := g.constScopes[i][name]; ok {
			return line, true
		}
	}
	return 0, false
}

// Output helpers
func (g *codeGen) writeln(s string) {
	g.writef("%s\n", s)
}

func (g *codeGen) writef(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	if strings.HasSuffix(strings.TrimRight(line, "\n"), "\n") || line == "\n" {
		g.sb.WriteString(line)
		return
	}
	indent := strings.Repeat("\t", g.indent)
	g.sb.WriteString(indent + line)
}

// importedModuleNames returns sorted module names from the imports map.
func importedModuleNames(imports map[string]bool) []string {
	var names []string
	for name := range imports {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// astUsesSpawn checks if any ast.SpawnExpr exists in the AST.
func astUsesSpawn(prog *ast.Program) bool {
	return WalkExprs(prog, func(e ast.Expr) bool {
		_, ok := e.(*ast.SpawnExpr)
		return ok
	})
}

// astUsesTaskMethods checks if any ast.DotExpr uses .value, .done, or .wait on a non-module target.
func astUsesTaskMethods(prog *ast.Program) bool {
	return WalkExprs(prog, func(e ast.Expr) bool {
		dot, ok := e.(*ast.DotExpr)
		if !ok || !taskMethodNames[dot.Field] {
			return false
		}
		if ident, ok := dot.Object.(*ast.IdentExpr); ok {
			if modules.IsModule(ident.Name) {
				return false
			}
		}
		return true
	})
}

var taskMethodNames = map[string]bool{"value": true, "done": true, "wait": true}

// astUsesParallel checks if any ast.ParallelExpr exists in the AST.
func astUsesParallel(prog *ast.Program) bool {
	return WalkExprs(prog, func(e ast.Expr) bool {
		_, ok := e.(*ast.ParallelExpr)
		return ok
	})
}

// sortedGoBridgeImports returns sorted package paths from goImports map.
func sortedGoBridgeImports(goImports map[string]string) []string {
	var pkgs []string
	for pkg := range goImports {
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)
	return pkgs
}

// generateGoBridgeCall generates a Go expression for a direct Go bridge call.
// rugoName is the user-visible name (e.g. "strconv.atoi") for error messages.
func (g *codeGen) generateGoBridgeCall(pkg string, sig *gobridge.GoFuncSig, argExprs []string, rugoName string) string {
	// Determine the Go package prefix to use in generated code
	pkgBase := gobridge.DefaultNS(pkg)
	if alias, ok := g.goImports[pkg]; ok && alias != "" {
		pkgBase = alias
	}

	// Custom codegen callback — bridge file owns its own logic
	if sig.Codegen != nil {
		return sig.Codegen(pkgBase, argExprs, rugoName)
	}

	// Build converted args
	var convertedArgs []string
	for i, arg := range argExprs {
		if i < len(sig.Params) {
			convertedArgs = append(convertedArgs, gobridge.TypeConvToGo(arg, sig.Params[i]))
		}
	}

	// For variadic functions, handle special conversion
	if sig.Variadic && len(sig.Params) == 1 && sig.Params[0] == gobridge.GoStringSlice {
		var strArgs []string
		for _, arg := range argExprs {
			strArgs = append(strArgs, gobridge.TypeConvToGo(arg, gobridge.GoString))
		}
		call := fmt.Sprintf("%s.%s(%s)", pkgBase, sig.GoName, strings.Join(strArgs, ", "))
		if len(sig.Returns) == 0 {
			return call
		}
		return gobridge.TypeWrapReturn(call, sig.Returns[0])
	}

	// Handle special method-chain calls (e.g. time.Now().Unix())
	// Note: base64/time method-chains now handled by Codegen callbacks.
	// This is a generic fallback for future method-chain packages.
	if strings.Contains(sig.GoName, ".") {
		call := fmt.Sprintf("%s.%s(%s)", pkgBase, sig.GoName, strings.Join(convertedArgs, ", "))
		if len(sig.Returns) == 0 {
			return call
		}
		if sig.Returns[0] == gobridge.GoInt {
			return fmt.Sprintf("interface{}(int(%s))", call)
		}
		return gobridge.TypeWrapReturn(call, sig.Returns[0])
	}

	// Error panic format with Rugo function name
	panicFmt := fmt.Sprintf(`panic(rugo_bridge_err("%s", _err))`, rugoName)

	call := fmt.Sprintf("%s.%s(%s)", pkgBase, sig.GoName, strings.Join(convertedArgs, ", "))

	// Handle return types
	if len(sig.Returns) == 0 {
		// Void Go functions need wrapping since Rugo assigns all expressions
		return fmt.Sprintf("func() interface{} { %s; return nil }()", call)
	}

	if len(sig.Returns) == 1 {
		// Single error return: panic on error, return nil
		if sig.Returns[0] == gobridge.GoError {
			return fmt.Sprintf("func() interface{} { if _err := %s; _err != nil { %s }; return nil }()", call, panicFmt)
		}
		return gobridge.TypeWrapReturn(call, sig.Returns[0])
	}

	// (T, error): panic on error
	if len(sig.Returns) == 2 && sig.Returns[1] == gobridge.GoError {
		return fmt.Sprintf("func() interface{} { _v, _err := %s; if _err != nil { %s }; return %s }()",
			call, panicFmt, gobridge.TypeWrapReturn("_v", sig.Returns[0]))
	}

	// (T, bool): return nil if false
	if len(sig.Returns) == 2 && sig.Returns[1] == gobridge.GoBool {
		return fmt.Sprintf("func() interface{} { _v, _ok := %s; if !_ok { return nil }; return %s }()",
			call, gobridge.TypeWrapReturn("_v", sig.Returns[0]))
	}

	// Default: just wrap first return
	return gobridge.TypeWrapReturn(call, sig.Returns[0])
}

// writeGoBridgeRuntime emits helper functions needed by Go bridge calls.
// Helpers are declared by bridge files via RuntimeHelpers on GoFuncSig,
// deduplicated by key, and emitted once.
func (g *codeGen) writeGoBridgeRuntime() {
	g.sb.WriteString("\n// --- Go Bridge Helpers ---\n\n")

	emitted := map[string]bool{}
	for pkg := range g.goImports {
		for _, h := range gobridge.AllRuntimeHelpers(pkg) {
			if !emitted[h.Key] {
				emitted[h.Key] = true
				g.sb.WriteString(h.Code)
			}
		}
	}
}

// argCountError produces a human-friendly argument count mismatch error.
func argCountError(name string, got, expected int) error {
	argWord := "arguments"
	if expected == 1 {
		argWord = "argument"
	}
	gotDesc := fmt.Sprintf("%d were", got)
	if got == 0 {
		gotDesc = "none were"
	} else if got == 1 {
		gotDesc = "1 was"
	}
	return fmt.Errorf("%s() takes %d %s but %s given", name, expected, argWord, gotDesc)
}

// --- Type inference helpers for codegen ---

// exprType returns the inferred type of an expression.
func (g *codeGen) exprType(e ast.Expr) RugoType {
	if g.typeInfo == nil {
		return TypeDynamic
	}
	return g.typeInfo.ExprType(e)
}

// exprIsTyped returns true if the expression has a resolved primitive type.
func (g *codeGen) exprIsTyped(e ast.Expr) bool {
	return g.exprType(e).IsTyped()
}

// currentFuncTypeInfo returns the type info for the function being generated.
func (g *codeGen) currentFuncTypeInfo() *FuncTypeInfo {
	if g.typeInfo == nil || g.currentFunc == nil {
		return nil
	}
	return g.typeInfo.FuncTypes[funcKey(g.currentFunc)]
}

// varType returns the inferred type of a variable in the current scope.
func (g *codeGen) varType(name string) RugoType {
	if g.typeInfo == nil {
		return TypeDynamic
	}
	scope := g.varTypeScope
	if scope == "" && g.currentFunc != nil {
		scope = funcKey(g.currentFunc)
	}
	return g.typeInfo.VarType(scope, name)
}

// condExpr wraps a condition string for use in if/while.
// If the condition is typed bool, use it directly; otherwise wrap with rugo_to_bool.
func (g *codeGen) condExpr(condStr string, condExpr ast.Expr) string {
	if g.exprType(condExpr) == TypeBool {
		return condStr
	}
	return fmt.Sprintf("rugo_to_bool(%s)", g.boxed(condStr, g.exprType(condExpr)))
}

// boxed wraps a typed value in interface{} if it's a resolved primitive type.
// This is needed when passing typed values to runtime helpers that expect interface{}.
func (g *codeGen) boxed(s string, t RugoType) string {
	if t.IsTyped() {
		return fmt.Sprintf("interface{}(%s)", s)
	}
	return s
}

// goTyped returns true if the expression will produce a Go-typed value at runtime,
// not an interface{}. Variables stored as interface{} (varType is dynamic) return false
// even if the expression type is inferred as typed, because Go's type system won't
// allow using raw operators on interface{} values.
func (g *codeGen) goTyped(e ast.Expr) bool {
	if ident, ok := e.(*ast.IdentExpr); ok {
		return g.varType(ident.Name).IsTyped()
	}
	return g.exprType(e).IsTyped()
}

// ensureFloat wraps int expressions with float64() for mixed numeric ops.
func (g *codeGen) ensureFloat(s string, t RugoType) string {
	if t == TypeInt {
		return fmt.Sprintf("float64(%s)", s)
	}
	return s
}

// boxedArgs returns comma-joined args, boxing typed values for runtime helpers.
func (g *codeGen) boxedArgs(args []string, exprs []ast.Expr) string {
	result := make([]string, len(args))
	for i, a := range args {
		result[i] = g.boxed(a, g.exprType(exprs[i]))
	}
	return strings.Join(result, ", ")
}

// typedCallArgs generates the argument list for a user-defined function call,
// converting typed args to match the function's typed param signature.
func (g *codeGen) typedCallArgs(funcName string, args []string, argExprs []ast.Expr) string {
	if g.typeInfo == nil {
		return strings.Join(args, ", ")
	}
	fti, ok := g.typeInfo.FuncTypes[funcName]
	if !ok {
		return strings.Join(args, ", ")
	}

	result := make([]string, len(args))
	for i, a := range args {
		argType := g.exprType(argExprs[i])
		if i < len(fti.ParamTypes) && fti.ParamTypes[i].IsTyped() {
			// Target param is typed — ensure arg matches.
			if argType == fti.ParamTypes[i] {
				result[i] = a // Already the right type.
			} else if argType.IsTyped() && argType.IsNumeric() && fti.ParamTypes[i].IsNumeric() {
				// Numeric promotion.
				if fti.ParamTypes[i] == TypeFloat && argType == TypeInt {
					result[i] = fmt.Sprintf("float64(%s)", a)
				} else if fti.ParamTypes[i] == TypeInt && argType == TypeFloat {
					result[i] = fmt.Sprintf("int(%s)", a)
				} else {
					result[i] = a
				}
			} else if argType.IsTyped() {
				// Type mismatch — shouldn't happen with correct inference,
				// but be safe.
				result[i] = a
			} else {
				// Arg is interface{} but param is typed — need type assertion.
				result[i] = fmt.Sprintf("%s.(%s)", a, fti.ParamTypes[i].GoType())
			}
		} else {
			// Target param is interface{} — box typed args.
			result[i] = g.boxed(a, argType)
		}
	}
	return strings.Join(result, ", ")
}
