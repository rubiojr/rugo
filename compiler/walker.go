package compiler

import (
	"fmt"
	"strings"

	"github.com/rubiojr/rugo/parser"
)

// Walker converts the flat []int32 AST from egg into typed AST nodes.
type Walker struct {
	p       *parser.Parser
	ast     []int32
	lineMap []int // maps preprocessed line → original source line (nil if 1:1)
}

// Walk converts a flat AST into a typed Program.
func Walk(p *parser.Parser, ast []int32) (*Program, error) {
	w := &Walker{p: p, ast: ast}
	return w.walkProgram()
}

// WalkWithLineMap converts a flat AST into a typed Program, applying a line map
// from preprocessed lines back to original source lines.
func WalkWithLineMap(p *parser.Parser, ast []int32, lineMap []int) (*Program, error) {
	w := &Walker{p: p, ast: ast, lineMap: lineMap}
	return w.walkProgram()
}

// tokenLine returns the original source line for a token index.
func (w *Walker) tokenLine(idx int32) int {
	tok := w.p.Token(idx)
	line := tok.Position().Line
	if w.lineMap != nil && line > 0 && line <= len(w.lineMap) {
		return w.lineMap[line-1]
	}
	return line
}

// firstTokenLine returns the original source line from the first terminal token
// found by recursively traversing into non-terminals.
func (w *Walker) firstTokenLine(ast []int32) int {
	for i := 0; i < len(ast); i++ {
		if ast[i] >= 0 {
			return w.tokenLine(ast[i])
		}
		// Recurse into non-terminal children: -sym, count, children...
		if i+1 < len(ast) {
			count := int(ast[i+1])
			if count > 0 {
				inner := ast[i+2 : i+2+count]
				if line := w.firstTokenLine(inner); line > 0 {
					return line
				}
			}
			i += 1 + count
		}
	}
	return 0
}

// readNonTerminal reads a non-terminal header (-sym, count) and returns the symbol and sub-slice.
func (w *Walker) readNonTerminal(ast []int32) (parser.Symbol, []int32, []int32) {
	if len(ast) < 2 || ast[0] >= 0 {
		return 0, nil, ast
	}
	sym := parser.Symbol(-ast[0])
	count := ast[1]
	children := ast[2 : 2+count]
	rest := ast[2+count:]
	return sym, children, rest
}

// readToken reads a terminal token index and returns it.
func (w *Walker) readToken(ast []int32) (token, []int32) {
	if len(ast) == 0 {
		return token{}, ast
	}
	idx := ast[0]
	tok := w.p.Token(idx)
	return token{
		src: tok.Src(),
		ch:  parser.Symbol(tok.Ch),
	}, ast[1:]
}

type token struct {
	src string
	ch  parser.Symbol
}

func (w *Walker) walkProgram() (*Program, error) {
	sym, children, _ := w.readNonTerminal(w.ast)
	if sym != parser.RugoProgram {
		return nil, fmt.Errorf("expected Program, got %v", sym)
	}
	prog := &Program{}
	for len(children) > 0 {
		// Skip EOF token
		if children[0] >= 0 {
			tok := w.p.Token(children[0])
			if parser.Symbol(tok.Ch) == parser.RugoTOK_EOF {
				break
			}
		}
		var stmt Statement
		var err error
		stmt, children, err = w.walkStatement(children)
		if err != nil {
			return nil, err
		}
		prog.Statements = append(prog.Statements, stmt)
	}
	return prog, nil
}

