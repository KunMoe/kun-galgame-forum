# UP · 更新日志与待办看板

> UP 轨，2026-09-23。`/api/update/**` 的 11 条旧路由，全部由 `UpdateHandler` 独占（普查确认没有别的 handler 共用、没有别的前缀挂它）。迁移号段 170–174。
> 契约与变异题同一个提交，早于实现。

## 1. 普查（生产实测 2026-09-23）

### 1.1 更新日志 `update_log`

| 事实 | 数字 |
|---|---|
| 行数 | 1234，全部由用户 2 写（`user_id` 默认值就是 2） |
| `type` 取值 | `feat` 594 · `fix` 237 · `mod` 141 · `refactor` 128 · `docs` 40 · `styles` 32 · `chore` 30 · `pref` 28 · `sec` 2 · `test` 2；**没有第 11 种** |
| `version` | 最长 8 字符；**1 行带前导空格**（`" 2.16.11"`），其余全是 `x.y.z`；空串 0 行 |
| `content` | 最长 180 字符；空白 0 行；**99 行多行**；图片 token 0 行；含 `http(s)://` 7 行；没有 Markdown 标记 |
| `created` 并列 | 0 组（种子数据里要自己造并列，见 §7） |
| `updated > created` | 23 行（被编辑过） |
| 索引 | `idx_update_log_created (created DESC)` |

### 1.2 待办 `todo`

| 事实 | 数字 |
|---|---|
| 行数 | 260，作者 13 人（用户 2 写了 204 条） |
| `type` × `status` | forum：0→41、1→19、2→171、3→10；patch：0→3、1→4、2→11、3→1 |
| `status × claimed_user_id × completed_time` | 0：44 行全无认领者；**1：19 行无认领者**（075 加列之前认领的）+ 4 行有；2：182 行**全部**有 `completed_time`，其中 16 行有认领者；3：11 行全无 `completed_time`，2 行有认领者 |
| `content` | 最长 920；空白 0 行；30 行多行；**1 行含图片 token**（#202，一张截图，界面上一直显示成 `/image/…` 路径）；26 行含链接 |
| `created` 并列 | 0 组 |
| 首页动态 | `feed_activity` 里 `TODO_CREATION` 260、`UPDATE_LOG_CREATION` 1234，由触发器 `trg_feed_todo` / `trg_feed_update_log` 维护（034、074、076），链接 `/update/todo`、`/update/history` |

### 1.3 旧路由与调用方

| # | 旧路由 | 权限 | 网页调用方 |
|---|---|---|---|
| 1 | `GET /update/history` | 公开 | `components/update/History.vue`（`useKunFetch`，页码 + `total`） |
| 2 | `POST /update/history` | `update_log.create` | 同上 `handleUpdateLogAction` |
| 3 | `PUT /update/history` | `update_log.edit` | 同上 |
| 4 | `DELETE /update/history?update_log_id=` | `update_log.delete` | **无** |
| 5 | `GET /update/todo` | 公开 | `components/update/Todo.vue`（页码 + `status` 过滤 + `total`） |
| 6 | `POST /update/todo` | 任意登录用户 | 同上 `handleTodoAction` |
| 7 | `PUT /update/todo` | 任意登录用户，handler 里限发起者 | 同上 |
| 8 | `POST /update/todo/claim` | `update_log.edit` | 同上 `claimTodo` |
| 9 | `POST /update/todo/complete` | 登录，handler 里限认领者或 `update_log.edit` | 同上 `completeTodo` |
| 10 | `POST /update/todo/discard` | 登录，handler 里按状态分：待处理限发起者，进行中限认领者或 `update_log.edit` | 同上 `discardTodo` |
| 11 | `DELETE /update/todo?todo_id=` | `update_log.delete` | **无** |

