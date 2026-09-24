package apiv1

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func mapUserPlane(err error, own bool) error {
	return mapUserPlaneAt(err, own, nil)
}

type pointerOf func(upstream string) (v1 string, ok bool)

func mapUserPlaneAt(err error, own bool, pointer pointerOf) error {
	if err == nil {
		return nil
	}
	switch {
	// A grant too narrow for folder:read and an expired session are different
	// faults with different cures, and folding them together cost an outage on
	// 2026-09-08: this returned code 205, whose client-side handler re-checks
	// /api/user/status — which answers from this site's own cookie, finds it
	// perfectly healthy, and returns without a word. Every collection read
	// failed and nobody was told. Code 235 is what the five other scope-starved
	// paths here already use (playtime, cover votes, edits, submissions, image
	// upload) and its handler says the one thing that actually fixes it.
	case errors.Is(err, catalogclient.ErrInsufficientScope):
		return problem.New(problem.CodeScopeRequired, "The credential is valid but lacks the scope this operation needs.")
	case errors.Is(err, catalogclient.ErrUnauthorized):
		return problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	case errors.Is(err, catalogclient.ErrNotFound):
		return notFound()
	case errors.Is(err, catalogclient.ErrNotConfigured), errors.Is(err, catalogclient.ErrUpstream):
		return problem.Unavailable(err)
	}
	var api *catalogclient.UserAPIError
	if !errors.As(err, &api) {
		return problem.Unavailable(err)
	}
	switch api.Status {
	case http.StatusForbidden:
		// Catalog answers this when the credential is not a user token (dev's
		// /auth/login tokens carry no client id). Folding it into 404 made the
		// playtime list read as "nothing here" on 2026-09-24's browser pass.
		if api.ProblemCode == "USER_IDENTITY_REQUIRED" || api.ProblemCode == "SITE_NOT_BOUND" {
			return problem.Internal(err)
		}
		if api.ProblemCode == "CLAIM_NOT_OWNED" || api.ProblemCode == "TENANT_MISMATCH" {
			return notFound()
		}
		if own {
			return problem.New(problem.CodePermissionRequired, "The caller cannot perform this operation.")
		}
		return notFound()
	case http.StatusNotFound:
		return notFound()
	case http.StatusConflict:
		return mapCatalog409(api)
	case http.StatusPreconditionFailed:
		return problem.New(problem.CodePreconditionFailed, "If-Match did not match the current representation.")
	case http.StatusPreconditionRequired:
		return problem.Internal(err)
	case http.StatusUnprocessableEntity:
		return mapCatalog422(api, pointer)
	case http.StatusTooManyRequests:
		// 429 used to fall through to 500 + the caller's fallback sentence, so
		// user 90769's blown daily quota on 2026-09-20 rendered as
		// 「读取收藏夹列表失败」 — a data-corruption story for a rate limit.
		slog.Warn("catalog user plane: upstream 429 mapped to 503", "upstream_status", api.Status, "err", err)
		p := problem.Unavailable(err)
		if api.RetryAfter != "" {
			h := http.Header{}
			h.Set("Retry-After", api.RetryAfter)
			return huma.ErrorWithHeaders(p, h)
		}
		return p
	case http.StatusBadRequest:
		if api.ProblemCode == problem.CodeInvalidCursor {
			return problem.New(problem.CodeInvalidCursor, "The cursor cannot be parsed or is no longer valid.",
				problem.AtParameter("cursor", problem.ReasonInvalidFormat, "restart from the first page", nil))
		}
		return problem.Internal(err)
	default:
		if api.Status >= 500 {
			return problem.Unavailable(err)
		}
		slog.Warn("catalog user plane: unmapped upstream status", "upstream_status", api.Status, "err", err)
		return problem.Unavailable(err)
	}
}

