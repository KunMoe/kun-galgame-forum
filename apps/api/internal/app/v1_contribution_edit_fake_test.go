package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// g7bCatalog speaks the nextmoe-infra /v2 editing plane the way infra 5dc86518
// answers it: create/revert/amend return a ProposalRecord, decisions return a
// decision record, the moderation queue serves open only, and no face carries a
// decision note.
type g7bCatalog struct {
	mu  sync.Mutex
	srv *httptest.Server

	actors    map[string]g7bActor
	owners    map[int64]int
	proposals map[int64]*g7bProposal
	revisions []*g7bRevision
	snapshots map[int64]map[string]any
	idem      map[string]g7bIdem
	nextProp  int64
	nextRev   int64
	nextAmend int64
	calls     []g7bCall

	failRevisions bool
	failVocab     bool
	emptyState    int64
}

type g7bActor struct {
	uid       int
	moderator bool
	trusted   bool
	noScope   bool
	quota     bool
}

type g7bProposal struct {
	id         int64
	workID     int64
	site       string
	entityType string
	proposer   int
	note       string
	state      string
	patch      map[string]any
	amendments []g7bAmendment
	decider    int
	decidedAt  *time.Time
	baseSeq    int
	created    time.Time
	updated    time.Time
}

type g7bAmendment struct {
	id      int64
	seq     int
	amender int
	note    string
	set     map[string]any
	unset   []string
	created time.Time
}

type g7bRevision struct {
	id         int64
	workID     int64
	seq        int
	action     string
	changed    []string
	actor      int
	amender    int
	proposalID int64
	site       string
	created    time.Time
	snapshot   map[string]any
}

type g7bIdem struct {
	body string
	id   int64
}

type g7bCall struct {
	method  string
	path    string
	query   url.Values
	ifMatch string
	idemKey string
	auth    string
	body    string
}

const (
	g7bAppKey   = "g7b-app-key"
	g7bSite     = "kungal"
	g7bWorkType = "catalog.work"
)

var g7bClock = time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC)

func newG7bCatalog(t *testing.T) *g7bCatalog {
	t.Helper()
	c := &g7bCatalog{
		actors:    map[string]g7bActor{},
		owners:    map[int64]int{},
		proposals: map[int64]*g7bProposal{},
		snapshots: map[int64]map[string]any{},
		idem:      map[string]g7bIdem{},
		nextProp:  957100001,
		nextRev:   957200001,
		nextAmend: 957300001,
	}
	c.srv = httptest.NewServer(http.HandlerFunc(c.serve))
	t.Cleanup(c.srv.Close)
	return c
}

func (c *g7bCatalog) tick() time.Time {
	g7bClock = g7bClock.Add(time.Second)
	return g7bClock
}

func (c *g7bCatalog) etag(p *g7bProposal) string {
	return `"p` + strconv.FormatInt(p.id, 10) + "." + strconv.FormatInt(p.updated.UnixMicro(), 10) + `"`
}

func (c *g7bCatalog) etagOf(id int64) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.etag(c.proposals[id])
}

func (c *g7bCatalog) addProposal(p *g7bProposal) *g7bProposal {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p.id == 0 {
		p.id = c.nextProp
		c.nextProp++
	}
	if p.site == "" {
		p.site = g7bSite
	}
	if p.entityType == "" {
		p.entityType = g7bWorkType
	}
	if p.state == "" {
		p.state = "open"
	}
	if p.patch == nil {
		p.patch = map[string]any{}
	}
	if p.created.IsZero() {
		p.created = c.tick()
	}
	if p.updated.IsZero() {
		p.updated = p.created
	}
	c.proposals[p.id] = p
	return p
}

func (c *g7bCatalog) addRevision(workID int64, action string, actor int, proposalID int64, snapshot map[string]any, changed ...string) *g7bRevision {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.addRevisionLocked(workID, action, actor, 0, proposalID, snapshot, changed)
}

