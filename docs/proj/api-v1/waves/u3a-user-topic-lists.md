# U3a · 某用户的话题、回复、评论

> 用户轨第三段的前半。旧路由 **3 条**：`GET /user/:id/topics`、`/replies`、`/comments`，都是 `UserHandler` 的方法，纯本地 SQL。迁移号段 120–129，**本段没有迁移**。
> U3b（galgames / galgame-comments / resources / ratings，依赖 catalog 与 community）另起一段。`/user/:id/toolsets`（`ToolsetHandler`）与 `/user/:id/collections`（`GalgameCollectionHandler`）按看板归 **G 轨**（2026-09-23 与 G 确认），U 轨共 23 条（U1 12 + U2 4 + U3a 3 + U3b 4）；G 会沿用本段定下的子集合形状（§2、§3）。
> 普查在 [census/user.md](census/user.md) §2.1–§2.4、§10；本文 §1 补网页与生产事实。契约先于实现提交，变异清单见 §7。
> 铁律 3（`gid ≡ work_id`）：本段不涉及 galgame id。

## 1. 普查补充

| 事实 | 证据 |
|---|---|
| 调用方 | `components/user/Topic.vue`、`Reply.vue`、`Comment.vue`，另 `components/user/CollectionTopic.vue`（收藏 tab 用 `topics?type=topic_favorite`）；全部是**页码分页**（`KunPagination` + URL 里的 `page`），`limit` 50；`../kungal-apps` 零命中 |
| 网页可见的类型 | 话题 tab：`topic` / `topic_like` / `topic_upvote`，`topic_hide` 只给本人或持 `topic.view_hidden` 者（`constants/user.ts:83`）；`topic_favorite` 在收藏 tab；回复：`reply_created` / `reply_target` / `reply_like`；评论：`comment_created` / `comment_target` / `comment_like` |
| 正文怎么用 | 回复 tab 显示 `markdownToText(reply.content)`——**整段 Markdown 源文下发、网页转纯文本**；评论 tab 直接显示纯文本 |
| **泄漏（普查 bug #1，严重）** | `type=topic|topic_like|topic_upvote|topic_favorite` 不排除 `status = 1`：匿名访客能读到任何人被隐藏话题的标题。生产被隐藏话题 322 条 / 213 人 |
| 泄漏（普查 §2.2 第二条） | 持 `topic.view_restricted` 者在**别人的**资料页看到受限话题——`access_repo.go:34`「role/user 授权永不让话题出现在共享列表里」被破 |
| 泄漏（普查 §2.3/§2.4） | 回复与评论列表**完全不查所属话题的可见性**：被隐藏或受限话题里的回复正文对匿名可读 |
| 分页 | 6 处 `ORDER BY created DESC` 无 `id` 决胜键（普查 bug #7） |
| `type` | 开放裸字符串，未知值静默回落成「本人发的」（01 §4 禁止） |
| 偏好输入 | 用 `KUNGalgameSettings` cookie / `X-Kungal-Nsfw` 头决定 SFW（01 §3 禁止）；话题轨的 `/topics` 用显式 `include_nsfw` |
| `comment_target` | `topic_comment.target_user_id` 90 天内仍有 519 条新写入（生产 2026-09-22），**不是死路**，保留 |

## 2. 逐条去向

| 旧路由 | v1 | 档 |
|---|---|---|
| `GET /user/:id/topics?type=` | `GET /api/v1/users/{user_id}/topics?relation=` | optional |
| `GET /user/:id/replies?type=` | `GET /api/v1/users/{user_id}/replies?relation=` | optional |
| `GET /user/:id/comments?type=` | `GET /api/v1/users/{user_id}/comments?relation=` | optional |

`legacy_route_baseline` 下调 **3**。子资源而不是 `/topics?author_id=`：`relation` 里有「赞过的」「收到的」，不是话题集合上的作者过滤；而且不去动话题轨的列表引擎。

## 3. 共同规则（三条都适用）

- **页码集合**（K11 browse，网页要分页器）：`collect.PageNumber`（`page` ≥1 默认 1，`limit` 1–100 **默认 50**）、深度上限、`repr.PageList[T]`（`total` + `total_relation`）。`total` 与 `items` **同一谓词**。
- **`relation`**：封闭枚举，必填。未知值 → `400 UNKNOWN_ENUM_VALUE`（schema 枚举产出；不再静默回落）。
- **`include_nsfw`**：`true`/`false`，默认 `false`，与 `/topics` 同一语义（K 标准：v1 不读偏好 cookie）。`false` 时排除 NSFW 话题及其下的回复、评论。
- **排序**：`created_at DESC, id DESC`（话题按话题的时间与 id；回复按回复的；评论按评论的——包括 `liked` / `received` 关系，与旧实现同一排序对象）。**决胜键必须有**。
- **资料主人不可渲染**（不存在、`status != 0`）→ `404 NOT_FOUND`（与 U2 `GET /users/{user_id}` 同一口径，K24）；`userclient` 失败 → `503`。旧实现的 `hideTarget` 在 OAuth 出错时放行（fail-open，普查 bug #8）。
- **话题可见性一律按共享列表**：任何关系下，列出的话题、以及列出的回复 / 评论**所属的话题**，都必须满足 `topic.status != 1` 与 `SharedListPredicate(authenticated)`（`internal/topic/repository/access_repo.go`）。持 `topic.view_restricted` **不**放宽（共享列表不变量）。唯一例外是 §4.1 的 `hidden`。
- 被列出的回复、评论本身须 `status = 0`（旧实现如此）。
- `id` / `topic_id` 是十进制字符串；时间是 RFC 3339 UTC 秒精度。

