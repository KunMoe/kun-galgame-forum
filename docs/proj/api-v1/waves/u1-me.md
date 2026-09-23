# U1 · 我的：状态、签到、萌萌点、偏好、资料

> 用户轨第一段。U 轨共 25 条旧路由（`/api/user/**` 全部），迁移号段 120–129，内部切三个 PR 串行：
>
> | 段 | 范围 | 旧路由 |
> |---|---|---|
> | **U1**（本文） | 只属于「我」的面：状态、签到、萌萌点流水、云端偏好、成人向显示、@ 搜索、资料三写、创作者两条 | **12** |
> | U2 | 公开资料 `/user/:id` + 名片 `/user/:id/floating` → `/users/{user_id}` + 批量引用；**通知偏好两条**（见 §3.4） | 4 |
> | U3 | 「某用户的 X」：topics / replies / comments / galgames / galgame-comments / resources / ratings（`UserHandler`）+ 寄挂的 toolsets / collections | 9 |
>
> 12 + 4 + 9 = 25，与 `routes.golden` 里 `/api/user/**` 的行数相等。§2 的逐条表是权威。
>
> **本段没有迁移**。契约先于实现提交，变异清单见 §9。普查在 [census/user.md](census/user.md)；那份普查写于 45127518，之后新加的 `ContentPrefsHandler` 三条（preferences ×2、nsfw）由本文 §1.3 补查。

## 1. 普查补充（生产实测 2026-09-22）

### 1.1 签到时区 bug 是真的，而且每天都在发生

普查 §3.2 的推断用 OAuth 库核实了：

| 事实 | 数字 |
|---|---|
| 论坛今天 `daily_check_in = 1` 的人 | 524 |
| OAuth 今天（北京日）`reason = daily_checkin` 且键为 `kungal:checkin:%` 的流水 | **421** |
| 差 | **103 人今天签到没到账**（约 20%） |
| 近 14 天签到流水按北京小时分布 | 00 时 1469 条是全天最高峰——大量签到正落在 00:00–08:00 的错位窗口里 |

机制：幂等键日期取进程本地时区（容器是 UTC），闸门由 Asia/Shanghai 的 cron 在北京 00:00 清零。北京 00:00–08:00 签到的人，键与「昨天北京 08:00 之后那次」逐字相同、`delta` 是新随机数 → OAuth `400/16004` → `Award` 只打一行 warn。用户看到「+N」，实际 0 分，当天也不能再签。

### 1.2 其余数字

| 事实 | 数字 / 证据 |
|---|---|
| `kungal_user_state` | 100,756 行；**63,363 行余额恰好是 7**（`Ensure` 的种子值，普查 C3 #2） |
| 本地缓存余额范围 | **−16 … 38,150**。余额可以是负数，所以 `moemoepoint` 字段**不能**声明 `minimum: 0` |
| 有静音偏好的人 | 55 人，单人最多 19 项 |
| `role_permission_override` | 27 行（`/perm/bundles` 泄露的是真配置——但那条属 P 轨，不在本段） |
| 调用方（Flutter App `../kungal-apps`） | 本段 12 条零命中；`docs/proj/app-direct-api.md` 列了 `/user/status` 与 `/user/notification-preferences` 两条 |

### 1.3 `ContentPrefsHandler` 三条（普查之后才有）

| 旧路由 | 做什么 | 调用方 |
|---|---|---|
| `GET /api/user/preferences` | 代理 OAuth `GET /auth/me/preferences/{client_id}`，命名空间由服务端定、客户端不可选 | `composables/useCloudPreferences.ts` 的 `pull` |
| `PUT /api/user/preferences` | 代理写，透传 `If-Match`；18006 → 旧码「冲突」、18001 → 旧码「云端不可用」 | 同上 `push` |
| `PUT /api/user/nsfw` | 代理 OAuth `PUT /auth/me/nsfw`，**并把新立场写回 Redis 会话**（否则要等会话刷新才生效） | `composables/useContentStance.ts` |

