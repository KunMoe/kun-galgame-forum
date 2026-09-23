# 普查 · Galgame 域 G1（核心读面、点赞、月历、外链、草稿、封面投票、游玩时长、收藏夹）

> 只读普查。写于 2026-09-22，基于 `api-v1/g-galgame` @ `ef0c9fb3`（= master）。没有连库、没有起服务，全部结论来自代码。
> 需要生产库才能确认的数字集中在 §12。

## 0. 范围与总览

`routes.golden` 里属于本段的路由共 **23 条**（15 GET / 1 POST / 4 PUT / 1 PATCH / 2 DELETE）。其中 22 条挂在 `/api/galgame/**`，1 条挂在 `/api/user/:id/collections`（用户域路径、本域实现；用户域普查把它标成寄挂）。

| # | 方法 | 路径 | handler | 档位 | 拥有者 |
|---|---|---|---|---|---|
| 1 | GET | `/api/galgame` | `galgame/handler/galgame_handler.go:40` | public | 本域 |
| 2 | GET | `/api/galgame/:gid` | `galgame_handler.go:25` | optional | 本域 |
| 3 | PUT | `/api/galgame/:gid/like` | `galgame_handler.go:57` | required | 本域 |
| 4 | GET | `/api/galgame/interactions/mine` | `galgame_handler.go:74` | required | 本域 |
| 5 | GET | `/api/galgame/calendar` | `calendar_handler.go:19` | public | 本域 |
| 6 | GET | `/api/galgame/calendar/today` | `calendar_handler.go:27` | public | 本域 |
| 7 | GET | `/api/galgame/calendar/pending` | `calendar_handler.go:35` | public | 本域 |
| 8 | GET | `/api/galgame/calendar/tba` | `calendar_handler.go:43` | public | 本域 |
| 9 | GET | `/api/galgame/calendar/upcoming` | `calendar_handler.go:51` | public | 本域 |
| 10 | GET | `/api/galgame/collected-calendar` | `galgame_handler.go:53` | public | 本域 |
| 11 | GET | `/api/galgame/:gid/link/all` | `galgame_proxy_handler.go:18` | optional | 本域 |
| 12 | GET | `/api/galgame/drafts` | `drafts_handler.go:18` | public | 本域 |
| 13 | PUT | `/api/galgame/:gid/cover/:coverId/vote` | `cover_vote_handler.go:29` | required | 本域 |
| 14 | DELETE | `/api/galgame/:gid/cover/:coverId/vote` | `cover_vote_handler.go:33` | required | 本域 |
| 15 | PUT | `/api/galgame/:gid/playtime` | `playtime_handler.go:34` | required | 本域 |
| 16 | GET | `/api/galgame/playtime/mine` | `playtime_handler.go:67` | required | 本域 |
| 17 | POST | `/api/galgame/collection` | `collection_handler.go:25` | required | 本域 |
| 18 | PATCH | `/api/galgame/collection/:cid` | `collection_handler.go:45` | required | 本域 |
| 19 | DELETE | `/api/galgame/collection/:cid` | `collection_handler.go:69` | required | 本域 |
| 20 | GET | `/api/galgame/collection/:cid` | `collection_handler.go:89` | optional | 本域 |
| 21 | GET | `/api/galgame/:gid/collections/mine` | `collection_handler.go:104` | required | 本域 |
| 22 | PUT | `/api/galgame/:gid/collections` | `collection_handler.go:124` | required | 本域 |
| 23 | GET | `/api/user/:id/collections` | `collection_handler.go:147` | optional | 本域 |

路由注册在 `internal/app/router.go:144`（`GET /galgame`）、`:161–167`（月历 / collected-calendar / drafts）、`:224–229`（links / detail / collection 读面 / user collections）、`:290–303`（interactions / playtime / like / cover vote / collection 写面）。字面量 `/galgame/<segment>` 必须写在 `/galgame/:gid` 前面（`router.go:145-147`）；`/galgame/collection/:cid` 是三段路径，不会被两段的 `:gid` 吃掉。

整组 `/api` 挂了 `NamePreference` 与 `ContentStance`（`router.go:47-48`）。`routes.golden` 里本段每条都带着这两层。鉴权档位来自 golden：public 没有 `OptionalAuth`；optional 有 `OptionalAuth`；required 是 `OptionalAuth → Auth`。本段 **零个** `RequirePermission` / `RequireModerator`。收藏夹写面的 staff 能力走 `user.Can(perm.CollectionEditAny)` / `user.Can(perm.CollectionDeleteAny)`（`collection_handler.go:63/83`），Bearer 恒 false。

**信封**：全部走旧信封 `pkg/response`。成功 `200 {code:0, message:"成功", data:…}`。`OKMessage` 只有 `code`/`message`（点赞、收藏夹更新/删除/成员资格）。`Paginated` 是 `data:{items,total}`，本段只有 `GET /user/:id/collections` 用它。错误 `pkg/response.Error` → `{code, message}` + HTTP 状态。本段出现的体码：`205`（401）、`233`（400/403/404/422/429/500/503 混用）、`235`（403 缺 OAuth scope）。**零个** problem+json、零个 `object` 判别字段、零个字符串 id、零个 `viewer` 块。点赞状态、收藏状态、封面是否投过票、自己的游玩时长，全部跟资源字段平铺。

**身份**：路径 `:gid` / 字段 `id` / `galgame_id` 是论坛自己的 galgame id。catalog 的作品 id 是另一套整数，10,289 个数值撞车（`catalog_face.go:701-703`）。封面投票与游玩时长写面走 `CatalogWorkIDs`（`cover_vote_handler.go:78`、`playtime.go:113`）。收藏夹成员资格、`isFavorited`、`GetMyInteractions` 的 `work_ids` 把论坛 gid 直接当成 catalog `work_id`。v1 已存在的 `GET /api/v1/galgames/{galgame_id}/moyu-patches`（`apiv1/register.go:17`）走 `CatalogWorkIDForGID`（`moyu.go:97`），映射是对的。

**SFW**：本段读面的闸是 `utils.IsSFW`（`pkg/utils/settings.go:42`）。已登录走 `ContentStance` 的账号姿态（`middleware/content_stance.go:19`，`content.Stance.AllowsNSFW`，`hide` 为 SFW）；匿名走 cookie `KUNGalgameSettings.showKUNGalgameContentLimit`，取值不是 `nsfw`/`all` 就算 SFW。`IsSFW` **不读** `X-Kungal-Nsfw`。`NamePreference` 读同一颗 cookie 的 `showKUNGalgamePreferOriginalName`（`settings.go:52`），经 `c.Context()` 进 catalog 选名。v1 这两项都要换成显式查询参数。

catalog 查询的年龄轴始终打开（`openPopulation` 设 `nsfw=true`，`catalog_face.go:50`）。编辑轴才是 SFW 闸：`ApplyWorksGate` 给 SFW 读者加 `content_limit=sfw`（`catalog_face.go:72`）。本地 SQL 闸是 `galgame.content_limit IS NULL OR = 'sfw'`（`list_repo.go:362`）。镜像写入走 `contentLimitOf`（`catalog_wire.go:271`）：claim 没有合法 `content_limit` 时 **回落到 `content_rating`（r18→nsfw）**。这是用年龄轴当 SFW 闸。

**引擎钉死**：`GET /api/galgame` 只看 `library` 布尔。`library=true` 走 catalog（`galgame_library.go:27`）；缺省走本地 SQL（`galgame_service.go:369-373`）。空过滤不会把浏览页切到 catalog。测试钉死这件事（`galgame_library_test.go:34`）。

**v1 现状**：`/api/v1` 下 galgame 族目前只有 `GET /api/v1/galgames/{galgame_id}/moyu-patches`（`routes.golden:185`）。本段 23 条旧路由一条都没迁。

---

## 1. `GET /api/galgame`（浏览列表，页码集合）

链路：`GalgameHandler.GetList`（`galgame_handler.go:40`）→ `ParseQueryAndValidate` 进 `dto.GalgameListRequest`（`galgame_dto.go:10`）→ `GalgameService.GetList`（`galgame_service.go:312`）。`library=true` 进 `catalogLibrary`（`galgame_library.go:27`）；否则 `hydrateListCards` → `GalgameListRepository.ListIDs`（`list_repo.go:55`）→ `HydrateCardsByIDs`（`galgame_service.go:416`）。

### 1.1 调用方

搜 `useKunFetch`/`kunFetch` 的 `` `/galgame` ``、`'/galgame'`、`/galgame?indexed`，以及 `apps/web/server/**`、`apps/web/shared/**`、`/home/kun/Desktop/code/website/kungal-apps/**/*.dart`。

