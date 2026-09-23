# G2 · 题库

> G 轨第二段（G0 改号之后），2026-09-23。`QuizHandler` 的 **13** 条旧路由：12 条 `/api/galgame-quiz/**`，加上 `/api/galgame/search/picker`。迁移号 **143**（G 号段 140–159；140–142 已占用）。
> 契约与变异题同一个提交，早于实现。普查全文在 [census/galgame-quiz.md](census/galgame-quiz.md)，结论已对过代码。评论墙已在 RC（D6 / D18），本轨不重做。

## 1. 普查（生产实测 2026-09-23）

### 1.1 数字

| 事实 | 数字 |
|---|---|
| `galgame_quiz` | 267 行 |
| `type` | single 217、multiple 31、judge 19、**fill 0、essay 0** |
| `category` | plot 102、character 72、trivia 28、system 22、company 15、other 11、music 11、voice 6 |
| `spoiler_level` | none 192、portion 51、serious 24 |
| `difficulty` | 1–10 全部有行 |
| `hide_galgame` | true 25 |
| 关联 | 49 题没有关联作品；383 条链接覆盖 218 题、207 部作品；单题最多 14 条；3 条指向没有本地 `galgame` 行的作品 |
| 作答 | 11,116 条 answerer 行（正确 6,441、错误 4,675）加 267 条 author 行。质量评分 1,185 条。作答最多的题 251 人（p99 209），旧面 100 行截断会切掉真实题 |
| 收藏 | **4** 行；每题的 `favorite_count` 都对得上 |
| 文本 | 最长题干 164（API 上限 200）；描述 438（上限 20000）；解析 795（上限 2000）；最多 15 个选项；最长选项 91 |
| 题干标记 | 1 条含 `\|\|spoiler\|\|`，18 条含换行，0 条含 `/image/`。选项 0 条剧透。描述 35 条看起来像 Markdown；解析 8 条看起来像 Markdown，1 条含 `\|\|` |

### 1.2 调用方

- `apps/web/server/**`：零命中（普查 §1.1；2026-09-23 再搜 `galgame-quiz` / `search/picker` 仍零）。
- Flutter App（`/home/kun/Desktop/code/website/kungal-apps`）：Dart 源零命中（普查 §1.1；2026-09-23 再搜 `galgame-quiz`、`galgame_quiz`、`search/picker` 仍零）。`docs/proj/app-direct-api.md` 未列出本段路径。
- 网页：
  - 浏览 `components/galgame/quiz/{Container,List}.vue`（`pages/galgame-quiz/index.vue`），`limit` **50**，默认 `sort_field=update_time` `sort_order=desc`
  - 作品页 tab `components/galgame/quiz/GalgamePanel.vue`（`components/galgame/Galgame.vue:184-185`），`galgame_id` + `limit=12`
  - 用户页 `pages/user/[id]/quiz.vue` + `components/user/Quiz.vue`，出题 tab `user_id` + `limit=50`；本人「答题」tab 打 `/mine/answered`
  - 作答 `pages/galgame-quiz/[id].vue` SSR + `components/galgame/quiz/Play.vue` + `play/{AnswerInput,Result}.vue` + `DetailPanel.vue`
  - 出题 / 编辑 `pages/edit/galgame/quiz.vue` + `components/galgame/quiz/{Form,Publish,GalgamePicker}.vue` + `content/Editor.vue`；编辑拉 `/edit`
  - 作品选择器 `components/galgame/SearchAutocomplete.vue`（唯一 fetch 点）← 只有 `GalgamePicker.vue` 渲染它
  - 动态卡 `components/activity/card/Quiz.vue` + `composables/useMyQuizInteractions.ts` 打 `/mine/favorites`
  - 收藏按钮 `components/favorite/Toggle.vue`（PUT 切换）；`favorite/Toggle.spec.ts` mock 了 `/galgame-quiz/:id/favorite`
  - 评论墙已经走 RC：`components/galgame/quiz/comment/CommunityContainer.vue`

### 1.3 旧面的问题（普查编号 E*，对应 census 发现号）

| 编号 | 事实 |
|---|---|
| E1 | `GetAllQuizzes` 的 `isSFW` 传入后丢弃（census 发现 1，`quiz_handler.go:31`，`quiz_service.go:62-84`）。SFW 列表含 NSFW 作品题 |
| E2 | `hydrateCards` / `GetQuizAnswers` 的 `Hydrate` 丢 OAuth 错，Placeholder `Status=0` 可渲染（发现 2） |
| E3 | `GetQuizPlay` 的 `User(..., _)` 故障时作者是全零 `{id:0}`，题目仍 200（发现 3） |
| E4 | `total` 是 SQL `COUNT(*)`，`quiz_data` 在 Go 里丢掉不可渲染作者，谓词不同（发现 4） |
| E5 | 全部 `ORDER BY` 无 `id` 决胜键（发现 5，`quiz_repo.go:90,107,271`） |
| E6 | 未知 `sort_field` 静默回落 `q.created`（发现 6）。前端 `time` 碰巧落到同一列 |
| E7 | `Count` / `Scan` / `Pluck` 都不接 error（发现 7） |
| E8 | `FindByID` 任何 error 当 404（发现 8） |
| E9 | `IncrementView` 在 goroutine 里丢错（发现 9）；SSR 每次加浏览 |
| E10 | `GET /answers` 对未作答者下发 `is_correct`（发现 10）；`DetailPanel.vue:55-60` 无条件拉取 |
| E11 | `GET /answers` 不查题目是否存在、不查作者可渲染（发现 11） |
| E12 | 答题 / 收藏 / 评分 / 编辑 / 删除不查作者 `IsRenderable`（发现 12） |
| E13 | `POST …/answer`、`PUT …`、`PUT …/quality`、`DELETE …` 的路径 `:id` 不读，以 body/query `quiz_id` 为准（发现 13） |
| E14 | `PUT …/favorite` 是切换；检查-再-写在事务外（发现 14） |
| E15 | `regradeAnswers` 只翻错→对；改 `type` 完全不重判（发现 15） |
| E16 | `QuizCorrectReward = 0` 但 UI 写「答对可获得萌萌点」（发现 16，`Container.vue:93`，`pages/galgame-quiz/index.vue:5`） |
| E17 | 创建 / 答题 POST 无 `Idempotency-Key`；答题并发唯一约束变成 500（发现 17） |
| E18 | picker `CatalogItemGID` 在无 claim 时返回 catalog work id，再当论坛 gid 二次查找（发现 18）。**G0 之后这条不再成立**：picker 的 id 就是 work id，普查 §4 / 发现 18 的撞车作废 |
| E19 | picker / play / edit 的作品名走 `NamePreference` cookie（发现 19） |
| E20 | `contentLimitOf` 在 claim 无 `content_limit` 时用 `content_rating` 推导展示轴（发现 20） |
| E21 | `stripQuizContent` / `mustJSON` 丢 unmarshal 错（发现 21） |
| E22 | 编辑每次跑全文 trust（发现 22），违反 K18 |
| E23 | `galgame_ids` 不存在性校验（发现 23） |
| E24 | 前端禁用 fill/essay，API `oneof` 仍接受（发现 24） |
| E25 | 出题表题干 maxlength 2000、API max 200（发现 25，`Form.vue:305`） |
| E26 | `page` / `limit` 因 `min=1` 事实必填（发现 26） |
| E27 | `GetMyAnswered` 解析了整份 `QuizListRequest` 却忽略筛选（发现 27） |
| E28 | `GetMyFavorites` 无界 id 数组（发现 28） |
| E29 | `GetQuizAnswers` 硬截 100、无分页（发现 29）；生产最多 251 人 |
| E30 | `Award` 在事务里 `go` 出去，键是 `KeyNonce`（发现 30） |