三条都设 `Cache-Control: no-store`（v1 本来就全部 no-store，K15）。年龄确认已于 2026-09-23 退役（`docs/oauth/15`），`adult_confirmed` 恒真——v1 不再下发它。

## 2. 逐条去向

| # | 旧路由 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /user/status` | `GET /api/v1/me` | required |
| 2 | `POST /user/check-in` | `POST /api/v1/me/check-ins` | required |
| 3 | `GET /user/moemoepoint/log` | `GET /api/v1/me/moemoepoint-entries`（游标） | required |
| ~~4~~ | `GET /user/notification-preferences` | **移到 U2**（§3.4） | — |
| ~~5~~ | `PUT /user/notification-preferences` | **移到 U2**（§3.4） | — |
| 6 | `GET /user/preferences` | `GET /api/v1/me/preferences` | required |
| 7 | `PUT /user/preferences` | `PUT /api/v1/me/preferences` | required |
| 8 | `PUT /user/nsfw` | `PUT /api/v1/me/nsfw-display` | required |
| 9 | `GET /user/search` | `GET /api/v1/users?q=` | required |
| 10 | `PUT /user/bio` | `PATCH /api/v1/me/profile`（`bio`） | required |
| 11 | `PUT /user/username` | `PATCH /api/v1/me/profile`（`name`） | required |
| 12 | `POST /user/avatar` | `PUT /api/v1/me/avatar` | required |
| 13 | `GET /user/creator/status` | `GET /api/v1/me/creator-status` | required |
| 14 | `POST /user/creator/apply` | `POST /api/v1/me/creator-applications` | required |

本段 12 条旧路由对应 11 个 v1 操作（bio 与 username 并成一个 `PATCH`）。`legacy_route_baseline` 在本段下调 **12**。

全部 v1 操作挂 `internal/user/apiv1/`，tag `users`。

## 3. 形状

### 3.1 `GET /api/v1/me` → `Me`

```
Me  object="me", id, moemoepoint, has_checked_in_today, has_unread_messages,
    is_creator, toolset_upload_today_bytes
```

| 字段 | 类型 | 来源与语义 |
|---|---|---|
| `id` | id 字符串 | 会话里的用户 id |
| `moemoepoint` | integer，**无 minimum**（生产有 −16） | `kungal_user_state.moemoepoint`，**是 OAuth 余额的缓存**（C3），description 里写明会滞后 |
| `has_checked_in_today` | bool | `daily_check_in = 1`；闸门按 Asia/Shanghai 日重置 |
| `has_unread_messages` | bool | 通知（`receiver_id = 我 AND status = 'unread' AND type NOT IN 已静音的库值`）或私信（对方发的、没有我的 `chat_message_read_by` 行；已静音 `chat` 时不计）任一未读。**不含系统公告**：`system_message` 生产 0 行、没有写入方，M 轨删掉它的管理面且不重建（M 契约 §7，2026-09-22 协调） |
| `is_creator` | bool | `role.IsCreator(userclient.User(id).Roles)`（含 site roles） |
| `toolset_upload_today_bytes` | integer ≥0 | `daily_toolset_upload_bytes` |

旧名对照：`moemoepoints` → `moemoepoint`（一义两名）、`is_check_in` → `has_checked_in_today`、`has_new_message` → `has_unread_messages`、`daily_toolset_upload_bytes` → `toolset_upload_today_bytes`。

**吞错全部取消**：`kungal_user_state` 查询失败、两个未读计数任一失败 → `500 INTERNAL_ERROR`（旧实现静默当 0，红点消失）。`userclient.User` 失败 → `503 SERVICE_UNAVAILABLE`。**没有 state 行**（从未在论坛出现过）不是错误：先 `Ensure`，再读。

### 3.2 `POST /api/v1/me/check-ins` → `CheckIn`

```
CheckIn  object="check_in", check_in_date (YYYY-MM-DD, Asia/Shanghai),
         moemoepoint_awarded (0..7), moemoepoint (签到后的余额, 无 minimum)
```

