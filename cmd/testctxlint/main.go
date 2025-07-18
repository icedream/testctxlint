// Package main runs the analyzer. It's the CLI entrypoint.
package main

import (
	"fmt"
	"os"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/icedream/testctxlint"
)

// Build information set by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Handle version flag before singlechecker.Main processes it
	for _, arg := range os.Args {
		if arg == "-V" {
			fmt.Printf("testctxlint version %s, commit %s, built at %s\n", version, commit, date)
			os.Exit(0)
		}
	}

	singlechecker.Main(testctxlint.Analyzer)
}
