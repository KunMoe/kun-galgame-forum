# P · 权限

> 2026-09-23 立。分支 `api-v1/p-perm`，迁移号段 185–189（**本轨不用**）。
> 契约与变异题同一个提交，早于实现。

## 1. 普查

### 1.1 路由与调用方

七条旧路由由三个 handler 独占：`RolePermissionHandler`、`UserPermissionHandler`、`PermissionAuditHandler`（`internal/admin/handler/*permission*.go`），普查确认 `internal/admin` 里别的 handler 不与它们共用 service。

| # | 旧 | 方法 | 档 | 调用方 |
|---|---|---|---|---|
| 1 | `GET /api/perm/mine` | `UserPermissionHandler.GetMine` | authed | `plugins/perm-mine.ts`（universal，每个登录用户拉一次进 `useState('kun-perm-mine')`） |
| 2 | `GET /api/perm/bundles` | `RolePermissionHandler.GetBundles` | **公开** | **无** |
| 3 | `GET /api/admin/role-permissions` | `.GetMatrix` | `RequireAdmin` | `pages/admin/permission.vue` |
| 4 | `PUT /api/admin/role-permissions/:role` | `.Replace` | `RequireAdmin` | 同上：保存循环 + 单角色重置 |
| 5 | `GET /api/admin/user-permissions/:uid` | `UserPermissionHandler.GetView` | `RequireAdmin` | `components/admin/permission/UserPanel.vue` |
| 6 | `PUT /api/admin/user-permissions/:uid` | `.Replace` | `RequireAdmin` | 同上 |
| 7 | `GET /api/admin/permission-audit` | `PermissionAuditHandler.List` | `RequireAdmin` | `components/admin/permission/AuditLog.vue` |

`apps/web/server/` 零调用；`../kungal-apps` 零调用。`docs/proj/app-direct-api.md:42` 提到 `/api/perm/mine`，删路由时同步改。

管理面用的是角色闸 `RequireAdmin` 而不是权限键闸，这是有意的（覆盖永远不能把管理员自己锁在门外），v1 保持：`user.CanAdminister()`。

### 1.2 生产取值（`kungalgame`，2026-09-23）

- `role_permission_override` 27 行：moderator revoke 23、creator grant 3、admin revoke 1。26 个不同的键全部在现行目录里。
- `user_permission_override` **0 行**。
- `permission_audit_log` 7 行，全是 `role` + `replace`，操作人都是 uid 2，2026-07-21 至 2026-08-16。
- 目录（`perm.Catalog()`）59 个键：版主基线 56 + 管理员独有 3。

### 1.3 语义与疑似 bug

1. **`/perm/bundles` 匿名公开当前生效的四个角色权限全集**，含那 27 行真实覆盖——任何访客都能读出版主被撤了哪 23 项。全仓零调用方（网页早已改用 `/perm/mine`）。普查（`census/user.md` §6.2）的结论是删，不是迁。
2. **`/perm/mine` 的 Bearer 保证放错了层。** handler 里一个 `if user.ViaBearer()` 挡着，service 走的是 `perm.EffectiveForUser` → `perm.CanUser`，正是 `bearer_guard_test` 禁止的函数，而守卫的正则只认字面量 `perm.CanUser(`，看不见这一层间接（`census/user.md` §6.1、#19）。
3. **多角色保存是按角色串行的 `PUT`，「版主 ⊆ 管理员」每次只拿一个角色的新值对另一个角色的旧值校验。** 管理员被撤了 X、版主也被撤了 X，想把 X 同时还给两者：先存版主 → 版主有 X 而管理员没有 → 拒；先存管理员也只是换一个角色失败的组合（同一次保存里再夹一个「两者都撤 Y」就两种顺序都存不进去）。网页固定按 creator → moderator → admin 存，中途失败时前面的角色已经写进去了。
4. **校验与写入不在同一个事务里。** service 在事务外读当前覆盖、校验，再由仓储开事务整体替换；两个管理员同时改版主和管理员，「版主 ⊆ 管理员」可以被打破。写频率极低，但这条不变式是权限语义本身。
5. 全部校验失败都是 `400` + 中文句子（`233`）：ren 锁定、级别不够、未知权限、非法 effect、重复、空操作（授予基线本有的 / 撤销基线本无的）、增删自己不持有的键、层级冲突。客户端无法按位置标注。
6. 覆盖行上的 `updated_by` / `updated_at` 网页从没用过；变更历史在审计日志里。
7. 审计列表：`page` / `limit` 静默夹取；操作人经 `users` 旁挂 map 下发，userclient 失败时 map 为空，界面静默退化成「用户 #2」。
8. 写后刷新只发生在处理请求的那个实例上，其余实例最多滞后 60 秒（`StartRefresher`）。不变，写进描述。