- 200（不是 201：签到没有可 GET 的地址，不发 `Location`）。`Idempotency-Key` **可选**（K12「必须支持」）。
- 今天已签 → `409 ALREADY_EXISTS`（复用 infra `me` 域码）。
- **K22 · 签到的日期、键与奖励都由「北京日」决定，且奖励是确定的。**
  - 日期：`time.Now().In(Asia/Shanghai)`，与 cron 清零同一个时区（`cron.scheduleTZ`，把它导出或另建共享常量，不许写第二个字面量）。
  - 幂等键：`moemoepoint.Key("daily_checkin", "<user_id>_<YYYY-MM-DD>")` → `kungal:daily_checkin:12_2026-09-23`。事件段换名，与旧键 `kungal:checkin:…` 的命名空间**永不相交**，部署当天也不会撞上旧键。
  - 奖励：`delta = fnv32a(键) % 8`，**不再是 `rand`**。同一人同一天永远是同一个数，所以任何重试都是 OAuth 眼里的同体重放，16004 从结构上消失。
- 顺序：`Ensure` → 闸门 `UPDATE … SET daily_check_in = 1 WHERE user_id = ? AND daily_check_in = 0`（没抢到 → 409）→ `delta == 0` 直接成功（不调 OAuth）→ 否则 **`AwardSync`**（同步，不再 `go func`）。
  - `AwardSync` 失败 → **把闸门还原成 0**，回 `503 SERVICE_UNAVAILABLE`。用户可以立刻重试，而且因为键与 delta 都确定，重试不会重复加分。
  - 旧实现的「先回一个随机数再异步加分」取消：v1 回的 `moemoepoint_awarded` 就是到账的数，`moemoepoint` 是 OAuth 返回后写进缓存的余额（`AwardSync` 已经写了缓存列，签到完重读 state 即可）。
- 旧路由 `POST /api/user/check-in` 在本段删除前，**共用同一个 service 方法**，所以它也一起修好了。

### 3.3 `GET /api/v1/me/moemoepoint-entries` → `List[MoemoepointEntry]`（游标）

```
MoemoepointEntry  object="moemoepoint_entry", id, delta, reason, ref,
                  created_at, source
```

- 参数：`cursor`、`limit`（1–50，默认 20）、`reason`（可选）。
  - `limit` 上限是 **50** 不是 100：OAuth `/users/{id}/moemoepoint/log` 的上限就是 50（旧代码 `limit > 50 → 20`）。`> 50` → `400 LIMIT_TOO_LARGE`。这是 K11「1–100」的具名偏离，理由是上游上限。
  - `reason`：开放词表（OAuth 会加新 reason），`pattern ^[a-z][a-z0-9_]{0,39}$`。形状非法 → `400 INVALID_PARAMETER`（huma 对 query 参数的校验失败在本仓映射成 400，与 `page_test.go` 一致）；形状合法但上游不认 → 上游回 400 → 我们回 `400 INVALID_PARAMETER`（`parameter: reason`）。**不再**把上游 400 说成我们的 500。
- 游标：内部是 OAuth 的 `before_id`，编码成 `cur_…`，指纹绑定 `(user_id, reason)`。换了 `reason` 用旧游标 → `400 INVALID_CURSOR`。末页（上游 `has_more = false`）省略 `next_cursor`。
- `delta`：integer，**无 minimum**（扣分是负数）。`reason`：开放枚举字符串，maxLength 40。`ref`：maxLength 80，可为空串。
- `source`：封闭枚举 `this_site`（`source_app` 等于本站 OAuth client id，旧名 `is_local`）/ `account_center`（`source_app = "oauth"`：改名扣分、注册礼、管理员调整）/ `other_site`。原始 `source_app` 不下发——生产上它只有两个 hex client id 和 `oauth` 三种值，旧网页唯一真正显示过的标签就是 `oauth` →「账号中心」。初稿写成布尔 `is_from_this_site`，把这个标签弄丢了，网页验收时发现、改成枚举。
- 上游不可达 → `503 SERVICE_UNAVAILABLE`。
- 网页 `MoemoepointLog.vue` 把 `content_removed` 显示成「被移除」的改写逻辑保留（见 `kungal-moemoepoint-debit-reason`：这是 infra 唯一暴露的扣分理由）。

