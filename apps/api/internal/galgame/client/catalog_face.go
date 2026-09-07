package client

import (
	"cmp"
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/pkg/errors"
)

const catalogIDsChunk = 100

const catalogSpoilerCeiling = 2

var anchorSourceKeys = []string{"curated", "galgame_wiki"}

const (
	gidLookupHitTTL  = 30 * time.Minute
	gidLookupMissTTL = 2 * time.Minute
)

type gidLookupEntry struct {
	catalogID int64
	found     bool
	expire    time.Time
}

// The works list gates localized{} and latin behind include=names, the same
// switch as the four-slot names block it replaces. Dropping "names" here
// because nothing reads that block any more leaves every list row with only
// display_name, so every Chinese title on the site reverts to the original.
const (
	catalogBriefInclude       = "names,covers,refs,labels"
	catalogDetailBriefInclude = "names,intros,labels,covers,refs"
)

func openPopulation(q url.Values) url.Values {
	q.Set("nsfw", "true")
	return q
}

func OpenPopulation(q url.Values) url.Values { return openPopulation(q) }

// openPopulation is unconditional: content_rating is how old you must be to
// play the game, content_limit is the entry's own display verdict, and only the
// second one is kungal's gate. Leaving the age gate closed for an SFW reader
// left 493 of the 7,781 listed games visible, because 93% of them are r18. The
// v2 cutover dropped this call and rewrote the assertions that pinned it, so a
// green suite is not evidence here — 4b669e54 is.
func applyWorksGate(q url.Values, contentLimit string) url.Values {
	openPopulation(q)
	switch contentLimit {
	case "sfw", "nsfw":
		q.Set("content_limit", contentLimit)
	}
	return q
}

func ApplyWorksGate(q url.Values, isSFW bool) url.Values {
	return applyWorksGate(q, contentLimitFor(isSFW))
}

func contentLimitFor(isSFW bool) string {
	if isSFW {
		return "sfw"
	}
	return "all"
}

func (c *GalgameClient) catalogIDsForGIDs(ctx context.Context, gids []int) (map[int]int64, *errors.AppError) {
	out := make(map[int]int64, len(gids))
	var missing []int
	now := time.Now()

	c.gidMu.RLock()
	for _, gid := range gids {
		if e, ok := c.gidCache[gid]; ok && now.Before(e.expire) {
			if e.found {
				out[gid] = e.catalogID
			}
		} else {
			missing = append(missing, gid)
		}
	}
	c.gidMu.RUnlock()
	if len(missing) == 0 {
		return out, nil
	}

	gidStride := max(catalogIDsChunk/len(anchorSourceKeys), 1)

	resolved := make(map[int]int64, len(missing))
	for start := 0; start < len(missing); start += gidStride {
		end := min(start+gidStride, len(missing))
		chunk := missing[start:end]

		refs := make([]string, 0, len(chunk)*len(anchorSourceKeys))
		want := make(map[string]int, len(chunk)*len(anchorSourceKeys))
		for _, gid := range chunk {
			ext := strconv.Itoa(gid)
			for _, source := range anchorSourceKeys {
				token := source + ":" + ext
				refs = append(refs, token)
				want[token] = gid
			}
		}
		q := url.Values{
			"refs":    {strings.Join(refs, ",")},
			"limit":   {strconv.Itoa(catalogIDsChunk)},
			"include": {"refs"},
		}
		openPopulation(q)
		data, appErr := c.CatalogGet(ctx, "/catalog/works", q)
		if appErr != nil {
			return nil, appErr
		}
		var parsed struct {
			Items   []CatalogWorkListItem `json:"items"`
			Missing []string              `json:"missing"`
		}
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, errors.ErrInternal("解析 Catalog 批量解析响应失败")
		}
		miss := make(map[string]bool, len(parsed.Missing))
		for _, token := range parsed.Missing {
			miss[token] = true
		}
		inChunk := make(map[int]bool, len(chunk))
		for _, gid := range chunk {
			inChunk[gid] = true
		}
		for i := range parsed.Items {
			row := &parsed.Items[i]
			if g := row.gid(); g > 0 && inChunk[g] {
				resolved[g] = row.ID
			}
			for _, ref := range row.Refs {
				token := ref.Source + ":" + ref.ExternalID
				gid, ok := want[token]
				if !ok || miss[token] {
					continue
				}
				resolved[gid] = row.ID
			}
		}
	}

	var unresolved []int
	for _, gid := range missing {
		if _, ok := resolved[gid]; !ok {
			unresolved = append(unresolved, gid)
		}
	}
	if len(unresolved) > 0 {
		adopted, appErr := c.adoptedWorkIDs(ctx, unresolved)
		if appErr != nil {
			return nil, appErr
		}
		for gid, id := range adopted {
			resolved[gid] = id
			out[gid] = id
		}
	}

	c.gidMu.Lock()
	if len(c.gidCache) > batchCacheMaxEntries {
		clear(c.gidCache)
	}
	for _, gid := range missing {
		id, ok := resolved[gid]
		ttl := gidLookupMissTTL
		if ok {
			ttl = gidLookupHitTTL
			out[gid] = id
		}
		c.gidCache[gid] = gidLookupEntry{catalogID: id, found: ok, expire: now.Add(ttl)}
	}
	c.gidMu.Unlock()
	return out, nil
}

