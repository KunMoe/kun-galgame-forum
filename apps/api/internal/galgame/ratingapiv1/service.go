package ratingapiv1

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

var errUnconfigured = errors.New("galgame rating faces are not configured")

type Users interface {
	Users(ctx context.Context, ids []int) (map[int]userclient.User, error)
}

type AwardFunc func(userID, delta int, reason, ref, idempotencyKey string)

// NotifyFunc writes the "liked" message to the rating's author.
type NotifyFunc func(tx *gorm.DB, senderID, receiverID int, preview string, workID int) error

type SyncFunc func(ctx context.Context, workID int, accessToken, playStatus string)

type Deps struct {
	Store  *repository.RatingStore
	Rows   workrepr.Rows
	Works  *workrepr.Hydrator
	Users  Users
	Check  *gate.CheckService
	Scan   *gate.ScanService
	Award  AwardFunc
	Notify NotifyFunc
	Sync   SyncFunc
	CDN    string
}

type Service struct {
	Deps
}

func New(d Deps) *Service {
	return &Service{Deps: d}
}

func (s *Service) ready() bool {
	return s != nil && s.Store != nil && s.Rows != nil && s.Works != nil && s.Users != nil
}

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func (s *Service) requireActive(ctx context.Context, viewer *middleware.UserInfo) *problem.Problem {
	users, err := s.Users.Users(ctx, []int{viewer.ID})
	if err != nil {
		return problem.Unavailable(err)
	}
	if u, ok := users[viewer.ID]; ok && !userclient.IsRenderable(u) {
		return problem.New(problem.CodeAccountBanned, "The signed-in user's account is banned.")
	}
	return nil
}

// visibleWorks returns the catalog rows of the works catalog still shows; a work
// whose claim was hidden is absent, and its ratings with it.
func (s *Service) visibleWorks(ctx context.Context, ids []int) (map[int]client.CatalogWorkListItem, *problem.Problem) {
	rows, appErr := s.Rows.CatalogRowsByWorkIDs(ctx, ids, workrepr.RowInclude, "all")
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	return rows, nil
}

func scorePtr(v int) *int {
	if v < 1 || v > 10 {
		return nil
	}
	return &v
}

func scoresOf(r repository.RatingRecord) AspectScores {
	return AspectScores{
		Art: scorePtr(r.Art), Story: scorePtr(r.Story), Music: scorePtr(r.Music),
		Character: scorePtr(r.Character), Route: scorePtr(r.Route), System: scorePtr(r.System),
		Voice: scorePtr(r.Voice), ReplayValue: scorePtr(r.ReplayValue),
	}
}

func gameTypesOf(raw json.RawMessage) []GameType {
	var keys []string
	_ = json.Unmarshal(raw, &keys)
	out := make([]GameType, 0, len(keys))
	for _, k := range keys {
		if slices.Contains(workrepr.GameTypes, k) {
			out = append(out, GameType(k))
		}
	}
	return out
}

func canDelete(viewer *middleware.UserInfo, r repository.RatingRecord, workCreator int) bool {
	return viewer.ID == r.UserID || (workCreator > 0 && viewer.ID == workCreator) || viewer.Can(perm.RatingDeleteAny)
}

// authorRef hides a banned author's rating and keeps a missing account's as a
// deleted-user reference.
func (s *Service) authorRef(users map[int]userclient.User, id int) (repr.UserRef, bool) {
	u, ok := users[id]
	if !ok {
		return repr.DeletedUserRef(id), true
	}
	if !userclient.IsRenderable(u) {
		return repr.UserRef{}, false
	}
	return repr.NewUserRef(s.CDN, u), true
}

func (s *Service) summary(ctx context.Context, r repository.RatingRecord, work *client.CatalogWorkListItem, author repr.UserRef) RatingSummary {
	out := RatingSummary{
		Object:       "rating",
		ID:           repr.ID(r.ID),
		Author:       author,
		Recommend:    Recommend(r.Recommend),
		Overall:      r.Overall,
		GameTypes:    gameTypesOf(r.GalgameType),
		PlayStatus:   PlayStatus(r.PlayStatus),
		SpoilerLevel: SpoilerLevel(r.SpoilerLevel),
		ShortSummary: r.ShortSummary,
		AspectScores: scoresOf(r),
		ViewCount:    max(r.View, 0),
		LikeCount:    max(r.LikeCount, 0),
		CommentCount: max(r.CommentCount, 0),
		CreatedAt:    repr.Timestamp(r.Created),
		UpdatedAt:    repr.Timestamp(r.Updated),
	}
	if work != nil {
		ref := workrepr.Ref(ctx, work, s.CDN)
		out.Work = &ref
	}
	return out
}

func (s *Service) viewerOf(ctx context.Context, r repository.RatingRecord, workCreator int, liked bool) *RatingViewer {
	viewer := v1.User(ctx)
	if viewer == nil {
		return nil
	}
	return &RatingViewer{
		HasLiked:  liked,
		CanEdit:   viewer.ID == r.UserID,
		CanDelete: canDelete(viewer, r, workCreator),
	}
}