### 3.4 通知偏好——移到 U2

2026-09-22 与 M 轨协调：v1 的通知类型 token 由 M 轨新建的 `internal/message/notifytype` 定义（18 个 token + 伪静音键 `chat`，其中 13 个与库值不同名，如 `favorite` → `favorited`、`solution` → `best_answer_chosen`；全表在 M 的 `docs/proj/api-v1/waves/m-message.md` §3.3）。线上的静音键必须是这些 v1 token，存储仍是库值、经 `notifytype` 互译。

`notifytype` 在 M 合并之前不在 master 上，所以 `GET/PUT /api/v1/me/notification-preferences` 从 U1 移出，**在 U2 里基于 `notifytype` 实现**。U2 契约沿用这里原先定下的规则：未知 token → `422 VALIDATION_FAILED` / `UNKNOWN_VALUE`（`pointer /muted_types/<i>`，`params.allowed`）；GET 读失败 → 500；PUT 先 `Ensure`；`maxItems 64`、`uniqueItems`。

### 3.5 云端偏好 `GET/PUT /api/v1/me/preferences`

```
Preferences  object="preferences", doc: object, version (integer ≥0), written_at (date-time | null)
```

- 命名空间由服务端定（本站 OAuth client id），客户端**不可**指定——否则能摸到跨站的 `global` 文档（旧 DTO 上那条注释的理由原样保留）。
- `doc` 是自由 JSON 对象（`map[string]any`），不做 schema 约束；上限 64 KB 由上游判。
- 响应头 `ETag: "<version>"`。PUT 接受可选 `If-Match: "<version>"`，原样转给上游。
- 错误：
  - 上游 18006 → `412 PRECONDITION_FAILED`（复用 infra platform 码，新加进本仓注册表）；
  - 18001（令牌缺 `preferences` scope）→ `403 SCOPE_REQUIRED`；
  - 18005（超 64 KB）→ `422 VALIDATION_FAILED`，`pointer /doc`、`reason TOO_LONG`、`params.max_length = 65536`；
  - 18004（`doc` 不是对象）在我们这层 schema 已挡掉；
  - 上游 400/7（`If-Match` 不是版本号）→ `400 INVALID_PARAMETER`，`header: If-Match`；
  - 其它上游失败 → `503`。
- 网页原来靠旧体码判「云端不可用」「冲突」，改成判 `code === 'SCOPE_REQUIRED'` / `status === 412`。

### 3.6 `PUT /api/v1/me/nsfw-display`

```
请求  { nsfw_display: "hide" | "blur" | "show" }   封闭枚举
响应  NsfwDisplay  object="nsfw_display", nsfw_display
```

- 成功后照旧：写回 Redis 会话的内容立场（`middleware.SetSessionContentStance`，只对 cookie 会话；Bearer 没有会话可写）、`userClient.Invalidate`。写回失败只 warn（旧行为，理由同旧注释：只是晚一次刷新生效）。
- 不再下发 `adult_confirmed`（2026-09-23 退役，恒真）。写回会话时传 `true`。
- 上游 18001（令牌缺 `preferences` scope，这个 scope 同时覆盖这次账号级写）→ `403 SCOPE_REQUIRED`，网页据此静默降级，与云端偏好同一条规则（初稿漏了，验收时补上）；18007 在我们这层 schema 已挡掉；其它上游失败 → `503`。

### 3.7 `GET /api/v1/users?q=` → `List[UserRef]`（不分页）

