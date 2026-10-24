package app

import (
	"fmt"
	"net/http"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1CreateTopicRejections(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)

	type hit struct {
		name   string
		body   map[string]any
		status int
		code   string
		ptr    string
		reason string
		min    int
	}
	base := func(over map[string]any) map[string]any {
		b := map[string]any{
			"title": "ok", "content_markdown": "ok", "category": "galgame",
			"sections": []string{"news"}, "is_nsfw": false, "access_scope": "public",
		}
		for k, v := range over {
			if v == nil {
				delete(b, k)
			} else {
				b[k] = v
			}
		}
		return b
	}
	cases := []hit{
		{"blank title", base(map[string]any{"title": "   "}), 422, problem.CodeValidationFailed, "/title", problem.ReasonTooShort, 1},
		{"blank body", base(map[string]any{"content_markdown": "\n\t"}), 422, problem.CodeValidationFailed, "/content_markdown", problem.ReasonTooShort, 1},
		{"archived section", base(map[string]any{"sections": []string{"resource-release"}}), 422, problem.CodeValidationFailed, "/sections/0", problem.ReasonNotAllowedValue, 0},
		{"role missing roles", base(map[string]any{"access_scope": "role"}), 422, problem.CodeValidationFailed, "/access_roles", problem.ReasonRequired, 0},
		{"role with users", base(map[string]any{"access_scope": "role", "access_roles": []string{"creator"}, "access_user_ids": []string{"2"}}), 422, problem.CodeValidationFailed, "/access_user_ids", problem.ReasonInconsistentWith, 0},
		{"users missing users", base(map[string]any{"access_scope": "users"}), 422, problem.CodeValidationFailed, "/access_user_ids", problem.ReasonRequired, 0},
		{"users with roles", base(map[string]any{"access_scope": "users", "access_user_ids": []string{fmt.Sprint(w3UserBob)}, "access_roles": []string{"creator"}}), 422, problem.CodeValidationFailed, "/access_roles", problem.ReasonInconsistentWith, 0},
		{"public with roles", base(map[string]any{"access_roles": []string{"creator"}}), 422, problem.CodeValidationFailed, "/access_roles", problem.ReasonInconsistentWith, 0},
		{"public with users", base(map[string]any{"access_user_ids": []string{fmt.Sprint(w3UserBob)}}), 422, problem.CodeValidationFailed, "/access_user_ids", problem.ReasonInconsistentWith, 0},
	}
	for i, c := range cases {
		resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(20+i), nil, c.body)
		if resp.StatusCode != c.status {
			t.Errorf("%s status %d %s", c.name, resp.StatusCode, raw)
			continue
		}
		m := problemMap(t, raw)
		if m["code"] != c.code {
			t.Errorf("%s code %v", c.name, m["code"])
		}
		found := false
		for _, e := range problemErrors(t, raw) {
			if fmt.Sprint(e["pointer"]) == c.ptr && fmt.Sprint(e["reason"]) == c.reason {
				found = true
				if c.min > 0 {
					params, _ := e["params"].(map[string]any)
					if params == nil || asInt(params["min_length"]) != c.min {
						t.Errorf("%s min_length %v", c.name, e["params"])
					}
				}
			}
		}
		if !found {
			t.Errorf("%s missing %s %s in %v", c.name, c.ptr, c.reason, problemErrors(t, raw))
		}
	}
}

func TestV1CreateTopicDailyLimit(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	if err := f.db.Exec(`UPDATE kungal_user_state SET moemoepoint = -16 WHERE user_id = ?`, w3UserAlice).Error; err != nil {
		t.Fatal(err)
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(40), nil, map[string]any{
		"title": "x", "content_markdown": "y", "category": "galgame",
		"sections": []string{"news"}, "is_nsfw": false, "access_scope": "public",
	})
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	m := problemMap(t, raw)
	if m["code"] != problem.CodeTopicDailyLimitReached || asInt(m["limit"]) != 0 {
		t.Fatalf("body %v", m)
	}
}

func TestV1CreateTopicDailyLimitBeatsPaid(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	if err := f.db.Exec(`UPDATE kungal_user_state SET moemoepoint = 5 WHERE user_id = ?`, w3UserAlice).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`UPDATE topic SET created = NOW() WHERE user_id = ?`, w3UserAlice).Error; err != nil {
		t.Fatal(err)
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(41), nil, map[string]any{
		"title": "x", "content_markdown": "y", "category": "galgame",
		"sections": []string{"resource-seeking"}, "is_nsfw": false, "access_scope": "public",
	})
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	m := problemMap(t, raw)
	if m["code"] != problem.CodeTopicDailyLimitReached || asInt(m["limit"]) != 1 {
		t.Fatalf("want 429 limit 1 got %v", m)
	}
}

func TestV1CreateTopicPaidInsufficient(t *testing.T) {
	f := newWriteFix(t, nil)
	f.putSession(t, "sess-bob", w3UserBob)
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-bob", "/topics", keyUUID(42), nil, map[string]any{
		"title": "x", "content_markdown": "y", "category": "galgame",
		"sections": []string{"t-help"}, "is_nsfw": false, "access_scope": "public",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	m := problemMap(t, raw)
	if m["code"] != problem.CodeMoemoepointInsufficient || asInt(m["required"]) != 10 {
		t.Fatalf("%v", m)
	}
}

func TestV1CreateTopicTrustDeny(t *testing.T) {
	f := newWriteFix(t, denyChecker{})
	f.alice(t)
	var before int64
	f.db.Raw(`SELECT COUNT(*) FROM topic WHERE user_id = ?`, w3UserAlice).Scan(&before)
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(43), nil, map[string]any{
		"title": "x", "content_markdown": "y", "category": "galgame",
		"sections": []string{"news"}, "is_nsfw": false, "access_scope": "public",
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	if problemMap(t, raw)["code"] != problem.CodeContentRejected {
		t.Fatalf("%s", raw)
	}
	var after int64
	f.db.Raw(`SELECT COUNT(*) FROM topic WHERE user_id = ?`, w3UserAlice).Scan(&after)
	if after != before {
		t.Fatalf("wrote %d -> %d", before, after)
	}
	if len(f.snapshotAwards()) != 0 {
		t.Fatalf("awards %+v", f.snapshotAwards())
	}
}
