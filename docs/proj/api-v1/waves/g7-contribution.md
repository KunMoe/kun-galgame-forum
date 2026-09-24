# G7 · 投稿 / 认领审核队列 / 资料编辑引擎

> G 轨第七段（G0 改号之后），2026-09-24。投稿、认领审核、资料编辑引擎：**22** 条旧路由，全部是论坛作 BFF、调用方自己的 OAuth token 打 catalog 认领面与编辑引擎。G 号段 140–159 与 195 已用；本轨无迁移。
> 契约与变异题同一个提交，早于实现。普查全文在 [census/galgame-contribution.md](census/galgame-contribution.md)。普查写于 G0 之前：作品 id 现已是 catalog work id，路径与字段一律 `work_id` / `{work_id}`（根 CLAUDE.md 铁律 3）；普查 §0 身份、`workIDOf` / `gidOf` / `product_work_id` / 10,289 撞车、§7 命名表里的 `galgame_id` / `catalog_work_id` 全部作废。本契约以代码为准，不沿用普查行号。
> 状态迁移的 v1 先例是 `PATCH /api/v1/admin/review-items/{review_item_id}` 的 `state` 字段：一次迁移是对资源 `state` 的 `PATCH`，不是动词路径，也不是 `DELETE` 作品。

## 1. 普查（生产实测 2026-09-24，catalog 库与论坛库；数字按任务书原文）

### 1.1 数字

| 事实 | 数字 |
|---|---|
| catalog 作品按 claim | 未认领（null）205,929；`draft` 52,475（其中 site `kungal` 52,459）；`live` 12,418（`kungal` 12,358）；`hidden` 32；`declined` 31；`pending` **0** |
| claim 事件（site `kungal`） | → draft 53,924；→ live 12,438；→ pending 74（曾经有过）；→ declined 35；→ hidden 37。586 个不同 actor；最近一条 2026-09-23 19:59Z |
| 谁写下了草稿 | actor 0（系统回填）53,539 条 draft 事件；最多的人类 actor 59 |
| 每 actor 的 claim 数 | 最大 1,283 部不同作品，p99 414 |
| claim reason 长度 | 最大 76 个字符 |
| 编辑提案（`catalog.work`） | `kungal`：merged 776，declined 19，withdrawn 17，open 2；`galgame_wiki`（导入历史）：merged 530 |
| 提案 note / decision note 长度 | 最大 385 / 174 个字符 |
| 修正（amendment） | 25 |
| 编辑修订（`catalog.work`） | 13,047 行，覆盖 11,054 部作品；每部最大 15，p99 4；最大 `seq` 15；action：created 9,686，direct 1,849，merged 1,501，reverted 11 |
| 论坛 `galgame` 行 | 16,238；published 9,821；`creator_user_id IS NULL` 2,743 |
| 论坛 `galgame_activity` | `GALGAME_EDIT` 775，`GALGAME_PR_CREATION` 472；`wiki_pr_id` 有值 472（范围 79–1446：cutover 之后就是 catalog 提案 id），`edit_revision_id` 463，`wiki_revision_id` 312 |

`pending` 队列此刻为空，但事件里曾经有过 74 次进入 pending，审核面与 `state=pending` 过滤仍要建模。`unban` 前端没有按钮，后端与 infra 都有；隐藏作品 32 部，是这条路径的现存数据。

### 1.2 调用方

搜过 `apps/web/app/**`、`apps/web/server/**`、`apps/web/shared/**`。Nitro 对本段 22 条**零命中**。Flutter App：普查 Dart 源零命中；`docs/proj/app-direct-api.md` 未列这 22 条为现网调用。本 worktree 不含 `kungal-apps`，本节按普查 + 文档再核。

| 旧路由 | 网页 |
|---|---|
| `POST /galgame/submit` | `edit/galgame/Footer.vue:82`（`confirm_duplicates` 先 false，体码 236 再确认）；页面 `pages/edit/galgame/create.vue` |
| `POST /galgame/:id/resubmit` | `edit/galgame/Mine.vue:45`，仅 `declined` |
| `DELETE /galgame/:id`（撤回） | `Mine.vue:17` |
| `DELETE /galgame/:id/draft` | `Mine.vue:67`，仅 `draft` |
| `GET /galgame/mine` | `Mine.vue:3` → `useGalgameClaimList.ts:9`（`before:0, limit:20`）；页面 `pages/edit/galgame/mine.vue` |
| `GET /galgame/audited` | `edit/galgame/Audited.vue:3`；页面 `pages/edit/galgame/audited.vue`（`permissions: ['galgame.claim.review']`） |
| `GET /galgame/search/wizard` | `edit/galgame/Wizard.vue:35`（`q` + `limit:12`）；页面 `pages/edit/galgame/publish.vue` |
| `GET /admin/galgame/submissions` | `pages/admin/submissions.vue:28,50` |
| `POST /admin/galgame/:id/review` | `submissions.vue:128`（`approve` / `decline` / `ban`；无 `unban` 按钮） |
| `GET …/edit/bootstrap` | `galgame/edit/Container.vue:17`；页面 `pages/galgame/[id]/edit.vue`（`auth`） |
| `POST …/edit/proposals` | `edit/Container.vue:117` |
| `GET …/edit/proposals` | `edit/Container.vue:58` |
| `GET …/edit/revisions` | `galgame/history/Container.vue:14`；`activity/card/GalgameEdit.vue:25`（`limit=200`）；页面 `pages/galgame/[id]/history.vue` 无 auth |
| `GET …/edit/diff` | `history/Container.vue:26`；`GalgameEdit.vue:31`（`from=seq-1&to=seq`） |
| `POST …/edit/revert` | `history/Container.vue:50`（不传 `note`） |
| `GET /galgame-edit/mine` | `galgame-edit/mine/Container.vue:7`；`edit/Container.vue:51`（`query:{gid}`） |
| `GET /galgame-edit/queue` | `galgame-edit/review/Container.vue:11`；模板 `useRole().canModerate` |
| `GET /galgame-edit/proposals/:id` | `galgame-edit/review/Detail.vue:17` |
| `POST …/amend` | `Detail.vue:226` |
| `POST …/merge` | `Detail.vue:242` |
| `POST …/decline` | `Detail.vue:265` |
| `POST …/withdraw` | `edit/Container.vue:158`；`mine/Container.vue:25` |

校验副本：`apps/web/app/validations/galgame.ts` 的 `submitGalgameSchema`（至少一个标题、至少一个简介、别名条数）。OAuth 现申请 `catalog:edit`（投稿与编辑共用）；缺 scope 旧体码 235，v1 的 `SCOPE_REQUIRED` 译文已在。

G0 之后网页「我的提交」已经用 `item.work_id` 做预览 / 编辑 / 撤回（`Mine.vue:126`），不再走普查里的 `galgameClaimGid`。拒绝原因仍读 `last_reason`，解码层仍丢掉它。

### 1.3 旧面的问题（普查编号 E*，对应 census 发现号）

G0 之后作废、不再建模的：发现 1 的 `product_work_id` 按钮链（网页已改读 `work_id`）、发现 13 / 14 的 `gid` 翻译失败、发现 17 的「EntityID 是 catalog 不是论坛 gid」、发现 33 / 47 的路径 `:gid` 溢出、§0 的 10,289 撞车、§7 命名表里「`work_id` 改成 `catalog_work_id`」。翻译助手 `workIDOf` / `gidOf` 已从代码里消失。

| 编号 | 事实 |
|---|---|
| E1 | `v2Claim.item()`（`v2_user.go:271-284`）仍不填 `last_reason` / `last_event_at` / `first_acted_at`。拒绝原因在 `Mine.vue:118` 读空（普查 #3）。`product_work_id` 仍不解码，但 v1 不再发这一列 |
| E2 | `MyClaims`（`user_claims.go:134-141`）不把 v2 `next_cursor` 译成 `next_before`，恒 0。每 actor 最多 1,283 条，20 条之后不可达（#2） |
| E3 | `Submit` 校验了 `release_date` 然后丢掉（`submission_service.go:69-77`）。**catalog mint 现在收 `released`**（infra `me_write_routes.go:34`、`me_claims_mint.go:70`）。v1 必须送达 |
| E4 | `age_limit` 非 `"r18"` 静默写成 `content_rating=0`（`submission_payload.go:97-101`）（#5） |
| E5 | `original_language` 未知值原样透传（`olangOf`）（#6） |
| E6 | 向导 `total` 是过滤前、`items` 是过滤后（#7） |
| E7 | 向导 `OpenPopulation` → `nsfw=true` 且不设 `content_limit`，SFW 读者看到 NSFW 标题（#8） |
| E8 | `contentLimitOf` 用年龄轴冒充展示轴（#9） |
| E9 | `GET …/edit/proposals` 与 `GET …/edit/diff` 完全公开，hidden 作品可读（#10）；修订 optional，同样不查 hidden（#11） |
| E10 | `EditHandler.userMap` 不调 `IsRenderable`；OAuth 失败 warn 后空 map（#12） |
| E11 | `submitSideEffects` 把 catalog 提案 id 写入 `galgame_activity.wiki_pr_id`（`edit_handler.go:398-401`）。生产 472 行全是 cutover 之后的提案 id（#15）。**继续写这一列，不迁移** |
| E12 | `v2Proposal.proposal()` 不填 `decided_by_uid`（#16） |
| E13 | 通过投稿的 +3 萌萌点在 claim-event cron，不在 Review HTTP（#18）。本轨不把加分搬进 HTTP |
| E14 | `SubmitLocal` 失败只 warn（#19）：catalog 有作品、本地没有 `creator_user_id` |
| E15 | `DeleteLocalDraft` 对带资源的行 `Exec` 0 行且 **返回 nil**（`galgame_repo.go:189-192`，测试钉住行仍在）。真正的 SQL 错误才 500（#20 的口径比代码窄） |
| E16 | `attachBanner` 是 mint 之后的提案+合并，失败不回滚（#21） |
| E17 | 全部 POST 无 `Idempotency-Key`（#22） |
| E18 | `limit` / `before` / `page` 非法值静默变默认（#23） |
| E19 | Review 的 `to_state` 本地写死，不读 catalog（`user_claims.go:76-83`）（#27）。unban 被写死成 `live` |
| E20 | bootstrap `can_review` 是「schema 有可审字段」，不是「你能审」（#28） |
| E21 | 投稿队列门是 `galgame.claim.review`，编辑队列门是 `CanModerate` 角色（#29）。**写进开放问题，本轨不统一** |
| E22 | `NamePreference` 挂在全部 22 条上（#32）。v1 不读这颗 cookie |
| E23 | `edit/Container.vue:58` 用 `proposer_uid !== userStore.id` 滤掉自己的提案（#43） |
| E24 | 合并者给自己的提案合并不发萌萌点（`edit_handler.go:778`）（#44）。保持 |
| E25 | `PendingQueue` 用应用 key 打作品搜索，不按审核者 token（#45）。infra 现有 `GET /v2/moderation/claims` |
| E26 | 上游 409 非 `DUPLICATE_SUSPECTS` 把英文诊断句写进用户信封（普查 §0.1） |
| E27 | 四语槽 `name_en_us` … 写进 mint；catalog 要的是 `catalog.work.titles` / `intros` 的 `{lang, title, kind}` / `{lang, intro}` 列表 |

