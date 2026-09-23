# G1 · 工具集

> G 轨第一段（G0 改号之后），2026-09-23。`internal/toolset` 的 **16** 条旧路由：列表 / 用户列表 / 详情 / 写 3 + 资源 4 + 分片上传 4 + 实用性 2。迁移号 **142**（G 号段 140–159；140、141 已由 G0 占用）。
> 契约与变异题同一个提交，早于实现。普查全文在 [census/galgame-resources-toolsets.md](census/galgame-resources-toolsets.md) §1、§4–§8、§9–§13，结论已对过代码。评论墙已在 RC（D18 / D21），本轨不重做。

## 1. 普查（生产实测 2026-09-23）

### 1.1 数字

| 事实 | 数字 |
|---|---|
| `galgame_toolset` | 119 行，全部 `status = 0`；最新创建 2026-09-19；9 行 `comment_count > 0` |
| `type` | extractor 46、others 19、emulator 16、converter 11、launcher 10、script 10、translator 6、debug 1（`engine` / `docs` 各 0） |
| `language` | zh-cn 110、en-us 7、ja-jp 1、others 1（`zh-tw` 0） |
| `platform` | windows 101、emulator 9、others 8、mac 1（`linux` 0） |
| `version` | stable 110、alpha 5、beta 3、rc 1 |
| `galgame_toolset_resource` | s3 84（82 有 `artifact_uuid`，12 行 `content` 为空）、user 26（2 行 `content` 为空）。全部 `status = 0`。10 行的发布者不是工具作者 |
| `galgame_toolset_practicality` | 61 行，**(toolset_id, user_id) 重复 0**；rate 1:3、3:1、4:1、5:56 |
| `galgame_toolset_alias` | 50 |
| `galgame_toolset_contributor` | 124 |
| `galgame_toolset_category_relation` | **0** |
| `kungal_user_state.daily_toolset_upload_bytes > 0` | 0 个用户 |
| 作者与资源发布者 | 48 个不同用户，OAuth `status` 全部为 0 |

### 1.2 调用方

- `apps/web/server/**`：零 API 调用。
- Flutter App（`../kungal-apps`）：Dart 源零命中。`docs/tickets/02-forum-direct-api-prereqs.md:86` 与论坛 `docs/proj/app-direct-api.md:120` 把 `GET /api/toolset/:id/resource/detail` 列为前瞻契约，不是现网调用。
- 网页：
  - 浏览 `components/toolset/card/{Container,Nav,Card}.vue`（`pages/toolset/index.vue`）
  - 搜索 tab `components/search/List.vue`（`type==='toolset'` 时打的是 `GET /toolset`，不是 `/search`）
  - 详情 `pages/toolset/[id]/index.vue` + `components/toolset/Detail.vue` + `PracticalityChart.vue`
  - 资源 `components/toolset/resource/{List,Item,Container,LinkForm,Upload,ResumeList}.vue`
  - 编辑 `pages/edit/toolset/{create,rewrite}.vue` + `components/edit/toolset/{Toolset,Rewrite}.vue` + `rewriteStore.ts`
  - 用户页 `pages/user/[id]/toolset.vue` + `components/user/Toolset.vue`
  - 评论墙已经走 RC：`components/toolset/comment/CommunityContainer.vue` → `useCommunityCommentList({ kind: 'toolset' })` → `GET /api/v1/wall-comments?subject_type=toolset&subject_id=…`（D18）
- 别的域读这些表但不走这些路由：首页动态（`trg_feed_galgame_toolset` / `_resource`）、删号预览、`kungal_user_state` 日额度 cron。动态触发器仍有效；本轨收窄它的 `UPDATE OF` 列（E14），不改 feed 形状。

### 1.3 旧面的问题（普查编号 E*，对应 census 发现号）

| 编号 | 事实 |
|---|---|
| E1 | 路径参数大量是假的（census §1.1）：`GET/PUT/DELETE …/resource` 与 `…/resource/detail` 读 query/body 的 `toolset_resource_id`；upload complete/resume/abort 读 body `artifact_uuid`；init 的 `toolsetID` 传到 service 后未使用。`GET /toolset/999/resource/detail?toolset_resource_id=1` 返回 1 号资源 |
| E2 | `GET …/resource/detail` 挂 public 组、无 OptionalAuth，匿名下发预签名 URL、提取码、解压密码，并且每次 GET `download + 1`；artifact 换链失败 slog.Warn，仍回库里的旧 `content`（census §5.1，发现 24） |
| E3 | 全部 POST 不读 `Idempotency-Key`（发现 45） |
| E4 | `galgame_toolset.status` / `galgame_toolset_resource.status` 119/119 与 110/110 都是 0，全仓没有写成别的值的路径；列表却 `WHERE status != 1`（发现 35；K26 先例） |
| E5 | 未知 `sort_field` 静默回落 `created`，未知 `sort_order` 除 `asc` 外全是 `DESC`，未知 `type=` 精确匹配空集（发现 36） |
| E6 | handler 里 `Page==0 → 1`、`Limit==0 → 24` 在 `validate:"min=1"` 之后，死代码（发现 37） |
| E7 | `CountFiltered` 含封禁作者，`GetList` 再 `continue` 掉 `!IsRenderable`，`total` 与 `items` 不同谓词（发现 38） |
| E8 | 详情贡献者循环不跑 `IsRenderable`（发现 39） |
| E9 | `homepage` `json.Marshal` / `Unmarshal` 丢错（发现 40） |
| E10 | type/language/platform/version 后端开放、前端封闭（发现 41） |
| E11 | 实用性表无 `(toolset_id, user_id)` 唯一；`Upsert` 先 Find 再 Create（发现 42）。生产重复 0 |
| E12 | 实用性 GET/PUT 不验证工具存在（发现 43） |
| E13 | `FindUserRating` 非「找不到」的错误被丢掉（发现 44） |
| E14 | GET 详情 `IncrementView` 走 GORM `Update`，刷 `updated`，并触发 `trg_feed_galgame_toolset`（发现 46）。资源 `IncrementDownload` 同理刷资源行并触发 `trg_feed_galgame_toolset_resource` |
| E15 | `(toolset_id, content)` 唯一让同一工具第二个空 content 的 s3 资源 500（发现 31）。生产 12 行 s3 `content = ''`；同一工具是否两行空 content 以迁移前查询为准 |
| E16 | 任何人登录都可往别人的工具下 POST 资源，成功后 `AddContributor`（发现 32）。生产 10 行，保留 |
| E17 | 资源创建萌萌点 `ref` 是 `toolset:<toolsetID>`，删除是 `toolset_resource:<id>`（发现 33） |
| E18 | 网页删工具按 `3 + 资源数 × 3` 警告，后端只扣 3（发现 34） |
| E19 | complete/resume/abort 不校验上传归属（发现 27）；init 不检查工具是否存在（发现 26） |
| E20 | Abort 在 artifact.Delete 失败时仍 200（发现 28） |
| E21 | `firstComplete` Redis SETNX 出错返回 true，额度可被加两次（发现 29） |
| E22 | 上传额度读 `kungal_user_state.moemoepoint` 缓存（发现 30，C3） |
| E23 | `userclient.Hydrate` / `User` 丢错 fail-open，占位用户 `Status==0` 可渲染（census §1.3） |
| E24 | 详情嵌 `comment_preview`（community 失败 slog.Warn 后空数组）；RC 之后墙已经在 `wall-comments` |

