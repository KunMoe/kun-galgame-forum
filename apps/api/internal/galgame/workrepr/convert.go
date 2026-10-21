package workrepr

import (
	"cmp"
	"context"
	"regexp"
	"slices"
	"strings"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/imageclient"
)

func Name(displayName, latin string, localized map[string]client.LocalizedValue) repr.CatalogName {
	out := make(map[string]repr.LocalizedName, len(localized))
	for tag, n := range localized {
		out[tag] = repr.LocalizedName{Value: n.Value, IsMachine: n.Machine}
	}
	if displayName == "" {
		displayName = firstLocalized(out, latin)
	}
	return repr.NewCatalogName(displayName, latin, out)
}

func firstLocalized(localized map[string]repr.LocalizedName, latin string) string {
	if latin != "" {
		return latin
	}
	for _, tag := range []string{"zh-Hans", "zh", "zh-Hant", "ja", "en"} {
		if n, ok := localized[tag]; ok {
			return n.Value
		}
	}
	for _, n := range localized {
		return n.Value
	}
	return ""
}

// Ref reads is_nsfw from the same derivation that fills the local
// galgame.content_limit cache (migration 079), so a ref never disagrees with the
// forum's own SFW filter; cover is the portrait slot's original, not _mini.
func Ref(ctx context.Context, it *client.CatalogWorkListItem, cdn string) repr.WorkRef {
	brief := client.CatalogItemToBrief(ctx, it)
	var sexual *int
	if slot := portraitSlot(it); slot != nil {
		sexual = slot.Sexual
	}
	cover := ImageFromURL(cdn, brief.EffectivePortraitURL, brief.EffectivePortraitWidth,
		brief.EffectivePortraitHeight, brief.EffectivePortraitThumbhash, sexual)
	name := Name(it.DisplayName, it.Latin, client.LocalizedValues(it.Localized))
	return repr.NewWorkRef(int(it.ID), name, cover, brief.ContentLimit == "nsfw")
}

func Banner(it *client.CatalogWorkListItem, cdn string) *repr.Image {
	slots := it.CoverSlots
	if slots == nil {
		slots = it.Covers
	}
	if slots == nil || slots.Banner == nil {
		return nil
	}
	b := slots.Banner
	return ImageFromURL(cdn, b.URL, b.Width, b.Height, b.Thumbhash, b.Sexual)
}

func portraitSlot(it *client.CatalogWorkListItem) *client.CoverSlot {
	slots := it.CoverSlots
	if slots == nil {
		slots = it.Covers
	}
	if slots == nil {
		return nil
	}
	return slots.Portrait
}

func Maker(it *client.CatalogWorkListItem) *CompanyRef {
	l := it.MakerLabel()
	if l == nil {
		return nil
	}
	ref := CompanyRefOf(l.ID, l.DisplayName, "", client.LocalizedValues(l.Localized))
	return &ref
}

// TopTags are the first n of a work's tags in the order its page lists them,
// folded per canonical tag the way the page folds them, without spoilers, and
// without adult tags unless includeNSFW.
func TopTags(it *client.CatalogWorkListItem, includeNSFW bool, n int) []TagRef {
	type folded struct {
		spoiler int
		sexual  bool
		at      int
	}
	seen := map[int64]*folded{}
	order := make([]int, 0, len(it.Tags))
	for i, t := range it.Tags {
		if t.CanonicalID == 0 || t.Tier == client.TagTierHidden {
			continue
		}
		if f, ok := seen[t.CanonicalID]; ok {
			f.spoiler = max(f.spoiler, t.Spoiler)
			f.sexual = f.sexual || t.Sexual
			continue
		}
		seen[t.CanonicalID] = &folded{spoiler: t.Spoiler, sexual: t.Sexual, at: i}
		order = append(order, i)
	}
	out := make([]TagRef, 0, n)
	for _, i := range order {
		t := it.Tags[i]
		f := seen[t.CanonicalID]
		if f.spoiler > 0 || (f.sexual && !includeNSFW) {
			continue
		}
		if len(out) == n {
			break
		}
		out = append(out, TagRef{
			Object: "tag", ID: repr.ID(int(t.CanonicalID)), IsSexual: f.sexual,
			CatalogName: Name(cmp.Or(t.DisplayName, t.Name), "", client.LocalizedValues(t.Localized)),
		})
	}
	return out
}

func CompanyRefOf(id int64, displayName, latin string, localized map[string]client.LocalizedValue) CompanyRef {
	return CompanyRef{Object: "company", ID: repr.ID(int(id)), CatalogName: Name(displayName, latin, localized)}
}

var releaseDigits = regexp.MustCompile(`^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$`)

func Release(date *string) (*string, *string) {
	if date == nil || !releaseDigits.MatchString(*date) {
		return nil, nil
	}
	d := *date
	var full, precision string
	switch len(d) {
	case 4:
		full, precision = d+"-01-01", "year"
	case 7:
		full, precision = d+"-01", "month"
	default:
		full, precision = d, "day"
	}
	if strings.HasSuffix(full, "-00") || strings.Contains(full, "-00-") {
		return nil, nil
	}
	return &full, &precision
}

var (
	langTag  = regexp.MustCompile(`^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$`)
	sourceID = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
)

// Lang is nil for an empty or malformed tag.
func Lang(tag string) *string {
	if len(tag) > 35 || !langTag.MatchString(tag) {
		return nil
	}
	return &tag
}

func Intros(rows []client.CatalogIntro) []CatalogIntro {
	out := make([]CatalogIntro, 0, len(rows))
	for _, r := range rows {
		if strings.TrimSpace(r.Intro) == "" || Lang(r.Lang) == nil {
			continue
		}
		in := CatalogIntro{Locale: r.Lang, Value: r.Intro, IsMachine: r.Machine}
		if src := strings.ToLower(r.Source); len(src) <= 64 && sourceID.MatchString(src) {
			in.DataSource = &src
		}
		out = append(out, in)
	}
	return out
}

var pageID = regexp.MustCompile(`^[a-z]?[0-9]{1,20}$`)

// AppendLink skips a link that names a site but no address: there is nothing to open.
// It also skips an address already in out: before 2.52.0 catalog sent some twice
// (company 6450's official_site and cien were both ci-en.dlsite.com/creator/65), and
// pages key links by URL.
func AppendLink(out []CatalogLink, l client.CatalogLink) []CatalogLink {
	site := strings.ToLower(l.Source)
	if l.URL == "" || len(site) > 64 || !sourceID.MatchString(site) {
		return out
	}
	if slices.ContainsFunc(out, func(o CatalogLink) bool { return o.URL == l.URL }) {
		return out
	}
	link := CatalogLink{Site: site, URL: l.URL, IsIdentity: l.IsIdentity}
	if l.IsIdentity && pageID.MatchString(l.PageID) {
		link.PageID = &l.PageID
	}
	return append(out, link)
}

func ImageFromURL(cdn, url string, width, height int, thumbhash string, sexual *int) *repr.Image {
	if url == "" {
		return nil
	}
	hash := url
	if i := strings.LastIndexByte(hash, '/'); i >= 0 {
		hash = hash[i+1:]
	}
	hash = strings.TrimSuffix(hash, ".webp")
	meta := &imageclient.ImageMeta{Width: width, Height: height, Thumbhash: thumbhash}
	if sexual != nil {
		g := int16(*sexual)
		meta.Sexual = &g
	}
	return repr.NewImage(cdn, hash, meta)
}

func ImageFromHash(cdn, hash string) *repr.Image {
	if hash == "" {
		return nil
	}
	return repr.NewImage(cdn, hash, nil)
}
