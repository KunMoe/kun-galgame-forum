package client

import (
	"encoding/json"
	"testing"
)

const nameLinksFixture = `{
  "id": 900,
  "links": [
    {"source": "vndb", "url": "https://vndb.org/s2066", "is_identity": true, "page_id": "s2066"},
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
	if len(n.Links) != 8 {
		t.Fatalf("links = %d, want 8", len(n.Links))
	}
	if l := n.Links[0]; !l.IsIdentity || l.PageID != "s2066" || n.Links[1].IsIdentity {
		t.Fatalf("identity flag lost: %+v", n.Links[:2])
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
