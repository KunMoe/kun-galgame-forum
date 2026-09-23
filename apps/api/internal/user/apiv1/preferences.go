package apiv1

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

type getPreferencesOutput struct {
	ETag string `header:"ETag" maxLength:"32" doc:"Quoted document version, such as \"4\"."`
	Body Preferences
}

type putPreferencesInput struct {
	IfMatch string `header:"If-Match" maxLength:"21" pattern:"^\"[0-9]{1,19}\"$" doc:"Quoted document version to replace, such as \"4\". Absent means last-write-wins."`
	Body    putPreferencesBody
}

type putPreferencesBody struct {
	Doc map[string]any `json:"doc" doc:"Cloud preference document. A JSON object."`
}

type putPreferencesOutput struct {
	ETag string `header:"ETag" maxLength:"32" doc:"Quoted document version, such as \"4\"."`
	Body Preferences
}

type prefDoc struct {
	Doc       map[string]any `json:"doc"`
	Version   int            `json:"version"`
	UpdatedAt *string        `json:"updated_at"`
}

func (s *Users) getPreferences(ctx context.Context, _ *struct{}) (*getPreferencesOutput, error) {
	if prob := s.readyOAuth(); prob != nil {
		return nil, prob
	}
	if v1.User(ctx) == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	data, err := s.oauth.GetPreferences(accessToken(ctx))
	if err != nil {
		return nil, mapPreferencesError(err)
	}
	pref, err := parsePreferences(data)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &getPreferencesOutput{ETag: etagVersion(pref.Version), Body: pref}, nil
}

func (s *Users) putPreferences(ctx context.Context, in *putPreferencesInput) (*putPreferencesOutput, error) {
	if prob := s.readyOAuth(); prob != nil {
		return nil, prob
	}
	if v1.User(ctx) == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	doc := in.Body.Doc
	if doc == nil {
		doc = map[string]any{}
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, problem.Internal(err)
	}
	data, err := s.oauth.PutPreferences(accessToken(ctx), raw, in.IfMatch)
	if err != nil {
		return nil, mapPreferencesError(err)
	}
	pref, err := parsePreferences(data)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &putPreferencesOutput{ETag: etagVersion(pref.Version), Body: pref}, nil
}

func parsePreferences(data json.RawMessage) (Preferences, error) {
	var src prefDoc
	if err := json.Unmarshal(data, &src); err != nil {
		return Preferences{}, err
	}
	if src.Doc == nil {
		src.Doc = map[string]any{}
	}
	var updated *repr.DateTime
	if src.UpdatedAt != nil && *src.UpdatedAt != "" {
		t, err := time.Parse(time.RFC3339, *src.UpdatedAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, *src.UpdatedAt)
		}
		if err == nil {
			ts := repr.Timestamp(t)
			updated = &ts
		}
	}
	return Preferences{
		Object:    "preferences",
		Doc:       src.Doc,
		Version:   src.Version,
		WrittenAt: updated,
	}, nil
}

func etagVersion(version int) string {
	return `"` + strconv.Itoa(version) + `"`
}
