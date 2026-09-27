package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"kun-galgame-api/pkg/problem"
)

var (
	topicKeys = []string{
		"access_scope", "author", "author_moemoepoint", "best_answer", "bumped_at", "category",
		"comment_count", "content", "cover_images", "created_at", "dislike_count", "edited_at",
		"favorite_count", "hidden_by", "id", "is_nsfw", "like_count", "mini_apps", "object",
		"pinned_reply", "reactions", "reply_count", "sections", "state", "title", "upvote_count",
		"upvoted_at", "view_count", "viewer",
	}
	replyKeys = []string{
		"author", "author_moemoepoint", "comments", "content", "created_at", "dislike_count",
		"edited_at", "floor", "id", "is_best_answer", "is_pinned", "like_count", "object",
		"reactions", "topic_id", "viewer",
	}
	commentKeys = []string{
		"author", "content", "created_at", "edited_at", "id", "in_reply_to_user", "like_count",
		"object", "parent_comment_id", "reply_floor", "reply_id", "viewer",
	}
	reactionKeys = []string{"count", "reaction", "reactors", "viewer"}
)

func TestV1GetTopicPublicFields(t *testing.T) {
	f := newDetailFix(t)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicPublic), "", "/topics/{topic_id}", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	var topic map[string]json.RawMessage
	if err := json.Unmarshal(body, &topic); err != nil {
		t.Fatal(err)
	}
	if keys := objectKeys(t, body); !equalStrings(keys, topicKeys) {
		t.Errorf("Topic keys %v, want %v", keys, topicKeys)
	}
	cover := `[{"url":"https://image.test.example/cc/cc/` + d1CoverHash + `.webp","hash":"` + d1CoverHash +
		`","width":640,"height":360,"thumbhash":"AbC+","sexual":"safe"}]`
	want := map[string]string{
		"object": `"topic"`, "id": `"920000201"`, "title": `"d1-public"`, "state": `"published"`,
		"hidden_by": "null", "access_scope": `"public"`, "category": `"galgame"`,
		"sections": `["g-news"]`, "cover_images": cover, "is_nsfw": `false`,
		"author":             `{"object":"user","id":"920000001","name":"alice","avatar":null,"avatar_frame":null}`,
		"author_moemoepoint": `-16`,
		"view_count":         `10`, "like_count": `4`, "dislike_count": `0`,
		"favorite_count": `3`, "upvote_count": `4`, "reply_count": `5`, "comment_count": `6`,
		"mini_apps":  `["poll"]`,
		"created_at": `"2026-02-01T12:00:00Z"`, "edited_at": `"2026-02-01T13:00:00Z"`,
		"bumped_at": `"2026-02-01T15:00:00Z"`, "upvoted_at": `"2026-02-01T12:00:00Z"`,
		"viewer": `null`,
	}
	for field, w := range want {
		if got := string(topic[field]); got != w {
			t.Errorf("%s = %s, want %s", field, got, w)
		}
	}
	var content struct {
		Object   string `json:"object"`
		Children []struct {
			Object string `json:"object"`
		} `json:"children"`
	}
	if err := json.Unmarshal(topic["content"], &content); err != nil {
		t.Fatal(err)
	}
	if content.Object != "document" || len(content.Children) == 0 || content.Children[0].Object != "paragraph" {
		t.Errorf("content first node = %+v", content)
	}

	var reactions []map[string]json.RawMessage
	if err := json.Unmarshal(topic["reactions"], &reactions); err != nil {
		t.Fatal(err)
	}
	if len(reactions) != 2 {
		t.Fatalf("reactions %d", len(reactions))
	}
	for i, r := range reactions {
		if keys := objectKeys(t, mustRaw(t, r)); !equalStrings(keys, reactionKeys) {
			t.Errorf("reaction %d keys %v", i, keys)
		}
	}
	if string(reactions[0]["reaction"]) != `"like"` || string(reactions[1]["reaction"]) != `"heart"` {
		t.Errorf("reaction order %s %s", reactions[0]["reaction"], reactions[1]["reaction"])
	}
	if string(reactions[0]["count"]) != `4` || string(reactions[1]["count"]) != `1` {
		t.Errorf("counts %s %s", reactions[0]["count"], reactions[1]["count"])
	}
	var likeReactors []map[string]any
	if err := json.Unmarshal(reactions[0]["reactors"], &likeReactors); err != nil {
		t.Fatal(err)
	}
	if len(likeReactors) != 3 {
		t.Fatalf("like reactors %d, want 3 (banned skipped)", len(likeReactors))
	}
	if likeReactors[0]["id"] != "920000001" || likeReactors[1]["id"] != "920000003" || likeReactors[2]["id"] != "920000005" {
		t.Errorf("like reactors %+v", likeReactors)
	}
	if string(reactions[0]["viewer"]) != `null` {
		t.Errorf("anon reaction viewer %s", reactions[0]["viewer"])
	}

	for _, field := range []string{"pinned_reply", "best_answer"} {
		if keys := objectKeys(t, topic[field]); !equalStrings(keys, replyKeys) {
			t.Errorf("%s keys %v, want %v", field, keys, replyKeys)
		}
	}
	var pinned, best map[string]json.RawMessage
	_ = json.Unmarshal(topic["pinned_reply"], &pinned)
	_ = json.Unmarshal(topic["best_answer"], &best)
	if string(pinned["id"]) != `"920000301"` || string(pinned["is_pinned"]) != `true` || string(pinned["is_best_answer"]) != `false` {
		t.Errorf("pinned %+v", pinned)
	}
	if string(best["id"]) != `"920000302"` || string(best["is_best_answer"]) != `true` || string(best["is_pinned"]) != `false` {
		t.Errorf("best %+v", best)
	}
	if string(pinned["author_moemoepoint"]) != `0` {
		t.Errorf("bob moe %s", pinned["author_moemoepoint"])
	}
	assertPinnedComments(t, pinned["comments"])
}

