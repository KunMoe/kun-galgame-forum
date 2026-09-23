package workrepr

import (
	"context"
	"regexp"
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

func Ref(ctx context.Context, it *client.CatalogWorkListItem, cdn string) repr.WorkRef {
	brief := client.CatalogItemToBrief(ctx, it)
	cover := ImageFromURL(cdn, brief.EffectivePortraitURL, brief.EffectivePortraitWidth,
		brief.EffectivePortraitHeight, brief.EffectivePortraitThumbhash)
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
	return ImageFromURL(cdn, b.URL, b.Width, b.Height, b.Thumbhash)
}

func Maker(it *client.CatalogWorkListItem) *CompanyRef {
	l := it.MakerLabel()
	if l == nil {
		return nil
	}
	ref := CompanyRefOf(l.ID, l.DisplayName, "", client.LocalizedValues(l.Localized))
	return &ref
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

// AppendLink skips a link that names a site but no address: there is nothing to open.
func AppendLink(out []CatalogLink, site, url string) []CatalogLink {
	site = strings.ToLower(site)
	if url == "" || len(site) > 64 || !sourceID.MatchString(site) {
		return out
	}
	return append(out, CatalogLink{Site: site, URL: url})
}

func ImageFromURL(cdn, url string, width, height int, thumbhash string) *repr.Image {
	if url == "" {
		return nil
	}
	hash := url
	if i := strings.LastIndexByte(hash, '/'); i >= 0 {
		hash = hash[i+1:]
	}
	hash = strings.TrimSuffix(hash, ".webp")
	return repr.NewImage(cdn, hash, &imageclient.ImageMeta{Width: width, Height: height, Thumbhash: thumbhash})
}

func ImageFromHash(cdn, hash string) *repr.Image {
	if hash == "" {
		return nil
	}
	return repr.NewImage(cdn, hash, nil)
}
