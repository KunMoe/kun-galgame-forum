package apiv1

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"slices"
	"sort"
	"strings"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/entityapiv1"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) listCollectionEntries(ctx context.Context, in *listCollectionEntriesInput) (*collectionEntryPageOutput, error) {
	folder, token, _, err := s.visibleFolder(ctx, in.CollectionID)
	if err != nil {
		return nil, err
	}
	pg := browsePage(in.Page, in.Limit)
	if prob := pg.CheckDepth(); prob != nil {
		return nil, prob
	}
	if s.hydrator == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	items, iErr := s.loadFolderItems(ctx, folder, token)
	if iErr != nil {
		return nil, mapUserPlane(iErr, token != "")
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CreatedAt != items[j].CreatedAt {
			return items[i].CreatedAt > items[j].CreatedAt
		}
		return items[i].WorkID > items[j].WorkID
	})
	byWork := make(map[int]catalogclient.FolderItem, len(items))
	for _, it := range items {
		byWork[int(it.WorkID)] = it
	}
	population, p := s.folderPopulation(ctx, folder, items, in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	var ordered []folderMember
	if in.Sort == "position_asc" {
		ordered = slices.Clone(population)
		slices.SortStableFunc(ordered, func(a, b folderMember) int {
			return cmp.Compare(byWork[a.WorkID].Position, byWork[b.WorkID].Position)
		})
	} else {
		ordered = sortFolderMembers(population, in.Sort)
	}
	n, rel := collect.ClampTotal(len(ordered))
	start := pg.Offset()
	pageIDs := []int{}
	if start < len(ordered) {
		for _, m := range ordered[start:min(start+pg.Limit, len(ordered))] {
			pageIDs = append(pageIDs, m.WorkID)
		}
	}
	summaries, p := s.hydrator.ByIDs(ctx, pageIDs, in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	details, appErr := s.works.CatalogRowsByWorkIDs(ctx, pageIDs, "tags,credits", workrepr.ContentLimit(in.IncludeNSFW))
	if appErr != nil {
		slog.Warn("collection: entry tags and credits unreadable", "folder_id", folder.ID, "err", appErr)
	}
	entries := make([]CollectionEntry, 0, len(summaries))
	for _, sum := range summaries {
		id, ok := repr.ParseID(sum.ID)
		if !ok {
			continue
		}
		it := byWork[id]
		added, ok := folderTime(it.CreatedAt, folder.ID, "item.created_at")
		if !ok {
			continue
		}
		entry := CollectionEntry{
			Object: "collection_entry", WorkSummary: sum, Note: noteOrNull(it.Note), AddedAt: added,
			TopTags: []workrepr.TagRef{}, ScenarioCredits: []entityapiv1.CreditNameRef{}, IllustrationCredits: []entityapiv1.CreditNameRef{},
		}
		if row, ok := details[id]; ok {
			entry.TopTags = workrepr.TopTags(&row, in.IncludeNSFW, 6)
			entry.ScenarioCredits = creditRefs(row.Credited(3, "scenario"))
			entry.IllustrationCredits = creditRefs(row.Credited(3, "illustration", "character-design"))
		}
		entries = append(entries, entry)
	}
	return &collectionEntryPageOutput{Body: repr.NewPageList(entries, n, rel)}, nil
}

func creditRefs(credits []client.CatalogCredit) []entityapiv1.CreditNameRef {
	out := make([]entityapiv1.CreditNameRef, 0, len(credits))
	for _, c := range credits {
		out = append(out, entityapiv1.CreditNameRef{
			Object: "credit_name", ID: repr.ID(int(c.ID)),
			CatalogName: workrepr.Name(c.DisplayName, c.Latin, client.LocalizedValues(c.Localized)),
		})
	}
	return out
}

func noteOrNull(note string) *string {
	if note == "" {
		return nil
	}
	return &note
}

func (s *Service) patchCollectionWork(ctx context.Context, in *patchCollectionWorkInput) (*collectionItemOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	body := in.Body
	if body.Note == nil && body.AfterWorkID == nil && body.BeforeWorkID == nil {
		return nil, validationFailed(problem.AtPointer("", problem.ReasonRequired, "send at least one of note, after_work_id, before_work_id", nil))
	}
	if body.AfterWorkID != nil && body.BeforeWorkID != nil {
		return nil, validationFailed(problem.AtPointer("/before_work_id", problem.ReasonNotAllowedValue, "send after_work_id or before_work_id, not both", nil))
	}
	folderID, ok := parseCollectionID(in.CollectionID)
	if !ok {
		return nil, notFound()
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	if body.Note != nil {
		trimmed := strings.TrimSpace(*body.Note)
		body.Note = &trimmed
		if _, p := s.checkCollectionText(ctx, trimmed, user.ID); p != nil {
			return nil, p
		}
	}
	patch := catalogclient.FolderItemPatch{Note: body.Note, AfterWorkID: derefOr(body.AfterWorkID, ""), BeforeWorkID: derefOr(body.BeforeWorkID, "")}
	item, err := s.catalog.PatchFolderItem(ctx, token, folderID, int64(workID), patch)
	if err != nil && upstreamNotFound(err) &&
		body.Note != nil && body.AfterWorkID == nil && body.BeforeWorkID == nil && user.Can(perm.CollectionEditAny) {
		item, err = s.catalog.ModeratePatchFolderItem(ctx, token, folderID, int64(workID), *body.Note)
	}
	if err != nil {
		return nil, mapUserPlaneAt(err, true, itemPointer)
	}
	added, ok := folderTime(item.CreatedAt, folderID, "item.created_at")
	if !ok {
		return nil, problem.Unavailable(errors.New("catalog folder item: created_at unparseable"))
	}
	return &collectionItemOutput{Body: CollectionItem{
		Object: "collection_item", CollectionID: repr.ID(int(folderID)), WorkID: repr.ID(workID),
		Note: noteOrNull(item.Note), AddedAt: added,
	}}, nil
}

func itemPointer(upstream string) (string, bool) {
	switch upstream {
	case "", "/note", "/after_work_id", "/before_work_id":
		return upstream, true
	}
	return "", false
}

func collectionPointer(upstream string) (string, bool) {
	switch upstream {
	case "/name":
		return "/title", true
	case "/description", "/visibility", "/is_default", "/cover_work_id":
		return upstream, true
	case "/tags":
		return "/tag_names", true
	}
	return "", false
}
