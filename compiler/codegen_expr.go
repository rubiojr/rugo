package compiler

import (
	"fmt"
	"github.com/rubiojr/rugo/ast"
	"github.com/rubiojr/rugo/preprocess"
	"strings"

	"github.com/rubiojr/rugo/gobridge"
	"github.com/rubiojr/rugo/modules"
	"github.com/rubiojr/rugo/parser"
)

// buildExpr converts a Rugo AST expression into a Go output AST expression.
func (g *codeGen) buildExpr(e ast.Expr) (GoExpr, error) {
	switch ex := e.(type) {
	case *ast.IntLiteral:
		lit := GoIntLit{Value: ex.Value}
		if g.exprIsTyped(e) {
			return lit, nil
		}
		return GoCastExpr{Type: "interface{}", Value: lit}, nil

	case *ast.FloatLiteral:
		lit := GoFloatLit{Value: ex.Value}
		if g.exprIsTyped(e) {
			return lit, nil
		}
		return GoCastExpr{Type: "interface{}", Value: lit}, nil

	case *ast.BoolLiteral:
		lit := GoBoolLit{Value: ex.Value}
		if g.exprIsTyped(e) {
			return lit, nil
		}
		return GoCastExpr{Type: "interface{}", Value: lit}, nil

	case *ast.NilLiteral:
		return GoCastExpr{Type: "interface{}", Value: GoNilExpr{}}, nil

	case *ast.StringLiteral:
		if ex.Raw {
			lit := GoStringLit{Value: goEscapeString(ex.Value)}
			if g.exprIsTyped(e) {
				return lit, nil
			}
			return GoCastExpr{Type: "interface{}", Value: lit}, nil
		}
		return g.buildStringLiteral(ex.Value, g.exprIsTyped(e))

	case *ast.IdentExpr:
		// Bare function name without parens: treat as zero-arg call (Ruby semantics).
		if !g.isDeclared(ex.Name) {
			if expected, ok := g.funcDefs[ex.Name]; ok {
				if expected.Min != 0 {
					return nil, fmt.Errorf("function '%s' expects %d argument(s), called with 0", ex.Name, expected.Min)
				}
				call := GoCallExpr{Func: fmt.Sprintf("rugofn_%s", ex.Name)}
				if g.typeInfo != nil {
					if fti, ok := g.typeInfo.FuncTypes[ex.Name]; ok && fti.ReturnType.IsTyped() {
						return call, nil
					}
				}
				return GoCastExpr{Type: "interface{}", Value: call}, nil
			}
		}
		// Sibling constant reference within a namespace
		if g.currentFunc != nil && g.currentFunc.Namespace != "" && !g.isDeclared(ex.Name) {
			nsKey := g.currentFunc.Namespace + "." + ex.Name
			if g.nsVarNames[nsKey] {
				return GoIdentExpr{Name: fmt.Sprintf("rugons_%s_%s", g.currentFunc.Namespace, ex.Name)}, nil
			}
		}
		return GoIdentExpr{Name: ex.Name}, nil

	case *ast.BinaryExpr:
		return g.buildBinaryExpr(ex)
	case *ast.UnaryExpr:
		return g.buildUnaryExpr(ex)
	case *ast.IndexExpr:
		return g.buildIndexExpr(ex)
	case *ast.SliceExpr:
		return g.buildSliceExpr(ex)
	case *ast.ArrayLiteral:
		return g.buildArrayLiteral(ex)
	case *ast.HashLiteral:
		return g.buildHashLiteral(ex)

	case *ast.DotExpr:
		return g.buildDotExpr(ex)
	case *ast.CallExpr:
		return g.buildCallExpr(ex)
	case *ast.LoweredTryExpr:
		return g.buildLoweredTryExpr(ex)
	case *ast.LoweredSpawnExpr:
		return g.buildLoweredSpawnExpr(ex)
	case *ast.LoweredParallelExpr:
		return g.buildLoweredParallelExpr(ex)
	case *ast.FnExpr:
		return g.buildFnExpr(ex)

	default:
		return nil, fmt.Errorf("unknown expression type: %T", e)
	}
}

