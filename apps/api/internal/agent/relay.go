package agent

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/gofiber/fiber/v3"
)

const (
	responseLimit = 8 << 20
	uploadLimit   = 16 << 20
)

var (
	errUnconfigured  = errors.New("agent: KUN_AGENT_API_BASE is empty")
	errResponseLarge = errors.New("agent: response larger than the relay accepts")
)

var forwardedRequestHeaders = []string{fiber.HeaderContentType, "Idempotency-Key"}

var forwardedResponseHeaders = []string{fiber.HeaderContentType, fiber.HeaderRetryAfter, fiber.HeaderLocation}

// Relay serves /api/v1/agent/* from the NextMoe agent under the caller's own
// token. The agent finds the site from the token's client_id, so the forum
// stores nothing of a conversation.
type Relay struct {
	base     string
	http     *http.Client
	stream   *http.Client
	resolver apiv1.IdentityResolver
}

func NewRelay(base string, resolver apiv1.IdentityResolver) *Relay {
	return &Relay{
		base: strings.TrimRight(base, "/"),
		http: &http.Client{Timeout: 30 * time.Second},
		// A run streams for minutes; only the wait for its first byte is bounded.
		stream: &http.Client{Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
			DisableCompression:    true,
		}},
		resolver: resolver,
	}
}

func (r *Relay) Mount(v1 fiber.Router) {
	v1.Get("/agent/runs/:id/stream", r.Stream)
	v1.Get("/agent/uploads/:id", r.Upload)
	v1.Add([]string{fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete},
		"/agent/*", r.Forward)
}

func (r *Relay) upstream(c fiber.Ctx, ctx context.Context, rest string) (*http.Request, *problem.Problem) {
	id, p := apiv1.RequireIdentity(c, r.resolver)
	if p != nil {
		return nil, p
	}
	if plain, err := url.PathUnescape(rest); err != nil || plain == "" ||
		strings.Contains(plain, "..") || strings.ContainsAny(plain, "?#\\") {
		return nil, problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
	}
	if r.base == "" {
		return nil, problem.Unavailable(errUnconfigured)
	}

	target := r.base + "/v2/agent/" + rest
	if q := c.Request().URI().QueryString(); len(q) > 0 {
		target += "?" + string(q)
	}
	req, err := http.NewRequestWithContext(ctx, c.Method(), target, bytes.NewReader(c.Body()))
	if err != nil {
		return nil, problem.Internal(err)
	}
	req.Header.Set(fiber.HeaderAuthorization, "Bearer "+id.AccessToken)
	req.Header.Set("CF-Connecting-IP", clientIP(c))
	req.Header.Set(problem.HeaderRequestID, problem.RequestID(c))
	for _, h := range forwardedRequestHeaders {
		if v := c.Get(h); v != "" {
			req.Header.Set(h, v)
		}
	}
	return req, nil
}

func (r *Relay) Forward(c fiber.Ctx) error {
	req, p := r.upstream(c, c.Context(), c.Params("*"))
	if p != nil {
		return problem.Write(c, p)
	}
	req.Header.Set(fiber.HeaderAccept, "application/json")
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
	copyResponseHeaders(c, resp)
	return c.Status(resp.StatusCode).Send(body)
}

// An attachment is the user's own bytes served from the site's origin, so it
// is sandboxed: an image type the agent's settings later allow (svg) must not
// run script here.
func (r *Relay) Upload(c fiber.Ctx) error {
	req, p := r.upstream(c, c.Context(), "uploads/"+url.PathEscape(c.Params("id")))
	if p != nil {
		return problem.Write(c, p)
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return problem.Write(c, problem.Unavailable(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, uploadLimit+1))
	if err != nil {
		return problem.Write(c, problem.Unavailable(err))
	}
	if len(body) > uploadLimit {
		return problem.Write(c, problem.Internal(errResponseLarge))
	}
	copyResponseHeaders(c, resp)
	if resp.StatusCode < http.StatusMultipleChoices {
		if v := resp.Header.Get(fiber.HeaderCacheControl); v != "" {
			c.Set(fiber.HeaderCacheControl, v)
		}
		c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
		c.Set(fiber.HeaderContentSecurityPolicy, "sandbox; default-src 'none'; img-src 'self'")
	}
	return c.Status(resp.StatusCode).Send(body)
}

// The upstream read hangs on its own context rather than the request's, and
// ends when the browser goes away (the write fails) or the run finishes; a
// dropped stream is picked back up through Last-Event-ID.
func (r *Relay) Stream(c fiber.Ctx) error {
	ctx, cancel := context.WithCancel(context.Background())
	req, p := r.upstream(c, ctx, "runs/"+url.PathEscape(c.Params("id"))+"/stream")
	if p != nil {
		cancel()
		return problem.Write(c, p)
	}
	req.Header.Set(fiber.HeaderAccept, "text/event-stream")
	if last := c.Get("Last-Event-ID"); last != "" {
		req.Header.Set("Last-Event-ID", last)
	}
	resp, err := r.stream.Do(req)
	if err != nil {
		cancel()
		return problem.Write(c, problem.Unavailable(err))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, responseLimit))
		resp.Body.Close()
		cancel()
		copyResponseHeaders(c, resp)
		return c.Status(resp.StatusCode).Send(body)
	}

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache, no-transform")
	c.Set("X-Accel-Buffering", "no")
	if v := resp.Header.Get("X-Vercel-Ai-Ui-Message-Stream"); v != "" {
		c.Set("X-Vercel-Ai-Ui-Message-Stream", v)
	}
	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer cancel()
		defer resp.Body.Close()
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := w.Write(buf[:n]); werr != nil {
					return
				}
				if werr := w.Flush(); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	})
}

func copyResponseHeaders(c fiber.Ctx, resp *http.Response) {
	for _, h := range forwardedResponseHeaders {
		if v := resp.Header.Get(h); v != "" {
			c.Set(h, v)
		}
	}
}

// The agent trusts CF-Connecting-IP only from a private peer, and the forum
// has no trusted-proxy setup, so the same rule applies here: a client that
// reaches the API directly cannot name its own address.
func clientIP(c fiber.Ctx) string {
	return visitorIP(c.IP(), c.Get("CF-Connecting-IP"))
}

func visitorIP(peer, forwarded string) string {
	if ip := net.ParseIP(peer); ip != nil && (ip.IsPrivate() || ip.IsLoopback()) {
		if forwarded = strings.TrimSpace(forwarded); net.ParseIP(forwarded) != nil {
			return forwarded
		}
	}
	return peer
}
