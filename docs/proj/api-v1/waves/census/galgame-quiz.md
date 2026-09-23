# 普查 · G4 galgame 题库（QuizHandler，13 条，含 picker）

> 只读普查。写于 2026-09-22，基于 `api-v1/g-galgame` @ `ef0c9fb3`（= master）。没有连库、没有起服务，全部结论来自代码。
> 需要生产库才能确认的数字集中在 §9。

## 0. 范围与总览

本段共 **13 条**，全部是 `QuizHandler`：12 条挂在 `/api/galgame-quiz/**`，1 条是 `/api/galgame/search/picker`（通用作品选择器）。原本同段普查的 `CommunityCommentHandler` 7 条已于 2026-09-22 按用户裁决划归 RC 轨，拆到 [galgame-comments.md](galgame-comments.md)；下文 §7–§10 里偶尔提到评论的，以那份为准。

| # | 方法 | 路径 | handler | 档位 |
|---|---|---|---|---|
| 1 | GET | `/api/galgame-quiz/all` | `galgame/handler/quiz_handler.go:25` `GetAllQuizzes` | optional |
| 2 | GET | `/api/galgame-quiz/:id` | `quiz_handler.go:74` `GetQuizPlay` | optional |
| 3 | GET | `/api/galgame-quiz/:id/answers` | `quiz_handler.go:198` `GetQuizAnswers` | optional |
| 4 | GET | `/api/galgame-quiz/:id/edit` | `quiz_handler.go:180` `GetQuizForEdit` | required |
| 5 | GET | `/api/galgame-quiz/mine/answered` | `quiz_handler.go:39` `GetMyAnswered` | required |
| 6 | GET | `/api/galgame-quiz/mine/favorites` | `quiz_handler.go:55` `GetMyFavorites` | required |
| 7 | GET | `/api/galgame/search/picker` | `quiz_handler.go:63` `SearchGalgames` | public |
| 8 | POST | `/api/galgame-quiz` | `quiz_handler.go:86` `CreateQuiz` | required |
| 9 | PUT | `/api/galgame-quiz/:id` | `quiz_handler.go:164` `UpdateQuiz` | required |
| 10 | DELETE | `/api/galgame-quiz/:id` | `quiz_handler.go:134` `DeleteQuiz` | required |
| 11 | POST | `/api/galgame-quiz/:id/answer` | `quiz_handler.go:102` `AnswerQuiz` | required |
| 12 | PUT | `/api/galgame-quiz/:id/favorite` | `quiz_handler.go:149` `ToggleQuizFavorite` | required |
| 13 | PUT | `/api/galgame-quiz/:id/quality` | `quiz_handler.go:118` `RateQuizQuality` | required |

路由注册：`internal/app/router.go:160`（picker，public 组）、`:189-193`（answers，显式 `OptionalAuth`）、`:206-207`（all / play，`optAuth`）、`:222` `RegisterReads`（comments 读面，必须在 auth 边界之前，见 `:220-221` 注释）、`:305` `RegisterWrites`、`:321-329`（题库写面）。中间件链见 `testdata/routes.golden:7-8,17,79-85,104-105,124,232-234,238,244,298-300,311-312`：全部带 `cors.New → middleware.NamePreference → ContentStance`；public 的 picker 到此为止，optional 再加 `OptionalAuth`，required 再加 `Auth`。本段**没有** `RequirePermission` / `RequireModerator`。

**不在本段**：`ResourceCommentHandler` 的 15 条（含 `GET/POST/DELETE /api/galgame-quiz/:id/comments*`，`routes.golden:8,81,234`）。题目讨论墙的列表/发表/删除走那一轨。本段的 `PUT/DELETE /api/galgame/comments/:postId` 与 `PUT …/like`、`POST …/flag` 却是**所有社区评论族共用的编辑/点赞/举报面**——网页 `communityComment.ts:53-55,76,99,122,145,168` 把 rating / website / toolset / resource / quiz 的 `editUrl` 都指到这里，`Like.vue:30` 与 `FlagModal.vue:38` 不分锚点一律打这两条。普查按「这条路由实际干什么」写，并标明跨族调用。

**信封**：全部走旧信封 `pkg/response`。成功 `200 {code:0, message:"成功", data:…}`；`OKMessage` 成功无 `data`（删除题目、切换收藏、删除评论、举报）。错误 `pkg/response.Error` → `{code, message}` + HTTP 状态。本段体码实际出现的是 `205`（401）、`233`（400/403/404/422/429/500/503 混用）、`234`（403 封禁）。零个 problem+json、零个 `object` 判别字段、零个字符串 id、零个 `viewer` 块。

**v1 现状**：`/api/v1` 下题库与 galgame 社区评论一条都没有迁。话题评论的 v1 已在 W5a（`docs/proj/api-v1/waves/w5a-comments.md`），对比见 §6。

**身份口径**：`gid` / `galgame_id` / 题库关联的 `galgame_ids[]` 都是论坛自己的作品 id。catalog work id 是另一套整数，任务书记录约 10,289 个数值撞车。picker 经 `CatalogItemGID`（`client/catalog_wire.go:388`）取值：有 kungal/galgame_wiki claim 且 `site_work_id > 0` 时用论坛 gid，否则退回 `it.ID`（catalog work id）。见 §4。

---

## 1. 题库列表与详情读面

### 1.1 `GET /api/galgame-quiz/all`（#1）

链路：`QuizHandler.GetAllQuizzes`（`quiz_handler.go:25`）→ `QuizService.GetAllQuizzes`（`service/quiz_service.go:62`）→ `QuizRepository.ListPaginated`（`repository/quiz_repo.go:45`）→ `hydrateCards`（`quiz_service.go:94`）。

**调用方**

| 位置 | 用途 |
|---|---|
| `apps/web/app/components/galgame/quiz/Container.vue:32,35` | 题库首页「全部」tab，SSR `useKunFetch`，`page/limit/sort_field/sort_order/category/type/difficulty` |
| `apps/web/app/components/galgame/quiz/GalgamePanel.vue:14-16` | 作品页「本作题库」tab，`galgame_id` + `page` + `limit=12` |
| `apps/web/app/components/user/Quiz.vue:36-40` | 用户页「出题」tab，`user_id` + `page` + `limit=50` |

页面入口：`pages/galgame-quiz/index.vue:12` 渲染 `GalgameQuizContainer`；`pages/user/[id]/quiz.vue:19` 渲染 `UserQuiz`；`components/galgame/Galgame.vue:184-185` 作品页 quiz tab。`apps/web/server/**` 搜 `galgame-quiz` / `galgame-quiz/all` 零命中。`/home/kun/Desktop/code/website/kungal-apps` 搜 `galgame-quiz`、`galgame_quiz`、`search/picker` 零命中（`packages/kungal_api/lib/kungal_api.dart` 是空 library；`apps/kungal/lib/features/galgame/presentation/galgame_screen.dart:8` 是占位 `Text('galgame')`）。`docs/proj/app-direct-api.md` 未列出本段任何路径。

**请求**（`dto.QuizListRequest`，`dto/quiz_dto.go:5`）

