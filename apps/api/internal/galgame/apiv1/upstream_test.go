package apiv1

import (
	"errors"
	"net/http"
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p *problem.Problem
			if !errors.As(mapUserPlane(tc.err, tc.own), &p) || p.Code != tc.want {
				t.Fatalf("got %+v, want %s", p, tc.want)
			}
		})
	}
}
