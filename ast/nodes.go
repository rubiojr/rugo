package ast

// Node is the interface for all AST nodes.
type Node interface {
	node()
}

// Statement is the interface for statement nodes.
type Statement interface {
	Node
	stmt()
	StmtLine() int
	StmtEndLine() int
}

// BaseStmt provides common fields for all statements.
type BaseStmt struct {
	SourceLine int // start line in the original source
	EndLine    int // end line in the original source (0 if unknown)
}

func (b BaseStmt) StmtLine() int    { return b.SourceLine }
func (b BaseStmt) StmtEndLine() int { return b.EndLine }

// Expr is the interface for expression nodes.
type Expr interface {
	Node
	expr()
}

// Program is the root node.
type Program struct {
	Statements []Statement
	SourceFile string       // display path of the source file
	RawSource  string       // original source before preprocessing (for comment access)
	Structs    []StructInfo // struct definitions found during preprocessing
}

// StructInfo holds metadata about a struct definition extracted during
// preprocessing. Structs are expanded into constructor functions before
// parsing, so they don't appear in the AST as nodes.
type StructInfo struct {
	Name   string   // struct name (e.g. "Dog")
	Fields []string // field names
	Line   int      // 1-based line number of the struct keyword in original source
}

func (p *Program) node() {}

// UseStmt represents use "rugo_module" (Rugo stdlib module).
type UseStmt struct {
	BaseStmt
	Module string
}

func (u *UseStmt) node() {}
func (u *UseStmt) stmt() {}

// ImportStmt represents import "go/pkg" [as alias] (Go stdlib bridge).
type ImportStmt struct {
	BaseStmt
	Package string // Go package path (e.g. "strings", "path/filepath")
	Alias   string // optional alias (e.g. "fp" for filepath)
}

func (i *ImportStmt) node() {}
func (i *ImportStmt) stmt() {}

// RequireStmt represents require "path" [as "alias" | with mod1, mod2, ...].
type RequireStmt struct {
	BaseStmt
	Path  string
	Alias string   // empty means use filename as namespace
	With  []string // selective sub-module names (mutually exclusive with Alias)
}

func (r *RequireStmt) node() {}
func (r *RequireStmt) stmt() {}

// SandboxStmt represents the sandbox directive for Landlock-based process sandboxing.
// A bare `sandbox` (all fields empty) means maximum restriction (deny everything).
type SandboxStmt struct {
	BaseStmt
	RO      []string // read-only paths
	RW      []string // read-write paths
	ROX     []string // read + execute paths
	RWX     []string // read + write + execute paths
	Connect []int    // allowed TCP connect ports
	Bind    []int    // allowed TCP bind ports
}

func (s *SandboxStmt) node() {}
func (s *SandboxStmt) stmt() {}

// FuncDef represents def name(params) body end.
type FuncDef struct {
	BaseStmt
	Name       string
	Params     []string
	Body       []Statement
	Namespace  string // set during require resolution for namespaced functions
	SourceFile string // original source file for //line directives (set for require'd functions)
}

func (f *FuncDef) node() {}
func (f *FuncDef) stmt() {}

// TestDef represents test "name" body end.
type TestDef struct {
	BaseStmt
	Name string
	Body []Statement
}

func (t *TestDef) node() {}
func (t *TestDef) stmt() {}

// BenchDef represents bench "name" body end.
type BenchDef struct {
	BaseStmt
	Name string
	Body []Statement
}

func (b *BenchDef) node() {}
func (b *BenchDef) stmt() {}

// IfStmt represents if/elsif/else/end.
type IfStmt struct {
	BaseStmt
	Condition    Expr
	Body         []Statement
	ElsifClauses []ElsifClause
	ElseBody     []Statement
}

func (i *IfStmt) node() {}
func (i *IfStmt) stmt() {}

// ElsifClause is one elsif branch.
type ElsifClause struct {
	Condition Expr
	Body      []Statement
}

// WhileStmt represents while cond body end.
type WhileStmt struct {
	BaseStmt
	Condition Expr
	Body      []Statement
}

func (w *WhileStmt) node() {}
func (w *WhileStmt) stmt() {}

// ForStmt represents for var [, var2] in expr body end.
type ForStmt struct {
	BaseStmt
	Var        string // value variable (or key for hashes)
	IndexVar   string // optional second variable (index for arrays, value for hashes)
	Collection Expr
	Body       []Statement
}

func (f *ForStmt) node() {}
func (f *ForStmt) stmt() {}

// BreakStmt represents break.
type BreakStmt struct{ BaseStmt }

func (b *BreakStmt) node() {}
func (b *BreakStmt) stmt() {}

// NextStmt represents next (continue).
type NextStmt struct{ BaseStmt }

func (n *NextStmt) node() {}
func (n *NextStmt) stmt() {}

// ReturnStmt represents return [expr].
type ReturnStmt struct {
	BaseStmt
	Value Expr // nil if bare return
}

