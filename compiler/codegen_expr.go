package compiler

import (
	"fmt"
	"github.com/rubiojr/rugo/ast"
	"strings"

	"github.com/rubiojr/rugo/gobridge"
	"github.com/rubiojr/rugo/modules"
	"github.com/rubiojr/rugo/parser"
)

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
				if expected.Min != 0 {
					return "", fmt.Errorf("function '%s' expects %d argument(s), called with 0", ex.Name, expected.Min)
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
	case *ast.LoweredTryExpr:
		return g.loweredTryExpr(ex)
	case *ast.LoweredSpawnExpr:
		return g.loweredSpawnExpr(ex)
	case *ast.LoweredParallelExpr:
		return g.loweredParallelExpr(ex)
	case *ast.FnExpr:
		return g.fnExpr(ex)
	default:
		return "", fmt.Errorf("unknown expression type: %T", e)
	}
}

func (g *codeGen) stringLiteral(value string, typed bool) (string, error) {
	if ast.HasInterpolation(value) {
		format, exprStrs, err := ast.ProcessInterpolation(value)
		if err != nil {
			return "", err
		}
		args := make([]string, len(exprStrs))
		argTypes := make([]RugoType, len(exprStrs))
		for i, exprStr := range exprStrs {
			goExpr, typ, err := g.compileInterpolatedExpr(exprStr)
			if err != nil {
				return "", fmt.Errorf("interpolation error in #{%s}: %w", exprStr, err)
			}
			args[i] = goExpr
			argTypes[i] = typ
		}
		escapedFmt := goEscapeString(format)
		if len(args) > 0 {
			// Optimization: when all interpolated expressions are typed strings,
			// emit direct concatenation instead of fmt.Sprintf.
			allString := true
			for _, t := range argTypes {
				if t != TypeString {
					allString = false
					break
				}
			}
			if allString {
				return g.buildStringConcat(escapedFmt, args), nil
			}
			// Wrap each arg in rugo_to_string so types like []byte
			// are properly converted (fmt.Sprintf %v prints []byte as
			// integer list instead of string content).
			wrappedArgs := make([]string, len(args))
			for i, a := range args {
				wrappedArgs[i] = fmt.Sprintf("rugo_to_string(%s)", a)
			}
			return fmt.Sprintf(`fmt.Sprintf("%s", %s)`, escapedFmt, strings.Join(wrappedArgs, ", ")), nil
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

// compileInterpolatedExpr parses a rugo expression string and returns the
// generated Go code along with the inferred type of the expression.
func (g *codeGen) compileInterpolatedExpr(exprStr string) (string, RugoType, error) {
	src := exprStr + "\n"
	p := &parser.Parser{}
	flatAST, err := p.Parse("<interpolation>", []byte(src))
	if err != nil {
		// Replace raw Go runtime panics (e.g. "index out of range") with
		// a user-friendly message.
		if strings.Contains(err.Error(), "runtime error:") {
			return "", TypeDynamic, fmt.Errorf("syntax error in expression")
		}
		// Detect bare-style calls with commas inside interpolation (e.g. #{append arr, val})
		// and suggest the function call form instead.
		if strings.Contains(exprStr, ",") {
			parts := strings.Fields(exprStr)
			if len(parts) >= 2 {
				return "", TypeDynamic, fmt.Errorf("syntax error: bare-style call not supported inside interpolation — use %s(%s) instead", parts[0], strings.Join(parts[1:], " "))
			}
		}
		return "", TypeDynamic, fmt.Errorf("syntax error in expression")
	}
	prog, err := ast.Walk(p, flatAST)
	if err != nil {
		return "", TypeDynamic, fmt.Errorf("walking: %w", err)
	}
	if len(prog.Statements) == 0 {
		return `""`, TypeString, nil
	}
	var expr ast.Expr
	switch s := prog.Statements[0].(type) {
	case *ast.ExprStmt:
		expr = s.Expression
	case *ast.AssignStmt:
		expr = s.Value
	default:
		return "", TypeDynamic, fmt.Errorf("unexpected statement type in interpolation: %T", s)
	}
	goExpr, err := g.exprString(expr)
	if err != nil {
		return "", TypeDynamic, err
	}
	return goExpr, g.interpExprType(expr), nil
}

// interpExprType infers the type of an interpolated expression.
// Since interpolated expressions are parsed separately from the main AST,
// we check the AST node directly and use varType for identifier lookups.
func (g *codeGen) interpExprType(e ast.Expr) RugoType {
	switch ex := e.(type) {
	case *ast.IdentExpr:
		return g.varType(ex.Name)
	case *ast.StringLiteral:
		return TypeString
	case *ast.IntLiteral:
		return TypeInt
	case *ast.FloatLiteral:
		return TypeFloat
	case *ast.BoolLiteral:
		return TypeBool
	default:
		return TypeDynamic
	}
}

// buildStringConcat emits a Go string concatenation expression from a format
// string with %v placeholders and corresponding typed string arguments.
func (g *codeGen) buildStringConcat(escapedFmt string, args []string) string {
	segments := strings.Split(escapedFmt, "%v")
	var parts []string
	for i, seg := range segments {
		if seg != "" {
			parts = append(parts, fmt.Sprintf(`"%s"`, seg))
		}
		if i < len(args) {
			parts = append(parts, args[i])
		}
	}
	if len(parts) == 0 {
		return `""`
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "(" + strings.Join(parts, " + ") + ")"
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
					// Zero-param entries (vars/consts) — property access, no parens needed.
					if len(sig.Params) == 0 {
						return g.generateGoBridgeCall(pkg, sig, nil, nsName+"."+e.Field), nil
					}
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
					if strings.HasPrefix(dot.Field, "_") {
						return "", fmt.Errorf("'%s' is private to module '%s'", dot.Field, nsName)
					}
					nsKey := nsName + "." + dot.Field
					if expected, ok := g.funcDefs[nsKey]; ok {
						if len(e.Args) < expected.Min || len(e.Args) > expected.Max {
							return "", arityCountError(nsName+"."+dot.Field, len(e.Args), expected)
						}
						if expected.HasDefaults {
							return fmt.Sprintf("rugons_%s_%s(%s)", nsName, dot.Field, argStr), nil
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
		case "range":
			if len(e.Args) < 1 || len(e.Args) > 2 {
				return "", fmt.Errorf("range expects 1 or 2 arguments, got %d", len(e.Args))
			}
			return fmt.Sprintf("rugo_range(%s)", g.boxedArgs(args, e.Args)), nil
		default:
			// Sibling function call within a namespace: resolve unqualified
			// calls against the current function's namespace first.
			if g.currentFunc != nil && g.currentFunc.Namespace != "" {
				nsKey := g.currentFunc.Namespace + "." + ident.Name
				if expected, ok := g.funcDefs[nsKey]; ok {
					if len(e.Args) < expected.Min || len(e.Args) > expected.Max {
						return "", arityCountError(ident.Name, len(e.Args), expected)
					}
					if expected.HasDefaults {
						return fmt.Sprintf("rugons_%s_%s(%s)", g.currentFunc.Namespace, ident.Name, argStr), nil
					}
					typedArgs := g.typedCallArgs(nsKey, args, e.Args)
					return fmt.Sprintf("rugons_%s_%s(%s)", g.currentFunc.Namespace, ident.Name, typedArgs), nil
				}
			}
			// User-defined function — validate argument count
			if expected, ok := g.funcDefs[ident.Name]; ok {
				if len(e.Args) < expected.Min || len(e.Args) > expected.Max {
					return "", arityCountError(ident.Name, len(e.Args), expected)
				}
				if expected.HasDefaults {
					return fmt.Sprintf("rugofn_%s(%s)", ident.Name, argStr), nil
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

func (g *codeGen) loweredTryExpr(e *ast.LoweredTryExpr) (string, error) {
	exprStr, err := g.exprString(e.Expr)
	if err != nil {
		return "", err
	}

	// Build handler body as GoStmt nodes
	g.pushScope()
	g.declareVar(e.ErrVar)
	var handlerBody []GoStmt
	if e.ResultExpr != nil {
		stmts, berr := g.buildStmts(e.Handler)
		if berr != nil {
			g.popScope()
			return "", berr
		}
		handlerBody = append(handlerBody, stmts...)
		val, verr := g.exprString(e.ResultExpr)
		if verr != nil {
			g.popScope()
			return "", verr
		}
		handlerBody = append(handlerBody, GoAssignStmt{Target: "r", Op: "=", Value: GoRawExpr{Code: val}})
	} else {
		stmts, berr := g.buildStmts(e.Handler)
		if berr != nil {
			g.popScope()
			return "", berr
		}
		handlerBody = append(handlerBody, stmts...)
	}
	g.popScope()

	ir := GoIIFEExpr{
		ReturnType: "(r interface{})",
		Body: []GoStmt{
			GoDeferStmt{Body: []GoStmt{
				GoIfStmt{Cond: GoRawExpr{Code: "e := recover(); e != nil"}, Body: append(
					[]GoStmt{
						GoAssignStmt{Target: fmt.Sprintf("%s", e.ErrVar), Op: ":=", Value: GoRawExpr{Code: "fmt.Sprint(e)"}},
						GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", e.ErrVar)}},
					},
					handlerBody...,
				)},
			}},
		},
		Result: GoRawExpr{Code: exprStr},
	}

	return g.goExprStr(ir), nil
}

func (g *codeGen) loweredSpawnExpr(e *ast.LoweredSpawnExpr) (string, error) {
	// Build body as GoStmt nodes
	g.pushScope()
	bodyStmts, err := g.buildStmts(e.Body)
	if err != nil {
		g.popScope()
		return "", err
	}
	if e.ResultExpr != nil {
		val, verr := g.exprString(e.ResultExpr)
		if verr != nil {
			g.popScope()
			return "", verr
		}
		bodyStmts = append(bodyStmts, GoAssignStmt{Target: "t.result", Op: "=", Value: GoRawExpr{Code: val}})
	}
	g.popScope()

	ir := GoIIFEExpr{
		Body: []GoStmt{
			GoRawStmt{Code: "t := &rugoTask{done: make(chan struct{})}"},
			GoGoStmt{Body: []GoStmt{
				GoDeferStmt{Body: []GoStmt{
					GoIfStmt{Cond: GoRawExpr{Code: "e := recover(); e != nil"}, Body: []GoStmt{
						GoRawStmt{Code: "t.err = fmt.Sprint(e)"},
					}},
					GoRawStmt{Code: "close(t.done)"},
				}},
				GoRawStmt{Code: "// spawn body"},
			}},
		},
		Result: GoRawExpr{Code: "interface{}(t)"},
	}

	// Insert body stmts into the goroutine body, replacing the placeholder
	goStmt := ir.Body[1].(GoGoStmt)
	goStmt.Body = append(goStmt.Body[:len(goStmt.Body)-1], bodyStmts...)
	ir.Body[1] = goStmt

	return g.goExprStr(ir), nil
}

func (g *codeGen) fnExpr(e *ast.FnExpr) (string, error) {
	// Build lambda body as GoStmt nodes
	g.lambdaDepth++
	g.lambdaOuterFunc = append(g.lambdaOuterFunc, g.currentFunc)
	g.pushScope()
	g.lambdaScopeBase = append(g.lambdaScopeBase, len(g.scopes)-1)
	for _, p := range e.Params {
		g.declareVar(p.Name)
	}
	savedFunc := g.currentFunc
	savedInFunc := g.inFunc
	g.inFunc = true

	restoreLambda := func() {
		g.inFunc = savedInFunc
		g.currentFunc = savedFunc
		g.lambdaScopeBase = g.lambdaScopeBase[:len(g.lambdaScopeBase)-1]
		g.lambdaOuterFunc = g.lambdaOuterFunc[:len(g.lambdaOuterFunc)-1]
		g.popScope()
		g.lambdaDepth--
	}

	bodyStmts, berr := g.buildStmts(e.Body)
	if berr != nil {
		restoreLambda()
		return "", berr
	}
	restoreLambda()

	// Build preamble: arity check + param unpacking
	var preamble []GoStmt
	hasDefaults := ast.HasDefaults(e.Params)
	if hasDefaults {
		minArity := ast.MinArity(e.Params)
		maxArity := len(e.Params)
		preamble = append(preamble, GoRawStmt{Code: fmt.Sprintf(
			"if len(_args) < %d || len(_args) > %d { panic(fmt.Sprintf(\"lambda takes %d to %d arguments but %%d %%s given\", len(_args), map[bool]string{true: \"was\", false: \"were\"}[len(_args) == 1])) }",
			minArity, maxArity, minArity, maxArity)})
	} else {
		nParams := len(e.Params)
		if nParams == 1 {
			preamble = append(preamble, GoRawStmt{Code: fmt.Sprintf(
				"if len(_args) != %d { panic(fmt.Sprintf(\"lambda takes %d argument but %%d were given\", len(_args))) }",
				nParams, nParams)})
		} else {
			preamble = append(preamble, GoRawStmt{Code: fmt.Sprintf(
				"if len(_args) != %d { panic(fmt.Sprintf(\"lambda takes %d arguments but %%d %%s given\", len(_args), map[bool]string{true: \"was\", false: \"were\"}[len(_args) == 1])) }",
				nParams, nParams)})
		}
	}

	for i, p := range e.Params {
		if p.Default != nil {
			defaultExpr, derr := g.exprString(p.Default)
			if derr != nil {
				return "", derr
			}
			preamble = append(preamble, GoRawStmt{Code: fmt.Sprintf("var %s interface{}", p.Name)})
			preamble = append(preamble, GoRawStmt{Code: fmt.Sprintf("if len(_args) > %d { %s = _args[%d] } else { %s = %s }", i, p.Name, i, p.Name, defaultExpr)})
		} else {
			preamble = append(preamble, GoRawStmt{Code: fmt.Sprintf("var %s interface{}", p.Name)})
			preamble = append(preamble, GoRawStmt{Code: fmt.Sprintf("if len(_args) > %d { %s = _args[%d] }", i, p.Name, i)})
		}
		preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", p.Name)}})
	}

	// Assemble full body
	var fullBody []GoStmt
	fullBody = append(fullBody, preamble...)
	fullBody = append(fullBody, bodyStmts...)
	fullBody = append(fullBody, GoReturnStmt{Value: GoRawExpr{Code: "nil"}})

	// Render using GoIIFEExpr — but we need interface{}(...) wrapping
	ir := GoIIFEExpr{
		ReturnType: "interface{}",
		Body:       fullBody,
	}

	// The lambda is wrapped in interface{}(func(...) ... { ... })
	// We render the IIFE without the outer func()...{}() and wrap manually
	// Actually, GoIIFEExpr renders as func() T { body }() — but lambdas are
	// interface{}(func(_args ...interface{}) interface{} { body }) — not called.
	// We need a different rendering for lambdas.

	// Use a custom rendering since lambda shape differs from IIFE
	p := &goPrinter{indent: g.w.indent + 1}
	var sb strings.Builder
	sb.WriteString("interface{}(func(_args ...interface{}) interface{} {\n")
	for _, s := range fullBody {
		p.printStmt(s)
	}
	sb.WriteString(p.sb.String())
	for range g.w.indent {
		sb.WriteByte('\t')
	}
	sb.WriteString("})")

	_ = ir // suppress unused
	return sb.String(), nil
}

func (g *codeGen) loweredParallelExpr(e *ast.LoweredParallelExpr) (string, error) {
	n := len(e.Branches)

	if n == 0 {
		return "interface{}([]interface{}{})", nil
	}

	// Build each branch as GoStmt nodes
	type branchInfo struct {
		stmts  []GoStmt
		isExpr bool
	}
	branches := make([]branchInfo, n)
	for _, br := range e.Branches {
		if br.Expr != nil {
			code, err := g.exprString(br.Expr)
			if err != nil {
				return "", err
			}
			branches[br.Index] = branchInfo{
				stmts:  []GoStmt{GoRawStmt{Code: fmt.Sprintf("_results[%d] = %s", br.Index, code)}},
				isExpr: true,
			}
		} else {
			g.pushScope()
			stmts, err := g.buildStmts(br.Stmts)
			if err != nil {
				g.popScope()
				return "", err
			}
			g.popScope()
			branches[br.Index] = branchInfo{stmts: stmts, isExpr: false}
		}
	}

	// Build goroutine nodes
	var goroutines []GoStmt
	for _, bc := range branches {
		goroutineBody := []GoStmt{
			GoRawStmt{Code: "defer _wg.Done()"},
			GoDeferStmt{Body: []GoStmt{
				GoIfStmt{Cond: GoRawExpr{Code: "e := recover(); e != nil"}, Body: []GoStmt{
					GoRawStmt{Code: `_parOnce.Do(func() { _parErr = fmt.Sprint(e) })`},
				}},
			}},
		}
		goroutineBody = append(goroutineBody, bc.stmts...)
		goroutines = append(goroutines, GoGoStmt{Body: goroutineBody})
	}

	body := []GoStmt{
		GoRawStmt{Code: fmt.Sprintf("_results := make([]interface{}, %d)", n)},
		GoRawStmt{Code: "var _wg sync.WaitGroup"},
		GoRawStmt{Code: "var _parErr string"},
		GoRawStmt{Code: "var _parOnce sync.Once"},
		GoRawStmt{Code: fmt.Sprintf("_wg.Add(%d)", n)},
	}
	body = append(body, goroutines...)
	body = append(body,
		GoRawStmt{Code: "_wg.Wait()"},
		GoIfStmt{Cond: GoRawExpr{Code: `_parErr != ""`}, Body: []GoStmt{
			GoRawStmt{Code: "panic(_parErr)"},
		}},
		GoRawStmt{Code: "out := make([]interface{}, len(_results))"},
		GoRawStmt{Code: "copy(out, _results)"},
	)

	ir := GoIIFEExpr{
		Body:   body,
		Result: GoRawExpr{Code: "interface{}(out)"},
	}

	return g.goExprStr(ir), nil
}
