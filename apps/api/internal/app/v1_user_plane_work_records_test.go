package app

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

type g6RecordCall struct {
	token  string
	userID int64
	states []string
	counts bool
	record string
	sort   string
	total  bool
}

// Under record=playtime or record=any, g6WorkExtra is a work with only a
// playtime: it has no state there.
func g6Records(record string) []catalogclient.AccountWorkRecord {
	at := func(day int) time.Time { return time.Date(2026, 10, day, 8, 0, 0, 0, time.UTC) }
	mins := func(n int) *int { return &n }
	main := "main"
	recs := []catalogclient.AccountWorkRecord{
		{WorkID: g6WorkLive, State: "doing", StateUpdatedAt: at(5), PlaytimeMinutes: mins(120),
			Rating: &catalogclient.Rating{ID: 5001, WorkID: g6WorkLive, Overall: 8}},
		{WorkID: g6WorkOwner, State: "done", Completion: &main, StateUpdatedAt: at(4), PlaytimeMinutes: mins(5)},
		{WorkID: g6WorkNSFW, State: "wish", StateUpdatedAt: at(3)},
		{WorkID: g6WorkHidden, State: "dropped", StateUpdatedAt: at(2)},
		{WorkID: g6WorkExtra, State: "done", StateUpdatedAt: at(1)},
	}
	if record == catalogclient.RecordPlaytime || record == catalogclient.RecordAny {
		recs[4] = catalogclient.AccountWorkRecord{WorkID: g6WorkExtra, PlaytimeMinutes: mins(300)}
	}
	return recs
}

