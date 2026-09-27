package repr

import (
	"encoding/json"
	"testing"

	"kun-galgame-api/pkg/userclient"
)

func TestNewUserRefAvatarFromHash(t *testing.T) {
	u := userclient.User{ID: 3, Name: "alice", AvatarImageHash: testHash, Avatar: "https://ignored.example/a.png"}
	ref := NewUserRef("https://cdn", u)
	if ref.Object != "user" || string(ref.ID) != "3" || ref.Name == nil || *ref.Name != "alice" {
		t.Errorf("ref %+v", ref)
	}
	if ref.Avatar == nil || ref.Avatar.Hash != testHash {
		t.Errorf("avatar %+v", ref.Avatar)
	}
	if ref.Avatar.Width != nil || ref.Avatar.Sexual != nil {
		t.Errorf("hash-only avatar must have null meta: %+v", ref.Avatar)
	}
}

func TestNewUserRefExternalURLIsNullAvatar(t *testing.T) {
	u := userclient.User{ID: 8, Name: "hotlink", Avatar: "https://i0.hdslb.com/face.png", AvatarImageHash: ""}
	ref := NewUserRef("https://cdn", u)
	if ref.Avatar != nil {
		t.Errorf("external URL avatar = %+v, want null", ref.Avatar)
	}
	raw, _ := json.Marshal(ref)
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["avatar"] != nil {
		t.Errorf("avatar marshalled %s", raw)
	}
}

func TestDeletedUserRefHasANullName(t *testing.T) {
	raw, err := json.Marshal(DeletedUserRef(99))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"object":"user","id":"99","name":null,"avatar":null,"avatar_frame":null}` {
		t.Errorf("deleted user = %s", raw)
	}
}

func TestNewUserRefAvatarFrame(t *testing.T) {
	frame := func(d *userclient.Decoration) map[string]any {
		t.Helper()
		raw, err := json.Marshal(NewUserRef("https://cdn", userclient.User{ID: 5, Name: "f", Cosmetics: userclient.Cosmetics{AvatarFrame: d}}))
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		v, ok := m["avatar_frame"]
		if !ok {
			t.Fatalf("avatar_frame missing from %s", raw)
		}
		f, _ := v.(map[string]any)
		return f
	}

	both := frame(&userclient.Decoration{StaticURL: "https://img.example/f.png", AnimatedURL: "https://img.example/f.webp"})
	if both["static_url"] != "https://img.example/f.png" || both["animated_url"] != "https://img.example/f.webp" {
		t.Errorf("frame with both images = %v", both)
	}

	still := frame(&userclient.Decoration{StaticURL: "https://img.example/f.png"})
	if v, ok := still["animated_url"]; !ok || v != nil || still["static_url"] != "https://img.example/f.png" {
		t.Errorf("a still-only frame must carry animated_url null: %v", still)
	}

	if f := frame(&userclient.Decoration{AnimatedURL: "https://img.example/f.webp"}); f != nil {
		t.Errorf("a frame without a still image = %v, want null", f)
	}
	if f := frame(nil); f != nil {
		t.Errorf("no frame = %v, want null", f)
	}
}
