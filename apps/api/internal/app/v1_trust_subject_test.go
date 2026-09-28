package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/testdb"
	trustapiv1 "kun-galgame-api/internal/trust/apiv1"
	userapiv1 "kun-galgame-api/internal/user/apiv1"
	"kun-galgame-api/pkg/communityclient"
)

type tsProfiles struct{}

func (tsProfiles) ModerationProfile(_ context.Context, id int) (*userapiv1.UserProfile, bool, error) {
	if id != tsUserBanned {
		return nil, false, nil
	}
	name, bio := "banned", "bio"
	return &userapiv1.UserProfile{
		Object: "user", ID: repr.ID(id), Name: &name, Bio: &bio, Roles: []userapiv1.UserRole{},
		CreatedAt: repr.Timestamp(tsTime(1)), Moemoepoint: 7, Counts: userapiv1.UserCounts{ReplyCount: 3},
	}, false, nil
}

func TestV1ReviewItemSubject(t *testing.T) {
	f := newTrustFix(t)
	f.app.TrustV1.WithSubjects(trustapiv1.Subjects{
		"forum_topic": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			switch id {
			case 9001:
				return trustapiv1.Subject{
					Hidden: true, Title: "3 楼", Markdown: "hello **world**", Path: "/topic/7?reply=3",
					ParentTitle: "the topic", ParentPath: "/topic/7", AuthorID: tsUserBanned, CreatedAt: tsTime(2),
				}, nil
			case 9002:
				return trustapiv1.Subject{}, trustapiv1.ErrSubjectGone
			}
			return trustapiv1.Subject{}, errors.New("database down")
		},
	}, tsProfiles{}, &content.Converter{SiteBase: apiv1.SiteOrigin})

	resp, body := f.reviewItem(t, "sess-mod", "9001")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	subject, _ := body["subject"].(map[string]any)
	doc, _ := subject["content"].(map[string]any)
	blocks, _ := doc["children"].([]any)
	if subject["object"] != "review_subject" || subject["state"] != "hidden" || subject["title"] != "3 楼" ||
		subject["page_path"] != "/topic/7?reply=3" || subject["parent_title"] != "the topic" || subject["parent_path"] != "/topic/7" ||
		subject["authored_at"] == nil || len(blocks) != 1 {
		t.Errorf("a hidden subject is still shown to the reviewer: %+v", subject)
	}
	author, _ := body["subject_author"].(map[string]any)
	profile, _ := author["profile"].(map[string]any)
	if author["object"] != "review_author" || author["is_account_active"] != false ||
		profile["id"] != fmt.Sprint(tsUserBanned) || profile["name"] != "banned" {
		t.Errorf("a banned author keeps a profile for the reviewer: %+v", author)
	}

	resp, body = f.reviewItem(t, "sess-mod", "9002")
	subject, _ = body["subject"].(map[string]any)
	doc, _ = subject["content"].(map[string]any)
	blocks, _ = doc["children"].([]any)
	if resp.StatusCode != http.StatusOK || subject["state"] != "gone" || subject["title"] != "" ||
		subject["page_path"] != nil || blocks == nil || len(blocks) != 0 || body["subject_author"] != nil {
		t.Errorf("a gone subject says so and carries nothing else: %d %+v", resp.StatusCode, body)
	}

	resp, body = f.reviewItem(t, "sess-mod", "9003")
	if resp.StatusCode != http.StatusOK || body["subject"] != nil || body["subject_author"] != nil {
		t.Errorf("a subject that cannot be read leaves the item readable: %d %+v", resp.StatusCode, body)
	}
}