## 2. 范围与寻址

路径段就是目标。资源或上传不属于路径里的 `{toolset_id}` → `404 NOT_FOUND`（修 E1）。G17：写路径里以 `{x_id}` 结尾的那段必须有 GET 且 200 带 `id`；本轨因此多一条 `GET /toolsets/{toolset_id}/resources/{resource_id}`（任务书未列，见 §8 O1）。

| # | 旧 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /toolset` | `GET /api/v1/toolsets` 页码集合（K11），默认 `limit` 24，最大 100 | optional |
| 2 | `GET /user/:id/toolsets` | `GET /api/v1/users/{user_id}/toolsets` 页码，U3 形 | optional |
| 3 | `GET /toolset/:id` | `GET /api/v1/toolsets/{toolset_id}` | optional |
| — | — | `GET /api/v1/toolsets/{toolset_id}/source` 编辑源 | required |
| 4 | `POST /toolset` | `POST /api/v1/toolsets` → **201** + `Location` + `Toolset`；`Idempotency-Key` **必带**（K12） | required |
| 5 | `PUT /toolset/:id` | `PATCH /api/v1/toolsets/{toolset_id}` 部分更新；只有改过的文本走 trust（K18） | required |
| 6 | `DELETE /toolset/:id` | `DELETE /api/v1/toolsets/{toolset_id}` → **204** | required |
| — | — | `GET /api/v1/toolsets/{toolset_id}/resources/{resource_id}`（G17） | optional |
| 7 | `POST /toolset/:id/resource` | `POST /api/v1/toolsets/{toolset_id}/resources` → **201**；`Idempotency-Key` **必带** | required |
| 8 | `PUT /toolset/:id/resource` | `PATCH /api/v1/toolsets/{toolset_id}/resources/{resource_id}` | required |
| 9 | `DELETE /toolset/:id/resource` | `DELETE /api/v1/toolsets/{toolset_id}/resources/{resource_id}` → **204** | required |
| 10 | `GET /toolset/:id/resource/detail` | `POST /api/v1/toolsets/{toolset_id}/resources/{resource_id}/downloads` → **200**（K-G3） | optional |
| 11 | `POST /toolset/:id/upload/init` | `POST /api/v1/toolsets/{toolset_id}/uploads` → **201**；`Idempotency-Key` 支持 | required |
| 12 | `POST /toolset/:id/upload/resume` | `GET /api/v1/toolsets/{toolset_id}/uploads/{upload_id}` | required |
| 13 | `POST /toolset/:id/upload/complete` | `PATCH /api/v1/toolsets/{toolset_id}/uploads/{upload_id}` 体 `{ "state": "completed", "parts": […] }` | required |
| 14 | `POST /toolset/:id/upload/abort` | `DELETE /api/v1/toolsets/{toolset_id}/uploads/{upload_id}` → **204** | required |
| 15 | `GET /toolset/:id/practicality` | **删除，无替代**：详情带汇总与 `viewer.practicality_rating` | — |
| 16 | `PUT /toolset/:id/practicality` | `PUT /api/v1/toolsets/{toolset_id}/practicality` 体 `{ "rating": 1..5 }` → **200**（K16 槽） | required |

16 条旧路由全删（含被删除的 GET practicality），`legacy_route_baseline` 下调 **16**。v1 共 **17** 个操作（多了 source 与资源 GET）。

代码位置：`internal/toolset/apiv1/**`；测试 `internal/app/v1_toolset_*_test.go`。`listUserToolsets` 的 OpenAPI tag 是 `users`（U3 同形），其余 `toolsets`。

## 3. 形状

### 3.1 命名

| 旧 | v1 | 为什么 |
|---|---|---|
| 路径 `:id` / 字段无对齐 | `{toolset_id}` / `{resource_id}` / `{upload_id}` / `{user_id}` | 路径参数与字段同名（K1） |
| `created` / `edited` / `updated` | `created_at` / `edited_at` / `updated_at` | 禁用名 |
| `view` / `download` | `view_count` / `download_count` | 计数 `_count` |
| `user` | `author`（工具）/ `poster`（资源） | 禁用名；一条资源里有两个人 |
| `status` 裸整数 | **不下发**（K-G1，无生命周期） | 与 Problem.status 撞型；无人写入 |
| `resource`（数组） | `resources` | 数组用复数 |
| `homepage` | `homepage_urls` | 带 scheme 的 URL 数组 |
| `query` | `q` | 与其它页码集合一致 |
| `sort_field` + `sort_order` | `sort` 封闭 token `<键>_<asc\|desc>` | 01 §4 |
| `type`（资源 `s3`/`user`） | `resource_type`：`file` / `link` | 存储值仍是 `s3` / `user`；v1 按用途命名 |
| `content`（逗号拼接 URL 或预签名） | 链接：`url`（生产 26 条 link 资源每条恰好一个 URL，最长 130，A5 不做数组）；文件的秘密只在 download | GET 不再下发秘密 |
| `code` | `extraction_code` | 顶层 `code` 禁用；这是提取码 |
| `password` | `archive_password` | 解压密码 |
| `size`（s3 字节数字符串 / user 自由文本） | `file_size`（integer，文件）/ `size_label`（string，链接） | 同名两种语义（发现 49） |
| `artifact_uuid` | `artifact_id`（只在创建 file 资源时出现） | 外部 id 说清是谁的；路径上的会话叫 `{upload_id}`，值相同 |
| `filesize`（上传请求） | `file_size` | |
| `rate` | `rating` | |
| `practicality_avg` 类型 `any` | `practicality_average: number \| null` | 无评分时 `null`，两位小数 |
| `rating_counts` map | `practicality_distribution: [5]integer` | 下标 0 = 1 星 |
| `content_markdown` / `content_html`（详情） | 详情 `content: ContentDocument`；源文只在 `GET …/source` | 01 §6 |
| `comment_preview` | **不下发** | 网页改读 `wall-comments?limit=5`（D18） |
| `mine` | `viewer.practicality_rating` | K9 |
| `is_liked` 一类 | 本域没有赞 | — |
| `resource_update_time` | `resource_updated_at` | `_at` |