- `apps/web/server/**`：零调用。Flutter App（`../kungal-apps`）：零调用；`docs/proj/app-direct-api.md` 没列这些路由。
- 其它消费者：`activity` 域直接查表取 `todo.status` 与 `update_log.version` 给首页卡片（`activity_repo.go` 的 `FetchTodoStatuses` / `FetchUpdateLogVersions`），不经过这些路由，不受影响；网页 `activity/card/Note.vue` 用整数状态查标签，改常量时要给它留整数映射。
- **在途 PR #177**（`feat/todo-release-reopen`，基于旧 master）给本域再加两条旧路由：`POST /update/todo/release`（进行中 → 待处理，限认领者本人）、`POST /update/todo/reopen`（已废弃 → 待处理，新权限 `update_log.reopen`，只给 admin / ren）。本契约把这两条迁移**直接收进 v1**（§3.3），权限键与 #177 同名同归属。#177 先合，本轨 rebase 后连它的两条一起删（基线多降 2）；#177 不合，本轨自己加那个权限键。两种顺序结果相同。

### 1.4 旧面的问题

| 旧面 | 事实 |
|---|---|
| 列表 `ORDER BY created DESC` 无决胜键 | 生产没有并列，但 offset 翻页在插入时错位 |
| `type` 是自由字符串 | 服务端只校验 `required`，词表只在网页 zod 里；`pref` 是 `perf` 的笔误，`styles` 是复数 |
| 待办 `status` 是裸整数 | 0/1/2/3 |
| 状态迁移全是动词路径 | `/claim` `/complete` `/discard` |
| 错误全是 `233` 中文句子 | 包括「状态刚刚发生了变化」的 409 |
| 更新日志写面不查 OAuth 当前封禁 | 会话只在刷新令牌时才知道封禁（RC 教训 ④） |
| 待办无人可删 | `DELETE` 路由在，网页没有按钮；任何登录用户都能建待办，没有人能清理 |
| 更新日志响应只有 `user_id` 裸整数 | 网页从不显示作者 |

## 2. 范围

| 旧 | v1 | 档 |
|---|---|---|
| `GET /update/history` | `GET /api/v1/update-logs`（**游标**） | optional |
| — | `GET /api/v1/update-logs/{update_log_id}`（G17 要求） | optional |
| `POST /update/history` | `POST /api/v1/update-logs` → **201** | required |
| `PUT /update/history` | `PATCH /api/v1/update-logs/{update_log_id}` | required |
| `DELETE /update/history` | `DELETE /api/v1/update-logs/{update_log_id}` → **204** | required |
| `GET /update/todo` | `GET /api/v1/todos`（**游标**） | optional |
| — | `GET /api/v1/todos/{todo_id}`（G17 要求） | optional |
| `POST /update/todo` | `POST /api/v1/todos` → **201** | required |
| `PUT /update/todo`、`/claim`、`/complete`、`/discard`（+ #177 的 `/release`、`/reopen`） | `PATCH /api/v1/todos/{todo_id}`：改内容，或 `{"state": …}` 迁移 | required |
| `DELETE /update/todo` | `DELETE /api/v1/todos/{todo_id}` → **204** | required |

11 条旧路由全删，`legacy_route_baseline` 224 → **213**（#177 先合则 226 → 213）。10 个 v1 操作。

**两个集合都是游标**（K11）：更新日志与待办都是按创建时间追加的流，不是「需要跳到第 N 页」的可浏览结果集。网页从分页器改成「加载更多」。

## 3. 形状

### 3.1 `UpdateLog`（`object: "update_log"`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | id | |
| `change_type` | 封闭枚举 `feat` `perf` `fix` `style` `mod` `chore` `sec` `refactor` `docs` `test` | 旧 `type`。不叫 `type`：Problem 的 `type` 是 URI（G8）；不叫 `category`：话题的 `category` 是另一套词表 |
| `release_version` | string ≤ 20，自由文本 | 旧 `version`。不叫 `version`：`Preferences.version` 是整数（G8） |
| `text` | string ≤ 1000，自由文本 | 旧 `content`。**纯文本**，按 K20 的命名：不是 Markdown，不下发文档树（§4.1） |
| `created_at` / `updated_at` | date-time | |
| `viewer` | `UpdateLogViewer \| null` | `can_edit`（`update_log.edit`）、`can_delete`（`update_log.delete`）；匿名为 `null`；Bearer 恒 `false` |

