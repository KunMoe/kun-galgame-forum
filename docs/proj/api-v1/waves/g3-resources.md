# G3 · Galgame 下载资源

> G 轨第三段（G0 改号之后），2026-09-23。`internal/galgame/handler.ResourceHandler` 的 **11** 条旧路由，加上搜索波留下的 `GET /api/search?type=resource`。迁移号 **145**（G 号段 140–159；140–144 已占用：G0 140/141、G1 142、G2 143、G1.1 144）。
> 契约与变异题同一个提交，早于实现。普查全文在 [census/galgame-resources-toolsets.md](census/galgame-resources-toolsets.md) §0–§3、§8–§13，结论已对过代码。普查写于 G0 之前：作品 id 现已是 catalog work id，列 `galgame_resource.galgame_id` 已改名 `work_id`（迁移 141）；普查发现 #14 与 §9 的 `galgame_id` / `catalog_work_id` 命名建议作废。评论墙已在 RC（`subject_type=galgame_resource`），本轨不重做。

## 1. 普查（生产实测 2026-09-23）

### 1.1 数字

| 事实 | 数字 |
|---|---|
| `galgame_resource` 行 | 50,377，来自 720 个不同用户 |
| `status` | 0（valid）46,950；1（expired）3,427；过去 30 天失效 43 |
| `type` | game 46,296；collection 2,905；cg 846；patch 218；voice 39；ost 23；other 11；wallpaper 11；tool 11；video 10；artbook 6；crack_fix 1（每一个都是 `resourcevocab.TypeKeys` 的键；没有遗留的 `image` / `ai` / `others`） |
| `languages` 元素 | zh-cn 45,026；ja-jp 5,470；zh-tw 191；other 111；en-us 63；空数组 0 行 |
| `platforms` 元素 | win 43,663；and 5,876；oth 1,983；dvd 398；swi 178；ios 174；psp 126；mac 44；lin 37；psv 27；… |
| `runtimes` 元素 | native-win 43,655；native-and 3,822；other 2,252；tyranor 1,737；kirikiroid2 1,561；onscripter 1,373；gamehub 1,116；winlator 599；native-ios 174；tyranor-next 144；joiplay 5；renpy-android 3 |
| `platforms` 与 `runtimes` 都空 | 0 行 |
| `title` | 空 49,929 行；最长 35 |
| `version_label` | 空 50,360；官方最新 7，汉化版 7，稳定版 2，镜像版 1；最长 4 |
| `note` 最长 | 2,309 |
| `code` / `password` 最长 | 47 / 127；至少有其中一个的 40,079 行 |
| `size` | 50,356 行匹配 `N[.NN] MB\|GB`；21 行是 `∞GB` |
| 链接 | 每个资源 1–6 条（50,377 个资源，0 个没有链接）；scheme https 51,375、magnet 355、http 29、**`ttps` 5、`tps` 1**（资源 390、13947、18001、35234、35100、36049）；16 条含全角 `？` |
| 每部作品 | 最多 123 个资源；p99 32 |
| 赞 | 8,821 行；`like_count` p99 3，最大 22 |
| `comment_count` | 108 行 > 0，最大 7 |
| `edited` | 非空 0 行 |
| `galgame.resource_publish_banned` | 499 部作品；464 个资源落在它们上面 |
| `galgame.published` | true 9,820 / false 6,375；696 部已发布作品没有资源（粘性旗）；**0 个资源落在未发布作品上** |
| `galgame.content_limit` | sfw 7,500；nsfw 8,678；NULL 17；20,268 个资源落在 nsfw 作品上 |

### 1.2 调用方

- `apps/web/server/**`：零 API 调用。命中只是 sitemap 的**页面路径** `server/utils/kunSitemapSources.ts`（`path: '/galgame-resource'`），不是这个 API。
- Flutter App（`/home/kun/Desktop/code/website/kungal-apps`）：Dart 源零命中。`packages/kungal_api/lib/kungal_api.dart` 是空 `library;`。`docs/tickets/02-forum-direct-api-prereqs.md` 与论坛 `docs/proj/app-direct-api.md:118-119` 把下载面列为前瞻契约，不是现网调用。
- 网页：
  - 浏览墙 `pages/galgame/resource/index.vue`（`GET /galgame-resource`，`limit=50`，`keywords`）
  - 资源页 `pages/galgame/resource/[id]/index.vue` + `components/galgame/resource/detail/{Hero,Panel,Info,Recommendations}.vue`
  - 作品页列表 `components/galgame/resource/{Resource,Link,LinkDetailModal,LinkEditModal,Like,ExpireStatus,Help,Card,BuyLegitNotice}.vue`
  - 报失效 `composables/useReportResourceExpired.ts`
  - 用户页改自己的链接 `components/user/Resource.vue`（列表仍走用户域 `GET /user/:id/resources`，本轨只切它的编辑 / 标有效）
  - 搜索 tab 与总览 `components/search/{List,ResourceCard,Overview}.vue` + `utils/search/{lanes,overview}.ts`（`type==='resource'` 打 `GET /search?type=resource`）
  - 作品页禁发开关 `components/galgame/Header.vue`
  - 评论墙已经走 RC：`components/galgame/resource/comment/CommunityContainer.vue` → `useCommunityCommentList({ kind: 'resource' })` → `GET /api/v1/wall-comments?subject_type=galgame_resource&subject_id=…`

### 1.3 旧面的问题（普查编号 E*，对应 census 发现号）

| 编号 | 事实 |
|---|---|
| E1 | 路径参数大量是假的（census §1.1）：作品下的 CRUD / like / valid / expired 读 body/query 的 `galgame_id` / `galgame_resource_id`；`GET …/resource/all` 读 query `galgame_id`。`GET /galgame/5/resource/all?galgame_id=3` 返回 3 号作品 |
| E2 | `GET /galgame-resource/:id` 找不到时 `response.OK(c, "not found")`（发现 1）：HTTP 200、`data:"not found"`。同资源的下载面是 404 |
| E3 | `rowToMeta` 把第一条下载 URL 写入 `link_domain`（发现 2）。字段名叫 domain，值是完整磁链 / 网盘 URL |
| E4 | `GET …/:id/detail` 匿名下发 `link[]` / `code` / `password`，不查 SFW / hidden / banned / expired（发现 3）。每次 GET `download + 1` |
| E5 | 作品列表不查 SFW / published / hidden（发现 4） |
| E6 | 墙的 `COUNT(*)` 与 `hydrateCards` 谓词不同：Go 再丢掉不可渲染作者和 catalog brief 缺失（发现 5） |
| E7 | `ORDER BY created DESC` / `like_count DESC` / `RANDOM()` 均无 `id` 决胜键（发现 6） |
| E8 | 资源搜索 `note ILIKE` 不转义 `%` / `_`（发现 7）；工具列表有 `escapeLike` |
| E9 | catalog 搜索失败静默退化成只搜备注（发现 8） |
| E10 | `CountAll` / `ListPaginated` / `Scan` / `IncrementView` / `IncrementDownload` / `IsResourcePublishBanned` 的 error 全部丢掉（发现 9） |
| E11 | `User` / `Hydrate` 丢 OAuth 错，Placeholder `Status=0` 可渲染（发现 11、12） |
| E12 | 创建不验证 catalog 作品存在，`PublishLocal` 可种出本地 `galgame` 行（发现 13） |
| E13 | `claimOnFirstResource` 在**每一次**创建之后都打 `adoptAndPublish`（`resource_service.go:381`）。普查发现 14 的「论坛 gid 当 catalog work id」在 G0 之后作废；剩下的问题是：后发的资源也会去认领，上游 429 曾按 ERROR 打过（2026-09-23 生产事故） |
| E14 | 创建奖 3 萌萌点，删除扣 `like_count+5`（发现 15）。网页文案写「扣除 5」 |
| E15 | 创建 / 点赞 / 删除的幂等键是 `KeyNonce`（发现 16） |
| E16 | `PUT …/like` 是开关；取消赞仍写 `type="liked"` 消息（发现 17） |
| E17 | 网页「17 天内不换链接就删除」无 cron（发现 18） |
| E18 | `MarkExpired` 在 checker 未配置时直接标死（发现 19）。**保留**：失败关闭到「算它死了」，verdict `unchecked` |
| E19 | `SetResourcePublishBan` 对不存在的 id upsert 本地行（发现 20） |
| E20 | body `{banned}` 缺省就是 false（发现 21） |
| E21 | 失效资源进入全站墙与作品列表（发现 22）。**保留**：列表可按 `state` 过滤，默认都在 |
| E22 | 全部 POST 不读 `Idempotency-Key`（发现 45） |
| E23 | GET 详情 `IncrementView`、下载 `IncrementDownload` 走能刷 `updated` 的路径，并触发 `trg_feed_galgame_resource`（`AFTER INSERT OR UPDATE OR DELETE`，函数只读 `id` / `user_id` / `work_id` / `created`） |
| E24 | 全 `/api` 挂 `NamePreference`，墙上的 `galgame_name` 随 cookie 变（发现 47） |
| E25 | `VersionLabelKeys` 从未用于校验（发现 48）；服务端接受任意单行 ≤64 字 |
| E26 | `resource_count - 1` 无下限，能打到负数 |
| E27 | 禁发中的作品：创建 / 更新 403、删除仍允许（迁移 061）。403 没有 problem code |

