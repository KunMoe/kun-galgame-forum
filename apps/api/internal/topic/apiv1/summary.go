package apiv1

import (
	"fmt"
	"slices"
	"strings"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/imageclient"

	"github.com/danielgtaylor/huma/v2"
)

const sectionEnum = "g-walkthrough,g-chatting,g-article,g-seeking,g-news,g-releases,g-other," +
	"t-crack,t-web,t-languages,t-help,t-linux,t-practical,t-ai,t-android,t-adobe,t-algorithm,t-other," +
	"o-anime,o-comics,o-music,o-novel,o-daily,o-essay,o-forum,o-patch,o-other"

type SectionSlug string

func IsSectionSlug(s string) bool {
	return slices.Contains(strings.Split(sectionEnum, ","), s)
}

func (SectionSlug) Schema(huma.Registry) *huma.Schema {
	parts := strings.Split(sectionEnum, ",")
	enum := make([]any, len(parts))
	maxLen := 0
	for i, p := range parts {
		enum[i] = p
		if n := len(p); n > maxLen {
			maxLen = n
		}
	}
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        enum,
		MaxLength:   &maxLen,
		Description: "Section slug. Hyphenated URL segment of /section/{key}.",
	}
}

type MiniAppKind string

func (MiniAppKind) Schema(huma.Registry) *huma.Schema {
	n := 7
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        []any{"poll", "lottery"},
		MaxLength:   &n,
		Description: "Mini-app kind attached to the topic. poll or lottery.",
	}
}

type TopicSummary struct {
	Object        string         `json:"object" enum:"topic" maxLength:"5" doc:"Type discriminant. Always topic."`
	ID            repr.DecimalID `json:"id" doc:"Topic id. JSON string of a decimal integer."`
	Title         string         `json:"title" maxLength:"233" doc:"Topic title as stored. Free text; never use it as a decision input."`
	State         string         `json:"state" enum:"published,hidden" maxLength:"9" doc:"Lifecycle state. A hidden topic is visible only to its author and to staff."`
	Category      string         `json:"category" enum:"galgame,technique,others" maxLength:"9" doc:"Topic category."`
	Sections      []SectionSlug  `json:"sections" maxItems:"3" doc:"Section slugs, in stored order. Empty array if none. Hyphenated URL segments of /section/{key}."`
	CoverImages   []repr.Image   `json:"cover_images" maxItems:"9" doc:"Cover images in stored token order. Tokens that do not parse are skipped, and so are stickers: a sticker is never a cover. Empty array if none."`
	Author        repr.UserRef   `json:"author" doc:"Topic author."`
	ViewCount     int            `json:"view_count" minimum:"0" doc:"Lifetime view count."`
	LikeCount     int            `json:"like_count" minimum:"0" doc:"Like count."`
	ReplyCount    int            `json:"reply_count" minimum:"0" doc:"Reply count."`
	CommentCount  int            `json:"comment_count" minimum:"0" doc:"Comment count."`
	HasBestAnswer bool           `json:"has_best_answer" doc:"Whether a best-answer reply is set."`
	MiniApps      []MiniAppKind  `json:"mini_apps" maxItems:"2" doc:"Mini-apps attached to the topic, in registry order. Empty array if none."`
	IsNSFW        bool           `json:"is_nsfw" doc:"Whether the topic is marked NSFW."`
	BumpedAt      repr.DateTime  `json:"bumped_at" doc:"Bump time. A reply, a comment, an upvote, a new best answer, an edit of the title or body, and creating a poll or a lottery set it to now, but only for topics created within the last 3 months. Casting a vote and entering a lottery do not. It is not a last-activity time."`
	CreatedAt     repr.DateTime  `json:"created_at" doc:"Creation time."`
	UpvotedAt     *repr.DateTime `json:"upvoted_at" doc:"Time of the latest upvote. null when the topic has never been upvoted."`
}

func topicState(status int) (string, error) {
	switch status {
	case 0:
		return "published", nil
	case 1:
		return "hidden", nil
	default:
		return "", fmt.Errorf("unmapped topic status %d", status)
	}
}

