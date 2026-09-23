package app

import (
	"encoding/json"
	"fmt"
	"kun-galgame-api/pkg/catalogclient"
	"net/http"
	"testing"
	"time"

	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/pkg/problem"
)

func TestV1GetWorkStripsSexualTags(t *testing.T) {
	f := newWorkFix(t)
	resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkLive), "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	tags, _ := body["tags"].([]any)
	for _, raw := range tags {
		tag, _ := raw.(map[string]any)
		if tag["is_sexual"] == true {
			t.Fatalf("default GET kept a sexual tag: %+v", tag)
		}
	}
	resp, nsfw := f.wk(t, http.MethodGet, g4WorkPath(g4WorkLive)+"?include_nsfw=true", "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("nsfw %d %+v", resp.StatusCode, nsfw)
	}
	found := false
	for _, raw := range nsfw["tags"].([]any) {
		tag, _ := raw.(map[string]any)
		if tag["is_sexual"] == true {
			found = true
		}
	}
	if !found {
		t.Fatal("include_nsfw=true must show the adult tag")
	}
}

func TestV1GetWorkDropsUnrenderablePeople(t *testing.T) {
	f := newWorkFix(t)
	resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkLive), "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	for _, raw := range body["contributors"].([]any) {
		c, _ := raw.(map[string]any)
		if strID(c["id"]) == idStr(g4UserGoneA) || strID(c["id"]) == idStr(g4UserGoneB) {
			t.Fatalf("unrenderable contributor present: %+v", c)
		}
	}
}

func TestV1GetWorkCreatorUnrenderableIsNull(t *testing.T) {
	f := newWorkFix(t)
	if err := f.db.Exec(`UPDATE galgame SET creator_user_id = ? WHERE id = ?`, g4UserGoneA, g4WorkLive).Error; err != nil {
		t.Fatal(err)
	}
	resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkLive), "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	if body["creator"] != nil {
		t.Fatalf("creator %+v, want null", body["creator"])
	}
}

func TestV1GetWorkIncrementsViewWithoutTouchingUpdated(t *testing.T) {
	f := newWorkFix(t)
	var before time.Time
	if err := f.db.Raw(`SELECT updated FROM galgame WHERE id = ?`, g4WorkLive).Scan(&before).Error; err != nil {
		t.Fatal(err)
	}
	view := f.scalar(t, `SELECT view FROM galgame WHERE id = ?`, g4WorkLive)
	daily := `SELECT COALESCE(SUM(count), 0) FROM galgame_view_daily WHERE entity_id = ? AND day = CURRENT_DATE`
	today := f.scalar(t, daily, g4WorkLive)
	resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkLive), "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	if asInt(body["view_count"]) != view+1 {
		t.Errorf("view_count %v want %d", body["view_count"], view+1)
	}
	var after time.Time
	if err := f.db.Raw(`SELECT updated FROM galgame WHERE id = ?`, g4WorkLive).Scan(&after).Error; err != nil {
		t.Fatal(err)
	}
	if !after.Equal(before) {
		t.Errorf("updated moved %s -> %s", before, after)
	}
	if got := f.scalar(t, daily, g4WorkLive); got != today+1 {
		t.Errorf("daily bucket %d want %d: the 7d/30d rankings read it", got, today+1)
	}
	if resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkNoLocal), "/works/{work_id}", "", nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("no-local get %d %+v", resp.StatusCode, body)
	}
	if got := f.scalar(t, daily, g4WorkNoLocal); got != 0 {
		t.Errorf("a work with no forum row got a daily bucket: %d", got)
	}
}

func TestV1GetWorkCoverVotesFallBackToPublicTallies(t *testing.T) {
	f := newWorkFix(t)
	f.user.covers = []catalogclient.CoverTally{{ID: 11, ImageHash: g4PortraitHash, VoteCount: 3}}
	first := func(label string) map[string]any {
		t.Helper()
		resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkLive), "/works/{work_id}", "sess-alice", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: get %d %+v", label, resp.StatusCode, body)
		}
		covers, _ := body["covers"].([]any)
		if len(covers) == 0 {
			t.Fatalf("%s: no covers %+v", label, body["covers"])
		}
		c, _ := covers[0].(map[string]any)
		return c
	}
	for _, err := range []error{catalogclient.ErrUnauthorized, catalogclient.ErrInsufficientScope} {
		f.user.userCoverErr = err
		c := first(err.Error())
		viewer, _ := c["viewer"].(map[string]any)
		if asInt(c["vote_count"]) != 3 || viewer == nil || viewer["has_voted"] != false {
			t.Errorf("%v: signed-in reader lost the public tallies: %+v", err, c)
		}
	}
	f.user.userCoverErr = nil
	f.user.userCovers = []catalogclient.CoverTally{{ID: 11, ImageHash: g4PortraitHash, VoteCount: 4, Voted: true}}
	c := first("viewer lane")
	viewer, _ := c["viewer"].(map[string]any)
	if asInt(c["vote_count"]) != 4 || viewer == nil || viewer["has_voted"] != true {
		t.Errorf("viewer lane not used: %+v", c)
	}
}