## 2. 范围与寻址

路径段就是目标。id 永不来自身体或 query（修 E1）。集合 **`/api/v1/galgame-resources`**，对象 **`galgame_resource`**（墙的 `subject_type` 已经是这个 token）。路径参数 `{resource_id}`（短式，与 G1 的 `{resource_id}`、活动嵌入 `ActivityResource.resource_id` 同义同型）。

G17：写路径里每一个 `{x_id}` 前缀必须有 GET 且 200 带 `id`。本轨因此多一条 `GET /works/{work_id}`（任务书未列，见 §8 O1）：`POST /works/{work_id}/resources` 与 `PUT`/`DELETE /works/{work_id}/resource-publish-ban` 都寻址 `{work_id}`，而现 spec 只有 `GET /works/{work_id}/moyu-patches`。`PATCH`/`DELETE /galgame-resources/{resource_id}` 的 GET 已在下表第 2 行。

| # | 旧 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /galgame-resource` + `GET /search?type=resource` | `GET /api/v1/galgame-resources` 页码集合（K11），`q` 搜索（搜索 tab 的资源车道搬到这里） | optional |
| 2 | `GET /galgame-resource/:id` | `GET /api/v1/galgame-resources/{resource_id}` | optional |
| 3 | `GET /galgame-resource/:id/detail` | `POST /api/v1/galgame-resources/{resource_id}/downloads`（K-G16：拿链接 / 提取码 / 解压密码的唯一路径；计一次下载） | optional |
| — | — | `GET /api/v1/galgame-resources/{resource_id}/source` 给编辑者（Markdown 与秘密，不计下载；G1 先例） | required |
| 4 | `GET /galgame/:gid/resource/all` | `GET /api/v1/works/{work_id}/resources` 页码 | optional |
| 5 | `POST /galgame/:gid/resource` | `POST /api/v1/works/{work_id}/resources` → **201** + `Location`；`Idempotency-Key` **必带** | required |
| 6 | `PUT /galgame/:gid/resource` | `PATCH /api/v1/galgame-resources/{resource_id}`（部分更新；也收 `state: "valid"`） | required |
| 7 | `DELETE /galgame/:gid/resource` | `DELETE /api/v1/galgame-resources/{resource_id}` → **204** | required |
| 8 | `PUT …/resource/like`（切换） | `PUT` + `DELETE /api/v1/galgame-resources/{resource_id}/like`（K16 槽；都 200） | required |
| 9 | `PUT …/resource/valid` | 折进第 6 行（`PATCH` 带 `state: "valid"`；作者或 `resource.edit_any`） | required |
| 10 | `PUT …/resource/expired` | `POST /api/v1/galgame-resources/{resource_id}/expiry-reports` → 200 `{verdict, state}` | required |
| 11 | `PUT /admin/galgame/:gid/resource-publish-ban` | `PUT` + `DELETE /api/v1/works/{work_id}/resource-publish-ban`（K16 槽；staff 权限 `perm.GalgameBanResourcePublish`，仅 cookie） | required |
| — | — | `GET /api/v1/works/{work_id}` → **200** `WorkRef`（G17；G4 会把 200 扩成 `Work`，见 O1） | optional |

11 条 ResourceHandler 路由加 1 条 `GET /search`（现已收窄成只剩 `type=resource`）全删，`legacy_route_baseline` 下调 **12**。v1 共 **14** 个操作（多了 source、like 拆槽、禁发拆槽、G17 的 getWork；valid 折进 PATCH）。

代码位置：资源面 `internal/galgame/resourceapiv1/**`；`getWork` 挂在已有的 `internal/galgame/apiv1`（与 `listWorkMoyuPatches` 同包，G4 扩形状时碰得到）。测试 `internal/app/v1_galgame_resource_*_test.go`。`listWorkResources` / `createWorkResource` / `getWork` / 禁发槽的 OpenAPI tag 是 `works`，其余 `galgame-resources`。

## 3. 形状

### 3.1 命名

| 旧 | v1 | 为什么 |
|---|---|---|
| 路径 `:gid` / `:id`，字段不对齐 | `{work_id}` / `{resource_id}` | 路径参数与字段同名（K1）；`{resource_id}` 与 `ActivityResource.resource_id` 同义 |
| `gid` / `galgame_id` | `work_id` | G0：论坛作品 id 就是 catalog work id |
| `created` / `edited` / `updated` | `created_at` / `edited_at` / `updated_at` | 禁用名 |
| `view` / `download` | `view_count` / `download_count` | 计数 `_count` |
| `user` | `author` | 禁用名。一条 galgame 资源只有一个人（对比 G1 工具资源的 `poster`：那里工具作者和资源发布者是两个人） |
| `is_liked` | `viewer.has_liked` | K9 |
| `status` 裸整数 0/1 | `state`：`valid` / `expired` | 与 Problem.status 撞型；`state` 在 G8 例外表里，词表可以和待办 / 上传会话不同 |
| `type` | `resource_type` | 问题目录的 `type` 是 URI（G8）；g-plan 已裁，与 `ActivityResource` / GE `/works?resource_type=` 同名同词表 |
| `language` / `platform` 标量 + `languages` / `platforms` 数组 | 只保留数组 `resource_languages` / `resource_platforms` | 标量是兼容投影，v1 丢掉；元素类型与 `WorkSummary` / `ActivityResource` 相同 |
| `runtimes` | `resource_runtimes` | 跟另外两轴同一前缀；现 spec 未占用 `runtimes`，改名是为了三轴一组 |
| `link`（数组） | `download_urls` | `links` 是 `CatalogLink[]`（G8）；工具集单条叫 `download_url`（`string \| null`），数组用复数 |
| `link_domain` | **不下发** | 普查 #2 泄漏第一条 URL；供应商用 `provider_names` |
| `code` | `extraction_code` | 顶层 `code` 禁用；与 G1 同名同型 |
| `password` | `archive_password` | 与 G1 同名同型 |
| `note` / `note_html` | 读面 `content: ContentDocument`；源面 `content_markdown` | `ActivityResource.note` 是 string\|null 摘录（G8）；01：Markdown 源叫 `content_markdown` |
| `resources`+`total` / 裸数组 | `repr.PageList` | 页码集合 |
| `keywords` | `q` | 与其它搜索一致；x2-search 资源车道搬过来 |
| `galgame` 摘要 / `galgame_name` | `work: WorkRef \| null`（本面恒有值） | 现 spec 每个 `work` 都是可空 `WorkRef`（G8）；名字三件套，不再走 `NamePreference` |
| `dlsite_purchase_url` 等 omitempty | `dlsite: DlsiteOffer \| null` | 静态形状，没有就整段 `null` |
| `sort_field` + `sort_order` | 墙只有 `created_desc`（默认）/ `created_asc`；作品列表无 `sort` | 墙没有排序 UI（A5） |