### 3.2 对象

**`ToolsetSummary`（`object: "toolset"`，列表条目）与 `Toolset`（同 object，详情）重叠字段同名同型（G8）。**

公共字段：`id`、`name`、`aliases`、`type`、`language`、`platform`、`version`、`homepage_urls`、`author`（UserRef）、`view_count`、`download_count`（该工具下资源 `SUM(download)`，无资源为 0）、`comment_count`（本地列，RC 墙写入维护，D21）、`practicality_average`（两位小数的 number，无评分时 `null`）、`practicality_count`、`practicality_distribution`（恰好 5 个非负整数，下标 0 = 1 星）、`created_at`、`updated_at`、`edited_at`（可空）、`resource_updated_at`。

- `resource_updated_at` 列是 `NOT NULL DEFAULT CURRENT_TIMESTAMP`，v1 **从不发 null**（任务书写可空，见 §8 O2）。
- `aliases` / `homepage_urls` 永不 `null`（空数组）。
- `author` 在列表与详情里都是可渲染用户；不可渲染的工具不会出现在列表，详情是 404（K-G7）。

**`Toolset` 额外：**

- `content`：`ContentDocument`，由列 `description` 经 `internal/apiv1/content.Converter.Convert`（`markdown.ParseStored`）产出，与 `internal/doc/apiv1`、话题、墙同一管线。空简介是空文档，不是 `null`。
- `contributors`：`UserRef[]`，不可渲染的丢掉（修 E8）；永不 `null`。
- `resources`：`ToolsetResourceSummary[]`，按 `created_at DESC, id DESC`。
- `viewer`：`can_edit`、`can_delete`、`practicality_rating`（integer 1–5 或 `null`）；匿名为 `null`。Bearer 的两个 `can_*` 不含 staff（K2）。`can_edit` = 作者或 `user.Can(perm.ToolsetEditAny)`；`can_delete` 对 `perm.ToolsetDeleteAny`。任何人登录都可以往别人的工具加资源，这条能力不放进 `viewer`——匿名合法拥有的能力不能做成 viewer 标志（05 §9）。

**`ToolsetSource`（`object: "toolset_source"`）**：`toolset_id`、`content_markdown`（maxLength 2000）。先例是 `WallCommentSource`（`GET /wall-comments/{id}/source`），不是 `TopicSource`：公开 `Toolset` 已经带全部可编辑元数据，源文只要 Markdown。要求 `viewer.can_edit`，否则 `403 PERMISSION_REQUIRED`。档位 required。

**`ToolsetResourceSummary`（`object: "toolset_resource"`）**：`id`、`resource_type`（`file` / `link`）、`file_size`（integer ≥0，`file` 时从 artifact / 存储的字节数来；`link` 时 `null`）、`size_label`（`link` 时用户自由文本，≤107；`file` 时 `null`）、`note`（空串表示没有）、`download_count`、`poster`（UserRef；发布者不可渲染的资源从嵌套列表里丢掉，与 `contributors` 同口径，也与 4.8 / downloads 的 404 一致；生产 48 个发布者全部可渲染）、`created_at`、`viewer`（`can_edit`、`can_delete`；匿名为 `null`）。**不含** `url` / `extraction_code` / `archive_password` / `artifact_id`。

`viewer.can_edit` = 发布者或 `user.Can(perm.ToolsetResourceEditAny)`；`can_delete` 对 `perm.ToolsetResourceDeleteAny`。嵌在详情里的条目同样带 `viewer`（K9）。

**`ToolsetDownload`（`object: "toolset_download"`）**，POST downloads 的 200：

| 字段 | `resource_type=link` | `resource_type=file` |
|---|---|---|
| `url` | 存储的链接 | artifact 预签名 URL |
| `expires_at` | `null` | artifact 的过期时间（不再丢掉） |
| `extraction_code` | 存储的 `code`，没有则空串 | 同左 |
| `archive_password` | 存储的 `password`，没有则空串 | 同左 |

预签名失败 → `503 SERVICE_UNAVAILABLE`，不回库里的旧值（修 E2）。计数 `download_count + 1` 只在本次 200 之后加一次。

**`ToolsetUpload`（`object: "toolset_upload"`）**：`id`（等于 `artifact_uuid` / 路径 `{upload_id}`，UUID 字符串）、`toolset_id`、`filename`、`file_size`、`state`（`pending` / `completed`）、`multipart`、`upload_url`（可空；非分片 pending 时有）、`part_size`（可空）、`parts`（`{part_number, url}[]`，永不 null）、`uploaded_parts`（`{part_number, etag, size}[]`，GET pending 分片时有，否则 `[]`）、`expires_at`（可空）、`created_at`、`completed_at`（可空）。G17 要带 `id` 的 GET，本对象就是。

**`ToolsetPracticality`（`object: "toolset_practicality"`）**，PUT 的 200：`toolset_id`、`practicality_average`、`practicality_count`、`practicality_distribution`、`viewer: { practicality_rating }`（PUT 是 required，viewer 非 null，rating 就是刚写入的 1–5）。前四个字段与 `Toolset` 同名同型。

### 3.3 词表（K-G2）

封闭枚举，取值取自网页常量加上生产里出现过的。存储值不改（`others` 保持这一拼写，无数据迁移）。

