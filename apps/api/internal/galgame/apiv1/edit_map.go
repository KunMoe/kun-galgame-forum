package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

type editCatalog interface {
	CreateMyProposal(ctx context.Context, token string, req catalogclient.UserEditCreateRequest, idempotencyKey string) (*catalogclient.EditProposal, string, error)
	GetMyProposal(ctx context.Context, token string, id int64) (*catalogclient.EditProposal, string, error)
	GetModerationProposal(ctx context.Context, token string, id int64) (*catalogclient.EditProposal, string, error)
	GetPublicProposal(ctx context.Context, id int64) (*catalogclient.EditProposal, error)
	WithdrawMyProposal(ctx context.Context, token string, id int64, ifMatch string) (*catalogclient.EditProposal, string, error)
	AmendMyProposal(ctx context.Context, token string, id int64, set map[string]any, unset []string, note, ifMatch, idempotencyKey string) (*catalogclient.EditProposal, string, error)
	DecideProposal(ctx context.Context, token string, id int64, decision, note, ifMatch string) (string, error)
	RevertToRevision(ctx context.Context, token string, revisionID int64, note, idempotencyKey string) (*catalogclient.EditProposal, string, error)
	ListEditProposalsUserPage(ctx context.Context, token string, f catalogclient.UserEditProposalFilter) (*catalogclient.ProposalPage, error)
	ListPublicProposalsPage(ctx context.Context, f catalogclient.EditProposalFilter) (*catalogclient.ProposalPage, error)
	ListEditRevisions(ctx context.Context, entityType string, entityID int64, limit int) ([]catalogclient.EditRevision, error)
	DiffEditRevisions(ctx context.Context, entityType string, entityID int64, fromSeq, toSeq int) (*catalogclient.EditDiff, error)
	RevisionIDBySeq(ctx context.Context, entityType string, entityID int64, seq int) (int64, error)
	GetEditSchemaUser(ctx context.Context, token, entityType string, entityID int64) (*catalogclient.EditSchema, error)
	EditSnapshotUser(ctx context.Context, token, entityType string, entityID int64) (map[string]any, error)
	Vocabularies(ctx context.Context) (map[string]catalogclient.Vocabulary, error)
}

// editing() asserts this at runtime, so a drifted signature would compile and
// then answer 500 on every edit face.
var _ editCatalog = (*catalogclient.Client)(nil)

func (s *Service) WithEditing(n msgService.Notifier) *Service {
	s.notifier = n
	return s
}

func (s *Service) editing() (editCatalog, *problem.Problem) {
	if s == nil || s.catalog == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	c, ok := s.catalog.(editCatalog)
	if !ok {
		return nil, problem.Internal(errUnconfigured)
	}
	return c, nil
}

// editWork is G4's work visibility (hidden or unknown is NOT_FOUND) plus the
// merged redirect.
func (s *Service) editWork(ctx context.Context, workID int) (*client.CatalogWorkListItem, *problem.Problem) {
	row, p := s.catalogWork(ctx, workID)
	if p == nil {
		return row, nil
	}
	if p.Code != problem.CodeNotFound {
		return nil, p
	}
	if d := s.detailCatalog(); d != nil {
		_, _, movedTo, appErr := d.CatalogWorkDetail(ctx, workID)
		if appErr != nil {
			return nil, catalogUnavailable(appErr)
		}
		if movedTo != 0 {
			return nil, mergedWork(movedTo)
		}
	}
	return nil, p
}

func (s *Service) workName(ctx context.Context, workID int) string {
	if s.works == nil {
		return ""
	}
	rows, appErr := s.works.CatalogRowsByWorkIDs(ctx, []int{workID}, "names", "all")
	if appErr != nil {
		slog.Warn("galgame edit: work name unreadable", "work_id", workID, "err", appErr)
		return ""
	}
	row, ok := rows[workID]
	if !ok {
		return ""
	}
	brief := client.CatalogItemToBrief(ctx, &row)
	return client.BriefName(&brief)
}

// The revision and proposal listings are read with the forum's own application
// key, not the reader's token, so a 401/403 here is this deployment's key being
// wrong and never the user's grant. Passed on as INVALID_CREDENTIAL it would
// prompt a perfectly valid session to log out and back in, which cannot fix it.
func mapAppPlane(err error) error {
	var api *catalogclient.UserAPIError
	switch {
	case errors.Is(err, catalogclient.ErrUnauthorized), errors.Is(err, catalogclient.ErrInsufficientScope),
		errors.As(err, &api) && (api.Status == http.StatusUnauthorized || api.Status == http.StatusForbidden):
		slog.Error("galgame edit: catalog refused the application key", "err", err)
		return problem.Unavailable(err)
	}
	return mapUserPlane(err, false)
}

