# G4 · 作品详情

> G 轨第四段（G0 改号之后），2026-09-23。作品详情页及其卫星：**5** 条旧路由。G 号段 140–159；**145 是已用的最后一个号**（G0 140/141、G1 142、G2 143、G1.1 144、G3 145）。本轨无迁移。
> 契约与变异题同一个提交，早于实现。普查全文在 [census/galgame-core.md](census/galgame-core.md) §0、§2、§4、§5、§9–§13。普查写于 G0 之前：作品 id 现已是 catalog work id，路径与字段一律 `work_id` / `{work_id}`（根 CLAUDE.md 铁律 3）；普查 §0 身份、发现 1–3、10,289 撞车全部作废。`DELETE /galgame/:id` 是投稿撤回（`GalgameSubmissionHandler.Withdraw`），归 **G7**，本轨不碰。
> 编排者 2026-09-23：`Work` 是 `WorkRef` 的严格超集——WorkRef 每个字段同名同型，只加不改。G3 已落地的薄面 `GET /api/v1/works/{work_id}`（200 `WorkRef`）由本轨把 200 扩成 `Work`。

## 1. 普查（生产实测 2026-09-24，论坛库；数字按任务书原文）

### 1.1 数字

| 事实 | 数字 |
|---|---|
| `galgame` 行 | 16,211；published 9,820；`content_limit = 'nsfw'` 8,693；`resource_publish_banned` 499；`creator_user_id` 有值 13,495 |
| `galgame_like` | 39,452 行，来自 3,749 个用户、落在 8,899 部作品上；`like_count` 从不为负；每部作品的 `like_count` 与行数一致；439 条赞落在未发布作品上，0 条落在没有本地行的作品上 |
| 每用户赞数 | 最大 7,203；p99 124；中位 1 |
| 链到 `/galgame/<id>` 的 `liked` 消息 | 52,997 |
| `galgame_contributor` | 每部最多 44，p99 14；source 0：17,965 行，source 1：4,294 |
| 每部 `galgame_rating` | 最多 62，p99 22 |
| `galgame.view` | 最大 2,845,980 |
| 本地 `galgame.favorite_count > 0` | 12,857 部，最大 417（现行详情发的是 catalog 的 favorites 指标，不是这一列） |
| 本地重定向表 | **无**（`galgame_redirect` 自 G0 起已不在；合并是 catalog 的） |

### 1.2 调用方

- 网页 `GET /galgame/:id`：
  - `apps/web/app/pages/galgame/[id]/index.vue:14`（当前页文件是 `[id]`，普查写的 `[gid]` 已过时；另带无用 query `galgame_id`）
  - `apps/web/app/components/galgame/PreviewModal.vue:19`（投稿审核预览，G7 的页面读 G4 的详情）
  - `apps/web/app/components/galgame-edit/review/Detail.vue:75`（审阅提案时拉详情，只取 tag / official / engine / series / character / staff 的 id→name 表）
- Nitro（服务端调用，普查容易漏）：`apps/web/server/utils/kunOgCard.ts:104` `buildGalgame`，SFW 且 `indexed` 才出卡。
- 赞：`apps/web/app/components/galgame/Like.vue:36`（`PUT` 切换；响应不读，前端自己翻转计数）。
- 外链：`apps/web/app/components/galgame/link/Link.vue:9`（lazy；另带无用 query `galgame_id`；只用 `name` / `link`）。身份链（VNDB / Bangumi / 批评空间）不走这条，走详情的 `refs`。
- 我的互动：`apps/web/app/composables/useMyGalgameInteractions.ts:42`，卡片 `ensureLoaded(gids)` 把作品 id 放进 `work_ids`（`card/Card.vue:56`）。
- `GET /galgame/drafts`：`apps/web/app/**`、`apps/web/server/**`、`apps/web/shared/**` **零命中**（只剩 `router.go` 与 golden）。方案③之后用户不再认领游戏。
- Flutter App：普查写 Dart 源零命中；`docs/proj/app-direct-api.md` 把 `/api/galgame/:gid` 列为前瞻供数，不是现网调用。本 worktree 不含 `kungal-apps`，本节按普查 + 文档再核。

详情页还读、但不属于本轨写面的：资源列表已是 G3 `GET /works/{work_id}/resources`；评分墙本应是 GR `GET /ratings?work_id=`（**今天并没有**：`Galgame.vue:72` 用详情嵌入的 `ratings[]`，`header/rating/*` 与雷达也吃这份嵌入；全站评分墙 `rating/card/Container.vue` 不带 `work_id`）。封面投票写、游玩时长写、收藏夹写仍是 G6。

### 1.3 旧面的问题（普查编号 E*，对应 census 发现号）

G0 之后作废、不再建模的：发现 1–3（论坛 gid 当 catalog work id / `liked[]` 与 `favorited[]` 两套 id / 收藏夹 work id 当 gid）、§0 的 10,289 撞车、§9 的 `CatalogWorkIDForGID` 翻译、§10 命名表里「先译 catalog id」。

| 编号 | 事实 |
|---|---|
| E1 | 详情对 SFW 读者不加整页 `content_limit` 闸（`catalog_face.go:519` 的 `openPopulation`），只剥 `category=="sexual"` 的标签（`galgame_service.go:243`，`galgame_mapper.go:265`）。列表有闸，深链没有（发现 5） |
| E2 | `GET …/link/all` 对不存在 / hidden 回 **200 空数组**（`catalog_detail.go:296`），详情是 404（发现 7）。`GalgameLink.id` / `user` 恒零值（发现 8）；`LinkDisplayName` 产出中文标签（GE E9 同类，F8） |
| E3 | `PUT /:id/like` 是切换（`interaction_repo.go:41`）。不查 catalog、不查 hidden、不查 `published`；`Ensure` 可种本地行。调用者是 `creator_user_id` → `400/233 "您不能给自己点赞"`。`creator_user_id` 空时 `ownerID=0`，闸放行，随后给 user 0 调萌萌点（发现 9）。`like_count-1` 无下限（生产从未为负）。同一请求重放会把状态打回去 |
| E4 | `GetMyInteractions` 的 `liked[]` 忽略 `work_ids`、一次拉该用户全部赞（生产单用户最多 7,203）。`favorited[]` 走 `MyFolderHoldings`。`work_ids` 空 / token 空 / catalog 空：只回 likes，`favorited=[]`，不报错。`folder:read` 不足只打 `scopeWarn` 一行/分钟（`scope_warn.go:27`，文案 `galgame: my folders unreadable, token lacks folder:read`），仍 200 空收藏。详情 `is_favorited` 同口径（发现 22） |
| E5 | `Hydrate` 丢 OAuth 错，缺人填 Placeholder `Status=0`（发现 10）。贡献者不跑 `IsRenderable`；评分跑了。作者被封禁时详情仍 200 |
| E6 | 详情 `introduction[].intro` 是服务端渲染的 HTML（`galgame_mapper.go:52`）；`intro_text` 是第一条的 Markdown 源。名字走 `NamePreference` cookie（发现 17、18） |
| E7 | `IncrementView` 在 goroutine 里（`galgame_service.go:205`），错误丢掉（发现 19） |
| E8 | 详情把该作品全部 `galgame_rating` 嵌进去，无分页，`ORDER BY created DESC` 无 id 决胜（发现 18）。生产单作品最多 62，p99 22 |
| E9 | `favorite_count` 来自 catalog popularity `source=nextmoe metric=favorites`（`catalog_face.go:212`），与本地 `galgame.favorite_count` 不是同一计数 |
| E10 | `is_on_forum` 详情 = 本地行存在（`galgame_service.go:230`），列表 = `published`（发现 16）。详情另有 `indexed = published` |
| E11 | `status` 裸整数 0/2（claim live → 0）；`moved_to` / `galgame_redirect` 已随 G0 消失。页面今天找不到就 `KunNull`，不再 301 |
| E12 | `CatalogWorkDetail` 走 `CatalogGet`（`catalog_face.go:301`），404 体丢掉，**分不出合并和缺失**。会社 / 人名 / 角色已经走 `catalogGetRecord`（先看 `ENTITY_MERGED`）。封面槽 `catCoverSlot.Sexual`、详情封面 / 截图行的 `Sexual` 都按 `int` 解码，未判定与「判为安全的 0」不分（g-plan 欠着的活） |
| E13 | `GET /galgame/drafts` 是旧认领漏斗（`claimed=false`），零调用方（发现 6） |
| E14 | `parseCSVInts` 静默丢非法 id、超 100 截断（发现 21） |

