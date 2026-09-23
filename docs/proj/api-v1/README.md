# 论坛 API v1 · 重整规范

> 本仓自有工程文档，**不是** infra 镜像。状态：📐 2026-09-18 立项，试点域 = 话题（topic）。

## 缘起与授权

2026-09-18，用户在评估 App 直连方案时拍板：

- 「一切 api 的写法全部按照现代最佳实践做，infra 应该就是一个比较完善的案例了，论坛这边缺什么就补上什么，不标准的全部做标准，一个 api 一个 api 的进行迁移。」
- 「这次一定要做好最严格的测试和防漂移措施……部署之前就把所有的问题全部扼杀。」
- 「目前的很多字段命名非常随意……全部重命名，趁这次 api 重构把问题全部解决。」
- 错误码：「以后绝对会再加回来（i18n），所以类似于现在这样硬编码的错误是完全不可行的……错误码这方面需要严格遵循现代最佳实践。」
- 正文格式：**结构化节点 JSON**（同日 AskUserQuestion 拍板）。
- 分页：话题列表改游标 +「加载更多」；/galgame 这类浏览页必须保留分页器，「用符合现代最佳设计的方式做」→ 见 [01 §4](01-standard.md)。

本规范取代 2026-09-17 的「App 复用 `/api/*` 与原信封」决定（`docs/proj/app-direct-api.md` §0、kungal-apps 03 §7）。**App 只绑定 `/api/v1`**；旧 `/api/*` 只剩网页在用，随网页迁移逐条删除。

## 现状的问题（立项依据，2026-09-18 普查）

| 问题 | 证据 |
|---|---|
| 错误码不能当判据 | `pkg/errors`：`233` 一个码覆盖约 750 处调用（`ErrBadRequest` 300 / `ErrInternal` 243 / `ErrNotFound` 121 / `ErrForbidden` 65 / `ErrValidation` 21），客户端只能读中文 `message` |
| 「登录失效」是一个长得像 HTTP 状态码的数字 | `205` 是 HTTP 401 响应体里的业务码 |
| 前后端类型无机械约束 | 网页 `shared/types/` 42 个手写文件、2472 行；请求体完全无类型；snake_case 迁移时找出约 40 处发送字段名不对 |
| 命名随意 | `gid` / `tid`、`created`、`view`、`section`（数组）、`comment`（数组）、`is_nsfw_topic` 与 `is_nsfw` 同义不同名、`status` 是裸整数 |
| 动词/非幂等 PUT | `PUT /topic/:tid/like` 是切换（重放一次就撤销）；`/lottery/draw`、`/enter` 等动词路径 |
| 分页 | `len(rows) == limit` 判末页、排序无 tie-breaker、未知排序字段静默回落到 `created` |

## 与 infra 规范的关系

infra `refs/api-v2/`（01 公理与黑名单 · 02 协议 · 04 表示 · 05 集合 · 06 写面 · 07 治理 · 10 错误码）**默认全部适用**，本目录只写三类东西：

1. infra 条款在论坛**不适用**的地方，以及原因；
2. 论坛自己的决定（编号 **K1…**，只增不改，理由写在决定旁边）；
3. 论坛的具体词表：命名表、`kungal` 错误域、资源类型。

阅读时先读 infra 那几章，再读这里。infra 文档在源工作区 `nextmoe-infra/refs/api-v2/`，只读。

## 阅读顺序

**新开一个会话来做一个域，先读 [05-session-sop.md](05-session-sop.md)**，它是作业手册，其余是它引用的材料。

| 文件 | 内容 |
|---|---|
| [05-session-sop.md](05-session-sop.md) | 独立会话作业手册：认领、worktree、临时库、五步、九条闸、PR、上线 |
| [01-standard.md](01-standard.md) | 面与凭证、错误与 i18n、表示层、集合、写面、正文文档、命名表 |
| [02-governance.md](02-governance.md) | 契约单一来源、代码生成、CI 门、演进与退役、App 兼容、逐端点迁移清单 |
| [04-parallel-tracks.md](04-parallel-tracks.md) | 并行轨协议：五条车道、共享面预分配、九条测试闸、合并策略 |
| [03-content-doc.md](03-content-doc.md) | 正文节点树：形状、普查、Markdown 转换规则、客户端渲染 |
| [waves/](waves/) | 每一波的任务书与验收记录 |

## 波次看板

**认领方式：把空分支推上去。** 远端分支就是认领表，没有别的地方登记：

```bash
git ls-remote --heads origin 'api-v1/*'
```

分支名 `api-v1/<波次>-<域>`。迁移号只能从自己那一段里取（W3 与 W4 撞过 098）；104–109 留给在途修复。

### 已完成

