package service

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strconv"

	"kun-galgame-api/internal/galgame/client"
	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"
)

type UserContentService struct {
	userContentRepo *repository.UserContentRepository
	galgameClient   *client.GalgameClient
	galgameStats    *galgameService.GalgameUserStatsService
	userClient      *userclient.Client
	community       *communityclient.Client
}

func NewUserContentService(
	userContentRepo *repository.UserContentRepository,
	galgameClient *client.GalgameClient,
	galgameStats *galgameService.GalgameUserStatsService,
	userClient *userclient.Client,
	community *communityclient.Client,
) *UserContentService {
	return &UserContentService{
		userContentRepo: userContentRepo,
		galgameClient:   galgameClient,
		galgameStats:    galgameStats,
		userClient:      userClient,
		community:       community,
	}
}

func (s *UserContentService) hideTarget(ctx context.Context, userID int) bool {
	u, _, _ := s.userClient.User(ctx, userID)
	return !userclient.IsRenderable(u)
}

func (s *UserContentService) GetUserGalgameCards(
	ctx context.Context,
	userID int,
	req *dto.UserGalgamesRequest,
	isSFW bool,
) ([]dto.UserGalgameCard, int64, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return []dto.UserGalgameCard{}, 0, nil
	}
	if req.Type == "galgame_publish" {
		ids, total, err := s.galgameStats.PublishedWorkIDs(ctx, int64(userID), req.Page, req.Limit)
		if err != nil || len(ids) == 0 {
			return []dto.UserGalgameCard{}, total, nil
		}
		return s.galgameCardsByIDs(ctx, ids, isSFW), total, nil
	}

	if req.Type == "galgame_contributed" {
		ids, err := s.galgameStats.ContributedWorkIDs(ctx, int64(userID))
		if err != nil {
			return []dto.UserGalgameCard{}, 0, nil
		}
		total := int64(len(ids))
		ids = pageSlice(ids, req.Page, req.Limit)
		if len(ids) == 0 {
			return []dto.UserGalgameCard{}, total, nil
		}
		return s.galgameCardsByIDs(ctx, ids, isSFW), total, nil
	}

	ids, total, err := s.userContentRepo.FindUserWorkIDs(userID, req.Type, req.Page, req.Limit, req.ShowNoResource)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取用户 Galgame 列表失败")
	}
	if len(ids) == 0 {
		return []dto.UserGalgameCard{}, total, nil
	}

	briefMap, galgameErr := s.galgameClient.GetBatchPublic(ctx, ids, isSFW)
	if galgameErr != nil {
		return []dto.UserGalgameCard{}, total, nil
	}

	briefs := make([]client.GalgameBrief, 0, len(ids))
	for _, id := range ids {
		if b, ok := briefMap[id]; ok {
			briefs = append(briefs, b)
		}
	}
	return s.buildGalgameCards(ctx, briefs), total, nil
}

func (s *UserContentService) galgameCardsByIDs(
	ctx context.Context,
	ids []int,
	isSFW bool,
) []dto.UserGalgameCard {
	briefMap, galgameErr := s.galgameClient.GetBatchPublic(ctx, ids, isSFW)
	if galgameErr != nil {
		return []dto.UserGalgameCard{}
	}
	briefs := make([]client.GalgameBrief, 0, len(ids))
	for _, id := range ids {
		if b, ok := briefMap[id]; ok {
			briefs = append(briefs, b)
		}
	}
	return s.buildGalgameCards(ctx, briefs)
}

func pageSlice(ids []int, page, limit int) []int {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	start := (page - 1) * limit
	if start >= len(ids) {
		return nil
	}
	return ids[start:min(start+limit, len(ids))]
}

func (s *UserContentService) buildGalgameCards(
	ctx context.Context,
	briefs []client.GalgameBrief,
) []dto.UserGalgameCard {
	if len(briefs) == 0 {
		return []dto.UserGalgameCard{}
	}

	ids := make([]int, len(briefs))
	for i, b := range briefs {
		ids[i] = b.ID
	}
	localMap := s.userContentRepo.FindGalgameLocalStats(ids)
	metaRows := s.userContentRepo.FindResourceMetaByWorkIDs(ids)
	platformMap, languageMap := groupResourceMeta(metaRows)

	userIDs := collectUniqueIDs(ids, func(id int) int { return userclient.DerefID(localMap[id].CreatorUserID) })
	userMap := s.userClient.Hydrate(ctx, userIDs)

	cards := make([]dto.UserGalgameCard, 0, len(briefs))
	for _, b := range briefs {
		l := localMap[b.ID]
		u := userMap[userclient.DerefID(l.CreatorUserID)]
		cards = append(cards, dto.UserGalgameCard{
			ID:                  b.ID,
			Name:                b.Name,
			NameOriginal:        b.NameOriginal,
			User:                dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			ContentLimit:        b.ContentLimit,
			View:                l.View,
			LikeCount:           l.LikeCount,
			ResourceUpdateTime:  utils.RFC3339OrEmpty(l.ResourceUpdateTime),
			Platform:            emptyStrSlice(platformMap[b.ID]),
			Language:            emptyStrSlice(languageMap[b.ID]),
			ReleaseDate:         b.ReleaseDate,
			ReleaseDateTBA:      b.ReleaseDateTBA,
			EffectiveBannerHash: b.EffectiveBannerHash,
			EffectiveBannerURL:  b.EffectiveBannerURL,

			EffectivePortraitHash:      b.EffectivePortraitHash,
			EffectivePortraitURL:       b.EffectivePortraitURL,
			EffectivePortraitWidth:     b.EffectivePortraitWidth,
			EffectivePortraitHeight:    b.EffectivePortraitHeight,
			EffectivePortraitThumbhash: b.EffectivePortraitThumbhash,

			Company: b.Company,
		})
	}
	return cards
}

