package app

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

const (
	g7aSpecOne   = "/work-submissions/{work_id}"
	g7aSpecColl  = "/work-submissions"
	g7aSpecMine  = "/me/work-submissions"
	g7aSpecRevs  = "/me/work-submission-reviews"
	g7aSpecCands = "/work-submission-candidates"
)

func g7aKey(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }

func g7aFirstError(t *testing.T, body map[string]any) (string, string) {
	t.Helper()
	errs, _ := body["errors"].([]any)
	if len(errs) == 0 {
		t.Fatalf("no errors[] in %+v", body)
	}
	e, _ := errs[0].(map[string]any)
	pointer, _ := e["pointer"].(string)
	if pointer == "" {
		pointer, _ = e["parameter"].(string)
	}
	if pointer == "" {
		pointer, _ = e["header"].(string)
	}
	reason, _ := e["reason"].(string)
	return pointer, reason
}

func g7aObj(body map[string]any, key string) map[string]any {
	m, _ := body[key].(map[string]any)
	return m
}

func g7aState(t *testing.T, f *g7aFix, id int64) string {
	t.Helper()
	f.user.lock.Lock()
	defer f.user.lock.Unlock()
	if c := f.user.claims[id]; c != nil {
		return c.state
	}
	return ""
}

func TestV1WorkSubmissionCreate(t *testing.T) {
	f := newG7aFix(t)
	body := g7aMintBody("新作品", map[string]any{
		"aliases":                []string{"别名", " ", "别名"},
		"introductions":          []map[string]any{{"locale": "zh-Hans", "value": "简介"}},
		"release_date":           "2004-04-28",
		"release_date_precision": "month",
	})
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(1), body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, got)
	}
	id := got["id"].(string)
	if loc := resp.Header.Get("Location"); loc != "/api/v1/work-submissions/"+id {
		t.Errorf("Location %q", loc)
	}
	if etag := resp.Header.Get("ETag"); etag != `"c`+id+`.1"` {
		t.Errorf("201 ETag %q, want the claim's own version, not the mint record's", etag)
	}
	if got["display_name"] != "新作品" {
		t.Errorf("display_name %v: the 201 is read back, not the partial mint record", got["display_name"])
	}
	if got["state"] != "pending" || got["is_nsfw"] != true || got["content_rating"] != "r18" || got["has_banner_attached"] != true {
		t.Errorf("body %+v", got)
	}
	if sub := g7aObj(got, "submitter"); sub["name"] != "alice" {
		t.Errorf("submitter %+v", sub)
	}
	if ev := g7aObj(got, "last_event"); ev["to_state"] != "pending" || ev["from_state"] != nil {
		t.Errorf("last_event %+v", ev)
	}
	if v := g7aObj(got, "viewer"); v["can_withdraw"] != true || v["can_submit"] != false || v["can_review"] != false {
		t.Errorf("viewer %+v", v)
	}

	if len(f.user.mints) != 1 {
		t.Fatalf("mints %d", len(f.user.mints))
	}
	m := f.user.mints[0]
	if m.IdempotencyKey != g7aKey(1) {
		t.Errorf("idempotency key %q not forwarded", m.IdempotencyKey)
	}
	if m.Released == nil || m.Released.Y != 2004 || m.Released.M != 4 || m.Released.D != 0 {
		t.Errorf("released %+v, want {2004 4 0} for month precision", m.Released)
	}
	if m.Fields["catalog.work.display_name"] != "新作品" || m.Fields["catalog.work.olang"] != "ja" ||
		m.Fields["catalog.work.content_rating"] != 2 || m.Fields["catalog.work.display_nsfw"] != true {
		t.Errorf("fields %+v", m.Fields)
	}
	titles, _ := m.Fields["catalog.work.titles"].([]any)
	if len(titles) != 2 {
		t.Fatalf("titles %+v: blank and repeated aliases are dropped", titles)
	}
	if alias := titles[1].(map[string]any); alias["lang"] != "" || alias["kind"] != 1 || alias["title"] != "别名" {
		t.Errorf("alias %+v", alias)
	}
	intros, _ := m.Fields["catalog.work.intros"].([]any)
	if len(intros) != 1 || intros[0].(map[string]any)["lang"] != "zh-Hans" {
		t.Errorf("intros %+v", intros)
	}
	var creator int
	workID, _ := strconv.Atoi(id)
	f.db.Raw(`SELECT creator_user_id FROM galgame WHERE id = ? AND published = false`, workID).Scan(&creator)
	if creator != w3UserAlice {
		t.Errorf("local creator %d", creator)
	}
}

func TestV1WorkSubmissionCreateReleaseDays(t *testing.T) {
	f := newG7aFix(t)
	for i, tc := range []struct {
		precision *string
		want      catalogclient.WorkSubmitDate
	}{
		{nil, catalogclient.WorkSubmitDate{Y: 2004, M: 4, D: 28}},
		{g7aStr("year"), catalogclient.WorkSubmitDate{Y: 2004}},
	} {
		extra := map[string]any{"release_date": "2004-04-28"}
		if tc.precision != nil {
			extra["release_date_precision"] = *tc.precision
		}
		key := g7aKey(200 + i)
		resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", key, g7aMintBody("Release"+idStr(i), extra))
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create %d %+v", resp.StatusCode, got)
		}
		if r := f.user.mints[len(f.user.mints)-1].Released; r == nil || *r != tc.want {
			t.Errorf("case %d released %+v, want %+v", i, r, tc.want)
		}
	}
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(4), g7aMintBody("NoDate", nil))
	if resp.StatusCode != http.StatusCreated || f.user.mints[len(f.user.mints)-1].Released != nil {
		t.Fatalf("TBA mint %d %+v", resp.StatusCode, got)
	}
}

func TestV1WorkSubmissionCreateAxesAreIndependent(t *testing.T) {
	f := newG7aFix(t)
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(5),
		g7aMintBody("Axes", map[string]any{"is_nsfw": true, "content_rating": "all_ages"}))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, got)
	}
	m := f.user.mints[0]
	if m.Fields["catalog.work.display_nsfw"] != true || m.Fields["catalog.work.content_rating"] != 0 {
		t.Errorf("fields %+v: the two axes must be written as sent", m.Fields)
	}
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(6),
		g7aMintBody("Axes2", map[string]any{"is_nsfw": false, "content_rating": "r18"}))
	if resp.StatusCode != http.StatusCreated || f.user.mints[1].Fields["catalog.work.display_nsfw"] != false ||
		f.user.mints[1].Fields["catalog.work.content_rating"] != 2 {
		t.Fatalf("r18 + sfw %d %+v %+v", resp.StatusCode, got, f.user.mints[1].Fields)
	}

	noRating := g7aMintBody("Axes3", nil)
	delete(noRating, "content_rating")
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(7), noRating)
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	noNSFW := g7aMintBody("Axes4", nil)
	delete(noNSFW, "is_nsfw")
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(8), noNSFW)
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if len(f.user.mints) != 2 {
		t.Errorf("mints %d after two refusals", len(f.user.mints))
	}
}

