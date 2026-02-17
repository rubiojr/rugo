package compiler

import (
	"fmt"
	"github.com/rubiojr/rugo/ast"
	"strings"

	"github.com/rubiojr/rugo/modules"
)

func (g *codeGen) generateTestHarness(tests []*ast.TestDef, topStmts []ast.Statement, setup, teardown, setupFile, teardownFile *ast.FuncDef) (string, error) {
	// Emit each test as a function
	for i, t := range tests {
		funcName := fmt.Sprintf("rugo_test_%d", i)
		g.writef("func %s() (passed bool, skipped bool, skipReason string, failReason string) {\n", funcName)
		g.w.Indent()
		g.writeln("defer func() {")
		g.w.Indent()
		g.writeln("if r := recover(); r != nil {")
		g.w.Indent()
		g.writeln(`if reason, ok := r.(rugoTestSkip); ok {`)
		g.w.Indent()
		g.writeln("skipped = true")
		g.writeln("skipReason = string(reason)")
		g.writeln("return")
		g.w.Dedent()
		g.writeln("}")
		g.writeln(`failColor := "\033[31m"`)
		g.writeln(`failReset := "\033[0m"`)
		g.writeln(`if os.Getenv("NO_COLOR") != "" {`)
		g.w.Indent()
		g.writeln(`failColor = ""`)
		g.writeln(`failReset = ""`)
		g.w.Dedent()
		g.writeln(`}`)
		g.writeln(`failReason = fmt.Sprintf("%v", r)`)
		g.writeln(`fmt.Fprintf(os.Stderr, "  %sFAIL%s: %v\n", failColor, failReset, r)`)
		g.writeln("passed = false")
		g.w.Dedent()
		g.writeln("}")
		g.w.Dedent()
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
		g.w.Dedent()
		g.writeln("}")
		g.writeln("")
	}

	// Main function: delegate to runtime test runner
	g.writeln("func main() {")
	g.w.Indent()
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
	g.w.Indent()
	for i, t := range tests {
		escapedName := goEscapeString(t.Name)
		g.writef("{Name: \"%s\", Func: rugo_test_%d},\n", escapedName, i)
	}
	g.w.Dedent()

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
	g.w.Dedent()
	g.writeln("}")

	return g.w.String(), nil
}

func (g *codeGen) generateBenchHarness(benches []*ast.BenchDef, topStmts []ast.Statement) (string, error) {
	// Emit each benchmark as a function
	for i, b := range benches {
		funcName := fmt.Sprintf("rugo_bench_%d", i)
		g.writef("func %s() {\n", funcName)
		g.w.Indent()
		g.pushScope()
		g.varTypeScope = fmt.Sprintf("__bench_%p", b)
		for _, s := range b.Body {
			if err := g.writeStmt(s); err != nil {
				return "", err
			}
		}
		g.varTypeScope = ""
		g.popScope()
		g.w.Dedent()
		g.writeln("}")
		g.writeln("")
	}

	// Main function: run benchmarks via runtime runner
	g.writeln("func main() {")
	g.w.Indent()
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
	g.w.Indent()
	for i, b := range benches {
		escapedName := goEscapeString(b.Name)
		g.writef("{Name: \"%s\", Func: rugo_bench_%d},\n", escapedName, i)
	}
	g.w.Dedent()
	g.writeln("})")

	g.popScope()
	g.w.Dedent()
	g.writeln("}")

	return g.w.String(), nil
}

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
		g.w.Indent()
		for _, f := range funcs {
			if len(f.Params) != 1 {
				continue
			}
			if m.DispatchMainOnly && f.Namespace != "" {
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
		g.w.Dedent()
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
		g.w.source = f.SourceFile
		defer func() { g.sourceFile = saved; g.w.source = saved }()
	}

	hasDefaults := ast.HasDefaults(f.Params)

	// Check if this function has typed inference info.
	fti := g.funcTypeInfo(f)

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

	if hasDefaults {
		// Functions with default params use variadic signature
		g.writef("func %s(_args ...interface{}) %s {\n", goName, retType)
	} else {
		// Functions without defaults keep typed params
		params := make([]string, len(f.Params))
		for i, p := range f.Params {
			if fti != nil && fti.ParamTypes[i].IsTyped() {
				params[i] = p.Name + " " + fti.ParamTypes[i].GoType()
			} else {
				params[i] = p.Name + " interface{}"
			}
		}
		g.writef("func %s(%s) %s {\n", goName, strings.Join(params, ", "), retType)
	}
	g.w.Indent()
	g.pushScope()
	// Recursion depth guard
	g.writef("rugo_check_depth(%q)\n", f.Name)
	g.writef("defer func() { rugo_call_depth-- }()\n")

	if hasDefaults {
		// Emit arity range check
		minArity := ast.MinArity(f.Params)
		maxArity := len(f.Params)
		if minArity == 1 {
			g.writef("if len(_args) < %d { panic(fmt.Sprintf(\"%s() takes %d to %d arguments but %%d were given\", len(_args))) }\n", minArity, f.Name, minArity, maxArity)
		} else {
			g.writef("if len(_args) < %d { panic(fmt.Sprintf(\"%s() takes %d to %d arguments but %%d were given\", len(_args))) }\n", minArity, f.Name, minArity, maxArity)
		}
		g.writef("if len(_args) > %d { panic(fmt.Sprintf(\"%s() takes %d to %d arguments but %%d were given\", len(_args))) }\n", maxArity, f.Name, minArity, maxArity)

		// Unpack required params and declare optional ones with defaults
		for i, p := range f.Params {
			g.declareVar(p.Name)
			if p.Default == nil {
				// Required param — unpack from _args
				g.writef("var %s interface{} = _args[%d]\n", p.Name, i)
			} else {
				// Optional param — use default if not provided
				defaultExpr, err := g.exprString(p.Default)
				if err != nil {
					return err
				}
				g.writef("var %s interface{}\n", p.Name)
				g.writef("if len(_args) > %d { %s = _args[%d] } else { %s = %s }\n", i, p.Name, i, p.Name, defaultExpr)
			}
			g.writef("_ = %s\n", p.Name)
		}
	} else {
		// Mark params as declared
		for _, p := range f.Params {
			g.declareVar(p.Name)
		}
	}

	g.currentFunc = f
	g.inFunc = true
	for _, s := range f.Body {
		if err := g.writeStmt(s); err != nil {
			return err
		}
	}
	if !g.bodyHasImplicitReturn(f.Body) {
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
	g.w.Dedent()
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
	g.w.LineDirective(line)
}

// writePanicHandler emits the defer/recover block used in all main() functions.
func (g *codeGen) writePanicHandler() {
	g.writeln(`defer func() {`)
	g.w.Indent()
	g.writeln(`if e := recover(); e != nil {`)
	g.w.Indent()
	g.writeln(`if shellErr, ok := e.(rugoShellError); ok {`)
	g.w.Indent()
	g.writeln(`os.Exit(shellErr.code)`)
	g.w.Dedent()
	g.writeln(`}`)
	g.writeln(`rugo_panic_handler(e)`)
	g.w.Dedent()
	g.writeln(`}`)
	g.w.Dedent()
	g.writeln(`}()`)
}

// captureOutput runs fn while writing to a temporary buffer,
// then restores the original buffer and returns the captured output.
func (g *codeGen) captureOutput(fn func() error) (string, error) {
	return g.w.Capture(fn)
}
