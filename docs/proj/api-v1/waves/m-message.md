# M · 消息（通知 + 私信）

> 契约提交，2026-09-22。普查见 [census/message.md](census/message.md)（下称「普查」）。
> 本文是实现的**唯一依据**。实现不得自行新增错误码、迁移号或 `app.go` 字段；缺了就停下来报告。
> 轨：M，迁移号段 130–134，分支 `api-v1/m-message`。

## 0. 范围

`/api/message/**` 的 **11 条**旧路由，全部删除（§6）。取而代之的是 **10 个** v1 操作（§2）。

**不在本轨**（归 U 轨，`/user/**`）：`GET/PUT /api/user/notification-preferences`、`GET /api/user/status` 的红点。它们读写同一批表，所以本轨给 U 轨留了两样东西（§7）：v1 通知类型词表的 Go 定义，以及「未读」谓词。

**也不在本轨**：`/api/community/wall/*` 与 `/api/community/following`（X 轨）、`POST /api/image/message`（X 轨，私信图片上传，本轨网页继续调用它）。

## 1. 生产取值（2026-09-22 实测，只读）

普查 §9 的 15 条问题，按对契约的影响挑出来：

| 问题 | 结果 | 决定了什么 |
|---|---|---|
| `system_message` 行数 | **0**（`system_message_read_state` 有 75,800 行，全是空游标） | 系统公告**不迁移**，两条旧路由直接删，网页的「系统消息」页与侧栏行一起删（§4 裁决 A1） |
| `message` 行数 / 近 30 天 | 356,380 / 52,570 | 收件箱查询必须有索引（迁移 130） |
| `message` 上的索引 | 只有 pkey、`(type, created DESC)`、两个 partial | **没有 `receiver_id` 索引**，普查 S3 成立 |
| `message.type` 取值 | 17 个；`admin` **0 行**；`lottery-expired` 0 行但代码能写出来 | v1 词表 18 个，`admin` 不进词表（§3.3） |
| 同一收件人同一 `created` 的撞车组 | 2 | 少，但存在，排序必须带 `id` 决胜键；测试种子必须造并列 |
| `sender_id = 0` 的行 | 0 | 普查 S2「无名氏镜像行」是潜伏缺陷，不需要数据修复 |
| 镜像 `liked` 未读 | 3 | 普查 S2「标不掉」成立但量极小；v1 的已读标记按 id 走，顺带能清掉它们 |
| `link` 形状 | 全部落在 7 个前缀：`/galgame/N`（29.8 万）、`/topic/N[?reply=N\|?comment=N]`、`/galgame-quiz/N`、`/galgame/resource/N`、`/toolset/N`、`/website/<域名>`、`/galgame-rating/N`；无空、无截断 | v1 下发 `path`（§3.1、K-M3） |
| `content` | 206 条空；7,140 条含 `<`（绝大多数是 Markdown 自动链接 `<https://…>`，535 条 `<br>`）；2,454 条含图片 token；1,250 条超过 233 字 | 这是 **Markdown 片段快照**，不是正文，也不是纯文本（K-M2） |
| `chat_room` | 1,926 个，全是 `private`；**419 个没有任何消息**（旧 GET 的副作用造的） | v1 读面不建房（§4 C2）；空房不清理（§8） |
| `chat_message` | 9,950 条；447 条已撤回；461 条含图片；最长 901 字；`edit_time` 非空 0 条 | 撤回的**正文仍在库里**（§4 C5）；没有编辑 |
| `chatroom_name` ≠ 所属房间名 | 5,367 条，但能被旧 `OR` 子句拉进别的房间的 **0 条** | v1 只按 `chat_room_id` 查；不修数据 |
| `chat_room.last_message_content` 里的「xxx撤回了一条消息」 | 27 行 | v1 不读这一列（§4 C4） |
| `chat_room_admin` / `chat_message_reaction` | 0 / 0 | 死表，本轨不碰（§8） |
| 同 `(sender, receiver, type, link)` 的重复通知组 | 4,593 | 非原子去重的后果；写入方不在本轨（§8） |
| 静音键 | 19 个键全部有人用，没有 `wiki:*` / `system` 残留 | 静音语义保持，v1 通知列表按静音分区（§2.1） |

## 2. v1 操作

