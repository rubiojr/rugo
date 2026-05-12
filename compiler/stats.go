package compiler

import (
	"encoding/json"
	"fmt"
	goast "go/ast"
	goparser "go/parser"
	"go/token"
	"regexp"
	"sort"
	"strings"

	"github.com/rubiojr/rugo/ast"
)

// Stats summarises how much of a Rugo program was resolved to concrete types
// and how that translated into the generated Go source. The numbers are
// designed to be diffable across commits as a CI signal: more "dynamic" and
// more rugo_* helper calls means more runtime boxing and worse perf.
type Stats struct {
	SourceFile string       `json:"source_file"`
	Source     SourceStats  `json:"source"`
	Generated  GeneratedGo  `json:"generated"`
	Hotspots   []Hotspot    `json:"hotspots,omitempty"`
}

// SourceStats counts typed vs dynamic items in the Rugo program.
type SourceStats struct {
	Functions FunctionStats `json:"functions"`
	Params    Counter       `json:"params"`
	Returns   Counter       `json:"returns"`
	Locals    Counter       `json:"locals"`
	Exprs     Counter       `json:"expressions"`
}

// FunctionStats classifies user-defined functions by how much was inferred.
type FunctionStats struct {
	Total        int `json:"total"`
	FullyTyped   int `json:"fully_typed"`
	PartialTyped int `json:"partial_typed"`
	Untyped      int `json:"untyped"`
}

// Counter is the standard typed/dynamic tally used for params, returns, etc.
//
// Annotated counts how many came from explicit "name : Type" source
// annotations (as opposed to being inferred from usage). Annotated counts
// are a subset of Typed.
type Counter struct {
	Total     int `json:"total"`
	Typed     int `json:"typed"`
	Dynamic   int `json:"dynamic"`
	Annotated int `json:"annotated,omitempty"`
}

// Pct returns the typed percentage as a 0..100 float. Zero when Total is 0.
func (c Counter) Pct() float64 {
	if c.Total == 0 {
		return 0
	}
	return 100.0 * float64(c.Typed) / float64(c.Total)
}

// AnnotatedPct returns the annotated percentage as a 0..100 float.
func (c Counter) AnnotatedPct() float64 {
	if c.Total == 0 {
		return 0
	}
	return 100.0 * float64(c.Annotated) / float64(c.Total)
}

// GeneratedGo counts the shape of the emitted Go source, focusing on the
// constructs that signal runtime boxing.
type GeneratedGo struct {
	TotalLines           int            `json:"total_lines"`
	UserFunctions        int            `json:"user_functions"`
	InterfaceOccurrences int            `json:"interface_occurrences"`
	BoxingCasts          int            `json:"boxing_casts"`
	RugoHelperCalls      int            `json:"rugo_helper_calls"`
	HelperBreakdown      map[string]int `json:"helper_breakdown"`
}

// Hotspot pinpoints a Rugo source location where inference widened to dynamic
// inside a hot context (loop, recursive function).
type Hotspot struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// ComputeStats analyses a compiled program and produces a Stats summary.
// Pass the inferred TypeInfo (may be nil — in which case all source-level
// counts collapse to dynamic) and the generated Go source.
func ComputeStats(prog *ast.Program, ti *TypeInfo, goSource, sourceFile string) *Stats {
	s := &Stats{SourceFile: sourceFile}
	s.Generated.HelperBreakdown = map[string]int{}

	collectSourceStats(prog, ti, &s.Source)
	collectExprStats(ti, &s.Source.Exprs)
	collectLocalStats(prog, ti, &s.Source.Locals)
	analyzeGenerated(goSource, &s.Generated)
	s.Hotspots = findHotspots(prog, ti)

	return s
}

