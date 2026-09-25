package communityclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

func newTestClient(baseURL string) *communityclient.Client {
	return communityclient.New(communityclient.Config{BaseURL: baseURL, ClientID: "cid", ClientSecret: "sec"})
}

func TestNotConfigured(t *testing.T) {
	c := communityclient.New(communityclient.Config{BaseURL: "http://x"})
	if c.Configured() {
		t.Fatal("Configured() true without creds")
	}
	if _, err := c.GetComments(context.Background(), communityclient.AnchorSiteGame, "1", "", ""); !errors.Is(err, communityclient.ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
}

func TestGetCommentsEnvelopeAndAuth(t *testing.T) {
	var gotAuth, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功",
			"data": map[string]any{
				"thread": map[string]any{"id": 7, "site": "kungal", "kind": 1, "anchor_kind": 1, "anchor_id": "42", "content_rating": 0, "status": 0, "posts_count": 2, "participants_count": 1, "highest_post_number": 2, "created_by": 1, "created_at": "2026-07-13T00:00:00Z"},
				"posts": []any{
					map[string]any{"id": 100, "thread_id": 7, "post_number": 1, "author_id": 1, "content_raw": "hi", "content_html": "<p>hi</p>", "content_rating": 0, "status": 0, "created_at": "2026-07-13T00:00:00Z"},
				},
				"next_cursor": "",
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).GetComments(context.Background(), communityclient.AnchorSiteGame, "42", "", "")
	if err != nil {
		t.Fatalf("GetComments: %v", err)
	}
	if gotAuth != "Basic Y2lkOnNlYw==" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotPath != "/comments" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotQuery, "anchor_kind=1") || !strings.Contains(gotQuery, "anchor_id=42") {
		t.Errorf("query %q missing the anchor", gotQuery)
	}
	if out.Thread == nil || out.Thread.ID != 7 || len(out.Posts) != 1 || out.Posts[0].ID != 100 {
		t.Errorf("decoded = %+v", out)
	}
}

// An anchor nobody has commented on answers with no thread at all — the page
// must render empty rather than dereference it.
func TestGetCommentsUncommentedAnchor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功", "data": map[string]any{"posts": []any{}},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).GetComments(context.Background(), communityclient.AnchorSiteGame, "42", "", "")
	if err != nil {
		t.Fatalf("GetComments: %v", err)
	}
	if out.Thread != nil || len(out.Posts) != 0 {
		t.Errorf("decoded = %+v, want no thread and no posts", out)
	}
}

