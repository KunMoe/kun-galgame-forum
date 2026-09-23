package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/news/service"
	"kun-galgame-api/pkg/newsclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

var errUnconfigured = errors.New("apiv1 news: service is not configured")

const feedSort = "published_desc"

type Service struct {
	news    *newsclient.Client
	users   *userclient.Client
	archive *service.ArchiveService
	month   *service.MonthService
	cdn     string
}

func New(news *newsclient.Client, users *userclient.Client, cdn string) *Service {
	if news == nil {
		return &Service{users: users, cdn: cdn}
	}
	return &Service{
		news:    news,
		users:   users,
		archive: service.NewArchiveService(news),
		month:   service.NewMonthService(news),
		cdn:     cdn,
	}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.news == nil {
		return problem.Unavailable(errUnconfigured)
	}
	return nil
}

func invalidCursor() *problem.Problem {
	return problem.New(problem.CodeInvalidCursor, "The cursor is malformed or was issued for different parameters.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "cursor does not decode for this collection", nil))
}

// upstreamProblem keeps every failure of the news face a 503 except the one a
// caller can fix: a cursor the face no longer accepts.
func upstreamProblem(err error, withCursor bool) *problem.Problem {
	if withCursor && errors.Is(err, newsclient.ErrBadRequest) {
		return invalidCursor()
	}
	return problem.Unavailable(err)
}

func (f NewsFilter) archiveFilter() service.ArchiveFilter {
	return service.ArchiveFilter{Lane: f.Lane, Source: f.NewsSource}
}

func newsItem(it newsclient.Item) NewsItem {
	return NewsItem{
		Object:      "news_item",
		ID:          repr.DecimalID(strconv.FormatInt(it.ID, 10)),
		NewsSource:  it.Source.Key,
		Lane:        it.Lane,
		Title:       it.Title,
		Preview:     it.Preview,
		SourceURL:   it.SourceURL,
		PublishedAt: repr.Timestamp(it.PublishedAt),
	}
}

func (s *Service) listNewsItems(ctx context.Context, in *listNewsItemsInput) (*listNewsItemsOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	if in.Month > 0 && in.Year == 0 {
		return nil, problem.New(problem.CodeInvalidParameter, "month narrows a year, so it needs year.",
			problem.AtParameter("month", problem.ReasonRequired, "send year together with month", nil))
	}
	fp := collect.Fingerprint(feedSort, in.Lane, in.NewsSource, strconv.Itoa(in.Year), strconv.Itoa(in.Month), strconv.Itoa(in.Limit))
	keys, curErr := collect.DecodeCursor(in.Cursor, feedSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	upstreamCursor := ""
	if keys != nil {
		if len(keys) != 1 || keys[0] == "" {
			return nil, invalidCursor()
		}
		upstreamCursor = keys[0]
	}
	after, before := service.Window(in.Year, in.Month)
	feed, err := s.news.Feed(ctx, newsclient.FeedQuery{
		Lane:            in.Lane,
		Source:          in.NewsSource,
		Cursor:          upstreamCursor,
		Limit:           in.Limit,
		PublishedAfter:  after,
		PublishedBefore: before,
	})
	if err != nil {
		return nil, upstreamProblem(err, upstreamCursor != "")
	}
	items := make([]NewsItem, 0, len(feed.Items))
	for _, it := range feed.Items {
		items = append(items, newsItem(it))
	}
	var next *string
	if feed.NextCursor != "" {
		cur := collect.EncodeCursor(feedSort, fp, feed.NextCursor)
		next = &cur
	}
	var total *int
	if in.IncludeTotal {
		n := int(feed.Count)
		total = &n
	}
	return &listNewsItemsOutput{Body: repr.NewCountedList(items, next, total)}, nil
}

func (s *Service) listNewsSources(ctx context.Context, _ *struct{}) (*listNewsSourcesOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	sources, err := s.news.Sources(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	var ids []int
	for _, src := range sources {
		if src.PublisherUID > 0 {
			ids = append(ids, int(src.PublisherUID))
		}
	}
	users := map[int]userclient.User{}
	if len(ids) > 0 && s.users != nil {
		found, err := s.users.Users(ctx, ids)
		if err != nil {
			slog.Warn("news sources: publisher lookup failed", "error", err)
		} else {
			users = found
		}
	}
	items := make([]NewsSource, 0, len(sources))
	for _, src := range sources {
		item := NewsSource{
			Object:      "news_source",
			Key:         src.Key,
			DisplayName: src.DisplayName,
			HomepageURL: src.HomepageURL,
			ColumnURL:   src.ColumnURL,
			Attribution: src.Attribution,
		}
		if u, ok := users[int(src.PublisherUID)]; ok && userclient.IsRenderable(u) {
			ref := repr.NewUserRef(s.cdn, u)
			item.ForumAccount = &ref
		}
		items = append(items, item)
	}
	return &listNewsSourcesOutput{Body: repr.NewList(items, nil)}, nil
}

func (s *Service) getNewsArchive(ctx context.Context, in *getNewsArchiveInput) (*getNewsArchiveOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	filter := in.archiveFilter()
	years, err := s.archive.Years(ctx, filter)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	out := NewsArchive{Object: "news_archive", Years: make([]YearCount, 0, len(years)), Months: []MonthCount{}}
	known := false
	for _, y := range years {
		out.Years = append(out.Years, YearCount{Year: y.Year, Count: y.Count})
		known = known || y.Year == in.Year
	}
	if known {
		months, err := s.archive.Months(ctx, filter, in.Year)
		if err != nil {
			return nil, problem.Unavailable(err)
		}
		for _, m := range months {
			if m.Count > 0 {
				out.Months = append(out.Months, MonthCount{Month: m.Month, Count: m.Count})
			}
		}
	}
	return &getNewsArchiveOutput{Body: out}, nil
}

func (s *Service) monthItems(ctx context.Context, path NewsMonthPath, f NewsFilter) ([]newsclient.Item, *problem.Problem) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	items, err := s.month.Items(ctx, f.archiveFilter(), path.Year, path.Month)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	return items, nil
}

func (s *Service) getNewsMonth(ctx context.Context, in *getNewsMonthInput) (*getNewsMonthOutput, error) {
	items, prob := s.monthItems(ctx, in.NewsMonthPath, in.NewsFilter)
	if prob != nil {
		return nil, prob
	}
	counts := service.DayCounts(items, in.Year, in.Month)
	days := make([]DayCount, 0, len(counts))
	for _, d := range counts {
		days = append(days, DayCount{Day: d.Day, Count: int64(d.Count)})
	}
	return &getNewsMonthOutput{Body: NewsMonth{
		Object:    "news_month",
		Year:      in.Year,
		Month:     in.Month,
		ItemCount: len(items),
		Days:      days,
	}}, nil
}

func (s *Service) listNewsMonthItems(ctx context.Context, in *listNewsMonthItemsInput) (*listNewsMonthItemsOutput, error) {
	if prob := in.CheckDepth(); prob != nil {
		return nil, prob
	}
	items, prob := s.monthItems(ctx, in.NewsMonthPath, in.NewsFilter)
	if prob != nil {
		return nil, prob
	}
	if in.Day > 0 {
		items = service.OnDay(items, in.Day)
	}
	start := min(in.Offset(), len(items))
	end := min(start+in.Limit, len(items))
	page := make([]NewsItem, 0, end-start)
	for _, it := range items[start:end] {
		page = append(page, newsItem(it))
	}
	total, relation := collect.ClampTotal(len(items))
	return &listNewsMonthItemsOutput{Body: repr.NewPageList(page, total, relation)}, nil
}