不发作者：1234 行全是同一个人，网页也从不显示（A5）。以后要加是加法。

### 3.2 `Todo`（`object: "todo"`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | id | |
| `project` | 封闭枚举 `forum` `patch` | 旧 `type`：这条待办属于论坛还是补丁站 |
| `state` | 封闭枚举 `pending` `in_progress` `done` `discarded` | 旧 `status` 0/1/2/3 |
| `text` | string ≤ 1000，自由文本 | 旧 `content`，纯文本 |
| `author` | `UserRef` | 发起者 |
| `claimer` | `UserRef \| null` | 认领者；从没被认领、被放弃、被重新启用、或 075 之前认领的都是 `null` |
| `completed_at` | date-time \| null | 仅 `done` 非空 |
| `created_at` / `updated_at` | date-time | |
| `viewer` | `TodoViewer \| null` | 见下；匿名为 `null` |

`TodoViewer` 的每个标志与 §3.3 的闸**逐条等价**（同一个函数算出来），网页不再自己镜像权限：

| 标志 | 为真当且仅当 |
|---|---|
| `can_edit` | 调用者是发起者，且状态是 `pending` 或 `in_progress` |
| `can_delete` | `update_log.delete` |
| `can_claim` | 状态 `pending` 且 `update_log.edit` |
| `can_complete` | 状态 `in_progress` 且（调用者是认领者 或 `update_log.edit`） |
| `can_discard` | 状态 `pending` 且调用者是发起者；或状态 `in_progress` 且（认领者 或 `update_log.edit`） |
| `can_release` | 状态 `in_progress` 且调用者是认领者 |
| `can_reopen` | 状态 `discarded` 且 `update_log.reopen` |

权限一律 `user.Can`：Bearer 请求永不持有 `update_log.*`，于是 Bearer 的 `can_claim` / `can_delete` / `can_reopen` 恒假，`can_complete` / `can_discard` 只剩「我是认领者 / 发起者」那一支。

### 3.3 待办的状态机（`PATCH /todos/{todo_id}` 带 `state`）

| 从 → 到 | 谁 | 副作用 |
|---|---|---|
| `pending` → `in_progress`（认领） | `update_log.edit` | 认领者 = 调用者 |
| `pending` → `discarded` | 发起者 | — |
| `in_progress` → `done` | 认领者 或 `update_log.edit` | `completed_at` = 现在 |
| `in_progress` → `discarded` | 认领者 或 `update_log.edit` | — |
| `in_progress` → `pending`（放弃） | **只有**认领者本人 | 清空认领者 |
| `discarded` → `pending`（重新启用） | `update_log.reopen` | 清空认领者 |

- `done` 是终态。
- **表里没有的 (当前, 目标) 一律 `409 INVALID_STATE_TRANSITION`**，包括「设成当前值」。`detail` 写当前状态与允许的目标。与 T3 抽奖「设成当前值 → 200」不同，理由：这里每条迁移的执行者规则都不一样，一个空操作无法归属到任何一条规则上；而且「认领一条别人已认领的待办」若回 200，客户端会以为自己认领成功了。客户端收到 409 就重读。
- **先判结构，再判权限**：表里没有 → 409；表里有但调用者不够格 → `403 PERMISSION_REQUIRED`。待办是公开的，409 不泄露任何东西。
- 迁移是带状态守卫的 `UPDATE … WHERE id = ? AND status = ?`（沿用旧面）。守卫没命中（别人抢先一步）→ `409 INVALID_STATE_TRANSITION`。
- `state` **不得**与 `project` / `text` 同时出现 → `422 VALIDATION_FAILED`，`errors: [{pointer: "/state", reason: "INCONSISTENT_WITH"}]`（T3 同款）。
- 075 之前认领的 19 条「进行中、无认领者」：只有 `update_log.edit` 能完成或废弃，没人能放弃（放弃限认领者本人）。

### 3.4 待办的内容编辑（`PATCH /todos/{todo_id}` 不带 `state`）

`TodoPatch`：`project?`、`text?`，只改传来的。

