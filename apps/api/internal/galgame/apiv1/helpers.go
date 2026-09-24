package apiv1

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
)

type accessTokenCtxKey struct{}
type idempotencyKeyCtxKey struct{}

func withAccessToken(ctx huma.Context, next func(huma.Context)) {
	fc := humafiber.Unwrap(ctx)
	ctx = huma.WithValue(ctx, accessTokenCtxKey{}, middleware.GetAccessToken(fc))
	next(huma.WithValue(ctx, idempotencyKeyCtxKey{}, fc.Get("Idempotency-Key")))
}

func accessToken(ctx context.Context) string {
	s, _ := ctx.Value(accessTokenCtxKey{}).(string)
	return s
}

func idempotencyKey(ctx context.Context) string {
	s, _ := ctx.Value(idempotencyKeyCtxKey{}).(string)
	return s
}

// Every collection write speaks to the catalog as the signed-in person, so a
// session with no OAuth access token cannot make one. That is a re-login, not
// a server fault: the token is minted at sign-in and this site never had one
// to begin with for a session that predates the folder scopes.
func requireToken(ctx context.Context) (string, *problem.Problem) {
	tok := accessToken(ctx)
	if tok == "" {
		return "", problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	return tok, nil
}

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func mergedWork(currentID int64) *problem.Problem {
	p := problem.New(problem.CodeEntityMerged, fmt.Sprintf("This work was merged into %d.", currentID))
	p.SetExtension("object", "work")
	p.SetExtension("current_id", strconv.FormatInt(currentID, 10))
	return p
}

func selfLikeForbidden() *problem.Problem {
	return problem.New(problem.CodeSelfLikeForbidden, "Users cannot like what they wrote themselves.")
}

func invalidParameter(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeInvalidParameter, "A request parameter is invalid.", fields...)
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request body is invalid.", fields...)
}

func problemResponses(byStatus map[int]string) map[string]*huma.Response {
	out := make(map[string]*huma.Response, len(byStatus))
	for status, desc := range byStatus {
		out[strconv.Itoa(status)] = &huma.Response{
			Description: desc,
			Content: map[string]*huma.MediaType{
				problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
			},
		}
	}
	return out
}

func parseWorkID(raw string) (int, bool) {
	return repr.ParseID(repr.DecimalID(raw))
}

func parseWorkIDs(parts []repr.DecimalID) ([]int, *problem.Problem) {
	out := make([]int, 0, len(parts))
	for i, part := range parts {
		id, ok := repr.ParseID(repr.DecimalID(strings.TrimSpace(string(part))))
		if !ok {
			return nil, invalidParameter(problem.AtParameter("work_ids", problem.ReasonInvalidFormat,
				"every id must be a positive decimal integer; item "+strconv.Itoa(i)+" is not", nil))
		}
		out = append(out, id)
	}
	return out, nil
}

func renderable(users map[int]userclient.User, id int) bool {
	u, ok := users[id]
	return ok && userclient.IsRenderable(u)
}

func catalogUnavailable(appErr *legacyErrors.AppError) *problem.Problem {
	return problem.Unavailable(appErr)
}

func upstreamStatus(err error) string {
	var api *catalogclient.UserAPIError
	switch {
	case errors.As(err, &api):
		return strconv.Itoa(api.Status)
	case errors.Is(err, catalogclient.ErrUnauthorized):
		return "401"
	case errors.Is(err, catalogclient.ErrInsufficientScope):
		return "403"
	case errors.Is(err, catalogclient.ErrNotFound):
		return "404"
	case errors.Is(err, catalogclient.ErrUpstream):
		return "5xx"
	default:
		return "transport"
	}
}
