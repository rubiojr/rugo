package compiler

import (
	"fmt"
	"github.com/rubiojr/rugo/ast"
	"strings"

	"github.com/rubiojr/rugo/gobridge"
	"github.com/rubiojr/rugo/modules"
)

// builtinFuncs are always-available function names.
var builtinFuncs = map[string]bool{
	"puts":           true,
	"print":          true,
	"len":            true,
	"append":         true,
	"raise":          true,
	"exit":           true,
	"type_of":        true,
	"range":          true,
	"__shell__":      true,
	"__capture__":    true,
	"__pipe_shell__": true,
}

// identCheck implements ast.Check and reports undefined identifier references.
type identCheck struct {
	sourceFile string
}

// UndefinedIdentCheck returns a Check that reports undefined identifier references.
func UndefinedIdentCheck(sourceFile string) ast.Check {
	return &identCheck{sourceFile: sourceFile}
}

func (ic *identCheck) Name() string { return "undefined-ident" }

func (ic *identCheck) Check(prog *ast.Program) error {
	// Collect all globally-visible identifiers.
	global := make(map[string]bool)

	// Builtins
	for k := range builtinFuncs {
		global[k] = true
	}

	// Struct constructors (expanded to FuncDefs by preprocessor)
	for _, si := range prog.Structs {
		global[si.Name] = true
	}

	// First pass: collect top-level names (functions, variables, modules, namespaces).
	var funcs []*ast.FuncDef
	namespaces := make(map[string]bool)
	imports := make(map[string]bool)       // Rugo stdlib modules
	goImports := make(map[string]string)   // Go bridge: path → alias
	nsVarNames := make(map[string]bool)    // "ns.var" qualified names
	funcDefs := make(map[string]bool)      // function names (including namespaced)
	withNamespaces := make(map[string]bool) // namespaces from "with" requires

	for _, s := range prog.Statements {
		switch st := s.(type) {
		case *ast.FuncDef:
			if st.Namespace != "" {
				namespaces[st.Namespace] = true
				global[st.Namespace] = true
				funcDefs[st.Namespace+"."+st.Name] = true
			} else {
				global[st.Name] = true
				funcDefs[st.Name] = true
			}
			funcs = append(funcs, st)
		case *ast.UseStmt:
			imports[st.Module] = true
			global[st.Module] = true
		case *ast.ImportStmt:
			ns := goBridgeNamespace(st)
			goImports[st.Package] = st.Alias
			global[ns] = true
			// Also register the Go package's default namespace (last path component)
			// since codegen resolves both aliased and default names.
			parts := strings.Split(st.Package, "/")
			defaultNS := parts[len(parts)-1]
			global[defaultNS] = true
		case *ast.RequireStmt:
			if len(st.With) > 0 {
				for _, name := range st.With {
					withNamespaces[name] = true
					global[name] = true
				}
			} else {
				ns := st.Alias
				if ns == "" {
					ns = requireNamespace(st.Path)
				}
				if ns != "" {
					namespaces[ns] = true
					global[ns] = true
				}
			}
		case *ast.AssignStmt:
			if st.Namespace != "" {
				namespaces[st.Namespace] = true
				global[st.Namespace] = true
				nsVarNames[st.Namespace+"."+st.Target] = true
			} else {
				global[st.Target] = true
			}
		}
	}

	w := &identWalker{
		sourceFile:     ic.sourceFile,
		global:         global,
		namespaces:     namespaces,
		imports:        imports,
		goImports:      goImports,
		nsVarNames:     nsVarNames,
		funcDefs:       funcDefs,
		withNamespaces: withNamespaces,
	}

	// Check top-level statements (non-function bodies)
	topScope := make(map[string]bool)
	for _, s := range prog.Statements {
		switch s.(type) {
		case *ast.FuncDef, *ast.UseStmt, *ast.ImportStmt, *ast.RequireStmt, *ast.SandboxStmt:
			continue
		}
		if assign, ok := s.(*ast.AssignStmt); ok && assign.Namespace != "" {
			continue
		}
		if err := w.checkStmt(s, topScope); err != nil {
			return err
		}
	}

	// Collect sibling names per namespace (functions and constants in the same required file).
	nsSiblings := make(map[string]map[string]bool) // namespace → set of bare names
	for _, f := range funcs {
		if f.Namespace != "" {
			if nsSiblings[f.Namespace] == nil {
				nsSiblings[f.Namespace] = make(map[string]bool)
			}
			nsSiblings[f.Namespace][f.Name] = true
		}
	}
	for _, s := range prog.Statements {
		if a, ok := s.(*ast.AssignStmt); ok && a.Namespace != "" {
			if nsSiblings[a.Namespace] == nil {
				nsSiblings[a.Namespace] = make(map[string]bool)
			}
			nsSiblings[a.Namespace][a.Target] = true
		}
	}

	// Check function bodies
	for _, f := range funcs {
		scope := make(map[string]bool)
		for _, p := range f.Params {
			scope[p.Name] = true
		}
		// Add sibling names from the same namespace
		if f.Namespace != "" {
			for name := range nsSiblings[f.Namespace] {
				scope[name] = true
			}
		}
		for _, s := range f.Body {
			if err := w.checkStmt(s, scope); err != nil {
				return err
			}
		}
	}

	// Check test and bench blocks
	for _, s := range prog.Statements {
		switch st := s.(type) {
		case *ast.TestDef:
			scope := make(map[string]bool)
			for _, bs := range st.Body {
				if err := w.checkStmt(bs, scope); err != nil {
					return err
				}
			}
		case *ast.BenchDef:
			scope := make(map[string]bool)
			for _, bs := range st.Body {
				if err := w.checkStmt(bs, scope); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// identWalker walks AST nodes checking identifier references against known scopes.
type identWalker struct {
	sourceFile     string
	global         map[string]bool
	namespaces     map[string]bool
	imports        map[string]bool
	goImports      map[string]string
	nsVarNames     map[string]bool
	funcDefs       map[string]bool
	withNamespaces map[string]bool
}

// isDefined checks if name is known in the local scope or global scope.
func (w *identWalker) isDefined(name string, localScope map[string]bool) bool {
	if localScope != nil && localScope[name] {
		return true
	}
	return w.global[name]
}

// checkStmt validates identifier references in a statement.
// localScope is nil for top-level statements.
func (w *identWalker) checkStmt(s ast.Statement, localScope map[string]bool) error {
	switch st := s.(type) {
	case *ast.AssignStmt:
		if err := w.checkExpr(st.Value, st.StmtLine(), localScope); err != nil {
			return err
		}
		// Assignment introduces the variable
		if localScope != nil {
			localScope[st.Target] = true
		}
	case *ast.IndexAssignStmt:
		if err := w.checkExpr(st.Object, st.StmtLine(), localScope); err != nil {
			return err
		}
		if err := w.checkExpr(st.Index, st.StmtLine(), localScope); err != nil {
			return err
		}
		if err := w.checkExpr(st.Value, st.StmtLine(), localScope); err != nil {
			return err
		}
	case *ast.DotAssignStmt:
		if err := w.checkExpr(st.Object, st.StmtLine(), localScope); err != nil {
			return err
		}
		if err := w.checkExpr(st.Value, st.StmtLine(), localScope); err != nil {
			return err
		}
	case *ast.ExprStmt:
		if err := w.checkExpr(st.Expression, st.StmtLine(), localScope); err != nil {
			return err
		}
	case *ast.ReturnStmt:
		if st.Value != nil {
			if err := w.checkExpr(st.Value, st.StmtLine(), localScope); err != nil {
				return err
			}
		}
	case *ast.ImplicitReturnStmt:
		if err := w.checkExpr(st.Value, st.StmtLine(), localScope); err != nil {
			return err
		}
	case *ast.TryResultStmt:
		if err := w.checkExpr(st.Value, st.StmtLine(), localScope); err != nil {
			return err
		}
	case *ast.SpawnReturnStmt:
		if st.Value != nil {
			if err := w.checkExpr(st.Value, st.StmtLine(), localScope); err != nil {
				return err
			}
		}
	case *ast.TryHandlerReturnStmt:
		if st.Value != nil {
			if err := w.checkExpr(st.Value, st.StmtLine(), localScope); err != nil {
				return err
			}
		}
	case *ast.IfStmt:
		if err := w.checkExpr(st.Condition, st.StmtLine(), localScope); err != nil {
			return err
		}
		// if/else don't create their own scope — variables leak out
		for _, bs := range st.Body {
			if err := w.checkStmt(bs, localScope); err != nil {
				return err
			}
		}
		for _, ec := range st.ElsifClauses {
			if err := w.checkExpr(ec.Condition, st.StmtLine(), localScope); err != nil {
				return err
			}
			for _, bs := range ec.Body {
				if err := w.checkStmt(bs, localScope); err != nil {
					return err
				}
			}
		}
		for _, bs := range st.ElseBody {
			if err := w.checkStmt(bs, localScope); err != nil {
				return err
			}
		}
	case *ast.WhileStmt:
		if err := w.checkExpr(st.Condition, st.StmtLine(), localScope); err != nil {
			return err
		}
		// while creates its own scope but can read/modify outer vars
		innerScope := childScope(localScope)
		for _, bs := range st.Body {
			if err := w.checkStmt(bs, innerScope); err != nil {
				return err
			}
		}
	case *ast.ForStmt:
		if err := w.checkExpr(st.Collection, st.StmtLine(), localScope); err != nil {
			return err
		}
		innerScope := childScope(localScope)
		innerScope[st.Var] = true
		if st.IndexVar != "" {
			innerScope[st.IndexVar] = true
		}
		for _, bs := range st.Body {
			if err := w.checkStmt(bs, innerScope); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkExpr validates identifier references in an expression.
func (w *identWalker) checkExpr(e ast.Expr, line int, localScope map[string]bool) error {
	if e == nil {
		return nil
	}
	switch ex := e.(type) {
	case *ast.IdentExpr:
		if !w.isDefined(ex.Name, localScope) {
			return w.undefinedError(ex.Name, line)
		}
	case *ast.CallExpr:
		// Check for namespaced calls: ns.func(args)
		if dot, ok := ex.Func.(*ast.DotExpr); ok {
			if ns, ok := dot.Object.(*ast.IdentExpr); ok {
				if err := w.checkNamespacedCall(ns.Name, dot.Field, line, localScope); err != nil {
					return err
				}
				// Args still need checking
				for _, a := range ex.Args {
					if err := w.checkExpr(a, line, localScope); err != nil {
						return err
					}
				}
				return nil
			}
		}
		// Regular call: check the function name and args
		if err := w.checkExpr(ex.Func, line, localScope); err != nil {
			return err
		}
		for _, a := range ex.Args {
			if err := w.checkExpr(a, line, localScope); err != nil {
				return err
			}
		}
	case *ast.DotExpr:
		// Check the receiver
		if err := w.checkExpr(ex.Object, line, localScope); err != nil {
			return err
		}
		// For namespace access (non-call), validate the field exists
		if ns, ok := ex.Object.(*ast.IdentExpr); ok {
			if err := w.checkNamespacedAccess(ns.Name, ex.Field, line, localScope); err != nil {
				return err
			}
		}
	case *ast.BinaryExpr:
		if err := w.checkExpr(ex.Left, line, localScope); err != nil {
			return err
		}
		if err := w.checkExpr(ex.Right, line, localScope); err != nil {
			return err
		}
	case *ast.UnaryExpr:
		if err := w.checkExpr(ex.Operand, line, localScope); err != nil {
			return err
		}
	case *ast.IndexExpr:
		if err := w.checkExpr(ex.Object, line, localScope); err != nil {
			return err
		}
		if err := w.checkExpr(ex.Index, line, localScope); err != nil {
			return err
		}
	case *ast.SliceExpr:
		if err := w.checkExpr(ex.Object, line, localScope); err != nil {
			return err
		}
		if err := w.checkExpr(ex.Start, line, localScope); err != nil {
			return err
		}
		if err := w.checkExpr(ex.Length, line, localScope); err != nil {
			return err
		}
	case *ast.ArrayLiteral:
		for _, el := range ex.Elements {
			if err := w.checkExpr(el, line, localScope); err != nil {
				return err
			}
		}
	case *ast.HashLiteral:
		for _, p := range ex.Pairs {
			if err := w.checkExpr(p.Key, line, localScope); err != nil {
				return err
			}
			if err := w.checkExpr(p.Value, line, localScope); err != nil {
				return err
			}
		}
	case *ast.FnExpr:
		// Lambda: create inner scope with params
		innerScope := childScope(localScope)
		for _, p := range ex.Params {
			innerScope[p.Name] = true
		}
		for _, bs := range ex.Body {
			if err := w.checkStmt(bs, innerScope); err != nil {
				return err
			}
		}
	case *ast.LoweredTryExpr:
		if err := w.checkExpr(ex.Expr, line, localScope); err != nil {
			return err
		}
		innerScope := childScope(localScope)
		if ex.ErrVar != "" {
			innerScope[ex.ErrVar] = true
		}
		for _, bs := range ex.Handler {
			if err := w.checkStmt(bs, innerScope); err != nil {
				return err
			}
		}
		if ex.ResultExpr != nil {
			if err := w.checkExpr(ex.ResultExpr, line, innerScope); err != nil {
				return err
			}
		}
	case *ast.LoweredSpawnExpr:
		innerScope := childScope(localScope)
		for _, bs := range ex.Body {
			if err := w.checkStmt(bs, innerScope); err != nil {
				return err
			}
		}
		if ex.ResultExpr != nil {
			if err := w.checkExpr(ex.ResultExpr, line, innerScope); err != nil {
				return err
			}
		}
	case *ast.LoweredParallelExpr:
		for _, br := range ex.Branches {
			if br.Expr != nil {
				if err := w.checkExpr(br.Expr, line, localScope); err != nil {
					return err
				}
			}
			for _, bs := range br.Stmts {
				if err := w.checkStmt(bs, localScope); err != nil {
					return err
				}
			}
		}
	case *ast.StringLiteral:
		// Skip interpolation checking — interpolation expressions embedded in
		// strings (e.g. test.run snippets) may reference variables from a
		// different compilation scope. Interpolation idents are validated
		// during codegen when the Go compiler processes the generated fmt.Sprintf.
	}
	return nil
}


func (w *identWalker) undefinedError(name string, line int) error {
	src := w.sourceFile
	if src == "" {
		src = "<unknown>"
	}
	return fmt.Errorf("%s:%d: undefined variable '%s'", src, line, name)
}

// isLocalVar checks if name is a local variable that shadows a namespace/module.
// In codegen, isDeclared() takes priority over namespace/module lookups.
func (w *identWalker) isLocalVar(name string, localScope map[string]bool) bool {
	return localScope != nil && localScope[name]
}

// checkNamespacedCall validates ns.func() calls against known function registries.
// Local variables shadow namespaces (e.g. if `x = hash`, then x.keys() is a method call).
func (w *identWalker) checkNamespacedCall(nsName, field string, line int, localScope map[string]bool) error {
	// Local variables shadow namespaces — treat as method call on value
	if w.isLocalVar(nsName, localScope) {
		return nil
	}

	// Rugo stdlib module call
	if w.imports[nsName] {
		if _, ok := modules.LookupFunc(nsName, field); !ok {
			return fmt.Errorf("%s:%d: unknown function %s.%s in module %q", w.sourceFile, line, nsName, field, nsName)
		}
		return nil
	}

	// Go bridge call
	if pkg, ok := gobridge.PackageForNS(nsName, w.goImports); ok {
		if _, ok := gobridge.Lookup(pkg, field); !ok {
			return fmt.Errorf("%s:%d: unknown function %s.%s in Go bridge package %q", w.sourceFile, line, nsName, field, pkg)
		}
		return nil
	}

	// Require namespace — check funcDefs and nsVarNames
	if w.namespaces[nsName] {
		nsKey := nsName + "." + field
		if !w.funcDefs[nsKey] && !w.nsVarNames[nsKey] {
			return fmt.Errorf("%s:%d: undefined function %s.%s (check that the function exists in the required module)", w.sourceFile, line, nsName, field)
		}
		return nil
	}

	// "with" namespace — functions are merged into global scope, dot calls not expected
	// Unknown receiver — could be a value with methods, skip
	return nil
}

// checkNamespacedAccess validates ns.field access (non-call) against known registries.
func (w *identWalker) checkNamespacedAccess(nsName, field string, line int, localScope map[string]bool) error {
	// Local variables shadow namespaces — treat as property access
	if w.isLocalVar(nsName, localScope) {
		return nil
	}

	// Require namespace — check funcDefs and nsVarNames
	if w.namespaces[nsName] {
		nsKey := nsName + "." + field
		if !w.funcDefs[nsKey] && !w.nsVarNames[nsKey] {
			return fmt.Errorf("%s:%d: undefined: %s.%s (check that the function or variable exists in the required module)", w.sourceFile, line, nsName, field)
		}
		return nil
	}

	// Module/bridge field access is handled at codegen — skip here
	return nil
}

// childScope creates a new scope that inherits from the parent.
// New assignments go to the child; lookups check child then parent via isDefined.
func childScope(parent map[string]bool) map[string]bool {
	child := make(map[string]bool)
	if parent != nil {
		for k := range parent {
			child[k] = true
		}
	}
	return child
}

// requireNamespace extracts the default namespace from a require path.
func requireNamespace(path string) string {
	// Strip version suffix
	if idx := strings.Index(path, "@"); idx > 0 {
		path = path[:idx]
	}
	// Use last path component
	base := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		base = path[idx+1:]
	}
	// Strip extension
	base = strings.TrimSuffix(base, ".rugo")
	base = strings.TrimSuffix(base, ".rg")
	return base
}


