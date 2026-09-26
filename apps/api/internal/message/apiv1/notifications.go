package apiv1

import (
	"context"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/message/notifytype"
	"kun-galgame-api/internal/message/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type listNotificationsInput struct {
	collect.Page
	IsMuted          bool            `query:"is_muted" default:"false" doc:"When false, only types the caller has not muted. When true, only types the caller has muted. The two partitions are disjoint and together are every row. is_muted=false with notification_type set to a muted type is a legal empty list."`
	NotificationType notifytype.Type `query:"notification_type" required:"false" doc:"When set, restrict the chosen partition to this notification type. Omitted means every type in the partition. An unknown token is UNKNOWN_ENUM_VALUE."`
}

type listNotificationsOutput struct {
	Body repr.List[Notification]
}

type getNotificationSummaryInput struct{}

type getNotificationSummaryOutput struct {
	Body NotificationSummary
}

type markNotificationsReadInput struct {
	Body NotificationReadMarkerWrite
}

type markNotificationsReadOutput struct {
	Body NotificationReadMarker
}

type deleteNotificationInput struct {
	NotificationID string `path:"notification_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Notification id."`
}

type deleteNotificationOutput struct{}

type getNotificationInput struct {
	NotificationID string `path:"notification_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Notification id."`
}

type getNotificationOutput struct {
	Body Notification
}

func parseNotificationPos(keys []string) (*repository.V1NotificationPos, *problem.Problem) {
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
	return &repository.V1NotificationPos{Created: created, ID: id}, nil
}

func encodeNotificationCursor(sort, fp string, created time.Time, id int) string {
	return collect.EncodeCursor(sort, fp, created.UTC().Format(time.RFC3339Nano), strconv.Itoa(id))
}

