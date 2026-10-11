package apiv1

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

type gateProbe struct {
	ran     atomic.Int32
	lookups atomic.Int32
	gated   map[int]bool
	err     error
}

func (g *gateProbe) lookup(_ context.Context, userID int) (bool, error) {
	g.lookups.Add(1)
	return g.gated[userID], g.err
}

func (g *gateProbe) register(api huma.API) {
	handler := func(context.Context, *struct {
		UserID   string `path:"user_id"`
		AuthorID string `query:"author_id" required:"false"`
	}) (*struct{}, error) {
		g.ran.Add(1)
		return nil, nil
	}
	huma.Register(api, ProfileGated(Optional(huma.Operation{
		OperationID: "testGatedPath",
		Method:      http.MethodGet,
		Path:        "/_test/users/{user_id}",
		Summary:     "Gated by path",
		Description: "Test-only.",
	}), "user_id"), handler)
	huma.Register(api, ProfileGated(Optional(huma.Operation{
		OperationID: "testGatedQuery",
		Method:      http.MethodGet,
		Path:        "/_test/things/{user_id}",
		Summary:     "Gated by query",
		Description: "Test-only.",
	}), "author_id"), handler)
}

func gateGet(t *testing.T, outcome middleware.IdentityOutcome, probe *gateProbe, lookup ProfileGateLookup, path string) *http.Response {
	t.Helper()
	app, _ := newTestAPI(t, Deps{Resolver: &fakeResolver{id: identityFor(outcome)}, ProfileGate: lookup}, probe.register)
	return do(t, app, http.MethodGet, path, "", nil)
}

func TestProfileGate(t *testing.T) {
	cases := []struct {
		name    string
		outcome middleware.IdentityOutcome
		path    string
		status  int
		lookups int32
	}{
		{"anonymous, gated path user", middleware.IdentityAnonymous, "/api/v1/_test/users/5", 403, 1},
		{"anonymous, open path user", middleware.IdentityAnonymous, "/api/v1/_test/users/6", 204, 1},
		{"expired cookie counts as anonymous", middleware.IdentitySessionMissing, "/api/v1/_test/users/5", 403, 1},
		{"signed in, gated path user", middleware.IdentitySessionOK, "/api/v1/_test/users/5", 204, 0},
		{"bearer, gated path user", middleware.IdentityBearerOK, "/api/v1/_test/users/5", 204, 0},
		{"anonymous, gated author", middleware.IdentityAnonymous, "/api/v1/_test/things/6?author_id=5", 403, 1},
		{"anonymous, path names the gated user but the gate reads the query", middleware.IdentityAnonymous, "/api/v1/_test/things/5?author_id=6", 204, 1},
		{"anonymous, no author", middleware.IdentityAnonymous, "/api/v1/_test/things/5", 204, 0},
		{"anonymous, malformed author is left to the handler", middleware.IdentityAnonymous, "/api/v1/_test/things/5?author_id=x5", 204, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			probe := &gateProbe{gated: map[int]bool{5: true}}
			resp := gateGet(t, tc.outcome, probe, probe.lookup, tc.path)
			if resp.StatusCode != tc.status {
				t.Fatalf("status %d, want %d", resp.StatusCode, tc.status)
			}
			if got := probe.lookups.Load(); got != tc.lookups {
				t.Errorf("lookups %d, want %d", got, tc.lookups)
			}
			assertCacheControl(t, resp)
			if tc.status == http.StatusNoContent {
				if probe.ran.Load() != 1 {
					t.Errorf("handler ran %d, want 1", probe.ran.Load())
				}
				return
			}
			if probe.ran.Load() != 0 {
				t.Errorf("handler ran %d, want 0", probe.ran.Load())
			}
			if p := decodeProblem(t, resp); p.Code != problem.CodeProfileSignInRequired {
				t.Errorf("code %s, want %s", p.Code, problem.CodeProfileSignInRequired)
			}
		})
	}
}

func TestProfileGateLookupFailureIsInternal(t *testing.T) {
	probe := &gateProbe{err: errors.New("db down")}
	resp := gateGet(t, middleware.IdentityAnonymous, probe, probe.lookup, "/api/v1/_test/users/5")
	if resp.StatusCode != http.StatusInternalServerError || probe.ran.Load() != 0 {
		t.Fatalf("status %d ran %d, want 500 without the handler", resp.StatusCode, probe.ran.Load())
	}
}

func TestProfileGateWithoutLookupFailsClosed(t *testing.T) {
	probe := &gateProbe{}
	resp := gateGet(t, middleware.IdentityAnonymous, probe, nil, "/api/v1/_test/users/5")
	if resp.StatusCode != http.StatusInternalServerError || probe.ran.Load() != 0 {
		t.Fatalf("status %d ran %d, want 500 without the handler", resp.StatusCode, probe.ran.Load())
	}
}

func TestProfileGateDocumentsTheRefusal(t *testing.T) {
	op := ProfileGated(Optional(huma.Operation{
		Description: "Lists things.",
		Responses: map[string]*huma.Response{
			"403": {Description: "PERMISSION_REQUIRED when the caller may not."},
		},
	}), "author_id")
	if !strings.HasSuffix(op.Description, "PROFILE_SIGN_IN_REQUIRED when the caller is anonymous and author_id names a user who shows their profile to signed-in users only.") {
		t.Errorf("description %q", op.Description)
	}
	if got := op.Responses["403"].Description; !strings.HasPrefix(got, "PERMISSION_REQUIRED when the caller may not. PROFILE_SIGN_IN_REQUIRED") {
		t.Errorf("403 %q keeps neither the declared code nor the refusal", got)
	}

	fresh := ProfileGated(Optional(huma.Operation{Description: "Gets a user."}), "user_id")
	if got := fresh.Responses["403"]; got == nil || !strings.Contains(got.Description, "PROFILE_SIGN_IN_REQUIRED") ||
		got.Content[problem.ContentType] == nil {
		t.Errorf("403 %+v", got)
	}
}

func TestProfileGateNeedsTheOptionalTier(t *testing.T) {
	for _, tier := range []func(huma.Operation) huma.Operation{Public, Required} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("no panic for a tier other than optional")
				}
			}()
			ProfileGated(tier(huma.Operation{}), "user_id")
		}()
	}
}
