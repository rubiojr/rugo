package compiler

import (
	"fmt"
	"github.com/rubiojr/rugo/ast"
	"strings"

	"github.com/rubiojr/rugo/modules"
)

func (g *codeGen) buildTestHarness(tests []*ast.TestDef, topStmts []ast.Statement, setup, teardown, setupFile, teardownFile *ast.FuncDef) ([]GoDecl, error) {
	var decls []GoDecl

	// Emit each test as a function
	for i, t := range tests {
		funcName := fmt.Sprintf("rugo_test_%d", i)

		testDefer := GoDeferStmt{Body: []GoStmt{
			GoIfStmt{
				Cond: GoRawExpr{Code: "r := recover(); r != nil"},
				Body: []GoStmt{
					GoIfStmt{
						Cond: GoRawExpr{Code: "reason, ok := r.(rugoTestSkip); ok"},
						Body: []GoStmt{
							GoRawStmt{Code: "skipped = true"},
							GoRawStmt{Code: "skipReason = string(reason)"},
							GoReturnStmt{},
						},
					},
					GoRawStmt{Code: `failColor := "\033[31m"`},
					GoRawStmt{Code: `failReset := "\033[0m"`},
					GoIfStmt{
						Cond: GoRawExpr{Code: `os.Getenv("NO_COLOR") != ""`},
						Body: []GoStmt{
							GoRawStmt{Code: `failColor = ""`},
							GoRawStmt{Code: `failReset = ""`},
						},
					},
					GoRawStmt{Code: `failReason = fmt.Sprintf("%v", r)`},
					GoRawStmt{Code: `fmt.Fprintf(os.Stderr, "  %sFAIL%s: %v\n", failColor, failReset, r)`},
					GoRawStmt{Code: "passed = false"},
				},
			},
		}}

		var body []GoStmt
		body = append(body, testDefer)

		g.pushScope()
		g.varTypeScope = fmt.Sprintf("__test_%p", t)
		bodyStmts, err := g.buildStmts(t.Body)
		if err != nil {
			return nil, err
		}
		g.varTypeScope = ""
		g.popScope()

		body = append(body, bodyStmts...)
		body = append(body, GoRawStmt{Code: "passed = true"})
		body = append(body, GoReturnStmt{})

		decls = append(decls, GoFuncDecl{
			Name:   funcName,
			Return: "(passed bool, skipped bool, skipReason string, failReason string)",
			Body:   body,
		})
	}

	// Main function: delegate to runtime test runner
	var mainBody []GoStmt
	mainBody = append(mainBody, g.buildPanicHandler())

	g.pushScope()
	topBody, err := g.buildStmts(topStmts)
	if err != nil {
		g.popScope()
		return nil, err
	}
	mainBody = append(mainBody, topBody...)

	// Build test cases and call the runtime runner
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

	var runnerCall strings.Builder
	runnerCall.WriteString("rugo_test_runner([]rugoTestCase{\n")
	for i, t := range tests {
		escapedName := goEscapeString(t.Name)
		fmt.Fprintf(&runnerCall, "\t{Name: \"%s\", Func: rugo_test_%d},\n", escapedName, i)
	}
	fmt.Fprintf(&runnerCall, "}, %s, %s, %s, %s, _test)", setupArg, teardownArg, setupFileArg, teardownFileArg)
	mainBody = append(mainBody, GoRawStmt{Code: runnerCall.String()})

	g.popScope()

	decls = append(decls, GoFuncDecl{
		Name: "main",
		Body: mainBody,
	})

	return decls, nil
}

func (g *codeGen) buildBenchHarness(benches []*ast.BenchDef, topStmts []ast.Statement) ([]GoDecl, error) {
	var decls []GoDecl

	// Emit each benchmark as a function
	for i, b := range benches {
		funcName := fmt.Sprintf("rugo_bench_%d", i)

		g.pushScope()
		g.varTypeScope = fmt.Sprintf("__bench_%p", b)
		bodyStmts, err := g.buildStmts(b.Body)
		if err != nil {
			return nil, err
		}
		g.varTypeScope = ""
		g.popScope()

		decls = append(decls, GoFuncDecl{
			Name: funcName,
			Body: bodyStmts,
		})
	}

	// Main function: run benchmarks via runtime runner
	var mainBody []GoStmt
	mainBody = append(mainBody, g.buildPanicHandler())

	g.pushScope()
	topBody, err := g.buildStmts(topStmts)
	if err != nil {
		g.popScope()
		return nil, err
	}
	mainBody = append(mainBody, topBody...)

	// Build bench cases and call the runtime runner
	var runnerCall strings.Builder
	runnerCall.WriteString("rugo_bench_runner([]rugoBenchCase{\n")
	for i, b := range benches {
		escapedName := goEscapeString(b.Name)
		fmt.Fprintf(&runnerCall, "\t{Name: \"%s\", Func: rugo_bench_%d},\n", escapedName, i)
	}
	runnerCall.WriteString("})")
	mainBody = append(mainBody, GoRawStmt{Code: runnerCall.String()})

	g.popScope()

	decls = append(decls, GoFuncDecl{
		Name: "main",
		Body: mainBody,
	})

	return decls, nil
}

func (g *codeGen) buildDispatchMaps(funcs []*ast.FuncDef, handlers map[string]bool) []GoDecl {
	var decls []GoDecl
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
		var sb strings.Builder
		fmt.Fprintf(&sb, "var rugo_%s_dispatch = map[string]func(interface{}) interface{}{\n", m.Name)
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
			fmt.Fprintf(&sb, "\t%q: %s,\n", f.Name, goName)
		}
		sb.WriteString("}\n")
		decls = append(decls, GoRawDecl{Code: sb.String()})
		decls = append(decls, GoBlankLine{})
	}
	return decls
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
