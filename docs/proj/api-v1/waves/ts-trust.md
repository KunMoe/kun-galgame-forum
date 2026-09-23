# TS · 举报与信任

> 2026-09-23 立。分支 `api-v1/ts-trust`，迁移号段 180–184（**本轨不用**）。
> 契约与变异题同一个提交，早于实现。

## 1. 普查

### 1.1 路由与调用方

七条旧路由全部由 `TrustHandler`（`internal/trust/handler/trust_handler.go`）独占，普查确认没有别的域共用它。

| # | 旧 | 方法 | 档 | 调用方 |
|---|---|---|---|---|
| 1 | `GET /api/report/reasons` | `GetReasons` | authed | `composables/useReportReasons.ts` |
| 2 | `POST /api/report/submit` | `SubmitReport` | authed | `components/report/Modal.vue`（`<ReportButton>` 挂在话题×2、回复、评论、galgame、用户主页六处） |
| 3 | `GET /api/admin/trust/review-items` | `ListReviewItems` | `trust.review` | `pages/admin/moderation.vue` |
| 4 | `GET /api/admin/trust/review-items/:id` | `GetReviewItem` | `trust.review` | 同上 |
| 5 | `POST /api/admin/trust/review-items/:id/claim` | `ClaimReviewItem` | `trust.review` | 同上 |
| 6 | `POST /api/admin/trust/review-items/:id/decide` | `DecideReviewItem` | `trust.review` | 同上 |
| 7 | `POST /api/trust/callback` | `Callback` | 公开，HMAC | **infra kun_trust 的处置派发器**（见 §1.4） |

`apps/web/server/`（Nitro）零调用；`../kungal-apps` 零调用。

### 1.2 生产取值（`kun_trust` 库，`site = 'kungal'`，2026-09-23）

- **审核条目 444**：`actioned` 44（来源 reports 9 / ai_text 20 / community_forward 15），`dismissed` 400（reports 2 / ai_text 174 / community_forward 224）；`pending` / `claimed` 此刻为 0。来源 `ai_image` / `mislabel` / `manual` / `ai_sample` 0 行。
- 条目的 `subject_kind`：community_post 243、galgame_resource 152、forum_topic 20、galgame_rating 17、galgame_collection 4、forum_reply 3、galgame 2、galgame_comment 2、user 1。**九种里网页的 `TRUST_SUBJECT_KIND` 只给六种写了中文标签**，其余显示裸 key。
- 条目字段空值：`severity` 433/444 为空，`classifier_score` 248，`report_weight_sum` 433，`subject_reach` 444（全空），`context_note` 10，`claimed_by` 128；`context_note` 最长 543 字。`subject_id` 全部是十进制数字。
- **举报 93**：forum_topic 44、user 25、galgame 19、forum_reply 3、galgame_comment 2。`note` 最长 703、`snapshot` 最长 233、`subject_url` 最长 53 且无空值。
- 理由 6 条，全是全局（无 kungal 扩展）：`abuse` `spam` `illegal` `rating_mislabel` `copyright` `other`。
- 处置：action `none` 226、`hide` 225、`remove` 11、`restrict` 2；`reason_code`：`review_dismissed` 226、`ai_scan_flagged` 194、`spam` 21、`abuse` 10、`other` 7、`illegal` 3、`rating_mislabel` 2、**`span` 1**（手打的错字）。`statement` 只有 1 行，17 字。
- 论坛侧 `trust_disposition_applied`：hide 22、remove 9、restrict 2。
- 已注册的 kungal subject kind 17 个，其中 12 个带回调地址 `http://kungal-api:2334/api/trust/callback`（docker 内网主机名）。

### 1.3 语义与疑似 bug

