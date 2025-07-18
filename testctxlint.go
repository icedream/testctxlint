package testctxlint

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/mod/semver"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

var Analyzer = &analysis.Analyzer{
	Name:     "testctxlint",
	Doc:      "check for any code where test context could be used but isn't",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	URL:      "https://pkg.go.dev/github.com/icedream/testctxlint",
}

func goVersionAtLeast124(goVersion string) bool {
	if goVersion == "" { // Maybe the stdlib?
		return true
	}
	version := versionFromGoVersion(goVersion)
	return semver.Compare(version, "v1.24") >= 0
}

var inTest bool // So Go version checks can be skipped during testing.

type scope struct {
	// Scope-defining node
	ast.Node

	// The function that declares this scope
	funcType *ast.FuncType

	// Parent scope or nil
	parent *scope
}

func (s *scope) isAncestorOf(sub *scope) bool {
	for p := sub.parent; p != nil; p = p.parent {
		if p == s {
			return true
		}
	}
	return false
}

func (s *scope) findNearestBenchmarkOrTestParam() *ast.Ident {
	for current := s; current != nil; current = current.parent {
		if id := benchmarkOrTestParam(current.funcType); id != nil {
			return id
		}
	}
	return nil
}

// run applies the analyzer to a package.
// It returns an error if the analyzer failed.
//
// On success, the Run function may return a result
// computed by the Analyzer; its type must match ResultType.
// The driver makes this result available as an input to
// another Analyzer that depends directly on this one (see
// Requires) when it analyzes the same package.
//
// To pass analysis results between packages (and thus
// potentially between address spaces), use Facts, which are
// serializable.
func run(pass *analysis.Pass) (interface{}, error) {
	if !inTest {
		// check go version >= 1.24 (before then tb.Context didn't even exist)
		if !goVersionAtLeast124(goVersion(pass.Pkg)) {
			return nil, nil
		}
	}

	if !imports(pass.Pkg, "context") {
		// package is not even using the context package
		return nil, nil
	}

	// look for calls to context.TODO or context.Background
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Collect all of the scopes which have test/benchmark params.
	scopes := []*scope{}
	findScope := func(pos token.Pos) *scope {
		// log.Println("findScope", len(scopes), pos)
		var closestScope *scope
		for _, s := range scopes {
			// ignore scopes not containing this token
			if s.Pos() > pos || pos > s.End() {
				continue
			}
			// ignore less specific scopes
			if closestScope != nil && s.isAncestorOf(closestScope) {
				continue
			}
			closestScope = s
		}
		return closestScope
	}
	inspect.Nodes([]ast.Node{
		(*ast.CallExpr)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
		(*ast.GoStmt)(nil),
	}, func(node ast.Node, push bool) bool {
		if !push {
			return false
		}

		switch node := node.(type) {
		case *ast.FuncLit:
			result := benchmarkOrTestParam(node.Type)
			if result == nil {
				return false // not a test/benchmark method
			}
			scopes = append(scopes, &scope{
				Node:     node,
				funcType: node.Type,
				parent:   findScope(node.Pos()),
			})

		case *ast.FuncDecl:
			result := benchmarkOrTestParam(node.Type)
			if result == nil {
				return false // not a test/benchmark method
			}
			scopes = append(scopes, &scope{
				Node:     node,
				funcType: node.Type,
				parent:   findScope(node.Pos()),
			})

		case *ast.GoStmt:
			f := funcFromGoAsyncCall(node)
			if funcLit, ok := f.(*ast.FuncLit); ok {
				scopes = append(scopes, &scope{
					Node:     funcLit,
					funcType: funcLit.Type,
					parent:   findScope(funcLit.Pos()),
				})
			}

		case *ast.CallExpr:
			if f := funcFromBenchOrTestRunCall(pass.TypesInfo, node); f != nil {
				if funcLit, ok := f.(*ast.FuncLit); ok {
					scopes = append(scopes, &scope{
						Node:     funcLit,
						funcType: funcLit.Type,
						parent:   findScope(funcLit.Pos()),
					})
				}
				// return true
			}

			return false
		}
		return true
	})

	// Check each scope for context creation calls
	for _, s := range scopes {
		for n := range ast.Preorder(s.Node) {
			if n == s.Node {
				continue // always descend into the region itself.
			}

			if findScope(n.Pos()) != s {
				continue // will be revisited via another scope
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				continue
			}

			x, sel, fn := forbiddenMethod(pass.TypesInfo, call)
			if x == nil {
				continue
			}

			forbidden := formatMethod(sel, fn) // e.g. "(*testing.T).Forbidden

			tb := s.findNearestBenchmarkOrTestParam()
			if tb == nil {
				continue
			}

			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fmt.Sprintf("call to %s from a test routine", forbidden),
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: fmt.Sprintf("replace %s with call to %s.Context", forbidden, tb.Name),
						TextEdits: []analysis.TextEdit{
							{
								Pos:     call.Pos(),
								End:     call.End(),
								NewText: fmt.Appendf(nil, "%s.Context()", tb),
							},
						},
					},
				},
			})
			continue
		}
	}

	return nil, nil
}

