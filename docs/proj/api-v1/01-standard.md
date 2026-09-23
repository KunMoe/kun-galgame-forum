# 01 · 标准

规范用语同 infra：**必须 / 不得** = CI 门或测试钉死；**应当 / 不应** = 偏离要在 K 表里写理由；**可以** = 自由。

## §0 infra 条款在论坛的适用性

infra `refs/api-v2` 的公理 A1–A14、黑名单 B1–B37、门 G1–G17 默认全部适用。下表只列**不适用或改写**的条款。

| infra 条款 | 论坛 | 理由 |
|---|---|---|
| A7 / 05 §5 `view=` · `include=` · `fields=` | **改写**，见 K10 | 一方 API，静态形状比投影参数更值钱 |
| A9 每个族必须有 `ids=` 批量读 | **改写为「应当」**：出现真实消费者时再加 | A5：不发无法兑现的承诺。加批量读是加法 |
| A10 / B34 / G11 同 URL 同字节、边缘缓存探针 | **不适用** | v1 是个性化的一方面，全部 `no-store`（K15）。随凭证变化的字段只能出现在 `viewer` 块里（K9） |
| A13 一个前缀一个凭证体系 | **改写**，见 K2 | 同一个主体（终端用户）有两种载体：网页 cookie 会话、App Bearer |
| B17 一个 API 只有一种分页风格 | **改写**，见 K11 | 恰好两种，每个集合在 spec 里声明一种，客户端不选 |
| B25 默认不发 `total` | 游标集合适用；页码集合**必发**带关系的 `total`（K11） | 分页器要画出最后一页 |
| 02 §5.2 / §6 `ETag`、`If-None-Match`、`Vary` | **暂不做** | K15。真有需要时按 infra 02 §6 加，是加法 |
| 02 §5.3 / §7 `RateLimit` 头 | **暂不适用** | 论坛自己没有限流器；上游（community）回 429 时透传成 `RATE_LIMITED` + `Retry-After` |
| 02 §8.2 `If-Match` 强制于审核类写 | **暂不做** | 论坛的审核写并发极低；进 W4 复议 |
| `nmk_` 应用密钥、MCP（G10） | **不适用** | 这个面没有应用密钥；将来开放给第三方是另一个面（infra 开发者平台 §16/§17） |
| 07 §3.0 preview / GA | **适用**，GA 改名「稳定」 | 稳定由用户宣布，前提是 App 首个公开版发布。见 [02 §4](02-governance.md) |

## §1 面与凭证

**K1 · 前缀 `/api/v1`，一份 OpenAPI。** 全部 v1 操作挂在 `/api/v1/**`，由同一个 huma API 注册，产出一份 spec。路径段用复数名词、kebab-case（`/topics/{topic_id}/replies`）。路径参数名与响应里的字段名一致（`{topic_id}` ↔ `topic_id`）。

**K2 · 一个主体，两种载体。** 这个面只有一种主体：终端用户（或匿名）。

- 网页：`kungal_session` cookie（BFF 不透明会话，`middleware/auth.go`）。
- App：`Authorization: Bearer <OP access token>`（`middleware/bearer.go`，`KUN_BEARER_CLIENT_IDS` 白名单）。
- 带了 `Authorization` 头就**只**走 Bearer，cookie 被忽略；不存在「两个都认」。
- Bearer 请求**永不**持有 staff 能力（`bearer_guard_test.go` 的规则原样适用于 v1）。

**K3 · 可选鉴权的读面：cookie 失效降级为匿名，Bearer 失效拒绝。**

- 操作分三档：`public`（不看凭证）、`optional`（有就用）、`required`。在 spec 里用 `security` 表达，`optional` 额外带一个空的安全要求项。
- `optional` 档：cookie 会话不存在或已过期 → 按匿名处理（cookie 是浏览器自动带的环境凭证，陈旧 cookie 不能让公开页面 401）；**Bearer 无效 → `401 INVALID_CREDENTIAL`**（Bearer 是客户端主动出示的，RFC 6750 要求拒绝，App 据此刷新令牌）。
- `required` 档：没有任何凭证 → `401 MISSING_CREDENTIAL`；有但无效 → `401 INVALID_CREDENTIAL`。
- 会话存储（Redis）本身出错 → `503 SERVICE_UNAVAILABLE`，**不得**当作匿名（现行 `OptionalAuth` 把 Redis 故障和「没有会话」混为一谈）。
- 每个 401 带 `WWW-Authenticate: Bearer realm="kungal"`（RFC 9110 §15.5.2）。
- 封禁用户 → `403 ACCOUNT_BANNED`；OAuth scope 不足 → `403 SCOPE_REQUIRED`。

