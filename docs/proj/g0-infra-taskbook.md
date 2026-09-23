# G0 同窗任务书 · infra 侧（kungal gid → catalog work id）

> 写给一个 **nextmoe-infra 会话**。由论坛 G 轨（`api-v1/g-galgame`）的会话起草，2026-09-23。
> 背景与全部数字见论坛仓 `docs/proj/gid-is-work-id.md`（`git show origin/api-v1/g-galgame:docs/proj/gid-is-work-id.md`）。
> 本文列的是**只有 infra 能做**的部分。论坛这边的改号命令、代码和迁移由 G 会话负责；两边在同一个维护窗口里接力完成。

## 0. 一段话背景

catalog 在 2026-07-06 按 gid 顺序、用自增序列铸 `catalog_work.id`，wiki 缺行导致从 gid 923 起整体错位（1 万以上稳定在 −185）。论坛为了照常路由，维护了一层 gid ↔ catalog id 映射。用户 2026-09-23 裁决：**论坛服从 infra 的 id**，把 `galgame.id` 一次性改号为 `catalog_work.id`，删掉全部映射，名字统一叫 `work_id`，作品页仍是 `/galgame/:id`，旧号码不做重定向。moyu 已于 2026-09-13 完成同样的改号，它的文档（`kun-galgame-patch/docs/proj/gid-is-work-id.md` 末节）早已写明 infra 欠一项：`product_work_id = id`。

## 1. 输入：改号表

论坛的改号命令在 `-apply` 的同一事务里，把改号表写进论坛库 `kungalgame` 的一张表（表名在 G0a 定稿时写进本节，暂定 `galgame_renumber_2026`），同时导出 TSV 到宿主机。列：`old_id int`、`new_id bigint`、`how text`（`curated` / `claim` / `already_catalog_id`）、`folded_into bigint`（10 组重复页里被合并掉的那一个，否则为 null）。

- 共 15,990 行；`old_id = new_id` 的 2,493 行；10 组 `new_id` 相同（重复页）。
- infra 的工具读这张表的方式，同现有 `jobs/forumratings`（同一 PG 实例、按库名连接）。**不得写论坛库**。

## 2. 要做的改写（全部在窗口里、论坛 `-apply` 提交之后、论坛 API 重新启动之前）

每一项都要：先 dry-run 输出计数 → 与下表的「预期」核对 → 再 apply → apply 后再跑一次验证查询。计数不符就停，报给 G 会话，不要「修一下再继续」。

### 2.1 catalog：kungal claim 的 `product_work_id := id`

- 范围：`catalog_work WHERE site = 'kungal' AND product_work_id IS DISTINCT FROM id`。实测 kungal claim 64,833 行，其中已相等约 1,260 行。
- 注意唯一索引 `uq_catalog_work_claim (medium_id, site, product_work_id)`：旧 gid 与别行的 id 大量数值重合（错位就是这么来的），一条 `UPDATE … SET product_work_id = id` 会在语句中途撞唯一约束。用两跳（先搬到不可能重合的区间，再落到 `id`），或临时让约束可延迟，二选一，写进报告。
- 读者会随之改变行为（普查已逐条确认）：`claimed_by.work_id` / v2 `claim.site_work_id`、`/v2/catalog/claim-events` 的 `product_work_id` 快照、`FindClaimed` / `LoadClaimedWorkIDs`、`/v2/catalog/revisions` 的 `site_work_id`、moyu 的 `forum_gid`。改完之后 moyu 指向论坛的链接自动就是新号，**moyu 不需要改代码**。
- `repr/resource.go:12` 的描述「The site's own work id, not the catalog id.」对 kungal 不再成立（moyu 仍共用 site `kungal`，字段保留）；改成如实的描述。

### 2.2 community：`site='kungal' AND anchor_kind=1`（site_game）的 `anchor_id`

- 表：`community_thread`（实测 70,480 串，其中 2,861 串有帖子）、`community_anchor_user`、`community_notification`。其余表只挂 `thread_id` / `post_id`，不用动。
- `anchor_id` 是十进制字符串，按改号表 `old_id → new_id` 改写。
- **10 组重复页**：两个旧号可能各有一串，改号后都想要同一个 `anchor_id`，而 site-local 锚点的串是唯一的。必须合并成一串：帖子挪到幸存串、重算计数、`community_anchor_user` 去重（唯一键 `(site, user_id, anchor_kind, anchor_id)`）、通知改指幸存串。哪一串做幸存者：有帖子的优先，都有就取帖子多的；写进报告。
- `cmd/retire-merged-comments`：`sweep.go:43-55` 的 `siteGameAnchorIsCatalogID` 加上 `"kungal": true`，同窗部署。改号之后 kungal 的 site_game 与 moyu 一样就是 catalog id，继续按 claim 判定会漏掉该清理的墙。
- dev-seed 的 `scripts/dev-seed/prune/kun_community.sql` 如有按论坛 gid 取样的逻辑，一并按新号修正（只影响开发库）。

### 2.3 trust：`subject_kind = 'galgame'`

