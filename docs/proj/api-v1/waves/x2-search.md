# X2-search · 站内搜索

> 2026-09-23 立。分支 `api-v1/x2-search`，迁移号段 202–203（**本轨不用**）。
> 契约与变异题同一个提交，早于实现。作品引用用协调者裁决的共享 `repr.WorkRef`（#199，`internal/apiv1/repr/work.go`，映射 `galgameapiv1.WorkRefOf`）；命名一律 `work_id` / `works` / `work`（#198）。

## 1. 普查

### 1.1 路由与调用方

六条旧路由全部由 `SearchHandler`（`internal/search/handler/search_handler.go`）独占。服务层 `SearchService` 另外依赖 galgame 域的 `GalgameEnricher` / `EntitySearchService` / `ResourceService`、toolset 域的 `ToolsetService`、community 客户端与锚点解析器；这些都是只读调用，不共用 handler。

| # | 旧 | 方法 | 档 | 调用方 |
|---|---|---|---|---|
| 1 | `GET /api/search?type=topic\|galgame\|resource\|user\|reply\|comment` | `Search` | optional | `components/search/List.vue`（搜索页各分区）、**`pages/admin/user.vue`（用户内容管理的用户搜索，`type=user`）** |
| 2 | `GET /api/search/quick` | `QuickSearch` | optional | `components/search/Palette.vue`（Ctrl-K 命令面板） |
| 3 | `GET /api/search/overview` | `Overview` | optional | `components/search/Container.vue`（「全部」分区 + 左栏计数，每个分区都拉） |
| 4 | `GET /api/search/gal-comment` | `SearchGalComments` | optional（不看身份） | `components/search/GalComments.vue` |
| 5 | `GET /api/search/entity` | `SearchEntities` | optional（不看身份） | `components/search/Entities.vue`，以及 **`components/filter/EntityMenu.vue`（`/galgame` 筛选栏的会社/标签选择器，G 轨的页面）** |
| 6 | `GET /api/search/entity/resolve` | `ResolveEntities` | optional（不看身份） | `composables/useEntityNames.ts`（筛选 chip 冷链接还原名字，`/galgame` 与搜索页共用） |

`apps/web/server/`（Nitro）零调用；`../kungal-apps` 零调用。

### 1.2 生产取值（`kungalgame`，2026-09-23）

- 话题：published 3,243（`public` 3,236 / `login` 7；其中 NSFW 192），hidden 322。`role` / `users` 访问范围 0 行。
- 回复：`status=0` 14,881、`status=1` 3；其中 978 条挂在隐藏话题下、19 条挂在 `login` 话题下。
- 评论：`status=0` 3,165。
- 三张表的排序时间戳（`status_update_time` / `created`）零并列；相关度分数本身大量并列。
- 回复最长 6,617 字，评论最长 1,000 字。
- galgame 16,143 行，已发布 9,773。

### 1.3 语义与疑似 bug

1. **`type=` 是一条路由里的六种资源**：topic / galgame / resource / user / reply / comment 各回各的形状，信封是 `Paginated{items,total}`；未知 `type` 由校验挡住，但 `default` 分支回 `[]`（不可达）。
2. **NSFW**：话题车道（含 overview / quick）**不过滤 NSFW 话题**——匿名访客的搜索结果里有 192 个 NSFW 话题，只靠前端模糊；回复/评论车道同样不看所在话题是否 NSFW。galgame / 资源 / 实体车道走 `utils.IsSFW`（账号姿态或 `KUNGalgameSettings` cookie）。v1 一律改成显式 `include_nsfw`（01 §3），与 `/topics` 列表同一个默认值 `false`。
3. **封禁作者在取数之后才剔除**：话题/回复/评论车道先按 SQL 算 `total`、取一页，再逐行 `IsRenderable` 丢掉封禁作者（回复/评论还会丢掉「所在话题的作者被封」的行）。`total` 与 `items` 不是同一个谓词，一页可能少几行。
4. **排序无 id 决胜键**：`relevance DESC, status_update_time|created DESC`。时间戳零并列所以今天是确定的（memory `kungal-forum-search`），但 v1 集合规则要求带 id。
5. **摘要是 Markdown 源码的切片**：回复/评论的 `content` 是 `SUBSTRING(原文)`，里面有 `**`、`![](/image/…)`、链接语法，前端当纯文本高亮显示。社区评论车道已经先 `markdown.ToPlainText` 再开窗，两边不一致。
6. **用户车道**：OAuth `/users/search` 只有 `q` + `limit`（上限 50），没有 offset，于是一次抓 50 在本地切页，`total` 最多 50——**第 51 个同名用户永远搜不到，而 `total` 说「共 50 位」是个确数**。`created` 由 OAuth 给，可能缺席。
7. **galgame 车道把 SFW 闸写死成 `false`**：`SearchGalgames(..., false, filter)`——参数名 `isSFW`，传的是字面量 `false`，所以搜索页的 Galgame 分区对 SFW 读者**也不过滤 NSFW 作品**，而同页的资源/实体分区是过滤的。overview / quick 同样传 `false`。
8. **失败车道与「没搜到」不可区分**：overview / quick 把失败的车道吞成空数组 + 计数 0。
9. `entity` 的 `family` 校验允许 `series` / `engine`，但 catalog 的实体搜索闭集没有这两个（memory：发 `series` 是 400）——这两个 family 在服务层被 `IsEntityFamily` 拦成中文 400，还是上游 400，待实现时确认。
10. **社区评论车道**的 `q` 必须 2–100 个字符（上游约束），其余车道 1–107。
11. 所有错误都是 `233` + 中文。

