package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v3"
)

type Tier string

const (
	TierPublic   Tier = "public"
	TierOptional Tier = "optional"
	TierRequired Tier = "required"
)

const metaTier = "apiv1.tier"

type IdentityResolver interface {
	ResolveIdentity(c fiber.Ctx) middleware.Identity
}

type ctxKey int

const (
	userCtxKey ctxKey = iota
)

func Public(op huma.Operation) huma.Operation {
	declareTier(&op, TierPublic)
	op.Security = nil
	return op
}

func Optional(op huma.Operation) huma.Operation {
	declareTier(&op, TierOptional)
	op.Security = []map[string][]string{
		{"session": {}},
		{"bearer": {}},
		{},
	}
	return op
}

func Required(op huma.Operation) huma.Operation {
	declareTier(&op, TierRequired)
	op.Security = []map[string][]string{
		{"session": {}},
		{"bearer": {}},
	}
	return op
}

func User(ctx context.Context) *middleware.UserInfo {
	u, _ := ctx.Value(userCtxKey).(*middleware.UserInfo)
	return u
}

func TierOf(api huma.API, method, fiberPath string) Tier {
	if api == nil {
		return ""
	}
	return TierFromOp(lookupOp(api, method, fiberPath))
}

func TierFromOp(op *huma.Operation) Tier {
	if op == nil || op.Metadata == nil {
		return TierPublic
	}
	t, _ := op.Metadata[metaTier].(Tier)
	if t == "" {
		return TierPublic
	}
	return t
}

func declareTier(op *huma.Operation, tier Tier) {
	if op.Metadata == nil {
		op.Metadata = map[string]any{}
	}
	op.Metadata[metaTier] = tier
	// Not through op.Errors: once it is non-empty huma adds 422 to every operation
	// with parameters, and when it is empty huma adds a `default` response. GET
	// /topics declared a 422 it can never return.
	ensureProblemResponse(op, http.StatusInternalServerError)
}

func lookupOp(api huma.API, method, fiberPath string) *huma.Operation {
	if api == nil {
		return nil
	}
	item := api.OpenAPI().Paths[specPath(fiberPath)]
	if item == nil {
		return nil
	}
	switch method {
	case http.MethodGet:
		return item.Get
	case http.MethodPost:
		return item.Post
	case http.MethodPut:
		return item.Put
	case http.MethodPatch:
		return item.Patch
	case http.MethodDelete:
		return item.Delete
	case http.MethodHead:
		return item.Head
	case http.MethodOptions:
		return item.Options
	case http.MethodTrace:
		return item.Trace
	default:
		return nil
	}
}

func specPath(fiberPath string) string {
	p := strings.TrimPrefix(fiberPath, Prefix)
	if p == "" {
		return "/"
	}
	parts := strings.Split(p, "/")
	for i, s := range parts {
		if strings.HasPrefix(s, ":") {
			parts[i] = "{" + s[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func newIdentityMiddleware(resolver IdentityResolver) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		fc := humafiber.Unwrap(ctx)

		op := ctx.Operation()
		tier := TierFromOp(op)
		if tier == TierPublic {
			next(ctx)
			return
		}
		if resolver == nil {
			writeProblem(ctx, problem.Internal(errNoResolver))
			return
		}

		id, p := resolveTier(fc, resolver, tier)
		if p != nil {
			writeProblem(ctx, p)
			return
		}
		if id.OK() {
			middleware.AttachIdentity(fc, id)
			ctx = huma.WithValue(ctx, userCtxKey, id.User)
		}
		next(ctx)
	}
}

// RequireIdentity applies the required tier to a Mount route, which huma's
// middleware never sees.
func RequireIdentity(c fiber.Ctx, resolver IdentityResolver) (middleware.Identity, *problem.Problem) {
	if resolver == nil {
		return middleware.Identity{}, problem.Internal(errNoResolver)
	}
	return resolveTier(c, resolver, TierRequired)
}

func resolveTier(fc fiber.Ctx, resolver IdentityResolver, tier Tier) (middleware.Identity, *problem.Problem) {
	id := resolver.ResolveIdentity(fc)
	if id.Err != nil {
		slog.Log(fc.Context(), identityLogLevel(id.Outcome), "apiv1 identity",
			"request_id", problem.RequestID(fc), "outcome", id.Outcome.String(), "err", id.Err)
	}
	if code := mapOutcome(tier, id.Outcome); code != "" {
		return id, identityProblem(code, id.Err)
	}
	return id, nil
}

var errNoResolver = errors.New("apiv1: identity resolver is not configured")

// Every visitor with an expired cookie produces one of the client outcomes on a
// public page; at ERROR they buried the store and key failures in production logs.
func identityLogLevel(outcome middleware.IdentityOutcome) slog.Level {
	switch outcome {
	case middleware.IdentityAnonymous, middleware.IdentitySessionMissing,
		middleware.IdentitySessionRefreshDead, middleware.IdentityBearerInvalid,
		middleware.IdentityBanned:
		return slog.LevelDebug
	default:
		return slog.LevelError
	}
}

func mapOutcome(tier Tier, outcome middleware.IdentityOutcome) string {
	switch outcome {
	case middleware.IdentityAnonymous:
		if tier == TierOptional {
			return ""
		}
		return problem.CodeMissingCredential
	case middleware.IdentitySessionMissing:
		if tier == TierOptional {
			return ""
		}
		return problem.CodeInvalidCredential
	case middleware.IdentitySessionStoreError:
		return problem.CodeServiceUnavailable
	case middleware.IdentitySessionRefreshDead:
		if tier == TierOptional {
			return ""
		}
		return problem.CodeInvalidCredential
	case middleware.IdentitySessionRefreshTransient:
		if tier == TierOptional {
			return ""
		}
		return problem.CodeServiceUnavailable
	case middleware.IdentityBanned:
		return problem.CodeAccountBanned
	case middleware.IdentitySessionInternalError:
		return problem.CodeInternalError
	case middleware.IdentitySessionOK, middleware.IdentityBearerOK:
		return ""
	case middleware.IdentityBearerInvalid:
		return problem.CodeInvalidCredential
	case middleware.IdentityBearerKeysUnavailable:
		return problem.CodeServiceUnavailable
	case middleware.IdentityBearerProvisioningFailed:
		return problem.CodeInternalError
	default:
		return problem.CodeInternalError
	}
}

func identityProblem(code string, cause error) *problem.Problem {
	switch code {
	case problem.CodeInternalError:
		return problem.Internal(cause)
	case problem.CodeMissingCredential:
		return problem.New(code, "The request has no credentials.")
	case problem.CodeInvalidCredential:
		return problem.New(code, "The credential is invalid, expired, or revoked.")
	case problem.CodeAccountBanned:
		return problem.New(code, "This account is banned.")
	case problem.CodeServiceUnavailable:
		return problem.New(code, "A dependency is unavailable. Retry the request.")
	default:
		return problem.Internal(cause)
	}
}