## 2. 范围与寻址

路径段就是目标。路径 `{quiz_id}` 与响应字段 `quiz_id` 同名（K1）。每个 id 来自路径，永不来自身体或 query（修 E13）。

集合是 **`/api/v1/quizzes`**，对象 **`quiz`**；本论坛只有这一种题目。

| # | 旧 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /galgame-quiz/all` | `GET /api/v1/quizzes` 页码集合（K11），默认 `limit` **50**（浏览页 `Container.vue:26`；作品页传 12，用户页传 50），最大 100 | optional |
| 2 | `GET /galgame-quiz/:id` | `GET /api/v1/quizzes/{quiz_id}` 作答形状 | optional |
| 3 | `GET /galgame-quiz/:id/answers` | `GET /api/v1/quizzes/{quiz_id}/answers` **游标**集合，最新在前，`id` 决胜 | optional |
| 4 | `GET /galgame-quiz/:id/edit` | `GET /api/v1/quizzes/{quiz_id}/source`，要求 `viewer.can_edit` | required |
| 5 | `GET /galgame-quiz/mine/answered` | `GET /api/v1/me/answered-quizzes` 页码，最新在前（条目是题，不是作答） | required |
| 6 | `GET /galgame-quiz/mine/favorites` | `GET /api/v1/me/quiz-states?quiz_ids=…`（1–100，逗号形；与 `/me/topic-states` 同形） | required |
| 7 | `GET /galgame/search/picker` | `GET /api/v1/work-suggestions?q=…&include_nsfw=…` 至多 12 个 `WorkRef`，不分页。不得放在 `/works/search`（会盖住 `/works/{work_id}`） | public |
| 8 | `POST /galgame-quiz` | `POST /api/v1/quizzes` → **201** + `Location` + `Quiz`；`Idempotency-Key` **必带**（K12） | required |
| 9 | `PUT /galgame-quiz/:id` | `PATCH /api/v1/quizzes/{quiz_id}` 部分更新 | required |
| 10 | `DELETE /galgame-quiz/:id` | `DELETE /api/v1/quizzes/{quiz_id}` → **204** | required |
| 11 | `POST /galgame-quiz/:id/answer` | `POST /api/v1/quizzes/{quiz_id}/answers` → **201**；`Idempotency-Key` **必带** | required |
| 12 | `PUT /galgame-quiz/:id/favorite` | `PUT` + `DELETE /api/v1/quizzes/{quiz_id}/favorite`（K16 槽；都幂等，都 200） | required |
| 13 | `PUT /galgame-quiz/:id/quality` | `PUT /api/v1/quizzes/{quiz_id}/quality-rating` 体 `{ "rating": 1..10 }` → **200** | required |

13 条旧路由全删，`legacy_route_baseline` 下调 **13**（G1 合入后基线 114 → 101，以生成物为准）。v1 共 **14** 个操作（收藏拆成置位 / 撤销）。

代码位置：`internal/quiz/apiv1/**`；测试 `internal/app/v1_quiz_*_test.go`。`listMyAnsweredQuizzes` / `listMyQuizStates` 的 OpenAPI tag 是 `me`，`listWorkSuggestions` 是 `works`，其余 `quizzes`。

## 3. 形状

### 3.1 命名

| 旧 | v1 | 为什么 |
|---|---|---|
| 路径 `:id` / 字段不对齐 | `{quiz_id}` | 路径参数与字段同名（K1） |
| `gid` / `galgame_id` / `galgame_ids` | `work_id` / `work_ids` | G0：论坛作品 id 就是 catalog work id |
| `created` / `updated` | `created_at` / `updated_at` | 禁用名 |
| `status_update_time` | `bumped_at` | 禁用名；答题 3 天窗顶，编辑也写这一列 |
| `view` | `view_count` | 计数 `_count`；排序 token 仍用 `view_*`（与话题列表同一套短名） |
| `user` | `author` | 禁用名 |
| `quiz_data` | `items` | 集合数组 |
| `type` | `quiz_type` | Problem 的 `type` 是 URI（G8） |
| `category` | `quiz_category` | 话题 / 版块 / 草稿的 `category` 是 `galgame` / `technique` / `others`（G8；门把 `category` 放进了例外表，词表不同仍要改名） |
| `hide_galgame` | `is_work_hidden` | F1 布尔 `is_` |
| `question` / `question_html` | 读面 `prompt`（受限内容文档）；源面 `prompt_text` | 文档与源拆开；`prompt` 现 spec 未占用 |
| `description` / `description_html` | 读面 `content`（完整 Markdown 内容文档）；源面 `description_markdown` | Website / Poll / Doc / Problem 的 `description` 是 string（G8） |
| `explanation` | 只出现在 `solution.explanation`（内容文档）；源面 `explanation_markdown` | `explanation` 现 spec 未占用 |
| `content`（含答案键的 JSONB） | 读面 `choices` + `solution`；写面 `choices` + `correct_choice_indexes` / `judge_answer` | 同名不同型（A4）；Poll 的 `options` 是对象数组，选项字符串改叫 `choices` |
| `galgames` | `works`：`WorkRef[]` | G0 之后就是作品引用；现 spec 没有名为 `works` 的属性 |
| `my_status` / `is_author` / `is_favorited` / `my_answer` | `viewer.*` | K9 |
| `submitted` 自由 JSON | `QuizSubmission`：`choice_indexes` + `judge_choice` | 静态形状 |
| `favorited: number[]` | `GET /me/quiz-states?quiz_ids=` 的 `items[].has_favorited` | 修 E28；与 `/me/topic-states` 同形 |
| `sort_field` + `sort_order` | `sort` 封闭 token `<键>_<asc\|desc>` | 01 §4 |
| `keywords` | `q` | 与其它搜索一致 |
| `quality_rating`（PUT 体） | `rating` | 与详情 `viewer.quality_rating`（可空）同名会撞可空性，G1 实用性 PUT 同一先例 |

### 3.2 对象

**`QuizSummary`（`object: "quiz"`，列表条目）与 `Quiz`（同 object，详情）重叠字段同名同型（G8）。**

公共字段：`id`、`prompt`、`quiz_type`、`quiz_category`、`difficulty`、`spoiler_level`、`author`（UserRef）、`view_count`、`answer_count`、`correct_count`、`favorite_count`、`quality_average`（一位小数的 number，无人评分时 `null`）、`quality_count`、`comment_count`（本地列，RC 墙写入维护）、`created_at`、`updated_at`、`bumped_at`。

- `prompt`：受限内容文档（§3.9）。列表与详情同一份。
- `author` 在列表与详情里都是可渲染用户；不可渲染作者的题不会出现在列表，详情是 404（K-G12）。
- `quality_average` 与 G1 的 `practicality_average` 同形：有评分才发数字。旧面 `count=0` 发 `0`，v1 发 `null`。
- 数组永不 `null`。

**`Quiz` 额外：**

