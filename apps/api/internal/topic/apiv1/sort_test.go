package apiv1

import (
	"strings"
	"testing"
)

func TestTopicLifecycle(t *testing.T) {
	state, by, err := topicLifecycle(0, "author")
	if err != nil || state != "published" || by != nil {
		t.Fatalf("published: %q %v %v", state, by, err)
	}
	state, by, err = topicLifecycle(1, "moderator")
	if err != nil || state != "hidden" || by == nil || *by != "moderator" {
		t.Fatalf("hidden: %q %v %v", state, by, err)
	}
	if _, _, err := topicLifecycle(1, ""); err == nil {
		t.Fatal("hidden with empty hidden_by must be a defect")
	}
	if _, _, err := topicLifecycle(1, "spam"); err == nil {
		t.Fatal("hidden with unknown hidden_by must be a defect")
	}
	if _, _, err := topicLifecycle(2, "author"); err == nil {
		t.Fatal("status 2 must be a defect")
	}
}

func TestLikesSortTokenMapsToLikeCount(t *testing.T) {
	spec, ok := lookupSort("likes_desc")
	if !ok {
		t.Fatal("likes_desc missing")
	}
	if spec.Key != "like_count" {
		t.Fatalf("likes_desc key %q, want like_count", spec.Key)
	}
	fav, ok := lookupSort("favorites_desc")
	if !ok || fav.Key != "favorite_count" {
		t.Fatalf("favorites_desc key %+v ok=%v", fav, ok)
	}
}

func TestFingerprintIncludesNSFWAndAuth(t *testing.T) {
	base := listFingerprint("bumped_desc", "", "", false, false)
	if listFingerprint("bumped_desc", "", "", true, false) == base {
		t.Fatal("include_nsfw must affect the cursor fingerprint")
	}
	if listFingerprint("bumped_desc", "", "", false, true) == base {
		t.Fatal("signed-in vs anonymous must affect the cursor fingerprint")
	}
	if listFingerprint("bumped_desc", "galgame", "", false, false) == base {
		t.Fatal("category must affect the cursor fingerprint")
	}
	if listFingerprint("created_desc", "", "", false, false) == base {
		t.Fatal("sort must affect the cursor fingerprint")
	}
}

func TestSortTableHasEighteenTokens(t *testing.T) {
	if n := len(sortSpecs); n != 18 {
		t.Fatalf("sort tokens %d, want 18", n)
	}
	if _, ok := lookupSort(""); !ok {
		t.Fatal("empty token should default")
	}
	def, _ := lookupSort("")
	if def.Token != defaultSort || def.Key != "status_update_time" || def.Direction != "desc" {
		t.Fatalf("default %+v", def)
	}
	if _, ok := lookupSort("hot"); ok {
		t.Fatal("unknown token must not resolve")
	}
}

func TestSectionEnumHasTwentySeven(t *testing.T) {
	parts := strings.Split(sectionEnum, ",")
	if len(parts) != 27 {
		t.Fatalf("sections %d, want 27", len(parts))
	}
}
