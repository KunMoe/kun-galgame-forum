package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/pkg/letmoeclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

const (
	letmoeListPath      = "/works/{work_id}/letmoe-resources"
	letmoeDownloadPath  = "/letmoe-resources/{resource_id}/downloads"
	letmoeResourcesPage = `{"object":"list","next_cursor":null,"total":null,"items":[
		{"object":"letmoe_resource","id":"1024","work_id":"61311","title":"心跳 Windows 电脑版 简体中文游戏本体资源下载",
		 "type":"game","origin":"repost","version":"","applicable_game_version":"","language":["zh-cn"],"platform":["win"],
		 "runtime":["native-win"],"engine_id":null,"size":"1.2 GB","bytes":null,"blake3":"","prerequisites":"",
		 "description":"","install_note":"","content_warning":"","content_limit":"nsfw","providers":["direct","baidu"],
		 "mirror_count":2,"direct_download":true,"composite_count":1,
		 "rating":{"quality":null,"stability":null,"compatibility":null,"count":0},"download_count":7,"view_count":30,
		 "web_url":"https://www.letmoe.com/resource/1024","created_at":"2026-10-01T00:00:00Z","updated_at":"2026-10-20T00:00:00Z",
		 "publisher":{"object":"user","id":"910000101","name":"letmoe-name","avatar_url":"","web_url":"https://www.letmoe.com/user/910000101"}}]}`
	letmoeFiles = `{"object":"letmoe_resource_files","resource_id":"1024","mirrors":[
		{"object":"letmoe_mirror","id":"1","kind":"hosted","provider":"","url":"https://dl.imoe.uk/a.zip","extract_code":"","archive_password":"","note":"","region":"","is_primary":true},
		{"object":"letmoe_mirror","id":"2","kind":"external_link","provider":"baidu","url":"https://pan.baidu.com/s/x","extract_code":"kun1","archive_password":"moe","note":"","region":"","is_primary":false}],
		"composites":[{"object":"letmoe_composite","id":"5","url":"https://dl.imoe.uk/c.zip","filename":"c.zip","bytes":1024,"blake3":"",
		 "patch":{"moyu_resource_id":"10463","name":"汉化补丁","type":["manual"],"language":["zh-Hans"],"publisher":"kun"}}]}`
)

type letmoeFix struct {
	app      *App
	spec     *specConformance
	upstream atomic.Int32
	status   atomic.Int32
	gotURL   atomic.Value
}

