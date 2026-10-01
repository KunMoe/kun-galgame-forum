package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"testing"

	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/problem"
)

const (
	newsSubImages  = "/me/news-submission-images"
	newsImagesPath = "POST /v2/me/news-images"
)

var asBanned = caller{session: "sess-banned"}

func newsImageHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (f *writeFix) uploadNewsImage(t *testing.T, who caller, form imageForm) (*http.Response, map[string]any) {
	t.Helper()
	raw, ct := form.encode(t)
	return f.uploadRaw(t, newsSubImages, who, raw, ct)
}

func TestV1NewsSubmissionImageUpload(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	data := []byte("banner bytes")
	resp, body := f.uploadNewsImage(t, asAlice, imageForm{data: data, partType: "image/webp"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload %d %+v", resp.StatusCode, body)
	}
	hash := newsImageHash(data)
	url := imageclient.MainURL("https://image.test.example", hash, "webp")
	if body["hash"] != hash || body["url"] != url || resp.Header.Get("Location") != url {
		t.Errorf("image %+v, Location %q", body, resp.Header.Get("Location"))
	}
	if asInt(body["width"]) != 1280 || asInt(body["height"]) != 720 || body["thumbhash"] != "AbC+" || body["sexual"] != nil {
		t.Errorf("dimensions come from the upload result, sexual stays null: %+v", body)
	}

	calls := up.calls()
	if len(calls) != 1 || calls[0].Method != http.MethodPost || calls[0].Path != "/v2/me/news-images" {
		t.Fatalf("upstream %+v", calls)
	}
	if calls[0].Header.Get("Authorization") != "Bearer access" {
		t.Errorf("the upload must carry the user's own token: %q", calls[0].Header.Get("Authorization"))
	}
	_, params, err := mime.ParseMediaType(calls[0].Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	form, err := multipart.NewReader(bytes.NewReader(calls[0].Raw), params["boundary"]).ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(form.Value) != 0 || len(form.File) != 1 || len(form.File["file"]) != 1 {
		t.Fatalf("upstream takes one part named file and nothing else: values %v files %v", form.Value, form.File)
	}
	part, _ := form.File["file"][0].Open()
	sent, _ := io.ReadAll(part)
	if !bytes.Equal(sent, data) {
		t.Errorf("upstream got %q", sent)
	}
}

func TestV1NewsSubmissionImageRefusedBeforeUpstream(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)

	resp, body := f.uploadNewsImage(t, asBanned, imageForm{})
	wantCode(t, resp, body, http.StatusForbidden, problem.CodeAccountBanned)

	resp, body = f.uploadNewsImage(t, caller{}, imageForm{})
	wantCode(t, resp, body, http.StatusUnauthorized, problem.CodeMissingCredential)

	resp, body = f.uploadRaw(t, newsSubImages, asAlice, []byte(`{}`), "application/json")
	wantCode(t, resp, body, http.StatusUnsupportedMediaType, problem.CodeUnsupportedMediaType)

	resp, body = f.uploadNewsImage(t, asAlice, imageForm{noFile: true})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := firstError(t, body); e["pointer"] != "/file" || e["reason"] != problem.ReasonRequired {
		t.Errorf("no file %+v", e)
	}

	resp, body = f.uploadNewsImage(t, asAlice, imageForm{partType: "image/gif"})
	wantCode(t, resp, body, http.StatusUnsupportedMediaType, problem.CodeUnsupportedMediaType)

	resp, body = f.uploadNewsImage(t, asAlice, imageForm{data: bytes.Repeat([]byte("x"), 4_000_001)})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := firstError(t, body); e["pointer"] != "/file" || e["reason"] != problem.ReasonTooLong {
		t.Errorf("too large %+v", e)
	}

	if n := len(up.calls()); n != 0 {
		t.Fatalf("upstream was called %d times", n)
	}

	resp, body = f.uploadNewsImage(t, asAlice, imageForm{data: bytes.Repeat([]byte("x"), 4_000_000)})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("4,000,000 bytes is allowed: %d %+v", resp.StatusCode, body)
	}
}

