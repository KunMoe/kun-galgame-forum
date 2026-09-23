package app

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
)

func contains(ids []string, id int) bool { return slices.Contains(ids, fmt.Sprint(id)) }

func TestV1SearchTopicsVisibility(t *testing.T) {
	f := newSearchFix(t)
	_, body := f.get(t, srTopicPath, "/search/topics?q=xsrchalpha&limit=100", "")
	ids := idsOf(body)
	if !contains(ids, srTopicTitle) || !contains(ids, srTopicBody) {
		t.Fatalf("visible topics missing: %v", ids)
	}
	for _, id := range []int{srTopicHidden, srTopicNSFW, srTopicLogin, srTopicBanned} {
		if contains(ids, id) {
			t.Errorf("topic %d must not match an anonymous default search: %v", id, ids)
		}
	}
	if ids[0] != fmt.Sprint(srTopicTitle) {
		t.Errorf("the title hit ranks first: %v", ids)
	}
	if asInt(body["total"]) != 3 || body["total_relation"] != "eq" {
		t.Errorf("total %v %v: three visible matches, the banned author's counted", body["total"], body["total_relation"])
	}
	first := itemsOf(body)[0]
	if first["object"] != "topic" || first["title"] != "xsrchalpha in the title" || first["state"] != "published" {
		t.Errorf("summary %+v", first)
	}

	_, body = f.get(t, srTopicPath, "/search/topics?q=xsrchalpha&limit=100&include_nsfw=true", "")
	if ids := idsOf(body); !contains(ids, srTopicNSFW) || asInt(body["total"]) != 4 {
		t.Errorf("include_nsfw=true: %v total %v", ids, body["total"])
	}

	_, body = f.get(t, srTopicPath, "/search/topics?q=xsrchalpha&limit=100", "sess-sr-alice")
	if ids := idsOf(body); !contains(ids, srTopicLogin) || contains(ids, srTopicHidden) {
		t.Errorf("signed in: %v", ids)
	}
}

