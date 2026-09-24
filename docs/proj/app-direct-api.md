# App 直连论坛 API（2026-09-17）

> 本仓自有工程笔记（**非** infra 镜像）。kungal-apps 工单 02「App 直连论坛 API 的四项前置」的论坛侧交付与契约。
> 取代 [app-aggregation-api.md](./app-aggregation-api.md) 的方向。
>
> **⚠️ 2026-09-18 方向变更：裁决 1 与 3 已被 [api-v1](./api-v1/README.md) 取代。** App 只绑定 `/api/v1`（problem+json、生成的 spec、字符串 id、游标分页、结构化正文），不再消费本文列出的 `/api/*` 旧端点，也不再用 `X-Kungal-Nsfw` 请求头（v1 改为显式查询参数 `nsfw=`；该请求头已于 2026-09-23 从论坛删除，见 §3）。幂等回放头在 v1 叫 `Idempotency-Replayed`。本文的 Bearer 校验（§1）、staff 能力拒绝、版本闸（§4）、assetlinks（§5）仍然有效；这几项迁到 v1 时以 `api-v1/CHANGELOG.md` 为准。
>
> **工单 02 回报对照**：① 放行端点与 curl 示例在 §1「放行范围」；② 幂等键在 §2；③ 版本闸在 §4；④ assetlinks 在 §5；⑤ galgame 供数结论在 §1「galgame 供数」。infra 工单 01 裁决要求通知、未读、下载也走论坛，这三面在 §1「放行范围」逐条列出，已全部确认可用 Bearer 访问。

## 裁决（2026-09-17，App 侧拍板）

1. **复用 `/api/*`，信封 `{code, message, data}` 不变。** 不另起 `/app/v1` 或 problem+json：App 消费的就是论坛现有面，同一数据两种信封并存没有收益。真出现负载分叉时，归宿是蓝图预留的 mobile BFF，不是平行 API 面。
2. **所有需要登录的路由统一接受 Bearer；staff 能力在 Bearer 通道一律拒绝**（生态「staff 面永不对外」）。
3. **NSFW 偏好走请求头**，不让 App 伪造 web 的 `KUNGalgameSettings` cookie。
4. **版本闸单端点，一张表覆盖全平台**；`min_version` 不按平台拆。
5. 网页补 `/app` 下载页与 `/app/oauth/callback` 兜底页。

## 1. 鉴权：`Authorization: Bearer`

App 用 AppAuth + PKCE 直接从 OP 换出 access token，然后 `Authorization: Bearer <token>` 直打 `https://www.kungal.com/api/*`（Cloudflare → kungal-api，Nitro 不在这条链上）。

### 校验（`internal/user/oauth/verify.go`）

- 本地用 OP 的 JWKS 验签：只收 ES256 / RS256，必须带 `kid`，头部 `typ` 必须是 `at+jwt`（挡掉 id_token）。
- `iss` 必须等于 OP 的**公网**源 `https://account.nextmoe.com`，`exp` 必填，容忍 30s 时钟偏差。
- `client_id` 必须在 `KUN_BEARER_CLIENT_IDS` 白名单里。**不看 `aud`**：OP 把 `aud` 填成站点域名（`www.kungal.com`），同站点的任何 client 都会带它。
- 用户 id 取自定义声明 `id`（跨库同一个整数）；`sub` 是 UUID。
- JWKS 按 kid 缓存；遇到未知 kid 才重拉，且最多 30s 一次。

**实测（2026-09-17）**：抽样 40 个线上会话里的 access token，34 个是 `ES256` + `typ: at+jwt`，kid `u1x9P7ub…` 与线上 JWKS 一致。另外 6 个 `HS256` 是 OP 切换非对称签名前签发、之后没再刷新的老会话，App 拿到的 token 不会是这种。本地 dev OP 走完整 PKCE 流程签出的 token 形状相同（`iss=http://127.0.0.1:9277`，`aud=["www.kungal.com"]`，`site_id=2`）。

### 语义（`internal/middleware/bearer.go`）

