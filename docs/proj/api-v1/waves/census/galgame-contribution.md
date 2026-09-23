# 普查 · Galgame 投稿 / 认领审核队列 / 资料编辑引擎（G2）

> 只读普查。写于 2026-09-22，基于 `api-v1/g-galgame` @ `ef0c9fb3`（= master）。没有连库、没有起服务、没有跑测试，全部结论来自代码。
> 需要生产库才能确认的数字集中在 §11。catalog 侧的 `claim_state` / 提案 `status` 不在本库。

本段 22 条旧路由分成三族：**投稿**（SubmissionHandler）、**认领审核队列**（ClaimReviewHandler）、**资料编辑引擎**（EditHandler，代理 infra catalog 编辑面）。评论、资源、评分、浏览列表、`GET /api/galgame/:gid` 详情都不在本段。

---

## 0. 范围与总览

`routes.golden` 里属于本段的路由共 **22 条**（11 GET / 9 POST / 2 DELETE）。

| # | 方法 | 路径 | handler | 档位 | golden |
|---|---|---|---|---|---|
| 1 | POST | `/api/galgame/submit` | `submission_handler.go:30` `Submit` | required | 245 |
| 2 | POST | `/api/galgame/:gid/resubmit` | `submission_handler.go:50` `Resubmit` | required | 242 |
| 3 | DELETE | `/api/galgame/:gid` | `submission_handler.go:66` `Withdraw` | required | 12 |
| 4 | DELETE | `/api/galgame/:gid/draft` | `submission_handler.go:81` `DeleteDraft` | required | 14 |
| 5 | GET | `/api/galgame/mine` | `submission_handler.go:96` `ListMine` | required | 122 |
| 6 | GET | `/api/galgame/audited` | `submission_handler.go:108` `ListAudit` | required + `galgame.claim.review` | 112 |
| 7 | GET | `/api/galgame/search/wizard` | `submission_handler.go:120` `SearchWithPending` | required | 125 |
| 8 | GET | `/api/admin/galgame/submissions` | `claim_review_handler.go:19` `PendingQueue` | required + `galgame.claim.review` | 46 |
| 9 | POST | `/api/admin/galgame/:gid/review` | `claim_review_handler.go:32` `Review` | required + `galgame.claim.review` | 218 |
| 10 | GET | `/api/galgame/:gid/edit/bootstrap` | `edit_handler.go:325` `Bootstrap` | required | 106 |
| 11 | POST | `/api/galgame/:gid/edit/proposals` | `edit_handler.go:378` `Submit` | required | 239 |
| 12 | GET | `/api/galgame/:gid/edit/proposals` | `edit_handler.go:708` `GameProposals` | **public** | 108 |
| 13 | GET | `/api/galgame/:gid/edit/revisions` | `edit_handler.go:448` `Revisions` | optional | 109 |
| 14 | GET | `/api/galgame/:gid/edit/diff` | `edit_handler.go:540` `Diff` | **public** | 107 |
| 15 | POST | `/api/galgame/:gid/edit/revert` | `edit_handler.go:502` `Revert` | required | 240 |
| 16 | GET | `/api/galgame-edit/mine` | `edit_handler.go:635` `Mine` | required | 69 |
| 17 | GET | `/api/galgame-edit/queue` | `edit_handler.go:611` `Queue` | required + `RequireModerator` | 71 |
| 18 | GET | `/api/galgame-edit/proposals/:id` | `edit_handler.go:736` `ProposalDetail` | required | 70 |
| 19 | POST | `/api/galgame-edit/proposals/:id/amend` | `edit_handler.go:774` `Amend` | required | 228 |
| 20 | POST | `/api/galgame-edit/proposals/:id/merge` | `edit_handler.go:809` `Merge` | required | 230 |
| 21 | POST | `/api/galgame-edit/proposals/:id/decline` | `edit_handler.go:861` `Decline` | required | 229 |
| 22 | POST | `/api/galgame-edit/proposals/:id/withdraw` | `edit_handler.go:902` `Withdraw` | required | 231 |

注册点：`router.go:148–158`（mine / audited / wizard，挂在 AUTH BOUNDARY 之前的 `userAuth`）、`:168–169`（diff / game proposals，完全公开）、`:225`（revisions，`optAuth`）、`:285–288`（submit / resubmit / withdraw / delete-draft）、`:331–340`（编辑写面 + mine / queue / detail）、`:385–389`（admin 投稿队列）。`GET /api/galgame/drafts`（`drafts_handler.go:18`）是**未发布作品浏览**，不是本段的投稿草稿，不要并进来。

**信封**：全部走旧信封 `pkg/response`。成功 `200 {code:0, message:"成功", data:…}`；`Withdraw` / `DeleteDraft` 走 `OKMessage`，**没有 `data`**。错误 `{code, message}`，编辑写面拒绝时额外带 `errors[]`（`errors.go:8–14` 的 `FieldError`）。体码：`205`（401）、`233`（400/403/404/409/500/503 混用）、`234`（封禁）、`235`（缺 `catalog:edit`，要重新登录）、`236`（同名嫌疑）。零个 problem+json、零个 `object` 判别字段、零个字符串 id、零个 `viewer` 块。

**两套修订系统，不要并成一套**：

| | 资料编辑引擎（本段 HTTP 路由） | 时间线镜像 `galgame_activity` |
|---|---|---|
| 权威 | infra catalog `/v2/me/proposals`、`/v2/moderation/*`、`/v2/catalog/revisions` | 本地表，给首页动态用 |
| 提案 id | catalog `edit_proposal.id` | 历史列 `wiki_pr_id`（026）；提交提案时 `edit_handler.go:438–441` **把 catalog 提案 id 写进这列** |
| 修订 id | catalog `edit_revision.id`（跨实体族的全局计数） | `edit_revision_id`（067，另一命名空间）+ 历史 `wiki_revision_id`（wiki `galgame_revision.id`） |
| 每作序号 | `seq` | `wiki_revision_number`（032；引擎的 `seq` 复用这个列名） |
| 谁在写 | 本段 13 条编辑路由 | cron `GalgameEditRevisionSync`（`galgame_edit_revision_sync.go:42`）+ 提交提案的 `submitSideEffects` |

本地 `galgame_pr` / `galgame_history` 表在迁移 005 已经 `DROP`。wiki 修订 feed 已退役。HTTP 路由读的永远是 catalog 引擎；动态卡片再回头打 `/edit/diff`。

**`gid` 与 catalog work id 是两套整数，约 10,289 个数值撞车。** 翻译入口是 `workIDOf` / `gidOf` / `CatalogWorkIDs` / `GIDsByCatalogIDs`（`edit_handler.go:112–115` 的注释把漏译的后果写死了：不会 404，会改到另一部作品）。认领过的 kungal 作品以 `claim.site_work_id` 为 gid；从未认领的作品以 catalog id 自己当 gid（`catalog_wire.go:375–395`）。`SubmitResult` 在 v2 改写后把 `GID` 和 `WorkID` 填成**同一个** registry 发行号（`user_claims.go:40`）。

**v1 现状**：`/api/v1` 下没有本段任何操作。

---

## 0.1 贯穿三族的事实（后面不重复）

**凭证。** 写面一律 `userToken` → `middleware.GetAccessToken`（`edit_handler.go:293–298`），空则 `401/205 "用户登录失效"`。catalog 用户面用会话 / Bearer 里的 OP access token；应用 key 只用于公开读（修订列表、作品提案列表、diff、vocabularies）和 `PendingQueue` 的 catalog 搜索。Bearer 请求能打通所有 `Auth` 写面（投稿、编辑提案），**打不通** `RequirePermission` / `RequireModerator`（`role.go:50–58` `staffCandidate`：`viaBearer` → `403/233 "管理操作请在网页端进行"`）。检查用的是 `user.Can` / `user.CanModerate`（`auth.go:61–67`），没有一处对 `user.Roles` 直接 `perm.CanUser` / `role.Can*`。

**`NamePreference`。** `/api` 组全挂（golden 每一行，本段 22 条都在）。它读 `KUNGalgameSettings` cookie 的 `ShowKUNGalgamePreferOriginalName`（`namepref.go:12–15` → `utils.PrefersOriginalName`）。向导搜索命中的 `GalgameBrief.Name` 经 `CatalogEntityNames`（`catalog_name.go:295–304`）吃这个偏好。v1 不读这颗 cookie。本段**没有任何 handler 调用 `utils.IsSFW`**，也没有任何 handler 读 `X-Kungal-Nsfw`。

