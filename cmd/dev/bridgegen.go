package dev

import (
	"context"
	"fmt"
	"go/format"
	"go/importer"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/rubiojr/rugo/gobridge"
	"github.com/urfave/cli/v3"
)

func bridgegenCommand() *cli.Command {
	return &cli.Command{
		Name:      "bridgegen",
		Usage:     "Generate a Go bridge package (_gen.go file, always overwritten)",
		ArgsUsage: "<go-package>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Preview classification without writing files",
			},
		},
		Action: bridgegenAction,
	}
}

// bridgeTier classifies how a Go function can be bridged.
type bridgeTier int

const (
	tierAuto     bridgeTier = iota // fully auto-generatable
	tierCastable                   // needs int64/[]byte/rune casts
	tierFunc                       // has function parameters
	tierBlocked                    // generics, interfaces, channels, etc.
)

func (t bridgeTier) String() string {
	switch t {
	case tierAuto:
		return "auto"
	case tierCastable:
		return "castable"
	case tierFunc:
		return "func"
	case tierBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}

// bridgedFunc holds classification results for a Go function.
type bridgedFunc struct {
	GoName   string
	RugoName string
	Sig      *types.Signature
	Tier       bridgeTier
	Reason     string // why it was blocked
	Params     []gobridge.GoType
	Returns    []gobridge.GoType
	FuncTypes  map[int]*gobridge.GoFuncType  // GoFunc param signatures
	ArrayTypes map[int]*gobridge.GoArrayType // fixed-size array return metadata
	Variadic   bool
	Doc        string
}

func bridgegenAction(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return fmt.Errorf("usage: rugo dev bridgegen <go-package>")
	}
	pkgPath := cmd.Args().First()
	dryRun := cmd.Bool("dry-run")

	imp := importer.Default()
	pkg, err := imp.Import(pkgPath)
	if err != nil {
		return fmt.Errorf("importing %s: %w", pkgPath, err)
	}

	// Enumerate and classify exported functions
	scope := pkg.Scope()
	var funcs []bridgedFunc
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)

		// Package-level functions
		if fn, ok := obj.(*types.Func); ok {
			if !fn.Exported() {
				continue
			}
			sig := fn.Type().(*types.Signature)
			if sig.Recv() != nil {
				continue
			}
			rugoName := toSnakeCase(name)
			bf := classifyFunc(name, rugoName, sig)
			funcs = append(funcs, bf)
			continue
		}

		// Package-level vars — enumerate their methods (e.g., base64.StdEncoding)
		if v, ok := obj.(*types.Var); ok && v.Exported() {
			varType := v.Type()
			// Dereference pointer types
			if ptr, ok := varType.(*types.Pointer); ok {
				varType = ptr.Elem()
			}
			named, ok := varType.(*types.Named)
			if !ok {
				continue
			}
			// Check methods on the named type and its pointer
			for _, base := range []*types.Named{named} {
				for j := 0; j < base.NumMethods(); j++ {
					m := base.Method(j)
					if !m.Exported() {
						continue
					}
					sig := m.Type().(*types.Signature)
					// Build method chain GoName: "VarName.MethodName"
					goName := name + "." + m.Name()
					rugoName := toSnakeCase(name) + "_" + toSnakeCase(m.Name())
					// Strip receiver from signature for classification
					bf := classifyFunc(goName, rugoName, sig)
					funcs = append(funcs, bf)
				}
			}
			// Also check pointer receiver methods
			ptrMethods := types.NewMethodSet(types.NewPointer(named))
			for j := 0; j < ptrMethods.Len(); j++ {
				sel := ptrMethods.At(j)
				m := sel.Obj().(*types.Func)
				if !m.Exported() {
					continue
				}
				// Skip if already added from value methods
				found := false
				for _, f := range funcs {
					if f.GoName == name+"."+m.Name() {
						found = true
						break
					}
				}
				if found {
					continue
				}
				sig := m.Type().(*types.Signature)
				goName := name + "." + m.Name()
				rugoName := toSnakeCase(name) + "_" + toSnakeCase(m.Name())
				bf := classifyFunc(goName, rugoName, sig)
				funcs = append(funcs, bf)
			}
		}
	}

	sort.Slice(funcs, func(i, j int) bool {
		if funcs[i].Tier != funcs[j].Tier {
			return funcs[i].Tier < funcs[j].Tier
		}
		return funcs[i].RugoName < funcs[j].RugoName
	})

	if dryRun {
		return printClassification(pkgPath, funcs)
	}

	return writeBridgeFile(pkgPath, funcs)
}

