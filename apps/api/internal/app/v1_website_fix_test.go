package app

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

const (
	wsCatMin   = 930000701
	wsCatMain  = 930000701
	wsCatTieA  = 930000702
	wsCatTieB  = 930000703
	wsCatEmpty = 930000704
	wsCatMax   = 930000709

	wsGroupSingle = 930000711
	wsGroupMulti  = 930000712
	wsGroupMin    = 930000711
	wsGroupMax    = 930000719

	wsTagA     = 930000721
	wsTagB     = 930000722
	wsTagMulti = 930000723
	wsTagLoose = 930000724
	wsTagMin   = 930000721
	wsTagMax   = 930000729

	wsSiteMin  = 930000801
	wsSiteTie  = 930000804
	wsSiteMax  = 930000899
	wsSiteMain = 930000801
	wsSiteNSFW = 930000802
	wsSiteIcon = 930000803

	wsIconHash = "abababababababababababababababababababababababababababababababab"
)

func wsHost(id int) string {
	return fmt.Sprintf("site%d.example", id)
}

// newWebsiteFix seeds seven websites that share one created instant and two
// categories that share one sort_order, so page boundaries land inside ties.
func newWebsiteFix(t *testing.T) *writeFix {
	t.Helper()
	f := newWriteFix(t, nil)
	f.alice(t)
	f.putSession(t, "sess-admin", w3UserGrant, "user", "admin")
	f.putSession(t, "sess-banned", w3UserBanned)
	f.cleanupWebsites(t)
	t.Cleanup(func() { f.cleanupWebsites(t) })

	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	base := time.Date(2025, 7, 20, 22, 3, 23, 0, time.UTC)
	for _, c := range []struct {
		id    int
		slug  string
		order int
	}{{wsCatMain, "wt-main", 10}, {wsCatTieA, "wt-tie-a", 20}, {wsCatTieB, "wt-tie-b", 20}, {wsCatEmpty, "wt-empty", 30}} {
		run(`INSERT INTO galgame_website_category (id, name, label, description, sort_order, created, updated) VALUES (?, ?, ?, '', ?, ?, ?)`,
			c.id, c.slug, "label "+c.slug, c.order, base, base)
	}
	run(`INSERT INTO galgame_website_tag_group (id, name, label, description, sort_order, multi_select, created, updated)
		VALUES (?, 'wt-single', 'single', '', 10, false, ?, ?), (?, 'wt-multi', 'multi', '', 20, true, ?, ?)`,
		wsGroupSingle, base, base, wsGroupMulti, base, base)
	for _, tg := range []struct {
		id, level int
		slug      string
		group     *int
	}{
		{wsTagA, 10, "wt-a", ptr(wsGroupSingle)}, {wsTagB, 5, "wt-b", ptr(wsGroupSingle)},
		{wsTagMulti, 3, "wt-multi-tag", ptr(wsGroupMulti)}, {wsTagLoose, -5, "wt-loose", nil},
	} {
		run(`INSERT INTO galgame_website_tag (id, level, name, label, description, group_id, created, updated) VALUES (?, ?, ?, ?, '', ?, ?, ?)`,
			tg.id, tg.level, tg.slug, "label "+tg.slug, tg.group, base, base)
	}

	site := func(id int, cat int, ageLimit string, created time.Time) {
		t.Helper()
		run(`INSERT INTO galgame_website (id, name, url, create_time, description, icon, icon_image_hash, language, age_limit, domain,
			category_id, user_id, created, updated, status)
			VALUES (?, ?, ?, '2014-05-01', 'a description long enough', '', '', 'zh-cn', ?, ?, ?, ?, ?, ?, 'normal')`,
			id, fmt.Sprintf("Site %d", id), wsHost(id), ageLimit, fmt.Sprintf(`["https://%s"]`, wsHost(id)),
			cat, w3UserStaff, created, created)
	}
	site(wsSiteMain, wsCatMain, "all", base.Add(-time.Hour))
	site(wsSiteNSFW, wsCatMain, "r18", base.Add(-2*time.Hour))
	site(wsSiteIcon, wsCatTieA, "all", base.Add(-3*time.Hour))
	for i := 0; i < 7; i++ {
		age := "all"
		if i%3 == 0 {
			age = "r18"
		}
		cat := wsCatTieA
		if i%2 == 0 {
			cat = wsCatMain
		}
		site(wsSiteTie+i, cat, age, base.Add(123456*time.Microsecond))
	}
	run(`UPDATE galgame_website SET icon_image_hash = ? WHERE id = ?`, wsIconHash, wsSiteIcon)
	run(`UPDATE galgame_website SET icon = 'https://favicon.example/f.ico' WHERE id = ?`, wsSiteMain)
	for _, rel := range [][2]int{{wsSiteMain, wsTagA}, {wsSiteMain, wsTagMulti}, {wsSiteMain, wsTagLoose}, {wsSiteTie, wsTagB}, {wsSiteTie + 1, wsTagB}} {
		run(`INSERT INTO galgame_website_tag_relation (galgame_website_id, galgame_website_tag_id, created, updated) VALUES (?, ?, ?, ?)`,
			rel[0], rel[1], base, base)
	}
	return f
}

