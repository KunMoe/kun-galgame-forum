package push

import (
	"strings"
	"testing"
	"time"

	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
)

const (
	testOrigin = "https://www.kungal.com"
	testHash   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testRev    = int64(1_700_000_000_000_000)
)

var occurred = time.Date(2026, 9, 26, 4, 30, 0, 123456000, time.UTC)

func TestFeedTypePartition(t *testing.T) {
	pairs := activityapiv1.FeedTypePairs()
	if len(pairs) != 22 {
		t.Fatalf("feedTypes = %d, want 22", len(pairs))
	}
	var in, out []string
	seen := map[string]bool{}
	for _, p := range pairs {
		seen[p.Feed] = true
		if IsPushed(p.Feed) {
			in = append(in, p.Feed)
		} else {
			out = append(out, p.Feed)
		}
	}
	if len(in) != 19 {
		t.Errorf("pushed from feedTypes = %d %v, want 19", len(in), in)
	}
	if len(out) != 3 {
		t.Errorf("never-pushed from feedTypes = %d %v, want 3", len(out), out)
	}
	for _, feed := range []string{"MESSAGE_SOLUTION", "TODO_CREATION", "UPDATE_LOG_CREATION"} {
		if IsPushed(feed) || !seen[feed] {
			t.Errorf("%s must be in feedTypes and never pushed", feed)
		}
	}
	if activityapiv1.KindOfFeed("MESSAGE_UPVOTE") != "" || IsPushed("MESSAGE_UPVOTE") {
		t.Error("MESSAGE_UPVOTE must stay outside feedTypes and the push table")
	}
	if got := len(pushed); got != 19 {
		t.Errorf("pushed table = %d, want 19", got)
	}
	for feed := range pushed {
		if !seen[feed] {
			t.Errorf("pushed %s is not in feedTypes", feed)
		}
	}
	for feed := range neverPushed {
		if IsPushed(feed) {
			t.Errorf("%s is both never-pushed and pushed", feed)
		}
	}
}

