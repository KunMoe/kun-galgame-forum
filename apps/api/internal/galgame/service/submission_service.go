package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

type SubmissionService struct {
	galgameClient *client.GalgameClient
	catalog       *catalogclient.Client
	galgameRepo   *repository.GalgameRepository
}

func NewSubmissionService(
	galgameClient *client.GalgameClient,
	catalog *catalogclient.Client,
	galgameRepo *repository.GalgameRepository,
) *SubmissionService {
	return &SubmissionService{
		galgameClient: galgameClient,
		catalog:       catalog,
		galgameRepo:   galgameRepo,
	}
}

const submissionSite = client.ClaimSiteKungal

type SubmitResult struct {
	GID        int    `json:"gid"`
	WorkID     int64  `json:"work_id"`
	ClaimState string `json:"claim_state"`
	// False when the submitter uploaded a banner that did not make it onto the
	// work. The submission itself still stands, so this is reported rather than
	// raised — but it has to be reported: silence here hid a broken cover patch
	// for the whole life of the feature.
	BannerAttached bool `json:"banner_attached"`
}

func (s *SubmissionService) Submit(
	ctx context.Context,
	accessToken string,
	uid int,
	form *SubmissionForm,
) (*SubmitResult, *errors.AppError) {
	if form.DisplayName() == "" {
		return nil, errors.ErrValidation("请至少填写一个语言的标题")
	}
	released, appErr := form.Released()
	if appErr != nil {
		return nil, appErr
	}
	// POST /v2/me/claims has no released key and mintClaim fills no
	// SubmitWorkParams.Released, so a minted work gets no catalog_release row at
	// all — and catalog.release proposals may only edit existing rows, so there
	// is nothing to file afterwards either. The date is validated and then
	// dropped; say so rather than losing it in silence.
	if released != nil {
		slog.Warn("submit: 发售日期无法送达 catalog, v2 建档面没有 released 字段",
			"uid", uid, "released", *released)
	}
	res, err := s.catalog.SubmitWorkUser(ctx, accessToken, catalogclient.UserWorkSubmitRequest{
		Fields: form.Fields(), ConfirmDuplicates: form.ConfirmDuplicates,
	})
	if err != nil {
		return nil, submitError(err)
	}
	gid := int(res.ProductWorkID)
	// Stamp the submitter now rather than waiting for the claim feed to say so
	// ten minutes later: it is what authorises the submitter to open their own
	// unpublished entry, and it survives a Redis flush. The row stays
	// published=false, so it shows up in no list and no feed.
	if uid > 0 {
		if err := s.galgameRepo.SubmitLocal(s.galgameRepo.DB().WithContext(ctx), gid, uid); err != nil {
			slog.Warn("submit: 记录本地投稿人失败", "gid", gid, "uid", uid, "error", err)
		}
	}
	patch := form.CoverPatch()
	attached := patch == nil
	if patch != nil {
		attached = s.attachBanner(ctx, accessToken, res.WorkID, patch)
	}
	return &SubmitResult{
		GID: gid, WorkID: res.WorkID, ClaimState: res.ClaimState, BannerAttached: attached,
	}, nil
}

// adoptAndPublish is one user gesture but two transactions upstream, so the
// claim failing is only fatal when the publish fails as well. That is what
// makes a retry work: the second attempt's claim is refused (the work is
// already this user's draft) while its publish succeeds. Returning early on the
// claim error instead left a half-adopted work with no way to finish it — and
// the wizard would keep offering it as unclaimed until the next daily reindex,
// so every retry hit the same refused claim.
func adoptAndPublish(
	ctx context.Context,
	catalog *catalogclient.Client,
	accessToken string,
	workID int64,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	_, claimErr := catalog.ActOnClaimUser(ctx, accessToken, workID,
		catalogclient.ClaimActionClaim, catalogclient.UserClaimActionRequest{ProductWorkID: workID})

	res, pubErr := catalog.ActOnClaimUser(ctx, accessToken, workID,
		catalogclient.ClaimActionPublish, catalogclient.UserClaimActionRequest{})
	if pubErr == nil {
		return res, nil
	}
	if claimErr != nil {
		return nil, claimActionError(claimErr)
	}
	return nil, claimActionError(pubErr)
}

func (s *SubmissionService) Resubmit(
	ctx context.Context,
	accessToken string,
	gid int,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	return s.act(ctx, accessToken, gid, catalogclient.ClaimActionSubmit, "")
}

func (s *SubmissionService) Withdraw(
	ctx context.Context,
	accessToken string,
	gid int,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	return s.act(ctx, accessToken, gid, catalogclient.ClaimActionWithdraw, "")
}

