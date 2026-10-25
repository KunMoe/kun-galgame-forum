package apiv1

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/entityapiv1"
	"kun-galgame-api/pkg/catalogclient"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

const (
	companyFollowSweepPage  = 100
	companyFollowSweepPages = 10
)

type companyTaxonomy interface {
	CatalogTaxonomyList(ctx context.Context, entity string, q url.Values) (*client.CatalogTaxonomyPage, *legacyErrors.AppError)
}

type companyFollowPathInput struct {
	CompanyID string `path:"company_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Company id."`
}

type CompanyFollowState struct {
	Object      string         `json:"object" enum:"company_follow_state" maxLength:"20" doc:"Type discriminant. Always company_follow_state."`
	ID          repr.DecimalID `json:"id" doc:"Company id, the same value as company_id."`
	CompanyID   repr.DecimalID `json:"company_id" doc:"Company id: the catalog company id, which is also the id in the web's /galgame/official/{id}."`
	IsFollowing bool           `json:"is_following" doc:"Whether the caller follows this company."`
}

type companyFollowStateOutput struct {
	Body CompanyFollowState
}

type listMyFollowedCompaniesInput struct {
	Page  int `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit int `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
}

type FollowedCompany struct {
	Object     string                     `json:"object" enum:"followed_company" maxLength:"16" doc:"Type discriminant. Always followed_company."`
	Company    entityapiv1.CompanySummary `json:"company" doc:"The followed company."`
	FollowedAt *repr.DateTime             `json:"followed_at" doc:"When the caller followed this company. null when catalog did not send a time."`
}

