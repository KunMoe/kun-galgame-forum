# G6 · catalog 用户面

> G 轨第六段（G0 改号之后），2026-09-24。封面投票、游玩时长、收藏夹：**11** 条旧路由。G 号段 140–159；**145 是已用的最后一个号**（G0 140/141、G1 142、G2 143、G1.1 144、G3 145）。本轨无迁移。
> 契约与变异题同一个提交，早于实现。普查全文在 [census/galgame-core.md](census/galgame-core.md) §6–§8、§10–§13。普查写于 G0 之前：作品 id 现已是 catalog work id，路径与字段一律 `work_id` / `{work_id}`（根 CLAUDE.md 铁律 3）；普查 §0 身份、发现 1–3、10,289 撞车、§10「先译 catalog id」全部作废。本契约以代码为准，不沿用普查行号。
> 本轨是论坛作 BFF：调用方自己的 OAuth access token 打 catalog `/v2/me/*` 与 `/v2/folders`。G4 已经在 `Work` 里给出本面的**读侧**（`covers[].vote_count` / `viewer.has_voted`、`playtimes[]`、`viewer.playtime`、`viewer.has_favorited`，K-G29）；本轨只迁**写**和调用者自己的列表，形状与 G4 同名同型。

## 1. 普查（生产实测 2026-09-24，catalog 库 `kun_catalog`；数字按任务书原文）

### 1.1 数字

| 事实 | 数字 |
|---|---|
| `catalog_user_folder` | 12,793 个收藏夹，11,427 个主人；10,964 个默认夹；可见性 `1`（public）11,664，`0`（private）1,129 — **没有第三值** |
| 收藏夹 `item_count` | 最大 3,826；p99 331；中位 2；每一夹的 `item_count` 都与条目行数一致 |
| 每主人收藏夹数 | 最大 41；p99 4 |
| `catalog_user_folder_item` | 273,871 条，11,102 个主人 |
| 收藏夹名 / 描述长度 | 最大 34 / 58 个字符 |
| `catalog_user_playtime` | 367 行，132 个用户；最大 11,340 分钟；4 行低于 10 分钟（已撤回）；每一行 `client_id` 都是论坛的 client；每用户最大 81，p99 60 |
| `catalog_user_work_state` | 1,286 行，324 个用户；(state/completion) `3/3` 602，`1/null` 239，`3/2` 102，`3/null` 90（done 无完成度 — G4 只读 `done`），`3/1` 90，`4/null` 71，`2/null` 68，`5/null` 24 |
| `catalog_cover_vote` | 487 票，364 个用户，287 部作品；`site` 空 370，`kungal` 117 |

论坛库里的 `galgame_collection` **仍是活的别名表**（`id` ↔ `catalog_folder_id`，迁移 091）：内容列冻结，`MintAlias` / `AliasByID` / `AliasesForFolders`（列表时懒铸）/ `DeleteAlias` / `FolderIDsOwnedBy` 仍读写 `id` / `user_id` / `catalog_folder_id`。`galgame_collection_item` 与 `galgame_collection_viewer` 冻结，活路径零读写。任务书「`galgame_collection*` 是冻结快照、活代码永不读写」对后两张表成立，对别名表不成立，见 §8 O1。

### 1.2 调用方

- 封面投票：`apps/web/app/components/galgame/Covers.vue:65`（已投则 DELETE，否则 PUT；读侧已是 G4 的 `WorkCover`）。
- 游玩时长写：`apps/web/app/components/galgame/header/PlaytimeModal.vue:79`（`PUT`，`{minutes?, status?}`；清除是 `minutes: 0, status: ''`）。读侧 `header/Playtime.vue` 吃 G4 的 `playtimes` / `viewer.playtime`。
- 游玩时长列表：`apps/web/app/components/user/Playtime.vue:11`，仅本人资料页 `pages/user/[id]/playtime.vue`（别人看到 `KunNull`）。
- 收藏夹创建 / 编辑：`apps/web/app/components/galgame/collection/EditModal.vue:70` / `:82`。
- 收藏夹详情 / 删除：`apps/web/app/pages/galgame/collection/[id].vue:13` / `:74`。默认夹删除按钮被 `!detail.is_default` 藏掉。
- 收藏夹卡片：`apps/web/app/components/galgame/collection/Card.vue:19`，链到 `/galgame/collection/${id}`。
- 选择器：`apps/web/app/components/galgame/collection/PickerModal.vue:29`（`GET …/collections/mine`）与 `:77`（`PUT …/collections` 整集替换）。`loaded` 为假时保存按钮禁用。`Favorite.vue` 打开它。
- 资料页收藏 tab：`apps/web/app/components/user/CollectionGalgame.vue:13`（`GET /user/:id/collections`），挂在 `pages/user/[id]/collection/[type].vue`。
- Nitro（`apps/web/server/**`）：本段 11 条**零命中**。
- Flutter App：普查 Dart 源零命中；`docs/proj/app-direct-api.md` 未列这 11 条为现网调用。本 worktree 不含 `kungal-apps`，本节按普查 + 文档再核。

校验副本：`apps/web/app/validations/collection.ts` 名称上限 **60**（catalog 是 100）；可见性 `public | private`。空默认夹展示名走 `apps/web/app/utils/collection.ts`（用主人名拼「…的收藏夹」）。

OAuth 现申请 `catalog:edit playtime:read playtime:write folder:read folder:write`（`oauth-auth.ts:56`）。刷新不会给旧会话补上新 scope；infra 可以整表拓宽 grant（2026-09-08 一次 UPDATE 修了 95,286 个会话）。网页旧信封把缺 scope 译成体码 235；v1 的 `SCOPE_REQUIRED` 译文已经是「当前登录缺少所需的授权，请退出登录后重新登录」。

### 1.3 旧面的问题（普查编号 E*，对应 census 发现号）

G0 之后作废、不再建模的：发现 1–3（论坛 gid 当 catalog work_id）、§0 的 10,289 撞车、§9 / §10 的 `CatalogWorkIDForGID` 翻译。封面投票与游玩时长的路径 id **已经是** work_id（`parseWorkID` 读 `:id`，`cover_vote_handler.go:41`，`playtime_handler.go:38`）。

