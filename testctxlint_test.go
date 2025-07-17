package testctxlint_test

import (
	"testing"

	"github.com/icedream/testctxlint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

func TestTestctxlint(t *testing.T) {
	analyzers := []*analysis.Analyzer{testctxlint.Analyzer}

	assert.NoError(t, analysis.Validate(analyzers))

	conf := packages.Config{
		Mode:  packages.LoadSyntax | packages.NeedModule,
		Tests: true,
	}
	initial, err := packages.Load(&conf, "./fixtures/unfixed/")
	require.NoError(t, err)

	opts := &checker.Options{}
	graph, err := checker.Analyze(analyzers, initial, opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, graph)

	t.Log(graph)

	// TODO - implement checks for exact errors/fixes provided
	for _, act := range graph.Roots {
		for _, diag := range act.Diagnostics {
			for i, fix := range diag.SuggestedFixes {
				assert.Equal(t, 0, i)
				t.Log("suggested fix", fix)
				assert.NotEmpty(t, fix.Message)
				assert.NotEmpty(t, fix.TextEdits)
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

	opts := &checker.Options{}

	b.ResetTimer()
	for b.Loop() {
		graph, err := checker.Analyze(analyzers, initial, opts)
		b.StopTimer()

		assert.NoError(b, err)
		assert.NotEmpty(b, graph)
		b.StartTimer()
	}
}