**`content_rating` 当 SFW 闸。** 向导把 catalog 行变成 `GalgameBrief` 时走 `contentLimitOf`（`catalog_wire.go:271–279`）：claim 上没有 `sfw`/`nsfw` 就回落到 `contentLimitFromRating`，`r18` → `nsfw`。这是年龄轴冒充展示轴。投稿写入时两轴是分开的（`submission_payload.go:95–101`：`display_nsfw` 来自 `content_limit`，`content_rating` 来自 `age_limit`）。

**封禁用户。** 本段三个 service / `EditHandler.userMap` **零次**调用 `userclient.IsRenderable`。`userMap`（`edit_handler.go:73–93`）OAuth 失败只 `slog.Warn` 后回空 map，成功则原样下发 `name`/`avatar`，不管 `Status`。

**幂等。** 本段所有 POST 都不读 `Idempotency-Key`。合并提案的萌萌点键是 `kungal:galgame_edit_merged:<proposalId>`（`edit_handler.go:844–846`）；通过投稿的萌萌点键在 cron 里，是 `kungal:claim_approved:<eventId>`（`galgame_claim_event_sync.go:246–248`），不在 HTTP 请求路径上。

**错误翻译，投稿 / 审核共用 `claimActionError`（`submission_service.go:244–268`）：**

| 上游 | 论坛 |
|---|---|
| `ErrInsufficientScope` | `403/235`「投稿需要新的授权…」 |
| `ErrUnauthorized` | `401/205` |
| `ErrNotFound` | `404/233 "条目不存在"` |
| `ErrNotConfigured` | `503/233 "资料库服务暂不可用"` |
| HTTP 403 | `403/233 "你没有权限执行此操作"` |
| HTTP 422 | `400/233` + 上游 `Message` |
| HTTP 409（非 `DUPLICATE_SUSPECTS`） | `409/233` + 上游 `Message`（**英文诊断句进用户信封**） |
| 其余 | `503/233 "资料库服务暂不可用"` |

编辑用户面 `userEditError`（`edit_handler.go:268–290`）同一套，缺 scope 的文案换成「编辑资料需要新的授权…」，422 带上 `errors[]`。应用 key 那几条读面走 `editError`（`:240–260`）：401/403 被当成**部署的 key 坏了**，回 `503`，避免把用户踢去重新登录。

---

## 1. 投稿族（1–7）

### 1.1 状态机（catalog `claim_state`，权威在 infra）

常量 `catalogclient/claims.go:12–33`。论坛本地**没有** claim 表，状态只在 catalog。

| 值 | 含义（从代码与 UI 徽章对出来） | 谁写 |
|---|---|---|
| `none` | 未被 kungal 认领 | catalog；向导把 `claim==nil` 标成这个（`catalog_wire.go:476–478`） |
| `draft` | 草稿（撤回后、或尚未提交审核） | `PATCH /v2/me/claims/{id} {state:"withdrawn"}` 的目标（`user_claims.go:61–62` 把 withdraw 映射成 `withdrawn`，响应 `To` 以 catalog 为准） |
| `pending` | 待审核 | mint 默认；`Resubmit` → `state:"pending"` |
| `live` | 已发布 | 审核 `approve`；受信任提交者 mint 直达；资源通道 `adoptAndPublish` 的 publish 半边 |
| `declined` | 已拒绝 | 审核 `decline` |
| `hidden` | 已下架 / 封禁 | 审核 `ban` |

论坛执行的转移（HTTP 本段 + 资源通道，**不是** catalog 的全集）：

```
（空） --Submit--> pending | live
declined --Resubmit--> pending
pending|live --Withdraw--> draft     （catalog 拒绝 live 删除，见 DeleteDraft 注释）
draft --DeleteDraft--> （catalog 软删作品；本地 DELETE galgame 行）
pending --Review approve--> live
pending --Review decline--> declined
* --Review ban--> hidden
hidden --Review unban--> （catalog 决定；前端没有按钮）
unclaimed --adoptAndPublish claim+publish--> live   （资源通道，不在本段 HTTP）
```

未知 `claim_state` 查询值：`ListMine` 回 `400/233 "未知的申请状态: …"`（`submission_service.go:296–299`）。词表是 `claimStates`（`:279–286`），含 `none`/`live`/`hidden`，默认列表却只用 `pending,declined,draft`（`mineStates` `:273–277`）——已通过的作品不出现在「我的提交」，UI 文案也这么写（`Mine.vue:86`）。

`ListAudit` **不传** `claim_state`，「审核历史什么都不藏」（`submission_bearer_test.go:199–201`）。

未知审核 `action`：`400/233 "未知的审核动作"`（`claim_review_service.go:91–92`）。词表：`approve` / `decline` / `ban` / `unban`。前端只发前三个（`submissions.vue:123`）。

### 1.2 `POST /api/galgame/submit`

链路：`Submit` → `SubmissionService.Submit`（`submission_service.go:48`）→ `catalog.SubmitWorkUser` `POST /v2/me/claims`（`user_claims.go:23`）→ 可选 `SubmitLocal`（`galgame_repo.go:202`）→ 可选 `attachBanner`（再走编辑引擎 `CreateEditProposalUser` + `MergeEditProposalUser`）。

**调用方**：`apps/web/app/components/edit/galgame/Footer.vue:82`。页面 `apps/web/app/pages/edit/galgame/create.vue`（`auth` + `prevent`）。`apps/web/server/**` 零命中。Flutter 零命中（§0.2）。

**请求体** `SubmissionForm`（`submission_payload.go:43–65`），无 `validate` tag，handler 只 `c.Bind().Body`：

| 字段 | 真实含义 |
|---|---|
| `name_en_us` / `name_ja_jp` / `name_zh_cn` / `name_zh_tw` | 四语标题。`DisplayName()` 按 ja→zh-Hans→zh-Hant→en 取第一个非空（`:81–88`），写成 `catalog.work.display_name` |
| `intro_*` | 四语简介，非空才进 `catalog.work.intros` |
| `content_limit` | 封闭 `sfw`/`nsfw`。缺席或其它值 → `400/233 "请选择内容限制 (SFW 或 NSFW)"`（`:62–64`）。**不得**把缺席当 sfw（961 部封面被分级器标 explicit 的作品曾因此进 SFW 架） |
| `age_limit` | 只认 `"r18"` → `content_rating=2`；**其它一切含空串静默成 0**（全年龄）（`:97–101`） |
| `original_language` | 产品码 `ja-jp`/`zh-cn`/`zh-tw`/`en-us` 译成 BCP 47；未知值**原样透传**（`olangOf` `:18–26`） |
| `aliases` | `[]string`，`kind=1` 别名标题 |
| `release_date` | 解析成 `WorkSubmitDate` 后**丢掉**：v2 mint 没有 `released` 键，只 `slog.Warn`（`:69–77`） |
| `banner_hash` | 图床 hash，事后打补丁 `catalog.work.covers:[{image_hash}]` |
| `confirm_duplicates` | catalog 同名软门；第一次 409/`236`，用户确认后再发 `true` |

前端 Zod（`validations/galgame.ts:51`）还校验「至少一个标题 / 至少一个简介 / 别名 ≤30 条」。后端**不校验简介非空、不校验别名条数、不校验标题长度**。前端把 `release_date_tba` 放进 body（`Footer.vue:37`），`SubmissionForm` 没有这个字段，静默丢弃。

`Fields()` 写出的 catalog 键：`catalog.work.display_name` / `olang` / `display_nsfw` / `content_rating` / `titles` / `intros`。`SubmitWorkUser` 另外带顶层 `display_name` + `field_values`（`user_claims.go:24–28`）。不传 `site`、不传 `actor`、不传 `product_work_id`——id 由 registry 发行（`submission_submit_test.go:129–145`）。

**响应** `SubmitResult`（`submission_service.go:37–46`）：`gid`（int）、`work_id`（int64）、`claim_state`、`banner_attached`。v2 解码把 `GID` 和 `WorkID` 填成同一个 `parseFlexID(out.ID)`（`user_claims.go:39–40`）。`banner_attached=false` 时投稿仍然成立，只是封面没挂上。

