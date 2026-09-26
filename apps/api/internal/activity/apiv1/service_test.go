package apiv1

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/client"
	legacyErrors "kun-galgame-api/pkg/errors"
)

type blockingCatalog struct {
	calls   atomic.Int32
	release chan struct{}
}

func (c *blockingCatalog) CatalogRowsByWorkIDs(_ context.Context, ids []int, _, _ string) (map[int]client.CatalogWorkListItem, *legacyErrors.AppError) {
	c.calls.Add(1)
	<-c.release
	out := make(map[int]client.CatalogWorkListItem, len(ids))
	for _, id := range ids {
		out[id] = client.CatalogWorkListItem{ID: int64(id)}
	}
	return out, nil
}

func TestCatalogRowsCoalescesConcurrentMisses(t *testing.T) {
	cat := &blockingCatalog{release: make(chan struct{})}
	s := New(nil, cat, nil, nil, nil, "")
	orders := [][]int{{3, 1, 2}, {1, 2, 3}, {2, 3, 1}, {3, 2, 1}, {1, 3, 2}, {2, 1, 3}, {3, 1, 2}, {1, 2, 3}}
	var wg sync.WaitGroup
	for _, ids := range orders {
		wg.Go(func() {
			rows, prob := s.catalogRows(context.Background(), ids, true)
			if prob != nil || len(rows) != 3 {
				t.Errorf("catalogRows(%v) = %d rows, %v", ids, len(rows), prob)
			}
		})
	}
	time.Sleep(200 * time.Millisecond)
	close(cat.release)
	wg.Wait()
	if n := cat.calls.Load(); n != 1 {
		t.Fatalf("%d concurrent misses on one page cost %d catalog calls, want 1", len(orders), n)
	}
}