- `apps/web/app/components/galgame/card/Container.vue:28` — `/galgame` 资源浏览页（`pages/galgame/index.vue:11` 挂它）。不带 `library`。
- `apps/web/app/components/galgame/library/Container.vue:10` — `/gallib`（`pages/gallib.vue:11`）带 `library: 'true'`。
- `apps/web/server/utils/kunSitemapSources.ts:195` — Nitro sitemap，`GET /api/galgame?indexed=true&page=N&limit=50`，带 SFW cookie（`:51-53`、`:109`）。

Flutter：`kungal-apps/apps/kungal/lib/features/galgame/presentation/galgame_screen.dart:8` 是占位 `Text('galgame')`；`packages/kungal_api` 里零个 galgame 路径。App 目前不打这条。

### 1.2 请求

`dto.GalgameListRequest`（`galgame_dto.go:10-32`），全部 query：

| 字段 | 校验 | 实际语义 |
|---|---|---|
| `page` | `min=1` | 零值过不了，**事实必填**。缺省 → `400/233 "请求参数验证失败"` |
| `limit` | `min=1,max=50` | 同上必填。浏览页写死 24（`useGalgameFilters.ts:89`）；sitemap 50 |
| `type` | 无 `oneof` | 资源类型。`""`/`"all"` 不过滤；其它值 `gr.type = ?`（`list_repo.go:133`）。未知值当精确匹配，命中 0 行 |
| `language` | 无 | 资源语言。同 `all` 规则。`gr.language = ?`（`:136`） |
| `platform` | 无 | 资源平台。`gr.platform = ?`（`:139`） |
| `game_type` | `omitempty,oneof=all ba_saku plot moe daily uncategorized` | 评分里的作品类型。未知 → 400。`all`/空不过滤。`uncategorized` = 没有任何非空 `galgame_type` 的评分（`:339`）。其它 = `galgame_type @> ["<token>"]`（`:342`） |
| `sort_field` | 无 `oneof` | 见下。未知值**静默回落** |
| `sort_order` | `omitempty,oneof=asc desc` | 空 → 服务层填 `"desc"`（`galgame_service.go:318`）。非法 → 400。SQL 里大小写不敏感，非 `asc` 一律 DESC（`list_repo.go:375`） |
| `include_providers` | 无 | CSV。`gr.provider &&` 包含其中任一（`:142`） |
| `exclude_only_providers` | 无 | CSV。资源必须与「全表减去这些」的补集相交（`:145`）。词表写死在 `allProviders`（`list_repo.go:28`）：`baidu aliyun quark pan123 tianyiyun caiyun xunlei uc lanzou other` |
| `released_from` / `released_to` | 服务层解析 | 只接受 `YYYY` 或 `YYYY-MM`（`pkg/utils/release_date.go:21-27`）。`YYYY-MM-DD` → `400/233 "非法的发售日期下限/上限 …"` |
| `released_months` | 服务层 | 逗号分隔 1–12。非法 token → 400 |
| `collected_from` / `collected_to` / `collected_months` | 同日期梯子 | 滤的是 **`galgame.created`（本站首次落行）**，不是 catalog 发售日（`list_repo.go:319`）。上界是「次日 0 点之前」 |
| `min_rating_count` | `omitempty,min=0` | 贝叶斯聚合的评分人数下限 |
| `min_rating` | `omitempty,min=0,max=10` | 贝叶斯分下限 |
| `show_no_resource` | bool | 浏览页**不发**。为真且 `indexed` 为假时：去掉 `published` 闸，也去掉「必须有资源」 |
| `indexed` | bool | sitemap 发 `true`。为真时去掉「必须有资源」，**仍要求 `published`** |
| `library` | bool | **唯一的引擎开关**。`true`/`1` 都能绑上（`list_query_test.go:16`） |

前端过滤条（`useGalgameFilters.ts:41`）的 URL 键是 camelCase（`sortField`、`gameType`、`releasedFrom`…），请求时再映射成 snake_case（`card/Container.vue:30-48`）。网页词表：资源类型 `game/patch/collection/voice/image/ai/video/others`（`constants/galgame.ts:52`）；语言 `ja-jp/en-us/zh-cn/zh-tw/others`；平台 `windows/mac/linux/emulator/app/others`；`game_type` 另加 `ba_saku/plot/moe/daily/uncategorized`（`card/Nav.vue:89`）。后端 `type`/`language`/`platform` 没有封闭枚举，前端多出来的校验只挡自己。

**本地 SQL 排序**（`listSortColumn`，`list_repo.go:36`）：

| `sort_field` | 列 | tie-breaker |
|---|---|---|
| `created` | `g.created` | `, g.id DESC`（`:80`） |
| `view` | `g.view` | 同 |
| `view_1d` | 当日 `galgame_view_daily` 子查询 | 同 |
| `view_7d` / `view_30d` | 对应列 | 同 |
| `release_date` | `g.release_date NULLS LAST` | 同 |
| `rating` | 贝叶斯分，无评分的排最后 | 同 |
| **其它（含 `time`、`popularity`）** | `g.resource_update_time` | 同 |

浏览页默认 `sortField=time`（`useGalgameFilters.ts:41`），落到 `resource_update_time`。资料库页默认 `popularity` 且 `library=true`。把资料库的 `sortField=popularity` 拿到 `/galgame` 上，本地引擎会按资源更新时间排，接口 200。

本地排序 **有 `g.id DESC` 决胜键**（`list_repo.go:80` 的注释写着并列是常态）。catalog 引擎没有论坛侧的 id 决胜。

**catalog 引擎排序**（`catalogLibrarySort`，`galgame_library.go:78`）：

| `sort_field` | 发给 catalog 的 `sort=` |
|---|---|
| `release_date` + `asc` | `released_asc` |
| `release_date` 其它 | `released_desc` |
| `time` / `updated` | `updated` |
| `relevance` | `relevance` |
| **default（含 `popularity`、`view`、`rating`）** | `popularity` |

catalog 引擎还把 `released_from`/`released_to` 译成 `released_after`/`released_before`（`galgame_library.go:50`）。`type`/`language`/`platform`/`game_type`/网盘/收录日/评分门槛/`indexed`/`show_no_resource` **全部丢掉**。引擎由 `library` 钉死，过滤空不空不会切引擎；`library=true` 时那些过滤只是被忽略。

`HasResourcePredicate`（`model/galgame_list.go:28`）为真时走 JOIN `galgame_resource` 的 DISTINCT 子查询。`type`/`language`/`platform` 为 `all` 或空不算谓词。

`applyPublished`（`list_repo.go:352`）：`show_no_resource && !indexed` 时放开；否则 `WHERE g.published`。`published` 自迁移 078 起是粘性 SEO 旗（`078_galgame_published_is_indexable.up.sql:4`），隐藏/封禁仍会在应用层撤掉。

`applyContentLimit`（`list_repo.go:362`）：SFW 时 `content_limit IS NULL OR = 'sfw'`。NULL 必须放行，否则镜像填上之前列表是空的（`list_repo_test.go:18`）。真正藏作品的是 hydrate 时 catalog 的闸。

### 1.3 响应

`dto.GalgameListPage`（`galgame_dto.go:94`）：`{galgames: GalgameListCard[], total}`。键名是 `galgames` 不是 `items`。页码信封，无 `total_relation`，无深度上限。

`GalgameListCard`（`galgame_dto.go:65`）：`id`（论坛 gid，JSON number）、`name`/`name_original`、`user`（`UserBrief`）、`content_limit`、`view`、`like_count`、`rating`/`rating_count`（贝叶斯，C=10，全表均值为先验，`list_repo.go:228`）、`resource_update_time`（RFC3339 或 `""`）、`is_on_forum`、`platform[]`/`language[]`（来自本地资源元数据）、`release_date`/`release_date_tba`、banner/portrait 的 hash/url/宽高/thumbhash、`company`。

`is_on_forum` 在列表里等于 `galgame.published`（`galgame_service.go:456`）。详情里同名字段等于「本地行存在」（`galgame_service.go:240`，`local.ID != 0`），详情另有 `indexed = local.Published`。同一键两义。

`user` 是冻结的 `creator_user_id`（`frozenCreatorBrief`，`galgame_mapper.go:35`）。`userclient.Hydrate`（`pkg/userclient/hydrate.go:32`）丢掉 `Users` 的 error，缺失 id 填 `Placeholder`（`Name: "已注销用户"`，`Status` 零值）。**列表卡片不调 `IsRenderable`**。封禁作者的名字照样出现。

hydrate 时 `briefMap` 缺的 id `continue` 掉（`galgame_service.go:442`）。`total` 来自 SQL `COUNT(*)` 或 catalog `res.Total`。SFW 镜像滞后、catalog 超时、gid 解析失败，都会让某一页的 `len(galgames)` 小于 `limit`，而 `total` 仍按未过滤的 id 算。分页器画错。