| 位置 | 名字 | 约束 | 真实含义 |
|---|---|---|---|
| query | `page` | `validate:"min=1"` | 页码。零值 0 过不了 `min=1`，**不填就是 400** |
| query | `limit` | `min=1,max=50` | 同上，事实必填 |
| query | `sort_field` | 无 `oneof` | 见下 |
| query | `sort_order` | `omitempty,oneof=asc desc` | 空则服务层改成 `"desc"`（`quiz_service.go:69-71`） |
| query | `category` | 无 `oneof` | 空或 `"all"` 不过滤；其它值原样 `q.category = ?` |
| query | `type` | 无 `oneof` | 同上，对 `q.type` |
| query | `difficulty` | `omitempty,min=1,max=10` | 0 / 缺席 = 不过滤（前端「全部难度」发 `0`，`_filters.ts:35`） |
| query | `galgame_id` | 无 | `>0` 时 `EXISTS (galgame_quiz_galgame … galgame_id = ?)`（`quiz_repo.go:56-60`）。这是论坛 gid |
| query | `user_id` | 无 | `>0` 时 `q.user_id = ?` |

`sort_field` 在 repo 的 `switch`（`quiz_repo.go:70-86`）：`view` / `view_1d` / `view_7d` / `view_30d` / `difficulty` / `answer_count` / `update_time`（→ `q.status_update_time`）。**未知值静默回落到 `q.created`**。前端词表（`constants/galgame-quiz.ts:85-94`、`_filters.ts:42-50`）多一个 `time`，标签是「创建时间」——后端没有这个 case，于是 `time` 也落到 `q.created`，碰巧对。`view_1d` 是相关子查询 `galgame_quiz_view_daily` 当天之和（`quiz_repo.go:73-75`）。

排序**没有 `id` tie-breaker**。`ORDER BY <col> asc|desc` 完事。

`Count(&total)` 与随后的 `Select.Scan` 走同一条 `query` 变量（`quiz_repo.go:67-93`），返回值都不接。SQL 失败时 `rows` 空、`total` 0，handler 仍 200。

**响应**（`dto.QuizListPage`，`quiz_dto.go:122`）：`{quiz_data: QuizCard[], total}`。不是 01 §4 的 `items`，也不是 `Paginated` 的 `{items,total}`。

`QuizCard`（`quiz_dto.go:107`，内嵌 `QuizStats`）：`id, user{id,name,avatar}, category, type, difficulty, spoiler_level, question_html, view, answer_count, correct_count, favorite_count, quality_average, quality_count, comment_count, created, updated, status_update_time, my_status`。

- **没有 `content`，没有解析，没有选项。** 题干只下发 `question_html` = `markdown.RenderQuestionPlain`（`quiz_mapper.go:39`，HTML escape + `||剧透||` 打码，**不跑 Markdown**，`infrastructure/markdown/markdown.go:157`）。答案键不在这个面上。
- `my_status` 由 `quizViewerStatus`（`quiz_service.go:123`）从 `galgame_quiz_answer` 算出：`unanswered` / `author` / `answered`（`is_correct IS NULL`，问答题）/ `correct` / `incorrect`。匿名 `viewerID=0` 时 `FindViewerAnswers` 直接空 map（`quiz_repo.go:251`），一律 `unanswered`。
- `quality_average` 是 `round(sum/count, 1)`（`quiz_mapper.go:12`）；`count=0` 时为 `0`。
- `user` 来自 `userClient.Hydrate`（`quiz_service.go:106`）。

**答案键不变量**：本面不下发 `content` / `explanation` / `submitted`。成立。

**错误**：query 校验失败 → `400/233`（`ParseQueryAndValidate`，`pkg/utils/validate.go:66`）。服务层恒 `nil` error（`quiz_service.go:83`）。不存在「题目不存在」——空页是 `{quiz_data:[], total:0}`。

**鉴权**：`OptionalAuth`。`optionalUID`（`handler/resource_handler.go:82`，同包）取会话用户，没有就是 0。staff 能力未用。Bearer 与 cookie 一样只用来填 `my_status`。

**可见性**

- **不查**关联作品是否 hidden / banned / NSFW / `galgame.published`。`isSFW` 从 handler 传进服务（`quiz_handler.go:31`）之后**从未再被读**（`quiz_service.go:62-84` 的签名有、函数体没有）。SFW 读者的题库列表与 NSFW 读者字节级相同。
- 封禁作者：`hydrateCards` 对 `!userclient.IsRenderable(u)` 的行 `continue`（`quiz_service.go:112-114`）。`IsRenderable` 是 `u.Status == 0`（`pkg/userclient/hydrate.go:48`）。
- `Hydrate` 丢掉 OAuth 错误（`hydrate.go:37` `users, _ := c.Users`），缺失 id 填 `Placeholder`（`userclient.go:211`，`Name:"已注销用户"`，`Status` 零值 0）。于是 OAuth 故障时封禁作者的题目**仍出现**，名字变成「已注销用户」。fail-open。
- `total` 是 SQL `COUNT(*)`（过滤后的全部行），`quiz_data` 在 Go 里丢掉不可渲染作者。分页器按 `total` 画，某一页可能短一截。

**偏好 cookie**：本 handler 调用 `utils.IsSFW(c)`（`quiz_handler.go:31`），而 `IsSFW`（`pkg/utils/settings.go:42`）在没有 `content.FromCtx` 时读 `KUNGalgameSettings` / 否则读 `ContentStance`。本路径 `isSFW` 随后被丢弃，所以**结果集现在不受它影响**；但调用点还在，v1 不能假装这个面已经跟 cookie 脱钩——要么删掉，要么变成显式 `include_nsfw`。整组 `/api` 还挂着 `NamePreference`（`router.go:47`）；本列表不下发作品名，所以本条不受原名偏好影响。

**测试**：没有针对本路由的 handler/service 测试。`ListPaginated` 零测试。

---

### 1.2 `GET /api/galgame-quiz/:id`（#2，作答页 / SSR）

链路：`GetQuizPlay`（`quiz_handler.go:74`）→ `QuizService.GetQuizPlay`（`quiz_service.go:138`）。

**调用方**

| 位置 | 用途 |
|---|---|
| `apps/web/app/pages/galgame-quiz/[id].vue:8-9` | **SSR**：`useKunFetch<GalgameQuizPlay>(\`/galgame-quiz/${route.params.id}\`)`，SEO 用题干/关联作品名/banner |
| `apps/web/app/components/galgame/quiz/Play.vue:62-64` | 作答后若 `hide_galgame` 再拉一次，只补 `galgames` |
| `Play.vue:97-100` | 编辑保存后整页重载 |

Nitro / Flutter：零调用方（同 §1.1 的搜索词，外加字面 `/galgame-quiz/`）。

**请求**：路径 `:id`，`strconv.Atoi`（`quiz_handler.go:75`）。非整数 → `400/233 "无效的题目 ID"`。`id<=0` 仍往下走，`FindByID` 找不到再 404。无 query、无 body。

**响应**（`dto.QuizPlay`，`quiz_dto.go:127`）：

`id, user, category, type, difficulty, spoiler_level, question, question_html, description_html, content, view, answer_count, correct_count, favorite_count, quality_average, quality_count, comment_count, created, updated, hide_galgame, galgames[], is_author, is_favorited, my_answer`。

`content` 在未作答时是 `stripQuizContent`（`quiz_content.go:120`）：

| `type` | 未作答下发 |
|---|---|
| `single` | `{options}`（去掉 `answer`） |
| `multiple` | `{options}`（去掉 `answers`） |
| `fill` | `{blank_count}`（去掉每个空的 `accepted`） |
| `judge` / `essay` / 其它 | `{}` |

