package app

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func lv(body map[string]any) map[string]any {
	v, _ := body["viewer"].(map[string]any)
	return v
}

func TestV1LotteryCreateChargesTheAuthor(t *testing.T) {
	f := newLotteryFix(t)
	payload := lotteryBody("signup", "manual", offlinePrize(1), pointPrize("fixed", 10, 2))
	resp, body := f.createLottery(t, w3TopicPub, "sess-alice", payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id := strID(body["id"])
	if got := resp.Header.Get("Location"); got != "/api/v1/lotteries/"+id {
		t.Errorf("Location %q", got)
	}
	if body["object"] != "lottery" || body["state"] != "open" || asInt(body["slot_count"]) != 3 {
		t.Fatalf("lottery %+v", body)
	}
	if body["seed"] != nil || body["seed_hash"] == nil {
		t.Errorf("seed %v seed_hash %v", body["seed"], body["seed_hash"])
	}
	prizes, _ := body["prizes"].([]any)
	point, _ := prizes[1].(map[string]any)
	if asInt(point["point_budget"]) != 20 || point["delivery"] != "point" {
		t.Errorf("point prize %+v", point)
	}
	offline, _ := prizes[0].(map[string]any)
	if offline["delivery"] != "offline" || offline["point_mode"] != nil || offline["code_count"] != nil {
		t.Errorf("offline prize %+v", offline)
	}

	awards := f.lotteryAwards(id)
	if len(awards) != 1 {
		t.Fatalf("awards %+v", awards)
	}
	a := awards[0]
	if a.userID != w3UserAlice || a.delta != -20 || a.reason != "content_removed" ||
		a.ref != "topic_lottery_escrow:"+id || a.key != "kungal:lottery_escrow:topic_lottery_"+id {
		t.Errorf("escrow award %+v", a)
	}
	if got := f.scalar(t, `SELECT moemoepoint FROM kungal_user_state WHERE user_id = ?`, w3UserAlice); got != 980 {
		t.Errorf("cached balance %d, want 980", got)
	}
	if got := f.scalar(t, `SELECT point_escrow FROM topic_lottery WHERE id = ?`, id); got != 20 {
		t.Errorf("point_escrow %d", got)
	}

	resp, anon := f.getLottery(t, id, "", "")
	if resp.StatusCode != http.StatusOK || anon["viewer"] != nil {
		t.Errorf("anonymous %d viewer %v", resp.StatusCode, anon["viewer"])
	}
	v := lv(body)
	if v["can_edit"] != true || v["can_draw"] != true || v["can_enter"] != false || v["enter_blocked_reason"] != "own_lottery" {
		t.Errorf("author viewer %+v", v)
	}
}

func TestV1LotteryCreateRefusesAnUncoveredBudget(t *testing.T) {
	f := newLotteryFix(t)
	f.setMoemoepoint(t, w3UserAlice, 150)
	resp, body := f.createLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", pointPrize("split", 500, 2)))
	if resp.StatusCode != http.StatusForbidden || body["code"] != "MOEMOEPOINT_INSUFFICIENT" || asInt(body["required"]) != 500 {
		t.Fatalf("uncovered budget %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_lottery WHERE topic_id = ?`, w3TopicPub); n != 0 {
		t.Errorf("a refused lottery was stored: %d rows", n)
	}
	if got := f.scalar(t, `SELECT moemoepoint FROM kungal_user_state WHERE user_id = ?`, w3UserAlice); got != 150 {
		t.Errorf("balance moved to %d", got)
	}
	if len(f.snapshotAwards()) != 0 {
		t.Errorf("awards %+v", f.snapshotAwards())
	}
}

func TestV1LotteryCreateGates(t *testing.T) {
	f := newLotteryFix(t)
	resp, body := f.createLottery(t, w3TopicPub, "sess-bob", lotteryBody("signup", "manual", offlinePrize(1)))
	if resp.StatusCode != http.StatusForbidden || body["code"] != "PERMISSION_REQUIRED" {
		t.Errorf("not the topic's author %d %+v", resp.StatusCode, body)
	}
	f.setMoemoepoint(t, w3UserAlice, 30)
	resp, body = f.createLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", offlinePrize(1)))
	if resp.StatusCode != http.StatusForbidden || body["code"] != "LOTTERY_CREATOR_INELIGIBLE" ||
		asInt(body["min_moemoepoint"]) != 100 || asInt(body["min_account_age_days"]) != 30 {
		t.Errorf("ineligible creator %d %+v", resp.StatusCode, body)
	}
	resp, body = f.createLottery(t, w3TopicPub, "sess-staff", lotteryBody("signup", "manual", offlinePrize(1)))
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("staff on someone else's topic %d %+v", resp.StatusCode, body)
	}
	resp, body = f.createLottery(t, w3TopicPub, "", lotteryBody("signup", "manual", offlinePrize(1)))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous %d %+v", resp.StatusCode, body)
	}
}

