package apiv1

import (
	"encoding/json"
	"slices"
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

func TestCreditsOfMergesRowsOfOneCreditName(t *testing.T) {
	var d client.CatalogWorkDetail
	if err := json.Unmarshal([]byte(`{"id": 49,
		"characters": [
			{"id": 234, "display_name": "瓜生桜乃", "kind": "main", "spoiler": 0},
			{"id": 237989, "display_name": "瓜生 桜乃", "kind": "main", "spoiler": 0},
			{"id": 999, "display_name": "Twist", "kind": "side", "spoiler": 2}
		],
		"credits": [{"role_key": "voice-actor", "role_name": "声优", "credits": [
			{"id": 8908, "display_name": "安玖深音", "character_id": 234},
			{"id": 481, "display_name": "力丸乃りこ", "character_id": 999},
			{"id": 8908, "display_name": "安玖深音", "character_id": 237989},
			{"id": 8908, "display_name": "安玖深音", "character_id": null},
			{"id": 8908, "display_name": "安玖深音", "character_id": 234}
		]}]
	}`), &d); err != nil {
		t.Fatal(err)
	}

	groups := creditsOf(&d)
	if len(groups) != 1 {
		t.Fatalf("groups %+v", groups)
	}
	people := groups[0].People
	if len(people) != 2 || people[0].ID != "8908" || people[1].ID != "481" {
		t.Fatalf("people %+v, want 8908 then 481, once each", people)
	}
	if got := people[0].VoicedCharacters; !slices.Equal(got, []VoicedCharacter{"瓜生桜乃", "瓜生 桜乃"}) {
		t.Errorf("8908 voiced %q, want both roster characters once each", got)
	}
	if got := people[1].VoicedCharacters; got == nil || len(got) != 0 {
		t.Errorf("481 voiced %q, want an empty array: naming a spoiler character gives it away", got)
	}
}