## 2. 范围

| 旧 | v1 |
|---|---|
| `GET /api/perm/mine` | `GET /api/v1/me/permissions` |
| `GET /api/perm/bundles` | **删除，不迁**（§1.3 第 1 条） |
| `GET /api/admin/role-permissions` | `GET /api/v1/admin/role-permissions` |
| `PUT /api/admin/role-permissions/:role` | `PATCH /api/v1/admin/role-permissions`，**一次原子地改任意几个角色**（§1.3 第 3 条） |
| `GET /api/admin/user-permissions/:uid` | `GET /api/v1/admin/user-permissions/{user_id}` |
| `PUT /api/admin/user-permissions/:uid` | `PUT /api/v1/admin/user-permissions/{user_id}` |
| `GET /api/admin/permission-audit` | `GET /api/v1/admin/permission-changes`，**页码集合** |

删 7 条，`legacy_route_baseline` 224 → **217**（以合并时 rebase 后重新生成为准）。

## 3. 形状

### 3.1 词表

| 类型 | 词表 | 说明 |
|---|---|---|
| `Permission` | **开放** `permission`，`^[a-z][a-z_]*(\.[a-z][a-z_]*)+$`，≤ 64 | 见 K-P1 |
| `effect` | 封闭 `grant` \| `revoke` | |
| `role` | 封闭 `creator` \| `moderator` \| `admin` \| `ren` | 与 `AccessGrants.roles` 的元素同一词表 |

**K-P1 · 权限键是开放词表，不是封闭枚举。** 键带点号命名空间（`comment.topic.edit`），它们存在库里（覆盖表、审计表）、镜像在网页里，而 F1 要求封闭枚举取值是 snake_case；更重要的是目录会随每个域加面而增长，客户端本来就只该检查自己认识的键。写面由服务端按当前目录校验成员资格（`422 UNKNOWN_VALUE`）。

### 3.2 `PermissionSet`（`GET /me/permissions`）

```json
{ "object": "permission_set", "permissions": ["topic.hide", "trust.review"] }
```

按目录顺序。**逐键经 `user.Can` 计算**，所以 Bearer 请求结构性地得到空数组（§1.3 第 2 条）。

### 3.3 `PermissionOverride`

`{ "permission": Permission, "effect": "grant" | "revoke" }`。读、写、审计三处同一个类型（G8）。不再下发 `updated_by` / `updated_at`（§1.3 第 6 条）。

### 3.4 `RolePermissions` / `RolePermissionMatrix`

```json
{
  "object": "role_permission_matrix",
  "catalog": ["topic.edit_any", "…"],
  "role_permissions": [
    {
      "object": "role_permissions", "role": "moderator",
      "baseline": ["…"], "overrides": [{ "permission": "doc.edit", "effect": "revoke" }], "effective": ["…"],
      "is_locked": false,
      "viewer": { "can_edit": true }
    }
  ]
}
```

- `role_permissions` 固定四项，顺序 creator、moderator、admin、ren。`catalog` 按目录顺序。
- `is_locked`：ren 恒持全部权限，永远 `true`；其余 `false`。
- `viewer.can_edit`：`!is_locked && rank(查看者) > rank(role)`（K9）。网页据此锁列，**删掉手抄的 `KUN_ROLE_RANK` / `kunRoleRank`**，这是 `pkg/perm/rank.go` 在网页的第三份镜像。

### 3.5 `RolePermissionMatrixPatch`（`PATCH /admin/role-permissions`）

```json
{ "changes": [
  { "role": "moderator", "overrides": [ … ] },
  { "role": "admin",     "overrides": [ … ] }
] }
```

