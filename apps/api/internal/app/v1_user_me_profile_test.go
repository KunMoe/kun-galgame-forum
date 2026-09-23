package app

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func TestV1SearchUsers(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/users?q=alice", "/users", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	ids := itemIDs(t, body)
	if len(ids) != 1 || ids[0] != strID(w3UserAlice) {
		t.Fatalf("items %v (banned must be filtered)", ids)
	}
	if _, ok := body["next_cursor"]; ok {
		t.Fatalf("search is not paginated: %v", body["next_cursor"])
	}
}

func TestV1SearchDoesNotPoisonUserCache(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/users?q=alice", "/users", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	before := f.nBatch.Load()
	if _, _, err := f.UserClient.User(context.Background(), w3UserAlice); err != nil {
		t.Fatal(err)
	}
	if f.nBatch.Load() <= before {
		t.Fatalf("User(%d) did not hit /users/batch after search (hot cache was poisoned)", w3UserAlice)
	}
}

func TestV1SearchBlankQueryIs422(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/users?q=%20%20", "/users", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "parameter", "q", "REQUIRED")
}

func TestV1SearchMissingQueryIs400(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/users", "/users", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("missing q %d %+v", resp.StatusCode, body)
	}
}

func TestV1SearchLimit21IsLimitTooLarge(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/users?q=alice&limit=21", "/users", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusBadRequest, "LIMIT_TOO_LARGE")
}

func TestV1SearchOAuthDownIs503(t *testing.T) {
	f := newMeFix(t)
	f.searchFail.Store(true)
	resp, body := f.call(t, http.MethodGet, "/api/v1/users?q=alice", "/users", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1PatchProfile(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodPatch, mePath+"/profile", "/me/profile", "sess-alice", "", nil, map[string]any{"bio": "new bio"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if body["object"] != "user" || body["bio"] != "new bio" {
		t.Fatalf("%+v", body)
	}
}

func TestV1PatchProfileRequiresAField(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodPatch, mePath+"/profile", "/me/profile", "sess-alice", "", nil, map[string]any{})
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "pointer", "", "REQUIRED")
}

func TestV1PatchProfileNameTaken(t *testing.T) {
	f := newMeFix(t)
	f.patchCode.Store(10007)
	resp, body := f.call(t, http.MethodPatch, mePath+"/profile", "/me/profile", "sess-alice", "", nil, map[string]any{"name": "taken"})
	mustCode(t, resp, body, http.StatusConflict, "USERNAME_TAKEN")
}

func TestV1PatchProfileInsufficientMoemoepoint(t *testing.T) {
	f := newMeFix(t)
	f.patchCode.Store(16006)
	resp, body := f.call(t, http.MethodPatch, mePath+"/profile", "/me/profile", "sess-alice", "", nil, map[string]any{"name": "newbie"})
	mustCode(t, resp, body, http.StatusForbidden, "MOEMOEPOINT_INSUFFICIENT")
	if _, ok := body["required"]; ok {
		t.Fatalf("required extension must not be set: %+v", body)
	}
}

func TestV1PatchProfileInvalidNameFormat(t *testing.T) {
	f := newMeFix(t)
	f.patchCode.Store(7)
	resp, body := f.call(t, http.MethodPatch, mePath+"/profile", "/me/profile", "sess-alice", "", nil, map[string]any{"name": "bad name"})
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "pointer", "/name", "INVALID_FORMAT")
}

func TestV1PatchProfileInvalidCredential(t *testing.T) {
	f := newMeFix(t)
	f.patchCode.Store(10002)
	resp, body := f.call(t, http.MethodPatch, mePath+"/profile", "/me/profile", "sess-alice", "", nil, map[string]any{"name": "alice2"})
	mustCode(t, resp, body, http.StatusUnauthorized, "INVALID_CREDENTIAL")
}

func TestV1PatchProfileOtherFailureIs503(t *testing.T) {
	f := newMeFix(t)
	f.patchCode.Store(10)
	resp, body := f.call(t, http.MethodPatch, mePath+"/profile", "/me/profile", "sess-alice", "", nil, map[string]any{"bio": "x"})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1PatchProfileNameChangeRefreshesBalance(t *testing.T) {
	f := newMeFix(t)
	f.moeBalance.Store(13)
	resp, _ := f.call(t, http.MethodPatch, mePath+"/profile", "/me/profile", "sess-alice", "", nil, map[string]any{"name": "alice2"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch %d", resp.StatusCode)
	}
	var bal int
	if err := f.db.Raw(`SELECT moemoepoint FROM kungal_user_state WHERE user_id = ?`, w3UserAlice).Scan(&bal).Error; err != nil {
		t.Fatal(err)
	}
	if bal != 13 {
		t.Fatalf("cached moemoepoint %d, want 13", bal)
	}
}

func TestV1AvatarRejectsPlainText(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.putAvatar(t, "sess-alice", "hello", "text/plain", "note.txt")
	mustCode(t, resp, body, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE")
}

func TestV1AvatarRejectsMissingFile(t *testing.T) {
	f := newMeFix(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.Close()
	resp, body := f.putAvatarRaw(t, "sess-alice", buf.Bytes(), mw.FormDataContentType())
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "pointer", "/file", "REQUIRED")
}

func TestV1AvatarRejectsTooLarge(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.putAvatar(t, "sess-alice", strings.Repeat("x", 4*1024*1024+1), "image/png", "big.png")
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "pointer", "/file", "TOO_LONG")
}

func TestV1AvatarUpstreamFailureIs503(t *testing.T) {
	f := newMeFix(t)
	f.avatarFail.Store(true)
	resp, body := f.putAvatar(t, "sess-alice", "png-bytes", "image/png", "pic.png")
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1AvatarRoundTrip(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.putAvatar(t, "sess-alice", "png-bytes", "image/png", "pic.png")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if body["hash"] != meAvatarHash {
		t.Fatalf("hash %v", body["hash"])
	}
	if got, _ := f.avatarName.Load().(string); got != "avatar" {
		t.Fatalf("upstream filename %q", got)
	}
	if got, _ := f.avatarCT.Load().(string); got != "image/png" {
		t.Fatalf("upstream content-type %q", got)
	}
}

func (f *meFix) putAvatar(t *testing.T, session, content, ct, filename string) (*http.Response, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", ct)
	part, err := mw.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, content); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return f.putAvatarRaw(t, session, buf.Bytes(), mw.FormDataContentType())
}

func (f *meFix) putAvatarRaw(t *testing.T, session string, raw []byte, contentType string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, mePath+"/avatar", bytes.NewReader(raw))
	req.Header.Set("Content-Type", contentType)
	if session != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: session})
	}
	resp, err := f.Fiber.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.checkPath(t, http.MethodPut, "/me/avatar", resp, body)
	return resp, problemMap(t, body)
}
