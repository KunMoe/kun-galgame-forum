# X1d · 图片上传

> X1 轨第四段，2026-09-23，分支 `api-v1/x1-image`，迁移号段 195–201（本段不用）。总览见 [x1a](x1a-friend-link-app.md) 开头。
> 契约与变异题同一个提交，早于实现。

## 1. 现状与生产取值（2026-09-23 20:10 Asia/Shanghai，只读）

| 事实 | 数字 |
|---|---|
| 今天上传过图的人 / 张数 | 3 人 / 6 张（单人最多 4 张） |
| 碰到每日 50 张上限的人 | 0；超过 50 的 0（旧面的「先查后加」有竞态，但线上没被踩过） |
| `kungal_user_state` 行数 | 100,834（行在**网页首次登录**时懒建，只用 App 的账号可能没有） |
| 图床给论坛 client 的单文件上限 | 10 MiB（`image_max_file_size`，infra 侧配置） |
| 服务器请求体上限 | 10 MiB（Fiber `BodyLimit`，全站一个值） |

## 2. 调用方

| 旧路由 | 网页 | Nitro | App | 外部 |
|---|---|---|---|---|
| `POST /image/topic` | `composables/useKunEditorAdapters.ts`（编辑器粘贴/拖放/工具栏，经 `@kungal/editor-*` 注入的 `uploadImage` 适配器）；`components/edit/topic/CoverPicker.vue`（话题封面） | 无 | 无 | `../kun-editor` 只有文档示例写着这个路径，代码只认注入的适配器（`file → string`） |
| `POST /image/message` | `components/message/pm/Container.vue`（私信内联图） | 无 | 无 | — |
| `POST /image/cover` | `components/KunCoverUpload.vue`（文档横幅、友链横幅、网站图标）；`components/KunImagesUpload.vue`（抽奖奖品图，读 `sexual`） | 无 | 无 | — |
| `POST /image/galgame` | `utils/uploadGalgameImage.ts` → `constants/galgameEdit.ts`（封面/截图行）、`components/edit/galgame/Footer.vue`（横幅） | 无 | 无 | — |

App（`../kungal-apps`）不调这四条。

## 3. 普查抓到的旧面问题

| # | 问题 | 去向 |
|---|---|---|
| 1 | 每日配额「先查后加」不是原子的，并发能越过 50；自增的错误被丢弃 | 预占：`UPDATE … SET n = n + 1 WHERE n < 50 RETURNING`，上传失败就退回 |
| 2 | 没有 `kungal_user_state` 行（只用 App 的账号）→ `500 查询用户失败` | 预占前先 `Ensure` 这一行 |
| 3 | 四条路由三种响应：裸字符串 token、`{hash,url,…,sexual:int}`、`{hash,url,…,size_bytes,deduplicated}` | 全部回 `Image`（B10：图片一个类型，处处如此） |
| 4 | `/image/cover` 名不副实：不裁切，就是 `topic` 预设外加一次 `sexual` 查询 | 并进 `POST /images`，每次上传都查一次 `sexual` |
| 5 | 图床的审核拒绝（60002）、额度（80008）、鉴权（80001–80005）都被类型断言漏掉，一律 `500 图片上传失败`；其余错误把图床的数字码和状态原样透给客户端 | §6 逐条映射 |
| 6 | 服务端不看 MIME，全交给图床 | 文件分片的 `Content-Type` 不是 `image/*` → `415`（同 `PUT /me/avatar`） |
| 7 | **全站 v1 的缺陷，两条路**：请求体超过 Fiber 的 `BodyLimit` 时抛 `ErrRequestEntityTooLarge`，`WriteFiberError` 把它包成 `500 INTERNAL_ERROR`；JSON 请求体超过 huma 的 1 MiB 上限时 huma 回 413，`problem.StatusToCode` 不认识 413，也落到 `500 INTERNAL_ERROR`。两条都记错误日志。01 §8 要求 v1 下每个失败都是 problem 体，状态也该是 413 | 新增平台码 `PAYLOAD_TOO_LARGE`（413），两条路都映射到它 |
| 8 | 10 MiB 的文件单独就碰到 10 MiB 的请求体上限，multipart 的边界和字段让它进不来：旧面写着「不能超过 10MB」，实际是「比 10MB 少几百字节」 | `BodyLimit` 加 64 KiB 余量，文件上限仍是 10 MiB |
| 9 | galgame 上传：catalog 回的 `variant_urls` 解码时丢了、`thumbhash` 解码后丢了 | `Image.thumbhash` 带上；变体不建模（调用方只读 `hash`） |

