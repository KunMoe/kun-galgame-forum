package oauth

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"kun-galgame-api/pkg/config"
)

const oauthHTTPTimeout = 10 * time.Second

const (
	CodeAccountBanned       = 10014
	CodeRefreshTokenExpired = 10003
	CodeInvalidToken        = 10002
	CodeInvalidGrant        = 15005
	CodeInvalidClientSecret = 15008

	CodePreferencesScopeMissing = 18001
	CodePreferencesConflict     = 18006
	CodeAdultConfirmationNeeded = 18008
)

type Error struct {
	Code       int
	HTTPStatus int
	Message    string
}

func (e *Error) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("oauth: code=%d http=%d msg=%q", e.Code, e.HTTPStatus, e.Message)
	}
	return fmt.Sprintf("oauth: http=%d msg=%q", e.HTTPStatus, e.Message)
}

func IsBanned(err error) bool {
	var oe *Error
	if !stderrors.As(err, &oe) {
		return false
	}
	return oe.Code == CodeAccountBanned || oe.HTTPStatus == http.StatusForbidden
}

func IsRefreshTokenDead(err error) bool {
	var oe *Error
	if !stderrors.As(err, &oe) {
		return false
	}
	switch oe.Code {
	case CodeRefreshTokenExpired, CodeInvalidToken, CodeInvalidGrant, CodeInvalidClientSecret:
		return true
	}
	return false
}

func IsTransient(err error) bool {
	var oe *Error
	if !stderrors.As(err, &oe) {
		return true
	}
	if oe.HTTPStatus == 0 || oe.HTTPStatus >= 500 {
		return true
	}
	if oe.Code == 0 {
		return true
	}
	return false
}

type Client struct {
	cfg        config.OAuthConfig
	httpClient *http.Client
}

func NewClient(cfg config.OAuthConfig) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: oauthHTTPTimeout},
	}
}

func decodeProtocol(resp *http.Response) (json.RawMessage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &Error{HTTPStatus: resp.StatusCode, Message: "读取响应体失败: " + err.Error()}
	}

	if resp.StatusCode == http.StatusOK {
		return json.RawMessage(body), nil
	}

	var p struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if jerr := json.Unmarshal(body, &p); jerr != nil {
		return nil, &Error{
			HTTPStatus: resp.StatusCode,
			Message:    fmt.Sprintf("解析响应失败: %v, body=%s", jerr, truncateBody(body)),
		}
	}
	msg := p.ErrorDescription
	if msg == "" {
		msg = p.Error
	}
	return nil, &Error{Code: oauthErrToCode(p.Error), HTTPStatus: resp.StatusCode, Message: msg}
}

func decodeHouse(resp *http.Response) (json.RawMessage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &Error{HTTPStatus: resp.StatusCode, Message: "读取响应体失败: " + err.Error()}
	}

	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if jerr := json.Unmarshal(body, &env); jerr != nil {
		return nil, &Error{
			HTTPStatus: resp.StatusCode,
			Message:    fmt.Sprintf("解析响应失败: %v, body=%s", jerr, truncateBody(body)),
		}
	}
	if resp.StatusCode == http.StatusOK && env.Code == 0 && len(env.Data) > 0 {
		return env.Data, nil
	}
	return nil, &Error{Code: env.Code, HTTPStatus: resp.StatusCode, Message: env.Message}
}

func oauthErrToCode(errStr string) int {
	switch errStr {
	case "invalid_token":
		return CodeInvalidToken
	case "invalid_grant", "unauthorized_client":
		return CodeInvalidGrant
	case "invalid_client":
		return CodeInvalidClientSecret
	default:
		return 0
	}
}

func truncateBody(b []byte) string {
	const max = 256
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "...(truncated)"
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// AdultConfirmed and NSFWDisplay ride the profile scope this client has always
// held, so reading them needed no new grant. Writing them does: PutAuthMeNSFW
// and the preferences pair are refused 18001 until the client's allowed_scopes
// gains `preferences` and the user re-authorizes.
type UserInfo struct {
	ID             int      `json:"id"`
	Sub            string   `json:"sub"`
	Name           string   `json:"name"`
	Email          string   `json:"email"`
	Picture        string   `json:"picture"`
	Roles          []string `json:"roles"`
	SiteRoles      []string `json:"site_roles"`
	UpdatedAt      int64    `json:"updated_at"`
	AdultConfirmed bool     `json:"adult_confirmed"`
	NSFWDisplay    string   `json:"nsfw_display"`
}

func (c *Client) ExchangeCode(code, codeVerifier string) (*TokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  c.cfg.RedirectURI,
		"client_id":     c.cfg.ClientID,
		"client_secret": c.cfg.ClientSecret,
		"code_verifier": codeVerifier,
	}
	data, err := c.postProtocol("/oauth/token", payload)
	if err != nil {
		return nil, err
	}
	var tok TokenResponse
	if jerr := json.Unmarshal(data, &tok); jerr != nil {
		return nil, &Error{Message: "解析 token 响应失败: " + jerr.Error()}
	}
	if tok.AccessToken == "" {
		return nil, &Error{Message: "token 响应缺 access_token"}
	}
	return &tok, nil
}