func TestV1WorkSubmissionCreateRefusals(t *testing.T) {
	f := newG7aFix(t)
	long := strings.Repeat("字", 501)
	many := make([]string, 100)
	for i := range many {
		many[i] = "alias " + idStr(i)
	}
	for i, tc := range []struct {
		name    string
		body    map[string]any
		pointer string
		reason  string
	}{
		{"blank title", g7aMintBody("   ", nil), "/titles/0/title", problem.ReasonRequired},
		{"long title", g7aMintBody(long, nil), "/titles/0/title", problem.ReasonTooLong},
		{"title locale", g7aMintBody("x", map[string]any{"titles": []map[string]any{{"locale": "ja-jp", "title": "x"}}}), "/titles/0/locale", problem.ReasonUnknownValue},
		{"olang", g7aMintBody("x", map[string]any{"original_language": "xx"}), "/original_language", problem.ReasonUnknownValue},
		{"olang null", g7aMintBody("x", map[string]any{"original_language": nil}), "/original_language", problem.ReasonRequired},
		{"too many", g7aMintBody("x", map[string]any{"aliases": many}), "/titles", problem.ReasonTooManyItems},
		{"intro locale", g7aMintBody("x", map[string]any{"introductions": []map[string]any{{"locale": "fr", "value": "v"}}}), "/introductions/0/locale", problem.ReasonUnknownValue},
		{"intro twice", g7aMintBody("x", map[string]any{"introductions": []map[string]any{{"locale": "ja", "value": "a"}, {"locale": "ja", "value": "b"}}}), "/introductions/1/locale", problem.ReasonDuplicateItem},
		{"year", g7aMintBody("x", map[string]any{"release_date": "1969-12-31"}), "/release_date", problem.ReasonOutOfRange},
		{"precision alone", g7aMintBody("x", map[string]any{"release_date_precision": "year"}), "/release_date_precision", problem.ReasonInconsistentWith},
	} {
		i := i
		t.Run(tc.name, func(t *testing.T) {
			resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(100+i), tc.body)
			wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
			if p, r := g7aFirstError(t, got); p != tc.pointer || r != tc.reason {
				t.Errorf("error %s %s, want %s %s", p, r, tc.pointer, tc.reason)
			}
		})
	}
	if len(f.user.mints) != 0 {
		t.Errorf("refused bodies reached catalog %d times", len(f.user.mints))
	}
}

func TestV1WorkSubmissionCreateFourSlotsAreNotTheShape(t *testing.T) {
	f := newG7aFix(t)
	body := g7aMintBody("Slots", map[string]any{"name_zh_cn": "四语槽", "intro_en_us": "x"})
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(9), body)
	if resp.StatusCode == http.StatusCreated {
		for _, tt := range f.user.mints[0].Fields["catalog.work.titles"].([]any) {
			if tt.(map[string]any)["title"] == "四语槽" {
				t.Fatal("a four-slot name was written as a title")
			}
		}
		if _, ok := f.user.mints[0].Fields["catalog.work.intros"]; ok {
			t.Fatal("a four-slot intro was written")
		}
		return
	}
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
}

func TestV1WorkSubmissionCreateNeedsIdempotencyKey(t *testing.T) {
	f := newG7aFix(t)
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", "", g7aMintBody("NoKey", nil))
	wantCode(t, resp, got, http.StatusBadRequest, problem.CodeInvalidParameter)
	if p, r := g7aFirstError(t, got); p != "Idempotency-Key" || r != problem.ReasonRequired {
		t.Errorf("error %s %s", p, r)
	}
	if len(f.user.mints) != 0 {
		t.Errorf("mint reached catalog without a key")
	}
}

func TestV1WorkSubmissionDuplicateSuspects(t *testing.T) {
	f := newG7aFix(t)
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(10), g7aMintBody("TwinTitle", nil))
	wantCode(t, resp, got, http.StatusConflict, problem.CodeDuplicateSuspects)
	suspects, _ := got["suspects"].([]any)
	if len(suspects) != 1 || suspects[0].(map[string]any)["id"] != idStr(g7aTwin) || suspects[0].(map[string]any)["display_name"] != "TwinTitle" {
		t.Errorf("suspects %+v", got["suspects"])
	}
	if d, _ := got["detail"].(string); strings.Contains(d, "confirm_duplicates") || strings.Contains(d, "re-send") {
		t.Errorf("catalog's sentence leaked into detail: %q", d)
	}
	if len(f.user.claims) != 14 {
		t.Errorf("a refused mint wrote a claim")
	}
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(11),
		g7aMintBody("TwinTitle", map[string]any{"is_duplicate_confirmed": true}))
	if resp.StatusCode != http.StatusCreated || !f.user.mints[1].ConfirmDuplicates {
		t.Fatalf("confirmed mint %d %+v", resp.StatusCode, got)
	}
}

func TestV1WorkSubmissionCreateTrustedLandsLive(t *testing.T) {
	f := newG7aFix(t)
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-trusted", g7aKey(12), g7aMintBody("Trusted", nil))
	if resp.StatusCode != http.StatusCreated || got["state"] != "live" {
		t.Fatalf("trusted mint %d %+v", resp.StatusCode, got)
	}
}