// helperCategories groups rugo_* runtime calls into buckets for readability.
// Calls not in this map fall under "other".
var helperCategories = map[string]string{
	// type coercion
	"rugo_to_bool":         "coerce",
	"rugo_to_int":          "coerce",
	"rugo_to_float":        "coerce",
	"rugo_to_string":       "coerce",
	"rugo_to_array_index":  "coerce",
	"rugo_to_go":           "coerce",
	"rugo_to_lambda":       "coerce",
	"rugo_float":           "coerce",
	// arithmetic on interface{}
	"rugo_add":    "arith",
	"rugo_sub":    "arith",
	"rugo_mul":    "arith",
	"rugo_div":    "arith",
	"rugo_mod":    "arith",
	"rugo_negate": "arith",
	"rugo_not":    "arith",
	// comparisons on interface{}
	"rugo_eq":      "compare",
	"rugo_neq":     "compare",
	"rugo_lt":      "compare",
	"rugo_gt":      "compare",
	"rugo_le":      "compare",
	"rugo_ge":      "compare",
	"rugo_compare": "compare",
	// indexing & dot access
	"rugo_index":       "access",
	"rugo_slice":       "access",
	"rugo_index_set":   "access",
	"rugo_array_index": "access",
	"rugo_dot_get":     "access",
	"rugo_dot_set":     "access",
	"rugo_dot_call":    "access",
	// builtins
	"rugo_puts":    "builtin",
	"rugo_print":   "builtin",
	"rugo_raise":   "builtin",
	"rugo_exit":    "builtin",
	"rugo_len":     "builtin",
	"rugo_type_of": "builtin",
	"rugo_append":  "builtin",
	// iteration
	"rugo_iterable":         "iter",
	"rugo_iterable_default": "iter",
	"rugo_range":            "iter",
	// methods
	"rugo_array_method": "method",
	"rugo_hash_method":  "method",
	// shell
	"rugo_shell":      "shell",
	"rugo_capture":    "shell",
	"rugo_pipe_shell": "shell",
}

// HelperCategory returns the category for a rugo_* helper, or "other".
func HelperCategory(name string) string {
	if c, ok := helperCategories[name]; ok {
		return c
	}
	return "other"
}

// collectSourceStats walks the AST counting user-defined functions, their
// parameters and inferred return types.
func collectSourceStats(prog *ast.Program, ti *TypeInfo, ss *SourceStats) {
	WalkStmts(prog, func(s ast.Statement) bool {
		f, ok := s.(*ast.FuncDef)
		if !ok {
			return true
		}
		ss.Functions.Total++
		fti := lookupFuncType(ti, f)

		typedParams := 0
		for i, p := range f.Params {
			ss.Params.Total++
			t := paramType(fti, i)
			if t.IsTyped() {
				ss.Params.Typed++
				typedParams++
			} else {
				ss.Params.Dynamic++
			}
			if p.TypeAnnot != "" {
				ss.Params.Annotated++
			}
		}

		ss.Returns.Total++
		rt := returnType(fti)
		if rt.IsTyped() {
			ss.Returns.Typed++
		} else {
			ss.Returns.Dynamic++
		}
		if f.ReturnType != "" {
			ss.Returns.Annotated++
		}

		// Classify the function as fully/partial/untyped. A function with no
		// params is fully typed when its return is typed.
		allTyped := rt.IsTyped() && typedParams == len(f.Params)
		anyTyped := rt.IsTyped() || typedParams > 0
		switch {
		case allTyped:
			ss.Functions.FullyTyped++
		case anyTyped:
			ss.Functions.PartialTyped++
		default:
			ss.Functions.Untyped++
		}
		// Don't recurse into func bodies for function counting.
		return false
	})
}