func (w *Walker) walkStatement(ast []int32) (Statement, []int32, error) {
	sym, children, rest := w.readNonTerminal(ast)
	if sym != parser.RugoStatement {
		return nil, rest, fmt.Errorf("expected Statement, got %v", sym)
	}
	// Get line number from the first token in this statement
	line := w.firstTokenLine(children)
	// The Statement contains exactly one child non-terminal
	innerSym, innerChildren, _ := w.readNonTerminal(children)
	var stmt Statement
	var err error
	switch innerSym {
	case parser.RugoImportStmt:
		stmt, err = w.walkImportStmt(innerChildren)
	case parser.RugoRequireStmt:
		stmt, err = w.walkRequireStmt(innerChildren)
	case parser.RugoFuncDef:
		stmt, err = w.walkFuncDef(innerChildren)
	case parser.RugoTestDef:
		stmt, err = w.walkTestDef(innerChildren)
	case parser.RugoIfStmt:
		stmt, err = w.walkIfStmt(innerChildren)
	case parser.RugoWhileStmt:
		stmt, err = w.walkWhileStmt(innerChildren)
	case parser.RugoForStmt:
		stmt, err = w.walkForStmt(innerChildren)
	case parser.RugoBreakStmt:
		stmt = &BreakStmt{}
	case parser.RugoNextStmt:
		stmt = &NextStmt{}
	case parser.RugoReturnStmt:
		stmt, err = w.walkReturnStmt(innerChildren)
	case parser.RugoAssignOrExpr:
		stmt, err = w.walkAssignOrExpr(innerChildren)
	default:
		return nil, rest, fmt.Errorf("unexpected statement type: %v", innerSym)
	}
	if err != nil {
		return nil, rest, err
	}
	// Set the source line on the statement
	if line > 0 {
		switch s := stmt.(type) {
		case *ImportStmt:
			s.SourceLine = line
		case *RequireStmt:
			s.SourceLine = line
		case *FuncDef:
			s.SourceLine = line
		case *TestDef:
			s.SourceLine = line
		case *IfStmt:
			s.SourceLine = line
		case *WhileStmt:
			s.SourceLine = line
		case *ForStmt:
			s.SourceLine = line
		case *BreakStmt:
			s.SourceLine = line
		case *NextStmt:
			s.SourceLine = line
		case *ReturnStmt:
			s.SourceLine = line
		case *ExprStmt:
			s.SourceLine = line
		case *AssignStmt:
			s.SourceLine = line
		case *IndexAssignStmt:
			s.SourceLine = line
		}
	}
	return stmt, rest, nil
}

func (w *Walker) walkImportStmt(ast []int32) (Statement, error) {
	// ImportStmt = "import" str_lit .
	_, ast = w.readToken(ast) // skip "import"
	tok, _ := w.readToken(ast)
	module := unquoteString(tok.src)
	return &ImportStmt{Module: module}, nil
}

func (w *Walker) walkRequireStmt(ast []int32) (Statement, error) {
	// RequireStmt = "require" str_lit [ "as" str_lit ] .
	_, ast = w.readToken(ast) // skip "require"
	tok, ast := w.readToken(ast)
	path := unquoteString(tok.src)
	alias := ""
	// Check for optional "as" alias
	if len(ast) > 0 && ast[0] >= 0 {
		nextTok := w.p.Token(ast[0])
		if parser.Symbol(nextTok.Ch) == parser.RugoTOK_as {
			_, ast = w.readToken(ast) // consume "as"
			aliasTok, _ := w.readToken(ast)
			alias = unquoteString(aliasTok.src)
		}
	}
	return &RequireStmt{Path: path, Alias: alias}, nil
}

func (w *Walker) walkTestDef(ast []int32) (Statement, error) {
	// TestDef = "test" str_lit Body "end" .
	_, ast = w.readToken(ast) // skip "test"
	nameTok, ast := w.readToken(ast)
	name := unquoteString(nameTok.src)

	var body []Statement
	if len(ast) > 0 && ast[0] < 0 {
		sym, children, rest := w.readNonTerminal(ast)
		if sym == parser.RugoBody {
			var err error
			body, err = w.walkBody(children)
			if err != nil {
				return nil, err
			}
			ast = rest
		}
	}
	_ = ast // "end" token
	return &TestDef{Name: name, Body: body}, nil
}

