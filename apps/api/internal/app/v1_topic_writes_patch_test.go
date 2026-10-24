package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
)

func TestV1PatchTopicKeepsAbsentFields(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	path := fmt.Sprintf("/api/v1/topics/%d", w3TopicNSFW)
	resp, raw := f.doJSON(t, http.MethodPatch, path, "sess-alice", "/topics/{topic_id}", "", nil, map[string]any{
		"title": "new-title",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	if out["title"] != "new-title" || out["is_nsfw"] != true {
		t.Fatalf("body %v", out)
	}
	covers, _ := out["cover_images"].([]any)
	if len(covers) != 1 || covers[0].(map[string]any)["hash"] != w3CoverHash {
		t.Fatalf("covers %v", out["cover_images"])
	}
}

func TestV1PatchTopicEditedAndBumpCutoff(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	var oldBump, oldEdited *time.Time
	var oldUpdated time.Time
	if err := f.db.Raw(`SELECT status_update_time, edited, updated FROM topic WHERE id = ?`, w3TopicOld).
		Row().Scan(&oldBump, &oldEdited, &oldUpdated); err != nil {
		t.Fatal(err)
	}
	resp, raw := f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicOld), "sess-alice", "/topics/{topic_id}", "", nil, map[string]any{
		"title": "old-2",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var bump time.Time
	var edited *time.Time
	if err := f.db.Raw(`SELECT status_update_time, edited FROM topic WHERE id = ?`, w3TopicOld).
		Row().Scan(&bump, &edited); err != nil {
		t.Fatal(err)
	}
	if !bump.Equal(*oldBump) {
		t.Fatalf("old topic bumped %v -> %v", oldBump, bump)
	}
	if edited == nil {
		t.Fatal("edited not set")
	}

	var pubBump time.Time
	f.db.Raw(`SELECT status_update_time FROM topic WHERE id = ?`, w3TopicPub).Scan(&pubBump)
	resp, raw = f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicPub), "sess-alice", "/topics/{topic_id}", "", nil, map[string]any{
		"content_markdown": "changed-body",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var pubBump2 time.Time
	f.db.Raw(`SELECT status_update_time FROM topic WHERE id = ?`, w3TopicPub).Scan(&pubBump2)
	if !pubBump2.After(pubBump) {
		t.Fatalf("recent topic not bumped")
	}
}

func TestV1PatchTopicNoop(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	var updated, edited *time.Time
	var updatedAt time.Time
	if err := f.db.Raw(`SELECT updated, edited FROM topic WHERE id = ?`, w3TopicPub).Row().Scan(&updatedAt, &edited); err != nil {
		t.Fatal(err)
	}
	resp, raw := f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicPub), "sess-alice", "/topics/{topic_id}", "", nil, map[string]any{})
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var updated2 time.Time
	var edited2 *time.Time
	if err := f.db.Raw(`SELECT updated, edited FROM topic WHERE id = ?`, w3TopicPub).Row().Scan(&updated2, &edited2); err != nil {
		t.Fatal(err)
	}
	if !updated2.Equal(updatedAt) {
		t.Fatalf("updated changed %v %v", updatedAt, updated2)
	}
	_ = updated
}

func TestV1PatchTopicSectionDecidesCategory(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	path := fmt.Sprintf("/api/v1/topics/%d", w3TopicPub)
	patch := func(body map[string]any) (int, map[string]any) {
		t.Helper()
		resp, raw := f.doJSON(t, http.MethodPatch, path, "sess-alice", "/topics/{topic_id}", "", nil, body)
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		return resp.StatusCode, out
	}
	category := func() string {
		var cat string
		f.db.Raw(`SELECT category FROM topic WHERE id = ?`, w3TopicPub).Scan(&cat)
		return cat
	}
	if code, out := patch(map[string]any{"category": "technique"}); code != 200 || out["category"] != "galgame" || out["section"] != "news" || category() != "galgame" {
		t.Fatalf("category alone %d %v, stored %s", code, out, category())
	}
	if code, out := patch(map[string]any{"category": "others", "sections": []string{"t-web"}}); code != 200 ||
		out["section"] != "development" || fmt.Sprint(out["sections"]) != "[development]" || out["category"] != "technique" || category() != "technique" {
		t.Fatalf("retired key %d %v, stored %s", code, out, category())
	}
	resp, raw := f.doJSON(t, http.MethodPatch, path, "sess-alice", "/topics/{topic_id}", "", nil, map[string]any{
		"sections": []string{"resource-release"},
	})
	if resp.StatusCode != 422 {
		t.Fatalf("archived %d %s", resp.StatusCode, raw)
	}
	errs := problemErrors(t, raw)
	if len(errs) == 0 || fmt.Sprint(errs[0]["pointer"]) != "/sections/0" || fmt.Sprint(errs[0]["reason"]) != problem.ReasonNotAllowedValue {
		t.Fatalf("%v", errs)
	}
	if err := f.db.Exec(`UPDATE topic_section_relation SET topic_section_id = ? WHERE topic_id = ?`,
		sectionID(t, f.db, "resource-release"), w3TopicPub).Error; err != nil {
		t.Fatal(err)
	}
	if code, out := patch(map[string]any{"title": "still archived", "sections": []string{"resource-release"}}); code != 200 || out["section"] != "resource-release" {
		t.Fatalf("a topic already archived may stay %d %v", code, out)
	}
}

