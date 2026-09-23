# 本地 galgame 行与 catalog 的漂移（2026-09-24）

> 起因：G5 上线后 `/works` 每次启动打出约 20 条 `workrepr: catalog did not render work, dropped`。协调会话 forum-37 派单，裁决见 §4。迁移 195。

## 1. 普查（线上只读，2026-09-24 ~02:00 CST，本地 galgame 全表 16,219 行）

| 类 | 行数 | 其中已发布 | 本地 content_limit 为 NULL | 进 SFW 列表 |
|---|---|---|---|---|
| 已合并掉（catalog `status=2`，**有** work redirect，`entity_type=5`） | 11 | 9 | 9 | 9 |
| 非 galgame 媒介（`medium_id=5` asmr） | 8 | 6 | 8 | 8 |
| 判决陈旧：本地 sfw、catalog 应为 nsfw | 9 | 5 | 0 | 9 |
| 判决陈旧：本地 nsfw、catalog 应为 sfw | 1 | 1 | 0 | 0 |
| 其余（一致） | 16,190 | | 0 | |

- 合并掉的页面本身已经 301 到幸存条目（`/galgame/211706` → `/galgame/309`），只有本地列表还看得见这些孤儿。
- 非 galgame 媒介里 5 行是 **live 的 kungal 认领**，各挂 1 个资源：线上共 6 个资源落在 asmr 作品上（51,009 个里），论坛任何一面都显示不出来；详情页是「未找到这个 Galgame」但回 HTTP 200（另开小 PR）。v1 `POST /works/{work_id}/resources` 经 `lookupWork` → catalog 批量面（按 `medium_id = galgame` 过滤）对这类作品回 404，新的不会再产生。

## 2. 为什么

镜像是单向的，而且**写过一次就不再核对**：漏掉的变更永远错下去，catalog 回答不了的行永远是 NULL，而 NULL 按设计放进 SFW 列表。

- **合并掉的**：合并折叠按游标把 `/v2/catalog/redirects` 读一遍，只折叠**那一刻存在**的本地行（`LocalIDsIn`，否则 unpark）。这些本地行建于 09-16/17，晚于它们 09-14 的合并，redirect 早已走过去。镜像其实看得见：变更信道把它们标 `gone=true`，但 `RunMirror` 把 gone 条目直接丢掉；批量水合也拿不到已删除的作品，于是补齐车道每轮都问、每轮都问不到（线上日志每轮 `requested 18 resolved 0`）。
- **非 galgame 媒介**：infra 的变更信道（`public_works_list.go:263`）与 `/v2/catalog/works`（`:150`）都按 `medium_id = galgame` 过滤，这类行既不上信道，也永远水合不到。
- **判决陈旧**：10 行的 catalog `updated_at` 全落在两次批量写里——09-20 09:39:58.2–59.26 UTC（28,063 行 galgame）与 09-23 03:11:05–19（约 6.4k 行）。本地值是 09-17 建行时补齐车道写的，之后决定判决的字段变了，论坛没有应用。上游那一半没定位（信道没送达，还是某个写者改了字段却没动 `updated_at`；这些行没有 `field_provenance`），已由 forum-37 转给 infra。论坛这一半是确定的：补齐车道只问 NULL/未确认的行，漏掉的变更没有任何机制纠正。

## 3. 改法

**迁移 195**：`galgame.catalog_checked_at timestamptz`、`galgame.catalog_rendered boolean NOT NULL DEFAULT true`。纯新增，随部署自动跑，不是 deploy-then-drop。

**核对车道**（取代补齐车道）：每轮取 `catalog_checked_at` 最旧的 500 行（NULL 最先），用与现在相同的批量水合（`nsfw=true`、不带 `content_limit`）问 catalog，每轮约 5 个请求，全表约 5.5 小时核对一遍。每个 id 的结局：

| catalog 的回答 | 本地 |
|---|---|
| 批量面返回了，且可渲染 | 写 content_limit / release_date，`rendered=true`，盖 `checked_at` |
| 批量面返回了，但认领是 hidden | `rendered=false`，盖 `checked_at` |
| 批量面**成功应答但没有这个 id** → 逐个问 `GET /v2/catalog/works/{id}`：`ENTITY_MERGED` | 幸存条目可渲染 → 交给合并折叠（进它的待办 hash）；`rendered=false`，盖 `checked_at` |
| 同上：`NOT_FOUND` | `rendered=false`，盖 `checked_at` |
| 同上：详情面反而找得到（两个面不一致） | 什么都不动，WARN |
| 任何传输错误、5xx、401、429、超时 | 什么都不动：不改 `rendered`，不盖 `checked_at` |

