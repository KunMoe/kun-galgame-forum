package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"
)

const (
	u2UserGone     = 930000888
	u2UserStatus2  = 930000810
	u2UserFallback = 930000811
	u2TopicToday   = 930000210
	u2TopicOldDay  = 930000211
	u2ToolA        = 930000701
	u2ToolB        = 930000702
	u2MsgFavorite  = 930000920
	u2UserProfiled = 930000812
)

func (f *meFix) getUser(t *testing.T, session, id string) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodGet, "/api/v1/users/"+id, "/users/{user_id}", session, "", nil, nil)
}

func (f *meFix) listUsersQuery(t *testing.T, session, query string) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodGet, "/api/v1/users?"+query, "/users", session, "", nil, nil)
}

func (f *meFix) getMuted(t *testing.T, session string) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodGet, mePath+"/notification-preferences", "/me/notification-preferences", session, "", nil, nil)
}

func (f *meFix) putMuted(t *testing.T, session string, types []string) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodPut, mePath+"/notification-preferences", "/me/notification-preferences", session, "", nil,
		map[string]any{"muted_types": types})
}

func (f *meFix) execSQL(t *testing.T, q string, args ...any) {
	t.Helper()
	if err := f.db.Exec(q, args...).Error; err != nil {
		t.Fatalf("sql: %v\n%s", err, q)
	}
}

func (f *meFix) insertTopic(t *testing.T, id, user, status int, created time.Time) {
	t.Helper()
	f.execSQL(t, `INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, ?, ?, 0, ?, 'galgame', ?, ?, ?, ?, false, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, '', 0)`,
		id, "u2-"+strconv.Itoa(id), "body", status, created, created, created, user)
}

func (f *meFix) mutedStored(t *testing.T, userID int) []string {
	t.Helper()
	var raw string
	if err := f.db.Raw(`SELECT muted_notification_types::text FROM kungal_user_state WHERE user_id = ?`, userID).Scan(&raw).Error; err != nil {
		t.Fatalf("muted stored: %v", err)
	}
	var keys []string
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		t.Fatalf("muted json %q: %v", raw, err)
	}
	return keys
}

func userCounts(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	c, ok := body["counts"].(map[string]any)
	if !ok {
		t.Fatalf("counts %v", body["counts"])
	}
	return c
}