func TestV1GetWorkCoverSexual(t *testing.T) {
	f := newWorkFix(t)
	resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkLive), "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	cover, _ := body["cover"].(map[string]any)
	if cover == nil || cover["sexual"] != nil {
		t.Errorf("unjudged portrait sexual %+v", cover)
	}
	covers, _ := body["covers"].([]any)
	if len(covers) < 2 {
		t.Fatalf("covers %v", body["covers"])
	}
	second, _ := covers[1].(map[string]any)
	img, _ := second["image"].(map[string]any)
	if img["sexual"] != "safe" {
		t.Errorf("judged-0 cover sexual %+v", img)
	}
	if cover["sexual"] != nil && asJSONType(cover["sexual"]) != asJSONType(img["sexual"]) {
		t.Errorf("Work.cover.sexual %+v vs portrait slot", cover["sexual"])
	}
}

func TestV1GetWorkMergedAndMissing(t *testing.T) {
	f := newWorkFix(t)
	resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkMerged), "/works/{work_id}", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeEntityMerged)
	if body["object"] != "work" || strID(body["current_id"]) != idStr(g4WorkLive) {
		t.Errorf("merged %+v", body)
	}
	resp, body = f.wk(t, http.MethodGet, g4WorkPath(g4WorkUnknown), "/works/{work_id}", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.wk(t, http.MethodGet, g4WorkPath(g4WorkHidden), "/works/{work_id}", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.wk(t, http.MethodGet, g4WorkPath(g4WorkNoLocal), "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("no local %d %+v", resp.StatusCode, body)
	}
}

func TestV1WorkRefIsSubsetOfWork(t *testing.T) {
	f := newWorkFix(t)
	row := f.cat.rows[g4WorkLive]
	ref := galgameapiv1.WorkRefOf(t.Context(), &row, geCDN)
	refRaw, err := json.Marshal(ref)
	if err != nil {
		t.Fatal(err)
	}
	var refMap map[string]any
	if err := json.Unmarshal(refRaw, &refMap); err != nil {
		t.Fatal(err)
	}
	resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkLive), "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	for k, v := range refMap {
		got, ok := body[k]
		if !ok {
			t.Errorf("Work missing WorkRef key %s", k)
			continue
		}
		if asJSONType(got) != asJSONType(v) {
			t.Errorf("Work[%s] type %s, WorkRef type %s", k, asJSONType(got), asJSONType(v))
		}
	}
}

func TestV1GetWorkKeepsCatalogsWiderVocabularies(t *testing.T) {
	f := newWorkFix(t)
	resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkLive), "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	kinds := map[string]string{}
	roster, _ := body["roster"].([]any)
	for _, r := range roster {
		c, _ := r.(map[string]any)
		kinds[fmt.Sprint(c["id"])] = fmt.Sprint(c["character_kind"])
	}
	if kinds["7001"] != "main" || kinds["7002"] != "unknown" {
		t.Errorf("roster kinds %v: catalog's roster_role includes unknown (5%% of prod rows) and those characters must stay", kinds)
	}
	aliases, _ := body["aliases"].([]any)
	found := false
	for _, a := range aliases {
		found = found || a == g4LongCJKTitle
	}
	if !found {
		t.Errorf("a 300-character CJK alias (900 bytes) was dropped: %v", aliases)
	}

	resp, body = f.wk(t, http.MethodGet, g4WorkPath(g4WorkNoOwner), "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK || body["content_rating"] != "sensitive" {
		t.Errorf("sensitive work: %d content_rating=%v", resp.StatusCode, body["content_rating"])
	}
}
