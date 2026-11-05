package app

import (
	"net/http"
	"testing"

	topicService "kun-galgame-api/internal/topic/service"
)

func (f *writeFix) revealReturned(t *testing.T, id, session string) (*http.Response, map[string]any) {
	t.Helper()
	return f.lotteryCall(t, http.MethodPost, "/api/v1/lotteries/"+id+"/returned-code-reveals",
		"/lotteries/{lottery_id}/returned-code-reveals", session, "", nil)
}

func returnedItems(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	raw, _ := body["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, _ := item.(map[string]any)
		out = append(out, m)
	}
	return out
}

func (f *writeFix) returnedNotices(t *testing.T, id string, senderID int) int {
	t.Helper()
	return f.scalar(t, `SELECT COUNT(*) FROM message WHERE type = 'lottery-returned'
		AND receiver_id = ? AND sender_id = ? AND subject_kind = 'lottery' AND subject_id = ?`,
		w3UserAlice, senderID, asInt(id))
}

func TestV1LotteryReturnsUnassignedCodesToTheAuthor(t *testing.T) {
	f := newLotteryFix(t)
	codes := []string{"KUN-RET-ALPHA-0001", "KUN-RET-BRAVO-0002"}
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", codePrize(codes...)))
	if resp, body := f.enterLottery(t, id, "sess-bob"); resp.StatusCode != http.StatusOK {
		t.Fatalf("enter %d %+v", resp.StatusCode, body)
	}

	resp, body := f.revealReturned(t, id, "sess-alice")
	if resp.StatusCode != http.StatusOK || len(returnedItems(t, body)) != 0 {
		t.Fatalf("an open lottery has returned nothing: %d %+v", resp.StatusCode, body)
	}

	drawn := f.drawLottery(t, id, "sess-alice")
	if v := lv(drawn); asInt(v["returned_code_count"]) != 1 {
		t.Errorf("author viewer %+v", v)
	}
	for _, s := range []string{"sess-bob", "sess-staff"} {
		_, view := f.getLottery(t, id, s, "")
		if v := lv(view); asInt(v["returned_code_count"]) != 0 {
			t.Errorf("%s sees returned_code_count %v", s, v["returned_code_count"])
		}
		if resp, body := f.revealReturned(t, id, s); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s reveals the author's codes: %d %+v", s, resp.StatusCode, body)
		}
	}

	resp, body = f.revealReturned(t, id, "sess-alice")
	items := returnedItems(t, body)
	if resp.StatusCode != http.StatusOK || len(items) != 1 || items[0]["return_reason"] != "unassigned" ||
		items[0]["object"] != "lottery_returned_code" {
		t.Fatalf("returned %d %+v", resp.StatusCode, body)
	}
	_, won := f.revealCode(t, id, "sess-bob")
	if won["redemption_code"] == items[0]["redemption_code"] {
		t.Fatal("the winner's code came back to the author")
	}
	if n := f.returnedNotices(t, id, 0); n != 1 {
		t.Errorf("draw-time notices to the author: %d", n)
	}
}

func TestV1LotteryReturnsAnExpiredCodeToTheAuthor(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", codePrize("KUN-RET-EXP-0001")))
	if resp, body := f.enterLottery(t, id, "sess-bob"); resp.StatusCode != http.StatusOK {
		t.Fatalf("enter %d %+v", resp.StatusCode, body)
	}
	f.drawLottery(t, id, "sess-alice")
	if n := f.returnedNotices(t, id, 0); n != 0 {
		t.Errorf("a full draw sent %d leftover notices", n)
	}
	if err := f.db.Exec(`UPDATE topic_lottery_entry SET claim_deadline = now() - interval '1 minute'
		WHERE lottery_id = ? AND code_id <> 0`, asInt(id)).Error; err != nil {
		t.Fatal(err)
	}
	topicService.NewLotteryDrawer(f.LotteryService).Run()

	_, body := f.revealReturned(t, id, "sess-alice")
	items := returnedItems(t, body)
	if len(items) != 1 || items[0]["return_reason"] != "forfeited" || items[0]["redemption_code"] != "KUN-RET-EXP-0001" {
		t.Fatalf("returned %+v", body)
	}
	if n := f.returnedNotices(t, id, w3UserBob); n != 1 {
		t.Errorf("expiry notices to the author: %d", n)
	}
}

func TestV1LotteryCancelReturnsEveryCode(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", codePrize("KUN-RET-CXL-0001", "KUN-RET-CXL-0002")))
	resp, body := f.patchLottery(t, id, "sess-alice", map[string]any{"state": "cancelled"})
	if resp.StatusCode != http.StatusOK || asInt(lv(body)["returned_code_count"]) != 2 {
		t.Fatalf("cancel %d %+v", resp.StatusCode, body)
	}
	if _, body := f.revealReturned(t, id, "sess-alice"); len(returnedItems(t, body)) != 2 {
		t.Fatalf("returned %+v", body)
	}
}

func TestV1LotteryWinnerGivingUpTellsTheAuthor(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", offlinePrize(1)))
	if resp, body := f.enterLottery(t, id, "sess-bob"); resp.StatusCode != http.StatusOK {
		t.Fatalf("enter %d %+v", resp.StatusCode, body)
	}
	winners := winnersOf(f.drawLottery(t, id, "sess-alice"))
	if len(winners) != 1 {
		t.Fatalf("winners %+v", winners)
	}
	if resp, body := f.patchWinner(t, id, strID(winners[0]["id"]), "sess-bob", "forfeited"); resp.StatusCode != http.StatusOK {
		t.Fatalf("give up %d %+v", resp.StatusCode, body)
	}
	if n := f.returnedNotices(t, id, w3UserBob); n != 1 {
		t.Errorf("give-up notices to the author: %d", n)
	}
}
