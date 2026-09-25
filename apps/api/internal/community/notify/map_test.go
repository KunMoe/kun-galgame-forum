package notify

import (
	"strings"
	"testing"

	"kun-galgame-api/internal/community/anchor"
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