### 3.2 对象

**`GalgameResource`（`object: "galgame_resource"`）列表条目与详情同一形状**（任务书：除非 G1 那种摘要 / 详情分裂有必要。这条资源没有嵌套子资源，不裂）。

| 字段 | 类型 | 空 / 可空 | 来源列 |
|---|---|---|---|
| `id` | DecimalID | 永不空 | `galgame_resource.id` |
| `work` | `WorkRef \| null` | schema 可空（G8：现 spec 每个 `work` 都是）；本面恒有值 | `work_id` → `CatalogRowsByWorkIDs` + `WorkRefOf` |
| `author` | UserRef | 永不空（不可渲染的资源不会出现，详情 404） | `user_id` |
| `resource_type` | 封闭 `resourcevocab.TypeKeys` | 永不空 | `type`（`CompatType` 之后） |
| `resource_languages` | `ResourceLanguage[]` | 永不 `null`，至少 1 | `languages`；空则 `LegacyLanguage(language)` |
| `resource_platforms` | `ResourcePlatform[]` | 永不 `null`，可为 `[]`（只有运行环境的模拟器资源） | `platforms` |
| `resource_runtimes` | 封闭 `resourcevocab.RuntimeKeys` 数组 | 永不 `null`，可为 `[]` | `runtimes` |
| `title` | string ≤200 | **空串表示没有**，不可空（§3.12） | `title` |
| `version_label` | 封闭枚举或 `null` | 空列 → `null` | `version_label`，读时映射，见 §3.3 |
| `size` | string ≤64 | 永不空；自由文本，含现存 `∞GB` | `size`。与 `ActivityResource.size` / 摸鱼补丁 `size` 同型 |
| `provider_names` | string[] | 永不 `null` | `provider_name` jsonb；写时由 `DetectProviderNamesFromURLs` 从链接派生，读面只出 |
| `content` | ContentDocument | 空备注是空文档，不是 `null` | `note` 经 `internal/apiv1/content.Converter`（`markdown.ParseStored`） |
| `state` | `valid` \| `expired` | 永不空 | `status` 0/1 |
| `view_count` | int ≥0 | | `view` |
| `download_count` | int ≥0 | | `download` |
| `like_count` | int ≥0 | | `like_count` |
| `comment_count` | int ≥0 | | `comment_count`（RC 墙写入维护） |
| `created_at` | date-time | | `created` |
| `updated_at` | date-time | | `updated` |
| `edited_at` | date-time \| null | 从未编辑为 `null`（生产全是） | `edited` |
| `dlsite` | `DlsiteOffer \| null` | 没有 DLsite workno 为 `null` | 读时 `storelink.Resolver.Resolve` |
| `viewer` | `GalgameResourceViewer \| null` | 匿名为 `null` | |

**不下发**：`link_domain`、任何链接、`extraction_code`、`archive_password`、`content_markdown`、推荐列表（旧详情的 `recommendations`）。同作品其它资源请打 `GET /works/{work_id}/resources`。

`viewer`：`has_liked`、`can_edit`、`can_delete`。Bearer 的两个 `can_*` 不含 staff（K2）。`can_edit` = 作者或 `user.Can(perm.ResourceEditAny)`；`can_delete` 对 `perm.ResourceDeleteAny`。`viewer` 在 G8 例外表里，Go 类型可与工具集的 Viewer 分开。

**`DlsiteOffer`**：`purchase_url`（`format: uri`，短链或联盟模板）、`coupon_url`（`string \| null`，`format: uri`）、`campaign_name`（`string \| null`，自由文本 ≤256）。`campaign_name` 有值表示正在进行的活动；静态优惠券页时为 `null`，网页沿用自己的说明。三者都没有时整段 `dlsite` 是 `null`，不发三个空串。

**`GalgameResourceDownload`（`object: "galgame_resource_download"`）**，POST downloads 的 200：

| 字段 | 类型 | 说明 |
|---|---|---|
| `download_urls` | string[]，1–20，downloadlink pattern，≤4096 | 存储的链接，顺序与库行一致。永不 `null` |
| `extraction_code` | string ≤1007 | 没有则空串。与 G1 同名同型 |
| `archive_password` | string ≤1007 | 没有则空串。与 G1 同名同型 |

成功才 `download_count + 1`。不收 `Idempotency-Key`（G1 同一条：幂等中间件只挂 required 档，这条是 optional、匿名可调）。

**`GalgameResourceSource`（`object: "galgame_resource_source"`）**：`resource_id`、`resource_type`、`title`、`version_label`、`resource_languages`、`resource_platforms`、`resource_runtimes`、`size`、`download_urls`、`extraction_code`、`archive_password`、`content_markdown`（maxLength 10000）。要求 `viewer.can_edit`，否则 `403 PERMISSION_REQUIRED`。档位 required。不计下载、不加浏览。

**`GalgameResourceEngagement`（`object: "galgame_resource_engagement"`）**，PUT/DELETE like 的 200：`resource_id`、`like_count`、`viewer: { has_liked }`（required 档，viewer 非 null）。`has_liked` 与详情同名同型。

**`GalgameResourceExpiryReport`（`object: "galgame_resource_expiry_report"`）**，POST expiry-reports 的 200：`resource_id`、`verdict`（`alive` / `dead` / `unchecked`）、`state`（该资源当前的 `valid` / `expired`，与 `GalgameResource.state` 同词表）。

**`WorkResourcePublishBan`（`object: "work_resource_publish_ban"`）**，PUT/DELETE 禁发槽的 200：`work_id`、`is_resource_publish_banned`。required 档，恒有值。

### 3.3 词表（K-G18）

封闭枚举。存储值不改（语言 / 平台 / 类型 / 运行环境的键与网页常量、GE、活动资源一致）。`version_label` 存储仍是中文，v1 用 snake_case token（F1：封闭枚举取值 `^[a-z][a-z0-9]*(_[a-z0-9]+)*$`，中文过不了门）。

