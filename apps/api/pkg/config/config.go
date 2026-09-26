package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	OAuth          OAuthConfig
	FileStorage    S3Config
	CORS           CORSConfig
	NextMoeAPI     NextMoeAPIConfig
	NewsAPI        NewsAPIConfig
	MoyuAPI        MoyuAPIConfig
	StickerAPI     StickerAPIConfig
	ImageClient    ImageClientConfig
	ArtifactClient ArtifactClientConfig
	LinkChecker    LinkCheckerConfig
	Trust          TrustConfig
	Catalog        CatalogClientConfig
	Community      CommunityConfig
	Dlsite         DlsiteConfig
	Lottery        LotteryConfig
	Bearer         BearerConfig
	AppRelease     AppReleaseConfig
}

// ClientIDs is the allow-list of OAuth clients whose user access tokens may
// call this API directly; empty closes the Bearer channel. JWKSURL defaults to
// the OP's internal address, while Issuer must be the OP's PUBLIC origin: that
// is what it stamps into iss.
type BearerConfig struct {
	Issuer    string
	JWKSURL   string
	ClientIDs []string
}

func (c BearerConfig) Enabled() bool { return len(c.ClientIDs) > 0 }

type AppReleaseConfig struct {
	MinVersion     string
	LatestVersion  string
	Notes          string
	Downloads      AppDownloads
	AndroidPackage *AndroidPackage
}

type AndroidPackage struct {
	URL    string
	Size   int64
	SHA256 string
}

type AppDownloads struct {
	Android string
	IOS     string
	Windows string
	Linux   string
}

// CodeKey seals the escrowed lottery activation codes (AES-256-GCM, 64 hex
// chars). Leaving it empty is a supported state: the lottery mini-app still
// works, it just refuses prizes delivered as a code rather than storing them in
// the clear. Rotating it strands every code already sealed with the old key.
type LotteryConfig struct {
	CodeKey string
}

type CommunityConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
}

type CatalogClientConfig struct {
	BaseURL string
}

type TrustConfig struct {
	BaseURL        string
	CallbackSecret string
	Site           string
	CheckEnabled   bool
	ScanEnabled    bool
}

type ArtifactClientConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
}

type LinkCheckerConfig struct {
	BaseURL              string
	APIKey               string
	CFAccessClientID     string
	CFAccessClientSecret string
}

type ImageClientConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
}

type NextMoeAPIConfig struct {
	BaseURL      string
	APIKey       string
	ImageCDNBase string
}

// NewsAPIConfig is a SECOND NextMoe credential, not a copy of the first: the
// news face is gated on scope news:read, which the catalog key does not carry.
// An empty APIKey leaves /news answering 503 instead of failing startup — the
// forum's catalogue must not stop booting over a partner index.
type NewsAPIConfig struct {
	BaseURL string
	APIKey  string
}

// MoyuAPIConfig reaches the /v2/moyu patch face. Its base must NOT default to
// KUN_NEXTMOE_API_BASE the way news and store do: that is the catalog process
// (http://catalog:9281 in prod), and /v2/moyu exists only on the api.nextmoe.dev
// gateway, which checks the key and forwards to moyu's backend. The face admits
// any valid key without a scope, so the key defaults to the catalog one; the
// gateway's rate tier belongs to the OAuth client, and the forum's is internal
// (unlimited), so sharing it costs nothing.
type MoyuAPIConfig struct {
	BaseURL string
	APIKey  string
}

// StickerAPIConfig reaches the /v2/sticker face, which sits on the same gateway
// under the same rule as MoyuAPIConfig: any valid key, no scope.
type StickerAPIConfig struct {
	BaseURL string
	APIKey  string
}

