package compiler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rubiojr/rugo/gobridge"
	"github.com/rubiojr/rugo/modules"
)

func (g *codeGen) buildRuntimeCode() string {
	var sb strings.Builder
	sb.WriteString(runtimeCorePre)

	for _, name := range importedModuleNames(g.imports) {
		if m, ok := modules.Get(name); ok {
			sb.WriteString(m.FullRuntime())
		}
	}

	sb.WriteString(runtimeCorePost)

	if g.hasSpawn || g.usesTaskMethods {
		sb.WriteString(runtimeSpawn)
	}

	if g.sandbox != nil {
		sb.WriteString(g.sandboxRuntimeCode())
	}

	if len(g.goImports) > 0 {
		sb.WriteString(g.goBridgeRuntimeCode())
	}
	return sb.String()
}

// sandboxRuntimeCode returns helper functions for Landlock-based sandboxing.
func (g *codeGen) sandboxRuntimeCode() string {
	return `
func rugo_sandbox_fs_ro(dir bool) landlock.AccessFSSet {
	rights := landlock.AccessFSSet(llsyscall.AccessFSReadFile)
	if dir { rights |= landlock.AccessFSSet(llsyscall.AccessFSReadDir) }
	return rights
}

func rugo_sandbox_fs_rw(dir bool) landlock.AccessFSSet {
	rights := landlock.AccessFSSet(llsyscall.AccessFSReadFile) |
		landlock.AccessFSSet(llsyscall.AccessFSWriteFile) |
		landlock.AccessFSSet(llsyscall.AccessFSTruncate) |
		landlock.AccessFSSet(llsyscall.AccessFSIoctlDev)
	if dir {
		rights |= landlock.AccessFSSet(llsyscall.AccessFSReadDir) |
			landlock.AccessFSSet(llsyscall.AccessFSRemoveDir) |
			landlock.AccessFSSet(llsyscall.AccessFSRemoveFile) |
			landlock.AccessFSSet(llsyscall.AccessFSMakeChar) |
			landlock.AccessFSSet(llsyscall.AccessFSMakeDir) |
			landlock.AccessFSSet(llsyscall.AccessFSMakeReg) |
			landlock.AccessFSSet(llsyscall.AccessFSMakeSock) |
			landlock.AccessFSSet(llsyscall.AccessFSMakeFifo) |
			landlock.AccessFSSet(llsyscall.AccessFSMakeBlock) |
			landlock.AccessFSSet(llsyscall.AccessFSMakeSym) |
			landlock.AccessFSSet(llsyscall.AccessFSRefer)
	}
	return rights
}

func rugo_sandbox_fs_rox(dir bool) landlock.AccessFSSet {
	rights := landlock.AccessFSSet(llsyscall.AccessFSExecute) |
		landlock.AccessFSSet(llsyscall.AccessFSReadFile)
	if dir { rights |= landlock.AccessFSSet(llsyscall.AccessFSReadDir) }
	return rights
}

func rugo_sandbox_fs_rwx(dir bool) landlock.AccessFSSet {
	rights := rugo_sandbox_fs_rw(dir) | landlock.AccessFSSet(llsyscall.AccessFSExecute)
	return rights
}

func rugo_sandbox_is_dir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
`
}