func TestV1WorkSubmissionCreateBanner(t *testing.T) {
	f := newG7aFix(t)
	hash := strings.Repeat("ab", 32)
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(13), g7aMintBody("Banner", map[string]any{"banner_hash": hash}))
	if resp.StatusCode != http.StatusCreated || got["has_banner_attached"] != true {
		t.Fatalf("banner %d %+v", resp.StatusCode, got)
	}
	covers, _ := f.user.proposals[0].Patch["catalog.work.covers"].([]any)
	if len(covers) != 1 || len(covers[0].(map[string]any)) != 1 || covers[0].(map[string]any)["image_hash"] != hash {
		t.Errorf("cover patch %+v: only image_hash may be sent", f.user.proposals[0].Patch)
	}
	if len(f.user.merges) != 1 || f.user.merges[0].state != "merge" || f.user.merges[0].ifMatch != `"p1"` {
		t.Errorf("merges %+v", f.user.merges)
	}

	f.user.mergeErr = errors.New("merge refused")
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(14), g7aMintBody("Banner2", map[string]any{"banner_hash": hash}))
	if resp.StatusCode != http.StatusCreated || got["has_banner_attached"] != false {
		t.Fatalf("failed merge %d %+v: the submission stands", resp.StatusCode, got)
	}

	f.user.mergeErr = nil
	f.user.mergeClosesAs = "declined"
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(600), g7aMintBody("BannerDeclined", map[string]any{"banner_hash": hash}))
	if resp.StatusCode != http.StatusCreated || got["has_banner_attached"] != false {
		t.Fatalf("merge closed as declined %d %+v: catalog did not attach the cover", resp.StatusCode, got)
	}

	f.user.mergeClosesAs = ""
	f.user.autoMerge = true
	merges := len(f.user.merges)
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(15), g7aMintBody("Banner3", map[string]any{"banner_hash": hash}))
	if resp.StatusCode != http.StatusCreated || got["has_banner_attached"] != true || len(f.user.merges) != merges {
		t.Fatalf("automerged banner %d %+v merges %d", resp.StatusCode, got, len(f.user.merges))
	}
}

func TestV1WorkSubmissionCreateUpstreamFailures(t *testing.T) {
	f := newG7aFix(t)
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-noscope", g7aKey(16), g7aMintBody("NoScope", nil))
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeScopeRequired)
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-429", g7aKey(17), g7aMintBody("Busy", nil))
	wantCode(t, resp, got, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	if resp.Header.Get("Retry-After") != "30" {
		t.Errorf("Retry-After %q", resp.Header.Get("Retry-After"))
	}
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-banned", g7aKey(18), g7aMintBody("Banned", nil))
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeAccountBanned)
}

func TestV1WorkSubmissionCreateTrustDeny(t *testing.T) {
	f := newG7aFixWith(t, denyChecker{})
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(19), g7aMintBody("Denied", nil))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeContentRejected)
	if len(f.user.mints) != 0 {
		t.Error("a refused text reached catalog")
	}
}

func TestV1WorkSubmissionGet(t *testing.T) {
	f := newG7aFix(t)
	resp, got := f.call(t, http.MethodGet, g7aSub(g7aDraft), g7aSpecOne, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || got["state"] != "draft" {
		t.Fatalf("own draft %d %+v", resp.StatusCode, got)
	}
	if etag := resp.Header.Get("ETag"); etag != `"c948000001.1"` {
		t.Errorf("ETag %q, want the upstream validator", etag)
	}
	if v := g7aObj(got, "viewer"); v["can_delete"] != true || v["can_submit"] != true || v["can_withdraw"] != false {
		t.Errorf("viewer %+v", v)
	}

	resp, got = f.call(t, http.MethodGet, g7aSub(g7aDeclined), g7aSpecOne, "sess-alice", "", nil)
	ev := g7aObj(got, "last_event")
	if resp.StatusCode != http.StatusOK || ev["note"] != "titles are wrong" || ev["from_state"] != "pending" || g7aObj(ev, "actor")["name"] != "staff" {
		t.Fatalf("declined %d %+v", resp.StatusCode, got)
	}
	if got["first_acted_at"] == nil || got["acted_count"] != float64(1) {
		t.Errorf("acted %v %v", got["first_acted_at"], got["acted_count"])
	}

	resp, raw := f.doJSON(t, http.MethodGet, g7aSub(g7aBobPending), "sess-alice", g7aSpecOne, "", nil, nil)
	wantCode(t, resp, problemMap(t, raw), http.StatusNotFound, problem.CodeNotFound)
	if strings.Contains(string(raw), "pending") || strings.Contains(string(raw), "BobPendingG7") {
		t.Errorf("someone else's claim leaked: %s", raw)
	}
	resp, got = f.call(t, http.MethodGet, g7aSub(948000999), g7aSpecOne, "sess-alice", "", nil)
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)

	resp, got = f.call(t, http.MethodGet, g7aSub(g7aBobPending), g7aSpecOne, "sess-staff", "", nil)
	if resp.StatusCode != http.StatusOK || got["state"] != "pending" {
		t.Fatalf("reviewer read %d %+v", resp.StatusCode, got)
	}
	if v := g7aObj(got, "viewer"); v["can_review"] != true || v["can_withdraw"] != false {
		t.Errorf("reviewer viewer %+v", v)
	}
	if sub := g7aObj(got, "submitter"); sub["name"] != "bob" {
		t.Errorf("submitter %+v", sub)
	}

	resp, got = f.call(t, http.MethodGet, g7aSub(g7aHiddenBare), g7aSpecOne, "sess-staff", "", nil)
	if resp.StatusCode != http.StatusOK || got["state"] != "hidden" || got["last_event"] != nil {
		t.Fatalf("hidden bare %d %+v", resp.StatusCode, got)
	}
	if sub := g7aObj(got, "submitter"); sub == nil || sub["name"] != nil {
		t.Errorf("banned submitter %+v, want a deleted user ref", sub)
	}

	snaps := f.user.snapshots
	resp, got = f.call(t, http.MethodGet, g7aSub(g7aHiddenLive), g7aSpecOne, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || got["state"] != "hidden" || got["content_rating"] != "all_ages" || f.user.snapshots != snaps+1 {
		t.Fatalf("own hidden %d %+v snapshots %d", resp.StatusCode, got, f.user.snapshots-snaps)
	}
	resp, got = f.call(t, http.MethodGet, g7aSub(g7aPending), g7aSpecOne, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || got["is_nsfw"] != true {
		t.Fatalf("nsfw pending %d %+v", resp.StatusCode, got)
	}
}

func TestV1WorkSubmissionBearerHoldsNoReviewPower(t *testing.T) {
	f := newG7aFix(t)
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	resp, got := f.callWith(t, http.MethodGet, g7aSub(g7aStaffOwn), g7aSpecOne, "", bearer, nil)
	if resp.StatusCode != http.StatusOK || g7aObj(got, "viewer")["can_review"] != false {
		t.Fatalf("bearer own read %d %+v", resp.StatusCode, got)
	}
	resp, got = f.callWith(t, http.MethodGet, g7aSub(g7aBobPending), g7aSpecOne, "", bearer, nil)
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
	resp, got = f.callWith(t, http.MethodPatch, g7aSub(g7aBobPending), g7aSpecOne, "", bearer, map[string]any{"state": "live"})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
	for _, path := range []string{"/api/v1/work-submissions", "/api/v1/me/work-submission-reviews"} {
		spec := g7aSpecColl
		if strings.HasPrefix(path, "/api/v1/me") {
			spec = g7aSpecRevs
		}
		resp, got = f.callWith(t, http.MethodGet, path, spec, "", bearer, nil)
		wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
	}
	if f.user.modCalls != 0 || len(f.user.decisions) != 0 {
		t.Errorf("a Bearer request reached the moderation face %d times", f.user.modCalls)
	}
	if g7aState(t, f, g7aBobPending) != "pending" {
		t.Error("bob's claim moved")
	}
}

func TestV1WorkSubmissionSubmitterMoves(t *testing.T) {
	f := newG7aFix(t)
	resp, got := f.call(t, http.MethodPatch, g7aSub(g7aDraft), g7aSpecOne, "sess-alice", "", map[string]any{"state": "pending"})
	if resp.StatusCode != http.StatusOK || got["state"] != "pending" || resp.Header.Get("ETag") != `"c948000001.2"` {
		t.Fatalf("submit draft %d %+v %q", resp.StatusCode, got, resp.Header.Get("ETag"))
	}
	if p := f.user.patches[0]; p.state != "pending" || p.ifMatch != "*" {
		t.Errorf("patch %+v: no If-Match sent means *", p)
	}
	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aDeclined), g7aSpecOne, "sess-alice", "", map[string]any{"state": "pending"})
	if resp.StatusCode != http.StatusOK || got["state"] != "pending" {
		t.Fatalf("resubmit %d %+v", resp.StatusCode, got)
	}
	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aLive), g7aSpecOne, "sess-alice", "", map[string]any{"state": "draft"})
	if resp.StatusCode != http.StatusOK || got["state"] != "draft" || f.user.patches[2].state != "withdrawn" {
		t.Fatalf("withdraw %d %+v %+v", resp.StatusCode, got, f.user.patches)
	}

	n := len(f.user.patches)
	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aDraft), g7aSpecOne, "sess-alice", "", map[string]any{"state": "pending"})
	wantCode(t, resp, got, http.StatusConflict, problem.CodeInvalidStateTransition)
	if d, _ := got["detail"].(string); !strings.Contains(d, "pending") || !strings.Contains(d, "draft") || strings.Contains(d, "cannot submit") {
		t.Errorf("detail %q", d)
	}
	if len(f.user.patches) != n {
		t.Error("a refused move reached catalog")
	}

	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aDraftRes), g7aSpecOne, "sess-bob", "", map[string]any{"state": "pending"})
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aDraftRes), g7aSpecOne, "sess-alice", "", map[string]any{"state": "live"})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
	if g7aState(t, f, g7aDraftRes) != "draft" {
		t.Error("draft moved")
	}
}

