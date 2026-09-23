package app

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestV1AdminWebsiteRenameRewritesLinks(t *testing.T) {
	f := newWebsiteFix(t)
	old := wsHost(wsSiteMain)
	lookalike := "x" + old
	now := time.Now()
	const feedA, feedB, feedC = 930000871, 930000872, 930000873
	t.Cleanup(func() {
		_ = f.db.Exec(`DELETE FROM feed_activity WHERE type = 'GALGAME_WEBSITE_COMMENT_CREATION' AND source_id BETWEEN ? AND ?`, feedA, feedC).Error
		_ = f.db.Exec(`DELETE FROM message WHERE sender_id = ? AND type = 'commented'`, w3UserBob).Error
	})
	for _, row := range []struct {
		id   int
		link string
	}{{feedA, "/website/" + old}, {feedB, "/website/" + old + "?comment=12"}, {feedC, "/website/" + lookalike}} {
		if err := f.db.Exec(`INSERT INTO feed_activity (type, source_id, user_id, content, link, created)
			VALUES ('GALGAME_WEBSITE_COMMENT_CREATION', ?, ?, 'c', ?, ?)`, row.id, w3UserBob, row.link, now).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, link := range []string{"/website/" + old + "#c", "/website/" + lookalike} {
		if err := f.db.Exec(`INSERT INTO message (content, link, type, sender_id, receiver_id, updated)
			VALUES ('c', ?, 'commented', ?, ?, ?)`, link, w3UserBob, w3UserAlice, now).Error; err != nil {
			t.Fatal(err)
		}
	}

	resp, body := f.ws(t, http.MethodPatch, fmt.Sprintf("/api/v1/admin/websites/%d", wsSiteMain), "/admin/websites/{website_id}",
		"sess-staff", nil, map[string]any{"host": "renamed-site.example"})
	if resp.StatusCode != http.StatusOK || body["host"] != "renamed-site.example" {
		t.Fatalf("rename %d %+v", resp.StatusCode, body)
	}
	feed := f.sqlIDs(t, `SELECT link FROM feed_activity WHERE type = 'GALGAME_WEBSITE_COMMENT_CREATION' AND source_id BETWEEN ? AND ? ORDER BY source_id`, feedA, feedC)
	want := []string{"/website/renamed-site.example", "/website/renamed-site.example?comment=12", "/website/" + lookalike}
	if fmt.Sprint(feed) != fmt.Sprint(want) {
		t.Errorf("feed links %v, want %v", feed, want)
	}
	msgs := f.sqlIDs(t, `SELECT link FROM message WHERE sender_id = ? AND type = 'commented' ORDER BY id`, w3UserBob)
	if fmt.Sprint(msgs) != fmt.Sprint([]string{"/website/renamed-site.example#c", "/website/" + lookalike}) {
		t.Errorf("notification links %v", msgs)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM feed_activity WHERE type = 'GALGAME_WEBSITE_CREATION' AND source_id = ? AND link = '/website/renamed-site.example'`, wsSiteMain); n != 1 {
		t.Errorf("the creation card follows the rename: %d", n)
	}

	resp, _ = f.ws(t, http.MethodPatch, fmt.Sprintf("/api/v1/admin/websites/%d", wsSiteMain), "/admin/websites/{website_id}",
		"sess-staff", nil, map[string]any{"title": "no rename"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch without rename %d", resp.StatusCode)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM feed_activity WHERE link LIKE '/website/renamed-site.example%'`); n != 3 {
		t.Errorf("links after a patch without a rename: %d", n)
	}
}