// buildSandboxApply returns GoStmt nodes for sandbox enforcement inside main().
func (g *codeGen) buildSandboxApply() []GoStmt {
	cfg := g.sandbox
	var stmts []GoStmt

	// Environment variable filtering
	if cfg.EnvSet {
		if len(cfg.Env) > 0 {
			for i, name := range cfg.Env {
				stmts = append(stmts, GoRawStmt{Code: fmt.Sprintf("rugo_sandbox_env_%d := os.Getenv(%q)", i, name)})
			}
			stmts = append(stmts, GoRawStmt{Code: "os.Clearenv()"})
			for i, name := range cfg.Env {
				stmts = append(stmts, GoIfStmt{
					Cond: fmtExpr(`rugo_sandbox_env_%d != ""`, i),
					Body: []GoStmt{GoRawStmt{Code: fmt.Sprintf("os.Setenv(%q, rugo_sandbox_env_%d)", name, i)}},
				})
			}
		} else {
			stmts = append(stmts, GoRawStmt{Code: "os.Clearenv()"})
		}
	}

	// Platform check + Landlock setup
	hasFS := len(cfg.RO) > 0 || len(cfg.RW) > 0 || len(cfg.ROX) > 0 || len(cfg.RWX) > 0
	hasNet := len(cfg.Connect) > 0 || len(cfg.Bind) > 0

	var landlockBody []GoStmt
	landlockBody = append(landlockBody, GoRawStmt{Code: "rugo_sandbox_cfg := landlock.V5.BestEffort()"})

	if hasFS {
		landlockBody = append(landlockBody, GoRawStmt{Code: "var rugo_sandbox_fs []landlock.Rule"})
		for _, p := range cfg.RO {
			landlockBody = append(landlockBody, GoRawStmt{Code: fmt.Sprintf("rugo_sandbox_fs = append(rugo_sandbox_fs, landlock.PathAccess(rugo_sandbox_fs_ro(rugo_sandbox_is_dir(%q)), %q))", p, p)})
		}
		for _, p := range cfg.RW {
			landlockBody = append(landlockBody, GoRawStmt{Code: fmt.Sprintf("rugo_sandbox_fs = append(rugo_sandbox_fs, landlock.PathAccess(rugo_sandbox_fs_rw(rugo_sandbox_is_dir(%q)), %q))", p, p)})
		}
		for _, p := range cfg.ROX {
			landlockBody = append(landlockBody, GoRawStmt{Code: fmt.Sprintf("rugo_sandbox_fs = append(rugo_sandbox_fs, landlock.PathAccess(rugo_sandbox_fs_rox(rugo_sandbox_is_dir(%q)), %q))", p, p)})
		}
		for _, p := range cfg.RWX {
			landlockBody = append(landlockBody, GoRawStmt{Code: fmt.Sprintf("rugo_sandbox_fs = append(rugo_sandbox_fs, landlock.PathAccess(rugo_sandbox_fs_rwx(rugo_sandbox_is_dir(%q)), %q))", p, p)})
		}
	}

	if hasNet {
		landlockBody = append(landlockBody, GoRawStmt{Code: "var rugo_sandbox_net []landlock.Rule"})
		for _, port := range cfg.Connect {
			landlockBody = append(landlockBody, GoRawStmt{Code: fmt.Sprintf("rugo_sandbox_net = append(rugo_sandbox_net, landlock.ConnectTCP(uint16(%d)))", port)})
		}
		for _, port := range cfg.Bind {
			landlockBody = append(landlockBody, GoRawStmt{Code: fmt.Sprintf("rugo_sandbox_net = append(rugo_sandbox_net, landlock.BindTCP(uint16(%d)))", port)})
		}
	}

	errHandler := func(msg string) GoIfStmt {
		return GoIfStmt{
			Cond: fmtExpr("err := %s; err != nil", msg),
			Body: []GoStmt{GoRawStmt{Code: fmt.Sprintf(`fmt.Fprintf(os.Stderr, "rugo: warning: sandbox: %%v\n", err)`)}},
		}
	}

	if !hasFS && !hasNet {
		landlockBody = append(landlockBody, errHandler("rugo_sandbox_cfg.Restrict()"))
	} else {
		if hasFS {
			landlockBody = append(landlockBody, GoIfStmt{
				Cond: rawExpr("err := rugo_sandbox_cfg.RestrictPaths(rugo_sandbox_fs...); err != nil"),
				Body: []GoStmt{GoRawStmt{Code: `fmt.Fprintf(os.Stderr, "rugo: warning: sandbox filesystem: %v\n", err)`}},
			})
		} else {
			landlockBody = append(landlockBody, GoIfStmt{
				Cond: rawExpr("err := rugo_sandbox_cfg.RestrictPaths(); err != nil"),
				Body: []GoStmt{GoRawStmt{Code: `fmt.Fprintf(os.Stderr, "rugo: warning: sandbox filesystem: %v\n", err)`}},
			})
		}
		if hasNet {
			landlockBody = append(landlockBody, GoIfStmt{
				Cond: rawExpr("err := rugo_sandbox_cfg.RestrictNet(rugo_sandbox_net...); err != nil"),
				Body: []GoStmt{GoRawStmt{Code: `fmt.Fprintf(os.Stderr, "rugo: warning: sandbox network: %v\n", err)`}},
			})
		} else {
			landlockBody = append(landlockBody, GoIfStmt{
				Cond: rawExpr("err := rugo_sandbox_cfg.RestrictNet(); err != nil"),
				Body: []GoStmt{GoRawStmt{Code: `fmt.Fprintf(os.Stderr, "rugo: warning: sandbox network: %v\n", err)`}},
			})
		}
	}

	stmts = append(stmts, GoIfStmt{
		Cond: rawExpr(`runtime.GOOS != "linux"`),
		Body: []GoStmt{GoRawStmt{Code: `fmt.Fprintln(os.Stderr, "rugo: warning: sandbox requires Linux with Landlock support, running unrestricted")`}},
		Else: landlockBody,
	})

	return stmts
}

