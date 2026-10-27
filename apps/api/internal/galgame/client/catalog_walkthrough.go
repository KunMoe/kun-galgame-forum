package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"kun-galgame-api/pkg/errors"
)

type CatalogWalkthrough struct {
	ID               string                     `json:"id"`
	WorkID           string                     `json:"work_id"`
	WorkContentLimit string                     `json:"work_content_limit"`
	WorkRouteHints   bool                       `json:"work_route_hints"`
	Release          *CatalogWalkthroughRelease `json:"release"`
	Source           string                     `json:"source"`
	SourceURL        *string                    `json:"source_url"`
	SourceAuthor     *string                    `json:"source_author"`
	SourceUpdatedOn  *string                    `json:"source_updated_on"`
	Lang             string                     `json:"lang"`
	Characters       []CatalogWalkthroughPerson `json:"characters"`
	EndingCount      int                        `json:"ending_count"`
	SaveCount        int                        `json:"save_count"`
	SegmentCount     int                        `json:"segment_count"`
	UpdatedAt        string                     `json:"updated_at"`
	Chart            *CatalogWalkthroughChart   `json:"chart"`
}

type CatalogWalkthroughRelease struct {
	ID          string   `json:"id"`
	Title       *string  `json:"title"`
	ReleaseKind string   `json:"release_kind"`
	Lang        *string  `json:"lang"`
	Companies   []string `json:"companies"`
}

type CatalogWalkthroughPerson struct {
	Key         string                      `json:"key"`
	Name        string                      `json:"name"`
	CharacterID *string                     `json:"character_id"`
	DisplayName *string                     `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
}

type CatalogWalkthroughChart struct {
	Sections []struct {
		Key           string                      `json:"key"`
		Title         string                      `json:"title"`
		Localized     map[string]catLocalizedName `json:"localized"`
		CharacterKeys []string                    `json:"character_keys"`
	} `json:"sections"`
	Entries []struct {
		Key                 string  `json:"key"`
		Label               string  `json:"label"`
		UnlockedByEndingKey *string `json:"unlocked_by_ending_key"`
	} `json:"entries"`
	Saves []struct {
		Key   string `json:"key"`
		Label string `json:"label"`
	} `json:"saves"`
	Endings []struct {
		Key           string                      `json:"key"`
		Name          string                      `json:"name"`
		Localized     map[string]catLocalizedName `json:"localized"`
		EndingType    string                      `json:"ending_type"`
		CharacterKeys []string                    `json:"character_keys"`
	} `json:"endings"`
	Segments []CatalogWalkthroughSegment `json:"segments"`
}

type CatalogWalkthroughNode struct {
	NodeType string  `json:"node_type"`
	Key      *string `json:"key"`
}

type CatalogWalkthroughSegment struct {
	Key                string                      `json:"key"`
	SectionKey         *string                     `json:"section_key"`
	FromNode           CatalogWalkthroughNode      `json:"from_node"`
	Condition          *string                     `json:"condition"`
	ConditionLocalized map[string]catLocalizedName `json:"condition_localized"`
	Steps              []struct {
		StepType     string                      `json:"step_type"`
		Text         *string                     `json:"text"`
		Localized    map[string]catLocalizedName `json:"localized"`
		Alternatives []struct {
			Text      string                      `json:"text"`
			Localized map[string]catLocalizedName `json:"localized"`
		} `json:"alternatives"`
		Achievement bool    `json:"achievement"`
		SaveKey     *string `json:"save_key"`
	} `json:"steps"`
	ToNode CatalogWalkthroughNode `json:"to_node"`
}

type CatalogWalkthroughPage struct {
	Items      []CatalogWalkthrough `json:"items"`
	NextCursor *string              `json:"next_cursor"`
}

func (c *GalgameClient) CatalogWalkthroughs(ctx context.Context, q url.Values) (*CatalogWalkthroughPage, *errors.AppError) {
	var page CatalogWalkthroughPage
	if appErr := c.getV2Native(ctx, "/v2/catalog/walkthroughs", q, &page); appErr != nil {
		return nil, appErr
	}
	return &page, nil
}

func (c *GalgameClient) CatalogWalkthrough(ctx context.Context, id string) (*CatalogWalkthrough, bool, *errors.AppError) {
	status, body, appErr := c.getV2(ctx, "/v2/catalog/walkthroughs/"+id, openPopulation(url.Values{}))
	if appErr != nil {
		return nil, false, appErr
	}
	if status == http.StatusNotFound {
		return nil, false, nil
	}
	if status < 200 || status >= 300 {
		return nil, false, c.v2Error(status, body)
	}
	var w CatalogWalkthrough
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, false, errors.ErrInternal("解析 Catalog 响应失败")
	}
	return &w, true, nil
}