func (g *codeGen) buildBinaryExpr(e *ast.BinaryExpr) (GoExpr, error) {
	leftType := g.exprType(e.Left)
	rightType := g.exprType(e.Right)

	left, err := g.buildExpr(e.Left)
	if err != nil {
		return nil, err
	}
	right, err := g.buildExpr(e.Right)
	if err != nil {
		return nil, err
	}

	p := &goPrinter{}
	leftStr := p.exprStr(left)
	rightStr := p.exprStr(right)
	boxedLeft := GoRawExpr{Code: g.boxed(leftStr, leftType)}
	boxedRight := GoRawExpr{Code: g.boxed(rightStr, rightType)}

	typedBinOp := func(op string) GoExpr {
		return GoParenExpr{Inner: GoBinaryExpr{Left: left, Op: op, Right: right}}
	}
	typedFloatBinOp := func(op string) GoExpr {
		return GoParenExpr{Inner: GoBinaryExpr{
			Left:  GoRawExpr{Code: g.ensureFloat(leftStr, leftType)},
			Op:    op,
			Right: GoRawExpr{Code: g.ensureFloat(rightStr, rightType)},
		}}
	}
	runtimeCall := func(fn string) GoExpr {
		return GoCallExpr{Func: fn, Args: []GoExpr{boxedLeft, boxedRight}}
	}

	bothGoTyped := g.goTyped(e.Left) && g.goTyped(e.Right)
	bothInts := leftType == TypeInt && rightType == TypeInt && bothGoTyped
	bothNumeric := leftType.IsNumeric() && rightType.IsNumeric() && leftType.IsTyped() && rightType.IsTyped() && bothGoTyped
	bothStrings := leftType == TypeString && rightType == TypeString && bothGoTyped
	sameTyped := leftType == rightType && leftType.IsTyped()

	switch e.Op {
	case "+":
		if bothInts || bothStrings {
			return typedBinOp("+"), nil
		}
		if bothNumeric {
			return typedFloatBinOp("+"), nil
		}
		return runtimeCall("rugo_add"), nil
	case "-":
		if bothInts {
			return typedBinOp("-"), nil
		}
		if bothNumeric {
			return typedFloatBinOp("-"), nil
		}
		return runtimeCall("rugo_sub"), nil
	case "*":
		if bothInts {
			return typedBinOp("*"), nil
		}
		if bothNumeric {
			return typedFloatBinOp("*"), nil
		}
		return runtimeCall("rugo_mul"), nil
	case "/":
		if bothInts {
			return typedBinOp("/"), nil
		}
		if bothNumeric {
			return typedFloatBinOp("/"), nil
		}
		return runtimeCall("rugo_div"), nil
	case "%":
		if bothInts {
			return typedBinOp("%"), nil
		}
		return runtimeCall("rugo_mod"), nil
	case "==":
		if sameTyped {
			return typedBinOp("=="), nil
		}
		return runtimeCall("rugo_eq"), nil
	case "!=":
		if sameTyped {
			return typedBinOp("!="), nil
		}
		return runtimeCall("rugo_neq"), nil
	case "<":
		if sameTyped && (leftType.IsNumeric() || leftType == TypeString) {
			return typedBinOp("<"), nil
		}
		return runtimeCall("rugo_lt"), nil
	case ">":
		if sameTyped && (leftType.IsNumeric() || leftType == TypeString) {
			return typedBinOp(">"), nil
		}
		return runtimeCall("rugo_gt"), nil
	case "<=":
		if sameTyped && (leftType.IsNumeric() || leftType == TypeString) {
			return typedBinOp("<="), nil
		}
		return runtimeCall("rugo_le"), nil
	case ">=":
		if sameTyped && (leftType.IsNumeric() || leftType == TypeString) {
			return typedBinOp(">="), nil
		}
		return runtimeCall("rugo_ge"), nil
	case "&&":
		if leftType == TypeBool && rightType == TypeBool {
			return typedBinOp("&&"), nil
		}
		// Ruby-like: return left if falsy, otherwise right
		return GoIIFEExpr{
			ReturnType: "interface{}",
			Body: []GoStmt{
				GoAssignStmt{Target: "_left", Op: ":=", Value: boxedLeft},
				GoIfStmt{
					Cond: GoUnaryExpr{Op: "!", Operand: GoCallExpr{Func: "rugo_to_bool", Args: []GoExpr{GoIdentExpr{Name: "_left"}}}},
					Body: []GoStmt{GoReturnStmt{Value: GoIdentExpr{Name: "_left"}}},
				},
			},
			Result: boxedRight,
		}, nil
	case "||":
		if leftType == TypeBool && rightType == TypeBool {
			return typedBinOp("||"), nil
		}
		// Ruby-like: return left if truthy, otherwise right
		return GoIIFEExpr{
			ReturnType: "interface{}",
			Body: []GoStmt{
				GoAssignStmt{Target: "_left", Op: ":=", Value: boxedLeft},
				GoIfStmt{
					Cond: GoCallExpr{Func: "rugo_to_bool", Args: []GoExpr{GoIdentExpr{Name: "_left"}}},
					Body: []GoStmt{GoReturnStmt{Value: GoIdentExpr{Name: "_left"}}},
				},
			},
			Result: boxedRight,
		}, nil
	default:
		return nil, fmt.Errorf("unknown operator: %s", e.Op)
	}
}