## 2. 范围与寻址

路径段就是目标。`{work_id}` / `{proposal_id}` 与响应字段对齐（K1）。作品集合仍是 `/api/v1/works`。投稿集合是 **`/api/v1/work-submissions`**，对象 **`work_submission`**，`id` 即 catalog work id。编辑提案集合是 **`/api/v1/edit-proposals`**，对象 **`edit_proposal`**。修订挂在作品下。没有 `gid`、没有 `galgame_id`、没有 `product_work_id`、没有 `catalog_work_id`。

G17：写路径里每个 `{x_id}`，以它结尾的那段必须有 GET，且 200 带 `id`。`POST /work-submissions` 的 `Location` 是 `GET /work-submissions/{work_id}`。`POST /works/{work_id}/edit-proposals` 的 `Location` 是 `GET /edit-proposals/{proposal_id}`（不是嵌在作品下的第三段 id）。`POST /edit-proposals/{proposal_id}/amendments` 不把 `{amendment_id}` 放进路径。

| # | 旧 | v1 | 档 |
|---|---|---|---|
| 1 | `POST /galgame/submit` | `POST /api/v1/work-submissions` → **201** + `Location` + `WorkSubmission`；`Idempotency-Key` **必带**（K12） | required |
| 2 | — | `GET /api/v1/work-submissions/{work_id}`（G17） | required |
| 3 | `POST /:id/resubmit`、`DELETE /:id`、`POST /admin/…/review` | `PATCH /api/v1/work-submissions/{work_id}` `{state, note?}` | required |
| 4 | `DELETE /:id/draft` | `DELETE /api/v1/work-submissions/{work_id}` → **204** | required |
| 5 | `GET /galgame/mine` | `GET /api/v1/me/work-submissions` 游标 | required |
| 6 | `GET /galgame/audited` | `GET /api/v1/me/work-submission-reviews` 游标 | required |
| 7 | `GET /admin/galgame/submissions` | `GET /api/v1/work-submissions?state=pending` 游标 | required |
| 8 | `GET /galgame/search/wizard` | `GET /api/v1/work-submission-candidates?q=` 游标，**无 `total`** | required |
| 9 | `GET …/edit/bootstrap` | `GET /api/v1/works/{work_id}/edit-form` | required |
| 10 | `POST …/edit/proposals` | `POST /api/v1/works/{work_id}/edit-proposals` → **201** + `Location`；`Idempotency-Key` 必带 | required |
| 11 | `GET …/edit/proposals` | `GET /api/v1/works/{work_id}/edit-proposals` 游标 | optional |
| 12 | `GET …/edit/revisions` | `GET /api/v1/works/{work_id}/edit-revisions` 页码 | optional |
| 13 | `GET …/edit/diff` | `GET /api/v1/works/{work_id}/edit-revisions/diff?from_seq=&to_seq=` | optional |
| 14 | `POST …/edit/revert` | `POST /api/v1/works/{work_id}/edit-reverts` → **201**；`Idempotency-Key` 必带 | required |
| 15 | `GET /galgame-edit/mine` | `GET /api/v1/me/edit-proposals` 游标 | required |
| 16 | `GET /galgame-edit/queue` | `GET /api/v1/edit-proposals` 游标 | required |
| 17 | `GET /galgame-edit/proposals/:id` | `GET /api/v1/edit-proposals/{proposal_id}` | required |
| 18 | `POST …/amend` | `POST /api/v1/edit-proposals/{proposal_id}/amendments` → **201**；`Idempotency-Key` 必带 | required |
| 19 | `POST …/merge` `…/decline` `…/withdraw` | `PATCH /api/v1/edit-proposals/{proposal_id}` `{state, note?}` | required |

22 条旧路由全删，`legacy_route_baseline` 下调 **22**。v1 共 **19** 个操作（撤回 / 重提 / 审核并进一条 PATCH；合并 / 拒绝 / 撤回提案并进一条 PATCH；单条 claim 的 GET 是 G17 新加的）。

代码位置：`internal/galgame/apiv1`（与 `getWork` 同包）。`listMyWorkSubmissions` / `listMyWorkSubmissionReviews` / `listMyEditProposals` 的 OpenAPI tag 是 `me`；投稿集合 `work-submissions`；编辑提案 `edit-proposals`；作品下的表单 / 提案 / 修订 / 回滚 `works`。测试 `internal/app/v1_contribution_*_test.go`。

`adoptAndPublish` 属于资源发布轨（G3），本轨 HTTP 不暴露。

## 3. 形状

### 3.1 命名

| 旧 | v1 | 为什么 |
|---|---|---|
| 路径 `:gid` / `:id`，字段 `gid` | `{work_id}` / `{proposal_id}`；投稿资源自己的 `id` 就是 work id | K1；G0 |
| `galgame_id` / `product_work_id` / `catalog_work_id` | 不发 | 铁律 3 |
| `claim_state` / 提案 `status` | `state` | 01 命名表；`state` 在 G8 白名单，词表可以按对象不同 |
| `from` / `to`（diff 查询） | `from_seq` / `to_seq` | 与响应字段对齐（普查 §7） |
| `banner_attached` | `has_banner_attached` | F1 |
| `confirm_duplicates` | `is_duplicate_confirmed` | F1 |
| `content_limit` / `display_nsfw` | `is_nsfw` | 与 `Work.is_nsfw` 同名同型（展示轴） |
| `age_limit` `all`/`r18` | `content_rating`：`all_ages` / `sensitive` / `r18` | 与 G4.1 `Work.content_rating` 同 schema |
| `name_en_us` 四语槽 | `titles: SubmissionTitle[]`（`locale` + `title`）+ `aliases: string[]` | catalog `catalog.work.titles`；别名走 G4 `Work.aliases` |
| `intro_*` 四语槽 | `intros: SubmissionIntro[]`（`locale` + `value`） | catalog `catalog.work.intros`；`locale`/`value` 与 `CatalogIntro` 同名同型 |
| `release_date` 校验后丢掉 | `release_date` + `release_date_precision`，mint 写成 `released` | 与 `Work` 同 schema；infra 现已收 |
| `can_review` / `can_decide` / `can_revert` 平铺 | `viewer.can_*` | K9 |
| `users` 旁挂 map | 在 id 所在处嵌 `UserRef` | K10；禁用名 `user` |
| `proposer_uid` | `proposer: UserRef` | |
| `decided_by_uid` | `decider: UserRef \| null` | |
| `actor_uid` / `amender_uid` | `actor` / `amender: UserRef \| null` | |
| 修订 `action` | `revision_action` | G8：`ReviewItemPatch.action` 已是 `none/hide/remove/warn_user/restrict/escalate_idp` |
| 创建 200 + 裸 id | 201 + `Location` + 资源 | 01 §5 |
| 动词路径 resubmit/review/merge/decline/withdraw | `PATCH` `state` | 与 `updateReviewItem` 同形 |
| `next_before` | `next_cursor`（`cur_` 前缀，G16） | K11 游标 |
| 向导 `total`（过滤前） | **不下发** | 过滤后没有诚实的总数 |
| 嵌在向导里的 `pending[]` | 网页另打 `GET /me/work-submissions?state=pending,declined` | 两种对象 |

`work_id` / `proposal_id` / `revision_id` / `seq` / `from_seq` / `to_seq` / `is_nsfw` / `content_rating` / `original_language` / `release_date` / `aliases` / `viewer` / `include_nsfw` 与现 spec 对齐或未占用。

### 3.2 对象

**`WorkSubmission`（`object: "work_submission"`）** — 创建 201 / GET 200 / PATCH 200：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | DecimalID | catalog work id；与路径 `{work_id}` 相同 |
| `work_id` | DecimalID | 与 `id` 同值同 schema（G8：`work_id` 已是 DecimalID） |
| `display_name` | string ≤512 | catalog；自由文本句 |
| `state` | `live` `draft` `pending` `declined` `hidden` | 封闭；重读自 catalog，**不**本地写死 |
| `is_nsfw` | bool | 展示轴；与 `Work.is_nsfw` 同型 |
| `content_rating` | `all_ages` `sensitive` `r18` | 年龄轴；与 `Work.content_rating` 同型 |
| `submitter` | `UserRef \| null` | 本地 `creator_user_id`；不可渲染为删除用户 ref；没有为 `null` |
| `last_event` | `ClaimEventRef \| null` | 最近一次转移；含 `reason`（修 E1） |
| `first_acted_at` | date-time \| null | 调用者第一次动手；单条 GET 在 me 面有、审核面可空 |
| `acted_count` | int ≥0 | |
| `has_banner_attached` | bool | 只在 **201 创建回执**出现；GET/PATCH 不下发（创建之后横幅已经是作品封面） |
| `created_at` | date-time \| null | 最近事件时间，没有事件为 `null` |
| `viewer` | `WorkSubmissionViewer \| null` | 匿名为 `null`；本面 GET 是 required，viewer 非 null |

**`ClaimEventRef`（`object: "claim_event"`）**：`id`、`from_state`（上列五态或 `null`）、`to_state`、`reason`（string \| null，≤2000）、`actor: UserRef`、`created_at`。

**`WorkSubmissionViewer`**：`can_submit`（从 `declined`/`draft` 可 `pending`）、`can_withdraw`（从 `pending`/`live` 可 `draft`）、`can_delete`（仅 `draft`）、`can_review`（cookie 且 `user.Can(galgame.claim.review)`；Bearer 恒 false）。

**`WorkSubmissionSummary`（`object: "work_submission"`）**：列表用。`id` / `work_id` / `display_name` / `state` / `work_summary: WorkSummary` / `last_event` / `viewer`。重叠字段同名同型。`work_summary` 键名与 G6 `/me/playtimes` 相同（G8：`work` 已是另一个 schema）。

**`WorkSubmissionCandidate`（`object: "work_submission_candidate"`）** — 向导命中：`work_summary: WorkSummary`、`state`（`none` `live` `draft` `pending`；过滤后不会出现 `hidden`/`declined`）。未认领是 `none`。

**`EditForm`（`object: "edit_form"`）**：`work_id`、`values`（字段键 → 当前值，空对象永不 `null`）、`fields: EditField[]`、`vocabularies: EditVocabulary[]`、`viewer: EditFormViewer \| null`。

**`EditField`**：`key`（`catalog.work.*` pattern）、`field_type`（`text` `i18nmap` `enum` `int` `date` `list` `ref` `imagehash`）、`diff_hint`（`inline` `lines` `items` `image`）、`is_deprecated`、`is_locked`、`can_propose`、`can_review`、`max_elements` ≥0、`max_suppressed` ≥0、`vocabulary`（空串=无）、`encoding`（`token` `int` 或空）、`base` ≥0、`is_nullable`、`element`（对象或 `null`）。布尔全部 `is_`/`can_`（F1；infra 的 `deprecated`/`locked`/`nullable`/`closed` 在本面改名）。

