package app

import (
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var lotteryKeySeq atomic.Int64

func nextLotteryKey() string {
	return keyUUID(int(900000 + lotteryKeySeq.Add(1)))
}

func newLotteryFix(t *testing.T) *writeFix {
	t.Helper()
	f := newWriteFix(t, nil)
	f.alice(t)
	f.setMoemoepoint(t, w3UserAlice, 1000)
	return f
}

func (f *writeFix) setMoemoepoint(t *testing.T, userID, n int) {
	t.Helper()
	if err := f.db.Exec(`UPDATE kungal_user_state SET moemoepoint = ? WHERE user_id = ?`, n, userID).Error; err != nil {
		t.Fatal(err)
	}
}

func (f *writeFix) lotteryCall(t *testing.T, method, rawURL, specPath, session, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, method, rawURL, session, specPath, idem, nil, payload)
	if len(body) == 0 {
		return resp, nil
	}
	return resp, problemMap(t, body)
}

func (f *writeFix) createLottery(t *testing.T, topicID int, session string, payload map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	return f.lotteryCall(t, http.MethodPost, "/api/v1/topics/"+strconv.Itoa(topicID)+"/lotteries",
		"/topics/{topic_id}/lotteries", session, nextLotteryKey(), payload)
}

// mustCreateLottery returns the new lottery's id.
func (f *writeFix) mustCreateLottery(t *testing.T, topicID int, session string, payload map[string]any) string {
	t.Helper()
	resp, body := f.createLottery(t, topicID, session, payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create lottery %d %+v", resp.StatusCode, body)
	}
	return strID(body["id"])
}

func (f *writeFix) getLottery(t *testing.T, id, session, query string) (*http.Response, map[string]any) {
	t.Helper()
	return f.lotteryCall(t, http.MethodGet, "/api/v1/lotteries/"+id+query, "/lotteries/{lottery_id}", session, "", nil)
}

func (f *writeFix) patchLottery(t *testing.T, id, session string, payload map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	return f.lotteryCall(t, http.MethodPatch, "/api/v1/lotteries/"+id, "/lotteries/{lottery_id}", session, "", payload)
}

func (f *writeFix) deleteLottery(t *testing.T, id, session string) (*http.Response, map[string]any) {
	t.Helper()
	return f.lotteryCall(t, http.MethodDelete, "/api/v1/lotteries/"+id, "/lotteries/{lottery_id}", session, "", nil)
}

func (f *writeFix) enterLottery(t *testing.T, id, session string) (*http.Response, map[string]any) {
	t.Helper()
	return f.lotteryCall(t, http.MethodPut, "/api/v1/lotteries/"+id+"/entries/me", "/lotteries/{lottery_id}/entries/me", session, "", nil)
}

func (f *writeFix) withdrawLottery(t *testing.T, id, session string) (*http.Response, map[string]any) {
	t.Helper()
	return f.lotteryCall(t, http.MethodDelete, "/api/v1/lotteries/"+id+"/entries/me", "/lotteries/{lottery_id}/entries/me", session, "", nil)
}

func (f *writeFix) revealCode(t *testing.T, id, session string) (*http.Response, map[string]any) {
	t.Helper()
	return f.lotteryCall(t, http.MethodPost, "/api/v1/lotteries/"+id+"/code-reveals", "/lotteries/{lottery_id}/code-reveals", session, "", nil)
}

func (f *writeFix) patchWinner(t *testing.T, lotteryID, winnerID, session, fulfillment string) (*http.Response, map[string]any) {
	t.Helper()
	return f.lotteryCall(t, http.MethodPatch, "/api/v1/lotteries/"+lotteryID+"/winners/"+winnerID,
		"/lotteries/{lottery_id}/winners/{winner_id}", session, "", map[string]any{"fulfillment": fulfillment})
}

func (f *writeFix) drawLottery(t *testing.T, id, session string) map[string]any {
	t.Helper()
	resp, body := f.patchLottery(t, id, session, map[string]any{"state": "drawn"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("draw %d %+v", resp.StatusCode, body)
	}
	return body
}

func lotteryBody(entryMode, drawMode string, prizes ...map[string]any) map[string]any {
	return map[string]any{
		"title":      "lottery",
		"entry_mode": entryMode,
		"draw_mode":  drawMode,
		"prizes":     prizes,
	}
}

func offlinePrize(slots int) map[string]any {
	return map[string]any{"title": "offline prize", "delivery": "offline", "slot_count": slots}
}

func pointPrize(mode string, amount, slots int) map[string]any {
	return map[string]any{"title": "point prize", "delivery": "point", "point_mode": mode, "point_amount": amount, "slot_count": slots}
}

func codePrize(codes ...string) map[string]any {
	return map[string]any{"title": "code prize", "delivery": "code", "slot_count": len(codes), "codes": codes}
}

func future(d time.Duration) string {
	return time.Now().Add(d).UTC().Format("2006-01-02T15:04:05Z")
}

func (f *writeFix) lotteryAwards(id string) []awardCall {
	var out []awardCall
	for _, a := range f.snapshotAwards() {
		if strings.HasSuffix(a.ref, ":"+id) && strings.HasPrefix(a.ref, "topic_lottery") {
			out = append(out, a)
		}
	}
	return out
}

func winnersOf(body map[string]any) []map[string]any {
	raw, _ := body["winners"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, w := range raw {
		if m, ok := w.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func fieldPointers(body map[string]any) map[string]string {
	out := map[string]string{}
	raw, _ := body["errors"].([]any)
	for _, e := range raw {
		m, _ := e.(map[string]any)
		ptr, _ := m["pointer"].(string)
		if ptr == "" {
			ptr, _ = m["parameter"].(string)
		}
		reason, _ := m["reason"].(string)
		out[ptr] = reason
	}
	return out
}
