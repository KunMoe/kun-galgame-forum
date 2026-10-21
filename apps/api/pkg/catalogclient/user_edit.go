package catalogclient

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

type UserEditCreateRequest struct {
	EntityType string         `json:"entity_type"`
	EntityID   int64          `json:"entity_id"`
	Patch      map[string]any `json:"patch"`
	Note       string         `json:"note,omitempty"`
}

func (c *Client) EditSnapshotUser(ctx context.Context, accessToken, entityType string, entityID int64) (map[string]any, error) {
	object := "work"
	if strings.HasPrefix(entityType, "catalog.") {
		object = strings.TrimPrefix(entityType, "catalog.")
		if object == "label" {
			object = "company"
		}
	}
	var snap v2Snapshot
	err := c.userV2JSON(ctx, http.MethodGet, accessToken,
		"/v2/moderation/snapshots/"+object+"/"+strconv.FormatInt(entityID, 10), nil, &snap, nil)
	if err != nil {
		return nil, err
	}
	return snap.values(), nil
}