func TestV1SearchTopicsWalk(t *testing.T) {
	f := newSearchFix(t)
	var want []string
	for id := srTopicTieMax; id >= srTopicTieMin; id-- {
		want = append(want, fmt.Sprint(id))
	}
	for _, limit := range []int{2, 3} {
		var got []string
		for page := 1; page <= 10; page++ {
			resp, body := f.get(t, srTopicPath, fmt.Sprintf("/search/topics?q=xsrchtie&page=%d&limit=%d", page, limit), "")
			if resp.StatusCode != http.StatusOK || asInt(body["total"]) != len(want) {
				t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
			}
			ids := idsOf(body)
			if len(ids) == 0 {
				break
			}
			got = append(got, ids...)
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
}

func TestV1SearchReplies(t *testing.T) {
	f := newSearchFix(t)
	_, body := f.get(t, srRepliesPath, "/search/replies?q=xsrchreply&limit=100", "")
	ids := idsOf(body)
	if !contains(ids, srReplyMin) || !contains(ids, srReplyMin+4) {
		t.Fatalf("visible replies missing: %v", ids)
	}
	for _, id := range []int{srReplyMin + 1, srReplyMin + 2, srReplyMin + 3} {
		if contains(ids, id) {
			t.Errorf("reply %d must not match: %v", id, ids)
		}
	}
	if asInt(body["total"]) != 3 {
		t.Errorf("total %v: two visible replies and the banned author's", body["total"])
	}
	for _, it := range itemsOf(body) {
		ex := fmt.Sprint(it["excerpt"])
		if strings.Contains(ex, "**") || strings.Contains(ex, "/image/") || strings.Contains(ex, "![") {
			t.Errorf("excerpt still carries markdown: %q", ex)
		}
		if it["id"] == fmt.Sprint(srReplyMin+4) && (!strings.HasPrefix(ex, "…") || !strings.Contains(ex, "xsrchreply")) {
			t.Errorf("a deep hit must be windowed around the keyword: %q", ex)
		}
		if it["object"] != "reply" || it["topic_title"] != "xsrchalpha in the title" {
			t.Errorf("hit %+v", it)
		}
	}

	_, body = f.get(t, srRepliesPath, "/search/replies?q=xsrchreply&limit=100&include_nsfw=true", "")
	if ids := idsOf(body); !contains(ids, srReplyMin+2) {
		t.Errorf("include_nsfw=true must reach replies under nsfw topics: %v", ids)
	}
}

func TestV1SearchRepliesWalk(t *testing.T) {
	f := newSearchFix(t)
	var want []string
	for id := srReplyTieMax; id >= srReplyTieMin; id-- {
		want = append(want, fmt.Sprint(id))
	}
	var got []string
	for page := 1; page <= 10; page++ {
		_, body := f.get(t, srRepliesPath, fmt.Sprintf("/search/replies?q=zzreplytie&page=%d&limit=3", page), "")
		ids := idsOf(body)
		if len(ids) == 0 {
			break
		}
		got = append(got, ids...)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("walked %v, want %v", got, want)
	}
}

func TestV1SearchComments(t *testing.T) {
	f := newSearchFix(t)
	_, body := f.get(t, srCommentsPath, "/search/comments?q=xsrchcomm&limit=100", "")
	if ids := idsOf(body); fmt.Sprint(ids) != fmt.Sprint([]string{fmt.Sprint(srCommentMin)}) {
		t.Errorf("anonymous: %v", ids)
	}
	_, body = f.get(t, srCommentsPath, "/search/comments?q=xsrchcomm&limit=100", "sess-sr-alice")
	if ids := idsOf(body); !contains(ids, srCommentMin+2) || contains(ids, srCommentMin+1) {
		t.Errorf("signed in: %v", ids)
	}
}

func TestV1SearchRejects(t *testing.T) {
	f := newSearchFix(t)
	for _, path := range []string{srTopicPath, srRepliesPath, srCommentsPath, srUsersPath, srWorksPath} {
		resp, body := f.get(t, path, path+"?q="+url.QueryEscape("   "), "")
		wantProblemCode(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
		wantErrorAt(t, body, "q", "TOO_SHORT")
	}
	resp, body := f.get(t, srTopicPath, "/search/topics?q=x&page=101&limit=100", "")
	wantProblemCode(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	wantErrorAt(t, body, "page", "OUT_OF_RANGE")
	resp, body = f.get(t, srTopicPath, "/search/topics", "")
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("missing q: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.get(t, srTopicPath, "/search/topics?q=x&limit=101", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("limit 101: %d %+v", resp.StatusCode, body)
	}
}

func TestV1SearchPostsUpstreamDown(t *testing.T) {
	f := newSearchFix(t)
	f.oauth.fail.Store(true)
	for _, path := range []string{srTopicPath, srRepliesPath, srCommentsPath} {
		resp, body := f.get(t, path, path+"?q=xsrch", "")
		wantProblemCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	}
}

func searchUser(id int, name string, status int, roles, siteRoles []string, created string) map[string]any {
	return map[string]any{"id": id, "name": name, "status": status, "roles": roles, "site_roles": siteRoles, "bio": "hi", "created_at": created}
}

func TestV1SearchUserLane(t *testing.T) {
	f := newSearchFix(t)
	f.oauth.search = []map[string]any{
		searchUser(srUserAlice, "alice", 0, []string{"user", "admin"}, nil, "2025-01-02T03:04:05Z"),
		searchUser(srUserBanned, "banned", 1, []string{"user"}, nil, ""),
		searchUser(srUserBob, "bob", 0, []string{"user"}, []string{"moderator"}, ""),
	}
	_, body := f.get(t, srUsersPath, "/search/users?q=a", "")
	items := itemsOf(body)
	if len(items) != 2 || asInt(body["total"]) != 2 || body["total_relation"] != "eq" {
		t.Fatalf("banned users are left out and not counted: %+v", body)
	}
	alice, bob := items[0], items[1]
	if alice["id"] != fmt.Sprint(srUserAlice) || fmt.Sprint(alice["roles"]) != "[admin]" || alice["registered_at"] != "2025-01-02T03:04:05Z" ||
		asInt(alice["topic_count"]) != 2 {
		t.Errorf("alice %+v", alice)
	}
	if fmt.Sprint(bob["roles"]) != "[moderator]" || bob["registered_at"] != nil {
		t.Errorf("bob %+v", bob)
	}

	_, body = f.get(t, srUsersPath, "/search/users?q=a", "sess-sr-alice")
	if asInt(itemsOf(body)[0]["topic_count"]) != 3 {
		t.Errorf("a signed-in caller's count includes login-only topics: %+v", itemsOf(body)[0])
	}

	f.oauth.search = nil
	for i := range 50 {
		f.oauth.search = append(f.oauth.search, searchUser(953000000+i, fmt.Sprint("u", i), 0, nil, nil, ""))
	}
	_, body = f.get(t, srUsersPath, "/search/users?q=u&page=3&limit=24", "")
	if asInt(body["total"]) != 50 || body["total_relation"] != "gte" || len(itemsOf(body)) != 2 {
		t.Errorf("capped at 50: total %v %v items %d", body["total"], body["total_relation"], len(itemsOf(body)))
	}

	f.oauth.fail.Store(true)
	resp, body := f.get(t, srUsersPath, "/search/users?q=zz", "")
	wantProblemCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

const srWorksBody = `{"total":12345,"items":[
  {"id":61311,"display_name":"紅殻のパンドラ","latin":"Koukaku no Pandora",
   "localized":{"zh-Hans":{"value":"红壳的潘多拉","machine":false}},
   "claim":{"site":"kungal","state":"live","content_limit":"nsfw"},
   "cover_slots":{"portrait":{"url":"https://img.example/aa/aa/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.webp","width":600,"height":850}}},
  {"id":61312,"display_name":"withdrawn","claim":{"site":"kungal","state":"hidden"}}
]}`

func TestV1SearchWorks(t *testing.T) {
	f := newSearchFix(t)
	f.catalog.body = srWorksBody
	resp, body := f.get(t, srWorksPath, "/search/works?q="+url.QueryEscape("恋爱 汉化")+"&limit=24&company_id=7&tag_ids=3,4&released_from=2020&released_to=2021-06&sort=released_asc", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	items := itemsOf(body)
	if len(items) != 1 || items[0]["object"] != "work" || items[0]["id"] != "61311" || items[0]["display_name"] != "紅殻のパンドラ" || items[0]["is_nsfw"] != true {
		t.Errorf("a hidden claim is left out; the ref is the shared WorkRef: %+v", items)
	}
	if asInt(body["total"]) != 10000 || body["total_relation"] != "gte" {
		t.Errorf("the index's count is clamped: %v %v", body["total"], body["total_relation"])
	}
	q := f.catalog.query
	for key, want := range map[string]string{
		"q": "恋爱 汉化", "sort": "released_asc", "company_id": "7", "tag_id": "3,4",
		"released_after": "2020-01-01", "released_before": "2021-06-30", "limit": "24",
	} {
		if got := q.Get(key); got != want {
			t.Errorf("catalog %s = %q, want %q", key, got, want)
		}
	}
	if q.Get("claim_state") != "" {
		t.Errorf("claim_state = %q: catalog is the existence layer", q.Get("claim_state"))
	}
	if !strings.Contains(q.Get("content_limit")+q.Get("nsfw"), "sfw") && q.Get("nsfw") != "false" {
		t.Errorf("the default is the SFW gate: %v", q)
	}
	sfwQuery := q.Encode()

	f.get(t, srWorksPath, "/search/works?q=x&include_nsfw=true", "")
	if f.catalog.query.Encode() == sfwQuery || strings.Contains(f.catalog.query.Get("content_limit"), "sfw") {
		t.Errorf("include_nsfw=true lifts the gate: %v", f.catalog.query)
	}

	resp, body = f.get(t, srWorksPath, "/search/works?q=x&released_from=2022&released_to=2021", "")
	wantProblemCode(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	wantErrorAt(t, body, "released_from", "INCONSISTENT_WITH")
	resp, body = f.get(t, srWorksPath, "/search/works?q=x&sort=rating_desc", "")
	wantProblemCode(t, resp, body, http.StatusBadRequest, "UNKNOWN_SORT")

	f.catalog.fail.Store(true)
	resp, body = f.get(t, srWorksPath, "/search/works?q=x", "")
	wantProblemCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func wallPost(id int, anchorKind int, anchorID string, author int, content string) string {
	return fmt.Sprintf(`{"post":{"id":%d,"thread_id":1,"post_number":1,"author_id":%d,"content_raw":%q,"status":0,"created_at":"2026-08-01T12:00:00Z"},
		"thread":{"thread_id":1,"title":"","anchor_kind":%d,"anchor_id":%q}}`, id, author, content, anchorKind, anchorID)
}

func TestV1SearchWallComments(t *testing.T) {
	f := newSearchFix(t)
	f.catalog.body = srWorksBody
	f.community.pages = map[string]string{
		"": `{"posts":[` + strings.Join([]string{
			wallPost(801, 1, fmt.Sprint(srWallWork), srUserBob, "**汉化** 真好"),
			wallPost(802, 2, "rating:55", srUserAlice, "汉化 评分"),
			wallPost(803, 2, "foreign:9", srUserAlice, "汉化 elsewhere"),
			wallPost(804, 1, "12", srUserBanned, "汉化 banned"),
		}, ",") + `],"next_cursor":"up-2"}`,
		"up-2": `{"posts":[` + wallPost(805, 2, "toolset:8", srUserBob, "汉化 工具") + `],"next_cursor":""}`,
	}
	resp, body := f.get(t, srWallPath, "/search/wall-comments?q="+url.QueryEscape("汉化"), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	items := itemsOf(body)
	if len(items) != 2 {
		t.Fatalf("foreign walls and banned authors are dropped: %+v", items)
	}
	game, rating := items[0], items[1]
	work, _ := game["work"].(map[string]any)
	if game["subject_type"] != "galgame" || game["subject_id"] != fmt.Sprint(srWallWork) || game["subject_path"] != fmt.Sprintf("/galgame/%d", srWallWork) ||
		work["id"] != fmt.Sprint(srWallWork) || strings.Contains(fmt.Sprint(game["excerpt"]), "**") {
		t.Errorf("game wall %+v", game)
	}
	if rating["subject_type"] != "galgame_rating" || rating["subject_id"] != "55" || rating["work"] != nil {
		t.Errorf("rating wall %+v", rating)
	}
	next, _ := body["next_cursor"].(string)
	if !strings.HasPrefix(next, "cur_") {
		t.Fatalf("next_cursor %v", body["next_cursor"])
	}
	_, body = f.get(t, srWallPath, "/search/wall-comments?q="+url.QueryEscape("汉化")+"&cursor="+next, "")
	if ids := idsOf(body); fmt.Sprint(ids) != "[805]" || body["next_cursor"] != nil {
		t.Errorf("second page %v %v", ids, body["next_cursor"])
	}
	if f.community.query.Get("cursor") != "up-2" {
		t.Errorf("the upstream cursor rides inside ours: %v", f.community.query)
	}

	resp, body = f.get(t, srWallPath, "/search/wall-comments?q="+url.QueryEscape("其他")+"&cursor="+next, "")
	wantProblemCode(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")
	resp, body = f.get(t, srWallPath, "/search/wall-comments?q="+url.QueryEscape(" 汉 "), "")
	wantProblemCode(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	wantErrorAt(t, body, "q", "TOO_SHORT")

	f.community.fail.Store(true)
	resp, body = f.get(t, srWallPath, "/search/wall-comments?q=abc", "")
	wantProblemCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1SearchUserLaneQueryCap(t *testing.T) {
	f := newSearchFix(t)
	f.oauth.search = []map[string]any{searchUser(srUserAlice, "alice", 0, []string{"user"}, nil, "")}

	resp, body := f.get(t, srUsersPath, "/search/users?q="+url.QueryEscape(strings.Repeat("名", 51)), "")
	wantProblemCode(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	if n := f.oauth.searched.Load(); n != 0 {
		t.Errorf("a 51-character q reached the account service %d times; it refuses anything over 50", n)
	}
	resp, body = f.get(t, srUsersPath, "/search/users?q="+url.QueryEscape(strings.Repeat("名", 50)), "")
	if resp.StatusCode != http.StatusOK || len(itemsOf(body)) != 1 {
		t.Errorf("50 characters is allowed: %d %+v", resp.StatusCode, body)
	}

	f.oauth.refuse.Store(true)
	resp, body = f.get(t, srUsersPath, "/search/users?q=alice", "")
	wantProblemCode(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
}
