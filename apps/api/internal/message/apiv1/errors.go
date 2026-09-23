package apiv1

import (
	"errors"
	"strconv"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

var errUnconfigured = errors.New("apiv1 messages: service is not configured")

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func parsePositiveID(s string) (int, bool) {
	return repr.ParseID(repr.DecimalID(s))
}

func parseUpToID(id repr.DecimalID) (int, *problem.Problem) {
	n, err := strconv.Atoi(string(id))
	if err != nil || n < 0 {
		return 0, notFound()
	}
	return n, nil
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
}

func permissionRequired() *problem.Problem {
	return problem.New(problem.CodePermissionRequired, "The caller may only recall their own messages.")
}

func invalidCursor() *problem.Problem {
	return problem.New(
		problem.CodeInvalidCursor,
		"The cursor cannot be parsed or is no longer valid.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil),
	)
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.messages == nil || s.messages.DB() == nil || s.chats == nil || s.chats.DB() == nil || s.users == nil || s.convert == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}