## 2. 范围

**一个车道一个集合**（K10：形状按操作静态确定），不再用 `type=` 在一条路由里切形状；也**不再有「总览」与「快速」两个组合面**——搜索页的「全部」分区、左栏计数与 Ctrl-K 面板由网页并发请求各车道自己拼（各车道自带 `total`，某条车道失败就是那一条请求失败，天然与「搜到 0 条」可区分，修 1.3 第 8 条）。

| 旧 | v1 | 本 PR |
|---|---|---|
| `/search?type=topic` | `GET /api/v1/search/topics` | ✅ |
| `/search?type=reply` | `GET /api/v1/search/replies` | ✅ |
| `/search?type=comment` | `GET /api/v1/search/comments` | ✅ |
| `/search?type=user`（含 `pages/admin/user.vue`） | `GET /api/v1/search/users` | ✅ |
| `/search?type=galgame` | `GET /api/v1/search/works` | ✅ |
| `/search/gal-comment` | `GET /api/v1/search/wall-comments` | ✅ 删旧 |
| `/search/quick` | 网页并发请求 topics / works / users 各 5 条 | ✅ 删旧 |
| `/search/overview` | 网页并发请求各车道 | ✅ 删旧 |
| `/search?type=resource` | —— | **留旧**：条目是 G 轨的资源摘要，形状未定 |
| `/search/entity` | —— | **留旧**：条目是 GE 轨的实体引用，形状未定；`/galgame` 筛选栏也用它 |
| `/search/entity/resolve` | —— | **留旧**：应当是 GE 实体集合的 `ids=` 批量读 |

本 PR 删 3 条（`/search/overview`、`/search/quick`、`/search/gal-comment`），`legacy_route_baseline` 下调 3（以合并时 rebase 后重新生成为准）。旧 `/search` 收窄成只剩 `type=resource`；搜索页的 Gal 工具分区本来就走 toolset 域的旧 `/toolset?query=`，不属本轨。

## 3. 共同规则

- 关键词参数叫 **`q`**（与 catalog `/v2` 同名）：1–107 字符（K19，按原始值计长度），去首尾空白后为空 → `400 INVALID_PARAMETER`，`parameter: q`，`reason: TOO_SHORT`，`params.min_length = 1`。
- **页码集合**（K11）：`collect.PageNumber`（`page` ≥1、`limit` 1–100，默认 20；网页固定发 24），深度上限 10000，`total` + `total_relation` 用 `collect.ClampTotal`。排序固定为相关度降序，**id 作最终决胜键**，没有 `sort` 参数（works 车道除外）。
- **`include_nsfw`**（布尔，默认 `false`）：话题车道排除 NSFW 话题；回复/评论车道排除挂在 NSFW 话题下的行；works 车道对应 catalog 的 SFW 闸（修 1.3 第 7 条）。网页按内容姿态传（与 `/topics` 列表同一个 `allowsNsfw`）。
- 档：topics / replies / comments / users 是 `optional`——登录用户额外看得到 `login` 访问范围的话题（及其回复/评论），与共享列表同一个谓词 `SharedListPredicate`；`role` / `users` 范围永不进搜索；Bearer 同样算已登录。works / wall-comments 是 `public`。
- 封禁作者：见 K-X2S2。

## 4. 形状

### 4.1 `GET /search/topics` → `PageList[TopicSummary]`

条目就是 `/topics` 列表的 `TopicSummary`，**同一个类型、同一个映射函数**：话题域导出一个按行渲染摘要的方法（新文件 `topic/apiv1/summaries.go`，不改已有代码）。谓词：`status != hidden`、`SharedListPredicate`、`include_nsfw`、每个关键词都要在 title / content / category 之一里出现（`ILIKE`）。排序：相关度（title 8 / category 3 / content 1，相邻命中加分，沿用 `relevance.go`）降序 → `bumped_at` 降序 → `id` 降序。