| 字段 | 取值 |
|---|---|
| `resource_type` | `game` `patch` `collection` `crack_fix` `mod` `tool` `walkthrough` `ost` `voice` `cg` `wallpaper` `artbook` `video` `other` |
| `resource_languages` 元素 | `zh-cn` `zh-tw` `ja-jp` `en-us` `other`（01 §3 语言标签例外；F1 的 `languageTagProperties` 已含 `resource_languages`） |
| `resource_platforms` 元素 | `resourcevocab.PlatformKeys`（`win` `and` `ios` … `oth`） |
| `resource_runtimes` 元素 | `resourcevocab.RuntimeKeys`（`native-win` `native-and` `native-ios` `winlator` `gamehub` `kirikiroid2` `krkrsdl2` `onscripter` `joiplay` `easyrpg` `renpy-android` `tyranor` `tyranor-next` `other`） |
| `version_label` | `official_latest` `stable` `mirror` `localized` `unknown`；没有则为 `null` |
| `state` | `valid` `expired` |
| `verdict` | `alive` `dead` `unchecked` |
| `sort`（仅 `GET /galgame-resources`） | `created_desc`（默认）`created_asc` |

`version_label` 存储 ↔ v1：

| 列 | v1 |
|---|---|
| `官方最新` | `official_latest` |
| `稳定版` | `stable` |
| `镜像版` | `mirror` |
| `汉化版` | `localized` |
| `未知版本` | `unknown` |
| `''` | `null` |

生产 `未知版本` 0 行，网页下拉里有，保留（K25 形态：客户端按 token 本地化）。写面只收 v1 token，写入中文列。

**`resource_runtimes` 的 kebab-case 是 01 §3 的具名例外**（与语言标签同一类）：存储、网页 `RUNTIME_OPTIONS`、生产全是 `native-win` 这种带连字符的键，改成 `native_win` 就要永久翻译。F1 门给这个属性加一条「小写 kebab-case 或 `other`」的检查，与 `languageTagProperties` 并列。实现 PR 改 `gates/repr.go` 与 01 §3 例外表。

未知过滤值 → `400 UNKNOWN_ENUM_VALUE`。缺席 = 不过滤（没有 `all` token）。未知 `sort` → `400 UNKNOWN_SORT`，不得静默回落。

### 3.4 列表（K-G17、K-G20）

**`GET /galgame-resources`**：页码（`collect.PageNumber` 的 `page` / `limit`，本集合默认 `limit` **50**，最大 100；浏览页 `index.vue` 的 `LIMIT = 50`；搜索 tab 传 24，总览传 6）。`CheckDepth` `page × limit ≤ 10000`，越界 `400 INVALID_PARAMETER` `OUT_OF_RANGE`。响应 `repr.PageList[GalgameResource]`，必发 `total` + `total_relation`。

Query：`page`、`limit`、`q`（1–107，与 x2-search 的 `q` 上限一致；缺席或去空白后为空 = 不搜）、`include_nsfw`（默认 `false`）、`state`（缺席 = `valid` 与 `expired` 都在）、`sort`（默认 `created_desc`）。

`q`：转义 ILIKE 打 `galgame_resource.note`（修 E8）；同时 `CatalogWorksSearch`（`sort=relevance`，`limit=100`，`include` 对齐 `WorkRefOf`）拿作品名命中的 `work_id`。有 catalog 命中时排序 `relevance DESC, created DESC, id DESC`（relevance = 该行 `work_id` 是否在 catalog 命中里）；没有则 `created DESC, id DESC`。catalog 失败 → `503 SERVICE_UNAVAILABLE`（修 E9：不再静默丢掉作品名半边）。每个排序带同方向的 `id` 决胜键（修 E7）。

**NSFW（K-G17）**：`include_nsfw=false`（默认）排除「本地 `galgame.content_limit = 'nsfw'`」的资源。NULL 放行（与今日墙相同，迁移 079 尚未同步）。**一条 SQL 谓词，COUNT 与页查询共用。** 旧 `utils.IsSFW` / cookie 路径删除。

**已发布**：每一条读（浏览、作品列表、详情、downloads、source）都要求该作品本地 `galgame.published = true`，否则 404。今日 0 行受影响；隐藏 / 封禁会取消发布（方案③）。禁发作品上的资源**仍然可见**（遗留，464 行）——记为开放问题 O2，本轨不改。

**不可渲染作者（K-G20）**：先按过滤条件取出作者 id 去重，经 `userclient.Users`（缓存）批量判定，**COUNT 与页查询共用同一个「作者可渲染」谓词**（修 E6）。`userclient` 失败 → `503 SERVICE_UNAVAILABLE`，不得 fail-open（修 E11）。详情 / downloads / source / 该资源上的每一笔写，作者不可渲染都是 `404`。

catalog RPC 失败（为页上的 `work` 调 `CatalogRowsByWorkIDs`）→ 503。hidden claim 被 `isRenderable` 丢掉：该行不出现在页上；COUNT 仍按 SQL 谓词（published + nsfw + 作者 + state + q）。hidden 且本地仍 published 的资源会造成轻微的 total 偏差，见 O3。

**`GET /works/{work_id}/resources`**：页码，默认 `limit` **50**，最大 100。排序固定 `state ASC`（`valid` 在前，对应旧 `status ASC`）、`created DESC`、`id DESC`。没有 `sort`。可选 `state`。 **不按 NSFW 闸**（调用者已经在这部作品的页面上；G2 作品 tab 传 `include_nsfw=true` 是同一个原因）。作品不存在、catalog hidden、或本地 `published=false` → 404。主人可渲染但没有资源 → 200，空 `items`，`total=0`。生产单作品最多 123，p99 32：`limit=100` 仍会切掉那一部 123 的第二页，见 O4。

失效资源留在每一个列表里（E21 保留）。墙与作品列表都接受可选 `state`。

### 3.5 下载（K-G16）

详情与列表不带秘密。秘密只从 `POST …/downloads` 来。档位 optional：匿名下载保持今天的行为。每次成功调用 `download_count + 1`（列更新，不写 `updated`，不触发 feed）。作者不可渲染、作品未发布、行不存在 → 同一句 `404 NOT_FOUND`。`userclient` 失败 → `503`。

`GET …/source` 给编辑者回同样的秘密，**不计下载**。无 `can_edit` → 403。

### 3.6 写面限额

取自现行 DTO、网页 zod / 表单、列宽、生产最长值，取三者最紧且真实的：

| 字段 | 限额 | 来源 |
|---|---|---|
| `resource_type` | 创建必填封闭枚举 | DTO required；`parseResourceAxes` / `IsType` |
| `resource_languages` | 1+，封闭，请求内唯一 | 生产 0 行空数组；旧面空则回退标量，v1 **不收**标量 `language` / `platform` |
| `resource_platforms` / `resource_runtimes` | 封闭，请求内唯一；两者不能都空；`HasRuntimeAxis(type)`（`game` `collection` `patch` `crack_fix` `mod` `tool`）时 runtimes 至少 1；非运行环境类型有 runtimes → `422 INCONSISTENT_WITH` | `parseResourceAxes`；生产两者都空 0 行 |
| `title` | ≤200，单行；可空串；K19：先按原始值判长 | DTO `max=200`；生产最长 35，空 49,929 |
| `version_label` | 枚举或 `null`；缺席 / `null` 都是「没有」 | 生产几乎全空；K25 |
| `size` | 必填；`filesize.Parse`：`N[.NN] MB\|GB`（大小写不敏感，归一成 `N[.NN] MB\|GB`）；读面自由文本，现存 `∞GB` 原样下发，写面不能再写成它 | DTO / `filesize.go` |
| `download_urls` | 1–20，每条 1–4096，downloadlink（`http` `https` `ftp` `ftps` `magnet` `ed2k` `thunder`）；出现即整组替换 | DTO `min=1,max=20,dive,downloadlink`；列 `text`；生产每资源 1–6 条。全角 `？` 仍过（生产 16 条，校验器特判） |
| `extraction_code` / `archive_password` | ≤1007；可空串 | 列 / DTO |
| `content_markdown` | ≤10000；可空 | 列 `varchar(10000)` / DTO `note` |
| `state`（仅 PATCH） | 只收 `valid` | 标有效折进 PATCH |

