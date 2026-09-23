package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/imageclient"
)

func CatalogNameOf(it *client.CatalogWorkListItem) repr.CatalogName {
	localized := make(map[string]repr.LocalizedName, len(it.Localized))
	for tag, n := range it.Localized {
		if n.Value != "" {
			localized[tag] = repr.LocalizedName{Value: n.Value, IsMachine: n.Machine}
		}
	}
	return repr.NewCatalogName(it.DisplayName, it.Latin, localized)
}

// WorkRefOf reads is_nsfw from the same derivation that fills the local
// galgame.content_limit cache (migration 079), so a ref never disagrees with the
// forum's own SFW filter; cover is the portrait slot's original, not _mini.
func WorkRefOf(ctx context.Context, it *client.CatalogWorkListItem, cdnBase string) repr.WorkRef {
	brief := client.CatalogItemToBrief(ctx, it)
	var cover *repr.Image
	if brief.EffectivePortraitHash != "" {
		cover = repr.NewImage(cdnBase, brief.EffectivePortraitHash, &imageclient.ImageMeta{
			Width:     brief.EffectivePortraitWidth,
			Height:    brief.EffectivePortraitHeight,
			Thumbhash: brief.EffectivePortraitThumbhash,
		})
	}
	return repr.NewWorkRef(int(it.ID), CatalogNameOf(it), cover, brief.ContentLimit == "nsfw")
}
