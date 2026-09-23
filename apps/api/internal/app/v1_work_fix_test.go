package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"
)

const (
	g4WorkLive     = 944000001
	g4WorkHidden   = 944000002
	g4WorkMerged   = 944000003
	g4WorkNoLocal  = 944000004
	g4WorkNoOwner  = 944000005
	g4WorkOwner    = 944000006
	g4WorkUnknown  = 944000099
	g4UserGoneA    = 930005101
	g4UserGoneB    = 930005102
	g4PortraitHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	g4SafeHash     = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

var g4LongCJKTitle = strings.Repeat("长", 300)

type workFix struct {
	*writeFix
	cat  *fakeCatalog
	user *fakeCatalogUser
}

type fakeCatalogUser struct {
	scopeFolders bool
	holdings     map[int64]bool
	containing   map[int64]bool
	covers       []catalogclient.CoverTally
	userCovers   []catalogclient.CoverTally
	userCoverErr error
	play         *catalogclient.PlaytimeSelf
	state        *catalogclient.WorkStateRecord
	holdingsErr  error
}

func (f *fakeCatalogUser) MyFoldersContaining(_ context.Context, _ string, workID int64) ([]catalogclient.Folder, error) {
	if f.scopeFolders {
		return nil, catalogclient.ErrInsufficientScope
	}
	if f.containing[workID] {
		return []catalogclient.Folder{{ID: 1}}, nil
	}
	return nil, nil
}

func (f *fakeCatalogUser) MyFolderHoldings(_ context.Context, _ string, _ []int64) ([]catalogclient.FolderHolding, error) {
	if f.holdingsErr != nil {
		return nil, f.holdingsErr
	}
	if f.scopeFolders {
		return nil, catalogclient.ErrInsufficientScope
	}
	out := []catalogclient.FolderHolding{}
	for id, ok := range f.holdings {
		if ok {
			out = append(out, catalogclient.FolderHolding{WorkID: id, FolderIDs: []int64{1}})
		}
	}
	return out, nil
}

func (f *fakeCatalogUser) WorkCoversUser(context.Context, string, int64) ([]catalogclient.CoverTally, error) {
	if f.userCoverErr != nil {
		return nil, f.userCoverErr
	}
	if f.userCovers == nil {
		return nil, catalogclient.ErrUnauthorized
	}
	return f.userCovers, nil
}

func (f *fakeCatalogUser) WorkCoverVotes(context.Context, int64) ([]catalogclient.CoverTally, error) {
	return f.covers, nil
}

func (f *fakeCatalogUser) MyPlaytime(context.Context, string, int64) (*catalogclient.PlaytimeSelf, error) {
	if f.scopeFolders {
		return nil, catalogclient.ErrInsufficientScope
	}
	return f.play, nil
}

func (f *fakeCatalogUser) MyWorkState(context.Context, string, int64) (*catalogclient.WorkStateRecord, error) {
	return f.state, nil
}

func newWorkFix(t *testing.T) *workFix {
	t.Helper()
	f := newWriteFix(t, nil)
	f.alice(t)
	f.addOAuthUser(g4UserGoneA, "gone-a", 1, nil)
	f.addOAuthUser(g4UserGoneB, "gone-b", 1, nil)
	cat := newWorkCatalog(t)
	user := &fakeCatalogUser{
		holdings:   map[int64]bool{g4WorkLive: true},
		containing: map[int64]bool{g4WorkLive: true},
	}
	f.GalgameV1 = galgameapiv1.New(cat, nil, f.UserClient, f.rdb, geCDN).WithWork(f.db, user, nil, f.recordAward)
	f.Fiber = newFiber()
	f.setupRoutes()
	f.spec = newSpecConformance(t)
	wf := &workFix{writeFix: f, cat: cat, user: user}
	wf.cleanupWorks(t)
	t.Cleanup(func() { wf.cleanupWorks(t) })
	wf.seedWorks(t)
	return wf
}

func newWorkCatalog(t *testing.T) *fakeCatalog {
	t.Helper()
	cat := &fakeCatalog{
		rows:       map[int]client.CatalogWorkListItem{},
		details:    map[int]*client.CatalogWorkDetail{},
		movedWorks: map[int]int64{g4WorkMerged: g4WorkLive},
		works: []geWork{
			{id: g4WorkLive, name: "LiveWork", limit: "sfw", rating: "all_ages"},
			{id: g4WorkNoLocal, name: "GhostWork", limit: "sfw", rating: "all_ages"},
			{id: g4WorkNoOwner, name: "NoOwner", limit: "sfw", rating: "sensitive"},
			{id: g4WorkOwner, name: "OwnerWork", limit: "sfw", rating: "all_ages"},
		},
	}
	for _, w := range cat.works {
		var row client.CatalogWorkListItem
		decodeInto(t, geRowJSON(w), &row)
		cat.rows[w.id] = row
	}
	var hidden client.CatalogWorkListItem
	decodeInto(t, resourceHiddenJSON(g4WorkHidden, "HiddenWork"), &hidden)
	cat.rows[g4WorkHidden] = hidden
	cat.works = append(cat.works, geWork{id: g4WorkHidden, name: "HiddenWork", limit: "sfw", rating: "all_ages"})
	cat.details[g4WorkLive] = g4LiveDetail(t)
	return cat
}

func g4LiveDetail(t *testing.T) *client.CatalogWorkDetail {
	t.Helper()
	raw := fmt.Sprintf(`{
		"id": %d, "display_name": "LiveWork", "latin": "livework", "olang": "ja",
		"content_rating": "all_ages", "created": "2026-01-01T00:00:00Z", "updated": "2026-01-02T00:00:00Z",
		"localized": {"zh-Hans": {"value": "现场作", "machine": false}},
		"claim": {"site": "kungal", "site_work_id": %d, "state": "live", "content_limit": "sfw"},
		"titles": [{"lang": "zh-Hans", "title": "别名一"}, {"lang": "ja", "title": %q}],
		"characters": [
			{"id": 7001, "display_name": "Lead", "kind": "main", "spoiler": 0},
			{"id": 7002, "display_name": "VoiceOnly", "kind": "unknown", "spoiler": 0}
		],
		"cover_slots": {"portrait": {"url": %q, "width": 600, "height": 850, "thumbhash": "AbC+"}},
		"covers": [
			{"id": 11, "url": %q, "kind": "main", "source": "vndb", "width": 600, "height": 850, "thumbhash": "AbC+"},
			{"id": 12, "url": %q, "kind": "dig", "source": "vndb", "width": 800, "height": 450, "sexual": 0, "thumbhash": "sigO"}
		],
		"tags": [
			{"canonical_id": 5101, "name": "纯爱", "display_name": "纯爱", "kind": "content", "spoiler": 0, "sexual": false, "tier": "normal", "work_count": 3},
			{"canonical_id": 5102, "name": "H", "display_name": "H", "kind": "content", "spoiler": 0, "sexual": true, "tier": "normal", "work_count": 1}
		],
		"refs": [{"source": "vndb", "external_id": "v1"}]
	}`, g4WorkLive, g4WorkLive, g4LongCJKTitle,
		fmt.Sprintf("https://image.other.example/aa/aa/%s.webp", g4PortraitHash),
		fmt.Sprintf("https://image.other.example/aa/aa/%s.webp", g4PortraitHash),
		fmt.Sprintf("https://image.other.example/bb/bb/%s.webp", g4SafeHash))
	var d client.CatalogWorkDetail
	decodeInto(t, raw, &d)
	return &d
}

func (f *workFix) cleanupWorks(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM message WHERE link LIKE '/galgame/944000%' OR sender_id IN (930005101, 930005102) OR receiver_id IN (930005101, 930005102)`,
		`DELETE FROM galgame_like WHERE work_id BETWEEN 944000001 AND 944000099 OR user_id BETWEEN 930005101 AND 930005199`,
		`DELETE FROM galgame_contributor WHERE work_id BETWEEN 944000001 AND 944000099`,
		`DELETE FROM galgame WHERE id BETWEEN 944000001 AND 944000099`,
		`DELETE FROM galgame_view_daily WHERE entity_id BETWEEN 944000001 AND 944000099`,
	} {
		_ = f.db.Exec(q).Error
	}
}

func (f *workFix) seedWorks(t *testing.T) {
	t.Helper()
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("work seed: %v\n%s", err, q)
		}
	}
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	run(`INSERT INTO galgame (id, view, like_count, created, updated, published, content_limit, creator_user_id, resource_update_time, resource_publish_banned)
		VALUES (?, 10, 0, ?, ?, true, 'sfw', ?, ?, false),
		       (?, 0, 0, ?, ?, true, 'sfw', NULL, ?, false),
		       (?, 0, 0, ?, ?, true, 'sfw', ?, ?, false)`,
		g4WorkLive, base, base, w3UserAlice, base,
		g4WorkNoOwner, base, base, base,
		g4WorkOwner, base, base, w3UserAlice, base)
	run(`INSERT INTO galgame_contributor (work_id, user_id, first_at, last_at, revision_count, source)
		VALUES (?, ?, ?, ?, 1, 0), (?, ?, ?, ?, 1, 0), (?, ?, ?, ?, 2, 0)`,
		g4WorkLive, g4UserGoneA, base, base,
		g4WorkLive, g4UserGoneB, base, base,
		g4WorkLive, w3UserBob, base, base)
}

func (f *workFix) wk(t *testing.T, method, rawURL, spec, session string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.ts(t, method, rawURL, spec, session, "", payload)
}

func g4WorkPath(id int) string {
	return "/api/v1/works/" + idStr(id)
}

func asJSONType(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case float64:
		return "number"
	case json.Number:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}
