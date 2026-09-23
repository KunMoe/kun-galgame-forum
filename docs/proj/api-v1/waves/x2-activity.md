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
