package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestV1LotteryEnterAndWithdraw(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", offlinePrize(1)))

	resp, body := f.enterLottery(t, id, "sess-bob")
	if resp.StatusCode != http.StatusOK || lv(body)["has_entered"] != true || asInt(body["entry_count"]) != 1 {
		t.Fatalf("enter %d %+v", resp.StatusCode, body)
	}
	resp, body = f.enterLottery(t, id, "sess-bob")
	if resp.StatusCode != http.StatusOK || asInt(body["entry_count"]) != 1 {
		t.Errorf("entering again must be a no-op: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.enterLottery(t, id, "sess-alice")
	if resp.StatusCode != http.StatusForbidden || body["code"] != "LOTTERY_INELIGIBLE" || body["reason"] != "own_lottery" {
		t.Errorf("author enters own lottery %d %+v", resp.StatusCode, body)
	}

	resp, body = f.withdrawLottery(t, id, "sess-other")
	if resp.StatusCode != http.StatusOK || lv(body)["has_entered"] != false {
		t.Errorf("withdrawing without an entry must be a no-op: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.withdrawLottery(t, id, "sess-bob")
	if resp.StatusCode != http.StatusOK || asInt(body["entry_count"]) != 0 || lv(body)["has_entered"] != false {
		t.Errorf("withdraw %d %+v", resp.StatusCode, body)
	}

	if _, body = f.enterLottery(t, id, "sess-bob"); asInt(body["entry_count"]) != 1 {
		t.Fatalf("re-enter %+v", body)
	}
	if err := f.db.Exec(`UPDATE topic_lottery SET deadline = NOW() - interval '1 minute' WHERE id = ?`, id).Error; err != nil {
		t.Fatal(err)
	}
	resp, body = f.withdrawLottery(t, id, "sess-bob")
	if resp.StatusCode != http.StatusConflict || body["code"] != "LOTTERY_CLOSED" {
		t.Errorf("withdraw after closes_at %d %+v", resp.StatusCode, body)
	}
	resp, body = f.enterLottery(t, id, "sess-other")
	if resp.StatusCode != http.StatusConflict || body["code"] != "LOTTERY_CLOSED" {
		t.Errorf("enter after closes_at %d %+v", resp.StatusCode, body)
	}
}

func TestV1LotteryEntryRequirements(t *testing.T) {
	f := newLotteryFix(t)
	reply := f.mustCreateLottery(t, w3TopicFloors, "sess-alice", lotteryBody("reply", "manual", offlinePrize(1)))
	resp, body := f.enterLottery(t, reply, "sess-other")
	if resp.StatusCode != http.StatusForbidden || body["code"] != "LOTTERY_INELIGIBLE" || body["reason"] != "reply_required" {
		t.Errorf("no reply %d %+v", resp.StatusCode, body)
	}
	if _, got := f.getLottery(t, reply, "sess-other", ""); lv(got)["enter_blocked_reason"] != "reply_required" || lv(got)["can_enter"] != false {
		t.Errorf("viewer disagrees with the write: %+v", lv(got))
	}
	if resp, body = f.enterLottery(t, reply, "sess-bob"); resp.StatusCode != http.StatusOK {
		t.Errorf("bob replied on this topic: %d %+v", resp.StatusCode, body)
	}

	gated := lotteryBody("signup", "manual", offlinePrize(1))
	gated["min_moemoepoint"] = 100
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", gated)
	resp, body = f.enterLottery(t, id, "sess-bob")
	if resp.StatusCode != http.StatusForbidden || body["reason"] != "moemoepoint_below_minimum" {
		t.Errorf("below minimum %d %+v", resp.StatusCode, body)
	}

	aged := lotteryBody("signup", "manual", offlinePrize(1))
	aged["min_account_age_days"] = 30
	id = f.mustCreateLottery(t, w3TopicPub, "sess-alice", aged)
	resp, body = f.enterLottery(t, id, "sess-bob")
	if resp.StatusCode != http.StatusForbidden || body["reason"] != "account_too_new" {
		t.Errorf("unknown account age %d %+v", resp.StatusCode, body)
	}

	floor := lotteryBody("floor", "manual", offlinePrize(1))
	floor["floor_rule"] = "2"
	id = f.mustCreateLottery(t, w3TopicFloors, "sess-alice", floor)
	resp, body = f.enterLottery(t, id, "sess-bob")
	if resp.StatusCode != http.StatusForbidden || body["reason"] != "no_signup" {
		t.Errorf("floor lottery %d %+v", resp.StatusCode, body)
	}
}

func (f *writeFix) walkEntries(t *testing.T, id, session string, limit int) []string {
	t.Helper()
	var ids []string
	cursor := ""
	for page := 0; page < 50; page++ {
		q := url.Values{"limit": {fmt.Sprint(limit)}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		resp, body := f.lotteryCall(t, http.MethodGet, "/api/v1/lotteries/"+id+"/entries?"+q.Encode(),
			"/lotteries/{lottery_id}/entries", session, "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("entries page %d: %d %+v", page, resp.StatusCode, body)
		}
		items, _ := body["items"].([]any)
		for _, it := range items {
			m, _ := it.(map[string]any)
			ids = append(ids, strID(m["id"]))
		}
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return ids
		}
		cursor = next
	}
	t.Fatal("entries never ended")
	return nil
}

func TestV1LotteryEntriesWalkTiedTimestamps(t *testing.T) {
	f := newLotteryFix(t)
	body := lotteryBody("signup", "manual", offlinePrize(1))
	body["is_entry_list_public"] = false
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", body)
	tie := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	for i, uid := range []int{w3MentionMin, w3MentionMin + 1, w3MentionMin + 2, w3MentionMin + 3, w3MentionMin + 4, w3MentionMin + 5, w3MentionMin + 6} {
		at := tie
		if i == 0 {
			at = tie.Add(-time.Minute)
		}
		if i == 6 {
			at = tie.Add(time.Minute)
		}
		if err := f.db.Exec(`INSERT INTO topic_lottery_entry (lottery_id, user_id, created, updated) VALUES (?, ?, ?, ?)`,
			id, uid, at, at).Error; err != nil {
			t.Fatal(err)
		}
	}

	resp, prob := f.lotteryCall(t, http.MethodGet, "/api/v1/lotteries/"+id+"/entries", "/lotteries/{lottery_id}/entries", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusForbidden || prob["code"] != "PERMISSION_REQUIRED" {
		t.Fatalf("a private entry list for a stranger: %d %+v", resp.StatusCode, prob)
	}

	var want []string
	rows, err := f.db.Raw(`SELECT id::text FROM topic_lottery_entry WHERE lottery_id = ? ORDER BY created, id`, id).Rows()
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		want = append(want, s)
	}
	_ = rows.Close()
	for _, limit := range []int{1, 2, 3} {
		if got := f.walkEntries(t, id, "sess-alice", limit); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
}

func TestV1LotteryDrawSettlesTheEscrow(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", pointPrize("fixed", 10, 3)))
	if _, body := f.enterLottery(t, id, "sess-bob"); asInt(body["entry_count"]) != 1 {
		t.Fatalf("enter %+v", body)
	}
	if _, before := f.getLottery(t, id, "sess-bob", ""); before["seed"] != nil {
		t.Fatalf("the seed leaked before the draw: %v", before["seed"])
	}

	drawn := f.drawLottery(t, id, "sess-alice")
	if drawn["state"] != "drawn" || drawn["seed"] == nil || drawn["drawn_at"] == nil {
		t.Fatalf("drawn %+v", drawn)
	}
	winners := winnersOf(drawn)
	if len(winners) != 1 || asInt(winners[0]["point_awarded"]) != 10 || winners[0]["fulfillment"] != "received" {
		t.Fatalf("winners %+v", winners)
	}
	if winners[0]["rank_key"] == nil || winners[0]["winning_floor"] != nil {
		t.Errorf("a signup winner has a rank key and no floor: %+v", winners[0])
	}
	if got := f.scalar(t, `SELECT point_escrow FROM topic_lottery WHERE id = ?`, id); got != 0 {
		t.Errorf("escrow left after the draw: %d", got)
	}
	if got := f.scalar(t, `SELECT moemoepoint FROM kungal_user_state WHERE user_id = ?`, w3UserAlice); got != 990 {
		t.Errorf("cached balance %d after paying 30 and getting 20 back, want 990", got)
	}
	var refunds []awardCall
	for _, a := range f.lotteryAwards(id) {
		if a.ref == "topic_lottery_escrow:"+id && a.delta > 0 {
			refunds = append(refunds, a)
		}
	}
	if len(refunds) != 1 || refunds[0].delta != 20 || refunds[0].userID != w3UserAlice ||
		refunds[0].key != "kungal:lottery_escrow_refund:topic_lottery_"+id {
		t.Errorf("30 escrowed, 10 paid, 20 back: %+v", refunds)
	}
	paid := 0
	for _, a := range f.snapshotAwards() {
		if a.ref == "topic_lottery:"+id && a.userID == w3UserBob {
			paid += a.delta
		}
	}
	if paid != 10 {
		t.Errorf("winner paid %d", paid)
	}
	if v := lv(drawn); v["can_draw"] != false || v["can_manage_fulfillment"] != true {
		t.Errorf("author viewer after the draw %+v", v)
	}

	resp, body := f.patchLottery(t, id, "sess-alice", map[string]any{"state": "cancelled"})
	if resp.StatusCode != http.StatusConflict || body["code"] != "INVALID_STATE_TRANSITION" {
		t.Errorf("cancel a drawn lottery %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchLottery(t, id, "sess-alice", map[string]any{"state": "drawn"})
	if resp.StatusCode != http.StatusConflict || body["code"] != "INVALID_STATE_TRANSITION" {
		t.Errorf("draw twice %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchLottery(t, id, "sess-alice", map[string]any{"title": "late"})
	if resp.StatusCode != http.StatusConflict || body["code"] != "LOTTERY_CLOSED" {
		t.Errorf("edit a drawn lottery %d %+v", resp.StatusCode, body)
	}
	resp, body = f.deleteLottery(t, id, "sess-alice")
	if resp.StatusCode != http.StatusConflict || body["code"] != "LOTTERY_DRAWN" {
		t.Errorf("author deletes a drawn lottery %d %+v", resp.StatusCode, body)
	}
	if resp, _ = f.deleteLottery(t, id, "sess-staff"); resp.StatusCode != http.StatusNoContent {
		t.Errorf("staff deletes a drawn lottery %d", resp.StatusCode)
	}
	if n := len(f.lotteryAwards(id)); n != 3 {
		t.Errorf("deleting a settled lottery must not refund again: %d awards", n)
	}
}

func TestV1LotteryCancelRefundsOnce(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", pointPrize("split", 20, 2)))
	resp, body := f.patchLottery(t, id, "sess-alice", map[string]any{"state": "cancelled", "title": "x"})
	if resp.StatusCode != http.StatusUnprocessableEntity || fieldPointers(body)["/state"] != "INCONSISTENT_WITH" {
		t.Errorf("state with another field %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchLottery(t, id, "sess-bob", map[string]any{"state": "cancelled"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("stranger cancels %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchLottery(t, id, "sess-alice", map[string]any{"state": "cancelled"})
	if resp.StatusCode != http.StatusOK || body["state"] != "cancelled" {
		t.Fatalf("cancel %d %+v", resp.StatusCode, body)
	}
	if resp, _ = f.deleteLottery(t, id, "sess-alice"); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete a cancelled lottery %d", resp.StatusCode)
	}
	var refunded int
	for _, a := range f.lotteryAwards(id) {
		if a.delta > 0 {
			refunded += a.delta
		}
	}
	if refunded != 20 {
		t.Errorf("20 escrowed, %d refunded across cancel and delete", refunded)
	}
	if got := f.scalar(t, `SELECT moemoepoint FROM kungal_user_state WHERE user_id = ?`, w3UserAlice); got != 1000 {
		t.Errorf("cached balance %d after a full refund", got)
	}

	open := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", pointPrize("fixed", 5, 1)))
	if resp, _ = f.deleteLottery(t, open, "sess-alice"); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete an open lottery %d", resp.StatusCode)
	}
	if awards := f.lotteryAwards(open); len(awards) != 2 || awards[1].delta != 5 {
		t.Errorf("an open lottery's escrow comes back on delete: %+v", awards)
	}
}

func TestV1LotteryPatch(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", pointPrize("fixed", 10, 1)))

	resp, body := f.patchLottery(t, id, "sess-alice", map[string]any{"draw_mode": "deadline", "closes_at": nil})
	if resp.StatusCode != http.StatusUnprocessableEntity || fieldPointers(body)["/closes_at"] != "REQUIRED" {
		t.Errorf("a patch without prizes still gets the whole shape checked: %d %+v", resp.StatusCode, body)
	}

	resp, body = f.patchLottery(t, id, "sess-alice", map[string]any{"prizes": []any{pointPrize("fixed", 10, 4)}})
	if resp.StatusCode != http.StatusOK || asInt(body["slot_count"]) != 4 {
		t.Fatalf("raise the budget %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchLottery(t, id, "sess-alice", map[string]any{"prizes": []any{pointPrize("fixed", 10, 2)}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("lower the budget %d %+v", resp.StatusCode, body)
	}
	var deltas []int
	for _, a := range f.lotteryAwards(id) {
		deltas = append(deltas, a.delta)
	}
	if fmt.Sprint(deltas) != "[-10 -30 20]" {
		t.Errorf("escrow moves %v, want [-10 -30 20]", deltas)
	}
	if got := f.scalar(t, `SELECT point_escrow FROM topic_lottery WHERE id = ?`, id); got != 20 {
		t.Errorf("point_escrow %d", got)
	}

	f.setMoemoepoint(t, w3UserAlice, 5)
	resp, body = f.patchLottery(t, id, "sess-alice", map[string]any{"prizes": []any{pointPrize("fixed", 10, 3)}})
	if resp.StatusCode != http.StatusForbidden || body["code"] != "MOEMOEPOINT_INSUFFICIENT" || asInt(body["required"]) != 10 {
		t.Errorf("an uncovered raise %d %+v", resp.StatusCode, body)
	}

	if _, got := f.enterLottery(t, id, "sess-bob"); asInt(got["entry_count"]) != 1 {
		t.Fatalf("enter %+v", got)
	}
	resp, body = f.patchLottery(t, id, "sess-alice", map[string]any{"prizes": []any{offlinePrize(1)}})
	if resp.StatusCode != http.StatusUnprocessableEntity || fieldPointers(body)["/prizes"] != "IMMUTABLE" {
		t.Errorf("prizes after an entry %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchLottery(t, id, "sess-alice", map[string]any{"entry_mode": "reply"})
	if resp.StatusCode != http.StatusUnprocessableEntity || fieldPointers(body)["/entry_mode"] != "IMMUTABLE" {
		t.Errorf("entry_mode after an entry %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchLottery(t, id, "sess-alice", map[string]any{"title": "renamed", "is_entry_list_public": false})
	if resp.StatusCode != http.StatusOK || body["title"] != "renamed" || body["is_entry_list_public"] != false {
		t.Errorf("scalar edit after an entry %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchLottery(t, id, "sess-bob", map[string]any{"title": "mine"})
	if resp.StatusCode != http.StatusForbidden || body["code"] != "PERMISSION_REQUIRED" {
		t.Errorf("stranger edits %d %+v", resp.StatusCode, body)
	}
}

func TestV1LotteryCodeReveal(t *testing.T) {
	f := newLotteryFix(t)
	codes := []string{"KUN-CODE-ALPHA-7781", "KUN-CODE-BRAVO-3352"}
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", codePrize(codes[0])))
	for _, s := range []string{"sess-bob", "sess-other"} {
		if resp, body := f.enterLottery(t, id, s); resp.StatusCode != http.StatusOK {
			t.Fatalf("enter %s %d %+v", s, resp.StatusCode, body)
		}
	}
	drawn := f.drawLottery(t, id, "sess-alice")
	winners := winnersOf(drawn)
	if len(winners) != 1 || winners[0]["claim_expires_at"] == nil {
		t.Fatalf("winners %+v", winners)
	}
	winnerSess, loserSess := "sess-bob", "sess-other"
	winnerRef, _ := winners[0]["winner"].(map[string]any)
	if strID(winnerRef["id"]) == fmt.Sprint(w3UserOther) {
		winnerSess, loserSess = "sess-other", "sess-bob"
	}

	resp, body := f.revealCode(t, id, loserSess)
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Errorf("an entrant who lost reveals %d %+v", resp.StatusCode, body)
	}
	if strings.Contains(fmt.Sprint(body), codes[0]) {
		t.Fatal("a loser saw the code")
	}
	_, loserView := f.getLottery(t, id, loserSess, "")
	if v := lv(loserView); v["can_reveal_code"] != false || v["winner_id"] != nil {
		t.Errorf("loser viewer %+v", v)
	}
	_, winnerView := f.getLottery(t, id, winnerSess, "")
	if v := lv(winnerView); v["can_reveal_code"] != true || strID(v["winner_id"]) != strID(winners[0]["id"]) {
		t.Errorf("winner viewer %+v", v)
	}

	for i := 0; i < 2; i++ {
		resp, body = f.revealCode(t, id, winnerSess)
		if resp.StatusCode != http.StatusOK || body["redemption_code"] != codes[0] || body["object"] != "lottery_code_reveal" {
			t.Fatalf("reveal %d: %d %+v", i, resp.StatusCode, body)
		}
	}
	if got := f.scalar(t, `SELECT COUNT(*) FROM topic_lottery_entry WHERE lottery_id = ? AND fulfillment = 'received'`, id); got != 1 {
		t.Errorf("a revealed code is received: %d", got)
	}

	reads := []func() []byte{
		func() []byte {
			_, b := f.doJSON(t, http.MethodGet, "/api/v1/lotteries/"+id, winnerSess, "/lotteries/{lottery_id}", "", nil, nil)
			return b
		},
		func() []byte {
			_, b := f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d/lotteries", w3TopicPub), winnerSess, "/topics/{topic_id}/lotteries", "", nil, nil)
			return b
		},
		func() []byte {
			_, b := f.doJSON(t, http.MethodGet, "/api/v1/lotteries/"+id+"/entries", "sess-alice", "/lotteries/{lottery_id}/entries", "", nil, nil)
			return b
		},
		func() []byte {
			_, b := f.doJSON(t, http.MethodGet, "/api/v1/lotteries/"+id+"/winners/"+strID(winners[0]["id"]), winnerSess,
				"/lotteries/{lottery_id}/winners/{winner_id}", "", nil, nil)
			return b
		},
	}
	for i, read := range reads {
		if strings.Contains(string(read()), codes[0]) {
			t.Errorf("read %d carries a redemption code", i)
		}
	}

	resp, body = f.patchWinner(t, id, strID(winners[0]["id"]), "sess-alice", "shipped")
	if resp.StatusCode != http.StatusConflict || body["code"] != "INVALID_STATE_TRANSITION" {
		t.Errorf("a code prize is moved by hand %d %+v", resp.StatusCode, body)
	}
	if err := f.db.Exec(`UPDATE topic_lottery_entry SET fulfillment = 'forfeited' WHERE lottery_id = ? AND prize_id > 0`, id).Error; err != nil {
		t.Fatal(err)
	}
	resp, body = f.revealCode(t, id, winnerSess)
	if resp.StatusCode != http.StatusConflict || body["code"] != "REDEMPTION_CODE_FORFEITED" {
		t.Errorf("reveal a forfeited code %d %+v", resp.StatusCode, body)
	}
}

func TestV1LotteryFulfillment(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", offlinePrize(1)))
	if resp, body := f.enterLottery(t, id, "sess-bob"); resp.StatusCode != http.StatusOK {
		t.Fatalf("enter %d %+v", resp.StatusCode, body)
	}
	winners := winnersOf(f.drawLottery(t, id, "sess-alice"))
	if len(winners) != 1 || winners[0]["fulfillment"] != "pending" {
		t.Fatalf("winners %+v", winners)
	}
	winner := strID(winners[0]["id"])

	steps := []struct {
		session, target string
		status          int
		code            string
	}{
		{"sess-bob", "shipped", http.StatusForbidden, "PERMISSION_REQUIRED"},
		{"sess-other", "received", http.StatusForbidden, "PERMISSION_REQUIRED"},
		{"sess-alice", "shipped", http.StatusOK, ""},
		{"sess-alice", "shipped", http.StatusOK, ""},
		{"sess-alice", "pending", http.StatusOK, ""},
		{"sess-bob", "received", http.StatusOK, ""},
		{"sess-alice", "pending", http.StatusConflict, "INVALID_STATE_TRANSITION"},
		{"sess-bob", "forfeited", http.StatusConflict, "INVALID_STATE_TRANSITION"},
	}
	for i, s := range steps {
		resp, body := f.patchWinner(t, id, winner, s.session, s.target)
		if resp.StatusCode != s.status || (s.code != "" && body["code"] != s.code) {
			t.Fatalf("step %d %s→%s: %d %+v", i, s.session, s.target, resp.StatusCode, body)
		}
		if s.status == http.StatusOK && body["fulfillment"] != s.target {
			t.Fatalf("step %d fulfillment %v", i, body["fulfillment"])
		}
	}
}
