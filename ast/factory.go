package ast

// Factory centralizes AST node creation for transform passes.
// It ensures consistent construction and provides a hook point for
// future enhancements like source position propagation.
type Factory struct{}

// NewFactory returns a new Factory.
func NewFactory() *Factory { return &Factory{} }

// --- Lowered concurrency nodes ---

// LoweredSpawn creates a LoweredSpawnExpr with the given body and optional result expression.
func (f *Factory) LoweredSpawn(body []Statement, resultExpr Expr) *LoweredSpawnExpr {
	return &LoweredSpawnExpr{Body: body, ResultExpr: resultExpr}
}

// LoweredParallel creates a LoweredParallelExpr with pre-categorized branches.
func (f *Factory) LoweredParallel(branches []ParallelBranch) *LoweredParallelExpr {
	return &LoweredParallelExpr{Branches: branches}
}

// LoweredTry creates a LoweredTryExpr with the given fields.
func (f *Factory) LoweredTry(expr Expr, errVar string, handler []Statement, resultExpr Expr) *LoweredTryExpr {
	return &LoweredTryExpr{Expr: expr, ErrVar: errVar, Handler: handler, ResultExpr: resultExpr}
}

// ParallelBranchExpr creates a ParallelBranch for a single expression.
func (f *Factory) ParallelBranchExpr(expr Expr, index int) ParallelBranch {
	return ParallelBranch{Expr: expr, Index: index}
}

// ParallelBranchStmts creates a ParallelBranch for a statement block.
func (f *Factory) ParallelBranchStmts(stmts []Statement, index int) ParallelBranch {
	return ParallelBranch{Stmts: stmts, Index: index}
}

// --- Copy helpers ---

// ProgramFrom creates a new Program copying metadata from src with new statements.
func (f *Factory) ProgramFrom(src *Program, stmts []Statement) *Program {
	return &Program{
		Statements: stmts,
		SourceFile: src.SourceFile,
		RawSource:  src.RawSource,
		Structs:    src.Structs,
	}
}

// FuncDefWithBody creates a shallow copy of a FuncDef with a new body.
func (f *Factory) FuncDefWithBody(src *FuncDef, body []Statement) *FuncDef {
	cp := *src
	cp.Body = body
	return &cp
}

// IfStmtWithBranches creates a shallow copy of an IfStmt with new branches.
func (f *Factory) IfStmtWithBranches(src *IfStmt, body, elseBody []Statement, elsifs []ElsifClause) *IfStmt {
	cp := *src
	cp.Body = body
	cp.ElseBody = elseBody
	cp.ElsifClauses = elsifs
	return &cp
}

// FnExprWithBody creates a shallow copy of a FnExpr with a new body.
func (f *Factory) FnExprWithBody(src *FnExpr, body []Statement) *FnExpr {
	return &FnExpr{Params: src.Params, Body: body}
}

// --- Implicit return nodes ---

// ImplicitReturn creates an ImplicitReturnStmt for function/lambda bodies.
func (f *Factory) ImplicitReturn(value Expr, line int) *ImplicitReturnStmt {
	return &ImplicitReturnStmt{BaseStmt: BaseStmt{SourceLine: line}, Value: value}
}

// TryResult creates a TryResultStmt for try/or handler results.
func (f *Factory) TryResult(value Expr, line int) *TryResultStmt {
	return &TryResultStmt{BaseStmt: BaseStmt{SourceLine: line}, Value: value}
}

// SpawnReturn creates a SpawnReturnStmt for return statements inside spawn blocks.
func (f *Factory) SpawnReturn(value Expr, line int) *SpawnReturnStmt {
	return &SpawnReturnStmt{BaseStmt: BaseStmt{SourceLine: line}, Value: value}
}

// TryHandlerReturn creates a TryHandlerReturnStmt for return statements inside try handlers.
func (f *Factory) TryHandlerReturn(value Expr, line int) *TryHandlerReturnStmt {
	return &TryHandlerReturnStmt{BaseStmt: BaseStmt{SourceLine: line}, Value: value}
}
