package app

import (
	"net/http"
	"strconv"
	"testing"
)

func TestV1SFWListsFailClosedOnUnsyncedVerdict(t *testing.T) {
	has := func(body map[string]any, id string) bool {
		for _, got := range adminItemIDs(body) {
			if got == id {
				return true
			}
		}
		return false
	}

	t.Run("resources", func(t *testing.T) {
		f := newResourceFix(t, nil)
		if err := f.db.Exec("UPDATE galgame SET content_limit = NULL WHERE id = ?", g3WorkSFW).Error; err != nil {
			t.Fatal(err)
		}
		for query, want := range map[string]bool{"": false, "&include_nsfw=true": true} {
			resp, body := f.rs(t, http.MethodGet, "/api/v1/galgame-resources?limit=100"+query, "/galgame-resources", "", "", nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("list%s %d %+v", query, resp.StatusCode, body)
			}
			if has(body, idStr(g3ResMain)) != want {
				t.Errorf("list%s: resource of a work without a verdict listed = %v, want %v", query, !want, want)
			}
		}
	})

	t.Run("ratings", func(t *testing.T) {
		f := newGRFix(t)
		f.run(t, "UPDATE galgame SET content_limit = NULL WHERE id = ?", grWorkSFW2)
		for query, wantAny := range map[string]bool{"": false, "&include_nsfw=true": true} {
			_, out := f.list(t, "", "work_id="+strconv.Itoa(grWorkSFW2)+query)
			if got := out["total"].(float64) > 0; got != wantAny {
				t.Errorf("ratings%s of a work without a verdict: total %v, want any = %v", query, out["total"], wantAny)
			}
		}
	})

	t.Run("quizzes", func(t *testing.T) {
		f := newQuizFix(t, nil)
		if err := f.db.Exec("UPDATE galgame SET content_limit = NULL WHERE id = ?", g2WorkSFW).Error; err != nil {
			t.Fatal(err)
		}
		for query, want := range map[string]bool{"": false, "&include_nsfw=true": true} {
			resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes?limit=100"+query, "/quizzes", "", "", nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("list%s %d %+v", query, resp.StatusCode, body)
			}
			if has(body, idStr(g2QMain)) != want {
				t.Errorf("list%s: quiz on a work without a verdict listed = %v, want %v", query, !want, want)
			}
		}
	})
}