全部 `required` 鉴权（K3），全部在 `/api/v1/me/**` 之下：通知和私信只属于凭证本人，路径里不出现收件人。

| # | 方法 + 路径 | operationId | 取代 |
|---|---|---|---|
| N1 | `GET /api/v1/me/notifications` | `listNotifications` | `GET /message`、`GET /message/muted` |
| N2 | `GET /api/v1/me/notifications/summary` | `getNotificationSummary` | `GET /message/nav/system` 的通知那一半 |
| N3 | `PUT /api/v1/me/notifications/read-marker` | `markNotificationsRead` | `PUT /message/system/read` |
| N4 | `DELETE /api/v1/me/notifications/{notification_id}` | `deleteNotification` | `DELETE /message/:id` |
| C1 | `GET /api/v1/me/conversations` | `listConversations` | `GET /message/nav/contact` |
| C2 | `GET /api/v1/me/conversations/{user_id}` | `getConversation` | （新；私信页头取对方信息，并回答「能不能给这个人发」） |
| C3 | `GET /api/v1/me/conversations/{user_id}/messages` | `listDirectMessages` | `GET /message/chat/history` |
| C4 | `POST /api/v1/me/conversations/{user_id}/messages` | `sendDirectMessage` | `POST /message/chat/send` |
| C5 | `PATCH /api/v1/me/conversations/{user_id}/messages/{message_id}` | `updateDirectMessage` | `POST /message/chat/recall` |
| C6 | `PUT /api/v1/me/conversations/{user_id}/read-marker` | `markDirectMessagesRead` | `GET /message/chat/history` 的副作用 |

`GET /message/admin`、`PUT /message/admin/read`、`nav/system` 的公告那一半：**删除，无替代**（§4 A1）。

**私聊会话按对方的 user id 寻址**（`{user_id}`），不按房间 id：从调用者的视角，一个私聊会话就是「我和这个人」，网页路由本来就是 `/message/user/{id}`，房间 id 从不出现在任何 v1 形状里。`{user_id}` 等于调用者自己 → `404 NOT_FOUND`（不存在和自己的会话）。

### 2.1 N1 `GET /me/notifications`

游标集合（K11）。参数：

| 参数 | 说明 |
|---|---|
| `cursor`、`limit` | 标准（1–100，默认 20；超限 `400 LIMIT_TOO_LARGE`；坏游标 `400 INVALID_CURSOR`） |
| `muted` | 布尔，默认 `false`。`false` = 调用者**没有**静音的类型；`true` = **只**看已静音的类型。两个分区互斥、并起来是全部 |
| `type` | 可选，封闭枚举（§3.3）。在分区之内再按类型过滤。未知值 `400 UNKNOWN_ENUM_VALUE` |

- 排序唯一：`created_at` 降序、`id` 降序决胜。**没有 `sort` 参数**（旧的 `asc` 没有调用方）。
- 游标的指纹绑定 `(user_id, muted, type)`：换了任何一个再用旧游标 → `400 INVALID_CURSOR`。
- `muted=false&type=<已静音的类型>` 是合法请求，结果为空列表——这是谓词的交集，不是静默失败。
- **发起人被封禁（`userclient.IsRenderable` 为假）的行不进 `items`**。过滤发生在 Go 里，所以一页可能不满 `limit`；实现必须**继续往后取**直到凑满 `limit+1` 条可渲染行或源耗尽，扫描上限 `10 × limit` 行，碰到上限就按最后扫描到的行出 `next_cursor`。**不得**因为过滤而返回「空 `items` + 无 `next_cursor`」却其实后面还有行。
- 发起人查不到（上游失败）：读面不整条失败（K17 读面条款），行照常下发、`actor` 为 `DeletedUserRef`（`name: null`）。
- 不发 `total`（游标集合，`include_total` 不提供：计数在 N2）。

### 2.2 N2 `GET /me/notifications/summary`

```json
{ "object": "notification_summary",
  "unread_count": 3, "muted_unread_count": 12,
  "latest": Notification | null }
```

