package service

import (
	"context"
	"log/slog"
	"strings"
	"unicode/utf8"

	"kun-galgame-api/internal/community/anchor"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/search/dto"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

// The community service refuses anything outside this window: one character
// extracts no trigram, so the index cannot help and the scan is the corpus.
const (
	galCommentMinRunes = 2
	galCommentMaxRunes = 100
)

const galCommentDefaultLimit = 24

// The same window the SQL lanes cut with, so a community row reads like the
// rows beside it in the overview.
const (
	galCommentSnippetLen  = 233
	galCommentSnippetLead = 30
)

// SearchGalComments searches the comment walls that live in the community
// primitive — Galgame pages and the resource pages around them. It is keyset,
// not offset: the upstream face answers a cursor and no total, and the rows a
// page yields vary anyway because walls anchored by another site are dropped
// here.
func (s *SearchService) SearchGalComments(
	ctx context.Context,
	raw, cursor string,
	limit int,
) (*dto.GalCommentResult, *errors.AppError) {
	q := strings.TrimSpace(raw)
	if n := utf8.RuneCountInString(q); n < galCommentMinRunes || n > galCommentMaxRunes {
		return nil, errors.ErrBadRequest("搜索评论需要 2-100 个字符")
	}
	// A forum running without the community backend has no comment walls at all,
	// so the lane is empty rather than broken.
	if s.community == nil || s.anchors == nil || !s.community.Configured() {
		return emptyGalComments(), nil
	}
	if limit <= 0 {
		limit = galCommentDefaultLimit
	}

	page, err := s.community.SearchPosts(ctx, q, communityclient.KindComments, cursor, limit)
	if err != nil {
		slog.Warn("search: community post search failed", "error", err)
		return nil, errors.ErrInternal("搜索评论失败")
	}
	return s.renderGalComments(ctx, q, page), nil
}

func (s *SearchService) renderGalComments(ctx context.Context, q string, page *communityclient.PostFeedResponse) *dto.GalCommentResult {
	refs := make([]anchor.Ref, 0, len(page.Posts))
	uids := make([]int, 0, len(page.Posts))
	for _, row := range page.Posts {
		refs = append(refs, anchor.Ref{Kind: row.Thread.AnchorKind, ID: row.Thread.AnchorID})
		uids = append(uids, int(row.Post.AuthorID))
	}
	targets := s.anchors.ResolveNamed(ctx, refs)
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.GalCommentItem, 0, len(page.Posts))
	for _, row := range page.Posts {
		target, ok := targets[anchor.Ref{Kind: row.Thread.AnchorKind, ID: row.Thread.AnchorID}]
		if !ok {
			continue
		}
		author := userMap[int(row.Post.AuthorID)]
		if !userclient.IsRenderable(author) {
			continue
		}
		title := target.Title
		if title == "" {
			title = target.Label
		}
		items = append(items, dto.GalCommentItem{
			ID:      row.Post.ID,
			Content: galCommentSnippet(row.Post.ContentRaw, q),
			Link:    target.Link,
			Title:   title,
			Label:   target.Label,
			WorkID:  target.WorkID,
			User:    dto.UserBrief{ID: author.ID, Name: author.Name, Avatar: author.Avatar},
			Created: row.Post.CreatedAt,
		})
	}
	return &dto.GalCommentResult{Items: items, NextCursor: page.NextCursor}
}

func emptyGalComments() *dto.GalCommentResult {
	return &dto.GalCommentResult{Items: []dto.GalCommentItem{}}
}

// galCommentSnippet windows the plain text around the first hit. Without it the
// card shows the opening of a long comment and highlights nothing, since the
// match the reader searched for is a thousand characters further down.
func galCommentSnippet(raw, q string) string {
	// The cap is the source's own length: stripping markdown only ever shortens
	// it, so this never truncates, and the window below needs the whole text to
	// find a hit near the end of a long comment.
	text := markdown.ToPlainText(raw, utf8.RuneCountInString(raw))
	runes := []rune(text)
	if len(runes) <= galCommentSnippetLen {
		return text
	}
	at := utf8.RuneCountInString(text[:max(strings.Index(strings.ToLower(text), strings.ToLower(q)), 0)])
	if at <= galCommentSnippetLead {
		return string(runes[:galCommentSnippetLen])
	}
	start := at - galCommentSnippetLead
	end := min(start+galCommentSnippetLen, len(runes))
	return "…" + string(runes[start:end])
}
