package workrepr

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

// Shaped like catalog's list face for work 4 on 2026-10-21: source tags carry
// no canonical id, and a canonical tag can arrive once per source.
const topTagsRow = `{"object":"list","items":[{"object":"work","id":"4","medium":"galgame","display_name":"Summer Pockets",
"localized":{},"content_limit":"sfw","tags":[
{"id":null,"display_name":"key","source":"bangumi","tier":null,"tag_kind":null,"spoiler":"none","is_sexual":false,"work_count":null},
{"id":"4","display_name":"Galgame","source":"bangumi","tier":"hidden","tag_kind":"meta","spoiler":"none","is_sexual":false,"work_count":null},
{"id":"11","display_name":"Island","source":"vndb","tier":null,"tag_kind":"content","spoiler":"none","is_sexual":false,"work_count":9},
{"id":"12","display_name":"Twist","source":"vndb","tier":null,"tag_kind":"content","spoiler":"major","is_sexual":false,"work_count":9},
{"id":"13","display_name":"Sex","source":"vndb","tier":null,"tag_kind":"content","spoiler":"none","is_sexual":true,"work_count":9},
{"id":"11","display_name":"Island","source":"bangumi","tier":null,"tag_kind":"content","spoiler":"minor","is_sexual":false,"work_count":9},
{"id":"14","display_name":"Summer","source":"vndb","tier":null,"tag_kind":"content","spoiler":"none","is_sexual":false,"work_count":9},
{"id":"15","display_name":"Ocean","source":"vndb","tier":null,"tag_kind":"content","spoiler":"none","is_sexual":false,"work_count":9},
{"id":"16","display_name":"Kid","source":"vndb","tier":null,"tag_kind":"content","spoiler":"none","is_sexual":false,"work_count":9},
{"id":"17","display_name":"Pirates","source":"vndb","tier":null,"tag_kind":"content","spoiler":"none","is_sexual":false,"work_count":9},
{"id":"18","display_name":"Village","source":"vndb","tier":null,"tag_kind":"content","spoiler":"none","is_sexual":false,"work_count":9}
]}]}`

func TestTopTagsFollowTheWorkPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(topTagsRow))
	}))
	t.Cleanup(srv.Close)
	rows, appErr := client.New(srv.URL, "nmk_test", "").CatalogRowsByWorkIDs(t.Context(), []int{4}, RowInclude+",tags", "all")
	if appErr != nil {
		t.Fatalf("rows: %v", appErr.Message)
	}
	row, ok := rows[4]
	if !ok {
		t.Fatalf("rows = %v", rows)
	}

	names := func(tags []TagRef) []string {
		out := make([]string, len(tags))
		for i, t := range tags {
			out[i] = string(t.ID) + ":" + t.DisplayName
		}
		return out
	}
	sfw := names(TopTags(&row, false, 5))
	want := []string{"14:Summer", "15:Ocean", "16:Kid", "17:Pirates", "18:Village"}
	if len(sfw) != 5 || sfw[0] != want[0] || sfw[4] != want[4] {
		t.Errorf("sfw top tags = %v, want %v: no source-only tags, no hidden tier, no spoilers (11 is minor once folded), no adult tags", sfw, want)
	}
	nsfw := TopTags(&row, true, 5)
	if len(nsfw) != 5 || nsfw[0].ID != "13" || !nsfw[0].IsSexual {
		t.Errorf("nsfw top tags = %v, want the adult tag first", names(nsfw))
	}
	if got := TopTags(&client.CatalogWorkListItem{}, true, 5); got == nil || len(got) != 0 {
		t.Errorf("a work without tags = %#v, want an empty array", got)
	}
}
