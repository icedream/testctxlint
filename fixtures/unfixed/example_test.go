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
	ctx := context.Background() // context.Background() -> t.Context()
	example(ctx)

	example(context.Background()) // context.Background() -> t.Context()

	t.Run("test", func(t2 *testing.T) {
		example(ctx)

		example(context.Background()) // context.Background() -> t2.Context()

		go func() {
			example(context.Background()) // context.Background() -> t2.Context()
		}()
	})

	go func() {
		example(context.Background()) // context.Background() -> t.Context()
	}()
}

func TestIpsum(t *testing.T) {
	ctx := context.TODO() // context.TODO() -> t.Context()
	example(ctx)

	t.Run("test", func(t2 *testing.T) {
		example(ctx)

		example(context.TODO()) // context.TODO() -> t2.Context()

		go func() {
			example(context.TODO()) // context.TODO() -> t2.Context()
		}()
	})

	go func() {
		example(context.TODO()) // context.TODO() -> t.Context()
	}()
}

func TestDolem(t *testing.T) {
	example(t.Context())

	t.Run("test", func(t *testing.T) {
		example(t.Context())
	})
}

func testSub(t *testing.T) {
	example(context.TODO()) // context.TODO() -> t.Context()
}

func TestMalem(t *testing.T) {
	sub := func(t2 *testing.T) {
		example(context.TODO()) // context.TODO() -> t2.Context()
	}
	t.Run("testSub", testSub)
	t.Run("sub", sub)
}

func BenchmarkIpsum(b *testing.B) {
	ctx := context.TODO() // context.TODO() -> b.Context()
	example(ctx)

	b.Run("test", func(b2 *testing.B) {
		example(ctx)

		example(context.Background()) // context.Background() -> b2.Context()
	})
}

func BenchmarkDolem(b *testing.B) {
	example(b.Context())

	b.Run("test", func(b2 *testing.B) {
		example(b2.Context())
	})
}