## 4. 逐条去向

| # | 旧路由 | v1 | 档 |
|---|---|---|---|
| 1 | `POST /image/topic` | `POST /api/v1/images`，`purpose=content` | required |
| 2 | `POST /image/cover` | `POST /api/v1/images`，`purpose=content` | required |
| 3 | `POST /image/message` | `POST /api/v1/images`，`purpose=message` | required |
| 4 | `POST /image/galgame` | `POST /api/v1/work-edit-images` | required |

4 条旧路由全删，`legacy_route_baseline` 下调 **4**。

## 5. 形状

### 5.1 `POST /api/v1/images`（`createImage`）→ `201 Image`

请求体 `multipart/form-data`：

| 字段 | 约束 |
|---|---|
| `file` | 必填；分片 `Content-Type` 必须是 `image/*`；≤ 10 MiB（图床给论坛的单文件上限） |
| `purpose` | 必填；封闭枚举 `content` / `message` |

- `purpose` 决定图床预设：`content` → `topic`（站内公开的一切：正文、话题封面、文档与友链横幅、网站图标、抽奖奖品），`message` → `message`（私信）。两个预设今天都只存主图，但图床把它们分开登记，v1 不把它们合并。
- 响应：`201` + `Location: <Image.url>` + `Image`。图片本身就是这次创建出来的资源，它的地址就是图床上的 `url`；论坛没有、也不需要 `GET /images/{hash}`。
- `Image.sexual`：上传后查一次图床的元数据。**新图在夜跑分级之前一定是 `null`**，只有去重命中了已分级的旧图才有值；查询失败也是 `null`（装饰性，不让上传失败）。
- `Image.thumbhash`：图床给了就带，空串 → `null`。
- **持久化形态不变**：客户端存进正文、封面列的仍是 `/image/<hash>` token 或裸 hash，由客户端用 `Image.hash` 拼出来；`Image.url` 只用于预览。GC 的 reference-ping 只认 token 与按列名找到的裸 hash 列（#200），存 `url` 会让图被回收。
- 每日配额：`content` 与 `message` 合用一个计数，每人每天 50 张（北京时间零点清零，沿用旧 cron）。预占在上传之前，上传失败（任何原因）退回；预占失败 → `429 IMAGE_DAILY_LIMIT_REACHED`（`limit: 50`）。
- 幂等：`Idempotency-Key` 可选（K12）。**重试必须重发同一份字节**：指纹含请求体，而 multipart 的 boundary 每次构造 `FormData` 都会变，换一份 `FormData` 重试 → `409 IDEMPOTENCY_KEY_REUSED`。内容寻址的上传本身就是幂等的（同一张图同一个 hash），键只省下一次配额。

### 5.2 `POST /api/v1/work-edit-images`（`createWorkEditImage`）→ `201 Image`

请求体 `multipart/form-data`：

| 字段 | 约束 |
|---|---|
| `file` | 必填；`image/*`；≤ 10 MiB |
| `preset` | 必填；封闭枚举 `cover` / `screenshot`（catalog 的叫法与取值，原样透传） |

- 用调用者自己的 OAuth 访问令牌代理 catalog `POST /v2/me/edit-images`：图挂在调用者的 catalog 身份下，供编辑提案里的封面/截图行引用（`hash`）。
- 不占每日配额（catalog 自己限流）。
- 响应：`201` + `Location: <Image.url>` + `Image`（`sexual` 恒 `null`：catalog 的上传回执不带分级）。
- 旧面的 `galgame_banner` / `galgame_screenshot` 不再接受。

## 6. 错误码

新增两个：

