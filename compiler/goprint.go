package compiler

import (
	"fmt"
	"strings"
)

// PrintGoFile serializes a GoFile tree to formatted Go source code.
func PrintGoFile(f *GoFile) string {
	p := &goPrinter{}
	p.printFile(f)
	return p.sb.String()
}

type goPrinter struct {
	sb     strings.Builder
	indent int
}

func (p *goPrinter) line(format string, args ...any) {
	p.writeIndent()
	fmt.Fprintf(&p.sb, format, args...)
	p.sb.WriteByte('\n')
}

func (p *goPrinter) raw(s string) {
	p.sb.WriteString(s)
}

func (p *goPrinter) blank() {
	p.sb.WriteByte('\n')
}

func (p *goPrinter) writeIndent() {
	for range p.indent {
		p.sb.WriteByte('\t')
	}
}

func (p *goPrinter) printFile(f *GoFile) {
	p.line("package %s", f.Package)
	p.blank()

	if len(f.Imports) > 0 {
		p.line("import (")
		p.indent++
		for _, imp := range f.Imports {
			if imp.Alias != "" {
				p.line("%s %q", imp.Alias, imp.Path)
			} else {
				p.line("%q", imp.Path)
			}
		}
		p.indent--
		p.line(")")
		p.blank()
	}

	for _, d := range f.Decls {
		p.printDecl(d)
	}

	if f.Init != nil {
		p.line("func main() {")
		p.indent++
		for _, s := range f.Init {
			p.printStmt(s)
		}
		p.indent--
		p.line("}")
	}
}

func (p *goPrinter) printDecl(d GoDecl) {
	switch dt := d.(type) {
	case GoVarDecl:
		if dt.Value != nil {
			p.line("var %s %s = %s", dt.Name, dt.Type, p.exprStr(dt.Value))
		} else {
			p.line("var %s %s", dt.Name, dt.Type)
		}
	case GoFuncDecl:
		p.printFuncDecl(dt)
	case GoRawDecl:
		p.raw(dt.Code)
	case GoBlankLine:
		p.blank()
	case GoComment:
		p.line("// %s", dt.Text)
	}
}

func (p *goPrinter) printFuncDecl(f GoFuncDecl) {
	var params []string
	for _, param := range f.Params {
		params = append(params, fmt.Sprintf("%s %s", param.Name, param.Type))
	}
	sig := fmt.Sprintf("func %s(%s)", f.Name, strings.Join(params, ", "))
	if f.Return != "" {
		sig += " " + f.Return
	}
	p.line("%s {", sig)
	p.indent++
	for _, s := range f.Body {
		p.printStmt(s)
	}
	p.indent--
	p.line("}")
	p.blank()
}

func (p *goPrinter) printStmt(s GoStmt) {
	switch st := s.(type) {
	case GoExprStmt:
		p.line("%s", p.exprStr(st.Expr))
	case GoAssignStmt:
		p.line("%s %s %s", st.Target, st.Op, p.exprStr(st.Value))
	case GoMultiAssignStmt:
		p.line("%s %s %s", strings.Join(st.Targets, ", "), st.Op, p.exprStr(st.Value))
	case GoReturnStmt:
		if st.Value != nil {
			p.line("return %s", p.exprStr(st.Value))
		} else {
			p.line("return")
		}
	case GoVarStmt:
		if st.Value != nil {
			p.line("var %s %s = %s", st.Name, st.Type, p.exprStr(st.Value))
		} else {
			p.line("var %s %s", st.Name, st.Type)
		}
	case GoIfStmt:
		p.printIf(st)
	case GoForStmt:
		p.printFor(st)
	case GoForRangeStmt:
		p.printForRange(st)
	case GoSwitchStmt:
		p.printSwitch(st)
	case GoDeferStmt:
		p.line("defer func() {")
		p.indent++
		for _, s := range st.Body {
			p.printStmt(s)
		}
		p.indent--
		p.line("}()")
	case GoGoStmt:
		p.line("go func() {")
		p.indent++
		for _, s := range st.Body {
			p.printStmt(s)
		}
		p.indent--
		p.line("}()")
	case GoBreakStmt:
		p.line("break")
	case GoContinueStmt:
		p.line("continue")
	case GoBlankLine:
		p.blank()
	case GoLineDirective:
		// //line directives must start at column 1 (no indentation)
		fmt.Fprintf(&p.sb, "//line %s:%d\n", st.File, st.Line)
	case GoComment:
		p.line("// %s", st.Text)
	case GoRawStmt:
		// Raw code may contain multiple lines; re-indent each.
		for _, ln := range strings.Split(strings.TrimRight(st.Code, "\n"), "\n") {
			if ln == "" {
				p.blank()
			} else {
				p.writeIndent()
				p.sb.WriteString(strings.TrimLeft(ln, "\t"))
				p.sb.WriteByte('\n')
			}
		}
	}
}

func (p *goPrinter) printIf(st GoIfStmt) {
	p.line("if %s {", p.exprStr(st.Cond))
	p.indent++
	for _, s := range st.Body {
		p.printStmt(s)
	}
	p.indent--
	for _, ei := range st.ElseIf {
		p.line("} else if %s {", p.exprStr(ei.Cond))
		p.indent++
		for _, s := range ei.Body {
			p.printStmt(s)
		}
		p.indent--
	}
	if len(st.Else) > 0 {
		p.line("} else {")
		p.indent++
		for _, s := range st.Else {
			p.printStmt(s)
		}
		p.indent--
	}
	p.line("}")
}

