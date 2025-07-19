// Package main runs the analyzer. It's the CLI entrypoint.
package main

import (
	"fmt"
	"os"

	"github.com/icedream/testctxlint"
	"golang.org/x/tools/go/analysis/singlechecker"
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
			_, _ = fmt.Fprintf(os.Stdout, "testctxlint version %s, commit %s, built at %s\n", version, commit, date)
			os.Exit(0)
		}
	}

	singlechecker.Main(testctxlint.Analyzer)
}