| 编号 | 事实 |
|---|---|
| E1 | `GET …/collections/mine` 曾经 `ensureDefault` 写出未命名公开默认夹（普查发现 12）。**当前代码已经停掉**：`collection_service.go:303-306` 写明 catalog 拒绝空名（deviation 112），空列表回空选择器。v1 GET 仍然不得写 |
| E2 | `GET /user/:id/collections` 与 `GET …/collections/mine` 经 `AliasesForFolders` **懒铸别名**（`collection_alias_repo.go:30-58`）。GET 仍有写。moyu 回填的 3,634 个默认夹从不经过本站 |
| E3 | `PUT /:id/collections` 是整集替换，空数组 = 从全部夹中移除（`collection_service.go:220`）。选择器读失败时 `collection_ids: []` 会把作品从每个夹里抹掉（G4 已经抓到） |
| E4 | 别人的私密夹：详情 404；主人无 OAuth token 时私密详情也失败（`collection_items.go:36`）。`ListForUser` 对封禁主人回 **200 空列表**，详情 404（发现 26） |
| E5 | 缺 `folder:*` / `catalog:edit` /（旧）`playtime:*` → 体码 `403/235`。曾经把 scope 不足折成 205，kunFetch 探针打 `/user/status` 觉得会话还活着，收藏夹整组静默失败（2026-09-08，`collection_err.go:19-26`） |
| E6 | catalog 用户面有过全站 429（BFF 共一个上游 IP；96 小时 8,155 次）。收藏夹把 429 译成 HTTP 429 + 中文句；游玩时长把 429 译成 **`400/233`**（`playtime_handler.go:101`）。都曾经落到 500（用户 90769，2026-09-20） |
| E7 | 撤回游玩时长写 `minutes = 0`；低于 `PlaytimeMinutesFloor`（10）的行存着、不进公开中位数。生产 4 行。catalog **现在有** `DELETE /v2/me/playtimes/{work_id}`（`me_routes.go:94`），旧面不用它 |
| E8 | `play_state` 只读 `"done"`（catalog `state=done` 无 completion，生产 90 行）不能当输入（`playstate.Valid`）。写面 body 字段仍叫 `status` |
| E9 | `/playtime/mine` 先算 `total` / `total_minutes` / `finished_works`，再按 SFW 丢掉卡片（`playtime.go:191-234`）。合计与 `items` 谓词不同。最多扫 10 页 × 100；生产每用户最大 81，扫得完。`finished_works` 不计只读 `"done"` |
| E10 | 创建收藏夹 HTTP 200、裸整数 id、无 `Idempotency-Key`。PATCH/DELETE 回 `OKMessage`。名称校验 1–60（catalog 100） |
| E11 | 首次加入 / 最后一次移除才动本地 `galgame.favorite_count` 和萌萌点（reason `liked`，`KeyNonce`，ref `galgame:<work_id>`），并写 `favorite` 消息。catalog 的 `nextmoe/favorites` 由 catalog 自己算，论坛不写。删夹只减本地计数、不发分；版主删别人的夹不碰计数 |
| E12 | 2026-09-07 收藏导入把条目打进已有的**公开**默认夹；可见性是泄漏面。公开面读别人的夹 |
| E13 | `limit>50` 静默夹（发现 21）。页码集合应 `400 LIMIT_TOO_LARGE` |
| E14 | 选择器 / 成员资格 / 详情 hydrate 仍把路径里的整数当 work_id（G0 之后这是对的）。详情 `galgames[]` 嵌在收藏夹信封里，3,826 条要靠页码切；catalog 条目按 `updated_at` 升序，本面进程内改成 `created_at` 降序（无 id 决胜） |
| E15 | 封面投票 PUT/DELETE 都幂等由上游保证；一部作品一张票，投 B 会把 A 上的票挪走。`Covers.vue:44-54` 客户端自己给其它封面 `voted` 减一。无可见性闸：hidden 作品只要 catalog 认 cover 就能投 |

## 2. 范围与寻址

路径段就是目标。`{work_id}` / `{cover_id}` / `{collection_id}` / `{user_id}` 与响应字段对齐（K1）。作品集合仍是 `/api/v1/works`。收藏夹集合是 **`/api/v1/collections`**，对象 **`collection`**。`collection_id` 是 catalog `catalog_user_folder.id`（十进制字符串），与论坛别名 cid 脱钩（K-G45）。

G17：写路径里每个 `{x_id}`，以它结尾的那段必须有 GET，且 200 带 `id`（`gates/repr.go:272`）。`PUT /works/{work_id}/covers/{cover_id}/vote` 因此需要 **`GET /works/{work_id}/covers/{cover_id}`**——`GET /works/{work_id}` 只覆盖 `{work_id}`；`GET /works/{work_id}/covers` 是列表，顶层没有 `id`。单张封面的 GET 是最小的真读面：200 就是 G4 的 `WorkCover`。

| # | 旧 | v1 | 档 |
|---|---|---|---|
| 1 | `PUT /galgame/:id/cover/:coverId/vote` | `PUT /api/v1/works/{work_id}/covers/{cover_id}/vote`（K16 槽，200） | required |
| 2 | `DELETE /galgame/:id/cover/:coverId/vote` | `DELETE` 同路径（K16，200） | required |
| — | — | `GET /api/v1/works/{work_id}/covers/{cover_id}`（G17，200 `WorkCover`） | optional |
| 3 | `PUT /galgame/:id/playtime` | `PUT /api/v1/works/{work_id}/playtime`（K16 槽，200 `WorkViewerPlaytime`） | required |
| — | — | `DELETE /api/v1/works/{work_id}/playtime`（撤回：上游写 0 + 删 work-state，200） | required |
| 4 | `GET /galgame/playtime/mine` | `GET /api/v1/me/playtimes` 页码集合 | required |
| 5 | `POST /galgame/collection` | `POST /api/v1/collections` → **201** + `Location` + `Collection`；`Idempotency-Key` **必带**（K12） | required |
| 6 | `GET /galgame/collection/:cid` | `GET /api/v1/collections/{collection_id}`（元数据，不嵌作品） | optional |
| 7 | `PATCH /galgame/collection/:cid` | `PATCH /api/v1/collections/{collection_id}` → 200 `Collection` | required |
| 8 | `DELETE /galgame/collection/:cid` | `DELETE` 同路径 → **204** | required |
| — | — | `GET /api/v1/collections/{collection_id}/works` 页码（作品列表从详情拆出） | optional |
| 9 | `PUT /galgame/:id/collections`（整集替换） | `PUT` + `DELETE /api/v1/collections/{collection_id}/works/{work_id}`（每夹一条槽，都 200） | required |
| — | — | `GET /api/v1/collections/{collection_id}/works/{work_id}`（G17；在夹中 → 200 `WorkSummary`，否则 404） | optional |
| 10 | `GET /galgame/:id/collections/mine` | `GET /api/v1/me/collections?work_id=` 页码；每条 `viewer.has_work` | required |
| 11 | `GET /user/:id/collections` | `GET /api/v1/users/{user_id}/collections` 页码，U3 形 | optional |

11 条旧路由全删，`legacy_route_baseline` 下调 **11**。v1 共 **16** 个操作（多了封面 GET、playtime DELETE、收藏夹作品列表、成员 GET）。

代码位置：`internal/galgame/apiv1`（与 `getWork` / 赞槽同包）。`listMyPlaytimes` / `listMyCollections` 的 OpenAPI tag 是 `me`；`listUserCollections` 是 `users`；封面与游玩时长槽是 `works`；其余 `collections`。测试 `internal/app/v1_user_plane_*_test.go`。

## 3. 形状

### 3.1 命名

| 旧 | v1 | 为什么 |
|---|---|---|
| 路径 `:gid` / `:id` / `:coverId` / `:cid` | `{work_id}` / `{cover_id}` / `{collection_id}` / `{user_id}` | K1；`cid` 禁用 |
| 论坛收藏夹 cid（别名） | catalog folder id | 活数据在 catalog；v1 GET 不得再懒铸别名 |
| `voted` | `viewer.has_voted` | K9；与 G4 `WorkCoverViewer` 同名同型 |
| 投票回执 `cover_id` 数字 | `cover_id` 十进制字符串；回执对象 `WorkCoverEngagement` | 01 §3 |
| body `status`（游玩） | `play_state` | 与 G4 `WorkViewerPlaytime.play_state` 同名同型（G8，含只读 `done` + `null`）；handler 拒 `done` |
| `my_playtime` / 写回执 `{minutes, status}` | `WorkViewerPlaytime` | G4 已落地 |
| `/playtime/mine` 的 `galgame` + `status` + `clients` + `truncated` + `finished_works` | `work: WorkSummary` + `play_state` + `client_count` + `is_truncated` + `finished_work_count` | F1；卡片与 `/works` 同形 |
| `user`（夹主人） | `owner: UserRef` | 禁用名 |
| `is_owner` 平铺 | `viewer.is_owner` / `viewer.can_edit` / `viewer.can_delete` | K9；停 `useCan('collection.*')` |
| `contains` | `viewer.has_work` | F1 布尔 |
| `galgames[]` 嵌在详情里 | `GET …/works` 的 `items: WorkSummary[]` | 最大 3,826；详情只留元数据 |
| `preview_covers: string[]`（URL） | `preview_covers: Image[]` | B10 一处一型；最多 4 |
| `created` / `updated` | `created_at` / `updated_at` | 禁用名 |
| 创建 200 裸 id | 201 + `Location` + `Collection` | 01 §5 |
| 整集 `collection_ids: []` | 每夹 `PUT`/`DELETE …/works/{work_id}` | 防盲删（E3） |

