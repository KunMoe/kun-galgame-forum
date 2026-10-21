package catalogclient

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	BaseURL    string
	AppKey     string
	HTTPClient *http.Client
}

type Client struct {
	appKey     string
	baseURL    string
	httpClient *http.Client
}

func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		appKey:     strings.TrimSpace(cfg.AppKey),
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: hc,
	}
}

// Configured is the user plane, which authenticates with the caller's own
// access token and so needs nothing but a host.
func (c *Client) Configured() bool { return c.baseURL != "" }

func (c *Client) AppConfigured() bool { return c.baseURL != "" && c.appKey != "" }

var (
	ErrNotConfigured = errors.New("catalogclient: not configured (empty base URL or application key)")
	ErrUnauthorized  = errors.New("catalogclient: unauthorized (check the application key)")
	ErrNotFound      = errors.New("catalogclient: entity not found")
	ErrUpstream      = errors.New("catalogclient: catalog service error")
)
