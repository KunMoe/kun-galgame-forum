package notify

import (
	"strconv"
	"strings"
	"testing"

	"kun-galgame-api/internal/community/anchor"
	"kun-galgame-api/internal/message/model"
	"kun-galgame-api/pkg/communityclient"
)

func ptrInt64(n int64) *int64 { return &n }
func ptrInt32(n int32) *int32 { return &n }
func ptrStr(s string) *string { return &s }

func baseNote(kind int32) communityclient.NotificationView {
	return communityclient.NotificationView{
		ID:         11,
		UserID:     3,
		Kind:       kind,
		ThreadID:   7,
		AnchorKind: communityclient.AnchorSiteGame,
		AnchorID:   "4178",
		PostID:     ptrInt64(100),
		PostNumber: ptrInt32(2),
		ActorID:    ptrInt64(9),
		ActorCount: 1,
		ItemCount:  1,
		UpdatedAt:  "2026-09-16T00:00:00Z",
		Seq:        50,
	}
}

func TestMapNotification(t *testing.T) {
	game := &anchor.Target{Link: "/galgame/4178", Label: "Galgame", WorkID: 4178}
	quiz := &anchor.Target{Link: "/galgame-quiz/4", Label: "游戏答题"}
	top := &communityclient.PostView{ID: 100, ReplyToPostID: 0, ContentRaw: "[@kun](kungal-user:9) hello"}
	reply := &communityclient.PostView{ID: 100, ReplyToPostID: 50, ContentRaw: "a reply"}

	cases := []struct {
		name    string
		note    communityclient.NotificationView
		post    *communityclient.PostView
		target  *anchor.Target
		skip    bool
		typ     string
		link    string
		content string
		sender  int
		status  string
	}{
		{
			name: "kind 1 top-level is commented",
			note: baseNote(communityclient.InboxReplied), post: top, target: game,
			typ: "commented", link: "/galgame/4178?comment=100",
		},
		{
			name: "kind 1 reply is replied",
			note: baseNote(communityclient.InboxReplied), post: reply, target: game,
			typ: "replied", link: "/galgame/4178?comment=100",
		},
		{
			name: "kind 1 unresolved post is replied",
			note: baseNote(communityclient.InboxReplied), post: nil, target: game,
			typ: "replied", link: "/galgame/4178?comment=100", content: "",
		},
		{
			name: "kind 2 mentioned",
			note: baseNote(communityclient.InboxMentioned), post: top, target: game,
			typ: "mentioned",
		},
		{
			name: "kind 3 followed",
			note: baseNote(communityclient.InboxPosted), post: top, target: game,
			typ: "followed",
		},
		{
			name: "kind 5 liked",
			note: baseNote(communityclient.InboxLiked), post: top, target: game,
			typ: "liked",
		},
		{
			name: "kind 4 skipped",
			note: baseNote(communityclient.InboxThreadCreated), post: top, target: game,
			skip: true,
		},
		{
			name: "kind 6 skipped",
			note: baseNote(communityclient.InboxAnswerAccepted), post: top, target: game,
			skip: true,
		},
		{
			name: "kind 7 skipped",
			note: baseNote(communityclient.InboxFeedbackStatus), post: top, target: game,
			skip: true,
		},
		{
			name: "kind 8 followed needs no target",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxFollowed)
				n.ThreadID = 0
				n.AnchorKind = 0
				n.AnchorID = ""
				n.PostID = nil
				n.PostNumber = nil
				n.ActorCount = 4
				n.ItemCount = 4
				return n
			}(),
			post: nil, target: nil,
			typ: "user-followed", link: "/user/9", sender: 9,
		},
		{
			name: "kind 9 skipped",
			note: baseNote(communityclient.InboxFolloweeTopic), post: top, target: game,
			skip: true,
		},
		{
			name: "kind 10 followee activity",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxFolloweeActivity)
				n.ThreadID = 0
				n.AnchorKind = 0
				n.AnchorID = ""
				n.PostID = nil
				n.PostNumber = nil
				n.ActorCount = 1
				n.ItemCount = 3
				n.Activity = &communityclient.NotificationActivityView{
					ID: 101, Site: "kungal", Key: "kungal:topic:12", Verb: "publish",
					ObjectKind: "topic", ObjectLabel: "Topic", Title: "Hello topic",
					URL: "https://www.kungal.com/topic/12?from=feed", ContentLimit: "sfw",
					OccurredAt: "2026-09-26T00:00:00Z",
				}
				return n
			}(),
			post: nil, target: nil,
			typ: "followee-activity", link: "/topic/12?from=feed", content: "Hello topic", sender: 9,
		},
		{
			name: "kind 10 kungal.com host",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxFolloweeActivity)
				n.ThreadID = 0
				n.AnchorKind = 0
				n.AnchorID = ""
				n.PostID = nil
				n.PostNumber = nil
				n.Activity = &communityclient.NotificationActivityView{
					Title: "Bare host", URL: "https://kungal.com/galgame/9",
				}
				return n
			}(),
			post: nil, target: nil,
			typ: "followee-activity", link: "/galgame/9", content: "Bare host", sender: 9,
		},
		{
			name: "kind 10 foreign host falls back",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxFolloweeActivity)
				n.ThreadID = 0
				n.PostID = nil
				n.PostNumber = nil
				n.Activity = &communityclient.NotificationActivityView{
					Title: "Other site", URL: "https://example.com/topic/1",
				}
				return n
			}(),
			post: nil, target: nil,
			typ: "followee-activity", link: "/user/9", content: "Other site", sender: 9,
		},
		{
			name: "kind 10 over-long path falls back",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxFolloweeActivity)
				n.ThreadID = 0
				n.PostID = nil
				n.PostNumber = nil
				path := "/" + strings.Repeat("a", 100)
				n.Activity = &communityclient.NotificationActivityView{
					Title: "Long", URL: "https://www.kungal.com" + path,
				}
				return n
			}(),
			post: nil, target: nil,
			typ: "followee-activity", link: "/user/9", content: "Long", sender: 9,
		},
		{
			name: "kind 10 null activity falls back",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxFolloweeActivity)
				n.ThreadID = 0
				n.PostID = nil
				n.PostNumber = nil
				n.ItemCount = 2
				n.Activity = nil
				return n
			}(),
			post: nil, target: nil,
			typ: "followee-activity", link: "/user/9", content: "", sender: 9,
		},
		{
			name: "kind 10 read_at maps to read",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxFolloweeActivity)
				n.ThreadID = 0
				n.PostID = nil
				n.PostNumber = nil
				n.ReadAt = ptrStr("2026-09-16T01:00:00Z")
				n.Activity = &communityclient.NotificationActivityView{
					Title: "Read", URL: "https://www.kungal.com/topic/1",
				}
				return n
			}(),
			post: nil, target: nil,
			typ: "followee-activity", link: "/topic/1", status: "read", sender: 9,
		},
		{
			name: "unresolvable anchor skipped",
			note: baseNote(communityclient.InboxPosted), post: top, target: nil,
			skip: true,
		},
		{
			name: "quiz preview empty",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxPosted)
				n.AnchorKind = communityclient.AnchorSiteResource
				n.AnchorID = "quiz:4"
				n.PostID = nil
				return n
			}(),
			post: top, target: quiz,
			typ: "followed", link: "/galgame-quiz/4", content: "",
		},
		{
			name: "null actor is sender 0",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxLiked)
				n.ActorID = nil
				return n
			}(),
			post: top, target: game, typ: "liked", sender: 0,
		},
		{
			name: "read_at maps to read",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxLiked)
				n.ReadAt = ptrStr("2026-09-16T01:00:00Z")
				return n
			}(),
			post: top, target: game, typ: "liked", status: "read",
		},
		{
			name: "site_resource uses resolver link",
			note: func() communityclient.NotificationView {
				n := baseNote(communityclient.InboxPosted)
				n.AnchorKind = communityclient.AnchorSiteResource
				n.AnchorID = "toolset:3"
				n.PostID = nil
				return n
			}(),
			post: nil, target: &anchor.Target{Link: "/toolset/3", Label: "Gal 工具"},
			typ: "followed", link: "/toolset/3", content: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapNotification(tc.note, tc.post, tc.target)
			if tc.skip {
				if got != nil {
					t.Fatalf("got %+v, want skip", got)
				}
				return
			}
			if got == nil {
				t.Fatal("skipped, want a row")
			}
			if got.Type != tc.typ {
				t.Errorf("type = %q, want %q", got.Type, tc.typ)
			}
			if tc.link != "" && got.Link != tc.link {
				t.Errorf("link = %q, want %q", got.Link, tc.link)
			}
			if tc.status != "" && got.Status != tc.status {
				t.Errorf("status = %q, want %q", got.Status, tc.status)
			}
			if tc.name == "null actor is sender 0" && got.SenderID != 0 {
				t.Errorf("sender = %d, want 0", got.SenderID)
			}
			if tc.name == "kind 8 followed needs no target" {
				if got.SenderID != 9 {
					t.Errorf("sender = %d, want 9", got.SenderID)
				}
				if got.ActorCount != 4 || got.ItemCount != 4 {
					t.Errorf("counts actor=%d item=%d", got.ActorCount, got.ItemCount)
				}
				if got.CommunityThreadID != nil || got.CommunityPostNumber != nil {
					t.Errorf("thread/post fields set: %+v", got)
				}
			}
			if strings.HasPrefix(tc.name, "kind 10") {
				if got.SenderID != 9 {
					t.Errorf("sender = %d, want 9", got.SenderID)
				}
				if got.CommunityThreadID != nil || got.CommunityPostNumber != nil {
					t.Errorf("thread/post fields set: %+v", got)
				}
				if tc.content != "" && got.Content != tc.content {
					t.Errorf("content = %q, want %q", got.Content, tc.content)
				}
				if tc.name == "kind 10 followee activity" && (got.ItemCount != 3 || got.ActorCount != 1) {
					t.Errorf("counts actor=%d item=%d", got.ActorCount, got.ItemCount)
				}
				if tc.name == "kind 10 null activity falls back" && got.Content != "" {
					t.Errorf("null activity content = %q, want empty", got.Content)
				}
			}
			if tc.name == "quiz preview empty" && got.Content != "" {
				t.Errorf("quiz content = %q, want empty", got.Content)
			}
			if strings.Contains(tc.name, "unresolved") && got.Content != "" {
				t.Errorf("unresolved content = %q, want empty", got.Content)
			}
			if got.CommunityNotificationID == nil || *got.CommunityNotificationID != tc.note.ID {
				t.Errorf("community id = %v, want %d", got.CommunityNotificationID, tc.note.ID)
			}
			if got.CreatedAt.IsZero() {
				t.Error("created is zero")
			}
		})
	}
}

