package client

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/namepref"
)

const catalogNameCreditsCap = 50

// catLocalizedName is one locale slot of the public face's name primitive.
// Machine marks a machine-translated fill-in: since wave 209 such a row may
// occupy a locale that has no source-provenance name, so a rendered name can
// itself be machine text.
type catLocalizedName struct {
	Value   string `json:"value"`
	Kind    string `json:"kind"`
	Machine bool   `json:"machine"`
}

func (n *catLocalizedName) UnmarshalJSON(b []byte) error {
	var aux struct {
		Value     string `json:"value"`
		Kind      string `json:"kind"`
		Machine   bool   `json:"machine"`
		IsMachine bool   `json:"is_machine"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	n.Value = aux.Value
	n.Kind = aux.Kind
	n.Machine = aux.Machine || aux.IsMachine
	return nil
}

type LocalizedValue struct {
	Value   string
	Machine bool
}

// LocalizedValues hands a localized{} block to code outside this package, which
// cannot name its element type. Empty values are dropped.
func LocalizedValues(m map[string]catLocalizedName) map[string]LocalizedValue {
	out := make(map[string]LocalizedValue, len(m))
	for tag, n := range m {
		if n.Value != "" {
			out[tag] = LocalizedValue{Value: n.Value, Machine: n.Machine}
		}
	}
	return out
}

// CatalogAlias is one row of an entity's alias list. Wave 209 turned these from
// bare strings into rows that carry their own language and provenance.
type CatalogAlias struct {
	Value   string `json:"value"`
	Lang    string `json:"lang"`
	Kind    string `json:"kind"`
	Machine bool   `json:"machine"`
}

// catAliases decodes an alias list in either wire shape. The label browse row
// only grew an aliases field after wave 209 had already converted aliases to
// rows, while engines still send bare strings — and one CatalogTaxonomyItem
// covers all four vocabularies, so a []string field there made every
// /catalog/labels read answer "解析 Catalog 词表响应失败", taking /galgame/official
// down with it.
type catAliases []CatalogAlias

func (a *catAliases) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	rows := make(catAliases, 0, len(raw))
	for _, r := range raw {
		r = bytes.TrimSpace(r)
		if len(r) > 0 && r[0] == '"' {
			var value string
			if err := json.Unmarshal(r, &value); err != nil {
				return err
			}
			rows = append(rows, CatalogAlias{Value: value})
			continue
		}
		var row CatalogAlias
		if err := json.Unmarshal(r, &row); err != nil {
			return err
		}
		rows = append(rows, row)
	}
	*a = rows
	return nil
}

// Values flattens the rows for display, minus blanks and whatever name the
// header already shows. Since wave 209 a localized name is an alias row too, so
// a label rendered as 猫猫社 would otherwise list 猫猫社 as its own alias.
func (a catAliases) Values(rendered string) []string {
	out := make([]string, 0, len(a))
	for _, alias := range a {
		if alias.Value == "" || alias.Value == rendered {
			continue
		}
		out = append(out, alias.Value)
	}
	return out
}

// CatalogIntro is the single intro shape the work, character, name, label and
// tag faces have all shared since wave 209. Tag intros never set Machine.
type CatalogIntro struct {
	Lang    string `json:"lang"`
	Intro   string `json:"intro"`
	Source  string `json:"source"`
	Machine bool   `json:"machine"`
}

func (in *CatalogIntro) UnmarshalJSON(b []byte) error {
	var aux struct {
		Lang      string `json:"lang"`
		Intro     string `json:"intro"`
		Value     string `json:"value"`
		Source    string `json:"source"`
		Machine   bool   `json:"machine"`
		IsMachine bool   `json:"is_machine"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	in.Lang = aux.Lang
	in.Intro = aux.Intro
	if in.Intro == "" {
		in.Intro = aux.Value
	}
	in.Source = aux.Source
	in.Machine = aux.Machine || aux.IsMachine
	return nil
}

// introLangOrder is the order the site presents introductions in. Anything
// catalog sends that is not listed keeps its own tag and sorts after these.
var introLangOrder = []string{"zh-Hans", "zh-Hant", "ja", "en"}

// canonicalIntroLang folds the retiring product-locale keys onto the BCP-47 tags
// the detail face already sends, so a work read through the list brief and the
// same work read through the detail agree on what to call a language.
func canonicalIntroLang(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "zh-cn", "zh-hans", "zh":
		return "zh-Hans"
	case "zh-tw", "zh-hant", "zh-hk":
		return "zh-Hant"
	case "ja-jp", "ja":
		return "ja"
	case "en-us", "en":
		return "en"
	default:
		return strings.TrimSpace(lang)
	}
}

// OrderIntros drops blanks, keeps the first row per language and sorts the rest
// into introLangOrder. Rendering is the caller's job — IntroText needs the raw
// markdown, the reader needs it rendered.
func OrderIntros(rows []CatalogIntro) []dto.GalgameIntro {
	out := make([]dto.GalgameIntro, 0, len(rows))
	seen := make(map[string]bool, len(rows))
	for _, r := range rows {
		lang := canonicalIntroLang(r.Lang)
		if r.Intro == "" || lang == "" || seen[lang] {
			continue
		}
		seen[lang] = true
		out = append(out, dto.GalgameIntro{Lang: lang, Intro: r.Intro, Machine: r.Machine})
	}
	rank := func(lang string) int {
		for i, l := range introLangOrder {
			if l == lang {
				return i
			}
		}
		return len(introLangOrder)
	}
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i].Lang) < rank(out[j].Lang) })
	return out
}

