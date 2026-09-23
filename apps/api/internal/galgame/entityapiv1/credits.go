package entityapiv1

import (
	"context"
	"slices"
	"strconv"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/problem"
)

var personPage = map[string]func(string) string{
	"vndb":    func(id string) string { return "https://vndb.org/s" + id },
	"bangumi": func(id string) string { return "https://bgm.tv/person/" + id },
	"erogamescape": func(id string) string {
		return "https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/creater.php?creater=" + id
	},
}

var characterPage = map[string]func(string) string{
	"vndb":    func(id string) string { return "https://vndb.org/" + id },
	"bangumi": func(id string) string { return "https://bgm.tv/character/" + id },
}

type searchInput struct {
	Q     string `query:"q" required:"true" minLength:"1" maxLength:"100" doc:"Name search, required: this family has no browse order. The collection is catalog's 100 best name matches in relevance order. Free text; never use it as a decision input."`
	Page  int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 100."`
	Limit int    `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Page size. 1–100, default 20. Values above 100 are rejected, not clamped."`
}

func (in searchInput) query() (string, *problem.Problem) {
	q := trimQuery(in.Q)
	if q == "" {
		return "", problem.New(problem.CodeInvalidParameter, "q is only whitespace.",
			problem.AtParameter("q", problem.ReasonTooShort, "q needs at least one character besides whitespace", nil))
	}
	return q, checkDepth(in.Page, in.Limit, true)
}

type listCreditNamesOutput struct {
	Body repr.PageList[CreditNameRef]
}

func (s *Service) listCreditNames(ctx context.Context, in *searchInput) (*listCreditNamesOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	q, prob := in.query()
	if prob != nil {
		return nil, prob
	}
	hits, _, appErr := s.catalog.CatalogEntitySearch(ctx, "names", q, 1, searchDepth)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	rows := make([]CreditNameRef, 0, len(hits))
	for i := range hits {
		h := &hits[i]
		rows = append(rows, CreditNameRef{
			Object:      "credit_name",
			ID:          repr.ID(int(h.ID)),
			CatalogName: workrepr.Name(h.DisplayName, h.Latin, client.LocalizedValues(h.Localized)),
		})
	}
	return &listCreditNamesOutput{Body: pageList(rows, in.Page, in.Limit)}, nil
}

type creditNamePathInput struct {
	CreditNameID string `path:"credit_name_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Credit name id."`
}

type getCreditNameOutput struct {
	Body CreditName
}

func (s *Service) creditName(ctx context.Context, rawID string, limit, offset int) (*client.CatalogName, *problem.Problem) {
	id, ok := pathID(rawID)
	if !ok {
		return nil, notFound()
	}
	n, found, movedTo, appErr := s.catalog.CatalogNameDetail(ctx, int64(id), limit, offset)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	if movedTo != 0 {
		return nil, merged("credit_name", movedTo)
	}
	if !found {
		return nil, notFound()
	}
	return n, nil
}

func creditNameRef(p *client.CatalogPerson) CreditNameRef {
	return CreditNameRef{
		Object:      "credit_name",
		ID:          repr.ID(int(p.ID)),
		CatalogName: workrepr.Name(p.DisplayName, p.Latin, client.LocalizedValues(p.Localized)),
		Lang:        workrepr.Lang(p.Lang),
	}
}

func (s *Service) getCreditName(ctx context.Context, in *creditNamePathInput) (*getCreditNameOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	n, prob := s.creditName(ctx, in.CreditNameID, 1, 0)
	if prob != nil {
		return nil, prob
	}
	links := make([]workrepr.CatalogLink, 0, len(n.Refs)+len(n.Links))
	for _, ref := range n.Refs {
		if tpl, ok := personPage[ref.Source]; ok && ref.ExternalID != "" {
			links = workrepr.AppendLink(links, ref.Source, tpl(ref.ExternalID))
		}
	}
	for _, l := range n.Links {
		links = workrepr.AppendLink(links, l.Source, l.URL)
	}
	siblings := make([]CreditNameRef, 0, len(n.Siblings))
	for i := range n.Siblings {
		siblings = append(siblings, creditNameRef(&n.Siblings[i]))
	}
	var gender *string
	if n.Gender != nil {
		switch *n.Gender {
		case 1:
			g := "male"
			gender = &g
		case 2:
			g := "female"
			gender = &g
		}
	}
	return &getCreditNameOutput{Body: CreditName{
		Object:      "credit_name",
		ID:          repr.ID(int(n.ID)),
		CatalogName: workrepr.Name(n.DisplayName, n.Latin, client.LocalizedValues(n.Localized)),
		Lang:        workrepr.Lang(n.Lang),
		Photo:       workrepr.ImageFromHash(s.cdn, n.PhotoHash),
		Gender:      gender,
		BirthYear:   positive(n.BirthY, 9999),
		BirthMonth:  positive(n.BirthM, 12),
		BirthDay:    positive(n.BirthD, 31),
		Intros:      workrepr.Intros(n.Intros),
		Links:       links,
		Siblings:    siblings,
	}}, nil
}

