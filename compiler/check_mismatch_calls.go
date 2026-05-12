package compiler

import (
	"fmt"

	"github.com/rubiojr/rugo/ast"
)

// checkCallSites validates literal arguments at every direct call to a
// user-defined function with annotated parameters: when the argument is
// a literal whose static type concretely conflicts with the parameter's
// annotation, emit a structured compile-time error.
//
// Only literal arguments are flagged. Variable arguments and computed
// expressions are intentionally silent — inference is conservative and a
// variable may legitimately hold a value of the annotated type even when
// the inferrer cannot prove it.
//
// The compatibility rule is the permissive one (compatibleWithAnnotation),
// the same predicate used at return sites: numeric types are mutually
// compatible, `string` / `bool` annotations accept anything, and `any`
// accepts anything. This mirrors what codegen actually does at the call
// boundary (rugo_to_int / rugo_to_float / rugo_to_string / rugo_to_bool
// coercions and numeric casts).
//
// Module / method calls (`str.upper(...)`, `obj.method(...)`) are skipped
// because they have no Rugo-level parameter annotations to compare. Calls
// resolved through a current namespace (sibling calls within a require'd
// module) ARE checked, matching codegen's resolution order.
func checkCallSites(prog *ast.Program, sourceFile string) error {
	c := &callChecker{
		sourceFile: sourceFile,
		funcs:      collectAnnotatedFuncs(prog),
	}
	if len(c.funcs) == 0 {
		return nil
	}
	for _, s := range prog.Statements {
		if err := c.checkStmt(s, ""); err != nil {
			return err
		}
	}
	return nil
}

// callChecker walks the program and validates every direct CallExpr's
// literal arguments against the callee's parameter annotations.
//
// The funcs map is keyed by funcKey (i.e. "ns.name" for namespaced
// functions, bare "name" for top-level), and stores the *original*
// ast.FuncDef rather than going through TypeInfo. Inference may widen or
// erase annotated param types (e.g. for functions with default values),
// but the annotations themselves live on the ast.Param nodes and are the
// ground truth for this check.
type callChecker struct {
	sourceFile string
	funcs      map[string]*ast.FuncDef
}

// collectAnnotatedFuncs builds the lookup of user-defined functions with
// at least one annotated parameter. Functions without any annotated
// parameter have no contract to violate, so they are not tracked.
func collectAnnotatedFuncs(prog *ast.Program) map[string]*ast.FuncDef {
	out := make(map[string]*ast.FuncDef)
	for _, s := range prog.Statements {
		f, ok := s.(*ast.FuncDef)
		if !ok {
			continue
		}
		if !anyParamAnnotated(f.Params) {
			continue
		}
		key := funcKey(f)
		// First definition wins; codegen reports duplicates separately.
		if _, exists := out[key]; !exists {
			out[key] = f
		}
	}
	return out
}

func anyParamAnnotated(params []ast.Param) bool {
	for _, p := range params {
		if p.TypeAnnot != "" {
			return true
		}
	}
	return false
}