// The second half of the gid bridge. A work minted through the submission face
// carries NO external_ref anchor — there is no upstream to have issued one — so
// the anchor lookup answers "no such work" rather than an error, and every page
// of a post-switchover entry would 404 silently.
//
// THE ROUND-TRIP CHECK IS NOT DEFENSIVE, IT IS THE WHOLE CORRECTNESS ARGUMENT:
// a legacy gid is also a syntactically valid work id, so resolving gid 42 by
// fetching work 42 would hand back a different game. An adopted id satisfies
// `claim.site_work_id == the id asked for` by construction; a coincidence does not.
//
// The content gates are open because this resolves an IDENTITY. Filtering it by
// the reader's preference would make an r18 entry unresolvable rather than
// merely invisible; visibility belongs to the row fetch that follows.
func (c *GalgameClient) adoptedWorkIDs(ctx context.Context, gids []int) (map[int]int64, *errors.AppError) {
	ids := make([]int64, len(gids))
	for i, gid := range gids {
		ids[i] = int64(gid)
	}
	rows, appErr := c.worksByCatalogIDs(ctx, ids, "", "all")
	if appErr != nil {
		return nil, appErr
	}
	out := make(map[int]int64, len(rows))
	for i := range rows {
		row := &rows[i]
		if gid := row.gid(); gid > 0 && int64(gid) == row.ID {
			out[gid] = row.ID
		}
	}
	return out, nil
}

func (c *GalgameClient) worksByCatalogIDs(ctx context.Context, ids []int64, include, contentLimit string) ([]CatalogWorkListItem, *errors.AppError) {
	var out []CatalogWorkListItem
	for start := 0; start < len(ids); start += catalogIDsChunk {
		end := min(start+catalogIDsChunk, len(ids))
		chunk := ids[start:end]

		raw := make([]string, len(chunk))
		for i, id := range chunk {
			raw[i] = strconv.FormatInt(id, 10)
		}
		q := url.Values{
			"ids":   {strings.Join(raw, ",")},
			"limit": {strconv.Itoa(catalogIDsChunk)},
		}
		if include != "" {
			q.Set("include", include)
		}
		applyWorksGate(q, contentLimit)

		data, appErr := c.CatalogGet(ctx, "/catalog/works", q)
		if appErr != nil {
			return nil, appErr
		}
		var parsed catWorksListData
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, errors.ErrInternal("解析 Catalog 作品列表响应失败")
		}
		out = append(out, parsed.Items...)
	}
	return out, nil
}

func (c *GalgameClient) CatalogWorkIDs(ctx context.Context, gids []int) (map[int]int64, *errors.AppError) {
	if len(gids) == 0 {
		return map[int]int64{}, nil
	}
	return c.catalogIDsForGIDs(ctx, gids)
}

func (c *GalgameClient) GIDsByCatalogIDs(ctx context.Context, ids []int64) (map[int64]int, *errors.AppError) {
	if len(ids) == 0 {
		return map[int64]int{}, nil
	}
	rows, appErr := c.worksByCatalogIDs(ctx, ids, "", "all")
	if appErr != nil {
		return nil, appErr
	}
	out := make(map[int64]int, len(rows))
	for i := range rows {
		if gid := rows[i].gid(); gid > 0 {
			out[rows[i].ID] = gid
		}
	}
	return out, nil
}

