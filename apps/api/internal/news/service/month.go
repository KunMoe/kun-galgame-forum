package service

import (
	"context"
	"strconv"
	"time"

	"kun-galgame-api/pkg/newsclient"
)

const (
	monthTTL = 10 * time.Minute
	// The news face paginates by opaque cursor and has no offset, so a page
	// number cannot be turned into a request. A month is small enough to fetch
	// whole instead, which also makes the day breakdown exact rather than
	// probed. The cap is a runaway guard at 600 items — the busiest month on
	// record holds fewer than 200.
	monthPageCap = 12
)

// MonthService holds one month of items at a time, which is what both the
// paginator and the day breakdown are cut from.
type MonthService struct {
	news  *newsclient.Client
	items *ttlCache[[]newsclient.Item]
}

func NewMonthService(news *newsclient.Client) *MonthService {
	return &MonthService{news: news, items: newTTLCache[[]newsclient.Item](monthTTL)}
}

func (s *MonthService) Items(ctx context.Context, f ArchiveFilter, year, month int) ([]newsclient.Item, error) {
	key := f.key() + "|" + strconv.Itoa(year) + "-" + strconv.Itoa(month)
	if cached, ok := s.items.get(key); ok {
		return cached, nil
	}

	q := f.query()
	q.PublishedAfter, q.PublishedBefore = Window(year, month)
	q.Limit = newsclient.MaxLimit

	out := []newsclient.Item{}
	for range monthPageCap {
		page, err := s.news.Feed(ctx, q)
		if err != nil {
			return nil, err
		}
		out = append(out, page.Items...)
		if page.NextCursor == "" {
			break
		}
		q.Cursor = page.NextCursor
	}

	s.items.set(key, out)
	return out, nil
}

type DayCount struct {
	Day   int
	Count int
}

// DayCounts is one entry per calendar day of the month, zeros included: the day
// strip has to show which days are empty, not omit them.
func DayCounts(items []newsclient.Item, year, month int) []DayCount {
	last := monthStart(year, month+1).AddDate(0, 0, -1).Day()
	out := make([]DayCount, last)
	for i := range out {
		out[i].Day = i + 1
	}
	for _, it := range items {
		if d := it.PublishedAt.In(zone).Day(); d <= last {
			out[d-1].Count++
		}
	}
	return out
}

func OnDay(items []newsclient.Item, day int) []newsclient.Item {
	out := make([]newsclient.Item, 0, len(items))
	for _, it := range items {
		if it.PublishedAt.In(zone).Day() == day {
			out = append(out, it)
		}
	}
	return out
}
