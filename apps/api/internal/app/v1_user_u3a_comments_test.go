package app

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestV1UserCommentsAuthoredWalk(t *testing.T) {
	f := newU3aFix(t)
	want := f.u3aIDs(t, `SELECT topic_comment.id::text FROM topic_comment
		JOIN topic ON topic.id = topic_comment.topic_id
		WHERE topic_comment.user_id = ? AND topic_comment.status = 0 AND `+sharedTopicSQL(false)+`
		ORDER BY topic_comment.created DESC, topic_comment.id DESC`, u3aOwner)
	if len(want) < 5 {
		t.Fatalf("authored comments seed too thin: %v", want)
	}
	got, total := f.walkUserList(t, "", "comments", "authored")
	if total != len(want) || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("walked %v total %d, want %v", got, total, want)
	}
}

func TestV1UserCommentsOmitsRestrictedParent(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "comments", "authored", "")
	if listContains(body, u3aCommOnUsers) || listContains(body, u3aCommOnHidden) || listContains(body, u3aCommHiddenStatus) {
		t.Fatalf("restricted/hidden parent or hidden comment leaked: %v", itemIDs(t, body))
	}
	received := f.userListOK(t, "", "comments", "received", "")
	if listContains(received, u3aCommReceivedUsers) || listContains(received, u3aCommReceivedHidden) {
		t.Fatalf("received included a comment on a hidden or restricted topic: %v", itemIDs(t, received))
	}
	if !listContains(received, u3aCommReceived) {
		t.Fatalf("received missed the visible comment: %v", itemIDs(t, received))
	}
}

func TestV1UserCommentsLikedOmitsHiddenParent(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "comments", "liked", "")
	if listContains(body, u3aCommReceivedHidden) {
		t.Fatalf("liked included a comment on a hidden topic: %v", itemIDs(t, body))
	}
	if !listContains(body, u3aCommLiked) {
		t.Fatalf("liked missed the visible comment: %v", itemIDs(t, body))
	}
}

func TestV1UserCommentsExcerptTruncated(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "comments", "authored", "")
	ex, _ := listItem(t, body, u3aCommLong)["excerpt"].(string)
	if utf8.RuneCountInString(ex) != 200 || !strings.HasSuffix(ex, "…") {
		t.Fatalf("long excerpt %q (%d runes)", ex, utf8.RuneCountInString(ex))
	}
	if ex != strings.Repeat("z", 199)+"…" {
		t.Fatalf("long excerpt %q", ex)
	}
}

func TestV1UserCommentsItemShape(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "comments", "authored", "")
	it := listItem(t, body, u3aCommTieMax)
	if it["object"] != "comment" || strID(it["topic_id"]) != strconv.Itoa(u3aTopicTieMin) {
		t.Fatalf("item %+v", it)
	}
	if it["excerpt"] != "ctie-"+strconv.Itoa(u3aCommTieMax) {
		t.Fatalf("excerpt %v", it["excerpt"])
	}
	if it["created_at"] != "2026-08-02T15:00:00Z" {
		t.Fatalf("created_at %v", it["created_at"])
	}
}

func TestV1UserCommentsNSFWDefaultExcluded(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "", "comments", "authored", "")
	if listContains(body, u3aCommOnNSFW) {
		t.Fatalf("default authored comments included NSFW parent: %v", itemIDs(t, body))
	}
	with := f.userListOK(t, "", "comments", "authored", "&include_nsfw=true")
	if !listContains(with, u3aCommOnNSFW) {
		t.Fatalf("include_nsfw=true missed NSFW parent comment: %v", itemIDs(t, with))
	}
}

func TestV1UserCommentsModeratorDoesNotSeeRestrictedParent(t *testing.T) {
	f := newU3aFix(t)
	body := f.userListOK(t, "sess-staff", "comments", "authored", "")
	if listContains(body, u3aCommOnUsers) {
		t.Fatalf("topic.view_restricted widened authored comments: %v", itemIDs(t, body))
	}
	received := f.userListOK(t, "sess-staff", "comments", "received", "")
	if listContains(received, u3aCommReceivedUsers) {
		t.Fatalf("topic.view_restricted widened received comments: %v", itemIDs(t, received))
	}
}
