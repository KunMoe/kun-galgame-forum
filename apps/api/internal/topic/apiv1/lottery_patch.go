package apiv1

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func (b LotteryPatch) firstFieldBesidesState() string {
	switch {
	case b.Title != nil:
		return "/title"
	case b.Description != nil:
		return "/description"
	case b.EntryMode != nil:
		return "/entry_mode"
	case b.FloorRule != nil:
		return "/floor_rule"
	case b.DrawMode != nil:
		return "/draw_mode"
	case b.DrawThreshold != nil:
		return "/draw_threshold"
	case b.ClosesAt.Present:
		return "/closes_at"
	case b.MinAccountAgeDays != nil:
		return "/min_account_age_days"
	case b.MinMoemoepoint != nil:
		return "/min_moemoepoint"
	case b.IsEntryListPublic != nil:
		return "/is_entry_list_public"
	case b.Prizes != nil:
		return "/prizes"
	default:
		return ""
	}
}

func notOpenTransition(state, target string) *problem.Problem {
	return invalidStateTransition("The lottery is " + state + "; only an open lottery can become " + target + ".")
}

func (l *Lotteries) refundEscrow(authorID, lotteryID, amount int) {
	if amount > 0 {
		l.award(authorID, amount, moemoepoint.ReasonContentApproved, escrowRef(lotteryID), service.EscrowRefundKey(lotteryID))
	}
}

func (l *Lotteries) updateLottery(ctx context.Context, in *updateLotteryInput) (*lotteryOutput, error) {
	lottery, _, user, prob := l.visibleLottery(ctx, in.LotteryID)
	if prob != nil {
		return nil, prob
	}
	if !capsForLottery(lottery, user).Manage {
		return nil, permissionRequired()
	}
	if in.Body.State != nil {
		return l.moveLotteryState(ctx, lottery, user, in.Body)
	}
	if lottery.Status != model.LotteryStatusOpen {
		return nil, lotteryClosed()
	}
	return l.editLottery(ctx, lottery, user, in.Body)
}

func (l *Lotteries) moveLotteryState(ctx context.Context, lottery *model.TopicLottery, user *middleware.UserInfo, body LotteryPatch) (*lotteryOutput, error) {
	if other := body.firstFieldBesidesState(); other != "" {
		return nil, validationFailed(problem.AtPointer("/state", problem.ReasonInconsistentWith, other, nil))
	}
	target := *body.State
	switch target {
	case model.LotteryStatusDrawn:
		if err := l.svc.DrawOpen(ctx, lottery.ID); err != nil {
			if errors.Is(err, service.ErrLotteryNotOpen) {
				return nil, l.currentStateTransition(lottery.ID, target)
			}
			return nil, problem.Internal(err)
		}
	case model.LotteryStatusCancelled:
		var escrow int
		err := l.db().Transaction(func(tx *gorm.DB) error {
			held, ok, err := l.lots.CancelIfOpen(tx, lottery.ID, time.Now())
			if err != nil {
				return err
			}
			if !ok {
				return engageError{p: l.currentStateTransition(lottery.ID, target)}
			}
			escrow = held
			if held > 0 {
				return l.lots.AdjustCachedMoemoepoint(tx, lottery.UserID, held)
			}
			return nil
		})
		if err != nil {
			return nil, mapEngageErr(err)
		}
		l.refundEscrow(lottery.UserID, lottery.ID, escrow)
	}
	return l.lotteryOut(ctx, lottery.ID, user, false)
}

func (l *Lotteries) currentStateTransition(lotteryID int, target string) *problem.Problem {
	row, err := l.lots.FindByID(lotteryID)
	if err != nil {
		return problem.Internal(err)
	}
	return notOpenTransition(row.Status, target)
}

type lotteryEdit struct {
	shape       lotteryShape
	fields      map[string]any
	prizes      []prizePlan
	rewrite     bool
	moderation  string
	textChanged bool
}