func TestMapLiveEveryPushedType(t *testing.T) {
	note := "publisher note"
	cover := &repr.Image{Hash: testHash}
	work := &repr.WorkRef{ID: repr.ID(88), CatalogName: repr.NewCatalogName("Work 88", "", nil), Cover: cover, IsNSFW: false}
	topic := &topicapiv1.TopicSummary{Title: "Topic title", IsNSFW: false, CoverImages: []repr.Image{*cover}}
	digest := &activityapiv1.TopicDigest{ExcerptMarkdown: "body **md**"}
	performer := &repr.UserRef{Object: "user", ID: repr.ID(7)}
	toolset := &activityapiv1.ActivityToolset{Title: "Toolset title"}

	type want struct {
		key, verb, kind, label, title, excerptSrc string
		notify, work, cover                       bool
		limit                                     string
	}
	cases := []struct {
		feed     string
		source   int
		nsfw     bool
		backfill bool
		a        *activityapiv1.Activity
		want     want
	}{
		{"TOPIC_CREATION", 123, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/topic/123", Topic: topic, TopicDigest: digest, ExcerptMarkdown: "Topic title",
		}, want{"topic_creation:123", "publish", "topic", "话题", "Topic title", "topicDigest.ExcerptMarkdown", true, false, true, "sfw"}},
		{"GALGAME_RESOURCE_CREATION", 88, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/galgame/resource/88", Work: work, Resource: &activityapiv1.ActivityResource{Note: &note},
		}, want{"galgame_resource_creation:88", "publish", "galgame_resource", "Galgame 资源", "Work 88", "resource.Note", true, true, true, "sfw"}},
		{"TOOLSET_CREATION", 4, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/toolset/4", ExcerptMarkdown: "Toolset title",
		}, want{"toolset_creation:4", "publish", "toolset", "工具集", "Toolset title", "empty", true, false, false, "sfw"}},
		{"TOOLSET_RESOURCE_CREATION", 5, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/toolset/4", Toolset: toolset,
		}, want{"toolset_resource_creation:5", "publish", "toolset_resource", "工具资源", "Toolset title", "empty", true, false, false, "sfw"}},
		{"GALGAME_QUIZ_CREATION", 9, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/galgame-quiz/9", Work: work, ExcerptMarkdown: "What is X?",
		}, want{"galgame_quiz_creation:9", "publish", "galgame_quiz", "题目", "Work 88", "excerptMarkdown", true, true, true, "sfw"}},
		{"GALGAME_WEBSITE_CREATION", 2, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/website/site.example", ExcerptMarkdown: "Site name",
		}, want{"galgame_website_creation:2", "publish", "galgame_website", "网站", "Site name", "empty", true, false, false, "sfw"}},
		{"GALGAME_CREATION", 88, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/galgame/88", Work: work,
		}, want{"galgame_creation:88", "publish", "galgame", "Galgame", "Work 88", "empty", true, true, true, "sfw"}},
		{"TOPIC_REPLY_CREATION", 10, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/topic/123?reply=1", ExcerptMarkdown: "a **reply**",
			Reply: &activityapiv1.ActivityReply{TopicTitle: "Topic title"},
		}, want{"topic_reply_creation:10", "reply", "topic_reply", "回复", "Topic title", "excerptMarkdown", false, false, false, "sfw"}},
		{"TOPIC_COMMENT_CREATION", 11, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/topic/123", ExcerptMarkdown: "a comment",
			Comment: &activityapiv1.ActivityComment{TopicTitle: "Topic title"},
		}, want{"topic_comment_creation:11", "comment", "topic_comment", "话题评论", "Topic title", "excerptMarkdown", false, false, false, "sfw"}},
		{"GALGAME_COMMENT_CREATION", 12, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/galgame/88", Work: work, ExcerptMarkdown: "wall comment",
		}, want{"galgame_comment_creation:12", "comment", "galgame_comment", "Galgame 评论", "Work 88", "excerptMarkdown", false, true, true, "sfw"}},
		{"GALGAME_RESOURCE_COMMENT_CREATION", 13, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/galgame/resource/88", Work: work, ExcerptMarkdown: "res comment",
		}, want{"galgame_resource_comment_creation:13", "comment", "galgame_resource_comment", "资源评论", "Work 88", "excerptMarkdown", false, true, true, "sfw"}},
		{"GALGAME_RATING_COMMENT_CREATION", 14, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/galgame-rating/1", Work: work, ExcerptMarkdown: "rating comment",
		}, want{"galgame_rating_comment_creation:14", "comment", "galgame_rating_comment", "评分评论", "Work 88", "excerptMarkdown", false, true, true, "sfw"}},
		{"GALGAME_QUIZ_COMMENT_CREATION", 15, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/galgame-quiz/9", Work: work, ExcerptMarkdown: "would spoil",
		}, want{"galgame_quiz_comment_creation:15", "comment", "galgame_quiz_comment", "题目评论", "Work 88", "empty", false, true, true, "sfw"}},
		{"GALGAME_WEBSITE_COMMENT_CREATION", 16, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/website/site.example", ExcerptMarkdown: "site comment",
		}, want{"galgame_website_comment_creation:16", "comment", "galgame_website_comment", "网站评论", "Page name", "excerptMarkdown", false, false, false, "sfw"}},
		{"TOOLSET_COMMENT_CREATION", 17, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/toolset/4", ExcerptMarkdown: "tool comment",
		}, want{"toolset_comment_creation:17", "comment", "toolset_comment", "工具集评论", "Page name", "excerptMarkdown", false, false, false, "sfw"}},
		{"GALGAME_RATING_CREATION", 18, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/galgame-rating/18", Work: work,
			GalgameRating: &activityapiv1.ActivityRating{ShortSummary: "worth a try"},
		}, want{"galgame_rating_creation:18", "rate", "galgame_rating", "评分", "Work 88", "rating.ShortSummary", false, true, true, "sfw"}},
		{"TOPIC_UPVOTE", 19, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/topic/123", ExcerptMarkdown: "nice", Topic: topic,
		}, want{"topic_upvote:19", "like", "topic", "话题", "Topic title", "excerptMarkdown", false, false, true, "sfw"}},
		{"GALGAME_EDIT", 20, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/galgame/88", Work: work,
		}, want{"galgame_edit:20", "edit", "galgame", "Galgame", "Work 88", "empty", false, true, true, "sfw"}},
		{"GALGAME_PR_CREATION", 21, false, false, &activityapiv1.Activity{
			Performer: performer, Path: "/galgame/88", Work: work,
		}, want{"galgame_pr_creation:21", "edit", "galgame", "Galgame", "Work 88", "empty", false, true, true, "sfw"}},
	}

	if len(cases) != len(pushed) {
		t.Fatalf("cases = %d, pushed = %d", len(cases), len(pushed))
	}
	seen := map[string]bool{}
	for _, tc := range cases {
		seen[tc.feed] = true
		page := ""
		if tc.feed == "TOOLSET_COMMENT_CREATION" || tc.feed == "GALGAME_WEBSITE_COMMENT_CREATION" {
			page = "Page name"
		}
		item, err := MapLive(tc.feed, tc.source, tc.a, page, tc.nsfw, tc.backfill, testOrigin, testRev, occurred)
		if err != nil {
			t.Fatalf("%s: %v", tc.feed, err)
		}
		if item.Key != tc.want.key || item.Verb != tc.want.verb || item.ObjectKind != tc.want.kind || item.ObjectLabel != tc.want.label {
			t.Errorf("%s identity = %s %s %s %s, want %+v", tc.feed, item.Key, item.Verb, item.ObjectKind, item.ObjectLabel, tc.want)
		}
		if item.Notify != tc.want.notify {
			t.Errorf("%s notify = %v, want %v", tc.feed, item.Notify, tc.want.notify)
		}
		if item.Title != tc.want.title {
			t.Errorf("%s title = %q, want %q", tc.feed, item.Title, tc.want.title)
		}
		if tc.want.excerptSrc == "empty" && item.Excerpt != "" {
			t.Errorf("%s excerpt = %q, want empty", tc.feed, item.Excerpt)
		}
		if tc.want.excerptSrc != "empty" && item.Excerpt == "" {
			t.Errorf("%s excerpt empty (source %s)", tc.feed, tc.want.excerptSrc)
		}
		if tc.feed == "GALGAME_QUIZ_COMMENT_CREATION" && item.Excerpt != "" {
			t.Errorf("quiz comment excerpt = %q", item.Excerpt)
		}
		if (item.WorkID != 0) != tc.want.work {
			t.Errorf("%s work_id = %d, want present=%v", tc.feed, item.WorkID, tc.want.work)
		}
		if (item.CoverImageHash != "") != tc.want.cover {
			t.Errorf("%s cover = %q, want present=%v", tc.feed, item.CoverImageHash, tc.want.cover)
		}
		if item.ContentLimit != tc.want.limit {
			t.Errorf("%s content_limit = %s, want %s", tc.feed, item.ContentLimit, tc.want.limit)
		}
		if item.URL != testOrigin+tc.a.Path {
			t.Errorf("%s url = %s", tc.feed, item.URL)
		}
		if item.ActorID != 7 || item.Revision != testRev {
			t.Errorf("%s actor/rev = %d %d", tc.feed, item.ActorID, item.Revision)
		}
		if item.OccurredAt != "2026-09-26T04:30:00.123456Z" {
			t.Errorf("%s occurred_at = %s", tc.feed, item.OccurredAt)
		}
	}
	for feed := range pushed {
		if !seen[feed] {
			t.Errorf("no case for %s", feed)
		}
	}
}

