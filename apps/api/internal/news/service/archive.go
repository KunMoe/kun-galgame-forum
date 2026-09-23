package service

import (
	"context"
	"strconv"
	"sync"
	"time"

	"kun-galgame-api/pkg/newsclient"

	"golang.org/x/sync/errgroup"
)

// Both partners publish on China time, so the archive buckets on it as well. A
// UTC year boundary would file everything published before 08:00 on 1 January
// under the previous year, and the archive's counts would then disagree with
// the dates the feed itself renders.
var zone = time.FixedZone("CST", 8*60*60)

const (
	archiveTTL = time.Hour
	// How far back the year walk is allowed to reach before giving up. It is a
	// runaway guard, not a claim about the corpus: the walk normally stops on
	// its own the first time a boundary counts zero.
	yearSpan = 30
)

type YearCount struct {
	Year  int
	Count int64
}

type MonthCount struct {
	Month int
	Count int64
}

type ArchiveFilter struct {
	Lane   string
	Source string
}

func (f ArchiveFilter) key() string { return f.Lane + "|" + f.Source }

func (f ArchiveFilter) query() newsclient.FeedQuery {
	return newsclient.FeedQuery{Lane: f.Lane, Source: f.Source}
}

// ArchiveService derives the year/month index the news face does not publish.
// Every bucket costs one upstream count, so the whole thing is cached per
// filter; if the news face ever grows a group-by endpoint, this collapses into
// a single read.
type ArchiveService struct {
	news   *newsclient.Client
	years  *ttlCache[[]YearCount]
	months *ttlCache[[]MonthCount]
}

func NewArchiveService(news *newsclient.Client) *ArchiveService {
	return &ArchiveService{
		news:   news,
		years:  newTTLCache[[]YearCount](archiveTTL),
		months: newTTLCache[[]MonthCount](archiveTTL),
	}
}

// Years walks backwards from the newest item, one boundary per year. Counting
// how many items are older than 1 January of year N gives year N's own total by
// subtraction, so the walk pays a single request per year and terminates the
// moment nothing older is left.
func (s *ArchiveService) Years(ctx context.Context, f ArchiveFilter) ([]YearCount, error) {
	key := f.key()
	if cached, ok := s.years.get(key); ok {
		return cached, nil
	}

	q := f.query()
	q.Limit = 1
	head, err := s.news.Feed(ctx, q)
	if err != nil {
		return nil, err
	}

	out := []YearCount{}
	if len(head.Items) > 0 {
		remaining := head.Count
		newest := head.Items[0].PublishedAt.In(zone).Year()
		for year := newest; remaining > 0 && year > newest-yearSpan; year-- {
			older, err := s.countBefore(ctx, f, monthStart(year, 1))
			if err != nil {
				return nil, err
			}
			out = append(out, YearCount{Year: year, Count: remaining - older})
			remaining = older
		}
	}

	s.years.set(key, out)
	return out, nil
}

func (s *ArchiveService) Months(ctx context.Context, f ArchiveFilter, year int) ([]MonthCount, error) {
	key := f.key() + "|" + strconv.Itoa(year)
	if cached, ok := s.months.get(key); ok {
		return cached, nil
	}

	// cum[m] is how many items are older than the start of month m+1, so twelve
	// subtractions turn thirteen boundary counts into twelve buckets.
	cum := make([]int64, 13)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(4)
	for m := range cum {
		g.Go(func() error {
			n, err := s.countBefore(gctx, f, monthStart(year, m+1))
			cum[m] = n
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	out := make([]MonthCount, 0, 12)
	for m := 1; m <= 12; m++ {
		out = append(out, MonthCount{Month: m, Count: cum[m] - cum[m-1]})
	}

	s.months.set(key, out)
	return out, nil
}

func (s *ArchiveService) countBefore(ctx context.Context, f ArchiveFilter, boundary time.Time) (int64, error) {
	q := f.query()
	q.PublishedBefore = boundary.Add(-time.Microsecond)
	return s.news.Count(ctx, q)
}

// Window turns a year, optionally narrowed to one month, into the bounds the
// news face takes. A month without a year has nothing to anchor to and drops.
func Window(year, month int) (after, before time.Time) {
	if year <= 0 {
		return time.Time{}, time.Time{}
	}
	if month <= 0 {
		return monthStart(year, 1), monthStart(year+1, 1).Add(-time.Microsecond)
	}
	return monthStart(year, month), monthStart(year, month+1).Add(-time.Microsecond)
}

// month 13 is deliberate: time.Date normalises it to January of the next year,
// which is what the last bucket of every walk needs.
func monthStart(year, month int) time.Time {
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, zone)
}

// cacheCap bounds a map whose key includes a client-supplied source, so an
// unknown source cannot grow it without limit. Real traffic never approaches it
// — a handful of lanes times a handful of partners times the years they cover.
const cacheCap = 256

type ttlCache[T any] struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]ttlEntry[T]
}

type ttlEntry[T any] struct {
	value T
	built time.Time
}

func newTTLCache[T any](ttl time.Duration) *ttlCache[T] {
	return &ttlCache[T]{ttl: ttl, items: make(map[string]ttlEntry[T])}
}

func (c *ttlCache[T]) get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[key]
	if !ok || time.Since(entry.built) >= c.ttl {
		var zero T
		return zero, false
	}
	return entry.value, true
}

func (c *ttlCache[T]) set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= cacheCap {
		clear(c.items)
	}
	c.items[key] = ttlEntry[T]{value: value, built: time.Now()}
}
