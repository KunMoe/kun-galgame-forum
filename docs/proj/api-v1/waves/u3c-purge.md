# U3c · 管理员清空用户内容 + 清空存档

> 用户轨最后一段，破坏性。旧路由 **2 条**，都是 `PurgeHandler` 的方法：`GET /api/admin/user/:id/content-stats`、`DELETE /api/admin/user/:id/content`。迁移号段 120–129，本段用 **120**。
> 用户裁决（2026-09-24，经协调会话 forum-7e 转达）：
> ① **清空存档**：与清空同一个事务，把清空删掉的每一行写进存档表，撤销一次清空变成一条查询。每行存表名、主键、整行 jsonb、目标用户、操作者、清空 id、时间；保留 30 天，应用内 cron 清理；恢复 SQL 写进本文并有测试（清空 → 从存档恢复 → 行与原来相等）；中途失败的清空既不删任何行、也不留任何存档行。
> ② **网页确认**要操作者手输目标用户 id，并提示先封禁或注销账号。
> 起因：2026-09-23 生产两次误清（用户 1484、104136）。104136 那次清空比注销早 20 秒，账号在这 20 秒里还在发帖；清空没留任何记录，恢复靠 dead tuple 取证（记忆 `prod-accidental-purge-restore`）。
> 契约与变异题同一个提交，早于实现。

## 1. 普查（2026-09-24）

### 1.1 调用方

| 位置 | 调用 |
|---|---|
| `apps/web/app/components/admin/UserCard.vue` | 两处 `kunFetch`：`/admin/user/{id}/content-stats`（「查看内容」）与 `DELETE /admin/user/{id}/content`（「一键清除全部内容」，确认框是 `useComponentMessageStore().alert`，**不要求输入任何东西**） |
| `apps/web/app/pages/admin/user.vue` | 列表页，页面级 `permissions: ['user.purge_content']`；已有一个指向账号中心封禁/注销的 `KunInfo` |
| `apps/web/server/` | 零 |
| `../kungal-apps` | 零 |

### 1.2 旧面行为

- 权限：两条都挂 `RequirePermission(perm.UserPurgeContent)`。`user.purge_content` 只在 admin 包里（`pkg/perm/perm.go:143`），版主没有。
- 预览：`counts()` 的每个 `Count` **丢弃错误**，查询失败显示 0；社区帖子数失败同样静默 0。
- 清空（`purge_service.go`）：
  1. OAuth 查目标（走 userclient **~10 分钟热缓存**）；查询出错 → 拒绝；持版主能力 → 拒绝；查不到 → 当作已消失的普通账号，可清。
  2. 本地一个事务：计数器重算、私聊房整房删除、网站条目转交给列默认的导入占位账号（#200）。
  3. catalog 收藏夹：要操作者本人的令牌，**令牌为空时静默跳过**。
  4. community `AuthorPurge`。
  5. 回整份统计；错误一律 `233` 中文。
- **没有任何删除记录。** 日志只有一行 `purge: user content purged … local_total`。
- 不看抽奖状态。T4 的话题硬删发现过「级联删掉进行中的抽奖」这个洞（`409 LOTTERY_DRAWN` + 退托管）；用户清空删的是同一批表，没有这层保护。

### 1.3 覆盖面：对照真实 schema

临时库（bootstrap 后）与生产（只读）的列集合、外键图、触发器**完全一致**；生产只多一张一次性表 `_user_merge_109310_to_1922`（`tbl/col/row_id/moved_at`，13 行，无用户列）。

记忆里「清单只覆盖一半」是 2026-08-30 的审计；之后 `461f2e437`（purge every table a user owns）与 #200 补齐了绝大部分。按列名扫出的全部用户列与现状：

