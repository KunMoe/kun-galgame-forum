package entityapiv1

import (
	"context"
	"sync"
	"time"
)

// index holds a whole catalog vocabulary walk. A stale copy is served while one
// rebuild runs in the background; a failed rebuild keeps the previous rows and
// their timestamp, so the next reader retries instead of an outage being cached
// for a whole ttl.
type index[T any] struct {
	mu       sync.Mutex
	rows     []T
	ok       bool
	built    time.Time
	building chan struct{}
}

func (x *index[T]) get(ctx context.Context, ttl time.Duration, build func(context.Context) ([]T, error)) ([]T, error) {
	x.mu.Lock()
	rows, ok, fresh, building := x.rows, x.ok, time.Since(x.built) < ttl, x.building
	if ok && fresh {
		x.mu.Unlock()
		return rows, nil
	}
	if building != nil {
		x.mu.Unlock()
		if ok {
			return rows, nil
		}
		select {
		case <-building:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return x.get(ctx, ttl, build)
	}
	done := make(chan struct{})
	x.building = done
	x.mu.Unlock()

	if ok {
		go func() {
			built, err := build(context.WithoutCancel(ctx))
			x.finish(built, err, done)
		}()
		return rows, nil
	}
	built, err := build(ctx)
	x.finish(built, err, done)
	if err != nil {
		return nil, err
	}
	return built, nil
}

func (x *index[T]) finish(rows []T, err error, done chan struct{}) {
	x.mu.Lock()
	if err == nil {
		x.rows, x.ok, x.built = rows, true, time.Now()
	}
	x.building = nil
	x.mu.Unlock()
	close(done)
}
