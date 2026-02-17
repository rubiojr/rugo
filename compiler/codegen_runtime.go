package compiler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rubiojr/rugo/gobridge"
	"github.com/rubiojr/rugo/modules"
)

func (g *codeGen) writeRuntime() {
	g.w.Raw(runtimeCorePre)

	// Module runtimes (only for use'd modules)
	for _, name := range importedModuleNames(g.imports) {
		if m, ok := modules.Get(name); ok {
			g.w.Raw(m.FullRuntime())
		}
	}

	g.w.Raw(runtimeCorePost)

	if g.hasSpawn || g.usesTaskMethods {
		g.writeSpawnRuntime()
	}

	// Sandbox runtime (Landlock self-sandboxing)
	if g.sandbox != nil {
		g.writeSandboxRuntime()
	}

	// Go bridge helpers (only if any Go packages are imported)
	if len(g.goImports) > 0 {
		g.writeGoBridgeRuntime()
	}
}

func (g *codeGen) writeSpawnRuntime() {
	g.w.Raw(runtimeSpawn)
}

// writeSandboxRuntime emits helper functions for Landlock-based sandboxing.
func (g *codeGen) writeSandboxRuntime() {
	g.w.Raw(`
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
`)
}

// writeSandboxApply emits the sandbox enforcement call inside main().
func (g *codeGen) writeSandboxApply() {
	// Environment variable filtering (works on all platforms, runs before Landlock).
	// Only active when env: was explicitly specified in the sandbox directive or CLI.
	cfg := g.sandbox
	if cfg.EnvSet {
		if len(cfg.Env) > 0 {
			// Save allowed env vars, clear all, restore allowed
			for i, name := range cfg.Env {
				g.writef("rugo_sandbox_env_%d := os.Getenv(%q)\n", i, name)
			}
			g.writeln("os.Clearenv()")
			for i, name := range cfg.Env {
				g.writef("if rugo_sandbox_env_%d != \"\" {\n", i)
				g.w.Indent()
				g.writef("os.Setenv(%q, rugo_sandbox_env_%d)\n", name, i)
				g.w.Dedent()
				g.writeln("}")
			}
		} else {
			// env: [] — clear all environment variables
			g.writeln("os.Clearenv()")
		}
	}

	g.writeln(`if runtime.GOOS != "linux" {`)
	g.w.Indent()
	g.writeln(`fmt.Fprintln(os.Stderr, "rugo: warning: sandbox requires Linux with Landlock support, running unrestricted")`)
	g.w.Dedent()
	g.writeln("} else {")
	g.w.Indent()
	g.writeln("rugo_sandbox_cfg := landlock.V5.BestEffort()")

	hasFS := len(cfg.RO) > 0 || len(cfg.RW) > 0 || len(cfg.ROX) > 0 || len(cfg.RWX) > 0
	hasNet := len(cfg.Connect) > 0 || len(cfg.Bind) > 0

	if hasFS {
		g.writeln("var rugo_sandbox_fs []landlock.Rule")
		for _, p := range cfg.RO {
			g.writef("rugo_sandbox_fs = append(rugo_sandbox_fs, landlock.PathAccess(rugo_sandbox_fs_ro(rugo_sandbox_is_dir(%q)), %q))\n", p, p)
		}
		for _, p := range cfg.RW {
			g.writef("rugo_sandbox_fs = append(rugo_sandbox_fs, landlock.PathAccess(rugo_sandbox_fs_rw(rugo_sandbox_is_dir(%q)), %q))\n", p, p)
		}
		for _, p := range cfg.ROX {
			g.writef("rugo_sandbox_fs = append(rugo_sandbox_fs, landlock.PathAccess(rugo_sandbox_fs_rox(rugo_sandbox_is_dir(%q)), %q))\n", p, p)
		}
		for _, p := range cfg.RWX {
			g.writef("rugo_sandbox_fs = append(rugo_sandbox_fs, landlock.PathAccess(rugo_sandbox_fs_rwx(rugo_sandbox_is_dir(%q)), %q))\n", p, p)
		}
	}

	if hasNet {
		g.writeln("var rugo_sandbox_net []landlock.Rule")
		for _, port := range cfg.Connect {
			g.writef("rugo_sandbox_net = append(rugo_sandbox_net, landlock.ConnectTCP(uint16(%d)))\n", port)
		}
		for _, port := range cfg.Bind {
			g.writef("rugo_sandbox_net = append(rugo_sandbox_net, landlock.BindTCP(uint16(%d)))\n", port)
		}
	}

	// Apply restrictions
	if !hasFS && !hasNet {
		// Bare sandbox: maximum restriction
		g.writeln(`if err := rugo_sandbox_cfg.Restrict(); err != nil {`)
		g.w.Indent()
		g.writeln(`fmt.Fprintf(os.Stderr, "rugo: warning: sandbox: %v\n", err)`)
		g.w.Dedent()
		g.writeln("}")
	} else {
		if hasFS {
			g.writeln("if err := rugo_sandbox_cfg.RestrictPaths(rugo_sandbox_fs...); err != nil {")
			g.w.Indent()
			g.writeln(`fmt.Fprintf(os.Stderr, "rugo: warning: sandbox filesystem: %v\n", err)`)
			g.w.Dedent()
			g.writeln("}")
		} else {
			// No FS rules but has net rules — still restrict FS (deny all)
			g.writeln("if err := rugo_sandbox_cfg.RestrictPaths(); err != nil {")
			g.w.Indent()
			g.writeln(`fmt.Fprintf(os.Stderr, "rugo: warning: sandbox filesystem: %v\n", err)`)
			g.w.Dedent()
			g.writeln("}")
		}
		if hasNet {
			g.writeln("if err := rugo_sandbox_cfg.RestrictNet(rugo_sandbox_net...); err != nil {")
			g.w.Indent()
			g.writeln(`fmt.Fprintf(os.Stderr, "rugo: warning: sandbox network: %v\n", err)`)
			g.w.Dedent()
			g.writeln("}")
		} else {
			// No net rules but has FS rules — still restrict net (deny all)
			g.writeln("if err := rugo_sandbox_cfg.RestrictNet(); err != nil {")
			g.w.Indent()
			g.writeln(`fmt.Fprintf(os.Stderr, "rugo: warning: sandbox network: %v\n", err)`)
			g.w.Dedent()
			g.writeln("}")
		}
	}

	g.w.Dedent()
	g.writeln("}")
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
func (g *codeGen) writeGoBridgeRuntime() {
	g.w.Raw("\n// --- Go Bridge Helpers ---\n\n")

	// Always emit rugo_to_byte_slice — it's used by TypeConvToGo for GoByteSlice
	// params in bridge calls, struct wrapper methods, and external type wrappers.
	g.w.Raw(gobridge.ByteSliceHelper.Code)

	emitted := map[string]bool{gobridge.ByteSliceHelper.Key: true}
	for pkg := range g.goImports {
		for _, h := range gobridge.AllRuntimeHelpers(pkg) {
			if !emitted[h.Key] {
				emitted[h.Key] = true
				g.w.Raw(h.Code)
			}
		}
		// Auto-emit helpers for GoType-based conversions
		if !emitted[gobridge.RuneHelper.Key] && gobridge.PackageNeedsRuneHelper(pkg) {
			emitted[gobridge.RuneHelper.Key] = true
			g.w.Raw(gobridge.RuneHelper.Code)
		}
		if !emitted[gobridge.StringSliceHelper.Key] && gobridge.PackageNeedsHelper(pkg, gobridge.GoStringSlice) {
			emitted[gobridge.StringSliceHelper.Key] = true
			g.w.Raw(gobridge.StringSliceHelper.Code)
		}
	}
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
