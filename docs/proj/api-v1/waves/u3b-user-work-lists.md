# U3b · 某用户的作品、资源、评分、评论墙帖子

> 用户轨第三段的后半。旧路由 **4 条**，都是 `UserHandler` 的方法：`GET /user/:id/galgames`、`/resources`、`/ratings`、`/galgame-comments`。迁移号段 120–129，**本段没有迁移**。
> 形状沿用 U3a 定下的「某用户的 X」约定（[u3a-user-topic-lists.md](u3a-user-topic-lists.md) §2–§3）：用户下的子集合、封闭 `relation`、资料主人不可渲染 → 404、显式 `include_nsfw`、决胜键必有。
> 资源形状由 G3 定（[g3-resources.md](g3-resources.md)），评论墙帖子的渲染与每面墙的读闸留在 wall 包（RC，2026-09-23 协调，见 §4.4）。铁律 3：作品 id 一律是 catalog work id，名字叫 `work` / `work_id`。
> 普查在 [census/user.md](census/user.md) §2.5–§2.8；契约先于实现提交，变异清单见 §7。

## 1. 普查补充（生产 2026-09-22/23）

| 事实 | 数字 / 证据 |
|---|---|
| `galgame_like` | 39,582 行 |
| `galgame_resource` | 50,433 行（有效 47,013 / 失效 3,420）；`galgame_resource_like` 8,805 行 |
| `galgame_post_like`（community 帖子点赞的本地镜像，**六面墙都镜像**，RC 确认） | 1,401 行 |
| `galgame_favorite` | 冻结于 2026-07-06，无写入方；网页的类型表早已不含 `galgame_favorite`。**v1 不提供**——收藏走 G 的收藏夹面 |
| 资源泄漏（普查 §2.7） | 旧面对任何人下发 `code` / `password`：可以按 user_id 遍历任意用户全部资源的提取码与解压密码 |
| 评论墙帖子（普查 §2.6） | 旧实现每页把作者的整个 feed 拉一遍（最多 40 页 × 100），匿名可达；坏游标静默从头开始；游标不带 `cur_`；服务端渲染 HTML |
| 上游事实（RC 读 infra `community/repository/author.go`） | `GET /authors/{id}/posts` 只回可见帖，键集 `community_post.id DESC`，`after` = 上页末尾的 post id，`limit` ≤ 100，可选单个 `anchor_kind`；**顺序稳定、按 id 新→旧**。`created` 只在导入的历史数据上与 id 不一致（056–060 保留了原始时间） |
| 调用方 | `components/user/Galgame.vue`（作品 + 评论墙两个半边）、`Resource.vue`（列表半边；编辑与标记有效 G3 已改走 `PATCH /galgame-resources/{id}`）、`Rating.vue`、`Overview.vue`（动态 tab 取作品 / 评分 / 资源各几条）；`../kungal-apps` 零命中 |

## 2. 逐条去向

| 旧路由 | v1 | 档 |
|---|---|---|
| `GET /user/:id/galgames?type=` | `GET /api/v1/users/{user_id}/works?relation=published\|contributed\|liked` | optional |
| `GET /user/:id/resources?type=` | `GET /api/v1/users/{user_id}/galgame-resources?relation=published\|liked[&state=valid\|expired]` | optional |
| `GET /user/:id/ratings` | **不新建**：网页改调 GR 已有的 `GET /api/v1/ratings?author_id={user_id}&sort=created_desc` | — |
| `GET /user/:id/galgame-comments?type=` | `GET /api/v1/users/{user_id}/wall-comments?relation=authored\|liked[&subject_type=…]` | optional |

`legacy_route_baseline` 下调 **4**。本段之后 `UserHandler` 没有方法了，连同 struct、构造与 `app.go` 字段一起删。

## 3. 共同规则

同 U3a §3：资料主人不存在或 `status != 0` → `404 NOT_FOUND`，`userclient` 失败 → `503`；`include_nsfw` 默认 `false`；未知 `relation` / `state` / `subject_type` → `400 UNKNOWN_ENUM_VALUE`；id 是十进制字符串。作品与资源是**页码**集合（`collect.PageNumber`，默认 `limit` 24 / 50 见各节，`repr.PageList`）；评论墙帖子是**游标**集合（上游只有游标）。

## 4. 四个集合