func (s *Service) listNotifications(ctx context.Context, in *listNotificationsInput) (*listNotificationsOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := mustUser(ctx)
	if p != nil {
		return nil, p
	}
	mutedTypes, _, p := s.mutedTypes(user.ID)
	if p != nil {
		return nil, p
	}
	fp := notificationFingerprint(user.ID, in.IsMuted, in.NotificationType)
	keys, curErr := collect.DecodeCursor(in.Cursor, notificationSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	pos, posErr := parseNotificationPos(keys)
	if posErr != nil {
		return nil, posErr
	}
	limit := in.Limit
	if limit <= 0 {
		limit = collect.DefaultLimit
	}
	onlyType := notifytype.ToDB(in.NotificationType)

	// Banned actors and unknown stored types are dropped in Go, so a short SQL
	// page is not the end of the source; keep reading until the page is full
	// or the scan cap is hit.
	type kept struct {
		item    Notification
		created time.Time
		id      int
	}
	items := make([]kept, 0, limit+1)
	maxScan := pageRefillFactor * limit
	scanned := 0
	var last repository.V1NotificationRow
	haveLast := false
	exhausted := false
	for scanned < maxScan {
		batch := limit + 1
		if rem := maxScan - scanned; rem < batch {
			batch = rem
		}
		rows, err := s.messages.FindNotificationsKeyset(user.ID, in.IsMuted, mutedTypes, onlyType, pos, batch)
		if err != nil {
			return nil, problem.Internal(err)
		}
		if len(rows) == 0 {
			exhausted = true
			break
		}
		users := s.hydrateUsers(ctx, userclient.CollectIDs(rows, func(r repository.V1NotificationRow) int { return r.SenderID }))
		for _, row := range rows {
			scanned++
			last = row
			haveLast = true
			n, ok := s.mapNotification(row, users)
			if ok {
				items = append(items, kept{item: n, created: row.Created, id: row.ID})
			}
			if len(items) == limit+1 {
				break
			}
		}
		if len(items) == limit+1 {
			break
		}
		if len(rows) < batch {
			exhausted = true
			break
		}
		pos = &repository.V1NotificationPos{Created: last.Created, ID: last.ID}
	}

	var next *string
	if len(items) > limit {
		lastKept := items[limit-1]
		items = items[:limit]
		cur := encodeNotificationCursor(notificationSort, fp, lastKept.created, lastKept.id)
		next = &cur
	} else if haveLast && scanned >= maxScan && !exhausted {
		cur := encodeNotificationCursor(notificationSort, fp, last.Created, last.ID)
		next = &cur
	}
	out := make([]Notification, len(items))
	for i, it := range items {
		out[i] = it.item
	}
	return &listNotificationsOutput{Body: repr.NewList(out, next)}, nil
}

func (s *Service) getNotificationSummary(ctx context.Context, _ *getNotificationSummaryInput) (*getNotificationSummaryOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := mustUser(ctx)
	if p != nil {
		return nil, p
	}
	mutedTypes, chatMuted, p := s.mutedTypes(user.ID)
	if p != nil {
		return nil, p
	}
	dmUnread, p := s.directMessageUnread(ctx, user.ID)
	if p != nil {
		return nil, p
	}
	unread, err := s.messages.CountUnreadPartition(user.ID, false, mutedTypes)
	if err != nil {
		return nil, problem.Internal(err)
	}
	mutedUnread, err := s.messages.CountUnreadPartition(user.ID, true, mutedTypes)
	if err != nil {
		return nil, problem.Internal(err)
	}

	rows, err := s.messages.FindNotificationsKeyset(user.ID, false, mutedTypes, "", nil, pageRefillFactor)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var latest *Notification
	if len(rows) > 0 {
		users := s.hydrateUsers(ctx, userclient.CollectIDs(rows, func(r repository.V1NotificationRow) int { return r.SenderID }))
		for _, row := range rows {
			n, ok := s.mapNotification(row, users)
			if ok {
				cp := n
				latest = &cp
				break
			}
		}
	}
	return &getNotificationSummaryOutput{Body: NotificationSummary{
		Object:                   "notification_summary",
		UnreadCount:              unread,
		MutedUnreadCount:         mutedUnread,
		DirectMessageUnreadCount: dmUnread,
		IsDirectMessageMuted:     chatMuted,
		Latest:                   latest,
	}}, nil
}

func (s *Service) markNotificationsRead(ctx context.Context, in *markNotificationsReadInput) (*markNotificationsReadOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := mustUser(ctx)
	if p != nil {
		return nil, p
	}
	upTo, p := parseUpToID(in.Body.UpToID)
	if p != nil {
		return nil, p
	}
	mutedTypes, _, p := s.mutedTypes(user.ID)
	if p != nil {
		return nil, p
	}
	marked, ids, unread, err := s.messages.MarkReadUpTo(user.ID, upTo, in.Body.IsMuted, mutedTypes)
	if err != nil {
		return nil, problem.Internal(err)
	}
	s.forwardRead(user.ID, ids)
	return &markNotificationsReadOutput{Body: NotificationReadMarker{
		Object:      "notification_read_marker",
		MarkedCount: marked,
		UnreadCount: unread,
	}}, nil
}

func (s *Service) deleteNotification(ctx context.Context, in *deleteNotificationInput) (*deleteNotificationOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := mustUser(ctx)
	if p != nil {
		return nil, p
	}
	id, ok := parsePositiveID(in.NotificationID)
	if !ok {
		return nil, notFound()
	}
	status, commID, found, err := s.messages.DeleteNotification(id, user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if !found {
		return nil, notFound()
	}
	if status == "unread" && commID != nil && *commID > 0 {
		s.forwardRead(user.ID, []int64{*commID})
	}
	return &deleteNotificationOutput{}, nil
}

func (s *Service) getNotification(ctx context.Context, in *getNotificationInput) (*getNotificationOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := mustUser(ctx)
	if p != nil {
		return nil, p
	}
	id, ok := parsePositiveID(in.NotificationID)
	if !ok {
		return nil, notFound()
	}
	row, err := s.messages.FindNotification(id, user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if row == nil {
		return nil, notFound()
	}
	users := s.hydrateUsers(ctx, []int{row.SenderID})
	n, ok := s.mapNotification(*row, users)
	if !ok {
		return nil, notFound()
	}
	return &getNotificationOutput{Body: n}, nil
}
