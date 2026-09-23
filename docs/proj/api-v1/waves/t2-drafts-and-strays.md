# T2 · 草稿与零散读面

> 话题轨第二段。旧路由 **7 条**：草稿 4 + `/topic/interactions/mine` + `/topic/:tid/reply/locate` + `GET /api/resource`。
> 迁移号 **110**。契约先于实现提交，变异清单见 §7。

## 1. 普查结论（生产实测 2026-09-22）

| 事实 | 数字 / 证据 |
|---|---|
| 草稿总量 | 69 条 / 57 人，单人最多 5 条（上限 30），最早 2026-07-05 |
| `category` 空串 | 39 / 69（`galgame` 13 / `technique` 12 / `others` 5） |
| `sections` | 最多 2 个；`is_nsfw` 为真 8 条 |
| `idx_topic_draft_user` | `(user_id, updated DESC)`，**没有 `id` 决胜键** |
| `topic_favorite` | 8,642 行 / 3,807 人，单人最多 **127** |
| `topic_reaction` | 15,334 行，单人最多 **559**，p99 = 34 |
| `GET /api/resource` | 全站**零调用方**（网页、Nitro `server/`、Flutter App、`app-direct-api.md` 都没有） |

## 2. 逐条去向

| 旧路由 | v1 去向 |
|---|---|
| `GET /topic/draft` | `GET /api/v1/me/topic-drafts`（游标集合） |
| `POST /topic/draft` | `POST /api/v1/me/topic-drafts` |
| `GET /topic/draft/:id` | `GET /api/v1/me/topic-drafts/{draft_id}` |
| `DELETE /topic/draft/:id` | `DELETE /api/v1/me/topic-drafts/{draft_id}` |
| `GET /topic/interactions/mine` | `GET /api/v1/me/topic-states?topic_ids=…`（批量道，必填） |
| `GET /topic/:tid/reply/locate` | **不重建**。`Comment` 加 `reply_floor`，网页改打已有的 `GET /api/v1/comments/{comment_id}` |
| `GET /api/resource` | **纯删除**，零调用方 |

## 3. 草稿：四个面

### 3.1 草稿是快照，不是可编辑文档（K21）

`Save()` 永远 `draftRepo.Create()`，**没有更新路径**。载入一份草稿再保存会变成两份。

这不是 bug，是产品设计：按钮写的是「保存当前为草稿」，载入只是把内容覆盖进编辑器 store，全程不跟踪草稿 id（`useTopicDraft.ts` / `DraftModal.vue` 实测）。所以 v1 **没有 `PATCH`/`PUT`**，写进契约免得以后有人「把它修好」。

配套：30 条上限要用户自己删。满了回 `409 DRAFT_LIMIT_REACHED`（旧行为是 `ErrBadRequest` + 中文）。

### 3.2 命名

| 旧 | v1 | 理由 |
|---|---|---|
| `section`（数组用单数键） | `sections` | 立项依据点名的那条 |
| `content` | `content_markdown` | 与 `TopicCreate` 逐字一致 |
| `cover_images`（`/image/{hash}` token） | `cover_image_hashes` | 与 `TopicCreate` 逐字一致；网页两处都用同一个 `coverHashFromToken` |
| `updated` | `updated_at` | 并补 `created_at` |
| `id`（整数） | `id`（十进制字符串） | |

`category` 生产有 39 条空串 → v1 是**可空**的 `"galgame"|"technique"|"others"|null`，不是带空串成员的枚举。草稿是半成品，`null` 才是它的真实含义。

### 3.3 形状

```
TopicDraft        object=topic_draft, id, title, content_markdown, category|null,
                  sections[], is_nsfw, cover_image_hashes[], created_at, updated_at
TopicDraftSummary object=topic_draft_summary, id, title, summary, created_at, updated_at
```

`summary` 是 `LEFT(content,120)`，原始 Markdown 带 token，**自由文本，永不作判据**。
`sections` / `cover_image_hashes` 数组永不为 null。

写面 `POST` 请求体与 `TopicCreate` 同名同形，但**全部可选**（草稿是半成品）：
`title` ≤233、`content_markdown` ≤100007、`category`、`sections` ≤3、`is_nsfw`、`cover_image_hashes` ≤9。
标题与正文**去空白后都为空**则 `422 VALIDATION_FAILED`（`REQUIRED`，pointer `/content_markdown`）。
K19：`maxLength` 作用于原始值，去空白在其后。
K12：`POST` 要 `Idempotency-Key`。

