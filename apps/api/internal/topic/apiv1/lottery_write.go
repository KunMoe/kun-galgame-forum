package apiv1

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

type prizePlan struct {
	title       string
	description string
	images      []string
	adult       []string
	delivery    string
	pointMode   string
	pointAmount int
	slots       int
	codes       []string
}

func (p prizePlan) budget() int {
	if p.delivery != model.LotteryDeliveryPoint {
		return 0
	}
	return service.PrizePointBudget(p.pointMode, p.pointAmount, p.slots)
}

type lotteryShape struct {
	entryMode     string
	floorRule     string
	drawMode      string
	drawThreshold int
	closesAt      *time.Time
	prizes        []prizePlan
	// fromRequest says whether prizes came from this request, so a violation
	// can point at /prizes/N; stored prizes can only be blamed via the field
	// that made them wrong.
	fromRequest bool
}

func (s lotteryShape) slots() int {
	n := 0
	for _, p := range s.prizes {
		n += p.slots
	}
	return n
}

func (s lotteryShape) budget() int {
	n := 0
	for _, p := range s.prizes {
		n += p.budget()
	}
	return n
}

func prizePointer(i int, field string) string {
	return fmt.Sprintf("/prizes/%d/%s", i, field)
}

func notAllowed(pointer, detail string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonNotAllowedValue, detail, nil)
}

func planPrizes(in []LotteryPrizeInput, codesEnabled bool) ([]prizePlan, []problem.FieldError) {
	var fields []problem.FieldError
	out := make([]prizePlan, len(in))
	for i, raw := range in {
		p := prizePlan{
			title:    strings.TrimSpace(raw.Title),
			delivery: deliveryFromWire(raw.Delivery),
			slots:    raw.SlotCount,
			images:   make([]string, 0, len(raw.ImageHashes)),
			adult:    make([]string, 0, len(raw.AdultImageHashes)),
		}
		if p.title == "" {
			fields = append(fields, tooShort(prizePointer(i, "title")))
		}
		if p.slots < 1 {
			min := 1.0
			fields = append(fields, problem.AtPointer(prizePointer(i, "slot_count"), problem.ReasonOutOfRange,
				"a prize has at least one winner", &problem.FieldParams{Minimum: &min}))
		}
		if raw.Description != nil {
			p.description = strings.TrimSpace(*raw.Description)
		}
		for _, h := range raw.ImageHashes {
			p.images = append(p.images, string(h))
		}
		for _, h := range raw.AdultImageHashes {
			if !slices.Contains(p.images, string(h)) {
				fields = append(fields, problem.AtPointer(prizePointer(i, "adult_image_hashes"),
					problem.ReasonInconsistentWith, prizePointer(i, "image_hashes"), nil))
				break
			}
			p.adult = append(p.adult, string(h))
		}
		hasPoint := raw.PointMode != nil || raw.PointAmount != nil
		switch p.delivery {
		case model.LotteryDeliveryCode:
			if !codesEnabled {
				fields = append(fields, notAllowed(prizePointer(i, "delivery"),
					"code delivery is not available: this site holds no key to seal codes with"))
			}
			if hasPoint {
				fields = append(fields, notAllowed(prizePointer(i, "point_mode"), "only a point prize takes point_mode and point_amount"))
			}
			for j, c := range raw.Codes {
				code := strings.TrimSpace(c)
				if code == "" {
					fields = append(fields, tooShort(fmt.Sprintf("/prizes/%d/codes/%d", i, j)))
					continue
				}
				if len([]rune(code)) > lotteryCodeLimit {
					max := lotteryCodeLimit
					fields = append(fields, problem.AtPointer(fmt.Sprintf("/prizes/%d/codes/%d", i, j),
						problem.ReasonTooLong, "a redemption code is at most 200 characters", &problem.FieldParams{MaxLength: &max}))
					continue
				}
				p.codes = append(p.codes, code)
			}
			if len(raw.Codes) != p.slots {
				fields = append(fields, problem.AtPointer(prizePointer(i, "codes"),
					problem.ReasonInconsistentWith, prizePointer(i, "slot_count"), nil))
			}
		case model.LotteryDeliveryPoint:
			if len(raw.Codes) > 0 {
				fields = append(fields, notAllowed(prizePointer(i, "codes"), "only a code prize takes codes"))
			}
			if raw.PointMode == nil {
				fields = append(fields, problem.AtPointer(prizePointer(i, "point_mode"), problem.ReasonRequired, "required for a point prize", nil))
			} else {
				p.pointMode = *raw.PointMode
			}
			if raw.PointAmount == nil {
				fields = append(fields, problem.AtPointer(prizePointer(i, "point_amount"), problem.ReasonRequired, "required for a point prize", nil))
			} else {
				p.pointAmount = *raw.PointAmount
			}
			if p.pointMode != model.LotteryPointFixed && raw.PointAmount != nil && p.pointAmount < p.slots {
				fields = append(fields, problem.AtPointer(prizePointer(i, "point_amount"),
					problem.ReasonInconsistentWith, prizePointer(i, "slot_count"), nil))
			}
		default:
			if len(raw.Codes) > 0 {
				fields = append(fields, notAllowed(prizePointer(i, "codes"), "only a code prize takes codes"))
			}
			if hasPoint {
				fields = append(fields, notAllowed(prizePointer(i, "point_mode"), "only a point prize takes point_mode and point_amount"))
			}
		}
		out[i] = p
	}
	return out, fields
}

