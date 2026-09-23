package apiv1

import (
	"context"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/message/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type listConversationsInput struct {
	collect.Page
}

type listConversationsOutput struct {
	Body repr.List[Conversation]
}

type conversationPathInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The other participant's user id. Equal to the caller is NOT_FOUND."`
}

type getConversationOutput struct {
	Body Conversation
}

type markDirectMessagesReadInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The other participant's user id. Equal to the caller is NOT_FOUND."`
	Body   DirectMessageReadMarkerWrite
}

type markDirectMessagesReadOutput struct {
	Body DirectMessageReadMarker
}

func parseConversationPos(keys []string) (*repository.V1ConversationPos, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 2 {
		return nil, invalidCursor()
	}
	at, err := time.Parse(time.RFC3339Nano, keys[0])
	if err != nil {
		return nil, invalidCursor()
	}
	id, err := strconv.Atoi(keys[1])
	if err != nil || id <= 0 {
		return nil, invalidCursor()
	}
	return &repository.V1ConversationPos{LastMessageAt: at, RoomID: id}, nil
}

func encodeConversationCursor(fp string, at time.Time, roomID int) string {
	return collect.EncodeCursor(conversationSort, fp, at.UTC().Format(time.RFC3339Nano), strconv.Itoa(roomID))
}

func (s *Service) emptyConversation(peer repr.UserRef) Conversation {
	return Conversation{
		Object:       "conversation",
		ID:           peer.ID,
		Peer:         peer,
		MessageCount: 0,
		UnreadCount:  0,
	}
}

func (s *Service) mapConversation(
	row repository.V1ConversationRow,
	peer repr.UserRef,
	callerID int,
	users map[int]userclient.User,
	doc content.ContentDocument,
) Conversation {
	sender, _ := s.userRef(users, row.LastSenderID)
	last := s.mapDirectMessage(lastMessageRow(row), sender, doc, callerID)
	at := last.CreatedAt
	return Conversation{
		Object:        "conversation",
		ID:            peer.ID,
		Peer:          peer,
		MessageCount:  row.MessageCount,
		UnreadCount:   row.UnreadCount,
		LastMessage:   &last,
		LastMessageAt: &at,
	}
}

func (s *Service) listConversations(ctx context.Context, in *listConversationsInput) (*listConversationsOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := mustUser(ctx)
	if p != nil {
		return nil, p
	}
	fp := conversationFingerprint(user.ID)
	keys, curErr := collect.DecodeCursor(in.Cursor, conversationSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	pos, posErr := parseConversationPos(keys)
	if posErr != nil {
		return nil, posErr
	}
	limit := in.Limit
	if limit <= 0 {
		limit = collect.DefaultLimit
	}

	type kept struct {
		row  repository.V1ConversationRow
		peer repr.UserRef
	}
	items := make([]kept, 0, limit+1)
	maxScan := pageRefillFactor * limit
	scanned := 0
	var last repository.V1ConversationRow
	haveLast := false
	exhausted := false
	for scanned < maxScan {
		batch := limit + 1
		if rem := maxScan - scanned; rem < batch {
			batch = rem
		}
		rows, err := s.chats.FindConversationsKeyset(user.ID, batch, pos)
		if err != nil {
			return nil, problem.Internal(err)
		}
		if len(rows) == 0 {
			exhausted = true
			break
		}
		ids := make([]int, 0, len(rows)*2)
		ids = append(ids, userclient.CollectIDs(rows, func(r repository.V1ConversationRow) int { return r.PeerID })...)
		ids = append(ids, userclient.CollectIDs(rows, func(r repository.V1ConversationRow) int { return r.LastSenderID })...)
		users := s.hydrateUsers(ctx, ids)
		for _, row := range rows {
			scanned++
			last = row
			haveLast = true
			peer, keep := s.userRef(users, row.PeerID)
			if keep {
				items = append(items, kept{row: row, peer: peer})
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
		pos = &repository.V1ConversationPos{LastMessageAt: last.LastMessageAt, RoomID: last.RoomID}
	}

	var next *string
	if len(items) > limit {
		lastKept := items[limit-1]
		items = items[:limit]
		cur := encodeConversationCursor(fp, lastKept.row.LastMessageAt, lastKept.row.RoomID)
		next = &cur
	} else if haveLast && scanned >= maxScan && !exhausted {
		cur := encodeConversationCursor(fp, last.LastMessageAt, last.RoomID)
		next = &cur
	}

	sources := make([]string, 0, len(items))
	srcIdx := make([]int, len(items))
	for i, it := range items {
		srcIdx[i] = -1
		if !it.row.LastIsRecall {
			srcIdx[i] = len(sources)
			sources = append(sources, it.row.LastContent)
		}
	}
	docs, p := s.convertBodies(ctx, sources)
	if p != nil {
		return nil, p
	}
	ids := make([]int, 0, len(items)*2)
	ids = append(ids, userclient.CollectIDs(items, func(it kept) int { return it.row.PeerID })...)
	ids = append(ids, userclient.CollectIDs(items, func(it kept) int { return it.row.LastSenderID })...)
	users := s.hydrateUsers(ctx, ids)
	out := make([]Conversation, len(items))
	for i, it := range items {
		doc := content.NewDocument(nil)
		if srcIdx[i] >= 0 {
			doc = docs[srcIdx[i]]
		}
		out[i] = s.mapConversation(it.row, it.peer, user.ID, users, doc)
	}
	return &listConversationsOutput{Body: repr.NewList(out, next)}, nil
}

func (s *Service) getConversation(ctx context.Context, in *conversationPathInput) (*getConversationOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := mustUser(ctx)
	if p != nil {
		return nil, p
	}
	peerID, peerUser, p := s.requirePeer(ctx, user.ID, in.UserID)
	if p != nil {
		return nil, p
	}
	peer := repr.NewUserRef(s.cdn, peerUser)
	row, err := s.chats.FindConversation(user.ID, peerID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if row == nil {
		return &getConversationOutput{Body: s.emptyConversation(peer)}, nil
	}
	users := s.hydrateUsers(ctx, []int{row.LastSenderID, peerID})
	doc := content.NewDocument(nil)
	if !row.LastIsRecall {
		docs, p := s.convertBodies(ctx, []string{row.LastContent})
		if p != nil {
			return nil, p
		}
		doc = docs[0]
	}
	return &getConversationOutput{Body: s.mapConversation(*row, peer, user.ID, users, doc)}, nil
}

func (s *Service) markDirectMessagesRead(ctx context.Context, in *markDirectMessagesReadInput) (*markDirectMessagesReadOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := mustUser(ctx)
	if p != nil {
		return nil, p
	}
	peerID, _, p := s.requirePeer(ctx, user.ID, in.UserID)
	if p != nil {
		return nil, p
	}
	upTo, p := parseUpToID(in.Body.UpToID)
	if p != nil {
		return nil, p
	}
	roomID, err := s.chats.FindRoomByName(repository.PrivateRoomName(user.ID, peerID))
	if err != nil {
		return nil, problem.Internal(err)
	}
	if roomID == 0 {
		return &markDirectMessagesReadOutput{Body: DirectMessageReadMarker{
			Object: "direct_message_read_marker",
		}}, nil
	}
	marked, unread, err := s.chats.MarkDirectReadUpTo(user.ID, roomID, upTo)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &markDirectMessagesReadOutput{Body: DirectMessageReadMarker{
		Object:      "direct_message_read_marker",
		MarkedCount: marked,
		UnreadCount: unread,
	}}, nil
}