### 4.1 `GET /users/{user_id}/works` → `PageList<WorkSummary>`

条目：GE 的共享作品卡片 `workrepr.WorkSummary`（经 `workrepr.Hydrator.ByIDs`）。默认 `limit` **24**（旧 tab 的页大小）。

| `relation` | 旧 `type` | 来源 | 排序 |
|---|---|---|---|
| `published` | `galgame_publish` | 本地 `galgame`：`creator_user_id = 他 AND published = true`（今天的 `PublishedIDsByCreator`，缺 `id` 决胜键） | `galgame.created DESC, galgame.id DESC` |
| `liked` | `galgame_like` | `galgame_like JOIN galgame`：`published = true` | `galgame.created DESC, galgame.id DESC`——与旧实现同一排序对象（作品的时间，不是赞的时间；同 U3a 的 `liked`） |
| `contributed` | `galgame_contributed` | catalog 已合并的编辑提案（`GalgameUserStatsService.ContributedWorkIDs`，上限 `contributedScan = 200` 条提案）——**只能在 Go 里分页**，旧实现每页都重拉一次；保留这一点，但**整份 id 先交 catalog 水合**（`ByIDs`，按 `include_nsfw` 带 content limit），`total` = catalog 返回的条数，再切页。贡献过的作品常常没有本地行、没有 `content_limit` 镜像，本地谓词判不了（§9） | 提案列表给出的顺序，重复 work 去重保留首次出现 |

- **`show_no_resource` 不迁。** 旧面只对 `liked` 生效（`EXISTS galgame_resource`）。`published` 本身就是「发过资源」的粘性标记（方案③），两者只在资源已全删的已发布作品上不同；v1 不做这层过滤，网页不再传。

- NSFW（`published` / `liked`）：`include_nsfw=false` 排除本地 `galgame.content_limit = 'nsfw'` 的作品（NULL 放行，与 G3 K-G17 同一谓词）；**COUNT 与页共用**。
- `ByIDs` 会丢掉 catalog 里已隐藏的作品：`published` / `liked` 下该行不出现在页上，`total` 仍按 SQL 算——与 G3 的开放问题 O3 同一种已知偏差，写进 description。`contributed` 没有这个偏差（`total` 就是水合后的条数）。catalog RPC 失败 → `503`（旧实现静默回空列表 + 原 `total`，普查 bug #9）。

### 4.2 `GET /users/{user_id}/galgame-resources` → `PageList<GalgameResource>`

条目：G3 的 `resourceapiv1.GalgameResource`（`object:"galgame_resource"`）——**不带下载链接、提取码、解压密码**。取这些只能走 `POST /galgame-resources/{resource_id}/downloads`（计一次下载）或编辑者的 `GET …/source`。这就关掉了普查 §2.7 的遍历。默认 `limit` **50**。

| `relation` | 旧 `type` | 语义 |
|---|---|---|
| `published` | `valid` / `expire` | 他发的资源 |
| `liked` | `galgame_resource_like` | 他赞过的资源 |

- **`state`**（可选，`valid` / `expired`，缺席 = 都在）：与 G3 的列表同一个过滤器。旧的 `valid` / `expire` 两个「类型」其实就是这个状态过滤，不再是关系。
- **可见性与 G3 的列表完全相同**（G3 §3.4）：所属作品本地 `published = true`；`include_nsfw=false` 排除 `content_limit = 'nsfw'`；作者不可渲染的资源从 COUNT 与页中一起去掉（`liked` 下别人的资源同样适用）；`userclient` / catalog 失败 → `503`。**这几个谓词不复制**，整条管线复用 G3 的 `listGalgameResources`：
  - `repository.ResourceListFilter` 加两个字段 `UploaderID int`（`r.user_id = ?`）与 `LikedBy int`（`EXISTS (SELECT 1 FROM galgame_resource_like l WHERE l.galgame_resource_id = r.id AND l.user_id = ?)`），在 `apply` 里与其它条件并列；
  - `resourceapiv1` 导出一个方法 `ListForUser(ctx, filter, page, limit) (repr.PageList[GalgameResource], *problem.Problem)`：`DistinctAuthors → keepAuthors → Count → ClampTotal → List → assemble`，与 `listGalgameResources` 同一顺序；`assemble` 因此不必导出。
  - **`listGalgameResources` 不动**（G 的条件）：它有 `q` / catalog 命中的相关度排序与排序 token，`ListForUser` 没有；不改调，G3 的浏览面零行为风险。
  - G 已批准（2026-09-23），diff 请 G 过目。