## §2 错误与 i18n

**形状**：逐字采用 infra 02 §3 与 10：`application/problem+json`，成员 `type` `title` `status` `detail` `instance` `code` `request_id` `errors[]`。`errors[]` 每项恰好一个位置成员（`pointer` / `parameter` / `header`）、一个 `reason`、英文 `detail`。

**K4 · 注册表与域。**

- 注册表是 Go 代码（`pkg/problem`）里被 handler 引用的常量集合，**唯一来源**。`openapi/problems.json` 与 `GET /api/v1/problems` 都由它生成。
- 通用失败**复用 infra 的 code，且 type URI 与 infra 逐字相同**（`https://developer.nextmoe.dev/problems/platform/not-found`）。code 在整个平台唯一，一个 code 只有一个 type URI（infra 10 §5）。
  - 摸鱼的 `/v2/moyu` 把复用的 platform code 挂在 `problems/moyu/…` 下，违反这一条。那是摸鱼的问题，不要照抄。
- 论坛专属失败进 **`kungal` 域**：`https://developer.nextmoe.dev/problems/kungal/{kebab-code}`。构词、长度、语法按 infra 10 §2（`<主语>_<判决>`，UPPER_SNAKE，≤63，无服务名前缀，无状态码）。
- infra 其它域里含义相同的 code 直接复用，type URI 用 infra 的域：
  - `PERMISSION_REQUIRED`（moderation）：论坛权限键 `perm.*` 不足一律用它；
  - `INVALID_STATE_TRANSITION` 与 `ALREADY_EXISTS`（me）；
  - `DUPLICATE_SUSPECTS`（me）。
- **新增 `kungal` 码前**，先查 infra 注册表 `apps/api/internal/platform/apiv2/problem/registry.go`，确认没有同名或同义的码。

**K5 · 本地化参数 `params`（infra 的缺口，论坛补上并提给 infra）。**

infra 的 `errors[]` 只有英文 `detail`。客户端要本地化「标题最多 233 个字」这种文案，就只能去解析英文，这正是用户说的「看不懂英文报错」的来源。所以：

- `errors[]` 每项**可以**带 `params` 对象，键由 `reason` 决定，注册表写死：

  | `reason` | `params` |
  |---|---|
  | `TOO_LONG` | `{ "max_length": int }` |
  | `TOO_SHORT` | `{ "min_length": int }` |
  | `OUT_OF_RANGE` | `{ "minimum"?: number, "maximum"?: number }` |
  | `TOO_MANY_ITEMS` | `{ "max_items": int }` |
  | `TOO_FEW_ITEMS` | `{ "min_items": int }` |
  | `UNKNOWN_VALUE` | `{ "allowed": [string] }`（封闭词表才发） |
  | 其余 | 不发 |

- `TOO_FEW_ITEMS` 是论坛补的 reason，infra 的 13 条里没有「数组项数不足」。论坛的请求体有 `minItems` 约束（如话题至少选 1 个版块），所以需要它。和 `params` 一起提给 infra。
- `params` 是有类型的对象（`pkg/problem.FieldParams`，全部指针字段），不是自由 map。

- 顶层 problem 的扩展成员同理：每个 code 在注册表里声明自己带哪些扩展成员及类型（infra 已有先例：`ENTITY_MERGED` 带 `current_id`、`DUPLICATE_SUSPECTS` 带 `suspects[]`）。没声明的扩展成员不得出现。
- huma 自带校验产生的错误**必须**映射到精确的 `reason` 与 `params`。infra 的 `fieldFromHuma` 把它们一律映射成 `INVALID_FORMAT`，论坛不照抄。映射表必须有测试覆盖每一条 v1 schema 可能触发的 huma 校验消息。

