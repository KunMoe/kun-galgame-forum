# 权限系统：permission-first 授权（2026-07）

> 本仓自有工程笔记（**非** infra 镜像）。记录 kungal 论坛这一层的授权模型：
> 51 个纯论坛权限、7 个 infra 代理操作、两层运行时覆盖（role / user）与审计。
> 初版记录 2026-07 的实现状态（commits `2e574ab7` / `e446d9bf` / `93a2fcbf`）；
> 2026-08 词表随社区原语扩到 51 key，本文档计数已对齐 `perm_test.go` 金表。

## 一、两类权限：谁说了算

论坛的授权分成两类，边界刻意分明——只有第一类由本仓做最终裁决。

**纯论坛权限（51 个，PURE-FORUM）。** 内容审核与站务管理的能力，真值 **完全在** `apps/api/pkg/perm`。每个执行点调用 `perm.CanUser(uid, roles, p)`（不是判角色字符串），resolver 就是这些操作的真闸。它们是唯一进入覆盖系统的权限。

**infra 代理操作（7 个，INFRA-PROXY）。** 论坛只是带着调用者的 token 转发给 infra（编辑引擎 / kun_trust），由 infra 重新判定真正的权限。本地那道门只是 **fail-fast / 可见性镜像**，所以刻意 **停留在 `pkg/role`（`CanModerate` / `CanAdminister`）**、写在 `pkg/perm` 与覆盖系统 **之外**，并在代码里用注释标出它镜像的 infra key。给它们建 `pkg/perm` key 会谎称论坛拥有裁决权——真值在 infra。

**wave 178：从 9 项降到 7 项。** infra 现在自己持有作品的 per-user 归属（`catalog_work.owner_user_id`，由 `galgame.creator_user_id` 回填）并从用户 token 推导编辑能力，于是「提案裁决」与「修订回滚」两道论坛本地门（`decideEntry`、revert 里的内联 `CanAdminister || IsEntityOwner`）被**删除**而非改写：整条编辑链走 Bearer 面，鉴权只有 infra 一处。删掉比留着更安全——原第 4 项当初存在，正是因为上一版镜像漂移过（裸版主能过本地门、点 merge、再被 infra 403）。前端要显示按钮时读编辑面下发的能力投影（`can_review` → `can_decide` / `can_revert`），不再复刻任何本地判定。

| # | 操作 | 论坛本地门（router） | 镜像 infra key | 阈值 |
|---|---|---|---|---|
| 1 | 直发建条目 | `POST /galgame` · `RequireModerator` | `galgame.create` | 版主+ |
| 2 | 提交审核队列 | ~~`GET /admin/galgame/messages`~~(wave 169 退役)· `GET /admin/galgame/submissions` + `POST /admin/galgame/:gid/review` · `RequireModerator` | `catalog.claim.review` | 版主+（**纯查看门**,见 wave 179) |
| 3 | 提案查看 / 队列 | `GET /galgame-edit/queue` · `RequireModerator`；提案详情 `reviewEntry` | `galgame.review` | 版主+ 或条目创建者（**纯查看门**） |
| 4 | Wiki 条目编辑 | `PUT /galgame-{tag,official,engine,series}` · `RequireAdmin` | `galgame.taxonomy.edit_any` | **管理员+**（见下） |
| 5 | Wiki 条目删除 | `DELETE …/:id` · `RequireAdmin` | `galgame.taxonomy.edit_any` | **管理员+** |
| 6 | Wiki 条目回滚 | `POST …/:id/revert` · `RequireAdmin` | `galgame.taxonomy.review` | **管理员+** |
| 7 | Trust 举报收件箱 | `/admin/trust/review-items*` · `RequireModerator` | `trust.queue_access` | 版主+ |

