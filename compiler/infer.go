package compiler

import (
	"fmt"

	"github.com/rubiojr/rugo/ast"
)

// Infer runs type inference on a parsed program, returning type annotations
// for expressions and function signatures. The inference is conservative:
// anything that can't be proven typed remains TypeDynamic (interface{}).
func Infer(prog *ast.Program) *TypeInfo {
	ti := &TypeInfo{
		ExprTypes: make(map[ast.Expr]RugoType),
		FuncTypes: make(map[string]*FuncTypeInfo),
		VarTypes:  make(map[string]map[string]RugoType),
	}

	// Collect all function definitions.
	var funcs []*ast.FuncDef
	var topStmts []ast.Statement
	for _, s := range prog.Statements {
		switch st := s.(type) {
		case *ast.FuncDef:
			funcs = append(funcs, st)
			ti.FuncTypes[funcKey(st)] = &FuncTypeInfo{
				ParamTypes: make([]RugoType, len(st.Params)),
				ReturnType: TypeUnknown,
			}
		default:
			topStmts = append(topStmts, s)
		}
	}

	// Fixed-point iteration: infer until stable.
	// Functions may call each other, so we iterate until no types change.
	for round := 0; round < 10; round++ {
		// Snapshot function signatures to detect changes.
		snapshot := snapshotFuncTypes(ti.FuncTypes)

		for _, f := range funcs {
			inferFunc(ti, f)
		}

		// Infer top-level statements (including bench/test blocks that call functions).
		scope := newTypeScope(nil)
		for _, s := range topStmts {
			inferStmt(ti, scope, s)
		}
		ti.VarTypes[""] = scope.vars

		if funcTypesEqual(snapshot, ti.FuncTypes) {
			break
		}
	}

	return ti
}

// funcKey returns the inference key for a function definition.
func funcKey(f *ast.FuncDef) string {
	if f.Namespace != "" {
		return f.Namespace + "." + f.Name
	}
	return f.Name
}

// typeScope tracks variable types within a scope.
type typeScope struct {
	vars   map[string]RugoType
	parent *typeScope
}

func newTypeScope(parent *typeScope) *typeScope {
	return &typeScope{vars: make(map[string]RugoType), parent: parent}
}

func (s *typeScope) get(name string) RugoType {
	if t, ok := s.vars[name]; ok {
		return t
	}
	if s.parent != nil {
		return s.parent.get(name)
	}
	return TypeDynamic
}

func (s *typeScope) set(name string, t RugoType) {
	if existing, ok := s.vars[name]; ok {
		// Variable reassigned — unify types.
		s.vars[name] = unifyTypes(existing, t)
	} else {
		s.vars[name] = t
	}
}

// inferFunc infers types for a single function.
func inferFunc(ti *TypeInfo, f *ast.FuncDef) {
	fti := ti.FuncTypes[funcKey(f)]
	scope := newTypeScope(nil)

	// Bind parameters to their current inferred types.
	for i, p := range f.Params {
		scope.set(p, fti.ParamTypes[i])
	}

	// Walk the function body.
	var returnTypes []RugoType
	for _, s := range f.Body {
		inferStmt(ti, scope, s)
		collectReturns(ti, scope, s, &returnTypes)
	}

	// Record variable types for this function scope.
	ti.VarTypes[funcKey(f)] = scope.vars

	// If a parameter was reassigned from a dynamic expression (e.g.
	// s = str.trim(s)), its var type will have been widened to dynamic.
	// Widen ParamTypes to match so the Go declaration uses interface{}.
	for i, p := range f.Params {
		if fti.ParamTypes[i].IsTyped() && scope.get(p) == TypeDynamic {
			fti.ParamTypes[i] = TypeDynamic
		}
	}

	// Infer return type from collected returns.
	retType := TypeUnknown
	for _, rt := range returnTypes {
		if rt == TypeUnknown {
			continue // Skip unresolved returns.
		}
		retType = unifyTypes(retType, rt)
	}
	// Functions without explicit return → nil (dynamic).
	if retType == TypeUnknown && len(returnTypes) == 0 {
		retType = TypeDynamic
	}

	changed := false
	if retType != fti.ReturnType {
		fti.ReturnType = retType
		changed = true
	}
	_ = changed
}

