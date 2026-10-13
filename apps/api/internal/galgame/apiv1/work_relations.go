package apiv1

import (
	"cmp"
	"context"
	"slices"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
)

type RelatedWork struct {
	repr.WorkRef
	Medium               string         `json:"medium" enum:"galgame,manga,novel,anime,asmr,doujin_game,music" maxLength:"11" doc:"What kind of work it is. Only a galgame has a page on this forum; link nothing else."`
	ReleaseDate          *string        `json:"release_date" format:"date" maxLength:"10" doc:"Release date. A month- or year-precise date is the first day of that month or year. null when catalog has none."`
	ReleaseDatePrecision *string        `json:"release_date_precision" enum:"day,month,year" maxLength:"5" doc:"How much of release_date is known. null when release_date is null."`
	Relations            []WorkRelation `json:"relations" minItems:"1" doc:"How the work relates to this one, most significant first. Never empty."`
}

type WorkRelation struct {
	RelationType string `json:"relation_type" maxLength:"64" pattern:"^[a-z][a-z0-9_]*$" doc:"The relation token, such as sequel_of or fandisc_of. An open vocabulary; display phrase instead."`
	Phrase       string `json:"phrase" maxLength:"512" doc:"The relation in Chinese as catalog words it, from this work's side. Display it as is. Free text; never use it as a decision input."`
}

var relationRank = map[string]int{
	"sequel_of":           0,
	"fandisc_of":          1,
	"side_story_of":       2,
	"remake_of":           3,
	"alternative_version": 4,
	"collects":            5,
	"same_series":         6,
	"same_setting":        7,
	"alternative_setting": 8,
	"crossover_with":      9,
	"shares_character":    10,
	"adaptation_of":       11,
}

var relatedMedia = map[string]bool{
	"galgame": true, "manga": true, "novel": true, "anime": true,
	"asmr": true, "doujin_game": true, "music": true,
}

func rankOf(relationType string) int {
	if r, ok := relationRank[relationType]; ok {
		return r
	}
	return len(relationRank)
}

func relatedWorksOf(ctx context.Context, d *client.CatalogWorkDetail, cdn string) []RelatedWork {
	type entry struct {
		id   int64
		work RelatedWork
	}
	var entries []entry
	at := map[int64]int{}
	for i := range d.Relations {
		rel := &d.Relations[i]
		item := &rel.Work.CatalogWorkListItem
		if item.ID <= 0 || !relatedMedia[item.Medium] || !client.CatalogItemRenderable(item) {
			continue
		}
		edge := WorkRelation{RelationType: rel.RelationType, Phrase: rel.Phrase}
		if idx, seen := at[item.ID]; seen {
			if w := &entries[idx].work; !slices.Contains(w.Relations, edge) {
				w.Relations = append(w.Relations, edge)
			}
			continue
		}
		date, precision := workrepr.Release(item.ReleaseDate)
		if date != nil && rel.Work.ReleaseDatePrecision != nil {
			switch p := *rel.Work.ReleaseDatePrecision; p {
			case "day", "month", "year":
				precision = &p
			}
		}
		at[item.ID] = len(entries)
		entries = append(entries, entry{id: item.ID, work: RelatedWork{
			WorkRef:              workrepr.Ref(ctx, item, cdn),
			Medium:               item.Medium,
			ReleaseDate:          date,
			ReleaseDatePrecision: precision,
			Relations:            []WorkRelation{edge},
		}})
	}
	for i := range entries {
		slices.SortStableFunc(entries[i].work.Relations, func(a, b WorkRelation) int {
			return cmp.Compare(rankOf(a.RelationType), rankOf(b.RelationType))
		})
	}
	slices.SortStableFunc(entries, func(a, b entry) int {
		return cmp.Or(
			cmp.Compare(rankOf(a.work.Relations[0].RelationType), rankOf(b.work.Relations[0].RelationType)),
			compareReleaseDate(a.work.ReleaseDate, b.work.ReleaseDate),
			cmp.Compare(a.id, b.id),
		)
	})
	out := make([]RelatedWork, len(entries))
	for i := range entries {
		out[i] = entries[i].work
	}
	return out
}

func compareReleaseDate(a, b *string) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	}
	return cmp.Compare(*a, *b)
}
