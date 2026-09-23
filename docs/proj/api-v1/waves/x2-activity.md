# X2 · 动态流（activity）

> 2026-09-23 立。分支 `api-v1/x2-activity`，迁移号 206（**本轨不用**）。
> 契约与变异题同一个提交，早于实现。

## 1. 普查

### 1.1 路由与调用方

三条旧路由由 `ActivityHandler`（`internal/activity/handler/activity_handler.go`）独占，服务层只被它用（`internal/admin/service/overview_service.go` 只引用了 `feed_activity` 的类型字符串，不用这个服务）。

| # | 旧 | 调用方 | 取值 |
|---|---|---|---|
| 1 | `GET /api/activity?type=&cursor=&limit=&show_no_resource=` | `components/activity/Container.vue`（`/activity/category`，按类型筛选） | `type=all` 等于第 3 条 |
| 2 | `GET /api/activity/tab?types=\|tab=&cursor=&limit=&show_no_resource=&force_sfw=` | `components/home/ActivityFeed.vue`（首页动态页签） | 网页只发 `types`；`tab=` 的五个内置页签（`homeTabTypes`）**零调用方** |
| 3 | `GET /api/activity/timeline?cursor=&limit=&show_no_resource=` | `components/activity/Timeline.vue`（`/activity`） | |

`apps/web/server/` 零调用；`../kungal-apps` 零调用。三条都是公开档，NSFW 立场取自 `ContentStance` 中间件（cookie / Bearer 的立场），首页「全部」页签另发 `force_sfw=true`。

首页页签的 `types` 来自持久化的设置 `feedTabs`（`setting-panel/components/FeedTabs.vue`，词表在 `constants/activity.ts` 的 `KUN_FEED_KIND_GROUPS`），存的是旧的大写类型名外加两个伪类型 `TOPIC_NORMAL` / `TOPIC_RESOURCE_HELP`。**v1 不改存储格式**，网页在发请求时翻译（§6）。

### 1.2 生产取值（`kungalgame.feed_activity`，2026-09-23，共 100,904 行）

| 类型 | 行数 | 其中 NSFW | 说明 |
|---|---|---|---|
| GALGAME_RESOURCE_CREATION | 50,735 | 0 | |
| TOPIC_REPLY_CREATION | 13,884 | 405 | `content` 是**整条回复的 Markdown 源**（含 `kungal-user:` / `kungal-reply:` token） |
| GALGAME_COMMENT_CREATION | 10,235 | 0 | 社区帖正文（纯文本），仍在增长 |
| GALGAME_CREATION | 9,773 | 0 | **全部 `user_id = 0`**：作者取 `galgame.creator_user_id` |
| GALGAME_RATING_CREATION | 4,117 | 0 | |
| TOPIC_CREATION | 3,236 | 190 | `content` 是标题 |
| TOPIC_COMMENT_CREATION | 3,052 | 82 | `content` 是前 100 字 |
| UPDATE_LOG_CREATION | 1,234 | 0 | 最后一条 2026-06-04 |
| TOPIC_UPVOTE | 918 | 15 | `content` 是推的备注 |
| **MESSAGE_UPVOTE** | 895 | 14 | 见 §1.3 第 2 条 |
| GALGAME_EDIT / PR_CREATION | 773 / 471 | 0 | |
| GALGAME_QUIZ_CREATION | 267 | 0 | `content` 是题干 |
| TODO_CREATION | 261 | 0 | |
| GALGAME_RATING_COMMENT_CREATION | 232 | 0 | 156 行带 work_id |
| GALGAME_RESOURCE_COMMENT_CREATION | 185 | 0 | |
| GALGAME_WEBSITE_COMMENT_CREATION / WEBSITE_CREATION | 119 / 87 | 21 / 19 | |
| TOOLSET_CREATION / RESOURCE_CREATION / COMMENT_CREATION | 119 / 110 / 74 | 0 | |
| MESSAGE_SOLUTION | 73 | 9 | `content` 是通知行里的回复摘录 |
| GALGAME_QUIZ_COMMENT_CREATION | 54 | 0 | |