func (w *Walker) walkFuncDef(ast []int32) (Statement, error) {
	// FuncDef = "def" ident '(' [ ParamList ] ')' Body "end" .
	_, ast = w.readToken(ast) // "def"
	nameTok, ast := w.readToken(ast)
	_, ast = w.readToken(ast) // '('

	var params []string
	// Check if next is ParamList (non-terminal) or ')' (terminal)
	if len(ast) > 0 && ast[0] < 0 {
		sym, children, rest := w.readNonTerminal(ast)
		if sym == parser.RugoParamList {
			params = w.walkParamList(children)
			ast = rest
		}
	}

	_, ast = w.readToken(ast) // ')'

	// Body
	var body []Statement
	if len(ast) > 0 && ast[0] < 0 {
		sym, children, _ := w.readNonTerminal(ast)
		if sym == parser.RugoBody {
			var err error
			body, err = w.walkBody(children)
			if err != nil {
				return nil, err
			}
		}
	}

	// "end"
	// _, _ = w.readToken(ast)

	return &FuncDef{Name: nameTok.src, Params: params, Body: body}, nil
}

func (w *Walker) walkParamList(ast []int32) []string {
	// ParamList = ident { ',' ident } .
	var params []string
	for len(ast) > 0 {
		tok, rest := w.readToken(ast)
		ast = rest
		if tok.ch == parser.RugoTOK_002c { // ','
			continue
		}
		params = append(params, tok.src)
	}
	return params
}

func (w *Walker) walkBody(ast []int32) ([]Statement, error) {
	// Body = { Statement } .
	var stmts []Statement
	for len(ast) > 0 {
		stmt, rest, err := w.walkStatement(ast)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
		ast = rest
	}
	return stmts, nil
}

func (w *Walker) walkIfStmt(ast []int32) (Statement, error) {
	// IfStmt = "if" Expr Body { "elsif" Expr Body } [ "else" Body ] "end" .
	_, ast = w.readToken(ast) // "if"

	cond, ast, err := w.walkExpr(ast)
	if err != nil {
		return nil, err
	}

	// Body
	var body []Statement
	sym, children, rest := w.readNonTerminal(ast)
	if sym == parser.RugoBody {
		body, err = w.walkBody(children)
		if err != nil {
			return nil, err
		}
		ast = rest
	}

	var elsifClauses []ElsifClause
	var elseBody []Statement

	for len(ast) > 0 {
		// Check next token
		if ast[0] >= 0 {
			tok, rest := w.readToken(ast)
			if tok.ch == parser.RugoTOK_elsif {
				ast = rest
				elsifCond, r, err := w.walkExpr(ast)
				if err != nil {
					return nil, err
				}
				ast = r
				sym, children, rest := w.readNonTerminal(ast)
				if sym == parser.RugoBody {
					elsifBody, err := w.walkBody(children)
					if err != nil {
						return nil, err
					}
					elsifClauses = append(elsifClauses, ElsifClause{
						Condition: elsifCond,
						Body:      elsifBody,
					})
					ast = rest
				}
			} else if tok.ch == parser.RugoTOK_else {
				ast = rest
				sym, children, rest := w.readNonTerminal(ast)
				if sym == parser.RugoBody {
					elseBody, err = w.walkBody(children)
					if err != nil {
						return nil, err
					}
					ast = rest
				}
			} else {
				// "end"
				break
			}
		} else {
			break
		}
	}

	return &IfStmt{
		Condition:    cond,
		Body:         body,
		ElsifClauses: elsifClauses,
		ElseBody:     elseBody,
	}, nil
}

func (w *Walker) walkWhileStmt(ast []int32) (Statement, error) {
	// WhileStmt = "while" Expr Body "end" .
	_, ast = w.readToken(ast) // "while"
	cond, ast, err := w.walkExpr(ast)
	if err != nil {
		return nil, err
	}
	sym, children, rest := w.readNonTerminal(ast)
	var body []Statement
	if sym == parser.RugoBody {
		body, err = w.walkBody(children)
		if err != nil {
			return nil, err
		}
		_ = rest
	}
	return &WhileStmt{Condition: cond, Body: body}, nil
}