func TestCommentOnAnchor(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功",
			"data": map[string]any{
				"thread": map[string]any{"id": 7, "site": "kungal", "kind": 1, "anchor_kind": 1, "anchor_id": "42", "posts_count": 1},
				"post":   map[string]any{"id": 100, "thread_id": 7, "post_number": 1, "author_id": 3, "content_raw": "hi"},
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).CommentOnAnchor(context.Background(), communityclient.CommentRequest{
		AnchorKind: communityclient.AnchorSiteGame, AnchorID: "42", AuthorID: 3, Body: "hi",
	})
	if err != nil {
		t.Fatalf("CommentOnAnchor: %v", err)
	}
	if gotPath != "/comments" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"anchor_id":"42"`) || !strings.Contains(gotBody, `"author_id":3`) {
		t.Errorf("body %q missing the anchor or author", gotBody)
	}
	if out.Thread.ID != 7 || out.Post.ID != 100 {
		t.Errorf("decoded = %+v", out)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   map[string]any
		want   error
	}{
		{"forbidden site binding", http.StatusForbidden, nil, communityclient.ErrForbidden},
		{"tl0 rate limit", http.StatusTooManyRequests, nil, communityclient.ErrRateLimited},
		{"business code", http.StatusOK, map[string]any{"code": 40001, "message": "bad"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				if tc.body != nil {
					_ = json.NewEncoder(w).Encode(tc.body)
				}
			}))
			defer srv.Close()

			_, err := newTestClient(srv.URL).GetComments(context.Background(), communityclient.AnchorSiteGame, "42", "", "")
			if tc.want != nil {
				if !errors.Is(err, tc.want) {
					t.Errorf("err = %v, want %v", err, tc.want)
				}
				return
			}
			var apiErr *communityclient.APIError
			if !errors.As(err, &apiErr) || apiErr.Code != 40001 {
				t.Errorf("err = %v, want *APIError code=40001", err)
			}
		})
	}
}

func TestToggleReactionResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/posts/100/reaction" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"added": true, "author_id": 55, "thread_id": 7, "anchor_kind": 1, "anchor_id": "42"},
		})
	}))
	defer srv.Close()

	res, err := newTestClient(srv.URL).ToggleReaction(context.Background(), 100, communityclient.ReactionToggleRequest{UserID: 9, Kind: communityclient.ReactionLike})
	if err != nil {
		t.Fatalf("ToggleReaction: %v", err)
	}
	if !res.Added || res.AuthorID != 55 || res.AnchorID != "42" {
		t.Errorf("result = %+v", res)
	}
}

func TestAuthorPosts(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{
				"posts": []any{
					map[string]any{
						"post":   map[string]any{"id": 100, "thread_id": 7, "post_number": 3, "author_id": 55, "content_raw": "hi", "status": 0, "created_at": "2026-07-16T00:00:00Z"},
						"thread": map[string]any{"thread_id": 7, "title": "", "anchor_kind": 1, "anchor_id": "42"},
					},
				},
				"next_cursor": "99",
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).AuthorPosts(context.Background(), 55, "150", 20, communityclient.AnchorSiteGame)
	if err != nil {
		t.Fatalf("AuthorPosts: %v", err)
	}
	if gotPath != "/authors/55/posts" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"after=150", "limit=20", "anchor_kind=1"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if len(out.Posts) != 1 || out.Posts[0].Post.ID != 100 || out.Posts[0].Thread.AnchorID != "42" || out.NextCursor != "99" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestAuthorStats(t *testing.T) {
	var gotQuery string
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit, gotQuery = true, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"stats": []any{
				map[string]any{"author_id": 55, "visible_posts": 9},
				map[string]any{"author_id": 56, "visible_posts": 0},
			}},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	out, err := c.AuthorStats(context.Background(), []int64{55, 56})
	if err != nil {
		t.Fatalf("AuthorStats: %v", err)
	}
	if !strings.Contains(gotQuery, "ids=55%2C56") {
		t.Errorf("query %q missing joined ids", gotQuery)
	}
	if len(out.Stats) != 2 || out.Stats[0].VisiblePosts != 9 {
		t.Errorf("decoded = %+v", out)
	}
	hit = false
	if res, err := c.AuthorStats(context.Background(), nil); err != nil || len(res.Stats) != 0 || hit {
		t.Errorf("empty AuthorStats hit=%v res=%+v err=%v", hit, res, err)
	}
}

func TestAuthorPurge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/authors/55/purge" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{
				"posts_purged": 3, "reactions_deleted": 2,
				"anchor_subscriptions_deleted": 4, "notifications_deleted": 5,
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).AuthorPurge(context.Background(), 55)
	if err != nil {
		t.Fatalf("AuthorPurge: %v", err)
	}
	if out.PostsPurged != 3 || out.ReactionsDeleted != 2 || out.AnchorSubscriptionsDeleted != 4 || out.NotificationsDeleted != 5 {
		t.Errorf("decoded = %+v", out)
	}
}

func TestRestoreAuthorPurge(t *testing.T) {
	nothingLeft := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/authors/55/purge/restore" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		if nothingLeft {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 4, "message": "no purge of this author in the last 30 days is left to restore"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{
				"posts_restored": 3, "reactions_restored": 2, "read_states_restored": 1,
				"anchor_subscriptions_restored": 4, "notifications_restored": 5,
			},
		})
	}))
	defer srv.Close()
	cli := newTestClient(srv.URL)

	out, err := cli.RestoreAuthorPurge(context.Background(), 55)
	if err != nil {
		t.Fatalf("RestoreAuthorPurge: %v", err)
	}
	if out.PostsRestored != 3 || out.ReactionsRestored != 2 || out.ReadStatesRestored != 1 ||
		out.AnchorSubscriptionsRestored != 4 || out.NotificationsRestored != 5 {
		t.Errorf("decoded = %+v", out)
	}

	nothingLeft = true
	_, err = cli.RestoreAuthorPurge(context.Background(), 55)
	var apiErr *communityclient.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound || !strings.Contains(apiErr.Msg, "left to restore") {
		t.Errorf("nothing to restore: %v", err)
	}
}

func TestResolvePosts(t *testing.T) {
	var gotPath, gotBody string
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit, gotPath = true, r.URL.Path
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"posts": []any{
				map[string]any{
					"post":   map[string]any{"id": 100, "thread_id": 7, "author_id": 55, "content_raw": "hi", "status": 0},
					"thread": map[string]any{"thread_id": 7, "anchor_kind": 1, "anchor_id": "42"},
				},
			}},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	out, err := c.ResolvePosts(context.Background(), []int64{100, 200})
	if err != nil {
		t.Fatalf("ResolvePosts: %v", err)
	}
	if gotPath != "/posts/resolve" || !strings.Contains(gotBody, `"ids":[100,200]`) {
		t.Errorf("path = %q body = %q", gotPath, gotBody)
	}
	if len(out.Posts) != 1 || out.Posts[0].Post.ID != 100 {
		t.Errorf("decoded = %+v", out)
	}
	hit = false
	if res, err := c.ResolvePosts(context.Background(), nil); err != nil || len(res.Posts) != 0 || hit {
		t.Errorf("empty ResolvePosts hit=%v res=%+v err=%v", hit, res, err)
	}
}

func TestSearchPostsQuery(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功",
			"data": map[string]any{"posts": []any{}},
		})
	}))
	defer srv.Close()

	// kind 0 is topic, not "unset": dropping it would silently widen the search
	// to every kind, which is what the community service defaults to.
	if _, err := newTestClient(srv.URL).SearchPosts(context.Background(), "汉化", communityclient.KindTopic, "", 0); err != nil {
		t.Fatalf("SearchPosts: %v", err)
	}
	if gotPath != "/search/posts" || !strings.Contains(gotQuery, "kind=0") {
		t.Errorf("path = %q query = %q", gotPath, gotQuery)
	}
}

func TestCommentOnAnchorOmitsEmptyMentions(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"thread": map[string]any{"id": 7}, "post": map[string]any{"id": 1}},
		})
	}))
	defer srv.Close()

	if _, err := newTestClient(srv.URL).CommentOnAnchor(context.Background(), communityclient.CommentRequest{
		AnchorKind: communityclient.AnchorSiteGame, AnchorID: "42", AuthorID: 3, Body: "hi",
	}); err != nil {
		t.Fatalf("CommentOnAnchor: %v", err)
	}
	if strings.Contains(gotBody, "mention_user_ids") {
		t.Errorf("empty mentions were sent: %s", gotBody)
	}
}

func TestSetAnchorNotification(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"user_id": 3, "anchor_kind": 1, "anchor_id": "7", "notification_level": 3},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).SetAnchorNotification(context.Background(), 3, communityclient.AnchorSiteGame, "7", communityclient.NotificationWatching)
	if err != nil {
		t.Fatalf("SetAnchorNotification: %v", err)
	}
	if gotPath != "/anchors/notification" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"user_id":3`) || !strings.Contains(gotBody, `"anchor_kind":1`) || !strings.Contains(gotBody, `"level":3`) {
		t.Errorf("body = %q", gotBody)
	}
	if out.NotificationLevel != 3 {
		t.Errorf("decoded = %+v", out)
	}
}

