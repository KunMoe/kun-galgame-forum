package workrepr

import (
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
)

type TagExclusionInput struct {
	ExcludeTagID   []repr.DecimalID `query:"exclude_tag_ids" maxItems:"100" doc:"Catalog tag ids. Drops every work carrying any of them, from any source and at any spoiler level. Comma-separated. Omitted means no exclusion."`
	ExcludeTagTree []repr.DecimalID `query:"exclude_tag_trees" maxItems:"100" doc:"Catalog tag ids. Like exclude_tag_ids, and each also drops the works carrying any tag below it in the tag hierarchy. Comma-separated. Omitted means no exclusion."`
}

func (in TagExclusionInput) Exclusion() client.TagExclusion {
	return client.NewTagExclusion(parseTagIDs(in.ExcludeTagID), parseTagIDs(in.ExcludeTagTree))
}

func parseTagIDs(raw []repr.DecimalID) []int64 {
	out := make([]int64, 0, len(raw))
	for _, s := range raw {
		if id, ok := repr.ParseID(s); ok {
			out = append(out, int64(id))
		}
	}
	return out
}
