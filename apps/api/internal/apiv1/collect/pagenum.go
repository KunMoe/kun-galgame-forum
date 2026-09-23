package collect

import "kun-galgame-api/pkg/problem"

// MaxDepth bounds page × limit on every page-number collection (01 §4): an
// unbounded offset is what makes deep pages slow and lets rows shift under a
// reader.
const MaxDepth = 10000

type PageNumber struct {
	Page  int `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit int `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Page size. 1–100, default 20. Values above 100 are rejected, not clamped."`
}

func (p PageNumber) Offset() int {
	return (p.Page - 1) * p.Limit
}

func (p PageNumber) CheckDepth() *problem.Problem {
	if p.Page*p.Limit <= MaxDepth {
		return nil
	}
	last := float64(MaxDepth / p.Limit)
	return problem.New(problem.CodeInvalidParameter, "The page is deeper than a page-number collection goes.",
		problem.AtParameter("page", problem.ReasonOutOfRange, "page × limit may not exceed 10000",
			&problem.FieldParams{Maximum: &last}))
}

// ClampTotal caps a count at MaxDepth, since nothing past it can be paged to,
// and says whether the count is exact.
func ClampTotal(n int) (int, string) {
	if n > MaxDepth {
		return MaxDepth, "gte"
	}
	return n, "eq"
}
