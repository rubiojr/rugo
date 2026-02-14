package doc

import (
	"os"
	"strings"
	"testing"

	"github.com/rubiojr/rugo/gobridge"
)

func TestExtract_FileDoc(t *testing.T) {
	src := `# File-level documentation.
# Second line.

def foo()
end
`
	fd := Extract(src, "test.rugo")
	if fd.Doc != "File-level documentation.\nSecond line." {
		t.Errorf("file doc = %q", fd.Doc)
	}
}

func TestExtract_FuncDoc(t *testing.T) {
	src := `# Adds two numbers.
def add(a, b)
  return a + b
end
`
	fd := Extract(src, "test.rugo")
	if len(fd.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(fd.Funcs))
	}
	f := fd.Funcs[0]
	if f.Name != "add" {
		t.Errorf("name = %q", f.Name)
	}
	if f.Doc != "Adds two numbers." {
		t.Errorf("doc = %q", f.Doc)
	}
	if len(f.Params) != 2 || f.Params[0] != "a" || f.Params[1] != "b" {
		t.Errorf("params = %v", f.Params)
	}
	if f.Line != 2 {
		t.Errorf("line = %d", f.Line)
	}
}

func TestExtract_StructDoc(t *testing.T) {
	src := `# A Dog with a name.
struct Dog
  name
  breed
end
`
	fd := Extract(src, "test.rugo")
	if len(fd.Structs) != 1 {
		t.Fatalf("expected 1 struct, got %d", len(fd.Structs))
	}
	s := fd.Structs[0]
	if s.Name != "Dog" {
		t.Errorf("name = %q", s.Name)
	}
	if s.Doc != "A Dog with a name." {
		t.Errorf("doc = %q", s.Doc)
	}
	if len(s.Fields) != 2 || s.Fields[0] != "name" || s.Fields[1] != "breed" {
		t.Errorf("fields = %v", s.Fields)
	}
}

func TestExtract_BlankLineBreaksAttachment(t *testing.T) {
	src := `# This is orphaned.

def foo()
end
`
	fd := Extract(src, "test.rugo")
	if len(fd.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(fd.Funcs))
	}
	if fd.Funcs[0].Doc != "" {
		t.Errorf("expected empty doc, got %q", fd.Funcs[0].Doc)
	}
	if fd.Doc != "This is orphaned." {
		t.Errorf("file doc = %q", fd.Doc)
	}
}

func TestExtract_UndocumentedItems(t *testing.T) {
	src := `def foo()
end

def bar(x)
end
`
	fd := Extract(src, "test.rugo")
	if len(fd.Funcs) != 2 {
		t.Fatalf("expected 2 funcs, got %d", len(fd.Funcs))
	}
	for _, f := range fd.Funcs {
		if f.Doc != "" {
			t.Errorf("%s has unexpected doc: %q", f.Name, f.Doc)
		}
	}
}

func TestExtract_MethodDoc(t *testing.T) {
	src := `# Makes the dog bark.
def Dog.bark()
  return "woof"
end
`
	fd := Extract(src, "test.rugo")
	if len(fd.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(fd.Funcs))
	}
	if fd.Funcs[0].Name != "Dog.bark" {
		t.Errorf("name = %q", fd.Funcs[0].Name)
	}
	if fd.Funcs[0].Doc != "Makes the dog bark." {
		t.Errorf("doc = %q", fd.Funcs[0].Doc)
	}
}

func TestExtract_HeredocSkipsComments(t *testing.T) {
	src := `msg = <<HEREDOC
# This is NOT a comment
HEREDOC

# Real doc.
def foo()
end
`
	fd := Extract(src, "test.rugo")
	if len(fd.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(fd.Funcs))
	}
	if fd.Funcs[0].Doc != "Real doc." {
		t.Errorf("doc = %q", fd.Funcs[0].Doc)
	}
}

func TestExtract_MultilineDoc(t *testing.T) {
	src := `# Line one.
# Line two.
# Line three.
def foo()
end
`
	fd := Extract(src, "test.rugo")
	expected := "Line one.\nLine two.\nLine three."
	if fd.Funcs[0].Doc != expected {
		t.Errorf("doc = %q, want %q", fd.Funcs[0].Doc, expected)
	}
}

