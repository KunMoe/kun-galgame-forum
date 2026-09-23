# U2 · 公开资料、批量用户引用、通知偏好

> 用户轨第二段。旧路由 **4 条**：`GET /user/:id`、`GET /user/:id/floating`、`GET`/`PUT /user/notification-preferences`。迁移号段 120–129，**本段没有迁移**。
> U1 见 [u1-me.md](u1-me.md)（已上线，#183）；U3（「某用户的 X」9 条）等 G0 的 `work_id` 重编号。
> 普查在 [census/user.md](census/user.md) §1、§3.5；本文 §1 补生产数字。契约先于实现提交，变异清单见 §7。

## 1. 普查补充（生产实测 2026-09-23）

| 事实 | 数字 / 证据 |
|---|---|
| OAuth `users.status` | `0` 130,401 / `1` 82 / `2` 4。契约只说「非 0 时调用方应隐藏或脱敏」（`docs/oauth/03-cross-service.md:63`），**开放**整数 |
| 资料页「Galgame 工具」计数 | 数的是 **`galgame_website`（网站目录，87 行）**，不是 `galgame_toolset`（119 行）。数错了表，从来不对 |
| 「今日发布话题」 | `created >= CURRENT_DATE`，而生产库时区是 **UTC**——「今天」从北京 08:00 才开始，与 U1 修掉的签到是同一类错 |
| 话题数 | 含被版主隐藏的话题（生产 322 条 / 213 人），而话题列表都排除 `status = 1` |
| 名片调用方 | 全仓只有 `components/edit/topic/AccessUserPicker.vue`（话题 ACL 选人，按 id 补名字，**逐 id 一个请求**）；`KunAvatar` 的 `disable-floating` 是 KunUI 的属性，没有数据源 |
| 公开资料调用方 | `pages/user.vue`（`useKunFetch`），`UserInfo` 被 `ProfileHeader` / `info/Info.vue` 读全部 17 个计数；`docs/proj/app-direct-api.md` 列为 App 公开面；`../kungal-apps` 零命中 |
| 通知偏好调用方 | `components/message/NotificationPreference.vue`、`pages/message/muted.vue`；App 文档列为 Bearer 面 |

## 2. 逐条去向

| # | 旧路由 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /user/:id` | `GET /api/v1/users/{user_id}` | public |
| 2 | `GET /user/:id/floating` | `GET /api/v1/users?ids=`（U1 的 `/users` 加一个过滤器） | required |
| 3 | `GET /user/notification-preferences` | `GET /api/v1/me/notification-preferences` | required |
| 4 | `PUT /user/notification-preferences` | `PUT /api/v1/me/notification-preferences` | required |

4 条旧路由 → 3 个新操作 + 1 个既有操作加过滤器。`legacy_route_baseline` 下调 **4**。`UserHandler` 在本段之后只剩 U3 的 7 个列表方法。

## 3. 形状

### 3.1 `GET /api/v1/users/{user_id}` → `UserProfile`

```
UserProfile  object="user", id, name, avatar: Image|null, bio, roles: [string],
             created_at, moemoepoint, counts: UserCounts
```

- `object` 与 `UserRef`、`MyProfile` 同为 `"user"`（K10：摘要与详情同一个 object，重叠字段同名同型）。`name` / `bio` 按 G8 是 `string|null`；本面只下发可渲染的用户，所以实际不会是 null。
- `roles`：OAuth `roles` ∪ `site_roles`（`userclient` 已合并）里属于 `creator` / `moderator` / `admin` / `ren` 的那些，**封闭**枚举，`maxItems 4`。展示用（徽标），**不是**权限判据。（初稿写的是开放词表；G8 要求全 spec 的 `roles` 同型，而话题 ACL 的 `AccessGrants.roles` 已是这四个值的封闭枚举——网页也只用这四个显示徽标。隐含的 `user` 与站点自定义角色不下发。）
- `created_at`：注册时间，取 OAuth `created_at`；解析不了才退回 `kungal_user_state.created`（「首次出现在论坛」，见记忆 `kungal-user-state-lazy-provisioning`），description 写明。
- `moemoepoint`：`kungal_user_state` 的**缓存**余额（C3），**无 minimum**（生产有负数），与 U1 `Me.moemoepoint` 同型。OAuth 把余额当隐私，论坛一直公开展示（资料页、排行榜）；本段**保持公开**，这是对普查 §1.1 那条口径分歧的裁决（K24）。
- **不下发** `status`：见 3.1.1。