1. **管理面四条是原样透传。** `TrustService` 把 infra 的 `data` 当 `json.RawMessage` 回给网页，论坛 API 的形状就是 infra 当天的形状：`status` / `source` / `action` 是裸整数，`reporter_id` / `claimed_by` 是裸用户号，`reason_id` 是 infra 的内部行号（网页显示「理由 #3」）。
2. **举报人自带的 `subject_url` 只校验是不是 http(s)。** 审核页优先拿它当「查看内容」的链接（`moderation.vue` 的 `subjectHref`），于是任何登录用户都能把一条外站链接摆到版主面前。
3. **`reason_code` 是自由输入框**，生产里已经有一条 `span`。
4. **理由接口的兜底**：infra 失败时回本地硬编码的中文列表（`internal/trust/reasons.go`，标签还和 infra 不一样：「辱骂骚扰」对「辱骂/骚扰」），而随后的提交照样失败——兜底只让用户多填一次表。
5. 旧错误一律 `233` + 中文句子；上游 409 被翻成 `233`「该条目状态已变化」。
6. 列表：`status` / `source` 用 `-1` 表示全部，`limit` 静默夹到 200，`page` 无上限。
7. **权限两层不一致**：论坛闸是 `trust.review`（版主基线），infra 闸是 `trust.queue_access`（全局角色或站点版主）。给一个无角色用户单独授予 `trust.review`，论坛放行、infra 403。生产里没有个人授予（`user_permission_override` 0 行），不建模，只在 v1 里把上游 403 译成 `PERMISSION_REQUIRED`。
8. 服务端不拦「举报自己的内容」（网页藏了按钮），也不查举报对象是否存在或可见。infra 按 `(site, kind, subject_id, reporter)` 去重，重复举报天然幂等。不改：查存在性要给每个 kind 各写一个查询，违背「新增可举报类型 ≤ 1 注册 + 1 挂载 + 1 适配器」。
9. 网页把 `snapshot` 截到 1000 个 UTF-16 单元，服务端允许 2000 个字符。
10. **G0 相关**：galgame 举报的 `subject_id` 是论坛现在的 gid（19 条举报、2 个条目）。G0b 把 gid 改号成 catalog work id 之后，这几条的「查看内容」会指向别的作品。已记给 G 轨，本轨不处理。
11. **不属本轨、但必须报告**：93 条举报里 **64 条（69%）停在 `received`，从没开出审核条目**——权重和没到 infra 的聚合阈值（平台默认值，kungal 无站点策略）。举报人看到「举报已提交」，没有任何版主会看到它们。这是 infra 的策略，论坛侧无可修。

### 1.4 回调不迁（K-TS1）

`POST /api/trust/callback` 是 infra → 论坛的机器对机器回调：凭证是 HMAC 签名，调用方是 infra 的派发器，地址登记在 infra 的 `trust_subject_kind.callback_url` 里（12 行）。

- 它不属于 v1 面。K2 规定 v1 只有一种主体（终端用户或匿名）；把 webhook 写进 App 绑定的 spec，App 的生成客户端里就多一个它永远不该调的操作。
- 挪路径要先改 infra 的 12 行登记、再留一段双路径过渡期，换来的只是路由表好看。
- 所以原样保留，行为不变。`legacy_route_baseline` 的地板因此从 1（`/healthz`）变成 2。

## 2. 范围

| 旧 | v1 |
|---|---|
| `GET /api/report/reasons` | `GET /api/v1/report-reasons` |
| `POST /api/report/submit` | `POST /api/v1/reports` → **204** |
| `GET /api/admin/trust/review-items` | `GET /api/v1/admin/review-items`，**页码集合** |
| `GET /api/admin/trust/review-items/:id` | `GET /api/v1/admin/review-items/{review_item_id}` |
| `POST …/:id/claim` + `POST …/:id/decide` | `PATCH /api/v1/admin/review-items/{review_item_id}`（状态迁移，B7） |
| `POST /api/trust/callback` | **不迁**（§1.4） |

删 6 条，`legacy_route_baseline` 224 → **218**（以合并时 rebase 后重新生成为准）。

## 3. 形状

### 3.1 词表

| 类型 | 词表 | 取值 | 理由 |
|---|---|---|---|
| `TrustSubjectKind` | **开放** `trust_subject_kind`，`^[a-z][a-z0-9_]{0,63}$` | 论坛自己声明的 16 个（`gate.CanonicalSubjectKinds`）+ community 服务注册的 `community_post` | 收件箱里会出现别的服务注册的 kind；新增可举报类型不应该改 spec |
| `ReportReasonKey` | **开放** `report_reason`，`^[a-z][a-z0-9_]{0,63}$` | 生产 6 个 | infra 管理员可以增删，是数据不是代码 |
| `ReviewItem.opened_by` | **开放** `review_item_origin` | `reports` `ai_text` `ai_image` `community_forward` `mislabel` `manual` `ai_sample`；上游出现未知整数时发 `unknown` | infra 的来源会增加；不能因为上游多一种来源就让收件箱 500 |
| `ReviewItem.state` | 封闭 | `pending` `claimed` `actioned` `dismissed` | infra 状态机是封闭的 |
| `ReviewItemPatch.action` | 封闭 | `none` `hide` `remove` `warn_user` `restrict` `escalate_idp` | 同上，对应 infra 0–5 |