| 表.列 | 现在的处理 | 生产 | 本段裁决 |
|---|---|---|---|
| 话题、回复、评论、评分、资源、工具、工具资源、投票、抽奖、草稿、题目、收藏夹、待办、私聊、通知、`galgame_activity`、`galgame_contributor`、`user_permission_override.user_id`、`feed_activity`、`kungal_user_state`、`system_message*`、`user_follow`、`user_friend`，以及 24 张互动表的 `user_id` | 删除 | — | 不变 |
| `galgame_website.user_id` | 转交给列默认账号（#200：共享条目） | 默认值 2 | 不变 |
| `todo.claimed_user_id` | 进行中的认领（`status = 1`）放回待认领；已关闭的保留 | 262 条中 20 条有认领人 | 不变 |
| `chat_message.receiver_id` | 随私聊房整房删除 | 9,964 条，全是私聊 | 不变 |
| `chat_room.last_message_sender_id` | 刷新成房间里剩下的最后一条 | — | 不变 |
| **`toolset_upload.user_id`** | **漏**：只在他自己的工具被删时级联；传到别人工具里的上传记录留下 | 0 行 | **补删除** |
| `topic_lottery_code.claimed_by` | 保留 | 0 行 | 豁免：已兑出的码。放回去会让一个已经给出去的 key 被第二个人领走 |
| `topic_access_grant`（`subject_type = 'user'`） | 保留 | 0 行 | 豁免：别人话题上的访问名单，是那个话题作者的设置 |
| `galgame.creator_user_id` | 保留 | 16,239 行中 13,495 行非空 | 豁免：catalog 共享行，只是署名；`published` 是粘性 SEO 标记（方案③） |
| `topic_comment.target_user_id` | 保留 | 3,169 | 豁免：别人的评论 |
| `doc_article.author_id`、`update_log.user_id` | 保留 | 各 1 个作者（管理员） | 豁免：受保护的 staff 内容 |
| `permission_audit_log.operator_id`、`role_permission_override.updated_by`、`user_permission_override.updated_by` | 保留 | — | 豁免：审计 |

### 1.4 清空实际改动的行不止清单上的表

一条 `DELETE FROM topic WHERE user_id = ?` 会沿外键级联删掉**别人的**回复、评论、赞、表情、投票、抽奖……清单上 20 来条语句，实际触及 60 多张表。三条 `ON DELETE SET NULL` 会改写**别人的行**：

| 外键 | 生产非空行 |
|---|---|
| `topic.best_answer_id → topic_reply` | 104 |
| `topic.pinned_reply_id → topic_reply` | 25 |
| `topic_comment.parent_comment_id → topic_comment` | 1,292 |

另外 15 个 `feed_sync_*` AFTER 触发器维护 `feed_activity`：删内容会连带删首页动态行，网站转交会把动态行的 `user_id` 改成占位账号。清空自己的 `UPDATE`（计数器重算、待办放回、私聊房最后一条、网站转交）同样改写别人的行。

**所以存档必须捕获「这个事务改动的每一行」，而不只是清单上那几条语句删的。** 手写一份级联表清单就是在重复「清单只覆盖一半」的错误。

### 1.5 远端部分不在论坛库里

- community `AuthorPurge`（infra `community/repository/author.go`）把帖子**正文清空**（`content_raw = ''`、`content_html = ''`，status 置删除），并删掉反应、阅读状态、订阅、通知。在 infra 侧不可逆。
- catalog `DELETE /v2/moderation/users/{id}/folders` 删收藏夹。

两者都不在论坛库、不在论坛事务里，**存档覆盖不到**。本段只负责论坛库；远端的可逆性是 infra 的后续（§12）。

### 1.6 规模

生产上单个用户最多：资源 9,131、收到的通知 49,380、首页动态 9,821。最坏的一次清空约 8 万行进存档（jsonb，量级 100–200 MB，保留 30 天）。典型的 spam 账号是几百行。生产 API 容器（2026-09-24 03:30 创建）以来没有清空记录。

## 2. 去向

| 旧路由 | v1 | operationId |
|---|---|---|
| `GET /api/admin/user/:id/content-stats` | `GET /api/v1/admin/user-contents/{user_id}` | `getUserContent` |
| `DELETE /api/admin/user/:id/content` | `DELETE /api/v1/admin/user-contents/{user_id}` → **204** | `purgeUserContent` |

