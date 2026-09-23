# G5 · 浏览、资料库、发售月历、收录月份、RSS、实体搜索

> G 轨第五段（G0 改号之后），2026-09-24。浏览页、资料库页、发售月历、收录月份条、Galgame RSS、站内实体搜索：**10** 条旧路由。G 号段 140–159；**145 是已用的最后一个号**（G0 140/141、G1 142、G2 143、G1.1 144、G3 145）。本轨无迁移。
> 契约与变异题同一个提交，早于实现。普查全文在 [census/galgame-core.md](census/galgame-core.md) §1、§3、§10–§13。普查写于 G0 / GE 之前：作品 id 现已是 catalog work id；GE 已在 `list_repo.go` 里加了 jsonb 轴谓词，旧标量 `gr.language` / `gr.platform` 仍在。本契约以代码为准，不沿用普查行号。
> 编排者裁定：两个引擎两份集合，永不互相切换；`/rss/galgame` 与 `/search/entity*` 删除无替代操作；实体 tab 改打 GE 族集合的 `q`；`/tags` 与 `/companies` 加 `ids`。

## 1. 普查（生产实测 2026-09-24，论坛库；数字按任务书原文）

### 1.1 数字

| 事实 | 数字 |
|---|---|
| `galgame` published | 9,821；有 ≥1 资源 9,259；SFW（`content_limit` NULL 或 `sfw`）且有 ≥1 资源 4,673；published 且 `content_limit` NULL 15 |
| `resource_update_time` NULL（浏览人口） | 0；published 行里该列并列值 0 |
| `created` 并列（published 行） | 1 |
| published 且 `release_date` NULL | 498 |
| 收录月份（`date_trunc('month', created)`，published 且有资源） | 30 |
| G0 窗口（2026-09-23 09:00–12:00Z）新建的 `galgame` 行 | 30 |
| 按 `created` 最新的 10 部 published | 7 部 `content_limit = 'nsfw'`。旧 RSS 先取 10 再在水合时丢掉 NSFW，feed 大约只剩 3 条 |
| `galgame_resource` 轴 | `type` / `language` / `platform` 是标量 `text`；多值轴是 jsonb `languages` / `platforms` / `runtimes`；`provider` 是 `text[]`（51,009 行） |
| `galgame_rating.galgame_type` | 全部 4,134 行都是 jsonb 数组 |
| `galgame` 浏览量列 | `view`、`view_7d`、`view_30d`（日桶 `galgame_view_daily` 供 `view_1d`） |

### 1.2 调用方

- 浏览 `GET /galgame`（不带 `library`）：
  - `apps/web/app/components/galgame/card/Container.vue:28`（`pages/galgame/index.vue` 挂它）。`limit` 写死 24。
  - 筛选条 `components/galgame/card/Nav.vue` + `composables/useGalgameFilters.ts`。URL 键是 camelCase，请求时映射成 snake_case。`isShowAdvanced` 时 `onMounted` 打收录月份。
- 资料库 `GET /galgame?library=true`：`components/galgame/library/Container.vue:10`（`pages/gallib.vue`）。默认 `sortField=popularity`。不发资源轴 / 网盘 / 收录日 / 评分门槛。
- Nitro sitemap：`apps/web/server/utils/kunSitemapSources.ts:260` `GET /api/galgame?indexed=true&page=N&limit=50`，带 SFW cookie。`PAGE_SIZE=50`，`MAX_PAGES=130`（最多 6,500 条；published 9,821，sitemap 客户端自己截断）。
- 月历：`components/galgame/calendar/Container.vue`（`pages/galgame-calendar/index.vue`）打 `/galgame/calendar`、`/upcoming`、`/pending`、`/tba`。侧栏 `composables/useGalgameReleaseToday.ts:60` 打 `/galgame/calendar/today`，按 `expires_in` 本地缓存。
- 收录月份：仅浏览页高级筛选 `card/Nav.vue:175` `GET /galgame/collected-calendar`。实体页挂同一 Nav 但不拉。
- RSS：Nitro `apps/web/server/routes/rss/galgame.xml.ts:17` `GET /api/rss/galgame`，缓存 5 分钟。
- 实体搜索：
  - `components/search/Entities.vue:53` 与 `utils/search/overview.ts:64` → `GET /search/entity`（tab 与「全部」分区）。
  - `components/filter/EntityMenu.vue:37` → 同路由，`family=company|tag`（浏览筛选栏与搜索页 Galgame 筛选 `search/GalgameFilter.vue`、标签页 `galgame/tag/Container.vue`）。
  - `composables/useEntityNames.ts:35` → `GET /search/entity/resolve`（冷链芯片还原名字，只解析 company / tag）。
- Flutter App：本 worktree 不含 `kungal-apps`。普查写 Dart 源零命中；`docs/proj/app-direct-api.md:84` 把 `GET /api/galgame` 列为前瞻供数，不是现网调用。

### 1.3 旧面的问题（普查编号 E*，对应 census 发现号）

G0 之后作废、不再建模的：发现 1–3（论坛 gid 当 catalog work id）、§0 的 10,289 撞车、§10 命名表里「先译 catalog id」。

| 编号 | 事实 |
|---|---|
| E1 | `GET /galgame` 用 `library` 布尔切引擎（`galgame_service.go:144`）。空过滤不会切；`library=true` 时类型 / 语言 / 平台 / 网盘 / 收录日 / 评分门槛全部丢掉（census 发现 27） |
| E2 | 本地 `sort_field` 未知值静默回落到 `resource_update_time`（`list_repo.go:51`）；catalog 引擎回落到 `popularity`（`galgame_library.go:94`）。`library=false` 时 `popularity` 按资源更新时间排（发现 14） |
| E3 | `type` / `language` / `platform` 开放字符串，未知当精确匹配、命中 0 行。网页浏览条仍发旧键 `windows` / `app` / `others` / `image` / `ai`（`constants/galgame.ts`）；实体页已经走资源轴（发现 15） |
| E4 | `page` / `limit` 因 `min=1` 事实必填；非法日期回中文 400 句（`release_date.go:46`）。不接受 `YYYY-MM-DD` |
| E5 | 列表 `total` 来自 SQL / catalog 计数，水合时 catalog 不认识或 hidden 的 id `continue` 掉（`galgame_service.go:205`），页短、`total` 不动、不打日志（发现 11） |
| E6 | SFW 闸在 SQL 里是 `content_limit IS NULL OR = 'sfw'`（`list_repo.go:369`，`list_repo_test.go` 钉 NULL 必须放行）。RSS **没有**走这道闸：`FindRecentWorkIDs` 只看 `published` 再 `GetBatchPublic(..., true)` 丢行（`rss_repo.go:26`，`rss_handler.go:38`）。最新 10 部里 7 部 NSFW，feed 大约 3 条（g-plan 线上 bug） |
| E7 | 本地排序决胜键恒 `g.id DESC`（`list_repo.go:81`），升序时方向与排序键相反。`created` 在 published 行里有 1 处并列 |
| E8 | 月历只读 catalog 一页 `limit=100`、不跟游标（`calendar_service.go:20`）。更忙的月份静默截断。`GetTodayFlag` 只扫当前月第一页（发现 13） |
| E9 | `GetUpcoming` 最多 24 个月、并发 8，失败的月份直接跳过（`calendar_service.go:190`） |
| E10 | `CollectedCalendar` 的 Scan 失败回空数组，handler 不回 error（`galgame_handler.go:33`，发现 20） |
| E11 | 实体 `family` 校验允许 `series` / `engine`（`search_dto.go:16`）。catalog 2.4.0 已把这两族加进 `/v2/catalog/search` 的 `object` 闭集（`catalog_v2_test.go:291`）；未跑过夜索引时上游给空页而不是 400。GE 的 `/series`、`/engines` 的 `q` 走本地子串，不打实体搜索 |
| E12 | `parseCSVInts` 静默丢非法 id、超 100 截断（G4 E14 同类）。resolve 只认 company / tag |
| E13 | 浏览页与资料库共用 `useGalgameFilters`；资料库的 `sortField=popularity` 拿到本地引擎是 200，按资源更新时间排 |
| E14 | GE 加的 jsonb 轴（`PlatformAxis` / `LanguageAxis`）与旧标量 `gr.language` / `gr.platform` 并列。`GET /galgame` 仍走标量；类型没有 jsonb 轴，GE 的 `resource_type` 已经打 `gr.type` |