`subject_kind` 在请求与响应里是同一个开放类型——G8 按名比较 enum，请求若用封闭枚举就会与响应冲突，而收件箱里的 kind 本来就多于可举报的 kind。

### 3.2 `ReportReason`

```json
{ "object": "report_reason", "key": "spam", "display_name": "垃圾信息" }
```

- `display_name` 是 infra 登记的标签原文。infra 的理由表只有一种语言（`name_cn`），所以不发 `localized{}`；上游加了再加，是加法（01 §7 的「带 display_name」一支）。
- `severity` 不下发：两端都没有消费者（A5）。

### 3.3 `ReviewItemSummary` / `ReviewItem`（都是 `object: "review_item"`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | DecimalID | 审核条目号 |
| `subject_kind` | `TrustSubjectKind` | |
| `subject_id` | DecimalID | |
| `opened_by` | `review_item_origin` | 谁开出了这个条目（旧名 `source`，与 `MoemoepointEntry.source` 同名异型，G8） |
| `state` | 封闭 | |
| `priority` | number ≥ 0 | 队列的排序键 |
| `severity` | integer ≥ 0 \| null | |
| `classifier_score` | number 0–1 \| null | |
| `report_weight_sum` | number ≥ 0 \| null | |
| `reach_count` | integer ≥ 0 \| null | 开条目时内容已触达的人数（旧名 `subject_reach`） |
| `context_note` | string ≤ 4000 \| null | 非举报来源的判定依据。超长截断 |
| `claimant` | UserRef \| null | 认领人（旧 `claimed_by` 裸用户号） |
| `claimed_at` | DateTime \| null | |
| `decider` | UserRef \| null | 处置人（旧 `decided_by`） |
| `decided_at` | DateTime \| null | |
| `created_at` | DateTime | |

`ReviewItem` = 以上全部 + `reports: [ReviewReport]`（按 id 升序，只在详情与 PATCH 响应里）。

### 3.4 `ReviewReport`（`object: "report"`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | DecimalID | |
| `reporter` | UserRef | 旧 `reporter_id` |
| `reason_key` | `ReportReasonKey` \| null | 旧 `reason_id`（infra 行号）。理由已停用、不在可用列表里时为 `null` |
| `note` | string ≤ 1000 \| null | |
| `snapshot` | string ≤ 2000 \| null | 举报人提交的内容快照 |
| `subject_url` | uri ≤ 512 \| null | **只有以 `https://www.kungal.com/` 开头才下发**，否则 `null`（§4 第 2 条） |
| `weight` | number ≥ 0 | 举报人权重 |
| `created_at` | DateTime | |

### 3.5 `ReportCreate`（`POST /reports` 的请求体）

| 字段 | 类型 | 规则 |
|---|---|---|
| `subject_kind` | `TrustSubjectKind` | 必须是论坛声明的 kind（`gate.CanonicalSubjectKinds`），否则 `422 UNKNOWN_VALUE` |
| `subject_id` | DecimalID | |
| `reason_key` | `ReportReasonKey` | 必须在当前可用理由里，否则 `422 UNKNOWN_VALUE` |
| `note` | string ≤ 1000 \| null | 自由文本 |
| `snapshot` | string ≤ 2000 \| null | 自由文本 |
| `subject_url` | uri ≤ 512 \| null | 必须以 `https://www.kungal.com/` 开头，否则 `422 NOT_ALLOWED_VALUE` |

举报人**只**取自凭证。`Idempotency-Key` 可选（K12「全部 POST 支持」；不强制，因为 infra 本来就按举报人 + 对象去重）。成功 `204`：举报对举报人不可回读，没有自己能 GET 的 id（与 RC 的 D13 `flags` 同理）。

