# X2-R · 排行与首页

> 2026-09-23 立。分支 `api-v1/x2-ranking`，迁移号段 204–205（**本轨不用**）。
> 契约与变异题同一个提交，早于实现。

## 1. 普查

### 1.1 路由与调用方

四条旧路由由两个 handler 独占：`RankingHandler`（`internal/ranking/**`）与 `HomeHandler`（`internal/home/**`），两个包除 `app.go` 外没有任何导入方。

| # | 旧 | 方法 | 档 | 调用方 |
|---|---|---|---|---|
| 1 | `GET /api/ranking/galgame` | `GetGalgameRanking` | 公开 | `components/ranking/Galgame.vue` |
| 2 | `GET /api/ranking/topic` | `GetTopicRanking` | 公开 | `components/ranking/Topic.vue` |
| 3 | `GET /api/ranking/user` | `GetUserRanking` | 公开 | `components/ranking/User.vue` |
| 4 | `GET /api/home` | `GetHome` | 公开 | **无** |

`apps/web/server/`（Nitro）零调用；`../kungal-apps` 零调用；`docs/proj/app-direct-api.md` 没有提到这四条。

`/api/home` 自 `0bbade0a9`（2026-06-22，首页改成五个页签的动态流）起就没有调用方了，只剩 `shared/types/home.ts` 的类型还被搜索结果的类型借用。

### 1.2 生产取值（`kungalgame`，2026-09-23）

- `galgame`：已发布 9,773；其中有资源 9,209；已发布且本地 `content_limit = 'nsfw'` 4,970（**51%**）；`creator_user_id` 为空 152。
- `galgame_rating`：4,117 行、1,581 个作品；**没有状态列**。
- `galgame_resource`：`status` 0 47,306、1 3,429（下架）。
- `topic`：未隐藏 3,243（公开 3,236，其中 NSFW 190；仅登录 7），隐藏 322。
- `topic_reply`：`status` 0 14,881、1 3；`topic_comment` 3,165 全部 `status` 0。
- `kungal_user_state`：100,817 行。
- **第 50 名附近的并列**：按资源数排时，第 50 名附近每个值有 14–41 个作品并列（29 × 14、28 × 21、27 × 23…）；按浏览数则基本不并列。

### 1.3 语义与疑似 bug

1. **`/api/home` 零调用方**（§1.1）。带 60 秒 Redis 缓存的整个 `HomeService` 都是死代码。v1 的动作是删，不是迁。
2. **网页只要前 50 名**：三个排行永远是 `page=1&limit=50&sort_order=desc`，没有分页器，`asc` 从没被请求过。
3. **排序没有决胜键**。按资源数排时第 50 名附近几十个并列，前 50 的集合与顺序每次请求都可能不同。
4. **SFW 的作品排行先 `LIMIT` 再滤 NSFW**：SQL 取前 50，再按 catalog 简表的 `content_limit` 丢掉 NSFW。已发布作品一半是 NSFW，所以 SFW 访客的「前 50」通常只剩二十几条。catalog 失败时整页静默回空数组。
5. 话题与用户排行在 `LIMIT` 之后丢掉被封禁作者 / 用户，返回少于 `limit` 条。保留这个做法（封禁判据只有 OAuth 知道），但名次在丢弃之后连续编号。
6. **用户排行的计数口径不一致**：话题数只数未隐藏、公开的话题，回复 / 评论 / 资源却连隐藏的、下架的、以及隐藏话题与仅登录话题里的都数。
7. 隐式输入：SFW 取自偏好 cookie（`utils.IsSFW`）。v1 改成显式 `include_nsfw`。
8. 作品排行的 `user` 是本地 `creator_user_id`（建页的人），不检查是否可渲染；OAuth 查不到时回「已注销用户」这个中文占位名。
9. 旧错误全是 `233`；排序字段写错回中文句子。
10. `Hydrate` 吞掉 OAuth 错误：OAuth 不可用时整页作者都变成「已注销用户」，封禁用户也会混进来（占位用户的 `status` 是 0）。

## 2. 范围