func (g *codeGen) buildUnaryExpr(e *ast.UnaryExpr) (GoExpr, error) {
	operandType := g.exprType(e.Operand)
	operand, err := g.buildExpr(e.Operand)
	if err != nil {
		return nil, err
	}
	p := &goPrinter{}
	switch e.Op {
	case "-":
		if operandType == TypeInt || operandType == TypeFloat {
			return GoParenExpr{Inner: GoUnaryExpr{Op: "-", Operand: operand}}, nil
		}
		return GoCallExpr{Func: "rugo_negate", Args: []GoExpr{GoRawExpr{Code: g.boxed(p.exprStr(operand), operandType)}}}, nil
	case "!":
		if operandType == TypeBool {
			return GoParenExpr{Inner: GoUnaryExpr{Op: "!", Operand: operand}}, nil
		}
		return GoCallExpr{Func: "rugo_not", Args: []GoExpr{GoRawExpr{Code: g.boxed(p.exprStr(operand), operandType)}}}, nil
	default:
		return nil, fmt.Errorf("unknown unary operator: %s", e.Op)
	}
}

func (g *codeGen) buildIndexExpr(e *ast.IndexExpr) (GoExpr, error) {
	obj, err := g.buildExpr(e.Object)
	if err != nil {
		return nil, err
	}
	idx, err := g.buildExpr(e.Index)
	if err != nil {
		return nil, err
	}
	return GoCallExpr{Func: "rugo_index", Args: []GoExpr{obj, idx}}, nil
}

func (g *codeGen) buildSliceExpr(e *ast.SliceExpr) (GoExpr, error) {
	obj, err := g.buildExpr(e.Object)
	if err != nil {
		return nil, err
	}
	start, err := g.buildExpr(e.Start)
	if err != nil {
		return nil, err
	}
	length, err := g.buildExpr(e.Length)
	if err != nil {
		return nil, err
	}
	return GoCallExpr{Func: "rugo_slice", Args: []GoExpr{obj, start, length}}, nil
}

func (g *codeGen) buildArrayLiteral(e *ast.ArrayLiteral) (GoExpr, error) {
	elems := make([]GoExpr, len(e.Elements))
	for i, el := range e.Elements {
		expr, err := g.buildExpr(el)
		if err != nil {
			return nil, err
		}
		elems[i] = expr
	}
	return GoCastExpr{Type: "interface{}", Value: GoSliceLit{Type: "[]interface{}", Elements: elems}}, nil
}