func TestMapLiveBackfillClearsNotify(t *testing.T) {
	a := &activityapiv1.Activity{
		Performer: &repr.UserRef{ID: repr.ID(7)}, Path: "/topic/1",
		Topic: &topicapiv1.TopicSummary{Title: "T"}, TopicDigest: &activityapiv1.TopicDigest{},
	}
	item, err := MapLive("TOPIC_CREATION", 1, a, "", false, true, testOrigin, testRev, occurred)
	if err != nil {
		t.Fatal(err)
	}
	if item.Notify {
		t.Fatal("backfill publish still notified")
	}
}

func TestMapLiveContentLimitAndEmptyTitle(t *testing.T) {
	a := &activityapiv1.Activity{
		Performer: &repr.UserRef{ID: repr.ID(7)}, Path: "/topic/1",
		Topic: &topicapiv1.TopicSummary{Title: "T", IsNSFW: true}, TopicDigest: &activityapiv1.TopicDigest{},
	}
	item, err := MapLive("TOPIC_CREATION", 1, a, "", false, false, testOrigin, testRev, occurred)
	if err != nil || item.ContentLimit != "nsfw" {
		t.Fatalf("topic nsfw: %v %s", err, item.ContentLimit)
	}
	work := &repr.WorkRef{ID: repr.ID(1), CatalogName: repr.NewCatalogName("W", "", nil), IsNSFW: true}
	g := &activityapiv1.Activity{Performer: &repr.UserRef{ID: repr.ID(7)}, Path: "/galgame/1", Work: work}
	item, err = MapLive("GALGAME_CREATION", 1, g, "", false, false, testOrigin, testRev, occurred)
	if err != nil || item.ContentLimit != "nsfw" {
		t.Fatalf("work nsfw: %v %s", err, item.ContentLimit)
	}
	item, err = MapLive("GALGAME_CREATION", 1, g, "", true, false, testOrigin, testRev, occurred)
	if err != nil || item.ContentLimit != "nsfw" {
		t.Fatalf("row nsfw: %v %s", err, item.ContentLimit)
	}
	empty := &activityapiv1.Activity{Performer: &repr.UserRef{ID: repr.ID(7)}, Path: "/galgame/1", Work: &repr.WorkRef{ID: repr.ID(1)}}
	if item, err = MapLive("GALGAME_CREATION", 1, empty, "", false, false, testOrigin, testRev, occurred); err != nil || item.Title != "Galgame" {
		t.Fatalf("a nameless work falls back to the object label: %v %q", err, item.Title)
	}
}