`json.Unmarshal` 失败被 `_ =` 丢掉，结果是空 `options` / `blank_count=0`。

`my_answer` 仅当 `currentUserID != 0` 且 `FindAnswer` 命中时出现（`quiz_service.go:156-167`）。作者在创建时被插入 `role='author'` 行（`quiz_service.go:240-244`），所以**作者打开作答页就会带 `my_answer`**，且 `Answer: quiz.Content` 是**含答案键的完整 JSONB**。真正作答过的人同样在 `my_answer.answer` 里拿到完整 `content`，外加 `submitted` / `is_correct` / `explanation` / `quality_rating`。未作答（含匿名）`my_answer` 为 null，`content` 是剥离后的。

`galgames`：`hide_galgame && my_answer == nil` 时下发 `[]`（`quiz_service.go:169-172`）。一旦有 answer 行（作者或作答者）就 `galgamesDetailFor`。`fetchDetailBriefs(..., false)`（`quiz_service.go:659`）——**第三个参数 `isSFW` 写死 `false`**，走 `GetBatchDetailPublic` → `contentLimitFor(false) = "all"`（`client.go:236-238`，`catalog_face.go:76-80`）。SFW 读者作答后（以及作者预览）会看到 NSFW 作品的 banner / 名字。`CatalogRowsByGIDs` 会跳过 `claim.state == hidden`（`catalog_face.go:314` + `catalog_wire.go:334`），hidden 作品从数组里消失，id 对不上的也消失（catalog 超时则 `fetchDetailBriefs` 丢错，回空 map，`quiz_service.go:752`）。

`is_author` = 当前用户 == `quiz.UserID`。`is_favorited` = `FindQuizFavorite`（`quiz_repo.go:176`）。匿名都是 false。

`description_html` = `markdown.Render`（完整 Markdown，`quiz_service.go:183`）。`question` 是源文，`question_html` 仍是 `RenderQuestionPlain`。

**答案键不变量**：未作答剥离成立；作者因 author 行被当成「已参与」拿到完整键——出题者看自己的题，语义上说得通，但和「作答前不下发」的字面不变量对作者不成立。SSR 载荷就是这个 JSON：匿名爬虫拿不到键；已登录已作答/作者的 SSR HTML 里带完整 `my_answer.answer`。

**副作用**：`IncrementView`（`quiz_repo.go:112`）在 `IsRenderable` 通过之后 `go func()` 里 `view + 1` 并 `viewstats.BumpDaily`。每个 SSR、每次刷新、作者自己看，都加浏览。goroutine 错误全丢。GET 非幂等。

**错误**

| 触发 | 状态 | 体 |
|---|---|---|
| `:id` 非整数 | 400 | `233 "无效的题目 ID"` |
| `FindByID` 失败（含 SQL 错，`quiz_repo.go:30-32` 任何 error 都当没有） | 404 | `233 "题目不存在"` |
| 作者 `!IsRenderable` | 404 | `233 "题目不存在"` |

**鉴权**：optional。staff 未用。

**可见性**：只挡封禁作者（且 OAuth 错时 fail-open，见下）。**不查**关联作品 NSFW / published / 本地 `galgame` 行是否存在。题目本身没有 hidden/draft 状态——`galgame_quiz` 没有 status 列（`model/quiz.go:8`，迁移 `045_create_galgame_quiz.up.sql:27`），发表即公开。

`author, _, _ := s.userClient.User(...)`（`quiz_service.go:146`）丢掉 `found` 和 `error`。OAuth 故障或用户不存在时 `author` 是零值 `User{Status:0}`，`IsRenderable` 为 true，题目照常返回，作者 `user` 是 `{id:0,name:"",avatar:""}`。与列表的 Placeholder 还不一样：这里连「已注销用户」都没有。

**测试**：无。`stripQuizContent` / `GetQuizPlay` 零测试。

---

### 1.3 `GET /api/galgame-quiz/:id/answers`（#3）

链路：`GetQuizAnswers`（`quiz_handler.go:198`）→ `QuizService.GetQuizAnswers`（`quiz_service.go:601`）→ `FindQuizAnswerers`（`quiz_repo.go:266`，`role='answerer'`，`ORDER BY created DESC`，`limit=100`）。

注册在 `router.go:189-193`，`optAuth` 组之前，自己挂了 `OptionalAuth`。档位与 play 相同。

**调用方**：`apps/web/app/components/galgame/quiz/DetailPanel.vue:55-60`，`onMounted` **无条件** `kunFetch`。作答页底部「查看详情」面板，未作答访客也会打这条。Nitro / Flutter 零调用方。

**请求**：路径 `:id`，Atoi 失败 400。handler **不把 `:id` 当 quiz 存在性检查**——服务层没有 `FindByID`。不存在的 id → 空数组 200。

**响应**：`[]QuizAnswererRecord`（`quiz_dto.go:59`）：`user, submitted?, is_correct, created`。handler 直接 `response.OK`（`quiz_handler.go:207`），服务层不回 error。

`submitted` 仅当 `FindAnswer(quizID, viewerID)` 命中时写入（`quiz_service.go:608,626-628`）。作者的 author 行也算命中，所以**作者能看到别人的作答原文**（填空/问答的自由文本）。未作答访客 `submitted` omitempty 缺席。

**`is_correct` 无条件下发。** 未作答、匿名，都能看见每条记录是「正确」还是「错误」。`DetailPanel.vue:179-194` 正是用这个画「正确/错误」chip。这是作答前泄漏判分结果。完整选项键仍不在这个面上。

`FindQuizAnswerers` 没有 `id` tie-breaker，截断 100 条，没有 `next_cursor`，没有 total。超过 100 的作答者永远看不到。`Scan` 错误丢掉。

封禁作答者：`Hydrate` + `IsRenderable` skip（`quiz_service.go:618-619`），同样 fail-open Placeholder。**不查题目作者是否可渲染**——封禁作者的题目，play 回 404，本面仍列出作答者。

**答案键不变量**：键本身没有下发；判分结果（`is_correct`）对未作答者下发。

**测试**：无。

---

### 1.4 `GET /api/galgame-quiz/:id/edit`（#4）

链路：`GetQuizForEdit`（`quiz_handler.go:180`）→ `QuizService.GetQuizForEdit`（`quiz_service.go:568`）。`Auth` 组（`router.go:328`）。

**调用方**：`Play.vue:88-90` 点「编辑」时 `kunFetch<QuizEditData>(\`/galgame-quiz/${id}/edit\`)`。`pages/edit/galgame/quiz.vue` 是**空白出题表**，不打这条。Nitro / Flutter 零。

**请求**：路径 `:id`，Atoi 失败 400。

**响应**（`dto.QuizEditData`，`quiz_dto.go:44`）：`id, galgame_ids, hide_galgame, category, type, difficulty, spoiler_level, question, description, content, explanation, galgames[]`。`content` 是库里的完整 JSONB（含答案键）。`description` / `question` / `explanation` 是源文。`galgames` 是 `QuizGalgameBrief`：`id, content_limit, name`（`quiz_dto.go:80`），经 `GetBatch`（`quiz_service.go:687`，`content_limit="all"`，`client.go:251-252`），**不按 SFW 过滤**。

