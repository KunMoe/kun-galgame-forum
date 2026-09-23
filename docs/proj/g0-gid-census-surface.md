# G0 census — gid-bearing strings, web, v1 surface, mapping tests

Read-only census of the forum worktree `kun-galgame-forum-g-galgame` on branch `api-v1/g-galgame`. Sections 4, 7, 8, 9 only. Other G0 census sections are owned by a parallel executor.

G0 action vocabulary: **delete** (mapping layer, goes away) · **pass-through** (the value becomes the work id with no translation; only a rename) · **data-rewrite** (stored value must be rewritten by the renumber) · **external** (the number is sent to / stored in another service; named) · **decide** (orchestrator must rule; why in one line).

Every row is one live line. Taxonomy paths under `/galgame/{tag,official,engine,series,staff,character,resource,collection}/…` carry other ids; they are listed where the same `/galgame/` prefix would otherwise hide them.

---

## 4. gids embedded in strings

Positive control: searched `` `/galgame/${` `` (78 hits in `apps/web`), `"/galgame/" +` / `fmt.Sprintf("/galgame/%d"` / `"/galgame/" + strconv` (Go builders), `galgame#%d`, `site_game:`, `CacheKey`/`cacheKey`/`KeyPrefix`/`CursorKey`/`gidCache`/`moyu:patches`, `cursor` in `apps/` (~57 hits examined), `kind: 'galgame'` OG refs, `json-ld`/`canonical`/`og:url`, `rdb.(Get|Set)` under `internal/galgame`. Flagged rows below; taxonomy `/galgame/{family}` and `/galgame/resource|official|tag|engine/…` hits are listed with their true id meaning.