func storedPrizePlans(rows []model.TopicLotteryPrize) []prizePlan {
	out := make([]prizePlan, len(rows))
	for i, r := range rows {
		out[i] = prizePlan{
			title: r.Name, description: r.Description, images: r.ImageHashes, adult: r.NSFWHashes,
			delivery: r.Delivery, pointMode: r.PointMode, pointAmount: r.PointAmount, slots: r.Slots,
		}
	}
	return out
}

// validateShape checks the lottery as it would be stored, whatever part of it
// this request supplied. The legacy update skipped this whenever the prizes
// were left out, so a lottery could be turned into one that is never drawn.
func validateShape(s lotteryShape) []problem.FieldError {
	var fields []problem.FieldError
	if n := s.slots(); n > constants.MaxSlotsPerPrize {
		max := float64(constants.MaxSlotsPerPrize)
		fields = append(fields, problem.AtPointer("/prizes", problem.ReasonOutOfRange,
			"at most 500 slots across all prizes", &problem.FieldParams{Maximum: &max}))
	}
	if n := s.budget(); n > lotteryPointBudgetLimit {
		max := float64(lotteryPointBudgetLimit)
		fields = append(fields, problem.AtPointer("/prizes", problem.ReasonOutOfRange,
			"at most 100000 moemoepoint across the point prizes", &problem.FieldParams{Maximum: &max}))
	}
	switch s.drawMode {
	case model.LotteryDrawDeadline:
		if s.closesAt == nil {
			fields = append(fields, problem.AtPointer("/closes_at", problem.ReasonRequired, "required when draw_mode is deadline", nil))
		}
	case model.LotteryDrawThreshold:
		switch {
		case s.drawThreshold <= 0:
			fields = append(fields, problem.AtPointer("/draw_threshold", problem.ReasonRequired, "required when draw_mode is threshold", nil))
		case s.drawThreshold < s.slots():
			fields = append(fields, problem.AtPointer("/draw_threshold", problem.ReasonInconsistentWith, "/prizes", nil))
		}
	}
	if s.entryMode != model.LotteryEntryFloor {
		return fields
	}
	if s.drawMode == model.LotteryDrawThreshold {
		fields = append(fields, problem.AtPointer("/draw_mode", problem.ReasonInconsistentWith, "/entry_mode", nil))
	}
	if strings.TrimSpace(s.floorRule) == "" {
		fields = append(fields, problem.AtPointer("/floor_rule", problem.ReasonRequired, "required when entry_mode is floor", nil))
	} else if _, err := service.ParseFloorRule(s.floorRule, s.slots()); err != nil {
		if errors.Is(err, service.ErrFloorRuleCount) {
			fields = append(fields, problem.AtPointer("/floor_rule", problem.ReasonInconsistentWith, "/prizes", nil))
		} else {
			fields = append(fields, problem.AtPointer("/floor_rule", problem.ReasonInvalidFormat,
				"a comma-separated list of floors such as 8,18,28, or every:N", nil))
		}
	}
	for i, p := range s.prizes {
		if p.delivery != model.LotteryDeliveryPoint || p.pointMode != model.LotteryPointRandom {
			continue
		}
		pointer := "/entry_mode"
		if s.fromRequest {
			pointer = prizePointer(i, "point_mode")
		}
		fields = append(fields, problem.AtPointer(pointer, problem.ReasonInconsistentWith, "/entry_mode", nil))
		break
	}
	return fields
}

