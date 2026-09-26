package app

import (
	"strconv"
	"testing"
)

func TestV1UserPostListsNameTopicAndAuthor(t *testing.T) {
	f := newU3aFix(t)
	f.addOAuthUser(u3aStatus2, "u3a-banned", 2, nil)
	f.execSQL(t, `INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'banned-reply', 11, ?, ?, 0, 0, now(), now())`, u3aReplyBanned, u3aStatus2, u3aTopicTieMin)
	f.execSQL(t, `INSERT INTO topic_comment
		(id, content, topic_id, topic_reply_id, user_id, target_user_id, parent_comment_id, status, created, updated)
		VALUES (?, 'banned-comment', ?, ?, ?, ?, NULL, 0, now(), now())`,
		u3aCommBanned, u3aTopicTieMin, u3aReplyTieMin, u3aStatus2, u3aOwner)

	for _, tc := range []struct {
		collection   string
		peer, banned int
	}{
		{"replies", u3aReplyPeerPub, u3aReplyBanned},
		{"comments", u3aCommReceived, u3aCommBanned},
	} {
		body := f.userListOK(t, "", tc.collection, "received", "")
		if listContains(body, tc.banned) {
			t.Fatalf("%s: a banned author's post is listed: %v", tc.collection, itemIDs(t, body))
		}
		it := listItem(t, body, tc.peer)
		author, _ := it["author"].(map[string]any)
		if it["topic_title"] != "u3a-"+strconv.Itoa(u3aTopicTieMin) ||
			author["object"] != "user" || author["id"] != strconv.Itoa(u3aPeer) || author["name"] != "u3a-peer" {
			t.Fatalf("%s item %+v", tc.collection, it)
		}
	}
}
