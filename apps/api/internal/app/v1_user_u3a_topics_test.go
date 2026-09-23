package app

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

func TestV1UserTopicsAuthoredWalk(t *testing.T) {
	f := newU3aFix(t)
	want := f.u3aIDs(t, `SELECT topic.id::text FROM topic WHERE topic.user_id = ? AND `+sharedTopicSQL(false)+`
		ORDER BY topic.created DESC, topic.id DESC`, u3aOwner)
	if len(want) < 5 {
		t.Fatalf("authored seed too thin: %v", want)
	}
	got, total := f.walkUserList(t, "", "topics", "authored")
	if total != len(want) || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("walked %v total %d, want %v", got, total, want)
	}
}

func TestV1UserTopicsLikedWalk(t *testing.T) {
	f := newU3aFix(t)
	want := f.u3aIDs(t, `SELECT topic.id::text FROM topic
		JOIN topic_reaction ON topic_reaction.topic_id = topic.id AND topic_reaction.reaction = 'like'
		WHERE topic_reaction.user_id = ? AND `+sharedTopicSQL(false)+`
		ORDER BY topic.created DESC, topic.id DESC`, u3aOwner)
	if len(want) < 4 {
		t.Fatalf("liked seed too thin: %v", want)
	}
	got, total := f.walkUserList(t, "", "topics", "liked")
	if total != len(want) || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("walked %v total %d, want %v", got, total, want)
	}
}

func TestV1UserTopicsAuthoredOmitsHidden(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "topics", "authored", "")
	if listContains(body, u3aTopicHidden) || listContains(body, u3aTopicHidUsers) {
		t.Fatalf("hidden topics leaked in authored: %v", itemIDs(t, body))
	}
}

func TestV1UserTopicsLikedOmitsRestricted(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "topics", "liked", "")
	if listContains(body, u3aTopicUsers) || listContains(body, u3aPeerUsers) || listContains(body, u3aTopicRole) ||
		listContains(body, u3aTopicHidden) || listContains(body, u3aPeerHidden) {
		t.Fatalf("hidden or restricted topics leaked in liked: %v", itemIDs(t, body))
	}
	if !listContains(body, u3aPeerPub) || !listContains(body, u3aTopicTieMax) {
		t.Fatalf("visible likes missing: %v", itemIDs(t, body))
	}
}

func TestV1UserTopicsModeratorDoesNotSeeRestricted(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "sess-staff", "topics", "authored", "")
	if listContains(body, u3aTopicUsers) || listContains(body, u3aTopicRole) {
		t.Fatalf("topic.view_restricted widened authored: %v", itemIDs(t, body))
	}
	liked := f.userListOK(t, "sess-staff", "topics", "liked", "")
	if listContains(liked, u3aTopicUsers) || listContains(liked, u3aPeerUsers) {
		t.Fatalf("topic.view_restricted widened liked: %v", itemIDs(t, liked))
	}
}

func TestV1UserTopicsHiddenForbidden(t *testing.T) {
	f := newU3aFix(t)
	owner := strconv.Itoa(u3aOwner)
	for _, session := range []string{"", "sess-bob"} {
		resp, body := f.userList(t, session, "topics", owner, "relation=hidden")
		mustCode(t, resp, body, http.StatusForbidden, "PERMISSION_REQUIRED")
	}
}

func TestV1UserTopicsHiddenAllowsViewHidden(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "sess-staff", "topics", "hidden", "")
	if !listContains(body, u3aTopicHidden) || !listContains(body, u3aTopicHidUsers) {
		t.Fatalf("staff hidden list %v, want public-hidden and users-hidden", itemIDs(t, body))
	}
	own := f.userListOK(t, "sess-owner", "topics", "hidden", "")
	if !listContains(own, u3aTopicHidden) || !listContains(own, u3aTopicHidUsers) {
		t.Fatalf("owner hidden list %v", itemIDs(t, own))
	}
}

func TestV1UserTopicsNSFWDefaultExcluded(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "topics", "authored", "")
	if listContains(body, u3aTopicNSFW) {
		t.Fatalf("default authored included NSFW: %v", itemIDs(t, body))
	}
	with := f.userListOK(t, "", "topics", "authored", "&include_nsfw=true")
	if !listContains(with, u3aTopicNSFW) {
		t.Fatalf("include_nsfw=true missed NSFW: %v", itemIDs(t, with))
	}
}

func TestV1UserTopicsLoginScopeFollowsSharedList(t *testing.T) {
	f := newU3aFix(t)
	anon := f.userListOK(t, "", "topics", "authored", "")
	if listContains(anon, u3aTopicLogin) {
		t.Fatalf("anonymous authored included login-scoped: %v", itemIDs(t, anon))
	}
	auth := f.userListOK(t, "sess-bob", "topics", "authored", "")
	if !listContains(auth, u3aTopicLogin) {
		t.Fatalf("signed-in authored missed login-scoped: %v", itemIDs(t, auth))
	}
}

func TestV1UserTopicsUpvotedAndFavoritedOmitHidden(t *testing.T) {
	f := newU3aFix(t)
	up := f.userListOK(t, "", "topics", "upvoted", "")
	fav := f.userListOK(t, "", "topics", "favorited", "")
	if listContains(up, u3aTopicHidden) || listContains(up, u3aTopicUsers) {
		t.Fatalf("upvoted leaked hidden/restricted: %v", itemIDs(t, up))
	}
	if listContains(fav, u3aTopicHidden) || listContains(fav, u3aTopicNSFW) {
		t.Fatalf("favorited leaked hidden/NSFW: %v", itemIDs(t, fav))
	}
	if !listContains(fav, u3aPeerPub) || !listContains(up, u3aTopicTieMin+3) {
		t.Fatalf("visible upvotes/favorites missing: up %v fav %v", itemIDs(t, up), itemIDs(t, fav))
	}
}

func TestV1UserTopicsItemShape(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "topics", "authored", "")
	it := listItem(t, body, u3aTopicTieMax)
	if it["object"] != "topic" || strID(it["id"]) != strconv.Itoa(u3aTopicTieMax) {
		t.Fatalf("item %+v", it)
	}
	if it["title"] != "u3a-"+strconv.Itoa(u3aTopicTieMax) {
		t.Fatalf("title %v", it["title"])
	}
	if it["created_at"] != "2026-08-02T15:00:00Z" {
		t.Fatalf("created_at %v", it["created_at"])
	}
}
