package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
)

const (
	acTopicLogin   = 950000105
	acReplyLogin   = 950000205
	acCommentLogin = 950000302
	acUpvoteLogin  = 950000402
	acMsgLogin     = 950000503
)

type feedKey struct {
	typ    string
	source int
}

var loginTopicRows = []feedKey{
	{"TOPIC_CREATION", acTopicLogin},
	{"TOPIC_REPLY_CREATION", acReplyLogin},
	{"TOPIC_COMMENT_CREATION", acCommentLogin},
	{"TOPIC_UPVOTE", acUpvoteLogin},
	{"MESSAGE_SOLUTION", acMsgLogin},
}

func (f *activityFix) seedLoginTopic(t *testing.T) {
	t.Helper()
	at := acTie.Add(8 * time.Hour)
	f.run(t, `INSERT INTO topic (
		id, title, content, view, status, category, status_update_time, created, updated,
		user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
		favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
	) VALUES (?, 'members', 'members body', 0, 0, 'galgame', ?, ?, ?, ?, false, 'login', '', 0, 0, 1, 1, 0, 1, 0, 0, '', 0)`,
		acTopicLogin, at, at, at, acUserAlice)
	f.run(t, `INSERT INTO topic_section_relation (topic_id, topic_section_id, created, updated) VALUES (?, ?, ?, ?)`,
		acTopicLogin, sectionID(t, f.db, "galgame-discussion"), at, at)
	f.run(t, `INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'members reply', 1, ?, ?, 0, 0, ?, ?)`, acReplyLogin, acUserBob, acTopicLogin, at, at)
	f.run(t, `INSERT INTO topic_comment (id, topic_id, topic_reply_id, user_id, target_user_id, content, status, created, updated)
		VALUES (?, ?, ?, ?, ?, 'members comment', 0, ?, ?)`, acCommentLogin, acTopicLogin, acReplyLogin, acUserAlice, acUserBob, at, at)
	f.run(t, `INSERT INTO topic_upvote (id, topic_id, user_id, description, created, updated) VALUES (?, ?, ?, 'members push', ?, ?)`,
		acUpvoteLogin, acTopicLogin, acUserBob, at, at)
	f.run(t, `UPDATE topic SET best_answer_id = ? WHERE id = ?`, acReplyLogin, acTopicLogin)
	f.run(t, `INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, is_nsfw, created, access_scope)
		VALUES ('MESSAGE_SOLUTION', ?, ?, 0, 'members reply', ?, false, ?, 'login')`,
		acMsgLogin, acUserAlice, fmt.Sprintf("/topic/%d?reply=1", acTopicLogin), at)
}

func (f *activityFix) signIn(t *testing.T, userID int) {
	t.Helper()
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: userID, Name: "bob", Roles: []string{}},
		OAuthAccessToken: "access",
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.app.Redis.Set(context.Background(), middleware.SessionKey("ac-session"), data, middleware.SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
	f.session = "ac-session"
}

func (f *activityFix) feedScopes(t *testing.T, keys []feedKey) map[feedKey]string {
	t.Helper()
	out := map[feedKey]string{}
	for _, k := range keys {
		var scopes []string
		if err := f.db.Raw(`SELECT access_scope FROM feed_activity WHERE type = ? AND source_id = ?`, k.typ, k.source).
			Scan(&scopes).Error; err != nil {
			t.Fatal(err)
		}
		if len(scopes) == 1 {
			out[k] = scopes[0]
		}
	}
	return out
}

func (f *activityFix) wantScopes(t *testing.T, keys []feedKey, want string) {
	t.Helper()
	got := f.feedScopes(t, keys)
	for _, k := range keys {
		if got[k] != want {
			t.Errorf("%s/%d scope = %q, want %q", k.typ, k.source, got[k], want)
		}
	}
}

func (f *activityFix) setScope(t *testing.T, scope string) {
	t.Helper()
	f.run(t, `UPDATE topic SET access_scope = ? WHERE id = ?`, scope, acTopicLogin)
}

func (f *activityFix) loginRowIDs(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, k := range loginTopicRows {
		out[f.rowID(t, k.typ, k.source)] = true
	}
	return out
}

func TestV1ActivityLoginScopeTriggers(t *testing.T) {
	f := newActivityFix(t)
	f.seedLoginTopic(t)
	f.wantScopes(t, loginTopicRows, "login")

	f.setScope(t, "public")
	f.wantScopes(t, loginTopicRows, "public")

	f.setScope(t, "login")
	f.wantScopes(t, loginTopicRows, "login")

	f.setScope(t, "role")
	if got := f.feedScopes(t, loginTopicRows); len(got) != 0 {
		t.Fatalf("role-scoped topic kept feed rows: %v", got)
	}

	// The MESSAGE_SOLUTION row was seeded without a message behind it, so the
	// rebuild from live tables cannot bring it back.
	f.setScope(t, "login")
	f.wantScopes(t, loginTopicRows[:4], "login")

	f.run(t, `INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'late reply', 2, ?, ?, 0, 0, ?, ?)`, acReplyLogin+1, acUserAlice, acTopicLogin, acTie, acTie)
	f.wantScopes(t, []feedKey{{"TOPIC_REPLY_CREATION", acReplyLogin + 1}}, "login")

	f.run(t, `UPDATE topic SET status = 1 WHERE id = ?`, acTopicLogin)
	if got := f.feedScopes(t, loginTopicRows[:4]); len(got) != 0 {
		t.Fatalf("hidden topic kept feed rows: %v", got)
	}
}

func TestV1ActivityLoginScopeReads(t *testing.T) {
	f := newActivityFix(t)
	f.seedLoginTopic(t)
	login := f.loginRowIDs(t)
	present := func(items []map[string]any) int {
		return len(activityIDs(items, login))
	}

	if n := present(f.walk(t, url.Values{}, 7)); n != 0 {
		t.Fatalf("anonymous stream shows %d login-topic activities", n)
	}
	topicTab := func() url.Values { return url.Values{"activity_types": {"topic_creation"}, "sort": {"bumped_desc"}} }
	if n := present(f.walk(t, topicTab(), 3)); n != 0 {
		t.Fatal("anonymous topic tab shows the login topic")
	}

	f.signIn(t, acUserBob)
	if n := present(f.walk(t, url.Values{}, 7)); n != len(loginTopicRows) {
		t.Fatalf("signed-in stream shows %d of %d login-topic activities", n, len(loginTopicRows))
	}
	if n := present(f.walk(t, topicTab(), 3)); n != 1 {
		t.Fatalf("signed-in topic tab shows the login topic %d times", n)
	}
}