func TestAnchorStates(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"states": []any{
				map[string]any{"user_id": 3, "anchor_kind": 1, "anchor_id": "7", "notification_level": 3},
			}},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).AnchorStates(context.Background(), 3, []communityclient.AnchorRef{
		{AnchorKind: communityclient.AnchorSiteGame, AnchorID: "7"},
	})
	if err != nil {
		t.Fatalf("AnchorStates: %v", err)
	}
	if gotPath != "/anchors/states" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"anchor_id":"7"`) {
		t.Errorf("body = %q", gotBody)
	}
	if len(out.States) != 1 {
		t.Errorf("decoded = %+v", out)
	}
}

func TestListAnchorSubscriptions(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"subscriptions": []any{}, "next_cursor": "9"},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).ListAnchorSubscriptions(context.Background(), 3, -1, "abc", 40)
	if err != nil {
		t.Fatalf("ListAnchorSubscriptions: %v", err)
	}
	if gotPath != "/users/3/anchor-subscriptions" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"anchor_kind=-1", "cursor=abc", "limit=40"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if out.NextCursor != "9" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestNotificationFeed(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"notifications": []any{}, "next_after": 12},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).NotificationFeed(context.Background(), 0, 500)
	if err != nil {
		t.Fatalf("NotificationFeed: %v", err)
	}
	if gotPath != "/notifications/feed" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotQuery, "after=0") || !strings.Contains(gotQuery, "limit=500") {
		t.Errorf("query = %q", gotQuery)
	}
	if out.NextAfter != 12 {
		t.Errorf("decoded = %+v", out)
	}
}

func TestMarkNotificationsRead(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"marked": 2, "unread_count": 0},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).MarkNotificationsRead(context.Background(), 3, []int64{11, 12})
	if err != nil {
		t.Fatalf("MarkNotificationsRead: %v", err)
	}
	if gotPath != "/users/3/notifications/read" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"ids":[11,12]`) {
		t.Errorf("body = %q", gotBody)
	}
	if strings.Contains(gotBody, `"all"`) {
		t.Errorf("sent all: %s", gotBody)
	}
	if out.Marked != 2 {
		t.Errorf("decoded = %+v", out)
	}
}

