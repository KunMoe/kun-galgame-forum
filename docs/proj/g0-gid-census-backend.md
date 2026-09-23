# G0 backend census: semantic uses of the galgame gid

Forum worktree `kun-galgame-forum-g-galgame`, branch `api-v1/g-galgame`. Read-only census of backend (and the web files the mapping already reaches) uses of the forum galgame id (`galgame.id`, historically **gid**) for the 2026-09-23 decision that after one maintenance-window renumber **forum galgame id ≡ `catalog_work.id` always**, named **`work_id`**, with **no mapping layer**.

G0 action values: **delete** (mapping layer goes away) · **pass-through** (the value becomes the work id with no translation; only a rename) · **data-rewrite** (stored value must be rewritten by the renumber) · **external** (the number is sent to / stored in another service) · **decide** (orchestrator must rule).

Each row is one `file:line`. Quoted text is the source line.

---

## 1. The mapping layer

Every function, type, cache, and test that translates gid ↔ catalog id, or checks whether one is the other, plus every caller followed from the seed symbols.

### 1.1 Core bridge (gid → catalog id)

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/client/catalog_face.go:19` | `var anchorSourceKeys = []string{"curated", "galgame_wiki"}` | External-ref sources whose `external_id` is treated as a forum gid | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:21` | `func isAnchorSource(source string) bool {` | True when a catalog ref is a gid-bearing `curated` / `galgame_wiki` anchor | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:35` | `type gidLookupEntry struct {` | Cache row: forum gid → catalog work id + found + expiry | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:83` | `func (c *GalgameClient) catalogIDsForGIDs(ctx context.Context, gids []int) (map[int]int64, *errors.AppError) {` | Batch gid → catalog work id via `refs=curated:gid,galgame_wiki:gid` then `adoptedWorkIDs` | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:115` | `token := source + ":" + ext` | `curated:<gid>` / `galgame_wiki:<gid>` token sent as catalog `refs=` | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:121` | `"refs":    {strings.Join(refs, ",")},` | Catalog works-list query that resolves gids through external refs | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:168` | `adopted, appErr := c.adoptedWorkIDs(ctx, unresolved)` | Fallback for gids with no curated/wiki ref (lazy-minted pages) | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:189` | `c.gidCache[gid] = gidLookupEntry{catalogID: id, found: ok, expire: now.Add(ttl)}` | In-process cache write of a gid resolution (hit TTL 30m, miss 2m) | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:195` | `func (c *GalgameClient) CatalogWorkIDForGID(ctx context.Context, gid int) (int64, bool, *errors.AppError) {` | Single-gid wrapper over `catalogIDsForGIDs` | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:217` | `func (c *GalgameClient) adoptedWorkIDs(ctx context.Context, gids []int) (map[int]int64, *errors.AppError) {` | Fetches catalog works **by the gid as a catalog id** and accepts the row only when `gid() == ID` | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:229` | `if gid := row.gid(); gid > 0 && int64(gid) == row.ID {` | Round-trip check: an adopted work’s claim `site_work_id` must equal its catalog id | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:268` | `func (c *GalgameClient) CatalogWorkIDs(ctx context.Context, gids []int) (map[int]int64, *errors.AppError) {` | Exported gid → catalog id map | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:292` | `func (c *GalgameClient) CatalogRowsByGIDs(ctx context.Context, gids []int, include, contentLimit string) (map[int]CatalogWorkListItem, *errors.AppError) {` | Resolve gids to catalog ids, fetch works, re-key the map by gid | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:347` | `func (c *GalgameClient) MirrorByGIDs(ctx context.Context, gids []int) (map[int]CatalogMirror, *errors.AppError) {` | Mirror fill lane: local row ids (gids) → catalog content_limit / release_date | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:502` | `func (c *GalgameClient) CatalogWorkDetail(ctx context.Context, gid int) (*catWorkDetail, bool, *errors.AppError) {` | Detail page: gid → catalog id, then `GET /catalog/works/{catalogID}` | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:679` | `func (c *GalgameClient) GIDsToCatalogIDs(ctx context.Context, gids []int) (map[int]int64, *errors.AppError) {` | Merge-sync export of `catalogIDsForGIDs` (alive vs dead gid) | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:688` | `func (c *GalgameClient) ForgetGIDs(gids []int) {` | Drop cached gid resolutions before merge-sync decides a gid is dead | **delete** |
| `apps/api/internal/galgame/client/client.go:104` | `gidCache map[int]gidLookupEntry` | Process-lifetime gid → catalog id cache on `GalgameClient` | **delete** |
| `apps/api/internal/galgame/client/client.go:135` | `gidCache:     map[int]gidLookupEntry{},` | Cache construction | **delete** |
| `apps/api/internal/galgame/client/client.go:236` | `func (c *GalgameClient) GetBatchDetailPublic(ctx context.Context, ids []int, isSFW bool) (map[int]GalgameDetailBrief, *errors.AppError) {` | Batch keyed by forum gid via `CatalogRowsByGIDs` | **delete** |
| `apps/api/internal/galgame/client/client.go:251` | `func (c *GalgameClient) GetBatch(ctx context.Context, ids []int) (map[int]GalgameBrief, *errors.AppError) {` | Batch keyed by forum gid | **delete** |
| `apps/api/internal/galgame/client/client.go:255` | `func (c *GalgameClient) GetBatchPublic(ctx context.Context, ids []int, isSFW bool) (map[int]GalgameBrief, *errors.AppError) {` | SFW-gated batch keyed by forum gid | **delete** |
| `apps/api/internal/galgame/client/client.go:261` | `func (c *GalgameClient) batchByGIDs(ctx context.Context, ids []int, contentLimit string) (map[int]GalgameBrief, *errors.AppError) {` | Implements GetBatch* through `CatalogRowsByGIDs` | **delete** |

### 1.2 Reverse bridge (catalog id → gid)

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/client/catalog_face.go:275` | `func (c *GalgameClient) GIDsByCatalogIDs(ctx context.Context, ids []int64) (map[int64]int, *errors.AppError) {` | Catalog work id → `row.gid()` (claim `site_work_id` or catalog id if unclaimed) | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:704` | `func (c *GalgameClient) GIDsForCatalogIDs(ctx context.Context, ids []int64) (map[int64]int, *errors.AppError) {` | Same reverse map; merge-sync requires a round-trip through `GIDsToCatalogIDs` | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:724` | `func (c *GalgameClient) StaleGIDsForCatalogIDs(ctx context.Context, ids []int64) (map[int64][]int, *errors.AppError) {` | Per catalog id, curated/wiki `external_id`s that are **not** the current gid (pages to fold) | **delete** |
| `apps/api/internal/galgame/client/catalog_changes.go:81` | `func (c *GalgameClient) MirrorByCatalogIDs(ctx context.Context, ids []int64) (map[int]CatalogMirror, *errors.AppError) {` | Changes feed names catalog ids; map is re-keyed by kungal `claim.SiteWorkID` (forum gid) | **delete** |
| `apps/api/internal/galgame/client/catalog_changes.go:98` | `out[row.Claim.SiteWorkID] = mirrorOf(row)` | Local galgame row id written from the claim’s site work id, never from catalog `id` | **delete** |
| `apps/api/internal/galgame/client/catalog_name.go:251` | `func (c *GalgameClient) CatalogRowsByCatalogIDs(` | Fetch works by catalog id, keyed by catalog id (no gid re-key); character/staff then call `gid()` via `CatalogItemToNextMoeItem` | **delete** (the gid extraction on the item) |

### 1.3 Wire decode: `gid()`, `CatalogItemGID`, `SiteWorkID`

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/client/catalog_wire.go:13` | `// SiteWorkID is the CLAIMING site's own id for the work — the forum's gid when` | Documents that `claim.site_work_id` is the forum gid when site is kungal | **delete** |
| `apps/api/internal/galgame/client/catalog_wire.go:20` | `SiteWorkID   int    `json:"site_work_id"`` | Claim field: forum gid (kungal) or moyu page id (moyu) | **pass-through** after infra rewrites kungal claims so this equals catalog id; then **delete** the dual-space comment |
| `apps/api/internal/galgame/client/catalog_wire.go:340` | `func CatalogItemGID(it *CatalogWorkListItem) int { return it.gid() }` | Exported alias of `gid()` | **delete** |
| `apps/api/internal/galgame/client/catalog_wire.go:366` | `func (it *CatalogWorkListItem) GID() int { return it.gid() }` | Method alias of `gid()` | **delete** |
| `apps/api/internal/galgame/client/catalog_wire.go:387` | `func (it *CatalogWorkListItem) gid() int {` | Forum page id of a catalog list row: kungal `SiteWorkID`, else catalog `ID` | **delete** |
| `apps/api/internal/galgame/client/catalog_wire.go:389` | `return it.Claim.SiteWorkID` | Claimed kungal work: the forum gid | **delete** |
| `apps/api/internal/galgame/client/catalog_wire.go:392` | `return int(it.ID)` | Unclaimed work: catalog id used as the forum page id | **pass-through** (this is already the post-G0 identity) |
| `apps/api/internal/galgame/client/catalog_wire.go:399` | `const claimSiteLegacy = "galgame_wiki"` | Retired claim site spelling still treated as kungal | **decide** (claim-site alias vs gid mapping; not itself a gid) |
| `apps/api/internal/galgame/client/catalog_wire.go:401` | `func isKungalClaim(site string) bool {` | `kungal` or `galgame_wiki` may name a forum gid on the claim | **pass-through** (site identity stays; it stops being a gid discriminator) |
| `apps/api/internal/galgame/client/catalog_wire.go:462` | `ID:               it.gid(),` | `GalgameBrief.ID` is the forum gid; `WorkID` is catalog `it.ID` | **pass-through** (`ID` becomes the work id; drop the split) |
| `apps/api/internal/galgame/client/catalog_wire.go:463` | `WorkID:           it.ID,` | Catalog registry id carried beside the forum gid on every brief | **delete** (the second field; one id remains) |
| `apps/api/internal/galgame/client/catalog_wire.go:514` | `ID:               it.gid(),` | Calendar/search/tag card id is the forum gid | **pass-through** |
| `apps/api/internal/galgame/client/catalog_detail.go:10` | `func CatalogDetailToFull(ctx context.Context, d *catWorkDetail, gid int) dto.NextMoeGalgameDetailFull {` | Detail DTO `ID` is the **caller’s gid**, not `d.ID` | **pass-through** (pass the work id; stop taking a separate gid) |
| `apps/api/internal/galgame/client/catalog_detail.go:13` | `ID:               gid,` | Page id on the detail payload | **pass-through** |
| `apps/api/internal/galgame/client/catalog_detail.go:290` | `func (c *GalgameClient) CatalogWorkLinks(ctx context.Context, gid int) ([]GalgameLink, *errors.AppError) {` | Link proxy: gid through `CatalogWorkDetail` | **delete** (the translation); remaining fetch is **pass-through** |
| `apps/api/internal/galgame/client/catalog_face.go:579` | `func (c *GalgameClient) CatalogMemberGIDs(ctx context.Context, filter url.Values, isSFW bool, pageCap int) ([]int, *errors.AppError) {` | Walks a taxonomy membership and returns `row.gid()` list | **delete** (`gid()`); remaining walk is **pass-through** of catalog ids |
| `apps/api/internal/galgame/client/catalog_face.go:625` | `if gid := res.Items[i].gid(); gid > 0 {` | Member id for engine/tag/series/official rollup | **pass-through** |
| `apps/api/internal/galgame/client/catalog_v2.go:464` | `if n, ok := digitString(claim["site_work_id"]); ok {` | v2 JSON rewrite: keep `site_work_id` as a number (forum gid on kungal claims) | **pass-through** (field stays on catalog’s claim object) |
| `apps/api/internal/galgame/client/client.go:195` | `ID                  int     `json:"id"`` | Brief `id` is forum gid | **pass-through** (rename to work_id in the public contract elsewhere) |
| `apps/api/internal/galgame/client/client.go:196` | `WorkID              int64   `json:"work_id"`` | Brief `work_id` is catalog id, a second space | **delete** |

### 1.4 `workIDOf` / `gidOf` (per-package translators)

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/handler/edit_handler.go:112` | `// THE ID-SPACE TRAP, and the reason for every workIDOf / gidOf / entryOf call` | Documents overlapping gid and catalog-id spaces | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:130` | `gid := h.gidOf(ctx, workID)` | Catalog work id → forum gid to look up local owner | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:138` | `func (h *EditHandler) workIDOf(ctx context.Context, gid int64) (int64, *errors.AppError) {` | Path `:gid` → catalog entity id for edit proposals | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:176` | `func (h *EditHandler) gidOf(ctx context.Context, workID int64) int {` | Catalog entity id → forum gid via `GIDsByCatalogIDs` | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:335` | `workID, appErr := h.workIDOf(ctx, gid)` | Bootstrap snapshot/schema: forum gid → catalog work id | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:402` | `workID, appErr := h.workIDOf(c.Context(), gid)` | Create edit proposal `EntityID` | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:453` | `workID, appErr := h.workIDOf(c.Context(), gid)` | List revisions for a game | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:522` | `workID, appErr := h.workIDOf(ctx, gid)` | Revert to a seq | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:549` | `workID, appErr := h.workIDOf(c.Context(), gid)` | Diff face: forum gid → catalog work id | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:578` | `if gidByWork, appErr = h.galgameClient.GIDsByCatalogIDs(ctx, workIDs); appErr != nil {` | Enrich proposals with forum gid | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:588` | `rows, appErr := h.galgameClient.CatalogRowsByGIDs(ctx, gids, "names,covers", "all")` | Brief cards keyed by the translated gid | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:601` | `item := proposalItem{EditProposal: items[i], GID: gidByWork[items[i].EntityID]}` | Proposal JSON `gid` is the forum page id | **pass-through** (field becomes work_id) |
| `apps/api/internal/galgame/handler/edit_handler.go:642` | `workID, appErr := h.workIDOf(c.Context(), int64(gid))` | `Mine?gid=` filter → catalog `EntityID` | **delete** |
| `apps/api/internal/galgame/handler/edit_handler.go:719` | `workID, appErr := h.workIDOf(c.Context(), gid)` | Game-scoped proposal list | **delete** |
| `apps/api/internal/galgame/handler/cover_vote_handler.go:56` | `workID, appErr := h.workIDOf(c, gid)` | Path gid → catalog work id before cover vote | **delete** |
| `apps/api/internal/galgame/handler/cover_vote_handler.go:78` | `func (h *CoverVoteHandler) workIDOf(c fiber.Ctx, gid int64) (int64, *errors.AppError) {` | Cover-vote translator via `CatalogWorkIDs` | **delete** |
| `apps/api/internal/galgame/service/cover_votes.go:16` | `ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})` | Detail-page cover tallies: gid → catalog work id | **delete** |
| `apps/api/internal/galgame/service/submission_service.go:84` | `gid := int(res.ProductWorkID)` | After mint, catalog’s returned product/site work id is stored as local `galgame.id` | **pass-through** (mint already returns catalog id as both; after G0 they stay equal) |
| `apps/api/internal/galgame/service/submission_service.go:118` | `catalogclient.ClaimActionClaim, catalogclient.UserClaimActionRequest{ProductWorkID: workID})` | Adopt: sends catalog work id as kungal `site_work_id` | **pass-through** (after G0 that is the forum id) |
| `apps/api/internal/galgame/service/submission_service.go:181` | `workID, appErr := s.workIDOf(ctx, gid)` | Delete-draft: forum gid → catalog work id | **delete** |
| `apps/api/internal/galgame/service/submission_service.go:205` | `workID, appErr := s.workIDOf(ctx, gid)` | Submit/withdraw claim action | **delete** |
| `apps/api/internal/galgame/service/submission_service.go:218` | `func (s *SubmissionService) workIDOf(ctx context.Context, gid int) (int64, *errors.AppError) {` | Submission translator via `CatalogWorkIDs` | **delete** |
| `apps/api/internal/galgame/service/claim_review_service.go:68` | `GID:     row.GID(),` | Admin pending-queue row id is `gid()` | **pass-through** |
| `apps/api/internal/galgame/service/claim_review_service.go:97` | `ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})` | Review action: forum gid → catalog work id | **delete** |
| `apps/api/internal/galgame/service/playtime.go:45` | `ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})` | Detail own-playtime: gid → catalog work id | **delete** |
| `apps/api/internal/galgame/service/playtime.go:113` | `ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})` | Report playtime / work-state | **delete** |
| `apps/api/internal/galgame/service/playtime.go:184` | `ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})` | Rating → catalog work-state sync | **delete** |
| `apps/api/internal/galgame/service/playtime.go:263` | `gidByWork, appErr := s.galgameClient.GIDsByCatalogIDs(ctx, workIDs)` | List-mine: catalog playtime work ids → forum gids | **delete** |
| `apps/api/internal/galgame/service/playtime.go:391` | `gid, ok := gidByWork[workID]` | Folded playtime row’s forum page id | **pass-through** |
| `apps/api/internal/galgame/service/galgame_user_stats.go:111` | `gidByWork, appErr := s.galgameClient.GIDsByCatalogIDs(ctx, workIDs)` | Merged edit proposals → forum gids for “contributed” | **delete** |
| `apps/api/internal/galgame/service/galgame_edit_revision_sync.go:156` | `gids, appErr := s.galgameClient.GIDsByCatalogIDs(ctx, workIDs)` | Revision feed entity ids → local `galgame_activity.galgame_id` | **delete** |
| `apps/api/internal/galgame/apiv1/moyu.go:29` | `CatalogWorkIDForGID(ctx context.Context, gid int) (int64, bool, *legacyErrors.AppError)` | v1 moyu patches: path `galgame_id` is a forum gid | **delete** |
| `apps/api/internal/galgame/apiv1/moyu.go:97` | `catalogID, found, appErr := s.works.CatalogWorkIDForGID(ctx, gid)` | Translate before calling moyu `refs=catalog:<id>` | **delete** |
| `apps/web/shared/utils/galgameClaimState.ts:45` | `export const galgameClaimGid = (item: UserClaimItem): number =>` | Frontend: claim `product_work_id` is the forum gid | **delete** |
| `apps/web/shared/utils/galgameClaimState.ts:46` | `item.product_work_id ?? 0` | Explicitly refuses falling back to catalog `work_id` because the spaces overlap | **delete** |
| `apps/web/app/pages/admin/submissions.vue:61` | `const gidOf = (row: PendingClaim) => row.gid` | Admin queue field accessor (the value is already a forum gid from `row.GID()`) | **pass-through** (rename) |

### 1.5 Callers of `CatalogItemGID` / `gid()` on search/list rows

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/service/galgame_library.go:67` | `if gid := client.CatalogItemGID(&res.Items[i]); gid > 0 {` | Browse/search card id for `HydrateCardsByIDs` | **pass-through** |
| `apps/api/internal/galgame/service/resource_service.go:125` | `if gid := client.CatalogItemGID(&res.Items[i]); gid > 0 {` | Resource search: catalog hits → local resource filter ids | **pass-through** |
| `apps/api/internal/galgame/service/quiz_service.go:715` | `if gid := client.CatalogItemGID(&res.Items[i]); gid > 0 {` | Quiz galgame-option search | **pass-through** |
| `apps/api/internal/galgame/service/tag_service.go:209` | `func catalogItemsToNextMoe(ctx context.Context, items []client.CatalogWorkListItem) []dto.NextMoeGalgameItem {` | Shared mapper; each item’s `ID` is `it.gid()` | **pass-through** |
| `apps/api/internal/galgame/service/drafts_service.go:64` | `Items: s.enricher.ToCards(ctx, catalogItemsToNextMoe(ctx, res.Items)),` | Wizard supply cards keyed by gid | **pass-through** |
| `apps/api/internal/galgame/service/calendar_service.go:91` | `Items: s.enricher.ToCards(ctx, catalogItemsToNextMoe(ctx, page.Items)),` | Calendar cards keyed by gid | **pass-through** |
| `apps/api/internal/search/service/search_service.go:253` | `items = append(items, client.CatalogItemToNextMoeItem(ctx, &res.Items[i]))` | Site search galgame lane; card id is `gid()` | **pass-through** |
| `apps/api/internal/galgame/service/engine_service.go:68` | `memberIDs, appErr := s.galgameClient.CatalogMemberGIDs(ctx,` | Engine detail member list is forum gids | **pass-through** |
| `apps/api/internal/galgame/service/tag_service.go:186` | `memberIDs, appErr := s.galgameClient.CatalogMemberGIDs(ctx,` | Tag detail members | **pass-through** |
| `apps/api/internal/galgame/service/series_service.go:171` | `memberIDs, appErr := s.galgameClient.CatalogMemberGIDs(ctx,` | Series detail members | **pass-through** |
| `apps/api/internal/galgame/service/official_service.go:72` | `members, appErr := s.galgameClient.CatalogLabelRollupMembers(ctx, id,` | Official rollup members; each `m.GID` is `gid()` | **pass-through** |
| `apps/api/internal/galgame/service/character_service.go:98` | `items = append(items, client.CatalogItemToNextMoeItem(ctx, &row))` | Character works: catalog work fetched by catalog id, card id still `gid()` | **pass-through** |
| `apps/api/internal/galgame/service/staff_service.go:88` | `rows, appErr := s.galgameClient.CatalogRowsByCatalogIDs(ctx, ids, isSFW)` | Staff credits: catalog ids in, `CatalogItemToNextMoeItem` emits gids | **pass-through** |
| `apps/api/internal/galgame/service/galgame_service.go:180` | `rows, appErr := s.galgameClient.CatalogRowsByGIDs(ctx, []int{galgameID}, "names", "all")` | Local owner-name lookup keyed by forum gid | **delete** (the gid keying) |
| `apps/api/internal/galgame/service/galgame_service.go:198` | `d, found, appErr := s.galgameClient.CatalogWorkDetail(ctx, galgameID)` | Detail page: path id is a forum gid | **pass-through** |
| `apps/api/internal/galgame/service/galgame_service.go:425` | `briefMap, appErr := s.galgameClient.GetBatchPublic(ctx, ids, isSFW)` | Card hydrate keyed by forum gid | **pass-through** |
| `apps/api/internal/galgame/service/rating_service.go:404` | `if d, found, appErr := s.galgameClient.CatalogWorkDetail(ctx, galgameID); appErr == nil && found {` | Rating-detail game header | **pass-through** |
| `apps/api/internal/galgame/service/galgame_proxy_service.go:36` | `rows, appErr := s.galgameClient.CatalogWorkLinks(ctx, gidInt)` | `/galgame/:gid` links proxy | **pass-through** |
| `apps/api/internal/galgame/service/galgame_catalog_mirror.go:199` | `rows, appErr := s.galgameClient.MirrorByGIDs(ctx, chunk)` | Fill lane asks catalog by local gid | **delete** |
| `apps/api/internal/community/anchor/anchor.go:141` | `briefs, appErr := r.galgame.GetBatch(ctx, gids)` | Name enrichment for `site_game` walls; ids are forum gids from `anchor_id` | **pass-through** |

GetBatch / GetBatchPublic / GetBatchDetailPublic (all go through `CatalogRowsByGIDs`) are also called from `apps/api/internal/ranking/service/ranking_service.go:38`, `apps/api/internal/user/service/user_content_service.go:87`, `:106`, `:356`, `:407`, `apps/api/internal/home/service/home_service.go:103`, `apps/api/internal/rss/handler/rss_handler.go:54`, `apps/api/internal/activity/service/activity_service.go:894`, `:941`, `apps/api/internal/galgame/service/collection_service.go:444`, `apps/api/internal/galgame/service/resource_service.go:626`, `:641`, `apps/api/internal/galgame/service/rating_service.go:374`, `:389`, `apps/api/internal/galgame/service/quiz_service.go:687`. Each of those `ids` / `galgameIDs` is a forum gid today. **pass-through** after the batch functions stop translating.

### 1.6 Merge-sync as a mapping consumer

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/service/galgame_merge_sync.go:212` | `s.galgameClient.ForgetGIDs(candidates)` | Drop gid cache so a merged-away gid is not a 30-minute stale hit | **delete** |
| `apps/api/internal/galgame/service/galgame_merge_sync.go:213` | `alive, appErr := s.galgameClient.GIDsToCatalogIDs(ctx, candidates)` | A local id that still resolves is **not** the merged-away page (overlap trap) | **delete** |
| `apps/api/internal/galgame/service/galgame_merge_sync.go:232` | `gidOfWork, appErr := s.galgameClient.GIDsForCatalogIDs(ctx, works)` | Survivor catalog id → forum gid | **delete** |
| `apps/api/internal/galgame/service/galgame_merge_sync.go:240` | `back, appErr := s.galgameClient.GIDsToCatalogIDs(ctx, roundTrip)` | Round-trip: unclaimed survivor gid must map back to the same work | **delete** |
| `apps/api/internal/galgame/service/galgame_merge_sync.go:295` | `stale, appErr := s.galgameClient.StaleGIDsForCatalogIDs(ctx, works)` | Extra curated gids on the survivor are pages to fold | **delete** |
| `apps/api/internal/galgame/service/galgame_merge_sync.go:343` | `resolved, appErr := s.galgameClient.GIDsToCatalogIDs(ctx, liveStale)` | Stale gid must resolve **to this work** before fold | **delete** |
| `apps/api/internal/galgame/service/galgame_merge_sync.go:403` | `s.galgameClient.ForgetGIDs([]int{oldGID})` | After fold, the dead gid must not resolve from cache | **delete** |

After G0 the merge-sync still consumes `/v2/catalog/redirects` (catalog ids). Decision 4 says merged-away **URLs** 404; the fold of child rows remains. The gid-bridge proofs above go away because `galgame.id` is the catalog id.

### 1.7 `product_work_id` / `site_work_id` as the forum gid in catalog clients

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/pkg/catalogclient/user_claims.go:12` | `ProductWorkID     int64          `json:"product_work_id,omitempty"`` | Submit body: kungal’s own id for the work | **external** (catalog claim `site_work_id`) |
| `apps/api/pkg/catalogclient/user_claims.go:30` | `body["site_work_id"] = strconv.FormatInt(req.ProductWorkID, 10)` | v2 mint/claim writes forum gid as `site_work_id` | **external** |
| `apps/api/pkg/catalogclient/user_claims.go:40` | `return &WorkSubmitResult{WorkID: id, ProductWorkID: id, ClaimState: out.State}, nil` | After v2 mint, both fields are the **catalog** id (lazy row: gid already equals work id) | **pass-through** |
| `apps/api/pkg/catalogclient/user_claims.go:54` | `body["site_work_id"] = strconv.FormatInt(req.ProductWorkID, 10)` | Claim action: forum gid (or catalog id on adopt) as `site_work_id` | **external** |
| `apps/api/pkg/catalogclient/claims.go:65` | `ProductWorkID *int64` | Claim-event feed: kungal page id of the affected work | **external** |
| `apps/api/pkg/catalogclient/claims.go:148` | `if n := parseFlexID(rows[i].ProductWorkID); n > 0 {` | Parses `product_work_id` JSON as the forum gid | **external** |
| `apps/api/pkg/catalogclient/work_revisions.go:15` | `ProductWorkID *int64    `json:"product_work_id"`` | Contributor feed: Go name still `ProductWorkID` | **external** |
| `apps/api/pkg/catalogclient/work_revisions.go:54` | `if n := parseFlexID(it.SiteWorkID); n != 0 {` | v2 revision `site_work_id` mapped into `ProductWorkID` (forum gid) | **external** |
| `apps/api/pkg/catalogclient/v2_user.go:223` | `SiteWorkID    json.RawMessage `json:"site_work_id"`` | v2 revision wire: claiming site’s id | **external** |
| `apps/api/internal/galgame/service/galgame_claim_event_sync.go:157` | `if ev.ProductWorkID == nil \|\| *ev.ProductWorkID <= 0 {` | Unanchored claim event is skipped; the anchor is the forum gid | **external** |
| `apps/api/internal/galgame/service/galgame_claim_event_sync.go:208` | `gid := int(*ev.ProductWorkID)` | Seeds / unpublishes local `galgame.id = product_work_id` | **data-rewrite** of stored rows; after G0 the incoming value is already the work id (**pass-through** + **external** rewrite of legacy claims) |
| `apps/api/internal/galgame/service/galgame_contributor_sync.go:97` | `gid := *it.ProductWorkID` | Contributor upsert keyed by forum gid | **data-rewrite** of `galgame_contributor.galgame_id`; incoming feed after infra rewrite is **pass-through** |
| `apps/web/shared/types/galgame.ts:227` | `product_work_id: number \| null` | Frontend claim item: forum gid | **delete** (use `work_id`) |

`LookupWikiLabel` (`apps/api/internal/galgame/client/catalog_taxonomy.go:289`) reuses `anchorSourceKeys` to resolve a **wiki label / official id**, not a galgame gid. **decide**: same source-key list as the gid bridge; it is a different identity and must not be deleted as part of a gid-bridge sweep without a separate ruling.

### 1.8 Tests that pin the translation

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/client/catalog_rename_test.go:25` | `if got := it.gid(); got != tc.want {` | `galgame_wiki` site still yields the claim’s SiteWorkID as gid | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:38` | `// site_work_id and work_id are different numbers on the same row: the first is` | Dual-space fixture | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:53` | `t.Errorf("site_work_id = %d, want the forum gid 777", it.Claim.SiteWorkID)` | 777 is a forum gid, 4242 is a catalog id | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:80` | `ids, appErr := c.catalogIDsForGIDs(t.Context(), []int{7})` | `galgame_wiki:7` resolves to catalog 9001 | **delete** |
| `apps/api/internal/galgame/client/catalog_stale_gid_test.go:12` | `` `{"source":"curated","external_id":"5904"},`+ `` | Extra curated gid 5904 is stale vs canonical 61295 | **delete** |
| `apps/api/internal/galgame/client/catalog_stale_gid_test.go:25` | `got, appErr := c.StaleGIDsForCatalogIDs(t.Context(), []int64{61101})` | Catalog 61101 → stale gids 5904 and 7001 | **delete** |
| `apps/api/internal/galgame/client/catalog_face_test.go:441` | `func TestCatalogMemberGIDsIsNeverNil(t *testing.T) {` | Member walk returns gids | **pass-through** (keep the walk, drop gid naming) |
| `apps/api/internal/galgame/client/catalog_v2_test.go:315` | `ids, appErr := c.CatalogWorkIDs(ctx, gids)` | Live mapping assertion | **delete** |
| `apps/api/internal/app/v1_galgame_moyu_test.go:47` | `func (f fakeWorkIDs) CatalogWorkIDForGID(_ context.Context, gid int) (int64, bool, *legacyErrors.AppError) {` | Fake: forum gid `moyuGID` → catalog `moyuCatalogID` | **delete** |
| `apps/api/internal/galgame/service/claim_unclaimed_test.go:71` | `if !strings.Contains(rec.bodies[0], `"site_work_id":"4649"`) && !strings.Contains(rec.bodies[0], `"product_work_id":4649`) {` | Claim body must name forum gid 4649 | **external** |
| `apps/api/internal/galgame/service/submission_submit_test.go:129` | `if _, present := rec.body["product_work_id"]; present {` | Kungal mint must **not** send a product/site work id (catalog assigns it) | **pass-through** |
| `apps/web/shared/utils/galgameClaimState.spec.ts:10` | `({ work_id: 4649, product_work_id }) as UserClaimItem` | Fixture: catalog 4649 vs a possibly different `product_work_id` | **delete** |

---

## 2. Places that skip the mapping

Calls into `pkg/catalogclient` or `internal/galgame/client` that send a **forum gid** as if it were a **catalog work id** (or the reverse: treat a catalog folder `work_id` as a forum gid). Examined every `s.catalog.` / `h.catalog.` call in `apps/api/internal/**/*.go` (66 hits of `s.catalog.`).

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/service/collection_service.go:234` | `holding, err := s.catalog.MyFoldersContaining(ctx, token, int64(galgameID))` | Forum gid sent as `contains_work_id` | **external** (catalog folder memberships currently store gids as work ids) |
| `apps/api/internal/galgame/service/collection_service.go:247` | `if err := s.catalog.PutFolderItem(ctx, token, fid, int64(galgameID)); err != nil {` | Forum gid written to catalog as folder item `work_id` | **external** |
| `apps/api/internal/galgame/service/collection_service.go:255` | `if err := s.catalog.DeleteFolderItem(ctx, token, fid, int64(galgameID)); err != nil {` | Forum gid deleted from catalog as `work_id` | **external** |
| `apps/api/internal/galgame/service/collection_service.go:307` | `holding, err := s.catalog.MyFoldersContaining(ctx, token, int64(galgameID))` | Picker: same skip | **external** |
| `apps/api/pkg/catalogclient/user_folders.go:167` | `q.Set("contains_work_id", strconv.FormatInt(workID, 10))` | Query param is whatever the caller passed (today: a gid) | **external** |
| `apps/api/pkg/catalogclient/user_folders.go:290` | `func (c *Client) PutFolderItem(ctx context.Context, token string, folderID, workID int64) error {` | Path `/v2/me/folders/{folder}/items/{workID}` | **external** |
| `apps/api/internal/galgame/service/galgame_service.go:124` | `holdings, err := s.catalog.MyFolderHoldings(ctx, token, ids)` | `GetMyInteractions` `workIDs` query values are forum gids, sent as catalog `work_ids` | **external** |
| `apps/api/internal/galgame/service/galgame_service.go:116` | `for _, id := range workIDs {` | Those ids are the galgame page ids on screen | **external** |
| `apps/api/internal/galgame/service/galgame_service.go:149` | `folders, err := s.catalog.MyFoldersContaining(ctx, token, int64(galgameID))` | Heart on the detail page: forum gid as catalog work id | **external** |
| `apps/api/pkg/catalogclient/user_folders.go:182` | `q.Set("work_ids", joinInt64s(chunk))` | Holdings batch: same skip | **external** |
| `apps/api/internal/galgame/service/resource_service.go:393` | `if _, appErr := adoptAndPublish(ctx, s.catalog, accessToken, int64(gid)); appErr != nil {` | First-resource silent claim: **forum gid used as catalog `work_id` and as `site_work_id`** | **external** (this is the skip; after G0 it becomes **pass-through**) |
| `apps/api/internal/galgame/service/collection_service.go:214` | `out = append(out, int(it.WorkID))` | Catalog folder item `work_id` used as local `galgame.id` for `DecrementFavoriteCounts` | **pass-through** after catalog rewrites stored items; today the stored value **is** a gid because of the write skip |
| `apps/api/internal/galgame/service/collection_service.go:357` | `ids := pageOfWorkIDs(items, page, limit)` | Folder page: catalog `WorkID`s passed to `HydrateCardsByIDs` (GetBatchPublic, which is the **mapping** keyed by gid) | **pass-through** after both sides share work ids |
| `apps/api/internal/galgame/service/collection_service.go:437` | `byFolder[f.ID] = append(byFolder[f.ID], int(it.WorkID))` | Preview covers: catalog work id used as GetBatchPublic gid | **pass-through** |
| `apps/api/internal/galgame/service/collection_items.go:16` | `WorkID    int64  `json:"w"`` | Redis `kungal:folder-items:v1:` cache stores catalog folder `work_id`s (today: gids) | **external** (flush on cutover) |

Catalog-client calls that **do** go through `CatalogWorkIDs` / `workIDOf` (not skips): playtime, cover votes, edit proposals, claim review, submission act/delete-draft. Cover **vote** HTTP (`user_votes.go:55`) then ignores `workID` and addresses `/v2/me/cover-votes/{coverID}`; `WorkCoverVotes` / `WorkCoversUser` still send the mapped catalog work id.

---

## 3. gids sent to or stored in other services

### 3.1 Community (`site_game` / `anchor_id`)

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/pkg/communityclient/types.go:5` | `AnchorSiteGame      = 1` | Community anchor kind for a galgame wall | **external** (community service) |
| `apps/api/internal/wall/apiv1/subject.go:45` | `typ: "galgame", anchorKind: communityclient.AnchorSiteGame,` | v1 wall `subject_type=galgame` → `anchor_kind=1` | **external** |
| `apps/api/internal/wall/apiv1/subject.go:47` | `maxLength: 5000, feedType: "GALGAME_COMMENT_CREATION", linkPrefix: "/galgame/",` | Wall URL `/galgame/<id>`; `<id>` is the forum gid | **data-rewrite** of stored `anchor_id`s; live code **pass-through** after both sides share work ids |
| `apps/api/internal/wall/apiv1/subject.go:82` | `return sp.anchorPrefix + strconv.Itoa(id)` | For galgame, prefix is empty: `anchor_id` **is** the decimal gid | **external** |
| `apps/api/internal/wall/apiv1/write.go:144` | `AnchorKind: spec.anchorKind, AnchorID: spec.anchorID(sid),` | Create comment: `sid` is the path subject id (forum gid) | **external** |
| `apps/api/internal/wall/apiv1/effects.go:87` | `AnchorKind: sub.spec.anchorKind, AnchorID: sub.spec.anchorID(sub.id),` | Follow/state keyed by the same gid | **external** |
| `apps/api/internal/community/anchor/anchor.go:70` | `gid, err := strconv.Atoi(ref.ID)` | Resolve `site_game` `anchor_id` as a forum gid | **data-rewrite** of existing threads; then **pass-through** |
| `apps/api/internal/community/anchor/anchor.go:74` | `out[ref] = Target{Link: "/galgame/" + ref.ID, Label: "Galgame", GalgameID: gid}` | Inbox/profile link uses the stored anchor id | **data-rewrite** |
| `apps/api/internal/community/notify/map.go:26` | `link = "/galgame/" + n.AnchorID + "?comment=" + strconv.FormatInt(*n.PostID, 10)` | Notification link: rewrite `/galgame/<n>` only; `comment=` is a community post id | **data-rewrite** (`<n>`); **pass-through** (`comment`) |
| `apps/api/internal/galgame/service/galgame_merge_sync.go:417` | `"old_anchor", "site_game:"+strconv.Itoa(oldGID),` | Fold does **not** move the community thread; logs the stranded `site_game:<gid>` | **decide** (infra must re-anchor the 10 G0 duplicate folds and every later catalog merge; decision 4 404s the old URL) |
| `apps/api/internal/galgame/repository/galgame_merge_repo.go:272` | `// at site_game:<gid>, which lives in infra and does not move when the forum` | Same constraint | **decide** |
| `apps/api/internal/user/service/galgame_comment_pagination.go:51` | `res, err := s.community.AuthorPosts(ctx, int64(userID), after, communityPostBatchSize, communityclient.AnchorSiteGame)` | Profile comments: threads whose `anchor_id` is a gid | **external** |

### 3.2 Trust

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/trust/gate/kinds.go:7` | `SubjectKindGalgame        = "galgame"` | Trust subject kind whose `subject_id` **would** be a galgame id | **decide** (kind is registered; no live `ScanBg` with this kind was found) |
| `apps/api/internal/trust/gate/kinds.go:26` | `SubjectKindGalgame,` | Ensured at startup into the trust service | **external** (trust) |
| `apps/api/internal/trust/gate/scan.go:20` | `SubjectKindGalgameCollection = "galgame_collection"` | Collection scans use the **local alias id**, not a gid | (not a gid) |
| `apps/api/internal/galgame/service/collection_service.go:102` | `s.scan.ScanBg(gate.SubjectKindGalgameCollection, strconv.Itoa(id), moderationText, int64(userID))` | `id` is `galgame_collection.id` | (not a gid) |
| `apps/api/internal/galgame/service/rating_service.go:215` | `s.scan.ScanBg(gate.SubjectKindGalgameRating, strconv.Itoa(rating.ID), req.ShortSummary, int64(userID))` | Rating id, not gid | (not a gid) |
| `apps/api/internal/galgame/service/resource_service.go:380` | `s.scan.ScanBg(gate.SubjectKindGalgameResource, strconv.Itoa(res.ID), moderationText, int64(userID))` | Resource id | (not a gid) |
| `apps/api/internal/galgame/service/quiz_service.go:256` | `s.scan.ScanBg(gate.SubjectKindGalgameQuiz, strconv.Itoa(quiz.ID), moderationText, int64(userID))` | Quiz id | (not a gid) |
| `apps/api/internal/galgame/service/comment_enforce.go:6` | `SubjectKindGalgameComment = "galgame_comment"` | Legacy comment id via `galgame_comment_community_map.old_comment_id` | (not a gid) |

Searched `ScanBg(gate.SubjectKindGalgame` — 0 call sites. Trust `galgame` kind is reserved; live scans use rating/resource/collection/quiz/comment ids.

### 3.3 Moemoepoint idempotency refs

`moemoepoint.Ref(kind, id)` → `kind + ":" + id` (`apps/api/internal/moemoepoint/pusher.go:129`). OAuth idempotency key is `kungal:<event>:<ref>` via `moemoepoint.Key` / `KeyNonce`. After the renumber a **new** work id can equal a **retired** gid of a different game; reusing `galgame:<n>` would dedupe a real like/approval.

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/moemoepoint/pusher.go:129` | `func Ref(kind string, id int) string {` | Builds `kind:id` | **decide** (whether `galgame:` refs are rewritten, namespaced, or left to collide) |
| `apps/api/internal/galgame/service/galgame_service.go:87` | `moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", galgameID))` | Unlike: ref is forum gid | **decide** + **external** (OAuth) |
| `apps/api/internal/galgame/service/galgame_service.go:91` | `moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", galgameID))` | Like: same | **decide** + **external** |
| `apps/api/internal/galgame/service/collection_service.go:285` | `moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", galgameID))` | First-add / last-remove favourite | **decide** + **external** |
| `apps/api/internal/galgame/service/galgame_claim_event_sync.go:247` | `moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame", gid),` | Claim approval award; **key** is `claim_approved:<eventID>` (stable), **ref** is `galgame:<gid>` | **decide** (ref collision) / key is event-scoped |
| `apps/api/internal/galgame/handler/edit_handler.go:845` | `moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_pr", int(prop.EntityID)),` | Ref is catalog **entity / work id**, already, with key `galgame_edit_merged:<proposalID>` | **pass-through** (already catalog space; kind `galgame_pr` is not `galgame`) |
| `apps/api/internal/galgame/service/rating_service.go:205` | `moemoepoint.Ref("galgame_rating", rating.ID)` | Rating id | (not a gid) |
| `apps/api/internal/galgame/service/resource_service.go:370` | `moemoepoint.Ref("galgame_resource", res.ID)` | Resource id | (not a gid) |
| `apps/api/internal/galgame/service/quiz_service.go:246` | `moemoepoint.Ref("galgame_quiz", quiz.ID)` | Quiz id | (not a gid) |
| `apps/api/internal/galgame/service/interaction.go:15` | `moemoepoint.Award(userID, delta, reason, ref, moemoepoint.KeyNonce(reason, ref))` | Like/favourite key is nonce-suffixed; OAuth still stores `ref` as `galgame:<gid>` | **decide** |

`moemoepoint.Ref("` hits examined: 49. Kind `galgame` (the gid-bearing kind): 3 production call sites above. Kind `galgame_pr` uses catalog entity id. Like/favourite use `KeyNonce(reason, ref)` so the **key** is unique per call; OAuth still **dedupes on `ref`** for the same user/reason if the platform treats ref as the idempotency identity — **decide**.

### 3.4 Image service

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/image/service/galgame_upload.go:32` | `res, err := s.catalogClient.UploadEditImageUser(ctx, accessToken, r, filename, preset)` | Upload is content-addressed; **no gid** is sent | (none) |

Searched image client / upload paths for `galgame_id` / gid as an image key: none. Covers live on catalog works (catalog ids after mapping).

### 3.5 OpenSearch / search

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/search/service/search_service.go:244` | `res, appErr := s.galgameClient.CatalogWorksSearch(ctx, q)` | Galgame search is catalog’s works-search; card ids are `gid()` | **pass-through** |

Searched `OpenSearch`, `opensearch`, `elastic`, `meilisearch`, `typesense`, `zincsearch` in `*.go`/`*.ts`/`*.vue`: **0 hits**. No local search index stores gids.

### 3.6 OG card service

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/web/shared/utils/ogCard.ts:21` | `export const kunOgCardPath = (kind: KunOgCardKind, id: number): string =>` | Card URL `/og/galgame/<id>`; `<id>` is the forum page id | **pass-through** (URL number becomes the work id; old numbers are not redirected per decision 3) |
| `apps/web/server/utils/kunOgCard.ts:103` | `const buildGalgame = async (id: number): Promise<KunOgCard \| null> => {` | Fetches `/galgame/${id}` with `galgame_id: id` | **pass-through** |
| `apps/web/server/utils/kunOgCard.ts:104` | `const game = await fetchKunApi<GalgameDetail>(`/galgame/${id}`, {` | Forum gid today | **pass-through** |
| `apps/web/server/routes/og/[kind]/[id].ts:9` | `const kind = getRouterParam(event, 'kind') as KunOgCardKind` | Path param `id` for kind `galgame` | **pass-through** |

OG renderer (`og.nextmoe.dev`) is addressed with a signed card payload, not a stored gid. Old `/og/galgame/<retired-gid>` will 404 with the page.

### 3.7 Catalog cover votes, edit proposals, claims, folders, moyu

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/pkg/catalogclient/cover_votes.go:31` | `if err := c.appV2JSON(ctx, "/v2/catalog/works/"+strconv.FormatInt(workID, 10), q, &rec); err != nil {` | Cover tallies: catalog work id (mapped first) | **external** (catalog); code path **pass-through** after G0 |
| `apps/api/pkg/catalogclient/user_edit.go:277` | `"/v2/catalog/works/"+strconv.FormatInt(workID, 10)+"/covers?nsfw=true"` | User-lane cover tallies | **external** |
| `apps/api/internal/galgame/handler/edit_handler.go:407` | `EntityType: entityTypeGame, EntityID: workID,` | Edit proposal entity id is catalog work id (via `workIDOf`) | **external** |
| `apps/api/pkg/catalogclient/user_claims.go:52` | `body := map[string]any{"work_id": id}` | Claim POST names catalog work id | **external** |
| `apps/api/pkg/moyuclient/client.go:90` | `"refs":    {"catalog:" + strconv.FormatInt(catalogWorkID, 10)},` | Moyu patches: catalog work id after `CatalogWorkIDForGID` | **external** (moyu); translation **delete** |
| `apps/api/internal/galgame/apiv1/moyu.go:21` | `moyuCacheKey = "moyu:patches:v1:"` | Redis key suffix is the **catalog** id | **pass-through** (already catalog space) |

Catalog **folder items** (section 2) currently **store forum gids as `work_id`**. Infra must rewrite those rows in the same window or every favourite points at the wrong work once kungal’s ids move. **external** + **decide** (who rewrites catalog `folder_item.work_id` and kungal claim `site_work_id` / `product_work_id`).

---

## 5. Cron jobs and background workers

Jobs registered in `apps/api/internal/infrastructure/cron/cron.go` (Asia/Shanghai). Gid-keyed ones:

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/infrastructure/cron/cron.go:74` | `schedule(c, "*/10 * * * *", "galgame claim 同步", jobs.GalgameClaimSync)` | Claim-events feed consumer | see 5.1 |
| `apps/api/internal/infrastructure/cron/cron.go:78` | `schedule(c, "*/10 * * * *", "galgame revision 同步", jobs.GalgameRevisionSync)` | Edit-revision → `galgame_activity` | see 5.2 |
| `apps/api/internal/infrastructure/cron/cron.go:82` | `schedule(c, "*/15 * * * *", "galgame contributor 同步", jobs.GalgameContributorSync)` | Revisions → `galgame_contributor.galgame_id` | see 5.3 |
| `apps/api/internal/infrastructure/cron/cron.go:86` | `schedule(c, "*/10 * * * *", "galgame catalog 镜像信道同步", jobs.GalgameCatalogMirror)` | `/v2/catalog/changes` → local `content_limit` / `release_date` | see 5.4 |
| `apps/api/internal/infrastructure/cron/cron.go:97` | `schedule(c, "*/2 * * * *", "galgame catalog 镜像增量同步", jobs.GalgameCatalogMirrorFill)` | Fill lane: local gids through `MirrorByGIDs` | see 5.4 |
| `apps/api/internal/infrastructure/cron/cron.go:101` | `schedule(c, "*/30 * * * *", "galgame 合并同步", jobs.GalgameMergeSync)` | `/v2/catalog/redirects` → fold | see 5.5 |
| `apps/api/internal/infrastructure/cron/cron.go:48` | `schedule(c, "0 0 * * *", "浏览量滚动统计", func() {` | `galgame_view_daily.entity_id` is a gid | **data-rewrite** of daily rows; rollup itself **pass-through** |
| `apps/api/internal/infrastructure/viewstats/viewstats.go:23` | `func BumpDaily(db *gorm.DB, table string, entityID int) error {` | `entityID` for `galgame_view_daily` is `galgame.id` | **data-rewrite** |
| `apps/api/internal/galgame/repository/galgame_repo.go:87` | `_ = viewstats.BumpDaily(r.db, viewstats.GalgameDaily, id)` | View increment keys the daily bucket by gid | **pass-through** after PK rewrite |

Not gid-keyed (listed so a “none found” is checkable): daily user-state reset (`cron.go:44`), upload-cache cleanup (`:56`), image reference-ping (`:61`), topic mini-app deadlines (`:107`), dlsite campaign refresh (`:111`).

### 5.1 Claim-events feed (`GalgameClaimEventSync`)

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/service/galgame_claim_event_sync.go:55` | `claimCursorKey = "catalog:claim:cron:since"` | Redis watermark is a **claim event id**, not a gid | **pass-through** |
| `apps/api/internal/galgame/service/galgame_claim_event_sync.go:93` | `page, err := s.catalog.ClaimEventsSince(ctx, maxSeen, s.batch, claimSite)` | `GET` claim-events, site=`kungal` | **external** |
| `apps/api/internal/galgame/service/galgame_claim_event_sync.go:208` | `gid := int(*ev.ProductWorkID)` | Local stub / unpublish keyed by forum gid | **data-rewrite** of `galgame.id`; after infra claim rewrite **pass-through** |
| `apps/api/internal/galgame/service/galgame_claim_event_sync.go:247` | `moemoepoint.Ref("galgame", gid),` | Approval award ref | **decide** (see 3.3) |
| `apps/api/internal/galgame/service/galgame_claim_event_sync.go:264` | `key := claimSubmitterKeyPrefix + strconv.FormatInt(ev.WorkID, 10)` | Redis submitter cache keyed by **catalog work id** | **pass-through** |

### 5.2 Edit-revision mirror (`GalgameEditRevisionSync`)

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/service/galgame_edit_revision_sync.go:156` | `gids, appErr := s.galgameClient.GIDsByCatalogIDs(ctx, workIDs)` | Feed `entity_id` is catalog work id → local `galgame_activity.galgame_id` | **delete** (mapping); column **data-rewrite** |
| `apps/api/migrations/067_galgame_activity_edit_revision_id.up.sql:14` | `-- against the old feed — every (galgame_id, revision) pair it ever delivered` | Activity cards still keyed by forum gid | **data-rewrite** |

### 5.3 Contributor sync

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/service/galgame_contributor_sync.go:97` | `gid := *it.ProductWorkID` | `galgame_contributor.galgame_id` = claim site work id | **data-rewrite** |
| `apps/api/migrations/069_galgame_contributor.up.sql:22` | `galgame_id     BIGINT NOT NULL,` | Deliberately **no FK** to `galgame` | **data-rewrite** (must be explicit; CASCADE will not touch it) |

### 5.4 Catalog changes / content_limit / release_date mirror

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/service/galgame_catalog_mirror.go:23` | `mirrorCursorKey = "catalog:changes:content-limit:cursor"` | Redis cursor on `/v2/catalog/changes` | **pass-through** |
| `apps/api/internal/galgame/service/galgame_catalog_mirror.go:125` | `page, appErr := s.galgameClient.CatalogChanges(ctx, cursor, client.CatalogChangesLimit)` | Feed items are catalog work ids | **pass-through** |
| `apps/api/internal/galgame/service/galgame_catalog_mirror.go:143` | `rows, appErr := s.galgameClient.MirrorByCatalogIDs(ctx, ids)` | Re-key by kungal `site_work_id` before `UPDATE galgame WHERE id = gid` | **delete** (re-key); after G0 update by catalog id |
| `apps/api/internal/galgame/service/galgame_catalog_mirror.go:199` | `rows, appErr := s.galgameClient.MirrorByGIDs(ctx, chunk)` | Pending local rows resolved **by gid** through the mapping layer | **delete** |
| `apps/api/internal/galgame/model/local.go:21` | `ContentLimit          *string    `gorm:"column:content_limit"` | Cached catalog verdict on the local row | **pass-through** (column stays; row id is rewritten) |
| `apps/api/internal/galgame/model/local.go:14` | `ReleaseDate           *time.Time `gorm:"column:release_date"` | Cached catalog date | **pass-through** |

This **is** the content_limit cache. There is no separate Redis content_limit map; the fill lane’s `unresolved` map (`galgame_catalog_mirror.go:72`) is in-process, keyed by local gid, TTL 6h.

### 5.5 Catalog redirects / merge-fold (`galgame_redirect`, `galgame_merge_discarded`, migrations 081/082)

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/service/galgame_merge_sync.go:24` | `MergeCursorKey = "catalog:redirects:merge:cursor"` | Redis cursor on `/v2/catalog/redirects` | **pass-through** |
| `apps/api/internal/galgame/service/galgame_merge_sync.go:32` | `mergeDeferredKey = "catalog:redirects:merge:deferred"` | Hash `oldGID → survivor catalog work id` | **data-rewrite** of parked keys (old gids) or flush |
| `apps/api/internal/galgame/service/galgame_merge_sync.go:104` | `page, appErr := s.galgameClient.CatalogRedirects(ctx, cursor, client.CatalogRedirectsLimit)` | Catalog ids only | **pass-through** |
| `apps/api/migrations/081_galgame_redirect.up.sql:21` | `old_gid integer     PRIMARY KEY,` | Dead forum gid; **no FK** (the galgame row is deleted) | **decide** (decision 4: merged-away id 404s; ledger exists to 301 today — GetDetail still returns `MovedTo`) |
| `apps/api/internal/galgame/service/galgame_service.go:207` | `if newGID, moved := s.mergeRepo.RedirectTarget(galgameID); moved {` | Detail 404 → `MovedTo` for the browser | **decide** (conflicts with decision 4 unless this path is removed) |
| `apps/api/migrations/082_galgame_merge_discarded.up.sql:12` | `CREATE TABLE IF NOT EXISTS galgame_merge_discarded (` | Archived unique-key losers (`old_gid`, `new_gid`, jsonb `row`) | **data-rewrite** of existing archive gids if they must stay queryable |

#### How `Fold` moves one page into another (reuse for the 10 G0 duplicate folds)

Implementation: `apps/api/internal/galgame/repository/galgame_merge_repo.go` `Fold(oldGID, newGID)`.

1. Load the dead `galgame` row. Seed the survivor with `ON CONFLICT DO NOTHING` copying `created` / `resource_update_time` (`:91`).
2. **Movable tables** (no unique peer key) — `UPDATE … SET galgame_id = new WHERE galgame_id = old` (`:23–28`, `:99`): `galgame_resource`, `galgame_activity`, `galgame_quiz`, `galgame_comment_community_map`.
3. **Contributor clash**: add `revision_count` and `LEAST/GREATEST` timestamps on matching `user_id`, then the unique-table path drops the dead row (`:110`).
4. **Ratings clash**: `preferEngagedRating` (`:247`) deletes the **survivor’s** rating when the dead one has more `like_count + comment_count`, after archiving.
5. **Unique tables** (`:34–40`): `galgame_like(user_id)`, `galgame_favorite(user_id)`, `galgame_rating(user_id)`, `galgame_contributor(user_id)`, `galgame_quiz_galgame(quiz_id)`. Move rows whose peer is free; archive then `DELETE` the rest (`:125–144`).
6. **`galgame_view_daily`**: `entity_id` (not `galgame_id`) — `INSERT … ON CONFLICT (entity_id, day) DO UPDATE SET count = count + EXCLUDED.count`, then delete the dead buckets (`:148–157`).
7. **Parent**: `view` summed, `published` OR, `resource_publish_banned` OR, `creator_user_id` COALESCE, timestamps LEAST/GREATEST (`:161`). Then `DELETE FROM galgame WHERE id = old` (`:173`). Like/favorite/resource are `ON DELETE CASCADE`; rating is `ON DELETE RESTRICT` — children must move first (`:62`).
8. **Feed**: `UPDATE feed_activity SET galgame_id = new, link = '/galgame/'||new WHERE galgame_id = old` for comment rows that have no trigger (`:184`). Trigger-backed sources already moved.
9. **Recount** like/resource/rating/contributor/view_7d/view_30d (`:275`). `favorite_count` and `comment_count` are **not** recounted (`:265–274`): favourites live in catalog folders; comments live at `site_game:<gid>` in community.
10. Ledger: `INSERT INTO galgame_redirect (old_gid, new_gid)` and chase chains `UPDATE galgame_redirect SET new_gid = survivor WHERE new_gid = old` (`:196–206`).

What Fold does **not** move: catalog folder items (infra merge rehangs those); community `site_game` threads; `message.link` / topic body `/galgame/<n>` (merge-sync does not rewrite user text; G0’s content rewrite is a separate pass); Redis `gidCache`.

G0’s 10 duplicate pairs should call this `Fold` (pick survivor, then `UPDATE galgame.id` on the keeper). Unique-key clashes use the same archive table.

---

## 6. DB schema

### 6.1 Context list, confirmed or corrected

Context tables/columns, checked against `apps/api/internal/galgame/model/**` and `apps/api/migrations/**`:

| table.column | Context | Status | notes |
|---|---|---|---|
| `galgame.id` | listed | confirmed | PK; GORM `GalgameLocal` |
| `galgame_collection_item.galgame_id` | listed | confirmed | Frozen snapshot since 091 alias cutover; **no FK** to `galgame` (043) |
| `feed_activity.galgame_id` | listed | confirmed | **no FK**; default 0 |
| `galgame_resource.galgame_id` | listed | confirmed | FK CASCADE/CASCADE (baseline) |
| `galgame_like.galgame_id` | listed | confirmed | FK CASCADE/CASCADE |
| `galgame_rating.galgame_id` | listed | confirmed | FK CASCADE **ON DELETE RESTRICT** |
| `galgame_activity.galgame_id` | listed | confirmed | **no FK** (021) |
| `galgame_quiz.galgame_id` | listed | confirmed, still present | 047 said deploy-then-drop; **no later `DROP COLUMN`**. GORM `GalgameQuiz` no longer maps it; `Fold` still `UPDATE galgame_quiz SET galgame_id` (`merge_repo.go:26`) |
| `galgame_quiz_galgame.galgame_id` | listed | confirmed | PK with `quiz_id`; **no FK** to `galgame` |
| `galgame_contributor.galgame_id` | listed | confirmed | Recreated in 069 **without FK** (005 dropped the wiki-era table that had CASCADE) |
| `galgame_comment_community_map.galgame_id` | listed | confirmed | **no FK** (057) |
| `galgame_favorite.galgame_id` | listed | confirmed, still live | 043 promised drop in 044; **there is no 044**. Baseline FK CASCADE/CASCADE still applies. `Fold` unique-table. Model `GalgameFavorite` |
| `galgame_redirect.old_gid` / `new_gid` | listed | confirmed | 081; **no FKs** |
| `galgame_merge_discarded.old_gid` / `new_gid` | listed | confirmed | 082 |
| `galgame_view_daily.entity_id` | listed | confirmed | **no FK** (050); gid in `entity_id` |
| `message.link` | listed | confirmed | `/galgame/<gid>` written in Go, not a column of type gid |
| `feed_activity.link` | listed | confirmed | `'/galgame/' \|\| id` in triggers |
| `topic.content` / `topic_reply.content` | listed | confirmed as **text** holding `/galgame/<n>` | rewrite is a content pass, not a typed column |

### 6.2 Gid-bearing columns the Context missed

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/migrations/070_feed_sync_galgame_published.up.sql:21` | `PERFORM feed_upsert('GALGAME_CREATION', NEW.id, 0, NEW.id, '', '/galgame/' \|\| NEW.id, false, NEW.created);` | `feed_activity.source_id` **and** `galgame_id` are both `galgame.id` for `GALGAME_CREATION` (unique `(type, source_id)`) | **data-rewrite** `source_id` as well as `galgame_id` |
| `apps/api/migrations/034_create_feed_activity.up.sql:29` | `p_type text, p_sid int, p_uid int, p_gid int,` | `p_gid` is the forum gid stored in `feed_activity.galgame_id` | **data-rewrite** of existing rows; function **pass-through** |
| `apps/api/internal/galgame/model/local.go:5` | `type GalgameLocal struct {` / `ID … primaryKey` | Local counters live on `galgame` keyed by gid | **data-rewrite** of PK |
| `apps/api/migrations/000_baseline.up.sql:614` | `CREATE SEQUENCE IF NOT EXISTS public.galgame_id_seq` | Sequence owned by `galgame.id`; after rewrite it must sit above the max work id | **decide** (moyu pushed the sequence to 2e9) |
| `apps/api/internal/galgame/service/galgame_catalog_mirror.go:72` | `unresolved   map[int]time.Time` | In-process set of local gids catalog did not resolve | **delete** / flush on cutover |
| `apps/api/internal/galgame/service/collection_items.go:21` | `return "kungal:folder-items:v1:" + strconv.FormatInt(folder.ID, 10) + ":" + folder.UpdatedAt` | Redis cache of folder `work_id`s (gids today) | **external** (flush) |
| `apps/api/internal/galgame/service/galgame_merge_sync.go:32` | `mergeDeferredKey = "catalog:redirects:merge:deferred"` | Redis hash of forum gids | **data-rewrite** or flush |
| `apps/api/internal/wall/apiv1/effects.go:66` | `if err := s.store.FeedUpsert(sub.spec.feedType, post.ID, authorID, galgameID,` | Hand-written `GALGAME_COMMENT_CREATION` `p_gid` | **data-rewrite** of those feed rows |

Dropped tables that **used** to have `galgame_id` (005 / 060) and must not be rewritten: `galgame_alias`, `galgame_comment`, `galgame_engine_relation`, `galgame_history`, `galgame_link`, `galgame_official_relation`, `galgame_pr`, `galgame_tag_relation`. Wiki-era `galgame_contributor` was dropped in 005 and recreated in 069.

### 6.3 Foreign keys to `galgame(id)`

Live FKs remaining from baseline (tables in 005/060 are gone):

| constraint (baseline) | column | ON UPDATE | ON DELETE | still live? |
|---|---|---|---|---|
| `galgame_like_galgame_id_fkey` | `galgame_like.galgame_id` | CASCADE | CASCADE | yes |
| `galgame_favorite_galgame_id_fkey` | `galgame_favorite.galgame_id` | CASCADE | CASCADE | yes (044 never shipped) |
| `galgame_rating_galgame_id_fkey` | `galgame_rating.galgame_id` | CASCADE | **RESTRICT** | yes |
| `galgame_resource_galgame_id_fkey` | `galgame_resource.galgame_id` | CASCADE | CASCADE | yes |

`ON UPDATE CASCADE` on those four means `UPDATE galgame SET id = work_id` **will rewrite the child column**. Everything else in §6.1 has **no FK** and will **not** follow a PK update: `galgame_collection_item`, `feed_activity` (`galgame_id` **and** `GALGAME_CREATION.source_id`), `galgame_activity`, `galgame_quiz.galgame_id`, `galgame_quiz_galgame`, `galgame_contributor`, `galgame_comment_community_map`, `galgame_view_daily.entity_id`, `galgame_redirect`, `galgame_merge_discarded`.

Quoted FK lines:

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/migrations/000_baseline.up.sql:4192` | `ADD CONSTRAINT galgame_like_galgame_id_fkey FOREIGN KEY (galgame_id) REFERENCES public.galgame(id) ON UPDATE CASCADE ON DELETE CASCADE;` | like row’s game | **data-rewrite** via CASCADE if PK is updated |
| `apps/api/migrations/000_baseline.up.sql:4148` | `ADD CONSTRAINT galgame_favorite_galgame_id_fkey FOREIGN KEY (galgame_id) REFERENCES public.galgame(id) ON UPDATE CASCADE ON DELETE CASCADE;` | frozen favourite snapshot | **data-rewrite** via CASCADE |
| `apps/api/migrations/000_baseline.up.sql:4324` | `ADD CONSTRAINT galgame_rating_galgame_id_fkey FOREIGN KEY (galgame_id) REFERENCES public.galgame(id) ON UPDATE CASCADE ON DELETE RESTRICT;` | rating; delete of parent is forbidden | **data-rewrite** via CASCADE; folds must move ratings first |
| `apps/api/migrations/000_baseline.up.sql:4368` | `ADD CONSTRAINT galgame_resource_galgame_id_fkey FOREIGN KEY (galgame_id) REFERENCES public.galgame(id) ON UPDATE CASCADE ON DELETE CASCADE;` | resource | **data-rewrite** via CASCADE |
| `apps/api/migrations/043_create_galgame_collection.up.sql:37` | `galgame_id    INTEGER     NOT NULL,` | collection item: **no** `REFERENCES galgame` | **data-rewrite** (explicit) |
| `apps/api/migrations/045_create_galgame_quiz.up.sql:30` | `galgame_id    INTEGER,                                  -- optional linked game; NULL = general trivia` | quiz: **no FK** | **data-rewrite** (explicit) |
| `apps/api/migrations/047_quiz_description_and_multi_galgame.up.sql:16` | `galgame_id INTEGER NOT NULL,` | join table: **no FK** to galgame | **data-rewrite** (explicit) |
| `apps/api/migrations/069_galgame_contributor.up.sql:15` | `-- Deliberately NO foreign key to galgame.` | contributor | **data-rewrite** (explicit) |
| `apps/api/migrations/021_create_galgame_activity.up.sql:24` | `galgame_id       INTEGER     NOT NULL,` | activity: **no FK** | **data-rewrite** (explicit) |
| `apps/api/migrations/034_create_feed_activity.up.sql:14` | `galgame_id INTEGER     NOT NULL DEFAULT 0,` | feed: **no FK** | **data-rewrite** (explicit) |
| `apps/api/migrations/050_add_view_stats.up.sql:16` | `entity_id BIGINT NOT NULL,` | view daily: **no FK** | **data-rewrite** (explicit) |
| `apps/api/migrations/057_galgame_community_comments.up.sql:15` | `galgame_id     int    NOT NULL` | comment map: **no FK** | **data-rewrite** (explicit) |
| `apps/api/migrations/081_galgame_redirect.up.sql:14` | `-- old_gid carries no foreign key: the fold deletes the galgame row it names.` | redirect ledger | **decide** (keep vs drop under decision 4) |

### 6.4 SQL functions / triggers that take or compute a gid

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/migrations/034_create_feed_activity.up.sql:28` | `CREATE OR REPLACE FUNCTION feed_upsert(` | `p_gid` → `feed_activity.galgame_id` | **pass-through** (signature stays; stored values rewritten) |
| `apps/api/migrations/070_feed_sync_galgame_published.up.sql:17` | `CREATE OR REPLACE FUNCTION feed_sync_galgame() RETURNS trigger AS $$` | `NEW.id` as both `p_sid` and `p_gid`; link `'/galgame/' \|\| NEW.id` | **pass-through** (fires on the new PK) |
| `apps/api/migrations/034_create_feed_activity.up.sql:94` | `PERFORM feed_upsert('GALGAME_RESOURCE_CREATION', NEW.id, NEW.user_id, NEW.galgame_id, '', '/galgame/' \|\| NEW.galgame_id, false, NEW.created);` | Resource trigger: `p_gid` and link from `galgame_resource.galgame_id` | **pass-through** after that column is rewritten (CASCADE or explicit) |
| `apps/api/migrations/034_create_feed_activity.up.sql:111` | `PERFORM feed_upsert('GALGAME_RATING_CREATION', NEW.id, NEW.user_id, NEW.galgame_id,` | Rating trigger | **pass-through** after CASCADE |
| `apps/api/migrations/035_filter_feed_nsfw.up.sql:91` | `SELECT galgame_id INTO v_gid FROM galgame_rating WHERE id = NEW.galgame_rating_id;` | Rating-comment feed copies the rating’s gid | **pass-through** |
| `apps/api/migrations/051_feed_galgame_quiz.up.sql:3` | `-- a trigger and backfilled from existing rows. galgame_id = 0 on purpose — a quiz` | Quiz feed **does not** store the linked game id (`p_gid = 0`); link is `/galgame-quiz/<quiz id>` | (not a gid) |
| `apps/api/internal/wall/repository/v1_store.go:164` | `return s.db.Exec("SELECT feed_upsert(?, ?, ?, ?, ?, ?, ?, ?)",` | Go caller of `feed_upsert`; 4th arg is gid for galgame walls | **data-rewrite** of existing comment feed rows |
| `apps/api/internal/galgame/service/interaction.go:27` | `link := fmt.Sprintf("/galgame/%d", galgameID)` | `message.link` for like/favourite | **data-rewrite** |
| `apps/api/internal/galgame/service/interaction.go:81` | `link := fmt.Sprintf("/galgame/%d?comment=%d", galgameID, commentID)` | Mention link; `comment=` is a community post id | **data-rewrite** (`<gid>` only) |
| `apps/api/internal/message/service/notifier.go:126` | `return fmt.Sprintf("/galgame/%d", spec.GalgameID)` | Notifier galgame links | **data-rewrite** of stored `message.link` |

Trigger coverage for `ON UPDATE` of `galgame_id` on resource/rating/like/activity: 034’s comment at merge_repo.go:68 says those tables have AFTER UPDATE triggers that re-upsert `feed_activity` with the new gid and link. A PK change on `galgame` that CASCADEs into `galgame_resource.galgame_id` will fire those updates. `GALGAME_CREATION` is keyed by `source_id = galgame.id` and is rewritten by `feed_sync_galgame` on UPDATE of `galgame` itself (`070`: upserts using `NEW.id`). If the PK update is implemented as `UPDATE galgame SET id = …`, that trigger sees the new id. If it is delete+insert, `GALGAME_CREATION` is deleted then re-inserted — **decide** the PK rewrite technique (moyu used UPDATE with CASCADE).