func (c *g7bCatalog) addRevisionLocked(workID int64, action string, actor, amender int, proposalID int64, snapshot map[string]any, changed []string) *g7bRevision {
	seq := 1
	for _, r := range c.revisions {
		if r.workID == workID && r.seq >= seq {
			seq = r.seq + 1
		}
	}
	if changed == nil {
		changed = []string{}
	}
	r := &g7bRevision{
		id: c.nextRev, workID: workID, seq: seq, action: action, changed: changed,
		actor: actor, amender: amender, proposalID: proposalID, site: g7bSite,
		created: c.tick(), snapshot: snapshot,
	}
	c.nextRev++
	c.revisions = append(c.revisions, r)
	cur := c.snapshots[workID]
	if cur == nil {
		cur = map[string]any{}
	}
	for k, v := range snapshot {
		cur[k] = v
	}
	c.snapshots[workID] = cur
	return r
}

func (c *g7bCatalog) record(r *http.Request, body string) {
	c.calls = append(c.calls, g7bCall{
		method: r.Method, path: r.URL.Path, query: r.URL.Query(),
		ifMatch: r.Header.Get("If-Match"), idemKey: r.Header.Get("Idempotency-Key"),
		auth: strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), body: body,
	})
}

func (c *g7bCatalog) callsTo(method, prefix string) []g7bCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []g7bCall
	for _, call := range c.calls {
		if call.method == method && strings.HasPrefix(call.path, prefix) {
			out = append(out, call)
		}
	}
	return out
}

func (c *g7bCatalog) resetCalls() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = nil
}

func (c *g7bCatalog) proposal(id int64) g7bProposal {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.proposals[id]
	if p == nil {
		return g7bProposal{}
	}
	return *p
}

func (c *g7bCatalog) proposalCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.proposals)
}

func g7bProblem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "title": code, "detail": detail, "status": status})
}

func g7bJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (c *g7bCatalog) serve(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.record(r, string(raw))
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	path := r.URL.Path

	if strings.HasPrefix(path, "/v2/catalog/") || path == "/v2/vocabularies" {
		if token != g7bAppKey && !strings.HasPrefix(path, "/v2/catalog/schemas/") {
			g7bProblem(w, http.StatusUnauthorized, "INVALID_CREDENTIAL", "app key refused")
			return
		}
		c.serveApp(w, r, path)
		return
	}
	actor, ok := c.actors[token]
	if !ok {
		g7bProblem(w, http.StatusUnauthorized, "INVALID_CREDENTIAL", "token refused")
		return
	}
	if actor.noScope {
		g7bProblem(w, http.StatusForbidden, "SCOPE_REQUIRED", "this operation requires the catalog:edit scope")
		return
	}
	if actor.quota {
		w.Header().Set("Retry-After", "30")
		g7bProblem(w, http.StatusTooManyRequests, "QUOTA_EXCEEDED", "daily quota used up")
		return
	}
	seg := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case path == "/v2/me/proposals" && r.Method == http.MethodGet:
		c.listMine(w, r, actor)
	case path == "/v2/me/proposals" && r.Method == http.MethodPost:
		c.create(w, r, actor, raw)
	case len(seg) == 4 && seg[1] == "me" && seg[2] == "proposals" && r.Method == http.MethodGet:
		c.getMine(w, r, actor, seg[3])
	case len(seg) == 4 && seg[1] == "me" && seg[2] == "proposals" && r.Method == http.MethodPatch:
		c.patchMine(w, r, actor, seg[3], raw)
	case len(seg) == 5 && seg[1] == "me" && seg[4] == "amendments" && r.Method == http.MethodPost:
		c.amend(w, r, actor, seg[3], raw)
	case path == "/v2/moderation/proposals" && r.Method == http.MethodGet:
		c.listQueue(w, r, actor)
	case len(seg) == 4 && seg[1] == "moderation" && seg[2] == "proposals" && r.Method == http.MethodGet:
		c.getModeration(w, r, actor, seg[3])
	case len(seg) == 5 && seg[1] == "moderation" && seg[4] == "decisions" && r.Method == http.MethodPost:
		c.decide(w, r, actor, seg[3], raw)
	case path == "/v2/moderation/reverts" && r.Method == http.MethodPost:
		c.revert(w, r, actor, raw)
	case len(seg) == 5 && seg[1] == "moderation" && seg[2] == "snapshots" && r.Method == http.MethodGet:
		id, _ := strconv.ParseInt(seg[4], 10, 64)
		vals, ok := c.snapshots[id]
		if !ok || seg[3] != "work" {
			g7bProblem(w, http.StatusNotFound, "NOT_FOUND", "no such entity")
			return
		}
		g7bJSON(w, http.StatusOK, map[string]any{
			"object": "snapshot", "entity_type": g7bWorkType, "entity_id": strconv.FormatInt(id, 10), "field_values": vals,
		})
	default:
		g7bProblem(w, http.StatusNotFound, "NOT_FOUND", "no route "+r.Method+" "+path)
	}
}