- `unread_count`：非静音分区里 `is_read=false` 的行数；`muted_unread_count`：静音分区里的。**SQL 计数，含被封禁发起人的行**（封禁状态只有 OAuth 知道）。这与 N1 的 `items` 是不同谓词，写进字段描述，客户端不得拿它去推断列表长度。
- `latest`：非静音分区里按 N1 排序的第一条**可渲染**行（同 N1 的过滤）。没有 → `null`。
- 取代旧 `nav/system` 的五个 `Count` 吞错误：任何一个查询失败 → `500 INTERNAL_ERROR`。

### 2.3 N3 `PUT /me/notifications/read-marker`

请求体 `{ "up_to_id": ID, "muted": bool }`（`muted` 默认 `false`）。

- 语义：把**这个分区里** `id ≤ up_to_id` 的全部未读行标为已读。**只**动这个分区；**不**动 `id > up_to_id` 的行。
- 响应 200：`{ "object": "notification_read_marker", "marked_count": n, "unread_count": m }`，`m` 是标完之后这个分区剩下的未读数。
- 幂等（K16）：重放同一个请求 `marked_count` 为 0，仍是 200。
- `up_to_id` 不必指向一条存在的行，也不必属于调用者：它只是一个上界。
- 标记包含被封禁发起人的行（它们本来就不显示，旧实现里正是它们让红点永远熄不灭，普查 S1）。
- 标掉的行里有镜像行（`community_notification_id` 非空）时，按 id 转发给上游 community（沿用 `forwardRead`，§4 N6）。

### 2.4 N4 `DELETE /me/notifications/{notification_id}`

- 204。不存在或不属于调用者 → `404 NOT_FOUND`（旧的回 200，普查 S4）。两者不可区分。
- 删的是未读镜像行 → 转发已读给上游（沿用）。

### 2.5 C1 `GET /me/conversations`

游标集合。只列**至少有一条消息**的会话；**对方不可渲染（封禁）的会话不列**（同 N1 的过滤与补页规则）。

- 排序：`last_message_at` 降序、会话内部 id（房间 id）降序决胜。游标里放的是 `(last_message_at, room_id)`，房间 id 只活在不透明游标里。
- `last_message_at` 与 `last_message` 取自 `chat_message` 的**最后一行**（`MAX(id)`），**不**读 `chat_room.last_message_*` 这几列（§4 C4）。
- 条目是 `Conversation`（§3.4）。

### 2.6 C2 `GET /me/conversations/{user_id}`

- 返回 `Conversation`。还没有任何消息（包括房间不存在）也返回 200：`message_count: 0`、`unread_count: 0`、`last_message: null`、`last_message_at: null`。**不建房。**
- 对方不存在或不可渲染 → `404 NOT_FOUND`；`{user_id}` 是自己 → `404`；上游 `/users/batch` 失败 → `503 SERVICE_UNAVAILABLE`（这里要判断对方是否存在，判据只有 OAuth 知道，同 K17）。

### 2.7 C3 `GET /me/conversations/{user_id}/messages`

游标集合，`id` 降序（最新在前；网页自己倒过来画）。

- 房间不存在 → 200 空列表。**不建房、不写已读**（普查 S2：GET 有两种副作用，v1 全部去掉）。
- 只按 `chat_room_id` 查，**没有** `OR chatroom_name = ?`（普查 S4）。
- 对方不存在或不可渲染 → `404`；自己 → `404`；上游失败 → `503`。
- 条目是 `DirectMessage`（§3.5）。

### 2.8 C4 `POST /me/conversations/{user_id}/messages`

- **必填 `Idempotency-Key`**（K12：创建用户内容）。
- 请求体 `{ "content_markdown": string }`，`minLength 1`、`maxLength 1000`（K19：长度作用于原始值）。经 `markdown.NormalizeStoredContent` 之后去首尾空白为空 → `422 VALIDATION_FAILED`，`/content_markdown` + `REQUIRED`。
- 对方不存在或不可渲染 → `404`；自己 → `404`；上游失败 → `503`（K17，失败关闭）。
- 房间不存在就建：`INSERT … ON CONFLICT (name) DO NOTHING` 再按名字查回，两行参与者同样 `ON CONFLICT (chat_room_id, user_id) DO NOTHING`，**并发首发不得 500**（普查 S2 的竞态）。建房、插消息、更新 `chat_room` 的 `last_message_time`/`last_message_sender_id` 在同一个事务里。
- **房间一律按名字找**：名字是 `<较小 id>-<较大 id>`，`chat_room.name` 有唯一索引。生产实测房名与参与者 100% 一致、没有任何一对用户有两个房间、没有无参与者的房间，所以按名字查与旧的「参与者交集」查询等价，且走索引。
- `chat_room.last_message_content` 照旧写入正文（旧读者删完后它就没有读者了，§8），`last_message_sender_name` 照旧。
- 201 + `Location: /api/v1/me/conversations/{user_id}/messages/{message_id}` + `DirectMessage`。
- 这个 `Location` 指向的路径**没有 GET**。这是刻意的：单条私信没有独立的读面需求，`Location` 仍按 01 §5 给出资源的规范路径。