**答案键不变量**：本面故意下发完整键。靠「作者或 `perm.QuizEditAny`」挡。`user.Can(perm.QuizEditAny)`（`quiz_handler.go:190`）走 `UserInfo.Can`（`middleware/auth.go:61`）：`viaBearer` 恒 false。Bearer 不能靠 staff 编别人的题；作者比的是 `quiz.UserID != userID`（`quiz_service.go:575`），自己的题 Bearer 也能拉编辑数据。

**错误**：不存在 404 `"题目不存在"`；非作者且无 `quiz.edit_any` → `403/233 "没有编辑该题目的权限"`。

**可见性**：不查作者 `IsRenderable`。版主可以编辑封禁用户的题目。不查关联作品 hidden（`GetBatch`/`CatalogRowsByGIDs` 仍 skip hidden claim，`catalog_face.go:314`）。

**测试**：无。

---

### 1.5 `GET /api/galgame-quiz/mine/answered`（#5）

链路：`GetMyAnswered`（`quiz_handler.go:39`）→ `QuizService.GetMyAnswered`（`quiz_service.go:86`）→ `ListAnsweredByUser`（`quiz_repo.go:96`，`JOIN galgame_quiz_answer … role='answerer'`，`ORDER BY a.created DESC`）。

**调用方**：`Container.vue:31`（题库页「我的答题」tab）；`user/Quiz.vue:36`（本人资料「答题」tab；`isOwner` 为假时强制回 `publish`，`user/Quiz.vue:14-19`，避免在别人主页画出自己的作答史）。Nitro / Flutter 零。

**请求**：仍 parse 整个 `QuizListRequest`，但服务只取 `Page`/`Limit`（`quiz_handler.go:48`）。URL 上的 `category`/`type`/`sort_field`/`galgame_id` **静默忽略**。题库页切到「我的答题」时 `Container.vue:35-45` 仍把筛选 query 带上。

排序无 `id` tie-breaker。`Count`/`Scan` 丢错。卡片形状与 #1 相同，`hydrateCards(..., userID)` 填 `my_status`。答案键不下发。

**鉴权**：required。`user.ID` 强制，看不了别人的作答史（资料页靠前端 `isOwner` 藏 tab，API 本身只认会话）。

**测试**：无。

---

### 1.6 `GET /api/galgame-quiz/mine/favorites`（#6）

链路：`GetMyFavorites`（`quiz_handler.go:55`）→ `QuizService.GetMyFavorites`（`quiz_service.go:595`）→ `FindFavoritedQuizIDs`（`quiz_repo.go:187`，`Pluck("quiz_id")`）。

**调用方**：`composables/useMyQuizInteractions.ts:9-10`。实际使用者只有 `components/activity/card/Quiz.vue:20-21,50-55`（动态流卡片上的收藏按钮要一份「我收藏了哪些 id」）。题库作答页的 `FavoriteToggle` 不走这条，用 play 下发的 `is_favorited`。Nitro / Flutter 零。

**请求**：无。**响应**：`{favorited: number[]}`（`quiz_handler.go:60` `fiber.Map`）。无分页、无上限、无 `object`。`Pluck` 丢错 → 空数组。

**鉴权**：required。不查题目是否还在（FK `ON DELETE CASCADE`，`048_add_galgame_quiz_favorite.up.sql:9`，删题会带走收藏行）。

**测试**：无。

---

## 2. 题库写面：出题 / 编辑 / 删除

出题没有创作者角色门、没有审核队列（迁移 `045_create_galgame_quiz.up.sql:15-16` 原文「MVP has NO review gate」）。登录即可。trust `gate.CheckService.Decision` 在创建/编辑跑全文（题干+描述+解析+选项/填空/问答参考，`quiz_service.go:50-52,207,476`）；`deny` → `422/233 "内容包含违禁词，无法发布"`（`trust/gate/compose.go:9`）。`hold` 只打日志，题目照样落库，再 `scan.ScanBg(SubjectKindGalgameQuiz, …)`。

`galgame_ids` **不校验**这些整数是不是论坛 gid、是不是存在、是不是 published、是不是 hidden。`SetQuizGalgames`（`quiz_repo.go:221`）只跳过 `<=0` 与重复。可以写入任意正整数，包括 catalog work id。

创建/编辑/答题/删除/收藏的萌萌点都走 `InteractionHelpers.AdjustMoemoepoint`（`service/interaction.go:14`）→ `moemoepoint.Award`（goroutine、best-effort、`KeyNonce` 每次新键）。本地 `kungal_user_state.moemoepoint` 只在 OAuth 成功回写时更新（C3 缓存列）。

常量和网页文案：`QuizCreateReward = 2`，`QuizCorrectReward = 0`（`internal/constants/moemoepoint.go:22-23`）。答对**加不了分**。题库首页文案仍写「答对可获得萌萌点」（`Container.vue:93`，`pages/galgame-quiz/index.vue:5`）。出题表写「出题即得 2 萌萌点」（`Form.vue:252`）。

### 2.1 `POST /api/galgame-quiz`（#8）

**调用方**：`Form.vue:230` `kunFetch<GalgameQuizCard>('/galgame-quiz', {method:'POST', body})`。`pages/edit/galgame/quiz.vue`（auth middleware）和题库页/作品页的 `GalgameQuizPublish` 都渲染这个表。Nitro / Flutter 零。

**请求**（`dto.CreateQuizRequest`，`quiz_dto.go:17`）：

| 字段 | 约束 | 含义 |
|---|---|---|
| `galgame_ids` | `omitempty,dive,min=1` | 论坛 gid 数组，可空 |
| `hide_galgame` | bool | 作答前隐藏关联作品 |
| `category` | `required,oneof=plot character system music voice company trivia other` | 库 CHECK 同词表（`045:32`） |
| `type` | `required,oneof=single multiple judge fill essay` | 库 CHECK 同词表。前端出题 UI 只启用 single/multiple/judge（`constants/galgame-quiz.ts:25-29`，`Form.vue:284`「填空、问答即将实装」）；**API 仍接受 fill/essay** |
| `difficulty` | `required,min=1,max=10` | |
| `spoiler_level` | `omitempty,oneof=none portion serious` | 空 → 服务层 `"none"`（`quiz_service.go:214-217`） |
| `question` | `required,min=1,max=200` | 题干。前端 textarea `:maxlength="2000"`（`Form.vue:305`），zod 才是 200（`validations/galgame-quiz.ts:18-21`） |
| `description` | `max=20000` | Markdown 源，`NormalizeStoredContent` |
| `content` | `required` | 类型载荷，见下 |
| `explanation` | `max=2000` | 作答后才给看的解析 |

`content` 校验在 `validateQuizContent`（`quiz_content.go:59`），未知 `type` 理论上走不到（DTO oneof 已挡）：

| type | JSON | 规则 |
|---|---|---|
| single | `{options: string[], answer: int}` | ≥2 选项，`answer` 在范围内 |
| multiple | `{options, answers: int[]}` | ≥2 选项，≥1 个答案，每项在范围内；**不查重复** |
| judge | `{answer: bool}` | 解得出来就行 |
| fill | `{blanks: [{accepted: string[]}]}` | ≥1 空，每空至少一个非空白 `accepted` |
| essay | `{reference: string}` | trim 后非空 |

格式错 / 规则破 → `400/233` 中文（「题目内容格式错误」「单选题至少需要 2 个选项」…）。

