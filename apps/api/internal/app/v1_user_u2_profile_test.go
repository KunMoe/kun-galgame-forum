package app

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestV1GetUserRoundTrip(t *testing.T) {
	f := newMeFix(t)
	f.addOAuthUser(u2UserProfiled, "profiled", 0, map[string]any{
		"roles": []string{"user", "creator", "moderator"}, "bio": "profiled-bio",
		"created_at": "2026-01-15T08:00:00Z",
	})
	id := strconv.Itoa(u2UserProfiled)
	resp, body := f.getUser(t, "", id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get user %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "user" || body["id"] != id {
		t.Fatalf("user %+v", body)
	}
	if body["name"] != "profiled" || body["bio"] != "profiled-bio" {
		t.Fatalf("name/bio %+v", body)
	}
	if body["created_at"] != "2026-01-15T08:00:00Z" {
		t.Fatalf("created_at %v", body["created_at"])
	}
	if fmt.Sprint(body["roles"]) != "[creator moderator]" {
		t.Fatalf("roles %v, want the badge roles only", body["roles"])
	}
	counts := userCounts(t, body)
	if asInt(counts["reply_count"]) != 0 || asInt(counts["topic_count"]) != 0 {
		t.Fatalf("counts %+v", counts)
	}
}

func TestV1GetUserTopicCountExcludesHidden(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.getUser(t, "sess-alice", strconv.Itoa(w3UserAlice))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get user %d %+v", resp.StatusCode, body)
	}
	if asInt(userCounts(t, body)["topic_count"]) != 6 {
		t.Fatalf("topic_count %v, want 6 visible (2 hidden excluded)", userCounts(t, body)["topic_count"])
	}
}

func TestV1GetUserToolsetCount(t *testing.T) {
	f := newMeFix(t)
	t.Cleanup(func() {
		f.db.Exec(`DELETE FROM galgame_toolset WHERE id IN (?, ?)`, u2ToolA, u2ToolB)
	})
	for _, id := range []int{u2ToolA, u2ToolB} {
		f.execSQL(t, `INSERT INTO galgame_toolset (id, name, description, status, view, type, language, platform, homepage, resource_update_time, version, user_id, created, updated)
			VALUES (?, ?, '', 0, 0, '', '', '', '[]'::jsonb, NOW(), '', ?, NOW(), NOW())`,
			id, "u2-tool-"+strconv.Itoa(id), w3UserAlice)
	}
	resp, body := f.getUser(t, "", strconv.Itoa(w3UserAlice))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get user %d %+v", resp.StatusCode, body)
	}
	if asInt(userCounts(t, body)["toolset_count"]) != 2 {
		t.Fatalf("toolset_count %v, want 2", userCounts(t, body)["toolset_count"])
	}
}

func TestV1GetUserTopicTodayIsBeijingDay(t *testing.T) {
	f := newMeFix(t)
	// The day is tomorrow in Beijing, not a fixed date: the fixture seeds
	// Alice's topics two days before the real clock, and a pinned 2026-09-24
	// counted all seven of them from 2026-09-25 16:00 UTC on.
	bj := time.FixedZone("Asia/Shanghai", 8*60*60)
	y, m, d := time.Now().In(bj).AddDate(0, 0, 1).Date()
	dayStart := time.Date(y, m, d, 0, 0, 0, 0, bj)
	f.UserService.WithClock(func() time.Time { return dayStart.Add(30 * time.Minute) })
	t.Cleanup(func() {
		f.db.Exec(`DELETE FROM topic WHERE id IN (?, ?)`, u2TopicToday, u2TopicOldDay)
	})
	f.insertTopic(t, u2TopicToday, w3UserAlice, 0, dayStart.Add(40*time.Minute))
	f.insertTopic(t, u2TopicOldDay, w3UserAlice, 0, dayStart.Add(-10*time.Minute))
	resp, body := f.getUser(t, "", strconv.Itoa(w3UserAlice))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get user %d %+v", resp.StatusCode, body)
	}
	if asInt(userCounts(t, body)["topic_today_count"]) != 1 {
		t.Fatalf("topic_today_count %v, want 1 (00:40 Beijing counts, 23:50 the day before does not, though both share a UTC day)", userCounts(t, body)["topic_today_count"])
	}
}

func TestV1GetUserCommunityCommentCountUnknownIsNull(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.getUser(t, "", strconv.Itoa(w3UserAlice))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get user %d %+v", resp.StatusCode, body)
	}
	if got := userCounts(t, body)["community_comment_count"]; got != nil {
		t.Fatalf("community_comment_count %v, want null", got)
	}
}

func TestV1GetUserBannedIs404(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.getUser(t, "", strconv.Itoa(w3UserBanned))
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1GetUserStatus2Is404(t *testing.T) {
	f := newMeFix(t)
	f.addOAuthUser(u2UserStatus2, "deleted", 2, nil)
	resp, body := f.getUser(t, "", strconv.Itoa(u2UserStatus2))
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1GetUserMissingIs404(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.getUser(t, "", strconv.Itoa(u2UserGone))
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1GetUserOAuthDownIs503(t *testing.T) {
	f := newMeFix(t)
	f.failOA.Store(true)
	f.UserClient.Invalidate(w3UserAlice)
	resp, body := f.getUser(t, "", strconv.Itoa(w3UserAlice))
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1GetUserStatsFailureIs500(t *testing.T) {
	f := newMeFix(t)
	f.deadStats(t)
	resp, body := f.getUser(t, "", strconv.Itoa(w3UserAlice))
	mustCode(t, resp, body, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func TestV1GetUserStateFailureIs500(t *testing.T) {
	f := newMeFix(t)
	f.deadState(t)
	resp, body := f.getUser(t, "", strconv.Itoa(w3UserAlice))
	mustCode(t, resp, body, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func TestV1GetUserCreatedAtFallsBackToState(t *testing.T) {
	f := newMeFix(t)
	created := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	f.addOAuthUser(u2UserFallback, "fallback", 0, map[string]any{"created_at": "not-a-date"})
	f.execSQL(t, `INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated)
		VALUES (?, 7, ?, ?) ON CONFLICT (user_id) DO UPDATE SET created = EXCLUDED.created`,
		u2UserFallback, created, created)
	resp, body := f.getUser(t, "", strconv.Itoa(u2UserFallback))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get user %d %+v", resp.StatusCode, body)
	}
	if body["created_at"] != "2026-03-01T12:00:00Z" {
		t.Fatalf("created_at %v, want state fallback", body["created_at"])
	}
}
