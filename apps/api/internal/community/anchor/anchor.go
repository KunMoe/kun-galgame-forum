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
