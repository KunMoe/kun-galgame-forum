package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/stickerclient"

	"github.com/gofiber/fiber/v3"
)

type fakeSource struct {
	packs []stickerclient.Pack
	err   error
	calls int
}

func (f *fakeSource) OfficialPacks(context.Context) ([]stickerclient.Pack, error) {
	f.calls++
	return f.packs, f.err
}

func hash(n int) string { return fmt.Sprintf("%064x", n) }

func pack(id string, created time.Time, title map[string]string, stickers ...stickerclient.Sticker) stickerclient.Pack {
	return stickerclient.Pack{ID: id, Title: title, CreatedAt: created, Stickers: stickers}
}

func sticker(id string, pos int, h string, w, ht int) stickerclient.Sticker {
	return stickerclient.Sticker{ID: id, Position: pos, Image: stickerclient.Image{Hash: h, Width: w, Height: ht}}
}

const (
	idA = "01a07fd4-7f77-7650-a419-3b7716c84063"
	idB = "01a07fd4-7f77-792c-bb35-5600514eee88"
	idC = "01a07fd4-7f77-797c-9256-b26cf97175e3"
	sid = "01a07fd4-0000-7000-8000-00000000000"
)

func newStickerAPI(t *testing.T, src *fakeSource) (*fiber.App, *Service, *time.Time) {
	t.Helper()
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	images := func(hashes []string) map[string]imageclient.ImageMeta {
		out := map[string]imageclient.ImageMeta{}
		for _, h := range hashes {
			if h == hash(1) {
				out[h] = imageclient.ImageMeta{Width: 640, Height: 600, Thumbhash: "pUgK"}
			}
		}
		return out
	}
	s := New(src, images, "https://image.test.example")
	s.now = func() time.Time { return now }
	app := fiber.New(fiber.Config{ErrorHandler: v1.WriteFiberError})
	v1.Setup(app, v1.Deps{}, Register(s))
	return app, s, &now
}

func call(t *testing.T, app *fiber.App, ifNoneMatch string) (*http.Response, []byte) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sticker-packs", nil)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

