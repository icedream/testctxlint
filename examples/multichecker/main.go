/*
Example application
*/
package main

import (
	"github.com/icedream/testctxlint"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
)

func main() {
	multichecker.Main(
		testctxlint.Analyzer,
		printf.Analyzer,
	)
}
