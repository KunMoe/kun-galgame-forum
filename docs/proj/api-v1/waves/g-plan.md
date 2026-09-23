# G 轨计划：galgame 主域迁 /api/v1

> 2026-09-23 立。G 轨共 85 条旧路由：core 23、contribution 22、resources + toolsets 27、quiz 13，普查见 `census/galgame-*.md`。G0（改号，#188 + #192）已上线：galgame id 就是 catalog work id，v1 一律叫 `work_id`，集合叫 `works`，对象是 `object: "work"`（根 CLAUDE.md 铁律 3，01 §3）。迁移号段 140–159，140、141、142 已用。

## PR 顺序

先做不依赖作品表示的段，等 GE 的 `WorkSummary` 与实体摘要落地之后，再做作品详情与列表。

| 段 | 范围 | 旧路由 | 状态 |
|---|---|---|---|
| G1 | 工具集：列表、用户列表、详情、写、资源、分片上传、实用性（[契约](g1-toolsets.md)，迁移 142） | 16 | ✅ 2026-09-23 #215；G1.1（资源字段改名 `toolset_resource_type`、`download_url` 可空、迁移 144） |
| G2 | 题库 `/quizzes`（[契约](g2-quizzes.md)，迁移 143） | 13 | 契约已定，实现中（分支 `api-v1/g2-quiz`） |
| G3 | galgame 资源：浏览、详情、按作品列、写、赞、有效/失效、发布禁止 | 11 | |
| G4 | 作品详情 `GET /works/{work_id}`、赞、外链、我的互动；删 `/galgame/drafts` | 6 | 等 GE 的实体摘要 |
| G5 | 浏览 `/works`（本地引擎）+ 资料库集合（catalog 引擎）+ sitemap + 发售月历 + collected months；外加 `/rss/galgame`（2026-09-23 由 X1 移交：改读 `/works?sort=created_desc`，旧 handler 照常退役） | 9 + 1 | 等 GE 的 `WorkSummary` |
| G6 | catalog 用户面：封面投票、游玩时长、收藏夹（含 `/users/{user_id}/collections`） | 11 | |
| G7 | 投稿、认领审核、资料编辑引擎（`census/galgame-contribution.md`） | 22 | |

G1 的 16 条加 G3 的 11 条就是 resources + toolsets 普查的 27 条。

### 逐段路由清单（2026-09-23 从 master 的 `routes.golden` 核过，G1 之后 G 余 73 条）

- **G2**（13）：`/galgame-quiz` 的 12 条（`GET all`、`GET :id`、`GET :id/answers`、`GET :id/edit`、`GET mine/answered`、`GET mine/favorites`、`POST`、`PUT :id`、`DELETE :id`、`POST :id/answer`、`PUT :id/favorite`、`PUT :id/quality`）＋ `GET /galgame/search/picker`
- **G3**（12）：`POST` / `PUT` / `DELETE /galgame/:id/resource`、`GET /galgame/:id/resource/all`、`PUT /galgame/:id/resource/{expired,valid,like}`、`GET /galgame-resource`、`GET /galgame-resource/:id`、`GET /galgame-resource/:id/detail`、`PUT /admin/galgame/:id/resource-publish-ban`，外加 `GET /search`（#211 之后只剩 `type=resource`，由 G3 的资源集合接住）
- **G4**（6）：`GET /galgame/:id`、`DELETE /galgame/:id`、`PUT /galgame/:id/like`、`GET /galgame/:id/link/all`、`GET /galgame/interactions/mine`、`GET /galgame/drafts`（删，无替代）
- **G5**（10）：`GET /galgame`、`GET /galgame/calendar`、`GET /galgame/calendar/{pending,tba,today,upcoming}`、`GET /galgame/collected-calendar`、`GET /rss/galgame`，外加 `GET /search/entity` 与 `GET /search/entity/resolve`（#211 留下；站内搜索的实体 tab 与筛选栏的实体 chip，形状用 GE 的实体摘要）
- **G6**（11）：`PUT` / `DELETE /galgame/:id/cover/:coverId/vote`、`PUT /galgame/:id/playtime`、`GET /galgame/playtime/mine`、`POST /galgame/collection`、`GET` / `PATCH` / `DELETE /galgame/collection/:cid`、`PUT /galgame/:id/collections`、`GET /galgame/:id/collections/mine`、`GET /user/:id/collections`
- **G7**（21）：`POST /galgame/submit`、`GET /galgame/search/wizard`、`GET /galgame/mine`、`GET /galgame/audited`、`POST /galgame/:id/resubmit`、`DELETE /galgame/:id/draft`、`GET /galgame/:id/edit/{bootstrap,diff,revisions}`、`GET` / `POST /galgame/:id/edit/proposals`、`POST /galgame/:id/edit/revert`、`GET /galgame-edit/{mine,queue}`、`GET /galgame-edit/proposals/:id`、`POST /galgame-edit/proposals/:id/{amend,decline,merge,withdraw}`、`GET /admin/galgame/submissions`、`POST /admin/galgame/:id/review`