| code | 状态 | 域 | 何时 |
|---|---|---|---|
| `PAYLOAD_TOO_LARGE` | 413 | platform | 请求体超过上限，在任何 v1 操作上：JSON 请求体 1 MiB（huma 的每操作上限），其余 10 MiB + 64 KiB（服务器上限） |
| `IMAGE_DAILY_LIMIT_REACHED` | 429 | kungal | 今天的 50 张用完了；扩展 `limit` |

两个操作共有：

| 情况 | 回 |
|---|---|
| 没有凭证 / 凭证无效 | `401 MISSING_CREDENTIAL` / `INVALID_CREDENTIAL` |
| 请求体不是 `multipart/form-data`，或文件分片不是 `image/*` | `415 UNSUPPORTED_MEDIA_TYPE` |
| 缺 `file` / 缺枚举字段 | `422 VALIDATION_FAILED`，`/file` 或 `/purpose`、`/preset` → `REQUIRED` |
| 枚举字段取值不在表里 | `422 VALIDATION_FAILED` → `UNKNOWN_VALUE` |
| 文件 > 10 MiB（请求体没超上限时由操作自己判） | `422 VALIDATION_FAILED`，`/file` → `TOO_LONG`（`max_length`） |
| 请求体超服务器上限 | `413 PAYLOAD_TOO_LARGE` |

`createImage` 的上游（图床）：

| 图床 | 回 |
|---|---|
| 图片客户端没配置 | `503 SERVICE_UNAVAILABLE` |
| 80007 超过站点上限 | `422 VALIDATION_FAILED`，`/file` → `TOO_LONG` |
| 80009 预设不接受这种格式 | `415 UNSUPPORTED_MEDIA_TYPE` |
| 60002 审核拒绝 | `422 CONTENT_REJECTED` |
| 80008 站点额度用尽、80001–80005 论坛自己的凭证问题、不可达、其余任何错误 | `503 SERVICE_UNAVAILABLE` |

`createWorkEditImage` 的上游（catalog）：

| catalog | 回 |
|---|---|
| 没配置 | `503 SERVICE_UNAVAILABLE` |
| 401 | `401 INVALID_CREDENTIAL`（论坛替用户持有的令牌不被接受，要重新登录） |
| 403 缺 scope | `403 SCOPE_REQUIRED` |
| 其余 403 | `403 PERMISSION_REQUIRED` |
| 413 | `422 VALIDATION_FAILED`，`/file` → `TOO_LONG` |
| 415 | `415 UNSUPPORTED_MEDIA_TYPE` |
| 422 | `422 VALIDATION_FAILED`，`/file` → `INVALID_FORMAT`，`detail` 带 catalog 的原话 |
| 429 | `429 RATE_LIMITED` |
| 其余 4xx、5xx、不可达 | `503 SERVICE_UNAVAILABLE` |

## 7. 共享面改动

1. **`PAYLOAD_TOO_LARGE`**：平台码，登记进 registry；`apiv1.WriteFiberError` 遇到 Fiber 的 413、`problem.StatusToCode` 遇到 huma 的 413，都回它，不再是 `500 INTERNAL_ERROR`。影响所有 v1 操作：今天任何过大的请求体都是一条 500 加一行错误日志。
2. **`BodyLimit`**：`10 MiB` → `10 MiB + 64 KiB`。旧路由的处理器各自还在查文件大小，不受影响。

## 8. 网页

| 文件 | 改什么 |
|---|---|
| `composables/useKunEditorAdapters.ts` | `POST /images`（`purpose=content`），返回 **`/image/<hash>`**（拼 token，不是 `url`）；编辑器适配器的 `file → string` 契约不变 |
| `components/edit/topic/CoverPicker.vue` | 同上，推进 `coverImages` 的仍是 token |
| `components/message/pm/Container.vue` | `purpose=message`，`![name](/image/<hash>)` |
| `components/KunCoverUpload.vue` | `hash` 进 `v-model`、`url` 做预览（字段名不变） |
| `components/KunImagesUpload.vue` | `sexual >= 2` 改为 `sexual === 'explicit'` |
| `utils/uploadGalgameImage.ts`（+ spec） | `POST /work-edit-images`，`preset` 从 `galgame_banner`/`galgame_screenshot` 改成 `cover`/`screenshot`；手写的结果类型删掉，用生成的 `Image` |

