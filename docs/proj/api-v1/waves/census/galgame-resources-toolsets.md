# 普查 · Galgame 资源 / 工具集 / 分片上传 / 实用性（G3）

> 只读普查。写于 2026-09-22，基于 `api-v1/g-galgame` @ `ef0c9fb3`（= master）。没有连库、没有起服务，全部结论来自代码。
> 需要生产库才能确认的数字集中在 §12。
> 本域有两套都叫 `ResourceHandler` 的类型：`internal/galgame/handler.ResourceHandler`（作品下载资源）和 `internal/toolset/handler.ResourceHandler`（工具集资源）。下文分别写作 **GalgameResourceHandler** 与 **ToolsetResourceHandler**。

## 0. 范围与总览

`routes.golden` 里属于本段的路由共 **27 条**（9 GET / 7 POST / 8 PUT / 3 DELETE）。其中 11 条挂在 `/api/galgame*` 与 `/api/admin/galgame*`，15 条挂在 `/api/toolset*`，1 条寄挂在 `/api/user/:id/toolsets`。

| # | 方法 | 路径 | handler | 档位 | 拥有者 |
|---|---|---|---|---|---|
| 1 | GET | `/api/galgame-resource` | `galgame/handler/resource_handler.go:25` | optional | galgame 资源 |
| 2 | GET | `/api/galgame-resource/:id` | `resource_handler.go:38` | optional | galgame 资源 |
| 3 | GET | `/api/galgame-resource/:id/detail` | `resource_handler.go:55` | optional | galgame 资源 |
| 4 | GET | `/api/galgame/:gid/resource/all` | `resource_handler.go:69` | optional | galgame 资源 |
| 5 | POST | `/api/galgame/:gid/resource` | `resource_handler.go:89` | required | galgame 资源 |
| 6 | PUT | `/api/galgame/:gid/resource` | `resource_handler.go:106` | required | galgame 资源 |
| 7 | DELETE | `/api/galgame/:gid/resource` | `resource_handler.go:121` | required | galgame 资源 |
| 8 | PUT | `/api/galgame/:gid/resource/like` | `resource_handler.go:158` | required | galgame 资源 |
| 9 | PUT | `/api/galgame/:gid/resource/valid` | `resource_handler.go:173` | required | galgame 资源 |
| 10 | PUT | `/api/galgame/:gid/resource/expired` | `resource_handler.go:189` | required | galgame 资源 |
| 11 | PUT | `/api/admin/galgame/:gid/resource-publish-ban` | `resource_handler.go:140` | required + `perm.GalgameBanResourcePublish` | galgame 资源 |
| 12 | GET | `/api/toolset` | `toolset/handler/toolset_handler.go:23` | optional | toolset |
| 13 | GET | `/api/toolset/:id` | `toolset_handler.go:78` | optional | toolset |
| 14 | GET | `/api/user/:id/toolsets` | `toolset_handler.go:39` | public | toolset 寄挂 |
| 15 | POST | `/api/toolset` | `toolset_handler.go:60` | required | toolset |
| 16 | PUT | `/api/toolset/:id` | `toolset_handler.go:91` | required | toolset |
| 17 | DELETE | `/api/toolset/:id` | `toolset_handler.go:113` | required | toolset |
| 18 | GET | `/api/toolset/:id/resource/detail` | `toolset/handler/resource_handler.go:23` | **public**（`api` 组，无 OptionalAuth） | 工具集资源 |
| 19 | POST | `/api/toolset/:id/resource` | `resource_handler.go:36` | required | 工具集资源 |
| 20 | PUT | `/api/toolset/:id/resource` | `resource_handler.go:59` | required | 工具集资源 |
| 21 | DELETE | `/api/toolset/:id/resource` | `resource_handler.go:77` | required | 工具集资源 |
| 22 | POST | `/api/toolset/:id/upload/init` | `toolset/handler/upload_handler.go:23` | required | 分片上传 |
| 23 | POST | `/api/toolset/:id/upload/complete` | `upload_handler.go:46` | required | 分片上传 |
| 24 | POST | `/api/toolset/:id/upload/resume` | `upload_handler.go:64` | required | 分片上传 |
| 25 | POST | `/api/toolset/:id/upload/abort` | `upload_handler.go:81` | required | 分片上传 |
| 26 | GET | `/api/toolset/:id/practicality` | `toolset/handler/practicality_handler.go:22` | optional | 实用性 |
| 27 | PUT | `/api/toolset/:id/practicality` | `practicality_handler.go:32` | required | 实用性 |

路由注册在 `internal/app/router.go:142`（工具集资源详情，public）、`:200–202`（galgame-resource 三读）、`:219`（`/galgame/:gid/resource/all`）、`:234–236`（toolset 列表/详情/实用性读）、`:92`（`/user/:id/toolsets`）、`:309–314`（galgame 资源写）、`:342–352`（toolset 写 + 上传）、`:391–394`（admin 发布禁止）。中间件链以 `apps/api/internal/app/testdata/routes.golden` 为准：全部 `/api` 都先过 `cors.New → middleware.NamePreference → ContentStance`；optional 再加 `OptionalAuth`；required 再加 `Auth`；11 号再加 `RequirePermission`。

**信封**：全部端点走旧信封 `pkg/response`。成功 `200 {code:0, message:"成功", data:…}`，`Paginated` 是 `data:{items,total}`（只用于 toolset 两条列表）。错误 `pkg/response.Error` → `{code, message}` + 一个 HTTP 状态码。本域体码只有 `205`（401 登录失效，来自 `MustGetUser`）、`233`（400/403/404/422/500 混用）、`234`（403 封禁）。信任闸 `gate.ErrContentBlocked` 是 `422/233 "内容包含违禁词，无法发布"`（`trust/gate/compose.go:9`）。零个 problem+json、零个 `object` 判别字段、零个字符串 id、零个 `viewer` 块。若干写面走 `OKMessage`（有 `message` 无 `data`）。

**v1 现状**：`/api/v1` 下 galgame 族目前只有 `GET /galgames/{galgame_id}/moyu-patches`（`galgame/apiv1/register.go:16`）。本段 27 条**一条都没有迁**。

**`gid` 不是 catalog work id。** 论坛 `galgame.id` / 路径 `:gid` / 字段 `galgame_id` 是论坛自己的作品 id。catalog 的作品 id 在 brief 里叫 `WorkID`（`catalog_wire.go:13–17` 的注释写着 10,289 个数值撞车）。本段几乎处处用论坛 gid；唯一把论坛 gid 当成 catalog work id 往上游送的是 `claimOnFirstResource`（§3.1）。

---

## 1. 共性（27 条都受影响）

### 1.1 路径参数大量是假的

handler 读的是 query / body 里的 id，路径段经常从头到尾没被读过。调用方因此必须两边都写。

| 路由 | 路径段 | 真正读的 |
|---|---|---|
| `GET /api/galgame/:gid/resource/all` | `:gid` | query `galgame_id`（`dto.GalgameResourcesRequest`，`resource_dto.go:9`） |
| `POST/PUT/DELETE /api/galgame/:gid/resource` 及 `/like` `/valid` `/expired` | `:gid` | body/query 的 `galgame_id` / `galgame_resource_id` |
| `GET /api/galgame-resource/:id` | `:id` **有读**（`strconv.Atoi(c.Params("id"))`） | 网页额外传了没用的 `?resource_id=`（`pages/galgame/resource/[id]/index.vue:14`） |
| `GET /api/galgame-resource/:id/detail` | `:id` **有读** | 网页额外传了没用的 `?galgame_resource_id=` |
| `GET /api/toolset/:id/resource/detail` | `:id` | query `toolset_resource_id`（`toolset/dto/resource_dto.go:8`） |
| `PUT/DELETE /api/toolset/:id/resource` | `:id` | body/query `toolset_resource_id` |
| `POST /api/toolset/:id/upload/{complete,resume,abort}` | `:id` | body `artifact_uuid` |
| `POST /api/toolset/:id/upload/init` | `:id` 传到 service 后**未使用**（`upload_service.go:59–112` 的 `toolsetID` 形参无引用） | body `filename`/`filesize` |
| `GET /api/toolset/:id` / `PUT` / `DELETE` / practicality / `POST …/resource` | `:id` **有读** | 网页仍额外传 `?toolset_id=`（`pages/toolset/[id]/index.vue:19`） |
| `DELETE /api/toolset/:id` | `:id` **有读** | 网页额外传 `?toolset_id=`（`Detail.vue:47`） |

后果与用户域 `/floating` 同构：`GET /api/galgame/5/resource/all?galgame_id=3` 返回 **3** 号作品的资源。`GET /api/toolset/999/resource/detail?toolset_resource_id=1` 返回 **1** 号工具资源的下载地址，与路径上的工具无关。

### 1.2 `NamePreference` / `IsSFW` / `X-Kungal-Nsfw`