**错误**（在 `claimActionError` 之上）：无标题 `400/233 "请至少填写一个语言的标题"`；`content_limit` 非法同上；`release_date` 格式 `400/233`；`DUPLICATE_SUSPECTS` → `409/236 "资料库中已有同名作品"`（`submitError` `:236–241`）。前端靠 `code===236` 弹确认框（`Footer.vue:90–96`）。

**鉴权**：`Auth`。Bearer 可以投稿（token 原样转给 catalog）。无 staff 闸。

**本地副作用**：`SubmitLocal` 在 `galgame` 表 `EnsureLocalStub` + `SetCreatorIfUnset`（`published` 仍是 false）。失败只 warn（`submission_service.go:89–92`），catalog 作品已在。`attachBanner` 失败回 `banner_attached:false` 并 `slog.Error`。

**测试**：`submission_submit_test.go`（mint id、field map、live 直达、236、banner 合并/已合并/失败、content_limit、无标题）、`submission_payload_test.go`（locale / cover keys / 空列表 / 日期精度 / 未知 olang）。无 handler 测试。

### 1.3 `POST /api/galgame/:gid/resubmit`

`act(..., ClaimActionSubmit)` → `PATCH /v2/me/claims/{workId} {state:"pending"}`。无请求体。路径 `:gid` 经 `submissionGID`（`Atoi`，`<=0` → `400/233 "无效的 Galgame ID"`）再 `workIDOf`（找不到 → `404/233 "条目不存在"`）。

**调用方**：`Mine.vue:45`，仅 `claim_state===declined` 时渲染按钮。

**响应**：`ClaimActionResult` `{work_id, from_state, to_state, event_id}`。网页 `kunFetch<unknown>`，不读 body。

**测试**：`submission_bearer_test.go:106` 覆盖的是 Withdraw 的 user-plane；Resubmit 与它共用 `act`。`TestClaimErrorsCarryTheTokenTaxonomy` 用 Resubmit 打 235/205。

### 1.4 `DELETE /api/galgame/:gid`（撤回，不是删作品）

`act(..., ClaimActionWithdraw)` → `PATCH … {state:"withdrawn"}`。成功 `OKMessage "撤回成功"`，无 data。网页 `kunFetch<string>`（`Mine.vue:17`）。

路径叫 DELETE 资源，语义是状态机撤回。与 `GET /api/galgame/:gid` 详情同 URL 不同方法。

**调用方**：`Mine.vue:17`，`claim_state!==draft` 且不是 declined（declined 走 resubmit）。

### 1.5 `DELETE /api/galgame/:gid/draft`

`DeleteDraft`（`submission_service.go:176`）：先 `DeleteMyClaim` `DELETE /v2/me/claims/{workId}`（catalog 只允许草稿；live/pending 要先撤回），成功后再 `DeleteLocalDraft`：

```sql
DELETE FROM galgame WHERE id = ? AND NOT EXISTS (SELECT 1 FROM galgame_resource r WHERE r.galgame_id = galgame.id)
```

（`galgame_repo.go:173–176`）。有资源的草稿**拒绝删本地行**（资源 `ON DELETE CASCADE`）。catalog 已经删了、本地还在时回 `500/233 "草稿已删除, 但本站条目清理失败, 请联系管理员"`。

**调用方**：`Mine.vue:67`，仅 `claim_state===draft`。

**测试**：`TestDeleteDraftStopsAtAnUpstreamRefusal`（`submission_submit_test.go:397`）；`TestDeleteLocalDraftRefusesARowThatCarriesAResource`（`galgame_repo_test.go:16`）。

### 1.6 `GET /api/galgame/mine` 与 `GET /api/galgame/audited`

共用 `listClaims` → `GET /v2/me/claims`（`user_claims.go:111`）。

查询：`claim_state`（CSV，仅 mine；未知值 400）、`before`（int64，非法/`atoiOr` 回 0）、`limit`（默认 20，非法回 20，**无上限、不报 `LIMIT_TOO_LARGE`**）。`kind=submitted`（mine）/ `audited`（audit）。

**响应** `UserClaimPage`（`claims.go:174–178`）：`items[]`、`next_before`、`total`。项 `UserClaimItem`（`:156–172`）：`work_id`、`display_name`、`site`、`product_work_id`、`claim_state`、`last_event_id` / `last_from_state` / `last_to_state` / `last_reason` / `last_actor_uid` / `last_event_at`、`first_acted_at`、`acted_count`。

**解码丢字段。** `v2Claim.item()`（`v2_user.go:271–284`）只填 `WorkID`（优先 `id` 再 `work_id`）、`DisplayName`、`ClaimState`、`LastToState`/`LastFromState`/`LastEventID`/`LastActorUID`/`ActedCount`。**不读、也不声明** `product_work_id` / `site` / `last_reason` / `last_event_at` / `first_acted_at`。于是 JSON 里 `product_work_id` 恒为 `null`。前端 `galgameClaimGid`（`galgameClaimState.ts:45–46`）是 `item.product_work_id ?? 0`，**故意不回落到 `work_id`**（注释写着两套 id 撞车会把「我的审核」链到无关作品）。结果：撤回 / 重提 / 删草稿 / 「继续编辑」按钮的 `v-if="galgameClaimGid(item)"`（`Mine.vue:126`）全部为假。向导 pending 的「继续编辑」同理（`Wizard.vue:144`）。向导测试只断言 `WorkID==64689`（`wizard_search_test.go:187`），不断言 `ProductWorkID`。

**`next_before` 恒 0。** `MyClaims`（`user_claims.go:134–141`）只抄 `Items` 和 `Total`，不把 v2 的 `next_cursor` 译成 `NextBefore`。前端 `hasMore` 是 `nextBefore>0 && items.length<total`（`useGalgameClaimList.ts:29–31`）。`total>20` 时「加载更多」永不出现，多出来的行用户看不到。

**调用方**：mine → `Mine.vue:3` → `useGalgameClaimList.ts:9`（`before:0, limit:20`），页面 `pages/edit/galgame/mine.vue`（`auth`）。audited → `Audited.vue:3`，页面 `pages/edit/galgame/audited.vue`（`permissions: ['galgame.claim.review']`）。

**鉴权**：mine 只有 `Auth`（golden:122，**没有** OptionalAuth）。audited 再加 `RequirePermission(perm.GalgameClaimReview)`（`router.go:150–153`）。Bearer 打 audited → 403「管理操作请在网页端进行」。

**可见性**：这两条列的是**调用者自己的** claim，不经过 SFW、不经过 `published`、不经过 `IsRenderable`。`hidden` 作品会出现在 audited 里。

**测试**：`submission_bearer_test.go` `TestListMineIsTheTokensOwnClaims` / `TestListAuditIsTheClaimsTheCallerReviewed`。无分页测试、无 `product_work_id` 测试。

### 1.7 `GET /api/galgame/search/wizard`

`SearchWithPending`（`submission_service.go:345`）：一半 `CatalogWorksSearch` `GET /catalog/works/search`（实际 `doV2` 改写到 `/v2/catalog/works`，`catalog_v2.go:67–68`），一半 `MyClaims` pending+declined。

查询：`q`、`page`（默认 1）、`limit`（默认 12）。**故意不传 `claim_state` / `claimed`**（`:365–370`：索引 facet 滞后一天）。`OpenPopulation` 无条件 `nsfw=true`（`catalog_face.go:50–52`），**不设 `content_limit`**——SFW 读者在向导里看到 NSFW 标题。`include=names,covers,refs`。

过滤在 Go 里：`CatalogItemWizardEligible`（`catalog_wire.go:351–364`）留下未认领，或 kungal/galgame_wiki 认领且 `site_work_id>0` 且状态 `live|draft|pending`。`hidden`/`declined`/外站认领/gid=0 丢掉。

**`total` 与 `items` 不同谓词。** `total` 是 catalog 搜索的 `res.Total`（过滤前）；`items` 是过滤后。测试夹具 `total:2` 对 8 行里留下 4 行（`wizard_search_test.go:115–127`）把这个差写进了断言。分页器会画错。无 `id` tie-breaker（页码交给上游）。

