package entitypreview

import (
	"context"
	"net/http"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

const maxRefs = 50

type listInput struct {
	TopicIDs           []repr.DecimalID `query:"topic_ids" required:"false" maxItems:"50" doc:"Topic ids, comma-separated."`
	WorkIDs            []repr.DecimalID `query:"work_ids" required:"false" maxItems:"50" doc:"Work ids, comma-separated."`
	GalgameResourceIDs []repr.DecimalID `query:"galgame_resource_ids" required:"false" maxItems:"50" doc:"Galgame resource ids, comma-separated."`
	UserIDs            []repr.DecimalID `query:"user_ids" required:"false" maxItems:"50" doc:"User ids, comma-separated."`
	CharacterIDs       []repr.DecimalID `query:"character_ids" required:"false" maxItems:"50" doc:"Character ids, comma-separated."`
	CompanyIDs         []repr.DecimalID `query:"company_ids" required:"false" maxItems:"50" doc:"Company ids, comma-separated."`
	TagIDs             []repr.DecimalID `query:"tag_ids" required:"false" maxItems:"50" doc:"Tag ids, comma-separated."`
	SeriesIDs          []repr.DecimalID `query:"series_ids" required:"false" maxItems:"50" doc:"Series ids, comma-separated."`
	CreditNameIDs      []repr.DecimalID `query:"credit_name_ids" required:"false" maxItems:"50" doc:"Credit name ids, comma-separated."`
	RatingIDs          []repr.DecimalID `query:"rating_ids" required:"false" maxItems:"50" doc:"Rating ids, comma-separated."`
}

type EntityPreviewList struct {
	Object string                 `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items  []content.EntityTarget `json:"items" doc:"One target per requested id, in the order topic_ids, work_ids, galgame_resource_ids, user_ids, character_ids, company_ids, tag_ids, series_ids, credit_name_ids, rating_ids, each in request order. A target this reader may not preview is unavailable_entity; a user who no longer exists has name null. Empty array, never null."`
}

type listOutput struct {
	Body EntityPreviewList
}

func Register(convert *content.Converter) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listEntityPreviews",
			Method:      http.MethodGet,
			Path:        "/entity-previews",
			Summary:     "Preview linked entities",
			Description: "Returns what an entity_card or entity_link would show for each id, resolved exactly as in a content document. " +
				"It is the same for every reader. At least one id and at most 50 in all. Not paginated.",
			Tags: []string{"content"},
			Responses: problemResponses(map[int]string{
				422: "VALIDATION_FAILED when no id is given, more than 50 are given in all, or an id is not a positive integer.",
				503: "SERVICE_UNAVAILABLE when a lookup the previews need cannot be made.",
			}),
		}), func(ctx context.Context, in *listInput) (*listOutput, error) {
			refs, prob := parseRefs(in)
			if prob != nil {
				return nil, prob
			}
			targets, err := convert.Targets(ctx, refs)
			if err != nil {
				return nil, problem.Unavailable(err)
			}
			return &listOutput{Body: EntityPreviewList{Object: "list", Items: targets}}, nil
		})
	}
}

func parseRefs(in *listInput) ([]content.EntityRef, *problem.Problem) {
	groups := []struct {
		param string
		typ   content.EntityType
		ids   []repr.DecimalID
	}{
		{"topic_ids", content.EntityTopic, in.TopicIDs},
		{"work_ids", content.EntityWork, in.WorkIDs},
		{"galgame_resource_ids", content.EntityGalgameResource, in.GalgameResourceIDs},
		{"user_ids", content.EntityUser, in.UserIDs},
		{"character_ids", content.EntityCharacter, in.CharacterIDs},
		{"company_ids", content.EntityCompany, in.CompanyIDs},
		{"tag_ids", content.EntityTag, in.TagIDs},
		{"series_ids", content.EntitySeries, in.SeriesIDs},
		{"credit_name_ids", content.EntityCreditName, in.CreditNameIDs},
		{"rating_ids", content.EntityRating, in.RatingIDs},
	}
	var refs []content.EntityRef
	for _, g := range groups {
		for _, raw := range g.ids {
			id, ok := repr.ParseID(raw)
			if !ok {
				return nil, validationFailed(problem.AtParameter(g.param, problem.ReasonInvalidFormat, "ids are positive integers", nil))
			}
			refs = append(refs, content.EntityRef{Type: g.typ, ID: id})
		}
	}
	if len(refs) == 0 {
		return nil, validationFailed(problem.AtParameter("topic_ids", problem.ReasonRequired, "give at least one id", nil))
	}
	if len(refs) > maxRefs {
		max := maxRefs
		return nil, validationFailed(problem.AtParameter("topic_ids", problem.ReasonTooManyItems, "at most 50 ids in all", &problem.FieldParams{MaxItems: &max}))
	}
	return refs, nil
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
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