`visibility` / `is_default` / `item_count` / `minutes` / `vote_count` / `has_voted` / `has_favorited` / `play_state` / `work_id` / `id` / `viewer` / `is_truncated` 与现 spec 对齐或未占用。

### 3.2 对象

**`WorkCover`**：G4 已落地，本轨 GET 单张原样回。`id` 即路径 `{cover_id}`。

**`WorkCoverEngagement`（`object: "work_cover_engagement"`）** — 投票槽 200：

| 字段 | 类型 | 说明 |
|---|---|---|
| `work_id` | DecimalID | |
| `cover_id` | DecimalID | 与 `WorkCover.id` 同 schema（G8） |
| `vote_count` | int ≥0 | 该封面公开票数；与 `WorkCover.vote_count` 同 schema |
| `viewer` | `{ has_voted: bool }` | required 档，非 null；`has_voted` 与 `WorkCoverViewer` 同型 |

一部作品一张票：PUT 封面 B 成功后，其它封面的 `has_voted` 变 false。回执只带 B；网页继续按 `Covers.vue` 那样给其它封面减一，或再 GET 详情。

**`WorkViewerPlaytime`**：G4 已落地。`minutes` ≥0；`play_state` 枚举 `wish, doing, done_one_route, done_main, done_all, on_hold, dropped, done` 或 `null`。写面请求体的 `play_state` **同一 schema**（G8）；`done` 在 handler 拒。

**`WorkPlaytime`（`object: "work_playtime"`）** — `/me/playtimes` 的条目：

| 字段 | 类型 | 说明 |
|---|---|---|
| `work` | `WorkSummary` | 与 `/works` 条目同形 |
| `minutes` | int ≥0 | 跨 client 取 MAX；低于 10 的当作 0 |
| `play_state` | 与 `WorkViewerPlaytime.play_state` 同 schema | 没有状态为 `null` |
| `client_count` | int ≥0 | 扫到的 `(work, client)` 行数；生产几乎总是 1 |

信封在 `PageList` 上加：`total_minutes`（int ≥0）、`finished_work_count`（int ≥0，只数 `done_one_route` / `done_main` / `done_all`）、`is_truncated`（与月历同 schema：布尔）。`total` / `total_minutes` / `finished_work_count` 与 `items` **同一谓词**（含 `include_nsfw`）。

**`Collection`（`object: "collection"`）** — 详情 200 / 创建 201 / PATCH 200：

| 字段 | 类型 | 空 / 可空 | 来源 |
|---|---|---|---|
| `id` | DecimalID | | catalog folder id |
| `name` | string ≤100 | 可空串（导入的未命名默认夹） | catalog |
| `description` | string ≤500 | 空串 | catalog |
| `visibility` | `private` \| `public` | 永不空 | catalog |
| `is_default` | bool | | catalog |
| `item_count` | int ≥0 | catalog 原值，**不**随 `include_nsfw` 变 | catalog |
| `owner` | `UserRef` | 不可渲染且调用者不是主人 → 整段 404 | `owner_uid` |
| `preview_covers` | `Image[]` | 永不 `null`，最多 4，缺 url 的跳过 | 最早加入的 4 条成员的 banner（catalog 条目 `updated_at` 升序的头 4 条，与现行 `FolderPreviewItems` 相同） |
| `created_at` / `updated_at` | date-time | catalog 解析失败打 WARN，该行丢掉 | catalog |
| `viewer` | `CollectionViewer \| null` | 匿名为 `null` | |

详情**不**嵌作品。作品走 `GET …/works`。

**`CollectionSummary`（`object: "collection"`）**：Collection 去掉不需要画卡的字段后的子集。重叠字段同名同型。列表用它。`preview_covers` 在摘要上保留。

**`CollectionViewer`**：`is_owner`、`can_edit`、`can_delete`、`has_work`。`can_*` 含 staff 时仅 cookie 会话（K2）；Bearer 恒 false。`has_work`：请求带了 `work_id` 且该夹已收这部作品为 true，否则 false。

**`CollectionWorkEngagement`（`object: "collection_work_engagement"`）** — 成员槽 200：`collection_id`、`work_id`、`viewer: { has_work }`（required 档，非 null）。

### 3.3 词表

| 字段 | 取值 |
|---|---|
| `visibility` | `private` `public`（封闭；infra 顺序如此） |
| `play_state` | G4 八值 + `null`：`wish` `doing` `done_one_route` `done_main` `done_all` `on_hold` `dropped` `done`。写面 handler 拒 `done` |
| 写面 `minutes` | 0–60000 的整数；schema 与 `WorkViewerPlaytime.minutes` 同形（≥0，无 maximum，G8），越界在 handler 用 `OUT_OF_RANGE` `{minimum:0, maximum:60000}` 拦 |

未知封闭枚举 → `400 UNKNOWN_ENUM_VALUE`。缺席 = 不过滤。

### 3.4 封面投票槽（K-G42）

K16：`PUT` 置位、`DELETE` 撤销，都幂等，都 200 `WorkCoverEngagement`。空体。

`cover_id` 是 G4 `WorkCover.id`（`catalog_work_cover.id`）。封面必须属于这条 `{work_id}`：属于别的作品、或不存在 → `404 NOT_FOUND`（不透露封面在哪部作品上）。作品 catalog 不认识或 hidden → 404。

一部作品一张票，由 catalog 保证（`me_routes.go:125` "One ballot per work"）。同一封面再 PUT 是 200、票数不变。未投再 DELETE 是 200、`has_voted: false`（catalog DELETE 对缺席是 204，论坛补票数）。

缺 `catalog:edit` → `403 SCOPE_REQUIRED`（infra `identity.go:117-119` 把 `/v2/me/cover-votes` 放在 editing plane）。读侧仍按 G4：缺 scope / 用户面 401 降到公开计数、`has_voted: false`。

实现回执的 `vote_count`：catalog 的 `CoverVote` 表示没有 `vote_count`（`repr/me.go:132-138`）。PUT 之后再读该作品的公开票仓（现行 `WorkCoverVotes`），填回执。失败 503。

### 3.5 游玩时长（K-G43、K-G44）

**槽** `PUT` / `DELETE /works/{work_id}/playtime`。路径没有第二段 `{id}`，G17 只要求已有的 `GET /works/{work_id}`。

`PUT` 体：

```json
{ "minutes": 120, "play_state": "doing" }
```

两字段都是指针：缺席 = 不动那一轴；至少一个必须出现，否则 `400 INVALID_PARAMETER` `REQUIRED`。`minutes = 0` 是撤回时长（上游 `PUT {minutes:0}`）。`play_state: null` 是删除 work-state（上游 `DELETE /v2/me/work-states/{work_id}`，404 当成功）。`play_state: "done"` → `422 VALIDATION_FAILED` `NOT_ALLOWED_VALUE`（schema 因 G8 含 `done`，handler 拒）。

catalog 两轴映射仍只在 `internal/galgame/playstate`：七个可写值 ↔ `state` + `completion`；`wish`/`doing`/`on_hold`/`dropped` 不带 completion。写完再读折叠分钟（其它 client 可能更大），回 `WorkViewerPlaytime`。低于 10 的分钟对外是 0，状态仍在。分钟与状态都空则 `minutes: 0, play_state: null`（网页当清除）。

`DELETE` 撤回本站这一条：**上游写 `minutes = 0`，并删除 work-state**。与清除按钮同一语义。catalog 虽有真 DELETE（`me_routes.go:94`），本轨不用——公开中位数靠「低于 floor 的行仍在表里」识别撤回；真删会让「曾经报过」消失。变异 #5 钉住。