## 9. 变异题（先于实现提交）

| # | 破坏 | 必须变红的断言 |
|---|---|---|
| 1 | 不查每日配额 | 第 51 张 → `429 IMAGE_DAILY_LIMIT_REACHED`，`limit` 50 |
| 2 | 上传失败不退回预占 | 图床失败一次后计数不变 |
| 3 | 配额改回「先查后加」 | 计数 45 时并发 20 个上传，恰好 5 个成功、15 个 429，计数停在 50 |
| 4 | 没有状态行时不 `Ensure` | 库里没有该用户的 `kungal_user_state` 行也能上传，之后计数为 1 |
| 5 | `purpose` 到预设的映射错位 | 假图床收到的 `preset`：`content` → `topic`，`message` → `message` |
| 6 | 不查文件大小 | 10 MiB + 1 字节的文件 → `422`，`/file` `TOO_LONG` |
| 7 | `WriteFiberError` 不映射 413 | 11 MiB 的请求体 → `413 PAYLOAD_TOO_LARGE`，problem 体 |
| 7b | `StatusToCode` 不映射 413 | 任一 JSON 写操作收到 2 MiB 的请求体 → `413 PAYLOAD_TOO_LARGE` |
| 8 | 不查文件分片的 MIME | `text/plain` 分片 → `415` |
| 9 | 审核拒绝映射成 503 | 图床回 60002 → `422 CONTENT_REJECTED`，计数退回 |
| 10 | 图床鉴权失败透传成 4xx | 图床回 80003 → `503` |
| 11 | 不查 `sexual` | 假图床对该 hash 报 2 → `sexual: "explicit"` |
| 12 | `preset` 透传错位 | 假 catalog 收到的 `preset` 与请求一致 |
| 13 | 转发论坛自己的凭证而不是用户令牌 | 假 catalog 收到的 `Authorization` 是调用者的令牌 |
| 14 | catalog 缺 scope 映射成 503 | catalog 回 403 `SCOPE_REQUIRED` → `403 SCOPE_REQUIRED` |
| 15 | 网页编辑器适配器返回 `url` | 适配器的返回值是 `/image/<hash>`（vitest） |

图床与 catalog 都用 `httptest.Server` 假冒，按请求回数据并记录收到的表单与头。

## 10. 迁移

**无。**

## 11. 实现时对本契约的修正（2026-09-23，只增不改）

1. **审核拒绝回 `IMAGE_REJECTED`（新 code，kungal 域，422），不是 `CONTENT_REJECTED`**。`CONTENT_REJECTED` 在注册表里的定义是「信任与安全检查拒绝了提交的**文字**」，网页给它的文案是「内容包含违禁词」；拿它回图片审核，会告诉用户图里有违禁词。新增的 code 因此是三个，不是两个。
2. **`PAYLOAD_TOO_LARGE` 进 kungal 域，不是 platform**。01 §3：通用失败复用 infra 的 code 与 type URI，infra 没有的进 kungal 域。infra 的注册表没有 413 的 code，它自己的 `StatusToCode` 同样没有 413，所以 infra 的 JSON 请求体超过 huma 的 1 MiB 也是 500。这条应当提给 infra 进 platform 域，与 `IDEMPOTENCY_REQUEST_IN_PROGRESS` 同一处境。
3. **G4 的推导多了 413**：`RequiredStatuses` 给每个带请求体的操作都推导出 413，02 的 G4 行同步改成「有请求体 400 + 413 + 415 + 422」。§7 只写了注册表与 `BodyLimit` 两处，漏了这一处：413 是任何带请求体的操作都真实会发的状态，G4 要求声明。spec 因此在每个写操作上多出一个 413 响应。
4. **两个操作挂了一个操作中间件 `requireMultipart`**。huma 在处理器之前读 multipart 表单，请求体不是 multipart 时它报的是 body 位置的校验错误，映射出来是 `422 VALIDATION_FAILED`，不是 §6 写的 415。中间件先看 `Content-Type`，不是 `multipart/form-data` 就回 415。
5. **上传不查封禁**。v1 的内容写操作都先向账号服务确认调用者没被封禁；上传没有照做：旧面不查，查了会让账号服务故障连带上传失败，而引用这张图的那次写（话题、回复、私信）本来就会拦下被封禁的账号。
6. **`catalogclient` 的旧预设映射删了**。`mapEditImagePreset` 只为把 `galgame_banner` / `galgame_screenshot` 翻成 catalog 的 `cover` / `screenshot`；旧路由删掉之后它是恒等函数，客户端直接透传 `preset`。
7. **变异 3 按旧面的真实语义重写**：「先数、上传、成功后再加一」，窗口是整个上传过程，而不是两条 SQL 之间的几微秒。测试里假图床每次上传睡 30 ms。
8. **`PAYLOAD_TOO_LARGE` 最终进 platform 域**（取代第 2 条）。infra#294（9605ffae，2026-09-23 13:08Z 部署）把它加成平台码，论坛逐字照抄 infra 注册表：`platform`、413、`https://developer.nextmoe.dev/problems/platform/payload-too-large`、title「Payload too large」、描述「The request body is larger than this operation accepts. Retrying the same body cannot succeed.」。kungal 域的那条删了。

