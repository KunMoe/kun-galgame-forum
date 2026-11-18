package apiv1

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/setting"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/platformsettings"
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

	AppMinVersion           string `json:"app_min_version" maxLength:"32" pattern:"^[0-9]+\\.[0-9]+\\.[0-9]+$" doc:"Oldest App version still allowed to run. MAJOR.MINOR.PATCH, never above app_latest_version."`
	AppLatestVersion        string `json:"app_latest_version" maxLength:"32" pattern:"^[0-9]+\\.[0-9]+\\.[0-9]+$" doc:"Newest published App version. MAJOR.MINOR.PATCH."`
	AppReleaseNotes         string `json:"app_release_notes" maxLength:"2000" doc:"Release notes of the latest version; empty for none. Free text; never use it as a decision input."`
	AppAndroidDownloadURL   string `json:"app_android_download_url" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"Android download; empty for the download page."`
	AppIOSDownloadURL       string `json:"app_ios_download_url" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"iOS download; empty for the download page."`
	AppWindowsDownloadURL   string `json:"app_windows_download_url" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"Windows download; empty for the download page."`
	AppLinuxDownloadURL     string `json:"app_linux_download_url" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"Linux download; empty for the download page."`
	AppAndroidPackageURL    string `json:"app_android_package_url" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"The .apk of app_latest_version for the App's in-place update; empty for none. Set together with app_android_package_size and app_android_package_sha256."`
	AppAndroidPackageSize   int    `json:"app_android_package_size" minimum:"0" doc:"Byte length of that .apk; 0 for none."`
	AppAndroidPackageSHA256 string `json:"app_android_package_sha256" maxLength:"64" pattern:"^([0-9a-f]{64})?$" doc:"SHA-256 of that .apk, lowercase hex; empty for none."`

	IsTrustCheckEnabled bool `json:"is_trust_check_enabled" doc:"Whether user text is checked against the word list before it is written."`
	IsTrustScanEnabled  bool `json:"is_trust_scan_enabled" doc:"Whether user text is sent for a scan after it is written."`

	NotificationRetentionDays int `json:"notification_retention_days" minimum:"0" maximum:"3650" doc:"Days a read notification is kept before the daily job deletes it; unread ones are never deleted. 0 keeps them all."`

	DlsiteLinkTemplate string `json:"dlsite_link_template" maxLength:"512" pattern:"^(https://\\S*\\{workno\\}\\S*)?$" doc:"DLsite affiliate link with {workno} in place of the work number, used until a short link is minted; empty turns the purchase link off."`
	DlsiteCouponURL    string `json:"dlsite_coupon_url" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"Standing coupon landing page, shown while the store face reports no campaign; empty for none."`
}

type SettingsPatch struct {
	IsTopicClassifyEnabled          *bool   `json:"is_topic_classify_enabled,omitempty" doc:"New value."`
	IsTopicSummaryEnabled           *bool   `json:"is_topic_summary_enabled,omitempty" doc:"New value."`
	TopicAIModel                    *string `json:"topic_ai_model,omitempty" minLength:"1" maxLength:"120" pattern:"^[A-Za-z0-9@_./:-]+$" doc:"New value."`
	TopicAITimeoutMs                *int    `json:"topic_ai_timeout_ms,omitempty" minimum:"1000" maximum:"60000" doc:"New value."`
	TopicSummaryMinLength           *int    `json:"topic_summary_min_length,omitempty" minimum:"0" maximum:"100000" doc:"New value."`
	TopicSummaryRegenerateThreshold *int    `json:"topic_summary_regenerate_threshold,omitempty" minimum:"1" maximum:"100000" doc:"New value."`

	AppMinVersion           *string `json:"app_min_version,omitempty" maxLength:"32" pattern:"^[0-9]+\\.[0-9]+\\.[0-9]+$" doc:"New value."`
	AppLatestVersion        *string `json:"app_latest_version,omitempty" maxLength:"32" pattern:"^[0-9]+\\.[0-9]+\\.[0-9]+$" doc:"New value."`
	AppReleaseNotes         *string `json:"app_release_notes,omitempty" maxLength:"2000" doc:"New value. Free text; never use it as a decision input."`
	AppAndroidDownloadURL   *string `json:"app_android_download_url,omitempty" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"New value."`
	AppIOSDownloadURL       *string `json:"app_ios_download_url,omitempty" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"New value."`
	AppWindowsDownloadURL   *string `json:"app_windows_download_url,omitempty" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"New value."`
	AppLinuxDownloadURL     *string `json:"app_linux_download_url,omitempty" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"New value."`
	AppAndroidPackageURL    *string `json:"app_android_package_url,omitempty" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"New value."`
	AppAndroidPackageSize   *int    `json:"app_android_package_size,omitempty" minimum:"0" doc:"New value."`
	AppAndroidPackageSHA256 *string `json:"app_android_package_sha256,omitempty" maxLength:"64" pattern:"^([0-9a-f]{64})?$" doc:"New value."`

	IsTrustCheckEnabled *bool `json:"is_trust_check_enabled,omitempty" doc:"New value."`
	IsTrustScanEnabled  *bool `json:"is_trust_scan_enabled,omitempty" doc:"New value."`

	NotificationRetentionDays *int `json:"notification_retention_days,omitempty" minimum:"0" maximum:"3650" doc:"New value."`

	DlsiteLinkTemplate *string `json:"dlsite_link_template,omitempty" maxLength:"512" pattern:"^(https://\\S*\\{workno\\}\\S*)?$" doc:"New value."`
	DlsiteCouponURL    *string `json:"dlsite_coupon_url,omitempty" maxLength:"512" pattern:"^(https://\\S+)?$" doc:"New value."`
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
	platform     *platformsettings.Client
}

