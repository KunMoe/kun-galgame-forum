package apiv1

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"

	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

const (
	Prefix          = "/api/v1"
	wwwAuthenticate = `Bearer realm="kungal"`
	retryAfter503   = "5"
)

const (
	infoTitle       = "KUN Galgame Forum API"
	infoVersion     = "1.0.0-preview"
	infoDescription = "The first-party API of the KUN Galgame forum for its website and app. It is in preview.\n\n" +
		"Clients must ignore unknown fields in responses.\n" +
		"Clients must tolerate unseen values of open vocabularies; an unknown content node type renders its children or plain text.\n" +
		"Clients must handle an unknown error `code` with a fallback keyed on `status`."
	productionServer = "https://www.kungal.com/api/v1"
	SiteOrigin       = "https://www.kungal.com"
)

type Deps struct {
	Resolver IdentityResolver
	Redis    *redis.Client
}

var installOnce sync.Once

func IsV1Path(path string) bool {
	return path == Prefix || strings.HasPrefix(path, Prefix+"/")
}

func Setup(app *fiber.App, deps Deps, registrars ...func(huma.API)) huma.API {
	installHumaGlobals()

	app.Use(Prefix, v1Headers)

	cfg := huma.DefaultConfig(infoTitle, infoVersion)
	cfg.OpenAPIPath = ""
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	cfg.CreateHooks = nil
	cfg.AllowAdditionalPropertiesByDefault = true
	cfg.Info.Description = infoDescription
	if cfg.Info.Extensions == nil {
		cfg.Info.Extensions = map[string]any{}
	}
	cfg.Info.Extensions["x-stability"] = "preview"
	cfg.Servers = []*huma.Server{{URL: productionServer}}
	cfg.Transformers = append(cfg.Transformers, stampProblemTransformer)

	group := app.Group(Prefix)
	api := humafiber.NewWithGroup(app, group, cfg)
	api.UseMiddleware(newIdentityMiddleware(deps.Resolver))
	api.UseMiddleware(newIdempotencyMiddleware(deps.Redis))
	api.UseMiddleware(strictBooleans)

	declareSecurity(api.OpenAPI())
	registerMeta(api)
	for _, register := range registrars {
		register(api)
	}
	sealDocument(api.OpenAPI())
	mirrorHead(app)
	// Fiber matches in registration order and the legacy /api group spreads
	// Auth over every path below it: without this, an unmatched
	// /api/v1 request fell through into it and answered a legacy envelope.
	app.Use(Prefix, unmatched(api.OpenAPI()))
	return api
}

// Fiber adds its automatic HEAD routes at startup, which lands them behind the
// unmatched handler: every v1 HEAD answered 405.
func mirrorHead(app *fiber.App) {
	for _, r := range app.GetRoutes(true) {
		if r.Method != fiber.MethodGet || !IsV1Path(r.Path) || len(r.Handlers) == 0 {
			continue
		}
		rest := make([]any, 0, len(r.Handlers)-1)
		for _, h := range r.Handlers[1:] {
			rest = append(rest, h)
		}
		app.Add([]string{fiber.MethodHead}, r.Path, r.Handlers[0], rest...)
	}
}

func v1Headers(c fiber.Ctx) error {
	c.Set(problem.HeaderRequestID, problem.RequestID(c))
	c.Set("Cache-Control", "no-store")
	err := c.Next()
	switch h := &c.Response().Header; c.Response().StatusCode() {
	case http.StatusUnauthorized:
		if len(h.Peek(fiber.HeaderWWWAuthenticate)) == 0 {
			c.Set(fiber.HeaderWWWAuthenticate, wwwAuthenticate)
		}
	case http.StatusServiceUnavailable:
		if len(h.Peek(fiber.HeaderRetryAfter)) == 0 {
			c.Set(fiber.HeaderRetryAfter, retryAfter503)
		}
	}
	return err
}