func (u *g6User) WorkRecords(_ context.Context, token string, q catalogclient.WorkRecordsQuery) (catalogclient.WorkRecordsPage, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	uid := q.UserID
	if token != "" {
		if err := u.gate(token); err != nil {
			return catalogclient.WorkRecordsPage{}, err
		}
		uid = u.uid(token)
	} else if u.private[uid] {
		return catalogclient.WorkRecordsPage{}, &catalogclient.UserAPIError{
			Status: http.StatusForbidden, ProblemCode: catalogclient.CodePlayRecordsPrivate, Message: "private",
		}
	}
	u.recordCalls = append(u.recordCalls, g6RecordCall{token: token, userID: uid, states: q.States, counts: q.Counts, record: q.Record, sort: q.Sort, total: q.PlaytimeTotal})

	var matched []catalogclient.AccountWorkRecord
	if uid == w3UserAlice {
		for _, r := range g6Records(q.Record) {
			if q.SFWOnly && r.WorkID == g6WorkNSFW {
				continue
			}
			state := cmp.Or(r.State, catalogclient.WorkStateNone)
			if len(q.States) > 0 && !slices.Contains(q.States, state) {
				continue
			}
			matched = append(matched, r)
		}
	}
	out := catalogclient.WorkRecordsPage{}
	if q.Counts {
		out.StateCounts = map[string]int{}
		for _, s := range catalogclient.WorkStates() {
			out.StateCounts[s] = 0
		}
		for _, r := range matched {
			out.StateCounts[cmp.Or(r.State, catalogclient.WorkStateNone)]++
		}
	}
	if q.PlaytimeTotal && !u.noTotal {
		var sum int64
		for _, r := range matched {
			if r.PlaytimeMinutes != nil {
				sum += int64(*r.PlaytimeMinutes)
			}
		}
		out.PlaytimeTotalMinutes = &sum
	}
	if q.Sort == "playtime_desc" {
		minutes := func(r catalogclient.AccountWorkRecord) int {
			if r.PlaytimeMinutes == nil {
				return -1
			}
			return *r.PlaytimeMinutes
		}
		slices.SortStableFunc(matched, func(a, b catalogclient.AccountWorkRecord) int { return minutes(b) - minutes(a) })
	}
	start := 0
	if q.Cursor != "" {
		start, _ = strconv.Atoi(strings.TrimPrefix(q.Cursor, "cur_"))
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	end := min(start+limit, len(matched))
	if start < end {
		out.Items = matched[start:end]
	}
	if end < len(matched) {
		out.NextCursor = "cur_" + strconv.Itoa(end)
	}
	return out, nil
}

func (u *g6User) MyPrivacy(_ context.Context, token string) (catalogclient.PrivacySettings, error) {
	if err := u.gate(token); err != nil {
		return catalogclient.PrivacySettings{}, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.privacyOf(u.uid(token)), nil
}

func (u *g6User) PatchMyPrivacy(_ context.Context, token string, patch catalogclient.PrivacyPatch) (catalogclient.PrivacySettings, error) {
	if err := u.gate(token); err != nil {
		return catalogclient.PrivacySettings{}, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	uid := u.uid(token)
	if u.private == nil {
		u.private = map[int64]bool{}
	}
	if u.companiesPublic == nil {
		u.companiesPublic = map[int64]bool{}
	}
	if patch.PlayRecords != "" {
		u.private[uid] = patch.PlayRecords == catalogclient.PlayRecordsPrivate
	}
	if patch.FollowedCompanies != "" {
		u.companiesPublic[uid] = patch.FollowedCompanies == catalogclient.PlayRecordsPublic
	}
	at := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	u.privacyAt = &at
	return u.privacyOf(uid), nil
}

func (u *g6User) privacyOf(uid int64) catalogclient.PrivacySettings {
	out := catalogclient.PrivacySettings{
		PlayRecords: catalogclient.PlayRecordsPublic, FollowedCompanies: catalogclient.PlayRecordsPrivate, UpdatedAt: u.privacyAt,
	}
	if u.private[uid] {
		out.PlayRecords = catalogclient.PlayRecordsPrivate
	}
	if u.companiesPublic[uid] {
		out.FollowedCompanies = catalogclient.PlayRecordsPublic
	}
	return out
}

func (f *g6Fix) setPrivate(uid int64) {
	f.user.mu.Lock()
	defer f.user.mu.Unlock()
	if f.user.private == nil {
		f.user.private = map[int64]bool{}
	}
	f.user.private[uid] = true
}

func (f *g6Fix) recordCalls() []g6RecordCall {
	f.user.mu.Lock()
	defer f.user.mu.Unlock()
	return append([]g6RecordCall(nil), f.user.recordCalls...)
}

const g6RecordsSpec = "/users/{user_id}/work-records"

func g6RecordsPath(uid int, query string) string {
	p := "/api/v1/users/" + idStr(uid) + "/work-records"
	if query != "" {
		p += "?" + query
	}
	return p
}

func recordWorkIDs(t *testing.T, body map[string]any) []string {
	t.Helper()
	var out []string
	for _, raw := range body["items"].([]any) {
		item := raw.(map[string]any)
		out = append(out, item["work_summary"].(map[string]any)["id"].(string))
	}
	return out
}

func TestV1ListUserWorkRecordsVisitor(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, ""), g6RecordsSpec, "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	got := recordWorkIDs(t, body)
	want := []string{idStr(g6WorkLive), idStr(g6WorkOwner), idStr(g6WorkExtra)}
	if !slices.Equal(got, want) {
		t.Fatalf("items %v, want %v: the adult work hidden, the unrenderable one dropped", got, want)
	}
	items := body["items"].([]any)
	live := items[0].(map[string]any)
	if live["play_state"] != "doing" || asInt(live["playtime_minutes"]) != 120 || live["state_updated_at"] != "2026-10-05T08:00:00Z" {
		t.Errorf("live %+v", live)
	}
	rating := live["user_rating"].(map[string]any)
	if rating["id"] != "5001" || asInt(rating["overall"]) != 8 {
		t.Errorf("rating %+v", rating)
	}
	owner := items[1].(map[string]any)
	if owner["play_state"] != "done_main" || owner["playtime_minutes"] != nil || owner["user_rating"] != nil {
		t.Errorf("a withdrawn playtime must read null: %+v", owner)
	}
	if extra := items[2].(map[string]any); extra["play_state"] != "done" {
		t.Errorf("done without completion %+v", extra)
	}
	counts := body["state_counts"].(map[string]any)
	if asInt(counts["wish"]) != 0 || asInt(counts["doing"]) != 1 || asInt(counts["done"]) != 2 || asInt(counts["dropped"]) != 1 || asInt(counts["on_hold"]) != 0 {
		t.Errorf("counts follow include_nsfw and keep the unrenderable work: %+v", counts)
	}
	for _, c := range f.recordCalls() {
		if c.token != "" {
			t.Errorf("a visitor's read must use the app key, sent token %q", c.token)
		}
	}

	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "include_nsfw=true"), g6RecordsSpec, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous nsfw %d %+v", resp.StatusCode, body)
	}
	if got := recordWorkIDs(t, body); !slices.Contains(got, idStr(g6WorkNSFW)) {
		t.Errorf("include_nsfw lists the adult work: %v", got)
	}
	if asInt(body["state_counts"].(map[string]any)["wish"]) != 1 {
		t.Errorf("include_nsfw counts it: %+v", body["state_counts"])
	}
}

