package app

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/config"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

var updateGolden = flag.Bool("update-routes", false, "rewrite testdata/routes.golden")

const goldenPath = "testdata/routes.golden"

func buildManifest(t *testing.T) []string {
	t.Helper()

	lines := make([]string, 0, 256)
	for _, r := range resolveRoutes(t) {
		lines = append(lines, fmt.Sprintf("%-6s %-46s %s",
			r.Method, r.Path, strings.Join(r.chain, " → ")))
	}
	sort.Strings(lines)
	return lines
}

type route struct {
	fiber.Route
	chain []string
}

func resolveRoutes(t *testing.T) []route {
	t.Helper()
	return routesOf(t, boot(t))
}

func boot(t *testing.T) *App {
	t.Helper()
	a := &App{Fiber: fiber.New(), Config: testConfig()}
	a.setupRoutes()
	return a
}

func routesOf(t *testing.T, a *App) []route {
	t.Helper()

	all, endpoints := a.Fiber.GetRoutes(), a.Fiber.GetRoutes(true)

	var (
		out     []route
		pending []fiber.Route
		method  string
		next    int
	)
	for _, r := range all {
		if r.Method != method {
			method, pending = r.Method, nil
		}
		if next >= len(endpoints) || !sameRoute(r, endpoints[next]) {
			pending = append(pending, r)
			continue
		}
		next++
		if r.Method == fiber.MethodHead {
			continue
		}
		chain := make([]string, 0, len(r.Handlers)+2)
		for _, u := range pending {
			if u.Path == "/" || r.Path == u.Path || strings.HasPrefix(r.Path, u.Path+"/") {
				for _, h := range u.Handlers {
					chain = append(chain, handlerName(h))
				}
			}
		}
		for _, h := range r.Handlers {
			chain = append(chain, handlerName(h))
		}
		out = append(out, route{Route: r, chain: chain})
	}
	if next != len(endpoints) {
		t.Fatalf("resolved %d of %d endpoints — the Use/endpoint lockstep broke",
			next, len(endpoints))
	}
	if len(out) == 0 {
		t.Fatal("no routes registered")
	}
	return out
}

func sameRoute(a, b fiber.Route) bool {
	if a.Method != b.Method || a.Path != b.Path || len(a.Handlers) != len(b.Handlers) {
		return false
	}
	for i := range a.Handlers {
		if reflect.ValueOf(a.Handlers[i]).Pointer() != reflect.ValueOf(b.Handlers[i]).Pointer() {
			return false
		}
	}
	return true
}

func handlerName(h fiber.Handler) string {
	full := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	if i := strings.LastIndex(full, "/"); i >= 0 {
		full = full[i+1:]
	}
	full = strings.TrimSuffix(full, "-fm")
	if i := strings.Index(full, ".func"); i >= 0 {
		full = full[:i]
	}
	full = strings.NewReplacer("(", "", ")", "", "*", "").Replace(full)
	parts := strings.Split(full, ".")
	last := parts[len(parts)-1]
	// A closure is named after wherever its constructor was inlined: Go 1.26
	// (CI) called Idempotent's "setupRoutes.Idempotent", Go 1.27 (local)
	// "middleware.Idempotent", and the golden only ever matched one of them.
	if strings.HasPrefix(last, "Require") || slices.Contains([]string{"Auth", "Idempotent", "v1Headers"}, last) {
		return last
	}
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	return strings.Join(parts, ".")
}

func TestRouteManifest(t *testing.T) {
	got := strings.Join(buildManifest(t), "\n") + "\n"

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Log("golden rewritten — review the diff before committing")
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read %s: %v (regenerate with -update-routes)", goldenPath, err)
	}
	if string(want) == got {
		return
	}

	wantLines, gotLines := strings.Split(string(want), "\n"), strings.Split(got, "\n")
	t.Errorf("route table changed (%d → %d routes). If intended, regenerate:\n"+
		"  go test ./internal/app/ -run TestRouteManifest -update-routes\n"+
		"and READ the diff — a middleware leaking out of its group shows up as a\n"+
		"changed chain on routes you never touched.", len(wantLines)-1, len(gotLines)-1)
	for _, d := range diffLines(wantLines, gotLines) {
		t.Error(d)
	}
}

func diffLines(want, got []string) []string {
	inWant := make(map[string]bool, len(want))
	for _, l := range want {
		inWant[l] = true
	}
	inGot := make(map[string]bool, len(got))
	for _, l := range got {
		inGot[l] = true
	}
	var out []string
	for _, l := range want {
		if l != "" && !inGot[l] {
			out = append(out, "  - "+l)
		}
	}
	for _, l := range got {
		if l != "" && !inWant[l] {
			out = append(out, "  + "+l)
		}
	}
	if len(out) > 40 {
		out = append(out[:40], fmt.Sprintf("  … and %d more", len(out)-40))
	}
	return out
}

// Fails on the exact shape of the 2026-07 gate leak. A gate that escapes its
// group lands on routes that already carry their own, so a second Require* in
// one chain means one of them was never meant to be there. A route legitimately
// needing two capabilities must say so in ONE gate — stacked gates AND together
// silently.
func TestNoRouteCarriesTwoAuthorizationGates(t *testing.T) {
	for _, r := range resolveRoutes(t) {
		var gates []string
		for _, n := range r.chain {
			if strings.HasPrefix(n, "Require") {
				gates = append(gates, n)
			}
		}
		if len(gates) > 1 {
			t.Errorf("%s %s carries %d authorization gates (%s) — one of them "+
				"leaked from an empty-prefix group above it in router.go",
				r.Method, r.Path, len(gates), strings.Join(gates, ", "))
		}
	}
}