// CatalogPerson is the name projection the public face uses wherever a person
// is named inside another record: roster voices, credit rows, siblings.
type CatalogPerson struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Lang        string                      `json:"lang"`
	Latin       string                      `json:"latin"`
	Localized   map[string]catLocalizedName `json:"localized"`
}

func (p *CatalogPerson) Name(ctx context.Context) string {
	return CatalogEntityName(ctx, p.Localized, p.DisplayName, p.Latin)
}

var catalogZhLocales = []string{"zh-Hans", "zh", "zh-Hant"}

type CatalogName struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Lang        string                      `json:"lang"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Latin       string                      `json:"latin"`
	PersonID    int64                       `json:"person_id"`

	PhotoHash string           `json:"photo_hash"`
	Gender    *int             `json:"gender"`
	BirthY    *int             `json:"birth_y"`
	BirthM    *int             `json:"birth_m"`
	BirthD    *int             `json:"birth_d"`
	Siblings  []CatalogPerson  `json:"siblings"`
	Intros    []CatalogIntro   `json:"intros"`
	Refs      []catRef         `json:"refs"`
	Links     []catRelatedLink `json:"links"`
	Credits   []struct {
		Work  catWorkBrief `json:"work"`
		Roles []struct {
			RoleKey     string `json:"role_key"`
			RoleName    string `json:"role_name"`
			CharacterID int64  `json:"character_id"`
			Character   string `json:"character"`
		} `json:"roles"`
	} `json:"credits"`
	NextOffset *int `json:"next_offset"`
}

func (c *GalgameClient) CatalogNameDetail(
	ctx context.Context, id int64, limit, offset int,
) (*CatalogName, bool, int64, *errors.AppError) {
	if limit <= 0 || limit > catalogNameCreditsCap {
		limit = catalogNameCreditsCap
	}
	q := url.Values{
		"include": {"credits"},
		"limit":   {strconv.Itoa(limit)},
		"offset":  {strconv.Itoa(max(offset, 0))},
	}
	openPopulation(q)

	data, found, movedTo, appErr := c.catalogGetRecord(ctx, "/catalog/names/"+strconv.FormatInt(id, 10), q)
	if appErr != nil || !found {
		return nil, false, movedTo, appErr
	}
	var n CatalogName
	if err := json.Unmarshal(data, &n); err != nil {
		return nil, false, 0, errors.ErrInternal("解析 Catalog 名义详情响应失败")
	}
	return &n, true, 0, nil
}

func (c *GalgameClient) CatalogRowsByCatalogIDs(
	ctx context.Context, ids []int64, isSFW bool,
) (map[int64]CatalogWorkListItem, *errors.AppError) {
	rows, appErr := c.worksByCatalogIDs(ctx, ids, catalogBriefInclude, contentLimitFor(isSFW))
	if appErr != nil {
		return nil, appErr
	}
	out := make(map[int64]CatalogWorkListItem, len(rows))
	for _, r := range rows {
		out[r.ID] = r
	}
	return out, nil
}

func PickCatalogName(displayName, latin string) string {
	if displayName != "" {
		return displayName
	}
	return latin
}

// CatalogEntityName renders an entity name for this site's Chinese readers:
// localized["zh-Hans"] ?? display_name ?? latin. Wave 209 made that chain
// terminal across every projection of the public face — detail records, roster
// rows, voices, credits, via_label and search hits all carry the same three
// fields — so there is exactly one render path and no projection needs its own.
//
// A reader who asked for 原名 flips the head of that chain, which is why the
// preference travels on the context rather than a parameter: one switch here
// reaches every projection, and a projection that took its own shortcut would
// silently keep rendering Chinese.
func CatalogEntityName(ctx context.Context, localized map[string]catLocalizedName, displayName, latin string) string {
	zh := pickLocalized(localized, catalogZhLocales)
	if namepref.PrefersOriginal(ctx) {
		return cmp.Or(displayName, zh, latin)
	}
	if zh != "" {
		return zh
	}
	return PickCatalogName(displayName, latin)
}

// CatalogEntityNames adds the name the reader did not choose for a secondary
// line, empty when that is already what CatalogEntityName rendered.
func CatalogEntityNames(ctx context.Context, localized map[string]catLocalizedName, displayName, latin string) (name, original string) {
	name = CatalogEntityName(ctx, localized, displayName, latin)
	other := displayName
	if namepref.PrefersOriginal(ctx) {
		other = pickLocalized(localized, catalogZhLocales)
	}
	if other != "" && other != name {
		original = other
	}
	return name, original
}

// CatalogVocabularyName renders a translated controlled vocabulary — tags,
// character traits. It deliberately ignores the 原名 preference: a tag's
// non-Chinese form is the English vocabulary token it was imported under, so
// honouring the preference here answers 金发 with Blonde, which is neither a
// name nor Japanese.
func CatalogVocabularyName(localized map[string]catLocalizedName, vocabulary string) string {
	if zh := pickLocalized(localized, catalogZhLocales); zh != "" {
		return zh
	}
	return vocabulary
}

func pickLocalized(localized map[string]catLocalizedName, locales []string) string {
	for _, locale := range locales {
		if entry, ok := localized[locale]; ok && entry.Value != "" {
			return entry.Value
		}
	}
	return ""
}