作品不可报（hidden / 未知）→ 404。infra 对 playtime **不再要** `playtime:write`（`me_routes.go:90`「Any app may call this」）；论坛仍申请该 scope。若上游仍回 `SCOPE_REQUIRED`（旧会话、或以后改回），映射 `403 SCOPE_REQUIRED`。

**`GET /me/playtimes`**：页码，`page` 默认 1，`limit` 默认 24 最大 100。生产每用户最大 81，一页 100 即全量。`include_nsfw` 默认 `false`。服务端扫 `/v2/me/playtimes` 与 `/v2/me/work-states`（帽仍 10×100），触顶 `is_truncated: true` + WARN。折叠：MAX 分钟、`client_count`、状态并上。排序：playtime 行按扫到的最后下标倒序（「最近改动」代理，catalog 是 `updated_at` 升序），只有状态的排在后面。不暴露 `sort=`。

SFW：NSFW 作品从 `items` **和** `total` / `total_minutes` / `finished_work_count` 一起去掉（修 E9）。个人页只对本人；没有 `/users/{user_id}/playtimes`。

### 3.6 收藏夹（K-G45、K-G46、K-G48）

创建：`name` 必填，原始值 1–100 字符（K19，对齐 infra，不再 60）；`description` ≤500，缺席 = 空串；`visibility` **必填** `private`/`public`（不默认公开，E12）；`is_default` 可选，缺席 false。catalog 把空 `visibility` 当 private（`me_folder.go:25`）；本面不把缺席当 private，缺席 → `REQUIRED`。`is_default: true` 会清掉上一任默认旗；`false` 在 PATCH 上 catalog 拒（`ErrFolderDefaultOnly`）→ `422 VALIDATION_FAILED`。

trust：创建与 PATCH 只扫本次提交的文本（K18）。`DecisionDeny` → `422 CONTENT_REJECTED`。Hold 仍创建并后台扫描。

删除：主人删默认夹 → `409 INVALID_STATE_TRANSITION`（先把默认旗移走）。staff（cookie + `user.Can(perm.CollectionDeleteAny)`）可以删别人的默认夹（catalog 版主面接受）。DELETE 204。别名行：v1 不写 `galgame_collection`；旧路由删掉之后别名表不再有读者，删表留给后面某段。

别人的私密夹、不存在的 id、封禁主人且调用者不是本人：一律 **`404 NOT_FOUND`**（存在性不披露）。`ListForUser` 旧的「封禁主人 200 空列表」改成 404，与 U3 / 详情对齐。OAuth 查主人失败 → 503，fail-closed。

`GET /users/{user_id}/collections`：U3 形。页码，`limit` 默认 24 最大 100。主人不可渲染 404。调用者不是主人时只含 `visibility=public`。本人 + token：`/v2/me/folders`（含私密）。本人配额耗尽不再静默掉到公开面（旧 `listFoldersForViewer`），而是 `503 SERVICE_UNAVAILABLE`（§3.15）。`include_nsfw` 默认 false，只影响 `preview_covers` 的水合。

`GET /me/collections`：required。本人全部夹，含私密。可选 `work_id`：每条 `viewer.has_work`。无 `work_id` 时 `has_work` 恒 false（K10 静态形状）。排序：`is_default` 先，然后 `updated_at DESC, id DESC`。生产每主人最大 41，一页 24 够分页器。GET **不**创建默认夹、**不**铸别名。

`GET /collections/{collection_id}/works`：页码，默认 24 最大 100。深度 `page × limit ≤ 10000`（最大夹 3,826，够用）。排序 `created_at DESC, work_id DESC`（补 id 决胜）。`include_nsfw` 默认 false：NSFW 作品从 `items` 与 `total` 一起去掉；`Collection.item_count` 仍是 catalog 原值。水合失败的 id 丢掉并 WARN，`total` 不减（与 G5 K-G40 同一纪律）。

公开夹走 catalog 应用密钥 `/v2/folders`（省用户面配额，2026-09-20 user 90769）。私密夹只有主人带 token 走 `/v2/me/folders/{id}`。6 小时 Redis 缓存可留，键含 `folder.UpdatedAt`；v1 语义不依赖它。

### 3.7 成员资格（K-G47）

没有整集替换。每条成员是槽：

- `PUT /collections/{collection_id}/works/{work_id}` 加入（catalog `PUT /v2/me/folders/{id}/items/{work_id}`，已在则原样，幂等）
- `DELETE` 移出（catalog DELETE，不在也是 204 → 论坛 200 `has_work: false`）

夹必须属于调用者。别人的夹、staff 改别人的成员：`404 NOT_FOUND`（旧面 403「不属于您」会披露存在；本面不披露）。作品 hidden / 未知：404。夹满 10,000 → `422 VALIDATION_FAILED`。

选择器：`GET /me/collections?work_id=` 画出勾选；保存时对差集逐条 PUT/DELETE。读失败则 `loaded=false`，页面已禁用保存。服务端没有「空数组 = 清空」这条路。

`GET …/works/{work_id}`：在夹中 → 200 `WorkSummary`（`id` 即 `{work_id}`，过 G17）；否则 404。

### 3.8 副作用（K-G50）

| 事件 | 本地 `galgame.favorite_count`（「最多收藏」排行） | 萌萌点 | `favorite` 消息 | catalog `nextmoe/favorites` |
|---|---|---|---|---|
| 某作品从「零个夹」到「至少一个夹」 | +1（`Ensure` 本地行，不碰 `published`） | 给作品 `creator_user_id` +1，reason `liked`，`KeyNonce`，ref `galgame:<work_id>`；主人是 0 / 是调用者则跳过 | 写一条，同一对 (sender, receiver, link) 已有则跳过 | catalog 自己算，论坛不写 |
| 某作品从「至少一个夹」到「零个夹」 | `GREATEST(−1, 0)` | −1，同一 ref | 不写 | catalog 自己算 |
| 在已有收藏上再加 / 再减其它夹 | 不动 | 不动 | 不动 | catalog 自己算 |
| 主人删夹，若干作品因此离开书库 | 这些 work_id −1 | 不动（旧面如此） | 不动 | catalog 自己算 |
| staff 删别人的夹 | 不动（读不到内容） | 不动 | 不动 | catalog 自己算 |
| 创建 / 改元数据 / 封面投票 / 游玩时长 | 不动 | 不动 | 不动 | 不动 |

本地事务失败只记 ERROR，上游已经写完（旧面如此）。萌萌点走 pusher 异步。

### 3.9 默认夹从哪来（K-G48）

catalog **不会**在读时创建默认夹。`POST /v2/me/folders` 拒绝空名；未命名默认夹只来自 2026-09-07 回填（生产 10,964 个）。v1：

- GET 任何收藏夹面零写入。
- 新用户一个夹都没有：`GET /me/collections` 空列表；选择器走已有的「还没有收藏夹，点击新建」。
- 网页从空选择器创建第一条时带 `is_default: true`。已有默认夹时再带 true 会把旗移过来。

### 3.10 NSFW 与可见性

收藏夹本身没有 NSFW 旗。`include_nsfw` 只过滤夹里的作品卡片与预览图，默认 `false`。私密夹对非主人 404。公开夹匿名可读。hidden 作品仍能被夹着：公开条目列表「应用自己的编辑闸再水合」（catalog `listPublicFolderItems` 的说明）；本面 SFW 读者看不到 NSFW 作品卡。

封面投票 / 游玩时长写不按本地 `published` 闸。catalog 拒 hidden 则 404。

### 3.12 实现时按 G8 / G14 / F1 改的名（只增不改，以此为准）

