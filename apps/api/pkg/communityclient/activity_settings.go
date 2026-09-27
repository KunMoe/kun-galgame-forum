package communityclient

import (
	"context"
	"net/http"
)

type ActivitySettings struct {
	UserID    int64   `json:"user_id"`
	Hidden    bool    `json:"hidden"`
	UpdatedAt *string `json:"updated_at"`
}

type ActivitySettingsRequest struct {
	Hidden bool `json:"hidden"`
}

func (c *Client) GetActivitySettings(ctx context.Context, userID int64) (*ActivitySettings, error) {
	var out ActivitySettings
	err := c.do(ctx, http.MethodGet, "/users/"+itoa(userID)+"/activity-settings", nil, &out)
	return &out, err
}

func (c *Client) PutActivitySettings(ctx context.Context, userID int64, hidden bool) (*ActivitySettings, error) {
	var out ActivitySettings
	err := c.do(ctx, http.MethodPut, "/users/"+itoa(userID)+"/activity-settings", ActivitySettingsRequest{Hidden: hidden}, &out)
	return &out, err
}