## 2. 范围与寻址

两个引擎两份集合，页面钉死，空过滤不切引擎（K-G30）。路径段就是目标。

| # | 旧 | v1 | 引擎 | 档 |
|---|---|---|---|---|
| 1a | `GET /galgame` | `GET /api/v1/works` → `PageList[WorkSummary]` | 本地 SQL（published；默认还要 ≥1 资源） | public |
| 1b | `GET /galgame?library=true` | `GET /api/v1/library-works` → `PageList[WorkSummary]` | catalog 搜索人口（`ApplyWorksGate`） | public |
| 2 | `GET /galgame/calendar` | `GET /api/v1/release-calendar` → `ReleaseCalendarMonth` | catalog | public |
| 3 | `GET /galgame/calendar/today` | `GET /api/v1/release-calendar/today` → `ReleaseCalendarToday` | catalog | public |
| 4 | `GET /galgame/calendar/pending` | `GET /api/v1/release-calendar/pending` → `ReleaseCalendarPending` | catalog | public |
| 5 | `GET /galgame/calendar/tba` | `GET /api/v1/release-calendar/tba` → `ReleaseCalendarTBA` | catalog | public |
| 6 | `GET /galgame/calendar/upcoming` | `GET /api/v1/release-calendar/upcoming` → `ReleaseCalendarUpcoming` | catalog | public |
| 7 | `GET /galgame/collected-calendar` | `GET /api/v1/works/collected-months` → `WorkCollectedMonths` | 本地 SQL，与 `/works` 同一人口 | public |
| 8 | `GET /rss/galgame` | **删除**。Nitro 读 `GET /api/v1/works?sort=resource_updated_desc&limit=20`（不带 `include_nsfw`），用 `WorkSummary` 拼条目（排序键见 K-G37） | — | — |
| 9 | `GET /search/entity` | **删除，无新操作**。搜索 tab / 总览 / 筛选栏实体选择器改打 GE 族集合的 `q` | — | — |
| 10 | `GET /search/entity/resolve` | **删除**。`GET /tags` 与 `GET /companies` 加 `ids`（1–100 十进制 id，逗号形，`explode: false`），仍是 `PageList`，缺席的 id 直接不出现 | — | — |

10 条旧路由全删，`legacy_route_baseline` 下调 **10**（当前基线 49 → 39，以合并时 rebase 后重新生成为准）。v1 新操作 **8** 个（listWorks、listLibraryWorks、五条月历、listWorkCollectedMonths），外加两条已有 GE 操作的加法（`ids`）。

`GET /works/{work_id}` 已在。`GET /works` 段数不同，不阴影。`GET /works/collected-months` 与 `GET /works/{work_id}` 段数相同：必须先注册静态段（`TestNoRouteIsShadowed`）。实现把 `listWorkCollectedMonths` 写在 `getWork` 前面。

代码位置：浏览 / 资料库 / 收录月份挂已有的 `internal/galgame/apiv1`（与 `getWork` 同包，OpenAPI tag `works`；资料库 tag 同 `works` 或单独 `library-works`，实现时二选一，以生成物为准）。月历 `internal/galgame/calendarapiv1`，tag `release-calendar`。`ids` 加在 `internal/galgame/entityapiv1` 已有的 `listTags` / `listCompanies`。测试 `internal/app/v1_works_browse_*_test.go`、`v1_release_calendar_*_test.go`。

X2 已有 `GET /search/works`（catalog 搜索、`q` 必填、条目 `WorkRef`）。本轨的 `/library-works` 是同一 catalog 人口的**浏览**面：`q` 可缺席，条目 `WorkSummary`，默认 `popularity_desc`。两份集合，形状静态（K10）。

## 3. 形状

### 3.1 命名

| 旧 | v1 | 为什么 |
|---|---|---|
| `GET /galgame` 与 `?library=true` 切引擎 | `/works` 与 `/library-works` | 两份集合（K-G30）；`library` 是动词/布尔 |
| `galgames` + `total` | `repr.PageList`：`items` + `total` + `total_relation` | 页码集合（K11、F9） |
| `page` / `limit` 事实必填，上限 50 | `collect.PageNumber`，默认 `limit` **24**，最大 100 | 浏览页写死 24；sitemap 要 50；超 100 是 `400 LIMIT_TOO_LARGE`，不夹 |
| `sort_field` + `sort_order` | `sort=` 封闭 token | 未知 → `400 UNKNOWN_SORT`，不得静默回落 |
| `type` / `language` / `platform` 开放标量 | `resource_type`（单值，与 GE 同名同词表）；`resource_platforms` / `resource_languages`（逗号形数组，词表 `resourcevocab`） | 标量列是兼容投影；v1 打 jsonb 轴。不发 `all` token（缺席 = 不过滤） |
| `include_providers` | `resource_providers` | F1：查询参数以 `include_` 开头必须是布尔（`include_nsfw` / `include_resourceless`）。网盘键是数组 |
| `exclude_only_providers` | `excluded_sole_providers` | 同一词表；语义仍是「不能只靠这些盘」 |
| `indexed` / `show_no_resource` | `include_resourceless`（默认 `false`） | 布尔 `include_`。sitemap 打开它以列出全部 published。不再提供「连 unpublished 也列」——那个开关没有调用方 |
| cookie / `IsSFW` | `include_nsfw`（默认 `false`） | 01 §3 |
| `view` / `view_7d` / `view_30d` | 排序 token `view_*`；条目字段已是 `view_count` 等 | WorkSummary 已有 |
| `resource_update_time` | 排序 token `resource_updated_*`；条目 `resource_updated_at` | WorkSummary 已有 |
| `GET /rss/galgame` 的 `created` / `user` / `name` | Nitro 从 `WorkSummary` 取 `id`、名字三件套、`banner`、`resource_updated_at` | `created` / `user` 禁用名；WorkSummary 没有 `created_at`、没有作者 |
| 月历条目 `GalgameCard` | `WorkSummary` | GE 已落地 |
| 月历信封 `month: "YYYY-MM"` | `calendar_month` | G8：`month` 已是 `MonthCount` / `NewsMonth` 的整数 1–12 |
| 月历 `meta.count` / pending `count` | `item_count` | F9：`total` 只能出现在页码集合（带 `total_relation`）或带 `include_total` 的游标集合。X1c 的 `NewsMonth` 同一改名 |
| `expires_in` | **保持** `expires_in`（整数秒，≥0） | 现 spec 未占用；侧栏按秒缓存 |
| `family=staff` | `GET /credit-names?q=` | GE D1：论坛「制作人员」寻址的是 `credit_name` |
| `GET /search/entity/resolve?ids=` | `GET /tags?ids=` / `GET /companies?ids=` | 01 A9 出现消费者再加批量读；形状是该集合自己的 `PageList` |

