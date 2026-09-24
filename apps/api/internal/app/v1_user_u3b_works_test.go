package app

import (
	"fmt"
	"net/http"
	"testing"
)

func TestV1UserWorksPublishedWalk(t *testing.T) {
	f := newU3bFix(t)
	want := f.sqlIDs(t, `SELECT galgame.id::text FROM galgame
		WHERE galgame.creator_user_id = ? AND galgame.published
		  AND galgame.id BETWEEN ? AND ?
		  AND galgame.content_limit = 'sfw'
		ORDER BY galgame.created DESC, galgame.id DESC`, u3bOwner, u3bWorkMin, u3bWorkMax)
	if len(want) < 5 {
		t.Fatalf("published seed too thin: %v", want)
	}
	got, total := f.walkU3b(t, "works", "published", "")
	if total != len(want) || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("walked %v total %d, want %v", got, total, want)
	}
}

func TestV1UserWorksLikedWalk(t *testing.T) {
	f := newU3bFix(t)
	want := f.sqlIDs(t, `SELECT galgame.id::text FROM galgame
		JOIN galgame_like ON galgame_like.work_id = galgame.id
		WHERE galgame_like.user_id = ? AND galgame.published
		  AND galgame.id BETWEEN ? AND ?
		  AND galgame.content_limit = 'sfw'
		ORDER BY galgame.created DESC, galgame.id DESC`, u3bOwner, u3bWorkMin, u3bWorkMax)
	if len(want) < 4 {
		t.Fatalf("liked seed too thin: %v", want)
	}
	got, total := f.walkU3b(t, "works", "liked", "")
	if total != len(want) || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("walked %v total %d, want %v", got, total, want)
	}
}

func TestV1UserWorksPublishedOmitsUnpublished(t *testing.T) {
	f := newU3bFix(t)
	body := f.u3bListOK(t, "works", "published", "")
	if listContains(body, u3bWorkUnpub) {
		t.Fatalf("unpublished work leaked: %v", itemIDs(t, body))
	}
	if !listContains(body, u3bWorkTieMax) {
		t.Fatalf("published work missing: %v", itemIDs(t, body))
	}
}

func TestV1UserWorksNSFWTotalMatchesWalk(t *testing.T) {
	f := newU3bFix(t)
	body := f.u3bListOK(t, "works", "published", "")
	if listContains(body, u3bWorkNSFW) {
		t.Fatalf("default published included NSFW: %v", itemIDs(t, body))
	}
	got, total := f.walkU3b(t, "works", "published", "")
	if total != len(got) {
		t.Fatalf("total %d walked %d", total, len(got))
	}
	with := f.u3bListOK(t, "works", "published", "&include_nsfw=true")
	if !listContains(with, u3bWorkNSFW) {
		t.Fatalf("include_nsfw=true missed NSFW: %v", itemIDs(t, with))
	}
}

func TestV1UserWorksContributedFiltersBeforePage(t *testing.T) {
	f := newU3bFix(t)
	resp, body := f.u3bList(t, "", "works", "relation=contributed&page=1&limit=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("contributed %d %+v", resp.StatusCode, body)
	}
	if listContains(body, u3bWorkNSFW) || listContains(body, u3bWorkGhostNS) {
		t.Fatalf("contributed default included NSFW: %v", itemIDs(t, body))
	}
	got := itemIDs(t, body)
	wantPage := []string{strconvI(u3bWorkTieMax), strconvI(u3bWorkNoLocal)}
	if fmt.Sprint(got) != fmt.Sprint(wantPage) {
		t.Fatalf("page %v, want %v", got, wantPage)
	}
	if asInt(body["total"]) != 3 {
		t.Fatalf("total %v, want 3", body["total"])
	}
	walked, total := f.walkU3b(t, "works", "contributed", "")
	want := []string{strconvI(u3bWorkTieMax), strconvI(u3bWorkNoLocal), strconvI(u3bWorkTieMin)}
	if total != 3 || fmt.Sprint(walked) != fmt.Sprint(want) {
		t.Fatalf("walked %v total %d, want %v", walked, total, want)
	}
}

func TestV1UserWorksContributedIncludeNSFW(t *testing.T) {
	f := newU3bFix(t)
	body := f.u3bListOK(t, "works", "contributed", "&include_nsfw=true")
	if !listContains(body, u3bWorkNSFW) || !listContains(body, u3bWorkGhostNS) || asInt(body["total"]) != 5 {
		t.Fatalf("include_nsfw contributed %v total %v", itemIDs(t, body), body["total"])
	}
}

func strconvI(n int) string { return fmt.Sprintf("%d", n) }