func (g *codeGen) buildHashLiteral(e *ast.HashLiteral) (GoExpr, error) {
	pairs := make([]GoMapPair, len(e.Pairs))
	for i, p := range e.Pairs {
		key, err := g.buildExpr(p.Key)
		if err != nil {
			return nil, err
		}
		val, err := g.buildExpr(p.Value)
		if err != nil {
			return nil, err
		}
		pairs[i] = GoMapPair{Key: key, Value: val}
	}
	return GoCastExpr{Type: "interface{}", Value: GoMapLit{KeyType: "interface{}", ValType: "interface{}", Pairs: pairs}}, nil
}

func (g *codeGen) buildStringLiteral(value string, typed bool) (GoExpr, error) {
	if preprocess.HasInterpolation(value) {
		format, exprStrs, err := preprocess.ProcessInterpolation(value)
		if err != nil {
			return nil, err
		}
		args := make([]string, len(exprStrs))
		argTypes := make([]RugoType, len(exprStrs))
		for i, exprStr := range exprStrs {
			goExpr, typ, err := g.compileInterpolatedExpr(exprStr)
			if err != nil {
				return nil, fmt.Errorf("interpolation error in #{%s}: %w", exprStr, err)
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
				return g.buildStringConcatExpr(escapedFmt, args), nil
			}
			// Wrap each arg in rugo_to_string
			goArgs := make([]GoExpr, len(args))
			for i, a := range args {
				goArgs[i] = GoCallExpr{Func: "rugo_to_string", Args: []GoExpr{GoRawExpr{Code: a}}}
			}
			return GoFmtSprintf{Format: escapedFmt, Args: goArgs}, nil
		}
		lit := GoStringLit{Value: escapedFmt}
		if typed {
			return lit, nil
		}
		return GoCastExpr{Type: "interface{}", Value: lit}, nil
	}
	lit := GoStringLit{Value: goEscapeString(value)}
	if typed {
		return lit, nil
	}
	return GoCastExpr{Type: "interface{}", Value: lit}, nil
}

