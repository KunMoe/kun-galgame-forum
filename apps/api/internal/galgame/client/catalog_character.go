package client

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"kun-galgame-api/pkg/errors"
)

const catalogCharacterWorksCap = 50

type CatalogCharacter struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Lang        string                      `json:"lang"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Latin       string                      `json:"latin"`
	Image       string                      `json:"image"`
	Figure      string                      `json:"figure"`
	ImageMeta   ArtMeta                     `json:"-"`
	FigureMeta  ArtMeta                     `json:"-"`
	Intros      []CatalogIntro              `json:"intros"`
	Traits      []CatalogCharacterTrait     `json:"traits"`
	Refs        []catRef                    `json:"refs"`
	Works       []struct {
		Work   catWorkBrief    `json:"work"`
		Voices []CatalogPerson `json:"voices"`
	} `json:"works"`
	NextOffset *int `json:"next_offset"`
}

// CatalogCharacterTrait carries two generations of the same Chinese name. Wave
// 212 wave A put the entity name primitive on traits and marked name_zh /
// group_zh superseded; wave B deletes them, and since they are the only reason
// a trait renders 金发 rather than Blonde, reading them alone would turn every
// character's traits back to English on catalog's deploy without erroring.
type CatalogCharacterTrait struct {
	ID int64 `json:"id"`
	// The vocabulary name is `name` on v1 and `display_name` on v2. Every sibling
	// entity carries both and prefers the v2 spelling; this one did not, so an
	// untranslated trait rendered as an empty chip instead of Blonde.
	Name           string                      `json:"name"`
	DisplayName    string                      `json:"display_name"`
	Localized      map[string]catLocalizedName `json:"localized"`
	NameZh         string                      `json:"name_zh"`
	Group          string                      `json:"group"`
	GroupLocalized map[string]catLocalizedName `json:"group_localized"`
	GroupZh        string                      `json:"group_zh"`
	Spoiler        int                         `json:"spoiler"`
	Sexual         bool                        `json:"sexual"`
	Lie            bool                        `json:"lie"`
}

func (c *GalgameClient) CatalogCharacterDetail(
	ctx context.Context, id int64, limit, offset int, withWorks bool,
) (*CatalogCharacter, bool, int64, *errors.AppError) {
	if limit <= 0 || limit > catalogCharacterWorksCap {
		limit = catalogCharacterWorksCap
	}
	q := url.Values{
		"spoilers": {strconv.Itoa(catalogSpoilerCeiling)},
		"limit":    {strconv.Itoa(limit)},
		"offset":   {strconv.Itoa(max(offset, 0))},
	}
	if withWorks {
		q.Set("include", "works")
	}
	openPopulation(q)

	data, found, movedTo, appErr := c.catalogGetRecord(ctx, "/catalog/characters/"+strconv.FormatInt(id, 10), q)
	if appErr != nil || !found {
		return nil, false, movedTo, appErr
	}
	var ch CatalogCharacter
	if err := json.Unmarshal(data, &ch); err != nil {
		return nil, false, 0, errors.ErrInternal("解析 Catalog 角色详情响应失败")
	}
	if meta := c.resolveArtMeta([]string{ch.Image, ch.Figure}); meta != nil {
		ch.ImageMeta, ch.FigureMeta = meta[ch.Image], meta[ch.Figure]
	}
	return &ch, true, 0, nil
}
