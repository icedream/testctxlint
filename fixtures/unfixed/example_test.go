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
	ctx := context.Background() // fix: ctx := t.Context()
	example(ctx)

	example(context.Background()) // fix: example(t.Context())

	t.Run("test", func(t2 *testing.T) {
		example(ctx)

		example(context.Background()) // fix: example(t2.Context())

		go func() {
			example(context.Background()) // fix: example(t2.Context())
		}()
	})

	go func() {
		example(context.Background()) // fix: example(t.Context())
	}()

	go func(t2 *testing.T) {
		example(context.Background()) // fix: example(t2.Context())
	}(t)
}

func TestIpsum(t *testing.T) {
	ctx := context.TODO() // fix: ctx := t.Context()
	example(ctx)

	t.Run("test", func(t2 *testing.T) {
		example(ctx)

		example(context.TODO()) // fix: example(t2.Context())

		go func() {
			example(context.TODO()) // fix: example(t2.Context())
		}()
	})

	go func() {
		example(context.TODO()) // fix: example(t.Context())
	}()

	go func(t2 *testing.T) {
		example(context.TODO()) // fix: example(t2.Context())
	}(t)
}

func TestDolem(t *testing.T) {
	example(t.Context())

	t.Run("test", func(t *testing.T) {
		example(t.Context())
	})
}

func testSub(t *testing.T) {
	example(context.TODO()) // fix: example(t.Context())
}

func TestMalem(t *testing.T) {
	sub := func(t2 *testing.T) {
		example(context.TODO()) // fix: example(t2.Context())
	}
	t.Run("testSub", testSub)
	t.Run("sub", sub)
}

func BenchmarkIpsum(b *testing.B) {
	ctx := context.TODO() // fix: ctx := b.Context()
	example(ctx)

	b.Run("test", func(b2 *testing.B) {
		example(ctx)

		example(context.Background()) // fix: example(b2.Context())
	})
}

func BenchmarkDolem(b *testing.B) {
	example(b.Context())

	b.Run("test", func(b2 *testing.B) {
		example(b2.Context())
	})
}

func TestUnnamedParam(*testing.T) {
	ctx := context.Background() // fix: ctx := t.Context()
	example(ctx)

	example(context.TODO()) // fix: example(t.Context())
}

func BenchmarkUnnamedParam(*testing.B) {
	ctx := context.Background() // fix: ctx := b.Context()
	example(ctx)

	example(context.TODO()) // fix: example(b.Context())
}

func testSubUnnamed(*testing.T) {
	example(context.Background()) // fix: example(t.Context())
}