// inferStmt infers types within a statement, updating the scope.
func inferStmt(ti *TypeInfo, scope *typeScope, s ast.Statement) {
	switch st := s.(type) {
	case *ast.AssignStmt:
		t := inferExpr(ti, scope, st.Value)
		scope.set(st.Target, t)

	case *ast.ExprStmt:
		inferExpr(ti, scope, st.Expression)

	case *ast.IfStmt:
		inferExpr(ti, scope, st.Condition)
		ti.ExprTypes[st.Condition] = inferExpr(ti, scope, st.Condition)
		for _, s := range st.Body {
			inferStmt(ti, scope, s)
		}
		for _, clause := range st.ElsifClauses {
			inferExpr(ti, scope, clause.Condition)
			ti.ExprTypes[clause.Condition] = inferExpr(ti, scope, clause.Condition)
			for _, s := range clause.Body {
				inferStmt(ti, scope, s)
			}
		}
		for _, s := range st.ElseBody {
			inferStmt(ti, scope, s)
		}

	case *ast.WhileStmt:
		inferExpr(ti, scope, st.Condition)
		ti.ExprTypes[st.Condition] = inferExpr(ti, scope, st.Condition)
		// Infer body twice: same rationale as ast.ForStmt — the first pass
		// may widen variable types that affect expression types.
		for pass := 0; pass < 2; pass++ {
			for _, s := range st.Body {
				inferStmt(ti, scope, s)
			}
		}

	case *ast.ForStmt:
		inferExpr(ti, scope, st.Collection)
		// For loop vars are dynamic (collection element type unknown).
		scope.set(st.Var, TypeDynamic)
		if st.IndexVar != "" {
			scope.set(st.IndexVar, TypeDynamic)
		}
		// Infer body twice: the first pass may widen variable types
		// (e.g. lines = lines + dynamic_var), and the second pass
		// ensures expression types reflect the widened variables.
		for pass := 0; pass < 2; pass++ {
			for _, s := range st.Body {
				inferStmt(ti, scope, s)
			}
		}

	case *ast.ReturnStmt:
		if st.Value != nil {
			inferExpr(ti, scope, st.Value)
		}

	case *ast.IndexAssignStmt:
		inferExpr(ti, scope, st.Object)
		inferExpr(ti, scope, st.Index)
		inferExpr(ti, scope, st.Value)

	case *ast.DotAssignStmt:
		inferExpr(ti, scope, st.Object)
		inferExpr(ti, scope, st.Value)

	case *ast.BenchDef:
		blockScope := newTypeScope(scope)
		for _, s := range st.Body {
			inferStmt(ti, blockScope, s)
		}
		ti.VarTypes[fmt.Sprintf("__bench_%p", st)] = blockScope.vars

	case *ast.TestDef:
		blockScope := newTypeScope(scope)
		for _, s := range st.Body {
			inferStmt(ti, blockScope, s)
		}
		ti.VarTypes[fmt.Sprintf("__test_%p", st)] = blockScope.vars
	}
}

// collectReturns gathers return types from a statement tree.
func collectReturns(ti *TypeInfo, scope *typeScope, s ast.Statement, out *[]RugoType) {
	switch st := s.(type) {
	case *ast.ReturnStmt:
		if st.Value != nil {
			t := inferExpr(ti, scope, st.Value)
			*out = append(*out, t)
		} else {
			*out = append(*out, TypeNil)
		}
	case *ast.IfStmt:
		for _, s := range st.Body {
			collectReturns(ti, scope, s, out)
		}
		for _, clause := range st.ElsifClauses {
			for _, s := range clause.Body {
				collectReturns(ti, scope, s, out)
			}
		}
		for _, s := range st.ElseBody {
			collectReturns(ti, scope, s, out)
		}
	case *ast.WhileStmt:
		for _, s := range st.Body {
			collectReturns(ti, scope, s, out)
		}
	case *ast.ForStmt:
		for _, s := range st.Body {
			collectReturns(ti, scope, s, out)
		}
	}
}

