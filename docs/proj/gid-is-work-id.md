# gid ≡ work_id —— 论坛作品 id 与 catalog 对齐（G0）

> 2026-09-23 立。所有数字当天在生产库（`kungalgame`、`kun_catalog`、`kun_community`、`kun_trust`，infra postgres 容器）只读实测。
> 这是 G 轨的第一个 PR（G0），G1–G4 的 v1 契约全部等它。

## 1. 为什么

论坛的作品 id（`galgame.id`，历史上叫 gid）和 catalog 的 `catalog_work.id` 本该是同一个数，实际上不是。

**迁移时的铁律**是 2026-07-15 用户定的：「由于涉及到大量的 SEO，所以 kungal+moyu 的游戏 id 不能变，例如 /galgame/1 对应的游戏必须是 枯れない世界と終わる花」。infra 的定案（`nextmoe-infra/refs/docs/nextmoe-draft/19-open-api-program-and-wiki-retirement.md` §5.1）把它实现成「gid 是契约层的键，由 claim 三元组 `site + product_work_id` 承载」，**没有**让 `catalog_work.id` 本身等于 gid。

`catalog_work` 在 2026-07-06 按 gid 的顺序、但用**自增序列**铸号，wiki 每缺一行，后面整体错一位：

| gid 区间 | 行数 | catalog id − gid | 相等 |
|---|---|---|---|
| 1 – 4,999 | 4,921 | 中位 −6（922 之前为 0） | 920 |
| 5,000 – 9,999 | 4,704 | 中位 −171 | 0 |
| 10,000 – 59,999 | 4,452 | −185 ~ −191 | 0 |
| 60,000+（wiki 退役后论坛自编） | 338 | 中位 +144,329 | 0 |

`/galgame/1` 恰好对得上，是因为前 922 个号没有错位。论坛为了照常路由，造了一整层 gid ↔ catalog id 的映射（`curated` 外部引用 + kungal claim），而有些面干脆没走映射：收藏夹把 gid 原样当 work id 发给 catalog，catalog 里论坛写入的收藏条目有 13.5 万条存的是 gid。

moyu 在 2026-09-13 已经把自己改号到 catalog id（`kun-galgame-patch/docs/proj/gid-is-work-id.md`），并在文末写明「kungal renumbers the same way」是欠着的。

## 2. 用户裁决（2026-09-23）

1. 论坛服从 infra 的 id：`galgame.id` 一次性改号为 catalog work id，之后 **论坛作品 id ≡ `catalog_work.id`，恒等**，论坛**不存在任何 id 映射**。
2. 名字统一叫 **`work_id`**：数据库列 `galgame_id` → `work_id`，Go/TS 标识符、v1 字段与路径参数同名。`gid`、`galgame_id` 从代码里消失。
3. 作品页路径不变，仍是 `/galgame/:id`（id 即 work id）。旧号码**不做重定向**；只保住本来就相等的那部分页面的 SEO。
4. catalog 自己的作品合并也**不做 301**：被合并掉的 id 直接 404。（被合并页面上的资源、评分等数据仍要并进幸存者，否则它们会挂在一个再也打不开的 id 上。只删 301 台账，不删搬数据。）
5. 用户写的帖子里指向本站作品页的链接，在改号事务里一起改号（只改号码，正文其余一字不动）。
6. 先做 G0，G1 起等它。
7. 改完之后，把 gid ≡ work_id 作为铁律写进 `CLAUDE.md`。

## 3. 改号的形状

论坛 `galgame` 共 15,990 行，**全部可解析**，没有需要停放的行：

| 解析来源 | 行数 |
|---|---|
| `curated` 外部引用（`entity_type=5`，`external_id` = gid） | 14,417 |
| kungal claim（`catalog_work.site='kungal' AND product_work_id = gid`，恰好一条） | 319 |
| gid 本身就是 catalog id（方案③之后懒建的行，1,250 行在 200,000 以上） | 1,254 |

- 号码不变 2,493 行，改号 13,497 行。已发布的 9,713 页里，URL 号码不变的 1,656 页（1,180 页本来就相等 + 476 页懒建行）。
- `curated` 引用与 claim 同时存在时，两者**没有一处矛盾**。
- 阳性对照：四类各抽样（错位锚点 14、相等锚点 5、claim 6、懒建 5，共 30 页），抓线上 `/galgame/<旧号>` 的页面标题，与改号目标在 `catalog_work_title` 里的全部标题比对，**30/30 命中**。即改号表给每个旧页面选的 catalog 作品，就是它今天在线上显示的那部。
- **10 部 catalog 作品各被两个论坛页面占着**，改号前必须先合并（合并规则与现有的 merge fold 同一套实现）：

  ```
  work   4082 <- 4115 (已发布,1 资源)  4107 (已发布,1 资源)
  work   5628 <- 5628 (已发布,1 资源)  5739 (已发布,1 资源)
  work   6523 <- 2663 (已发布,29 资源) 6679
  work   9113 <- 9296                  7454
  work  61123 <- 61320 (已发布,3 资源) 6698
  work 211869 <- 2066 (已发布,2 资源)  211869
  work 214969 <- 62992                 62560 (已发布,3 资源)
  work 215085 <- 62677 (已发布,3 资源) 3597 (已发布,6 资源)
  work 215086 <- 4541 (已发布,6 资源)  62678 (已发布,3 资源)
  work 228620 <- 55501 (已发布,4 资源) 228620 (已发布,4 资源)
  ```

  其中 4082 与 214969 两组，moyu 改号时也遇到过同样的重复。