- 整个 `/api` 组挂了 `middleware.NamePreference`（`router.go:47`），它读 `KUNGalgameSettings` cookie 的 `showKUNGalgamePreferOriginalName`（`pkg/utils/settings.go:52`），放进 context。本段 galgame 资源读面会经 `CatalogItemToBrief` → `it.Names(ctx)`（`catalog_wire.go:459–498`）吃到这个偏好，作品名随 cookie 变。01 §3：「v1 不走 `NamePreference`」。
- `utils.IsSFW`（`settings.go:42`）本段**只有** `GET /api/galgame-resource` 调用（`resource_handler.go:31`）。已登录走 `content.FromCtx` 的账户姿态；匿名走 cookie `showKUNGalgameContentLimit`，取值不是 `nsfw`/`all` 就当 SFW。`X-Kungal-Nsfw` 已被 Bearer 姿态层明确忽略（`middleware/bearer_stance_test.go:138`）。
- 其余 26 条**不读** `IsSFW`。详情、下载、按作品列出、全部 toolset 面，匿名/SFW 读者拿得到 NSFW 作品下的资源和提取码。

### 1.3 封禁用户渲染闸

`userclient.IsRenderable` 是 `u.Status == 0`（`pkg/userclient/hydrate.go:48`）。本段调用点：

| 位置 | 调了？ |
|---|---|
| `hydrateCards`（资源浏览/搜索）`resource_service.go:167` | 是，`continue` 掉 |
| `GetResourceDetail` `resource_service.go:194–196` | 是，改成 not-found |
| `GetResourceDownloadDetail` `:233–235` | 是，改成 404 |
| `GetGalgameResources` `:274` | 是，`continue` |
| `buildRecommendations` `:705` | 是，`continue` |
| `ToolsetService.GetList` `toolset_service.go:110` | 是，`continue` |
| `ToolsetService.GetDetail` `:194` | 是，改成 404（只查作者） |
| `ToolsetService.GetDetail` 贡献者循环 `:202–204` | **否**，封禁贡献者照样下发 |
| `Toolset ResourceService.GetResourceDetail` `:63–65` | 是，改成 404 |
| `CommentService.GetLatestForDetail` `:49` | 是，`continue` |

`Hydrate` 把 OAuth 错误丢掉（`hydrate.go:37` `users, _ :=`），缺失 id 填 `Placeholder`（`userclient.go:211`，`Status` 零值 = 0）。于是 **OAuth 故障期间 `IsRenderable` 对占位用户为 true**，封禁作者的资源继续出现，名字变成「已注销用户」。`GetResourceDetail` / `GetResourceDownloadDetail` 用的是 `s.userClient.User(ctx, id)` 的 `owner, _, _ :=`（`resource_service.go:194` / `:233`），错误与 `found` 一起丢掉，零值 `User{Status:0}` 同样 fail-open。

### 1.4 Bearer 与 staff

写面的「版主可改别人的」全部走 `user.Can(perm.*)`（`middleware/auth.go:61`，`viaBearer` 恒 false）。admin 发布禁止走 `RequirePermission` → `staffCandidate`（`middleware/role.go:50`），Bearer 直接 `403/233 "管理操作请在网页端进行"`。本段**没有** `perm.CanUser` / `role.Can*` 直接打在 `user.Roles` 上。这一条是对的。

Bearer 登录用户仍可：发/改/删**自己的**资源与工具、点赞、报失效、打实用性、走上传（但过不了 `ToolsetUploadBypass` 额度豁免）。

### 1.5 POST 无幂等键

本段全部 POST（创建资源、创建工具、创建工具资源、upload 四条）都不读 `Idempotency-Key`。01 §5 K12 要求全部 POST 支持该头。

Galgame 资源的萌萌点还用 `moemoepoint.KeyNonce`（`pusher.go:125`，键里带 `UnixNano`），同一次创建/点赞的重试在 OAuth 侧是不同键。工具集创建/删除用的是稳定 `Key("toolset_create", id)`（`toolset_service.go:158`），这一半是对的。

---

## 2. Galgame 资源读面（4 条）

链路共用 `galgame/service/resource_service.go` + `repository/resource_repo.go` + `resource_mapper.go`。表 `galgame_resource` / `galgame_resource_link` / `galgame_resource_like`，作品名与 SFW 判决来自 catalog brief，本地 `galgame.content_limit` 是 display_nsfw 的缓存（迁移 079）。

### 2.1 `GET /api/galgame-resource`（全站资源墙）

`GalgameResourceHandler.GetResourceList`（`resource_handler.go:25`）→ `ResourceService.GetResourceList`（`resource_service.go:83`）。

**调用方**：

- `apps/web/app/pages/galgame/resource/index.vue:14`（`page`/`limit=50`/`keywords`）。
- `apps/web/server/**`：零 API 调用。搜 `galgame-resource`、`/galgame/`+`resource`，唯一命中是 sitemap 的**页面路径** `server/utils/kunSitemapSources.ts:207`（`path: '/galgame-resource'`），不是这个 API。
- Flutter：`packages/kungal_api/lib/kungal_api.dart` 是空 `library;`；`apps/kungal/lib/features/galgame/presentation/galgame_screen.dart:8` 是占位 `Text('galgame')`。Dart 源里零路径片段。文档 `kungal-apps/docs/tickets/02-forum-direct-api-prereqs.md:84-87` 与论坛 `docs/proj/app-direct-api.md:118-120` 把下载面列为前瞻契约，不是现网调用。
- 搜索页的「Galgame 资源」tab 打的是 `GET /api/search?type=resource`（`search/handler/search_handler.go:118` → 同一套 `ResourceService.Search`），**不是**本路由。

**请求**：`dto.ResourceListRequest`（`resource_dto.go:3`）query `page` `limit` `keywords`。

- `page`/`limit` 带 `validate:"min=1"`（limit 还有 `max=50`），省略则为 0 → `400/233 "请求参数验证失败"`。schema 上看着可选，实际必填。网页始终传。
- `keywords` 可选，`max=107`。空则走全表分页；非空则 `strings.Fields` 拆词后走搜索。

**响应**：`dto.ResourceListPage`（`resource_dto.go:163`）`{resources, total}`。项是 `ResourceCard`（`:71`）：`id/view/galgame_id/user{id,name,avatar}/type/title/version_label/language/platform/languages/platforms/runtimes/size/status/download/like_count/is_liked/comment_count/dlsite_* /link_domain/provider_names/note/note_html/created/edited/galgame_name`。

卡片的 `link_domain` 在 `rowToCard` 里写死 `""`（`resource_mapper.go:96`）。`code`/`password`/`link[]` **不下发**。`note_html` 是服务端 `markdown.RenderHardWrap`。`is_liked` 来自 `FindLikedSet`（optionalUID=0 时全 false）。`status` 是裸整数（0 有效 / 1 失效）。`view`/`created`/`edited`/`user`/`is_liked` 全部违反 01 §3 命名表。

**分页**：`ORDER BY galgame_resource.created DESC`（`resource_repo.go:63`），**没有 `id` tie-breaker**。搜索分支在有 catalog 命中时前面加 `relevance DESC`（`:97`），同样没有 id。

**SFW 闸**（本段唯一一条在 SQL 里做的）：

```go
return q.Joins("JOIN galgame g ON g.id = galgame_resource.galgame_id").
    Where("g.content_limit IS NULL OR g.content_limit = 'sfw'")
```

（`resource_repo.go:44–49`，注释写明 count 与 page 必须同一谓词。）`content_limit` 是 catalog **display_nsfw** 的本地缓存（`079_galgame_content_limit_cache.up.sql:1`，`galgame_catalog_mirror.go:44`），**不是** `content_rating` 年龄轴。NULL 放行（尚未同步）。这一条轴是对的。

关键词半边会再打 catalog `CatalogWorksSearch`，`ApplyWorksGate(q, isSFW)`（`resource_service.go:117`）给上游也带 `content_limit=sfw`。catalog 失败则 `MatchedGalgameIDs` 回 `nil`（`:120`），搜索退化为只匹配本地 `note ILIKE`，**不报错**。

**`total` 与 `items` 不同谓词**：SQL `COUNT(*)` 按 content_limit 计；Go 里 `hydrateCards` 再丢掉「作者 `!IsRenderable`」和「catalog brief 缺失」（`resource_service.go:167–173`）。一页要 50 条可能回 40 条，分页器按 `total` 画。catalog 把 hidden claim（`claim_state=hidden`，`catalog_wire.go:334`）从 brief 里摘掉，这些行计入 total 但不会出现在 items。

**错误**：query 校验失败 → `400/233`。service **永不返回 error**（签名有 `*AppError`，实现恒 `nil`）。库挂了 `CountAll`/`ListPaginated`/`Scan` 的 error 全部丢掉（`resource_repo.go:54`/`65`），结果是空列表 + total=0。

**可见性缺口**（相对详情页更严，相对「隐藏/封禁作品」仍松）：