| 字段 | 取值 |
|---|---|
| `type` | `emulator` `translator` `extractor` `converter` `debug` `launcher` `script` `docs` `others` |
| `platform` | `windows` `mac` `linux` `emulator` `others` |
| `version` | `stable` `beta` `alpha` `rc` |
| `language` | `zh-cn` `zh-tw` `ja-jp` `en-us` `others` |
| `resource_type` | `file` `link` |
| `sort`（仅 `GET /toolsets`） | `resource_updated_desc`（默认）`resource_updated_asc` `created_desc` `created_asc` `view_desc` `view_asc` `name_asc` `name_desc` |
| 上传 `state` | `pending` `completed`（写面 PATCH 只收 `completed`） |

**`language` 是 01 §3 snake_case 规则的具名例外**：四个取值是小写 BCP 47 标签，WS 已经按同样的标签下发；写成 `zh_cn` 就不再是语言标签。`others` 本身是 snake_case。本契约提交同时把这一行写进 01 §3 的例外表。

未知过滤值 → `400 UNKNOWN_ENUM_VALUE`。缺席 = 不过滤（没有 `all` token）。未知 `sort` → `400 UNKNOWN_SORT`，不得静默回落（修 E5）。

`engine` 不收（A5）：网页 `KUN_TOOLSET_TYPE_CONST` 与 ICON_MAP 里有，但过滤与创建下拉（`KUN_GALGAME_TOOLSET_TYPE_MAP` / `kunGalgameToolsetTypeOptions` / `useToolsetFilters.ToolsetType`）都没有，生产 0 行，没有人能建出一个。网页实现时从 CONST 与 ICON_MAP 删掉。`docs` 同样 0 行，但下拉里选得到，保留。

### 3.4 列表

**`GET /toolsets`**：页码（`collect.PageNumber` 的 `page`/`limit`，本集合把默认 `limit` 改成 **24**，最大 100；`CheckDepth` `page × limit ≤ 10000`，越界 `400 INVALID_PARAMETER` `OUT_OF_RANGE`）。响应 `repr.PageList[ToolsetSummary]`，必发 `total` + `total_relation`。过滤 `type` `language` `platform` `version`，以及 `q`（1–100；缺席或去空白后为空 = 不搜）。`q` 做转义 ILIKE，列是 **`galgame_toolset.name`**（现仓储 `buildListQuery` 只搜这一列，不搜别名或简介）。

排序封闭，默认 `resource_updated_desc`（网页浏览页现在的默认，`useToolsetFilters.ts:34`）。每个排序带同方向的 `id` 决胜键：`created_desc` → `created DESC, id DESC`，`name_asc` → `name ASC, id ASC`，`resource_updated_desc` → `resource_update_time DESC, id DESC`，其余同理。

**不可渲染作者（K-G7）**：先按过滤条件取出作者 id 去重，经 `userclient.Users`（缓存）批量判定，**COUNT 与页查询共用同一个「作者可渲染」谓词**（修 E7）。`userclient` 失败 → `503 SERVICE_UNAVAILABLE`，不得 fail-open（修 E23）。不再有 `status != 1` 过滤（K-G1）。

**`GET /users/{user_id}/toolsets`**（U3 形，规则在 U3 分支，这里摘要）：

- `collect.PageNumber` + `repr.PageList[ToolsetSummary]`，本集合默认 `limit` 24。
- 排序固定 `created_at DESC, id DESC`，没有 `sort`，没有 `relation`（只有一种关系：这个人发的工具）。
- 主人不可渲染 → `404`；`userclient` 失败 → `503`。
- 不过滤 `type`/`q`。主人可渲染但没有工具 → 200，空 `items`，`total=0`。

### 3.5 下载（K-G3）

详情的 `resources` 不带秘密。秘密只从 `POST …/downloads` 来。档位 optional：匿名下载保持今天的行为。每次成功调用 `download_count + 1`（列更新，不刷 `updated`，不触发 feed）。发布者不可渲染、工具作者不可渲染、资源不属于该工具、行不存在 → 同一句 `404 NOT_FOUND`。`userclient` 失败 → `503`。

### 3.6 资源（K-G4）

- `resource_type`：`file`（库 `type='s3'`）与 `link`（库 `type='user'`）。
- **link**：一个 `url`，≤ **1007**（列 `content varchar(1007)` 与 DTO `max=1007`），存进 `content`。方案允许列表与 galgame 下载链接相同：`pkg/utils/validate.go` `downloadlink`（`http` `https` `ftp` `ftps` `magnet` `ed2k` `thunder`）。旧面把 `content` 当逗号拼接的多链接，但生产 26 行每行恰好一个 URL（最长 130），所以不做数组（A5）；将来要多链接是加法。
- **file**：创建时必带 `artifact_id`，必须指向**调用者**在**这个工具**上创建且已 `completed` 的 `toolset_upload` 行，否则 `422 UNKNOWN_REFERENCE`。已被别的资源占用（部分唯一 `artifact_uuid WHERE <> ''`）→ `409 ALREADY_EXISTS`。创建之后不可改 `artifact_id` / `resource_type`。
- 大小：file 下发 `file_size`（integer，来自 artifact / 完成时记下的字节）；link 下发 `size_label`（自由文本 ≤107）。网页的 `ResourceSizePattern`（`kb|mb|gb`）只在客户端，服务端不套。
- `extraction_code`（列 `code`）、`archive_password`（列 `password`）、`note` 均 ≤1007。file 的 PATCH 只能改 `archive_password` 与 `note`（与旧面一致）；link 还可改 `url`、`extraction_code`、`size_label`。
- 任何人登录都可往别人的工具加资源（生产 10 行），成功后成为 contributor。改 / 删需要发布者或既有的 `perm.ToolsetResource*Any`。
- `(toolset_id, content)` 唯一改为部分唯一 `WHERE content <> ''`（修 E15）。12 行 s3 空 content 不受新索引约束。生产实测（2026-09-23）：没有任何工具有两行空 content，非空 `(toolset_id, content)` 重复 0，所以新索引在生产上建得起来。

### 3.7 上传（K-G5）

迁移 142 加表 `toolset_upload`（§5 DDL）。init 写行。GET / PATCH / DELETE 要求行的 `user_id` 是调用者且 `toolset_id` 匹配路径，否则 `404`（与「不存在」同一句）。init 先确认工具存在且作者可渲染（404）；`userclient` 失败 503。

完成：在**同一事务**里把 `completed_at` 写成现在（已有值则不动）并加上日额度。第二次完成不计两次，取代 Redis SETNX（修 E21）。响应 200 `ToolsetUpload`，`state: "completed"`。

