package modules

import (
	"fmt"
	"sort"
	"strings"
)

// ArgType represents the expected type of a function argument.
type ArgType int

const (
	String ArgType = iota
	Int
	Float
	Bool
	Any
)

// FuncDef describes a function exposed by a module.
// The implementation function must be named <module>_<Name> in runtime.go
// and use typed parameters matching Args (e.g. func os_exec(command string) interface{}).
type FuncDef struct {
	// Name is the rugo function name (e.g. "exec").
	Name string
	// Args lists the expected typed arguments. The wrapper will convert
	// interface{} args to these types before calling the implementation.
	Args []ArgType
	// Variadic, when true, passes remaining args beyond Args as ...interface{}.
	// The implementation function should accept extra ...interface{} as its last parameter.
	Variadic bool
}

// Module represents a Rugo stdlib module that can be imported.
type Module struct {
	// Name is the rugo import name (e.g. "os", "http", "conv").
	Name string
	// Type is the Go struct type name used as the method receiver (e.g. "OS", "HTTP").
	Type string
	// Funcs describes the functions this module exposes.
	Funcs []FuncDef
	// GoImports lists additional Go imports this module needs beyond the base set.
	GoImports []string
	// Runtime is the Go source for the struct type and its methods (from embedded runtime.go).
	Runtime string
}

var registry = make(map[string]*Module)

// Register adds a module to the global registry.
func Register(m *Module) {
	registry[m.Name] = m
}

// Get returns a registered module by name.
func Get(name string) (*Module, bool) {
	m, ok := registry[name]
	return m, ok
}

// IsModule returns true if name is a registered module.
func IsModule(name string) bool {
	_, ok := registry[name]
	return ok
}

// LookupFunc resolves a module function to its Go runtime wrapper name.
func LookupFunc(module, funcName string) (string, bool) {
	m, ok := registry[module]
	if !ok {
		return "", false
	}
	for _, f := range m.Funcs {
		if f.Name == funcName {
			return fmt.Sprintf("rugo_%s_%s", m.Name, f.Name), true
		}
	}
	return "", false
}

// Names returns sorted names of all registered modules.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// CleanRuntime strips //go:build directives and the package declaration
// from embedded Go source so it can be emitted into a generated program.
func CleanRuntime(src string) string {
	lines := strings.Split(src, "\n")
	var result []string
	started := false
	for _, line := range lines {
		if !started {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "//go:build") || strings.HasPrefix(trimmed, "package ") {
				continue
			}
			started = true
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// FullRuntime returns the complete runtime source: the struct type and methods
// from Runtime plus auto-generated wrappers that handle interface{} conversion.
func (m *Module) FullRuntime() string {
	var sb strings.Builder
	if m.Runtime != "" {
		sb.WriteString(m.Runtime)
		if !strings.HasSuffix(m.Runtime, "\n") {
			sb.WriteString("\n")
		}
	}

	if len(m.Funcs) > 0 {
		sb.WriteString(fmt.Sprintf("var _%s = &%s{}\n\n", m.Name, m.Type))
	}

	for _, f := range m.Funcs {
		wrapperName := fmt.Sprintf("rugo_%s_%s", m.Name, f.Name)
		methodName := toPascalCase(f.Name)
		minArgs := len(f.Args)
		instanceVar := fmt.Sprintf("_%s", m.Name)

		sb.WriteString(fmt.Sprintf("func %s(args ...interface{}) interface{} {\n", wrapperName))

		if minArgs > 0 {
			sb.WriteString(fmt.Sprintf(
				"\tif len(args) < %d { panic(\"%s.%s: requires at least %d argument(s)\") }\n",
				minArgs, m.Name, f.Name, minArgs))
		}

		var callArgs []string
		for i, argType := range f.Args {
			callArgs = append(callArgs, argConversion(i, argType))
		}
		callStr := strings.Join(callArgs, ", ")

		call := fmt.Sprintf("%s.%s", instanceVar, methodName)
		if f.Variadic {
			if minArgs > 0 {
				sb.WriteString(fmt.Sprintf("\treturn %s(%s, args[%d:]...)\n", call, callStr, minArgs))
			} else {
				sb.WriteString(fmt.Sprintf("\treturn %s(args...)\n", call))
			}
		} else {
			sb.WriteString(fmt.Sprintf("\treturn %s(%s)\n", call, callStr))
		}

		sb.WriteString("}\n\n")
	}

	return sb.String()
}

func argConversion(index int, t ArgType) string {
	switch t {
	case String:
		return fmt.Sprintf("rugo_to_string(args[%d])", index)
	case Int:
		return fmt.Sprintf("rugo_to_int(args[%d])", index)
	case Float:
		return fmt.Sprintf("rugo_to_float(args[%d])", index)
	case Bool:
		return fmt.Sprintf("rugo_to_bool(args[%d])", index)
	default:
		return fmt.Sprintf("args[%d]", index)
	}
}

// toPascalCase converts a snake_case name to PascalCase.
// "get" → "Get", "to_s" → "ToS", "exec" → "Exec"
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
