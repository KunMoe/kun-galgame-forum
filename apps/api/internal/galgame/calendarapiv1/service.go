package calendarapiv1

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"sync"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

const (
	calendarPageLimit   = 100
	calendarPageCap     = 20
	upcomingMonthCap    = 24
	upcomingPageCap     = 5
	upcomingConcurrency = 8
	calendarMonthMax    = 7
)

var errUnconfigured = fmt.Errorf("release calendar faces are not configured")

type Catalog interface {
	workrepr.Rows
	CatalogCalendar(ctx context.Context, bucket string, q url.Values) (*client.CatalogWorksPage, *legacyErrors.AppError)
}

type Service struct {
	catalog Catalog
	works   *workrepr.Hydrator
	now     func() time.Time
}

func New(catalog Catalog, db *gorm.DB, cdn string) *Service {
	return &Service{
		catalog: catalog,
		works:   workrepr.NewHydrator(catalog, db, cdn),
		now:     time.Now,
	}
}

func (s *Service) WithClock(now func() time.Time) *Service {
	if now != nil {
		s.now = now
	}
	return s
}

var calendarLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return time.UTC
	}
	return loc
}()

func (s *Service) clock() time.Time {
	if s == nil || s.now == nil {
		return time.Now().In(calendarLocation)
	}
	return s.now().In(calendarLocation)
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.catalog == nil || s.works == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Service) listReleaseCalendarMonth(ctx context.Context, in *monthInput) (*monthOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	now := s.clock()
	month := in.Month
	if month == "" {
		month = now.Format("2006-01")
	}
	items, page, truncated, p := s.walk(ctx, "", monthQuery(month, in.IncludeNSFW), calendarPageCap, "release-calendar: month truncated at page cap", "calendar_month", month)
	if p != nil {
		return nil, p
	}
	summaries, p := s.summaries(ctx, items, "release-calendar: catalog row not renderable, dropped")
	if p != nil {
		return nil, p
	}
	today := orElse(page.Meta.Today, now.Format("2006-01-02"))
	calMonth := orElse(page.Month, month)
	return &monthOutput{Body: ReleaseCalendarMonth{
		Object:        "release_calendar_month",
		CalendarMonth: calMonth,
		Today:         today,
		Items:         summaries,
		PrevMonth:     monthPtr(shiftMonth(calMonth, -1)),
		NextMonth:     monthPtr(shiftMonth(calMonth, +1)),
		HasPrev:       derefBool(page.Meta.HasPrev),
		HasNext:       derefBool(page.Meta.HasNext),
		MinMonth:      monthPtr(page.Meta.MinMonth),
		MaxMonth:      monthPtr(page.Meta.MaxMonth),
		ItemCount:     len(summaries),
		IsTruncated:   truncated,
	}}, nil
}

func (s *Service) getReleaseCalendarToday(ctx context.Context, in *todayInput) (*todayOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	now := s.clock()
	month := now.Format("2006-01")
	items, page, truncated, p := s.walk(ctx, "", monthQuery(month, in.IncludeNSFW), calendarPageCap, "release-calendar: month truncated at page cap", "calendar_month", month)
	if p != nil {
		return nil, p
	}
	today := orElse(page.Meta.Today, now.Format("2006-01-02"))
	if truncated {
		slog.Warn("release-calendar: today computed on a truncated month", "calendar_month", month, "today", today)
	}
	has := false
	for i := range items {
		if !client.CatalogItemRenderable(&items[i]) {
			slog.Warn("release-calendar: catalog row not renderable, dropped", "work_id", items[i].ID)
			continue
		}
		if items[i].ReleaseDate != nil && *items[i].ReleaseDate == today {
			has = true
			break
		}
	}
	return &todayOutput{Body: ReleaseCalendarToday{
		Object:     "release_calendar_today",
		Today:      today,
		HasRelease: has,
		ExpiresIn:  secondsUntilMidnight(now),
	}}, nil
}