// writeDispatchMaps generates typed dispatch maps for modules that declare DispatchEntry.
// Each map maps user-defined function names to their Go implementations.
// When a module provides DispatchTransform, only functions matching transformed
// handler names from the source are included. Otherwise all eligible functions are included.

// sortedGoBridgeImports returns sorted package paths from goImports map.
func sortedGoBridgeImports(goImports map[string]string) []string {
	var pkgs []string
	for pkg := range goImports {
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)
	return pkgs
}

// generateGoBridgeCall generates a Go expression for a direct Go bridge call.
// rugoName is the user-visible name (e.g. "strconv.atoi") for error messages.
func (g *codeGen) generateGoBridgeCall(pkg string, sig *gobridge.GoFuncSig, argExprs []string, rugoName string) string {
	// Determine the Go package prefix to use in generated code
	pkgBase := gobridge.DefaultNS(pkg)
	if alias, ok := g.goImports[pkg]; ok && alias != "" {
		pkgBase = alias
	}

	// Custom codegen callback — bridge file owns its own logic
	if sig.Codegen != nil {
		return sig.Codegen(pkgBase, argExprs, rugoName)
	}

	// Struct return decomposition — auto-generate hash from struct fields
	if sig.StructReturn != nil {
		var convertedArgs []string
		for i, arg := range argExprs {
			if i < len(sig.Params) {
				convertedArgs = append(convertedArgs, gobridge.TypeConvToGo(arg, sig.Params[i]))
			}
		}
		call := fmt.Sprintf("%s.%s(%s)", pkgBase, sig.GoName, strings.Join(convertedArgs, ", "))
		hashCode := gobridge.StructDecompCode("_v", sig.StructReturn)
		panicFmt := fmt.Sprintf(`panic(rugo_bridge_err("%s", _err))`, rugoName)

		// Determine return pattern: struct alone, (struct, error), etc.
		hasError := len(sig.Returns) >= 2 && sig.Returns[len(sig.Returns)-1] == gobridge.GoError
		if hasError {
			return fmt.Sprintf("func() interface{} {\n\t_v, _err := %s\n\tif _err != nil { %s }\n\treturn %s\n}()",
				call, panicFmt, hashCode)
		}
		return fmt.Sprintf("func() interface{} {\n\t_v := %s\n\treturn %s\n}()",
			call, hashCode)
	}

	// Build converted args
	// For variadic functions, the last entry in sig.Params is a slice type
	// (e.g., GoStringSlice for ...string). We convert variadic args using the
	// element type so Go's variadic calling convention works naturally.
	var variadicElemType gobridge.GoType
	variadicIdx := -1
	if sig.Variadic && len(sig.Params) > 0 {
		lastParam := sig.Params[len(sig.Params)-1]
		if elem, ok := gobridge.SliceElemType(lastParam); ok {
			variadicElemType = elem
			variadicIdx = len(sig.Params) - 1
		}
	}

	var convertedArgs []string
	for i, arg := range argExprs {
		// Variadic args: convert using element type (e.g., GoString for ...string)
		if variadicIdx >= 0 && i >= variadicIdx {
			convertedArgs = append(convertedArgs, gobridge.TypeConvToGo(arg, variadicElemType))
			continue
		}
		if i < len(sig.Params) {
			// Struct handle unwrapping: extract the inner Go struct from the wrapper.
			if sig.StructCasts != nil {
				if wrapType, ok := sig.StructCasts[i]; ok {
					convertedArgs = append(convertedArgs, fmt.Sprintf("rugo_upcast_%s(%s).v", wrapType, arg))
					continue
				}
			}
			if sig.Params[i] == gobridge.GoFunc && sig.FuncTypes != nil {
				if ft, ok := sig.FuncTypes[i]; ok {
					convertedArgs = append(convertedArgs, gobridge.FuncAdapterConv(arg, ft))
					continue
				}
			}
			conv := gobridge.TypeConvToGo(arg, sig.Params[i])
			// Apply named type cast if specified
			if sig.TypeCasts != nil {
				if cast, ok := sig.TypeCasts[i]; ok {
					if strings.HasPrefix(cast, "*") {
						conv = fmt.Sprintf("*%s(%s)", cast[1:], conv)
					} else {
						conv = fmt.Sprintf("%s(%s)", cast, conv)
					}
				}
			}
			convertedArgs = append(convertedArgs, conv)
		}
	}

	// Build call expression (supports method-chain GoNames like "StdEncoding.EncodeToString")
	call := fmt.Sprintf("%s.%s(%s)", pkgBase, sig.GoName, strings.Join(convertedArgs, ", "))

	// Error panic format with Rugo function name
	panicFmt := fmt.Sprintf(`panic(rugo_bridge_err("%s", _err))`, rugoName)

	// Handle return types
	if len(sig.Returns) == 0 {
		// Void Go functions need wrapping since Rugo assigns all expressions
		return fmt.Sprintf("func() interface{} { %s; return nil }()", call)
	}

	// Helper: wrap return expression, handling fixed-size arrays and struct wrapping
	wrapRet := func(expr string, retIdx int, t gobridge.GoType) string {
		// Struct return wrapping: wrap Go struct pointer into opaque wrapper.
		if sig.StructReturnWraps != nil {
			if wrapType, ok := sig.StructReturnWraps[retIdx]; ok {
				return fmt.Sprintf("interface{}(&%s{v: %s})", wrapType, expr)
			}
		}
		if sig.ArrayTypes != nil {
			if _, ok := sig.ArrayTypes[retIdx]; ok {
				// Slice fixed array to dynamic slice: _v[:] then wrap
				return gobridge.TypeWrapReturn(expr+"[:]", t)
			}
		}
		return gobridge.TypeWrapReturn(expr, t)
	}

	if len(sig.Returns) == 1 {
		// Single error return: panic on error, return nil
		if sig.Returns[0] == gobridge.GoError {
			return fmt.Sprintf("func() interface{} { if _err := %s; _err != nil { %s }; return nil }()", call, panicFmt)
		}
		// Struct return wrapping or fixed-size array returns need IIFE
		if sig.StructReturnWraps != nil {
			if _, ok := sig.StructReturnWraps[0]; ok {
				return fmt.Sprintf("func() interface{} { _v := %s; return %s }()", call, wrapRet("_v", 0, sig.Returns[0]))
			}
		}
		if sig.ArrayTypes != nil {
			if _, ok := sig.ArrayTypes[0]; ok {
				return fmt.Sprintf("func() interface{} { _v := %s; return %s }()", call, wrapRet("_v", 0, sig.Returns[0]))
			}
		}
		return gobridge.TypeWrapReturn(call, sig.Returns[0])
	}

	// (T, error): panic on error
	if len(sig.Returns) == 2 && sig.Returns[1] == gobridge.GoError {
		return fmt.Sprintf("func() interface{} { _v, _err := %s; if _err != nil { %s }; return %s }()",
			call, panicFmt, wrapRet("_v", 0, sig.Returns[0]))
	}

	// (T, bool): return nil if false
	if len(sig.Returns) == 2 && sig.Returns[1] == gobridge.GoBool {
		return fmt.Sprintf("func() interface{} { _v, _ok := %s; if !_ok { return nil }; return %s }()",
			call, wrapRet("_v", 0, sig.Returns[0]))
	}

	// Multi-return: collect all values into []interface{} array
	n := len(sig.Returns)
	vars := make([]string, n)
	for i := range vars {
		vars[i] = fmt.Sprintf("_v%d", i)
	}
	assign := strings.Join(vars, ", ")
	var elems []string
	for i, v := range vars {
		elems = append(elems, gobridge.TypeWrapReturn(v, sig.Returns[i]))
	}
	arr := "[]interface{}{" + strings.Join(elems, ", ") + "}"
	return fmt.Sprintf("func() interface{} { %s := %s; return %s }()", assign, call, arr)
}

