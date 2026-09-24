package entityapiv1

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/galgame/workrepr"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

// memberPageCap bounds one membership walk at 200 catalog pages, as the legacy
// entity pages did.
const memberPageCap = 200

type workSort struct {
	Token string
	Field string
	Order string
}

var workSorts = func() []workSort {
	fields := []struct{ token, field string }{
		{"resource_updated", "resource_update_time"},
		{"created", "created"},
		{"view", "view"},
		{"view_1d", "view_1d"},
		{"view_7d", "view_7d"},
		{"view_30d", "view_30d"},
		{"release_date", "release_date"},
		{"rating", "rating"},
	}
	out := make([]workSort, 0, len(fields)*2)
	for _, f := range fields {
		out = append(out, workSort{f.token + "_desc", f.field, "desc"}, workSort{f.token + "_asc", f.field, "asc"})
	}
	return out
}()

const defaultWorkSort = "resource_updated_desc"

func WorkSortParts(token string) (field, order string, ok bool) {
	if token == "" {
		token = defaultWorkSort
	}
	for _, s := range workSorts {
		if s.Token == token {
			return s.Field, s.Order, true
		}
	}
	return "", "", false
}

type WorkSortToken string

func (WorkSortToken) Schema(huma.Registry) *huma.Schema {
	enum := make([]any, len(workSorts))
	maxLen := 0
	for i, s := range workSorts {
		enum[i] = s.Token
		maxLen = max(maxLen, len(s.Token))
	}
	return &huma.Schema{
		Type:      huma.TypeString,
		Enum:      enum,
		MaxLength: &maxLen,
		Default:   defaultWorkSort,
		Description: "Sort order. resource_updated: when a resource last changed. created: when the forum page was made. " +
			"view / view_1d / view_7d / view_30d: page reads, all time or over the last day, 7 or 30 days. " +
			"release_date: the release date. rating: the bayesian forum rating. Works the forum has no page for rank after every work it has.",
	}
}

type GameTypeFilter string

func (GameTypeFilter) Schema(huma.Registry) *huma.Schema {
	keys := append(append([]string{}, workrepr.GameTypes...), "uncategorized")
	enum := make([]any, len(keys))
	maxLen := 0
	for i, k := range keys {
		enum[i] = k
		maxLen = max(maxLen, len(k))
	}
	return &huma.Schema{Type: huma.TypeString, Enum: enum, MaxLength: &maxLen,
		Description: "Only works a forum rating labels with this game type; uncategorized is works no rating labels at all. Omitted means no filter."}
}

