package gobridge

import (
	"fmt"
	"strings"
)

// GenerateUpcastHelper generates a RuntimeHelper for a rugo_upcast_<wrapType>
// function that extracts a *WrapType from any opaque handle, walking embedded
// fields via DotGet recursively. This enables auto-upcasting: passing a
// QPushButton where a QWidget is expected.
func GenerateUpcastHelper(wrapType string) RuntimeHelper {
	fnName := "rugo_upcast_" + wrapType

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("func %s(v interface{}) *%s {\n", fnName, wrapType))
	// Fast path: direct assertion.
	sb.WriteString(fmt.Sprintf("\tif w, ok := v.(*%s); ok { return w }\n", wrapType))
	// Walk DotGet fields recursively.
	sb.WriteString("\ttype dg interface{ DotGet(string) (interface{}, bool) }\n")
	sb.WriteString("\tif obj, ok := v.(dg); ok {\n")
	sb.WriteString("\t\tvar walk func(interface{}, int) *" + wrapType + "\n")
	sb.WriteString("\t\twalk = func(cur interface{}, depth int) *" + wrapType + " {\n")
	sb.WriteString("\t\t\tif depth > 10 { return nil }\n")
	sb.WriteString(fmt.Sprintf("\t\t\tif w, ok := cur.(*%s); ok { return w }\n", wrapType))
	sb.WriteString("\t\t\tif g, ok := cur.(dg); ok {\n")
	// Try all single-char lowercase + common embedded field patterns.
	// Instead of enumerating all possible fields, we use a generic approach:
	// the DotGet interface iterates all fields. We define a FieldEnumerator.
	sb.WriteString("\t\t\t\tif e, ok2 := g.(interface{ DotEnumFields() []string }); ok2 {\n")
	sb.WriteString("\t\t\t\t\tfor _, f := range e.DotEnumFields() {\n")
	sb.WriteString("\t\t\t\t\t\tif val, ok3 := g.DotGet(f); ok3 && val != nil {\n")
	sb.WriteString("\t\t\t\t\t\t\tif r := walk(val, depth+1); r != nil { return r }\n")
	sb.WriteString("\t\t\t\t\t\t}\n")
	sb.WriteString("\t\t\t\t\t}\n")
	sb.WriteString("\t\t\t\t}\n")
	sb.WriteString("\t\t\t}\n")
	sb.WriteString("\t\t\treturn nil\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tif r := walk(obj, 0); r != nil { return r }\n")
	sb.WriteString("\t}\n")
	sb.WriteString(fmt.Sprintf("\tpanic(fmt.Sprintf(\"cannot convert %%T to %s\", v))\n", wrapType))
	sb.WriteString("}\n\n")

	return RuntimeHelper{
		Key:  fnName,
		Code: sb.String(),
	}
}

// StructWrapperTypeName returns the Go wrapper type name for a struct.
// ns is the Rugo namespace (e.g., "mymod"), goName is the PascalCase struct name.
func StructWrapperTypeName(ns, goName string) string {
	return fmt.Sprintf("rugo_struct_%s_%s", ns, goName)
}

// ExternalOpaqueWrapperTypeName returns the Go wrapper type name for an external type.
// ns is the Rugo namespace, pkgName is the external package name, goName is the type name.
func ExternalOpaqueWrapperTypeName(ns, pkgName, goName string) string {
	return fmt.Sprintf("rugo_ext_%s_%s_%s", ns, pkgName, goName)
}

