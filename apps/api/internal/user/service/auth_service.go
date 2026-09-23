package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/userclient"

	"github.com/redis/go-redis/v9"
)

type AuthService struct {
	stateRepo   *repository.StateRepository
	rdb         *redis.Client
	oauthClient *oauth.Client
	userClient  *userclient.Client
}

func NewAuthService(
	stateRepo *repository.StateRepository,
	rdb *redis.Client,
	oauthClient *oauth.Client,
	userClient *userclient.Client,
) *AuthService {
	return &AuthService{
		stateRepo:   stateRepo,
		rdb:         rdb,
		oauthClient: oauthClient,
		userClient:  userClient,
	}
}

func (s *AuthService) OAuthCallback(
	ctx context.Context,
	req *dto.OAuthCallbackRequest,
) (*dto.SessionResponse, *errors.AppError) {
	tokenResp, err := s.oauthClient.ExchangeCode(req.Code, req.CodeVerifier)
	if err != nil {
		if oauth.IsBanned(err) {
			return nil, errors.ErrAccountBanned()
		}
		return nil, errors.ErrBadRequest(fmt.Sprintf("OAuth 授权码交换失败: %v", err))
	}

	oauthUser, err := s.oauthClient.FetchUserInfo(tokenResp.AccessToken)
	if err != nil {
		if oauth.IsBanned(err) {
			return nil, errors.ErrAccountBanned()
		}
		return nil, errors.ErrBadRequest(fmt.Sprintf("获取 OAuth 用户信息失败: %v", err))
	}
	if oauthUser.ID <= 0 {
		return nil, errors.ErrInternal(
			"OAuth /oauth/userinfo 未返回用户 id; 请确认 OAuth server 已更新",
		)
	}

	if err := s.stateRepo.Ensure(oauthUser.ID); err != nil {
		return nil, errors.ErrInternal("初始化用户状态失败")
	}

	state, _ := s.stateRepo.FindByID(oauthUser.ID)
	moe := 0
	if state != nil {
		moe = state.Moemoepoint
	}

	avatar := oauthUser.Picture
	if u, ok, uerr := s.userClient.User(ctx, oauthUser.ID); uerr == nil && ok && u.Avatar != "" {
		avatar = u.Avatar
	}

	sessionToken, err := generateSessionToken()
	if err != nil {
		return nil, errors.ErrInternal("生成会话令牌失败")
	}

	respUser := newLoginUserProfile(oauthUser, avatar, moe)

	sessionData := middleware.SessionData{
		UserInfo: middleware.UserInfo{
			ID:             oauthUser.ID,
			Sub:            oauthUser.Sub,
			Name:           oauthUser.Name,
			Email:          oauthUser.Email,
			Roles:          respUser.Roles,
			AdultConfirmed: oauthUser.AdultConfirmed,
			NSFWDisplay:    oauthUser.NSFWDisplay,
		},
		OAuthAccessToken:  tokenResp.AccessToken,
		OAuthRefreshToken: tokenResp.RefreshToken,
		OAuthExpiresAt:    time.Now().Unix() + int64(tokenResp.ExpiresIn),
	}

	data, err := json.Marshal(sessionData)
	if err != nil {
		return nil, errors.ErrInternal("序列化会话数据失败")
	}
	s.rdb.Set(ctx, middleware.SessionKey(sessionToken), data, middleware.SessionTTL)

	return &dto.SessionResponse{
		Token: sessionToken,
		User:  respUser,
	}, nil
}

func newLoginUserProfile(u *oauth.UserInfo, avatar string, moe int) *dto.UserProfile {
	return &dto.UserProfile{
		ID:             u.ID,
		Sub:            u.Sub,
		Name:           u.Name,
		Avatar:         avatar,
		Roles:          role.Union(u.Roles, u.SiteRoles),
		Moemoepoint:    moe,
		Bio:            "",
		AdultConfirmed: u.AdultConfirmed,
		NSFWDisplay:    u.NSFWDisplay,
	}
}

func (s *AuthService) Logout(ctx context.Context, sessionToken string) error {
	val, err := s.rdb.Get(ctx, middleware.SessionKey(sessionToken)).Result()
	if err == nil {
		var session middleware.SessionData
		if json.Unmarshal([]byte(val), &session) == nil && session.OAuthRefreshToken != "" {
			_ = s.oauthClient.RevokeToken(session.OAuthRefreshToken)
		}
	}
	return s.rdb.Del(ctx, middleware.SessionKey(sessionToken)).Err()
}

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