## 2. 范围与寻址

路径段就是目标。路径 `{work_id}` 与响应字段 `work_id` / `id` 对齐（K1：作品自己的 id 字段仍是 `id`，与 `WorkRef.id` 同名同型；子资源才把路径参数写成 `work_id`）。集合 **`/api/v1/works`**，对象 **`work`**。

G3 为过 G17 落地的薄面 `GET /works/{work_id}` 本轨扩 200。网页 `/galgame/:id` 本轨切到这里。

| # | 旧 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /galgame/:id` | `GET /api/v1/works/{work_id}` → **200 `Work`**（拓宽 G3 薄面） | optional |
| 2 | `PUT /galgame/:id/like`（切换） | `PUT` + `DELETE /api/v1/works/{work_id}/like`（K16 槽；都 200） | required |
| 3 | `GET /galgame/:id/link/all` | **折进 `Work.links`**（`CatalogLink[]`）；独立路由删除 | — |
| 4 | `GET /galgame/interactions/mine` | `GET /api/v1/me/work-states?work_ids=` — `/me/topic-states` / G2 `/me/quiz-states` 形状：每个可读 id 一条 `{object: "work_state", work_id, has_liked, has_favorited}`，其余进 `missing`；1–100 逗号形 | required |
| 5 | `GET /galgame/drafts` | **删除，无替代**（零调用方；方案③退休的认领漏斗） | — |

5 条旧路由全删，`legacy_route_baseline` 下调 **5**。v1 共 **4** 个操作（getWork 拓宽、赞拆槽、work-states；外链与草稿不占操作）。`DELETE /galgame/:id` 仍在，直到 G7。

代码位置：拓宽后的 `getWork` 与赞槽挂已有的 `internal/galgame/apiv1`（与 `listWorkMoyuPatches` 同包）；`listMyWorkStates` 同包，OpenAPI tag `me`。测试 `internal/app/v1_work_*_test.go`。`getWork` / 赞槽 tag 是 `works`。

本轨一并做掉 g-plan「欠着的活」两条（见 K-G27）：

1. `catCoverSlot.Sexual`（以及详情封面 / 截图行上同形的 `Sexual`）改成指针，未判定是 `sexual: null`，判为安全是 `0` → `Image.sexual = "safe"`；竖版封面的等级流进 `WorkRef.cover.sexual`，横幅流进 `WorkSummary.banner.sexual`。
2. `internal/galgame/apiv1.WorkRefOf` 委托 `workrepr.Ref`（一个构建器）。

## 3. 形状

### 3.1 命名

| 旧 | v1 | 为什么 |
|---|---|---|
| 路径 `:gid` / `:id`，字段不对齐 | `{work_id}`；作品自己的 id 仍是 `id` | K1；与 `WorkRef.id` 同名同型 |
| `gid` / `galgame_id` | `work_id` | G0 |
| `name` / `name_original` / `NamePreference` | catalog 名字三件套 `display_name` / `latin` / `localized{}` | 01 §3；服务端不再挑 |
| `introduction[]`（HTML）/ `intro_text` | `intros: CatalogIntro[]`（`locale` / `value` / `is_machine` / `data_source`） | GE 已落地的形状；`value` 是 Markdown 源，不再渲染 HTML |
| `GET …/link/all` 的 `name`+`link` | `links: CatalogLink[]`（`site` / `url`） | GE；F8：客户端按 `site` 标标签，不发「官方网站」 |
| `refs` 自由 `map[string]string` | `external_ids: WorkExternalID[]`（`site` + `external_id`） | 01：数组与 map 的值要有 schema；禁止自由 map |
| `vndb_id` | `external_ids` 里 `site=vndb` 的那一条 | 不再单列 |
| `user` | `creator: UserRef \| null` | 禁用名；角色是创建者（`creator_user_id`），不是作者 |
| `contributor[]` | `contributors: UserRef[]` | 复数；不可渲染的丢掉 |
| `is_liked` / `is_favorited` / `voted` / `my_playtime` | `viewer.*`；每张封面自己的 `viewer.has_voted` | K9 |
| `content_limit` | `is_nsfw`（WorkRef 已有，编辑轴） | 详情整页不再另发 `content_limit` |
| `age_limit` `all`/`r18` | `content_rating`：`all_ages` / `r18` | 年龄轴，与编辑轴分开；`all` 不是 F1 词 |
| `original_language` | **保持** `original_language`（BCP-47 \| `null`） | 不叫 `lang`：`lang` 已是会社 / 角色 / 人名的「自身语言」（G8 同 schema 但本字段语义是作品原语言，另起名以免客户端混用） |
| `indexed` / 列表义的 `is_on_forum` | `is_published` | WorkSummary 已有；粘性 SEO 旗 |
| 详情义的 `is_on_forum`（本地行存在） | **不下发** | 浏览量在有本地行时 +1，没有就是 0；页面不再用这个旗藏眼睛 |
| `view` | `view_count` | WorkSummary 已有 |
| `rating` / `rating_count` | `rating_score` / `rating_count` | WorkSummary 已有；`rating` 会与 GR 的 PUT 体撞 |
| `platform[]` / `language[]` / `type[]` | `resource_platforms` / `resource_languages`（WorkSummary）+ `resource_types`（本面加，jsonb 轴、`resourcevocab.TypeKeys`） | 与 G3 / GE 同词表；不再读标量列 |
| `resource_update_time` | `resource_updated_at` | WorkSummary 已有 |
| `resource_publish_banned` | `is_resource_publish_banned` | F1；与 G3 `WorkResourcePublishBan` 同名同型 |
| `favorite_count`（catalog 指标） | **保持** `favorite_count`，来源写进字段文档：catalog popularity `source=nextmoe metric=favorites` | 与 `WorkStats.favorite_count` 同 schema（整数 ≥0）；本地列本面不下发 |
| `covers[]` 平铺 hash/url/`kind`/`voted` | `covers: WorkCover[]`：嵌 `image: Image` + `cover_slot` + `site` + `vote_count` + `viewer.has_voted` | `kind` 禁用；`Image` 一处一型（B10）；票是 G6 读侧 |
| `screenshots[]` | `screenshots: WorkScreenshot[]`：嵌 `image: Image` + `caption` + `site` | `violence` 不下发（01 图片 A5） |
| `official[]` | `companies: WorkCompany[]`（CompanySummary 字段 + `lang` + `links` + `attribution_roles`） | `roles` 是话题 `AccessGrants.roles` 的封闭枚举（G8） |
| `engine[]` | `engines: Engine[]` | GE 列表与详情同一形状 |
| `series[]`（id+name） | `series: SeriesSummary[]` | GE 实体摘要；侧栏只画名字，系列面板可不再 `GET /series/{id}` |
| `tag[]` 的 `category`+`spoiler_level` 整数 | `tags: WorkTag[]`（TagSummary + `spoiler`：`none`/`minor`/`major`） | `is_sexual` + `tag_kind` 取代 `content`/`meta`/`technical`/`sexual`；剧透与 `CharacterTrait.spoiler` 同词表（G8） |
| `staff[]` | `credits: WorkCreditGroup[]`（`role_key` + `display_name` + `people`） | 与 GE `CreditRole` 同字段名 |
| `characters[]` 的 `kind` 整数剧透 | `characters: WorkCharacter[]`（CharacterRef 字段 + `figure` + `character_kind` + `spoiler` + `identity` + `voices`） | `kind` 禁用 |
| `ratings[]` | **不下发**；页面改 `GET /ratings?work_id=` | GR 已有；详情只留贝叶斯摘要 |
| `external_ratings[].source`/`.score`/`.rank` | `external_ratings: WorkExternalRating[]`：`site` + `rating_value` + `source_rank` | `score` 是 Website 的整数（可负）；`rank` 是排行条目 1–100 |
| `playtimes[].source` | `playtimes: WorkPlaytimeAggregate[]`：`site` + `minutes` + `vote_count` | `source` 不如 `site` 与 `CatalogLink` 一组 |
| `my_playtime` | `viewer.playtime` | K9 |
| `dlsite_purchase_url` 等 omitempty | `dlsite: DlsiteOffer \| null` | G3 已有，同名同型 |
| `alias[]` | `aliases` | 与会社 / 引擎同名同型（`AliasName[]`，≤512，≤1000） |
| `created` / `updated` | `created_at` / `updated_at` | 禁用名；catalog 的时间 |
| `moved_to` / `status` | **不下发**；合并是 `404 ENTITY_MERGED` | GE 已注册 |
| `GET /interactions/mine` 的 `liked[]`/`favorited[]` | `/me/work-states` 每条 `has_liked` / `has_favorited` | 与 topic/quiz-states 同形；不再一次拉全表赞 |
| 封面 `sexual`/`violence` 整数 | `image.sexual`：`safe`/`suggestive`/`explicit`/`null`；`violence` 不下发 | `repr.Image` |