- 不看 `galgame.published`。
- 不看 `resource_publish_banned`（禁止发布之后，已有资源仍出现在墙上）。
- 不看 `galgame_resource.status`（失效资源混在最新列表里）。
- 不看 catalog `claim_state=hidden`（只在 hydrate 丢 brief 时顺便消失，total 仍算它们）。

**测试**：无本路由测试。`resource_service_test.go` 只测 `claimRefusedByState`。`resource_axes_test.go` 测写轴解析。`filesize_test.go` 测体积。`vocab_test.go` 测兼容标量。

### 2.2 `GET /api/galgame-resource/:id`（资源页）

`GetResourceDetail`（`resource_handler.go:38`）→ `resource_service.go:185`。

**调用方**：`apps/web/app/pages/galgame/resource/[id]/index.vue:13`。响应类型写成 `GalgameResourcePageData | 'not found'`，因为 200 的 data 可以是字符串 `"not found"`。Nitro / Flutter 零调用（搜路径片段 `galgame-resource/` 与 handler 名）。

**请求**：路径 `:id`。无 query schema。网页传的 `?resource_id=` 被忽略。

**副作用**：找到且作者可渲染之后 `IncrementView`（`resource_repo.go:209`，`UPDATE … view = view + 1`），error 丢掉。GET 非幂等。

**响应**：`dto.ResourceDetailPage`（`resource_dto.go:157`）`{galgame, resource, recommendations}`。

- `galgame` 是 `ResourceGalgameSummary`：`id/name/effective_banner_*/content_limit/view/resource_update_time/original_language/age_limit/platform[]/language[]/type[]`。`age_limit` 来自 catalog `content_rating` 的展示映射（`ageLimitFromRating`，`catalog_wire.go:288`），**不当闸**。`content_limit` 来自 brief 的 `contentLimitOf`（`:271`）：claim 上有 `sfw`/`nsfw` 用它，否则 **`content_rating=="r18"` 推导成 `nsfw`**（`contentLimitFromRating`，`:281`）。这是本段把年龄轴写进「编辑向 SFW 字段」的地方。
- `resource` 是 `ResourceMeta`。`rowToMeta` 把 **第一条下载 URL 全文**写进 `link_domain`（`resource_mapper.go:111–114`：`linkDomain = links[0]`）。字段名叫 domain，值是完整磁链/网盘 URL。`code`/`password`/`link[]` 仍不下发，但第一条 URL 已经泄漏。网页 `detail/Info.vue:19` 在 `provider_names` 为空时把 `link_domain` 当供应商名显示。
- `recommendations` 最多 6 条同作品其它资源（`FindRecommendations`，`resource_repo.go:138`，`ORDER BY like_count DESC`，无 id tie-breaker，**无 SFW、无 status 过滤**）。

**找不到时回 200**：`notFound != nil` → `response.OK(c, "not found")`（`resource_handler.go:49–50`）。HTTP 200、`code:0`、`data:"not found"`。作者被封禁走同一条（服务层把 `!IsRenderable` 译成 `ResourceNotFound`）。id 非数字才是 `400/233 "无效的资源 ID"`。

**可见性**：不查 `content_limit`、不查 `published`、不查 `status`、不查 catalog hidden（brief 缺失时 summary 是零值，**资源本体照样下发**）。SFW 读者打开一条 NSFW 作品下的资源页，前端靠 `content_limit === 'nsfw'` 关 SEO（`[id]/index.vue:20`），API 本身把整页 JSON 给出去。

**测试**：无。

### 2.3 `GET /api/galgame-resource/:id/detail`（下载详情）

`GetResourceDownloadDetail`（`resource_handler.go:55`）→ `resource_service.go:224`。

**调用方**：

- `apps/web/app/components/galgame/resource/detail/Info.vue:59`
- `apps/web/app/components/galgame/resource/LinkDetailModal.vue:91`
- 两者都额外传 `query: { galgame_resource_id }`，后端不读。
- `docs/proj/app-direct-api.md:119` 列为 App 下载入口（前瞻）。Flutter 源码零引用。

**请求**：路径 `:id`。匿名可打（optional）。

**副作用**：`IncrementDownload`（`resource_repo.go:213`）。每次打开「显示链接」计数 +1。`app-direct-api.md:115` 写明续传不要重复打这条——这条是外链/磁链，没有续传；真正受这条约束的是工具集那条。

**响应**：`ResourceDownloadDetail`（`resource_dto.go:132`）= `ResourceMeta` + `link[]` + `code` + `password`。`link` 是数组却用单数。`code` 是网盘提取码，`password` 是解压密码。匿名、不登录、不看 SFW、不看 hidden/banned/expired，只要知道资源 id 就能拿到全部下载地址和密码。

作者不可渲染 → `404/233 "未找到该资源"`（与 2.2 的 200 `"not found"` **裁决不同**）。id 非数字 → `400/233`。行不存在 → `404/233`。

**测试**：无。

### 2.4 `GET /api/galgame/:gid/resource/all`（作品页资源列表）

`GetGalgameResources`（`resource_handler.go:69`）→ `resource_service.go:248`。

**调用方**：`apps/web/app/components/galgame/resource/Resource.vue:27`，同时传路径 `gid` 和 query `galgame_id`（两边必须相同才能对上）。`docs/proj/app-direct-api.md:118` 列为 App 读面（前瞻）。

**请求**：只校验 query `galgame_id`（必填 `min=1`）。路径 `:gid` 不读。

**响应**：裸数组 `[]ResourceCard`（不是 `{items,total}`）。`FindByGalgameID`（`resource_repo.go:129`）`ORDER BY status ASC, RANDOM()`：有效在前，失效在后，同组每次刷新重排。无分页。含失效资源。`hydrateCards` 的 brief 缺失跳过**没有**发生在这里——本函数只跳 `!IsRenderable`，catalog 拉不到 brief 仍下发卡片，只是 DLsite 三字段为空（`resource_service.go:271–279`）。

**SFW / hidden**：不查。作品页本身可能已按 SFW 藏了入口，但本 API 对任意 `galgame_id` 匿名可打，NSFW/hidden 作品的资源卡片（含 `provider_names`、备注 Markdown）直接返回。

**错误**：缺 `galgame_id` → `400/233`。service 恒 `nil` error。

**测试**：无。`docs/proj/checks/get.md:50` 记过 `isLiked` 曾经 hardcode false，已改为 `FindLikedSet`；路由因此必须挂在 `optAuth`（`router.go:196–199` 的注释）。

---

## 3. Galgame 资源写面（6 条 + 1 条 admin）

DTO 在 `galgame/dto/resource_dto.go`。链接字段 `link` 带 `validate:"required,min=1,max=20,dive,downloadlink"`，方案见 §8。体积走 `filesize.Parse`（只接受 `N[.NN] MB|GB`，`filesize.go:8`；`820 KB` 失败）。轴走 `parseResourceAxes`（`resource_axes.go:19`）：类型/语言/平台/运行环境是封闭词表，未知值 **400**，不静默回落。`VersionLabelKeys`（`resourcevocab/vocab.go:28`）在全仓只有定义、没有引用，服务端接受任意单行 ≤64 字。

信任闸：`resourceModerationText(note, links)`（`resource_service.go:75`）→ `check.Decision`；deny → `422/233`；hold 仍写入并 `ScanBg`。

权限 helper 一律 `user.Can`（Bearer 无 staff）。

### 3.1 `POST /api/galgame/:gid/resource`

`CreateResource`（`resource_handler.go:89`）→ `resource_service.go:284`。

**调用方**：`LinkEditModal.vue:182`（`method: 'POST'`，body 带 `galgame_id`）。路径 `:gid` 不读。

**请求**：`CreateGalgameResourceRequest`（`resource_dto.go:13`）。`galgame_id` 必填。`type` 必填但 DTO 无 `oneof`（封闭在 `parseResourceAxes`）。`title` max 200，`version_label` max 64，`size` 必填，`code`/`password` max 1007，`note` max 10000，`link` 1–20 条 downloadlink。`language`/`platform` 是遗留标量；`languages`/`platforms`/`runtimes` 是新轴。新轴为空时回退遗留标量（`resource_axes.go:40–56`）。

**不检查作品是否存在于 catalog。** `PublishLocal`（`galgame_repo.go:90`）`ON CONFLICT` upsert 一行 `galgame(id, published=true)`。对一个从未出现过的 gid 发资源，会**当场种出**本地 `galgame` 行并把 `published` 置 true（迁移 078 的粘性 SEO 旗）。`IsResourcePublishBanned` 对不存在的行 `Pluck` 空切片，返回 false（`resource_repo.go:116`），禁发闸也绕过。

成功后事务内：插资源、写 `provider` text[] 与 `provider_name` jsonb、插 link、`resource_count + 1`、`resource_update_time`、萌萌点 `RewardCreateResource = 3`（`constants/moemoepoint.go:6`），ref 是 `galgame_resource:<新id>`（`resource_service.go:369`，注释说明曾经误用 gid）。事务外 `claimOnFirstResource`。