func TestV1WorkSubmissionIfMatch(t *testing.T) {
	f := newG7aFix(t)
	stale := http.Header{"If-Match": {`"c948000001.0"`}}
	resp, got := f.callWith(t, http.MethodPatch, g7aSub(g7aDraft), g7aSpecOne, "sess-alice", stale, map[string]any{"state": "pending"})
	wantCode(t, resp, got, http.StatusPreconditionFailed, problem.CodePreconditionFailed)
	if f.user.patches[0].ifMatch != `"c948000001.0"` || g7aState(t, f, g7aDraft) != "draft" {
		t.Errorf("patch %+v", f.user.patches)
	}

	resp, _ = f.call(t, http.MethodGet, g7aSub(g7aDraft), g7aSpecOne, "sess-alice", "", nil)
	current := http.Header{"If-Match": {resp.Header.Get("ETag")}}
	resp, got = f.callWith(t, http.MethodPatch, g7aSub(g7aDraft), g7aSpecOne, "sess-alice", current, map[string]any{"state": "pending"})
	if resp.StatusCode != http.StatusOK || f.user.patches[1].ifMatch != `"c948000001.1"` {
		t.Fatalf("current validator %d %+v %+v", resp.StatusCode, got, f.user.patches)
	}

	resp, got = f.callWith(t, http.MethodPatch, g7aSub(g7aBobPending), g7aSpecOne, "sess-staff", http.Header{"If-Match": {`"stale"`}}, map[string]any{"state": "live"})
	wantCode(t, resp, got, http.StatusPreconditionFailed, problem.CodePreconditionFailed)
	resp, got = f.callWith(t, http.MethodDelete, g7aSub(g7aDraftRes), g7aSpecOne, "sess-alice", http.Header{"If-Match": {`"stale"`}}, nil)
	wantCode(t, resp, got, http.StatusPreconditionFailed, problem.CodePreconditionFailed)
	if g7aState(t, f, g7aBobPending) != "pending" || g7aState(t, f, g7aDraftRes) != "draft" {
		t.Error("a stale write landed")
	}
}

