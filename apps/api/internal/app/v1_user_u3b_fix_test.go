package app

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/client"
)

const (
	u3bUserMin        = 971000001
	u3bUserMax        = 971000099
	u3bOwner          = 971000001
	u3bBannedUploader = 971000002
	u3bStatus2        = 971000003
	u3bFriend         = 971000005
	u3bGone           = 971000099

	u3bWorkMin     = 971000201
	u3bWorkMax     = 971000299
	u3bWorkTieMin  = 971000211
	u3bWorkTieMax  = 971000215
	u3bWorkUnpub   = 971000221
	u3bWorkNSFW    = 971000222
	u3bWorkNoLocal = 971000223
	u3bWorkPeer    = 971000224
	u3bWorkGhostNS = 971000225

	u3bResMin     = 971000301
	u3bResMax     = 971000399
	u3bResTieMin  = 971000301
	u3bResTieMax  = 971000305
	u3bResExpired = 971000310
	u3bResUnpub   = 971000311
	u3bResNSFW    = 971000312
	u3bResLiked   = 971000313
	u3bResBanned  = 971000314
	u3bResSecret  = 971000315

	u3bSecretURL  = "https://example.invalid/u3b-secret-dl"
	u3bSecretCode = "u3b-extract-AAAA"
	u3bSecretPass = "u3b-archive-BBBB"
)

type u3bFix struct {
	*resourceFix
	contribFail atomic.Bool
	contribIDs  []int
}

func newU3bFix(t *testing.T) *u3bFix {
	t.Helper()
	rf := newResourceFix(t, nil)
	f := &u3bFix{resourceFix: rf}
	f.contribIDs = []int{u3bWorkNSFW, u3bWorkTieMax, u3bWorkGhostNS, u3bWorkNoLocal, u3bWorkTieMin}
	f.ContributedWorkIDs = func(_ context.Context, uid int64) ([]int, error) {
		if f.contribFail.Load() {
			return nil, fmt.Errorf("catalog down")
		}
		if uid != int64(u3bOwner) {
			return []int{}, nil
		}
		out := make([]int, len(f.contribIDs))
		copy(out, f.contribIDs)
		return out, nil
	}
	f.Fiber = newFiber()
	f.setupRoutes()
	f.spec = newSpecConformance(t)
	f.addOAuthUser(u3bOwner, "u3b-owner", 0, nil)
	f.addOAuthUser(u3bBannedUploader, "u3b-banned", 1, nil)
	f.addOAuthUser(u3bFriend, "u3b-friend", 0, nil)
	f.addOAuthUser(u3bStatus2, "u3b-status2", 2, nil)
	f.putSession(t, "sess-u3b-owner", u3bOwner)
	f.cleanupU3b(t)
	t.Cleanup(func() { f.cleanupU3b(t) })
	f.seedU3b(t)
	return f
}

func (f *u3bFix) cleanupU3b(t *testing.T) {
	t.Helper()
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Errorf("u3b cleanup: %v\n%s", err, q)
		}
	}
	run(`DELETE FROM feed_activity WHERE user_id BETWEEN ? AND ? OR work_id BETWEEN ? AND ? OR source_id BETWEEN ? AND ?`,
		u3bUserMin, u3bUserMax, u3bWorkMin, u3bWorkMax, u3bWorkMin, u3bWorkMax)
	run(`DELETE FROM galgame_resource_like WHERE galgame_resource_id BETWEEN ? AND ? OR user_id BETWEEN ? AND ?`,
		u3bResMin, u3bResMax, u3bUserMin, u3bUserMax)
	run(`DELETE FROM galgame_resource_link WHERE galgame_resource_id BETWEEN ? AND ?`, u3bResMin, u3bResMax)
	run(`DELETE FROM galgame_resource WHERE id BETWEEN ? AND ? OR work_id BETWEEN ? AND ?`, u3bResMin, u3bResMax, u3bWorkMin, u3bWorkMax)
	run(`DELETE FROM galgame_like WHERE work_id BETWEEN ? AND ? OR user_id BETWEEN ? AND ?`, u3bWorkMin, u3bWorkMax, u3bUserMin, u3bUserMax)
	run(`DELETE FROM galgame WHERE id BETWEEN ? AND ?`, u3bWorkMin, u3bWorkMax)
	run(`DELETE FROM kungal_user_state WHERE user_id BETWEEN ? AND ?`, u3bUserMin, u3bUserMax)
}

func (f *u3bFix) addCatalogWork(t *testing.T, id int, name, limit string) {
	t.Helper()
	rating := "all_ages"
	if limit == "nsfw" {
		rating = "r18"
	}
	if limit == "" {
		limit = "sfw"
	}
	w := geWork{id: id, name: name, limit: limit, rating: rating}
	var row client.CatalogWorkListItem
	decodeInto(t, geRowJSON(w), &row)
	f.cat.mu.Lock()
	f.cat.rows[id] = row
	f.cat.works = append(f.cat.works, w)
	f.cat.mu.Unlock()
}