Abort：artifact.Delete 失败 → `503`，行不删（修 E20）。已经 `completed` → `409 INVALID_STATE_TRANSITION`（完成后要么绑成资源，要么留着，不能从这条路径删对象）。

额度不变：100 MiB + `kungal_user_state.moemoepoint` × 1 MiB，`perm.ToolsetUploadBypass` 跳过。读的是缓存列（C3），记为已知限制，本轨不改。超额度 → 复用 infra platform **`QUOTA_EXCEEDED`（429）**，带 `Retry-After` 到日额度 cron 的下一次清零（时区用 `cron.ScheduleLocation()`，U1 导出的那个，别自己写死）。论坛注册表目前没有这个码；infra `apps/api/internal/platform/apiv2/problem/registry.go` 有 `CodeQuotaExceeded`，status 429，type URI `https://developer.nextmoe.dev/problems/platform/quota-exceeded`。K4 要求同义码复用，所以**不**新增任务书里的 kungal `UPLOAD_QUOTA_EXCEEDED` / 403（§8 O5）。artifact 侧日配额耗尽也映射到同一码。

文件名必须是 `.7z` / `.zip` / `.rar`（大小写不敏感），≤1007。`file_size` 1–2147483648（2 GiB）。`content_type` 可选，≤100。类型 / 大小不合 → `422 VALIDATION_FAILED` + `errors[]`（`INVALID_FORMAT` / `OUT_OF_RANGE`）。

### 3.8 实用性（K-G6）

迁移 142 加 `(toolset_id, user_id)` 唯一索引（生产 0 重复）。写入 `INSERT … ON CONFLICT DO UPDATE`。工具不存在或作者不可渲染 → `404`。没有 `DELETE`：任务书只给 PUT，不能撤分（§8 O6）。PUT 幂等：同一人同一分再 PUT 是 200、计数不变。

### 3.9 写面限额

取自现行 DTO、网页 zod、列宽，取三者最紧且真实的：

| 字段 | 限额 | 来源 |
|---|---|---|
| `name` | 1–500，K19：先按原始值判长，再去空白；去空白后空 → `TOO_SHORT` | DTO `max=500`，列 `varchar(500)`，zod min 1 |
| `content_markdown` | ≤2000；创建可空（空文档）；去空白后仍可空 | DTO / 列 `varchar(2000)` |
| `aliases` | 最多 **17**，每条 1–500，去空白，请求内唯一 | 网页 `KunTagInput max-tags=17` `max-tag-length=500`；唯一 `(toolset_id, name)`；生产单工具最多 6 个、最长 31 |
| `homepage_urls` | 最多 **10**，每条 ≤500，`http`/`https` URL（zod `z.url().max(500)`；后端旧面不校验，v1 收进来） | 网页；条数旧面无上限，按 WS `urls` 的 10；生产单工具最多 2 条、最长 95、全部 http(s) |
| `type` / `language` / `platform` / `version` | 创建必填封闭枚举 | 网页 zod `z.enum`；后端旧面无 `oneof` |
| link `url` | 1–1007，`downloadlink` | 列 / DTO |
| `size_label` | ≤107；link 创建必填 | 列 / DTO |
| `extraction_code` / `archive_password` / `note` | ≤1007 | 列 / DTO |
| `filename` | 1–1007，`\.(7z\|zip\|rar)$` | 网页 zod |
| `file_size`（上传） | 1–2147483648 | `MaxLargeFileSize` |
| `rating` | 1–5 整数 | DTO `min=1,max=5` |

`PATCH` 出现即替换该字段；`aliases` / `homepage_urls` / `urls` 出现即整组替换，缺席不动。K18：只有本次提交的文本走 trust——工具是 `name`、`content_markdown`、`aliases`；资源是 `urls`（拼接后）、`note`。只改 type/platform/language/version/homepage_urls 或只改 `archive_password` 不跑 trust、不重新入扫描。deny → `422 CONTENT_REJECTED`。hold 仍写入并 `ScanBg`。

信任文本经 `markdown.NormalizeStoredContent`。`homepage` JSON 编解码失败 → `500`，不再吞掉（修 E9）。

创建 / 更新的权限：工具改删 = 作者或对应 `perm.Toolset*Any`。资源见 §3.6。先按 OAuth 当前记录判封禁。

删除工具：硬删（别名、贡献者、实用性、资源、分类关系、本轨新的 `toolset_upload` 随 FK CASCADE）。每个 file 资源先删 artifact（无 uuid 的遗留走 S3 `code`）。任一次对象删除失败 → `503`，事务不提交。萌萌点 −3。网页文案改为只扣 3（修 E18）。

删除资源：权限见上。file 先删 artifact/S3，失败 503 且行还在。萌萌点 −3。

### 3.10 萌萌点

| 事件 | 分 | 幂等键 | ref |
|---|---|---|---|
| 创建工具 | +3 | `toolset_create:<id>` | `toolset:<id>` |
| 删除工具 | −3 | `toolset_delete:<id>` | `toolset:<id>` |
| 创建资源 | +3 | `resource_create:<id>` | **`toolset_resource:<id>`**（与删除对齐，修 E17） |
| 删除资源 | −3 | `resource_delete:<id>` | `toolset_resource:<id>` |

键不变。OAuth 不可用 → 写面 503（既有 pusher 行为，不在本轨改）。

### 3.11 浏览计数

`GET /toolsets/{toolset_id}` 照旧 `view + 1`（话题详情同样在 GET 里计数），改用**不带 `updated` 的列更新**。迁移 142 把 `trg_feed_galgame_toolset` 收窄到 `INSERT OR DELETE OR UPDATE OF name, user_id, status, created`（函数读的列），浏览不再写 `feed_activity`（修 E14）。`trg_feed_galgame_toolset_resource` 同理，不包含 `download`。

## 4. 逐条操作

通用：401 `MISSING_CREDENTIAL` / `INVALID_CREDENTIAL`（required；optional 的坏 Bearer）、403 `ACCOUNT_BANNED`、500 `INTERNAL_ERROR`、503 `SERVICE_UNAVAILABLE`（会话存储 / userclient / artifact / 对象存储）。每个 401 带 `WWW-Authenticate`。v1 全部 `Cache-Control: no-store`。

### 4.1 `listToolsets` · `GET /toolsets` · optional · 200 `PageList<ToolsetSummary>`

