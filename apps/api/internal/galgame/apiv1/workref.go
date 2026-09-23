package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
)

func WorkRefOf(ctx context.Context, it *client.CatalogWorkListItem, cdnBase string) repr.WorkRef {
	return workrepr.Ref(ctx, it, cdnBase)
}