交叉字段 → `422 INCONSISTENT_WITH`。`PATCH` 出现即替换该字段；`download_urls` / 轴数组出现即整组替换，缺席不动。K18：只有本次提交的文本走 trust——`content_markdown`、`download_urls`。只改类型 / 轴 / `size` / `version_label` / `title` / `state` / 密码，不跑 trust、不重新入扫描。deny → `422 CONTENT_REJECTED`。hold 仍写入并 `ScanBg`。信任文本经 `markdown.NormalizeStoredContent` 后再拼。

### 3.7 萌萌点

| 事件 | 分 | 幂等键 | ref |
|---|---|---|---|
| 创建资源 | +3 | `kungal:galgame_resource_create:<id>` | `galgame_resource:<id>` |
| 删除资源 | **−3** | `kungal:galgame_resource_delete:<id>` | `galgame_resource:<id>` |
| 他人点赞 / 取消 | `liked` ±1 | `kungal:liked:galgame_resource_like_{row}` / `kungal:unliked:galgame_resource_like_{row}` | `galgame_resource:<id>`（自己的不给） |

键不再含 `KeyNonce`（修 E15）。创建 / 删除与 G1 工具资源错开前缀（G1 是 `kungal:resource_create:<id>`，两张表的 id 会撞）。发分等提交成功之后，经 pusher 异步推送（G1/G2；推送不阻塞写面）。点赞键用行的 `id`（表已有 serial）：取消再点会插入新行、拿到新 id，所以正好再给一次。

删除不再扣 `like_count+5`，网页「扣除 5 萌萌点」改成 3（修 E14）。

### 3.8 浏览与下载计数

`GET /galgame-resources/{resource_id}` 照旧 `view + 1`（话题 / 工具 / 题目详情同样在 GET 里计数），**同步**执行，错误上抛 500。`POST …/downloads` 成功才 `download + 1`，同样同步。两条都用**不带 `updated` 的列更新**。迁移 145 把 `trg_feed_galgame_resource` 收窄到 `INSERT OR DELETE OR UPDATE OF user_id, work_id, created`（函数读的列），浏览 / 下载 / 赞 / 失效不再写 `feed_activity`（修 E23）。

源面、列表、作品列表不加浏览。

### 3.9 创建、认领、禁发（K-G19）

**创建**：路径 `{work_id}` 必须是 catalog 作品（一次 `CatalogRowsByWorkIDs`，hidden 被 `isRenderable` 丢掉）→ 未知或 hidden 都是 `404 NOT_FOUND`。本地没有 `galgame` 行就在这个 id 下插入（铁律 3）；`PublishLocal` 把 `published` 置 true（粘性，迁移 078）。禁发作品 → `403 RESOURCE_PUBLISH_BANNED`（kungal 域新码；旧 403 没有 code，修 E27）。`Idempotency-Key` 必带。`Location`：`/api/v1/galgame-resources/{id}`。201 体是 `GalgameResource`（无秘密）。

**静默 catalog 认领（g-plan G3 注，2026-09-23 生产事故）**：只在这一次创建把本地行从 unpublished **翻成** published 时才打 `adoptAndPublish`（该作品的第一份资源），后续创建绝不打。没有 access token（Bearer 且没有网页那种 OP cookie token）整段跳过，与今日相同。上游 429 打 **WARN**，不是 ERROR。认领失败永不让创建失败。409「已有归属」继续 Info。

实现上：事务里先读本地 `published`（行不存在视为 false），`PublishLocal` 之后若本次是 false→true 才在提交成功之后调 `claimOnFirstResource`。

**PATCH**：作者或 `perm.ResourceEditAny`。禁发作品 → `403 RESOURCE_PUBLISH_BANNED`（遗留：改拒绝、删允许）。K18。`download_urls` 出现 = 整组替换（先删后插，同一事务）。写 `edited_at`（生产 0 行非空，旧 PUT 根本没写这一列）。`state: "valid"` 仅作者 / 编辑者可写，这是这条 PATCH 唯一的状态写入；已经是 `valid` 再写是 200 空操作。`state: "expired"` → `422` `/state` `NOT_ALLOWED_VALUE`（报失效走 expiry-reports）。

**删除**：作者或 `perm.ResourceDeleteAny`。硬删（链接、赞随 FK CASCADE）。`resource_count = GREATEST(resource_count - 1, 0)`（修 E26）。不把 `published` 拨回去。禁发中仍可删。萌萌点 −3。

**赞槽（K16）**：`INSERT … ON CONFLICT DO NOTHING RETURNING id` / `DELETE … RETURNING id`，只在真的动了行时改 `like_count` 并发分。置已置、撤未置都是 200 无副作用。自赞 → `403 SELF_LIKE_FORBIDDEN`（拓宽已注册描述，code 不变）。`liked` 消息只在真的插入一行时写，取消赞不写（修 E16）。`galgame_resource_like.updated` 是 `NOT NULL` 且无默认值：INSERT 必须带上这一列（W4 收藏/推的同一陷阱）。

**报失效**：任意登录用户。链接检查与今日相同：任一条活 → verdict `alive`，state 不动；全死 → state `expired` + 给作者 `expired` 消息；checker 不可用 / 未配置 → 标失效，verdict `unchecked`（E18 保留）。已经失效 → **200**，不改库、不重发消息、不调 checker，`verdict: "dead"`，`state: "expired"`（旧面 400）。网页「17 天内不换链接就删除」删掉（修 E17）。

**禁发槽**：作品必须存在于 catalog（同一套 `CatalogRowsByWorkIDs`）；未知 id **绝不**种本地行（修 E19）。catalog 有、本地无：PUT 插入一行只带 `id` + `resource_publish_banned=true`（不 `PublishLocal`）。DELETE 在没有本地行时是 200、`is_resource_publish_banned: false`。PUT 置位、DELETE 清除，都 200 带回当前旗。空体（不再有 `{banned: bool}`，修 E20）。仅 cookie；Bearer 过不了 `staffCandidate` → `403 PERMISSION_REQUIRED`。权限 `perm.GalgameBanResourcePublish`。

### 3.10 `getWork`（G17）

`GET /works/{work_id}` 回 `WorkRef`（带 `id`），只为让本轨写路径过 G17。catalog 未知或 hidden → 404。不查本地 `published`。网页 `/galgame/:id` 本轨不切到这里；G4 把 200 扩成 `Work`。

### 3.12 实现时按 G8 / G14 / F1 改的名（只增不改，以此为准）

G8 要求全 spec 里同名属性同型（含可空与 format），G14 要求每个字符串带 enum / format / pattern / 自由文本标记。任务书前文的几个名字撞了既有的面，契约里改成下表：