### 4.2 `GET /search/replies` → `PageList[ReplySearchHit]`

| 字段 | 类型 | 说明 |
|---|---|---|
| `object` | `"reply"` | 与 `Reply` 同一个 object，另一种摘要形状（K10） |
| `id` | DecimalID | |
| `topic_id` | DecimalID | |
| `topic_title` | string ≤ 233 | 所在话题标题。自由文本 |
| `floor` | integer ≥ 1 | |
| `excerpt` | string ≤ 240 | **纯文本**摘要：正文先转纯文本，再以最早命中处前 30 字为起点开 233 字的窗，被截的开头补 `…`。自由文本 |
| `author` | UserRef | |
| `created_at` | DateTime | |

谓词：回复 `status = 0`，所在话题可见（同 4.1 的话题谓词，含 `include_nsfw`），每个关键词都在正文里。排序：相关度降序 → `created_at` 降序 → `id` 降序。

### 4.3 `GET /search/comments` → `PageList[CommentSearchHit]`

同 4.2，`object: "comment"`，没有 `floor`。网页的定位链接仍是 `/topic/{topic_id}?comment={id}`。

### 4.4 `GET /search/users` → `PageList[UserSearchHit]`

| 字段 | 类型 | 说明 |
|---|---|---|
| `object` | `"user"` | 与 `UserRef` / `UserProfile` 同一个 object |
| `id` / `name` / `avatar` | 同 `UserRef` | |
| `bio` | string ≤ 107 \| null | 同 `UserProfile.bio` |
| `roles` | `UserRole` 数组（creator / moderator / admin / ren） | 同 `UserProfile.roles`，同一个 Go 类型 |
| `registered_at` | DateTime \| null | 账号注册时间，账号服务没给就 `null`。不叫 `created_at`：`UserProfile.created_at` 非空且有本站回落，语义不同 |
| `topic_count` / `reply_count` | integer ≥ 0 | 本站可见话题数（共享列表谓词，不含 NSFW 过滤）/ 可见回复数 |

**K-X2S1 · 用户车道的总数封顶在 50，关系是 `gte`。** 上游 `/users/search` 没有 offset、上限 50。拿满 50 条时 `total_relation = "gte"`（「50+」）；不满 50 条时是确数 `eq`。封禁/注销用户（`status != 0`）剔除后再计数，所以 `total` 与 `items` 同谓词。超出的页是空页（深度上限仍是 10000，只是上游够不着）。

### 4.5 `GET /search/works` → `PageList[WorkRef]`

条目就是共享的 `repr.WorkRef`（`object: "work"`、`id` 即 catalog work id、`display_name` / `latin` / `localized{}`、竖版原图 `cover`、展示轴 `is_nsfw`），由 `galgameapiv1.WorkRefOf` 从 catalog 行映射，**不自定义任何作品字段**。于是搜索页的 Galgame 分区不再显示评分、平台、浏览/点赞这些本站统计——那是 G 轨的作品摘要，等它有了再换。claim 为 `hidden` 等不可渲染的行照旧剔除（`CatalogItemRenderable`）。

参数：`company_id`、`tag_ids`（逗号形数组，≤ 10）、`released_from` / `released_to`（`YYYY` 或 `YYYY-MM`；前者晚于后者 → `400 INVALID_PARAMETER`，`INCONSISTENT_WITH`）、`sort` ∈ `relevance_desc` `popularity_desc` `updated_desc` `released_desc` `released_asc`（默认 `relevance_desc`，封闭词表，未知 → `400 UNKNOWN_ENUM_VALUE`）、`include_nsfw`。`total` 是 catalog 的计数，超过深度上限记 `gte`。catalog 失败 → `503`。

### 4.6 `GET /search/wall-comments` → 游标集合 `List[WallCommentSearchHit]`

上游 community 是 keyset、无 total，所以这是**游标集合**：`next_cursor` 用 `collect.EncodeCursor` 包住上游游标并绑定 `q`；换 `q` 复用旧游标 → `400 INVALID_CURSOR`。`q` 去空白后 2–100 字符（1.3 第 10 条），不足 → `400 INVALID_PARAMETER`（`TOO_SHORT`）。`limit` 1–50，默认 20。

| 字段 | 类型 | 说明 |
|---|---|---|
| `object` | `"wall_comment"` | 与 RC 的 `WallComment` 同一个 object |
| `id` | DecimalID | |
| `subject_type` / `subject_id` | 同 `WallComment`（同一个 Go 类型） | |
| `subject_path` | string，站内路径 | 该墙所在页面的路径（`/galgame/4121`、`/website/example.com`）：网站墙按 host 寻址，客户端从 id 拼不出来 |
| `work` | `WorkRef` \| null | `subject_type = galgame` 时的作品；其余为 `null`，catalog 没回答时也是 `null` |
| `excerpt` | 同 4.2 | |
| `author` | UserRef | |
| `created_at` | DateTime | |