### 3.4 列表分页

K11 判定为**游标**（「我的 xxx」是流）。排序 `updated_at DESC, id DESC`，`limit` 1–100 默认 20。

**迁移 110**：`idx_topic_draft_user` 换成 `(user_id, updated DESC, id DESC)`。
生产当前 `(user_id, updated)` 零并列，所以缺决胜键还没暴露——这正是 W4 踩过的坑：**没有并列的数据测不出缺失的决胜键**，所以测试种子必须造并列（见 §7）。

## 4. `me/topic-states`：只有批量道

```
GET /api/v1/me/topic-states?topic_ids=12,34,56
→ { object: "list", items: [ {object:"topic_state", topic_id, has_favorited, reactions:[…]} ], missing: […] }
```

- `topic_ids` **必填**，1–100 个。超过 100 → `422 VALIDATION_FAILED` / `TOO_MANY_ITEMS`，pointer `/topic_ids`。
- **不分页**：无 `cursor`、无 `limit`、无 `next_cursor`。
- 请求了但看不到的 id（不存在、话题不可读）一律进顶层 `missing[]`，不区分原因（不重新引入存在性 oracle）。
- 匿名调用 → `401 MISSING_CREDENTIAL`。这是 `me` 面。

### 为什么不做游标道

infra `05 §4` 的原型 `GET /v2/me/work-states` 两条道都有。这里**只做批量道**，因为：

- 两个调用方（`activity/card/Topic.vue`、`activity/card/TopicUpvote.vue`）都是首页动态流的卡片，手里已经有 topic id；
- 旧接口是**无界全量倾倒**：最重的用户一次拿 127 + 559 = 686 行，只为了渲染屏幕上的十来张卡，而且只增不减；
- 没有调用方的道就是投机——同一条理由让 `GET /api/resource` 被删掉。

需要「我的收藏」整页时再加游标道，那是 U 轨的事。

### 为什么不是 `VALIDATION_FAILED` 之外的码

infra `05 §4` 写的是 `400 TOO_MANY_IDS`。本仓有更细的校验词表（`TOO_MANY_ITEMS` + `max_items` 参数 + JSON pointer），一个扁平的 `TOO_MANY_IDS` 反而少说了是哪个参数、上限多少。**此处不适用 infra 条款**，用本仓词表。

## 5. `reply/locate`：不重建

网页唯一的调用方是 `topic/detail/Detail.vue`，只在 `?comment=` 深链时调，且**只读 `located.floor`** —— `page` / `reply_id` / `comment_id` 三个字段一个都没用过。

`page` 在 v1 里本来就没有意义：回复面是游标集合（`from_floor` 开窗），没有页码。

所以：

- `Comment` 加一个字段 `reply_floor`（所在回复的楼层）。`Comment` 本来就带 `reply_id`，楼层是同一类导航信息，而楼层才是人看得懂的地址。
- 网页改打已有的 `GET /api/v1/comments/{comment_id}`，拿 `reply_floor` 喂给 `loadInitialReplies({ fromFloor })`。往返次数不变，少一条路由。
- 可见性：`GET /api/v1/comments/{id}` 已经走话题可见性（W5a `TestV1GetComment` / `TestV1GetCommentNotFound`），与旧 `locate` 的 `requireTopicRead` 等价。

## 6. 错误码

新增 **1 个**，三处一译缺一不可：

| 码 | 域 | 状态 | 用处 |
|---|---|---|---|
| `DRAFT_LIMIT_REACHED` | kungal | 409 | 草稿已 30 条 |

`pkg/problem/registry.go`（常量 + `Codes`）、`registry_test.go` 的 `requiredCodes` 精确计数、`apps/web/i18n/locales/zh-CN/problem.json`。

## 7. 变异清单（先于实现提交）

每条是一行语义改动，每条都必须能让某个测试变红。编译不过的不算变异，要换等价破坏法。

