package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"

	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func TestV1TopicLikeSetRepeatRemove(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/reactions/like"
	spec := "/topics/{topic_id}/reactions/{reaction}"

	resp, body := f.do(t, http.MethodPut, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("set like %d %s", resp.StatusCode, body)
	}
	eng := jsonObj(t, body)
	if eng["like_count"] != float64(1) || eng["dislike_count"] != float64(0) {
		t.Fatalf("counts after like %+v", eng)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reaction WHERE topic_id = ? AND user_id = ? AND reaction = 'like'`, e1TopicPublic, e1UserBob) != 1 {
		t.Fatal("like row missing")
	}
	var rowID int
	if err := f.db.Raw(`SELECT id FROM topic_reaction WHERE topic_id = ? AND user_id = ? AND reaction = 'like'`, e1TopicPublic, e1UserBob).Scan(&rowID).Error; err != nil {
		t.Fatal(err)
	}
	calls := f.awards.snapshot()
	if len(calls) != 1 {
		t.Fatalf("awards %d: %+v", len(calls), calls)
	}
	wantKey := moemoepoint.Key("liked", "topic_reaction_"+strID(rowID))
	if calls[0] != (engageAward{e1UserAlice, 1, moemoepoint.ReasonLiked, moemoepoint.Ref("topic", e1TopicPublic), wantKey}) {
		t.Fatalf("award %+v", calls[0])
	}
	msgs := f.messages(t, e1UserAlice, "liked")
	if len(msgs) != 1 || msgs[0]["sender"] != e1UserBob || msgs[0]["link"] != "/topic/"+strID(e1TopicPublic) {
		t.Fatalf("liked message %+v", msgs)
	}
	wantNotifRefs(t, f.db, []string{notifRef("liked", "topic", e1TopicPublic, "topic", e1TopicPublic)},
		"receiver_id = ? AND type = 'liked'", e1UserAlice)

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
	unliked := moemoepoint.Key("unliked", "topic_reaction_"+strID(rowID))
	if calls[1].Key != unliked || calls[1].Delta != -1 || calls[1].UserID != e1UserAlice {
		t.Fatalf("unliked %+v want key %s", calls[1], unliked)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reaction WHERE topic_id = ? AND user_id = ? AND reaction = 'like'`, e1TopicPublic, e1UserBob) != 0 {
		t.Fatal("like row still present")
	}
}

func TestV1TopicLikeDislikeSwitch(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	likePath := "/api/v1/topics/" + strID(e1TopicPublic) + "/reactions/like"
	dislikePath := "/api/v1/topics/" + strID(e1TopicPublic) + "/reactions/dislike"
	spec := "/topics/{topic_id}/reactions/{reaction}"

	resp, body := f.do(t, http.MethodPut, likePath, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	var likeID int
	if err := f.db.Raw(`SELECT id FROM topic_reaction WHERE topic_id = ? AND user_id = ? AND reaction = 'like'`, e1TopicPublic, e1UserBob).Scan(&likeID).Error; err != nil {
		t.Fatal(err)
	}

	resp, body = f.do(t, http.MethodPut, dislikePath, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("like->dislike %d %s", resp.StatusCode, body)
	}
	eng := jsonObj(t, body)
	if eng["like_count"] != float64(0) || eng["dislike_count"] != float64(1) {
		t.Fatalf("switch counts %+v", eng)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reaction WHERE topic_id = ? AND user_id = ? AND reaction = 'like'`, e1TopicPublic, e1UserBob) != 0 {
		t.Fatal("like row still present after switch")
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reaction WHERE topic_id = ? AND user_id = ? AND reaction = 'dislike'`, e1TopicPublic, e1UserBob) != 1 {
		t.Fatal("dislike row missing")
	}
	calls := f.awards.snapshot()
	if len(calls) != 2 {
		t.Fatalf("awards %+v", calls)
	}
	if calls[1].Key != moemoepoint.Key("unliked", "topic_reaction_"+strID(likeID)) || calls[1].Delta != -1 {
		t.Fatalf("reversal %+v", calls[1])
	}

	resp, body = f.do(t, http.MethodPut, likePath, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("dislike->like %d %s", resp.StatusCode, body)
	}
	eng = jsonObj(t, body)
	if eng["like_count"] != float64(1) || eng["dislike_count"] != float64(0) {
		t.Fatalf("switch back %+v", eng)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reaction WHERE topic_id = ? AND reaction = 'dislike' AND user_id = ?`, e1TopicPublic, e1UserBob) != 0 {
		t.Fatal("dislike remained")
	}
}

func TestV1TopicEmojiAlongsideLike(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	spec := "/topics/{topic_id}/reactions/{reaction}"
	resp, body := f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicPublic)+"/reactions/like", sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicPublic)+"/reactions/heart", sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("heart %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["like_count"] != float64(1) {
		t.Fatalf("heart changed like_count %s", body)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reaction WHERE topic_id = ? AND user_id = ?`, e1TopicPublic, e1UserBob) != 2 {
		t.Fatal("want like and heart")
	}
}

func TestV1TopicSelfLikeForbiddenSelfDislikeOK(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	spec := "/topics/{topic_id}/reactions/{reaction}"
	resp, body := f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicPublic)+"/reactions/like", sess, spec, nil, nil)
	if resp.StatusCode != 403 {
		t.Fatalf("self-like %d %s", resp.StatusCode, body)
	}
	obj := jsonObj(t, body)
	if obj["code"] != problem.CodeSelfLikeForbidden {
		t.Fatalf("code %v", obj["code"])
	}
	resp, body = f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicPublic)+"/reactions/dislike", sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("self-dislike %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["dislike_count"] != float64(1) {
		t.Fatalf("%s", body)
	}
}