## 4. 要改写的数据

### 4.1 论坛库（改号命令，一个事务）

| 表.列 | 行数 | 说明 |
|---|---|---|
| `galgame.id` | 15,990 | 主键 |
| `galgame_collection_item.galgame_id` | 201,518 | 收藏夹导入前的本地快照 |
| `feed_activity.galgame_id` | 100,512 | 首页动态（触发器维护，迁移 034） |
| `galgame_resource.galgame_id` | 50,433 | FK，ON UPDATE CASCADE |
| `galgame_like.galgame_id` | 39,583 | FK，ON UPDATE CASCADE |
| `galgame_rating.galgame_id` | 4,111 | FK，ON UPDATE CASCADE |
| `galgame_activity.galgame_id` | 1,241 | |
| `galgame_quiz.galgame_id`、`galgame_quiz_galgame.galgame_id` | 267 / 11 | |
| `galgame_contributor.galgame_id` | 133 | |
| `galgame_view_daily.entity_id` | 909,190 | 浏览日桶 |
| `galgame_redirect.old_gid/new_gid`、`galgame_merge_discarded.*` | 224 / 30 | 301 台账随裁决 4 删除 |
| `galgame_favorite`、`galgame_comment_community_map` | 0 / 0 | 空表 |

链接（只改 `/galgame/` 后面紧跟的那一段数字，见 §5）：

| 表.列 | 命中 |
|---|---|
| `message.link` | 298,301 |
| `feed_activity.link` | 71,644 |
| `message.content` | 135 行 |
| `feed_activity.content` | 82 行 |
| `topic.content` / `topic_reply.content` / `topic_comment.content` | 48 / 63 / 5 行 |
| `todo.content` | 12 行 |
| `doc_article.content_markdown` | 3 行 |
| `chat_message.content` / `chat_room.last_message_content` | 49 / 4 行（私信，**待用户确认**是否算在裁决 5 里） |
| `galgame_resource_link.url` | 158 行，其中 153 条是外站 mikugame 的 `/galgame/<n>`，**只有 2 条是本站** |

### 4.2 infra（同一个窗口，infra 会话执行）

| 服务 | 数据 | 行数 | 要做什么 |
|---|---|---|---|
| catalog | `catalog_work.product_work_id`（`site='kungal'`） | 64,833 | 改写为 `catalog_work.id`。moyu 文档里 infra 欠的第 2 项；改完后 moyu 的 `forum_gid` 自动等于论坛页 id |
| community | `community_thread` 等表里 `site='kungal' AND anchor_kind=1`（site_game）的 `anchor_id` | 70,480 串 | 按改号表改写 |
| trust | `subject_kind='galgame'` 的 `subject_id` | 19 举报 + 2 审核项 | 按改号表改写 |
| catalog | `catalog_user_folder_item.work_id` 中论坛写入的 gid | 导入时 ~20 万条可精确判定 | 按改号表改写；9-07 之后新写入的约 2.7 万条**没有写入方记录**，见 §7 |

## 5. 链接改写规则

- 只认本站：无协议的相对路径，或 `http(s)://` + `kungal.com` / `www.kungal.com` / `www.kungal.org`；可带 `/<xx>-<yy>` 语言前缀（`/zh-cn`）。
- 只改 `/galgame/` 后面紧跟的那一段数字，要求其后是串尾或 `? # / ) > ] 空白` 之一。
- 不碰：`?comment=<x>` 里的 `<x>`（community 帖子 id）、`/galgame/resource/<n>`（资源 id）、`/galgame-quiz/<n>`、`/galgame-rating/<n>`、外站域名（mikugame、moyu 等，它们的 `/galgame/<n>` 是别家的号）、拼错的域名（`kungalgl.com`）。
- M 轨对 `message.link` 的普查（2026-09-22）：`/galgame/N` 298,041、`/galgame/N?comment=N&thread=N` 64、`/galgame/N?comment=N` 48、`/galgame/resource/N` 198。RC 的迁移 135 会改写带 `thread=` 的那 64 条，与本文件任意先后都成立。

## 6. 切换顺序

1. **G0a（惰性）**：改号命令 `cmd/align-galgame-ids`（plan / apply 两段，只读 plan 先出 TSV）+ 需要的准备迁移。运行中的代码不读它，可以正常部署。
2. **全量演练**：生产库全量拷贝到本地临时库，跑 plan → apply → 新代码，逐项核对不变量。
3. **窗口**：停 `kungal-api` → 快照 → `-apply`（一个事务）→ infra 同时改写 §4.2 → 合并 G0b（身份代码 + 列改名迁移）→ 部署 → 实测。moyu 第一次在实时流量下跑改号时死锁回滚（`SQLSTATE 40P01`），所以**必须停 API**。
4. 收尾：`CLAUDE.md` 写入铁律，清缓存，看日志。

## 7. 尚待决定

- **收藏条目的写入方**：catalog 没有记录每条收藏是哪个站写的。导入时的条目可以精确判定是 gid，但 2026-09-07 之后论坛与 moyu 新写的约 2.7 万条分不清。先请 infra 查请求日志能否按 OAuth client 归属；查不到再请用户裁决。
- ~~**私信里的链接**（§4.1）：是否与帖子一样改写。~~ 已改写：窗口里用 `-include-dm` 执行（`chat_message` 33 行、`chat_room` 2 行）。

## 8. 迁移号

G 轨号段 140–159。G0 预计用 140（准备）与 141（列改名、删 301 台账）。