### 3.2 `WorkSummary`（本轨不改）

本轨每一个作品列表的条目都是 `workrepr.WorkSummary`（`object: "work"`）。字段、可空、词表以 GE / G4 为准，本轨不加 `resource_types` / `favorite_count` / `viewer` / `created_at`。水合走 `workrepr.Hydrator.ByIDs`：一次 `CatalogRowsByWorkIDs`（`include=names,covers,refs,labels`，`content_limit` 由 `include_nsfw` 决定），再补本地计数与资源轴。

catalog 不认识、hidden、或对本读者不可渲染的 id **丢掉并打 WARN**（修 E5；旧面静默 `continue`）。页上的 `items` 可能短于 `limit`；页码集合的 `total` 仍是 SQL / catalog 计数（与 G3 hidden 行同一取舍）。操作描述写明。

### 3.3 `GET /works`（本地引擎，K-G31）

页码集合。`collect.PageNumber`：`page` ≥1 默认 1，`limit` 1–100 默认 **24**。`CheckDepth` `page × limit ≤ 10000`，越界 `400 INVALID_PARAMETER` `OUT_OF_RANGE`。`total` + `total_relation` 走 `collect.ClampTotal`。

**一条 SQL 谓词，COUNT 与页查询共用。** NSFW 在 SQL 里、LIMIT 之前过滤。

| 参数 | 类型 | 默认 | 语义 |
|---|---|---|---|
| `page` / `limit` | 页码 | 1 / 24 | 见上 |
| `sort` | 封闭，与 GE `WorkSortToken` 同一词表 | `resource_updated_desc` | 见下 |
| `resource_type` | `resourcevocab.TypeKeys` 单值 | 缺席 = 不过滤 | 至少一个该类型的资源（标量列 `galgame_resource.type`）。与 GE `/tags/{id}/works` 同名同词表 |
| `resource_platforms` | `ResourcePlatform[]`，逗号形，`explode: false`，去重 | 缺席 = 不过滤 | jsonb `platforms` 与该集合相交（同一条资源行上与其它轴 AND） |
| `resource_languages` | `ResourceLanguage[]`，逗号形 | 缺席 = 不过滤 | jsonb `languages`，同上 |
| `game_type` | `ba_saku` `plot` `moe` `daily` `uncategorized` | 缺席 = 不过滤 | 与 GE / GR 同词表。`uncategorized` = 没有任何评分标过非空 `galgame_type` |
| `resource_providers` | 封闭数组，逗号形 | 缺席 = 不过滤 | `gr.provider &&` 含其中任一 |
| `excluded_sole_providers` | 同一封闭数组 | 缺席 = 不过滤 | 资源必须与「词表减去这些」的补集相交 |
| `released_from` / `released_to` | `YYYY` 或 `YYYY-MM`（pattern 与 `/search/works` 相同） | 缺席 = 不过滤 | 镜列 `galgame.release_date`。`YYYY` 下界 1 月 1 日、上界 12 月 31 日；`YYYY-MM` 落到该月首末日。不接受 `YYYY-MM-DD`。前者晚于后者 → `400 INVALID_PARAMETER` `INCONSISTENT_WITH` |
| `released_months` | 1–12 的逗号形整数，去重 | 缺席 = 不过滤 | `EXTRACT(MONTH FROM g.release_date)` |
| `collected_from` / `collected_to` | 同日期梯子 | 缺席 = 不过滤 | **`galgame.created`**（本站首次落行），不是 catalog 发售日。上界是「次日 0 点之前」 |
| `collected_months` | 1–12 的逗号形整数 | 缺席 = 不过滤 | `EXTRACT(MONTH FROM g.created)` |
| `min_rating` | number 0–10 | 缺席 / 0 = 不过滤 | 贝叶斯分下限（C=10，全表均值为先验） |
| `min_rating_count` | integer ≥0 | 缺席 / 0 = 不过滤 | 评分人数下限 |
| `include_nsfw` | bool | `false` | `false`：`g.content_limit IS NULL OR = 'sfw'`。NULL 必须放行 |
| `include_resourceless` | bool | `false` | `false`：`EXISTS` 至少一条 `galgame_resource`。`true`：仍要求 `published`，不再要求资源。sitemap 打开它。任一资源轴 / 网盘过滤出现时，这条不再放行无资源行（谓词本身要求资源） |

不接受 `resource_runtimes`：浏览页与筛选条都不发运行环境（A5）。网页 `RUNTIME_OPTIONS` 只出现在资源写面。

未知封闭枚举 → `400 UNKNOWN_ENUM_VALUE`。畸形日期 / 月份 → `400 INVALID_PARAMETER`（不再回中文句）。未知 `sort` → `400 UNKNOWN_SORT`。未知参数名忽略（01 §4）。

**排序 token**（与 GE 实体作品子集合共用 `WorkSortToken`，含窗口浏览量的升序——浏览页有统一的方向按钮；g-plan 字面只写了 `view_{1d,7d,30d}_desc`，本轨按已落地的词表收下 `_asc`）：

`resource_updated_{desc,asc}` `created_{desc,asc}` `view_{desc,asc}` `view_1d_{desc,asc}` `view_7d_{desc,asc}` `view_30d_{desc,asc}` `release_date_{desc,asc}` `rating_{desc,asc}`。

每一个带 **同方向** 的 `id` 决胜键（修 E7：旧面升序时仍 `g.id DESC`）。`release_date` 另加 `NULLS LAST`。`rating` 无评分的排最后，再按贝叶斯分，再按 `id`。`view_1d` 是 `galgame_view_daily` 当日桶的 `COALESCE(SUM, 0)`。

人口：`g.published`。隐藏 / 封禁靠应用层把 `published` 扳假（迁移 078 粘性旗）。默认还要有资源。

删掉 `gr.language` / `gr.platform` 标量谓词（本轨之后 `GET /galgame` 不在，GE 已经走 jsonb 轴）。`gr.type` 留下：类型没有 jsonb 轴，GE 的 `resource_type` 已经打这一列。

