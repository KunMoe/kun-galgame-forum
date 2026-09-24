package client

import (
	"encoding/json"
	"testing"
)

const nameLinksFixture = `{
  "id": 900,
  "refs": [
    {"source": "vndb", "external_id": "7611"},
    {"source": "dlsite", "external_id": "RG12345"}
  ],
  "links": [
    {"source": "official_site", "url": "https://hk52.info"},
    {"source": "twitter", "url": "https://x.com/hozumik"},
    {"source": "pixiv", "url": "https://www.pixiv.net/users/3870762"},
    {"source": "web", "url": "https://cmc.booth.pm/"},
    {"source": "web", "url": "https://hozumik.fanbox.cc/"},
    {"source": "web", "url": "https://hozumik.tumblr.com/"},
    {"source": "web", "url": "https://anidb.net/cr1505"}
  ]
}`

func TestCatalogNameDecodesThePersonLinkLane(t *testing.T) {
	var n CatalogName
	if err := json.Unmarshal([]byte(nameLinksFixture), &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(n.Refs) != 2 {
		t.Fatalf("refs = %d, want 2 — the anchor lane must survive alongside links", len(n.Refs))
	}
	if len(n.Links) != 7 {
		t.Fatalf("links = %d, want 7", len(n.Links))
	}
	for i, link := range n.Links {
		if link.URL == "" {
			t.Errorf("links[%d] decoded with no URL", i)
		}
	}
}

func TestCatalogNameWithNoPersonLinksDecodesEmpty(t *testing.T) {
	var n CatalogName
	if err := json.Unmarshal([]byte(`{"id": 1, "links": []}`), &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(n.Links) != 0 {
		t.Errorf("links = %v, want empty", n.Links)
	}
}