命中项 `GalgameBrief`（`client.go:194`）：`id`（gid 两规则）、`work_id`（catalog id）、`vndb_id`、`name`（经 `NamePreference`）、`status` int（0=live 否则 2）、`claim_state`、`content_limit`（可能从 `content_rating` 回落）、`age_limit`（从 `content_rating` 来，这次是年龄轴本身）、`user_id`（brief 里恒 0，ToBrief 没填）、封面 URL 等。禁用名：`gid` 没出现在 brief 里，但 `id` 就是论坛 gid；`status` 裸整数；`work_id` 没说是 catalog 的。

pending 半边与 §1.6 同一套残缺 `UserClaimItem`。

**调用方**：`Wizard.vue:35`（`q` + `limit:12`，不传 `page`）。页面 `pages/edit/galgame/publish.vue`。`gameHref` 是 `/galgame/${hit.id || hit.work_id}`（`Wizard.vue:44`）——未认领作品的 `id` 等于 catalog id，点进去走论坛详情，详情再按「未认领 catalog id = gid」那条规则找行。

**测试**：`wizard_search_test.go`（不传 claim_state、gid 键、丢掉 hidden/declined、pending 走 user plane、空数组）。

---

## 2. 认领审核队列（8–9）

### 2.1 `GET /api/admin/galgame/submissions`

`PendingQueue`（`claim_review_service.go:43`）→ `CatalogWorksList` `GET /catalog/works`（应用 key，不是审核者 token）查询：`claimed=true`、`claim_state=pending`、`site=kungal`、`sort=updated`、`limit`（默认 30，`atoiOr`）、`include=names,covers,refs`、`OpenPopulation`（又是无条件 `nsfw=true`）、可选 `cursor`。

**响应** `PendingQueuePage`：`items:[{gid,name,state,updated}]`、`next_cursor`。`gid` 来自 `row.GID()`（site_work_id 规则）。`name` 经 `Names(ctx)`，所以吃 `NamePreference`。`updated` 是 catalog 的 `Updated` 字符串。`CatalogWorksList` **不填 Total**（`catalog_face.go:572–576`），前端也没用 total，靠 cursor 拉下一页（`submissions.vue:44–58`）。cursor 无 `cur_` 前缀（G16）。无 tie-breaker（交给上游 `sort=updated`）。

**调用方**：`pages/admin/submissions.vue:28` / `:50`。导航 `constants/admin.ts:84` `permissions: ['galgame.claim.review']`。页面自己 `definePageMeta({ permissions: ['galgame.claim.review'] })`。

**鉴权**：`RequirePermission(perm.GalgameClaimReview)`。这是**纯查看门**（`permissions.md:21–28`）：列表用应用 key 读，不按审核者身份过滤。Bearer 403。

**可见性**：只列 `pending`。不看本地 `published`、不看 SFW。`gid` 翻译错会让预览打开另一部作品（`submissions.vue:248` `GalgamePreviewModal :gid`）。

**测试**：无 PendingQueue 测试。

### 2.2 `POST /api/admin/galgame/:gid/review`

body `{action, reason}`，无 validate。`action` 必须在 `{approve,decline,ban,unban}`。`decline` 且 `reason==""` → `400/233 "拒绝时必须填写理由"`（空格算有内容）。`:gid` → `CatalogWorkIDs` → `ActOnClaimUser` `POST /v2/moderation/claims/{workId}/decisions` `{decision, note}`（`user_claims.go:67–83`）。

响应 `ClaimActionResult`。`approve` 的 `To` 本地写死 `"live"`，`ban` 写死 `"hidden"`，`unban` 落在 default 也是 `"live"`（`:76–83`），**不读 catalog 响应体**。`from_state` / `event_id` 在这条路径上是 Go 零值。

通过后的 +3 萌萌点**不在这个 handler 里发**，由 claim-event cron 在 `pending→live` 时 `AwardSync`（`galgame_claim_event_sync.go:239`）。前端文案写「由资料库事件同步发放」（`submissions.vue:142`）。cron 停了（缺 `claim_events:read`）就通过了也不加分，HTTP 仍 200。

**调用方**：`submissions.vue:128`。`unban` 没有按钮。

**鉴权**：同 §2.1 的查看门；真正能不能裁决是 catalog 看 token 的 `catalog.claim.review`。本地门与 infra 门用**不同的键名**（见 §5）。

**测试**：`TestReviewVerdictSpeaksAsTheModerator`（`submission_bearer_test.go:130`）。

---

## 3. 资料编辑引擎（10–22）

全部代理 catalog 编辑引擎。实体族写死 `catalog.work`（`edit_handler.go:116`），租户写死 `site=kungal`（`:110`）。`:gid` 先译成 work id，漏译的后果是改到另一部游戏。

提案状态词表（Queue / GameProposals 校验，`edit_handler.go:613–617` / `:713–717`）：`open` / `merged` / `declined` / `withdrawn` / `""`。未知 → `400/233 "未知的提案状态"`。`Mine` **不校验** `status`（`:650` 原样透传），未知值交给 catalog（huma 丢未知参数或 400，论坛不翻译）。

v2 提案解码：`v2Proposal.proposal()`（`v2_user.go:163–208`）填 id/entity/patch/status/times/proposer。`status` 空则 `"open"`。**不填 `decided_by_uid`、`base_revision_seq`**——前端 `decided_by_uid` 用来取审核人头像（`mine/Container.vue:61–64`），会一直是 `undefined`。

### 3.1 `GET /api/galgame/:gid/edit/bootstrap`

用户面：`EditSnapshotUser` `GET /v2/moderation/snapshots/work/{workId}` + `GetEditSchemaUser` `GET /v2/catalog/schemas/work` + `Vocabularies` `GET /v2/vocabularies`（应用 key，失败 warn 后空 map，枚举字段变只读，`:347–354`）。

响应 `{gid, values, fields, vocabularies, can_review}`。`can_review` 是「schema 里是否存在任一 `CanReview` 字段」（`anyReviewable` `:364`），**不是**「当前用户能不能审这部作品」。前端拿它决定是否展示别人的待审提案（`edit/Container.vue:64–70`）。

**调用方**：`galgame/edit/Container.vue:17`。页面 `pages/galgame/[gid]/edit.vue`（`auth`）。

**测试**：`TestEditBootstrapShape`、`TestEditBootstrapProjectionFollowsThePlane`、`TestEditUserPlaneStaleGrantAsksForReauth`。

### 3.2 `POST /api/galgame/:gid/edit/proposals`

body `{patch: map, note}`。`patch` 空 → `400/233 "没有需要保存的修改"`。键必须带前缀 `catalog.work.`（`FieldKeyPrefix`）。`note` >2000 → 校验错误。无幂等键。

`CreateEditProposalUser` `POST /v2/me/proposals`。响应 `{merged, proposal, revision?}`。`merged=false` 时 `submitSideEffects`：通知条目 `creator_user_id`（`NotifyRequested`），并 `INSERT galgame_activity (wiki_pr_id, … type='GALGAME_PR_CREATION')`——**catalog 提案 id 写进 wiki 列**（`edit_handler.go:438–441`）。`gidOf` 失败则副作用整段跳过（`:425–427`）。

**调用方**：`edit/Container.vue:117`。

**测试**：`TestEditSubmitPassesThePatchThrough`、`TestEditSubmitPayloadOnTheUserPlane`、`TestEditSubmitRidesTheUserToken`、`TestEditSubmitLocalValidation`、`TestEditUserPlaneErrorPassThrough`。

### 3.3 `GET /api/galgame/:gid/edit/proposals`（公开）

应用 key `ListEditProposals` `GET /v2/catalog/proposals`。默认 `status=open`。响应 `{gid, items, users}`。`items` 是**未 enrich** 的 `[]EditProposal`（没有 `gid` / `galgame` 嵌套），与 Queue/Mine 的 `proposalItem` 不同形。前端仍用 `GalgameEditProposalList` 去接（`edit/Container.vue:58`）。

**任何人可匿名读一部作品的待审提案全文（patch）。** 不查 `published`、不查 claim `hidden`、不查 NSFW。`workIDOf` 对 hidden 作品只要 gid 映射还在就会成功。golden:108 中间件链停在 `ContentStance`，没有 Auth。

**调用方**：`edit/Container.vue:58`。**测试**：`TestEditGameProposals`（匿名 200）。

### 3.4 `GET /api/galgame/:gid/edit/revisions`（optional）

