package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/testdb"
	trustapiv1 "kun-galgame-api/internal/trust/apiv1"
	userapiv1 "kun-galgame-api/internal/user/apiv1"
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