func (c *GalgameClient) CatalogRowsByGIDs(ctx context.Context, gids []int, include, contentLimit string) (map[int]CatalogWorkListItem, *errors.AppError) {
	if len(gids) == 0 {
		return map[int]CatalogWorkListItem{}, nil
	}
	idMap, appErr := c.catalogIDsForGIDs(ctx, gids)
	if appErr != nil {
		return nil, appErr
	}
	if len(idMap) == 0 {
		return map[int]CatalogWorkListItem{}, nil
	}
	ids := make([]int64, 0, len(idMap))
	for _, id := range idMap {
		ids = append(ids, id)
	}
	rows, appErr := c.worksByCatalogIDs(ctx, ids, include, contentLimit)
	if appErr != nil {
		return nil, appErr
	}
	out := make(map[int]CatalogWorkListItem, len(rows))
	for i := range rows {
		row := rows[i]
		if !row.isRenderable() {
			continue
		}
		if gid := row.gid(); gid > 0 {
			out[gid] = row
		}
	}
	return out, nil
}

// ContentLimitsByGIDs reads catalog's editorial display verdict for local rows.
// Both gates are open on purpose: this syncs what the verdict IS, and asking
// for it through a reader's own content_limit would only ever return rows that
// already agree with it.
func (c *GalgameClient) ContentLimitsByGIDs(ctx context.Context, gids []int) (map[int]string, *errors.AppError) {
	rows, appErr := c.CatalogRowsByGIDs(ctx, gids, "", "all")
	if appErr != nil {
		return nil, appErr
	}
	out := make(map[int]string, len(rows))
	for gid := range rows {
		row := rows[gid]
		out[gid] = contentLimitOf(row.Claim, row.ContentRating)
	}
	return out, nil
}

type catWorkDetail struct {
	ID            int64                       `json:"id"`
	DisplayName   string                      `json:"display_name"`
	Localized     map[string]catLocalizedName `json:"localized"`
	Latin         string                      `json:"latin"`
	OLang         string                      `json:"olang"`
	ContentRating string                      `json:"content_rating"`
	ReleaseDate   *string                     `json:"release_date"`
	Updated       string                      `json:"updated"`
	Created       string                      `json:"created"`
	Claim         *catClaim                   `json:"claim"`

	Titles []struct {
		Lang    string `json:"lang"`
		Title   string `json:"title"`
		Kind    string `json:"kind"`
		Machine bool   `json:"machine"`
	} `json:"titles"`
	Refs []catRef     `json:"refs"`
	Tags []catWorkTag `json:"tags"`
	// Wave 212's second half renames this block to "intros", matching every
	// other catalog face. Both keys decode so the rename is not a cutover.
	Intro  []CatalogIntro `json:"intro"`
	Intros []CatalogIntro `json:"intros"`
	Covers []struct {
		ID        int64  `json:"id"`
		Hash      string `json:"hash"`
		URL       string `json:"url"`
		Kind      string `json:"kind"`
		Sexual    int    `json:"sexual"`
		Violence  int    `json:"violence"`
		Source    string `json:"source"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Thumbhash string `json:"thumbhash"`
		VoteCount int    `json:"vote_count"`
	} `json:"covers"`
	CoverSlots  *catCoverSlots `json:"cover_slots"`
	Screenshots []struct {
		URL       string `json:"url"`
		Caption   string `json:"caption"`
		Sexual    int    `json:"sexual"`
		Violence  int    `json:"violence"`
		Source    string `json:"source"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Thumbhash string `json:"thumbhash"`
	} `json:"screenshots"`
	Labels     []catWorkLabel     `json:"labels"`
	Engines    []catWorkEngine    `json:"engines"`
	Links      []catWorkLink      `json:"links"`
	Series     []catWorkSeries    `json:"series"`
	Credits    []catCreditGroup   `json:"credits"`
	Characters []catWorkCharacter `json:"characters"`
	Ratings    []catRating        `json:"ratings"`
	Playtimes  []catPlaytime      `json:"playtimes"`
	Popularity []catPopularity    `json:"popularity"`
}

// The favourite count on a game page comes from here, not from
// galgame_local.favorite_count: catalog computes nextmoe/favorites as the
// number of distinct people holding the work in any folder on any site, so a
// favourite added from the patch site shows up in this number and never in the
// local counter. include=popularity was already being requested and thrown
// away.
type catPopularity struct {
	Source string `json:"source"`
	Metric string `json:"metric"`
	Value  int64  `json:"value"`
}

func catalogFavoriteCount(rows []catPopularity) int {
	for _, r := range rows {
		if r.Source == "nextmoe" && r.Metric == "favorites" {
			return int(r.Value)
		}
	}
	return 0
}