应用 key `ListEditRevisions`。`limit` 经 `queryInt`：非法/缺省 → 0 → `collectV2List` 当成 20（`v2_app.go:57–59`）。前端传 `limit:200`（`history/Container.vue:15`）；`collectV2List` 单页最多 100、最多走 20 页。

响应 `{gid, items, users, can_revert}`。`can_revert`：没 token → false；有 token 则拉 user-plane schema，任一可编辑字段 `CanReview==false` 则 false，零个可编辑字段也 false（`edit_handler.go:474–495`）。schema 失败 slog 后 **false**（失败关闭，不是失败开放）。

`items` 是 catalog `EditRevision`：`id`/`seq`/`action`（字符串，如 `merged`）/`changed_fields`/`snapshot`/`actor_uid`/`amender_uid`/`proposal_id`/`site`/`created_at`/legacy_*。v2 列表解码 `v2Revision.revision()`（`v2_user.go:239–255`）**不填 `snapshot` / legacy_***。历史页用 `seq` 和 `legacy_id??id` 对动态卡片（`GalgameEdit.vue:27`）。

**调用方**：`history/Container.vue:14`（页面 `pages/galgame/[gid]/history.vue` **无 auth middleware**，匿名可开）；动态 `activity/card/GalgameEdit.vue:25`（`limit=200` 再按 id 找 seq）。

**可见性**：与 §3.3 相同，hidden 作品的修订历史公开可读。optional：cookie 失效当匿名；Bearer 无效 401。

**测试**：`TestEditRevisionsCanRevertFromProjection`。

### 3.5 `GET /api/galgame/:gid/edit/diff`（公开）

query `from`/`to` 版本号，`<1` → `400/233 "需要 from/to 版本号"`。应用 key 把 seq 扫成 revision id 再 `GET /v2/catalog/revisions/{toId}?include=diff&diff_base={fromId}`。找不到 seq → `editError` → 404。

响应 `EditDiff` `{from_seq, to_seq, fields:[{key,kind,diff_hint,from,to}]}`。

**调用方**：`history/Container.vue:26`；`GalgameEdit.vue:31`（`from=seq-1&to=seq`）。公开，无 SFW/hidden 闸。

### 3.6 `POST /api/galgame/:gid/edit/revert`

body `{to_seq, note}`。`to_seq<1` → 400；note>2000 → 校验。本地**没有** owner/moderator 闸（wave 178 删了，`permissions.md:30`）。`RevisionIDBySeq` 找不到 → `400/233 "目标版本不存在"`（注意：其它 not found 是 404）。然后 `RevertEditEntityUser` `POST /v2/moderation/reverts` `{revision_id, reason: note}`。

响应 `EditRevertResult` `{proposal, revision}`。前端 `history/Container.vue:50` 不传 `note`。

**测试**：`TestEditRevertRidesTheUserToken`、`TestEditRevertDeniedByInfra`。

### 3.7 `GET /api/galgame-edit/mine`

用户面 `GET /v2/me/proposals`。query `gid`（>0 才译成 `entity_id`；非法当 0，不加过滤）、`status`（不校验）、`limit`。`Mine:true` 选 `/v2/me/proposals` 而不是 moderation 面（`user_edit.go:249–252`）。

响应 `{items:[]proposalItem, users}`。`enrich` 把 catalog `entity_id` 译成 `gid` + `GalgameBrief`；翻译失败 slog，该项 `gid=0`、`galgame` 缺席。前端标题回落 `Galgame #${item.gid || item.entity_id}`（`mine/Container.vue:13`）——`entity_id` 是 catalog work id，会链到 `/galgame/${catalogWorkId}`（`:69`），撞车时打开另一部作品。

**调用方**：`galgame-edit/mine/Container.vue:7`（页面 `pages/galgame-edit/mine.vue`）；编辑页 `edit/Container.vue:51`（`query:{gid}`）。

**测试**：`TestEditMineAsksForItsOwn`、`TestEditReadsRideTheUserPlane`。

### 3.8 `GET /api/galgame-edit/queue`

**本地门是 `RequireModerator`（角色）**，不是 `galgame.claim.review`。然后用户面 `GET /v2/moderation/proposals`。infra 再判 `catalog.edit.review`（测试里的拒绝文案 `permission denied: catalog.edit.review`，`edit_reads_bearer_test.go:93`）。

默认 `status=open`。无 cursor、无 total、无 page。`limit` 缺省 0→20。

**调用方**：`galgame-edit/review/Container.vue:11`。页面 `auth` only；模板用 `useRole().canModerate` 做 UI 闸（`:4` / `:34`），与本地门同为角色、与投稿审核键不同。

Bearer 403（`staffCandidate`）。被覆盖系统撤销了 `galgame.claim.review` 但仍是 moderator 的人：投稿队列 403，本队列 200（直到 infra 拒绝）。反过来，只有 perm override、没有 moderator 角色的人：投稿队列 200，本队列 403。

**测试**：`TestEditViewGates`（普通人 403 且不打 catalog）、`TestEditQueueAsksForEverybodys`、`TestEditQueueRelaysTheInfraDenial`。

### 3.9 `GET /api/galgame-edit/proposals/:id`

`reviewEntry`：先 `proposalForReview`（租户/族不对 → 伪装 404 `proposal outside the kungal tenant`，`edit_handler.go:666–668`），再要求 `user.CanModerate() || isGameOwner`（owner 来自本地 `galgame.creator_user_id`，`:682`）。过门之后拉 snapshot + schema，算 `can_decide`（effective_patch 优先，patch 里每个键都要 `CanReview`，`:688–706`）。

`can_decide` 只出现在这条 GET 上，Merge/Decline **不再读它**。

**调用方**：`review/Detail.vue:17`。页面 `pages/galgame-edit/review/[id].vue`（`auth`）。游戏所有者能打开工作台（`TestEditOwnerReview`）。

### 3.10 `POST …/amend` `POST …/merge` `POST …/decline` `POST …/withdraw`

| 路由 | 本地闸 | 上游 | 副作用 |
|---|---|---|---|
| amend | 仅 Auth（无 owner/moderator） | `POST /v2/me/proposals/{id}/amendments` | 无 |
| merge | 仅 `proposalForReview` 租户钉 | `POST /v2/moderation/proposals/{id}/decisions {decision:merge}` | 给提案人 +1 萌萌点（合并者≠提案人）；`Touch` `resource_update_time`；`NotifyMerged` |
| decline | 同 merge | `{decision:decline}` | `NotifyDeclined`（内容带理由） |
| withdraw | 仅 Auth | `PATCH /v2/me/proposals/{id} {state:withdrawn}` | 无 |

amend body `{set, unset, note}`，两者都空 → 校验错误。decline 的 `note` trim 空 → `400/233 "请填写拒绝理由"`。merge 允许空 note。

萌萌点 `Ref("galgame_pr", int(prop.EntityID))`（`edit_handler.go:845`）——`EntityID` 是 **catalog work id**，不是论坛 gid。键 `galgame_edit_merged:<proposalId>` 用提案 id，还好。`Award` 是 fire-and-forget。

**调用方**：amend/merge/decline → `review/Detail.vue:226/242/265`（先 amend 再 merge）。withdraw → `edit/Container.vue:158`、`mine/Container.vue:25`。

**测试**：`TestEditNoWriteAssertsAnActor`、`TestEditOwnerReview`、`TestEditDeclineNotification`、`TestEditTenantPin`、`TestEditUserPlaneWithoutTokenNeverCalls`、`edit_user_plane_test.go` 一组。

---

## 4. 编辑引擎状态机（提案）

论坛强制的状态词：`open` / `merged` / `declined` / `withdrawn`。转移由 catalog 执行，论坛只做字段校验：

```
（空 patch 拒绝） --Submit--> open | merged（automerge）
open --Amend--> open（effective_patch 变）
open --Merge--> merged  + 一条 edit_revision
open --Decline--> declined
open --Withdraw--> withdrawn
（任意 seq） --Revert--> 新提案或新修订（上游形状；论坛回 EditRevertResult）
```

未知 status 在 Queue/GameProposals 是 400；在 Mine 是透传。`v2Proposal.proposal()` 把空 status **静默填成 `open`**（`v2_user.go:168–170`）。

修订 `action` 字符串来自上游；feed 侧另有 int16：`0 created / 1 merged / 2 direct / 3 reverted`（`revisions.go:26–29`）。时间线 cron 丢掉 `created`（`isTimelineEdit`，`galgame_edit_revision_sync.go:123–125`）。

