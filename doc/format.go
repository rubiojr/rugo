package doc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rubiojr/rugo/compiler/gobridge"
	"github.com/rubiojr/rugo/modules"
)

// FormatFile formats a FileDoc for terminal display.
func FormatFile(fd *FileDoc) string {
	var sb strings.Builder

	if fd.Doc != "" {
		sb.WriteString(fd.Doc)
		sb.WriteString("\n\n")
	}

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

	return strings.TrimRight(sb.String(), "\n") + "\n"
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
			sb.WriteString(f.Doc)
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
			line += " " + m.Doc
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
			line += " " + pkg.Doc
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
		params = append(params, fmt.Sprintf("arg%d", i))
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
	for _, p := range sig.Params {
		params = append(params, gobridge.GoTypeName(p))
	}
	if sig.Variadic {
		last := len(params) - 1
		if last >= 0 {
			params[last] = params[last] + "..."
		}
	}

	var returns []string
	for _, r := range sig.Returns {
		if r == gobridge.GoError {
			continue // errors are auto-panicked, not visible to Rugo
		}
		returns = append(returns, gobridge.GoTypeName(r))
	}

	result := fmt.Sprintf("%s.%s(%s)", ns, name, strings.Join(params, ", "))
	if len(returns) > 0 {
		result += " -> " + strings.Join(returns, ", ")
	}

	return result
}