func formatMethod(sel *types.Selection, fn *types.Func) string {
	if sel == nil {
		return fn.FullName()
	}
	var ptr string
	rtype := sel.Recv()
	if p, ok := types.Unalias(rtype).(*types.Pointer); ok {
		ptr = "*"
		rtype = p.Elem()
	}
	return fmt.Sprintf("(%s%s).%s", ptr, rtype.String(), fn.Name())
}

// forbiddenMethod decomposes a call x.m() into (x, x.m, m) where
// x is a variable/pkgName, x.m is a selection, and m is the static callee m.
// Returns (nil, nil, nil) if call is not of this form.
func forbiddenMethod(info *types.Info, call *ast.CallExpr) (types.Object, *types.Selection, *types.Func) {
	// Compare to typeutil.StaticCallee.
	fun := ast.Unparen(call.Fun)
	e := call.Fun
	var fn *types.Func
	selExpr, ok := fun.(*ast.SelectorExpr)
	var sel *types.Selection
	if ok {
		sel = info.Selections[selExpr]
		if sel != nil {
			e = ast.Unparen(selExpr.X)

			fn, _ = sel.Obj().(*types.Func)
			if fn == nil {
				return nil, nil, nil
			}
		}
	}

	// If no selection found for expr, use call itself
	if fn == nil {
		fn = typeutil.StaticCallee(info, call)
		if fn == nil {
			return nil, nil, nil
		}
	}

	if !isContextCreationFn(fn) {
		return nil, nil, nil
	}

	var x types.Object
	if id, ok := e.(*ast.Ident); ok {
		switch v := info.Uses[id].(type) {
		case *types.PkgName:
			x = v
		case *types.Var:
			x = v
		default:
			return nil, nil, nil
		}
	} else {
		x = fn
	}

	return x, sel, fn
}

func benchmarkOrTestParam(fnTypeDecl *ast.FuncType) *ast.Ident {
	// Check that the function's arguments include "*testing.T" or "*testing.B".
	params := fnTypeDecl.Params.List

	for _, param := range params {
		if _, ok := typeIsTestingDotTOrB(param.Type); ok {
			if len(param.Names) > 0 {
				return param.Names[0]
			}
		}
	}

	return nil
}

func typeIsTestingDotTOrB(expr ast.Expr) (string, bool) {
	starExpr, ok := expr.(*ast.StarExpr)
	if !ok {
		return "", false
	}
	selExpr, ok := starExpr.X.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	varPkg := selExpr.X.(*ast.Ident)
	if varPkg.Name != "testing" {
		return "", false
	}

	varTypeName := selExpr.Sel.Name
	ok = varTypeName == "B" || varTypeName == "T"
	return varTypeName, ok
}

// isContextCreationFn reports whether the given func reference points to:
// - context.TODO
// - context.Background
func isContextCreationFn(fn *types.Func) bool {
	if fn == nil {
		panic("got nil for isContextCreationFn")
	}
	return isFunctionNamed(fn, "context", "TODO", "Background") ||
		isMethodNamed(fn, "context", "TODO", "Background")
}

// isMethodNamed reports when a function f is a method,
// in a package with the path pkgPath and the name of f is in names.
//
// (Unlike [analysisinternal.IsMethodNamed], it ignores the receiver type name.)
func isMethodNamed(f *types.Func, pkgPath string, names ...string) bool {
	if f == nil {
		return false
	}
	if f.Pkg() == nil || f.Pkg().Path() != pkgPath {
		return false // not at pkgPath
	}
	if f.Type().(*types.Signature).Recv() == nil {
		return false // not a method
	}
	for _, n := range names {
		if f.Name() == n {
			return true
		}
	}
	return false // not in names
}

// funcFromGoAsyncCall returns the func of a call from a go fun() statement.
func funcFromGoAsyncCall(goStmt *ast.GoStmt) ast.Expr {
	return funcFromGoCall(goStmt.Call)
}

// funcFromGoCall returns the func of a call statement.
func funcFromGoCall(call *ast.CallExpr) ast.Expr {
	fun := ast.Unparen(call.Fun)
	return fun
}

// funcFromBenchOrTestRunCall returns the func of a call from a t.Run("name", fun) expression.
func funcFromBenchOrTestRunCall(info *types.Info, call *ast.CallExpr) ast.Expr {
	if len(call.Args) != 2 {
		return nil
	}
	run := typeutil.Callee(info, call)
	if run, ok := run.(*types.Func); !ok || !isMethodNamed(run, "testing", "Run") {
		return nil
	}

	fun := ast.Unparen(call.Args[1])
	return fun
}