func positive(v *int, maximum int) *int {
	if v == nil || *v < 1 || *v > maximum {
		return nil
	}
	return v
}

type StreamQuery struct {
	Cursor      string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque cursor from a previous page of this collection."`
	Limit       int    `query:"limit" minimum:"1" maximum:"50" default:"20" doc:"Page size. 1–50, default 20. Values above 50 are rejected, not clamped. A page can come back shorter when works on it are hidden from this reader."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false. A cursor only continues the include_nsfw it was made with."`
}

const streamSort = "catalog"

func (in StreamQuery) offset(family, id string) (int, *problem.Problem) {
	fp := collect.Fingerprint(family, id, strconv.FormatBool(in.IncludeNSFW))
	keys, prob := collect.DecodeCursor(in.Cursor, streamSort, fp)
	if prob != nil {
		return 0, prob
	}
	if len(keys) == 0 {
		return 0, nil
	}
	offset, err := strconv.Atoi(keys[0])
	if err != nil || offset < 0 {
		return 0, problem.New(problem.CodeInvalidCursor, "The cursor cannot be parsed or is no longer valid.",
			problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil))
	}
	return offset, nil
}

func (in StreamQuery) next(family, id string, nextOffset *int) *string {
	if nextOffset == nil {
		return nil
	}
	fp := collect.Fingerprint(family, id, strconv.FormatBool(in.IncludeNSFW))
	cur := collect.EncodeCursor(streamSort, fp, strconv.Itoa(*nextOffset))
	return &cur
}

type listCreditsInput struct {
	CreditNameID string `path:"credit_name_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Credit name id."`
	StreamQuery
}

type listCreditsOutput struct {
	Body repr.List[Credit]
}

func (s *Service) listCreditNameCredits(ctx context.Context, in *listCreditsInput) (*listCreditsOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	offset, prob := in.offset("credit_name", in.CreditNameID)
	if prob != nil {
		return nil, prob
	}
	n, prob := s.creditName(ctx, in.CreditNameID, in.Limit, offset)
	if prob != nil {
		return nil, prob
	}
	ids := make([]int64, 0, len(n.Credits))
	for _, c := range n.Credits {
		ids = append(ids, c.Work.ID)
	}
	byID, prob := s.visibleRows(ctx, ids, in.IncludeNSFW)
	if prob != nil {
		return nil, prob
	}
	items := make([]Credit, 0, len(n.Credits))
	for _, c := range n.Credits {
		work, ok := byID[c.Work.ID]
		if !ok {
			continue
		}
		var keys []string
		labelOf := map[string]string{}
		creditChars := []CreditCharacter{}
		seenChar := map[string]bool{}
		for _, r := range c.Roles {
			key := client.StaffRoleCanonicalKey(r.RoleKey)
			labelOf[key] = client.StaffRoleLabel(r.RoleKey, r.RoleName)
			if !slices.Contains(keys, key) {
				keys = append(keys, key)
			}
			if r.Character == "" {
				continue
			}
			charKey := strconv.FormatInt(r.CharacterID, 10) + "\x00" + r.Character
			if seenChar[charKey] {
				continue
			}
			seenChar[charKey] = true
			cc := CreditCharacter{DisplayName: r.Character}
			if r.CharacterID > 0 {
				id := repr.ID(int(r.CharacterID))
				cc.CharacterID = &id
			}
			creditChars = append(creditChars, cc)
		}
		if len(keys) > 1 {
			keys = slices.DeleteFunc(keys, func(k string) bool { return k == client.StaffRoleOtherKey })
		}
		roles := make([]CreditRole, 0, len(keys))
		for _, key := range client.SortStaffRoleKeys(keys) {
			roles = append(roles, CreditRole{RoleKey: key, DisplayName: labelOf[key]})
		}
		items = append(items, Credit{Object: "credit", WorkSummary: work, CreditRoles: roles, Characters: creditChars})
	}
	return &listCreditsOutput{Body: repr.NewList(items, in.next("credit_name", in.CreditNameID, n.NextOffset))}, nil
}

// visibleRows hydrates the works a credit or appearance page names, dropping the
// ones this reader's content limit hides.
func (s *Service) visibleRows(ctx context.Context, ids []int64, includeNSFW bool) (map[int64]workrepr.WorkSummary, *problem.Problem) {
	out := map[int64]workrepr.WorkSummary{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, appErr := s.catalog.CatalogRowsByCatalogIDs(ctx, ids, !includeNSFW)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	ordered := make([]client.CatalogWorkListItem, 0, len(ids))
	for _, id := range ids {
		if row, ok := rows[id]; ok && client.CatalogItemRenderable(&row) {
			ordered = append(ordered, row)
		}
	}
	works, prob := s.works.FromRows(ctx, ordered)
	if prob != nil {
		return nil, prob
	}
	for i := range ordered {
		out[ordered[i].ID] = works[i]
	}
	return out, nil
}