func TestV1TopicUnknownReaction(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	resp, body := f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicPublic)+"/reactions/notatoken", sess, "/topics/{topic_id}/reactions/{reaction}", nil, nil)
	if resp.StatusCode != 400 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	obj := jsonObj(t, body)
	errs, _ := obj["errors"].([]any)
	if len(errs) == 0 {
		t.Fatalf("no errors %s", body)
	}
	fe, _ := errs[0].(map[string]any)
	if fe["parameter"] != "reaction" || fe["reason"] != problem.ReasonUnknownValue {
		t.Fatalf("field %+v", fe)
	}
}

func TestV1TopicReactionHiddenNotFoundEvenForAuthor(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	resp, body := f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicHiddenAuthor)+"/reactions/heart", sess, "/topics/{topic_id}/reactions/{reaction}", nil, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
}

func TestV1TopicReactionLoginScopeSignedIn(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	resp, body := f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicLogin)+"/reactions/like", sess, "/topics/{topic_id}/reactions/{reaction}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
}

func TestV1TopicConcurrentLikesOneRow(t *testing.T) {
	f := newEngageFix(t)
	f.putSession(t, "sess-bob", e1UserBob)
	path := "/api/v1/topics/" + strID(e1TopicConcurrent) + "/reactions/like"
	spec := "/topics/{topic_id}/reactions/{reaction}"
	var wg sync.WaitGroup
	wg.Add(2)
	errs := make([]int, 2)
	for i := range 2 {
		go func(i int) {
			defer wg.Done()
			resp, body := f.do(t, http.MethodPut, path, "sess-bob", spec, nil, nil)
			errs[i] = resp.StatusCode
			if resp.StatusCode != 200 {
				t.Errorf("concurrent %d %s", resp.StatusCode, body)
			}
		}(i)
	}
	wg.Wait()
	if f.count(t, `SELECT COUNT(*) FROM topic_reaction WHERE topic_id = ? AND user_id = ? AND reaction = 'like'`, e1TopicConcurrent, e1UserBob) != 1 {
		t.Fatal("want one like row")
	}
	if f.topicInt(t, e1TopicConcurrent, "like_count") != 1 {
		t.Fatalf("like_count %d want 1 (RETURNING guard)", f.topicInt(t, e1TopicConcurrent, "like_count"))
	}
	if len(f.awards.snapshot()) != 1 {
		t.Fatalf("awards %d", len(f.awards.snapshot()))
	}
}

func TestV1TopicEngagementMatchesGetTopic(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	spec := "/topics/{topic_id}/reactions/{reaction}"
	resp, body := f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicPublic)+"/reactions/like", sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	eng := jsonObj(t, body)
	resp, got := f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicPublic), sess, "/topics/{topic_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, got)
	}
	topic := jsonObj(t, got)
	for _, k := range []string{"like_count", "dislike_count", "favorite_count", "upvote_count", "upvoted_at"} {
		if !jsonEqual(eng[k], topic[k]) {
			t.Errorf("%s eng=%v topic=%v", k, eng[k], topic[k])
		}
	}
	eb, _ := json.Marshal(eng["reactions"])
	tb, _ := json.Marshal(topic["reactions"])
	if string(eb) != string(tb) {
		t.Errorf("reactions eng=%s topic=%s", eb, tb)
	}
	ev := eng["viewer"].(map[string]any)
	tv := topic["viewer"].(map[string]any)
	for _, k := range []string{"has_liked", "has_disliked", "has_favorited", "has_upvoted", "can_like", "can_upvote", "can_edit"} {
		if !jsonEqual(ev[k], tv[k]) {
			t.Errorf("viewer.%s eng=%v topic=%v", k, ev[k], tv[k])
		}
	}
}

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func TestV1EngageAwardsWaitForTheCommit(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)

	const cb = "engage_award_commit_test"
	_ = f.db.Callback().Create().Before("gorm:create").Register(cb, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "message" {
			return
		}
		_ = tx.AddError(errors.New("message insert refused by the test"))
	})
	t.Cleanup(func() { _ = f.db.Callback().Create().Remove(cb) })

	resp, body := f.do(t, http.MethodPut,
		"/api/v1/topics/"+strID(e1TopicPublic)+"/reactions/like", sess,
		"/topics/{topic_id}/reactions/{reaction}", nil, nil)
	if resp.StatusCode != 500 {
		t.Fatalf("like with a failing message insert %d %s", resp.StatusCode, body)
	}
	if calls := f.awards.snapshot(); len(calls) != 0 {
		t.Fatalf("awards escaped a rolled-back transaction: %+v", calls)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_reaction WHERE topic_id = ? AND user_id = ?`, e1TopicPublic, e1UserBob) != 0 {
		t.Fatal("reaction row survived the rollback")
	}
	if f.topicInt(t, e1TopicPublic, "like_count") != 0 {
		t.Fatalf("like_count %d after a rolled-back like", f.topicInt(t, e1TopicPublic, "like_count"))
	}
}
