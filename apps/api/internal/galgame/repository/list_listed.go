package repository

import (
	"log/slog"
	"sync"
	"time"
)

const listedWorkTTL = 10 * time.Minute

// listedWorks is one set for every series card. ListIDs once per series, with
// the count and the ORDER BY thrown away, ran 13.9 million queries between the
// 2026-10-11 stats reset and 2026-11-02.
type listedWorks struct {
	mu      sync.Mutex
	ids     map[int]struct{}
	at      time.Time
	retryAt time.Time
	ok      bool
	loading chan struct{}
}

// ListedWorkIDs is the works this forum lists: a published page, a row catalog
// will render, and at least one resource. Callers only read the map.
func (r *GalgameListRepository) ListedWorkIDs() map[int]struct{} {
	if r == nil || r.db == nil {
		return map[int]struct{}{}
	}
	r.listed.mu.Lock()
	if ids, ok := r.listed.fresh(); ok {
		r.listed.mu.Unlock()
		return ids
	}
	if r.listed.loading != nil {
		done := r.listed.loading
		r.listed.mu.Unlock()
		<-done
		return r.ListedWorkIDs()
	}
	done := make(chan struct{})
	r.listed.loading = done
	r.listed.mu.Unlock()

	ids, err := r.loadListedWorkIDs()

	r.listed.mu.Lock()
	if err != nil {
		slog.Warn("listed work set load failed", "error", err)
		r.listed.retryAt = time.Now().Add(30 * time.Second)
		if r.listed.ok {
			ids = r.listed.ids
		} else {
			ids = map[int]struct{}{}
		}
	} else {
		r.listed.ids = ids
		r.listed.at = time.Now()
		r.listed.ok = true
		r.listed.retryAt = time.Time{}
	}
	r.listed.loading = nil
	r.listed.mu.Unlock()
	close(done)
	return ids
}

func (s *listedWorks) fresh() (map[int]struct{}, bool) {
	if s.ok && time.Since(s.at) < listedWorkTTL {
		return s.ids, true
	}
	if !s.retryAt.IsZero() && time.Now().Before(s.retryAt) {
		if s.ok {
			return s.ids, true
		}
		return map[int]struct{}{}, true
	}
	return nil, false
}

func (r *GalgameListRepository) loadListedWorkIDs() (map[int]struct{}, error) {
	var rows []struct {
		ID int `gorm:"column:id"`
	}
	err := r.db.Raw(`
		SELECT g.id FROM galgame g
		WHERE g.published AND g.catalog_rendered
		  AND EXISTS (SELECT 1 FROM galgame_resource gr WHERE gr.work_id = g.id)
	`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	set := make(map[int]struct{}, len(rows))
	for _, row := range rows {
		set[row.ID] = struct{}{}
	}
	return set, nil
}
