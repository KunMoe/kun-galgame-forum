package agentmcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/apiv1/content"
	searchRepo "kun-galgame-api/internal/search/repository"
	"kun-galgame-api/internal/testdb"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/userclient"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	userAlice  = 953000001
	userBob    = 953000002
	userCarol  = 953000003
	userBanned = 953000009

	topicPublic  = 953000101
	topicHidden  = 953000102
	topicNSFW    = 953000103
	topicPrivate = 953000104
	topicBanned  = 953000105
	topicDaily   = 953000106

	replyBest   = 953000201
	replyLiked  = 953000202
	replyBanned = 953000203
	replyPlain  = 953000204
)

type fakeVerifier map[string]*oauth.AccessClaims

func (f fakeVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	if c, ok := f[raw]; ok {
		return c, nil
	}
	return nil, errors.New("bad token")
}

type fakeProfiles map[string]*oauth.UserInfo

func (f fakeProfiles) FetchUserInfo(token string) (*oauth.UserInfo, error) {
	if u, ok := f[token]; ok {
		return u, nil
	}
	return nil, errors.New("userinfo down")
}

type bearer string

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	if b != "" {
		r.Header.Set("Authorization", "Bearer "+string(b))
	}
	return http.DefaultTransport.RoundTrip(r)
}