Query：`page`、`limit`（默认 24）、`type`、`language`、`platform`、`version`、`sort`、`q`。

| 状态 | code |
|---|---|
| 400 | `UNKNOWN_ENUM_VALUE` `UNKNOWN_SORT` `LIMIT_TOO_LARGE` `INVALID_PARAMETER`（深度 / `q` 格式） |
| 401 | 坏 Bearer |
| 503 | userclient |

### 4.2 `listUserToolsets` · `GET /users/{user_id}/toolsets` · optional · 200 `PageList<ToolsetSummary>`

Query：`page`、`limit`（默认 24）。路径 `{user_id}` 非正整数 → 400 `INVALID_PARAMETER`。

| 状态 | code |
|---|---|
| 400 | `LIMIT_TOO_LARGE` `INVALID_PARAMETER` |
| 404 | 主人不可渲染 / 查无此人 |
| 503 | userclient |

### 4.3 `getToolset` · `GET /toolsets/{toolset_id}` · optional · 200 `Toolset`

副作用：`view_count + 1`，不刷 `updated_at`。

| 状态 | code |
|---|---|
| 404 | 不存在 / 作者不可渲染 |
| 503 | userclient / 正文用户查找 |

### 4.4 `getToolsetSource` · `GET /toolsets/{toolset_id}/source` · required · 200 `ToolsetSource`

| 状态 | code |
|---|---|
| 403 | `PERMISSION_REQUIRED`（无 `can_edit`） |
| 404 | 同详情 |

### 4.5 `createToolset` · `POST /toolsets` · required · 201 `Location` + `Toolset`

`Idempotency-Key` 必带。体 `ToolsetCreate`（§3.9）。`Location`：`/api/v1/toolsets/{id}`。作者自动成为 contributor。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（缺 / 格式错的幂等键） |
| 409 | `IDEMPOTENCY_KEY_REUSED` `IDEMPOTENCY_REQUEST_IN_PROGRESS` |
| 422 | `VALIDATION_FAILED`（`TOO_LONG` `TOO_SHORT` `TOO_MANY_ITEMS` `DUPLICATE_ITEM` `UNKNOWN_VALUE` `INVALID_FORMAT`）`CONTENT_REJECTED` |

### 4.6 `updateToolset` · `PATCH /toolsets/{toolset_id}` · required · 200 `Toolset`

体 `ToolsetPatch`，全部可选。无 `can_edit` → 403 `PERMISSION_REQUIRED`。K18。别名整组替换时撞唯一 → 409 `ALREADY_EXISTS`。

### 4.7 `deleteToolset` · `DELETE /toolsets/{toolset_id}` · required · 204

无 `can_delete` → 403。对象存储失败 → 503，什么都不删。

### 4.8 `getToolsetResource` · `GET /toolsets/{toolset_id}/resources/{resource_id}` · optional · 200 `ToolsetResourceSummary`

G17。不发秘密，不加下载计数。资源不属于该工具 / 不存在 / 工具作者不可渲染 / 发布者不可渲染 → 404。

### 4.9 `createToolsetResource` · `POST /toolsets/{toolset_id}/resources` · required · 201 `Location` + `ToolsetResourceSummary`

`Idempotency-Key` 必带。`Location`：`/api/v1/toolsets/{toolset_id}/resources/{id}`。`file` 必须带 `artifact_id`；`link` 必须带 `url` 与 `size_label`。交叉字段 → `422 INCONSISTENT_WITH`。工具不存在 / 作者不可渲染 → 404。任何人登录均可（E16 保留）。成功：contributor、刷新 `resource_updated_at`、萌萌点 +3（ref `toolset_resource:<id>`）。

### 4.10 `updateToolsetResource` · `PATCH /toolsets/{toolset_id}/resources/{resource_id}` · required · 200 `ToolsetResourceSummary`

无 `can_edit` → 403。file 改 `url` / `extraction_code` / `size_label` → `422` `/url` 等 `IMMUTABLE`。K18。路径工具不匹配 → 404。

### 4.11 `deleteToolsetResource` · `DELETE …/resources/{resource_id}` · required · 204

无 `can_delete` → 403。artifact/S3 失败 → 503，行还在。

### 4.12 `createToolsetDownload` · `POST …/resources/{resource_id}/downloads` · optional · 200 `ToolsetDownload`

空体。K12：支持 `Idempotency-Key`，不强制（不是用户内容）。匿名可省略。成功才 `download_count + 1`。预签名失败 503。可见性同 4.8。

### 4.13 `createToolsetUpload` · `POST /toolsets/{toolset_id}/uploads` · required · 201 `Location` + `ToolsetUpload`

`Idempotency-Key` 支持、不强制。体 `{filename, file_size, content_type?}`。`Location`：`/api/v1/toolsets/{toolset_id}/uploads/{upload_id}`。写 `toolset_upload` 行，`state: pending`。超额度 429 `QUOTA_EXCEEDED`。工具 404。

### 4.14 `getToolsetUpload` · `GET …/uploads/{upload_id}` · required · 200 `ToolsetUpload`

不是自己的 / 工具不匹配 / 不存在 → 404。向 artifact 换新的 part URL（旧 resume）。

### 4.15 `updateToolsetUpload` · `PATCH …/uploads/{upload_id}` · required · 200 `ToolsetUpload`

体 `{ "state": "completed", "parts": [{ "part_number", "etag" }] }`。`state` 不是 `completed` → 409 `INVALID_STATE_TRANSITION`。归属 404。`completed_at` 只写一次，与额度同一事务。

### 4.16 `deleteToolsetUpload` · `DELETE …/uploads/{upload_id}` · required · 204

已完成 409。artifact 失败 503。

### 4.17 `putToolsetPracticality` · `PUT /toolsets/{toolset_id}/practicality` · required · 200 `ToolsetPracticality`

体 `{ "rating": 1..5 }`。不存在 404。`ON CONFLICT DO UPDATE`。没有对应的 DELETE。

## 5. 预分配

### 5.1 迁移 142 `toolset_v1`

普通迁移，随部署跑，幂等。不是 deploy-then-drop。

