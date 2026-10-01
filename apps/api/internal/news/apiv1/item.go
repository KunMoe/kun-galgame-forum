package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/newsclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func parseNewsItemID(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil && n > 0
}

func newsItem(it newsclient.Item, banner *repr.Image) NewsItem {
	return NewsItem{
		Object:      "news_item",
		ID:          repr.DecimalID(strconv.FormatInt(it.ID, 10)),
		NewsSource:  it.Source.Key,
		Lane:        it.Lane,
		Title:       it.Title,
		Preview:     it.Preview,
		Banner:      banner,
		SourceURL:   it.SourceURL,
		HasBody:     it.HasBody,
		PublishedAt: repr.Timestamp(it.PublishedAt),
	}
}

func (s *Service) newsItems(ctx context.Context, src []newsclient.Item) []NewsItem {
	metas := s.bannerMetas(src)
	items := make([]NewsItem, 0, len(src))
	for _, it := range src {
		items = append(items, newsItem(it, s.banner(it.BannerHash, metas)))
	}
	s.fillSubmitters(ctx, src, items)
	return items
}

func (s *Service) bannerMetas(src []newsclient.Item) map[string]imageclient.ImageMeta {
	if s.images == nil {
		return nil
	}
	var hashes []string
	for _, it := range src {
		if it.BannerHash != "" {
			hashes = append(hashes, it.BannerHash)
		}
	}
	if len(hashes) == 0 {
		return nil
	}
	return s.images(hashes)
}

func (s *Service) banner(hash string, metas map[string]imageclient.ImageMeta) *repr.Image {
	var meta *imageclient.ImageMeta
	if m, ok := metas[hash]; ok {
		meta = &m
	}
	return repr.NewImage(s.cdn, hash, meta)
}

func (s *Service) fillSubmitters(ctx context.Context, src []newsclient.Item, items []NewsItem) {
	ids := userclient.CollectIDs(src, func(it newsclient.Item) int { return int(it.SubmitterUID) })
	users := map[int]userclient.User{}
	if len(ids) > 0 && s.users != nil {
		found, err := s.users.Users(ctx, ids)
		if err != nil {
			slog.Warn("news items: submitter lookup failed", "error", err)
		} else {
			users = found
		}
	}
	for i := range items {
		u, ok := users[int(src[i].SubmitterUID)]
		if !ok || !userclient.IsRenderable(u) {
			continue
		}
		ref := repr.NewUserRef(s.cdn, u)
		items[i].Submitter = &ref
	}
}

func (s *Service) getNewsItem(ctx context.Context, in *getNewsItemInput) (*getNewsItemOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	if s.convert == nil {
		return nil, problem.Unavailable(errNoConverter)
	}
	id, ok := parseNewsItemID(in.NewsItemID)
	if !ok {
		return nil, notFound()
	}
	it, err := s.news.Item(ctx, id)
	if err != nil {
		if errors.Is(err, newsclient.ErrNotFound) {
			return nil, notFound()
		}
		return nil, problem.Unavailable(err)
	}
	doc, err := s.convert.ConvertUntrusted(ctx, it.Body)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items := s.newsItems(ctx, []newsclient.Item{*it})
	return &getNewsItemOutput{Body: NewsItemDetail{NewsItem: items[0], Content: doc}}, nil
}
