package app

import (
	"net/http"
	"strconv"
	"testing"
)

const afUserFramed = 930000813

func afCosmetics(static, animated string) map[string]any {
	frame := map[string]any{"item_id": 3, "name": "sakura", "static_url": static}
	if animated != "" {
		frame["animated_url"] = animated
	}
	return map[string]any{"cosmetics": map[string]any{
		"profile_background": map[string]any{"item_id": 9, "name": "stars", "static_url": "https://img.example/decorations/bg.jpg"},
		"avatar_frame":       frame,
	}}
}

func TestV1ListUsersCarriesAvatarFrame(t *testing.T) {
	f := newMeFix(t)
	f.addOAuthUser(afUserFramed, "framed", 0, afCosmetics("https://img.example/decorations/f.png", "https://img.example/decorations/f.webp"))
	framed, bob := strconv.Itoa(afUserFramed), strconv.Itoa(w3UserBob)
	resp, body := f.listUsersQuery(t, "sess-alice", "ids="+framed+","+bob)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list users %d %+v", resp.StatusCode, body)
	}
	items := itemsOf(body)
	if len(items) != 2 || items[0]["id"] != framed || items[1]["id"] != bob {
		t.Fatalf("items %+v", items)
	}
	frame, _ := items[0]["avatar_frame"].(map[string]any)
	if frame["static_url"] != "https://img.example/decorations/f.png" || frame["animated_url"] != "https://img.example/decorations/f.webp" {
		t.Fatalf("framed avatar_frame %+v", items[0]["avatar_frame"])
	}
	if v, ok := items[1]["avatar_frame"]; !ok || v != nil {
		t.Fatalf("bob avatar_frame %v (present %v), want null", v, ok)
	}
}

func TestV1GetUserAvatarFrame(t *testing.T) {
	f := newMeFix(t)
	f.addOAuthUser(afUserFramed, "framed", 0, afCosmetics("https://img.example/decorations/f.png", ""))
	resp, body := f.getUser(t, "", strconv.Itoa(afUserFramed))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get user %d %+v", resp.StatusCode, body)
	}
	frame, _ := body["avatar_frame"].(map[string]any)
	if frame["static_url"] != "https://img.example/decorations/f.png" {
		t.Fatalf("avatar_frame %+v", body["avatar_frame"])
	}
	if v, ok := frame["animated_url"]; !ok || v != nil {
		t.Fatalf("animated_url %v (present %v), want null for a still-only frame", v, ok)
	}
}