| 情况 | 响应 |
|---|---|
| 带 `Bearer`（含空 token） | 只走 Bearer 路径，不看 cookie |
| token 无效/过期/client 不在白名单/通道未开 | `401 {"code":205}`，**可选登录的读路由也一样**，不降级为匿名（否则 App 发现不了过期） |
| JWKS 拉不到（OP 故障） | `500 {"code":233}`，不是 401，App 不要因此登出 |
| 首次见到该用户的初始化失败 | `500 {"code":233}`，下次请求会重试 |
| 访问任何管理/权限闸 | `403 {"code":233,"message":"管理操作请在网页端进行"}` |

- **没有 staff 能力**：Bearer 用户的 `roles` 会剥掉 `moderator` / `admin` / `ren`（`creator` 等保留）。`UserInfo.Can` / `CanModerate` / `CanAdminister` 对 Bearer 恒为 false，**连个人权限覆盖也不生效**；`/api/v1/me/permissions` 恒返回空列表。`internal/middleware/bearer_guard_test.go` 禁止任何代码绕过这些方法、直接在 `u.Roles` 上做能力检查。
- **首次见到用户**（每用户 24h 一次）：补做网页 OAuth 回调里做的两件事：`kungal_user_state` 行（没有这行，发帖会失败）和社区 trust boost。boost 用的是**未剥离**的 roles，因为它每个用户只声明一次、由 SETNX 守着，从 App 先声明一个剥离后的值，会把版主之后从网页登录时的 boost 锁掉。
- **封禁**：access token 有效期 15 分钟，没有吊销列表，所以封禁最多滞后 15 分钟，和网页会话的刷新周期相同。封禁用户发的内容仍在渲染层隐藏。
- **下游转发**：论坛把 App 的 token 原样当 Bearer 转给 catalog `/v2` 用户面和 OAuth `/auth/me`。catalog 不校验 `aud` / `client_id` 白名单，但会读取该 client 的 `catalog_site` 和 scope。社区接口和图床（`pkg/imageclient`）都走论坛 client 的 Basic 认证，不经过用户 token，不受影响。

### 对 infra 注册 kungal-app 的要求（工单 01）

- `site_id = 2`（www.kungal.com）：否则 `site_roles` 为空。
- `catalog_site = kungal`：否则编辑提案 / 认领返回 `SITE_NOT_BOUND`。
- `owner_user_id` 为空：否则会被当成第三方 client（编辑受限，catalog 管理面拒绝）。
- scope 需包含 `catalog:edit`、`folder:read`、`folder:write`。
- 固定的 client_id（如 `kungal-app`）后台生成不了，只能手工插入或写进 seed。
- playtime 按 `(user, work, client_id)` 分行存，App 上报的是独立一行，读取时取各行最大值。

**实际注册（2026-09-17，线上 `oauth_clients` 核对过）**：两个 public client，上述四条都满足，scope 均为 `openid profile email catalog:read catalog:edit folder:read folder:write`，RT 90 天。

| client_id | 平台 | redirect_uris |
|---|---|---|
| `kungal-app` | Android / iOS | `com.kungal.app://oauth2redirect`、`https://www.kungal.com/app/oauth/callback`、`com.kungal.app://logout` |
| `kungal-app-desktop` | Windows / Linux | `http://127.0.0.1/oauth/callback`（loopback，端口宽松匹配） |

**两个都要进白名单**：漏掉哪个，那个平台的所有用户都会 401。

### 放行范围

路由挂了 `OptionalAuth` / `Auth`，Bearer 才会被校验（完整清单与每条路由的中间件链见 `apps/api/internal/app/testdata/routes.golden`）。完全公开的路由不看 `Authorization` 头，带了无效 token 也照常返回。下表中：

- **匿名+**：匿名可读，带 Bearer 时附加登录态（如 `is_liked`），token 无效时 401；
- **公开**：完全不看 token；
- **Bearer**：必须登录。

**实测（2026-09-18，本地用 dev OP 真实走 PKCE 签出的 token）**：
- 下文「通知与未读」表里的读接口带 Bearer 全部返回 200；
- 不带 token 或 token 无效时返回 401/205；
- 两个下载详情接口都通过了鉴权。但 dev 库的用户全是「已注销」，资源都被过滤成 404，所以下载内容本身没能在 dev 跑通。

#### 首批读写

