package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

type submissionCatalog interface {
	SubmitWorkUser(ctx context.Context, token string, req catalogclient.UserWorkSubmitRequest) (*catalogclient.WorkSubmitResult, error)
	GetMyClaim(ctx context.Context, token string, workID int64) (*catalogclient.UserClaimItem, string, error)
	GetModerationClaim(ctx context.Context, token string, workID int64) (*catalogclient.UserClaimItem, string, error)
	PatchMyClaim(ctx context.Context, token string, workID int64, state, ifMatch string) (*catalogclient.UserClaimItem, string, error)
	DecideClaim(ctx context.Context, token string, workID int64, decision, note, ifMatch string) error
	DeleteMyClaim(ctx context.Context, token string, workID int64, ifMatch string) error
	MyClaims(ctx context.Context, token string, f catalogclient.UserClaimFilter) (*catalogclient.UserClaimPage, error)
	ListModerationClaims(ctx context.Context, token string, f catalogclient.UserClaimFilter) (*catalogclient.UserClaimPage, error)
	CreateMyProposal(ctx context.Context, token string, req catalogclient.UserEditCreateRequest, idempotencyKey string) (*catalogclient.EditProposal, string, error)
	DecideProposal(ctx context.Context, token string, id int64, decision, note, ifMatch string) error
	EditSnapshotUser(ctx context.Context, token, entityType string, entityID int64) (map[string]any, error)
}

var claimStates = map[string]bool{
	"live": true, "draft": true, "pending": true, "declined": true, "hidden": true,
}

type submissionCall struct {
	user  *middleware.UserInfo
	token string
	cat   submissionCatalog
}

func (s *Service) submissionCall(ctx context.Context) (*submissionCall, *problem.Problem) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	cat, ok := s.catalog.(submissionCatalog)
	if !ok {
		return nil, problem.Unavailable(errUnconfigured)
	}
	return &submissionCall{user: user, token: token, cat: cat}, nil
}

func (s *Service) createWorkSubmission(ctx context.Context, in *createWorkSubmissionInput) (*createWorkSubmissionOutput, error) {
	c, p := s.submissionCall(ctx)
	if p != nil {
		return nil, p
	}
	mint, p := buildMint(in.Body)
	if p != nil {
		return nil, p
	}
	text := gate.ComposeText(mint.text...)
	if p := s.checkContributionText(ctx, text, c.user.ID); p != nil {
		return nil, p
	}
	res, err := c.cat.SubmitWorkUser(ctx, c.token, catalogclient.UserWorkSubmitRequest{
		Fields: mint.fields, Released: mint.released,
		ConfirmDuplicates: in.Body.IsDuplicateConfirmed, IdempotencyKey: idempotencyKey(ctx),
	})
	if err != nil {
		return nil, mapUserPlaneAt(err, true, submissionPointer)
	}
	workID := int(res.WorkID)
	if s.store.Ready() {
		if err := s.store.SubmitLocal(workID, c.user.ID); err != nil {
			slog.Warn("submit: failed to record local creator", "work_id", workID, "err", err)
		}
	}
	attached := in.Body.BannerHash == "" || s.attachBanner(ctx, c, res.WorkID, in.Body.BannerHash)
	if s.scan != nil {
		s.scan.ScanBg(gate.SubjectKindGalgame, string(repr.ID(workID)), text, int64(c.user.ID))
	}
	body, p := s.toWorkSubmission(ctx, c.user, res.Claim, true, submissionAxes{isNSFW: in.Body.IsNSFW, rating: in.Body.ContentRating})
	if p != nil {
		return nil, p
	}
	return &createWorkSubmissionOutput{
		Location: "/api/v1/work-submissions/" + string(body.ID),
		ETag:     res.ETag,
		Body:     WorkSubmissionCreated{WorkSubmission: body, HasBannerAttached: attached},
	}, nil
}