func (c *g7bCatalog) serveApp(w http.ResponseWriter, r *http.Request, path string) {
	q := r.URL.Query()
	seg := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case path == "/v2/vocabularies":
		if c.failVocab {
			g7bProblem(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "down")
			return
		}
		g7bJSON(w, http.StatusOK, map[string]any{"object": "list", "items": []map[string]any{
			{"name": "content_rating", "closed": true, "values": []map[string]any{
				{"value": "all_ages", "display_name": "All ages"}, {"value": "sensitive", "display_name": "Sensitive"},
				{"value": "r18", "display_name": "R18", "description": "Adults only"},
			}},
		}})
	case path == "/v2/catalog/schemas/work":
		enc := "int"
		g7bJSON(w, http.StatusOK, map[string]any{
			"object": "object_schema", "target_object": "work", "entity_type": g7bWorkType,
			"include": []string{}, "full_set": []string{}, "list_include": []string{}, "list_full_set": []string{},
			"creation_disabled": false,
			"fields": []map[string]any{
				{"key": "catalog.work.display_name", "field_type": "text", "diff_hint": "inline", "deprecated": false, "max_suppressed": 0, "max_elements": 0, "vocabulary": "", "base": 0, "nullable": false},
				{"key": "catalog.work.content_rating", "field_type": "enum", "diff_hint": "inline", "deprecated": false, "max_suppressed": 0, "max_elements": 0, "vocabulary": "content_rating", "encoding": enc, "base": 0, "nullable": false},
				{"key": "catalog.work.titles", "field_type": "list", "diff_hint": "items", "deprecated": false, "max_suppressed": 50, "max_elements": 100, "vocabulary": "", "base": 0, "nullable": false,
					"element": map[string]any{"type": "object", "members": []map[string]any{
						{"key": "lang", "type": "text", "vocabulary": "", "base": 0, "nullable": true},
						{"key": "title", "type": "text", "vocabulary": "", "base": 0, "nullable": false},
					}}},
				{"key": "catalog.work.tag_ids", "field_type": "list", "diff_hint": "items", "deprecated": false, "max_suppressed": 0, "max_elements": 500, "vocabulary": "", "base": 0, "nullable": false,
					"element": map[string]any{"type": "ref", "members": []map[string]any{}}},
				{"key": "catalog.work.legacy_flag", "field_type": "text", "diff_hint": "inline", "deprecated": true, "max_suppressed": 0, "max_elements": 0, "vocabulary": "", "base": 0, "nullable": true},
				{"key": "catalog.work.future_thing", "field_type": "hologram", "diff_hint": "inline", "deprecated": false, "max_suppressed": 0, "max_elements": 0, "vocabulary": "", "base": 0, "nullable": true},
			},
		})
	case path == "/v2/catalog/proposals":
		c.listPublic(w, q)
	case len(seg) == 4 && seg[2] == "proposals":
		id, _ := strconv.ParseInt(seg[3], 10, 64)
		p := c.proposals[id]
		if p == nil {
			g7bProblem(w, http.StatusNotFound, "NOT_FOUND", "No proposal with this id.")
			return
		}
		g7bJSON(w, http.StatusOK, c.record_(p, false))
	case path == "/v2/catalog/revisions":
		c.listRevisions(w, q)
	case len(seg) == 4 && seg[2] == "revisions":
		c.getRevision(w, seg[3], q)
	default:
		g7bProblem(w, http.StatusNotFound, "NOT_FOUND", "no route "+path)
	}
}