| 前文 / 旧 | 实际 | 撞了谁 |
|---|---|---|
| 写面 body `status` | `play_state` | G8：`status` 是 Problem 的整数；且必须与 G4 `WorkViewerPlaytime.play_state` 同 schema（含 `done` + `null`） |
| `contains` | `viewer.has_work` | F1 布尔 |
| `clients` | `client_count` | F1 `_count` |
| `finished_works` | `finished_work_count` | F1 `_count` |
| `truncated` | `is_truncated` | F1；与 G5 月历同名同型（布尔） |
| `preview_covers: string[]` | `preview_covers: Image[]` | B10 |
| 投票回执平铺 `voted` | `WorkCoverEngagement.viewer.has_voted` | K9 |
| 收藏夹 `user` | `owner` | 禁用名 |
| `is_owner` 平铺 | `viewer.is_owner` | K9 |
| 名称 maxLength 60 | 100 | 对齐 infra；G4.1 课 |

`minutes` 不在 schema 上加 `maximum: 60000`：`WorkPlaytimeAggregate.minutes` / `WorkViewerPlaytime.minutes` 都没有上限。越界走 handler `OUT_OF_RANGE`。

`visibility` 现 spec 未占用。`item_count` 与 `NewsMonth.item_count` 同 schema（int64，≥0）。`is_default` 未占用。

每个字符串：封闭枚举、format、pattern 或自由文本句（G14）。`name` / `description` 是自由文本。

### 3.13 对 infra 的词表差（G4.1 课，逐行）

每一个本轨传给或读自 catalog 的封闭词表 / 长度上限，对 infra 的定义比，不对 dev 数据恰好有的值比。有意丢行的分支打 WARN。

| 字段 | infra 的集合 / 上限（文件:行） | 本轨 | 裁定 |
|---|---|---|---|
| 收藏夹 `visibility` | `private,public`（`repr/me.go:147`；解码 `me_folder.go:23-30`，空串 → private） | 同两值；创建必填，不把缺席当 private | 对齐闭集。第三值（生产 0）拒 `UNKNOWN_ENUM_VALUE` |
| 收藏夹 `name` | maxLength **100**，minLength 1，trim 后空拒（`repr/me.go:145`，`me_folder_routes.go:22`，`user_folder.go:53-56` `FolderNameMax`） | schema ≤100；trim 后空 → `TOO_SHORT` | 对齐。旧面 60 是收窄，生产最大 34，仍抬到 100 |
| 收藏夹 `description` | ≤500（`repr/me.go:146`，`model/folder.go:12`） | ≤500 | 对齐。生产最大 58 |
| 每用户夹数 | 200（`model/folder.go:13`，`user_folder.go:70`） | 超限透传 422 | 对齐。生产最大 41 |
| 每夹条目 | 10,000（`model/folder.go:14`，`user_folder.go:244`） | 超限透传 422 | 对齐。生产最大 3,826 |
| `is_default` | 至多一个；PATCH 只接受 true（`me_folder_routes.go:40`，`ErrFolderDefaultOnly`） | 同 | 对齐。发 false → 422 |
| work `state` | `wish,doing,done,on_hold,dropped`（`repr/me.go:196`） | 经 `playstate.ToCatalog` / `FromCatalog` 压成扁平八值 | 闭集的真子集映射。未知 `state` → `play_state: null` + WARN |
| work `completion` | `one_route,main,all` 或 null；「Never accompanies wish」（`repr/me.go:197`） | 只随三个 `done_*` 发送 | 对齐。未知 completion 且 state=done → 只读 `done`（生产 90 行） |
| playtime `minutes` | schema 0–60000（`me_routes.go:30`）；存储接受 0–60000（`user_playtime.go:191`）；聚合只计 **10–60000**（`model/playtime.go:5-7` `PlaytimeMinutesMin/Max`，`jobs/userplaytime/aggregate.go:134`） | 写 0–60000；读到 &lt;10 对外当 0 | 对齐。撤回写 0，不用 catalog DELETE |
| playtime 列表分页 | 游标，limit 1–100 默认 20（`collect/query.go:12`，`parse.MaxPageLimit=100`） | 对客户端仍是页码；服务端扫游标 | 不把 catalog 游标暴露给网页 |
| 封面投票 `vote` | 只 `up`（`repr/me.go:137`，`me_cover.go:41`） | 本面空体，服务端发 `up` | 不把 `vote` 暴露给客户端 |
| 封面投票 scope | `catalog:edit`（`identity.go:117-119`） | 缺则 `SCOPE_REQUIRED` | 对齐 |
| 收藏夹 scope | GET：`folder:read` 或 `folder:write`；写：`folder:write`（`identity.go:154-168`） | 同，缺则 `SCOPE_REQUIRED` | 对齐。公开 `/v2/folders` 走应用密钥，不要用户 token |
| playtime scope | **不要** `playtime:write`（`me_routes.go:90`） | 上游若仍拒再映射 `SCOPE_REQUIRED` | 不收窄；论坛继续申请该 scope |
| 公开夹 404 | 私密与不存在都是 404（`folder_public_routes.go:27-29`） | 同 | 对齐 |
| 条目列表闸 | 公开条目「应用自己的编辑闸再水合」（`folder_public_routes.go:51`） | `include_nsfw` 默认 false | 不把 catalog 的年龄轴暴露给客户端 |

仍然有意丢掉、且打 WARN 的：水合缺行的作品、解析失败的时间戳、没有 url 的预览图、未知 work `state` 的行（`play_state` 变 null，分钟仍在）、cover 票仓读失败时 GET 单张封面的 `vote_count=0` / `has_voted=false`（与 G4 详情降级一致）。

### 3.14 上游错误映射

catalog 用户面一次调用的失败，按下表进 v1。**不得**落到无 code 的 500。

| 上游 | v1 | 说明 |
|---|---|---|
| 缺 scope（`ErrInsufficientScope` / 403 `SCOPE_REQUIRED`） | `403 SCOPE_REQUIRED` | 旧会话、grant 未拓宽。网页用已有 zh-CN：「请退出登录后重新登录」。infra 可以整表拓宽，用户侧这句话仍是他能做的那一步 |
| 会话里没有 access token | `401 INVALID_CREDENTIAL` | cookie 还在、Redis 里没有 OP token |
| token 被拒（`ErrUnauthorized` / 401） | `401 INVALID_CREDENTIAL` | |
| 上游 403 非主人 / 非版主 | `404 NOT_FOUND` | 存在性不披露。本人对自己的夹被拒才是 `403 PERMISSION_REQUIRED` |
| 404 | `404 NOT_FOUND` | 含别人的私密夹 |
| 409（幂等键） | `409 IDEMPOTENCY_KEY_REUSED` / `IDEMPOTENCY_REQUEST_IN_PROGRESS` | 本面只有 POST 创建带键 |
| 422 默认夹不能删 | `409 INVALID_STATE_TRANSITION` | |
| 422 名称空 / 可见性非法 / 夹满 / 夹数满 / `is_default: false` | `422 VALIDATION_FAILED`（对应 `TOO_SHORT` / `UNKNOWN_VALUE` / `OUT_OF_RANGE`） | 中文不进 `detail`（K8） |
| 429（任何原因：用户面日配额、BFF 共 IP 限流） | `503 SERVICE_UNAVAILABLE`，转发 `Retry-After`，WARN 带上游状态 | §3.15；不嗅正文 |
| 5xx / `ErrUpstream` | `503 SERVICE_UNAVAILABLE` | |
| 传输失败 / 未配置 | `503 SERVICE_UNAVAILABLE` | |
| 上游 400 游玩时长越界 | `400 INVALID_PARAMETER` `OUT_OF_RANGE` | 本面 handler 应先拦住 |

### 3.15 编排者裁决（2026-09-24，覆盖前文同名条目）

**收藏夹 id 与页面路径（方案 B）。** v1 `collection_id` 是 catalog folder id，一个身份（与铁律 3 同理）。别名表 `galgame_collection` **冻结**：不再铸新行，只作「旧 cid → 现 folder id」的重定向表；本轨删掉 `MintAlias` / `AliasesForFolders` 的活调用（旧路由一并删）。

