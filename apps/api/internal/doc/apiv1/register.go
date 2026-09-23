package apiv1

import (
	"net/http"
	"reflect"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

// optionalBool is a query flag whose absence means "do not filter"; huma
// panics on pointer parameters, so presence is tracked here instead.
type optionalBool struct {
	Value bool
	IsSet bool
}

func (o *optionalBool) Receiver() reflect.Value { return reflect.ValueOf(o).Elem().Field(0) }

func (o *optionalBool) OnParamSet(isSet bool, _ any) { o.IsSet = isSet }

func (optionalBool) Schema(huma.Registry) *huma.Schema {
	return &huma.Schema{Type: huma.TypeBoolean}
}

func (o optionalBool) ptr() *bool {
	if !o.IsSet {
		return nil
	}
	v := o.Value
	return &v
}

type DocSortToken string

func (DocSortToken) Schema(huma.Registry) *huma.Schema {
	enum := make([]any, len(docSorts))
	maxLen := 0
	for i, s := range docSorts {
		enum[i] = s.Token
		maxLen = max(maxLen, len(s.Token))
	}
	return &huma.Schema{
		Type:      huma.TypeString,
		Enum:      enum,
		MaxLength: &maxLen,
		Default:   defaultDocSort,
		Description: "Sort order; ties break on id. Default position_asc. " +
			"position: the display order staff set with putDocOrder. published: published_at. views: view_count.",
	}
}

type listDocsInput struct {
	collect.Page
	Sort        DocSortToken `query:"sort" default:"position_asc"`
	DocCategory string       `query:"doc_category" enum:"galgame,notice,kun,other" maxLength:"7" doc:"When set, only docs on this shelf. Omitted means every shelf."`
	IsPinned    optionalBool `query:"is_pinned" doc:"true for pinned docs only, false for unpinned only. Omitted means both."`
}

type listDocsOutput struct {
	Body repr.List[DocSummary]
}

type getDocInput struct {
	DocSlug string `path:"doc_slug" pattern:"^[a-z0-9]+(?:-[a-z0-9]+)*$" maxLength:"128" doc:"The doc's slug."`
}

type getDocOutput struct {
	Body Doc
}

type adminDocInput struct {
	DocID string `path:"doc_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Doc id."`
}

type adminDocOutput struct {
	Body AdminDoc
}

type createDocInput struct {
	Body DocCreate
}

type createDocOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new doc's staff view, /api/v1/admin/docs/{doc_id}."`
	Body     AdminDoc
}

type updateDocInput struct {
	DocID string `path:"doc_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Doc id."`
	Body  DocPatch
}

type putDocOrderInput struct {
	Body DocOrder
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

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listDocs",
			Method:      http.MethodGet,
			Path:        "/docs",
			Summary:     "List docs",
			Description: "Every help-center doc. There are no drafts: a doc exists publicly or not at all. " +
				"A cursor collection; the cursor is bound to sort, doc_category and is_pinned.",
			Tags: []string{"docs"},
		}), s.listDocs)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "getDoc",
			Method:      http.MethodGet,
			Path:        "/docs/{doc_slug}",
			Summary:     "Get a doc",
			Description: "A doc with its body, addressed by the slug of its /doc/{slug} page. Each successful read counts one view.",
			Tags:        []string{"docs"},
			Responses: problemResponses(map[int]string{
				404: "NOT_FOUND when no doc has this slug.",
				503: "SERVICE_UNAVAILABLE when the account service cannot resolve the author or a mention.",
			}),
		}), s.getDoc)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getAdminDoc",
			Method:      http.MethodGet,
			Path:        "/admin/docs/{doc_id}",
			Summary:     "Get a doc's staff view",
			Description: "The doc as its editor needs it, Markdown source included. It needs the doc.edit permission, which a Bearer request never carries.",
			Tags:        []string{"docs"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks doc.edit.",
				404: "NOT_FOUND when the doc does not exist.",
			}),
		}), s.getAdminDoc)

		huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
			OperationID:   "createDoc",
			Method:        http.MethodPost,
			Path:          "/admin/docs",
			Summary:       "Create a doc",
			DefaultStatus: http.StatusCreated,
			Description: "Publishes a doc at once, last in display order. Location points at its staff view. " +
				"It needs the doc.create permission.",
			Tags: []string{"docs"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks doc.create.",
				409: "ALREADY_EXISTS when another doc uses the slug.",
			}),
		})), s.createDoc)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateDoc",
			Method:      http.MethodPatch,
			Path:        "/admin/docs/{doc_id}",
			Summary:     "Update a doc",
			Description: "Changes the fields sent and leaves the rest. edited_at moves only when a field other than is_pinned takes a new value. " +
				"It needs the doc.edit permission.",
			Tags: []string{"docs"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks doc.edit.",
				404: "NOT_FOUND when the doc does not exist.",
				409: "ALREADY_EXISTS when another doc uses the new slug.",
			}),
		}), s.updateDoc)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "deleteDoc",
			Method:        http.MethodDelete,
			Path:          "/admin/docs/{doc_id}",
			Summary:       "Delete a doc",
			DefaultStatus: http.StatusNoContent,
			Description:   "Deletes the doc for good; its /doc/{slug} page stops resolving. It needs the doc.delete permission.",
			Tags:          []string{"docs"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks doc.delete.",
				404: "NOT_FOUND when the doc does not exist.",
			}),
		}), s.deleteDoc)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "putDocOrder",
			Method:        http.MethodPut,
			Path:          "/admin/doc-order",
			Summary:       "Set the doc display order",
			DefaultStatus: http.StatusNoContent,
			Description: "Replaces the display order that listDocs sorts position_asc by. The list must name every doc exactly once, " +
				"so a list made before someone else created or deleted a doc is refused rather than half applied. It needs the doc.edit permission.",
			Tags: []string{"docs"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks doc.edit.",
			}),
		}), s.putDocOrder)
	}
}
