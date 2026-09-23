package apiv1

import (
	"context"
	"net/http"
	"sort"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

type problemType struct {
	Object      string `json:"object" enum:"problem_type" maxLength:"12" doc:"Type discriminant. Always problem_type."`
	Code        string `json:"code" pattern:"^[A-Z][A-Z0-9_]*[A-Z0-9]$" minLength:"2" maxLength:"63" doc:"Top-level error code. UPPER_SNAKE."`
	Domain      string `json:"domain" enum:"platform,kungal,me,moderation" maxLength:"16" doc:"Type URI domain segment."`
	Status      int    `json:"status" minimum:"400" maximum:"599" doc:"HTTP status this code is bound to. One status per code."`
	Type        string `json:"type" format:"uri" maxLength:"256" doc:"Problem type URI. The last path segment is the kebab-case form of code."`
	Title       string `json:"title" maxLength:"128" pattern:"^[ -~]+$" doc:"Stable English phrase for this type. Does not vary per request."`
	Description string `json:"description" maxLength:"512" doc:"English prose. Free text; never use it as a decision input."`
}

type problemReason struct {
	Object      string   `json:"object" enum:"problem_reason" maxLength:"14" doc:"Type discriminant. Always problem_reason."`
	Reason      string   `json:"reason" pattern:"^[A-Z][A-Z0-9_]*[A-Z0-9]$" minLength:"2" maxLength:"63" doc:"Field-level reason. UPPER_SNAKE. Disjoint from top-level codes."`
	Title       string   `json:"title" maxLength:"128" pattern:"^[ -~]+$" doc:"Stable English phrase for this reason."`
	Description string   `json:"description" maxLength:"512" doc:"English prose. Free text; never use it as a decision input."`
	ParamNames  []string `json:"param_names" doc:"Keys this reason can carry in a field error's params. Empty array when it carries none." maxItems:"8" maxLength:"32" pattern:"^[a-z][a-z0-9_]*$"`
}

type listProblemTypesOutput struct {
	Body repr.List[problemType]
}

type listProblemReasonsOutput struct {
	Body repr.List[problemReason]
}

func registerMeta(api huma.API) {
	tags := []string{"meta"}
	huma.Register(api, Public(huma.Operation{
		OperationID: "listProblemReasons",
		Method:      http.MethodGet,
		Path:        "/problems/reasons",
		Summary:     "List every field-level error reason",
		Description: "The closed registry of field-level reasons. Unauthenticated. Values in this list never appear as top-level codes.",
		Tags:        tags,
	}), listProblemReasons)
	huma.Register(api, Public(huma.Operation{
		OperationID: "listProblemTypes",
		Method:      http.MethodGet,
		Path:        "/problems",
		Summary:     "List every top-level error code",
		Description: "The closed registry of top-level error codes. Unauthenticated. Sorted by domain, then code.",
		Tags:        tags,
	}), listProblemTypes)
}

func listProblemTypes(context.Context, *struct{}) (*listProblemTypesOutput, error) {
	return &listProblemTypesOutput{Body: repr.NewList(ProblemTypes(), nil)}, nil
}

func listProblemReasons(context.Context, *struct{}) (*listProblemReasonsOutput, error) {
	return &listProblemReasonsOutput{Body: repr.NewList(ProblemReasons(), nil)}, nil
}

func ProblemTypes() []problemType {
	items := make([]problemType, 0, len(problem.Codes))
	for _, d := range problem.Codes {
		items = append(items, problemType{
			Object:      "problem_type",
			Code:        d.Code,
			Domain:      string(d.Domain),
			Status:      d.Status,
			Type:        d.TypeURI(),
			Title:       d.Title,
			Description: d.Description,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		ri, rj := domainRank(items[i].Domain), domainRank(items[j].Domain)
		if ri != rj {
			return ri < rj
		}
		return items[i].Code < items[j].Code
	})
	return items
}

func ProblemReasons() []problemReason {
	items := make([]problemReason, 0, len(problem.Reasons))
	for _, d := range problem.Reasons {
		names := d.Params
		if names == nil {
			names = []string{}
		}
		items = append(items, problemReason{
			Object:      "problem_reason",
			Reason:      d.Reason,
			Title:       d.Title,
			Description: d.Description,
			ParamNames:  names,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Reason < items[j].Reason
	})
	return items
}

func domainRank(domain string) int {
	for i, d := range problem.DomainOrder {
		if string(d) == domain {
			return i
		}
	}
	return len(problem.DomainOrder)
}
