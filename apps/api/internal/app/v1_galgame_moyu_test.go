package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/moyuclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

const (
	moyuWorkID      = 61311
	moyuPublisher   = 910000101
	moyuDeparted    = 910000102
	moyuAvatarHash  = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	moyuSpecPath    = "/works/{work_id}/moyu-patches"
	moyuPatchesPage = `{"object":"list","next_cursor":null,"total":null,"items":[
		{"object":"patch","id":"61311","vndb_id":"v4145","catalog_work_id":"61311",
		 "web_url":"https://www.moyu.moe/galgame/61311","resources":[
			{"object":"patch_resource","id":"10463","patch_id":"61311","name":"",
			 "storage":"s3","size":"0.571 MB","hash":"","model_name":"","localization_group_name":"",
			 "note":"","type":["manual","image"],"language":["zh-Hans"],"platform":["windows"],
			 "download_count":12,"like_count":0,"web_url":"https://www.moyu.moe/resource/10463",
			 "created_at":"2025-11-02T09:14:33Z","updated_at":"2026-08-29T02:51:07Z",
			 "publisher":{"object":"user","id":"910000101","name":"moyu-name","avatar_url":"https://moyu/a.webp"}},
			{"object":"patch_resource","id":"10464","patch_id":"61311","name":"AI patch",
			 "storage":"user","size":"1.2 GB","hash":"","model_name":"Sakura","localization_group_name":"",
			 "note":"**read me**","type":["ai"],"language":["zh-Hans","zh-Hant"],"platform":["windows","android"],
			 "download_count":3,"like_count":1,"web_url":"https://www.moyu.moe/resource/10464",
			 "created_at":"2025-11-02T09:14:33Z","updated_at":"2026-09-01T00:00:00Z",
			 "publisher":{"object":"user","id":"910000102","name":"","avatar_url":""}}]}]}`
)

type fakeWorks struct{ err *legacyErrors.AppError }

func (f fakeWorks) CatalogWorkExists(_ context.Context, workID int) (bool, *legacyErrors.AppError) {
	if f.err != nil {
		return false, f.err
	}
	return workID == moyuWorkID, nil
}

type fakeUsers struct{}

func (fakeUsers) Users(_ context.Context, ids []int) (map[int]userclient.User, error) {
	out := map[int]userclient.User{}
	for _, id := range ids {
		if id == moyuPublisher {
			out[id] = userclient.User{ID: id, Name: "forum-name", AvatarImageHash: moyuAvatarHash}
		}
	}
	return out, nil
}

type moyuFix struct {
	app      *App
	spec     *specConformance
	upstream atomic.Int32
	status   atomic.Int32
	gotQuery atomic.Value
}

func newMoyuFix(t *testing.T, works fakeWorks, apiKey string) *moyuFix {
	t.Helper()
	f := &moyuFix{}
	f.status.Store(http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.upstream.Add(1)
		f.gotQuery.Store(r.URL.RawQuery)
		if code := int(f.status.Load()); code != http.StatusOK {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(code)
			_, _ = io.WriteString(w, `{"status":429,"code":"RATE_LIMITED","detail":"slow down"}`)
			return
		}
		_, _ = io.WriteString(w, moyuPatchesPage)
	}))
	t.Cleanup(srv.Close)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })

	moyu := moyuclient.New(moyuclient.Config{BaseURL: srv.URL, APIKey: apiKey})
	f.app = &App{
		Fiber:     newFiber(),
		Config:    testConfig(),
		Redis:     rdb,
		GalgameV1: galgameapiv1.New(works, moyu, fakeUsers{}, rdb, "https://image.test.example"),
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	return f
}

func (f *moyuFix) get(t *testing.T, workID string) (*http.Response, []byte) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/works/"+workID+"/moyu-patches", nil)
	resp, err := f.app.Fiber.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.check(t, http.MethodGet, moyuSpecPath, resp, body)
	return resp, body
}

func problemCode(t *testing.T, body []byte) string {
	t.Helper()
	var p struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("problem body: %v\n%s", err, body)
	}
	return p.Code
}

