// Portions of this code are copied from golang.org/x/tools and marked as such.
// The following license applies to them:
//
// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the NOTICE.txt file.

package testctxlint

import "go/types"

// isPackageLevel reports whether obj is a package-level symbol.
//
// Copied from golang.org/x/tools/internal/typesinternal.
func isPackageLevel(obj types.Object) bool {
	return obj.Pkg() != nil && obj.Parent() == obj.Pkg().Scope()
}