- 表：`trust_report`（实测 19 行）、`trust_review_item`（2 行）、`trust_scan_result`、`trust_audit_log` 里同 kind 的行。`subject_id` 按改号表改写；`subject_url` 里若含 `/galgame/<旧号>`，只改这一段数字。
- 其余 `galgame_*` kind（rating / resource / collection / quiz / comment / toolset）的 `subject_id` 不是 gid，**不要动**。
- 改写前确认没有未投递的处置回调挂在这些行上（`callback.go` 在投递时才读 `subject_id`）；有就报告数量。

### 2.4 catalog：收藏条目里论坛写入的 gid

`catalog_user_folder_item.work_id` 没有写入方列（`me_folder.go:248` 丢弃了 OAuth client id），所以分两部分：

**(a) 导入时写入的条目：精确还原。** 2026-09-07 的论坛导入（`cmd/import-favorites`）逐行复制了论坛快照 `galgame_collection_item (collection_id, galgame_id, created, updated)`，并按 `resolve.go:49-52`「数值若恰好是现存 catalog id 就原样保留」写入。论坛库里那张快照表还在（201,518 行）。配对方式：

```
catalog_user_folder_import (site='forum', source_id = collection_id) → folder_id
catalog_user_folder_item i  ⋈  快照 s  ON i.folder_id = 映射后的 folder_id AND i.created_at = s.created
新 work_id = 改号表(s.galgame_id).new_id
```

- 实测：快照里每个 gid 在数值上都恰好是某个现存 catalog id，所以导入没有跳过任何一行（`SkippedNotLive` 应为 0），但 201,518 条里有 **148,548 条**在 catalog 看来挂在了别的作品上。
- 先报告配对结果：配上的行数、一个快照行配到多条或零条的数量、按 `created_at` 配不上的导入期条目数量。实测导入期条目约 20 万条，其中 135,504 条是原始 gid、64,302 条两种解读恰好同值、60 条已是 catalog id、757 条都不是。配对应覆盖几乎全部；覆盖不到的列出来，不要猜。
- 改写后在同一收藏夹内会出现重复（同一部作品先被错写成一个号、又被 moyu 正确写了一次），按唯一键 `(folder_id, work_id)` 去重，保留较早的 `created_at`。
- 与 `merge_rehang.go:291-298` 相同，改写的行要把 `updated_at` 更新为 now()，让同步游标看得到。
- `catalog_user_folder.item_count` 去重后重算。

**(b) 2026-09-07 之后新写入的条目（约 2.7 万条）：暂不改写，先查归属。** 论坛一直写 gid，moyu 一直写 catalog id，表里分不出来。请查：Traefik / API 容器的访问日志是否仍保留 `PUT /v2/me/folders/*/items/*` 请求，能否按 OAuth client（kungal / moyu）区分。把能归属的数量、日志覆盖的时间段报给 G 会话；**是否以及如何改写这部分，由用户裁决**，在裁决之前不动。

### 2.5 `jobs/forumratings`

- `aggregate.go:181-186`、`:214` 直接读论坛库 `galgame_rating.galgame_id`，`:247-252` 再用 claim `product_work_id` 映射到 catalog 作品。
- 论坛 G0 会把该列改名为 `work_id`，并且改号后论坛的 id 就是 catalog id。所以同窗改为：读 `work_id`，**删掉 claim 映射**（直接当 catalog work id 用）。
- 窗口期间不得运行。

## 3. 其它需要 infra 顺手确认的

- `jobs/galgametouch`（`galgametouch.go:14` 仍查已退役的 `site='galgame_wiki'`）：是否还在运行；若在运行，G0 之后它收到的已经是 catalog id，按 `product_work_id` 查不到任何行，改成按 id 直查或退役。
- `catalog_external_ref` 里 `source=curated, entity_type=5, external_id=<旧 gid>` 的 64,515 行：**保留为历史**，不要改写、不要删除（改写 `external_id` 会与作品自身 id 撞义；它们在 G0 之后不再是任何路由的键）。
- `docs/`、`refs/` 里任何写着「kungal 的产品 id 空间 = gid」「site_work_id ≠ catalog id（kungal）」的文字，改成如实的现状。

## 4. 窗口的接力顺序

1. G 会话宣布窗口、占部署档期，停 `kungal-api`（论坛所有 cron 在进程内，随之停止）。
2. G 会话快照论坛库，跑论坛改号 `-apply`（一个事务），写出改号表。
3. **infra 会话执行 §2.1–§2.5**（本任务书），逐项报计数。§2.4(b) 只查不改。
4. G 会话合并论坛 G0b（身份代码 + 列改名迁移），部署，实测。
5. infra 部署 §2.2 的 `retire-merged-comments` 与 §2.5 的 `forumratings` 改动（若尚未随 §3 前部署）。

G 会话在第 2 步完成后给出「改号表已就绪」；infra 在第 3 步完成后给出「infra 改写完成 + 计数」。任一步计数不符即中止，论坛侧用快照回滚。

## 5. 纪律

- 不写论坛库，不改论坛仓，不改 moyu 仓。
- 每一项改写先 dry-run、核对预期、再 apply、再验证；计数与本文不符就停下来报告。
- 生产操作不打印 DSN 与密钥（`refs/` 里 prod tools job 的既有做法）。
- 改写涉及的每张表先各自快照（或整库快照）再动手。