func TestMapNotificationGalgameCommentLink(t *testing.T) {
	n := baseNote(communityclient.InboxMentioned)
	game := &anchor.Target{Link: "/galgame/4178", Label: "Galgame", WorkID: 4178}
	got := MapNotification(n, nil, game)
	if got == nil || got.Link != "/galgame/4178?comment=100" {
		t.Fatalf("link = %v", got)
	}
}

func noticeNote(topic, rawURL string) communityclient.NotificationView {
	return communityclient.NotificationView{
		ID: 12, UserID: 3, Kind: communityclient.InboxNotice,
		ItemCount: 2, UpdatedAt: "2026-10-11T00:00:00Z", Seq: 60,
		Notice: &communityclient.NotificationNoticeView{
			ID: 5, Topic: topic, Ref: "offer:7",
			Title: "你订阅的「DLsite 1000 円优惠券」补货了", URL: rawURL,
		},
	}
}

func TestMapNotice(t *testing.T) {
	cases := []struct {
		name string
		note communityclient.NotificationView
		link string
	}{
		{"shop url", noticeNote("shop_restock", "https://www.kungal.com/shop"), "/shop"},
		{"bare host keeps query", noticeNote("shop_restock", "https://kungal.com/shop?tab=store"), "/shop?tab=store"},
		{"foreign host falls back to the shop", noticeNote("shop_restock", "https://evil.example/shop"), "/shop"},
		{"empty url falls back to the shop", noticeNote("shop_restock", ""), "/shop"},
		{"over-long path falls back", noticeNote("shop_restock", "https://www.kungal.com/"+strings.Repeat("a", 100)), "/shop"},
		{"other topic falls back to the inbox", noticeNote("account", "https://example.com/x"), "/message/notice"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapNotification(tc.note, nil, nil)
			if got == nil {
				t.Fatal("skipped, want a row")
			}
			if got.Type != "notice" || got.Link != tc.link || got.Content != "你订阅的「DLsite 1000 円优惠券」补货了" {
				t.Fatalf("type=%q link=%q content=%q", got.Type, got.Link, got.Content)
			}
			if got.SenderID != 0 || got.ItemCount != 2 || got.Status != "unread" ||
				got.CommunityThreadID != nil || got.CommunityPostNumber != nil {
				t.Fatalf("row %+v", got)
			}
		})
	}

	read := noticeNote("shop_restock", "https://www.kungal.com/shop")
	read.ReadAt = ptrStr("2026-10-11T01:00:00Z")
	if got := MapNotification(read, nil, nil); got == nil || got.Status != "read" {
		t.Fatalf("read notice %+v", got)
	}

	bare := noticeNote("shop_restock", "")
	bare.Notice = nil
	if got := MapNotification(bare, nil, nil); got != nil {
		t.Fatalf("a kind 11 row without notice mapped to %+v", got)
	}
}

