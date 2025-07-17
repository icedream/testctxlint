package example_test

import (
	"context"
	"testing"
)

func example(ctx context.Context) {
	go func() {
		<-ctx.Done()
	}()
}

func TestLorem(t *testing.T) {
	ctx := context.Background()
	example(ctx) // should error, fix to t.Context()

	example(context.Background()) // should error, fix to t.Context()

	t.Run("test", func(t2 *testing.T) {
		example(ctx) // should error, fix to t2.Context()

		example(context.Background()) // should error, fix to t2.Context()

		go func() {
			example(context.Background()) // should error, fix to t2.Context()
		}()
	})

	go func() {
		example(context.Background()) // should error, fix to t.Context()
	}()
}

func TestIpsum(t *testing.T) {
	ctx := context.TODO()
	example(ctx) // should error, fix to t.Context()

	t.Run("test", func(t2 *testing.T) {
		example(ctx) // should error, fix to t2.Context()

		example(context.TODO()) // should error, fix to t2.Context()

		go func() {
			example(context.TODO()) // should error, fix to t2.Context()
		}()
	})

	go func() {
		example(context.TODO()) // should error, fix to t.Context()
	}()
}

func TestDolem(t *testing.T) {
	example(t.Context())

	t.Run("test", func(t *testing.T) {
		example(t.Context())
	})
}

func testSub(t *testing.T) {
	example(context.TODO()) // should error, fix to t.Context()
}

func TestMalem(t *testing.T) {
	sub := func(t2 *testing.T) {
		example(context.TODO()) // should error, fix to t2.Context()
	}
	t.Run("testSub", testSub)
	t.Run("sub", sub)
}

func BenchmarkIpsum(b *testing.B) {
	ctx := context.TODO()
	example(ctx) // should error, fix to b.Context()

	b.Run("test", func(b2 *testing.B) {
		example(ctx) // should error, fix to b2.Context()

		example(context.Background()) // should error, fix to b2.Context()
	})
}

func BenchmarkDolem(b *testing.B) {
	example(b.Context())

	b.Run("test", func(b2 *testing.B) {
		example(b2.Context())
	})
}