- 两条都是 `required` 档，要 `user.purge_content`，一律 `user.Can`：Bearer 请求永不持有。匿名 → 401；缺权限（包括版主）→ `403 PERMISSION_REQUIRED`。
- 路径：G17 要求被 DELETE 寻址的路径有同路径的 GET 且回带 `id` 的对象（T4 的教训）。形状照 P 轨的 `/admin/user-permissions/{user_id}`：「某个用户的 X」挂在 `/admin/<x>/{user_id}`，对象的 `id` 就是用户 id。不用 `/admin/users/{user_id}/content`——那要先有 `GET /admin/users/{user_id}`。不用 `DELETE /admin/users/{user_id}`——读起来是删账号，而账号归 OAuth。
- `user_id`：`pattern ^[1-9][0-9]{0,18}$`，同 P。**超过 int32 的值 → `404 NOT_FOUND`**（用户 id 在每张表里都是 `integer`，传下去会在驱动里溢出成 500）。
- `legacy_route_baseline` 下调 **2**；`PurgeHandler` 连同 struct、构造、`app.go` 字段、`dto/purge_dto.go` 一起删。

## 3. 形状

### 3.1 `UserContent`（`object: "user_content"`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `object` | `"user_content"` | |
| `id` | 十进制字符串 | 用户 id |
| `is_protected` | bool | 目标持有版主能力（`role.CanModerate`，OAuth 当前记录）。为 `true` 时 `purgeUserContent` 回 `403 USER_PROTECTED`；网页据此禁用按钮 |
| `is_account_active` | bool | 账号中心报告账号可用（找得到且 `status = 0`）。封禁、注销、查不到都是 `false`。网页据此加重「先封禁或注销」的提示（裁决 ②） |
| `topic_count` `reply_count` `topic_comment_count` `rating_count` `resource_count` `toolset_count` `toolset_resource_count` `poll_count` `lottery_count` `draft_count` `quiz_count` `collection_count` `todo_count` `chat_message_count` `message_count` `interaction_count` | int ≥0 | 与旧 18 个统计同义（`chat_message_count` = 他发的私聊，`message_count` = 他发出或收到的通知，`interaction_count` = 24 张互动表的行数和） |
| `website_count` | int ≥0 | 他收录的网站。**转交、不删** |
| `community_post_count` | int ≥0 \| **null** | community 里他的可见帖子数；community 不可用时 `null`（旧面静默 0），并打 WARN |
| `total_count` | int ≥0 | 以上本地计数之和（不含 `community_post_count`，同旧面） |

计数查询出错 → `500`（旧面静默 0）。计数是预览，不是存档的行数：存档还会收下级联删掉的别人的行与被改写的行（§1.4）。

### 3.2 清空的回应

`204`，无 body（01 §5 删除；T4 同）。网页的成功提示用预览里的 `total_count`。清空 id 不下发：恢复是数据库操作，操作者按 `target_user_id` 在存档里找（§6）；它在日志里（§4 第 6 条）。

## 4. 清空语义

1. 权限闸（§2）。
2. **新鲜地**查目标：先 `userclient.Invalidate(id)` 再查。旧面走 10 分钟缓存，一个刚被提成版主的账号会在缓存里仍显示为普通用户。查询出错 → `503 SERVICE_UNAVAILABLE`，什么都不删；持版主能力 → `403 USER_PROTECTED`；查不到 → 可清（同旧面）。预览同样新鲜地查。
3. 本地**一个**事务：
   - 先写三个事务级设置（`set_config(…, true)`）：`kungal.purge_id`（新铸的 UUID）、`kungal.purge_target_user_id`、`kungal.purge_operator_id`。存档触发器只在这三个值存在时工作（§5）。
   - 锁住他的抽奖（他发起的，以及他话题里的）`FOR UPDATE`：任何一个 `drawing` → `409 LOTTERY_DRAWN`，什么都不删（同 T4：开奖正在写中奖名单）。
   - **托管不退。** T4 硬删别人的话题时把托管退给发起人；这里发起人就是被清空的人，而且存档里的抽奖行带着 `point_escrow`，恢复会把它原样带回来——先退款再恢复就是双份。
   - 其余语句同旧面，另加 `DELETE FROM toolset_upload WHERE user_id = ?`（§1.3）。
   - 任何一步失败 → 整个事务回滚：没有删除，也没有存档行（触发器写的存档行在同一个事务里）。