// Approving a submission moves the claim state and nothing else, so a banner
// proposal left open at submit time is never decided by anyone and the cover
// simply never appears — which is why the only way to get one on used to be
// editing the entry a second time. The submitter owns the work, so merge it here.
func (s *SubmissionService) attachBanner(
	ctx context.Context,
	accessToken string,
	workID int64,
	patch map[string]any,
) bool {
	created, err := s.catalog.CreateEditProposalUser(ctx, accessToken, catalogclient.UserEditCreateRequest{
		EntityType: catalogclient.EntityTypeWork, EntityID: workID,
		Patch: patch, Note: "投稿时提交的横幅图",
	})
	if err != nil {
		slog.Error("submit: 附加横幅图失败, 投稿已建立但封面丢失", "work", workID, "error", err)
		return false
	}
	if created.Merged {
		return true
	}
	if _, err := s.catalog.MergeEditProposalUser(ctx, accessToken, created.Proposal.ID, ""); err != nil {
		slog.Error("submit: 合并横幅图提案失败, 投稿已建立但封面未落地",
			"work", workID, "proposal", created.Proposal.ID, "error", err)
		return false
	}
	return true
}

func (s *SubmissionService) DeleteDraft(
	ctx context.Context,
	accessToken string,
	gid int,
) *errors.AppError {
	workID, appErr := s.workIDOf(ctx, gid)
	if appErr != nil {
		return appErr
	}
	// Owner-only and draft-only are catalog's checks, and they have to stay
	// catalog's: the local delete below cascades, so it must not run until the
	// authority has agreed this row is a disposable draft.
	if err := s.catalog.DeleteMyClaim(ctx, accessToken, workID); err != nil {
		return claimActionError(err)
	}
	if err := s.galgameRepo.DeleteLocalDraft(gid); err != nil {
		slog.Error("delete draft: 上游草稿已删除, 本地行未能清理", "gid", gid, "error", err)
		return errors.ErrInternal("草稿已删除, 但本站条目清理失败, 请联系管理员")
	}
	return nil
}

func (s *SubmissionService) act(
	ctx context.Context,
	accessToken string,
	gid int,
	action string,
	reason string,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	workID, appErr := s.workIDOf(ctx, gid)
	if appErr != nil {
		return nil, appErr
	}
	res, err := s.catalog.ActOnClaimUser(ctx, accessToken, workID, action, catalogclient.UserClaimActionRequest{
		Reason: reason,
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	return res, nil
}

func (s *SubmissionService) workIDOf(ctx context.Context, gid int) (int64, *errors.AppError) {
	ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})
	if appErr != nil {
		return 0, appErr
	}
	workID, ok := ids[gid]
	if !ok {
		return 0, errors.ErrNotFound("条目不存在")
	}
	return workID, nil
}

// Catalog's duplicate gate is soft: the same request with confirm_duplicates
// mints anyway. Left on the generic 409 path it reached the submitter as
// catalog's own English "re-send with confirm_duplicates=true" — an instruction
// about a field the wizard has no control for, so a legitimate same-title work
// could not be submitted at all. It only started firing once the mint actually
// carried titles.
func submitError(err error) *errors.AppError {
	var apiErr *catalogclient.UserAPIError
	if stderrors.As(err, &apiErr) && apiErr.ProblemCode == "DUPLICATE_SUSPECTS" {
		return errors.ErrDuplicateSuspects("资料库中已有同名作品")
	}
	return claimActionError(err)
}

func claimActionError(err error) *errors.AppError {
	var apiErr *catalogclient.UserAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrInsufficientScope):
		return errors.ErrReauthRequired("投稿需要新的授权，请退出登录后重新登录以授予该权限")
	case stderrors.Is(err, catalogclient.ErrUnauthorized):
		return errors.ErrAuthExpired()
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return errors.ErrNotFound("条目不存在")
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return errors.New(errors.CodeBiz, "资料库服务暂不可用", http.StatusServiceUnavailable)
	case stderrors.As(err, &apiErr):
		switch apiErr.Status {
		case http.StatusForbidden:
			return errors.ErrForbidden("你没有权限执行此操作")
		case http.StatusUnprocessableEntity:
			return errors.ErrValidation(apiErr.Message)
		case http.StatusConflict:
			return errors.New(errors.CodeBiz, apiErr.Message, http.StatusConflict)
		}
		slog.Error("claim action: 上游错误", "status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Message)
	default:
		slog.Warn("claim action: catalog 不可达", "error", err)
	}
	return errors.New(errors.CodeBiz, "资料库服务暂不可用", http.StatusServiceUnavailable)
}

const claimPageLimit = 20