**K6 · 新增的两个 code。**

| code | 域 | status | 什么时候 |
|---|---|---|---|
| `ACCOUNT_BANNED` | kungal | 403 | 当前用户已被封禁（取代体码 `234`） |
| `IDEMPOTENCY_REQUEST_IN_PROGRESS` | kungal | 409 | 同一 `Idempotency-Key` 的首个请求还在处理（取代 `237`）。infra 没有并发态，这条应当提给 infra 进 platform 域 |

**K7 · 旧体码的去向**（只在 v1 生效，旧路由不变）：

| 旧 | v1 |
|---|---|
| `205`（HTTP 401） | `401 MISSING_CREDENTIAL` / `401 INVALID_CREDENTIAL` |
| `233`（400/403/404/500 混用） | 每个调用点按语义选码，**不得**有兜底的「业务错误」码：输入 → `INVALID_PARAMETER` / `UNKNOWN_ENUM_VALUE` / `VALIDATION_FAILED`（带 `errors[]`）；不存在或不可见 → `NOT_FOUND`；权限键不足 → `PERMISSION_REQUIRED`；状态机 → `INVALID_STATE_TRANSITION`；领域规则 → 该域的具名 `kungal` 码；我们的 bug → `INTERNAL_ERROR`；上游不可用 → `SERVICE_UNAVAILABLE` |
| `234` | `403 ACCOUNT_BANNED` |
| `235` | `403 SCOPE_REQUIRED` |
| `236` | `409 DUPLICATE_SUSPECTS` |
| `237` | `409 IDEMPOTENCY_REQUEST_IN_PROGRESS` |
| `238`（HTTP 422） | `409 IDEMPOTENCY_KEY_REUSED`。**status 变了**：跟 infra 注册表走，一码一 status |

**K8 · 客户端本地化规则（网页与 App 相同）。**

1. 面向用户的错误文案**只**由客户端从 `code`、`errors[].reason` 与 `params` 产出。`title`、`detail` **不得**显示给终端用户：它们是英文诊断信息，只进日志与开发者工具。
2. 客户端必须为未知 `code` 准备按 `status` 的兜底文案（infra 客户端契约第三句）。
3. 网页的译文目录放在 `apps/web/i18n/locales/<locale>/problem.json`，现在只有 `zh-CN`。格式用 vue-i18n 的具名插值（`{max_length}`），将来接回 `@nuxtjs/i18n` 时原样作为消息源，不用改写。
4. 译文覆盖是 CI 门（F2）：注册表里每个 `code` 与 `reason` 都必须有 `zh-CN` 译文，目录里也不得有注册表之外的键。
5. 500 的 `detail` 不得泄漏内部信息（SQL、堆栈、上游响应体）。完整错误链只进服务端日志，并带上 `request_id`。

## §3 表示层

逐字采用 infra 04 的：`object` 判别字段（集合是 `"list"`）、snake_case、**全部 id 是字符串**（`pattern ^[0-9]+$`，`maxLength 20`）、时间戳 RFC 3339 UTC 秒精度 `Z` 结尾、日期 `YYYY-MM-DD`、枚举一律字符串并标注开放/封闭、数组与 map 永不 `null`、`omitempty` 只在指针上、每个字符串有 `maxLength`、每个数值有 `minimum`。

**K9 · 随查看者变化的字段一律放进 `viewer`。** 资源本身的字段对所有查看者相同。「我点过赞吗」「我能编辑吗」这类字段放在同一层级的 `viewer` 对象里；匿名时 `viewer` 是 `null`。

```json
{
  "object": "topic", "id": "4121", "like_count": 12,
  "viewer": { "has_liked": true, "has_favorited": false, "can_edit": true }
}
```

- `can_*` 由服务端按权限闸计算，App 与网页**不再**各自镜像一份权限判断。这是 `perm` 三处镜像（Go / `useCan.ts` / `permission.ts`）的长期出路。
- Bearer 请求的 `can_*` 同样不含任何 staff 能力（K2）。
- 嵌套条目上的查看者状态同样用 `viewer`，例如表情汇总项 `{ "reaction": "…", "count": 3, "viewer": { "has_reacted": true } }`。