- `q` **必填**，`minLength 1`、`maxLength 64`；**去空白后为空 → `422 VALIDATION_FAILED`**（`parameter q`、`REQUIRED`）。旧实现空 `q` 回 `[]`，把「你没输入」和「没人叫这个名字」混在一起。
- `limit` 1–20，默认 8，`> 20` → `400 LIMIT_TOO_LARGE`。
- **不分页**：无 `cursor`，无 `next_cursor`，响应仍是 `{object:"list", items}`。
- 过滤掉 `status != 0` 的用户（旧行为）。
- 上游失败 → `503`。
- **修 bug（普查 #3）**：`userclient.SearchUsers` **不再写 `c.hot`**。`/users/search` 的响应没有 `site_roles`，写进热缓存会让命中者的站点角色在全站展示层消失 10 分钟、`is_creator` 变 false；而且契约本来就说搜索结果不应缓存（`docs/oauth/03-cross-service.md:124`）。
- 这是 `/users` 集合的第一个过滤器；U2 会在同一个操作上加 `ids=`（批量引用）。那时 `q` 与 `ids` 互斥、恰好一个必填，**本段不预留 `ids`**。

### 3.8 `PATCH /api/v1/me/profile` → `MyProfile`

```
请求  { name?: string|null (1..17), bio?: string|null (0..107) }   至少一个字段非 null
响应  MyProfile  object="user", id, name, avatar: Image|null, bio
```

- 两个字段都缺（或都是 null）→ `422 VALIDATION_FAILED`（`pointer ""`，`REQUIRED`）。字段可为 null 是 G8 的要求：`name` 在 `UserRef` 里是 `string|null`，同名必须同型；null 与缺席同义，都是「不改」。
- 响应是**本站定义的形状**，不再透传 OAuth 的 `UserResponse`（旧实现会把上游新加的任何字段——包括值为空串的 `email` 键——原样发给浏览器）。数据从上游 `PATCH /auth/me` 的返回里取 `id / name / avatar_image_hash / avatar / bio` 映射；头像按 `Image` 规则解析（hash 优先）。
- 错误映射（上游码 → v1）：
  - 10007 名字重复 → **`409 USERNAME_TAKEN`**（新 kungal 码）；
  - 16006 余额不足以改名 → `403 MOEMOEPOINT_INSUFFICIENT`，**不带** `required` 扩展（价格在 OAuth 配置中心，论坛不知道）；
  - 7 字段约束 → `422 VALIDATION_FAILED`（我们这层 schema 已挡掉长度；上游的字符白名单只有它知道，这时 `pointer /name`、`reason INVALID_FORMAT`）；
  - 10001–10003 → `401 INVALID_CREDENTIAL`；其它 → `503`。
- 成功后：`userClient.Invalidate(id)`；**改了名**则额外 `GetMoemoepoint` 回源一次并写缓存列（修普查 §4.3：改名扣了 17 分，顶栏余额却一直不动）。这次回源失败只 warn——改名本身已经成功。

### 3.9 `PUT /api/v1/me/avatar` → `Image`

- 请求体 `multipart/form-data`，字段 `file`，必须是 `image/*`，≤ 4 MiB。
  - 缺 `file` → `422 VALIDATION_FAILED`（`pointer /file`、`REQUIRED`）；
  - MIME 不是 `image/*` → `415 UNSUPPORTED_MEDIA_TYPE`；
  - 超 4 MiB → `422 VALIDATION_FAILED`（`pointer /file`、`TOO_LONG`、`params.max_length = 4194304`）。
- 转发上游 `POST /auth/me/avatar`：**论坛自己重建 multipart**（字段 `file`、文件名固定 `avatar`、MIME 用我们判定过的值），不再把客户端的 `Content-Type` 头原样转发（普查 bug #17）。
- 响应：`Image`（`url` / `hash` / `width` / `height` / `thumbhash: null` / `sexual: null`），取上游返回的 `hash` / `url` / `width` / `height`。不透传 `variant_urls` / `deduplicated`。
- `PUT` 而不是 `POST`：替换「我的头像」这个槽位，天然幂等（同一张图 image_service 去重）。
- 成功后 `userClient.Invalidate(id)`。上游失败 → `503`。

