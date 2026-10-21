package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestContributorSyncCountsForumOriginOnce(t *testing.T) {
	db := testdb.Open(t)
	const work = 945000001
	cleanup := func() {
		db.Exec("DELETE FROM galgame_contributor WHERE work_id = ?", work)
		db.Exec("DELETE FROM galgame WHERE id = ?", work)
	}
	cleanup()
	t.Cleanup(cleanup)
	if err := db.Create(&model.GalgameLocal{ID: work}).Error; err != nil {
		t.Fatal(err)
	}

	rev := func(id int, site string, actor int) string {
		return fmt.Sprintf(`{"id":"%d","target_object":"work","entity_id":"%d","seq":2,"action":"merged","changed_fields":[],`+
			`"actor_uid":"%d","site":%q,"origin_site":"kungal","created_at":"2026-10-21T10:00:00Z"}`, id, work, actor, site)
	}
	var mu sync.Mutex
	var asked []url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		asked = append(asked, r.URL.Query())
		mu.Unlock()
		items := []string{rev(15, "kungal", 7), rev(18, "editor", 8), rev(25, "kungal", 9)}
		if r.URL.Query().Get("cursor") != "" {
			items = nil
		}
		_, _ = fmt.Fprintf(w, `{"object":"list","items":[%s]}`, strings.Join(items, ","))
	}))
	t.Cleanup(srv.Close)

	mr := miniredis.RunT(t)
	if err := mr.Set(legacyContributorCursorKey, "20"); err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	s := NewGalgameContributorSync(catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "nmk_test"}),
		repository.NewGalgameContributorRepository(db), rdb)

	s.Run()

	if len(asked) == 0 || asked[0].Get("origin_site") != "kungal" || asked[0].Has("site") {
		t.Fatalf("feed asked with %v, want origin_site=kungal and no site=", asked)
	}
	var rows []struct {
		UserID        int `gorm:"column:user_id"`
		RevisionCount int `gorm:"column:revision_count"`
	}
	if err := db.Raw(`SELECT user_id, revision_count FROM galgame_contributor WHERE work_id = ? ORDER BY user_id`, work).
		Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(rows) != "[{8 1} {9 1}]" {
		t.Errorf("contributors = %v, want 8 (an editor edit from the forum) and 9 once each; 7 was counted under site=kungal", rows)
	}
	if got, _ := mr.Get(contributorCursorKey); got != "25" {
		t.Errorf("origin watermark = %q, want 25", got)
	}
	if got, _ := mr.Get(legacyContributorCursorKey); got != "20" {
		t.Errorf("legacy watermark moved to %q", got)
	}
}
