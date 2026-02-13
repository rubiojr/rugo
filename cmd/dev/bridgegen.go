package dev

import (
	"context"
	"fmt"
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
		Usage:     "Scaffold a Go bridge package",
		ArgsUsage: "<go-package>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Preview classification without writing files",
			},
			&cli.BoolFlag{
				Name:  "expand",
				Usage: "Add new functions to an existing bridge file (skip already registered)",
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
	Tier     bridgeTier
	Reason     string // why it was blocked
	Params     []gobridge.GoType
	Returns    []gobridge.GoType
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
	expand := cmd.Bool("expand")

	imp := importer.Default()
	pkg, err := imp.Import(pkgPath)
	if err != nil {
		return fmt.Errorf("importing %s: %w", pkgPath, err)
	}

	// Collect existing bridge functions if expanding
	existing := map[string]bool{}
	if expand {
		if funcs := gobridge.PackageFuncs(pkgPath); funcs != nil {
			for name := range funcs {
				existing[name] = true
			}
		}
	}

	// Enumerate and classify exported functions
	scope := pkg.Scope()
	var funcs []bridgedFunc
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		fn, ok := obj.(*types.Func)
		if !ok {
			continue
		}
		// Skip unexported functions
		if !fn.Exported() {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sig.Recv() != nil {
			continue // skip methods
		}
		rugoName := toSnakeCase(name)
		if expand && existing[rugoName] {
			continue
		}
		bf := classifyFunc(name, rugoName, sig)
		funcs = append(funcs, bf)
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

	return writeBridgeFile(pkgPath, funcs, expand)
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
			bf.Tier = tierFunc
			bf.Reason = fmt.Sprintf("param %d: func type", i)
			return bf
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
		case types.Int32: // rune is int32
			return gobridge.GoRune, tierCastable, ""
		case types.Float32:
			return gobridge.GoFloat64, tierCastable, ""
		case types.Int8, types.Int16:
			return gobridge.GoInt, tierCastable, ""
		case types.Uint, types.Uint16, types.Uint32, types.Uint64:
			return gobridge.GoInt, tierCastable, ""
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
	case gobridge.GoInt64:
		return "GoInt64"
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

func writeBridgeFile(pkgPath string, funcs []bridgedFunc, expand bool) error {
	ns := gobridge.DefaultNS(pkgPath)
	fileName := filepath.Join("gobridge", ns+".go")

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

	if expand {
		return writeExpandedBridge(fileName, pkgPath, genFuncs, skipped)
	}

	var sb strings.Builder
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
		sb.WriteString("\t\t\t// --- Skipped (need custom Codegen) ---\n")
		for _, f := range skipped {
			sb.WriteString(fmt.Sprintf("\t\t\t// %s (%s): %s\n", f.RugoName, f.Tier, f.Reason))
		}
	}

	sb.WriteString("\t\t},\n")
	sb.WriteString("\t})\n")
	sb.WriteString("}\n")

	if _, err := os.Stat(fileName); err == nil && !expand {
		fmt.Printf("File %s already exists. Use --expand to add new functions.\n", fileName)
		fmt.Println("\nGenerated code (stdout):")
		fmt.Print(sb.String())
		return nil
	}

	if err := os.WriteFile(fileName, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", fileName, err)
	}

	fmt.Printf("Created %s\n", fileName)
	fmt.Printf("  %d functions generated, %d skipped\n", len(genFuncs), len(skipped))
	return nil
}

func writeExpandedBridge(fileName, pkgPath string, genFuncs, skipped []bridgedFunc) error {
	if len(genFuncs) == 0 && len(skipped) == 0 {
		fmt.Println("No new functions to add.")
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n// --- New functions for %s (generated by bridgegen) ---\n", pkgPath))
	for _, f := range genFuncs {
		sb.WriteString(formatFuncEntry(f))
	}
	if len(skipped) > 0 {
		sb.WriteString("\t\t\t// --- Skipped (need custom Codegen) ---\n")
		for _, f := range skipped {
			sb.WriteString(fmt.Sprintf("\t\t\t// %s (%s): %s\n", f.RugoName, f.Tier, f.Reason))
		}
	}

	fmt.Printf("Add these entries to %s:\n\n", fileName)
	fmt.Print(sb.String())
	fmt.Printf("\n  %d functions generated, %d skipped\n", len(genFuncs), len(skipped))
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
