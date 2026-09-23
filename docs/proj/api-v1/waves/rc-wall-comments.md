# RC · 评论墙（资源评论 + galgame 评论）

> RC 轨，分支 `api-v1/rc-resource-comment`，迁移号段 135–139（本轨只用 135）。
> 契约先于实现提交；变异清单见 §8，**实现之前**已在分支上。

## 0. 范围：为什么是 22 条不是 15 条

看板原来把 `ResourceCommentHandler` 的 15 条划给 RC，把 `CommunityCommentHandler` 的 7 条留在 G。普查发现两者是**一个连通分量**：

- 六面评论墙（galgame / 评分 / 网站 / 工具 / 资源 / 题目）在网页上共用同一套组件：`comment/community/{Row,Like,FlagModal,Composer}.vue`、`useCommunityCommentList.ts`、`utils/communityComment.ts`。
- 五面资源墙的**编辑、点赞、举报**本来就走 G 的 `PUT /galgame/comments/:postId`、`…/like`、`…/flag`；`ResourceCommentHandler` 只管列表、发表、删除。
- 帖子 id 是 infra 社区原语的全局 id，逐帖操作与锚点无关。

2026-09-22 用户拍板：**RC 吞并 `CommunityCommentHandler`**。RC = 22 条，G = 85 条。G 会话已确认不碰这两个 handler、`internal/galgame/service/{community_comment_*,resource_comment_*,feed_parity}.go` 与上述网页组件。

**不在本轨**：`/community/wall/read`、`/community/wall/follow`、`/community/following`（关注与已读回执，X 轨，`/message/follow` 页也在用）；`GET /user/:id/…` 下的「某用户的评论」列表（U 轨）；详情页里嵌的「最新 N 条评论」（各自的域）。

## 1. 普查（2026-09-22，生产 `kun_community` + `kungalgame`）

### 1.1 旧路由

| # | 旧路由 | handler | 调用方 |
|---|---|---|---|
| 1–5 | `GET /{galgame-rating,website,toolset,galgame-resource,galgame-quiz}/:id/comments` | `ResourceCommentHandler.*List` | `useCommunityCommentList.ts`（经 `communityCommentSurface`） |
| 6–10 | `POST` 同上 | `*Create` | `Composer.vue` |
| 11–15 | `DELETE …/comments/:postId` | `*Delete` | `Row.vue` |
| 16 | `GET /galgame/:gid/comments` | `CommunityCommentHandler.List` | `useCommunityCommentList.ts` |
| 17 | `GET /galgame/:gid/comments/locate` | `.Locate` | `galgame/comment/CommunityContainer.vue`（旧链接 `?comment=<旧id>&thread=`） |
| 18 | `POST /galgame/:gid/comments` | `.Create` | `Composer.vue` |
| 19 | `PUT /galgame/comments/:postId` | `.Update` | `Row.vue`，**六面共用** |
| 20 | `DELETE /galgame/comments/:postId` | `.Delete` | `Row.vue`（galgame 面） |
| 21 | `PUT /galgame/comments/:postId/like` | `.ToggleLike` | `Like.vue`，**六面共用** |
| 22 | `POST /galgame/comments/:postId/flag` | `.Flag` | `FlagModal.vue`，**六面共用** |

Nitro `server/`：零调用。Flutter App（`../kungal-apps`）：零调用。

### 1.2 生产取值

| 量 | galgame | rating | resource | quiz | toolset | website |
|---|---|---|---|---|---|---|
| 非空墙 / 空墙 | 2861 / 67619 | 153 / 3782 | 124 / 38947 | 33 / 221 | 27 / 91 | 38 / 49 |
| 帖子 visible / held / deleted | 10735 / 13 / 105 | 229 / 3 / 2 | 179 / 1 / 20 | 53 / 1 / 2 | 74 / 0 / 0 | 119 / 0 / 1 |
| 顶层 / 回复 | 10392 / 461 | 234 / **0** | 135 / 65 | 48 / 8 | 45 / 29 | 92 / 28 |
| 带 `target_user_id` | 220 | **234（全部）** | 65 | 8 | 29 | 28 |
| 最长 / p99 | 1775 / 184 | 709 / 586 | 216 / 124 | 191 | 944 | 243 / 200 |
| 单墙最多帖 | 119 | 8 | 7 | 9 | 16 | 18 |
| 赞 | 1352 | 11 | 7 | 17 | 3 | 11 |
| 锚点行已不存在 | 5 | 0 | 6 | 0 | 1 | 0 |

