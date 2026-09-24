# API v1 changelog

## 2026-09-24 (G6 cover votes, playtime, collections)

Breaking for `PUT` / `DELETE /api/galgame/:id/cover/:coverId/vote`, `PUT /api/galgame/:id/playtime`, `GET /api/galgame/playtime/mine`, `POST /api/galgame/collection`, `GET` / `PATCH` / `DELETE /api/galgame/collection/:cid`, `PUT /api/galgame/:id/collections`, `GET /api/galgame/:id/collections/mine` and `GET /api/user/:id/collections`; all eleven are gone. No App build calls them; `docs/proj/app-direct-api.md` names the replacements.

**A collection's id is now the catalog folder id.** The old forum id is a different number for 4,079 of 9,338 collections, so the two must never be swapped. The page moves from `/galgame/collection/{cid}` to `/collection/{collection_id}`. An old link still works: a full page load answers `301`, and an in-site link goes through a redirect page. Both resolve the old id with `GET /api/v1/collection-aliases/{alias_id}`. No stored content links to the old path (0 occurrences on prod), so there is no link-rewrite migration.

Offered:

- `GET /api/v1/works/{work_id}/covers/{cover_id}` → `WorkCover`.
- `PUT` / `DELETE /api/v1/works/{work_id}/covers/{cover_id}/vote` → `WorkCoverEngagement` (`vote_count`, `viewer.has_voted`). Both are idempotent.
- `PUT /api/v1/works/{work_id}/playtime` `{minutes?, play_state?}` → `WorkViewerPlaytime`. An omitted field is left unchanged; `play_state: null` clears the state. `done` is display-only and refused.
- `DELETE /api/v1/works/{work_id}/playtime` removes the forum's own report and clears the work state. The response is read back from catalog, so another app's minutes can remain in it.
- `GET /api/v1/me/playtimes`: page-number; `total_minutes`, `finished_work_count` and `is_truncated` on the envelope; `include_nsfw`.
- `POST /api/v1/collections`: `Idempotency-Key` required; 201 + `Location`; `visibility` must be sent.
- `GET` / `PATCH` / `DELETE /api/v1/collections/{collection_id}`.
- `GET /api/v1/collections/{collection_id}/works` → `PageList<WorkSummary>`.
- `GET` / `PUT` / `DELETE /api/v1/collections/{collection_id}/works/{work_id}`: one folder's membership, as a slot.
- `GET /api/v1/me/collections` (`work_id=` sets `viewer.has_work`) and `GET /api/v1/users/{user_id}/collections`.
- `GET /api/v1/collection-aliases/{alias_id}`: read-only. It is `404` whenever the target collection is not visible to the caller.

Fixed:

- **The picker could remove a work from every folder.** The old write replaced the whole membership set, so a failed read that saved `collection_ids: []` emptied every folder. Membership is now written one folder at a time, and only for the folders the user changed.
- **A read created data.** Listing collections minted an alias row for every folder without one. No GET writes now, and the alias table is frozen.
- **A collection's page showed an empty grid when the work lookup failed.** It is `503` now.
- **A catalog rate limit surfaced as a data failure:** a 429 fell through to a 500 and rendered as 「读取收藏夹列表失败」. Every upstream 429 is `503 SERVICE_UNAVAILABLE` now, with the upstream `Retry-After`. A session missing a scope is `403 SCOPE_REQUIRED` (legacy code 235).
- **Withdrawing a playtime left a 0-minute row.** It now deletes the forum's row only (infra#296 scoped catalog's delete to the calling app).
- **Deleting a folder whose contents could not be read** still went ahead, leaving the local favourite ranking too high. It is refused with `503` now, and nothing is deleted.
- **`/me/playtimes` counted NSFW works in its totals** while hiding them from the list. Totals and items now share one predicate.
- Request-body errors on these operations are `422 VALIDATION_FAILED`, and parameter errors stay `400`.

## 2026-09-24 (G5 galgame browse, library, release calendar, entity search)

Breaking for `GET /api/galgame` (both engines), `GET /api/galgame/calendar{,/today,/pending,/tba,/upcoming}`, `GET /api/galgame/collected-calendar`, `GET /api/rss/galgame`, `GET /api/search/entity` and `GET /api/search/entity/resolve`; all ten are gone. No App build calls them; `docs/proj/app-direct-api.md` names the replacements.

Offered:

- `GET /api/v1/works`: the forum's browse list, a page-number collection of `WorkSummary` (`limit` 1–100, default 24). It shows published works with at least one forum resource; `include_resourceless=true` lists every published work.
  - Filters: `resource_type`, `resource_platforms`, `resource_languages`, `game_type`, `resource_providers`, `excluded_sole_providers`, `released_from` / `released_to` / `released_months`, `collected_from` / `collected_to` / `collected_months`, `min_rating`, `min_rating_count`, `include_nsfw`.
  - `sort` tokens: `resource_updated_*` (default desc), `created_*`, `view_*`, `view_1d_*`, `view_7d_*`, `view_30d_*`, `release_date_*`, `rating_*`.
- `GET /api/v1/library-works`: catalog's browse population, a page-number collection of `WorkSummary`.
  - Params: `q`, `sort` (`popularity_desc` default, `released_desc`, `released_asc`, `updated_desc`, `relevance_desc`), `released_from` / `released_to`, `include_nsfw`.
  - It takes none of the forum filters.
- `GET /api/v1/release-calendar` (`month`), `…/today`, `…/pending` (`year`), `…/tba`, `…/upcoming`: `WorkSummary` items on Asia/Tokyo time.
  - A window catalog could not finish is marked `is_truncated`.
  - `upcoming` is `503` when a month fails.
- `GET /api/v1/works/collected-months`: `{year, month}` pairs over the `/works` population.
- `GET /api/v1/tags` and `GET /api/v1/companies` take `ids` (1–100, comma form, not with `q`): the same page, holding only those ids.
- `GET /api/v1/characters` items are now `CharacterSummary`. This is a strict superset of the old `CharacterRef` item, adding `image` and `catalog_work_count`; only the generated schema name changed, from `PageListCharacterRef` to `PageListCharacterSummary`.

Every v1 operation: an unknown enum token longer than the enum's keys (such as `resource_platforms=windows`) answers `400 UNKNOWN_ENUM_VALUE`. It used to be `400 INVALID_PARAMETER`, because the implied length error was counted too.

Fixed:

- **The galgame RSS showed about three of its ten items.** It took the newest ten works and then dropped the NSFW ones, and seven of those ten were NSFW. `/works` filters NSFW in SQL before the limit. The feed now lists works by their newest resource, dated by that same field. It carries no author or description.
- **The sitemap stopped at 130 pages of 50 (6,500 works).** It now pages until `total`. This was a latent cap, not an observed loss: the sitemap lists SFW works only, and that population is about 4,800. The earlier wording here, "listed 6,500 of 9,821", was wrong (corrected 2026-09-24).
- **Busy calendar months were cut at 100 works**, and `upcoming` left out a month that failed to load.
- **The collected-months strip answered `[]` on a database error.**
- **The library silently ignored the browse page's filters.** It shared the route through a `library` flag and dropped every forum filter. The two collections are now separate.
- **Unknown sort fields fell back to resource-update order,** and unknown resource keys matched nothing and returned an empty page. Both are `400` now.
- **Ascending sorts broke ties on `id DESC`.** The tie-breaker now follows the sort's direction.
- **The entity resolve silently dropped malformed ids and cut the list at 100.**

## 2026-09-24 (G4.1 work detail vocabularies)

`GET /api/v1/works/{work_id}` narrowed catalog's vocabularies and limits and lost data without saying so. Checked against infra's definitions, these are fixed:

- **Roster rows with role `unknown` were dropped.** Catalog's `roster_role` includes `unknown` (5% of roster rows; mostly characters reached only through a voice credit). `roster[].character_kind` now includes `unknown`.
- **`sensitive` age rating was reported as `all_ages`** (3,016 works). `content_rating` is now `all_ages` | `sensitive` | `r18`. It is a label on the age axis only; the SFW gate stays `is_nsfw`.
- **Credit role names were cut at 128 characters**; catalog allows 512 and so does `credits[].display_name` now.
- **Screenshot captions were cut at 512 characters**; `screenshots[].caption` now takes catalog's 2048.
- **A source rank of 0 became `null`**; `external_ratings[].source_rank` now carries what catalog publishes (≥ 0).
- **Lengths were measured in bytes, not characters,** so a 300-character CJK alias (900 bytes) was dropped. Aliases, voiced-character names and external ids are measured in characters.

