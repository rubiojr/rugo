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
		ExprTypes:   make(map[ast.Expr]RugoType),
		FuncTypes:   make(map[string]*FuncTypeInfo),
		VarTypes:    make(map[string]map[string]RugoType),
		VarUseTypes: make(map[ast.Expr]RugoType),
	}

	// Collect all function definitions (skip duplicates — codegen validates them).
	var funcs []*ast.FuncDef
	var topStmts []ast.Statement
	for _, s := range prog.Statements {
		switch st := s.(type) {
		case *ast.FuncDef:
			key := funcKey(st)
			if _, exists := ti.FuncTypes[key]; exists {
				continue // duplicate — codegen will report the error
			}
			funcs = append(funcs, st)
			fti := &FuncTypeInfo{
				ParamTypes:    make([]RugoType, len(st.Params)),
				AnnotatedArgs: make([]bool, len(st.Params)),
				ReturnType:    TypeUnknown,
				HasDefaults:   ast.HasDefaults(st.Params),
			}
			// Seed param types from annotations.
			for i, p := range st.Params {
				if p.TypeAnnot == "" {
					continue
				}
				if t, ok := ParseTypeAnnotation(p.TypeAnnot); ok {
					fti.ParamTypes[i] = t
					fti.AnnotatedArgs[i] = true
				}
			}
			// Seed return type from annotation.
			if st.ReturnType != "" {
				if t, ok := ParseTypeAnnotation(st.ReturnType); ok {
					fti.ReturnType = t
					fti.AnnotatedReturn = true
				}
			}
			ti.FuncTypes[key] = fti
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
//
// Two parallel maps:
//   - vars:    "storage union" — accumulates types via unify on every set.
//              Used by codegen to decide concrete vs. interface{} storage.
//   - current: "flow-sensitive" — replaces on every set, reflecting the
//              variable's type at the current program point. Used by
//              callers that need per-use precision (Tier 3).
//
// Both layers are cloned and re-merged identically across control-flow
// branch joins; they only diverge within straight-line code where the
// same variable is reassigned to a different concrete type.
type typeScope struct {
	vars    map[string]RugoType
	current map[string]RugoType
	parent  *typeScope
}

func newTypeScope(parent *typeScope) *typeScope {
	return &typeScope{
		vars:    make(map[string]RugoType),
		current: make(map[string]RugoType),
		parent:  parent,
	}
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

// currentGet returns the flow-sensitive type of name at this point, walking
// the parent chain. Used by callers (e.g., Ident reads) that want per-use
// precision instead of the conservative storage union.
func (s *typeScope) currentGet(name string) RugoType {
	if t, ok := s.current[name]; ok {
		return t
	}
	if s.parent != nil {
		return s.parent.currentGet(name)
	}
	return TypeDynamic
}

func (s *typeScope) set(name string, t RugoType) {
	if existing, ok := s.vars[name]; ok {
		// Storage union: unify across all assignments so codegen sees the
		// full set of types this variable can ever hold.
		s.vars[name] = unifyTypes(existing, t)
	} else {
		s.vars[name] = t
	}
	// Flow layer: replace, so per-use reads see the latest assignment.
	s.current[name] = t
}

func cloneTypeVars(src map[string]RugoType) map[string]RugoType {
	dst := make(map[string]RugoType, len(src))
	for name, t := range src {
		dst[name] = t
	}
	return dst
}

func inferIfStmt(ti *TypeInfo, scope *typeScope, st *ast.IfStmt) {
	ti.ExprTypes[st.Condition] = inferExpr(ti, scope, st.Condition)

	baseVars := cloneTypeVars(scope.vars)
	baseCurrent := cloneTypeVars(scope.current)
	newBranchScope := func() *typeScope {
		branchScope := newTypeScope(scope.parent)
		branchScope.vars = cloneTypeVars(baseVars)
		branchScope.current = cloneTypeVars(baseCurrent)
		return branchScope
	}

	var branchScopes []*typeScope

	bodyScope := newBranchScope()
	for _, s := range st.Body {
		inferStmt(ti, bodyScope, s)
	}
	branchScopes = append(branchScopes, bodyScope)

	for _, clause := range st.ElsifClauses {
		ti.ExprTypes[clause.Condition] = inferExpr(ti, scope, clause.Condition)
		clauseScope := newBranchScope()
		for _, s := range clause.Body {
			inferStmt(ti, clauseScope, s)
		}
		branchScopes = append(branchScopes, clauseScope)
	}

	if len(st.ElseBody) > 0 {
		elseScope := newBranchScope()
		for _, s := range st.ElseBody {
			inferStmt(ti, elseScope, s)
		}
		branchScopes = append(branchScopes, elseScope)
	} else {
		// No else branch means assignments may be skipped entirely.
		branchScopes = append(branchScopes, newBranchScope())
	}

	joinBranchScopes(scope, baseVars, baseCurrent, branchScopes)
}

// joinBranchScopes merges per-branch typeScopes into scope, accounting for
// both the storage-union (vars) and flow-sensitive (current) layers.
//
// A branch that does not assign to a name contributes to the "untouched"
// path: it brings the pre-fork base value of the name into the union, and
// — if the name had no pre-fork binding — contributes TypeNil to model
// the absence of an assignment on that path.
//
// For loops, callers should include an explicit empty branch in
// branchScopes to model "loop body may not have executed".
func joinBranchScopes(scope *typeScope, baseVars, baseCurrent map[string]RugoType, branchScopes []*typeScope) {
	names := make(map[string]bool)
	for name := range baseVars {
		names[name] = true
	}
	for name := range baseCurrent {
		names[name] = true
	}
	for _, branchScope := range branchScopes {
		for name := range branchScope.vars {
			names[name] = true
		}
		for name := range branchScope.current {
			names[name] = true
		}
	}

	for name := range names {
		baseType, existed := baseVars[name]
		mergedType := TypeUnknown
		if existed {
			mergedType = baseType
		}
		assignedInAllPaths := true
		for _, branchScope := range branchScopes {
			t, ok := branchScope.vars[name]
			if !ok {
				assignedInAllPaths = false
				continue
			}
			mergedType = unifyTypes(mergedType, t)
		}
		if !existed && !assignedInAllPaths {
			mergedType = unifyTypes(mergedType, TypeNil)
		}
		if mergedType != TypeUnknown {
			scope.vars[name] = mergedType
		}

		// Merge the flow-sensitive layer in parallel. If any branch
		// leaves the variable untouched, its baseline current type (or
		// TypeNil if it was never bound) participates in the union.
		baseCur, baseCurExisted := baseCurrent[name]
		mergedCur := TypeUnknown
		curAssignedInAll := true
		for _, branchScope := range branchScopes {
			t, ok := branchScope.current[name]
			if !ok {
				curAssignedInAll = false
				if baseCurExisted {
					mergedCur = unifyTypes(mergedCur, baseCur)
				}
				continue
			}
			mergedCur = unifyTypes(mergedCur, t)
		}
		if !baseCurExisted && !curAssignedInAll {
			mergedCur = unifyTypes(mergedCur, TypeNil)
		}
		if mergedCur != TypeUnknown {
			scope.current[name] = mergedCur
		}
	}
}

// mergeLoopCurrent models "loop body may not have executed" by joining the
// post-loop flow-sensitive types with the pre-loop snapshot. The storage
// (vars) layer is not touched here because set() already unifies on every
// write — the union of all writes is preserved naturally.
func mergeLoopCurrent(scope *typeScope, baseCurrent map[string]RugoType) {
	for name, t := range scope.current {
		if base, ok := baseCurrent[name]; ok {
			scope.current[name] = unifyTypes(base, t)
		} else {
			scope.current[name] = unifyTypes(t, TypeNil)
		}
	}
}

// inferCaseStmt walks a case statement using branch-scope semantics so the
// per-clause assignments are joined at the merge point. When the case lacks
// an else clause, an empty branch is added to model the "no clause matched"
// path.
func inferCaseStmt(ti *TypeInfo, scope *typeScope, st *ast.CaseStmt) {
	inferExpr(ti, scope, st.Subject)
	ti.ExprTypes[st.Subject] = inferExpr(ti, scope, st.Subject)

	baseVars := cloneTypeVars(scope.vars)
	baseCurrent := cloneTypeVars(scope.current)
	newBranchScope := func() *typeScope {
		branchScope := newTypeScope(scope.parent)
		branchScope.vars = cloneTypeVars(baseVars)
		branchScope.current = cloneTypeVars(baseCurrent)
		return branchScope
	}

	var branchScopes []*typeScope

	for _, oc := range st.OfClauses {
		for _, v := range oc.Values {
			inferExpr(ti, scope, v)
			ti.ExprTypes[v] = inferExpr(ti, scope, v)
		}
		clauseScope := newBranchScope()
		if oc.ArrowExpr != nil {
			inferExpr(ti, clauseScope, oc.ArrowExpr)
		}
		for _, s := range oc.Body {
			inferStmt(ti, clauseScope, s)
		}
		branchScopes = append(branchScopes, clauseScope)
	}

	for _, clause := range st.ElsifClauses {
		inferExpr(ti, scope, clause.Condition)
		ti.ExprTypes[clause.Condition] = inferExpr(ti, scope, clause.Condition)
		clauseScope := newBranchScope()
		for _, s := range clause.Body {
			inferStmt(ti, clauseScope, s)
		}
		branchScopes = append(branchScopes, clauseScope)
	}

	if len(st.ElseBody) > 0 {
		elseScope := newBranchScope()
		for _, s := range st.ElseBody {
			inferStmt(ti, elseScope, s)
		}
		branchScopes = append(branchScopes, elseScope)
	} else {
		// No else clause means no `of`/elsif may have matched — model
		// the "no-match" path as an empty branch.
		branchScopes = append(branchScopes, newBranchScope())
	}

	joinBranchScopes(scope, baseVars, baseCurrent, branchScopes)
}

// inferCaseExpr is the expression-form counterpart to inferCaseStmt; it
// also branch-joins the per-clause writes so callers reading variables
// after a case-expression see the union of all paths.
func inferCaseExpr(ti *TypeInfo, scope *typeScope, ex *ast.CaseExpr) RugoType {
	inferExpr(ti, scope, ex.Subject)

	baseVars := cloneTypeVars(scope.vars)
	baseCurrent := cloneTypeVars(scope.current)
	newBranchScope := func() *typeScope {
		branchScope := newTypeScope(scope.parent)
		branchScope.vars = cloneTypeVars(baseVars)
		branchScope.current = cloneTypeVars(baseCurrent)
		return branchScope
	}

	var branchScopes []*typeScope

	for _, oc := range ex.OfClauses {
		for _, v := range oc.Values {
			inferExpr(ti, scope, v)
		}
		clauseScope := newBranchScope()
		if oc.ArrowExpr != nil {
			inferExpr(ti, clauseScope, oc.ArrowExpr)
		}
		for _, s := range oc.Body {
			inferStmt(ti, clauseScope, s)
		}
		branchScopes = append(branchScopes, clauseScope)
	}

	for _, clause := range ex.ElsifClauses {
		inferExpr(ti, scope, clause.Condition)
		clauseScope := newBranchScope()
		for _, s := range clause.Body {
			inferStmt(ti, clauseScope, s)
		}
		branchScopes = append(branchScopes, clauseScope)
	}

	if len(ex.ElseBody) > 0 {
		elseScope := newBranchScope()
		for _, s := range ex.ElseBody {
			inferStmt(ti, elseScope, s)
		}
		branchScopes = append(branchScopes, elseScope)
	} else {
		branchScopes = append(branchScopes, newBranchScope())
	}

	joinBranchScopes(scope, baseVars, baseCurrent, branchScopes)
	return TypeDynamic
}

// inferFunc infers types for a single function.
func inferFunc(ti *TypeInfo, f *ast.FuncDef) {
	fti := ti.FuncTypes[funcKey(f)]
	scope := newTypeScope(nil)

	// Functions with default params use variadic signature (_args ...interface{}),
	// so all params are interface{} at runtime — force them to TypeDynamic.
	// The return value is also boxed through interface{}, so we cannot honor a
	// return-type annotation either. Annotations are still parsed and recorded
	// for documentation, but the runtime shape forces dynamic.
	if ast.HasDefaults(f.Params) {
		for i := range fti.ParamTypes {
			fti.ParamTypes[i] = TypeDynamic
		}
		fti.ReturnType = TypeDynamic
		fti.AnnotatedReturn = false
	}

	// Bind parameters to their current inferred types.
	for i, p := range f.Params {
		scope.set(p.Name, fti.ParamTypes[i])
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
	// s = str.trim(s)), or to a type incompatible with its inferred shape
	// (e.g. `a = "x"; a = 42`), its var type will have been widened to
	// TypeDynamic (legacy) or a union (post-bitmask). Either way, the
	// narrow Go signature is no longer safe -- widen ParamTypes to
	// TypeDynamic so the Go declaration uses interface{}.
	//
	// Annotated params are NOT widened — the user has asserted the type,
	// so we trust them. The conflict detector below catches mismatches.
	for i, p := range f.Params {
		if i < len(fti.AnnotatedArgs) && fti.AnnotatedArgs[i] {
			continue
		}
		if !fti.ParamTypes[i].IsTyped() {
			continue
		}
		inferredVar := scope.get(p.Name)
		if inferredVar == TypeDynamic || inferredVar.IsUnion() {
			fti.ParamTypes[i] = TypeDynamic
		}
	}

	// Infer return type from collected returns.
	retType := TypeUnknown
	hasUnresolved := false
	for _, rt := range returnTypes {
		if rt == TypeUnknown {
			hasUnresolved = true
			continue // Skip unresolved returns.
		}
		retType = unifyTypes(retType, rt)
	}
	// Functions without explicit return → nil (dynamic).
	if retType == TypeUnknown && len(returnTypes) == 0 {
		retType = TypeDynamic
	}
	// If some returns resolved to a concrete type but others remain
	// unresolved, the unresolved paths will emit interface{} at codegen.
	// Widen to TypeDynamic to avoid Go type mismatches.
	if hasUnresolved && retType.IsTyped() {
		retType = TypeDynamic
	}

	if !fti.AnnotatedReturn && retType != fti.ReturnType {
		fti.ReturnType = retType
	}
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
		inferIfStmt(ti, scope, st)

	case *ast.CaseStmt:
		inferCaseStmt(ti, scope, st)

	case *ast.WhileStmt:
		inferExpr(ti, scope, st.Condition)
		ti.ExprTypes[st.Condition] = inferExpr(ti, scope, st.Condition)
		baseCurrentWhile := cloneTypeVars(scope.current)
		// Infer body twice: the first pass widens variable types via
		// unify-on-set (vars layer) and seeds VarUseTypes for reads; the
		// second pass propagates the widening into expression types and
		// unifies VarUseTypes with all earlier observations.
		for pass := 0; pass < 2; pass++ {
			for _, s := range st.Body {
				inferStmt(ti, scope, s)
			}
		}
		// Model "loop body may not have executed" for the flow-sensitive
		// layer by unioning back with the pre-loop snapshot.
		mergeLoopCurrent(scope, baseCurrentWhile)

	case *ast.ForStmt:
		inferExpr(ti, scope, st.Collection)
		// Infer loop variable type from collection.
		loopVarType := inferForVarType(ti, scope, st.Collection)
		scope.set(st.Var, loopVarType)
		if st.IndexVar != "" {
			scope.set(st.IndexVar, loopVarType)
		}
		baseCurrentFor := cloneTypeVars(scope.current)
		// Infer body twice: the first pass may widen variable types
		// (e.g. lines = lines + dynamic_var), and the second pass
		// ensures expression types reflect the widened variables.
		for pass := 0; pass < 2; pass++ {
			for _, s := range st.Body {
				inferStmt(ti, scope, s)
			}
		}
		mergeLoopCurrent(scope, baseCurrentFor)

	case *ast.ReturnStmt:
		if st.Value != nil {
			inferExpr(ti, scope, st.Value)
		}

	case *ast.ImplicitReturnStmt:
		inferExpr(ti, scope, st.Value)

	case *ast.TryResultStmt:
		inferExpr(ti, scope, st.Value)

	case *ast.SpawnReturnStmt:
		if st.Value != nil {
			inferExpr(ti, scope, st.Value)
		}

	case *ast.TryHandlerReturnStmt:
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
	case *ast.ImplicitReturnStmt:
		t := inferExpr(ti, scope, st.Value)
		*out = append(*out, t)
	case *ast.SpawnReturnStmt, *ast.TryHandlerReturnStmt, *ast.TryResultStmt:
		// These are exits from nested concurrency contexts, not function returns.
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
	case *ast.CaseStmt:
		for _, oc := range st.OfClauses {
			for _, s := range oc.Body {
				collectReturns(ti, scope, s, out)
			}
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
		// Record the flow-sensitive (per-use) type for downstream
		// consumers (Tier 3 callsite/return checks). The function return
		// value remains the conservative storage-union type so that
		// ExprTypes drives codegen decisions consistently.
		//
		// Unify into any pre-existing entry so that re-walks (e.g. a
		// loop body inferred twice) accumulate the union of all observed
		// types instead of merely keeping the last pass's view.
		cur := scope.currentGet(ex.Name)
		if prev, hadPrev := ti.VarUseTypes[e]; hadPrev {
			cur = unifyTypes(prev, cur)
		}
		ti.VarUseTypes[e] = cur
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

	case *ast.LoweredTryExpr:
		inferExpr(ti, scope, ex.Expr)
		for _, s := range ex.Handler {
			inferStmt(ti, scope, s)
		}
		if ex.ResultExpr != nil {
			inferExpr(ti, scope, ex.ResultExpr)
		}
		return TypeDynamic

	case *ast.LoweredSpawnExpr:
		for _, s := range ex.Body {
			inferStmt(ti, scope, s)
		}
		if ex.ResultExpr != nil {
			inferExpr(ti, scope, ex.ResultExpr)
		}
		return TypeDynamic

	case *ast.LoweredParallelExpr:
		for _, br := range ex.Branches {
			if br.Expr != nil {
				inferExpr(ti, scope, br.Expr)
			}
			for _, s := range br.Stmts {
				inferStmt(ti, scope, s)
			}
		}
		return TypeDynamic

	case *ast.CaseExpr:
		return inferCaseExpr(ti, scope, ex)

	case *ast.FnExpr:
		// Walk lambda body using the parent scope so that assignments
		// to captured variables (e.g. result = result + v) widen
		// their types in the enclosing scope.
		childScope := newTypeScope(scope)
		for _, p := range ex.Params {
			childScope.set(p.Name, TypeDynamic)
		}
		for _, s := range ex.Body {
			inferStmt(ti, childScope, s)
		}
		// Propagate any widened captured variables back to the parent scope.
		for name, childType := range childScope.vars {
			if _, isParam := childScope.vars[name]; isParam {
				// Skip lambda params — check if it's actually a param.
				isLambdaParam := false
				for _, p := range ex.Params {
					if p.Name == name {
						isLambdaParam = true
						break
					}
				}
				if isLambdaParam {
					continue
				}
			}
			// This variable was assigned inside the lambda — if the parent
			// scope knows about it, widen its type.
			if parentType := scope.get(name); parentType != TypeDynamic {
				scope.set(name, unifyTypes(parentType, childType))
			}
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
			// Don't propagate param types for functions with defaults —
			// they use variadic signature so all params are interface{}.
			if !fti.HasDefaults {
				// Only propagate resolved types to avoid poisoning with Dynamic.
				// Skip annotated params: those are user assertions and must
				// remain sticky -- a mismatched call site is the caller's
				// bug, not a reason to widen the param to a union. Without
				// this guard, post-union unification would pollute the param
				// (e.g. `int` unioned with `array` from `f([1,2,3])` becomes
				// `int|array`), and the return-type check would fire on the
				// polluted union before the call-site check could run.
				for i, at := range argTypes {
					if i >= len(fti.ParamTypes) || !at.IsResolved() || at == TypeDynamic {
						continue
					}
					if i < len(fti.AnnotatedArgs) && fti.AnnotatedArgs[i] {
						continue
					}
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
				if !fti.HasDefaults {
					for i, at := range argTypes {
						if i >= len(fti.ParamTypes) || !at.IsResolved() || at == TypeDynamic {
							continue
						}
						if i < len(fti.AnnotatedArgs) && fti.AnnotatedArgs[i] {
							continue
						}
						fti.ParamTypes[i] = unifyTypes(fti.ParamTypes[i], at)
					}
				}
				return fti.ReturnType
			}
		}
	}

	return TypeDynamic
}

// inferForVarType returns the loop variable type for a for-in collection.
// Integer literals and range() calls produce TypeInt loop variables;
// all other collections (arrays, hashes, variables) remain TypeDynamic.
func inferForVarType(ti *TypeInfo, scope *typeScope, coll ast.Expr) RugoType {
	// for i in 100
	if _, ok := coll.(*ast.IntLiteral); ok {
		return TypeInt
	}
	// for i in someIntVar
	if ident, ok := coll.(*ast.IdentExpr); ok {
		if scope.get(ident.Name) == TypeInt {
			return TypeInt
		}
	}
	// for i in range(...)
	if call, ok := coll.(*ast.CallExpr); ok {
		if fn, ok := call.Func.(*ast.IdentExpr); ok && fn.Name == "range" {
			return TypeInt
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
		annot := make([]bool, len(v.AnnotatedArgs))
		copy(annot, v.AnnotatedArgs)
		snap[k] = &FuncTypeInfo{
			ParamTypes:      params,
			AnnotatedArgs:   annot,
			ReturnType:      v.ReturnType,
			AnnotatedReturn: v.AnnotatedReturn,
			HasDefaults:     v.HasDefaults,
		}
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