**`claimOnFirstResource`（`resource_service.go:389`）把论坛 gid 当成 catalog work id**：

```go
if _, appErr := adoptAndPublish(ctx, s.catalog, accessToken, int64(gid)); appErr != nil {
```

`adoptAndPublish`（`submission_service.go:111`）的形参就叫 `workID`，打的是 catalog `/v2` 用户认领/发布。forum gid 与 catalog work id 有 10,289 个数值撞车（任务书；`catalog_wire.go:13–17`）。这条会认领**另一个作品**，或对一个不存在的 work id 失败后只 `slog.Warn`（`:398`）。方案③写明用户不认领游戏、旧认领道已撤；这里仍在「第一份资源」时静默认领。`accessToken == ""`（Bearer 且没带网页那种 OP cookie token 的路径）则整段跳过（`:390`）。

响应：`OKMessage "资源创建成功"`，无 id。网页靠 refresh 列表拿到新行。

错误：体积/轴/downloadlink/校验 → `400/233`；禁发 → `403/233 "该游戏已被禁止发布下载资源"`；信任 deny → `422/233`；事务失败 → `500/233 "创建 Galgame 资源失败"`。

**测试**：无创建测试。`TestClaimRefusedByState`（`resource_service_test.go:10`）只测 409 被当成「已有归属」。`claim_unclaimed_test.go` 测的是投稿向的 `adoptAndPublish`，不是本路由。

### 3.2 `PUT /api/galgame/:gid/resource`

`UpdateResource`（`resource_handler.go:106`）→ `resource_service.go:409`。`canModerate = user.Can(perm.ResourceEditAny)`。

**调用方**：`LinkEditModal.vue:182`（编辑时 PUT）；`user/Resource.vue:75`（个人主页改自己的链接，同时再打 `/valid`）。

**请求**：`UpdateGalgameResourceRequest`（`resource_dto.go:30`）。真正的主键是 body `galgame_resource_id`。body `galgame_id` **有字段、不写入**（`UpdateFields` 的 map 不含它，`:448`）。路径 `:gid` 不读。作者或 `resource.edit_any`。禁发中的作品连编辑都 403（迁移 061 注释写明「Create/Update 拒，DELETE 仍允许」）。

删光旧 link 再插新的（同一事务）。不改 `user_id`/`galgame_id`/`status`。

响应：`OKMessage "资源更新成功"`。

错误：不存在 `404/233 "未找到这个 Galgame 资源"`；非作者且无 perm `403/233`；其余同创建。

### 3.3 `DELETE /api/galgame/:gid/resource`

`DeleteResource`（`resource_handler.go:121`）→ `resource_service.go:492`。query `galgame_resource_id`。`user.Can(perm.ResourceDeleteAny)`。

**调用方**：`detail/Info.vue:34`；`LinkDetailModal.vue:120`。

扣萌萌点 `-(like_count + 5)`（`:504`）。创建时奖的是 **3**（`RewardCreateResource`）。前端文案写「扣除 5 萌萌点」（`Info.vue:29`、`LinkDetailModal.vue:114`）。创建 3、删除 5，对不上。键是 `KeyNonce`，同一资源删两次（第二次已 404）不会重复扣；第一次的扣款与创建的 +3 在 OAuth 侧对不上账。

不把 `galgame.published` 拨回去（078 粘性旗，这一条是对的）。`resource_count - 1` 无下限，重复/错扣能把计数打到负数。

禁发中仍可删。响应 `OKMessage "资源已删除"`。

### 3.4 `PUT /api/galgame/:gid/resource/like`（真·开关）

`ToggleLike`（`resource_handler.go:158`）→ `resource_service.go:517`。body `galgame_resource_id`。

**调用方**：`Like.vue:32`。前端自己乐观更新 `isLiked`，后端不回新状态。

已赞则删赞 `delta=-1`，否则加赞 `delta=+1`。给作者 `AdjustMoemoepoint` 同一 delta（`ReasonLiked`）。**不能给自己赞**（`400/233 "您不能给自己的资源点赞"`）。

无论赞还是取消，都调用 `CreateGalgameMessageWithContent(..., "liked", …)`（`:554`）。消息表按 `(sender, receiver, type, link)` 去重（`interaction.go:30`），取消赞不会发第二条，也不会删除第一条。取消赞仍走「liked」类型。

`FindLike` 把任意 error 当成「没赞过」（`resource_repo.go:279`）。并发双赞：唯一索引 `(galgame_resource_id, user_id)`（`000_baseline.up.sql:3672`）让第二笔 Create 失败 → 整单 500。

响应 `OKMessage "操作成功"`，不回 `is_liked`/`like_count`。PUT 开关，01 §5 要求拆成 PUT/DELETE 槽位。

### 3.5 `PUT /api/galgame/:gid/resource/valid`（标记，非开关）

`MarkValid`（`resource_handler.go:173`）→ `resource_service.go:567`。body `galgame_resource_id`。作者或 `resource.edit_any`。把 `status` 写成 **0**。已经是 0 再打一次也是 0（幂等写入）。任何人可报失效、只有作者/版主能改回有效——注释写的就是这个原因（`:564`）。

**调用方**：`Link.vue:102`；`user/Resource.vue:79`（与 PUT 资源并行）。

响应 `OKMessage "资源已标记为有效"`。

### 3.6 `PUT /api/galgame/:gid/resource/expired`（标记，非开关）

`MarkExpired`（`resource_handler.go:189`）→ `resource_service.go:581`。**任意登录用户**。已是 `status==1` → `400/233 "该资源已经被标记为失效"`（不是幂等）。

若配置了 `linkChecker` 且有链接：`CheckShare`（`pkg/linkcheck/linkcheck.go:71`）聚合结果。**任一条 `alive` 则整单视为活**（`:115`），返回 `{verdict:"alive", marked:false}`，**不改库**。全死才标失效。checker 出错/未配置 → `verdict==""`，照样标失效（fail-closed 到「算它死了」）。

前端 `useReportResourceExpired.ts:31` 读 `marked===false` 当「仍可用」。文案声称「17 天内不换链接就删除」（`:24`）；全仓 `cron/**` 与 `internal/infrastructure/cron/cron.go` **没有**按失效时间删资源的任务。这句文案没有实现。

标成功时给作者发 `type="expired"` 消息。响应 `{verdict, marked}`（`ReportExpireResult`，`resource_dto.go:60`），不是 `OKMessage`。

### 3.7 `PUT /api/admin/galgame/:gid/resource-publish-ban`

`SetResourcePublishBan`（`resource_handler.go:140`）。**路径 `:gid` 有读**。body `{banned: bool}`，无 validate tag，缺字段就是 Go 零值 `false`（解禁）。`c.Bind().Body` 失败 → `400/233 "请求格式错误"`。

`RequirePermission(perm.GalgameBanResourcePublish)`（`router.go:393`，`perm.go:41`）。Bearer 过不了 `staffCandidate`。

`SetResourcePublishBanned`（`resource_repo.go:122`）upsert `galgame` 行，只保证 `id` 与 `resource_publish_banned`。对一个不存在的 gid 会种出一行几乎全零的本地作品并把禁发打开。不回读 catalog。

**调用方**：`apps/web/app/components/galgame/Header.vue:52`。Nitro / Flutter 零调用。

响应按 `banned` 两条中文 `OKMessage`。不回当前状态。

---

## 4. 工具集（6 条）

表 `galgame_toolset`（`toolset/model/toolset.go:8`）。`status` 整数，列表 `WHERE status != 1`（`toolset_repo.go:39`）；**全仓没有把 `galgame_toolset.status` 写成 1 的写路径**（搜 `toolset.*status` / `status.*toolset` 只命中 feed 回填的 `status <> 1`）。详情 `FindByID` 不看 status，status=1 的行仍可按 id 打开。删除是物理删（`DeleteByID`），不是改 status。

类型/语言/平台/版本在**前端**是封闭枚举（`apps/web/app/constants/toolset.ts` + `validations/toolset.ts`），**后端 DTO 无 `oneof`**（`toolset_dto.go:24`）。未知 `type=` 过滤器会精确匹配一个永远为空的集合；未知 `sort_field` **静默回落** `created`（`mapper.go:41`）；未知 `sort_order` 除 `asc` 以外全是 `DESC`（`:54`）。01 §4 要求未知枚举 400。

### 4.1 `GET /api/toolset`

`GetList`（`toolset_handler.go:23`）→ `toolset_service.go:74`。

**调用方**：`toolset/card/Container.vue:17`（浏览，带 type/language/platform/version/sort_field/sort_order/query/page/limit）；`search/List.vue:60`（搜索 tab `type==='toolset'` 时打的是 **本路由** `query={query,page,limit}`，不是 `/search`）。

**请求**：`ToolsetListRequest`（`toolset_dto.go:11`）。`page`/`limit` `min=1`（limit max 100）——省略 → 400。handler 里 `Page==0 → 1`、`Limit==0 → 24`（`:28–33`）在校验之后，**死代码**。`query` max 100，ILIKE 有 `escapeLike`（`toolset_repo.go:61`）。过滤器取值 `all` 或空表示不筛。