### 3.2 对象包含关系（K-G22）

**`WorkRef` ⊂ `WorkSummary` ⊂ `Work`，重叠字段同名同型（G8）。** `object` 都是 `"work"`。

`WorkRef`（`internal/apiv1/repr/work.go`）：`object`、`id`、`display_name`、`latin`、`localized`、`cover`（竖版原图 `Image \| null`）、`is_nsfw`。

`WorkSummary`（`internal/galgame/workrepr`）在 WorkRef 上加：`banner`、`release_date`、`release_date_precision`、`maker`、`view_count`、`like_count`、`rating_score`、`rating_count`、`resource_platforms`、`resource_languages`、`resource_updated_at`、`is_published`。没有 `viewer`。没有 `favorite_count`、没有 `resource_types`——这两项只出现在 `Work`（G5 的列表卡片不画它们）。

核过：WorkSummary 嵌入 `repr.WorkRef`，WorkRef 的字段一个都没缺、没有改名改型。没有开放问题。

`Work` 在 WorkSummary 上加详情字段与 `viewer`。Go 类型嵌入 `workrepr.WorkSummary`，JSON 展平。

### 3.3 `Work` 字段（200 体）

WorkSummary 的字段原样出现（类型、可空、来源见 `workrepr/types.go`）。本面额外：

| 字段 | 类型 | 空 / 可空 | 来源 |
|---|---|---|---|
| `aliases` | `AliasName[]` | 永不 `null`，去掉与 `display_name` 相同的 | catalog `titles[]` |
| `original_language` | BCP-47 \| `null` | catalog 无 olang 为 `null` | catalog `olang`，经现行 `productLocale` 收到 `ja-jp` / `zh-cn` / `zh-tw` / `en-us`；对不上 pattern 的丢成 `null` |
| `content_rating` | `all_ages` \| `r18` | 永不空 | catalog `content_rating`：`r18` → `r18`，其余 → `all_ages`（旧 `age_limit`） |
| `intros` | `CatalogIntro[]` | 永不 `null`；`value` 是 Markdown 源 | catalog intros，经 `workrepr.Intros` |
| `links` | `CatalogLink[]` | 永不 `null`；没有 url 的跳过（`AppendLink`） | catalog `links`（旧 `/link/all`） |
| `external_ids` | `WorkExternalID[]` | 永不 `null` | catalog `refs[]` |
| `covers` | `WorkCover[]` | 永不 `null` | catalog `covers[]` + 票仓（G6 读侧） |
| `screenshots` | `WorkScreenshot[]` | 永不 `null` | catalog `screenshots[]` |
| `companies` | `WorkCompany[]` | 永不 `null` | catalog `labels` / companies |
| `engines` | `Engine[]` | 永不 `null` | catalog engines |
| `series` | `SeriesSummary[]` | 永不 `null` | catalog series，按 GE 摘要补本站计数与样本 |
| `tags` | `WorkTag[]` | 永不 `null`；`include_nsfw=false` 时不含 `is_sexual` 的 | catalog tags，hidden 层仍跳过 |
| `credits` | `WorkCreditGroup[]` | 永不 `null` | catalog credits |
| `characters` | `WorkCharacter[]` | 永不 `null` | catalog characters |
| `resource_types` | `ResourceType[]` | 永不 `null`，去重，按词表顺序 | 本地资源 jsonb `type` 轴（`resourcevocab.TypeKeys`）；与 WorkSummary 的平台 / 语言同一套轴，不读标量 `type` |
| `favorite_count` | int ≥0 | | catalog popularity `source=nextmoe metric=favorites` |
| `is_resource_publish_banned` | bool | 无本地行是 `false` | 本地 `galgame.resource_publish_banned` |
| `creator` | `UserRef \| null` | 无 `creator_user_id`、或不可渲染，为 `null` | 本地 `creator_user_id` → `userclient`；失败 503 |
| `contributors` | `UserRef[]` | 永不 `null`；不可渲染的丢掉；最多 50 | `galgame_contributor`，现行 `contributorMaxPerGalgame` |
| `external_ratings` | `WorkExternalRating[]` | 永不 `null` | catalog `ratings` 块（VNDB 等，不是本站 GR） |
| `playtimes` | `WorkPlaytimeAggregate[]` | 永不 `null` | catalog playtimes 聚合 |
| `dlsite` | `DlsiteOffer \| null` | 无 DLsite workno 为 `null` | G3 同一 `storelink.Resolver` |
| `created_at` | date-time | | catalog created |
| `updated_at` | date-time | | catalog updated |
| `viewer` | `WorkViewer \| null` | 匿名为 `null` | |

