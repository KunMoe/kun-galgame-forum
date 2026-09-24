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
| `operation` | `text NOT NULL` `CHECK IN ('delete','update','insert')` | `insert`：清空事务里新写的行（§13） |
| `row_pk` | `jsonb NOT NULL` | 主键列 → 值（复合主键多个键）。生产与临时库每张表都有主键（已核） |
| `row_data` | `jsonb NOT NULL` | 改动前的整行 `to_jsonb(OLD)`；`insert` 存新行 |
| `new_values` | `jsonb` | 只对 `update`：被改动的列 → 改动后的值 |
| `created_at` | `timestamptz NOT NULL DEFAULT now()` | 索引（保留期清理） |
| `restored_at` | `timestamptz` | 恢复过就不能再恢复（§6） |

**对裁决 ① 的两处扩展**，理由：`operation` / `new_values` —— 清空不只删行（§1.4 的 SET NULL、网站转交、待办放回、计数器），只存删掉的行就撤销不了这些改写；`restored_at` —— 计数器按差值恢复（§6），跑两次会加两遍。

### 5.2 怎么捕获：触发器，而不是清单

- 函数 `user_purge_capture()`：`AFTER INSERT OR UPDATE OR DELETE … FOR EACH ROW WHEN (coalesce(current_setting('kungal.purge_id', true), '') <> '')`。
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

**完整撤销是两步：论坛库 + community**（U3c.1，§14）。一条命令做完两步：

```bash
# 在 kungal-neo 上，/etc/dokploy/compose/kun-visual-novel-forum-iunwa9/code 目录里
sudo docker compose -f docker-compose.prod.yml -p kun-visual-novel-forum-iunwa9 run --rm tools purge-restore <purge_id>          # 演练：论坛库回滚，不调 community
sudo docker compose -f docker-compose.prod.yml -p kun-visual-novel-forum-iunwa9 run --rm tools purge-restore -commit <purge_id>  # 提交：论坛库，然后 community
```

`tools` 容器带的是 API 的同一份环境，数据库与 S2S 凭证都不出现在命令行上。community 回 `404` 表示那边没有可恢复的内容（本站没清空过他、已撤销、超过 30 天），不算失败。community 失败时论坛库那半已提交，重跑同一条命令：论坛库报「已恢复」，只补 community。

论坛库那一步就是迁移 120 装的函数 `user_purge_restore(p_purge_id uuid)`，工具只是替你开事务、读报告、再调 community。手工只恢复论坛库时：

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
3. 按存档 id **倒序逐行**撤销（`delete` 插回、`insert` 删掉、`update` 按列改回）。`update`：计数器列（名字以 `_count` / `_sum` 结尾的数值列）**加回差值** `旧 − 新`——清空之后新增的赞不会被抹掉；其余列**比较后置回**：当前值仍等于清空写下的新值才改回旧值，否则跳过、计入报告（清空之后有人动过它，以现在为准）。
4. 切回 `origin`，**校验外键**：对每个带恢复行的子表、它的每个外键（多列外键同样处理），找恢复行里指向不存在父行的——有一个就 `RAISE EXCEPTION`，整个恢复回滚（例：清空后别人删了一个话题，而他在那里的回复要插回来）。报错信息写明表、列、条数；操作者按需把那几行从存档里删掉再跑。最后把本次处理的存档行记上 `restored_at`。

必须以超级用户执行（`session_replication_role`）；生产的 `postgres` 就是。

**恢复不了的**：catalog 收藏夹（§1.5）；账号中心的封禁 / 注销（在账号中心撤销）。community 帖子正文与反应自 infra #298 起可以恢复（上面的第二步），同样只有 30 天。清空之后懒建的 `kungal_user_state` 会让原行冲突被跳过（记忆 `prod-accidental-purge-restore`：当时的修法是按存档行 UPDATE）。恢复后 ~10 分钟内内容仍可能被 userclient 缓存判成不可渲染。清空之后才加的列，恢复时是 NULL；`NOT NULL` 无默认值的会让恢复报错，要手工补。

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

## 13. 实现时对契约的更正（2026-09-24）

实现、变异与协调会话的审查条件带来的，已改在上文对应位置，这里留底：