网盘词表（封闭，与 `list_repo.go:29` / `PROVIDER_KEY_OPTIONS` 一致）：

`baidu` `aliyun` `quark` `pan123` `tianyiyun` `caiyun` `xunlei` `uc` `lanzou` `other`。

### 3.4 `GET /library-works`（catalog 引擎，K-G33）

页码集合，默认 `limit` 24，最大 100，深度 10000。`total` 是 catalog 的计数，超过深度记 `gte`。

**只接受 catalog 搜索人口能做的事。** 论坛侧过滤（资源轴、网盘、收录日、评分门槛、`include_resourceless`、`game_type`）不出现在 schema 里；带上它们按未知参数忽略，不得再静默丢、也不得切回本地引擎。

| 参数 | 类型 | 默认 | 语义 |
|---|---|---|---|
| `page` / `limit` | 页码 | 1 / 24 | 实现把 `page` 译成 catalog 的 `cur_`（沿用 `v2CatalogQuery` 的页码游标，不是 offset） |
| `q` | 1–107，自由文本 | 缺席 = 不搜 | catalog `q`。去空白后为空 → `400 INVALID_PARAMETER` `TOO_SHORT`。资料库页今天不发；X2 `/search/works` 是带 `q` 的搜索面 |
| `sort` | `popularity_desc` `released_desc` `released_asc` `updated_desc` `relevance_desc` | `popularity_desc` | 译成 catalog `WorkSort`：`popularity` / `released_desc` / `released_asc` / `updated` / `relevance`。catalog 列表词表还有 `id`，本面不收（A5，页面没有） |
| `released_from` / `released_to` | 同 `/works` 的日期梯子 | 缺席 = 不过滤 | 译成 catalog `released_after` / `released_before`（`YYYY-MM-DD` 闭区间）。catalog 自己的这两键只收日（`listWorksInput` maxLength 10） |
| `include_nsfw` | bool | `false` | `ApplyWorksGate`：年龄轴始终 `nsfw=true`（`openPopulation`）；SFW 另加 `content_limit=sfw` |

`include` 由服务端写成 `workrepr.RowInclude`，客户端不选（K10）。catalog 失败 → `503`。不可渲染的行丢掉并打 WARN，`total` 仍是 catalog 的数。

### 3.5 发售月历（K-G34、K-G35）

条目都是 `WorkSummary`。钟是 **Asia/Tokyo**，`LoadLocation` 失败则 UTC（现 `calendar_service.go:27`）。理由：日本 Galgame 的街机日按 JST 切月；infra 月历的家时区也是 Asia/Tokyo（`catalog_feed_routes.go:27`，`catalog_calendar.go:17` 用 `FixedZone`）。论坛容器有的没有 tzdata，保留 UTC 回落。`today` 与「当前月 / 当前年」按这个钟。`expires_in` 是到东京次日 0 点的整秒。

五条操作、五种信封（K10：形状按操作静态确定，不用 `window=` 切）。全部 `include_nsfw` 默认 `false`，经 `ApplyWorksGate`。全部 public。

**跟游标把一个月走完**（修 E8）。每页向 catalog 要 `limit=100`（catalog 上限），最多 **20** 页 = 2,000 部。触顶：200，`is_truncated: true`，打 WARN，**不得**静默截成 100。`item_count` 在未截断时等于 `items.length`，并与 catalog `include_total` 对齐；截断时 `item_count` 仍是走完的条数，操作描述写明「可能少于 catalog 的 total」。

pending / tba 同一套游标与帽。cancelled 在 infra 是空窗（`catalog_calendar.go:144`），本面不发 `status=cancelled`。

**`upcoming` 里失败的一个月是整段 `503 SERVICE_UNAVAILABLE`**（修 E9）。理由：GE D18 与 G3 对 catalog 失败都是 503；缺一个月的「未发售」看起来像完整答案，正是旧 bug。当前月都拉不到 → 同样 503。客户端重试整页。

每条日历请求把 catalog 行经 `workrepr.FromRows` 变成 `WorkSummary`。不可渲染的丢掉并打 WARN。

#### `ReleaseCalendarMonth`（`object: "release_calendar_month"`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `calendar_month` | string，pattern `^[0-9]{4}-(0[1-9]|1[0-2])$`，maxLength 7 | 窗口。query `month` 缺席则东京当前月。非法 → `400 INVALID_PARAMETER` |
| `today` | `format: date` | 东京今天 |
| `items` | `WorkSummary[]` | 永不 `null`。按 catalog 顺序 |
| `prev_month` / `next_month` | 同 `calendar_month` 的 pattern，可空 | 相邻月；没有为 `null` |
| `has_prev` / `has_next` | bool | catalog `meta`；缺席当 false |
| `min_month` / `max_month` | 同 pattern，可空 | 有发售的最早 / 最晚月 |
| `item_count` | int ≥0 | 见上 |
| `is_truncated` | bool | 触 2,000 帽 |

query：`month`（YYYY-MM）、`include_nsfw`。

#### `ReleaseCalendarToday`（`object: "release_calendar_today"`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `today` | date | |
| `has_release` | bool | **在走完的当前月**里是否存在 `release_date == today`（日精度）。月精度 `"2026-09"` 不等于今天 |
| `expires_in` | int ≥0 | 到东京次日 0 点的秒数 |

无 query 除 `include_nsfw`。当前月触帽仍按已取到的行判断，并打 WARN（与月面同一帽）。

#### `ReleaseCalendarPending`（`object: "release_calendar_pending"`）

query：`year`（YYYY；缺席 = 东京当前年）、`include_nsfw`。发给 catalog `precision=year`（infra `calendarPrecision` = `day,month,year`）。

字段：`year`（整数，format int64，1–9999，与 `YearCount.year` 同型——G8；下限不沿用新闻的 1970，作品发售年可以更早，**实现时若 G8 因 minimum 以外的 shape 撞上再抬**；shape() 不含 minimum，所以 1–9999 与 1970–9999 能共存）、`items`、`item_count`、`is_truncated`。

G8 复核：`year` 的 shape 是 `integer/int64`（NewsMonth / YearCount）。本面 `year` 必须带 `format: int64`。

#### `ReleaseCalendarTBA`（`object: "release_calendar_tba"`）

发给 catalog `status=unknown`（infra `calendarStatus` = `released,dated,announced,cancelled,unknown`；`announced` 与 `unknown` 落同一未定桶，本面只发 `unknown`，与旧 pending/tba 拆分一致）。字段：`items`、`item_count`、`is_truncated`。

#### `ReleaseCalendarUpcoming`（`object: "release_calendar_upcoming"`）

从东京当前月走到 `max_month`，最多 **24** 个月（旧 `upcomingMonthCap`）。每个月按月面同一套游标走完，单月帽 **5** 页 = 500（给 24 路扇出封顶；触顶的那个月 `is_truncated: true` 仍出现在 `entries` 里）。过滤：可渲染且 `release_date >=` 当月（字典序，月精度 `"2026-08"` 排在该月所有日之前，旧注释保留）。

