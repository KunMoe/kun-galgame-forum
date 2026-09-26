package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"kun-galgame-api/internal/infrastructure/markdown"
)

func legacySticker(t *testing.T, position int) string {
	t.Helper()
	h, ok := markdown.LegacyStickerHash(1, position)
	if !ok {
		t.Fatalf("sticker 1/%d", position)
	}
	return h
}

func coverHashes(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		h, _ := m["hash"].(string)
		out = append(out, h)
	}
	return out
}

func imageNodes(v any) []map[string]any {
	var out []map[string]any
	switch n := v.(type) {
	case map[string]any:
		if n["object"] == "image" {
			out = append(out, n)
		}
		for _, child := range n {
			out = append(out, imageNodes(child)...)
		}
	case []any:
		for _, child := range n {
			out = append(out, imageNodes(child)...)
		}
	}
	return out
}

func TestV1StickerOnlyTopicHasNoCover(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	sticker := legacySticker(t, 1)
	body := map[string]any{
		"title":            "moon cakes",
		"content_markdown": `![鲲 Galgame 表情包 \[1\] - 1](/image/` + sticker + `_320 "鲲 Galgame 表情包 \[1] - 1") and /image/` + w3CoverHash,
		"category":         "galgame",
		"sections":         []string{"g-news"},
		"is_nsfw":          false,
		"access_scope":     "public",
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(1), nil, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d %s", resp.StatusCode, raw)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if got := coverHashes(out["cover_images"]); !slices.Equal(got, []string{w3CoverHash}) {
		t.Fatalf("derived covers %v, want only the upload", got)
	}

	resp, raw = f.doJSON(t, http.MethodGet, "/api/v1/topics/"+strID(out["id"]), "sess-alice", "/topics/{topic_id}", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail status %d %s", resp.StatusCode, raw)
	}
	var detail map[string]any
	if err := json.Unmarshal(raw, &detail); err != nil {
		t.Fatal(err)
	}
	imgs := imageNodes(detail["content"])
	if len(imgs) != 1 || imgs[0]["is_sticker"] != true {
		t.Fatalf("image nodes %v, want the sticker flagged", imgs)
	}
}

func TestV1StoredStickerCoversAreSkippedOnRead(t *testing.T) {
	f := newWriteFix(t, nil)
	stored := fmt.Sprintf(`["/image/%s","/image/%s","/image/%s"]`, legacySticker(t, 2), w3CoverHash, legacySticker(t, 3))
	if err := f.db.Exec(`UPDATE topic SET cover_images = ? WHERE id = ?`, stored, w3TopicPub).Error; err != nil {
		t.Fatal(err)
	}

	resp, raw := f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d", w3TopicPub), "sess-other", "/topics/{topic_id}", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail status %d %s", resp.StatusCode, raw)
	}
	var detail map[string]any
	if err := json.Unmarshal(raw, &detail); err != nil {
		t.Fatal(err)
	}
	if got := coverHashes(detail["cover_images"]); !slices.Equal(got, []string{w3CoverHash}) {
		t.Fatalf("detail covers %v", got)
	}

	resp, raw = f.doJSON(t, http.MethodGet, "/api/v1/topics?limit=100", "sess-other", "/topics", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status %d %s", resp.StatusCode, raw)
	}
	var list struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	for _, it := range list.Items {
		if strID(it["id"]) == fmt.Sprint(w3TopicPub) {
			if got := coverHashes(it["cover_images"]); !slices.Equal(got, []string{w3CoverHash}) {
				t.Fatalf("summary covers %v", got)
			}
			return
		}
	}
	t.Fatalf("topic %d missing from the list", w3TopicPub)
}