**`EditVocabulary`（`object: "vocabulary"`）**：`name`、`is_closed`、`values: [{value, display_name, description}]`。透传 `/v2/vocabularies`。失败则 `vocabularies: []` 并 WARN `galgame edit: vocabularies unreadable`，枚举字段按只读画。

**`EditFormViewer`**：`can_review` —— **schema 里至少有一个字段 `can_review=true`**（普查 #28 / E20）。不是「调用者持有 `catalog.edit.review`」，也不是 `CanModerate`。Bearer 与 cookie 都按 schema 投影；staff 能力另见工作台 `can_decide`。

**`EditProposal`（`object: "edit_proposal"`）** — 创建 201 / 详情 200 / PATCH 200：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | DecimalID | 提案 id |
| `state` | `open` `merged` `declined` `withdrawn` | 重读自 catalog |
| `work_id` | DecimalID | `entity_id`，族是 `catalog.work` |
| `work_summary` | `WorkSummary` | 水合失败则仍 200，`display_name` 回落到空，WARN |
| `note` | string \| null ≤2000 | 与现 spec `note` 同型（string\|null，G8） |
| `decision_note` | string \| null ≤2000 | |
| `proposer` | `UserRef` | 不可渲染 → 删除用户 ref |
| `decider` | `UserRef \| null` | 未决为 `null`；补 E12 |
| `base_revision_seq` | int ≥0 | |
| `patch` | object | 键是 `catalog.work.*`；空对象永不 `null`。公开列表也发（编辑页要画别人的待审） |
| `effective_patch` | object | 折叠修正之后；工作台用它算 `can_decide` |
| `amendments` | `EditAmendment[]` | 永不 `null` |
| `created_at` / `updated_at` | date-time | |
| `decided_at` | date-time \| null | |
| `viewer` | `EditProposalViewer \| null` | |

**`EditProposalSummary`（`object: "edit_proposal"`）**：列表用，含 `patch`（编辑页要滤自己的、画别人的）。重叠字段同名同型。

**`EditProposalViewer`**：`is_proposer`、`can_withdraw`（`is_proposer` 且 `state=open`）、`can_decide`（cookie 且（`user.CanModerate()` 或作品 `creator_user_id` 是调用者）且 effective_patch 每个键 schema `can_review`；Bearer 恒 false）、`can_amend`（infra 接受的人：提案人或有审查资格者；本地不另加门，失败由上游 404/403 映射）。

**`EditAmendment`（`object: "edit_amendment"`）**：`id`、`seq` ≥1、`amender: UserRef`、`note` string\|null ≤2000、`created_at`。`set` / `unset` 只出现在 **201 创建回执**里，列表与详情的 amendments 链按 infra 公开形状不带 patch 片段。

**`EditRevision`（`object: "edit_revision"`）**：`id`、`seq` ≥1、`revision_action`（`created` `merged` `direct` `reverted`）、`changed_fields: string[]`、`actor: UserRef`、`amender: UserRef \| null`、`proposal_id: DecimalID \| null`、`created_at`。不发 `snapshot`、不发 `legacy_*`（v2 列表本来就不填）。

**`EditRevisionDiff`（`object: "edit_revision_diff"`）**：`from_seq`、`to_seq`、`fields: EditFieldChange[]`。`EditFieldChange`：`key`、`from`、`to`（JSON 任意值，含 `null`）。`field_type` / `diff_hint` 属于 schema，不在一条 diff 上重复（infra `repr.FieldDiff`）。

**`EditRevert`（`object: "edit_revert"`）** — 201：`proposal: EditProposal`、`revision: EditRevision \| null`。

**`WorkEditProposalCreateResult`（`object: "edit_proposal"`）**：就是 `EditProposal`。自动合并时 `state: merged`，并多一个 `revision: EditRevision`（同响应体上的可选字段；未合并为 `null`）。

### 3.3 词表

| 字段 | 取值 |
|---|---|
| 投稿 `state`（资源 / 查询） | `live` `draft` `pending` `declined` `hidden`；向导候选另加 `none`。查询缺席 = 不过滤（「我的」= 调用者有过的全部状态；审核队列缺席 = `pending`，与 infra `listModerationClaims` 相同） |
| 投稿 PATCH `state`（请求） | 提交者：`pending` `draft`；审核者：`live` `declined` `hidden` `unban`。`unban` 只出现在请求里，响应永远是 catalog 重读后的五态之一 |
| 提案 `state` | `open` `merged` `declined` `withdrawn` |
| 提案 PATCH `state` | `merged` `declined`（审核者）`withdrawn`（提案人） |
| `revision_action` | `created` `merged` `direct` `reverted` |
| `content_rating` | `all_ages` `sensitive` `r18`（G4.1） |
| `title_kind`（只在写入 titles 时；别名走 `aliases`） | `official` |
| `original_language` | 与 `Work.original_language` 同 pattern（BCP-47 \| null）；写入时 handler 拒 `null` |
| `release_date_precision` | `day` `month` `year` \| `null`（与 `Work` 同 schema） |
| 审核决策（内部打 catalog） | `approve` `decline` `ban` `unban` |

未知封闭枚举 → `400 UNKNOWN_ENUM_VALUE`。缺席过滤 = 不过滤。

### 3.4 认领状态机（K-G55）

权威在 infra `catalog/service/claim_lifecycle.go:42-56`。论坛本地没有 claim 表。v1 只暴露论坛实际执行的转移；响应永远是 **catalog 重读**（`GET /v2/me/claims/{id}` 或 `GET /v2/moderation/claims/{id}`），修 E19。

提交者 `PATCH`（cookie 或 Bearer，token 原样转给 catalog）：

| 当前 | `state` | catalog | 结果 |
|---|---|---|---|
| `draft` 或 `declined` | `pending` | `PATCH /v2/me/claims/{id} {state:pending}` | 重提 / 提交审核 |
| `pending` 或 `live` | `draft` | `{state:withdrawn}`（`draft` 是 withdrawn 的旧拼法，infra 两者同一 action） | 撤回 |

审核者 `PATCH`（**仅 cookie** 且 `user.Can(perm.GalgameClaimReview)`；Bearer 永不，K2）：

| 当前 | `state` | catalog `decision` | 结果 |
|---|---|---|---|
| `pending` | `live` | `approve` | 通过 |
| `pending` | `declined` | `decline`；`note` trim 后必填 | 拒绝 |
| `live` `draft` `pending` `declined` | `hidden` | `ban` | 下架 |
| `hidden` | `unban` | `unban` | **恢复被隐藏之前的状态**（infra `priorState`：最近一次 `to_state=hidden` 的 `from_state`；没有历史或 from 仍是 hidden/none → `live`。测试：draft 封禁后 unban → draft，live 封禁后 → live，无历史 → live，`claim_lifecycle_test.go:144-159`） |

`declined` 的 `note` 去空白后为空 → `422 VALIDATION_FAILED` `REQUIRED` pointer `/note`（infra `ErrClaimReasonRequired`，trim 之后）。空格不算有内容。`ban` / `approve` / `unban` / 提交者转移的 `note` 可选，≤2000。

非法转移 → `409 INVALID_STATE_TRANSITION`，`detail` 写明当前状态与允许的目标（英文诊断；K8 不给终端用户看）。不要把 infra 的英文 `cannot submit a claim in state "live"` 原样放进 `detail`，用论坛自己的句子。

持有 `catalog.claim.trusted` 的调用者 mint 直达 `live`：201 的 `state` 以 catalog 为准。v1 **不**给提交者 `PATCH state=live`（infra 允许从 draft 自发布；论坛 UI 没有这条，本轨不新开）。

### 3.5 编辑提案状态机

```
（空 patch 拒） --POST--> open | merged（automerge；kungal overlay AutomergeReview）
open --amend--> open
open --PATCH merged--> merged  + 一条 edit_revision
open --PATCH declined--> declined   （note trim 后必填）
open --PATCH withdrawn--> withdrawn （提案人）
（任意 seq） --POST edit-reverts--> 新提案和/或新修订
```

未知 `state` 查询值一律 `400 UNKNOWN_ENUM_VALUE`（Mine / Queue / 作品下列表同一词表，修 E18 的两套纪律）。空 status 不得静默填 `open`：catalog 回了空 → WARN 并丢掉该行。

### 3.6 投稿请求体（K-G63）

不要四语槽。形状对齐 `editspec/work.go` 的 mint `field_values`，键名给网页用的是 v1 的，BFF 再写成 `catalog.work.*`。

```json
{
  "display_name": "クラナド",
  "titles": [{"locale": "ja", "title": "CLANNAD"}],
  "aliases": ["クラナド"],
  "intros": [{"locale": "zh-Hans", "value": "……"}],
  "original_language": "ja",
  "content_rating": "r18",
  "is_nsfw": true,
  "release_date": "2004-04-28",
  "release_date_precision": "day",
  "banner_hash": "0123…cdef",
  "is_duplicate_confirmed": false
}
```

| 字段 | 规则 |
|---|---|
| `display_name` | 必填，trim 后非空，≤500 个字符（editspec `validateDisplayName`；mint 顶层 maxLength 512，取严的） |
| `titles` | 至少 1 条、最多 100（含稍后合并的 aliases）。`locale` 与 `CatalogIntro.locale` 同 schema；`title` ≤500。`title_kind` 不出现：这一列全是 official（kind=0） |
| `aliases` | `string[]`，与 `Work.aliases` 同 schema（元素 ≤512，maxItems 1000）；BFF 写成 titles kind=1、`lang=""`。与 official 合计仍受 100 条上限 |
| `intros` | 0–4 条；`locale` 封闭 `en` `ja` `zh-Hans` `zh-Hant`（`work_intros.go:13`）；`value` ≤50000（`maxIntroRunes`）。前端继续要求至少一条，后端不要求（旧面如此） |
| `original_language` | 必填；schema 因 G8 与 `Work.original_language` 同为 BCP-47 \| null，handler 拒 `null`。取值必须在 editspec `olangAllowed`（`work.go:44-50` 那 47 个标签）。未知 → `422 UNKNOWN_VALUE`，**不**原样透传 |
| `content_rating` | 必填，三值。**不**从 `is_nsfw` 推导，也不反推 |
| `is_nsfw` | 必填。缺席不得当 false（961 部封面被分级器标 explicit 的作品曾因此进 SFW 架） |
| `release_date` | 可选，`format: date`，与 `Work.release_date` 同 schema。有值则 mint 带 `released`（年/月/日按 precision 切；`year` → 只有 `y`）。缺席 = TBA，不写 catalog_release 行 |
| `release_date_precision` | 有 `release_date` 时默认 `day`；与 `Work.release_date_precision` 同 schema |
| `banner_hash` | 可选，与 `Image.hash` 同 pattern（64 hex）。事后 `attachBanner` |
| `is_duplicate_confirmed` | 默认 `false`。第一次同名软门 409；用户确认后再发 `true` |

`display_name` 缺席时 BFF 用 titles 按 ja → zh-Hans → zh-Hant → en 挑第一条非空（旧 `DisplayName()`），再写入 `catalog.work.display_name`。两者都空 → `422` `/display_name` `REQUIRED`。

### 3.7 可见性（K-G58）