**wave 179：第 2 项从「权威镜像」降为「纯查看门」，条数仍是 7。** 投稿生命周期(建条目 / 认领 / 重新提交 / 撤回 / 我的提交 + 四道审核裁决)整条迁到 Bearer 面：infra 自己按 token 推导 actor 与 site、按 `catalog_work.owner_user_id` 校验归属、按 **token 里的角色** 判 `catalog.claim.review`。于是 `POST /admin/galgame/:gid/review` 上的 `RequireModerator` 只决定「哪个页面能打开」,裁决权全在 infra——和 178 对 `galgame.review` 的处理同一个判断。镜像 key 随之从 `edit.galgame.game.status`(wiki 时代的状态流转门,早已随 wiki 退役)更正为真正被判的 `catalog.claim.review`。**别把这道本地门「对齐 infra」收紧成第二个答案**:权限台改授权后 token 侧即时生效,本地门只会滞后。同波论坛侧删除了 `catalogclient.EditActor` 与 S2S 的 `SubmitWork` / `ActOnClaim`——S2S 面上已没有任何「代表某个人行动」的调用。

**（wave 178 退役：原第 4 项「提案裁决」与第 5 项「修订回滚」。）** 二者的论坛本地门已删除，`POST /galgame-edit/proposals/:id/{amend,merge,decline}` 与 `POST /galgame/:gid/edit/revert` 只做 auth（有会话即有 token 可转发），鉴权全在 infra 的 Bearer 编辑面（`edit.catalog.work.review` + 从 token 推导的 owner）。别按「表里少了两行」再补回本地门。

**(已退役 wave 169:词表写路径与 staff lane 随 wiki 退役整体撤除,以下为历史记录。)** **taxonomy 比 infra 更严，且不许「改回去」。** infra 把 taxonomy 编辑/删除/回滚开给版主+，kungal 刻意用 `RequireAdmin` 收紧到 **admin ⊂ ren**（站长拍板，commit `f819503c`：公开创建、admin-only 编辑/删除/回滚）。CREATE（`POST /galgame-tag` 等）仍对任意登录用户开放。这是有意的策略差异，不是 bug，别在「对齐 infra」时把它松回版主。

> 代码里有两处 7-op 清单：`pkg/perm` 的包注释（权威）与前端 `KUN_PROXY_PERMISSIONS` 只读展示表——后者逐字镜像前者的命名，不得分叉；本表是第三处，三处必须同一个 commit 一起改。`edit.galgame.game.status` 不是第十项：它是「提交审核队列」这一项在状态流转上的又一处镜像门（见表中第 2 行）。

## 二、51-key 目录

标签取自 `apps/web/app/constants/permission.ts`（`KUN_PERMISSION_META`），与 UI 保持一致。除「管理」组两项为 **管理员+**（admin ⊂ ren）外，其余 49 项均为 **版主+**（moderator ⊂ admin ⊂ ren）。声明顺序即目录顺序（`pkg/perm/perm.go`）。