func installHumaGlobals() {
	installOnce.Do(func() {
		huma.DefaultArrayNullable = false
		prevNew := huma.NewError
		huma.NewErrorWithContext = func(ctx huma.Context, status int, msg string, errs ...error) huma.StatusError {
			if ctx != nil && IsV1Path(ctx.URL().Path) {
				p := problem.FromHuma(ctx, status, msg, errs...)
				stampProblem(ctx, p)
				return p
			}
			return prevNew(status, msg, errs...)
		}
		huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
			return problem.FromHuma(nil, status, msg, errs...)
		}
	})
}

// A handler's problem is written by huma, not problem.Write: until W0a-5 the
// cause of every v1 handler's 500 was logged nowhere.
func stampProblemTransformer(ctx huma.Context, status string, v any) (any, error) {
	switch p := v.(type) {
	case *problem.Problem:
		stampProblem(ctx, p)
		problem.LogCause(p)
		return p, nil
	case problem.Problem:
		stampProblem(ctx, &p)
		problem.LogCause(&p)
		return &p, nil
	default:
		return v, nil
	}
}

func stampProblem(ctx huma.Context, p *problem.Problem) {
	if ctx == nil || p == nil {
		return
	}
	fc := humafiber.Unwrap(ctx)
	p.RequestID = problem.RequestID(fc)
	p.Instance = problem.Instance(fc)
}

func declareSecurity(doc *huma.OpenAPI) {
	if doc.Components == nil {
		doc.Components = &huma.Components{}
	}
	if doc.Components.SecuritySchemes == nil {
		doc.Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	doc.Components.SecuritySchemes["session"] = &huma.SecurityScheme{
		Type:        "apiKey",
		In:          "cookie",
		Name:        "kungal_session",
		Description: "Opaque BFF session cookie issued to the website. The cookie is ambient; a stale value on an optional operation is treated as anonymous.",
	}
	doc.Components.SecuritySchemes["bearer"] = &huma.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "OAuth access token presented by the first-party app. A Bearer is presented on purpose; an invalid token is always 401 INVALID_CREDENTIAL.",
	}
}

func WriteFiberError(c fiber.Ctx, err error) error {
	var p *problem.Problem
	if errors.As(err, &p) {
		return problem.Write(c, p)
	}
	var fe *fiber.Error
	if errors.As(err, &fe) && fe.Code == fiber.StatusRequestEntityTooLarge {
		return problem.Write(c, problem.New(problem.CodePayloadTooLarge, "The request body is larger than the server accepts."))
	}
	return problem.Write(c, problem.Internal(err))
}

func unmatched(doc *huma.OpenAPI) fiber.Handler {
	return func(c fiber.Ctx) error {
		if allow := allowedMethods(doc, strings.TrimPrefix(c.Path(), Prefix)); len(allow) > 0 {
			c.Set(fiber.HeaderAllow, strings.Join(allow, ", "))
			return problem.Write(c, problem.New(problem.CodeMethodNotAllowed, "The path exists but this method does not."))
		}
		return problem.Write(c, problem.New(problem.CodeNotFound, "Nothing visible exists at this URL."))
	}
}

func allowedMethods(doc *huma.OpenAPI, path string) []string {
	var allow []string
	for tmpl, item := range doc.Paths {
		if !matchesTemplate(tmpl, path) {
			continue
		}
		for method, op := range map[string]*huma.Operation{
			http.MethodGet: item.Get, http.MethodPost: item.Post, http.MethodPut: item.Put,
			http.MethodPatch: item.Patch, http.MethodDelete: item.Delete,
		} {
			if op != nil {
				allow = append(allow, method)
			}
		}
		if item.Get != nil {
			allow = append(allow, http.MethodHead)
		}
	}
	slices.Sort(allow)
	return slices.Compact(allow)
}

func matchesTemplate(tmpl, path string) bool {
	want, got := strings.Split(tmpl, "/"), strings.Split(path, "/")
	if len(want) != len(got) {
		return false
	}
	for i, seg := range want {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			if got[i] == "" {
				return false
			}
		} else if seg != got[i] {
			return false
		}
	}
	return true
}

func writeProblem(ctx huma.Context, p *problem.Problem) {
	fc := humafiber.Unwrap(ctx)
	if err := problem.Write(fc, p); err != nil {
		slog.Error("apiv1 write problem", "request_id", problem.RequestID(fc), "err", err)
	}
}
