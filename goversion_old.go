//go:build !go1.24
// +build !go1.24

package testctxlint

import "go/types"

func goVersion(pkg *types.Package) string {
	// types.Package.GoVersion did not exist before Go 1.21.
	if p, ok := any(pkg).(interface{ GoVersion() string }); ok {
		return p.GoVersion()
	}
	return ""
}