**不下发**：`ratings[]`（改打 GR）、`moved_to`、`status`、`vndb_id`、`intro_text`、HTML `introduction`、`is_on_forum`、`violence`、本地 `galgame.favorite_count`、自由 `refs` map。

`view_count` 是本次 GET **已经 +1** 之后的值（与话题 / 工具 / 题目 / 资源详情相同）。没有本地行则仍是 0，不因此 500。

### 3.4 嵌套形状

**`WorkCover`（`object: "work_cover"`）**

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | DecimalID \| `null` | catalog 封面 id，投票路径用；对不上票仓为 `null` |
| `image` | `Image` | 原图；`sexual` 来自该行（指针解码，见 K-G27） |
| `cover_slot` | 封闭：`main` `pkgfront` `dig` `pkgback` `pkgcontent` `pkgside` `pkgmed` `other` | 旧 `kind`；空 / 未知 → `other` |
| `site` | 与 `CatalogLink.site` 同 pattern | 旧 `source`（vndb、dlsite…）；开放词表 |
| `sort_order` | int 0–9999 | 与现 spec 的 `sort_order` 同 schema（G8） |
| `vote_count` | int ≥0 | 公开票仓；失败时 0（旧 hydrate 降级） |
| `viewer` | `{ has_voted: bool } \| null` | 匿名为 `null`；`has_voted` 与 PollViewer 同型（G8） |

写面仍是 G6 的 `PUT`/`DELETE /galgame/:id/cover/:coverId/vote`。本面只读。缺 `catalog:edit` / 用户面 401 时降到公开计数、`has_voted: false`（旧 `cover_votes.go:25` 的降级，保留）。

**`WorkScreenshot`（`object: "work_screenshot"`）**：`image`（`Image`）、`caption`（string ≤512，没有则空串，自由文本）、`site`（同 `CatalogLink.site`）、`sort_order`（0–9999）。没有票。

**`WorkCompany`（`object: "company"`）** = CompanySummary 的字段（`id`、名字三件、`company_kind`、`logo`、`aliases`、`catalog_work_count`）+ 与 `Company` 同名同型的 `lang`、`links` + `attribution_roles`（封闭 `developer` `publisher` `circle` `brand` 数组，去重，本作署名）。不发 `intros`（页面不画）。官网由客户端从 `links` 里挑 `official_site`。

**`WorkTag`（`object: "tag"`）** = TagSummary 的字段 + `spoiler`（`none` \| `minor` \| `major`，与 `CharacterTrait.spoiler` 同词表）。catalog 整数 0/1/2 映过来。`include_nsfw=false` 时整条成人标签不出现（修 E1 的剥标签；**不** 404 整部作品）。

**`WorkCreditGroup`**：`role_key`（与 GE `CreditRole.role_key` 同 pattern）、`display_name`（catalog `role_name`，自由文本）、`people: WorkCreditPerson[]`。`WorkCreditPerson` = `CreditNameRef` 的字段 + `voiced_characters: string[]`（旧 `characters[]` 原文串，自由文本 ≤512；空数组永不 `null`）。

**`WorkCharacter`（`object: "character"`）** = CharacterRef 的字段（`id`、名字三件）+ `image` / `figure`（`Image \| null`，与 `Character` 同名同型）+ `character_kind`（封闭 `main` `secondary` `appears`）+ `spoiler`（`none`/`minor`/`major`）+ `identity`（string ≤512，没有则空串，自由文本）+ `voices: CreditNameRef[]`。未知 `kind` 的角色丢掉并打一条警告，不 500。

**`WorkExternalID`**：`site`（与 `CatalogLink.site` 同 pattern）、`external_id`（string ≤64，自由文本；catalog 的外部号）。

**`WorkExternalRating`（`object: "work_external_rating"`）**：`site`、`rating_value`（number，该源自己的刻度，自由）、`vote_count`（int ≥0）、`source_rank`（int ≥1 \| `null`）、`buckets: {bucket: number, vote_count}[]`（该源自己的桶：批评空间是 0/10/…/100，其余是分数）、`stats: {mean, stdev, minimum, maximum} \| null`。不用 `score` / `rank` / `average`（G8）。

**`WorkPlaytimeAggregate`**：`site`、`minutes`（int ≥0）、`vote_count`（int ≥0）。catalog 各源的聚合。写面仍是 G6。

**`WorkViewerPlaytime`**：`minutes`（int ≥0）、`play_status`（GR 的 `PlayStatus` \| `null`）。整段在 `viewer.playtime`；没有记录为 `null`。catalog 只读 `"done"`（无 completion）**不能**放进 `play_status`（G8 与 `Rating.play_status` 同枚举，没有 `done`）→ `play_status: null`，分钟仍发。缺 `playtime:read` 与旧详情一样整段 `null`，打 `galgame detail: own playtime unavailable, token lacks playtime:read`（`scope_warn`，一行/分钟）。

**`DlsiteOffer`**：原样复用 G3。

**`Engine` / `SeriesSummary` / `TagSummary` / `CompanySummary` / `CreditNameRef` / `CatalogIntro` / `CatalogLink` / `Image` / `UserRef`**：原样复用。

### 3.5 `WorkViewer`（详情的 `viewer`）

匿名为整个 `viewer: null`。Bearer 的 `can_*` 不含 staff（K2）。

| 字段 | 类型 | 说明 |
|---|---|---|
| `has_liked` | bool | 本地 `galgame_like` |
| `has_favorited` | bool | catalog `MyFoldersContaining`；缺 `folder:read` → `false`，打 `galgame: favourite state unreadable, token lacks folder:read`（旧 `isFavorited`，保留） |
| `playtime` | `WorkViewerPlaytime \| null` | 见上 |
| `can_ban_resource_publish` | bool | cookie 且 `user.Can(perm.GalgameBanResourcePublish)`；Bearer 恒 `false` |

封面票不在这一层（在每张 `WorkCover.viewer`）。收藏的写仍是 G6；本面只给 Header 停掉 `useCan('galgame.ban_resource_publish')`。

### 3.6 `WorkState` 与赞回执

**`WorkState`（`object: "work_state"`）**：`work_id`、`has_liked`、`has_favorited`。照 `/me/topic-states`：请求里每个调用者可读的 id 都回一条（没赞 / 没收藏是 `false`，那是答案不是缺席），保持请求顺序；catalog 不认识或 hidden 的 id 进 `missing`，两种原因不区分。

**`WorkEngagement`（`object: "work_engagement"`）**，PUT/DELETE like 的 200：`work_id`、`like_count`、`viewer: { has_liked }`（required 档，viewer 非 null）。`has_liked` 与详情同名同型。

### 3.7 词表