### 变异执行结果

16/16 杀（含 7b）。第一次跑 12 号没编译过（写死 `"cover"` 让 `preset` 成了未使用变量），改成「两个取值都映射到 `cover`」重跑，被杀。

| # | 结果 |
|---|---|
| 1 | `TestV1ImagesDailyLimit`、`TestV1ImagesQuotaIsAtomic` |
| 2、9、10 | `TestV1ImagesUpstreamFailuresRefund` |
| 3 | `TestV1ImagesQuotaIsAtomic` |
| 4 | `TestV1ImagesProvisionTheStateRow` |
| 5、11 | `TestV1ImagesShapeAndPresets` |
| 6、8 | `TestV1ImagesRequestChecks` |
| 7、7b | `TestV1OversizedBodiesAre413` |
| 12、13 | `TestV1WorkEditImages` |
| 14 | `TestV1WorkEditImageUpstreamErrors` |
| 15 | `app/utils/uploadImage.spec.ts`（vitest） |

7 号那条测试走真实连接：`app.Test` 拒绝发出超过 `BodyLimit` 的请求体，而边写边读的客户端会和服务端的关连接赛跑（第一次通过，第二次读到 `connection reset by peer`）。服务端只凭 `Content-Length` 就拒绝，所以测试只发请求头。

### 实测（开发库 + 本机 infra 图床与 catalog，worktree 构建，无头 Chromium）

- **真图床**：`POST /images` 传一张 64×64 的 PNG → `201`，`Location` 与 `url` 相同，`hash`、`width`、`height`、`thumbhash` 都是图床给的真值，`sexual` 为 `null`（新图）；开发库里该用户的计数随之加一。
- **真 catalog**：`POST /work-edit-images` → `403 SCOPE_REQUIRED`。登录接口签的令牌不带 `catalog:edit`，拿不到带这个 scope 的令牌，成功路径只由假 catalog 的库测试覆盖；这条客户端代码与旧路由相同，只删了预设翻译。
- **浏览器**（8/9）：
  - 话题编辑器「插入图片」→ 文档里的 `<img src>` 是 `/image/<hash>`；
  - 发布弹窗的封面选择器上传 → 预览显示该图；持久化的编辑器状态（`KUNGalgameEditTopic`）与图片历史（`KUNGalgameImageHistory`）里只有 `/image/<hash>` token，没有 CDN 地址；
  - 私信附图 → `purpose=message`，待发送的图用 token；没有发出这条私信；
  - 全程没有请求旧的 `/api/image/*`。
  - 唯一的失败是一条 Vue 水合不一致：服务端用测试手写的用户 cookie 渲染了头像与用户名（`devtest-3`），客户端换成了账号的真实资料。这是测试脚手架的问题，与本段无关。
- 开发库的 URL 用的是 `ImageCDNBase`，它在开发环境指向线上 CDN，所以预览图在开发环境是 404；断言一律比 hash 与 token，不看图片是否加载出来。
- 测试完删了会话键（再请求回 401）、计数清零。