type WorksQuery struct {
	Page             int                       `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit            int                       `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	Sort             WorkSortToken             `query:"sort" default:"resource_updated_desc"`
	ResourceType     workrepr.ResourceType     `query:"resource_type" doc:"Only works with at least one forum resource of this type. Omitted means no filter."`
	ResourcePlatform workrepr.ResourcePlatform `query:"resource_platform" doc:"Only works with at least one forum resource for this platform. Omitted means no filter."`
	ResourceLanguage workrepr.ResourceLanguage `query:"resource_language" doc:"Only works with at least one forum resource in this language. Omitted means no filter."`
	ResourceRuntime  workrepr.ResourceRuntime  `query:"resource_runtime" doc:"Only works with at least one forum resource that runs through this runtime. emulator matches every emulator runtime, including a resource whose uploader named none. Omitted means no filter."`
	GameType         GameTypeFilter            `query:"game_type"`
	IncludeNSFW      bool                      `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
}

const worksDescription = "A page-number collection. Without a resource or game_type filter it holds every catalog work filed under the entity, " +
	"including works the forum has no page for; any of those filters narrows it to works with a forum resource. " +
	"Ties break on catalog's own order, or on descending id once a filter applies."

func (p WorksQuery) filter() model.GalgameListFilter {
	f := model.GalgameListFilter{
		Type:         string(p.ResourceType),
		PlatformAxis: string(p.ResourcePlatform),
		LanguageAxis: string(p.ResourceLanguage),
		GameType:     string(p.GameType),
		SortField:    "resource_update_time",
		SortOrder:    "desc",
		Page:         p.Page,
		Limit:        p.Limit,
	}
	if p.ResourceRuntime != "" {
		f.RuntimeAxes = []string{string(p.ResourceRuntime)}
	}
	token := string(p.Sort)
	if token == "" {
		token = defaultWorkSort
	}
	for _, s := range workSorts {
		if s.Token == token {
			f.SortField, f.SortOrder = s.Field, s.Order
		}
	}
	return f
}

type member struct {
	workID int
	via    *client.CatalogLabelVia
}

type walkFunc func(ctx context.Context, catalogSort string, isSFW bool) ([]member, *legacyErrors.AppError)

func (s *Service) walkMembers(key, id string) walkFunc {
	return func(ctx context.Context, catalogSort string, isSFW bool) ([]member, *legacyErrors.AppError) {
		q := url.Values{key: {id}}
		if catalogSort != "" {
			q.Set("sort", catalogSort)
		}
		ids, appErr := s.catalog.CatalogMemberWorkIDs(ctx, q, isSFW, memberPageCap)
		if appErr != nil {
			return nil, appErr
		}
		out := make([]member, len(ids))
		for i, workID := range ids {
			out[i] = member{workID: workID}
		}
		return out, nil
	}
}

// memberPage is the entity lane: catalog membership, then the forum's own
// ranking and paging. A resource or game-type filter drops the page to local SQL,
// which only knows works that carry a resource; the lane never switches on
// whether the filters happen to be empty.
func (s *Service) memberPage(
	ctx context.Context, walk walkFunc, p WorksQuery, keep func(member) bool,
) ([]int, map[int]*client.CatalogLabelVia, int, *problem.Problem) {
	f := p.filter()
	catalogSort := service.CatalogMemberSort(f)
	members, appErr := walk(ctx, catalogSort, !p.IncludeNSFW)
	if appErr != nil {
		return nil, nil, 0, unavailable(appErr)
	}
	ids := make([]int, 0, len(members))
	via := make(map[int]*client.CatalogLabelVia, len(members))
	for _, m := range members {
		if keep != nil && !keep(m) {
			continue
		}
		ids = append(ids, m.workID)
		if m.via != nil {
			via[m.workID] = m.via
		}
	}
	if len(ids) == 0 {
		return []int{}, via, 0, nil
	}
	f.RestrictIDs = ids
	if service.EntityUsesLocalList(f) {
		f.SFWOnly = !p.IncludeNSFW
		pageIDs, total, err := s.lists.ListIDs(f)
		if err != nil {
			return nil, nil, 0, problem.Internal(err)
		}
		return pageIDs, via, int(total), nil
	}
	if catalogSort == "" {
		ids = s.lists.OrderRestrictIDs(ids, f)
	}
	return pageOf(ids, p.Page, p.Limit), via, len(ids), nil
}

func (s *Service) worksPage(ctx context.Context, walk walkFunc, p WorksQuery) (repr.PageList[workrepr.WorkSummary], *problem.Problem) {
	if prob := checkDepth(p.Page, p.Limit, false); prob != nil {
		return repr.PageList[workrepr.WorkSummary]{}, prob
	}
	ids, _, count, prob := s.memberPage(ctx, walk, p, nil)
	if prob != nil {
		return repr.PageList[workrepr.WorkSummary]{}, prob
	}
	items, prob := s.works.ByIDs(ctx, ids, p.IncludeNSFW)
	if prob != nil {
		return repr.PageList[workrepr.WorkSummary]{}, prob
	}
	total, relation := collect.ClampTotal(count)
	return repr.NewPageList(items, total, relation), nil
}

type listTaggedWorksInput struct {
	TagIDs      []repr.DecimalID `query:"tag_ids" required:"true" minItems:"1" maxItems:"10" doc:"Tag ids, comma-separated. Only works carrying every one of them. 1–10."`
	Page        int              `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit       int              `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool             `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
}

type workPageOutput struct {
	Body repr.PageList[workrepr.WorkSummary]
}

func (s *Service) listTaggedWorks(ctx context.Context, in *listTaggedWorksInput) (*workPageOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if prob := checkDepth(in.Page, in.Limit, false); prob != nil {
		return nil, prob
	}
	ids := make([]string, 0, len(in.TagIDs))
	seen := map[string]bool{}
	for _, raw := range in.TagIDs {
		id, ok := repr.ParseID(raw)
		if !ok {
			return nil, problem.New(problem.CodeInvalidParameter, "tag_ids holds something that is not a tag id.",
				problem.AtParameter("tag_ids", problem.ReasonInvalidFormat, "each tag id is a positive decimal integer", nil))
		}
		key := repr.ID(id)
		if !seen[string(key)] {
			seen[string(key)] = true
			ids = append(ids, string(key))
		}
	}
	q := url.Values{
		"tag_id":  {strings.Join(ids, ",")},
		"page":    {strconv.Itoa(in.Page)},
		"limit":   {strconv.Itoa(in.Limit)},
		"include": {workrepr.RowInclude},
		"sort":    {"released_desc"},
	}
	client.ApplyWorksGate(q, !in.IncludeNSFW)
	res, appErr := s.catalog.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	rows := make([]client.CatalogWorkListItem, 0, len(res.Items))
	for i := range res.Items {
		if client.CatalogItemRenderable(&res.Items[i]) {
			rows = append(rows, res.Items[i])
		}
	}
	items, prob := s.works.FromRows(ctx, rows)
	if prob != nil {
		return nil, prob
	}
	total, relation := collect.ClampTotal(int(res.Total))
	return &workPageOutput{Body: repr.NewPageList(items, total, relation)}, nil
}