// The affiliate link is assembled SERVER-side and shipped as a ready URL: the
// affiliate id stays out of the browser bundle, and this project's frontend
// build cannot be trusted with env vars (NUXT_PUBLIC_* / process.env.* come out
// undefined in the generic prod image), so a template baked into the frontend
// would silently produce broken links in production.
//
// LinkTemplate is a whole template, not assembled parts: DLsite's affiliate path
// differs per site segment, so a path change stays an env edit.
//
// StoreAPIKey reaches infra's /v2/store face, which mints the per-site short
// link that carries click attribution. It is a THIRD developer key, separate
// from the catalog and news ones: the face is gated on the scope store:read and
// the v2 limiter buckets per key, so minting a few thousand links must not spend
// the catalogue's minute budget. The links belong to the OAuth client rather
// than the key, so a separate key does not split attribution.
type DlsiteConfig struct {
	LinkTemplate string
	CouponURL    string
	StoreAPIBase string
	StoreAPIKey  string
}

func (c DlsiteConfig) Configured() bool { return c.LinkTemplate != "" }

func (c DlsiteConfig) StoreConfigured() bool { return c.StoreAPIBase != "" && c.StoreAPIKey != "" }

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type OAuthConfig struct {
	ServerURL    string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	JWTSecret    string
}

type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

type CORSConfig struct {
	AllowOrigins string
}