func (l *Lotteries) planEdit(lottery *model.TopicLottery, body LotteryPatch) (*lotteryEdit, *problem.Problem) {
	var bad []problem.FieldError
	entered := lottery.EntryCount > 0
	shape := lotteryShape{
		entryMode:     lottery.EntryMode,
		floorRule:     lottery.FloorRule,
		drawMode:      lottery.DrawMode,
		drawThreshold: lottery.DrawThreshold,
		closesAt:      lottery.Deadline,
	}
	if body.EntryMode != nil && *body.EntryMode != lottery.EntryMode {
		if entered {
			bad = append(bad, problem.AtPointer("/entry_mode", problem.ReasonImmutable, "anyone who entered would be dropped from the draw", nil))
		}
		shape.entryMode = *body.EntryMode
	}
	if body.FloorRule != nil {
		rule := strings.TrimSpace(*body.FloorRule)
		if entered && rule != lottery.FloorRule {
			bad = append(bad, problem.AtPointer("/floor_rule", problem.ReasonImmutable, "the lottery already has entries", nil))
		}
		shape.floorRule = rule
	}
	if body.DrawMode != nil {
		shape.drawMode = *body.DrawMode
	}
	if body.DrawThreshold != nil {
		shape.drawThreshold = *body.DrawThreshold
	}
	if body.ClosesAt.Present {
		closesAt, fields := parseClosesAt(body.ClosesAt, time.Now())
		bad = append(bad, fields...)
		shape.closesAt = closesAt
	}

	edit := &lotteryEdit{rewrite: body.Prizes != nil}
	if edit.rewrite {
		if entered {
			bad = append(bad, problem.AtPointer("/prizes", problem.ReasonImmutable, "prizes cannot change once anyone has entered", nil))
		}
		prizes, fields := planPrizes(body.Prizes, l.svc.CodesEnabled())
		bad = append(bad, fields...)
		shape.prizes, shape.fromRequest, edit.prizes = prizes, true, prizes
	} else {
		stored, err := l.lots.FindPrizes(lottery.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		shape.prizes = storedPrizePlans(stored)
	}
	if len(bad) == 0 {
		bad = append(bad, validateShape(shape)...)
	}

	title, description := lottery.Title, lottery.Description
	if body.Title != nil {
		title = strings.TrimSpace(*body.Title)
		if title == "" {
			bad = append(bad, tooShort("/title"))
		}
		edit.textChanged = true
	}
	if body.Description != nil {
		description = strings.TrimSpace(*body.Description)
		edit.textChanged = true
	}
	if len(bad) > 0 {
		return nil, validationFailed(bad...)
	}
	if edit.rewrite {
		edit.textChanged = true
	}
	edit.moderation = lotteryModerationText(title, description, edit.prizes)

	floorRule, threshold := "", 0
	if shape.entryMode == model.LotteryEntryFloor {
		floorRule = shape.floorRule
	}
	if shape.drawMode == model.LotteryDrawThreshold {
		threshold = shape.drawThreshold
	}
	edit.fields = map[string]any{
		"title":          title,
		"description":    description,
		"entry_mode":     shape.entryMode,
		"floor_rule":     floorRule,
		"draw_mode":      shape.drawMode,
		"draw_threshold": threshold,
		"deadline":       shape.closesAt,
		"updated":        time.Now(),
	}
	if body.MinAccountAgeDays != nil {
		edit.fields["min_account_age_days"] = *body.MinAccountAgeDays
	}
	if body.MinMoemoepoint != nil {
		edit.fields["min_moemoepoint"] = *body.MinMoemoepoint
	}
	if body.IsEntryListPublic != nil {
		edit.fields["show_entrants"] = *body.IsEntryListPublic
	}
	edit.shape = shape
	return edit, nil
}

func (l *Lotteries) editLottery(ctx context.Context, lottery *model.TopicLottery, user *middleware.UserInfo, body LotteryPatch) (*lotteryOutput, error) {
	edit, prob := l.planEdit(lottery, body)
	if prob != nil {
		return nil, prob
	}
	decision, matched := "", []string(nil)
	if edit.textChanged {
		d, m, prob := l.rejectContent(ctx, edit.moderation, lottery.UserID)
		if prob != nil {
			return nil, prob
		}
		decision, matched = d, m
	}

	delta := 0
	err := l.db().Transaction(func(tx *gorm.DB) error {
		locked, err := l.lots.LockByID(tx, lottery.ID)
		if err != nil {
			return err
		}
		if locked.Status != model.LotteryStatusOpen {
			return engageError{p: lotteryClosed()}
		}
		if locked.EntryCount > 0 && (edit.rewrite || locked.EntryMode != edit.shape.entryMode) {
			return engageError{p: validationFailed(problem.AtPointer("/prizes", problem.ReasonImmutable,
				"someone entered while this edit was being made", nil))}
		}
		if edit.rewrite {
			delta = edit.shape.budget() - locked.PointEscrow
			edit.fields["point_escrow"] = edit.shape.budget()
			if delta > 0 {
				if err := l.chargeEscrow(tx, locked.UserID, delta); err != nil {
					return err
				}
			} else if delta < 0 {
				if err := l.lots.AdjustCachedMoemoepoint(tx, locked.UserID, -delta); err != nil {
					return err
				}
			}
		}
		if err := l.lots.UpdateFields(tx, lottery.ID, edit.fields); err != nil {
			return err
		}
		if !edit.rewrite {
			return nil
		}
		if err := l.lots.DeletePrizes(tx, lottery.ID); err != nil {
			return err
		}
		return l.writePrizes(tx, lottery.ID, edit.prizes)
	})
	if err != nil {
		return nil, mapEngageErr(err)
	}
	if delta != 0 {
		reason := moemoepoint.ReasonContentRemoved
		if delta < 0 {
			reason = moemoepoint.ReasonContentApproved
		}
		l.award(lottery.UserID, -delta, reason, escrowRef(lottery.ID),
			moemoepoint.KeyNonce("lottery_escrow_change", "topic_lottery_"+strconv.Itoa(lottery.ID)))
	}
	if edit.textChanged {
		l.scanLottery(decision, matched, lottery.ID, lottery.UserID, edit.moderation)
	}
	return l.lotteryOut(ctx, lottery.ID, user, false)
}

func (l *Lotteries) deleteLottery(ctx context.Context, in *lotteryInput) (*deleteLotteryOutput, error) {
	lottery, _, user, prob := l.visibleLottery(ctx, in.LotteryID)
	if prob != nil {
		return nil, prob
	}
	caps := capsForLottery(lottery, user)
	if !caps.Manage {
		return nil, permissionRequired()
	}
	if !caps.canDelete(lottery.Status) {
		return nil, lotteryDrawn()
	}
	var escrow int
	err := l.db().Transaction(func(tx *gorm.DB) error {
		held, ok, err := l.lots.DeleteDeletable(tx, lottery.ID, caps.Staff)
		if err != nil {
			return err
		}
		if !ok {
			return engageError{p: lotteryDrawn()}
		}
		escrow = held
		if held > 0 {
			return l.lots.AdjustCachedMoemoepoint(tx, lottery.UserID, held)
		}
		return nil
	})
	if err != nil {
		return nil, mapEngageErr(err)
	}
	l.refundEscrow(lottery.UserID, lottery.ID, escrow)
	return &deleteLotteryOutput{}, nil
}
