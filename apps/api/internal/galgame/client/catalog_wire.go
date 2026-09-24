package client

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
)

type catClaim struct {
	Site         string `json:"site"`
	SiteWorkID   int    `json:"site_work_id"`
	State        string `json:"state"`
	ContentLimit string `json:"content_limit"`
}

type catRef struct {
	Source     string `json:"source"`
	ExternalID string `json:"external_id"`
}

type catRelatedLink struct {
	Source string `json:"source"`
	URL    string `json:"url"`
}

type CoverSlot struct {
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Thumbhash string `json:"thumbhash"`
	Sexual    *int   `json:"sexual"`
	Violence  int    `json:"violence"`
	Source    string `json:"source"`
}

type catCoverSlots struct {
	Portrait *CoverSlot `json:"portrait"`
	Banner   *CoverSlot `json:"banner"`
}

// catIntros is the works-list brief's intro block, held in the [{lang, intro,
// source, machine}] shape every other catalog face already sends. Wave 212's
// second half turns the wire shape from an object keyed by the four product
// locales into exactly that array, so both decode here and the object branch
// dies with the slots. Decoding only one of the two would take down every
// galgame detail surface the day catalog switches, the way wave 210's names
// block did.
type catIntros []CatalogIntro

// productLocaleSlots is the key order of the retiring object shape. The array
// shape carries its own BCP-47 lang and never reaches this list.
var productLocaleSlots = []string{"ja-jp", "zh-cn", "zh-tw", "en-us"}

func (in *catIntros) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*in = nil
		return nil
	}
	if b[0] == '[' {
		var rows []CatalogIntro
		if err := json.Unmarshal(b, &rows); err != nil {
			return err
		}
		*in = rows
		return nil
	}

	var slots map[string]CatalogIntro
	if err := json.Unmarshal(b, &slots); err != nil {
		return err
	}
	rows := make([]CatalogIntro, 0, len(slots))
	for _, key := range productLocaleSlots {
		if slot, ok := slots[key]; ok && slot.Intro != "" {
			slot.Lang = key
			rows = append(rows, slot)
		}
	}
	*in = rows
	return nil
}

type catWorkLabel struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	LabelKind   string                      `json:"label_kind"`
	Kind        string                      `json:"kind"`
	Role        string                      `json:"role"`
	Lang        string                      `json:"lang"`
	WorkCount   int                         `json:"work_count"`
}

func (l *catWorkLabel) Name(ctx context.Context) string {
	return CatalogEntityName(ctx, l.Localized, l.DisplayName, "")
}

// What the company DID on this work, never what it IS: v2 sends both and the
// two look alike ("publisher" is a value of each).
func (l *catWorkLabel) AttributionRole() string {
	if l.Role != "" {
		return l.Role
	}
	return l.Kind
}

// A doujin work's maker is its circle and a commercial one's is its developer,
// so the card cannot just take the first company: works carry the publisher
// first as often as not.
var companyRoleRank = map[string]int{
	"developer": 0,
	"circle":    1,
	"brand":     2,
	"publisher": 3,
}

func makerName(ctx context.Context, labels []catWorkLabel) string {
	if l := makerLabel(labels); l != nil {
		return l.Name(ctx)
	}
	return ""
}

// MakerLabel is the credited company a card names as the work's maker, nil when
// no credited company has a name.
func (it *CatalogWorkListItem) MakerLabel() *catWorkLabel {
	return makerLabel(it.Labels)
}

func makerLabel(labels []catWorkLabel) *catWorkLabel {
	var best *catWorkLabel
	bestRank := len(companyRoleRank) + 1
	for i := range labels {
		rank, ok := companyRoleRank[labels[i].AttributionRole()]
		if !ok {
			rank = len(companyRoleRank)
		}
		if best != nil && rank >= bestRank {
			continue
		}
		if labels[i].hasName() {
			best, bestRank = &labels[i], rank
		}
	}
	return best
}

func (l *catWorkLabel) hasName() bool {
	if l.DisplayName != "" {
		return true
	}
	for _, n := range l.Localized {
		if n.Value != "" {
			return true
		}
	}
	return false
}