func (p *goPrinter) printFor(st GoForStmt) {
	if st.Init != "" {
		p.line("for %s; %s; %s {", st.Init, st.Cond, st.Post)
	} else if st.Cond != "" {
		p.line("for %s {", st.Cond)
	} else {
		p.line("for {")
	}
	p.indent++
	for _, s := range st.Body {
		p.printStmt(s)
	}
	p.indent--
	p.line("}")
}

func (p *goPrinter) printForRange(st GoForRangeStmt) {
	if st.Value != "" {
		p.line("for %s, %s := range %s {", st.Key, st.Value, p.exprStr(st.Collection))
	} else {
		p.line("for %s := range %s {", st.Key, p.exprStr(st.Collection))
	}
	p.indent++
	for _, s := range st.Body {
		p.printStmt(s)
	}
	p.indent--
	p.line("}")
}

func (p *goPrinter) printSwitch(st GoSwitchStmt) {
	if st.Tag != nil {
		p.line("switch %s {", p.exprStr(st.Tag))
	} else {
		p.line("switch {")
	}
	for _, c := range st.Cases {
		var vals []string
		for _, v := range c.Values {
			vals = append(vals, p.exprStr(v))
		}
		p.line("case %s:", strings.Join(vals, ", "))
		p.indent++
		for _, s := range c.Body {
			p.printStmt(s)
		}
		p.indent--
	}
	if st.Default != nil {
		p.line("default:")
		p.indent++
		for _, s := range st.Default {
			p.printStmt(s)
		}
		p.indent--
	}
	p.line("}")
}

func (p *goPrinter) exprStr(e GoExpr) string {
	switch ex := e.(type) {
	case GoRawExpr:
		return ex.Code
	case GoIIFEExpr:
		return p.printIIFE(ex)
	case GoIdentExpr:
		return ex.Name
	case GoIntLit:
		return ex.Value
	case GoFloatLit:
		return ex.Value
	case GoStringLit:
		return fmt.Sprintf(`"%s"`, ex.Value)
	case GoBoolLit:
		if ex.Value {
			return "true"
		}
		return "false"
	case GoNilExpr:
		return "nil"
	case GoBinaryExpr:
		return fmt.Sprintf("%s %s %s", p.exprStr(ex.Left), ex.Op, p.exprStr(ex.Right))
	case GoUnaryExpr:
		return fmt.Sprintf("%s%s", ex.Op, p.exprStr(ex.Operand))
	case GoCastExpr:
		return fmt.Sprintf("%s(%s)", ex.Type, p.exprStr(ex.Value))
	case GoTypeAssert:
		return fmt.Sprintf("%s.(%s)", p.exprStr(ex.Value), ex.Type)
	case GoCallExpr:
		args := make([]string, len(ex.Args))
		for i, a := range ex.Args {
			args[i] = p.exprStr(a)
		}
		return fmt.Sprintf("%s(%s)", ex.Func, strings.Join(args, ", "))
	case GoMethodCallExpr:
		args := make([]string, len(ex.Args))
		for i, a := range ex.Args {
			args[i] = p.exprStr(a)
		}
		return fmt.Sprintf("%s.%s(%s)", p.exprStr(ex.Object), ex.Method, strings.Join(args, ", "))
	case GoDotExpr:
		return fmt.Sprintf("%s.%s", p.exprStr(ex.Object), ex.Field)
	case GoSliceLit:
		elems := make([]string, len(ex.Elements))
		for i, el := range ex.Elements {
			elems[i] = p.exprStr(el)
		}
		return fmt.Sprintf("%s{%s}", ex.Type, strings.Join(elems, ", "))
	case GoMapLit:
		pairs := make([]string, len(ex.Pairs))
		for i, pair := range ex.Pairs {
			pairs[i] = fmt.Sprintf("%s: %s", p.exprStr(pair.Key), p.exprStr(pair.Value))
		}
		return fmt.Sprintf("map[%s]%s{%s}", ex.KeyType, ex.ValType, strings.Join(pairs, ", "))
	case GoFmtSprintf:
		args := make([]string, len(ex.Args))
		for i, a := range ex.Args {
			args[i] = p.exprStr(a)
		}
		if len(args) > 0 {
			return fmt.Sprintf(`fmt.Sprintf("%s", %s)`, ex.Format, strings.Join(args, ", "))
		}
		return fmt.Sprintf(`fmt.Sprintf("%s")`, ex.Format)
	case GoStringConcat:
		parts := make([]string, len(ex.Parts))
		for i, part := range ex.Parts {
			parts[i] = p.exprStr(part)
		}
		return strings.Join(parts, " + ")
	case GoIndexExpr:
		return fmt.Sprintf("%s[%s]", p.exprStr(ex.Object), p.exprStr(ex.Index))
	case GoParenExpr:
		return fmt.Sprintf("(%s)", p.exprStr(ex.Inner))
	default:
		return "<unknown expr>"
	}
}

func (p *goPrinter) printIIFE(e GoIIFEExpr) string {
	retType := e.ReturnType
	if retType == "" {
		retType = "interface{}"
	}

	// Build IIFE as a string, respecting current indent for nested lines.
	var sb strings.Builder
	fmt.Fprintf(&sb, "func() %s {\n", retType)
	inner := &goPrinter{indent: p.indent + 1}
	for _, s := range e.Body {
		inner.printStmt(s)
	}
	if e.Result != nil {
		inner.line("return %s", inner.exprStr(e.Result))
	}
	sb.WriteString(inner.sb.String())
	p.writeIndent()
	sb.WriteString("}()")
	return sb.String()
}
