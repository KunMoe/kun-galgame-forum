package apiv1

import (
	"errors"
	"log/slog"
	"net/http"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request body is invalid.", fields...)
}

func mapSubmissionUpstream(err error) error {
	if err == nil {
		return nil
	}
	switch {
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
		slog.Warn("news submission: unmapped upstream error", "err", err)
		return problem.Internal(err)
	}
	switch api.Status {
	case http.StatusNotFound:
		return notFound()
	case http.StatusConflict:
		return mapSubmission409(api)
	case http.StatusUnprocessableEntity:
		return mapSubmission422(api)
	case http.StatusTooManyRequests:
		return mapSubmission429(api)
	default:
		if api.Status >= 500 {
			return problem.Unavailable(err)
		}
		slog.Warn("news submission: unmapped upstream status", "upstream_status", api.Status, "upstream_code", api.ProblemCode)
		return problem.Internal(err)
	}
}

func mapSubmission409(api *catalogclient.UserAPIError) error {
	switch api.ProblemCode {
	case problem.CodeInvalidStateTransition:
		return problem.New(problem.CodeInvalidStateTransition, "The current state does not allow this transition; re-read and retry.")
	case problem.CodeIdempotencyRequestInProgress:
		return problem.New(problem.CodeIdempotencyRequestInProgress, "A request with the same Idempotency-Key is still being processed.")
	case problem.CodeIdempotencyKeyReused:
		return problem.New(problem.CodeIdempotencyKeyReused, "The same Idempotency-Key was sent with a different request.")
	default:
		slog.Warn("news submission: unmapped upstream status", "upstream_status", api.Status, "upstream_code", api.ProblemCode)
		return problem.Internal(api)
	}
}

func mapSubmission422(api *catalogclient.UserAPIError) error {
	fields := make([]problem.FieldError, 0, len(api.FieldErrors))
	for _, e := range api.FieldErrors {
		at, ok := submissionPointer(e.Pointer)
		if !ok {
			slog.Warn("news submission: upstream field error without a v1 pointer", "upstream_pointer", e.Pointer, "upstream_reason", e.Reason)
			continue
		}
		reason := problem.ReasonNotAllowedValue
		if def, known := problem.LookupReason(e.Reason); known && len(def.Params) == 0 {
			reason = e.Reason
		}
		fields = append(fields, problem.AtPointer(at, reason, "catalog refused this field", nil))
	}
	if len(fields) == 0 {
		return validationFailed(problem.AtPointer("", problem.ReasonNotAllowedValue, "catalog refused the write", nil))
	}
	return validationFailed(fields...)
}

func submissionPointer(upstream string) (string, bool) {
	switch upstream {
	case "/title":
		return "/title", true
	case "/summary":
		return "/preview", true
	case "/body":
		return "/content_markdown", true
	case "/source_url":
		return "/source_url", true
	default:
		return "", false
	}
}

func mapSubmission429(api *catalogclient.UserAPIError) error {
	code := problem.CodeRateLimited
	detail := "A rate limit was exceeded."
	if api.ProblemCode == problem.CodeQuotaExceeded {
		code = problem.CodeQuotaExceeded
		detail = "The caller's daily submission quota is exhausted."
	}
	p := problem.New(code, detail)
	if api.RetryAfter == "" {
		return p
	}
	h := http.Header{}
	h.Set("Retry-After", api.RetryAfter)
	return huma.ErrorWithHeaders(p, h)
}
