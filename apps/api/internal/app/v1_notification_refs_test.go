package app

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"

	galgameRepo "kun-galgame-api/internal/galgame/repository"
	galgameService "kun-galgame-api/internal/galgame/service"
	msgModel "kun-galgame-api/internal/message/model"
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/testdb"
	topicService "kun-galgame-api/internal/topic/service"

	"gorm.io/gorm"
)

func notifRef(typ, kind string, id int, rootKind string, rootID int) string {
	return fmt.Sprintf("%s %s:%d %s:%d", typ, kind, id, rootKind, rootID)
}

func notifRefs(t *testing.T, db *gorm.DB, where string, args ...any) []string {
	t.Helper()
	var out []string
	if err := db.Raw(`SELECT type || ' ' || COALESCE(subject_kind, '-') || ':' || COALESCE(subject_id::text, '-')
		|| ' ' || COALESCE(root_kind, '-') || ':' || COALESCE(root_id::text, '-')
		FROM message WHERE `+where+` ORDER BY id`, args...).Scan(&out).Error; err != nil {
		t.Fatal(err)
	}
	return out
}

func wantNotifRefs(t *testing.T, db *gorm.DB, want []string, where string, args ...any) {
	t.Helper()
	if got := notifRefs(t, db, where, args...); !slices.Equal(got, want) {
		t.Errorf("notification refs where %s %v:\n got  %q\n want %q", where, args, got, want)
	}
}

func TestNotificationRefsOfEditMentions(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	resp, raw := f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/replies/%d", w3ReplyMin), "sess-bob",
		"/replies/{reply_id}", "", nil, map[string]any{"content_markdown": "edited " + mention(w3UserAlice)})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reply edit %d %s", resp.StatusCode, raw)
	}
	wantNotifRefs(t, f.db, []string{notifRef("mentioned", "reply", w3ReplyMin, "topic", w3TopicFloors)},
		"sender_id = ? AND type = 'mentioned'", w3UserBob)

	resp, raw = f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicPub), "sess-alice",
		"/topics/{topic_id}", "", nil, map[string]any{"content_markdown": "edited " + mention(w3UserBob)})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("topic edit %d %s", resp.StatusCode, raw)
	}
	wantNotifRefs(t, f.db, []string{notifRef("mentioned", "topic", w3TopicPub, "topic", w3TopicPub)},
		"sender_id = ? AND type = 'mentioned'", w3UserAlice)
}

func TestNotificationRefsOfLotteryNotices(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", codePrize("KUN-REF-CODE-0001")))
	for _, s := range []string{"sess-bob", "sess-other"} {
		if resp, body := f.enterLottery(t, id, s); resp.StatusCode != http.StatusOK {
			t.Fatalf("enter %s %d %+v", s, resp.StatusCode, body)
		}
	}
	f.drawLottery(t, id, "sess-alice")
	lottery := asInt(id)
	winner := f.scalar(t, `SELECT user_id FROM topic_lottery_entry WHERE lottery_id = ? AND code_id <> 0`, lottery)
	loser := w3UserBob
	if winner == w3UserBob {
		loser = w3UserOther
	}
	want := []string{notifRef("lottery-won", "lottery", lottery, "topic", w3TopicPub)}
	wantNotifRefs(t, f.db, want, "receiver_id = ? AND type = 'lottery-won'", winner)
	want = []string{notifRef("lottery-closed", "lottery", lottery, "topic", w3TopicPub)}
	wantNotifRefs(t, f.db, want, "receiver_id = ? AND type = 'lottery-closed'", loser)

	if err := f.db.Exec(`UPDATE topic_lottery_entry SET claim_deadline = now() - interval '1 minute'
		WHERE lottery_id = ? AND code_id <> 0`, lottery).Error; err != nil {
		t.Fatal(err)
	}
	topicService.NewLotteryDrawer(f.LotteryService).Run()
	want = []string{notifRef("lottery-expired", "lottery", lottery, "topic", w3TopicPub)}
	wantNotifRefs(t, f.db, want, "receiver_id = ? AND type = 'lottery-expired'", winner)
}

func TestNotificationRefsOfClosedPolls(t *testing.T) {
	f := newPollFix(t, nil)
	if resp, body := f.voteSet(t, w5bPollSingle, "sess-bob", []string{strconv.Itoa(w5bOptSingleA)}); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, body)
	}
	if err := f.db.Exec(`UPDATE topic_poll SET deadline = now() - interval '1 minute' WHERE id = ?`, w5bPollSingle).Error; err != nil {
		t.Fatal(err)
	}
	var closed []struct{ ID, TopicID int }
	if err := f.db.Raw(`SELECT DISTINCT p.id, p.topic_id FROM topic_poll p JOIN topic_poll_vote v ON v.poll_id = p.id
		WHERE v.user_id = ? AND p.deadline <= now() AND p.user_id <> v.user_id`, w3UserBob).Scan(&closed).Error; err != nil {
		t.Fatal(err)
	}
	topicService.NewLotteryDrawer(f.LotteryService).Run()

	want := make([]string, 0, len(closed))
	for _, p := range closed {
		want = append(want, notifRef("poll-closed", "poll", p.ID, "topic", p.TopicID))
	}
	got := notifRefs(t, f.db, "receiver_id = ? AND type = 'poll-closed'", w3UserBob)
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Contains(want, notifRef("poll-closed", "poll", w5bPollSingle, "topic", w3TopicPub)) || !slices.Equal(got, want) {
		t.Errorf("poll-closed refs %q, want %q", got, want)
	}
}

