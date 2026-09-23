package app

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	imageapiv1 "kun-galgame-api/internal/image/apiv1"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/imageclient"

	"github.com/gofiber/fiber/v3"
)

type fakeImageHost struct {
	mu         sync.Mutex
	presets    []string
	failCode   int
	failStatus int
	delay      time.Duration
	sexual     map[string]int16
}

func (h *fakeImageHost) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /image/upload", func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		data, _ := io.ReadAll(file)
		sum := sha256.Sum256(data)
		h.mu.Lock()
		h.presets = append(h.presets, r.FormValue("preset"))
		failCode, failStatus, delay := h.failCode, h.failStatus, h.delay
		h.mu.Unlock()
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		if failCode != 0 {
			w.WriteHeader(failStatus)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": failCode, "message": "refused"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "ok", "data": map[string]any{
			"hash": hex.EncodeToString(sum[:]), "url": "https://elsewhere.example/x.webp",
			"width": 640, "height": 360, "thumbhash": "AbC+",
		}})
	})
	mux.HandleFunc("POST /image/meta-batch", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Hashes []string `json:"hashes"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		metas := map[string]any{}
		h.mu.Lock()
		for _, hash := range in.Hashes {
			if s, ok := h.sexual[hash]; ok {
				metas[hash] = map[string]any{"width": 640, "height": 360, "sexual": s}
			}
		}
		h.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"metas": metas}})
	})
	return mux
}

func (h *fakeImageHost) seenPresets() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.presets...)
}

func (h *fakeImageHost) grade(hash string, sexual int16) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sexual[hash] = sexual
}

func (h *fakeImageHost) slow(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.delay = d
}

func (h *fakeImageHost) fail(code, status int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failCode, h.failStatus = code, status
}

type fakeCatalogEdit struct {
	mu      sync.Mutex
	auth    []string
	presets []string
	status  int
	code    string
}

func (c *fakeCatalogEdit) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/me/edit-images", func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		data, _ := io.ReadAll(file)
		sum := sha256.Sum256(data)
		c.mu.Lock()
		c.auth = append(c.auth, r.Header.Get("Authorization"))
		c.presets = append(c.presets, r.FormValue("preset"))
		status, code := c.status, c.code
		c.mu.Unlock()
		if status != 0 {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "title": "refused", "detail": "the catalog says no"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"hash": hex.EncodeToString(sum[:]), "url": "https://elsewhere.example/y.webp",
			"width": 1280, "height": 720, "thumbhash": "XyZ=", "size_bytes": len(data),
		})
	})
	return mux
}

func (c *fakeCatalogEdit) respond(status int, code string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status, c.code = status, code
}

func (c *fakeCatalogEdit) last() (auth, preset string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.auth[len(c.auth)-1], c.presets[len(c.presets)-1]
}

func newImagesFix(t *testing.T) (*writeFix, *fakeImageHost, *fakeCatalogEdit) {
	t.Helper()
	f := newWriteFix(t, nil)
	host := &fakeImageHost{sexual: map[string]int16{}}
	hs := httptest.NewServer(host.handler())
	t.Cleanup(hs.Close)
	cat := &fakeCatalogEdit{}
	cs := httptest.NewServer(cat.handler())
	t.Cleanup(cs.Close)
	images := imageclient.New(imageclient.Config{BaseURL: hs.URL, ClientID: "c", ClientSecret: "s"})
	f.ImagesV1 = imageapiv1.New(images, catalogclient.New(catalogclient.Config{BaseURL: cs.URL}), f.db, "https://image.test.example")
	f.Fiber = newFiber()
	f.setupRoutes()
	f.alice(t)
	return f, host, cat
}

// app.Test refuses to send a body over BodyLimit, and a client that streams
// one races the server's close: the first run passed, the second read
// "connection reset by peer". The server refuses on Content-Length alone, so
// only the headers are sent.
func (f *writeFix) postHeadersOnly(t *testing.T, path, contentType string, length int) (*http.Response, []byte) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = f.Fiber.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() { _ = f.Fiber.Shutdown() })
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := fmt.Fprintf(conn, "POST /api/v1%s HTTP/1.1\r\nHost: kungal.test\r\nContent-Type: %s\r\nContent-Length: %d\r\nCookie: %s=sess-alice\r\n\r\n",
		path, contentType, length, middleware.SessionCookieName); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	f.spec.checkPath(t, http.MethodPost, path, resp, body)
	return resp, body
}

type imageForm struct {
	fields   map[string]string
	data     []byte
	partType string
	noFile   bool
}

func (in imageForm) encode(t *testing.T) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range in.fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if !in.noFile {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="file"; filename="pic.png"`)
		ct := in.partType
		if ct == "" {
			ct = "image/png"
		}
		h.Set("Content-Type", ct)
		part, err := mw.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		data := in.data
		if data == nil {
			data = []byte("PNG!")
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes(), mw.FormDataContentType()
}