### 3.6 `ReviewItemPatch`（`PATCH /admin/review-items/{review_item_id}`）

```json
{ "state": "actioned", "action": "hide", "reason_code": "spam", "statement": "…" }
```

| 字段 | 规则 |
|---|---|
| `state` | 必填，`claimed` \| `actioned` \| `dismissed` |
| `action` | `actioned` 时必填（缺 → `422 REQUIRED`）；其余状态出现 → `422 INCONSISTENT_WITH /state` |
| `reason_code` | `^[a-z][a-z0-9_]{0,63}$`；`actioned` 时必填；其余状态出现 → `INCONSISTENT_WITH` |
| `statement` | ≤ 1000；只允许与 `actioned` 同时出现 |

- `claimed` = 旧 `claim`（只有 `pending` 能被认领，认领人是调用者）；`actioned` / `dismissed` = 旧 `decide`（`pending` 或 `claimed` 都能直接处置）。
- 响应 `200` + 写后重读的完整 `ReviewItem`。
- 网页的 `reason_code` 从输入框改成**理由下拉**（取自 `GET /report-reasons`），堵掉 `span` 这类错字；服务端只校验格式，因为生产里还有 `ai_scan_flagged` 这类系统码。

## 4. 逐条裁决

| 旧面 | 裁决 |
|---|---|
| 管理面透传 infra 的 JSON | 论坛自己的表示层（§3），枚举全部是 token，人全部是 `UserRef` |
| `subject_url` 只查 http(s) | 写：必须在本站源下（`apiv1.SiteOrigin`）；读：不在本站源下的一律发 `null`——生产的 93 条全部在本站源下，读侧过滤只防以后 |
| `reason_code` 自由输入 | 格式校验 + 网页下拉（§3.6） |
| 理由接口失败回本地中文表 | **`503`**。本地表 `internal/trust/reasons.go` 随旧路由删除。理由在进程内缓存 5 分钟 |
| 列表 `-1` 表示全部 | 缺席 = 不过滤（A6） |
| 列表 `source` 过滤 | **不提供**：两端都没用过（A5），要时是加法 |
| 列表 `limit` 静默夹到 200、`page` 无上限 | `limit` 1–100、默认 20，深度上限 10000（01 §4，复用 `collect.PageNumber`） |
| 上游 409 → `233` 中文 | `409 INVALID_STATE_TRANSITION`，`detail` 写当前状态（冲突后再读一次；再读失败就不写） |
| 上游其它失败 | 照 U1 的 K23：只映射列出的上游码，**未列出的一律 `503`**，完整上游错误只进日志 |
| Bearer | `trust.review` 一律经 `user.Can` 判断，Bearer 永远 403 |

上游错误映射（审核三条）：

| 上游 | v1 |
|---|---|
| 未配置 / 网络 / 5xx / 400 / 其它 | `503 SERVICE_UNAVAILABLE` |
| 401 | `401 INVALID_CREDENTIAL` |
| 403 | `403 PERMISSION_REQUIRED` |
| 404 | `404 NOT_FOUND` |
| 409 | `409 INVALID_STATE_TRANSITION` |

举报提交：上游 429 → `429 RATE_LIMITED`；上游 422（论坛已先行校验过 kind 与理由，所以只可能是 infra 侧没登记这个 kind）→ `503`；其它同上表第一行。

## 5. 预分配

- 迁移：**无**。
- 错误码：**无新增**。用到 `PERMISSION_REQUIRED` `NOT_FOUND` `INVALID_STATE_TRANSITION` `VALIDATION_FAILED`（`REQUIRED` `UNKNOWN_VALUE` `NOT_ALLOWED_VALUE` `INCONSISTENT_WITH` `TOO_LONG` `INVALID_FORMAT`）`RATE_LIMITED` `SERVICE_UNAVAILABLE` `INVALID_CREDENTIAL`。
- K 编号：并行会话各自往下编会撞号（U2 已用 K24），本轨用带前缀的 **K-TS1…**，合并后由看板维护者决定是否改成全局号。
  - **K-TS1**：S2S 回调不属于 v1 面（§1.4）。
  - **K-TS2**：用户提交的外链只接受本站源；读侧同一谓词过滤，存量数据不可信时读侧也不下发（§4）。