- `content`：完整 Markdown 内容文档，由列 `description` 经 `internal/apiv1/content.Converter.Convert`（与 `internal/doc/apiv1`、话题、墙同一管线）产出。空描述是空文档，不是 `null`。
- `choices`：`string[]`，每项 ≤200。`single` / `multiple` 是选项原文，不含答案键；`judge` 是 `[]`。永不 `null`。
- `solution`：`QuizSolution | null`。调用者能看见答案键时是对象，否则 `null`（K-G9）。
- `is_work_hidden`：列 `hide_galgame`。
- `works`：`WorkRef[]`。`is_work_hidden` 为真且调用者看不见答案键时是 `[]`；否则用 `WorkRefOf` 从一次 `client.CatalogRowsByWorkIDs` 的 catalog 行构建（K-G10）。catalog 失败 → 503。hidden claim 的行被 `CatalogItemRenderable` 丢掉，数组里不出现。NSFW 旗在 `WorkRef.is_nsfw` 上，详情不再按 SFW 藏作品。
- `viewer`：`QuizViewer`；匿名为 `null`。

**`QuizSolution`（`object: "quiz_solution"`）**：`correct_choice_indexes`（`integer[]`，0 起；`judge` 是 `[]`）、`is_statement_true`（`bool | null`，非判断题为 `null`）、`explanation`（完整 Markdown 内容文档，空解析是空文档）。

**`QuizSource`（`object: "quiz_source"`）**：`quiz_id`、`quiz_type`、`quiz_category`、`difficulty`、`spoiler_level`、`prompt_text`（maxLength 200）、`description_markdown`（maxLength 20000）、`explanation_markdown`（maxLength 2000）、`choices`、`correct_choice_indexes`、`judge_answer`、`work_ids`、`is_work_hidden`。要求 `viewer.can_edit`，否则 `403 PERMISSION_REQUIRED`。档位 required。故意带完整答案键。

**`QuizAnswer`（`object: "quiz_answer"`，作答集合条目）**：`id`（作答行 id）、`quiz_id`、`answerer`（UserRef）、`submission`（`QuizSubmission | null`）、`is_correct`（`bool | null`）、`answered_at`（列 `created`）。不可渲染的作答者从集合里丢掉（K-G12）。`submission` 与 `is_correct` 只发给已经作答、或能看见答案键的调用者（作者、`can_edit`）；其余人（含匿名）这两项都是 `null`（K-G9，修 E10）。集合从不带 `solution`。

**`QuizSubmission`**：`choice_indexes`（`integer[]`，0 起；判断题是 `[]`）、`is_statement_true`（`bool | null`，非判断题为 `null`；作答体里可缺席）。写面与读面同一形状。

**`QuizAnswerResult`（`object: "quiz_answer_result"`，POST answers 的 201）**：`answer`（刚写入的 `QuizAnswer`，`submission` / `is_correct` 非 null）+ `solution`（非 null）。不再发 `reward_delta`（恒 0，A5）。

**`QuizViewer`（详情 / 写面回的 `viewer`）**：`has_answered`、`answer`（`QuizViewerAnswer | null`：调用者自己的 `submission`、`is_correct`、`answered_at`；未作答为 `null`）、`can_edit`、`can_delete`、`has_favorited`、`quality_rating`（1–10 或 `null`）。匿名为整个 `viewer: null`。Bearer 的 `can_*` 不含 staff（K2）。`can_edit` = 作者或 `user.Can(perm.QuizEditAny)`；`can_delete` 对 `perm.QuizDeleteAny`。作者的 author 行不算作答：`has_answered` 为假，`answer` 为 `null`，但作者能看见 `solution`（K-G9）。

**`QuizSummaryViewer`（列表条目的 `viewer`）**：`has_answered`、`is_correct`（未作答为 `null`）。Go 类型与 `QuizViewer` 分开，JSON 键仍是 `viewer`（`viewer` 在 G8 例外表里，G1 的 `ToolsetViewer` / `ResourceViewer` 同一做法）。匿名整段 `null`。作者在列表上 `has_answered` 为假、`is_correct` 为 `null`（任务书给定的两字段；旧 `my_status=author` 的笔形图标不再从列表 viewer 来，见 §8 O5）。

**`QuizEngagement`（`object: "quiz_engagement"`，PUT/DELETE favorite 的 200）**：`quiz_id`、`favorite_count`、`viewer: { has_favorited }`（required 档，viewer 非 null）。`has_favorited` 与详情同名同型。

**`QuizQuality`（`object: "quiz_quality"`，PUT quality-rating 的 200）**：`quiz_id`、`quality_average`、`quality_count`、`viewer: { quality_rating }`（可空，与详情 `QuizViewer.quality_rating` 同型；这条响应里恒有刚写入的 1–10）。平均值从数据库重算，一位小数（修 §3.3 内存加法）。

**`QuizState`（`object: "quiz_state"`）**：`quiz_id`、`has_favorited`。照 `/me/topic-states`：请求里每个调用者可读的 id 都回一条（没收藏是 `has_favorited: false`，那是答案不是缺席），保持请求顺序；不存在、已删除、作者不可渲染的 id 进 `missing`，两种原因不区分，不能拿来探测 id。

### 3.3 词表（K-G8）

封闭枚举。fill / essay 生产 0 行，网页不能出（K26 先例），v1 不收；创建 → 请求体枚举在进 handler 之前就拒（`422 VALIDATION_FAILED` `UNKNOWN_VALUE`）。判分代码删除。

| 字段 | 取值 |
|---|---|
| `quiz_type` | `single` `multiple` `judge` |
| `quiz_category` | `plot` `character` `system` `music` `voice` `company` `trivia` `other`（存储值不改，仍是单数 `other`） |
| `spoiler_level` | `none` `portion` `serious` |
| `difficulty` | 整数 1–10（不是枚举） |
| `sort`（仅 `GET /quizzes`） | 见下 |

**`GET /quizzes` 的 `sort`**，网页 `quizSortFieldOptions` + 升降序按钮的笛卡尔积，默认 **`bumped_at_desc`**（`Container.vue:24-25` 现在的 `update_time` + `desc`）：

`bumped_at_desc`（默认）`bumped_at_asc` `created_desc` `created_asc` `view_desc` `view_asc` `view_1d_desc` `view_1d_asc` `view_7d_desc` `view_7d_asc` `view_30d_desc` `view_30d_asc` `difficulty_desc` `difficulty_asc` `answer_count_desc` `answer_count_asc`

每个排序带同方向的 `id` 决胜键：`bumped_at_desc` → `q.status_update_time DESC, q.id DESC`，`created_asc` → `q.created ASC, q.id ASC`，`view_1d_*` 用当天 `galgame_quiz_view_daily` 之和的子查询再加 `id`，其余同理。未知 `sort` → `400 UNKNOWN_SORT`，不得静默回落（修 E6）。

未知过滤值 → `400 UNKNOWN_ENUM_VALUE`。缺席 = 不过滤（没有 `all` token）。`difficulty` 缺席不过滤；送 `0` 是 `OUT_OF_RANGE`，网页「全部难度」改为不传该参数（修 E26 的 0 哨兵）。

### 3.4 列表（K-G11、K-G12）

**`GET /quizzes`**：页码（`collect.PageNumber` 的 `page` / `limit`，本集合默认 `limit` **50**，最大 100；`CheckDepth` `page × limit ≤ 10000`，越界 `400 INVALID_PARAMETER` `OUT_OF_RANGE`）。响应 `repr.PageList[QuizSummary]`，必发 `total` + `total_relation`。