| 字段 | 取值 |
|---|---|
| `content_rating` | `all_ages` `r18` |
| `cover_slot` | `main` `pkgfront` `dig` `pkgback` `pkgcontent` `pkgside` `pkgmed` `other` |
| `attribution_roles` 元素 | `developer` `publisher` `circle` `brand` |
| `character_kind` | `main` `secondary` `appears` |
| `spoiler`（标签、角色） | `none` `minor` `major` |
| `resource_types` 元素 | `resourcevocab.TypeKeys`（与 G3 `resource_type` 同一封闭词表） |
| `play_status` | 与 GR `PlayStatus` 相同：`wish` `doing` `done_one_route` `done_main` `done_all` `on_hold` `dropped` |
| `original_language` | BCP-47 pattern，与 `lang` 相同；F1 的 `languageTagProperties` 实现时加上这个属性名 |

`site`（外链、封面、截图、外部 id、外部评分、游玩聚合）与 `CatalogLink.site` 同一开放 pattern。未知封闭枚举 → 400 `UNKNOWN_ENUM_VALUE`。缺席 = 不过滤。

### 3.8 NSFW 与可见性（K-G23）

`GET /works/{work_id}` 的 `include_nsfw` 默认 `false`，**只**从 `tags` 里拿掉 `is_sexual` 的条目（现行 `withoutSexualTags`，GE 标签 / 角色详情同一参数名）。作品本身不因 NSFW 404；`WorkRef.is_nsfw` 告诉网页画闸与模糊。封面 / 截图 / 角色图带着 `image.sexual`，网页模糊。

catalog 不认识或 claim `hidden` → `404 NOT_FOUND`。**不查**本地 `published`（G3 O1 / 方案③：第一份资源打在未发布作品上）。本地没有行：本地计数 0 / `false` / `[]`，仍 200。

合并：`404 ENTITY_MERGED`，扩展 `object: "work"`、`current_id`（十进制字符串）。实现把 `CatalogWorkDetail` 改走已有的 `catalogGetRecord`（会社 / 人名 / 角色同一条）。若 catalog 对作品合并仍回普通 404，本面退化为 `NOT_FOUND`（§8 O1）。不发 `Link: rel="canonical"`（GE 同一理由）。网页对 `ENTITY_MERGED` `navigateTo(/galgame/{current_id}, {redirectCode: 301})`。

### 3.9 赞槽（K-G24）

K16：`PUT` 置位、`DELETE` 撤销，都幂等，都 200 `WorkEngagement`。空体。

作品必须存在于 catalog（一次 `CatalogRowsByWorkIDs`，hidden 被 `isRenderable` 丢掉）→ 未知或 hidden 都是 `404 NOT_FOUND`，**不**种本地行。catalog 有、本地无：插入一行只带 `id`（`published` 列默认 `false`，迁移 068），**不** `PublishLocal`。

`INSERT … ON CONFLICT DO NOTHING RETURNING id` / `DELETE … RETURNING id`，只在真的动了行时改 `like_count` 并发分、发消息。置已置、撤未置都是 200 无副作用。`like_count = GREATEST(like_count ± 1, 0)`。

调用者是 `creator_user_id` → `403 SELF_LIKE_FORBIDDEN`（已注册描述「Users cannot like what they wrote themselves.」已够用，不改文案）。`creator_user_id` 空 / 0：闸放行，**不**调萌萌点、**不**写 `liked` 消息（修 E3 给 user 0 发分）。

萌萌点只在行真的变了、且有主人时 ±1，reason `liked`，稳定键（W4 / G3，无 `KeyNonce`）：

| 事件 | 键 | ref |
|---|---|---|
| 点赞 / 取消 | `kungal:liked:galgame_like_{row}` / `kungal:unliked:galgame_like_{row}` | `galgame:<work_id>` |

行有 serial `id`（`galgame_like.id`）；取消再点会插入新行、拿到新 id，所以正好再给一次。发分等提交成功之后经 pusher 异步推。`liked` 消息只在真的插入一行时写，取消不写。`galgame_like.updated` 是 `NOT NULL` 且无默认值：INSERT 必须带上（W4 / G3 同一陷阱；走 GORM 模型 Create 即可）。

`userclient` 失败 → 503（判主人 / 可渲染时）。catalog 失败 → 503。

### 3.10 `/me/work-states`（K-G25）

档位 required。Query `work_ids` 必填，1–100，逗号形（`explode: false`，与 `me/topic-states` 相同；超 100 / 缺席 / 空 / 非正十进制 → `400 INVALID_PARAMETER` `TOO_MANY_ITEMS` / `REQUIRED` / `INVALID_FORMAT`，不静默截断，修 E14）。不分页。去重，保持请求顺序。

可读 = 一次 `CatalogRowsByWorkIDs` 认得出且不是 hidden。否则进 `missing`。

`has_liked`：这些 id 在 `galgame_like` 里该用户有行。`has_favorited`：同一批 id **一次** `MyFolderHoldings`。缺 `folder:read` → 每条 `has_favorited: false`，打旧日志 `galgame: my folders unreadable, token lacks folder:read`（`scope_warn`，一行/分钟），**不** 403。catalog holdings 失败（非 scope）→ **503**（比旧面 200 空收藏更硬；修「静默丢收藏」）。无 token 走不进 required 档。

旧面 `liked[]` 一次拉全表（最大 7,203）不再提供；卡片已经按页传 id。

### 3.11 浏览计数（K-G26）

`GET /works/{work_id}` 同步 `view + 1`，错误上抛 500。用**不带 `updated` 的列更新**（`UPDATE galgame SET view = view + 1 WHERE id = $1`，不走会戳 `updated` 的 GORM `Updates`）。没有本地行：0 行 UPDATE，`view_count` 仍 0，200。源面不存在。列表不加浏览。

### 3.12 实现时按 G8 / G14 / F1 改的名（只增不改，以此为准）

G8 要求全 spec 里同名属性同型（含可空与 format），G14 要求每个字符串带 enum / format / pattern / 自由文本标记。任务书前文的几个名字撞了既有的面，契约里改成下表：

| 前文 | 实际 | 撞了谁 |
|---|---|---|
| 会社署名数组叫 `roles` | `attribution_roles` | 话题 `AccessGrants.roles` 是 `creator`/`moderator`/`admin`/`ren` |
| 外部评分叫 `score` | `rating_value` | Website 的 `score` 是整数，可负，−2000…400 |
| 外部评分叫 `rank` | `source_rank` | 排行条目的 `rank` 是 1–100、不可空 |
| 封面槽叫 `kind` | `cover_slot` | 01 禁用名 |
| 角色类型叫 `kind` | `character_kind` | 同上 |
| `refs` 自由 map | `external_ids[]` | 自由 map 过不了 G14 |
| 外链 / 来源叫 `source` | `site` | 与 GE `CatalogLink.site` 一组；避免和别的 `source` 撞 |
| 年龄轴叫 `age_limit` 取值 `all` | `content_rating`：`all_ages` / `r18` | `all` 不是 F1 词；与编辑轴 `is_nsfw` 分开 |
| 作品原语言叫 `lang` | `original_language` | `lang` 已是会社 / 角色 / 人名的自身语言 |
| 标签 `category` + 整数剧透 | `tag_kind` + `is_sexual` + `spoiler` | GE 已定；`spoiler` 与 `CharacterTrait.spoiler` 同词表 |
| `user` | `creator` | 禁用名 |
| 简介 HTML / `intro_text` | `intros`（`CatalogIntro`） | GE 代码形状是 `locale`/`value`/`is_machine`/`data_source`（GE 契约草稿曾写 `lang`/`source`，以代码为准） |
| 链接数组叫 `links` 且元素是 `{name,link}` | `links: CatalogLink[]` | 角色 / 会社 / 人名的 `links` 已是 `CatalogLink[]` |
| 嵌入 `ratings[]` | 不下发 | 详情体积；GR 已有 `GET /ratings?work_id=` |
| `favorite_count` 用本地列 | 用 catalog nextmoe/favorites，名字仍是 `favorite_count` | 与 `WorkStats.favorite_count` 同 schema（整数 ≥0）；文档写清来源。本地列不下发 |