| 原契约 | 改为 | 原因 |
|---|---|---|
| 存档只收 `delete` / `update` | 另收 `insert`，恢复时删掉 | `feed_sync_*` 在计数器重算触发的 UPDATE 上走 `feed_upsert`：动态行缺失时，清空会**新插**一行。不收它，恢复后多一行；往返测试里专门删掉一行动态来钉住它 |
| 恢复：先成批插回删掉的行，再倒序撤销 `update` | 全部按存档 id 倒序逐行处理 | 同一个唯一键可能在一次清空里先删后插（或先插后删），只有严格倒序才对。代价：10 万行的恢复 2.1 秒（临时库实测） |
| — | **清空不再重算 `topic_reply.comment_count`** | 迁移 102（2026-09-22）已删这一列，生产同样没有；从那天起，清空任何写过话题评论的用户都会 500：`column "comment_count" of relation "topic_reply" does not exist`。1484 与 104136 那两次跑通了，是因为两人都没有话题评论（生产只读核过：`topic_comment` 里两人 0 行）。写 DB 测试时第一次跑就红在这里 |
| `WHEN (current_setting(...) <> '')` | `WHEN (coalesce(current_setting('kungal.purge_id', true), '') <> '')` | 协调会话条件 1：写明「设过又结束后读回 `''` 而不是 NULL」 |
| 迁移「毫秒级」 | 文件开头 `SET LOCAL lock_timeout = '10s'` | 协调会话条件 3：迁移运行器把整个文件当一次简单查询发出，Postgres 把它当一个隐式事务，已锁的表会一直写阻塞到提交。某张表被长查询占着时，10 秒内失败；迁移失败时新 API 起不来（`depends_on: service_completed_successfully`），旧容器继续服务，重新部署即可。挂触发器用 `CREATE OR REPLACE TRIGGER`，只挂还没挂的表，重跑无副作用 |
| 保留期「删 30 天前的行」 | 每批 5000 行，按 `created_at` | 协调会话条件 6。一次清空的所有行共用 `created_at`（事务开始时间），恢复过的也按清空时间算，所以一次清空要么整体留、要么整体删 |
| `UserColumns` 只列业务表 | 另列 `user_purge_archive` 的两列（豁免） | 覆盖闸 2 扫到了存档表自己的用户列 |
| 网页：`total_count` 为 0 就禁用清空按钮 | 本地为 0 但 `community_post_count` 不为 0 时仍可点 | 远端失败后本地已清空，要能重试把社区那半做完 |

**性能（协调会话条件 1，临时库，5 万行 `UPDATE … SET n = n + 1`，各 5 次）**：不挂触发器 74–105 ms（中位 93）；挂触发器、会话从未设过 81–91 ms（中位 84）；挂触发器、设过又结束 82–91 ms（中位 89）；0 行进存档。噪声大于差异，`WHEN` 为假时事件不入队，plpgsql 不进入。清空事务内：每行 `UPDATE` 捕获约 19 µs、`DELETE` 约 13 µs；生产最重的一次清空（约 8 万行）多 1–2 秒。

**测试**：`internal/admin/repository/purge_archive_db_test.go`（往返：全库快照 → 清空 → 每一条被删 / 被改 / 新写的行都在存档里 → 恢复 → 全库快照逐行相等；失败不留痕；设置随事务结束；恢复保留之后的计数变化；缺父行时整体中止；只能恢复一次；保留期；开奖中；覆盖闸）与 `internal/app/v1_user_purge_test.go`（经真实服务路径清空后存档里有行、操作者 id 正确，权限、受保护目标、新鲜读、上游失败与重试、开奖中）。

## 14. U3c.1 · 一条命令撤销整次清空（2026-09-24，契约先于实现）

infra #298（2026-09-24 上线）让 community 的作者清空也在同一事务里留 30 天存档（`kun_community.community_purge_archive`）。撤销是 `POST /authors/{id}/purge/restore`，S2S 凭证与站点绑定同清空，回各项恢复计数。没有可恢复的就回 `404`：本站没清空过他、已经撤销过、或超过 30 天（infra `docs/community/01-service-and-contract.md` §4）。于是**完整撤销 = 论坛库 + community 两步**，§6 改成用这个工具。

### 14.1 工具 `cmd/purge-restore`