func TestV1WorkSubmissionReview(t *testing.T) {
	f := newG7aFix(t)
	resp, got := f.call(t, http.MethodPatch, g7aSub(g7aBobPending), g7aSpecOne, "sess-staff", "", map[string]any{"state": "live"})
	if resp.StatusCode != http.StatusOK || got["state"] != "live" || f.user.decisions[0].state != "approve" || f.user.decisions[0].ifMatch != "*" {
		t.Fatalf("approve %d %+v %+v", resp.StatusCode, got, f.user.decisions)
	}

	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aDeclined), g7aSpecOne, "sess-staff", "", map[string]any{"state": "live"})
	wantCode(t, resp, got, http.StatusConflict, problem.CodeInvalidStateTransition)
	if d, _ := got["detail"].(string); !strings.Contains(d, "declined") {
		t.Errorf("detail %q", d)
	}
	if len(f.user.decisions) != 1 || g7aState(t, f, g7aDeclined) != "declined" {
		t.Error("an illegal decision reached catalog")
	}

	for _, note := range []any{nil, "", "   "} {
		body := map[string]any{"state": "declined"}
		if note != nil {
			body["note"] = note
		}
		resp, got = f.call(t, http.MethodPatch, g7aSub(g7aStaffOwn), g7aSpecOne, "sess-staff", "", body)
		wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
		if p, r := g7aFirstError(t, got); p != "/note" || r != problem.ReasonRequired {
			t.Errorf("note %v: %s %s", note, p, r)
		}
	}
	if len(f.user.decisions) != 1 {
		t.Error("a blank decline reached catalog")
	}
	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aStaffOwn), g7aSpecOne, "sess-staff", "", map[string]any{"state": "declined", "note": " 缺少简介 "})
	if resp.StatusCode != http.StatusOK || got["state"] != "declined" || g7aObj(got, "last_event")["note"] != "缺少简介" {
		t.Fatalf("decline %d %+v", resp.StatusCode, got)
	}
	if v := g7aObj(got, "viewer"); v["can_submit"] != true {
		t.Errorf("the reviewer submitted this claim, viewer %+v", v)
	}

	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aLive), g7aSpecOne, "sess-staff", "", map[string]any{"state": "hidden"})
	if resp.StatusCode != http.StatusOK || got["state"] != "hidden" {
		t.Fatalf("ban %d %+v", resp.StatusCode, got)
	}

	for _, tc := range []struct {
		id   int
		want string
	}{{g7aHiddenDraf, "draft"}, {g7aHiddenLive, "live"}, {g7aHiddenBare, "live"}} {
		resp, got = f.call(t, http.MethodPatch, g7aSub(tc.id), g7aSpecOne, "sess-staff", "", map[string]any{"state": "unban"})
		if resp.StatusCode != http.StatusOK || got["state"] != tc.want {
			t.Fatalf("unban %d: %d %+v, want %s", tc.id, resp.StatusCode, got, tc.want)
		}
	}

	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aPending), g7aSpecOne, "sess-alice", "", map[string]any{"state": "hidden"})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
}

func TestV1WorkSubmissionReviewReadsTheOutcomeBack(t *testing.T) {
	f := newG7aFix(t)
	f.user.forceState[g7aBobPending] = "hidden"
	resp, got := f.call(t, http.MethodPatch, g7aSub(g7aBobPending), g7aSpecOne, "sess-staff", "", map[string]any{"state": "live"})
	if resp.StatusCode != http.StatusOK || got["state"] != "hidden" {
		t.Fatalf("approve %d %+v, want catalog's read-back state", resp.StatusCode, got)
	}
}

func TestV1WorkSubmissionDelete(t *testing.T) {
	f := newG7aFix(t)
	resp, got := f.call(t, http.MethodDelete, g7aSub(g7aDraft), g7aSpecOne, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d %+v", resp.StatusCode, got)
	}
	if f.user.deletes[0].ifMatch != "*" || g7aState(t, f, g7aDraft) != "" {
		t.Errorf("deletes %+v", f.user.deletes)
	}
	var n int64
	f.db.Raw(`SELECT count(*) FROM galgame WHERE id = ?`, g7aDraft).Scan(&n)
	if n != 0 {
		t.Error("local draft row not cleaned")
	}

	resp, got = f.call(t, http.MethodDelete, g7aSub(g7aDraftRes), g7aSpecOne, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusNoContent || g7aState(t, f, g7aDraftRes) != "" {
		t.Fatalf("delete with resource %d %+v", resp.StatusCode, got)
	}
	f.db.Raw(`SELECT count(*) FROM galgame g JOIN galgame_resource r ON r.work_id = g.id WHERE g.id = ?`, g7aDraftRes).Scan(&n)
	if n != 1 {
		t.Errorf("a page carrying a resource was deleted with it (%d)", n)
	}

	calls := len(f.user.deletes)
	for _, id := range []int{g7aPending, g7aLive} {
		resp, got = f.call(t, http.MethodDelete, g7aSub(id), g7aSpecOne, "sess-alice", "", nil)
		wantCode(t, resp, got, http.StatusConflict, problem.CodeInvalidStateTransition)
		f.db.Raw(`SELECT count(*) FROM galgame WHERE id = ?`, id).Scan(&n)
		if n != 1 {
			t.Errorf("%d: local row gone", id)
		}
	}
	if len(f.user.deletes) != calls || g7aState(t, f, g7aPending) != "pending" || g7aState(t, f, g7aLive) != "live" {
		t.Error("a non-draft delete reached catalog")
	}
	resp, got = f.call(t, http.MethodDelete, g7aSub(g7aWizDraft), g7aSpecOne, "sess-alice", "", nil)
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
}

func g7aWalk(t *testing.T, f *g7aFix, base, spec, session string) []string {
	t.Helper()
	return g7aWalkBy(t, f, base, spec, session, adminItemIDs)
}

func g7aCandidateIDs(body map[string]any) []string {
	var out []string
	for _, it := range body["items"].([]any) {
		out = append(out, g7aObj(it.(map[string]any), "work_summary")["id"].(string))
	}
	return out
}

func g7aWalkBy(t *testing.T, f *g7aFix, base, spec, session string, ids func(map[string]any) []string) []string {
	t.Helper()
	var out []string
	next := ""
	for range 50 {
		url := base
		if next != "" {
			url += "&cursor=" + next
		}
		resp, got := f.call(t, http.MethodGet, url, spec, session, "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("walk %s: %d %+v", url, resp.StatusCode, got)
		}
		out = append(out, ids(got)...)
		c, _ := got["next_cursor"].(string)
		if c == "" {
			return out
		}
		next = c
	}
	t.Fatal("walk did not end")
	return nil
}

func TestV1WorkSubmissionListMine(t *testing.T) {
	f := newG7aFix(t)
	got := g7aWalk(t, f, "/api/v1/me/work-submissions?state=pending,declined,draft&limit=2", g7aSpecMine, "sess-alice")
	want := []string{idStr(g7aDraft), idStr(g7aPending), idStr(g7aDeclined), idStr(g7aDraftRes)}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("walk %v, want %v", got, want)
	}
	all := g7aWalk(t, f, "/api/v1/me/work-submissions?limit=3", g7aSpecMine, "sess-alice")
	if len(all) != 7 || len(slices.Compact(slices.Sorted(slices.Values(all)))) != 7 {
		t.Errorf("all states %v", all)
	}

	resp, body := f.call(t, http.MethodGet, "/api/v1/me/work-submissions?state=declined&include_total=true", g7aSpecMine, "sess-alice", "", nil)
	items, _ := body["items"].([]any)
	if resp.StatusCode != http.StatusOK || body["total"] != float64(1) || len(items) != 1 {
		t.Fatalf("declined %d %+v", resp.StatusCode, body)
	}
	row := items[0].(map[string]any)
	if g7aObj(row, "last_event")["note"] != "titles are wrong" || row["first_acted_at"] == nil ||
		g7aObj(row, "work_summary")["id"] != idStr(g7aDeclined) || g7aObj(row, "viewer")["can_submit"] != true {
		t.Errorf("row %+v", row)
	}

	resp, body = f.call(t, http.MethodGet, "/api/v1/me/work-submissions?state=approved", g7aSpecMine, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownEnumValue)

	resp, body = f.call(t, http.MethodGet, "/api/v1/me/work-submissions?state=draft&limit=1", g7aSpecMine, "sess-alice", "", nil)
	cur, _ := body["next_cursor"].(string)
	if resp.StatusCode != http.StatusOK || cur == "" {
		t.Fatalf("first page %d %+v", resp.StatusCode, body)
	}
	resp, body = f.call(t, http.MethodGet, "/api/v1/me/work-submissions?state=pending&limit=1&cursor="+cur, g7aSpecMine, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidCursor)
}

