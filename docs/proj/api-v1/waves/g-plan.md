# G 轨计划：galgame 主域迁 /api/v1

> 2026-09-23 立。G 轨共 85 条旧路由：core 23、contribution 22、resources + toolsets 27、quiz 13，普查见 `census/galgame-*.md`。G0（改号，#188 + #192）已上线：galgame id 就是 catalog work id，v1 一律叫 `work_id`，集合叫 `works`，对象是 `object: "work"`（根 CLAUDE.md 铁律 3，01 §3）。迁移号段 140–159，140、141、142 已用。

## PR 顺序

先做不依赖作品表示的段，等 GE 的 `WorkSummary` 与实体摘要落地之后，再做作品详情与列表。

| 段 | 范围 | 旧路由 | 状态 |
|---|---|---|---|
| G1 | 工具集：列表、用户列表、详情、写、资源、分片上传、实用性（[契约](g1-toolsets.md)，迁移 142） | 16 | 契约已提交，实现中 |
| G2 | 题库 `/galgame-quiz`（`census/galgame-quiz.md`） | 13 | |
| G3 | galgame 资源：浏览、详情、按作品列、写、赞、有效/失效、发布禁止 | 11 | |
| G4 | 作品详情 `GET /works/{work_id}`、赞、外链、我的互动；删 `/galgame/drafts` | 6 | 等 GE 的实体摘要 |
| G5 | 浏览 `/works`（本地引擎）+ 资料库集合（catalog 引擎）+ sitemap + 发售月历 + collected months；外加 `/rss/galgame`（2026-09-23 由 X1 移交：改读 `/works?sort=created_desc`，旧 handler 照常退役） | 9 + 1 | 等 GE 的 `WorkSummary` |
| G6 | catalog 用户面：封面投票、游玩时长、收藏夹（含 `/users/{user_id}/collections`） | 11 | |
| G7 | 投稿、认领审核、资料编辑引擎（`census/galgame-contribution.md`） | 22 | |

G1 的 16 条加 G3 的 11 条就是 resources + toolsets 普查的 27 条。

## 跨轨约定（已与各轨对齐）

- **`object: "work"` 的三种形状**，重叠字段同名同型（G8）：
  - `repr.WorkRef`（c8，#199）：id、名字三件套 `CatalogName`、`cover` = 竖版封面原图、`is_nsfw` = 编辑轴。
  - `WorkSummary`（GE，`internal/galgame/workrepr`）：WorkRef 加 `banner`、`release_date` + 精度、`maker`、计数、评分、资源平台/语言（资源轴词表）、`resource_updated_at`、`is_published`。没有 `viewer`。
  - `Work`（G4，详情）：WorkSummary 加详情字段与 `viewer`。
- **筛选词表**（GE 先落地，G5 原样复用）：`resource_type` / `resource_platform` / `resource_language` 取 `internal/galgame/resourcevocab` 的键；`game_type` 词表归 GR；排序 token `resource_updated_{desc,asc}` `created_{desc,asc}` `view_{desc,asc}` `view_{1d,7d,30d}_desc` `release_date_{desc,asc}` `rating_{desc,asc}`，默认 `resource_updated_desc`。GE 在 `list_repo.go` 里加的是一条新的轴谓词路径，旧的标量路径不动，直到 G5 删掉 `/api/galgame`。
- **U3 的「某用户的 X」形状**：`/users/{user_id}/<x>`、页码集合、主人不可渲染 404。G 的 `/users/{user_id}/toolsets`（G1）与 `/users/{user_id}/collections`（G6）照此。
- **`ENTITY_MERGED`**（GE 注册）：G4 的合并作品复用，仍是 404（用户裁决不重定向），只是带 `current_id`。
- **GE 独有的面**：多标签交集 `GET /api/v1/tagged-works?tag_ids=`；实体的作品子集合 `…/{entity_id}/works`。

## 欠着的活

- `internal/galgame/apiv1.WorkRefOf`（c8）与 GE 的 `workrepr.Ref` 重复（GE 为避开 import 环复制了一份）。G4 碰 `internal/galgame/apiv1` 时让 `WorkRefOf` 委托给 `workrepr.Ref`。
- GE 的 `WorkSummary` 与 intro / link 形状已随 #210 落地：intro 是 `{locale, value, is_machine, data_source}`，外链是 `{site, url}`，嵌入字段名是 `work_summary`（`work` / `works` 归 X2 的 `WorkRef`）。G4 的 `Work` 与 G5 的 `/works` 直接复用。

- `catCoverSlot.Sexual` 目前按 `int` 解码，分不清「判为安全的 0」和「没判」，所以 `WorkRef.cover.sexual` 与 GE 的 `banner.sexual` 恒为 `null`。要改成指针，再把竖版封面的等级传进 `ImageMeta.Sexual`。在 G4 之前做。
- `galgame_renumber_2026` 已无读者（folder 改写已完成），可在 G 的某个迁移里删表。
- 普查 galgame-core 第 34 条（U 转交）：`PublishedToday` 用进程时区算「今天」，应改用 `cron.ScheduleLocation()`；同一个 `Stats()` 把 catalog 错误吞成 0。