**响应**：`Paginated` `{items,total}`。项 `ToolsetCard`（`toolset_dto.go:46`）：`id/name/user/type/platform/language/version/view/download/comment_count/practicality_avg/resource_update_time`。`practicality_avg` 与 `resource_update_time` 的 Go 类型是 `any`（无评分时 JSON `null`）。`download` 是该工具下资源 `SUM(download)`。`user` 禁用名。

**分页**：`Order(sortField + " " + sortOrder)`（`toolset_repo.go:78`）。字段白名单 `created|view|name|resource_update_time`，**无 id tie-breaker**。拼接前已白名单，不是注入。

**`total` vs `items`**：`CountFiltered` 含封禁作者；`GetList` 再 `continue` 掉 `!IsRenderable`（`toolset_service.go:110`）。与资源墙同一类分页器偏差。无 SFW（工具集没有 content_limit）。

**测试**：无列表测试。

### 4.2 `GET /api/user/:id/toolsets`

`GetUserToolsets`（`toolset_handler.go:39`）。public。把路径 `:id` 写进 `req.UserID` 再走同一 `GetList`。`fiber.Params[int]` 解不出或 ≤0 → `400/233 "无效的用户 ID"`。

**调用方**：`apps/web/app/components/user/Toolset.vue:14`（`page`/`limit=24`）。用户域普查已经把它记为寄挂。

无鉴权。看别人的工具列表。page/limit 同样因 `min=1` 必填，但网页传了。handler 的默认值仍是死代码。

### 4.3 `GET /api/toolset/:id`

`GetDetail`（`toolset_handler.go:78`）→ `toolset_service.go:176`。

**调用方**：`pages/toolset/[id]/index.vue:17`（额外 `query.toolset_id` 无用）。

不存在或作者不可渲染 → `404/233 "未找到该工具"`。不看 `status`。`IncrementView` 放在 `go` 里（`:198`），失败无日志。

**响应**：`ToolsetDetailResponse`（`toolset_dto.go:69`）。`content_markdown`/`content_html` 来自 `description` 列（命名已经是 v1 方向，但 HTML 仍服务端渲染）。`homepage` 解 jsonb，失败被 `_ = json.Unmarshal` 吞掉变成 `[]`（`:208`）。`resource` 是数组却用单数，项只有 `id/type/size/download/status`，**无下载 URL**。`download` 是 int64 总和，卡片上同名字段是 int。`contributors` 不跑 `IsRenderable`（`:202`）。`comment_preview` 最多 5 条 community 帖（`comment_service.go:27`），community 失败 slog.Warn 后回空数组。`created`/`updated`/`edited` 禁用名。`user` 禁用名。

### 4.4 `POST /api/toolset`

`Create`（`toolset_handler.go:60`）→ `toolset_service.go:119`。

**调用方**：`edit/toolset/Toolset.vue:44`。前端 zod 封闭 type/language/platform/version 与 URL homepage；后端 `CreateToolsetRequest` 只 `name` required、`description` max 2000、`version` max 233，`type`/`language`/`platform`/`homepage`/`aliases` **无词表、无条数上限、homepage 不校验 URL**。`json.Marshal(req.Homepage)` 错误丢掉（`:132`）。

信任闸过了才写。萌萌点 +3，稳定幂等键 `toolset_create:<id>`。响应整行 `GalgameToolset`（含 `description` 不叫 `content_markdown`、含 `user_id` 整数）。无 Idempotency-Key。

### 4.5 `PUT /api/toolset/:id`

`Update`（`toolset_handler.go:91`）。作者或 `user.Can(perm.ToolsetEditAny)`。路径 `:id` 有读。

**调用方**：`edit/toolset/Rewrite.vue:32`。body 仍带前端的 `toolset_id`，后端忽略。

不存在 404；非作者 403。信任闸 deny 422。`homepage` 同样 `json.Marshal` 丢错。响应 `OKMessage "工具更新成功"`。

### 4.6 `DELETE /api/toolset/:id`

`Delete`（`toolset_handler.go:113`）。作者或 `user.Can(perm.ToolsetDeleteAny)`。

**调用方**：`toolset/Detail.vue:46`。前端按 `3 + 资源数×3` 警告扣点（`:36`）；后端只 `adjustMoemoepoint(..., -3, Key("toolset_delete", id))`（`toolset_service.go:343`），**不**按资源再扣。资源行随 `DeleteAllRelated` 物理删。S3 遗留对象在事务里用 `context.Background()` 删（`:330`），失败只 Warn，事务继续，对象可能残留。

---

## 5. 工具集资源（4 条）

表 `galgame_toolset_resource`。真实唯一约束是 `(toolset_id, content)`（`000_baseline.up.sql:3732`）以及部分唯一 `artifact_uuid WHERE artifact_uuid <> ''`（`036_toolset_resource_artifact.up.sql:9`）。GORM 标签 `uniqueIndex:idx_toolset_resource` 只打在 `ToolsetID` 上（`model/toolset.go:78`），与库不符（本仓不开 AutoMigrate，标签是过时文档）。

`type` 封闭 `s3|user`（DTO `oneof`）。s3 的下载 URL 不落库，读时向 artifact 换预签名；`content` 对 s3 常为 `""`。于是 **同一个工具下第二个 `content=''` 的 s3 资源撞 `(toolset_id, content)` → 事务失败 → `500/233 "创建资源失败"`**。网页 UI 允许一个工具挂多个资源（`Detail.vue:110` `resources.value.push`）。

任何人登录都可以往**别人的工具**下 POST 资源（不查作者）；成功后 `AddContributor`。这是设计还是洞，生产数据里「非作者资源」的数量见 §12。

### 5.1 `GET /api/toolset/:id/resource/detail`

挂在 **public** `api` 组（`router.go:142`，`routes.golden:157` 链上没有 OptionalAuth）。不看登录、不看 SFW、不看工具 status。

`GetResourceDetail`（`toolset/handler/resource_handler.go:23`）→ `toolset/service/resource_service.go:54`。真正的键是 query `toolset_resource_id`。路径 `:id` 不读。

**调用方**：`toolset/resource/Item.vue:89`。`app-direct-api.md:120`：s3 的 `content` 是 artifact 预签名 URL，文档写 24h、支持 Range；实现把 `dl.Url` 写进 `resource.Content`（`:72`），`ExpiresAt` **丢掉不回**。换链失败 slog.Warn，仍返回行，`content` 保持库里的旧值（遗留 s3 可能是对象键）。

**副作用**：`go IncrementDownload`（`:68`）。每次「显示链接」+1。匿名可刷。

**响应**：把整个 `model.GalgameToolsetResource` 内嵌进 DTO（`resource_dto.go:35`），再加 `user`。于是 JSON 带 `id/content/type/artifact_uuid/code/password/size/note/download/status/edited/toolset_id/user_id/created/updated/user`。预签名 URL、提取码、解压密码对**完全匿名**的请求下发。`artifact_uuid` 一并泄漏。

作者不可渲染 → `404/233 "未找到该资源"`。

**测试**：无。

### 5.2 `POST /api/toolset/:id/resource`

路径 `:id` **有读**（这是本段少数路径段当真的写面），作为 `toolset_id`。工具不存在 → 404。不查作者、不查 status。

**调用方**：`LinkForm.vue:97`（`type=user` 自定义链接）；s3 则是 Upload 成功后同一表单带 `artifact_uuid`。

后端不验证 `artifact_uuid` 属于当前用户、不验证已经 Complete、不验证 size 对 s3 是纯数字（前端 zod 有，后端只有 `max=107` 字符串）。信任闸只拼 `content`+`note`，**不含** artifact uuid。萌萌点 +3，键 `resource_create:<resourceID>`，ref 却是 `toolset:<toolsetID>`（`resource_service.go:122`）——创建/删除的 ref 对不上（删除用 `toolset_resource:<id>`，`:225`）。

响应整行资源（无 user）。无幂等键。撞唯一约束 → 500。

### 5.3 `PUT /api/toolset/:id/resource`

body `toolset_resource_id`。路径 `:id` 不读。作者或 `user.Can(perm.ToolsetResourceEditAny)`。

`type=="user"` 才允许改 `content`/`code`/`size`；s3 只能改 `password`/`note`（`resource_service.go:173–177`）。不改 `type`/`artifact_uuid`。信任闸。响应刷新后的行。

**调用方**：`Item.vue:169`。

### 5.4 `DELETE /api/toolset/:id/resource`

query `toolset_resource_id`。作者或 `user.Can(perm.ToolsetResourceDeleteAny)`。

s3：优先 `art.Delete(artifact_uuid)`，否则遗留 `s3.Delete(code)`；失败 Warn，**仍然删行**。对象可能残留。萌萌点 -3，键 `resource_delete:<id>`。无事务包住「删对象 + 删行 + 扣点」。