### 2.9 C5 `PATCH /me/conversations/{user_id}/messages/{message_id}`

撤回是状态迁移（01 §5），不是删除：撤回后这条消息仍以墓碑出现在双方的历史里。

- 请求体 `{ "state": "recalled" }`。`state` 是封闭枚举，只接受 `recalled`（`sent` 不是可迁移的目标：撤回不可逆）。
- 消息不在这个会话里（不存在、或属于别的房间）→ `404`。
- 消息是对方发的 → `403 PERMISSION_REQUIRED`（它在调用者的会话里、调用者看得见它，所以不是 404）。
- 已经撤回 → **200，无副作用**（重放安全，同 W5b §5.5 第 3 条的教训）。
- 成功 → 200 + 更新后的 `DirectMessage`（`state: "recalled"`、`content: null`、`recalled_at` 非空）。
- **不再**往 `chat_room.last_message_content` 写「xxx撤回了一条消息」（§4 C4、01 §7）。撤回的若是房间最后一条，`last_message_content` 置为 `''`。
- 没有撤回时限：沿用现状，写进描述。

### 2.10 C6 `PUT /me/conversations/{user_id}/read-marker`

请求体 `{ "up_to_id": ID }`。把这个会话里**对方发的**、`id ≤ up_to_id` 的消息标为调用者已读（写 `chat_message_read_by`，`ON CONFLICT DO NOTHING`）。

- 200：`{ "object": "direct_message_read_marker", "marked_count": n, "unread_count": m }`。
- 房间不存在 → 200、`0/0`。对方不存在 / 自己 / 上游失败同 C3。
- 幂等。

## 3. 形状

通用：id 是 `repr.DecimalID`；时间是 `repr.DateTime`（秒精度 UTC）；人是 `repr.UserRef`；数组永不为 null。

### 3.1 `Notification`

| 字段 | 类型 | 说明 |
|---|---|---|
| `object` | `"notification"` | |
| `id` | ID | |
| `type` | 封闭枚举（§3.3） | |
| `actor` | `UserRef` | 触发这条通知的人。旧 `sender`。查不到 → `DeletedUserRef`。**不可为 null**：18 个类型全都有发起人，`sender_id=0` 生产 0 行 |
| `actor_count` | int ≥1 | 折叠的镜像行里有几个人（只有 `followed_thread_activity` 会 >1，生产最大 10）。库里 `<1` 下发 1 |
| `item_count` | int ≥1 | 折叠的镜像行里有几条上游帖子。同上 |
| `path` | string，`maxLength 100`，`^/` | 站内网页路径，旧 `link` 原样（K-M3） |
| `excerpt_markdown` | string，`maxLength 1000` | 旧 `content`（K-M2）。可以是 `""` |
| `source` | 封闭枚举 `local` \| `community` | 旧 `community: bool`。`community` = 从 infra 社区原语镜像来的行 |
| `is_read` | bool | 旧 `status`（K-M1） |
| `created_at` | DateTime | 旧 `created`。镜像行是上游的 `updated_at`，折叠时会前移 |

不下发：`receiver_id`（恒等于调用者）、`updated`（无读者）。

### 3.2 `NotificationSummary`、两个 `*ReadMarker`

见 §2.2、§2.3、§2.10。

### 3.3 通知类型词表（v1 token ↔ 库值）

01 §3 要求封闭枚举是 snake_case；普查 §5 指出有 5 个名字与语义对不上，这次一并改掉。**库里的值不动**，映射在 Go 里做，一处定义（`internal/message/notifytype`，§7）。