- 排序：资源的 `created DESC, id DESC`。**`liked` 也按资源的时间，不按赞的时间**——与旧实现同一排序对象，是有意的，别「修」成按赞的时间（`galgame_resource_like` 的时间不进排序）。

### 4.3 评分：切到 `GET /ratings?author_id=`

GR 的 `listRatings` 已有 `author_id`、页码、`include_nsfw`、`sort`。网页 `Rating.vue` 改调它（`sort=created_desc`，`limit` 24），旧 `/user/:id/ratings` 删除。本段不在 U 域新建任何评分面。资料主人不可渲染时，资料页本身已是 404（U2），评分 tab 不会被打开。

### 4.4 `GET /users/{user_id}/wall-comments` → `List<WallComment>`（游标）

条目：RC 的 `wallapiv1.WallComment`——一个渲染器，内容是结构化正文，不再服务端渲染 HTML（普查 §2.6 第 3 条）。

`relation`：

| v1 | 旧 `type` | 来源 |
|---|---|---|
| `authored` | `galgame_comment` | community `GET /authors/{id}/posts`，**直接用它的 `after` 游标翻页**（id 新→旧）。不再为按 `created` 排序而整份拉取；description 写明「按帖子 id 新→旧，导入的历史帖的时间可能不单调」 |
| `liked` | `galgame_comment_like` | 本地 `galgame_post_like` 键集（`id DESC`，游标里存上页末尾的 like id）分批，经 `/posts/resolve` 取帖；已删帖的赞自然掉出。同样受下面「3 次」上限 |

`subject_type`（可选，RC 的六值词表）：`galgame` 下推为 `anchor_kind = AnchorSiteGame`；其余五种共用 `AnchorSiteResource`，按锚点前缀（`subjectSpec.anchorPrefix`）**取回后过滤**。所以页可以短于 `limit` 而仍带 `next_cursor`（与 X2 activity 同一约定）。**每个请求最多 3 次上游取页**（补足页用），匿名路径不能再被放大。网页的 galgame tab 传 `subject_type=galgame`，保持今天的范围。

**读闸留在 wall 包（RC 裁决）**：RC 的 `render` 一次只接一个墙，每面墙的读闸在 `resolveSubject`（作品在 catalog 缺失 → 看不到；评分 / 资源 / 工具 / 题目的主人被封 → 看不到；题目 `hide_galgame` 或带剧透 → 只有作者、持该墙编辑 / 删除权限者、答过题的人能看）。个人主页列表绕过这些闸就会泄漏剧透题的评论——正是 `getWallComment` 已经堵住的那个洞。所以在 `internal/wall/apiv1` 导出（本 PR 加，RC 审 diff）：

```go
func (s *Service) RenderAuthored(ctx context.Context, viewer *middleware.UserInfo,
    rows []communityclient.AuthorPostView) ([]WallComment, *problem.Problem)
```