func TestV1GetTopicSignedInViewer(t *testing.T) {
	f := newDetailFix(t)
	f.putSession(t, "sess-alice", d1UserAlice)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicPublic), "sess-alice", "/topics/{topic_id}", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	var topic struct {
		Viewer *struct {
			HasLiked     bool `json:"has_liked"`
			HasDisliked  bool `json:"has_disliked"`
			HasFavorited bool `json:"has_favorited"`
			HasUpvoted   bool `json:"has_upvoted"`
		} `json:"viewer"`
		Reactions []struct {
			Reaction string `json:"reaction"`
			Viewer   *struct {
				HasReacted bool `json:"has_reacted"`
			} `json:"viewer"`
		} `json:"reactions"`
	}
	if err := json.Unmarshal(body, &topic); err != nil {
		t.Fatal(err)
	}
	if topic.Viewer == nil || !topic.Viewer.HasLiked || topic.Viewer.HasDisliked || !topic.Viewer.HasFavorited || !topic.Viewer.HasUpvoted {
		t.Errorf("viewer %+v", topic.Viewer)
	}
	if len(topic.Reactions) < 1 || topic.Reactions[0].Viewer == nil || !topic.Reactions[0].Viewer.HasReacted {
		t.Errorf("like viewer %+v", topic.Reactions)
	}
}