// collectLocalStats counts named locals across every known scope in TypeInfo.
// Top-level vars, function-local vars, bench/test blocks all participate.
func collectLocalStats(prog *ast.Program, ti *TypeInfo, c *Counter) {
	if ti == nil {
		return
	}
	// Build the set of param names per function so we can exclude them.
	paramSets := map[string]map[string]bool{}
	WalkStmts(prog, func(s ast.Statement) bool {
		f, ok := s.(*ast.FuncDef)
		if !ok {
			return true
		}
		key := f.Name
		if f.Namespace != "" {
			key = f.Namespace + "." + f.Name
		}
		set := map[string]bool{}
		for _, p := range f.Params {
			set[p.Name] = true
		}
		paramSets[key] = set
		return false
	})

	for scope, vars := range ti.VarTypes {
		params := paramSets[scope]
		for name, t := range vars {
			if params != nil && params[name] {
				continue
			}
			c.Total++
			if t.IsTyped() {
				c.Typed++
			} else {
				c.Dynamic++
			}
		}
	}
}

// collectExprStats tallies every expression that inference observed.
func collectExprStats(ti *TypeInfo, c *Counter) {
	if ti == nil {
		return
	}
	for _, t := range ti.ExprTypes {
		c.Total++
		if t.IsTyped() {
			c.Typed++
		} else {
			c.Dynamic++
		}
	}
}

func lookupFuncType(ti *TypeInfo, f *ast.FuncDef) *FuncTypeInfo {
	if ti == nil {
		return nil
	}
	key := f.Name
	if f.Namespace != "" {
		key = f.Namespace + "." + f.Name
	}
	return ti.FuncTypes[key]
}

func paramType(fti *FuncTypeInfo, i int) RugoType {
	if fti == nil || i >= len(fti.ParamTypes) {
		return TypeDynamic
	}
	return fti.ParamTypes[i]
}

func returnType(fti *FuncTypeInfo) RugoType {
	if fti == nil {
		return TypeDynamic
	}
	return fti.ReturnType
}

// userFuncName reports whether a Go function name in the generated source
// corresponds to user-authored Rugo code (as opposed to a runtime helper).
var (
	testHarnessRE  = regexp.MustCompile(`^rugo_test_\d+$`)
	benchHarnessRE = regexp.MustCompile(`^rugo_bench_\d+$`)
)

func userFuncName(name string) bool {
	if name == "main" {
		return true
	}
	if strings.HasPrefix(name, "rugofn_") {
		return true
	}
	if strings.HasPrefix(name, "rugons_") {
		return true
	}
	if testHarnessRE.MatchString(name) {
		return true
	}
	if benchHarnessRE.MatchString(name) {
		return true
	}
	return false
}

// analyzeGenerated parses the emitted Go source and counts how many user
// functions use interface{}, perform boxing casts, or call rugo_* helpers.
//
// Runtime helper definitions (func rugo_xxx) are excluded so the numbers
// reflect what user-authored code costs, not the boilerplate.
func analyzeGenerated(src string, g *GeneratedGo) {
	g.TotalLines = strings.Count(src, "\n") + 1
	if src == "" {
		return
	}
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "", src, goparser.SkipObjectResolution)
	if err != nil {
		// Fall back to a coarse regex count so stats stay useful even if the
		// emitted source isn't fully parseable.
		analyzeGeneratedFallback(src, g)
		return
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*goast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name == nil || !userFuncName(fn.Name.Name) {
			continue
		}
		g.UserFunctions++
		walkUserFunc(fn, g)
	}
}

func walkUserFunc(fn *goast.FuncDecl, g *GeneratedGo) {
	visitNode(fn.Type, g)
	if fn.Body != nil {
		visitNode(fn.Body, g)
	}
}

func visitNode(n goast.Node, g *GeneratedGo) {
	goast.Inspect(n, func(node goast.Node) bool {
		switch x := node.(type) {
		case *goast.InterfaceType:
			if x.Methods != nil && len(x.Methods.List) == 0 {
				g.InterfaceOccurrences++
			}
		case *goast.CallExpr:
			if isInterfaceTypeExpr(x.Fun) {
				g.BoxingCasts++
			}
			name, ok := callName(x.Fun)
			if ok && strings.HasPrefix(name, "rugo_") && !userFuncName(name) {
				g.RugoHelperCalls++
				g.HelperBreakdown[name]++
			}
		}
		return true
	})
}