## 6. 网页

- `useReportReasons.ts`、`report/Modal.vue`：换类型化客户端；标签用 `display_name`；失败走 `reportProblem`。
- `pages/admin/moderation.vue`：列表走页码集合（固定发 `limit=30`），状态页签用 token，「全部」= 不带 `state`；详情与 PATCH 换类型化客户端；认领人 / 举报人显示 `UserRef`；举报理由显示 `display_name`；`reason_code` 改下拉。
- `constants/trust.ts`：映射表的键从整数换成 token；补上收件箱里出现过的另三种 kind 的标签。
- `shared/types/trust.ts`、`shared/types/report.ts`：删除。
- `legacy-fetch-baseline` 下调 6（`moderation.vue` 4、`Modal.vue` 1、`useReportReasons.ts` 1）。

## 7. 变异题（先于实现提交）

测试用内存假 trust 上游（`httptest`），它记录收到的每个请求。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 列表不查 `trust.review` | 普通用户 → `403 PERMISSION_REQUIRED` |
| 2 | `user.Can(perm.TrustReview)` 换成 `perm.CanUser(u.ID, u.Roles, perm.TrustReview)` | 被个人授予 `trust.review` 的用户经 Bearer 访问 → 403（静态守卫 `bearer_guard_test` 同时会红） |
| 3 | 提交时 `reporter_id` 不取自凭证（写成 0） | 假上游收到的 `reporter_id` 等于调用者 |
| 4 | 写侧不查 `subject_url` 的源 | `https://evil.example/topic/1` → `422 NOT_ALLOWED_VALUE /subject_url`，假上游一个请求也没收到 |
| 5 | 读侧原样下发 `subject_url` | 上游举报带外站链接 → 详情里该举报的 `subject_url` 为 `null` |
| 6 | 不先校验 `reason_key` | 未知理由 → `422 UNKNOWN_VALUE /reason_key`（去掉之后假上游回 422，论坛回 503） |
| 7 | 列表不转发 `state` | `state=dismissed` 的 `items` 全是 dismissed，`total` 等于假上游里 dismissed 的条数 |
| 8 | 列表不强制 `site=kungal` | 假上游只在 `site=kungal` 时返回数据，否则返回别站条目 → 断言只看到本站 |
| 9 | 去掉「`actioned` 必须带 `action`」 | → `422 REQUIRED /action`（去掉之后假上游回 400，论坛回 503） |
| 10 | 非 `actioned` 时静默忽略 `action` | `{"state":"dismissed","action":"hide"}` → `422 INCONSISTENT_WITH /action`，假上游没收到 decide |
| 11 | 上游 409 译成 503 | 认领已被认领的条目 → `409 INVALID_STATE_TRANSITION` |
| 12 | 理由接口在上游失败时回内置列表 | 假上游 500 → `GET /report-reasons` 为 `503` |

## 8. 实现时对本契约的修正（2026-09-23，只增不改）

1. **举报详情里的理由是对象，不是键。** `ReviewReport.reason_key`（可空）与 `ReportCreate.reason_key`（必填、非空）同名不同可空性，G8 拦下。改成 `report_reason: ReportReason | null`，顺带让审核页直接拿到 `display_name`，不用再按键查一次。
2. **提交举报先按 OAuth 当前记录判封禁**（与 RC 的 `requireActive` 同理：会话只在刷新令牌时才知道自己被封）。封禁 → `403 ACCOUNT_BANNED`；OAuth 不可用 → `503`（K17 的失败关闭），举报不发出。
3. **操作描述里不点名 reason。** G4 把描述里的每个大写蛇形词都当错误码去注册表里找，`UNKNOWN_VALUE` 这种 reason 会被判「不在注册表」。422 的描述改成白话写出哪一项、为什么。
4. PATCH 的路径参数最初放在嵌入的未导出结构体里，F10 当场红（W2 踩过的同一个坑）；改成直写字段。
5. 审核页的「处置理由」下拉复用 `useReportReasons`，与举报弹窗共用同一份缓存。

## 9. 验收记录