公开 / optional 读作品下的提案、修订、diff：**复用 G4 的 `catalogWork`**（`internal/galgame/apiv1/like.go:44-58`）。catalog 不认识或 claim `hidden` → `404 NOT_FOUND`。合并 → `404 ENTITY_MERGED` `object: "work"` + `current_id`。**不查**本地 `published`。

提案 `site != kungal` 或 `entity_type != catalog.work` → 404（旧租户钉，`edit_handler.go:605-608`）。

`GET /work-submissions/{work_id}`：先 `GET /v2/me/claims/{id}`；404 且 cookie 持有 `galgame.claim.review` 再 `GET /v2/moderation/claims/{id}`；仍无 → 404。别人的 claim 与不存在不区分。Bearer 审核者走不进第二步（K2）。

向导：`include_nsfw` 默认 `false`，**显式**参数。合格谓词仍是 `CatalogItemWizardEligible`（未认领，或 kungal/galgame_wiki 且状态 live\|draft\|pending）；过滤发生在上游页之后，所以没有诚实的 `total`，也不收 `include_total`（F9）。

NSFW 作品本身不因 `include_nsfw=false` 从投稿 GET 里 404（提交者要看见自己的 NSFW 草稿）。

### 3.8 副作用（K-G60）

| 事件 | 做什么 | 失败 |
|---|---|---|
| mint 成功 | `SubmitLocal`（`EnsureLocalStub` + `SetCreatorIfUnset`；`published` 保持 false） | **不**失败请求。`slog.Warn("submit: failed to record local creator", "work_id", …)` |
| mint 且带了 `banner_hash` | `CreateEditProposalUser` + 若未自动合并则 `MergeEditProposalUser`；201 的 `has_banner_attached` | **不**回滚 mint。`slog.Error("submit: banner attach failed, submission stands without cover", "work_id", …)` / `("submit: banner proposal merge failed, submission stands without cover", "work_id", "proposal_id", …)` |
| 提案创建且 `state=open`（未自动合并） | `NotifyRequested` 给作品 `creator_user_id`；`INSERT galgame_activity (wiki_pr_id, work_id, user_id, type='GALGAME_PR_CREATION')`，**`wiki_pr_id` = catalog 提案 id**（`ON CONFLICT (wiki_pr_id) DO NOTHING`） | **不**失败请求。`slog.Warn("galgame edit: requested notification failed", "proposal_id", …)` / `("galgame edit: activity timeline write failed", "proposal_id", …)` |
| 提案 `PATCH merged` 且 merger ≠ proposer | `moemoepoint.Award` +1，reason `content_approved`，键 `kungal:galgame_edit_merged:<proposal_id>`，ref `galgame_pr:<work_id>`；`Touch` `resource_update_time`；`NotifyMerged` | Award 仍是 fire-and-forget（pusher 自己记）。Touch / 通知失败 `slog.Warn("galgame edit: resource_update_time bump failed", "work_id", …)` / `("galgame edit: decision notification failed", "proposal_id", "kind", "merged", …)`，请求仍 200 |
| 提案 `PATCH declined` | `NotifyDeclined`（内容带理由） | WARN，请求仍 200 |
| 投稿审核 `live` | **不**在本 HTTP 发萌萌点（cron `kungal:claim_approved:<eventId>`，+3） | — |
| GET | **零写入** | — |

自合并不发萌萌点（E24）。`wiki_pr_id` 继续写 catalog 提案 id，无迁移；命名债见 §8 O2。

### 3.9 鉴权

本地门只投影 infra 的决定（wave 178 删了本地裁决门）。本地门的唯一用途是避免一次注定失败的上游调用，且只用 `user.Can` / `user.CanModerate`（K2：Bearer 永不持有 staff）。

| 面 | 本地门 | infra |
|---|---|---|
| 投稿写、我的投稿、向导、edit-form、建提案、修正、撤回提案、回滚 | `Auth`（required） | token 所有权 / `catalog:edit` |
| 投稿队列、我审核过的、投稿 PATCH 审核态 | cookie + `user.Can(perm.GalgameClaimReview)`；Bearer → `403 PERMISSION_REQUIRED`，0 次 moderation 调用 | `catalog.claim.review` |
| 编辑队列 `GET /edit-proposals`、工作台 `can_decide`、提案 PATCH `merged`/`declined` | cookie + `user.CanModerate()`（**角色**，不是 `galgame.claim.review`）；作品主人可进工作台只读/决定（旧 `isGameOwner`） | `catalog.edit.review` + token 推导的 owner（kungal overlay `OwnerReview: true`） |

两套键的统一是开放问题（§8 O1），本轨按代码原样写。

每个 `viewer.can_*` 是 infra 答案的投影（K9）。匿名 `viewer: null`。Bearer 的 `can_review` / `can_decide` 不含 staff。

### 3.10 对 infra 的词表差（G4.1 课，逐行）

每一个本轨传给或读自 catalog 的封闭词表 / 长度上限，对 infra 的定义比，不对 dev 数据恰好有的值比。有意丢行的分支打 WARN，带 greppable 属性（`work_id` / `proposal_id` / `field` / `value`）。

| 字段 | infra 的集合 / 上限（文件:行） | 本轨 | 裁定 |
|---|---|---|---|
| claim `state` | `none,live,draft,pending,declined,hidden`（`repr/me.go:16`；整数 0/1/2/3/4 在 `model/constants.go:201-207`；null site = unclaimed → `none`） | 投稿资源不下发 `none`；向导候选发 `none`。未知值 WARN 丢掉该行 | 对齐闭集 |
| 提交者转移 | `live\|pending\|withdrawn`，`draft` = withdrawn 旧拼（`me_write_routes.go:185`，`me_claims.go:447-454`）；from 见 `claim_lifecycle.go:46-49` | 只暴露 `pending`/`draft` | 收窄：不给提交者自发布 `live` |
| 审核决策 | `approve\|decline\|ban\|unban`（`me_write_routes.go:246`）；unban 无固定目标（`repr/me.go:85-97`） | PATCH `live/declined/hidden/unban`；响应重读 | 对齐。unban 落点见 §3.4 |
| decline note | trim 空拒（`claim_lifecycle.go:140-141` `ErrClaimReasonRequired`）；schema ≤2000 | 同；空格不算有内容 | 对齐 |
| 提案 `state` | `open,merged,declined,withdrawn`（`repr/me.go:63`；`editing.go:10-20` 0/1/2/3） | 同四值。空 status **不**填 `open`，WARN 丢行 | 对齐。旧解码静默填 open 是缺陷 |
| 修订 `action` | `created,merged,direct,reverted`（`repr/edit.go:11`；`editing.go:23-35`） | `revision_action` 同四值。未知 WARN 丢行 | 对齐。改名避 G8 |
| `display_name` | mint 顶层 ≤512（`me_write_routes.go:171`）；editspec ≤500 runes（`work.go:360`） | schema ≤500 | 取严的 |
| titles | 1–100 条；title ≤500 runes；kind 0/1/2（`work_titles.go:20-22,36-77`）；alias 可空 lang | official 走 `titles`，alias 走 `aliases`；合计 100 | 对齐。kind=3 search_hint 本面不写 |
| intros | lang ∈ `en,ja,zh-Hans,zh-Hant`；intro ≤50000（`work_intros.go:13,50`，`curated.go:107`） | 同 | 对齐。`CatalogIntro.value` 上限 65535 是读侧，写入取 editspec |
| olang | 47 个标签（`work.go:44-50`） | 未知 422，不透传 | 对齐 |
| `content_rating` | 0/1/2 = all_ages/sensitive/r18（`work.go:377-386`；G4.1 三值） | 同三字符串 | 对齐。禁止从 `is_nsfw` 推导 |
| `display_nsfw` | bool（`work.go:275-278`） | `is_nsfw` | 展示轴；与年龄轴分开 |
| `released` | `{y 1970–2200, m 0–12, d 0–31}`（`me_write_routes.go:161-165`）；只 mint 车道收 | `release_date` + precision 译成它 | **送达**。旧面丢掉是 bug |
| 提案 / 修正 / 回滚 note | ≤2000（`me_write_routes.go:213,239,247,263`） | schema ≤2000；K19：先按原始值判长 | 对齐。生产最大 385/174 |
| 修订每作 | 生产最大 seq 15 | 页码 `page`+`limit`，默认 20 | 一页盖住 |
| 每 actor claims | 最大 1,283 | 游标 limit 1–100 默认 20 | 修 E2 |
| 向导页大小 | 旧默认 12，无上限 | limit 1–100 默认 20；非法 400 | 修 E18 |
| 公开提案 | `/v2/catalog/proposals` **不发 patch / decision_note**（`catalog_edit_routes.go:100`） | 作品下列表仍发 `patch`（编辑页要画）；hidden 作品 404 | 修 E9 的 hidden 泄漏；patch 对可见作品保留 |
| 单条提案 | `GET /v2/me/proposals/{id}`、`GET /v2/moderation/proposals/{id}?include=patch,amendments`、`GET /v2/catalog/proposals/{id}` | 工作台走 moderation/me，带 patch | 用上旧代码没用的单条面 |
| 单条 claim | `GET /v2/me/claims/{id}`、`GET /v2/moderation/claims/{id}`（`me_write_routes.go:44,87`） | G17 GET 用这两条，**不**用作品搜索 | 修「没有单条读」 |
| 审核队列 | `GET /v2/moderation/claims`，默认 pending，cursor 是 `event\|work`（`me_claims.go:133-180`） | `GET /work-submissions?state=pending` 走它，**不**走应用 key 作品搜索 | 修 E25 |
| 审核队列 `claim_state` | live/draft/pending/declined/hidden，无 `none`（`me_claims.go:191-208`） | 同；缺席 = pending | 对齐 |
| If-Match | PATCH/决定/修正/删草稿 **必带**，缺席 428（`me_write_routes.go:51,78,84,93,111`） | BFF 内部发 `If-Match: *`（旧 `ifMatchStar`），**不**暴露给客户端 | 01 暂不做 If-Match；428 当论坛 bug → 500 |
| 幂等 | catalog 用户面 POST 支持 Idempotency-Key | 四个创建 POST **必带**（K12） | 论坛比 infra 更严 |
| claim 日配额 | 429 `QUOTA_EXCEEDED`（`me_write_routes.go:34`） | 任何上游 429 → 503 + `Retry-After` + WARN（G6 §3.15） | 不嗅正文 |
| `catalog.work` 字段键 | `editspec/work.go:16-37` 那一张（display_name / olang / content_rating / titles[+suppr] / intros / display_nsfw / tag_ids / labels / engine_ids / series_ids / links / covers / screenshots / credits[+suppr] / roster[+suppr]） | 提案 `patch` 键必须带前缀 `catalog.work.`；未知键 422 | 对齐。covers 元素只允许 editspec 成员（`image_hash`/`kind`/`portrait_pinned`/`sexual`/`violence`） |

仍然有意丢掉、且打 WARN 的：提案 `site` 不是 kungal 或族不是 `catalog.work` 的行（404，不出现在列表）、修订 `revision_action` 不在四值、claim/提案 `state` 不在闭集、水合失败的 `WorkSummary`（条目仍在，名字空）、时间戳解析失败的行。