func (c *g7bCatalog) reviews(actor g7bActor, workID int64) bool {
	return actor.moderator || (c.owners[workID] != 0 && c.owners[workID] == actor.uid)
}

func (c *g7bCatalog) record_(p *g7bProposal, withPatch bool) map[string]any {
	state := p.state
	if p.id == c.emptyState {
		state = ""
	}
	rec := map[string]any{
		"object": "proposal", "id": strconv.FormatInt(p.id, 10), "state": state,
		"target_object": "work", "entity_type": p.entityType, "entity_id": strconv.FormatInt(p.workID, 10),
		"note": p.note, "proposer_uid": strconv.Itoa(p.proposer), "site": p.site,
		"base_revision_seq": p.baseSeq, "decided_by_uid": nil, "decided_at": nil,
		"created_at": p.created.Format(time.RFC3339), "updated_at": p.updated.Format(time.RFC3339),
	}
	if p.decider != 0 {
		rec["decided_by_uid"] = strconv.Itoa(p.decider)
	}
	if p.decidedAt != nil {
		rec["decided_at"] = p.decidedAt.Format(time.RFC3339)
	}
	if withPatch {
		rec["patch"] = p.patch
		eff := map[string]any{}
		for k, v := range p.patch {
			eff[k] = v
		}
		for _, a := range p.amendments {
			for k, v := range a.set {
				eff[k] = v
			}
			for _, k := range a.unset {
				delete(eff, k)
			}
		}
		rec["effective_patch"] = eff
	}
	return rec
}

func (c *g7bCatalog) withAmendments(rec map[string]any, p *g7bProposal) map[string]any {
	list := make([]map[string]any, 0, len(p.amendments))
	for _, a := range p.amendments {
		list = append(list, map[string]any{
			"object": "amendment", "id": strconv.FormatInt(a.id, 10), "seq": a.seq,
			"amender_uid": strconv.Itoa(a.amender), "note": a.note, "created_at": a.created.Format(time.RFC3339),
		})
	}
	rec["amendments"] = list
	return rec
}

func (c *g7bCatalog) page(w http.ResponseWriter, q url.Values, rows []*g7bProposal, withPatch bool) {
	sort.Slice(rows, func(i, j int) bool { return rows[i].id > rows[j].id })
	if cur := q.Get("cursor"); cur != "" {
		before, err := strconv.ParseInt(strings.TrimPrefix(cur, "cur_"), 10, 64)
		if err != nil || !strings.HasPrefix(cur, "cur_") {
			g7bProblem(w, http.StatusBadRequest, "INVALID_CURSOR", "bad cursor")
			return
		}
		kept := rows[:0]
		for _, p := range rows {
			if p.id < before {
				kept = append(kept, p)
			}
		}
		rows = kept
	}
	limit := 20
	if l := q.Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
		if limit > 100 {
			g7bProblem(w, http.StatusBadRequest, "LIMIT_TOO_LARGE", "limit")
			return
		}
	}
	body := map[string]any{"object": "list"}
	if len(rows) > limit {
		rows = rows[:limit]
		body["next_cursor"] = "cur_" + strconv.FormatInt(rows[len(rows)-1].id, 10)
	}
	items := make([]map[string]any, 0, len(rows))
	for _, p := range rows {
		items = append(items, c.record_(p, withPatch))
	}
	body["items"] = items
	g7bJSON(w, http.StatusOK, body)
}

func g7bStateOK(s string) bool {
	return s == "" || s == "open" || s == "pending" || s == "merged" || s == "declined" || s == "withdrawn"
}

