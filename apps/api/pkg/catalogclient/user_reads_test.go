package catalogclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func parseQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	q, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("query %q: %v", raw, err)
	}
	return q
}

func TestEditSnapshotUser_TravelsAsTheUser(t *testing.T) {
	srv, got := recordingServer(t, 0,
		`{"code":0,"message":"ok","data":{"values":{"catalog.work.name_zh_cn":"现值"}}}`)

	values, err := userClient(srv.URL).EditSnapshotUser(context.Background(), "user-jwt", "catalog.work", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodGet || got.path != "/v2/moderation/snapshots/work/1000" {
		t.Fatalf("snapshot hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if got.query != "" {
		t.Fatalf("snapshot query = %q, want the id on the path", got.query)
	}
	if values["catalog.work.name_zh_cn"] != "现值" {
		t.Fatalf("values decoded wrong: %v", values)
	}
}

func TestListEditProposalsUser_MineIsTheToken(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"items":[`+
		`{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"open","proposer_uid":9,"patch":{}}`+
		`],"total":1}}`)

	items, err := userClient(srv.URL).ListEditProposalsUser(context.Background(), "user-jwt",
		UserEditProposalFilter{EntityType: "catalog.work", EntityID: 1000, Status: "open", Limit: 20, Mine: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodGet || got.path != "/v2/me/proposals" {
		t.Fatalf("list hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	q := parseQuery(t, got.query)
	if q.Has("mine") {
		t.Fatalf("mine is the path, not a query flag: %q", got.query)
	}
	if q.Has("proposer_uid") || q.Has("site") {
		t.Fatalf("the user plane names neither a proposer nor a site: %q", got.query)
	}
	if q.Get("entity_id") != "1000" || q.Get("state") != "open" || q.Get("limit") != "20" {
		t.Fatalf("filter lost on the wire: %q", got.query)
	}
	if len(items) != 1 || items[0].ID != 7 {
		t.Fatalf("items decoded wrong: %+v", items)
	}
}

func TestListEditProposalsUser_V2OmitsPatchAndCarriesTimes(t *testing.T) {
	srv, _ := recordingServer(t, 0, `{"object":"list","items":[{`+
		`"id":"1064","state":"merged","target_object":"work","entity_id":"17",`+
		`"proposer_uid":"2","site":"kungal","created_at":"2026-08-15T11:35:34Z"`+
		`}]}`)

	items, err := userClient(srv.URL).ListEditProposalsUser(context.Background(), "user-jwt",
		UserEditProposalFilter{Mine: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items decoded wrong: %+v", items)
	}
	it := items[0]
	if it.Patch == nil {
		t.Fatal("omitted patch must decode to an empty map")
	}
	if it.Amendments == nil {
		t.Fatal("omitted amendments must decode to an empty slice")
	}
	if it.CreatedAt.UTC().Format("2006-01-02") != "2026-08-15" {
		t.Fatalf("created_at = %v, want 2026-08-15", it.CreatedAt)
	}
	if it.Status != "merged" || it.ID != 1064 || it.EntityType != "catalog.work" {
		t.Fatalf("decoded wrong: %+v", it)
	}
}

func TestListEditProposalsUser_QueueOmitsMine(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"items":[],"total":0}}`)

	if _, err := userClient(srv.URL).ListEditProposalsUser(context.Background(), "mod-jwt",
		UserEditProposalFilter{EntityType: "catalog.work", Status: "open"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parseQuery(t, got.query).Has("mine") {
		t.Fatalf("the queue read must not carry mine at all: %q", got.query)
	}
}

func TestListEditProposalsUser_QueueDenialStaysADenial(t *testing.T) {
	srv, _ := recordingServer(t, http.StatusForbidden,
		`{"code":233,"message":"permission denied: catalog.edit.review"}`)

	_, err := userClient(srv.URL).ListEditProposalsUser(context.Background(), "user-jwt",
		UserEditProposalFilter{EntityType: "catalog.work"})
	if errors.Is(err, ErrInsufficientScope) {
		t.Fatal("a permission denial must not be reported as a scope denial")
	}
	var apiErr *UserAPIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden {
		t.Fatalf("queue denial = %v, want a 403 UserAPIError", err)
	}
}

func TestWorkCoversUser_BallotFromTheToken(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"covers":[`+
		`{"id":88,"image_hash":"abc","vote_count":3,"voted":true}]}}`)

	covers, err := userClient(srv.URL).WorkCoversUser(context.Background(), "user-jwt", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodGet || got.path != "/v2/catalog/works/1000/covers" {
		t.Fatalf("covers hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if strings.Contains(got.query, "uid") || strings.Contains(got.query, "user") {
		t.Fatalf("the viewer is the token, not a query parameter: %q", got.query)
	}
	if !strings.Contains(got.query, "nsfw=true") {
		t.Fatalf("query = %q, want the population gate open — without it every r18 work is a 404", got.query)
	}
	if len(covers) != 1 || covers[0].ID != 88 || !covers[0].Voted || covers[0].VoteCount != 3 {
		t.Fatalf("covers decoded wrong: %+v", covers)
	}
}

func TestUploadEditImageUser_SendsNoActorUID(t *testing.T) {
	var (
		gotPath   string
		gotAuth   string
		gotFields = map[string]string{}
		gotFile   []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Errorf("content type: %v", err)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			raw, _ := io.ReadAll(part)
			if part.FileName() != "" {
				gotFile = raw
				continue
			}
			gotFields[part.FormName()] = string(raw)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"object":"edit_image","hash":"h1","url":"https://cdn/image/h1","width":1920,"height":1080,"size_bytes":4,"is_deduplicated":false}`))
	}))
	t.Cleanup(srv.Close)

	res, err := userClient(srv.URL).UploadEditImageUser(context.Background(), "user-jwt",
		bytes.NewReader([]byte("PNG!")), "cover.png", "cover")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/v2/me/edit-images" {
		t.Fatalf("upload hit %s", gotPath)
	}
	if gotAuth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", gotAuth)
	}
	if strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatal("the user plane must never attach the client credential")
	}
	if _, ok := gotFields["actor_uid"]; ok {
		t.Fatalf("the upload asserts no actor: %v", gotFields)
	}
	if gotFields["preset"] != "cover" || string(gotFile) != "PNG!" {
		t.Fatalf("upload body wrong: fields %v file %q", gotFields, gotFile)
	}
	if res.Hash != "h1" || res.Width != 1920 {
		t.Fatalf("upload result decoded wrong: %+v", res)
	}
}

func TestUploadEditImageUser_ScopeDenial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":233,"message":"missing required scope: catalog:edit"}`))
	}))
	t.Cleanup(srv.Close)

	_, err := userClient(srv.URL).UploadEditImageUser(context.Background(), "old-jwt",
		bytes.NewReader([]byte("x")), "cover.png", "cover")
	if !errors.Is(err, ErrInsufficientScope) {
		t.Fatalf("upload scope denial = %v, want ErrInsufficientScope", err)
	}
}

func TestUploadEditImageUser_RefusesAnEmptyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("an empty token must not reach the catalog")
	}))
	t.Cleanup(srv.Close)

	_, err := userClient(srv.URL).UploadEditImageUser(context.Background(), "",
		bytes.NewReader([]byte("x")), "cover.png", "cover")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("empty token = %v, want ErrUnauthorized", err)
	}
}
