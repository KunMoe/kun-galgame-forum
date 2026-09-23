package engagement_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"kun-galgame-api/internal/community/anchor"
	"kun-galgame-api/internal/community/engagement"
	"kun-galgame-api/pkg/communityclient"
)

type fakeCommunity struct {
	paths          []string
	bodies         map[string]string
	threadStates   []map[string]any
	anchorStates   []map[string]any
	commentsThread map[string]any
	readLevel      int32
}

func serve(t *testing.T, fake *fakeCommunity) *engagement.Service {
	t.Helper()
	if fake.bodies == nil {
		fake.bodies = map[string]string{}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		fake.paths = append(fake.paths, r.Method+" "+r.URL.Path)
		fake.bodies[r.Method+" "+r.URL.Path] = string(b)
		var data any
		switch r.URL.Path {
		case "/threads/states":
			data = map[string]any{"states": fake.threadStates}
		case "/anchors/states":
			data = map[string]any{"states": fake.anchorStates}
		case "/comments":
			data = map[string]any{"posts": []any{}, "thread": fake.commentsThread}
		case "/anchors/notification":
			data = map[string]any{
				"user_id": 3, "anchor_kind": 1, "anchor_id": "7", "notification_level": 1,
			}
		default:
			level := fake.readLevel
			if level == 0 {
				level = 3
			}
			data = map[string]any{
				"thread_id": 7, "user_id": 3, "last_read_post_number": 9,
				"highest_post_number": 9, "unread_count": 0, "notification_level": level,
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "成功", "data": data})
	}))
	t.Cleanup(srv.Close)
	cli := communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	return engagement.New(cli, anchor.New(nil, nil), nil)
}

func gameRef() anchor.Ref {
	return anchor.Ref{Kind: communityclient.AnchorSiteGame, ID: "7"}
}

func TestWallReadDoesNotEnrolAStranger(t *testing.T) {
	fake := &fakeCommunity{}
	svc := serve(t, fake)

	state, appErr := svc.WallRead(context.Background(), 3, gameRef(), 7)
	if appErr != nil {
		t.Fatalf("WallRead: %v", appErr)
	}
	if slices.Contains(fake.paths, "POST /threads/7/read") {
		t.Errorf("reported a read receipt for a wall with no state row: %v", fake.paths)
	}
	if state.Following {
		t.Errorf("state = %+v, want not following", state)
	}
	if state.NotificationLevel != communityclient.NotificationNormal {
		t.Errorf("level = %d, want 1", state.NotificationLevel)
	}
}

func TestWallReadAdvancesAThreadRowHolder(t *testing.T) {
	fake := &fakeCommunity{threadStates: []map[string]any{
		{"thread_id": 7, "user_id": 3, "last_read_post_number": 4, "highest_post_number": 9,
			"unread_count": 5, "notification_level": 3},
	}}
	svc := serve(t, fake)

	state, appErr := svc.WallRead(context.Background(), 3, gameRef(), 7)
	if appErr != nil {
		t.Fatalf("WallRead: %v", appErr)
	}
	if !slices.Contains(fake.paths, "POST /threads/7/read") {
		t.Errorf("no read receipt sent for a thread-row holder: %v", fake.paths)
	}
	if slices.Contains(fake.paths, "POST /anchors/states") {
		t.Errorf("looked up the anchor after a thread row was present: %v", fake.paths)
	}
	if !state.Following {
		t.Errorf("state = %+v, want following", state)
	}
}

func TestWallReadAdvancesAnAnchorWatcherWithAThread(t *testing.T) {
	fake := &fakeCommunity{anchorStates: []map[string]any{
		{"user_id": 3, "anchor_kind": 1, "anchor_id": "7", "notification_level": 3},
	}}
	svc := serve(t, fake)

	state, appErr := svc.WallRead(context.Background(), 3, gameRef(), 7)
	if appErr != nil {
		t.Fatalf("WallRead: %v", appErr)
	}
	if !slices.Contains(fake.paths, "POST /threads/7/read") {
		t.Errorf("no read receipt sent for an anchor watcher: %v", fake.paths)
	}
	if !state.Following {
		t.Errorf("state = %+v, want following", state)
	}
}

func TestWallReadFindsTheThreadWhenNoneIsSent(t *testing.T) {
	fake := &fakeCommunity{
		commentsThread: map[string]any{"id": 7, "anchor_kind": 1, "anchor_id": "7"},
		threadStates: []map[string]any{
			{"thread_id": 7, "user_id": 3, "last_read_post_number": 4, "highest_post_number": 9,
				"unread_count": 5, "notification_level": 3},
		},
	}
	svc := serve(t, fake)

	state, appErr := svc.WallRead(context.Background(), 3, gameRef(), 0)
	if appErr != nil {
		t.Fatalf("WallRead: %v", appErr)
	}
	if !slices.Contains(fake.paths, "POST /threads/7/read") || state.ThreadID != 7 {
		t.Errorf("thread not resolved from the anchor: state %+v, paths %v", state, fake.paths)
	}
}

func TestWallFollowUnfollowWritesLevelOneNeverMuted(t *testing.T) {
	fake := &fakeCommunity{commentsThread: map[string]any{"id": 7, "anchor_kind": 1, "anchor_id": "7"}}
	svc := serve(t, fake)

	state, appErr := svc.WallFollow(context.Background(), 3, gameRef(), false)
	if appErr != nil {
		t.Fatalf("WallFollow: %v", appErr)
	}
	if !slices.Contains(fake.paths, "POST /anchors/notification") {
		t.Errorf("did not write the anchor level: %v", fake.paths)
	}
	if !slices.Contains(fake.paths, "POST /threads/7/notification") {
		t.Errorf("did not write the thread level: %v", fake.paths)
	}
	for path, body := range fake.bodies {
		if strings.Contains(path, "/notification") && strings.Contains(body, `"level":0`) {
			t.Errorf("%s wrote muted: %s", path, body)
		}
		if strings.Contains(path, "/notification") && !strings.Contains(body, `"level":1`) {
			t.Errorf("%s did not write level 1: %s", path, body)
		}
	}
	if state.Following || state.NotificationLevel != 1 {
		t.Errorf("state = %+v, want not following at level 1", state)
	}
}

func TestWallFollowNoThreadWritesOnlyAnchor(t *testing.T) {
	fake := &fakeCommunity{}
	svc := serve(t, fake)

	state, appErr := svc.WallFollow(context.Background(), 3, gameRef(), true)
	if appErr != nil {
		t.Fatalf("WallFollow: %v", appErr)
	}
	if !slices.Contains(fake.paths, "POST /anchors/notification") {
		t.Errorf("did not write the anchor level: %v", fake.paths)
	}
	for _, p := range fake.paths {
		if strings.Contains(p, "/threads/") && strings.Contains(p, "/notification") {
			t.Errorf("wrote a thread level on a wall with no thread: %v", fake.paths)
		}
	}
	if !state.Following || state.ThreadID != 0 {
		t.Errorf("state = %+v, want following with no thread", state)
	}
}

func TestWallReadRejectsUnknownAnchor(t *testing.T) {
	fake := &fakeCommunity{}
	svc := serve(t, fake)
	if _, appErr := svc.WallRead(context.Background(), 3, anchor.Ref{Kind: 1, ID: "abc"}, 0); appErr == nil {
		t.Fatal("unrecognised ref was accepted")
	}
}