func (f *u3bFix) seedU3b(t *testing.T) {
	t.Helper()
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("u3b seed: %v\n%s", err, q)
		}
	}
	tie := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	older := tie.Add(-time.Hour)

	insWork := func(id int, creator int, published bool, limit *string, created time.Time) {
		t.Helper()
		run(`INSERT INTO galgame (id, view, created, updated, published, content_limit, resource_update_time, resource_count, resource_publish_banned, creator_user_id)
			VALUES (?, 0, ?, ?, ?, ?, ?, 0, false, ?)`,
			id, created, created, published, limit, created, creator)
	}
	sfw := "sfw"
	nsfw := "nsfw"
	for id := u3bWorkTieMin; id <= u3bWorkTieMax; id++ {
		lim := &sfw
		if id == u3bWorkTieMin {
			lim = nil
		}
		insWork(id, u3bOwner, true, lim, tie)
		f.addCatalogWork(t, id, "U3bTie"+strconv.Itoa(id), "sfw")
	}
	insWork(u3bWorkUnpub, u3bOwner, false, &sfw, older)
	f.addCatalogWork(t, u3bWorkUnpub, "U3bUnpub", "sfw")
	insWork(u3bWorkNSFW, u3bOwner, true, &nsfw, older)
	f.addCatalogWork(t, u3bWorkNSFW, "U3bNSFW", "nsfw")
	f.addCatalogWork(t, u3bWorkNoLocal, "U3bGhost", "sfw")
	f.addCatalogWork(t, u3bWorkGhostNS, "U3bGhostNSFW", "nsfw")
	insWork(u3bWorkPeer, u3bFriend, true, &sfw, older)
	f.addCatalogWork(t, u3bWorkPeer, "U3bPeer", "sfw")

	for _, id := range []int{u3bWorkTieMin, u3bWorkTieMin + 2, u3bWorkTieMax, u3bWorkUnpub, u3bWorkNSFW, u3bWorkPeer} {
		run(`INSERT INTO galgame_like (work_id, user_id, created, updated) VALUES (?, ?, ?, ?)`, id, u3bOwner, tie, tie)
	}

	insRes := func(id, work, user, status int, code, pass, note string, created time.Time) {
		t.Helper()
		run(`INSERT INTO galgame_resource (id, type, language, platform, title, version_label, languages, platforms, runtimes, size, code, password, note, status, work_id, user_id, like_count, comment_count, view, download, provider_name, created, updated)
			VALUES (?, 'game', 'zh-cn', 'windows', '', '', '["zh-cn"]'::jsonb, '["win"]'::jsonb, '["native-win"]'::jsonb, '1 GB', ?, ?, ?, ?, ?, ?, 0, 0, 0, 0, '[]'::jsonb, ?, ?)`,
			id, code, pass, note, status, work, user, created, created)
	}
	for id := u3bResTieMin; id <= u3bResTieMax; id++ {
		insRes(id, u3bWorkTieMax, u3bOwner, 0, "", "", "tie-"+strconv.Itoa(id), tie)
	}
	insRes(u3bResExpired, u3bWorkTieMax, u3bOwner, 1, "", "", "expired pack", older)
	insRes(u3bResUnpub, u3bWorkUnpub, u3bOwner, 0, "", "", "on unpublished", older)
	insRes(u3bResNSFW, u3bWorkNSFW, u3bOwner, 0, "", "", "nsfw pack", older)
	insRes(u3bResLiked, u3bWorkPeer, u3bFriend, 0, "", "", "friend pack", older)
	insRes(u3bResBanned, u3bWorkPeer, u3bBannedUploader, 0, "", "", "banned pack", older)
	insRes(u3bResSecret, u3bWorkTieMax, u3bOwner, 0, u3bSecretCode, u3bSecretPass, "secret pack", older)
	run(`INSERT INTO galgame_resource_link (url, galgame_resource_id, created, updated) VALUES (?, ?, ?, ?)`,
		u3bSecretURL, u3bResSecret, older, older)
	run(`INSERT INTO galgame_resource_like (galgame_resource_id, user_id, created, updated) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
		u3bResLiked, u3bOwner, older, older, u3bResBanned, u3bOwner, older, older)
}

func (f *u3bFix) u3bList(t *testing.T, session, collection, query string) (*http.Response, map[string]any) {
	t.Helper()
	return f.rs(t, http.MethodGet, "/api/v1/users/"+strconv.Itoa(u3bOwner)+"/"+collection+"?"+query,
		"/users/{user_id}/"+collection, session, "", nil)
}

func (f *u3bFix) u3bListOK(t *testing.T, collection, relation, extra string) map[string]any {
	t.Helper()
	resp, body := f.u3bList(t, "", collection, "relation="+relation+"&limit=100"+extra)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s %s: %d %+v", collection, relation, resp.StatusCode, body)
	}
	return body
}

func (f *u3bFix) walkU3b(t *testing.T, collection, relation, extra string) ([]string, int) {
	t.Helper()
	var ids []string
	total := -1
	for page := 1; page <= 20; page++ {
		q := fmt.Sprintf("relation=%s&page=%d&limit=2%s", relation, page, extra)
		resp, body := f.u3bList(t, "", collection, q)
		if resp.StatusCode != http.StatusOK || body["object"] != "list" {
			t.Fatalf("%s %s page %d: %d %+v", collection, relation, page, resp.StatusCode, body)
		}
		if body["total_relation"] != "eq" {
			t.Fatalf("%s page %d total_relation %v", collection, page, body["total_relation"])
		}
		n := asInt(body["total"])
		if total < 0 {
			total = n
		} else if n != total {
			t.Fatalf("%s page %d total %d, want %d", collection, page, n, total)
		}
		got := itemIDs(t, body)
		if len(got) == 0 {
			break
		}
		ids = append(ids, got...)
	}
	return ids, total
}