// buildStringConcatExpr builds a GoStringConcat from a format string
// with %v placeholders and corresponding typed string arguments.
func (g *codeGen) buildStringConcatExpr(escapedFmt string, args []string) GoExpr {
	segments := strings.Split(escapedFmt, "%v")
	var parts []GoExpr
	for i, seg := range segments {
		if seg != "" {
			parts = append(parts, GoStringLit{Value: seg})
		}
		if i < len(args) {
			parts = append(parts, GoRawExpr{Code: args[i]})
		}
	}
	if len(parts) == 0 {
		return GoStringLit{Value: ""}
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return GoStringConcat{Parts: parts}
}

func (g *codeGen) buildLoweredTryExpr(e *ast.LoweredTryExpr) (GoExpr, error) {
	triedExpr, err := g.buildExpr(e.Expr)
	if err != nil {
		return nil, err
	}

	g.pushScope()
	g.declareVar(e.ErrVar)
	var handlerBody []GoStmt
	if e.ResultExpr != nil {
		stmts, berr := g.buildStmts(e.Handler)
		if berr != nil {
			g.popScope()
			return nil, berr
		}
		handlerBody = append(handlerBody, stmts...)
		val, verr := g.buildExpr(e.ResultExpr)
		if verr != nil {
			g.popScope()
			return nil, verr
		}
		handlerBody = append(handlerBody, GoAssignStmt{Target: "r", Op: "=", Value: val})
	} else {
		stmts, berr := g.buildStmts(e.Handler)
		if berr != nil {
			g.popScope()
			return nil, berr
		}
		handlerBody = append(handlerBody, stmts...)
	}
	g.popScope()

	return GoIIFEExpr{
		ReturnType: "(r interface{})",
		Body: []GoStmt{
			GoDeferStmt{Body: []GoStmt{
				GoIfStmt{Cond: GoRawExpr{Code: "e := recover(); e != nil"}, Body: append(
					[]GoStmt{
						GoAssignStmt{Target: e.ErrVar, Op: ":=", Value: GoRawExpr{Code: "fmt.Sprint(e)"}},
						GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", e.ErrVar)}},
					},
					handlerBody...,
				)},
			}},
		},
		Result: triedExpr,
	}, nil
}

func (g *codeGen) buildLoweredSpawnExpr(e *ast.LoweredSpawnExpr) (GoExpr, error) {
	g.pushScope()
	bodyStmts, err := g.buildStmts(e.Body)
	if err != nil {
		g.popScope()
		return nil, err
	}
	if e.ResultExpr != nil {
		val, verr := g.buildExpr(e.ResultExpr)
		if verr != nil {
			g.popScope()
			return nil, verr
		}
		bodyStmts = append(bodyStmts, GoAssignStmt{Target: "t.result", Op: "=", Value: val})
	}
	g.popScope()

	goroutineBody := []GoStmt{
		GoDeferStmt{Body: []GoStmt{
			GoIfStmt{Cond: GoRawExpr{Code: "e := recover(); e != nil"}, Body: []GoStmt{
				GoRawStmt{Code: "t.err = fmt.Sprint(e)"},
			}},
			GoRawStmt{Code: "close(t.done)"},
		}},
	}
	goroutineBody = append(goroutineBody, bodyStmts...)

	return GoIIFEExpr{
		Body: []GoStmt{
			GoRawStmt{Code: "t := &rugoTask{done: make(chan struct{})}"},
			GoGoStmt{Body: goroutineBody},
		},
		Result: GoRawExpr{Code: "interface{}(t)"},
	}, nil
}

func (g *codeGen) buildLoweredParallelExpr(e *ast.LoweredParallelExpr) (GoExpr, error) {
	n := len(e.Branches)

	if n == 0 {
		return GoRawExpr{Code: "interface{}([]interface{}{})"}, nil
	}

	type branchInfo struct {
		stmts  []GoStmt
		isExpr bool
	}
	branches := make([]branchInfo, n)
	for _, br := range e.Branches {
		if br.Expr != nil {
			code, err := g.buildExpr(br.Expr)
			if err != nil {
				return nil, err
			}
			p := &goPrinter{}
			branches[br.Index] = branchInfo{
				stmts:  []GoStmt{GoRawStmt{Code: fmt.Sprintf("_results[%d] = %s", br.Index, p.exprStr(code))}},
				isExpr: true,
			}
		} else {
			g.pushScope()
			stmts, err := g.buildStmts(br.Stmts)
			if err != nil {
				g.popScope()
				return nil, err
			}
			g.popScope()
			branches[br.Index] = branchInfo{stmts: stmts, isExpr: false}
		}
	}

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

	return GoIIFEExpr{
		Body:   body,
		Result: GoRawExpr{Code: "interface{}(out)"},
	}, nil
}

// buildCondExpr builds a condition expression for use in if/while.
// If the condition is typed bool, returns it directly; otherwise wraps with rugo_to_bool.
func (g *codeGen) buildCondExpr(e ast.Expr) (GoExpr, error) {
	expr, err := g.buildExpr(e)
	if err != nil {
		return nil, err
	}
	if g.exprType(e) == TypeBool {
		return expr, nil
	}
	s := (&goPrinter{}).exprStr(expr)
	return GoRawExpr{Code: fmt.Sprintf("rugo_to_bool(%s)", g.boxed(s, g.exprType(e)))}, nil
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
	goExpr, err := g.buildExpr(expr)
	if err != nil {
		return "", TypeDynamic, err
	}
	pr := &goPrinter{}
	return pr.exprStr(goExpr), g.interpExprType(expr), nil
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

func (g *codeGen) buildDotExpr(e *ast.DotExpr) (GoExpr, error) {
	if e.Field == "__type__" {
		return nil, fmt.Errorf("cannot access .__type__ directly — use type_of() instead")
	}
	// Rugo stdlib or namespace access without call
	if ns, ok := e.Object.(*ast.IdentExpr); ok {
		nsName := ns.Name
		// Local variables shadow namespaces for dot access
		if !g.isDeclared(nsName) {
			if g.imports[nsName] {
				if goFunc, ok := modules.LookupFunc(nsName, e.Field); ok {
					return GoCastExpr{Type: "interface{}", Value: GoRawExpr{Code: goFunc}}, nil
				}
			}
			// Go bridge function reference (without call)
			if pkg, ok := gobridge.PackageForNS(nsName, g.goImports); ok {
				if sig, ok := gobridge.Lookup(pkg, e.Field); ok {
					// Zero-param entries (vars/consts) — property access, no parens needed.
					if len(sig.Params) == 0 {
						return GoRawExpr{Code: g.generateGoBridgeCall(pkg, sig, nil, nsName+"."+e.Field)}, nil
					}
					return nil, fmt.Errorf("go bridge function %s.%s must be called with arguments", nsName, e.Field)
				}
			}
			// Known require namespace — function reference
			if g.namespaces[nsName] {
				return GoCastExpr{Type: "interface{}", Value: GoIdentExpr{Name: fmt.Sprintf("rugons_%s_%s", nsName, e.Field)}}, nil
			}
		}
		// Not a known namespace or shadowed by variable — dot access (handles both hashes and tasks at runtime)
		g.usesTaskMethods = g.usesTaskMethods || taskMethodNames[e.Field]
		return GoCallExpr{Func: "rugo_dot_get", Args: []GoExpr{GoIdentExpr{Name: nsName}, GoStringLit{Value: e.Field}}}, nil
	}
	obj, err := g.buildExpr(e.Object)
	if err != nil {
		return nil, err
	}
	// Dot access on non-ident expressions (handles both hashes and tasks at runtime)
	g.usesTaskMethods = g.usesTaskMethods || taskMethodNames[e.Field]
	return GoCallExpr{Func: "rugo_dot_get", Args: []GoExpr{obj, GoStringLit{Value: e.Field}}}, nil
}

func (g *codeGen) buildCallExpr(e *ast.CallExpr) (GoExpr, error) {
	pr := &goPrinter{}
	goArgs := make([]GoExpr, len(e.Args))
	for i, a := range e.Args {
		expr, err := g.buildExpr(a)
		if err != nil {
			return nil, err
		}
		goArgs[i] = expr
	}

	// Check for namespaced function calls: ns.func(args)
	if dot, ok := e.Func.(*ast.DotExpr); ok {
		if ns, ok := dot.Object.(*ast.IdentExpr); ok {
			nsName := ns.Name
			// Local variables shadow namespaces for dot calls
			if !g.isDeclared(nsName) {
				// Rugo stdlib module call
				if g.imports[nsName] {
					if goFunc, ok := modules.LookupFunc(nsName, dot.Field); ok {
						return GoCallExpr{Func: goFunc, Args: goArgs}, nil
					}
					return nil, fmt.Errorf("unknown function %s.%s in module %q", nsName, dot.Field, nsName)
				}
				// Go bridge call — render args to strings for generateGoBridgeCall
				if pkg, ok := gobridge.PackageForNS(nsName, g.goImports); ok {
					if sig, ok := gobridge.Lookup(pkg, dot.Field); ok {
						if !sig.Variadic && len(e.Args) != len(sig.Params) {
							return nil, argCountError(nsName+"."+dot.Field, len(e.Args), len(sig.Params))
						}
						strArgs := make([]string, len(goArgs))
						for i, a := range goArgs {
							strArgs[i] = pr.exprStr(a)
						}
						return GoRawExpr{Code: g.generateGoBridgeCall(pkg, sig, strArgs, nsName+"."+dot.Field)}, nil
					}
					return nil, fmt.Errorf("unknown function %s.%s in Go bridge package %q", nsName, dot.Field, pkg)
				}
				// Known require namespace
				if g.namespaces[nsName] {
					if strings.HasPrefix(dot.Field, "_") {
						return nil, fmt.Errorf("'%s' is private to module '%s'", dot.Field, nsName)
					}
					nsKey := nsName + "." + dot.Field
					if expected, ok := g.funcDefs[nsKey]; ok {
						if len(e.Args) < expected.Min || len(e.Args) > expected.Max {
							return nil, arityCountError(nsName+"."+dot.Field, len(e.Args), expected)
						}
						if expected.HasDefaults {
							return GoCallExpr{Func: fmt.Sprintf("rugons_%s_%s", nsName, dot.Field), Args: goArgs}, nil
						}
					}
					typedArgs := g.typedCallExprs(nsKey, goArgs, e.Args)
					return GoCallExpr{Func: fmt.Sprintf("rugons_%s_%s", nsName, dot.Field), Args: typedArgs}, nil
				}
			}
			// Not a known namespace or shadowed by variable — dispatch via generic DotCall
			argStrs := make([]string, len(goArgs))
			for i, a := range goArgs {
				argStrs[i] = pr.exprStr(a)
			}
			argStr := strings.Join(argStrs, ", ")
			return GoRawExpr{Code: fmt.Sprintf("rugo_dot_call(%s, %q, %s)", nsName, dot.Field, argStr)}, nil
		}
		// Non-ident object: e.g. tasks[i].wait(n), q.push(val)
		obj, oerr := g.buildExpr(dot.Object)
		if oerr != nil {
			return nil, oerr
		}
		argStrs := make([]string, len(goArgs))
		for i, a := range goArgs {
			argStrs[i] = pr.exprStr(a)
		}
		argStr := strings.Join(argStrs, ", ")
		return GoRawExpr{Code: fmt.Sprintf("rugo_dot_call(%s, %q, %s)", pr.exprStr(obj), dot.Field, argStr)}, nil
	}

	// Check for built-in functions (globals)
	if ident, ok := e.Func.(*ast.IdentExpr); ok {
		boxed := g.boxedExprs(goArgs, e.Args)
		switch ident.Name {
		case "puts":
			return GoCallExpr{Func: "rugo_puts", Args: boxed}, nil
		case "print":
			return GoCallExpr{Func: "rugo_print", Args: boxed}, nil
		case "__shell__":
			return GoCallExpr{Func: "rugo_shell", Args: goArgs}, nil
		case "__capture__":
			return GoCallExpr{Func: "rugo_capture", Args: goArgs}, nil
		case "__pipe_shell__":
			return GoCallExpr{Func: "rugo_pipe_shell", Args: goArgs}, nil
		case "len":
			call := GoCallExpr{Func: "rugo_len", Args: boxed}
			if g.exprType(e) == TypeInt {
				return GoTypeAssert{Value: call, Type: "int"}, nil
			}
			return call, nil
		case "append":
			return GoCallExpr{Func: "rugo_append", Args: boxed}, nil
		case "raise":
			return GoCallExpr{Func: "rugo_raise", Args: boxed}, nil
		case "exit":
			return GoCallExpr{Func: "rugo_exit", Args: boxed}, nil
		case "type_of":
			if len(e.Args) != 1 {
				return nil, fmt.Errorf("type_of expects 1 argument, got %d", len(e.Args))
			}
			return GoCallExpr{Func: "rugo_type_of", Args: boxed}, nil
		case "range":
			if len(e.Args) < 1 || len(e.Args) > 2 {
				return nil, fmt.Errorf("range expects 1 or 2 arguments, got %d", len(e.Args))
			}
			return GoCallExpr{Func: "rugo_range", Args: boxed}, nil
		default:
			// Sibling function call within a namespace
			if g.currentFunc != nil && g.currentFunc.Namespace != "" {
				nsKey := g.currentFunc.Namespace + "." + ident.Name
				if expected, ok := g.funcDefs[nsKey]; ok {
					if len(e.Args) < expected.Min || len(e.Args) > expected.Max {
						return nil, arityCountError(ident.Name, len(e.Args), expected)
					}
					if expected.HasDefaults {
						return GoCallExpr{Func: fmt.Sprintf("rugons_%s_%s", g.currentFunc.Namespace, ident.Name), Args: goArgs}, nil
					}
					typedArgs := g.typedCallExprs(nsKey, goArgs, e.Args)
					return GoCallExpr{Func: fmt.Sprintf("rugons_%s_%s", g.currentFunc.Namespace, ident.Name), Args: typedArgs}, nil
				}
			}
			// User-defined function — validate argument count
			if expected, ok := g.funcDefs[ident.Name]; ok {
				if len(e.Args) < expected.Min || len(e.Args) > expected.Max {
					return nil, arityCountError(ident.Name, len(e.Args), expected)
				}
				if expected.HasDefaults {
					return GoCallExpr{Func: fmt.Sprintf("rugofn_%s", ident.Name), Args: goArgs}, nil
				}
				typedArgs := g.typedCallExprs(ident.Name, goArgs, e.Args)
				return GoCallExpr{Func: fmt.Sprintf("rugofn_%s", ident.Name), Args: typedArgs}, nil
			}
			// Lambda variable call — dynamic dispatch via type assertion
			if g.isDeclared(ident.Name) {
				argStrs := make([]string, len(goArgs))
				for i, a := range goArgs {
					argStrs[i] = pr.exprStr(a)
				}
				argStr := strings.Join(argStrs, ", ")
				return GoRawExpr{Code: fmt.Sprintf("%s.(func(...interface{}) interface{})(%s)", ident.Name, argStr)}, nil
			}
			typedArgs := g.typedCallExprs(ident.Name, goArgs, e.Args)
			return GoCallExpr{Func: fmt.Sprintf("rugofn_%s", ident.Name), Args: typedArgs}, nil
		}
	}

	// Dynamic call (shouldn't happen in v1 but handle gracefully)
	funcExpr, err := g.buildExpr(e.Func)
	if err != nil {
		return nil, err
	}
	argStrs := make([]string, len(goArgs))
	for i, a := range goArgs {
		argStrs[i] = pr.exprStr(a)
	}
	argStr := strings.Join(argStrs, ", ")
	return GoRawExpr{Code: fmt.Sprintf("%s.(%s)(%s)", pr.exprStr(funcExpr), "func(...interface{}) interface{}", argStr)}, nil
}

func (g *codeGen) buildFnExpr(e *ast.FnExpr) (GoExpr, error) {
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
		return nil, berr
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
			defaultExpr, derr := g.buildExpr(p.Default)
			if derr != nil {
				return nil, derr
			}
			pr := &goPrinter{}
			preamble = append(preamble, GoVarStmt{Name: p.Name, Type: "interface{}"})
			preamble = append(preamble, GoRawStmt{Code: fmt.Sprintf("if len(_args) > %d { %s = _args[%d] } else { %s = %s }", i, p.Name, i, p.Name, pr.exprStr(defaultExpr))})
		} else {
			preamble = append(preamble, GoVarStmt{Name: p.Name, Type: "interface{}"})
			preamble = append(preamble, GoRawStmt{Code: fmt.Sprintf("if len(_args) > %d { %s = _args[%d] }", i, p.Name, i)})
		}
		preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", p.Name)}})
	}

	// Assemble full body
	var fullBody []GoStmt
	fullBody = append(fullBody, preamble...)
	fullBody = append(fullBody, bodyStmts...)
	fullBody = append(fullBody, GoReturnStmt{Value: GoRawExpr{Code: "nil"}})

	return GoLambdaExpr{Body: fullBody}, nil
}