| 字段 | 类型 | 说明 |
|---|---|---|
| `today` | date | |
| `entries` | `ReleaseCalendarUpcomingEntry[]` | 不叫 `months`：G8 里 `NewsArchive.months` 已是 `MonthCount[]` |
| `item_count` | int ≥0 | 卡片总数 |

`ReleaseCalendarUpcomingEntry`：`calendar_month`、`items: WorkSummary[]`、`is_truncated`。没有作品的月不出现。

### 3.6 `GET /works/collected-months`（K-G36）

不是页码集合（30 个格子，F9 不要求 `page`）。Scan / 查询失败 → **500 `INTERNAL_ERROR`**，不得回空数组（修 E10）。

`WorkCollectedMonths`（`object: "work_collected_months"`）：`items: WorkCollectedMonth[]`。`WorkCollectedMonth`：`year`（integer/int64，1–9999）、`month`（integer/int64，1–12）。排序：年降、月升（旧 `ListCollectedCalendar`）。

query：`include_nsfw`（默认 false）。人口与 `/works` 默认相同：`published` + 有资源 + SFW 闸（NULL 放行）。打开 `include_nsfw` 则不过 SFW 闸。不接受 `include_resourceless`：条是给「本站有资源的收录月」筛子用的。

### 3.7 实体族的 `q` 与 `ids`

本轨不新建实体搜索操作。搜索 tab 的六个 family 对应：

| 旧 `family` | v1 | `q` 后端 | `q` 上限 | 带 `q` 的深度 |
|---|---|---|---|---|
| `tag` | `GET /tags?q=` | catalog 实体搜索 `object=tag`，再过 hidden / 成人闸 | 100 | `page × limit ≤ 100` |
| `company` | `GET /companies?q=` | catalog `object=company`（客户端仍写 `labels`） | 100 | 100 |
| `character` | `GET /characters?q=`（`q` **必填**） | catalog `object=character` | 100 | 100 |
| `staff` | `GET /credit-names?q=`（`q` **必填**） | catalog `object=credit_name`（客户端仍写 `names`） | 100 | 100 |
| `series` | `GET /series?q=` | **本地**子串，打全部 7,250 个系列名（含尚无本站作品的） | 100 | 普通 10000 |
| `engine` | `GET /engines?q=` | **本地**子串，名字与别名 | 100 | 普通 10000 |

六个族的 `q` 都能服务搜索 tab。x2-search 发现 9（catalog 闭集可能没有 series/engine）对 catalog 2.4.0 已过时；GE 对这两族本来就不打实体搜索，走本地索引，空页只会发生在「真的没有子串命中」。

旧 `keywords` 上限 107；GE 的 `q` 是 100。搜索页把关键词裁到 100 再发，超长不再 400。

**`ids` 加法**（只加在 `/tags` 与 `/companies`，resolve 的两个消费者）：

- 1–100 个十进制 id，逗号形，`explode: false`，pattern 与 `{tag_id}` / `{company_id}` 相同。
- 与 `q` 同时出现 → `400 INVALID_PARAMETER` `INCONSISTENT_WITH`（与 `/users?ids=` 同一互斥）。
- 缺席的 / 对本读者不可见的 id **不出现**，不进 `missing`（这是 `PageList` 不是 `BatchList`；01 A9 的批量读在消费者出现时加，形状跟集合走）。
- 成人标签在 `include_nsfw=false` 时当作不可见，不出现。
- 顺序：请求顺序，去重。
- `page` / `limit` 仍在；ids 最多 100，一页够。非法 token / 非正十进制 / 超过 100 → `400`，不静默截断（修 E12）。

`company_kind` 与 `ids` 可以一起用（先按 id 取再滤 kind）。

### 3.8 NSFW 与可见性

`/works` 与 collected-months：SQL 闸，NULL 放行，LIMIT 之前。水合再走 catalog 的编辑轴；镜像滞后的 NSFW 行可能进 id 页再被丢掉（WARN，页短）。

`/library-works` 与月历：`ApplyWorksGate`。年龄轴始终打开（`openPopulation`，`catalog_face.go:34` 的注释：关掉的话 SFW 读者只能看到大约 7% 的作品）。

不读 `KUNGalgameSettings`、不读已删除的 `X-Kungal-Nsfw`。

### 3.12 实现时按 G8 / G14 / F1 改的名（只增不改，以此为准）

| 前文 / 旧 | 实际 | 撞了谁 |
|---|---|---|
| 月历窗口叫 `month`（YYYY-MM 字符串） | `calendar_month` | `MonthCount.month` / `NewsMonth.month` 是 1–12 的整数 |
| 月历人数叫 `count` / `total` | `item_count` | F9：`total` 留给页码集合；X1c `NewsMonth.item_count` 同型 |
| upcoming 的按月数组叫 `months` | `entries` | `NewsArchive.months` 是 `MonthCount[]` |
| `include_providers` | `resource_providers` | F1：`include_` 查询参数必须是布尔 |
| `exclude_only_providers` | `excluded_sole_providers` | 去掉中间的 `only`；数组用复数 |
| `indexed` / `show_no_resource` | `include_resourceless` | 一个布尔 |
| 网盘键与 G3 `provider_names` | 不叫 `provider_names` | G3 那列是从 URL 派生的展示名，开放字符串数组 |

`resource_type` / `game_type` / `include_nsfw` / `released_from` / `released_to` / `sort` / `q` / `page` / `limit` / `has_prev` / `has_next` / `today` / `expires_in` / `has_release` / `is_truncated` / `year`（integer/int64）与现 spec 对齐或未占用。`resource_platforms` / `resource_languages` 作为**查询参数**（G8 的同名检查跳过 `param`）；JSON 里同名已是 WorkSummary 的数组，实现时数组元素类型必须是同一个 `ResourcePlatform` / `ResourceLanguage`。

`calendar_month` / `prev_month` / `next_month` / `min_month` / `max_month`：pattern `^[0-9]{4}-(0[1-9]|1[0-2])$`，maxLength 7，G14 有 pattern。可空的四个用指针。

每个字符串：封闭枚举、format、pattern 或自由文本句（G14）。`q` 是自由文本。

### 3.13 对 infra 的词表差（G4.1 课，逐行）

每一个本轨传给或读自 catalog 的封闭词表 / 长度上限，对 infra 的定义比，不对 dev 数据恰好有的值比。有意丢行的分支打 WARN。

