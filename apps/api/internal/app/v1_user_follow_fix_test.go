package app

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func (c *fakeCommunity) seedFollow(follower, followee int64, at string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.addFollowLocked(follower, followee, at)
}

func (c *fakeCommunity) addFollowLocked(follower, followee int64, at string) {
	if c.userFollows[follower] == nil {
		c.userFollows[follower] = map[int64]fakeFollow{}
	}
	if _, ok := c.userFollows[follower][followee]; !ok {
		c.userFollows[follower][followee] = fakeFollow{at: at, notify: "all"}
	}
}

func (c *fakeCommunity) handleUserFollow(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodPost && r.URL.Path == "/follows/states" {
		c.serveFollowStates(w, r)
		return true
	}
	uid, target, kind, ok := parseUserFollowPath(r.URL.Path)
	if !ok {
		return false
	}
	switch {
	case kind == "edge" && r.Method == http.MethodPut:
		c.serveFollowPut(w, uid, target)
	case kind == "edge" && r.Method == http.MethodPatch:
		c.serveFollowPatch(w, r, uid, target)
	case kind == "edge" && r.Method == http.MethodDelete:
		c.serveFollowDelete(w, uid, target)
	case kind == "followers" && r.Method == http.MethodGet:
		c.serveFollowList(w, r, uid, true)
	case kind == "following" && r.Method == http.MethodGet:
		c.serveFollowList(w, r, uid, false)
	default:
		return false
	}
	return true
}

func parseUserFollowPath(path string) (uid, target int64, kind string, ok bool) {
	rest, found := strings.CutPrefix(path, "/users/")
	if !found {
		return 0, 0, "", false
	}
	uidStr, tail, found := strings.Cut(rest, "/")
	if !found {
		return 0, 0, "", false
	}
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil || uid <= 0 {
		return 0, 0, "", false
	}
	switch tail {
	case "followers":
		return uid, 0, "followers", true
	case "following":
		return uid, 0, "following", true
	}
	targetStr, found := strings.CutPrefix(tail, "following/")
	if !found || targetStr == "" || strings.Contains(targetStr, "/") {
		return 0, 0, "", false
	}
	target, err = strconv.ParseInt(targetStr, 10, 64)
	if err != nil || target <= 0 {
		return 0, 0, "", false
	}
	return uid, target, "edge", true
}

func (c *fakeCommunity) serveFollowPut(w http.ResponseWriter, follower, followee int64) {
	c.followPuts.Add(1)
	if follower == followee {
		writeEnvelope(w, http.StatusUnprocessableEntity, 40001, "cannot follow yourself", nil)
		return
	}
	if c.followCap.Load() {
		writeEnvelope(w, http.StatusUnprocessableEntity, 40001, "following limit reached", nil)
		return
	}
	_, existed := c.userFollows[follower][followee]
	c.addFollowLocked(follower, followee, time.Now().UTC().Format(time.RFC3339))
	notify := c.userFollows[follower][followee].notify
	writeEnvelope(w, 200, 0, "", map[string]any{
		"follower_id": follower, "followee_id": followee, "following": true, "created": !existed, "notify": notify,
	})
}

func (c *fakeCommunity) serveFollowPatch(w http.ResponseWriter, r *http.Request, follower, followee int64) {
	b, _ := io.ReadAll(r.Body)
	c.lastFollowPatch = string(b)
	edge, ok := c.userFollows[follower][followee]
	if !ok {
		writeEnvelope(w, http.StatusNotFound, 4, "not following", nil)
		return
	}
	var req struct {
		Notify string `json:"notify"`
	}
	_ = json.Unmarshal(b, &req)
	edge.notify = req.Notify
	c.userFollows[follower][followee] = edge
	writeEnvelope(w, 200, 0, "", map[string]any{
		"follower_id": follower, "followee_id": followee, "notify": edge.notify,
	})
}

func (c *fakeCommunity) serveFollowDelete(w http.ResponseWriter, follower, followee int64) {
	_, existed := c.userFollows[follower][followee]
	if existed {
		delete(c.userFollows[follower], followee)
	}
	writeEnvelope(w, 200, 0, "", map[string]any{
		"follower_id": follower, "followee_id": followee, "following": false, "deleted": existed,
	})
}

