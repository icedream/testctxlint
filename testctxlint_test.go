package testctxlint_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/icedream/testctxlint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

var rxFixHint = regexp.MustCompile(`\s+//\s+fix:\s+(.+)\s*$`)

func TestTestctxlint_Run(t *testing.T) {
	// Most of this code is just setting up the analyzer the same way the main
	// package does behind the scenes, with some irrelevant elements skipped

	analyzers := []*analysis.Analyzer{testctxlint.Analyzer}

	assert.NoError(t, analysis.Validate(analyzers))

	conf := packages.Config{
		Mode:  packages.LoadSyntax | packages.NeedModule,
		Tests: true,
	}

	pkgs, err := packages.Load(&conf, "./fixtures/unfixed/")
	require.NoError(t, err)
	require.NotEmpty(t, pkgs)
	require.False(t, pkgs[0].IllTyped)

	for _, pkg := range pkgs {
		diagnostics := []analysis.Diagnostic{}
		inputs := make(map[*analysis.Analyzer]any)

		module := &analysis.Module{}
		if mod := pkg.Module; mod != nil {
			module.Path = mod.Path
			module.Version = mod.Version
			module.GoVersion = mod.GoVersion
		}

		loadedFiles := map[string][]byte{}
		readFile := func(n string) ([]byte, error) {
			n = filepath.Clean(n)
			if fixedFile, ok := loadedFiles[n]; ok {
				return fixedFile, nil
			}
			data, err := os.ReadFile(n)
			if err == nil {
				loadedFiles[n] = data
			}
			return data, err
		}
		emulateFixedFile := func(n string, data []byte) {
			n = filepath.Clean(n)
			loadedFiles[n] = data
		}

		pass := &analysis.Pass{
			Analyzer:     testctxlint.Analyzer,
			Fset:         pkg.Fset,
			Files:        pkg.Syntax,
			OtherFiles:   pkg.OtherFiles,
			IgnoredFiles: pkg.IgnoredFiles,
			Pkg:          pkg.Types,
			TypesInfo:    pkg.TypesInfo,
			TypesSizes:   pkg.TypesSizes,
			TypeErrors:   pkg.TypeErrors,
			Module:       module,

			ResultOf: inputs,
			ReadFile: readFile,
			Report: func(d analysis.Diagnostic) {
				diagnostics = append(diagnostics, d)
			},
		}

		// simplified dependency loop without fact export/import
		for _, dep := range testctxlint.Analyzer.Requires {
			inputs[dep], err = dep.Run(pass)
			require.NoError(t, err)
		}

		result, err := testctxlint.Analyzer.Run(pass)
		assert.NoError(t, err)
		assert.IsType(t, testctxlint.Analyzer.ResultType, result)

		for _, diag := range diagnostics {
			posn := pkg.Fset.Position(diag.Pos)
			t.Logf("%s: %s\n", posn, diag.Message)

			end := pkg.Fset.Position(diag.End)
			if !end.IsValid() {
				end = posn
			}

			assert.Equal(t, posn.Line, end.Line, "fixes must not span more than a single line")

			data, _ := pass.ReadFile(posn.Filename)
			lines := strings.Split(string(data), "\n")
			for i := posn.Line; i <= end.Line; i++ {
				line := lines[i-1]

				// print code snippet with context
				for i := posn.Line - 1; i <= end.Line+1; i++ {
					if i >= 1 && i < len(lines) {
						t.Logf("%d\t%s\n", i, lines[i-1])
					}
				}

				// extract expected fix hint
				fixHintMatch := rxFixHint.FindStringSubmatch(line)
				if assert.NotNil(t, fixHintMatch, "linter must not trigger on correct lines") {
					expectedFixedLine := fixHintMatch[1]

					// apply all the fixes
					for i, fix := range diag.SuggestedFixes {
						assert.Equal(t, 0, i)
						assert.NotEmpty(t, fix.Message)
						assert.NotEmpty(t, fix.TextEdits)
						for _, edit := range fix.TextEdits {
							fixPosn := pkg.Fset.Position(edit.Pos)
							fixEndn := pkg.Fset.Position(edit.End)
							if !fixEndn.IsValid() {
								fixEndn = fixPosn
							}

							// apply suggested fix and match up with expected result
							line = line[0:fixPosn.Column-1] + string(edit.NewText) + line[fixEndn.Column-1:]
						}
					}

					// match up end result with fix hint
					lineWithoutHint := line[0 : len(line)-len(fixHintMatch[0])]
					assert.Equal(t, expectedFixedLine, strings.TrimSpace(lineWithoutHint))

					// remove the hint so we can test for leftover hints
					lines[i-1] = lineWithoutHint
				}
			}

			// check if any leftover hints exist (errors the linter did not catch)
			resultCode := strings.Join(lines, "\n")
			emulateFixedFile(posn.Filename, []byte(resultCode))
		}

		for name, data := range loadedFiles {
			lines := strings.Split(string(data), "\n")
			for lineIndex, line := range lines {
				assert.NotRegexp(t, rxFixHint, line, "linter did not catch bad line in %s:%d", name, lineIndex+1)
			}
		}
	}
}

func BenchmarkTestctxlint(b *testing.B) {
	analyzers := []*analysis.Analyzer{testctxlint.Analyzer}

	assert.NoError(b, analysis.Validate(analyzers))

	conf := packages.Config{
		Mode:  packages.LoadSyntax | packages.NeedModule,
		Tests: true,
	}

	initial, err := packages.Load(&conf, "./fixtures/unfixed/")
	require.NoError(b, err)
	require.NotEmpty(b, initial)
	require.False(b, initial[0].IllTyped)

	opts := &checker.Options{}

	b.ResetTimer()
	for b.Loop() {
		graph, err := checker.Analyze(analyzers, initial, opts)
		assert.NoError(b, err)
		assert.NotEmpty(b, graph)
	}
}