func proposalPointer(upstream string) (string, bool) {
	switch {
	case strings.HasPrefix(upstream, "/patch/"), upstream == "/note":
		return upstream, true
	}
	return "", false
}

func amendmentPointer(upstream string) (string, bool) {
	switch {
	case strings.HasPrefix(upstream, "/set/"), strings.HasPrefix(upstream, "/unset"), upstream == "/note":
		return upstream, true
	case strings.HasPrefix(upstream, "/patch/"):
		return "/set/" + strings.TrimPrefix(upstream, "/patch/"), true
	}
	return "", false
}

func revertPointer(upstream string) (string, bool) {
	switch upstream {
	case "/reason":
		return "/note", true
	case "/revision_id":
		return "/to_seq", true
	}
	return "", false
}

func pointerKey(key string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(key)
}

func checkFieldKeys(member string, keys []string) *problem.Problem {
	var fields []problem.FieldError
	for _, key := range keys {
		if !strings.HasPrefix(key, catalogclient.FieldKeyPrefix) {
			fields = append(fields, problem.AtPointer("/"+member+"/"+pointerKey(key), problem.ReasonNotAllowedValue,
				"field keys start with "+catalogclient.FieldKeyPrefix, nil))
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return validationFailed(fields...)
}

var editProposalStates = map[string]bool{"open": true, "merged": true, "declined": true, "withdrawn": true}

var editRevisionActions = map[string]bool{"created": true, "merged": true, "direct": true, "reverted": true}

func tenantProposal(p *catalogclient.EditProposal) bool {
	return p != nil && p.Site == catalogSiteKungal && p.EntityType == catalogclient.EntityTypeWork
}

func proposalUserIDs(props ...*catalogclient.EditProposal) []int {
	var ids []int
	for _, p := range props {
		ids = append(ids, int(p.ProposerUID))
		if p.DecidedByUID != nil {
			ids = append(ids, int(*p.DecidedByUID))
		}
		for _, a := range p.Amendments {
			ids = append(ids, int(a.AmenderUID))
		}
	}
	return ids
}

// editStanding is review standing as this site can see it: the permission, or
// owning the work locally. catalog's owner is catalog_work.owner_user_id, which
// no v2 face exposes, so a disagreement ends in catalog's 403.
func editStanding(user *middleware.UserInfo, owner int) bool {
	if user == nil || user.ViaBearer() {
		return false
	}
	return user.Can(perm.GalgameEditProposalReview) || (owner > 0 && owner == user.ID)
}

func proposalViewer(user *middleware.UserInfo, p *catalogclient.EditProposal, owner int) *EditProposalViewer {
	if user == nil {
		return nil
	}
	open := p.Status == "open"
	isProposer := int(p.ProposerUID) == user.ID
	standing := editStanding(user, owner)
	return &EditProposalViewer{
		IsProposer:  isProposer,
		CanWithdraw: open && isProposer,
		CanDecide:   open && standing,
		CanAmend:    open && (isProposer || standing),
	}
}

func (s *Service) editOwners(user *middleware.UserInfo, workIDs []int) (map[int]int, *problem.Problem) {
	if user == nil || user.Can(perm.GalgameEditProposalReview) || s.store == nil || !s.store.Ready() {
		return map[int]int{}, nil
	}
	owners, err := s.store.CreatorsOf(workIDs)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return owners, nil
}

func amendmentsOf(p *catalogclient.EditProposal, refs map[int]repr.UserRef) []EditAmendment {
	out := make([]EditAmendment, 0, len(p.Amendments))
	for _, a := range p.Amendments {
		out = append(out, editAmendmentOf(a, refs))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}

func editAmendmentOf(a catalogclient.EditAmendment, refs map[int]repr.UserRef) EditAmendment {
	return EditAmendment{
		Object: "edit_amendment", ID: repr.ID(int(a.ID)), Seq: a.Seq,
		Amender: userRefOf(refs, int(a.AmenderUID)), Note: optionalNote(a.Note),
		CreatedAt: repr.Timestamp(a.CreatedAt),
	}
}

func nonNilMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func decidedAt(p *catalogclient.EditProposal) *repr.DateTime {
	if p.DecidedAt == nil || p.DecidedAt.IsZero() {
		return nil
	}
	t := repr.Timestamp(*p.DecidedAt)
	return &t
}

func deciderOf(p *catalogclient.EditProposal, refs map[int]repr.UserRef) *repr.UserRef {
	if p.DecidedByUID == nil {
		return nil
	}
	return userRefPtr(refs, int(*p.DecidedByUID))
}

func editProposalOf(p *catalogclient.EditProposal, refs map[int]repr.UserRef, work workrepr.WorkSummary, viewer *EditProposalViewer) EditProposal {
	return EditProposal{
		Object: "edit_proposal", ID: repr.ID(int(p.ID)), State: p.Status,
		WorkID: repr.ID(int(p.EntityID)), WorkSummary: work, Note: optionalNote(p.Note),
		Proposer: userRefOf(refs, int(p.ProposerUID)), Decider: deciderOf(p, refs),
		BaseRevisionSeq: max(p.BaseRevisionSeq, 0),
		Patch:           nonNilMap(p.Patch), EffectivePatch: nonNilMap(p.EffectivePatch),
		Amendments: amendmentsOf(p, refs),
		CreatedAt:  repr.Timestamp(p.CreatedAt), UpdatedAt: repr.Timestamp(p.UpdatedAt),
		DecidedAt: decidedAt(p), Viewer: viewer,
	}
}

func editProposalSummaryOf(p *catalogclient.EditProposal, refs map[int]repr.UserRef, work workrepr.WorkSummary, viewer *EditProposalViewer) EditProposalSummary {
	return EditProposalSummary{
		Object: "edit_proposal", ID: repr.ID(int(p.ID)), State: p.Status,
		WorkID: repr.ID(int(p.EntityID)), WorkSummary: work, Note: optionalNote(p.Note),
		Proposer: userRefOf(refs, int(p.ProposerUID)), Decider: deciderOf(p, refs),
		BaseRevisionSeq: max(p.BaseRevisionSeq, 0),
		CreatedAt:       repr.Timestamp(p.CreatedAt), UpdatedAt: repr.Timestamp(p.UpdatedAt),
		DecidedAt: decidedAt(p), Viewer: viewer,
	}
}

func workSummaryOr(works map[int]workrepr.WorkSummary, workID int) workrepr.WorkSummary {
	if w, ok := works[workID]; ok {
		return w
	}
	return emptyWorkSummary(workID, "")
}

// keptProposals drops what the v1 vocabulary cannot carry. An empty state used
// to be read as open, which put decided proposals back in the review queue.
func keptProposals(items []catalogclient.EditProposal) []*catalogclient.EditProposal {
	out := make([]*catalogclient.EditProposal, 0, len(items))
	for i := range items {
		p := &items[i]
		switch {
		case !tenantProposal(p):
			slog.Warn("galgame edit: dropping proposal outside the kungal work tenant",
				"proposal_id", p.ID, "site", p.Site, "entity_type", p.EntityType)
		case !editProposalStates[p.Status]:
			slog.Warn("galgame edit: dropping proposal with unknown state", "proposal_id", p.ID, "value", p.Status)
		default:
			out = append(out, p)
		}
	}
	return out
}

func (s *Service) editProposalSummaries(ctx context.Context, user *middleware.UserInfo, items []catalogclient.EditProposal) ([]EditProposalSummary, *problem.Problem) {
	kept := keptProposals(items)
	refs, p := s.lookupUserRefs(ctx, proposalUserIDs(kept...))
	if p != nil {
		return nil, p
	}
	var workIDs []int
	for _, it := range kept {
		if id := int(it.EntityID); !slices.Contains(workIDs, id) {
			workIDs = append(workIDs, id)
		}
	}
	works := s.workSummaries(ctx, workIDs)
	owners, p := s.editOwners(user, workIDs)
	if p != nil {
		return nil, p
	}
	out := make([]EditProposalSummary, 0, len(kept))
	for _, it := range kept {
		workID := int(it.EntityID)
		out = append(out, editProposalSummaryOf(it, refs, workSummaryOr(works, workID), proposalViewer(user, it, owners[workID])))
	}
	return out, nil
}

// editProposalBody renders one proposal read or written through the user plane.
// A state outside the vocabulary is a catalog contract break, not a row to drop.
// After a write catalog has already accepted, a failed user or owner lookup
// degrades the body instead of failing: a client retry would write again.
func (s *Service) editProposalBody(ctx context.Context, user *middleware.UserInfo, p *catalogclient.EditProposal, afterWrite bool) (EditProposal, *problem.Problem) {
	if !editProposalStates[p.Status] {
		slog.Error("galgame edit: catalog answered a proposal state outside the vocabulary", "proposal_id", p.ID, "value", p.Status)
		return EditProposal{}, problem.Internal(errors.New("unknown proposal state " + strconv.Quote(p.Status)))
	}
	workID := int(p.EntityID)
	var refs map[int]repr.UserRef
	owners, pr := s.editOwners(user, []int{workID})
	if afterWrite {
		refs = s.userRefsAfterWrite(ctx, proposalUserIDs(p))
		if pr != nil {
			slog.Warn("galgame edit: owner lookup failed after the write", "proposal_id", p.ID, "err", pr)
			owners = map[int]int{}
		}
	} else {
		if pr != nil {
			return EditProposal{}, pr
		}
		if refs, pr = s.lookupUserRefs(ctx, proposalUserIDs(p)); pr != nil {
			return EditProposal{}, pr
		}
	}
	works := s.workSummaries(ctx, []int{workID})
	return editProposalOf(p, refs, workSummaryOr(works, workID), proposalViewer(user, p, owners[workID])), nil
}

func (s *Service) userRefsAfterWrite(ctx context.Context, ids []int) map[int]repr.UserRef {
	refs, p := s.lookupUserRefs(ctx, ids)
	if p != nil {
		slog.Warn("galgame edit: user lookup failed after the write", "err", p)
		return map[int]repr.UserRef{}
	}
	return refs
}

func editRevisionOf(r catalogclient.EditRevision, refs map[int]repr.UserRef) EditRevision {
	out := EditRevision{
		Object: "edit_revision", ID: repr.ID(int(r.ID)), Seq: r.Seq, RevisionAction: r.Action,
		ChangedFields: make([]EditFieldKey, 0, len(r.ChangedFields)), Actor: userRefOf(refs, int(r.ActorUID)),
		CreatedAt: repr.Timestamp(r.CreatedAt),
	}
	for _, k := range r.ChangedFields {
		out.ChangedFields = append(out.ChangedFields, EditFieldKey(k))
	}
	if r.AmenderUID != nil {
		out.LastAmender = userRefPtr(refs, int(*r.AmenderUID))
	}
	if r.ProposalID != nil {
		id := repr.ID(int(*r.ProposalID))
		out.ProposalID = &id
	}
	return out
}

func keptRevisions(workID int, items []catalogclient.EditRevision) []catalogclient.EditRevision {
	out := make([]catalogclient.EditRevision, 0, len(items))
	for _, r := range items {
		if !editRevisionActions[r.Action] {
			slog.Warn("galgame edit: dropping revision with unknown action", "work_id", workID, "revision_id", r.ID, "value", r.Action)
			continue
		}
		out = append(out, r)
	}
	return out
}

func revisionUserIDs(revs []catalogclient.EditRevision) []int {
	var ids []int
	for _, r := range revs {
		ids = append(ids, int(r.ActorUID))
		if r.AmenderUID != nil {
			ids = append(ids, int(*r.AmenderUID))
		}
	}
	return ids
}

// mergedRevision runs after catalog has already written the proposal. Its
// failure must not fail the request: the client would retry and file again.
func (s *Service) mergedRevision(ctx context.Context, cat editCatalog, workID int, proposalID int64) *EditRevision {
	revs, err := cat.ListEditRevisions(ctx, catalogclient.EntityTypeWork, int64(workID), 100)
	if err != nil {
		slog.Warn("galgame edit: merged revision not found", "work_id", workID, "proposal_id", proposalID, "err", err)
		return nil
	}
	for _, r := range keptRevisions(workID, revs) {
		if r.ProposalID == nil || *r.ProposalID != proposalID {
			continue
		}
		refs, p := s.lookupUserRefs(ctx, revisionUserIDs([]catalogclient.EditRevision{r}))
		if p != nil {
			slog.Warn("galgame edit: merged revision users unreadable", "work_id", workID, "proposal_id", proposalID, "err", p)
			return nil
		}
		out := editRevisionOf(r, refs)
		return &out
	}
	slog.Warn("galgame edit: merged revision not found", "work_id", workID, "proposal_id", proposalID)
	return nil
}
