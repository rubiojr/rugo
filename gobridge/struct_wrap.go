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
// struct type with DotGet and DotSet methods for a discovered struct.
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

	// DotCall method (stub — methods are Phase 2)
	sb.WriteString(fmt.Sprintf("func (w *%s) DotCall(method string, args ...interface{}) (interface{}, bool) {\n", wrapType))
	sb.WriteString("\treturn nil, false\n")
	sb.WriteString("}\n\n")

	return RuntimeHelper{
		Key:  wrapType,
		Code: sb.String(),
	}
}