### 3.10 创作者 `GET /api/v1/me/creator-status` 与 `POST /api/v1/me/creator-applications`

```
CreatorStatus  object="creator_status", is_creator,
               eligibility: { is_eligible, merged_pr_count, published_galgame_count,
                              long_review_count, moemoepoint,
                              required_merged_pr_count, required_published_galgame_count,
                              required_long_review_count, required_moemoepoint },
               application: CreatorApplication | null
CreatorApplication  object="creator_application", id, state, statement,
                    decline_reason (string|null), created_at, reviewed_at (date-time|null)
```

- `state`：封闭枚举 `pending` / `approved` / `declined`（上游 `status` 字段；取值以 `docs/oauth/08-creator-applications.md` 为准，实现时核对，出现第四个值就回 500 并在报告里写明——**不许**静默当成其中一个）。
- `reviews_100` → `long_review_count`：实现时核实，它数的是**简评不少于 100 字**的评分（`rating_repo.go` 的 `char_length(short_summary) >= 100`），不是满分评分。契约初稿按名字猜成了 `full_score_review_count`，验收时更正。
- `moemoepoint`（资格里的）**回源 OAuth**，失败 → `503`（旧实现 `moe, _ :=` 吞成 0，够格的人被判不够格）。
- `POST /me/creator-applications`：请求 `{ statement?: string (0..500) }`，`201 Created` + `Location: /api/v1/me/creator-status` + `CreatorApplication`。`Idempotency-Key` 可选。
  - 不够格 → **`403 CREATOR_INELIGIBLE`**（新 kungal 码）；
  - 上游 17001 已是创作者 → `409 INVALID_STATE_TRANSITION`（复用 infra `me` 域码）；
  - 17002 已有待审申请 → `409 ALREADY_EXISTS`；
  - 17003 冷却期 → **`409 CREATOR_APPLICATION_COOLDOWN`**（新 kungal 码）；
  - 其它上游失败 → `503`。
- 上游中文 `message` **不再**透传给用户（K8）。

## 4. 调用上游需要的令牌

§3.5–§3.10 要用调用者自己的 OAuth access token。v1 handler 拿到的是 `context.Context`，令牌在 Fiber locals 里（`middleware.GetAccessToken`）。

**不改 `internal/apiv1/**`**（共享地基）。做法：在本域注册时给这几个操作挂 **operation 级 huma middleware**（`huma.Operation.Middlewares`），它在 API 级身份中间件之后运行，用 `humafiber.Unwrap` 取出令牌与会话 cookie，`huma.WithValue` 放进 ctx。令牌为空（理论上 required 档不会发生）→ `401 INVALID_CREDENTIAL`。

## 5. 错误码

新增 **6 个**，三处一译缺一不可（`registry.go` 常量 + `Codes`、`registry_test.go` 的 `requiredCodes`、`zh-CN/problem.json`）：

| 码 | 域 | 状态 | 用处 |
|---|---|---|---|
| `ALREADY_EXISTS` | me（复用 infra，type URI 逐字同 infra） | 409 | 今天已签到；已有待审创作者申请 |
| `INVALID_STATE_TRANSITION` | me（复用 infra） | 409 | 已经是创作者还申请 |
| `PRECONDITION_FAILED` | platform（复用 infra） | 412 | 云端偏好 `If-Match` 不符 |
| `USERNAME_TAKEN` | kungal | 409 | 改名撞名 |
| `CREATOR_INELIGIBLE` | kungal | 403 | 不满足创作者申请条件 |
| `CREATOR_APPLICATION_COOLDOWN` | kungal | 409 | 被拒后冷却期未过 |

新增 `kungal` 码前已查 infra 注册表：无同名同义码。`INVALID_STATE_TRANSITION` 与 `ALREADY_EXISTS` 是 01 §2 K4 点名可复用的。如果本仓注册表里还没有 `me` 域常量，加上，type URI 形如 `https://developer.nextmoe.dev/problems/me/already-exists`。

