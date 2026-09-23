package apiv1

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

func TestRequiredStatuses(t *testing.T) {
	query := &huma.Param{Name: "q", In: "query"}
	path := &huma.Param{Name: "topic_id", In: "path"}
	idem := &huma.Param{Name: idempotencyHeader, In: "header"}
	body := &huma.RequestBody{}
	for _, c := range []struct {
		name string
		path string
		op   huma.Operation
		want []int
	}{
		{"public, no input", "/x", Public(huma.Operation{}), []int{500}},
		{"a query parameter", "/x", Public(huma.Operation{Parameters: []*huma.Param{query}}), []int{400, 500}},
		{"a path parameter", "/x/{topic_id}", Public(huma.Operation{Parameters: []*huma.Param{path}}), []int{400, 404, 500}},
		{"optional tier", "/x", Optional(huma.Operation{}), []int{401, 403, 500, 503}},
		{"required tier", "/x", Required(huma.Operation{}), []int{401, 403, 500, 503}},
		{"a body", "/x", Public(huma.Operation{RequestBody: body}), []int{400, 413, 415, 422, 500}},
		{"an idempotency key", "/x", Public(huma.Operation{Parameters: []*huma.Param{idem}}), []int{409, 500}},
	} {
		got := RequiredStatuses(c.path, &c.op)
		slices.Sort(got)
		got = slices.Compact(got)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, got, c.want)
		}
	}
}

type statusQuery struct {
	Q string `query:"q" maxLength:"8" doc:"Q."`
}

type statusBody struct {
	Body struct {
		Name string `json:"name" maxLength:"8" doc:"Name."`
	}
}

func TestDeclaredStatusesAreExactlyTheDerivedOnes(t *testing.T) {
	_, api := newTestAPI(t, Deps{}, func(api huma.API) {
		huma.Register(api, Public(huma.Operation{OperationID: "plain", Method: http.MethodGet, Path: "/plain", Summary: "P"}),
			func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })
		huma.Register(api, Public(huma.Operation{OperationID: "query", Method: http.MethodGet, Path: "/query", Summary: "Q"}),
			func(context.Context, *statusQuery) (*struct{}, error) { return nil, nil })
		huma.Register(api, Optional(huma.Operation{OperationID: "opt", Method: http.MethodGet, Path: "/opt", Summary: "O"}),
			func(context.Context, *statusQuery) (*struct{}, error) { return nil, nil })
		huma.Register(api, IdempotencyRequired(Required(huma.Operation{OperationID: "write", Method: http.MethodPost, Path: "/write", Summary: "W"})),
			func(context.Context, *statusBody) (*struct{}, error) { return nil, nil })
	})
	for path, want := range map[string][]string{
		"/plain": {"204", "500"},
		"/query": {"204", "400", "500"},
		"/opt":   {"204", "400", "401", "403", "500", "503"},
		"/write": {"204", "400", "401", "403", "409", "413", "415", "422", "500", "503"},
	} {
		item := api.OpenAPI().Paths[path]
		op := item.Get
		if op == nil {
			op = item.Post
		}
		got := slices.Sorted(maps.Keys(op.Responses))
		if !slices.Equal(got, want) {
			t.Errorf("%s declares %v, want %v", path, got, want)
		}
	}
}