// writeGoBridgeRuntime emits helper functions needed by Go bridge calls.
// Helpers are declared by bridge files via RuntimeHelpers on GoFuncSig,
// deduplicated by key, and emitted once.
func (g *codeGen) goBridgeRuntimeCode() string {
	var sb strings.Builder
	sb.WriteString("\n// --- Go Bridge Helpers ---\n\n")

	sb.WriteString(gobridge.ByteSliceHelper.Code)

	emitted := map[string]bool{gobridge.ByteSliceHelper.Key: true}
	for pkg := range g.goImports {
		for _, h := range gobridge.AllRuntimeHelpers(pkg) {
			if !emitted[h.Key] {
				emitted[h.Key] = true
				sb.WriteString(h.Code)
			}
		}
		if !emitted[gobridge.RuneHelper.Key] && gobridge.PackageNeedsRuneHelper(pkg) {
			emitted[gobridge.RuneHelper.Key] = true
			sb.WriteString(gobridge.RuneHelper.Code)
		}
		if !emitted[gobridge.StringSliceHelper.Key] && gobridge.PackageNeedsHelper(pkg, gobridge.GoStringSlice) {
			emitted[gobridge.StringSliceHelper.Key] = true
			sb.WriteString(gobridge.StringSliceHelper.Code)
		}
	}
	return sb.String()
}

// arityCountError produces a human-friendly argument count error for range arity.
func arityCountError(name string, got int, arity funcArity) error {
	gotDesc := fmt.Sprintf("%d were", got)
	if got == 0 {
		gotDesc = "none were"
	} else if got == 1 {
		gotDesc = "1 was"
	}
	if arity.Min == arity.Max {
		argWord := "arguments"
		if arity.Max == 1 {
			argWord = "argument"
		}
		return fmt.Errorf("%s() takes %d %s but %s given", name, arity.Max, argWord, gotDesc)
	}
	return fmt.Errorf("%s() takes %d to %d arguments but %s given", name, arity.Min, arity.Max, gotDesc)
}

// argCountError produces a human-friendly argument count mismatch error.
func argCountError(name string, got, expected int) error {
	return arityCountError(name, got, funcArity{Min: expected, Max: expected})
}
