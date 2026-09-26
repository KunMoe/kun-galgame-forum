package stickerclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	packA = "01a07fd4-7f77-7650-a419-3b7716c84063"
	packB = "01a07fd4-7f77-792c-bb35-5600514eee88"
)

func facePack(id, title string) string {
	return fmt.Sprintf(`{"object":"pack","id":%q,"title":{"zh-cn":%q},"description":{},"official":true,
		"content_rating":"all_ages","sticker_count":1,"view_count":3,"download_count":0,
		"cover":{"url":"https://img/c.webp","thumb_url":"https://img/c_320.webp"},"cover_sticker_id":"01a07fd4-0000-7000-8000-000000000001",
		"tags":[],"author":{"object":"author","id":"2","name":"鲲"},
		"created_at":"2026-08-30T14:22:44Z","updated_at":"2026-08-30T14:22:44Z","published_at":"2026-08-30T14:22:44Z"`, id, title)
}

func TestOfficialPacks(t *testing.T) {
	var reqs []*http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs = append(reqs, r)
		switch {
		case r.URL.Path == "/v2/sticker/packs" && r.URL.Query().Get("cursor") == "":
			_, _ = fmt.Fprintf(w, `{"object":"list","items":[%s}],"next_cursor":"cur_2"}`, facePack(packA, "鲲 Galgame 表情包 [1]"))
		case r.URL.Path == "/v2/sticker/packs":
			_, _ = fmt.Fprintf(w, `{"object":"list","items":[%s}]}`, facePack(packB, "鲲 Galgame 表情包 [2]"))
		case strings.HasPrefix(r.URL.Path, "/v2/sticker/packs/"):
			id := strings.TrimPrefix(r.URL.Path, "/v2/sticker/packs/")
			_, _ = fmt.Fprintf(w, `%s,"stickers":[{"object":"sticker","id":"01a07fd4-0000-7000-8000-00000000000%c","pack_id":%q,"position":1,
				"image":{"hash":"%064x","url":"https://img/x.webp","thumb_url":"https://img/x_320.webp","width":320,"height":300}}],
				"works":[],"characters":[]}`, facePack(id, "from the pack itself"), id[len(id)-1], id, len(reqs))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	packs, err := New(Config{BaseURL: srv.URL + "/", APIKey: "nmk_live_test"}).OfficialPacks(context.Background())
	if err != nil {
		t.Fatalf("OfficialPacks: %v", err)
	}
	if len(reqs) != 4 {
		t.Fatalf("%d requests, want 2 list pages and 2 packs", len(reqs))
	}
	for _, r := range reqs {
		if got := r.Header.Get("Authorization"); got != "Bearer nmk_live_test" {
			t.Errorf("%s auth = %q", r.URL, got)
		}
	}
	first, second := reqs[0].URL.Query(), reqs[1].URL.Query()
	if first.Get("official") != "true" || first.Get("limit") != "100" || first.Has("nsfw") {
		t.Errorf("first page query = %v", first)
	}
	if second.Get("cursor") != "cur_2" || second.Get("official") != "true" {
		t.Errorf("second page query = %v", second)
	}

	if len(packs) != 2 || packs[0].ID != packA || packs[1].ID != packB {
		t.Fatalf("packs = %+v", packs)
	}
	p := packs[1]
	if !p.CreatedAt.Equal(time.Date(2026, 8, 30, 14, 22, 44, 0, time.UTC)) || p.Title["zh-cn"] != "from the pack itself" {
		t.Errorf("pack = %+v", p)
	}
	if len(p.Stickers) != 1 || p.Stickers[0].Position != 1 || p.Stickers[0].Image.Width != 320 ||
		p.Stickers[0].Image.Hash != fmt.Sprintf("%064x", 4) {
		t.Errorf("stickers = %+v", p.Stickers)
	}
}

func TestOfficialPacksFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":401,"code":"MISSING_CREDENTIAL","detail":"An application key is required"}`))
	}))
	t.Cleanup(srv.Close)

	_, err := New(Config{BaseURL: srv.URL, APIKey: "nmk_live_test"}).OfficialPacks(context.Background())
	if !errors.Is(err, ErrUpstream) || !strings.Contains(err.Error(), "MISSING_CREDENTIAL") {
		t.Errorf("err = %v", err)
	}

	if _, err := New(Config{BaseURL: srv.URL}).OfficialPacks(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("keyless err = %v", err)
	}
}