4. 提交之后：catalog 收藏夹（操作者令牌；v1 的 `withUpstream` 在令牌为空时已回 401，不再有「静默跳过」），然后 community `AuthorPurge`。任一失败 → `503 SERVICE_UNAVAILABLE`，ERROR 日志带 `purge_id`。本地已提交；重试是安全的（本地这次删 0 行，远端两个操作都幂等）。
5. `204`。
6. 成功日志一行：`purge_id`、`operator_id`、`target_id`、按表的存档行数、community / catalog 回执。

## 5. 存档（迁移 120）

### 5.1 表 `user_purge_archive`

| 列 | 类型 | |
|---|---|---|
| `id` | `bigserial` PK | |
| `purge_id` | `uuid NOT NULL` | 一次清空一个；索引 |
| `target_user_id` | `integer NOT NULL` | 索引 |
| `operator_id` | `integer NOT NULL` | |
| `table_name` | `text NOT NULL` | |
| `operation` | `text NOT NULL` `CHECK IN ('delete','update')` | |
| `row_pk` | `jsonb NOT NULL` | 主键列 → 值（复合主键多个键）。生产与临时库每张表都有主键（已核） |
| `row_data` | `jsonb NOT NULL` | 改动前的整行 `to_jsonb(OLD)` |
| `new_values` | `jsonb` | 只对 `update`：被改动的列 → 改动后的值 |
| `created_at` | `timestamptz NOT NULL DEFAULT now()` | 索引（保留期清理） |
| `restored_at` | `timestamptz` | 恢复过就不能再恢复（§6） |

**对裁决 ① 的两处扩展**，理由：`operation` / `new_values` —— 清空不只删行（§1.4 的 SET NULL、网站转交、待办放回、计数器），只存删掉的行就撤销不了这些改写；`restored_at` —— 计数器按差值恢复（§6），跑两次会加两遍。

### 5.2 怎么捕获：触发器，而不是清单

- 函数 `user_purge_capture()`：`AFTER DELETE OR UPDATE … FOR EACH ROW WHEN (current_setting('kungal.purge_id', true) <> '')`。
  - `WHEN` 为假时事件不入队：清空之外的任何写几乎零开销，设置是事务级的，连接池里的别的请求看不到。
  - `UPDATE` 只在某列真的变了时记一行（`feed_upsert` 的同值 upsert 不记）。
  - 主键由 `pg_index` 按表现取，不靠传参。
- 挂在 **public 下除 `user_purge_archive` 与 `_migrations` 之外的每一张表**上（`trg_user_purge_archive`）。外键级联、SET NULL、`feed_sync_*` 触发器产生的删改都在同一个事务里，一个不漏。
- 挂载用一个幂等函数 `user_purge_archive_attach()`：给还没挂的表挂上。**以后新建表的迁移在末尾调一次**；闸（§7）会在 CI 的 `db` 作业上抓漏挂的表。bootstrap 脚本在最后调一次（步骤 7 会重建 `galgame_contributor`）。

### 5.3 保留期

`cron.Jobs` 加一项，每天 03:30（北京时间）`DELETE FROM user_purge_archive WHERE created_at < now() - interval '30 days'`，打 INFO 带删除行数。恢复过的行一样到期删除。

### 5.4 隐私

存档里有被清空账号的私聊，也有对方在同一个私聊房里写的消息、别人挂在他话题下的回复。只能经数据库读（没有任何 API 面），30 天后删除。

## 6. 恢复

迁移 120 装一个函数 `user_purge_restore(p_purge_id uuid)`，**恢复就是这条查询**：

```sql
-- 生产：ssh kungal-neo "sudo -n docker exec -i kun-visual-novel-infra-vqvqbc-postgres-1 psql -U postgres -d kungalgame"
-- 1. 找清空 id
SELECT purge_id, operator_id, min(created_at) AS purged_at, count(*) AS archived_rows,
       bool_or(restored_at IS NOT NULL) AS restored
  FROM user_purge_archive WHERE target_user_id = <uid>
 GROUP BY purge_id, operator_id ORDER BY purged_at DESC;

-- 2. 先演练，再提交
BEGIN;
SELECT * FROM user_purge_restore('<purge_id>');
-- 看报告：每张表、每种操作的 archived / restored；两者不等的行是冲突，被跳过
ROLLBACK;   -- 报告无误后把这一行换成 COMMIT 再跑一遍
```