---

## 5. 审核授权：两套键，四处镜像

### 5.1 纯论坛键 `galgame.claim.review`

`perm.GalgameClaimReview = "galgame.claim.review"`（`perm.go:42`），进 moderator bundle（`:114`）。用在：

- `GET /galgame/audited`、`GET /admin/galgame/submissions`、`POST /admin/galgame/:gid/review`（`router.go:152/385/388`）
- 全部经 `user.Can` → Bearer 恒 false

BE↔FE 镜像（任务书说的四处，代码里还能再数出页面 meta）：

| # | 位置 | 键 |
|---|---|---|
| 1 | `pkg/perm/perm.go:42` + bundle `:114` | `galgame.claim.review` |
| 2 | `apps/web/app/composables/useCan.ts:35` `MODERATOR_PERMISSIONS` | `'galgame.claim.review'` |
| 3 | `apps/web/app/constants/permission.ts:61` `KUN_PERMISSION_META` | `'galgame.claim.review'` |
| 4 | `pkg/perm/perm_test.go:17` 金表 `allPerms` | `GalgameClaimReview` |

另外还写在 `constants/admin.ts:84`（导航）、`pages/admin/submissions.vue:4`、`pages/edit/galgame/audited.vue:4`、`Wizard.vue:19` `useCan('galgame.claim.review')`。`useCan` 优先读 `/api/perm/mine` 的 override 列表（`useCan.ts:88–91`）。

### 5.2 编辑队列用角色，不用这个键

`GET /galgame-edit/queue` 是 `RequireModerator` → `user.CanModerate()` → `role.CanModerate`（`role.go:11–21`、`auth.go:65–67`）。前端 `useRole().canModerate`（`useRole.ts:13–17`）看会话里的 `moderator|admin|ren`。infra 镜像键是 `galgame.review` / 实测文案 `catalog.edit.review`（`permissions.md:22`，`edit_reads_bearer_test.go:93`）。

### 5.3 七项 INFRA-PROXY 展示表（另一套名字）

`KUN_PROXY_PERMISSIONS`（`permission.ts:150–186`）用 `galgame.review_submission`、`galgame.review`、`galgame.create`…——**不是** `galgame.claim.review`。`permissions.md:34` 说权威在 `pkg/perm` 包注释，**当前 `perm.go` 没有包注释**，三处同步已经断了一处。文档表第 2 行还写着 submissions 走 `RequireModerator`，代码是 `RequirePermission(perm.GalgameClaimReview)`。

wave 178：amend/merge/decline/revert 的本地裁决门已删，真值在 infra（`edit.catalog.work.review` + token 推导的 owner）。`can_review` / `can_decide` / `can_revert` 是投影，前端不应再镜像一份。

---

## 6. 调用方总表

搜过的范围：`apps/web/app/**`、`apps/web/server/**`、`apps/web/shared/**`，字面量 `/galgame/submit`、`/galgame/mine`、`/galgame/audited`、`/galgame/search/wizard`、`/galgame-edit`、`/admin/galgame/submissions`、`/edit/bootstrap`、`/edit/proposals`、`/edit/revisions`、`/edit/diff`、`/edit/revert`、`/resubmit`、`/draft`。Flutter：`/home/kun/Desktop/code/website/kungal-apps` 下 `packages/kungal_api/lib/kungal_api.dart`（只有 `library;`）、`apps/kungal/lib/features/galgame/presentation/galgame_screen.dart`（占位 `Text('galgame')`）、`lib/app/router.dart`、`lib/core/network/dio.dart`（无路径）、`docs/03-contracts-and-sdk.md`（前瞻，不列这些旧路径）。`docs/proj/app-direct-api.md` 放行表没有本段 22 条。

| 路由 | 网页调用方 | Nitro | Flutter |
|---|---|---|---|
| POST `/galgame/submit` | `edit/galgame/Footer.vue:82` | 无 | 无 |
| POST `/galgame/:gid/resubmit` | `edit/galgame/Mine.vue:45` | 无 | 无 |
| DELETE `/galgame/:gid` | `Mine.vue:17` | 无 | 无 |
| DELETE `/galgame/:gid/draft` | `Mine.vue:67` | 无 | 无 |
| GET `/galgame/mine` | `Mine.vue:3` → `useGalgameClaimList.ts:9` | 无 | 无 |
| GET `/galgame/audited` | `edit/galgame/Audited.vue:3` | 无 | 无 |
| GET `/galgame/search/wizard` | `edit/galgame/Wizard.vue:35` | 无 | 无 |
| GET `/admin/galgame/submissions` | `pages/admin/submissions.vue:28,50` | 无 | 无 |
| POST `/admin/galgame/:gid/review` | `submissions.vue:128` | 无 | 无 |
| GET `…/edit/bootstrap` | `galgame/edit/Container.vue:17` | 无 | 无 |
| POST `…/edit/proposals` | `edit/Container.vue:117` | 无 | 无 |
| GET `…/edit/proposals` | `edit/Container.vue:58` | 无 | 无 |
| GET `…/edit/revisions` | `galgame/history/Container.vue:14`；`activity/card/GalgameEdit.vue:25` | 无 | 无 |
| GET `…/edit/diff` | `history/Container.vue:26`；`GalgameEdit.vue:31` | 无 | 无 |
| POST `…/edit/revert` | `history/Container.vue:50` | 无 | 无 |
| GET `/galgame-edit/mine` | `galgame-edit/mine/Container.vue:7`；`edit/Container.vue:51` | 无 | 无 |
| GET `/galgame-edit/queue` | `galgame-edit/review/Container.vue:11` | 无 | 无 |
| GET `/galgame-edit/proposals/:id` | `galgame-edit/review/Detail.vue:17` | 无 | 无 |
| POST `…/amend` | `Detail.vue:226` | 无 | 无 |
| POST `…/merge` | `Detail.vue:242` | 无 | 无 |
| POST `…/decline` | `Detail.vue:265` | 无 | 无 |
| POST `…/withdraw` | `edit/Container.vue:158`；`mine/Container.vue:25` | 无 | 无 |

`apps/web/server/utils/kunOgCard.ts:104` 打的是 `GET /galgame/${id}` 详情，不是本段。

---

## 7. 命名问题清单（迁 v1 时逐条执行）

| 现状 | 问题 | v1 |
|---|---|---|
| 路径 `:gid`、响应 `gid` | 禁用名 | `{galgame_id}` / `galgame_id` |
| `work_id` | 没说是 catalog 的；与 gid 撞车 | `catalog_work_id` |
| `product_work_id` | 论坛 gid 的另一个名字 | 删，只留 `galgame_id` |
| `created` / `updated`（本地行、部分上游） | 禁用名 | `created_at` / `updated_at` |
| `status` int（`GalgameBrief.status` 0/2） | 裸整数；与 Problem.status 撞型 | `state` 字符串 |
| `claim_state` / 提案 `status` | 生命周期应叫 `state` | `state` |
| `user` 没出现，但 `users` map 以 uid 为键、值为 `{id,name,avatar}` | 应用 key 读面下发的查看者无关数据，可留；键名用 `UserRef` | `users: { [id]: UserRef }` |
| `can_review` / `can_decide` / `can_revert` | 随查看者变，应进 `viewer` | `viewer.can_review` 等；匿名 `viewer: null` |
| `is_liked` 本段没有 | — | — |
| `next_before` / `next_cursor` | 第三、第四种分页；cursor 无 `cur_` | 声明一种；G16 |
| `from` / `to`（diff 版本号） | 不是 id、不是时间 | `from_seq` / `to_seq`（响应已经是这个，query 应对齐） |
| `to_seq` | 可以 | 保留 |
| POST `…/resubmit` `…/review` `…/amend` `…/merge` `…/decline` `…/withdraw` `…/revert` | 动词路径（B7） | 资源 + 决策子资源，见 §12 |
| DELETE `/galgame/:gid` | 语义是撤回 claim，不是删 galgame | 不要用 DELETE 主资源 |
| `banner_attached` | 布尔未 `is_`/`has_` 前缀 | `has_banner_attached` |
| `acted_count` | 计数 OK | 可留 |
| `content_limit` vs `display_nsfw` vs `age_limit` vs `content_rating` | 两轴四名 | 展示轴 `display_nsfw`；年龄轴 `content_rating` 字符串枚举 |