- 近三月：galgame 551/553/330，resource 12/101/87，rating 20/35/28，quiz 7/40/9。**全部是活跃域。**
- Markdown 真实在用：galgame 的 md 链接 157、`/image/` token 56、md 图片 81、`**` 38、提及 77、裸 HTML 23、换行 2084；其余五面也都有 md 链接与图片。**评论墙是完整 Markdown，不是 K20 那种受限文档。**
- 编辑过的帖子 67 条，版主编辑 0 条；`content_rating ≠ 0` 的 0 条。
- 自赞 0。本地赞镜像 `galgame_post_like` 1401 行 = 上游 `community_reaction` 1401 行，两边一致。
- 墙的状态：kungal 只有 4 堵 galgame 墙是 `3`（上游 `ThreadStatusDeleted`）；`closed`（1）与 `hidden`（2）是 0 行。
- 回复里带 target 的数量恰好等于回复数：**上游按 `reply_to_post_id` 自己推导 target**，论坛不传。唯一由论坛传 target 的是评分墙的顶层帖（旧网页传评分作者）。
- 旧链接：`message.link` 里带 `thread=` 的 64 条，**全部是 `mentioned`**；其中 36 条能经 `galgame_comment_community_map` 映射到 post id，28 条映射不到（旧评论在迁移前就没了）。
- `comment_count` 漂移（本地计数 vs 上游 visible 帖数）：toolset 22/27、rating 76/153（**旧写面从不维护 rating 的计数**，且没有任何读面读它）、galgame 12/2861、resource 1、quiz 1、website 0。
- 锚点父行的状态：resource `status=1` 是「已失效」（`MarkExpired`），不是隐藏；toolset 的 `status` 全表 119 行都是 0，没有任何代码写 1；website `closed` 3 堵墙，只表示站点关了。

### 1.3 疑似 bug（照实记，裁决见 §2）

| # | 现象 | 证据 |
|---|---|---|
| B1 | **删帖权限不绑定帖子所在的墙**。`DELETE /galgame/comments/:id` 用 `comment.galgame.delete` 当版主权限、却不查这条帖子是不是 galgame 墙的；`DELETE /galgame-quiz/1/comments/:id` 同理，持 `comment.quiz.delete` 就能删**任何**墙上的帖，还会把 quiz 1 的计数减一 | `community_comment_handler.go` `Delete`、`resource_comment_write.go` `DeleteComment` |
| B2 | 资源主人删帖的归属校验是**翻遍整堵墙**找这条帖子（`postInResourceThread` 逐页拉上游） | `resource_comment_write.go` |
| B3 | 评分墙的「主人」是 **galgame 的创建者**，不是评分作者 | `resourceOwner` → `galgameOwner` |
| B4 | 网站墙的路径段 `:domain` 被忽略，id 实际取自查询串 / 请求体 `website_id` | `websiteID(c)` |
| B5 | 发表时不查父资源是否可读：网站墙对不存在的 id 照样建墙；galgame 墙对任何数字建墙并**插入本地 `galgame` 行**（孤儿来源之一）；评分 / 资源 / 工具只查存在、不查作者是否被封禁，而它们的详情页会因作者被封而 404 | `resolveCreateCtx` |
| B6 | 编辑一律上限 5000，发表时评分 1314、其余四面 1007：发表不了的长度可以编辑进去 | `Update` vs `*Create` |
| B7 | 点赞是**切换**：重放一次就撤销（违反 PUT 幂等） | `ToggleLike` |
| B8 | 上游不可用时读面**静默返回空墙**，与「真的没人评论」不可区分 | `isCommunityDown` |
| B9 | 上游 4xx 一律映射成 HTTP 400 + 上游的业务码 | `mapCommunityError` |
| B10 | 题目闸在数据库出错时判为「锁住」（失败关闭，这一条是对的），但读 `galgame_quiz_answer` 出错时判为「没答过」 | `commentAreaLocked` / `hasAnsweredQuiz` |
| B11 | 自赞：网页拦、服务端不拦（上游允许，只是不发分） | `Like.vue` vs `likeEffects` |
| B12 | 被封禁作者的帖子在列表里直接跳过；它的回复失去根，网页把它们当顶层画 | `renderPosts` |