**K10 · 形状按操作静态确定，不做投影参数。** 列表条目用摘要形状（如 `TopicSummary`），详情用完整形状（`Topic`），两者都是 `object: "topic"`，重叠字段同名同型（G8 钉死）。不提供 `view=` / `include=` / `fields=`。

理由：openapi-typescript 与 tonik 都无法按查询参数的取值收窄响应类型，投影参数会让两端的每个字段都变成「可能缺席」。一方客户端要的是强类型。将来真有需要，加参数是加法，默认形状不变。

**图片**：一个类型，处处如此（B10）。

```json
{ "url": "https://…", "hash": "9f3c…", "width": 800, "height": 1131, "thumbhash": "3PcN…", "sexual": "safe" }
```

- 键名与 infra `repr.Image` 的同名键逐字一致，App 用一个解码器同时解两边。
- `sexual`：`safe` / `suggestive` / `explicit` / `null`（未判定）。图床的 0/1/2 映射过来；缺席是 `null`，**不是** `safe`。
- 尺寸未知发 `null`。
- 不发 `violence` 与 `source`：图床不产这两项（A5）。

**用户引用**：`{ "object": "user", "id": "3", "name": "…", "avatar": Image | null }`。

- 用户的主标签叫 `name`：它就是 OIDC 标准声明 `name`，也是 OAuth `/users/batch` 返回的键名。
- infra 04 §2 的 `display_name` 规则管的是 catalog 实体，用户不是 catalog 实体。

**命名规则（F1 门钉死）**：

| 规则 | 例 |
|---|---|
| 布尔以 `is_` / `has_` / `can_` 开头 | `is_nsfw`、`has_best_answer`、`viewer.can_edit` |
| 时间戳以 `_at` 结尾且 `format: date-time`；反之亦然 | `created_at`、`last_activity_at` |
| 日期以 `_date` 结尾且 `format: date` | `release_date` |
| 计数以 `_count` 结尾，整数，`minimum: 0`；时间窗计数写成 `<名词>_<窗口>_count` | `reply_count`、`view_7d_count` |
| 引用 id 以 `_id` / `_ids` 结尾且为字符串 | `topic_id`、`user_ids` |
| 外部命名空间的 id 名字里说清是谁的 | galgame 作品的 id 叫 `work_id`：G0（2026-09-23）之后论坛作品 id 就是 catalog 的 `catalog_work.id`，同一个数，不存在「论坛 id」与「catalog id」两套（根 CLAUDE.md 铁律 3）。其它外部 id 仍要写明归属，如 `vndb_id` |
| 数组用复数 | `sections`、`comments`、`cover_images` |
| Markdown 源叫 `content_markdown`，结构化正文叫 `content` | 两者同名不同型就违反 A4 |

**枚举值**：封闭枚举的取值是小写 ASCII snake_case。例外只走具名清单，加一项要写理由：

| 词表 | 取值形如 | 理由 |
|---|---|---|
| 话题版块 `sections` / `section` | `g-walkthrough` | 它们同时是 `/section/{key}` 页面的 URL 段，改拼写就改了公开 URL。单数的 `section` 是同一个词表（`listTopics` 的过滤参数、`Section` 对象的键），X1b 加入 |
| 语言标签（`language`；工具集 `interface_language`；资源 `languages` / `resource_language` / `resource_languages`） | `zh-cn` | 小写 BCP 47 标签：存储、写路径与网页词表都用它，WS 的 `language` 已按同样的标签下发；改成 `zh_cn` 就要在每次读写上永久翻译。F1 对这几个属性改查「小写 BCP 47 或 `other` / `others`」 |

**禁用名**（任何 property 或参数名都不得是）：`kind`、`gid`、`tid`、`uid`、`rid`、`pid`、`cid`、`created`、`updated`、`edited`、`view`、`status_update_time`、`user`，以及成功响应顶层的 `code` / `message` / `data` / `success`。

**命名对照表**（迁移时逐条执行，新增条目随波追加）：