事务（`quiz_service.go:233-248`）：插 `galgame_quiz`、写 `galgame_quiz_galgame`、插 `galgame_quiz_answer{role:author}`、`AdjustMoemoepoint(+2, content_approved, ref=galgame_quiz:<id>)`。失败 → `500/233 "创建题目失败"`。**没有 `Idempotency-Key`**（K12）。弱网重试 = 两道题 + 两次 +2。成功 200（不是 201），body 是 `CreatedQuiz`（`quiz_dto.go:157`）：卡片形，**无 `content`、无 `my_status`、无 `status_update_time`**，`QuizStats` 全 0。`user` 再次 `User(..., _)` 丢错。

触发器 `trg_feed_galgame_quiz`（迁移 051）写 `feed_activity` `GALGAME_QUIZ_CREATION`。

**鉴权**：required。不查 `perm`。Bearer 可以出题（用自己的 user id）。

**测试**：`write_moderation_test.go:62` `TestQuizAuthoringModerationText` 只测拼进 trust 的文本，不打 HTTP、不落库。

---

### 2.2 `PUT /api/galgame-quiz/:id`（#9）

**调用方**：`Form.vue:209-213`，body 另带 `quiz_id: editData.id`。

**路径 `:id` 完全不读。** 真正的 id 是 body `quiz_id`（`dto.UpdateQuizRequest`，`quiz_dto.go:30`，其余字段与创建相同）。`PUT /api/galgame-quiz/1` 配 `{"quiz_id":2,…}` 改的是 2。与 user 域 `/floating` 同一类「路径参数是假的」。

权限：作者或 `user.Can(perm.QuizEditAny)`（Bearer 无 staff）。trust 每次拿**合并后的全文**重跑（K18 的反面：分类/关联作品/隐藏开关变了也跑）。

`regradable := req.Type == quiz.Type`（`quiz_service.go:487`）。同类型才 `regradeAnswers`（`:532`）：跳过 essay；对每个 answerer 重新 `gradeQuiz`；**只把「错→对」翻过来**并补发 `QuizCorrectReward`（当前是 0，补发是空操作）；「对→错」不动。`gradeQuiz` 报错就 `continue`（`:540-542`），那一行保持旧判分。改题型则整批不重判，库里留下对不上新规则的 `is_correct`。`status_update_time` 每次编辑都 `now()`（`:498`），3 天顶帖窗口是答题用的（`BumpAnswerStats`，`quiz_repo.go:131-133`，`QuizBumpCutoff` = 3 天，`model/quiz.go:36-40`），编辑不走窗口。

响应 `{regraded: int}`（`quiz_handler.go:177`）。前端用这个弹「N 条作答已更正」（`Form.vue:217-222`）。

错误：404 不存在；403 无权限；400 内容校验；422 违禁；500 `"更新题目失败"`。

**测试**：`TestQuizBumpCutoff`（`model/quiz_test.go:8`）只测窗口函数。重判无测试。

---

### 2.3 `DELETE /api/galgame-quiz/:id`（#10）

**调用方**：`Play.vue:110-113` `` `/galgame-quiz/${id}?quiz_id=${id}` ``，`method: 'DELETE'`。

**又是路径假参数。** handler `ParseQueryAndValidate` 进 `dto.DeleteQuizRequest{QuizID query:"quiz_id"}`（`quiz_dto.go:76`，`quiz_handler.go:139-143`）。不带 query → 400；路径与 query 不一致 → 按 query 删。

权限：作者或 `user.Can(perm.QuizDeleteAny)`（Bearer 无 staff）。事务：`DeleteByID`（FK 级联 answer / favorite / galgame 关联）+ `AdjustMoemoepoint(-2, content_removed, ref=galgame_quiz:<id>)`。成功 `OKMessage "题目已删除"`，无 data。

不查作者 `IsRenderable`。不把删除当 204。

**测试**：无。

---

## 3. 题库互动：答题 / 收藏 / 质量

### 3.1 `POST /api/galgame-quiz/:id/answer`（#11）

**调用方**：`Play.vue:49-51`，body `{quiz_id, submitted}`。路径 id 与 body 同值，但 **handler 只读 body**（`dto.AnswerQuizRequest`，`quiz_dto.go:66`）。`POST /galgame-quiz/1/answer` 配 `{"quiz_id":2, submitted}` 答的是 2。

**调用方之外**：Nitro / Flutter 零。

**判分**（`gradeQuiz`，`quiz_content.go:139`）全在服务端：

| type | submitted | 对错 |
|---|---|---|
| single | `{value:int}` | `value == content.answer` |
| multiple | `{values:int[]}` | 与 `answers` 集合相等（`intSetEqual`，`quiz_content.go:295`） |
| judge | `{value:bool}` | 相等 |
| fill | `{values:string[]}` | 长度相同且每空 `normalizeFillAnswer`（小写、去空白、再去掉所有空格，`quiz_content.go:268`）命中该空任一 `accepted`；长度不对直接错，不是 400 |
| essay | `{text}` | **`gradeQuiz` default 返回 `nil, nil`**（`:177-178`），`is_correct` 保持 null，无萌萌点 |

submitted JSON 解失败 → `400/233 "答案格式错误"`。essay 的 text 与 fill 的 values 会进 trust（`quizAnswerModerationText`，`:246`）；single/multiple/judge 是下标/布尔，审核文本为空，跳过 check（`quiz_service.go:297`）。

已有 answer 行：`role=="author"` → `400/233 "不能回答自己出的题目"`；否则 `"您已经回答过该题目了"`。唯一约束 `UNIQUE (quiz_id, user_id)`（`045:75`）。并发双提交：第二笔 INSERT 冲突 → 事务失败 → `500/233 "提交答案失败"`，而不是那条 400。

**不查作者 IsRenderable。** 封禁作者的题 play 回 404，知道 id 仍能答。不查题目关联作品可见性。

事务：插 answerer 行、`BumpAnswerStats`（`answer_count+1`，对则 `correct_count+1`，创建 3 天内顶 `status_update_time`）、`CreateQuizAnswerMessage`（类型 `quiz-answered`，按 sender+receiver+type+link 去重，`interaction.go:44-67`）、答对则 `Award(+QuizCorrectReward)`（当前 0，不调用 OAuth）。然后 scan 填空/问答。

响应 `QuizAnswerResult`：**含完整 `answer: quiz.Content`（答案键）**、`submitted`、`is_correct`、`explanation`、`reward_delta`。这是作答后第一次合法拿到键的地方（作者走 play 的 `my_answer`）。

无幂等键。重试在「已回答」上 400，不会双写；并发窗口见上。

**测试**：`TestQuizAnswerModerationText`（`write_moderation_test.go:38`）只测哪些 submitted 会进 trust 文本。

---

### 3.2 `PUT /api/galgame-quiz/:id/favorite`（#12）

**调用方**：`Play.vue:257-261` `FavoriteToggle` `endpoint=/galgame-quiz/${id}/favorite`；`activity/card/Quiz.vue:50-55` 同组件；`favorite/Toggle.spec.ts:25,31` mock 了这条路径。`FavoriteToggle.vue:57-60` 一律 `method: 'PUT'`，无 body。