### 3.11 上游错误映射

catalog 用户面一次调用的失败，按下表进 v1。**不得**落到无 code 的 500。**不得**把上游英文句子写进响应 `detail`（E26）。复用 G6 `mapUserPlane`（`internal/galgame/apiv1/upstream.go:15`），本轨多出来的 409 码按行追加。

| 上游 | v1 | 说明 |
|---|---|---|
| 缺 scope（`ErrInsufficientScope` / 403 `SCOPE_REQUIRED`） | `403 SCOPE_REQUIRED` | |
| 会话里没有 access token | `401 INVALID_CREDENTIAL` | |
| token 被拒（`ErrUnauthorized` / 401） | `401 INVALID_CREDENTIAL` | |
| 403 `USER_IDENTITY_REQUIRED` | `500 INTERNAL_ERROR` | G6 §3.16：凭据不是用户 token，论坛自己发错了 |
| 403 `SITE_NOT_BOUND` | `500 INTERNAL_ERROR` | kungal 客户端未绑 site，是部署问题 |
| 403 `CLAIM_NOT_OWNED` / `TENANT_MISMATCH` / 非主人 | `404 NOT_FOUND` | 存在性不披露。本人对自己的资源被拒才是 `403 PERMISSION_REQUIRED` |
| 404 | `404 NOT_FOUND` | |
| 409 `DUPLICATE_SUSPECTS` | `409 DUPLICATE_SUSPECTS`，扩展 `suspects[]`（`id` + `display_name`） | infra me 域，type URI 与 infra 逐字相同（K4） |
| 409 `ALREADY_EXISTS` | `409 ALREADY_EXISTS` | refs 已指向一部作品还带了 field_values / released |
| 409 幂等 | `409 IDEMPOTENCY_KEY_REUSED` / `IDEMPOTENCY_REQUEST_IN_PROGRESS` | |
| 409 `INVALID_STATE_TRANSITION` / `DECISION_ALREADY_MADE` / `ClaimTransitionError` | `409 INVALID_STATE_TRANSITION` | `detail` 用论坛自己的英文，写明当前态与允许目标 |
| 412 | `409 INVALID_STATE_TRANSITION` | 丢了比赛；BFF 发的是 `If-Match: *`，412 几乎不该出现 |
| 428 | `500 INTERNAL_ERROR` | BFF 忘了 If-Match，论坛 bug |
| 422 缺 decline note / 空 display_name / 非法 olang / 超长 / 空 titles | `422 VALIDATION_FAILED`，`errors[]` 的 pointer 映射到 v1 字段（`/note`、`/display_name`、`/titles/0/title`、`/original_language`、`/released`） | 中文不进 `detail` |
| 422 其它认得出的 | 按 pointer/`reason` 映射 | |
| 认不出的 422 | `422 VALIDATION_FAILED` `NOT_ALLOWED_VALUE`（pointer 空）+ WARN 带上游 code | G6 §3.16 |
| 400 | `500 INTERNAL_ERROR` | 论坛发错了请求 |
| 429（日配额或共 IP） | `503 SERVICE_UNAVAILABLE`，转发 `Retry-After`，WARN `catalog user plane: upstream 429 mapped to 503` `upstream_status` | G6 §3.15 |
| 5xx / `ErrUpstream` / 传输 / 未配置 | `503 SERVICE_UNAVAILABLE` | |

### 3.12 实现时按 G8 / G14 / F1 改的名（只增不改，以此为准）

| 前文 / 旧 | 实际 | 撞了谁 |
|---|---|---|
| 修订 `action` | `revision_action` | G8：`ReviewItemPatch.action` |
| `confirm_duplicates` | `is_duplicate_confirmed` | F1 布尔 |
| `banner_attached` | `has_banner_attached` | F1 |
| `display_nsfw` | `is_nsfw` | F1；且与 `Work.is_nsfw` 同 schema |
| `merged` 布尔 | 不发；看 `state` | F1 |
| schema `deprecated` / `locked` / `nullable` / vocab `closed` | `is_deprecated` / `is_locked` / `is_nullable` / `is_closed` | F1 |
| 嵌作品键 `work` | `work_summary` | G8：`work` 已是另一个 schema（G6.1） |
| `note` 非空 string | `string \| null` | G8：`ReportCreate.note` / `ActivityResource.note` 已是 string\|null |
| 四语槽 | `titles` / `intros` 用 `locale` | `lang` 已是会社/角色/人名的自身语言且可空；intro 的 `zh-Hans` 也过不了 F1 对非 languageTag 属性的 snake_case 检查 |
| 向导 `total` | 不下发 | F9：过滤后没有诚实总数，不能声明 `total` |

`state` / `viewer` / `object` 在 G8 白名单（`gates/repr.go:14-19`），词表可以按对象不同。`seq` / `from_seq` / `to_seq` / `proposal_id` / `revision_id` / `title_kind` / `decision_note` / `effective_patch` / `revision_action` / `has_banner_attached` / `is_duplicate_confirmed` / `can_submit` / `can_withdraw` / `can_delete` / `can_review` / `can_decide` / `can_revert` / `can_amend` / `is_proposer` 现 spec 未占用。

每个字符串：封闭枚举、format、pattern 或自由文本句（G14）。`display_name` / `title` / `value` / `note` / `decision_note` / `aliases[]` 是自由文本。

### 3.13 编排者裁决（2026-09-24，覆盖前文与 §7 同名条目）

- **作品下的提案列表不发 `patch`（推翻 O5）。** infra 公开面 `GET /v2/catalog/proposals` 有意不发 patch 与 decision note（`catalog_edit_routes.go:100`），匿名 / optional 调用也拿不到别的来源。`listWorkEditProposals` 的条目是 infra 公开形状：`id`、`state`、`work_id`、`proposer`、`note`、`base_revision_seq`、`created_at` / `updated_at`、`viewer`；**无 `patch` / `effective_patch` / `decision_note`**。patch 只在工作台 `GET /edit-proposals/{proposal_id}` 下发（提案人、作品主人或审核者，走 me / moderation 单条面）。编辑页「别人的待审提案」画成摘要行，审核者点进工作台看 diff。`EditProposalSummary` 与 `EditProposal` 重叠字段同名同型（K10）。变异 #8 追加：匿名或普通用户的作品下列表出现 `patch` → 必须变红。
- **`edit-form` 的 `viewer.can_review` 删掉。** infra `ObjectSchema` 明写「Actor capabilities are not evaluated」（`repr/schema.go:13`），「schema 里有可审字段」不随调用者变，放进 `viewer`（K9）是错的。能不能审一个提案看工作台的 `viewer.can_decide`。`EditField` 的每个属性逐项对 infra `SchemaField` 核：infra 没有的（包括 `can_propose` / `can_review` 若不存在）不发，不从论坛本地推导。
- **`viewer.can_revert` 同理核实。** 旧 `canRevert` 由 schema 的 `CanReview` 推导；若实现时核实 schema 不随 token 变，`can_revert` 等价于「有用户 token」，则删掉这个字段：回滚按钮给已登录者，infra 拒绝时 `403 PERMISSION_REQUIRED`。若 infra 的用户面 schema 确实随 token 变（例如 owner overlay），保留并在 §3.16（实现记录）写明依据的 infra 文件:行。
- **`edit-form` 没有 hidden 例外。** 可见性就是 G4 `catalogWork`：hidden / 未知 404；草稿作品不是 hidden，照常可编辑。删掉 4.9 里「snapshot 对 hidden 所有者 200 则放行」那一句。
- **`WorkSubmission` 不发 `created_at`。** 前文把它定义成「最近事件时间」，名不副实；时间在 `last_event.created_at` 与 `first_acted_at`。
- **201 回执各用自己的 schema，不在同一对象上按操作增减字段（K10）。** 投稿创建回 `WorkSubmissionCreated`（`WorkSubmission` 全部字段 + `has_banner_attached`）；修正创建回 `EditAmendment`，**不带** `set` / `unset`（客户端自己刚发的），列表 / 详情同形。
- **§3.3 词表里的 `title_kind` 行删掉**：请求体不收它（titles 全是 official，别名走 `aliases`）。
- **O1（两套审核门）维持原样交编排者；O3（If-Match `*`）、O4（不开自发布）、O6–O9 照准。**

### 3.14 编排者裁决二（2026-09-24，37 批准契约时定，覆盖前文、§5 与 §7 同名条目）

- **O1 → 两个队列都改权限键（permission-first，见记忆 kungal-permission-first）。** 审投稿与审编辑是两种能力，分成两个键：
  - `galgame.claim.review` 不变：投稿队列、我审核过的、投稿 PATCH 审核态。
  - 新键 **`galgame.edit_proposal.review`**：编辑队列 `GET /edit-proposals`、工作台的审核者路径（作品主人路径照旧，infra `OwnerReview`）、提案 PATCH `merged` / `declined` 的本地预检。
  - 新键进 `pkg/perm` 的 moderator bundle。基线在代码里，覆盖表 `role_permission_override` / `user_permission_override` 只存增量，新键没有增量行，所以默认授予 = moderator / admin / ren = 今天 `CanModerate()` 的集合：**上线时没人多拿、没人少拿，不需要种子迁移**。
  - 检查一律 `user.Can(perm.GalgameEditProposalReview)`，不看角色；Bearer 恒 false（K2）。
  - 镜像四处一起改：`pkg/perm/perm.go` 键 + bundle、`pkg/perm/perm_test.go` 金表、`apps/web/app/composables/useCan.ts`、`apps/web/app/constants/permission.ts` 的 `KUN_PERMISSION_META`（`/admin/permission` 矩阵从这里画）。再加 BE↔FE 镜像测试（记忆 kungal-claim-review-authz 的那条）。
  - 网页 `galgame-edit/review/Container.vue` 停用 `useRole().canModerate`，改 `useCan('galgame.edit_proposal.review')`。
  - 这推翻了阶段一「INFRA-PROXY 七项留在角色门、永不进 `pkg/perm`」的规矩里「编辑审核」这一项。先例是 `galgame.claim.review`：同样是 infra 真值前面的一道论坛查看门。
- **O2 → 不迁移，列名 `wiki_pr_id` 只留在库里。** 任何 v1 字段一旦暴露它，名字是 `proposal_id`。现在 v1 没有字段暴露它（动态流的 `galgame_pr_creation` 条目不带这个 id）。
- **If-Match 不写死 `*`（推翻 O3 / K-G64 后半）。** infra 要它，是为了挡住两个审核者的竞态，`*` 把这层保护丢了。
  - 读：`GET /work-submissions/{work_id}` 与 `GET /edit-proposals/{proposal_id}` 回 **`ETag` 响应头**，原样转发 catalog 的校验值（`me_write_routes.go:189,222`）。每个写响应（投稿 PATCH、提案 PATCH、修正 201）也带重读后的新 `ETag`。01 把 ETag 列为「暂不做、以后是加法」，这是第一次加，只加在这几个面。
  - 写：`PATCH` / `DELETE /work-submissions/{work_id}`、`POST /edit-proposals/{proposal_id}/amendments`、`PATCH /edit-proposals/{proposal_id}` 收可选的 **`If-Match`** 请求头，原样转给 catalog。缺席才回落 `*`，照顾 App 与旧客户端。网页每次都带：列表条目不带校验值，网页在写之前取一次单条 GET（或用上一个写响应的 `ETag`）。
  - 上游 412 → **`412 PRECONDITION_FAILED`**（论坛注册表已有，与 infra 同码），`detail` 用论坛自己的句子。上游 428 仍是论坛 bug → 500。
  - 测试钉住两条路径：带 `If-Match` 时原样转发、412 映射成 `PRECONDITION_FAILED`；不带时发 `*`。