- **新页面路径 `/collection/{collection_id}`**：顶层 `collection` 段，永远不会被读成旧的 `/galgame/collection/{cid}`（两个 id 空间都是从 1 起的 serial，撞号 4,079 个，旧路径不能复用给 folder id）。卡片、选择器、资料页一律链到新路径。
- **别名查询** `GET /api/v1/collection-aliases/{alias_id}` → 200 `CollectionAlias`（`object: "collection_alias"`、`alias_id`、`collection_id`）。optional 档，只读，**永不创建**。未知 cid → 404。**可见性与 `GET /collections/{collection_id}` 相同**：目标夹对调用者不可见（别人的私密夹、主人不可渲染）→ 404，别名本身不披露存在性。G17：路径只有 GET，无写。
- **旧路径两种到达方式都要通**：整页加载 `/galgame/collection/{cid}` → Nitro server route 调别名查询 → **301** 到 `/collection/{collection_id}`（不可见 / 未知 → 404 页）；站内客户端导航（旧 markdown 里的链接）→ 保留一个薄页 `pages/galgame/collection/[id].vue`，只调别名查询后 `router.replace`，查询完成前不渲染任何夹内容（不闪别人的夹）。
- **库里存着的旧链接：0 条。** 普查（2026-09-24，生产只读）：G0 列出的全部带链接列——`message.link` / `message.content`、`feed_activity.link` / `.content`、`chat_message.content`、`chat_room.last_message_content`、`topic.content`、`topic_reply.content`、`topic_comment.content`、`todo.content`、`doc_article.content_markdown`、`galgame_resource_link.url`——以及 infra `kun_community.community_post.content_raw`，`/galgame/collection/<n>` 出现 **0 次**，distinct cid **0** 个（收藏通知链的是 `/galgame/<work_id>`）。所以**没有**链接改写迁移；重定向只承载站外分享与书签。

**游玩时长撤回仍写 0（编排者复核后改判，2026-09-24）。** catalog 的时长是**每 client 一行**：`UNIQUE (actor_uid, work_id, client_id)`（生产索引 `uq_catalog_user_playtime`；infra `model/playtime.go:13-15`；PUT 按这三列 upsert，`user_playtime.go:57-58`）。而 `DELETE /v2/me/playtimes/{work_id}` 是 `DELETE … WHERE actor_uid AND work_id`，**不带 client**（`user_playtime.go:146-155`），也没有按 client 删的端点。用它撤回会把该用户在 moyu / App 等其它 client 报的时长一起抹掉。所以 `DELETE /works/{work_id}/playtime` = 用论坛自己的 token `PUT {minutes: 0}`（只动论坛这一行，低于 floor 不进公开中位数）+ 删 work-state。重复撤回是 200。生产 367 行目前都是论坛的 client，改判前没有数据受损。

**撤回的回执不写死 `{minutes: 0}`。** catalog 的「我的时长」是**跨 client 取最大值**（infra#296 之后仍然如此），论坛这一行归零或删掉之后，该用户在别的 app 报的分钟仍在。DELETE 与 PUT 一样：写完**重读**（`MyPlaytime` + `MyWorkState` 折叠），回的就是此刻 `GET /works/{work_id}` 的 `viewer.playtime` 会给的值；两轴都空才是 `null`。变异 #21 钉住。

**以后换成 DELETE。** infra#296（spec 2.24.1）把 `DeleteMine` 收窄到调用方的 `client_id` 并拒绝空 client。本轨不做运行时版本判断，仍写 0；#296 在生产部署并验证之后，另开一个小 PR 把撤回换成 DELETE（不留 0 行），回执语义不变。

**上游 429 一律 `503 SERVICE_UNAVAILABLE`，转发 `Retry-After`，不嗅正文。** 打一行 WARN，带上游状态码。前文 `QUOTA_EXCEEDED` 的拆分作废：日配额与共享 IP 限流都不是调用者这一次请求的错，正文子串也不是可靠判据。

## 4. 逐条操作

通用：401 `MISSING_CREDENTIAL` / `INVALID_CREDENTIAL`（required；optional 的坏 Bearer）、403 `ACCOUNT_BANNED`、500 `INTERNAL_ERROR`、503 `SERVICE_UNAVAILABLE`（会话存储 / userclient / catalog / 传输）。每个 401 带 `WWW-Authenticate`。v1 全部 `Cache-Control: no-store`。缺 OAuth token 的已登录会话：写与私密读走 401 `INVALID_CREDENTIAL`。

### 4.1 `getWorkCover` · `GET /works/{work_id}/covers/{cover_id}` · optional · 200 `WorkCover`

G17 为投票槽准备的读面。封面必须属于该作品。票仓失败时 `vote_count=0`，登录者 `has_voted=false`，打 WARN，仍 200。

| 状态 | code |
|---|---|
| 401 | 坏 Bearer |
| 404 | 作品 / 封面不存在、hidden、封面不属于该作品、`ENTITY_MERGED` |
| 503 | catalog |

### 4.2 `putWorkCoverVote` · `PUT /works/{work_id}/covers/{cover_id}/vote` · required · 200 `WorkCoverEngagement`

空体。已投再 PUT 是 200、票数不变。一部作品一张票。

| 状态 | code |
|---|---|
| 403 | `SCOPE_REQUIRED`（缺 `catalog:edit`）`ACCOUNT_BANNED` |
| 404 | 作品 / 封面不存在、hidden、封面不属于该作品 |
| 503 | catalog / 票仓 |

### 4.3 `deleteWorkCoverVote` · `DELETE /works/{work_id}/covers/{cover_id}/vote` · required · 200 `WorkCoverEngagement`

未投再 DELETE 是 200、`has_voted: false`。404 / 403 同 4.2。

### 4.4 `putWorkPlaytime` · `PUT /works/{work_id}/playtime` · required · 200 `WorkViewerPlaytime`

体见 §3.5。`play_state: "done"` → 422。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（两字段都缺、`minutes` 越界、id） |
| 422 | `VALIDATION_FAILED`（`done`）`UNKNOWN_ENUM_VALUE` |
| 403 | `SCOPE_REQUIRED` `ACCOUNT_BANNED` |
| 404 | 作品不可报 |
| 503 | catalog / 共享 429 |

### 4.5 `deleteWorkPlaytime` · `DELETE /works/{work_id}/playtime` · required · 200 `WorkViewerPlaytime`

用论坛 token `PUT {minutes:0}` 并删除 work-state（§3.15；不用 catalog DELETE，它不分 client）。回 `{minutes:0, play_state:null}`。404 同 4.4。

### 4.6 `listMyPlaytimes` · `GET /me/playtimes` · required · 200 页码信封（§3.2）

Query：`page`、`limit`、`include_nsfw`（默认 false）。

| 状态 | code |
|---|---|
| 400 | `LIMIT_TOO_LARGE` `INVALID_PARAMETER`（深度） |
| 403 | `SCOPE_REQUIRED` `ACCOUNT_BANNED` |
| 503 | catalog / 共享 429 |

### 4.7 `createCollection` · `POST /collections` · required · 201 `Location` + `Collection`

`Idempotency-Key` 必带。`Location`：`/api/v1/collections/{collection_id}`。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（缺幂等键 / 格式） |
| 409 | `IDEMPOTENCY_KEY_REUSED` `IDEMPOTENCY_REQUEST_IN_PROGRESS` |
| 422 | `CONTENT_REJECTED` `VALIDATION_FAILED`（夹数满、名称） |
| 403 | `SCOPE_REQUIRED` `ACCOUNT_BANNED` |
| 503 | catalog / 共享 429 |

### 4.8 `getCollection` · `GET /collections/{collection_id}` · optional · 200 `Collection`

