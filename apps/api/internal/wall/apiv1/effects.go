package apiv1

import (
	"context"
	"log/slog"
	"strconv"
	"time"
	"unicode/utf8"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/constants"
	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/wall/repository"
	"kun-galgame-api/pkg/communityclient"
)

const feedPreviewLength = 100

func (sub *subject) pageLink() string {
	if sub.spec.typ == "website" {
		return sub.spec.linkPrefix + sub.websiteSlug
	}
	return sub.spec.linkPrefix + strconv.Itoa(sub.id)
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// Everything here runs after the comment is committed upstream, so a failure
// is logged and never turns the write into an error the client would retry.
func (s *Service) afterCreate(ctx context.Context, sub *subject, authorID int, body string, post *communityclient.PostView) {
	switch {
	case sub.spec.typ == "galgame":
		if err := s.store.BumpGalgame(sub.id, 1); err != nil {
			slog.Warn("wall counter bump failed (best-effort)", "subject", sub.spec.typ, "id", sub.id, "error", err)
		}
	case sub.spec.counterTable != "":
		if err := s.store.BumpCount(sub.spec.counterTable, sub.id, 1); err != nil {
			slog.Warn("wall counter bump failed (best-effort)", "subject", sub.spec.typ, "id", sub.id, "error", err)
		}
	}

	if sub.spec.hasOwner && sub.spec.typ != "galgame_rating" && post.ReplyToPostID == 0 &&
		sub.ownerID > 0 && sub.ownerID != authorID && !s.ownerFollows(ctx, sub, post.ThreadID) {
		msg := repository.Message{
			SenderID: authorID, ReceiverID: sub.ownerID, Type: "commented",
			Content: content.PlainText(body, constants.TextPreviewLength), Link: sub.pageLink(),
		}
		if err := s.store.InsertMessageOnce(msg); err != nil {
			slog.Warn("wall owner notification failed (best-effort)", "receiver_id", sub.ownerID, "link", msg.Link, "error", err)
		}
	}

	created, err := parseUpstreamTime(post.CreatedAt)
	if err != nil {
		created = time.Now()
	}
	workID := 0
	if sub.spec.typ == "galgame" {
		workID = sub.id
	}
	if err := s.store.FeedUpsert(sub.spec.feedType, post.ID, authorID, workID,
		truncateRunes(body, feedPreviewLength), sub.pageLink(), sub.isNSFW, created); err != nil {
		slog.Warn("wall feed upsert failed (best-effort)", "post_id", post.ID, "error", err)
	}
}

// ownerFollows reports whether the community service already notifies the owner
// of this wall, so the forum does not notify them a second time. A failed
// lookup counts as not following: a duplicate beats a missed notification.
func (s *Service) ownerFollows(ctx context.Context, sub *subject, threadID int64) bool {
	owner := int64(sub.ownerID)
	if threadID > 0 {
		st, err := s.community.ThreadStates(ctx, owner, []int64{threadID})
		if err != nil {
			return false
		}
		if len(st.States) > 0 {
			return true
		}
	}
	as, err := s.community.AnchorStates(ctx, owner, []communityclient.AnchorRef{{
		AnchorKind: sub.spec.anchorKind, AnchorID: sub.spec.anchorID(sub.id),
	}})
	return err == nil && len(as.States) > 0
}

func (s *Service) afterDelete(sub *subject, postID int64) {
	switch {
	case sub.spec.typ == "galgame":
		if err := s.store.BumpGalgame(sub.id, -1); err != nil {
			slog.Warn("wall counter bump failed (best-effort)", "subject", sub.spec.typ, "id", sub.id, "error", err)
		}
	case sub.spec.counterTable != "":
		if err := s.store.BumpCount(sub.spec.counterTable, sub.id, -1); err != nil {
			slog.Warn("wall counter bump failed (best-effort)", "subject", sub.spec.typ, "id", sub.id, "error", err)
		}
	}
	if err := s.store.FeedDelete(sub.spec.feedType, postID); err != nil {
		slog.Warn("wall feed delete failed (best-effort)", "post_id", postID, "error", err)
	}
	legacy, err := s.store.LegacyFeedID(sub.spec.typ == "galgame", sub.spec.legacyKey, postID)
	if err == nil && legacy > 0 {
		err = s.store.FeedDelete(sub.spec.feedType, int64(legacy))
	}
	if err != nil {
		slog.Warn("wall legacy feed delete failed (best-effort)", "post_id", postID, "error", err)
	}
}

func (s *Service) notifyNewMentions(editorID, workID int, oldBody, newBody string, postID int64) {
	old := map[int]bool{}
	for _, id := range markdown.ExtractMentionIDs(oldBody) {
		old[id] = true
	}
	var added []int
	for _, id := range dedupeMentions[struct{}](markdown.ExtractMentionIDs(newBody), editorID, nil) {
		if !old[id] {
			added = append(added, id)
		}
	}
	if len(added) == 0 {
		return
	}
	preview := truncateRunes(markdown.StripReferenceTokens(newBody), 233)
	var h galgameService.InteractionHelpers
	for _, id := range added {
		if err := h.CreateGalgameCommentMention(s.store.DB(), editorID, id, preview, workID, int(postID)); err != nil {
			slog.Warn("wall mention notification failed (best-effort)", "work_id", workID, "post_id", postID, "receiver_id", id, "error", err)
		}
	}
}