type catWorkCharacter struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Latin       string                      `json:"latin"`
	Kind        string                      `json:"kind"`
	Spoiler     int                         `json:"spoiler"`
	Image       string                      `json:"image"`
	Figure      string                      `json:"figure"`
	Identity    string                      `json:"identity"`
	Voices      []CatalogPerson             `json:"voices"`
	ImageMeta   ArtMeta                     `json:"-"`
	FigureMeta  ArtMeta                     `json:"-"`
}

type catCreditGroup struct {
	RoleKey  string          `json:"role_key"`
	RoleName string          `json:"role_name"`
	Credits  []catCreditItem `json:"credits"`
}

type catCreditItem struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Latin       string                      `json:"latin"`
	CharacterID int64                       `json:"character_id"`
	Character   string                      `json:"character"`
}

func (c *catCreditItem) Name(ctx context.Context) string {
	return CatalogEntityName(ctx, c.Localized, c.DisplayName, c.Latin)
}

type catWorkTag struct {
	Name        string                      `json:"name"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Source      string                      `json:"source"`
	CanonicalID int64                       `json:"canonical_id"`
	Tier        string                      `json:"tier"`
	Kind        string                      `json:"kind"`
	Spoiler     int                         `json:"spoiler"`
	Sexual      bool                        `json:"sexual"`
	WorkCount   int                         `json:"work_count"`
}

func (t *catWorkTag) Label() string {
	return CatalogVocabularyName(t.Localized, cmp.Or(t.DisplayName, t.Name))
}

type catWorkSeries struct {
	ID          int64                       `json:"id"`
	Name        string                      `json:"name"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
}

func (s *catWorkSeries) Label(ctx context.Context) string {
	return CatalogEntityName(ctx, s.Localized, cmp.Or(s.DisplayName, s.Name), "")
}

func (c *GalgameClient) CatalogWorkDetail(ctx context.Context, gid int) (*catWorkDetail, bool, *errors.AppError) {
	idMap, appErr := c.catalogIDsForGIDs(ctx, []int{gid})
	if appErr != nil {
		return nil, false, appErr
	}
	catalogID, ok := idMap[gid]
	if !ok {
		return nil, false, nil
	}
	// The tag panel's 剧透等级 filter defaults to level 0 and reveals the rest on
	// demand, so it needs the rows to filter: asking for spoilers=0 here made
	// levels 1 and 2 match nothing, forever. SEO text must still cut back to
	// level 0 — see pages/galgame/[gid]/index.vue.
	q := url.Values{
		"spoilers": {strconv.Itoa(catalogSpoilerCeiling)},
		"include":  {"credits"},
	}
	openPopulation(q)
	data, appErr := c.CatalogGet(ctx, "/catalog/works/"+strconv.FormatInt(catalogID, 10), q)
	if appErr != nil {
		if appErr.StatusCode == 404 {
			return nil, false, nil
		}
		return nil, false, appErr
	}
	var d catWorkDetail
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, false, errors.ErrInternal("解析 Catalog 作品详情响应失败")
	}
	if d.Claim != nil && d.Claim.State == claimStateHidden {
		return nil, false, nil
	}
	c.hydrateRosterArt(d.Characters)
	return &d, true, nil
}

func (c *GalgameClient) hydrateRosterArt(chars []catWorkCharacter) {
	urls := make([]string, 0, len(chars)*2)
	for _, ch := range chars {
		urls = append(urls, ch.Image, ch.Figure)
	}
	meta := c.resolveArtMeta(urls)
	if meta == nil {
		return
	}
	for i := range chars {
		chars[i].ImageMeta = meta[chars[i].Image]
		chars[i].FigureMeta = meta[chars[i].Figure]
	}
}

type CatalogWorksPage struct {
	Items      []CatalogWorkListItem
	NextCursor string
	Total      int64
	Count      int64
	Month      string
	Year       string
	Meta       catCalendarMeta
}

func (c *GalgameClient) CatalogWorksList(ctx context.Context, q url.Values) (*CatalogWorksPage, *errors.AppError) {
	data, appErr := c.CatalogGet(ctx, "/catalog/works", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed catWorksListData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 作品列表响应失败")
	}
	page := &CatalogWorksPage{Items: parsed.Items}
	if parsed.NextCursor != nil {
		page.NextCursor = *parsed.NextCursor
	}
	return page, nil
}

func (c *GalgameClient) CatalogMemberGIDs(ctx context.Context, filter url.Values, isSFW bool, pageCap int) ([]int, *errors.AppError) {
	members, appErr := c.catalogMembers(ctx, filter, isSFW, pageCap)
	if appErr != nil {
		return nil, appErr
	}
	gids := make([]int, 0, len(members))
	for _, m := range members {
		gids = append(gids, m.GID)
	}
	return gids, nil
}