私密且非主人、不存在、主人不可渲染且非本人 → 404。

| 状态 | code |
|---|---|
| 401 | 坏 Bearer |
| 404 | 见上 |
| 503 | userclient / catalog |

### 4.9 `updateCollection` · `PATCH /collections/{collection_id}` · required · 200 `Collection`

部分更新。空体 → 400。非主人：cookie + `user.Can(perm.CollectionEditAny)` 走 catalog `/v2/moderation/folders/{id}`；Bearer 与无能力者 404。staff 不得改 `is_default`（catalog 版主面无此字段）。只扫提交的文本。

### 4.10 `deleteCollection` · `DELETE /collections/{collection_id}` · required · 204

主人删默认夹 → 409。staff 删除走 `/v2/moderation/folders/{id}`，cookie-only。Bearer 当非 staff。

### 4.11 `listCollectionWorks` · `GET /collections/{collection_id}/works` · optional · 200 `PageList<WorkSummary>`

可见性同 4.8。Query：`page`、`limit`、`include_nsfw`。

| 状态 | code |
|---|---|
| 400 | `LIMIT_TOO_LARGE` `INVALID_PARAMETER` |
| 404 | 同 4.8 |
| 503 | catalog / 水合 |

### 4.12 `getCollectionWork` · `GET /collections/{collection_id}/works/{work_id}` · optional · 200 `WorkSummary`

G17。不在夹中 → 404。`include_nsfw=false` 且作品 NSFW → 404（与列表同一谓词）。

### 4.13 `putCollectionWork` · `PUT /collections/{collection_id}/works/{work_id}` · required · 200 `CollectionWorkEngagement`

加入。已在再 PUT 是 200、`has_work: true`。夹必须是调用者的。首次加入书库时发 §3.8 的本地副作用。

| 状态 | code |
|---|---|
| 403 | `SCOPE_REQUIRED` `ACCOUNT_BANNED` |
| 404 | 夹 / 作品 |
| 422 | 夹满 |
| 503 | catalog / 共享 429 / 萌萌点上游 |

### 4.14 `deleteCollectionWork` · `DELETE /collections/{collection_id}/works/{work_id}` · required · 200 `CollectionWorkEngagement`

不在再 DELETE 是 200、`has_work: false`。最后一次移出书库时发 §3.8。

### 4.15 `listMyCollections` · `GET /me/collections` · required · 200 `PageList<CollectionSummary>`

Query：`page`、`limit`、`work_id`（可选）、`include_nsfw`。GET 零写入。

### 4.16 `listUserCollections` · `GET /users/{user_id}/collections` · optional · 200 `PageList<CollectionSummary>`

U3：主人不可渲染 404；非主人只有公开夹。

| 状态 | code |
|---|---|
| 401 | 坏 Bearer |
| 404 | 主人不可渲染 / 未知用户 |
| 503 | userclient / catalog |

## 5. 预分配

### 5.1 迁移

**无。** G 号段下一个空号是 **146**，本轨不用。`galgame_collection` 别名表本轨停写，不 DROP（旧 URL 与回滚材料还在）。`galgame_renumber_2026` 删表仍按 g-plan 留给后面某段。本地 `galgame.favorite_count` 列已有。

### 5.2 错误码

不新增。复用：

- **`SCOPE_REQUIRED`**（platform，403）。缺 `catalog:edit` / `folder:read` / `folder:write`。
- **`CONTENT_REJECTED`**（kungal，422）。
- **`INVALID_STATE_TRANSITION`**（me，409）。默认夹不能删。
- **`ENTITY_MERGED`**（作品路径）。

其余：`NOT_FOUND`、`PERMISSION_REQUIRED`、`ACCOUNT_BANNED`、`SERVICE_UNAVAILABLE`、`INVALID_PARAMETER`（`REQUIRED` / `INVALID_FORMAT` / `OUT_OF_RANGE` / `TOO_SHORT` / `TOO_LONG` / `TOO_MANY_ITEMS`）、`VALIDATION_FAILED`（`NOT_ALLOWED_VALUE` / `UNKNOWN_VALUE` / `OUT_OF_RANGE`）、`UNKNOWN_ENUM_VALUE`、`LIMIT_TOO_LARGE`、`IDEMPOTENCY_KEY_REUSED`、`IDEMPOTENCY_REQUEST_IN_PROGRESS`。

共享 IP 的上游 429 走 **`SERVICE_UNAVAILABLE`**，不是新码。

### 5.3 K 决定

| 编号 | 决定 |
|---|---|
| **K-G41** | 本轨是 BFF：用户 token 打 catalog `/v2/me/*` 与 `/v2/folders`。论坛库不存票、时长、夹内容 |
| **K-G42** | 封面投票是 K16 槽，见 §3.4。G17 的读面是 `GET /works/{work_id}/covers/{cover_id}` → `WorkCover` |
| **K-G43** | 游玩时长槽见 §3.5 / §3.15。DELETE = 论坛 token `PUT {minutes:0}`（每 client 一行；infra#296 部署前 catalog 的 DELETE 会删掉其它 client）+ 清状态；回执重读（跨 client 最大值），不写死 0。`play_state` 与 G4 同 schema，handler 拒 `done` |
| **K-G44** | `/me/playtimes` 页码；合计与 `items` 同一谓词；生产每用户 ≤81 |
| **K-G45** | `collection_id` = catalog folder id；页面 `/collection/{collection_id}`；别名表冻结为重定向表，`GET /collection-aliases/{alias_id}` 只读、可见性同详情；旧路径 Nitro 301 + 薄页 `router.replace`（§3.15） |
| **K-G46** | 别人的私密夹 404。封禁主人的公开夹对路人 404 |
| **K-G47** | 成员资格是每夹每作品一条槽，没有整集替换，见 §3.7 |
| **K-G48** | v1 GET 零写入：不建默认夹、不铸别名。默认夹来自 catalog 回填或创建时的 `is_default` |
| **K-G49** | 缺 scope → `403 SCOPE_REQUIRED`。上游任何 429 → `503 SERVICE_UNAVAILABLE` + 转发 `Retry-After` + WARN，不嗅正文（§3.15）。永不 500 |
| **K-G50** | 副作用见 §3.8。保留本地收藏计数、萌萌点、`favorite` 消息；不写 catalog favorites 指标 |
| **K-G51** | 收藏夹 staff 改删仅 cookie + `user.Can`；Bearer 永不持有（K2） |
| **K-G52** | 名称 / 描述 / 可见性 / 分钟 / 状态词表对齐 infra（§3.13），不按旧 DTO 60 字收窄 |

权限不新增。`collection.edit_any` / `collection.delete_any` 沿用，经 `viewer.can_*` 下发。

## 6. 网页

切到生成的类型化客户端；手写 `CollectionDetail` / `MyCollectionForGalgame` / `PlaytimeMinePage` 改成生成物别名或删除。`legacy-fetch-baseline` 按删掉的调用点下调。`Idempotency-Key` 创建收藏夹必带（把目标算进键）。`SCOPE_REQUIRED` 的 zh-CN 已在 `problem.json`：toast 请重新登录（与旧 235 同一句话）。`SERVICE_UNAVAILABLE` 已有译文（上游 429 也走它）。按钮显隐读 `viewer.can_*`。

