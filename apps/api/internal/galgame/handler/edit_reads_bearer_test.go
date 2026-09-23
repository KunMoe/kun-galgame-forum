package handler

import (
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/internal/middleware"
)

func editFaceCalls(f *fakeEditFace) []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]recordedRequest, 0, len(f.requests))
	for i := range f.requests {
		if f.requests[i].Face != "other" {
			out = append(out, f.requests[i])
		}
	}
	return out
}

func TestEditReadsRideTheUserPlane(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		user       *middleware.UserInfo
	}{
		{"bootstrap", "/api/galgame/1000/edit/bootstrap", moderatorUser},
		{"queue", "/api/galgame-edit/queue", moderatorUser},
		{"mine", "/api/galgame-edit/mine", plainUser},
		{"workbench", "/api/galgame-edit/proposals/7", moderatorUser},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeEditFace{}
			app := editTestApp(t, fake.server(t).URL, tc.user)
			if status, raw := doJSON(t, app, "GET", tc.path, ""); status != http.StatusOK {
				t.Fatalf("%s: status = %d body %s", tc.name, status, raw)
			}
			calls := editFaceCalls(fake)
			if len(calls) == 0 {
				t.Fatalf("%s reached no catalog face at all", tc.name)
			}
			for _, r := range calls {
				if r.Face != "user" {
					t.Fatalf("%s still speaks S2S: %s %s", tc.name, r.Method, r.Path)
				}
				if r.Auth != "Bearer user-jwt" {
					t.Fatalf("%s auth = %q, want the session's own bearer", tc.name, r.Auth)
				}
				if strings.Contains(r.Query, "site=") || strings.Contains(r.Query, "proposer_uid=") {
					t.Fatalf("%s names a site or a proposer: %q", tc.name, r.Query)
				}
			}
		})
	}
}

func TestEditMineAsksForItsOwn(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	if status, raw := doJSON(t, app, "GET", "/api/galgame-edit/mine", ""); status != http.StatusOK {
		t.Fatalf("mine: status = %d body %s", status, raw)
	}
	req := fake.callTo("/v2/me/proposals")
	if req == nil {
		t.Fatalf("mine did not reach the user-plane list: %+v", fake.requests)
	}
	if strings.Contains(req.Query, "mine=") {
		t.Fatalf("mine is the path, not a query flag: %q", req.Query)
	}
}

func TestEditQueueAsksForEverybodys(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, moderatorUser)
	if status, raw := doJSON(t, app, "GET", "/api/galgame-edit/queue", ""); status != http.StatusOK {
		t.Fatalf("queue: status = %d body %s", status, raw)
	}
	req := fake.callTo("/v2/moderation/proposals")
	if req == nil {
		t.Fatalf("queue did not reach the user-plane list: %+v", fake.requests)
	}
	if strings.Contains(req.Query, "mine") {
		t.Fatalf("queue query = %q, want no mine flag", req.Query)
	}
	if !strings.Contains(req.Query, "state=open") && !strings.Contains(req.Query, "status=open") {
		t.Fatalf("queue query = %q, want the status filter", req.Query)
	}
}

func TestEditQueueRelaysTheInfraDenial(t *testing.T) {
	fake := &fakeEditFace{userStatus: http.StatusForbidden,
		userBody: `{"code":233,"message":"permission denied: catalog.edit.review"}`}
	app := editTestApp(t, fake.server(t).URL, moderatorUser)
	status, raw := doJSON(t, app, "GET", "/api/galgame-edit/queue", "")
	if status != http.StatusForbidden {
		t.Fatalf("refused queue: status = %d body %s, want 403", status, raw)
	}
}