func (s *UserContentService) GetUserGalgameComments(
	ctx context.Context,
	userID int,
	req *dto.UserGalgameCommentsRequest,
	_ bool,
) ([]dto.UserGalgameComment, string, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return []dto.UserGalgameComment{}, "", nil
	}
	if req.Type == "galgame_comment_like" {
		return s.likedGalgameComments(ctx, userID, req.After, req.Limit)
	}
	return s.authoredGalgameComments(ctx, userID, req.After, req.Limit)
}

func (s *UserContentService) authoredGalgameComments(ctx context.Context, userID int, after string, limit int) ([]dto.UserGalgameComment, string, *errors.AppError) {
	posts, err := s.collectAuthorGalgamePosts(ctx, userID)
	if err != nil {
		if communityDown(err) {
			return []dto.UserGalgameComment{}, "", nil
		}
		return nil, "", errors.ErrInternal("获取用户 Galgame 评论列表失败")
	}
	page, nextCursor := paginateGalgameCommentEntries(authoredGalgameCommentEntries(posts), after, limit)
	owner := s.userClient.Hydrate(ctx, []int{userID})[userID]
	items := make([]dto.UserGalgameComment, 0, len(page))
	for _, entry := range page {
		av := entry.Post
		items = append(items, dto.UserGalgameComment{
			ID:          av.Post.ID,
			WorkID:      anchorWorkID(av.Thread),
			Content:     av.Post.ContentRaw,
			ContentHtml: markdown.Render(av.Post.ContentRaw),
			User:        dto.UserBrief{ID: owner.ID, Name: owner.Name, Avatar: owner.Avatar},
			Created:     av.Post.CreatedAt,
			Deleted:     false,
		})
	}
	return items, nextCursor, nil
}

func (s *UserContentService) likedGalgameComments(ctx context.Context, userID int, after string, limit int) ([]dto.UserGalgameComment, string, *errors.AppError) {
	rows, err := s.userContentRepo.FindUserLikedPostIDs(userID)
	if err != nil {
		return nil, "", errors.ErrInternal("获取用户 Galgame 评论列表失败")
	}
	if len(rows) == 0 {
		return []dto.UserGalgameComment{}, "", nil
	}

	postIDs := make([]int64, len(rows))
	for i, row := range rows {
		postIDs[i] = row.PostID
	}
	posts, err := s.resolveGalgameCommentPosts(ctx, postIDs)
	if err != nil {
		if communityDown(err) {
			return []dto.UserGalgameComment{}, "", nil
		}
		return nil, "", errors.ErrInternal("获取用户 Galgame 评论列表失败")
	}

	page, nextCursor := paginateGalgameCommentEntries(likedGalgameCommentEntries(postIDs, posts), after, limit)
	authorIDs := make([]int, 0, len(page))
	for _, entry := range page {
		if entry.Post != nil {
			authorIDs = append(authorIDs, int(entry.Post.Post.AuthorID))
		}
	}
	userMap := s.userClient.Hydrate(ctx, authorIDs)

	items := make([]dto.UserGalgameComment, 0, len(page))
	for _, entry := range page {
		if entry.Post == nil {
			items = append(items, dto.UserGalgameComment{ID: entry.ID, Deleted: true, User: dto.UserBrief{}})
			continue
		}
		av := entry.Post
		author := userMap[int(av.Post.AuthorID)]
		if !userclient.IsRenderable(author) {
			items = append(items, dto.UserGalgameComment{ID: entry.ID, Deleted: true, User: dto.UserBrief{}})
			continue
		}
		items = append(items, dto.UserGalgameComment{
			ID:          av.Post.ID,
			WorkID:      anchorWorkID(av.Thread),
			Content:     av.Post.ContentRaw,
			ContentHtml: markdown.Render(av.Post.ContentRaw),
			User:        dto.UserBrief{ID: author.ID, Name: author.Name, Avatar: author.Avatar},
			Created:     av.Post.CreatedAt,
			Deleted:     false,
		})
	}
	return items, nextCursor, nil
}

