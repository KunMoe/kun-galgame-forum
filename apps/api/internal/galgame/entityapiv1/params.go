package entityapiv1

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

var errUnconfigured = errors.New("galgame entity faces are not configured")

// searchDepth is how many catalog matches a q= search reaches: relevance order
// is useless past the first page or two, and the hidden and adult gates run on
// this window, so total counts exactly what can be paged.
const searchDepth = 100

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func merged(object string, currentID int64) *problem.Problem {
	p := problem.New(problem.CodeEntityMerged, fmt.Sprintf("This %s was merged into %d.", object, currentID))
	p.SetExtension("object", object)
	p.SetExtension("current_id", strconv.FormatInt(currentID, 10))
	return p
}

func unavailable(appErr *legacyErrors.AppError) *problem.Problem {
	return problem.Unavailable(appErr)
}

func pathID(raw string) (int, bool) {
	return repr.ParseID(repr.DecimalID(raw))
}

// checkDepth applies the page-number depth limit, which a q= search narrows to
// searchDepth.
func checkDepth(page, limit int, searching bool) *problem.Problem {
	if !searching {
		return collect.PageNumber{Page: page, Limit: limit}.CheckDepth()
	}
	if page*limit <= searchDepth {
		return nil
	}
	last := float64(searchDepth / limit)
	return problem.New(problem.CodeInvalidParameter, "A q= search reaches its first 100 matches only.",
		problem.AtParameter("page", problem.ReasonOutOfRange, "page × limit may not exceed 100 when q is set",
			&problem.FieldParams{Maximum: &last}))
}

func pageOf[T any](rows []T, page, limit int) []T {
	start := (page - 1) * limit
	if start >= len(rows) {
		return []T{}
	}
	return rows[start:min(start+limit, len(rows))]
}

func pageList[T any](rows []T, page, limit int) repr.PageList[T] {
	total, relation := collect.ClampTotal(len(rows))
	return repr.NewPageList(pageOf(rows, page, limit), total, relation)
}

func trimQuery(q string) string {
	return strings.TrimSpace(q)
}

func parseEntityIDs(parts []repr.DecimalID) ([]int, *problem.Problem) {
	out := make([]int, 0, len(parts))
	for i, part := range parts {
		id, ok := repr.ParseID(repr.DecimalID(strings.TrimSpace(string(part))))
		if !ok {
			return nil, problem.New(problem.CodeInvalidParameter, "ids holds something that is not a positive decimal integer.",
				problem.AtParameter("ids", problem.ReasonInvalidFormat,
					"every id must be a positive decimal integer; item "+strconv.Itoa(i)+" is not", nil))
		}
		out = append(out, id)
	}
	return out, nil
}

func uniqueIDs(ids []int) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// nameMatches is the substring search the in-memory indexes answer: every name
// the reader might type, case-folded.
func nameMatches(q string, name repr.CatalogName, extra ...string) bool {
	q = strings.ToLower(q)
	if strings.Contains(strings.ToLower(name.DisplayName), q) {
		return true
	}
	if name.Latin != nil && strings.Contains(strings.ToLower(*name.Latin), q) {
		return true
	}
	for _, n := range name.Localized {
		if strings.Contains(strings.ToLower(n.Value), q) {
			return true
		}
	}
	for _, e := range extra {
		if strings.Contains(strings.ToLower(e), q) {
			return true
		}
	}
	return false
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

func errUnknownVocab(field, value string) error {
	return fmt.Errorf("catalog sent %s %q, which is not in the declared vocabulary", field, value)
}