// Approving a submission moves the claim state and nothing else, so a banner
// proposal left open here was never decided by anyone and the cover never
// appeared. The submitter owns the work, so the proposal is merged here.
//
// A cover element takes image_hash, kind, portrait_pinned, sexual and violence
// and nothing else: the first version also sent sort_order, source and
// source_key, catalog refused every one, and no submission banner reached
// production before 2026-08.
func (s *Service) attachBanner(ctx context.Context, c *submissionCall, workID int64, hash string) bool {
	prop, etag, err := c.cat.CreateMyProposal(ctx, c.token, catalogclient.UserEditCreateRequest{
		EntityType: catalogclient.EntityTypeWork, EntityID: workID,
		Patch: map[string]any{"catalog.work.covers": []any{map[string]any{"image_hash": hash}}},
		Note:  "Banner uploaded with the submission",
	}, "")
	if err != nil {
		slog.Error("submit: banner attach failed, submission stands without cover", "work_id", workID, "err", err)
		return false
	}
	if prop.Status == "merged" {
		return true
	}
	if err := c.cat.DecideProposal(ctx, c.token, prop.ID, "merge", "", etag); err != nil {
		slog.Error("submit: banner proposal merge failed, submission stands without cover",
			"work_id", workID, "proposal_id", prop.ID, "err", err)
		return false
	}
	return true
}

func (s *Service) getWorkSubmission(ctx context.Context, in *workSubmissionIDInput) (*workSubmissionOutput, error) {
	c, p := s.submissionCall(ctx)
	if p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	item, etag, owner, err := s.readSubmission(ctx, c, int64(workID))
	if err != nil {
		return nil, err
	}
	return s.submissionResponse(ctx, c, item, etag, owner)
}

func (s *Service) readSubmission(ctx context.Context, c *submissionCall, workID int64) (*catalogclient.UserClaimItem, string, bool, error) {
	item, etag, err := c.cat.GetMyClaim(ctx, c.token, workID)
	if err == nil {
		return item, etag, true, nil
	}
	if !upstreamNotFound(err) || !c.user.Can(perm.GalgameClaimReview) {
		return nil, "", false, mapUserPlane(err, false)
	}
	item, etag, err = c.cat.GetModerationClaim(ctx, c.token, workID)
	if err != nil {
		return nil, "", false, mapUserPlane(err, false)
	}
	return item, etag, false, nil
}

func upstreamNotFound(err error) bool {
	if errors.Is(err, catalogclient.ErrNotFound) {
		return true
	}
	var api *catalogclient.UserAPIError
	if !errors.As(err, &api) {
		return false
	}
	return api.Status == http.StatusNotFound ||
		api.Status == http.StatusForbidden && (api.ProblemCode == "CLAIM_NOT_OWNED" || api.ProblemCode == "TENANT_MISMATCH")
}

func (s *Service) submissionResponse(ctx context.Context, c *submissionCall, item *catalogclient.UserClaimItem, etag string, owner bool) (*workSubmissionOutput, error) {
	axes, err := s.readSubmissionAxes(ctx, c, item.WorkID)
	if err != nil {
		return nil, err
	}
	body, p := s.toWorkSubmission(ctx, c.user, *item, owner, axes)
	if p != nil {
		return nil, p
	}
	return &workSubmissionOutput{ETag: etag, Body: body}, nil
}

var reviewerTargets = map[string]bool{"live": true, "declined": true, "hidden": true, "unban": true}

func (s *Service) updateWorkSubmission(ctx context.Context, in *updateWorkSubmissionInput) (*workSubmissionOutput, error) {
	c, p := s.submissionCall(ctx)
	if p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	id := int64(workID)
	target := in.Body.State
	if !reviewerTargets[target] {
		cur, _, err := c.cat.GetMyClaim(ctx, c.token, id)
		if err != nil {
			return nil, mapUserPlane(err, false)
		}
		upstream, allowed := submitterMove(cur.ClaimState, target)
		if upstream == "" {
			return nil, invalidTransition("claim", cur.ClaimState, allowed)
		}
		item, etag, err := c.cat.PatchMyClaim(ctx, c.token, id, upstream, ifMatchOrAny(in.IfMatch))
		if err != nil {
			return nil, mapUserPlane(err, true)
		}
		return s.submissionResponse(ctx, c, item, etag, true)
	}

	if !c.user.Can(perm.GalgameClaimReview) {
		return nil, permissionRequired()
	}
	note := trimmedNote(in.Body.Note)
	if target == "declined" && note == "" {
		return nil, noteRequired()
	}
	cur, _, err := c.cat.GetModerationClaim(ctx, c.token, id)
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	decision, allowed := reviewerDecision(cur.ClaimState, target)
	if decision == "" {
		return nil, invalidTransition("claim", cur.ClaimState, allowed)
	}
	if err := c.cat.DecideClaim(ctx, c.token, id, decision, note, ifMatchOrAny(in.IfMatch)); err != nil {
		return nil, mapUserPlane(err, true)
	}
	item, etag, err := c.cat.GetModerationClaim(ctx, c.token, id)
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	creator, p := s.localCreator(workID)
	if p != nil {
		return nil, p
	}
	return s.submissionResponse(ctx, c, item, etag, creator == c.user.ID)
}