过滤：`work_id`、`author_id`、`quiz_type`、`quiz_category`、`difficulty`（精确 1–10）、`spoiler_level`、`include_nsfw`（默认 `false`）。

**NSFW（K-G11）**：`include_nsfw=false`（默认）排除「关联了任一本地 `galgame.content_limit = 'nsfw'` 的作品」的题。这是迁移 079 的缓存，与 `WorkRef.is_nsfw` 同一编辑轴。`content_limit` 为 NULL 的本地行放行（与浏览列表相同）。没有关联作品的题放行。指向没有本地 `galgame` 行的链接放行（JOIN 不上）。**一条 SQL 谓词，COUNT 与页查询共用。** 旧 `isSFW` 计算后丢弃的路径删除（修 E1）。

**不可渲染作者（K-G12）**：先按过滤条件取出作者 id 去重，经 `userclient.Users`（缓存）批量判定，**COUNT 与页查询共用同一个「作者可渲染」谓词**（修 E4）。`userclient` 失败 → `503 SERVICE_UNAVAILABLE`，不得 fail-open（修 E2、E3）。详情、作答集合、源面、以及该题上的每一笔写，作者不可渲染都是 `404`。作答集合里不可渲染的作答者丢掉。

**`GET /me/answered-quizzes`**：页码，默认 `limit` 50，排序固定 `a.created DESC, a.id DESC`（`role='answerer'`），没有 `sort`，没有分类过滤（修 E27）。条目是 `QuizSummary`（调用者作答过的题）。作者不可渲染的题按 K-G12 排除，COUNT 与页同一谓词。`userclient` 失败 503。

### 3.5 作答集合（K-G9）

**`GET /quizzes/{quiz_id}/answers`**：游标（`collect.Cursor`：`cursor`、`limit` 1–100 默认 20，`include_total`）。`ORDER BY created DESC, id DESC`。追加流，生产最多 251 人，游标翻完（修 E29）。题目不存在或作者不可渲染 → `404`（修 E11）。游标绑在该 `quiz_id` 上。

答案键不变量见 K-G9：本面不下发 `solution`；`is_correct` 与 `submission` 对未作答且不能看见键的调用者都是 `null`。

### 3.6 写面限额

取自现行 DTO、网页 zod、列宽、生产最长值，取三者最紧且真实的：

| 字段 | 限额 | 来源 |
|---|---|---|
| `prompt_text` | 1–200；K19：先按原始值判长，再去空白；去空白后空 → `TOO_SHORT`。允许 `\|\|spoiler\|\|` | DTO `max=200`；生产最长 164；zod 200（`Form.vue:305` 的 maxlength 2000 是前端 bug，v1 对齐 200，修 E25） |
| `description_markdown` | ≤20000；可空 | DTO / 列 |
| `explanation_markdown` | ≤2000；可空 | DTO / 列 |
| `choices` | `single` / `multiple`：2–20 条，每条 1–200，去空白，请求内唯一；`judge` 不收（有值 → `INCONSISTENT_WITH`） | 生产最多 15、最长 91；旧面无条数上限，按投票选项 20 |
| `correct_choice_indexes` | 0 起的整数；`single` 恰好 1；`multiple` ≥1、唯一、落在 `[0, len(choices))`；`judge` 必须是 `[]` 或缺席 | 旧 `answer` / `answers` |
| `judge_answer` | `judge` 必填 bool；其它题型必须是 `null` 或缺席 | |
| `work_ids` | 0–20，唯一，十进制字符串；每个必须是现存 catalog 作品，一次 `CatalogRowsByWorkIDs` 批查；缺的 → `422 UNKNOWN_REFERENCE` | 生产单题最多 14；旧面不校验（修 E23） |
| `difficulty` | 1–10 | DTO / CHECK |
| `spoiler_level` | 创建缺席视为 `none` | 服务层旧行为 |
| `rating`（质量） | 1–10 整数 | DTO |
| `quiz_ids`（me/favorites） | 1–100 个逗号形 id | T2 `topic_ids` 同一上限 |

类型交叉字段 → `422 INCONSISTENT_WITH`（`errors[].pointer` 指向多余的那一侧）。

`PATCH` 出现即替换该字段；`choices` / `work_ids` 出现即整组替换，缺席不动。`quiz_type` **不可变**：出现且与库存不同 → `422 VALIDATION_FAILED`，`/quiz_type` 上 `IMMUTABLE`（修 E15：今日改类型会跳过重判）。出现且等于库存，当作没改。

K18：只有本次提交的文本走 trust——`prompt_text`、`description_markdown`、`explanation_markdown`、`choices`。只改分类 / 难度 / 剧透 / 关联作品 / `is_work_hidden` / 答案键下标，不跑 trust、不重新入扫描。deny → `422 CONTENT_REJECTED`。hold 仍写入并 `ScanBg`。信任文本经 `markdown.NormalizeStoredContent` 后再拼。

答案键（`correct_choice_indexes` / `judge_answer` / `choices` 的条数或顺序以致下标含义改变，实现上：库存 JSONB 的选项或键有变）变化时，**同一事务**里对每个 answerer 双向重判，并从数据库重算 `correct_count`（对→错也翻，修 E15）。`QuizCorrectReward` 仍是 0，重判不发萌萌点。`gradeQuiz` 失败不得 `continue` 留下旧分，整笔 PATCH 失败。

`work_ids` 批查 catalog 失败 → `503`。hidden / 不存在的 id 一律 `UNKNOWN_REFERENCE`，不区分原因。不要求本地 `galgame` 行（生产 3 条孤儿链接仍可保留，只要 catalog 认这个 work id）。

### 3.7 萌萌点

| 事件 | 分 | 幂等键 | ref |
|---|---|---|---|
| 创建题目 | +2 | `kungal:quiz_create:<id>` | `galgame_quiz:<id>` |
| 删除题目 | −2 | `kungal:quiz_delete:<id>` | `galgame_quiz:<id>` |
| 他人收藏 / 取消 | `liked` ±1 | `kungal:favorited:quiz_favorite_{row}` / `kungal:unfavorited:quiz_favorite_{row}` | `galgame_quiz:<id>`（自己的不给，W4 收藏先例：`engage_favorite.go:35-36,68-69`） |
| 答对 | 0 | 不调用 | — |

键不再含 `KeyNonce`（修 E30）。发分等提交成功之后，经 pusher 异步推送（与 G1、W4 相同；实现时核实，旧稿「OAuth 不可用 → 503」不对：推送不阻塞写面，靠稳定幂等键重试）。收藏键用行的 `id`（迁移 143 加列）：取消再收藏会插入新行、拿到新 id，所以正好再给一次。

答对奖励保持 0。网页「答对可获得萌萌点」删掉（修 E16）。出题 +2 / 删除 −2 的文案保留。

### 3.8 浏览计数（K-G15）

`GET /quizzes/{quiz_id}` 照旧 `view + 1`（话题详情同样在 GET 里计数），**同步**执行，错误上抛 500，不丢在 goroutine 里（修 E9）。列更新不写 `updated`。迁移 143 把 `trg_feed_galgame_quiz` 收窄到函数真正读的列，浏览 / 作答计数 / 质量 / 收藏不再写 `feed_activity`。日表 `viewstats.BumpDaily` 也在这次请求里跑，失败同样 500。

源面、作答集合、列表不加浏览。