- `changes` 1–4 项；每项**整体替换**该角色的覆盖集合（空数组 = 重置为基线）。没列出的角色不动。
- 全部角色在**同一个事务**里按**合并后的新状态**校验并写入；任何一项不过，什么都不写。每个被替换的角色写一条审计。
- 事务内先取 `pg_advisory_xact_lock`，读当前覆盖、校验、写入、写审计都在锁内（§1.3 第 4 条）；`user-permissions` 的 `PUT` 按用户取同一类锁。
- 响应 `200` + 写后的完整 `RolePermissionMatrix`。

**K-P2 · 角色矩阵的写是一次原子 PATCH。** 「版主 ⊆ 管理员」是跨角色的不变式，按角色拆开的写不可能总在某个顺序下合法（§1.3 第 3 条）。

### 3.6 `UserPermissions`

```json
{
  "object": "user_permissions", "id": "4242",
  "roles": ["moderator"],
  "baseline": ["…"], "overrides": [ … ], "effective": ["…"],
  "is_locked": false,
  "viewer": { "can_edit": true }
}
```

- `id` 是该用户的 id（G17：被 `PUT` 寻址的 `/admin/user-permissions/{user_id}` 必须能同路径 GET 到带 `id` 的对象，与 T4 的 `admin_topic` 同理）。
- `roles`：OAuth 当前记录里的角色，只保留四个有级别的角色（`user` 等不参与权限计算）。
- `baseline`：角色层给出的权限（旧名 `role_effective`）。与 `RolePermissions` 同名同义：一层「基线 + 覆盖 → 生效」。
- `is_locked`：目标持有 ren 时 `true`。
- `viewer.can_edit`：`!is_locked && rank(查看者) > rank(目标)`。

`PUT` 请求体 `{ "overrides": [PermissionOverride] }`，整体替换；响应 `200` + `UserPermissions`。

### 3.7 `PermissionChange`（`GET /admin/permission-changes`）

```json
{
  "object": "permission_change", "id": "7",
  "actor": { "object": "user", "id": "2", … },
  "target_role": "moderator", "target_user": null,
  "before": [ … ], "after": [ … ],
  "created_at": "2026-08-16T16:08:55Z"
}
```

- `target_role`（`creator` \| `moderator` \| `admin` \| null）与 `target_user`（UserRef \| null）恰好一个非空。
- 旧的 `action`（`replace` / `reset`）不下发：它就是 `after` 是否为空（`buildAuditRow` 就是这么算的），客户端自己判断。
- 页码集合，`limit` 默认 20，排序固定 `created_at DESC, id DESC`，深度上限 10000。

## 4. 错误

全部管理面：非管理员 → `403 PERMISSION_REQUIRED`（Bearer 恒如此）。`/me/permissions` 只有鉴权档的错误。

写面的校验失败一律 `422 VALIDATION_FAILED`，**位置落在请求里出错的那一项**：

| 情形 | 位置 | reason |
|---|---|---|
| 改 `ren` | `/changes/i/role` | `NOT_ALLOWED_VALUE` |
| 同一个角色出现两次 | `/changes/i/role` | `DUPLICATE_ITEM` |
| 目标角色级别 ≥ 自己 | `/changes/i/role` | `NOT_PERMITTED` |
| 目标用户持有 ren | 参数 `user_id` | `NOT_ALLOWED_VALUE` |
| 目标用户级别 ≥ 自己 | 参数 `user_id` | `NOT_PERMITTED` |
| 未知权限键 | `…/overrides/j/permission` | `UNKNOWN_VALUE` |
| 同一个键出现两次 | `…/overrides/j/permission` | `DUPLICATE_ITEM` |
| 空操作：授予基线本有的 / 撤销基线本无的 | `…/overrides/j/effect` | `NOT_ALLOWED_VALUE` |
| 增删自己不持有的键 | 该键在请求里：`…/overrides/j/permission`；是被删掉的旧覆盖：`…/overrides` | `NOT_PERMITTED` |
| 新状态下版主持有管理员没有的键 | `/changes`，每个键一条，`detail` 写键名 | `INCONSISTENT_WITH` |

判定顺序：结构（角色、重复、未知、空操作）→ 持有 → 层级；前一段有错就只报前一段。

