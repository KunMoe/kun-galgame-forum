package handler

import (
	"encoding/json"
	"time"

	"kun-galgame-api/internal/trust/dto"
	"kun-galgame-api/internal/trust/enforce"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/trustclient"

	"github.com/gofiber/fiber/v3"
)

type TrustHandler struct {
	enforce        *enforce.Service
	callbackSecret string
}

func NewTrustHandler(enforceService *enforce.Service, callbackSecret string) *TrustHandler {
	return &TrustHandler{enforce: enforceService, callbackSecret: callbackSecret}
}

func (h *TrustHandler) Callback(c fiber.Ctx) error {
	body := c.Body()
	if !trustclient.VerifyCallbackSignature(
		h.callbackSecret,
		c.Get("X-Trust-Timestamp"),
		c.Get("X-Trust-Signature"),
		body,
		time.Now(),
	) {
		return response.Error(c, errors.ErrUnauthorized("回调签名校验失败"))
	}

	var cb dto.TrustCallback
	if err := json.Unmarshal(body, &cb); err != nil {
		return response.Error(c, errors.ErrBadRequest("回调内容无效"))
	}

	if err := h.enforce.Apply(c.Context(), cb); err != nil {
		return response.Error(c, errors.ErrInternal("处置执行失败"))
	}
	return response.OK(c, fiber.Map{"ok": true})
}