| 前文 | 实际 | 撞了谁 |
|---|---|---|
| `title` 空为 `null` | `title` 空串，不可空 | 工具集 / 文档 / 活动工具的 `title` 都是 string，不可空。G8 的 `shape()` 含可空性 |
| 链接数组叫 `links` | `download_urls` | 角色 / 公司 / 人名的 `links` 是 `CatalogLink[]` |
| 单条 URL 叫 `download_url` | 数组 `download_urls` | 工具集下载的 `download_url` 是 `string \| null` |
| 源面 `note_markdown` | `content_markdown` | 01：Markdown 源叫 `content_markdown`；读面正文已经是 `content` |
| `runtimes` | `resource_runtimes` | 现 spec 未占用；与 `resource_platforms` / `resource_languages` 一组 |
| `version_label` 用列里的中文当枚举 | snake_case token（§3.3） | F1：封闭枚举取值必须是 snake_case |
| `note` 作读面正文 | `content` | `ActivityResource.note` 是 string\|null 摘录 |

`resource_type` / `resource_languages` / `resource_platforms` / `size` / `extraction_code` / `archive_password` / `resource_id` / `work` / `author` / `state` / `viewer` 与现 spec 对齐，保持。`provider_names`、`dlsite`、`verdict`、`is_resource_publish_banned` 现 spec 未占用。

`size` 读面 maxLength **64**（与 `ActivityResource.size`、摸鱼补丁 `size` 相同）。列是 `varchar(107)`，生产正规体积与 `∞GB` 都远短于 64。G8 的 `shape()` 不含 maxLength；仍取 64，避免以后把门收紧。

## 4. 逐条操作

通用：401 `MISSING_CREDENTIAL` / `INVALID_CREDENTIAL`（required；optional 的坏 Bearer）、403 `ACCOUNT_BANNED`、500 `INTERNAL_ERROR`、503 `SERVICE_UNAVAILABLE`（会话存储 / userclient / catalog）。每个 401 带 `WWW-Authenticate`。v1 全部 `Cache-Control: no-store`。

### 4.1 `listGalgameResources` · `GET /galgame-resources` · optional · 200 `PageList<GalgameResource>`

Query：`page`、`limit`（默认 50）、`q`、`include_nsfw`（默认 false）、`state`、`sort`（默认 `created_desc`）。

| 状态 | code |
|---|---|
| 400 | `UNKNOWN_ENUM_VALUE` `UNKNOWN_SORT` `LIMIT_TOO_LARGE` `INVALID_PARAMETER`（深度 / `q` 格式 / `include_nsfw`） |
| 401 | 坏 Bearer |
| 503 | userclient / catalog |

### 4.2 `getGalgameResource` · `GET /galgame-resources/{resource_id}` · optional · 200 `GalgameResource`

副作用：`view_count + 1`，不刷 `updated_at`。找不到 / 作者不可渲染 / 作品未发布 → 404（修 E2，不再 200 字符串）。

| 状态 | code |
|---|---|
| 404 | 不存在 / 作者不可渲染 / 未发布 |
| 503 | userclient / catalog / 正文用户查找 |

### 4.3 `createGalgameResourceDownload` · `POST /galgame-resources/{resource_id}/downloads` · optional · 200 `GalgameResourceDownload`

空体。**不收 `Idempotency-Key`**（与 G1 4.12 相同）。成功才 `download_count + 1`。可见性同 4.2。NSFW 不闸；`work.is_nsfw` 给网页做 SEO。

### 4.4 `getGalgameResourceSource` · `GET /galgame-resources/{resource_id}/source` · required · 200 `GalgameResourceSource`

| 状态 | code |
|---|---|
| 403 | `PERMISSION_REQUIRED`（无 `can_edit`） |
| 404 | 同详情 |
| 503 | userclient |

### 4.5 `listWorkResources` · `GET /works/{work_id}/resources` · optional · 200 `PageList<GalgameResource>`

Query：`page`、`limit`（默认 50）、`state`。不按 NSFW 闸。

| 状态 | code |
|---|---|
| 400 | `LIMIT_TOO_LARGE` `INVALID_PARAMETER` `UNKNOWN_ENUM_VALUE` |
| 404 | 作品不存在 / hidden / 未发布 |
| 503 | userclient / catalog |

### 4.6 `createWorkResource` · `POST /works/{work_id}/resources` · required · 201 `Location` + `GalgameResource`

`Idempotency-Key` 必带。体 `GalgameResourceCreate`（§3.6）。`Location`：`/api/v1/galgame-resources/{id}`。萌萌点 +3。第一份资源才 `adoptAndPublish`。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（缺 / 格式错的幂等键） |
| 403 | `RESOURCE_PUBLISH_BANNED` |
| 404 | catalog 未知 / hidden |
| 409 | `IDEMPOTENCY_KEY_REUSED` `IDEMPOTENCY_REQUEST_IN_PROGRESS` |
| 422 | `VALIDATION_FAILED`（`TOO_LONG` `TOO_SHORT` `TOO_MANY_ITEMS` `TOO_FEW_ITEMS` `DUPLICATE_ITEM` `UNKNOWN_VALUE` `INVALID_FORMAT` `INCONSISTENT_WITH` `OUT_OF_RANGE`）`CONTENT_REJECTED` |
| 503 | userclient / catalog |

### 4.7 `updateGalgameResource` · `PATCH /galgame-resources/{resource_id}` · required · 200 `GalgameResource`

体 `GalgameResourcePatch`，全部可选。无 `can_edit` → 403 `PERMISSION_REQUIRED`。禁发 → `RESOURCE_PUBLISH_BANNED`。K18。`state: "valid"` 见 §3.9。

### 4.8 `deleteGalgameResource` · `DELETE /galgame-resources/{resource_id}` · required · 204

无 `can_delete` → 403。萌萌点 −3。`resource_count` 永不低于 0。

### 4.9 `putGalgameResourceLike` · `PUT /galgame-resources/{resource_id}/like` · required · 200 `GalgameResourceEngagement`

空体。已赞再 PUT 是 200、计数不变。自赞 → `403 SELF_LIKE_FORBIDDEN`。资源 404。

### 4.10 `deleteGalgameResourceLike` · `DELETE /galgame-resources/{resource_id}/like` · required · 200 `GalgameResourceEngagement`

未赞再 DELETE 是 200、计数不变。不写 `liked` 消息。资源 404。

### 4.11 `createGalgameResourceExpiryReport` · `POST /galgame-resources/{resource_id}/expiry-reports` · required · 200 `GalgameResourceExpiryReport`

空体。`Idempotency-Key` **支持、不强制**（不是创建用户内容）。已经失效 → 200 无副作用。

| 状态 | code |
|---|---|
| 404 | 同详情 |
| 409 | `IDEMPOTENCY_KEY_REUSED` `IDEMPOTENCY_REQUEST_IN_PROGRESS`（若带了键） |
| 503 | userclient |

### 4.12 `putWorkResourcePublishBan` · `PUT /works/{work_id}/resource-publish-ban` · required · 200 `WorkResourcePublishBan`

空体。cookie + `perm.GalgameBanResourcePublish`。catalog 未知 → 404，不写本地行。

| 状态 | code |
|---|---|
| 403 | `PERMISSION_REQUIRED`（含 Bearer） |
| 404 | catalog 未知 / hidden |
| 503 | catalog |

