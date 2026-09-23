package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	"kun-galgame-api/internal/moemoepoint"
	userRepo "kun-galgame-api/internal/user/repository"
)

const (
	meAvatarHash = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	mePath       = "/api/v1/me"
)

type meFix struct {
	*writeFix
	nSearch      atomic.Int32
	moeBalance   atomic.Int32
	awardFail    atomic.Bool
	moeFail      atomic.Bool
	lastAwardKey atomic.Value
	patchCode    atomic.Int32
	prefCode     atomic.Int32
	prefHTTP     atomic.Int32
	nsfwFail     atomic.Bool
	nsfwNoScope  atomic.Bool
	creatorPost  atomic.Int32
	creatorState atomic.Value
	logFail      atomic.Bool
	searchFail   atomic.Bool
	searchRefuse atomic.Bool
	avatarFail   atomic.Bool
	avatarCT     atomic.Value
	avatarName   atomic.Value
}

func newMeFix(t *testing.T) *meFix {
	t.Helper()
	base := newWriteFix(t, nil)
	f := &meFix{writeFix: base}
	f.moeBalance.Store(30)
	f.prefHTTP.Store(200)
	f.creatorState.Store("pending")
	f.installUpstream()
	moemoepoint.SetDefault(moemoepoint.NewAwarder(f.UserClient, f.db))
	t.Cleanup(func() { moemoepoint.SetDefault(moemoepoint.NewAwarder(nil, nil)) })
	f.alice(t)
	return f
}

