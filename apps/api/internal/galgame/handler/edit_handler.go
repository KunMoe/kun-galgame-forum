package handler

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/userclient"

	"github.com/gofiber/fiber/v3"
)

type EditHandler struct {
	catalog       *catalogclient.Client
	galgameClient *client.GalgameClient
	users         *userclient.Client
	notifier      msgService.Notifier
	repo          *repository.GalgameRepository
	owners        EntryOwners
}

type EntryOwners interface {
	OwnerOf(gid int) int
}

type repoOwners struct{ repo *repository.GalgameRepository }

func (r repoOwners) OwnerOf(gid int) int {
	if r.repo == nil || gid <= 0 {
		return 0
	}
	row := r.repo.FindLocal(gid)
	if row.CreatorUserID == nil {
		return 0
	}
	return *row.CreatorUserID
}

func NewEditHandler(
	catalog *catalogclient.Client,
	galgameClient *client.GalgameClient, users *userclient.Client,
	notifier msgService.Notifier, repo *repository.GalgameRepository,
) *EditHandler {
	return &EditHandler{
		catalog: catalog, galgameClient: galgameClient,
		users: users, notifier: notifier, repo: repo, owners: repoOwners{repo: repo},
	}
}

func (h *EditHandler) WithOwners(owners EntryOwners) *EditHandler {
	h.owners = owners
	return h
}

type editUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