catalog 引擎：先 `CatalogItemRenderable`（claim 不是 `hidden`，`catalog_wire.go:334`），再 `CatalogItemGID` 取论坛 gid，再 `HydrateCardsByIDs`。`total` 是 catalog 的总数，含被 `continue` 掉的行。

### 1.4 错误

- query 绑定失败 → `400/233 "查询参数格式错误"`
- 校验失败（缺 page/limit、非法 `game_type`/`sort_order`、`min_rating>10`、`limit>50`）→ `400/233` 带中文校验句
- 日期/月份解析失败 → `400/233` 带 `err.Error()`（中文，「非法的发售日期下限…」）
- catalog 未配置（仅 library 分支）→ `500/233 "Galgame 目录未启用"`（`galgame_library.go:34`）
- catalog / hydrate 失败 → 透传 `AppError`（通常 `500/233`）

匿名可达。无登录、无 capability。

### 1.5 可见性

本地引擎：`published` +（默认）存在资源 + SFW 的 `content_limit` 镜像。隐藏/封禁靠应用层把 `published` 扳假（`list_repo.go:349` 注释）。catalog 引擎：`ApplyWorksGate` + `CatalogItemRenderable`。详情页（§2）对 SFW 读者 **不** 用同一道整页闸。

### 1.6 测试

- `handler/list_query_test.go` — `library` 绑定
- `repository/list_repo_test.go` — SFW NULL 放行、收录日过滤、collected-calendar 闸、`OrderRestrictIDs`（实体页路径，本路由不走 `RestrictIDs`）
- `service/galgame_library_test.go` — 无 `library` 不打 catalog；`library=true` 不发 `claim_state`，SFW 发 `content_limit=sfw` 且 `nsfw=true`

无 handler 级契约测试。

---

## 2. 详情、点赞、我的互动

### 2.1 `GET /api/galgame/:gid`

链路：`GetDetail`（`galgame_handler.go:25`）→ `strconv.Atoi(:gid)` → `GalgameService.GetDetail`（`galgame_service.go:192`）→ `CatalogWorkDetail`（`catalog_face.go:502`）→ `CatalogDetailToFull` → 本地计数 / 点赞 / 收藏 / 资源轴 / 评分 / 封面票 / 自己的游玩时长。

