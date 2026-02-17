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

// --- Structured expression types ---

// GoIdentExpr represents a Go identifier reference.
type GoIdentExpr struct {
	Name string
}

func (GoIdentExpr) goExpr() {}

// GoIntLit represents an integer literal.
type GoIntLit struct {
	Value string
}

func (GoIntLit) goExpr() {}

// GoFloatLit represents a float literal.
type GoFloatLit struct {
	Value string
}

func (GoFloatLit) goExpr() {}

// GoStringLit represents a Go string literal (with quotes).
type GoStringLit struct {
	Value string // Go-escaped content WITHOUT quotes
}

func (GoStringLit) goExpr() {}

// GoBoolLit represents a boolean literal.
type GoBoolLit struct {
	Value bool
}

func (GoBoolLit) goExpr() {}

// GoNilExpr represents the nil value.
type GoNilExpr struct{}

func (GoNilExpr) goExpr() {}

// GoBinaryExpr represents: left op right
type GoBinaryExpr struct {
	Left  GoExpr
	Op    string
	Right GoExpr
}

func (GoBinaryExpr) goExpr() {}

// GoUnaryExpr represents: op operand
type GoUnaryExpr struct {
	Op      string
	Operand GoExpr
}

func (GoUnaryExpr) goExpr() {}

// GoCastExpr represents a type conversion: type(value)
type GoCastExpr struct {
	Type  string
	Value GoExpr
}

func (GoCastExpr) goExpr() {}

// GoTypeAssert represents a type assertion: value.(type)
type GoTypeAssert struct {
	Value GoExpr
	Type  string
}

func (GoTypeAssert) goExpr() {}

// GoCallExpr represents a function call: func(args...)
type GoCallExpr struct {
	Func string
	Args []GoExpr
}

func (GoCallExpr) goExpr() {}

// GoMethodCallExpr represents a method call: obj.Method(args...)
type GoMethodCallExpr struct {
	Object GoExpr
	Method string
	Args   []GoExpr
}

func (GoMethodCallExpr) goExpr() {}

// GoDotExpr represents field access: obj.Field
type GoDotExpr struct {
	Object GoExpr
	Field  string
}

func (GoDotExpr) goExpr() {}

// GoSliceLit represents a slice literal: []T{a, b, c}
type GoSliceLit struct {
	Type     string
	Elements []GoExpr
}

func (GoSliceLit) goExpr() {}

// GoMapLit represents a map literal: map[K]V{k: v, ...}
type GoMapLit struct {
	KeyType string
	ValType string
	Pairs   []GoMapPair
}

func (GoMapLit) goExpr() {}

// GoMapPair represents a key-value pair in a map literal.
type GoMapPair struct {
	Key   GoExpr
	Value GoExpr
}

// GoFmtSprintf represents: fmt.Sprintf(format, args...)
type GoFmtSprintf struct {
	Format string
	Args   []GoExpr
}

func (GoFmtSprintf) goExpr() {}

// GoStringConcat represents string concatenation: a + b + c
type GoStringConcat struct {
	Parts []GoExpr
}

func (GoStringConcat) goExpr() {}

// GoIndexExpr represents indexing: obj[idx]
type GoIndexExpr struct {
	Object GoExpr
	Index  GoExpr
}

func (GoIndexExpr) goExpr() {}

// GoParenExpr represents a parenthesized expression: (expr)
type GoParenExpr struct {
	Inner GoExpr
}

func (GoParenExpr) goExpr() {}