```sql
-- 1. 上传会话，所有权可查（K-G5）
CREATE TABLE IF NOT EXISTS toolset_upload (
    artifact_uuid varchar(36) PRIMARY KEY,
    toolset_id    integer NOT NULL REFERENCES galgame_toolset(id) ON UPDATE CASCADE ON DELETE CASCADE,
    user_id       integer NOT NULL,
    filename      varchar(1007) NOT NULL,
    file_size     bigint NOT NULL,
    created       timestamp(3) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at  timestamp(3) without time zone
);
CREATE INDEX IF NOT EXISTS idx_toolset_upload_toolset_id ON toolset_upload (toolset_id);
CREATE INDEX IF NOT EXISTS idx_toolset_upload_user_id ON toolset_upload (user_id);

-- 2. 实用性一人一票（K-G6）。生产 0 重复。
CREATE UNIQUE INDEX IF NOT EXISTS galgame_toolset_practicality_toolset_id_user_id_key
    ON galgame_toolset_practicality (toolset_id, user_id);

-- 3. 空 content 的 file 资源不再撞唯一（K-G4 / E15）
DROP INDEX IF EXISTS galgame_toolset_resource_toolset_id_content_key;
CREATE UNIQUE INDEX IF NOT EXISTS galgame_toolset_resource_toolset_id_content_key
    ON galgame_toolset_resource (toolset_id, content)
    WHERE content <> '';

-- 4. 浏览 / 下载不再写 feed（E14）。函数不动，只把触发器收窄到函数真正读的列：
--    feed_sync_galgame_toolset 读 status / user_id / name / created，
--    feed_sync_galgame_toolset_resource 读 note / content / user_id / toolset_id / created。
DROP TRIGGER IF EXISTS trg_feed_galgame_toolset ON galgame_toolset;
CREATE TRIGGER trg_feed_galgame_toolset
    AFTER INSERT OR DELETE OR UPDATE OF name, user_id, status, created
    ON galgame_toolset
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_toolset();

DROP TRIGGER IF EXISTS trg_feed_galgame_toolset_resource ON galgame_toolset_resource;
CREATE TRIGGER trg_feed_galgame_toolset_resource
    AFTER INSERT OR DELETE OR UPDATE OF note, content, user_id, toolset_id, created
    ON galgame_toolset_resource
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_toolset_resource();
```

第 3 步在生产上已查过：非空 `(toolset_id, content)` 重复 0，新唯一建得起来。第 4 步不改函数体（2026-09-23 从生产 `pg_get_functiondef` 核对过列清单），只改触发条件。

### 5.2 错误码

新增（三处一译：`registry.go` + `registry_test.go` + `zh-CN/problem.json`）：

- **`QUOTA_EXCEEDED`**（platform，429，type URI 与 infra 逐字相同）。扩展：无（Retry-After 走头，不走 problem 扩展）。论坛注册表目前没有；本轨第一次引入。

不新增 `UPLOAD_QUOTA_EXCEEDED`。其余复用：`NOT_FOUND`、`PERMISSION_REQUIRED`、`ACCOUNT_BANNED`、`SERVICE_UNAVAILABLE`、`VALIDATION_FAILED`（`INVALID_FORMAT` / `UNKNOWN_REFERENCE` / `INCONSISTENT_WITH` / `DUPLICATE_ITEM` / `TOO_SHORT` / `TOO_LONG` / `TOO_MANY_ITEMS` / `TOO_FEW_ITEMS` / `OUT_OF_RANGE` / `IMMUTABLE` / `UNKNOWN_VALUE`）、`CONTENT_REJECTED`、`ALREADY_EXISTS`、`INVALID_STATE_TRANSITION`、`UNKNOWN_ENUM_VALUE`、`UNKNOWN_SORT`、`LIMIT_TOO_LARGE`、`INVALID_PARAMETER`、`IDEMPOTENCY_KEY_REUSED`、`IDEMPOTENCY_REQUEST_IN_PROGRESS`。

### 5.3 K 决定

| 编号 | 决定 |
|---|---|
| **K-G1** | 没有生命周期。119/119 工具与全部资源 `status = 0`，无人写入别的值（K26 先例）。v1 的工具和资源都没有 `state`。列表不再 `status != 1` |
| **K-G2** | 词表封闭，取自网页常量加生产取值。`language` 的四个 BCP 47 标签是 01 §3 snake_case 的具名例外。未知过滤 400；缺席不过滤。存储值不改 |
| **K-G3** | 下载秘密只从 POST 来，永不从 GET 来。详情不带秘密。`POST …/downloads` 下发秘密并 +1 下载计数。档位 optional。预签名失败 503 |
| **K-G4** | `file` / `link`；link 是一个 `url`，走 downloadlink；file 的 `artifact_id` 必须是调用者在本工具上完成的上传。任何人可给别人的工具加资源。空 content 的唯一改为部分唯一 |
| **K-G5** | 上传会话落本地表，所有权可查。完成与额度同一事务。Abort 失败 503。额度公式不变，读缓存萌萌点（C3）。超额度 `QUOTA_EXCEEDED` 429 |
| **K-G6** | 实用性一人一票唯一索引 + `ON CONFLICT`。详情带平均 / 计数 / 五档分布。`viewer.practicality_rating`。PUT 回 `ToolsetPracticality` |
| **K-G7** | 作者不可渲染：详情 404，列表 COUNT 与页同一谓词排除。userclient 失败 503。贡献者不可渲染则从详情丢掉 |

权限不新增。

## 6. 网页

切到生成的类型化客户端；手写 `shared/types/toolset.ts` 改成生成物别名或删除。`legacy-fetch-baseline` 按删掉的调用点下调（现网 17 处 `kunFetch` / `useKunFetch` 打旧 `/toolset`）。