func Load() (*Config, error) {
	dbURL, err := requireEnv("KUN_DATABASE_URL")
	if err != nil {
		return nil, err
	}
	oauthServerURL, err := requireEnv("OAUTH_SERVER_URL")
	if err != nil {
		return nil, err
	}
	oauthClientID, err := requireEnv("OAUTH_CLIENT_ID")
	if err != nil {
		return nil, err
	}
	oauthClientSecret, err := requireEnv("OAUTH_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}
	oauthRedirectURI, err := requireEnv("OAUTH_REDIRECT_URI")
	if err != nil {
		return nil, err
	}

	nextMoeBase := envOrDefault("KUN_NEXTMOE_API_BASE", "http://127.0.0.1:9281")
	nextMoeKey := envOrDefault("KUN_NEXTMOE_API_KEY", "")
	if nextMoeBase != "" && nextMoeKey == "" {
		return nil, fmt.Errorf(
			"KUN_NEXTMOE_API_KEY 未设置: catalog /v2 读面硬依赖 nmk_ developer API key; 已配置 KUN_NEXTMOE_API_BASE=%q 但 KUN_NEXTMOE_API_KEY 为空, 不做静默降级",
			nextMoeBase,
		)
	}

	oauthOrigin := oauthOriginOf(oauthServerURL)

	appRelease, err := loadAppRelease()
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			Port: envOrDefault("SERVER_PORT", "2334"),
			Mode: envOrDefault("SERVER_MODE", "dev"),
		},
		Database: DatabaseConfig{
			URL:             dbURL,
			MaxOpenConns:    envOrDefaultInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envOrDefaultInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: envOrDefaultInt("DB_CONN_MAX_LIFETIME", 300),
		},
		Redis: RedisConfig{
			Host:     envOrDefault("REDIS_HOST", "127.0.0.1"),
			Port:     envOrDefault("REDIS_PORT", "6379"),
			Password: envOrDefault("REDIS_PASSWORD", ""),
			DB:       envOrDefaultInt("REDIS_DB", 0),
		},
		OAuth: OAuthConfig{
			ServerURL:    oauthServerURL,
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
			RedirectURI:  oauthRedirectURI,
			JWTSecret:    envOrDefault("JWT_SECRET", ""),
		},
		FileStorage: S3Config{
			Endpoint:  envOrDefault("FILE_STORAGE_ENDPOINT", ""),
			Region:    envOrDefault("FILE_STORAGE_REGION", ""),
			Bucket:    envOrDefault("FILE_STORAGE_BUCKET", ""),
			AccessKey: envOrDefault("FILE_STORAGE_ACCESS_KEY", ""),
			SecretKey: envOrDefault("FILE_STORAGE_SECRET_KEY", ""),
		},
		CORS: CORSConfig{
			AllowOrigins: envOrDefault(
				"CORS_ALLOW_ORIGINS",
				"http://127.0.0.1:2333,https://www.kungal.com",
			),
		},
		NextMoeAPI: NextMoeAPIConfig{
			BaseURL:      nextMoeBase,
			APIKey:       nextMoeKey,
			ImageCDNBase: envOrDefault("KUN_IMAGE_PUBLIC_BASE_URL", "https://image.kungal.iloveren.link"),
		},
		NewsAPI: NewsAPIConfig{
			BaseURL: envOrDefault("KUN_NEWS_API_BASE", nextMoeBase),
			APIKey:  envOrDefault("KUN_NEWS_API_KEY", ""),
		},
		MoyuAPI: MoyuAPIConfig{
			BaseURL: envOrDefault("KUN_MOYU_API_BASE", "https://api.nextmoe.dev"),
			APIKey:  envOrDefault("KUN_MOYU_API_KEY", nextMoeKey),
		},
		StickerAPI: StickerAPIConfig{
			BaseURL: envOrDefault("KUN_STICKER_API_BASE", "https://api.nextmoe.dev"),
			APIKey:  envOrDefault("KUN_STICKER_API_KEY", nextMoeKey),
		},
		ImageClient: ImageClientConfig{
			BaseURL:      envOrDefault("KUN_IMAGE_CLIENT_BASE_URL", "http://127.0.0.1:9278"),
			ClientID:     envOrDefault("KUN_IMAGE_CLIENT_ID", ""),
			ClientSecret: envOrDefault("KUN_IMAGE_CLIENT_SECRET", ""),
		},
		ArtifactClient: ArtifactClientConfig{
			BaseURL:      envOrDefault("KUN_ARTIFACT_CLIENT_BASE_URL", "http://127.0.0.1:9279"),
			ClientID:     envOrDefault("KUN_ARTIFACT_CLIENT_ID", ""),
			ClientSecret: envOrDefault("KUN_ARTIFACT_CLIENT_SECRET", ""),
		},
		LinkChecker: LinkCheckerConfig{
			BaseURL:              envOrDefault("LINK_CHECKER_BASE_URL", ""),
			APIKey:               envOrDefault("LINK_CHECKER_API_KEY", ""),
			CFAccessClientID:     envOrDefault("CF_ACCESS_CLIENT_ID", ""),
			CFAccessClientSecret: envOrDefault("CF_ACCESS_CLIENT_SECRET", ""),
		},
		Trust: TrustConfig{
			BaseURL:        envOrDefault("KUN_TRUST_BASE_URL", "http://127.0.0.1:9283"),
			CallbackSecret: envOrDefault("KUN_TRUST_CALLBACK_SECRET", ""),
			Site:           envOrDefault("KUN_TRUST_SITE", "kungal"),
			CheckEnabled:   envOrDefaultBool("KUN_TRUST_CHECK_ENABLED", false),
			ScanEnabled:    envOrDefaultBool("KUN_TRUST_SCAN_ENABLED", false),
		},
		Catalog: CatalogClientConfig{
			BaseURL: envOrDefault("KUN_CATALOG_API_BASE", "http://127.0.0.1:9281"),
		},
		Lottery: LotteryConfig{
			CodeKey: envOrDefault("KUN_LOTTERY_CODE_KEY", ""),
		},
		Community: CommunityConfig{
			BaseURL:      envOrDefault("KUN_COMMUNITY_API_BASE", ""),
			ClientID:     envOrDefault("KUN_COMMUNITY_CLIENT_ID", ""),
			ClientSecret: envOrDefault("KUN_COMMUNITY_CLIENT_SECRET", ""),
		},
		Dlsite: DlsiteConfig{
			LinkTemplate: envOrDefault("KUN_DLSITE_LINK_TEMPLATE", ""),
			CouponURL:    envOrDefault("KUN_DLSITE_COUPON_URL", ""),
			StoreAPIBase: envOrDefault("KUN_STORE_API_BASE", nextMoeBase),
			StoreAPIKey:  envOrDefault("KUN_STORE_API_KEY", ""),
		},
		Bearer: BearerConfig{
			Issuer:    envOrDefault("KUN_OIDC_ISSUER", oauthOrigin),
			JWKSURL:   envOrDefault("KUN_OIDC_JWKS_URL", oauthOrigin+"/oauth/jwks"),
			ClientIDs: splitCSV(os.Getenv("KUN_BEARER_CLIENT_IDS")),
		},
		AppRelease: appRelease,
	}, nil
}

