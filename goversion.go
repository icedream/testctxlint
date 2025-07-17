//go:build go1.24
// +build go1.24

package testctxlint

import "go/types"

func goVersion(pkg *types.Package) string {
	return pkg.GoVersion()
}
