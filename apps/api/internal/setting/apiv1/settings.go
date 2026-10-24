package apiv1

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/setting"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

var errUnconfigured = errors.New("apiv1 settings: service is not configured")

type Settings struct {
	IsAIConfigured                  bool   `json:"is_ai_configured" doc:"Whether the server has a model endpoint (KUN_AI_BASE_URL and KUN_AI_TOKEN). Without one the two switches below have no effect."`
	IsTopicClassifyEnabled          bool   `json:"is_topic_classify_enabled" doc:"Whether the model decides a new topic's section. Off, the author picks one."`
	IsTopicSummaryEnabled           bool   `json:"is_topic_summary_enabled" doc:"Whether the model writes summaries for long topics."`
	TopicAIModel                    string `json:"topic_ai_model" maxLength:"120" pattern:"^[A-Za-z0-9@_./:-]+$" doc:"Model id sent to the endpoint."`
	TopicAITimeoutMs                int    `json:"topic_ai_timeout_ms" minimum:"1000" maximum:"60000" doc:"How long publishing waits for the model before it falls back to the author's section."`
	TopicSummaryMinLength           int    `json:"topic_summary_min_length" minimum:"0" maximum:"100000" doc:"A body needs at least this many characters of text to get a summary."`
	TopicSummaryRegenerateThreshold int    `json:"topic_summary_regenerate_threshold" minimum:"1" maximum:"100000" doc:"Characters inserted plus deleted by edits since the summary was written before it is written again."`
}

type SettingsPatch struct {
	IsTopicClassifyEnabled          *bool   `json:"is_topic_classify_enabled,omitempty" doc:"New value."`
	IsTopicSummaryEnabled           *bool   `json:"is_topic_summary_enabled,omitempty" doc:"New value."`
	TopicAIModel                    *string `json:"topic_ai_model,omitempty" minLength:"1" maxLength:"120" pattern:"^[A-Za-z0-9@_./:-]+$" doc:"New value."`
	TopicAITimeoutMs                *int    `json:"topic_ai_timeout_ms,omitempty" minimum:"1000" maximum:"60000" doc:"New value."`
	TopicSummaryMinLength           *int    `json:"topic_summary_min_length,omitempty" minimum:"0" maximum:"100000" doc:"New value."`
	TopicSummaryRegenerateThreshold *int    `json:"topic_summary_regenerate_threshold,omitempty" minimum:"1" maximum:"100000" doc:"New value."`
}

type settingsOutput struct {
	Body Settings
}

type patchSettingsInput struct {
	Body SettingsPatch
}

type Service struct {
	store        *setting.Store
	aiConfigured bool
}

func New(store *setting.Store, aiConfigured bool) *Service {
	return &Service{store: store, aiConfigured: aiConfigured}
}

func (s *Service) admin(ctx context.Context) (int, *problem.Problem) {
	if s == nil || s.store == nil {
		return 0, problem.Internal(errUnconfigured)
	}
	user := v1.User(ctx)
	if user == nil {
		return 0, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	if !user.Can(perm.SettingManage) {
		return 0, problem.New(problem.CodePermissionRequired, "This needs "+string(perm.SettingManage)+".")
	}
	return user.ID, nil
}

func (s *Service) current() Settings {
	return Settings{
		IsAIConfigured:                  s.aiConfigured,
		IsTopicClassifyEnabled:          s.store.Bool(setting.TopicAIClassifyEnabled),
		IsTopicSummaryEnabled:           s.store.Bool(setting.TopicAISummaryEnabled),
		TopicAIModel:                    s.store.String(setting.TopicAIModel),
		TopicAITimeoutMs:                s.store.Int(setting.TopicAITimeoutMs),
		TopicSummaryMinLength:           s.store.Int(setting.TopicSummaryMinLength),
		TopicSummaryRegenerateThreshold: s.store.Int(setting.TopicSummaryRegenerateThreshold),
	}
}

type getSettingsInput struct{}

func (s *Service) getSettings(ctx context.Context, _ *getSettingsInput) (*settingsOutput, error) {
	if _, p := s.admin(ctx); p != nil {
		return nil, p
	}
	return &settingsOutput{Body: s.current()}, nil
}

func (s *Service) patchSettings(ctx context.Context, in *patchSettingsInput) (*settingsOutput, error) {
	uid, p := s.admin(ctx)
	if p != nil {
		return nil, p
	}
	b := in.Body
	values := map[string]any{}
	put := func(key string, v any, set bool) {
		if set {
			values[key] = v
		}
	}
	put(setting.TopicAIClassifyEnabled, deref(b.IsTopicClassifyEnabled), b.IsTopicClassifyEnabled != nil)
	put(setting.TopicAISummaryEnabled, deref(b.IsTopicSummaryEnabled), b.IsTopicSummaryEnabled != nil)
	put(setting.TopicAIModel, deref(b.TopicAIModel), b.TopicAIModel != nil)
	put(setting.TopicAITimeoutMs, deref(b.TopicAITimeoutMs), b.TopicAITimeoutMs != nil)
	put(setting.TopicSummaryMinLength, deref(b.TopicSummaryMinLength), b.TopicSummaryMinLength != nil)
	put(setting.TopicSummaryRegenerateThreshold, deref(b.TopicSummaryRegenerateThreshold), b.TopicSummaryRegenerateThreshold != nil)
	if len(values) > 0 {
		if err := s.store.Set(values, uid); err != nil {
			return nil, problem.Internal(err)
		}
	}
	return &settingsOutput{Body: s.current()}, nil
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		denied := func() map[string]*huma.Response {
			return map[string]*huma.Response{
				"403": {
					Description: "PERMISSION_REQUIRED without setting.manage, which a Bearer request never carries.",
					Content:     map[string]*huma.MediaType{problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}}},
				},
			}
		}
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getAdminSettings",
			Method:      http.MethodGet,
			Path:        "/admin/settings",
			Summary:     "Read the site settings",
			Description: "Every runtime setting with its current value; a setting nobody changed reads as its default. Needs setting.manage.",
			Tags:        []string{"admin"},
			Responses:   denied(),
		}), s.getSettings)
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateAdminSettings",
			Method:      http.MethodPatch,
			Path:        "/admin/settings",
			Summary:     "Change site settings",
			Description: fmt.Sprintf("Writes the settings that are sent and returns all of them. Other API instances see a change within %d seconds. Needs setting.manage. An empty object writes nothing.", 30),
			Tags:        []string{"admin"},
			Responses:   denied(),
		}), s.patchSettings)
	}
}