- 保持输入顺序；每个不同的 (spec, id) 墙只 resolve 一次，按墙分组 render，再按输入顺序拼回。
- **galgame 墙批量判存在**（RC 审查的阻断项）：`resolveSubject` 的 galgame 分支是一次不缓存的 catalog 详情调用；一页评论跨 20 部作品就是串行 20 次、全从论坛唯一出口 IP 打 catalog 的每 IP 限流。`wallapiv1.New` 多收一个 `GalgamesResolver`（接 `CatalogRowsByWorkIDs(ids, "", "all")`，≤100 个 id 一次请求；隐藏的 claim 不返回、`all` 保留 NSFW，与详情判存在同义），`RenderAuthored` 先一次查完所有 galgame 墙，其余墙照走 `resolveSubject`；`resolveSubject` / `locate` 不动。
- **丢弃、不报错**：锚点不认识的行；墙 `resolveSubject` 回 `NOT_FOUND` 或 `QUIZ_ANSWER_REQUIRED` 的行；被扣留（held）且不是查看者本人的帖；作者不可渲染的帖（`render` 本来就跳过）。
- 只有 `resolveSubject` / `render` 的 `Unavailable` 或 `Internal` 让整次调用失败（503 / 500）。
- 名字叫 Authored，`liked` 的行（`/posts/resolve` 回的同一形状）也用它。
- **先按锚点过滤、再渲染**（RC）：`subject_type` 的过滤用导出的 `SubjectTypeOfAnchor(kind, anchorID)`（建在 `subjectFromAnchor` 之上，一份映射），在调 `RenderAuthored` 之前做——要丢的墙永远不 resolve，被滤掉的行上的 catalog / OAuth 抖动不会把整页变成 503。
- 测试（RC 要求）：某行的墙 `Unavailable`（如 galgame 锚点的假 catalog 挂掉）→ 整次调用 503——这是唯一**不能**丢的错误路径；同墙两行的批次只 resolve 一次（数假 store / catalog 的调用次数）。
- **假上游照抄 infra**（`platform/community/handler/author.go` + `repository/author.go`；U1 的教训是假上游形状与真的不同）：只回可见帖（held、deleted 连作者本人的也不回——所以 `RenderAuthored` 的 held 规则只对 `/posts/resolve` 的行生效）；`post id DESC`；`after` 不含（`id < after`）；`limit` 缺省 20、夹到 100；`anchor_kind` 缺省 −1 = 全部（客户端 < 0 时不传）；`len(rows) == limit` 时 `next_cursor` = 末行 id，否则 `""`——**满的最后一页也带游标、下一次是空页**，补页循环把「空页且无游标」当结束，不是错误；条目 `{post, thread:{thread_id, title, anchor_kind, anchor_id}}`。

游标：query 参数名 `cursor`（v1 标准），值 `cur_…`，**指纹绑定 `relation` + `subject_type` + 资料主人**；换了过滤条件复用旧游标 → `400 INVALID_CURSOR`（旧实现坏游标静默从头开始）。`limit` 1–100，默认 24。

## 5. 错误码

**不新增**。`NOT_FOUND`、`UNKNOWN_ENUM_VALUE`、`INVALID_PARAMETER`（页码深度）、`LIMIT_TOO_LARGE`、`INVALID_CURSOR`、`SERVICE_UNAVAILABLE`、`INTERNAL_ERROR`，以及 optional 档的 `INVALID_CREDENTIAL`。

## 6. 网页

| 文件 | 改什么 |
|---|---|
| `components/user/Galgame.vue` | 作品半边 → `/users/{user_id}/works`（`galgame_publish` → `published`、`galgame_contributed` → `contributed`、`galgame_like` → `liked`），卡片按 `WorkSummary` 渲染（照 GE / G 已迁的作品卡片组件）；评论半边 → `/users/{user_id}/wall-comments?subject_type=galgame`（`galgame_comment` → `authored`、`galgame_comment_like` → `liked`），用 RC 的评论墙渲染组件显示 `WallComment`，游标翻页 |
| `components/user/Resource.vue` | 列表半边 → `/users/{user_id}/galgame-resources`（`valid` → `relation=published&state=valid`、`expire` → `relation=published&state=expired`、`galgame_resource_like` → `relation=liked`），条目按 `GalgameResource` 渲染（照 G3 已迁的资源卡片）；**下载必须走 G3 的 downloads 面**，列表不再有链接 |
| `components/user/Rating.vue` | → `GET /ratings?author_id=`，条目按 GR 的 `RatingSummary` 渲染（照 GR 已迁的评分卡片） |
| `components/user/Overview.vue` | 动态 tab 里剩下的三处旧调用（作品 / 评分 / 资源）一起切；`show_no_resource` 不再传（§4.1） |
| 手写类型 | 删掉无人再用的 `UserGalgame` 等 |

tab 的 URL 段不变（`/user/:id/galgame/galgame-like` 等），只在调用处把旧 `type` 映射成 `relation` / `state`。`CHANGELOG.md` 记一条。

## 7. 变异清单（先于实现提交）

