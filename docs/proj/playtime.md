# 游玩时长

论坛不存储任何游玩时长。逐条记录、跨应用折叠、公开中位数全部在 catalog（infra）里，论坛是 BFF。

## 三层数据

| 层                                                              | 位置                                                      | 谁能读                                                     |
| --------------------------------------------------------------- | --------------------------------------------------------- | ---------------------------------------------------------- |
| 逐条上报 `catalog_user_playtime(actor_uid, work_id, client_id)` | catalog                                                   | 只有本人（`GET /v2/me/playtimes`、`GET /v2/me/playtimes/{id}`） |
| 跨应用折叠                                                      | catalog 读时计算，`MAX(minutes)`                          | 只有本人                                                   |
| 公开中位数 `catalog_work_playtime(source_id = nextmoe)`         | catalog，`jobs/userplaytime` 夜跑写                       | 所有人（`GET /v1/catalog/works/{id}` 的 `playtimes`）      |

公开那一层的门槛由 infra 定：只统计 `state = done` 的上报者、每人取跨应用 `MAX`、`minutes ∈ [10, 60000]`、`percentile_disc(0.5)`、**至少 3 位上报者**，不够格的行会被删掉。论坛不复制这套判断，只在前端画 `minVotes` 那条线（`constants/galgame-playtime.ts`）。

## 论坛这一侧

- 授权 scope 在 `apps/web/app/utils/oauth-auth.ts`：`playtime:read playtime:write`。这两个 scope 在 infra 是 self-service，不需要申请。
- 老 token 里没有它们，upstream 回 403，`catalogclient.ErrInsufficientScope` → `errors.ErrReauthRequired`，提示重新登录。和 `catalog:edit` 当初一样，不做全站 session bump。
- 写：`PUT /galgame/:gid/playtime`，`{minutes?, status?}` **两个字段互相独立，各自缺席即不动那一轴**，两个都缺席是 400。`minutes` 是**绝对累计值，不是增量**；`minutes = 0` 就是撤回时长，`status: ""` 是删除状态（`DELETE /v2/me/work-states/{id}`，404 也算成功）。时长写 `PUT /v2/me/playtimes/{id}` `{minutes}`，状态写 `PUT /v2/me/work-states/{id}`；写完再读一次折叠分钟回来，因为用户的其它应用可能报了更大的数，直接回显自己写的会和页面上的数字打架。
- 读：galgame 详情里的 `my_playtime`（和封面投票一样是逐请求带 token 的水合，不进缓存）；`GET /galgame/playtime/mine` 是个人页的列表。
- `/mine` 是同步面不是浏览面：v2 按变更升序 + `cursor=`。**`/v2/me/playtimes` 和 `/v2/me/work-states` 各扫一遍再取并集**，因为只标了状态没记时长的作品不在 playtime 里；论坛一次每面最多扫 10 页 × 100 条，本地折叠（MAX 分钟、按扫到的行数计 `clients`），先按每部作品在行流里最后一次出现的下标倒序（v2 行不再带时间戳，这是「最近改动优先」的代理）排 playtime 那批，只有状态的排在后面（`minutes: 0, clients: 0`），再分页；任一面扫不完时 `truncated = true`。`FinishedWorks` 只数三个 `done_*`，只读的 `done` 不计。

## 游玩状态（work-state）

状态不在 playtime 行上。catalog 另有 `/v2/me/work-states/{id}`（列表 `/v2/me/work-states`），两根轴：`state ∈ wish|doing|done|on_hold|dropped`，可选 `completion ∈ one_route|main|all`。公开中位数只计 `state = done` 的上报者。

论坛把这两根轴压成**一张扁平的 7 值词表**，因为所有展示点（徽章、计数、`/galgame-rating` 的筛选面）都只读一个字段：

| 论坛 | catalog |
| --- | --- |
| `wish` 想玩 | `wish` |
| `doing` 游玩中 | `doing` |
| `done_one_route` 单线通关 | `done` + `one_route` |
| `done_main` 主线通关 | `done` + `main` |
| `done_all` 全线通关 | `done` + `all` |
| `on_hold` 搁置中 | `on_hold` |
| `dropped` 已弃坑 | `dropped` |

映射只在 `internal/galgame/playstate` 一个包里（前端对应 `constants/galgame-playtime.ts`）。另有第 8 个值 `done`「已通关」**只读不可写**：别的应用报了 done 却没报完成度时 catalog 就返回它，读到必须渲染成「已通关」而不是空徽章；`playstate.Valid("done") == false`，客户端送它是 400。

写状态时**不再先 GET 回填 `completion`**。那一步存在的原因是论坛以前只能说 done、说不出通关到哪；现在用户直接在选择器里挑单线/主线/全线，他的选择就是权威。

`galgame_rating.play_status` 用的是同一张词表（迁移 094 把 `not_started/in_progress/finished_*` 全量改写过来）。传播是**单向**的：发布或编辑评分会把状态推到 catalog（`PlaytimeService.SyncWorkState`，best-effort，失败只 `slog.Warn`——评分已经落库了，catalog 挂了不能让用户看到失败）；反过来改 work-state **永不**回头改已发布的评分，评分是快照。

## 第三方客户端

桌面客户端 / tracker 走 infra 开发者平台自建应用，**论坛不需要任何改动**：

- `PUT /v1/playtime/by-ref/{source}/{externalID}` 让客户端用手里的 vndb / dlsite id 直接上报，不必先查 work id。
- `POST /v1/playtime/batch`（≤ 200）是首次登录的库存同步。
- 逐条记录按 `(user, work, client_id)` 唯一，所以论坛和客户端是**并列**关系，不是覆盖关系。v2 个人列表不再带 `client_id`，论坛不再把某一行标成「其它应用」；`clients` 仍是扫到的每部作品的应用行数。
- 聚合作业有 `--exclude-clients`：某一条上报通道被刷了，可以整条摘掉而不动别人的数据。这也是「先允许手填」的底气。

## 两条不要碰的线

- **不要用萌萌点奖励上报时长。** 时长能换积分，中位数当天变成垃圾。要奖励就奖励写评分（那条路上有审核和 T&S）。
- **个人页是私密的。** `/mine` 只返回本人的行，`/user/:id/playtime` 也只对本人渲染。它会暴露用户玩过哪些 R18 作品，所以列表照常走站点的 SFW 开关。

## 冷启动

三人门槛意味着上线后相当长一段时间里，绝大多数作品不会有 `nextmoe` 中位数——抽 41 部作品验证时该源一行都没有。前端写成「有就显示、没有就整块不渲染」，所以不会出现空槽。
