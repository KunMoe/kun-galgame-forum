package apiv1

import (
	"context"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/message/notifytype"
	"kun-galgame-api/pkg/problem"
)

type getNotificationPreferencesOutput struct {
	Body NotificationPreferences
}

type putNotificationPreferencesInput struct {
	Body putNotificationPreferencesBody
}

type putNotificationPreferencesBody struct {
	MutedTypes []MutedType `json:"muted_types" maxItems:"19" uniqueItems:"true" doc:"Notification types to mute, replacing the stored set. Empty array mutes nothing."`
}

type putNotificationPreferencesOutput struct {
	Body NotificationPreferences
}

func (s *Users) getNotificationPreferences(ctx context.Context, _ *struct{}) (*getNotificationPreferencesOutput, error) {
	if prob := s.readyUsers(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	stored, err := s.users.MutedNotificationTypes(user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &getNotificationPreferencesOutput{Body: NotificationPreferences{
		Object:     "notification_preferences",
		MutedTypes: mutedFromStored(stored),
	}}, nil
}

func (s *Users) putNotificationPreferences(ctx context.Context, in *putNotificationPreferencesInput) (*putNotificationPreferencesOutput, error) {
	if prob := s.readyUsers(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	stored := mutedToStored(in.Body.MutedTypes)
	if err := s.users.ReplaceMutedNotificationTypes(user.ID, stored); err != nil {
		return nil, problem.Internal(err)
	}
	return &putNotificationPreferencesOutput{Body: NotificationPreferences{
		Object:     "notification_preferences",
		MutedTypes: mutedFromStored(stored),
	}}, nil
}

func mutedToStored(tokens []MutedType) []string {
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if string(tok) == notifytype.KeyChat {
			out = append(out, notifytype.KeyChat)
			continue
		}
		out = append(out, notifytype.ToDB(notifytype.Type(tok)))
	}
	return out
}

func mutedFromStored(dbKeys []string) []MutedType {
	out := make([]MutedType, 0, len(dbKeys))
	for _, db := range dbKeys {
		if db == notifytype.KeyChat {
			out = append(out, MutedType(notifytype.KeyChat))
			continue
		}
		t, ok := notifytype.FromDB(db)
		if !ok {
			continue
		}
		out = append(out, MutedType(t))
	}
	return out
}
