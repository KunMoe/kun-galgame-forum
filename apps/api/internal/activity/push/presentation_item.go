package push

import (
	"context"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/userclient"
)

const (
	anchorKindGame     int16 = 1
	anchorKindResource int16 = 2

	prefixResource = "resource:"
	prefixRating   = "rating:"
	prefixQuiz     = "quiz:"
	prefixToolset  = "toolset:"
	prefixWebsite  = "website:"

	titleGalgame  = "Galgame"
	titleResource = "Galgame 资源"
	titleRating   = "评分"
	titleQuiz     = "Galgame 题目"
	titleToolset  = "工具集"
	titleWebsite  = "网站"
)

func presentationTombstone(k presentationKey, rev int64) communityclient.AnchorPresentationItem {
	return communityclient.AnchorPresentationItem{
		AnchorKind: int32(k.Kind), AnchorID: k.ID, Revision: rev, Removed: true,
	}
}

func (p *Presenter) workRef(ctx context.Context, st *presentationState, workID int) (repr.WorkRef, bool) {
	row, ok := st.catalog[workID]
	if !ok {
		return repr.WorkRef{}, false
	}
	return galgameapiv1.WorkRefOf(ctx, &row, p.cdn), true
}

func (p *Presenter) workTiedItem(k presentationKey, path, rawTitle, fallback string, ref repr.WorkRef, rev int64) communityclient.AnchorPresentationItem {
	item := communityclient.AnchorPresentationItem{
		AnchorKind: int32(k.Kind), AnchorID: k.ID, Revision: rev,
		Title: presentationTitle(rawTitle, fallback), URL: p.origin + path,
		ContentLimit: "sfw",
	}
	if id := mustID(ref.ID); id > 0 {
		item.WorkID = int64(id)
	}
	if hash := coverHash(ref.Cover); hash != "" {
		item.CoverImageHash = hash
	}
	if ref.IsNSFW {
		item.ContentLimit = "nsfw"
	}
	return item
}