| 分组 | key | 标签 | 阈值 |
|---|---|---|---|
| 话题 | `topic.edit_any` | 编辑任意话题 | 版主+ |
| 话题 | `topic.hide` | 隐藏话题 | 版主+ |
| 话题 | `topic.set_best_answer` | 设置最佳答案 | 版主+ |
| 回复 | `reply.edit_any` | 编辑任意回复 | 版主+ |
| 回复 | `reply.delete_any` | 删除任意回复 | 版主+ |
| 回复 | `reply.pin` | 置顶回复 | 版主+ |
| 评论 | `comment.topic.edit` | 编辑话题评论 | 版主+ |
| 评论 | `comment.topic.delete` | 删除话题评论 | 版主+ |
| 评论 | `comment.galgame.edit` | 编辑 Galgame 评论 | 版主+ |
| 评论 | `comment.galgame.delete` | 删除 Galgame 评论 | 版主+ |
| 评论 | `comment.rating.edit` | 编辑评分评论 | 版主+ |
| 评论 | `comment.rating.delete` | 删除评分评论 | 版主+ |
| 评论 | `comment.website.edit` | 编辑网站评论 | 版主+ |
| 评论 | `comment.website.delete` | 删除网站评论 | 版主+ |
| 评论 | `comment.toolset.edit` | 编辑工具集评论 | 版主+ |
| 评论 | `comment.toolset.delete` | 删除工具集评论 | 版主+ |
| 评论 | `comment.resource.edit` | 编辑资源评论 | 版主+ |
| 评论 | `comment.resource.delete` | 删除资源评论 | 版主+ |
| 评论 | `comment.quiz.edit` | 编辑题目评论 | 版主+ |
| 评论 | `comment.quiz.delete` | 删除题目评论 | 版主+ |
| 投票 | `poll.create_any` | 为任意话题创建投票 | 版主+ |
| 投票 | `poll.edit_any` | 编辑任意投票 | 版主+ |
| 投票 | `poll.delete_any` | 删除任意投票 | 版主+ |
| 投票 | `poll.view_restricted` | 查看受限/匿名投票结果 | 版主+ |
| Galgame | `galgame.ban_resource_publish` | 禁止游戏资源发布 | 版主+ |
| 收藏夹 | `collection.edit_any` | 编辑任意收藏夹 | 版主+ |
| 收藏夹 | `collection.delete_any` | 删除任意收藏夹 | 版主+ |
| 题目 | `quiz.edit_any` | 编辑任意题目 | 版主+ |
| 题目 | `quiz.delete_any` | 删除任意题目 | 版主+ |
| 资源 | `resource.edit_any` | 编辑任意游戏资源 | 版主+ |
| 资源 | `resource.delete_any` | 删除任意游戏资源 | 版主+ |
| 评分 | `rating.delete_any` | 删除任意评分 | 版主+ |
| 工具集 | `toolset.edit_any` | 编辑任意工具集 | 版主+ |
| 工具集 | `toolset.delete_any` | 删除任意工具集 | 版主+ |
| 工具集 | `toolset.resource.edit_any` | 编辑任意工具集资源 | 版主+ |
| 工具集 | `toolset.resource.delete_any` | 删除任意工具集资源 | 版主+ |
| 工具集 | `toolset.upload_bypass` | 向任意工具集上传 | 版主+ |
| 文档 | `doc.create` | 创建文档 | 版主+ |
| 文档 | `doc.edit` | 编辑文档 | 版主+ |
| 文档 | `doc.delete` | 删除文档 | 版主+ |
| 网站 | `website.create` | 创建网站 | 版主+ |
| 网站 | `website.edit` | 编辑网站 | 版主+ |
| 网站 | `website.delete` | 删除网站 | 版主+ |
| 友链 | `friend_link.create` | 创建友链 | 版主+ |
| 友链 | `friend_link.edit` | 编辑友链 | 版主+ |
| 友链 | `friend_link.delete` | 删除友链 | 版主+ |
| 更新日志与待办 | `update_log.create` | 创建更新日志与待办 | 版主+ |
| 更新日志与待办 | `update_log.edit` | 编辑更新日志；认领待办，并完成 / 废弃任何进行中的待办 | 版主+ |
| 更新日志与待办 | `update_log.delete` | 删除更新日志与待办 | 版主+ |
| 管理 | `admin.dashboard` | 管理总览与统计 | 管理员+ |
| 管理 | `user.purge_content` | 清除用户全部内容 | 管理员+ |

编译期 bundle（`Bundles`）只列三个可授予的管理角色：`moderator`（49 项）、`admin`/`ren`（51 项，由 `moderatorPerms` 追加两个 admin-only key 组合而来，`moderator ⊂ admin` 结构性成立）。`user` / `creator` 及任何未知角色解析为空。`perm_test.go` 逐格 pin 住这张金表。

## 三、CanUser 解析顺序与不变量

`perm.CanUser(uid, roles, p)`（`pkg/perm/user_overrides.go`）是唯一的执行入口，顺序固定：

1. `uid <= 0`（匿名 / 无身份）→ 退化为纯角色判定 `Can(roles, p)`。
2. `roles` 含 `ren` → 返回 `IsKnownPermission(p)`（**全目录**），防御性 pin 住，任何 role/user 覆盖都锁不住站长。
3. 该用户存在个人 **revoke** → `false`（压过任何角色授予）。
4. 该用户存在个人 **grant** → `true`（即便是无角色的普通用户也能持有）。
5. 否则 → 角色层判定 `Can(roles, p)` = 编译期基线 ± role 覆盖。

`Can`（`perm.go`）是底层角色原语，供 bundle / 测试 / 管理矩阵使用；执行点必须调 `CanUser`，个人覆盖才会被兑现。

**不变量：**