**K-P3 · 委派规则的失败是 `422` + `NOT_PERMITTED`，不是 `403`。** 调用者已经过了管理员闸；被拒绝的是请求体里的某个位置，客户端需要知道是哪一格。这正是注册表里 `NOT_PERMITTED` 的定义（「about the actor, not the field's state」），它此前没有调用点。

其余：

- 用户不存在（OAuth 查无此人）→ `404 NOT_FOUND`。
- OAuth `/users/batch` 不可用 → `503`（K17 的失败关闭：级别比较依赖目标的角色，判据未知时不放行）。审计列表的 `actor` / `target_user` 水合失败同样 `503`（与 T4 的列表一致）。

## 5. 预分配

- 迁移：**无**。
- 错误码：**无新增**。
- K 编号：并行会话会撞号，本轨用 **K-P1…**（同 TS 轨的做法）。
  - **K-P1** 权限键是开放词表（§3.1）。
  - **K-P2** 角色矩阵的写是一次原子 PATCH（§3.5）。
  - **K-P3** 委派失败是 `422 NOT_PERMITTED`（§4）。

## 6. 网页

- `plugins/perm-mine.ts` → `GET /me/permissions`。
- `pages/admin/permission.vue`：读矩阵走 `useApi`；「保存」把全部脏角色放进**一个** `PATCH`；「重置」= 只含该角色、`overrides: []` 的 `PATCH`。
- `Matrix.vue`：列锁改看 `viewer.can_edit`；`UserPanel.vue`：只读改看 `is_locked` / `viewer.can_edit`，`role_effective` → `baseline`；`AuditLog.vue`：页码集合，`actor` / `target_user` 直接是 `UserRef`，「重置」由 `after` 为空推出。
- 删：`shared/types/permission.ts`、`constants/permission.ts` 里的 `KUN_ROLE_RANK` / `kunRoleRank`（若再无引用）。
- `legacy-fetch-baseline` 下调 8（`perm-mine.ts` 1、`permission.vue` 3、`UserPanel.vue` 3、`AuditLog.vue` 1）。
- `docs/proj/app-direct-api.md` 第 42 行改指 `/api/v1/me/permissions`。

## 7. 变异题（先于实现提交）

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | `/me/permissions` 改用 `perm.EffectiveForUser` 计算 | 被个人授予 `topic.hide` 的用户经 Bearer 访问 → `permissions` 为空 |
| 2 | 管理面去掉 `CanAdminister` 闸 | 版主 `GET /admin/role-permissions` → `403 PERMISSION_REQUIRED` |
| 3 | 级别比较 `<=` 写成 `<` | 管理员改 `admin` 角色 → `422 NOT_PERMITTED /changes/0/role` |
| 4 | 接受 `ren` | `changes` 里有 `ren` → `422 NOT_ALLOWED_VALUE /changes/0/role` |
| 5 | 去掉持有检查 | 被个人撤销了 `doc.edit` 的管理员去撤销版主的 `doc.edit` → `422 NOT_PERMITTED` |
| 6 | 每个角色拿**库里的**另一角色校验，而不是合并后的新状态 | 库里管理员与版主都撤了 X；一个 `PATCH` 同时清空两者的覆盖 → `200`，两者都重新持有 X |
| 7 | 逐角色各开事务写 | 两项 `changes` 第二项不合法 → `422`，第一项的覆盖行与审计行都没有写入 |
| 8 | 接受空操作 | 给版主授予其基线本有的键 → `422 NOT_ALLOWED_VALUE /changes/0/overrides/0/effect` |
| 9 | 去掉层级检查 | 只撤销管理员的某个版主持有的键 → `422 INCONSISTENT_WITH /changes` |
| 10 | 目标持有 ren 时照常处理 | `PUT /admin/user-permissions/{ren 用户}` → `422 NOT_ALLOWED_VALUE`（参数 `user_id`） |
| 11 | 审计排序去掉 `id` 决胜键 | 种子里有同一 `created_at` 的审计行跨页；小 `limit` 翻完所有页与 SQL `ORDER BY created_at DESC, id DESC` 逐条相等 |
| 12 | 写后不刷新进程内覆盖表 | `PATCH` 撤销版主的 `topic.hide` 之后，版主立刻 `GET /me/permissions` 不再含它 |