- `created` 有 376 个重复值（100,904 行 / 100,528 个不同时刻），游标必须带决胜键。表上的唯一约束是 `(type, source_id)`，现行键集 `(created, type, source_id)` 是全序。
- 可见性由触发器维护：隐藏 / 非公开话题的 TOPIC_CREATION 与其下的回复、评论、推、通知镜像行都会被删掉。实测：隐藏或非公开话题的动态行 0、隐藏回复被当作最佳答案 0、带赞的隐藏回复 0、可见公开话题缺动态行 0。

### 1.3 语义与疑似 bug

1. **服务端产出中文句子**：galgame 类条目的 `content` 被改写成「在《%s》发布了下载资源」「编辑了《%s》」「对《%s》提出了更新请求」；未知名作品叫 `galgame#123`；提及渲染成「@用户」。违反 01 §7。
2. **MESSAGE_UPVOTE 是 TOPIC_UPVOTE 的重复**：它来自 `message` 表里 `type = 'upvoted'` 的通知行（发件人是推的人，`content` 是话题标题）。895 行里 727 行与一条 TOPIC_UPVOTE 同话题、同一时刻（5 秒内）。时间线（不按类型筛）两条都出，于是同一次推在时间线上出现两遍；分类页的筛选菜单里两个类型的中文标签都是「话题被推」。
3. **两种类型的正文是服务端渲染的 HTML**：TOPIC_REPLY_CREATION 与 GALGAME_COMMENT_CREATION 在服务端过 `markdown.Render`，网页 `renderKatex` 后 `v-html`（`card/Reply.vue`、`card/GalgameComment.vue`）。K13 要的是结构化正文。
4. **只要话题的伪类型路径**：`/activity/tab` 在解析后只剩 TOPIC_CREATION 时改从 `topic` 表按**顶帖时间**出（`FetchTopicFeed`），其余情况都按动态行的创建时间出。同样的「只看话题」在分类页（`/activity?type=TOPIC_CREATION`）却按创建时间出——两条路由对同一个筛选给出不同的排序，且游标格式相同、互相串用不报错。
5. **话题的「普通 / 资源求助」拆分是隐式的**：时间线与分类页固定只要「非求助版块」的新话题（`sectionMode = "normal"`），首页页签用两个伪类型切换。
6. 页签路由的 `tab=` 分支（`GetTab`、`homeTabTypes`）零调用方。
7. `TopicUpvote.vue` 用 `unique_id.split(':')` 取种子，而 `unique_id` 的格式是 `TYPE-id`，永远取不到，随机文案的种子恒为话题 id。
8. `card/EntityComment.vue` 要求 `activity.data`，而 TOOLSET_COMMENT_CREATION / GALGAME_WEBSITE_COMMENT_CREATION 从来不带 `data`，这个分支是死的，两类都落到通用卡片。
9. 置顶回复（top reply）、最佳答案、被引用的回复这三处查询**不看回复状态**；生产上此刻都是 0 行，但只要有一条隐藏回复被赞过或被引用，它的摘录就会出现在动态里。
10. 游标坏了静默从头开始（`decodeCursor` 返回 nil）；`limit` 超过 50 回 `233`。
11. 旧错误一律 `233` + 中文句子。

## 2. 范围

| 旧 | v1 |
|---|---|
| `GET /api/activity`、`/api/activity/tab`、`/api/activity/timeline` | **一个**游标集合 `GET /api/v1/activities` |

删 3 条，`legacy_route_baseline` 下调 3（以合并时 rebase 后重新生成为准）。

## 3. 集合 `GET /api/v1/activities`

公开档（不看凭证）。游标集合（K11），`limit` 1–100、默认 20。