### 4.13 `deleteWorkResourcePublishBan` · `DELETE /works/{work_id}/resource-publish-ban` · required · 200 `WorkResourcePublishBan`

权限与 404 同 4.12。没有本地行 → 200、`is_resource_publish_banned: false`。

### 4.14 `getWork` · `GET /works/{work_id}` · optional · 200 `WorkRef`

G17。catalog 未知 / hidden → 404。**不**要求本地 `published`（那是资源读面的闸；G4 扩成 `Work` 时再定作品页自己的可见性）。网页作品页本轨不切到这条。

| 状态 | code |
|---|---|
| 404 | catalog 未知 / hidden |
| 503 | catalog |

## 5. 预分配

### 5.1 迁移 145 `galgame_resource_v1`

普通迁移，随部署跑，幂等。不是 deploy-then-drop。

```sql
-- 1. Repair 6 links whose scheme lost its first letters, so the v1
--    downloadlink pattern holds on read. Named rows (galgame_resource.id):
--    ttps://  → https://  on 390, 13947, 18001, 35234, 35100
--    tps://   → https://  on 36049
UPDATE galgame_resource_link
SET url = 'h' || url
WHERE url LIKE 'ttps://%';

UPDATE galgame_resource_link
SET url = 'ht' || url
WHERE url LIKE 'tps://%';

-- 2. Browse ORDER BY created DESC, id DESC (K-G16). idx_galgame_resource_created
--    is (created DESC) only (016); ties have no id key. Work-list lookups use
--    idx_galgame_resource_galgame_id, which 141 left sitting on work_id — no
--    new work_id index.
CREATE INDEX IF NOT EXISTS idx_galgame_resource_created_id
    ON galgame_resource (created DESC, id DESC);

-- 3. View / download / like / expiry no longer write feed. The function reads
--    NEW.id / NEW.user_id / NEW.work_id / NEW.created (141, pg_get_functiondef).
DROP TRIGGER IF EXISTS trg_feed_galgame_resource ON galgame_resource;
CREATE TRIGGER trg_feed_galgame_resource
    AFTER INSERT OR DELETE OR UPDATE OF user_id, work_id, created
    ON galgame_resource
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_resource();
```

`∞GB` 体积不动（读面自由文本）。全角 `？` 磁链不动（校验器允许，生产 16 条）。第 3 步不改函数体，只改触发条件。

### 5.2 错误码

新增（三处一译：`registry.go` + `registry_test.go` + `zh-CN/problem.json`）：

- **`RESOURCE_PUBLISH_BANNED`**（kungal，403）。该作品禁止发布下载资源。创建与 PATCH 用。删除不走这条。

拓宽已有码的描述（不改 code / status / type URI）：

- **`SELF_LIKE_FORBIDDEN`**（kungal，403，已注册）。现在的描述是「Users cannot like their own topics, replies or comments.」G3 资源自赞走同一码；描述改为「Users cannot like their own topics, replies, comments or galgame resources.」。

不新增自赞的同义码。其余复用：`NOT_FOUND`、`PERMISSION_REQUIRED`、`ACCOUNT_BANNED`、`SERVICE_UNAVAILABLE`、`VALIDATION_FAILED`（`INVALID_FORMAT` / `UNKNOWN_VALUE` / `NOT_ALLOWED_VALUE` / `INCONSISTENT_WITH` / `DUPLICATE_ITEM` / `TOO_SHORT` / `TOO_LONG` / `TOO_MANY_ITEMS` / `TOO_FEW_ITEMS` / `OUT_OF_RANGE`）、`CONTENT_REJECTED`、`UNKNOWN_ENUM_VALUE`、`UNKNOWN_SORT`、`LIMIT_TOO_LARGE`、`INVALID_PARAMETER`、`IDEMPOTENCY_KEY_REUSED`、`IDEMPOTENCY_REQUEST_IN_PROGRESS`。

### 5.3 K 决定

| 编号 | 决定 |
|---|---|
| **K-G16** | 下载秘密只从 `POST …/downloads` 来，永不从 GET 来（源面除外，源面要 `can_edit`）。列表 / 详情不带链接、提取码、解压密码，不下发 `link_domain`。档位 optional。计数见 §3.8 |
| **K-G17** | 每条读都要求本地 `published=true`，否则 404。浏览 `include_nsfw=false` 默认排除 nsfw 作品上的资源，一条 SQL 谓词 COUNT 与页共用。作品列表 / 详情 / downloads 不按 NSFW 闸，`WorkRef.is_nsfw` 给网页。失效资源留在列表里，可选 `state`。禁发作品上的资源仍可见（O2） |
| **K-G18** | 词表见 §3.3。`version_label` 中文列 ↔ snake_case token。`resource_runtimes` 的 kebab-case 是 01 §3 / F1 具名例外。未知过滤 400；缺席不过滤 |
| **K-G19** | 写面见 §3.6–§3.9。创建查 catalog、缺本地行按 work_id 插入、禁发 `RESOURCE_PUBLISH_BANNED`、认领只在 published 翻转时打。删除 −3。赞 K16。报失效已失效回 200。禁发槽不种未知 id |
| **K-G20** | 与 G1 K-G7 / G2 K-G12 相同：列表 COUNT 与页同一「作者可渲染」谓词；详情 / downloads / source / 写面作者不可渲染是 404；`userclient` 失败 503 |
| **K-G21** | 详情 GET `view_count + 1`、downloads 成功 `download_count + 1`，同步、不写 `updated`。迁移 145 收窄 feed 触发器 |

权限不新增。`resource.edit_any` / `resource.delete_any` / `galgame.ban_resource_publish` 沿用。

## 6. 网页

切到生成的类型化客户端；手写 `shared/types/galgame-resource.ts` 改成生成物别名或删除。`legacy-fetch-baseline` 按删掉的调用点下调。