**调用方**（搜 `` `/galgame/${ ``、`fetchKunApi(\`/galgame/`）：

- `apps/web/app/pages/galgame/[gid]/index.vue:14`（另带无用 query `galgame_id`，handler 只读路径）
- `apps/web/app/components/galgame/PreviewModal.vue:19`
- `apps/web/app/components/galgame-edit/review/Detail.vue:75`
- `apps/web/server/utils/kunOgCard.ts:104`（Nitro OG 卡）

Flutter 零调用。`apps/web/shared/**` 只有类型。

**请求**：路径 `:gid`。非法 → `400/233 "无效的 Galgame ID"`。无 query 被消费。`optionalUID` + `GetAccessToken` + `IsSFW`。

**找不到**：catalog 没有对应 work → 查 `galgame_redirect`（`galgame_service.go:207`）。有迁移目标则 `200 {moved_to: newGID}`（前端 301，`[gid]/index.vue:20`）；否则 `404/233 "未找到该 Galgame"`。claim `state=hidden` 在 `CatalogWorkDetail` 里当成找不到（`catalog_face.go:531`）。

catalog 详情用 `openPopulation`（`nsfw=true`），**不加 `content_limit`**（`catalog_face.go:519`）。SFW 读者直接打开 NSFW 作品的 URL 会拿到整页。SFW 只做一件事：`withoutSexualTags` 去掉 `category=="sexual"` 的标签（`galgame_service.go:253`，`galgame_mapper.go:264`）。封面/截图的 `sexual` 整型仍下发。

成功后 `go IncrementView`（`galgame_service.go:215`，`galgame_repo.go:84`）。没有本地行就是 0 行 UPDATE。浏览量与详情是否 SFW、是否登录无关。

响应 `dto.GalgameDetail`（`galgame_dto.go:245`），字段很多，要点：

- `id` 论坛 gid；`moved_to` 仅重定向体
- `vndb_id`、`user`（作者 `UserBrief`）、`name`/`name_original`
- `introduction[]`：catalog 实际有的语言，Markdown **已渲染 HTML**（`renderIntros`，`galgame_mapper.go:52`）；`intro_text` 是第一条的 **源文**（`:61`）
- `content_limit`、`age_limit`、`original_language`、`status`（int：claim live → 0，其它 → 2，`catalog_wire.go:322`）
- `is_on_forum` = 本地行存在；`indexed` = `published`
- `view`/`like_count` 本地；`favorite_count` 来自 catalog popularity `source=nextmoe metric=favorites`（`catalog_face.go:431`），与本地 `galgame.favorite_count` 不是同一计数（本地只跟本站写路径走，`collection_repo.go:30`）
- `is_liked` 本地 `galgame_like`；`is_favorited` 见 2.3 的 id 混用
- `resource_publish_banned` 本地旗，不挡读
- `covers[]`/`screenshots[]`：`sexual`/`violence` 是 int；`voted`/`vote_count`/`id` 由 `hydrateCoverVotes` 填（§6）
- `platform`/`language`/`type` 来自本站资源元数据，不是 catalog 平台
- `contributor[]`：最多 50（`contributorMaxPerGalgame`），`UserBrief`，**不跑 `IsRenderable`**
- `ratings[]`：该 gid 下全部 `galgame_rating`，`ORDER BY created DESC` 无 id 决胜（`detail_rating_repo.go:36`），作者 `IsRenderable` 才留下（`galgame_service.go:304`）。`view` 在这里是评分的「世界观」分项（`galgame_dto.go:187`）
- `rating`/`rating_count` 贝叶斯，与列表同一套
- `playtimes[]` catalog 聚合；`my_playtime` 见 §7
- `created`/`updated`；`dlsite_*` 商店链
- `refs` 外部 id 图

作者 hydrate 同样吞 OAuth 错（`hydrateDetailUsers` → `Hydrate`）。作者被封禁时详情仍 200，名字可能是真名（batch 成功）或「已注销用户」（失败）。

错误：gid 非数字 400；catalog 失败透传；找不到 404。optional 档：无 cookie 当匿名；Bearer 无效走 Auth 中间件（本路由在 `optAuth` 组，Bearer 坏了的行为取决于 `OptionalAuth` 实现，本普查不展开）。

测试：`galgame_mapper_test.go` 只测 DTO 映射；无 GetDetail handler 测试。封面票有 `cover_votes_bearer_test.go`。

### 2.2 `PUT /api/galgame/:gid/like`（切换）

`ToggleLike`（`galgame_handler.go:57`）→ `MustGetUser` → `GalgameService.ToggleLike`（`galgame_service.go:71`）。

**调用方**：`apps/web/app/components/galgame/Like.vue:36`。Flutter 零。

无请求体。响应 `OKMessage "操作成功"`，没有新的 `like_count` / `is_liked`。前端自己翻转（`Like.vue:19`）。

语义是 **toggle**：有行就删并 `like_count-1`，没有就插入并 `+1`（`interaction_repo.go:41`）。同一请求重放会把状态打回去。违反 PUT 幂等（01 §5）。

存在性：不查 catalog、不查 `published`、不查隐藏。`ToggleLike` 会 `Ensure` 出一行 `galgame`（`interaction_repo.go:47`）。对不存在的 gid 也能点赞。

自己给自己：`ownerOf` 读 `creator_user_id`（`galgame_service.go:165`）。相等 → `400/233 "您不能给自己点赞"`。没有本地行或 `creator_user_id` 空时 `ownerID=0`，闸放行。随后 `AdjustMoemoepoint(tx, 0, ±1, …)`（`galgame_service.go:86`）——`InteractionHelpers.AdjustMoemoepoint` **不挡 userID=0**（`interaction.go:14`）。消息创建会挡 `receiverID<=0`（`:24`）。

点赞加分的幂等键是 `moemoepoint.KeyNonce(reason, ref)`，ref = `galgame:<gid>`。取消时 delta=-1 同一 ref。

事务失败 → `500/233 "点赞失败"`。`like_count-1` 没有 `GREATEST`，能减到负数。

鉴权：required。Bearer 能赞（普通用户能力）。无 staff 路径。无可见性闸。

测试：无。

### 2.3 `GET /api/galgame/interactions/mine`

`MyInteractions`（`galgame_handler.go:74`）→ `parseCSVInts(c.Query("work_ids"), 100)`（`galgame_handler.go:84`）→ `GetMyInteractions`（`galgame_service.go:107`）。

**调用方**：`apps/web/app/composables/useMyGalgameInteractions.ts:42`。卡片 `ensureLoaded(gids)` 把 **论坛 gid** 放进 `work_ids`（`useMyGalgameInteractions.ts:55`，`card/Card.vue:56`）。非法 token、非正数、重复，parser **静默丢掉**（`galgame_handler.go:93`）。超过 100 截断。

响应 `{liked: int[], favorited: int[]}`（`galgame_dto.go:5`）。

- `liked`：该用户全部 `galgame_like.galgame_id`（`interaction_repo.go:31`），**忽略 `work_ids`**。一次拉全表。
- `favorited`：`catalog.MyFolderHoldings(ctx, token, ids)`，ids 就是 query 里那些整数，当作 catalog `work_id`（`galgame_service.go:124`）。返回的 `h.WorkID` 原样塞进数组（`:137`）。

前端把 `favorited` 再当 gid 去点亮心（`useMyGalgameInteractions.ts:18`，`:83`）。论坛 gid 与 catalog work id 撞车的那 10,289 个数会点亮错误的卡片。

`work_ids` 空、token 空、catalog 空：只回 likes，`favorited=[]`，**不报错**（`galgame_service.go:112`）。`folder:read` 不足只 `scopeWarn` 一行/分钟（`scope_warn.go:27`），仍 200 空收藏。

详情上的 `is_favorited` 走 `isFavorited`（`galgame_service.go:145`）：`MyFoldersContaining(ctx, token, int64(galgameID))`，同样把 gid 当 work id。scope 不足 → false。

测试：`galgame_interactions_test.go` 钉的是「走 holdings 不走全量 folder walk」，fixture 的 `work_id=42` 按 catalog id 处理。无 gid 映射测试。

---

## 3. 月历

五条 public GET，全部 `utils.IsSFW` + catalog。handler 用 `collectQuery` 把全部 query 透传（`entity_handler.go:194`），服务层只取自己认识的键。

时区：`Asia/Tokyo`，加载失败则 UTC（`calendar_service.go:27`）。「今天」「当前月」按这个钟。`ExpiresIn` 是到东京次日 0 点的秒数（`:125`）。

上游：`CatalogCalendar` → `/catalog/calendar` 在 v2 改写下进 `/v2/catalog/calendar`（`catalog_v2.go:76`）。`limit=100`（`calendarPageLimit`），**不跟游标翻页**。一个月超过 100 部，items 被截，`Count` 靠 `include_total=true`（`catalog_v2.go:169`）。

`ApplyWorksGate`：SFW 带 `content_limit=sfw`，年龄轴打开。卡片经 `GalgameEnricher.ToCards`（`galgame_enricher.go:31`），形状是 `GalgameCard`（比列表卡多 `status`、`release_precision`），同样不跑 `IsRenderable`。

### 3.1 `GET /api/galgame/calendar`

`GetMonth`（`calendar_handler.go:19`，`calendar_service.go:76`）。query `month`（`YYYY-MM`）；空则东京当前月。

响应 `CalendarMonthPage`（`calendar_dto.go:13`）：`month`、`today`、`items[]`、`meta{prev_month,next_month,has_prev,has_next,min_month,max_month,count}`。v2 月历不回 `month`/`today`/`min`/`max`，服务层自己补（`calendar_service.go:22-26`、`:86`）。`HasPrev`/`HasNext` 来自上游指针，缺省当 false（`:243`）。

**调用方**：`apps/web/app/components/galgame/calendar/Container.vue:31`。

错误：catalog 失败透传。非法 `month` 原样发给上游。

### 3.2 `GET /api/galgame/calendar/today`

`GetTodayFlag`（`calendar_service.go:105`）。无 query。拉**当前月**第一页，扫 `release_date == today`。

响应 `{today, has_release, expires_in}`。

超过 100 部且「今天」排在第 101 名之后，`has_release` 为 false。

**调用方**：`apps/web/app/composables/useGalgameReleaseToday.ts:60`（侧栏，本地缓存 `expires_in` 秒）。

### 3.3 `GET /api/galgame/calendar/pending`

query `year`。v2 改写加 `precision=year`（`catalog_v2.go:162`，`catalog_calendar_query_test.go:22`）。响应 `{year, items, count}`。

**调用方**：`calendar/Container.vue:53`。

### 3.4 `GET /api/galgame/calendar/tba`

无 query。v2 加 `status=unknown`（`catalog_v2.go:164`）。响应 `{items, count}`。pending 与 tba 在上游曾经落到同一未定桶，靠 precision/status 分开（测试注释，`catalog_calendar_query_test.go:20`）。

**调用方**：`calendar/Container.vue:65`。

### 3.5 `GET /api/galgame/calendar/upcoming`

`GetUpcoming`（`calendar_service.go:166`）。从当前月走到 `max_month`，最多 24 个月（`upcomingMonthCap`），并发 8（`:18`）。第 0 月用第一次拉取；其余 `go fetchMonthRaw`，**失败的月份直接跳过**（`:190`，`err == nil` 才写入）。

过滤：`CatalogItemRenderable` 且 `release_date >= 当月`（字典序，月精度 `"2026-08"` 排在该月所有日之前，`:205`）。再按卡片 `release_date[:7]` 分桶。`count` 是卡片数，不是上游 total。

**调用方**：`calendar/Container.vue:43`。

N+1：最多 24 次 catalog 月历。无缓存。

### 3.6 `GET /api/galgame/collected-calendar`

`CollectedCalendar`（`galgame_handler.go:53`）→ `listRepo.ListCollectedCalendar`（`list_repo.go:250`）。**本地 SQL**，与浏览页同一人口：`published` + 有资源 + SFW `content_limit`。按 `galgame.created` 的年/月聚合。

响应 `[]{year, month}`。handler **不回 error**（Scan 失败就是空数组）。无分页。

**调用方**：`apps/web/app/components/galgame/card/Nav.vue:163`，仅 `isShowAdvanced` 的 `/galgame` 浏览页 `onMounted` 拉一次（`:171`）。实体页挂同一 Nav 但不拉。

测试：`list_repo_test.go` `TestCollectedCalendarHonoursTheReadersGate`。CalendarService 本身无测试。

鉴权：六条全部匿名。无 hidden/ban 之外的读闸（catalog hidden 在 `CatalogItemRenderable`；本地 collected-calendar 靠 `published`）。

---

## 4. `GET /api/galgame/:gid/link/all`

`GalgameProxyHandler.GetGalgameLinks`（`galgame_proxy_handler.go:18`）→ `GalgameProxyService.GetGalgameLinks`（`galgame_proxy_service.go:28`）→ `CatalogWorkLinks`（`catalog_detail.go:290`）→ 再走一遍 `CatalogWorkDetail`。

**调用方**：`apps/web/app/components/galgame/link/Link.vue:9`（lazy，另带无用 query `galgame_id`）。Flutter 零。Nitro 零。

gid 非正数 → `400/233 "无效的 Galgame ID"`。catalog 找不到或 hidden → **200 空数组**（`catalog_detail.go:296`），与详情的 404 不一致。

响应 `[]GalgameLink`（`galgame_proxy_dto.go:3`）：`id`、`user`、`galgame_id`、`name`、`link`。服务层只填 `galgame_id`/`name`/`link`（`galgame_proxy_service.go:42`）。**`id` 恒 0，`user` 恒零值**。`Link.vue` 只用 `name`/`link`。

无 SFW。封面级 NSFW 外链对 SFW 读者可见。optional 档，但不读 uid。

测试：无。

---

## 5. `GET /api/galgame/drafts`

`DraftsHandler.GetDrafts`（`drafts_handler.go:18`）。public。`parseCollectionPage` 默认 24、上限 50、非法静默夹（`collection_handler.go:174`）。query：`official_id`/`tag_id`/`engine_id`/`series_id`/`original_language`（`drafts_handler.go:20`）。

服务：`claimed=false` 打 catalog works 搜索（`drafts_service.go:36`）。v2 路径 `/v2/catalog/works`，`label_id` 改写为 `company_id`，`page` 改写成 `cur_` 游标（`catalog_v2.go:67`、`:119`、`:144`）。`OpenPopulation` 只开年龄轴，**不设 `content_limit`**（`drafts_query_test.go:68`：「unclaimed-works funnel」）。SFW 读者看到 NSFW 未认领作品。

`encodePageCursor` 编码的是 **页码本身** 的十进制，不是 `(page-1)*limit`（`catalog_v2.go:196`）。测试只断言 `cur_` 前缀（`drafts_query_test.go:59`）。若上游把游标当 offset，第 2 页会跳 2 行而不是 24 行。

响应 `{items: GalgameCard[], total}`（`galgame_dto.go:99`）。`claimed=false` 的语义是 catalog 上还没有 kungal 认领。方案③之后用户不再认领游戏；这条是旧漏斗。

**调用方**：搜 `/galgame/drafts`、`GetDrafts`，在 `apps/web/app/**`、`apps/web/server/**`、`apps/web/shared/**`、`kungal-apps/**/*.dart` **零命中**（只剩 `router.go:167` 与 golden）。

鉴权：完全匿名。无 SFW。无 hidden 闸（未认领行本来没有 kungal claim）。

测试：`drafts_query_test.go`（claimed=false、page→cursor、entity 过滤、nsfw=true）。

---

## 6. 封面投票

`CoverVoteHandler.cast`（`cover_vote_handler.go:37`）。PUT 投票、DELETE 取消。required。

**调用方**：`apps/web/app/components/galgame/Covers.vue:63`，已投则 DELETE，否则 PUT。Flutter 零。

路径 `:gid`、`:coverId`（int64>0）。gid 非法 → `400/233 "无效的 Galgame ID"`；coverId 非法 → `400/233 "无效的封面 ID"`。无 body。

token 空 → `401/205`。catalog/client 未配置 → `503/233 "封面投票服务暂不可用"`。

`workIDOf`：`CatalogWorkIDs` 把论坛 gid 译成 catalog work id（`cover_vote_handler.go:78`）。译不到 → `404/233 "条目不存在"`。然后 `VoteCover` / `UnvoteCover` 带用户 token。

响应 `{cover_id, vote_count, voted}`。PUT/DELETE 都是置位/撤销，幂等由上游保证。前端假设投一张会把它票从其它封面上挪走（`Covers.vue:42`）。

错误映射（`coverVoteError`，`:90`）：

| 上游 | 论坛 |
|---|---|
| `ErrInsufficientScope` | `403/235`「投票需要新的授权…」 |
| `ErrUnauthorized` | `401/205` |
| `ErrNotFound` | `404/233 "条目或封面不存在"` |
| `ErrNotConfigured` | 503 同上 |
| HTTP 403 | `403/233 "你没有权限执行此操作"` |
| 其它 UserAPIError / 传输 | 503，日志 |

缺 `catalog:edit`（或投票用的那个写 scope）的旧会话走 235，kunFetch 会提示重新登录。

详情页读票：`hydrateCoverVotes`（`cover_votes.go:12`）同样先译 work id。有 token 走用户面；`ErrInsufficientScope` **或** `ErrUnauthorized` 降到公开计数，丢掉 `voted`（`:30-34` 注释：用户面曾经对所有人 401）。失败只 warn，封面仍下发，`vote_count=0` `voted=false`。

无可见性闸：hidden/NSFW 作品只要 gid 能译成 work id 就能投。Bearer 能投，无 staff。

测试：`cover_vote_handler_test.go`（用户面 Bearer、DELETE、235、404、未登录、未配置）；`cover_votes_bearer_test.go`（读面降级）。

---

## 7. 游玩时长

OAuth 授权范围现为 `playtime:read playtime:write`（`apps/web/app/utils/oauth-auth.ts:56`）。旧会话刷新不会自动加上（同文件 `:49-54`）。缺 scope 的读在详情里静默变 `my_playtime=null`；写面 403/235。

### 7.1 `PUT /api/galgame/:gid/playtime`

`PlaytimeHandler.Report`（`playtime_handler.go:34`）。required。body `{minutes?: int, status?: string}`，至少一个。

- `minutes`：0–60000（`PlaytimeMinutesMax`，`user_playtime.go:17`）。越界 → `400/233 "游玩时长超出可记录的范围"`。0 是撤回：写入后聚合忽略 <10 的值（`PlaytimeMinutesFloor`，`:18`；`playtimeWithdrawn`，`playtime.go:28`）
- `status`：空字符串 = 删 work-state。非空必须是 `playstate.Valid`：`wish/doing/done_one_route/done_main/done_all/on_hold/dropped`（`playstate.go:19`）。未知 → `400/233 "未知的游玩状态"`。读回可能出现只读 `"done"`（catalog 有 `state=done` 无 completion，`playstate.go:69`），不能再当输入

gid 非法 400。body 绑定失败或两个字段都缺 → `400/233 "请求参数错误"`。token 空 → `401/205`。

gid → catalog work id（`playtime.go:113`）。没有 → `404/233 "条目不存在"`。先 `ReportPlaytime`（若有 minutes），再 `PutWorkState`/`DeleteWorkState`（若有 status）。只改其中一个时，另一个从上游读回。

响应 `GalgameMyPlaytime{minutes, status}`。撤回后两者都空时服务层回 `nil`（`playtime.go:218`），JSON `data: null`。前端把「`minutes===0 && status===''`」当清除成功（`PlaytimeModal.vue:75`）。

错误（`playtimeError`，`playtime_handler.go:84`）：

| 上游 | 论坛 |
|---|---|
| `ErrInsufficientScope` | `403/235`「记录游玩时长需要新的授权…」 |
| `ErrUnauthorized` | `401/205` |
| `ErrNotFound` | `404/233 "条目不存在"` |
| `ErrNotConfigured` | `503/233 "游玩时长服务暂不可用"` |
| HTTP 429 | `400/233 "操作太频繁, 请稍后再试"`（HTTP 400，不是 429） |
| HTTP 400 | `400/233 "游玩时长不被接受"` |
| 其它 | 503 |

**调用方**：`apps/web/app/components/galgame/header/PlaytimeModal.vue:64`。Flutter 零。

无可见性闸。映射 gid 是对的。

测试：`playtime_handler_test.go`（用户面、completion、坏输入、235、只改 status、只改 minutes、空 status 删除）。

### 7.2 `GET /api/galgame/playtime/mine`

`ListMine`（`playtime_handler.go:67`）。`page`/`limit` 静默夹（默认 24，最大 50）。token 空 401/205。缺 scope 403/235。

实现：最多 10 页 × 100 条扫 `/v2/me/playtimes` 与 work-states（`playtime.go:17-19`、`:316`）。触顶 `truncated=true`。按 work 折叠多客户端（取最大 minutes，`clients++`）。`GIDsByCatalogIDs` 译回论坛 gid。译不到的 work 丢掉。

然后 **先** 算 `total`/`total_minutes`/`finished_works`（`:270`），再对当前页 `HydrateCardsByIDs(..., isSFW)`。SFW 读者：NSFW 作品仍计入合计，卡片被跳过（`:303`）。`total` 与 `items` 谓词不同。分页按折叠后的全量切，一页可能空一半。

`finished_works` 计 `done_one_route`/`done_main`/`done_all`，不计只读 `"done"`（`playStateFinished`，`playtime.go:32`）。

响应 `PlaytimeMinePage`（`playtime_dto.go:10`）：`items[{galgame: GalgameListCard, minutes, status, clients}]`、`total`、`total_minutes`、`finished_works`、`truncated`。

**调用方**：`apps/web/app/components/user/Playtime.vue:11`（仅本人资料页，`constants/user.ts:231`）。

测试：`playtime_test.go` 折叠/撤回/通关计数。无 ListMine 的 SFW/`total` 测试。无 handler 测试。

---

## 8. 收藏夹

收藏夹本体在 catalog `/v2/me/folders`。本站 `galgame_collection` 自迁移 091 起只是 **别名表**（论坛 `cid` ↔ `catalog_folder_id`，`collection_alias_repo.go:9`，`091_galgame_collection_catalog_alias.up.sql`）。内容列冻结。成员资格在 catalog；本地 `galgame.favorite_count` 只跟本站写路径走。

可见性：catalog 只有 `public`/`private`（`catalogclient.FolderVisibility*`，`user_folders.go:21`）。`restricted` 已随切走丢掉。模型注释写着切走当天生产 0 行 restricted、0 行 `galgame_collection_viewer`（`model/collection.go:5`）。DTO `oneof=public private`（`collection_dto.go:8/14`）。前端 `CollectionVisibility = 'public' | 'private'`（`shared/types/collection.ts:3`）。

**本分支仍会在空文件夹列表时创建未命名默认收藏夹**（`ensureDefault`，`collection_service.go:342`，name=`""`，`visibility=public`，`is_default=true`）。`GetMyCollectionsForGalgame` 在 `MyFolders` 为空时调用它（`:304`）—— **GET 有写副作用**。任务书提到的 PR #178 要停掉这件事；当前工作区代码还在。默认名空，是为了沿用「用主人名当标签」的旧渲染（`:339` 注释）。

所有写面需要用户 OAuth token（`collectionToken`，`collection_handler.go:166`），空 → `401/205`。缺 `folder:read`/`folder:write` → `403/235`（`collection_err.go:27`，注释记了 2026-09-08：曾经把 scope 不足折成 205，kunFetch 探针打 `/user/status` 觉得会话还活着，收藏夹整组静默失败）。

### 8.1 `POST /api/galgame/collection`

`Create`（`collection_handler.go:25`）。body `CreateCollectionRequest`：`name` required 1–60、`description` max 500、`visibility` required public/private。校验失败 400。信任闸 `gate.DecisionDeny` → `422/233 "内容包含违禁词，无法发布"`（`compose.go:9`）。Hold 仍创建并后台扫描。

`catalog.CreateFolder` 成功后 `MintAlias`。别名插入失败：上游文件夹留下，下次 list 懒铸别名（`collection_service.go:90`）；本次 `500/233 "创建收藏夹失败"`。

响应 `data` 为 **裸整数 id**（论坛 cid）。无 `Idempotency-Key`。HTTP 200 不是 201。

**调用方**：`apps/web/app/components/galgame/collection/EditModal.vue:70`。

### 8.2 `PATCH /api/galgame/collection/:cid`

`Update`（`collection_handler.go:45`）。cid 非数字 → `400/233 "无效的收藏夹 ID"`。body 三字段全可选指针；全空 → `400/233 "没有要修改的内容"`。只扫描本次提交的文本（`collection_service.go:116`）。

`aliasForMutation`：别名不存在 → 404；非主人且 `!user.Can(perm.CollectionEditAny)` → **404**（不是 403）（`collection_service.go:479`）。主人走 `PatchFolder`；版主走 `ModeratePatchFolder`（用户 token，catalog 判资格）。Bearer 的 `Can` 恒 false，不能改别人的。

响应 `OKMessage "收藏夹已更新"`。

**调用方**：`EditModal.vue:82`。

### 8.3 `DELETE /api/galgame/collection/:cid`

`Delete`（`collection_handler.go:69`）。`user.Can(perm.CollectionDeleteAny)`。主人不能删 `is_default`（先本地拦，`collection_service.go:164`，`403/233 "默认收藏夹不能删除"`；上游 422 也会译成同一句）。版主删别人的不读内容、不改本地 favorite_count（`:151`）。

主人路径：`MyFolders` + `MyFolderItems` 算「离开书库」的 work，再 `DecrementFavoriteCounts`。这里的 work id **按 catalog work id 去减 `galgame.id`**（`worksLeavingTheLibrary` 回 `int(it.WorkID)`，`collection_service.go:215`；`DecrementFavoriteCounts` `WHERE id IN ?`）。gid/work_id 再混一次。某个 sibling folder 读失败则整段放弃（`:206`），计数宁可不减。

别名删除失败只 log（`:180`）。响应 `OKMessage "收藏夹已删除"`。

**调用方**：`pages/galgame/collection/[id].vue:74`。默认夹按钮被 `!detail.is_default` 藏掉（`:124`）。

### 8.4 `GET /api/galgame/collection/:cid`

optional。`parseCollectionPage`。`GetDetail`（`collection_service.go:354`）。

别名 404。主人判定 `viewerID == alias.UserID`。`userClient.User` 的 found/error 丢掉（`:361`）。OAuth 故障时 `User{Status:0}`，`IsRenderable` 为 true，封禁主人的公开夹仍可读。封禁且非主人 → 404。

`loadFolderContents`（`collection_items.go:28`）：**先走公开面**（省配额，2026-09-20 user 90769）。公开夹主人也走公开面。`PublicFolder` 404 且是主人且有 token → `MyFolder`（私密夹）。匿名/路人看私密 → 404。主人 **没有** OAuth token 时，条件 `token==""` 让私密夹也失败（`collection_items.go:36`）。

items 按 `created_at` 降序在进程内排（catalog 是 `updated_at` 升序水印，`collection_service.go:371`），再切片。`pageOfWorkIDs` 把 `FolderItem.WorkID` 交给 `HydrateCardsByIDs`（论坛 gid 批量接口）。**catalog work id 被当成 gid 去解析。** hydrate 失败吞掉，`galgames=[]`，`total` 仍是全量（`:377`）。SFW 再滤掉一批，`total` 不动。`item_count` 来自 folder 元数据，可能与 `total`、与当前页 `galgames` 都不同。

排序只有 `created_at` 字符串，无 id 决胜。6 小时 Redis 缓存（`folderItemsCacheTTL`），键含 `folder.UpdatedAt`。

响应 `CollectionDetail`：`id`（cid）、`name`/`description`/`visibility`/`is_default`/`item_count`/`is_owner`/`owner`/`galgames`/`total`/`created`/`updated`（`time.Time`，解析失败为零值，序列化 `0001-01-01T00:00:00Z`）。

**调用方**：`pages/galgame/collection/[id].vue:13`。

### 8.5 `GET /api/galgame/:gid/collections/mine`

required + token。gid 非法 400。`MyFolders`；空则 `ensureDefault`（写）。再 `MyFoldersContaining(gid 当 work id)`。按 `is_default` 先、`updated_at` 新先（`sortFolders`，`:502`）。懒铸别名。

响应 `{collections: MyCollectionForGalgame[]}`，`contains` 表示该夹是否已收这个「work id」。gid 映射错则 `contains` 全假，保存时 PUT 的也是错 id。

**调用方**：`collection/PickerModal.vue:28`。

### 8.6 `PUT /api/galgame/:gid/collections`

`SetMembership`（`collection_handler.go:124`）。body `{collection_ids: int[]}`，`max=50,dive,gt=0`。空数组 = 从全部夹中移除。去重。

`FolderIDsOwnedBy`：cid 必须属于当前用户。有一个不是 → `403/233 "收藏夹不存在或不属于您"`（`collection_service.go:228`）。**不能靠 staff 改别人的成员资格。**

当前持有：`MyFoldersContaining(int64(galgameID))`。差集 `PutFolderItem` / `DeleteFolderItem`，路径里的 id 就是传入的整数（`user_folders.go:290`）。首次加入/最后一次移除才动本地 `favorite_count` 和萌萌点（reason 仍是 `liked`，ref `galgame:<gid>`）。本地事务失败只 log，上游已经写完（`:292`）。

这是「整集成员资格替换」，重放同 body 无翻转。路径是资源不是动词。gid 仍当 work id。

**调用方**：`PickerModal.vue:72`。

### 8.7 `GET /api/user/:id/collections`

optional。`ListForUser`（`collection_service.go:398`）。owner 封禁且查看者不是本人 → **200 空列表**（与详情 404 不同）。OAuth 错 fail-open（`:399`）。

本人 + token：`MyFolders`（含私密）。429 配额则退回 `PublicFolders`（丢掉私密，`collection_items.go:116`）。路人：只有公开夹。

全量拉回再本地分页。`total` 是 folder 数。预览封面：每夹最多 4 条 `FolderPreviewItems`，work id 再当 gid 去 `GetBatchPublic`。失败的夹封面为空数组。

响应 `Paginated` `{items: CollectionSummary[], total}`。`preview_covers` 是 URL 字符串数组。

**调用方**：`apps/web/app/components/user/CollectionGalgame.vue:13`。

limit 越界静默夹到 50，01 §4 要求 `400 LIMIT_TOO_LARGE`。

测试：`collection_adapter_test.go`（排序、分页、离开书库）；`collection_items_test.go`（公开面、缓存、私密回退、配额回退）；`collection_scope_err_test.go`（235 vs 205、429）。无 Create/SetMembership 的 gid 映射测试。无 handler 测试。

---

## 9. 已存在的 v1 面

`GET /api/v1/galgames/{galgame_id}/moyu-patches`（`apiv1/register.go:14`，`moyu.go:89`）。public。`galgame_id` 字符串 `^[1-9][0-9]{0,18}$`。`CatalogWorkIDForGID` 译 catalog id；没有 → `NOT_FOUND`。摸鱼补丁 Redis 缓存 30 分钟（`moyu.go:22`）。信封 `repr.List`，`object: moyu_patch`，id 是摸鱼站的，文档写明既不是论坛 gid 也不是 catalog work id（`moyu.go:59`）。`publisher` 是 `repr.UserRef`，OAuth 缺失走 `DeletedUserRef`。无 SFW。无分页。

G1 迁 v1 时应沿用这套：字符串 `{galgame_id}`、先译 catalog id、problem+json、`viewer` 装个性化字段。不要沿用收藏夹那种把 gid 塞进 `work_id` 的写法。

---

## 10. 命名问题清单（迁 v1 时逐条执行）

| 现状 | 问题 | v1 |
|---|---|---|
| 路径 `:gid`、字段 `gid` | 01 §3 禁用名 | `{galgame_id}` / `galgame_id` |
| 收藏夹路径 `:cid` | 禁用名 | `{collection_id}` |
| catalog 作品 id 叫 `work_id` / query `work_ids` | 与论坛 gid 撞车；前端实际传的是 gid | `catalog_work_id`；互动接口收 `galgame_id` |
| `created` / `updated` | 禁用名 | `created_at` / `updated_at` |
| `view`（列表浏览量）与 `view`（评分世界观） | 禁用名；一义两名 | `view_count` / `worldview` |
| `user`（作者、评分作者、贡献者、外链作者） | 禁用名 | `author` / `contributor` / 外链里删掉零值 |
| `status` int（详情 0/2） | 裸整数；与 Problem.status 撞型 | `state` 封闭字符串 |
| `galgames`（列表信封） | 页码集合应 `items` + `object:list` | `items` |
| `is_on_forum` | 列表=published，详情=有本地行 | 拆成 `is_indexed` / 删 |
| `is_liked` / `is_favorited` / `voted` | 随查看者变 | `viewer.has_liked` 等 |
| `content_limit` | 编辑轴，取值 `sfw`/`nsfw` | 可留，或改 `display` 封闭枚举；年龄轴另叫 `content_rating` |
| `type`/`language`/`platform` 过滤 | 开放字符串，未知静默空集 | 封闭枚举 + `400 UNKNOWN_ENUM_VALUE` |
| `sort_field`+`sort_order` | 未知 sort_field 静默回落 | `sort=` 封闭 token + `400 UNKNOWN_SORT` |
| `library=true` | 动词/布尔切引擎 | 两个集合：`/galgames` 与 `/catalog-works`（或 `/galgames?population=indexed`） |
| `/galgame/:gid/like` PUT toggle | 非幂等 | `PUT`/`DELETE` 槽位 |
| `/galgame/:gid/link/all` | 动词 `all` | `/galgames/{galgame_id}/links` |
| `/galgame/drafts` | 旧认领漏斗 | 不迁或改名 `/unindexed-works`；零调用方 |
| `/galgame/interactions/mine` | 动词路径 | 列表条目的 `viewer`，或 `/me/galgame-likes` |
| `/galgame/playtime/mine` | 单数+动词 | `/me/playtimes` |
| `/galgame/:gid/playtime` | 可留成子资源 | `PUT /galgames/{galgame_id}/playtime` |
| `kind`（封面） | 禁用名 | `cover_slot` 或沿用 catalog 词 |
| `sexual`/`violence` int | v1 Image 用 `safe/suggestive/explicit/null` | `repr.Image` |
| 全部 id JSON number | 01 §3 | 十进制字符串 |

---

## 11. 发现清单（平铺，不排序）

1. `SetMembership` / `GetMyCollectionsForGalgame` / `isFavorited` / `GetMyInteractions` 把论坛 gid 当作 catalog `work_id`（`collection_service.go:235/248/311`，`galgame_service.go:124/149`）。封面投票与游玩时长走 `CatalogWorkIDs`（`cover_vote_handler.go:78`，`playtime.go:113`）。v1 moyu 走 `CatalogWorkIDForGID`（`moyu.go:97`）。
2. `GetMyInteractions` 的 `liked[]` 是 gid，`favorited[]` 是 holdings 的 `WorkID`；前端把两者都当 gid（`useMyGalgameInteractions.ts:50-83`）。
3. `pageOfWorkIDs` / `resolvePreviewCovers` / `worksLeavingTheLibrary` / `DecrementFavoriteCounts` 用 `FolderItem.WorkID` 去碰 `galgame.id`（`collection_service.go:215/456/533`）。
4. `contentLimitOf` 在 claim 无合法 `content_limit` 时用 `content_rating` 填镜像（`catalog_wire.go:271-286`）。本地 SQL SFW 闸读这列（`list_repo.go:362`）。
5. `GET /galgame/:gid` 对 SFW 读者不加整页 `content_limit` 闸（`catalog_face.go:519`），只剥 sexual 标签（`galgame_service.go:253`）。列表有闸，深链没有。
6. `GET /galgame/drafts` 开年龄轴、不开编辑轴（`drafts_service.go:57`，`drafts_query_test.go:68`）。零网页/Nitro/Flutter 调用方。
7. `GetGalgameLinks` 对不存在/hidden 回 200 `[]`（`catalog_detail.go:296`）；详情回 404。
8. `GalgameLink.id`/`user` 恒零值（`galgame_proxy_service.go:42`）。
9. `PUT /:gid/like` 是 toggle（`interaction_repo.go:41`）。无目标存在性/可见性检查。`ownerID=0` 时给 user 0 调萌萌点（`galgame_service.go:86`，`interaction.go:14`）。`like_count-1` 能变负。
10. `Hydrate` 丢 OAuth error，缺人填 `Placeholder`（`hydrate.go:37`，`userclient.go:211`，硬编码「已注销用户」）。列表卡、贡献者、enricher **不**调 `IsRenderable`。评分调了（`galgame_service.go:304`）。收藏夹主人调了但 found/error 丢掉（`collection_service.go:361/399`）—— OAuth 故障时封禁闸 fail-open。
11. 列表 `total` 与 `items`：SQL/catalog 计数之后 hydrate `continue`（`galgame_service.go:442`）。收藏夹详情 hydrate 失败变空数组、`total` 保留（`:377`）。游玩时长合计在 SFW 过滤之前算（`playtime.go:270-293`）。
12. `GET .../collections/mine` 在空列表时 `ensureDefault` 写出未命名公开默认夹（`collection_service.go:304/342`）。GET 有写。PR #178 尚未进入本分支。
13. catalog 月历 `limit=100` 不跟游标（`calendar_service.go:20`）。`GetUpcoming` 吞掉失败月份（`:190`）。`GetTodayFlag` 只扫当前月第一页。
14. `sort_field` 未知值静默回落：本地 `resource_update_time`（`list_repo.go:51`），catalog `popularity`（`galgame_library.go:90`）。`library=false` 时 `popularity` 按资源更新时间排。
15. `type`/`language`/`platform` 开放字符串，未知当精确匹配。`page`/`limit` 因 `min=1` 事实必填。
16. `is_on_forum` 列表= `published`（`galgame_service.go:456`），详情= 本地行存在（`:240`）。
17. `NamePreference` + 匿名 `IsSFW` 仍读 `KUNGalgameSettings`（`router.go:47`，`settings.go:42/52`）。本段所有 public/optional 读面都受影响。
18. 详情 `introduction[].intro` 是 HTML（`galgame_mapper.go:52`）。评分全量塞进详情，无分页，`ORDER BY created DESC` 无 id 决胜（`detail_rating_repo.go:36`）。
19. `IncrementView` 在 goroutine 里（`galgame_service.go:215`），错误丢掉。
20. `ListIDs`/`BayesianRatings`/`ListCollectedCalendar`/`UserLikedGalgames`/`FindLocal`/`FindRatingsByGalgame` 的 `Scan`/`Pluck`/`First` 不接 error。
21. `parseCollectionPage` / playtime ListMine / drafts 的 `limit>50` 静默夹。`parseCSVInts` 静默丢非法 id。
22. 封面投票/游玩时长/收藏夹缺 scope → `403/235`。`isFavorited` / `GetMyInteractions` / 详情 `my_playtime` 缺 scope → 200 当「未收藏/无记录」。
23. playtime 上游 429 被译成论坛 `400/233`（`playtime_handler.go:101`）。
24. `Create` 收藏夹 200 + 裸整数 id，无幂等键。`OKMessage` 2xx 顶层无资源对象。
25. 私密收藏夹：主人无 OAuth token 时详情失败（`collection_items.go:36`）。路人 404。
26. `ListForUser` 对封禁主人回空列表 200；`GetDetail` 回 404。
27. catalog 引擎丢掉浏览页的类型/语言/平台/收录日/评分过滤（`galgame_library.go:44-56`）。`library` 钉引擎这件事本身是对的（`galgame_library_test.go:34`）。
28. `released_from` 不接受 `YYYY-MM-DD`（`release_date.go:46`）。
29. `GalgameListPage.galgames` 与草稿/月历的 `items` 与用户收藏夹的 `items` 三种键。
30. Flutter App 本段零调用（搜 `kungal-apps/**/*.dart` 的 `/galgame` API 路径，只有占位屏）。
31. Bearer staff：收藏夹用 `user.Can`，正确。本段无 `perm.CanUser` / `role.Can*`。
32. `encodePageCursor` 编码页码而非 offset（`catalog_v2.go:196`）。drafts/library 翻页依赖上游把该游标当页码。
33. 本段 23 条里，点赞、外链、GetDetail、ListMine、Create/SetMembership **没有**打到 handler 的契约测试。

---

## 12. 需要生产库回答的问题

请按下面的顺序跑。库是论坛库 `kungalgame`。catalog / OAuth 库标了出来。

1. **gid 与 catalog work id 撞车规模**（对应发现 1–3）：
   论坛侧只能间接估计。需要 catalog 库：
   `SELECT count(*) FROM (SELECT site_work_id::bigint AS gid FROM claims WHERE site='kungal' AND site_work_id ~ '^[0-9]+$') c JOIN works w ON w.id = c.gid AND w.id <> (SELECT work_id FROM claims c2 WHERE c2.site='kungal' AND c2.site_work_id = c.gid::text LIMIT 1);`
   以及任务书已给的 10,289 是否仍成立：对每个 kungal `site_work_id` 看是否存在 `works.id` 等于该整数且不是该 claim 的 work。

2. **`galgame.content_limit` 分布**（对应发现 4、镜像滞后）：
   ```sql
   SELECT content_limit IS NULL AS is_null, content_limit, count(*)
   FROM galgame GROUP BY 1, 2 ORDER BY 3 DESC;
   SELECT count(*) FROM galgame WHERE published;
   SELECT count(*) FROM galgame WHERE published AND content_limit IS NULL;
   ```

3. **`published` 与资源**：
   ```sql
   SELECT g.published,
          EXISTS (SELECT 1 FROM galgame_resource r WHERE r.galgame_id = g.id) AS has_resource,
          count(*)
   FROM galgame g GROUP BY 1, 2;
   ```

4. **点赞**：
   ```sql
   SELECT count(*) FROM galgame_like;
   SELECT count(*) FROM galgame WHERE like_count < 0;
   SELECT count(*) FROM galgame g
     WHERE g.like_count <> (SELECT count(*) FROM galgame_like l WHERE l.galgame_id = g.id);
   SELECT percentile_cont(0.99) WITHIN GROUP (ORDER BY c)
   FROM (SELECT count(*) c FROM galgame_like GROUP BY user_id) t;
   ```
   最后一个决定 `interactions/mine` 一次 `liked[]` 有多大。

5. **收藏夹别名**：
   ```sql
   SELECT count(*) FROM galgame_collection;
   SELECT count(*) FROM galgame_collection WHERE catalog_folder_id IS NULL;
   SELECT visibility, is_default, count(*) FROM galgame_collection GROUP BY 1, 2;
   SELECT count(*) FROM galgame_collection WHERE name = '';
   SELECT count(*) FROM galgame_collection_item;
   SELECT count(*) FROM galgame_collection_viewer;
   ```
   `visibility`/`name` 是冻结列，与 catalog 真源会漂。`galgame_collection_viewer` 若仍为 0，restricted 不用建模。

6. **冻结表 `galgame_favorite`**（用户域已经问过，本段互动不再读它，再确认一次）：
   ```sql
   SELECT count(*) FROM galgame_favorite;
   SELECT max(created) FROM galgame_favorite;
   ```

7. **列表过滤词表是否有人用**：
   ```sql
   SELECT type, count(*) FROM galgame_resource GROUP BY 1 ORDER BY 2 DESC;
   SELECT language, count(*) FROM galgame_resource GROUP BY 1 ORDER BY 2 DESC;
   SELECT platform, count(*) FROM galgame_resource GROUP BY 1 ORDER BY 2 DESC;
   SELECT unnest(provider) p, count(*) FROM galgame_resource GROUP BY 1 ORDER BY 2 DESC;
   ```
   `galgame_type`（评分，对应 `game_type=`）：
   ```sql
   SELECT jsonb_array_elements_text(galgame_type) t, count(*)
   FROM galgame_rating WHERE galgame_type <> '[]'::jsonb
   GROUP BY 1 ORDER BY 2 DESC;
   SELECT count(*) FROM galgame_rating WHERE galgame_type = '[]'::jsonb;
   ```

8. **贝叶斯评分先验**：
   ```sql
   SELECT count(*), avg(overall), min(overall), max(overall) FROM galgame_rating;
   SELECT count(DISTINCT galgame_id) FROM galgame_rating;
   ```

9. **月历 100 条截断**：本地没有发售日全量。镜像列：
   ```sql
   SELECT date_trunc('month', release_date) m, count(*)
   FROM galgame WHERE release_date IS NOT NULL
   GROUP BY 1 HAVING count(*) > 80
   ORDER BY 2 DESC LIMIT 20;
   ```
   真正的月历人口在 catalog。这条只提示哪些月可能撞 `limit=100`。

10. **合并重定向**：
    ```sql
    SELECT count(*) FROM galgame_redirect;
    SELECT count(*) FROM galgame_redirect r
      LEFT JOIN galgame g ON g.id = r.new_gid WHERE g.id IS NULL;
    ```

11. **详情评分体积**：
    ```sql
    SELECT max(c), percentile_cont(0.99) WITHIN GROUP (ORDER BY c)
    FROM (SELECT count(*) c FROM galgame_rating GROUP BY galgame_id) t;
    ```

12. **游玩时长**在 catalog 库，本库没有表。需要 orchestrator 打 catalog：
    每用户 playtime 行数、是否有人超过 1000 行（10 页 × 100 的扫盘上限）。

13. **sitemap 深度**：`indexed=true` 的 `published` 行数（问 3 已含）。`MAX_PAGES=130` × `limit=50` = 6500；超过就要看 sitemap 是否截断。

---

## 13. 提议的 v1 资源表（提议，不是契约）

| 旧 | 提议 | 理由 |
|---|---|---|
| `GET /api/galgame`（无 library） | `GET /v1/galgames` 页码集合 | 01 §4 K11：浏览页要分页器。过滤/排序封闭枚举。`include_nsfw=` 取代 cookie |
| `GET /api/galgame?library=true` | `GET /v1/catalog-works`（或 `/v1/galgames` 另一个 `x-population`） | 两个引擎两份集合，页面钉死，空过滤不切引擎 |
| `GET /api/galgame?indexed=true` | 同一 `GET /v1/galgames?indexed=true` | sitemap 仍要页码 + `total` |
| `GET /api/galgame/:gid` | `GET /v1/galgames/{galgame_id}` | 字符串 id；`viewer` 装 liked/favorited/my_playtime/cover voted；hidden/NSFW 统一闸 |
| `PUT /api/galgame/:gid/like` | `PUT` + `DELETE /v1/galgames/{galgame_id}/like` | K16 槽位，幂等 |
| `GET /api/galgame/interactions/mine` | 列表/卡片走 `viewer`；全量 likes 另开 `/v1/me/galgame-likes` 游标 | 去掉 `work_ids` 混 id；likes 全量拉改为游标 |
| `GET /api/galgame/calendar` 及其子路径 | `GET /v1/release-calendar` + query `window=month\|upcoming\|pending\|tba` + `month`/`year` | 一个集合，窗口是过滤；today 旗可以是 `HEAD` 或小资源 `/v1/release-calendar/today` |
| `GET /api/galgame/collected-calendar` | `GET /v1/galgames/collected-months` | 给过滤 UI 的月份 enumeration |
| `GET /api/galgame/:gid/link/all` | `GET /v1/galgames/{galgame_id}/links` | 去掉 `all`；找不到与详情同一 `NOT_FOUND` |
| `GET /api/galgame/drafts` | **不迁，删旧路由** | 零调用方；`claimed=false` 是旧认领漏斗 |
| `PUT/DELETE .../cover/:coverId/vote` | `PUT/DELETE /v1/galgames/{galgame_id}/covers/{cover_id}/vote` | 已经是槽位；继续先 gid→catalog work id |
| `PUT /api/galgame/:gid/playtime` | `PUT /v1/galgames/{galgame_id}/playtime` | 已是替换；缺 scope → `403 SCOPE_REQUIRED` |
| `GET /api/galgame/playtime/mine` | `GET /v1/me/playtimes` 页码或游标 | `total` 与 `items` 同一谓词；先译 catalog id |
| `POST /api/galgame/collection` | `POST /v1/collections` → 201 + Location | K12 幂等键 |
| `GET/PATCH/DELETE /api/galgame/collection/:cid` | `/v1/collections/{collection_id}` | 私密 404；`viewer.is_owner` |
| `GET /api/user/:id/collections` | `GET /v1/users/{user_id}/collections` | 「某人的 X」是 X 的过滤 |
| `GET /api/galgame/:gid/collections/mine` | `GET /v1/me/collections?contains_galgame_id=` | GET 不再创建默认夹 |
| `PUT /api/galgame/:gid/collections` | `PUT /v1/galgames/{galgame_id}/collection-memberships` | 整集替换已经是合法 PUT；body 里的 id 是 collection_id，work 必须先译 catalog id |

已存在的 `GET /v1/galgames/{galgame_id}/moyu-patches` 保持不动，作为 gid→catalog 映射与字符串 id 的样板。