| 旧 | v1 |
|---|---|
| `gid` / `galgame_id`、路径 `:gid` | `work_id`、`{work_id}`；集合是 `/works`（例：`/api/v1/works/{work_id}/moyu-patches`）。这一行原先写的是 `galgame_id` 与 `catalog_work_id`，G0 改号后作废 |
| `tid`、路径 `:tid` | `topic_id`、`{topic_id}` |
| `created` / `edited` / `updated` | `created_at` / `edited_at` / `updated_at` |
| `status_update_time` | `bumped_at`：顶帖时间。回复等互动会刷新它，但创建超过 3 个月的话题不再被顶（`model.BumpCutoff`），所以它不是「最后活跃时间」。命名同 Discourse |
| `upvote_time` | `upvoted_at` |
| `view` / `view_7d` / `view_30d` | `view_count` / `view_7d_count` / `view_30d_count` |
| `section`（数组） | `sections` |
| `comment`（数组） | `comments` |
| `is_nsfw_topic` | `is_nsfw` |
| `user`（话题、回复的作者） | `author`。指向人的字段按这个人在资源里的角色命名（`author`、`actor`、`follower`…），不叫 `user`：一条资源里常有好几个人，`user` 分不清是谁 |
| `status`（资源生命周期整数） | `state`（封闭字符串枚举，取值由普查决定）。不叫 `status`：Problem 的 `status` 是整数，同名不同型会被 G8 拦下；infra 的资源生命周期也叫 `state` |
| `content`（请求体里的 Markdown） | `content_markdown` |
| `cover_images` + `cover_image_meta`（hash 数组 + 旁挂 map） | `cover_images: [Image]` |
| `is_liked` / `is_favorited` / … | `viewer.has_liked` / `viewer.has_favorited` / … |

**论坛的偏好 cookie 不再是输入。** v1 不读 `KUNGalgameSettings`、不读 `X-Kungal-Nsfw`、不走 `NamePreference`。影响结果集的输入一律是显式查询参数（如 `include_nsfw=true`）。名字按 infra 04 §8 发全部（`display_name` + `latin` + `localized{}`），由客户端挑。

## §4 集合

**K11 · 恰好两种分页风格，按集合的性质选，每个集合在 spec 里声明一种（`x-pagination`），客户端不选。**

| 风格 | 用于 | 依据 |
|---|---|---|
| **游标（stream）** | 追加型、按时间或活跃度流动的集合：话题列表、回复、通知、私信、动态、我的 xxx | infra 05 §2 原样 |
| **页码（browse）** | 需要分页器的「可浏览结果集」：`/galgame` 浏览、搜索结果、排行、管理表格 | 见下 |

为什么不是全部游标：分页器要能「跳到第 N 页」「显示共 M 页」，而游标只能一页一页往后走。

业界对这个需求的做法是一致的：页码加**深度上限**，总数带**关系**。

- Elasticsearch：`from + size` 上限 10,000，`hits.total` 是 `{value, relation: "eq" | "gte"}`。
- GitHub Search API：只能取前 1,000 条。
- Algolia：`paginationLimitedTo`，默认 1,000。
- GitLab：列表默认 offset，大集合另给 keyset。

深翻页之所以慢、之所以在插入时错位，是因为 offset 没有上限。给了上限，这两个问题就有界了。infra B17 真正要防的是「五种分页 = 五套客户端代码」。这里恰好两种，每种一份实现，集合自己声明用哪种，这个意图仍然成立。

**游标集合**（逐字采用 infra 05 §1–§3）：

- 参数：`cursor`、`limit`（1–100，默认 20）。
- 响应：`{ "object": "list", "items": [...], "next_cursor": "cur_…" }`。末页**省略** `next_cursor`。
- 多取一行判断是否还有下一页；每个排序都带 `id` 作 tie-breaker。
- `limit` 超限 → `400 LIMIT_TOO_LARGE`（不 clamp）；游标坏了 → `400 INVALID_CURSOR`。
- 游标**绑定**生成它的排序与过滤条件；换了条件再用旧游标 → `400 INVALID_CURSOR`，不静默从头开始。
- `include_total=true` 才发 `total`，与 `items` 同一谓词。
- 双向浏览（回复「从第 N 楼开始、再往前加载」）用锚点过滤参数 + 反向排序 token 表达，仍是同一种游标，不引入 `prev_cursor`。具体参数在 W2 定。

