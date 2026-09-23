package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/artifactclient/gen"
)

const (
	g1HiddenA = 930002101
	g1HiddenB = 930002102

	g1TSTied = 930002201
	g1TSMain = 930002210
	g1TSBob  = 930002211
	g1TSBad  = 930002212
	g1TSGone = 930002299

	g1ResLink  = 930002301
	g1ResFile  = 930002302
	g1ResEmpty = 930002303
	g1ResBob   = 930002304
	g1ResOther = 930002305

	g1UpAliceA = "aaaaaaaa-bbbb-4ccc-8ddd-000000000001"
	g1UpAliceB = "aaaaaaaa-bbbb-4ccc-8ddd-000000000002"
	g1UpPend   = "aaaaaaaa-bbbb-4ccc-8ddd-000000000003"
	g1UpBob    = "aaaaaaaa-bbbb-4ccc-8ddd-000000000004"
	g1UpOther  = "aaaaaaaa-bbbb-4ccc-8ddd-000000000005"
)

type artifactFake struct {
	mu           sync.Mutex
	n            int
	sessions     map[string]*artSess
	failDelete   map[string]bool
	failDownload map[string]bool
	quota        bool
}

type artSess struct {
	uuid      string
	size      int64
	name      string
	completed bool
}

func newArtifactFake() *artifactFake {
	return &artifactFake{
		sessions:     map[string]*artSess{},
		failDelete:   map[string]bool{},
		failDownload: map[string]bool{},
	}
}

func (a *artifactFake) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/artifacts", a.init)
	mux.HandleFunc("POST /api/v1/artifacts/{uuid}/complete", a.complete)
	mux.HandleFunc("GET /api/v1/artifacts/{uuid}/resume", a.resume)
	mux.HandleFunc("GET /api/v1/artifacts/{uuid}/download", a.download)
	mux.HandleFunc("DELETE /api/v1/artifacts/{uuid}", a.delete)
	return mux
}

func (a *artifactFake) write(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "data": data, "message": ""})
}

func (a *artifactFake) init(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.quota {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		a.write(w, 50012, nil)
		return
	}
	var req gen.InitUploadRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	a.n++
	uuid := fmt.Sprintf("bbbbbbbb-cccc-4ddd-8eee-%012d", a.n)
	a.sessions[uuid] = &artSess{uuid: uuid, size: req.FileSize, name: req.Name}
	url := "https://upload.test/" + uuid
	a.write(w, 0, gen.InitUploadResponse{
		Uuid: uuid, Multipart: false, ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		UploadUrl: &url,
	})
}

func (a *artifactFake) complete(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	uuid := r.PathValue("uuid")
	s, ok := a.sessions[uuid]
	if !ok {
		a.write(w, 50001, nil)
		return
	}
	s.completed = true
	a.write(w, 0, gen.ArtifactResponse{Uuid: uuid, FileSize: s.size, Name: s.name, Status: 1})
}

func (a *artifactFake) resume(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	uuid := r.PathValue("uuid")
	s, ok := a.sessions[uuid]
	if !ok {
		a.write(w, 50001, nil)
		return
	}
	url := "https://upload.test/" + uuid
	a.write(w, 0, map[string]any{
		"uuid": uuid, "multipart": false, "expires_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		"upload_url": url, "file_size": s.size,
	})
}

func (a *artifactFake) download(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	uuid := r.PathValue("uuid")
	if a.failDownload[uuid] {
		w.WriteHeader(http.StatusInternalServerError)
		a.write(w, 1, nil)
		return
	}
	if _, ok := a.sessions[uuid]; !ok {
		a.sessions[uuid] = &artSess{uuid: uuid, size: 10, completed: true}
	}
	exp := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	a.write(w, 0, gen.DownloadResponse{Url: "https://dl.test/" + uuid, ExpiresAt: &exp})
}

func (a *artifactFake) delete(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	uuid := r.PathValue("uuid")
	if a.failDelete[uuid] {
		w.WriteHeader(http.StatusInternalServerError)
		a.write(w, 1, nil)
		return
	}
	delete(a.sessions, uuid)
	a.write(w, 0, gen.DeleteData{Deleted: true, Uuid: uuid})
}

func (a *artifactFake) seed(uuid string, size int64, completed bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions[uuid] = &artSess{uuid: uuid, size: size, completed: completed, name: uuid + ".zip"}
}

func newToolsetFix(t *testing.T, checker gate.Checker) *writeFix {
	t.Helper()
	f := newWriteFix(t, checker)
	f.alice(t)
	f.putSession(t, "sess-banned", w3UserBanned)
	f.addOAuthUser(g1HiddenA, "hidden-a", 1, nil)
	f.addOAuthUser(g1HiddenB, "hidden-b", 1, nil)
	f.cleanupToolsets(t)
	t.Cleanup(func() { f.cleanupToolsets(t) })
	f.seedToolsets(t)
	return f
}