| 端点 | 方式 | 说明 |
|---|---|---|
| `GET /api/v1/topics` | 匿名+ | 话题列表。旧 `GET /api/topic` 已于 2026-09-19 删除。参数与响应以 `apps/api/openapi/kungal-v1.json` 为准（`listTopics`）：`cursor` + `limit`（1–100），`sort` 取声明的 token（如 `bumped_desc`），`include_nsfw=true` 才含 NSFW；错误是 problem+json，见 `docs/proj/api-v1/` |
| `GET /api/v1/topics/{topic_id}`、`GET /api/v1/topics/{topic_id}/replies` | 匿名+ | 话题详情与楼层。旧 `GET /api/topic/:tid`、`GET /api/topic/:tid/reply` 已于 2026-09-22 删除。响应以 `apps/api/openapi/kungal-v1.json` 为准；回复走游标分页（`cursor` + `limit`），不是页码 |
| `GET /api/v1/works` | 公开 | 列表（前瞻；App 尚未调用）。旧 `GET /api/galgame` 已删除。参数与响应以 `apps/api/openapi/kungal-v1.json` 为准（`listWorks`） |
| `GET /api/v1/library-works` | 公开 | 资料库列表（前瞻；App 尚未调用）。旧 `GET /api/galgame?library=true` 已删除。参数与响应以 `listLibraryWorks` 为准 |
| `GET /api/v1/works/{work_id}` | 匿名+ | 作品详情。旧 `GET /api/galgame/:gid` 已删除。`include_nsfw` 只控制成人标签是否出现；错误是 problem+json |
| `PUT` / `DELETE /api/v1/works/{work_id}/like` | Bearer | 点赞槽。都 200 `WorkEngagement`。自赞 `403 SELF_LIKE_FORBIDDEN` |
| `PUT` / `DELETE /api/v1/works/{work_id}/covers/{cover_id}/vote` | Bearer | 封面投票槽（前瞻；App 尚未调用）。都 200 `WorkCoverEngagement` |
| `PUT` / `DELETE /api/v1/works/{work_id}/playtime` | Bearer | 游玩时长槽（前瞻；App 尚未调用）。都 200 `WorkViewerPlaytime`。DELETE 只撤回本 client 这一行，回执可能仍带其它 client 的分钟 |
| `GET /api/v1/me/playtimes` | Bearer | 调用者自己的游玩时长列表（前瞻；App 尚未调用）。页码；`include_nsfw` |
| `POST /api/v1/collections` | Bearer | 创建收藏夹（前瞻；App 尚未调用）。**必须**带幂等键。201 `Collection`。id 是 catalog folder id，页面 `/collection/{collection_id}` |
| `GET` / `PATCH` / `DELETE /api/v1/collections/{collection_id}` | 匿名+ / Bearer | 收藏夹元数据（前瞻；App 尚未调用）。DELETE 204。旧 `/galgame/collection/{cid}` 经 `GET /api/v1/collection-aliases/{alias_id}` 301 |
| `GET /api/v1/collections/{collection_id}/works` | 匿名+ | 收藏夹作品列表（前瞻；App 尚未调用）。页码 `PageList<WorkSummary>` |
| `PUT` / `DELETE /api/v1/collections/{collection_id}/works/{work_id}` | Bearer | 收藏夹成员槽（前瞻；App 尚未调用）。每夹一条，没有整集替换 |
| `GET /api/v1/me/collections` | Bearer | 调用者自己的收藏夹（前瞻；App 尚未调用） |
| `GET /api/v1/users/{user_id}/collections` | 匿名+ | 用户收藏夹列表（前瞻；App 尚未调用） |
| `POST /api/v1/work-submissions` | Bearer | 投稿新作品（前瞻；App 尚未调用）。**必须**带幂等键。201 `WorkSubmissionCreated`；同名作品 `409 DUPLICATE_SUSPECTS`，确认后带 `is_duplicate_confirmed: true` 重发。旧 `POST /api/galgame/submit` 已删除 |
| `GET` / `PATCH` / `DELETE /api/v1/work-submissions/{work_id}` | Bearer | 自己的投稿（前瞻；App 尚未调用）。GET 回 `ETag`，写时作 `If-Match` 发回（不发 = 不校验）。PATCH `{state: pending \| draft}`；审核态 Bearer 恒 403。DELETE 只删草稿，204。旧 `DELETE /api/galgame/:id`、`/draft`、`/resubmit` 已删除 |
| `GET /api/v1/me/work-submissions` | Bearer | 自己的投稿列表（前瞻；App 尚未调用）。游标；`state` 逗号分隔。旧 `GET /api/galgame/mine` 已删除 |
| `GET /api/v1/work-submission-candidates` | Bearer | 投稿前搜索（前瞻；App 尚未调用）。游标、无 `total`；`include_nsfw` 默认 false。旧 `GET /api/galgame/search/wizard` 已删除 |
| `GET /api/v1/users/{user_id}` | 公开 | 公开资料。未知或封禁/注销用户 404 |
| `POST /api/v1/topics`、`POST /api/v1/topics/{topic_id}/replies` | Bearer | **必须**带幂等键（§2）。旧 `POST /api/topic`、`POST /api/topic/:tid/reply` 已于 2026-09-22 删除；话题 id 在路径上，不再放进请求体 |
| `GET /api/v1/me/account` | Bearer | 当前用户；Bearer 下 `roles` 已剥掉 staff 角色，`content_stance` 恒为 `null`（立场问账号中心） |