| 字段 | infra 的集合 / 上限（文件:行） | 本轨 | 裁定 |
|---|---|---|---|
| 月历 `precision` | `day, month, year`（`catalog_calendar.go:19`，`catalog_feed_routes.go:29`） | pending 只发 `year`；不收客户端的 `precision` | 闭集的真子集，发给上游的值都在闭集里 |
| 月历 `status` | `released, dated, announced, cancelled, unknown`（`:20`） | tba 只发 `unknown`。不发 `cancelled`（infra 空窗）、不发 `announced`（与 `unknown` 同一未定桶，旧 pending/tba 靠 precision/status 拆开） | 不把闭集收窄成响应枚举；本面不暴露这个参数 |
| 月历 `month` | YYYY-MM，非法 400（`catalog_calendar.go:178`） | 同 pattern；缺席 = 东京当前月 | 对齐 |
| 月历 `year` | YYYY，非法 400（`:163`） | 同；缺席 = 东京当前年 | 对齐 |
| 月历家时区 | Asia/Tokyo（`catalog_feed_routes.go:27`，`catalog_calendar.go:17` FixedZone） | `LoadLocation("Asia/Tokyo")`，失败 UTC | 保留回落（容器可能无 tzdata） |
| 月历 `include` | `WorkListInclude` = titles,refs,intros,covers,companies,ratings,tags,credits（`collect/work.go:29`）。多要一个是 400 `UNKNOWN_INCLUDE`（路由描述 `catalog_feed_routes.go:71`） | 服务端固定 `RowInclude`（names,covers,refs,labels → v2 改写 titles,covers,refs,companies） | 不把 `include` 暴露给客户端。不要 titles 以外的列表块 |
| 月历分页 | 游标，`limit` 1–100 默认 20（CollectionInput） | 服务端 `limit=100` 跟 `next_cursor`，帽 20 页 | 对客户端仍是「一个月一个信封」，不是游标集合 |
| 作品列表 `sort` | `id, updated, relevance, released_desc, released_asc, popularity`（`collect/work.go:46`）。搜索子集去掉 `id` / `updated`（`SearchSort` `:52`） | `/library-works` 收五值（无 `id`），token 带方向后译回 catalog | 不收 catalog 的裸 `popularity`（01：排序 token 是 `<键>_<方向>`） |
| 作品列表 `q` | maxLength **512**（`collection.go:78`） | **107**（与 X2 `/search/works`、话题搜索同一论坛上限） | 收窄。超 107 在 huma 拦下，到不了 catalog |
| 作品 `released_after` / `released_before` | YYYY-MM-DD（`:91`） | 论坛梯子 YYYY / YYYY-MM，服务端展开成日 | 对齐 catalog 的日键；不把日键暴露给网页（旧面也不收日） |
| 作品 `content_limit` | `sfw, nsfw` 逗号形（`:82`） | `ApplyWorksGate`：SFW 发 `sfw`，否则不发这条（年龄轴仍 `nsfw=true`） | 对齐编辑轴。不把 `content_limit` 暴露给客户端 |
| 作品 `nsfw` | 年龄轴，r18 要 `nsfw=true`（`:77`） | 恒 `true`（`openPopulation`） | 不收窄。SFW 读者靠编辑轴 |
| 实体搜索 `object` | `work, character, credit_name, company, tag, series, engine, trait`（`catalog_search_routes.go:16`） | 本轨不打这条。GE 已对 tag/company/character/credit_name 使用对应 object；series/engine 走本地 `q` | 六族都能服务搜索 tab。不引入 `trait` |
| 实体搜索 `q` | maxLength **512**（`:15`） | GE 已是 **100** | 本轨不改 GE 上限 |
| `WorkSummary.release_date_precision` | `day, month, year`，与 `release_date` 同在（`repr/resource.go:52`） | `workrepr.Release` 由日期字符串长度派生，闭集相同（`convert.go:92`） | 不收窄。畸形日期两字段都 `null` |
| `WorkSummary` 列表 `include` 能填的块 | 见 `WorkListInclude`；`view=full` 不含 credits | 只要 names/covers/refs/labels 能画出 WorkSummary | 多要的块不要。Maker 来自 companies |
| catalog 列表 `cover.sexual` | `safe/suggestive/explicit/null` | G4.1 已把 v2 槽位等级映进 `WorkRef.cover` | 本轨不改映射；丢掉没有 URL 的封面仍 WARN（G4.1） |

仍然有意丢掉、且打 WARN 的：catalog 不可渲染 / hidden 的作品行（列表与月历）、月历触帽截断、水合时 id 缺席。未映射的资源轴键按词表顺序忽略（WorkSummary 已有的 `inVocabOrder`）。

## 4. 逐条操作

通用：本轨全部 public，不看凭证。会话存储出错仍是 `503`（K3 不把 Redis 故障当匿名，但本轨不读身份）。500 `INTERNAL_ERROR`、503 `SERVICE_UNAVAILABLE`（catalog）。v1 全部 `Cache-Control: no-store`。每个 401（若坏 Bearer 落到本路由——public 不验）仍带 `WWW-Authenticate`。

### 4.1 `listWorks` · `GET /works` · public · 200 `PageList<WorkSummary>`

Query：§3.3。NSFW 在 SQL、LIMIT 之前。一条 COUNT/页谓词。水合缺行 WARN。

| 状态 | code |
|---|---|
| 400 | `UNKNOWN_ENUM_VALUE` `UNKNOWN_SORT` `LIMIT_TOO_LARGE` `INVALID_PARAMETER`（深度 / 日期 / 月份 / 布尔 / 数组格式） |
| 503 | catalog 水合 |
| 500 | SQL |

### 4.2 `listLibraryWorks` · `GET /library-works` · public · 200 `PageList<WorkSummary>`

Query：§3.4。论坛过滤不在 schema。catalog 未配置 / 失败 → 503（旧面 500「目录未启用」归 503）。

| 状态 | code |
|---|---|
| 400 | `UNKNOWN_SORT` `LIMIT_TOO_LARGE` `INVALID_PARAMETER`（`q` 空白、日期、深度） |
| 503 | catalog |

### 4.3 `listReleaseCalendarMonth` · `GET /release-calendar` · public · 200 `ReleaseCalendarMonth`

Query：`month`、`include_nsfw`。跟游标，帽 20×100。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（`month` 格式） |
| 503 | catalog（含翻页中途失败：整段 503，不回半个月） |

### 4.4 `getReleaseCalendarToday` · `GET /release-calendar/today` · public · 200 `ReleaseCalendarToday`

Query：`include_nsfw`。在走完的当前月上算 `has_release`。

### 4.5 `listReleaseCalendarPending` · `GET /release-calendar/pending` · public · 200 `ReleaseCalendarPending`

Query：`year`、`include_nsfw`。catalog `precision=year`。

### 4.6 `listReleaseCalendarTBA` · `GET /release-calendar/tba` · public · 200 `ReleaseCalendarTBA`

Query：`include_nsfw`。catalog `status=unknown`。

### 4.7 `listReleaseCalendarUpcoming` · `GET /release-calendar/upcoming` · public · 200 `ReleaseCalendarUpcoming`

Query：`include_nsfw`。任一失败月 → 503。

### 4.8 `listWorkCollectedMonths` · `GET /works/collected-months` · public · 200 `WorkCollectedMonths`

Query：`include_nsfw`。失败 500。注册在 `getWork` 之前。

### 4.9 `listTags` / `listCompanies` 的 `ids` 加法

已有操作。新增 query `ids`（§3.7）。与 `q` 互斥。其它错误码不变。