type caller struct {
	session string
	bearer  string
}

var (
	asAlice     = caller{session: "sess-alice"}
	asBobBearer = caller{bearer: "bob-token"}
)

func (f *writeFix) uploadRaw(t *testing.T, path string, who caller, raw []byte, contentType string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1"+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", contentType)
	if who.session != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: who.session})
	}
	if who.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+who.bearer)
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
	f.spec.checkPath(t, http.MethodPost, path, resp, body)
	return resp, problemMap(t, body)
}

func (f *writeFix) uploadImage(t *testing.T, who caller, form imageForm) (*http.Response, map[string]any) {
	t.Helper()
	raw, ct := form.encode(t)
	return f.uploadRaw(t, "/images", who, raw, ct)
}

func (f *writeFix) dailyImages(t *testing.T, userID int) int {
	t.Helper()
	var n int
	if err := f.db.Raw(`SELECT daily_image_count FROM kungal_user_state WHERE user_id = ?`, userID).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *writeFix) setDailyImages(t *testing.T, userID, n int) {
	t.Helper()
	if err := f.db.Exec(`UPDATE kungal_user_state SET daily_image_count = ? WHERE user_id = ?`, n, userID).Error; err != nil {
		t.Fatal(err)
	}
}

func contentImage(data string) imageForm {
	return imageForm{fields: map[string]string{"purpose": "content"}, data: []byte(data)}
}

func TestV1ImagesShapeAndPresets(t *testing.T) {
	f, host, _ := newImagesFix(t)
	resp, body := f.uploadImage(t, asAlice, contentImage("one"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload %d %+v", resp.StatusCode, body)
	}
	sum := sha256.Sum256([]byte("one"))
	hash := hex.EncodeToString(sum[:])
	want := "https://image.test.example/" + hash[:2] + "/" + hash[2:4] + "/" + hash + ".webp"
	if body["hash"] != hash || body["url"] != want || resp.Header.Get("Location") != want {
		t.Errorf("image %+v location %q", body, resp.Header.Get("Location"))
	}
	if asInt(body["width"]) != 640 || body["thumbhash"] != "AbC+" || body["sexual"] != nil {
		t.Errorf("a new image has no grade yet: %+v", body)
	}

	sum = sha256.Sum256([]byte("graded"))
	host.grade(hex.EncodeToString(sum[:]), 2)
	_, body = f.uploadImage(t, asAlice, imageForm{fields: map[string]string{"purpose": "message"}, data: []byte("graded")})
	if body["sexual"] != "explicit" {
		t.Errorf("a deduplicated graded image carries its grade: %+v", body)
	}
	if got := strings.Join(host.seenPresets(), ","); got != "topic,message" {
		t.Errorf("presets sent to the image host: %s", got)
	}
	if n := f.dailyImages(t, w3UserAlice); n != 2 {
		t.Errorf("daily count %d", n)
	}
}

func TestV1ImagesDailyLimit(t *testing.T) {
	f, _, _ := newImagesFix(t)
	f.setDailyImages(t, w3UserAlice, 49)
	if resp, body := f.uploadImage(t, asAlice, contentImage("fiftieth")); resp.StatusCode != http.StatusCreated {
		t.Fatalf("50th upload %d %+v", resp.StatusCode, body)
	}
	resp, body := f.uploadImage(t, asAlice, contentImage("fifty-first"))
	if resp.StatusCode != http.StatusTooManyRequests || body["code"] != "IMAGE_DAILY_LIMIT_REACHED" || asInt(body["limit"]) != 50 {
		t.Errorf("51st upload %d %+v", resp.StatusCode, body)
	}
	if n := f.dailyImages(t, w3UserAlice); n != 50 {
		t.Errorf("daily count %d", n)
	}
}

func TestV1ImagesQuotaIsAtomic(t *testing.T) {
	f, host, _ := newImagesFix(t)
	host.slow(30 * time.Millisecond)
	f.setDailyImages(t, w3UserAlice, 45)
	var mu sync.Mutex
	statuses := map[int]int{}
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			resp, _ := f.uploadImage(t, asAlice, contentImage(strings.Repeat("x", i+1)))
			mu.Lock()
			statuses[resp.StatusCode]++
			mu.Unlock()
		})
	}
	wg.Wait()
	if statuses[http.StatusCreated] != 5 || statuses[http.StatusTooManyRequests] != 15 {
		t.Errorf("20 concurrent uploads at 45/50: %v", statuses)
	}
	if n := f.dailyImages(t, w3UserAlice); n != 50 {
		t.Errorf("daily count %d", n)
	}
}