## 6. 新 K 决定

- **K22**（§3.2）：签到的日期、幂等键、奖励都由北京日决定；奖励是键的确定函数。
- **K23**：OAuth 代理面不透传上游的响应体、体码与中文句子。每个上游码在本文逐条映射到 v1 码；**没列出的上游失败一律 `503`**，完整上游错误只进日志。

## 7. 网页

逐个调用点切到生成的类型化客户端（照 W5b / T2 的样子），手写类型删掉：

| 文件 | 改什么 |
|---|---|
| `components/kun/top-bar/Nav.vue` | `/user/status` → `GET /me`，字段按 §3.1 改名 |
| `plugins/validate-session.client.ts` | 同上 |
| **`utils/kunFetch.ts` 的会话探针** | 它自己 raw `$fetch` 打 `/api/user/status` 并判 `code !== 0`。改打 `/api/v1/me`：**2xx = 会话活着、401 = 死了**，其它（网络错、5xx）= 不下结论（维持现有「不确定就别登出」的语义）。**必须与删旧路由同一个 PR**，否则探针打到 401 的旧路由上会把所有人判成已登出（普查 bug #11） |
| `components/kun/top-bar/UserInfo.vue` | 签到 → `POST /me/check-ins`；「今天已签」按 `code === 'ALREADY_EXISTS'` 处理 |
| `components/kun/top-bar/MoemoepointLog.vue` | 翻页从 `before_id` / `has_more` 改成 `cursor` / `next_cursor`；`is_local` → `is_from_this_site` |
| `composables/useCloudPreferences.ts` | 路径换 v1；冲突改判 412、不可用改判 `SCOPE_REQUIRED`；`If-Match` 照发 |
| `composables/useContentStance.ts` | `/user/nsfw` → `PUT /me/nsfw-display`，不再读 `adult_confirmed` |
| `composables/useKunEditorAdapters.ts`、`components/edit/topic/AccessUserPicker.vue` | `/user/search` → `GET /users?q=`，读 `items`；id 是字符串 |
| `components/user/setting/Bio.vue`、`Username.vue` | → `PATCH /me/profile`；撞名按 `USERNAME_TAKEN`、余额不足按 `MOEMOEPOINT_INSUFFICIENT` |
| `components/user/setting/Avatar.vue` | → `PUT /me/avatar`（`FormData` 字段 `file`），读 `url` |
| `components/kun/top-bar/CreatorApply.vue` | → `GET /me/creator-status` / `POST /me/creator-applications`，字段按 §3.10 改名，`status` → `state` |

新码的 `zh-CN` 译文进 `apps/web/i18n/locales/zh-CN/problem.json`。

## 8. 删旧路由

12 条全删，连同 handler 方法、只被它们用到的 DTO、手写 TS 类型。`UserHandler` 的 struct 本身**留着**（U2/U3 的 9+2 条还挂在它上面）；`ProfileHandler`、`ContentPrefsHandler` 整个删；`CreatorHandler` 整个删（`CreatorService` 留，v1 共用）。`rg` 证明零调用方（含 `apps/web/server/`、`../kungal-apps`）。`deadcode -test ./...` 跑到不动点。`routes.golden` 重生成，`legacy_route_baseline` 下调 12。

`docs/proj/app-direct-api.md` 里 `/api/user/status` 一行改成 v1 路径（通知偏好那行随 U2 改），`CHANGELOG.md` 记一条（App 可见）。

## 9. 变异清单（先于实现提交）

每条一行语义改动，每条都必须让某个测试变红。编译不过的换等价破坏。

