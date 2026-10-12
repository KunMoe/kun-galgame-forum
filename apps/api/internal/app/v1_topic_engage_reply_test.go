package app

import (
	"encoding/json"
	"net/http"
	"testing"

	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
)

func TestV1ReplyLikeSetRepeatRemove(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	path := "/api/v1/replies/" + strID(e1ReplyDave) + "/reactions/like"
	spec := "/replies/{reply_id}/reactions/{reaction}"

	resp, body := f.do(t, http.MethodPut, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("set like %d %s", resp.StatusCode, body)
	}
	eng := jsonObj(t, body)
	if eng["like_count"] != float64(1) || eng["dislike_count"] != float64(0) {
		t.Fatalf("counts after like %+v", eng)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reply_reaction WHERE topic_reply_id = ? AND user_id = ? AND reaction = 'like'`, e1ReplyDave, e1UserBob) != 1 {
		t.Fatal("like row missing")
	}
	if f.replyInt(t, e1ReplyDave, "like_count") != 1 {
		t.Fatalf("like_count %d", f.replyInt(t, e1ReplyDave, "like_count"))
	}
	rowID := f.reactionRowID(t, e1ReplyDave, e1UserBob, "like")
	calls := f.awards.snapshot()
	if len(calls) != 1 {
		t.Fatalf("awards %d: %+v", len(calls), calls)
	}
	wantKey := moemoepoint.Key("liked", "topic_reply_reaction_"+strID(rowID))
	if calls[0] != (engageAward{e1UserDave, 1, moemoepoint.ReasonLiked, moemoepoint.Ref("topic_reply", e1ReplyDave), wantKey}) {
		t.Fatalf("award %+v", calls[0])
	}
	msgs := f.messages(t, e1UserDave, "liked")
	if len(msgs) != 1 || msgs[0]["sender"] != e1UserBob || msgs[0]["link"] != "/topic/"+strID(e1TopicPublic)+"?reply=1" {
		t.Fatalf("liked message %+v", msgs)
	}
	wantNotifRefs(t, f.db, []string{notifRef("liked", "reply", e1ReplyDave, "topic", e1TopicPublic)},
		"receiver_id = ? AND type = 'liked'", e1UserDave)

	resp, body = f.do(t, http.MethodPut, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("repeat like %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["like_count"] != float64(1) {
		t.Fatalf("repeat changed count %s", body)
	}
	if len(f.awards.snapshot()) != 1 {
		t.Fatalf("repeat awarded again %+v", f.awards.snapshot())
	}

	resp, body = f.do(t, http.MethodDelete, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("remove like %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["like_count"] != float64(0) {
		t.Fatalf("remove count %s", body)
	}
	calls = f.awards.snapshot()
	if len(calls) != 2 {
		t.Fatalf("unliked awards %d %+v", len(calls), calls)
	}
	unliked := moemoepoint.Key("unliked", "topic_reply_reaction_"+strID(rowID))
	if calls[1] != (engageAward{e1UserDave, -1, moemoepoint.ReasonLiked, moemoepoint.Ref("topic_reply", e1ReplyDave), unliked}) {
		t.Fatalf("unliked %+v want key %s", calls[1], unliked)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reply_reaction WHERE topic_reply_id = ? AND user_id = ? AND reaction = 'like'`, e1ReplyDave, e1UserBob) != 0 {
		t.Fatal("like row still present")
	}
	if f.replyInt(t, e1ReplyDave, "like_count") != 0 {
		t.Fatalf("like_count after remove %d", f.replyInt(t, e1ReplyDave, "like_count"))
	}
}