- 状态是 `done` / `discarded` → `409 INVALID_STATE_TRANSITION`（先于权限，同 §3.3）；调用者不是发起者 → `403 PERMISSION_REQUIRED`。管理者**不能**改别人的待办（旧面相同）。
- `text` 按原始值判 `maxLength: 1000`（K19），`NormalizeStoredContent` 之后去空白为空 → `422 VALIDATION_FAILED` `/text` `TOO_SHORT`（`min_length: 1`）。
- trust 检查只对**本次提交且与现值不同**的 `text` 跑（K18）；拒 → `422 CONTENT_REJECTED`，什么都不写；hold → 照写、记日志；写成后后台扫描。
- 空对象 `{}`：权限照判，通过则 200、不写。

### 3.5 写面的公共规则

- 所有写面先按 OAuth 当前记录判调用者（RC 教训 ④）：已封禁 → `403 ACCOUNT_BANNED`；`/users/batch` 失败 → `503 SERVICE_UNAVAILABLE`，什么都不写（K17）。
- `POST /todos`：`Idempotency-Key` **必需**（任何登录用户都能建，是用户内容，K12）；`TodoCreate` = `project`、`text`（同 §3.4 的校验与 trust 检查）；新待办恒为 `pending`、无认领者。
- `POST /update-logs`：`update_log.create`；`Idempotency-Key` 可选（管理内容）；`UpdateLogCreate` = `change_type`、`release_version`（1–20，去首尾空白后为空 → `TOO_SHORT`）、`text`（1–1000，同上）。不跑 trust 检查（旧面也不跑）。
- `PATCH /update-logs/{update_log_id}`：`update_log.edit`，三个字段都可选，只改传来的。
- 两个 `DELETE`：先权限（`update_log.delete`）→ 403，再存在性 → 404。成功 204。
- 两个 `GET` 单条：不存在 → 404。
- 写入的 `text` 先过 `NormalizeStoredContent`（与旧面相同；贴进来的图床地址变成 `/image/<hash>` token，图片 GC 的 reference-ping 只认 token）。
- `Location`：`/api/v1/update-logs/{id}`、`/api/v1/todos/{id}`。

### 3.6 列表

- `GET /update-logs`：`cursor`、`limit`（1–100，默认 20）。排序固定 `created_at DESC, id DESC`，没有 `sort` 参数，没有过滤。
- `GET /todos`：同上，外加 `state`（封闭枚举，缺席 = 不过滤）与 `include_total`（与 `items` 同一谓词）。游标绑定 `state`；换了 `state` 再用旧游标 → `400 INVALID_CURSOR`。

## 4. 逐条裁决

### 4.1 为什么是 `text: string`，不是 `content: ContentDocument`

两张表的正文从来都是纯文本：网页一直用 `<pre>` 按原样显示；更新日志 0 个图片 token、0 个 Markdown 标记。K20 已经定了纯文本字段叫 `text`。

把它们做成 K20 那种「受限文档」要两样东西：纯文本管线（现在只在 `topic/apiv1/comment_content.go`，搬进 `internal/apiv1/content` 是动地基）和编辑用的 `…/source` 操作（每个资源再加一个面）。为了 260 行里的 1 张截图不值得。代价照实写下：#202 的截图仍显示成路径。以后要升级成文档，是加一个 `content` 字段，加法。

### 4.2 取值重命名（迁移 170）

| 旧 | v1 | 行数 | 理由 |
|---|---|---|---|
| `pref` | `perf` | 28 | 笔误（「性能优化」） |
| `styles` | `style` | 32 | 与其余单数 token、与 conventional commits 一致 |
| `" 2.16.11"` | `"2.16.11"` | 1 | 前导空格；v1 写面本来就先去空白 |

**改库，不在边界上翻译**：边界翻译会让库里和接口里永远是两套拼法。迁移 170 是三条幂等 `UPDATE`，随部署自动跑；部署窗口里旧网页会把这 60 行的标签显示成空白，几分钟，无害。`mod`、`sec`、`chore` 不改：它们是缩写不是错字，与 `feat`、`docs` 同一风格。

### 4.3 其余