func TestV1ListUserWorkRecordsFilteredTabCountsEveryState(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "state=done,doing"), g6RecordsSpec, "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	if got := recordWorkIDs(t, body); !slices.Equal(got, []string{idStr(g6WorkLive), idStr(g6WorkOwner), idStr(g6WorkExtra)}) {
		t.Errorf("items %v", got)
	}
	counts := body["state_counts"].(map[string]any)
	if asInt(counts["dropped"]) != 1 || asInt(counts["done"]) != 2 {
		t.Errorf("a filtered tab still counts the other tabs: %+v", counts)
	}

	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "state=finished"), g6RecordsSpec, "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownEnumValue)
}

func TestV1ListUserWorkRecordsCursor(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "limit=2"), g6RecordsSpec, "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page 1 %d %+v", resp.StatusCode, body)
	}
	next, _ := body["next_cursor"].(string)
	if next == "" {
		t.Fatalf("page 1 has no next_cursor: %+v", body)
	}

	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "limit=2&cursor="+next), g6RecordsSpec, "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page 2 %d %+v", resp.StatusCode, body)
	}
	if _, ok := body["state_counts"]; ok {
		t.Errorf("state_counts belongs to the first page only: %+v", body)
	}
	if got := recordWorkIDs(t, body); !slices.Equal(got, []string{idStr(g6WorkExtra)}) {
		t.Errorf("page 2 %v", got)
	}

	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "state=doing&cursor="+next), g6RecordsSpec, "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidCursor)
	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserBob, "cursor="+next), g6RecordsSpec, "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidCursor)
}

func TestV1ListUserWorkRecordsPrivate(t *testing.T) {
	f := newG6Fix(t)
	f.setPrivate(w3UserAlice)

	resp, body := f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, ""), g6RecordsSpec, "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePlayRecordsPrivate)
	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, ""), g6RecordsSpec, "", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePlayRecordsPrivate)

	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "state=doing"), g6RecordsSpec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("the owner reads a private library: %d %+v", resp.StatusCode, body)
	}
	calls := f.recordCalls()
	last := calls[len(calls)-1]
	if last.token != g6TokAlice {
		t.Errorf("the owner's read must carry their own token, sent %q", last.token)
	}

	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserBanned, ""), g6RecordsSpec, "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1PrivacySettings(t *testing.T) {
	f := newG6Fix(t)
	const path, spec = "/api/v1/me/privacy-settings", "/me/privacy-settings"

	resp, body := f.call(t, http.MethodGet, path, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || body["is_play_records_public"] != true || body["changed_at"] != nil {
		t.Fatalf("default %d %+v", resp.StatusCode, body)
	}

	resp, body = f.call(t, http.MethodPatch, path, spec, "sess-alice", "", map[string]any{})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, body = f.call(t, http.MethodPatch, path, spec, "sess-alice", "", map[string]any{"is_play_records_public": false})
	if resp.StatusCode != http.StatusOK || body["is_play_records_public"] != false || body["changed_at"] != "2026-10-14T09:00:00Z" {
		t.Fatalf("turn off %d %+v", resp.StatusCode, body)
	}
	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, ""), g6RecordsSpec, "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePlayRecordsPrivate)

	resp, body = f.call(t, http.MethodPatch, path, spec, "sess-alice", "", map[string]any{"is_play_records_public": true})
	if resp.StatusCode != http.StatusOK || body["is_play_records_public"] != true {
		t.Fatalf("turn on %d %+v", resp.StatusCode, body)
	}

	resp, body = f.call(t, http.MethodGet, path, spec, "", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous %d %+v", resp.StatusCode, body)
	}
}