func TestV1NewsSubmissionImageUpstreamErrors(t *testing.T) {
	field := func(reason string) string {
		return `{"code":"VALIDATION_FAILED","errors":[{"pointer":"/file","reason":"` + reason + `"}]}`
	}
	cases := []struct {
		name               string
		upStatus           int
		upBody, upRetry    string
		status             int
		code               string
		pointer, reason    string
		retry              string
		upstreamHashBroken bool
	}{
		{name: "moderation or type refused", upStatus: 422, upBody: field("NOT_ALLOWED_VALUE"), status: 422, code: problem.CodeImageRejected},
		{name: "undecodable", upStatus: 422, upBody: field("INVALID_FORMAT"), status: 422, code: problem.CodeValidationFailed, pointer: "/file", reason: problem.ReasonInvalidFormat},
		{name: "body too large upstream", upStatus: 413, upBody: `{"code":"PAYLOAD_TOO_LARGE"}`, status: 422, code: problem.CodeValidationFailed, pointer: "/file", reason: problem.ReasonTooLong},
		{name: "daily upload limit, no Retry-After", upStatus: 429, upBody: `{"code":"QUOTA_EXCEEDED"}`, status: 429, code: problem.CodeQuotaExceeded},
		{name: "request quota, Retry-After kept", upStatus: 429, upBody: `{"code":"QUOTA_EXCEEDED"}`, upRetry: "120", status: 429, code: problem.CodeQuotaExceeded, retry: "120"},
		{name: "rate limited", upStatus: 429, upBody: `{"code":"RATE_LIMITED"}`, upRetry: "7", status: 429, code: problem.CodeRateLimited, retry: "7"},
		{name: "scope", upStatus: 403, upBody: `{"code":"SCOPE_REQUIRED"}`, status: 403, code: problem.CodeScopeRequired},
		{name: "token refused", upStatus: 401, upBody: `{"code":"INVALID_CREDENTIAL"}`, status: 401, code: problem.CodeInvalidCredential},
		{name: "image leg down", upStatus: 503, upBody: `{"code":"SERVICE_UNAVAILABLE"}`, status: 503, code: problem.CodeServiceUnavailable},
		{name: "no usable hash", upStatus: 201, upBody: `{"object":"news_image","url":"https://x.example/a.webp","hash":"nope"}`, status: 503, code: problem.CodeServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, up := newNewsSubmissionFix(t, nil)
			up.fail(newsImagesPath, tc.upStatus, tc.upBody, tc.upRetry)
			resp, body := f.uploadNewsImage(t, asAlice, imageForm{})
			wantCode(t, resp, body, tc.status, tc.code)
			if tc.pointer != "" {
				if e := firstError(t, body); e["pointer"] != tc.pointer || e["reason"] != tc.reason {
					t.Errorf("field %+v", e)
				}
			}
			if got := resp.Header.Get("Retry-After"); tc.status == http.StatusTooManyRequests && got != tc.retry {
				t.Errorf("Retry-After %q, want %q", got, tc.retry)
			}
		})
	}
}

