package service

import (
	"strings"
	"testing"

	msgService "kun-galgame-api/internal/message/service"
	topicModel "kun-galgame-api/internal/topic/model"
)

func TestDrawLeftoversCountsTheEmptyTail(t *testing.T) {
	lottery := &topicModel.TopicLottery{ID: 3, TopicID: 9, UserID: 5, Title: "t"}
	code := topicModel.TopicLotteryPrize{ID: 1, Delivery: topicModel.LotteryDeliveryCode}
	offline := topicModel.TopicLotteryPrize{ID: 2, Delivery: topicModel.LotteryDeliveryManual}
	point := topicModel.TopicLotteryPrize{ID: 3, Delivery: topicModel.LotteryDeliveryPoint}
	slots := []topicModel.TopicLotteryPrize{code, code, offline, point, point}

	spec, ok := drawLeftovers(lottery, slots, 1, 20)
	if !ok {
		t.Fatal("four empty slots must notify the author")
	}
	if spec.ReceiverID != 5 || spec.SenderID != 0 || spec.Kind != msgService.NotifyLotteryReturned || spec.TopicID != 9 {
		t.Fatalf("spec %+v", spec)
	}
	for _, want := range []string{"有 4 个名额", "1 个兑换码", "1 份线下奖品", "20 萌萌点"} {
		if !strings.Contains(spec.Content, want) {
			t.Errorf("content %q lacks %q", spec.Content, want)
		}
	}

	if _, ok := drawLeftovers(lottery, slots, 5, 0); ok {
		t.Error("a full draw has nothing to return")
	}
	if _, ok := drawLeftovers(lottery, []topicModel.TopicLotteryPrize{point}, 0, 0); ok {
		t.Error("an empty point slot with no escrow returns nothing")
	}
}