- **ren 处处 pin 全目录**：写入路径（`validateReplace` 拒绝 `role==ren`、`validateUserReplace` 拒绝 ren 持有者）、应用路径（`applyOverrides` 对 ren 短路到 `fullCatalogSet`；`buildResolver` 恒纳入 ren）、`CanUser`（第 2 步）三处都 pin。`TestCanUserRenImmunity` 证明即使塞入「撤销全部」的个人行，ren 仍持 51 项。
- **moderator ⊆ admin 在写入时按「拟态」校验**：`validateReplace` 用 `EffectiveSet`（纯函数，不依赖已安装的全局表）对拟提交的状态计算 `effective(moderator)` 与 `effective(admin)`，前者不被后者包含即 400。
- **`user` 角色隐式且缺席**：OAuth 契约里从不作为 claim 下发，`editableRoles` 不含它，永不可管理。
- **`creator` 可管理但默认为空**：`editableRoles` 含 `creator`，`EffectiveBundles` 恒返回 `creator`（默认 `[]`）——管理员可给它授予 key，但基线无任何权限。

## 四、Delta 模型

两张覆盖表都 **只存偏离量**：

- `role_permission_override`（migration 062）：一行一个 `(role, permission, effect)`，`effect ∈ {grant, revoke}`。`effective(role) = (baseline ∪ grants) − revokes`。
- `user_permission_override`（migration 063）：一行一个 `(user_id, permission, effect)`。`effective(user) = ((role baseline ± role 覆盖) ∪ 个人 grants) − 个人 revokes`。

**空表 = 编译期基线**（金表原样通过所有测试）。**重置 = 删除该 subject 的所有行**（PUT `overrides: []`）。**写入时拒绝 no-op 行**：role 路径以 **编译期基线** 为参照（`BaselineHas`——授予基线已有的、撤销基线没有的均 400）；user 路径以 **目标的角色派生有效集** 为参照（`perm.Can`，刻意不含个人 delta，因为要算的正是相对角色集的 delta）。

## 五、运行时机制

`pkg/perm` 用 `atomic.Pointer` 持有两张「有效表」（`current` 角色层、`userCurrent` 用户层），`SetOverrides` / `SetUserOverrides` 原子换入新表，`CanUser` 读时无锁，与刷新无竞态。

`PermissionOverrideSync`（`internal/admin/service/permission_override_sync.go`）是刷新 **两层** 的唯一 `Load` 路径——**启动加载 + 60s 后台刷新器 + 每次写入直通（write-through）** 都走它，内存有效表从一条代码路径收敛。`Load` 是 fail-atomic：任一层读失败就原样返回错误、**不触碰** `pkg/perm`，沿用上一份有效集。

**失败姿态：告警、保留上一份已知/基线、绝不阻塞请求。** 启动加载失败 → 告警并留用编译期基线（安全已知态，`app.go`）；刷新器失败 → 告警并留用上一次有效集，永不 crash；写入后 `Load` 失败 → 只是内存短暂陈旧（刷新器会收敛），仍返回按新写入行构造的视图。

## 六、管理面

`/admin/permission`（`pages/admin/permission.vue`，`middleware: 'admin'`）两个 tab：**权限矩阵**（`Matrix.vue`：51 行 × creator/moderator/admin/ren 四列，勾选即相对基线的 grant/revoke，小圆点标偏离；ren 列锁定只读）与 **变更日志**（`AuditLog.vue`）。矩阵下方另有 `ProxyList.vue` 只读列出 7 个 infra 代理操作（可见但不可覆盖）。保存为「每个脏角色一次 PUT，整体替换该角色覆盖集」。

**每用户「权限调整」面板**（`UserPanel.vue`）挂在 `/admin/user` 的 `UserCard` 上：以 `baseline`（角色派生集，旧名 `role_effective`）为偏离参照，`PUT /api/v1/admin/user-permissions/{user_id}` 发送工作集相对该参照的 delta 作为个人覆盖全集（replace 语义）；ren 持有者（`is_locked`）或级别不低于操作者（`viewer.can_edit = false`）时整面板只读。