func TestV1NewsSubmissionBanner(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	data := []byte("the banner")
	hash := newsImageHash(data)
	if resp, body := f.uploadNewsImage(t, asAlice, imageForm{data: data}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload %d %+v", resp.StatusCode, body)
	}

	withBanner := newsCreateBody()
	withBanner["banner_image_hash"] = hash
	resp, body := f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", keyUUID(30), withBanner)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id, _ := body["id"].(string)
	banner, _ := body["banner"].(map[string]any)
	if banner["hash"] != hash || banner["url"] != imageclient.MainURL("https://image.test.example", hash, "webp") {
		t.Errorf("created banner %+v", body["banner"])
	}
	sent := lastNewsBody(t, up)
	if sent["banner_hash"] != hash {
		t.Errorf("upstream create %v", sent)
	}

	resp, body = f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", keyUUID(31), newsCreateBody())
	if raw, present := body["banner"]; resp.StatusCode != http.StatusCreated || !present || raw != nil {
		t.Errorf("no banner is null, never absent: %d %+v", resp.StatusCode, body)
	}
	if _, has := lastNewsBody(t, up)["banner_hash"]; has {
		t.Errorf("a create without a banner must not send banner_hash")
	}

	_, body = f.subCall(t, http.MethodGet, "/api/v1/me/news-submissions/"+id, newsSubItem, "sess-alice", "", nil)
	if got, _ := body["banner"].(map[string]any); got["hash"] != hash {
		t.Errorf("get banner %+v", body["banner"])
	}
	_, body = f.subCall(t, http.MethodGet, "/api/v1/me/news-submissions?limit=2", newsSubList, "sess-alice", "", nil)
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("list %+v", body)
	}
	if first, _ := items[0].(map[string]any); first["banner"] != nil {
		t.Errorf("newest has no banner: %+v", first["banner"])
	}
	if second, _ := items[1].(map[string]any)["banner"].(map[string]any); second["hash"] != hash {
		t.Errorf("list banner %+v", items[1])
	}

	resp, body = f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/"+id, newsSubItem, "sess-alice", "", map[string]any{"banner_image_hash": ""})
	if resp.StatusCode != http.StatusOK || body["banner"] != nil {
		t.Errorf("clear %d %+v", resp.StatusCode, body)
	}
	sent = lastNewsBody(t, up)
	if v, has := sent["banner_hash"]; !has || v != "" || len(sent) != 1 {
		t.Errorf("clearing sends banner_hash \"\" and nothing else: %v", sent)
	}

	resp, body = f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/"+id, newsSubItem, "sess-alice", "", map[string]any{"banner_image_hash": hash})
	if got, _ := body["banner"].(map[string]any); resp.StatusCode != http.StatusOK || got["hash"] != hash {
		t.Errorf("set again %d %+v", resp.StatusCode, body)
	}

	before := len(up.calls())
	resp, body = f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/"+id, newsSubItem, "sess-alice", "", map[string]any{"state": "withdrawn", "banner_image_hash": ""})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if len(up.calls()) != before {
		t.Errorf("withdrawing together with a banner edit must not reach upstream")
	}
}

func TestV1NewsSubmissionBannerMustComeFromTheNewsUpload(t *testing.T) {
	f, _ := newNewsSubmissionFix(t, nil)
	foreign := newsImageHash([]byte("uploaded with the forum's own image client"))

	body := newsCreateBody()
	body["banner_image_hash"] = foreign
	resp, got := f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", keyUUID(40), body)
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := firstError(t, got); e["pointer"] != "/banner_image_hash" || e["reason"] != problem.ReasonUnknownReference {
		t.Errorf("create %+v", e)
	}

	resp, got = f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "sess-alice", "", map[string]any{"banner_image_hash": foreign})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := firstError(t, got); e["pointer"] != "/banner_image_hash" || e["reason"] != problem.ReasonUnknownReference {
		t.Errorf("patch %+v", e)
	}

	body["banner_image_hash"] = "not-a-hash"
	resp, got = f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", keyUUID(41), body)
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := firstError(t, got); e["pointer"] != "/banner_image_hash" {
		t.Errorf("malformed %+v", e)
	}
}

func lastNewsBody(t *testing.T, up *fakeMeNews) map[string]any {
	t.Helper()
	calls := up.calls()
	for i := len(calls) - 1; i >= 0; i-- {
		if calls[i].Method == http.MethodGet || calls[i].Path == "/v2/me/news-images" {
			continue
		}
		var out map[string]any
		if err := json.Unmarshal(calls[i].Raw, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	t.Fatal("no write reached upstream")
	return nil
}

func TestV1NewsSubmissionImageOversizedBodyIs413(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	resp, got := f.postHeadersOnly(t, newsSubImages, "multipart/form-data; boundary=x", 11*1024*1024)
	if body := problemMap(t, got); resp.StatusCode != http.StatusRequestEntityTooLarge || body["code"] != problem.CodePayloadTooLarge {
		t.Errorf("11 MiB body: %d %+v", resp.StatusCode, body)
	}
	if n := len(up.calls()); n != 0 {
		t.Errorf("an oversized body reached upstream %d times", n)
	}
}
