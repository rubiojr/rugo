package compiler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compileForStats compiles src in-memory and returns the CompileResult.
// Helper keeps every test 3-liner clean.
func compileForStats(t *testing.T, src string, disableInfer bool) *CompileResult {
	t.Helper()
	c := &Compiler{DisableInfer: disableInfer}
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)

	resolved, err := c.resolveRequires(prog)
	require.NoError(t, err)

	gen, err := generate(resolved, "test.rugo", false, nil, false, disableInfer)
	require.NoError(t, err)

	return &CompileResult{
		GoSource:   gen.GoSource,
		Program:    resolved,
		SourceFile: "test.rugo",
		TypeInfo:   gen.TypeInfo,
	}
}

func statsFor(t *testing.T, src string, disableInfer bool) *Stats {
	t.Helper()
	res := compileForStats(t, src, disableInfer)
	return ComputeStats(res.Program, res.TypeInfo, res.GoSource, res.SourceFile)
}

func TestStatsFullyTypedProgram(t *testing.T) {
	src := `
def add(a, b)
  return a + b
end

x = 1
y = 2
z = add(x, y)
puts(z)
`
	s := statsFor(t, src, false)

	assert.Equal(t, 1, s.Source.Functions.Total, "one user function")
	assert.Equal(t, 1, s.Source.Functions.FullyTyped, "add should be fully typed")
	assert.Equal(t, 0, s.Source.Functions.Untyped)

	assert.Equal(t, 2, s.Source.Params.Total)
	assert.Equal(t, 2, s.Source.Params.Typed, "both params inferred as int")
	assert.InDelta(t, 100.0, s.Source.Params.Pct(), 0.01)

	assert.Equal(t, 1, s.Source.Returns.Total)
	assert.Equal(t, 1, s.Source.Returns.Typed)

	assert.Equal(t, 3, s.Source.Locals.Total, "x, y, z")
	assert.Equal(t, 3, s.Source.Locals.Typed)
}

func TestStatsDynamicProgramWithoutInfer(t *testing.T) {
	src := `
def add(a, b)
  return a + b
end

x = 1
y = 2
z = add(x, y)
puts(z)
`
	typed := statsFor(t, src, false)
	dynamic := statsFor(t, src, true)

	// With inference disabled the generated Go must lean on interface{} and
	// boxing/coercion helpers more than the typed build.
	assert.Greater(t, dynamic.Generated.InterfaceOccurrences, typed.Generated.InterfaceOccurrences,
		"no-infer should produce more interface{} occurrences")
	assert.GreaterOrEqual(t, dynamic.Generated.RugoHelperCalls, typed.Generated.RugoHelperCalls,
		"no-infer should call at least as many runtime helpers")

	// Source-level stats are unaffected by codegen choices because Infer()
	// still runs to feed the stats command.
	assert.Equal(t, typed.Source.Functions.Total, dynamic.Source.Functions.Total)
	assert.Equal(t, typed.Source.Params.Typed, dynamic.Source.Params.Typed)
}

func TestStatsJSONRoundTrip(t *testing.T) {
	src := `
def greet(name)
  return "hi, #{name}"
end

puts(greet("rugo"))
`
	s := statsFor(t, src, false)

	out, err := s.JSON()
	require.NoError(t, err)
	require.NotEmpty(t, out)

	var decoded Stats
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))

	assert.Equal(t, s.SourceFile, decoded.SourceFile)
	assert.Equal(t, s.Source.Functions.Total, decoded.Source.Functions.Total)
	assert.Equal(t, s.Generated.TotalLines, decoded.Generated.TotalLines)
	assert.Equal(t, s.Generated.UserFunctions, decoded.Generated.UserFunctions)
}

func TestStatsTextRendererIncludesKeySections(t *testing.T) {
	src := `
def f(x)
  return x + 1
end
puts(f(2))
`
	s := statsFor(t, src, false)
	out := s.Text()

	for _, want := range []string{
		"type stats",
		"Rugo source",
		"Functions:",
		"Params:",
		"Generated Go",
		"User functions:",
		"rugo_* helper calls:",
	} {
		assert.True(t, strings.Contains(out, want), "missing %q in text output", want)
	}
}

func TestUserFuncNameClassification(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"main", true},
		{"rugofn_add", true},
		{"rugons_str_upper", true},
		{"rugo_test_3", true},
		{"rugo_bench_12", true},
		{"rugo_add", false},
		{"rugo_to_bool", false},
		{"rugo_panic_handler", false},
		{"fmt_Printf", false},
		{"", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, userFuncName(c.name), "userFuncName(%q)", c.name)
	}
}

func TestStatsCounterPct(t *testing.T) {
	assert.InDelta(t, 0.0, Counter{}.Pct(), 0.001)
	assert.InDelta(t, 50.0, Counter{Total: 4, Typed: 2}.Pct(), 0.001)
	assert.InDelta(t, 100.0, Counter{Total: 3, Typed: 3}.Pct(), 0.001)
}

func TestStatsHelperBreakdownCategorised(t *testing.T) {
	src := `
def add(a, b)
  return a + b
end
puts(add(1, 2))
`
	s := statsFor(t, src, true) // no-infer to maximise helper calls
	require.NotNil(t, s.Generated.HelperBreakdown)

	total := 0
	for _, v := range s.Generated.HelperBreakdown {
		total += v
	}
	assert.Equal(t, s.Generated.RugoHelperCalls, total,
		"per-category helper counts must sum to the total")
}
