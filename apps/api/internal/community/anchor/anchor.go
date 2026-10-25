package anchor

import (
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/communityclient"

	"gorm.io/gorm"
)

// Ref is a comment wall's anchor as the community service reports it.
type Ref struct {
	Kind int32
	ID   string
}

// Target is the forum page that hosts a wall, ready to render as a result row.
type Target struct {
	Link  string
	Label string
	// Title is the game's name, and only ResolveNamed fills it: the galgame
	// table caches no title, because catalog owns the name.
	Title string
	// WorkID is set only for a site_game wall; the galgame comment surfaces
	// key off it.
	WorkID int
}

// site_resource packs the source into the anchor id as a prefix, because one
// anchor kind carries five different kinds of page.
var resourcePrefixes = map[string]struct {
	linkPrefix string
	label      string
}{
	"rating":   {"/galgame-rating/", "游戏评分"},
	"toolset":  {"/toolset/", "Gal 工具"},
	"resource": {"/galgame/resource/", "下载资源"},
	"quiz":     {"/galgame-quiz/", "游戏答题"},
	// /galgame-walkthrough/:id redirects to /galgame/:work/walkthrough/:id; the
	// anchor carries no work id.
	"walkthrough": {"/galgame-walkthrough/", "游戏攻略"},
}

// Resolver turns anchors back into links. A website wall is the one that needs
// the database: /website/:domain is keyed by the site's URL, not by its id.
type Resolver struct {
	db      *gorm.DB
	galgame *client.GalgameClient
}

func New(db *gorm.DB, galgame *client.GalgameClient) *Resolver {
	return &Resolver{db: db, galgame: galgame}
}

// Resolve maps every anchor it recognises. An anchor it does not — a catalog
// wall another site opened, a source this forum has since retired — is left out
// rather than linked to a page that would 404.
func (r *Resolver) Resolve(refs []Ref) map[Ref]Target {
	out := make(map[Ref]Target, len(refs))
	websiteIDs := make([]int, 0)
	websiteRefs := make(map[int][]Ref)

	for _, ref := range refs {
		if _, done := out[ref]; done {
			continue
		}
		switch ref.Kind {
		case communityclient.AnchorSiteGame:
			workID, err := strconv.Atoi(ref.ID)
			if err != nil || workID <= 0 {
				continue
			}
			out[ref] = Target{Link: "/galgame/" + ref.ID, Label: "Galgame", WorkID: workID}
		case communityclient.AnchorSiteResource:
			source, rawID, ok := strings.Cut(ref.ID, ":")
			id, err := strconv.Atoi(rawID)
			if !ok || err != nil || id <= 0 {
				continue
			}
			if source == "website" {
				if _, seen := websiteRefs[id]; !seen {
					websiteIDs = append(websiteIDs, id)
				}
				websiteRefs[id] = append(websiteRefs[id], ref)
				continue
			}
			page, known := resourcePrefixes[source]
			if !known {
				continue
			}
			out[ref] = Target{Link: page.linkPrefix + rawID, Label: page.label}
		}
	}

	for id, domain := range r.websiteDomains(websiteIDs) {
		for _, ref := range websiteRefs[id] {
			out[ref] = Target{Link: "/website/" + domain, Label: "网站"}
		}
	}
	return out
}

// OnHiddenWorks stays apart from Resolve: the notification poller drops, for
// good, a notification whose anchor Resolve leaves out.
func (r *Resolver) OnHiddenWorks(refs []Ref) (map[Ref]bool, error) {
	type key struct {
		source string
		id     int
	}
	ids := map[string][]int{"game": nil, "resource": nil, "rating": nil, "quiz": nil, "walkthrough": nil}
	byKey := map[key][]Ref{}
	for _, ref := range refs {
		var k key
		switch ref.Kind {
		case communityclient.AnchorSiteGame:
			k.source = "game"
			k.id, _ = strconv.Atoi(ref.ID)
		case communityclient.AnchorSiteResource:
			source, rawID, _ := strings.Cut(ref.ID, ":")
			k.source = source
			k.id, _ = strconv.Atoi(rawID)
		}
		if _, tied := ids[k.source]; !tied || k.id <= 0 {
			continue
		}
		if _, seen := byKey[k]; !seen {
			ids[k.source] = append(ids[k.source], k.id)
		}
		byKey[k] = append(byKey[k], ref)
	}
	out := map[Ref]bool{}
	if len(byKey) == 0 {
		return out, nil
	}
	var rows []struct {
		Source string `gorm:"column:source"`
		ID     int    `gorm:"column:id"`
	}
	err := r.db.Raw(`
		SELECT 'game' AS source, id FROM galgame WHERE id IN @game AND NOT catalog_rendered
		UNION ALL
		SELECT 'resource', t.id FROM galgame_resource t JOIN galgame g ON g.id = t.work_id
		WHERE t.id IN @resource AND NOT g.catalog_rendered
		UNION ALL
		SELECT 'rating', t.id FROM galgame_rating t JOIN galgame g ON g.id = t.work_id
		WHERE t.id IN @rating AND NOT g.catalog_rendered
		UNION ALL
		SELECT DISTINCT 'quiz', t.quiz_id FROM galgame_quiz_galgame t JOIN galgame g ON g.id = t.work_id
		WHERE t.quiz_id IN @quiz AND NOT g.catalog_rendered
		UNION ALL
		SELECT DISTINCT 'walkthrough', t.id FROM (
			SELECT substring(fa.link FROM '^/galgame-walkthrough/([0-9]{1,18})$')::bigint AS id, fa.work_id
			FROM feed_activity fa WHERE fa.type = 'GALGAME_WALKTHROUGH_COMMENT_CREATION'
		) t JOIN galgame g ON g.id = t.work_id
		WHERE t.id IN @walkthrough AND NOT g.catalog_rendered`,
		map[string]any{"game": ids["game"], "resource": ids["resource"], "rating": ids["rating"], "quiz": ids["quiz"], "walkthrough": ids["walkthrough"]},
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		for _, ref := range byKey[key{row.Source, row.ID}] {
			out[ref] = true
		}
	}
	return out, nil
}

func (r *Resolver) websiteDomains(ids []int) map[int]string {
	if len(ids) == 0 {
		return nil
	}
	var rows []struct {
		ID  int    `gorm:"column:id"`
		URL string `gorm:"column:url"`
	}
	if err := r.db.Table("galgame_website").Select("id, url").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil
	}
	domains := make(map[int]string, len(rows))
	for _, row := range rows {
		if row.URL != "" {
			domains[row.ID] = row.URL
		}
	}
	return domains
}