---

## 8. 疑似问题（平铺，不排序）

1. `v2Claim.item()`（`v2_user.go:271-284`）丢掉 `product_work_id`。前端 `galgameClaimGid`（`galgameClaimState.ts:45-46`）只读这一列 → 「我的提交 / 我的审核 / 向导 pending」上所有按 gid 行动的按钮不出现。
2. `MyClaims`（`user_claims.go:134-141`）不填 `next_before`。`useGalgameClaimList.ts:29-31` 的加载更多永不亮。
3. 同一函数还丢掉 `last_reason`：拒绝原因在 `Mine.vue:118` / `Wizard.vue:139` 读的是空。
4. `Submit` 校验了 `release_date` 然后丢掉（`submission_service.go:69-77`）。前端仍收集这个字段（`Footer.vue:36`）。
5. `age_limit` 非 `"r18"` 静默写成全年龄 `content_rating=0`（`submission_payload.go:97-101`）。
6. `original_language` 未知值原样透传（`olangOf` `:18-26`）。
7. 向导 `total` 是过滤前、`items` 是过滤后（`submission_service.go:350-358` + `wizardItems` `:390-392`）。
8. 向导与审核队列 `OpenPopulation` → `nsfw=true` 且不设 `content_limit`（`catalog_face.go:50-52`，`claim_review_service.go:58`，`submission_service.go:377`）。SFW 读者看到 NSFW 标题。本段 handler 零次 `utils.IsSFW`。
9. `contentLimitOf` 在 claim 缺 `content_limit` 时用 `content_rating` 冒充展示轴（`catalog_wire.go:271-279`）。向导 brief 走这条。
10. `GET …/edit/proposals` 与 `GET …/edit/diff` 完全公开（golden:107-108），不查 hidden/banned/NSFW/`published`。`workIDOf` 不看 claim 状态。
11. `GET …/edit/revisions` optional，同样不查 hidden。
12. `EditHandler.userMap`（`edit_handler.go:73-93`）不调 `IsRenderable`；OAuth 错只 warn 回空 map。
13. `enrich` 的 gid 翻译失败只 warn（`:578-580`），响应里 `gid=0`，前端链到 `/galgame/0` 或回落到 catalog `entity_id`（`mine/Container.vue:13,69`）。
14. `gidOf` 失败 slog 回 0（`:176-185`）。`submitSideEffects` 因此整段跳过通知和时间线。
15. `submitSideEffects` 把 catalog 提案 id 写入 `galgame_activity.wiki_pr_id`（`edit_handler.go:438-441`）。067 专门把引擎 id 和 wiki id 拆开，这里又把提案 id 塞回 wiki 列。
16. `v2Proposal.proposal()` 不填 `decided_by_uid`（`v2_user.go:187-201`）。
17. Merge 的萌萌点 `Ref` 用 catalog `EntityID`（`edit_handler.go:845`）。
18. 通过投稿的 +3 分在 cron（`galgame_claim_event_sync.go:239`），不在 Review HTTP。cron 停了接口仍 200。
19. `SubmitLocal` 失败只 warn（`submission_service.go:90-92`）：catalog 有作品、本地没有 `creator_user_id`，提交者打不开自己的未发布页，owner 闸也认不出人。
20. `DeleteDraft` catalog 已删、本地 `DELETE` 失败 → 500，两边不一致（`submission_service.go:191-193`）。
21. `attachBanner` 是投稿成功后的第二次、第三次写（提案+合并），失败不回滚 mint。
22. 所有 POST 无 `Idempotency-Key`。合并双击可以双发通知；萌萌点键按 proposal id 倒是幂等。
23. `limit`/`before`/`page` 非法值 `atoiOr` / `queryInt` 静默变默认，不报 400。
24. `Mine` 的 `status=` 不校验；Queue/GameProposals 校验。同一词表两套纪律。
25. `ListMine` 接受 `claim_state=none|live|hidden`（`claimStates`），默认又不返回它们。
26. Review 的 `unban` 后端有、前端无。
27. Review 响应的 `to_state` 是本地写死的，不读上游（`user_claims.go:76-83`）。
28. `can_review`（bootstrap）是「schema 有可审字段」，不是「你能审」；与 `can_decide` / `RequireModerator` 三套含义。
29. 投稿审核门是 `galgame.claim.review`，编辑队列门是 `CanModerate` 角色。override 只动其中一个时两张后台表分叉（§5.2）。
30. `permissions.md:21` 仍写 submissions 走 `RequireModerator`；`router.go:385` 是 `RequirePermission(perm.GalgameClaimReview)`。
31. `permissions.md:34` 说 `pkg/perm` 包注释是 7-op 权威；`perm.go` 无包注释。`KUN_PROXY_PERMISSIONS` 的键与 `galgame.claim.review` 不是同一个字符串。
32. `NamePreference` 挂在全部 22 条上，向导/队列的 `name` 随 cookie 变。
33. `DELETE /galgame/:gid` 语义是撤回 claim。
34. `adoptAndPublish`（`submission_service.go:111`）本段 HTTP 不调用，资源通道会（`resource_service.go:393`）。任务书说不要复活认领；这条已经在发资源时静默认领+发布。
35. `CatalogGet` 当前走 `doV2`（`client.go:149-150`）。`catalog_v2_shape_test.go:9` 仍写「dormant behind catalogReadsV1」，仓库里已经没有 `catalogReadsV1` 这个符号。任务书记忆与代码不一致，以 `doV2` 为准。
36. 公开 `GameProposals`/`Diff`/`Revisions` 用应用 key，401/403 被译成 503「编辑服务暂不可用」（`editError` `:247-250`），把 key 配错说成服务宕机。
37. `queryInt` 对 `limit=200` 的 revisions 会让 `collectV2List` 连打最多 2 页；公开 diff 把 seq 扫成 id 可能连打最多 20 页（`revisionIDsBySeq` `:303`）。
38. 前端 Zod 要求至少一个 intro，后端不要求。
39. `Footer.vue:37` 发送 `release_date_tba`，后端无此字段。
40. `ClaimActionResult` 的 JSON 键是 `from_state`/`to_state`，与 Go 字段名 `From`/`To` 不同，调用方用 `unknown` 所以没爆。
41. 历史页文案仍写「鲲Galgame百科」（`GalgameEdit.vue:116`）。
42. `history.vue` 无 `auth` middleware，与 Revisions 的 optional 一致；Revert 按钮靠 `can_revert` 投影。
43. `edit/Container.vue:69` 用 `proposer_uid !== userStore.id` 滤掉自己的提案——这是前端镜像的权限，Bearer/会话 uid 来自 pinia，不是 `viewer`。
44. 合并者给自己的提案合并不发萌萌点（`prop.ProposerUID != mergerID`，`edit_handler.go:843`），这是故意的。
45. `PendingQueue` 用应用 key 列全站 pending，不按审核者；任何有 `galgame.claim.review` 的 cookie 会话都能看全部。
46. wizard `page` 参数存在但前端不传，永远第 1 页。
47. `submissionGID` 用 `Atoi`，`parseGid` 用 `ParseInt` 64 位。gid 溢出在投稿路径和编辑路径表现不同。
48. `ListAudit` 的网页按钮链到 `/galgame/${gid}`，非 live 走预览（`Audited.vue:61-67`）；`product_work_id` 为 null 时预览按钮也不出现。

---

## 9. 测试（按路由）

| 路由 | 测试 |
|---|---|
| Submit | `submission_submit_test.go`、`submission_payload_test.go` |
| Resubmit | `submission_bearer_test.go` 错误分类（借 Resubmit） |
| Withdraw | `TestOwnerActionSpeaksAsTheSubmitter` |
| DeleteDraft | `TestDeleteDraftStopsAtAnUpstreamRefusal`；`galgame_repo_test.go` 本地拒绝 |
| ListMine / ListAudit | `submission_bearer_test.go` 两条 |
| SearchWithPending | `wizard_search_test.go` |
| PendingQueue | **无** |
| Review | `TestReviewVerdictSpeaksAsTheModerator` |
| 编辑读写 | `edit_handler_test.go`、`edit_user_plane_test.go`、`edit_reads_bearer_test.go`（bootstrap/submit/queue/mine/detail/amend/merge/decline/withdraw/revert/proposals/tenant/can_decide/can_revert） |
| Diff | 无独立测试（走 unconfigured / 通用） |
| adoptAndPublish | `claim_unclaimed_test.go`（资源通道，不是本段 HTTP） |

