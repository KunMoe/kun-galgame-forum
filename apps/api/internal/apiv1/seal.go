package apiv1

import (
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

const (
	schemaRefPrefix = "#/components/schemas/"
	ProblemRef      = schemaRefPrefix + "Problem"
)

func sealDocument(doc *huma.OpenAPI) {
	for path, item := range doc.Paths {
		for _, op := range operations(item) {
			for _, status := range RequiredStatuses(path, op) {
				ensureProblemResponse(op, status)
			}
		}
	}
	reg := doc.Components.Schemas
	for name, s := range reg.Map() {
		if t := reg.TypeFromRef(schemaRefPrefix + name); t != nil {
			nullablePointers(t, s)
		}
	}
	walkDocSchemas(doc, markClosedEnums)
	walkDocSchemas(doc, nullableEnums)
	walkDocSchemas(doc, omitTrueAdditionalProperties)
}

// openapi-typescript renders additionalProperties: true as [key: string]: unknown,
// so in W0b-1 reading a misspelled field of a response type-checked as unknown.
func omitTrueAdditionalProperties(s *huma.Schema) {
	if ap, ok := s.AdditionalProperties.(bool); ok && ap {
		s.AdditionalProperties = nil
	}
}

func RequiredStatuses(path string, op *huma.Operation) []int {
	need := []int{http.StatusInternalServerError}
	params, pathParam := false, strings.Contains(path, "{")
	idempotent := false
	for _, p := range op.Parameters {
		switch {
		case p.In == "path":
			params, pathParam = true, true
		case p.In == "query":
			params = true
		case p.In == "header" && p.Name == idempotencyHeader:
			idempotent = true
		}
	}
	if params {
		need = append(need, http.StatusBadRequest)
	}
	if len(op.Security) > 0 {
		need = append(need, http.StatusUnauthorized, http.StatusForbidden, http.StatusServiceUnavailable)
	}
	if pathParam {
		need = append(need, http.StatusNotFound)
	}
	if op.RequestBody != nil {
		need = append(need, http.StatusBadRequest, http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType, http.StatusUnprocessableEntity)
	}
	if idempotent {
		need = append(need, http.StatusConflict)
	}
	return need
}

func ensureProblemResponse(op *huma.Operation, status int) {
	if op.Responses == nil {
		op.Responses = map[string]*huma.Response{}
	}
	key := strconv.Itoa(status)
	if op.Responses[key] != nil {
		return
	}
	op.Responses[key] = &huma.Response{
		Description: http.StatusText(status),
		Content: map[string]*huma.MediaType{
			problem.ContentType: {Schema: &huma.Schema{Ref: ProblemRef}},
		},
	}
}

// huma renders a pointer-to-struct field as a bare $ref and a pointer to a
// SchemaProvider type as non-null: UserRef.avatar and a *DateTime were typed
// non-null while the code sends null for them.
func nullablePointers(t reflect.Type, s *huma.Schema) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := range t.NumField() {
		f := t.Field(i)
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if f.Anonymous && name == "" {
			nullablePointers(f.Type, s)
			continue
		}
		p := s.Properties[name]
		if p == nil || f.Type.Kind() != reflect.Pointer || omitsNil(opts) {
			continue
		}
		if p.Ref == "" {
			p.Nullable = true
			continue
		}
		s.Properties[name] = &huma.Schema{
			Description: p.Description,
			AnyOf:       []*huma.Schema{{Ref: p.Ref}, {Type: "null"}},
		}
	}
}

func omitsNil(opts string) bool {
	for opt := range strings.SplitSeq(opts, ",") {
		if opt == "omitempty" || opt == "omitzero" {
			return true
		}
	}
	return false
}

func markClosedEnums(s *huma.Schema) {
	if len(s.Enum) == 0 {
		return
	}
	if s.Extensions == nil {
		s.Extensions = map[string]any{}
	}
	if _, ok := s.Extensions["x-vocabulary-closed"]; !ok {
		s.Extensions["x-vocabulary-closed"] = true
	}
}

// huma renders a nullable enum as type [string, null] with an enum that has no
// null, and its own validator short-circuits null before the enum: every
// Image.sexual of null failed the W0a-5 contract test's JSON Schema validator.
func nullableEnums(s *huma.Schema) {
	if s.Nullable && len(s.Enum) > 0 && !slices.Contains(s.Enum, nil) {
		s.Enum = append(s.Enum, nil)
	}
}

func walkDocSchemas(doc *huma.OpenAPI, fn func(*huma.Schema)) {
	seen := map[*huma.Schema]bool{}
	visit := func(s *huma.Schema) { walkSchema(s, seen, fn) }
	for _, s := range doc.Components.Schemas.Map() {
		visit(s)
	}
	for _, item := range doc.Paths {
		for _, op := range operations(item) {
			for _, p := range op.Parameters {
				visit(p.Schema)
			}
			if op.RequestBody != nil {
				for _, c := range op.RequestBody.Content {
					visit(c.Schema)
				}
			}
			for _, resp := range op.Responses {
				for _, c := range resp.Content {
					visit(c.Schema)
				}
			}
		}
	}
}

func walkSchema(s *huma.Schema, seen map[*huma.Schema]bool, fn func(*huma.Schema)) {
	if s == nil || seen[s] {
		return
	}
	seen[s] = true
	fn(s)
	for _, p := range s.Properties {
		walkSchema(p, seen, fn)
	}
	walkSchema(s.Items, seen, fn)
	for _, x := range s.OneOf {
		walkSchema(x, seen, fn)
	}
	for _, x := range s.AnyOf {
		walkSchema(x, seen, fn)
	}
	for _, x := range s.AllOf {
		walkSchema(x, seen, fn)
	}
	walkSchema(s.Not, seen, fn)
}

func operations(item *huma.PathItem) []*huma.Operation {
	var ops []*huma.Operation
	for _, op := range []*huma.Operation{
		item.Get, item.Put, item.Post, item.Delete, item.Options, item.Head, item.Patch, item.Trace,
	} {
		if op != nil {
			ops = append(ops, op)
		}
	}
	return ops
}