## 5. 预分配

### 5.1 迁移

**无。** G 号段下一个空号是 **146**，本轨不用。`galgame.published` / `content_limit` / 资源轴 jsonb / `provider text[]` / 日桶都已在。`galgame_renumber_2026` 删表仍按 g-plan 留给后面某段。

不需要新索引才能开工。若 `/works` 的 jsonb `@>` 在生产 EXPLAIN 里全表扫，再开 146，先告诉编排者。

### 5.2 错误码

不新增。复用：

`INVALID_PARAMETER`（`REQUIRED` / `INVALID_FORMAT` / `OUT_OF_RANGE` / `TOO_SHORT` / `TOO_LONG` / `TOO_MANY_ITEMS` / `INCONSISTENT_WITH`）、`UNKNOWN_ENUM_VALUE`、`UNKNOWN_SORT`、`LIMIT_TOO_LARGE`、`SERVICE_UNAVAILABLE`、`INTERNAL_ERROR`。

非法日期走 `INVALID_PARAMETER` + `parameter: released_from`（或对应键）+ `INVALID_FORMAT`，不再有中文 `detail` 给终端用户看（K8）。

### 5.3 K 决定

| 编号 | 决定 |
|---|---|
| **K-G30** | 本地浏览与 catalog 资料库是两份集合：`/works` 与 `/library-works`。页面钉死。空过滤不切引擎 |
| **K-G31** | `/works` 一条 COUNT/页谓词；NSFW 在 SQL、LIMIT 之前；`content_limit` NULL 放行；每个 `sort` 带同方向 `id` 决胜；未知 `sort` → `400 UNKNOWN_SORT` |
| **K-G32** | `include_resourceless` 默认 false。sitemap 打开它列出全部 published。不再提供 unpublished 列表 |
| **K-G33** | `/library-works` 只声明 catalog 做得到的参数。论坛过滤不进 schema |
| **K-G34** | 月历跟 catalog 游标把窗口走完，帽 20×100，触顶 `is_truncated` + WARN。钟是 Asia/Tokyo（UTC 回落）。`today` 在走完的当前月上算 |
| **K-G35** | `upcoming` 失败的一个月是整段 503，不跳过 |
| **K-G36** | 收录月份失败是 500，不是空数组。人口与 `/works` 默认相同 |
| **K-G37** | 删除 `GET /rss/galgame`。Nitro 读 `/works?sort=resource_updated_desc&limit=20`，用 WorkSummary 拼 feed，条目日期 = `resource_updated_at`（修 RSS 的 LIMIT-前 NSFW）。**编排者 2026-09-24 改判排序键**：旧 feed 按 `created` 排、日期取 `created`；`created` 被懒建本地行的批量创建冲掉（G0 窗口 30 行、09-23 恢复批次），且 WorkSummary 没有 `created_at`，按 `created` 排却用 `resource_updated_at` 当日期会让条目日期乱序。按资源最近更新排，日期与排序同一列，读者按 guid（作品 URL）去重 |
| **K-G38** | 删除 `/search/entity*`。搜索 tab 打 GE 六族 `q`。`/tags` 与 `/companies` 加 `ids`（PageList，缺席即无） |
| **K-G39** | 删掉 `gr.language` / `gr.platform` 标量谓词。`gr.type` 与 jsonb 轴留下 |
| **K-G40** | 水合后 catalog 不可渲染的 id 丢掉并打 WARN；`total` 不减 |

权限不新增。本轨无写面。

## 6. 网页

切到生成的类型化客户端。`include_nsfw` 跟内容姿态（`useContentStance`）。`legacy-fetch-baseline` 按删掉的调用点下调。手写 `GalgameCard` 在这些页面改成 `WorkSummary`（或边上的翻译，与实体页 `workSummaryToCard` 同一条）。

| 文件 | 改什么 |
|---|---|
| `card/Container.vue` + `useGalgameFilters.ts` | `GET /works`。URL 仍可 camelCase；请求键改 `resource_type` / `resource_platforms` / `resource_languages` / `resource_providers` / `excluded_sole_providers` / `sort` / `include_nsfw`。默认 `sort=resource_updated_desc`。去掉 `all` token，缺席即不过滤。类型 / 平台 / 语言改资源轴词表（与实体页 `axes` 同一套；旧分享链接里的 `windows` / `app` / `others` 映射到 `win` / `and` / `oth` / `other`，GE 网页已有这条） |
| `card/Nav.vue` | 收录月份改 `GET /works/collected-months`。网盘多选发新键名。`sortOrder` 拼进 `sort` token |
| `library/Container.vue` + `library/Nav.vue` | `GET /library-works`。`sort` 用资料库五值。只发 `released_from` / `released_to` / `include_nsfw`。不要带资源轴 |
| `calendar/Container.vue` + `Month.vue` / `MonthList.vue` | 五条新路径。`data.month` → `calendar_month`；`meta.count` → `item_count`；upcoming `months` → `entries`。卡片吃 `WorkSummary`。`is_truncated` 时给一句「本月未列完」 |
| `useGalgameReleaseToday.ts` | `GET /release-calendar/today`。继续按 `expires_in` 缓存；键改跟 `include_nsfw` 而不是 cookie |
| `server/utils/kunSitemapSources.ts` | `GET /works?include_resourceless=true&limit=50&sort=resource_updated_desc`（不带 `include_nsfw`）。`pick` 改 `items`；`lastmod` 改 `resource_updated_at`（无资源则为缺席）。`MAX_PAGES` 今天 130×50=6,500 < 9,821 published：提到 200（深度上限刚好 10,000）或按 `total` 停，否则 sitemap 仍截 |
| `server/routes/rss/galgame.xml.ts` | 删对 `/rss/galgame` 的调用。`GET /works?sort=resource_updated_desc&limit=20`。`link` `/galgame/{id}`；标题按 `useWorkName` / `catalogNameText`；`image` = `banner?.url`；`date` = `resource_updated_at`（浏览人口该列 0 个 NULL）；不发作者（WorkSummary 没有）；description 空（没有简介）。缓存策略可保持 5 分钟 |
| `search/Entities.vue` + `utils/search/overview.ts` | 「全部」并发六族 `q`（limit 8）；单族 limit 24。`staff` → `/credit-names`。关键词裁到 100。失败的族是那一条请求失败，不再吞成 0 |
| `filter/EntityMenu.vue` | `family=company` → `GET /companies?q=&limit=20`；`tag` → `GET /tags?q=&include_nsfw=` |
| `useEntityNames.ts` | `GET /tags?ids=` / `GET /companies?ids=`。芯片名字读 `display_name` / `localized` |
| `search/GalgameFilter.vue` + `galgame/tag/Container.vue` | 随 EntityMenu / useEntityNames 走 |
| `search/items.ts` | family 值可以仍是旧 tab 键，请求时映射到路径 |
| `docs/proj/app-direct-api.md` | 列表改 `GET /api/v1/works`；资料库 `GET /api/v1/library-works`（前瞻） |
| `shared/types/galgame.ts` | 删掉已迁的 `GalgameCalendar*` / 列表卡手写类型，或改成生成物别名 |