- **CHANGELOG 要写的已修缺陷**：「我的投稿」以前的 `next_before` 恒 0，20 条以后永远到不了（每 actor 最多 1,283 条）；v1 是游标集合。
- **变异题追加**（接 §7）：
  - #23：编辑队列或提案 PATCH `merged` 用角色而不是 `galgame.edit_proposal.review` 判（撤销覆盖后的 moderator 仍能进）→ 403；
  - #24：`If-Match` 不转发、写死 `*`（夹具 catalog 对旧值回 412）→ 必须 `412 PRECONDITION_FAILED`；
  - #25：缺席 `If-Match` 时没有回落 `*`（上游收到空头）→ 上游收到 `*`；
  - #26：单条 GET 不回 `ETag` → 响应头存在，且等于上游值。

### 3.16 实现记录（G7b 编辑引擎半，2026-09-24；编排者逐条批准，覆盖前文同名条目）

逐条对 nextmoe-infra `5dc86518` 核过（路径相对 `apps/api/internal/platform/`）。

- **`EditProposal.decision_note` 删掉。** infra 任何面都不发它：`apiv2/repr/me.go:59-78` 的 `ProposalRecord` 没有这一列，`apiv2/handler/me_proposals.go` 的 `proposalFrom` 不填；只有决定 POST 的 `ProposalDecisionRecord.note` 回显一次（`repr/me.go:99-107`）。留着它每次 GET 都是 `null`。拒绝理由照旧经 `NotifyDeclined` 送到提案人（内容 `作品名：理由`），测试钉住。
- **`viewer.can_revert` 与 `EditForm.viewer` 删掉。** `GET /v2/catalog/schemas/{object}` 不鉴权、不看调用者（`apiv2/handler/catalog_schema_routes.go:22-30`，`getCatalogSchema` 不收 actor），schema 不随 token 变。回滚按钮给所有已登录者；infra 拒绝 → `403 PERMISSION_REQUIRED`。匿名本就合法可「试」的动作不做成 viewer 旗（05 §9）。修订列表信封不带 `viewer`。`EditField` 只发 infra `SchemaField` 有的属性（`repr/schema.go`），没有 `is_locked` / `can_propose` / `can_review`。
- **编辑队列分两面。** `/v2/moderation/proposals` 只出 `open`（`me_proposals.go:397` 强制 `f.Status = editing.StatusOpen`；旧面的状态标签页因此永远显示 open）。`state=open`（缺席）走 moderation + 用户 token；`merged` / `declined` / `withdrawn` 走公开面 `GET /v2/catalog/proposals?object=work&site=kungal&state=…`（应用 key），同一道本地键门在前。两面的条目都不带 patch。游标用 `collect.EncodeCursor` 绑定 (面, state)，跨过滤条件复用 → `400 INVALID_CURSOR`。
- **建提案 / 回滚回的是提案，不是修订。** `createMyProposal` 与 `revertModeration` 都回 `ProposalRecord`（`me_write_routes.go:398-409,563-574`）。自动合并（`state=merged`）时，BFF 读 `GET /v2/catalog/revisions?object=work&entity_id=`（默认 `recorded_desc` 新的在前，`catalog_edit_routes.go:32`；`limit=100`，生产每作至多 15 条），按 `proposal_id` 配出修订。读失败或配不上：**上游写已成功，绝不回 5xx**（客户端会重试、再建一个提案），201 照回、`revision: null`，WARN `galgame edit: merged revision not found`。
- **修正回的是整条提案。** `amendMyProposal` 回提案 + 修正链（`proposalAfterWrite`，`me_proposals.go:360-372`）。v1 201 是链里 `seq` 最大的那条 `EditAmendment`；`ETag` 取同一响应。
- **`can_decide`（仅 UI 提示，infra 是权威）** = cookie ∧ `state=open` ∧（`user.Can(galgame.edit_proposal.review)` ∨ 本地 `creator_user_id` = 调用者）。infra 的主人是 `catalog_work.owner_user_id`（`catalog/editspec/work.go:141-151`，`editing/engine.go:73-78`），v2 没有任何面暴露它（`repr/resource.go:9-15` 的 `Claim` 只有 site/state/content_limit），所以用本地创建者代位；不一致只会落到 infra 403 → `PERMISSION_REQUIRED`。Bearer 恒 false。
- **`can_amend`** = `state=open` ∧（`is_proposer` ∨（cookie ∧（键 ∨ 本地主人）））。infra `fenceProposal(…, proposerOrReviewer=true)`（`me_proposals.go:117-128`）。
- **工作台读法。** 先 `GET /v2/me/proposals/{id}?include=patch,amendments`（infra 对非提案人 404，`me_proposals.go:231-247`）；404 且 cookie ∧（键 ∨ 本地主人）再 `GET /v2/moderation/proposals/{id}?include=patch,amendments`（要该作品的审查资格，`me_proposals.go:249-270`）。上游 403 / `TENANT_MISMATCH` / 404 一律 404。Bearer 不走第二步（K2）。两条路都再钉一次租户（`site=kungal` ∧ `entity_type=catalog.work`）。
- **合并通知不带「（审核时有修正）」后缀。** 旧面读 `rev.AmenderUID`，但 v2 决定回的是 `ProposalDecisionRecord`，这个后缀自割接以来一次都没出现过；v1 代码不许写人类语言字面量（F8），通知内容就是作品名。
- **实现时按 G8 改的名（只增不改）**：表单当前值 `values` → `field_values`（infra 原名；`values` 已是词表的数组）；diff 的 `fields` → `field_changes`（`fields` 已是表单字段数组，`changes` 已被 `RolePermissionMatrixPatch` 占用）；修订的 `amender` → `last_amender`（`EditAmendment.amender` 非空，修订的可空；也正是 infra「最后一个修正人」的意思）；词表的 `name` → `vocabulary`（`name` 在全 spec 是可空）；元素 `type` → `element_type`、成员 `type` → `member_type`（`type` 已是 problem 的 uri）。`EditField.encoding` 是 `token`/`int`/`null`（无词表时为 `null`，不用空串）。
- **工作台第二步不做本地预检。** 作品主人的资格要作品 id，读到提案之前不知道，所以 cookie 调用者在 me 面 404 之后一律再问 moderation 面，由 infra 判资格；403 / 404 都回 404。Bearer 不走第二步。
- **提案 PATCH 先读公开面。** `GET /v2/catalog/proposals/{id}`（应用 key）拿作品 id、提案人、状态、site：租户钉、非 open 的本地 409、撤回只许提案人、决定的本地资格（键或本地主人）都在任何用户面调用之前判完；Bearer 的 `merged`/`declined` 连公开面都不读。写完用 me（撤回）或 moderation（决定）面重读，响应与 `ETag` 取自重读。
- **写成功之后不回 5xx（评审，2026-09-24）。** 提案 PATCH 的副作用（萌萌点、`resource_update_time`、通知）按写本身的回答跑：决定看决定记录的 `to_state`（infra 读回结果，抑制规则可能把 merge 关成 declined），不看之后的重读。重读失败 → WARN `galgame edit: proposal read-back failed after the write`，用公开面读到的提案 + 新状态回 200，`ETag` 为空。建提案 / 回滚 / 修正 / 决定之后的用户或主人查询失败 → WARN，用删除用户 ref 和空主人表回 2xx；修正回执里没有修正链 → 重读一次。Bearer 的 `merged` / `declined` 在任何 catalog 调用之前就 403。变异题追加 R1（重读失败让决定 5xx）、R2（写后查询失败让写 5xx）、R7（Bearer 先读公开面再拒）。
- **CORS 放行 `If-Match`、暴露 `ETag`**（`internal/middleware/cors.go`），否则跨域客户端既送不出校验值也读不到它。
- **旧路由删除顺带删了 `optAuth` 分组。** 它最后一个用户是 `GET /galgame/:id/edit/revisions`。空前缀 `Group` 在 Fiber 里是 `/api` 上的 `Use()`，所以此后注册的遗留路由（管理员清除内容、投稿）的链上不再有 `OptionalAuth`；它们都挂着 `Auth`，`Auth` 自己解析身份，少的只是一次重复解析。
- **变异题追加**（接 §3.14）：
  - #27：`PATCH state=declined` 的 `NotifyDeclined` 内容不带理由 → 提案人的通知里必须有 note 原文；
  - #28：编辑队列 `state=merged`（或 declined / withdrawn）仍打 `/v2/moderation/proposals` → moderation 面 0 次调用，上游是 `/v2/catalog/proposals?state=merged&site=kungal`；
  - #29：自动合并后修订读失败时回 5xx（或 201 前就报错）→ 201，`state: merged`，`revision: null`，上游提案只建了一次。

## 4. 逐条操作

通用：401 `MISSING_CREDENTIAL` / `INVALID_CREDENTIAL`（required；optional 的坏 Bearer）、403 `ACCOUNT_BANNED`、500 `INTERNAL_ERROR`、503 `SERVICE_UNAVAILABLE`（会话存储 / userclient / catalog / 传输）。每个 401 带 `WWW-Authenticate`。v1 全部 `Cache-Control: no-store`。缺 OAuth token 的已登录会话：写与私密读走 401 `INVALID_CREDENTIAL`。请求体的错一律 `422 VALIDATION_FAILED`；`400 INVALID_PARAMETER` 只给参数（路径、查询、`Idempotency-Key`）（G6 §3.16）。

### 4.1 `createWorkSubmission` · `POST /work-submissions` · required · 201 `Location` + `WorkSubmission`

`Idempotency-Key` 必带。`Location`：`/api/v1/work-submissions/{work_id}`。体见 §3.6。同名软门：`is_duplicate_confirmed=false`（默认）且 catalog 拒 → 409，`suspects[]` 列出 live 同名作品，什么都不写。再带 `true` 才 mint。持有 `catalog.claim.trusted` 时 catalog 直达 `live`。副作用 §3.8。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（缺幂等键 / 格式） |
| 403 | `SCOPE_REQUIRED` `ACCOUNT_BANNED` |
| 409 | `DUPLICATE_SUSPECTS` `ALREADY_EXISTS` `IDEMPOTENCY_KEY_REUSED` `IDEMPOTENCY_REQUEST_IN_PROGRESS` |
| 422 | `VALIDATION_FAILED`（缺标题、缺 `is_nsfw`、非法 olang / `content_rating`、超长）`CONTENT_REJECTED` |
| 503 | catalog / 共享 429 |

### 4.2 `getWorkSubmission` · `GET /work-submissions/{work_id}` · required · 200 `WorkSubmission`

