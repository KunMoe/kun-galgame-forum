package apiv1

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/storelink"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/linkcheck"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

var errUnconfigured = errors.New("apiv1 galgame resource: service is not configured")

type Catalog interface {
	CatalogRowsByWorkIDs(ctx context.Context, ids []int, include, contentLimit string) (map[int]client.CatalogWorkListItem, *legacyErrors.AppError)
	CatalogWorksSearch(ctx context.Context, q url.Values) (*client.CatalogWorksPage, *legacyErrors.AppError)
}

type ClaimFunc func(ctx context.Context, accessToken string, workID int64) *legacyErrors.AppError

type ShareChecker interface {
	CheckShare(ctx context.Context, urls []string, passcode string) linkcheck.Status
}

type AwardFunc func(userID, delta int, reason, ref, idempotencyKey string)

type pendingAward struct {
	userID int
	delta  int
	reason string
	ref    string
	key    string
}

type accessTokenCtxKey struct{}

type Service struct {
	store      *repository.ResourceV1Store
	users      *userclient.Client
	convert    *content.Converter
	check      *gate.CheckService
	scan       *gate.ScanService
	award      AwardFunc
	getCatalog func() Catalog
	getClaim   func() ClaimFunc
	getCheck   func() ShareChecker
	storeLinks *storelink.Resolver
	cdn        string
}

func New(
	store *repository.ResourceV1Store,
	users *userclient.Client,
	convert *content.Converter,
	check *gate.CheckService,
	scan *gate.ScanService,
	award AwardFunc,
	getCatalog func() Catalog,
	getClaim func() ClaimFunc,
	getCheck func() ShareChecker,
	storeLinks *storelink.Resolver,
	cdn string,
) *Service {
	if check == nil {
		check = gate.NewCheckService(nil)
	}
	if scan == nil {
		scan = gate.NewScanService(nil)
	}
	if award == nil {
		award = moemoepoint.Award
	}
	return &Service{
		store: store, users: users, convert: convert,
		check: check, scan: scan, award: award,
		getCatalog: getCatalog, getClaim: getClaim, getCheck: getCheck,
		storeLinks: storeLinks, cdn: cdn,
	}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.store == nil || !s.store.Ready() || s.users == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Service) catalog() Catalog {
	if s == nil || s.getCatalog == nil {
		return nil
	}
	return s.getCatalog()
}

func (s *Service) claim() ClaimFunc {
	if s == nil || s.getClaim == nil {
		return nil
	}
	return s.getClaim()
}

func (s *Service) shareCheck() ShareChecker {
	if s == nil || s.getCheck == nil {
		return nil
	}
	return s.getCheck()
}

func (s *Service) requireActive(ctx context.Context) (*middleware.UserInfo, *problem.Problem) {
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	users, err := s.users.Users(ctx, []int{user.ID})
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if u, ok := users[user.ID]; ok && !userclient.IsRenderable(u) {
		return nil, problem.New(problem.CodeAccountBanned, "The signed-in user's account is banned.")
	}
	return user, nil
}

func (s *Service) lookupUsers(ctx context.Context, ids []int) (map[int]userclient.User, *problem.Problem) {
	if s.users == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	users, err := s.users.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if users == nil {
		users = map[int]userclient.User{}
	}
	return users, nil
}

func (s *Service) keepAuthors(ctx context.Context, ids []int) ([]int, *problem.Problem) {
	users, p := s.lookupUsers(ctx, ids)
	if p != nil {
		return nil, p
	}
	keep := make([]int, 0, len(ids))
	for _, id := range ids {
		if renderable(users, id) {
			keep = append(keep, id)
		}
	}
	return keep, nil
}

func (s *Service) convertBody(ctx context.Context, source string) (content.ContentDocument, *problem.Problem) {
	if s.convert == nil {
		return content.ContentDocument{}, problem.Internal(errUnconfigured)
	}
	docs, err := s.convert.Convert(ctx, []string{source})
	if err != nil {
		return content.ContentDocument{}, problem.Unavailable(err)
	}
	return docs[0], nil
}

func (s *Service) rejectContent(ctx context.Context, text string, authorID int) (decision string, matched []string, p *problem.Problem) {
	if trimSpace(text) == "" {
		return gate.DecisionAllow, nil, nil
	}
	aid := int64(authorID)
	decision, matched = s.check.Decision(ctx, text, &aid)
	if decision == gate.DecisionDeny {
		return decision, matched, contentRejected()
	}
	return decision, matched, nil
}

func (s *Service) flushAwards(jobs []pendingAward) {
	for _, j := range jobs {
		s.award(j.userID, j.delta, j.reason, j.ref, j.key)
	}
}

func (s *Service) afterCommit(err error, jobs []pendingAward) error {
	if err != nil {
		return err
	}
	s.flushAwards(jobs)
	return nil
}

func (s *Service) lookupWork(ctx context.Context, workID int) (*client.CatalogWorkListItem, *problem.Problem) {
	cat := s.catalog()
	if cat == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	rows, appErr := cat.CatalogRowsByWorkIDs(ctx, []int{workID}, includeRefs, "all")
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	row, ok := rows[workID]
	if !ok || !client.CatalogItemRenderable(&row) {
		return nil, notFound()
	}
	cp := row
	return &cp, nil
}