#### 通知与未读

论坛**没有** SSE / WebSocket，网页也是轮询。红点只轮询 `GET /api/v1/me` 一个接口，打开消息页时再拉列表。社区墙的通知已由论坛镜像进 `/api/message`（`community: true` 的行），不必再单独拉社区。

| 端点 | 方式 | 说明 |
|---|---|---|
| `GET /api/v1/me` | Bearer | `has_unread_messages`：通知或私信未读即为 true（已静音的类型不计；系统公告不再计入）；另有 `moemoepoint` `has_checked_in_today` `is_creator` `toolset_upload_today_bytes` |
| `GET /api/message/nav/system` | Bearer | 两行：`route:"notice"`（通知）和 `route:"system"`（系统公告），各带 `unread_count` `count` `content` `last_message_time` |
| `GET /api/message` | Bearer | `page`、`limit`（≤30）、`sort_order=asc\|desc`（**必填**）→ `{messages:[{id, sender{id,name,avatar}, receiver_id, link, content, status:"unread"\|"read", type, created, item_count, actor_count, community}], total}`。这个端点**忽略** `type` 参数 |
| `PUT /api/message/system/read` | Bearer | 全部通知标为已读（同时转发给社区） |
| `DELETE /api/message/:id` | Bearer | 删除一条通知 |
| `GET /api/message/admin`、`PUT /api/message/admin/read` | Bearer | 系统公告 `[{id, is_read, content, admin, created}]` 与全部已读 |
| `GET /api/message/muted` | Bearer | 被静音类型的通知，参数同 `/api/message`，可加 `type` |
| `GET/PUT /api/v1/me/notification-preferences` | Bearer | `{muted_types}`，token 为 v1 通知类型 |
| `GET /api/message/nav/contact` | Bearer | 私信会话列表，每项带 `unread_count` |
| `GET /api/message/chat/history` | Bearer | `receiver_id`、`page`、`limit`（≤50） |
| `POST /api/message/chat/send` | Bearer | `{receiver_id, content}`（≤1000 字），**没有**幂等键 |
| `POST /api/message/chat/recall` | Bearer | `{message_id}` |
| `GET /api/community/following` | Bearer | 关注的评论墙，`cursor` `limit`（≤50）→ `{items:[{anchor_kind, anchor_id, link, title, label, galgame_id}], next_cursor}` |
| `POST /api/community/wall/follow` | Bearer | `{anchor_kind, anchor_id, following}` |
| `POST /api/community/wall/read` | Bearer | `{anchor_kind, anchor_id, thread_id}` |

#### 下载

论坛调用 artifact 和社区时用的是论坛自己 client 的 Basic 认证，App 的用户 token 不会转发过去，这符合 infra「community/artifact 不收用户令牌」的裁决。每调用一次下载详情接口，下载计数就 +1，所以断点续传时**不要**重复调用。