var mineStates = []string{
	catalogclient.ClaimStatePending,
	catalogclient.ClaimStateDeclined,
	catalogclient.ClaimStateDraft,
}

var claimStates = map[string]bool{
	catalogclient.ClaimStateNone:     true,
	catalogclient.ClaimStateLive:     true,
	catalogclient.ClaimStateDraft:    true,
	catalogclient.ClaimStatePending:  true,
	catalogclient.ClaimStateDeclined: true,
	catalogclient.ClaimStateHidden:   true,
}

func (s *SubmissionService) ListMine(
	ctx context.Context,
	accessToken string,
	query url.Values,
) (*catalogclient.UserClaimPage, *errors.AppError) {
	states := mineStates
	if raw := query.Get("claim_state"); raw != "" {
		states = splitCSV(raw)
		for _, st := range states {
			if !claimStates[st] {
				return nil, errors.ErrBadRequest("未知的申请状态: " + st)
			}
		}
	}
	return s.listClaims(ctx, accessToken, query, catalogclient.ClaimKindSubmitted, states)
}

func (s *SubmissionService) ListAudit(
	ctx context.Context,
	accessToken string,
	query url.Values,
) (*catalogclient.UserClaimPage, *errors.AppError) {
	return s.listClaims(ctx, accessToken, query, catalogclient.ClaimKindAudited, nil)
}

func (s *SubmissionService) listClaims(
	ctx context.Context,
	accessToken string,
	query url.Values,
	kind string,
	states []string,
) (*catalogclient.UserClaimPage, *errors.AppError) {
	page, err := s.catalog.MyClaims(ctx, accessToken, catalogclient.UserClaimFilter{
		ClaimStates: states,
		Before:      int64(atoiOr(query.Get("before"), 0)),
		Limit:       atoiOr(query.Get("limit"), claimPageLimit),
		Kind:        kind,
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	if page.Items == nil {
		page.Items = []catalogclient.UserClaimItem{}
	}
	return page, nil
}

const wizardSearchInclude = "names,covers,refs"

const wizardDefaultLimit = 12

type WizardSearchPage struct {
	Items   []client.GalgameBrief         `json:"items"`
	Pending []catalogclient.UserClaimItem `json:"pending"`
	Total   int64                         `json:"total"`
}

func (s *SubmissionService) SearchWithPending(
	ctx context.Context,
	accessToken string,
	query url.Values,
) (*WizardSearchPage, *errors.AppError) {
	items, total, appErr := s.wizardItems(ctx, query)
	if appErr != nil {
		return nil, appErr
	}
	pending, appErr := s.wizardPending(ctx, accessToken)
	if appErr != nil {
		return nil, appErr
	}
	return &WizardSearchPage{Items: items, Pending: pending, Total: total}, nil
}

func (s *SubmissionService) wizardItems(
	ctx context.Context,
	query url.Values,
) ([]client.GalgameBrief, int64, *errors.AppError) {
	// Neither claim_state nor claimed here on purpose: both are the index's and
	// lag a day, while the claimed_by CatalogItemWizardEligible reads is the
	// registry's. Gating on both hid every work that entered an actionable state
	// since the last reindex — an approved submission stayed unfindable for a
	// day — and claimed=true additionally hid the whole unclaimed supply, which
	// is exactly what the wizard's 认领并发布 button exists to adopt.
	q := url.Values{
		"q":       {query.Get("q")},
		"page":    {strconv.Itoa(atoiOr(query.Get("page"), 1))},
		"limit":   {strconv.Itoa(atoiOr(query.Get("limit"), wizardDefaultLimit))},
		"include": {wizardSearchInclude},
	}
	client.OpenPopulation(q)

	res, appErr := s.galgameClient.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil, 0, appErr
	}
	items := make([]client.GalgameBrief, 0, len(res.Items))
	for i := range res.Items {
		row := &res.Items[i]
		if !client.CatalogItemWizardEligible(row) {
			continue
		}
		b := client.CatalogItemToBrief(ctx, row)
		items = append(items, b)
	}
	return items, res.Total, nil
}

func (s *SubmissionService) wizardPending(
	ctx context.Context,
	accessToken string,
) ([]catalogclient.UserClaimItem, *errors.AppError) {
	page, err := s.catalog.MyClaims(ctx, accessToken, catalogclient.UserClaimFilter{
		ClaimStates: []string{catalogclient.ClaimStatePending, catalogclient.ClaimStateDeclined},
		Limit:       wizardDefaultLimit,
		Kind:        catalogclient.ClaimKindSubmitted,
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	if page.Items == nil {
		return []catalogclient.UserClaimItem{}, nil
	}
	return page.Items, nil
}