| 旧 | v1 |
|---|---|
| `GET /api/ranking/topic` | `GET /api/v1/rankings/topics` |
| `GET /api/ranking/user` | `GET /api/v1/rankings/users` |
| `GET /api/ranking/galgame` | `GET /api/v1/rankings/works` |
| `GET /api/home` | **删除，不迁**（§1.3 第 1 条） |

删 4 条，`legacy_route_baseline` 159 → **155**（以合并时 rebase 后重新生成为准）。

## 3. 形状

### 3.1 集合的形态（K-X2R1）

**K-X2R1 · 排行是「前 N 名」列表，不是页码集合。** 01 §4 把「排行」列在页码集合里，但唯一的消费者只读固定的前 50 名、没有分页器；排行的深页没有意义，页码集合还得每次多数一次 `total`。所以：

- 参数 `limit`：1–100，默认 50；超过 100 → `400 LIMIT_TOO_LARGE`（不夹取）。没有 `page`、`cursor`、`total`。
- 响应 `{ "object": "list", "items": [...] }`（`repr.List`，不发 `next_cursor`）。
- 以后真要翻页，是加法。

**K-X2R2 · 只有降序。** `asc` 从没被请求过（A5）；排序 token 形如 `views_desc`，与 `listTopics` 的词形一致。每个排序都以 `id DESC` 决胜（用户排行以 `user_id DESC`）。

**K-X2R3 · 用户排行的计数只数匿名访客看得见的东西**：话题 = 未隐藏且公开的话题；回复 / 评论 = `status = 0` 且所在话题未隐藏、公开；资源 = `status = 0`。与话题数原本的口径对齐（§1.3 第 6 条）。

**K-X2R4 · `include_nsfw` 在 SQL 里、`LIMIT` 之前生效**（§1.3 第 4 条）。话题用 `is_nsfw`；作品用本地 `content_limit` 缓存列，谓词与 `list_repo` 的 SFW 闸相同（`content_limit IS NULL OR content_limit = 'sfw'`，display_nsfw 编辑轴）。

### 3.2 条目

名次 `rank`：1 起，按返回顺序连续编号（丢弃不可渲染的条目之后再编）。
指标值 `metric_value`：该排序的数值（计数为整数值，评分为两位小数）。不叫 `value`——正文节点里的 `value` 是字符串（G8）；不叫 `score`——`Website.score` 是整数（G8）。

**`TopicRankingEntry`**（`object: "topic_ranking_entry"`）：`rank`、`metric_value`、`topic`。
`topic` 是 `object: "topic"` 的又一个摘要形状（K10）：`id`、`title`、`author`（UserRef），全部与 `Topic` 同名同型。

**`UserRankingEntry`**（`object: "user_ranking_entry"`）：`rank`、`metric_value`、`member`（UserRef）、`bio`（string | null，与 `UserProfile.bio` 同型）。人叫 `member`：`user` 是禁用名。

**`WorkRankingEntry`**（`object: "work_ranking_entry"`）：`rank`、`metric_value`、`work`（共享的 `repr.WorkRef`，由 `galgameapiv1.WorkRefOf` 从 catalog 行映射；协调会话 2026-09-23 裁决：v1 一律叫 `work_id`、集合名 `works`、对象 `work`，嵌入时属性叫 `work`）、`creator`（UserRef | null：本地建页人；未记录、查不到或不可渲染时为 `null`）。

### 3.3 操作

| operationId | 路径 | 档 | 参数 |
|---|---|---|---|
| `listTopicRanking` | `GET /rankings/topics` | 公开 | `sort`：`views_desc`（默认）`replies_desc` `comments_desc` `likes_desc` `upvotes_desc` `favorites_desc`；`limit`；`include_nsfw`（默认 `false`） |
| `listUserRanking` | `GET /rankings/users` | 公开 | `sort`：`moemoepoint_desc`（默认）`topics_desc` `replies_desc` `comments_desc` `resources_desc`；`limit` |
| `listWorkRanking` | `GET /rankings/works` | 公开 | `sort`：`views_desc`（默认）`likes_desc` `favorites_desc` `resources_desc` `rating_desc`；`limit`；`include_nsfw`；`include_resourceless`（默认 `false`，旧 `show_no_resource`） |