**为何管理面用角色门（`user.CanAdminister()`）而非权限键**：覆盖系统绝不能把管理员锁在「修复覆盖的那个界面」之外（self-lockout 防护）。所以这套 meta-surface（`/api/v1/admin/role-permissions`、`/api/v1/admin/user-permissions/{user_id}`、`/api/v1/admin/permission-changes`）刻意坐落在可覆盖系统 **之外**，与 infra 代理镜像同类——见 `internal/permission/apiv1/ops.go` 的 `requireAdmin`。2026-09-23 起（P 轨）角色矩阵的写是**一次原子 `PATCH`**：多个角色在同一事务、同一把 advisory 锁下按合并后的新状态校验并写入（旧的逐角色 `PUT` 在「同时把一个键还给版主与管理员」时两种顺序都存不进去）。

## 七、审计

`permission_audit_log`（migration 064）append-only，应用层从不 UPDATE/DELETE。每次 role/user 替换都 **在同一事务里** 写 **恰好一行**（先在删除前 `SELECT` 捕获 before 行，再删、插、写审计——`repository/permission_audit_repo.go` 的 `writeAudit`），审计与变更永不漂移。字段：`subject_kind ∈ {role,user}`、`subject`（角色名或十进制 uid）、`action`（新集为空=`reset`，否则=`replace`）、`before_rows`/`after_rows`（紧凑 jsonb `{permission,effect}` 数组）。经 `GET /api/v1/admin/permission-changes`（页码集合，`created_at DESC, id DESC`）读出，`actor` / `target_user` 用 OAuth 批量水合为 `UserRef`，OAuth 不可用时整页 `503`。

## 八、前端镜像契约

- **静态表须与 `pkg/perm` 逐字一致**：`useCan.ts` 的 `MODERATOR_PERMISSIONS`（49）+ `ADMIN_ONLY_PERMISSIONS`（2）是编译期基线的手抄镜像，字符串即 wire 契约，必须与 `pkg/perm` lockstep。`constants/permission.ts` 的标签表按 `ForumPermission` 类型约束，少一个/多一个 key 会在 **构建时** 报错——标签永不会静默漂移。
- **运行时可见性走 `GET /api/v1/me/permissions`**：`perm-mine.ts`（universal 插件，非 `.client`）为 **每个登录用户** 各拉一次，写入 `useState('kun-perm-mine')`。因为个人 grant 可能落在无角色账号上，所以不再「roles 为空就跳过」；只有匿名访客跳过。`useCan` 优先读这份已折叠了 role 层 + 个人 delta 的有效表，未就绪/失败时退回静态角色表。它逐键经 `user.Can` 计算，所以 Bearer 请求结构性地得到空列表。原来的公开 `GET /perm/bundles` 零调用方且把真实覆盖配置暴露给匿名访客，2026-09-23 删除。
- **前端 gating 仅 UX**：真正边界是后端（`RequirePermission` → `CanUser`，同样应用覆盖）。组件分支于具名 **能力**，从不判角色 tier。
- **审核工作台消费投影而非镜像 infra bundle**：编辑提案的评审 UI 读接口下发的 `can_review` / `can_decide` 投影（见特殊判定点），不在前端复刻 infra 的 bundle。

## 九、特殊判定点