**调用方**：`Item.vue:129`。前端文案「消耗 3 萌萌点」与后端一致（和工具删除的文案不一致，见 4.6）。

---

## 6. 分片上传（4 条 POST）

`UploadHandler` + `UploadService`（`toolset/service/upload_service.go`）。对象存在 infra artifact，论坛用自己的 client 凭据调，不转发用户 token。Init 把 `Public: true`（`:75`）交给 artifact。

四条都在 `authed` 组。路径 `:id` 对 complete/resume/abort **完全不读**；对 init 传到 service 后也不用。任意登录用户只要有 `artifact_uuid` 就能 complete/resume/abort **别人的**上传会话（所有权在不在 artifact 侧本普查看不到实现，论坛层没有）。

**调用方**全部在 `apps/web/app/components/toolset/resource/Upload.vue`：abort `:128`，complete `:288`，init `:327`，resume `:395`。Nitro / Flutter 零调用。

### 6.1 `POST /api/toolset/:id/upload/init`

body `filename` required、`filesize` required min=1（JSON 键是 `filesize` 不是 `file_size`）、`content_type` 可选。扩展名只允许 `.7z/.zip/.rar`（`parseArchiveFilename`，`:200`）。大小上限 `MaxLargeFileSize = 2GiB`（`:21`）。

额度：`user.Can(perm.ToolsetUploadBypass)` 则跳过。否则 `daily_toolset_upload_bytes + incoming > 100MiB + moemoepoint×1MiB` → `400/233 "超出今日上传额度, 请明天再试"`（`:42–56`）。**`moemoepoint` 读的是 `kungal_user_state` 缓存列**（C3 的配额误用，用户域普查已记）。`First` 找不到用户行时 `state` 零值，额度就是 100MiB。每日上海 00:00 cron 清零（`cron.go:142`）。

**不检查工具是否存在、不检查当前用户是不是作者。** 登录即可向 artifact 开一个 public 上传。响应 `UploadInitResponse`：`artifact_uuid/multipart/upload_url/part_size/parts[]/expires_at`。字段 `filesize`（请求）与 `file_size`（01 命名）不一致。

错误映射 `mapArtifactErr`（`:209`）：过大/额度/MIME/大小不符/停用 → 400；未配置 → 500；会话不存在 → 404；其余 → 500。全部仍是体码 233。

### 6.2 `POST /api/toolset/:id/upload/complete`

body `artifact_uuid` required、`parts[]` 可选（`part_number`/`etag` 无 validate）。不读路径、不校验 parts 非空。

成功后 `firstComplete`：Redis `SETNX toolset:upload:done:<uuid>` 24h（`:192`）。**Redis 出错返回 true**（`:194`），重试会再次把 `daily_toolset_upload_bytes` 加上文件大小。额度被放大。

响应 `{artifact_uuid, size}`。此时还没有 `galgame_toolset_resource` 行；前端再 POST `/resource` 把 uuid 绑上去。Complete 成功、Create 失败 → artifact 孤儿。

### 6.3 `POST /api/toolset/:id/upload/resume`

`MustGetUser` 之后**丢掉 user**（`upload_handler.go:65`）。任何人登录凭 uuid 换新的 part URL。无所有权。响应形状同 init，另加 `uploaded_parts[]`。

### 6.4 `POST /api/toolset/:id/upload/abort`

调用 `art.Delete`；失败 slog.Warn，**仍 `OKMessage "上传已取消"`**（`upload_service.go:185–189`）。前端以为取消了，对象可能还在。不校验该 uuid 是否属于当前用户。不回滚 `daily_toolset_upload_bytes`（本来也只在 complete 时加）。

**测试**：四条都无。

---

## 7. 实用性（2 条）

表 `galgame_toolset_practicality`（`rate` 1–5，`user_id`，`toolset_id`）。**没有 `(toolset_id, user_id)` 唯一索引**（baseline 只有 pkey + 两个 FK，后续迁移也没加）。GORM 模型同样没有 unique tag（`model/toolset.go:42`）。`Upsert` 是先 `Find` 再 `Create`/`Update`（`practicality_repo.go:73`），并发双 POST 可插入两行，平均值被同一人算两次。

两条都不检查工具是否存在。对不存在的 toolset_id，GET 回全 0；PUT 插入一行指向不存在工具的评分（FK 指向 `galgame_toolset(id)`，工具不在会 500）。

### 7.1 `GET /api/toolset/:id/practicality`

optional。路径 `:id` 有读。`optionalUID` 决定 `mine`。

**调用方**：`toolset/Detail.vue:81`（额外 `query.toolset_id` 无用）。详情页已经带了 `rating_counts`/`practicality_avg`，这条主要是为了 `mine`。

响应 `{counts: map[int]int64, avg: float64, mine: *int}`。JSON 对象键变成字符串 `"1"`…`"5"`。`counts` 固定填 1–5（缺的为 0）。`avg` 先 `AVG` 再 `Round(*100)/100`。`FindUserRating` 的非「找不到」错误被 `p, _ :=` 丢掉（`practicality_service.go:24`），库挂时 `mine` 为 null。

### 7.2 `PUT /api/toolset/:id/practicality`

required。body `{rate}` `min=1,max=5` required。作者、路人、Bearer 都可以打。把自己的星星写成 `rate`（整体替换，语义上幂等；并发见上）。响应 `OKMessage "评分成功"`，不回新的 avg。

**调用方**：`Detail.vue:100`。

**测试**：无。

---

## 8. 下载链接校验器

`downloadlink` 注册在 `pkg/utils/validate.go:18`。方案允许列表（`:28`）：

`http` `https` `ftp` `ftps` `magnet` `ed2k` `thunder`

`javascript:` / `data:` 拒绝。`url.Parse` 之后还要求 `Host`、`Opaque`、`RawQuery`、`Fragment` 至少一个非空，好让 `magnet:?xt=…` 通过（标准 `url` tag 会拒）。全角 `？` 的 magnet 也过（生产有 16 行，测试原文 `validate_test.go:13`）。空串、裸 host、`#anchor`、`ttps://…` 拒绝。

本段只有 galgame 资源的 `link[]` 使用该 tag。工具集 `type=user` 的 `content` 是逗号分隔 URL 字符串，**不走** downloadlink（前端 `applyResourceLinkBlur` 自己认）。

---

## 9. 命名问题清单（迁 v1 时逐条执行）

| 现状 | 问题 | v1 |
|---|---|---|
| 路径 `:gid`、字段无 `galgame_id` 对齐 | 禁用名；且常被忽略 | `{galgame_id}`，与字段同名 |
| `galgame_id`（论坛作品 id） | 易与 catalog work id 混淆 | 继续叫 `galgame_id`；catalog 的叫 `catalog_work_id`（01 §3） |
| `created` / `edited` / `updated` | 禁用名 | `created_at` / `edited_at` / `updated_at` |
| `view` / `download` | 禁用名 / 计数无 `_count` | `view_count` / `download_count` |
| `user` | 禁用名 | `author` |
| `is_liked` | 随查看者变 | `viewer.has_liked` |
| `status` 裸整数（资源 0/1，工具 0/1） | 与 Problem.status 撞型 | `state` 封闭字符串（`valid`/`expired`，工具若保留 hidden 再定） |
| `link`（数组） / `resource`（数组） | 数组用单数 | `links` / `resources` |
| `link_domain` | 实际是第一条完整 URL | 不要这个字段；供应商用 `provider_names` |
| `code` | 成功响应顶层禁用；此处是提取码 | `share_code` 或类似，避开顶层 `code` |
| `filesize`（上传请求） | 计数/大小命名 | `file_size` |
| `note_html` / `content_html` | 01 要结构化正文 | `content` 节点树；源继续 `content_markdown` |
| `resources`+`total` vs `items`+`total` vs 裸数组 | 三种集合信封 | 二选一，spec 声明 |
| `practicality_avg` 类型 `any` | 同一字段 null 或 number | `practicality_avg: number \| null` |
| `language`/`platform` 标量 + `languages`/`platforms`/`runtimes` 数组 | 两代轴并存 | 只保留数组；标量是兼容投影，v1 丢掉 |
| `/resource/like` `/valid` `/expired` `/upload/*` | 动词路径 | 资源槽位，见 §13 |
| `GET …/detail` | 动词/别名，实际是下载子资源 | `/download` 或单独下载表示 |

---

## 10. 疑似问题（平铺，不排序）