**页码集合**：

- 参数：`page`（≥1，默认 1）、`limit`（1–100，默认由集合声明）。
- 深度上限：`page × limit ≤ 10000`。越界 → `400 INVALID_PARAMETER`，`errors: [{ "parameter": "page", "reason": "OUT_OF_RANGE", "params": { "maximum": <最大页> } }]`。
- 响应：`{ "object": "list", "items": [...], "total": 12345, "total_relation": "eq" }`。
  - `total_relation` ∈ `eq` / `gte`（封闭）。计数超过深度上限时可以只数到上限并发 `gte`，界面显示「10000+」。
- 不发 `next_cursor`，不发 `page_count`（客户端用 `total` 与 `limit` 算）。
- 这种风格**提议给 infra**：catalog `/v2/catalog/works` 只有游标，论坛 `/galgame` 的分页器因此没法建在 `/v2` 上。

**排序**：`sort=` 是每个集合各自声明的封闭枚举，token 形如 `<键>_<asc|desc>`（`last_activity_desc`），方向写进 token。未知 token → `400 UNKNOWN_SORT`，**不得**静默回落（现行 `topicOrderCol` 回落到 `created`）。

**过滤**：封闭词表的未知值 → `400 UNKNOWN_ENUM_VALUE`；缺席 = 不过滤（不需要 `all` 这种 token，A6）。布尔只收 `true` / `false`。未知的参数名忽略（infra 02 §4）。

## §5 写面

逐字采用 infra 06 §1。几条落到论坛的具体形态：

- 创建：`POST` → `201 Created` + `Location` + 完整资源体。
- 部分更新与状态迁移：`PATCH`。非法迁移 → `409 INVALID_STATE_TRANSITION`，`detail` 写明当前状态与允许的目标状态。
- 删除：`DELETE` → `204`，无 body。
- **切换类互动改成「置位 / 撤销」**：`PUT /topics/{topic_id}/like` 点赞，`DELETE /topics/{topic_id}/like` 取消。两者都天然幂等，重放不会翻转状态。现行 `PUT` 当切换用，违反 PUT 的幂等语义。
- **动词路径改成资源**（B7）。抽奖是典型：
  - 参与 = `POST /lotteries/{lottery_id}/entries`；
  - 退出 = `DELETE /lotteries/{lottery_id}/entries/me`；
  - 开奖 / 取消 = `PATCH /lotteries/{lottery_id}` 改状态。
  - 具体形态在 W5 定。
- 请求体里引用的 id 同样是字符串。请求体的 schema 同样受 G14 约束（`maxLength` 等），前端表单校验与之同源（W3 评估从 spec 生成 Zod）。

**K12 · 幂等键。**

- 全部 `POST` **必须**支持 `Idempotency-Key`。创建用户内容的 `POST`（话题、回复、评论）**必须携带**：缺席 → `400 INVALID_PARAMETER`，`errors: [{ "header": "Idempotency-Key", "reason": "REQUIRED" }]`。
  - 理由：App 在弱网下会重试。论坛现行实现已经这样要求，infra 只要求「支持」。
- 值：UUID（任意版本）或 ULID，不带前缀。格式错 → `400 INVALID_PARAMETER` + `reason: INVALID_FORMAT`。
- 作用域：`(主体, 操作, key)`，窗口 24 小时。
- **指纹是 `method + 路径 + 请求体`**，不只是请求体。同一个操作的不同目标（`/topics/1/upvotes` 与 `/topics/2/upvotes`）指纹不同，所以**客户端生成键时必须把目标算进去**；否则换一个目标复用同一个键，会撞 `409 IDEMPOTENCY_KEY_REUSED`，而且键只在成功后释放，一次失败就把整个窗口堵死 24 小时（2026-09-22 网页的推就是这么堵的）。
- 重放同 key 同请求 → 原响应 + `Idempotency-Replayed: true`（infra 的头名；现行 `Idempotent-Replayed` 是 Stripe 的拼法，v1 不用）。
- 同 key 不同请求 → `409 IDEMPOTENCY_KEY_REUSED`；首个请求仍在处理 → `409 IDEMPOTENCY_REQUEST_IN_PROGRESS`。
- 存储 2xx–4xx 的最终响应（与 infra 同）；5xx 与 429 释放键，允许重试。

