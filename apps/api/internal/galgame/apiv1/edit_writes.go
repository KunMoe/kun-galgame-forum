package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"kun-galgame-api/internal/constants"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

func noteText(note *string) string {
	if note == nil {
		return ""
	}
	return *note
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func (s *Service) createWorkEditProposal(ctx context.Context, in *createWorkEditProposalInput) (*createWorkEditProposalOutput, error) {
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
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.editWork(ctx, workID); p != nil {
		return nil, p
	}
	if len(in.Body.Patch) == 0 {
		return nil, validationFailed(problem.AtPointer("/patch", problem.ReasonRequired, "name at least one field", nil))
	}
	if p := checkFieldKeys("patch", mapKeys(in.Body.Patch)); p != nil {
		return nil, p
	}
	prop, _, err := cat.CreateMyProposal(ctx, token, catalogclient.UserEditCreateRequest{
		EntityType: catalogclient.EntityTypeWork, EntityID: int64(workID),
		Patch: in.Body.Patch, Note: noteText(in.Body.Note),
	}, idempotencyKey(ctx))
	if err != nil {
		return nil, mapUserPlaneAt(err, true, proposalPointer)
	}
	var revision *EditRevision
	switch prop.Status {
	case "merged":
		revision = s.mergedRevision(ctx, cat, workID, prop.ID)
	case "open":
		s.proposalFiled(ctx, prop, workID)
	}
	body, p := s.editProposalBody(ctx, user, prop, true)
	if p != nil {
		return nil, p
	}
	return &createWorkEditProposalOutput{
		Location: editProposalLocation + strconv.FormatInt(prop.ID, 10),
		Body:     WorkEditProposalCreateResult{EditProposal: body, Revision: revision},
	}, nil
}

func (s *Service) proposalFiled(ctx context.Context, prop *catalogclient.EditProposal, workID int) {
	owner, p := s.localCreator(workID)
	if p != nil {
		slog.Warn("galgame edit: requested notification failed", "proposal_id", prop.ID, "err", p)
	} else if owner > 0 && s.notifier != nil {
		if err := s.notifier.Emit(nil, msgService.Spec{
			SenderID: int(prop.ProposerUID), ReceiverID: owner,
			Kind: msgService.NotifyRequested, Content: s.workName(ctx, workID), WorkID: workID,
		}); err != nil {
			slog.Warn("galgame edit: requested notification failed", "proposal_id", prop.ID, "err", err)
		}
	}
	if s.store != nil && s.store.Ready() {
		if err := s.store.InsertProposalActivity(prop.ID, workID, int(prop.ProposerUID)); err != nil {
			slog.Warn("galgame edit: activity timeline write failed", "proposal_id", prop.ID, "err", err)
		}
	}
}

func (s *Service) createWorkEditRevert(ctx context.Context, in *createWorkEditRevertInput) (*createWorkEditRevertOutput, error) {
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
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.editWork(ctx, workID); p != nil {
		return nil, p
	}
	revisionID, err := cat.RevisionIDBySeq(ctx, catalogclient.EntityTypeWork, int64(workID), in.Body.ToSeq)
	if err != nil {
		return nil, mapAppPlane(err)
	}
	prop, _, err := cat.RevertToRevision(ctx, token, revisionID, noteText(in.Body.Note), idempotencyKey(ctx))
	if err != nil {
		return nil, mapUserPlaneAt(err, true, revertPointer)
	}
	var revision *EditRevision
	if prop.Status == "merged" {
		revision = s.mergedRevision(ctx, cat, workID, prop.ID)
	}
	body, p := s.editProposalBody(ctx, user, prop, true)
	if p != nil {
		return nil, p
	}
	return &createWorkEditRevertOutput{
		Location: editProposalLocation + strconv.FormatInt(prop.ID, 10),
		Body:     EditRevert{Object: "edit_revert", Proposal: body, Revision: revision},
	}, nil
}

func (s *Service) createEditProposalAmendment(ctx context.Context, in *createEditProposalAmendmentInput) (*createEditProposalAmendmentOutput, error) {
	cat, p := s.editing()
	if p != nil {
		return nil, p
	}
	caller, p := s.requireActive(ctx)
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
	if len(in.Body.Set) == 0 && len(in.Body.Unset) == 0 {
		return nil, validationFailed(problem.AtPointer("/set", problem.ReasonRequired, "set or unset must name at least one field", nil))
	}
	if p := checkFieldKeys("set", mapKeys(in.Body.Set)); p != nil {
		return nil, p
	}
	unset := make([]string, 0, len(in.Body.Unset))
	for _, k := range in.Body.Unset {
		unset = append(unset, string(k))
	}
	if p := checkFieldKeys("unset", unset); p != nil {
		return nil, p
	}
	prop, etag, err := cat.AmendMyProposal(ctx, token, int64(id), in.Body.Set, unset,
		noteText(in.Body.Note), in.IfMatch, idempotencyKey(ctx))
	if err != nil {
		return nil, mapUserPlaneAt(err, true, amendmentPointer)
	}
	latest := latestAmendment(prop)
	if latest == nil {
		if again, _, p := readProposal(ctx, cat, caller, token, int64(id)); p == nil {
			latest = latestAmendment(again)
		}
	}
	if latest == nil {
		return nil, problem.Internal(errors.New("catalog answered an amendment with no amendment chain"))
	}
	refs := s.userRefsAfterWrite(ctx, []int{int(latest.AmenderUID)})
	return &createEditProposalAmendmentOutput{
		Location: editProposalLocation + strconv.Itoa(id),
		ETag:     etag,
		Body:     editAmendmentOf(*latest, refs),
	}, nil
}

func (s *Service) updateEditProposal(ctx context.Context, in *updateEditProposalInput) (*editProposalOutput, error) {
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
	decision := in.Body.State != "withdrawn"
	if decision && user.ViaBearer() {
		return nil, permissionRequired()
	}
	current, err := cat.GetPublicProposal(ctx, int64(id))
	if err != nil {
		return nil, mapAppPlane(err)
	}
	if !tenantProposal(current) {
		return nil, notFound()
	}
	workID := int(current.EntityID)
	if current.Status != "open" {
		return nil, invalidTransition("proposal", current.Status, nil)
	}
	if decision {
		owner, p := s.localCreator(workID)
		if p != nil {
			return nil, p
		}
		if !editStanding(user, owner) {
			return nil, permissionRequired()
		}
	} else if int(current.ProposerUID) != user.ID {
		return nil, permissionRequired()
	}
	note := noteText(in.Body.Note)
	if in.Body.State == "declined" && trimmedNote(in.Body.Note) == "" {
		return nil, noteRequired()
	}

	written := *current
	switch in.Body.State {
	case "withdrawn":
		after, _, err := cat.WithdrawMyProposal(ctx, token, int64(id), in.IfMatch)
		if err != nil {
			return nil, mapUserPlane(err, true)
		}
		written = *after
	case "merged", "declined":
		verb := map[string]string{"merged": "merge", "declined": "decline"}[in.Body.State]
		outcome, err := cat.DecideProposal(ctx, token, int64(id), verb, note, in.IfMatch)
		if err != nil {
			return nil, mapUserPlaneAt(err, true, proposalPointer)
		}
		if outcome == "" {
			outcome = in.Body.State
		}
		now := time.Now().UTC()
		decider := int64(user.ID)
		written.Status, written.DecidedByUID, written.DecidedAt = outcome, &decider, &now
		switch outcome {
		case "merged":
			s.proposalMerged(ctx, current, workID, user.ID)
		case "declined":
			s.notifyDecision(ctx, current, workID, user.ID, msgService.NotifyDeclined, s.declineContent(ctx, workID, note))
		}
	}

	read := cat.GetModerationProposal
	if !decision {
		read = cat.GetMyProposal
	}
	after, etag, err := read(ctx, token, int64(id))
	if err != nil {
		slog.Warn("galgame edit: proposal read-back failed after the write", "proposal_id", id, "state", written.Status, "err", err)
		after, etag = &written, ""
	}
	body, p := s.editProposalBody(ctx, user, after, true)
	if p != nil {
		return nil, p
	}
	return &editProposalOutput{ETag: etag, Body: body}, nil
}

func latestAmendment(p *catalogclient.EditProposal) *catalogclient.EditAmendment {
	var latest *catalogclient.EditAmendment
	for i := range p.Amendments {
		if latest == nil || p.Amendments[i].Seq > latest.Seq {
			latest = &p.Amendments[i]
		}
	}
	return latest
}

func (s *Service) declineContent(ctx context.Context, workID int, note string) string {
	if name := s.workName(ctx, workID); name != "" {
		return name + "：" + note
	}
	return note
}

func (s *Service) proposalMerged(ctx context.Context, prop *catalogclient.EditProposal, workID, mergerID int) {
	if int(prop.ProposerUID) != mergerID && s.award != nil {
		s.award(int(prop.ProposerUID), constants.RewardPRMerge, moemoepoint.ReasonContentApproved,
			moemoepoint.Ref("galgame_pr", workID),
			moemoepoint.Key("galgame_edit_merged", strconv.FormatInt(prop.ID, 10)))
	}
	if s.store != nil && s.store.Ready() {
		if err := s.store.TouchEdited(workID); err != nil {
			slog.Warn("galgame edit: resource_update_time bump failed", "work_id", workID, "err", err)
		}
	}
	s.notifyDecision(ctx, prop, workID, mergerID, msgService.NotifyMerged, s.workName(ctx, workID))
}

func (s *Service) notifyDecision(_ context.Context, prop *catalogclient.EditProposal, workID, senderID int, kind msgService.NotifyKind, content string) {
	if s.notifier == nil {
		return
	}
	if err := s.notifier.Emit(nil, msgService.Spec{
		SenderID: senderID, ReceiverID: int(prop.ProposerUID),
		Kind: kind, Content: content, WorkID: workID,
	}); err != nil {
		slog.Warn("galgame edit: decision notification failed", "proposal_id", prop.ID, "kind", kind, "err", err)
	}
}