## 2. 裁决

| # | 裁决 |
|---|---|
| D1 · 集合形状 | **一个扁平集合 `wall-comments`，按「墙的主体」过滤**，不做 `/galgames/{galgame_id}/comments`。原因：G17 要求写路径里的每个 `{x_id}` 都有 GET，而六种父资源在 v1 里**一个都还没有** GET；等它们出现再嵌套会是一次破坏性改路径。扁平集合对 App 也更好：一套代码服务六面墙 |
| D2 · 主体 | `subject_type`（封闭枚举）+ `subject_id`（十进制字符串）。取值：`galgame` `galgame_rating` `galgame_resource` `galgame_quiz` `toolset` `website`。上游锚点（`anchor_kind` 整数 + `rating:12` 这种字符串）**不出现在 v1 里** |
| D3 · 名字 | 资源叫 `wall_comment`，路径 `/wall-comments/{wall_comment_id}`。`comment` 与 `/comments/{comment_id}` 已属话题评论（另一张表、另一个 id 空间）；「wall」是网页代码里本来就在用的词（`/community/wall/read`、`CommunityWallState`） |
| D4 · 正文 | 完整 Markdown，走 W1 的内容文档管线：读面发 `content: ContentDocument`，源文只在 `GET …/source` 发 `content_markdown`。**不适用 K20**（§1.2：六面都在用链接、图片、加粗） |
| D5 · 父资源可读闸（修 B5） | 墙可读 ⇔ 主体可读，读写同一闸：`galgame` 经 `CatalogWorkIDForGID` 能解析（有缓存）；`galgame_rating` 行存在且评分作者可渲染；`galgame_resource` 行存在且发布者可渲染；`galgame_quiz` 行存在且出题人可渲染，再过题目闸；`toolset` 行存在且主人可渲染；`website` 行存在。不满足 → `404 NOT_FOUND`。catalog / OAuth 不可用 → `503`（K17 同理，读面也失败关闭：墙属于主体，主体都判不了就不该回一堵空墙） |
| D6 · 题目闸 | 题目隐藏了作品名或有剧透（`hide_galgame` 或 `spoiler_level ≠ none`）时，只有出题人与答过题的人能读写这堵墙。其余人（含匿名）→ `403 QUIZ_ANSWER_REQUIRED`（新码）。取代旧的 `locked: true` 空页。数据库出错一律 `500`，不再猜（修 B10） |
| D7 · 删帖权限（修 B1、B2、B3） | 能删 ⇔ 作者本人；或持**这条帖子所在的墙**对应的删帖权限（`comment.{galgame,rating,website,toolset,resource,quiz}.delete`）；或是这堵墙的主人（资源发布者 / 工具主人 / 出题人 / **评分作者**）。帖子所在的墙由上游 `ResolvePosts` 的线程上下文判定，不由请求里的任何东西判定。galgame 墙与网站墙没有主人 |
| D8 · 编辑权限 | 作者本人；或持这堵墙的编辑权限（`comment.*.edit`，版主编辑，上游记 `edited_by_moderator`）。与旧 `resolveModEdit` 相同，只是不再有「查不到就按任一编辑权限放行」的兜底 |
| D9 · 长度（修 B6） | 每面自己的上限，**发表与编辑相同**：galgame 5000，galgame_rating 1314，其余四面 1007。schema 上的 `maxLength` 是 5000（一个请求体服务六面），面的上限在 handler 里判，超了是 `422 VALIDATION_FAILED` + `TOO_LONG` + `max_length`。K19 照旧：先判原始值长度，再去首尾空白 |
| D10 · 回复与 `addressee` | 六面都收 `parent_comment_id`（必须是同一堵墙上未删除的帖，否则 `422` + `UNKNOWN_REFERENCE` 指向它）。`addressee`（这条帖是对谁说的）读面原样下发上游存储的 target。**不叫 `in_reply_to_user`**：话题评论的同名字段不可空，而墙上大多数顶层帖没有 target，同名异型会被 G8 拦下。写入时**不由客户端给**：有父帖由上游推导；**评分墙的顶层帖**由服务端填评分作者（保留旧网页「→ 评分作者」的显示，234 条历史数据全是这样）。评分墙从此也允许嵌套回复——旧数据 0 条回复，渲染不变 |
| D11 · 提及 | 与现状一致：只有 galgame 墙把 `@` 提及交给上游发通知、编辑时补发新增提及。其余五面的 `@` 照样解析成 `mention` 节点、但不通知。**是否推广到六面是产品决定，不在本轨** |
| D12 · 点赞（修 B7、B11） | K16 槽位：`PUT` 置位、`DELETE` 撤销，都幂等、都回 200 + `WallComment`。上游只有切换，所以以本地镜像 `galgame_post_like` 判当前态；切换结果与意图相反（镜像陈旧）时再切一次并修镜像。自赞 `403 SELF_LIKE_FORBIDDEN`。萌萌点照旧：他人点赞 +1，撤销 −1 |
| D13 · 举报 | `POST /wall-comments/{id}/flags` → `204`。举报不可回读（没有自己的 id），所以不是 201。`reason` 封闭枚举 `spam` `abuse` `off_topic` `other` `nsfw_mislabel`（上游 0–4），`note` ≤ 500。不能举报自己的帖（`403 PERMISSION_REQUIRED`） |
| D14 · 删除后的帖 | 列表里保留墓碑（`state: deleted`，`content` 是**空文档**——不是 `null`：`content` 在话题、回复、评论上都不可空，G8 要求同名同型），回复不失根。`GET` 单条同样回墓碑。编辑、点赞、举报墓碑 → `409 INVALID_STATE_TRANSITION`。**删除墓碑 → `204`，无副作用**（不再减计数，不再删 feed）|
| D15 · 待审帖 | 上游 `status=1`（上游叫 hidden，论坛界面叫「审核中」）：只有作者看得见，`state: held`。发表时上游直接判成 held 的，`201` 照回，`state: held` |
| D16 · 封禁作者（B12） | 与话题一致：被封禁作者的帖在列表里略去、`GET` 单条 `404`。回复失根照旧由客户端当顶层画。不改 |
| D17 · 上游错误（修 B8、B9） | 上游不可用 → `503 SERVICE_UNAVAILABLE`，**读面也是**（不再回空墙）。上游 429（新人限流）→ `429 RATE_LIMITED`（新增，复用 infra platform 码）。上游不发 `Retry-After`（infra `handler/errors.go` 的沙箱分支只有状态码与原因），所以论坛也不发。上游 422 "content blocked by word list" → `422 CONTENT_REJECTED`。上游 409 "thread is not open"（墙被关，生产 0 行）→ `409 INVALID_STATE_TRANSITION`。上游 404 → `404 NOT_FOUND`。其余上游 4xx 是我们的 bug → `500 INTERNAL_ERROR` 并记日志 |
| D18 · 没有 `total` | 列表不发 `total`：上游的 `posts_count` 含删除与待审帖，与 `items` 不是同一谓词（K11）。需要数字的页面用父资源自己的 `comment_count`（资源、题目详情已经在发） |
| D19 · `locate` 不重建 | 旧 `GET /galgame/:gid/comments/locate` 只服务 64 条 `?comment=<旧id>&thread=` 的提及通知链接。迁移 135 把能映射的 36 条改写成 `?comment=<post_id>`，映射不到的 28 条去掉查询串只留页面链接。站外流传的旧链接从此只会打开页面、不再滚到那一楼，接受 |
| D20 · 关注与已读 | 仍走 X 轨的 `/community/wall/*`。网页在本地从主体推出旧锚点（这段映射只为调旧路由而存在，X 轨迁完即删）。v1 不发 `thread_id`；`WallRead` 在 `thread_id` 缺席时自己向上游解析——这是对 X 轨旧服务的一处**只加不改**的回落，PR 里点名 |
| D21 · 副作用照旧 | 发表：galgame 墙补本地行并 +1 计数；网站 / 工具 / 资源 / 题目 +1 计数；工具 / 资源 / 题目通知主人（去重 + 已关注者不重复通知）；首页 feed 同步。删除：计数 −1，feed 删除。评分墙的 `comment_count` 继续不维护（没有读者） |