func oauthOriginOf(serverURL string) string {
	s := strings.TrimRight(serverURL, "/")
	for _, suffix := range []string{"/api/v1", "/api"} {
		if base, ok := strings.CutSuffix(s, suffix); ok {
			return base
		}
	}
	return s
}

const appDownloadPage = "https://www.kungal.com/app"

var releaseVersionRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)

// A malformed or inverted pair is refused at boot: the App force-updates any
// install below min_version, so min above latest would send every user to a
// build that does not exist.
func loadAppRelease() (AppReleaseConfig, error) {
	cfg := AppReleaseConfig{
		MinVersion:    envOrDefault("KUN_APP_MIN_VERSION", "0.1.0"),
		LatestVersion: envOrDefault("KUN_APP_LATEST_VERSION", "0.1.0"),
		Notes:         envOrDefault("KUN_APP_RELEASE_NOTES", ""),
		Downloads: AppDownloads{
			Android: envOrDefault("KUN_APP_DOWNLOAD_ANDROID", appDownloadPage),
			IOS:     envOrDefault("KUN_APP_DOWNLOAD_IOS", appDownloadPage),
			Windows: envOrDefault("KUN_APP_DOWNLOAD_WINDOWS", appDownloadPage),
			Linux:   envOrDefault("KUN_APP_DOWNLOAD_LINUX", appDownloadPage),
		},
	}
	minV, err := parseReleaseVersion("KUN_APP_MIN_VERSION", cfg.MinVersion)
	if err != nil {
		return cfg, err
	}
	latestV, err := parseReleaseVersion("KUN_APP_LATEST_VERSION", cfg.LatestVersion)
	if err != nil {
		return cfg, err
	}
	if slices.Compare(minV, latestV) > 0 {
		return cfg, fmt.Errorf("KUN_APP_MIN_VERSION=%s 高于 KUN_APP_LATEST_VERSION=%s", cfg.MinVersion, cfg.LatestVersion)
	}
	cfg.AndroidPackage, err = loadAndroidPackage()
	return cfg, err
}

var sha256HexRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

func loadAndroidPackage() (*AndroidPackage, error) {
	rawURL := os.Getenv("KUN_APP_ANDROID_PACKAGE_URL")
	rawSize := os.Getenv("KUN_APP_ANDROID_PACKAGE_SIZE")
	sum := os.Getenv("KUN_APP_ANDROID_PACKAGE_SHA256")
	if rawURL == "" && rawSize == "" && sum == "" {
		return nil, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || len(rawURL) > 512 {
		return nil, fmt.Errorf("KUN_APP_ANDROID_PACKAGE_URL=%q 不是 512 字符以内的 https 链接", rawURL)
	}
	size, err := strconv.ParseInt(rawSize, 10, 64)
	if err != nil || size <= 0 {
		return nil, fmt.Errorf("KUN_APP_ANDROID_PACKAGE_SIZE=%q 不是正整数字节数", rawSize)
	}
	if !sha256HexRe.MatchString(sum) {
		return nil, fmt.Errorf("KUN_APP_ANDROID_PACKAGE_SHA256=%q 不是 64 位小写十六进制", sum)
	}
	return &AndroidPackage{URL: rawURL, Size: size, SHA256: sum}, nil
}

func parseReleaseVersion(key, v string) ([]int, error) {
	m := releaseVersionRe.FindStringSubmatch(v)
	if m == nil {
		return nil, fmt.Errorf("%s=%q 不是 MAJOR.MINOR.PATCH 形式的版本号", key, v)
	}
	out := make([]int, 3)
	for i, part := range m[1:] {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("%s=%q: %w", key, v, err)
		}
		out[i] = n
	}
	return out, nil
}

func splitCSV(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func requireEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("环境变量 %s 未设置", key)
	}
	return val, nil
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}

func envOrDefaultBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return fallback
}