**K16 · 查看者槽位的置位与撤销**（W4 定）。「我的赞」「我的收藏」「本话题的最佳答案」「置顶回复」是槽位，不是有自己身份的资源：

- `PUT` 置位、`DELETE` 撤销，都幂等：置已置的、撤未置的都是 200 且无副作用。
- 两者都回 200，响应体是客户端重画所需的状态（目标的 engagement 快照，或完整资源）。
- 删掉有自己身份的资源（回复、草稿）才是 204。
- `/topics/{topic_id}/reactions/{reaction}` 指调用者自己那条表情，凭证即主语，与抽奖的 `entries/me` 同理。
- 可重复、会扣费的互动（推）是 `POST` 建记录，必须带幂等键。

**K17 · 上游依赖不可用时，写面失败关闭**（2026-09-22 定）。v1 的每个话题/回复写与互动，都要先判定目标可见，而可见性包含「作者是否被封禁」，这个判据只有 OAuth 的 `/users/batch` 知道。

- 该调用失败 → `503 SERVICE_UNAVAILABLE`，写不发生。
- 这与旧写面不同：旧写面根本不查作者，旧读面用 `Hydrate` 吞掉错误、回占位符。取舍是明确的：**宁可暂时写不进去，也不要在判据未知时放行对封禁用户内容的互动**。
- `pkg/userclient` 有约 10 分钟的热缓存，所以上游短暂抖动通常不会穿透到用户。
- 读面不受此条约束：列表与详情在作者查不到时按不可渲染处理（`name: null`），不整条失败。

**K18 · `PATCH` 只对「本次提交的文本」跑内容检查**（2026-09-22 定）。标题与正文都没变时（只改分类、封面、NSFW 开关、访问范围），不跑 trust 检查、不重新入扫描队列。

- 旧 `PUT` 每次编辑都拿合并后的全文重跑一次，于是正文在发布之后才进禁用词表的话题，连分类都改不了。
- 理由：没有新文本就没有新内容可判。已存在的正文仍受后台扫描与举报处置覆盖。

**K19 · 自由文本的 `maxLength` 作用于原始值，去空白发生在其后**（2026-09-22 定）。schema 的长度上限由 huma 在进入 handler 之前执行，handler 再去首尾空白。

- 所以「30 个字符 + 首尾各一个空格」是 `422 TOO_LONG`，而不是先裁成 30 再通过。
- 客户端应当**按原始值计数**（网页的推备注就是裁到 30 个 rune，含空白）。
- 这条是实现与 W4 裁决文字的出入，取实现为准；若将来要改成「先去空白再判长度」，得同时抬高 schema 上限，属于契约变更。

**K20 · 评论正文是「受限」的内容文档：字段叫 `content`，但不跑 Markdown 解析**（2026-09-22 定）。K13 早就把评论算进正文文档的适用范围（03 §1 原文「话题、回复（以后还有评论）」），W2 把评论发成 `text: string` 是偏离；W5 归位成 `content: ContentDocument`。

- 但评论**不是 Markdown**，历史数据也从不是：生产 3162 条里 40 条含 `*`/`_`、3 条含 `**`、0 条列表、1 条行首 `#`。套完整 Markdown 管线会把这些当标记吃掉，是一次静默的改写。
- 所以评论的文档由**纯文本管线**产出，只可能出现这六种节点：`paragraph`、`text`、`break`（544 条评论有换行，必须保真）、`image`（`/image/<hash>` token 解析而来）、`mention` 与 `reply_reference`（行内 token）。不产出 `emphasis` / `strong` / `heading` / `list` / `code` / `link`。
- 节点词表与话题、回复**共用同一个封闭联合**，客户端不需要第二个渲染器；评论只是从不使用其中大部分成员。
- 写面收的是纯文本源，字段叫 `text`（不叫 `content_markdown`，因为它不是 Markdown）；`GET /comments/{id}/source` 回同一个 `text`。
- 理由：token 本来就在库里可能出现，而图片 GC 的 reference-ping 正是靠扫这些 token 存活；把它当纯字符串下发，界面上就是一行 `/image/9f3c…` 的路径。语义正确的做法是解析它，而不是承认这个洞。