这是 **PUT 切换**（01 §5 / K16 禁止的那种）：`FindQuizFavorite` 已有则删并 `delta=-1`，否则插并 `+1`（`quiz_service.go:428-457`），再 `AdjustQuizFavoriteCount`，非本人则 `Award(delta, liked, ref=galgame_quiz:<id>)`。`KeyNonce` 每次新键，所以取消收藏的 −1 不会撞创建时的幂等键。响应 `OKMessage "操作成功"`，不下发新的 `favorited`/`count`（前端乐观更新）。

检查-再-写在事务外：两个并发 PUT 可能都看见「未收藏」、都 INSERT，一个撞唯一约束 500，或者都看见「已收藏」、都 DELETE，`favorite_count` −2。

路径 `:id` 这回是真的（`quiz_handler.go:154`）。不存在 → 404。不要求先作答。作者可以收藏自己的题（不给自己发 liked 分，`:448`）。不查作者可渲染。

**测试**：前端 Toggle.spec 只断言 `kunFetch` 被叫；API 无测试。

---

### 3.3 `PUT /api/galgame-quiz/:id/quality`（#13）

**调用方**：`play/Result.vue:72-74`，body `{quiz_id, quality_rating}`。**路径 `:id` 不读**，以 body `quiz_id` 为准（`dto.RateQuizQualityRequest`，`quiz_dto.go:71`，`1..10`）。

必须已有 answer 行，否则 `403/233 "请先回答题目再评分"`；`role=="author"` → `"不能给自己出的题目评分"`。已有 `quality_rating` 则改票（`sumDelta = new-old`，`countDelta=0`），否则首票 `countDelta=1`。这是**置位/改值**，不是切换；重复同一分数仍写库。响应 `{quality_average, quality_count, quality_rating}`，平均值用事务前读到的 `quiz.QualitySum` 加 delta **在内存里算**（`quiz_service.go:395-401`），并发评分会算错显示、库里的触发器/列才是真值。

不查作者可渲染。无测试。

---

## 4. `GET /api/galgame/search/picker`（#7）

QuizHandler 实现，但是通用作品选择器。挂在 public 组（`router.go:160`），**没有 OptionalAuth**。`routes.golden:124`：`cors → NamePreference → ContentStance → QuizHandler.SearchGalgames`。

**调用方（全部）**

搜索词：`/galgame/search/picker`、`search/picker`、`GalgameSearchAutocomplete`、`QuizGalgameOption`。

| 位置 | 角色 |
|---|---|
| `apps/web/app/components/galgame/SearchAutocomplete.vue:33-36` | **唯一 fetch 点**，`kunFetch('/galgame/search/picker', {query:{keywords}})` |
| `apps/web/app/components/galgame/quiz/GalgamePicker.vue:92` | 唯一渲染 `GalgameSearchAutocomplete` 的地方 |
| `apps/web/app/components/galgame/quiz/Form.vue:263` | 唯一渲染 `GalgameQuizGalgamePicker` 的地方（无 `galgameId` 预填时） |

Nitro `apps/web/server/**` 零。Flutter 零。没有作品编辑页、收藏夹、提交向导在用它——向导走 `/galgame/search/wizard`（`router.go:155-159`，另一条）。组件文件名是通用的 `galgame/SearchAutocomplete.vue`，类型却叫 `QuizGalgameOption`。

**请求**（`dto.QuizGalgameSearchRequest`，`quiz_dto.go:176`）：`keywords` `required,min=1`。空/缺席 400。无 `page`，服务端 `limit=12`（`quiz_service.go:694`）。

**实现**（`SearchGalgameOptions`，`quiz_service.go:696`）：

1. `url.Values{q, limit:12, sort:relevance}` + `client.ApplyWorksGate(q, isSFW)`（`catalog_face.go:72`）。SFW → `content_limit=sfw`；无论 SFW 都 `nsfw=true`（`openPopulation`，`:50-53`，注释钉死 **不要用 `content_rating` 年龄轴当人口闸**）。
2. `CatalogWorksSearch` → `CatalogGet("/catalog/works/search")`（`catalog_face.go:640`）→ `doV2` 改写成 **`/v2/catalog/works`**（`catalog_v2.go:67-68`）。当前代码注释写「Catalog is read entirely over /v2」（`catalog_v2.go:38-40`）；`catalog_v2_shape_test.go:9` 仍留着「dormant behind catalogReadsV1」的过时句。测试 `TestQuizPicker_OffersTheCatalogWithoutAClaimGate`（`works_population_gate_test.go:65`）断言路径 `/v2/catalog/works`，且 **不带 `claim_state` / `claimed`**。
3. 对每条 hit：`CatalogItemRenderable` 丢掉 `claim.state==hidden`（`catalog_wire.go:334`）；`CatalogItemGID` 取 id（`:388-394`）：kungal/galgame_wiki claim 的 `site_work_id`（论坛 gid），否则 **catalog work id**。
4. 收集到的整数再当**论坛 gid** 丢进 `GetBatchDetailPublic`（`quiz_service.go:726`）。名字/banner/officials 来自第二次查找，不来自搜索 hit 自己的标题。

因此：未认领的 catalog 行会出现在搜索命中里（picker「就是 catalog」，测试钉死），它的数字 id 是 catalog work id。第二次查找按论坛 gid 去 `refs=curated:<n>,galgame_wiki:<n>` 解析（`catalog_face.go:112-124`）。对不上任何 kungal claim 的 catalog id 在第二次被丢掉，搜索命中静默消失。对得上的是**另一个**论坛作品（数值撞车）——选项的 `id` 仍是搜索给的那个数，`name`/`banner` 却是论坛 gid 那部作品的。用户按名字点下去，`galgame_ids` 里写下的可能是错的作品，或一个从未作为论坛 gid 存在的 catalog id。`galgame_quiz_galgame.galgame_id` 无 FK（`047_quiz_description_and_multi_galgame.up.sql:14`，045 原文也写「No FK to galgame」）。

`isSFW` 第二次 `fetchDetailBriefs(ctx, ids, isSFW)` 会按 `content_limit=sfw` 再滤一次，SFW 读者第一次命中的 nsfw 行在第二次消失。

catalog 报错 → 空数组 200（`quiz_service.go:707-709`），不是 503。

**响应**：`[]QuizGalgameOption`：`id, name, banner, banner_thumbhash?, officials[]`（`quiz_dto.go:180`）。`id` 声称是论坛 gid。`name` 走 `CatalogEntityNames`（`catalog_wire.go:498`）→ `namepref.PrefersOriginal(ctx)`（`catalog_name.go:295-304`），即 `NamePreference` 中间件读的 `KUNGalgameSettings.showKUNGalgamePreferOriginalName`（`middleware/namepref.go:12`，`utils/settings.go:52`）。v1 必须改成显式参数或下发全名块（01 §3 末段）。

**鉴权**：完全匿名。ContentStance 仍跑，所以登录用户的账号 NSFW 立场会进 `IsSFW`；匿名走 cookie。

**测试**：`works_population_gate_test.go:65` 钉死无 claim 闸、空上游回空数组。不覆盖 gid/catalog 撞车。

---

## 7. 命名问题清单（迁 v1 时逐条执行）