别站开的墙、已退役的来源、封禁作者的帖子在取数之后剔除，一页可能少于 `limit`。community 或账号服务不可用 → `503`。

## 5. 逐条裁决

| 旧面 | 裁决 |
|---|---|
| `type=` 一条路由多种形状 | 一车道一集合（§2） |
| overview / quick 组合面 | 删除，网页并发请求各车道（§2） |
| 话题/回复/评论不过滤 NSFW | `include_nsfw`，默认 `false` |
| galgame 车道 SFW 写死 `false` | 按 `include_nsfw` |
| 摘要是 Markdown 切片 | 纯文本 `excerpt`（§4.2） |
| 无 id 决胜键 | 加 id |
| 失败车道吞成 0 | 每条车道一个请求，失败就是失败 |
| 用户车道 `total` 假装确数 | K-X2S1 |
| 封禁作者取数后剔除 | K-X2S2 |

**K-X2S2 · 封禁作者照旧在取数之后剔除，`total` 不减。** 封禁状态只有 OAuth 知道（`IsRenderable`），没法下推进 SQL 谓词。页码集合的 `total` 因此是「匹配的可见内容数，含随后被发现作者已封禁的行」，一页可能少几行。与 `/topics` 列表相同的取舍；写进操作描述。

**K-X2S3 · 搜索没有组合面。** 一个请求回九种形状，就是 `type=` 换了个位置；而且组合面只能把失败的车道吞成空，客户端分不清。车道并发由客户端做，HTTP/2 下多几个请求不贵。

## 7. 预分配

- 迁移：**无**。
- 错误码：**无新增**。用到 `INVALID_PARAMETER`（`TOO_SHORT` / `TOO_LONG` / `OUT_OF_RANGE`）、`UNKNOWN_ENUM_VALUE`、`INVALID_CURSOR`、`LIMIT_TOO_LARGE`（若 PageNumber 用它）、`INVALID_CREDENTIAL`、`SERVICE_UNAVAILABLE`、`INTERNAL_ERROR`。

## 8. 网页

- `components/search/{Container,List,Overview,Palette,GalComments,Result,…}.vue`：topics / replies / comments / users / works / wall-comments 换类型化客户端；「全部」分区与左栏计数改成并发请求各车道（资源、资料库、工具三条仍走旧面）；Ctrl-K 面板并发请求三条车道。
- 卡片：话题卡吃 `TopicSummary`、回复/评论/用户/墙帖卡吃各自的 hit；Galgame 分区用 `WorkRef` 画一张只有封面与名字的卡（名字按 `CatalogName` 的规则选：`localized[locale] ?? display_name ?? latin`）。
- `pages/admin/user.vue` 的用户搜索换 `GET /search/users`。
- `components/filter/EntityMenu.vue`、`composables/useEntityNames.ts` 不动（等 GE）。
- `shared/types/search.ts` 删掉已迁车道的手写类型。

## 9. 变异题（先于实现提交）

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 话题车道去掉 `status != hidden` | 隐藏话题的标题含关键词 → 不在结果里 |
| 2 | 话题车道不看 `include_nsfw` | 默认请求里没有 NSFW 话题；`include_nsfw=true` 才有 |
| 3 | 话题车道 `SharedListPredicate` 恒按已登录 | 匿名搜不到 `login` 话题，登录后搜得到 |
| 4 | 回复车道不查所在话题可见性 | 隐藏话题下的回复不在结果里 |
| 5 | 回复车道不看所在话题的 NSFW | 默认请求里没有 NSFW 话题下的回复 |
| 6 | 排序去掉 `id` 决胜键 | 种子里 7 行相关度与时间戳完全相同且跨页，`limit` 2 / 3 翻完所有页，顺序是 id 降序、无重无漏 |
| 7 | 摘要不转纯文本 | 正文含 `**粗体**` 与图片 token 的回复，`excerpt` 里没有 `**` 与 `/image/` |
| 8 | 用户车道拿满 50 条仍报 `eq` | 假 OAuth 回 50 条 → `total_relation = "gte"` |
| 9 | 用户车道不剔除 `status != 0` | 假 OAuth 回一个封禁用户 → 不在结果里、不计入 `total` |
| 10 | 空白 `q` 放行 | `q="   "` → `400 INVALID_PARAMETER`，`parameter: q`，`TOO_SHORT` |
| 11 | works 车道的 SFW 闸不看 `include_nsfw`（写死） | 默认请求带 SFW 闸，`include_nsfw=true` 不带 |
| 12 | 墙帖游标不绑定 `q` | 拿 `q=汉化` 的游标去翻 `q=其他` → `400 INVALID_CURSOR` |