type fixture struct {
	url string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testdb.Open(t)

	accounts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := map[int]int{userAlice: 0, userBob: 0, userCarol: 0, userBanned: 1}
		var users []map[string]any
		for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
			id, _ := strconv.Atoi(raw)
			if s, ok := status[id]; ok {
				users = append(users, map[string]any{"id": id, "name": "u" + raw[len(raw)-1:], "status": s, "roles": []string{"user"}})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"users": users, "not_found": []int{}}})
	}))
	t.Cleanup(accounts.Close)
	uc := userclient.New(userclient.Config{
		BaseURL: accounts.URL, ClientID: "c", ClientSecret: "s",
		ImageCDNBase: "https://image.test.example", HTTPTimeout: 2 * time.Second,
	})

	claims := func(id int, roles ...string) *oauth.AccessClaims {
		return &oauth.AccessClaims{ID: id, Name: "n", Roles: append([]string{"user"}, roles...)}
	}
	h := Handler(Deps{
		Verifier: fakeVerifier{
			"alice": claims(userAlice), "bob": claims(userBob), "carol": claims(userCarol),
			"admin": claims(userCarol, "admin"), "bob-no-profile": claims(userBob),
		},
		Profiles: fakeProfiles{
			"alice": {AdultConfirmed: true, NSFWDisplay: "hide"},
			"bob":   {AdultConfirmed: true, NSFWDisplay: "show"},
			"carol": {AdultConfirmed: false, NSFWDisplay: "show"},
			"admin": {AdultConfirmed: true, NSFWDisplay: "show"},
		},
		Search: searchRepo.NewSearchRepository(db),
		Topics: topicapiv1.New(nil, topicRepo.NewTopicRepository(db), topicRepo.NewTopicTaxonomyRepository(db),
			topicRepo.NewReplyRepository(db), nil, uc, &content.Converter{}, "https://image.test.example"),
		Rows:     topicRepo.NewTopicRepository(db),
		Taxonomy: topicRepo.NewTopicTaxonomyRepository(db),
		Replies:  topicRepo.NewReplyRepository(db),
		Users:    uc,
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	cleanup := func() {
		_ = db.Exec(`DELETE FROM topic_reply WHERE id BETWEEN 953000001 AND 953000999`).Error
		_ = db.Exec(`DELETE FROM topic WHERE id BETWEEN 953000001 AND 953000999`).Error
	}
	cleanup()
	t.Cleanup(cleanup)
	run := func(q string, args ...any) {
		t.Helper()
		if err := db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	topic := func(id, user, status int, scope string, nsfw bool, section, title, body string) {
		t.Helper()
		run(`INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, ?, ?, 0, ?, 'galgame', ?, ?, ?, ?, ?, ?, '', 0, 0, 0, 0, 0, 0, 0, 0, ?, 0)`,
			id, title, body, status, at, at, at, user, nsfw, scope, map[bool]string{true: "author", false: ""}[status == 1])
		run(`INSERT INTO topic_section_relation (topic_id, topic_section_id, created, updated)
			SELECT ?, id, ?, ? FROM topic_section WHERE name = ?`, id, at, at, section)
	}
	topic(topicPublic, userAlice, 0, "public", false, "galgame-discussion", "xmcpcat 猫耳作品讨论", "大家推荐的 **猫耳** 作品")
	topic(topicHidden, userAlice, 1, "public", false, "galgame-discussion", "xmcpcat hidden", "body")
	topic(topicNSFW, userAlice, 0, "public", true, "galgame-discussion", "xmcpcat adult", "body")
	topic(topicPrivate, userAlice, 0, "users", false, "galgame-discussion", "xmcpcat private", "body")
	topic(topicBanned, userBanned, 0, "public", false, "galgame-discussion", "xmcpcat by banned", "body")
	topic(topicDaily, userBob, 0, "public", false, "daily", "xmcpcat daily", "body")
	run(`INSERT INTO topic_access_grant (topic_id, subject_type, subject_value) VALUES (?, 'user', ?)`, topicPrivate, strconv.Itoa(userBob))
	reply := func(id, user, floor, likes int, body string) {
		t.Helper()
		run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
			VALUES (?, ?, ?, ?, ?, 0, ?, ?, ?)`, id, body, floor, user, topicPublic, likes, at, at)
	}
	reply(replyPlain, userBob, 1, 0, "no likes")
	reply(replyBest, userBob, 2, 0, "the best answer")
	reply(replyLiked, userCarol, 3, 5, "a liked reply")
	reply(replyBanned, userBanned, 4, 9, "a reply by a banned author")
	run(`UPDATE topic SET best_answer_id = ? WHERE id = ?`, replyBest, topicPublic)
	return &fixture{url: srv.URL + Path}
}

func (f *fixture) session(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-agent", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: f.url, HTTPClient: &http.Client{Transport: bearer(token)}, MaxRetries: -1, DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func (f *fixture) call(t *testing.T, token, tool string, args map[string]any) (string, []Card, bool) {
	t.Helper()
	res, err := f.session(t, token).CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s: a tool failure must be a result, not a protocol error: %v", tool, err)
	}
	var text strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text.WriteString(tc.Text)
		}
	}
	var out Cards
	if res.StructuredContent != nil {
		raw, _ := json.Marshal(res.StructuredContent)
		_ = json.Unmarshal(raw, &out)
	}
	return text.String(), out.Cards, res.IsError
}

func cardIDs(cards []Card) []string {
	ids := make([]string, len(cards))
	for i, c := range cards {
		ids[i] = c.ID
	}
	slices.Sort(ids)
	return ids
}

func id(n int) string { return strconv.Itoa(n) }

func TestToolsAreListedWithSchemas(t *testing.T) {
	f := newFixture(t)
	res, err := f.session(t, "bob").ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		byName[tool.Name] = tool
	}
	if len(byName) != 2 || byName["search_topics"] == nil || byName["get_topic"] == nil {
		t.Fatalf("tools: %v", byName)
	}
	for name, tool := range byName {
		raw, _ := json.Marshal(tool.InputSchema)
		if tool.Description == "" || tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("%s needs a description and a read-only hint", name)
		}
		if name == "search_topics" && (!strings.Contains(string(raw), `"galgame-walkthrough"`) || !strings.Contains(string(raw), `"likes"`)) {
			t.Errorf("section and sort are enums: %s", raw)
		}
	}
}

func TestSearchSeesWhatTheUserSeesInSiteSearch(t *testing.T) {
	f := newFixture(t)

	text, cards, isErr := f.call(t, "bob", "search_topics", map[string]any{"query": "xmcpcat"})
	if isErr {
		t.Fatalf("search failed: %s", text)
	}
	want := []string{id(topicPublic), id(topicNSFW), id(topicDaily)}
	slices.Sort(want)
	if got := cardIDs(cards); !slices.Equal(got, want) {
		t.Fatalf("hidden, granted-only and banned-author topics stay out of search: got %v want %v\n%s", got, want, text)
	}
	for _, c := range cards {
		if c.Kind != "kungal:topic" || c.URL != "/topic/"+c.ID || c.Title == "" {
			t.Errorf("card %+v", c)
		}
		if want := map[bool]string{true: "nsfw", false: "sfw"}[c.ID == id(topicNSFW)]; c.ContentLimit != want {
			t.Errorf("card %s content_limit %q, want %q", c.ID, c.ContentLimit, want)
		}
	}
	if !strings.Contains(text, "#"+id(topicPublic)) || !strings.Contains(text, "Galgame 讨论") {
		t.Errorf("the model reads ids and section labels:\n%s", text)
	}

	for _, token := range []string{"alice", "carol", "bob-no-profile"} {
		_, cards, _ = f.call(t, token, "search_topics", map[string]any{"query": "xmcpcat"})
		if slices.Contains(cardIDs(cards), id(topicNSFW)) {
			t.Errorf("%s: a hide stance, an unattested account or a missing profile hides adult topics", token)
		}
	}

	_, cards, _ = f.call(t, "bob", "search_topics", map[string]any{"query": "xmcpcat", "section": "daily", "sort": "newest"})
	if got := cardIDs(cards); !slices.Equal(got, []string{id(topicDaily)}) {
		t.Errorf("section filter: %v", got)
	}

	text, _, isErr = f.call(t, "bob", "search_topics", map[string]any{"query": "xmcpcat", "sort": "hottest"})
	if !isErr {
		t.Errorf("an unknown sort is refused by the schema:\n%s", text)
	}
}

func TestGetTopicFollowsTheDetailPageRules(t *testing.T) {
	f := newFixture(t)

	text, cards, isErr := f.call(t, "bob", "get_topic", map[string]any{"topic_id": topicPublic})
	if isErr {
		t.Fatalf("get failed: %s", text)
	}
	if len(cards) != 1 || cards[0].ID != id(topicPublic) || cards[0].Subtitle == "" {
		t.Fatalf("one card for the topic: %+v", cards)
	}
	if facts := cards[0].Facts; len(facts) != 6 || facts[3].Label != "回复" || facts[4].Value == "" {
		t.Errorf("card facts: %+v", facts)
	}
	best, liked := strings.Index(text, "the best answer"), strings.Index(text, "a liked reply")
	if !strings.Contains(text, "大家推荐的 猫耳 作品") || best < 0 || liked < best || !strings.Contains(text, "最佳答案") {
		t.Errorf("plain body, then the best answer before the most liked reply:\n%s", text)
	}
	if strings.Contains(text, "banned author") || strings.Contains(text, "no likes") {
		t.Errorf("a banned author's reply and an unliked reply are left out:\n%s", text)
	}

	if text, _, isErr = f.call(t, "bob", "get_topic", map[string]any{"topic_id": topicPrivate}); isErr {
		t.Errorf("a granted user reads a private topic: %s", text)
	}
	for _, c := range []struct {
		token string
		topic int
		why   string
	}{
		{"carol", topicPrivate, "an ungranted user"},
		{"bob", topicHidden, "a hidden topic"},
		{"admin", topicHidden, "staff powers do not ride a forwarded token"},
		{"bob", topicBanned, "a banned author's topic"},
		{"alice", topicNSFW, "an adult topic under a hide stance"},
		{"bob", 953000999, "a missing topic"},
	} {
		text, cards, isErr := f.call(t, c.token, "get_topic", map[string]any{"topic_id": c.topic})
		if !isErr || len(cards) != 0 || text == "" {
			t.Errorf("%s must be an error result with a reason: err=%v cards=%v %q", c.why, isErr, cards, text)
		}
	}
	if text, _, isErr := f.call(t, "alice", "get_topic", map[string]any{"topic_id": topicHidden}); isErr || !strings.Contains(text, "已被隐藏") {
		t.Errorf("the author reads her own hidden topic, marked as hidden: %v %s", isErr, text)
	}
}

func TestABadTokenIsAToolError(t *testing.T) {
	f := newFixture(t)
	for _, token := range []string{"", "forged"} {
		for tool, args := range map[string]map[string]any{
			"search_topics": {"query": "xmcpcat"},
			"get_topic":     {"topic_id": topicPublic},
		} {
			text, cards, isErr := f.call(t, token, tool, args)
			if !isErr || len(cards) != 0 || !strings.Contains(text, "重新登录") {
				t.Errorf("%s with token %q: err=%v cards=%v %q", tool, token, isErr, cards, text)
			}
		}
	}
}