func TestV1LotteryCreateLimit(t *testing.T) {
	f := newLotteryFix(t)
	for i := 0; i < 10; i++ {
		f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", offlinePrize(1)))
	}
	resp, body := f.createLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", offlinePrize(1)))
	if resp.StatusCode != http.StatusUnprocessableEntity || fieldPointers(body)["topic_id"] != "TOO_MANY_ITEMS" {
		t.Fatalf("eleventh lottery %d %+v", resp.StatusCode, body)
	}
}

func TestV1LotteryCreateShapeErrors(t *testing.T) {
	f := newLotteryFix(t)
	cases := []struct {
		name    string
		payload map[string]any
		pointer string
		reason  string
	}{
		{"deadline without closes_at", lotteryBody("signup", "deadline", offlinePrize(1)), "/closes_at", "REQUIRED"},
		{"closes_at in the past", func() map[string]any {
			b := lotteryBody("signup", "deadline", offlinePrize(1))
			b["closes_at"] = future(-time.Hour)
			return b
		}(), "/closes_at", "OUT_OF_RANGE"},
		{"threshold below slots", func() map[string]any {
			b := lotteryBody("signup", "threshold", offlinePrize(3))
			b["draw_threshold"] = 2
			return b
		}(), "/draw_threshold", "INCONSISTENT_WITH"},
		{"floor with threshold", func() map[string]any {
			b := lotteryBody("floor", "threshold", offlinePrize(1))
			b["floor_rule"], b["draw_threshold"] = "8", 5
			return b
		}(), "/draw_mode", "INCONSISTENT_WITH"},
		{"floor rule count", func() map[string]any {
			b := lotteryBody("floor", "manual", offlinePrize(2))
			b["floor_rule"] = "8"
			return b
		}(), "/floor_rule", "INCONSISTENT_WITH"},
		{"floor rule format", func() map[string]any {
			b := lotteryBody("floor", "manual", offlinePrize(1))
			b["floor_rule"] = "every:x"
			return b
		}(), "/floor_rule", "INVALID_FORMAT"},
		{"floor with a random pool", func() map[string]any {
			b := lotteryBody("floor", "manual", pointPrize("random", 10, 2))
			b["floor_rule"] = "every:5"
			return b
		}(), "/prizes/0/point_mode", "INCONSISTENT_WITH"},
		{"codes do not match slots", func() map[string]any {
			p := codePrize("A", "B")
			p["slot_count"] = 3
			return lotteryBody("signup", "manual", p)
		}(), "/prizes/0/codes", "INCONSISTENT_WITH"},
		{"point prize without mode", lotteryBody("signup", "manual",
			map[string]any{"title": "p", "delivery": "point", "point_amount": 5, "slot_count": 1}), "/prizes/0/point_mode", "REQUIRED"},
		{"pool below slots", lotteryBody("signup", "manual", pointPrize("split", 2, 3)), "/prizes/0/point_amount", "INCONSISTENT_WITH"},
		{"codes on an offline prize", lotteryBody("signup", "manual",
			map[string]any{"title": "p", "delivery": "offline", "slot_count": 1, "codes": []string{"A"}}), "/prizes/0/codes", "NOT_ALLOWED_VALUE"},
		{"adult image not among images", lotteryBody("signup", "manual",
			map[string]any{"title": "p", "delivery": "offline", "slot_count": 1,
				"image_hashes": []string{w3CoverHash}, "adult_image_hashes": []string{gradedExplicitHash}}),
			"/prizes/0/adult_image_hashes", "INCONSISTENT_WITH"},
		{"no winners", lotteryBody("signup", "manual", offlinePrize(0)), "/prizes/0/slot_count", "OUT_OF_RANGE"},
		{"blank prize title", lotteryBody("signup", "manual",
			map[string]any{"title": "   ", "delivery": "offline", "slot_count": 1}), "/prizes/0/title", "TOO_SHORT"},
	}
	for _, c := range cases {
		resp, body := f.createLottery(t, w3TopicPub, "sess-alice", c.payload)
		if resp.StatusCode != http.StatusUnprocessableEntity || body["code"] != "VALIDATION_FAILED" {
			t.Errorf("%s: %d %+v", c.name, resp.StatusCode, body)
			continue
		}
		if got := fieldPointers(body)[c.pointer]; got != c.reason {
			t.Errorf("%s: %s is %q, want %q (%+v)", c.name, c.pointer, got, c.reason, body["errors"])
		}
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_lottery WHERE topic_id = ?`, w3TopicPub); n != 0 {
		t.Errorf("a refused lottery was stored: %d rows", n)
	}
}

func TestV1LotteryIsInvisibleWithItsTopic(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicRole, "sess-alice", lotteryBody("signup", "manual", offlinePrize(1)))
	if resp, _ := f.getLottery(t, id, "sess-alice", ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("author reads %d", resp.StatusCode)
	}
	for _, sess := range []string{"sess-bob", ""} {
		resp, body := f.getLottery(t, id, sess, "")
		if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
			t.Errorf("%q reads a lottery under an unreadable topic: %d %+v", sess, resp.StatusCode, body)
		}
		resp, _ = f.lotteryCall(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d/lotteries", w3TopicRole),
			"/topics/{topic_id}/lotteries", sess, "", nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%q lists an unreadable topic's lotteries: %d", sess, resp.StatusCode)
		}
		resp, _ = f.enterLottery(t, id, sess)
		if sess != "" && resp.StatusCode != http.StatusNotFound {
			t.Errorf("%q enters a lottery under an unreadable topic: %d", sess, resp.StatusCode)
		}
	}
}

func TestV1LotteryListIsNewestFirst(t *testing.T) {
	f := newLotteryFix(t)
	a := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", offlinePrize(1)))
	b := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", offlinePrize(1)))
	if err := f.db.Exec(`UPDATE topic_lottery SET created = '2026-07-01T00:00:00Z' WHERE id IN (?, ?)`, a, b).Error; err != nil {
		t.Fatal(err)
	}
	resp, body := f.lotteryCall(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d/lotteries", w3TopicPub),
		"/topics/{topic_id}/lotteries", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	var ids []string
	for _, it := range items {
		m, _ := it.(map[string]any)
		ids = append(ids, strID(m["id"]))
	}
	if fmt.Sprint(ids) != fmt.Sprint([]string{b, a}) {
		t.Errorf("same created, want id desc: %v", ids)
	}
	if body["next_cursor"] != nil {
		t.Errorf("an unpaginated list has no next_cursor")
	}
}

func TestV1LotteryWithholdsAdultPrizeImages(t *testing.T) {
	f := newLotteryFix(t)
	id := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual",
		map[string]any{"title": "p", "delivery": "offline", "slot_count": 1,
			"image_hashes":       []string{strings.Repeat("a", 64), strings.Repeat("b", 64), gradedExplicitHash},
			"adult_image_hashes": []string{strings.Repeat("b", 64)}}))

	images := func(query string) []map[string]any {
		t.Helper()
		resp, body := f.getLottery(t, id, "sess-bob", query)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get %d", resp.StatusCode)
		}
		prizes, _ := body["prizes"].([]any)
		p, _ := prizes[0].(map[string]any)
		raw, _ := p["images"].([]any)
		out := make([]map[string]any, len(raw))
		for i, r := range raw {
			out[i], _ = r.(map[string]any)
		}
		return out
	}
	sfw := images("")
	if len(sfw) != 3 {
		t.Fatalf("every hash stays, so an edit form writes the gallery back whole: %+v", sfw)
	}
	if sfw[0]["image"] == nil {
		t.Errorf("a safe image was withheld: %+v", sfw[0])
	}
	if sfw[1]["image"] != nil || sfw[1]["is_marked_adult"] != true || sfw[1]["hash"] != strings.Repeat("b", 64) {
		t.Errorf("an author-marked image reached a SFW reader: %+v", sfw[1])
	}
	if sfw[2]["image"] != nil || sfw[2]["is_graded_explicit"] != true || sfw[2]["is_marked_adult"] != false {
		t.Errorf("a graded-explicit image reached a SFW reader: %+v", sfw[2])
	}
	for i, img := range images("?include_nsfw=true") {
		if img["image"] == nil {
			t.Errorf("include_nsfw=true still withheld image %d", i)
		}
	}
}
