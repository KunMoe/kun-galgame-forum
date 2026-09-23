package ratingapiv1

import (
	"context"
	"errors"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

const likerLimit = 50

type RatingSortToken string

var ratingSorts = []string{
	"created_desc", "created_asc", "view_desc", "view_asc", "overall_desc", "overall_asc",
}

func (RatingSortToken) Schema(huma.Registry) *huma.Schema {
	s := enumSchema(ratingSorts, "Sort order; ties break on id in the same direction. created: when the rating was written. view: page reads. overall: the overall score.")
	s.Default = "created_desc"
	return s
}

type listRatingsInput struct {
	Page         int             `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit        int             `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	Sort         RatingSortToken `query:"sort" default:"created_desc"`
	WorkID       string          `query:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Only ratings of this work. Omitted means every work."`
	AuthorID     string          `query:"author_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Only ratings by this user. Omitted means everyone."`
	SpoilerLevel SpoilerLevel    `query:"spoiler_level" doc:"Only ratings declaring this spoiler level. Omitted means any."`
	PlayStatus   PlayStatus      `query:"play_status" doc:"Only ratings with this play status. Omitted means any."`
	GameType     GameType        `query:"game_type" doc:"Only ratings filing the work under this game type. Omitted means any."`
	IncludeNSFW  bool            `query:"include_nsfw" default:"false" doc:"When true, ratings of adult works are included. Default false."`
}

type listRatingsOutput struct {
	Body repr.PageList[RatingSummary]
}

func (s *Service) listRatings(ctx context.Context, in *listRatingsInput) (*listRatingsOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	page := collect.PageNumber{Page: in.Page, Limit: in.Limit}
	if p := page.CheckDepth(); p != nil {
		return nil, p
	}
	token := string(in.Sort)
	if token == "" {
		token = "created_desc"
	}
	cut := strings.LastIndexByte(token, '_')
	column, dir := token[:cut], token[cut+1:]
	workID, _ := repr.ParseID(repr.DecimalID(in.WorkID))
	authorID, _ := repr.ParseID(repr.DecimalID(in.AuthorID))
	rows, count, err := s.Store.List(repository.RatingQuery{
		WorkID:       workID,
		AuthorID:     authorID,
		SpoilerLevel: string(in.SpoilerLevel),
		PlayStatus:   string(in.PlayStatus),
		GameType:     string(in.GameType),
		SFWOnly:      !in.IncludeNSFW,
		SortColumn:   column,
		Descending:   dir == "desc",
		Offset:       page.Offset(),
		Limit:        in.Limit,
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.summaries(ctx, rows)
	if p != nil {
		return nil, p
	}
	total, relation := collect.ClampTotal(count)
	return &listRatingsOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

// summaries drops the ratings of banned authors and of works catalog no longer
// shows. Both are only known upstream, so the page's total still counts them.
func (s *Service) summaries(ctx context.Context, rows []repository.RatingRecord) ([]RatingSummary, *problem.Problem) {
	if len(rows) == 0 {
		return []RatingSummary{}, nil
	}
	userIDs := make([]int, 0, len(rows))
	workIDs := make([]int, 0, len(rows))
	ratingIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		userIDs = append(userIDs, r.UserID)
		workIDs = append(workIDs, r.WorkID)
		ratingIDs = append(ratingIDs, r.ID)
	}
	users, err := s.Users.Users(ctx, userIDs)
	if err != nil {
		users = map[int]userclient.User{}
	}
	works, p := s.visibleWorks(ctx, workIDs)
	if p != nil {
		return nil, p
	}
	viewer := v1.User(ctx)
	liked := map[int]bool{}
	creators := map[int]int{}
	if viewer != nil {
		if liked, err = s.Store.LikedSet(viewer.ID, ratingIDs); err != nil {
			return nil, problem.Internal(err)
		}
		if creators, err = s.Store.WorkCreators(workIDs); err != nil {
			return nil, problem.Internal(err)
		}
	}
	out := make([]RatingSummary, 0, len(rows))
	for _, r := range rows {
		author, ok := s.authorRef(users, r.UserID)
		if !ok {
			continue
		}
		work, ok := works[r.WorkID]
		if !ok {
			continue
		}
		item := s.summary(ctx, r, &work, author)
		item.Viewer = s.viewerOf(ctx, r, creators[r.WorkID], liked[r.ID])
		out = append(out, item)
	}
	return out, nil
}

type ratingPathInput struct {
	RatingID string `path:"rating_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Rating id."`
}

type ratingOutput struct {
	Body Rating
}

func (s *Service) getRating(ctx context.Context, in *ratingPathInput) (*ratingOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	id, ok := repr.ParseID(repr.DecimalID(in.RatingID))
	if !ok {
		return nil, notFound()
	}
	rating, p := s.detail(ctx, id)
	if p != nil {
		return nil, p
	}
	if err := s.Store.IncrementView(id); err != nil {
		return nil, problem.Internal(err)
	}
	rating.ViewCount++
	return &ratingOutput{Body: *rating}, nil
}

// detail assembles the full rating. A banned author or a work catalog no longer
// shows is NOT_FOUND, the same answer as a rating that does not exist.
func (s *Service) detail(ctx context.Context, id int) (*Rating, *problem.Problem) {
	r, err := s.Store.Get(id)
	if errors.Is(err, repository.ErrRatingNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	likerIDs, err := s.Store.RecentLikers(id, likerLimit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	users, uerr := s.Users.Users(ctx, append([]int{r.UserID}, likerIDs...))
	if uerr != nil {
		users = map[int]userclient.User{}
	}
	author, ok := s.authorRef(users, r.UserID)
	if !ok {
		return nil, notFound()
	}
	works, p := s.visibleWorks(ctx, []int{r.WorkID})
	if p != nil {
		return nil, p
	}
	work, ok := works[r.WorkID]
	if !ok {
		return nil, notFound()
	}
	summaries, p := s.Works.FromRows(ctx, []client.CatalogWorkListItem{work})
	if p != nil {
		return nil, p
	}
	likers := make([]repr.UserRef, 0, len(likerIDs))
	for _, uid := range likerIDs {
		if ref, ok := s.authorRef(users, uid); ok {
			likers = append(likers, ref)
		}
	}
	sum := s.summary(ctx, r, &work, author)
	viewer := v1.User(ctx)
	if viewer != nil {
		liked, err := s.Store.LikedSet(viewer.ID, []int{id})
		if err != nil {
			return nil, problem.Internal(err)
		}
		creators, err := s.Store.WorkCreators([]int{r.WorkID})
		if err != nil {
			return nil, problem.Internal(err)
		}
		sum.Viewer = s.viewerOf(ctx, r, creators[r.WorkID], liked[id])
	}
	return &Rating{
		Object: sum.Object, ID: sum.ID, Work: sum.Work, Author: sum.Author,
		Recommend: sum.Recommend, Overall: sum.Overall, GameTypes: sum.GameTypes,
		PlayStatus: sum.PlayStatus, SpoilerLevel: sum.SpoilerLevel, ShortSummary: sum.ShortSummary,
		AspectScores: sum.AspectScores, ViewCount: sum.ViewCount, LikeCount: sum.LikeCount,
		CommentCount: sum.CommentCount, CreatedAt: sum.CreatedAt, UpdatedAt: sum.UpdatedAt,
		Viewer: sum.Viewer, WorkSummary: summaries[0], Likers: likers,
	}, nil
}
