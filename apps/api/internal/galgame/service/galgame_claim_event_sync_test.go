package service

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/pkg/catalogclient"

	"github.com/redis/go-redis/v9"
)

func ptr[T any](v T) *T { return &v }

func event(id int64, from *string, to string, productWorkID *int64) *catalogclient.ClaimEventFeedItem {
	return &catalogclient.ClaimEventFeedItem{
		ID: id, WorkID: 900 + id, FromState: from, ToState: to,
		ProductWorkID: productWorkID, Site: "kungal",
	}
}

func TestEffectOfTransition(t *testing.T) {
	gid := ptr(int64(4321))
	cases := []struct {
		name string
		ev   *catalogclient.ClaimEventFeedItem
		want claimEffect
	}{
		{"birth into live seeds the stub", event(1, nil, catalogclient.ClaimStateLive, gid), claimEffectSeedStub},
		{"approval seeds the stub", event(2, ptr(catalogclient.ClaimStatePending), catalogclient.ClaimStateLive, gid), claimEffectSeedStub},
		{"ban unpublishes, never deletes (the children cascade)", event(3, ptr(catalogclient.ClaimStateLive), catalogclient.ClaimStateHidden, gid), claimEffectUnpublish},
		{"submit remembers the submitter", event(4, ptr(catalogclient.ClaimStateDraft), catalogclient.ClaimStatePending, gid), claimEffectRememberSubmitter},
		{"withdrawal does not unpublish", event(5, ptr(catalogclient.ClaimStateLive), catalogclient.ClaimStateDraft, gid), claimEffectNone},
		{"decline does not unpublish", event(6, ptr(catalogclient.ClaimStatePending), catalogclient.ClaimStateDeclined, gid), claimEffectNone},
		{"no product anchor is inert", event(7, nil, catalogclient.ClaimStateLive, nil), claimEffectNone},
		{"zero product anchor is inert", event(8, nil, catalogclient.ClaimStateLive, ptr(int64(0))), claimEffectNone},
		{"an unknown state is reported, not guessed", event(9, nil, "archived", gid), claimEffectUnknownState},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectOf(tc.ev); got != tc.want {
				t.Errorf("effectOf = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOnlyApprovalAwardsFromTheFeed(t *testing.T) {
	gid := ptr(int64(11))
	if !isApproval(event(1, ptr(catalogclient.ClaimStatePending), catalogclient.ClaimStateLive, gid)) {
		t.Error("pending → live is the approval route and must award")
	}
	if isApproval(event(2, ptr(catalogclient.ClaimStateDraft), catalogclient.ClaimStateLive, gid)) {
		t.Error("draft → live is the owner publishing; the request path already awarded it")
	}
	if isApproval(event(3, nil, catalogclient.ClaimStateLive, gid)) {
		t.Error("a claim born live has no submission to reward")
	}
	if isApproval(event(4, ptr(catalogclient.ClaimStatePending), catalogclient.ClaimStateDeclined, gid)) {
		t.Error("a decline is not an approval")
	}
}

func TestClaimantAttribution(t *testing.T) {
	gid := ptr(int64(11))
	s := NewGalgameClaimEventSync(nil, nil, redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1}))

	// draft -> live is the publish half of adoptAndPublish, which the resource
	// lane fires silently on a first download link. Adopting an unclaimed draft
	// is not authorship: 2,073 kungal pages named a creator the retired wiki
	// does not, and the complaint that found it was five games at once.
	adopt := event(1, ptr(catalogclient.ClaimStateDraft), catalogclient.ClaimStateLive, gid)
	adopt.ActorUID = 61516
	if got := s.claimantOf(t.Context(), adopt, 11); got != 0 {
		t.Errorf("draft adopt: claimant = %d, want 0 — taking over an unclaimed draft "+
			"does not make the actor the entry's author", got)
	}

	withdrawn := event(5, ptr(catalogclient.ClaimStateDeclined), catalogclient.ClaimStateLive, gid)
	withdrawn.ActorUID = 61516
	if got := s.claimantOf(t.Context(), withdrawn, 11); got != 0 {
		t.Errorf("declined -> live: claimant = %d, want 0", got)
	}

	born := event(2, nil, catalogclient.ClaimStateLive, gid)
	born.ActorUID = 7
	if got := s.claimantOf(t.Context(), born, 11); got != 7 {
		t.Errorf("born live: claimant = %d, want the event's actor 7", got)
	}

	// gid 0 keeps the local-row fallback out of it: with neither a memo nor a
	// row, an approval must still refuse to name the reviewer.
	approval := event(3, ptr(catalogclient.ClaimStatePending), catalogclient.ClaimStateLive, gid)
	approval.ActorUID = 2
	if got := s.claimantOf(t.Context(), approval, 0); got != 0 {
		t.Errorf("approval without memo: claimant = %d, want 0 (never the reviewer)", got)
	}

	unban := event(4, ptr(catalogclient.ClaimStateHidden), catalogclient.ClaimStateLive, gid)
	unban.ActorUID = 2
	if got := s.claimantOf(t.Context(), unban, 11); got != 0 {
		t.Errorf("unban: claimant = %d, want 0 — lifting a ban does not make the admin the author", got)
	}
}

type claimFeedStub struct {
	mu    sync.Mutex
	total int64
	page  int
	asked []url.Values
}

// The feed is keyset-paged: cursor is cur_<base64url of the decimal watermark>,
// and the page it returns is the events strictly after it.
func decodeWatermark(t *testing.T, cur string) int64 {
	t.Helper()
	if cur == "" {
		return 0
	}
	if !strings.HasPrefix(cur, "cur_") {
		t.Fatalf("cursor %q does not start with cur_ — catalog answers 400 for anything else", cur)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(cur, "cur_"))
	if err != nil {
		t.Fatalf("cursor %q is not base64url: %v", cur, err)
	}
	n, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		t.Fatalf("cursor %q does not carry a decimal event id: %v", cur, err)
	}
	return n
}