type fakeFollow struct {
	at     string
	notify string
}

type fakeFollowEdge struct {
	other int64
	at    string
}

func (c *fakeCommunity) serveFollowList(w http.ResponseWriter, r *http.Request, uid int64, followers bool) {
	c.followLists.Add(1)
	var edges []fakeFollowEdge
	if followers {
		for follower, targets := range c.userFollows {
			if edge, ok := targets[uid]; ok {
				edges = append(edges, fakeFollowEdge{other: follower, at: edge.at})
			}
		}
	} else if c.userFollows[uid] != nil {
		for followee, edge := range c.userFollows[uid] {
			edges = append(edges, fakeFollowEdge{other: followee, at: edge.at})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].at != edges[j].at {
			return edges[i].at > edges[j].at
		}
		return edges[i].other > edges[j].other
	})
	start, _ := strconv.Atoi(r.URL.Query().Get("cursor"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if start < 0 {
		start = 0
	}
	end := min(start+limit, len(edges))
	users := make([]map[string]any, 0, max(end-start, 0))
	if start < len(edges) {
		for _, e := range edges[start:end] {
			var at any
			if e.at != "" {
				at = e.at
			}
			users = append(users, map[string]any{"user_id": e.other, "followed_at": at})
		}
	}
	next := ""
	if end < len(edges) {
		next = strconv.Itoa(end)
	}
	writeEnvelope(w, 200, 0, "", map[string]any{"users": users, "next_cursor": next})
}

func (c *fakeCommunity) serveFollowStates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ViewerID int64   `json:"viewer_id"`
		UserIDs  []int64 `json:"user_ids"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	states := make([]map[string]any, 0, len(req.UserIDs))
	for _, id := range req.UserIDs {
		followers := 0
		for _, targets := range c.userFollows {
			if _, ok := targets[id]; ok {
				followers++
			}
		}
		following := 0
		if c.userFollows[id] != nil {
			following = len(c.userFollows[id])
		}
		_, viewerFollows := c.userFollows[req.ViewerID][id]
		viewerFollows = req.ViewerID > 0 && viewerFollows
		_, followsViewer := c.userFollows[id][req.ViewerID]
		followsViewer = req.ViewerID > 0 && followsViewer
		var viewerNotify any
		if viewerFollows {
			viewerNotify = c.userFollows[req.ViewerID][id].notify
		}
		states = append(states, map[string]any{
			"user_id": id, "followers_count": followers, "following_count": following,
			"viewer_follows": viewerFollows, "follows_viewer": followsViewer,
			"viewer_notify": viewerNotify,
		})
	}
	writeEnvelope(w, 200, 0, "", map[string]any{"states": states})
}

type followFix struct {
	*writeFix
	cm *fakeCommunity
}

func newFollowFix(t *testing.T) *followFix {
	t.Helper()
	cm := newFakeCommunity()
	base := newWriteFixCommunity(t, nil, cm)
	base.alice(t)
	return &followFix{writeFix: base, cm: cm}
}

func (f *followFix) followOp(t *testing.T, method, session, userID string) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, method, "/api/v1/me/following/"+userID, "/me/following/{user_id}", session, "", nil)
}

func (f *followFix) patchFollowNotify(t *testing.T, session, userID string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, http.MethodPatch, "/api/v1/me/following/"+userID, "/me/following/{user_id}", session, "", payload)
}

func (f *followFix) listFollows(t *testing.T, session, owner, relation, query string) (*http.Response, map[string]any) {
	t.Helper()
	path := "/api/v1/users/" + owner + "/" + relation
	if query != "" {
		path += "?" + query
	}
	return f.callJSON(t, http.MethodGet, path, "/users/{user_id}/"+relation, session, "", nil)
}

func (f *writeFix) callJSON(t *testing.T, method, rawURL, spec, session, key string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, method, rawURL, session, spec, key, nil, payload)
	return resp, problemMap(t, body)
}