func (w *Walker) walkForStmt(ast []int32) (Statement, error) {
	// ForStmt = "for" ident [ ',' ident ] "in" Expr Body "end" .
	_, ast = w.readToken(ast) // "for"
	varTok, ast := w.readToken(ast)

	var indexVar string
	// Check for optional comma and second ident
	if len(ast) > 0 && ast[0] >= 0 {
		tok := w.p.Token(ast[0])
		if parser.Symbol(tok.Ch) == parser.RugoTOK_002c { // ','
			_, ast = w.readToken(ast) // consume ','
			idxTok, r := w.readToken(ast)
			indexVar = idxTok.src
			ast = r
		}
	}

	_, ast = w.readToken(ast) // "in"

	coll, ast, err := w.walkExpr(ast)
	if err != nil {
		return nil, err
	}

	sym, children, _ := w.readNonTerminal(ast)
	var body []Statement
	if sym == parser.RugoBody {
		body, err = w.walkBody(children)
		if err != nil {
			return nil, err
		}
	}
	// "end" consumed by parser
	return &ForStmt{Var: varTok.src, IndexVar: indexVar, Collection: coll, Body: body}, nil
}

func (w *Walker) walkReturnStmt(ast []int32) (Statement, error) {
	// ReturnStmt = "return" [ Expr ] .
	_, ast = w.readToken(ast) // "return"
	if len(ast) == 0 {
		return &ReturnStmt{}, nil
	}
	val, _, err := w.walkExpr(ast)
	if err != nil {
		return nil, err
	}
	return &ReturnStmt{Value: val}, nil
}

func (w *Walker) walkAssignOrExpr(ast []int32) (Statement, error) {
	// AssignOrExpr = Expr [ '=' Expr ] .
	lhs, ast, err := w.walkExpr(ast)
	if err != nil {
		return nil, err
	}
	if len(ast) > 0 {
		// '=' followed by Expr
		_, ast = w.readToken(ast) // '='
		rhs, _, err := w.walkExpr(ast)
		if err != nil {
			return nil, err
		}
		// LHS must be an identifier or index expression for assignment
		if ident, ok := lhs.(*IdentExpr); ok {
			return &AssignStmt{Target: ident.Name, Value: rhs}, nil
		}
		if idx, ok := lhs.(*IndexExpr); ok {
			return &IndexAssignStmt{Object: idx.Object, Index: idx.Index, Value: rhs}, nil
		}
		return nil, fmt.Errorf("invalid assignment target")
	}
	return &ExprStmt{Expression: lhs}, nil
}

func (w *Walker) walkExpr(ast []int32) (Expr, []int32, error) {
	// Expr = OrExpr .
	sym, children, rest := w.readNonTerminal(ast)
	if sym != parser.RugoExpr {
		return nil, rest, fmt.Errorf("expected Expr, got %v", sym)
	}
	expr, err := w.walkOrExpr(children)
	return expr, rest, err
}

func (w *Walker) walkOrExpr(ast []int32) (Expr, error) {
	// OrExpr = AndExpr { "||" AndExpr } .
	sym, children, rest := w.readNonTerminal(ast)
	if sym != parser.RugoOrExpr {
		return nil, fmt.Errorf("expected OrExpr, got %v", sym)
	}
	ast = children

	left, ast, err := w.walkAndExpr(ast)
	if err != nil {
		return nil, err
	}
	_ = rest
	for len(ast) > 0 {
		_, ast = w.readToken(ast) // "||"
		right, r, err := w.walkAndExpr(ast)
		if err != nil {
			return nil, err
		}
		ast = r
		left = &BinaryExpr{Left: left, Op: "||", Right: right}
	}
	return left, nil
}

func (w *Walker) walkAndExpr(ast []int32) (Expr, []int32, error) {
	// AndExpr = CompExpr { "&&" CompExpr } .
	sym, children, rest := w.readNonTerminal(ast)
	if sym != parser.RugoAndExpr {
		return nil, rest, fmt.Errorf("expected AndExpr, got %v", sym)
	}

	left, remaining, err := w.walkCompExpr(children)
	if err != nil {
		return nil, rest, err
	}
	for len(remaining) > 0 {
		_, remaining = w.readToken(remaining) // "&&"
		right, r, err := w.walkCompExpr(remaining)
		if err != nil {
			return nil, rest, err
		}
		remaining = r
		left = &BinaryExpr{Left: left, Op: "&&", Right: right}
	}
	return left, rest, nil
}