| 现状 | 问题 | v1 |
|---|---|---|
| 路径 `:gid`、`:id`、`:postId` | 禁用短名；与响应字段不同名（K1） | `{galgame_id}` / `{quiz_id}` / `{comment_id}` |
| picker / 关联的 `id` | 未标明是论坛 gid 还是 catalog work id | `galgame_id`；catalog 的叫 `catalog_work_id` |
| `created` / `updated` / `edited` | 01 §3 禁用 | `created_at` / `updated_at` / `edited_at` |
| `status_update_time` | 禁用名 | `bumped_at`（答题顶，3 天窗） |
| `view` | 禁用名 | `view_count`；`view_1d` → `view_1d_count` |
| `user` | 禁用名 | `author` |
| `quiz_data` | 数组不叫 `items` | `items` |
| `my_status` / `is_author` / `is_favorited` / `is_liked` / `is_correct` | 随查看者变化，应进 `viewer` | `viewer.has_answered` 等 |
| `content` 在 QuizPlay = 剥离后的选项 JSON；在评论 = Markdown 源 | 同名不同型（A4） | 题：`prompt` / `choices`；评论：`content` 文档 + 源 `content_markdown` 或 `text` |
| `my_answer.answer` | 其实是完整题目 JSONB（含键），不是用户作答 | 作答后下发 `solution` / `answer_key`；用户作答已是 `submitted` |
| `status` int（评论 0/1/2） | 与 Problem.status 撞型 | `state`: `visible` / `held` / `deleted` |
| `type` / `category` / `spoiler_level` | 开放或仅创建时 oneof；列表过滤未知值当等值查 | 封闭枚举，未知 → `400 UNKNOWN_ENUM_VALUE` |
| `sort_field`+`sort_order` | 未知 sort 静默回落 | `sort=bumped_at_desc` 一类 token，未知 → `400 UNKNOWN_SORT` |
| `favorited: number[]` | 不是 list 对象 | `object:"list", items:[{quiz_id}]` 或嵌进 me |
| `locked` / `anchor_kind` / `anchor_id` | 对 galgame 墙 lock 恒 false；`kind` 禁用 | 需要再决定是否下发锚点 |
| 全部 id 是 JSON number | 01 §3 | 十进制字符串；community post id 是 int64，`maxLength 20` |
| `/galgame-quiz/:id/answer` | 动词路径（B7） | `POST …/answers` |
| `/galgame-quiz/:id/favorite` PUT 切换 | 非幂等 PUT | `PUT`/`DELETE …/favorite` |
| `/galgame/search/picker` | 动词「picker」 | `GET /galgames` 的搜索，或 `/galgame-search-hits` |
| `/mine/answered` `/mine/favorites` | 动词过去分词 | `/me/galgame-quiz-answers` `/me/galgame-quiz-favorites` |

---

## 8. 疑似 bug 与契约债（平铺，不排序）

每一条都带坐标。不修。

1. `GetAllQuizzes` 的 `isSFW` 传入后丢弃（`quiz_handler.go:31`，`quiz_service.go:62-84`）。SFW 列表含 NSFW 作品题；play 的 `galgamesDetailFor(..., false)`（`quiz_service.go:659`）作答后还下发 NSFW banner。
2. `hydrateCards` / `GetQuizAnswers` / 评论 `renderPosts` 的 `Hydrate` 丢 OAuth 错（`userclient/hydrate.go:37`），Placeholder `Status=0`（`userclient.go:211`）让 `IsRenderable` 为真。封禁作者内容在 OAuth 故障时可见。
3. `GetQuizPlay` 的 `User(..., _)`（`quiz_service.go:146`）更进一步：故障时作者是全零 `{id:0}`，题目仍 200。
4. `total` 与 `items` 谓词不同：题库列表 SQL COUNT vs Go 丢封禁作者（`quiz_service.go:110-120`）；评论 `Thread.PostsCount` vs `renderPosts` 过滤（`community_comment_service.go:86-90`）。
5. 题库所有 `ORDER BY` 无 `id` tie-breaker（`quiz_repo.go:90,107,271`）。
6. 未知 `sort_field` 静默回落 `q.created`（`quiz_repo.go:69-86`）。
7. `ListPaginated` / `ListAnsweredByUser` / `FindQuizAnswerers` / `FindViewerAnswers` / `FindFavoritedQuizIDs` / `CountLikes` / `LikedSet` 的 `Count`/`Scan`/`Pluck` 都不接 error。
8. `FindByID` 任何 error 当 404（`quiz_repo.go:30-32`），库挂了也是「题目不存在」。
9. `IncrementView` 在 goroutine 里丢错（`quiz_repo.go:112-117`）；SSR 每次加浏览。
10. `GET /answers` 对未作答者下发 `is_correct`（`quiz_service.go:621-624`）；`DetailPanel.vue:55-60` 无条件拉取。
11. `GET /answers` 不查题目是否存在、不查题目作者可渲染（`quiz_service.go:601-631`）。play 对封禁作者 404，本面仍列作答者。
12. `AnswerQuiz` / `ToggleQuizFavorite` / `RateQuizQuality` / `UpdateQuiz` / `DeleteQuiz` 不查作者 `IsRenderable`。封禁作者的题仍能答、收藏、评分。
13. `POST …/answer`、`PUT …`、`PUT …/quality`、`DELETE …` 的路径 `:id` 不读，以 body/query `quiz_id` 为准（`quiz_handler.go:107-111,139-143,123-127,169-173`）。
14. `PUT …/favorite` 是切换（`quiz_service.go:428-436`）。检查-再-写在事务外，并发能把 `favorite_count` 打穿或 500。
15. `regradeAnswers` 只翻错→对，grade 错 `continue`（`quiz_service.go:538-542`）；改 `type` 完全不重判（`:487,512`）。
16. `QuizCorrectReward = 0`（`constants/moemoepoint.go:23`）但 UI 写「答对可获得萌萌点」（`Container.vue:93`）。
17. 创建/答题 POST 无 `Idempotency-Key`。创建弱网重试双题双分。答题并发唯一约束变成 500（`quiz_service.go:344`）。
18. picker `CatalogItemGID` 在无 claim 时返回 catalog work id（`catalog_wire.go:388-394`），随后当论坛 gid 去 `GetBatchDetailPublic`（`quiz_service.go:715-726`）。数值撞车时选项名字与 id 可以不是同一部作品。`TestQuizPicker_OffersTheCatalogWithoutAClaimGate` 钉死搜索**不**加 claim 闸。
19. picker / play / edit 的作品名走 `NamePreference` cookie（`catalog_name.go:298`，`middleware/namepref.go:12`）。v1 禁止。
20. `contentLimitOf` 在 claim 无 `content_limit` 时用 `content_rating` 推导展示轴（`catalog_wire.go:271-278`）。未认领行（picker 会搜到）走这条回退。闸本身 `ApplyWorksGate` 用的是 `content_limit`（`catalog_face.go:63-73`），这条回退出现在 **brief 字段** `QuizGalgameBrief.ContentLimit` / `QuizGalgameDetail.ContentLimit`。
21. `stripQuizContent` / `mustJSON` 丢 unmarshal/marshal 错（`quiz_content.go:124,312-314`）。坏 JSONB 变成空选项，用户仍能提交。
22. 编辑每次跑全文 trust（`quiz_service.go:476-480`），违反 K18。
23. `galgame_ids` 不存在性校验（`quiz_repo.go:221-241`）。
24. 前端禁用 fill/essay，API `oneof` 仍接受（`quiz_dto.go:21` vs `constants/galgame-quiz.ts:25`）。
25. 出题表题干 maxlength 2000、API max 200（`Form.vue:305` vs `quiz_dto.go:24`）。
26. `page`/`limit` 因 `min=1` 事实必填，校验失败文案走「长度不能小于」（`validate.go:96-97`）。
27. `GetMyAnswered` 解析了整份 `QuizListRequest` 却忽略筛选（`quiz_handler.go:44-48`）。
28. `GetMyFavorites` 无界 id 数组（`quiz_repo.go:187-195`）。
29. `GetQuizAnswers` 硬截 100、无分页（`quiz_service.go:599,266-274`）。
30. `Award` 在事务里 `go` 出去（`interaction.go:14-15` + `pusher.go:55`）：本地提交成功、OAuth 失败时用户看到成功但分没到。`KeyNonce` 含纳秒，失败不能用同键重试。
50. 本段 20 条 HTTP 路由没有一条契约测试；没有 `GetQuizPlay` / `stripQuizContent` 测试。