func parseClosesAt(d LotteryClosesAt, now time.Time) (*time.Time, []problem.FieldError) {
	if !d.Present || d.Value == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, string(*d.Value))
	if err != nil {
		return nil, []problem.FieldError{problem.AtPointer("/closes_at", problem.ReasonInvalidFormat,
			"must be a real instant in RFC 3339 UTC with second precision", nil)}
	}
	if !t.After(now) {
		return nil, []problem.FieldError{problem.AtPointer("/closes_at", problem.ReasonOutOfRange, "must be in the future", nil)}
	}
	utc := t.UTC()
	return &utc, nil
}

func lotteryModerationText(title, description string, prizes []prizePlan) string {
	parts := make([]string, 0, 2+len(prizes)*2)
	parts = append(parts, title, description)
	for _, p := range prizes {
		parts = append(parts, p.title, p.description)
	}
	return gate.ComposeText(parts...)
}

func lotteryCreatorIneligible() *problem.Problem {
	p := problem.New(problem.CodeLotteryCreatorIneligible, "Starting a lottery needs an older account or more moemoepoint.")
	p.SetExtension("min_account_age_days", constants.LotteryMinAccountAgeDays)
	p.SetExtension("min_moemoepoint", constants.LotteryMinMoemoepoint)
	return p
}

func escrowRef(lotteryID int) string {
	return moemoepoint.Ref("topic_lottery_escrow", lotteryID)
}

