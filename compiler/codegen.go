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
	caseCounter     int                  // counter for unique case temp variable names
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
			if st.Namespace == "" && builtinFuncs[st.Name] {
				return "", fmt.Errorf("%s:%d: cannot redefine builtin function %q", g.sourceFile, st.SourceLine, st.Name)
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
		"var _ = strconv.Itoa",
		"var _ = utf8.RuneCountInString",
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
	seenExtraSuppressors := map[string]bool{}
	for _, pkg := range sortedGoBridgeImports(g.goImports) {
		bp := gobridge.GetPackage(pkg)
		if bp == nil {
			continue
		}
		for _, extra := range bp.ExtraImports {
			target := bridgeExtraImportTypeSuppressor(bp, extra)
			if target == "" {
				continue
			}
			line := "var _ " + target
			if seenExtraSuppressors[line] {
				continue
			}
			seenExtraSuppressors[line] = true
			suppressors = append(suppressors, line)
		}
	}
	file.Decls = append(file.Decls, GoRawDecl{Code: strings.Join(suppressors, "\n") + "\n"})
	file.Decls = append(file.Decls, GoBlankLine{})

	// Runtime helpers
	file.Decls = append(file.Decls, GoRawDecl{Code: g.buildRuntimeCode()})

	// Package-level variables from require'd files
	for _, nv := range nsVars {
		expr, err := g.buildExpr(nv.Value)
		if err != nil {
			return "", err
		}
		file.Decls = append(file.Decls, GoVarDecl{
			Name:  fmt.Sprintf("rugons_%s_%s", nv.Namespace, nv.Target),
			Type:  "interface{}",
			Value: expr,
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

	// Dispatch maps
	dispatchHandlers := collectDispatchHandlers(prog.Statements, g.imports)
	file.Decls = append(file.Decls, g.buildDispatchMaps(funcs, dispatchHandlers)...)

	// Test harness
	if len(tests) > 0 {
		harnessDecls, herr := g.buildTestHarness(tests, topStmts, setupFunc, teardownFunc, setupFileFunc, teardownFileFunc)
		if herr != nil {
			return "", herr
		}
		file.Decls = append(file.Decls, harnessDecls...)
		return PrintGoFile(file), nil
	}

	// Bench harness
	if len(benches) > 0 {
		harnessDecls, herr := g.buildBenchHarness(benches, topStmts)
		if herr != nil {
			return "", herr
		}
		file.Decls = append(file.Decls, harnessDecls...)
		return PrintGoFile(file), nil
	}

	// Main function body
	var mainBody []GoStmt
	mainBody = append(mainBody, g.buildPanicHandler())
	if g.sandbox != nil {
		mainBody = append(mainBody, g.buildSandboxApply()...)
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

func bridgeExtraImportTypeSuppressor(bp *gobridge.Package, extraPath string) string {
	if bp == nil {
		return ""
	}
	qualifier := gobridge.DefaultNS(extraPath)
	if qualifier == "" {
		return ""
	}
	for _, sig := range bp.Funcs {
		if target := bridgeCastTargetForQualifier(sig.TypeCasts, qualifier); target != "" {
			return target
		}
		for _, ft := range sig.FuncTypes {
			if target := bridgeFuncTypeCastTarget(ft, qualifier); target != "" {
				return target
			}
		}
	}
	return ""
}

func bridgeCastTargetForQualifier(casts map[int]string, qualifier string) string {
	if len(casts) == 0 || qualifier == "" {
		return ""
	}
	keys := make([]int, 0, len(casts))
	for k := range casts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		target := gobridge.TypeCastTarget(casts[k])
		bare := strings.TrimLeft(target, "*")
		if strings.HasPrefix(bare, qualifier+".") {
			return target
		}
	}
	return ""
}

func bridgeFuncTypeCastTarget(ft *gobridge.GoFuncType, qualifier string) string {
	if ft == nil {
		return ""
	}
	if target := bridgeCastTargetForQualifier(ft.TypeCasts, qualifier); target != "" {
		return target
	}
	for _, nested := range ft.FuncTypes {
		if target := bridgeFuncTypeCastTarget(nested, qualifier); target != "" {
			return target
		}
	}
	return ""
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
	switch ex := e.(type) {
	case *ast.IdentExpr:
		// Handler vars are promoted to package-level as interface{},
		// so they are never Go-typed even if type inference says they are.
		if g.handlerVars[ex.Name] && !g.inFunc {
			return false
		}
		return g.varType(ex.Name).IsTyped()
	case *ast.UnaryExpr:
		operandType := g.exprType(ex.Operand)
		switch ex.Op {
		case "!":
			return operandType == TypeBool && g.goTyped(ex.Operand)
		case "-":
			return operandType.IsNumeric() && g.goTyped(ex.Operand)
		default:
			return false
		}
	case *ast.BinaryExpr:
		leftType := g.exprType(ex.Left)
		rightType := g.exprType(ex.Right)
		leftGoTyped := g.goTyped(ex.Left)
		rightGoTyped := g.goTyped(ex.Right)
		bothGoTyped := leftGoTyped && rightGoTyped
		bothInts := leftType == TypeInt && rightType == TypeInt && bothGoTyped
		bothNumeric := leftType.IsNumeric() && rightType.IsNumeric() && leftType.IsTyped() && rightType.IsTyped() && bothGoTyped
		bothStrings := leftType == TypeString && rightType == TypeString && bothGoTyped
		sameTyped := leftType == rightType && leftType.IsTyped()

		switch ex.Op {
		case "+":
			return bothInts || bothStrings || bothNumeric
		case "-", "*", "/":
			return bothInts || bothNumeric
		case "%":
			return bothInts
		case "==", "!=":
			return sameTyped && bothGoTyped
		case "<", ">", "<=", ">=":
			return sameTyped && bothGoTyped && (leftType.IsNumeric() || leftType == TypeString)
		case "&&", "||":
			return leftType == TypeBool && rightType == TypeBool && bothGoTyped
		default:
			return false
		}
	default:
		return g.exprType(e).IsTyped()
	}
}

// ensureFloat wraps int expressions with float64() for mixed numeric ops.
func (g *codeGen) ensureFloat(s string, t RugoType) string {
	if t == TypeInt {
		return fmt.Sprintf("float64(%s)", s)
	}
	return s
}

// boxedExprs wraps typed GoExpr args in GoCastExpr for runtime helpers.
func (g *codeGen) boxedExprs(args []GoExpr, exprs []ast.Expr) []GoExpr {
	result := make([]GoExpr, len(args))
	for i, a := range args {
		if g.exprType(exprs[i]).IsTyped() {
			result[i] = GoCastExpr{Type: "interface{}", Value: a}
		} else {
			result[i] = a
		}
	}
	return result
}

// typedCallExprs generates GoExpr arguments for a user-defined function call,
// converting typed args to match the function's typed param signature.
func (g *codeGen) typedCallExprs(funcName string, args []GoExpr, argExprs []ast.Expr) []GoExpr {
	if g.typeInfo == nil {
		return args
	}
	fti, ok := g.typeInfo.FuncTypes[funcName]
	if !ok {
		return args
	}

	result := make([]GoExpr, len(args))
	for i, a := range args {
		argType := g.exprType(argExprs[i])
		if i < len(fti.ParamTypes) && fti.ParamTypes[i].IsTyped() {
			paramType := fti.ParamTypes[i]
			if !g.goTyped(argExprs[i]) {
				switch paramType {
				case TypeInt:
					result[i] = GoCallExpr{Func: "rugo_to_int", Args: []GoExpr{a}}
				case TypeFloat:
					result[i] = GoCallExpr{Func: "rugo_to_float", Args: []GoExpr{a}}
				case TypeString:
					result[i] = GoCallExpr{Func: "rugo_to_string", Args: []GoExpr{a}}
				case TypeBool:
					result[i] = GoCallExpr{Func: "rugo_to_bool", Args: []GoExpr{a}}
				default:
					result[i] = GoTypeAssert{Value: a, Type: paramType.GoType()}
				}
				continue
			}
			if argType == paramType {
				result[i] = a
			} else if argType.IsTyped() && argType.IsNumeric() && paramType.IsNumeric() {
				if paramType == TypeFloat && argType == TypeInt {
					result[i] = GoCastExpr{Type: "float64", Value: a}
				} else if paramType == TypeInt && argType == TypeFloat {
					result[i] = GoCastExpr{Type: "int", Value: a}
				} else {
					result[i] = a
				}
			} else if argType.IsTyped() {
				result[i] = a
			} else {
				result[i] = GoTypeAssert{Value: a, Type: paramType.GoType()}
			}
		} else {
			if argType.IsTyped() {
				result[i] = GoCastExpr{Type: "interface{}", Value: a}
			} else {
				result[i] = a
			}
		}
	}
	return result
}