- **运行**：`docker compose -f docker-compose.prod.yml -p kun-visual-novel-forum-iunwa9 run --rm tools purge-restore [-commit] <purge_id>`。`tools` 服务与 API 用同一份环境（`kungal-api-env`：数据库、S2S 凭证），命令行上只有清空 id。API 镜像是 distroless 单二进制，放不进第二个命令；`tools` 镜像每次合并随 API 一起构建，含全部 `cmd/*`。
- **默认演练**：论坛库的 `user_purge_restore` 在一个会回滚的事务里跑，打印报告；**不调 community**（它没有演练模式）。
- **`-commit`**：论坛库恢复提交，再用存档里的 `target_user_id` 调 community 恢复，打印两份报告。
- **论坛库那半已恢复过**（该清空 id 的存档行全有 `restored_at`）：打印「论坛库已恢复」，继续 community 那半。这是 community 失败后重跑的路径。
- **存档里没有这个清空 id**（不存在或已过期）：报错退出，不去猜目标用户。
- **community `404`** → 打印「community 没有可恢复的内容」及上游原话，**不算失败**；其它错误 → 非零退出。此时论坛库那半已提交，重跑安全。
- 数据库连接用 API 自己的 `KUN_DATABASE_URL`；生产是 `postgres` 超级用户，`session_replication_role` 需要它。
- 客户端加 `communityclient.RestoreAuthorPurge(ctx, authorID)`，回 infra `RestoreResponse` 的五个计数。

无新路由、无迁移、无网页改动。

### 14.2 变异清单（先于实现提交）

| # | 破坏 | 应当变红的性质 |
|---|---|---|
| 1 | 演练也提交 | 演练之后目标的内容仍然不在，存档的 `restored_at` 仍为空 |
| 2 | 演练也调 community | 演练时假 community 的恢复调用次数为 0 |
| 3 | community `404` 当成失败 | `404` 时恢复成功，报告「没有可恢复的」 |
| 4 | 论坛库已恢复时报错、不继续 community | 第二次 `-commit` 成功，并且调了 community |
| 5 | 用操作者 id 而不是存档里的 `target_user_id` 调 community | 假 community 收到的作者 id 是清空目标 |

### 14.3 实现与验收（2026-09-24）

- `communityclient.RestoreAuthorPurge`；`PurgeRepository.RestoreArchive(purge_id, commit)`（先读存档头：无行 → `ErrPurgeNotArchived`，全已恢复 → 跳过论坛库那半）；`PurgeService.Restore`；`cmd/purge-restore` 打印两份报告。
- 测试：`internal/admin/service/purge_restore_db_test.go`（演练、提交、community `404` 视为无可恢复、community 失败后重跑、未知清空 id）与 `pkg/communityclient` 的 `TestRestoreAuthorPurge`。
- 在临时库上跑了真二进制：演练（回滚、不调 community，退出 0）→ `-commit`（两半都恢复）→ 再 `-commit`（论坛库「已恢复」、community「没有可恢复的」，退出 0）→ 未知 id（退出 1）→ 非 UUID（退出 2）。

## 15. U3c 上线验收（2026-09-24，#236 = 1a9dd75f2）

- 部署：API 容器 06:17:36Z 重建，migrate 退出 0；`_migrations` 记着 `120_user_purge_archive`（14:17:37 +08）。
- 生产只读：91 张表、89 个 `trg_user_purge_archive`，没挂的只有 `_migrations` 与 `user_purge_archive`；存档 0 行；三个函数都在。
- 运行中的 `/app` 含 `/admin/user-contents/{user_id}` 与 `kungal.purge_target_user_id`，不含 `PurgeHandler`。
- 匿名 `GET` / `DELETE /api/v1/admin/user-contents/1` → `401 MISSING_CREDENTIAL`（路由已注册；未知 v1 路径是 404）。
- 日志：除 infra #298 重启 community 带来的 16 条 `lookup community: no such host` 外无 ERROR；WARN 都是既有类别。镜像核对（每 2 分钟一批 UPDATE）的「核对完成」照常出现，`WHEN` 闸在真实批量写上没有造成异常。
- 没做：生产带登录的清空（没有生产会话，也不在生产铸会话）。
