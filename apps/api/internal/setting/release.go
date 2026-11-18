package setting

// AppRelease is the version gate the App reads from /app/version. An empty
// download URL means the platform's download page; an empty package URL means
// no in-place update is published.
type AppRelease struct {
	MinVersion           string
	LatestVersion        string
	Notes                string
	AndroidDownloadURL   string
	IOSDownloadURL       string
	WindowsDownloadURL   string
	LinuxDownloadURL     string
	AndroidPackageURL    string
	AndroidPackageSize   int
	AndroidPackageSHA256 string
}

func (s *Store) AppRelease() AppRelease {
	return AppRelease{
		MinVersion:           s.String(AppMinVersion),
		LatestVersion:        s.String(AppLatestVersion),
		Notes:                s.String(AppReleaseNotes),
		AndroidDownloadURL:   s.String(AppAndroidDownloadURL),
		IOSDownloadURL:       s.String(AppIOSDownloadURL),
		WindowsDownloadURL:   s.String(AppWindowsDownloadURL),
		LinuxDownloadURL:     s.String(AppLinuxDownloadURL),
		AndroidPackageURL:    s.String(AppAndroidPackageURL),
		AndroidPackageSize:   s.Int(AppAndroidPackageSize),
		AndroidPackageSHA256: s.String(AppAndroidPackageSHA256),
	}
}
