package retrier

import (
	"context"
	"testing"
)

func TestPopRetrier(t *testing.T) {
	t.Parallel()
	r := NewRetrier()
	ctx := context.Background()

	testDbFunc := func(ctx context.Context) error { return nil }

	dbp := r.UseRetrierPopDB(testDbFunc)
	dbp(ctx)
}

func TestSetParams(t *testing.T) {
	t.Parallel()
	r := NewRetrier()
	r.SetParams(1, 1, 1)
}