| 参数 | 类型 | 说明 |
|---|---|---|
| `kinds` | 封闭枚举数组（逗号形） | 只要这些类型。缺席 = 全部类型。未知 token → `400 UNKNOWN_ENUM_VALUE` |
| `topic_sections` | `normal` \| `help` \| `all`，默认 `normal` | `topic_creation` 条目取哪些版块的话题：`help` = 资源 / 求助三个版块（`g-seeking` `g-other` `t-help`），`normal` = 其余，`all` = 不限。只作用于 `topic_creation` |
| `sort` | `occurred_desc`（默认）\| `bumped_desc` | `bumped_desc` = 按话题顶帖时间，**只在 `kinds` 恰好是 `topic_creation` 时合法**，否则 `400 INVALID_PARAMETER`（`parameter: sort`，`INCONSISTENT_WITH`） |
| `include_nsfw` | bool，默认 `false` | 为 `true` 才出 NSFW 条目 |
| `include_galgames_without_resources` | bool，默认 `false` | `galgame_creation` 条目默认只出有下载资源的作品（旧 `show_no_resource`） |

- 排序 `occurred_desc`：`occurred_at DESC`，决胜键 `(类型, 源 id)` 降序——即现行索引 `idx_feed_activity_keyset`。`bumped_desc`：`topic.bumped_at DESC, topic.id DESC`。
- 游标绑定排序与全部过滤条件（`kinds` 规范化排序后、`topic_sections`、两个布尔）；换了条件再用旧游标 → `400 INVALID_CURSOR`，不静默从头开始。
- 条目在组装时可能被丢掉（作者已封禁、作品在 catalog 查不到或按 NSFW 立场不可见），所以一页可以少于 `limit`；服务端最多补取 5 轮。**`next_cursor` 缺席才表示到底了**，页短不代表到底（沿用现行语义，写进描述）。
- 结果在 Redis 缓存 30 秒（沿用），键含全部过滤条件与游标。

## 4. 表示层

### 4.1 `Activity`（`object: "activity"`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | DecimalID | 动态行 id（`feed_activity.id`） |
| `kind` | 封闭枚举 `activity_kind`（§4.2） | |
| `occurred_at` | DateTime | 事件时间（动态行的 `created`）。`bumped_desc` 下仍是话题的创建时间，排序键在 `topic.bumped_at` |
| `actor` | UserRef \| null | 做这件事的人。`galgame_creation` 取作品的创建者，没有就是 `null` |
| `path` | string，`^/` | 站内路径（与 `Notification.path` 同义），如 `/topic/4121`、`/galgame-quiz/281` |
| `excerpt_markdown` | string，≤ 1000 字 | 动态行里存的文字快照：话题 / 工具 / 网站 / 题目的标题或题干，回复、评论、推备注、待办、更新日志的正文。galgame 作品类条目为空串。**客户端按 `kind` 决定怎么显示，服务端不再改写成句子** |
| `topic` | `TopicSummary` \| null | 与 `listTopics` 同一个 schema。只在 `topic_creation` / `topic_upvote` |
| `topic_digest` | `TopicDigest` \| null | 同上两类（§4.3） |
| `reply` | `ActivityReply` \| null | `topic_reply_creation` / `best_answer_set`（§4.4） |
| `comment` | `ActivityComment` \| null | `topic_comment_creation` |
| `work` | `WorkRef` \| null（共享，`repr.WorkRef`，由 `galgameapiv1.WorkRefOf` 从 catalog 行映射） | 带作品的 galgame 类条目（动态行 `work_id > 0`） |
| `work_digest` | `{ developer_names, intro_excerpt, release }` \| null | `galgame_creation` / `galgame_edit` / `galgame_pr_creation`（§4.5） |
| `work_stats` | `{ resource_count, like_count, favorite_count }` \| null | `galgame_creation` |
| `work_revision` | `{ revision_id, revision_number }` \| null | `galgame_edit`，修订号未知时为 `null` |
| `rating` | `ActivityRating` \| null | `galgame_rating_creation`（§4.5） |
| `resource` | `ActivityResource` \| null | `galgame_resource_creation`（§4.5） |
| `quiz` | `ActivityQuiz` \| null | `galgame_quiz_creation`（§4.5） |
| `toolset` | `{ toolset_id, title }` \| null | `toolset_resource_creation`（所属工具） |
| `todo` | `{ todo_id, state }` \| null | `todo_creation`；`state` 与 `Todo.state` 同一词表 |
| `update_log` | `{ update_log_id, release_version }` \| null | `update_log_creation` |