| 文件 | 改什么 |
|---|---|
| `card/Container.vue` + `useToolsetFilters.ts` + `card/Nav.vue` | `GET /toolsets`；`query` → `q`；过滤不再传 `all`（缺席）；`sort_field`+`sort_order` → 一个 `sort` token；默认 `resource_updated_desc`（与现网一致）；`limit` 24 |
| `search/List.vue` | 工具 tab 改 `GET /toolsets?q=` |
| `pages/toolset/[id]/index.vue` | `GET /toolsets/{id}`；JSON-LD 简介从 `content` 抽纯文本（不再读 `content_markdown`）；评论 JSON-LD 改 `wall-comments?subject_type=toolset&subject_id=&limit=5`；字段改名；`id` 是字符串 |
| `Detail.vue` | 删 `GET …/practicality`；图和星星读详情的 `practicality_*` + `viewer.practicality_rating`；删工具文案改为「消耗 3 萌萌点」；编辑 / 删除按钮读 `viewer.can_*`；加资源按钮：已登录即可 |
| `PracticalityChart.vue` | `counts` map 改读长度为 5 的数组 |
| `resource/Item.vue` | 「获取链接」改 `POST …/downloads`；编辑 / 删除读嵌套 `viewer`；file 的大小读 `file_size`；去掉 `status` 圆点 |
| `resource/LinkForm.vue` | `POST …/resources`，`resource_type` + `url` / `artifact_id`；必带幂等键 |
| `resource/Upload.vue` | init/resume/complete/abort 改 4.13–4.16；`filesize` → `file_size`；complete 体是 `{state:"completed", parts}` |
| `edit/toolset/Toolset.vue` | `POST /toolsets`，201 的 `id` 导航；幂等键 |
| `edit/toolset/Rewrite.vue` + `rewriteStore.ts` | 打开时 `GET …/source` 填 `content_markdown`，元数据来自详情；`PATCH` 只发改过的字段 |
| `user/Toolset.vue` | `GET /users/{user_id}/toolsets`；主人 404 走用户页既有处理 |
| `constants/toolset.ts` | 过滤 UI 去掉 `all`；`sort` 改 token；`KUN_TOOLSET_TYPE_CONST` 与 ICON_MAP 删 `engine`（K-G2） |
| `validations/toolset.ts` | 与 §3.9 对齐或删掉，改信 spec |
| `docs/proj/app-direct-api.md` | 下载入口改 `POST /api/v1/toolsets/{toolset_id}/resources/{resource_id}/downloads` |

`QUOTA_EXCEEDED` 的 zh-CN 译文进 `problem.json`。按钮显隐读 `viewer.can_*`，删掉 `useCan('toolset.edit_any')` 等本地镜像（详情 / 资源条目）。

## 7. 变异题（先于实现提交）

种子：至少 7 个工具同一 `created` 瞬间（跨页边界）；其中两个作者随后在测试里被标成不可渲染；同一工具两个已完成的上传；一个空 content 的 file 资源。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 上传 PATCH 不再核对 `toolset_upload.user_id` 是否为调用者 | 别人完成该会话 → 404，额度不变 |
| 2 | 列表 COUNT 不再排除不可渲染作者 | `total` 等于 SQL 里作者可渲染的行数，且等于本页可走完的条目数 |
| 3 | `userclient` 失败时吞错继续（旧 Hydrate） | 列表 / 详情 / 用户列表 → 503，不回占位作者 |
| 4 | 排序去掉 `id` 决胜键 | 小 `limit` 全量遍历与 SQL 逐条相等、无重无漏（种子有并列 `created`） |
| 5 | 未知 `sort` 回落到 `created_desc` | → 400 `UNKNOWN_SORT` |
| 6 | `PATCH` 不论字段是否变化都跑 trust | 只改 `type`、旧正文已进禁用词表 → 200，库里 type 已改 |
| 7 | 完成上传不写 `completed_at`、额度另事务 | 同一上传第二次 PATCH 后 `daily_toolset_upload_bytes` 加了两遍 |
| 8 | Abort 在 artifact.Delete 失败时仍 204 | → 503，`toolset_upload` 行还在 |
| 9 | PUT practicality 不先查工具 | 不存在的 id → 404，表无新行 |
| 10 | 资源写面 / downloads 不核对路径 `{toolset_id}` | 资源属于别的工具 → 404 |
| 11 | `POST /toolsets` 不要求 `Idempotency-Key` | 缺头 → 400，`header Idempotency-Key` `REQUIRED` |
| 12 | 创建 file 资源不验证上传的 `toolset_id` / `user_id` / `completed_at` | 别人的、别的工具的、或未完成的 `artifact_id` → 422 `UNKNOWN_REFERENCE`，无新行 |

## 8. 开放问题（编排者已裁，2026-09-23）

| # | 事实 | 裁决 |
|---|---|---|
| O1 | G17 要求 `PATCH/DELETE …/resources/{resource_id}` 有同路径 GET，任务书寻址表漏了 | 加 `getToolsetResource`（4.8） |
| O2 | `resource_update_time` 列 `NOT NULL DEFAULT CURRENT_TIMESTAMP` | `resource_updated_at` 从不发 null |
| O3 | `engine` 在 CONST 里、不在任何下拉里，生产 0 行 | 不收（A5），网页删掉（§3.3） |
| O4 | 任务书写 link 1–10 条 URL | 改成一个 `url`：生产 26 条每条恰好一个（最长 130）。不改列类型 |
| O5 | infra 已有 platform `QUOTA_EXCEEDED` 429 | 复用（K4），不新增 kungal 专码 |
| O6 | K16 槽位通常有 `DELETE` | 只有 PUT：网页没有撤分入口，A5。要撤分时加 `DELETE` 是加法 |
| O7 | `TopicSource` 带全部可编辑字段 | 按墙的先例只回 Markdown，元数据读公开详情 |
| O8 | 网页浏览默认 `resource_update_time desc` | 默认 `resource_updated_desc`，与现网一致 |
| O9 | `homepage_urls` 上限未测 | 已测：单工具最多 2 条、最长 95、全部 http(s)；10 × 500 不会拒掉现存数据 |
| O10 | 别名单工具最大值未测 | 已测：最多 6 个、最长 31；17 不会拒掉现存数据 |
| O11 | 同一工具两行空 content | 已测：0。非空重复 0。迁移 142 第 3 步安全 |
| O12 | 嵌套资源在发布者不可渲染时出现、单条 GET 却 404 | 嵌套列表里也丢掉，与 contributors 和 404 同口径（§3.2） |
| O13 | toolset 墙 `comment_count` 与上游 22/27 漂移 | 本轨不修；D21 之后的新写入对齐，历史对账另起 |
| O14 | 分类关系 0 行 | 不建模 |
| O15 | Flutter 前瞻文档仍写 GET detail | 实现 PR 改 `app-direct-api.md` 与 apps 那张票 |

---

删旧路由时：16 条、`ToolsetHandler` / `ResourceHandler` / `UploadHandler` / `PracticalityHandler`、旧 DTO、`CommentService.GetLatestForDetail`（详情不再嵌预览）。`legacy_route_baseline` −16。`rg` 证明零调用方，含 `apps/web/server/` 与 `../kungal-apps`。
