package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
)

type revFeedStub struct {
	mu    sync.Mutex
	total int64
	page  int
	asked []int64
}

func (f *revFeedStub) server(t *testing.T) *catalogclient.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		since := int64(0)
		if cur := r.URL.Query().Get("cursor"); cur != "" {
			raw, _ := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(cur, "cur_"))
			since, _ = strconv.ParseInt(string(raw), 10, 64)
		}
		f.mu.Lock()
		f.asked = append(f.asked, since)
		f.mu.Unlock()

		items := []string{}
		for id := since + 1; id <= f.total && len(items) < f.page; id++ {
			items = append(items, fmt.Sprintf(
				`{"id":"%d","target_object":"work","entity_id":"%d","seq":2,"action":"merged","changed_fields":[],`+
					`"actor_uid":"5","site":"kungal","created_at":"2026-07-30T10:00:00Z"}`, id, 1000+id))
		}
		_, _ = fmt.Fprintf(w, `{"object":"list","items":[%s]}`, strings.Join(items, ","))
	}))
	t.Cleanup(srv.Close)
	return catalogclient.New(catalogclient.Config{
		BaseURL: srv.URL, AppKey: "nmk_test",
	})
}

func TestFeedHeadWalksToTheLastID(t *testing.T) {
	stub := &revFeedStub{total: 250, page: 100}
	s := NewGalgameEditRevisionSync(stub.server(t), nil, nil, nil)
	s.batch = stub.page

	head, err := s.feedHead(context.Background())
	if err != nil {
		t.Fatalf("feedHead: %v", err)
	}
	if head != 250 {
		t.Errorf("head = %d, want 250 (the feed's last id)", head)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	want := []int64{0, 100, 200}
	if len(stub.asked) != len(want) {
		t.Fatalf("asked = %v, want %v", stub.asked, want)
	}
	for i := range want {
		if stub.asked[i] != want[i] {
			t.Fatalf("asked = %v, want %v", stub.asked, want)
		}
	}
}

func TestFeedHeadOnEmptyFeed(t *testing.T) {
	stub := &revFeedStub{total: 0, page: 100}
	s := NewGalgameEditRevisionSync(stub.server(t), nil, nil, nil)
	s.batch = stub.page

	head, err := s.feedHead(context.Background())
	if err != nil {
		t.Fatalf("feedHead: %v", err)
	}
	if head != 0 {
		t.Errorf("head = %d, want 0", head)
	}
}

func TestOnlyEditsReachTheTimeline(t *testing.T) {
	cases := []struct {
		action int16
		want   bool
		name   string
	}{
		{catalogclient.EditActionCreated, false, "created"},
		{catalogclient.EditActionMerged, true, "merged"},
		{catalogclient.EditActionDirect, true, "direct"},
		{catalogclient.EditActionReverted, true, "reverted"},
	}
	for _, c := range cases {
		if got := isTimelineEdit(c.action); got != c.want {
			t.Errorf("isTimelineEdit(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestFeedItemCarriesWorkIDAndRevisionNumber(t *testing.T) {
	var item catalogclient.EditRevisionFeedItem
	raw := `{"id":41,"entity_family":"galgame","entity_type":"galgame.game",
		"entity_id":1207,"seq":8,"action":2,"changed_fields":["galgame.game.name_ja_jp"],
		"actor_uid":9,"amender_uid":3,"proposal_id":null,"site":"kungal",
		"created_at":"2026-07-30T10:00:00Z"}`
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if item.EntityID != 1207 {
		t.Errorf("entity_id = %d, want the work id 1207", item.EntityID)
	}
	if item.Seq != 8 {
		t.Errorf("seq = %d, want the revision number 8", item.Seq)
	}
	if item.ActorUID != 9 {
		t.Errorf("actor_uid = %d — the card attributes the edit to the PROPOSER", item.ActorUID)
	}
}
