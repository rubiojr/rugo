package gobridge

import (
	"fmt"
	"go/types"
	"strings"
	"unicode"
)

// Tier classifies how a Go function can be bridged to Rugo.
type Tier int

const (
	TierAuto     Tier = iota // fully auto-generatable
	TierCastable             // needs int64/[]byte/rune casts
	TierFunc                 // has function parameters
	TierBlocked              // generics, interfaces, channels, etc.
)

func (t Tier) String() string {
	switch t {
	case TierAuto:
		return "auto"
	case TierCastable:
		return "castable"
	case TierFunc:
		return "func"
	case TierBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}

// ClassifiedFunc holds classification results for a Go function.
type ClassifiedFunc struct {
	GoName     string
	RugoName   string
	Sig        *types.Signature
	Tier       Tier
	Reason     string // why it was blocked
	Params     []GoType
	Returns    []GoType
	FuncTypes  map[int]*GoFuncType  // GoFunc param signatures
	ArrayTypes map[int]*GoArrayType // fixed-size array return metadata
	Variadic   bool
	Doc        string
}

// ClassifyFunc determines how a Go function can be bridged.
func ClassifyFunc(goName, rugoName string, sig *types.Signature) ClassifiedFunc {
	bf := ClassifiedFunc{
		GoName:   goName,
		RugoName: rugoName,
		Sig:      sig,
		Variadic: sig.Variadic(),
	}

	params := sig.Params()
	hasCast := false
	for i := 0; i < params.Len(); i++ {
		t := params.At(i).Type()
		gt, tier, reason := ClassifyGoType(t, true)
		if tier == TierBlocked {
			bf.Tier = TierBlocked
			bf.Reason = fmt.Sprintf("param %d: %s", i, reason)
			return bf
		}
		if tier == TierFunc {
			funcSig, ok := params.At(i).Type().Underlying().(*types.Signature)
			if !ok {
				bf.Tier = TierFunc
				bf.Reason = fmt.Sprintf("param %d: func type (not a signature)", i)
				return bf
			}
			ft := ClassifyFuncType(funcSig)
			if ft == nil {
				bf.Tier = TierFunc
				bf.Reason = fmt.Sprintf("param %d: func with unbridgeable signature", i)
				return bf
			}
			if bf.FuncTypes == nil {
				bf.FuncTypes = map[int]*GoFuncType{}
			}
			bf.FuncTypes[i] = ft
			hasCast = true
			bf.Params = append(bf.Params, gt)
			continue
		}
		if tier == TierCastable {
			hasCast = true
		}
		bf.Params = append(bf.Params, gt)
	}

	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		t := results.At(i).Type()
		gt, tier, reason := ClassifyGoType(t, false)
		if tier == TierBlocked {
			bf.Tier = TierBlocked
			bf.Reason = fmt.Sprintf("return %d: %s", i, reason)
			return bf
		}
		if tier == TierCastable {
			hasCast = true
		}
		if arr, ok := t.Underlying().(*types.Array); ok {
			if bf.ArrayTypes == nil {
				bf.ArrayTypes = map[int]*GoArrayType{}
			}
			elemGT, _, _ := ClassifyGoType(arr.Elem(), false)
			bf.ArrayTypes[i] = &GoArrayType{Elem: elemGT, Size: int(arr.Len())}
		}
		bf.Returns = append(bf.Returns, gt)
	}

	if hasCast {
		bf.Tier = TierCastable
	} else {
		bf.Tier = TierAuto
	}
	return bf
}

// ClassifyFuncType builds a GoFuncType from a Go function signature.
// Returns nil if any param/return type is unbridgeable.
func ClassifyFuncType(sig *types.Signature) *GoFuncType {
	ft := &GoFuncType{}

	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		t := params.At(i).Type()
		gt, tier, _ := ClassifyGoType(t, true)
		if tier == TierBlocked || tier == TierFunc {
			return nil
		}
		ft.Params = append(ft.Params, gt)
		// Detect named types that need explicit casts in the adapter.
		if named, ok := t.(*types.Named); ok {
			if pkg := named.Obj().Pkg(); pkg != nil {
				if _, isBasic := named.Underlying().(*types.Basic); isBasic {
					if ft.TypeCasts == nil {
						ft.TypeCasts = make(map[int]string)
					}
					ft.TypeCasts[i] = pkg.Name() + "." + named.Obj().Name()
				}
			}
		}
	}

	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		t := results.At(i).Type()
		gt, tier, _ := ClassifyGoType(t, false)
		if tier == TierBlocked || tier == TierFunc {
			return nil
		}
		ft.Returns = append(ft.Returns, gt)
		// Detect named return types that need explicit casts in the adapter.
		if named, ok := t.(*types.Named); ok {
			if pkg := named.Obj().Pkg(); pkg != nil {
				if _, isBasic := named.Underlying().(*types.Basic); isBasic {
					if ft.TypeCasts == nil {
						ft.TypeCasts = make(map[int]string)
					}
					// Use negative index to distinguish return casts from param casts.
					ft.TypeCasts[-(i + 1)] = pkg.Name() + "." + named.Obj().Name()
				}
			}
		}
	}

	return ft
}