同一条目只有与它的 `kind` 对应的块非空，其余块恒为 `null`——块与 `kind` 的对应表写进字段描述，测试钉住。

### 4.2 `activity_kind`（封闭，22 个）

旧类型名的小写，两处例外：

- `MESSAGE_SOLUTION` → **`best_answer_set`**（它是「作者设了最佳答案」的通知镜像，旧名说不清）；
- `MESSAGE_UPVOTE` **不建模**：它是 `topic_upvote` 的通知回声（§1.3 第 2 条），v1 的查询一律排除这一类（K-X2A1）。

`topic_creation` `topic_reply_creation` `topic_comment_creation` `topic_upvote` `best_answer_set` `galgame_creation` `galgame_edit` `galgame_pr_creation` `galgame_resource_creation` `galgame_resource_comment_creation` `galgame_comment_creation` `galgame_rating_creation` `galgame_rating_comment_creation` `galgame_quiz_creation` `galgame_quiz_comment_creation` `galgame_website_creation` `galgame_website_comment_creation` `toolset_creation` `toolset_resource_creation` `toolset_comment_creation` `todo_creation` `update_log_creation`

### 4.3 `TopicDigest`

动态卡片比列表卡片多的那部分。

| 字段 | 说明 |
|---|---|
| `excerpt_markdown` | 正文前 300 字（Markdown 源，含图片 token） |
| `favorite_count` | |
| `edited_at` | DateTime \| null |
| `top_reply` | `ReplyExcerpt` \| null：赞最多的**可见**回复（赞数 > 0） |
| `best_answer` | `ReplyExcerpt` \| null：最佳答案（可见时） |
| `latest_upvote` | `{ upvoter, note, upvoted_at }` \| null：最近一次推（网页只显示一条，旧接口发全部） |
| `latest_reply` / `latest_comment` | `ReplyExcerpt` / `CommentExcerpt` \| null：最新一条可见回复或评论，**至多一个非空**（旧的 `kind: reply\|comment` 字段——`kind` 是禁用名） |
| `reactions` | `[ReactionSummary]`，与话题详情同一个 schema；本集合是公开档，`viewer` 恒为 `null`，「我的表情」照旧由网页从 `/me/topic-states` 取 |

`ReplyExcerpt`：`reply_id` `floor` `author` `excerpt_markdown`（≤ 200） `like_count` `created_at`。`CommentExcerpt`：`comment_id` `author` `excerpt_markdown`（≤ 200） `created_at`。作者已封禁的摘录整条不出（`null`）。

### 4.4 `ActivityReply` / `ActivityComment`

- `ActivityReply`：`reply_id` `topic_id` `topic_title` `floor` `content`（**ContentDocument**，由动态行的 Markdown 经正文转换器产出；提及与楼层引用成为节点，不再是服务端拼的「@名字」）、`quoted`（`{ floor, excerpt_markdown }` \| null：回复里第一个楼层引用指向的**可见**回复）。`best_answer_set` 的 `reply_id` / `floor` 取话题当前的最佳答案，`content` 转换自通知行里存的摘录。
- `ActivityComment`：`comment_id` `topic_id` `topic_title` `quoted`（所评论的回复，`{ floor, excerpt_markdown }` \| null，回复不可见时为 `null`）。评论正文在 `excerpt_markdown`。

### 4.5 作品类的块