| # | 破坏 | 应当变红的性质 |
|---|---|---|
| 1 | 签到日期改回 `time.Now()`（进程本地时区，测试里设 `TZ=UTC`、时钟钉在北京 01:00） | `check_in_date` 是北京日，且键里的日期是北京日 |
| 2 | `delta` 改回 `rand.IntN(8)` | 同一用户同一天两次计算得到同一个 `delta`（重放同体） |
| 3 | `AwardSync` 失败时不还原闸门 | 上游失败 → 503 之后 `has_checked_in_today` 仍为 false，且能立刻重签成功 |
| 4 | 闸门 `WHERE daily_check_in = 0` 去掉 | 第二次签到 → `409 ALREADY_EXISTS` |
| 5 | `GET /me` 的未读计数失败改回当 0 | 计数查询失败 → 500 |
| 5b | `GET /me` 的未读通知不排除已静音类型 | 只有已静音类型的未读通知时 `has_unread_messages = false` |
| 6 | 萌萌点流水游标指纹去掉 `reason` | 换 `reason` 复用游标 → `INVALID_CURSOR` |
| 7 | 上游 `has_more = false` 时仍然下发 `next_cursor` | 末页省略 `next_cursor` |
| 8 | 流水 `limit` 上限从 50 改成 100 | `limit=51` → `LIMIT_TOO_LARGE` |
| ~~9~~ | ~~通知偏好未知键改回静默丢弃~~ | 随通知偏好移到 U2 |
| 10 | `SearchUsers` 恢复写 `c.hot` | 一次 `/users?q=` 之后 `userClient.User(id)` 仍会回源（热缓存里没有这个人） |
| 11 | `/users?q=` 空白 `q` 回空列表而不是 422 | `q="  "` → 422 |
| 12 | 上游 10007 映射成 `503` | 撞名 → `409 USERNAME_TAKEN` |
| 13 | 上游 18006 映射成 `503` | `If-Match` 不符 → `412 PRECONDITION_FAILED` |
| 14 | 创作者资格里的 `GetMoemoepoint` 错误改回吞成 0 | 上游余额查询失败 → 503 |
| 15 | 头像不校验 MIME | `text/plain` 文件 → 415 |

上游（OAuth）在测试里用 `httptest.Server` 假冒，不打真服务。

## 10. 验收时对契约的更正（2026-09-23）

契约门在实现后抓出的，全部改在上文对应位置，这里留底：

| 原契约 | 改为 | 原因 |
|---|---|---|
| `Preferences.updated_at`（可空） | `written_at` | G8：全 spec 里 `updated_at` 都是非空时间戳，同名必须同型 |
| `CreatorApplication.message` / 请求体 `message` | `statement` | G6：`message` 是成功响应里禁用的信封键 |
| `PATCH /me/profile` 的 `name` / `bio` 是 `string` | `string\|null` | G8：见 §3.8 |
| `full_score_review_count` | `long_review_count` | 见 §3.10，字段数的是长评，不是满分评 |
| `reason` 格式错 → 422 | → 400 `INVALID_PARAMETER` | 本仓 query 校验的既有映射 |
| `PUT /me/avatar` 用 huma 的 `MultipartFormFiles` | 请求体是 `multipart.Form`，schema 手写在操作上 | `MultipartFormFiles` 会把 `huma.FormFile` 的 Go 字段（`IsSet`/`Size`…）发成组件，G2/G14/F1 全红 |
| `/me` 的上游错误判定 | service 层哨兵错误（`service.ErrUpstream`、`CreatorService.ErrAccountUnavailable`） | 初版按错误信息里的 `"userclient:"` 前缀判断，改一句文案就会把 503 变成 500 |
| `MoemoepointEntry.is_from_this_site`（布尔） | `source`（`this_site`/`account_center`/`other_site`） | 见 §3.3：布尔丢了「账号中心」这个来源 |
| 上游 `created_at` 解析失败静默发 `0001-01-01` | `500` | 同一类「吞错发零值」的毛病 |
| 成人向显示写回会话 | 仅 cookie 会话，**Bearer 请求不写回**，即使它碰巧带着 cookie | K2：带 `Authorization` 的请求只认 Bearer |

## 11. 迁移

**无**。本段不改表结构。
