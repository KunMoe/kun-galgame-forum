package app

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestV1UserRepliesAuthoredWalk(t *testing.T) {
	f := newU3aFix(t)
	want := f.u3aIDs(t, `SELECT topic_reply.id::text FROM topic_reply
		JOIN topic ON topic.id = topic_reply.topic_id
		WHERE topic_reply.user_id = ? AND topic_reply.status = 0 AND `+sharedTopicSQL(false)+`
		ORDER BY topic_reply.created DESC, topic_reply.id DESC`, u3aOwner)
	if len(want) < 5 {
		t.Fatalf("authored replies seed too thin: %v", want)
	}
	got, total := f.walkUserList(t, "", "replies", "authored")
	if total != len(want) || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("walked %v total %d, want %v", got, total, want)
	}
}

func TestV1UserRepliesOmitsHiddenParent(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "replies", "authored", "")
	if listContains(body, u3aReplyOnHidden) || listContains(body, u3aReplyHiddenStatus) {
		t.Fatalf("hidden parent or hidden reply leaked: %v", itemIDs(t, body))
	}
	received := f.userListOK(t, "", "replies", "received", "")
	if listContains(received, u3aReplyPeerOnHidden) {
		t.Fatalf("received included a reply on a hidden topic: %v", itemIDs(t, received))
	}
	if !listContains(received, u3aReplyPeerPub) || !listContains(received, u3aReplySelf) {
		t.Fatalf("received missed a visible peer or self reply: %v", itemIDs(t, received))
	}
}

func TestV1UserRepliesLikedOmitsHiddenParent(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "replies", "liked", "")
	if listContains(body, u3aReplyLikedHidden) {
		t.Fatalf("liked included a reply on a hidden topic: %v", itemIDs(t, body))
	}
	if !listContains(body, u3aReplyLiked) {
		t.Fatalf("liked missed the visible reply: %v", itemIDs(t, body))
	}
}

func TestV1UserRepliesExcerptTruncated(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "replies", "authored", "")
	ex, _ := listItem(t, body, u3aReplyLong)["excerpt"].(string)
	if utf8.RuneCountInString(ex) != 200 || !strings.HasSuffix(ex, "…") {
		t.Fatalf("long excerpt %q (%d runes)", ex, utf8.RuneCountInString(ex))
	}
	if ex != strings.Repeat("w", 199)+"…" {
		t.Fatalf("long excerpt %q", ex)
	}
	liked := f.userListOK(t, "", "replies", "liked", "")
	if got := listItem(t, liked, u3aReplyLiked)["excerpt"]; got != "liked-pub" {
		t.Fatalf("markdown excerpt %v, want liked-pub", got)
	}
}

func TestV1UserRepliesItemShape(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "replies", "authored", "")
	it := listItem(t, body, u3aReplyTieMax)
	if it["object"] != "reply" || strID(it["topic_id"]) != strconv.Itoa(u3aTopicTieMin) {
		t.Fatalf("item %+v", it)
	}
	if asInt(it["floor"]) != 5 {
		t.Fatalf("floor %v", it["floor"])
	}
	if it["created_at"] != "2026-08-02T15:00:00Z" {
		t.Fatalf("created_at %v", it["created_at"])
	}
}

func TestV1UserRepliesNSFWDefaultExcluded(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "replies", "authored", "")
	if listContains(body, u3aReplyOnNSFW) {
		t.Fatalf("default authored replies included NSFW parent: %v", itemIDs(t, body))
	}
	with := f.userListOK(t, "", "replies", "authored", "&include_nsfw=true")
	if !listContains(with, u3aReplyOnNSFW) {
		t.Fatalf("include_nsfw=true missed NSFW parent reply: %v", itemIDs(t, with))
	}
}

func TestV1UserRepliesModeratorDoesNotSeeRestrictedParent(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "sess-staff", "replies", "authored", "")
	if listContains(body, u3aReplyOnUsers) {
		t.Fatalf("topic.view_restricted widened authored replies: %v", itemIDs(t, body))
	}
}