func newLetmoeFix(t *testing.T) *letmoeFix {
	t.Helper()
	f := &letmoeFix{}
	f.status.Store(http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.upstream.Add(1)
		f.gotURL.Store(r.URL.String())
		if code := int(f.status.Load()); code != http.StatusOK {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(code)
			_, _ = io.WriteString(w, `{"code":"SCOPE_REQUIRED","detail":"needs letmoe:files"}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/file") {
			_, _ = io.WriteString(w, letmoeFiles)
			return
		}
		_, _ = io.WriteString(w, letmoeResourcesPage)
	}))
	t.Cleanup(srv.Close)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })

	letmoe := letmoeclient.New(letmoeclient.Config{BaseURL: srv.URL, APIKey: "nmk_live_test"})
	f.app = &App{
		Fiber:     newFiber(),
		Config:    testConfig(),
		Redis:     rdb,
		GalgameV1: galgameapiv1.New(fakeWorks{}, nil, fakeUsers{}, rdb, "https://image.test.example").WithLetmoe(letmoe),
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	return f
}

func (f *letmoeFix) do(t *testing.T, method, url, specPath string) (*http.Response, []byte) {
	t.Helper()
	resp, err := f.app.Fiber.Test(httptest.NewRequest(method, url, nil))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.check(t, method, specPath, resp, body)
	return resp, body
}

func TestV1GalgameLetmoeResources(t *testing.T) {
	f := newLetmoeFix(t)

	resp, body := f.do(t, http.MethodGet, "/api/v1/works/61311/letmoe-resources", letmoeListPath)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	if u, _ := f.gotURL.Load().(string); u != "/v2/letmoe/works/61311/resources?include=publisher&limit=100&nsfw=true" {
		t.Errorf("upstream url = %q", u)
	}
	var list struct {
		Items []struct {
			ID                  string   `json:"id"`
			Version             *string  `json:"version_text"`
			DescriptionMarkdown *string  `json:"note_markdown"`
			Providers           []string `json:"providers"`
			DirectDownload      bool     `json:"has_direct_download"`
			Publisher           struct {
				Name *string `json:"name"`
			} `json:"publisher"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("items = %s", body)
	}
	r := list.Items[0]
	if r.Version != nil || r.DescriptionMarkdown != nil {
		t.Errorf("empty upstream strings must be null: %s", body)
	}
	if !r.DirectDownload || len(r.Providers) != 2 {
		t.Errorf("direct_download / providers = %v / %v", r.DirectDownload, r.Providers)
	}
	if r.Publisher.Name == nil || *r.Publisher.Name != "forum-name" {
		t.Errorf("publisher must come from this forum's account service: %s", body)
	}

	f.do(t, http.MethodGet, "/api/v1/works/61311/letmoe-resources", letmoeListPath)
	if n := f.upstream.Load(); n != 1 {
		t.Errorf("upstream calls = %d, want 1: the second read must come from the cache", n)
	}

	t.Run("letmoe holds nothing", func(t *testing.T) {
		f := newLetmoeFix(t)
		f.status.Store(http.StatusNotFound)
		resp, body := f.do(t, http.MethodGet, "/api/v1/works/61311/letmoe-resources", letmoeListPath)
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"items":[]`) {
			t.Errorf("status %d: %s", resp.StatusCode, body)
		}
	})
	t.Run("unknown work", func(t *testing.T) {
		f := newLetmoeFix(t)
		resp, body := f.do(t, http.MethodGet, "/api/v1/works/999/letmoe-resources", letmoeListPath)
		if resp.StatusCode != http.StatusNotFound || problemCode(t, body) != "NOT_FOUND" {
			t.Errorf("status %d: %s", resp.StatusCode, body)
		}
	})
}

func TestV1LetmoeResourceDownload(t *testing.T) {
	f := newLetmoeFix(t)

	resp, body := f.do(t, http.MethodPost, "/api/v1/letmoe-resources/1024/downloads", letmoeDownloadPath)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	if u, _ := f.gotURL.Load().(string); u != "/v2/letmoe/resources/1024/file?nsfw=true" {
		t.Errorf("upstream url = %q", u)
	}
	var dl struct {
		Mirrors []struct {
			IsHosted       bool    `json:"is_hosted"`
			Provider       *string `json:"provider"`
			ExtractionCode string  `json:"extraction_code"`
		} `json:"mirrors"`
		Composites []struct {
			PatchName      string   `json:"patch_name"`
			PatchLanguages []string `json:"patch_languages"`
			MoyuResourceID string   `json:"moyu_resource_id"`
		} `json:"composites"`
	}
	if err := json.Unmarshal(body, &dl); err != nil {
		t.Fatal(err)
	}
	if len(dl.Mirrors) != 2 || !dl.Mirrors[0].IsHosted || dl.Mirrors[0].Provider != nil || dl.Mirrors[1].ExtractionCode != "kun1" {
		t.Errorf("mirrors = %s", body)
	}
	if len(dl.Composites) != 1 || dl.Composites[0].PatchName != "汉化补丁" ||
		len(dl.Composites[0].PatchLanguages) != 1 || dl.Composites[0].MoyuResourceID != "10463" {
		t.Errorf("composites = %s", body)
	}

	t.Run("key lacks letmoe:files", func(t *testing.T) {
		f := newLetmoeFix(t)
		f.status.Store(http.StatusForbidden)
		resp, body := f.do(t, http.MethodPost, "/api/v1/letmoe-resources/1024/downloads", letmoeDownloadPath)
		if resp.StatusCode != http.StatusServiceUnavailable || problemCode(t, body) != "SERVICE_UNAVAILABLE" {
			t.Errorf("status %d: %s", resp.StatusCode, body)
		}
	})
	t.Run("gone on letmoe", func(t *testing.T) {
		f := newLetmoeFix(t)
		f.status.Store(http.StatusNotFound)
		resp, body := f.do(t, http.MethodPost, "/api/v1/letmoe-resources/1024/downloads", letmoeDownloadPath)
		if resp.StatusCode != http.StatusNotFound || problemCode(t, body) != "NOT_FOUND" {
			t.Errorf("status %d: %s", resp.StatusCode, body)
		}
	})
}
