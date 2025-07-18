# testctxlint

[![Go Reference](https://pkg.go.dev/badge/github.com/icedream/testctxlint.svg)](https://pkg.go.dev/github.com/icedream/testctxlint)

A Go linter that detects usage of `context.Background()` and `context.TODO()` in test functions and suggests using `t.Context()` or `b.Context()` instead. The linter detects problematic context usage in test functions, their subtests, and goroutines launched from tests.

Built using Go's analysis framework, testctxlint integrates seamlessly with existing Go tooling and provides automatic fixes for detected issues.

## Why use test contexts?

Go 1.24 introduced `t.Context()` and `b.Context()` methods for `*testing.T` and `*testing.B` respectively ([see Go 1.24 changelog](https://tip.golang.org/doc/go1.24#testingpkgtesting)). These provide contexts that are automatically cancelled when the test finishes, making them more appropriate for test scenarios than `context.Background()` or `context.TODO()`.

## Installation

### CLI Tool

```bash
go install github.com/icedream/testctxlint/cmd/testctxlint@latest
```

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

### With golangci-lint

Currently, testctxlint is not included in golangci-lint by default. You can run it separately or create a custom linter configuration.

### With IDEs

Most Go IDEs that support the Go analysis framework should be able to use testctxlint. The exact integration method depends on your IDE.

### In CI/CD

```bash
# Install and run in CI
go install github.com/icedream/testctxlint/cmd/testctxlint@latest
testctxlint ./...
```

## Sample Output

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

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

See [LICENSE.txt](LICENSE.txt) for details on this software's license.

This software incorporates material from third parties. For these portions,
see [NOTICE.txt](NOTICE.txt) for details.