**3.1.1 不可渲染的用户一律 404（K24）。** OAuth 查无此人、或 `status != 0`（封禁 / 注销，生产 86 人）→ `404 NOT_FOUND`，与「看不到」同一个回答。旧 `/user/:id` 对封禁用户回 200 半资料（`created` 是 `0001-01-01`、计数全 0、`status` 是开放整数而网页只认 0/1），旧 `/floating` 回 404——两面裁决不同。v1 统一成 404，与渲染层隐藏封禁用户内容（`userclient.IsRenderable`）同一口径。`userclient` 失败 → `503`。

**3.1.2 `UserCounts`**（全部 integer ≥0，除注明者）

| v1 | 旧 | 语义（v1 里改了的加粗） |
|---|---|---|
| `topic_count` | `topic` | 发的话题，**不含被隐藏的（`status = 1`）**，与话题列表同一谓词 |
| `poll_count` | `topic_poll` | 发起的投票 |
| `lottery_count` | `topic_lottery` | 发起的抽奖 |
| `reply_count` | `reply_created` | 回复（`status = 0`） |
| `topic_comment_count` | `comment_created` | 话题评论（`status = 0`） |
| `community_comment_count` | `galgame_comment` | 六面评论墙上的可见帖（community 原语 `visible_posts`）。**integer\|null**：community 不可用且没有缓存值时是 `null`，**不再冒充 0** |
| `published_galgame_count` | `galgame` | 发布的 galgame（G 域 `GalgameUserStatsService`） |
| `contributed_galgame_count` | `contribute_galgame` | 贡献过编辑的 galgame（同上） |
| `published_galgame_today_count` | `daily_galgame_count` | 今天发布的（同上，G 域口径） |
| `galgame_rating_count` | `galgame_rating` | 评分 |
| `galgame_resource_count` | `galgame_resource` | 发的资源 |
| `toolset_count` | `galgame_toolset` | **发的工具（`galgame_toolset` 表）**；旧值数的是网站目录，是 bug |
| `toolset_resource_count` | `galgame_toolset_resource` | 工具资源 |
| `received_upvote_count` | `upvote` | 自己的话题被推的次数 |
| `received_like_count` | `like` | 自己的话题收到的赞 |
| `received_dislike_count` | `dislike` | 自己的话题收到的踩 |
| `topic_today_count` | `daily_topic_count` | 今天发的话题，**「今天」是北京日**（与 cron、U1 签到同一个时区常量） |

- G 域那三个计数由 `GalgameUserStatsService.Stats` 给出，它自己把 catalog 错误吞成 0，而且「今日发布」按**进程本地时区**算（`galgame_user_stats.go` 的 `time.Now()`，容器是 UTC）。G0 正在重写 galgame 域，**本段不改它**，description 写明；已告知 G 轨。
- 本地 SQL 计数失败 → `500`（旧实现 500，不变）。

### 3.2 `GET /api/v1/users?ids=` → `ListUserRef`

U1 已有 `GET /users?q=`。本段加 `ids`：

- `ids`：逗号分隔的十进制 id，1–100 个（逗号形，见 `kungal-api-v1-rebuild`：huma 只读首个重复键），去重，保持请求顺序。超过 100 个 → **`400 INVALID_PARAMETER`**（`TOO_MANY_ITEMS`）：上限在 schema 里，平台层在 handler 之前就拒了，与 `me/topic-states?topic_ids=` 同一行为（初稿误写成 422）。
- **`q` 与 `ids` 恰好一个**。都缺 → `422 VALIDATION_FAILED`（`parameter q`，`REQUIRED`）；都给 → `422`（`parameter ids`，`INCONSISTENT_WITH`）。
- 响应 `{object:"list", items:[UserRef], missing:[id]}`：请求了但不存在或不可渲染的 id 进 `missing`，不区分原因（与 `me/topic-states` 同一规则）。`q` 模式下 `missing` 恒为 `[]`。`missing` 是新增字段，永远存在。
- 走 `userclient.Users`（`/users/batch`，热缓存 + singleflight），一次请求最多一次上游批量。上游失败 → `503`。
- 档位不变（required）：唯一调用方是编辑器里的 ACL 选人。

### 3.3 通知偏好 `GET/PUT /api/v1/me/notification-preferences`

```
NotificationPreferences  object="notification_preferences", muted_types: [MutedType]
MutedType = notifytype.Type 的 18 个 v1 token ∪ "chat"（封闭）
```