func (f *writeFix) cleanupToolsets(t *testing.T) {
	t.Helper()
	const owned = `SELECT id FROM galgame_toolset WHERE id BETWEEN 930002201 AND 930002299 OR user_id BETWEEN 930000001 AND 930002199`
	for _, q := range []string{
		`DELETE FROM galgame_toolset_resource WHERE toolset_id IN (` + owned + `) OR user_id BETWEEN 930000001 AND 930002199`,
		`DELETE FROM galgame_toolset_alias WHERE toolset_id IN (` + owned + `)`,
		`DELETE FROM galgame_toolset_contributor WHERE toolset_id IN (` + owned + `)`,
		`DELETE FROM galgame_toolset_practicality WHERE toolset_id IN (` + owned + `)`,
		`DELETE FROM toolset_upload WHERE toolset_id IN (` + owned + `)`,
		`DELETE FROM galgame_toolset WHERE id IN (` + owned + `)`,
		`DELETE FROM kungal_user_state WHERE user_id IN (930002101, 930002102)`,
	} {
		_ = f.db.Exec(q).Error
	}
}

func (f *writeFix) seedToolsets(t *testing.T) {
	t.Helper()
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("toolset seed: %v\n%s", err, q)
		}
	}
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	tied := time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)
	run(`INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated) VALUES (?, 7, ?, ?), (?, 7, ?, ?)
		ON CONFLICT (user_id) DO NOTHING`, g1HiddenA, base, base, g1HiddenB, base, base)

	authors := []int{w3UserAlice, w3UserBob, w3UserOther, w3UserGrant, w3UserStaff, g1HiddenA, g1HiddenB}
	for i := 0; i < 7; i++ {
		id := g1TSTied + i
		run(`INSERT INTO galgame_toolset (id, name, description, type, language, platform, version, homepage, user_id, view, comment_count, created, updated, resource_update_time)
			VALUES (?, ?, '', 'extractor', 'zh-cn', 'windows', 'stable', '[]', ?, ?, 0, ?, ?, ?)`,
			id, fmt.Sprintf("Tied %d", id), authors[i], 10-i, tied, tied, tied)
	}

	run(`INSERT INTO galgame_toolset (id, name, description, type, language, platform, version, homepage, user_id, view, comment_count, created, updated, resource_update_time)
		VALUES (?, 'Main tool', 'hello world', 'extractor', 'zh-cn', 'windows', 'stable', '["https://example.com"]', ?, 3, 0, ?, ?, ?)`,
		g1TSMain, w3UserAlice, base, base, base)
	run(`INSERT INTO galgame_toolset_alias (name, toolset_id, created, updated) VALUES ('alias-a', ?, ?, ?)`, g1TSMain, base, base)
	run(`INSERT INTO galgame_toolset_contributor (toolset_id, user_id, created, updated) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
		g1TSMain, w3UserAlice, base, base, g1TSMain, w3UserBob, base, base)

	run(`INSERT INTO galgame_toolset (id, name, description, type, language, platform, version, homepage, user_id, view, comment_count, created, updated, resource_update_time)
		VALUES (?, 'Bob tool', '', 'launcher', 'en-us', 'mac', 'beta', '[]', ?, 0, 0, ?, ?, ?)`,
		g1TSBob, w3UserBob, base.Add(-time.Hour), base, base)
	run(`INSERT INTO galgame_toolset (id, name, description, type, language, platform, version, homepage, user_id, view, comment_count, created, updated, resource_update_time)
		VALUES (?, 'Bad homepage', '', 'others', 'ja-jp', 'linux', 'alpha', '[]', ?, 0, 0, ?, ?, ?)`,
		g1TSBad, w3UserAlice, base.Add(-2*time.Hour), base, base)

	run(`INSERT INTO galgame_toolset_resource (id, content, type, artifact_uuid, code, password, size, note, download, toolset_id, user_id, created, updated)
		VALUES (?, 'https://files.example/a.zip', 'user', '', 'ex1', 'pw1', '10mb', 'a note', 4, ?, ?, ?, ?)`,
		g1ResLink, g1TSMain, w3UserAlice, base, base)
	run(`INSERT INTO galgame_toolset_resource (id, content, type, artifact_uuid, code, password, size, note, download, toolset_id, user_id, created, updated)
		VALUES (?, '', 's3', ?, '', 'pw2', '2048', 'file note', 2, ?, ?, ?, ?)`,
		g1ResFile, g1UpAliceA, g1TSMain, w3UserAlice, base.Add(time.Minute), base)
	run(`INSERT INTO galgame_toolset_resource (id, content, type, artifact_uuid, code, password, size, note, download, toolset_id, user_id, created, updated)
		VALUES (?, '', 's3', '', 'legacy-key', '', '', '', 0, ?, ?, ?, ?)`,
		g1ResEmpty, g1TSMain, w3UserAlice, base.Add(2*time.Minute), base)
	run(`INSERT INTO galgame_toolset_resource (id, content, type, artifact_uuid, code, password, size, note, download, toolset_id, user_id, created, updated)
		VALUES (?, 'https://files.example/bob.zip', 'user', '', '', '', '1mb', '', 1, ?, ?, ?, ?)`,
		g1ResBob, g1TSMain, w3UserBob, base.Add(3*time.Minute), base)
	run(`INSERT INTO galgame_toolset_resource (id, content, type, artifact_uuid, code, password, size, note, download, toolset_id, user_id, created, updated)
		VALUES (?, 'https://files.example/other.zip', 'user', '', '', '', '2mb', '', 0, ?, ?, ?, ?)`,
		g1ResOther, g1TSBob, w3UserBob, base, base)

	run(`INSERT INTO toolset_upload (artifact_uuid, toolset_id, user_id, filename, file_size, created, completed_at)
		VALUES (?, ?, ?, 'a.zip', 2048, ?, ?), (?, ?, ?, 'b.zip', 4096, ?, ?), (?, ?, ?, 'p.zip', 1024, ?, NULL),
		(?, ?, ?, 'bob.zip', 512, ?, ?), (?, ?, ?, 'other.zip', 256, ?, ?)`,
		g1UpAliceA, g1TSMain, w3UserAlice, base, base,
		g1UpAliceB, g1TSMain, w3UserAlice, base, base,
		g1UpPend, g1TSMain, w3UserAlice, base,
		g1UpBob, g1TSMain, w3UserBob, base, base,
		g1UpOther, g1TSBob, w3UserAlice, base, base)

	run(`INSERT INTO galgame_toolset_practicality (rate, user_id, toolset_id, created, updated)
		VALUES (5, ?, ?, ?, ?), (4, ?, ?, ?, ?)`,
		w3UserAlice, g1TSMain, base, base, w3UserBob, g1TSMain, base, base)

	f.art.seed(g1UpAliceA, 2048, true)
	f.art.seed(g1UpAliceB, 4096, true)
	f.art.seed(g1UpPend, 1024, false)
	f.art.seed(g1UpBob, 512, true)
	f.art.seed(g1UpOther, 256, true)
}

func (f *writeFix) ts(t *testing.T, method, rawURL, spec, session, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, method, rawURL, session, spec, idem, nil, payload)
	if len(body) == 0 {
		return resp, nil
	}
	return resp, problemMap(t, body)
}

func createToolsetBody(name string, extra map[string]any) map[string]any {
	body := map[string]any{
		"title": name, "toolset_type": "extractor", "interface_language": "zh-cn", "platform": "windows", "release_channel": "stable",
	}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func (f *writeFix) walkToolsets(t *testing.T, query string, limit int) []string {
	t.Helper()
	var got []string
	for page := 1; page <= 40; page++ {
		url := fmt.Sprintf("/api/v1/toolsets?page=%d&limit=%d%s", page, limit, query)
		resp, body := f.ts(t, http.MethodGet, url, "/toolsets", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("walk %s: %d %+v", url, resp.StatusCode, body)
		}
		ids := adminItemIDs(body)
		got = append(got, ids...)
		if len(ids) < limit {
			return got
		}
	}
	t.Fatal("walk did not end")
	return nil
}

func errorHeader(body map[string]any, header string) map[string]any {
	for _, e := range problemErrorsOf(body) {
		if e["header"] == header {
			return e
		}
	}
	return nil
}

func errorParam(body map[string]any, param string) map[string]any {
	for _, e := range problemErrorsOf(body) {
		if e["parameter"] == param {
			return e
		}
	}
	return nil
}

func idStr(n int) string { return strconv.Itoa(n) }

func wantCode(t *testing.T, resp *http.Response, body map[string]any, status int, code string) {
	t.Helper()
	if resp.StatusCode != status || body["code"] != code {
		t.Fatalf("got %d %v, want %d %s\n%+v", resp.StatusCode, body["code"], status, code, body)
	}
}

func renderableAuthorsSQL() string {
	return strings.Join([]string{
		strconv.Itoa(w3UserAlice), strconv.Itoa(w3UserBob), strconv.Itoa(w3UserOther),
		strconv.Itoa(w3UserGrant), strconv.Itoa(w3UserStaff),
	}, ",")
}