// classifyFunc determines how a Go function can be bridged.
func classifyFunc(goName, rugoName string, sig *types.Signature) bridgedFunc {
	bf := bridgedFunc{
		GoName:   goName,
		RugoName: rugoName,
		Sig:      sig,
		Variadic: sig.Variadic(),
	}

	// Classify params
	params := sig.Params()
	hasCast := false
	for i := 0; i < params.Len(); i++ {
		t := params.At(i).Type()
		// For variadic params, keep the slice type (codegen handles individual arg conversion)
		gt, tier, reason := classifyGoType(t, true)
		if tier == tierBlocked {
			bf.Tier = tierBlocked
			bf.Reason = fmt.Sprintf("param %d: %s", i, reason)
			return bf
		}
		if tier == tierFunc {
			// Try to build a FuncType from the function signature
			funcSig, ok := params.At(i).Type().Underlying().(*types.Signature)
			if !ok {
				bf.Tier = tierFunc
				bf.Reason = fmt.Sprintf("param %d: func type (not a signature)", i)
				return bf
			}
			ft := classifyFuncType(funcSig)
			if ft == nil {
				bf.Tier = tierFunc
				bf.Reason = fmt.Sprintf("param %d: func with unbridgeable signature", i)
				return bf
			}
			if bf.FuncTypes == nil {
				bf.FuncTypes = map[int]*gobridge.GoFuncType{}
			}
			bf.FuncTypes[i] = ft
			hasCast = true // func params are at least castable tier
			bf.Params = append(bf.Params, gt)
			continue
		}
		if tier == tierCastable {
			hasCast = true
		}
		bf.Params = append(bf.Params, gt)
	}

	// Classify returns
	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		t := results.At(i).Type()
		gt, tier, reason := classifyGoType(t, false)
		if tier == tierBlocked {
			bf.Tier = tierBlocked
			bf.Reason = fmt.Sprintf("return %d: %s", i, reason)
			return bf
		}
		if tier == tierCastable {
			hasCast = true
		}
		// Track fixed-size array metadata
		if arr, ok := t.Underlying().(*types.Array); ok {
			if bf.ArrayTypes == nil {
				bf.ArrayTypes = map[int]*gobridge.GoArrayType{}
			}
			elemGT, _, _ := classifyGoType(arr.Elem(), false)
			bf.ArrayTypes[i] = &gobridge.GoArrayType{Elem: elemGT, Size: int(arr.Len())}
		}
		bf.Returns = append(bf.Returns, gt)
	}

	if hasCast {
		bf.Tier = tierCastable
	} else {
		bf.Tier = tierAuto
	}
	return bf
}

// classifyFuncType builds a GoFuncType from a Go function signature.
// Returns nil if any param/return type is unbridgeable.
func classifyFuncType(sig *types.Signature) *gobridge.GoFuncType {
	ft := &gobridge.GoFuncType{}

	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		gt, tier, _ := classifyGoType(params.At(i).Type(), true)
		if tier == tierBlocked || tier == tierFunc {
			return nil
		}
		ft.Params = append(ft.Params, gt)
	}

	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		gt, tier, _ := classifyGoType(results.At(i).Type(), false)
		if tier == tierBlocked || tier == tierFunc {
			return nil
		}
		ft.Returns = append(ft.Returns, gt)
	}

	return ft
}

