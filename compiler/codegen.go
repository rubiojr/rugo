package compiler

import (
	_ "embed"
	"fmt"
	"github.com/rubiojr/rugo/ast"
	"sort"
	"strings"

	"github.com/rubiojr/rugo/gobridge"
	"github.com/rubiojr/rugo/modules"
)

//go:embed templates/runtime_core_pre.go.tmpl
var runtimeCorePre string

//go:embed templates/runtime_core_post.go.tmpl
var runtimeCorePost string

//go:embed templates/runtime_spawn.go.tmpl
var runtimeSpawn string

// funcArity stores the arity range for a user-defined function.
type funcArity struct {
	Min         int  // number of required params (no default)
	Max         int  // total number of params (required + optional)
	HasDefaults bool // true if any param has a default value
}

// codeGen generates Go source code from a typed AST.
type codeGen struct {
	sb              strings.Builder
	indent          int
	declared        map[string]bool // track declared variables per scope
	scopes          []map[string]bool
	constScopes     []map[string]int // track constant bindings: name → line of first assignment
	inFunc          bool
	imports         map[string]bool      // Rugo stdlib modules imported (via use)
	goImports       map[string]string    // Go bridge packages: path → alias
	namespaces      map[string]bool      // known require namespaces
	nsVarNames      map[string]bool      // namespaced var names: "ns.name" → true
	sourceFile      string               // original source filename for //line directives
	hasSpawn        bool                 // whether spawn is used
	hasParallel     bool                 // whether parallel is used
	hasBench        bool                 // whether bench blocks are present
	usesTaskMethods bool                 // whether .value/.done/.wait appear
	funcDefs        map[string]funcArity // user function name → arity info
	handlerVars     map[string]bool      // top-level vars promoted to package-level for handler access
	testMode        bool                 // include rats blocks in output
	typeInfo        *TypeInfo            // inferred type information (nil disables typed codegen)
	currentFunc     *ast.FuncDef         // current function being generated (for type lookups)
	varTypeScope    string               // override scope key for varType lookups (test/bench blocks)
	inSpawn         int                  // nesting depth of spawn blocks (>0 means inside spawn)
	inTryHandler    int                  // nesting depth of try/or handler defers (>0 means inside handler)
	lambdaDepth     int                  // nesting depth of lambda bodies (>0 means inside fn)
	lambdaScopeBase []int                // scope index at each lambda entry (stack)
	lambdaOuterFunc []*ast.FuncDef       // enclosing function at each lambda entry (stack)
	sandbox         *SandboxConfig       // Landlock sandbox config (nil = no sandbox)
}

// generate produces Go source code from a ast.Program AST.
func generate(prog *ast.Program, sourceFile string, testMode bool, sandbox *SandboxConfig) (string, error) {
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
		funcDefs:    make(map[string]funcArity),
		testMode:    testMode,
		typeInfo:    ti,
		sandbox:     sandbox,
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
		case *ast.SandboxStmt:
			// Placement and duplicate checks are done in validateSandboxPlacement.
			// If sandbox is already set (CLI override), skip the script directive.
			if g.sandbox != nil {
				continue
			}
			g.sandbox = &SandboxConfig{
				RO: st.RO, RW: st.RW, ROX: st.ROX, RWX: st.RWX,
				Connect: st.Connect, Bind: st.Bind, Env: st.Env, EnvSet: st.EnvSet,
			}
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
		g.funcDefs[key] = funcArity{Min: ast.MinArity(f.Params), Max: len(f.Params), HasDefaults: ast.HasDefaults(f.Params)}
	}

	// Register namespaces from require'd constants
	for _, nv := range nsVars {
		g.namespaces[nv.Namespace] = true
		g.nsVarNames[nv.Namespace+"."+nv.Target] = true
	}

	// Identify top-level variables referenced by user-defined functions.
	// All def functions are emitted as top-level Go functions but top-level
	// variables live inside main(). Promote referenced vars to package-level.
	// If a function also assigns to the same name, the var is still promoted
	// (so reads before the assignment see the top-level value) and the
	// assignment inside the function creates a local shadow.
	{
		// Collect top-level assignment targets
		topVarNames := make(map[string]bool)
		for _, s := range topStmts {
			if a, ok := s.(*ast.AssignStmt); ok && a.Namespace == "" {
				topVarNames[a.Target] = true
			}
		}
		// Collect idents referenced by any non-namespaced function.
		// Always promote if the top-level name is referenced, even if
		// the function also assigns to it (the assignment will create
		// a local shadow at codegen time).
		for _, f := range funcs {
			if f.Namespace != "" {
				continue
			}
			refs := collectIdents(f.Body)
			for name := range refs {
				if topVarNames[name] {
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
	g.writeln(`"math"`)
	g.writeln(`"os"`)
	g.writeln(`"os/exec"`)
	g.writeln(`"reflect"`)
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
		"fmt": true, "math": true, "os": true, "os/exec": true,
		"reflect": true, "runtime/debug": true, "strings": true, "sort": true,
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
		bp := gobridge.GetPackage(pkg)
		if bp != nil && bp.NoGoImport {
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
		// Emit extra imports for external type wrappers (e.g. external dependencies).
		if bp != nil {
			for _, extra := range bp.ExtraImports {
				if !emittedImports[extra] {
					g.writef("\"%s\"\n", extra)
					emittedImports[extra] = true
				}
			}
		}
	}
	// Sandbox imports (go-landlock)
	if g.sandbox != nil {
		if !emittedImports["runtime"] {
			g.writeln(`"runtime"`)
			emittedImports["runtime"] = true
		}
		g.writeln(`"github.com/landlock-lsm/go-landlock/landlock"`)
		g.writeln(`llsyscall "github.com/landlock-lsm/go-landlock/landlock/syscall"`)
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
	if g.sandbox != nil {
		g.writeln("var _ = landlock.V5")
		g.writeln("var _ = llsyscall.AccessFSExecute")
		g.writeln("var _ = runtime.GOOS")
	}
	// Silence unused import warnings for Go module requires.
	// Stdlib bridge packages are always used by the runtime, but external
	// Go module packages may be imported without immediate function calls.
	for _, pkg := range sortedGoBridgeImports(g.goImports) {
		bp := gobridge.GetPackage(pkg)
		if bp == nil || !bp.External {
			continue
		}
		for _, sig := range bp.Funcs {
			// Skip struct constructors (their GoName is a type, not a function).
			if sig.Codegen != nil {
				continue
			}
			alias := g.goImports[pkg]
			pkgBase := alias
			if pkgBase == "" {
				pkgBase = gobridge.DefaultNS(pkg)
			}
			g.writef("var _ = %s.%s\n", pkgBase, sig.GoName)
			break
		}
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
	if g.sandbox != nil {
		g.writeSandboxApply()
	}
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
		// Handler vars are promoted to package-level as interface{},
		// so they are never Go-typed even if type inference says they are.
		if g.handlerVars[ident.Name] && !g.inFunc {
			return false
		}
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
