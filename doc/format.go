package doc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rubiojr/rugo/gobridge"
	"github.com/rubiojr/rugo/modules"
)

// FormatFile formats a FileDoc for terminal display.
// When items have Source set (from recursive extraction), they are grouped
// by source file with headers.
func FormatFile(fd *FileDoc) string {
	var sb strings.Builder

	if fd.Doc != "" {
		sb.WriteString(fd.Doc)
		sb.WriteString("\n\n")
	}

	// Check if any items have source info for grouping
	hasSource := false
	for _, f := range fd.Funcs {
		if f.Source != "" && f.Doc != "" {
			hasSource = true
			break
		}
	}
	if !hasSource {
		for _, s := range fd.Structs {
			if s.Source != "" && s.Doc != "" {
				hasSource = true
				break
			}
		}
	}

	if !hasSource {
		// Single-file mode: flat list
		for _, s := range fd.Structs {
			if s.Doc == "" {
				continue
			}
			formatStruct(&sb, s)
			sb.WriteString("\n")
		}
		for _, f := range fd.Funcs {
			if f.Doc == "" {
				continue
			}
			formatFunc(&sb, f)
			sb.WriteString("\n")
		}
	} else {
		// Multi-file mode: group by source
		sources := sourceOrder(fd)
		for _, src := range sources {
			sb.WriteString(src + ":\n\n")
			for _, s := range fd.Structs {
				if s.Source != src || s.Doc == "" {
					continue
				}
				sb.WriteString("  ")
				formatStruct(&sb, s)
				sb.WriteString("\n")
			}
			for _, f := range fd.Funcs {
				if f.Source != src || f.Doc == "" {
					continue
				}
				sb.WriteString("  ")
				formatFunc(&sb, f)
				sb.WriteString("\n")
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n") + "\n"
}

// sourceOrder returns the unique source paths in stable order,
// preserving the order they first appear in funcs/structs.
func sourceOrder(fd *FileDoc) []string {
	seen := make(map[string]bool)
	var order []string
	for _, s := range fd.Structs {
		if s.Source != "" && s.Doc != "" && !seen[s.Source] {
			seen[s.Source] = true
			order = append(order, s.Source)
		}
	}
	for _, f := range fd.Funcs {
		if f.Source != "" && f.Doc != "" && !seen[f.Source] {
			seen[f.Source] = true
			order = append(order, f.Source)
		}
	}
	return order
}

// FormatSymbol formats a single symbol lookup result.
func FormatSymbol(docStr, signature string) string {
	var sb strings.Builder
	sb.WriteString(signature)
	sb.WriteString("\n")
	if docStr != "" {
		sb.WriteString("    ")
		sb.WriteString(strings.ReplaceAll(docStr, "\n", "\n    "))
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatModule formats a stdlib module for terminal display.
func FormatModule(m *modules.Module) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("module %s", m.Name))
	sb.WriteString("\n")
	if m.Doc != "" {
		sb.WriteString("    ")
		sb.WriteString(m.Doc)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	for _, f := range m.Funcs {
		sig := formatModuleFuncSig(m.Name, f)
		sb.WriteString(sig)
		sb.WriteString("\n")
		if f.Doc != "" {
			sb.WriteString("    ")
			sb.WriteString(strings.ReplaceAll(f.Doc, "\n", "\n    "))
			sb.WriteString("\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n") + "\n"
}

// FormatBridgePackage formats a Go bridge package for terminal display.
func FormatBridgePackage(pkg *gobridge.Package) string {
	var sb strings.Builder

	ns := gobridge.DefaultNS(pkg.Path)
	sb.WriteString(fmt.Sprintf("package %s (Go: %s)", ns, pkg.Path))
	sb.WriteString("\n")
	if pkg.Doc != "" {
		sb.WriteString("    ")
		sb.WriteString(pkg.Doc)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Show structs first
	for _, si := range pkg.Structs {
		if len(si.Fields) > 0 {
			sb.WriteString(fmt.Sprintf("struct %s { ", si.GoName))
			var fields []string
			for _, f := range si.Fields {
				fields = append(fields, fmt.Sprintf("%s: %s", f.RugoName, gobridge.GoTypeName(f.Type)))
			}
			sb.WriteString(strings.Join(fields, ", "))
			sb.WriteString(" }\n")
		} else {
			sb.WriteString(fmt.Sprintf("struct %s {}\n", si.GoName))
		}
		// Show methods indented under the struct
		for _, m := range si.Methods {
			sb.WriteString(fmt.Sprintf("  .%s", formatMethodSig(m)))
			sb.WriteString("\n")
		}
	}
	if len(pkg.Structs) > 0 {
		sb.WriteString("\n")
	}

	// Sort function names for stable output
	names := make([]string, 0, len(pkg.Funcs))
	for name := range pkg.Funcs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		sig := pkg.Funcs[name]
		sigStr := formatBridgeFuncSig(ns, name, sig)
		sb.WriteString(sigStr)
		sb.WriteString("\n")
		if sig.Doc != "" {
			sb.WriteString("    ")
			sb.WriteString(sig.Doc)
			sb.WriteString("\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n") + "\n"
}

// FormatAllModules lists all available modules and bridge packages.
func FormatAllModules() string {
	var sb strings.Builder

	sb.WriteString("Modules (use):\n")
	for _, name := range modules.Names() {
		m, _ := modules.Get(name)
		line := fmt.Sprintf("  %-12s", name)
		if m.Doc != "" {
			first, _, _ := strings.Cut(m.Doc, "\n")
			line += " " + first
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\nBridge packages (import):\n")
	for _, path := range gobridge.PackageNames() {
		ns := gobridge.DefaultNS(path)
		pkg := gobridge.GetPackage(path)
		line := fmt.Sprintf("  %-12s", ns)
		if pkg != nil && pkg.Doc != "" {
			first, _, _ := strings.Cut(pkg.Doc, "\n")
			line += " " + first
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatStruct(sb *strings.Builder, s StructDoc) {
	sb.WriteString(fmt.Sprintf("struct %s", s.Name))
	if len(s.Fields) > 0 {
		sb.WriteString(" { ")
		sb.WriteString(strings.Join(s.Fields, ", "))
		sb.WriteString(" }")
	}
	sb.WriteString("\n")
	if s.Doc != "" {
		sb.WriteString("    ")
		sb.WriteString(strings.ReplaceAll(s.Doc, "\n", "\n    "))
		sb.WriteString("\n")
	}
}

func formatFunc(sb *strings.Builder, f FuncDoc) {
	sb.WriteString("def ")
	sb.WriteString(f.Name)
	if len(f.Params) > 0 {
		sb.WriteString("(")
		sb.WriteString(strings.Join(f.Params, ", "))
		sb.WriteString(")")
	}
	sb.WriteString("\n")
	if f.Doc != "" {
		sb.WriteString("    ")
		sb.WriteString(strings.ReplaceAll(f.Doc, "\n", "\n    "))
		sb.WriteString("\n")
	}
}

func formatModuleFuncSig(modName string, f modules.FuncDef) string {
	var params []string
	for i, a := range f.Args {
		if i < len(f.ArgNames) {
			params = append(params, f.ArgNames[i])
		} else {
			params = append(params, fmt.Sprintf("arg%d", i))
		}
		_ = a
	}
	if f.Variadic {
		params = append(params, "...")
	}
	sig := fmt.Sprintf("%s.%s(%s)", modName, f.Name, strings.Join(params, ", "))
	return sig
}

func formatBridgeFuncSig(ns, name string, sig gobridge.GoFuncSig) string {
	var params []string
	for i, p := range sig.Params {
		// Show struct type name instead of placeholder for struct params.
		if sig.StructCasts != nil {
			if wrapType, ok := sig.StructCasts[i]; ok {
				params = append(params, structNameFromWrapper(wrapType))
				continue
			}
		}
		params = append(params, gobridge.GoTypeName(p))
	}
	if sig.Variadic {
		last := len(params) - 1
		if last >= 0 {
			params[last] = params[last] + "..."
		}
	}

	var returns []string
	for i, r := range sig.Returns {
		if r == gobridge.GoError {
			continue // errors are auto-panicked, not visible to Rugo
		}
		// Show struct type name instead of placeholder for struct returns.
		if sig.StructReturnWraps != nil {
			if wrapType, ok := sig.StructReturnWraps[i]; ok {
				returns = append(returns, structNameFromWrapper(wrapType))
				continue
			}
		}
		returns = append(returns, gobridge.GoTypeName(r))
	}

	// Hide constructors with Codegen (zero-value constructors show their return type).
	if sig.Codegen != nil {
		result := fmt.Sprintf("%s.%s(%s)", ns, name, strings.Join(params, ", "))
		result += " -> " + sig.GoName
		return result
	}

	result := fmt.Sprintf("%s.%s(%s)", ns, name, strings.Join(params, ", "))
	if len(returns) > 0 {
		result += " -> " + strings.Join(returns, ", ")
	}

	return result
}

// structNameFromWrapper extracts the Go struct name from a wrapper type name.
// "rugo_struct_mymod_Config" → "Config"
func structNameFromWrapper(wrapType string) string {
	parts := strings.Split(wrapType, "_")
	if len(parts) >= 4 {
		return parts[len(parts)-1]
	}
	return wrapType
}

// formatMethodSig formats a struct method signature for doc display.
func formatMethodSig(m gobridge.GoStructMethodInfo) string {
	var params []string
	for i, p := range m.Params {
		if m.StructCasts != nil {
			if wrapType, ok := m.StructCasts[i]; ok {
				params = append(params, structNameFromWrapper(wrapType))
				continue
			}
		}
		params = append(params, gobridge.GoTypeName(p))
	}

	var returns []string
	for i, r := range m.Returns {
		if r == gobridge.GoError {
			continue
		}
		if m.StructReturnWraps != nil {
			if wrapType, ok := m.StructReturnWraps[i]; ok {
				returns = append(returns, structNameFromWrapper(wrapType))
				continue
			}
		}
		returns = append(returns, gobridge.GoTypeName(r))
	}

	result := fmt.Sprintf("%s(%s)", m.RugoName, strings.Join(params, ", "))
	if len(returns) > 0 {
		result += " -> " + strings.Join(returns, ", ")
	}
	return result
}
