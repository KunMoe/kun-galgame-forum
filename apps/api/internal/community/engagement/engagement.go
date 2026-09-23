package engagement

import (
	"context"
	stderrors "errors"
	"log/slog"
	"math"
	"net/http"

	"kun-galgame-api/internal/community/anchor"
	msgRepo "kun-galgame-api/internal/message/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"
)

type Service struct {
	community *communityclient.Client
	anchors   *anchor.Resolver
	messages  *msgRepo.MessageRepository
}

func New(community *communityclient.Client, anchors *anchor.Resolver, messages *msgRepo.MessageRepository) *Service {
	return &Service{community: community, anchors: anchors, messages: messages}
}

type WallState struct {
	AnchorKind        int32  `json:"anchor_kind"`
	AnchorID          string `json:"anchor_id"`
	ThreadID          int64  `json:"thread_id"`
	Following         bool   `json:"following"`
	NotificationLevel int32  `json:"notification_level"`
}

type FollowItem struct {
	AnchorKind int32  `json:"anchor_kind"`
	AnchorID   string `json:"anchor_id"`
	Link       string `json:"link"`
	Title      string `json:"title"`
	Label      string `json:"label"`
	GalgameID  int    `json:"galgame_id,omitempty"`
}

type FollowList struct {
	Items      []FollowItem `json:"items"`
	NextCursor string       `json:"next_cursor"`
}

func (s *Service) WallRead(ctx context.Context, userID int, ref anchor.Ref, threadID int64) (*WallState, *errors.AppError) {
	if _, ok := s.anchors.Resolve([]anchor.Ref{ref})[ref]; !ok {
		return nil, errors.ErrBadRequest("评论区不存在")
	}
	idle := &WallState{
		AnchorKind:        ref.Kind,
		AnchorID:          ref.ID,
		ThreadID:          threadID,
		NotificationLevel: communityclient.NotificationNormal,
	}
	if !s.community.Configured() {
		return idle, nil
	}
	// The v1 wall list does not expose the community thread id, so a v1 page
	// sends 0 and the thread is looked up here.
	if threadID == 0 {
		if page, err := s.community.GetComments(ctx, ref.Kind, ref.ID, "", "1"); err == nil && page.Thread != nil {
			threadID = page.Thread.ID
		}
	}

	var threadRow *communityclient.ThreadUserView
	var anchorRow *communityclient.AnchorSubscriptionView
	if threadID > 0 {
		existing, err := s.community.ThreadStates(ctx, int64(userID), []int64{threadID})
		if err != nil {
			slog.Warn("community engagement: thread state lookup failed", "thread_id", threadID, "error", err)
			return idle, nil
		}
		if len(existing.States) > 0 {
			threadRow = &existing.States[0]
		}
	}
	if threadRow == nil {
		states, err := s.community.AnchorStates(ctx, int64(userID), []communityclient.AnchorRef{
			{AnchorKind: ref.Kind, AnchorID: ref.ID},
		})
		if err != nil {
			slog.Warn("community engagement: anchor state lookup failed", "anchor_id", ref.ID, "error", err)
			return idle, nil
		}
		if len(states.States) > 0 {
			anchorRow = &states.States[0]
		}
	}

	watchingAnchor := anchorRow != nil && anchorRow.NotificationLevel == communityclient.NotificationWatching
	if threadID > 0 && (threadRow != nil || watchingAnchor) {
		view, err := s.community.MarkThreadRead(ctx, threadID, int64(userID), math.MaxInt32)
		if err != nil {
			slog.Warn("community engagement: mark read failed", "thread_id", threadID, "error", err)
		} else {
			threadRow = view
			if s.messages != nil {
				if mErr := s.messages.MarkCommunityThreadRead(userID, threadID, view.LastReadPostNumber); mErr != nil {
					slog.Warn("community engagement: local thread read sync failed", "thread_id", threadID, "error", mErr)
				}
			}
		}
	}

	level := int32(communityclient.NotificationNormal)
	if threadRow != nil {
		level = threadRow.NotificationLevel
	} else if anchorRow != nil {
		level = anchorRow.NotificationLevel
	}
	return &WallState{
		AnchorKind:        ref.Kind,
		AnchorID:          ref.ID,
		ThreadID:          threadID,
		Following:         level == communityclient.NotificationWatching,
		NotificationLevel: level,
	}, nil
}

func (s *Service) WallFollow(ctx context.Context, userID int, ref anchor.Ref, following bool) (*WallState, *errors.AppError) {
	if _, ok := s.anchors.Resolve([]anchor.Ref{ref})[ref]; !ok {
		return nil, errors.ErrBadRequest("评论区不存在")
	}
	if !s.community.Configured() {
		return nil, errors.ErrInternal("评论服务未配置")
	}

	page, err := s.community.GetComments(ctx, ref.Kind, ref.ID, "", "1")
	if err != nil {
		return nil, mapErr(err, "设置关注失败")
	}

	// Unfollow writes normal (1), never muted (0): muted also silences replies
	// and mentions addressed to this user.
	level := int32(communityclient.NotificationNormal)
	if following {
		level = communityclient.NotificationWatching
	}

	if _, err := s.community.SetAnchorNotification(ctx, int64(userID), ref.Kind, ref.ID, level); err != nil {
		return nil, mapErr(err, "设置关注失败")
	}

	var threadID int64
	if page.Thread != nil {
		threadID = page.Thread.ID
		if _, err := s.community.SetThreadNotification(ctx, threadID, int64(userID), level); err != nil {
			return nil, mapErr(err, "设置关注失败")
		}
	}

	return &WallState{
		AnchorKind:        ref.Kind,
		AnchorID:          ref.ID,
		ThreadID:          threadID,
		Following:         following,
		NotificationLevel: level,
	}, nil
}

func (s *Service) Following(ctx context.Context, userID int, cursor string, limit int) (*FollowList, *errors.AppError) {
	empty := &FollowList{Items: []FollowItem{}}
	if !s.community.Configured() {
		return empty, nil
	}
	page, err := s.community.ListAnchorSubscriptions(ctx, int64(userID), -1, cursor, limit)
	if err != nil {
		return nil, mapErr(err, "获取关注列表失败")
	}

	refs := make([]anchor.Ref, 0, len(page.Subscriptions))
	for _, row := range page.Subscriptions {
		if row.NotificationLevel != communityclient.NotificationWatching {
			continue
		}
		refs = append(refs, anchor.Ref{Kind: row.AnchorKind, ID: row.AnchorID})
	}
	targets := s.anchors.ResolveNamed(ctx, refs)

	items := make([]FollowItem, 0, len(page.Subscriptions))
	for _, row := range page.Subscriptions {
		if row.NotificationLevel != communityclient.NotificationWatching {
			continue
		}
		target, ok := targets[anchor.Ref{Kind: row.AnchorKind, ID: row.AnchorID}]
		if !ok {
			continue
		}
		title := target.Title
		if title == "" {
			title = target.Label
		}
		items = append(items, FollowItem{
			AnchorKind: row.AnchorKind,
			AnchorID:   row.AnchorID,
			Link:       target.Link,
			Title:      title,
			Label:      target.Label,
			GalgameID:  target.GalgameID,
		})
	}
	return &FollowList{Items: items, NextCursor: page.NextCursor}, nil
}

func mapErr(err error, fallback string) *errors.AppError {
	var apiErr *communityclient.APIError
	if stderrors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
		return errors.ErrNotFound("评论区不存在")
	}
	slog.Warn("community engagement request failed", "error", err)
	return errors.ErrInternal(fallback)
}