// inferExpr infers and records the type of an expression.
func inferExpr(ti *TypeInfo, scope *typeScope, e ast.Expr) RugoType {
	t := inferExprInner(ti, scope, e)
	ti.ExprTypes[e] = t
	return t
}

func inferExprInner(ti *TypeInfo, scope *typeScope, e ast.Expr) RugoType {
	switch ex := e.(type) {
	case *ast.IntLiteral:
		return TypeInt
	case *ast.FloatLiteral:
		return TypeFloat
	case *ast.StringLiteral:
		return TypeString
	case *ast.BoolLiteral:
		return TypeBool
	case *ast.NilLiteral:
		return TypeNil

	case *ast.IdentExpr:
		return scope.get(ex.Name)

	case *ast.BinaryExpr:
		left := inferExpr(ti, scope, ex.Left)
		right := inferExpr(ti, scope, ex.Right)
		return inferBinaryOp(ex.Op, left, right)

	case *ast.UnaryExpr:
		operand := inferExpr(ti, scope, ex.Operand)
		return inferUnaryOp(ex.Op, operand)

	case *ast.CallExpr:
		return inferCall(ti, scope, ex)

	case *ast.ArrayLiteral:
		for _, elem := range ex.Elements {
			inferExpr(ti, scope, elem)
		}
		return TypeArray

	case *ast.HashLiteral:
		for _, pair := range ex.Pairs {
			inferExpr(ti, scope, pair.Key)
			inferExpr(ti, scope, pair.Value)
		}
		return TypeHash

	case *ast.IndexExpr:
		inferExpr(ti, scope, ex.Object)
		inferExpr(ti, scope, ex.Index)
		return TypeDynamic // element type unknown

	case *ast.SliceExpr:
		inferExpr(ti, scope, ex.Object)
		inferExpr(ti, scope, ex.Start)
		inferExpr(ti, scope, ex.Length)
		return TypeDynamic

	case *ast.DotExpr:
		inferExpr(ti, scope, ex.Object)
		return TypeDynamic

	case *ast.TryExpr:
		inferExpr(ti, scope, ex.Expr)
		return TypeDynamic

	case *ast.SpawnExpr:
		// Walk the spawn body so expressions inside get typed.
		// Spawn shares the parent scope via Go closure.
		for _, s := range ex.Body {
			inferStmt(ti, scope, s)
		}
		return TypeDynamic

	case *ast.ParallelExpr:
		// Walk the parallel body so expressions inside get typed.
		for _, s := range ex.Body {
			inferStmt(ti, scope, s)
		}
		return TypeDynamic

	default:
		return TypeDynamic
	}
}

// inferBinaryOp returns the result type of a binary operation.
func inferBinaryOp(op string, left, right RugoType) RugoType {
	switch op {
	case "+":
		// string + anything → string (concatenation)
		if left == TypeString && right == TypeString {
			return TypeString
		}
		if left == TypeInt && right == TypeInt {
			return TypeInt
		}
		if left.IsNumeric() && right.IsNumeric() {
			return TypeFloat
		}
		// If either side is unknown, we can't resolve yet.
		if left == TypeUnknown || right == TypeUnknown {
			return TypeUnknown
		}
		return TypeDynamic

	case "-", "*":
		if left == TypeInt && right == TypeInt {
			return TypeInt
		}
		if left.IsNumeric() && right.IsNumeric() {
			return TypeFloat
		}
		if left == TypeUnknown || right == TypeUnknown {
			return TypeUnknown
		}
		return TypeDynamic

	case "/":
		if left == TypeInt && right == TypeInt {
			return TypeInt
		}
		if left.IsNumeric() && right.IsNumeric() {
			return TypeFloat
		}
		if left == TypeUnknown || right == TypeUnknown {
			return TypeUnknown
		}
		return TypeDynamic

	case "%":
		if left == TypeInt && right == TypeInt {
			return TypeInt
		}
		if left == TypeUnknown || right == TypeUnknown {
			return TypeUnknown
		}
		return TypeDynamic

	case "==", "!=":
		// TypeBool only when codegen emits native Go == / !=
		// (same typed type). Cross-type falls back to rugo_eq which returns interface{}.
		if left == right && left.IsTyped() {
			return TypeBool
		}
		if left == TypeUnknown || right == TypeUnknown {
			return TypeUnknown
		}
		return TypeDynamic

	case "<", ">", "<=", ">=":
		// TypeBool only when codegen emits native Go comparison
		// (same typed type that is numeric or string).
		if left == right && left.IsTyped() && (left.IsNumeric() || left == TypeString) {
			return TypeBool
		}
		if left == TypeUnknown || right == TypeUnknown {
			return TypeUnknown
		}
		return TypeDynamic

	case "&&", "||":
		if left == TypeBool && right == TypeBool {
			return TypeBool
		}
		if left == TypeUnknown || right == TypeUnknown {
			return TypeUnknown
		}
		return TypeDynamic

	default:
		return TypeDynamic
	}
}