func (s *Service) listReleaseCalendarPending(ctx context.Context, in *pendingInput) (*pendingOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	now := s.clock()
	year := in.Year
	if year == 0 {
		year = now.Year()
	}
	q := bucketQuery(in.IncludeNSFW)
	q.Set("year", strconv.Itoa(year))
	q.Set("precision", "year")
	items, _, truncated, p := s.walk(ctx, "/pending", q, calendarPageCap, "release-calendar: pending truncated at page cap", "year", strconv.Itoa(year))
	if p != nil {
		return nil, p
	}
	summaries, p := s.summaries(ctx, items, "release-calendar: catalog row not renderable, dropped")
	if p != nil {
		return nil, p
	}
	return &pendingOutput{Body: ReleaseCalendarPending{
		Object:      "release_calendar_pending",
		Year:        year,
		Items:       summaries,
		ItemCount:   len(summaries),
		IsTruncated: truncated,
	}}, nil
}

func (s *Service) listReleaseCalendarTBA(ctx context.Context, in *tbaInput) (*tbaOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	q := bucketQuery(in.IncludeNSFW)
	q.Set("status", "unknown")
	items, _, truncated, p := s.walk(ctx, "/tba", q, calendarPageCap, "release-calendar: tba truncated at page cap", "status", "unknown")
	if p != nil {
		return nil, p
	}
	summaries, p := s.summaries(ctx, items, "release-calendar: catalog row not renderable, dropped")
	if p != nil {
		return nil, p
	}
	return &tbaOutput{Body: ReleaseCalendarTBA{
		Object:      "release_calendar_tba",
		Items:       summaries,
		ItemCount:   len(summaries),
		IsTruncated: truncated,
	}}, nil
}

func (s *Service) listReleaseCalendarUpcoming(ctx context.Context, in *upcomingInput) (*upcomingOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	now := s.clock()
	start := now.Format("2006-01")
	today := now.Format("2006-01-02")
	firstItems, first, truncated, p := s.walk(ctx, "", monthQuery(start, in.IncludeNSFW), upcomingPageCap, "release-calendar: upcoming month truncated at page cap", "calendar_month", start)
	if p != nil {
		return nil, p
	}
	if first.Meta.Today != "" {
		today = first.Meta.Today
	}
	last := orElse(first.Meta.MaxMonth, shiftMonth(start, upcomingMonthCap))
	months := monthRange(start, last, upcomingMonthCap)
	type monthRows struct {
		raw     []client.CatalogWorkListItem
		isTrunc bool
	}
	fetched := make([]monthRows, len(months))
	if len(fetched) > 0 {
		fetched[0] = monthRows{firstItems, truncated}
	}
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr *problem.Problem
	)
	sem := make(chan struct{}, upcomingConcurrency)
	for i := 1; i < len(months); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			raw, _, isTrunc, p := s.walk(ctx, "", monthQuery(months[i], in.IncludeNSFW), upcomingPageCap, "release-calendar: upcoming month truncated at page cap", "calendar_month", months[i])
			if p != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = p
				}
				mu.Unlock()
				return
			}
			fetched[i] = monthRows{raw, isTrunc}
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	entries := make([]ReleaseCalendarUpcomingEntry, 0, len(months))
	total := 0
	for i, m := range months {
		raw, isTrunc := fetched[i].raw, fetched[i].isTrunc
		kept := make([]client.CatalogWorkListItem, 0, len(raw))
		for j := range raw {
			it := &raw[j]
			if !client.CatalogItemRenderable(it) {
				slog.Warn("release-calendar: catalog row not renderable, dropped", "work_id", it.ID, "calendar_month", m)
				continue
			}
			if it.ReleaseDate != nil && *it.ReleaseDate >= start {
				kept = append(kept, *it)
			}
		}
		if len(kept) == 0 {
			continue
		}
		summaries, p := s.works.FromRows(ctx, kept)
		if p != nil {
			return nil, p
		}
		entries = append(entries, ReleaseCalendarUpcomingEntry{
			CalendarMonth: m,
			Items:         summaries,
			IsTruncated:   isTrunc,
		})
		total += len(summaries)
	}
	return &upcomingOutput{Body: ReleaseCalendarUpcoming{
		Object:    "release_calendar_upcoming",
		Today:     today,
		Entries:   entries,
		ItemCount: total,
	}}, nil
}