func (w *Walker) walkCompExpr(ast []int32) (Expr, []int32, error) {
	// CompExpr = AddExpr [ comp_op AddExpr ] .
	sym, children, rest := w.readNonTerminal(ast)
	if sym != parser.RugoCompExpr {
		return nil, rest, fmt.Errorf("expected CompExpr, got %v", sym)
	}

	left, remaining, err := w.walkAddExpr(children)
	if err != nil {
		return nil, rest, err
	}
	if len(remaining) > 0 {
		opTok, remaining := w.readToken(remaining) // comp_op
		right, _, err := w.walkAddExpr(remaining)
		if err != nil {
			return nil, rest, err
		}
		left = &BinaryExpr{Left: left, Op: opTok.src, Right: right}
	}
	return left, rest, nil
}

func (w *Walker) walkAddExpr(ast []int32) (Expr, []int32, error) {
	// AddExpr = MulExpr { ('+' | '-') MulExpr } .
	sym, children, rest := w.readNonTerminal(ast)
	if sym != parser.RugoAddExpr {
		return nil, rest, fmt.Errorf("expected AddExpr, got %v", sym)
	}

	left, remaining, err := w.walkMulExpr(children)
	if err != nil {
		return nil, rest, err
	}
	for len(remaining) > 0 {
		opTok, remaining2 := w.readToken(remaining)
		right, r, err := w.walkMulExpr(remaining2)
		if err != nil {
			return nil, rest, err
		}
		remaining = r
		left = &BinaryExpr{Left: left, Op: opTok.src, Right: right}
	}
	return left, rest, nil
}

func (w *Walker) walkMulExpr(ast []int32) (Expr, []int32, error) {
	// MulExpr = UnaryExpr { ('*' | '/' | '%') UnaryExpr } .
	sym, children, rest := w.readNonTerminal(ast)
	if sym != parser.RugoMulExpr {
		return nil, rest, fmt.Errorf("expected MulExpr, got %v", sym)
	}

	left, remaining, err := w.walkUnaryExpr(children)
	if err != nil {
		return nil, rest, err
	}
	for len(remaining) > 0 {
		opTok, remaining2 := w.readToken(remaining)
		right, r, err := w.walkUnaryExpr(remaining2)
		if err != nil {
			return nil, rest, err
		}
		remaining = r
		left = &BinaryExpr{Left: left, Op: opTok.src, Right: right}
	}
	return left, rest, nil
}

func (w *Walker) walkUnaryExpr(ast []int32) (Expr, []int32, error) {
	// UnaryExpr = '!' Postfix | '-' Postfix | Postfix .
	sym, children, rest := w.readNonTerminal(ast)
	if sym != parser.RugoUnaryExpr {
		return nil, rest, fmt.Errorf("expected UnaryExpr, got %v", sym)
	}

	// Check first element: if it's a token for '!' or '-', it's a unary op
	if len(children) > 0 && children[0] >= 0 {
		tok := w.p.Token(children[0])
		ch := parser.Symbol(tok.Ch)
		if ch == parser.RugoTOK_0021 || ch == parser.RugoTOK_002d { // '!' or '-'
			_, remaining := w.readToken(children) // consume op
			operand, err := w.walkPostfix(remaining)
			if err != nil {
				return nil, rest, err
			}
			return &UnaryExpr{Op: tok.Src(), Operand: operand}, rest, nil
		}
	}

	expr, err := w.walkPostfix(children)
	return expr, rest, err
}

func (w *Walker) walkPostfix(ast []int32) (Expr, error) {
	// Postfix = Primary { Suffix } .
	sym, children, _ := w.readNonTerminal(ast)
	if sym != parser.RugoPostfix {
		return nil, fmt.Errorf("expected Postfix, got %v", sym)
	}

	expr, remaining, err := w.walkPrimary(children)
	if err != nil {
		return nil, err
	}

	for len(remaining) > 0 {
		s, r, _ := w.readNonTerminal(remaining)
		if s != parser.RugoSuffix {
			break
		}
		var serr error
		expr, serr = w.walkSuffix(expr, r)
		if serr != nil {
			return nil, serr
		}
		// After consuming Suffix's children, remaining is the rest of the Postfix children
		remaining = remaining[2+remaining[1]:]
	}
	return expr, nil
}