func (h *EditHandler) userMap(ctx context.Context, uids map[int]bool) map[int]editUser {
	out := make(map[int]editUser, len(uids))
	if h.users == nil || len(uids) == 0 {
		return out
	}
	ids := make([]int, 0, len(uids))
	for id := range uids {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	resolved, err := h.users.Users(ctx, ids)
	if err != nil {
		slog.Warn("galgame edit: user enrichment failed", "error", err)
		return out
	}
	for id, u := range resolved {
		out[id] = editUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
	}
	return out
}

func collectProposalUIDs(items []catalogclient.EditProposal) map[int]bool {
	uids := make(map[int]bool)
	for i := range items {
		uids[int(items[i].ProposerUID)] = true
		if items[i].DecidedByUID != nil {
			uids[int(*items[i].DecidedByUID)] = true
		}
		for _, a := range items[i].Amendments {
			uids[int(a.AmenderUID)] = true
		}
	}
	return uids
}

// Must equal the forum OAuth client's oauth_clients.catalog_site binding.
const catalogSite = "kungal"

// THE ID-SPACE TRAP, and the reason for every workIDOf / gidOf / entryOf call
// in this file: a kungal URL carries a gid, the engine speaks registry work
// ids, and the two spaces OVERLAP. A missed translation does not fail — it
// silently addresses a different game.
const entityTypeGame = catalogclient.EntityTypeWork

const fieldKeyPrefix = catalogclient.FieldKeyPrefix

var errEditDown = errors.New(errors.CodeBiz, "资料库编辑服务暂不可用", http.StatusServiceUnavailable)

func (h *EditHandler) ownerOf(_ context.Context, gid int64) int {
	if h.owners == nil {
		return 0
	}
	return h.owners.OwnerOf(int(gid))
}

func (h *EditHandler) isGameOwner(ctx context.Context, workID, uid int64) bool {
	gid := h.gidOf(ctx, workID)
	if gid == 0 {
		return false
	}
	owner := h.ownerOf(ctx, int64(gid))
	return owner > 0 && int64(owner) == uid
}

func (h *EditHandler) workIDOf(ctx context.Context, gid int64) (int64, *errors.AppError) {
	if h.galgameClient == nil {
		return 0, errEditDown
	}
	ids, appErr := h.galgameClient.CatalogWorkIDs(ctx, []int{int(gid)})
	if appErr != nil {
		return 0, appErr
	}
	workID, ok := ids[int(gid)]
	if !ok {
		return 0, errors.ErrNotFound("条目不存在")
	}
	return workID, nil
}

type editEntry struct {
	GID      int
	OwnerUID int
	Name     string
}

func (h *EditHandler) entryOf(ctx context.Context, workID int64) editEntry {
	gid := h.gidOf(ctx, workID)
	if gid == 0 {
		return editEntry{}
	}
	entry := editEntry{GID: gid, OwnerUID: h.ownerOf(ctx, int64(gid))}
	if h.galgameClient != nil {
		if rows, appErr := h.galgameClient.CatalogRowsByGIDs(ctx, []int{gid}, "names", "all"); appErr == nil {
			if row, ok := rows[gid]; ok {
				brief := client.CatalogItemToBrief(ctx, &row)
				entry.Name = client.BriefName(&brief)
			}
		}
	}
	return entry
}

func (h *EditHandler) gidOf(ctx context.Context, workID int64) int {
	if h.galgameClient == nil {
		return 0
	}
	gids, appErr := h.galgameClient.GIDsByCatalogIDs(ctx, []int64{workID})
	if appErr != nil {
		slog.Warn("galgame edit: work id → gid failed", "work", workID, "error", appErr)
		return 0
	}
	return gids[workID]
}

func (h *EditHandler) notifyDecision(prop *catalogclient.EditProposal, gid int, senderID int64, kind msgService.NotifyKind, content string) {
	if h.notifier == nil {
		return
	}
	if err := h.notifier.Emit(nil, msgService.Spec{
		SenderID: int(senderID), ReceiverID: int(prop.ProposerUID),
		Kind: kind, Content: content, GalgameID: gid,
	}); err != nil {
		slog.Warn("galgame edit: decision notification failed",
			"proposal", prop.ID, "kind", kind, "error", err)
	}
}

// The engine names the field it refused in the problem body. Passing it through
// is what lets the form put "第 1 行 标题：必填" under the control instead of
// showing the whole rejection as one toast.
func problemFields(in []catalogclient.ProblemFieldError) []errors.FieldError {
	if len(in) == 0 {
		return nil
	}
	out := make([]errors.FieldError, 0, len(in))
	for _, e := range in {
		out = append(out, errors.FieldError{
			Pointer:   e.Pointer,
			Parameter: e.Parameter,
			Header:    e.Header,
			Reason:    e.Reason,
			Detail:    e.Detail,
		})
	}
	return out
}

func editStatusError(status int, message string) *errors.AppError {
	switch status {
	case http.StatusForbidden:
		return errors.ErrForbidden("你没有权限执行此操作")
	case http.StatusNotFound:
		return errors.ErrNotFound("条目或提案不存在")
	case http.StatusUnprocessableEntity:
		return errors.ErrValidation(message)
	case http.StatusConflict:
		return errors.New(errors.CodeBiz,
			"操作冲突（条目已被他人修改或提案已关闭），请刷新后重试", http.StatusConflict)
	}
	return nil
}

// The revision and proposal listings are read with the forum's own application
// key, not the reader's token, so a 401/403 here is this deployment's key being
// wrong and never the user's grant — routing it through userEditError would
// prompt a perfectly valid session to log out and back in, which cannot fix it.
func editError(c fiber.Ctx, err error) error {
	var apiErr *catalogclient.UserAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return response.Error(c, errEditDown)
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return response.Error(c, errors.ErrNotFound("条目或提案不存在"))
	case stderrors.Is(err, catalogclient.ErrUnauthorized),
		stderrors.Is(err, catalogclient.ErrInsufficientScope):
		slog.Error("galgame edit: 应用 key 被 catalog 拒绝, 编辑历史全站不可读", "error", err)
		return response.Error(c, errEditDown)
	case stderrors.As(err, &apiErr):
		if appErr := editStatusError(apiErr.Status, apiErr.Message); appErr != nil {
			return response.Error(c, appErr)
		}
		slog.Error("galgame edit: upstream error", "status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Message)
		return response.Error(c, errEditDown)
	default:
		slog.Warn("galgame edit: catalog unreachable", "error", err)
		return response.Error(c, errEditDown)
	}
}

// The insufficient-scope case must never be folded into the generic 403. The
// user's grant predates `catalog:edit` and no refresh can widen it, so the only
// fix is a re-login, and code 235 is what the frontend keys that prompt on. A
// 233 tells a user they may not edit when in fact they may; a 205 logs out a
// perfectly live session.
func userEditError(c fiber.Ctx, err error) error {
	var apiErr *catalogclient.UserAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrInsufficientScope):
		return response.Error(c, errors.ErrReauthRequired(
			"编辑资料需要新的授权，请退出登录后重新登录以授予该权限"))
	case stderrors.Is(err, catalogclient.ErrUnauthorized):
		return response.Error(c, errors.ErrAuthExpired())
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return response.Error(c, errors.ErrNotFound("条目或提案不存在"))
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return response.Error(c, errEditDown)
	case stderrors.As(err, &apiErr):
		if appErr := editStatusError(apiErr.Status, apiErr.Message); appErr != nil {
			return response.Error(c, appErr.WithFieldErrors(problemFields(apiErr.FieldErrors)))
		}
		slog.Error("galgame edit: user-plane upstream error",
			"status", apiErr.Status, "code", apiErr.ProblemCode, "msg", apiErr.Message)
		return response.Error(c, errEditDown)
	default:
		slog.Warn("galgame edit: catalog user plane unreachable", "error", err)
		return response.Error(c, errEditDown)
	}
}