func mapCatalog409(api *catalogclient.UserAPIError) error {
	switch api.ProblemCode {
	case problem.CodeDuplicateSuspects:
		// Catalog's detail tells the caller to resend with confirm_duplicates,
		// a field v1 clients do not have; on the old route that English was the
		// whole message and a same-title work could not be submitted at all.
		p := problem.New(problem.CodeDuplicateSuspects, "Live works share a submitted title; nothing was written.")
		suspects := make([]map[string]string, 0, len(api.Suspects))
		for _, s := range api.Suspects {
			suspects = append(suspects, map[string]string{"id": s.ID, "display_name": s.DisplayName})
		}
		p.SetExtension("suspects", suspects)
		return p
	case problem.CodeAlreadyExists:
		return problem.New(problem.CodeAlreadyExists, "The target already exists in catalog.")
	case problem.CodeInvalidStateTransition, "DECISION_ALREADY_MADE":
		return problem.New(problem.CodeInvalidStateTransition, "The current state does not allow this transition; re-read and retry.")
	case problem.CodeIdempotencyRequestInProgress:
		return problem.New(problem.CodeIdempotencyRequestInProgress, "A request with the same Idempotency-Key is still being processed.")
	case problem.CodeIdempotencyKeyReused:
		return problem.New(problem.CodeIdempotencyKeyReused, "The same Idempotency-Key was sent with a different request.")
	}
	return problem.Internal(fmt.Errorf("catalog user plane: unmapped upstream 409 %q: %w", api.ProblemCode, api))
}

func mapCatalog422(api *catalogclient.UserAPIError, pointer pointerOf) error {
	msg := strings.ToLower(api.Message + " " + api.ProblemCode)
	switch {
	case strings.Contains(msg, "default folder cannot be deleted"):
		return problem.New(problem.CodeInvalidStateTransition, "The default collection cannot be deleted.")
	case strings.Contains(msg, "may keep at most"):
		max := float64(catalogclient.FoldersPerUserMax)
		return problem.New(problem.CodeValidationFailed, "The account already holds as many collections as catalog allows.",
			problem.AtPointer("", problem.ReasonOutOfRange, "folder count exceeds the catalog cap",
				&problem.FieldParams{Maximum: &max}))
	case strings.Contains(msg, "may hold at most"):
		max := float64(catalogclient.FolderItemsMax)
		return problem.New(problem.CodeValidationFailed, "The collection is full.",
			problem.AtParameter("work_id", problem.ReasonOutOfRange, "folder item count exceeds the catalog cap",
				&problem.FieldParams{Maximum: &max}))
	case strings.Contains(msg, "is_default can only be set"):
		return validationFailed(problem.AtPointer("/is_default", problem.ReasonNotAllowedValue, "false is not accepted", nil))
	case strings.Contains(msg, "folder name is required") || strings.Contains(msg, "name is required"):
		return validationFailed(titleTooShort())
	case strings.Contains(msg, "visibility must be"):
		allowed := []string{"private", "public"}
		return problem.New(problem.CodeValidationFailed, "visibility is not in the closed vocabulary.",
			problem.AtPointer("/visibility", problem.ReasonUnknownValue, "visibility must be private or public",
				&problem.FieldParams{Allowed: &allowed}))
	default:
		if fields := attributedFields(api.FieldErrors, pointer); len(fields) > 0 {
			return validationFailed(fields...)
		}
		slog.Warn("catalog user plane: unrecognised upstream 422 passed on as VALIDATION_FAILED",
			"upstream_status", api.Status, "upstream_code", api.ProblemCode, "upstream_message", api.Message)
		return validationFailed(problem.AtPointer("", problem.ReasonNotAllowedValue, "catalog refused the write", nil))
	}
}

// Catalog sends TOO_LONG / UNKNOWN_VALUE without the params the zh-CN template
// fills, so a parameterised reason is passed on as NOT_ALLOWED_VALUE.
func attributedFields(errs []catalogclient.ProblemFieldError, pointer pointerOf) []problem.FieldError {
	if pointer == nil {
		return nil
	}
	out := make([]problem.FieldError, 0, len(errs))
	for _, e := range errs {
		at, ok := pointer(e.Pointer)
		if !ok {
			slog.Warn("catalog user plane: upstream field error without a v1 pointer",
				"upstream_pointer", e.Pointer, "upstream_reason", e.Reason)
			continue
		}
		reason := problem.ReasonNotAllowedValue
		if def, known := problem.LookupReason(e.Reason); known && len(def.Params) == 0 {
			reason = e.Reason
		}
		out = append(out, problem.AtPointer(at, reason, "catalog refused this field", nil))
	}
	return out
}