func TestV1GalgameMoyuPatches(t *testing.T) {
	f := newMoyuFix(t, fakeWorks{}, "nmk_live_test")

	resp, body := f.get(t, "61311")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	if q, _ := f.gotQuery.Load().(string); q != "include=resources%2Cpublisher&nsfw=true&refs=catalog%3A61311" {
		t.Errorf("upstream query = %q", q)
	}

	var list struct {
		Object string `json:"object"`
		Items  []struct {
			ID        string `json:"id"`
			WebURL    string `json:"web_url"`
			Resources []struct {
				ID           string   `json:"id"`
				Name         *string  `json:"name"`
				Storage      string   `json:"storage"`
				ModelName    *string  `json:"model_name"`
				NoteMarkdown *string  `json:"note_markdown"`
				Types        []string `json:"types"`
				Publisher    struct {
					ID     string  `json:"id"`
					Name   *string `json:"name"`
					Avatar *struct {
						Hash string `json:"hash"`
					} `json:"avatar"`
				} `json:"publisher"`
			} `json:"resources"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || len(list.Items[0].Resources) != 2 {
		t.Fatalf("items = %s", body)
	}
	first, second := list.Items[0].Resources[0], list.Items[0].Resources[1]
	if first.Name != nil || first.ModelName != nil || first.NoteMarkdown != nil {
		t.Errorf("empty upstream strings must be null: %s", body)
	}
	if len(first.Types) != 2 || first.Types[1] != "image" {
		t.Errorf("an upstream token outside the known set must pass through: %v", first.Types)
	}
	if first.Publisher.Name == nil || *first.Publisher.Name != "forum-name" ||
		first.Publisher.Avatar == nil || first.Publisher.Avatar.Hash != moyuAvatarHash {
		t.Errorf("publisher must come from this forum's account service: %+v", first.Publisher)
	}
	if second.Publisher.ID != "910000102" || second.Publisher.Name != nil {
		t.Errorf("a departed publisher keeps its id and a null name: %+v", second.Publisher)
	}
	if second.NoteMarkdown == nil || *second.NoteMarkdown != "**read me**" {
		t.Errorf("note_markdown = %v", second.NoteMarkdown)
	}

	f.get(t, "61311")
	if n := f.upstream.Load(); n != 1 {
		t.Errorf("upstream calls = %d, want 1: the second read must come from the cache", n)
	}
}

func TestV1GalgameMoyuPatchesErrors(t *testing.T) {
	t.Run("unknown galgame", func(t *testing.T) {
		f := newMoyuFix(t, fakeWorks{}, "nmk_live_test")
		resp, body := f.get(t, "999")
		if resp.StatusCode != http.StatusNotFound || problemCode(t, body) != "NOT_FOUND" {
			t.Errorf("status %d: %s", resp.StatusCode, body)
		}
	})
	t.Run("malformed id", func(t *testing.T) {
		f := newMoyuFix(t, fakeWorks{}, "nmk_live_test")
		resp, body := f.get(t, "0")
		if resp.StatusCode != http.StatusBadRequest || problemCode(t, body) != "INVALID_PARAMETER" {
			t.Errorf("status %d: %s", resp.StatusCode, body)
		}
	})
	t.Run("moyu refuses", func(t *testing.T) {
		f := newMoyuFix(t, fakeWorks{}, "nmk_live_test")
		f.status.Store(http.StatusTooManyRequests)
		resp, body := f.get(t, "61311")
		if resp.StatusCode != http.StatusServiceUnavailable || problemCode(t, body) != "SERVICE_UNAVAILABLE" {
			t.Errorf("status %d: %s", resp.StatusCode, body)
		}
		f.status.Store(http.StatusOK)
		if resp, _ := f.get(t, "61311"); resp.StatusCode != http.StatusOK {
			t.Errorf("a refusal must not be cached, got %d", resp.StatusCode)
		}
	})
	t.Run("no key", func(t *testing.T) {
		f := newMoyuFix(t, fakeWorks{}, "")
		resp, body := f.get(t, "61311")
		if resp.StatusCode != http.StatusServiceUnavailable || problemCode(t, body) != "SERVICE_UNAVAILABLE" {
			t.Errorf("status %d: %s", resp.StatusCode, body)
		}
		if n := f.upstream.Load(); n != 0 {
			t.Errorf("upstream calls = %d without a key", n)
		}
	})
	t.Run("catalog down", func(t *testing.T) {
		f := newMoyuFix(t, fakeWorks{err: legacyErrors.ErrInternal("catalog down")}, "nmk_live_test")
		resp, body := f.get(t, "61311")
		if resp.StatusCode != http.StatusServiceUnavailable || problemCode(t, body) != "SERVICE_UNAVAILABLE" {
			t.Errorf("status %d: %s", resp.StatusCode, body)
		}
	})
}
