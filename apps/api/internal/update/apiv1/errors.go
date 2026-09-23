package apiv1

import (
	"fmt"
	"strings"

	"kun-galgame-api/pkg/problem"
)

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
}

func permissionRequired(detail string) *problem.Problem {
	return problem.New(problem.CodePermissionRequired, detail)
}

func contentRejected() *problem.Problem {
	return problem.New(problem.CodeContentRejected, "The trust-and-safety check refused the submitted text. Nothing was written.")
}

func invalidTransition(from int, to string) *problem.Problem {
	current, _ := stateToken(from)
	allowed := allowedTargets(from)
	targets := "none; it is final"
	if len(allowed) > 0 {
		targets = strings.Join(allowed, ", ")
	}
	return problem.New(problem.CodeInvalidStateTransition,
		fmt.Sprintf("The todo is %s and cannot move to %s. From %s it can move to: %s.", current, to, current, targets))
}

func notEditable(from int) *problem.Problem {
	current, _ := stateToken(from)
	return problem.New(problem.CodeInvalidStateTransition,
		fmt.Sprintf("The todo is %s; only a pending or in_progress todo can be edited.", current))
}

func invalidCursor() *problem.Problem {
	return problem.New(
		problem.CodeInvalidCursor,
		"The cursor cannot be parsed or is no longer valid.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil),
	)
}
