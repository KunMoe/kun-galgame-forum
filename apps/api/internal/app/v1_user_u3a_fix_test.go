package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	topicRepo "kun-galgame-api/internal/topic/repository"
)

const (
	u3aUserMin = 970000001
	u3aUserMax = 970000099
	u3aOwner   = 970000001
	u3aPeer    = 970000002
	u3aStatus2 = 970000003
	u3aGone    = 970000099

	u3aTopicMin = 970000201
	u3aTopicMax = 970000299
	u3aReplyMin = 970000301
	u3aReplyMax = 970000399
	u3aCommMin  = 970000401
	u3aCommMax  = 970000499

	u3aTopicTieMin   = 970000211
	u3aTopicTieMax   = 970000215
	u3aTopicHidden   = 970000221
	u3aTopicHidUsers = 970000222
	u3aTopicUsers    = 970000223
	u3aTopicRole     = 970000224
	u3aTopicNSFW     = 970000225
	u3aTopicLogin    = 970000226
	u3aPeerPub       = 970000231
	u3aPeerHidden    = 970000232
	u3aPeerUsers     = 970000233

	u3aReplyTieMin       = 970000311
	u3aReplyTieMax       = 970000315
	u3aReplyPeerPub      = 970000316
	u3aReplySelf         = 970000317
	u3aReplyHiddenStatus = 970000318
	u3aReplyLong         = 970000319
	u3aReplyLiked        = 970000320
	u3aReplyOnHidden     = 970000321
	u3aReplyPeerOnHidden = 970000322
	u3aReplyLikedHidden  = 970000323
	u3aReplyOnUsers      = 970000324
	u3aReplyPeerOnUsers  = 970000325
	u3aReplyOnNSFW       = 970000326
	u3aReplyBanned       = 970000327

	u3aCommTieMin         = 970000411
	u3aCommTieMax         = 970000415
	u3aCommReceived       = 970000416
	u3aCommLong           = 970000417
	u3aCommHiddenStatus   = 970000418
	u3aCommLiked          = 970000419
	u3aCommOnHidden       = 970000421
	u3aCommReceivedHidden = 970000422
	u3aCommOnUsers        = 970000431
	u3aCommReceivedUsers  = 970000432
	u3aCommOnNSFW         = 970000441
	u3aCommBanned         = 970000442
)

func newU3aFix(t *testing.T) *meFix {
	t.Helper()
	f := newMeFix(t)
	t.Cleanup(func() { f.cleanupU3a(t) })
	f.cleanupU3a(t)
	f.addOAuthUser(u3aOwner, "u3a-owner", 0, nil)
	f.addOAuthUser(u3aPeer, "u3a-peer", 0, nil)
	f.putSession(t, "sess-owner", u3aOwner)
	f.seedU3a(t)
	return f
}

func (f *meFix) cleanupU3a(t *testing.T) {
	t.Helper()
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Errorf("u3a cleanup: %v\n%s", err, q)
		}
	}
	run(`DELETE FROM feed_activity WHERE user_id BETWEEN ? AND ?`, u3aUserMin, u3aUserMax)
	run(`DELETE FROM topic_comment_like WHERE topic_comment_id BETWEEN ? AND ?`, u3aCommMin, u3aCommMax)
	run(`DELETE FROM topic_comment WHERE id BETWEEN ? AND ?`, u3aCommMin, u3aCommMax)
	run(`DELETE FROM topic_reply_reaction WHERE topic_reply_id BETWEEN ? AND ?`, u3aReplyMin, u3aReplyMax)
	run(`UPDATE topic SET pinned_reply_id = NULL, best_answer_id = NULL WHERE id BETWEEN ? AND ?`, u3aTopicMin, u3aTopicMax)
	run(`DELETE FROM topic_reply WHERE id BETWEEN ? AND ?`, u3aReplyMin, u3aReplyMax)
	run(`DELETE FROM topic_reaction WHERE topic_id BETWEEN ? AND ?`, u3aTopicMin, u3aTopicMax)
	run(`DELETE FROM topic_favorite WHERE topic_id BETWEEN ? AND ?`, u3aTopicMin, u3aTopicMax)
	run(`DELETE FROM topic_upvote WHERE topic_id BETWEEN ? AND ?`, u3aTopicMin, u3aTopicMax)
	run(`DELETE FROM topic_access_grant WHERE topic_id BETWEEN ? AND ?`, u3aTopicMin, u3aTopicMax)
	run(`DELETE FROM topic WHERE id BETWEEN ? AND ?`, u3aTopicMin, u3aTopicMax)
}