func TestV1PatchTopicScopeDropsGrants(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	resp, raw := f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicRole), "sess-alice", "/topics/{topic_id}", "", nil, map[string]any{
		"access_scope":    "users",
		"access_user_ids": []string{fmt.Sprint(w3UserBob)},
	})
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var nRole, nUser int64
	f.db.Raw(`SELECT COUNT(*) FROM topic_access_grant WHERE topic_id = ? AND subject_type = 'role'`, w3TopicRole).Scan(&nRole)
	f.db.Raw(`SELECT COUNT(*) FROM topic_access_grant WHERE topic_id = ? AND subject_type = 'user'`, w3TopicRole).Scan(&nUser)
	if nRole != 0 || nUser != 1 {
		t.Fatalf("grants role=%d user=%d", nRole, nUser)
	}
}

func TestV1PatchTopicPaidFreeAwards(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	f.putSession(t, "sess-staff", w3UserStaff, "user", "moderator")
	resp, raw := f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicPaid), "sess-staff", "/topics/{topic_id}", "", nil, map[string]any{
		"sections": []string{"news"},
	})
	if resp.StatusCode != 200 {
		t.Fatalf("to free %d %s", resp.StatusCode, raw)
	}
	aw := f.snapshotAwards()
	if len(aw) != 1 || aw[0].userID != w3UserAlice || aw[0].delta != 13 ||
		aw[0].reason != moemoepoint.ReasonContentApproved {
		t.Fatalf("refund %+v", aw)
	}
	resp, raw = f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicPaid), "sess-staff", "/topics/{topic_id}", "", nil, map[string]any{
		"sections": []string{"tech-help"},
	})
	if resp.StatusCode != 200 {
		t.Fatalf("to paid %d %s", resp.StatusCode, raw)
	}
	aw = f.snapshotAwards()
	if len(aw) != 2 || aw[1].userID != w3UserAlice || aw[1].delta != -13 ||
		aw[1].reason != moemoepoint.ReasonContentRemoved {
		t.Fatalf("charge %+v", aw)
	}
	_ = constants.CostConsumeSection
}

func TestV1PatchTopicStateTransitions(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	staffHdr := http.Header{}
	staffHdr.Set("Authorization", "Bearer staff-token")

	hide := func(id int, session string, hdr http.Header, want int) {
		t.Helper()
		resp, raw := f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", id), session, "/topics/{topic_id}", "", hdr, map[string]any{
			"state": "hidden",
		})
		if resp.StatusCode != want {
			t.Fatalf("hide %d as %s: %d %s", id, session, resp.StatusCode, raw)
		}
	}
	hide(w3TopicPub, "sess-alice", nil, 200)
	var hiddenBy string
	var status int
	if err := f.db.Raw(`SELECT status, hidden_by FROM topic WHERE id = ?`, w3TopicPub).Row().Scan(&status, &hiddenBy); err != nil {
		t.Fatal(err)
	}
	if status != 1 || hiddenBy != "author" {
		t.Fatalf("author hide %d %s", status, hiddenBy)
	}
	resp, raw := f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicPub), "sess-alice", "/topics/{topic_id}", "", nil, map[string]any{
		"state": "published",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("author unhide %d %s", resp.StatusCode, raw)
	}

	hide(w3TopicPub, "sess-staff", nil, 200)
	if err := f.db.Raw(`SELECT status, hidden_by FROM topic WHERE id = ?`, w3TopicPub).Row().Scan(&status, &hiddenBy); err != nil {
		t.Fatal(err)
	}
	if hiddenBy != "moderator" {
		t.Fatalf("staff hide %s", hiddenBy)
	}

	resp, raw = f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicPub), "sess-alice", "/topics/{topic_id}", "", nil, map[string]any{
		"state": "published",
	})
	if resp.StatusCode != 403 {
		t.Fatalf("author unhide moderator %d %s", resp.StatusCode, raw)
	}

	hide(w3TopicNSFW, "", staffHdr, 403)

	resp, raw = f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicModHide), "sess-alice", "/topics/{topic_id}", "", nil, map[string]any{
		"state": "hidden",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("same-state %d %s", resp.StatusCode, raw)
	}

	resp, raw = f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicHidden), "sess-other", "/topics/{topic_id}", "", nil, map[string]any{
		"title": "nope",
	})
	if resp.StatusCode != 404 {
		t.Fatalf("stranger hidden %d %s", resp.StatusCode, raw)
	}

	resp, raw = f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicOld), "sess-bob", "/topics/{topic_id}", "", nil, map[string]any{
		"title": "stolen", "state": "hidden",
	})
	if resp.StatusCode != 403 {
		t.Fatalf("mixed edit+hide without hide %d %s", resp.StatusCode, raw)
	}
	var title string
	f.db.Raw(`SELECT title FROM topic WHERE id = ?`, w3TopicOld).Scan(&title)
	if title != "old" {
		t.Fatalf("mixed wrote %q", title)
	}
}
