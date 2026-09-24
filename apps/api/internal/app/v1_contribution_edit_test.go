package app

import (
	"net/http"
	"slices"
	"strconv"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1EditFormShape(t *testing.T) {
	f := newG7bFix(t)
	spec := "/works/{work_id}/edit-form"
	path := g7bWorkPath(g7bWork) + "/edit-form"
	f.up.failVocab = true
	resp, body := f.call(t, http.MethodGet, path, spec, g7bSessAlice, "", nil)
	if v, _ := body["vocabularies"].([]any); resp.StatusCode != http.StatusOK || len(v) != 0 {
		t.Errorf("vocabulary failure %d %+v", resp.StatusCode, body["vocabularies"])
	}
	f.up.failVocab = false
	resp, body = f.call(t, http.MethodGet, path, spec, g7bSessAlice, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("edit-form %d %+v", resp.StatusCode, body)
	}
	if _, ok := body["viewer"]; ok {
		t.Errorf("edit form carries no viewer: the schema is the same for every caller %+v", body["viewer"])
	}
	values, _ := body["field_values"].(map[string]any)
	if values["catalog.work.display_name"] != "Bob Name" {
		t.Errorf("field_values %+v", values)
	}
	fields, _ := body["fields"].([]any)
	byKey := map[string]map[string]any{}
	for _, raw := range fields {
		fl, _ := raw.(map[string]any)
		byKey[fl["key"].(string)] = fl
	}
	if _, ok := byKey["catalog.work.future_thing"]; ok {
		t.Error("a field_type outside the vocabulary must be dropped, not passed on")
	}
	if fl := byKey["catalog.work.content_rating"]; fl["encoding"] != "int" || fl["vocabulary"] != "content_rating" || fl["element"] != nil {
		t.Errorf("content_rating %+v", fl)
	}
	if fl := byKey["catalog.work.display_name"]; fl["encoding"] != nil {
		t.Errorf("no vocabulary means encoding null %+v", fl)
	}
	if fl := byKey["catalog.work.legacy_flag"]; fl["is_deprecated"] != true || fl["is_nullable"] != true {
		t.Errorf("legacy_flag %+v", fl)
	}
	titles := byKey["catalog.work.titles"]
	el, _ := titles["element"].(map[string]any)
	members, _ := el["members"].([]any)
	if el["element_type"] != "object" || len(members) != 2 || asInt(titles["max_elements"]) != 100 {
		t.Errorf("titles element %+v", titles)
	}
	for _, k := range []string{"is_locked", "can_propose", "can_review"} {
		if _, ok := titles[k]; ok {
			t.Errorf("%s is not on infra's schema and must not be derived here", k)
		}
	}
	vocab, _ := body["vocabularies"].([]any)
	if len(vocab) != 1 || vocab[0].(map[string]any)["vocabulary"] != "content_rating" {
		t.Errorf("vocabularies %+v", vocab)
	}

	for _, c := range []struct {
		name, path, session, code string
		status                    int
	}{
		{"hidden", g7bWorkPath(g7bWorkHidden) + "/edit-form", g7bSessAlice, problem.CodeNotFound, 404},
		{"unknown", g7bWorkPath(g7bWorkUnknown) + "/edit-form", g7bSessAlice, problem.CodeNotFound, 404},
		{"merged", g7bWorkPath(g7bWorkMerged) + "/edit-form", g7bSessAlice, problem.CodeEntityMerged, 404},
		{"anonymous", path, "", problem.CodeMissingCredential, 401},
		{"no token", path, g7bSessNoToken, problem.CodeInvalidCredential, 401},
		{"scope", path, g7bSessNoscope, problem.CodeScopeRequired, 403},
		{"banned", path, g7bSessBanned, problem.CodeAccountBanned, 403},
		{"quota", path, g7bSessQuota, problem.CodeServiceUnavailable, 503},
	} {
		resp, body := f.call(t, http.MethodGet, c.path, spec, c.session, "", nil)
		if resp.StatusCode != c.status || body["code"] != c.code {
			t.Errorf("%s: %d %v, want %d %s", c.name, resp.StatusCode, body["code"], c.status, c.code)
		}
		if c.name == "merged" && body["current_id"] != strconv.Itoa(g7bWork) {
			t.Errorf("merged current_id %+v", body)
		}
		if c.name == "quota" && resp.Header.Get("Retry-After") != "30" {
			t.Errorf("quota Retry-After %q", resp.Header.Get("Retry-After"))
		}
	}
}

func TestV1WorkEditProposalsListIsPublicShape(t *testing.T) {
	f := newG7bFix(t)
	spec := "/works/{work_id}/edit-proposals"
	path := g7bWorkPath(g7bWork) + "/edit-proposals"
	resp, body := f.call(t, http.MethodGet, path, spec, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anon list %d %+v", resp.StatusCode, body)
	}
	got := itemIDs(t, body)
	want := []string{g7bID(f.pOpenBanned), g7bID(f.pOpenOther), g7bID(f.pOpenAlice)}
	if !slices.Equal(got, want) {
		t.Errorf("open items %v want %v (empty state and other tenants dropped)", got, want)
	}
	items, _ := body["items"].([]any)
	for _, raw := range items {
		it, _ := raw.(map[string]any)
		for _, k := range []string{"patch", "effective_patch", "decision_note", "amendments"} {
			if _, ok := it[k]; ok {
				t.Errorf("list item carries %s %+v", k, it)
			}
		}
		if it["viewer"] != nil {
			t.Errorf("anonymous viewer %+v", it["viewer"])
		}
	}
	banned := itemByID(body, f.pOpenBanned)
	if prop, _ := banned["proposer"].(map[string]any); prop["name"] != nil || prop["avatar"] != nil {
		t.Errorf("banned proposer must be a deleted ref %+v", banned["proposer"])
	}
	alice := itemByID(body, f.pOpenAlice)
	if ws, _ := alice["work_summary"].(map[string]any); ws["id"] != strconv.Itoa(g7bWork) {
		t.Errorf("work_summary %+v", alice["work_summary"])
	}

	_, body = f.call(t, http.MethodGet, path, spec, g7bSessAlice, "", nil)
	v, _ := itemByID(body, f.pOpenAlice)["viewer"].(map[string]any)
	if v["is_proposer"] != true || v["can_withdraw"] != true || v["can_decide"] != false || v["can_amend"] != true {
		t.Errorf("proposer viewer %+v", v)
	}
	v, _ = itemByID(body, f.pOpenOther)["viewer"].(map[string]any)
	if v["is_proposer"] != false || v["can_amend"] != false || v["can_decide"] != false {
		t.Errorf("stranger viewer %+v", v)
	}
	_, body = f.call(t, http.MethodGet, path, spec, g7bSessBob, "", nil)
	v, _ = itemByID(body, f.pOpenAlice)["viewer"].(map[string]any)
	if v["can_decide"] != true || v["can_amend"] != true {
		t.Errorf("the work's owner may decide %+v", v)
	}

	_, body = f.call(t, http.MethodGet, path+"?state=merged", spec, "", "", nil)
	if got := itemIDs(t, body); !slices.Equal(got, []string{g7bID(f.pMergedBob)}) {
		t.Errorf("merged %v", got)
	}
	merged := itemByID(body, f.pMergedBob)
	if d, _ := merged["decider"].(map[string]any); d["id"] != strconv.Itoa(w3UserStaff) || merged["decided_at"] == nil {
		t.Errorf("decider %+v", merged)
	}

	resp, body = f.call(t, http.MethodGet, path+"?state=pending", spec, "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownEnumValue)

	var walked []string
	cursor := ""
	for range 10 {
		u := path + "?limit=1"
		if cursor != "" {
			u += "&cursor=" + cursor
		}
		resp, body = f.call(t, http.MethodGet, u, spec, "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("walk %d %+v", resp.StatusCode, body)
		}
		walked = append(walked, itemIDs(t, body)...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			break
		}
		cursor = next
	}
	if !slices.Equal(walked, want) {
		t.Errorf("walk %v want %v", walked, want)
	}
	_, body = f.call(t, http.MethodGet, path+"?limit=1", spec, "", "", nil)
	next, _ := body["next_cursor"].(string)
	resp, body = f.call(t, http.MethodGet, path+"?limit=1&state=merged&cursor="+next, spec, "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidCursor)
}

func TestV1HiddenWorkEditHistoryIsNotFound(t *testing.T) {
	f := newG7bFix(t)
	for _, c := range []struct{ path, spec string }{
		{g7bWorkPath(g7bWorkHidden) + "/edit-proposals", "/works/{work_id}/edit-proposals"},
		{g7bWorkPath(g7bWorkHidden) + "/edit-revisions", "/works/{work_id}/edit-revisions"},
		{g7bWorkPath(g7bWorkHidden) + "/edit-revisions/diff?from_seq=1&to_seq=1", "/works/{work_id}/edit-revisions/diff"},
	} {
		for _, session := range []string{"", g7bSessAlice} {
			resp, body := f.call(t, http.MethodGet, c.path, c.spec, session, "", nil)
			wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
			if _, ok := body["items"]; ok {
				t.Errorf("%s leaked items", c.path)
			}
		}
	}
	if n := len(f.up.callsTo(http.MethodGet, "/v2/catalog/")); n != 0 {
		t.Errorf("a hidden work must not reach catalog's history faces, %d calls", n)
	}
}

func TestV1WorkEditRevisionsPage(t *testing.T) {
	f := newG7bFix(t)
	spec := "/works/{work_id}/edit-revisions"
	path := g7bWorkPath(g7bWork) + "/edit-revisions"
	resp, body := f.call(t, http.MethodGet, path, spec, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("revisions %d %+v", resp.StatusCode, body)
	}
	if asInt(body["total"]) != 3 || body["total_relation"] != "eq" {
		t.Errorf("total %+v", body)
	}
	if _, ok := body["viewer"]; ok {
		t.Error("revision list carries no viewer")
	}
	items, _ := body["items"].([]any)
	var seqs []int
	for _, raw := range items {
		seqs = append(seqs, asInt(raw.(map[string]any)["seq"]))
	}
	if !slices.Equal(seqs, []int{3, 2, 1}) {
		t.Errorf("newest first %v", seqs)
	}
	top, _ := items[0].(map[string]any)
	if top["revision_action"] != "merged" || top["proposal_id"] != g7bID(f.pMergedBob) {
		t.Errorf("merged revision %+v", top)
	}
	if a, _ := top["last_amender"].(map[string]any); a["id"] != strconv.Itoa(w3UserStaff) {
		t.Errorf("last_amender %+v", top["last_amender"])
	}
	if first, _ := items[2].(map[string]any); first["last_amender"] != nil || first["proposal_id"] != nil {
		t.Errorf("created revision %+v", first)
	}

	var walked []int
	for page := 1; page <= 3; page++ {
		_, body = f.call(t, http.MethodGet, path+"?limit=1&page="+strconv.Itoa(page), spec, "", "", nil)
		its, _ := body["items"].([]any)
		for _, raw := range its {
			walked = append(walked, asInt(raw.(map[string]any)["seq"]))
		}
		if asInt(body["total"]) != 3 {
			t.Errorf("page %d total %+v", page, body["total"])
		}
	}
	if !slices.Equal(walked, []int{3, 2, 1}) {
		t.Errorf("page walk %v", walked)
	}
	resp, body = f.call(t, http.MethodGet, path+"?limit=101", spec, "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeLimitTooLarge)
}

func TestV1WorkEditRevisionDiff(t *testing.T) {
	f := newG7bFix(t)
	spec := "/works/{work_id}/edit-revisions/diff"
	path := g7bWorkPath(g7bWork) + "/edit-revisions/diff"
	resp, body := f.call(t, http.MethodGet, path+"?from_seq=1&to_seq=3", spec, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("diff %d %+v", resp.StatusCode, body)
	}
	changes, _ := body["field_changes"].([]any)
	if len(changes) != 1 {
		t.Fatalf("changes %+v", body)
	}
	ch, _ := changes[0].(map[string]any)
	if ch["key"] != "catalog.work.display_name" || ch["from"] != "Edit One" || ch["to"] != "Bob Name" {
		t.Errorf("change %+v", ch)
	}
	if asInt(body["from_seq"]) != 1 || asInt(body["to_seq"]) != 3 {
		t.Errorf("seqs %+v", body)
	}
	calls := f.up.callsTo(http.MethodGet, "/v2/catalog/revisions/")
	if len(calls) != 1 || calls[0].path != "/v2/catalog/revisions/"+g7bID(f.rev3) ||
		calls[0].query.Get("include") != "diff" || calls[0].query.Get("diff_base") != g7bID(f.rev1) {
		t.Errorf("upstream diff call %+v", calls)
	}

	for _, c := range []struct {
		q, code string
		status  int
	}{
		{"?to_seq=2", problem.CodeInvalidParameter, 400},
		{"?from_seq=0&to_seq=2", problem.CodeInvalidParameter, 400},
		{"?from_seq=1&to_seq=99", problem.CodeNotFound, 404},
	} {
		resp, body := f.call(t, http.MethodGet, path+c.q, spec, "", "", nil)
		if resp.StatusCode != c.status || body["code"] != c.code {
			t.Errorf("%s: %d %v", c.q, resp.StatusCode, body["code"])
		}
	}
}

func TestV1ListMyEditProposals(t *testing.T) {
	f := newG7bFix(t)
	spec := "/me/edit-proposals"
	resp, body := f.call(t, http.MethodGet, "/api/v1/me/edit-proposals", spec, g7bSessAlice, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mine %d %+v", resp.StatusCode, body)
	}
	want := []string{g7bID(f.pPlainAlice), g7bID(f.pHidden), g7bID(f.pWithdrawn), g7bID(f.pDeclined), g7bID(f.pOpenAlice)}
	if got := itemIDs(t, body); !slices.Equal(got, want) {
		t.Errorf("mine %v want %v (other site, catalog.tag and empty state dropped)", got, want)
	}
	calls := f.up.callsTo(http.MethodGet, "/v2/me/proposals")
	if len(calls) != 1 || calls[0].query.Get("entity_type") != "catalog.work" || calls[0].auth != "g7b-alice" {
		t.Errorf("upstream %+v", calls)
	}

	_, body = f.call(t, http.MethodGet, "/api/v1/me/edit-proposals?work_id="+strconv.Itoa(g7bWork)+"&state=declined", spec, g7bSessAlice, "", nil)
	if got := itemIDs(t, body); !slices.Equal(got, []string{g7bID(f.pDeclined)}) {
		t.Errorf("filtered %v", got)
	}
	resp, body = f.call(t, http.MethodGet, "/api/v1/me/edit-proposals?state=nope", spec, g7bSessAlice, "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownEnumValue)
	resp, body = f.call(t, http.MethodGet, "/api/v1/me/edit-proposals", spec, "", "", nil)
	wantCode(t, resp, body, http.StatusUnauthorized, problem.CodeMissingCredential)

	resp, body = f.bearer(t, http.MethodGet, "/api/v1/me/edit-proposals", spec, "bob-token", "", nil)
	if resp.StatusCode != http.StatusOK || !slices.Equal(itemIDs(t, body), []string{g7bID(f.pMergedBob)}) {
		t.Errorf("bearer mine %d %v", resp.StatusCode, itemIDs(t, body))
	}
}

func TestV1EditProposalQueue(t *testing.T) {
	f := newG7bFix(t)
	spec := "/edit-proposals"
	resp, body := f.call(t, http.MethodGet, "/api/v1/edit-proposals", spec, g7bSessStaff, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("queue %d %+v", resp.StatusCode, body)
	}
	want := []string{g7bID(f.pStaffOwn), g7bID(f.pPlainAlice), g7bID(f.pHidden), g7bID(f.pOpenBanned), g7bID(f.pOpenOther), g7bID(f.pOpenAlice)}
	if got := itemIDs(t, body); !slices.Equal(got, want) {
		t.Errorf("queue %v want %v", got, want)
	}
	if v, _ := itemByID(body, f.pOpenAlice)["viewer"].(map[string]any); v["can_decide"] != true {
		t.Errorf("reviewer viewer %+v", v)
	}
	if n := len(f.up.callsTo(http.MethodGet, "/v2/moderation/proposals")); n != 1 {
		t.Errorf("open reads the moderation queue, %d calls", n)
	}

	f.up.resetCalls()
	_, body = f.call(t, http.MethodGet, "/api/v1/edit-proposals?state=merged", spec, g7bSessStaff, "", nil)
	if got := itemIDs(t, body); !slices.Equal(got, []string{g7bID(f.pMergedBob)}) {
		t.Errorf("merged queue %v", got)
	}
	if n := len(f.up.callsTo(http.MethodGet, "/v2/moderation/proposals")); n != 0 {
		t.Errorf("moderation serves open only; a merged tab must not ask it, %d calls", n)
	}
	pub := f.up.callsTo(http.MethodGet, "/v2/catalog/proposals")
	if len(pub) != 1 || pub[0].query.Get("state") != "merged" || pub[0].query.Get("site") != "kungal" || pub[0].auth != g7bAppKey {
		t.Errorf("public face call %+v", pub)
	}

	f.up.resetCalls()
	for _, c := range []struct {
		name string
		do   func() (*http.Response, map[string]any)
	}{
		{"user", func() (*http.Response, map[string]any) {
			return f.call(t, http.MethodGet, "/api/v1/edit-proposals", spec, g7bSessAlice, "", nil)
		}},
		{"owner", func() (*http.Response, map[string]any) {
			return f.call(t, http.MethodGet, "/api/v1/edit-proposals", spec, g7bSessBob, "", nil)
		}},
		{"bearer moderator", func() (*http.Response, map[string]any) {
			return f.bearer(t, http.MethodGet, "/api/v1/edit-proposals", spec, "staff-token", "", nil)
		}},
	} {
		resp, body := c.do()
		if resp.StatusCode != http.StatusForbidden || body["code"] != problem.CodePermissionRequired {
			t.Errorf("%s: %d %v", c.name, resp.StatusCode, body["code"])
		}
	}
	if n := len(f.up.calls); n != 0 {
		t.Errorf("refused queue reads reached catalog %d times", n)
	}
}

func TestV1EditQueueUsesThePermissionNotTheRole(t *testing.T) {
	f := newG7bFix(t)
	f.denyEditReview(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/edit-proposals", "/edit-proposals", g7bSessStaff, "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
	_, body = f.call(t, http.MethodGet, g7bWorkPath(g7bWork)+"/edit-proposals", "/works/{work_id}/edit-proposals", g7bSessStaff, "", nil)
	if v, _ := itemByID(body, f.pOpenAlice)["viewer"].(map[string]any); v["can_decide"] != false {
		t.Errorf("a moderator without the key has no standing %+v", v)
	}
}

func TestV1GetEditProposalWorkbench(t *testing.T) {
	f := newG7bFix(t)
	spec := "/edit-proposals/{proposal_id}"
	path := g7bProposalPath(f.pOpenAlice)
	resp, body := f.call(t, http.MethodGet, path, spec, g7bSessAlice, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("proposer read %d %+v", resp.StatusCode, body)
	}
	want := f.up.etagOf(f.pOpenAlice)
	if resp.Header.Get("ETag") != want {
		t.Errorf("ETag %q want %q", resp.Header.Get("ETag"), want)
	}
	if patch, _ := body["patch"].(map[string]any); patch["catalog.work.display_name"] != "Alice Name" {
		t.Errorf("patch %+v", body["patch"])
	}
	if _, ok := body["decision_note"]; ok {
		t.Error("catalog publishes no decision note")
	}
	if am, _ := body["amendments"].([]any); am == nil || len(am) != 0 {
		t.Errorf("amendments %+v", body["amendments"])
	}
	v, _ := body["viewer"].(map[string]any)
	if v["is_proposer"] != true || v["can_decide"] != false || v["can_withdraw"] != true {
		t.Errorf("proposer viewer %+v", v)
	}
	if n := len(f.up.callsTo(http.MethodGet, "/v2/moderation/proposals/")); n != 0 {
		t.Errorf("the proposer's own face answers first, %d moderation reads", n)
	}

	resp, body = f.call(t, http.MethodGet, path, spec, g7bSessStaff, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reviewer read %d %+v", resp.StatusCode, body)
	}
	if v, _ := body["viewer"].(map[string]any); v["can_decide"] != true || v["is_proposer"] != false {
		t.Errorf("reviewer viewer %+v", v)
	}
	resp, body = f.call(t, http.MethodGet, path, spec, g7bSessBob, "", nil)
	if v, _ := body["viewer"].(map[string]any); resp.StatusCode != http.StatusOK || v["can_decide"] != true {
		t.Errorf("owner %d %+v", resp.StatusCode, body["viewer"])
	}

	_, body = f.call(t, http.MethodGet, g7bProposalPath(f.pMergedBob), spec, g7bSessStaff, "", nil)
	am, _ := body["amendments"].([]any)
	if len(am) != 1 || asInt(am[0].(map[string]any)["seq"]) != 1 {
		t.Errorf("amendments %+v", body["amendments"])
	}
	if v, _ := body["viewer"].(map[string]any); v["can_decide"] != false || v["can_amend"] != false {
		t.Errorf("a decided proposal cannot be decided again %+v", v)
	}

	f.up.resetCalls()
	for _, c := range []struct {
		name string
		do   func() (*http.Response, map[string]any)
	}{
		{"stranger", func() (*http.Response, map[string]any) {
			return f.call(t, http.MethodGet, path, spec, g7bSessOther, "", nil)
		}},
		{"other site", func() (*http.Response, map[string]any) {
			return f.call(t, http.MethodGet, g7bProposalPath(f.pOtherSite), spec, g7bSessAlice, "", nil)
		}},
		{"other family", func() (*http.Response, map[string]any) {
			return f.call(t, http.MethodGet, g7bProposalPath(f.pTag), spec, g7bSessAlice, "", nil)
		}},
		{"unknown", func() (*http.Response, map[string]any) {
			return f.call(t, http.MethodGet, g7bProposalPath(957199999), spec, g7bSessStaff, "", nil)
		}},
	} {
		resp, body := c.do()
		if resp.StatusCode != http.StatusNotFound || body["code"] != problem.CodeNotFound {
			t.Errorf("%s: %d %v", c.name, resp.StatusCode, body["code"])
		}
		if _, ok := body["patch"]; ok {
			t.Errorf("%s leaked a patch", c.name)
		}
	}

	f.up.resetCalls()
	resp, body = f.bearer(t, http.MethodGet, path, spec, "staff-token", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	if n := len(f.up.callsTo(http.MethodGet, "/v2/moderation/")); n != 0 {
		t.Errorf("a Bearer caller never reads the review face, %d calls", n)
	}
	resp, body = f.bearer(t, http.MethodGet, g7bProposalPath(f.pMergedBob), spec, "bob-token", "", nil)
	if v, _ := body["viewer"].(map[string]any); resp.StatusCode != http.StatusOK || v["can_decide"] != false || v["is_proposer"] != true {
		t.Errorf("bearer proposer %d %+v", resp.StatusCode, body["viewer"])
	}

	_, body = f.call(t, http.MethodGet, g7bProposalPath(f.pOpenBanned), spec, g7bSessStaff, "", nil)
	if p, _ := body["proposer"].(map[string]any); p["name"] != nil || p["id"] != strconv.Itoa(w3UserBanned) {
		t.Errorf("banned proposer %+v", body["proposer"])
	}
}

func TestV1EditReadsWriteNothing(t *testing.T) {
	f := newG7bFix(t)
	count := func() [3]int {
		return [3]int{
			f.scalar(t, `SELECT count(*) FROM galgame_activity WHERE work_id BETWEEN 957000001 AND 957000099`),
			f.scalar(t, `SELECT count(*) FROM message WHERE link LIKE '/galgame/957000%'`),
			f.scalar(t, `SELECT count(*) FROM galgame WHERE id BETWEEN 957000001 AND 957000099`),
		}
	}
	before := count()
	w := g7bWorkPath(g7bWork)
	for _, c := range []struct{ path, spec, session string }{
		{w + "/edit-form", "/works/{work_id}/edit-form", g7bSessAlice},
		{w + "/edit-proposals", "/works/{work_id}/edit-proposals", g7bSessStaff},
		{w + "/edit-revisions", "/works/{work_id}/edit-revisions", g7bSessAlice},
		{w + "/edit-revisions/diff?from_seq=1&to_seq=2", "/works/{work_id}/edit-revisions/diff", ""},
		{"/api/v1/me/edit-proposals", "/me/edit-proposals", g7bSessAlice},
		{"/api/v1/edit-proposals", "/edit-proposals", g7bSessStaff},
		{g7bProposalPath(f.pOpenAlice), "/edit-proposals/{proposal_id}", g7bSessStaff},
	} {
		resp, body := f.call(t, http.MethodGet, c.path, c.spec, c.session, "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s %d %+v", c.path, resp.StatusCode, body)
		}
	}
	if after := count(); after != before {
		t.Errorf("reads wrote rows: %v → %v", before, after)
	}
	for _, m := range []string{http.MethodPost, http.MethodPatch, http.MethodDelete} {
		if n := len(f.up.callsTo(m, "/v2/")); n != 0 {
			t.Errorf("reads made %d upstream %s calls", n, m)
		}
	}
}