func TestExtract_NoParenDef(t *testing.T) {
	src := `# No params.
def greet
  puts "hello"
end
`
	fd := Extract(src, "test.rugo")
	if len(fd.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(fd.Funcs))
	}
	if fd.Funcs[0].Name != "greet" {
		t.Errorf("name = %q", fd.Funcs[0].Name)
	}
	if len(fd.Funcs[0].Params) != 0 {
		t.Errorf("params = %v", fd.Funcs[0].Params)
	}
}

func TestLookupSymbol(t *testing.T) {
	src := `# File doc.

# A helper function.
def helper(x)
end

# A Point.
struct Point
  x
  y
end
`
	fd := Extract(src, "test.rugo")

	doc, sig, found := LookupSymbol(fd, "helper")
	if !found {
		t.Fatal("helper not found")
	}
	if doc != "A helper function." {
		t.Errorf("doc = %q", doc)
	}
	if sig != "def helper(x)" {
		t.Errorf("sig = %q", sig)
	}

	doc, sig, found = LookupSymbol(fd, "Point")
	if !found {
		t.Fatal("Point not found")
	}
	if doc != "A Point." {
		t.Errorf("doc = %q", doc)
	}
	if sig != "struct Point { x, y }" {
		t.Errorf("sig = %q", sig)
	}

	_, _, found = LookupSymbol(fd, "missing")
	if found {
		t.Error("expected missing to not be found")
	}
}

func TestExtract_FileDocBeforeCode(t *testing.T) {
	src := `# Module documentation.
# More details.
require "./client"
require "./repo"
`
	fd := Extract(src, "test.rugo")
	if fd.Doc != "Module documentation.\nMore details." {
		t.Errorf("file doc = %q", fd.Doc)
	}
}

