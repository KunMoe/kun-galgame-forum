package apiv1

import (
	"context"
	"encoding/json"
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

// The relation ends of work 198 (Rewrite) as catalog sent them on 2026-10-13,
// after the v2 rewrite, plus three ends that must not reach the page.
func TestRelatedWorksOfMergesPerWorkAndOrdersByRelation(t *testing.T) {
	var d client.CatalogWorkDetail
	if err := json.Unmarshal([]byte(`{"id": 198, "relations": [
		{"relation_type": "adaptation_of", "phrase": "被改编为", "work": {"id": 213783, "content_limit": "sfw", "medium": "anime", "display_name": "Rewrite"}},
		{"relation_type": "fandisc_of", "phrase": "有 Fandisc", "work": {"id": 849, "content_limit": "sfw", "medium": "galgame", "display_name": "Rewrite Harvest festa!", "claim": {"site": "kungal", "site_work_id": 849, "state": "live"}}},
		{"relation_type": "fandisc_of", "phrase": "有 Fandisc", "work": {"id": 2679, "content_limit": "sfw", "medium": "galgame", "display_name": "Rewrite: Cradles Tale", "release_date": "2011-12-01", "release_date_precision": "month", "claim": {"site": "kungal", "site_work_id": 2679, "state": "live"}}},
		{"relation_type": "sequel_of", "phrase": "有续作", "work": {"id": 2679, "content_limit": "sfw", "medium": "galgame", "display_name": "Rewrite: Cradles Tale", "release_date": "2011-12-01", "release_date_precision": "month", "claim": {"site": "kungal", "site_work_id": 2679, "state": "live"}}},
		{"relation_type": "shares_character", "phrase": "角色出演", "work": {"id": 234749, "content_limit": "sfw", "medium": "galgame", "display_name": "偽りのアリス"}},
		{"relation_type": "side_story_of", "phrase": "有外传", "work": {"id": 849, "content_limit": "sfw", "medium": "galgame", "display_name": "Rewrite Harvest festa!", "claim": {"site": "kungal", "site_work_id": 849, "state": "live"}}},
		{"relation_type": "side_story_of", "phrase": "有外传", "work": {"id": 10, "content_limit": "sfw", "medium": "galgame", "display_name": "dated later", "release_date": "2014-05-30"}},
		{"relation_type": "side_story_of", "phrase": "有外传", "work": {"id": 9, "content_limit": "sfw", "medium": "galgame", "display_name": "undated"}},
		{"relation_type": "side_story_of", "phrase": "有外传", "work": {"id": 2680, "content_limit": "sfw", "medium": "galgame", "display_name": "Rewrite オカ研活動記録外伝", "release_date": "2012-08-31", "release_date_precision": "day"}},
		{"relation_type": "side_story_of", "phrase": "有外传", "work": {"id": 2680, "content_limit": "sfw", "medium": "galgame", "display_name": "Rewrite オカ研活動記録外伝", "release_date": "2012-08-31", "release_date_precision": "day"}},
		{"relation_type": "side_story_of", "phrase": "有外传", "work": {"id": 11, "content_limit": "sfw", "medium": "galgame", "display_name": "undated, higher id"}},
		{"relation_type": "side_story_of", "phrase": "有外传", "work": {"id": 5000, "content_limit": "sfw", "medium": "galgame", "display_name": "hidden", "claim": {"site": "kungal", "site_work_id": 5000, "state": "hidden"}}},
		{"relation_type": "side_story_of", "phrase": "有外传", "work": {"id": 5001, "content_limit": "sfw", "medium": "film", "display_name": "a medium the forum does not publish"}},
		{"relation_type": "lyrics_by", "phrase": "作词", "work": {"id": 5002, "content_limit": "sfw", "medium": "music", "display_name": "a relation the ranking does not know"}}
	]}`), &d); err != nil {
		t.Fatal(err)
	}

	got := relatedWorksOf(context.Background(), &d, "")
	var ids []string
	for _, w := range got {
		ids = append(ids, string(w.ID))
	}
	want := []string{"2679", "849", "2680", "10", "9", "11", "234749", "213783", "5002"}
	if len(ids) != len(want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ids = %v, want %v", ids, want)
		}
	}

	phrases := func(w RelatedWork) (out []string) {
		for _, r := range w.Relations {
			out = append(out, r.Phrase)
		}
		return out
	}
	if p := phrases(got[0]); len(p) != 2 || p[0] != "有续作" || p[1] != "有 Fandisc" {
		t.Errorf("2679 phrases = %v, want 有续作 then 有 Fandisc in one entry", p)
	}
	if p := phrases(got[1]); len(p) != 2 || p[0] != "有 Fandisc" || p[1] != "有外传" {
		t.Errorf("849 phrases = %v, want 有 Fandisc then 有外传 in one entry", p)
	}
	if p := phrases(got[2]); len(p) != 1 {
		t.Errorf("2680 phrases = %v, want one: the edge came twice", p)
	}
	if got[0].ReleaseDatePrecision == nil || *got[0].ReleaseDatePrecision != "month" || *got[0].ReleaseDate != "2011-12-01" {
		t.Errorf("2679 date = %v / %v, want catalog's month precision kept", got[0].ReleaseDate, got[0].ReleaseDatePrecision)
	}
	if got[7].Medium != "anime" || got[0].Medium != "galgame" {
		t.Errorf("media = %s / %s", got[0].Medium, got[7].Medium)
	}
	if got[0].Object != "work" || got[0].Localized == nil {
		t.Errorf("2679 is not a full work ref: %+v", got[0].WorkRef)
	}
}

func TestRelatedWorksOfIsAnEmptyArrayWhenThereAreNone(t *testing.T) {
	got := relatedWorksOf(context.Background(), &client.CatalogWorkDetail{ID: 1}, "")
	if got == nil || len(got) != 0 {
		t.Fatalf("got %#v, want a non-nil empty slice", got)
	}
	b, _ := json.Marshal(got)
	if string(b) != "[]" {
		t.Errorf("encodes as %s, want []", b)
	}
}