func TestV1WorkSubmissionListReviewsAndQueue(t *testing.T) {
	f := newG7aFix(t)
	reviewed := g7aWalk(t, f, "/api/v1/me/work-submission-reviews?limit=2", g7aSpecRevs, "sess-staff")
	want := []string{idStr(g7aDeclined), idStr(g7aHiddenLive), idStr(g7aHiddenDraf), idStr(g7aLive), idStr(g7aWizDecl)}
	slices.Sort(reviewed)
	slices.Sort(want)
	if !slices.Equal(reviewed, want) {
		t.Errorf("reviewed %v, want %v", reviewed, want)
	}
	resp, body := f.call(t, http.MethodGet, "/api/v1/me/work-submission-reviews", g7aSpecRevs, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)

	searches := len(f.cat.searched)
	queue := g7aWalk(t, f, "/api/v1/work-submissions?limit=2", g7aSpecColl, "sess-staff")
	want = []string{idStr(g7aPending), idStr(g7aBobPending), idStr(g7aStaffOwn), idStr(g7aGhostOwner)}
	slices.Sort(queue)
	slices.Sort(want)
	if !slices.Equal(queue, want) {
		t.Errorf("queue %v, want %v", queue, want)
	}
	if f.user.modCalls == 0 || len(f.cat.searched) != searches {
		t.Errorf("queue read %d moderation calls and %d app-key searches", f.user.modCalls, len(f.cat.searched)-searches)
	}
	resp, body = f.call(t, http.MethodGet, "/api/v1/work-submissions?state=hidden&include_total=true", g7aSpecColl, "sess-staff", "", nil)
	items, _ := body["items"].([]any)
	if resp.StatusCode != http.StatusOK || body["total"] != float64(3) || len(items) != 3 {
		t.Fatalf("hidden queue %d %+v", resp.StatusCode, body)
	}
	for _, it := range items {
		row := it.(map[string]any)
		if row["last_event"] != nil || row["first_acted_at"] != nil || g7aObj(row, "work_summary")["display_name"] == "" {
			t.Errorf("hidden row %+v", row)
		}
	}
	resp, body = f.call(t, http.MethodGet, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
}

func TestV1WorkSubmissionCandidates(t *testing.T) {
	f := newG7aFix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/work-submission-candidates?q=Wizard", g7aSpecCands, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("candidates %d %+v", resp.StatusCode, body)
	}
	if _, ok := body["total"]; ok {
		t.Error("a filtered page must not carry total")
	}
	states := map[string]string{}
	for _, it := range body["items"].([]any) {
		row := it.(map[string]any)
		states[g7aObj(row, "work_summary")["id"].(string)] = row["state"].(string)
	}
	if states[idStr(g7aWizSFW)] != "none" || states[idStr(g7aWizDraft)] != "draft" || len(states) != 2 {
		t.Errorf("SFW candidates %v", states)
	}
	q := f.cat.searched[len(f.cat.searched)-1]
	if q.Get("content_limit") != "sfw" || q.Get("nsfw") != "true" {
		t.Errorf("search query %v: the display gate belongs on the request", q)
	}

	resp, body = f.call(t, http.MethodGet, "/api/v1/work-submission-candidates?q=Wizard&include_nsfw=true", g7aSpecCands, "sess-alice", "", nil)
	found := false
	for _, it := range body["items"].([]any) {
		if g7aObj(it.(map[string]any), "work_summary")["id"] == idStr(g7aWizNSFW) {
			found = true
		}
	}
	if resp.StatusCode != http.StatusOK || !found {
		t.Fatalf("include_nsfw %d %+v", resp.StatusCode, body)
	}

	f.works.lag = true
	resp, body = f.call(t, http.MethodGet, "/api/v1/work-submission-candidates?q=Wizard", g7aSpecCands, "sess-alice", "", nil)
	for _, it := range body["items"].([]any) {
		if g7aObj(it.(map[string]any), "work_summary")["id"] == idStr(g7aWizNSFW) {
			t.Fatal("an NSFW work passed a lagging index")
		}
	}

	resp, body = f.call(t, http.MethodGet, "/api/v1/work-submission-candidates?q=%20%20", g7aSpecCands, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)
}

func TestV1WorkSubmissionCandidatesPaging(t *testing.T) {
	f := newG7aFix(t)
	got := g7aWalkBy(t, f, "/api/v1/work-submission-candidates?q=Wizard&include_nsfw=true&limit=1", g7aSpecCands, "sess-alice", g7aCandidateIDs)
	slices.Sort(got)
	if len(slices.Compact(got)) != 3 || len(got) != 3 {
		t.Errorf("paged candidates %v", got)
	}
}