type catWorkEngine struct {
	ID          int64                       `json:"id"`
	Name        string                      `json:"name"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	WorkCount   int                         `json:"work_count"`
}

type catWorkLink struct {
	Source string `json:"source"`
	URL    string `json:"url"`
}

type catPlaytime struct {
	Source    string `json:"source"`
	Minutes   int    `json:"minutes"`
	VoteCount int    `json:"vote_count"`
}

type catRatingBucket struct {
	Score int `json:"score"`
	Count int `json:"count"`
}

type catRatingStats struct {
	Average *float64 `json:"average"`
	Stdev   *float64 `json:"stdev"`
	Min     *float64 `json:"min"`
	Max     *float64 `json:"max"`
}

// Distribution and Stats are detail-face only, and what each source carries
// changes over time — erogamescape and vndb both gained a histogram in 2026-08,
// so nothing downstream may hardcode which source has what. Distribution keys
// are the source's own buckets, not a shared axis: erogamescape's are deciles
// (0, 10, … 100), everyone else's are points. The works-list ratings block
// never carries either field.
type catRating struct {
	Source       string            `json:"source"`
	Score        float64           `json:"score"`
	VoteCount    int               `json:"vote_count"`
	Rank         *int              `json:"rank"`
	Distribution []catRatingBucket `json:"distribution"`
	Stats        *catRatingStats   `json:"stats"`
}

type CatalogWorkListItem struct {
	ID            int64     `json:"id"`
	Medium        string    `json:"medium"`
	DisplayName   string    `json:"display_name"`
	ContentRating string    `json:"content_rating"`
	ContentLimit  string    `json:"content_limit"`
	OLang         string    `json:"olang"`
	ReleaseDate   *string   `json:"release_date"`
	Claim         *catClaim `json:"claim"`
	Cover         string    `json:"cover"`
	Updated       string    `json:"updated"`

	Localized  map[string]catLocalizedName `json:"localized"`
	Latin      string                      `json:"latin"`
	Intros     catIntros                   `json:"intros"`
	Labels     []catWorkLabel              `json:"labels"`
	Ratings    []catRating                 `json:"ratings"`
	Covers     *catCoverSlots              `json:"covers"`
	CoverSlots *catCoverSlots              `json:"cover_slots"`
	Refs       []catRef                    `json:"refs"`

	ViaLabel *CatalogLabelVia `json:"via_label"`
}

type CatalogLabelVia struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
}

func (v *CatalogLabelVia) Name(ctx context.Context) string {
	return CatalogEntityName(ctx, v.Localized, v.DisplayName, "")
}

// catWorkBrief is the shared work projection embedded in the character, name,
// tag and series faces. Only the id is read: every work row the forum renders
// is re-hydrated from the works list face, which is where the claim state and
// the local row live.
type catWorkBrief struct {
	ID int64 `json:"id"`
}

type catWorksListData struct {
	Items      []CatalogWorkListItem `json:"items"`
	NextCursor *string               `json:"next_cursor"`
	Count      int64                 `json:"count"`
	Total      int64                 `json:"total"`
	Month      string                `json:"month"`
	Year       string                `json:"year"`
	Meta       catCalendarMeta       `json:"meta"`
}

type catCalendarMeta struct {
	Today    string `json:"today"`
	MinMonth string `json:"min_month"`
	MaxMonth string `json:"max_month"`
	HasPrev  *bool  `json:"has_prev"`
	HasNext  *bool  `json:"has_next"`
}

type catWorksSearchData struct {
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Limit int                   `json:"limit"`
	Items []CatalogWorkListItem `json:"items"`
}

func hashFromURL(u string) string {
	if u == "" {
		return ""
	}
	base := u
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimSuffix(base, ".webp")
}

// Catalog's shelf verdict, read as sent. The forum used to derive it from the
// claim, falling back to rating == r18 for an unclaimed work; catalog's shelf
// also counts a work whose cover art is all explicit, so 18 unclaimed works
// showed explicit covers to SFW readers (2026-09-24). An unusable verdict fails
// closed.
func contentLimitOf(workID int64, limit string) string {
	switch limit {
	case "sfw", "nsfw":
		return limit
	}
	slog.Warn("catalog work carries no usable content_limit, shown as nsfw", "work_id", workID, "content_limit", limit)
	return "nsfw"
}

func ageLimitFromRating(rating string) string {
	if rating == "r18" {
		return "r18"
	}
	return "all"
}

func productLocale(olang string) string {
	tag := strings.ToLower(strings.TrimSpace(olang))
	switch {
	case tag == "":
		return ""
	case tag == "ja" || strings.HasPrefix(tag, "ja-"):
		return "ja-jp"
	case tag == "zh-hant" || strings.HasPrefix(tag, "zh-hant-") ||
		tag == "zh-tw" || tag == "zh-hk":
		return "zh-tw"
	case tag == "zh" || strings.HasPrefix(tag, "zh"):
		return "zh-cn"
	case tag == "en" || strings.HasPrefix(tag, "en-"):
		return "en-us"
	default:
		return olang
	}
}

const (
	claimStateLive    = "live"
	claimStateDraft   = "draft"
	claimStatePending = "pending"
	claimStateHidden  = "hidden"
	claimStateNone    = "none"
)

func statusFromClaimState(state string) int {
	if state == claimStateLive {
		return GalgameStatusPublished
	}
	return galgameStatusVndbDraft
}

const (
	GalgameStatusPublished = 0
	galgameStatusVndbDraft = 2
)

func (it *CatalogWorkListItem) isRenderable() bool {
	return it.Claim == nil || it.Claim.State != claimStateHidden
}

func CatalogItemRenderable(it *CatalogWorkListItem) bool { return it.isRenderable() }

// CatalogItemWizardEligible reports whether a search row belongs in the publish
// wizard's supply: an unclaimed registry row, which anyone may adopt, or a
// kungal claim in state live, draft or pending. This is the wizard's ONLY
// claim-state gate, because the search face answers from two clocks — the
// claim_state facet is the index's, while claimed_by is re-hydrated from the
// registry — and they disagree in BOTH directions until the daily
// reindex-catalog run: a just-declined work passes the facet while already
// reading "declined", and a just-approved one is dropped by a facet it no
// longer matches.
func CatalogItemWizardEligible(it *CatalogWorkListItem) bool {
	if it.Claim == nil {
		return true
	}
	if !isKungalClaim(it.Claim.Site) || it.Claim.SiteWorkID <= 0 {
		return false
	}
	switch it.Claim.State {
	case claimStateLive, claimStateDraft, claimStatePending:
		return true
	default:
		return false
	}
}

func (it *CatalogWorkListItem) ClaimState() string {
	if it.Claim == nil {
		return ""
	}
	return it.Claim.State
}

const ClaimSiteKungal = "kungal"

const claimSiteLegacy = "galgame_wiki"

func isKungalClaim(site string) bool {
	return site == ClaimSiteKungal || site == claimSiteLegacy
}

func refsMap(refs []catRef) map[string]string {
	if refs == nil {
		return nil
	}
	out := make(map[string]string, len(refs))
	for _, r := range refs {
		prev, seen := out[r.Source]
		switch {
		case !seen:
			out[r.Source] = r.ExternalID
		case r.Source == sourceVNDB && !isVndbWorkID(prev) && isVndbWorkID(r.ExternalID):
			out[r.Source] = r.ExternalID
		}
	}
	return out
}

const sourceVNDB = "vndb"

func isVndbWorkID(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func portraitFields(covers *catCoverSlots) (hash, url string, w, h int, thumb string) {
	if covers == nil || covers.Portrait == nil {
		return "", "", 0, 0, ""
	}
	s := covers.Portrait
	return hashFromURL(s.URL), s.URL, s.Width, s.Height, s.Thumbhash
}

func coverFields(covers *catCoverSlots, fallbackURL string) (hash, url string, w, h int, thumb string) {
	slot := (*CoverSlot)(nil)
	if covers != nil {
		if covers.Banner != nil {
			slot = covers.Banner
		} else if covers.Portrait != nil {
			slot = covers.Portrait
		}
	}
	if slot == nil {
		return hashFromURL(fallbackURL), fallbackURL, 0, 0, ""
	}
	return hashFromURL(slot.URL), slot.URL, slot.Width, slot.Height, slot.Thumbhash
}

func CatalogItemToBrief(ctx context.Context, it *CatalogWorkListItem) GalgameBrief {
	name, original := it.Names(ctx)
	b := GalgameBrief{
		ID:               int(it.ID),
		Name:             name,
		NameOriginal:     original,
		AgeLimit:         ageLimitFromRating(it.ContentRating),
		ContentLimit:     contentLimitOf(it.ID, it.ContentLimit),
		OriginalLanguage: productLocale(it.OLang),
		ReleaseDate:      it.ReleaseDate,
		Refs:             refsMap(it.Refs),
		Company:          makerName(ctx, it.Labels),
	}
	if it.Claim != nil {
		b.Status = statusFromClaimState(it.Claim.State)
		b.ClaimState = it.Claim.State
	} else {
		b.Status = galgameStatusVndbDraft
		b.ClaimState = claimStateNone
	}
	b.VndbID = b.Refs["vndb"]
	slots := it.CoverSlots
	if slots == nil {
		slots = it.Covers
	}
	b.EffectiveBannerHash, b.EffectiveBannerURL,
		b.EffectiveBannerWidth, b.EffectiveBannerHeight,
		b.EffectiveBannerThumbhash = coverFields(slots, it.Cover)
	b.EffectivePortraitHash, b.EffectivePortraitURL,
		b.EffectivePortraitWidth, b.EffectivePortraitHeight,
		b.EffectivePortraitThumbhash = portraitFields(slots)
	return b
}

// Names renders the title this site shows and, when it differs, the work's own
// title underneath. Wave 212 put the same primitive on works that every other
// catalog entity has had since wave 209, retiring the four product-locale slots
// that structurally could not hold a Korean, Russian or untagged title.
func (it *CatalogWorkListItem) Names(ctx context.Context) (name, original string) {
	return CatalogEntityNames(ctx, it.Localized, it.DisplayName, it.Latin)
}

func CatalogItemToDetailBrief(ctx context.Context, it *CatalogWorkListItem) GalgameDetailBrief {
	b := GalgameDetailBrief{GalgameBrief: CatalogItemToBrief(ctx, it)}
	b.Intros = OrderIntros(it.Intros)
	for _, l := range it.Labels {
		b.Officials = append(b.Officials, l.Name(ctx))
	}
	return b
}
