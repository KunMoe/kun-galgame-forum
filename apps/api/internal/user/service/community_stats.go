package service

import (
	"context"
	"sync"
	"time"
)

const communityStatsTTL = 7 * time.Minute

type visiblePostsCache struct {
	mu      sync.Mutex
	entries map[int]visiblePostsEntry
}

type visiblePostsEntry struct {
	count   int64
	expires time.Time
}

func newVisiblePostsCache() *visiblePostsCache {
	return &visiblePostsCache{entries: make(map[int]visiblePostsEntry)}
}

func (c *visiblePostsCache) get(userID int) (count int64, fresh bool, cached bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[userID]
	if !ok {
		return 0, false, false
	}
	return e.count, time.Now().Before(e.expires), true
}

func (c *visiblePostsCache) set(userID int, count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[userID] = visiblePostsEntry{count: count, expires: time.Now().Add(communityStatsTTL)}
}

func (s *UserService) communityVisiblePosts(ctx context.Context, userID int) (int64, bool) {
	if count, fresh, _ := s.commentCache.get(userID); fresh {
		return count, true
	}
	if s.community == nil || !s.community.Configured() {
		return s.staleOrUnknown(userID)
	}
	resp, err := s.community.AuthorStats(ctx, []int64{int64(userID)})
	if err != nil {
		return s.staleOrUnknown(userID)
	}
	var count int64
	for _, st := range resp.Stats {
		if st.AuthorID == int64(userID) {
			count = st.VisiblePosts
			break
		}
	}
	s.commentCache.set(userID, count)
	return count, true
}

func (s *UserService) staleOrUnknown(userID int) (int64, bool) {
	if stale, _, ok := s.commentCache.get(userID); ok {
		return stale, true
	}
	// Community errors used to become 0, which looked like the user had never posted.
	return 0, false
}