// classifyGoType maps a Go type to a GoType and tier.
func classifyGoType(t types.Type, isParam bool) (gobridge.GoType, bridgeTier, string) {
	// Check named types first (error interface)
	if named, ok := t.(*types.Named); ok {
		if named.Obj().Pkg() == nil && named.Obj().Name() == "error" {
			return gobridge.GoError, tierAuto, ""
		}
		// Fall through to underlying type
		t = t.Underlying()
	}

	switch u := t.(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.String:
			return gobridge.GoString, tierAuto, ""
		case types.Int:
			return gobridge.GoInt, tierAuto, ""
		case types.Float64:
			return gobridge.GoFloat64, tierAuto, ""
		case types.Bool:
			return gobridge.GoBool, tierAuto, ""
		case types.Byte:
			return gobridge.GoByte, tierCastable, ""
		case types.Int64:
			return gobridge.GoInt64, tierCastable, ""
		case types.Int32:
			if u.Name() == "rune" {
				return gobridge.GoRune, tierCastable, ""
			}
			return gobridge.GoInt32, tierCastable, ""
		case types.Float32:
			return gobridge.GoFloat32, tierCastable, ""
		case types.Int8, types.Int16:
			return gobridge.GoInt, tierCastable, ""
		case types.Uint16:
			return gobridge.GoInt, tierCastable, ""
		case types.Uint:
			return gobridge.GoUint, tierCastable, ""
		case types.Uint32:
			return gobridge.GoUint32, tierCastable, ""
		case types.Uint64:
			return gobridge.GoUint64, tierCastable, ""
		default:
			return 0, tierBlocked, fmt.Sprintf("unsupported basic type %s", u.Name())
		}
	case *types.Slice:
		elem := u.Elem()
		if b, ok := elem.Underlying().(*types.Basic); ok {
			switch b.Kind() {
			case types.String:
				return gobridge.GoStringSlice, tierAuto, ""
			case types.Byte:
				return gobridge.GoByteSlice, tierCastable, ""
			}
		}
		return 0, tierBlocked, fmt.Sprintf("unsupported slice type []%s", elem)
	case *types.Signature:
		return gobridge.GoFunc, tierFunc, "function parameter"
	case *types.Interface:
		return 0, tierBlocked, "interface type"
	case *types.Pointer:
		return 0, tierBlocked, fmt.Sprintf("pointer to %s", u.Elem())
	case *types.Struct:
		return 0, tierBlocked, "struct type"
	case *types.Map:
		return 0, tierBlocked, "map type"
	case *types.Chan:
		return 0, tierBlocked, "channel type"
	case *types.Array:
		if b, ok := u.Elem().Underlying().(*types.Basic); ok && b.Kind() == types.Byte {
			return gobridge.GoByteSlice, tierCastable, ""
		}
		return 0, tierBlocked, fmt.Sprintf("array type [%d]%s", u.Len(), u.Elem())
	default:
		return 0, tierBlocked, fmt.Sprintf("unknown type %T", t)
	}
}

// toSnakeCase converts PascalCase to snake_case.
// Handles consecutive uppercase (e.g., "IsNaN" → "is_nan", "FMA" → "fma").
func toSnakeCase(s string) string {
	// Pre-process known abbreviations to avoid splitting them
	abbreviations := map[string]string{
		"NaN": "nan", "URL": "url", "URI": "uri", "HTTP": "http",
		"HTTPS": "https", "JSON": "json", "XML": "xml", "ID": "id",
		"UTF": "utf", "TCP": "tcp", "UDP": "udp", "IP": "ip",
		"TLS": "tls", "SSL": "ssl", "API": "api", "SQL": "sql",
		"DNS": "dns", "EOF": "eof", "FMA": "fma",
	}
	for abbr, lower := range abbreviations {
		s = strings.ReplaceAll(s, abbr, "_"+lower+"_")
	}
	// Clean up double underscores and leading/trailing underscores
	var result []rune
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				if unicode.IsLower(prev) || unicode.IsDigit(prev) {
					result = append(result, '_')
				}
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	// Clean up: collapse multiple underscores, trim leading/trailing
	out := strings.Trim(string(result), "_")
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	return out
}

