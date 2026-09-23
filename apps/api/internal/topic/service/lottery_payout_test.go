package service

import (
	"testing"

	topicModel "kun-galgame-api/internal/topic/model"
)

func winnersWithIDs(ids ...int) []topicModel.TopicLotteryEntry {
	out := make([]topicModel.TopicLotteryEntry, len(ids))
	for i, id := range ids {
		out[i] = topicModel.TopicLotteryEntry{UserID: id}
	}
	return out
}

func sum(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}

func TestPointPayoutsFixedIgnoresWinnerCount(t *testing.T) {
	prize := topicModel.TopicLotteryPrize{
		ID: 1, PointMode: topicModel.LotteryPointFixed, PointAmount: 50, Slots: 5,
	}
	got := pointPayouts("seed", prize, winnersWithIDs(7, 8))
	for _, amount := range got {
		if amount != 50 {
			t.Fatalf("fixed payout = %v, want every winner on 50", got)
		}
	}
}

func TestPointPayoutsSplitSpendsThePoolExactly(t *testing.T) {
	prize := topicModel.TopicLotteryPrize{
		ID: 1, PointMode: topicModel.LotteryPointSplit, PointAmount: 1000, Slots: 3,
	}
	got := pointPayouts("seed", prize, winnersWithIDs(1, 2, 3))
	if sum(got) != 1000 {
		t.Fatalf("split paid out %d of a 1000 pool (%v)", sum(got), got)
	}
	// 1000/3 leaves one point over; it goes to the best-ranked winner.
	if got[0] != 334 || got[1] != 333 || got[2] != 333 {
		t.Fatalf("split remainder = %v, want [334 333 333]", got)
	}
}

// Fewer winners than slots is the case a pool exists for: the budget is still
// spent, the shares just get bigger.
func TestPointPayoutsSplitWithUnfilledSlots(t *testing.T) {
	prize := topicModel.TopicLotteryPrize{
		ID: 1, PointMode: topicModel.LotteryPointSplit, PointAmount: 900, Slots: 9,
	}
	got := pointPayouts("seed", prize, winnersWithIDs(1, 2))
	if sum(got) != 900 || got[0] != 450 {
		t.Fatalf("split with 2 of 9 slots = %v, want [450 450]", got)
	}
}

func TestPointPayoutsRandomSpendsThePoolAndFloorsAtOne(t *testing.T) {
	prize := topicModel.TopicLotteryPrize{
		ID: 42, PointMode: topicModel.LotteryPointRandom, PointAmount: 500, Slots: 6,
	}
	winners := winnersWithIDs(11, 22, 33, 44, 55, 66)
	got := pointPayouts("a-revealed-seed", prize, winners)
	if sum(got) != 500 {
		t.Fatalf("random paid out %d of a 500 pool (%v)", sum(got), got)
	}
	for _, amount := range got {
		if amount < 1 {
			t.Fatalf("random payout %v left a winner with nothing", got)
		}
	}
}

// The whole point of deriving the split from the revealed seed is that anyone
// can recompute it; a payout that moved between runs would not be checkable.
func TestPointPayoutsRandomIsReproducibleFromTheSeed(t *testing.T) {
	prize := topicModel.TopicLotteryPrize{
		ID: 42, PointMode: topicModel.LotteryPointRandom, PointAmount: 777, Slots: 4,
	}
	winners := winnersWithIDs(11, 22, 33, 44)
	first := pointPayouts("seed-one", prize, winners)
	again := pointPayouts("seed-one", prize, winners)
	other := pointPayouts("seed-two", prize, winners)

	for i := range first {
		if first[i] != again[i] {
			t.Fatalf("same seed gave %v then %v", first, again)
		}
	}
	same := true
	for i := range first {
		if first[i] != other[i] {
			same = false
		}
	}
	if same {
		t.Fatalf("a different seed produced the same split %v", first)
	}
}

// A pooled prize is shared among the winners of THAT prize, so two point prizes
// in one lottery must not bleed into each other.
func TestStampPointPayoutsGroupsByPrize(t *testing.T) {
	big := topicModel.TopicLotteryPrize{
		ID: 1, Delivery: topicModel.LotteryDeliveryPoint,
		PointMode: topicModel.LotteryPointSplit, PointAmount: 600, Slots: 2,
	}
	small := topicModel.TopicLotteryPrize{
		ID: 2, Delivery: topicModel.LotteryDeliveryPoint,
		PointMode: topicModel.LotteryPointFixed, PointAmount: 10, Slots: 1,
	}
	goods := topicModel.TopicLotteryPrize{
		ID: 3, Delivery: topicModel.LotteryDeliveryManual, Slots: 1,
	}
	slots := []topicModel.TopicLotteryPrize{big, big, small, goods}
	winners := winnersWithIDs(1, 2, 3, 4)

	stampPointPayouts("seed", slots, winners)

	if winners[0].PointAwarded != 300 || winners[1].PointAwarded != 300 {
		t.Fatalf("pooled prize split as %d/%d, want 300/300",
			winners[0].PointAwarded, winners[1].PointAwarded)
	}
	if winners[2].PointAwarded != 10 {
		t.Fatalf("fixed prize paid %d, want 10", winners[2].PointAwarded)
	}
	if winners[3].PointAwarded != 0 {
		t.Fatalf("a physical prize was stamped with %d moemoepoint", winners[3].PointAwarded)
	}
}

func TestPrizePointTotal(t *testing.T) {
	if got := PrizePointBudget(topicModel.LotteryPointFixed, 50, 4); got != 200 {
		t.Fatalf("fixed total = %d, want 200", got)
	}
	if got := PrizePointBudget(topicModel.LotteryPointSplit, 50, 4); got != 50 {
		t.Fatalf("pooled total = %d, want the pool itself (50)", got)
	}
	// A prize written before point_mode existed reads as fixed.
	if got := PrizePointBudget("", 50, 4); got != 200 {
		t.Fatalf("legacy blank mode total = %d, want 200", got)
	}
}
