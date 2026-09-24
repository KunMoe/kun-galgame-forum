package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

// adoptAndPublish is one user gesture but two transactions upstream, so the
// claim failing is only fatal when the publish fails as well. That is what
// makes a retry work: the second attempt's claim is refused (the work is
// already this user's draft) while its publish succeeds. Returning early on the
// claim error instead left a half-adopted work with no way to finish it — and
// the wizard would keep offering it as unclaimed until the next daily reindex,
// so every retry hit the same refused claim.
func AdoptAndPublish(
	ctx context.Context,
	catalog *catalogclient.Client,
	accessToken string,
	workID int64,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	return adoptAndPublish(ctx, catalog, accessToken, workID)
}

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
