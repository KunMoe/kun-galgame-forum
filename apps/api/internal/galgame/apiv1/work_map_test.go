package apiv1

import (
	"encoding/json"
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

func TestTagsOfMergesSourceRowsOfOneCanonicalTag(t *testing.T) {
	var d client.CatalogWorkDetail
	if err := json.Unmarshal([]byte(`{"id": 59609, "tags": [
		{"canonical_id": 947, "display_name": "同人小说", "source": "bangumi", "kind": "meta", "spoiler": 0, "tier": "core", "work_count": 326},
		{"canonical_id": 5101, "display_name": "纯爱", "source": "vndb", "kind": "content", "spoiler": 0, "tier": "core", "work_count": 3},
		{"canonical_id": 947, "display_name": "Fan Novel", "source": "vndb", "kind": "meta", "spoiler": 2, "tier": "core", "work_count": 326},
		{"canonical_id": 5102, "display_name": "H", "source": "bangumi", "kind": "content", "spoiler": 0, "tier": "core", "work_count": 1},
		{"canonical_id": 5102, "display_name": "H", "source": "vndb", "kind": "content", "spoiler": 0, "sexual": true, "tier": "core", "work_count": 1}
	]}`), &d); err != nil {
		t.Fatal(err)
	}

	nsfw := tagsOf(&d, true)
	if len(nsfw) != 3 {
		t.Fatalf("got %d tags, want 3 (one per canonical id): %+v", len(nsfw), nsfw)
	}
	if nsfw[0].ID != "947" || nsfw[1].ID != "5101" || nsfw[2].ID != "5102" {
		t.Errorf("order = %s,%s,%s, want catalog's first-seen order 947,5101,5102", nsfw[0].ID, nsfw[1].ID, nsfw[2].ID)
	}
	if nsfw[0].DisplayName != "同人小说" {
		t.Errorf("name = %q, want the first row's", nsfw[0].DisplayName)
	}
	if nsfw[0].Spoiler != "major" {
		t.Errorf("spoiler = %q, want major — one source grading it major must not be outvoted", nsfw[0].Spoiler)
	}
	if !nsfw[2].IsSexual {
		t.Error("5102 lost the sexual flag its vndb row carries")
	}

	sfw := tagsOf(&d, false)
	if len(sfw) != 2 || sfw[0].ID != "947" || sfw[1].ID != "5101" {
		t.Errorf("default read kept a tag one source marks sexual: %+v", sfw)
	}
}
