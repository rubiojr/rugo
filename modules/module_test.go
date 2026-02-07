package modules

import (
	"strings"
	"testing"
)

func withCleanRegistry(t *testing.T) {
	t.Helper()
	old := registry
	registry = make(map[string]*Module)
	t.Cleanup(func() { registry = old })
}

func TestRegisterAndGet(t *testing.T) {
	withCleanRegistry(t)

	Register(&Module{
		Name:  "test",
		Funcs: []FuncDef{{Name: "foo", Args: []ArgType{String}}},
	})

	got, ok := Get("test")
	if !ok {
		t.Fatal("expected module to be found")
	}
	if got.Name != "test" {
		t.Errorf("name = %q, want %q", got.Name, "test")
	}

	_, ok = Get("nonexistent")
	if ok {
		t.Error("expected false for unknown module")
	}
}

func TestIsModule(t *testing.T) {
	withCleanRegistry(t)
	Register(&Module{Name: "test"})

	if !IsModule("test") {
		t.Error("expected true for registered module")
	}
	if IsModule("nope") {
		t.Error("expected false for unregistered module")
	}
}

func TestNames(t *testing.T) {
	withCleanRegistry(t)
	Register(&Module{Name: "beta"})
	Register(&Module{Name: "alpha"})

	names := Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("Names() = %v, want [alpha beta]", names)
	}
}

func TestLookupFunc(t *testing.T) {
	withCleanRegistry(t)
	Register(&Module{
		Name: "mymod",
		Funcs: []FuncDef{
			{Name: "hello", Args: []ArgType{String}},
			{Name: "count", Args: []ArgType{Int}},
		},
	})

	goName, ok := LookupFunc("mymod", "hello")
	if !ok || goName != "rugo_mymod_hello" {
		t.Errorf("LookupFunc = (%q, %v), want (rugo_mymod_hello, true)", goName, ok)
	}

	_, ok = LookupFunc("mymod", "missing")
	if ok {
		t.Error("expected false for unknown function")
	}

	_, ok = LookupFunc("unknown", "hello")
	if ok {
		t.Error("expected false for unknown module")
	}
}

func TestCleanRuntime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		notWant string
	}{
		{
			name:    "strips build tag and package",
			input:   "//go:build ignore\n\npackage foo\n\nfunc bar() {}\n",
			want:    "func bar() {}",
			notWant: "package",
		},
		{
			name:  "preserves comments after header",
			input: "//go:build ignore\n\npackage foo\n\n// a comment\nvar x = 1\n",
			want:  "// a comment",
		},
		{
			name:  "no header to strip",
			input: "func bar() {}\n",
			want:  "func bar() {}",
		},
		{
			name:    "strips only leading blank lines",
			input:   "//go:build ignore\n\npackage foo\n\n\nfunc a() {}\n\n\nfunc b() {}\n",
			want:    "func a() {}\n\n\nfunc b() {}",
			notWant: "//go:build",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanRuntime(tt.input)
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Errorf("expected %q in:\n%s", tt.want, got)
			}
			if tt.notWant != "" && strings.Contains(got, tt.notWant) {
				t.Errorf("did not expect %q in:\n%s", tt.notWant, got)
			}
		})
	}
}

func TestFullRuntimeTypedArgs(t *testing.T) {
	m := &Module{
		Name: "test",
		Type: "Test",
		Funcs: []FuncDef{
			{Name: "greet", Args: []ArgType{String}},
			{Name: "add", Args: []ArgType{Int, Int}},
		},
		Runtime: "// impl\n",
	}

	got := m.FullRuntime()

	// Wrapper signature
	if !strings.Contains(got, "func rugo_test_greet(args ...interface{}) interface{} {") {
		t.Errorf("missing greet wrapper:\n%s", got)
	}
	// Calls struct method
	if !strings.Contains(got, "return _test.Greet(rugo_to_string(args[0]))") {
		t.Errorf("missing struct method call:\n%s", got)
	}
	// Arg count validation
	if !strings.Contains(got, `len(args) < 2`) {
		t.Errorf("missing arg count check for add:\n%s", got)
	}
	// Two int conversions via struct call
	if !strings.Contains(got, "return _test.Add(rugo_to_int(args[0]), rugo_to_int(args[1]))") {
		t.Errorf("missing struct method call for add:\n%s", got)
	}
}