### 3.9 文本形状（K-G14）

**`prompt`（读面）**是受限内容文档：只产出 `paragraph`、`text`、`break`、`inline_spoiler`。从 `prompt_text` 按旧 `RenderQuestionPlain` 的语义构建：`||x||` → `inline_spoiler`（正则 `(?s)\|\|(.*?)\|\|`，见 `markdown.go:28`），换行 → `break`，其余是字面 `text`。不跑 Markdown，不解析 `/image/`（生产题干 0 条含该 token，旧 HTML 管线也不解析）。

话题评论的 K20 管线（`internal/topic/apiv1/comment_content.go`）产出 `paragraph` / `text` / `break` / `image` / `mention` / `reply_reference`，**不能**产出 `inline_spoiler`。墙走完整 `Converter`。本轨在 `internal/quiz/apiv1` 用 `content` 包已导出的 `NewDocument` / `NewParagraph` / `NewText` / `NewBreak` / `NewInlineSpoiler` 自建这一步，不改共享管线（§8 O4，编排者可改口去扩展 K20）。

18 条含换行的题干在旧 HTML 里换行会被折叠；v1 的 `break` 保真，与 K20 对 544 条评论换行的处理一致。

**`content`（详情的描述）与 `solution.explanation`** 是完整 Markdown 内容文档，走 `Converter`。生产描述 35 条、解析 8 条看起来像 Markdown。

源面回 `prompt_text`、`description_markdown`、`explanation_markdown`。

### 3.10 作品建议

**`GET /work-suggestions`**：public 档，不看凭证。`q` 必填 1–100（自由文本）；去空白后空 → `TOO_SHORT`。`include_nsfw` 默认 `false` → `client.ApplyWorksGate` 的 `content_limit=sfw`（`nsfw=true` 的人口闸仍开，与今日 picker 相同）。`sort=relevance`，`limit=12`，写死，不另给 `limit` 参数。

一次 `CatalogWorksSearch`，`include` 对齐 `WorkRefOf` 所需（`names,covers`）。每条命中经 `CatalogItemRenderable` 丢掉 hidden claim，然后 `WorkRefOf`。不再二次 `GetBatchDetailPublic`（G0 之后 id 已是 work id，二次查找是普查 18 的根）。至多 12 条。

响应 `{ "object": "list", "items": WorkRef[] }`，不分页，不发 `total`。catalog 失败 → `503`（今日是空数组 200）。不下发 `NamePreference` 挑过的单名：`WorkRef` 带名字三件套。没有 officials（WorkRef 没有这一项；旧 picker 有，见 §8 O7）。

### 3.11 查看者与作答（K-G9、K-G13、K-G15）

答案键不变量，测试钉死：

- 键与解析只发给：已经作答本题的人、作者、或 `viewer.can_edit`。匿名永远没有。
- 列表、作答前的详情、作答前的作答集合，既没有键，也没有从键派生的字段。
- 作答集合里，`is_correct` 与 `submission` 只发给已经作答或能看见键的人；其余人两项都是 `null`。
- 详情：`single` / `multiple` 有 `choices`（无键），`judge` 的 `choices` 是 `[]`。键在 `solution`。
- POST answers 的 201 是调用者的作答加上 `solution`。
- 作答在同一事务里给作者写一条 `quiz-answered` 通知（旧面就有，生产 10,423 条，契约初稿漏了，实现时补回）：`sender` 作答者、`receiver` 作者、`link` `/galgame-quiz/{id}`、同一 sender / receiver / link 已有则不重复写；正文照旧面 `选择「A. 选项」，回答正确`（多选以「、」连接，判断题 `正确` / `错误`）。

作答：

- 作者答自己的题 → `403 SELF_ANSWER_FORBIDDEN`（W4 自赞 / 自推的 `SELF_*_FORBIDDEN` 构词）。
- 第二次作答 → `409 ALREADY_EXISTS`。
- 并发第二笔撞 `UNIQUE (quiz_id, user_id)` → 同一 `409 ALREADY_EXISTS`，不得 500（修 E17）。先查到 author 行走自答码，走不到唯一约束。

质量评分：

- 没有 answerer 行 → `403 QUIZ_ANSWER_REQUIRED`（已注册；描述从「题目墙」拓宽为「需要调用者已经作答本题的动作」，墙与评分共用）。
- 作者 → 同一 `403 QUIZ_ANSWER_REQUIRED`：作者只有 author 行、没有 answerer 行，前提就是不满足。网页不给作者画评分控件（没作答就没有结果页），不另立自操作码（编排者裁，K26）。
- 值是槽位：再 PUT 替换。没有 DELETE（网页没有撤分入口，A5）。
- 平均值在同一事务里从 `quality_sum` / `quality_count` **读库**再算，不在内存里加 delta（修旧 `RateQuizQuality` 的内存加法）。

收藏：K16。`INSERT … ON CONFLICT DO NOTHING RETURNING id` / `DELETE … RETURNING id`，只在真的动了行时改 `favorite_count` 并发分。置已置、撤未置都是 200 无副作用。作者可以收藏自己的题，不给分。

### 3.12 实现时按 G8 / G14 改的名（2026-09-23，只增不改，以此为准）

G8 要求全 spec 里同名属性同型（含可空与 format），G14 要求每个字符串带 enum / format / pattern / 自由文本标记。任务书前文的几个名字撞了既有的面，契约里改成下表：

| 前文 | 实际 | 撞了谁 |
|---|---|---|
| `type` | `quiz_type` | 问题目录条目的 `type`（URI） |
| `category` | `quiz_category` | 话题 / 版块 / 草稿的 `category`（`galgame` / `technique` / `others`）。`category` 在 G8 例外表里，闸看不见词表分歧 |
| 读面 `description`（内容文档） | `content` | 网站、投票、文档、Problem 的 `description` 是 string |
| `options` | `choices` | 投票的 `options` 是对象数组 |
| `hide_galgame` | `is_work_hidden` | F1：布尔以 `is_` / `has_` / `can_` 开头 |
| PUT 质量的 `quality_rating` | `rating`（1–10） | 详情 `viewer.quality_rating` 可空。G1 实用性 PUT 同一先例 |
| 排序 `update_time` / `time` / `view` | `bumped_at_*` / `created_*` / `view_*` | 跟字段名走；`view_desc` 与话题列表同一短名 |

实现时 G8 / F1 / G14 又逼出下面几条（2026-09-23，以此为准）：

| 前文 | 实际 | 为什么 |
|---|---|---|
| 答案键 `judge_answer`、作答 `judge_choice` | 两处都叫 `is_statement_true`（`bool \| null`，非判断题为 `null`） | F1 布尔前缀；同名同型，判分就是两值相等 |
| 详情 `viewer.answer` 是 `QuizViewerAnswer` | `QuizAnswer \| null`（与作答集合同一形状，`QuizViewerAnswer` 删除） | G8：`answer` 在 POST 201 里是 `QuizAnswer` |
| POST 201 的 `answer` / `solution` 不可空 | 可空，这条响应里恒有值 | G8：详情的 `viewer.answer`、`solution` 可空（G1 实用性同一先例） |
| 下标数组 `integer[]` | 元素是具名类型 `ChoiceIndex`（0–19） | G14：数组元素的数字要有 minimum |
| 作答体缺 `is_statement_true` | 可缺席（`required: false`），缺席与 `null` 同义 | 单选 / 多选的客户端不必送判断题字段 |