## 3. 操作（9 个）

| operationId | 方法 路径 | 档 | 成功 |
|---|---|---|---|
| `listWallComments` | `GET /wall-comments?subject_type=&subject_id=&cursor=&limit=` | optional | 200 `List<WallComment>` |
| `createWallComment` | `POST /wall-comments` | required，**必带 `Idempotency-Key`** | 201 + `Location` + `WallComment` |
| `getWallComment` | `GET /wall-comments/{wall_comment_id}` | optional | 200 `WallComment` |
| `getWallCommentSource` | `GET /wall-comments/{wall_comment_id}/source` | required | 200 `WallCommentSource` |
| `updateWallComment` | `PATCH /wall-comments/{wall_comment_id}` | required | 200 `WallComment` |
| `deleteWallComment` | `DELETE /wall-comments/{wall_comment_id}` | required | 204 |
| `likeWallComment` | `PUT /wall-comments/{wall_comment_id}/like` | required | 200 `WallComment` |
| `unlikeWallComment` | `DELETE /wall-comments/{wall_comment_id}/like` | required | 200 `WallComment` |
| `flagWallComment` | `POST /wall-comments/{wall_comment_id}/flags` | required，`Idempotency-Key` 可选 | 204 |

### 3.1 列表

- 游标集合，`limit` 1–100、默认 20。顺序是墙内楼号升序（上游 `post_number`，每墙唯一）。
- 游标包的是**最后扫过的上游楼号**，绑定 `subject_type` + `subject_id`；换了主体再用 → `400 INVALID_CURSOR`。
- 被封禁作者的帖、别人的待审帖在服务端略去；服务端**不**为补满一页继续读，所以一页可以少于 `limit`。有 `next_cursor` 就继续翻。
- `subject_type` / `subject_id` 缺席 → `400 INVALID_PARAMETER`（`REQUIRED`，`parameter`）；`subject_type` 不在词表 → `400 UNKNOWN_ENUM_VALUE`；`subject_id` 不是正整数 → `400 INVALID_PARAMETER`（`INVALID_FORMAT`）。