func (c *g7bCatalog) listMine(w http.ResponseWriter, r *http.Request, actor g7bActor) {
	q := r.URL.Query()
	if !g7bStateOK(q.Get("state")) {
		g7bProblem(w, http.StatusBadRequest, "UNKNOWN_ENUM_VALUE", "state")
		return
	}
	var rows []*g7bProposal
	for _, p := range c.proposals {
		if p.proposer != actor.uid || p.site != g7bSite {
			continue
		}
		if et := q.Get("entity_type"); et != "" && p.entityType != et {
			continue
		}
		if id := q.Get("entity_id"); id != "" && strconv.FormatInt(p.workID, 10) != id {
			continue
		}
		if st := q.Get("state"); st != "" && p.state != st {
			continue
		}
		rows = append(rows, p)
	}
	c.page(w, q, rows, false)
}

func (c *g7bCatalog) listQueue(w http.ResponseWriter, r *http.Request, actor g7bActor) {
	q := r.URL.Query()
	if !actor.moderator {
		g7bProblem(w, http.StatusForbidden, "PERMISSION_REQUIRED", "the whole queue requires a catalog review permission")
		return
	}
	var rows []*g7bProposal
	for _, p := range c.proposals {
		if et := q.Get("entity_type"); et != "" && p.entityType != et {
			continue
		}
		if p.site == g7bSite && p.state == "open" {
			rows = append(rows, p)
		}
	}
	c.page(w, q, rows, false)
}

func (c *g7bCatalog) listPublic(w http.ResponseWriter, q url.Values) {
	var rows []*g7bProposal
	for _, p := range c.proposals {
		if s := q.Get("site"); s != "" && p.site != s {
			continue
		}
		if q.Get("object") == "work" && p.entityType != g7bWorkType {
			continue
		}
		if id := q.Get("entity_id"); id != "" && strconv.FormatInt(p.workID, 10) != id {
			continue
		}
		if st := q.Get("state"); st != "" && p.state != st {
			continue
		}
		rows = append(rows, p)
	}
	c.page(w, q, rows, false)
}

func (c *g7bCatalog) lookup(w http.ResponseWriter, raw string) *g7bProposal {
	id, _ := strconv.ParseInt(raw, 10, 64)
	p := c.proposals[id]
	if p == nil {
		g7bProblem(w, http.StatusNotFound, "NOT_FOUND", "No proposal with this id.")
	}
	return p
}

func (c *g7bCatalog) detail(w http.ResponseWriter, r *http.Request, p *g7bProposal, status int) {
	include := r.URL.Query().Get("include")
	rec := c.record_(p, strings.Contains(include, "patch"))
	if strings.Contains(include, "amendments") || r.Method != http.MethodGet {
		rec = c.withAmendments(rec, p)
	}
	w.Header().Set("ETag", c.etag(p))
	g7bJSON(w, status, rec)
}

func (c *g7bCatalog) getMine(w http.ResponseWriter, r *http.Request, actor g7bActor, raw string) {
	p := c.lookup(w, raw)
	if p == nil {
		return
	}
	if p.proposer != actor.uid {
		g7bProblem(w, http.StatusNotFound, "NOT_FOUND", "No proposal with this id.")
		return
	}
	c.detail(w, r, p, http.StatusOK)
}

func (c *g7bCatalog) getModeration(w http.ResponseWriter, r *http.Request, actor g7bActor, raw string) {
	p := c.lookup(w, raw)
	if p == nil {
		return
	}
	if p.site != g7bSite {
		g7bProblem(w, http.StatusForbidden, "TENANT_MISMATCH", "this proposal was filed on another catalog site.")
		return
	}
	if !c.reviews(actor, p.workID) {
		g7bProblem(w, http.StatusForbidden, "PERMISSION_REQUIRED", "needs review standing on this entity")
		return
	}
	c.detail(w, r, p, http.StatusOK)
}

func (c *g7bCatalog) precondition(w http.ResponseWriter, r *http.Request, p *g7bProposal) bool {
	im := strings.TrimSpace(r.Header.Get("If-Match"))
	if im == "" {
		g7bProblem(w, http.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "this operation requires If-Match.")
		return false
	}
	if im != "*" && im != c.etag(p) {
		g7bProblem(w, http.StatusPreconditionFailed, "PRECONDITION_FAILED", "If-Match did not match the current representation.")
		return false
	}
	return true
}