func ptr(n int) *int { return &n }

func (f *writeFix) cleanupWebsites(t *testing.T) {
	t.Helper()
	sites := `SELECT id FROM galgame_website WHERE id BETWEEN 930000801 AND 930000899 OR user_id BETWEEN 930000001 AND 930000999`
	for _, q := range []string{
		`DELETE FROM galgame_website_like WHERE website_id IN (` + sites + `)`,
		`DELETE FROM galgame_website_favorite WHERE website_id IN (` + sites + `)`,
		`DELETE FROM galgame_website_tag_relation WHERE galgame_website_id IN (` + sites + `)`,
		`DELETE FROM galgame_website WHERE id BETWEEN 930000801 AND 930000899 OR user_id BETWEEN 930000001 AND 930000999`,
		`DELETE FROM galgame_website_tag WHERE id BETWEEN 930000721 AND 930000729 OR name LIKE 'wt-%'`,
		`DELETE FROM galgame_website_tag_group WHERE id BETWEEN 930000711 AND 930000719 OR name LIKE 'wt-%'`,
		`DELETE FROM galgame_website_category WHERE id BETWEEN 930000701 AND 930000709 OR name LIKE 'wt-%'`,
	} {
		_ = f.db.Exec(q).Error
	}
}

func (f *writeFix) ws(t *testing.T, method, rawURL, specPath, session string, hdr http.Header, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, method, rawURL, session, specPath, "", hdr, payload)
	if len(body) == 0 {
		return resp, nil
	}
	return resp, problemMap(t, body)
}

func (f *writeFix) walkWebsites(t *testing.T, query string, limit int) []string {
	t.Helper()
	var got []string
	cursor := ""
	for page := 0; page < 100; page++ {
		url := fmt.Sprintf("/api/v1/websites?limit=%d%s", limit, query)
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		resp, body := f.ws(t, http.MethodGet, url, "/websites", "", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("walk %s: %d %+v", url, resp.StatusCode, body)
		}
		got = append(got, adminItemIDs(body)...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return got
		}
		cursor = next
	}
	t.Fatal("walk did not end")
	return nil
}

func websiteBody(host, title string, extra map[string]any) map[string]any {
	body := map[string]any{
		"host": host, "title": title, "description": "a description long enough",
		"website_category_id": fmt.Sprint(wsCatMain), "website_tag_ids": []string{},
		"is_nsfw": false, "language": "zh-cn",
	}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func errorAt(body map[string]any, pointer string) map[string]any {
	for _, e := range problemErrorsOf(body) {
		if e["pointer"] == pointer {
			return e
		}
	}
	return nil
}
