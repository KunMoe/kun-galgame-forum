package followees

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"kun-galgame-api/pkg/communityclient"

	"golang.org/x/sync/singleflight"
)

const (
	ttl        = time.Minute
	pageSize   = 100
	maxPages   = 50
	maxEntries = 10000
)

var ErrUnavailable = errors.New("followees: the community follow graph is not configured")

type Source interface {
	ListFollowing(ctx context.Context, userID int64, cursor string, limit int) (*communityclient.FollowListResponse, error)
}

type entry struct {
	ids     []int
	expires time.Time
}

type Cache struct {
	src    Source
	mu     sync.Mutex
	byUser map[int]entry
	gen    map[int]uint64
	flight singleflight.Group
}

func New(src Source) *Cache {
	return &Cache{src: src, byUser: map[int]entry{}, gen: map[int]uint64{}}
}

func NewFromClient(c *communityclient.Client) *Cache {
	if c == nil || !c.Configured() {
		return New(nil)
	}
	return New(c)
}

func (c *Cache) IDs(ctx context.Context, userID int) ([]int, error) {
	if c == nil || c.src == nil {
		return nil, ErrUnavailable
	}
	now := time.Now()
	c.mu.Lock()
	e, ok := c.byUser[userID]
	c.mu.Unlock()
	if ok && now.Before(e.expires) {
		return e.ids, nil
	}
	v, err, _ := c.flight.Do(strconv.Itoa(userID), func() (any, error) {
		c.mu.Lock()
		gen := c.gen[userID]
		c.mu.Unlock()
		ids, err := c.fetch(context.WithoutCancel(ctx), userID)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		if c.gen[userID] == gen {
			if len(c.byUser) >= maxEntries {
				clear(c.byUser)
				clear(c.gen)
			}
			c.byUser[userID] = entry{ids: ids, expires: time.Now().Add(ttl)}
		}
		c.mu.Unlock()
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]int), nil
}

func (c *Cache) Forget(userID int) {
	if c == nil {
		return
	}
	c.mu.Lock()
	delete(c.byUser, userID)
	c.gen[userID]++
	c.mu.Unlock()
	c.flight.Forget(strconv.Itoa(userID))
}

func (c *Cache) fetch(ctx context.Context, userID int) ([]int, error) {
	ids := []int{}
	cursor := ""
	for range maxPages {
		page, err := c.src.ListFollowing(ctx, int64(userID), cursor, pageSize)
		if err != nil {
			return nil, err
		}
		for _, u := range page.Users {
			ids = append(ids, int(u.UserID))
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	return ids, nil
}
