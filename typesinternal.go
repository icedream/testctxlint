package testctxlint

import "go/types"

// isPackageLevel reports whether obj is a package-level symbol.
//
// Copied from golang.org/x/tools/internal/typesinternal.
func isPackageLevel(obj types.Object) bool {
	return obj.Pkg() != nil && obj.Parent() == obj.Pkg().Scope()
}