### 3.2 `WallComment`

```json
{
  "object": "wall_comment",
  "id": "10523",
  "subject_type": "galgame_resource",
  "subject_id": "812",
  "parent_comment_id": "10500",
  "root_comment_id": "10500",
  "author": { "object": "user", "id": "3", "name": "…", "avatar": null },
  "addressee": { "object": "user", "id": "7", "name": "…", "avatar": null },
  "state": "visible",
  "content": { "…": "ContentDocument，state 为 deleted 时是空文档" },
  "like_count": 2,
  "created_at": "2026-09-22T10:00:00Z",
  "edited_at": null,
  "is_edited_by_moderator": false,
  "viewer": { "has_liked": false, "can_edit": false, "can_delete": false, "can_like": true, "can_flag": true }
}
```

- `state`：封闭枚举 `visible` / `held` / `deleted`。
- `parent_comment_id` / `root_comment_id`：顶层帖都是 `null`。
- `addressee`：`null` 或 `UserRef`；指向的人被封禁时照样发 `UserRef`，`name` 为 `null`（与 `author` 相同的规则）。
- `viewer`：匿名为 `null`。`can_like` = 已登录、非作者、未删除；`can_flag` 同；`can_edit` / `can_delete` 见 D8 / D7，墓碑上恒为 `false`。Bearer 请求的 `can_*` 不含任何 staff 能力（K2）。

### 3.3 请求体