函数做四件事，全在调用者的事务里：

1. **没有未恢复的行 → 报错**（不存在的 id、已经恢复过的 id）。
2. `SET LOCAL session_replication_role = replica`：关掉触发器与外键检查，把 `operation = 'delete'` 的行按原主键原样插回（`jsonb_populate_record`，`ON CONFLICT DO NOTHING`）。关掉触发器是必须的：插回一个话题会让 `feed_sync_topic` 新铸一行动态，再插回存档里那行原动态就撞唯一键；原样插回的行本来就包括原来的动态行。
3. 按存档 id **倒序**撤销 `update`：计数器列（名字以 `_count` / `_sum` 结尾的数值列）**加回差值** `旧 − 新`——清空之后新增的赞不会被抹掉；其余列**比较后置回**：当前值仍等于清空写下的新值才改回旧值，否则跳过、计入报告（清空之后有人动过它，以现在为准）。
4. 切回 `origin`，**校验外键**：对每个带恢复行的子表、每个单列外键，找恢复行里指向不存在父行的——有一个就 `RAISE EXCEPTION`，整个恢复回滚（例：清空后别人删了一个话题，而他在那里的回复要插回来）。报错信息写明表、列、条数；操作者按需把那几行从存档里删掉再跑。最后把本次处理的存档行记上 `restored_at`。

必须以超级用户执行（`session_replication_role`）；生产的 `postgres` 就是。

**恢复不了的**：community 帖子正文与反应、catalog 收藏夹（§1.5）；账号中心的封禁 / 注销（在账号中心撤销）。清空之后懒建的 `kungal_user_state` 会让原行冲突被跳过（记忆 `prod-accidental-purge-restore`：当时的修法是按存档行 UPDATE）。恢复后 ~10 分钟内内容仍可能被 userclient 缓存判成不可渲染。清空之后才加的列，恢复时是 NULL；`NOT NULL` 无默认值的会让恢复报错，要手工补。

## 7. 覆盖闸（新）

1. **每张表都挂着捕获触发器**：public 下除 `user_purge_archive`、`_migrations` 外，每张表都有 `trg_user_purge_archive`。失败信息指向 `SELECT user_purge_archive_attach();`。
2. **每个用户列都有归宿**：按列名（`user_id`、`*_user_id`、`sender_id`、`receiver_id`、`follower_id`、`followed_id`、`friend_id`、`author_id`、`operator_id`、`claimed_by`、`updated_by`…）从 `information_schema` 扫出的每个 (表, 列)，都必须在 `repository.UserColumns` 里登记为 删除 / 转交 / 放回 / 豁免（豁免必须带理由）。新表加了用户列而没登记 → 红。这取代记忆里「加表要改三处」：`cmd/purge-staging-verify` 的列清单改为从同一份登记表派生。
3. 标成「删除」的每个 (表, 列)，清空之后该用户的行数为 0（DB 测试在种子覆盖到的表上断言；staging 工具在生产副本上断言全部）。

## 8. 错误码

新增 1 个（三处一译：`registry.go` + `registry_test.go` + `problem.json`）：

| code | 域 | status | 什么时候 |
|---|---|---|---|
| `USER_PROTECTED` | kungal | 403 | 目标持有版主能力，他的内容永不被清空（staff 内容里有别人要读的站点文档；这个功能是清理 spam 账号用的）。调用者有 `user.purge_content`，被拒的是目标，所以不是 `PERMISSION_REQUIRED`。infra 注册表没有同义码（已查） |

复用：`MISSING_CREDENTIAL` / `INVALID_CREDENTIAL`、`PERMISSION_REQUIRED`、`NOT_FOUND`（超范围 id）、`LOTTERY_DRAWN`（清空）、`SERVICE_UNAVAILABLE`（OAuth / community / catalog）、`INTERNAL_ERROR`。

## 9. 网页

