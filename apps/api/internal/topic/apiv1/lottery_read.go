package apiv1

import (
	"context"
	"slices"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type lotteryBundle struct {
	prizes  map[int][]model.TopicLotteryPrize
	winners map[int][]model.TopicLotteryEntry
	codes   map[int]int
	mine    map[int]model.TopicLotteryEntry
	users   map[int]userclient.User
	meta    map[string]imageclient.ImageMeta
}

func (l *Lotteries) loadBundle(ctx context.Context, lotteries []model.TopicLottery, viewer *middleware.UserInfo) (*lotteryBundle, *problem.Problem) {
	ids := make([]int, len(lotteries))
	for i, row := range lotteries {
		ids[i] = row.ID
	}
	prizes, err := l.lots.FindPrizesForLotteries(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	winners, err := l.lots.FindWinnersForLotteries(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	codes, err := l.lots.CountCodesForLotteries(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	b := &lotteryBundle{
		prizes:  map[int][]model.TopicLotteryPrize{},
		winners: map[int][]model.TopicLotteryEntry{},
		codes:   codes,
		mine:    map[int]model.TopicLotteryEntry{},
	}
	if viewer != nil {
		b.mine, err = l.lots.FindEntriesForUser(ids, viewer.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	var hashes []string
	for _, p := range prizes {
		b.prizes[p.LotteryID] = append(b.prizes[p.LotteryID], p)
		hashes = append(hashes, p.ImageHashes...)
	}
	uids := make([]int, 0, len(lotteries)+len(winners))
	for _, row := range lotteries {
		uids = append(uids, row.UserID)
	}
	for _, w := range winners {
		b.winners[w.LotteryID] = append(b.winners[w.LotteryID], w)
		uids = append(uids, w.UserID)
	}
	users, prob := l.reads.lookupUsers(ctx, uids)
	if prob != nil {
		return nil, prob
	}
	b.users = users
	if len(hashes) > 0 && l.images != nil {
		b.meta = l.images(hashes)
	}
	return b, nil
}

func (l *Lotteries) buildLotteries(ctx context.Context, lotteries []model.TopicLottery, viewer *middleware.UserInfo, includeNSFW bool) ([]Lottery, *problem.Problem) {
	out := make([]Lottery, 0, len(lotteries))
	if len(lotteries) == 0 {
		return out, nil
	}
	b, prob := l.loadBundle(ctx, lotteries, viewer)
	if prob != nil {
		return nil, prob
	}
	for i := range lotteries {
		row := &lotteries[i]
		if u, ok := b.users[row.UserID]; ok && !userclient.IsRenderable(u) {
			continue
		}
		item, prob := l.mapLottery(ctx, row, b, viewer, includeNSFW)
		if prob != nil {
			return nil, prob
		}
		out = append(out, item)
	}
	return out, nil
}

func (l *Lotteries) mapPrize(p model.TopicLotteryPrize, b *lotteryBundle, includeNSFW bool) LotteryPrize {
	images := make([]LotteryPrizeImage, 0, len(p.ImageHashes))
	for _, hash := range p.ImageHashes {
		meta, known := b.meta[hash]
		img := LotteryPrizeImage{
			Hash:             hash,
			IsMarkedAdult:    slices.Contains(p.NSFWHashes, hash),
			IsGradedExplicit: known && meta.IsSexuallyExplicit(),
		}
		if includeNSFW || (!img.IsMarkedAdult && !img.IsGradedExplicit) {
			var m *imageclient.ImageMeta
			if known {
				m = &meta
			}
			img.Image = repr.NewImage(l.reads.cdn, hash, m)
		}
		images = append(images, img)
	}
	out := LotteryPrize{
		Object:      "lottery_prize",
		ID:          repr.ID(p.ID),
		Title:       p.Name,
		Description: p.Description,
		Images:      images,
		Delivery:    deliveryToWire(p.Delivery),
		SlotCount:   p.Slots,
	}
	switch p.Delivery {
	case model.LotteryDeliveryPoint:
		mode := p.PointMode
		if mode == "" {
			mode = model.LotteryPointFixed
		}
		amount := p.PointAmount
		budget := service.PrizePointBudget(mode, amount, p.Slots)
		out.PointMode, out.PointAmount, out.PointBudget = &mode, &amount, &budget
	case model.LotteryDeliveryCode:
		n := b.codes[p.ID]
		out.CodeCount = &n
	}
	return out
}

// Storage says manual; the wire says offline, because draw_mode already has a
// manual that means something else.
func deliveryToWire(stored string) string {
	if stored == model.LotteryDeliveryManual {
		return "offline"
	}
	return stored
}

func deliveryFromWire(wire string) string {
	if wire == "offline" {
		return model.LotteryDeliveryManual
	}
	return wire
}

func (l *Lotteries) mapLottery(ctx context.Context, row *model.TopicLottery, b *lotteryBundle, viewer *middleware.UserInfo, includeNSFW bool) (Lottery, *problem.Problem) {
	author := repr.DeletedUserRef(row.UserID)
	if u, ok := b.users[row.UserID]; ok {
		author = repr.NewUserRef(l.reads.cdn, u)
	}
	prizes := make([]LotteryPrize, 0, len(b.prizes[row.ID]))
	slots := 0
	for _, p := range b.prizes[row.ID] {
		prizes = append(prizes, l.mapPrize(p, b, includeNSFW))
		slots += p.Slots
	}
	winners := make([]LotteryWinner, 0, len(b.winners[row.ID]))
	for _, w := range b.winners[row.ID] {
		u, ok := b.users[w.UserID]
		if !ok || !userclient.IsRenderable(u) {
			continue
		}
		winners = append(winners, mapWinner(l.reads.cdn, w, u))
	}
	out := Lottery{
		Object:            "lottery",
		ID:                repr.ID(row.ID),
		TopicID:           repr.ID(row.TopicID),
		Author:            author,
		Title:             row.Title,
		Description:       row.Description,
		EntryMode:         row.EntryMode,
		DrawMode:          row.DrawMode,
		ClosesAt:          repr.TimestampPtr(row.Deadline),
		MinAccountAgeDays: row.MinAccountAgeDays,
		MinMoemoepoint:    row.MinMoemoepoint,
		IsEntryListPublic: row.ShowEntrants,
		State:             row.Status,
		SeedHash:          optionalString(row.SeedHash),
		EntryCount:        row.EntryCount,
		SlotCount:         slots,
		DrawnAt:           repr.TimestampPtr(row.DrawnAt),
		Prizes:            prizes,
		Winners:           winners,
		CreatedAt:         repr.Timestamp(row.CreatedAt),
		UpdatedAt:         repr.Timestamp(row.UpdatedAt),
	}
	if row.EntryMode == model.LotteryEntryFloor {
		out.FloorRule = optionalString(row.FloorRule)
	}
	if row.DrawMode == model.LotteryDrawThreshold && row.DrawThreshold > 0 {
		n := row.DrawThreshold
		out.DrawThreshold = &n
	}
	if row.Status == model.LotteryStatusDrawn {
		out.Seed = optionalString(row.Seed)
	}
	if viewer != nil {
		v, prob := l.lotteryViewer(ctx, row, b.mine, viewer)
		if prob != nil {
			return Lottery{}, prob
		}
		out.Viewer = v
	}
	return out, nil
}

func mapWinner(cdn string, w model.TopicLotteryEntry, u userclient.User) LotteryWinner {
	out := LotteryWinner{
		Object:         "lottery_winner",
		ID:             repr.ID(w.ID),
		PrizeID:        repr.ID(w.PrizeID),
		Winner:         repr.NewUserRef(cdn, u),
		RankKey:        optionalString(w.RankKey),
		Fulfillment:    w.Fulfillment,
		PointAwarded:   w.PointAwarded,
		ClaimExpiresAt: repr.TimestampPtr(w.ClaimDeadline),
	}
	if w.WonAt != nil {
		out.WonAt = repr.Timestamp(*w.WonAt)
	}
	if w.ReplyFloor > 0 {
		n := w.ReplyFloor
		out.WinningFloor = &n
	}
	return out
}

type lotteryCaps struct {
	Manage      bool
	Staff       bool
	ViewEntries bool
}

func capsForLottery(row *model.TopicLottery, user *middleware.UserInfo) lotteryCaps {
	if user == nil {
		return lotteryCaps{ViewEntries: row.ShowEntrants}
	}
	staff := user.Can(perm.LotteryManageAny)
	author := user.ID == row.UserID
	return lotteryCaps{
		Manage:      author || staff,
		Staff:       staff,
		ViewEntries: row.ShowEntrants || author || user.Can(perm.LotteryViewRestricted),
	}
}

func (c lotteryCaps) canDelete(state string) bool {
	switch state {
	case model.LotteryStatusDrawing:
		return false
	case model.LotteryStatusDrawn:
		return c.Staff
	default:
		return c.Manage
	}
}

func (l *Lotteries) lotteryViewer(ctx context.Context, row *model.TopicLottery, mine map[int]model.TopicLotteryEntry, user *middleware.UserInfo) (*LotteryViewer, *problem.Problem) {
	caps := capsForLottery(row, user)
	open := row.Status == model.LotteryStatusOpen
	v := &LotteryViewer{
		CanEdit:              caps.Manage && open,
		CanDelete:            caps.canDelete(row.Status),
		CanDraw:              caps.Manage && open,
		CanCancel:            caps.Manage && open,
		CanViewEntries:       caps.ViewEntries,
		CanManageFulfillment: caps.Manage && row.Status == model.LotteryStatusDrawn,
	}
	entry, entered := mine[row.ID]
	v.HasEntered = entered
	if !entered {
		reason, err := l.svc.EntryBlock(ctx, row, user.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		if reason == "" {
			v.CanEnter = true
		} else {
			v.EnterBlockedReason = &reason
		}
		return v, nil
	}
	if entry.PrizeID > 0 {
		id := repr.ID(entry.ID)
		v.WinnerID = &id
		v.CanRevealCode = entry.CodeID > 0 && entry.Fulfillment != model.LotteryFulfillForfeited
	}
	return v, nil
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (l *Lotteries) listTopicLotteries(ctx context.Context, in *listTopicLotteriesInput) (*listTopicLotteriesOutput, error) {
	if prob := l.ready(); prob != nil {
		return nil, prob
	}
	topic, user, prob := l.reads.visibleTopic(ctx, in.TopicID)
	if prob != nil {
		return nil, prob
	}
	rows, err := l.lots.FindByTopicNewestFirst(topic.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, prob := l.buildLotteries(ctx, rows, user, in.IncludeNSFW)
	if prob != nil {
		return nil, prob
	}
	return &listTopicLotteriesOutput{Body: repr.NewList(items, nil)}, nil
}

func (l *Lotteries) getLottery(ctx context.Context, in *getLotteryInput) (*lotteryOutput, error) {
	lottery, _, user, prob := l.visibleLottery(ctx, in.LotteryID)
	if prob != nil {
		return nil, prob
	}
	return l.lotteryOut(ctx, lottery.ID, user, in.IncludeNSFW)
}

func (l *Lotteries) lotteryOut(ctx context.Context, lotteryID int, user *middleware.UserInfo, includeNSFW bool) (*lotteryOutput, error) {
	row, err := l.lots.FindByID(lotteryID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, prob := l.buildLotteries(ctx, []model.TopicLottery{*row}, user, includeNSFW)
	if prob != nil {
		return nil, prob
	}
	if len(items) == 0 {
		return nil, notFound()
	}
	return &lotteryOutput{Body: items[0]}, nil
}

const entrySort = "created_asc"

func entryFingerprint(lotteryID int) string {
	return collect.Fingerprint("lottery_entries", strconv.Itoa(lotteryID))
}

func parseEntryPos(keys []string) (*repository.LotteryEntryKey, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 2 {
		return nil, invalidCursor()
	}
	created, err := time.Parse(time.RFC3339Nano, keys[0])
	if err != nil {
		return nil, invalidCursor()
	}
	id, err := strconv.Atoi(keys[1])
	if err != nil || id <= 0 {
		return nil, invalidCursor()
	}
	return &repository.LotteryEntryKey{Created: created, ID: id}, nil
}

func (l *Lotteries) listLotteryEntries(ctx context.Context, in *listLotteryEntriesInput) (*listLotteryEntriesOutput, error) {
	lottery, _, user, prob := l.visibleLottery(ctx, in.LotteryID)
	if prob != nil {
		return nil, prob
	}
	if !capsForLottery(lottery, user).ViewEntries {
		return nil, permissionRequired()
	}
	fp := entryFingerprint(lottery.ID)
	keys, curErr := collect.DecodeCursor(in.Cursor, entrySort, fp)
	if curErr != nil {
		return nil, curErr
	}
	pos, prob := parseEntryPos(keys)
	if prob != nil {
		return nil, prob
	}
	limit := in.Limit
	if limit <= 0 {
		limit = collect.DefaultLimit
	}
	rows, err := l.lots.FindEntriesAfter(lottery.ID, limit+1, pos)
	if err != nil {
		return nil, problem.Internal(err)
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	uids := make([]int, len(rows))
	for i, row := range rows {
		uids[i] = row.UserID
	}
	users, prob := l.reads.lookupUsers(ctx, uids)
	if prob != nil {
		return nil, prob
	}
	items := make([]LotteryEntry, 0, len(rows))
	for _, row := range rows {
		u, ok := users[row.UserID]
		if !ok || !userclient.IsRenderable(u) {
			continue
		}
		items = append(items, LotteryEntry{
			Object:    "lottery_entry",
			ID:        repr.ID(row.ID),
			Entrant:   repr.NewUserRef(l.reads.cdn, u),
			CreatedAt: repr.Timestamp(row.CreatedAt),
		})
	}
	var next *string
	if hasMore {
		last := rows[len(rows)-1]
		cursor := collect.EncodeCursor(entrySort, fp,
			last.CreatedAt.UTC().Format(time.RFC3339Nano), strconv.Itoa(last.ID))
		next = &cursor
	}
	return &listLotteryEntriesOutput{Body: repr.NewList(items, next)}, nil
}