// goTypeConst returns the Go source for a GoType constant.
func goTypeConst(t gobridge.GoType) string {
	switch t {
	case gobridge.GoString:
		return "GoString"
	case gobridge.GoInt:
		return "GoInt"
	case gobridge.GoFloat64:
		return "GoFloat64"
	case gobridge.GoBool:
		return "GoBool"
	case gobridge.GoByte:
		return "GoByte"
	case gobridge.GoStringSlice:
		return "GoStringSlice"
	case gobridge.GoByteSlice:
		return "GoByteSlice"
	case gobridge.GoInt32:
		return "GoInt32"
	case gobridge.GoInt64:
		return "GoInt64"
	case gobridge.GoUint32:
		return "GoUint32"
	case gobridge.GoUint64:
		return "GoUint64"
	case gobridge.GoUint:
		return "GoUint"
	case gobridge.GoFloat32:
		return "GoFloat32"
	case gobridge.GoRune:
		return "GoRune"
	case gobridge.GoFunc:
		return "GoFunc"
	case gobridge.GoDuration:
		return "GoDuration"
	case gobridge.GoError:
		return "GoError"
	default:
		return "GoString"
	}
}

func printClassification(pkgPath string, funcs []bridgedFunc) error {
	counts := map[bridgeTier]int{}
	for _, f := range funcs {
		counts[f.Tier]++
	}

	fmt.Printf("Package: %s\n", pkgPath)
	fmt.Printf("Total: %d functions\n", len(funcs))
	fmt.Printf("  auto:     %d\n", counts[tierAuto])
	fmt.Printf("  castable: %d\n", counts[tierCastable])
	fmt.Printf("  func:     %d\n", counts[tierFunc])
	fmt.Printf("  blocked:  %d\n\n", counts[tierBlocked])

	for _, f := range funcs {
		marker := "✓"
		if f.Tier == tierBlocked {
			marker = "✗"
		} else if f.Tier == tierFunc {
			marker = "λ"
		} else if f.Tier == tierCastable {
			marker = "~"
		}
		line := fmt.Sprintf("  %s %-25s → %-25s [%s]", marker, f.GoName, f.RugoName, f.Tier)
		if f.Reason != "" {
			line += "  // " + f.Reason
		}
		fmt.Println(line)
	}
	return nil
}

func writeBridgeFile(pkgPath string, funcs []bridgedFunc) error {
	ns := gobridge.DefaultNS(pkgPath)
	fileName := filepath.Join("gobridge", ns+"_gen.go")

	// Filter to generatable functions (auto + castable)
	var genFuncs []bridgedFunc
	var skipped []bridgedFunc
	for _, f := range funcs {
		if f.Tier == tierAuto || f.Tier == tierCastable {
			genFuncs = append(genFuncs, f)
		} else {
			skipped = append(skipped, f)
		}
	}

	var sb strings.Builder
	sb.WriteString("// Code generated by rugo dev bridgegen; DO NOT EDIT.\n\n")
	sb.WriteString("package gobridge\n\n")
	sb.WriteString("func init() {\n")
	sb.WriteString("\tRegister(&Package{\n")
	sb.WriteString(fmt.Sprintf("\t\tPath: %q,\n", pkgPath))
	sb.WriteString(fmt.Sprintf("\t\tDoc:  \"Functions from Go's %s package.\",\n", pkgPath))
	sb.WriteString("\t\tFuncs: map[string]GoFuncSig{\n")

	for _, f := range genFuncs {
		sb.WriteString(formatFuncEntry(f))
	}

	// Add skipped functions as comments
	if len(skipped) > 0 {
		sb.WriteString("\t\t\t// --- Skipped (need custom Codegen or _custom.go) ---\n")
		for _, f := range skipped {
			sb.WriteString(fmt.Sprintf("\t\t\t// %s (%s): %s\n", f.RugoName, f.Tier, f.Reason))
		}
	}

	sb.WriteString("\t\t},\n")
	sb.WriteString("\t})\n")
	sb.WriteString("}\n")

	formatted, err := format.Source([]byte(sb.String()))
	if err != nil {
		return fmt.Errorf("formatting %s: %w", fileName, err)
	}

	if err := os.WriteFile(fileName, formatted, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", fileName, err)
	}

	fmt.Printf("Generated %s\n", fileName)
	fmt.Printf("  %d functions generated, %d skipped\n", len(genFuncs), len(skipped))

	// Generate smoke tests — skip functions overridden by _custom.go
	customFuncs := gobridge.PackageFuncs(pkgPath)
	var testFuncs []bridgedFunc
	for _, f := range genFuncs {
		if customFuncs != nil {
			if existing, ok := customFuncs[f.RugoName]; ok {
				// Custom override exists — skip if signature differs
				if len(existing.Params) != len(f.Params) {
					continue
				}
			}
		}
		testFuncs = append(testFuncs, f)
	}
	testFile := filepath.Join("rats", "gobridge", "auto", ns+"_gen_test.rugo")
	if err := writeTestFile(testFile, pkgPath, ns, testFuncs); err != nil {
		return fmt.Errorf("writing tests: %w", err)
	}
	fmt.Printf("Generated %s\n", testFile)

	return nil
}

