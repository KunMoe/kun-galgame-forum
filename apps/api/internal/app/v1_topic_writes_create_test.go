package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
)

func TestV1CreateTopicSuccess(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	h1 := w3CoverHash
	body := map[string]any{
		"title":            "  hello world  ",
		"content_markdown": "see /image/" + h1 + " and more",
		"category":         "galgame",
		"sections":         []string{"g-news", "g-walkthrough"},
		"is_nsfw":          true,
		"access_scope":     "public",
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(1), nil, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d %s", resp.StatusCode, raw)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	id := strID(out["id"])
	if resp.Header.Get("Location") != "/api/v1/topics/"+id {
		t.Fatalf("Location %q", resp.Header.Get("Location"))
	}
	if out["title"] != "hello world" {
		t.Fatalf("title %v", out["title"])
	}
	secs, _ := out["sections"].([]any)
	if len(secs) != 2 || secs[0] != "g-news" || secs[1] != "g-walkthrough" {
		t.Fatalf("sections %v", out["sections"])
	}
	covers, _ := out["cover_images"].([]any)
	if len(covers) != 1 {
		t.Fatalf("derived covers %v", out["cover_images"])
	}
	ch, _ := covers[0].(map[string]any)
	if ch["hash"] != h1 {
		t.Fatalf("cover hash %v", ch["hash"])
	}
	if out["access_scope"] != "public" {
		t.Fatalf("scope %v", out["access_scope"])
	}

	var title, content, scope, cat string
	var nsfw bool
	var coverStored string
	if err := f.db.Raw(`SELECT title, content, access_scope, category, is_nsfw, cover_images FROM topic WHERE id = ?`, id).
		Row().Scan(&title, &content, &scope, &cat, &nsfw, &coverStored); err != nil {
		t.Fatal(err)
	}
	if title != "hello world" || scope != "public" || cat != "galgame" || !nsfw {
		t.Fatalf("row %q %q %q %v", title, scope, cat, nsfw)
	}
	if want := `["/image/` + h1 + `"]`; coverStored != want {
		t.Fatalf("stored covers %q, want %q", coverStored, want)
	}
	var nRel int64
	f.db.Raw(`SELECT COUNT(*) FROM topic_section_relation WHERE topic_id = ?`, id).Scan(&nRel)
	if nRel != 2 {
		t.Fatalf("section rels %d", nRel)
	}
	aw := f.snapshotAwards()
	if len(aw) != 1 || aw[0].userID != w3UserAlice || aw[0].delta != constants.RewardCreateTopic ||
		aw[0].reason != moemoepoint.ReasonContentApproved || aw[0].ref != moemoepoint.Ref("topic", asInt(out["id"])) ||
		aw[0].key != moemoepoint.Key("topic_created", "topic_"+id) {
		t.Fatalf("award %+v", aw)
	}
}

func TestV1CreateTopicCoversExactAndNone(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	h := w3CoverAlt
	mk := func(covers any, key int) map[string]any {
		return map[string]any{
			"title": "c", "content_markdown": "see /image/" + w3CoverHash,
			"category": "galgame", "sections": []string{"g-news"}, "is_nsfw": false,
			"access_scope": "public", "cover_image_hashes": covers,
		}
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(2), nil, mk([]string{h}, 2))
	if resp.StatusCode != 201 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	covers, _ := out["cover_images"].([]any)
	if len(covers) != 1 || covers[0].(map[string]any)["hash"] != h {
		t.Fatalf("exact covers %v", out["cover_images"])
	}
	resp, raw = f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(3), nil, mk([]any{}, 3))
	if resp.StatusCode != 201 {
		t.Fatalf("empty %d %s", resp.StatusCode, raw)
	}
	_ = json.Unmarshal(raw, &out)
	covers, _ = out["cover_images"].([]any)
	if len(covers) != 0 {
		t.Fatalf("none %v", out["cover_images"])
	}
}

func TestV1CreateTopicPaidAward(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	body := map[string]any{
		"title": "paid", "content_markdown": "x", "category": "galgame",
		"sections": []string{"g-seeking"}, "is_nsfw": false, "access_scope": "public",
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(4), nil, body)
	if resp.StatusCode != 201 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	id := strID(out["id"])
	aw := f.snapshotAwards()
	if len(aw) != 1 || aw[0].delta != -constants.CostConsumeSection ||
		aw[0].reason != moemoepoint.ReasonContentRemoved ||
		aw[0].key != moemoepoint.Key("topic_created", "topic_"+id) {
		t.Fatalf("paid award %+v", aw)
	}
}