func TestFullRuntimeAllArgTypes(t *testing.T) {
	m := &Module{
		Name: "t",
		Type: "T",
		Funcs: []FuncDef{
			{Name: "f", Args: []ArgType{String, Int, Float, Bool, Any}},
		},
	}

	got := m.FullRuntime()

	for _, expect := range []string{
		"rugo_to_string(args[0])",
		"rugo_to_int(args[1])",
		"rugo_to_float(args[2])",
		"rugo_to_bool(args[3])",
		"args[4]",
	} {
		if !strings.Contains(got, expect) {
			t.Errorf("missing %q in:\n%s", expect, got)
		}
	}
}

func TestFullRuntimeVariadic(t *testing.T) {
	m := &Module{
		Name: "test",
		Type: "Test",
		Funcs: []FuncDef{
			{Name: "call", Args: []ArgType{String}, Variadic: true},
		},
	}

	got := m.FullRuntime()

	if !strings.Contains(got, "return _test.Call(rugo_to_string(args[0]), args[1:]...)") {
		t.Errorf("missing variadic struct method call:\n%s", got)
	}
}

func TestFullRuntimePureVariadic(t *testing.T) {
	m := &Module{
		Name:  "test",
		Type:  "Test",
		Funcs: []FuncDef{{Name: "raw", Variadic: true}},
	}

	got := m.FullRuntime()

	if !strings.Contains(got, "return _test.Raw(args...)") {
		t.Errorf("expected pure variadic struct call:\n%s", got)
	}
}

func TestFullRuntimeNoArgs(t *testing.T) {
	m := &Module{
		Name:  "test",
		Type:  "Test",
		Funcs: []FuncDef{{Name: "noop"}},
	}

	got := m.FullRuntime()

	if !strings.Contains(got, "return _test.Noop()") {
		t.Errorf("expected no-arg struct call:\n%s", got)
	}
}

func TestFullRuntimePreservesImplCode(t *testing.T) {
	m := &Module{
		Name:    "test",
		Type:    "Test",
		Funcs:   []FuncDef{{Name: "foo", Args: []ArgType{String}}},
		Runtime: "type Test struct{}\nfunc (Test) Foo(s string) interface{} { return s }\n",
	}

	got := m.FullRuntime()

	if !strings.Contains(got, "type Test struct{}") {
		t.Errorf("missing struct type:\n%s", got)
	}
	if !strings.Contains(got, "func rugo_test_foo(args ...interface{}) interface{}") {
		t.Errorf("missing wrapper:\n%s", got)
	}
}

func TestFullRuntimeSnakeCaseMethod(t *testing.T) {
	m := &Module{
		Name: "conv",
		Type: "Conv",
		Funcs: []FuncDef{
			{Name: "to_s", Args: []ArgType{Any}},
			{Name: "to_i", Args: []ArgType{Any}},
		},
	}

	got := m.FullRuntime()

	if !strings.Contains(got, "_conv.ToS(args[0])") {
		t.Errorf("expected snake_case → PascalCase conversion for to_s:\n%s", got)
	}
	if !strings.Contains(got, "_conv.ToI(args[0])") {
		t.Errorf("expected snake_case → PascalCase conversion for to_i:\n%s", got)
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"get", "Get"},
		{"exec", "Exec"},
		{"to_s", "ToS"},
		{"to_i", "ToI"},
		{"my_long_name", "MyLongName"},
		{"a", "A"},
	}
	for _, tt := range tests {
		if got := toPascalCase(tt.input); got != tt.want {
			t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