func TestWorkNamePrefersChinese(t *testing.T) {
	zh := repr.NewCatalogName("オリジナル", "Original", map[string]repr.LocalizedName{"zh-Hant": {Value: "繁體名"}, "zh-Hans": {Value: "简体名"}})
	if got := workName(zh); got != "简体名" {
		t.Fatalf("zh-Hans first: %q", got)
	}
	hant := repr.NewCatalogName("オリジナル", "Original", map[string]repr.LocalizedName{"zh-Hant": {Value: "繁體名"}})
	if got := workName(hant); got != "繁體名" {
		t.Fatalf("zh-Hant when no zh-Hans: %q", got)
	}
	if got := workName(repr.NewCatalogName("オリジナル", "Original", nil)); got != "オリジナル" {
		t.Fatalf("display name without zh: %q", got)
	}
}

func TestMapLiveNeverPushed(t *testing.T) {
	a := &activityapiv1.Activity{Performer: &repr.UserRef{ID: repr.ID(7)}, Path: "/"}
	if _, err := MapLive("TODO_CREATION", 1, a, "", false, false, testOrigin, testRev, occurred); err != ErrNotPushed {
		t.Fatalf("err = %v", err)
	}
}

func TestOrigin(t *testing.T) {
	if got := Origin("https://www.kungal.com/api/auth/callback"); got != "https://www.kungal.com" {
		t.Fatalf("got %q", got)
	}
	if originReady(Origin("http://127.0.0.1:2334/api/auth/callback")) {
		t.Fatal("http loopback must not start the pusher")
	}
}

func TestCutTitleOnRunes(t *testing.T) {
	long := strings.Repeat("字", 201)
	a := &activityapiv1.Activity{
		Performer: &repr.UserRef{ID: repr.ID(7)}, Path: "/topic/1",
		Topic: &topicapiv1.TopicSummary{Title: long}, TopicDigest: &activityapiv1.TopicDigest{},
	}
	item, err := MapLive("TOPIC_CREATION", 1, a, "", false, false, testOrigin, testRev, occurred)
	if err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(item.Title)); n != 200 {
		t.Fatalf("title runes = %d", n)
	}
}

func TestMapLiveStripsControlCharacters(t *testing.T) {
	a := &activityapiv1.Activity{
		Performer: &repr.UserRef{ID: repr.ID(7)}, Path: "/topic/1",
		Topic:       &topicapiv1.TopicSummary{Title: "line\none\x00two"},
		TopicDigest: &activityapiv1.TopicDigest{ExcerptMarkdown: "a\x08b\r\nc\td"},
	}
	item, err := MapLive("TOPIC_CREATION", 1, a, "", false, false, testOrigin, testRev, occurred)
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "line one two" {
		t.Fatalf("title = %q", item.Title)
	}
	if item.Excerpt != "ab\nc\td" {
		t.Fatalf("excerpt = %q", item.Excerpt)
	}
}
