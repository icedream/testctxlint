// Package main runs the analyzer. It's the CLI entrypoint.
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/icedream/testctxlint"
)

func main() {
	singlechecker.Main(testctxlint.Analyzer)
}
