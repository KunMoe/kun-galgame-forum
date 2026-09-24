package apiv1

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

func TestMapUserPlaneForbidden(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  *catalogclient.UserAPIError
		own  bool
		want string
	}{
		{"not a user token", &catalogclient.UserAPIError{Status: http.StatusForbidden, ProblemCode: "USER_IDENTITY_REQUIRED"}, false, problem.CodeInternalError},
		{"not a user token on own", &catalogclient.UserAPIError{Status: http.StatusForbidden, ProblemCode: "USER_IDENTITY_REQUIRED"}, true, problem.CodeInternalError},
		{"someone else's", &catalogclient.UserAPIError{Status: http.StatusForbidden, ProblemCode: "PERMISSION_REQUIRED"}, false, problem.CodeNotFound},
		{"own refused", &catalogclient.UserAPIError{Status: http.StatusForbidden, ProblemCode: "PERMISSION_REQUIRED"}, true, problem.CodePermissionRequired},
		{"catalog 400", &catalogclient.UserAPIError{Status: http.StatusBadRequest, ProblemCode: "INVALID_PARAMETER"}, false, problem.CodeInternalError},
		{"site not bound", &catalogclient.UserAPIError{Status: http.StatusForbidden, ProblemCode: "SITE_NOT_BOUND"}, true, problem.CodeInternalError},
		{"claim not owned", &catalogclient.UserAPIError{Status: http.StatusForbidden, ProblemCode: "CLAIM_NOT_OWNED"}, true, problem.CodeNotFound},
		{"precondition", &catalogclient.UserAPIError{Status: http.StatusPreconditionFailed, ProblemCode: "PRECONDITION_FAILED"}, true, problem.CodePreconditionFailed},
		{"if-match missing", &catalogclient.UserAPIError{Status: http.StatusPreconditionRequired, ProblemCode: "PRECONDITION_REQUIRED"}, true, problem.CodeInternalError},
		{"already exists", &catalogclient.UserAPIError{Status: http.StatusConflict, ProblemCode: "ALREADY_EXISTS"}, true, problem.CodeAlreadyExists},
		{"invalid transition", &catalogclient.UserAPIError{Status: http.StatusConflict, ProblemCode: "INVALID_STATE_TRANSITION", Message: "cannot submit a claim in state \"live\""}, true, problem.CodeInvalidStateTransition},
		{"decision already made", &catalogclient.UserAPIError{Status: http.StatusConflict, ProblemCode: "DECISION_ALREADY_MADE"}, true, problem.CodeInvalidStateTransition},
		{"key reused", &catalogclient.UserAPIError{Status: http.StatusConflict, ProblemCode: "IDEMPOTENCY_KEY_REUSED"}, true, problem.CodeIdempotencyKeyReused},
		{"unknown conflict", &catalogclient.UserAPIError{Status: http.StatusConflict, ProblemCode: "SOMETHING_NEW"}, true, problem.CodeInternalError},
		{"stale cursor", &catalogclient.UserAPIError{Status: http.StatusBadRequest, ProblemCode: "INVALID_CURSOR"}, true, problem.CodeInvalidCursor},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p *problem.Problem
			if !errors.As(mapUserPlane(tc.err, tc.own), &p) || p.Code != tc.want {
				t.Fatalf("got %+v, want %s", p, tc.want)
			}
			if strings.Contains(strings.ToLower(p.Detail), "cannot submit") {
				t.Errorf("upstream English leaked into detail: %q", p.Detail)
			}
		})
	}
}

func TestMapUserPlaneDuplicateSuspects(t *testing.T) {
	err := &catalogclient.UserAPIError{
		Status:      http.StatusConflict,
		ProblemCode: problem.CodeDuplicateSuspects,
		Message:     "re-send with confirm_duplicates=true",
		Suspects:    []catalogclient.DuplicateSuspect{{ID: "9", DisplayName: "CLANNAD"}},
	}
	var p *problem.Problem
	if !errors.As(mapUserPlane(err, true), &p) || p.Code != problem.CodeDuplicateSuspects {
		t.Fatalf("got %+v", p)
	}
	if strings.Contains(p.Detail, "confirm_duplicates") {
		t.Errorf("upstream English leaked: %q", p.Detail)
	}
}

func TestMapUserPlaneAtFieldErrors(t *testing.T) {
	pointer := func(up string) (string, bool) {
		if up == "/field_values/catalog.work.titles" {
			return "/titles", true
		}
		return "", false
	}
	err := &catalogclient.UserAPIError{
		Status: http.StatusUnprocessableEntity, ProblemCode: "VALIDATION_FAILED",
		FieldErrors: []catalogclient.ProblemFieldError{
			{Pointer: "/field_values/catalog.work.titles", Reason: problem.ReasonTooLong},
			{Pointer: "/field_values/catalog.work.links", Reason: problem.ReasonRequired},
		},
	}
	var p *problem.Problem
	if !errors.As(mapUserPlaneAt(err, true, pointer), &p) || p.Code != problem.CodeValidationFailed {
		t.Fatalf("got %+v", p)
	}
	if len(p.Errors) != 1 || *p.Errors[0].Pointer != "/titles" || p.Errors[0].Reason != problem.ReasonNotAllowedValue {
		t.Fatalf("errors = %+v; want only /titles, TOO_LONG degraded for its missing params", p.Errors)
	}

	if !errors.As(mapUserPlane(err, true), &p) || len(p.Errors) != 1 || *p.Errors[0].Pointer != "" {
		t.Fatalf("without a translator every upstream pointer must stay unattributed: %+v", p)
	}
}