命名按 2026-09-23 协调会话的裁决：作品的 id 就是 catalog work id，v1 一律叫 `work` / `work_id`；嵌入的引用用共享的 `repr.WorkRef`（`object: "work"`、名字三件套 `display_name` / `latin` / `localized{}`、竖版原图 `cover`、`is_nsfw` 走展示轴），本轨不自己定义作品的表示。

- `work_digest`：`developer_names`（catalog 的品牌标签名，≤ 10 个）、`intro_excerpt`（首选简介的前 300 字，没有为 `null`）、`release`（发售日，**按记录精度**：`YYYY` / `YYYY-MM` / `YYYY-MM-DD`，未定为 `null`；不叫 `release_date` 是因为 F1 要求 `_date` 必须是完整日期）。
- `ActivityRating`：`rating_id` `overall`（1–10） `play_status`（`wish` `doing` `done_main` `done_one_route` `done_all` `dropped`） `recommend`（`strong_yes` `yes` `neutral` `no` `strong_no`） `spoiler_level`（`none` `portion` `serious`） `short_summary`（`spoiler_level ≠ none` 时恒为 `null`，沿用旧的剧透保护） `like_count`。生产取值见 §1.2 之外的核对：六个 / 五个 / 三个取值全部出现、没有别的值。
- `ActivityResource`：`resource_id` `resource_type`（`game` `collection` `image` `patch` `voice` `video` `ai` `others`，生产八种全在） `language` `platform`（开放词表，生产取值 `zh-cn` `zh-tw` `ja-jp` `en-us` `others` / `windows` `mac` `linux` `app` `emulator` `others`） `size`（字符串，如 `1.7GB`） `note`（≤ 300 字摘录，空为 `null`） `like_count`。
- `ActivityQuiz`：`quiz_id` `category` `question_type`（都是开放词表：题库还在加题型） `difficulty`（1–10） `answer_count` `correct_count` `favorite_count` `description_excerpt`（≤ 200 字）。题干在 `excerpt_markdown`。
- 作品在 catalog 查不到（或按 `include_nsfw` 不可见）的作品类条目整条不出，沿用旧行为。
- G 轨若先合入了评分 / 资源 / 题目自己的 schema，这几个块在 rebase 时换成那边的同名同型字段（G8 会逼着对齐）。

## 5. 逐条裁决

| 旧面 | 裁决 |
|---|---|
| 三条路由、两种排序、同一种游标 | 一个集合；排序显式（`sort`），游标绑排序与过滤 |
| 伪类型 `TOPIC_NORMAL` / `TOPIC_RESOURCE_HELP` | 显式参数 `topic_sections` |
| 服务端中文句子、`galgame#123` | 删；客户端按 `kind` + 结构化块拼 |
| 回复 / galgame 评论下发 HTML | 回复走 `reply.content` 文档；galgame 评论的正文在 `excerpt_markdown`，客户端按纯文本显示（K20：评论不是 Markdown） |
| MESSAGE_UPVOTE | 不建模，排除（K-X2A1） |
| 摘录不看回复状态 | 置顶回复、最佳答案、被引用回复、最新回复 / 评论一律只取 `status = 0` 且作者可渲染的 |
| `tab=` 内置页签 | 删（零调用方） |
| 坏游标静默从头来 | `400 INVALID_CURSOR` |
| `limit` 1–50，超限 `233` | 1–100，超限 `400 LIMIT_TOO_LARGE` |
| catalog / OAuth 失败时悄悄降级 | `503 SERVICE_UNAVAILABLE`（与其它 v1 读面一致） |

K 编号用前缀：

- **K-X2A1**：`message_upvote` 不进 v1 动态流——它与 `topic_upvote` 是同一件事的两条行。
- **K-X2A2**：`bumped_desc` 只对「只看话题」合法，其余组合拒绝而不是悄悄换排序。

## 6. 网页