不归 G：`GET /user/:id/galgames` 与 `GET /user/:id/galgame-comments`（U3b）；`POST /image/galgame`（X1d #216，四条 `/image` 一起删）；`GET /ranking/galgame`（#213）；`/galgame-rating*`（GR）。

## 跨轨约定（已与各轨对齐）

- **`object: "work"` 的三种形状**，重叠字段同名同型（G8）：
  - `repr.WorkRef`（c8，#199）：id、名字三件套 `CatalogName`、`cover` = 竖版封面原图、`is_nsfw` = 编辑轴。
  - `WorkSummary`（GE，`internal/galgame/workrepr`）：WorkRef 加 `banner`、`release_date` + 精度、`maker`、计数、评分、资源平台/语言（资源轴词表）、`resource_updated_at`、`is_published`。没有 `viewer`。
  - `Work`（G4，详情）：WorkSummary 加详情字段与 `viewer`。
- **筛选词表**（GE 先落地，G5 原样复用）：`resource_type` / `resource_platform` / `resource_language` 取 `internal/galgame/resourcevocab` 的键；`game_type` 词表归 GR；排序 token `resource_updated_{desc,asc}` `created_{desc,asc}` `view_{desc,asc}` `view_{1d,7d,30d}_desc` `release_date_{desc,asc}` `rating_{desc,asc}`，默认 `resource_updated_desc`。GE 在 `list_repo.go` 里加的是一条新的轴谓词路径，旧的标量路径不动，直到 G5 删掉 `/api/galgame`。
- **U3 的「某用户的 X」形状**：`/users/{user_id}/<x>`、页码集合、主人不可渲染 404。G 的 `/users/{user_id}/toolsets`（G1）与 `/users/{user_id}/collections`（G6）照此。
- **galgame 资源的字段名**（2026-09-23 裁，#209 的活动资源是第一个用例）：类型叫 `resource_type`，取 `internal/galgame/resourcevocab` 的封闭词表（与 GE 的 `/…/works?resource_type=` 同名同词表）；平台 / 语言是数组 `resource_platforms` / `resource_languages`（`workrepr` 的元素类型，与 `WorkSummary` 同形），不用旧的单值 `platform`；活动里的评分对象叫 `galgame_rating`，标量 PUT 体仍叫 `rating`。工具集的 `file` / `link` 因此改名 `toolset_resource_type`（G1.1）。
- **`ENTITY_MERGED`**（GE 注册）：G4 的合并作品复用，仍是 404（用户裁决不重定向），只是带 `current_id`。
- **GE 独有的面**：多标签交集 `GET /api/v1/tagged-works?tag_ids=`；实体的作品子集合 `…/{entity_id}/works`。

## 欠着的活

- `internal/galgame/apiv1.WorkRefOf`（c8）与 GE 的 `workrepr.Ref` 重复（GE 为避开 import 环复制了一份）。G4 碰 `internal/galgame/apiv1` 时让 `WorkRefOf` 委托给 `workrepr.Ref`。
- GE 的 `WorkSummary` 与 intro / link 形状已随 #210 落地：intro 是 `{locale, value, is_machine, data_source}`，外链是 `{site, url}`，嵌入字段名是 `work_summary`（`work` / `works` 归 X2 的 `WorkRef`）。G4 的 `Work` 与 G5 的 `/works` 直接复用。

- `catCoverSlot.Sexual` 目前按 `int` 解码，分不清「判为安全的 0」和「没判」，所以 `WorkRef.cover.sexual` 与 GE 的 `banner.sexual` 恒为 `null`。要改成指针，再把竖版封面的等级传进 `ImageMeta.Sexual`。在 G4 之前做。
- G3 的资源类型守卫用 `slices.Contains(resourcevocab.TypeKeys, t)`，**不要**用 `resourcevocab.IsType`：它的索引含 `LegacyTypeKeys`，`IsType("image")` 为真，而 `workrepr.ResourceType` 的 schema 枚举只有 `TypeKeys`，回出去的 200 违反自己的 spec（c8 在 #209 里撞上并用变异题钉住）。生产 0 行遗留类型，TypeKeys 之外的值按数据错 500。
- `galgame_renumber_2026` 已无读者（folder 改写已完成），可在 G 的某个迁移里删表。
- 普查 galgame-core 第 34 条（U 转交）：`PublishedToday` 用进程时区算「今天」，应改用 `cron.ScheduleLocation()`；同一个 `Stats()` 把 catalog 错误吞成 0。