Also: `roster[].identity` is removed. It is catalog's opaque proposal token, not display text, and nothing read it. Every row still dropped on purpose (a cover or screenshot without a usable URL, a malformed ref, rating or playtime row) is now logged at WARN.

## 2026-09-24 (G4 galgame work detail)

Breaking for `GET /api/galgame/:id`, `PUT /api/galgame/:id/like`, `GET /api/galgame/:id/link/all`, `GET /api/galgame/interactions/mine` and `GET /api/galgame/drafts`; all five are gone. `DELETE /api/galgame/:id` (submission withdraw) stays until G7. No App build calls them; `docs/proj/app-direct-api.md` names the replacements.

Offered:

- `GET /api/v1/works/{work_id}` now answers the full `Work` (G3 answered a `WorkRef`; every `WorkRef` field keeps its name and type). Beyond the `WorkSummary` fields, it carries:
  - `aliases`, `original_language`, `content_rating` (`all_ages` | `r18`), `intros` (Markdown source), `links` (the old `/link/all`), `external_refs`;
  - `covers` (with `vote_count` and `viewer.has_voted`), `screenshots`, `companies` (with `attribution_roles`), `engines`, `series`, `tags`, `credits`, `roster`;
  - `external_ratings` (`rating_value`, `source_rank`, `buckets`, `stats`), `playtimes`, `resource_types`, `favorite_count` (catalog's favourites metric), `is_resource_publish_banned`, `creator`, `contributors`, `dlsite`;
  - `viewer` (`has_liked`, `has_favorited`, `playtime` with `play_state`, `can_ban_resource_publish`).

  It does not carry the ratings list; read `GET /api/v1/ratings?work_id=`. Adult tags are left out unless `include_nsfw=true`; the work itself is not gated (`is_nsfw`). A merged work is `404 ENTITY_MERGED` with `object: "work"` and `current_id`. Every read counts one view.
- `PUT` / `DELETE /api/v1/works/{work_id}/like` (slot, both 200 `{work_id, like_count, viewer: {has_liked}}`). Liking your own page is `403 SELF_LIKE_FORBIDDEN` (was `400`); an unknown or hidden work is `404`.
- `GET /api/v1/me/work-states?work_ids=` (1–100, comma form): `has_liked` / `has_favorited` per readable id, the rest in `missing`. It replaces the dump of every like the caller ever made. If the caller's folders cannot be read, `has_favorited` is `false` and `has_liked` is still answered.

`WorkRef.cover.sexual` (on every face that carries a `WorkRef`) is now the portrait's real grade; it was always `null`.

Fixed:

- **The folder picker could empty every folder.** It re-reads the caller's folders when it opens. If that read failed, it showed an empty list, and Save sent `PUT /galgame/:id/collections {collection_ids: []}`, taking the work out of every folder the caller had put it in. Save now waits for a successful read.
- A like on a page with no creator paid moemoepoint to user 0, and replaying `PUT /like` undid the like (it was a toggle).
- The view counter ran in a goroutine that dropped its error. It now counts synchronously and still feeds the 7-day / 30-day rankings.
- `/link/all` answered an unknown or hidden work with `200 []`; the links are now part of the work, which is `404`.

## 2026-09-24 (G3 galgame resources)

Breaking for every `/api/galgame-resource*` and `/api/galgame/:id/resource*` route, `PUT /api/admin/galgame/:id/resource-publish-ban`, and `GET /api/search` (its last lane, `type=resource`). 12 legacy routes go; no App build calls them.

Offered:

- `GET /api/v1/galgame-resources`: page-number (`limit` 1–100, default 50); `q` searches notes and work names; `include_nsfw`, `state` (`valid` / `expired`) and `sort` (`created_desc` default, `created_asc`). Resources on NSFW works are left out unless `include_nsfw=true`, and resources by unrenderable authors are left out of both `items` and `total`. A catalog failure is `503`, not a silent notes-only search.
- `GET /api/v1/galgame-resources/{resource_id}` (counts a view), `PATCH` (partial; `state: "valid"` marks it working again), `DELETE` (204), and `GET …/source` (every editable field with the links and codes, for editors, no download counted).
- `POST …/downloads`: the only way to get `download_urls`, `extraction_code` and `archive_password`; it counts the download. Anonymous callers may use it.
- `PUT` / `DELETE …/like` (slot) and `POST …/expiry-reports` (`{verdict: alive | dead | unchecked, state}`; an already expired resource is 200, no longer 400).
- `GET /api/v1/works/{work_id}/resources` (page-number) and `POST` (Idempotency-Key required, 201). A banned work is `403 RESOURCE_PUBLISH_BANNED`.
- `PUT` / `DELETE /api/v1/works/{work_id}/resource-publish-ban` (staff, cookie only) and `GET /api/v1/works/{work_id}` (a `WorkRef`; G4 widens it to the full work).

Names: `resource_type`, `resource_languages`, `resource_platforms`, `resource_runtimes` (stored kebab-case keys such as `native-win`), `version_label` tokens (`official_latest` `stable` `mirror` `localized` `unknown`), `state`, `size` (free text), `content` (the note as a document; the Markdown is `content_markdown` on `/source`), `work` (`WorkRef`), `dlsite` (`{purchase_url, coupon_url, campaign_name} | null`). No resource GET carries a link or a code any more; the old `link_domain` (which held the first full URL) is gone. Deleting a resource takes back 3 moemoepoint, the same as creating it gave, and the like counts stay with the author.

## 2026-09-24 (X1d image uploads, 413 everywhere)

Breaking for `POST /api/image/topic`, `/api/image/cover`, `/api/image/message` and `/api/image/galgame`; all four are gone. No App build calls them.

Offered:

- `POST /api/v1/images`: multipart `file` + `purpose` (`content` | `message`). Returns `201` with `Location` and an `Image`. It replaces the topic, cover and message routes. `content` covers everything published on the site; `message` is for private messages. The daily limit stays at 50 per user (Asia/Shanghai), and a failed upload no longer counts against it.
- `POST /api/v1/work-edit-images`: multipart `file` + `preset` (`cover` | `screenshot`). It proxies catalog `/v2/me/edit-images` with the caller's own token and returns `201` with an `Image`. The old preset names `galgame_banner` / `galgame_screenshot` are no longer accepted.

Shape vs the retired routes: every upload returns an `Image` (`url`, `hash`, `width`, `height`, `thumbhash`, `sexual`). The topic and message routes used to return a bare `/image/<hash>` string. The client now builds that string from `hash`, and should keep persisting the token or the hash, never `url`. `sexual` is a string (`safe` | `suggestive` | `explicit`) or `null` for an image not graded yet; it used to be an integer.

New codes: `IMAGE_DAILY_LIMIT_REACHED` (429, with `limit`) and `IMAGE_REJECTED` (422, the image host's moderation), both kungal; and `PAYLOAD_TOO_LARGE` (413), a platform code shared verbatim with infra.

**Every v1 operation:** a request body over the limit used to answer `500 INTERNAL_ERROR`. The limit is 1 MiB for a JSON body and 10 MiB + 64 KiB for anything else. It now answers `413 PAYLOAD_TOO_LARGE`, and every operation with a request body declares 413.

## 2026-09-24 (user name search limit)

- `GET /api/v1/search/users` and `GET /api/v1/users?q=`: `q` is now at most 50 characters (was 107 and 64). The account service refuses longer name queries, and the forum used to pass them on and answer `503`. A longer `q` is now `400 INVALID_PARAMETER` at the edge.
- If the account service refuses a name query for any other reason, the answer is also `400 INVALID_PARAMETER` on `q`, not `503`.

## 2026-09-23 (G2 quizzes)

Breaking for every `/api/galgame-quiz*` route and `GET /api/galgame/search/picker`. 13 legacy routes go; no App build calls them.

Offered:

- `GET /api/v1/quizzes`: page-number (`limit` 1–100, default 50); filters `work_id`, `author_id`, `quiz_type`, `quiz_category`, `difficulty`, `spoiler_level`, `include_nsfw`; `sort` defaults to `bumped_at_desc`. Quizzes linked to an NSFW work are left out unless `include_nsfw=true`; quizzes by unrenderable authors are left out of both `items` and `total`.
- `GET` / `PATCH` / `DELETE /api/v1/quizzes/{quiz_id}`, `POST /api/v1/quizzes` (Idempotency-Key required, 201), and `GET …/source` (the Markdown and the answer key, for editors).
- `GET …/answers` (cursor, newest first) and `POST …/answers` (Idempotency-Key required, 201 `{answer, solution}`; the author may not answer, `403 SELF_ANSWER_FORBIDDEN`; a second answer is `409 ALREADY_EXISTS`).
- `PUT` / `DELETE …/favorite` (slot, both 200) and `PUT …/quality-rating` `{rating}` 1–10 (only after answering: `403 QUIZ_ANSWER_REQUIRED`, which the author also gets).
- `GET /api/v1/me/answered-quizzes` (page-number) and `GET /api/v1/me/quiz-states?quiz_ids=` (the `/me/topic-states` shape: `has_favorited` per readable id, the rest in `missing`).
- `GET /api/v1/work-suggestions?q=`: at most 12 `WorkRef`s from catalog for the quiz work picker.

The answer key (`solution`: `correct_choice_indexes`, `is_statement_true`, `explanation`) and every field derived from it (`is_correct`, other people's `submission`) reach only people who answered, the author and editors. `prompt` is a small document (text, line breaks, inline spoilers); `content` and `explanation` are full documents. Only `single`, `multiple` and `judge` exist. `quiz_type` cannot change, and a changed key regrades every answer both ways. A correct answer earns no moemoepoint, as before, and the author still gets the quiz-answered notice.

## 2026-09-23 (GR galgame ratings)

Breaking for the six `/api/galgame-rating*` faces; they are gone. No App build calls them.

Offered:

- `GET /api/v1/ratings` (page-number, optional identity): `sort` ∈ `created|view|overall` × `desc|asc` (default `created_desc`, ties on `id`), filters `work_id`, `author_id`, `spoiler_level`, `play_status`, `game_type`, and `include_nsfw`. Ratings by banned authors and of works catalog no longer shows are left out of `items` but counted in `total`.
- `GET /api/v1/ratings/{rating_id}` (optional identity): a `Rating` with `work_summary` (the shared `WorkSummary`) and the 50 newest `likers`; every read counts one view.
- `POST /api/v1/ratings` (required, `Idempotency-Key` required): `201` with `Location`. `work_id` catalog does not show is `422` / `UNKNOWN_REFERENCE`; a second rating of the same work is `409 ALREADY_EXISTS`.
- `PATCH /api/v1/ratings/{rating_id}` (author only): changes the fields sent and returns the full `Rating`.
- `DELETE /api/v1/ratings/{rating_id}`: `204`; the author, the work page's creator, or a session holding `rating.delete_any`.
- `PUT` / `DELETE /api/v1/ratings/{rating_id}/like`: a K16 slot returning `rating_engagement`; replaying either changes nothing, liking one's own rating is `403 SELF_LIKE_FORBIDDEN`.

Shape vs the retired faces: ids are strings; `user` → `author` (`UserRef`), `galgame` → `work` (`WorkRef | null`), `galgame_type` → `game_types`, `view` → `view_count`, `created` / `updated` → `created_at` / `updated_at`, `is_liked` / `liked_users` → `viewer.has_liked` / `likers`, and the eight flat aspect fields → `aspect_scores{}` where an unrated aspect is `null`, not `0` (a request sends all eight keys). Every vocabulary is a closed enum the schema checks. `SELF_LIKE_FORBIDDEN` now covers anything the caller wrote.

## 2026-09-23 (U3b user work lists)

Breaking for `GET /api/user/:id/galgames`, `GET /api/user/:id/galgame-comments`, `GET /api/user/:id/resources`, and `GET /api/user/:id/ratings`.

Offered:

- `GET /api/v1/users/{user_id}/works?relation=` — page-number collection (`page`, `limit` 1–100 default 24, `include_nsfw` default false). `relation` is `published` / `contributed` / `liked` (were `galgame_publish` / `galgame_contributed` / `galgame_like`). Items are `WorkSummary`. `show_no_resource` is gone. `contributed`'s `total` counts only the works catalog returns for the requested `include_nsfw`, so pages add up to it.
- `GET /api/v1/users/{user_id}/galgame-resources?relation=` — page-number collection (`limit` default 50). `relation` is `published` / `liked` (were `valid` / `expire` / `galgame_resource_like`); optional `state=valid|expired` is the old valid/expire split. Items are `GalgameResource` and never carry download links, extraction codes, or archive passwords. Resources on unpublished (hidden or banned) works no longer appear, the same rule as the other resource lists; the legacy list did not check.
- `GET /api/v1/users/{user_id}/wall-comments?relation=` — cursor collection (`cursor` in, `next_cursor` out; a page may be shorter than `limit`, even empty, while `next_cursor` is present). `relation` is `authored` / `liked` (were `galgame_comment` / `galgame_comment_like`); the galgame tab sends `subject_type=galgame`. Items are `WallComment` (structured document, not server-rendered HTML).
- Ratings move to the existing `GET /api/v1/ratings?author_id={user_id}&sort=created_desc`.

Retired: `GET /api/user/:id/galgames`, `GET /api/user/:id/galgame-comments`, `GET /api/user/:id/resources`, `GET /api/user/:id/ratings`.

## 2026-09-23 (U3a user topic lists)

Breaking for `GET /api/user/:id/topics`, `GET /api/user/:id/replies`, and `GET /api/user/:id/comments`.

Offered:

- `GET /api/v1/users/{user_id}/topics?relation=` — page-number collection (`page`, `limit` 1–100 default 50, `include_nsfw` default false). `relation` is `authored` / `liked` / `upvoted` / `favorited` / `hidden` (were `topic` / `topic_like` / `topic_upvote` / `topic_favorite` / `topic_hide`). `hidden` is the owner or `topic.view_hidden` only; anyone else gets 403. Items are `{ object, id, title, created_at }`.
- `GET /api/v1/users/{user_id}/replies?relation=` — same envelope. `relation` is `authored` / `received` / `liked` (were `reply_created` / `reply_target` / `reply_like`). Items carry `excerpt` (server-made plain text, at most 200 characters) instead of Markdown `content`.
- `GET /api/v1/users/{user_id}/comments?relation=` — same envelope. `relation` is `authored` / `received` / `liked` (were `comment_created` / `comment_target` / `comment_like`). Items carry `excerpt` instead of `content`.

Envelope is `{ items, total, total_relation }` (was `{ topics|replies|comments, total }`). `id` / `topic_id` are decimal strings; `created` is `created_at`. Hidden and restricted topics, and replies/comments on them, no longer appear on another user's profile. Unknown or banned profile owners answer 404.

Retired: `GET /api/user/:id/topics`, `GET /api/user/:id/replies`, `GET /api/user/:id/comments`.

## G0 window (galgame id is the catalog work id)

## 2026-09-23 (X2 activity stream)

Breaking for `GET /api/activity`, `/api/activity/tab` and `/api/activity/timeline`; all three are gone.

Offered:

- `GET /api/v1/activities`: one cursor collection for the home tabs, `/activity` and `/activity/category` (`limit` 1–100, default 20).
  - Filters: `activity_types` (closed enum, comma form; absent means every type), `topic_sections` (`normal` / `help` / `all`, only affects `topic_creation`), `include_nsfw` and `include_galgames_without_resources`.
  - Sort: `occurred_desc` (default) or `bumped_desc`. `bumped_desc` is allowed only when `activity_types` is exactly `topic_creation`; otherwise it is `400 INVALID_PARAMETER`.
  - The cursor is bound to the sort and every filter, so a changed filter with an old cursor is `400 INVALID_CURSOR` instead of a silent restart.
  - A page can be short, because activities by banned users or on works catalog does not show are dropped. Only an absent `next_cursor` means the end.

Shape vs the retired faces:

- Each item is `activity` with `activity_type`, `performer` (`UserRef` or `null`) and `occurred_at`, plus one type-specific block: `topic` (`TopicSummary`), `topic_digest`, `reply`, `comment`, `work` (`WorkRef`), `work_digest`, `work_stats`, `work_revision` (`revision_number` and `legacy_revision_id`, either may be `null`), `galgame_rating`, `resource`, `quiz`, `toolset`, `todo` or `update_log`.
- `galgame_rating`:
  - `play_status` is the full vocabulary, including `on_hold`.
  - `short_summary` is a non-null string: `""` when there is none, and also when `spoiler_level` is not `none`.
- `resource`:
  - `resource_type` is the resource vocabulary key, as on the `/…/works?resource_type=` filters.
  - `resource_platforms` and `resource_languages` are arrays of vocabulary keys, replacing the single `platform` / `language` strings.
- The upvote notification echo (`MESSAGE_UPVOTE`) is no longer in the stream.

## 2026-09-23 (X2 search)

Breaking for `GET /api/search/overview`, `/api/search/quick` and `/api/search/gal-comment`; all three are gone. The legacy `GET /api/search` now answers only `type=resource`. `/api/search/entity` and `/api/search/entity/resolve` are unchanged until G/GE define those shapes.

Offered, one collection per lane:

- `GET /api/v1/search/topics`, `/replies`, `/comments`: page-number, optional auth, `include_nsfw` defaults to false. NSFW topics, and the replies and comments in them, were returned to anonymous searchers before.
- `GET /api/v1/search/users`: page-number. OAuth caps the search at 50, so `total_relation` is `gte` when the cap is hit (was an exact-looking 50).
- `GET /api/v1/search/works`: page-number, public. It honours `include_nsfw` (the old galgame lane hard-wired its SFW gate off).
- `GET /api/v1/search/wall-comments`: cursor, public, `q` 2–100 characters.

Every lane breaks ties on id, and reply and comment excerpts are plain text instead of Markdown slices. The old overview and quick search each came back as one response. Clients now call each lane themselves, so a lane that fails is its own failed request, not an empty section with a count of 0. The toolset lane is `GET /api/v1/toolsets?q=` (G1).

## 2026-09-23 (X2 rankings)

Breaking for `GET /api/ranking/topic`, `/api/ranking/user`, `/api/ranking/galgame`, and the uncalled `GET /api/home`; all four are gone.

Offered:

- `GET /api/v1/rankings/topics`, `/rankings/users`, `/rankings/works`: top-N lists (`limit` 1–100, default 50) with no pages and no total, each with its own `sort` vocabulary (`views_desc` …).
  - Every sort breaks ties on id.
  - Items are `topic_ranking_entry` / `user_ranking_entry` / `work_ranking_entry`, with `rank` numbered after dropped rows and the sorted `metric_value`.
  - Each entry embeds `topic` (the topic list's `TopicSummary`), `member` (`UserRef`) or `work` (`WorkRef`).
- `include_nsfw` (topics and works) is applied before `LIMIT`, so an SFW top 50 has 50 rows (it had about 25).
- User counts include only what an anonymous visitor can see.
- OAuth or catalog being down is `503`. It used to be 「已注销用户」 authors or an empty list.

## 2026-09-23 (X2 admin overview)

Breaking for `GET /api/admin/overview/all` and `/api/admin/overview/stats`; both are gone.

Offered (both need `admin.dashboard`; a Bearer request is always `403`):

- `GET /api/v1/admin/overview`: nine totals since the site opened, as named `_count` fields.
- `GET /api/v1/admin/overview/daily?days=`: exactly `days` buckets (1–365, default 30), oldest first, one per Beijing calendar day. Days without activity are included, with zeros.

Vs the retired faces:

- `work_count` counts only published works; it included about 6,400 unpublished stub rows.
- The window is `days` buckets, not `days + 1`.
- The series is dense, not sparse.
- Buckets no longer depend on the connection's time zone.
- The server no longer sends Chinese labels.

## 2026-09-23 (X1c news, topic RSS)

Breaking for `GET /api/news`, `/api/news/sources`, `/api/news/archive`, `/api/news/month` and `GET /api/rss/topic`; all five are gone. `GET /api/rss/galgame` stays on the legacy route until the works browse collection lands.

Offered:

- `GET /api/v1/news-items`: cursor list of partner news, newest first (`limit` 1–50, the news service's own cap). Filters: `lane`, `news_source`, `year`, `month` (`month` needs `year`: 400 `INVALID_PARAMETER`, was silently ignored). The cursor is bound to every filter and to `limit`; `total` only with `include_total=true`.
- `GET /api/v1/news-sources`: the whole partner directory, not paged.
- `GET /api/v1/news-archive`: `years`, plus `months` for the one `year` asked about. `months` lists only months with items (was all twelve, empty ones at 0).
- `GET /api/v1/news-archive/{year}/{month}`: `item_count` and `days` (every day of the month, empty days included).
- `GET /api/v1/news-archive/{year}/{month}/items`: the month's items as a page-number collection, with an optional `day`.

Field names vs the retired faces: an item's `source_key` is `news_source`, and there is no page-level `sources` map; resolve the key against `/news-sources`. A source's `name` is `display_name` and `publisher` is `forum_account` (`UserRef` or `null`). `count` is `total`. `banner_url` is not sent. The topic RSS feed at `/rss/topic.xml` keeps its URL; its summaries are now plain text instead of raw Markdown.

## 2026-09-23 (G1.1 toolset resources)

Breaking for toolset resources, a few hours after G1 went live:

- Renamed: a resource's `resource_type` (`file` / `link`) is now `toolset_resource_type`, both on the resource objects and in the `POST …/resources` body. `resource_type` is kept for the galgame resource vocabulary that the `/…/works?resource_type=` filters already use.
- `POST …/resources/{resource_id}/downloads`: `download_url` is nullable. `null` means the resource has no link or file on record; the download is not counted, and the extraction code or note may still carry a link. A file resource without a file answered 503 before; it now answers 200 with `null`.
- `GET …/resources/{resource_id}/source` leaves out `link_url` when a link resource has none on record, instead of sending an empty string.

Migration 144 moves the one link that was pasted into an extraction code into the link field.

## 2026-09-23 (G1 toolsets)

Breaking for every `/api/toolset*` route and `GET /api/user/:id/toolsets`. 16 legacy routes go.

Offered:

- `GET /api/v1/toolsets`: page-number (`limit` 1–100, default 24); filters `toolset_type`, `interface_language`, `platform`, `release_channel`, `q`; `sort` defaults to `resource_updated_desc`. Toolsets whose author is not renderable are excluded from both `items` and `total`.
- `GET /api/v1/users/{user_id}/toolsets`: page-number, newest first; an unrenderable user is 404.
- `GET` / `PATCH` / `DELETE /api/v1/toolsets/{toolset_id}`, `POST /api/v1/toolsets` (Idempotency-Key required, 201), and `GET …/source` (the Markdown, for editors).
- Resources: `GET` / `PATCH` / `DELETE …/resources/{resource_id}`, `POST …/resources` (Idempotency-Key required), and `GET …/resources/{resource_id}/source` (secrets, for editors, no download counted).
- `POST …/resources/{resource_id}/downloads`: the only way to get a download link, extraction code or archive password, and it counts the download. Anonymous callers may use it.
- Uploads: `POST …/uploads` (init), `GET …/uploads/{upload_id}` (resume), `PATCH …/uploads/{upload_id}` `{state: "completed"}`, `DELETE …/uploads/{upload_id}` (abort). Only the caller who started an upload can see or change it. Over the daily quota → `429 QUOTA_EXCEEDED` with `Retry-After`.
- `PUT …/practicality` `{rating}`. The legacy GET is gone: the detail carries the average, count, five-bucket distribution and `viewer.practicality_rating`.

Names: `title`, `toolset_type`, `release_channel`, `interface_language`, `toolset_resources`; a resource has `resource_type` `file`/`link` and a nested `archive` or `link`. The detail sends `content` (a document), not HTML, and no comment preview (read `/wall-comments?subject_type=toolset`). `resource_updated_at` is typed nullable to match the work summary's field of the same name, but a toolset always has one.

## 2026-09-23 (GE galgame entities)

Breaking for all eighteen `/api/galgame-{tag,official,engine,series,staff,character}*` faces; they are gone. No App build calls them.

Offered (all public `GET`s):

- Tags: `GET /api/v1/tags` (page-number; `q` searches, `include_nsfw` adds adult tags), `GET /api/v1/tags/{tag_id}` (an adult tag is `NOT_FOUND` without `include_nsfw=true`), `GET /api/v1/tags/{tag_id}/works`, and `GET /api/v1/tagged-works?tag_ids=` (works carrying every one of 1–10 tags; more than 10 is `400`, no longer truncated).
- Companies (the old 会社 / official): `GET /api/v1/companies` (`q`, `company_kind`), `GET /api/v1/companies/{company_id}`, `GET /api/v1/companies/{company_id}/works` (items `company_work` with `via_company`; `via=own|imprint`), `GET /api/v1/companies/{company_id}/graph`, and `GET /api/v1/wiki-company-redirects/{wiki_company_id}` for retired wiki numbers.
- Engines: `GET /api/v1/engines` (`q`), `GET /api/v1/engines/{engine_id}`, `GET /api/v1/engines/{engine_id}/works`.
- Series: `GET /api/v1/series` (`q`, `include_nsfw`), `GET /api/v1/series/{series_id}`, `GET /api/v1/series/{series_id}/works`.
- Staff are `credit_name`s (a name someone is credited under): `GET /api/v1/credit-names?q=` (`q` required), `GET /api/v1/credit-names/{credit_name_id}`, and the cursor list `GET /api/v1/credit-names/{credit_name_id}/credits`.
- Characters: `GET /api/v1/characters?q=` (`q` required), `GET /api/v1/characters/{character_id}` (adult traits need `include_nsfw=true`), and the cursor list `GET /api/v1/characters/{character_id}/appearances`.

Every works sub-collection is page-number, returns `WorkSummary` (`object: "work"`, shared with the coming `/works`), and takes `resource_type`, `resource_platform` and `resource_language` (resource-axis keys such as `win`, `zh-cn`), `game_type` and `sort` (`<field>_<asc|desc>`, default `resource_updated_desc`).

Shape vs the retired faces: every name is the catalog primitive `display_name` / `latin` / `localized{}` (the server no longer picks one by the name-preference cookie); intros are `intros[]` with `locale` / `value` / `is_machine` / `data_source`; links are `{site, url}` and clients label `site`; roles are `{role_key, display_name}`; NSFW is the explicit `include_nsfw`, never the settings cookie. A merged company, credit name or character answers `404 ENTITY_MERGED` with `object` and `current_id`, replacing `200 {moved_to}`. Counts are `catalog_work_count` (catalog's, NSFW included) beside each collection's `total`.

## 2026-09-23 (X2 wall follows and read receipts)

Offered:

- `GET /api/v1/me/walls` — the comment walls the caller follows (cursor; a page can be short while `next_cursor` is present). Items are `followed_wall` with `subject_type`, `subject_id`, and a `work` (galgame walls) or `website` (website walls) reference
- `GET /api/v1/me/walls/{subject_type}/{subject_id}` — the caller's `wall_state` (`is_following`); reading marks nothing
- `PUT` / `DELETE /api/v1/me/walls/{subject_type}/{subject_id}/follow` — follow / unfollow; unfollowing keeps replies and mentions notifying
- `PUT /api/v1/me/walls/{subject_type}/{subject_id}/read-marker` — marks the wall read (only for a wall the caller wrote on or follows) and returns the state

Walls are addressed by the same `subject_type` vocabulary as `/wall-comments`. A missing page is `404`, an unanswered spoiler quiz `403 QUIZ_ANSWER_REQUIRED`, an upstream failure `503`.

Retired: `POST /api/community/wall/read`, `POST /api/community/wall/follow`, `GET /api/community/following`.

## 2026-09-23 (X2-auth account)

Offered:

- `GET /api/v1/me/account` — who the credential is: `name` and `avatar` from the account center's current record (both `null` when the account no longer exists), the ranked `roles` the credential carries, and `content_stance` `{is_adult_confirmed, nsfw_display}`. `content_stance` is always `null` for a Bearer request, which carries no stance; read it from the account center. A Bearer request never carries moderator, admin or ren.

Retired: `GET /api/auth/me`. `POST /api/auth/oauth/callback` and `POST /api/auth/logout` stay outside v1: they are the web's cookie-session plumbing, not faces the App may call.

## 2026-09-23 (G0 galgame id is the catalog work id)

Breaking for the one v1 galgame face.

- `GET /api/v1/galgames/{galgame_id}/moyu-patches` (`listGalgameMoyuPatches`) is now `GET /api/v1/works/{work_id}/moyu-patches` (`listWorkMoyuPatches`), tag `works`. Same response. The old path answers 404. No App build calls it.
- Every galgame id the site shows — the `/galgame/:id` page, every legacy `/api/*` field named `gid` or `galgame_id` — is now the catalog work id. 13,493 pages changed number in the renumber; merged-away numbers are not redirected. The legacy field names are unchanged until their faces move to v1.

## 2026-09-23 (X1b sections)

Breaking for `GET /api/section` and `GET /api/category`; both are gone.

Offered:

- `GET /api/v1/topics?section=<slug>`: a new filter on the existing topic collection. Everything else is unchanged: visibility, `include_nsfw`, sorts, and cursors issued before the filter existed.
- `GET /api/v1/sections?category=`: every topic section in vocabulary order (not paged), with `topic_count`, `view_count` and `latest_topic` (`{object: "topic", id, title, created_at}` or `null`). Empty sections are listed with 0 topics.

The section page is now the topic list filtered by section, with cursor "load more" instead of page numbers. It honors the NSFW stance, which the old face ignored.

## 2026-09-23 (X1a friend links, app version)

Breaking for `GET /api/friend-link`, the four `/api/admin/friend-link*` writes and `GET /api/app/version`; all six are gone.

Offered:

- `GET /api/v1/friend-links`: a flat cursor list, shelf by shelf (`official` → `galgame` → `others`), with an optional `friend_link_category` filter. The old grouped object is gone; clients group the list themselves.
- `GET` / `PATCH` / `DELETE /api/v1/admin/friend-links/{friend_link_id}` and `POST /api/v1/admin/friend-links`, gated by `friend_link.*`.
- `PUT /api/v1/admin/friend-link-order`: `{friend_link_category, friend_link_ids}`, which must name every link on that shelf exactly once.
- `GET /api/v1/app/version`: the app's version gate. The fields are unchanged, plus `object: "app_version"` and no envelope. **App-visible.**

Friend-link field names vs the retired faces: `friend_link_category` (was `category`), `title` (was `name`), `url` (was `link`; must be http or https), `banner` as an `Image` or `null` (was `banner` / `banner_image_hash` / `banner_url`), and `state` = `normal` / `down` (was `status`; `essential` was never used and is gone). `sort_order`, `created` and `updated` are not sent.

## 2026-09-23 (D docs)

Breaking for every `/api/doc/**` face and `GET /api/admin/doc/article`; all sixteen are gone.

Offered:

- `GET /api/v1/docs`: cursor list of help-center docs (`limit` 1–100). `sort` is `position_asc` (default, the staff-set display order), `published_desc` or `views_desc`. Optional filters: `doc_category` and `is_pinned`.
- `GET /api/v1/docs/{doc_slug}`: one doc with `author` and `content` (the content-document node tree). Each read counts one view.
- `GET` / `PATCH` / `DELETE /api/v1/admin/docs/{doc_id}` and `POST /api/v1/admin/docs`: the staff view (`admin_doc`, with `content_markdown`), gated by `doc.edit` / `doc.create` / `doc.delete`.
- `PUT /api/v1/admin/doc-order`: `{doc_ids}` must name every doc exactly once.

Shape vs the retired faces: `doc_category` is a closed enum `galgame` / `notice` / `kun` / `other` and replaces `category_id` + `category{}`. `banner` is an `Image` or `null` and replaces `banner` / `banner_image_hash` / `banner_url`. `is_pinned` was `is_pin`, `view_count` was `view`, `published_at` was `published_time`, `edited_at` was `edited_time`. `path`, `status`, `sort_order`, `created`, `updated`, `tag_ids`, `content_html` and `toc` are gone; clients build `/doc/{slug}` themselves and take the table of contents from the `heading` nodes.

Removed without replacement: doc categories as their own resource (now the enum), doc tags (never used), and the draft/hidden `status` (never used; every doc is public).

## 2026-09-23 (WS website directory)

Offered:

- `GET /api/v1/websites` — cursor-paged directory, newest listing first; `include_nsfw` (default false), `website_category_id`, `website_tag_id`, `include_total`
- `GET /api/v1/websites/{website_host}` — detail by the page's host key; counts a view; `viewer` has `has_liked`, `has_favorited`, `can_edit`, `can_delete`
- `PUT` / `DELETE /api/v1/websites/{website_host}/like` and `/favorite` — idempotent slots returning `website_engagement`
- `GET /api/v1/website-categories`, `GET /api/v1/website-categories/{website_category_slug}`, `GET /api/v1/website-tags`, `GET /api/v1/website-tags/{website_tag_slug}`, `GET /api/v1/website-tag-groups`
- Staff (`website.*`, never a Bearer request): `POST /api/v1/admin/websites` and `GET` / `PATCH` / `DELETE /api/v1/admin/websites/{website_id}` (edit source `admin_website`); the same four for `website-categories`, `website-tags`, `website-tag-groups`
- New error code `WEBSITE_CATEGORY_NOT_EMPTY` (409, extension `website_count`)

Field names vs the retired faces: `host` (was `url` / the card's `domain`), `title` (was `name`), `urls` (was the `domain` array; every element is an http(s) URL), `founded` (was `create_time`), `is_nsfw` (was `age_limit`), `state` (was `status`, same values), `score` (was `price` and `level`), `icon: Image | null` plus `external_icon_url` (were `icon` / `icon_image_hash` / `icon_url`), `view_count` (was `view`), `website_category` / `website_tags` (were `category` / `tags`), `slug` / `label` on categories, tags and groups (were `name` / `label`), `website_tag_group_id` (was `group_id`), `is_multi_select` (was `multi_select`). The detail no longer carries `comment`; read the wall through `/api/v1/wall-comments`.

Retired: every `/api/website`, `/api/website-tag*`, `/api/website-category*` and `/api/website-tag-group` route (21).

## 2026-09-23 (P permissions)

Offered:

- `GET /api/v1/me/permissions` — the caller's effective permission keys in catalog order. Always empty for a Bearer request
- `GET` / `PATCH /api/v1/admin/role-permissions` — the role matrix; `PATCH {changes: [{role, overrides}]}` replaces the listed roles' overrides in one transaction, validated on the combined new state
- `GET` / `PUT /api/v1/admin/user-permissions/{user_id}` — one user's permission layer; `PUT {overrides}` replaces it
- `GET /api/v1/admin/permission-changes` — page-number collection of permission audit entries

Permission keys are an open vocabulary (`^[a-z][a-z_]*(\.[a-z][a-z_]*)+$`), so tolerate keys you do not know. Delegation refusals are `422 VALIDATION_FAILED` with `NOT_PERMITTED` at the refused position. Admin faces need the admin role; a Bearer request never has it.

Retired: `GET /api/perm/mine`, `GET /api/perm/bundles` (no replacement: it published the live role matrix to anonymous visitors), `GET /api/admin/role-permissions`, `PUT /api/admin/role-permissions/:role`, `GET`/`PUT /api/admin/user-permissions/:uid`, `GET /api/admin/permission-audit`.

## 2026-09-23 (TS reports and review inbox)

Offered:

- `GET /api/v1/report-reasons` — public; the reasons the trust service currently offers, `{key, display_name}`. Refreshed every five minutes; `503` when the trust service is down (there is no built-in fallback list any more)
- `POST /api/v1/reports` — `204`; `subject_kind` must be a kind this forum registers, `reason_key` one of the listed reasons, `subject_url` (optional) must start with `https://www.kungal.com/`. `Idempotency-Key` optional; repeating a report of the same content counts once
- `GET /api/v1/admin/review-items` — page-number collection (`page`, `limit` 1–100, optional `state`), needs `trust.review`; a Bearer request never has it
- `GET` / `PATCH /api/v1/admin/review-items/{review_item_id}` — the item with its reports; `PATCH {state: claimed | actioned | dismissed}` claims or decides it

Vocabulary: `state` `pending` / `claimed` / `actioned` / `dismissed`; `opened_by` is open (`reports`, `ai_text`, `ai_image`, `community_forward`, `mislabel`, `manual`, `ai_sample`, and `unknown` for an origin this forum does not know yet); `action` `none` / `hide` / `remove` / `warn_user` / `restrict` / `escalate_idp`. `subject_kind` and `reason_key` are open vocabularies.

Retired: `GET /api/report/reasons`, `POST /api/report/submit`, `GET /api/admin/trust/review-items`, `GET /api/admin/trust/review-items/:id`, `POST /api/admin/trust/review-items/:id/claim`, `POST /api/admin/trust/review-items/:id/decide`. `POST /api/trust/callback` stays: it is infra's HMAC callback, not an end-user face.

## 2026-09-23 (UP update logs and todo board)

Offered:

- `GET /api/v1/update-logs` — cursor-paged changelog, newest first; `GET /api/v1/update-logs/{update_log_id}`
- `POST` / `PATCH` / `DELETE /api/v1/update-logs…` — staff only (`update_log.create` / `.edit` / `.delete`); never available to a Bearer request
- `GET /api/v1/todos` — cursor-paged todo board, newest first; `state` filter and `include_total`; `GET /api/v1/todos/{todo_id}`
- `POST /api/v1/todos` — any signed-in user; `Idempotency-Key` required; trust-and-safety checked (`CONTENT_REJECTED`)
- `PATCH /api/v1/todos/{todo_id}` — either edit `project` / `text` (author only) or move `state`: claim, complete, discard, release, reopen. Every allowed move is mirrored by a `viewer.can_*` flag
- `DELETE /api/v1/todos/{todo_id}` — `update_log.delete`

Field names vs the retired faces: `change_type` (was `type`; tokens `perf` and `style`, were `pref` and `styles`), `release_version` (was `version`), `text` (was `content`, plain text, not Markdown), `project` (was the todo's `type`), `state` `pending` / `in_progress` / `done` / `discarded` (was integer `status` 0–3), `author` / `claimer` (were `user` / `claimed_user`), `completed_at` (was `completed_time`, now only set when `done`).

Retired: every `/api/update/**` route (`history` GET/POST/PUT/DELETE, `todo` GET/POST/PUT/DELETE, `todo/claim`, `todo/complete`, `todo/discard`).

## 2026-09-23 (U2 users)

Breaking for `GET /api/user/:id`, `GET /api/user/:id/floating`, and `GET`/`PUT /api/user/notification-preferences`.

Offered:

- `GET /api/v1/users/{user_id}` — public profile; unknown and banned/deleted users answer 404
- `GET /api/v1/users?ids=` — batch `UserRef` lookup (comma-separated ids, 1–100); requested ids that are unknown or not renderable sit in `missing`
- `GET` / `PUT /api/v1/me/notification-preferences` — `{ muted_types }` using v1 notification type tokens (`favorited`, `best_answer_chosen`, `reply_pinned`, `followed_thread_activity`, `lottery_drawn`, …; `chat` stays `chat`)

`UserProfile` counts sit under `counts` and are renamed (`topic` → `topic_count`, `reply_created` → `reply_count`, `galgame` → `published_galgame_count`, `galgame_toolset` → `toolset_count`, `upvote` → `received_upvote_count`, `daily_topic_count` → `topic_today_count`, …). `status` is gone.

Retired: `GET /api/user/:id`, `GET /api/user/:id/floating`, `GET`/`PUT /api/user/notification-preferences`.

## 2026-09-23 (U1 me)

Breaking for `/api/user/status` and the other U1 `/api/user/*` faces in this segment.

Offered:

- `GET /api/v1/me` — caller's cached moemoepoint, today's check-in gate, unread flag, creator flag, today's toolset upload bytes
- `POST /api/v1/me/check-ins` — Beijing-day check-in; reward is a deterministic function of that day (0–7)
- `GET /api/v1/me/moemoepoint-entries` — cursor ledger (`limit` 1–50); each entry has `source`: `this_site` / `account_center` / `other_site`
- `GET` / `PUT /api/v1/me/preferences` — cloud preference document; `If-Match` is the quoted version
- `PUT /api/v1/me/nsfw-display` — `hide` / `blur` / `show`
- `GET /api/v1/users?q=` — name search (`limit` 1–20), not paged
- `PATCH /api/v1/me/profile` — `name` and/or `bio`
- `PUT /api/v1/me/avatar` — multipart field `file`
- `GET /api/v1/me/creator-status` / `POST /api/v1/me/creator-applications`

`GET /api/v1/me` field names vs the retired status face: `moemoepoint` (was `moemoepoints`), `has_checked_in_today` (was `is_check_in`), `has_unread_messages` (was `has_new_message`; notifications and private messages, not system announcements), `toolset_upload_today_bytes` (was `daily_toolset_upload_bytes`).

Retired: `GET /api/user/status`. The rest of this segment's `/api/user/*` faces are gone with it (`check-in`, `moemoepoint/log`, `preferences`, `nsfw`, `search`, `bio`, `username`, `avatar`, `creator/status`, `creator/apply`). `GET`/`PUT /api/user/notification-preferences` stays until U2.

## 2026-09-18

First `/api/v1` surface.

Offered:

- `GET /api/v1/problems` — error-code catalogue
- `GET /api/v1/problems/reasons` — field-level reason catalogue
- `GET /api/v1/topics` — cursor-paged topic list (`listTopics`)

Error codes in the registry (`pkg/problem`, `openapi/problems.json`):

- platform: `MALFORMED_BODY`, `INVALID_PARAMETER`, `UNKNOWN_ENUM_VALUE`, `LIMIT_TOO_LARGE`, `INVALID_CURSOR`, `UNKNOWN_SORT`, `MISSING_CREDENTIAL`, `INVALID_CREDENTIAL`, `SCOPE_REQUIRED`, `NOT_FOUND`, `METHOD_NOT_ALLOWED`, `IDEMPOTENCY_KEY_REUSED`, `UNSUPPORTED_MEDIA_TYPE`, `VALIDATION_FAILED`, `INTERNAL_ERROR`, `SERVICE_UNAVAILABLE`
- kungal: `ACCOUNT_BANNED`, `IDEMPOTENCY_REQUEST_IN_PROGRESS`

## 2026-09-18 (W0b-1)

- The document no longer writes `"additionalProperties": true` on object schemas. It meant the same as leaving the keyword out, and unknown request-body fields are still accepted. Code generators rendered it as a catch-all map (openapi-typescript: `[key: string]: unknown`), which let a misspelled field compile. Regenerate client types.

## 2026-09-18 (W0b-3)

Both changes break clients generated from the earlier document. No deployment served that document.

- `TopicSummary.user` is now `author`. A field that refers to a person is named for the role that person plays in the resource; `user` is a forbidden name (gate G8).
- List bodies no longer declare `total`. None of the three collections accepts `include_total`, so the field could never appear. A collection that offers `include_total` returns a counted list that declares `total`, and gate F9 keeps the parameter and the field together.

## 2026-09-18 (W1 contract)

Additive.

- The document now defines the content node vocabulary: `ContentDocument`, the `BlockNode` and `InlineNode` unions (`oneOf`, discriminated on `object`) and their member schemas. No operation returns a document yet; the root extension `x-content-document` references it so it stays in the published document, and it goes away when the topic detail read returns `content`. Rules for rendering, including unknown node types, are in `docs/proj/api-v1/03-content-doc.md`.

## 2026-09-19 (W2)

Additive, apart from the removed extension.

- `GET /api/v1/topics/{topic_id}` (`getTopic`), `GET /api/v1/topics/{topic_id}/replies` (`listTopicReplies`), `GET /api/v1/replies/{reply_id}` (`getReply`) and `POST /api/v1/topics/{topic_id}/views` (`recordTopicView`).
- Reading a topic does not count a view. A client calls `recordTopicView` once when a reader actually opens the topic, and never when it prefetches or renders elsewhere.
- Replies are in floor order with a cursor; `from_floor` opens the list at a floor, and `sort=floor_desc` with `from_floor` one below reads back from it. The pinned reply and the best answer come with the topic and also appear at their floors.
- The root extension `x-content-document` is gone: `Topic.content` references `ContentDocument`.

## 2026-09-19 (moyu patches)

Additive.

- `GET /api/v1/galgames/{galgame_id}/moyu-patches` (`listGalgameMoyuPatches`): the pages www.moyu.moe holds for a galgame, each with its live resources. Public, never paged, and an answer may be up to 30 minutes old. It replaces the website's browser call to moyu's retired `/api/hikari`; the forum reads NextMoe's `/v2/moyu` face server-side.
- `types`, `languages`, `platforms` and `storage` are moyu's own vocabulary and are open (`x-vocabulary: moyu_patch_vocabulary`): show an unknown token as it is.
- `note_markdown` is moyu's Markdown source, not a content document. No download link, share code or password is carried; send a reader to `web_url`.
- `publisher` is a `UserRef` resolved from this forum's account service; the account is the same one on the forum.
- `SERVICE_UNAVAILABLE` when moyu, the catalog or the account service cannot be reached.

## 2026-09-19 (W3 / W4: topic writes and interactions)

Additive.

- Writes: `createTopic` (`POST /topics`), `updateTopic` (`PATCH /topics/{topic_id}`, including `state` for hiding and showing), `createReply` (`POST /topics/{topic_id}/replies`), `updateReply` (`PATCH /replies/{reply_id}`), `deleteReply` (`DELETE /replies/{reply_id}`). Both creates require `Idempotency-Key` and answer 201 with `Location` and the full resource.
- Edit context: `getTopicSource` and `getReplySource` return `content_markdown` and the editable fields to callers with `can_edit`. `content_markdown` is sent nowhere else.
- Interactions: `PUT` / `DELETE` on `/topics/{topic_id}/reactions/{reaction}`, `/topics/{topic_id}/favorite` and `/replies/{reply_id}/reactions/{reaction}` set and unset the caller's own state, answer 200 with an engagement snapshot, and are idempotent. like and dislike are the reaction tokens `like` and `dislike`.
- `upvoteTopic` (`POST /topics/{topic_id}/upvotes`) records a repeatable, charged upvote and requires `Idempotency-Key`; `listTopicUpvotes`, `listTopicReactions` and `listReplyReactions` are cursor lists, newest first.
- `setBestAnswer` / `clearBestAnswer` and `pinReply` / `unpinReply` on `/topics/{topic_id}/best-answer` and `/topics/{topic_id}/pinned-reply` answer 200 with the topic.
- `TopicViewer` gains `can_edit`, `can_hide`, `can_unhide`, `can_like`, `can_upvote`, `can_set_best_answer`, `can_pin_reply`; `ReplyViewer` gains `can_edit`, `can_delete`, `can_like`.
- New codes: `PERMISSION_REQUIRED` (moderation domain, as infra), `CONTENT_REJECTED`, `TOPIC_DAILY_LIMIT_REACHED` (with `limit`), `MOEMOEPOINT_INSUFFICIENT` (with `required`), `SELF_LIKE_FORBIDDEN`, `SELF_UPVOTE_FORBIDDEN`.
- `bumped_at` now also names upvotes, a new best answer, and edits of the title or body, which bump under the same 3-month rule.

## 2026-09-22 (legacy topic routes removed)

Breaking for anything still on `/api/topic/**`. Nothing was: the web moved with W1–W4 and the App binds `/api/v1` only (`docs/proj/app-direct-api.md`).

22 legacy routes are gone. Their v1 replacements, in the same order:

| removed | use instead |
|---|---|
| `POST /api/topic` | `POST /api/v1/topics` |
| `PUT /api/topic/:tid` | `PATCH /api/v1/topics/{topic_id}` |
| `GET /api/topic/:tid` | `GET /api/v1/topics/{topic_id}` |
| `PUT /api/topic/:tid/hide` | `PATCH /api/v1/topics/{topic_id}` with `state` |
| `PUT /api/topic/:tid/like`、`/dislike`、`/reaction` | `PUT` / `DELETE /api/v1/topics/{topic_id}/reactions/{reaction}` |
| `PUT /api/topic/:tid/favorite` | `PUT` / `DELETE /api/v1/topics/{topic_id}/favorite` |
| `PUT /api/topic/:tid/upvote` | `POST /api/v1/topics/{topic_id}/upvotes` |
| `GET /api/topic/:tid/upvotes` | `GET /api/v1/topics/{topic_id}/upvotes` |
| `GET /api/topic/:tid/reaction/history` | `GET /api/v1/topics/{topic_id}/reactions` |
| `PUT /api/topic/:tid/best-answer` | `PUT` / `DELETE /api/v1/topics/{topic_id}/best-answer` |
| `POST /api/topic/:tid/reply` | `POST /api/v1/topics/{topic_id}/replies` |
| `PUT /api/topic/:tid/reply` | `PATCH /api/v1/replies/{reply_id}` |
| `DELETE /api/topic/:tid/reply` | `DELETE /api/v1/replies/{reply_id}` |
| `GET /api/topic/:tid/reply` | `GET /api/v1/topics/{topic_id}/replies` |
| `GET /api/topic/:tid/reply/detail` | `GET /api/v1/replies/{reply_id}` |
| `PUT /api/topic/:tid/reply/like`、`/dislike`、`/reaction` | `PUT` / `DELETE /api/v1/replies/{reply_id}/reactions/{reaction}` |
| `PUT /api/topic/:tid/reply/pin` | `PUT` / `DELETE /api/v1/topics/{topic_id}/pinned-reply` |
| `GET /api/topic/:tid/reply/reaction/history` | `GET /api/v1/replies/{reply_id}/reactions` |

`internal/middleware/idempotency.go` went with them, and with it the envelope codes `237` / `238`. v1 idempotency is `internal/apiv1/idempotency.go`: `409 IDEMPOTENCY_REQUEST_IN_PROGRESS` and `409 IDEMPOTENCY_KEY_REUSED`.

A caller left on one of these now gets `401`, not `404`: the auth boundary in `router.go` is a `Use()` on `/api`, so every unmatched legacy path answers 401 to an anonymous request. That is how `/api/**` has always answered an unknown path — only `/api/v1/**` answers the proper `404` problem+json.

Still legacy, still mounted, each waiting on its own wave: `/topic/:tid/reply/locate`, `/topic/interactions/mine`, the four `/topic/draft*`, the four `/topic/:tid/comment*`, the six `/topic/:tid/poll*` and the eleven `/topic/:tid/lottery*`.

## 2026-09-22 (RC · comment walls)

Additive for the App; the web's legacy comment-wall routes are removed.

- Nine operations on one collection, `/api/v1/wall-comments`, serve the comment walls of six kinds of page: `listWallComments`, `createWallComment` (requires `Idempotency-Key`), `getWallComment`, `getWallCommentSource`, `updateWallComment`, `deleteWallComment`, `likeWallComment` / `unlikeWallComment` (`PUT` / `DELETE …/like`, both idempotent) and `flagWallComment` (`POST …/flags` → `204`).
- A wall is named by `subject_type` (`galgame`, `galgame_rating`, `galgame_resource`, `galgame_quiz`, `toolset`, `website`) and `subject_id`. The collection is flat because none of the six pages has a v1 read yet, and gate G17 refuses a write under a path id with no `GET`.
- `WallComment` is its own resource with its own id space; it is not a topic `Comment`. Its body is full Markdown as a `ContentDocument`. A deleted comment stays in lists as a tombstone (`state: deleted`, empty document) so its replies keep their parent. `addressee` is the person the comment is addressed to and can be `null`, which is why it is not called `in_reply_to_user`.
- The list sends no `total`. The page's own `comment_count` is the number to show.
- New error codes: `RATE_LIMITED` (platform, 429; the community service's new-account limit, no `Retry-After` because upstream sends none), `QUIZ_ANSWER_REQUIRED` (kungal, 403; the wall of a quiz that hides its game or has spoilers is open only to its author, those who answered, and staff with that wall's permissions) and `INVALID_STATE_TRANSITION` (me, 409; editing, liking or flagging a tombstone, or posting to a closed wall). It was named in 01 §2 but had never been registered.

Removed legacy routes (22):

| Legacy | v1 |
|---|---|
| `GET /api/{galgame-rating,website,toolset,galgame-resource,galgame-quiz}/:id/comments`, `GET /api/galgame/:gid/comments` | `GET /api/v1/wall-comments?subject_type=…&subject_id=…` |
| `POST` on the same six | `POST /api/v1/wall-comments` |
| `DELETE /api/{…five…}/:id/comments/:postId`, `DELETE /api/galgame/comments/:postId` | `DELETE /api/v1/wall-comments/{wall_comment_id}` |
| `PUT /api/galgame/comments/:postId` | `PATCH /api/v1/wall-comments/{wall_comment_id}` |
| `PUT /api/galgame/comments/:postId/like` (a toggle) | `PUT` / `DELETE /api/v1/wall-comments/{wall_comment_id}/like` |
| `POST /api/galgame/comments/:postId/flag` | `POST /api/v1/wall-comments/{wall_comment_id}/flags` |
| `GET /api/galgame/:gid/comments/locate` | not rebuilt; migration 135 rewrote the notification links that needed it |

Following a wall and its read receipts stay on the legacy `/api/community/wall/*` routes for now; `thread_id` may be sent as `0` and the server finds the thread.

## 2026-09-23 (M · messages)

Twelve operations under `/api/v1/me/**` replace the eleven `/api/message/**` routes, which are deleted. The App never called the old routes, so nothing it shipped breaks.

Notifications:

- `GET /api/v1/me/notifications` (`listNotifications`) — cursor list, newest first. `is_muted` picks the partition (types the caller has not muted, or only the muted ones); `notification_type` narrows it. Rows whose actor is banned are left out.
- `GET /api/v1/me/notifications/summary` (`getNotificationSummary`) — `unread_count`, `muted_unread_count`, `latest`. The counts are SQL counts and include rows the list leaves out.
- `GET /api/v1/me/notifications/{notification_id}` (`getNotification`)
- `PUT /api/v1/me/notifications/read-marker` (`markNotificationsRead`) — `{up_to_id, is_muted}` marks one partition read up to an id. It never touches the other partition or rows newer than `up_to_id`.
- `DELETE /api/v1/me/notifications/{notification_id}` (`deleteNotification`) — `404` for a row that is not the caller's (the old route answered 200).
- `Notification.notification_type` is a closed vocabulary of 18 tokens. Five differ in meaning from their old stored names: `favorited`, `followed_thread_activity` (new comments in a thread you follow, not "someone followed you"), `best_answer_chosen`, `reply_pinned`, `resource_link_reported` (someone reported your link, not "your thing expired"). The other renames are snake_case only: `quiz_answered`, `edit_requested`, `edit_merged`, `edit_declined`, `lottery_won`, `lottery_drawn`, `lottery_code_expired`, `poll_closed`. The table is in `waves/m-message.md` §3.3.
- `path` is the in-site web path of the target. A structured target reference may be added later as an addition.

Private messages, addressed by the other participant's user id:

- `GET /api/v1/me/conversations` (`listConversations`), `GET /api/v1/me/conversations/{user_id}` (`getConversation`) — works before any message exists and creates nothing.
- `GET /api/v1/me/conversations/{user_id}/messages` (`listDirectMessages`), `GET …/messages/{message_id}` (`getDirectMessage`) — reading never marks anything read.
- `POST /api/v1/me/conversations/{user_id}/messages` (`sendDirectMessage`) — `Idempotency-Key` required. Returns `201` with the message.
- `PATCH /api/v1/me/conversations/{user_id}/messages/{message_id}` (`updateDirectMessage`) — `{state: "recalled"}`. A recalled message carries an empty `content` document.
- `PUT /api/v1/me/conversations/{user_id}/read-marker` (`markDirectMessagesRead`)

System announcements (`/api/message/admin`) are gone with no replacement. The table was empty and nothing wrote to it.