func TestV1ImagesUpstreamFailuresRefund(t *testing.T) {
	f, host, _ := newImagesFix(t)
	for _, tc := range []struct {
		code, status int
		want         int
		problem      string
	}{
		{60002, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, "IMAGE_REJECTED"},
		{80003, http.StatusUnauthorized, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
		{80008, http.StatusTooManyRequests, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
		{80009, http.StatusUnsupportedMediaType, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE"},
		{80007, http.StatusRequestEntityTooLarge, http.StatusUnprocessableEntity, "VALIDATION_FAILED"},
		{80099, http.StatusBadRequest, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
	} {
		host.fail(tc.code, tc.status)
		resp, body := f.uploadImage(t, asAlice, contentImage("x"))
		if resp.StatusCode != tc.want || body["code"] != tc.problem {
			t.Errorf("image host %d: %d %+v", tc.code, resp.StatusCode, body)
		}
		if n := f.dailyImages(t, w3UserAlice); n != 0 {
			t.Errorf("image host %d: a failed upload counted, daily count %d", tc.code, n)
		}
	}
}

func TestV1ImagesProvisionTheStateRow(t *testing.T) {
	f, _, _ := newImagesFix(t)
	if err := f.db.Exec(`DELETE FROM kungal_user_state WHERE user_id = ?`, w3UserBob).Error; err != nil {
		t.Fatal(err)
	}
	if resp, body := f.uploadImage(t, asBobBearer, contentImage("app")); resp.StatusCode != http.StatusCreated {
		t.Fatalf("app-only account %d %+v", resp.StatusCode, body)
	}
	if n := f.dailyImages(t, w3UserBob); n != 1 {
		t.Errorf("daily count %d", n)
	}
}

func TestV1ImagesRequestChecks(t *testing.T) {
	f, host, _ := newImagesFix(t)
	for name, tc := range map[string]struct {
		form    imageForm
		status  int
		code    string
		pointer string
		reason  string
	}{
		"text part":       {imageForm{fields: map[string]string{"purpose": "content"}, partType: "text/plain"}, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "", ""},
		"no file":         {imageForm{fields: map[string]string{"purpose": "content"}, noFile: true}, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "/file", "REQUIRED"},
		"no purpose":      {imageForm{}, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "/purpose", "REQUIRED"},
		"unknown purpose": {imageForm{fields: map[string]string{"purpose": "avatar"}}, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "/purpose", "UNKNOWN_VALUE"},
		"file over 10 MiB": {imageForm{fields: map[string]string{"purpose": "content"}, data: bytes.Repeat([]byte{1}, 10*1024*1024+1)},
			http.StatusUnprocessableEntity, "VALIDATION_FAILED", "/file", "TOO_LONG"},
	} {
		resp, body := f.uploadImage(t, asAlice, tc.form)
		if resp.StatusCode != tc.status || body["code"] != tc.code {
			t.Errorf("%s: %d %+v", name, resp.StatusCode, body)
			continue
		}
		if tc.pointer != "" {
			errs, _ := body["errors"].([]any)
			first, _ := errs[0].(map[string]any)
			if first["pointer"] != tc.pointer || first["reason"] != tc.reason {
				t.Errorf("%s: %+v", name, errs)
			}
		}
	}
	if resp, body := f.uploadRaw(t, "/images", asAlice, []byte(`{"purpose":"content"}`), "application/json"); resp.StatusCode != http.StatusUnsupportedMediaType || body["code"] != "UNSUPPORTED_MEDIA_TYPE" {
		t.Errorf("json body: %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.uploadImage(t, caller{}, contentImage("anon")); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous: %d %+v", resp.StatusCode, body)
	}
	if got := host.seenPresets(); len(got) != 0 {
		t.Errorf("refused requests reached the image host: %v", got)
	}
	if n := f.dailyImages(t, w3UserAlice); n != 0 {
		t.Errorf("refused requests counted: %d", n)
	}
}

func TestV1OversizedBodiesAre413(t *testing.T) {
	f, host, _ := newImagesFix(t)
	resp, got := f.postHeadersOnly(t, "/images", "multipart/form-data; boundary=x", 11*1024*1024)
	if body := problemMap(t, got); resp.StatusCode != http.StatusRequestEntityTooLarge || body["code"] != "PAYLOAD_TOO_LARGE" {
		t.Errorf("11 MiB body: %d %+v", resp.StatusCode, body)
	}
	if got := host.seenPresets(); len(got) != 0 {
		t.Errorf("oversized body reached the image host: %v", got)
	}

	payload := map[string]any{"title": "big", "content": strings.Repeat("a", 2*1024*1024)}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", "018f5b3a-0000-7000-8000-000000000413", nil, payload)
	if body := problemMap(t, raw); resp.StatusCode != http.StatusRequestEntityTooLarge || body["code"] != "PAYLOAD_TOO_LARGE" {
		t.Errorf("2 MiB JSON body: %d %+v", resp.StatusCode, body)
	}
}

func TestV1ImagesUnconfigured(t *testing.T) {
	f, _, _ := newImagesFix(t)
	f.ImagesV1 = imageapiv1.New(nil, nil, f.db, "https://image.test.example")
	f.Fiber = newFiber()
	f.setupRoutes()
	if resp, body := f.uploadImage(t, asAlice, contentImage("x")); resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("no image client: %d %+v", resp.StatusCode, body)
	}
	raw, ct := imageForm{fields: map[string]string{"preset": "cover"}}.encode(t)
	if resp, body := f.uploadRaw(t, "/work-edit-images", asAlice, raw, ct); resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("no catalog client: %d %+v", resp.StatusCode, body)
	}
	if n := f.dailyImages(t, w3UserAlice); n != 0 {
		t.Errorf("daily count %d", n)
	}
}

func TestV1WorkEditImages(t *testing.T) {
	f, host, cat := newImagesFix(t)
	for _, tc := range []struct {
		who    caller
		preset string
		auth   string
	}{
		{asAlice, "cover", "Bearer access"},
		{asBobBearer, "screenshot", "Bearer bob-token"},
	} {
		raw, ct := imageForm{fields: map[string]string{"preset": tc.preset}, data: []byte(tc.preset)}.encode(t)
		resp, body := f.uploadRaw(t, "/work-edit-images", tc.who, raw, ct)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("%s: %d %+v", tc.preset, resp.StatusCode, body)
		}
		sum := sha256.Sum256([]byte(tc.preset))
		hash := hex.EncodeToString(sum[:])
		if body["hash"] != hash || resp.Header.Get("Location") != body["url"] || body["thumbhash"] != "XyZ=" ||
			asInt(body["width"]) != 1280 || body["sexual"] != nil {
			t.Errorf("%s: %+v", tc.preset, body)
		}
		if auth, preset := cat.last(); auth != tc.auth || preset != tc.preset {
			t.Errorf("catalog saw %q %q, want %q %q", auth, preset, tc.auth, tc.preset)
		}
	}
	if len(host.seenPresets()) != 0 || f.dailyImages(t, w3UserAlice) != 0 {
		t.Error("work edit images went to the image host or counted against the daily limit")
	}
	raw, ct := imageForm{fields: map[string]string{"preset": "galgame_banner"}}.encode(t)
	if resp, body := f.uploadRaw(t, "/work-edit-images", asAlice, raw, ct); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("legacy preset name: %d %+v", resp.StatusCode, body)
	}
}