// checkStmt validates a statement and recurses into any nested bodies.
// currentNS is the namespace of the enclosing function (empty at the
// top level), used to resolve sibling calls within a require'd module.
func (c *callChecker) checkStmt(s ast.Statement, currentNS string) error {
	line, file := s.StmtLine(), s.StmtSource()
	if file == "" {
		file = c.sourceFile
	}

	switch st := s.(type) {
	case *ast.FuncDef:
		ns := st.Namespace
		for _, child := range st.Body {
			if err := c.checkStmt(child, ns); err != nil {
				return err
			}
		}

	case *ast.ExprStmt:
		return c.walkExpr(st.Expression, line, file, currentNS)

	case *ast.AssignStmt:
		return c.walkExpr(st.Value, line, file, currentNS)

	case *ast.IndexAssignStmt:
		if err := c.walkExpr(st.Object, line, file, currentNS); err != nil {
			return err
		}
		if err := c.walkExpr(st.Index, line, file, currentNS); err != nil {
			return err
		}
		return c.walkExpr(st.Value, line, file, currentNS)

	case *ast.DotAssignStmt:
		if err := c.walkExpr(st.Object, line, file, currentNS); err != nil {
			return err
		}
		return c.walkExpr(st.Value, line, file, currentNS)

	case *ast.ReturnStmt:
		if st.Value != nil {
			return c.walkExpr(st.Value, line, file, currentNS)
		}

	case *ast.ImplicitReturnStmt:
		return c.walkExpr(st.Value, line, file, currentNS)

	case *ast.TryResultStmt:
		return c.walkExpr(st.Value, line, file, currentNS)

	case *ast.SpawnReturnStmt:
		if st.Value != nil {
			return c.walkExpr(st.Value, line, file, currentNS)
		}

	case *ast.TryHandlerReturnStmt:
		if st.Value != nil {
			return c.walkExpr(st.Value, line, file, currentNS)
		}

	case *ast.IfStmt:
		if err := c.walkExpr(st.Condition, line, file, currentNS); err != nil {
			return err
		}
		for _, child := range st.Body {
			if err := c.checkStmt(child, currentNS); err != nil {
				return err
			}
		}
		for _, ec := range st.ElsifClauses {
			if err := c.walkExpr(ec.Condition, line, file, currentNS); err != nil {
				return err
			}
			for _, child := range ec.Body {
				if err := c.checkStmt(child, currentNS); err != nil {
					return err
				}
			}
		}
		for _, child := range st.ElseBody {
			if err := c.checkStmt(child, currentNS); err != nil {
				return err
			}
		}

	case *ast.WhileStmt:
		if err := c.walkExpr(st.Condition, line, file, currentNS); err != nil {
			return err
		}
		for _, child := range st.Body {
			if err := c.checkStmt(child, currentNS); err != nil {
				return err
			}
		}

	case *ast.ForStmt:
		if err := c.walkExpr(st.Collection, line, file, currentNS); err != nil {
			return err
		}
		for _, child := range st.Body {
			if err := c.checkStmt(child, currentNS); err != nil {
				return err
			}
		}

	case *ast.CaseStmt:
		if err := c.walkExpr(st.Subject, line, file, currentNS); err != nil {
			return err
		}
		for _, oc := range st.OfClauses {
			for _, v := range oc.Values {
				if err := c.walkExpr(v, line, file, currentNS); err != nil {
					return err
				}
			}
			if oc.ArrowExpr != nil {
				if err := c.walkExpr(oc.ArrowExpr, line, file, currentNS); err != nil {
					return err
				}
			}
			for _, child := range oc.Body {
				if err := c.checkStmt(child, currentNS); err != nil {
					return err
				}
			}
		}
		for _, ec := range st.ElsifClauses {
			if err := c.walkExpr(ec.Condition, line, file, currentNS); err != nil {
				return err
			}
			for _, child := range ec.Body {
				if err := c.checkStmt(child, currentNS); err != nil {
					return err
				}
			}
		}
		for _, child := range st.ElseBody {
			if err := c.checkStmt(child, currentNS); err != nil {
				return err
			}
		}

	case *ast.TestDef:
		for _, child := range st.Body {
			if err := c.checkStmt(child, ""); err != nil {
				return err
			}
		}

	case *ast.BenchDef:
		for _, child := range st.Body {
			if err := c.checkStmt(child, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkExpr walks an expression, checking every CallExpr it finds and
// recursing into nested fn lambdas and lowered concurrency wrappers.
// CallExprs use the enclosing statement's source line for diagnostics
// (CallExpr nodes don't carry their own position).
func (c *callChecker) walkExpr(e ast.Expr, line int, file string, currentNS string) error {
	if e == nil {
		return nil
	}
	if call, ok := e.(*ast.CallExpr); ok {
		if err := c.checkCall(call, line, file, currentNS); err != nil {
			return err
		}
	}
	switch ex := e.(type) {
	case *ast.CallExpr:
		if err := c.walkExpr(ex.Func, line, file, currentNS); err != nil {
			return err
		}
		for _, a := range ex.Args {
			if err := c.walkExpr(a, line, file, currentNS); err != nil {
				return err
			}
		}

	case *ast.BinaryExpr:
		if err := c.walkExpr(ex.Left, line, file, currentNS); err != nil {
			return err
		}
		return c.walkExpr(ex.Right, line, file, currentNS)

	case *ast.UnaryExpr:
		return c.walkExpr(ex.Operand, line, file, currentNS)

	case *ast.IndexExpr:
		if err := c.walkExpr(ex.Object, line, file, currentNS); err != nil {
			return err
		}
		return c.walkExpr(ex.Index, line, file, currentNS)

	case *ast.SliceExpr:
		if err := c.walkExpr(ex.Object, line, file, currentNS); err != nil {
			return err
		}
		if err := c.walkExpr(ex.Start, line, file, currentNS); err != nil {
			return err
		}
		return c.walkExpr(ex.Length, line, file, currentNS)

	case *ast.DotExpr:
		return c.walkExpr(ex.Object, line, file, currentNS)

	case *ast.ArrayLiteral:
		for _, el := range ex.Elements {
			if err := c.walkExpr(el, line, file, currentNS); err != nil {
				return err
			}
		}

	case *ast.HashLiteral:
		for _, p := range ex.Pairs {
			if err := c.walkExpr(p.Key, line, file, currentNS); err != nil {
				return err
			}
			if err := c.walkExpr(p.Value, line, file, currentNS); err != nil {
				return err
			}
		}

	case *ast.FnExpr:
		// Inside a fn lambda body, each statement carries its own line.
		for _, child := range ex.Body {
			if err := c.checkStmt(child, currentNS); err != nil {
				return err
			}
		}
		// Default-value expressions are evaluated in the enclosing scope.
		for _, p := range ex.Params {
			if p.Default != nil {
				if err := c.walkExpr(p.Default, line, file, currentNS); err != nil {
					return err
				}
			}
		}

	case *ast.CaseExpr:
		if err := c.walkExpr(ex.Subject, line, file, currentNS); err != nil {
			return err
		}
		for _, oc := range ex.OfClauses {
			for _, v := range oc.Values {
				if err := c.walkExpr(v, line, file, currentNS); err != nil {
					return err
				}
			}
			if oc.ArrowExpr != nil {
				if err := c.walkExpr(oc.ArrowExpr, line, file, currentNS); err != nil {
					return err
				}
			}
			for _, child := range oc.Body {
				if err := c.checkStmt(child, currentNS); err != nil {
					return err
				}
			}
		}
		for _, ec := range ex.ElsifClauses {
			if err := c.walkExpr(ec.Condition, line, file, currentNS); err != nil {
				return err
			}
			for _, child := range ec.Body {
				if err := c.checkStmt(child, currentNS); err != nil {
					return err
				}
			}
		}
		for _, child := range ex.ElseBody {
			if err := c.checkStmt(child, currentNS); err != nil {
				return err
			}
		}

	case *ast.LoweredTryExpr:
		if err := c.walkExpr(ex.Expr, line, file, currentNS); err != nil {
			return err
		}
		for _, child := range ex.Handler {
			if err := c.checkStmt(child, currentNS); err != nil {
				return err
			}
		}
		if ex.ResultExpr != nil {
			if err := c.walkExpr(ex.ResultExpr, line, file, currentNS); err != nil {
				return err
			}
		}

	case *ast.LoweredSpawnExpr:
		for _, child := range ex.Body {
			if err := c.checkStmt(child, currentNS); err != nil {
				return err
			}
		}
		if ex.ResultExpr != nil {
			if err := c.walkExpr(ex.ResultExpr, line, file, currentNS); err != nil {
				return err
			}
		}

	case *ast.LoweredParallelExpr:
		for _, br := range ex.Branches {
			if br.Expr != nil {
				if err := c.walkExpr(br.Expr, line, file, currentNS); err != nil {
					return err
				}
			}
			for _, child := range br.Stmts {
				if err := c.checkStmt(child, currentNS); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// checkCall validates a single CallExpr against the callee's annotated
// parameters. Only direct identifier-named calls are checked: method and
// module calls (`obj.foo(...)`, `str.upper(...)`) are skipped because
// they have no Rugo-level annotation to compare.
//
// Within a namespaced function body, a bare `f(...)` call is first
// resolved as `currentNS.f` (sibling call within the same module) before
// falling back to a top-level `f`, matching codegen's resolution order
// in buildCallExpr.
func (c *callChecker) checkCall(call *ast.CallExpr, line int, file string, currentNS string) error {
	ident, ok := call.Func.(*ast.IdentExpr)
	if !ok {
		return nil
	}
	fn := c.resolveCallee(ident.Name, currentNS)
	if fn == nil {
		return nil
	}
	for i, arg := range call.Args {
		if i >= len(fn.Params) {
			break
		}
		p := fn.Params[i]
		if p.TypeAnnot == "" {
			continue
		}
		annot, ok := ParseTypeAnnotation(p.TypeAnnot)
		if !ok {
			continue
		}
		argType, isLit := literalType(arg)
		if !isLit {
			continue
		}
		if compatibleWithAnnotation(annot, argType) {
			continue
		}
		return &ast.UserError{Msg: fmt.Sprintf(
			"%s:%d: cannot pass %s literal as argument %d to '%s' (parameter '%s' declared as %s)",
			file, line, displayTypeName(argType), i+1, displayCalleeName(ident.Name, fn.Namespace),
			p.Name, displayTypeName(annot),
		)}
	}
	return nil
}

// resolveCallee mirrors codegen's bare-identifier resolution: try the
// current namespace's sibling function first, then a top-level function.
func (c *callChecker) resolveCallee(name, currentNS string) *ast.FuncDef {
	if currentNS != "" {
		if f, ok := c.funcs[currentNS+"."+name]; ok {
			return f
		}
	}
	return c.funcs[name]
}

// displayCalleeName returns the user-facing name for a callee in error
// messages. Namespaced sibling calls show as "ns.name" even though the
// source-level call was bare, so the user can find the offending def.
func displayCalleeName(name, namespace string) string {
	if namespace == "" {
		return name
	}
	return namespace + "." + name
}

// literalType reports the static type of an expression that is plainly a
// literal in source. Variables, function calls, indexing, and other
// computed expressions return (TypeUnknown, false) and are never flagged.
//
// Unary "-" applied to a numeric literal counts as a literal of the
// same numeric type so `f(-1)` is treated like `f(1)`. Unary "!" applied
// to a bool literal counts as a bool literal.
func literalType(e ast.Expr) (RugoType, bool) {
	switch ex := e.(type) {
	case *ast.IntLiteral:
		return TypeInt, true
	case *ast.FloatLiteral:
		return TypeFloat, true
	case *ast.StringLiteral:
		return TypeString, true
	case *ast.BoolLiteral:
		return TypeBool, true
	case *ast.NilLiteral:
		return TypeNil, true
	case *ast.ArrayLiteral:
		return TypeArray, true
	case *ast.HashLiteral:
		return TypeHash, true
	case *ast.UnaryExpr:
		switch ex.Op {
		case "-":
			if t, ok := literalType(ex.Operand); ok && (t == TypeInt || t == TypeFloat) {
				return t, true
			}
		case "!":
			if t, ok := literalType(ex.Operand); ok && t == TypeBool {
				return t, true
			}
		}
	}
	return TypeUnknown, false
}