- 发表：`{ "subject_type": "…", "subject_id": "…", "content_markdown": "…", "parent_comment_id": "…" | 省略 }`。`content_markdown` schema `minLength 1`、`maxLength 5000`，面的上限见 D9；全是空白 → `422` + `TOO_SHORT`。
- 编辑：`{ "content_markdown": "…" }`，同样的长度规则。正文与现存源文相同 → 直接回当前 `WallComment`，不打上游。
- 举报：`{ "reason": "spam", "note": "…" }`，`note` 可省略，≤ 500。

### 3.4 `WallCommentSource`

`{ "object": "wall_comment_source", "wall_comment_id": "…", "content_markdown": "…" }`。要求 `viewer.can_edit`，否则 `403 PERMISSION_REQUIRED`。

### 3.5 错误码全表

| 状态 | code | 哪些操作 |
|---|---|---|
| 400 | `INVALID_PARAMETER` `UNKNOWN_ENUM_VALUE` `LIMIT_TOO_LARGE` `INVALID_CURSOR` | list（`INVALID_PARAMETER` 也用于缺 `Idempotency-Key`） |
| 401 | `MISSING_CREDENTIAL` `INVALID_CREDENTIAL` | 全部 required；optional 档的坏 Bearer |
| 403 | `ACCOUNT_BANNED` | 全部 |
| 403 | `QUIZ_ANSWER_REQUIRED` | list、create、get（题目墙） |
| 403 | `PERMISSION_REQUIRED` | source、update、delete、flag（自己的帖） |
| 403 | `SELF_LIKE_FORBIDDEN` | like |
| 404 | `NOT_FOUND` | 全部（主体不可读 / 帖不存在 / 帖在本站六面之外 / 作者被封 / 别人的待审帖） |
| 409 | `INVALID_STATE_TRANSITION` | update、like、unlike、flag（墓碑）；create（墙被关） |
| 409 | `IDEMPOTENCY_KEY_REUSED` `IDEMPOTENCY_REQUEST_IN_PROGRESS` | create、flag |
| 422 | `VALIDATION_FAILED`（`TOO_LONG` `TOO_SHORT` `UNKNOWN_REFERENCE` `UNKNOWN_VALUE`） | create、update、flag |
| 422 | `CONTENT_REJECTED` | create、update |
| 429 | `RATE_LIMITED` | create、update |
| 500 | `INTERNAL_ERROR` | 全部 |
| 503 | `SERVICE_UNAVAILABLE` | 全部 |

## 4. 预分配

- **迁移 135**：改写 `message.link` 里的 64 条 `?comment=<旧id>&thread=<…>`（D19）。只动带 `thread=` 的行；可重跑。**普通迁移，随部署自动跑**，不是 deploy-then-drop。
- **错误码（新增 2 个）**：
  - `RATE_LIMITED`（platform，429，type URI 与 infra 逐字相同）；
  - `QUIZ_ANSWER_REQUIRED`（kungal，403）——先查过 infra 注册表，没有同义码。
  - 三处一译：`registry.go` + `registry_test.go` + `zh-CN/problem.json`。
- **代码位置**：`internal/wall/apiv1/**`（新包）；`internal/app/wall_v1.go` 组装；`router.go` 的 `apiv1.Setup(...)` 加一行；`app.go` 加一个字段。测试 `internal/app/v1_wall_*_test.go`，上游用测试里的内存假社区服务（`httptest`）。
- **删除**：22 条旧路由、两个 handler、两个 service 里变成死代码的部分；`legacy_route_baseline` −22；网页手写类型 `shared/types/galgame-community-comment.ts` 里被取代的部分。

## 5. 网页