func TestV1WorkSubmissionReadsWriteNothing(t *testing.T) {
	f := newG7aFix(t)
	var before int64
	f.db.Raw(`SELECT count(*) FROM galgame WHERE id BETWEEN 948000001 AND 948000999`).Scan(&before)
	for _, c := range []struct{ path, spec, session string }{
		{g7aSub(g7aDraft), g7aSpecOne, "sess-alice"},
		{g7aSub(g7aBobPending), g7aSpecOne, "sess-staff"},
		{g7aSub(g7aHiddenBare), g7aSpecOne, "sess-staff"},
		{"/api/v1/me/work-submissions", g7aSpecMine, "sess-alice"},
		{"/api/v1/me/work-submission-reviews", g7aSpecRevs, "sess-staff"},
		{"/api/v1/work-submissions", g7aSpecColl, "sess-staff"},
		{"/api/v1/work-submission-candidates?q=Wizard", g7aSpecCands, "sess-alice"},
	} {
		resp, body := f.call(t, http.MethodGet, c.path, c.spec, c.session, "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s %d %+v", c.path, resp.StatusCode, body)
		}
	}
	var after int64
	f.db.Raw(`SELECT count(*) FROM galgame WHERE id BETWEEN 948000001 AND 948000999`).Scan(&after)
	u := f.user
	if after != before || len(u.patches)+len(u.decisions)+len(u.deletes)+len(u.mints)+len(u.proposals)+len(u.merges) != 0 {
		t.Errorf("reads wrote: rows %d→%d, calls %d/%d/%d/%d/%d/%d", before, after,
			len(u.patches), len(u.decisions), len(u.deletes), len(u.mints), len(u.proposals), len(u.merges))
	}
	for id, c := range u.claims {
		if c.version != 1 {
			t.Errorf("claim %d moved during reads", id)
		}
	}
}

func TestV1WorkSubmissionCreateCountsCharacters(t *testing.T) {
	f := newG7aFix(t)
	title := strings.Repeat("字", 500)
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(300),
		g7aMintBody(title, map[string]any{"aliases": []string{strings.Repeat("别", 500)}}))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("500 CJK characters %d %+v: limits count characters, not bytes", resp.StatusCode, got)
	}
}

func TestV1WorkSubmissionDeclaredCodes(t *testing.T) {
	f := newG7aFix(t)
	one := g7aSub(g7aDraft)
	for _, tc := range []struct {
		name, method, url, spec, session string
		body                             any
		status                           int
		code                             string
	}{
		{"get no scope", http.MethodGet, one, g7aSpecOne, "sess-noscope", nil, 403, problem.CodeScopeRequired},
		{"get banned", http.MethodGet, one, g7aSpecOne, "sess-banned", nil, 403, problem.CodeAccountBanned},
		{"get busy", http.MethodGet, one, g7aSpecOne, "sess-429", nil, 503, problem.CodeServiceUnavailable},
		{"get anonymous", http.MethodGet, one, g7aSpecOne, "", nil, 401, problem.CodeMissingCredential},
		{"patch no scope", http.MethodPatch, one, g7aSpecOne, "sess-noscope", map[string]any{"state": "pending"}, 403, problem.CodeScopeRequired},
		{"patch banned", http.MethodPatch, one, g7aSpecOne, "sess-banned", map[string]any{"state": "pending"}, 403, problem.CodeAccountBanned},
		{"patch busy", http.MethodPatch, one, g7aSpecOne, "sess-429", map[string]any{"state": "pending"}, 503, problem.CodeServiceUnavailable},
		{"patch unknown state", http.MethodPatch, one, g7aSpecOne, "sess-alice", map[string]any{"state": "approved"}, 422, problem.CodeValidationFailed},
		{"delete no scope", http.MethodDelete, one, g7aSpecOne, "sess-noscope", nil, 403, problem.CodeScopeRequired},
		{"delete banned", http.MethodDelete, one, g7aSpecOne, "sess-banned", nil, 403, problem.CodeAccountBanned},
		{"delete busy", http.MethodDelete, one, g7aSpecOne, "sess-429", nil, 503, problem.CodeServiceUnavailable},
		{"mine no scope", http.MethodGet, "/api/v1/me/work-submissions", g7aSpecMine, "sess-noscope", nil, 403, problem.CodeScopeRequired},
		{"mine banned", http.MethodGet, "/api/v1/me/work-submissions", g7aSpecMine, "sess-banned", nil, 403, problem.CodeAccountBanned},
		{"mine busy", http.MethodGet, "/api/v1/me/work-submissions", g7aSpecMine, "sess-429", nil, 503, problem.CodeServiceUnavailable},
		{"mine limit", http.MethodGet, "/api/v1/me/work-submissions?limit=101", g7aSpecMine, "sess-alice", nil, 400, problem.CodeLimitTooLarge},
		{"mine cursor", http.MethodGet, "/api/v1/me/work-submissions?cursor=cur_garbage", g7aSpecMine, "sess-alice", nil, 400, problem.CodeInvalidCursor},
		{"reviews limit", http.MethodGet, "/api/v1/me/work-submission-reviews?limit=101", g7aSpecRevs, "sess-staff", nil, 400, problem.CodeLimitTooLarge},
		{"reviews state", http.MethodGet, "/api/v1/me/work-submission-reviews?state=none", g7aSpecRevs, "sess-staff", nil, 400, problem.CodeUnknownEnumValue},
		{"reviews cursor", http.MethodGet, "/api/v1/me/work-submission-reviews?cursor=cur_garbage", g7aSpecRevs, "sess-staff", nil, 400, problem.CodeInvalidCursor},
		{"reviews banned", http.MethodGet, "/api/v1/me/work-submission-reviews", g7aSpecRevs, "sess-banned", nil, 403, problem.CodeAccountBanned},
		{"queue limit", http.MethodGet, "/api/v1/work-submissions?limit=101", g7aSpecColl, "sess-staff", nil, 400, problem.CodeLimitTooLarge},
		{"queue state", http.MethodGet, "/api/v1/work-submissions?state=approved", g7aSpecColl, "sess-staff", nil, 400, problem.CodeUnknownEnumValue},
		{"queue cursor", http.MethodGet, "/api/v1/work-submissions?cursor=cur_garbage", g7aSpecColl, "sess-staff", nil, 400, problem.CodeInvalidCursor},
		{"queue banned", http.MethodGet, "/api/v1/work-submissions", g7aSpecColl, "sess-banned", nil, 403, problem.CodeAccountBanned},
		{"candidates limit", http.MethodGet, "/api/v1/work-submission-candidates?q=W&limit=101", g7aSpecCands, "sess-alice", nil, 400, problem.CodeLimitTooLarge},
		{"candidates cursor", http.MethodGet, "/api/v1/work-submission-candidates?q=W&cursor=cur_garbage", g7aSpecCands, "sess-alice", nil, 400, problem.CodeInvalidCursor},
		{"candidates banned", http.MethodGet, "/api/v1/work-submission-candidates?q=W", g7aSpecCands, "sess-banned", nil, 403, problem.CodeAccountBanned},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, got := f.call(t, tc.method, tc.url, tc.spec, tc.session, "", tc.body)
			wantCode(t, resp, got, tc.status, tc.code)
		})
	}

	f.cat.fail.Store(true)
	resp, got := f.call(t, http.MethodGet, "/api/v1/work-submission-candidates?q=W", g7aSpecCands, "sess-alice", "", nil)
	wantCode(t, resp, got, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	f.cat.fail.Store(false)

	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(400), g7aMintBody("ExistsTitle", nil))
	wantCode(t, resp, got, http.StatusConflict, problem.CodeAlreadyExists)
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(401), g7aMintBody("InflightTitle", nil))
	wantCode(t, resp, got, http.StatusConflict, problem.CodeIdempotencyRequestInProgress)

	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(402), g7aMintBody("Reused", nil))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first %d %+v", resp.StatusCode, got)
	}
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(402), g7aMintBody("Reused2", nil))
	wantCode(t, resp, got, http.StatusConflict, problem.CodeIdempotencyKeyReused)
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(402), g7aMintBody("Reused", nil))
	if resp.StatusCode != http.StatusCreated || resp.Header.Get("Idempotency-Replayed") != "true" || len(f.user.mints) != 3 {
		t.Errorf("replay %d %q mints %d", resp.StatusCode, resp.Header.Get("Idempotency-Replayed"), len(f.user.mints))
	}
}

