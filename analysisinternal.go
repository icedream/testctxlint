package testctxlint

import (
	"go/types"
	"slices"
)

// imports returns true if path is imported by pkg.
//
// Copied from golang.org/x/tools/internal/analysisinternal.
func imports(pkg *types.Package, importName string) bool {
	for _, imp := range pkg.Imports() {
		if imp.Name() == importName {
			return true
		}
	}
	return false
}

// isFunctionNamed reports whether obj is a package-level function
// defined in the given package and has one of the given names.
// It returns false if obj is nil.
//
// This function avoids allocating the concatenation of "pkg.Name",
// which is important for the performance of syntax matching.
//
// Copied from golang.org/x/tools/internal/analysisinternal.
func isFunctionNamed(obj types.Object, pkgPath string, names ...string) bool {
	f, ok := obj.(*types.Func)
	return ok &&
		isPackageLevel(obj) &&
		f.Pkg().Path() == pkgPath &&
		f.Type().(*types.Signature).Recv() == nil &&
		slices.Contains(names, f.Name())
}
