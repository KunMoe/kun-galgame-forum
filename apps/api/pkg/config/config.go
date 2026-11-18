package config

import (
	"fmt"
	"strconv"
	"strings"

	"kun-galgame-api/pkg/imageclient"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	OAuth          OAuthConfig
	CORS           CORSConfig
	NextMoeAPI     NextMoeAPIConfig
	NewsAPI        NewsAPIConfig
	MoyuAPI        MoyuAPIConfig
	LetmoeAPI      LetmoeAPIConfig
	StickerAPI     StickerAPIConfig
	ImageClient    ImageClientConfig
	ArtifactClient ArtifactClientConfig
	LinkChecker    LinkCheckerConfig
	Trust          TrustConfig
	Catalog        CatalogClientConfig
	Community      CommunityConfig
	Chat           ChatConfig
	Dlsite         DlsiteConfig
	Lottery        LotteryConfig
	Bearer         BearerConfig
	Agent          AgentConfig
	AI             AIConfig
}

// AIConfig points at an OpenAI-compatible chat completions endpoint. Either
// value empty turns topic classification and summaries off.
type AIConfig struct {
	BaseURL string
	Token   string
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

// APIBase reaches the NextMoe agent, which /api/v1/agent/* relays to under the
// caller's own token. MCPAddr is a listener of its own so that no public route
// reaches it: the Dokploy domain points at SERVER_PORT.
type AgentConfig struct {
	APIBase string
	MCPAddr string
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

// BaseURL reaches NextMoe chat, which /api/v1/chat/* relays to under the
// caller's own token. Enabled is the cutover: the forum's own direct messages
// stop taking writes and stop counting as unread.
type ChatConfig struct {
	BaseURL string
	Enabled bool
}

type CatalogClientConfig struct {
	BaseURL string
}

type TrustConfig struct {
	BaseURL        string
	CallbackSecret string
	Site           string
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

// Image, artifact and community authenticate the forum by its own OAuth
// client, so their ClientID/ClientSecret are always the OAuth pair.
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

// LetmoeAPIConfig reaches the /v2/letmoe face on the same gateway as moyu, so
// its base must not default to KUN_NEXTMOE_API_BASE either. The metadata routes
// admit any valid key; /file, which answers the download links, needs the
// operator-granted scope letmoe:files on the key and answers SCOPE_REQUIRED
// without it.
type LetmoeAPIConfig struct {
	BaseURL string
	APIKey  string
}

// StickerAPIConfig reaches the /v2/sticker face, which sits on the same gateway
// under the same rule as MoyuAPIConfig: any valid key, no scope.
type StickerAPIConfig struct {
	BaseURL string
	APIKey  string
}

// StoreAPIKey reaches infra's /v2/store face, which mints the per-site short
// link that carries click attribution. It is a THIRD developer key, separate
// from the catalog and news ones: the face is gated on the scope store:read and
// the v2 limiter buckets per key, so minting a few thousand links must not spend
// the catalogue's minute budget. The links belong to the OAuth client rather
// than the key, so a separate key does not split attribution.
type DlsiteConfig struct {
	StoreAPIBase string
	StoreAPIKey  string
}

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
}

type CORSConfig struct {
	AllowOrigins string
}

func Load() (*Config, error) {
	if err := loadStore(); err != nil {
		return nil, err
	}
	dbURL := lookupEnvOnly(DatabaseURLEnv)
	if dbURL == "" {
		return nil, fmt.Errorf("环境变量 %s 未设置", DatabaseURLEnv)
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
			ImageCDNBase: envOrDefault("KUN_IMAGE_PUBLIC_BASE_URL", imageclient.DefaultPublicBase),
		},
		NewsAPI: NewsAPIConfig{
			BaseURL: envOrDefault("KUN_NEWS_API_BASE", nextMoeBase),
			APIKey:  envOrDefault("KUN_NEWS_API_KEY", ""),
		},
		MoyuAPI: MoyuAPIConfig{
			BaseURL: envOrDefault("KUN_MOYU_API_BASE", "https://api.nextmoe.dev"),
			APIKey:  envOrDefault("KUN_MOYU_API_KEY", nextMoeKey),
		},
		LetmoeAPI: LetmoeAPIConfig{
			BaseURL: envOrDefault("KUN_LETMOE_API_BASE", "https://api.nextmoe.dev"),
			APIKey:  envOrDefault("KUN_LETMOE_API_KEY", nextMoeKey),
		},
		StickerAPI: StickerAPIConfig{
			BaseURL: envOrDefault("KUN_STICKER_API_BASE", "https://api.nextmoe.dev"),
			APIKey:  envOrDefault("KUN_STICKER_API_KEY", nextMoeKey),
		},
		ImageClient: ImageClientConfig{
			BaseURL:      envOrDefault("KUN_IMAGE_CLIENT_BASE_URL", "http://127.0.0.1:9278"),
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
		},
		ArtifactClient: ArtifactClientConfig{
			BaseURL:      envOrDefault("KUN_ARTIFACT_CLIENT_BASE_URL", "http://127.0.0.1:9279"),
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
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
		},
		Catalog: CatalogClientConfig{
			BaseURL: envOrDefault("KUN_CATALOG_API_BASE", "http://127.0.0.1:9281"),
		},
		AI: AIConfig{
			BaseURL: envOrDefault("KUN_AI_BASE_URL", ""),
			Token:   envOrDefault("KUN_AI_TOKEN", ""),
		},
		Lottery: LotteryConfig{
			CodeKey: envOrDefault("KUN_LOTTERY_CODE_KEY", ""),
		},
		Community: CommunityConfig{
			BaseURL:      envOrDefault("KUN_COMMUNITY_API_BASE", ""),
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
		},
		Chat: ChatConfig{
			BaseURL: envOrDefault("KUN_CHAT_API_BASE", "http://127.0.0.1:9285"),
			Enabled: envOrDefaultBool("KUN_CHAT_ENABLED", false),
		},
		Dlsite: DlsiteConfig{
			StoreAPIBase: envOrDefault("KUN_STORE_API_BASE", nextMoeBase),
			StoreAPIKey:  envOrDefault("KUN_STORE_API_KEY", ""),
		},
		Bearer: BearerConfig{
			Issuer:    envOrDefault("KUN_OIDC_ISSUER", oauthOrigin),
			JWKSURL:   envOrDefault("KUN_OIDC_JWKS_URL", oauthOrigin+"/oauth/jwks"),
			ClientIDs: splitCSV(lookup("KUN_BEARER_CLIENT_IDS", "")),
		},
		Agent: AgentConfig{
			APIBase: envOrDefault("KUN_AGENT_API_BASE", ""),
			MCPAddr: envOrDefault("KUN_AGENT_MCP_ADDR", "127.0.0.1:2335"),
		},
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
	val := lookup(key, "")
	if val == "" {
		return "", fmt.Errorf("%s 未设置 (环境变量与配置中心都没有)", key)
	}
	return val, nil
}

func envOrDefault(key, fallback string) string {
	return lookup(key, fallback)
}

func envOrDefaultInt(key string, fallback int) int {
	if n, err := strconv.Atoi(lookup(key, strconv.Itoa(fallback))); err == nil {
		return n
	}
	return fallback
}

func envOrDefaultBool(key string, fallback bool) bool {
	if b, err := strconv.ParseBool(lookup(key, strconv.FormatBool(fallback))); err == nil {
		return b
	}
	return fallback
}