| 文件 | 改什么 |
|---|---|
| `Covers.vue` | `PUT`/`DELETE /works/{id}/covers/{cover_id}/vote`；读回 `vote_count` / `viewer.has_voted`；`cover_id` 是字符串 |
| `header/PlaytimeModal.vue` | `PUT /works/{id}/playtime` 体 `{minutes, play_state}`；清除改 `DELETE`；读回 `WorkViewerPlaytime`；`done` 仍只在选择器里映成 `done_one_route` |
| `header/Playtime.vue` | 读侧已是 G4；写随 Modal |
| `user/Playtime.vue` | `GET /me/playtimes`；`work.display_name`；`play_state`；`finished_work_count`；`is_truncated` |
| `pages/user/[id]/playtime.vue` | 继续只给本人挂组件 |
| `collection/EditModal.vue` | `POST /collections`（201 + 资源，幂等键；名称上限 100；可带 `is_default`）/ `PATCH /collections/{id}`；id 改字符串 |
| `collection/PickerModal.vue` | `GET /me/collections?work_id=`；保存对差集逐条 PUT/DELETE `…/works/{work_id}`；`loaded` 为假仍禁用保存；链接 `/collection/{collection_id}` |
| `Favorite.vue` | 选择器保存后仍翻转本地心；读侧已是 G4 `has_favorited` |
| `pages/collection/[id].vue`（新） | 收藏夹详情页：`GET /collections/{id}` + `GET …/works?include_nsfw=`；删改读 `viewer.can_*`（停 `useCan`）；DELETE 204 |
| `pages/galgame/collection/[id].vue` | 改成薄页：只调 `GET /collection-aliases/{id}`，成功后 `router.replace` 到 `/collection/{collection_id}`，404 渲染空态；查询完成前不渲染夹内容 |
| `server/routes/galgame/collection/[id].ts`（新，或等效 route rule） | 整页加载旧路径：调别名查询 → 301 `/collection/{collection_id}`；404 → 404 |
| `collection/Card.vue` | `preview_covers[].url`；href `/collection/{collection_id}` |
| `user/CollectionGalgame.vue` | `GET /users/{user_id}/collections` |
| `pages/user/[id]/collection/[type].vue` | 随 CollectionGalgame |
| `validations/collection.ts` | 名称 max 100 |
| `utils/collection.ts` | 未命名默认夹仍用主人名；`owner.name` 可 null |
| `docs/proj/app-direct-api.md` | 投票 / 时长 / 收藏夹改上表路径（前瞻） |

旧 `/galgame/collection/{cid}` 分享链接：按 §3.15 经别名查询 301 / `router.replace` 到 `/collection/{collection_id}`。库内存着的旧链接 0 条，无改写迁移。

## 7. 变异题（先于实现提交）

种子：一部 catalog 作品带两张封面；一部 hidden；一个未知 work / cover / folder id；一个公开夹、一个私密夹（另一用户）；调用者自己的默认夹 + 一个非默认夹；一个 `state=done` 无 completion 的 work-state；一条 `minutes=5` 的撤回行；`folder:read` 不足的 token；catalog 用户面 429（日配额与无配额两种）；Bearer 持有 `collection.edit_any` 角色的会话；选择器读失败的夹具。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | `GET /collections/{id}` 对另一用户的私密夹回 200 或 403 | 404 `NOT_FOUND`，body 不出现 `private`、不出现主人 id |
| 2 | 成员资格恢复整集替换，且接受 `collection_ids: []`（旧 `SetMembership`） | 无该操作。读失败后 0 次对 `…/works/{work_id}` 的 PUT/DELETE；空差集不发请求 |
| 3 | `GET /me/collections` 在空列表时创建默认夹（旧 `ensureDefault`） | 调用前后 `catalog_user_folder` 行数相同；200 `items: []` |
| 4 | `PUT /works/{id}/playtime` 接受 `play_state: "done"` 并写成 catalog `state=done` 无 completion | 422 `VALIDATION_FAILED` `NOT_ALLOWED_VALUE`；上游 0 次 PUT work-state |
| 5 | `DELETE /works/{id}/playtime` 改打 catalog `DELETE /v2/me/playtimes/{id}`（会删掉其它 client 的行） | 上游收到的是 `PUT {minutes:0}`，0 次 catalog DELETE；再撤回一次仍 200 `{minutes:0, play_state:null}` |
| 6 | 缺 `folder:write` / `catalog:edit` 回 500 或 401 | `403 SCOPE_REQUIRED` |
| 7 | 上游 429（无 Daily quota）回 500 或 400（旧 playtime） | `503 SERVICE_UNAVAILABLE`，`Retry-After` 若上游有则转发 |
| 8 | Bearer 调用者 `viewer.can_edit` / `can_delete` 为 true，或 PATCH 别人的夹 200 | `can_*` 全 false；PATCH 别人的夹 404；catalog 版主面 0 次调用 |
| 9 | 投票槽改回切换：已投再 PUT 变成未投 | 已投再 PUT → 200，`has_voted: true`，票数不变 |
| 10 | `visibility: "restricted"`（或任何闭集外的值）创建成功 | `400 UNKNOWN_ENUM_VALUE` |
| 11 | `/me/playtimes` 在 `include_nsfw=false` 时 `total` 仍含 NSFW 作品（旧合计） | `total` 等于能翻到的 `items` 数；NSFW 的 minutes 不进 `total_minutes` |
| 12 | `GET /me/collections` 或 `GET /users/{id}/collections` 仍 `MintAlias` | 调用后 `galgame_collection` 无新行 |
| 13 | 主人 DELETE 默认夹 204 | `409 INVALID_STATE_TRANSITION`；夹仍在 |
| 14 | `GET /works/{id}/covers/{cover_id}` 对属于另一部作品的封面回 200 | 404 |
| 15 | 上游 429（带 Daily quota 正文）映射成 429 或 500 | `503 SERVICE_UNAVAILABLE`；`Retry-After` 若上游有则转发 |
| 16 | `minutes: 60001` 回 200 | `400 INVALID_PARAMETER` `OUT_OF_RANGE` `maximum: 60000` |
| 17 | 封禁主人的公开夹对路人 200 | 404 |
| 18 | 选择器保存把「读到的含有」写成空差集却仍 DELETE 每一个夹 | 只有用户勾掉的那些夹收到 DELETE；未打开过的夹 0 次 DELETE |
| 19 | 别名查询对别人的私密夹回 200（披露存在） | 404；对主人本人 200 |
| 20 | 别名查询对未知 cid 铸一行 / 回 200 | 404；`galgame_collection` 行数不变 |
| 21 | 撤回回执写死 `{minutes:0, play_state:null}` | 夹具里另一 client 有 300 分钟：DELETE 后回执 `minutes: 300`，与随后 `GET /works/{id}` 的 `viewer.playtime` 相同 |

## 8. 开放问题

| # | 事实 | 裁决 |
|---|---|---|
| O1 | 别名表仍在读写；旧 URL `/galgame/collection/{cid}` 与 folder id 撞号（4,079 / 9,338） | **编排者裁决方案 B（§3.15）**：v1 用 folder id，新页 `/collection/{id}`，别名表冻结为重定向表，旧路径 301 + 薄页；库内旧链接 0 条，无改写迁移 |
| O2 | catalog 已有 `DELETE /v2/me/playtimes/{work_id}` | **不用**（§3.15 复核）：它按 (user, work) 删，不分 client；时长每 client 一行。撤回写 0 只动论坛这一行。以后若 infra 给出按 client 删，再换 |
| O3 | infra 对 playtime 不再要求 `playtime:*` | 论坛继续申请。上游若拒再 `SCOPE_REQUIRED`。不在本轨删 scope |
| O4 | 本地 `favorite_count` 与 catalog `nextmoe/favorites` 会漂（其它站点的收藏进指标、不进本地列） | 接受。详情数字用 catalog；本地列只当排行键（G4 已写） |
| O5 | 空名默认夹的展示靠网页拼主人名 | 保持。API 下发空串 `name`，客户端继续 `collectionDisplayName` |
| O6 | catalog 投票回执没有 `vote_count` | PUT/DELETE 后再读票仓。票仓失败 503，不回 0 装成功 |
| O7 | 网页创建表单默认勾「公开」 | 保持为表单默认；API 不默认公开。E12 的泄漏面是读闸，已经 404 |