func (r *ReturnStmt) node() {}
func (r *ReturnStmt) stmt() {}

// ExprStmt is a statement that is just an expression.
type ExprStmt struct {
	BaseStmt
	Expression Expr
}

func (e *ExprStmt) node() {}
func (e *ExprStmt) stmt() {}

// AssignStmt represents target = value.
type AssignStmt struct {
	BaseStmt
	Target    string
	Value     Expr
	Namespace string // non-empty for top-level assignments from require'd files
}

func (a *AssignStmt) node() {}
func (a *AssignStmt) stmt() {}

// IndexAssignStmt represents obj[index] = value.
type IndexAssignStmt struct {
	BaseStmt
	Object Expr
	Index  Expr
	Value  Expr
}

func (ia *IndexAssignStmt) node() {}
func (ia *IndexAssignStmt) stmt() {}

// DotAssignStmt represents obj.field = value.
type DotAssignStmt struct {
	BaseStmt
	Object Expr
	Field  string
	Value  Expr
}

func (da *DotAssignStmt) node() {}
func (da *DotAssignStmt) stmt() {}

// BinaryExpr represents left op right.
type BinaryExpr struct {
	Left  Expr
	Op    string
	Right Expr
}

func (b *BinaryExpr) node() {}
func (b *BinaryExpr) expr() {}

// UnaryExpr represents op operand.
type UnaryExpr struct {
	Op      string
	Operand Expr
}

func (u *UnaryExpr) node() {}
func (u *UnaryExpr) expr() {}

// CallExpr represents func(args...).
type CallExpr struct {
	Func Expr
	Args []Expr
}

func (c *CallExpr) node() {}
func (c *CallExpr) expr() {}

// IndexExpr represents obj[index].
type IndexExpr struct {
	Object Expr
	Index  Expr
}

func (i *IndexExpr) node() {}
func (i *IndexExpr) expr() {}

// SliceExpr represents obj[start, length].
type SliceExpr struct {
	Object Expr
	Start  Expr
	Length Expr
}

func (s *SliceExpr) node() {}
func (s *SliceExpr) expr() {}

// DotExpr represents obj.field (used for namespace.func access).
type DotExpr struct {
	Object Expr
	Field  string
}

func (d *DotExpr) node() {}
func (d *DotExpr) expr() {}

// IdentExpr is a variable/function reference.
type IdentExpr struct {
	Name string
}

func (i *IdentExpr) node() {}
func (i *IdentExpr) expr() {}

// IntLiteral is an integer literal.
type IntLiteral struct {
	Value string
}

func (i *IntLiteral) node() {}
func (i *IntLiteral) expr() {}

// FloatLiteral is a floating point literal.
type FloatLiteral struct {
	Value string
}

func (f *FloatLiteral) node() {}
func (f *FloatLiteral) expr() {}

// StringLiteral is a string literal (with quotes stripped).
type StringLiteral struct {
	Value string // raw string content including interpolation markers
	Raw   bool   // true for single-quoted raw strings (no escape processing)
}

func (s *StringLiteral) node() {}
func (s *StringLiteral) expr() {}

// BoolLiteral is true or false.
type BoolLiteral struct {
	Value bool
}

func (b *BoolLiteral) node() {}
func (b *BoolLiteral) expr() {}

// NilLiteral represents nil.
type NilLiteral struct{}

func (n *NilLiteral) node() {}
func (n *NilLiteral) expr() {}

// ArrayLiteral is [elem, ...].
type ArrayLiteral struct {
	Elements []Expr
}

func (a *ArrayLiteral) node() {}
func (a *ArrayLiteral) expr() {}

// HashLiteral is {key => value, ...}.
type HashLiteral struct {
	Pairs []HashPair
}

func (h *HashLiteral) node() {}
func (h *HashLiteral) expr() {}

// HashPair is a key => value pair.
type HashPair struct {
	Key   Expr
	Value Expr
}

// TryExpr represents try expr or err handler end.
type TryExpr struct {
	Expr    Expr        // expression to try
	ErrVar  string      // error variable name
	Handler []Statement // handler body; last expression is the result
}

func (t *TryExpr) node() {}
func (t *TryExpr) expr() {}

// SpawnExpr represents spawn body end (goroutine-backed concurrency).
type SpawnExpr struct {
	Body []Statement // last expression is the task result
}

func (s *SpawnExpr) node() {}
func (s *SpawnExpr) expr() {}

// ParallelExpr represents parallel body end (fan-out concurrency).
// Each statement in Body runs in its own goroutine; returns an array of results.
type ParallelExpr struct {
	Body []Statement
}

func (p *ParallelExpr) node() {}
func (p *ParallelExpr) expr() {}

// FnExpr represents fn(params) body end (first-class lambda).
type FnExpr struct {
	Params []string
	Body   []Statement
}

func (f *FnExpr) node() {}
func (f *FnExpr) expr() {}