func TestMapNotificationRefs(t *testing.T) {
	top := &communityclient.PostView{ID: 100, ContentRaw: "hi"}
	target := &anchor.Target{Link: "/x"}
	on := func(kind int32, anchorKind int32, anchorID string, post *int64) communityclient.NotificationView {
		n := baseNote(kind)
		n.AnchorKind = anchorKind
		n.AnchorID = anchorID
		n.PostID = post
		return n
	}
	followed := baseNote(communityclient.InboxFollowed)
	followed.PostID = nil
	activity := baseNote(communityclient.InboxFolloweeActivity)
	activity.Activity = &communityclient.NotificationActivityView{ID: 101, Title: "a", URL: "https://www.kungal.com/topic/1"}
	noActivity := baseNote(communityclient.InboxFolloweeActivity)
	notice := baseNote(communityclient.InboxNotice)
	notice.Notice = &communityclient.NotificationNoticeView{ID: 31, Topic: "shop_restock", Title: "restock"}

	cases := []struct {
		name string
		note communityclient.NotificationView
		want string
	}{
		{"site_game post", on(communityclient.InboxReplied, communityclient.AnchorSiteGame, "4178", ptrInt64(100)), "community_post:100 work:4178"},
		{"resource post", on(communityclient.InboxLiked, communityclient.AnchorSiteResource, "resource:42", ptrInt64(100)), "community_post:100 resource:42"},
		{"rating post", on(communityclient.InboxMentioned, communityclient.AnchorSiteResource, "rating:11", ptrInt64(100)), "community_post:100 rating:11"},
		{"toolset post", on(communityclient.InboxPosted, communityclient.AnchorSiteResource, "toolset:3", ptrInt64(7)), "community_post:7 toolset:3"},
		{"website post", on(communityclient.InboxReplied, communityclient.AnchorSiteResource, "website:5", ptrInt64(100)), "community_post:100 website:5"},
		{"quiz without a post", on(communityclient.InboxPosted, communityclient.AnchorSiteResource, "quiz:4", nil), "quiz:4 quiz:4"},
		{"unknown source", on(communityclient.InboxPosted, communityclient.AnchorSiteResource, "board:4", ptrInt64(100)), "-:- -:-"},
		{"kind 8", followed, "user:9 user:9"},
		{"kind 10", activity, "activity:101 -:-"},
		{"kind 10 without activity", noActivity, "-:- -:-"},
		{"kind 11", notice, "notice:31 -:-"},
	}
	for _, c := range cases {
		msg := MapNotification(c.note, top, target)
		if msg == nil {
			t.Fatalf("%s: mapped to nil", c.name)
		}
		if got := refText(msg.Ref); got != c.want {
			t.Errorf("%s: refs %q, want %q", c.name, got, c.want)
		}
	}
}

func refText(r model.Ref) string {
	part := func(kind string, id int64) string {
		if kind == "" && id == 0 {
			return "-:-"
		}
		return kind + ":" + strconv.FormatInt(id, 10)
	}
	return part(r.SubjectKind, r.SubjectID) + " " + part(r.RootKind, r.RootID)
}
