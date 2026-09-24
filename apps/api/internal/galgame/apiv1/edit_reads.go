package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sort"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

var (
	editFieldTypes   = []string{"text", "i18nmap", "enum", "int", "date", "list", "ref", "imagehash"}
	editDiffHints    = []string{"inline", "lines", "items", "image"}
	editElementTypes = []string{"object", "text", "ref"}
	editMemberTypes  = []string{"text", "int", "enum", "bool", "ref", "imagehash"}
)

// revisionWalkMax bounds one work's chain read; production's longest is 15.
const revisionWalkMax = 2000

func (s *Service) getWorkEditForm(ctx context.Context, in *editWorkInput) (*editFormOutput, error) {
	cat, p := s.editing()
	if p != nil {
		return nil, p
	}
	if _, p := s.requireActive(ctx); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.editWork(ctx, workID); p != nil {
		return nil, p
	}
	values, err := cat.EditSnapshotUser(ctx, token, catalogclient.EntityTypeWork, int64(workID))
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	schema, err := cat.GetEditSchemaUser(ctx, token, catalogclient.EntityTypeWork, int64(workID))
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	vocab, err := cat.Vocabularies(ctx)
	if err != nil {
		slog.Warn("galgame edit: vocabularies unreadable", "work_id", workID, "err", err)
		vocab = nil
	}
	fields := editFieldsOf(workID, schema.Fields)
	return &editFormOutput{Body: EditForm{
		Object: "edit_form", WorkID: repr.ID(workID), FieldValues: nonNilMap(values),
		Fields: fields, Vocabularies: editVocabulariesOf(fields, vocab),
	}}, nil
}

func editFieldsOf(workID int, in []catalogclient.EditSchemaField) []EditField {
	out := make([]EditField, 0, len(in))
	for _, f := range in {
		field, bad := editFieldOf(f)
		if bad != "" {
			slog.Warn("galgame edit: dropping schema field outside the vocabulary",
				"work_id", workID, "field", f.Key, "attribute", bad)
			continue
		}
		out = append(out, field)
	}
	return out
}

func editFieldOf(f catalogclient.EditSchemaField) (EditField, string) {
	if !slices.Contains(editFieldTypes, f.Kind) {
		return EditField{}, "field_type=" + f.Kind
	}
	if !slices.Contains(editDiffHints, f.DiffHint) {
		return EditField{}, "diff_hint=" + f.DiffHint
	}
	out := EditField{
		Key: f.Key, FieldType: f.Kind, DiffHint: f.DiffHint, IsDeprecated: f.Deprecated,
		MaxElements: max(f.MaxElements, 0), MaxSuppressed: max(f.MaxSuppressed, 0),
		Vocabulary: f.Vocabulary, Base: max(f.Base, 0), IsNullable: f.Nullable,
	}
	switch f.Encoding {
	case "":
	case "token", "int":
		enc := f.Encoding
		out.Encoding = &enc
	default:
		return EditField{}, "encoding=" + f.Encoding
	}
	if f.Element != nil {
		if !slices.Contains(editElementTypes, f.Element.Type) {
			return EditField{}, "element.type=" + f.Element.Type
		}
		el := &EditFieldElement{ElementType: f.Element.Type, Members: make([]EditFieldElementMember, 0, len(f.Element.Members))}
		for _, m := range f.Element.Members {
			if !slices.Contains(editMemberTypes, m.Type) {
				return EditField{}, "element.members." + m.Key + ".type=" + m.Type
			}
			el.Members = append(el.Members, EditFieldElementMember{
				Key: m.Key, MemberType: m.Type, Vocabulary: m.Vocabulary, Base: max(m.Base, 0), IsNullable: m.Nullable,
			})
		}
		out.Element = el
	}
	return out, ""
}

