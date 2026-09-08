package newsclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSourcesDecodesDirectory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/news/sources" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("missing bearer, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"sources":[
			{"key":"ymgal","display_name":"月幕 Galgame","homepage_url":"https://www.ymgal.games",
			 "attribution":"本条情报转载自月幕 Galgame","column_url":"","publisher_uid":114748},
			{"key":"galgame_hihyou","display_name":"Galgame 批评","homepage_url":"https://space.bilibili.com/2072586344",
			 "attribution":"本条情报转载自 Galgame 批评","column_url":"https://space.bilibili.com/2072586344/article","publisher_uid":115235}
		]}}`))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL, APIKey: "k"}).Sources(context.Background())
	if err != nil {
		t.Fatalf("Sources: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d sources, want 2", len(got))
	}
	if got[0].Key != "ymgal" || got[0].DisplayName != "月幕 Galgame" || got[0].PublisherUID != 114748 {
		t.Errorf("first source decoded wrong: %+v", got[0])
	}
	if got[1].ColumnURL == "" || got[1].Attribution == "" {
		t.Errorf("attribution and column url must survive: %+v", got[1])
	}
}

func TestSourcesUnconfigured(t *testing.T) {
	if _, err := New(Config{}).Sources(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("got %v, want ErrNotConfigured", err)
	}
}

// The v2 face renamed two fields the forum reads and shrank the source block to
// two of them. Reading only the v1 names left `preview` empty on every card —
// the 情报 tab rendered titles and nothing else — and left `Key` empty, which
// the handler uses as the map key for the whole page's sources, so both
// partners collapsed onto "" and ?source= filtered on an empty string.
func TestFeedReadsTheV2FieldNames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/news" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","items":[
			{"object":"news_item","id":"4447","title":"同人Gal也有大制作？",
			 "summary":"同人游戏社团ももいろたんざく最近公开了其第三作的众筹消息。",
			 "source":{"object":"news_source","name":"ymgal","display_name":"月幕 Galgame"},
			 "source_url":"https://www.ymgal.games/co/article/876900909601259520",
			 "published_at":"2026-08-16T11:11:00Z"}],"total":4439}`))
	}))
	defer srv.Close()

	feed, err := New(Config{BaseURL: srv.URL, APIKey: "k"}).Feed(context.Background(), FeedQuery{Limit: 1})
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(feed.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(feed.Items))
	}
	it := feed.Items[0]
	if it.ID != 4447 {
		t.Errorf("string id not parsed: %d", it.ID)
	}
	if it.Preview == "" {
		t.Errorf("summary must land in Preview, or every card is a bare title: %+v", it)
	}
	if it.Source.Key != "ymgal" {
		t.Errorf("source name must land in Key: %+v", it.Source)
	}
	if feed.Count != 4439 {
		t.Errorf("total = %d", feed.Count)
	}
}

// The v1 names still win where both are present, so restoring the richer face
// upstream needs no change here.
func TestFeedPrefersTheV1NamesWhenBothArrive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":1,"title":"t","preview":"v1 lede","summary":"v2 lede",
			"source":{"key":"ymgal","name":"ignored","display_name":"月幕 Galgame",
			 "homepage_url":"https://www.ymgal.games","attribution":"转载自月幕"},
			"lane":"column","banner_url":"https://img.example/a.webp",
			"published_at":"2026-08-16T11:11:00Z"}]}`))
	}))
	defer srv.Close()

	feed, err := New(Config{BaseURL: srv.URL, APIKey: "k"}).Feed(context.Background(), FeedQuery{})
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	it := feed.Items[0]
	if it.Preview != "v1 lede" || it.Source.Key != "ymgal" {
		t.Errorf("v1 names must win: %+v %+v", it, it.Source)
	}
	if it.Lane != "column" || it.BannerURL == "" || it.Source.Attribution == "" {
		t.Errorf("the rich fields must still decode when upstream sends them: %+v", it)
	}
}
