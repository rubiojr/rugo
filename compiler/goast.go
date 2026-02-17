package compiler

// Go output AST types represent the structure of generated Go programs.
// Codegen builds a GoFile tree; the printer serializes it to Go source.

// --- Interfaces ---

// GoDecl is a top-level declaration (function, variable, raw code).
type GoDecl interface{ goDecl() }

// GoStmt is a statement inside a function body.
type GoStmt interface{ goStmt() }

// GoExpr is an expression (currently just raw strings wrapping exprString output).
type GoExpr interface{ goExpr() }

// --- File level ---

// GoFile represents a complete Go source file.
type GoFile struct {
	Package string
	Imports []GoImport
	Decls   []GoDecl // top-level declarations (vars, funcs, runtime)
	Init    []GoStmt // main() or test/bench harness body
}

// GoImport represents a single import.
type GoImport struct {
	Path  string
	Alias string // empty for default alias
}

// --- Declaration level ---

// GoVarDecl represents: var name type [= value]
type GoVarDecl struct {
	Name  string
	Type  string
	Value GoExpr // nil for uninitialized
}

func (GoVarDecl) goDecl() {}

// GoFuncDecl represents: func name(params) returnType { body }
type GoFuncDecl struct {
	Name   string
	Params []GoParam
	Return string   // empty for no return type
	Body   []GoStmt // nil for declaration-only (not used, but complete)
}

func (GoFuncDecl) goDecl() {}

// GoParam represents a function parameter.
type GoParam struct {
	Name string
	Type string
}

// GoRawDecl is an escape hatch for raw Go code at the declaration level
// (runtime templates, complex generated blocks).
type GoRawDecl struct {
	Code string
}

func (GoRawDecl) goDecl() {}

// --- Statement level ---

// GoExprStmt is an expression used as a statement.
type GoExprStmt struct {
	Expr GoExpr
}

func (GoExprStmt) goStmt() {}

// GoAssignStmt represents: target op value (e.g., x := expr, x = expr)
type GoAssignStmt struct {
	Target string // variable name
	Op     string // ":=" or "="
	Value  GoExpr
}

func (GoAssignStmt) goStmt() {}

// GoMultiAssignStmt represents: t1, t2 := expr
type GoMultiAssignStmt struct {
	Targets []string
	Op      string // ":=" or "="
	Value   GoExpr
}

func (GoMultiAssignStmt) goStmt() {}

// GoReturnStmt represents: return [expr]
type GoReturnStmt struct {
	Value GoExpr // nil for bare return
}

func (GoReturnStmt) goStmt() {}

// GoVarStmt represents: var name type [= value]
type GoVarStmt struct {
	Name  string
	Type  string
	Value GoExpr // nil for uninitialized
}

func (GoVarStmt) goStmt() {}

// GoIfStmt represents: if cond { body } [else if ...] [else { body }]
type GoIfStmt struct {
	Cond   GoExpr
	Body   []GoStmt
	ElseIf []GoElseIf
	Else   []GoStmt // nil for no else
}

func (GoIfStmt) goStmt() {}

// GoElseIf represents one else-if branch.
type GoElseIf struct {
	Cond GoExpr
	Body []GoStmt
}

// GoForStmt represents: for [init]; [cond]; [post] { body }
type GoForStmt struct {
	Init string // empty for while-style: for cond { }
	Cond string
	Post string
	Body []GoStmt
}

func (GoForStmt) goStmt() {}

// GoForRangeStmt represents: for key, value := range collection { body }
type GoForRangeStmt struct {
	Key        string // "_" if unused
	Value      string // empty for single-var range
	Collection GoExpr
	Body       []GoStmt
}

func (GoForRangeStmt) goStmt() {}

// GoSwitchStmt represents: switch [tag] { case ...: ... default: ... }
type GoSwitchStmt struct {
	Tag     GoExpr // nil for tagless switch
	Cases   []GoCase
	Default []GoStmt // nil for no default
}

func (GoSwitchStmt) goStmt() {}

// GoCase represents one case in a switch statement.
type GoCase struct {
	Values []GoExpr
	Body   []GoStmt
}

// GoDeferStmt represents: defer func() { body }()
type GoDeferStmt struct {
	Body []GoStmt
}

func (GoDeferStmt) goStmt() {}

// GoGoStmt represents: go func() { body }()
type GoGoStmt struct {
	Body []GoStmt
}

func (GoGoStmt) goStmt() {}

// GoBreakStmt represents: break
type GoBreakStmt struct{}

func (GoBreakStmt) goStmt() {}

// GoContinueStmt represents: continue
type GoContinueStmt struct{}

func (GoContinueStmt) goStmt() {}

// GoBlankLine emits a blank line in the output.
type GoBlankLine struct{}

func (GoBlankLine) goStmt() {}
func (GoBlankLine) goDecl() {}

// GoLineDirective represents: //line file:N
type GoLineDirective struct {
	File string
	Line int
}

func (GoLineDirective) goStmt() {}

// GoComment represents: // text
type GoComment struct {
	Text string
}

func (GoComment) goStmt() {}
func (GoComment) goDecl() {}

// GoRawStmt is an escape hatch for raw Go code at the statement level.
type GoRawStmt struct {
	Code string
}

func (GoRawStmt) goStmt() {}

// --- Expression level ---

// GoRawExpr wraps a raw Go expression string (from exprString).
type GoRawExpr struct {
	Code string
}

func (GoRawExpr) goExpr() {}

// GoIIFEExpr represents a self-calling function: func() T { body; return expr }()
type GoIIFEExpr struct {
	ReturnType string   // e.g., "interface{}", "(r interface{})"
	Body       []GoStmt // statements inside the IIFE
	Result     GoExpr   // final return expression; nil to omit
}

func (GoIIFEExpr) goExpr() {}
