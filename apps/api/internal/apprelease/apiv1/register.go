package apiv1

import (
	"context"
	"errors"
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/setting"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

type AppDownloads struct {
	Android string `json:"android" format:"uri" maxLength:"512" doc:"Android package, or the download page while none is published."`
	IOS     string `json:"ios" format:"uri" maxLength:"512" doc:"iOS package, or the download page while none is published."`
	Windows string `json:"windows" format:"uri" maxLength:"512" doc:"Windows installer, or the download page while none is published."`
	Linux   string `json:"linux" format:"uri" maxLength:"512" doc:"Linux package, or the download page while none is published."`
}

type AndroidPackage struct {
	URL      string `json:"url" format:"uri" maxLength:"512" doc:"The .apk file itself, never a page. Always https."`
	FileSize int64  `json:"file_size" minimum:"1" doc:"Exact byte length of the file."`
	SHA256   string `json:"sha256" pattern:"^[0-9a-f]{64}$" minLength:"64" maxLength:"64" doc:"SHA-256 of the file, lowercase hex."`
}

type AppVersion struct {
	Object         string          `json:"object" enum:"app_version" maxLength:"11" doc:"Type discriminant. Always app_version."`
	MinVersion     string          `json:"min_version" pattern:"^[0-9]+\\.[0-9]+\\.[0-9]+$" maxLength:"32" doc:"Oldest app version still allowed to run; older installs must update. MAJOR.MINOR.PATCH."`
	LatestVersion  string          `json:"latest_version" pattern:"^[0-9]+\\.[0-9]+\\.[0-9]+$" maxLength:"32" doc:"Newest published app version. MAJOR.MINOR.PATCH, never below min_version."`
	Notes          string          `json:"notes" maxLength:"2000" doc:"Release notes of the latest version. Empty string if none. Free text; never use it as a decision input."`
	Downloads      AppDownloads    `json:"downloads" doc:"Where each platform downloads the latest version."`
	AndroidPackage *AndroidPackage `json:"android_package" doc:"The Android package of latest_version, for an in-place update. null while none is published: open downloads.android instead. Before installing, check the file's file_size and sha256 against these, and that the package's version is latest_version and its signing certificate is the running app's."`
}

type appVersionOutput struct {
	Body AppVersion
}

var errUnconfigured = errors.New("apiv1 app version: no settings store")

const downloadPage = "https://www.kungal.com/app"

type Service struct {
	settings *setting.Store
}

func New(settings *setting.Store) *Service {
	return &Service{settings: settings}
}

func orDownloadPage(url string) string {
	if url == "" {
		return downloadPage
	}
	return url
}

func (s *Service) getAppVersion(_ context.Context, _ *struct{}) (*appVersionOutput, error) {
	if s == nil || s.settings == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	r := s.settings.AppRelease()
	var pkg *AndroidPackage
	if r.AndroidPackageURL != "" {
		pkg = &AndroidPackage{URL: r.AndroidPackageURL, FileSize: int64(r.AndroidPackageSize), SHA256: r.AndroidPackageSHA256}
	}
	return &appVersionOutput{Body: AppVersion{
		Object:        "app_version",
		MinVersion:    r.MinVersion,
		LatestVersion: r.LatestVersion,
		Notes:         r.Notes,
		Downloads: AppDownloads{
			Android: orDownloadPage(r.AndroidDownloadURL),
			IOS:     orDownloadPage(r.IOSDownloadURL),
			Windows: orDownloadPage(r.WindowsDownloadURL),
			Linux:   orDownloadPage(r.LinuxDownloadURL),
		},
		AndroidPackage: pkg,
	}}, nil
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "getAppVersion",
			Method:      http.MethodGet,
			Path:        "/app/version",
			Summary:     "Get the app version gate",
			Description: "The app's version gate: an install older than min_version must update before it runs. " +
				"Read from the site settings, which are validated when an admin writes them, so the versions are always well formed and ordered.",
			Tags: []string{"app"},
		}), s.getAppVersion)
	}
}