func (s *Service) catalogRows(ctx context.Context, ids []int) (map[int]client.CatalogWorkListItem, *problem.Problem) {
	if len(ids) == 0 {
		return map[int]client.CatalogWorkListItem{}, nil
	}
	cat := s.catalog()
	if cat == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	rows, appErr := cat.CatalogRowsByWorkIDs(ctx, ids, includeRefs, "all")
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	if rows == nil {
		rows = map[int]client.CatalogWorkListItem{}
	}
	return rows, nil
}

func (s *Service) workRefs(ctx context.Context, rows map[int]client.CatalogWorkListItem, ids []int) map[int]repr.WorkRef {
	out := map[int]repr.WorkRef{}
	for _, id := range ids {
		row, ok := rows[id]
		if !ok || !client.CatalogItemRenderable(&row) {
			continue
		}
		cp := row
		out[id] = galgameapiv1.WorkRefOf(ctx, &cp, s.cdn)
	}
	return out
}

func (s *Service) searchWorkIDs(ctx context.Context, qtext string, includeNSFW bool) ([]int, *problem.Problem) {
	cat := s.catalog()
	if cat == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	q := url.Values{
		"q":       {qtext},
		"page":    {"1"},
		"limit":   {strconv.Itoa(catalogCap)},
		"sort":    {"relevance"},
		"include": {includeRefs},
	}
	client.ApplyWorksGate(q, !includeNSFW)
	res, appErr := cat.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	ids := make([]int, 0, len(res.Items))
	for i := range res.Items {
		if !client.CatalogItemRenderable(&res.Items[i]) {
			continue
		}
		if id := int(res.Items[i].ID); id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s *Service) visibleResource(ctx context.Context, rawID string) (*model.GalgameResource, map[int]userclient.User, *problem.Problem) {
	id, ok := parseID(rawID)
	if !ok {
		return nil, nil, notFound()
	}
	row, err := s.store.Find(id)
	if err != nil {
		if errors.Is(err, repository.ErrResourceNotFound) {
			return nil, nil, notFound()
		}
		return nil, nil, problem.Internal(err)
	}
	published, err := s.store.WorkPublished(row.WorkID)
	if err != nil {
		return nil, nil, problem.Internal(err)
	}
	if !published {
		return nil, nil, notFound()
	}
	users, p := s.lookupUsers(ctx, []int{row.UserID})
	if p != nil {
		return nil, nil, p
	}
	if !renderable(users, row.UserID) {
		return nil, nil, notFound()
	}
	return row, users, nil
}

func (s *Service) assemble(ctx context.Context, rows []model.GalgameResource, viewer *middleware.UserInfo) ([]GalgameResource, *problem.Problem) {
	if len(rows) == 0 {
		return []GalgameResource{}, nil
	}
	ids := make([]int, len(rows))
	authorIDs := make([]int, 0, len(rows))
	workIDs := make([]int, 0, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
		authorIDs = append(authorIDs, r.UserID)
		workIDs = append(workIDs, r.WorkID)
	}
	users, p := s.lookupUsers(ctx, authorIDs)
	if p != nil {
		return nil, p
	}
	catRows, p := s.catalogRows(ctx, workIDs)
	if p != nil {
		return nil, p
	}
	works := s.workRefs(ctx, catRows, workIDs)
	providers, err := s.store.ProviderNames(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var liked map[int]bool
	if viewer != nil {
		liked, err = s.store.LikedSet(viewer.ID, ids)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	notes := make([]string, len(rows))
	for i, r := range rows {
		notes[i] = r.Note
	}
	if s.convert == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	docs, err := s.convert.Convert(ctx, notes)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	out := make([]GalgameResource, 0, len(rows))
	for i, r := range rows {
		ref, ok := works[r.WorkID]
		if !ok {
			continue
		}
		author, ok := users[r.UserID]
		if !ok {
			author = userclient.User{ID: r.UserID}
		}
		work := ref
		item := fromRow(r, author, &work, providers[r.ID], docs[i], s.dlsiteFrom(ctx, catRows, r.WorkID), nil, s.cdn)
		if viewer != nil {
			item.Viewer = &GalgameResourceViewer{
				HasLiked:  liked[r.ID],
				CanEdit:   canEditResource(r.UserID, viewer),
				CanDelete: canDeleteResource(r.UserID, viewer),
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) dlsiteFrom(ctx context.Context, rows map[int]client.CatalogWorkListItem, workID int) *DlsiteOffer {
	if s.storeLinks == nil {
		return nil
	}
	it, ok := rows[workID]
	if !ok {
		return nil
	}
	brief := client.CatalogItemToBrief(ctx, &it)
	return dlsiteOf(s.storeLinks.Resolve(workID, brief.DlsiteWorkno()))
}

func (s *Service) one(ctx context.Context, row *model.GalgameResource, viewer *middleware.UserInfo) (GalgameResource, *problem.Problem) {
	items, p := s.assemble(ctx, []model.GalgameResource{*row}, viewer)
	if p != nil {
		return GalgameResource{}, p
	}
	if len(items) == 0 {
		return GalgameResource{}, notFound()
	}
	return items[0], nil
}