func TestExtractDir(t *testing.T) {
	dir := t.TempDir()

	// Entry file with file-level doc
	writeFile(t, dir, "main.rugo", "# Top-level doc.\nrequire \"./lib\"\n")
	// Library with functions
	writeFile(t, dir, "lib.rugo", "# Adds numbers.\ndef add(a, b)\n  return a + b\nend\n\n# Subtracts.\ndef sub(a, b)\n  return a - b\nend\n")
	// Another file with a struct
	writeFile(t, dir, "types.rugo", "# A Point.\nstruct Point\n  x\n  y\nend\n")

	fd, err := ExtractDir(dir, dir+"/main.rugo")
	if err != nil {
		t.Fatal(err)
	}
	if fd.Doc != "Top-level doc." {
		t.Errorf("doc = %q", fd.Doc)
	}
	if len(fd.Funcs) != 2 {
		t.Errorf("expected 2 funcs, got %d", len(fd.Funcs))
	}
	if len(fd.Structs) != 1 {
		t.Errorf("expected 1 struct, got %d", len(fd.Structs))
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(dir+"/"+name, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFormatBridgePackage_WithStructs(t *testing.T) {
	pkg := &gobridge.Package{
		Path: "example.com/mymod",
		Doc:  "Test module.",
		Funcs: map[string]gobridge.GoFuncSig{
			"greet": {GoName: "Greet", Params: []gobridge.GoType{gobridge.GoString}, Returns: []gobridge.GoType{gobridge.GoString}},
		},
		Structs: []gobridge.GoStructInfo{
			{
				GoName:   "Config",
				RugoName: "config",
				Fields: []gobridge.GoStructFieldInfo{
					{GoName: "Name", RugoName: "name", Type: gobridge.GoString},
					{GoName: "Port", RugoName: "port", Type: gobridge.GoInt},
				},
			},
		},
	}

	out := FormatBridgePackage(pkg)

	// Struct definition appears
	if !strings.Contains(out, "struct Config") {
		t.Errorf("missing struct Config in output:\n%s", out)
	}
	if !strings.Contains(out, "name: string") {
		t.Errorf("missing field name: string in output:\n%s", out)
	}
	if !strings.Contains(out, "port: int") {
		t.Errorf("missing field port: int in output:\n%s", out)
	}
	// Function still appears
	if !strings.Contains(out, "mymod.greet(string) -> string") {
		t.Errorf("missing greet function in output:\n%s", out)
	}
}

func TestFormatBridgePackage_NoStructs(t *testing.T) {
	pkg := &gobridge.Package{
		Path: "strings",
		Doc:  "String functions.",
		Funcs: map[string]gobridge.GoFuncSig{
			"to_upper": {GoName: "ToUpper", Params: []gobridge.GoType{gobridge.GoString}, Returns: []gobridge.GoType{gobridge.GoString}},
		},
	}

	out := FormatBridgePackage(pkg)

	if strings.Contains(out, "struct") {
		t.Errorf("unexpected struct in output:\n%s", out)
	}
	if !strings.Contains(out, "strings.to_upper(string) -> string") {
		t.Errorf("missing to_upper function in output:\n%s", out)
	}
}

func TestFormatBridgeFuncSig_StructParams(t *testing.T) {
	sig := gobridge.GoFuncSig{
		GoName:      "GetName",
		Params:      []gobridge.GoType{gobridge.GoString}, // placeholder
		Returns:     []gobridge.GoType{gobridge.GoString},
		StructCasts: map[int]string{0: "rugo_struct_mymod_Config"},
	}

	out := formatBridgeFuncSig("mymod", "get_name", sig)

	if !strings.Contains(out, "mymod.get_name(Config)") {
		t.Errorf("expected struct type name in params, got: %s", out)
	}
	if !strings.Contains(out, "-> string") {
		t.Errorf("expected string return, got: %s", out)
	}
}

func TestFormatBridgeFuncSig_StructReturn(t *testing.T) {
	sig := gobridge.GoFuncSig{
		GoName:            "NewConfig",
		Params:            []gobridge.GoType{gobridge.GoString, gobridge.GoInt},
		Returns:           []gobridge.GoType{gobridge.GoString}, // placeholder
		StructReturnWraps: map[int]string{0: "rugo_struct_mymod_Config"},
	}

	out := formatBridgeFuncSig("mymod", "new_config", sig)

	if !strings.Contains(out, "-> Config") {
		t.Errorf("expected struct type name in return, got: %s", out)
	}
	if !strings.Contains(out, "mymod.new_config(string, int)") {
		t.Errorf("expected normal params, got: %s", out)
	}
}

func TestFormatBridgeFuncSig_Constructor(t *testing.T) {
	sig := gobridge.GoFuncSig{
		GoName:  "Config",
		Returns: []gobridge.GoType{gobridge.GoString},
		Codegen: func(pkgBase string, args []string, rugoName string) string { return "" },
	}

	out := formatBridgeFuncSig("mymod", "config", sig)

	if !strings.Contains(out, "-> Config") {
		t.Errorf("expected struct name from GoName in constructor return, got: %s", out)
	}
}

func TestFormatBridgePackage_MultipleStructs(t *testing.T) {
	pkg := &gobridge.Package{
		Path:  "example.com/multi",
		Doc:   "Multi-struct module.",
		Funcs: map[string]gobridge.GoFuncSig{},
		Structs: []gobridge.GoStructInfo{
			{GoName: "Config", RugoName: "config", Fields: []gobridge.GoStructFieldInfo{
				{GoName: "Name", RugoName: "name", Type: gobridge.GoString},
			}},
			{GoName: "Server", RugoName: "server", Fields: []gobridge.GoStructFieldInfo{
				{GoName: "Host", RugoName: "host", Type: gobridge.GoString},
				{GoName: "Debug", RugoName: "debug", Type: gobridge.GoBool},
			}},
		},
	}

	out := FormatBridgePackage(pkg)

	if !strings.Contains(out, "struct Config { name: string }") {
		t.Errorf("missing Config struct in output:\n%s", out)
	}
	if !strings.Contains(out, "struct Server { host: string, debug: bool }") {
		t.Errorf("missing Server struct in output:\n%s", out)
	}
}

func TestStructNameFromWrapper(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"rugo_struct_mymod_Config", "Config"},
		{"rugo_struct_sm_Server", "Server"},
		{"short", "short"},
	}
	for _, tt := range tests {
		got := structNameFromWrapper(tt.input)
		if got != tt.want {
			t.Errorf("structNameFromWrapper(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