- **批量翻转熔断**：一轮里会从 `rendered=true` 变成 `false` 的行数超过 `max(20, 5% × 本轮问的行数)` 时，这些翻转一个都不做，打 WARN 带计数，它们的 `checked_at` 也不盖（下一轮还在队首，继续熔断直到有人看）。catalog 以前在我们脚下改过读面行为，整个作品库被清空比陈旧更糟。
- `false → true` 永远放行。
- 变更信道：水合到的行同样写 `rendered=true` 与 `checked_at`；`gone=true` 的本地行立刻走上表「成功应答但没有这个 id」那一路，同样过熔断（按页计）。
- 进内存的 6 小时孤儿备忘录删掉：问不到的行盖了 `checked_at` 就排到队尾，不会再饿死别人。
- 合并折叠本身不动。它的 `Fold` 会在搬完子表后删掉旧的 galgame 行——这是已上线、有 `galgame_merge_discarded` 兜底的正规路径（§4 裁决 1）。它**没有**「幸存条目是 hidden 认领就暂存」的判断（206987 还在暂存是 G0 之前的旧代码留下的），所以镜像交接前自己先验证幸存条目可渲染，不可渲染就只标 `rendered=false`。

**本地列表**：凡是先在本地 SQL 里翻页、再去 catalog 水合的列表都加 `g.catalog_rendered`——galgame 浏览与实体页（`list_repo`）、发售月历、资源列表、评分列表、排行、用户作品。与 SFW 与否无关：catalog 对谁都不渲染它们。

**线上数据**：不写。核对车道部署后一轮（约 5.5 小时）收敛。这些行不会泄漏 NSFW：catalog 的闸在渲染时把它们丢掉，本地列表只是页数不齐。

## 4. 裁决（forum-37，2026-09-24）

1. 有幸存条目的合并行交给合并折叠（它会删旧行，是正规路径）；幸存条目是 hidden 认领的保持暂存；没有幸存条目的只标 `rendered=false`。
2. asmr 等非 galgame 媒介：先标 `rendered=false`、不列出；要不要让 catalog 读面支持媒介过滤、kungal 能不能认领非 galgame 作品，由用户定。
3. 熔断与「只有成功应答且没有这个 id 才标 false」是硬条件，各自要有测试和变异。

## 5. 变异题（先于实现提交）

| # | 破坏 | 必须变红的断言 |
|---|---|---|
| 1 | 批量面出错时照样把问到的行标 `rendered=false` | 批量面 500 / 429 之后，`rendered` 与 `checked_at` 都没变 |
| 2 | 详情面出错时当作 NOT_FOUND | 详情面 503 之后，该行 `rendered` 与 `checked_at` 都没变 |
| 3 | 熔断阈值失效 | 问 100 行、catalog 突然一行都不给 → 一行都不翻，`checked_at` 不盖 |
| 4 | 熔断把 `false → true` 也拦了 | 一轮里 30 行从 false 回到 true，全部放行 |
| 5 | 核对顺序不按 `checked_at` | 刚核对过的行不在下一轮里；NULL 的行先于有值的行 |
| 6 | 返回了但 hidden 的行当作可渲染 | hidden 认领 → `rendered=false` |
| 7 | 合并的行不交给折叠 | `ENTITY_MERGED` 且幸存条目可渲染 → 折叠待办里有 旧→新 |
| 8 | 幸存条目不可渲染也交给折叠 | 幸存条目 hidden → 待办里没有它，旧行 `rendered=false` |
| 9 | 信道的 gone 条目被丢掉 | 信道 `gone=true` 的本地行 → `rendered=false` |
| 10 | 信道的 gone 不过熔断 | 一页 30 个本地 gone → 一行都不翻 |
| 11 | 列表不排除 `rendered=false` | 浏览、实体页、月历、资源、评分、排行、用户作品各少那一行 |
| 12 | 判决陈旧永不纠正 | 已有 `sfw` 的行，catalog 现在答 `nsfw`，核对一轮后是 `nsfw` |

## 6. 迁移

**195**：新增两列，现有行 `catalog_rendered = true`、`catalog_checked_at = NULL`（部署后核对车道从头走一遍）。随部署自动跑，无手动步骤。