## 4. 三个集合

### 4.1 `GET /users/{user_id}/topics`

`relation`：

| v1 | 旧 `type` | 语义 |
|---|---|---|
| `authored` | `topic` | 他发的话题 |
| `liked` | `topic_like` | 他赞过的话题（`topic_reaction.reaction = 'like'`） |
| `upvoted` | `topic_upvote` | 他推过的话题 |
| `favorited` | `topic_favorite` | 他收藏的话题 |
| `hidden` | `topic_hide` | 他发的、被隐藏的话题（`status = 1`）。**仅本人或 `user.Can(perm.TopicViewHidden)`**；否则 `403 PERMISSION_REQUIRED`（匿名同样 403——这个集合对他们不存在）。本关系**不**套共享列表谓词（本人本来就能读自己的话题；管理者有权限） |

条目 `UserTopicItem`：`object:"topic"`、`id`、`title`、`created_at`。（与 `TopicSummary` 同为 `object:"topic"`；重叠字段同名同型，G8。）

### 4.2 `GET /users/{user_id}/replies`

`relation`：`authored`（旧 `reply_created`）、`received`（旧 `reply_target`：他发的话题下别人的回复）、`liked`（旧 `reply_like`）。

条目 `UserReplyItem`：`object:"reply"`、`id`、`topic_id`、`floor`（integer ≥1）、`excerpt`、`created_at`。

- **`excerpt`**：回复正文转成的**纯文本**，最多 200 个字符（rune），超出截断并以 `…` 结尾；`maxLength: 200`——JSON Schema 的 `maxLength` 数的是字符不是字节（实现初版按「200 × 4 字节」声明成 800，合并前改正）。旧实现下发整段 Markdown 源文、网页再转纯文本；v1 在服务端转，列表不再搬整段正文。
- `received` 排除他自己在自己话题下的回复？**不排除**——旧实现不排除，本段不改语义。

### 4.3 `GET /users/{user_id}/comments`

`relation`：`authored`（旧 `comment_created`）、`received`（旧 `comment_target`：`topic_comment.target_user_id = 他`）、`liked`（旧 `comment_like`）。

条目 `UserCommentItem`：`object:"comment"`、`id`、`topic_id`、`excerpt`（评论是纯文本，K20；同样 200 字符截断）、`created_at`。

## 5. 错误码

**不新增**。`NOT_FOUND`、`PERMISSION_REQUIRED`、`UNKNOWN_ENUM_VALUE`、`INVALID_PARAMETER`（深度上限 / 页码越界）、`LIMIT_TOO_LARGE`、`SERVICE_UNAVAILABLE`、`INTERNAL_ERROR`，以及 optional 档的 `INVALID_CREDENTIAL`（Bearer 无效）。

## 6. 网页

| 文件 | 改什么 |
|---|---|
| `components/user/Topic.vue`、`Reply.vue`、`Comment.vue`、`CollectionTopic.vue` | → 类型化客户端（SSR 用 `useApi`），`type` → `relation`（映射见 §4），`data.topics/replies/comments` → `items`，`created` → `created_at`，`reply.content` → `excerpt`（不再 `markdownToText`），`include_nsfw` 取网页当前的 NSFW 立场（照 `/topics` 调用点的写法），分页用 `total` |
| `constants/user.ts` | tab 的路由段（`topic-like` 等 URL）**不变**；只在调用处把旧 `type` 映射成 `relation` |
| `shared/types/user.ts` | 删掉 `UserTopic` / `UserReply` / `UserComment` 等手写类型（无人再用时） |

`CHANGELOG.md` 记一条。

## 7. 变异清单（先于实现提交）

| # | 破坏 | 应当变红的性质 |
|---|---|---|
| 1 | `authored` 话题去掉 `status != 1` | 被隐藏的话题不出现在任何非 `hidden` 关系里 |
| 2 | `liked` 话题去掉共享列表谓词 | 受限（`users` / `role`）话题不出现 |
| 3 | 持 `topic.view_restricted` 时跳过共享列表谓词（旧行为） | 版主看别人的资料页也看不到受限话题 |
| 4 | 回复列表不 join 所属话题的可见性 | 被隐藏话题里的回复不出现 |
| 5 | 评论列表不查所属话题的可见性 | 受限话题里的评论不出现 |
| 6 | `hidden` 的权限判断去掉 `perm.TopicViewHidden` 分支 | 持权限者能看别人的 `hidden` |
| 7 | `hidden` 对陌生人放行 | 陌生人 → 403 |
| 8 | 排序去掉 `id` 决胜键 | 同一秒的行跨页全量遍历无重无漏（种子必须有并列的 `created`） |
| 9 | `total` 用不带可见性过滤的计数 | `total` 等于实际可翻到的行数 |
| 10 | 资料主人不可渲染时仍返回列表 | 封禁用户 → 404 |
| 11 | `include_nsfw` 默认当 `true` | 默认不含 NSFW |
| 12 | `excerpt` 不截断 | 长回复的 `excerpt` 不超过 200 字符并以 `…` 结尾 |

## 8. 删旧路由

3 条全删，连同 `UserHandler.GetUserTopics` / `GetUserReplies` / `GetUserComments`、`content_repo.go` 里只被它们用到的查询与 DTO。`UserHandler` 之后只剩 U3b 的 4 个方法。`routes.golden` 重生成，`legacy_route_baseline` 下调 3。

## 9. 迁移

**无**。