| 波 | 范围 | 状态 |
|---|---|---|
| W0a | 后端地基（huma、problem 注册表、身份、幂等、游标、spec 生成、CI 门、DB 测试进 CI）+ `GET /api/v1/topics` | ✅ 2026-09-18 |
| W0b | 前端地基（生成类型、类型化客户端、错误本地化目录）+ 话题列表切到 v1（[记录](waves/w0b-frontend-foundation.md)） | ✅ 2026-09-19 |
| W1 | 结构化正文文档（[规格](03-content-doc.md)、[验收](waves/w1-content-doc.md)） | ✅ 2026-09-19 |
| W2 | 话题详情 + 回复读面（[记录](waves/w2-topic-detail.md)） | ✅ 2026-09-22 |
| W3 | 话题 / 回复写面（[记录](waves/w3-topic-writes.md)） | ✅ 2026-09-22 |
| W4 | 互动：点赞、收藏、推、表情、最佳答案、置顶（[记录](waves/w4-interactions.md)） | ✅ 2026-09-22 |
| W5a | 话题评论（[契约](waves/w5a-comments.md)，迁移 101） | ✅ 2026-09-22 |
| W5b | 投票（8 个 v1 端点取代 6 条旧路由，[契约](waves/w5b-polls.md)，迁移 103） | ✅ 2026-09-22 |
| RC | 评论墙：六面墙的 22 条旧路由 → `/api/v1/wall-comments` 9 个操作（[契约](waves/rc-wall-comments.md)，迁移 135，PR #180） | ✅ 2026-09-23 |

### 待认领

旧 `/api/*` 路由 **260 条**。`legacy_route_baseline` = 261，因为它数的是「所有非 `/api/v1` 的路由」，`/healthz` 也在里面——**基线的地板是 1，不是 0**。

> **切分依据是「共用同一个老 handler」，不是 URL 前缀。** 这个仓库里老 handler 大量跨前缀：`ResourceCommentHandler` 一个人管 15 条 / 5 个前缀；`/api/admin/**` 的 24 条分属 11 个 handler、各归各的域；`/api/user/:id/toolsets` 是 toolset 的面；`GET /api/resource` 是 `TopicHandler.GetResourceList`，即话题列表的资源区分面。**按前缀分轨会让两个 session 撞在同一个 handler 上，而且谁也删不掉它**——共用 handler 要等它服务的**所有**前缀都迁完才能删。下表按连通分量切，每行对外零耦合。

| 轨 | 模块（含跨前缀的归属） | 路由 | 迁移号段 | 备注 |
|---|---|---|---|---|
| **T** | **话题**：`/topic/**` 11 + `/admin/topic*` 3 | **14** | 110–119 | 已认领（本轨）。~~**T1** 清理旧评论/投票 10 条~~ ✅ → ~~**T2** 草稿 4 + `interactions/mine` + `reply/locate` + 删 `/resource` 共 7 条~~ ✅（迁移 110）→ **T3** 抽奖 11 条 → **T4** 管理面 3 条（在 `internal/admin/**`，页码集合） |
| U | 用户 `/user/**`（不含「某用户的 X」列表面，那些归各自的域） | 25 | 120–129 | 资料、签到、偏好、创作者、红点 |
| M | 消息 `/message/**` | 11 | 130–134 | 私信 + 系统通知 |
| G | galgame 主域 + `-edit` + `-quiz` + `-resource` + `toolset` + 各自的 admin/user 面（**不含** `/galgame/:gid/comments*` 与 `/galgame/comments/*`，已归 RC） | 85 | 140–159 | 最大的一坨，**一个 owner**，内部自己切 3–4 个 PR 串行 |
| GE | galgame 实体六件套 `-character` `-engine` `-official` `-series` `-staff` `-tag` | 18 | 160–164 | 共用 `EntityHandler`，必须同一轨 |
| D | 文档 `/doc` + `/website-tag` + `/website-category` | 26 | 165–169 | 共用 `TagHandler` / `CategoryHandler` |
| UP | 更新日志 `/update/**` | 11 | 170–174 | |
| WS | 站点 `/website` + `/website-tag-group` | 11 | 175–179 | |
| TS | 举报与信任 `/report` `/trust` + `/admin/trust*` | 7 | 180–184 | |
| P | 权限 `/perm` + `/admin` 的权限面 | 7 | 185–189 | |
| GR | galgame 评分 `/galgame-rating` | 6 | 190–194 | |
| X | 零散：`/search` 6、`/friend-link`(+admin) 5、`/news` 4、`/image` 4、`/ranking` 3、`/community` 3、`/auth` 3、`/activity` 3、`/rss` 2、`/category`+`/section` 2、`/admin` 总览 2、`/home` 1、`/app` 1 | 39 | 195–209 | 可拆成几个小 PR |

**不存在「admin 轨」**：`/api/admin/**` 是 11 个 handler 各自域的管理面（`TopicAdminHandler`→话题、`PurgeHandler`→用户、`ArticleHandler`→文档、`TrustHandler`→信任…），各域迁自己的。

**同时开几个**：建议 **4–5 条并行**，做完一条补一条。瓶颈不是 session 数量，是①合并即部署且必须串行、②`registry.go`/`problem.json`/`router.go`/`app.go`/两个金文件是全轨共用、③你自己的注意力。首批推荐 **T（本轨）+ U + M + RC**，想再加一条就上 **G**（它最长，早开早完）。

执行方式：**每个域一个独立会话**，在自己的 worktree 里按 [05-session-sop.md](05-session-sop.md) 从普查做到 PR；合并即上线。2026-09-22 之前是「督查派发 cursor-agent + 直接落 master」，已由 PR 流程取代——原因见 [04 §8](04-parallel-tracks.md)。
