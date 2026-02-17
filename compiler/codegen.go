package compiler

import (
	_ "embed"
	"fmt"
	"github.com/rubiojr/rugo/ast"
	"sort"
	"strings"

	"github.com/rubiojr/rugo/gobridge"
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
	w               goWriter
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
	lambdaDepth     int                  // nesting depth of lambda bodies (>0 means inside fn)
	lambdaScopeBase []int                // scope index at each lambda entry (stack)
	lambdaOuterFunc []*ast.FuncDef       // enclosing function at each lambda entry (stack)
	sandbox         *SandboxConfig       // Landlock sandbox config (nil = no sandbox)
}

// generate produces Go source code from a ast.Program AST.
func generate(prog *ast.Program, sourceFile string, testMode bool, sandbox *SandboxConfig) (string, error) {
	// Run AST transform chain before type inference and codegen.
	prog = ast.Chain(
		ast.ConcurrencyLowering(),
		ast.ImplicitReturnLowering(),
	).Transform(prog)

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
	g.w.source = sourceFile
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

	// --- Build GoFile ---
	file := &GoFile{Package: "main"}

	// Imports
	file.Imports = g.buildImports(needsSyncImport, needsTimeImport)

	// Unused import suppressors
	var suppressors []string
	suppressors = append(suppressors,
		"var _ = fmt.Sprintf",
		"var _ = os.Exit",
		"var _ = exec.Command",
		"var _ = strings.NewReader",
		"var _ = sort.Slice",
		"var _ = debug.Stack",
	)
	if needsSyncImport {
		suppressors = append(suppressors, "var _ sync.Once")
	}
	if needsTimeImport {
		suppressors = append(suppressors, "var _ = time.Now")
	}
	if g.sandbox != nil {
		suppressors = append(suppressors, "var _ = landlock.V5", "var _ = llsyscall.AccessFSExecute", "var _ = runtime.GOOS")
	}
	for _, pkg := range sortedGoBridgeImports(g.goImports) {
		bp := gobridge.GetPackage(pkg)
		if bp == nil || !bp.External {
			continue
		}
		for _, sig := range bp.Funcs {
			if sig.Codegen != nil {
				continue
			}
			alias := g.goImports[pkg]
			pkgBase := alias
			if pkgBase == "" {
				pkgBase = gobridge.DefaultNS(pkg)
			}
			suppressors = append(suppressors, fmt.Sprintf("var _ = %s.%s", pkgBase, sig.GoName))
			break
		}
	}
	file.Decls = append(file.Decls, GoRawDecl{Code: strings.Join(suppressors, "\n") + "\n"})
	file.Decls = append(file.Decls, GoBlankLine{})

	// Runtime helpers (capture via old writer)
	runtimeCode, _ := g.w.Capture(func() error {
		g.writeRuntime()
		return nil
	})
	file.Decls = append(file.Decls, GoRawDecl{Code: runtimeCode})

	// Package-level variables from require'd files
	for _, nv := range nsVars {
		expr, err := g.exprString(nv.Value)
		if err != nil {
			return "", err
		}
		file.Decls = append(file.Decls, GoVarDecl{
			Name:  fmt.Sprintf("rugons_%s_%s", nv.Namespace, nv.Target),
			Type:  "interface{}",
			Value: GoRawExpr{Code: expr},
		})
	}
	if len(nsVars) > 0 {
		file.Decls = append(file.Decls, GoBlankLine{})
	}

	// Package-level variables for user-defined function access
	if len(g.handlerVars) > 0 {
		names := make([]string, 0, len(g.handlerVars))
		for name := range g.handlerVars {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			file.Decls = append(file.Decls, GoVarDecl{Name: name, Type: "interface{}"})
		}
		file.Decls = append(file.Decls, GoBlankLine{})
	}

	// User-defined functions
	for _, f := range funcs {
		decl, err := g.buildFunc(f)
		if err != nil {
			return "", err
		}
		file.Decls = append(file.Decls, decl)
	}

	// Dispatch maps (capture via old writer)
	dispatchHandlers := collectDispatchHandlers(prog.Statements, g.imports)
	dispatchCode, _ := g.w.Capture(func() error {
		g.writeDispatchMaps(funcs, dispatchHandlers)
		return nil
	})
	if dispatchCode != "" {
		file.Decls = append(file.Decls, GoRawDecl{Code: dispatchCode})
	}

	// Test harness (returns entire file via old writer — capture and append)
	if len(tests) > 0 {
		harnessCode, herr := g.w.Capture(func() error {
			_, err := g.generateTestHarness(tests, topStmts, setupFunc, teardownFunc, setupFileFunc, teardownFileFunc)
			return err
		})
		if herr != nil {
			return "", herr
		}
		file.Decls = append(file.Decls, GoRawDecl{Code: harnessCode})
		return PrintGoFile(file), nil
	}

	// Bench harness
	if len(benches) > 0 {
		harnessCode, herr := g.w.Capture(func() error {
			_, err := g.generateBenchHarness(benches, topStmts)
			return err
		})
		if herr != nil {
			return "", herr
		}
		file.Decls = append(file.Decls, GoRawDecl{Code: harnessCode})
		return PrintGoFile(file), nil
	}

	// Main function body
	var mainBody []GoStmt
	mainBody = append(mainBody, g.buildPanicHandler())
	if g.sandbox != nil {
		sandboxCode, _ := g.w.Capture(func() error {
			g.writeSandboxApply()
			return nil
		})
		mainBody = append(mainBody, GoRawStmt{Code: sandboxCode})
	}
	g.pushScope()
	mainStmts, merr := g.buildStmts(topStmts)
	if merr != nil {
		return "", merr
	}
	mainBody = append(mainBody, mainStmts...)
	g.popScope()

	file.Init = mainBody
	return PrintGoFile(file), nil
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