func isInterfaceTypeExpr(e goast.Expr) bool {
	switch v := e.(type) {
	case *goast.InterfaceType:
		return v.Methods == nil || len(v.Methods.List) == 0
	case *goast.ParenExpr:
		return isInterfaceTypeExpr(v.X)
	}
	return false
}

func callName(e goast.Expr) (string, bool) {
	switch v := e.(type) {
	case *goast.Ident:
		return v.Name, true
	case *goast.SelectorExpr:
		if id, ok := v.X.(*goast.Ident); ok {
			return id.Name + "." + v.Sel.Name, true
		}
	}
	return "", false
}

// analyzeGeneratedFallback is the regex-based last resort when the emitted
// Go source can't be parsed (should only happen on internal bugs).
var (
	fallbackIfaceRE   = regexp.MustCompile(`\binterface\s*\{\s*\}`)
	fallbackBoxingRE  = regexp.MustCompile(`\binterface\s*\{\s*\}\s*\(`)
	fallbackHelperRE  = regexp.MustCompile(`\b(rugo_[a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	fallbackDefHelpRE = regexp.MustCompile(`\bfunc\s+(rugo_[a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
)

func analyzeGeneratedFallback(src string, g *GeneratedGo) {
	g.InterfaceOccurrences = len(fallbackIfaceRE.FindAllString(src, -1))
	g.BoxingCasts = len(fallbackBoxingRE.FindAllString(src, -1))

	defs := map[string]int{}
	for _, m := range fallbackDefHelpRE.FindAllStringSubmatch(src, -1) {
		defs[m[1]]++
	}
	for _, m := range fallbackHelperRE.FindAllStringSubmatch(src, -1) {
		name := m[1]
		if defs[name] > 0 {
			defs[name]--
			continue
		}
		if userFuncName(name) {
			continue
		}
		g.RugoHelperCalls++
		g.HelperBreakdown[name]++
	}
}

// findHotspots scans for dynamic widening inside hot constructs (loops,
// recursive functions). It's a best-effort signal — false negatives are
// fine, false positives are surprises so we err on the conservative side.
func findHotspots(prog *ast.Program, ti *TypeInfo) []Hotspot {
	if ti == nil {
		return nil
	}
	var out []Hotspot
	WalkStmts(prog, func(s ast.Statement) bool {
		switch st := s.(type) {
		case *ast.ForStmt:
			if t := ti.ExprType(st.Collection); !t.IsTyped() && t != TypeArray && t != TypeHash {
				// Already dynamic at the source — not a "widening", just a fact.
			}
			// for x in coll → loop var is interface{} unless coll is a typed int range.
			if loopVarType(ti, st) == TypeDynamic {
				out = append(out, Hotspot{
					File:    prog.SourceFile,
					Line:    st.StmtLine(),
					Kind:    "dynamic-loop-var",
					Message: fmt.Sprintf("loop variable %q is dynamic; consider annotating the source collection or using `for i in 0..N`", st.Var),
				})
			}
		}
		return true
	})
	return out
}

func loopVarType(ti *TypeInfo, st *ast.ForStmt) RugoType {
	// Mirror inferForVarType's logic with what's exposed on TypeInfo.
	if _, ok := st.Collection.(*ast.IntLiteral); ok {
		return TypeInt
	}
	if call, ok := st.Collection.(*ast.CallExpr); ok {
		if fn, ok := call.Func.(*ast.IdentExpr); ok && fn.Name == "range" {
			return TypeInt
		}
	}
	return ti.ExprType(st.Collection)
}

// Text renders a human-friendly, fixed-column report of the stats.
func (s *Stats) Text() string {
	var b strings.Builder

	fmt.Fprintf(&b, "=== type stats: %s ===\n", s.SourceFile)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Rugo source")
	fmt.Fprintf(&b, "  Functions:   %5d   fully typed: %d   partial: %d   untyped: %d\n",
		s.Source.Functions.Total,
		s.Source.Functions.FullyTyped,
		s.Source.Functions.PartialTyped,
		s.Source.Functions.Untyped)
	fmt.Fprintf(&b, "  Params:      %5d   typed: %d (%.1f%%)   dynamic: %d   annotated: %d (%.1f%%)\n",
		s.Source.Params.Total, s.Source.Params.Typed, s.Source.Params.Pct(), s.Source.Params.Dynamic,
		s.Source.Params.Annotated, s.Source.Params.AnnotatedPct())
	fmt.Fprintf(&b, "  Returns:     %5d   typed: %d (%.1f%%)   dynamic: %d   annotated: %d (%.1f%%)\n",
		s.Source.Returns.Total, s.Source.Returns.Typed, s.Source.Returns.Pct(), s.Source.Returns.Dynamic,
		s.Source.Returns.Annotated, s.Source.Returns.AnnotatedPct())
	fmt.Fprintf(&b, "  Locals:      %5d   typed: %d (%.1f%%)   dynamic: %d\n",
		s.Source.Locals.Total, s.Source.Locals.Typed, s.Source.Locals.Pct(), s.Source.Locals.Dynamic)
	fmt.Fprintf(&b, "  Expressions: %5d   typed: %d (%.1f%%)   dynamic: %d\n",
		s.Source.Exprs.Total, s.Source.Exprs.Typed, s.Source.Exprs.Pct(), s.Source.Exprs.Dynamic)

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Generated Go")
	fmt.Fprintf(&b, "  Total lines:           %d\n", s.Generated.TotalLines)
	fmt.Fprintf(&b, "  User functions:        %d\n", s.Generated.UserFunctions)
	fmt.Fprintf(&b, "  interface{} types:     %d\n", s.Generated.InterfaceOccurrences)
	fmt.Fprintf(&b, "  Boxing casts:          %d   (interface{}(...))\n", s.Generated.BoxingCasts)
	fmt.Fprintf(&b, "  rugo_* helper calls:   %d\n", s.Generated.RugoHelperCalls)

	if len(s.Generated.HelperBreakdown) > 0 {
		// Aggregate by category, then list the top callers per category.
		byCat := map[string]int{}
		type kv struct {
			name  string
			count int
		}
		flat := make([]kv, 0, len(s.Generated.HelperBreakdown))
		for name, n := range s.Generated.HelperBreakdown {
			byCat[HelperCategory(name)] += n
			flat = append(flat, kv{name, n})
		}
		sort.Slice(flat, func(i, j int) bool {
			if flat[i].count != flat[j].count {
				return flat[i].count > flat[j].count
			}
			return flat[i].name < flat[j].name
		})

		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "  Helper breakdown (by category)")
		cats := make([]string, 0, len(byCat))
		for c := range byCat {
			cats = append(cats, c)
		}
		sort.Slice(cats, func(i, j int) bool { return byCat[cats[i]] > byCat[cats[j]] })
		for _, c := range cats {
			fmt.Fprintf(&b, "    %-10s %d\n", c+":", byCat[c])
		}

		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "  Top helpers")
		top := flat
		if len(top) > 10 {
			top = top[:10]
		}
		for _, kv := range top {
			fmt.Fprintf(&b, "    %-22s %d\n", kv.name, kv.count)
		}
	}

	if len(s.Hotspots) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "Hotspots")
		for _, h := range s.Hotspots {
			fmt.Fprintf(&b, "  %s:%d  [%s] %s\n", h.File, h.Line, h.Kind, h.Message)
		}
	}

	return b.String()
}

// JSON renders the stats as indented JSON suitable for diffing across runs.
func (s *Stats) JSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}