func editVocabulariesOf(fields []EditField, vocab map[string]catalogclient.Vocabulary) []EditVocabulary {
	named := map[string]bool{}
	for _, f := range fields {
		named[f.Vocabulary] = true
		if f.Element != nil {
			for _, m := range f.Element.Members {
				named[m.Vocabulary] = true
			}
		}
	}
	out := make([]EditVocabulary, 0)
	for name, v := range vocab {
		if name == "" || !named[name] {
			continue
		}
		values := make([]EditVocabularyValue, 0, len(v.Values))
		for _, it := range v.Values {
			values = append(values, EditVocabularyValue{Value: it.Value, DisplayName: it.DisplayName, Description: it.Description})
		}
		out = append(out, EditVocabulary{Object: "vocabulary", Vocabulary: name, IsClosed: v.Closed, Values: values})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Vocabulary < out[j].Vocabulary })
	return out
}

func (s *Service) listWorkEditProposals(ctx context.Context, in *listWorkEditProposalsInput) (*editProposalListOutput, error) {
	cat, p := s.editing()
	if p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.editWork(ctx, workID); p != nil {
		return nil, p
	}
	fp := collect.Fingerprint(in.WorkID, in.State)
	upstream, p := unwrapUpstreamCursor(in.Cursor, "work_edit_proposals", fp)
	if p != nil {
		return nil, p
	}
	page, err := cat.ListPublicProposalsPage(ctx, catalogclient.EditProposalFilter{
		EntityType: catalogclient.EntityTypeWork, EntityID: int64(workID), Site: catalogSiteKungal,
		Status: in.State, Cursor: upstream, Limit: in.Limit,
	})
	if err != nil {
		return nil, mapAppPlane(err)
	}
	items, p := s.editProposalSummaries(ctx, v1.User(ctx), page.Items)
	if p != nil {
		return nil, p
	}
	return &editProposalListOutput{Body: repr.NewList(items, wrapUpstreamCursor("work_edit_proposals", fp, page.NextCursor))}, nil
}

func (s *Service) listWorkEditRevisions(ctx context.Context, in *listWorkEditRevisionsInput) (*editRevisionListOutput, error) {
	cat, p := s.editing()
	if p != nil {
		return nil, p
	}
	if p := in.CheckDepth(); p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.editWork(ctx, workID); p != nil {
		return nil, p
	}
	all, err := cat.ListEditRevisions(ctx, catalogclient.EntityTypeWork, int64(workID), revisionWalkMax)
	if err != nil {
		return nil, mapAppPlane(err)
	}
	kept := keptRevisions(workID, all)
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].Seq > kept[j].Seq })
	total, relation := collect.ClampTotal(len(kept))
	if len(all) >= revisionWalkMax {
		relation = "gte"
	}
	start := min(in.Offset(), len(kept))
	end := min(start+in.Limit, len(kept))
	pageRows := kept[start:end]
	refs, p := s.lookupUserRefs(ctx, revisionUserIDs(pageRows))
	if p != nil {
		return nil, p
	}
	items := make([]EditRevision, 0, len(pageRows))
	for _, r := range pageRows {
		items = append(items, editRevisionOf(r, refs))
	}
	return &editRevisionListOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

func (s *Service) getWorkEditRevisionDiff(ctx context.Context, in *workEditRevisionDiffInput) (*editRevisionDiffOutput, error) {
	cat, p := s.editing()
	if p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.editWork(ctx, workID); p != nil {
		return nil, p
	}
	diff, err := cat.DiffEditRevisions(ctx, catalogclient.EntityTypeWork, int64(workID), in.FromSeq, in.ToSeq)
	if err != nil {
		return nil, mapAppPlane(err)
	}
	changes := make([]EditFieldChange, 0, len(diff.Fields))
	for _, f := range diff.Fields {
		changes = append(changes, EditFieldChange{Key: f.Key, From: f.From, To: f.To})
	}
	return &editRevisionDiffOutput{Body: EditRevisionDiff{
		Object: "edit_revision_diff", FromSeq: in.FromSeq, ToSeq: in.ToSeq, FieldChanges: changes,
	}}, nil
}

