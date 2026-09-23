package apiv1

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
)

type accessTokenCtxKey struct{}

func withAccessToken(ctx huma.Context, next func(huma.Context)) {
	fc := humafiber.Unwrap(ctx)
	next(huma.WithValue(ctx, accessTokenCtxKey{}, middleware.GetAccessToken(fc)))
}

func accessToken(ctx context.Context) string {
	s, _ := ctx.Value(accessTokenCtxKey{}).(string)
	return s
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