1. `GET /api/galgame-resource/:id` 找不到时 `response.OK(c, "not found")`（`resource_handler.go:50`），200 + 字符串；同资源的下载面是 404。
2. `rowToMeta` 把第一条下载 URL 写入 `link_domain`（`resource_mapper.go:113`），资源页匿名可读。
3. `GET /api/galgame-resource/:id/detail` 匿名下发 `link[]`/`code`/`password`，不查 SFW/hidden/banned/expired（`resource_service.go:224`）。
4. `GET /api/galgame/:gid/resource/all` 不查 SFW/hidden，路径 `:gid` 不读（`resource_dto.go:9`）。
5. 资源墙 `hydrateCards` 丢行导致 `total`≠`items`（`resource_service.go:167`）。
6. `ORDER BY created DESC` / `like_count DESC` / `RANDOM()` 均无 id tie-breaker（`resource_repo.go:63` `:142` `:133`）。
7. 资源搜索 `note ILIKE` 不转义 `%`/`_`（`resource_repo.go:77`）；工具列表有 `escapeLike`。
8. catalog 搜索失败静默退化（`MatchedGalgameIDs` `:120` 回 nil）。
9. `CountAll`/`ListPaginated`/`Scan`/`IncrementView`/`IncrementDownload`/`IsResourcePublishBanned` 的 error 全部丢掉。
10. `fetchGalgameBriefs` / `GetBatchPublic` `briefMap, _ =`（`resource_service.go:626` `:641`）。
11. `User(ctx, id)` 的 `_, _` fail-open（`:194` `:233`）；`Hydrate` 同样（`hydrate.go:37`）。
12. `Placeholder.Status==0` 让「已注销」占位可渲染（`userclient.go:211` + `hydrate.go:48`）。
13. `CreateResource` 不验证 catalog 作品存在，`PublishLocal` 可种出本地 `galgame` 行（`galgame_repo.go:90`）。
14. `claimOnFirstResource` 把论坛 gid 当 catalog work id（`resource_service.go:393`）。
15. 创建奖 3 萌萌点、删除扣 `like_count+5`（`constants/moemoepoint.go:6` vs `resource_service.go:504`）。
16. 创建/点赞/删除 galgame 资源的幂等键是 `KeyNonce`（`pusher.go:125`），每次不同。
17. `PUT …/like` 是开关；取消赞仍写 `type="liked"` 消息（`resource_service.go:554`）。
18. 前端「17 天不换链接就删除」无 cron（`useReportResourceExpired.ts:24`）。
19. `MarkExpired` 在 checker 未配置时直接标死（`resource_service.go:593`）。
20. `SetResourcePublishBan` 对不存在的 gid upsert 本地行（`resource_repo.go:122`）。
21. body `{banned}` 缺省就是 false（`resource_handler.go:136`）。
22. 失效资源进入全站墙与推荐（无 `status` 过滤）。
23. `contentLimitFromRating` 用 `content_rating==r18` 填 `content_limit`（`catalog_wire.go:281`）。
24. 工具集 `GET …/resource/detail` 无 OptionalAuth，匿名预签名 URL + 计数 +1（`router.go:142`，`resource_service.go:68`）。
25. 该详情路径 `:id` 不读，只认 `toolset_resource_id`（`resource_dto.go:8`）。
26. 上传 init 的 `toolsetID` 形参未使用（`upload_service.go:59`）。
27. complete/resume/abort 不校验上传归属。
28. `Abort` 失败仍 200（`upload_service.go:186`）。
29. `firstComplete` Redis 出错当首次（`:194`），额度可被加两次。
30. 上传额度读本地 `moemoepoint` 缓存（`:47`）。
31. `(toolset_id, content)` 唯一导致同一工具第二个空 content 的 s3 资源 500（`000_baseline.up.sql:3732`）。
32. 工具集创建资源不查是否作者（`resource_service.go:86`）。
33. 创建/删除工具资源的 moemoepoint `ref` 不一致（`toolset` vs `toolset_resource`）。
34. 删工具前端按 3+3n 扣点、后端只扣 3（`Detail.vue:36` vs `toolset_service.go:343`）。
35. `GetList` 过滤 `status != 1` 但无人写入 status=1；详情不看 status（`toolset_repo.go:39` `:84`）。
36. `sort_field`/`sort_order`/`type` 未知值静默回落（`mapper.go:41` `:54`，`toolset_repo.go:40`）。
37. handler 里 page/limit 默认值是死代码（`toolset_handler.go:28`），校验 `min=1` 已经拒了 0。
38. 工具卡片 `total` 含封禁作者、items 不含（`toolset_service.go:110`）。
39. 贡献者列表不跑 `IsRenderable`（`:202`）。
40. `homepage` `json.Marshal`/`Unmarshal` 丢错（`:132` `:208`）。
41. 工具 type/language/platform/version 后端开放、前端封闭（`toolset_dto.go:24` vs `validations/toolset.ts:12`）。
42. 实用性表无 `(toolset_id, user_id)` 唯一约束；`Upsert` 非原子（`practicality_repo.go:73`）。
43. 实用性 GET/PUT 不验证工具存在。
44. `GetPracticality` `FindUserRating` 丢错（`practicality_service.go:24`）。
45. POST 均无 `Idempotency-Key`。
46. GET 浏览/详情/下载带计数副作用（view/download）。
47. 全 `/api` 挂 `NamePreference`（`router.go:47`），资源墙上的 `galgame_name` 随 cookie 变。
48. `VersionLabelKeys` 从未用于校验（`vocab.go:28` 全仓唯一引用是定义）。
49. 工具 s3 `size` 存的是字节数字符串，galgame 资源 `size` 是 `N MB`；同一字段名两种语义。
50. `GalgameResourceHandler.GetResourceDetail` 与 `ToolsetResourceHandler.GetResourceDetail` 同名、不同鉴权、不同键（路径 vs query）。

---

## 11. 数据库表清单

**本段写：**

| 表 | 列 / 操作 | 备注 |
|---|---|---|
| `galgame_resource` | INSERT/UPDATE/DELETE；`view`/`download`/`like_count`/`status`/`provider`/`provider_name`/轴列 | 论坛自有 |
| `galgame_resource_link` | 整表替换 | 唯一 `(galgame_resource_id, url)` |
| `galgame_resource_like` | 开关插入/删除 | 唯一 `(galgame_resource_id, user_id)` |
| `galgame` | `published=true`（首次资源）、`resource_count`、`resource_update_time`、`resource_publish_banned` | `published` 粘性；`content_limit` 本段只读 |
| `galgame_toolset` | CRUD；`view++`；`resource_update_time` | `status` 列在，写路径不碰 |
| `galgame_toolset_alias` | 整表替换 | 唯一 `(toolset_id, name)` |
| `galgame_toolset_contributor` | 创建工具/资源时加 | 唯一 `(toolset_id, user_id)` |
| `galgame_toolset_resource` | CRUD；`download++` | 唯一 `(toolset_id, content)`；部分唯一 `artifact_uuid` |
| `galgame_toolset_practicality` | upsert | **无**用户+工具唯一 |
| `galgame_toolset_category_relation` | 随工具删除 | 本段读路径不碰分类 |
| `kungal_user_state` | `daily_toolset_upload_*` += ；读 `moemoepoint` | 缓存列当额度 |
| `message` | liked / expired | 去重键含 link=`/galgame/<gid>` |

**本段读：** `galgame.content_limit`（079 缓存）、catalog briefs（作品名/banner/DLsite/claim）、OAuth `users/batch`（作者）、artifact（预签名）、community（工具详情评论预览）、linkcheck（报失效）。

**不在本地的**：作品元数据在 catalog；工具文件在 artifact；评论在 community；萌萌点在 OAuth。

---

## 12. 需要生产库回答的问题

请按下面的顺序跑，每条对应上面的一个裁决。库是论坛的 `kungalgame`。

1. **资源 `status` 分布**（墙要不要带失效）：
   `SELECT status, count(*) FROM galgame_resource GROUP BY 1;`

2. **资源类型/语言/平台/运行环境实际取值**（v1 封闭词表；未知值不要建模）：
   `SELECT type, count(*) FROM galgame_resource GROUP BY 1 ORDER BY 2 DESC;`
   `SELECT language, count(*) FROM galgame_resource GROUP BY 1 ORDER BY 2 DESC;`
   `SELECT platform, count(*) FROM galgame_resource GROUP BY 1 ORDER BY 2 DESC;`
   `SELECT langs, count(*) FROM (SELECT jsonb_array_elements_text(languages) AS langs FROM galgame_resource) t GROUP BY 1 ORDER BY 2 DESC;`
   平台/运行环境同样从 `platforms`/`runtimes` 拆。空数组行数：
   `SELECT count(*) FROM galgame_resource WHERE languages = '[]'::jsonb;`
   `SELECT count(*) FROM galgame_resource WHERE platforms = '[]'::jsonb AND runtimes = '[]'::jsonb;`
   （098 注释写 emulator 有 3459 行轴为空。）

3. **`title` / `version_label` 空与自由文本**：
   `SELECT count(*) FROM galgame_resource WHERE title = '';`
   `SELECT version_label, count(*) FROM galgame_resource GROUP BY 1 ORDER BY 2 DESC LIMIT 30;`