func (s *Service) listMyEditProposals(ctx context.Context, in *listMyEditProposalsInput) (*editProposalListOutput, error) {
	cat, p := s.editing()
	if p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	var workID int64
	if in.WorkID != "" {
		id, ok := parseWorkID(in.WorkID)
		if !ok {
			return nil, invalidParameter(problem.AtParameter("work_id", problem.ReasonInvalidFormat, "a positive decimal id", nil))
		}
		workID = int64(id)
	}
	fp := collect.Fingerprint(strconv.Itoa(user.ID), in.WorkID, in.State)
	upstream, p := unwrapUpstreamCursor(in.Cursor, "my_edit_proposals", fp)
	if p != nil {
		return nil, p
	}
	page, err := cat.ListEditProposalsUserPage(ctx, token, catalogclient.UserEditProposalFilter{
		EntityType: catalogclient.EntityTypeWork, EntityID: workID, Status: in.State,
		Cursor: upstream, Limit: in.Limit, Mine: true,
	})
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	items, p := s.editProposalSummaries(ctx, user, page.Items)
	if p != nil {
		return nil, p
	}
	return &editProposalListOutput{Body: repr.NewList(items, wrapUpstreamCursor("my_edit_proposals", fp, page.NextCursor))}, nil
}

// catalog's moderation queue serves open proposals only and ignores state=, so
// the other states come from the public face; asking moderation for them
// answered open rows under a merged tab.
func (s *Service) listEditProposals(ctx context.Context, in *listEditProposalsInput) (*editProposalListOutput, error) {
	cat, p := s.editing()
	if p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	if !user.Can(perm.GalgameEditProposalReview) {
		return nil, permissionRequired()
	}
	fp := collect.Fingerprint(in.State)
	upstream, p := unwrapUpstreamCursor(in.Cursor, "edit_proposal_queue", fp)
	if p != nil {
		return nil, p
	}
	var page *catalogclient.ProposalPage
	var err error
	if in.State == "open" {
		token, p := requireToken(ctx)
		if p != nil {
			return nil, p
		}
		page, err = cat.ListEditProposalsUserPage(ctx, token, catalogclient.UserEditProposalFilter{
			EntityType: catalogclient.EntityTypeWork, Cursor: upstream, Limit: in.Limit,
		})
		if err != nil {
			return nil, mapUserPlane(err, true)
		}
	} else {
		page, err = cat.ListPublicProposalsPage(ctx, catalogclient.EditProposalFilter{
			EntityType: catalogclient.EntityTypeWork, Site: catalogSiteKungal, Status: in.State,
			Cursor: upstream, Limit: in.Limit,
		})
		if err != nil {
			return nil, mapAppPlane(err)
		}
	}
	items, p := s.editProposalSummaries(ctx, user, page.Items)
	if p != nil {
		return nil, p
	}
	return &editProposalListOutput{Body: repr.NewList(items, wrapUpstreamCursor("edit_proposal_queue", fp, page.NextCursor))}, nil
}

func (s *Service) getEditProposal(ctx context.Context, in *editProposalIDInput) (*editProposalOutput, error) {
	cat, p := s.editing()
	if p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	id, ok := parseWorkID(in.ProposalID)
	if !ok {
		return nil, notFound()
	}
	prop, etag, p := readProposal(ctx, cat, user, token, int64(id))
	if p != nil {
		return nil, p
	}
	body, p := s.editProposalBody(ctx, user, prop, false)
	if p != nil {
		return nil, p
	}
	return &editProposalOutput{ETag: etag, Body: body}, nil
}

// readProposal is the workbench read: the proposer's own face first, then the
// review face, where catalog checks standing (permission or ownership) itself.
// A Bearer caller never takes the second step. Every refusal reads as absence.
func readProposal(ctx context.Context, cat editCatalog, user *middleware.UserInfo, token string, id int64) (*catalogclient.EditProposal, string, *problem.Problem) {
	prop, etag, err := cat.GetMyProposal(ctx, token, id)
	if err != nil && errors.Is(err, catalogclient.ErrNotFound) && !user.ViaBearer() {
		prop, etag, err = cat.GetModerationProposal(ctx, token, id)
	}
	if err != nil {
		return nil, "", asProblem(mapUserPlane(err, false))
	}
	if !tenantProposal(prop) {
		slog.Warn("galgame edit: proposal outside the kungal work tenant", "proposal_id", id, "site", prop.Site, "entity_type", prop.EntityType)
		return nil, "", notFound()
	}
	return prop, etag, nil
}

func asProblem(err error) *problem.Problem {
	var p *problem.Problem
	if errors.As(err, &p) {
		return p
	}
	return problem.Internal(err)
}