func submitterMove(current, target string) (string, []string) {
	switch current {
	case "draft", "declined":
		if target == "pending" {
			return "pending", nil
		}
		return "", []string{"pending"}
	case "pending", "live":
		if target == "draft" {
			return "withdrawn", nil
		}
		return "", []string{"draft"}
	}
	return "", nil
}

func reviewerDecision(current, target string) (string, []string) {
	var allowed []string
	switch current {
	case "pending":
		allowed = []string{"live", "declined", "hidden"}
	case "live", "draft", "declined":
		allowed = []string{"hidden"}
	case "hidden":
		allowed = []string{"unban"}
	}
	for _, a := range allowed {
		if a != target {
			continue
		}
		switch target {
		case "live":
			return "approve", nil
		case "declined":
			return "decline", nil
		case "hidden":
			return "ban", nil
		case "unban":
			return "unban", nil
		}
	}
	return "", allowed
}

func (s *Service) deleteWorkSubmission(ctx context.Context, in *workSubmissionWriteInput) (*noContentOutput, error) {
	c, p := s.submissionCall(ctx)
	if p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	cur, _, err := c.cat.GetMyClaim(ctx, c.token, int64(workID))
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	if cur.ClaimState != "draft" {
		return nil, problem.New(problem.CodeInvalidStateTransition,
			"The claim is in state "+cur.ClaimState+"; only a draft can be deleted. Withdraw it to draft first.")
	}
	// Owner-only and draft-only are catalog's checks and stay catalog's: the
	// local delete below cascades, so it runs only after the authority agreed
	// this is a disposable draft.
	if err := c.cat.DeleteMyClaim(ctx, c.token, int64(workID), ifMatchOrAny(in.IfMatch)); err != nil {
		return nil, mapUserPlane(err, true)
	}
	if s.store.Ready() {
		if err := s.store.DeleteLocalDraft(workID); err != nil {
			slog.Error("delete draft: catalog draft gone, local row not cleaned", "work_id", workID, "err", err)
			return nil, problem.Internal(err)
		}
	}
	return &noContentOutput{}, nil
}

type submissionAxes struct {
	isNSFW bool
	rating string
}

// A hidden work is not in catalog's public rows at all, so the review face
// reads its two axes from the edit snapshot instead.
func (s *Service) readSubmissionAxes(ctx context.Context, c *submissionCall, workID int64) (submissionAxes, error) {
	if s.works == nil {
		return submissionAxes{}, problem.Unavailable(errUnconfigured)
	}
	rows, appErr := s.works.CatalogRowsByWorkIDs(ctx, []int{int(workID)}, "names", "all")
	if appErr != nil {
		return submissionAxes{}, catalogUnavailable(appErr)
	}
	if row, ok := rows[int(workID)]; ok {
		return submissionAxes{
			isNSFW: client.CatalogItemToBrief(ctx, &row).ContentLimit == "nsfw",
			rating: contentRatingOf(row.ContentRating),
		}, nil
	}
	snap, err := c.cat.EditSnapshotUser(ctx, c.token, catalogclient.EntityTypeWork, workID)
	if err != nil {
		return submissionAxes{}, mapUserPlane(err, false)
	}
	nsfw, _ := snap["catalog.work.display_nsfw"].(bool)
	code, _ := snap["catalog.work.content_rating"].(float64)
	return submissionAxes{isNSFW: nsfw, rating: ratingFromCode(int(code))}, nil
}

func ratingFromCode(code int) string {
	switch code {
	case 1:
		return "sensitive"
	case 2:
		return "r18"
	default:
		return "all_ages"
	}
}
