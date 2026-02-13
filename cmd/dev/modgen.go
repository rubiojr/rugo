// Package dev implements developer tooling subcommands for Rugo.
package dev

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/urfave/cli/v3"
)

// Command returns the "dev" CLI command group.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "dev",
		Usage: "Developer tools for Rugo",
		Commands: []*cli.Command{
			modgenCommand(),
			bridgegenCommand(),
		},
	}
}

func modgenCommand() *cli.Command {
	return &cli.Command{
		Name:      "modgen",
		Usage:     "Scaffold a new Rugo module",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "funcs",
				Usage: "Comma-separated function names (e.g. sprintf,printf)",
			},
		},
		Action: modgenAction,
	}
}

var validModName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func modgenAction(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return fmt.Errorf("usage: rugo dev modgen <name> [--funcs f1,f2,...]")
	}

	name := cmd.Args().First()
	if !validModName.MatchString(name) {
		return fmt.Errorf("invalid module name %q: must be lowercase alphanumeric with underscores", name)
	}

	funcs := parseFuncs(cmd.String("funcs"))
	dir := filepath.Join("modules", name)
	pkg := name + "mod"
	typeName := toPascalCase(name)

	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("directory %s already exists", dir)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data := modgenData{
		Name:     name,
		Pkg:      pkg,
		Type:     typeName,
		Funcs:    funcs,
		HasFuncs: len(funcs) > 0,
	}

	files := []struct {
		name string
		tmpl string
	}{
		{filepath.Join(dir, name+".go"), registrationTmpl},
		{filepath.Join(dir, "runtime.go"), runtimeTmpl},
		{filepath.Join(dir, "stubs.go"), stubsTmpl},
	}

	for _, f := range files {
		if err := writeTemplate(f.name, f.tmpl, data); err != nil {
			return err
		}
		fmt.Printf("Created %s\n", f.name)
	}

	if err := addBlankImport(name); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not update main.go: %v\n", err)
		fmt.Printf("Add manually: _ \"github.com/rubiojr/rugo/modules/%s\"\n", name)
	} else {
		fmt.Println("Added import to main.go")
	}

	fmt.Printf("\nFill in the method implementations in %s/runtime.go\n", dir)
	return nil
}

func parseFuncs(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var funcs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			funcs = append(funcs, p)
		}
	}
	return funcs
}

type modgenData struct {
	Name     string
	Pkg      string
	Type     string
	Funcs    []string
	HasFuncs bool
}

func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func writeTemplate(path, tmplStr string, data modgenData) error {
	t, err := template.New("").Funcs(template.FuncMap{
		"pascal": toPascalCase,
	}).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()

	return t.Execute(f, data)
}

func addBlankImport(name string) error {
	data, err := os.ReadFile("main.go")
	if err != nil {
		return err
	}

	importLine := fmt.Sprintf("\t_ \"github.com/rubiojr/rugo/modules/%s\"", name)
	content := string(data)

	// Check if already imported
	if strings.Contains(content, importLine) {
		return nil
	}

	// Find the last blank import line and insert after it
	lines := strings.Split(content, "\n")
	var result []string
	inserted := false
	for i, line := range lines {
		result = append(result, line)
		if !inserted && strings.Contains(line, "_ \"github.com/rubiojr/rugo/modules/") {
			// Check if next line is still an import or closing paren
			if i+1 < len(lines) {
				next := strings.TrimSpace(lines[i+1])
				if next == ")" || !strings.Contains(next, "_ \"github.com/rubiojr/rugo/modules/") {
					result = append(result, importLine)
					inserted = true
				}
			}
		}
	}

	if !inserted {
		return fmt.Errorf("could not find insertion point in main.go")
	}

	return os.WriteFile("main.go", []byte(strings.Join(result, "\n")), 0o644)
}

var registrationTmpl = `package {{.Pkg}}

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "{{.Name}}",
		Type: "{{.Type}}",
{{- if .HasFuncs}}
		Funcs: []modules.FuncDef{
{{- range .Funcs}}
			{Name: "{{.}}", Args: []modules.ArgType{modules.Any}},
{{- end}}
		},
{{- end}}
		Runtime: modules.CleanRuntime(runtime),
	})
}
`

var runtimeTmpl = `//go:build ignore

package {{.Pkg}}

// --- {{.Name}} module ---

type {{.Type}} struct{}
{{range .Funcs}}
func (*{{$.Type}}) {{. | pascal}}(val interface{}) interface{} {
	// TODO: implement
	return nil
}
{{end}}`

var stubsTmpl = `package {{.Pkg}}

import "fmt"

// Runtime helper stubs for standalone compilation and testing.

func rugo_to_string(v interface{}) string { return fmt.Sprintf("%v", v) }
`
