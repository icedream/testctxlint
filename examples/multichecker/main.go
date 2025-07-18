package main

import (
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"github.com/icedream/testctxlint"
)

func main() {
	multichecker.Main(
		testctxlint.Analyzer,
		printf.Analyzer,
	)
}