func (c *g7bCatalog) replay(w http.ResponseWriter, r *http.Request, raw []byte) (bool, string) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		return false, ""
	}
	if prior, ok := c.idem[key]; ok {
		if prior.body != string(raw) {
			g7bProblem(w, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED", "The same Idempotency-Key was sent with a different request.")
			return true, key
		}
		p := c.proposals[prior.id]
		w.Header().Set("ETag", c.etag(p))
		g7bJSON(w, http.StatusCreated, c.record_(p, false))
		return true, key
	}
	return false, key
}

func (c *g7bCatalog) create(w http.ResponseWriter, r *http.Request, actor g7bActor, raw []byte) {
	if done, _ := c.replay(w, r, raw); done {
		return
	}
	var in struct {
		EntityType string         `json:"entity_type"`
		EntityID   string         `json:"entity_id"`
		Patch      map[string]any `json:"patch"`
		Note       string         `json:"note"`
	}
	_ = json.Unmarshal(raw, &in)
	if len(in.Patch) == 0 {
		g7bProblem(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "patch is empty")
		return
	}
	if _, bad := in.Patch["catalog.work.titles"].(string); bad {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "VALIDATION_FAILED", "detail": "titles must be a list",
			"errors": []map[string]any{{"pointer": "/patch/catalog.work.titles", "reason": "INVALID_FORMAT", "detail": "must be a list"}},
		})
		return
	}
	workID, _ := strconv.ParseInt(in.EntityID, 10, 64)
	now := c.tick()
	p := &g7bProposal{
		id: c.nextProp, workID: workID, site: g7bSite, entityType: in.EntityType, proposer: actor.uid,
		note: in.Note, state: "open", patch: in.Patch, created: now, updated: now, baseSeq: c.seqOf(workID),
	}
	c.nextProp++
	c.proposals[p.id] = p
	if key := r.Header.Get("Idempotency-Key"); key != "" {
		c.idem[key] = g7bIdem{body: string(raw), id: p.id}
	}
	if actor.trusted {
		c.mergeLocked(p, actor.uid, "merged")
	}
	w.Header().Set("ETag", c.etag(p))
	g7bJSON(w, http.StatusCreated, c.record_(p, false))
}

func (c *g7bCatalog) seqOf(workID int64) int {
	seq := 0
	for _, r := range c.revisions {
		if r.workID == workID && r.seq > seq {
			seq = r.seq
		}
	}
	return seq
}

func (c *g7bCatalog) mergeLocked(p *g7bProposal, decider int, action string) {
	now := c.tick()
	p.state = "merged"
	p.decider = decider
	p.decidedAt = &now
	p.updated = now
	amender := 0
	if n := len(p.amendments); n > 0 {
		amender = p.amendments[n-1].amender
	}
	changed := make([]string, 0, len(p.patch))
	for k := range p.patch {
		changed = append(changed, k)
	}
	sort.Strings(changed)
	c.addRevisionLocked(p.workID, action, p.proposer, amender, p.id, p.patch, changed)
}

func (c *g7bCatalog) patchMine(w http.ResponseWriter, r *http.Request, actor g7bActor, raw string, body []byte) {
	p := c.lookup(w, raw)
	if p == nil {
		return
	}
	if p.proposer != actor.uid && !c.reviews(actor, p.workID) {
		g7bProblem(w, http.StatusForbidden, "PERMISSION_REQUIRED", "only the proposer may change this proposal")
		return
	}
	if !c.precondition(w, r, p) {
		return
	}
	var in struct {
		State string `json:"state"`
	}
	_ = json.Unmarshal(body, &in)
	if in.State != "withdrawn" {
		g7bProblem(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "patch must withdraw or amend.")
		return
	}
	if p.state != "open" {
		g7bProblem(w, http.StatusConflict, "INVALID_STATE_TRANSITION", "proposal is not open")
		return
	}
	p.state = "withdrawn"
	p.updated = c.tick()
	c.detail(w, r, p, http.StatusOK)
}

