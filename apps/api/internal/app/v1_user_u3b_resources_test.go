package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestV1UserResourcesPublishedWalk(t *testing.T) {
	f := newU3bFix(t)
	want := f.sqlIDs(t, `SELECT r.id::text FROM galgame_resource r
		JOIN galgame g ON g.id = r.work_id
		WHERE r.user_id = ? AND r.id BETWEEN ? AND ?
		  AND g.published = true
		  AND g.content_limit = 'sfw'
		ORDER BY r.created DESC, r.id DESC`, u3bOwner, u3bResMin, u3bResMax)
	if len(want) < 5 {
		t.Fatalf("published resources too thin: %v", want)
	}
	got, total := f.walkU3b(t, "galgame-resources", "published", "")
	if total != len(want) || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("walked %v total %d, want %v", got, total, want)
	}
}

func TestV1UserResourcesOmitsUnpublishedWork(t *testing.T) {
	f := newU3bFix(t)
	body := f.u3bListOK(t, "galgame-resources", "published", "")
	if listContains(body, u3bResUnpub) {
		t.Fatalf("resource on unpublished work leaked: %v", itemIDs(t, body))
	}
}

func TestV1UserResourcesLikedOmitsBannedAuthor(t *testing.T) {
	f := newU3bFix(t)
	body := f.u3bListOK(t, "galgame-resources", "liked", "")
	if listContains(body, u3bResBanned) {
		t.Fatalf("banned uploader resource leaked in liked: %v", itemIDs(t, body))
	}
	if !listContains(body, u3bResLiked) {
		t.Fatalf("liked renderable resource missing: %v", itemIDs(t, body))
	}
	got, total := f.walkU3b(t, "galgame-resources", "liked", "")
	if total != len(got) {
		t.Fatalf("liked total %d walked %d", total, len(got))
	}
	for _, id := range got {
		if id == strconvI(u3bResBanned) {
			t.Fatalf("banned id in walk %v", got)
		}
	}
}

func TestV1UserResourcesStateValidExcludesExpired(t *testing.T) {
	f := newU3bFix(t)
	body := f.u3bListOK(t, "galgame-resources", "published", "&state=valid")
	if listContains(body, u3bResExpired) {
		t.Fatalf("state=valid included expired: %v", itemIDs(t, body))
	}
	all := f.u3bListOK(t, "galgame-resources", "published", "")
	if !listContains(all, u3bResExpired) {
		t.Fatalf("unfiltered published missed expired: %v", itemIDs(t, all))
	}
}

func TestV1UserResourcesNSFWDefaultExcluded(t *testing.T) {
	f := newU3bFix(t)
	body := f.u3bListOK(t, "galgame-resources", "published", "")
	if listContains(body, u3bResNSFW) {
		t.Fatalf("default published included NSFW resource: %v", itemIDs(t, body))
	}
	with := f.u3bListOK(t, "galgame-resources", "published", "&include_nsfw=true")
	if !listContains(with, u3bResNSFW) {
		t.Fatalf("include_nsfw=true missed NSFW resource: %v", itemIDs(t, with))
	}
}

func TestV1UserResourcesOmitsSecrets(t *testing.T) {
	f := newU3bFix(t)
	resp, body := f.u3bList(t, "", "galgame-resources", "relation=published&limit=100")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	raw, _ := json.Marshal(body)
	s := string(raw)
	for _, secret := range []string{u3bSecretURL, u3bSecretCode, u3bSecretPass} {
		if strings.Contains(s, secret) {
			t.Errorf("list leaked %q", secret)
		}
	}
	items, _ := body["items"].([]any)
	if len(items) == 0 {
		t.Fatal("empty list")
	}
	for _, it := range items {
		m, _ := it.(map[string]any)
		if _, ok := m["download_urls"]; ok {
			t.Errorf("item has download_urls: %+v", m)
		}
		if _, ok := m["extraction_code"]; ok {
			t.Errorf("item has extraction_code: %+v", m)
		}
		if _, ok := m["archive_password"]; ok {
			t.Errorf("item has archive_password: %+v", m)
		}
	}
}