## §6 正文文档（K13）

帖子、回复、评论等 Markdown 正文，v1 以**结构化节点树**下发，字段名 `content`。完整规格见 [03-content-doc.md](03-content-doc.md)（W1 定稿），这里只定原则：

- 节点词表以 **mdast**（unified / remark 的 Markdown AST 规范）为基础，含 GFM（表格、删除线、任务列表、脚注）与 math。kun-editor 的 Milkdown 内部就是 remark / mdast，编辑器与渲染器用的是同一套模型。
- 论坛扩展用具名节点表达，不塞进 `html` 节点：
  - 剧透（行内 / 块）；
  - @提及（带 `user_id`）；
  - 楼层引用（带 `reply_id` 与楼层号）；
  - 图片：带完整 Image 对象；`/image/<hash>` token 在服务端解析；
  - 视频；
  - 代码块：带语言。
- 词表是**封闭**的，客户端遇到未知节点时渲染其子节点（或其纯文本），不报错。新增节点类型是加法。
- 原始 HTML：现行管线允许 HTML 再用 bluemonday 清洗。v1 不下发 `html` 节点。W1 先普查库里真实出现的 HTML 标签，逐一映射到节点或降级为文本。
- 网页用 Vue 组件递归渲染，**不再 `v-html`**，从结构上消灭正文 XSS；App 原生渲染。KaTeX 在渲染组件里做，不再在客户端对 HTML 字符串做后处理。
- `content_markdown`（源文）只在编辑语境下发（作者或有编辑权者请求编辑数据时）。
- 目录（TOC）由客户端从 heading 节点算，不另发。

## §7 i18n 总则

- 面向机器的文本一律英文且稳定：`code`、`reason`、枚举值、字段名。不做 `Accept-Language` 协商（infra D15 / D20）。
- 用户产生的内容按原文下发。
- **服务端不得产出给终端用户看的句子**。通知「xxx 回复了你的话题」、萌萌点流水的原因、系统消息，一律下发结构化事件：事件类型 token + 参与者引用 + 目标引用 + 参数，句子由客户端按 locale 拼。这条在消息 / 通知域迁移时落地，现行的中文句子是 App i18n 的最大障碍。
- 封闭枚举的显示标签由客户端按 token 查译文，例如分类、话题状态、表情。
- 属于数据的词表（管理员能增删的）要么带 `display_name` + `localized{}` 下发，要么升格为封闭枚举。每个词表迁移时二选一，写进对应波的任务书。

## §8 协议头

- 每条响应带 `X-Request-ID: req_<26 位 ULID>`。客户端带来合法的 `req_` 值就沿用，否则新铸；错误体里同值。
- **K15**：v1 全部响应（含错误）`Cache-Control: no-store`。
- 401 带 `WWW-Authenticate`（K3）。
- CORS 沿用站点现有的 origin 白名单（网页带 cookie，不能用 `*`）；`Access-Control-Expose-Headers` 加上 `X-Request-ID`、`Idempotency-Replayed`、`Location`、`Deprecation`、`Sunset`、`Link`、`Retry-After`。
- 退役头（`Deprecation` RFC 9745 / `Sunset` RFC 8594 / `Link rel="deprecation"`）的格式按 infra 02 §5.3。
- 未注册的 `/api/v1/**` 路径、错误的方法、不支持的 media type，都回同一套 problem 体（`NOT_FOUND` / `METHOD_NOT_ALLOWED` / `UNSUPPORTED_MEDIA_TYPE`），不得落到 Fiber 默认页或旧信封。
- panic 恢复产出 `500 INTERNAL_ERROR` 的 problem 体，带 `request_id`。