func (w *Walker) walkSuffix(obj Expr, ast []int32) (Expr, error) {
	// Suffix = '(' [ ArgList ] ')' | '[' Expr ']' .
	if len(ast) == 0 {
		return obj, nil
	}
	tok, rest := w.readToken(ast)
	switch tok.ch {
	case parser.RugoTOK_0028: // '('
		// Function call
		var args []Expr
		for len(rest) > 0 {
			if rest[0] >= 0 {
				// Check if it's ')'
				nextTok := w.p.Token(rest[0])
				if parser.Symbol(nextTok.Ch) == parser.RugoTOK_0029 {
					break
				}
			}
			if rest[0] < 0 {
				// ArgList
				s, children, r := w.readNonTerminal(rest)
				if s == parser.RugoArgList {
					var err error
					args, err = w.walkArgList(children)
					if err != nil {
						return nil, err
					}
					rest = r
				}
			}
		}
		// consume ')'
		return &CallExpr{Func: obj, Args: args}, nil

	case parser.RugoTOK_005b: // '['
		// Index access
		idx, rest, err := w.walkExpr(rest)
		if err != nil {
			return nil, err
		}
		_ = rest // consume ']'
		return &IndexExpr{Object: obj, Index: idx}, nil

	case parser.RugoTOK_002e: // '.'
		// Dot access: namespace.func
		fieldTok, _ := w.readToken(rest)
		return &DotExpr{Object: obj, Field: fieldTok.src}, nil
	}
	return obj, nil
}

func (w *Walker) walkArgList(ast []int32) ([]Expr, error) {
	// ArgList = Expr { ',' Expr } .
	var args []Expr
	for len(ast) > 0 {
		if ast[0] >= 0 {
			// comma token, skip
			_, ast = w.readToken(ast)
			continue
		}
		expr, rest, err := w.walkExpr(ast)
		if err != nil {
			return nil, err
		}
		args = append(args, expr)
		ast = rest
	}
	return args, nil
}

func (w *Walker) walkPrimary(ast []int32) (Expr, []int32, error) {
	// Primary = ident | integer | float_lit | str_lit | "true" | "false" | "nil"
	//         | ArrayLit | HashLit | '(' Expr ')' .
	sym, children, rest := w.readNonTerminal(ast)
	if sym != parser.RugoPrimary {
		return nil, rest, fmt.Errorf("expected Primary, got %v", sym)
	}

	if len(children) == 0 {
		return nil, rest, fmt.Errorf("empty Primary")
	}

	// Check if first element is a non-terminal (ArrayLit, HashLit, TryExpr)
	if children[0] < 0 {
		innerSym := parser.Symbol(-children[0])
		switch innerSym {
		case parser.RugoArrayLit:
			expr, err := w.walkArrayLit(children[2 : 2+children[1]])
			return expr, rest, err
		case parser.RugoHashLit:
			expr, err := w.walkHashLit(children[2 : 2+children[1]])
			return expr, rest, err
		case parser.RugoTryExpr:
			expr, err := w.walkTryExpr(children[2 : 2+children[1]])
			return expr, rest, err
		case parser.RugoSpawnExpr:
			expr, err := w.walkSpawnExpr(children[2 : 2+children[1]])
			return expr, rest, err
		case parser.RugoParallelExpr:
			expr, err := w.walkParallelExpr(children[2 : 2+children[1]])
			return expr, rest, err
		}
	}

	// Terminal token
	tok, remaining := w.readToken(children)
	switch tok.ch {
	case parser.Rugoident:
		return &IdentExpr{Name: tok.src}, rest, nil
	case parser.Rugointeger:
		return &IntLiteral{Value: tok.src}, rest, nil
	case parser.Rugofloat_lit:
		return &FloatLiteral{Value: tok.src}, rest, nil
	case parser.Rugostr_lit:
		return &StringLiteral{Value: unquoteString(tok.src)}, rest, nil
	case parser.RugoTOK_true:
		return &BoolLiteral{Value: true}, rest, nil
	case parser.RugoTOK_false:
		return &BoolLiteral{Value: false}, rest, nil
	case parser.RugoTOK_nil:
		return &NilLiteral{}, rest, nil
	case parser.RugoTOK_0028: // '(' Expr ')'
		expr, _, err := w.walkExpr(remaining)
		return expr, rest, err
	default:
		return nil, rest, fmt.Errorf("unexpected token in Primary: %v %q", tok.ch, tok.src)
	}
}

