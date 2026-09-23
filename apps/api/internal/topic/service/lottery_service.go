package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/constants"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/moemoepoint"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/secretbox"
	"kun-galgame-api/pkg/userclient"
)

type AwardFunc func(userID, delta int, reason, ref, key string)

type LotteryService struct {
	lotteryRepo *repository.LotteryRepository
	stateRepo   *userRepo.StateRepository
	userClient  *userclient.Client
	notifier    msgService.Notifier
	box         *secretbox.Box
	award       AwardFunc
}

func NewLotteryService(
	lotteryRepo *repository.LotteryRepository,
	stateRepo *userRepo.StateRepository,
	userClient *userclient.Client,
	notifier msgService.Notifier,
	box *secretbox.Box,
) *LotteryService {
	return &LotteryService{
		lotteryRepo: lotteryRepo, stateRepo: stateRepo,
		userClient: userClient, notifier: notifier, box: box, award: moemoepoint.Award,
	}
}

func (s *LotteryService) SetAward(award AwardFunc) {
	if award != nil {
		s.award = award
	}
}

func (s *LotteryService) Repo() *repository.LotteryRepository { return s.lotteryRepo }

func (s *LotteryService) CodesEnabled() bool { return s.box.Enabled() }

func (s *LotteryService) SealCode(raw string) (string, error) {
	return s.box.Seal(strings.TrimSpace(raw))
}

func (s *LotteryService) OpenCode(sealed string) (string, error) {
	return s.box.Open(sealed)
}

// CreatorEligible is the anti-scam bar, not a permission. A lottery advertises
// free goods to every reader of a topic, which is exactly what a throwaway
// account is for, so an author needs either an account old enough to be
// inconvenient to farm or enough moemoepoint to have actually contributed.
func (s *LotteryService) CreatorEligible(ctx context.Context, userID int) bool {
	state, err := s.stateRepo.FindByID(userID)
	if err == nil && state.Moemoepoint >= constants.LotteryMinMoemoepoint {
		return true
	}
	u, _, _ := s.userClient.User(ctx, userID)
	days, ok := accountAgeDays(u)
	return ok && days >= constants.LotteryMinAccountAgeDays
}

// OAuth returns created_at as a string and it is not a column this database can
// filter on, so every account-age decision happens in Go after a hydrate.
func accountAgeDays(u userclient.User) (int, bool) {
	if u.CreatedAt == "" {
		return 0, false
	}
	t, err := time.Parse(time.RFC3339, u.CreatedAt)
	if err != nil {
		return 0, false
	}
	return int(time.Since(t).Hours() / 24), true
}

const (
	EntryBlockNotOpen       = "not_open"
	EntryBlockPastClosesAt  = "past_closes_at"
	EntryBlockNoSignup      = "no_signup"
	EntryBlockOwnLottery    = "own_lottery"
	EntryBlockReplyRequired = "reply_required"
	EntryBlockBelowMinimum  = "moemoepoint_below_minimum"
	EntryBlockAccountTooNew = "account_too_new"
)

// EntryBlock is the one place that decides whether userID may enter, and why
// not. The button state and the write guard both read it, so they cannot drift
// into a button that is enabled for a request the server refuses.
func (s *LotteryService) EntryBlock(ctx context.Context, lottery *topicModel.TopicLottery, userID int) (string, error) {
	if lottery.Status != topicModel.LotteryStatusOpen {
		return EntryBlockNotOpen, nil
	}
	if lottery.Deadline != nil && time.Now().After(*lottery.Deadline) {
		return EntryBlockPastClosesAt, nil
	}
	if lottery.EntryMode == topicModel.LotteryEntryFloor {
		return EntryBlockNoSignup, nil
	}
	if lottery.UserID == userID {
		return EntryBlockOwnLottery, nil
	}
	if lottery.EntryMode == topicModel.LotteryEntryReply {
		replied, err := s.lotteryRepo.HasRepliedTo(lottery.TopicID, userID)
		if err != nil {
			return "", err
		}
		if !replied {
			return EntryBlockReplyRequired, nil
		}
	}
	if lottery.MinMoemoepoint > 0 {
		state, err := s.stateRepo.FindByID(userID)
		if err != nil || state.Moemoepoint < lottery.MinMoemoepoint {
			return EntryBlockBelowMinimum, nil
		}
	}
	if lottery.MinAccountAgeDays > 0 {
		u, _, _ := s.userClient.User(ctx, userID)
		days, ok := accountAgeDays(u)
		if !ok || days < lottery.MinAccountAgeDays {
			return EntryBlockAccountTooNew, nil
		}
	}
	return "", nil
}

// isPointPool reports whether point_amount is the whole prize's budget rather
// than one winner's share.
func isPointPool(mode string) bool {
	return mode == topicModel.LotteryPointSplit || mode == topicModel.LotteryPointRandom
}

// PrizePointBudget is what a point prize pays out in total when every slot is
// filled, which is what its author is charged for it.
func PrizePointBudget(mode string, amount, slots int) int {
	if isPointPool(mode) {
		return amount
	}
	return amount * slots
}

var (
	ErrLotteryNotOpen  = errors.New("lottery is not open")
	ErrFloorRuleFormat = errors.New("floor rule is neither a comma-separated list of floors nor every:N")
	ErrFloorRuleCount  = errors.New("floor rule names a different number of floors than there are slots")
)

// ParseFloorRule turns the author's rule into the ordered floors that win. Two
// forms only: an explicit list ("8,18,28") and "every:N". Both are verifiable
// by a reader counting floors, which is the entire point of a floor lottery.
func ParseFloorRule(rule string, slots int) ([]int, error) {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return nil, ErrFloorRuleFormat
	}
	if after, ok := strings.CutPrefix(rule, "every:"); ok {
		step, err := strconv.Atoi(strings.TrimSpace(after))
		if err != nil || step <= 0 {
			return nil, ErrFloorRuleFormat
		}
		floors := make([]int, 0, slots)
		for i := 1; i <= slots; i++ {
			floors = append(floors, step*i)
		}
		return floors, nil
	}

	seen := map[int]bool{}
	floors := make([]int, 0, slots)
	for _, part := range strings.Split(rule, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || n <= 0 {
			return nil, ErrFloorRuleFormat
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		floors = append(floors, n)
	}
	if len(floors) != slots {
		return nil, ErrFloorRuleCount
	}
	sort.Ints(floors)
	return floors, nil
}