func (f *meFix) seedU3a(t *testing.T) {
	t.Helper()
	tie := time.Date(2026, 8, 2, 15, 0, 0, 0, time.UTC)
	older := tie.Add(-time.Hour)

	insTopic := func(id, user, status int, scope string, nsfw bool, created time.Time) {
		t.Helper()
		hiddenBy := ""
		if status == 1 {
			hiddenBy = "moderator"
		}
		f.execSQL(t, `INSERT INTO topic (
				id, title, content, view, status, category, status_update_time, created, updated,
				user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
				favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
			) VALUES (?, ?, 'body', 0, ?, 'galgame', ?, ?, ?, ?, ?, ?, '', 0, 0, 0, 0, 0, 0, 0, 0, ?, 0)`,
			id, "u3a-"+strconv.Itoa(id), status, created, created, created, user, nsfw, scope, hiddenBy)
	}
	for id := u3aTopicTieMin; id <= u3aTopicTieMax; id++ {
		insTopic(id, u3aOwner, 0, "public", false, tie)
	}
	insTopic(u3aTopicHidden, u3aOwner, 1, "public", false, older)
	insTopic(u3aTopicHidUsers, u3aOwner, 1, "users", false, older)
	insTopic(u3aTopicUsers, u3aOwner, 0, "users", false, older)
	insTopic(u3aTopicRole, u3aOwner, 0, "role", false, older)
	insTopic(u3aTopicNSFW, u3aOwner, 0, "public", true, older)
	insTopic(u3aTopicLogin, u3aOwner, 0, "login", false, older)
	insTopic(u3aPeerPub, u3aPeer, 0, "public", false, tie)
	insTopic(u3aPeerHidden, u3aPeer, 1, "public", false, older)
	insTopic(u3aPeerUsers, u3aPeer, 0, "users", false, older)

	for _, id := range []int{u3aTopicTieMin, u3aTopicTieMin + 2, u3aTopicTieMax, u3aTopicHidden, u3aTopicUsers, u3aTopicNSFW, u3aPeerPub, u3aPeerHidden, u3aPeerUsers} {
		f.execSQL(t, `INSERT INTO topic_reaction (topic_id, user_id, reaction, created) VALUES (?, ?, 'like', ?)`,
			id, u3aOwner, tie)
	}
	for _, id := range []int{u3aTopicTieMin + 1, u3aTopicTieMin + 3, u3aTopicHidden, u3aTopicUsers, u3aTopicLogin} {
		f.execSQL(t, `INSERT INTO topic_upvote (topic_id, user_id, description, created, updated) VALUES (?, ?, '', ?, ?)`,
			id, u3aOwner, older, older)
	}
	for _, id := range []int{u3aTopicTieMin, u3aTopicNSFW, u3aTopicHidden, u3aPeerPub} {
		f.execSQL(t, `INSERT INTO topic_favorite (topic_id, user_id, created, updated) VALUES (?, ?, ?, ?)`,
			id, u3aOwner, older, older)
	}

	insReply := func(id, topic, user, floor, status int, body string, created time.Time) {
		t.Helper()
		f.execSQL(t, `INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)`, id, body, floor, user, topic, status, created, created)
	}
	for id := u3aReplyTieMin; id <= u3aReplyTieMax; id++ {
		insReply(id, u3aTopicTieMin, u3aOwner, id-u3aReplyTieMin+1, 0, "tie-"+strconv.Itoa(id), tie)
	}
	insReply(u3aReplyPeerPub, u3aTopicTieMin, u3aPeer, 6, 0, "peer-pub", older)
	insReply(u3aReplySelf, u3aTopicTieMin, u3aOwner, 7, 0, "self", older)
	insReply(u3aReplyHiddenStatus, u3aTopicTieMin, u3aOwner, 8, 1, "hidden-reply", older)
	insReply(u3aReplyLong, u3aTopicTieMin, u3aOwner, 9, 0, longRunes("w"), older)
	insReply(u3aReplyLiked, u3aTopicTieMin, u3aPeer, 10, 0, "**liked-pub**", older)
	insReply(u3aReplyOnHidden, u3aTopicHidden, u3aOwner, 1, 0, "on-hidden", older)
	insReply(u3aReplyPeerOnHidden, u3aTopicHidden, u3aPeer, 2, 0, "peer-on-hidden", older)
	insReply(u3aReplyLikedHidden, u3aTopicHidden, u3aPeer, 3, 0, "liked-hidden", older)
	insReply(u3aReplyOnUsers, u3aTopicUsers, u3aOwner, 1, 0, "on-users", older)
	insReply(u3aReplyPeerOnUsers, u3aTopicUsers, u3aPeer, 2, 0, "peer-on-users", older)
	insReply(u3aReplyOnNSFW, u3aTopicNSFW, u3aOwner, 1, 0, "on-nsfw", older)

	f.execSQL(t, `INSERT INTO topic_reply_reaction (topic_reply_id, user_id, reaction, created) VALUES (?, ?, 'like', ?), (?, ?, 'like', ?)`,
		u3aReplyLiked, u3aOwner, older, u3aReplyLikedHidden, u3aOwner, older)

	insComment := func(id, topic, reply, user, target, status int, body string, created time.Time) {
		t.Helper()
		f.execSQL(t, `INSERT INTO topic_comment
			(id, content, topic_id, topic_reply_id, user_id, target_user_id, parent_comment_id, status, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)`,
			id, body, topic, reply, user, target, status, created, created)
	}
	for id := u3aCommTieMin; id <= u3aCommTieMax; id++ {
		insComment(id, u3aTopicTieMin, u3aReplyTieMin, u3aOwner, u3aPeer, 0, "ctie-"+strconv.Itoa(id), tie)
	}
	insComment(u3aCommReceived, u3aTopicTieMin, u3aReplyTieMin, u3aPeer, u3aOwner, 0, "received-pub", older)
	insComment(u3aCommLong, u3aTopicTieMin, u3aReplyTieMin, u3aOwner, u3aPeer, 0, longRunes("z"), older)
	insComment(u3aCommHiddenStatus, u3aTopicTieMin, u3aReplyTieMin, u3aOwner, u3aPeer, 1, "hidden-comment", older)
	insComment(u3aCommLiked, u3aTopicTieMin, u3aReplyTieMin, u3aPeer, u3aOwner, 0, "liked-pub-c", older)
	insComment(u3aCommOnHidden, u3aTopicHidden, u3aReplyOnHidden, u3aOwner, u3aPeer, 0, "c-on-hidden", older)
	insComment(u3aCommReceivedHidden, u3aTopicHidden, u3aReplyOnHidden, u3aPeer, u3aOwner, 0, "c-received-hidden", older)
	insComment(u3aCommOnUsers, u3aTopicUsers, u3aReplyOnUsers, u3aOwner, u3aPeer, 0, "c-on-users", older)
	insComment(u3aCommReceivedUsers, u3aTopicUsers, u3aReplyOnUsers, u3aPeer, u3aOwner, 0, "c-received-users", older)
	insComment(u3aCommOnNSFW, u3aTopicNSFW, u3aReplyOnNSFW, u3aOwner, u3aPeer, 0, "c-on-nsfw", older)

	f.execSQL(t, `INSERT INTO topic_comment_like (topic_comment_id, user_id, created, updated) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
		u3aCommLiked, u3aOwner, older, older, u3aCommReceivedHidden, u3aOwner, older, older)
}

func longRunes(ch string) string {
	return strings.Repeat(ch, 210)
}

func (f *meFix) userList(t *testing.T, session, collection, userID, query string) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodGet, "/api/v1/users/"+userID+"/"+collection+"?"+query,
		"/users/{user_id}/"+collection, session, "", nil, nil)
}

func (f *meFix) userListOK(t *testing.T, session, collection, relation, extra string) map[string]any {
	t.Helper()
	q := "relation=" + relation + "&limit=100" + extra
	resp, body := f.userList(t, session, collection, strconv.Itoa(u3aOwner), q)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s %s: %d %+v", collection, relation, resp.StatusCode, body)
	}
	return body
}

func (f *meFix) walkUserList(t *testing.T, session, collection, relation string) ([]string, int) {
	t.Helper()
	owner := strconv.Itoa(u3aOwner)
	var ids []string
	total := -1
	for page := 1; page <= 20; page++ {
		q := fmt.Sprintf("relation=%s&page=%d&limit=2", relation, page)
		resp, body := f.userList(t, session, collection, owner, q)
		if resp.StatusCode != http.StatusOK || body["object"] != "list" {
			t.Fatalf("%s %s page %d: %d %+v", collection, relation, page, resp.StatusCode, body)
		}
		if body["total_relation"] != "eq" {
			t.Fatalf("%s page %d total_relation %v", collection, page, body["total_relation"])
		}
		n := asInt(body["total"])
		if total < 0 {
			total = n
		} else if n != total {
			t.Fatalf("%s page %d total %d, want %d", collection, page, n, total)
		}
		got := itemIDs(t, body)
		if len(got) == 0 {
			break
		}
		ids = append(ids, got...)
	}
	return ids, total
}

func listContains(body map[string]any, id int) bool {
	want := strconv.Itoa(id)
	raw, _ := body["items"].([]any)
	for _, it := range raw {
		m, _ := it.(map[string]any)
		if strID(m["id"]) == want {
			return true
		}
	}
	return false
}

func listItem(t *testing.T, body map[string]any, id int) map[string]any {
	t.Helper()
	want := strconv.Itoa(id)
	raw, _ := body["items"].([]any)
	for _, it := range raw {
		m, _ := it.(map[string]any)
		if strID(m["id"]) == want {
			return m
		}
	}
	t.Fatalf("missing item %d in %v", id, itemIDs(t, body))
	return nil
}

func (f *meFix) u3aIDs(t *testing.T, q string, args ...any) []string {
	t.Helper()
	return f.sqlIDs(t, q, args...)
}

func sharedTopicSQL(authenticated bool) string {
	return "topic.status != 1 AND " + topicRepo.SharedListPredicate("topic", authenticated) + " AND topic.is_nsfw = false"
}
