package push

import (
	"errors"
	"strconv"
	"strings"
	"time"

	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/communityclient"
)

var ErrNotPushed = errors.New("activity push: type is not pushed")

var zhLocales = []string{"zh-Hans", "zh", "zh-Hant"}

const (
	titleRunes   = 200
	excerptRunes = 300
	occurredFmt  = "2006-01-02T15:04:05.000000Z"
)

type spec struct {
	Verb        string
	ObjectKind  string
	ObjectLabel string
	Notify      bool
	WorkTied    bool
	TopicKind   bool
}

var pushed = map[string]spec{
	"TOPIC_CREATION":                    {Verb: "publish", ObjectKind: "topic", ObjectLabel: "话题", Notify: true, TopicKind: true},
	"GALGAME_RESOURCE_CREATION":         {Verb: "publish", ObjectKind: "galgame_resource", ObjectLabel: "Galgame 资源", Notify: true, WorkTied: true},
	"TOOLSET_CREATION":                  {Verb: "publish", ObjectKind: "toolset", ObjectLabel: "工具集", Notify: true},
	"TOOLSET_RESOURCE_CREATION":         {Verb: "publish", ObjectKind: "toolset_resource", ObjectLabel: "工具资源", Notify: true},
	"GALGAME_QUIZ_CREATION":             {Verb: "publish", ObjectKind: "galgame_quiz", ObjectLabel: "题目", Notify: true, WorkTied: true},
	"GALGAME_WEBSITE_CREATION":          {Verb: "publish", ObjectKind: "galgame_website", ObjectLabel: "网站", Notify: true},
	"GALGAME_CREATION":                  {Verb: "publish", ObjectKind: "galgame", ObjectLabel: "Galgame", Notify: true, WorkTied: true},
	"TOPIC_REPLY_CREATION":              {Verb: "reply", ObjectKind: "topic_reply", ObjectLabel: "回复", TopicKind: true},
	"TOPIC_COMMENT_CREATION":            {Verb: "comment", ObjectKind: "topic_comment", ObjectLabel: "话题评论", TopicKind: true},
	"GALGAME_COMMENT_CREATION":          {Verb: "comment", ObjectKind: "galgame_comment", ObjectLabel: "Galgame 评论", WorkTied: true},
	"GALGAME_RESOURCE_COMMENT_CREATION": {Verb: "comment", ObjectKind: "galgame_resource_comment", ObjectLabel: "资源评论", WorkTied: true},
	"GALGAME_RATING_COMMENT_CREATION":   {Verb: "comment", ObjectKind: "galgame_rating_comment", ObjectLabel: "评分评论", WorkTied: true},
	"GALGAME_QUIZ_COMMENT_CREATION":     {Verb: "comment", ObjectKind: "galgame_quiz_comment", ObjectLabel: "题目评论", WorkTied: true},
	"GALGAME_WEBSITE_COMMENT_CREATION":  {Verb: "comment", ObjectKind: "galgame_website_comment", ObjectLabel: "网站评论"},
	"TOOLSET_COMMENT_CREATION":          {Verb: "comment", ObjectKind: "toolset_comment", ObjectLabel: "工具集评论"},
	"GALGAME_RATING_CREATION":           {Verb: "rate", ObjectKind: "galgame_rating", ObjectLabel: "评分", WorkTied: true},
	"TOPIC_UPVOTE":                      {Verb: "like", ObjectKind: "topic", ObjectLabel: "话题", TopicKind: true},
	"GALGAME_EDIT":                      {Verb: "edit", ObjectKind: "galgame", ObjectLabel: "Galgame", WorkTied: true},
	"GALGAME_PR_CREATION":               {Verb: "edit", ObjectKind: "galgame", ObjectLabel: "Galgame", WorkTied: true},
}

var neverPushed = map[string]struct{}{
	"MESSAGE_SOLUTION":    {},
	"TODO_CREATION":       {},
	"UPDATE_LOG_CREATION": {},
}

func IsPushed(feedType string) bool {
	_, ok := pushed[feedType]
	return ok
}

func KeyOf(feedType string, sourceID int) string {
	kind := activityapiv1.KindOfFeed(feedType)
	if kind == "" {
		kind = strings.ToLower(feedType)
	}
	return kind + ":" + strconv.Itoa(sourceID)
}