func userToken(c fiber.Ctx) (string, *errors.AppError) {
	token := middleware.GetAccessToken(c)
	if token == "" {
		return "", errors.ErrAuthExpired()
	}
	return token, nil
}

func parseGid(c fiber.Ctx) (int64, *errors.AppError) {
	gid, err := strconv.ParseInt(c.Params("gid"), 10, 64)
	if err != nil || gid <= 0 {
		return 0, errors.ErrBadRequest("无效的 Galgame ID")
	}
	return gid, nil
}

func parseProposalID(c fiber.Ctx) (int64, *errors.AppError) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.ErrBadRequest("无效的提案 ID")
	}
	return id, nil
}

func queryInt(c fiber.Ctx, key string) int {
	n, err := strconv.Atoi(c.Query(key))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func (h *EditHandler) Bootstrap(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	ctx := c.Context()
	workID, appErr := h.workIDOf(ctx, gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	values, err := h.catalog.EditSnapshotUser(ctx, token, entityTypeGame, workID)
	if err != nil {
		return userEditError(c, err)
	}
	schema, err := h.catalog.GetEditSchemaUser(ctx, token, entityTypeGame, workID)
	if err != nil {
		return userEditError(c, err)
	}
	// An unreachable vocabulary face costs the enum fields their options, not the
	// page: the form falls back to read-only for them, which is what shipped
	// before this call existed.
	vocab, err := h.catalog.Vocabularies(ctx)
	if err != nil {
		slog.Warn("galgame edit: vocabularies unreadable", "gid", gid, "error", err)
		vocab = map[string]catalogclient.Vocabulary{}
	}
	return response.OK(c, fiber.Map{
		"gid":          gid,
		"values":       values,
		"fields":       schema.Fields,
		"vocabularies": vocab,
		"can_review":   anyReviewable(schema.Fields),
	})
}

func anyReviewable(fields []catalogclient.EditSchemaField) bool {
	for _, f := range fields {
		if f.CanReview {
			return true
		}
	}
	return false
}

type editSubmitRequest struct {
	Patch map[string]any `json:"patch"`
	Note  string         `json:"note"`
}

func (h *EditHandler) Submit(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req editSubmitRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if len(req.Patch) == 0 {
		return response.Error(c, errors.ErrValidation("没有需要保存的修改"))
	}
	for key := range req.Patch {
		if !strings.HasPrefix(key, fieldKeyPrefix) {
			return response.Error(c, errors.ErrValidation("提案包含非法字段: "+key))
		}
	}
	if len(req.Note) > 2000 {
		return response.Error(c, errors.ErrValidation("编辑说明过长"))
	}
	workID, appErr := h.workIDOf(c.Context(), gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	result, err := h.catalog.CreateEditProposalUser(c.Context(), token, catalogclient.UserEditCreateRequest{
		EntityType: entityTypeGame, EntityID: workID,
		Patch: req.Patch, Note: req.Note,
	})
	if err != nil {
		return userEditError(c, err)
	}
	if !result.Merged {
		h.submitSideEffects(c.Context(), &result.Proposal)
	}
	out := fiber.Map{"merged": result.Merged, "proposal": result.Proposal}
	if result.Revision != nil {
		out["revision"] = result.Revision
	}
	return response.OK(c, out)
}

func (h *EditHandler) submitSideEffects(ctx context.Context, prop *catalogclient.EditProposal) {
	entry := h.entryOf(ctx, prop.EntityID)
	if entry.GID == 0 {
		return
	}
	if h.notifier != nil && entry.OwnerUID > 0 {
		if err := h.notifier.Emit(nil, msgService.Spec{
			SenderID: int(prop.ProposerUID), ReceiverID: entry.OwnerUID,
			Kind: msgService.NotifyRequested, Content: entry.Name,
			GalgameID: entry.GID,
		}); err != nil {
			slog.Warn("galgame edit: requested notification failed", "proposal", prop.ID, "error", err)
		}
	}
	if h.repo != nil {
		if err := h.repo.DB().WithContext(ctx).Exec(`
			INSERT INTO galgame_activity (wiki_pr_id, galgame_id, user_id, type, created)
			VALUES (?, ?, ?, 'GALGAME_PR_CREATION', now())
			ON CONFLICT (wiki_pr_id) DO NOTHING
		`, prop.ID, entry.GID, prop.ProposerUID).Error; err != nil {
			slog.Warn("galgame edit: activity timeline write failed", "proposal", prop.ID, "error", err)
		}
	}
}

func (h *EditHandler) Revisions(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	workID, appErr := h.workIDOf(c.Context(), gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	items, err := h.catalog.ListEditRevisions(c.Context(), entityTypeGame, workID, queryInt(c, "limit"))
	if err != nil {
		return editError(c, err)
	}
	uids := make(map[int]bool)
	for i := range items {
		uids[int(items[i].ActorUID)] = true
		if items[i].AmenderUID != nil {
			uids[int(*items[i].AmenderUID)] = true
		}
	}
	return response.OK(c, fiber.Map{
		"gid": gid, "items": items, "users": h.userMap(c.Context(), uids),
		"can_revert": h.canRevert(c, workID),
	})
}

func (h *EditHandler) canRevert(c fiber.Ctx, workID int64) bool {
	token := middleware.GetAccessToken(c)
	if token == "" {
		return false
	}
	schema, err := h.catalog.GetEditSchemaUser(c.Context(), token, entityTypeGame, workID)
	if err != nil {
		slog.Warn("galgame edit: revert projection failed", "work", workID, "error", err)
		return false
	}
	editable := 0
	for _, f := range schema.Fields {
		if f.Locked || f.Deprecated {
			continue
		}
		editable++
		if !f.CanReview {
			return false
		}
	}
	return editable > 0
}

type editRevertRequest struct {
	ToSeq int    `json:"to_seq"`
	Note  string `json:"note"`
}

func (h *EditHandler) Revert(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req editRevertRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if req.ToSeq < 1 {
		return response.Error(c, errors.ErrBadRequest("需要目标版本号"))
	}
	if len(req.Note) > 2000 {
		return response.Error(c, errors.ErrValidation("说明过长"))
	}
	ctx := c.Context()
	workID, appErr := h.workIDOf(ctx, gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	revisionID, err := h.catalog.RevisionIDBySeq(ctx, entityTypeGame, workID, req.ToSeq)
	if err != nil {
		if stderrors.Is(err, catalogclient.ErrNotFound) {
			return response.Error(c, errors.ErrBadRequest("目标版本不存在"))
		}
		return editError(c, err)
	}
	result, err := h.catalog.RevertEditEntityUser(ctx, token, revisionID, req.Note)
	if err != nil {
		return userEditError(c, err)
	}
	return response.OK(c, result)
}

func (h *EditHandler) Diff(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	from, to := queryInt(c, "from"), queryInt(c, "to")
	if from < 1 || to < 1 {
		return response.Error(c, errors.ErrBadRequest("需要 from/to 版本号"))
	}
	workID, appErr := h.workIDOf(c.Context(), gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	diff, err := h.catalog.DiffEditRevisions(c.Context(), entityTypeGame, workID, from, to)
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, diff)
}

type proposalItem struct {
	catalogclient.EditProposal
	GID     int                  `json:"gid"`
	Galgame *client.GalgameBrief `json:"galgame,omitempty"`
}

func (h *EditHandler) enrich(ctx context.Context, items []catalogclient.EditProposal) []proposalItem {
	workIDs := make([]int64, 0, len(items))
	seen := make(map[int64]bool, len(items))
	for i := range items {
		if id := items[i].EntityID; !seen[id] {
			seen[id] = true
			workIDs = append(workIDs, id)
		}
	}
	var gidByWork map[int64]int
	if len(workIDs) > 0 && h.galgameClient != nil {
		var appErr *errors.AppError
		if gidByWork, appErr = h.galgameClient.GIDsByCatalogIDs(ctx, workIDs); appErr != nil {
			slog.Warn("galgame edit: work id → gid enrichment failed", "error", appErr)
		}
	}
	gids := make([]int, 0, len(gidByWork))
	for _, gid := range gidByWork {
		gids = append(gids, gid)
	}
	var briefs map[int]client.GalgameBrief
	if len(gids) > 0 && h.galgameClient != nil {
		rows, appErr := h.galgameClient.CatalogRowsByGIDs(ctx, gids, "names,covers", "all")
		if appErr != nil {
			slog.Warn("galgame edit: brief enrichment failed", "error", appErr)
		} else {
			briefs = make(map[int]client.GalgameBrief, len(rows))
			for gid := range rows {
				row := rows[gid]
				briefs[gid] = client.CatalogItemToBrief(ctx, &row)
			}
		}
	}
	out := make([]proposalItem, 0, len(items))
	for i := range items {
		item := proposalItem{EditProposal: items[i], GID: gidByWork[items[i].EntityID]}
		if b, ok := briefs[item.GID]; ok {
			brief := b
			item.Galgame = &brief
		}
		out = append(out, item)
	}
	return out
}

func (h *EditHandler) Queue(c fiber.Ctx) error {
	status := c.Query("status", "open")
	switch status {
	case "open", "merged", "declined", "withdrawn", "":
	default:
		return response.Error(c, errors.ErrBadRequest("未知的提案状态"))
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	items, err := h.catalog.ListEditProposalsUser(c.Context(), token, catalogclient.UserEditProposalFilter{
		EntityType: entityTypeGame,
		Status:     status, Limit: queryInt(c, "limit"),
	})
	if err != nil {
		return userEditError(c, err)
	}
	return response.OK(c, fiber.Map{
		"items": h.enrich(c.Context(), items),
		"users": h.userMap(c.Context(), collectProposalUIDs(items)),
	})
}

func (h *EditHandler) Mine(c fiber.Ctx) error {
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var entityID int64
	if gid := queryInt(c, "gid"); gid > 0 {
		workID, appErr := h.workIDOf(c.Context(), int64(gid))
		if appErr != nil {
			return response.Error(c, appErr)
		}
		entityID = workID
	}
	items, err := h.catalog.ListEditProposalsUser(c.Context(), token, catalogclient.UserEditProposalFilter{
		EntityType: entityTypeGame, EntityID: entityID, Mine: true,
		Status: c.Query("status"), Limit: queryInt(c, "limit"),
	})
	if err != nil {
		return userEditError(c, err)
	}
	return response.OK(c, fiber.Map{
		"items": h.enrich(c.Context(), items),
		"users": h.userMap(c.Context(), collectProposalUIDs(items)),
	})
}

func (h *EditHandler) proposalForReview(ctx context.Context, token string, id int64) (*catalogclient.EditProposal, error) {
	prop, err := h.catalog.GetEditProposalUser(ctx, token, id)
	if err != nil {
		return nil, err
	}
	if prop.Site != catalogSite || prop.EntityType != entityTypeGame {
		return nil, &catalogclient.UserAPIError{Status: http.StatusNotFound, Message: "proposal outside the kungal tenant"}
	}
	return prop, nil
}

func (h *EditHandler) reviewEntry(c fiber.Ctx, token string, id int64) (*catalogclient.EditProposal, error) {
	ctx := c.Context()
	prop, err := h.proposalForReview(ctx, token, id)
	if err != nil {
		return nil, err
	}
	user := middleware.GetUser(c)
	if user == nil {
		return nil, &catalogclient.UserAPIError{Status: http.StatusForbidden, Message: "review entry denied"}
	}
	if !role.CanModerate(user.Roles) && !h.isGameOwner(ctx, prop.EntityID, int64(user.ID)) {
		return nil, &catalogclient.UserAPIError{Status: http.StatusForbidden, Message: "review entry denied"}
	}
	return prop, nil
}

func canDecide(prop *catalogclient.EditProposal, fields []catalogclient.EditSchemaField) bool {
	patch := prop.EffectivePatch
	if len(patch) == 0 {
		patch = prop.Patch
	}
	if len(patch) == 0 {
		return false
	}
	reviewable := make(map[string]bool, len(fields))
	for _, f := range fields {
		reviewable[f.Key] = f.CanReview
	}
	for key := range patch {
		if !reviewable[key] {
			return false
		}
	}
	return true
}

func (h *EditHandler) GameProposals(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	status := c.Query("status", "open")
	switch status {
	case "open", "merged", "declined", "withdrawn", "":
	default:
		return response.Error(c, errors.ErrBadRequest("未知的提案状态"))
	}
	workID, appErr := h.workIDOf(c.Context(), gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	items, err := h.catalog.ListEditProposals(c.Context(), catalogclient.EditProposalFilter{
		EntityType: entityTypeGame, EntityID: workID, Site: catalogSite,
		Status: status, Limit: queryInt(c, "limit"),
	})
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, fiber.Map{
		"gid": gid, "items": items,
		"users": h.userMap(c.Context(), collectProposalUIDs(items)),
	})
}

func (h *EditHandler) ProposalDetail(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	ctx := c.Context()
	prop, err := h.reviewEntry(c, token, id)
	if err != nil {
		return userEditError(c, err)
	}
	values, err := h.catalog.EditSnapshotUser(ctx, token, entityTypeGame, prop.EntityID)
	if err != nil {
		return userEditError(c, err)
	}
	schema, err := h.catalog.GetEditSchemaUser(ctx, token, entityTypeGame, prop.EntityID)
	if err != nil {
		return userEditError(c, err)
	}
	enriched := h.enrich(ctx, []catalogclient.EditProposal{*prop})
	return response.OK(c, fiber.Map{
		"proposal":   enriched[0],
		"values":     values,
		"fields":     schema.Fields,
		"users":      h.userMap(ctx, collectProposalUIDs([]catalogclient.EditProposal{*prop})),
		"can_decide": canDecide(prop, schema.Fields),
	})
}

type editAmendRequest struct {
	Set   map[string]any `json:"set"`
	Unset []string       `json:"unset"`
	Note  string         `json:"note"`
}

func (h *EditHandler) Amend(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req editAmendRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if len(req.Set) == 0 && len(req.Unset) == 0 {
		return response.Error(c, errors.ErrValidation("没有需要修改的内容"))
	}
	for key := range req.Set {
		if !strings.HasPrefix(key, fieldKeyPrefix) {
			return response.Error(c, errors.ErrValidation("提案包含非法字段: "+key))
		}
	}
	if len(req.Note) > 2000 {
		return response.Error(c, errors.ErrValidation("说明过长"))
	}
	amendment, err := h.catalog.AmendEditProposalUser(c.Context(), token, id, req.Set, req.Unset, req.Note)
	if err != nil {
		return userEditError(c, err)
	}
	return response.OK(c, amendment)
}

type editDecisionRequest struct {
	Note string `json:"note"`
}

func (h *EditHandler) Merge(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req editDecisionRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if len(req.Note) > 2000 {
		return response.Error(c, errors.ErrValidation("说明过长"))
	}
	ctx := c.Context()
	prop, err := h.proposalForReview(ctx, token, id)
	if err != nil {
		return userEditError(c, err)
	}
	rev, err := h.catalog.MergeEditProposalUser(ctx, token, id, req.Note)
	if err != nil {
		return userEditError(c, err)
	}
	h.mergeSideEffects(ctx, prop, int64(user.ID), rev)
	return response.OK(c, rev)
}

func (h *EditHandler) mergeSideEffects(ctx context.Context, prop *catalogclient.EditProposal, mergerID int64, rev *catalogclient.EditRevision) {
	if prop.ProposerUID != mergerID {
		moemoepoint.Award(int(prop.ProposerUID), constants.RewardPRMerge,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_pr", int(prop.EntityID)),
			moemoepoint.Key("galgame_edit_merged", strconv.FormatInt(prop.ID, 10)))
	}
	entry := h.entryOf(ctx, prop.EntityID)
	if h.repo != nil && entry.GID > 0 {
		if err := h.repo.Touch(h.repo.DB().WithContext(ctx), entry.GID); err != nil {
			slog.Warn("galgame edit: resource_update_time bump failed", "gid", entry.GID, "error", err)
		}
	}
	content := entry.Name
	if rev != nil && rev.AmenderUID != nil {
		content = strings.TrimSpace(content + "（审核时有修正）")
	}
	h.notifyDecision(prop, entry.GID, mergerID, msgService.NotifyMerged, content)
}

func (h *EditHandler) Decline(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req editDecisionRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if strings.TrimSpace(req.Note) == "" {
		return response.Error(c, errors.ErrValidation("请填写拒绝理由"))
	}
	if len(req.Note) > 2000 {
		return response.Error(c, errors.ErrValidation("说明过长"))
	}
	ctx := c.Context()
	target, err := h.proposalForReview(ctx, token, id)
	if err != nil {
		return userEditError(c, err)
	}
	prop, err := h.catalog.DeclineEditProposalUser(ctx, token, id, req.Note)
	if err != nil {
		return userEditError(c, err)
	}
	entry := h.entryOf(ctx, target.EntityID)
	content := req.Note
	if entry.Name != "" {
		content = entry.Name + "：" + req.Note
	}
	h.notifyDecision(target, entry.GID, int64(user.ID), msgService.NotifyDeclined, content)
	return response.OK(c, prop)
}

func (h *EditHandler) Withdraw(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	prop, err := h.catalog.WithdrawEditProposalUser(c.Context(), token, id)
	if err != nil {
		return userEditError(c, err)
	}
	return response.OK(c, prop)
}
