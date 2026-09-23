package app

import (
	"net/http"
	"testing"

	"kun-galgame-api/internal/apiv1/collect"
)

func TestV1MoemoepointWalksPages(t *testing.T) {
	f := newMeFix(t)
	seen := map[string]bool{}
	order := make([]string, 0, 7)
	cur := ""
	pages := 0
	for {
		pages++
		if pages > 10 {
			t.Fatal("did not terminate")
		}
		q := "?limit=2"
		if cur != "" {
			q += "&cursor=" + cur
		}
		resp, body := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries"+q, "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d %d %+v", pages, resp.StatusCode, body)
		}
		ids := itemIDs(t, body)
		for _, id := range ids {
			if seen[id] {
				t.Fatalf("duplicate %s", id)
			}
			seen[id] = true
			order = append(order, id)
		}
		cur = nextCursor(body)
		if cur == "" {
			break
		}
	}
	if len(order) != 7 {
		t.Fatalf("got %v, want 7 entries", order)
	}
	want := []string{"7", "6", "5", "4", "3", "2", "1"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order %v, want %v", order, want)
		}
	}
}

func TestV1MoemoepointLastPageOmitsNextCursor(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries?limit=50", "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if _, ok := body["next_cursor"]; ok {
		t.Fatalf("last page still has next_cursor %v", body["next_cursor"])
	}
}

func TestV1MoemoepointCursorBoundToReason(t *testing.T) {
	f := newMeFix(t)
	resp, first := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries?limit=2&reason=liked", "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, first)
	}
	cur := nextCursor(first)
	if cur == "" {
		t.Fatal("expected another page")
	}
	resp, body := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries?limit=2&reason=daily_checkin&cursor="+cur, "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")
}

func TestV1MoemoepointLimit51IsLimitTooLarge(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries?limit=51", "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusBadRequest, "LIMIT_TOO_LARGE")
}

func TestV1MoemoepointUnknownReasonIsInvalidParameter(t *testing.T) {
	f := newMeFix(t)
	f.logReason400.Store(true)
	resp, body := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries?reason=liked", "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	fieldErr(t, body, "parameter", "reason", "INVALID_FORMAT")
}

func TestV1MoemoepointMalformedCursorIsInvalidCursor(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries?cursor=nope", "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")
}

func TestV1MoemoepointOAuthDownIs503(t *testing.T) {
	f := newMeFix(t)
	f.logFail.Store(true)
	resp, body := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries", "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1MoemoepointFromThisSite(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries?limit=50", "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	foundLocal, foundRemote := false, false
	for _, it := range items {
		m, _ := it.(map[string]any)
		if strID(m["id"]) == "3" && m["is_from_this_site"] == false {
			foundRemote = true
		}
		if strID(m["id"]) == "7" && m["is_from_this_site"] == true {
			foundLocal = true
		}
	}
	if !foundLocal || !foundRemote {
		t.Fatalf("is_from_this_site not mapped: %+v", items)
	}
}

func TestV1MoemoepointEncodeCursorShape(t *testing.T) {
	cur := collect.EncodeCursor("id_desc", collect.Fingerprint("1", ""), "7")
	if cur[:4] != "cur_" {
		t.Fatalf("cursor %s", cur)
	}
}