func TestNotificationRefsOfWorkFavorite(t *testing.T) {
	f := newG6Fix(t)
	if err := f.db.Exec(`INSERT INTO galgame (id, updated, published, content_limit, creator_user_id) VALUES (?, now(), true, 'sfw', ?)
		ON CONFLICT (id) DO UPDATE SET creator_user_id = EXCLUDED.creator_user_id`, g6WorkExtra, w3UserBob).Error; err != nil {
		t.Fatal(err)
	}
	resp, body := f.call(t, http.MethodPut, g6col(g6FolderAlicePub)+"/works/"+idStr(g6WorkExtra),
		"/collections/{collection_id}/works/{work_id}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put %d %+v", resp.StatusCode, body)
	}
	wantNotifRefs(t, f.db, []string{notifRef("favorite", "work", g6WorkExtra, "work", g6WorkExtra)},
		"sender_id = ? AND receiver_id = ? AND type = 'favorite'", w3UserAlice, w3UserBob)
}

func TestNotificationDedupKeysOnTheSubject(t *testing.T) {
	db := testdb.Open(t)
	const sender, receiver, work = 971400001, 971400002, 971400101
	clean := func() { db.Exec(`DELETE FROM message WHERE receiver_id = ?`, receiver) }
	clean()
	t.Cleanup(clean)

	works := galgameRepo.NewWorkV1Store(db)
	resources := galgameRepo.NewResourceV1Store(db)
	var ratings galgameService.InteractionHelpers
	notifier := msgService.NewNotifier(msgRepo.NewMessageRepository(db))
	for range 2 {
		for _, err := range []error{
			works.CreateLikedMessage(db, sender, receiver, "work", work),
			resources.CreateResourceMessage(db, sender, receiver, "liked", "resource", work, 501),
			ratings.CreateRatingLikedMessage(db, sender, receiver, "rating", work, 601),
		} {
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, proposal := range []int64{701, 702, 701} {
		if err := notifier.Emit(nil, msgService.Spec{
			SenderID: sender, ReceiverID: receiver, Kind: msgService.NotifyRequested,
			Content: "same work name", WorkID: work, Ref: msgModel.EditProposalRef(proposal, work),
		}); err != nil {
			t.Fatal(err)
		}
	}
	wantNotifRefs(t, db, []string{
		notifRef("liked", "work", work, "work", work),
		notifRef("liked", "resource", 501, "work", work),
		notifRef("liked", "rating", 601, "work", work),
		notifRef("requested", "edit_proposal", 701, "work", work),
		notifRef("requested", "edit_proposal", 702, "work", work),
	}, "receiver_id = ?", receiver)
}

func TestExpiredFoldNoticeLeavesTheSubscriptionWorking(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	f.setSubscription(t, "sess-bob", w3TopicFloors, "watching")
	f.mustReply(t, w3TopicFloors, "sess-other", 1, "first")
	first := f.notices(t, w3UserBob, "subscribed-topic")
	if len(first) != 1 {
		t.Fatalf("first fold %+v", first)
	}
	if err := f.db.Exec(`UPDATE message SET status = 'read', created = now() - interval '400 days' WHERE id = ?`, first[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := msgRepo.NewMessageRepository(f.db).DeleteReadBefore(time.Now().AddDate(0, 0, -365), 5000); err != nil {
		t.Fatal(err)
	}
	if n := f.notices(t, w3UserBob, "subscribed-topic"); len(n) != 0 {
		t.Fatalf("the read fold survived retention: %+v", n)
	}

	floor := f.mustReply(t, w3TopicFloors, "sess-other", 2, "second")
	again := f.notices(t, w3UserBob, "subscribed-topic")
	if len(again) != 1 || again[0].Status != "unread" || again[0].ItemCount != 1 {
		t.Fatalf("a reply after the expired fold %+v, want one fresh fold", again)
	}
	wantNotifRefs(t, f.db, []string{notifRef("subscribed-topic", "reply", f.replyAtFloor(t, w3TopicFloors, floor), "topic", w3TopicFloors)},
		"receiver_id = ? AND type = 'subscribed-topic'", w3UserBob)
	f.markRead(t, "sess-bob", w3TopicFloors, floor)
	if n := f.notices(t, w3UserBob, "subscribed-topic"); len(n) != 1 || n[0].Status != "read" {
		t.Fatalf("reading to the end did not close the new fold: %+v", n)
	}
}
