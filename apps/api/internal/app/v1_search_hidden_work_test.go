package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const (
	srHiddenWork   = 952000401
	srShownWork    = 952000402
	srHiddenRes    = 952000501
	srShownRes     = 952000502
	srHiddenRating = 952000601
	srHiddenQuiz   = 952000701
)

func (f *searchFix) seedWorkTiedWalls(t *testing.T) {
	t.Helper()
	cleanup := func() {
		for _, q := range []string{
			`DELETE FROM galgame_quiz WHERE id = 952000701`,
			`DELETE FROM galgame_rating WHERE id = 952000601`,
			`DELETE FROM galgame_resource WHERE id IN (952000501, 952000502)`,
			`DELETE FROM galgame WHERE id IN (952000401, 952000402)`,
		} {
			_ = f.db.Exec(q).Error
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO galgame (id, updated, catalog_rendered) VALUES (?, now(), false), (?, now(), true)`, []any{srHiddenWork, srShownWork}},
		{`INSERT INTO galgame_resource (id, work_id, user_id, updated) VALUES (?, ?, ?, now()), (?, ?, ?, now())`,
			[]any{srHiddenRes, srHiddenWork, srUserBob, srShownRes, srShownWork, srUserBob}},
		{`INSERT INTO galgame_rating (id, work_id, user_id, recommend, overall, updated) VALUES (?, ?, ?, 'yes', 8, now())`,
			[]any{srHiddenRating, srHiddenWork, srUserBob}},
		{`INSERT INTO galgame_quiz (id, user_id, type, question, spoiler_level) VALUES (?, ?, 'single', 'q', 'none')`,
			[]any{srHiddenQuiz, srUserBob}},
		{`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?), (?, ?)`,
			[]any{srHiddenQuiz, srShownWork, srHiddenQuiz, srHiddenWork}},
	} {
		if err := f.db.Exec(q.sql, q.args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q.sql)
		}
	}
}

func TestV1SearchWallCommentsDropsUnrenderedWorks(t *testing.T) {
	f := newSearchFix(t)
	f.seedWorkTiedWalls(t)
	f.catalog.body = srWorksBody
	f.community.pages = map[string]string{
		"": `{"posts":[` + strings.Join([]string{
			wallPost(901, 1, "61312", srUserBob, "藏起 catalog hides the claim"),
			wallPost(902, 1, fmt.Sprint(srHiddenWork), srUserBob, "藏起 catalog has no such work"),
			wallPost(903, 2, fmt.Sprintf("resource:%d", srHiddenRes), srUserBob, "藏起 resource"),
			wallPost(904, 2, fmt.Sprintf("rating:%d", srHiddenRating), srUserBob, "藏起 rating"),
			wallPost(905, 2, fmt.Sprintf("quiz:%d", srHiddenQuiz), srUserBob, "藏起 quiz"),
			wallPost(906, 1, fmt.Sprint(srWallWork), srUserBob, "藏起 shown work"),
			wallPost(907, 2, fmt.Sprintf("resource:%d", srShownRes), srUserBob, "藏起 shown resource"),
		}, ",") + `],"next_cursor":""}`,
	}
	search := func() []string {
		t.Helper()
		resp, body := f.get(t, srWallPath, "/search/wall-comments?q="+url.QueryEscape("藏起"), "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%d %+v", resp.StatusCode, body)
		}
		return idsOf(body)
	}

	if got := fmt.Sprint(search()); got != "[906 907]" {
		t.Errorf("hits %s, want only the walls of shown works", got)
	}

	if err := f.db.Exec(`UPDATE galgame SET catalog_rendered = true WHERE id = ?`, srHiddenWork).Error; err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(search()); got != "[903 904 905 906 907]" {
		t.Errorf("restored hits %s; the work wall stays out while catalog does not show it", got)
	}

	f.catalog.fail.Store(true)
	if got := fmt.Sprint(search()); got != "[901 902 903 904 905 906 907]" {
		t.Errorf("catalog down hits %s; with no answer the game walls are kept", got)
	}
}