`prompt`、`solution`、`explanation`、`works`、`difficulty`、`spoiler_level` 现 spec 未占用，保持。`viewer` 子形状按父对象分 Go 类型（G8 例外表）。

## 4. 逐条操作

通用：401 `MISSING_CREDENTIAL` / `INVALID_CREDENTIAL`（required；optional 的坏 Bearer）、403 `ACCOUNT_BANNED`、500 `INTERNAL_ERROR`、503 `SERVICE_UNAVAILABLE`（会话存储 / userclient / catalog / 萌萌点上游）。每个 401 带 `WWW-Authenticate`。v1 全部 `Cache-Control: no-store`。

### 4.1 `listQuizzes` · `GET /quizzes` · optional · 200 `PageList<QuizSummary>`

Query：`page`、`limit`（默认 50）、`work_id`、`author_id`、`quiz_type`、`quiz_category`、`difficulty`、`spoiler_level`、`include_nsfw`、`sort`（默认 `bumped_at_desc`）。

| 状态 | code |
|---|---|
| 400 | `UNKNOWN_ENUM_VALUE` `UNKNOWN_SORT` `LIMIT_TOO_LARGE` `INVALID_PARAMETER`（深度 / id 格式 / `include_nsfw`） |
| 401 | 坏 Bearer |
| 503 | userclient |

### 4.2 `getQuiz` · `GET /quizzes/{quiz_id}` · optional · 200 `Quiz`

副作用：`view_count + 1`，不刷 `updated_at`。

| 状态 | code |
|---|---|
| 404 | 不存在 / 作者不可渲染 |
| 503 | userclient / catalog / 正文用户查找 |

### 4.3 `listQuizAnswers` · `GET /quizzes/{quiz_id}/answers` · optional · 200 `List<QuizAnswer>`

Query：`cursor`、`limit`（默认 20）、`include_total`。

| 状态 | code |
|---|---|
| 400 | `INVALID_CURSOR` `LIMIT_TOO_LARGE` |
| 404 | 同详情 |
| 503 | userclient |

### 4.4 `getQuizSource` · `GET /quizzes/{quiz_id}/source` · required · 200 `QuizSource`

| 状态 | code |
|---|---|
| 403 | `PERMISSION_REQUIRED`（无 `can_edit`） |
| 404 | 同详情 |
| 503 | userclient |

### 4.5 `listMyAnsweredQuizzes` · `GET /me/answered-quizzes` · required · 200 `PageList<QuizSummary>`

Query：`page`、`limit`（默认 50）。

| 状态 | code |
|---|---|
| 400 | `LIMIT_TOO_LARGE` `INVALID_PARAMETER` |
| 503 | userclient |

### 4.6 `listMyQuizStates` · `GET /me/quiz-states` · required · 200 `{object: "list", items: QuizState[], missing: string[]}`

Query：`quiz_ids` 必填，1–100，逗号形（`explode: false`，与 `me/topic-states` 相同）。不分页。`missing` 与 `/me/topic-states` 同义。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（缺席 / 空 / 超过 100 / 非正十进制；与 `/me/topic-states` 相同，实现时按先例改） |

### 4.7 `listWorkSuggestions` · `GET /work-suggestions` · public · 200 `List<WorkRef>`

Query：`q`（1–100）、`include_nsfw`（默认 false）。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（`q` 格式） |
| 422 | `VALIDATION_FAILED`（`q` 过短 / 过长） |
| 503 | catalog |

### 4.8 `createQuiz` · `POST /quizzes` · required · 201 `Location` + `Quiz`

`Idempotency-Key` 必带。体 `QuizCreate`（§3.6）。`Location`：`/api/v1/quizzes/{id}`。写入 author 行。萌萌点 +2。触发器写 `GALGAME_QUIZ_CREATION`。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（缺 / 格式错的幂等键） |
| 409 | `IDEMPOTENCY_KEY_REUSED` `IDEMPOTENCY_REQUEST_IN_PROGRESS` |
| 422 | `VALIDATION_FAILED`（`TOO_LONG` `TOO_SHORT` `TOO_MANY_ITEMS` `TOO_FEW_ITEMS` `DUPLICATE_ITEM` `UNKNOWN_VALUE` `UNKNOWN_REFERENCE` `INCONSISTENT_WITH` `OUT_OF_RANGE`）`CONTENT_REJECTED` |
| 503 | userclient / catalog / 萌萌点上游 |

### 4.9 `updateQuiz` · `PATCH /quizzes/{quiz_id}` · required · 200 `Quiz`

体 `QuizPatch`，全部可选。无 `can_edit` → 403 `PERMISSION_REQUIRED`。K18。`quiz_type` 变更 → 422 `/quiz_type` `IMMUTABLE`。键变化则双向重判。不再回 `{regraded}`（§8 O6）。

### 4.10 `deleteQuiz` · `DELETE /quizzes/{quiz_id}` · required · 204

无 `can_delete` → 403。硬删（作答、收藏、关联、质量票随 FK CASCADE）。萌萌点 −2。

### 4.11 `createQuizAnswer` · `POST /quizzes/{quiz_id}/answers` · required · 201 `Location` + `QuizAnswerResult`

`Idempotency-Key` 必带。体 `QuizSubmission`。`Location`：`/api/v1/quizzes/{quiz_id}`（作答页就是详情；本波不加 `GET …/answers/{answer_id}`，见 §8 O10）。作者 → `403 SELF_ANSWER_FORBIDDEN`。已作答 / 并发唯一约束 → `409 ALREADY_EXISTS`。题目 404。判分在服务端。

| 状态 | code |
|---|---|
| 400 | `INVALID_PARAMETER`（幂等键） |
| 403 | `SELF_ANSWER_FORBIDDEN` |
| 404 | 同详情 |
| 409 | `ALREADY_EXISTS` `IDEMPOTENCY_KEY_REUSED` `IDEMPOTENCY_REQUEST_IN_PROGRESS` |
| 422 | `VALIDATION_FAILED` `CONTENT_REJECTED` |
| 503 | userclient |

### 4.12 `putQuizFavorite` · `PUT /quizzes/{quiz_id}/favorite` · required · 200 `QuizEngagement`

空体。已收藏再 PUT 是 200、计数不变。题目 404。

### 4.13 `deleteQuizFavorite` · `DELETE /quizzes/{quiz_id}/favorite` · required · 200 `QuizEngagement`

未收藏再 DELETE 是 200、计数不变。题目 404。

### 4.14 `putQuizQualityRating` · `PUT /quizzes/{quiz_id}/quality-rating` · required · 200 `QuizQuality`

体 `{ "rating": 1..10 }`。未作答（含作者）→ `403 QUIZ_ANSWER_REQUIRED`。题目 404。

## 5. 预分配

### 5.1 迁移 143 `quiz_v1`

普通迁移，随部署跑，幂等。不是 deploy-then-drop。生产 fill/essay 0 行、收藏 4 行且无重复，下列 DDL 建得起来。

