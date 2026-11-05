package service

import (
	"fmt"
	"log/slog"
	"strings"

	msgModel "kun-galgame-api/internal/message/model"
	msgService "kun-galgame-api/internal/message/service"
	topicModel "kun-galgame-api/internal/topic/model"
)

func returnedSpec(lottery *topicModel.TopicLottery, senderID int, content string) msgService.Spec {
	return msgService.Spec{
		SenderID:   senderID,
		ReceiverID: lottery.UserID,
		Kind:       msgService.NotifyLotteryReturned,
		Content:    content,
		TopicID:    lottery.TopicID,
		Ref:        msgModel.LotteryRef(lottery.ID, lottery.TopicID),
	}
}

// The draw fills slots in order, so the empty ones are the tail of the list.
func drawLeftovers(
	lottery *topicModel.TopicLottery,
	slots []topicModel.TopicLotteryPrize,
	filled, refund int,
) (msgService.Spec, bool) {
	if filled >= len(slots) {
		return msgService.Spec{}, false
	}
	codes, offline := 0, 0
	for _, prize := range slots[filled:] {
		switch prize.Delivery {
		case topicModel.LotteryDeliveryCode:
			codes++
		case topicModel.LotteryDeliveryManual:
			offline++
		}
	}
	parts := make([]string, 0, 3)
	if codes > 0 {
		parts = append(parts, fmt.Sprintf("%d 个兑换码已退回给您, 可在抽奖中查看", codes))
	}
	if offline > 0 {
		parts = append(parts, fmt.Sprintf("%d 份线下奖品无需发放", offline))
	}
	if refund > 0 {
		parts = append(parts, fmt.Sprintf("%d 萌萌点已退回给您", refund))
	}
	if len(parts) == 0 {
		return msgService.Spec{}, false
	}
	content := fmt.Sprintf("您的抽奖「%s」有 %d 个名额没有中奖者: %s",
		lottery.Title, len(slots)-filled, strings.Join(parts, "; "))
	return returnedSpec(lottery, 0, content), true
}

// The winner is the sender so that each forfeited code gets its own notice:
// the message table keeps one row per sender, receiver, type and lottery.
func expiredCodeReturned(lottery *topicModel.TopicLottery, winnerID int) msgService.Spec {
	return returnedSpec(lottery, winnerID, fmt.Sprintf(
		"您的抽奖「%s」中有一个兑换码超过领取期限未被领取, 已退回给您, 可在抽奖中查看", lottery.Title))
}

func (s *LotteryService) NotifyPrizeGivenUp(lottery *topicModel.TopicLottery, prize *topicModel.TopicLotteryPrize, winnerID int) {
	spec := returnedSpec(lottery, winnerID, fmt.Sprintf(
		"一位中奖者放弃了您的抽奖「%s」中的奖品「%s」, 这份奖品无需再发放", lottery.Title, prize.Name))
	if err := s.notifier.Emit(nil, spec); err != nil {
		slog.Warn("抽奖奖品放弃通知发送失败", "lottery_id", lottery.ID, "error", err)
	}
}