type FollowedCompanyList struct {
	Object        string            `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items         []FollowedCompany `json:"items" doc:"Members of this page, newest follow first. Empty array, never null."`
	Total         int               `json:"total" minimum:"0" doc:"Follows in the swept list, minus companies on this page that catalog no longer renders."`
	TotalRelation string            `json:"total_relation" enum:"eq,gte" maxLength:"3" doc:"eq when total is exact, gte when it stopped at the depth limit and there are at least that many."`
	IsTruncated   bool              `json:"is_truncated" doc:"Whether the upstream sweep stopped at the 10×100 cap."`
}

type listMyFollowedCompaniesOutput struct {
	Body FollowedCompanyList
}

func (s *Service) registerCompanyFollows(api huma.API) {
	tag := []string{"me"}
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getMyCompanyFollow",
		Method:      http.MethodGet,
		Path:        "/me/followed-companies/{company_id}",
		Summary:     "Whether the caller follows a company",
		Description: "Reads the caller's follow of this company. A company the caller does not follow is 200 with is_following false, not 404.",
		Tags:        tag,
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "MISSING_CREDENTIAL when the request has no credentials; INVALID_CREDENTIAL when the session has no OAuth access token or catalog rejects it.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.getMyCompanyFollow)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putMyCompanyFollow",
		Method:      http.MethodPut,
		Path:        "/me/followed-companies/{company_id}",
		Summary:     "Follow a company",
		Description: "Follows the company. Repeating the follow changes nothing. The response is is_following true.",
		Tags:        tag,
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "MISSING_CREDENTIAL when the request has no credentials; INVALID_CREDENTIAL when the session has no OAuth access token or catalog rejects it.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when catalog does not know the company, including an id that was merged away.",
			422: "VALIDATION_FAILED when the account already follows as many companies as catalog allows.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.putMyCompanyFollow)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "deleteMyCompanyFollow",
		Method:      http.MethodDelete,
		Path:        "/me/followed-companies/{company_id}",
		Summary:     "Unfollow a company",
		Description: "Withdraws the caller's follow of this company. Repeating the unfollow is 200 with is_following false.",
		Tags:        tag,
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "MISSING_CREDENTIAL when the request has no credentials; INVALID_CREDENTIAL when the session has no OAuth access token or catalog rejects it.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.deleteMyCompanyFollow)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyFollowedCompanies",
		Method:      http.MethodGet,
		Path:        "/me/followed-companies",
		Summary:     "List the caller's followed companies",
		Description: "A page-number collection of the companies the caller follows, newest follow first. A company catalog no longer renders is dropped from this page's items and from total.",
		Tags:        tag,
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE or INVALID_PARAMETER when page × limit is too deep.",
			401: "MISSING_CREDENTIAL when the request has no credentials; INVALID_CREDENTIAL when the session has no OAuth access token or catalog rejects it.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listMyFollowedCompanies)
}

func (s *Service) getMyCompanyFollow(ctx context.Context, in *companyFollowPathInput) (*companyFollowStateOutput, error) {
	token, id, p := s.companyFollowCall(ctx, in.CompanyID)
	if p != nil {
		return nil, p
	}
	following, err := s.catalog.MyCompanyFollow(ctx, token, id)
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	return &companyFollowStateOutput{Body: companyFollowState(id, following)}, nil
}

func (s *Service) putMyCompanyFollow(ctx context.Context, in *companyFollowPathInput) (*companyFollowStateOutput, error) {
	token, id, p := s.companyFollowCall(ctx, in.CompanyID)
	if p != nil {
		return nil, p
	}
	if err := s.catalog.FollowCompany(ctx, token, id); err != nil {
		return nil, mapUserPlane(err, false)
	}
	return &companyFollowStateOutput{Body: companyFollowState(id, true)}, nil
}

func (s *Service) deleteMyCompanyFollow(ctx context.Context, in *companyFollowPathInput) (*companyFollowStateOutput, error) {
	token, id, p := s.companyFollowCall(ctx, in.CompanyID)
	if p != nil {
		return nil, p
	}
	if err := s.catalog.UnfollowCompany(ctx, token, id); err != nil {
		return nil, mapUserPlane(err, false)
	}
	return &companyFollowStateOutput{Body: companyFollowState(id, false)}, nil
}

func (s *Service) companyFollowCall(ctx context.Context, raw string) (string, int64, *problem.Problem) {
	if _, p := s.requireActive(ctx); p != nil {
		return "", 0, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return "", 0, p
	}
	id, ok := parseCompanyID(raw)
	if !ok {
		return "", 0, notFound()
	}
	if s.catalog == nil {
		return "", 0, problem.Unavailable(errUnconfigured)
	}
	return token, id, nil
}

func (s *Service) listMyFollowedCompanies(ctx context.Context, in *listMyFollowedCompaniesInput) (*listMyFollowedCompaniesOutput, error) {
	if _, p := s.requireActive(ctx); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	pg := browsePage(in.Page, in.Limit)
	if prob := pg.CheckDepth(); prob != nil {
		return nil, prob
	}
	rows, truncated, err := s.sweepCompanyFollows(ctx, token)
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	byID, err := s.followedCompanySummaries(ctx, rows)
	if err != nil {
		return nil, err
	}
	visible := make([]FollowedCompany, 0, len(rows))
	for _, rec := range rows {
		sum, ok := byID[rec.CompanyID]
		if !ok {
			continue
		}
		visible = append(visible, FollowedCompany{
			Object:     "followed_company",
			Company:    sum,
			FollowedAt: followTimePtr(rec.CreatedAt),
		})
	}
	total, rel := collect.ClampTotal(len(visible))
	if truncated {
		rel = "gte"
	}
	items := []FollowedCompany{}
	if start := pg.Offset(); start < len(visible) {
		items = visible[start:min(start+pg.Limit, len(visible))]
	}
	return &listMyFollowedCompaniesOutput{Body: FollowedCompanyList{
		Object:        "list",
		Items:         items,
		Total:         total,
		TotalRelation: rel,
		IsTruncated:   truncated,
	}}, nil
}

func (s *Service) sweepCompanyFollows(ctx context.Context, token string) ([]catalogclient.CompanyFollowRecord, bool, error) {
	var all []catalogclient.CompanyFollowRecord
	cursor := ""
	for i := 0; i < companyFollowSweepPages; i++ {
		rows, next, err := s.catalog.ListMyCompanyFollows(ctx, token, cursor, companyFollowSweepPage)
		if err != nil {
			return nil, false, err
		}
		all = append(all, rows...)
		if next == "" {
			return all, false, nil
		}
		cursor = next
	}
	return all, true, nil
}

func (s *Service) followedCompanySummaries(ctx context.Context, rows []catalogclient.CompanyFollowRecord) (map[int64]entityapiv1.CompanySummary, error) {
	out := map[int64]entityapiv1.CompanySummary{}
	if len(rows) == 0 {
		return out, nil
	}
	lister, ok := s.works.(companyTaxonomy)
	if !ok {
		return nil, problem.Internal(errUnconfigured)
	}
	for start := 0; start < len(rows); start += companyFollowSweepPage {
		chunk := rows[start:min(start+companyFollowSweepPage, len(rows))]
		ids := make([]string, len(chunk))
		for i, row := range chunk {
			ids[i] = strconv.FormatInt(row.CompanyID, 10)
		}
		q := client.OpenPopulation(url.Values{"ids": {strings.Join(ids, ",")}})
		res, appErr := lister.CatalogTaxonomyList(ctx, "labels", q)
		if appErr != nil {
			return nil, problem.Unavailable(appErr)
		}
		if res == nil {
			continue
		}
		for i := range res.Items {
			sum, ok := entityapiv1.CompanySummaryFromTaxonomy(s.cdn, &res.Items[i])
			if !ok {
				slog.Warn("followed companies: unknown company_kind, row dropped", "company_id", res.Items[i].ID, "company_kind", res.Items[i].Kind)
				continue
			}
			id, ok := repr.ParseID(sum.ID)
			if !ok {
				continue
			}
			out[int64(id)] = sum
		}
	}
	return out, nil
}

func companyFollowState(id int64, following bool) CompanyFollowState {
	raw := repr.ID(int(id))
	return CompanyFollowState{
		Object:      "company_follow_state",
		ID:          raw,
		CompanyID:   raw,
		IsFollowing: following,
	}
}

func parseCompanyID(raw string) (int64, bool) {
	id, ok := parseWorkID(raw)
	if !ok {
		return 0, false
	}
	return int64(id), true
}

func followTimePtr(raw string) *repr.DateTime {
	if raw == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		stamp := repr.Timestamp(t)
		return &stamp
	}
	stamp := repr.DateTime(raw)
	return &stamp
}