| 旧面 | 裁决 |
|---|---|
| 状态整数、动词路径 | `state` 封闭枚举 + 一个 `PATCH`（01 §5） |
| 管理者能改别人的待办吗 | 不能（旧面如此）。管理者能认领、完成、废弃、删除 |
| 待处理的待办谁能废弃 | 只有发起者（旧面如此）。管理者要清理用删除 |
| 被封禁用户发的待办 | **不藏**，照常显示作者（旧面如此）。看板是任务列表不是内容流；要清理用删除。与 K17 不冲突：写面照样拒封禁的调用者 |
| 删除没有网页按钮 | 网页按 `viewer.can_delete` 加删除按钮（带确认），更新日志同理 |
| 一张图贴进待办 | 维持现状（§4.1） |

## 5. 预分配

- 迁移：**170** `update_log_change_type_tokens`（§4.2）。
- 权限：`update_log.reopen`（admin / ren，与 #177 同名同归属；#177 先合就复用）。三处镜像：`pkg/perm`、`useCan.ts`、`constants/permission.ts`。
- 错误码：**无新增**。用到 `NOT_FOUND`、`PERMISSION_REQUIRED`、`INVALID_STATE_TRANSITION`、`VALIDATION_FAILED`、`CONTENT_REJECTED`、`ACCOUNT_BANNED`、`SERVICE_UNAVAILABLE`、`INVALID_CURSOR`、`LIMIT_TOO_LARGE`、`UNKNOWN_ENUM_VALUE`、幂等三件。

## 6. 网页

- `components/update/History.vue`：`useCursorList` + 「加载更多」；创建 `POST`（幂等键）、编辑 `PATCH`；按 `viewer` 显示编辑 / 删除。
- `components/update/Todo.vue`：`useCursorList`，`state` 过滤、`include_total` 显示「共 N 项」；所有按钮只看 `viewer.can_*`；认领 / 完成 / 废弃 / 放弃 / 重新启用都是 `PATCH {state}`；409 时提示并重读。
- 两个 Modal：字段改成 `change_type` / `release_version` / `text`、`project` / `text`；zod 词表改用生成的枚举。
- `shared/types/update-log.ts` 删除；`constants/update.ts` 改成 v1 token；`activity/card/Note.vue` 仍收整数状态（首页动态是 X 轨的旧面），给它留一个整数 → token 的映射。
- `tests/api/legacy-fetch-baseline` 下调 7（`useKunFetch` 2 + `kunFetch` 5）。
- `useCursorList` 加一个只读的 `total`（最近一页带回来的 `total`，没有就是 `undefined`）：第一个用 `include_total` 的集合。

## 7. 变异题（先于实现提交）

种子：更新日志与待办各有 **7 行同一 `created` 秒**，跨页边界（limit 2、3 各翻一遍）。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 认领不查 `update_log.edit` | 普通用户 `PATCH {state: in_progress}` → 403，行不变 |
| 2 | 认领不记认领者（`claimed_user_id` 不写） | 认领后响应 `claimer.id` = 调用者，库里同值 |
| 3 | 迁移 `UPDATE` 去掉 `AND status = ?` 守卫 | 仓储层：按过期的「从」状态迁移一条已被改掉的行 → `moved = false`；handler → 409 |
| 4 | 放弃允许 `update_log.edit`（不只认领者） | 非认领者的版主放弃 → 403 |
| 5 | 放弃 / 重新启用不清认领者 | 之后 `claimer` 为 `null`，库里 `NULL` |
| 6 | 重新启用不查 `update_log.reopen` | 版主（有 `edit` 无 `reopen`）→ 403 |
| 7 | `done` 不是终态（允许 `done → pending`） | → 409 `INVALID_STATE_TRANSITION` |
| 8 | 内容编辑放行非发起者的管理者 | 版主改别人的 `text` → 403 |
| 9 | 列表排序去掉 `id` 决胜键 | 两个集合的游标全量遍历与 SQL 排序逐条相等、无重无漏 |
| 10 | `total` 不带 `state` 过滤 | `state=done&include_total=true` 的 `total` = SQL 里 `done` 的行数 |
| 11 | 建 / 改待办跳过 trust 检查 | 拒绝型 checker：`POST` 与改 `text` 的 `PATCH` → 422 `CONTENT_REJECTED`，库里无变化 |
| 12 | 更新日志写面不查权限 | 普通用户 `POST` / `PATCH` / `DELETE` → 403 |
| 13 | 写面不按 OAuth 当前记录判封禁 | 会话里未封禁、OAuth 里已封禁的用户建待办 → 403 `ACCOUNT_BANNED` |
| 14 | `viewer.can_release` 算成「认领者 或 `update_log.edit`」 | 等价性测试：每个 (调用者, 状态) 组合下 `viewer.can_*` 为真 ⇔ 对应 `PATCH` 成功 |