G17。提交者或（cookie）审核者；其余 404。

| 状态 | code |
|---|---|
| 403 | `ACCOUNT_BANNED` |
| 404 | 不是自己的、不存在、hidden 对非审核者 |
| 503 | catalog |

### 4.3 `updateWorkSubmission` · `PATCH /work-submissions/{work_id}` · required · 200 `WorkSubmission`

体 `{state, note?}`。转移见 §3.4。响应是 catalog 重读，不是本地 `to_state`。审核态仅 cookie + `galgame.claim.review`；Bearer 403，0 次 `/v2/moderation/claims/{id}/decisions`。

| 状态 | code |
|---|---|
| 403 | `PERMISSION_REQUIRED`（Bearer 审 / 无键）`SCOPE_REQUIRED` `ACCOUNT_BANNED` |
| 404 | 不是自己的（提交者路径）、不存在 |
| 409 | `INVALID_STATE_TRANSITION` |
| 422 | `VALIDATION_FAILED`（`declined` 且 note 空） |

### 4.4 `deleteWorkSubmission` · `DELETE /work-submissions/{work_id}` · required · 204

仅 `draft`。catalog `DELETE /v2/me/claims/{id}` 先；成功后再 `DeleteLocalDraft`。带 `galgame_resource` 的本地行按设计保留（`NOT EXISTS` 守卫，0 行不是错误）→ 仍 **204**。其它状态 catalog 拒 → `409 INVALID_STATE_TRANSITION`。catalog 已删、本地 `Exec` 返回 SQL 错误 → `500 INTERNAL_ERROR`，`slog.Error("delete draft: catalog draft gone, local row not cleaned", "work_id", …)`。

### 4.5 `listMyWorkSubmissions` · `GET /me/work-submissions` · required · 200 游标信封

Query：`cursor`、`limit`（1–100，默认 20）、`state`（CSV，可选）、`include_total`。缺席 `state` = 调用者 `kind=submitted` 的全部状态。网页「我的提交」传 `state=pending,declined,draft`（与旧默认 `mineStates` 相同，已通过的 live 不进这张表）。游标绑定过滤条件。

### 4.6 `listMyWorkSubmissionReviews` · `GET /me/work-submission-reviews` · required · 200 游标信封

`kind=audited`。仅 cookie + `galgame.claim.review`。Bearer 403。`state` 可选。网页不传 `state`（旧「审核历史什么都不藏」）。

### 4.7 `listWorkSubmissions` · `GET /work-submissions` · required · 200 游标信封

审核队列。仅 cookie + `galgame.claim.review`。Query：`state`（缺席 = `pending`）、`cursor`、`limit`、`include_total`。上游 `GET /v2/moderation/claims`。BFF 把 infra 的 `event|work` 包进 `cur_`。

### 4.8 `listWorkSubmissionCandidates` · `GET /work-submission-candidates` · required · 200 游标信封（无 `total`）

Query：`q`（必填，trim 空 → 422 `/q` `REQUIRED`）、`cursor`、`limit`（默认 20）、`include_nsfw`（默认 `false`）。合格谓词见 §3.7。不把调用者自己的 pending 嵌进同一信封；网页另打 4.5。

### 4.9 `getWorkEditForm` · `GET /works/{work_id}/edit-form` · required · 200 `EditForm`

`GET /v2/moderation/snapshots/work/{id}` + `GET /v2/catalog/schemas/work` + `GET /v2/vocabularies`。hidden / 未知 404（提交者打开自己的 hidden 走 4.2 的审核路径，不走这张公开作品闸的例外：本面 required，catalog snapshot 对 hidden 所有者仍 200 则放行；否则 404）。`viewer.can_review` 见 §3.2。

### 4.10 `createWorkEditProposal` · `POST /works/{work_id}/edit-proposals` · required · 201 `Location` + `EditProposal`

`Idempotency-Key` 必带。`Location`：`/api/v1/edit-proposals/{proposal_id}`。体 `{patch, note?}`。`patch` 空 → 422 `REQUIRED`。键必须 `catalog.work.` 前缀。自动合并：`state: merged` 且 `revision` 非 null，**不**跑 §3.8 的 NotifyRequested / 时间线。未合并才跑。

### 4.11 `listWorkEditProposals` · `GET /works/{work_id}/edit-proposals` · optional · 200 游标信封

Query：`state`（缺席 = `open`）、`cursor`、`limit`。G4 可见性。匿名 `viewer: null`。

### 4.12 `listWorkEditRevisions` · `GET /works/{work_id}/edit-revisions` · optional · 200 页码信封

Query：`page`、`limit`（默认 20）。生产每作最多 15 条。`viewer.can_revert`：有 token 且 schema 每个非锁定字段 `can_review`（旧 `canRevert`）；schema 失败 → false + WARN `galgame edit: revert projection failed`（失败关闭）。匿名 `viewer: null`。G4 可见性。

### 4.13 `getWorkEditRevisionDiff` · `GET /works/{work_id}/edit-revisions/diff` · optional · 200 `EditRevisionDiff`

Query：`from_seq`、`to_seq`，均 ≥1。缺席 / ＜1 → 400 `INVALID_PARAMETER`。找不到 seq → 404。G4 可见性。上游用 `GET /v2/catalog/revisions/{toId}?include=diff&diff_base={fromId}`。

### 4.14 `createWorkEditRevert` · `POST /works/{work_id}/edit-reverts` · required · 201 `Location` + `EditRevert`

`Idempotency-Key` 必带。体 `{to_seq, note?}`。`to_seq` ＜1 → 422。本地无 owner/moderator 闸（wave 178）。`Location` 在产生了提案时指向 `/api/v1/edit-proposals/{proposal_id}`。

### 4.15 `listMyEditProposals` · `GET /me/edit-proposals` · required · 200 游标信封

Query：`work_id`（可选）、`state`（可选，未知 400）、`cursor`、`limit`。上游 `GET /v2/me/proposals`。

### 4.16 `listEditProposals` · `GET /edit-proposals` · required · 200 游标信封

编辑队列。仅 cookie + `user.CanModerate()`。Bearer 403，0 次 `/v2/moderation/proposals`。Query：`state`（缺席 = `open`）、`cursor`、`limit`。

### 4.17 `getEditProposal` · `GET /edit-proposals/{proposal_id}` · required · 200 `EditProposal`

工作台。租户钉失败伪装 404。过门之后拉 snapshot + schema，填 `viewer.can_decide`。作品主人（本地 `creator_user_id`）与 moderator 可进。Bearer 的 `can_decide` 恒 false；决定写在 4.19，Bearer 403。

### 4.18 `createEditProposalAmendment` · `POST /edit-proposals/{proposal_id}/amendments` · required · 201 `EditAmendment`

`Idempotency-Key` 必带。体 `{set?, unset?, note?}`。两者都空 → 422。`set` 的键必须 `catalog.work.`。`Location`：`/api/v1/edit-proposals/{proposal_id}`。

### 4.19 `updateEditProposal` · `PATCH /edit-proposals/{proposal_id}` · required · 200 `EditProposal`

体 `{state, note?}`。`merged` / `declined`：cookie 审核者（本地只做租户钉；真值在 infra）。`withdrawn`：提案人。`declined` 的 note trim 后必填。响应重读。副作用 §3.8。Bearer 发 `merged`/`declined` → 403，0 次 decisions。

## 5. 预分配

### 5.1 迁移

**无。** G 号段 140–145 与 195 已用，146–159 空着，本轨不用。不改表：`galgame.creator_user_id` / `published` / `resource_update_time` 已有；`galgame_activity.wiki_pr_id` 继续写 catalog 提案 id（§8 O2）；`galgame_renumber_2026` 删表仍按 g-plan 留给后面某段。本段 HTTP 不改 `galgame.published`（078 粘性 SEO 旗；hidden/ban 的 unpublish 在 cron）。

### 5.2 错误码

不新增 kungal 码。复用：

- **`DUPLICATE_SUSPECTS`**（me，409）。infra 已注册（`apiv2/problem/registry.go:148`），type URI `https://developer.nextmoe.dev/problems/me/duplicate-suspects`，扩展 `suspects[]`（`id` + `display_name`）。**实现时写入本仓 `pkg/problem` 注册表**（目前还没有这一行），与 infra 逐字相同。
- **`INVALID_STATE_TRANSITION`**（me，409）。非法 claim / 提案转移、非 draft 删除、已决提案。
- **`ALREADY_EXISTS`**（me，409）。
- **`SCOPE_REQUIRED`**（platform，403）。缺 `catalog:edit`。
- **`PERMISSION_REQUIRED`**（moderation，403）。缺 `galgame.claim.review` / `CanModerate`；Bearer 审。
- **`ENTITY_MERGED`**（作品路径）。
- **`CONTENT_REJECTED`**（kungal，422）。

其余：`NOT_FOUND`、`ACCOUNT_BANNED`、`SERVICE_UNAVAILABLE`、`INTERNAL_ERROR`、`INVALID_PARAMETER`（`REQUIRED` / `INVALID_FORMAT` / `OUT_OF_RANGE` / `TOO_SHORT` / `TOO_LONG` / `TOO_MANY_ITEMS`）、`VALIDATION_FAILED`（`NOT_ALLOWED_VALUE` / `UNKNOWN_VALUE` / `REQUIRED` / `TOO_SHORT`）、`UNKNOWN_ENUM_VALUE`、`LIMIT_TOO_LARGE`、`INVALID_CURSOR`、`IDEMPOTENCY_KEY_REUSED`、`IDEMPOTENCY_REQUEST_IN_PROGRESS`。

`CLAIM_NOT_OWNED` / `TENANT_MISMATCH` / `DECISION_ALREADY_MADE` **不**引进论坛注册表：分别折成 404 与 `INVALID_STATE_TRANSITION`。上游 429 走 `SERVICE_UNAVAILABLE`，不是 `QUOTA_EXCEEDED`。

### 5.3 K 决定

| 编号 | 决定 |
|---|---|
| **K-G53** | 本轨是 BFF：用户 token 打 catalog `/v2/me/claims`、`/v2/moderation/claims`、`/v2/me/proposals`、`/v2/moderation/proposals`、`/v2/catalog/revisions`、snapshot / schema / vocabularies。论坛库不存 claim / 提案 / 修订 |
| **K-G54** | `work_id` = catalog work id = 投稿资源 `id`。`proposal_id` / `revision_id` 是十进制字符串；每作 `seq` 是整数。不翻译 id |
| **K-G55** | 认领与提案的状态迁移都是 `PATCH` `state`，见 §3.4 / §3.5。先例 `updateReviewItem`。响应重读 catalog |
| **K-G56** | 同名软门是注册过的 `409 DUPLICATE_SUSPECTS` + `suspects[]` + 体字段 `is_duplicate_confirmed` |
| **K-G57** | 向导是游标集合、无 `total`、`include_nsfw` 默认 false。我的投稿 / 审核队列 / 编辑队列是游标。修订是页码（生产每作 ≤15） |
| **K-G58** | 作品下提案 / 修订 / diff 复用 G4 `catalogWork`：hidden / 未知 404。租户钉 404 |
| **K-G59** | `viewer.can_*` 是 infra 投影。`can_review`（表单）= schema 有可审字段（#28）。`can_decide` = 工作台资格 ∧ patch 键都可审。`can_revert` = 非锁定字段都可审。匿名 `viewer: null`。Bearer 不含 staff |
| **K-G60** | 副作用见 §3.8。失败不回滚上游写（旧面如此），打具名 WARN/ERROR |
| **K-G61** | 无迁移。`wiki_pr_id` 继续写提案 id |
| **K-G62** | 缺 scope → `403 SCOPE_REQUIRED`。任何上游 429 → 503 + `Retry-After`。`USER_IDENTITY_REQUIRED` → 500。英文上游句不进 `detail` |
| **K-G63** | 投稿体是 catalog 字段形状（§3.6）。`released` 送达。`content_rating` 与 `is_nsfw` 两轴独立 |
| **K-G64** | 四个创建 POST 必带 `Idempotency-Key`。BFF 对 catalog 写发 `If-Match: *`，客户端不发 |
| **K-G65** | 审核队列走 `GET /v2/moderation/claims`；单条 claim 走 me / moderation GET；单条提案走 me / moderation GET。不用应用 key 作品搜索列 pending |