func (f *claimFeedStub) client(t *testing.T) *catalogclient.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		f.mu.Lock()
		f.asked = append(f.asked, q)
		f.mu.Unlock()

		if got := r.Header.Get("Authorization"); got != "Bearer nmk_test" {
			t.Errorf("Authorization = %q; the feed is on the application-key plane", got)
		}
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit < 1 || limit > 100 {
			// Not a clamp upstream: 400 LIMIT_TOO_LARGE.
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"status":400,"code":"LIMIT_TOO_LARGE","detail":"limit must be 1-100"}`))
			return
		}

		row := func(id int64) string {
			return fmt.Sprintf(
				`{"object":"claim_event","id":"%d","work_id":"%d","from_state":null,`+
					`"to_state":"live","actor_uid":"7","reason":null,"site":"kungal",`+
					`"product_work_id":"%d","created_at":"2026-07-30T10:00:00Z"}`,
				id, 5000+id, 100+id)
		}
		items, more := []string{}, false
		if q.Get("sort") == "recorded_desc" {
			for id := f.total; id > 0 && len(items) < limit; id-- {
				items = append(items, row(id))
			}
			more = f.total > int64(limit)
		} else {
			since := decodeWatermark(t, q.Get("cursor"))
			id := since + 1
			for ; id <= f.total && len(items) < limit; id++ {
				items = append(items, row(id))
			}
			more = id <= f.total
		}
		body := fmt.Sprintf(`{"object":"list","items":[%s]`, strings.Join(items, ","))
		if more {
			body += fmt.Sprintf(`,"next_cursor":"cur_%s"`,
				base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(int64(len(items)), 10))))
		}
		_, _ = w.Write([]byte(body + "}"))
	}))
	t.Cleanup(srv.Close)
	return catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "nmk_test"})
}

func (f *claimFeedStub) calls() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]url.Values(nil), f.asked...)
}

func TestClaimFeedHeadIsOneRequestNotAWalk(t *testing.T) {
	stub := &claimFeedStub{total: 64996}
	s := NewGalgameClaimEventSync(stub.client(t), nil, nil)

	head, err := s.catalog.ClaimEventHead(t.Context(), claimSite)
	if err != nil {
		t.Fatalf("ClaimEventHead: %v", err)
	}
	if head != 64996 {
		t.Errorf("head = %d, want the newest id 64996", head)
	}
	// Seeding used to walk to the end. At the feed's 100-row ceiling that is 650
	// pages, the 50-page cap stops at 5000, and the cron then replays 60k events
	// — every one of them re-running the reward and the unpublish.
	calls := stub.calls()
	if len(calls) != 1 {
		t.Fatalf("seeding took %d requests, want exactly 1", len(calls))
	}
	if calls[0].Get("sort") != "recorded_desc" || calls[0].Get("limit") != "1" {
		t.Errorf("seed asked %v, want sort=recorded_desc&limit=1", calls[0])
	}
	if calls[0].Get("site") != claimSite {
		t.Errorf("seed asked site=%q, want %q — an unfiltered feed seeds from another product's claims",
			calls[0].Get("site"), claimSite)
	}
}

func TestClaimFeedHeadOnEmptyFeed(t *testing.T) {
	stub := &claimFeedStub{total: 0}
	s := NewGalgameClaimEventSync(stub.client(t), nil, nil)

	head, err := s.catalog.ClaimEventHead(t.Context(), claimSite)
	if err != nil {
		t.Fatalf("ClaimEventHead: %v", err)
	}
	if head != 0 {
		t.Errorf("head = %d, want 0", head)
	}
}

func TestClaimFeedWalkSendsTheFeedsOwnVocabulary(t *testing.T) {
	stub := &claimFeedStub{total: 250}
	s := NewGalgameClaimEventSync(stub.client(t), nil, nil)

	page, err := s.catalog.ClaimEventsSince(t.Context(), 120, s.batch, claimSite)
	if err != nil {
		t.Fatalf("ClaimEventsSince: %v", err)
	}
	q := stub.calls()[0]
	// Catalog ignores a query parameter it does not declare, so a stale spelling
	// never errors — it silently returns the newest events instead of the oldest
	// unread ones, and the watermark walks backwards over history.
	if q.Get("sort") != "recorded_asc" {
		t.Errorf("sort = %q, want recorded_asc", q.Get("sort"))
	}
	if q.Has("since") {
		t.Errorf("still sending since=%q; the keyset cursor replaced it", q.Get("since"))
	}
	if got := decodeWatermark(t, q.Get("cursor")); got != 120 {
		t.Errorf("cursor carries %d, want the stored watermark 120", got)
	}
	if q.Get("site") != claimSite || q.Get("limit") != "100" {
		t.Errorf("asked %v, want site=%s&limit=100", q, claimSite)
	}

	if len(page.Items) != 100 || !page.HasMore {
		t.Fatalf("page = %d items HasMore=%v, want 100/true", len(page.Items), page.HasMore)
	}
	first := page.Items[0]
	if first.ID != 121 || first.WorkID != 5121 || first.ActorUID != 7 {
		t.Errorf("first = %+v, want the decimal STRING ids decoded to 121/5121/7", first)
	}
	if first.ProductWorkID == nil || *first.ProductWorkID != 221 {
		t.Errorf("product_work_id = %v, want 221", first.ProductWorkID)
	}
}

func TestClaimFeedStopsOnTheLastPage(t *testing.T) {
	stub := &claimFeedStub{total: 130}
	s := NewGalgameClaimEventSync(stub.client(t), nil, nil)

	page, err := s.catalog.ClaimEventsSince(t.Context(), 100, s.batch, claimSite)
	if err != nil {
		t.Fatalf("ClaimEventsSince: %v", err)
	}
	// The walk used to stop on a short page. next_cursor is the feed's own
	// end-of-collection signal and a full last page is not short.
	if len(page.Items) != 30 || page.HasMore {
		t.Fatalf("page = %d items HasMore=%v, want 30/false", len(page.Items), page.HasMore)
	}
}

func TestClaimFeedNeverSendsALimitOverTheCeiling(t *testing.T) {
	stub := &claimFeedStub{total: 10}
	s := NewGalgameClaimEventSync(stub.client(t), nil, nil)

	// The forum asked for 1000 a page until 2026-08-28. Over the ceiling the feed
	// answers 400 LIMIT_TOO_LARGE rather than clamping, which stalls the cron
	// outright, so the batch is capped on the way out and not merely configured
	// low.
	if _, err := s.catalog.ClaimEventsSince(t.Context(), 0, 1000, claimSite); err != nil {
		t.Fatalf("a batch of 1000 reached the feed: %v", err)
	}
	if got := stub.calls()[0].Get("limit"); got != "100" {
		t.Errorf("limit = %q, want it capped at 100", got)
	}
	if _, err := s.catalog.ClaimEventsSince(t.Context(), 0, s.batch, claimSite); err != nil {
		t.Fatalf("the configured batch %d was rejected by the feed: %v", s.batch, err)
	}
}

func TestClaimNamespacesAreDistinctFromTheWikiFeed(t *testing.T) {
	if strings.Contains(claimCursorKey, "wiki:") {
		t.Errorf("cursor key %q reuses the wiki namespace", claimCursorKey)
	}
	if claimCursorKey == "wiki:msg:cron:since" {
		t.Errorf("cursor key %q is the retired wiki message cursor", claimCursorKey)
	}
}