func (c *Client) FetchUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", c.cfg.ServerURL+"/oauth/userinfo", nil)
	if err != nil {
		return nil, &Error{Message: "创建 userinfo 请求失败: " + err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Message: "请求 userinfo 失败: " + err.Error()}
	}
	defer resp.Body.Close()

	data, derr := decodeProtocol(resp)
	if derr != nil {
		return nil, derr
	}
	var info UserInfo
	if jerr := json.Unmarshal(data, &info); jerr != nil {
		return nil, &Error{Message: "解析 userinfo 响应失败: " + jerr.Error()}
	}
	return &info, nil
}

func (c *Client) RevokeToken(refreshToken string) error {
	payload, err := json.Marshal(map[string]string{"token": refreshToken})
	if err != nil {
		return fmt.Errorf("序列化 revoke 请求失败: %w", err)
	}
	req, err := http.NewRequest("POST", c.cfg.ServerURL+"/oauth/revoke", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建 revoke 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) RefreshOAuthToken(refreshToken string) (*TokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     c.cfg.ClientID,
		"client_secret": c.cfg.ClientSecret,
	}
	data, err := c.postProtocol("/oauth/token", payload)
	if err != nil {
		return nil, err
	}
	var tok TokenResponse
	if jerr := json.Unmarshal(data, &tok); jerr != nil {
		return nil, &Error{Message: "解析刷新响应失败: " + jerr.Error()}
	}
	if tok.AccessToken == "" {
		return nil, &Error{Message: "刷新响应缺 access_token"}
	}
	return &tok, nil
}

func (c *Client) PatchAuthMe(accessToken string, body any) (json.RawMessage, error) {
	return c.doHouse(accessToken, "PATCH", "/auth/me", body, nil)
}

type NSFWState struct {
	NSFWDisplay      string  `json:"nsfw_display"`
	AdultConfirmedAt *string `json:"adult_confirmed_at"`
}

func (c *Client) PutAuthMeNSFW(accessToken, display string) (*NSFWState, error) {
	data, err := c.doHouse(
		accessToken, "PUT", "/auth/me/nsfw",
		map[string]string{"nsfw_display": display}, nil,
	)
	if err != nil {
		return nil, err
	}
	var state NSFWState
	if jerr := json.Unmarshal(data, &state); jerr != nil {
		return nil, &Error{Message: "解析 PUT /auth/me/nsfw 响应失败: " + jerr.Error()}
	}
	return &state, nil
}

// An OAuth token reaches exactly two namespaces upstream — its own client_id
// and the shared `global` — so the forum's document is always keyed by the
// configured client id. Taking the namespace from the request instead would
// hand the browser a proxy into `global`, which other sites also write.
func (c *Client) preferencesPath() string {
	return "/auth/me/preferences/" + url.PathEscape(c.cfg.ClientID)
}

func (c *Client) GetPreferences(accessToken string) (json.RawMessage, error) {
	return c.doHouse(accessToken, "GET", c.preferencesPath(), nil, nil)
}

func (c *Client) PutPreferences(
	accessToken string,
	doc json.RawMessage,
	ifMatch string,
) (json.RawMessage, error) {
	var headers map[string]string
	if ifMatch != "" {
		headers = map[string]string{"If-Match": ifMatch}
	}
	return c.doHouse(
		accessToken, "PUT", c.preferencesPath(),
		map[string]any{"doc": doc}, headers,
	)
}

func (c *Client) doHouse(
	accessToken, method, path string,
	body any,
	headers map[string]string,
) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		payload, jerr := json.Marshal(body)
		if jerr != nil {
			return nil, &Error{Message: "序列化 " + method + " " + path + " 请求失败: " + jerr.Error()}
		}
		reader = bytes.NewReader(payload)
	}
	req, rerr := http.NewRequest(method, c.cfg.ServerURL+path, reader)
	if rerr != nil {
		return nil, &Error{Message: "创建 " + method + " " + path + " 请求失败: " + rerr.Error()}
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		return nil, &Error{Message: "请求 " + method + " " + path + " 失败: " + derr.Error()}
	}
	defer resp.Body.Close()
	return decodeHouse(resp)
}

func (c *Client) UploadAvatar(accessToken string, body []byte, contentType string) (json.RawMessage, error) {
	req, rerr := http.NewRequest("POST", c.cfg.ServerURL+"/auth/me/avatar", bytes.NewReader(body))
	if rerr != nil {
		return nil, &Error{Message: "创建 POST /auth/me/avatar 请求失败: " + rerr.Error()}
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		return nil, &Error{Message: "请求 POST /auth/me/avatar 失败: " + derr.Error()}
	}
	defer resp.Body.Close()
	return decodeHouse(resp)
}

func (c *Client) postProtocol(path string, payload any) (json.RawMessage, error) {
	body, jerr := json.Marshal(payload)
	if jerr != nil {
		return nil, &Error{Message: "序列化请求失败: " + jerr.Error()}
	}
	req, rerr := http.NewRequest("POST", c.cfg.ServerURL+path, bytes.NewReader(body))
	if rerr != nil {
		return nil, &Error{Message: "创建请求失败: " + rerr.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		return nil, &Error{Message: "请求 OAuth 失败: " + derr.Error()}
	}
	defer resp.Body.Close()
	return decodeProtocol(resp)
}