`resource_platforms` / `resource_languages` / `rating_score` / `rating_count` / `view_count` / `like_count` / `is_published` / `is_nsfw` / `cover` / `banner` / `maker` / `dlsite` / `is_resource_publish_banned` / `aliases` / `has_liked` / `has_favorited` / `has_voted` / `vote_count` / `sort_order` / `viewer` / `work` / `id` 与现 spec 对齐，保持。`resource_types`、`content_rating`、`original_language`、`external_ids`、`attribution_roles`、`cover_slot`、`character_kind`、`playtimes`、`external_ratings`、`can_ban_resource_publish` 现 spec 未占用。

`CatalogIntro` / `CatalogLink` 按 GE **代码**（`locale`+`data_source`，`site`+`url`），不是 GE 契约 §3.3 里已经过时的 `lang`/`source`。

### 3.13 实现时 gate 逼出的改名与修正（2026-09-24，以此为准）

实现后 G7 / G8 / G14 在全 spec 上又撞了几处。下表覆盖 §3.1–§3.12 的同名条目：

| 契约前文 | 实际 | 撞了谁 / 为什么 |
|---|---|---|
| `external_ids: WorkExternalID[]` | `external_refs: WorkExternalRef[]`（`site` + `external_id`） | G7：以 `_ids` 结尾的属性必须是 id 字符串数组 |
| `characters: WorkCharacter[]` | `roster: WorkCharacter[]` | G8：`Credit.characters` 是 `CreditCharacter[]` |
| `WorkExternalRating.stats.minimum` / `maximum` | `lowest` / `highest` | G8：`FieldParams.minimum` / `maximum` 是非空 number |
| 外部评分的数值无下限 | `rating_value` / `bucket` / `mean` / `stdev` / `lowest` / `highest` 都 `minimum: 0` | G14：number 必须有 minimum；四个源的刻度都 ≥ 0 |
| `WorkCover.id: DecimalID \| null` | **非空** `DecimalID`：catalog 详情的封面行 id（`catalog_work_cover.id`，投票路径就收它），缺时退回票仓 id；两边都没有的行丢掉并打警告 | G8：`id` 全 spec 非空 |
| `WorkCover.image` / `WorkScreenshot.image: Image` | `Image \| null`，文档写「在封面 / 截图上永不为 null」 | G8：`image` 全 spec 是 `Image \| null`（抽奖奖品图、角色图）。G1 `resource_updated_at` 同一处理 |
| `viewer.playtime.play_status: PlayStatus \| null`，catalog 的只读 `done` 丢成 `null` | `viewer.playtime.play_state`：GR 的七个值 **加** `done`，`null` = 没有状态 | G8：`play_status` 在评分上非空。换名同时保住了旧页今天就在显示的「已通关（无完成度）」——丢成 `null` 是回退。`done` 只读，G6 写面仍不收（O6 改判） |

另外三处实现事实，契约前文写错或没写：

- **浏览计数也要进日桶。** 旧 `IncrementView` 在列上 +1 之后还调 `viewstats.BumpDaily(galgame_view_daily)`，`view_7d` / `view_30d` 排行靠它（迁移 050）。K-G26 只写了「列更新不碰 `updated`」，第一版实现照字面漏了日桶。现在：UPDATE 命中一行才进日桶（无本地行的作品不留孤儿桶）；两步任一出错 500。
- **v2 的竖版 / 横幅槽位此前永远不带等级。** 论坛的 v2 改写（`client/catalog_v2.go` `imageToSlot`）从 v2 `cover` / `banner` 合成 `cover_slots` 时只抄了 url / 宽高 / thumbhash，`sexual` 丢了，所以 K-G27 的指针解码单做是空转：`WorkRef.cover.sexual` 在生产里恒 `null`。改写现在把 v2 的 `safe`/`suggestive`/`explicit` 映成 0/1/2 写进槽位，`null` 保持缺席。槽位的等级只有 `workrepr` 读，旧面不受影响。**影响面**：所有带 `WorkRef` 的 v1 面（资源、题目、排行、搜索、活动…）的 `cover.sexual` 从恒 `null` 变成真实等级；网页按 `sexual` 模糊的地方要在浏览器里看一遍。
- **`resource_types` 读的是标量 `galgame_resource.type`**，不是 jsonb 轴（资源表只有平台 / 语言是 jsonb）。去重后过 `CompatType` + `TypeKeys`。

## 4. 逐条操作

通用：401 `MISSING_CREDENTIAL` / `INVALID_CREDENTIAL`（required；optional 的坏 Bearer）、403 `ACCOUNT_BANNED`、500 `INTERNAL_ERROR`、503 `SERVICE_UNAVAILABLE`（会话存储 / userclient / catalog）。每个 401 带 `WWW-Authenticate`。v1 全部 `Cache-Control: no-store`。

### 4.1 `getWork` · `GET /works/{work_id}` · optional · 200 `Work`

Query：`include_nsfw`（默认 `false`）。副作用：`view_count + 1`，不刷 `updated_at`（K-G26）。catalog 未知 / hidden → 404。合并 → 404 `ENTITY_MERGED`（`object: "work"`，`current_id`）。不要求本地 `published`。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（`include_nsfw`） |
| 401 | 坏 Bearer |
| 404 | 不存在 / hidden / `ENTITY_MERGED` |
| 503 | userclient / catalog |

### 4.2 `putWorkLike` · `PUT /works/{work_id}/like` · required · 200 `WorkEngagement`

空体。已赞再 PUT 是 200、计数不变。自赞 → `403 SELF_LIKE_FORBIDDEN`。catalog 未知 / hidden → 404，不写本地行。

| 状态 | code |
|---|---|
| 403 | `SELF_LIKE_FORBIDDEN` `ACCOUNT_BANNED` |
| 404 | catalog 未知 / hidden |
| 503 | userclient / catalog / 萌萌点上游 |

### 4.3 `deleteWorkLike` · `DELETE /works/{work_id}/like` · required · 200 `WorkEngagement`

未赞再 DELETE 是 200、计数不变。不写 `liked` 消息。404 同 4.2。

### 4.4 `listMyWorkStates` · `GET /me/work-states` · required · 200 `BatchList<WorkState>`

Query：`work_ids` 必填，1–100，逗号形。`missing` 与 `/me/topic-states` 同义。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（缺席 / 空 / 超过 100 / 非正十进制） |
| 503 | catalog holdings（非 scope）/ catalog 可读性 |

## 5. 预分配

### 5.1 迁移

**无。** G 号段下一个空号是 146，本轨不用。`galgame_like` 已有 serial `id` 与 `(user_id, work_id)` 唯一；`galgame.published` 默认 `false`（068）；`view` 列更新不需要新索引。`galgame_renumber_2026` 删表仍按 g-plan 留给后面某段。

