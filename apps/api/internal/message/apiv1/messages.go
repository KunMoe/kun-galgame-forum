package apiv1

import (
	"context"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/message/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type listDirectMessagesInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The other participant's user id. Equal to the caller is NOT_FOUND."`
	collect.Page
}

type listDirectMessagesOutput struct {
	Body repr.List[DirectMessage]
}

type sendDirectMessageInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The other participant's user id. Equal to the caller is NOT_FOUND."`
	Body   DirectMessageCreate
}

type sendDirectMessageOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"96" doc:"Canonical path of the new message, such as /api/v1/me/conversations/3/messages/41."`
	Body     DirectMessage
}

type getDirectMessageInput struct {
	UserID    string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The other participant's user id. Equal to the caller is NOT_FOUND."`
	MessageID string `path:"message_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Message id."`
}

type getDirectMessageOutput struct {
	Body DirectMessage
}

type updateDirectMessageInput struct {
	UserID    string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The other participant's user id. Equal to the caller is NOT_FOUND."`
	MessageID string `path:"message_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Message id."`
	Body      DirectMessagePatch
}

type updateDirectMessageOutput struct {
	Body DirectMessage
}

func parseMessagePos(keys []string) (*repository.V1MessagePos, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 1 {
		return nil, invalidCursor()
	}
	id, err := strconv.Atoi(keys[0])
	if err != nil || id <= 0 {
		return nil, invalidCursor()
	}
	return &repository.V1MessagePos{ID: id}, nil
}

func (s *Service) mapMessagePage(ctx context.Context, rows []repository.V1ChatMessageRow, callerID int) ([]DirectMessage, *problem.Problem) {
	if len(rows) == 0 {
		return []DirectMessage{}, nil
	}
	users := s.hydrateUsers(ctx, userclient.CollectIDs(rows, func(r repository.V1ChatMessageRow) int { return r.SenderID }))
	sources := make([]string, 0, len(rows))
	srcIdx := make([]int, len(rows))
	for i, row := range rows {
		srcIdx[i] = -1
		if !row.IsRecall {
			srcIdx[i] = len(sources)
			sources = append(sources, row.Content)
		}
	}
	docs, p := s.convertBodies(ctx, sources)
	if p != nil {
		return nil, p
	}
	out := make([]DirectMessage, len(rows))
	for i, row := range rows {
		sender, _ := s.userRef(users, row.SenderID)
		doc := content.NewDocument(nil)
		if srcIdx[i] >= 0 {
			doc = docs[srcIdx[i]]
		}
		out[i] = s.mapDirectMessage(row, sender, doc, callerID)
	}
	return out, nil
}

func (s *Service) listDirectMessages(ctx context.Context, in *listDirectMessagesInput) (*listDirectMessagesOutput, error) {
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
	fp := directMessageFingerprint(user.ID, peerID)
	keys, curErr := collect.DecodeCursor(in.Cursor, directMessageSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	pos, posErr := parseMessagePos(keys)
	if posErr != nil {
		return nil, posErr
	}
	limit := in.Limit
	if limit <= 0 {
		limit = collect.DefaultLimit
	}
	roomID, err := s.chats.FindRoomByName(repository.PrivateRoomName(user.ID, peerID))
	if err != nil {
		return nil, problem.Internal(err)
	}
	if roomID == 0 {
		return &listDirectMessagesOutput{Body: repr.NewList([]DirectMessage{}, nil)}, nil
	}
	rows, err := s.chats.FindDirectMessagesKeyset(roomID, limit+1, pos)
	if err != nil {
		return nil, problem.Internal(err)
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	items, p := s.mapMessagePage(ctx, rows, user.ID)
	if p != nil {
		return nil, p
	}
	var next *string
	if hasMore && len(rows) > 0 {
		cur := collect.EncodeCursor(directMessageSort, fp, strconv.Itoa(rows[len(rows)-1].ID))
		next = &cur
	}
	return &listDirectMessagesOutput{Body: repr.NewList(items, next)}, nil
}

func (s *Service) sendDirectMessage(ctx context.Context, in *sendDirectMessageInput) (*sendDirectMessageOutput, error) {
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
	normalized := markdown.NormalizeStoredContent(in.Body.ContentMarkdown)
	if strings.TrimSpace(normalized) == "" {
		min := 1
		return nil, validationFailed(problem.AtPointer("/content_markdown", problem.ReasonRequired,
			"content_markdown was blank after trimming whitespace",
			&problem.FieldParams{MinLength: &min}))
	}
	docs, p := s.convertBodies(ctx, []string{normalized})
	if p != nil {
		return nil, p
	}
	row, err := s.chats.SendDirectMessage(user.ID, peerID, user.Name, normalized)
	if err != nil {
		return nil, problem.Internal(err)
	}
	users := s.hydrateUsers(ctx, []int{user.ID})
	sender, _ := s.userRef(users, user.ID)
	body := s.mapDirectMessage(*row, sender, docs[0], user.ID)
	return &sendDirectMessageOutput{
		Location: "/api/v1/me/conversations/" + in.UserID + "/messages/" + strconv.Itoa(row.ID),
		Body:     body,
	}, nil
}

func (s *Service) updateDirectMessage(ctx context.Context, in *updateDirectMessageInput) (*updateDirectMessageOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := mustUser(ctx)
	if p != nil {
		return nil, p
	}
	peerID, ok := parsePositiveID(in.UserID)
	if !ok || peerID == user.ID {
		return nil, notFound()
	}
	msgID, ok := parsePositiveID(in.MessageID)
	if !ok {
		return nil, notFound()
	}
	roomID, err := s.chats.FindRoomByName(repository.PrivateRoomName(user.ID, peerID))
	if err != nil {
		return nil, problem.Internal(err)
	}
	if roomID == 0 {
		return nil, notFound()
	}
	row, err := s.chats.FindDirectMessage(msgID, roomID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if row == nil {
		return nil, notFound()
	}
	if row.SenderID != user.ID {
		return nil, permissionRequired()
	}
	if !row.IsRecall {
		now := time.Now().UTC()
		if err := s.chats.RecallDirectMessage(row.ID, roomID, now); err != nil {
			return nil, problem.Internal(err)
		}
		row.IsRecall = true
		row.RecallTime = &now
	}
	users := s.hydrateUsers(ctx, []int{row.SenderID})
	sender, _ := s.userRef(users, row.SenderID)
	docs, p := s.convertBodies(ctx, []string{row.Content})
	if p != nil {
		return nil, p
	}
	return &updateDirectMessageOutput{Body: s.mapDirectMessage(*row, sender, docs[0], user.ID)}, nil
}

func (s *Service) getDirectMessage(ctx context.Context, in *getDirectMessageInput) (*getDirectMessageOutput, error) {
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
	msgID, ok := parsePositiveID(in.MessageID)
	if !ok {
		return nil, notFound()
	}
	roomID, err := s.chats.FindRoomByName(repository.PrivateRoomName(user.ID, peerID))
	if err != nil {
		return nil, problem.Internal(err)
	}
	if roomID == 0 {
		return nil, notFound()
	}
	row, err := s.chats.FindDirectMessage(msgID, roomID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if row == nil {
		return nil, notFound()
	}
	users := s.hydrateUsers(ctx, []int{row.SenderID})
	sender, _ := s.userRef(users, row.SenderID)
	docs, p := s.convertBodies(ctx, []string{row.Content})
	if p != nil {
		return nil, p
	}
	return &getDirectMessageOutput{Body: s.mapDirectMessage(*row, sender, docs[0], user.ID)}, nil
}