- `home/ActivityFeed.vue`、`activity/Container.vue`、`activity/Timeline.vue` 换类型化客户端（`useApi` + 翻页用 `api.GET`）。
- 设置里存的旧 token 在发请求前翻译：大写转小写；`MESSAGE_SOLUTION` → `best_answer_set`；`MESSAGE_UPVOTE` 丢弃；`TOPIC_NORMAL` / `TOPIC_RESOURCE_HELP` → `topic_creation` + `topic_sections`（只选一个时相应取 `normal` / `help`，两个都选取 `all`）；解析结果恰好只有 `topic_creation` 时带 `sort=bumped_desc`（保住首页「话题」页签现在的顶帖序）。
- 首页「全部」页签固定 `include_nsfw=false`（旧 `force_sfw`），其余按用户的 NSFW 立场。
- `Card.vue` 按 `kind` 分派；21 个卡片组件改读新字段，文案由网页拼；回复卡用 `content/Document.vue` 渲染，不再 `v-html`。
- 分类页的筛选菜单去掉 MESSAGE_UPVOTE。
- `shared/types/activity.ts` 删除，改成生成物别名。
- `legacy-fetch-baseline` 下调 6。

## 7. 变异题（先于实现提交）

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 游标去掉 `(类型, 源 id)` 决胜键，只按时间 | 种子里多条动态共享同一 `created` 且跨页；小 `limit` 翻完与 SQL `ORDER BY created DESC, type DESC, source_id DESC` 逐条相等 |
| 2 | 不排除 `MESSAGE_UPVOTE` | 时间线里没有 `message_upvote` 行（种子里有） |
| 3 | `include_nsfw` 不生效 | 默认请求不含 NSFW 行；`include_nsfw=true` 才含 |
| 4 | `topic_sections` 不生效（恒 `all`） | 默认 `normal` 时求助版块的新话题不出现 |
| 5 | `bumped_desc` 不校验 `kinds` | `kinds=topic_creation,topic_reply_creation&sort=bumped_desc` → `400` |
| 6 | 游标不绑过滤条件 | 拿 `kinds=A` 的游标请求 `kinds=B` → `400 INVALID_CURSOR` |
| 7 | 作者已封禁的条目不丢 | 封禁用户的回复不出现在任何一页 |
| 8 | 置顶回复不看状态 | 隐藏回复即便赞最多也不当 `top_reply` |
| 9 | 块与 kind 不对应（给 `topic_reply_creation` 也填 `topic`） | 每条只有它 kind 对应的块非空 |
| 10 | 提及不转成节点（原样下发 token 文本） | `reply.content` 里提及是 `mention` 节点、带 `UserRef` |
| 11 | OAuth 失败时悄悄当匿名作者 | 用户服务 500 → `503` |

## 8. 实现时对本契约的修正（2026-09-23，只增不改）

上面 §1–§7 保持提交时的原文；实现时改了下面这些，以本节为准。