```sql
-- 1. 收藏行有自己的 id，W4 收藏键用得上（K-G13）
ALTER TABLE galgame_quiz_favorite
    ADD COLUMN IF NOT EXISTS id BIGSERIAL;
CREATE UNIQUE INDEX IF NOT EXISTS galgame_quiz_favorite_id_key
    ON galgame_quiz_favorite (id);

-- 2. 作答游标（K-G9 集合）。生产单题最多 251 行。
CREATE INDEX IF NOT EXISTS idx_galgame_quiz_answer_quiz_created_id
    ON galgame_quiz_answer (quiz_id, created DESC, id DESC)
    WHERE role = 'answerer';

-- 3. 只留三种题型（K-G8）。生产 fill/essay 0 行。
ALTER TABLE galgame_quiz DROP CONSTRAINT IF EXISTS galgame_quiz_type_check;
ALTER TABLE galgame_quiz ADD CONSTRAINT galgame_quiz_type_check
    CHECK (type IN ('single', 'multiple', 'judge'));

-- 4. 浏览 / 计数不再写 feed。函数读 NEW.id / NEW.user_id / NEW.question / NEW.created。
DROP TRIGGER IF EXISTS trg_feed_galgame_quiz ON galgame_quiz;
CREATE TRIGGER trg_feed_galgame_quiz
    AFTER INSERT OR DELETE OR UPDATE OF question, user_id, created
    ON galgame_quiz
    FOR EACH ROW EXECUTE FUNCTION feed_sync_galgame_quiz();
```

第 4 步不改函数体，只改触发条件；列表就是函数读的列（2026-09-23 在测试库 `pg_get_functiondef` 核过，与 G1 工具集触发器同一做法）。第 3 步的约束名同日在测试库 `pg_constraint` 核过。

### 5.2 错误码

新增（三处一译：`registry.go` + `registry_test.go` + `zh-CN/problem.json`）：

- **`SELF_ANSWER_FORBIDDEN`**（kungal，403）。作者不能答自己的题。构词同 W4 的 `SELF_LIKE_FORBIDDEN` / `SELF_UPVOTE_FORBIDDEN`；那两条的注册描述钉死在赞和推上，不能复用。

拓宽已有码的描述（不改 code / status / type URI）：

- **`QUIZ_ANSWER_REQUIRED`**（kungal，403，已注册）。现在的描述只谈题目墙。G2 评分「必须先作答」走同一码；描述改为「调用者必须已经作答本题；作者不能作答自己的题」（墙与评分）。这是共享注册表的一处文案，RC 的墙行为不变。

不新增 `SELF_LIKE_FORBIDDEN` 的同义码。其余复用：`NOT_FOUND`、`PERMISSION_REQUIRED`、`ACCOUNT_BANNED`、`SERVICE_UNAVAILABLE`、`VALIDATION_FAILED`（`INVALID_FORMAT` / `UNKNOWN_REFERENCE` / `INCONSISTENT_WITH` / `DUPLICATE_ITEM` / `TOO_SHORT` / `TOO_LONG` / `TOO_MANY_ITEMS` / `TOO_FEW_ITEMS` / `OUT_OF_RANGE` / `IMMUTABLE` / `UNKNOWN_VALUE`）、`CONTENT_REJECTED`、`ALREADY_EXISTS`、`UNKNOWN_ENUM_VALUE`、`UNKNOWN_SORT`、`LIMIT_TOO_LARGE`、`INVALID_PARAMETER`、`INVALID_CURSOR`、`IDEMPOTENCY_KEY_REUSED`、`IDEMPOTENCY_REQUEST_IN_PROGRESS`。

### 5.3 K 决定

| 编号 | 决定 |
|---|---|
| **K-G8** | 只有三种题型：`single` `multiple` `judge`。fill / essay 生产 0 行，网页不能出（K26）。创建 422。判分代码删除。库 CHECK 收到 `single` / `multiple` / `judge` |
| **K-G9** | 答案键不变量：键与解析只发给已作答、作者、`can_edit`；匿名没有。列表、作答前详情、作答前作答集合既无键也无派生。作答集合的 `is_correct` 与 `submission` 对其余人为 `null`。详情用 `choices` + `solution`。POST 201 = 作答 + `solution` |
| **K-G10** | `hide_galgame` → `is_work_hidden`。为真且调用者看不见键时 `works` 是 `[]`；否则 `WorkRef[]`，一次 `CatalogRowsByWorkIDs` + `WorkRefOf`。详情不再按 SFW 藏作品 |
| **K-G11** | 列表 `include_nsfw=false` 默认排除任一关联作品本地 `galgame.content_limit = 'nsfw'` 的题。一条 SQL 谓词，COUNT 与页共用。旧 `isSFW` 丢弃路径删除 |
| **K-G12** | 与 G1 K-G7 相同：列表 COUNT 与页同一「作者可渲染」谓词；详情 / 作答 / 源 / 写面作者不可渲染是 404；`userclient` 失败 503。作答者不可渲染则从作答集合丢掉 |
| **K-G13** | 写面限额见 §3.6。`quiz_type` 不可变。PATCH 只对改过的文本跑 trust。键变则双向重判。萌萌点键稳定，收藏键跟 W4。作者自答 `SELF_ANSWER_FORBIDDEN`；重复作答 409。质量评分先作答（作者没有作答行，同码）、平均值读库 |
| **K-G14** | `prompt` 是受限文档（paragraph / text / break / inline_spoiler），本包装配。`content` 与 `solution.explanation` 是完整 Markdown 文档。源面回 `prompt_text` / `description_markdown` / `explanation_markdown` |
| **K-G15** | 详情 GET `view_count + 1`，同步、不写 `updated`。`viewer` 形状见 §3.2 |

权限不新增。`quiz.edit_any` / `quiz.delete_any` 沿用。

## 6. 网页

切到生成的类型化客户端；手写 `shared/types/galgame-quiz.ts` 改成生成物别名或删除。`legacy-fetch-baseline` 按删掉的调用点下调。

| 文件 | 改什么 |
|---|---|
| `quiz/Container.vue` + `_filters.ts` | `GET /quizzes`；过滤不再传 `all` / `difficulty=0`；`sort_field`+`sort_order` → 一个 `sort` token；默认 `bumped_at_desc`；`limit` 50。删「答对可获得萌萌点」；题型介绍不再提填空 / 问答 |
| `pages/galgame-quiz/index.vue` | SEO description 删「答对题目获得萌萌点」 |
| `quiz/List.vue` | `question_html` 改渲染 `prompt`（`Document.vue`）；`my_status` 改 `viewer.has_answered` / `is_correct`；`user` → `author`；id 是字符串 |
| `quiz/GalgamePanel.vue` | `GET /quizzes?work_id=&limit=12` |
| `user/Quiz.vue` | 出题 tab `GET /quizzes?author_id=`；答题 tab `GET /me/answered-quizzes` |
| `pages/galgame-quiz/[id].vue` | `GET /quizzes/{id}`；SEO 题干从 `prompt` 抽纯文本（`documentPlainText`）；关联名从 `works[].display_name` |
| `quiz/Play.vue` | 作答 `POST …/answers`，必带幂等键，体是 `QuizSubmission`；编辑按钮读 `viewer.can_*`，打开时 `GET …/source`；删除 `DELETE …/{id}` → 204；收藏改 `PUT`/`DELETE …/favorite`；`hide_galgame` 揭晓读 `works`；不再读 `reward_delta` |
| `play/AnswerInput.vue` | 只留 single / multiple / judge；提交 `choice_indexes` / `is_statement_true`（§3.12） |
| `play/Result.vue` | 对错与解析读 `solution` + `viewer.answer`；质量 `PUT …/quality-rating` `{rating}`；删 fill / essay 分支 |
| `quiz/DetailPanel.vue` | `GET …/answers` 游标；未作答时 `is_correct` / `submission` 为 null，正确/错误 chip 不画；作答后才画 |
| `quiz/Form.vue` + `content/Editor.vue` | 创建 `POST /quizzes` 201 的 `id` 导航，幂等键；编辑 `PATCH` 只发改过的字段；题干 maxlength 200；选项 2–20、每条 200；关联 `work_ids` 最多 20；删 fill / essay UI 与「即将实装」 |
| `quiz/GalgamePicker.vue` + `SearchAutocomplete.vue` | `GET /work-suggestions?q=&include_nsfw=`；选项是 `WorkRef`（`id` / 名字三件套 / `cover` / `is_nsfw`） |
| `activity/card/Quiz.vue` + `useMyQuizInteractions.ts` | `GET /me/quiz-states?quiz_ids=` 逗号形，读 `has_favorited`；收藏 `PUT`/`DELETE` |
| `favorite/Toggle.vue` + `Toggle.spec.ts` | 题库调用点改为置位 / 撤销（组件已有 `action` 槽，W4 话题在用） |
| `constants/galgame-quiz.ts` | 词表去掉 fill / essay；过滤 UI 去掉 `all`；`sort` 改 token |
| `validations/galgame-quiz.ts` | 与 §3.6 对齐或删掉，改信 spec |
| `pages/edit/galgame/quiz.vue` + `quiz/Publish.vue` | 创建 201 导航 |