**闸**：`make lint` 零输出；`KUN_REQUIRE_TEST_DB=1 go test -count=1 -p 1 ./...` 66 个包全绿（专属库 `kungal_test_ts_trust`）；`make openapi` / `gen:api` 无漂移；`pnpm lint`、`pnpm typecheck`（`vue-tsc -b --force`）、`pnpm -F web test`（62 文件 426 条）全绿；`deadcode` 与 master 相比没有新增。TS 的测试不碰本地库：五个操作全是上游代理，测试用内存假 trust 上游 + 只回被请求 id 的假 OAuth。

**变异**（`internal/trust/apiv1`，逐条改、跑、还原）：12/12 变红。

| # | 改动 | 变红的测试 |
|---|---|---|
| 1 | 列表不查 `trust.review` | `TestV1ReviewInboxRejects`、`TestV1ReviewItemMissing` |
| 2 | `user.Can` → `perm.CanUser` | `TestV1ReviewBearerNeverReviews`（被个人授予 `trust.review` 的 Bearer 用户拿到了 200） |
| 3 | `reporter_id` 写成 0 | `TestV1CreateReport` |
| 4 | 写侧不查 `subject_url` 的源 | `TestV1CreateReportRejects/foreign_link`、`/look-alike_host` |
| 5 | 读侧原样下发 `subject_url` | `TestV1ReviewItemDetail` |
| 6 | 不先校验 `reason_key` | `TestV1CreateReportRejects/unknown_reason`（假上游回 422 → 论坛 503） |
| 7 | 列表不转发 `state` | `TestV1ReviewInboxStateFilter` |
| 8 | 列表不强制 `site` | `TestV1ReviewInboxWalk`、`TestV1ReviewInboxShape` |
| 9 | `actioned` 不要求 `action` | `TestV1ReviewItemDecideRejects/actioned_without_action` |
| 10 | 非 `actioned` 静默忽略 `action` | `TestV1ReviewItemDecideRejects/dismissed_with_action` |
| 11 | 上游 409 → 503 | `TestV1ReviewItemClaim` |
| 12 | 理由接口上游失败回内置列表 | `TestV1ReportReasonsUpstreamDown` |

排序：假上游按 `priority DESC, id DESC` 出，种子里 7 个条目共享同一 `priority` 跨页，`limit` 2 与 3 各翻一遍与期望序逐条相等。排序本身在 infra，论坛只负责不丢不重、`total` 与 `items` 同谓词。

**浏览器实测**（本分支自起 API :2344 + 网页 :2343，打**真的**本地 kun_trust 与 OAuth，管理员会话持本地账号中心签发的真令牌；无头 Chromium）：

- 匿名进 `/admin/moderation` → 跳登录页；普通用户 → 被中间件送回首页；普通用户直打 `GET /api/v1/admin/review-items` → `403 PERMISSION_REQUIRED`。
- 普通用户与管理员各在话题 4241 的「更多 → 举报」里选「垃圾信息」提交：`GET /report-reasons` 200、`POST /reports` 204、提示「举报已提交」；上游两行举报的 `reporter_id` 分别是 10 与 8，管理员的员工权重当场开出审核条目。
- 管理员在收件箱点开该条目：举报人显示为用户卡片（头像 + 名字），理由显示「垃圾信息」；点「认领」→ `PATCH` 200，显示「处理中，认领人 xiaohuo」；处置动作选「不处置（仅记录）」、理由从下拉选「垃圾信息」→ `PATCH` 200；上游落下 `action = 0, reason_code = spam` 的处置。「全部」页签按优先级列出 30 条并带分页器，游戏资源 / 社区评论 / 游戏评价等新 kind 都有中文标签。
- 清理：本地 kun_trust 的两行举报、审核条目与处置已删（审计表是哈希链，没动），两个会话已删，自起的两个进程已按 PID 停止。

## 10. 上线后（2026-09-23，infra 侧）

§1.3 第 11 条已由 infra 修复：kun_trust 给 `kungal` 加了站点策略，`aggregate_threshold = 0.5`（单条举报即开审核条目）；生产上 64 条停在 `received` 的举报被回填成 58 个待处理条目（user 20、forum_topic 18、galgame 17、forum_reply 3），优先级取严重度、`report_weight_sum` 取权重和，举报行已链接（status 1）。另外 G0 窗口把 galgame 举报的 `subject_id` / `subject_url` 改写到了新的 work id（§1.3 第 10 条）。
