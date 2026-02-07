package compiler

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

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
	constScopes     []map[string]bool // track constant bindings (uppercase names)
	inFunc          bool
	imports         map[string]bool // stdlib modules imported
	sourceFile      string          // original .rg filename for //line directives
	hasSpawn        bool            // whether spawn is used
	hasParallel     bool            // whether parallel is used
	usesTaskMethods bool            // whether .value/.done/.wait appear
	funcDefs        map[string]int  // user function name → param count
}

// generate produces Go source code from a Program AST.
func generate(prog *Program, sourceFile string) (string, error) {
	g := &codeGen{
		declared:    make(map[string]bool),
		scopes:      []map[string]bool{make(map[string]bool)},
		constScopes: []map[string]bool{make(map[string]bool)},
		imports:     make(map[string]bool),
		sourceFile:  sourceFile,
		funcDefs:    make(map[string]int),
	}
	return g.generate(prog)
}

func (g *codeGen) generate(prog *Program) (string, error) {
	// Collect imports and separate functions, tests, and top-level statements
	var funcs []*FuncDef
	var tests []*TestDef
	var topStmts []Statement
	var setupFunc *FuncDef
	var teardownFunc *FuncDef
	for _, s := range prog.Statements {
		switch st := s.(type) {
		case *FuncDef:
			if st.Name == "setup" && st.Namespace == "" {
				setupFunc = st
			} else if st.Name == "teardown" && st.Namespace == "" {
				teardownFunc = st
			}
			funcs = append(funcs, st)
		case *TestDef:
			tests = append(tests, st)
		case *RequireStmt:
			continue
		case *ImportStmt:
			g.imports[st.Module] = true
			continue
		default:
			topStmts = append(topStmts, s)
		}
		_ = s
	}

	// Build function definition registry for argument count validation
	for _, f := range funcs {
		key := f.Name
		if f.Namespace != "" {
			key = f.Namespace + "." + f.Name
		}
		g.funcDefs[key] = len(f.Params)
	}

	// Detect spawn/parallel usage to gate runtime emission and imports
	g.hasSpawn = astUsesSpawn(prog)
	g.hasParallel = astUsesParallel(prog)
	g.usesTaskMethods = astUsesTaskMethods(prog)
	needsSpawnRuntime := g.hasSpawn || g.usesTaskMethods
	needsSyncImport := needsSpawnRuntime || g.hasParallel

	g.writeln("package main")
	g.writeln("")
	g.writeln("import (")
	g.indent++
	g.writeln(`"fmt"`)
	g.writeln(`"os"`)
	g.writeln(`"os/exec"`)
	g.writeln(`"runtime/debug"`)
	g.writeln(`"strings"`)
	if needsSyncImport {
		g.writeln(`"sync"`)
	}
	if needsSpawnRuntime {
		g.writeln(`"time"`)
	}
	baseImports := map[string]bool{
		"fmt": true, "os": true, "os/exec": true,
		"runtime/debug": true, "strings": true,
	}
	for _, name := range importedModuleNames(g.imports) {
		if m, ok := modules.Get(name); ok {
			for _, imp := range m.GoImports {
				if baseImports[imp] {
					continue
				}
				if strings.Contains(imp, `"`) {
					// Already formatted (e.g. aliased import: alias "path")
					g.writef("%s\n", imp)
				} else {
					g.writef("\"%s\"\n", imp)
				}
			}
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
	g.writeln("var _ = debug.Stack")
	if needsSyncImport {
		g.writeln("var _ sync.Once")
	}
	if needsSpawnRuntime {
		g.writeln("var _ = time.Now")
	}
	g.writeln("")

	// Runtime helpers
	g.writeRuntime()

	// User-defined functions
	for _, f := range funcs {
		if err := g.writeFunc(f); err != nil {
			return "", err
		}
		g.writeln("")
	}

	// Dispatch maps for modules that declare DispatchEntry
	g.writeDispatchMaps(funcs)

	if len(tests) > 0 {
		return g.generateTestHarness(tests, topStmts, setupFunc, teardownFunc)
	}

	// Main function
	g.writeln("func main() {")
	g.indent++
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

func (g *codeGen) generateTestHarness(tests []*TestDef, topStmts []Statement, setup, teardown *FuncDef) (string, error) {
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
		for _, s := range t.Body {
			if err := g.writeStmt(s); err != nil {
				return "", err
			}
		}
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
	if setup != nil {
		setupArg = "rugofn_setup"
	}
	if teardown != nil {
		teardownArg = "rugofn_teardown"
	}
	g.writef("}, %s, %s, _test)\n", setupArg, teardownArg)

	g.popScope()
	g.indent--
	g.writeln("}")

	return g.sb.String(), nil
}

func (g *codeGen) writeRuntime() {
	g.sb.WriteString(runtimeCorePre)

	// Module runtimes (only for imported modules)
	for _, name := range importedModuleNames(g.imports) {
		if m, ok := modules.Get(name); ok {
			g.sb.WriteString(m.FullRuntime())
		}
	}

	g.sb.WriteString(runtimeCorePost)

	if g.hasSpawn || g.usesTaskMethods {
		g.writeSpawnRuntime()
	}
}

func (g *codeGen) writeSpawnRuntime() {
	g.sb.WriteString(runtimeSpawn)
}

// writeDispatchMaps generates typed dispatch maps for modules that declare DispatchEntry.
// Each map maps user-defined function names to their Go implementations.
// Only non-namespaced functions with exactly 1 parameter are included (handler convention).
func (g *codeGen) writeDispatchMaps(funcs []*FuncDef) {
	for _, name := range importedModuleNames(g.imports) {
		m, ok := modules.Get(name)
		if !ok || m.DispatchEntry == "" {
			continue
		}
		g.writef("var rugo_%s_dispatch = map[string]func(interface{}) interface{}{\n", m.Name)
		g.indent++
		for _, f := range funcs {
			if f.Namespace != "" || len(f.Params) != 1 {
				continue
			}
			g.writef("%q: rugofn_%s,\n", f.Name, f.Name)
		}
		g.indent--
		g.writeln("}")
		g.writeln("")
	}
}

func (g *codeGen) writeFunc(f *FuncDef) error {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p + " interface{}"
	}

	// Determine function name: namespaced or local
	var goName string
	if f.Namespace != "" {
		goName = fmt.Sprintf("rugons_%s_%s", f.Namespace, f.Name)
	} else {
		goName = fmt.Sprintf("rugofn_%s", f.Name)
	}

	g.writef("func %s(%s) interface{} {\n", goName, strings.Join(params, ", "))
	g.indent++
	g.pushScope()
	// Mark params as declared
	for _, p := range f.Params {
		g.declareVar(p)
	}
	g.inFunc = true
	for _, s := range f.Body {
		if err := g.writeStmt(s); err != nil {
			return err
		}
	}
	g.writeln("return nil")
	g.inFunc = false
	g.popScope()
	g.indent--
	g.writeln("}")
	return nil
}

// emitLineDirective writes a //line directive for the original source file.
func (g *codeGen) emitLineDirective(line int) {
	if line > 0 && g.sourceFile != "" {
		g.sb.WriteString(fmt.Sprintf("//line %s:%d\n", g.sourceFile, line))
	}
}

func (g *codeGen) writeStmt(s Statement) error {
	g.emitLineDirective(s.StmtLine())
	switch st := s.(type) {
	case *AssignStmt:
		return g.writeAssign(st)
	case *IndexAssignStmt:
		return g.writeIndexAssign(st)
	case *ExprStmt:
		return g.writeExprStmt(st)
	case *IfStmt:
		return g.writeIf(st)
	case *WhileStmt:
		return g.writeWhile(st)
	case *ForStmt:
		return g.writeFor(st)
	case *BreakStmt:
		g.writeln("break")
		return nil
	case *NextStmt:
		g.writeln("continue")
		return nil
	case *ReturnStmt:
		return g.writeReturn(st)
	case *FuncDef:
		// Nested functions not supported at codegen level (hoisted earlier)
		return fmt.Errorf("nested function definitions not supported")
	case *RequireStmt:
		// Handled at compiler level
		return nil
	case *ImportStmt:
		// Handled during generate phase
		return nil
	default:
		return fmt.Errorf("unknown statement type: %T", s)
	}
}

func (g *codeGen) writeAssign(a *AssignStmt) error {
	// Uppercase names are constants — reject reassignment
	if g.isConstant(a.Target) {
		return fmt.Errorf("line %d: cannot reassign constant %s", a.SourceLine, a.Target)
	}

	expr, err := g.exprString(a.Value)
	if err != nil {
		return err
	}
	if g.isDeclared(a.Target) {
		g.writef("%s = %s\n", a.Target, expr)
	} else {
		g.writef("%s := %s\n", a.Target, expr)
		g.declareVar(a.Target)
		if len(a.Target) > 0 && a.Target[0] >= 'A' && a.Target[0] <= 'Z' {
			g.declareConst(a.Target)
		}
	}
	// Suppress "declared but not used" by referencing with _
	g.writef("_ = %s\n", a.Target)
	return nil
}

func (g *codeGen) writeIndexAssign(ia *IndexAssignStmt) error {
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

func (g *codeGen) writeExprStmt(e *ExprStmt) error {
	expr, err := g.exprString(e.Expression)
	if err != nil {
		return err
	}
	g.writef("_ = %s\n", expr)
	return nil
}

func (g *codeGen) writeIf(i *IfStmt) error {
	cond, err := g.exprString(i.Condition)
	if err != nil {
		return err
	}
	g.writef("if rugo_to_bool(%s) {\n", cond)
	g.indent++
	g.pushScope()
	for _, s := range i.Body {
		if err := g.writeStmt(s); err != nil {
			return err
		}
	}
	g.popScope()
	g.indent--
	for _, ec := range i.ElsifClauses {
		cond, err := g.exprString(ec.Condition)
		if err != nil {
			return err
		}
		g.writef("} else if rugo_to_bool(%s) {\n", cond)
		g.indent++
		g.pushScope()
		for _, s := range ec.Body {
			if err := g.writeStmt(s); err != nil {
				return err
			}
		}
		g.popScope()
		g.indent--
	}
	if len(i.ElseBody) > 0 {
		g.writeln("} else {")
		g.indent++
		g.pushScope()
		for _, s := range i.ElseBody {
			if err := g.writeStmt(s); err != nil {
				return err
			}
		}
		g.popScope()
		g.indent--
	}
	g.writeln("}")
	return nil
}

func (g *codeGen) writeWhile(w *WhileStmt) error {
	cond, err := g.exprString(w.Condition)
	if err != nil {
		return err
	}
	g.writef("for rugo_to_bool(%s) {\n", cond)
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

func (g *codeGen) writeFor(f *ForStmt) error {
	coll, err := g.exprString(f.Collection)
	if err != nil {
		return err
	}

	iterVar := f.Var
	idxVar := f.IndexVar

	g.writef("for _, rugo_for_kv := range rugo_iterable(%s) {\n", coll)
	g.indent++
	g.pushScope()

	// Declare the loop variable(s)
	if idxVar != "" {
		// Two-variable form: for key, val in hash / for idx, val in arr
		g.writef("%s := rugo_for_kv.Key\n", iterVar)
		g.writef("_ = %s\n", iterVar)
		g.declareVar(iterVar)
		g.writef("%s := rugo_for_kv.Val\n", idxVar)
		g.writef("_ = %s\n", idxVar)
		g.declareVar(idxVar)
	} else {
		// Single-variable form: for val in arr
		g.writef("%s := rugo_for_kv.Val\n", iterVar)
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

func (g *codeGen) writeReturn(r *ReturnStmt) error {
	if r.Value == nil {
		g.writeln("return nil")
	} else {
		expr, err := g.exprString(r.Value)
		if err != nil {
			return err
		}
		g.writef("return %s\n", expr)
	}
	return nil
}

func (g *codeGen) exprString(e Expr) (string, error) {
	switch ex := e.(type) {
	case *IntLiteral:
		return fmt.Sprintf("interface{}(%s)", ex.Value), nil
	case *FloatLiteral:
		return fmt.Sprintf("interface{}(%s)", ex.Value), nil
	case *BoolLiteral:
		if ex.Value {
			return "interface{}(true)", nil
		}
		return "interface{}(false)", nil
	case *NilLiteral:
		return "interface{}(nil)", nil
	case *StringLiteral:
		if ex.Raw {
			escaped := goEscapeString(ex.Value)
			return fmt.Sprintf(`interface{}("%s")`, escaped), nil
		}
		return g.stringLiteral(ex.Value)
	case *IdentExpr:
		return fmt.Sprintf("interface{}(%s)", ex.Name), nil
	case *DotExpr:
		return g.dotExpr(ex)
	case *BinaryExpr:
		return g.binaryExpr(ex)
	case *UnaryExpr:
		return g.unaryExpr(ex)
	case *CallExpr:
		return g.callExpr(ex)
	case *IndexExpr:
		return g.indexExpr(ex)
	case *SliceExpr:
		return g.sliceExpr(ex)
	case *ArrayLiteral:
		return g.arrayLiteral(ex)
	case *HashLiteral:
		return g.hashLiteral(ex)
	case *TryExpr:
		return g.tryExpr(ex)
	case *SpawnExpr:
		return g.spawnExpr(ex)
	case *ParallelExpr:
		return g.parallelExpr(ex)
	default:
		return "", fmt.Errorf("unknown expression type: %T", e)
	}
}

func (g *codeGen) stringLiteral(value string) (string, error) {
	if hasInterpolation(value) {
		format, exprStrs := processInterpolation(value)
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
		return fmt.Sprintf(`interface{}("%s")`, escapedFmt), nil
	}
	escaped := goEscapeString(value)
	return fmt.Sprintf(`interface{}("%s")`, escaped), nil
}

// compileInterpolatedExpr parses a rugo expression string and generates Go code.
func (g *codeGen) compileInterpolatedExpr(exprStr string) (string, error) {
	// Wrap in a dummy statement so the parser can handle it
	src := exprStr + "\n"
	p := &parser.Parser{}
	// Parse as a full program with just this expression
	fullSrc := src
	ast, err := p.Parse("<interpolation>", []byte(fullSrc))
	if err != nil {
		return "", fmt.Errorf("parsing: %w", err)
	}
	prog, err := walk(p, ast)
	if err != nil {
		return "", fmt.Errorf("walking: %w", err)
	}
	if len(prog.Statements) == 0 {
		return `interface{}("")`, nil
	}
	// Extract the expression from the statement
	switch s := prog.Statements[0].(type) {
	case *ExprStmt:
		return g.exprString(s.Expression)
	case *AssignStmt:
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

func (g *codeGen) binaryExpr(e *BinaryExpr) (string, error) {
	left, err := g.exprString(e.Left)
	if err != nil {
		return "", err
	}
	right, err := g.exprString(e.Right)
	if err != nil {
		return "", err
	}
	switch e.Op {
	case "+":
		return fmt.Sprintf("rugo_add(%s, %s)", left, right), nil
	case "-":
		return fmt.Sprintf("rugo_sub(%s, %s)", left, right), nil
	case "*":
		return fmt.Sprintf("rugo_mul(%s, %s)", left, right), nil
	case "/":
		return fmt.Sprintf("rugo_div(%s, %s)", left, right), nil
	case "%":
		return fmt.Sprintf("rugo_mod(%s, %s)", left, right), nil
	case "==", "!=", "<", ">", "<=", ">=":
		return fmt.Sprintf("rugo_compare(%q, %s, %s)", e.Op, left, right), nil
	case "&&":
		return fmt.Sprintf("interface{}(rugo_to_bool(%s) && rugo_to_bool(%s))", left, right), nil
	case "||":
		return fmt.Sprintf("interface{}(rugo_to_bool(%s) || rugo_to_bool(%s))", left, right), nil
	default:
		return "", fmt.Errorf("unknown operator: %s", e.Op)
	}
}

func (g *codeGen) unaryExpr(e *UnaryExpr) (string, error) {
	operand, err := g.exprString(e.Operand)
	if err != nil {
		return "", err
	}
	switch e.Op {
	case "-":
		return fmt.Sprintf("rugo_negate(%s)", operand), nil
	case "!":
		return fmt.Sprintf("rugo_not(%s)", operand), nil
	default:
		return "", fmt.Errorf("unknown unary operator: %s", e.Op)
	}
}

func (g *codeGen) dotExpr(e *DotExpr) (string, error) {
	// Stdlib or namespace access without call
	if ns, ok := e.Object.(*IdentExpr); ok {
		nsName := ns.Name
		if nsName == "__tmod__" {
			nsName = "test"
		}
		if goFunc, ok := modules.LookupFunc(nsName, e.Field); ok {
			return fmt.Sprintf("interface{}(%s)", goFunc), nil
		}
		// Task method access (no-arg): task.value, task.done
		switch e.Field {
		case "value":
			g.usesTaskMethods = true
			return fmt.Sprintf("rugo_task_value(%s)", nsName), nil
		case "done":
			g.usesTaskMethods = true
			return fmt.Sprintf("rugo_task_done(%s)", nsName), nil
		}
		return fmt.Sprintf("interface{}(rugons_%s_%s)", nsName, e.Field), nil
	}
	obj, err := g.exprString(e.Object)
	if err != nil {
		return "", err
	}
	// Task method access on non-ident expressions
	switch e.Field {
	case "value":
		g.usesTaskMethods = true
		return fmt.Sprintf("rugo_task_value(%s)", obj), nil
	case "done":
		g.usesTaskMethods = true
		return fmt.Sprintf("rugo_task_done(%s)", obj), nil
	}
	return fmt.Sprintf("interface{}(%s)", obj), nil
}

func (g *codeGen) callExpr(e *CallExpr) (string, error) {
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
	if dot, ok := e.Func.(*DotExpr); ok {
		if ns, ok := dot.Object.(*IdentExpr); ok {
			nsName := ns.Name
			if nsName == "__tmod__" {
				nsName = "test"
			}
			if goFunc, ok := modules.LookupFunc(nsName, dot.Field); ok {
				return fmt.Sprintf("%s(%s)", goFunc, argStr), nil
			}
			// Task method calls: task.wait(n), task.value(), task.done()
			switch dot.Field {
			case "wait":
				g.usesTaskMethods = true
				return fmt.Sprintf("rugo_task_wait(%s, %s)", nsName, argStr), nil
			case "value":
				g.usesTaskMethods = true
				return fmt.Sprintf("rugo_task_value(%s)", nsName), nil
			case "done":
				g.usesTaskMethods = true
				return fmt.Sprintf("rugo_task_done(%s)", nsName), nil
			}
			// User module namespace call — validate argument count
			nsKey := nsName + "." + dot.Field
			if expected, ok := g.funcDefs[nsKey]; ok {
				if len(e.Args) != expected {
					return "", fmt.Errorf("wrong number of arguments for %s.%s (%d for %d)", nsName, dot.Field, len(e.Args), expected)
				}
			}
			return fmt.Sprintf("rugons_%s_%s(%s)", nsName, dot.Field, argStr), nil
		}
		// Non-ident object: e.g. tasks[i].wait(n)
		obj, oerr := g.exprString(dot.Object)
		if oerr != nil {
			return "", oerr
		}
		switch dot.Field {
		case "wait":
			g.usesTaskMethods = true
			return fmt.Sprintf("rugo_task_wait(%s, %s)", obj, argStr), nil
		case "value":
			g.usesTaskMethods = true
			return fmt.Sprintf("rugo_task_value(%s)", obj), nil
		case "done":
			g.usesTaskMethods = true
			return fmt.Sprintf("rugo_task_done(%s)", obj), nil
		}
	}

	// Check for built-in functions (globals)
	if ident, ok := e.Func.(*IdentExpr); ok {
		switch ident.Name {
		case "puts":
			return fmt.Sprintf("rugo_puts(%s)", argStr), nil
		case "print":
			return fmt.Sprintf("rugo_print(%s)", argStr), nil
		case "__shell__":
			return fmt.Sprintf("rugo_shell(%s)", argStr), nil
		case "__capture__":
			return fmt.Sprintf("rugo_capture(%s)", argStr), nil
		case "len":
			return fmt.Sprintf("rugo_len(%s)", argStr), nil
		case "append":
			return fmt.Sprintf("rugo_append(%s)", argStr), nil
		default:
			// User-defined function — validate argument count
			if expected, ok := g.funcDefs[ident.Name]; ok {
				if len(e.Args) != expected {
					return "", fmt.Errorf("wrong number of arguments for %s (%d for %d)", ident.Name, len(e.Args), expected)
				}
			}
			return fmt.Sprintf("rugofn_%s(%s)", ident.Name, argStr), nil
		}
	}

	// Dynamic call (shouldn't happen in v1 but handle gracefully)
	funcExpr, err := g.exprString(e.Func)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.(%s)(%s)", funcExpr, "func(...interface{}) interface{}", argStr), nil
}

func (g *codeGen) indexExpr(e *IndexExpr) (string, error) {
	obj, err := g.exprString(e.Object)
	if err != nil {
		return "", err
	}
	idx, err := g.exprString(e.Index)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("func() interface{} { switch o := (%s).(type) { case []interface{}: return rugo_array_index(o, rugo_to_int(%s)); case map[interface{}]interface{}: return o[%s]; default: panic(fmt.Sprintf(\"cannot index %%T\", o)) } }()", obj, idx, idx), nil
}

func (g *codeGen) sliceExpr(e *SliceExpr) (string, error) {
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

func (g *codeGen) arrayLiteral(e *ArrayLiteral) (string, error) {
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

func (g *codeGen) hashLiteral(e *HashLiteral) (string, error) {
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

func (g *codeGen) tryExpr(e *TryExpr) (string, error) {
	exprStr, err := g.exprString(e.Expr)
	if err != nil {
		return "", err
	}

	// Build the handler body as Go source in a temporary buffer.
	var handlerBuf strings.Builder
	savedSB := g.sb
	g.sb = strings.Builder{}

	g.pushScope()
	g.declareVar(e.ErrVar)

	for i, s := range e.Handler {
		isLast := i == len(e.Handler)-1
		if isLast {
			// Last statement: if it's a bare expression, assign to r (return value)
			if es, ok := s.(*ExprStmt); ok {
				val, verr := g.exprString(es.Expression)
				if verr != nil {
					g.popScope()
					g.sb = savedSB
					return "", verr
				}
				g.writef("r = %s\n", val)
				continue
			}
		}
		if werr := g.writeStmt(s); werr != nil {
			g.popScope()
			g.sb = savedSB
			return "", werr
		}
	}

	g.popScope()
	handlerCode := g.sb.String()
	g.sb = savedSB

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

func (g *codeGen) spawnExpr(e *SpawnExpr) (string, error) {
	// Generate the body code in a temporary buffer.
	savedSB := g.sb
	g.sb = strings.Builder{}

	g.pushScope()
	for i, s := range e.Body {
		isLast := i == len(e.Body)-1
		if isLast {
			// Last statement: if it's a bare expression, assign to t.result
			if es, ok := s.(*ExprStmt); ok {
				val, verr := g.exprString(es.Expression)
				if verr != nil {
					g.popScope()
					g.sb = savedSB
					return "", verr
				}
				g.writef("t.result = %s\n", val)
				continue
			}
		}
		if werr := g.writeStmt(s); werr != nil {
			g.popScope()
			g.sb = savedSB
			return "", werr
		}
	}
	g.popScope()
	bodyCode := g.sb.String()
	g.sb = savedSB

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

func (g *codeGen) parallelExpr(e *ParallelExpr) (string, error) {
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
		if es, ok := s.(*ExprStmt); ok {
			code, err := g.exprString(es.Expression)
			if err != nil {
				return "", err
			}
			stmts[i] = stmtCode{code: code, isExpr: true}
		} else {
			// Non-expression statement: generate into a temp buffer
			savedSB := g.sb
			g.sb = strings.Builder{}
			g.pushScope()
			if err := g.writeStmt(s); err != nil {
				g.popScope()
				g.sb = savedSB
				return "", err
			}
			g.popScope()
			code := g.sb.String()
			g.sb = savedSB
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

// Scope management
func (g *codeGen) pushScope() {
	g.scopes = append(g.scopes, make(map[string]bool))
	g.constScopes = append(g.constScopes, make(map[string]bool))
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

func (g *codeGen) declareConst(name string) {
	g.constScopes[len(g.constScopes)-1][name] = true
}

func (g *codeGen) isConstant(name string) bool {
	for i := len(g.constScopes) - 1; i >= 0; i-- {
		if g.constScopes[i][name] {
			return true
		}
	}
	return false
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

// astUsesSpawn checks if any SpawnExpr exists in the AST.
func astUsesSpawn(prog *Program) bool {
	return walkExpressions(prog, func(e Expr) bool {
		_, ok := e.(*SpawnExpr)
		return ok
	})
}

// astUsesTaskMethods checks if any DotExpr uses .value, .done, or .wait on a non-module target.
func astUsesTaskMethods(prog *Program) bool {
	return walkExpressions(prog, func(e Expr) bool {
		dot, ok := e.(*DotExpr)
		if !ok || !taskMethodNames[dot.Field] {
			return false
		}
		if ident, ok := dot.Object.(*IdentExpr); ok {
			if modules.IsModule(ident.Name) || ident.Name == "__tmod__" {
				return false
			}
		}
		return true
	})
}

var taskMethodNames = map[string]bool{"value": true, "done": true, "wait": true}

// astUsesParallel checks if any ParallelExpr exists in the AST.
func astUsesParallel(prog *Program) bool {
	return walkExpressions(prog, func(e Expr) bool {
		_, ok := e.(*ParallelExpr)
		return ok
	})
}
