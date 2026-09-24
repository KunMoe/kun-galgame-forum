package app

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/trust/gate"
)

const (
	g2HiddenA = 930003101
	g2HiddenB = 930003102

	g2QTied    = 930003201
	g2QMain    = 930003210
	g2QRegrade = 930003211
	g2QNSFW    = 930003212
	g2QOrphan  = 930003213
	g2QJudge   = 930003214
	g2QGone    = 930003299

	g2AnsBob     = 930003401
	g2AnsOther   = 930003402
	g2AnsGrant   = 930003403
	g2AnsStaff   = 930003404
	g2AnsRegrade = 930003410

	g2WorkSFW    = 942000001
	g2WorkNSFW   = 942000002
	g2WorkOrphan = 942000003
	g2WorkHidden = 942000004
	g2WorkMiss   = 942000099
)

type quizFix struct {
	*writeFix
	cat *fakeCatalog
}

func newQuizFix(t *testing.T, checker gate.Checker) *quizFix {
	t.Helper()
	f := newWriteFix(t, checker)
	f.alice(t)
	f.putSession(t, "sess-banned", w3UserBanned)
	f.putSession(t, "sess-grant", w3UserGrant)
	f.addOAuthUser(g2HiddenA, "hidden-a", 1, nil)
	f.addOAuthUser(g2HiddenB, "hidden-b", 1, nil)
	cat := newQuizCatalog(t)
	f.QuizCatalog = cat
	qf := &quizFix{writeFix: f, cat: cat}
	qf.cleanupQuizzes(t)
	t.Cleanup(func() { qf.cleanupQuizzes(t) })
	qf.seedQuizzes(t)
	return qf
}

func newQuizCatalog(t *testing.T) *fakeCatalog {
	t.Helper()
	cat := &fakeCatalog{
		rows: map[int]client.CatalogWorkListItem{},
		works: []geWork{
			{id: g2WorkSFW, name: "AlphaQuiz", limit: "sfw", rating: "all_ages"},
			{id: g2WorkNSFW, name: "BetaNSFW", limit: "nsfw", rating: "r18"},
			{id: g2WorkOrphan, name: "GammaGhost", limit: "sfw", rating: "all_ages"},
		},
	}
	for _, w := range cat.works {
		var row client.CatalogWorkListItem
		decodeInto(t, geRowJSON(w), &row)
		cat.rows[w.id] = row
	}
	var hidden client.CatalogWorkListItem
	decodeInto(t, quizHiddenJSON(g2WorkHidden, "DeltaHidden"), &hidden)
	cat.rows[g2WorkHidden] = hidden
	cat.works = append(cat.works, geWork{id: g2WorkHidden, name: "DeltaHidden", limit: "sfw", rating: "all_ages"})
	return cat
}

func quizHiddenJSON(id int, name string) string {
	return fmt.Sprintf(`{"id":%d,"display_name":%q,"latin":%q,
		"localized":{"zh-Hans":{"value":%q,"machine":true}},
		"content_rating":"all_ages","content_limit":"sfw","release_date":"2026-01-01",
		"claim":{"site":"kungal","site_work_id":%d,"state":"hidden","content_limit":"sfw"},
		"cover_slots":{"portrait":{"url":%q,"width":256,"height":361,"thumbhash":"pUgK"}}}`,
		id, name, name, name+"（中）", id, geImageURL(id))
}

func (f *quizFix) cleanupQuizzes(t *testing.T) {
	t.Helper()
	const owned = `SELECT id FROM galgame_quiz WHERE id BETWEEN 930003201 AND 930003299 OR user_id BETWEEN 930000001 AND 930003199`
	for _, q := range []string{
		`DELETE FROM message WHERE type = 'quiz-answered' AND link ~ '^/galgame-quiz/9300032[0-9]{2}$'`,
		`DELETE FROM galgame_quiz_answer WHERE quiz_id IN (` + owned + `) OR id BETWEEN 930003401 AND 930003499`,
		`DELETE FROM galgame_quiz_favorite WHERE quiz_id IN (` + owned + `)`,
		`DELETE FROM galgame_quiz_galgame WHERE quiz_id IN (` + owned + `)`,
		`DELETE FROM galgame_quiz_view_daily WHERE entity_id IN (` + owned + `)`,
		`DELETE FROM galgame_quiz WHERE id IN (` + owned + `)`,
		`DELETE FROM galgame WHERE id BETWEEN 942000001 AND 942000010`,
		`DELETE FROM kungal_user_state WHERE user_id IN (930003101, 930003102)`,
	} {
		_ = f.db.Exec(q).Error
	}
}