`SELF_ANSWER_FORBIDDEN` 的 zh-CN 译文进 `problem.json`。`QUIZ_ANSWER_REQUIRED` 若文案仍只谈墙，改成通用的「请先作答本题」。按钮显隐读 `viewer.can_*`。

## 7. 变异题（先于实现提交）

种子：至少 7 道题同一 `status_update_time` 瞬间（跨页边界）；其中两道的作者随后在测试里被标成不可渲染；一道 `is_work_hidden`、关联一部作品；一道 `single` 至少 3 个选项、至少 4 个 answerer（含并列 `created`、跨 `limit=2`）；一道他人已答错的题，PATCH 把正确答案改到他选的那项、再改走。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 详情在调用者未作答时仍填 `solution`（旧 `my_answer.answer` 路径） | 匿名 / 未作答 GET 的 `solution` 是 `null`，`choices` 不含下标键（K-G9） |
| 2 | 作答集合无条件下发 `is_correct`（旧行为） | 未作答 GET 每条 `is_correct` 与 `submission` 都是 `null`（K-G9） |
| 3 | 列表条目带上 `solution` 或 `correct_choice_indexes` | `QuizSummary` JSON 没有这些键；抽查 SQL 里的键不得出现在列表响应（K-G9） |
| 4 | 列表 COUNT 不再排除不可渲染作者 | `total` 等于 SQL 里作者可渲染的行数，且等于小 `limit` 走完的条目数 |
| 5 | `userclient` 失败时吞错继续（旧 Hydrate） | 列表 / 详情 / 作答集合 / 源面 → 503，不回占位作者 |
| 6 | 排序去掉 `id` 决胜键 | 小 `limit` 全量遍历与 SQL 逐条相等、无重无漏（种子有并列 `bumped_at`） |
| 7 | `PATCH` 不论字段是否变化都跑 trust | 只改 `quiz_category`、旧题干已进禁用词表 → 200，库里分类已改 |
| 8 | `PATCH` 接受 `quiz_type` 变更 | 改 `single` → `judge` → 422 `IMMUTABLE`，库里 type 未动 |
| 9 | 重判只翻错→对（旧 `regradeAnswers`） | 把正确答案从作答者选的项改走之后，该行 `is_correct` 为假，`correct_count` 等于 SQL 里为真的 answerer 数 |
| 10 | 作答 INSERT 把唯一约束冲突映射成 500 | 已有 answerer 行再 POST → 409 `ALREADY_EXISTS`，无第二行 |
| 11 | 去掉「作者不能答」的检查 | 作者 POST answers → 403 `SELF_ANSWER_FORBIDDEN`，无 answerer 行 |
| 12 | `is_work_hidden` 为真时仍填 `works` | 未作答 GET 的 `works` 是 `[]`；作答后同一调用者拿到非空 `WorkRef[]` |
| 13 | 作答不再给作者写 `quiz-answered` 通知（实现时补） | 作答后 `message` 里恰有一条 sender=作答者、receiver=作者、正文 `选择「正确」，回答正确` |

## 8. 开放问题

| # | 事实 | 裁决 |
|---|---|---|
| O1 | 任务书表头写默认 `limit` 24，并说「以浏览页为准」 | 浏览页 `Container.vue:26` 是 **50**。默认 50 |
| O2 | 任务书创建体用 `category` | 现 spec 里 `category` 是话题三值枚举。改名 `quiz_category`（G8）。过滤参数同名 |
| O3 | 任务书读面描述字段叫 `description` | 现 spec 里 `description` 是 string（网站 / 投票 / 文档 / Problem）。读面改用 `content`（与其它 Markdown 正文同一 `$ref`） |
| O4 | K20 评论管线不能产出 `inline_spoiler`；生产 1 条题干含 `\|\|` | G2 在自己包里用 `content` 的导出构造器装配；不改共享管线。编排者若要扩展 K20，改口即可 |
| O5 | 任务书列表 `viewer` 只有 `has_answered` / `is_correct` | 作者在列表上与未作答同形，旧笔形图标不再从 API 来。要图标是加法（`viewer.can_edit` 或 `is_author`） |
| O6 | 旧 PATCH 回 `{regraded:int}`，表单用来 toast | PATCH 回 `Quiz`（G1）。toast 改「已保存修改」。要回条数是加法 |
| O7 | 旧 picker 有 `banner` / `officials` | `WorkRef` 是封面肖像 + 名字三件套 + `is_nsfw`。选择器改画这些。officials 不另给 |
| O8 | 质量评分「必须先作答」 | 复用 `QUIZ_ANSWER_REQUIRED`，拓宽描述。作者评分同码（编排者裁：不立 `SELF_RATING_FORBIDDEN`） |
| O9 | PUT 体 `rating` 1–10；G1 实用性 PUT `rating` 1–5（尚未合进本 worktree 的 spec） | 跟任务书与 G1 槽位先例用 `rating`。G8 的 `shape()` 不含 min/max，合入后闸仍绿。范围写在 schema 上 |
| O10 | POST answers 201 的 Location | 指向 `/api/v1/quizzes/{quiz_id}`。本波不加单条 GET |
| O11 | 普查发现 18 的 gid / catalog id 撞车 | G0 之后作废。picker / 关联 id 都是 work id |
| O12 | 3 条链接没有本地 `galgame` 行 | 列表 NSFW 谓词 JOIN 不上就放行（K-G11 原文）。写面存在性查 catalog |
| O13 | 收藏批查回什么 | 编排者改裁：`/me/quiz-states`，与 `/me/topic-states` 同形（每个可读 id 一条 + `missing`），不另起「只回已收藏」的形状 |
| O14 | 网页 `Editor.vue` 选项无上限 | v1 上限 20；网页加上 |

---

删旧路由时：13 条、`QuizHandler` 里本段的方法、旧 quiz DTO 中不再被评论墙使用的部分、手写 TS 类型。`legacy_route_baseline` −13。`rg` 证明零调用方，含 `apps/web/server/` 与 `../kungal-apps`。