func (c *g7bCatalog) amend(w http.ResponseWriter, r *http.Request, actor g7bActor, raw string, body []byte) {
	p := c.lookup(w, raw)
	if p == nil {
		return
	}
	if p.proposer != actor.uid && !c.reviews(actor, p.workID) {
		g7bProblem(w, http.StatusForbidden, "PERMISSION_REQUIRED", "only the proposer, or someone with review standing on this entity, may change this proposal.")
		return
	}
	if !c.precondition(w, r, p) {
		return
	}
	if p.state != "open" {
		g7bProblem(w, http.StatusConflict, "INVALID_STATE_TRANSITION", "proposal is not open")
		return
	}
	var in struct {
		Set   map[string]any `json:"set"`
		Unset []string       `json:"unset"`
		Note  string         `json:"note"`
	}
	_ = json.Unmarshal(body, &in)
	now := c.tick()
	p.amendments = append(p.amendments, g7bAmendment{
		id: c.nextAmend, seq: len(p.amendments) + 1, amender: actor.uid, note: in.Note,
		set: in.Set, unset: in.Unset, created: now,
	})
	c.nextAmend++
	p.updated = now
	c.detail(w, r, p, http.StatusCreated)
}

func (c *g7bCatalog) decide(w http.ResponseWriter, r *http.Request, actor g7bActor, raw string, body []byte) {
	p := c.lookup(w, raw)
	if p == nil {
		return
	}
	if p.site != g7bSite {
		g7bProblem(w, http.StatusForbidden, "TENANT_MISMATCH", "this proposal was filed on another catalog site.")
		return
	}
	if !c.precondition(w, r, p) {
		return
	}
	if !c.reviews(actor, p.workID) {
		g7bProblem(w, http.StatusForbidden, "PERMISSION_REQUIRED", "no review standing")
		return
	}
	if p.state != "open" {
		g7bProblem(w, http.StatusConflict, "DECISION_ALREADY_MADE", "this proposal was already decided")
		return
	}
	var in struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	_ = json.Unmarshal(body, &in)
	from := p.state
	switch in.Decision {
	case "merge":
		c.mergeLocked(p, actor.uid, "merged")
	case "decline":
		now := c.tick()
		p.state, p.decider, p.decidedAt, p.updated = "declined", actor.uid, &now, now
	default:
		g7bProblem(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "decision must be merge or decline.")
		return
	}
	g7bJSON(w, http.StatusCreated, map[string]any{
		"object": "decision", "id": strconv.FormatInt(p.id, 10), "decision": in.Decision,
		"note": in.Note, "from_state": from, "to_state": p.state,
	})
}

func (c *g7bCatalog) revert(w http.ResponseWriter, r *http.Request, actor g7bActor, raw []byte) {
	if done, _ := c.replay(w, r, raw); done {
		return
	}
	var in struct {
		RevisionID string `json:"revision_id"`
		Reason     string `json:"reason"`
	}
	_ = json.Unmarshal(raw, &in)
	rid, _ := strconv.ParseInt(in.RevisionID, 10, 64)
	var target *g7bRevision
	for _, rev := range c.revisions {
		if rev.id == rid {
			target = rev
		}
	}
	if target == nil {
		g7bProblem(w, http.StatusNotFound, "NOT_FOUND", "no revision")
		return
	}
	snap := map[string]any{}
	for _, rev := range c.revisions {
		if rev.workID == target.workID && rev.seq <= target.seq {
			for k, v := range rev.snapshot {
				snap[k] = v
			}
		}
	}
	now := c.tick()
	p := &g7bProposal{
		id: c.nextProp, workID: target.workID, site: g7bSite, entityType: g7bWorkType, proposer: actor.uid,
		note: in.Reason, state: "open", patch: snap, created: now, updated: now, baseSeq: c.seqOf(target.workID),
	}
	c.nextProp++
	c.proposals[p.id] = p
	if key := r.Header.Get("Idempotency-Key"); key != "" {
		c.idem[key] = g7bIdem{body: string(raw), id: p.id}
	}
	if c.reviews(actor, target.workID) {
		c.mergeLocked(p, actor.uid, "reverted")
	}
	w.Header().Set("ETag", c.etag(p))
	g7bJSON(w, http.StatusCreated, c.record_(p, false))
}

