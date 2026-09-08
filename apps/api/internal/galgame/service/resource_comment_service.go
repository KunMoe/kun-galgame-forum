package service

import (
	"context"
	"strconv"

	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type ResourceCommentService struct {
	community  *communityclient.Client
	posts      *repository.CommunityPostRepository
	userClient *userclient.Client
	db         *gorm.DB
	helpers    InteractionHelpers
}

func NewResourceCommentService(
	community *communityclient.Client,
	posts *repository.CommunityPostRepository,
	userClient *userclient.Client,
	db *gorm.DB,
) *ResourceCommentService {
	return &ResourceCommentService{community: community, posts: posts, userClient: userClient, db: db}
}

type CommentSource struct {
	key        string
	anchorPref string
	feedType   string
	// /galgame-resource/:id is not a page. The legacy route rule only expands **
	// on the server, so an in-app navigation landed on the literal
	// /galgame/resource/**. Website pages are keyed by slug, not id, so this
	// stays empty and notifyWebsiteReply builds "/website/"+slug itself.
	linkPrefix string
}

var (
	sourceRating   = CommentSource{key: "rating", anchorPref: "rating:", feedType: "GALGAME_RATING_COMMENT_CREATION", linkPrefix: "/galgame-rating/"}
	sourceWebsite  = CommentSource{key: "website", anchorPref: "website:", feedType: "GALGAME_WEBSITE_COMMENT_CREATION"}
	sourceToolset  = CommentSource{key: "toolset", anchorPref: "toolset:", feedType: "TOOLSET_COMMENT_CREATION", linkPrefix: "/toolset/"}
	sourceResource = CommentSource{key: "resource", anchorPref: "resource:", feedType: "GALGAME_RESOURCE_COMMENT_CREATION", linkPrefix: "/galgame/resource/"}
	sourceQuiz     = CommentSource{key: "quiz", anchorPref: "quiz:", feedType: "GALGAME_QUIZ_COMMENT_CREATION", linkPrefix: "/galgame-quiz/"}
)

func SourceRating() CommentSource   { return sourceRating }
func SourceWebsite() CommentSource  { return sourceWebsite }
func SourceToolset() CommentSource  { return sourceToolset }
func SourceResource() CommentSource { return sourceResource }
func SourceQuiz() CommentSource     { return sourceQuiz }

func (src CommentSource) anchorID(resourceID int) string {
	return src.anchorPref + strconv.Itoa(resourceID)
}

func (src CommentSource) pageLink(resourceID int) string {
	return src.linkPrefix + strconv.Itoa(resourceID)
}

func (s *ResourceCommentService) GetComments(ctx context.Context, src CommentSource, resourceID, viewerID int, cursor string, limit int) (*CommunityCommentPage, *errors.AppError) {
	if s.commentAreaLocked(ctx, src, resourceID, viewerID) {
		return lockedCommentPage(), nil
	}

	thread, err := s.community.ResolveComments(ctx, communityclient.ResolveCommentsRequest{
		AnchorKind: communityclient.AnchorSiteResource, AnchorID: src.anchorID(resourceID), ContentRating: communityclient.RatingAll,
	})
	if err != nil {
		if isCommunityDown(err) {
			return emptyCommentPage(), nil
		}
		return nil, mapCommunityError(err)
	}

	posts, next := thread.Posts, thread.NextCursor
	if cursor != "" {
		page, perr := s.community.ListPosts(ctx, thread.Thread.ID, cursor, clampReadLimit(limit))
		if perr != nil {
			if isCommunityDown(perr) {
				return emptyCommentPage(), nil
			}
			return nil, mapCommunityError(perr)
		}
		posts, next = page.Posts, page.NextCursor
	}

	items := s.renderPosts(ctx, viewerID, posts)
	return &CommunityCommentPage{
		ThreadID: thread.Thread.ID, Posts: items, NextCursor: next, Total: int(thread.Thread.PostsCount),
	}, nil
}

func (s *ResourceCommentService) renderPosts(ctx context.Context, viewerID int, posts []communityclient.PostView) []*CommunityPostItem {
	userMap := s.userClient.Hydrate(ctx, collectPostUserIDs(posts))

	postIDs := make([]int64, 0, len(posts))
	for _, p := range posts {
		postIDs = append(postIDs, p.ID)
	}
	likeCounts := s.posts.CountLikes(postIDs)
	likedSet := s.posts.LikedSet(viewerID, postIDs)

	out := make([]*CommunityPostItem, 0, len(posts))
	for _, p := range posts {
		author := userMap[int(p.AuthorID)]
		if !userclient.IsRenderable(author) {
			continue
		}
		if !visibleTo(p.Status, p.AuthorID, viewerID) {
			continue
		}
		item := buildCommunityItem(p, 0, author, likeCounts[p.ID], likedSet[p.ID])
		if p.TargetUserID != 0 {
			t := userMap[int(p.TargetUserID)]
			tu := UserObj{ID: t.ID, Name: t.Name, Avatar: t.Avatar}
			item.TargetUser = &tu
		}
		out = append(out, item)
	}
	return out
}