权限不新增。`galgame.claim.review` 沿用，经 `viewer.can_review`（投稿）下发。编辑队列继续用 moderator 角色，见 §8 O1。

## 6. 网页

切到生成的类型化客户端；手写 `UserClaimItem` / `GalgameEditProposalList` / bootstrap 形状改成生成物别名或删除。`legacy-fetch-baseline` 按删掉的调用点下调。id 全是字符串。按钮显隐读 `viewer.can_*`，停掉 `edit/Container.vue:58` 的 `proposer_uid !== userStore.id`（改读 `viewer.is_proposer`）。`DUPLICATE_SUSPECTS` 的 zh-CN 进 `problem.json`（实现本轨加码时一起）。`SCOPE_REQUIRED` / `SERVICE_UNAVAILABLE` / `INVALID_STATE_TRANSITION` 已有译文。

| 文件 | 改什么 |
|---|---|
| `edit/galgame/Footer.vue` | `POST /work-submissions`（201，幂等键把标题算进指纹）；体改 titles/intros/`is_nsfw`/`content_rating`/`is_duplicate_confirmed`/`release_date`；读 `has_banner_attached`；409 `DUPLICATE_SUSPECTS` 弹确认 |
| `validations/galgame.ts` | 标题 ≤500、简介 ≤50000、别名合计 100；去掉四语槽 |
| `edit/galgame/Mine.vue` | `GET /me/work-submissions?state=pending,declined,draft`；撤回 `PATCH {state:draft}`；重提 `PATCH {state:pending}`；删草稿 `DELETE` 204；`work_id` 字符串；读 `last_event.reason` |
| `useGalgameClaimList.ts` | 游标 `next_cursor`；`hasMore` = 有 `next_cursor` |
| `edit/galgame/Audited.vue` | `GET /me/work-submission-reviews` |
| `edit/galgame/Wizard.vue` | `GET /work-submission-candidates?q=&include_nsfw=`（从内容姿态来）；pending 另打 mine；无 `total`；href `/galgame/${work_summary.id}` |
| `pages/admin/submissions.vue` | `GET /work-submissions?state=pending`；决定 `PATCH {state:live\|declined\|hidden, note?}`；id 字符串 |
| `galgame/edit/Container.vue` | `GET …/edit-form`；`viewer.can_review`；列表 `GET …/edit-proposals` 用 `viewer.is_proposer` 过滤；提交 `POST …/edit-proposals` 幂等键；撤回 `PATCH /edit-proposals/{id} {state:withdrawn}` |
| `galgame/history/Container.vue` | 页码修订；diff `from_seq`/`to_seq`；回滚 `POST …/edit-reverts` 幂等键；`viewer.can_revert` |
| `activity/card/GalgameEdit.vue` | 同上 diff 查询名；limit 不超过 100 |
| `galgame-edit/mine/Container.vue` | `GET /me/edit-proposals`；`work_summary.display_name`；id 字符串 |
| `galgame-edit/review/Container.vue` | `GET /edit-proposals`；停 `useRole().canModerate` 做唯一闸，读 403 |
| `galgame-edit/review/Detail.vue` | `GET /edit-proposals/{id}`；`viewer.can_decide`；amend 201；merge/decline `PATCH {state, note?}` |
| `pages/galgame/[id]/edit.vue` / `history.vue` | 随组件；history 保持无 auth |
| `pages/edit/galgame/{create,mine,audited,publish}.vue` | 随组件 |
| `docs/proj/app-direct-api.md` | 前瞻改上表路径 |

## 7. 变异题（先于实现提交）

种子：一部 catalog 作品带 draft claim（调用者所有）；一部 pending；一部 declined；一部 hidden（从 live 封禁，有 prior）；一部 hidden（无事件历史）；一部别人的 pending；一部 G4 hidden 作品带 open 提案与修订；一个未认领 NSFW 作品（向导）；一个同名 live 作品（软门）；一个封禁用户当提案人；Bearer 会话持有 moderator / `galgame.claim.review`；`catalog:edit` 不足的 token；catalog 用户面 429；自动合并的提案；自合并。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | `GET /work-submissions/{id}` 对另一用户的 pending 回 200 或 403 | 404 `NOT_FOUND`，body 不出现 `pending`、不出现 display_name |
| 2 | Bearer 调用者 `PATCH` 投稿 `state=live` 或提案 `state=merged` 200，或 `viewer.can_decide=true` | 403 `PERMISSION_REQUIRED`；catalog moderation 0 次调用；`can_decide` / 投稿 `can_review` 为 false |
| 3 | 任一 GET（mine / 队列 / 向导 / 表单 / 提案列表 / 修订 / diff / 工作台）写入 claim、提案、本地 `galgame`、`galgame_activity` | 调用前后行数与状态相同 |
| 4 | `PATCH` 投稿 `state=live` 从 `declined`（infra 禁止）回 200，或不映射 409 | `409 INVALID_STATE_TRANSITION`；`detail` 含当前 `declined`；catalog 状态未变 |
| 5 | `PATCH` 投稿或提案 `state=declined` 且 `note` 为 `""` / `"   "` 回 200 | `422 VALIDATION_FAILED` `REQUIRED` pointer `/note` |
| 6 | 向导在 `include_nsfw=false`（默认）把 NSFW 未认领作品放进 `items` | 该 `work_id` 不在 `items`；请求打 catalog 时带了展示轴闸，不是无条件 `nsfw=true` |
| 7 | 向导 200 带 `total`（过滤前）或 `total_relation` | 响应无 `total`；F9 绿 |
| 8 | 匿名 `GET /works/{hidden}/edit-proposals` 或 `…/edit-revisions` 或 diff 200 | 404；body 无 patch、无 seq |
| 9 | 封禁提案人的 `proposer.name` 为非 null 字符串 | `proposer` 是删除用户 ref：`name: null`，`avatar: null` |
| 10 | 审核 `PATCH state=live` 的 200 `state` 来自本地写死而不是 catalog 重读（夹具 catalog 回 `hidden`） | 200 `state` 等于重读值，不是 `"live"` |
| 11 | `DELETE /work-submissions/{id}` 对 `pending` 或 `live` 204 | 409 或 404；本地 `galgame` 行仍在；catalog 仍是原状态 |
| 12 | 同名 mint 409 的 `code` 是 `ALREADY_EXISTS` / `INVALID_STATE_TRANSITION` / 无 code，或 `detail` 含 `confirm_duplicates` | `409 DUPLICATE_SUSPECTS`；`suspects` 非空；`is_duplicate_confirmed=true` 才 mint |
| 13 | 上游 409/422 的英文 `detail`（如 `re-send with confirm_duplicates=true`）出现在响应 `detail` | `detail` 是论坛自己的英文短句；`code` 是注册表里的 |
| 14 | 创建体收 `name_zh_cn` / `intro_en_us` 并写成 titles | 四语槽不在 schema；只认 `titles` / `intros` |
| 15 | 有 `release_date` 的 mint 打 catalog 时不带 `released`（旧丢掉） | 上游 JSON 含 `released`；catalog_release 行存在 |
| 16 | `is_nsfw: true` 且不传 `content_rating` 被写成 `r18`，或 `content_rating: r18` 且 `is_nsfw: false` 被改成 true | `422` 缺 `content_rating`；两轴按原值分别写入 `catalog.work.display_nsfw` 与 `catalog.work.content_rating` |
| 17 | 合并者 == 提案人仍 `Award` | 0 次 moemoepoint；键 `galgame_edit_merged:<id>` 未写 |
| 18 | `POST /work-submissions` 或 `POST …/edit-proposals` 或 revert 或 amend 不带 `Idempotency-Key` 201 | `400 INVALID_PARAMETER` `REQUIRED` header `Idempotency-Key`；上游 0 次 POST |
| 19 | 带资源的 draft `DELETE` 把本地 `galgame` 行和 `galgame_resource` 一起删掉 | 204；本地行与资源仍在；catalog claim 已删 |
| 20 | 向导 `include_nsfw` 缺席时当 true（旧 `OpenPopulation`） | 默认 false；SFW 夹具看不到 NSFW 标题 |
| 21 | unban 的 200 `state` 恒 `live`（旧本地写死），夹具 prior 是 `draft` | 200 `state: "draft"` |
| 22 | `GET /work-submissions?state=pending` 打应用 key `/v2/catalog/works` | 上游是带用户 token 的 `GET /v2/moderation/claims` |

## 8. 开放问题

| # | 事实 | 裁决 |
|---|---|---|
| O1 | 投稿队列门是权限键 `galgame.claim.review`，编辑队列门是 moderator 角色（普查 §5.2 / E21）。override 只动其中一个时两张后台表分叉 | **本轨不统一**，按代码原样写。交给编排者 |
| O2 | `galgame_activity.wiki_pr_id` 自 cutover 起存的是 catalog 提案 id（472 行，79–1446） | **继续写，不迁移**。列名是债 |
| O3 | infra PATCH/决定强制 If-Match，缺席 428 | BFF 发 `If-Match: *`（旧面如此）。客户端不发。要不要改成真 ETag 交给后轨 |
| O4 | infra 允许 claim 所有者从 draft `PATCH state=live` 自发布 | 本轨不暴露；trusted mint 仍可直达 live |
| O5 | 公开 `GET /v2/catalog/proposals` 不发 patch；编辑页需要别人的待审 patch | 作品下列表对 **可见** 作品仍发 `patch`；hidden 404。是否进一步收到工作台，交给编排者 |
| O6 | `pending` 生产 0 行，但历史 74 次进入 | 队列与 `state=pending` 仍建模 |
| O7 | 通过投稿 +3 在 cron 不在 HTTP | 保持。cron 停了接口仍 200（E13） |
| O8 | 网页 Zod 要求至少一个 intro，后端不要求 | 保持：API 不要求 intro |
| O9 | `DeleteLocalDraft` 对带资源的行返回 nil | v1 204 且本地行保留，见 4.4。与普查「500」口径不同，以代码为准 |
