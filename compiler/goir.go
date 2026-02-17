package compiler

import (
	"fmt"
	"strings"
)

// Go IR nodes represent the structure of generated Go code fragments.
// They separate "what to generate" from "how to format it", eliminating
// manual string builder code with hardcoded indentation.

// goNode is a piece of Go code that can be emitted with proper indentation.
type goNode interface {
	goNode()
}

// goRaw is a literal line of Go code.
type goRaw struct {
	Code string
}

func (goRaw) goNode() {}

// goBlock is a sequence of Go nodes emitted in order.
type goBlock struct {
	Nodes []goNode
}

func (goBlock) goNode() {}

// goIIFE represents a self-calling function: func() T { ... }()
type goIIFE struct {
	ReturnType string   // e.g. "interface{}", "(r interface{})"
	Body       []goNode // statements inside the IIFE
	Return     string   // final return expression (empty = omit)
}

func (goIIFE) goNode() {}

// goDefer represents: defer func() { ... }()
type goDefer struct {
	Body []goNode
}

func (goDefer) goNode() {}

// goGoroutine represents: go func() { ... }()
type goGoroutine struct {
	Body []goNode
}

func (goGoroutine) goNode() {}

// goIf represents: if cond { ... }
type goIf struct {
	Cond string
	Body []goNode
}

func (goIf) goNode() {}

// goUserCode is pre-generated Go code from codegen (e.g., handler body).
// Lines are re-indented to match the surrounding context.
type goUserCode struct {
	Code string
}

func (goUserCode) goNode() {}

// emitGoIR renders a Go IR node tree into a string suitable for use as
// an expression (e.g., the right side of an assignment). The first line
// has no indentation; subsequent lines are indented relative to g.w.indent.
func (g *codeGen) emitGoIR(node goNode) string {
	var sb strings.Builder
	emitNode(&sb, node, g.w.indent, true)
	// Trim trailing newline so caller controls line ending
	s := sb.String()
	return strings.TrimRight(s, "\n")
}

func emitNode(sb *strings.Builder, node goNode, indent int, firstLine bool) {
	switch n := node.(type) {
	case goRaw:
		if firstLine {
			sb.WriteString(n.Code)
		} else {
			writeIndented(sb, indent, n.Code)
		}
		sb.WriteByte('\n')

	case goBlock:
		for i, child := range n.Nodes {
			emitNode(sb, child, indent, firstLine && i == 0)
		}

	case goIIFE:
		retType := n.ReturnType
		if retType == "" {
			retType = "interface{}"
		}
		opening := fmt.Sprintf("func() %s {", retType)
		if firstLine {
			sb.WriteString(opening)
		} else {
			writeIndented(sb, indent, opening)
		}
		sb.WriteByte('\n')
		for _, child := range n.Body {
			emitNode(sb, child, indent+1, false)
		}
		if n.Return != "" {
			writeIndented(sb, indent+1, fmt.Sprintf("return %s", n.Return))
			sb.WriteByte('\n')
		}
		writeIndented(sb, indent, "}()")
		sb.WriteByte('\n')

	case goDefer:
		writeIndented(sb, indent, "defer func() {")
		sb.WriteByte('\n')
		for _, child := range n.Body {
			emitNode(sb, child, indent+1, false)
		}
		writeIndented(sb, indent, "}()")
		sb.WriteByte('\n')

	case goGoroutine:
		writeIndented(sb, indent, "go func() {")
		sb.WriteByte('\n')
		for _, child := range n.Body {
			emitNode(sb, child, indent+1, false)
		}
		writeIndented(sb, indent, "}()")
		sb.WriteByte('\n')

	case goIf:
		writeIndented(sb, indent, fmt.Sprintf("if %s {", n.Cond))
		sb.WriteByte('\n')
		for _, child := range n.Body {
			emitNode(sb, child, indent+1, false)
		}
		writeIndented(sb, indent, "}")
		sb.WriteByte('\n')

	case goUserCode:
		for _, line := range strings.Split(n.Code, "\n") {
			if line != "" {
				writeIndented(sb, indent, strings.TrimLeft(line, "\t"))
				sb.WriteByte('\n')
			}
		}
	}
}

func writeIndented(sb *strings.Builder, indent int, code string) {
	sb.WriteString(strings.Repeat("\t", indent))
	sb.WriteString(code)
}