- `utils/communityComment.ts` 改成「主体 → `subject_type` / `subject_id` + 每面文案与上限」；旧 URL 全删，只留调 X 轨 `/community/wall/*` 用的锚点映射（D20）。
- `useCommunityCommentList` / `Composer` / `Row` / `Like` / `FlagModal` 全部改走类型化客户端；正文用 `~/components/content/Document.vue` 渲染，**不再 `v-html`**。
- 按钮显隐改读 `viewer.can_*`，删掉 `useCan(surface.deletePermission)` 与 `isAuthor` 的本地镜像。
- 题目墙：`403 QUIZ_ANSWER_REQUIRED` 渲染成现在的「答题后可见」态。
- 资源 / 题目墙的「N 条评论」改读父资源的 `comment_count`（D18）。
- galgame 墙的深链接：`?comment=<post_id>` 先 `GET /wall-comments/{id}` 确认存在再翻页滚动；`?thread=` 分支删掉（D19）。

## 6. 本轨不做

- 提及推广到六面（D11）。
- `/community/wall/*` 与 `/community/following` 的 v1 化（X 轨）。
- `comment_count` 的一次性对账（漂移见 §1.2）。计数由本轨的写面继续维护，历史漂移另起一轮。
- 空墙清扫（上游的事，见记忆 `kungal-community-lazy-thread-and-reads`）。

## 7. 九条闸的特别说明

- **闸 5（分页）**：楼号在墙内唯一，排序键不可能并列——「数据里要有并列排序键」这条在本域没有对象。取而代之：页边界上必须放**被略去的帖**（封禁作者、别人的待审帖），证明略去不会让游标跳过或重复下一条。
- **闸 4**：上游是 HTTP 服务，测试用 `httptest` 的内存假社区（楼号、墓碑、待审、反应、举报、429、不可用都要能造）；本地表（计数、镜像、通知、feed、题目）走自己的临时库 `kungal_test_rc_wall`。

## 8. 变异题（实现之前提交；每条都必须让某个测试变红）

> 2026-09-22 修订（仍在实现之前）：`in_reply_to_user` → `addressee`、墓碑的 `content` 由 `null` 改为空文档（两处都是 G8 同名同型）、`RATE_LIMITED` 不带 `Retry-After`（上游不发）。M11、M12 的断言随之改字。

| # | 改动 | 应当杀死它的断言 |
|---|---|---|
| M1 | 删帖的版主权限改成「持任一面的删帖权限即可」（不按帖子所在的墙选权限） | 只持 `comment.galgame.delete` 的人删评分墙的帖 → `403` |
| M2 | 主人删帖时拿**请求者自己拥有的任一主体**比对，而不是帖子所在的主体 | 工具 A 的主人删工具 B 墙上的帖 → `403` |
| M3 | 评分墙的主人改回 galgame 创建者 | 评分作者能删自己评分墙上的帖（204），galgame 创建者不能（403） |
| M4 | 题目闸去掉「答过题」判断 | 答过题的人读题目墙 → `200` |
| M5 | 题目闸对匿名放行 | 匿名读有剧透的题目墙 → `403 QUIZ_ANSWER_REQUIRED` |
| M6 | `PUT …/like` 在已赞时照样切换 | 连续两次 `PUT`：仍是 `has_liked: true`，`like_count` 不变，作者萌萌点只 +1 一次 |
| M7 | 去掉自赞检查 | 作者赞自己的帖 → `403 SELF_LIKE_FORBIDDEN` |
| M8 | 游标指纹不含 `subject_id` | 墙 A 的游标拿去翻墙 B → `400 INVALID_CURSOR` |
| M9 | 别人的待审帖不略去 | 非作者的列表里没有它，作者的列表里有它且 `state: held` |
| M10 | 面的长度上限统一成 5000 | 1008 字的工具墙评论 → `422 TOO_LONG`，`max_length: 1007`；编辑同样 |
| M11 | 评分墙顶层帖不填评分作者 | 新建评分墙顶层帖的 `addressee` 是评分作者 |
| M12 | 上游 429 映射成 503 | 新人限流 → `429 RATE_LIMITED` |
| M13 | 写面在 `/users/batch` 失败时放行（K17 失效） | OAuth 不可用时发表 → `503`，上游没有收到帖子 |
| M14 | 删除墓碑时照样减计数 | 删两次：`204` 两次，父资源 `comment_count` 只减 1 |

## 9. 实现中改的（都在实现提交里，理由写在这里）