| v1 `type` | 库 `message.type` | 含义 |
|---|---|---|
| `upvoted` | `upvoted` | 有人推了你的话题 |
| `liked` | `liked` | 有人赞了你的话题 / 回复 / galgame / 资源 / 评分 / 评论（靠 `path` 区分） |
| `favorited` | `favorite` | 有人收藏了你的内容 |
| `replied` | `replied` | 有人回复了你 |
| `commented` | `commented` | 有人评论了你 |
| `mentioned` | `mentioned` | 有人 @ 了你 |
| `followed_thread_activity` | `followed` | **你关注的评论区**有新评论（不是「有人关注了你」） |
| `best_answer_chosen` | `solution` | 你的回复被选为最佳答案 |
| `reply_pinned` | `pin-reply` | 你的回复被置顶 |
| `quiz_answered` | `quiz-answered` | 有人答了你出的题 |
| `resource_link_reported` | `expired` | **有人报告**你的资源链接失效 |
| `edit_requested` | `requested` | 有人对你的 galgame 提了更新请求 |
| `edit_merged` | `merged` | 你的更新请求被合并 |
| `edit_declined` | `declined` | 你的更新请求被拒绝 |
| `lottery_won` | `lottery-won` | 你中奖了 |
| `lottery_drawn` | `lottery-closed` | 你参与的抽奖开奖了 |
| `lottery_code_expired` | `lottery-expired` | 你的兑换码过了领取期限 |
| `poll_closed` | `poll-closed` | 你参与的投票截止了 |

库里出现词表之外的值（今天 0 行；`admin` 已确认 0 行）：**该行不下发**，并 `slog.Warn` 一次带 id。不得 500，不得下发未声明的枚举值。

`type` 查询参数、`Notification.type` 用 v1 token。静音偏好的存储（`kungal_user_state.muted_notification_types`）仍是库值，由映射层翻译——那张表归 U 轨。

### 3.4 `Conversation`

| 字段 | 说明 |
|---|---|
| `object` | `"conversation"` |
| `peer` | `UserRef`，对方 |
| `message_count` | 会话里消息总数（含已撤回） |
| `unread_count` | 对方发的、调用者没读过的条数（`chat_message_read_by` 里没有 `(message_id, 调用者)`） |
| `last_message` | `DirectMessage \| null` |
| `last_message_at` | DateTime \| null，等于 `last_message.created_at` |

没有 `id` 字段：会话的身份就是 `peer.id`（§2 寻址）。

### 3.5 `DirectMessage`

| 字段 | 说明 |
|---|---|
| `object` | `"direct_message"` |
| `id` | ID |
| `sender` | `UserRef` |
| `state` | 封闭枚举 `sent` \| `recalled` |
| `content` | `ContentDocument \| null`。`recalled` 时**恒为 null**（§4 C5）。用 `internal/apiv1/content` 的完整 Markdown 转换器（与回复同一个），图片 token 解析成 `Image` |
| `created_at` | DateTime |
| `recalled_at` | DateTime \| null |
| `viewer` | `{ "is_mine": bool }` —— 是不是调用者发的。网页据此决定左右布局与是否给撤回按钮 |

不下发：`chatroom_name`、`receiver_id`、`content_html`（K13）、`read_by`（恒空，A5）、`edit_time`（没有编辑功能，0 行）。

## 4. 逐条裁决（对普查）

### 通知

