package testctxlint

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"strings"

	"golang.org/x/mod/semver"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer is the main instance of the testctxlinter analyzer.
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

// inTest indicates if the analyzer is running as part of a test. This will
// disable the Go version check.
var inTest = len(os.Args) > 0 && strings.HasSuffix(strings.TrimSuffix(os.Args[0], ".exe"), ".test")

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
	if !shouldAnalyze(pass) {
		return nil, nil
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	scopeCol := collectScopes(inspect, pass)
	checkScopesForForbiddenCalls(pass, scopeCol)

	return nil, nil
}

func shouldAnalyze(pass *analysis.Pass) bool {
	if !inTest {
		// check go version >= 1.24 (before then tb.Context didn't even exist)
		if !goVersionAtLeast124(goVersion(pass.Pkg)) {
			return false
		}
	}

	if !imports(pass.Pkg, "context") {
		// package is not even using the context package
		return false
	}

	return true
}

func collectScopes(inspect *inspector.Inspector, pass *analysis.Pass) *scopeCollection {
	scopeCol := &scopeCollection{}

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
			if result := benchmarkOrTestParam(node.Type); result != nil {
				scopeCol.add(&scope{
					Node:     node,
					funcType: node.Type,
					parent:   scopeCol.findScope(node.Pos()),
				})
			}

		case *ast.FuncDecl:
			if result := benchmarkOrTestParam(node.Type); result != nil {
				scopeCol.add(&scope{
					Node:     node,
					funcType: node.Type,
					parent:   scopeCol.findScope(node.Pos()),
				})
			}

		case *ast.GoStmt:
			f := funcFromGoAsyncCall(node)
			if funcLit, ok := f.(*ast.FuncLit); ok {
				scopeCol.add(&scope{
					Node:     funcLit,
					funcType: funcLit.Type,
					parent:   scopeCol.findScope(funcLit.Pos()),
				})
			}

		case *ast.CallExpr:
			if f := funcFromBenchOrTestRunCall(pass.TypesInfo, node); f != nil {
				if funcLit, ok := f.(*ast.FuncLit); ok {
					scopeCol.add(&scope{
						Node:     funcLit,
						funcType: funcLit.Type,
						parent:   scopeCol.findScope(funcLit.Pos()),
					})
				}
			}

			return false
		}

		return true
	})

	return scopeCol
}

func checkScopesForForbiddenCalls(pass *analysis.Pass, scopeCol *scopeCollection) {
	for _, s := range scopeCol.scopes {
		checkScopeForForbiddenCalls(pass, s, scopeCol)
	}
}

// checkScopeForForbiddenCalls checks a single scope for forbidden context calls
func checkScopeForForbiddenCalls(pass *analysis.Pass, s *scope, scopeCol *scopeCollection) {
	// Use ast.Inspect for more efficient traversal of just this scope's subtree
	ast.Inspect(s.Node, func(n ast.Node) bool {
		if n == nil || n == s.Node {
			return true // always descend into the scope itself
		}

		// Skip nodes that belong to a different (nested) scope
		if foundScope := scopeCol.findScope(n.Pos()); foundScope != nil && foundScope != s {
			return false // will be handled when processing that scope
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		x, sel, fn := forbiddenMethod(pass.TypesInfo, call)
		if x == nil {
			return true
		}

		forbidden := formatMethod(sel, fn)

		tbInfo := s.findNearestBenchmarkOrTestParamWithInfo()
		if tbInfo == nil {
			return true
		}

		reportForbiddenCall(pass, call, forbidden, tbInfo)

		return true
	})
}

func reportForbiddenCall(pass *analysis.Pass, call *ast.CallExpr, forbidden string, tbInfo *testingParam) {
	message := "replace " + forbidden + " with " + tbInfo.ident.Name + ".Context"
	edits := []analysis.TextEdit{
		{
			// Replace context creation call
			Pos:     call.Pos(),
			End:     call.End(),
			NewText: []byte(tbInfo.ident.Name + ".Context()"),
		},
	}

	if tbInfo.isUnnamed {
		message = "name parameter as " + tbInfo.ident.Name + " and " + message
		edits = append([]analysis.TextEdit{
			{
				// Add parameter name before the type
				Pos:     tbInfo.param.Type.Pos(),
				End:     tbInfo.param.Type.Pos(),
				NewText: []byte(tbInfo.ident.Name + " "),
			},
		}, edits...)
	}

	pass.Report(analysis.Diagnostic{
		Pos:     call.Pos(),
		End:     call.End(),
		Message: fmt.Sprintf("call to %s from a test routine", forbidden),
		SuggestedFixes: []analysis.SuggestedFix{
			{
				Message:   message,
				TextEdits: edits,
			},
		},
	})
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

type testingParam struct {
	ident     *ast.Ident
	isUnnamed bool
	param     *ast.Field // The original parameter for unnamed params
}

func benchmarkOrTestParam(fnTypeDecl *ast.FuncType) *ast.Ident {
	// Check that the function's arguments include "*testing.T" or "*testing.B".
	params := fnTypeDecl.Params.List

	for _, param := range params {
		if testingType, ok := typeIsTestingDotTOrB(param.Type); ok {
			if len(param.Names) > 0 {
				return param.Names[0]
			}
			// Handle unnamed testing parameters by creating a synthetic identifier
			// with a conventional name based on the testing type
			var name string

			switch testingType {
			case "T":
				name = "t"
			case "B":
				name = "b"
			}

			return &ast.Ident{
				Name: name,
				// Use the position of the type for the synthetic identifier
				NamePos: param.Type.Pos(),
			}
		}
	}

	return nil
}

func benchmarkOrTestParamWithInfo(fnTypeDecl *ast.FuncType) *testingParam {
	// Check that the function's arguments include "*testing.T" or "*testing.B".
	params := fnTypeDecl.Params.List

	for _, param := range params {
		if testingType, ok := typeIsTestingDotTOrB(param.Type); ok {
			if len(param.Names) > 0 {
				return &testingParam{
					ident:     param.Names[0],
					isUnnamed: false,
					param:     param,
				}
			}
			// Handle unnamed testing parameters by creating a synthetic identifier
			// with a conventional name based on the testing type
			var name string

			switch testingType {
			case "T":
				name = "t"
			case "B":
				name = "b"
			}

			return &testingParam{
				ident: &ast.Ident{
					Name: name,
					// Use the position of the type for the synthetic identifier
					NamePos: param.Type.Pos(),
				},
				isUnnamed: true,
				param:     param,
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