func TestFollowUser(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{
				"follower_id": 3, "followee_id": 9, "following": true, "created": true,
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).FollowUser(context.Background(), 3, 9)
	if err != nil {
		t.Fatalf("FollowUser: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/users/3/following/9" {
		t.Errorf("method/path = %s %q", gotMethod, gotPath)
	}
	if out.FollowerID != 3 || out.FolloweeID != 9 || !out.Following || !out.Created {
		t.Errorf("decoded = %+v", out)
	}
}

func TestUnfollowUser(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{
				"follower_id": 3, "followee_id": 9, "following": false, "deleted": true,
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).UnfollowUser(context.Background(), 3, 9)
	if err != nil {
		t.Fatalf("UnfollowUser: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/users/3/following/9" {
		t.Errorf("method/path = %s %q", gotMethod, gotPath)
	}
	if out.Following || !out.Deleted {
		t.Errorf("decoded = %+v", out)
	}
}

func TestListFollowers(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{
				"users": []any{
					map[string]any{"user_id": 9, "followed_at": "2026-09-25T00:00:00Z"},
					map[string]any{"user_id": 8, "followed_at": nil},
				},
				"next_cursor": "abc",
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).ListFollowers(context.Background(), 3, "cur1", 40)
	if err != nil {
		t.Fatalf("ListFollowers: %v", err)
	}
	if gotPath != "/users/3/followers" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"cursor=cur1", "limit=40"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if len(out.Users) != 2 || out.Users[0].UserID != 9 || out.Users[1].FollowedAt != nil || out.NextCursor != "abc" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestListFollowing(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"users": []any{}, "next_cursor": ""},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).ListFollowing(context.Background(), 3, "", 0)
	if err != nil {
		t.Fatalf("ListFollowing: %v", err)
	}
	if gotPath != "/users/3/following" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "" {
		t.Errorf("empty cursor and limit still queried: %q", gotQuery)
	}
	if len(out.Users) != 0 {
		t.Errorf("decoded = %+v", out)
	}
}

func TestFollowStates(t *testing.T) {
	var gotBody string
	hit := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/follows/states" {
			t.Errorf("path = %q", r.URL.Path)
		}
		hit++
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"states": []any{
				map[string]any{"user_id": 9, "followers_count": 2, "following_count": 4, "viewer_follows": true, "follows_viewer": false},
			}},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	out, err := c.FollowStates(context.Background(), 3, []int64{9})
	if err != nil {
		t.Fatalf("FollowStates: %v", err)
	}
	if !strings.Contains(gotBody, `"viewer_id":3`) || !strings.Contains(gotBody, `"user_ids":[9]`) {
		t.Errorf("body = %q", gotBody)
	}
	if len(out.States) != 1 || !out.States[0].ViewerFollows || out.States[0].FollowersCount != 2 {
		t.Errorf("decoded = %+v", out)
	}

	hit = 0
	out, err = c.FollowStates(context.Background(), 0, []int64{9})
	if err != nil {
		t.Fatalf("FollowStates anonymous: %v", err)
	}
	if strings.Contains(gotBody, "viewer_id") {
		t.Errorf("anonymous viewer was sent: %s", gotBody)
	}
	if hit != 1 || len(out.States) != 1 {
		t.Errorf("hit=%d decoded=%+v", hit, out)
	}
}

func TestFollowStatesBatching(t *testing.T) {
	var sizes []int
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		var req communityclient.FollowStatesRequest
		_ = json.Unmarshal(b, &req)
		sizes = append(sizes, len(req.UserIDs))
		states := make([]any, 0, len(req.UserIDs))
		for _, id := range req.UserIDs {
			states = append(states, map[string]any{"user_id": id, "followers_count": 0, "following_count": 0})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"states": states}})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	ids := make([]int64, 150)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	out, err := c.FollowStates(context.Background(), 3, ids)
	if err != nil {
		t.Fatalf("FollowStates: %v", err)
	}
	if len(sizes) != 2 || sizes[0] != 100 || sizes[1] != 50 {
		t.Errorf("batch sizes = %v, want [100 50]", sizes)
	}
	if len(out.States) != 150 || out.States[0].UserID != 1 || out.States[149].UserID != 150 {
		t.Errorf("decoded len=%d first=%d last=%d", len(out.States), out.States[0].UserID, out.States[len(out.States)-1].UserID)
	}

	sizes = nil
	if res, err := c.FollowStates(context.Background(), 3, nil); err != nil || len(res.States) != 0 || len(sizes) != 0 {
		t.Errorf("empty FollowStates hit=%d res=%+v err=%v", len(sizes), res, err)
	}
}