| 端点 | 方式 | 说明 |
|---|---|---|
| `GET /api/v1/works/{work_id}/resources` | 匿名+ | 某作品的资源卡片列表 |
| `POST /api/v1/galgame-resources/{resource_id}/downloads` | 匿名+ | galgame 资源的下载信息：`download_urls` `extraction_code` `archive_password`。这些是外部网盘或磁链，文件不在论坛托管 |
| `POST /api/v1/toolsets/{toolset_id}/resources/{resource_id}/downloads` | 公开 | 工具资源下载。返回 `download_url`、`extraction_code`、`archive_password`；文件资源另有 `expires_at`。文件的 `download_url` 是 artifact **预签名 URL**：有效期 24 小时，支持 `Range` 断点续传，过期后再调一次换新 URL |
| `GET /api/v1/app/version` | 公开 | App 自身安装包的下载地址（§4） |

#### curl 示例

```sh
# 匿名
curl -s 'https://www.kungal.com/api/v1/topics?limit=10&sort=bumped_desc'
# Bearer：未读红点
curl -s 'https://www.kungal.com/api/v1/me' -H "Authorization: Bearer $AT"
# Bearer：发回复（幂等键必填）
curl -s -X POST 'https://www.kungal.com/api/v1/topics/4230/replies' \
  -H "Authorization: Bearer $AT" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H 'Content-Type: application/json' \
  -d '{"content_markdown":"…"}'
```

### galgame 供数

Go api 自己就能供数，**不依赖 Nitro**。`/api/galgame` 列表仍是旧面；作品详情是 `GET /api/v1/works/{work_id}`，由 Go api 直接调 catalog 并合并本地数据。Nitro（`apps/web/server/`）只有 sitemap、OG 图、RSS 和几条重定向中间件，没有任何 galgame 数据聚合。所以 App 直连 Go api，拿到的 galgame 数据和网页一致，对应 infra 工单 01 任务 C 的 (b)。

## 2. 幂等键：`Idempotency-Key`

挂在 v1 写端点上，且**必填**（`internal/apiv1/idempotency.go`）：`POST /api/v1/topics`、`POST /api/v1/topics/{topic_id}/replies`、`POST /api/v1/topics/{topic_id}/upvotes`、`POST /api/v1/collections`。旧的 `internal/middleware/idempotency.go` 随 2026-09-22 的旧路由清理一并删除，它那套 `code: 237/238` 的信封错误码不再存在。

- 值必须是标准 UUID（8-4-4-4-12，版本不限）或 26 位 Crockford ULID，否则 `422 INVALID_FORMAT`（`errors[]` 指向这个头）；不带则 `422`，理由 `required`。
- Redis 键为 `kungal:idem:v1:{uid}:{operationId}:{key}`，按用户和操作隔离，保存 **24 小时**。
- 请求指纹 = `sha256(方法 + " " + 路径 + "\n" + 请求体)`。**路径在指纹里**，所以客户端生成键时必须把目标算进去：同一个键换个话题再发，拿到的是 `409 IDEMPOTENCY_KEY_REUSED`（见 `docs/proj/api-v1/01-standard.md` K12）。
- 第一次请求：先写一个 2 分钟 TTL 的「处理中」标记，进程中途挂掉也不会锁死键 24 小时。

| 情况 | 响应 |
|---|---|
| 同键、同指纹、已完成 | 原样返回首次的状态码和响应体，加 `Idempotency-Replayed: true` |
| 同键、首次请求仍在处理 | `409 IDEMPOTENCY_REQUEST_IN_PROGRESS`，App 稍等后用同一个键重试 |
| 同键、不同指纹（换了内容或目标） | `409 IDEMPOTENCY_KEY_REUSED`，属于客户端 bug |

结果保存的范围是 **200–499，但不含 409 和 429**：

- 落进这个范围（含 4xx）的结果会被记下并在 24 小时内原样重放，所以**一次 `422` 会把这个键堵死一整天**，重试必须换新键；
- 5xx、409、429 不保存，键当场释放，可以用同一个键安全重试。

## 3. NSFW 偏好：~~`X-Kungal-Nsfw`~~ → 显式 `include_nsfw`

**`X-Kungal-Nsfw` 已删除**，论坛不再读取它。这个头在 2026-09-17 规定过，但**从未有任何客户端发送过**（kungal-apps 仓零引用，`git log -S 'X-Kungal-Nsfw'` 只有论坛侧提交），2026-09-18 方向变更又把 App 迁到 v1 的显式 `nsfw=` 查询参数上，所以删除它不影响任何已发布版本。