func TestListStickerPacks(t *testing.T) {
	early := time.Date(2026, 8, 30, 14, 22, 44, 0, time.UTC)
	src := &fakeSource{packs: []stickerclient.Pack{
		pack(idC, early.Add(time.Hour), map[string]string{"ja-jp": "スタンプ", "en-us": "Stamps"},
			sticker(sid+"3", 1, hash(3), 0, 0)),
		pack(idB, early, map[string]string{"zh-cn": "鲲 Galgame 表情包 [2]"},
			sticker(sid+"2", 1, hash(2), 320, 300)),
		pack(idA, early, map[string]string{"zh-tw": "鯤 [1]", "zh-cn": "鲲 Galgame 表情包 [1]"},
			sticker(sid+"1", 1, hash(1), 0, 0),
			sticker(sid+"9", 2, "not-a-hash", 0, 0)),
		pack("01a07fd4-7f77-7a68-9ada-e2855b1664ec", early, map[string]string{"zh-cn": "空"},
			sticker(sid+"8", 1, "", 0, 0)),
	}}
	app, _, _ := newStickerAPI(t, src)

	resp, raw := call(t, app, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, raw)
	}
	if got := resp.Header.Get("Cache-Control"); got != "private, max-age=3600" {
		t.Errorf("Cache-Control = %q", got)
	}
	etag := resp.Header.Get("ETag")
	if len(etag) != 34 || etag[0] != '"' {
		t.Errorf("ETag = %q, want a quoted strong tag", etag)
	}

	var body struct {
		Object string `json:"object"`
		Items  []struct {
			Object      string `json:"object"`
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Stickers    []struct {
				Object      string `json:"object"`
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
				Image       struct {
					URL       string  `json:"url"`
					Hash      string  `json:"hash"`
					Width     *int    `json:"width"`
					Height    *int    `json:"height"`
					Thumbhash *string `json:"thumbhash"`
				} `json:"image"`
			} `json:"stickers"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, p := range body.Items {
		order = append(order, p.ID+" "+p.DisplayName)
	}
	want := []string{idA + " 鲲 Galgame 表情包 [1]", idB + " 鲲 Galgame 表情包 [2]", idC + " スタンプ"}
	if fmt.Sprint(order) != fmt.Sprint(want) {
		t.Fatalf("packs = %v, want %v (created_at then id; a pack with no usable sticker dropped)", order, want)
	}

	a := body.Items[0]
	if a.Object != "sticker_pack" || len(a.Stickers) != 1 {
		t.Fatalf("pack A = %+v, want the malformed hash dropped", a)
	}
	st := a.Stickers[0]
	if st.Object != "sticker" || st.ID != sid+"1" || st.DisplayName != "鲲 Galgame 表情包 [1] - 1" ||
		st.Image.Hash != hash(1) || st.Image.URL != fmt.Sprintf("https://image.test.example/00/00/%s.webp", hash(1)) {
		t.Errorf("sticker = %+v", st)
	}
	if st.Image.Width == nil || *st.Image.Width != 640 || st.Image.Thumbhash == nil || *st.Image.Thumbhash != "pUgK" {
		t.Errorf("image meta = %+v, want the image service's", st.Image)
	}
	b := body.Items[1].Stickers[0].Image
	if b.Width == nil || *b.Width != 320 || *b.Height != 300 || b.Thumbhash != nil {
		t.Errorf("image without meta = %+v, want the face's own dimensions", b)
	}
	c := body.Items[2].Stickers[0].Image
	if c.Width != nil || c.Height != nil {
		t.Errorf("image with no dimensions anywhere = %+v, want null", c)
	}
}

func TestListStickerPacksNotModified(t *testing.T) {
	src := &fakeSource{packs: []stickerclient.Pack{
		pack(idA, time.Time{}, map[string]string{"zh-cn": "A"}, sticker(sid+"1", 1, hash(1), 0, 0)),
	}}
	app, _, _ := newStickerAPI(t, src)
	resp, _ := call(t, app, "")
	etag := resp.Header.Get("ETag")

	for _, inm := range []string{etag, "W/" + etag, `"stale", ` + etag, "*"} {
		resp, raw := call(t, app, inm)
		if resp.StatusCode != http.StatusNotModified || len(raw) != 0 {
			t.Errorf("If-None-Match %s: status %d body %q, want 304 and no body", inm, resp.StatusCode, raw)
		}
		if resp.Header.Get("ETag") != etag || resp.Header.Get("Cache-Control") != "private, max-age=3600" {
			t.Errorf("If-None-Match %s: headers %v", inm, resp.Header)
		}
	}
	if resp, _ := call(t, app, `"stale"`); resp.StatusCode != http.StatusOK {
		t.Errorf("a stale tag: status %d, want 200", resp.StatusCode)
	}
	if src.calls != 1 {
		t.Errorf("%d upstream reads, want 1 within the hour", src.calls)
	}
}

func TestListStickerPacksRefresh(t *testing.T) {
	src := &fakeSource{packs: []stickerclient.Pack{
		pack(idA, time.Time{}, map[string]string{"zh-cn": "A"}, sticker(sid+"1", 1, hash(1), 0, 0)),
	}}
	app, _, now := newStickerAPI(t, src)
	resp, _ := call(t, app, "")
	first := resp.Header.Get("ETag")

	*now = now.Add(time.Hour)
	src.err = errors.New("gateway down")
	resp, _ = call(t, app, "")
	if resp.StatusCode != http.StatusOK || resp.Header.Get("ETag") != first {
		t.Fatalf("while the face is down: status %d ETag %q, want the last good list", resp.StatusCode, resp.Header.Get("ETag"))
	}
	call(t, app, "")
	if src.calls != 2 {
		t.Errorf("%d upstream reads, want no retry for five minutes after a failure", src.calls)
	}

	*now = now.Add(5 * time.Minute)
	src.err = nil
	src.packs = append(src.packs, pack(idB, time.Time{}, map[string]string{"zh-cn": "B"}, sticker(sid+"2", 1, hash(2), 0, 0)))
	resp, _ = call(t, app, first)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("ETag") == first {
		t.Errorf("after a change: status %d ETag %q, want 200 under a new tag", resp.StatusCode, resp.Header.Get("ETag"))
	}
}

func TestListStickerPacksUnavailable(t *testing.T) {
	app, _, _ := newStickerAPI(t, &fakeSource{err: stickerclient.ErrNotConfigured})
	resp, raw := call(t, app, "")
	if resp.StatusCode != http.StatusServiceUnavailable || resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("status %d Cache-Control %q: %s", resp.StatusCode, resp.Header.Get("Cache-Control"), raw)
	}
	var p struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(raw, &p); err != nil || p.Code != "SERVICE_UNAVAILABLE" {
		t.Errorf("problem = %s", raw)
	}
}

func TestRefreshTeachesTheStickerSet(t *testing.T) {
	fresh := hash(0x5712c3e5)
	src := &fakeSource{packs: []stickerclient.Pack{
		pack(idA, time.Time{}, map[string]string{"zh-cn": "A"}, sticker(sid+"1", 1, fresh, 0, 0)),
	}}
	_, s, _ := newStickerAPI(t, src)
	if markdown.IsSticker(fresh) {
		t.Fatal("a sticker the packs never listed is already known")
	}
	s.Refresh()
	if src.calls != 1 || !markdown.IsSticker(fresh) {
		t.Fatalf("after a refresh: %d upstream reads, IsSticker %v", src.calls, markdown.IsSticker(fresh))
	}
}