## 8. 实现与验收（2026-09-23，只增不改）

契约没有改动。实现时补的几件事：

1. **`useCursorList` 多了只读的 `total`**：取最近一页带回来的 `total`，后续页不带就保留上一次的值；两条 vitest 钉住。`GET /todos` 是全站第一个真的用 `include_total` 的集合（`collect.Total` 早就在，一直没有消费者）。
2. **更新日志的 `PATCH` 空对象**同待办：权限照判，行不存在仍是 404，存在则 200 不写。
3. 旧的 `canEndClaimedTodo` 单测随 handler 一起删了；它测的规则现在由 `TestV1TodoTransitions` 与 `TestV1TodoViewerMatchesTheGates` 覆盖。

### 8.1 变异（14 条 + 9 拆成两个集合，15 次全杀）

| # | 红的测试 |
|---|---|
| 1 | `TestV1TodoTransitions`：普通用户认领 → 200 |
| 2 | `TestV1TodoTransitions`：认领后 `claimer` 为 `null` |
| 3 | `TestV1TodoTransitionIsGuarded`：`moved=true` |
| 4 | `TestV1TodoTransitions`：非认领者的 admin 放弃 → 200 |
| 5 | `TestV1TodoTransitions`：放弃后 `claimer` 仍是认领者 |
| 6 | `TestV1TodoTransitions`：版主重新启用 → 200 |
| 7 | `TestV1TodoTransitions`：`done → pending` → 200 |
| 8 | `TestV1TodoEdit`：版主改别人的待办 → 200 |
| 9a / 9b | `TestV1UpdateLogsWalk` / `TestV1TodosWalk`：去掉 `id` 后翻页重复 `…510` / `…620`、漏掉 5 行 |
| 10 | `TestV1TodosTotalAndFilter`：`total 13, want 3` |
| 11 | `TestV1TodoTrustCheck`：拒绝型 checker 下建成 201 |
| 12 | `TestV1UpdateLogWritesNeedTheirPermissions`：普通用户与 Bearer 版主建成 201 |
| 13 | `TestV1TodoCreate`：OAuth 里已封禁的会话用户建成 201 |
| 14 | `TestV1TodoViewerMatchesTheGates`：admin / 版主在非本人认领的待办上 `can_release=true` 而 `PATCH` 回 403 |

### 8.2 浏览器（开发栈，本 worktree 的 API 打本轨临时库，真 OAuth 水合）

- 匿名 / 普通用户（发起者）/ 版主（认领者）/ admin 四种身份下，看板每张卡的按钮与 §3.2 的表逐条一致：发起者只有「编辑 / 废弃」，认领者多「放弃」，admin 在已废弃上有「重新启用」而在别人认领的待办上没有「放弃」。
- 实点：普通用户新建（多行）→ 编辑 → 废弃；版主认领（显示「已被 ユキン 认领」）→ 放弃 → 完成另一条 → 筛「已完成」显示「共 2 项」；admin 重新启用 → 删除（确认框）。更新日志：25+3 条时首屏 20 条，「加载更多」到 28 条后按钮消失；新建 → 编辑 → 删除。
- 首页「站务」动态的待办卡仍按整数状态显示「待处理 / 进行中 / 已完成 / 已废弃」。
- 控制台只有两条与本轨无关的：`/api/v1/me/preferences` 503（测试会话的 OAuth 令牌是假的）与顶栏头像的水合不一致（`/topic` 同样复现）。
