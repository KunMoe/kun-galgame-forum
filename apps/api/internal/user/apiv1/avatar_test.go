package apiv1

import (
	"errors"
	"testing"

	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/problem"
)

func TestMapAvatarError(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		code   string
		status int
	}{
		{"unsupported format", &oauth.Error{HTTPStatus: 400, Code: 80009, Message: "不支持该图片格式"}, problem.CodeUnsupportedMediaType, 415},
		{"undecodable", &oauth.Error{HTTPStatus: 400, Code: 80010}, problem.CodeValidationFailed, 422},
		{"moderation", &oauth.Error{HTTPStatus: 400, Code: 60002}, problem.CodeImageRejected, 422},
		{"too large", &oauth.Error{HTTPStatus: 413, Code: 80007}, problem.CodePayloadTooLarge, 413},
		{"daily quota", &oauth.Error{HTTPStatus: 429, Code: 80008}, problem.CodeQuotaExceeded, 429},
		{"expired token", &oauth.Error{HTTPStatus: 401, Code: 10003}, problem.CodeInvalidCredential, 401},
		{"server fault", &oauth.Error{HTTPStatus: 500, Code: 1}, problem.CodeServiceUnavailable, 503},
		{"network", &oauth.Error{Message: "dial tcp: connection refused"}, problem.CodeServiceUnavailable, 503},
		{"not an oauth error", errors.New("boom"), problem.CodeServiceUnavailable, 503},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := mapAvatarError(c.err)
			if p.Code != c.code || p.Status != c.status {
				t.Fatalf("got %s %d, want %s %d", p.Code, p.Status, c.code, c.status)
			}
		})
	}
	p := mapAvatarError(&oauth.Error{HTTPStatus: 400, Code: 80010})
	if len(p.Errors) != 1 || p.Errors[0].Pointer == nil || *p.Errors[0].Pointer != "/file" || p.Errors[0].Reason != problem.ReasonInvalidFormat {
		t.Fatalf("undecodable errors %+v", p.Errors)
	}
}