func New(store *setting.Store, aiConfigured bool, platform *platformsettings.Client) *Service {
	return &Service{store: store, aiConfigured: aiConfigured, platform: platform}
}

type SiteFeatures struct {
	Object         string `json:"object" enum:"site_features" maxLength:"13" doc:"Type discriminant. Always site_features."`
	IsAgentEnabled bool   `json:"is_agent_enabled" doc:"Whether NextMoe has turned the AI assistant on for this site. Off, clients hide the assistant's entrances and its calls are refused."`
}

type siteFeaturesOutput struct {
	Body SiteFeatures
}

func (s *Service) getSiteFeatures(_ context.Context, _ *struct{}) (*siteFeaturesOutput, error) {
	var platform *platformsettings.Client
	if s != nil {
		platform = s.platform
	}
	return &siteFeaturesOutput{Body: SiteFeatures{
		Object:         "site_features",
		IsAgentEnabled: platform.Bool(platformsettings.AgentEnabled),
	}}, nil
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
	r := s.store.AppRelease()
	return Settings{
		AppMinVersion:             r.MinVersion,
		AppLatestVersion:          r.LatestVersion,
		AppReleaseNotes:           r.Notes,
		AppAndroidDownloadURL:     r.AndroidDownloadURL,
		AppIOSDownloadURL:         r.IOSDownloadURL,
		AppWindowsDownloadURL:     r.WindowsDownloadURL,
		AppLinuxDownloadURL:       r.LinuxDownloadURL,
		AppAndroidPackageURL:      r.AndroidPackageURL,
		AppAndroidPackageSize:     r.AndroidPackageSize,
		AppAndroidPackageSHA256:   r.AndroidPackageSHA256,
		IsTrustCheckEnabled:       s.store.Bool(setting.TrustCheckEnabled),
		IsTrustScanEnabled:        s.store.Bool(setting.TrustScanEnabled),
		NotificationRetentionDays: s.store.Int(setting.NotificationRetentionDays),
		DlsiteLinkTemplate:        s.store.String(setting.DlsiteLinkTemplate),
		DlsiteCouponURL:           s.store.String(setting.DlsiteCouponURL),

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
	put(setting.AppMinVersion, deref(b.AppMinVersion), b.AppMinVersion != nil)
	put(setting.AppLatestVersion, deref(b.AppLatestVersion), b.AppLatestVersion != nil)
	put(setting.AppReleaseNotes, deref(b.AppReleaseNotes), b.AppReleaseNotes != nil)
	put(setting.AppAndroidDownloadURL, deref(b.AppAndroidDownloadURL), b.AppAndroidDownloadURL != nil)
	put(setting.AppIOSDownloadURL, deref(b.AppIOSDownloadURL), b.AppIOSDownloadURL != nil)
	put(setting.AppWindowsDownloadURL, deref(b.AppWindowsDownloadURL), b.AppWindowsDownloadURL != nil)
	put(setting.AppLinuxDownloadURL, deref(b.AppLinuxDownloadURL), b.AppLinuxDownloadURL != nil)
	put(setting.AppAndroidPackageURL, deref(b.AppAndroidPackageURL), b.AppAndroidPackageURL != nil)
	put(setting.AppAndroidPackageSize, deref(b.AppAndroidPackageSize), b.AppAndroidPackageSize != nil)
	put(setting.AppAndroidPackageSHA256, deref(b.AppAndroidPackageSHA256), b.AppAndroidPackageSHA256 != nil)
	put(setting.TrustCheckEnabled, deref(b.IsTrustCheckEnabled), b.IsTrustCheckEnabled != nil)
	put(setting.TrustScanEnabled, deref(b.IsTrustScanEnabled), b.IsTrustScanEnabled != nil)
	put(setting.NotificationRetentionDays, deref(b.NotificationRetentionDays), b.NotificationRetentionDays != nil)
	put(setting.DlsiteLinkTemplate, deref(b.DlsiteLinkTemplate), b.DlsiteLinkTemplate != nil)
	put(setting.DlsiteCouponURL, deref(b.DlsiteCouponURL), b.DlsiteCouponURL != nil)
	if p := checkRelease(mergeRelease(s.store.AppRelease(), b)); p != nil {
		return nil, p
	}
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
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "getSiteFeatures",
			Method:      http.MethodGet,
			Path:        "/site/features",
			Summary:     "Read which site features are on",
			Description: "Switches the platform owns for this site, as NextMoe's config center last published them. The API polls them every minute, so a change shows within about a minute.",
			Tags:        []string{"meta"},
		}), s.getSiteFeatures)
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