### 4.1 Go link builders

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/message/service/notifier.go:126` | `return fmt.Sprintf("/galgame/%d", spec.GalgameID)` | forum `galgame.id` written into `message.link` | pass-through (builder); stored rows are data-rewrite |
| `apps/api/internal/galgame/service/interaction.go:27` | `link := fmt.Sprintf("/galgame/%d", galgameID)` | forum `galgame.id` written into `message.link` (like / favorite / expired) | pass-through (builder); stored rows are data-rewrite |
| `apps/api/internal/galgame/service/interaction.go:81` | `link := fmt.Sprintf("/galgame/%d?comment=%d", galgameID, commentID)` | path segment is forum `galgame.id`; `comment` is a community post id | pass-through on the path segment; never rewrite `comment=` |
| `apps/api/internal/community/notify/map.go:26` | `link = "/galgame/" + n.AnchorID + "?comment=" + strconv.FormatInt(*n.PostID, 10)` | `AnchorID` is community `site_game` id = forum gid; `PostID` is community post id | pass-through on the path segment; `comment=` untouched; **external** community |
| `apps/api/internal/community/anchor/anchor.go:74` | `out[ref] = Target{Link: "/galgame/" + ref.ID, Label: "Galgame", GalgameID: gid}` | community `AnchorSiteGame` id string = forum gid | pass-through; **external** community |
| `apps/api/internal/galgame/service/community_comment_write.go:75` | `content, "/galgame/"+strconv.Itoa(galgameID), false, post.CreatedAt)` | forum gid written into `feed_activity.link` | pass-through (builder); stored rows are data-rewrite |
| `apps/api/internal/galgame/repository/galgame_merge_repo.go:188` | `newGID, "/galgame/"+strconv.Itoa(oldGID), "/galgame/"+strconv.Itoa(newGID), oldGID)` | fold rewrites `feed_activity.link` from old gid to survivor gid | delete (merge 301 ledger / live link chase goes away per decision 4); fold data-move still needed |
| `apps/api/internal/wall/apiv1/subject.go:47` | `maxLength: 5000, feedType: "GALGAME_COMMENT_CREATION", linkPrefix: "/galgame/",` | prefix for wall feed / notice links of a galgame wall | pass-through |
| `apps/api/internal/wall/apiv1/effects.go:23` | `return sub.spec.linkPrefix + strconv.Itoa(sub.id)` | `sub.id` is wall `subject_id`; for `subject_type=galgame` that is the forum gid | pass-through |
| `apps/api/internal/activity/service/activity_service.go:905` | `return fmt.Sprintf("galgame#%d", b.ID)` | fallback display name when catalog brief has no title; `b.ID` is the forum gid the brief is keyed by | pass-through |

### 4.2 TS / Vue `/galgame/${…}` builders

Taxonomy / other-id rows first, then every builder whose interpolated number is a forum gid.

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/web/shared/utils/kunTaxonomyPaths.ts:5` | `` `/galgame/${family}` `` | taxonomy family slug (`tag`/`official`/`engine`), not a gid | pass-through |
| `apps/web/shared/utils/kunTaxonomyPaths.ts:8` | `` `/galgame/${family}/${catalogId}` `` | catalog entity id (tag/official/engine), not a forum gid | pass-through |
| `apps/web/tests/taxonomyRedirects.spec.ts:43` | `` `/galgame/${family}` `` | taxonomy family slug | pass-through |
| `apps/web/tests/taxonomyRedirects.spec.ts:51` | `` `/galgame/${family}/1234` `` | catalog entity id `1234` | pass-through |
| `apps/web/tests/taxonomyRedirects.spec.ts:122` | `` expect(to).toBe(`/galgame/${family}/6935`) `` | catalog entity id `6935` | pass-through |
| `apps/web/server/utils/kunSitemapSources.ts:214` | `loc: (r) => `/galgame/resource/${num(r, 'id')}`,` | `galgame_resource.id` | pass-through |
| `apps/web/server/utils/kunSitemapSources.ts:238` | `loc: (r) => `/galgame/official/${num(r, 'id')}`,` | catalog company/official id | pass-through |
| `apps/web/server/utils/kunSitemapSources.ts:246` | `loc: (r) => `/galgame/tag/${num(r, 'id')}`,` | catalog tag id | pass-through |
| `apps/web/server/utils/kunSitemapSources.ts:257` | `(r) => `/galgame/engine/${num(r, 'id')}`,` | catalog engine id | pass-through |
| `apps/web/server/utils/kunSitemapSources.ts:199` | `loc: (r) => `/galgame/${num(r, 'id')}`,` | sitemap loc; `id` is forum `galgame.id` from `GET /galgame?indexed=true` | pass-through after data-rewrite of `galgame.id` |
| `apps/web/server/utils/kunOgCard.ts:104` | `const game = await fetchKunApi<GalgameDetail>(`/galgame/${id}`, {` | OG builder fetches legacy detail by the number in `/og/galgame/:id` | pass-through |
| `apps/web/server/routes/rss/galgame.xml.ts:20` | `link: `${baseUrl}/galgame/${g.id}`,` | RSS item link; `g.id` is forum `galgame.id` | pass-through after data-rewrite |
| `apps/web/app/constants/trust.ts:49` | `return `/galgame/${id}`` | trust subject `kind=galgame`; `id` is forum gid (also sent to trust) | pass-through; **external** trust |
| `apps/web/app/composables/useReportResourceExpired.ts:31` | `` `/galgame/${galgameId}/resource/expired`, `` | API path; `galgameId` is forum gid | pass-through |
| `apps/web/app/pages/galgame/[gid]/index.vue:14` | `const { data } = await useKunFetch<GalgameDetail>(`/galgame/${gid.value}`, {` | page fetch; `gid` is the URL param | pass-through |
| `apps/web/app/pages/galgame/[gid]/index.vue:17` | `query: { galgame_id: gid.value }` | same forum gid repeated as query `galgame_id` | pass-through (rename field) |
| `apps/web/app/pages/galgame/[gid]/index.vue:21` | `await navigateTo(`/galgame/${data.value.moved_to}`, {` | `moved_to` is the survivor forum gid from the local 301 ledger | **delete** (decision 3+4: old numbers and catalog merges are not redirected) |
| `apps/web/app/pages/galgame/[gid]/index.vue:168` | `ogCard: { kind: 'galgame', id: galgame.id },` | builds `/og/galgame/{forum gid}` | pass-through |
| `apps/web/app/pages/galgame-rating/[id].vue:34` | `const gameUrl = `${kungal.domain.main}/galgame/${rating.galgame.id}`` | JSON-LD / absolute URL; `rating.galgame.id` is forum gid | pass-through |
| `apps/web/app/components/galgame/Header.vue:52` | `` `/admin/galgame/${props.galgame.id}/resource-publish-ban`, `` | admin API; `galgame.id` is forum gid | pass-through |
| `apps/web/app/components/galgame/Header.vue:358` | `@click="navigateTo(`/galgame/${galgame.id}/edit`)"` | page URL; `galgame.id` is forum gid | pass-through |
| `apps/web/app/components/galgame/Header.vue:386` | `:subject-url="`${kungal.domain.main}/galgame/${galgame.id}`"` | report snapshot URL; forum gid | pass-through |
| `apps/web/app/components/galgame/Like.vue:36` | `const result = await kunFetch(`/galgame/${props.galgameId}/like`, {` | API; `galgameId` is forum gid | pass-through |
| `apps/web/app/components/galgame/Info.vue:123` | `@click="navigateTo(`/galgame/${galgame.id}/history`)"` | page URL | pass-through |
| `apps/web/app/components/galgame/Covers.vue:63` | `` }>(`/galgame/${props.gid}/cover/${cover.id}/vote`, { `` | API; `props.gid` is forum gid; `cover.id` is cover id | pass-through |
| `apps/web/app/components/galgame/PreviewModal.vue:19` | `detail.value = await kunFetch<GalgameDetail>(`/galgame/${props.gid}`)` | API; `props.gid` is forum gid | pass-through |
| `apps/web/app/components/galgame/card/Card.vue:103` | `href: galgame.id > 0 ? `/galgame/${galgame.id}` : undefined,` | card href; `galgame.id` is forum gid | pass-through |
| `apps/web/app/components/galgame/link/Link.vue:10` | `` `/galgame/${gid.value}/link/all`, `` | API; route param `gid` | pass-through |
| `apps/web/app/components/galgame/resource/Resource.vue:27` | `` `/galgame/${gid.value}/resource/all`, `` | API; route param `gid` | pass-through |
| `apps/web/app/components/galgame/resource/Like.vue:32` | `const result = await kunFetch(`/galgame/${props.galgameId}/resource/like`, {` | API; forum gid | pass-through |
| `apps/web/app/components/galgame/resource/Link.vue:102` | `kunFetch(`/galgame/${props.resource.galgame_id}/resource/valid`, {` | API; `resource.galgame_id` is forum gid | pass-through |
| `apps/web/app/components/galgame/resource/LinkEditModal.vue:182` | `kunFetch(`/galgame/${props.galgameId}/resource`, { method, body })` | API; forum gid | pass-through |
| `apps/web/app/components/galgame/resource/LinkDetailModal.vue:120` | `kunFetch(`/galgame/${props.resource.galgame_id}/resource`, {` | API; forum gid | pass-through |
| `apps/web/app/components/galgame/resource/LinkDetailModal.vue:298` | `<KunLink size="sm" :to="`/galgame/${resource.galgame_id}`">` | page URL | pass-through |
| `apps/web/app/components/galgame/resource/BuyLegitNotice.vue:94` | `<KunLink size="sm" :to="`/galgame/${galgameId}`" class-name="inline">` | page URL | pass-through |
| `apps/web/app/components/galgame/resource/detail/Info.vue:34` | `` `/galgame/${props.resource.galgame_id}/resource`, `` | API | pass-through |
| `apps/web/app/components/galgame/resource/detail/Info.vue:43` | `await navigateTo(`/galgame/${props.resource.galgame_id}`)` | page URL | pass-through |
| `apps/web/app/components/galgame/resource/detail/Info.vue:189` | `<KunLink size="sm" :to="`/galgame/${resource.galgame_id}`">` | page URL | pass-through |
| `apps/web/app/components/galgame/resource/detail/Info.vue:230` | `<KunButton variant="flat" :href="`/galgame/${resource.galgame_id}`">` | page URL | pass-through |
| `apps/web/app/components/galgame/resource/detail/Hero.vue:64` | `:to="`/galgame/${props.galgame.id}`"` | page URL; resource-page summary `id` is forum gid | pass-through |
| `apps/web/app/components/galgame/resource/detail/Hero.vue:127` | `<KunButton :href="`/galgame/${galgame.id}`">` | page URL | pass-through |
| `apps/web/app/components/galgame/header/PlaytimeModal.vue:64` | `` `/galgame/${props.galgame.id}/playtime`, `` | API; then the handler maps gid→catalog work id | pass-through of the path; mapping call is **delete** |
| `apps/web/app/components/galgame/history/Container.vue:14` | `` `/galgame/${gid.value}/edit/revisions`, `` | API | pass-through |
| `apps/web/app/components/galgame/history/Container.vue:26` | `` `/galgame/${gid.value}/edit/diff`, `` | API | pass-through |
| `apps/web/app/components/galgame/history/Container.vue:50` | `` `/galgame/${gid.value}/edit/revert`, `` | API | pass-through |
| `apps/web/app/components/galgame/history/Container.vue:79` | `@click="navigateTo(`/galgame/${gid}`)"` | page URL | pass-through |
| `apps/web/app/components/galgame/history/Container.vue:88` | `@click="navigateTo(`/galgame/${gid}/edit`)"` | page URL | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:17` | `` `/galgame/${gid.value}/edit/bootstrap`, `` | API | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:22` | `` `/galgame/${gid.value}`, `` | API detail | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:59` | `` `/galgame/${gid.value}/edit/proposals`, `` | API | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:117` | `` `/galgame/${gid.value}/edit/proposals`, `` | API | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:187` | `@click="navigateTo(`/galgame/${gid}`)"` | page URL | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:196` | `@click="navigateTo(`/galgame/${gid}/history`)"` | page URL | pass-through |
| `apps/web/app/components/galgame/collection/PickerModal.vue:28` | `` `/galgame/${props.galgameId}/collections/mine` `` | API; forum gid sent as path, then holdings use it as catalog `work_id` | pass-through of the path; the skip-mapping send is **external** catalog |
| `apps/web/app/components/galgame/collection/PickerModal.vue:72` | `` `/galgame/${props.galgameId}/collections`, `` | API membership write; same skip-mapping | pass-through of the path; **external** catalog |
| `apps/web/app/components/galgame/quiz/Play.vue:198` | `:to="`/galgame/${g.id}`"` | quiz-linked galgame; `g.id` is forum gid | pass-through |
| `apps/web/app/components/galgame/quiz/DetailPanel.vue:94` | `<KunLink :to="`/galgame/${g.id}`" class="font-medium">` | quiz-linked galgame | pass-through |
| `apps/web/app/components/galgame/rating/detail/Galgame.vue:29` | `<KunLink :to="`/galgame/${galgame.id}`" underline="none">` | rating page game chip | pass-through |
| `apps/web/app/components/galgame/rating/detail/Detail.vue:66` | `await navigateTo(`/galgame/${props.data.galgame.id}`)` | rating page → game page | pass-through |
| `apps/web/app/components/galgame-edit/review/Detail.vue:28` | `: `/galgame/${proposal.value?.gid ?? ''}/edit`` | proposal `gid` field is forum gid | pass-through (rename field) |
| `apps/web/app/components/galgame-edit/review/Detail.vue:75` | `const detail = await kunFetch<GalgameDetail>(`/galgame/${gid}`, {` | API | pass-through |
| `apps/web/app/components/galgame-edit/review/Detail.vue:294` | `<KunLink :to="`/galgame/${proposal.gid}`" size="sm">` | page URL | pass-through |
| `apps/web/app/components/galgame-edit/review/Container.vue:62` | `:to="`/galgame/${(proposal as GalgameEditProposalItem).gid}`"` | page URL | pass-through |
| `apps/web/app/components/galgame-edit/mine/Container.vue:69` | `:to="`/galgame/${item.gid}`"` | page URL | pass-through |
| `apps/web/app/components/edit/galgame/Wizard.vue:44` | `const gameHref = (hit: SearchHit): string => `/galgame/${hit.id \|\| hit.work_id}`` | `hit.id` is forum gid; `hit.work_id` is catalog work id — mixed id spaces in one URL | **delete** the `\|\| hit.work_id` fallback (it is the mapping); remaining `hit.id` is pass-through |
| `apps/web/app/components/edit/galgame/Wizard.vue:145` | `:to="`/galgame/${galgameClaimGid(item)}/edit`"` | `galgameClaimGid` reads `product_work_id` (forum gid), never `work_id` | **delete** the helper (mapping); link becomes `work_id` pass-through |
| `apps/web/app/components/edit/galgame/Mine.vue:17` | `const res = await kunFetch<string>(`/galgame/${gid}`, {` | withdraw; `gid = galgameClaimGid(item)` | **delete** helper; path pass-through |
| `apps/web/app/components/edit/galgame/Mine.vue:45` | `const res = await kunFetch<unknown>(`/galgame/${gid}/resubmit`, {` | resubmit | **delete** helper; path pass-through |
| `apps/web/app/components/edit/galgame/Mine.vue:67` | `const res = await kunFetch<string>(`/galgame/${gid}/draft`, {` | delete draft | **delete** helper; path pass-through |
| `apps/web/app/components/edit/galgame/Mine.vue:130` | `<KunLink :to="`/galgame/${galgameClaimGid(item)}/edit`">` | page URL | **delete** helper; path pass-through |
| `apps/web/app/components/edit/galgame/Footer.vue:130` | `await navigateTo(isLive ? `/galgame/${created.gid}` : '/edit/galgame/mine')` | submit result `gid` (today equal to minted catalog id on new rows) | pass-through (rename field) |
| `apps/web/app/components/edit/galgame/Audited.vue:62` | `:to="`/galgame/${galgameClaimGid(item)}`"` | `product_work_id` as forum gid | **delete** helper; path pass-through |
| `apps/web/app/components/user/Resource.vue:75` | `kunFetch(`/galgame/${res.galgame_id}/resource`, {` | API | pass-through |
| `apps/web/app/components/user/Resource.vue:79` | `kunFetch(`/galgame/${res.galgame_id}/resource/valid`, {` | API | pass-through |
| `apps/web/app/components/user/Resource.vue:108` | `:to="`/galgame/${res.galgame_id}?tab=resource`"` | page URL | pass-through |
| `apps/web/app/components/user/Resource.vue:169` | `:href="`/galgame/${res.galgame_id}?tab=resource`"` | page URL | pass-through |
| `apps/web/app/components/user/Playtime.vue:60` | `:href="`/galgame/${item.galgame.id}`"` | page URL | pass-through |
| `apps/web/app/components/user/Overview.vue:58` | `href: `/galgame/${r.galgame_id}`` | page URL | pass-through |
| `apps/web/app/components/user/Galgame.vue:147` | `:href="c.galgame_id ? `/galgame/${c.galgame_id}?comment=${c.id}` : ''"` | path segment is forum gid; `comment` is community post id | pass-through on the path; never rewrite `comment=` |
| `apps/web/app/components/ranking/Galgame.vue:20` | `:to="`/galgame/${galgame.id}`"` | page URL | pass-through |
| `apps/web/app/components/search/Palette.vue:122` | `value: `/galgame/${galgame.id}`,` | command-palette target | pass-through |
| `apps/web/app/components/activity/card/Galgame.vue:9` | `gid.value ? `/galgame/${gid.value}` : props.activity.link` | `data.galgame_id` from feed | pass-through of builder; stored `activity.link` is data-rewrite |
| `apps/web/app/components/activity/card/GalgameComment.vue:9` | `gid.value ? `/galgame/${gid.value}` : props.activity.link` | same | pass-through of builder; stored link is data-rewrite |
| `apps/web/app/components/activity/card/GalgameInfo.vue:9` | `gid.value ? `/galgame/${gid.value}` : props.activity.link` | same | pass-through of builder; stored link is data-rewrite |
| `apps/web/app/components/activity/card/GalgameRating.vue:20` | `gid.value ? `/galgame/${gid.value}` : props.activity.link` | same | pass-through of builder; stored link is data-rewrite |
| `apps/web/app/components/activity/card/GalgameResource.vue:17` | `gid.value ? `/galgame/${gid.value}` : props.activity.link` | same | pass-through of builder; stored link is data-rewrite |
| `apps/web/app/components/activity/card/GalgameEdit.vue:25` | `` `/galgame/${gid}/edit/revisions?limit=200` `` | API; `gid` from `data.galgame_id` | pass-through |
| `apps/web/app/components/activity/card/GalgameEdit.vue:31` | `` `/galgame/${gid}/edit/diff?from=${seq - 1}&to=${seq}` `` | API; `seq` is revision seq, not a gid | pass-through |
| `apps/web/app/utils/communityComment.ts:41` | `subjectId: String(target.galgameId),` | v1 wall `subject_id` for `subject_type=galgame`; forum gid as decimal string | pass-through; **external** community via wall |
| `apps/web/app/utils/communityComment.ts:46` | `wallAnchor: { anchor_kind: SITE_GAME, anchor_id: String(target.galgameId) }` | community `site_game:<gid>` | pass-through; **external** community |
| `apps/web/app/components/galgame/comment/CommunityContainer.vue:49` | `found.data.subject_id !== String(gid)` | compares v1 wall `subject_id` to the URL gid | pass-through |

### 4.3 Cache keys (Redis and in-memory)

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/client/client.go:104` | `gidCache map[int]gidLookupEntry` | in-memory gid → catalog work id memo | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:90` | `if e, ok := c.gidCache[gid]; ok && now.Before(e.expire) {` | lookup by forum gid | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:189` | `c.gidCache[gid] = gidLookupEntry{catalogID: id, found: ok, expire: now.Add(ttl)}` | stores catalog id under forum gid | **delete** |
| `apps/api/internal/galgame/client/catalog_face.go:694` | `delete(c.gidCache, gid)` | `ForgetGIDs` after merge sync | **delete** |
| `apps/api/internal/galgame/client/client.go:26` | `type batchCacheKey struct {` / `id int` | in-memory brief/detail/label/tag caches keyed by forum gid | pass-through (key becomes work id); process restart after cutover |
| `apps/api/internal/galgame/apiv1/moyu.go:21` | `moyuCacheKey = "moyu:patches:v1:"` | Redis prefix | pass-through |
| `apps/api/internal/galgame/apiv1/moyu.go:130` | `key := moyuCacheKey + strconv.FormatInt(catalogID, 10)` | suffix is catalog work id after `CatalogWorkIDForGID` | pass-through of the key (already catalog id); the translation call is **delete** |
| `apps/api/internal/galgame/service/collection_items.go:21` | `return "kungal:folder-items:v1:" + strconv.FormatInt(folder.ID, 10) + ":" + folder.UpdatedAt` | Redis key is catalog folder id, not a gid | pass-through |
| `apps/api/internal/galgame/service/collection_items.go:16` | `WorkID    int64  `json:"w`` | cached payload `w` is catalog `work_id` as the forum wrote it (often the raw gid) | **data-rewrite** of Redis values; **external** catalog |
| `apps/api/internal/galgame/client/catalog_v2_cache.go:14` | `v2CacheKeyPrefix   = "nmcache:v2:"` | Redis prefix for catalog HTTP | **external** catalog |
| `apps/api/internal/galgame/client/catalog_v2_cache.go:29` | `return v2CacheKeyPrefix + v2CacheIdentity(v2Path, q)` | identity includes `/v2/catalog/works/{catalog id}` and `refs=curated:{gid}` | **delete** the `refs=curated:{gid}` queries; numeric path ids are catalog work ids (pass-through / **external** catalog) |
| `apps/api/internal/galgame/service/galgame_claim_event_sync.go:57` | `claimSubmitterKeyPrefix = "catalog:claim:submitter:"` | Redis; suffix is catalog `WorkID` | **external** catalog; pass-through |
| `apps/api/internal/home/service/home_service.go:43` | `cacheKey := fmt.Sprintf("home:v1:%t", isSFW)` | no gid in the key; cached payload embeds galgame cards with forum ids | pass-through of key; payload ids follow data-rewrite (flush cache at cutover) |
| `apps/api/internal/activity/service/activity_service.go:63` | `cacheKey := fmt.Sprintf("activity:v2:%s:%s:%d:%t:%t", typeStr, cursor, limit, isSFW, showNoResource)` | no gid in the key; cached payload embeds `galgame_id` / `link` | pass-through of key; payload follows data-rewrite (flush cache at cutover) |

### 4.4 Cursors

Positive control: `cursor` under `apps/` (~57 hits). Sitemap `next_cursor` is catalog/API pagination. Redis cron keys `catalog:redirects:merge:cursor`, `catalog:claim:cron:since`, `catalog:changes:content-limit:cursor`, `catalog:rev:cron:since`, `catalog:contrib:cron:since` are catalog-feed offsets. Activity cache keys include an activity keyset cursor. Community `next_cursor` is a post cursor. v2 `TestOffsetCursorRoundTrip` is catalog offset. **No cursor encodes a forum gid.**

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/service/galgame_merge_sync.go:24` | `MergeCursorKey = "catalog:redirects:merge:cursor"` | catalog `/v2/catalog/redirects` feed cursor | **external** catalog; **decide** whether this cron still exists after decision 4 (no 301s) |
| `apps/api/internal/galgame/service/galgame_claim_event_sync.go:55` | `claimCursorKey = "catalog:claim:cron:since"` | catalog claim-event feed id | **external** catalog |
| `apps/api/internal/galgame/service/galgame_catalog_mirror.go:23` | `mirrorCursorKey = "catalog:changes:content-limit:cursor"` | catalog changes feed cursor | **external** catalog |
| `apps/api/internal/galgame/service/galgame_edit_revision_sync.go:36` | `editRevisionCursorKey  = "catalog:rev:cron:since"` | catalog revision feed id | **external** catalog |
| `apps/api/internal/galgame/service/galgame_contributor_sync.go:33` | `contributorCursorKey     = "catalog:contrib:cron:since"` | catalog contributor feed id | **external** catalog |

### 4.5 Sitemap / RSS / OG / JSON-LD

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/web/server/utils/kunSitemapSources.ts:199` | `loc: (r) => `/galgame/${num(r, 'id')}`,` | indexed forum `galgame.id` | pass-through after data-rewrite; regenerate sitemap after cutover |
| `apps/web/server/routes/rss/galgame.xml.ts:20` | `link: `${baseUrl}/galgame/${g.id}`,` | RSS; `g.id` from `GET /rss/galgame` | pass-through after data-rewrite |
| `apps/api/internal/rss/handler/rss_handler.go:70` | `ID:     row.ID,` | RSS JSON `id` is forum `galgame.id` (hydrated via `GetBatchPublic` keyed by gid) | pass-through; hydration mapping is **delete** |
| `apps/web/shared/utils/ogCard.ts:22` | `` `/og/${kind}/${id}` `` | when `kind='galgame'`, `id` is forum gid | pass-through |
| `apps/web/server/routes/og/[kind]/[id].ts:10` | `const id = Number(getRouterParam(event, 'id'))` | `/og/galgame/:id` | pass-through |
| `apps/web/server/utils/kunOgCard.ts:104` | `const game = await fetchKunApi<GalgameDetail>(`/galgame/${id}`, {` | OG fetch by forum gid | pass-through |
| `apps/web/server/utils/kunOgCard.ts:105` | `galgame_id: id` | same gid as query | pass-through (rename field) |
| `apps/web/app/pages/galgame/[gid]/index.vue:45` | `const pageUrl = `${kungal.domain.main}${route.path}`` | JSON-LD `url` / canonical; `route.path` is `/galgame/{gid}` | pass-through (URL shape stays; number changes with the page id) |
| `apps/web/app/pages/galgame/[gid]/index.vue:73` | `url: pageUrl,` | schema.org VideoGame url | pass-through |
| `apps/web/app/pages/galgame-rating/[id].vue:34` | `const gameUrl = `${kungal.domain.main}/galgame/${rating.galgame.id}`` | rating JSON-LD itemReviewed url | pass-through |

### 4.6 Notification text

Notification *body* is a content preview (no gid). The gid lives in `message.link` / `feed_activity.link`. Builders in §4.1; stored columns are data-rewrite (other census). Tests that pin the link string:

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/community/notify/map_test.go:54` | `typ: "commented", link: "/galgame/4178?comment=100",` | fixture forum gid `4178` | pass-through of the path shape; fixture number is a gid |
| `apps/api/internal/community/anchor/anchor_test.go:21` | `refs[0]: "/galgame/1207",` | fixture forum gid `1207` | pass-through of the path shape |
| `apps/api/internal/message/repository/community_mirror_test.go:43` | `Content: "hello", Link: "/galgame/1", Status: "unread", Type: "followed",` | fixture forum gid `1` | pass-through of the path shape |
| `apps/api/internal/activity/service/activity_service.go:905` | `return fmt.Sprintf("galgame#%d", b.ID)` | human-visible fallback in the activity card title | pass-through |

---

## 7. Web

Positive control: listed `apps/web/app/pages/galgame/` (route files), searched `params as { gid`, `` `/galgame/${` `` (78 hits), `galgameClaimGid` / `work_ids` / `product_work_id` / `catalog_id` / `moved_to`, `gid:` in `apps/web/shared/types`, Nitro `apps/web/server/**`. Link-builder rows that interpolate a forum gid are repeated from §4.2 so this section stands alone.

### 7.1 Routes

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/web/app/pages/galgame/[gid]/index.vue` (filename) | directory `[gid]` | Nuxt page param name for `/galgame/:id` | pass-through (rename param to `id` / `work_id` in code; URL stays `/galgame/:id`) |
| `apps/web/app/pages/galgame/[gid]/index.vue:11` | `return parseInt((route.params as { gid: string }).gid)` | URL path segment is forum gid | pass-through (rename) |
| `apps/web/app/pages/galgame/[gid]/edit.vue:7` | `<GalgameEditContainer />` | child reads `route.params.gid` | pass-through |
| `apps/web/app/pages/galgame/[gid]/history.vue:2` | `<GalgameHistoryContainer />` | child reads `route.params.gid` | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:12` | `const gid = computed(() => parseInt((route.params as { gid: string }).gid))` | same URL param | pass-through |
| `apps/web/app/components/galgame/history/Container.vue:9` | `const gid = computed(() => parseInt((route.params as { gid: string }).gid))` | same URL param | pass-through |
| `apps/web/app/components/galgame/resource/Resource.vue:10` | `return parseInt((route.params as { gid: string }).gid)` | same URL param | pass-through |
| `apps/web/app/components/galgame/link/Link.vue:7` | `const gid = computed(() => parseInt((route.params as { gid: string }).gid))` | same URL param | pass-through |
| `apps/web/app/components/galgame/quiz/GalgamePanel.vue:3` | `const gid = computed(() => parseInt((route.params as { gid: string }).gid))` | same URL param | pass-through |
| `apps/web/app/components/galgame/comment/CommunityContainer.vue:10` | `const gid = parseInt((route.params as { gid: string }).gid)` | same URL param, compared to v1 `subject_id` | pass-through |
| `apps/web/nuxt.config.ts:101` | `// Without this the retired path falls through to /galgame/[gid] and` | documents that `/galgame/library` would be parsed as a gid | pass-through (comment rename) |
| `apps/web/app/pages/galgame/resource/[id]/index.vue:9` | `const resourceId = computed(() => Number((route.params as { id: string }).id))` | resource id, not a gid | pass-through |
| `apps/web/app/pages/galgame/collection/[id].vue:3` | `const collectionId = computed(() => Number((route.params as { id: string }).id))` | collection/folder id | pass-through |
| `apps/web/app/pages/galgame/tag/[id].vue:6` | `return Number((route.params as { id: string }).id)` | catalog tag id | pass-through |
| `apps/web/app/pages/galgame/official/[id]/index.vue` | official `[id]` | catalog company id | pass-through |
| `apps/web/app/pages/galgame/engine/[id].vue:4` | `return Number((route.params as { id: string }).id)` | catalog engine id | pass-through |
| `apps/web/app/pages/galgame/series/[id].vue:4` | `return Number((route.params as { id: string }).id)` | catalog series id | pass-through |
| `apps/web/app/pages/galgame/staff/[id].vue:5` | `const staffId = computed(() => Number((route.params as { id: string }).id))` | catalog person id | pass-through |
| `apps/web/app/pages/galgame/character/[id].vue:8` | `const characterId = computed(() => Number((route.params as { id: string }).id))` | catalog character id | pass-through |
| `apps/web/app/pages/galgame-quiz/[id].vue` (filename) | `[id]` | quiz id | pass-through |
| `apps/web/app/pages/galgame-rating/[id].vue` (filename) | `[id]` | rating id | pass-through |

### 7.2 Link builders (forum gid in the path)

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/web/app/pages/galgame/[gid]/index.vue:14` | `const { data } = await useKunFetch<GalgameDetail>(`/galgame/${gid.value}`, {` | URL gid | pass-through |
| `apps/web/app/pages/galgame/[gid]/index.vue:21` | `await navigateTo(`/galgame/${data.value.moved_to}`, {` | local merge 301 target | **delete** |
| `apps/web/app/pages/galgame-rating/[id].vue:34` | `const gameUrl = `${kungal.domain.main}/galgame/${rating.galgame.id}`` | forum gid | pass-through |
| `apps/web/app/constants/trust.ts:49` | `return `/galgame/${id}`` | forum gid | pass-through; **external** trust |
| `apps/web/app/composables/useReportResourceExpired.ts:31` | `` `/galgame/${galgameId}/resource/expired` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/Header.vue:52` | `` `/admin/galgame/${props.galgame.id}/resource-publish-ban` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/Header.vue:358` | `navigateTo(`/galgame/${galgame.id}/edit`)` | forum gid | pass-through |
| `apps/web/app/components/galgame/Header.vue:386` | `:subject-url="`${kungal.domain.main}/galgame/${galgame.id}`"` | forum gid | pass-through |
| `apps/web/app/components/galgame/Like.vue:36` | `` `/galgame/${props.galgameId}/like` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/Info.vue:123` | `navigateTo(`/galgame/${galgame.id}/history`)` | forum gid | pass-through |
| `apps/web/app/components/galgame/Covers.vue:63` | `` `/galgame/${props.gid}/cover/${cover.id}/vote` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/PreviewModal.vue:19` | `` `/galgame/${props.gid}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/card/Card.vue:103` | `href: galgame.id > 0 ? `/galgame/${galgame.id}` : undefined` | forum gid | pass-through |
| `apps/web/app/components/galgame/link/Link.vue:10` | `` `/galgame/${gid.value}/link/all` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/Resource.vue:27` | `` `/galgame/${gid.value}/resource/all` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/Like.vue:32` | `` `/galgame/${props.galgameId}/resource/like` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/Link.vue:102` | `` `/galgame/${props.resource.galgame_id}/resource/valid` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/LinkEditModal.vue:182` | `` `/galgame/${props.galgameId}/resource` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/LinkDetailModal.vue:120` | `` `/galgame/${props.resource.galgame_id}/resource` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/LinkDetailModal.vue:298` | `` `/galgame/${resource.galgame_id}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/BuyLegitNotice.vue:94` | `` `/galgame/${galgameId}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/detail/Info.vue:34` | `` `/galgame/${props.resource.galgame_id}/resource` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/detail/Info.vue:43` | `navigateTo(`/galgame/${props.resource.galgame_id}`)` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/detail/Info.vue:189` | `` `/galgame/${resource.galgame_id}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/detail/Info.vue:230` | `` `/galgame/${resource.galgame_id}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/detail/Hero.vue:64` | `` `/galgame/${props.galgame.id}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/resource/detail/Hero.vue:127` | `` `/galgame/${galgame.id}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/header/PlaytimeModal.vue:64` | `` `/galgame/${props.galgame.id}/playtime` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/history/Container.vue:14` | `` `/galgame/${gid.value}/edit/revisions` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/history/Container.vue:26` | `` `/galgame/${gid.value}/edit/diff` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/history/Container.vue:50` | `` `/galgame/${gid.value}/edit/revert` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/history/Container.vue:79` | `navigateTo(`/galgame/${gid}`)` | forum gid | pass-through |
| `apps/web/app/components/galgame/history/Container.vue:88` | `navigateTo(`/galgame/${gid}/edit`)` | forum gid | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:17` | `` `/galgame/${gid.value}/edit/bootstrap` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:22` | `` `/galgame/${gid.value}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:59` | `` `/galgame/${gid.value}/edit/proposals` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:117` | `` `/galgame/${gid.value}/edit/proposals` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:187` | `navigateTo(`/galgame/${gid}`)` | forum gid | pass-through |
| `apps/web/app/components/galgame/edit/Container.vue:196` | `navigateTo(`/galgame/${gid}/history`)` | forum gid | pass-through |
| `apps/web/app/components/galgame/collection/PickerModal.vue:28` | `` `/galgame/${props.galgameId}/collections/mine` `` | forum gid (later sent to catalog as work id) | pass-through of path; **external** catalog |
| `apps/web/app/components/galgame/collection/PickerModal.vue:72` | `` `/galgame/${props.galgameId}/collections` `` | forum gid | pass-through of path; **external** catalog |
| `apps/web/app/components/galgame/quiz/Play.vue:198` | `` `/galgame/${g.id}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/quiz/DetailPanel.vue:94` | `` `/galgame/${g.id}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/rating/detail/Galgame.vue:29` | `` `/galgame/${galgame.id}` `` | forum gid | pass-through |
| `apps/web/app/components/galgame/rating/detail/Detail.vue:66` | `navigateTo(`/galgame/${props.data.galgame.id}`)` | forum gid | pass-through |
| `apps/web/app/components/galgame-edit/review/Detail.vue:28` | `` `/galgame/${proposal.value?.gid ?? ''}/edit` `` | proposal `gid` | pass-through (rename field) |
| `apps/web/app/components/galgame-edit/review/Detail.vue:75` | `` `/galgame/${gid}` `` | proposal `gid` | pass-through |
| `apps/web/app/components/galgame-edit/review/Detail.vue:294` | `` `/galgame/${proposal.gid}` `` | proposal `gid` | pass-through |
| `apps/web/app/components/galgame-edit/review/Container.vue:62` | `` `/galgame/${(proposal as GalgameEditProposalItem).gid}` `` | proposal `gid` | pass-through |
| `apps/web/app/components/galgame-edit/mine/Container.vue:69` | `` `/galgame/${item.gid}` `` | proposal `gid` | pass-through |
| `apps/web/app/components/edit/galgame/Wizard.vue:44` | `` `/galgame/${hit.id \|\| hit.work_id}` `` | forum gid or catalog work id | **delete** the `work_id` fallback |
| `apps/web/app/components/edit/galgame/Wizard.vue:145` | `` `/galgame/${galgameClaimGid(item)}/edit` `` | `product_work_id` | **delete** helper |
| `apps/web/app/components/edit/galgame/Mine.vue:17` | `` `/galgame/${gid}` `` | `galgameClaimGid` | **delete** helper |
| `apps/web/app/components/edit/galgame/Mine.vue:45` | `` `/galgame/${gid}/resubmit` `` | `galgameClaimGid` | **delete** helper |
| `apps/web/app/components/edit/galgame/Mine.vue:67` | `` `/galgame/${gid}/draft` `` | `galgameClaimGid` | **delete** helper |
| `apps/web/app/components/edit/galgame/Mine.vue:130` | `` `/galgame/${galgameClaimGid(item)}/edit` `` | `product_work_id` | **delete** helper |
| `apps/web/app/components/edit/galgame/Footer.vue:130` | `` `/galgame/${created.gid}` `` | submit result `gid` | pass-through (rename field) |
| `apps/web/app/components/edit/galgame/Audited.vue:62` | `` `/galgame/${galgameClaimGid(item)}` `` | `product_work_id` | **delete** helper |
| `apps/web/app/components/user/Resource.vue:75` | `` `/galgame/${res.galgame_id}/resource` `` | forum gid | pass-through |
| `apps/web/app/components/user/Resource.vue:79` | `` `/galgame/${res.galgame_id}/resource/valid` `` | forum gid | pass-through |
| `apps/web/app/components/user/Resource.vue:108` | `` `/galgame/${res.galgame_id}?tab=resource` `` | forum gid | pass-through |
| `apps/web/app/components/user/Resource.vue:169` | `` `/galgame/${res.galgame_id}?tab=resource` `` | forum gid | pass-through |
| `apps/web/app/components/user/Playtime.vue:60` | `` `/galgame/${item.galgame.id}` `` | forum gid | pass-through |
| `apps/web/app/components/user/Overview.vue:58` | `` `/galgame/${r.galgame_id}` `` | forum gid | pass-through |
| `apps/web/app/components/user/Galgame.vue:147` | `` `/galgame/${c.galgame_id}?comment=${c.id}` `` | forum gid; `comment` is community post id | pass-through on path; never rewrite `comment=` |
| `apps/web/app/components/ranking/Galgame.vue:20` | `` `/galgame/${galgame.id}` `` | forum gid | pass-through |
| `apps/web/app/components/search/Palette.vue:122` | `` `/galgame/${galgame.id}` `` | forum gid | pass-through |
| `apps/web/app/components/activity/card/Galgame.vue:9` | `` `/galgame/${gid.value}` `` | `activity.data.galgame_id` | pass-through |
| `apps/web/app/components/activity/card/GalgameComment.vue:9` | `` `/galgame/${gid.value}` `` | `activity.data.galgame_id` | pass-through |
| `apps/web/app/components/activity/card/GalgameInfo.vue:9` | `` `/galgame/${gid.value}` `` | `activity.data.galgame_id` | pass-through |
| `apps/web/app/components/activity/card/GalgameRating.vue:20` | `` `/galgame/${gid.value}` `` | `activity.data.galgame_id` | pass-through |
| `apps/web/app/components/activity/card/GalgameResource.vue:17` | `` `/galgame/${gid.value}` `` | `activity.data.galgame_id` | pass-through |
| `apps/web/app/components/activity/card/GalgameEdit.vue:25` | `` `/galgame/${gid}/edit/revisions?limit=200` `` | `activity.data.galgame_id` | pass-through |
| `apps/web/app/components/activity/card/GalgameEdit.vue:31` | `` `/galgame/${gid}/edit/diff?from=${seq - 1}&to=${seq}` `` | `activity.data.galgame_id` | pass-through |
| `apps/web/app/pages/admin/submissions.vue:128` | `kunFetch<unknown>(`/admin/galgame/${gid}/review`, {` | pending-claim `gid` | pass-through (rename field) |

### 7.3 Web compares or converts galgame id against catalog id

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/web/shared/utils/galgameClaimState.ts:42` | `// work_id is catalog's id, product_work_id is the forum's galgame id. They are` | documents two id spaces | **delete** |
| `apps/web/shared/utils/galgameClaimState.ts:46` | `item.product_work_id ?? 0` | forum gid from claim; never `work_id` | **delete** (after G0 `work_id` is the page id) |
| `apps/web/app/components/edit/galgame/Wizard.vue:3` | `id: number` | wizard hit `id` is forum gid | pass-through (rename) |
| `apps/web/app/components/edit/galgame/Wizard.vue:4` | `work_id?: number` | catalog work id on the same hit | **delete** as a distinct field (becomes the same number) |
| `apps/web/app/components/edit/galgame/Wizard.vue:44` | `` `/galgame/${hit.id \|\| hit.work_id}` `` | mixes the two spaces | **delete** the fallback |
| `apps/web/app/components/edit/galgame/Wizard.vue:123` | `:key="`pending-${item.work_id}`"` | Vue key is catalog work id | pass-through (it is already work id) |
| `apps/web/app/components/edit/galgame/Wizard.vue:129` | `{{ item.display_name \|\| `#${item.work_id}` }}` | display fallback uses catalog work id | pass-through |
| `apps/web/app/components/edit/galgame/Wizard.vue:156` | `:key="`item-${hit.work_id ?? hit.id}`"` | prefers catalog work id, falls back to gid | **delete** the mixed key |
| `apps/web/app/components/edit/galgame/Mine.vue:112` | `:key="item.work_id"` | catalog work id as list key | pass-through |
| `apps/web/app/components/edit/galgame/Audited.vue:46` | `:key="item.work_id"` | catalog work id as list key | pass-through |
| `apps/web/app/composables/useMyGalgameInteractions.ts:18` | `for (const gid of ids) {` | local liked/favorited sets hold forum gids | pass-through |
| `apps/web/app/composables/useMyGalgameInteractions.ts:40` | `const query = chunk.length ? { work_ids: chunk.join(',') } : undefined` | forum gids sent to `GET /galgame/interactions/mine?work_ids=` which forwards them to catalog as work ids | **delete** the skip-mapping; after G0 the same numbers are catalog work ids (**external** catalog) |
| `apps/web/app/components/galgame/card/Card.vue:130` | `:key="card.galgame.catalog_id ?? card.galgame.id"` | `catalog_id` is catalog work id; `id` is forum gid | **delete** `catalog_id` once they are equal |
| `apps/web/shared/types/galgame.ts:186` | `catalog_id?: number` | optional catalog work id on a card | **delete** |
| `apps/web/shared/types/galgame.ts:224` | `work_id: number` | `UserClaimItem.work_id` is catalog work id | pass-through (becomes the page id) |
| `apps/web/shared/types/galgame.ts:227` | `product_work_id: number \| null` | forum gid on a claim | **delete** (infra also rewrites `product_work_id` to `catalog_work.id`) |
| `apps/web/app/pages/galgame/[gid]/index.vue:20` | `if (data.value?.moved_to) {` | local 301 ledger | **delete** |
| `apps/web/app/utils/communityComment.ts:41` | `subjectId: String(target.galgameId),` | forum gid as v1 wall `subject_id` | pass-through |
| `apps/web/app/utils/communityComment.ts:46` | `anchor_id: String(target.galgameId)` | forum gid as community `site_game` | pass-through; **external** community |
| `apps/web/app/components/galgame/comment/CommunityContainer.vue:49` | `found.data.subject_id !== String(gid)` | v1 `subject_id` vs URL gid | pass-through |
| `apps/web/app/pages/admin/submissions.vue:10` | `gid: number` | pending-claim JSON `gid` | pass-through (rename) |
| `apps/web/app/pages/admin/submissions.vue:61` | `const gidOf = (row: PendingClaim) => row.gid` | same | pass-through (rename) |

### 7.4 Nitro (`apps/web/server/**`)

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/web/server/utils/kunSitemapSources.ts:199` | `loc: (r) => `/galgame/${num(r, 'id')}`,` | forum `galgame.id` | pass-through after data-rewrite |
| `apps/web/server/routes/rss/galgame.xml.ts:20` | `link: `${baseUrl}/galgame/${g.id}`,` | forum `galgame.id` | pass-through after data-rewrite |
| `apps/web/server/utils/kunOgCard.ts:104` | `fetchKunApi<GalgameDetail>(`/galgame/${id}`, {` | `/og/galgame/:id` | pass-through |
| `apps/web/server/utils/kunOgCard.ts:105` | `galgame_id: id` | same gid as query | pass-through (rename) |
| `apps/web/server/utils/kunOgCard.ts:109` | `game.moved_to \|\|` | suppress OG for a redirected gid | **delete** with `moved_to` |
| `apps/web/server/routes/og/[kind]/[id].ts:10` | `const id = Number(getRouterParam(event, 'id'))` | when kind is `galgame`, forum gid | pass-through |
| `apps/web/shared/utils/ogCard.ts:3` | `'galgame',` | OG kind token | pass-through |
| `apps/web/server/middleware/legacy-taxonomy.ts` | (taxonomy redirects; ids are catalog entity ids) | not a forum gid | pass-through |

### 7.5 Shared types with gid / galgame_id fields

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/web/shared/types/galgame.ts:115` | `id: number` | `GalgameDetail.id` is forum gid | pass-through (rename in code if the field is called gid anywhere; JSON `id` stays the page id) |
| `apps/web/shared/types/galgame.ts:116` | `moved_to?: number` | survivor forum gid | **delete** |
| `apps/web/shared/types/galgame.ts:176` | `id: number` | `GalgameCard.id` is forum gid | pass-through |
| `apps/web/shared/types/galgame.ts:186` | `catalog_id?: number` | catalog work id beside the gid | **delete** |
| `apps/web/shared/types/galgame.ts:224` | `work_id: number` | claim catalog work id | pass-through |
| `apps/web/shared/types/galgame.ts:227` | `product_work_id: number \| null` | claim forum gid | **delete** |
| `apps/web/shared/types/galgame-edit.ts:112` | `gid: number` | `GalgameEditProposalItem.gid` | pass-through (rename to `work_id`) |
| `apps/web/shared/types/galgame-edit.ts:117` | `gid: number` | `GalgameEditBootstrap.gid` | pass-through (rename) |
| `apps/web/shared/types/galgame-edit.ts:144` | `gid: number` | `GalgameEditRevisionList.gid` | pass-through (rename) |
| `apps/web/shared/types/galgame-resource.ts:4` | `galgame_id: number` | resource parent forum gid | pass-through (rename to `work_id`) |
| `apps/web/shared/types/galgame-link.ts:4` | `galgame_id: number` | link parent forum gid | pass-through (rename) |
| `apps/web/shared/types/galgame-quiz.ts:161` | `galgame_ids: number[]` | quiz attached forum gids | pass-through (rename) |
| `apps/web/shared/types/galgame-rating.ts:45` | `id: number` | nested `galgame.id` on a rating card | pass-through |
| `apps/web/shared/types/galgame-community-comment.ts:15` | `galgame_id?: number` | follow-item forum gid | pass-through (rename) |
| `apps/web/shared/types/activity.ts:76` | `galgame_id?: number` | feed payload forum gid | pass-through (rename) |
| `apps/web/shared/types/user.ts:46` | `galgame_id: number` | `UserGalgameResource.galgame_id` | pass-through (rename) |
| `apps/web/shared/types/search.ts:47` | `galgame_id?: number` | search gal-comment hit | pass-through (rename) |
| `apps/web/shared/types/api/v1.d.ts:79` | `"/galgames/{galgame_id}/moyu-patches": {` | v1 path param | pass-through (rename to `work_id`) |
| `apps/web/shared/types/api/v1.d.ts:2906` | `galgame_id: string;` | v1 path param type | pass-through (rename) |
| `apps/web/shared/types/api/v1.d.ts:2351` | `subject_id: string;` | wall comment; when `subject_type=galgame` this is the forum gid | pass-through |
| `apps/web/shared/types/api/v1.d.ts:2366` | `subject_id: string;` | `WallCommentCreate.subject_id` | pass-through |
| `apps/web/app/pages/admin/submissions.vue:10` | `gid: number` | local `PendingClaim` | pass-through (rename) |

---

## 8. Existing v1 surface

Positive control: searched `package apiv1` (galgame, wall, topic, shared `internal/apiv1`), `Path:` under those packages, `galgame_id` / `json:"gid"` / `{galgame_id}` / `CatalogWorkIDForGID`. Topic v1 `category=galgame` is a topic category enum, not a galgame id (examined, not flagged as a gid). Live galgame-id-bearing v1 faces: moyu-patches and wall-comments `subject_id` when `subject_type=galgame`. G8 forbids a JSON field named `gid`.

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/apiv1/register.go:17` | `Path:        "/galgames/{galgame_id}/moyu-patches",` | v1 path param is forum gid | pass-through (rename to `work_id`) |
| `apps/api/internal/galgame/apiv1/moyu.go:82` | `GalgameID string `path:"galgame_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Galgame id."`` | path param parsed as forum gid | pass-through (rename) |
| `apps/api/internal/galgame/apiv1/moyu.go:93` | `gid, ok := repr.ParseID(repr.DecimalID(in.GalgameID))` | parsed forum gid | pass-through |
| `apps/api/internal/galgame/apiv1/moyu.go:97` | `catalogID, found, appErr := s.works.CatalogWorkIDForGID(ctx, gid)` | mapping layer gid → catalog work id | **delete** |
| `apps/api/internal/galgame/apiv1/moyu.go:59` | `ID        repr.DecimalID      `json:"id" doc:"The page's id on www.moyu.moe. Neither a galgame id nor a catalog work id."`` | moyu page id, not a forum gid | pass-through |
| `apps/api/internal/galgame/apiv1/moyu.go:130` | `key := moyuCacheKey + strconv.FormatInt(catalogID, 10)` | Redis suffix is catalog work id | pass-through of key; mapping call is **delete** |
| `apps/api/pkg/moyuclient/client.go:90` | `"refs":    {"catalog:" + strconv.FormatInt(catalogWorkID, 10)},` | catalog work id sent to moyu `/v2/moyu` | **external** moyu |
| `apps/api/internal/app/v1_galgame_moyu_test.go:27` | `moyuSpecPath    = "/galgames/{galgame_id}/moyu-patches"` | pins the path name | pass-through (rename in test) |
| `apps/api/internal/app/v1_galgame_moyu_test.go:22` | `moyuGID         = 4121` | fixture forum gid | pass-through of fixture after data-rewrite of what 4121 means |
| `apps/api/internal/app/v1_galgame_moyu_test.go:23` | `moyuCatalogID   = 61311` | fixture catalog work id, not equal to 4121 | **delete** the split; after G0 one number |
| `apps/api/internal/app/v1_galgame_moyu_test.go:47` | `func (f fakeWorkIDs) CatalogWorkIDForGID(_ context.Context, gid int) (int64, bool, *legacyErrors.AppError) {` | fake mapping 4121 → 61311 | **delete** |
| `apps/api/internal/app/v1_galgame_moyu_test.go:112` | `req := httptest.NewRequest(http.MethodGet, "/api/v1/galgames/"+galgameID+"/moyu-patches", nil)` | request uses forum gid | pass-through (rename path) |
| `apps/api/internal/app/v1_galgame_moyu_test.go:144` | `if q, _ := f.gotQuery.Load().(string); q != "include=resources%2Cpublisher&nsfw=true&refs=catalog%3A61311" {` | upstream asked with catalog id 61311 | **delete** the translated-id assertion (or rewrite as identity) |
| `apps/web/shared/types/api/v1.d.ts:79` | `"/galgames/{galgame_id}/moyu-patches": {` | generated path | pass-through (rename; file is generated) |
| `apps/web/shared/types/api/v1.d.ts:2906` | `galgame_id: string;` | generated path param | pass-through (rename) |
| `apps/api/internal/wall/apiv1/register.go:43` | `Path:        "/wall-comments",` | list; query `subject_id` | pass-through |
| `apps/api/internal/wall/apiv1/types.go:13` | `SubjectID           repr.DecimalID          `json:"subject_id" doc:"Id of that page: a galgame, rating, resource, quiz, toolset or website id, by subject_type."`` | when `subject_type=galgame`, forum gid | pass-through |
| `apps/api/internal/wall/apiv1/types.go:43` | `SubjectID       repr.DecimalID  `json:"subject_id" doc:"Id of that page."`` | create body; galgame wall → forum gid | pass-through |
| `apps/api/internal/wall/apiv1/types.go:60` | `SubjectID   string      `query:"subject_id" required:"true" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Id of that page."`` | list query; galgame wall → forum gid | pass-through |
| `apps/api/internal/wall/apiv1/subject.go:44` | `"galgame": {` | wall kind whose id is a forum gid | pass-through |
| `apps/api/internal/wall/apiv1/subject.go:45` | `typ: "galgame", anchorKind: communityclient.AnchorSiteGame,` | community anchor kind 1 (`site_game`) | pass-through; **external** community |
| `apps/api/internal/wall/apiv1/service.go:92` | `case "galgame":` | resolve galgame wall | pass-through |
| `apps/api/internal/wall/apiv1/service.go:93` | `found, gerr := s.galgame(ctx, id)` | existence check | pass-through of the id |
| `apps/api/internal/app/wall_v1.go:29` | `_, found, appErr := galgame.CatalogWorkIDForGID(ctx, galgameID)` | wall existence = mapping hit | **delete** (existence becomes a local `galgame` row / identity fetch) |
| `apps/api/internal/wall/apiv1/effects.go:23` | `return sub.spec.linkPrefix + strconv.Itoa(sub.id)` | feed/notice link `/galgame/{gid}` | pass-through |
| `apps/api/internal/wall/apiv1/effects.go:64` | `galgameID = sub.id` | feed `galgame_id` column | pass-through; stored rows data-rewrite |
| `apps/web/shared/types/api/v1.d.ts:2351` | `subject_id: string;` | wall comment field | pass-through |
| `apps/web/shared/types/api/v1.d.ts:2366` | `subject_id: string;` | create body | pass-through |
| `apps/api/internal/apiv1/gates/repr.go:31` | `"kind", "gid", "tid", "uid", "rid", "pid", "cid",` | G8 forbids JSON field name `gid` on v1 | pass-through (keep the forbid; the replacement name is `work_id`) |
| `apps/api/internal/apiv1/gates/gates_test.go:179` | `Gid repr.DecimalID `json:"gid" doc:"Galgame."`` | pins that `gid` is a forbidden v1 field name | pass-through (test stays) |
| `apps/api/internal/apiv1/gates/repr.go:70` | `case name == "id" \|\| strings.HasSuffix(name, "_id") \|\| name == "cursor" \|\| name == "next_cursor":` | G7: `galgame_id` / `work_id` must be JSON strings | pass-through |

---

## 9. Tests that pin the mapping

Positive control: searched `gid` in `*_test.go` (172 hits examined), `CatalogWorkIDForGID` / `gid()` / `site_work_id` / `the gid, never` / `catalog id, not the gid` / `work_ids` / `curated` / `product_work_id` in tests. Tests that only use `:gid` as a Fiber path param name (cover vote routing, playtime routing, edit handler mount) are not mapping pins and are omitted here. Mapping pins below.

| file:line | quoted line | what the number means there | G0 action |
|---|---|---|---|
| `apps/api/internal/galgame/client/catalog_rename_test.go:25` | `if got := it.gid(); got != tc.want {` | `gid()` from claim `site_work_id` vs catalog `ID` | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:30` | `if got := (&CatalogWorkListItem{ID: 14}).gid(); got != 14 {` | unclaimed row: gid = catalog id | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:42` | `func TestClaimCarriesTheSiteIDNotTheCatalogID(t *testing.T) {` | catalog id 4242 vs forum gid 777 | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:53` | `t.Errorf("site_work_id = %d, want the forum gid 777", it.Claim.SiteWorkID)` | claim site_work_id is the gid | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:55` | `if got := it.gid(); got != 777 {` | `gid()` must not return catalog 4242 | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:112` | `ids, appErr := c.catalogIDsForGIDs(t.Context(), []int{8})` | curated ref `8` → catalog 9002 | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:117` | `t.Errorf("gid 8 resolved to %d, want 9002 via the post-rename source key", ids[8])` | mapping via `curated` | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:85` | `t.Errorf("gid 7 resolved to %d, want 9001 via the pre-rename source key", ids[7])` | mapping via `galgame_wiki` | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:165` | `ids, appErr := c.catalogIDsForGIDs(t.Context(), []int{90210})` | adopted id with no anchor | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:169` | `if ids[90210] != 90210 {` | round-trip: claim.site_work_id == asked id | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:177` | `42: `{"id":42,"claim":{"site":"kungal","site_work_id":"7","state":"live"}}`,` | coincidence: catalog 42 claims gid 7 | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:184` | `t.Errorf("gid 42 resolved to work %d — the round-trip check must reject a "+` | must not treat catalog 42 as gid 42 | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:191` | `500: `{"id":500,"claim":{"site":"moyu","site_work_id":"500","state":"live"}}`,` | foreign claim resolved by catalog id | **delete** |
| `apps/api/internal/galgame/client/catalog_rename_test.go:202` | `func TestIdentityRouteResolvesAnUnclaimedWorkByCatalogID(t *testing.T) {` | unclaimed work id = gid | **delete** |
| `apps/api/internal/galgame/client/catalog_face_test.go:122` | `func liveRow(catalogID int64, gid int, name string) string {` | fixture splits catalog id and gid | **delete** |
| `apps/api/internal/galgame/client/catalog_face_test.go:125` | `` `"claim":{"site":"kungal","site_work_id":` + itoa(int64(gid)) + `,"state":"live"},` `` | site_work_id is the gid | **delete** |
| `apps/api/internal/galgame/client/catalog_face_test.go:158` | `func TestCatalogBridge_TwoHopAndGIDKeying(t *testing.T) {` | two-hop refs then ids | **delete** |
| `apps/api/internal/galgame/client/catalog_face_test.go:181` | `t.Errorf("works ids = %q, want 4242 (the catalog id, not the gid)", ids)` | second hop uses catalog id | **delete** |
| `apps/api/internal/galgame/client/catalog_face_test.go:192` | `t.Errorf("brief.ID = %d, want 777 (the gid, never the catalog id)", b.ID)` | result keyed by gid | **delete** |
| `apps/api/internal/galgame/client/catalog_face_test.go:195` | `t.Error("result is keyed by the catalog id — the two id spaces overlap, so this attaches another game's local stats")` | overlap hazard | **delete** |
| `apps/api/internal/galgame/client/catalog_face_test.go:238` | `func TestCatalogBridge_UnresolvedGIDIsAbsentNotAnError(t *testing.T) {` | miss is empty, not error | **delete** |
| `apps/api/internal/galgame/client/catalog_face_test.go:359` | `t.Errorf("made %d lookup calls, want 1 (the gid→catalog id memo is what keeps the second hop cheap)", lookups)` | gidCache | **delete** |
| `apps/api/internal/galgame/client/catalog_face_test.go:663` | `t.Errorf("own work = %+v, want gid 777 with no via — a company's own game must not read as borrowed", members[0])` | member list projects gid | **delete** (or rewrite as work_id identity) |
| `apps/api/internal/galgame/client/catalog_changes_test.go:77` | `func TestMirrorByCatalogIDsKeysByTheClaimNeverTheCatalogID(t *testing.T) {` | mirror keyed by gid 8471 not catalog 923 | **delete** |
| `apps/api/internal/galgame/client/catalog_changes_test.go:112` | `t.Errorf("catalog id %d leaked in as a gid: %v", catalogID, got)` | overlap | **delete** |
| `apps/api/internal/galgame/client/catalog_stale_gid_test.go:8` | `func TestStaleGIDsForCatalogIDs_NamesNonCanonicalAnchorRefs(t *testing.T) {` | extra `curated`/`galgame_wiki` refs are stale gids | **delete** |
| `apps/api/internal/galgame/client/catalog_stale_gid_test.go:30` | `if len(stale) != 2 \|\| stale[0] != 5904 \|\| stale[1] != 7001 {` | stale gids 5904, 7001 vs canonical 61295 | **delete** |
| `apps/api/internal/galgame/client/catalog_v2_test.go:303` | `func TestLiveV2GIDBridge(t *testing.T) {` | live `CatalogWorkIDs` / `CatalogRowsByGIDs` | **delete** |
| `apps/api/internal/galgame/client/catalog_v2_test.go:315` | `ids, appErr := c.CatalogWorkIDs(ctx, gids)` | gid → catalog id | **delete** |
| `apps/api/internal/galgame/client/catalog_detail_test.go:14` | `func detailStub(t *testing.T, gid int, catalogID int64, body string)` | stub splits gid and catalog id | **delete** |
| `apps/api/internal/galgame/client/catalog_detail_test.go:41` | `const gid, catalogID = 777, 4242` | 777 ≠ 4242 | **delete** |
| `apps/api/internal/app/v1_galgame_moyu_test.go:47` | `func (f fakeWorkIDs) CatalogWorkIDForGID(...)` | fake mapping | **delete** |
| `apps/api/internal/app/v1_galgame_moyu_test.go:51` | `if gid == moyuGID { return moyuCatalogID, true, nil }` | 4121 → 61311 | **delete** |
| `apps/api/internal/app/v1_galgame_moyu_test.go:144` | `refs=catalog%3A61311` | translated id on the wire | **delete** |
| `apps/api/internal/galgame/service/galgame_interactions_test.go:21` | `if !strings.Contains(r.URL.RawQuery, "work_ids=") {` | forum ids 42,99 sent as catalog work_ids | **delete** as a mapping skip; rewrite as identity send |
| `apps/api/internal/galgame/service/galgame_interactions_test.go:41` | `got := s.GetMyInteractions(context.Background(), 0, "user-jwt", []int{42, 99})` | those ints are forum gids | pass-through of the call after G0 (they are work ids) |
| `apps/api/internal/galgame/service/wizard_search_test.go:138` | `t.Errorf("ids = %d,%d,%d, want the gids 292,9978,5150",` | wizard `ID` is forum gid | **delete** the gid-vs-work split; rewrite as work_id |
| `apps/api/internal/galgame/service/wizard_search_test.go:141` | `if page.Items[2].ID != 14 \|\| page.Items[2].WorkID != 14 {` | unclaimed: id == work_id == catalog id | **delete** as a special case |
| `apps/api/internal/galgame/service/submission_submit_test.go:144` | `if res.GID != 90210 {` | submit returns gid | pass-through (rename field); today equal to work id on mint |
| `apps/api/internal/galgame/service/submission_submit_test.go:148` | `if res.WorkID != 90210 \|\| res.ClaimState != "pending" {` | also returns catalog work id | **delete** the extra field once they are one number |
| `apps/api/internal/galgame/service/submission_submit_test.go:352` | `workID, appErr := svc.workIDOf(t.Context(), res.GID)` | resolve freshly minted gid | **delete** |
| `apps/api/internal/galgame/service/submission_submit_test.go:357` | `t.Errorf("gid %d resolved to work %d, want %d", res.GID, workID, res.WorkID)` | mapping round-trip | **delete** |
| `apps/api/internal/galgame/service/claim_unclaimed_test.go:71` | `if !strings.Contains(rec.bodies[0], `"site_work_id":"4649"`) && !strings.Contains(rec.bodies[0], `"product_work_id":4649`) {` | claim body names the forum gid | **delete** as a distinct gid; after infra rewrite site_work_id == work id |
| `apps/api/internal/galgame/service/series_sfw_test.go:39` | `"id": gid + 1000, "display_name": "作品",` | fixture catalog id = gid+1000 | **delete** |
| `apps/api/internal/galgame/service/series_sfw_test.go:40` | `"claim": map[string]any{"site": "kungal", "site_work_id": gid, "state": "live"},` | site_work_id is the gid | **delete** |
| `apps/api/internal/galgame/service/galgame_merge_sync_test.go:133` | `t.Run("wiki-era old catalog id is not a local gid", func(t *testing.T) {` | catalog 5904 vs local gid 61295 | **delete** (merge 301 ledger goes away; fold tests that move data may remain rewritten) |
| `apps/api/internal/galgame/service/galgame_merge_sync_test.go:145` | `t.Run("stale gid is the redirect old id and still resolves", func(t *testing.T) {` | stale gid 5904 | **delete** |
| `apps/api/internal/galgame/service/galgame_merge_sync_test.go:158` | `func TestFold_CoincidenceGidDoesNotFold(t *testing.T) {` | catalog id 5000 is another work | **delete** |
| `apps/api/internal/galgame/service/galgame_merge_sync_test.go:174` | `t.Errorf("folded=%d deferred=%d, want 0/0 — 5000 is another work's catalog id", folded, deferred)` | overlap | **delete** |
| `apps/api/internal/galgame/service/galgame_contributor_sync_test.go:13` | `Site: "kungal", ProductWorkID: gid, CreatedAt: at,` | feed `product_work_id` treated as forum gid | **delete** |
| `apps/api/internal/galgame/service/galgame_contributor_sync_test.go:27` | `if len(gids) != 1 \|\| gids[0] != 4321 {` | refresh list is gids | **delete** (or rewrite as work_ids) |
| `apps/api/internal/galgame/service/galgame_edit_revision_sync_test.go:118` | `t.Errorf("entity_id = %d, want the gid 1207", item.EntityID)` | catalog revision `entity_id` called a gid | **delete** |
| `apps/api/internal/galgame/handler/cover_vote_handler_test.go:171` | `t.Fatalf("an unresolvable gid must not reach the vote face: %+v", fake.requests)` | vote face is reached only after mapping | **delete** the mapping gate |
| `apps/api/internal/galgame/service/cover_votes_bearer_test.go:28` | `` `{"id":"1000","claim":{"site":"kungal","site_work_id":"1","state":"live"},"refs":[{"source":"curated","external_id":"1"}]}` `` | gid 1 → catalog 1000 via curated | **delete** |
| `apps/api/internal/galgame/handler/edit_handler_test.go:46` | `` `{"id":"1000","claim":{"site":"kungal","site_work_id":"1","state":"live"},"refs":[{"source":"curated","external_id":"1"}]}` `` | same mapping stub | **delete** |
| `apps/web/shared/utils/galgameClaimState.spec.ts:10` | `({ work_id: 4649, product_work_id }) as UserClaimItem` | two id spaces on one claim | **delete** |
| `apps/web/shared/utils/galgameClaimState.spec.ts:57` | `it('never falls back to the catalog work_id for a forum link', () => {` | `galgameClaimGid` must ignore `work_id` | **delete** |
| `apps/web/shared/utils/galgameClaimState.spec.ts:58` | `expect(galgameClaimGid(claim(1207))).toBe(1207)` | product_work_id is the gid | **delete** |
| `apps/api/internal/apiv1/gates/gates_test.go:179` | `Gid repr.DecimalID `json:"gid" doc:"Galgame."`` | v1 must not expose a field named `gid` | pass-through (keep) |

---