| 文件 | 改什么 |
|---|---|
| `pages/galgame/resource/index.vue` | `GET /galgame-resources`；`keywords` → `q`；`limit` 50；`include_nsfw` 跟内容姿态 |
| `pages/galgame/resource/[id]/index.vue` | `GET /galgame-resources/{id}`；404 走空状态（不再判断 `data === 'not found'`）；SEO 从 `work.display_name` / `work.is_nsfw`；评论墙不变 |
| `resource/Card.vue` | `user` → `author`；标量 `platform` / `language` / `type` 改轴数组；`created` → `created_at`；名字读 `work.display_name` |
| `resource/detail/Hero.vue` | 不再吃 `ResourceGalgameSummary`；封面读 `work.cover`（WorkRef 没有 banner，见 O5）；NSFW chip 读 `work.is_nsfw` |
| `resource/detail/Info.vue` + `LinkDetailModal.vue` | 「获取链接」改 `POST …/downloads`；删除 `DELETE …/{id}` → 204，文案「3 萌萌点」；编辑先 `GET …/source` 回填（**不要**打 downloads，否则每打开一次编辑多算一次下载） |
| `resource/detail/Recommendations.vue` | 详情不再带推荐；改 `GET /works/{work_id}/resources` 排除当前 id，或删掉这一栏 |
| `resource/Resource.vue` | `GET /works/{work_id}/resources`；`limit=100`；按 `provider_names` 分组仍在客户端 |
| `resource/Link.vue` | 标有效改 `PATCH` `{state:"valid"}`；读 `state` / `viewer.can_edit` |
| `resource/LinkEditModal.vue` | 创建 `POST /works/{work_id}/resources` 201，必带幂等键；编辑 `PATCH` 只发改过的字段；轴只走数组；`version_label` 用 v1 token；`code`/`password`/`note`/`link` 改名 |
| `resource/Like.vue` | `PUT` / `DELETE …/like`；读响应的 `like_count` / `viewer.has_liked`；自赞走 `SELF_LIKE_FORBIDDEN` |
| `useReportResourceExpired.ts` + `ExpireStatus.vue` | `POST …/expiry-reports`；读 `verdict` / `state`；删「17 天内不换链接就删除」 |
| `user/Resource.vue` | 列表仍等用户域；编辑 / 标有效改 `PATCH`（可一次带上 `state:"valid"` 与新 `download_urls`） |
| `search/List.vue` + `utils/search/lanes.ts` + `overview.ts` | `type==='resource'` 改 `GET /galgame-resources?q=&include_nsfw=` |
| `search/ResourceCard.vue` | `galgame_name` → `work.display_name`；`note` 从 `content` 抽纯文本（`documentPlainText`）；计数 `_count` |
| `Header.vue` | 禁发改 `PUT` / `DELETE /works/{id}/resource-publish-ban`；读回 `is_resource_publish_banned`；按钮显隐继续 `useCan('galgame.ban_resource_publish')` 直到作品详情带 `viewer`（G4） |
| `shared/utils/galgameResourceVocab.ts` | `VERSION_LABEL_OPTIONS` 的 value 改 v1 token，label 仍中文 |
| `docs/proj/app-direct-api.md` | 作品列表改 `GET /api/v1/works/{work_id}/resources`；下载改 `POST /api/v1/galgame-resources/{resource_id}/downloads` |

`RESOURCE_PUBLISH_BANNED` 的 zh-CN 译文进 `problem.json`。`SELF_LIKE_FORBIDDEN` 若文案仍只谈帖子，改成通用的「不能给自己点赞」。按钮显隐读 `viewer.can_*`。

## 7. 变异题（先于实现提交）

种子：至少 7 个资源同一 `created` 瞬间（跨页边界）；其中两个作者随后在测试里被标成不可渲染；一部作品先 unpublished 再发第一份资源、再发第二份；一个未知 catalog id；一个已失效资源；一个带提取码与两条链接的资源。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 详情 GET 把 `download_urls` / `extraction_code` / `archive_password` 填进 `GalgameResource`（旧 `…/detail`） | 匿名 GET 详情 JSON 没有这些键（K-G16） |
| 2 | 列表条目带上 `download_urls` 或 `extraction_code` | `GalgameResource` 列表项 JSON 没有这些键；抽查 SQL 里的 URL 不得出现在列表响应（K-G16） |
| 3 | 列表 COUNT 不再排除不可渲染作者，或 NSFW 谓词只打在页查询上 | `total` 等于 SQL 里作者可渲染且（默认）非 nsfw 的行数，且等于小 `limit` 走完的条目数 |
| 4 | 每一次创建都打 `adoptAndPublish`（旧 `claimOnFirstResource`） | 已 published 作品上的第二份资源：认领客户端 0 次调用；第一份（unpublished→published）恰好 1 次 |
| 5 | 禁发 PUT 不先查 catalog、upsert 本地行（旧 `SetResourcePublishBanned`） | 未知 `work_id` → 404，`galgame` 表无新行 |
| 6 | 排序去掉 `id` 决胜键 | 小 `limit` 全量遍历与 SQL 逐条相等、无重无漏（种子有并列 `created`） |
| 7 | `PATCH` 不论字段是否变化都跑 trust | 只改 `resource_type`、旧备注已进禁用词表 → 200，库里 type 已改 |
| 8 | 已失效资源的 expiry-report 回 400（旧行为） | 再 POST → 200，`state: "expired"`，`expired` 消息条数不变 |
| 9 | `resource_count - 1` 不套 `GREATEST` | 从 0 再删 → 列仍 ≥ 0（或本条删在 count 已 0 的夹具上，断言不变成 −1） |
| 10 | 取消赞仍写 `type="liked"` 消息（旧行为） | DELETE like 之后 `message` 里该 `(sender, receiver, type, link)` 条数与赞之前相同 |
| 11 | 去掉「不能给自己赞」的检查 | 作者 PUT like → 403 `SELF_LIKE_FORBIDDEN`，无 like 行 |
| 12 | `userclient` 失败时吞错继续（旧 Hydrate） | 列表 / 详情 / downloads / source → 503，不回占位作者 |
| 13 | 未知 `sort` 回落到 `created_desc` | → 400 `UNKNOWN_SORT` |
| 14 | `POST /works/{work_id}/resources` 不要求 `Idempotency-Key` | 缺头 → 400，`header Idempotency-Key` `REQUIRED` |

## 8. 开放问题

| # | 事实 | 裁决 |
|---|---|---|
| O1 | G17 要求 `POST /works/{work_id}/resources` 与禁发槽有 `GET /works/{work_id}`，任务书寻址表漏了；G4 才拥有作品详情 `Work` | 本轨加 `getWork`，200 `WorkRef`。G4 把同一路径的 200 扩成 `Work`（重叠字段同名同型）。网页作品页本轨不切 |
| O2 | 464 个资源落在 `resource_publish_banned` 作品上，旧墙仍展示 | **不改**。2026-09-24 用户裁决：保持可见，浏览里也不藏。禁发只挡新资源 |
| O3 | catalog hidden 且本地仍 published 的资源：COUNT 按 SQL，页上丢掉 brief | 接受这点偏差。RPC 失败仍是 503。要 COUNT 也排除 hidden 得先有本地 hidden 缓存 |
| O4 | 单作品最多 123 个资源，页码 `limit` 最大 100 | 作品页传 `limit=100`；那一部会有第二页。把本集合上限抬到 200 是加法 |
| O5 | 资源页 Hero 用 `effective_banner_*`；`work` 必须是 `WorkRef`（G8），只有 `cover` | Hero 改画 `cover`。要 banner 等 G4 的 `Work` / `WorkSummary` |
| O6 | 旧详情带最多 6 条同作品推荐 | 详情不再嵌。网页改打作品列表或删栏 |
| O7 | 搜索在 catalog 失败时旧面退化为只搜备注 | v1 回 503。可用性换一致性；编排者若要退化，改口 |
| O8 | `GET /user/:id/resources` 是用户域 | 本轨不迁。用户页列表仍走旧路由，只把编辑 / 标有效切到 PATCH |
| O9 | 21 行 `∞GB` | 读面原样；写面 `filesize.Parse` 仍拒。不迁数据 |
| O10 | `idx_galgame_resource_galgame_id` 在 141 之后坐在 `work_id` 上，名字没改 | 不改名。作品列表用得上，不必再加一条 `work_id` 索引 |
| O11 | F1 例外表要加 `resource_runtimes` | 实现 PR 改 `gates/repr.go` 与 01 §3。本契约指定取值仍是存储的 kebab-case |
| O12 | Flutter 前瞻文档仍写 GET detail | 实现 PR 改 `app-direct-api.md` 与 apps 那张票 |

---

删旧路由时：11 条 ResourceHandler + `GET /search`（现只剩 `type=resource`）、旧资源 DTO 中不再被别的面使用的部分、手写 TS 类型。`legacy_route_baseline` −12。`rg` 证明零调用方，含 `apps/web/server/` 与 `../kungal-apps`。