// inferUnaryOp returns the result type of a unary operation.
func inferUnaryOp(op string, operand RugoType) RugoType {
	switch op {
	case "-":
		if operand == TypeInt {
			return TypeInt
		}
		if operand == TypeFloat {
			return TypeFloat
		}
		if operand == TypeUnknown {
			return TypeUnknown
		}
		return TypeDynamic
	case "!":
		if operand == TypeBool {
			return TypeBool
		}
		if operand == TypeUnknown {
			return TypeUnknown
		}
		return TypeDynamic
	default:
		return TypeDynamic
	}
}

// inferCall infers the return type of a function call and propagates
// argument types to function parameter inference.
func inferCall(ti *TypeInfo, scope *typeScope, e *ast.CallExpr) RugoType {
	// Infer argument types.
	argTypes := make([]RugoType, len(e.Args))
	for i, arg := range e.Args {
		argTypes[i] = inferExpr(ti, scope, arg)
	}

	// Check if this is a call to a user-defined function.
	if ident, ok := e.Func.(*ast.IdentExpr); ok {
		// Built-in functions return dynamic.
		switch ident.Name {
		case "puts", "print", "__shell__", "__capture__", "__pipe_shell__":
			return TypeDynamic
		case "len":
			return TypeInt
		case "append":
			return TypeArray
		}

		// User-defined function — propagate argument types.
		if fti, ok := ti.FuncTypes[ident.Name]; ok {
			// Only propagate resolved types to avoid poisoning with Dynamic.
			for i, at := range argTypes {
				if i < len(fti.ParamTypes) && at.IsResolved() && at != TypeDynamic {
					fti.ParamTypes[i] = unifyTypes(fti.ParamTypes[i], at)
				}
			}
			return fti.ReturnType
		}
	}

	// Namespace function call.
	if dot, ok := e.Func.(*ast.DotExpr); ok {
		if ns, ok := dot.Object.(*ast.IdentExpr); ok {
			key := ns.Name + "." + dot.Field
			if fti, ok := ti.FuncTypes[key]; ok {
				for i, at := range argTypes {
					if i < len(fti.ParamTypes) && at.IsResolved() && at != TypeDynamic {
						fti.ParamTypes[i] = unifyTypes(fti.ParamTypes[i], at)
					}
				}
				return fti.ReturnType
			}
		}
	}

	return TypeDynamic
}

// snapshotFuncTypes creates a deep copy of function type info for change detection.
func snapshotFuncTypes(m map[string]*FuncTypeInfo) map[string]*FuncTypeInfo {
	snap := make(map[string]*FuncTypeInfo, len(m))
	for k, v := range m {
		params := make([]RugoType, len(v.ParamTypes))
		copy(params, v.ParamTypes)
		snap[k] = &FuncTypeInfo{ParamTypes: params, ReturnType: v.ReturnType}
	}
	return snap
}

// funcTypesEqual returns true if two function type maps are identical.
func funcTypesEqual(a, b map[string]*FuncTypeInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if av.ReturnType != bv.ReturnType {
			return false
		}
		if len(av.ParamTypes) != len(bv.ParamTypes) {
			return false
		}
		for i := range av.ParamTypes {
			if av.ParamTypes[i] != bv.ParamTypes[i] {
				return false
			}
		}
	}
	return true
}