| # | 破坏 | 应当变红的性质 |
|---|---|---|
| 1 | 作品 `published` 关系去掉 `published = true` | 未发布的作品不出现 |
| 2 | 作品 NSFW 谓词只作用于页、不作用于 COUNT | `total` 等于能翻到的行数 |
| 3 | 作品排序去掉 `id` 决胜键 | 同一时间的行跨页全量遍历无重无漏 |
| 4 | `contributed` 先切页再过滤 NSFW | 默认不含 NSFW 且 `total` 正确 |
| 5 | 资源不要求所属作品 `published = true` | 未发布作品上的资源不出现 |
| 6 | `liked` 资源不按作者可渲染过滤 | 被封作者的资源不出现，`total` 同步 |
| 7 | 忽略 `state` | `state=valid` 不含失效资源 |
| 8 | `RenderAuthored` 不丢 `QUIZ_ANSWER_REQUIRED` 的行 | 陌生人看不到剧透题下的评论，作者 / 答题者看得到 |
| 9 | `RenderAuthored` 保留别人的 held 帖 | held 帖只对本人出现 |
| 10 | 评论墙游标指纹去掉 `subject_type` | 换 `subject_type` 复用游标 → `INVALID_CURSOR` |
| 11 | 取消每请求 3 次上游取页的上限 | 假上游记到的调用次数 ≤ 3 |
| 12 | 资料主人不可渲染时仍返回（任一集合） | 封禁用户 → 404 |
| 13 | `LikedBy` 移出 `apply`、只加在页查询上（COUNT 不带它） | `relation=liked` 的 `total` 等于全量遍历到的条目数（同 G3 #3 钉作者谓词的方式；G 的条件） |

## 8. 删旧路由

4 条全删，连同 `UserHandler`（已无方法）、`UserContentService` 里只被它们用到的部分、`content_repo.go` / `galgame_comment_pagination.go` 里的死代码、`dto.UserGalgameComment` 等。`deadcode -test ./...` 到不动点。`routes.golden` 重生成，`legacy_route_baseline` 下调 4。

## 9. 验收时对契约的更正（2026-09-23）

实现、变异与浏览器实测抓出的，已改在上文对应位置，这里留底：

| 原契约 | 改为 | 原因 |
|---|---|---|
| `contributed` 先用本地 `content_limit` 滤 NSFW，再算 `total`、切页、水合 | 整份 id 交 catalog 水合，`total` = 返回条数，再切页 | 浏览器实测（dev，用户 2）：`total` 43，SFW 三页合计只有 7 条——没有本地行的 NSFW 作品本地放行、catalog 拒绝水合，正是普查 bug #6「`total` 与 `items` 不同谓词」 |
| 游标参数叫 `after` | `cursor` | v1 标准的 query 名；上游的 `after`（post id / like id）存在游标载荷里 |
| `listGalgameResources` 可以改调 `ListForUser` | 不动；`ListForUser` 单独一份 | G 的条件：浏览面有 `q` / 相关度排序，零行为风险 |
| 网页资源 tab：列表不再有链接，编辑走 `LinkEditModal` | 同上，另加：过期 tab 保存链接后再 `PATCH {state: valid}`；每行加「标记为有效」 | 旧 tab 是「改链接并标记有效」一步完成；`LinkEditModal` 只改链接，不带 `state` |
| 评论 tab 首页为空即显示「没有评论」 | 有 `next_cursor` 就显示「加载更多」 | 短页是契约允许的（§4.4），首页可能整页被丢（dev：`limit=3` 不带 `subject_type` 回 0 条 + 游标） |
| 未知枚举测试用旧值（`galgame_publish`） | 用长度在 `maxLength` 以内的值 | 超长值同时触发 TOO_LONG，平台映射成 `INVALID_PARAMETER` 而不是 `UNKNOWN_ENUM_VALUE`；这是 `internal/apiv1` 的既有行为 |
| 变异 #4「先切页再过滤」 | 「只水合当前页、`total` 按全部 id」 | 实现改了，变异跟着指向同一个 bug |
| — | 新增变异 #13（G）、RC-batch、RC-503、RC-503-batch（RC） | 见 §4.2 / §4.4 的条件 |
| galgame 墙逐个 `resolveSubject` | 一次批量判存在（§4.4） | RC 审查：逐个是串行的不缓存详情调用，按页数放大打 catalog 的每 IP 限流 |
| 变异 RC-once（去掉按墙去重） | 删去 | 批量之后它不改变任何可观测行为：重复的 galgame 墙落进同一次批量，其余墙的 resolve 只多一次本地查询，分组本来就按墙去重——等价变异 |

## 10. 迁移

**无**。