4. **提取码/密码非空**（下载面泄漏面有多大）：
   `SELECT count(*) FROM galgame_resource WHERE coalesce(code,'') <> '' OR coalesce(password,'') <> '';`
   `SELECT count(*) FROM galgame_resource_link;`
   `SELECT count(*) FROM galgame_resource WHERE id IN (SELECT galgame_resource_id FROM galgame_resource_link WHERE url ILIKE 'magnet:%');`

5. **全角 `？` magnet**（校验器特判）：
   `SELECT count(*) FROM galgame_resource_link WHERE url LIKE '%？%';`

6. **禁发旗与粘性 published**：
   `SELECT resource_publish_banned, count(*) FROM galgame GROUP BY 1;`
   `SELECT published, count(*) FROM galgame GROUP BY 1;`
   `SELECT count(*) FROM galgame g WHERE g.published AND NOT EXISTS (SELECT 1 FROM galgame_resource r WHERE r.galgame_id = g.id);`  —— 删光资源仍 published 的行。
   `SELECT count(*) FROM galgame_resource r LEFT JOIN galgame g ON g.id = r.galgame_id WHERE g.id IS NULL;` —— 孤儿（FK 在应该是 0）。

7. **资源墙 SFW 缓存 NULL**：
   `SELECT count(*) FROM galgame WHERE content_limit IS NULL;`
   `SELECT content_limit, count(*) FROM galgame GROUP BY 1;`

8. **点赞规模**（开关并发、删除扣点数）：
   `SELECT count(*) FROM galgame_resource_like;`
   `SELECT percentile_cont(0.99) WITHIN GROUP (ORDER BY like_count) FROM galgame_resource;`

9. **工具 `status` / 类型 / 语言 / 平台 / 版本**：
   `SELECT status, count(*) FROM galgame_toolset GROUP BY 1;`
   `SELECT type, count(*) FROM galgame_toolset GROUP BY 1 ORDER BY 2 DESC;`
   `SELECT language, count(*) FROM galgame_toolset GROUP BY 1 ORDER BY 2 DESC;`
   `SELECT platform, count(*) FROM galgame_toolset GROUP BY 1 ORDER BY 2 DESC;`
   `SELECT version, count(*) FROM galgame_toolset GROUP BY 1 ORDER BY 2 DESC;`
   —— status=1 若为 0，v1 不要这个状态；type 出现 `engine` 或自由字符串则后端词表不能照抄前端 CONST。

10. **工具资源 type 与空 content**：
    `SELECT type, count(*) FROM galgame_toolset_resource GROUP BY 1;`
    `SELECT count(*) FROM galgame_toolset_resource WHERE type = 's3' AND content = '';`
    `SELECT toolset_id, count(*) FROM galgame_toolset_resource GROUP BY 1 HAVING count(*) > 1;`
    `SELECT toolset_id, count(*) FROM galgame_toolset_resource WHERE content = '' GROUP BY 1 HAVING count(*) > 1;` —— 若 >0，唯一约束已经在挡第二份或历史有重复。

11. **非作者发布的工具资源**（5.2 是否被用过）：
    `SELECT count(*) FROM galgame_toolset_resource r JOIN galgame_toolset t ON t.id = r.toolset_id WHERE r.user_id <> t.user_id;`

12. **实用性重复行**：
    `SELECT count(*) FROM galgame_toolset_practicality;`
    `SELECT count(*) FROM (SELECT toolset_id, user_id FROM galgame_toolset_practicality GROUP BY 1,2 HAVING count(*) > 1) d;`
    `SELECT rate, count(*) FROM galgame_toolset_practicality GROUP BY 1 ORDER BY 1;`
    `SELECT count(*) FROM galgame_toolset_practicality p LEFT JOIN galgame_toolset t ON t.id = p.toolset_id WHERE t.id IS NULL;`

13. **上传额度缓存**：
    `SELECT count(*) FROM kungal_user_state WHERE daily_toolset_upload_bytes > 0;`
    `SELECT percentile_cont(0.99) WITHIN GROUP (ORDER BY daily_toolset_upload_bytes) FROM kungal_user_state;`

14. **孤儿 artifact_uuid**（Complete 成功、Create 失败）：这张表看不出来，只能看 artifact 侧未绑定对象。论坛侧能看的是：
    `SELECT count(*) FROM galgame_toolset_resource WHERE type = 's3' AND artifact_uuid = '';` —— 遗留双读行。
    `SELECT count(*) FROM galgame_toolset_resource WHERE type = 's3' AND artifact_uuid <> '';`

15. **`galgame_toolset.comment_count` 与 community 是否还对齐**（详情用列，预览另拉 community）：
    `SELECT count(*) FROM galgame_toolset WHERE comment_count > 0;`
    对账只能抽样。

---

## 13. 拟议 v1 资源地图（提案，不是契约）

路径用复数 kebab-case，路径参数与字段同名。开关拆成槽位。动词路径改成资源。`{galgame_id}` 是论坛作品 id。

| 旧 | 拟议 v1 | 理由 |
|---|---|---|
| `GET /api/galgame-resource` | `GET /v1/galgame-resources` | 全站可浏览集合，页码分页 |
| `GET /api/galgame-resource/:id` | `GET /v1/galgame-resources/{galgame_resource_id}` | 详情；找不到用 `NOT_FOUND`，不要 200 字符串 |
| `GET /api/galgame-resource/:id/detail` | `GET /v1/galgame-resources/{galgame_resource_id}/download` | 下载子资源；谁可读在契约里显式规定 |
| `GET /api/galgame/:gid/resource/all` | `GET /v1/galgames/{galgame_id}/resources` | 作品的子集合；读路径参数 |
| `POST /api/galgame/:gid/resource` | `POST /v1/galgames/{galgame_id}/resources` | 创建挂在作品下；Idempotency-Key |
| `PUT /api/galgame/:gid/resource` | `PUT /v1/galgame-resources/{galgame_resource_id}` | 主键在路径，不再 body 里另带一份 |
| `DELETE /api/galgame/:gid/resource` | `DELETE /v1/galgame-resources/{galgame_resource_id}` | 同上 |
| `PUT …/resource/like` | `PUT`/`DELETE /v1/galgame-resources/{galgame_resource_id}/likes/me` | 开关拆槽 |
| `PUT …/resource/valid` | `PUT /v1/galgame-resources/{galgame_resource_id}/state` body `{state:"valid"}` | 标记，不是开关 |
| `PUT …/resource/expired` | `POST /v1/galgame-resources/{galgame_resource_id}/expiry-reports` | 报失效是新资源，活链不改 state |
| `PUT /api/admin/galgame/:gid/resource-publish-ban` | `PUT /v1/galgames/{galgame_id}/resource-publish-ban` | 子资源旗，staff only |
| `GET /api/toolset` | `GET /v1/toolsets` | 浏览集合 |
| `GET /api/user/:id/toolsets` | `GET /v1/toolsets?owner_id={user_id}` | 「某人的 X」是过滤器 |
| `GET /api/toolset/:id` | `GET /v1/toolsets/{toolset_id}` | 详情 |
| `POST /api/toolset` | `POST /v1/toolsets` | Idempotency-Key |
| `PUT /api/toolset/:id` | `PUT /v1/toolsets/{toolset_id}` | 整体替换 |
| `DELETE /api/toolset/:id` | `DELETE /v1/toolsets/{toolset_id}` | 物理删保留，契约写清副作用 |
| `GET /api/toolset/:id/practicality` | `GET /v1/toolsets/{toolset_id}/practicality` | 汇总 + `viewer.mine` |
| `PUT /api/toolset/:id/practicality` | `PUT /v1/toolsets/{toolset_id}/practicality-ratings/me` | 当前用户的一票 |
| `GET /api/toolset/:id/resource/detail` | `GET /v1/toolset-resources/{toolset_resource_id}/download` | 键在路径；鉴权档位单独决定 |
| `POST /api/toolset/:id/resource` | `POST /v1/toolsets/{toolset_id}/resources` | 挂在工具下 |
| `PUT /api/toolset/:id/resource` | `PUT /v1/toolset-resources/{toolset_resource_id}` | 主键在路径 |
| `DELETE /api/toolset/:id/resource` | `DELETE /v1/toolset-resources/{toolset_resource_id}` | 同上 |
| `POST …/upload/init` | `POST /v1/toolset-uploads` | 上传会话是资源；body 带 `toolset_id` 并校验归属 |
| `POST …/upload/complete` | `POST /v1/toolset-uploads/{artifact_uuid}/complete` | 子资源动作 |
| `POST …/upload/resume` | `POST /v1/toolset-uploads/{artifact_uuid}/resume` | 同上 |
| `POST …/upload/abort` | `DELETE /v1/toolset-uploads/{artifact_uuid}` | 取消就是删会话 |

不要把 `claimOnFirstResource` 带进 v1。`galgame.published` 继续由第一份资源置位、不随删除清除。SFW 用显式查询参数（如 `include_nsfw=true`）替换 cookie / `IsSFW`。下载面必须单独规定：匿名是否能拿 `links`/`share_code`/`password`、hidden/NSFW 是否 404。
