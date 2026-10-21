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
	"strings"
	"testing"
)

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