- 线上是 **v1 token**（M 轨 `internal/message/notifytype`，如 `favorited`、`best_answer_chosen`），存储是库值，经 `notifytype.ToDB` / `FromDB` 互译；`chat` 原样存取（`notifytype.KeyChat`）。
- PUT 整体替换，幂等。schema：`maxItems 19`（全部可静音项）、`uniqueItems`、封闭枚举。**未知 token → `422 VALIDATION_FAILED`**，`reason UNKNOWN_VALUE`（由 schema 枚举产出；旧实现静默丢弃）。
- GET：库里的历史脏值（`FromDB` 不认识的）丢掉再下发；`FindByID` 失败 → `500`（旧实现当「一个都没静音」）；没有 state 行 → `[]`。PUT 先 `Ensure`。
- 这是 U1 契约 §3.4 定下、推迟到本段的那一组规则，原样执行。

## 4. 错误码

**不新增**。本段用到的全是既有码：`NOT_FOUND`、`SERVICE_UNAVAILABLE`、`INTERNAL_ERROR`、`VALIDATION_FAILED`、`MISSING_CREDENTIAL` / `INVALID_CREDENTIAL`。

## 5. 新 K 决定

- **K24**：公开资料只对可渲染用户存在——`status != 0` 与查无此人同为 `404`；`moemoepoint` 作为论坛展示数据保持公开（OAuth 视之为隐私，论坛的资料页与排行榜一直公开它，本条把分歧落成明文）。

## 6. 网页

| 文件 | 改什么 |
|---|---|
| `pages/user.vue` + `shared/types/user.ts` 的 `UserInfo` | → `GET /users/{user_id}`（SSR 走 `useApi`）；`UserInfo` 换成生成的 `UserProfile` 别名。404 走页面既有的「不存在」处理（读现有代码） |
| `components/user/ProfileHeader.vue`、`components/user/info/Info.vue` | 字段按 §3.1.2 改名（`counts.*`），删掉 `KUN_USER_STATUS_MAP` 的使用（`status` 不再下发）；`id` 是字符串 |
| 其余读 `UserInfo` 的 `pages/user/[id]/*.vue` | 只读 `id` / `name` 的照改类型（`id` 字符串）；U3 的列表调用不在本段 |
| `components/edit/topic/AccessUserPicker.vue` | 删掉逐 id 的 `/floating` raw `$fetch`，改成一次 `GET /users?ids=`，把缺失的 id 当「查不到」处理（现有逻辑是什么就保持什么） |
| `components/message/NotificationPreference.vue`、`pages/message/muted.vue`、`constants/notification.ts` | → v1。`notificationCategoryGroups` 各项的 `key` 从库值（`favorite`、`solution`、`pin-reply`…）换成 v1 token；M 轨留下的 `legacyMuteKeyToNotificationType`（库值 → v1 token）是给旧偏好接口做的翻译，本段之后偏好本身就是 v1 token，它的每个使用点改成直接用 token，然后删掉它 |

`docs/proj/app-direct-api.md` 的 `GET /api/user/:id` 与通知偏好两行改成 v1；`CHANGELOG.md` 记一条。

## 7. 变异清单（先于实现提交）

| # | 破坏 | 应当变红的性质 |
|---|---|---|
| 1 | 不可渲染判据去掉 `status != 0`（只看查无此人） | 封禁用户的资料 → 404 |
| 2 | `topic_count` 去掉 `status != 1` | 被隐藏的话题不计入 |
| 3 | `toolset_count` 改回数 `galgame_website` | 发的工具数正确 |
| 4 | `topic_today_count` 改回 `CURRENT_DATE`（UTC） | 北京 00:30 发的话题计入「今天」（测试钉时钟） |
| 5 | community 上游失败时回 0 而不是 null | `community_comment_count` 为 null |
| 6 | `ids` 模式把不可渲染的 id 静默丢掉而不进 `missing` | `missing` 收全 |
| 7 | `ids` 上限 100 → 1000 | 101 个 id → 400 `TOO_MANY_ITEMS` |
| 8 | `q` 与 `ids` 同时给不报错 | → 422 `INCONSISTENT_WITH` |
| 9 | 通知偏好 PUT 存 v1 token 而不经 `ToDB` | 存储是库值（直接查表），且红点的静音过滤仍生效 |
| 10 | GET 不经 `FromDB`，把库值原样下发 | 下发的是 v1 token（`favorite` → `favorited`） |
| 11 | 通知偏好 GET 的 `FindByID` 错误吞掉 | → 500 |

## 8. 删旧路由

4 条全删；`FloatingCardRequest/Response`、`UserProfileDetail`、`GetFloatingCard`、`FindFloatingStats`（吞错的那个）、`GetUserProfile` 等只被它们用到的全部删掉，`deadcode -test ./...` 到不动点。`routes.golden` 重生成，`legacy_route_baseline` 下调 4。

## 9. 迁移

**无**。
