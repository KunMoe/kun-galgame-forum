package apiv1

import (
	"context"
	"errors"
	"strings"

	"kun-galgame-api/internal/apiv1/repr"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/ranking/repository"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

var errUnconfigured = errors.New("ranking v1 is not configured")

type Users interface {
	Users(ctx context.Context, ids []int) (map[int]userclient.User, error)
}

type Works interface {
	CatalogRowsByWorkIDs(ctx context.Context, ids []int, include, contentLimit string) (map[int]client.CatalogWorkListItem, *legacyErrors.AppError)
}

type Service struct {
	repo  *repository.RankingRepository
	users Users
	works Works
	cdn   string
}

func New(repo *repository.RankingRepository, users Users, works Works, cdn string) *Service {
	return &Service{repo: repo, users: users, works: works, cdn: cdn}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.repo == nil || s.users == nil || s.works == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

type topicRankingInput struct {
	Sort        string `query:"sort" enum:"views_desc,replies_desc,comments_desc,likes_desc,upvotes_desc,favorites_desc" default:"views_desc" maxLength:"14" doc:"What the list ranks by, highest first; ties break on descending id. views: view count. replies, comments, likes, upvotes, favorites: those counts."`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"How many places. 1–100, default 50. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, NSFW topics are ranked too. Default false."`
}

type topicRankingOutput struct {
	Body repr.List[TopicRankingEntry]
}

type userRankingInput struct {
	Sort  string `query:"sort" enum:"moemoepoint_desc,topics_desc,replies_desc,comments_desc,resources_desc" default:"moemoepoint_desc" maxLength:"16" doc:"What the list ranks by, highest first; ties break on descending user id. moemoepoint: the balance this forum caches. topics, replies, comments: what the user has posted where anonymous visitors can read it. resources: the user's galgame resources that are not taken down."`
	Limit int    `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"How many places. 1–100, default 50. Values above 100 are rejected, not clamped."`
}

type userRankingOutput struct {
	Body repr.List[UserRankingEntry]
}

func sortKey(token string) string {
	return strings.TrimSuffix(token, "_desc")
}

func (s *Service) lookupUsers(ctx context.Context, rows []repository.RankedRow, owner func(repository.RankedRow) int) (map[int]userclient.User, *problem.Problem) {
	seen := make(map[int]bool, len(rows))
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		if id := owner(r); id > 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	users, err := s.users.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	return users, nil
}

func (s *Service) listTopicRanking(ctx context.Context, in *topicRankingInput) (*topicRankingOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	rows, err := s.repo.TopTopics(sortKey(in.Sort), in.IncludeNSFW, in.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	users, prob := s.lookupUsers(ctx, rows, func(r repository.RankedRow) int { return r.Owner })
	if prob != nil {
		return nil, prob
	}
	items := make([]TopicRankingEntry, 0, len(rows))
	for _, r := range rows {
		u, ok := users[r.Owner]
		if !ok || !userclient.IsRenderable(u) {
			continue
		}
		items = append(items, TopicRankingEntry{
			Object:      "topic_ranking_entry",
			Rank:        len(items) + 1,
			MetricValue: r.Value,
			Topic: RankedTopic{
				Object: "topic",
				ID:     repr.ID(r.ID),
				Title:  r.Title,
				Author: repr.NewUserRef(s.cdn, u),
			},
		})
	}
	return &topicRankingOutput{Body: repr.NewList(items, nil)}, nil
}

func (s *Service) listUserRanking(ctx context.Context, in *userRankingInput) (*userRankingOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	rows, err := s.repo.TopUsers(sortKey(in.Sort), in.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	users, prob := s.lookupUsers(ctx, rows, func(r repository.RankedRow) int { return r.ID })
	if prob != nil {
		return nil, prob
	}
	items := make([]UserRankingEntry, 0, len(rows))
	for _, r := range rows {
		u, ok := users[r.ID]
		if !ok || !userclient.IsRenderable(u) {
			continue
		}
		bio := u.Bio
		items = append(items, UserRankingEntry{
			Object:      "user_ranking_entry",
			Rank:        len(items) + 1,
			MetricValue: r.Value,
			Member:      repr.NewUserRef(s.cdn, u),
			Bio:         &bio,
		})
	}
	return &userRankingOutput{Body: repr.NewList(items, nil)}, nil
}

type workRankingInput struct {
	Sort                string `query:"sort" enum:"views_desc,likes_desc,favorites_desc,resources_desc,rating_desc" default:"views_desc" maxLength:"14" doc:"What the list ranks by, highest first; ties break on descending work id. views, likes, favorites, resources: those counts on this forum. rating: this forum's ratings, weighted toward the site-wide mean for works with few of them."`
	Limit               int    `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"How many places. 1–100, default 50. Values above 100 are rejected, not clamped."`
	IncludeNSFW         bool   `query:"include_nsfw" default:"false" doc:"When true, works this forum displays as adult content are ranked too. Default false."`
	IncludeResourceless bool   `query:"include_resourceless" default:"false" doc:"When true, published works without any resource are ranked too. Default false."`
}

type workRankingOutput struct {
	Body repr.List[WorkRankingEntry]
}

func (s *Service) listWorkRanking(ctx context.Context, in *workRankingInput) (*workRankingOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	rows, err := s.repo.TopWorks(sortKey(in.Sort), in.IncludeNSFW, in.IncludeResourceless, in.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	contentLimit := "sfw"
	if in.IncludeNSFW {
		contentLimit = "all"
	}
	catalog, appErr := s.works.CatalogRowsByWorkIDs(ctx, ids, "names,covers", contentLimit)
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	users, prob := s.lookupUsers(ctx, rows, func(r repository.RankedRow) int { return r.Owner })
	if prob != nil {
		return nil, prob
	}
	items := make([]WorkRankingEntry, 0, len(rows))
	for _, r := range rows {
		it, ok := catalog[r.ID]
		if !ok {
			continue
		}
		var creator *repr.UserRef
		if u, ok := users[r.Owner]; ok && userclient.IsRenderable(u) {
			ref := repr.NewUserRef(s.cdn, u)
			creator = &ref
		}
		work := galgameapiv1.WorkRefOf(ctx, &it, s.cdn)
		items = append(items, WorkRankingEntry{
			Object:      "work_ranking_entry",
			Rank:        len(items) + 1,
			MetricValue: r.Value,
			Work:        &work,
			Creator:     creator,
		})
	}
	return &workRankingOutput{Body: repr.NewList(items, nil)}, nil
}