func MapLive(feedType string, sourceID int, a *activityapiv1.Activity, pageName string, isNSFW, backfill bool, origin string, rev int64, occurred time.Time) (communityclient.ActivityWriteItem, error) {
	sp, ok := pushed[feedType]
	if !ok {
		return communityclient.ActivityWriteItem{}, ErrNotPushed
	}
	title := cutRunes(pageTitle(feedType, a, pageName), titleRunes)
	if title == "" {
		title = sp.ObjectLabel
	}
	item := communityclient.ActivityWriteItem{
		Key:         KeyOf(feedType, sourceID),
		ActorID:     int64(mustID(a.Performer.ID)),
		Revision:    rev,
		Verb:        sp.Verb,
		ObjectKind:  sp.ObjectKind,
		ObjectLabel: sp.ObjectLabel,
		Title:       title,
		Excerpt:     cutRunes(excerptOf(feedType, a), excerptRunes),
		URL:         origin + a.Path,
		OccurredAt:  occurred.UTC().Format(occurredFmt),
	}
	if sp.WorkTied && a.Work != nil {
		if id := mustID(a.Work.ID); id > 0 {
			item.WorkID = int64(id)
		}
		if hash := coverHash(a.Work.Cover); hash != "" {
			item.CoverImageHash = hash
		}
	}
	if sp.TopicKind {
		if hash := topicCover(a); hash != "" {
			item.CoverImageHash = hash
		}
	}
	item.ContentLimit = "sfw"
	if isNSFW || workNSFW(a) || topicNSFW(a) {
		item.ContentLimit = "nsfw"
	}
	if sp.Verb == "publish" && sp.Notify && !backfill {
		item.Notify = true
	}
	return item, nil
}

func Tombstone(key string, actorID int64, rev int64) communityclient.ActivityWriteItem {
	return communityclient.ActivityWriteItem{Key: key, ActorID: actorID, Revision: rev, Removed: true}
}

func pageTitle(feedType string, a *activityapiv1.Activity, pageName string) string {
	switch feedType {
	case "TOPIC_CREATION", "TOPIC_UPVOTE":
		if a.Topic != nil {
			return a.Topic.Title
		}
	case "TOPIC_REPLY_CREATION":
		if a.Reply != nil {
			return a.Reply.TopicTitle
		}
	case "TOPIC_COMMENT_CREATION":
		if a.Comment != nil {
			return a.Comment.TopicTitle
		}
	case "TOOLSET_CREATION", "GALGAME_WEBSITE_CREATION":
		return a.ExcerptMarkdown
	case "TOOLSET_RESOURCE_CREATION":
		if a.Toolset != nil {
			return a.Toolset.Title
		}
	case "TOOLSET_COMMENT_CREATION", "GALGAME_WEBSITE_COMMENT_CREATION":
		return pageName
	default:
		if a.Work != nil {
			return workName(a.Work.CatalogName)
		}
	}
	return ""
}

func workName(n repr.CatalogName) string {
	for _, locale := range zhLocales {
		if v := n.Localized[locale].Value; v != "" {
			return v
		}
	}
	latin := ""
	if n.Latin != nil {
		latin = *n.Latin
	}
	return client.PickCatalogName(n.DisplayName, latin)
}

func excerptOf(feedType string, a *activityapiv1.Activity) string {
	switch feedType {
	case "TOPIC_CREATION":
		if a.TopicDigest != nil {
			return content.PlainText(a.TopicDigest.ExcerptMarkdown, excerptRunes)
		}
	case "TOPIC_REPLY_CREATION", "TOPIC_COMMENT_CREATION",
		"GALGAME_COMMENT_CREATION", "GALGAME_RESOURCE_COMMENT_CREATION",
		"GALGAME_RATING_COMMENT_CREATION", "TOOLSET_COMMENT_CREATION",
		"GALGAME_WEBSITE_COMMENT_CREATION":
		return content.PlainText(a.ExcerptMarkdown, excerptRunes)
	case "GALGAME_QUIZ_COMMENT_CREATION":
		return ""
	case "TOPIC_UPVOTE":
		return content.PlainText(a.ExcerptMarkdown, excerptRunes)
	case "GALGAME_RATING_CREATION":
		if a.GalgameRating != nil {
			return a.GalgameRating.ShortSummary
		}
	case "GALGAME_RESOURCE_CREATION":
		if a.Resource != nil && a.Resource.Note != nil {
			return *a.Resource.Note
		}
	case "GALGAME_QUIZ_CREATION":
		return content.PlainText(a.ExcerptMarkdown, excerptRunes)
	}
	return ""
}

func topicCover(a *activityapiv1.Activity) string {
	if a.Topic == nil {
		return ""
	}
	for i := range a.Topic.CoverImages {
		if hash := coverHash(&a.Topic.CoverImages[i]); hash != "" {
			return hash
		}
	}
	return ""
}

func coverHash(img *repr.Image) string {
	if img == nil {
		return ""
	}
	if len(img.Hash) != 64 {
		return ""
	}
	for _, r := range img.Hash {
		if r < '0' || (r > '9' && r < 'a') || r > 'f' {
			return ""
		}
	}
	return img.Hash
}

func workNSFW(a *activityapiv1.Activity) bool {
	return a.Work != nil && a.Work.IsNSFW
}

func topicNSFW(a *activityapiv1.Activity) bool {
	return a.Topic != nil && a.Topic.IsNSFW
}

func mustID(id repr.DecimalID) int {
	n, _ := strconv.Atoi(string(id))
	return n
}

func cutRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