func TestV1WorkEditImageUpstreamErrors(t *testing.T) {
	f, _, cat := newImagesFix(t)
	for _, tc := range []struct {
		status int
		code   string
		want   int
		result string
	}{
		{http.StatusForbidden, "SCOPE_REQUIRED", http.StatusForbidden, "SCOPE_REQUIRED"},
		{http.StatusForbidden, "PERMISSION_REQUIRED", http.StatusForbidden, "PERMISSION_REQUIRED"},
		{http.StatusUnauthorized, "INVALID_CREDENTIAL", http.StatusUnauthorized, "INVALID_CREDENTIAL"},
		{http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE"},
		{http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", http.StatusUnprocessableEntity, "VALIDATION_FAILED"},
		{http.StatusUnprocessableEntity, "VALIDATION_FAILED", http.StatusUnprocessableEntity, "VALIDATION_FAILED"},
		{http.StatusTooManyRequests, "RATE_LIMITED", http.StatusTooManyRequests, "RATE_LIMITED"},
		{http.StatusConflict, "CONFLICT", http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
		{http.StatusBadGateway, "", http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
	} {
		cat.respond(tc.status, tc.code)
		raw, ct := imageForm{fields: map[string]string{"preset": "cover"}}.encode(t)
		resp, body := f.uploadRaw(t, "/work-edit-images", asAlice, raw, ct)
		if resp.StatusCode != tc.want || body["code"] != tc.result {
			t.Errorf("catalog %d %s: %d %+v", tc.status, tc.code, resp.StatusCode, body)
		}
	}
}