| # | 破坏 | 应当变红的性质 |
|---|---|---|
| 1 | 草稿列表键集去掉 `id` 决胜键，只按 `updated DESC` | 同秒并列跨页时的全量遍历无重无漏 |
| 2 | `limit` 上限从 100 改成 1000 | `LIMIT_TOO_LARGE` |
| 3 | 最后一页仍然下发 `next_cursor` | 末页信号 |
| 4 | 游标指纹去掉 user id | 换一个用户复用游标应当 `INVALID_CURSOR` |
| 5 | 草稿计数判据 `count >= 30` 改成 `> 30` | 第 31 条被 `DRAFT_LIMIT_REACHED` 拦住 |
| 6 | `GetByIDForUser` 不带 `user_id` 条件 | 读别人的草稿回 404 |
| 7 | `DeleteForUser` 不带 `user_id` 条件 | 删别人的草稿回 404 且对方草稿还在 |
| 8 | `category` 空串直接下发而不是 `null` | 空 `category` 的草稿 |
| 9 | `sections` 为空时下发 `null` | 数组永不为 null |
| 10 | `topic_ids` 的 `maxItems` 从 100 提到 1000 | 101 个 id → `TOO_MANY_ITEMS` |
| 11 | 看不到的话题 id 静默丢弃而不进 `missing[]` | `missing[]` 收全 |
| 12 | `me/topic-states` 把别人的收藏也算进来（去掉 `user_id` 过滤） | 别人的收藏不出现在我的 state 里 |
| 13 | `Comment.reply_floor` 恒 0 | 深链定位 |

## 8. 网页

- `useTopicDraft.ts` 四个调用改走生成的类型化客户端；`section` → `sections`、`content` → `content_markdown`、`cover_images` → `cover_image_hashes`（用已有的 `coverHashFromToken` / 反向构 token）、`updated` → `updated_at`。
- `useMyTopicInteractions.ts` 改成按 id 批量（与 `useMyGalgameInteractions` 同形），两个卡片组件把自己的 topic id 传进去。
- `topic/detail/Detail.vue` 删掉 `kunFetch('/topic/:tid/reply/locate')`，改打 `GET /api/v1/comments/{comment_id}` 读 `reply_floor`。
- 闸：`pnpm typecheck`（**不是** `pnpm vue-tsc --noEmit`，那个在本仓什么都不检查）+ eslint + vitest + 真的 SSR 渲染一遍改过的页面。

## 9. 删旧路由

7 条全删，`legacy_route_baseline` 290 → 283。删完跑 `deadcode -test ./...` 到不动点，把删除暴露出来的死代码一并清掉。

## 10. 迁移

**110**：`idx_topic_draft_user` → `(user_id, updated DESC, id DESC)`。
生产命令在任务结束时明说（`docs/proj/api-v1/README.md` 的号段 110–119 属本轨）。

## 11. 验收时抓到的

- **openapi-fetch 把数组 query 发成 `topic_ids=1&topic_ids=2`，huma 只读第一个。** spec 里写着 `explode: false`（逗号），客户端默认不看 spec。实测：两个 id 用重复键发过去，200，`items` 只有第一个，第二个**既不在 `items` 也不在 `missing`**——卡片会静默显示「没收藏」。`me/topic-states` 是 v1 第一个数组 query 参数，所以之前没人撞上。修在 `shared/utils/api/client.ts` 的 `querySerializer`（全局，之后各轨的数组参数都受益），`client.spec.ts` 钉住逗号形，删掉那一行测试即红。
- **推 T2b 时没跑网页闸**，`Comment` 新增的必填 `reply_floor` 让两个 spec 的夹具类型不全，master 的 `web` 作业红了，而四个新开的 session 恰好从那一刻分支。`97476fdc` 热修。教训写回本波：后端加必填字段 = 网页夹具也要跟，推之前 `pnpm typecheck` 一次。
- **`reply/locate` 按话题限定评论，`GET /comments/{id}` 不限定。** `?comment=<别的话题的评论>` 现在会拿那条评论的楼层去当前话题定位，找不到目标就提示「目标回复或评论可能已被删除」。通知链接永远是同一话题的一对 id，所以不为手工拼的链接多一次往返。
- **删除的连锁比预想的深**：7 条路由带走了整个 `TopicHandler` / `ReplyHandler` / `TopicDraftHandler`、整个旧 `TopicService`（`NewTopicService` 再无调用方）、`mapper.go`、三个 DTO 文件，外加 `ReplyService` 构造函数收窄到只剩 `replyRepo`（`ModerationRemove` 是它唯一的活方法）。`deadcode` 对 master 做差分，两轮到不动点。
- **旧 `kunFetch` 棘轮** 301 → 295，这一段删掉 6 处调用。

浏览器实测（开发库，用户 2：33 收藏 / 556 表态）：首页 30 张卡片**一次**批量请求、逗号形、30/0；收藏态逐卡与服务端一致；`?comment=3126` 深链走 `GET /comments/3126` 拿到楼层 35 并滚到位；草稿存 → 列表刷新 → 载入覆盖编辑器 → 删除 204，测试草稿已删。