无契约测试、无 DB 集成测试覆盖这 22 条 handler 的可见性。

---

## 10. 本段碰到的表

**写：**

| 表 | 列 | 谁写 |
|---|---|---|
| `galgame` | `id`（=论坛 gid）、`creator_user_id`（仅空时）、`published` 保持 false | `SubmitLocal` |
| `galgame` | 整行 DELETE，前提是没有 `galgame_resource` | `DeleteLocalDraft` |
| `galgame` | `resource_update_time` | Merge 成功 `Touch` |
| `galgame_activity` | `wiki_pr_id`, `galgame_id`, `user_id`, `type='GALGAME_PR_CREATION'` | `submitSideEffects` |
| `galgame_activity` | `edit_revision_id`, `wiki_revision_number`, `galgame_id`, `user_id`, `type='GALGAME_EDIT'` | cron `GalgameEditRevisionSync`（不是 HTTP） |

**读：** `galgame.creator_user_id`（owner 闸、通知对象）。claim_state / 提案 / 修订 **不在本库**。

`galgame.published` 是 078 的粘性 SEO 旗标，本段 HTTP **不改它**。hidden/ban 的 unpublish 在 cron `claimEffectUnpublish`（`galgame_claim_event_sync.go:170-171`）。不要在 v1 投稿面上动 `published`。

---

## 11. 需要生产库回答的问题

claim_state / 提案 status 在 catalog，本库没有。下面只问论坛库答得了的。

```sql
-- 1. 本地 galgame 行与 published（078 粘性旗标）
SELECT count(*) FROM galgame;
SELECT published, count(*) FROM galgame GROUP BY 1;
SELECT count(*) FROM galgame WHERE creator_user_id IS NULL;
SELECT count(*) FROM galgame WHERE content_limit IS NULL;
SELECT content_limit, count(*) FROM galgame GROUP BY 1;

-- 2. 投稿草稿删不掉的行（有资源的 draft：catalog 状态本库看不见，只能看「有资源」）
SELECT count(*) FROM galgame g
 WHERE EXISTS (SELECT 1 FROM galgame_resource r WHERE r.galgame_id = g.id)
   AND g.published = false;

-- 3. 时间线三套 id 命名空间
SELECT type, count(*) FROM galgame_activity GROUP BY 1;
SELECT
  count(*) FILTER (WHERE wiki_revision_id IS NOT NULL) AS wiki_rev_id,
  count(*) FILTER (WHERE wiki_pr_id IS NOT NULL) AS wiki_pr_id,
  count(*) FILTER (WHERE edit_revision_id IS NOT NULL) AS edit_rev_id,
  count(*) FILTER (WHERE wiki_revision_number IS NOT NULL) AS has_seq
FROM galgame_activity;

-- 4. GALGAME_PR_CREATION 是否在把 catalog 提案 id 写入 wiki_pr_id
SELECT count(*) FROM galgame_activity WHERE type = 'GALGAME_PR_CREATION';
SELECT min(wiki_pr_id), max(wiki_pr_id) FROM galgame_activity WHERE type = 'GALGAME_PR_CREATION';

-- 5. 同一整数是否同时出现在 wiki_revision_id 与 edit_revision_id（067 要防的撞车）
SELECT count(*) FROM galgame_activity a
 JOIN galgame_activity b ON a.wiki_revision_id = b.edit_revision_id
 WHERE a.wiki_revision_id IS NOT NULL AND b.edit_revision_id IS NOT NULL;

-- 6. 动态卡片还在靠 legacy_id 对 seq 的规模（wiki_revision_number IS NULL）
SELECT count(*) FROM galgame_activity
 WHERE type = 'GALGAME_EDIT' AND wiki_revision_number IS NULL;

-- 7. 本段会 Touch 的 resource_update_time 新鲜度
SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY extract(epoch from (now() - resource_update_time)))
  FROM galgame WHERE resource_update_time IS NOT NULL;
```

catalog 侧请另跑（本会话没有 catalog 库）：各 `claim_state` 行数、`pending` 队列长度、提案 `state` 分布、`kungal` site 的 open 提案数、同名 mint 被 `DUPLICATE_SUSPECTS` 拦住的次数。没有这些数，v1 不要为 `unban` / `claim_state=none` 建模。

---

## 12. 提议的 v1 资源地图（提案，不是契约）

| 旧 | 提议 v1 | 一行理由 |
|---|---|---|
| `POST /api/galgame/submit` | `POST /api/v1/galgame-submissions` | 名词化投稿集合；响应带 `galgame_id` + `catalog_work_id` 两个字符串 id |
| `GET /api/galgame/mine` | `GET /api/v1/me/galgame-submissions` | 「我的 xxx」游标集合；`state` 封闭枚举；必须返回论坛 `galgame_id` |
| `GET /api/galgame/audited` | `GET /api/v1/me/galgame-review-decisions` | 审核者视角的另一集合，不要和提交者列表共用 `kind=` |
| `GET /api/galgame/search/wizard` | `GET /api/v1/galgames?purpose=publish` 或独立 `GET /api/v1/galgame-publish-search` | 向导有自己的合格谓词，不能复用浏览引擎；`include_nsfw` 显式参数 |
| `POST /api/galgame/:gid/resubmit` | `POST /api/v1/galgame-submissions/{galgame_id}/decisions` `{decision:submit}` | 状态机决策，不要动词路径 |
| `DELETE /api/galgame/:gid` | 同上 `{decision:withdraw}` | DELETE 主资源会让人以为作品没了 |
| `DELETE /api/galgame/:gid/draft` | `DELETE /api/v1/galgame-submissions/{galgame_id}` | 只有 draft 可删；保持 catalog 先、本地后 |
| `GET /api/admin/galgame/submissions` | `GET /api/v1/galgame-submissions?state=pending` | 管理浏览页码/游标二选一；staff 仅 cookie |
| `POST /api/admin/galgame/:gid/review` | `POST /api/v1/galgame-submissions/{galgame_id}/decisions` `{decision, note}` | 与投稿者决策同一子资源，鉴权不同 |
| `GET …/edit/bootstrap` | `GET /api/v1/galgames/{galgame_id}/edit-form` | 静态形状：values + fields + vocabularies；`viewer.can_review` |
| `POST …/edit/proposals` | `POST /api/v1/galgames/{galgame_id}/edit-proposals` | 创建子资源；automerge 仍是同一响应的 `state` |
| `GET …/edit/proposals` | `GET /api/v1/galgames/{galgame_id}/edit-proposals` | 从 public 收到 `optional`；hidden 作品 404 |
| `GET …/edit/revisions` | `GET /api/v1/galgames/{galgame_id}/edit-revisions` | 游标；`viewer.can_revert` |
| `GET …/edit/diff` | `GET /api/v1/galgames/{galgame_id}/edit-revisions/diff?from_seq&to_seq` | 只读；query 名与字段对齐 |
| `POST …/edit/revert` | `POST /api/v1/galgames/{galgame_id}/edit-reverts` | 名词化；鉴权仍只在 infra |
| `GET /galgame-edit/mine` | `GET /api/v1/me/edit-proposals` | 与投稿 mine 分开，两种对象 |
| `GET /galgame-edit/queue` | `GET /api/v1/edit-proposals` | staff 浏览；本地门与投稿审核键对齐或明确分成两个能力 |
| `GET /galgame-edit/proposals/:id` | `GET /api/v1/edit-proposals/{proposal_id}` | 工作台；`viewer.can_decide` |
| `POST …/amend` | `POST /api/v1/edit-proposals/{proposal_id}/amendments` | 已经是子资源，只改前缀 |
| `POST …/merge` `…/decline` | `POST /api/v1/edit-proposals/{proposal_id}/decisions` `{decision:merge\|decline, note}` | 一个决策槽 |
| `POST …/withdraw` | `POST /api/v1/edit-proposals/{proposal_id}/decisions` `{decision:withdraw}` 或 `PATCH` `state=withdrawn` | 与 catalog v2 对齐也可以 PATCH |

不要把已删除的用户「认领游戏」面加回来。`adoptAndPublish` 若还要存在，属于资源发布轨，不属于本段。