可见性：

- 话题排行：未隐藏、`access_scope = 'public'`、`include_nsfw` 为假时排除 NSFW；作者被封禁或在 OAuth 查不到的话题跳过。
- 用户排行：OAuth 查不到或不可渲染的用户跳过。
- 作品排行：只含 `published`；`include_resourceless` 为假时只含有资源的作品（与 `/galgame` 列表同一个 `EXISTS` 谓词）；catalog 没返回的行（查无、被合并、不可渲染）跳过。

### 3.4 错误

| status | code | 何时 |
|---|---|---|
| 400 | `UNKNOWN_SORT` | `sort` 不在该集合的词表里 |
| 400 | `LIMIT_TOO_LARGE` | `limit` > 100 |
| 400 | `INVALID_PARAMETER` | `limit` < 1、布尔参数不是 `true` / `false` |
| 503 | `SERVICE_UNAVAILABLE` | OAuth `/users/batch` 不可用（不再像 `Hydrate` 那样吞掉，§1.3 第 10 条）；作品排行另含 catalog 不可用（旧面静默回空数组） |
| 500 | `INTERNAL_ERROR` | — |

## 4. 预分配

- 迁移：**无**。
- 错误码：**无新增**。
- K 编号：**K-X2R1…**（并行会话撞号，沿用 TS/P 的前缀做法）。

## 5. 网页

- `components/ranking/{Galgame,Topic,User}.vue` 换类型化客户端；作品行的图从 16:9 横幅换成 `work.cover`（竖版原图），名字按「优先原名」设置在 `display_name` / `localized` / `latin` 之间挑（与服务端旧的 `CatalogEntityName` 同一规则）；`include_nsfw` 取 `useContentStance().allowsNsfw`（与话题列表相同）；排序选择器的值改成 token。
- `components/ranking/pageData.ts`、`validations/ranking.ts`、`shared/types/ranking.ts` 随之收缩；`shared/types/home.ts` 里没人用的类型删掉（被搜索类型借用的部分另议，归 X2-search）。
- `legacy-fetch-baseline` 按删掉的调用点下调。

## 6. 变异题（先于实现提交）

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 话题排行不排除隐藏话题 | 隐藏话题不在结果里 |
| 2 | 话题排行不查 `access_scope` | 仅登录话题不在结果里 |
| 3 | 话题排行忽略 `include_nsfw` | 默认不含 NSFW 话题；`include_nsfw=true` 才含 |
| 4 | 话题排行去掉 `id DESC` 决胜键 | 种子里多个话题浏览数相同，结果顺序与 SQL `ORDER BY view DESC, id DESC` 逐条相等 |
| 5 | 不跳过被封禁作者的话题 | 封禁作者的话题不在结果里 |
| 6 | 名次取 SQL 位置而非丢弃后的位置 | 丢掉一条之后名次仍是 1..n 连续 |
| 7 | 用户排行的回复数不滤 `status` | 隐藏回复不计入 |
| 8 | 用户排行的回复数不看所在话题 | 隐藏话题里的回复不计入 |
| 9 | OAuth 失败时像 `Hydrate` 一样降级 | OAuth 500 → `503 SERVICE_UNAVAILABLE` |
| 10 | 作品排行不滤 `published` | 未发布作品不在结果里 |
| 11 | 作品排行忽略 `include_resourceless` | 默认不含没有资源的作品；`include_resourceless=true` 才含 |
| 12 | 作品排行的 NSFW 在 `LIMIT` 之后才滤（去掉 SQL 里的 `content_limit` 谓词，靠 catalog 丢行） | 前几名有 NSFW 作品时，`limit=3` 的 SFW 结果仍有 3 条 |
| 13 | 评分排序去掉 `id DESC` 决胜键 | 种子里两部作品的评分完全相同，顺序与 SQL 逐条相等 |
| 14 | catalog 失败时回空列表 | catalog 500 → `503 SERVICE_UNAVAILABLE` |