1. **`kind` → `activity_type`，参数 `kinds` → `activity_types`**：`kind` 在禁用名表里（契约门直接红）；照 `Notification.notification_type` 的先例改名。§3、§4、§7 里的 `kind` / `kinds` 一律读作新名。
2. **`actor` → `performer`**：`Notification.actor` 是非空 `UserRef`，本集合的同名字段可空（`galgame_creation` 没有创建者时），G8 不许同名不同型。
3. **`TopicDigest.best_answer` → `best_answer_excerpt`**：`Topic.best_answer` 已经是另一个形状（完整回复），G8 撞名。
4. **`UpvoteExcerpt.upvoted_at` 类型标成可空**：`TopicEngagement.upvoted_at` 可空，G8 要求同型；本集合里它恒有值（取自推的创建时间），网页只在有值时显示。
5. **`work_digest.developer_names` 的元素是 `DeveloperName`**（`maxLength` 256 + 自由文本标记），满足 G14；参数 `topic_sections` 补了 `maxLength`。
6. **不做 Redis 页缓存**（§3 最后一条作废）：`reply.content` 是 `ContentDocument`，节点是按 `object` 分派的联合类型，缓存的 JSON 读不回 Go 值。换成服务内的 catalog 行缓存（2 分钟、至多 20000 条）：一页里真正贵的是 catalog `/v2` 的批量取行，而它的限流按出口 IP 算，整站共用一个桶。
7. **复用而非另建**：`topic` 用 `listTopics` 的 `TopicSummary`（`topicapiv1` 新导出了 `MapSummary`），`reactions` 用话题详情的 `ReactionSummary`，作品用共享的 `repr.WorkRef`（`work` 可空指针，与其它 X2 分支一致；本轨不用 `works`）。
8. **`work_revision` 改形**：原文 `{ revision_id, revision_number }` 两个都必填。生产 773 条 `GALGAME_EDIT`：667 条有修订号（其中 461 条是编辑引擎写入、没有 wiki 修订 id），106 条只有退役 wiki 的修订 id，没有两者皆无的。实现的第一版要求两者都有，只给出了 206 条，网页那 567 条卡片会看不到差异——Gate 前自查发现、改掉。现在是 `{ revision_number: int ≥ 1 | null, legacy_revision_id: DecimalID | null }`，任一有值就出块；只有 `legacy_revision_id` 时网页照旧去编辑引擎的修订历史里按 `legacy_id` 找修订号（这两次取数是 GE 轨的旧路由，本轨不动）。`TestV1ActivitiesShape` 钉住三种行。
9. **变异 2 的读法**：`MESSAGE_UPVOTE` 在 v1 的类型词表里根本没有，排除是结构性的；变异改成「给它一个类型」，由 `TestV1ActivitiesNoUpvoteEcho` 杀。**变异 11**：正文转换器自己也调 OAuth，拿 `topic_reply_creation` 测会被转换器的 503 掩盖，测试改用不需要转换正文的 `topic_comment_creation`。
10. **网页**：`card/EntityComment.vue` 删除（§1.3 第 8 条的死分支，两类评论本来就落到通用卡）；作品封面改用 `WorkRef.cover` 的竖版原图（3:4），不再是 16:9 的 `_mini` 裁切；作品名走共享的 `utils/catalogName.ts` + `useWorkName()`（与 x2-ranking / x2-community 逐字节相同）；首屏与翻页走现成的 `useCursorList`（自带「后退恢复已加载的页」），不另写分页。
11. 数字（rebase 前）：`legacy_route_baseline` 159 → 156，`legacy-fetch-baseline` 189 → 183。
12. **catalog 行缓存的三条约束**（协调会话审 #209 时提出，都在本仓库出过事）：
    - 名字偏好：缓存的是 catalog 原始行（全部名字都在），名字按请求在 `WorkRefOf` / 客户端挑，所以缓存键不需要偏好开关。
    - 立场：键是 `(work_id, sfw)`，两种立场各向 catalog 以各自的 `content_limit` 取数，互不借用；`TestV1ActivitiesCacheKeepsStancesApart` 先以 NSFW 读者填缓存，再以 SFW 读者读，成人作品不得出现、SFW 作品不得消失。
    - 击穿：同一批未命中的 id（排序后连同立场作键）走 `singleflight`，首页冷启动或过期时并发的 N 个请求只花一次 catalog 调用；共享调用用 `context.WithoutCancel`，一个请求被取消不连累同批的其它请求。`TestCatalogRowsCoalescesConcurrentMisses` 钉住。

## 9. 验收记录（2026-09-23，rebase 到 master `9e7c496f` 之后）

- 门：`go build` / `make lint` 干净；全量库测试（本轨临时库，`-count=1 -p 1`）全绿；`make openapi`、路由 golden（`-update-routes`）、`pnpm gen:api` 均无漂移；网页 `lint` / `typecheck` / `test` 全绿；`deadcode` 在 `internal/activity` 下无条目。
- 基线（rebase 后重新生成）：`legacy_route_baseline` 153 → **150**；`legacy-fetch-baseline` 183 → **177**（X1a 先降了 6，本轨再降 6）。