### 5.2 错误码

不新增。复用：

- **`ENTITY_MERGED`**（platform，404，GE 已注册，扩展 `object` + `current_id`）。本轨 `object: "work"`。
- **`SELF_LIKE_FORBIDDEN`**（kungal，403）。描述已是「Users cannot like what they wrote themselves.」，作品自赞直接走，不改文案。

其余：`NOT_FOUND`、`PERMISSION_REQUIRED`、`ACCOUNT_BANNED`、`SERVICE_UNAVAILABLE`、`INVALID_PARAMETER`（`REQUIRED` / `INVALID_FORMAT` / `TOO_MANY_ITEMS` / `TOO_FEW_ITEMS`）、`VALIDATION_FAILED`。

### 5.3 K 决定

| 编号 | 决定 |
|---|---|
| **K-G22** | `Work` 是 `WorkRef` 的严格超集，且 `WorkRef` ⊂ `WorkSummary` ⊂ `Work`，重叠字段同名同型。G3 薄面 `getWork` 的 200 从 `WorkRef` 扩成 `Work`。WorkSummary 有而 WorkRef 没有的字段保持；WorkRef 的字段 WorkSummary 一个不缺 |
| **K-G23** | `include_nsfw` 默认 `false` 只剥成人标签，不 404 作品。`is_nsfw` 给网页。hidden / 未知 404；不查本地 `published`。合并见 K-G28 |
| **K-G24** | 赞槽见 §3.9。catalog 存在性、缺本地行按 id 插入不碰 `published`、自赞 `SELF_LIKE_FORBIDDEN`、无主不发分、稳定键、消息只在真赞、`like_count` ≥ 0 |
| **K-G25** | `/me/work-states` 见 §3.10。`has_liked` 本地；`has_favorited` 一次 holdings；缺 scope → `false` + 旧日志；catalog 失败 503；不可读进 `missing` |
| **K-G26** | 详情 GET 同步 `view + 1`，不写 `updated`，错误 500。无本地行 0 行 UPDATE 仍 200 |
| **K-G27** | 封面槽与详情封面 / 截图行的 `sexual` 按指针解码；竖版等级进 `WorkRef.cover.sexual`。`WorkRefOf` 委托 `workrepr.Ref` |
| **K-G28** | 合并作品 `404 ENTITY_MERGED` `object: "work"` + `current_id`。实现走 `catalogGetRecord`。用户裁决不重定向（由客户端 301） |
| **K-G29** | G6 的读侧留在 `Work`：每张封面的 `vote_count` / `viewer.has_voted`、`playtimes[]`、`viewer.playtime`。G6 只迁写 |

权限不新增。`galgame.ban_resource_publish` 沿用，经 `viewer.can_ban_resource_publish` 下发。

## 6. 网页

切到生成的类型化客户端；手写 `GalgameDetail` 改成生成物别名或删除。`legacy-fetch-baseline` 按删掉的调用点下调。`include_nsfw` 跟内容姿态。`ENTITY_MERGED` 的 zh-CN 已在 `problem.json`。按钮显隐读 `viewer.can_*`。

| 文件 | 改什么 |
|---|---|
| `pages/galgame/[id]/index.vue` | `GET /works/{work_id}?include_nsfw=`；`moved_to` 已不在，改认 `ENTITY_MERGED` → 301 `/galgame/{current_id}`；SEO：`is_published` 取代 `indexed`，`is_nsfw` 取代 `content_limit`；JSON-LD 名字读三件套，简介从 `intros[].value` 抽纯文本，作者读 `creator` |
| `PreviewModal.vue` | 同一 GET；横幅读 `banner`；名字三件套；`intros` 当 Markdown 喂编辑器 / 渲染器（不再 `KunContent` 吃 HTML）；分级读 `content_rating` |
| `galgame-edit/review/Detail.vue` | 同一 GET，id→name 表改读 `tags` / `companies` / `engines` / `series` / `characters` / `credits` |
| `server/utils/kunOgCard.ts` | `buildGalgame` 改生成客户端 `GET /works/{id}`；NSFW / 未发布不出卡（`is_nsfw` / `!is_published`）；封面 `cover.url`；会社 `companies[0].display_name`；徽章读 `resource_types` + `content_rating==="r18"` |
| `Galgame.vue` | 注入 `Work`；雷达 / 短评改 `GET /ratings?work_id=&include_nsfw=true&limit=100`（生产最多 62，一页够）；`handleRatingCreated` 改为刷新该列表 |
| `Header.vue` | 名字三件套 + `aliases`；芯片读 `resource_types` / `resource_languages` / `resource_platforms`；赞 / 收藏读 `viewer`；禁发读 `viewer.can_ban_resource_publish` 与 `is_resource_publish_banned`（停 `useCan`）；DLsite 读 `dlsite`；`is_nsfw` 取代 `content_limit` |
| `Like.vue` | `PUT` / `DELETE …/like`；读回 `like_count` / `viewer.has_liked`；自赞走 `SELF_LIKE_FORBIDDEN` |
| `Favorite.vue` | 读侧 `viewer.has_favorited` / `favorite_count`；写仍 G6 |
| `Covers.vue` | 读 `covers[].image` / `cover_slot` / `site` / `vote_count` / `viewer.has_voted`；写仍旧投票路由直到 G6 |
| `Gallery.vue` | 读 `screenshots[].image.sexual` 做模糊；`violence` 过滤器删掉（A5）；按 `site` 分组 |
| `Introduction.vue` | `intros`；tab 键是 `locale`；`value` 走 Markdown 渲染 |
| `link/Link.vue` | **不再**打 `/link/all`；读 `links` + `external_ids`（身份链仍用评分 map 的 url 模板） |
| `Info.vue` + `detail/Official.vue` | `companies` / `engines` / `series`；分级 `content_rating`；原语言 `original_language`；发售 `release_date` + `release_date_precision`（null 即待定，不再有 `release_date_tba`） |
| `Tag.vue` | `tag_kind` + `is_sexual` + `spoiler`；成人盒只在 `include_nsfw=true` 的载荷里有东西 |
| `Staff.vue` | `credits`；`display_name` 当职务标签；`CreditNameRef.display_name` 当人名 |
| `character/Panel.vue` | `characters`；`character_kind` / `spoiler`；图读 `image` / `figure` |
| `series/Panel.vue` | 可直接画 `series: SeriesSummary[]`，去掉逐个 `GET /series/{id}` |
| `contributor/Container.vue` | `contributors` + `creator` |
| `DlsitePurchase.vue` | `dlsite.purchase_url` / `coupon_url` / `campaign_name` |
| `header/Playtime.vue` + `PlaytimeModal.vue` | 读 `playtimes` / `viewer.playtime`；写仍 G6 |
| `header/RatingStrip.vue` + `header/rating/*` | 本站分读 `rating_score` / `rating_count`；外部分读 `external_ratings`；本站短评列表改 GR |
| `rating/radar/Card.vue` + `rating/Row.vue` | 数据源改 GR 列表 |
| `useMyGalgameInteractions.ts` + `card/Card.vue` | `GET /me/work-states?work_ids=` 逗号形，读 `has_liked` / `has_favorited`；`missing` 当未点亮 |
| `docs/proj/app-direct-api.md` | 详情改 `GET /api/v1/works/{work_id}`；赞改槽位（前瞻，App 现网不打） |