func TestTrustSubjectsReadHiddenContent(t *testing.T) {
	db := testdb.Open(t)
	const topicID, replyID, commentID, author = 958100001, 958100002, 958100003, 958100009
	cleanup := func() {
		_ = db.Exec(`DELETE FROM topic_comment WHERE id BETWEEN 958100001 AND 958100999`).Error
		_ = db.Exec(`DELETE FROM topic_reply WHERE id BETWEEN 958100001 AND 958100999`).Error
		_ = db.Exec(`DELETE FROM topic WHERE id BETWEEN 958100001 AND 958100999`).Error
	}
	cleanup()
	t.Cleanup(cleanup)
	at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, 'hidden topic', 'topic body', 0, 1, 'galgame', ?, ?, ?, ?, false, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, 'trust', 0)`,
			[]any{topicID, at, at, at, author}},
		{`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
			VALUES (?, 'reply body', 4, ?, ?, 1, 0, ?, ?)`, []any{replyID, author, topicID, at, at}},
		{`INSERT INTO topic_comment (id, content, topic_id, topic_reply_id, user_id, target_user_id, status, created, updated)
			VALUES (?, 'comment body', ?, ?, ?, ?, 1, ?, ?)`, []any{commentID, topicID, replyID, author, author, at, at}},
	} {
		if err := db.Exec(q.sql, q.args...).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	subjects := (&App{DB: db}).trustSubjects()
	ctx := context.Background()
	for _, c := range []struct {
		kind string
		id   int
		want trustapiv1.Subject
	}{
		{"forum_topic", topicID, trustapiv1.Subject{Hidden: true, Title: "hidden topic", Markdown: "topic body", Path: "/topic/958100001"}},
		{"forum_reply", replyID, trustapiv1.Subject{Hidden: true, Title: "4 楼", Markdown: "reply body", Path: "/topic/958100001?reply=4",
			ParentTitle: "hidden topic", ParentPath: "/topic/958100001"}},
		{"forum_comment", commentID, trustapiv1.Subject{Hidden: true, Markdown: "comment body", Path: "/topic/958100001?comment=958100003",
			ParentTitle: "hidden topic", ParentPath: "/topic/958100001"}},
	} {
		got, err := subjects[c.kind](ctx, c.id)
		if err != nil {
			t.Fatalf("%s: %v", c.kind, err)
		}
		c.want.AuthorID, c.want.CreatedAt = author, at
		if !got.CreatedAt.Equal(at) {
			t.Errorf("%s created %v", c.kind, got.CreatedAt)
		}
		got.CreatedAt = at
		if got != c.want {
			t.Errorf("%s: a hidden row is read for the reviewer\n got %+v\nwant %+v", c.kind, got, c.want)
		}
	}
	for _, kind := range []string{"forum_topic", "forum_reply", "forum_comment"} {
		if _, err := subjects[kind](ctx, 958100999); !errors.Is(err, trustapiv1.ErrSubjectGone) {
			t.Errorf("%s: a missing row is gone, got %v", kind, err)
		}
	}
}

func TestV1CreateReportNamesAuthor(t *testing.T) {
	f := newTrustFix(t)
	f.app.TrustV1.WithSubjects(trustapiv1.Subjects{
		"forum_topic": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			switch id {
			case 4121:
				return trustapiv1.Subject{Hidden: true, AuthorID: tsUserBanned}, nil
			case 4122:
				return trustapiv1.Subject{}, trustapiv1.ErrSubjectGone
			}
			return trustapiv1.Subject{}, errors.New("database down")
		},
	}, tsProfiles{}, nil)

	for _, id := range []string{"4121", "4122", "4123"} {
		resp, body := f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", nil, reportBody(map[string]any{"subject_id": id}))
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("%s: %d %+v", id, resp.StatusCode, body)
		}
	}
	got := f.trust.submissions()
	if got[0].AuthorID == nil || *got[0].AuthorID != tsUserBanned {
		t.Errorf("a report names the author the forum reads: %+v", got[0])
	}
	if got[1].AuthorID != nil || got[2].AuthorID != nil {
		t.Errorf("an author the forum cannot read is left out: %+v %+v", got[1], got[2])
	}
}

func TestV1ReviewAuthorHistory(t *testing.T) {
	f := newTrustFix(t)
	f.app.TrustV1.WithSubjects(trustapiv1.Subjects{
		"forum_topic": func(_ context.Context, id int) (trustapiv1.Subject, error) {
			if id == 9001 {
				return trustapiv1.Subject{Markdown: "x", AuthorID: tsUserBanned}, nil
			}
			return trustapiv1.Subject{}, trustapiv1.ErrSubjectGone
		},
	}, tsProfiles{}, &content.Converter{SiteBase: apiv1.SiteOrigin})
	banned := int64(tsUserBanned)
	f.trust.mu.Lock()
	// 9010 is actioned, 9011 and 9012 dismissed, 9014 dismissed on another site.
	for _, id := range []int64{9001, 9010, 9011, 9012, 9014} {
		f.trust.find(id).SubjectAuthorID = &banned
	}
	f.trust.mu.Unlock()

	resp, body := f.reviewItem(t, "sess-mod", "9001")
	author, _ := body["subject_author"].(map[string]any)
	if resp.StatusCode != http.StatusOK || asInt(author["past_actioned_count"]) != 1 || asInt(author["past_dismissed_count"]) != 2 {
		t.Errorf("an open item counts every decided item of its author on this site: %d %+v", resp.StatusCode, author)
	}
	for _, c := range f.trust.callsMatching("GET /api/v1/admin/trust/review-items?") {
		if !strings.Contains(c, "site=kungal") || !strings.Contains(c, "subject_author_id=940000003") {
			t.Errorf("history asks for this author on this site: %s", c)
		}
	}

	resp, body = f.reviewItem(t, "sess-mod", "9011")
	subject, _ := body["subject"].(map[string]any)
	author, _ = body["subject_author"].(map[string]any)
	profile, _ := author["profile"].(map[string]any)
	if resp.StatusCode != http.StatusOK || subject["state"] != "gone" || profile["id"] != fmt.Sprint(tsUserBanned) ||
		asInt(author["past_actioned_count"]) != 1 || asInt(author["past_dismissed_count"]) != 1 {
		t.Errorf("a gone subject keeps the author the trust service recorded, and a decided item is not its own past: %d %+v", resp.StatusCode, body)
	}

	resp, body = f.inbox(t, "sess-mod", url.Values{"author_id": {fmt.Sprint(tsUserBanned)}})
	if resp.StatusCode != http.StatusOK || fmt.Sprint(adminItemIDs(body)) != "[9001 9011 9010 9012]" || asInt(body["total"]) != 4 {
		t.Errorf("the inbox narrows to one author on this site: %d %v %+v", resp.StatusCode, adminItemIDs(body), body["total"])
	}
	resp, body = f.inbox(t, "sess-mod", url.Values{"author_id": {"0"}})
	wantProblem(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
}

func TestTrustSubjectsReadModeratedCommunityPosts(t *testing.T) {
	db := testdb.Open(t)
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		var req struct {
			IDs []int64 `json:"ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		posts := map[int64]map[string]any{
			11: {"id": 11, "author_id": 55, "content_raw": "hidden text", "status": 1, "created_at": "2026-09-01T12:00:00Z"},
			12: {"id": 12, "author_id": 55, "content_raw": "deleted text", "status": 2},
			13: {"id": 13, "author_id": 55, "content_raw": "", "status": 2},
		}
		out := []any{}
		for _, id := range req.IDs {
			if p, ok := posts[id]; ok {
				out = append(out, map[string]any{"post": p, "thread": map[string]any{"thread_id": 1, "anchor_kind": communityclient.AnchorSiteGame, "anchor_id": "42"}})
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"posts": out}})
	}))
	defer srv.Close()

	resolve := (&App{DB: db, Community: communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "c", ClientSecret: "s"})}).
		trustSubjects()["community_post"]
	ctx := context.Background()
	hidden, err := resolve(ctx, 11)
	if err != nil || !hidden.Hidden || hidden.Deleted || hidden.Markdown != "hidden text" || hidden.AuthorID != 55 ||
		hidden.Path != "/galgame/42?comment=11" || !hidden.CreatedAt.Equal(time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("a hidden post is read for the reviewer: %+v %v", hidden, err)
	}
	deleted, err := resolve(ctx, 12)
	if err != nil || !deleted.Deleted || deleted.Markdown != "deleted text" {
		t.Errorf("a deleted post keeps what it said: %+v %v", deleted, err)
	}
	for _, id := range []int{13, 14} {
		if _, err := resolve(ctx, id); !errors.Is(err, trustapiv1.ErrSubjectGone) {
			t.Errorf("post %d: a purged or missing post is gone, got %v", id, err)
		}
	}
	for _, p := range paths {
		if p != "/moderation/posts/resolve" {
			t.Errorf("the reviewer's read uses the moderation resolve, got %s", p)
		}
	}
}
