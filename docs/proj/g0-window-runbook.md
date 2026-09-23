# G0 维护窗口操作单

> 论坛 G 会话执行，infra 会话按 [`g0-infra-taskbook.md`](g0-infra-taskbook.md) 接力。背景与裁决见 [`gid-is-work-id.md`](gid-is-work-id.md)。
> 命令都在宿主机 `kungal-neo` 上跑；`$C` 代表 `sudo docker compose -f docker-compose.prod.yml -p kun-visual-novel-forum-iunwa9`，工作目录 `/etc/dokploy/compose/kun-visual-novel-forum-iunwa9/code`。

## 0. 窗口前

- G0b PR（#192）全绿、已审、**未合并**。它删掉了 `cmd/align-galgame-ids`，所以改号只能用合并之前的 tools 镜像跑（宿主机上现有的 `kungal-tools:latest` 是 G0a 版，2026-09-23T06:22Z 构建，内含 `align-galgame-ids`）。
- 通知 T / U / M / RC 会话：窗口期间不合并、不部署。别的轨一合并，webhook 可能在改号事务中途把旧版 API 拉起来。
- infra 会话已读完任务书，§2 各项的 dry-run 已在 staging 或本地演练过；`forumratings` 读 `work_id` 的版本已构建好，等窗口部署。
- 本地准备：`git show origin/master:apps/api/cmd/align-galgame-ids/catalog_map.sql > catalog_map.sql`，`scp` 到宿主机 `~/g0/`。
- 取样：演练库改号表里 20 个已发布、`old_id <> new_id` 的号，2026-09-23 在线上 `GET /api/galgame/<旧号>` 记下的 `vndb_id` 如下。窗口后 `GET /api/galgame/<新号>` 必须回同一个 `vndb_id`：

| 旧号 | 新号 | vndb_id |
|---|---|---|
| 5329 | 5234 | v12708 |
| 1970 | 1966 | v3346 |
| 20074 | 19889 | v16616 |
| 4612 | 4554 | v250 |
| 6095 | 5961 | v22739 |
| 4897 | 4833 | v13956 |
| 12223 | 12038 | v5171 |
| 61365 | 61164 | v65717 |
| 3492 | 3478 | v861 |
| 2081 | 2076 | v29098 |
| 57394 | 57207 | v61471 |
| 4713 | 4650 | v28946 |
| 3373 | 3361 | v55797 |
| 1994 | 1990 | v893 |
| 5242 | 5151 | v25725 |
| 22227 | 22042 | v19710 |
| 4497 | 4444 | v22161 |
| 5515 | 5415 | v32318 |
| 10522 | 10337 | v2183 |
| 21765 | 21580 | v19018 |

## 1. 停写（停机从这里开始）

```
$C stop kungal-api
```

论坛的 cron 都跑在 API 进程里，随之停止。web 继续运行，会报错，是预期的。

## 2. 快照

```
sudo docker exec kun-visual-novel-infra-vqvqbc-postgres-1 pg_dump -U postgres -Fc -d kungalgame > ~/g0/kungalgame-pre-g0.dump
```

演练时的库约 50 MB。快照要留到 infra 全部完成、线上验证通过之后。

## 3. 生成映射（必须在 infra 改写 claim 之前）

```
sudo docker exec -i kun-visual-novel-infra-vqvqbc-postgres-1 psql -U postgres -X -q -At -F $'\t' -d kun_catalog < ~/g0/catalog_map.sql > ~/g0/catalog-map.tsv
wc -l ~/g0/catalog-map.tsv
```

infra §2.1 把 `product_work_id` 改成 `id` 之后，claim 那一半会变成恒等映射，所以映射必须在它之前取。演练时是 64,850 行。

## 4. 改号

```
mkdir -p ~/g0/dry ~/g0/apply && sudo chown -R 10001 ~/g0/dry ~/g0/apply
$C --profile jobs run --rm -v ~/g0:/g0 tools align-galgame-ids -map /g0/catalog-map.tsv -check-live
$C --profile jobs run --rm -v ~/g0:/g0 tools align-galgame-ids -map /g0/catalog-map.tsv -report /g0/dry
$C --profile jobs run --rm -v ~/g0:/g0 tools align-galgame-ids -map /g0/catalog-map.tsv -report /g0/apply -apply
```

