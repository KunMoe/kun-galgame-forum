package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/problem"
)

func replyChoice(id int) []byte {
	b, _ := json.Marshal(map[string]string{"reply_id": strID(id)})
	return b
}

func upvoteJSON(note *string) []byte {
	if note == nil {
		return []byte(`{}`)
	}
	b, _ := json.Marshal(map[string]any{"note": *note})
	return b
}

func firstField(t *testing.T, body []byte) map[string]any {
	t.Helper()
	errs, _ := jsonObj(t, body)["errors"].([]any)
	if len(errs) == 0 {
		t.Fatalf("no errors %s", body)
	}
	fe, _ := errs[0].(map[string]any)
	if fe == nil {
		t.Fatalf("error 0 %s", body)
	}
	return fe
}

func TestV1TopicFavoriteSetRepeatRemove(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/favorite"
	spec := "/topics/{topic_id}/favorite"

	resp, body := f.do(t, http.MethodDelete, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("remove absent %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["favorite_count"] != float64(0) {
		t.Fatalf("absent remove %s", body)
	}
	if len(f.awards.snapshot()) != 0 {
		t.Fatalf("absent awarded %+v", f.awards.snapshot())
	}

	resp, body = f.do(t, http.MethodPut, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("set %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["favorite_count"] != float64(1) {
		t.Fatalf("set count %s", body)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_favorite WHERE topic_id = ? AND user_id = ?`, e1TopicPublic, e1UserBob) != 1 {
		t.Fatal("favorite row missing")
	}
	if f.topicInt(t, e1TopicPublic, "favorite_count") != 1 {
		t.Fatalf("favorite_count %d", f.topicInt(t, e1TopicPublic, "favorite_count"))
	}
	rowID := f.favoriteRowID(t, e1TopicPublic, e1UserBob)
	calls := f.awards.snapshot()
	if len(calls) != 1 {
		t.Fatalf("awards %d %+v", len(calls), calls)
	}
	wantKey := moemoepoint.Key("favorited", "topic_favorite_"+strID(rowID))
	if calls[0] != (engageAward{e1UserAlice, 1, moemoepoint.ReasonLiked, moemoepoint.Ref("topic", e1TopicPublic), wantKey}) {
		t.Fatalf("award %+v", calls[0])
	}
	msgs := f.messages(t, e1UserAlice, "favorite")
	if len(msgs) != 1 || msgs[0]["sender"] != e1UserBob || msgs[0]["link"] != "/topic/"+strID(e1TopicPublic) {
		t.Fatalf("favorite message %+v", msgs)
	}
	wantNotifRefs(t, f.db, []string{notifRef("favorite", "topic", e1TopicPublic, "topic", e1TopicPublic)},
		"receiver_id = ? AND type = 'favorite'", e1UserAlice)

	resp, body = f.do(t, http.MethodPut, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("repeat %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["favorite_count"] != float64(1) {
		t.Fatalf("repeat count %s", body)
	}
	if len(f.awards.snapshot()) != 1 {
		t.Fatalf("repeat awarded %+v", f.awards.snapshot())
	}

	resp, body = f.do(t, http.MethodDelete, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("remove %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["favorite_count"] != float64(0) {
		t.Fatalf("remove count %s", body)
	}
	calls = f.awards.snapshot()
	if len(calls) != 2 {
		t.Fatalf("unfavorited awards %d %+v", len(calls), calls)
	}
	unfav := moemoepoint.Key("unfavorited", "topic_favorite_"+strID(rowID))
	if calls[1] != (engageAward{e1UserAlice, -1, moemoepoint.ReasonLiked, moemoepoint.Ref("topic", e1TopicPublic), unfav}) {
		t.Fatalf("unfavorited %+v", calls[1])
	}
	if f.topicInt(t, e1TopicPublic, "favorite_count") != 0 {
		t.Fatalf("favorite_count after remove %d", f.topicInt(t, e1TopicPublic, "favorite_count"))
	}
}

func TestV1TopicFavoriteOwnNoAward(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/favorite"
	spec := "/topics/{topic_id}/favorite"
	resp, body := f.do(t, http.MethodPut, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["favorite_count"] != float64(1) {
		t.Fatalf("%s", body)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_favorite WHERE topic_id = ? AND user_id = ?`, e1TopicPublic, e1UserAlice) != 1 {
		t.Fatal("own favorite row missing")
	}
	if len(f.awards.snapshot()) != 0 {
		t.Fatalf("own favorite awarded %+v", f.awards.snapshot())
	}
	if len(f.messages(t, e1UserAlice, "favorite")) != 0 {
		t.Fatalf("own favorite messaged")
	}
}

func TestV1TopicConcurrentFavoritesOneRow(t *testing.T) {
	f := newEngageFix(t)
	f.putSession(t, "sess-bob", e1UserBob)
	path := "/api/v1/topics/" + strID(e1TopicConcurrent) + "/favorite"
	spec := "/topics/{topic_id}/favorite"
	var wg sync.WaitGroup
	wg.Add(2)
	codes := make([]int, 2)
	for i := range 2 {
		go func(i int) {
			defer wg.Done()
			resp, body := f.do(t, http.MethodPut, path, "sess-bob", spec, nil, nil)
			codes[i] = resp.StatusCode
			if resp.StatusCode != 200 {
				t.Errorf("concurrent %d %s", resp.StatusCode, body)
			}
		}(i)
	}
	wg.Wait()
	for _, c := range codes {
		if c == 500 {
			t.Fatal("concurrent favorite returned 500")
		}
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_favorite WHERE topic_id = ? AND user_id = ?`, e1TopicConcurrent, e1UserBob) != 1 {
		t.Fatal("want one favorite row")
	}
	if f.topicInt(t, e1TopicConcurrent, "favorite_count") != 1 {
		t.Fatalf("favorite_count %d want 1", f.topicInt(t, e1TopicConcurrent, "favorite_count"))
	}
	if len(f.awards.snapshot()) != 1 {
		t.Fatalf("awards %d", len(f.awards.snapshot()))
	}
}

func TestV1TopicFavoriteHiddenNotFound(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	resp, body := f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicHiddenAuthor)+"/favorite", sess, "/topics/{topic_id}/favorite", nil, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
}

func TestV1TopicUpvoteCreatesRowAwardsAndMessage(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/upvotes"
	spec := "/topics/{topic_id}/upvotes"
	resp, body := f.do(t, http.MethodPost, path, sess, spec, upvoteJSON(nil), idemKey("00000000-0000-4000-8000-000000000001"))
	if resp.StatusCode != 201 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	obj := jsonObj(t, body)
	if obj["object"] != "topic_upvote" || obj["topic_id"] != strID(e1TopicPublic) {
		t.Fatalf("identity %+v", obj)
	}
	var rowID int
	if err := f.db.Raw(`SELECT id FROM topic_upvote WHERE topic_id = ? AND user_id = ?`, e1TopicPublic, e1UserBob).Scan(&rowID).Error; err != nil {
		t.Fatal(err)
	}
	if obj["id"] != strID(rowID) {
		t.Fatalf("id %v want %s", obj["id"], strID(rowID))
	}
	if obj["note"] != nil {
		t.Fatalf("empty note %v", obj["note"])
	}
	upvoter, _ := obj["upvoter"].(map[string]any)
	if upvoter["object"] != "user" || upvoter["id"] != strID(e1UserBob) || upvoter["name"] != "bob" {
		t.Fatalf("upvoter %+v", upvoter)
	}
	if _, ok := obj["created_at"].(string); !ok {
		t.Fatalf("created_at %v", obj["created_at"])
	}
	if f.topicInt(t, e1TopicPublic, "upvote_count") != 1 {
		t.Fatalf("upvote_count %d", f.topicInt(t, e1TopicPublic, "upvote_count"))
	}
	if f.count(t, `SELECT COUNT(*) FROM topic WHERE id = ? AND upvote_time IS NOT NULL`, e1TopicPublic) != 1 {
		t.Fatal("upvote_time unset")
	}
	resp, got := f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicPublic), sess, "/topics/{topic_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("get topic %d %s", resp.StatusCode, got)
	}
	topic := jsonObj(t, got)
	if topic["upvote_count"] != float64(1) || topic["upvoted_at"] == nil {
		t.Fatalf("topic after upvote %s", got)
	}
	calls := f.awards.snapshot()
	if len(calls) != 2 {
		t.Fatalf("awards %d %+v", len(calls), calls)
	}
	ref := moemoepoint.Ref("topic_upvote", rowID)
	sent := moemoepoint.Key("upvote_sent", "topic_upvote_"+strID(rowID))
	recv := moemoepoint.Key("upvote_received", "topic_upvote_"+strID(rowID))
	if calls[0] != (engageAward{e1UserBob, -constants.CostUpvoteSender, moemoepoint.ReasonContentRemoved, ref, sent}) {
		t.Fatalf("sent %+v", calls[0])
	}
	if calls[1] != (engageAward{e1UserAlice, constants.RewardUpvoteOwner, moemoepoint.ReasonContentApproved, ref, recv}) {
		t.Fatalf("received %+v", calls[1])
	}
	msgs := f.messages(t, e1UserAlice, "upvoted")
	if len(msgs) != 1 || msgs[0]["sender"] != e1UserBob || msgs[0]["link"] != "/topic/"+strID(e1TopicPublic) {
		t.Fatalf("upvoted message %+v", msgs)
	}
	wantNotifRefs(t, f.db, []string{notifRef("upvoted", "topic", e1TopicPublic, "topic", e1TopicPublic)},
		"receiver_id = ? AND type = 'upvoted'", e1UserAlice)
}

func TestV1TopicUpvoteRepeatableAndNoteRules(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/upvotes"
	spec := "/topics/{topic_id}/upvotes"

	resp, body := f.do(t, http.MethodPost, path, sess, spec, upvoteJSON(nil), idemKey("00000000-0000-4000-8000-000000000011"))
	if resp.StatusCode != 201 {
		t.Fatalf("first %d %s", resp.StatusCode, body)
	}
	firstID := jsonObj(t, body)["id"]
	resp, body = f.do(t, http.MethodPost, path, sess, spec, upvoteJSON(nil), idemKey("00000000-0000-4000-8000-000000000012"))
	if resp.StatusCode != 201 {
		t.Fatalf("second %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["id"] == firstID {
		t.Fatal("second upvote reused id")
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_upvote WHERE topic_id = ? AND user_id = ?`, e1TopicPublic, e1UserBob) != 2 {
		t.Fatal("want two upvote rows")
	}
	if f.topicInt(t, e1TopicPublic, "upvote_count") != 2 {
		t.Fatalf("upvote_count %d", f.topicInt(t, e1TopicPublic, "upvote_count"))
	}
	calls := f.awards.snapshot()
	if len(calls) != 4 {
		t.Fatalf("awards %d %+v", len(calls), calls)
	}
	if calls[0].Key == calls[2].Key || calls[1].Key == calls[3].Key {
		t.Fatalf("repeat used same keys %+v", calls)
	}

	trimmed := "  hi  "
	resp, body = f.do(t, http.MethodPost, path, sess, spec, upvoteJSON(&trimmed), idemKey("00000000-0000-4000-8000-000000000013"))
	if resp.StatusCode != 201 {
		t.Fatalf("trim %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["note"] != "hi" {
		t.Fatalf("trimmed note %s", body)
	}

	blank := " \t  "
	resp, body = f.do(t, http.MethodPost, path, sess, spec, upvoteJSON(&blank), idemKey("00000000-0000-4000-8000-000000000014"))
	if resp.StatusCode != 201 {
		t.Fatalf("whitespace %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["note"] != nil {
		t.Fatalf("whitespace note %s", body)
	}

	long := strings.Repeat("a", 31)
	resp, body = f.do(t, http.MethodPost, path, sess, spec, upvoteJSON(&long), idemKey("00000000-0000-4000-8000-000000000015"))
	if resp.StatusCode != 422 {
		t.Fatalf("too long %d %s", resp.StatusCode, body)
	}
	obj := jsonObj(t, body)
	if obj["code"] != problem.CodeValidationFailed {
		t.Fatalf("too long code %v", obj["code"])
	}
	fe := firstField(t, body)
	if fe["pointer"] != "/note" || fe["reason"] != problem.ReasonTooLong {
		t.Fatalf("too long field %+v", fe)
	}
}

func TestV1TopicUpvoteForbiddenAndIdempotency(t *testing.T) {
	f := newEngageFix(t)
	spec := "/topics/{topic_id}/upvotes"
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/upvotes"

	alice := f.asAlice(t)
	resp, body := f.do(t, http.MethodPost, path, alice, spec, upvoteJSON(nil), idemKey("00000000-0000-4000-8000-000000000021"))
	if resp.StatusCode != 403 {
		t.Fatalf("self %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["code"] != problem.CodeSelfUpvoteForbidden {
		t.Fatalf("self code %v", jsonObj(t, body)["code"])
	}

	poor := f.asPoor(t)
	resp, body = f.do(t, http.MethodPost, path, poor, spec, upvoteJSON(nil), idemKey("00000000-0000-4000-8000-000000000022"))
	if resp.StatusCode != 403 {
		t.Fatalf("poor %d %s", resp.StatusCode, body)
	}
	obj := jsonObj(t, body)
	if obj["code"] != problem.CodeMoemoepointInsufficient {
		t.Fatalf("poor code %v", obj["code"])
	}
	if obj["required"] != float64(10) {
		t.Fatalf("required %v", obj["required"])
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_upvote WHERE user_id = ?`, e1UserPoor) != 0 {
		t.Fatal("poor wrote a row")
	}
	if len(f.awards.snapshot()) != 0 {
		t.Fatalf("poor awarded %+v", f.awards.snapshot())
	}

	bob := f.asBob(t)
	resp, body = f.do(t, http.MethodPost, path, bob, spec, upvoteJSON(nil), nil)
	if resp.StatusCode != 400 {
		t.Fatalf("missing key %d %s", resp.StatusCode, body)
	}
	fe := firstField(t, body)
	if fe["header"] != "Idempotency-Key" || fe["reason"] != problem.ReasonRequired {
		t.Fatalf("missing key field %+v", fe)
	}

	key := "00000000-0000-4000-8000-000000000023"
	resp, first := f.do(t, http.MethodPost, path, bob, spec, upvoteJSON(nil), idemKey(key))
	if resp.StatusCode != 201 {
		t.Fatalf("first %d %s", resp.StatusCode, first)
	}
	if resp.Header.Get("Idempotency-Replayed") == "true" {
		t.Fatal("first call replayed")
	}
	resp, again := f.do(t, http.MethodPost, path, bob, spec, upvoteJSON(nil), idemKey(key))
	if resp.StatusCode != 201 {
		t.Fatalf("replay %d %s", resp.StatusCode, again)
	}
	if resp.Header.Get("Idempotency-Replayed") != "true" {
		t.Fatal("missing Idempotency-Replayed")
	}
	if string(first) != string(again) {
		t.Fatalf("replay body %s != %s", again, first)
	}
	if f.count(t, `SELECT COUNT(*) FROM topic_upvote WHERE topic_id = ? AND user_id = ?`, e1TopicPublic, e1UserBob) != 1 {
		t.Fatal("replay wrote a second row")
	}
}

func TestV1TopicUpvoteBumpCutoff(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	created := f.topicTime(t, e1TopicOld, "created")
	if !created.Before(model.BumpCutoff(time.Now())) {
		t.Fatal("fixture old topic is inside the 3-month bump window")
	}
	before := f.topicTime(t, e1TopicOld, "status_update_time")
	resp, body := f.do(t, http.MethodPost, "/api/v1/topics/"+strID(e1TopicOld)+"/upvotes", sess, "/topics/{topic_id}/upvotes", upvoteJSON(nil), idemKey("00000000-0000-4000-8000-000000000031"))
	if resp.StatusCode != 201 {
		t.Fatalf("old %d %s", resp.StatusCode, body)
	}
	after := f.topicTime(t, e1TopicOld, "status_update_time")
	if !after.Equal(before) {
		t.Fatalf("old topic bumped %v -> %v", before, after)
	}
	if f.topicInt(t, e1TopicOld, "upvote_count") != 1 {
		t.Fatalf("old upvote_count %d", f.topicInt(t, e1TopicOld, "upvote_count"))
	}
}

func TestV1TopicUpvoteHiddenNotFound(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	resp, body := f.do(t, http.MethodPost, "/api/v1/topics/"+strID(e1TopicHiddenAuthor)+"/upvotes", sess, "/topics/{topic_id}/upvotes", upvoteJSON(nil), idemKey("00000000-0000-4000-8000-000000000033"))
	if resp.StatusCode != 404 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
}

func TestV1BestAnswerSetRepeatReplaceClear(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/best-answer"
	spec := "/topics/{topic_id}/best-answer"

	resp, body := f.do(t, http.MethodPut, path, sess, spec, replyChoice(e1ReplyDave), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("set %d %s", resp.StatusCode, body)
	}
	topic := jsonObj(t, body)
	ba, _ := topic["best_answer"].(map[string]any)
	if ba == nil || ba["id"] != strID(e1ReplyDave) {
		t.Fatalf("best_answer %v", topic["best_answer"])
	}
	if got := f.topicNullInt(t, e1TopicPublic, "best_answer_id"); got == nil || *got != e1ReplyDave {
		t.Fatalf("column %v", got)
	}
	calls := f.awards.snapshot()
	if len(calls) != 1 {
		t.Fatalf("set awards %+v", calls)
	}
	if calls[0].UserID != e1UserDave || calls[0].Delta != constants.RewardBestAnswer ||
		calls[0].Reason != moemoepoint.ReasonContentApproved ||
		calls[0].Ref != moemoepoint.Ref("topic_reply", e1ReplyDave) ||
		!strings.HasPrefix(calls[0].Key, moemoepoint.Key("best_answer_set", "topic_reply_"+strID(e1ReplyDave))+":") {
		t.Fatalf("set award %+v", calls[0])
	}
	msgs := f.messages(t, e1UserDave, "solution")
	if len(msgs) != 1 || msgs[0]["sender"] != e1UserAlice || msgs[0]["link"] != "/topic/"+strID(e1TopicPublic)+"?reply=1" {
		t.Fatalf("solution %+v", msgs)
	}
	wantNotifRefs(t, f.db, []string{notifRef("solution", "reply", e1ReplyDave, "topic", e1TopicPublic)},
		"receiver_id = ? AND type = 'solution'", e1UserDave)

	resp, body = f.do(t, http.MethodPut, path, sess, spec, replyChoice(e1ReplyDave), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("repeat %d %s", resp.StatusCode, body)
	}
	if len(f.awards.snapshot()) != 1 {
		t.Fatalf("repeat awarded %+v", f.awards.snapshot())
	}

	resp, body = f.do(t, http.MethodPut, path, sess, spec, replyChoice(e1ReplyCarol), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("replace %d %s", resp.StatusCode, body)
	}
	ba, _ = jsonObj(t, body)["best_answer"].(map[string]any)
	if ba["id"] != strID(e1ReplyCarol) {
		t.Fatalf("replaced %v", ba)
	}
	calls = f.awards.snapshot()
	if len(calls) != 3 {
		t.Fatalf("replace awards %+v", calls)
	}
	if calls[1].UserID != e1UserCarol || calls[1].Delta != constants.RewardBestAnswer ||
		!strings.HasPrefix(calls[1].Key, moemoepoint.Key("best_answer_set", "topic_reply_"+strID(e1ReplyCarol))+":") {
		t.Fatalf("new +7 %+v", calls[1])
	}
	if calls[2].UserID != e1UserDave || calls[2].Delta != -constants.RewardBestAnswer ||
		calls[2].Reason != moemoepoint.ReasonContentRemoved ||
		!strings.HasPrefix(calls[2].Key, moemoepoint.Key("best_answer_cleared", "topic_reply_"+strID(e1ReplyDave))+":") {
		t.Fatalf("prev -7 %+v", calls[2])
	}

	resp, body = f.do(t, http.MethodDelete, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("clear %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["best_answer"] != nil {
		t.Fatalf("cleared body %s", body)
	}
	if f.topicNullInt(t, e1TopicPublic, "best_answer_id") != nil {
		t.Fatal("column not NULL")
	}
	calls = f.awards.snapshot()
	if len(calls) != 4 {
		t.Fatalf("clear awards %+v", calls)
	}
	if calls[3].UserID != e1UserCarol || calls[3].Delta != -constants.RewardBestAnswer ||
		!strings.HasPrefix(calls[3].Key, moemoepoint.Key("best_answer_cleared", "topic_reply_"+strID(e1ReplyCarol))+":") {
		t.Fatalf("clear -7 %+v", calls[3])
	}
}

func TestV1BestAnswerOwnReplyNoAward(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/best-answer"
	spec := "/topics/{topic_id}/best-answer"
	resp, body := f.do(t, http.MethodPut, path, sess, spec, replyChoice(e1ReplyAlice), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("set own %d %s", resp.StatusCode, body)
	}
	ba, _ := jsonObj(t, body)["best_answer"].(map[string]any)
	if ba["id"] != strID(e1ReplyAlice) {
		t.Fatalf("own best_answer %v", ba)
	}
	if len(f.awards.snapshot()) != 0 {
		t.Fatalf("own set awarded %+v", f.awards.snapshot())
	}
	if len(f.messages(t, e1UserAlice, "solution")) != 0 {
		t.Fatal("own set messaged")
	}
	resp, body = f.do(t, http.MethodDelete, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("clear own %d %s", resp.StatusCode, body)
	}
	if len(f.awards.snapshot()) != 0 {
		t.Fatalf("own clear awarded %+v", f.awards.snapshot())
	}
}

func TestV1BestAnswerUnknownReply(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/best-answer"
	spec := "/topics/{topic_id}/best-answer"
	for _, id := range []int{e1ReplyOtherTopic, e1ReplyHidden, e1ReplyBanned} {
		resp, body := f.do(t, http.MethodPut, path, sess, spec, replyChoice(id), nil)
		if resp.StatusCode != 422 {
			t.Errorf("id %d: %d %s", id, resp.StatusCode, body)
			continue
		}
		if jsonObj(t, body)["code"] != problem.CodeValidationFailed {
			t.Errorf("id %d code %v", id, jsonObj(t, body)["code"])
		}
		fe := firstField(t, body)
		if fe["pointer"] != "/reply_id" {
			t.Errorf("id %d pointer %+v", id, fe)
		}
	}
}

func TestV1BestAnswerPermissionAndHidden(t *testing.T) {
	f := newEngageFix(t)
	spec := "/topics/{topic_id}/best-answer"
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/best-answer"
	body := replyChoice(e1ReplyDave)

	bob := f.asBob(t)
	resp, raw := f.do(t, http.MethodPut, path, bob, spec, body, nil)
	if resp.StatusCode != 403 || jsonObj(t, raw)["code"] != problem.CodePermissionRequired {
		t.Fatalf("stranger %d %s", resp.StatusCode, raw)
	}

	resp, raw = f.do(t, http.MethodPut, path, "", spec, body, authBearer("staff-token"))
	if resp.StatusCode != 403 || jsonObj(t, raw)["code"] != problem.CodePermissionRequired {
		t.Fatalf("bearer %d %s", resp.StatusCode, raw)
	}

	staff := f.asStaff(t)
	resp, raw = f.do(t, http.MethodPut, path, staff, spec, body, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("cookie staff %d %s", resp.StatusCode, raw)
	}

	alice := f.asAlice(t)
	resp, raw = f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicHiddenAuthor)+"/best-answer", alice, spec, replyChoice(e1ReplyOnHidden), nil)
	if resp.StatusCode != 404 {
		t.Fatalf("hidden %d %s", resp.StatusCode, raw)
	}
}

func TestV1PinReplyReplaceUnpinAndOwn(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/pinned-reply"
	spec := "/topics/{topic_id}/pinned-reply"

	resp, body := f.do(t, http.MethodPut, path, sess, spec, replyChoice(e1ReplyDave), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("pin %d %s", resp.StatusCode, body)
	}
	pr, _ := jsonObj(t, body)["pinned_reply"].(map[string]any)
	if pr == nil || pr["id"] != strID(e1ReplyDave) {
		t.Fatalf("pinned_reply %v", jsonObj(t, body)["pinned_reply"])
	}
	if got := f.topicNullInt(t, e1TopicPublic, "pinned_reply_id"); got == nil || *got != e1ReplyDave {
		t.Fatalf("column %v", got)
	}
	msgs := f.messages(t, e1UserDave, "pin-reply")
	if len(msgs) != 1 || msgs[0]["sender"] != e1UserAlice || msgs[0]["link"] != "/topic/"+strID(e1TopicPublic)+"?reply=1" {
		t.Fatalf("pin-reply %+v", msgs)
	}
	wantNotifRefs(t, f.db, []string{notifRef("pin-reply", "reply", e1ReplyDave, "topic", e1TopicPublic)},
		"receiver_id = ? AND type = 'pin-reply'", e1UserDave)

	resp, body = f.do(t, http.MethodPut, path, sess, spec, replyChoice(e1ReplyDave), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("repeat pin %d %s", resp.StatusCode, body)
	}
	if len(f.messages(t, e1UserDave, "pin-reply")) != 1 {
		t.Fatal("repeat pin messaged")
	}

	resp, body = f.do(t, http.MethodPut, path, sess, spec, replyChoice(e1ReplyCarol), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("replace %d %s", resp.StatusCode, body)
	}
	pr, _ = jsonObj(t, body)["pinned_reply"].(map[string]any)
	if pr["id"] != strID(e1ReplyCarol) {
		t.Fatalf("replaced pin %v", pr)
	}
	if len(f.messages(t, e1UserCarol, "pin-reply")) != 1 {
		t.Fatal("replace pin-reply missing")
	}

	resp, body = f.do(t, http.MethodDelete, path, sess, spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("unpin %d %s", resp.StatusCode, body)
	}
	if jsonObj(t, body)["pinned_reply"] != nil {
		t.Fatalf("unpinned body %s", body)
	}
	if f.topicNullInt(t, e1TopicPublic, "pinned_reply_id") != nil {
		t.Fatal("unpin column")
	}

	resp, body = f.do(t, http.MethodPut, path, sess, spec, replyChoice(e1ReplyAlice), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("own pin %d %s", resp.StatusCode, body)
	}
	if len(f.messages(t, e1UserAlice, "pin-reply")) != 0 {
		t.Fatal("own pin messaged")
	}
}

func TestV1PinReplyUnknownPermissionHidden(t *testing.T) {
	f := newEngageFix(t)
	spec := "/topics/{topic_id}/pinned-reply"
	path := "/api/v1/topics/" + strID(e1TopicPublic) + "/pinned-reply"
	alice := f.asAlice(t)
	for _, id := range []int{e1ReplyOtherTopic, e1ReplyHidden, e1ReplyBanned} {
		resp, body := f.do(t, http.MethodPut, path, alice, spec, replyChoice(id), nil)
		if resp.StatusCode != 422 {
			t.Errorf("id %d: %d %s", id, resp.StatusCode, body)
			continue
		}
		if jsonObj(t, body)["code"] != problem.CodeValidationFailed || firstField(t, body)["pointer"] != "/reply_id" {
			t.Errorf("id %d %s", id, body)
		}
	}

	bob := f.asBob(t)
	resp, body := f.do(t, http.MethodPut, path, bob, spec, replyChoice(e1ReplyDave), nil)
	if resp.StatusCode != 403 || jsonObj(t, body)["code"] != problem.CodePermissionRequired {
		t.Fatalf("stranger %d %s", resp.StatusCode, body)
	}

	resp, body = f.do(t, http.MethodPut, path, "", spec, replyChoice(e1ReplyDave), authBearer("staff-token"))
	if resp.StatusCode != 403 || jsonObj(t, body)["code"] != problem.CodePermissionRequired {
		t.Fatalf("bearer %d %s", resp.StatusCode, body)
	}

	staff := f.asStaff(t)
	resp, body = f.do(t, http.MethodPut, path, staff, spec, replyChoice(e1ReplyDave), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("cookie staff %d %s", resp.StatusCode, body)
	}

	resp, body = f.do(t, http.MethodPut, "/api/v1/topics/"+strID(e1TopicHiddenAuthor)+"/pinned-reply", alice, spec, replyChoice(e1ReplyOnHidden), nil)
	if resp.StatusCode != 404 {
		t.Fatalf("hidden %d %s", resp.StatusCode, body)
	}
}