## 8. 实现时对本契约的修正（2026-09-23，只增不改）

1. **`perm.EffectiveForUser` 进了 Bearer 守卫。** v1 不再调用它（`/me/permissions` 逐键走 `user.Can`，委派的「持有」判断用 `Operator.Holds = user.Can`），于是把它加进 `bearer_guard_test.go` 的正则，关掉普查 #19 那条「守卫看不见一层间接」的缝。
2. **`middleware.RequireAdmin` 删除**：它唯一的调用点就是这 5 条旧路由，删完后 `deadcode` 报不可达。
3. 用户层的 `effective` 由刚读出的覆盖行计算，而不是读进程内的 `perm.CanUser` 表（后者在别的实例上最多滞后 60 秒），语义与 `CanUser` 相同，由 `TestUserLayerMirrorsCanUser` 逐键钉住。
4. 422 的描述文字不能写 reason 名（G4 把描述里的大写 token 当错误码查注册表），改用自然语言描述各个位置。
5. `docs/proj/permissions.md` 同步改到 v1 路径。
6. **网页的「待保存」按已保存的状态计数，且只包含查看者能编辑的角色**（浏览器验收抓到，见 §9）。
7. **矩阵 `PATCH` 被拒时保留工作区**：原子写被拒就是什么都没写，旧页面「失败就重新拉矩阵」只会丢掉管理员的勾选。

## 9. 验收记录

**闸**（执行者跑、会话复跑）：`make lint` 零输出；`KUN_REQUIRE_TEST_DB=1 go test -count=1 -p 1 ./...` 全绿（专属库 `kungal_test_p_perm`，rebase 到 UP + TS 之后复跑）；`make openapi` / `gen:api` 无漂移，G8 在三轨合并后的整份文档上复跑通过；`pnpm lint`、`pnpm typecheck`、`pnpm -F web test` 全绿。

**变异**：12/12 变红（表见 PR）。第 11 条（审计排序去掉 `id` 决胜键）靠的是并列行按索引顺序（id 升序）出来、与 `id DESC` 不同；`permission_audit_log` 只有 `created_at DESC` 一条索引，换计划也不会让并列行恰好按 id 降序出来，所以没有像 M 轨那样在测试里删索引强制排序计划。

**浏览器实测**（本分支自起 API :2344 + 网页 :2343，dev 库；无头 Chromium；测试前把 `role_permission_override` 导出快照，测完原样导回、删掉测试产生的审计行）：

- 匿名进 `/admin/permission` → 登录页；普通用户 → 被送回首页；普通用户直打 `GET /api/v1/admin/role-permissions` → `403 PERMISSION_REQUIRED`，`/me/permissions` → 空数组。
- 管理员（rank 3）：矩阵四列里只有创作者、版主两列可编辑（`viewer.can_edit`），管理员与莲两列上锁；管理员自己不持有的 `poll.view_restricted` 那一格按持有规则禁用；`/me/permissions` 返回 58 个键（59 减去管理员角色被撤的一项）。
- **抓到的回归**：初版一打开页面就显示「待保存 26 项调整 · 创作者 3，版主 22，管理员 1」——「待保存」是拿工作区对比**基线**算的，存量覆盖全被当成未保存，连管理员不能编辑的 `admin` 列也进了保存集合。旧页面逐角色 `PUT`，于是 creator、moderator 先存进去、到 admin 那一步报错；换成原子 `PATCH` 后，同一次保存会因为 `/changes/2/role NOT_PERMITTED` **整体被拒，一项都存不进去**。改成对比已保存状态并只收可编辑的角色（§8 第 6 条）之后：打开页面「尚无待保存的调整」，点一格「待保存 1 项调整 · 版主 1」，保存 → `PATCH` 200 → 提示「已保存」，再点回去保存 → 还原。
- 「变更日志」页签：`GET /admin/permission-changes` 200，最新一条是「调整 · 角色 · 版主 · 操作人（用户卡片）· 移除覆盖 编辑文档」。
- 用户管理 → 某个无角色用户的「权限调整」面板：`GET /admin/user-permissions/{id}` 200，「角色：无角色」；勾「隐藏话题」保存 → `PUT` 200 且勾选保持；「重置为默认」→ `PUT` 200 且取消勾选，库里覆盖行归零。