const g6TotalSpec = "/users/{user_id}/playtime-total"

func g6TotalPath(uid int, query string) string {
	return "/api/v1/users/" + idStr(uid) + "/playtime-total?" + query
}

func TestV1UserPlaytimeTotal(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, g6TotalPath(w3UserAlice, ""), g6TotalSpec, "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "user_playtime_total" || asInt(body["total_playtime_minutes"]) != 425 {
		t.Fatalf("visitor %d %+v", resp.StatusCode, body)
	}
	calls := f.recordCalls()
	last := calls[len(calls)-1]
	if last.token != "" || last.record != "playtime" || !last.total {
		t.Errorf("a visitor's total must ask catalog with the app key for record=playtime and the total: %+v", last)
	}

	f.setPrivate(w3UserAlice)
	resp, body = f.call(t, http.MethodGet, g6TotalPath(w3UserAlice, ""), g6TotalSpec, "", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePlayRecordsPrivate)
	resp, body = f.call(t, http.MethodGet, g6TotalPath(w3UserAlice, "include_nsfw=true"), g6TotalSpec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(body["total_playtime_minutes"]) != 425 {
		t.Fatalf("the owner reads a private total: %d %+v", resp.StatusCode, body)
	}

	f.user.mu.Lock()
	f.user.noTotal = true
	f.user.mu.Unlock()
	resp, body = f.call(t, http.MethodGet, g6TotalPath(w3UserBob, ""), g6TotalSpec, "", "", nil)
	if resp.StatusCode != http.StatusOK || body["total_playtime_minutes"] != nil {
		t.Fatalf("a catalog without the total reads null: %d %+v", resp.StatusCode, body)
	}
}

func TestV1ListUserWorkRecordsAnyRecordSorted(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "record=any&sort=playtime_desc"), g6RecordsSpec, "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	want := []string{idStr(g6WorkExtra), idStr(g6WorkLive), idStr(g6WorkOwner)}
	if got := recordWorkIDs(t, body); !slices.Equal(got, want) {
		t.Fatalf("items %v, want %v: longest playtime first", got, want)
	}
	extra := body["items"].([]any)[0].(map[string]any)
	if extra["play_state"] != nil || extra["state_updated_at"] != nil || asInt(extra["playtime_minutes"]) != 300 {
		t.Errorf("a work with only a playtime %+v", extra)
	}
	if counts := body["state_counts"].(map[string]any); asInt(counts["none"]) != 1 || asInt(counts["done"]) != 1 {
		t.Errorf("counts %+v", counts)
	}
	calls := f.recordCalls()
	if c := calls[0]; c.record != "any" || c.sort != "playtime_desc" {
		t.Errorf("catalog asked for record %q sort %q", c.record, c.sort)
	}

	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "record=any&state=none"), g6RecordsSpec, "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || !slices.Equal(recordWorkIDs(t, body), []string{idStr(g6WorkExtra)}) {
		t.Fatalf("state=none %d %+v", resp.StatusCode, body)
	}
	if counts := body["state_counts"].(map[string]any); asInt(counts["none"]) != 1 || asInt(counts["doing"]) != 1 {
		t.Errorf("a filtered tab still counts every state under the same record: %+v", counts)
	}

	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "state=none"), g6RecordsSpec, "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)

	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "record=any&limit=1"), g6RecordsSpec, "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page 1 %d %+v", resp.StatusCode, body)
	}
	next, _ := body["next_cursor"].(string)
	resp, body = f.call(t, http.MethodGet, g6RecordsPath(w3UserAlice, "record=any&sort=playtime_desc&limit=1&cursor="+next), g6RecordsSpec, "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidCursor)
}
