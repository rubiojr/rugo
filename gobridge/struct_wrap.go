package gobridge

import (
	"fmt"
	"strings"
)

// StructWrapperTypeName returns the Go wrapper type name for a struct.
// ns is the Rugo namespace (e.g., "mymod"), goName is the PascalCase struct name.
func StructWrapperTypeName(ns, goName string) string {
	return fmt.Sprintf("rugo_struct_%s_%s", ns, goName)
}

// GenerateStructWrapper generates a RuntimeHelper containing the Go wrapper
// struct type with DotGet, DotSet, and DotCall methods for a discovered struct.
// pkgAlias is the Go package alias used in generated code (e.g., "mymod").
func GenerateStructWrapper(ns, pkgAlias string, si GoStructInfo) RuntimeHelper {
	wrapType := StructWrapperTypeName(ns, si.GoName)
	goType := fmt.Sprintf("*%s.%s", pkgAlias, si.GoName)

	var sb strings.Builder

	// Wrapper struct
	sb.WriteString(fmt.Sprintf("type %s struct{ v %s }\n\n", wrapType, goType))

	// DotGet method
	sb.WriteString(fmt.Sprintf("func (w *%s) DotGet(field string) (interface{}, bool) {\n", wrapType))
	sb.WriteString("\tswitch field {\n")
	for _, f := range si.Fields {
		sb.WriteString(fmt.Sprintf("\tcase %q:\n", f.RugoName))
		sb.WriteString(fmt.Sprintf("\t\treturn %s, true\n", TypeWrapReturn("w.v."+f.GoName, f.Type)))
	}
	sb.WriteString("\tcase \"__type__\":\n")
	sb.WriteString(fmt.Sprintf("\t\treturn %q, true\n", si.GoName))
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn nil, false\n")
	sb.WriteString("}\n\n")

	// DotSet method
	sb.WriteString(fmt.Sprintf("func (w *%s) DotSet(field string, val interface{}) bool {\n", wrapType))
	sb.WriteString("\tswitch field {\n")
	for _, f := range si.Fields {
		sb.WriteString(fmt.Sprintf("\tcase %q:\n", f.RugoName))
		sb.WriteString(fmt.Sprintf("\t\tw.v.%s = %s\n", f.GoName, TypeConvToGo("val", f.Type)))
		sb.WriteString("\t\treturn true\n")
	}
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn false\n")
	sb.WriteString("}\n\n")

	// DotCall method — dispatch to real Go methods
	sb.WriteString(fmt.Sprintf("func (w *%s) DotCall(method string, args ...interface{}) (interface{}, bool) {\n", wrapType))
	if len(si.Methods) > 0 {
		sb.WriteString("\tswitch method {\n")
		for _, m := range si.Methods {
			sb.WriteString(fmt.Sprintf("\tcase %q:\n", m.RugoName))
			writeMethodCase(&sb, m, ns)
		}
		sb.WriteString("\t}\n")
	}
	sb.WriteString("\treturn nil, false\n")
	sb.WriteString("}\n\n")

	return RuntimeHelper{
		Key:  wrapType,
		Code: sb.String(),
	}
}

// writeMethodCase writes the body for a single DotCall case.
func writeMethodCase(sb *strings.Builder, m GoStructMethodInfo, ns string) {
	rugoName := ns + "." + m.RugoName

	// Build converted args
	var convArgs []string
	for i, p := range m.Params {
		argExpr := fmt.Sprintf("args[%d]", i)
		if m.StructCasts != nil {
			if wrapType, ok := m.StructCasts[i]; ok {
				convArgs = append(convArgs, fmt.Sprintf("%s.(*%s).v", argExpr, wrapType))
				continue
			}
		}
		convArgs = append(convArgs, TypeConvToGo(argExpr, p))
	}
	callArgs := strings.Join(convArgs, ", ")
	call := fmt.Sprintf("w.v.%s(%s)", m.GoName, callArgs)

	// Handle return patterns
	returns := m.Returns
	if len(returns) == 0 {
		// Void
		sb.WriteString(fmt.Sprintf("\t\t%s\n", call))
		sb.WriteString("\t\treturn nil, true\n")
		return
	}

	if len(returns) == 1 && returns[0] == GoError {
		// (error) — panic on non-nil
		sb.WriteString(fmt.Sprintf("\t\tif _err := %s; _err != nil { panic(rugo_bridge_err(%q, _err)) }\n", call, rugoName))
		sb.WriteString("\t\treturn nil, true\n")
		return
	}

	if len(returns) == 2 && returns[1] == GoError {
		// (T, error) — panic on error, return T
		sb.WriteString(fmt.Sprintf("\t\t_v, _err := %s\n", call))
		sb.WriteString(fmt.Sprintf("\t\tif _err != nil { panic(rugo_bridge_err(%q, _err)) }\n", rugoName))
		sb.WriteString(fmt.Sprintf("\t\treturn %s, true\n", wrapMethodReturn("_v", 0, m)))
		return
	}

	if len(returns) == 1 {
		// (T) — return wrapped value
		sb.WriteString(fmt.Sprintf("\t\treturn %s, true\n", wrapMethodReturn(call, 0, m)))
		return
	}

	// Multi-return: collect into []interface{}
	vars := make([]string, len(returns))
	for i := range vars {
		vars[i] = fmt.Sprintf("_v%d", i)
	}
	sb.WriteString(fmt.Sprintf("\t\t%s := %s\n", strings.Join(vars, ", "), call))
	var elems []string
	for i, v := range vars {
		elems = append(elems, wrapMethodReturn(v, i, m))
	}
	sb.WriteString(fmt.Sprintf("\t\treturn []interface{}{%s}, true\n", strings.Join(elems, ", ")))
}

// wrapMethodReturn wraps a Go return value for a method.
func wrapMethodReturn(expr string, idx int, m GoStructMethodInfo) string {
	if m.StructReturnWraps != nil {
		if wrapType, ok := m.StructReturnWraps[idx]; ok {
			return fmt.Sprintf("interface{}(&%s{v: %s})", wrapType, expr)
		}
	}
	if idx < len(m.Returns) {
		return TypeWrapReturn(expr, m.Returns[idx])
	}
	return "interface{}(" + expr + ")"
}