| 文件 | 改什么 |
|---|---|
| `components/admin/UserCard.vue` | 两处调用换类型化客户端；统计标签按 `*_count` 字段；`community_post_count` 为 `null` 显示「?」；`is_protected` 时禁用按钮并说明原因；成功提示用预览的 `total_count` |
| 新 `components/admin/UserPurgeDialog.vue` | 替掉 `alert` 确认框：`KunModal` 里列出将删除的各项；**`is_account_active` 为真时**醒目提示「该账号仍可登录发帖：请先在账号中心封禁或注销，再清空，否则清空后仍会继续产生内容」并给账号中心的链接（同页面已有的 `oauthAdminUrl`）；**`KunInput` 手输目标用户 id，与 id 完全相等才能点确认**；说明「清空可在 30 天内由开发者从存档恢复，社区评论与收藏夹除外」 |
| 手写类型 | 删 `AdminUserContentStats` |
| legacy fetch 基线 | 下调 2 |

`CHANGELOG.md` 记一条。

## 10. 变异清单（先于实现提交）

| # | 破坏 | 应当变红的性质 |
|---|---|---|
| 1 | 清空 / 预览不查 `user.purge_content` | 版主（没有这个键）→ `403 PERMISSION_REQUIRED`，什么都没删 |
| 2 | 去掉「持版主能力的目标拒绝」 | 目标是版主 → `403 USER_PROTECTED`，他的内容还在，存档为空 |
| 3 | 去掉查目标前的 `Invalidate` | 假 OAuth 在一次缓存读之后把目标提成版主 → 清空仍是 `403 USER_PROTECTED` |
| 4 | `set_config(…, false)`（会话级而不是事务级） | 单连接池上，清空之后一条普通 DELETE 不产生任何存档行 |
| 5 | 清空拆成两个事务（先提交作者内容，再删私聊与通知） | 注入第二段失败：话题仍在、存档为空、`purgeUserContent` 回错 |
| 6 | `user_purge_archive_attach()` 漏挂一张表（`topic_reply_reaction`） | 覆盖闸 1 红；「清空→恢复→与原来相等」红（别人对他回复的表情回不来） |
| 7 | 触发器只挂 `AFTER DELETE`，不管 `UPDATE` | 「清空→恢复→与原来相等」红：别人话题的最佳答案、网站所有者、待办认领、子评论的父评论回不来 |
| 8 | 恢复把计数器写回旧值（比较后置回），而不是加差值 | 清空后别人又赞了同一个话题 → 恢复后 `like_count` = 原值 + 1 |
| 9 | 恢复不做外键校验 | 清空后删掉他回复所在的话题 → 恢复报错，**一行都没插回**（没有孤儿回复） |
| 10 | 恢复不看 `restored_at` | 同一个清空 id 第二次恢复报错，计数器不变 |
| 11 | 保留期写成 3 天 | 29 天前的存档行还在，31 天前的被删 |
| 12 | 去掉 `drawing` 抽奖保护 | 他有一个正在开奖的抽奖 → `409 LOTTERY_DRAWN`，什么都没删，存档为空 |

另有两条静态守卫不出变异题但必须过：`bearer_guard_test`（`user.Can` 换成 `perm.CanUser` 只有它杀得掉，T4 #9 的结论）与覆盖闸 2。

## 11. 删旧路由

2 条全删，连同 `PurgeHandler`、`dto/purge_dto.go`（计数结构搬进 repository 包）、`router.go` 两行、`app.go` 字段。`deadcode -test ./...` 到不动点；`routes.golden` 重生成；`legacy_route_baseline` 下调 2。

## 12. 迁移与后续

- **迁移 `120_user_purge_archive`**：纯加法（建表、三个函数、给现有每张表挂触发器），随部署自动跑，不是 deploy-then-drop。`CREATE TRIGGER` 对每张表拿一次 `SHARE ROW EXCLUSIVE` 锁，整个文件是一个隐式事务，毫秒级。down：删触发器、函数、表。
- `scripts/testdb-bootstrap.sh`：最后调一次 `user_purge_archive_attach()`。
- **后续（不在本段）**：
  - infra：community `AuthorPurge` 清空正文不可逆；要么 infra 侧留存档，要么改软删。报给协调会话转 infra。
  - T4 的话题硬删同样没有记录；它只要在事务里写上 `kungal.purge_*` 三个设置就能复用这个存档（`target_user_id` 填话题作者）。是否要做，由协调会话定。
  - 生产那张一次性表 `_user_merge_109310_to_1922` 会被挂上触发器（无害）；它本身该不该留，不归本段。