| 改了什么 | 为什么 |
|---|---|
| D6 加一条：持这堵墙编辑或删除权限的 staff 也能过题目闸 | 按原文，没答过剧透题的版主连这堵墙都读不到，更删不了违规帖 |
| 举报体的 `reason` → `flag_reason`；`note` 改成可空 | G8：`reason` 与 `Problem.errors[].reason` 同名异型；`note` 与 `UpvoteCreate.note` 可空性要一致 |
| list 与 create 显式声明 `404` | 这两个操作路径上没有 id，G4 推导不出 404，而主体闸会发它；契约一致性测试当场抓到 |
| 每个写操作先按 OAuth 当前记录查调用者：封禁 → `403 ACCOUNT_BANNED`，查不到 → `503` | 会话只在刷新令牌时才知道封禁，否则被封的人在令牌到期前一直能写 |
| galgame 墙 `@` 超过 20 人 → `422` + `TOO_MANY_ITEMS` | 旧写面的同一条规则（D11 保留现状），§3.5 漏写了这个 reason |
| 新注册 `INVALID_STATE_TRANSITION`（me 域）并把 `DomainMe` 加进域表 | 01 §2 早写了要复用它，但注册表里一直没有 |
| X 轨 `WallRead` 加 `thread_id` 缺席时的回落 | D20；只加不改，另附单测 |
| `moemoepoint` 的「赞给作者」AST 门认识注入的 `award` 调用形 | 删掉旧的 `ToggleLike` 让它只数到 14 处、低于 15 的防瞎阈值；新的点赞路径正是它该看的 |

## 10. 变异执行结果

脚本逐条改、跑、还原；两条最初「编译不过」（M6、M9 会留下未使用变量），按 §3.3 换成等价的恒真 / 恒假写法后重跑。

| # | 结果 | 杀死它的测试 |
|---|---|---|
| M1 | 杀死 | `TestV1WallDeletePermissions` |
| M2 | 杀死 | `TestV1WallDeletePermissions` |
| M3 | 杀死 | `TestV1WallDeletePermissions`、`TestV1WallCreateRatingAddressesTheRatingAuthor` |
| M4 | 杀死 | `TestV1WallQuizGate` |
| M5 | 杀死 | `TestV1WallQuizGate` |
| M6 | **先存活**，补测试后杀死 | `TestV1WallLikeIsASlot`。纠偏路径让重复 `PUT` 在接口层看不出差别；补的断言是「重放不得触达只会切换的上游」 |
| M7 | 杀死 | `TestV1WallLikeRefuses` |
| M8 | 杀死 | `TestV1WallListRefusesBadInput` |
| M9 | 杀死 | `TestV1WallListWalksEveryPageInPostOrder`、`TestV1WallListShapesATombstoneAndAHeldPost` |
| M10 | 杀死 | `TestV1WallCreateRefuses`、`TestV1WallUpdate` |
| M11 | 杀死 | `TestV1WallCreateRatingAddressesTheRatingAuthor` |
| M12 | 杀死 | `TestV1WallCreateRefuses` |
| M13 | 杀死 | `TestV1WallCreateRefuses` |
| M14 | 杀死 | `TestV1WallDeleteTwiceCountsOnce` |

## 11. 验收时抓到的

- 旧会话的封禁滞后（§9 第 4 行）是写测试时被 `ACCOUNT_BANNED` 用例抓到的：身份中间件对会话用户只在刷新时查封禁。
- 浏览器实测（开发环境，匿名 / 普通用户 / staff）：六面墙都由 SSR 渲染；题目剧透墙对匿名和普通用户显示「作答后可见」，对 staff 打开；发表 → 编辑（`**` 渲染成粗体，出现「已编辑」）→ 点赞 / 取消 → 刷新 → 删除成墓碑；举报弹窗 `204`；galgame `?comment=` 深链先 `GET` 单条再翻页。页面上的 `401 /api/user/preferences` 与头像 hydration 不一致都来自测试用的假会话（假 OAuth 令牌、假 user store cookie），与本轨无关。
- 开发库 community 里留下 2 条测试墓碑（资源 32023 墙上），以及评分 358 墙上的 1 条测试举报。