| 普查 | 裁决 |
|---|---|
| §2.1 排序无决胜键（S2） | `created DESC, id DESC`，键集分页 |
| §2.1 `sort_order` 字符串拼 SQL（S4） | 没有 sort 参数，不拼 |
| §2.1 `total` 与 `messages` 不同谓词（S1 的一半） | 列表不发 `total`；计数进 N2 并在描述里写明口径不同 |
| §2.1 `Count` 错误被丢 | 所有查询错误上抛 500 |
| §2.2 `type` 不在静音集合 → 静默空 | 语义改成「分区 ∩ 类型」，空是正确结果；未知 token 400 |
| §2.2 静音把通知从主列表整个拿走，而前端文案说「仍会保留」 | **行为保留**（主列表不显示已静音类型），文案是网页的错，网页轨改文案 |
| §2.3 删别人的 / 不存在的回 200（S4） | 404 |
| §2.4 mark-all-read 吃掉三样东西（S1） | N3 按分区 + `up_to_id` 上界。静音分区不被主列表清掉；SSR 之后新到的不被清掉；被封禁发起人的行被清掉（红点熄得灭了） |
| §2.4 `forwardRead` 失败丢弃剩余批次（S4） | **N6：本轨修**——失败的批次记日志后继续下一批。改一行，在 v1 共用的路径上 |
| §2.5/2.6 系统公告 | **A1：删除，无替代**。0 行、无写入方。`system_message` 与 `system_message_read_state` 两张表留着（U 轨的红点还在读，§7），drop 另走 deploy-then-drop（§8） |
| §2.7 nav 摘要按字节切中文（S2） | 不再切：`latest` 是完整 `Notification`，截断由客户端做 |
| §2.7 nav 摘要五个 `Count` 吞错误 | N2 任何查询失败 → 500 |
| §2.7 下标区分两行、`title:"zako~"` 等死字段 | 具名对象 |
| §3.1 没有 `receiver_id` 索引（S3） | 迁移 130 |
| §7 S2 镜像 `liked` 不被评论墙已读覆盖 | **不在本轨**（写方是 `/community/wall/read`，X 轨）。N3 能清掉它 |
| §7 S2 `sender_id = 0` 渲染成无名氏 | 生产 0 行；v1 映射成 `DeletedUserRef`，不崩 |

### 私信

| 普查 | 裁决 |
|---|---|
| **C1** §2.8 `route` 字段既是路由段又是 user id | `peer: UserRef` |
| **C2** §2.9 GET 建房 + 写已读（S2） | C2/C3 纯读；已读走 C6；建房只在 C4 |
| **C3** §2.9 建房竞态 500 | `ON CONFLICT (name) DO NOTHING` 再查回 |
| **C4** §2.11 中文句子持久化进 `last_message_content`（S3、01 §7） | v1 读面从 `chat_message` 取最后一条，不读那一列；撤回不再写句子 |
| **C5** 撤回后正文仍在库里（本轨实测：447 条） | v1 对 `recalled` 恒下发 `content: null`。**不擦库**：撤回的正文还是举报与处置的证据，是否物理擦除不是 API 层的决定（§8） |
| §2.9 `OR chatroom_name = ?`（S4） | 只按 `chat_room_id` |
| §2.9 `read_by` 恒空、`edit_time` 恒空 | 不下发 |
| §2.9 私信不过 `IsRenderable`（S3） | 对方不可渲染：C1 不列、C2/C3/C4/C6 回 404 |
| §2.10 收件人不查存在性（S3） | C4 查，K17 失败关闭 |
| §2.10 不回新消息，网页重拉整页 | 201 + `DirectMessage` |
| §2.11 查库失败与不存在合并成 404 | 查库失败 → 500 |
| §2.11 没有撤回时限 | 保持，写进描述 |

## 5. K 决定（本轨新增）

- **K-M1 · 通知的 `is_read` 是顶层字段，不进 `viewer`。** K9 管的是「同一个资源、不同查看者看到不同值」。一条通知只有一个合法查看者（收件人），`is_read` 是这一行本身的状态，不随查看者变化。`DirectMessage.viewer.is_mine` 则确实随查看者变（同一条消息，两个参与者看到相反的值），所以进 `viewer`。
- **K-M2 · `excerpt_markdown` 是截断的 Markdown 快照，不转结构化文档。** 它在写入时被截到 233 字（1,250 条历史行更长，下发上限给 1000 兜住），截断点可能落在任何语法中间；把它当 K13 正文解析会产出残缺的树。客户端把它当预览：剥成纯文本再显示（网页现有的 `markdownToText`）。字段名带 `_markdown` 后缀，就是为了不让人把它当 `content`。
- **K-M3 · 通知目标先下发站内路径 `path`，不下发结构化引用。** 生产 35.6 万行的 `link` 全部落在 7 个已知前缀，结构化是可行的，但今天唯一的消费者是网页，App 还不读通知（`kungal-apps` 里零调用）。按 A5 不发没人兑现的承诺；将来加 `target: {object, id, …}` 是加法，`path` 不动。
- **K-M4 · 私聊会话按对方 user id 寻址。** 理由见 §2。
- **K-M5 · 已读用「标记」而不是「全部已读」。** `PUT …/read-marker` + `up_to_id` 上界，是 K16 槽位的变体：置位幂等、重放无副作用，而上界让「我看到了哪里」成为请求的一部分，服务端不再替客户端猜。