func formatFuncEntry(f bridgedFunc) string {
	var sb strings.Builder

	// Params
	paramStrs := make([]string, len(f.Params))
	for i, p := range f.Params {
		paramStrs[i] = goTypeConst(p)
	}
	paramList := "nil"
	if len(paramStrs) > 0 {
		paramList = "[]GoType{" + strings.Join(paramStrs, ", ") + "}"
	}

	// Returns
	retStrs := make([]string, len(f.Returns))
	for i, r := range f.Returns {
		retStrs[i] = goTypeConst(r)
	}
	retList := "nil"
	if len(retStrs) > 0 {
		retList = "[]GoType{" + strings.Join(retStrs, ", ") + "}"
	}

	sb.WriteString(fmt.Sprintf("\t\t\t%q: {GoName: %q, Params: %s, Returns: %s",
		f.RugoName, f.GoName, paramList, retList))

	if f.Variadic {
		sb.WriteString(", Variadic: true")
	}

	if len(f.FuncTypes) > 0 {
		var parts []string
		for idx, ft := range f.FuncTypes {
			var fParams, fRets []string
			for _, p := range ft.Params {
				fParams = append(fParams, goTypeConst(p))
			}
			for _, r := range ft.Returns {
				fRets = append(fRets, goTypeConst(r))
			}
			pList := "nil"
			if len(fParams) > 0 {
				pList = "[]GoType{" + strings.Join(fParams, ", ") + "}"
			}
			rList := "nil"
			if len(fRets) > 0 {
				rList = "[]GoType{" + strings.Join(fRets, ", ") + "}"
			}
			parts = append(parts, fmt.Sprintf("%d: {Params: %s, Returns: %s}", idx, pList, rList))
		}
		sb.WriteString(fmt.Sprintf(", FuncTypes: map[int]*GoFuncType{%s}", strings.Join(parts, ", ")))
	}

	if len(f.ArrayTypes) > 0 {
		// Emit ArrayTypes metadata
		var parts []string
		for idx, at := range f.ArrayTypes {
			parts = append(parts, fmt.Sprintf("%d: {Elem: %s, Size: %d}", idx, goTypeConst(at.Elem), at.Size))
		}
		sb.WriteString(fmt.Sprintf(", ArrayTypes: map[int]*GoArrayType{%s}", strings.Join(parts, ", ")))
	}

	if f.Doc != "" {
		sb.WriteString(fmt.Sprintf(", Doc: %q", f.Doc))
	}

	sb.WriteString("},\n")
	return sb.String()
}