## 7. 变异题（先于实现提交）

种子：一部 catalog 作品带成人标签 + 普通标签、竖版封面 `sexual` 缺席、另一张封面 `sexual=0`；一部 hidden；一个未知 id；一个已合并 id（若夹具能模拟 `ENTITY_MERGED`）；本地无行的 catalog 作品；`creator_user_id` 空的本地行；作者即调用者的行；两个不可渲染贡献者 + 一个可渲染；同一用户对同一作品已赞；`work_ids` 里一个 hidden、一个可读未赞、一个可读已赞。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | `include_nsfw=false` 时仍把 `is_sexual` 标签放进 `tags`（旧详情对 SFW 漏闸的反向：本面要剥） | 默认 GET 的 `tags[].is_sexual` 全是 false；同一 id 带 `include_nsfw=true` 能看到成人标签 |
| 2 | 贡献者列表不再丢不可渲染用户（旧 `hydrateDetailUsers`） | 响应 `contributors` 不含该 id；`creator` 不可渲染时是 `null`，详情仍 200 |
| 3 | 去掉「不能给自己赞」的检查 | 作者 PUT like → 403 `SELF_LIKE_FORBIDDEN`，无 like 行，萌萌点 0 次 |
| 4 | `creator_user_id` 空时仍 `AdjustMoemoepoint(0, ±1)`（旧行为） | 无主作品 PUT like 插入行、计数 +1、萌萌点客户端 0 次调用、无 `liked` 消息 |
| 5 | PUT like 改回切换（旧 `ToggleLike`） | 已赞再 PUT → 200，`has_liked: true`，行数不变 |
| 6 | DELETE like 对未赞回 400 | 未赞再 DELETE → 200，`has_liked: false`，计数不变 |
| 7 | `like_count - 1` 不套 `GREATEST` | 从 0 再删 → 列仍 ≥ 0 |
| 8 | `/me/work-states` 把 hidden / 未知 id 回 `has_liked: false` 而不是 `missing` | 该 id 在 `missing`，不在 `items` |
| 9 | `folder:read` 不足时 403 | holdings 回 `ErrInsufficientScope` → 200，条目 `has_favorited: false`，日志含 `token lacks folder:read` |
| 10 | 详情 GET 走 GORM `Updates` 以致写入 `updated`（旧资源详情同类） | `updated` 列与请求前相同；`view` +1 |
| 11 | 把 `Work.id` 改名为 `work_id`，或把 `is_nsfw` 改成字符串 | WorkRef ⊂ Work 的断言红：每个 WorkRef JSON 键在 Work 里同名同型 |
| 12 | 未知 catalog id 的 PUT like 仍 `Ensure` 本地行（旧行为） | 404，`galgame` 表无新行 |
| 13 | `catCoverSlot.Sexual` / 封面行仍按 `int` 解码 | 未判定封面 `image.sexual === null`；判为 0 的 `=== "safe"`；`Work.cover.sexual` 与竖版槽位一致 |
| 14 | 合并作品回 200 或普通 404 且没有 `current_id`（在 catalog 能区分的夹具上） | 404 `ENTITY_MERGED`，`object === "work"`，`current_id` 是幸存者 |

## 8. 开放问题

| # | 事实 | 裁决 |
|---|---|---|
| O1 | ~~catalog 对作品是否发 `ENTITY_MERGED`~~ **已核实（infra 源码）**：`/v2/catalog/works/{id}` 对合并 id 答 `404 ENTITY_MERGED` + `object: "work"` + `current_id`（`apiv2/handler/catalog.go:101` → `mergedOrNotFound`，路由描述 "Merged ids are 404 ENTITY_MERGED with Link rel=canonical"）；r18 不带 `nsfw=true` 也是 404，论坛的 `openPopulation` 已带 | 实现把 `CatalogWorkDetail` 从 `CatalogGet` 换成 `catalogGetRecord`（K-G28）；部署后仍用生产 `galgame_merge_discarded` 的一个 id 匿名 GET 复核 |
| O2 | 作品页今天把评分嵌在详情里，**没有**打 `GET /ratings?work_id=` | 本轨不嵌。网页改 GR；`limit=100` 覆盖生产最大 62。雷达 / 本地面板跟走 |
| O3 | `WorkSummary` 没有 `resource_types` / `favorite_count` / `viewer` | 只加在 `Work`。G5 列表不画这些 |
| O4 | 嵌入完整 `SeriesSummary`（含最多 5 个 `sample_works`）比侧栏需要的多，可能 N+1 | 接受。一部作品通常 1–2 个系列；`sample_works` 可空数组。系列面板不再逐个 GET |
| O5 | 截图的 `violence` 整数今天参与 Gallery 过滤 | **不下发**（01 图片 A5）。网页只按 `image.sexual` 与内容姿态模糊 |
| O6 | catalog 只读 play state `"done"` 进不了 GR 的 `play_status` 枚举 | **改判（§3.13）**：字段叫 `viewer.playtime.play_state`，枚举 = GR 七值 + `done`；分钟照发。G6 写面继续不收 `done` |
| O7 | `Work.favorite_count`（catalog 指标）与活动 `WorkStats.favorite_count`（本地列）同名同 schema、不同计数 | 接受 G8。字段文档写清来源。本地列本面不下发 |
| O8 | 封面 / 截图行的 `Sexual` 与槽位一样是 `int` | 与槽位同一指针修复（K-G27），不另开范围 |
| O9 | 标签 `tag_kind` 只有 `content`/`meta`；旧 `technical` 若 catalog 仍发 | 按 `meta` 收下并打警告，不 500。实现时对生产 tag kind 再扫一眼 |
| O10 | `sort_order` G8 上限 9999 | 封面 / 截图用同一上限。超出的丢掉并打警告 |
| O11 | `GET /me/work-states` 与 catalog 用户面 `/v2/me/work-states`（游玩状态）同名不同物 | 论坛这条在 `/api/v1`，形状是 like/favorite。G6 迁游玩时长时不要复用这个路径 |
| O12 | g-plan 把 G4 写成 6 条（含 `DELETE /galgame/:id`） | 以本契约为准：5 条，DELETE 归 G7。`Mine.vue` 的撤回仍打旧 DELETE |
| O13 | Flutter 前瞻文档仍写 `/api/galgame/:gid` | 实现 PR 改 `app-direct-api.md` |
| O14 | 编辑资料按钮今天对所有登录者显示，没有 `can_edit` | 本轨不加。资料编辑是 G7 |
| O15 | `WorkRefOf` 用 hash 建封面，`workrepr.Ref` 从 URL 抽 hash | 委托之后只剩 `workrepr.Ref`；两边测试都要还绿 |

---

删旧路由时：`GET /galgame/:id`、`PUT /galgame/:id/like`、`GET /galgame/:id/link/all`、`GET /galgame/interactions/mine`、`GET /galgame/drafts`，以及只被它们用的 DTO / proxy handler / drafts handler。`DELETE /galgame/:id` 留下。`legacy_route_baseline` −5。`rg` 证明零调用方，含 `apps/web/server/` 与 `../kungal-apps`。