## 6. 删除

同一个 PR 里删：

- 11 条路由（`router.go` 的 `/message/**` 两组）；
- `internal/message/handler/`、`internal/message/dto/` 整个目录；
- `MessageService` / `ChatService` 中只被旧 handler 用到的方法与因此变死的仓储方法（用 `deadcode` 跑到不动点，§3.6 of SOP）；**保留**被别处调用的：`notifier.go` 全部、`prefs.go` 全部（U 轨在用）、`MarkCommunityThreadRead`、`UpsertCommunityMirror`、`MaxCommunitySeq`；
- 网页：`apps/web/shared/types/message.ts` 与 `chat-message.ts` 里被 v1 生成类型取代的手写类型、`pages/message/system.vue`、`components/message/aside/System.vue` 与 `SystemItem.vue` 的公告用法；
- `legacy_route_baseline` 290 → **279**（其他轨同期也在删，合并时以重新生成为准）。

零调用方已证：`apps/web/server/` 无 `/message` 调用；`../kungal-apps` 无 `/message` 调用。

## 7. 给 U 轨的两样东西

1. **`internal/message/notifytype`**（新包，本轨建）：v1 token 的 huma 枚举类型、`ToDB(token) string`、`FromDB(dbValue) (token, ok)`、全部 18 个 token 的有序列表，外加一个静音伪键 `chat`（它不是通知类型，是私信的静音开关）。U 轨迁通知偏好时，v1 的静音键**必须**用这里的 token，而不是库值。
2. **「未读」谓词**：通知未读 = `receiver_id = me AND status = 'unread' AND type NOT IN (静音库值)`；私信未读同 §3.4 `unread_count`（`chat` 被静音则不计）；**系统公告恒为 0**（`system_message` 0 行、无写入方），U 轨的红点可以直接去掉这一项。

## 8. 本轨不做（照实记）

- `system_message` / `system_message_read_state` 的 drop（U 轨红点还在读；deploy-then-drop）。
- `chat_room.last_message_content` / `last_message_sender_name` 的 drop（C4 仍写它们以防回滚，下一轮清理）。
- 419 个空房间的清理（v1 读面已经看不见它们）。
- `chat_room_admin` / `chat_message_reaction` 两张死表。
- 撤回正文的物理擦除（需要产品决定：举报证据 vs 隐私）。
- 通知写入方的六种去重写法与 4,593 组重复（写方分散在话题、galgame 各轨）。
- `purge_repo.go` 按作者删 `system_message`（表是空的，无害）。
- 私信的拉黑 / 频率限制（没有产品需求记录）。

## 9. 迁移

**130 `message_inbox_keyset`**，纯加、幂等、不动数据、部署顺序无关：

```sql
CREATE INDEX IF NOT EXISTS idx_message_receiver_created_id ON message (receiver_id, created DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_chat_message_room_id ON chat_message (chat_room_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_chat_room_participant_user ON chat_room_participant (user_id, chat_room_id);
```

`message` 35.6 万行，建索引持锁约 1 秒，随部署自动跑即可。

## 10. 错误码

**不新增。** 全部复用：`NOT_FOUND`、`PERMISSION_REQUIRED`、`VALIDATION_FAILED`（`REQUIRED` / `TOO_LONG` / `TOO_SHORT`）、`UNKNOWN_ENUM_VALUE`、`INVALID_CURSOR`、`LIMIT_TOO_LARGE`、`SERVICE_UNAVAILABLE`、`INTERNAL_ERROR`、`IDEMPOTENCY_*`、`MISSING_CREDENTIAL` / `INVALID_CREDENTIAL` / `ACCOUNT_BANNED`（框架给）。

## 11. 变异题

实现之前提交。实现完成后逐条执行，「改了什么 / 哪个测试红了」写进 PR。