func TestV1ReplyLikeDislikeSwitch(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	likePath := "/api/v1/replies/" + strID(e1ReplyDave) + "/reactions/like"
	dislikePath := "/api/v1/replies/" + strID(e1ReplyDave) + "/reactions/dislike"
	spec := "/replies/{reply_id}/reactions/{reaction}"

	resp, body := f.do(t, http.MethodPut, likePath, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	likeID := f.reactionRowID(t, e1ReplyDave, e1UserBob, "like")

	resp, body = f.do(t, http.MethodPut, dislikePath, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("like->dislike %d %s", resp.StatusCode, body)
	}
	eng := jsonObj(t, body)
	if eng["like_count"] != float64(0) || eng["dislike_count"] != float64(1) {
		t.Fatalf("switch counts %+v", eng)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reply_reaction WHERE topic_reply_id = ? AND user_id = ? AND reaction = 'like'`, e1ReplyDave, e1UserBob) != 0 {
		t.Fatal("like row still present after switch")
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reply_reaction WHERE topic_reply_id = ? AND user_id = ? AND reaction = 'dislike'`, e1ReplyDave, e1UserBob) != 1 {
		t.Fatal("dislike row missing")
	}
	if f.replyInt(t, e1ReplyDave, "like_count") != 0 || f.replyInt(t, e1ReplyDave, "dislike_count") != 1 {
		t.Fatalf("db counts like=%d dislike=%d", f.replyInt(t, e1ReplyDave, "like_count"), f.replyInt(t, e1ReplyDave, "dislike_count"))
	}
	calls := f.awards.snapshot()
	if len(calls) != 2 {
		t.Fatalf("awards %+v", calls)
	}
	if calls[1].Key != moemoepoint.Key("unliked", "topic_reply_reaction_"+strID(likeID)) || calls[1].Delta != -1 {
		t.Fatalf("reversal %+v", calls[1])
	}
}

func TestV1ReplyEmojiAlongsideLike(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	spec := "/replies/{reply_id}/reactions/{reaction}"
	base := "/api/v1/replies/" + strID(e1ReplyDave) + "/reactions/"
	resp, body := f.do(t, http.MethodPut, base+"like", sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodPut, base+"heart", sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("heart %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["like_count"] != float64(1) {
		t.Fatalf("heart changed like_count %s", body)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reply_reaction WHERE topic_reply_id = ? AND user_id = ?`, e1ReplyDave, e1UserBob) != 2 {
		t.Fatal("want like and heart")
	}
}

func TestV1ReplySelfLikeForbiddenSelfDislikeOK(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	spec := "/replies/{reply_id}/reactions/{reaction}"
	resp, body := f.do(t, http.MethodPut, "/api/v1/replies/"+strID(e1ReplyAlice)+"/reactions/like", sess, spec, nil, nil)
	if resp.StatusCode != 403 {
		t.Fatalf("self-like %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["code"] != problem.CodeSelfLikeForbidden {
		t.Fatalf("code %v", jsonObj(t, body)["code"])
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reply_reaction WHERE topic_reply_id = ?`, e1ReplyAlice) != 0 {
		t.Fatal("self-like wrote a row")
	}
	resp, body = f.do(t, http.MethodPut, "/api/v1/replies/"+strID(e1ReplyAlice)+"/reactions/dislike", sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("self-dislike %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["dislike_count"] != float64(1) {
		t.Fatalf("%s", body)
	}
}

func TestV1ReplyReactionHiddenAndBannedNotFound(t *testing.T) {
	f := newEngageFix(t)
	bob := f.asBob(t)
	alice := f.asAlice(t)
	spec := "/replies/{reply_id}/reactions/{reaction}"
	for _, tc := range []struct {
		name, session string
		id            int
	}{
		{"hidden reply", bob, e1ReplyHidden},
		{"reply of hidden topic stranger", bob, e1ReplyOnHidden},
		{"reply of hidden topic author", alice, e1ReplyOnHidden},
		{"banned author reply", bob, e1ReplyBanned},
	} {
		resp, body := f.do(t, http.MethodPut, "/api/v1/replies/"+strID(tc.id)+"/reactions/like", tc.session, spec, nil, nil)
		if resp.StatusCode != 404 {
			t.Errorf("%s: %d %s", tc.name, resp.StatusCode, body)
		}
		if jsonObj(t, body)["code"] != problem.CodeNotFound {
			t.Errorf("%s code %v", tc.name, jsonObj(t, body)["code"])
		}
	}
}

func TestV1ReplyEngagementMatchesGetReply(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	spec := "/replies/{reply_id}/reactions/{reaction}"
	resp, body := f.do(t, http.MethodPut, "/api/v1/replies/"+strID(e1ReplyDave)+"/reactions/like", sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	eng := jsonObj(t, body)
	resp, got := f.do(t, http.MethodGet, "/api/v1/replies/"+strID(e1ReplyDave), sess, "/replies/{reply_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, got)
	}
	reply := jsonObj(t, got)
	for _, k := range []string{"like_count", "dislike_count"} {
		if !jsonEqual(eng[k], reply[k]) {
			t.Errorf("%s eng=%v reply=%v", k, eng[k], reply[k])
		}
	}
	eb, _ := json.Marshal(eng["reactions"])
	rb, _ := json.Marshal(reply["reactions"])
	if string(eb) != string(rb) {
		t.Errorf("reactions eng=%s reply=%s", eb, rb)
	}
	ev := eng["viewer"].(map[string]any)
	tv := reply["viewer"].(map[string]any)
	for _, k := range []string{"has_liked", "has_disliked", "can_like", "can_edit"} {
		if !jsonEqual(ev[k], tv[k]) {
			t.Errorf("viewer.%s eng=%v reply=%v", k, ev[k], tv[k])
		}
	}
}