### 9.1 变异（§7 的 11 条 + §8 第 8 条补的 1 条 + §8 第 12 条补的 2 条，14 条全杀）

| # | 改动 | 红的测试 |
|---|---|---|
| 1 | 去掉 `(type, source_id)` 决胜键 | `TestV1ActivitiesWalk` |
| 2 | 给 `MESSAGE_UPVOTE` 一个类型 | `TestV1ActivitiesNoUpvoteEcho` |
| 3 | `include_nsfw` 不生效 | `TestV1ActivitiesFilters` |
| 4 | `topic_sections` 恒 `all` | `TestV1ActivitiesWalk`、`TestV1ActivitiesFilters` |
| 5 | `bumped_desc` 不校验类型 | `TestV1ActivitiesRejects`（两个子测试） |
| 6 | 游标不绑过滤条件 | `TestV1ActivitiesRejects` |
| 7 | 封禁的执行者不丢 | `TestV1ActivitiesWalk`、`TestV1ActivitiesShape` |
| 8 | 置顶回复不看状态 | `TestV1ActivitiesShape` |
| 9 | 回复条目也填 `topic` | `TestV1ActivitiesShape` |
| 10 | 提及原样下发 token 文本 | `TestV1ActivitiesShape` |
| 11 | OAuth 失败当作没有用户 | `TestV1ActivitiesUpstream` |
| 12 | `work_revision` 要求两个 id 都在 | `TestV1ActivitiesShape` |
| 13 | catalog 行缓存键去掉立场（§8 第 12 条） | `TestV1ActivitiesCacheKeepsStancesApart` |
| 14 | 去掉 `singleflight`（§8 第 12 条） | `TestCatalogRowsCoalescesConcurrentMisses`（8 个并发未命中花了 8 次调用） |

### 9.2 浏览器（API :2372 打本轨临时库 + 种子数据，网页 :2371，真 OAuth / catalog）

开发库还停在 G0 之前（`feed_activity.work_id` 不存在，迁移 141 要先跑 `align-galgame-ids`），所以 API 改指本轨临时库，种了 46 个话题、12 个 catalog 真实作品（id 1–12）、36 个资源、评分 / 编辑 / 待办 / 更新日志各几条；验收后种子行全部删掉，开发库没动过。

- 首页六个动态页签（话题 / Galgame / 全站 / Gal 资源 / 资源和求助 / 其他）各自出正确的类型；「话题」按顶帖序；「全站」固定 `include_nsfw=false`；客户端切页签后滚到底自动续页 30 → 36 并显示「没有更多动态了」。
- 卡片逐类核对：话题卡的最佳答案、最新评论（与最佳答案不重复）、表情与表情人头像、「有解答」徽章；回复卡的楼层引用与 `@提及` 节点；推话题、采纳最佳答案、评论、评分（剧透的那条只显示提示）、编辑卡的开发商 / 发售日 / 简介；作品名走中文本地化名，封面是竖版原图。
- `/activity` 首屏 50 条，滚到底续到 100；进一条话题再后退，100 条原样恢复（`useCursorList` 的快照）。`/activity/category` 切「最佳答案 / Galgame 编辑 / Galgame 资源 / 待办」各出对应条目。
- 「加载中后退」：页面数据还没到时后退 → 白屏、进度条卡住。`/galgame → /topic` 同样复现，与本轨无关（已知问题，未修）。
- 控制台：只有 umami 400 与登录态头像的水合不一致（两者都与本轨无关）；种子里编辑卡的修订号是编的，所以 GE 轨旧路由 `/api/galgame/3/edit/diff` 回 404、弹了一次「条目或提案不存在」。真实数据下这个修订号来自编辑引擎。