## 7. 变异题（先于实现提交）

种子：published SFW / NSFW / `content_limit` NULL 各一部，都有资源；一部 published 无资源；两部 `created` 相同（决胜键）；一部 `resource_update_time` 最新但是 NSFW；catalog 有、本地 published 的 hidden 行；一个月历月在假 catalog 里超过 100 部且「今天」排在第 101；upcoming 里一个月返回错误；collected-months 的 Scan 打失败；`/tags?ids=` 含一个成人标签和一个不存在的 id。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | `/works` 默认请求把 NSFW 过滤放到 LIMIT 之后（旧 RSS：先取 10 再丢） | 默认 `sort=created_desc&limit=10` 的 10 条全是 SFW 或 NULL；同一人口按 created 最新的 NSFW 不在页上 |
| 2 | `content_limit IS NULL` 的 published 行被 SFW 闸丢掉 | 默认 GET 含该 id（`list_repo_test.go` 同一断言） |
| 3 | COUNT 与页用不同谓词（页加 NSFW，COUNT 不加） | `total` 等于页上能翻到的行数，不含默认请求里的 NSFW |
| 4 | `id` 决胜改回恒 `DESC`（旧 `list_repo.go:81`） | 两行 `created` 相同、`sort=created_asc`，较小的 id 在前 |
| 5 | 未知 `sort=popularity_desc` 在 `/works` 上回 200 并按资源更新时间排（旧静默回落） | `400 UNKNOWN_SORT` |
| 6 | `include_resourceless` 被忽略 | 默认页没有无资源的 published 行；`include_resourceless=true` 有 |
| 7 | `/library-works` 接受并执行 `resource_type` / `game_type` / `collected_from`（旧 `library=true` 静默丢） | schema 无这些参数；带上它们结果与不带相同（未知参数忽略），且请求仍打 catalog 不打本地 SQL |
| 8 | 月历只读第一页 100、今天排在第 101 | `has_release: true`；月面 `items.length > 100`（假 catalog 给游标） |
| 9 | upcoming 跳过失败月（旧 `:190`） | 那一个月 catalog 500 → 整段 `503 SERVICE_UNAVAILABLE` |
| 10 | collected-months Scan 失败回 `[]` | `500 INTERNAL_ERROR`，不是 200 空数组 |
| 11 | `/tags?ids=` 静默丢掉非法 token / 超过 100 截断（旧 `parseCSVInts`） | 非法 → 400；101 个 id → `TOO_MANY_ITEMS` |
| 12 | `/tags?ids=` 在 `include_nsfw=false` 时仍给成人标签名字 | 该 id 不在 `items`；`total` 不含它 |
| 13 | 水合缺行不打日志（旧 `continue`） | 假 catalog 对某个 paged id 不回行 → 测试抓到 WARN；`items` 短于 `limit`；`total` 仍是 SQL 计数 |
| 14 | `/works` 与 `/library-works` 共用一个 handler、靠 `library` 切换 | 无 `library` 参数；资料库路径不进 `list_repo.ListIDs` |
| 15 | `resource_platforms=windows`（旧网页键）当精确匹配回空页 | `400 UNKNOWN_ENUM_VALUE` |
| 16 | 畸形 `released_from=2026-13` 或 `2026-01-01` 回 200 全表 | `400 INVALID_PARAMETER` |
| 17 | 月历 `precision` 只收 `year`，把 catalog 的 `month` 丢掉当 400 | 本面不暴露 `precision`；pending 发给上游的值是 `year` |
| 18 | catalog 月历 `status=unknown` 被收成闭集之外的值并丢行 | tba 发给上游的就是 `unknown`；假 catalog 回 `release_date_precision=year` 的行仍在 `items` 里（WorkSummary 收下三值） |

## 8. 开放问题

| # | 事实 | 裁决 |
|---|---|---|
| O1 | g-plan 排序字面是 `view_{1d,7d,30d}_desc`，GE 已落地两端 | **收下 GE 的 `WorkSortToken` 全表**（含 `_asc`）。浏览页有统一方向按钮 |
| O2 | 任务书把资源轴写成 `resource_platforms` 复数数组；GE 作品子集合是单值 `resource_platform` | 浏览用复数逗号形（页面今天是单选，发一项的数组）。GE 子集合不改 |
| O3 | 浏览页不筛 `resource_runtimes` | 本轨不声明该参数 |
| O4 | `/library-works` 与 X2 `/search/works` 同一 catalog 人口 | 两份集合：搜索 `q` 必填、条目 `WorkRef`；资料库 `q` 可缺席、条目 `WorkSummary`、默认热度 |
| O5 | catalog `q` 上限 512，本面 107 | 论坛搜索上限，不放宽 |
| O6 | 资料库 `page`→`cur_` 编的是页码不是 offset（census 发现 32） | 沿用现改写。若 catalog 把游标当 offset，第 2 页会错；本轨不单开修复 |
| O7 | sitemap `MAX_PAGES=130`×50 < 9,821 published | API 深度 10,000 够用。网页把页数抬到 `ceil(total/50)` 且 ≤200 |
| O8 | RSS 按 `created_desc` 取 20，G0 窗口的懒建行仍可能占满「最新」 | **改判（K-G37）**：`resource_updated_desc`，日期同列 |
| O9 | WorkSummary 没有 `created_at` / 作者 / 简介 | RSS 日期用 `resource_updated_at`（浏览人口 0 个 NULL，与排序同列）；不发作者；description 空。旧 feed 的作者与简介丢掉，接受（WorkSummary 不为 RSS 加字段） |
| O10 | upcoming 单月帽 500 vs 月面 2,000 | 扇出 24 路的预算。触顶的月带 `is_truncated` |
| O11 | `year` 的 G8 同伴是 1970–9999 | shape() 不含 minimum。实现若另有门再把下限写成 1 |
| O12 | Flutter 前瞻文档仍写 `GET /api/galgame` | 实现 PR 改 `app-direct-api.md` |
| O13 | 实体「全部」从 1 个旧请求变成 6 个 GE 请求 | 接受。与 X2 拆 overview 同一取舍 |
| O14 | `/users?ids=` 是 `BatchList`（有 `missing`）；标签 / 会社是 `PageList`（缺席即无） | 按任务书。芯片冷链不需要区分「没有」和「不可见」 |
| O15 | GE 带 `q` 的深度 100；搜索单族 limit 24 只能翻到第 4 页 | 与 GE D5 相同。本轨不抬 |

---

删旧路由时：`GET /galgame`、`GET /galgame/calendar`、`GET /galgame/calendar/{today,pending,tba,upcoming}`、`GET /galgame/collected-calendar`、`GET /rss/galgame`、`GET /search/entity`、`GET /search/entity/resolve`，以及只被它们用的 handler 方法 / DTO / RSS repo 的 `FindRecentWorkIDs`。`legacy_route_baseline` −10。`rg` 证明零调用方，含 `apps/web/server/` 与 `../kungal-apps`。