func TestV1WorkSubmissionCreatedETagIsTheWriteVersion(t *testing.T) {
	f := newG7aFix(t)
	resp, got := f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(500), g7aMintBody("Versioned", nil))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, got)
	}
	id, _ := strconv.Atoi(got["id"].(string))
	resp, got = f.callWith(t, http.MethodPatch, g7aSub(id), g7aSpecOne, "sess-alice",
		http.Header{"If-Match": {resp.Header.Get("ETag")}}, map[string]any{"state": "draft"})
	if resp.StatusCode != http.StatusOK || got["state"] != "draft" {
		t.Fatalf("withdraw with the 201 ETag %d %+v", resp.StatusCode, got)
	}
}

func TestV1WorkSubmissionWritesSurviveAFailedReadBack(t *testing.T) {
	f := newG7aFix(t)
	f.user.failAfterWrite = true
	f.user.onWrite = func() { f.failOA.Store(true) }
	reset := func() {
		f.failOA.Store(false)
		f.user.lock.Lock()
		f.user.readsFail = false
		f.user.lock.Unlock()
	}

	resp, got := f.call(t, http.MethodPatch, g7aSub(g7aGhostOwner), g7aSpecOne, "sess-staff", "", map[string]any{"state": "live"})
	if resp.StatusCode != http.StatusOK || got["state"] != "live" || g7aObj(got, "last_event")["to_state"] != "live" {
		t.Fatalf("approve %d %+v: the decision landed, so the answer is 200 from its record", resp.StatusCode, got)
	}
	if sub := g7aObj(got, "submitter"); sub == nil || sub["name"] != nil {
		t.Errorf("submitter %+v, want a deleted-user ref when the lookup fails after the write", sub)
	}
	if len(f.user.decisions) != 1 || resp.Header.Get("ETag") != "" {
		t.Errorf("decisions %d, ETag %q", len(f.user.decisions), resp.Header.Get("ETag"))
	}
	reset()

	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aHiddenDraf), g7aSpecOne, "sess-staff", "", map[string]any{"state": "unban"})
	if resp.StatusCode != http.StatusOK || got["state"] != "draft" {
		t.Fatalf("unban %d %+v, want the decision's restored state", resp.StatusCode, got)
	}
	reset()

	resp, got = f.call(t, http.MethodPatch, g7aSub(g7aDraft), g7aSpecOne, "sess-alice", "", map[string]any{"state": "pending"})
	if resp.StatusCode != http.StatusOK || got["state"] != "pending" || resp.Header.Get("ETag") != `"c948000001.2"` {
		t.Fatalf("submit %d %+v %q: answered from the PATCH record and its version", resp.StatusCode, got, resp.Header.Get("ETag"))
	}
	reset()

	hash := strings.Repeat("cd", 32)
	body := g7aMintBody("ReadBackFails", map[string]any{"banner_hash": hash})
	resp, got = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(501), body)
	if resp.StatusCode != http.StatusCreated || got["display_name"] != "ReadBackFails" || got["state"] != "pending" || got["last_event"] != nil {
		t.Fatalf("create %d %+v: the mint landed, so the answer is 201 from what was sent", resp.StatusCode, got)
	}
	if resp.Header.Get("ETag") != "" {
		t.Errorf("ETag %q: without a read-back there is no version to give", resp.Header.Get("ETag"))
	}
	reset()
	mints, proposals := len(f.user.mints), len(f.user.proposals)
	resp, _ = f.call(t, http.MethodPost, "/api/v1/work-submissions", g7aSpecColl, "sess-alice", g7aKey(501), body)
	if resp.StatusCode != http.StatusCreated || len(f.user.mints) != mints || len(f.user.proposals) != proposals {
		t.Errorf("retry %d minted %d and proposed %d more", resp.StatusCode, len(f.user.mints)-mints, len(f.user.proposals)-proposals)
	}
}

func TestV1WorkSubmissionDeleteSurvivesALocalCleanupFailure(t *testing.T) {
	f := newG7aFix(t)
	run := func(q string) {
		t.Helper()
		if err := f.db.Exec(q).Error; err != nil {
			t.Fatal(err)
		}
	}
	run(`CREATE OR REPLACE FUNCTION g7a_block_delete() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'g7a: local cleanup refused'; END $$`)
	run(`CREATE TRIGGER g7a_block_delete BEFORE DELETE ON galgame FOR EACH ROW
		WHEN (OLD.id = 948000001) EXECUTE FUNCTION g7a_block_delete()`)
	t.Cleanup(func() {
		_ = f.db.Exec(`DROP TRIGGER IF EXISTS g7a_block_delete ON galgame`).Error
		_ = f.db.Exec(`DROP FUNCTION IF EXISTS g7a_block_delete()`).Error
	})

	resp, got := f.call(t, http.MethodDelete, g7aSub(g7aDraft), g7aSpecOne, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d %+v: catalog already deleted the draft, so the answer is 204", resp.StatusCode, got)
	}
	var n int64
	f.db.Raw(`SELECT count(*) FROM galgame WHERE id = ?`, g7aDraft).Scan(&n)
	if g7aState(t, f, g7aDraft) != "" || n != 1 {
		t.Errorf("claim state %q, local rows %d: want the claim gone and the row left for ops", g7aState(t, f, g7aDraft), n)
	}
}
