package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type GalgameCollectionHandler struct {
	collectionService *service.CollectionService
}

func NewGalgameCollectionHandler(collectionService *service.CollectionService) *GalgameCollectionHandler {
	return &GalgameCollectionHandler{collectionService: collectionService}
}

func (h *GalgameCollectionHandler) Create(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.CreateCollectionRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := collectionToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	id, appErr := h.collectionService.Create(c.Context(), user.ID, token, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, id)
}

func (h *GalgameCollectionHandler) Update(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	cid, err := strconv.Atoi(c.Params("cid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的收藏夹 ID"))
	}
	var req dto.UpdateCollectionRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := collectionToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.collectionService.Update(c.Context(), user.ID, token,
		perm.CanUser(user.ID, user.Roles, perm.CollectionEditAny), cid, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "收藏夹已更新")
}

func (h *GalgameCollectionHandler) Delete(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	cid, err := strconv.Atoi(c.Params("cid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的收藏夹 ID"))
	}
	token, appErr := collectionToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.collectionService.Delete(c.Context(), user.ID, token,
		perm.CanUser(user.ID, user.Roles, perm.CollectionDeleteAny), cid); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "收藏夹已删除")
}

func (h *GalgameCollectionHandler) GetDetail(c fiber.Ctx) error {
	cid, err := strconv.Atoi(c.Params("cid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的收藏夹 ID"))
	}
	page, limit := parseCollectionPage(c, 24)
	detail, appErr := h.collectionService.GetDetail(
		c.Context(), optionalUID(c), middleware.GetAccessToken(c), cid, page, limit, utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

func (h *GalgameCollectionHandler) MyCollectionsForGalgame(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}
	token, appErr := collectionToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	cols, appErr := h.collectionService.GetMyCollectionsForGalgame(c.Context(), user.ID, token, gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{"collections": cols})
}

func (h *GalgameCollectionHandler) SetMembership(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}
	var req dto.SetCollectionMembershipRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := collectionToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.collectionService.SetMembership(c.Context(), user.ID, token, gid, req.CollectionIDs); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

func (h *GalgameCollectionHandler) GetUserCollections(c fiber.Ctx) error {
	ownerID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}
	page, limit := parseCollectionPage(c, 24)
	items, total, appErr := h.collectionService.ListForUser(
		c.Context(), ownerID, optionalUID(c), middleware.GetAccessToken(c), page, limit, utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.Paginated(c, items, total)
}

// Every collection write speaks to the catalog as the signed-in person, so a
// session with no OAuth access token cannot make one. That is a re-login, not
// a server fault: the token is minted at sign-in and this site never had one
// to begin with for a session that predates the folder scopes.
func collectionToken(c fiber.Ctx) (string, *errors.AppError) {
	token := middleware.GetAccessToken(c)
	if token == "" {
		return "", errors.ErrAuthExpired()
	}
	return token, nil
}

func parseCollectionPage(c fiber.Ctx, defLimit int) (page, limit int) {
	page, _ = strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	limit, _ = strconv.Atoi(c.Query("limit"))
	if limit < 1 {
		limit = defLimit
	}
	if limit > 50 {
		limit = 50
	}
	return
}
