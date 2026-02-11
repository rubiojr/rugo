package astmod

import (
	"fmt"
	"strings"

	"github.com/rubiojr/rugo/ast"
)

// --- ast module ---

type AST struct{}

func (*AST) ParseFile(path string) interface{} {
	c := &ast.Compiler{}
	prog, err := c.ParseFile(path)
	if err != nil {
		panic(fmt.Sprintf("ast.parse_file: %v", err))
	}
	return convertProgram(prog)
}

func (*AST) ParseSource(source, name string) interface{} {
	c := &ast.Compiler{}
	prog, err := c.ParseSource(source, name)
	if err != nil {
		panic(fmt.Sprintf("ast.parse_source: %v", err))
	}
	return convertProgram(prog)
}

func (*AST) SourceLines(prog, stmt interface{}) interface{} {
	progHash, ok := prog.(map[interface{}]interface{})
	if !ok {
		panic("ast.source_lines: first argument must be a program hash")
	}
	stmtHash, ok := stmt.(map[interface{}]interface{})
	if !ok {
		panic("ast.source_lines: second argument must be a statement hash")
	}

	rawSource, _ := progHash["raw_source"].(string)
	if rawSource == "" {
		return make([]interface{}, 0)
	}

	line, _ := stmtHash["line"].(int)
	endLine, _ := stmtHash["end_line"].(int)
	if line <= 0 || endLine <= 0 || endLine < line {
		return make([]interface{}, 0)
	}

	lines := strings.Split(rawSource, "\n")
	if line > len(lines) {
		return make([]interface{}, 0)
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	result := make([]interface{}, endLine-line+1)
	for i := line - 1; i < endLine; i++ {
		result[i-line+1] = lines[i]
	}
	return result
}

func convertProgram(prog *ast.Program) map[interface{}]interface{} {
	stmts := make([]interface{}, len(prog.Statements))
	for i, s := range prog.Statements {
		stmts[i] = convertStmt(s)
	}

	structs := make([]interface{}, len(prog.Structs))
	for i, si := range prog.Structs {
		fields := make([]interface{}, len(si.Fields))
		for j, f := range si.Fields {
			fields[j] = f
		}
		structs[i] = map[interface{}]interface{}{
			"name":   si.Name,
			"fields": fields,
			"line":   si.Line,
		}
	}

	return map[interface{}]interface{}{
		"source_file": prog.SourceFile,
		"raw_source":  prog.RawSource,
		"statements":  stmts,
		"structs":     structs,
	}
}

func convertStmt(s ast.Statement) map[interface{}]interface{} {
	m := map[interface{}]interface{}{
		"line":     s.StmtLine(),
		"end_line": s.StmtEndLine(),
	}

	switch st := s.(type) {
	case *ast.FuncDef:
		m["type"] = "def"
		m["name"] = st.Name
		params := make([]interface{}, len(st.Params))
		for i, p := range st.Params {
			params[i] = p
		}
		m["params"] = params
		m["body"] = convertBody(st.Body)

	case *ast.TestDef:
		m["type"] = "test"
		m["name"] = st.Name
		m["body"] = convertBody(st.Body)

	case *ast.BenchDef:
		m["type"] = "bench"
		m["name"] = st.Name
		m["body"] = convertBody(st.Body)

	case *ast.IfStmt:
		m["type"] = "if"
		m["body"] = convertBody(st.Body)
		elsifs := make([]interface{}, len(st.ElsifClauses))
		for i, ec := range st.ElsifClauses {
			elsifs[i] = map[interface{}]interface{}{
				"body": convertBody(ec.Body),
			}
		}
		m["elsif"] = elsifs
		m["else_body"] = convertBody(st.ElseBody)

	case *ast.WhileStmt:
		m["type"] = "while"
		m["body"] = convertBody(st.Body)

	case *ast.ForStmt:
		m["type"] = "for"
		m["var"] = st.Var
		if st.IndexVar != "" {
			m["index_var"] = st.IndexVar
		}
		m["body"] = convertBody(st.Body)

	case *ast.ReturnStmt:
		m["type"] = "return"

	case *ast.BreakStmt:
		m["type"] = "break"

	case *ast.NextStmt:
		m["type"] = "next"

	case *ast.AssignStmt:
		m["type"] = "assign"
		m["target"] = st.Target

	case *ast.IndexAssignStmt:
		m["type"] = "index_assign"

	case *ast.DotAssignStmt:
		m["type"] = "dot_assign"
		m["field"] = st.Field

	case *ast.ExprStmt:
		m["type"] = "expr"
		m["expr"] = convertExpr(st.Expression)

	case *ast.UseStmt:
		m["type"] = "use"
		m["module"] = st.Module

	case *ast.ImportStmt:
		m["type"] = "import"
		m["package"] = st.Package
		if st.Alias != "" {
			m["alias"] = st.Alias
		}

	case *ast.RequireStmt:
		m["type"] = "require"
		m["path"] = st.Path
		if st.Alias != "" {
			m["alias"] = st.Alias
		}
		if len(st.With) > 0 {
			with := make([]interface{}, len(st.With))
			for i, w := range st.With {
				with[i] = w
			}
			m["with"] = with
		}

	default:
		m["type"] = "unknown"
	}

	return m
}

func convertBody(stmts []ast.Statement) []interface{} {
	result := make([]interface{}, len(stmts))
	for i, s := range stmts {
		result[i] = convertStmt(s)
	}
	return result
}

func convertExpr(e ast.Expr) map[interface{}]interface{} {
	if e == nil {
		return map[interface{}]interface{}{"type": "nil"}
	}

	m := map[interface{}]interface{}{}

	switch ex := e.(type) {
	case *ast.CallExpr:
		m["type"] = "call"
		m["func"] = convertExpr(ex.Func)
		args := make([]interface{}, len(ex.Args))
		for i, a := range ex.Args {
			args[i] = convertExpr(a)
		}
		m["args"] = args

	case *ast.IdentExpr:
		m["type"] = "ident"
		m["name"] = ex.Name

	case *ast.DotExpr:
		m["type"] = "dot"
		m["object"] = convertExpr(ex.Object)
		m["field"] = ex.Field

	case *ast.StringLiteral:
		m["type"] = "string"
		m["value"] = ex.Value

	case *ast.IntLiteral:
		m["type"] = "int"
		m["value"] = ex.Value

	case *ast.FloatLiteral:
		m["type"] = "float"
		m["value"] = ex.Value

	case *ast.BoolLiteral:
		m["type"] = "bool"
		m["value"] = ex.Value

	case *ast.NilLiteral:
		m["type"] = "nil"

	case *ast.BinaryExpr:
		m["type"] = "binary"
		m["op"] = ex.Op
		m["left"] = convertExpr(ex.Left)
		m["right"] = convertExpr(ex.Right)

	case *ast.UnaryExpr:
		m["type"] = "unary"
		m["op"] = ex.Op

	case *ast.ArrayLiteral:
		m["type"] = "array"

	case *ast.HashLiteral:
		m["type"] = "hash"

	case *ast.IndexExpr:
		m["type"] = "index"

	default:
		m["type"] = "other"
	}

	return m
}
