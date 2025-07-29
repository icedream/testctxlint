<p align="center">
  <img src="assets/logo-icon.svg" alt="testctxlint logo" width="80">
  <br>
  <strong>testctxlint</strong>
</p>

[![Go Reference](https://pkg.go.dev/badge/github.com/icedream/testctxlint.svg)](https://pkg.go.dev/github.com/icedream/testctxlint)

A Go linter that detects usage of `context.Background()` and `context.TODO()` in test functions and suggests using `t.Context()` or `b.Context()` instead. The linter detects problematic context usage in test functions, their subtests, and goroutines launched from tests.

Built using Go's analysis framework, testctxlint integrates seamlessly with existing Go tooling and provides automatic fixes for detected issues.

## Why use test contexts?

Go 1.24 introduced `t.Context()` and `b.Context()` methods for `*testing.T` and `*testing.B` respectively ([see Go 1.24 changelog](https://tip.golang.org/doc/go1.24#testingpkgtesting)). These provide contexts that are automatically cancelled when the test finishes, making them more appropriate for test scenarios than `context.Background()` or `context.TODO()`.

## Alternatives

Before you use this linter, there also exists https://github.com/ldez/usetesting which is already part of golangci-lint. It implements similar functionality but is disabled by default.

If you already have `usetesting` enabled, make sure to enable its `context-background` and `context-todo` flags.

See https://github.com/icedream/testctxlint/issues/17#issuecomment-3101551112 for all the differences if you're interested.

## Installation

### CLI Tool

#### Using go install
```bash
go install github.com/icedream/testctxlint/cmd/testctxlint@latest
```

#### Using pre-built binaries
Download the appropriate binary for your platform from the [releases page](https://github.com/icedream/testctxlint/releases).

## Requirements

- Go 1.24 or later (required for `t.Context()` and `b.Context()` methods)

## Usage

### Command Line

Basic usage:
```bash
testctxlint ./...
```

Check a specific package:
```bash
testctxlint ./pkg/mypackage
```

Apply suggested fixes automatically:
```bash
testctxlint -fix ./...
```

Show fixes as a diff without applying them:
```bash
testctxlint -diff ./...
```

Get help:
```bash
testctxlint -help
```

#### Sample Output

When testctxlint finds issues, it provides clear messages and suggestions:

```
/path/to/file.go:15:9: call to context.Background from a test routine
/path/to/file.go:23:11: call to context.TODO from a test routine
```

With the `-fix` flag, it can automatically apply the suggested fixes:

```go
// Before
ctx := context.Background()

// After
ctx := t.Context()
```

### Programmatic Usage

```go
package main

import (
    "golang.org/x/tools/go/analysis"
    "golang.org/x/tools/go/analysis/singlechecker"
    "github.com/icedream/testctxlint"
)

func main() {
    // Use the analyzer directly
    singlechecker.Main(testctxlint.Analyzer)
}
```

Or integrate with other analyzers:
```go
package main

import (
    "golang.org/x/tools/go/analysis/multichecker"
    "github.com/icedream/testctxlint"
    // ... other analyzers
)

func main() {
    multichecker.Main(
        testctxlint.Analyzer,
        // ... other analyzers
    )
}
```

## Examples

### ❌ Bad: Using context.Background() or context.TODO()

```go
func TestSomething(t *testing.T) {
    // BAD: Using context.Background() in test
    ctx := context.Background()
    doSomething(ctx)
    
    // BAD: Using context.TODO() in test
    doSomethingElse(context.TODO())
    
    // BAD: In subroutines
    t.Run("subtest", func(t *testing.T) {
        ctx := context.Background() // This will be flagged
        doSomething(ctx)
    })
    
    // BAD: In goroutines
    go func() {
        ctx := context.TODO() // This will be flagged
        doSomething(ctx)
    }()
}

func BenchmarkSomething(b *testing.B) {
    // BAD: Using context.Background() in benchmark
    ctx := context.Background()
    for i := 0; i < b.N; i++ {
        doSomething(ctx)
    }
}
```

### ✅ Good: Using t.Context() or b.Context()

```go
func TestSomething(t *testing.T) {
    // GOOD: Using t.Context() in test
    ctx := t.Context()
    doSomething(ctx)
    
    // GOOD: Direct usage
    doSomethingElse(t.Context())
    
    // GOOD: In subroutines
    t.Run("subtest", func(t *testing.T) {
        ctx := t.Context() // Properly scoped test context
        doSomething(ctx)
    })
    
    // GOOD: In goroutines (though be careful with goroutines in tests)
    go func() {
        ctx := t.Context() // Uses the test context
        doSomething(ctx)
    }()
}

func BenchmarkSomething(b *testing.B) {
    // GOOD: Using b.Context() in benchmark
    ctx := b.Context()
    for i := 0; i < b.N; i++ {
        doSomething(ctx)
    }
}
```

## Integration

### With IDEs

Most Go IDEs that support the Go analysis framework should be able to use testctxlint. The exact integration method depends on your IDE.

### In CI/CD

#### Using go install
```bash
# Install and run in CI
go install github.com/icedream/testctxlint/cmd/testctxlint@latest
testctxlint ./...
```

#### Using pre-built binaries
```bash
# Download and extract pre-built binary
curl -L -o testctxlint.tar.gz https://github.com/icedream/testctxlint/releases/latest/download/testctxlint_linux_x86_64.tar.gz
tar -xzf testctxlint.tar.gz
./testctxlint ./...
```

## Benchmarking

The project includes performance benchmarks that can be used to monitor performance regressions:

### Running Benchmarks Locally

```bash
# Run benchmarks
go test -bench=. -benchmem

# Run benchmarks multiple times for statistical analysis
go test -bench=. -benchmem -count=5

# Compare benchmark results using benchstat
go install golang.org/x/perf/cmd/benchstat@latest
go test -bench=. -benchmem -count=5 | tee new.txt
# (make changes to code)
go test -bench=. -benchmem -count=5 | tee old.txt
benchstat old.txt new.txt
```

### Automated Benchmark Comparison

The repository includes a GitHub Actions workflow that automatically compares benchmark performance between the main branch and pull requests. When you open a PR:

1. Benchmarks are run on both the main branch and your PR branch
2. Performance differences are calculated using `benchstat`
3. Results are posted as a comment on the PR
4. Benchmark data is stored as artifacts for historical analysis

The benchmark comparison helps identify performance regressions and improvements, providing metrics for:
- **Execution time** (ns/op): How long operations take
- **Memory usage** (B/op): Bytes allocated per operation  
- **Allocations** (allocs/op): Number of memory allocations per operation

## Testing

You can run the usual `go test` commands to run the tests in this repository.

Additionally, we lint our code using [golangci-lint v2](https://github.com/golangci/golangci-lint).

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

See [LICENSE.txt](LICENSE.txt) for details on this software's license.

This software incorporates material from third parties. For these portions,
see [NOTICE.txt](NOTICE.txt) for details.
