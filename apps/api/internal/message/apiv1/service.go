package apiv1

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/message/notifytype"
	"kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

const (
	notificationSort  = "created_desc"
	conversationSort  = "last_message_desc"
	directMessageSort = "id_desc"
	excerptRuneLimit  = 1000
	pageRefillFactor  = 10
)

type Service struct {
	messages *repository.MessageRepository
	chats    *repository.ChatRepository
	users    *userclient.Client
	convert  *content.Converter
	cdn      string
	forward  *msgService.MessageService
}

func New(
	messages *repository.MessageRepository,
	chats *repository.ChatRepository,
	users *userclient.Client,
	convert *content.Converter,
	cdn string,
	forward *msgService.MessageService,
) *Service {
	return &Service{
		messages: messages,
		chats:    chats,
		users:    users,
		convert:  convert,
		cdn:      cdn,
		forward:  forward,
	}
}

func (s *Service) lookupUsers(ctx context.Context, ids []int) (map[int]userclient.User, *problem.Problem) {
	if s.users == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	users, err := s.users.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if users == nil {
		users = map[int]userclient.User{}
	}
	return users, nil
}

func (s *Service) hydrateUsers(ctx context.Context, ids []int) map[int]userclient.User {
	if s.users == nil {
		return map[int]userclient.User{}
	}
	users, _ := s.users.Users(ctx, ids)
	if users == nil {
		return map[int]userclient.User{}
	}
	return users
}

func (s *Service) userRef(users map[int]userclient.User, id int) (repr.UserRef, bool) {
	if u, ok := users[id]; ok {
		if !userclient.IsRenderable(u) {
			return repr.UserRef{}, false
		}
		return repr.NewUserRef(s.cdn, u), true
	}
	return repr.DeletedUserRef(id), true
}

func (s *Service) requirePeer(ctx context.Context, callerID int, raw string) (int, userclient.User, *problem.Problem) {
	id, ok := parsePositiveID(raw)
	if !ok || id == callerID {
		return 0, userclient.User{}, notFound()
	}
	users, p := s.lookupUsers(ctx, []int{id})
	if p != nil {
		return 0, userclient.User{}, p
	}
	u, ok := users[id]
	if !ok || !userclient.IsRenderable(u) {
		return 0, userclient.User{}, notFound()
	}
	return id, u, nil
}

func (s *Service) mutedTypes(userID int) ([]string, bool, *problem.Problem) {
	keys, err := s.messages.FindMutedTypes(userID)
	if err != nil {
		return nil, false, problem.Internal(err)
	}
	local, chatMuted := msgService.SplitMuted(keys)
	return local, chatMuted, nil
}

func (s *Service) convertBodies(ctx context.Context, sources []string) ([]content.ContentDocument, *problem.Problem) {
	if s.convert == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	docs, err := s.convert.Convert(ctx, sources)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	return docs, nil
}

func (s *Service) forwardRead(userID int, ids []int64) {
	if s.forward == nil || len(ids) == 0 {
		return
	}
	s.forward.ForwardRead(userID, ids)
}

func mustUser(ctx context.Context) (*middleware.UserInfo, *problem.Problem) {
	user := v1.User(ctx)
	if user == nil {
		return nil, notFound()
	}
	return user, nil
}

func notificationFingerprint(userID int, muted bool, typ notifytype.Type) string {
	return collect.Fingerprint("notifications", strconv.Itoa(userID), strconv.FormatBool(muted), string(typ))
}

func conversationFingerprint(userID int) string {
	return collect.Fingerprint("conversations", strconv.Itoa(userID))
}

func directMessageFingerprint(userID, peerID int) string {
	return collect.Fingerprint("direct_messages", strconv.Itoa(userID), strconv.Itoa(peerID))
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n])
}

func notificationPath(link string) string {
	if strings.HasPrefix(link, "/") {
		if len(link) > 100 {
			return link[:100]
		}
		return link
	}
	if link == "" {
		return "/"
	}
	out := "/" + link
	if len(out) > 100 {
		return out[:100]
	}
	return out
}

func countAtLeastOne(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