// GenerateExternalOpaqueWrapper generates a minimal RuntimeHelper for an external
// named type. The wrapper has DotGet (only __type__), empty DotSet, and empty DotCall.
// pkgName is the Go package name used to qualify the type (e.g., "qt6").
func GenerateExternalOpaqueWrapper(ns string, ext ExternalTypeInfo) RuntimeHelper {
	wrapType := ExternalOpaqueWrapperTypeName(ns, ext.PkgName, ext.GoName)
	goType := fmt.Sprintf("*%s.%s", ext.PkgName, ext.GoName)

	var sb strings.Builder

	// Wrapper struct
	sb.WriteString(fmt.Sprintf("type %s struct{ v %s }\n\n", wrapType, goType))

	// DotGet — __type__ + embedded struct fields
	sb.WriteString(fmt.Sprintf("func (w *%s) DotGet(field string) (interface{}, bool) {\n", wrapType))
	sb.WriteString("\tswitch field {\n")
	sb.WriteString("\tcase \"__type__\":\n")
	sb.WriteString(fmt.Sprintf("\t\treturn %q, true\n", ext.GoName))
	for _, f := range ext.EmbeddedFields {
		sb.WriteString(fmt.Sprintf("\tcase %q:\n", f.RugoName))
		if f.WrapType != "" {
			if f.WrapValue {
				sb.WriteString(fmt.Sprintf("\t\treturn interface{}(&%s{v: &w.v.%s}), true\n", f.WrapType, f.GoName))
			} else {
				sb.WriteString(fmt.Sprintf("\t\treturn interface{}(&%s{v: w.v.%s}), true\n", f.WrapType, f.GoName))
			}
		} else {
			sb.WriteString(fmt.Sprintf("\t\treturn %s, true\n", TypeWrapReturn("w.v."+f.GoName, f.Type)))
		}
	}
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn nil, false\n")
	sb.WriteString("}\n\n")

	// DotSet — no-op
	sb.WriteString(fmt.Sprintf("func (w *%s) DotSet(field string, val interface{}) bool {\n", wrapType))
	sb.WriteString("\treturn false\n")
	sb.WriteString("}\n\n")

	// DotEnumFields — list embedded field names for upcast walking.
	if len(ext.EmbeddedFields) > 0 {
		sb.WriteString(fmt.Sprintf("func (w *%s) DotEnumFields() []string {\n", wrapType))
		sb.WriteString("\treturn []string{")
		for i, f := range ext.EmbeddedFields {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%q", f.RugoName))
		}
		sb.WriteString("}\n")
		sb.WriteString("}\n\n")
	}

	// DotCall — dispatch to methods if discovered
	sb.WriteString(fmt.Sprintf("func (w *%s) DotCall(method string, args ...interface{}) (interface{}, bool) {\n", wrapType))
	if len(ext.Methods) > 0 {
		sb.WriteString("\tswitch method {\n")
		for _, m := range ext.Methods {
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
		if f.WrapType != "" {
			// Opaque struct field — wrap in the appropriate handle type.
			if f.WrapValue {
				sb.WriteString(fmt.Sprintf("\t\treturn interface{}(&%s{v: &w.v.%s}), true\n", f.WrapType, f.GoName))
			} else {
				sb.WriteString(fmt.Sprintf("\t\treturn interface{}(&%s{v: w.v.%s}), true\n", f.WrapType, f.GoName))
			}
		} else {
			sb.WriteString(fmt.Sprintf("\t\treturn %s, true\n", TypeWrapReturn("w.v."+f.GoName, f.Type)))
		}
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
		if f.WrapType != "" {
			continue // embedded struct fields are read-only
		}
		conv := TypeConvToGo("val", f.Type)
		if f.TypeCast != "" {
			if strings.HasPrefix(f.TypeCast, "*") {
				conv = fmt.Sprintf("*%s(%s)", f.TypeCast[1:], conv)
			} else {
				conv = fmt.Sprintf("%s(%s)", f.TypeCast, conv)
			}
		}
		sb.WriteString(fmt.Sprintf("\tcase %q:\n", f.RugoName))
		sb.WriteString(fmt.Sprintf("\t\tw.v.%s = %s\n", f.GoName, conv))
		sb.WriteString("\t\treturn true\n")
	}
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn false\n")
	sb.WriteString("}\n\n")

	// DotEnumFields — list embedded field names for upcast walking.
	var embeddedFields []GoStructFieldInfo
	for _, f := range si.Fields {
		if f.WrapType != "" {
			embeddedFields = append(embeddedFields, f)
		}
	}
	if len(embeddedFields) > 0 {
		sb.WriteString(fmt.Sprintf("func (w *%s) DotEnumFields() []string {\n", wrapType))
		sb.WriteString("\treturn []string{")
		for i, f := range embeddedFields {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%q", f.RugoName))
		}
		sb.WriteString("}\n")
		sb.WriteString("}\n\n")
	}

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
				// Use upcast helper to auto-extract from derived types.
				expr := fmt.Sprintf("rugo_upcast_%s(%s).v", wrapType, argExpr)
				// Value-type struct params need dereference (*wrapper.v).
				if m.StructParamValue != nil && m.StructParamValue[i] {
					expr = fmt.Sprintf("*rugo_upcast_%s(%s).v", wrapType, argExpr)
				}
				convArgs = append(convArgs, expr)
				continue
			}
		}
		// GoFunc adapter: wrap Rugo lambda into typed Go func.
		if p == GoFunc && m.FuncTypes != nil {
			if ft, ok := m.FuncTypes[i]; ok {
				conv := FuncAdapterConv(argExpr, ft)
				funcType := funcTypeName(ft)
				if m.TypeCasts != nil {
					if cast, ok := m.TypeCasts[i]; ok {
						funcType = cast
						conv = fmt.Sprintf("%s(%s)", cast, conv)
					}
				}
				if m.FuncParamPointer != nil && m.FuncParamPointer[i] {
					conv = fmt.Sprintf("func() *%s { _f := %s; return &_f }()", funcType, conv)
				}
				convArgs = append(convArgs, conv)
				continue
			}
		}
		conv := TypeConvToGo(argExpr, p)
		// Apply named type cast if needed (e.g., qt6.GestureType(rugo_to_int(arg))).
		if m.TypeCasts != nil {
			if cast, ok := m.TypeCasts[i]; ok {
				if strings.HasPrefix(cast, "*") {
					// Constructor returns pointer, param expects value — dereference.
					conv = fmt.Sprintf("*%s(%s)", cast[1:], conv)
				} else {
					conv = fmt.Sprintf("%s(%s)", cast, conv)
				}
			}
		}
		convArgs = append(convArgs, conv)
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
			// Value-type struct returns need address-of for the pointer wrapper.
			if m.StructReturnValue != nil && m.StructReturnValue[idx] {
				return fmt.Sprintf("func() interface{} { _sv := %s; return interface{}(&%s{v: &_sv}) }()", expr, wrapType)
			}
			return fmt.Sprintf("interface{}(&%s{v: %s})", wrapType, expr)
		}
	}
	if idx < len(m.Returns) {
		return TypeWrapReturn(expr, m.Returns[idx])
	}
	return "interface{}(" + expr + ")"
}