// chargeEscrow takes amount from the author's cached balance inside tx, after
// locking the row: two lotteries created at once serialize on that lock, and
// the second one sees what the first one took. OAuth hears about it after the
// commit, and its answer overwrites the cache.
func (l *Lotteries) chargeEscrow(tx *gorm.DB, authorID, amount int) error {
	if amount <= 0 {
		return nil
	}
	balance, err := l.reads.topics.LockMoemoepoint(tx, authorID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if balance < amount {
		return engageError{p: moemoepointInsufficient(amount)}
	}
	return l.lots.AdjustCachedMoemoepoint(tx, authorID, -amount)
}

func (l *Lotteries) writePrizes(tx *gorm.DB, lotteryID int, prizes []prizePlan) error {
	for i, p := range prizes {
		pointMode := p.pointMode
		if pointMode == "" {
			pointMode = model.LotteryPointFixed
		}
		row := &model.TopicLotteryPrize{
			LotteryID:   lotteryID,
			Name:        p.title,
			Description: p.description,
			ImageHashes: p.images,
			NSFWHashes:  p.adult,
			Delivery:    p.delivery,
			PointMode:   pointMode,
			PointAmount: p.pointAmount,
			Slots:       p.slots,
			SortOrder:   i,
		}
		if err := l.lots.CreatePrize(tx, row); err != nil {
			return err
		}
		for _, code := range p.codes {
			sealed, err := l.svc.SealCode(code)
			if err != nil {
				return err
			}
			if err := l.lots.CreateCode(tx, &model.TopicLotteryCode{
				LotteryID: lotteryID, PrizeID: row.ID, Secret: sealed,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Lotteries) createLottery(ctx context.Context, in *createLotteryInput) (*createLotteryOutput, error) {
	if prob := l.ready(); prob != nil {
		return nil, prob
	}
	topic, user, prob := l.reads.visibleTopic(ctx, in.TopicID)
	if prob != nil {
		return nil, prob
	}
	staff := user.Can(perm.LotteryCreateAny)
	if user.ID != topic.UserID && !staff {
		return nil, permissionRequired()
	}
	if !staff && !l.svc.CreatorEligible(ctx, user.ID) {
		return nil, lotteryCreatorIneligible()
	}
	count, err := l.lots.CountByTopicID(topic.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if count >= int64(constants.MaxLotteriesPerTopic) {
		max := constants.MaxLotteriesPerTopic
		return nil, validationFailed(problem.AtParameter("topic_id", problem.ReasonTooManyItems,
			"this topic already holds as many lotteries as it may", &problem.FieldParams{MaxItems: &max}))
	}

	body := in.Body
	now := time.Now()
	var fields []problem.FieldError
	title := strings.TrimSpace(body.Title)
	if title == "" {
		fields = append(fields, tooShort("/title"))
	}
	description := ""
	if body.Description != nil {
		description = strings.TrimSpace(*body.Description)
	}
	closesAt, closeFields := parseClosesAt(body.ClosesAt, now)
	fields = append(fields, closeFields...)
	prizes, prizeFields := planPrizes(body.Prizes, l.svc.CodesEnabled())
	fields = append(fields, prizeFields...)
	shape := lotteryShape{
		entryMode: body.EntryMode, drawMode: body.DrawMode,
		closesAt: closesAt, prizes: prizes, fromRequest: true,
	}
	if body.FloorRule != nil {
		shape.floorRule = strings.TrimSpace(*body.FloorRule)
	}
	if body.DrawThreshold != nil {
		shape.drawThreshold = *body.DrawThreshold
	}
	if len(closeFields) == 0 {
		fields = append(fields, validateShape(shape)...)
	}
	if len(fields) > 0 {
		return nil, validationFailed(fields...)
	}

	moderation := lotteryModerationText(title, description, prizes)
	decision, matched, prob := l.rejectContent(ctx, moderation, user.ID)
	if prob != nil {
		return nil, prob
	}
	seed, seedHash := service.NewDrawSeed()
	if shape.entryMode == model.LotteryEntryFloor {
		seed, seedHash = "", ""
	}
	row := &model.TopicLottery{
		TopicID:           topic.ID,
		UserID:            user.ID,
		Title:             title,
		Description:       description,
		EntryMode:         shape.entryMode,
		DrawMode:          shape.drawMode,
		Deadline:          closesAt,
		ShowEntrants:      body.IsEntryListPublic == nil || *body.IsEntryListPublic,
		Status:            model.LotteryStatusOpen,
		SeedHash:          seedHash,
		Seed:              seed,
		PointEscrow:       shape.budget(),
	}
	if shape.entryMode == model.LotteryEntryFloor {
		row.FloorRule = shape.floorRule
	}
	if shape.drawMode == model.LotteryDrawThreshold {
		row.DrawThreshold = shape.drawThreshold
	}
	if body.MinAccountAgeDays != nil {
		row.MinAccountAgeDays = *body.MinAccountAgeDays
	}
	if body.MinMoemoepoint != nil {
		row.MinMoemoepoint = *body.MinMoemoepoint
	}

	err = l.db().Transaction(func(tx *gorm.DB) error {
		if err := l.chargeEscrow(tx, user.ID, row.PointEscrow); err != nil {
			return err
		}
		if err := l.lots.Create(tx, row); err != nil {
			return err
		}
		if err := l.writePrizes(tx, row.ID, prizes); err != nil {
			return err
		}
		return l.lots.TouchTopic(tx, topic.ID, now)
	})
	if err != nil {
		return nil, mapEngageErr(err)
	}
	if row.PointEscrow > 0 {
		l.award(user.ID, -row.PointEscrow, moemoepoint.ReasonContentRemoved, escrowRef(row.ID),
			moemoepoint.Key("lottery_escrow", "topic_lottery_"+strconv.Itoa(row.ID)))
	}
	l.scanLottery(decision, matched, row.ID, user.ID, moderation)

	out, herr := l.lotteryOut(ctx, row.ID, user, false)
	if herr != nil {
		return nil, herr
	}
	return &createLotteryOutput{
		Location: "/api/v1/lotteries/" + strconv.Itoa(row.ID),
		Body:     out.Body,
	}, nil
}