| # | 改动（语义，不是语法） | 必须变红的断言 |
|---|---|---|
| M1 | N1 列表的 `ORDER BY` 去掉 `id` 决胜键 | 全量遍历（**种子里同一收件人有 ≥3 行同 `created`**，`limit=2`）出现重复或遗漏 |
| M2 | N1 `muted=false` 不排除静音类型 | 静音类型的行出现在默认分区 |
| M3 | N1 游标指纹不含 `type` | 带 `type=liked` 取到的游标换成 `type=replied` 使用应得 `400 INVALID_CURSOR` |
| M4 | N1 被封禁发起人的行被过滤后不补页（直接返回短页且不出 `next_cursor`） | 种子里一页全是封禁发起人、后面还有正常行时，第一页必须有 `next_cursor` 或已补满 |
| M5 | N3 忽略 `muted`，两个分区一起标 | `muted=false` 标完后，静音分区的未读数不变 |
| M6 | N3 忽略 `up_to_id`，标全部 | `id > up_to_id` 的行仍是未读 |
| M7 | N4 删不属于自己的行回 204 | 别人的通知 id → 404，且那一行还在 |
| M8 | C3 在房间不存在时建房（恢复旧 GET 副作用） | 对一个新对方调用 C3 之后 `chat_room` 行数不变 |
| M9 | C5 允许撤回对方的消息 | `403 PERMISSION_REQUIRED`，且那条消息仍是 `sent` |
| M10 | `recalled` 的消息仍下发 `content` | `content` 必须是 `null` |
| M11 | C4 不查收件人存在性 | 不存在的 user id → 404，且没有建房、没有插消息 |
| M12 | C1/C6 的 `unread_count` 把自己发的也算进去 | 只有自己发的消息的会话 `unread_count == 0` |
| M13 | C4 首发建房去掉 `ON CONFLICT`（改回先插后查） | 房间名已存在（预先插一个无参与者的同名房）时首发必须 201 而不是 500 |

## 11.5 首轮实现后对本契约的修正（督查，2026-09-22）

首轮实现照契约写完，契约门（`TestV1Gates`）报了 9 条。都是契约本身的错，不是实现的错，逐条改契约：

1. **`Notification.type` → `notification_type`**，查询参数 `type` 同改。G8：`Problem.type` 是 URI 字符串，同名不同型（W5b 的 `choice_type` 是同一个坑）。
2. **`muted` → `is_muted`**（N1 查询参数、N3 请求体）。F1：布尔必须以 `is_` / `has_` / `can_` 开头。
3. **`DirectMessage.content` 不可为 null。** G8：其它所有 `content` 都是非空 `ContentDocument`。撤回的消息下发**空文档**（`children: []`），客户端看 `state`。M10 的断言相应改成「撤回后 `content.children` 为空数组」。
4. **`Conversation` 加 `id`**，值等于 `peer.id`（即路径段 `{user_id}`）。G17：写面地址里的每个 `{…_id}` 前缀都必须有一个回 200 且带 `id` 的 GET。§3.4「没有 `id` 字段」作废；身份仍是对方。
5. **新增 N5 `GET /api/v1/me/notifications/{notification_id}`**（`getNotification`）。G17：N4 的 DELETE 需要同地址的 GET。不属于调用者 / 不存在 / 类型不在词表 / 发起人被封禁 → 404（与 N1 的可见性同一谓词）。
6. **新增 C7 `GET /api/v1/me/conversations/{user_id}/messages/{message_id}`**（`getDirectMessage`）。G17，也让 C4 的 `Location` 有了读面——§2.8「`Location` 指向的路径没有 GET」作废。对方检查同 C3；消息不在这个会话 → 404。
7. 一处响应描述里写了 reason 名 `TOO_LONG`，G4 把它当 code 查注册表。改措辞。

另外两条：

8. **变异补一题 M12b**：C6 回的 `unread_count` 把自己发的也算进去。首轮 13 题全杀，这一题活了（C6 的剩余未读用的是另一条 SQL，M12 只打在 C1 那条上）。
9. **community 转发没接上**：`App` 上没有 `MessageService`，首轮实现只好传 `community=nil`，N3/N4 标掉镜像行时不会通知上游。督查在 `app.go` 加一个字段 `Messages *msgService.MessageService`（共享面，只加一行）。

操作数因此是 **12 个**（N1–N5、C1–C7），取代 11 条旧路由。

## 12. 九条闸

照 [05-session-sop.md](../05-session-sop.md) §5。本轨的临时库：`kungal_test_m_message`。