// ClassifyGoType maps a Go type to a GoType and tier.
func ClassifyGoType(t types.Type, isParam bool) (GoType, Tier, string) {
	if named, ok := t.(*types.Named); ok {
		if named.Obj().Pkg() == nil && named.Obj().Name() == "error" {
			return GoError, TierAuto, ""
		}
		t = t.Underlying()
	}

	switch u := t.(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.String:
			return GoString, TierAuto, ""
		case types.Int:
			return GoInt, TierAuto, ""
		case types.Float64:
			return GoFloat64, TierAuto, ""
		case types.Bool:
			return GoBool, TierAuto, ""
		case types.Byte:
			return GoByte, TierCastable, ""
		case types.Int64:
			return GoInt64, TierCastable, ""
		case types.Int32:
			if u.Name() == "rune" {
				return GoRune, TierCastable, ""
			}
			return GoInt32, TierCastable, ""
		case types.Float32:
			return GoFloat32, TierCastable, ""
		case types.Int8, types.Int16:
			return GoInt, TierCastable, ""
		case types.Uint16:
			return GoInt, TierCastable, ""
		case types.Uint:
			return GoUint, TierCastable, ""
		case types.Uint32:
			return GoUint32, TierCastable, ""
		case types.Uint64:
			return GoUint64, TierCastable, ""
		default:
			return 0, TierBlocked, fmt.Sprintf("unsupported basic type %s", u.Name())
		}
	case *types.Slice:
		elem := u.Elem()
		if b, ok := elem.Underlying().(*types.Basic); ok {
			switch b.Kind() {
			case types.String:
				return GoStringSlice, TierAuto, ""
			case types.Byte:
				return GoByteSlice, TierCastable, ""
			}
		}
		return 0, TierBlocked, fmt.Sprintf("unsupported slice type []%s", elem)
	case *types.Signature:
		return GoFunc, TierFunc, "function parameter"
	case *types.Interface:
		return 0, TierBlocked, "interface type"
	case *types.Pointer:
		return 0, TierBlocked, fmt.Sprintf("pointer to %s", u.Elem())
	case *types.Struct:
		return 0, TierBlocked, "struct type"
	case *types.Map:
		return 0, TierBlocked, "map type"
	case *types.Chan:
		return 0, TierBlocked, "channel type"
	case *types.Array:
		if b, ok := u.Elem().Underlying().(*types.Basic); ok && b.Kind() == types.Byte {
			return GoByteSlice, TierCastable, ""
		}
		return 0, TierBlocked, fmt.Sprintf("array type [%d]%s", u.Len(), u.Elem())
	default:
		return 0, TierBlocked, fmt.Sprintf("unknown type %T", t)
	}
}

// ToSnakeCase converts PascalCase to snake_case.
// Handles consecutive uppercase (e.g., "IsNaN" → "is_nan", "FMA" → "fma").
func ToSnakeCase(s string) string {
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
	out := strings.Trim(string(result), "_")
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	return out
}

// GoTypeConst returns the Go source for a GoType constant.
func GoTypeConst(t GoType) string {
	switch t {
	case GoString:
		return "GoString"
	case GoInt:
		return "GoInt"
	case GoFloat64:
		return "GoFloat64"
	case GoBool:
		return "GoBool"
	case GoByte:
		return "GoByte"
	case GoStringSlice:
		return "GoStringSlice"
	case GoByteSlice:
		return "GoByteSlice"
	case GoInt32:
		return "GoInt32"
	case GoInt64:
		return "GoInt64"
	case GoUint32:
		return "GoUint32"
	case GoUint64:
		return "GoUint64"
	case GoUint:
		return "GoUint"
	case GoFloat32:
		return "GoFloat32"
	case GoRune:
		return "GoRune"
	case GoFunc:
		return "GoFunc"
	case GoDuration:
		return "GoDuration"
	case GoError:
		return "GoError"
	default:
		return "GoString"
	}
}
