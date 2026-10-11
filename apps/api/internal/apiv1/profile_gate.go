package apiv1

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

const metaProfileGate = "apiv1.profileGate"

type ProfileGateLookup func(ctx context.Context, userID int) (bool, error)

var errNoProfileGate = errors.New("apiv1: profile gate lookup is not configured")

// ProfileGated refuses an anonymous caller with PROFILE_SIGN_IN_REQUIRED when
// param names a user who shows their profile to signed-in users only. param is
// read from the path when the path names it, otherwise from the query.
func ProfileGated(op huma.Operation, param string) huma.Operation {
	if TierFromOp(&op) != TierOptional {
		panic("apiv1: a profile gate needs an optional-tier operation: a public one never learns who is signed in, a required one admits no anonymous caller")
	}
	op.Metadata[metaProfileGate] = param
	refusal := "PROFILE_SIGN_IN_REQUIRED when the caller is anonymous and " + param +
		" names a user who shows their profile to signed-in users only."
	op.Description = strings.TrimSpace(op.Description + " " + refusal)
	if resp := op.Responses[strconv.Itoa(http.StatusForbidden)]; resp != nil {
		resp.Description = strings.TrimSpace(resp.Description + " " + refusal)
		return op
	}
	ensureProblemResponse(&op, http.StatusForbidden)
	op.Responses[strconv.Itoa(http.StatusForbidden)].Description =
		"ACCOUNT_BANNED when the caller's account is banned. " + refusal
	return op
}

func newProfileGateMiddleware(lookup ProfileGateLookup) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		op := ctx.Operation()
		param, _ := op.Metadata[metaProfileGate].(string)
		if param == "" || User(ctx.Context()) != nil {
			next(ctx)
			return
		}
		raw := ctx.Query(param)
		if strings.Contains(op.Path, "{"+param+"}") {
			raw = ctx.Param(param)
		}
		userID, err := strconv.Atoi(raw)
		if err != nil || userID <= 0 {
			next(ctx)
			return
		}
		if lookup == nil {
			writeProblem(ctx, problem.Internal(errNoProfileGate))
			return
		}
		required, err := lookup(ctx.Context(), userID)
		if err != nil {
			writeProblem(ctx, problem.Internal(err))
			return
		}
		if required {
			writeProblem(ctx, problem.New(problem.CodeProfileSignInRequired,
				"The user shows their profile to signed-in users only. Sign in to read it."))
			return
		}
		next(ctx)
	}
}