---

## 9. 需要生产库回答的问题

编排者跑；我没有库。表/列名来自 `internal/galgame/model/{quiz,community_post,local}.go` 与 `migrations/045,046,047,048,050,051,052,057,065`。

```sql
-- 题库规模与词表（没有一行的分支不要进 v1 封闭枚举）
SELECT type, count(*) FROM galgame_quiz GROUP BY 1 ORDER BY 2 DESC;
SELECT category, count(*) FROM galgame_quiz GROUP BY 1 ORDER BY 2 DESC;
SELECT spoiler_level, count(*) FROM galgame_quiz GROUP BY 1;
SELECT difficulty, count(*) FROM galgame_quiz GROUP BY 1 ORDER BY 1;
SELECT hide_galgame, count(*) FROM galgame_quiz GROUP BY 1;

-- fill/essay 前端已藏，API 仍开：有没有行决定 v1 是否保留
SELECT type, count(*) FROM galgame_quiz WHERE type IN ('fill','essay') GROUP BY 1;

-- 关联作品：空关联、一对多、指向不存在的本地 galgame、可能的 catalog id 撞车
SELECT count(*) FROM galgame_quiz;
SELECT count(*) FROM galgame_quiz q
  WHERE NOT EXISTS (SELECT 1 FROM galgame_quiz_galgame g WHERE g.quiz_id = q.id);
SELECT count(*) AS links, count(DISTINCT quiz_id) AS quizzes, count(DISTINCT galgame_id) AS games
  FROM galgame_quiz_galgame;
SELECT count(*) FROM galgame_quiz_galgame g
  WHERE NOT EXISTS (SELECT 1 FROM galgame x WHERE x.id = g.galgame_id);
-- 同一整数既是论坛 gid 又被当作关联：需要与 catalog 对账，论坛侧先列出被引用的 galgame_id
SELECT galgame_id, count(*) FROM galgame_quiz_galgame GROUP BY 1 ORDER BY 2 DESC LIMIT 50;

-- 作答
SELECT role, count(*) FROM galgame_quiz_answer GROUP BY 1;
SELECT is_correct IS NULL AS is_null, is_correct, count(*)
  FROM galgame_quiz_answer WHERE role = 'answerer' GROUP BY 1, 2;
SELECT count(*) FROM galgame_quiz_answer WHERE role = 'answerer' AND quality_rating IS NOT NULL;
SELECT percentile_cont(0.99) WITHIN GROUP (ORDER BY c)
  FROM (SELECT count(*) c FROM galgame_quiz_answer WHERE role = 'answerer' GROUP BY quiz_id) t;
-- 超过 answers 面 100 条截断的题
SELECT count(*) FROM (
  SELECT quiz_id FROM galgame_quiz_answer WHERE role = 'answerer'
  GROUP BY quiz_id HAVING count(*) > 100
) t;

-- 收藏
SELECT count(*) FROM galgame_quiz_favorite;
SELECT count(*) FROM galgame_quiz WHERE favorite_count <> (
  SELECT count(*) FROM galgame_quiz_favorite f WHERE f.quiz_id = galgame_quiz.id
);

-- 浏览日表
SELECT count(*) FROM galgame_quiz_view_daily;
SELECT count(*) FROM galgame_quiz WHERE view_7d = 0 AND view > 0;

-- 封禁作者的题（需要 OAuth 用户 status；论坛侧只能列出 user_id 分布）
SELECT user_id, count(*) FROM galgame_quiz GROUP BY 1 ORDER BY 2 DESC LIMIT 20;
```

community 库（编排者若能连 infra community）还需要：`site_game` 锚点帖数、`status` 分布、`posts_count` vs 可见帖、held 比例、单墙最大帖数（决定游标 vs 页码）、post id 是否已超过 int32（feed 溢出）。

访问日志才能回答的：`sort_field=view_1d` 是否有人用；fill/essay 创建是否还有客户端；picker 的 keywords 是否常命中未认领作品。

---

## 10. 提议的 v1 资源图（只是提议，不是契约）

| 旧 | 提议 v1 | 理由 |
|---|---|---|
| `GET /api/galgame-quiz/all` | `GET /api/v1/galgame-quizzes` 页码集合 | 可浏览题库，要分页器；`galgame_id` / `author_id` / `type` / `category` 当过滤 |
| `GET /api/galgame-quiz/mine/answered` | `GET /api/v1/me/galgame-quiz-answers` 游标 | 「我的作答」是追加流 |
| `GET /api/galgame-quiz/mine/favorites` | `GET /api/v1/me/galgame-quiz-favorites` | 收藏槽位列表，不要裸 id 数组 |
| `GET /api/galgame-quiz/:id` | `GET /api/v1/galgame-quizzes/{quiz_id}` | 作答形状；未作答剥离答案键；`viewer` 持 `has_answered` / `has_favorited` / `can_edit` |
| `GET /api/galgame-quiz/:id/answers` | `GET /api/v1/galgame-quizzes/{quiz_id}/answers` | 未作答不得含 `submitted` **也不得含** `is_correct` |
| `GET /api/galgame-quiz/:id/edit` | `GET /api/v1/galgame-quizzes/{quiz_id}/source` | 编辑源文 + 完整 content，`can_edit` |
| `POST /api/galgame-quiz` | `POST /api/v1/galgame-quizzes` | 201 + Location；必填 Idempotency-Key |
| `PUT /api/galgame-quiz/:id` | `PATCH /api/v1/galgame-quizzes/{quiz_id}` | 部分更新；id 只在路径 |
| `DELETE /api/galgame-quiz/:id` | `DELETE /api/v1/galgame-quizzes/{quiz_id}` | 204；id 只在路径 |
| `POST /api/galgame-quiz/:id/answer` | `POST /api/v1/galgame-quizzes/{quiz_id}/answers` | 动词改资源；id 只在路径；必填幂等键 |
| `PUT /api/galgame-quiz/:id/favorite` | `PUT`/`DELETE /api/v1/galgame-quizzes/{quiz_id}/favorite` | K16 槽位 |
| `PUT /api/galgame-quiz/:id/quality` | `PUT /api/v1/galgame-quizzes/{quiz_id}/quality-rating` | 置位分数，不是切换 |
| `GET /api/galgame/search/picker` | `GET /api/v1/galgames` 的搜索（或独立 `galgame-search-hits`） | 这是作品搜索，不是题库子资源；id 必须是 `galgame_id`，未认领 catalog 行要么排除要么同时下发 `catalog_work_id` |

路径参数与响应字段同名。切换类互动禁止再 PUT 翻转。`gid` 全部消失。