func (f *quizFix) seedQuizzes(t *testing.T) {
	t.Helper()
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("quiz seed: %v\n%s", err, q)
		}
	}
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	tied := time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)
	ansTime := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)
	run(`INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated) VALUES (?, 7, ?, ?), (?, 7, ?, ?)
		ON CONFLICT (user_id) DO NOTHING`, g2HiddenA, base, base, g2HiddenB, base, base)
	run(`INSERT INTO galgame (id, view, created, updated, published, content_limit, resource_update_time)
		VALUES (?, 0, ?, ?, true, 'sfw', ?), (?, 0, ?, ?, true, 'nsfw', ?)`,
		g2WorkSFW, base, base, base, g2WorkNSFW, base, base, base)

	authors := []int{w3UserAlice, w3UserBob, w3UserOther, w3UserGrant, w3UserStaff, g2HiddenA, g2HiddenB}
	single := `{"options":["one","two","three"],"answer":1}`
	for i := 0; i < 7; i++ {
		id := g2QTied + i
		run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
			VALUES (?, ?, 'plot', 'single', 3, 'none', ?, '', ?::jsonb, '', false, 0, 0, 0, 0, 0, 0, 0, ?, ?, ?)`,
			id, authors[i], fmt.Sprintf("Tied %d", id), single, tied, tied, tied)
		run(`INSERT INTO galgame_quiz_answer (quiz_id, user_id, role, created, updated) VALUES (?, ?, 'author', ?, ?)`,
			id, authors[i], tied, tied)
	}

	run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
		VALUES (?, ?, 'character', 'single', 5, 'portion', ?, 'A **desc**', ?::jsonb, 'Because **reasons**', true, 3, 4, 1, 0, 0, 0, 0, ?, ?, ?)`,
		g2QMain, w3UserAlice, "What is ||the secret||?\nSecond line", single, base.Add(time.Hour), base, base)
	run(`INSERT INTO galgame_quiz_answer (quiz_id, user_id, role, created, updated) VALUES (?, ?, 'author', ?, ?)`,
		g2QMain, w3UserAlice, base, base)
	run(`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, g2QMain, g2WorkSFW)

	run(`INSERT INTO galgame_quiz_answer (id, quiz_id, user_id, role, submitted, is_correct, created, updated)
		VALUES (?, ?, ?, 'answerer', '{"value":0}'::jsonb, false, ?, ?)`,
		g2AnsBob, g2QMain, w3UserBob, ansTime, ansTime)
	run(`INSERT INTO galgame_quiz_answer (id, quiz_id, user_id, role, submitted, is_correct, created, updated)
		VALUES (?, ?, ?, 'answerer', '{"value":1}'::jsonb, true, ?, ?)`,
		g2AnsOther, g2QMain, w3UserOther, ansTime, ansTime)
	run(`INSERT INTO galgame_quiz_answer (id, quiz_id, user_id, role, submitted, is_correct, created, updated)
		VALUES (?, ?, ?, 'answerer', '{"value":0}'::jsonb, false, ?, ?)`,
		g2AnsGrant, g2QMain, w3UserGrant, ansTime.Add(time.Second), ansTime)
	run(`INSERT INTO galgame_quiz_answer (id, quiz_id, user_id, role, submitted, is_correct, created, updated)
		VALUES (?, ?, ?, 'answerer', '{"value":2}'::jsonb, false, ?, ?)`,
		g2AnsStaff, g2QMain, w3UserStaff, ansTime.Add(2*time.Second), ansTime)

	run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
		VALUES (?, ?, 'trivia', 'single', 2, 'none', 'Regrade me', '', ?::jsonb, '', false, 0, 1, 0, 0, 0, 0, 0, ?, ?, ?)`,
		g2QRegrade, w3UserAlice, single, base, base, base)
	run(`INSERT INTO galgame_quiz_answer (quiz_id, user_id, role, created, updated) VALUES (?, ?, 'author', ?, ?)`,
		g2QRegrade, w3UserAlice, base, base)
	run(`INSERT INTO galgame_quiz_answer (id, quiz_id, user_id, role, submitted, is_correct, created, updated)
		VALUES (?, ?, ?, 'answerer', '{"value":0}'::jsonb, false, ?, ?)`,
		g2AnsRegrade, g2QRegrade, w3UserOther, ansTime, ansTime)

	run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
		VALUES (?, ?, 'plot', 'single', 1, 'none', 'NSFW linked', '', ?::jsonb, '', false, 0, 0, 0, 0, 0, 0, 0, ?, ?, ?)`,
		g2QNSFW, w3UserBob, single, base.Add(-time.Hour), base, base)
	run(`INSERT INTO galgame_quiz_answer (quiz_id, user_id, role, created, updated) VALUES (?, ?, 'author', ?, ?)`,
		g2QNSFW, w3UserBob, base, base)
	run(`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, g2QNSFW, g2WorkNSFW)

	run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
		VALUES (?, ?, 'other', 'single', 1, 'none', 'Orphan link', '', ?::jsonb, '', false, 0, 0, 0, 0, 0, 0, 0, ?, ?, ?)`,
		g2QOrphan, w3UserOther, single, base.Add(-2*time.Hour), base, base)
	run(`INSERT INTO galgame_quiz_answer (quiz_id, user_id, role, created, updated) VALUES (?, ?, 'author', ?, ?)`,
		g2QOrphan, w3UserOther, base, base)
	run(`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, g2QOrphan, g2WorkOrphan)

	run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
		VALUES (?, ?, 'system', 'judge', 4, 'none', 'True or false', '', '{"answer":true}'::jsonb, '', false, 0, 0, 0, 0, 0, 0, 0, ?, ?, ?)`,
		g2QJudge, w3UserGrant, base.Add(-3*time.Hour), base, base)
	run(`INSERT INTO galgame_quiz_answer (quiz_id, user_id, role, created, updated) VALUES (?, ?, 'author', ?, ?)`,
		g2QJudge, w3UserGrant, base, base)
}

func (f *quizFix) qz(t *testing.T, method, rawURL, spec, session, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.ts(t, method, rawURL, spec, session, idem, payload)
}

func (f *quizFix) walkQuizzes(t *testing.T, query string, limit int) []string {
	t.Helper()
	var got []string
	for page := 1; page <= 40; page++ {
		url := fmt.Sprintf("/api/v1/quizzes?page=%d&limit=%d%s", page, limit, query)
		resp, body := f.qz(t, http.MethodGet, url, "/quizzes", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("walk %s: %d %+v", url, resp.StatusCode, body)
		}
		ids := adminItemIDs(body)
		got = append(got, ids...)
		if len(ids) < limit {
			return got
		}
	}
	t.Fatal("walk did not end")
	return nil
}

func (f *quizFix) walkAnswers(t *testing.T, quizID int, limit int) []string {
	t.Helper()
	var got []string
	cur := ""
	for i := 0; i < 40; i++ {
		url := fmt.Sprintf("/api/v1/quizzes/%d/answers?limit=%d", quizID, limit)
		if cur != "" {
			url += "&cursor=" + cur
		}
		resp, body := f.qz(t, http.MethodGet, url, "/quizzes/{quiz_id}/answers", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("walk answers: %d %+v", resp.StatusCode, body)
		}
		ids := adminItemIDs(body)
		got = append(got, ids...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return got
		}
		cur = next
	}
	t.Fatal("answer walk did not end")
	return nil
}

func createQuizBody(extra map[string]any) map[string]any {
	body := map[string]any{
		"prompt_text":            "A question",
		"quiz_type":              "single",
		"quiz_category":          "plot",
		"difficulty":             3,
		"choices":                []string{"yes", "no"},
		"correct_choice_indexes": []int{0},
	}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func renderableQuizAuthorsSQL() string {
	return strconv.Itoa(w3UserAlice) + "," + strconv.Itoa(w3UserBob) + "," +
		strconv.Itoa(w3UserOther) + "," + strconv.Itoa(w3UserGrant) + "," + strconv.Itoa(w3UserStaff)
}