- tools 镜像以 uid 10001 运行，报告目录必须归它。
- `-check-live` 把映射与现行桥（`CatalogWorkIDs`）逐个比对，不写库。有不一致就停。
- dry-run 执行全部写入后回滚。核对 `~/g0/dry/summary.txt` 与演练：`map entries` 约 66,108，`folds=10`，`galgame changed` 约 13,493，每张表 `sum_before = sum_after`，不变量全部通过。数字明显偏离就停。
- `-apply` 在一个事务里完成，演练耗时 23 秒。私信链接默认不改；用户裁决要改时加 `-include-dm`。
- 把 `~/g0/apply/folds.tsv` 交给 infra 会话。
- 核对 G0b 里预先改写的 `apps/api/pkg/dlsite/verified.tsv`（DLsite 购买白名单，4,320 行，原来按论坛 gid 取键，G0b 已按演练库的改号表改写）：导出线上改号表 `select old_id, new_id from galgame_renumber_2026`，对 `git show origin/master:apps/api/pkg/dlsite/verified.tsv` 的每个旧键逐一映射，结果必须与 G0b 分支里的文件完全一致。不一致就先改文件、重新构建，再合并。

## 5. infra 接力

infra 会话按任务书 §2 执行：claim `product_work_id := id`、community site_game 锚点（含 10 组合并）、trust、收藏条目、`moemoepoint_log.ref`，部署 `forumratings` 与 `retire-merged-comments`。每项都要先 dry-run 核对计数再 apply。论坛这边可以同时进行第 6 步。

## 6. 部署 G0b

```
gh pr merge 192 --squash          # 本地
# 等 build-and-push 完成（约 3 分钟）
$C pull kungal-api web
$C up -d migrate kungal-api web
```

- migrate 容器先跑 141（列改名、删 `galgame_redirect`、删序列），成功后 API 才启动（`depends_on: service_completed_successfully`）。
- webhook 不可信：看 `sudo docker ps --format '{{.Names}} {{.CreatedAt}}'`，确认三个容器都是刚创建的。
- 清缓存。只删下列前缀，**绝不 FLUSHDB**：DB 0 是 infra 共用的，会话键也在里面。

```
for p in 'nmcache:v2:*' 'moyu:patches:v1:*' 'kungal:folder-items:v1:*' 'activity:v2:*'; do
  sudo docker exec kun-visual-novel-infra-vqvqbc-redis-1 sh -c "redis-cli -n 0 --scan --pattern '$p' | xargs -r -n 500 redis-cli -n 0 del"
done
```

  `catalog:claim:submitter:` 与 `catalog:redirects:merge:deferred` 在 G0b 里换了新前缀，旧键自然过期。游标（`catalog:claim:cron:since`、`catalog:redirects:merge:cursor`、`catalog:rev:*`、`catalog:contrib:*`、`catalog:changes:*`）必须保留。

## 7. 验证

- 第 0 步的 20 个样本：`/api/galgame/<新号>` 回表里的 `vndb_id`；`/galgame/1`（未变号）照常。
- 抽一个变号作品：资源、评分、收藏夹、评论墙、游玩时长、moyu 补丁（`/api/v1/works/{work_id}/moyu-patches`）都在。
- 帖子里改写过的 `/galgame/<n>` 链接能打开对的作品；首页 feed 的链接同样。
- API 日志 15 分钟内没有新的错误类型，重点看 claim-event、merge、changes 这几个 cron 的首轮。
- infra 报告的每项计数与预期相符。

## 8. 回滚

- 第 4 步 `-apply` 之前：什么都没发生，`$C up -d kungal-api` 即可。
- `-apply` 之后、G0b 部署之前：用快照恢复 `kungalgame`（`pg_restore --clean --if-exists`），再启动旧 API。infra 已做的改写按它自己的报告撤回。
- G0b 部署之后：先确认问题是否只在代码。141 可以 down（列名改回），但 down 不撤销改号；需要撤销改号时只能恢复快照，并且 infra 同步撤回。

## 9. 收尾

- `CLAUDE.md` 铁律 3 已随 G0b 合并。
- `docs/proj/api-v1/CHANGELOG.md` 顶部 “G0 window” 一节的标题补上窗口日期。
- 删掉宿主机上的 `~/g0/`（含快照）要等 infra 全部完成并验证之后；本地演练库 `kun_ephem_20260922_g0_rehearsal` 与 dump 同时删除。
- `galgame_renumber_2026` 在 infra 确认不再读取之后由后续迁移删除。