func TestAuthorizationSitsBehindAuthentication(t *testing.T) {
	for _, r := range resolveRoutes(t) {
		gated, authed := false, false
		for _, n := range r.chain {
			switch {
			case strings.HasPrefix(n, "Require"):
				gated = true
			case n == "Auth":
				authed = true
			}
		}
		if gated && !authed {
			t.Errorf("%s %s is authorization-gated but not authenticated: %s",
				r.Method, r.Path, strings.Join(r.chain, " → "))
		}
	}
}

var publicWrites = map[string]string{
	"POST /api/trust/callback":      "HMAC X-Trust-Signature over the raw body",
	"POST /api/auth/oauth/callback": "the OAuth authorization code itself",
	"POST /api/auth/logout":         "destroys a session; nothing to protect",
	"POST /api/v1/topics/:topic_id/views": "an anonymous view beacon; it moves only a counter the same reader " +
		"could move by reloading the page",
	"POST /api/v1/toolsets/:toolset_id/resources/:resource_id/downloads": "issues a toolset download link, which " +
		"anonymous readers have always been given; it moves only that resource's download counter",
	"POST /api/v1/galgame-resources/:resource_id/downloads": "issues a galgame resource's links and codes, which " +
		"anonymous readers have always been given; it moves only that resource's download counter",
}

// Checked against the RESOLVED table, not against where the line happens to sit
// in router.go. A write registered above the auth boundary — the mirror image of
// the 2026-07 leak, and the dangerous direction — fails here even though it
// looks perfectly ordinary in the source.
func TestEveryWriteIsAuthenticated(t *testing.T) {
	for _, v := range unauthenticatedWrites(t, boot(t)) {
		t.Error(v)
	}
}

func TestAnOptionalV1WriteFailsTheAuthCheck(t *testing.T) {
	a := boot(t)
	huma.Register(a.APIv1, apiv1.Optional(huma.Operation{
		OperationID: "testOptionalWrite",
		Method:      http.MethodPost,
		Path:        "/_test/optional-write",
		Summary:     "Test-only optional write",
		Description: "A non-GET v1 operation that is not required.",
	}), func(context.Context, *struct{}) (*struct{}, error) {
		return nil, nil
	})
	flagged := slices.ContainsFunc(unauthenticatedWrites(t, a), func(v string) bool {
		return strings.HasPrefix(v, "POST /api/v1/_test/optional-write ")
	})
	if !flagged {
		t.Fatal("an optional v1 POST passed the write-authentication check")
	}
}

func unauthenticatedWrites(t *testing.T, a *App) []string {
	t.Helper()
	var bad []string
	for _, r := range routesOf(t, a) {
		if r.Method == fiber.MethodGet || r.Method == fiber.MethodOptions {
			continue
		}
		if apiv1.IsV1Path(r.Path) {
			if _, ok := publicWrites[r.Method+" "+r.Path]; ok {
				continue
			}
			if apiv1.TierOf(a.APIv1, r.Method, r.Path) != apiv1.TierRequired {
				bad = append(bad, fmt.Sprintf("%s %s mutates without the required v1 tier", r.Method, r.Path))
			}
			continue
		}
		if !strings.HasPrefix(r.Path, "/api") {
			continue
		}
		if _, ok := publicWrites[r.Method+" "+r.Path]; ok {
			continue
		}
		if !slices.Contains(r.chain, "Auth") {
			bad = append(bad, fmt.Sprintf("%s %s mutates without authentication: %s\n"+
				"If that is deliberate, add it to publicWrites with the reason.",
				r.Method, r.Path, strings.Join(r.chain, " → ")))
		}
	}
	return bad
}

func TestLegacyRouteCountRatchet(t *testing.T) {
	raw, err := os.ReadFile("testdata/legacy_route_baseline")
	if err != nil {
		t.Fatalf("read baseline: %v", err)
	}
	baseline, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("parse baseline: %v", err)
	}
	n := 0
	for _, r := range resolveRoutes(t) {
		if !strings.HasPrefix(r.Path, "/api/v1") {
			n++
		}
	}
	if n != baseline {
		t.Fatalf("legacy route count %d, baseline %d: a removed route lowers the baseline in the same commit, "+
			"or the slack lets a new legacy route in unnoticed", n, baseline)
	}
}

func testConfig() *config.Config {
	cfg := &config.Config{}
	cfg.CORS.AllowOrigins = "https://www.kungal.com"
	cfg.Server.Mode = "prod"
	return cfg
}

func TestNoRouteIsShadowed(t *testing.T) {
	byMethod := map[string][]route{}
	for _, r := range resolveRoutes(t) {
		byMethod[r.Method] = append(byMethod[r.Method], r)
	}
	for method, routes := range byMethod {
		for i, later := range routes {
			for _, earlier := range routes[:i] {
				if earlier.Path == later.Path {
					t.Errorf("%s %s is registered twice", method, later.Path)
					continue
				}
				if patternMatches(t, earlier.Path, later.Path) {
					t.Errorf("%s %s is unreachable: %s was registered first and "+
						"matches it (Fiber resolves in registration order, not by "+
						"specificity). Move the static route above the param one.",
						method, later.Path, earlier.Path)
				}
			}
		}
	}
}

func patternMatches(t *testing.T, pat, path string) bool {
	p, q := strings.Split(pat, "/"), strings.Split(path, "/")
	if len(p) != len(q) {
		return false
	}
	for i := range p {
		if strings.HasSuffix(p[i], "?") || strings.Contains(p[i], "*") {
			t.Fatalf("route %q uses an optional/wildcard segment; this check "+
				"assumes neither exists in the router", pat)
		}
		if strings.HasPrefix(p[i], ":") {
			continue
		}
		if p[i] != q[i] {
			return false
		}
	}
	return true
}
