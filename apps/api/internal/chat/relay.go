package chat

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/gofiber/fiber/v3"
)

const responseLimit = 8 << 20

var (
	errUnconfigured  = errors.New("chat: KUN_CHAT_API_BASE is empty")
	errResponseLarge = errors.New("chat: response larger than the relay accepts")
)

var forwardedRequestHeaders = []string{fiber.HeaderContentType, "Idempotency-Key"}

var forwardedResponseHeaders = []string{fiber.HeaderContentType, fiber.HeaderRetryAfter, fiber.HeaderLocation}

type Relay struct {
	base     string
	http     *http.Client
	resolver apiv1.IdentityResolver
}

func NewRelay(base string, resolver apiv1.IdentityResolver) *Relay {
	return &Relay{
		base:     strings.TrimRight(base, "/"),
		http:     &http.Client{Timeout: 30 * time.Second},
		resolver: resolver,
	}
}

func (r *Relay) Mount(v1 fiber.Router) {
	v1.Add([]string{fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete},
		"/chat/*", r.Forward)
}

func (r *Relay) Forward(c fiber.Ctx) error {
	id, p := apiv1.RequireIdentity(c, r.resolver)
	if p != nil {
		return problem.Write(c, p)
	}
	rest := c.Params("*")
	if plain, err := url.PathUnescape(rest); err != nil || plain == "" ||
		strings.Contains(plain, "..") || strings.ContainsAny(plain, "?#\\") {
		return problem.Write(c, problem.New(problem.CodeNotFound, "Nothing visible exists at this URL."))
	}
	if r.base == "" {
		return problem.Write(c, problem.Unavailable(errUnconfigured))
	}

	target := r.base + "/v2/chat/" + rest
	if q := c.Request().URI().QueryString(); len(q) > 0 {
		target += "?" + string(q)
	}
	req, err := http.NewRequestWithContext(c.Context(), c.Method(), target, bytes.NewReader(c.Body()))
	if err != nil {
		return problem.Write(c, problem.Internal(err))
	}
	req.Header.Set(fiber.HeaderAuthorization, "Bearer "+id.AccessToken)
	req.Header.Set(fiber.HeaderAccept, "application/json")
	req.Header.Set(problem.HeaderRequestID, problem.RequestID(c))
	for _, h := range forwardedRequestHeaders {
		if v := c.Get(h); v != "" {
			req.Header.Set(h, v)
		}
	}

	resp, err := r.http.Do(req)
	if err != nil {
		return problem.Write(c, problem.Unavailable(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		return problem.Write(c, problem.Unavailable(err))
	}
	if len(body) > responseLimit {
		return problem.Write(c, problem.Internal(errResponseLarge))
	}

	c.Set(problem.HeaderRequestID, problem.RequestID(c))
	c.Set(fiber.HeaderCacheControl, "no-store")
	for _, h := range forwardedResponseHeaders {
		if v := resp.Header.Get(h); v != "" {
			c.Set(h, v)
		}
	}
	return c.Status(resp.StatusCode).Send(body)
}
