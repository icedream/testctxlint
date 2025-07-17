package testctxlint

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/exp/typeparams"
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
type posEnd interface {
	Pos() token.Pos
	End() token.Pos
}

type scope struct {
	posEnd
	Type *ast.FuncType
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

	toDecl := localFunctionDecls(pass.TypesInfo, pass.Files)

	// asyncs maps nodes whose statements will be executed concurrently
	// with respect to some test function, to the call sites where they
	// are invoked asynchronously. There may be multiple such call sites
	// for e.g. test helpers.
	asyncs := make(map[ast.Node][]*extent)
	var regions []ast.Node
	addCall := func(c *extent) {
		if c != nil {
			r := c.region
			if asyncs[r] == nil {
				regions = append(regions, r)
			}
			asyncs[r] = append(asyncs[r], c)
		}
	}

	// Collect all of the go callee() and t.Run(name, callee) extents.
	funcScopes := []scope{}
	getCurrentFuncScope := func() scope {
		return funcScopes[len(funcScopes)-1]
	}
	inspect.Nodes([]ast.Node{
		(*ast.CallExpr)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
		(*ast.GoStmt)(nil),
	}, func(node ast.Node, push bool) bool {
		// check whether we just existed func scopes
		for i := len(funcScopes) - 1; i >= 0; i-- {
			if funcScopes[i].End() >= node.Pos() {
				continue
			}
			funcScopes = funcScopes[0:i]
		}

		if !push {
			return false
		}
		switch node := node.(type) {
		case *ast.FuncLit:
			result := benchmarkOrTestParam(node.Type)
			if result != nil {
				funcScopes = append(funcScopes, scope{posEnd: node, Type: node.Type})
				return true
			}
			return false // not a test/benchmark method

		case *ast.FuncDecl:
			result := benchmarkOrTestParam(node.Type)
			if result != nil {
				funcScopes = append(funcScopes, scope{posEnd: node, Type: node.Type})
				return true
			}
			return false // not a test/benchmark method

		case *ast.GoStmt:
			c := goAsyncCall(pass.TypesInfo, node, toDecl)
			addCall(c)

		case *ast.CallExpr:
			if c := tRunCall(pass.TypesInfo, node); c != nil {
				addCall(c)
				return true
			}

			x, sel, fn := forbiddenMethod(pass.TypesInfo, node)
			if x == nil {
				return true
			}

			// check if already in one of the subtest regions
			for _, region := range regions {
				for _, e := range asyncs[region] {
					if exprWithinScope(e.scope, node) {
						return true
					}
				}
			}

			forbidden := formatMethod(sel, fn) // e.g. "context.TODO", "(*testing.T).Forbidden"

			var context string
			funcDecl := getCurrentFuncScope()
			tb := benchmarkOrTestParam(funcDecl.Type)
			pass.Report(analysis.Diagnostic{
				Pos:     node.Pos(),
				End:     node.End(),
				Message: fmt.Sprintf("call to %s from a test routine%s", forbidden, context),
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: fmt.Sprintf("replace %s with call to %s.Context", forbidden, tb.Name),
						TextEdits: []analysis.TextEdit{
							{
								Pos:     node.Pos(),
								End:     node.End(),
								NewText: fmt.Appendf(nil, "%s.Context()", tb),
							},
						},
					},
				},
			})
			return false
		}
		return true
	})

	// Check for t.Forbidden() calls within each region r that is a
	// callee in some go r() or a t.Run("name", r).
	//
	// Also considers a special case when r is a go t.Forbidden() call.
	for _, region := range regions {
		ast.Inspect(region, func(n ast.Node) bool {
			if n == region {
				return true // always descend into the region itself.
			} else if asyncs[n] != nil {
				return false // will be visited by another region.
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			x, sel, fn := forbiddenMethod(pass.TypesInfo, call)
			if x == nil {
				return true
			}

			for _, e := range asyncs[region] {
				if !withinScope(e.scope, x) {
					forbidden := formatMethod(sel, fn) // e.g. "(*testing.T).Forbidden

					var context string
					var where analysis.Range = e.async // Put the report at the go fun() or t.Run(name, fun).
					if _, local := e.fun.(*ast.FuncLit); local {
						where = call // Put the report at the t.Forbidden() call.
					} else if id, ok := e.fun.(*ast.Ident); ok {
						context = fmt.Sprintf(" (%s calls %s)", id.Name, forbidden)
					}
					if funcLit, ok := e.fun.(*ast.FuncLit); ok {
						tb := benchmarkOrTestParam(funcLit.Type)
						pass.Report(analysis.Diagnostic{
							Pos:     where.Pos(),
							End:     where.End(),
							Message: fmt.Sprintf("call to %s from a test subroutine%s", forbidden, context),
							SuggestedFixes: []analysis.SuggestedFix{
								{
									Message: fmt.Sprintf("replace %s with call to %s.Context", forbidden, tb),
									TextEdits: []analysis.TextEdit{
										{
											Pos:     where.Pos(),
											End:     where.End(),
											NewText: fmt.Appendf(nil, "%s.Context()", tb),
										},
									},
								},
							},
						})
					}
				}
			}
			return true
		})
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

// isContextCreationCall reports whether call is a static call to one of:
// - context.TODO
// - context.Background
func isContextCreationCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	fn := typeutil.StaticCallee(pass.TypesInfo, call)
	if fn == nil {
		return false
	}
	return isContextCreationFn(fn)
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

// withinScope returns true if x.Pos() is in [scope.Pos(), scope.End()].
func withinScope(scope ast.Node, x types.Object) bool {
	if scope != nil {
		return x.Pos() != token.NoPos && scope.Pos() <= x.Pos() && x.Pos() <= scope.End()
	}
	return false
}

func exprWithinScope(scope ast.Node, x ast.Expr) bool {
	if scope != nil {
		return x.Pos() != token.NoPos && scope.Pos() <= x.Pos() && x.Pos() <= scope.End()
	}
	return false
}

// localFunctionDecls returns a mapping from *types.Func to *ast.FuncDecl in files.
func localFunctionDecls(info *types.Info, files []*ast.File) func(*types.Func) *ast.FuncDecl {
	var fnDecls map[*types.Func]*ast.FuncDecl // computed lazily
	return func(f *types.Func) *ast.FuncDecl {
		if f != nil && fnDecls == nil {
			fnDecls = make(map[*types.Func]*ast.FuncDecl)
			for _, file := range files {
				for _, decl := range file.Decls {
					if fnDecl, ok := decl.(*ast.FuncDecl); ok {
						if fn, ok := info.Defs[fnDecl.Name].(*types.Func); ok {
							fnDecls[fn] = fnDecl
						}
					}
				}
			}
		}
		// TODO: set f = f.Origin() here.
		return fnDecls[f]
	}
}

// extent describes a region of code that needs to be checked for
// t.Forbidden() calls as it is started asynchronously from an async
// node go fun() or t.Run(name, fun).
type extent struct {
	region ast.Node // region of code to check for t.Forbidden() calls.
	async  ast.Node // *ast.GoStmt or *ast.CallExpr (for t.Run)
	scope  ast.Node // Report t.Forbidden() if t is not declared within scope.
	fun    ast.Expr // fun in go fun() or t.Run(name, fun)
}

// goAsyncCall returns the extent of a call from a go fun() statement.
func goAsyncCall(info *types.Info, goStmt *ast.GoStmt, toDecl func(*types.Func) *ast.FuncDecl) *extent {
	return goCall(info, goStmt, goStmt.Call, toDecl)
}

// goCall returns the extent of a call statement.
func goCall(info *types.Info, region ast.Node, call *ast.CallExpr, toDecl func(*types.Func) *ast.FuncDecl) *extent {
	fun := ast.Unparen(call.Fun)
	if id := funcIdent(fun); id != nil {
		if lit := funcLitInScope(id); lit != nil {
			return &extent{region: lit, async: nil, scope: nil, fun: fun}
		}
	}

	if fn := typeutil.StaticCallee(info, call); fn != nil { // static call or method in the package?
		if decl := toDecl(fn); decl != nil {
			return &extent{region: decl, async: nil, scope: nil, fun: fun}
		}
	}

	// Check go statement for go t.Forbidden() or go func(){t.Forbidden()}().
	return &extent{region: region, async: nil, scope: nil, fun: fun}
}

// tRunCall returns the extent of a call from a t.Run("name", fun) expression.
func tRunCall(info *types.Info, call *ast.CallExpr) *extent {
	if len(call.Args) != 2 {
		return nil
	}
	run := typeutil.Callee(info, call)
	if run, ok := run.(*types.Func); !ok || !isMethodNamed(run, "testing", "Run") {
		return nil
	}

	fun := ast.Unparen(call.Args[1])
	if lit, ok := fun.(*ast.FuncLit); ok { // function lit?
		return &extent{region: lit, async: call, scope: lit, fun: fun}
	}

	if id := funcIdent(fun); id != nil {
		if lit := funcLitInScope(id); lit != nil { // function lit in variable?
			return &extent{region: lit, async: call, scope: lit, fun: fun}
		}
	}

	// Check within t.Run(name, fun) for calls to t.Forbidden,
	// e.g. t.Run(name, func(t *testing.T){ t.Forbidden() })
	return &extent{region: call, async: call, scope: fun, fun: fun}
}

func funcIdent(fun ast.Expr) *ast.Ident {
	switch fun := ast.Unparen(fun).(type) {
	case *ast.IndexExpr, *ast.IndexListExpr:
		x, _, _, _ := typeparams.UnpackIndexExpr(fun) // necessary?
		id, _ := x.(*ast.Ident)
		return id
	case *ast.Ident:
		return fun
	default:
		return nil
	}
}

// funcLitInScope returns a FuncLit that id is at least initially assigned to.
//
// TODO: This is closely tied to id.Obj which is deprecated.
func funcLitInScope(id *ast.Ident) *ast.FuncLit {
	// Compare to (*ast.Object).Pos().
	if id.Obj == nil {
		return nil
	}
	var rhs ast.Expr
	switch d := id.Obj.Decl.(type) {
	case *ast.AssignStmt:
		for i, x := range d.Lhs {
			if ident, isIdent := x.(*ast.Ident); isIdent && ident.Name == id.Name && i < len(d.Rhs) {
				rhs = d.Rhs[i]
			}
		}
	case *ast.ValueSpec:
		for i, n := range d.Names {
			if n.Name == id.Name && i < len(d.Values) {
				rhs = d.Values[i]
			}
		}
	}
	lit, _ := rhs.(*ast.FuncLit)
	return lit
}