- **`reviewEntry` 只剩查看门**（`galgame/handler/edit_handler.go`）：**查看**面（ProposalDetail、队列）= 版主+ 或条目创建者；**裁决**面（Amend/Merge/Decline）与**回滚**面在 wave 178 之后**没有任何论坛本地门**——infra 从用户 token 推导角色与作品归属，直接判。曾经的 `decideEntry` 分裂（裁决 = 管理员+ 或创建者）就是为修一个真实 bug 而生：裸版主能过本地门、点 merge、再被 infra 的 `edit.galgame.game.review` 403；把判定收归 infra 一处后，那个 bug 类别整体消失。`can_decide` / `can_revert` 现在**由投影算出**：前者要求提案 effective patch（无则 patch）的每个 key 在调用者的投影里 `can_review=true`（空 patch → false），后者要求所有未锁定、未废弃字段都 `can_review=true`（无 token → false）。所以按钮只在其背后的写真的会成功时才出现。
- **共享评论编辑路由的 anchor→surface 解析**（`community_comment_write.go` 的 `resolveModEdit`）：一条编辑路由被 galgame/rating/website/toolset 四个评论面共用，各有自己的 `comment.*.edit` key（运行时覆盖可令其分化），故须按 **本帖的 surface** 判权而非取并集。它花一次 S2S 读拿到 anchor（`site_game` → galgame；`site_resource` + `rating:`/`website:`/`toolset:` 前缀 → 对应面），映射到对应 key 后 `CanUser`；anchor 解析不出时 **回退到四个编辑 key 的防御性并集**，权限既不静默放宽也不静默丢失（非版主作者仍走 EditPost 的作者匹配，不受影响）。
- **投票受限结果**：`poll.view_restricted` 经 `CanUser` 判定，决定是否可看受限/匿名投票的结果与投票人记录（`topic/service/poll_service.go`）。
- **清除内容的目标护栏**：operator 的授权由 router 的 `user.purge_content` 门负责；`purge_service.go` 里那道 `role.CanModerate(target.Roles)` 是对 **目标** 状态的护栏（绝不清除版主/管理员的内容——含站点文档/更新日志），属身份/能力属性，故仍停留在 `pkg/role`，不是 `pkg/perm` 操作判定。OAuth 查不到目标身份时 fail-safe 拒绝。

## 十、运维

- **migrations 062 / 063 / 064**：全为 additive + 幂等（`IF NOT EXISTS`），**部署时自动执行**（不在 migrate `--exclude` 名单）。map 表由 infra 侧填充无关。无需手动 `--only`。
- **陈旧窗口**：手动 psql 改覆盖表 → 最多 60s 后由刷新器收敛（无需重启）；前端可见性随每次页面加载刷新（`perm-mine` 插件）；后端执行则是 write-through **即时** 生效（经管理面写入时同事务直通 `Load`）。
- **扩展路径（预留，未实现）**：可管理角色目前是固定白名单 `{creator, moderator, admin}`，角色本身来自 OAuth 的 named role claim。phase-2 设想的「自定义站点角色」需要 OAuth 侧先支持新角色下发，本仓的 role 覆盖层与矩阵可自然承接——但目前尚未实现。

## 关键文件索引

| 关注点 | 文件 |
|---|---|
| 词表 + 金表 + `Can` | `apps/api/pkg/perm/perm.go` |
| 角色覆盖层（set 代数） | `apps/api/pkg/perm/overrides.go` |
| 用户覆盖层 + `CanUser` | `apps/api/pkg/perm/user_overrides.go` |
| 校验 / 直通 / 读模型 | `apps/api/internal/admin/service/{role,user}_permission_service.go` |
| 单一 Load 路径 | `apps/api/internal/admin/service/permission_override_sync.go` |
| 事务内审计写入 | `apps/api/internal/admin/repository/permission_audit_repo.go` |
| v1 管理面 + self-lockout 注释 | `apps/api/internal/permission/apiv1/*.go` |
| Require* 中间件（旧路由） | `apps/api/internal/middleware/role.go` |
| 编辑提案查看门 + 能力投影 | `apps/api/internal/galgame/handler/edit_handler.go` |
| 评论 anchor→surface 判权 | `apps/api/internal/galgame/service/community_comment_write.go` |
| 前端静态镜像 + `useCan` | `apps/web/app/composables/useCan.ts` |
| 标签 / 分组 / 7 代理项 | `apps/web/app/constants/permission.ts` |
| `/me/permissions` 拉取插件 | `apps/web/app/plugins/perm-mine.ts` |
| 管理页 + 矩阵/用户/审计/代理 | `apps/web/app/pages/admin/permission.vue`、`components/admin/permission/*.vue` |

## 差异说明

曾存在两处 9-op 枚举命名不一致（前端展示表一度自造 `galgame.edit.submit` 等展示名），2026-07 已将 `KUN_PROXY_PERMISSIONS` 对齐到 `pkg/perm` 包注释的权威命名。wave 178 把清单降到 7 项（裁决 / 回滚两项的本地门删除），三处清单同 commit 同步，本表与代码现已一字不差。真正的执行门在 `router.go` 与 `edit_handler.go`，以表一为准。