func (f *meFix) installUpstream() {
	f.mux.HandleFunc("GET /users/search", func(w http.ResponseWriter, r *http.Request) {
		if f.searchFail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		f.nSearch.Add(1)
		if f.searchRefuse.Load() || utf8.RuneCountInString(strings.TrimSpace(r.URL.Query().Get("q"))) > 50 {
			writeHouse(w, http.StatusBadRequest, 9, nil)
			return
		}
		writeHouse(w, 200, 0, map[string]any{
			"users": []map[string]any{
				{"id": w3UserAlice, "name": "alice", "status": 0, "roles": []string{"user"}},
				{"id": w3UserBanned, "name": "banned", "status": 1, "roles": []string{"user"}},
			},
		})
	})
	f.mux.HandleFunc("GET /users/{id}/moemoepoint", func(w http.ResponseWriter, _ *http.Request) {
		if f.moeFail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeHouse(w, 200, 0, map[string]any{"balance": int(f.moeBalance.Load())})
	})
	f.mux.HandleFunc("POST /users/{id}/moemoepoint", func(w http.ResponseWriter, r *http.Request) {
		if f.awardFail.Load() {
			writeHouse(w, 500, 10, nil)
			return
		}
		var body struct {
			Delta          int    `json:"delta"`
			IdempotencyKey string `json:"idempotency_key"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.lastAwardKey.Store(body.IdempotencyKey)
		bal := int(f.moeBalance.Add(int32(body.Delta)))
		writeHouse(w, 200, 0, map[string]any{"user_id": w3UserAlice, "balance": bal, "applied": true})
	})
	f.mux.HandleFunc("GET /users/{id}/moemoepoint/log", func(w http.ResponseWriter, r *http.Request) {
		if f.logFail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 20
		}
		before, _ := strconv.Atoi(r.URL.Query().Get("before_id"))
		reason := r.URL.Query().Get("reason")
		items := fakeMoeLog()
		filtered := make([]map[string]any, 0, len(items))
		for _, it := range items {
			if reason != "" && it["reason"] != reason {
				continue
			}
			id := int(it["id"].(int))
			if before > 0 && id >= before {
				continue
			}
			filtered = append(filtered, it)
		}
		hasMore := len(filtered) > limit
		if hasMore {
			filtered = filtered[:limit]
		}
		writeHouse(w, 200, 0, map[string]any{"items": filtered, "has_more": hasMore})
	})
	f.mux.HandleFunc("PATCH /auth/me", func(w http.ResponseWriter, r *http.Request) {
		code := int(f.patchCode.Load())
		if code != 0 {
			status := 400
			if code == 10001 || code == 10002 || code == 10003 {
				status = 401
			}
			writeHouse(w, status, code, nil)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		name := "alice"
		if v, ok := body["name"].(string); ok {
			name = v
		}
		bio := "hello"
		if v, ok := body["bio"].(string); ok {
			bio = v
		}
		writeHouse(w, 200, 0, map[string]any{
			"uuid": "0097dd9f-b72b-4b1e-9d6c-2f1a3e5c7b90", "name": name, "bio": bio,
			"avatar": "", "avatar_image_hash": meAvatarHash,
		})
	})
	f.mux.HandleFunc("POST /auth/me/avatar", func(w http.ResponseWriter, r *http.Request) {
		if f.avatarFail.Load() {
			writeHouse(w, 500, 10, nil)
			return
		}
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		mr := multipart.NewReader(r.Body, params["boundary"])
		part, err := mr.NextPart()
		if err == nil {
			f.avatarName.Store(part.FileName())
			f.avatarCT.Store(part.Header.Get("Content-Type"))
			_, _ = io.Copy(io.Discard, part)
		}
		writeHouse(w, 200, 0, map[string]any{
			"hash": meAvatarHash, "url": "https://image.test.example/ff/ff/" + meAvatarHash + ".webp",
			"width": 64, "height": 64,
		})
	})
	f.mux.HandleFunc("PUT /auth/me/nsfw", func(w http.ResponseWriter, r *http.Request) {
		if f.nsfwFail.Load() {
			writeHouse(w, 500, 10, nil)
			return
		}
		if f.nsfwNoScope.Load() {
			writeHouse(w, 403, 18001, nil)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		writeHouse(w, 200, 0, map[string]any{"nsfw_display": body["nsfw_display"]})
	})
	f.mux.HandleFunc("GET /auth/me/preferences/{ns}", func(w http.ResponseWriter, _ *http.Request) {
		writeHouse(w, 200, 0, map[string]any{
			"doc": map[string]any{"show_rating": false}, "version": 4,
			"updated_at": "2026-09-22T08:31:00Z",
		})
	})
	f.mux.HandleFunc("PUT /auth/me/preferences/{ns}", func(w http.ResponseWriter, r *http.Request) {
		code := int(f.prefCode.Load())
		if code != 0 {
			status := int(f.prefHTTP.Load())
			if status == 0 {
				status = 400
			}
			writeHouse(w, status, code, nil)
			return
		}
		writeHouse(w, 200, 0, map[string]any{"doc": map[string]any{"ok": true}, "version": 5, "updated_at": "2026-09-22T08:32:00Z"})
	})
	f.mux.HandleFunc("GET /creator/applications/me", func(w http.ResponseWriter, _ *http.Request) {
		state, _ := f.creatorState.Load().(string)
		if state == "" {
			writeHouse(w, 200, 0, nil)
			return
		}
		writeHouse(w, 200, 0, fakeCreatorApp(state))
	})
	f.mux.HandleFunc("POST /creator/applications", func(w http.ResponseWriter, _ *http.Request) {
		code := int(f.creatorPost.Load())
		if code != 0 {
			writeHouse(w, 400, code, nil)
			return
		}
		writeHouse(w, 200, 0, fakeCreatorApp("pending"))
	})
}

func writeHouse(w http.ResponseWriter, httpStatus, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	env := map[string]any{"code": code, "message": "x"}
	if data != nil {
		env["data"] = data
	}
	_ = json.NewEncoder(w).Encode(env)
}

func fakeCreatorApp(state string) map[string]any {
	return map[string]any{
		"id": 9, "user_id": w3UserAlice, "source": "forum", "status": state,
		"message": "please", "decline_reason": "", "created_at": "2026-09-01T00:00:00Z",
	}
}

func fakeMoeLog() []map[string]any {
	out := make([]map[string]any, 0, 7)
	for i := 7; i >= 1; i-- {
		reason := "liked"
		if i%2 == 0 {
			reason = "daily_checkin"
		}
		src := "test-client"
		switch i {
		case 3:
			src = "other-app"
		case 5:
			src = "oauth"
		}
		out = append(out, map[string]any{
			"id": i, "delta": i, "reason": reason, "source_app": src,
			"ref": fmt.Sprintf("ref-%d", i), "created_at": "2026-09-01T00:00:00Z",
		})
	}
	return out
}

func (f *meFix) call(t *testing.T, method, rawURL, spec, session, key string, hdr http.Header, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, method, rawURL, session, spec, key, hdr, payload)
	return resp, problemMap(t, body)
}

func (f *meFix) deadStats(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f.UserService.ReplaceStatsRepo(userRepo.NewUserStatsRepository(f.db.WithContext(ctx)))
}

func (f *meFix) deadState(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f.UserService.ReplaceStateRepo(userRepo.NewStateRepository(f.db.WithContext(ctx)))
}

func fieldErr(t *testing.T, body map[string]any, locKey, loc, reason string) {
	t.Helper()
	errs, _ := body["errors"].([]any)
	if len(errs) == 0 {
		t.Fatalf("no errors in %+v", body)
	}
	e, _ := errs[0].(map[string]any)
	if e[locKey] != loc || e["reason"] != reason {
		t.Fatalf("field error %+v, want %s=%s reason=%s", e, locKey, loc, reason)
	}
}

func mustCode(t *testing.T, resp *http.Response, body map[string]any, status int, code string) {
	t.Helper()
	if resp.StatusCode != status || body["code"] != code {
		t.Fatalf("%d %v, want %d %s", resp.StatusCode, body["code"], status, code)
	}
}

func nextCursor(body map[string]any) string {
	s, _ := body["next_cursor"].(string)
	return s
}
