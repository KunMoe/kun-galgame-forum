package apiv1

import (
	"context"
	"errors"
	"log/slog"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
)

type ctxKey int

const (
	accessTokenCtxKey ctxKey = iota
	sessionCookieCtxKey
)

const (
	oauthNameTaken       = 10007
	oauthTokenMissing    = 10001
	oauthTokenInvalid    = 10002
	oauthTokenExpired    = 10003
	oauthFieldConstraint = 7
	oauthMoeInsufficient = 16006
	oauthCreatorAlready  = 17001
	oauthCreatorPending  = 17002
	oauthCreatorCooldown = 17003
)

// Operation middleware runs after identity middleware, which is what puts the token into Fiber locals.
func withUpstream(ctx huma.Context, next func(huma.Context)) {
	fc := humafiber.Unwrap(ctx)
	token := middleware.GetAccessToken(fc)
	if token == "" {
		if err := problem.Write(fc, problem.New(
			problem.CodeInvalidCredential,
			"The credential is invalid, expired, or revoked.",
		)); err != nil {
			slog.Error("apiv1 write problem", "request_id", problem.RequestID(fc), "err", err)
		}
		return
	}
	ctx = huma.WithValue(ctx, accessTokenCtxKey, token)
	ctx = huma.WithValue(ctx, sessionCookieCtxKey, fc.Cookies(middleware.SessionCookieName))
	next(ctx)
}

func accessToken(ctx context.Context) string {
	s, _ := ctx.Value(accessTokenCtxKey).(string)
	return s
}

func sessionCookie(ctx context.Context) string {
	s, _ := ctx.Value(sessionCookieCtxKey).(string)
	return s
}

func unavailable(err error) *problem.Problem {
	return problem.Unavailable(err)
}

func oauthCode(err error) int {
	var oe *oauth.Error
	if errors.As(err, &oe) {
		return oe.Code
	}
	return 0
}

func userclientCode(err error) int {
	var ue *userclient.OAuthError
	if errors.As(err, &ue) {
		return ue.Code
	}
	return 0
}

func mapPreferencesError(err error) *problem.Problem {
	switch oauthCode(err) {
	case oauth.CodePreferencesConflict:
		return problem.New(problem.CodePreconditionFailed, "If-Match did not match the current representation.")
	case oauth.CodePreferencesScopeMissing:
		return problem.New(problem.CodeScopeRequired, "The credential is valid but lacks the scope this operation needs.")
	case oauth.CodePreferencesTooLarge:
		maxLen := 65536
		return validationFailed(problem.AtPointer("/doc", problem.ReasonTooLong,
			"the preference document exceeds 64 KiB after compaction",
			&problem.FieldParams{MaxLength: &maxLen}))
	case oauthFieldConstraint:
		return problem.New(problem.CodeInvalidParameter, "If-Match is not a document version.",
			problem.AtHeader("If-Match", problem.ReasonInvalidFormat, "If-Match must be a quoted document version", nil))
	}
	return unavailable(err)
}

func mapProfileError(err error) *problem.Problem {
	switch oauthCode(err) {
	case oauthNameTaken:
		return problem.New(problem.CodeUsernameTaken, "The requested name is already in use by another account.")
	case oauthMoeInsufficient:
		return problem.New(problem.CodeMoemoepointInsufficient, "The caller's moemoepoint balance is below what a name change costs.")
	case oauthFieldConstraint:
		return validationFailed(problem.AtPointer("/name", problem.ReasonInvalidFormat,
			"the name does not match the allowed character set", nil))
	case oauthTokenMissing, oauthTokenInvalid, oauthTokenExpired:
		return problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	return unavailable(err)
}

func mapCreatorApplyError(err error) *problem.Problem {
	switch userclientCode(err) {
	case oauthCreatorAlready:
		return problem.New(problem.CodeInvalidStateTransition,
			"The current state is creator; legal targets: none.")
	case oauthCreatorPending:
		return problem.New(problem.CodeAlreadyExists,
			"The same subject already has a live record for this target.")
	case oauthCreatorCooldown:
		return problem.New(problem.CodeCreatorApplicationCooldown,
			"A declined creator application is still inside its cooldown window.")
	}
	return unavailable(err)
}