func (s *Service) walk(ctx context.Context, bucket string, q url.Values, pageCap int, warnMsg, warnKey, warnVal string) ([]client.CatalogWorkListItem, *client.CatalogWorksPage, bool, *problem.Problem) {
	var items []client.CatalogWorkListItem
	var last *client.CatalogWorksPage
	cursor := ""
	truncated := false
	for page := 0; page < pageCap; page++ {
		qq := cloneValues(q)
		if cursor != "" {
			qq.Set("cursor", cursor)
		}
		res, appErr := s.catalog.CatalogCalendar(ctx, bucket, qq)
		if appErr != nil {
			return nil, nil, false, problem.Unavailable(appErr)
		}
		last = res
		items = append(items, res.Items...)
		if res.NextCursor == "" {
			if last == nil {
				last = &client.CatalogWorksPage{}
			}
			return items, last, false, nil
		}
		cursor = res.NextCursor
		if page == pageCap-1 {
			truncated = true
		}
	}
	if truncated {
		slog.Warn(warnMsg, warnKey, warnVal, "pages", pageCap, "items", len(items))
	}
	return items, last, truncated, nil
}

func (s *Service) summaries(ctx context.Context, rows []client.CatalogWorkListItem, warn string) ([]workrepr.WorkSummary, *problem.Problem) {
	kept := make([]client.CatalogWorkListItem, 0, len(rows))
	for i := range rows {
		if !client.CatalogItemRenderable(&rows[i]) {
			slog.Warn(warn, "work_id", rows[i].ID)
			continue
		}
		kept = append(kept, rows[i])
	}
	return s.works.FromRows(ctx, kept)
}

func bucketQuery(includeNSFW bool) url.Values {
	q := url.Values{
		"limit":         {strconv.Itoa(calendarPageLimit)},
		"include":       {workrepr.RowInclude},
		"include_total": {"true"},
	}
	client.ApplyWorksGate(q, !includeNSFW)
	return q
}

func monthQuery(month string, includeNSFW bool) url.Values {
	q := bucketQuery(includeNSFW)
	if month != "" {
		q.Set("month", month)
	}
	return q
}

func cloneValues(q url.Values) url.Values {
	out := url.Values{}
	for k, vs := range q {
		out[k] = append([]string(nil), vs...)
	}
	return out
}

func orElse(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func derefBool(p *bool) bool {
	return p != nil && *p
}

func monthPtr(s string) *string {
	if s == "" || len(s) > calendarMonthMax {
		return nil
	}
	v := s
	return &v
}

func secondsUntilMidnight(now time.Time) int {
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	n := int(midnight.Sub(now).Seconds())
	if n < 0 {
		return 0
	}
	return n
}

func shiftMonth(ym string, n int) string {
	y, m := parseYM(ym)
	if y == 0 {
		return ""
	}
	total := y*12 + (m - 1) + n
	if total < 0 {
		return ""
	}
	return formatYM(total/12, total%12+1)
}

func monthRange(start, end string, capN int) []string {
	sy, sm := parseYM(start)
	ey, em := parseYM(end)
	if sy == 0 {
		return nil
	}
	if ey < sy || (ey == sy && em < sm) {
		return []string{formatYM(sy, sm)}
	}
	out := make([]string, 0, capN)
	y, m := sy, sm
	for len(out) < capN {
		out = append(out, formatYM(y, m))
		if y == ey && m == em {
			break
		}
		if m++; m > 12 {
			m = 1
			y++
		}
	}
	return out
}

func parseYM(s string) (int, int) {
	if len(s) < 7 {
		return 0, 1
	}
	y, _ := strconv.Atoi(s[:4])
	m, _ := strconv.Atoi(s[5:7])
	if m < 1 || m > 12 {
		m = 1
	}
	return y, m
}

func formatYM(y, m int) string {
	return fmt.Sprintf("%04d-%02d", y, m)
}