现在：**Bearer 请求不带任何内容偏好**。`/api/v1` 的每个结果集都由显式查询参数 `include_nsfw` 决定（01-standard §3），不读偏好 cookie、请求头，也不读账号分级；App 按账号设置自己传 `include_nsfw`。

- 2026-09-23 到 2026-09-24 之间，服务端曾用 Bearer token 回源 `/oauth/userinfo` 解析账号分级（`middleware.BearerStance`，Redis 键 `kungal:bearer-stance:<sub>`，TTL 5 分钟），但只有旧路由读它；旧路由退役后它随死代码清理（γ）删除，残留缓存键靠 TTL 自然过期。
- 「优先显示原名」偏好仍只认 `KUNGalgameSettings` cookie。

## 4. 版本闸：`GET /api/v1/app/version`

公开接口，不带信封（2026-09-23 随 X1a 从 `/api/app/version` 迁来，形状只多了 `object`）：

```json
{ "object": "app_version", "min_version": "0.1.0", "latest_version": "0.1.0", "notes": "",
  "downloads": { "android": "…", "ios": "…", "windows": "…", "linux": "…" } }
```

- 数据来自环境变量 `KUN_APP_MIN_VERSION` / `KUN_APP_LATEST_VERSION` / `KUN_APP_RELEASE_NOTES` / `KUN_APP_DOWNLOAD_{ANDROID,IOS,WINDOWS,LINUX}`，改了要重启 kungal-api。
- 版本号必须是 `MAJOR.MINOR.PATCH`；格式不对或 min 高于 latest，服务**启动失败**。否则 App 会把所有用户强更到一个不存在的版本。
- 某个平台没配下载地址时，回落到 `https://www.kungal.com/app`，所以 App 总能拿到一个可以打开的链接。下载页会把这种平台显示为「即将推出」。

## 5. App Links 与网页

- 上线地址 `https://www.kungal.com/.well-known/assetlinks.json`（源文件 `apps/web/public/.well-known/assetlinks.json`）：`com.kungal.app`，**指纹是全 0 占位**，等 App 侧提供直发签名钥的 SHA-256 后替换。本地验证返回 200 + `application/json`，没有重定向。
- 裸域 `kungal.com` 会 302 到 www，而 Android 验证不跟随跳转，所以 App 的 intent-filter **只能声明 `www.kungal.com`**。
- `/app`：下载页，数据来自版本闸接口。
- `/app/oauth/callback`：App Links 验证失败或没装 App 时的落地页。页面提示「请在 App 中完成登录」，不消费授权码，挂载后把 `code` / `state` 从地址栏清掉，并带 `noindex` 和 `referrer: no-referrer`；已加入 sitemap 排除列表。

## 6. 上线

- **不需要数据库迁移**（幂等键存在 Redis 里）。
- `docker-compose.prod.yml` 已写入 `KUN_OIDC_ISSUER`、`KUN_OIDC_JWKS_URL`（内网 `http://oauth:9277/oauth/jwks`，已从生产网络实测可达）和 `KUN_APP_*`。
- `KUN_BEARER_CLIENT_IDS=kungal-app,kungal-app-desktop` 直接写在 compose 里（client_id 是公开标识，和 `OAUTH_CLIENT_ID` 同样处理），不走 Dokploy 面板。以后 infra 新增一方 client 时改这一行；要紧急关掉 Bearer 通道，就把它改成空值后重新部署。

## 7. 没做 / 已知缺口

- **ETag / 304**：论坛 API 目前没有（线上响应头里没有 `etag`，Cloudflare `DYNAMIC`）。
- **429**：来自 Cloudflare，**不带信封**，App 要按 HTTP 状态码处理。
- **catalog `/v2/me` 的按 IP 限额**：上游把用户 token 也按客户端 IP 分桶，而论坛只有一个出口 IP。App 用户的请求经论坛转发，会加重这个已知问题；论坛侧无法修复。
- **论坛面 OpenAPI 化（路 A）**：尚未开始；App 先用手写薄客户端。