func (c *g7bCatalog) revisionRecord(rev *g7bRevision) map[string]any {
	rec := map[string]any{
		"object": "revision", "id": strconv.FormatInt(rev.id, 10), "target_object": "work",
		"entity_id": strconv.FormatInt(rev.workID, 10), "site_work_id": strconv.FormatInt(rev.workID, 10),
		"seq": rev.seq, "action": rev.action, "changed_fields": rev.changed,
		"actor_uid": strconv.Itoa(rev.actor), "amender_uid": nil, "proposal_id": nil,
		"site": rev.site, "created_at": rev.created.Format(time.RFC3339),
	}
	if rev.amender != 0 {
		rec["amender_uid"] = strconv.Itoa(rev.amender)
	}
	if rev.proposalID != 0 {
		rec["proposal_id"] = strconv.FormatInt(rev.proposalID, 10)
	}
	return rec
}

func (c *g7bCatalog) listRevisions(w http.ResponseWriter, q url.Values) {
	if c.failRevisions {
		g7bProblem(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "down")
		return
	}
	var rows []*g7bRevision
	for _, rev := range c.revisions {
		if id := q.Get("entity_id"); id != "" && strconv.FormatInt(rev.workID, 10) != id {
			continue
		}
		rows = append(rows, rev)
	}
	desc := q.Get("sort") != "recorded_asc"
	sort.Slice(rows, func(i, j int) bool {
		if desc {
			return rows[i].id > rows[j].id
		}
		return rows[i].id < rows[j].id
	})
	if cur := q.Get("cursor"); cur != "" {
		pivot, _ := strconv.ParseInt(strings.TrimPrefix(cur, "cur_"), 10, 64)
		kept := rows[:0]
		for _, rev := range rows {
			if (desc && rev.id < pivot) || (!desc && rev.id > pivot) {
				kept = append(kept, rev)
			}
		}
		rows = kept
	}
	limit := 20
	if l := q.Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}
	body := map[string]any{"object": "list"}
	if len(rows) > limit {
		rows = rows[:limit]
		body["next_cursor"] = "cur_" + strconv.FormatInt(rows[len(rows)-1].id, 10)
	}
	items := make([]map[string]any, 0, len(rows))
	for _, rev := range rows {
		items = append(items, c.revisionRecord(rev))
	}
	body["items"] = items
	g7bJSON(w, http.StatusOK, body)
}

func (c *g7bCatalog) stateAt(workID int64, seq int) map[string]any {
	snap := map[string]any{}
	for _, rev := range c.revisions {
		if rev.workID == workID && rev.seq <= seq {
			for k, v := range rev.snapshot {
				snap[k] = v
			}
		}
	}
	return snap
}

func (c *g7bCatalog) getRevision(w http.ResponseWriter, raw string, q url.Values) {
	id, _ := strconv.ParseInt(raw, 10, 64)
	var rev, base *g7bRevision
	baseID, _ := strconv.ParseInt(q.Get("diff_base"), 10, 64)
	for _, r := range c.revisions {
		if r.id == id {
			rev = r
		}
		if r.id == baseID {
			base = r
		}
	}
	if rev == nil {
		g7bProblem(w, http.StatusNotFound, "NOT_FOUND", "no revision")
		return
	}
	rec := c.revisionRecord(rev)
	if q.Get("include") == "diff" {
		from := map[string]any{}
		if base != nil {
			from = c.stateAt(base.workID, base.seq)
			rec["diff_base"] = strconv.FormatInt(base.id, 10)
		}
		to := c.stateAt(rev.workID, rev.seq)
		keys := make([]string, 0)
		for k := range to {
			keys = append(keys, k)
		}
		for k := range from {
			if !slices.Contains(keys, k) {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		diff := make([]map[string]any, 0)
		for _, k := range keys {
			a, _ := json.Marshal(from[k])
			b, _ := json.Marshal(to[k])
			if string(a) != string(b) {
				diff = append(diff, map[string]any{"key": k, "from": from[k], "to": to[k]})
			}
		}
		rec["diff"] = diff
	}
	g7bJSON(w, http.StatusOK, rec)
}