func (w *Walker) walkArrayLit(ast []int32) (Expr, error) {
	// ArrayLit = '[' [ Expr { ',' Expr } ] ']' .
	_, ast = w.readToken(ast) // '['
	var elems []Expr
	for len(ast) > 0 {
		if ast[0] >= 0 {
			tok := w.p.Token(ast[0])
			ch := parser.Symbol(tok.Ch)
			if ch == parser.RugoTOK_005d { // ']'
				break
			}
			if ch == parser.RugoTOK_002c { // ','
				_, ast = w.readToken(ast)
				continue
			}
		}
		expr, rest, err := w.walkExpr(ast)
		if err != nil {
			return nil, err
		}
		elems = append(elems, expr)
		ast = rest
	}
	return &ArrayLiteral{Elements: elems}, nil
}

func (w *Walker) walkHashLit(ast []int32) (Expr, error) {
	// HashLit = '{' [ HashEntry { ',' HashEntry } ] '}' .
	_, ast = w.readToken(ast) // '{'
	var pairs []HashPair
	for len(ast) > 0 {
		if ast[0] >= 0 {
			tok := w.p.Token(ast[0])
			ch := parser.Symbol(tok.Ch)
			if ch == parser.RugoTOK_007d { // '}'
				break
			}
			if ch == parser.RugoTOK_002c { // ','
				_, ast = w.readToken(ast)
				continue
			}
		}
		if ast[0] < 0 {
			sym, children, rest := w.readNonTerminal(ast)
			if sym == parser.RugoHashEntry {
				key, remaining, err := w.walkExpr(children)
				if err != nil {
					return nil, err
				}
				_, remaining = w.readToken(remaining) // "=>"
				value, _, err := w.walkExpr(remaining)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, HashPair{Key: key, Value: value})
			}
			ast = rest
		}
	}
	return &HashLiteral{Pairs: pairs}, nil
}

func (w *Walker) walkTryExpr(ast []int32) (Expr, error) {
	// TryExpr = "try" Expr "or" ident Body "end" .
	_, ast = w.readToken(ast) // "try"

	expr, ast, err := w.walkExpr(ast)
	if err != nil {
		return nil, err
	}

	_, ast = w.readToken(ast) // "or"

	errTok, ast := w.readToken(ast) // ident (error variable)

	sym, children, _ := w.readNonTerminal(ast)
	var handler []Statement
	if sym == parser.RugoBody {
		handler, err = w.walkBody(children)
		if err != nil {
			return nil, err
		}
	}
	// "end" is consumed by the parser

	return &TryExpr{
		Expr:    expr,
		ErrVar:  errTok.src,
		Handler: handler,
	}, nil
}

func (w *Walker) walkSpawnExpr(ast []int32) (Expr, error) {
	// SpawnExpr = "spawn" Body "end" .
	_, ast = w.readToken(ast) // "spawn"

	sym, children, _ := w.readNonTerminal(ast)
	var body []Statement
	if sym == parser.RugoBody {
		var err error
		body, err = w.walkBody(children)
		if err != nil {
			return nil, err
		}
	}
	// "end" is consumed by the parser

	return &SpawnExpr{
		Body: body,
	}, nil
}

func (w *Walker) walkParallelExpr(ast []int32) (Expr, error) {
	// ParallelExpr = "parallel" Body "end" .
	_, ast = w.readToken(ast) // "parallel"

	sym, children, _ := w.readNonTerminal(ast)
	var body []Statement
	if sym == parser.RugoBody {
		var err error
		body, err = w.walkBody(children)
		if err != nil {
			return nil, err
		}
	}

	return &ParallelExpr{
		Body: body,
	}, nil
}

// unquoteString removes surrounding quotes and processes escape sequences.
func unquoteString(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\r`, "\r")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\"`, "\"")
	s = strings.ReplaceAll(s, `\\`, "\\")
	return s
}