func communityDown(err error) bool {
	if stderrors.Is(err, communityclient.ErrNotConfigured) || stderrors.Is(err, communityclient.ErrForbidden) {
		return true
	}
	var apiErr *communityclient.APIError
	return err != nil && !stderrors.As(err, &apiErr) && !stderrors.Is(err, communityclient.ErrRateLimited)
}

func anchorWorkID(thread communityclient.PostThreadContext) int {
	if thread.AnchorKind != communityclient.AnchorSiteGame {
		return 0
	}
	workID, _ := strconv.Atoi(thread.AnchorID)
	return workID
}

func (s *UserContentService) GetUserResources(
	ctx context.Context,
	userID int,
	req *dto.UserResourcesRequest,
	isSFW bool,
) (*dto.UserResourcesResponse, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return &dto.UserResourcesResponse{Resources: []dto.UserResourceItem{}, Total: 0}, nil
	}
	rows, total, err := s.userContentRepo.FindUserResources(userID, req.Type, req.Page, req.Limit)
	if err != nil {
		return nil, errors.ErrInternal("获取用户资源列表失败")
	}

	resourceIDs := make([]int, len(rows))
	workIDs := collectUniqueIDs(rows, func(r repository.UserResource) int { return r.WorkID })
	for i, r := range rows {
		resourceIDs[i] = r.ID
	}

	var linkMap map[int][]string
	if len(resourceIDs) > 0 {
		linkMap, _ = s.userContentRepo.FindResourceLinks(resourceIDs)
	}

	var briefMap map[int]client.GalgameBrief
	if len(workIDs) > 0 {
		briefMap, _ = s.galgameClient.GetBatchPublic(ctx, workIDs, isSFW)
	}

	items := make([]dto.UserResourceItem, 0, len(rows))
	for _, r := range rows {
		b, hasBrief := briefMap[r.WorkID]
		if !hasBrief {
			continue
		}
		links := linkMap[r.ID]
		if links == nil {
			links = []string{}
		}
		name := b.Name
		items = append(items, dto.UserResourceItem{
			ID:          r.ID,
			WorkID:      r.WorkID,
			GalgameName: name,
			Type:        r.Type,
			Language:    r.Language,
			Platform:    r.Platform,
			Size:        r.Size,
			Link:        links,
			Code:        r.Code,
			Password:    r.Password,
			Note:        r.Note,
			Status:      r.Status,
			Created:     r.Created,
		})
	}

	return &dto.UserResourcesResponse{Resources: items, Total: total}, nil
}

func (s *UserContentService) GetUserRatings(
	ctx context.Context,
	userID int,
	req *dto.UserRatingsRequest,
	isSFW bool,
) (*dto.UserRatingsResponse, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return &dto.UserRatingsResponse{RatingData: []dto.UserRatingItem{}, Total: 0}, nil
	}
	rows, total, err := s.userContentRepo.FindUserRatings(userID, req.Page, req.Limit)
	if err != nil {
		return nil, errors.ErrInternal("获取用户评分列表失败")
	}

	workIDs := collectUniqueIDs(rows, func(r repository.UserRating) int { return r.WorkID })
	var briefMap map[int]client.GalgameBrief
	if len(workIDs) > 0 {
		briefMap, _ = s.galgameClient.GetBatchPublic(ctx, workIDs, isSFW)
	}

	uids := collectUniqueIDs(rows, func(r repository.UserRating) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.UserRatingItem, 0, len(rows))
	for _, r := range rows {
		b, hasBrief := briefMap[r.WorkID]
		if !hasBrief {
			continue
		}
		var galgameType []string
		if r.GalgameType != "" {
			_ = json.Unmarshal([]byte(r.GalgameType), &galgameType)
		}

		galgame := dto.UserRatingGalgame{ID: r.WorkID}
		if hasBrief {
			galgame = dto.UserRatingGalgame{
				ID:           b.ID,
				Name:         b.Name,
				ContentLimit: b.ContentLimit,
			}
		}

		u := userMap[r.UserID]
		items = append(items, dto.UserRatingItem{
			ID:           r.ID,
			User:         dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Recommend:    r.Recommend,
			Overall:      r.Overall,
			View:         r.View,
			GalgameType:  galgameType,
			PlayStatus:   r.PlayStatus,
			ShortSummary: r.ShortSummary,
			Art:          r.Art,
			Story:        r.Story,
			Music:        r.Music,
			Character:    r.Character,
			Route:        r.Route,
			System:       r.System,
			Voice:        r.Voice,
			ReplayValue:  r.ReplayValue,
			SpoilerLevel: r.SpoilerLevel,
			LikeCount:    r.LikeCount,
			Created:      r.Created,
			Updated:      r.Updated,
			Galgame:      galgame,
		})
	}

	return &dto.UserRatingsResponse{RatingData: items, Total: total}, nil
}