func TestV1CreateTopicMentionsCap(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	ids := make([]int, 0, 12)
	for i := w3MentionMin; i <= w3MentionMax; i++ {
		ids = append(ids, i)
	}
	body := map[string]any{
		"title": "m", "content_markdown": mentionBody(ids...),
		"category": "galgame", "sections": []string{"g-news"}, "is_nsfw": false, "access_scope": "public",
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(5), nil, body)
	if resp.StatusCode != 201 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	tid := asInt(out["id"])
	var recs []struct {
		Receiver int    `gorm:"column:receiver_id"`
		Link     string `gorm:"column:link"`
	}
	if err := f.db.Raw(`SELECT receiver_id, link FROM message WHERE sender_id = ? AND type = 'mentioned' AND link = ? ORDER BY id`,
		w3UserAlice, fmt.Sprintf("/topic/%d", tid)).Scan(&recs).Error; err != nil {
		t.Fatal(err)
	}
	if len(recs) != 10 {
		t.Fatalf("mentions %d %+v", len(recs), recs)
	}
	for i, r := range recs {
		if r.Receiver != w3MentionMin+i {
			t.Fatalf("order %d got %d", i, r.Receiver)
		}
	}
	want := make([]string, len(recs))
	for i := range want {
		want[i] = notifRef("mentioned", "topic", tid, "topic", tid)
	}
	wantNotifRefs(t, f.db, want, "sender_id = ? AND type = 'mentioned'", w3UserAlice)
}

func TestV1CreateTopicIdempotentReplay(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	body := map[string]any{
		"title": "idem", "content_markdown": "x", "category": "galgame",
		"sections": []string{"g-news"}, "is_nsfw": false, "access_scope": "public",
	}
	key := keyUUID(6)
	resp1, raw1 := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", key, nil, body)
	resp2, raw2 := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", key, nil, body)
	if resp1.StatusCode != 201 || resp2.StatusCode != 201 {
		t.Fatalf("%d %d", resp1.StatusCode, resp2.StatusCode)
	}
	var a, b map[string]any
	_ = json.Unmarshal(raw1, &a)
	_ = json.Unmarshal(raw2, &b)
	if a["id"] != b["id"] {
		t.Fatalf("ids %v %v", a["id"], b["id"])
	}
	var n int64
	f.db.Raw(`SELECT COUNT(*) FROM topic WHERE user_id = ? AND title = 'idem'`, w3UserAlice).Scan(&n)
	if n != 1 {
		t.Fatalf("rows %d", n)
	}
}

func TestV1CreateTopicMissingIdempotencyKey(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", "", nil, map[string]any{
		"title": "x", "content_markdown": "y", "category": "galgame",
		"sections": []string{"g-news"}, "is_nsfw": false, "access_scope": "public",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	m := problemMap(t, raw)
	if m["code"] != problem.CodeInvalidParameter {
		t.Fatalf("code %v", m["code"])
	}
	errs := problemErrors(t, raw)
	if len(errs) == 0 || fmt.Sprint(errs[0]["header"]) != "Idempotency-Key" {
		t.Fatalf("errors %+v", errs)
	}
}

func TestV1CreateTopicUsersScopeDropsAuthor(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	body := map[string]any{
		"title": "u", "content_markdown": "x", "category": "galgame",
		"sections": []string{"g-news"}, "is_nsfw": false, "access_scope": "users",
		"access_user_ids": []string{fmt.Sprint(w3UserAlice), fmt.Sprint(w3UserBob), fmt.Sprint(w3UserGrant)},
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(7), nil, body)
	if resp.StatusCode != 201 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	var vals []string
	f.db.Raw(`SELECT subject_value FROM topic_access_grant WHERE topic_id = ? ORDER BY ctid`, asInt(out["id"])).Scan(&vals)
	if len(vals) != 2 || vals[0] != fmt.Sprint(w3UserBob) || vals[1] != fmt.Sprint(w3UserGrant) {
		t.Fatalf("grants %v", vals)
	}
}