func TestV1GetTopicNotFoundCases(t *testing.T) {
	f := newDetailFix(t)
	type hit struct {
		name, path, session string
		hdr                 http.Header
		status              int
	}
	staffHdr := http.Header{}
	staffHdr.Set("Authorization", "Bearer staff-token")
	f.putSession(t, "sess-alice", d1UserAlice)
	f.putSession(t, "sess-bob", d1UserBob)
	f.putSession(t, "sess-staff", d1UserStaff, "user", "moderator")
	f.putSession(t, "sess-creator", d1UserCarol, "user", "creator")
	f.putSession(t, "sess-carol", d1UserCarol)

	var cores []string
	notFoundHits := []hit{
		{"missing", "/api/v1/topics/920000299", "", nil, 404},
		{"hidden stranger", "/api/v1/topics/920000220", "", nil, 404},
		{"login anon", "/api/v1/topics/920000221", "", nil, 404},
		{"banned author", "/api/v1/topics/920000224", "", nil, 404},
	}
	for _, h := range notFoundHits {
		resp, body := f.do(t, http.MethodGet, h.path, h.session, "/topics/{topic_id}", h.hdr)
		if resp.StatusCode != 404 {
			t.Errorf("%s: %d %s", h.name, resp.StatusCode, body)
			continue
		}
		code, _, _, _, _ := decodeProblemBody(t, body)
		if code != problem.CodeNotFound {
			t.Errorf("%s code %s", h.name, code)
		}
		cores = append(cores, problemCore(t, body))
	}
	for i := 1; i < len(cores); i++ {
		if cores[i] != cores[0] {
			t.Errorf("404 body %d != 0:\n%s\n%s", i, cores[i], cores[0])
		}
	}

	okHits := []hit{
		{"hidden author", "/api/v1/topics/920000220", "sess-alice", nil, 200},
		{"hidden cookie staff", "/api/v1/topics/920000220", "sess-staff", nil, 200},
		{"login signed in", "/api/v1/topics/920000221", "sess-bob", nil, 200},
		{"role granted", "/api/v1/topics/920000222", "sess-creator", nil, 200},
		{"users granted", "/api/v1/topics/920000223", "sess-bob", nil, 200},
		{"gone author", "/api/v1/topics/920000225", "", nil, 200},
	}
	for _, h := range okHits {
		resp, body := f.do(t, http.MethodGet, h.path, h.session, "/topics/{topic_id}", h.hdr)
		if resp.StatusCode != 200 {
			t.Errorf("%s: %d %s", h.name, resp.StatusCode, body)
		}
	}
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/920000225", "", "/topics/{topic_id}", nil)
	var gone struct {
		Author struct {
			Name *string `json:"name"`
			ID   string  `json:"id"`
		} `json:"author"`
	}
	_ = json.Unmarshal(body, &gone)
	if resp.StatusCode != 200 || gone.Author.Name != nil || gone.Author.ID != "920000004" {
		t.Errorf("gone author %d %+v %s", resp.StatusCode, gone.Author, body)
	}

	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/920000220", "", "/topics/{topic_id}", staffHdr)
	if resp.StatusCode != 404 {
		t.Errorf("bearer staff: %d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/920000222", "sess-bob", "/topics/{topic_id}", nil)
	if resp.StatusCode != 404 {
		t.Errorf("role without grant: %d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/920000223", "sess-carol", "/topics/{topic_id}", nil)
	if resp.StatusCode != 404 {
		t.Errorf("users without grant: %d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/920000222", "sess-carol", "/topics/{topic_id}", nil)
	if resp.StatusCode != 404 {
		t.Errorf("role scope, signed in without the role: %d %s", resp.StatusCode, body)
	}

	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/9999999999999999999", "", "/topics/{topic_id}", nil)
	if resp.StatusCode != 404 {
		t.Errorf("overflow id: %d %s", resp.StatusCode, body)
	}
}

func TestV1GetTopicUserServiceBound(t *testing.T) {
	f := newDetailFix(t)
	f.UserClient.Invalidate(d1UserAlice, d1UserBanned, d1UserBob, d1UserCarol, d1UserStaff, d1UserGone)
	f.batchCalls.Store(0)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicPublic), "", "/topics/{topic_id}", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	if n := f.batchCalls.Load(); n > 2 {
		t.Errorf("getTopic user-service calls %d, want <= 2", n)
	}
}

func assertPinnedComments(t *testing.T, raw json.RawMessage) {
	t.Helper()
	var comments []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &comments); err != nil {
		t.Fatal(err)
	}
	if len(comments) != 4 {
		t.Fatalf("comments %d, want 4 (banned commenter omitted)", len(comments))
	}
	for i, c := range comments {
		if keys := objectKeys(t, mustRaw(t, c)); !equalStrings(keys, commentKeys) {
			t.Errorf("comment %d keys %v", i, keys)
		}
	}
	if docPlainText(t, comments[0]["content"]) != "parent-text" || string(comments[0]["parent_comment_id"]) != `null` {
		t.Errorf("parent %s", comments[0])
	}
	if string(comments[0]["like_count"]) != `2` {
		t.Errorf("parent likes %s", comments[0]["like_count"])
	}
	if docPlainText(t, comments[1]["content"]) != "orphan-text" || string(comments[1]["parent_comment_id"]) != `"920000403"` {
		t.Errorf("orphan keeps parent id: %s", comments[1])
	}
	if string(comments[1]["in_reply_to_user"]) != `{"object":"user","id":"920000002","name":"banned","avatar":null,"avatar_frame":null}` {
		t.Errorf("in_reply_to banned %s", comments[1]["in_reply_to_user"])
	}
	if docPlainText(t, comments[2]["content"]) != "child-text" || string(comments[2]["parent_comment_id"]) != `"920000401"` {
		t.Errorf("child %s", comments[2])
	}
	if string(comments[3]["in_reply_to_user"]) != `{"object":"user","id":"920000004","name":null,"avatar":null,"avatar_frame":null}` {
		t.Errorf("gone target %s", comments[3]["in_reply_to_user"])
	}
	if string(comments[0]["viewer"]) != `null` {
		t.Errorf("anon comment viewer")
	}
}

func mustRaw(t *testing.T, m map[string]json.RawMessage) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