type CatalogRollupMember struct {
	GID int
	Via *CatalogLabelVia
}

func (c *GalgameClient) CatalogLabelRollupMembers(ctx context.Context, labelID, sort string, isSFW bool, pageCap int) ([]CatalogRollupMember, *errors.AppError) {
	q := url.Values{"label_id": {labelID}, "label_rollup": {"1"}}
	if sort != "" {
		q.Set("sort", sort)
	}
	return c.catalogMembers(ctx, q, isSFW, pageCap)
}

func (c *GalgameClient) catalogMembers(ctx context.Context, filter url.Values, isSFW bool, pageCap int) ([]CatalogRollupMember, *errors.AppError) {
	members := []CatalogRollupMember{}
	cursor := ""
	for page := 0; page < pageCap; page++ {
		q := url.Values{}
		for k, v := range filter {
			q[k] = v
		}
		q.Set("limit", strconv.Itoa(catalogIDsChunk))
		ApplyWorksGate(q, isSFW)
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := c.CatalogWorksList(ctx, q)
		if appErr != nil {
			return nil, appErr
		}
		for i := range res.Items {
			if !res.Items[i].isRenderable() {
				continue
			}
			if gid := res.Items[i].gid(); gid > 0 {
				members = append(members, CatalogRollupMember{
					GID: gid,
					Via: res.Items[i].ViaLabel,
				})
			}
		}
		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}
	return members, nil
}

func (c *GalgameClient) CatalogWorksSearch(ctx context.Context, q url.Values) (*CatalogWorksPage, *errors.AppError) {
	data, appErr := c.CatalogGet(ctx, "/catalog/works/search", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed catWorksSearchData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 搜索响应失败")
	}
	return &CatalogWorksPage{Items: parsed.Items, Total: parsed.Total}, nil
}

func (c *GalgameClient) CatalogCalendar(ctx context.Context, bucket string, q url.Values) (*CatalogWorksPage, *errors.AppError) {
	data, appErr := c.CatalogGet(ctx, "/catalog/calendar"+bucket, q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed catWorksListData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 月历响应失败")
	}
	count := parsed.Count
	if count == 0 {
		count = parsed.Total
	}
	page := &CatalogWorksPage{
		Items: parsed.Items, Count: count,
		Month: parsed.Month, Year: parsed.Year, Meta: parsed.Meta,
	}
	if parsed.NextCursor != nil {
		page.NextCursor = *parsed.NextCursor
	}
	return page, nil
}

// GIDsToCatalogIDs is the gid bridge exported for the merge sync, which reads it
// as a negative: a gid that resolves to nothing is a gid whose catalog work is
// gone, and only such a gid may be folded into a survivor.
func (c *GalgameClient) GIDsToCatalogIDs(ctx context.Context, gids []int) (map[int]int64, *errors.AppError) {
	return c.catalogIDsForGIDs(ctx, gids)
}

// ForgetGIDs drops cached resolutions so the next lookup asks catalog.
//
// The merge sync must call this before it decides a gid is dead. gidLookupHitTTL
// is 30 minutes, and the redirect cursor advances whether or not a fold happened
// — one stale hit is one page that stays 404 forever.
func (c *GalgameClient) ForgetGIDs(gids []int) {
	if len(gids) == 0 {
		return
	}
	c.gidMu.Lock()
	for _, gid := range gids {
		delete(c.gidCache, gid)
	}
	c.gidMu.Unlock()
}

// GIDsForCatalogIDs reads the bridge the other way: catalog id to the forum gid
// that names the work. It is the claim's site_work_id, or for an unclaimed work
// the catalog id itself — which is a CANDIDATE, not an answer, because 10,289 of
// the forum's gids are also the catalog id of a different work. The caller has
// to round-trip it through GIDsToCatalogIDs before acting on it.
func (c *GalgameClient) GIDsForCatalogIDs(ctx context.Context, ids []int64) (map[int64]int, *errors.AppError) {
	if len(ids) == 0 {
		return map[int64]int{}, nil
	}
	rows, appErr := c.worksByCatalogIDs(ctx, ids, "", "all")
	if appErr != nil {
		return nil, appErr
	}
	out := make(map[int64]int, len(rows))
	for i := range rows {
		if gid := rows[i].gid(); gid > 0 {
			out[rows[i].ID] = gid
		}
	}
	return out, nil
}