func (p *Presenter) livePresentation(ctx context.Context, k presentationKey, st *presentationState, rev int64) (communityclient.AnchorPresentationItem, bool) {
	switch {
	case k.Kind == anchorKindGame:
		id, ok := parseStrictInt(k.ID)
		if !ok {
			return communityclient.AnchorPresentationItem{}, false
		}
		ref, ok := p.workRef(ctx, st, id)
		if !ok {
			return communityclient.AnchorPresentationItem{}, false
		}
		return p.workTiedItem(k, "/galgame/"+k.ID, workName(ref.CatalogName), titleGalgame, ref, rev), true
	case k.Kind == anchorKindResource && strings.HasPrefix(k.ID, prefixResource):
		id, ok := parseStrictInt(strings.TrimPrefix(k.ID, prefixResource))
		if !ok {
			return communityclient.AnchorPresentationItem{}, false
		}
		row, ok := st.resources[id]
		if !ok || !st.published[row.WorkID] {
			return communityclient.AnchorPresentationItem{}, false
		}
		u, present := st.users[row.UserID]
		if !present || !userclient.IsRenderable(u) {
			return communityclient.AnchorPresentationItem{}, false
		}
		ref, ok := p.workRef(ctx, st, row.WorkID)
		if !ok {
			return communityclient.AnchorPresentationItem{}, false
		}
		return p.workTiedItem(k, "/galgame/resource/"+strconv.Itoa(id), workName(ref.CatalogName), titleResource, ref, rev), true
	case k.Kind == anchorKindResource && strings.HasPrefix(k.ID, prefixRating):
		id, ok := parseStrictInt(strings.TrimPrefix(k.ID, prefixRating))
		if !ok {
			return communityclient.AnchorPresentationItem{}, false
		}
		row, ok := st.ratings[id]
		if !ok {
			return communityclient.AnchorPresentationItem{}, false
		}
		if u, present := st.users[row.UserID]; present && !userclient.IsRenderable(u) {
			return communityclient.AnchorPresentationItem{}, false
		}
		ref, ok := p.workRef(ctx, st, row.WorkID)
		if !ok {
			return communityclient.AnchorPresentationItem{}, false
		}
		return p.workTiedItem(k, "/galgame-rating/"+strconv.Itoa(id), workName(ref.CatalogName), titleRating, ref, rev), true
	case k.Kind == anchorKindResource && strings.HasPrefix(k.ID, prefixQuiz):
		return p.liveQuiz(ctx, k, st, rev)
	case k.Kind == anchorKindResource && strings.HasPrefix(k.ID, prefixToolset):
		id, ok := parseStrictInt(strings.TrimPrefix(k.ID, prefixToolset))
		if !ok {
			return communityclient.AnchorPresentationItem{}, false
		}
		row, ok := st.toolsets[id]
		if !ok || row.Status == 1 {
			return communityclient.AnchorPresentationItem{}, false
		}
		u, present := st.users[row.UserID]
		if !present || !userclient.IsRenderable(u) {
			return communityclient.AnchorPresentationItem{}, false
		}
		return communityclient.AnchorPresentationItem{
			AnchorKind: int32(k.Kind), AnchorID: k.ID, Revision: rev,
			Title: presentationTitle(row.Name, titleToolset), URL: p.origin + "/toolset/" + strconv.Itoa(id),
			ContentLimit: "sfw",
		}, true
	case k.Kind == anchorKindResource && strings.HasPrefix(k.ID, prefixWebsite):
		id, ok := parseStrictInt(strings.TrimPrefix(k.ID, prefixWebsite))
		if !ok {
			return communityclient.AnchorPresentationItem{}, false
		}
		row, ok := st.websites[id]
		if !ok {
			return communityclient.AnchorPresentationItem{}, false
		}
		limit := "sfw"
		if row.AgeLimit != "all" {
			limit = "nsfw"
		}
		return communityclient.AnchorPresentationItem{
			AnchorKind: int32(k.Kind), AnchorID: k.ID, Revision: rev,
			Title: presentationTitle(row.Name, titleWebsite), URL: p.origin + "/website/" + row.URL,
			ContentLimit: limit,
		}, true
	}
	return communityclient.AnchorPresentationItem{}, false
}

func (p *Presenter) liveQuiz(ctx context.Context, k presentationKey, st *presentationState, rev int64) (communityclient.AnchorPresentationItem, bool) {
	id, ok := parseStrictInt64(strings.TrimPrefix(k.ID, prefixQuiz))
	if !ok {
		return communityclient.AnchorPresentationItem{}, false
	}
	row, ok := st.quizzes[id]
	if !ok {
		return communityclient.AnchorPresentationItem{}, false
	}
	u, present := st.users[row.UserID]
	if !present || !userclient.IsRenderable(u) {
		return communityclient.AnchorPresentationItem{}, false
	}
	item := communityclient.AnchorPresentationItem{
		AnchorKind: int32(k.Kind), AnchorID: k.ID, Revision: rev,
		Title:        presentationTitle(content.PlainText(row.Question, 1000), titleQuiz),
		URL:          p.origin + "/galgame-quiz/" + strconv.FormatInt(id, 10),
		ContentLimit: "sfw",
	}
	var returned []repr.WorkRef
	for _, wid := range st.quizWorks[id] {
		ref, ok := p.workRef(ctx, st, wid)
		if !ok {
			continue
		}
		returned = append(returned, ref)
		if ref.IsNSFW {
			item.ContentLimit = "nsfw"
		}
	}
	if !row.HideGalgame && len(returned) == 1 {
		ref := returned[0]
		if id := mustID(ref.ID); id > 0 {
			item.WorkID = int64(id)
		}
		if hash := coverHash(ref.Cover); hash != "" {
			item.CoverImageHash = hash
		}
	}
	return item, true
}