// writeTestFile generates smoke tests for all bridged functions.
func writeTestFile(path, pkgPath, ns string, funcs []bridgedFunc) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("# Auto-generated smoke tests by rugo dev bridgegen; DO NOT EDIT.\n")
	sb.WriteString("use \"test\"\n")
	sb.WriteString(fmt.Sprintf("import %q\n\n", pkgPath))

	for _, f := range funcs {
		// Skip void functions and functions with complex return types
		if len(f.Returns) == 0 {
			continue
		}
		// Skip functions whose only return is error (void semantics)
		if len(f.Returns) == 1 && f.Returns[0] == gobridge.GoError {
			continue
		}
		// Skip GoFunc params — need lambda args we can't auto-generate
		// Skip GoError params — can't construct error values in Rugo
		hasFunc := false
		for _, p := range f.Params {
			if p == gobridge.GoFunc || p == gobridge.GoError {
				hasFunc = true
			}
		}
		if hasFunc {
			continue
		}

		args := testArgs(f)
		expectedType := testReturnType(f)
		if expectedType == "" {
			continue
		}

		sb.WriteString(fmt.Sprintf("rats \"%s.%s is callable\"\n", ns, f.RugoName))
		if f.Variadic && len(f.Params) == 1 {
			sb.WriteString(fmt.Sprintf("  result = try %s.%s(%s) or nil\n", ns, f.RugoName, args[0]))
		} else {
			sb.WriteString(fmt.Sprintf("  result = try %s.%s(%s) or nil\n", ns, f.RugoName, strings.Join(args, ", ")))
		}
		sb.WriteString(fmt.Sprintf("  if result != nil\n"))
		sb.WriteString(fmt.Sprintf("    test.assert_eq(type_of(result), %q)\n", expectedType))
		sb.WriteString(fmt.Sprintf("  end\n"))
		sb.WriteString("end\n\n")
	}

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// testArgs generates safe zero-value arguments for a function call.
func testArgs(f bridgedFunc) []string {
	var args []string
	for _, p := range f.Params {
		args = append(args, testZeroValue(p))
	}
	return args
}

// testZeroValue returns a safe Rugo literal for a GoType.
func testZeroValue(t gobridge.GoType) string {
	switch t {
	case gobridge.GoString:
		return `"a"`
	case gobridge.GoInt:
		return "1"
	case gobridge.GoFloat64:
		return "1.0"
	case gobridge.GoBool:
		return "false"
	case gobridge.GoByte:
		return "0"
	case gobridge.GoStringSlice:
		return `["a"]`
	case gobridge.GoByteSlice:
		return `"a"`
	case gobridge.GoInt64:
		return "1"
	case gobridge.GoInt32:
		return "1"
	case gobridge.GoUint32:
		return "1"
	case gobridge.GoUint64:
		return "1"
	case gobridge.GoUint:
		return "1"
	case gobridge.GoFloat32:
		return "1.0"
	case gobridge.GoRune:
		return `"a"`
	case gobridge.GoDuration:
		return "1"
	default:
		return "nil"
	}
}

// testReturnType returns the expected Rugo type_of() string for a return pattern.
func testReturnType(f bridgedFunc) string {
	returns := f.Returns
	// (T, error) → T
	if len(returns) >= 2 && returns[len(returns)-1] == gobridge.GoError {
		returns = returns[:len(returns)-1]
	}
	// (T, bool) → T or nil
	if len(returns) == 2 && returns[1] == gobridge.GoBool {
		return "" // can't predict nil vs value
	}
	// Multi-return → Array
	if len(returns) > 1 {
		return "Array"
	}
	if len(returns) == 0 {
		return ""
	}
	switch returns[0] {
	case gobridge.GoString, gobridge.GoByteSlice:
		return "String"
	case gobridge.GoInt, gobridge.GoInt32, gobridge.GoInt64, gobridge.GoUint32, gobridge.GoUint64, gobridge.GoUint:
		return "Integer"
	case gobridge.GoFloat64, gobridge.GoFloat32:
		return "Float"
	case gobridge.GoBool:
		return "Bool"
	case gobridge.GoRune:
		return "String"
	case gobridge.GoStringSlice:
		return "Array"
	case gobridge.GoDuration:
		return "Integer"
	default:
		return ""
	}
}