func topicLifecycle(status int, hiddenBy string) (string, *string, error) {
	state, err := topicState(status)
	if err != nil {
		return "", nil, err
	}
	if status == 0 {
		return state, nil, nil
	}
	switch hiddenBy {
	case "author", "moderator", "trust":
		s := hiddenBy
		return state, &s, nil
	default:
		return "", nil, fmt.Errorf("hidden topic has invalid hidden_by %q", hiddenBy)
	}
}

func coverImagesWithMeta(cdn string, tokens model.ImageTokens, metaByHash map[string]imageclient.ImageMeta) []repr.Image {
	tokens = withoutStickers(tokens)
	if len(tokens) == 0 {
		return []repr.Image{}
	}
	out := make([]repr.Image, 0, len(tokens))
	for _, tok := range tokens {
		var meta *imageclient.ImageMeta
		if hash, _, ok := markdown.ParseContentImageRef(tok); ok {
			if m, found := metaByHash[hash]; found {
				cp := m
				meta = &cp
			}
		}
		img := repr.NewImageFromToken(cdn, tok, meta)
		if img == nil {
			continue
		}
		out = append(out, *img)
	}
	return out
}

func MapSummary(cdn string, row repository.TopicKeysetRow, author repr.UserRef, sections, miniApps []string) (TopicSummary, error) {
	return mapSummary(cdn, row, author, sections, miniApps)
}

func mapSummary(cdn string, row repository.TopicKeysetRow, author repr.UserRef, sections, miniApps []string) (TopicSummary, error) {
	state, err := topicState(row.Status)
	if err != nil {
		return TopicSummary{}, err
	}
	return TopicSummary{
		Object:        "topic",
		ID:            repr.ID(row.ID),
		Title:         row.Title,
		State:         state,
		Category:      row.Category,
		Sections:      toSectionSlugs(sections),
		CoverImages:   coverImages(cdn, row.CoverImages),
		Author:        author,
		ViewCount:     row.View,
		LikeCount:     row.LikeCount,
		ReplyCount:    row.ReplyCount,
		CommentCount:  row.CommentCount,
		HasBestAnswer: row.BestAnswerID != nil,
		MiniApps:      toMiniAppKinds(miniApps),
		IsNSFW:        row.IsNSFW,
		BumpedAt:      repr.Timestamp(row.StatusUpdateTime),
		CreatedAt:     repr.Timestamp(row.Created),
		UpvotedAt:     repr.TimestampPtr(row.UpvoteTime),
	}, nil
}

func toSectionSlugs(ss []string) []SectionSlug {
	out := make([]SectionSlug, len(ss))
	for i, s := range ss {
		out[i] = SectionSlug(s)
	}
	return out
}

func toMiniAppKinds(ss []string) []MiniAppKind {
	out := make([]MiniAppKind, len(ss))
	for i, s := range ss {
		out[i] = MiniAppKind(s)
	}
	return out
}

func coverImages(cdn string, tokens model.ImageTokens) []repr.Image {
	tokens = withoutStickers(tokens)
	if len(tokens) == 0 {
		return []repr.Image{}
	}
	metaByToken := markdown.ResolveContentImageMeta([]string(tokens))
	out := make([]repr.Image, 0, len(tokens))
	for _, tok := range tokens {
		var meta *imageclient.ImageMeta
		if m, ok := metaByToken[tok]; ok {
			meta = &m
		}
		img := repr.NewImageFromToken(cdn, tok, meta)
		if img == nil {
			continue
		}
		out = append(out, *img)
	}
	return out
}

func withoutStickers(tokens model.ImageTokens) model.ImageTokens {
	out := make(model.ImageTokens, 0, len(tokens))
	for _, tok := range tokens {
		if hash, _, ok := markdown.ParseContentImageRef(tok); ok && markdown.IsSticker(hash) {
			continue
		}
		out = append(out, tok)
	}
	return out
}
