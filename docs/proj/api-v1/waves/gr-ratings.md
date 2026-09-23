# GR · galgame 评分

> 契约与变异题同一个提交，早于实现。基于 `api-v1/gr-rating`（= GE 分支 `api-v1/ge-entities`，因为评分详情嵌 GE 的 `WorkSummary`；GE 合并后 rebase 到 master）。生产数字实测于 2026-09-23。

## 0. 范围

`RatingHandler` 管 6 条旧路由，只有这 6 条：

| # | 旧路由 | 档位 | 调用方 |
|---|---|---|---|
| 1 | `GET /galgame-rating/all` | public | 评分列表页 `components/galgame/rating/card/Container.vue`、sitemap（Nitro） |
| 2 | `GET /galgame-rating/:id` | optional | 评分详情页 `pages/galgame-rating/[id].vue` |
| 3 | `POST /galgame-rating` | required | 发布表单 `rating/Publish.vue`（挂在 G 的作品页上） |
| 4 | `PUT /galgame-rating/:id` | required | 同上，编辑模式 |
| 5 | `DELETE /galgame-rating/:id` | required | 详情页 `rating/detail/Detail.vue` |
| 6 | `PUT /galgame-rating/:id/like` | required | 详情页 `rating/detail/Like.vue` |

Flutter App 零调用；`docs/proj/app-direct-api.md` 没有这些路由。

**不在本轨**：作品页上嵌的评分列表（G 的作品详情负责，复用本轨的 `RatingSummary`）；`GET /user/:id/ratings`（U3，建议直接用本轨的 `?author_id=`）；评分的评论墙（RC 已迁，`subject_type=galgame_rating`）；外部数据源评分（catalog，G）。

## 1. 普查

### 1.1 生产数字（`kungalgame`，2026-09-23）

| 事实 | 数字 |
|---|---|
| 评分 | 4,119 条，1,582 部作品，1,129 个作者；同一作者同一作品 **0** 条重复 |
| `recommend` | `yes` 1,660 · `neutral` 1,099 · `strong_yes` 1,057 · `no` 215 · `strong_no` 88（五值，库外 0 行） |
| `play_status` | `done_all` 2,880 · `done_main` 455 · `done_one_route` 386 · `wish` 177 · `doing` 141 · `dropped` 73 · `on_hold` 7 |
| `spoiler_level` | `none` 3,113 · `portion` 812 · `serious` 194 |
| `overall` | 全部落在 1–10 |
| 八个分项 | 全部落在 0–10；0 = 没打这一项 |
| `galgame_type` | 全部是非空数组；取值 `plot` / `daily` / `ba_saku` / `moe` |
| `short_summary` | 1,032 条为空，最长 1,314（= 上限）；1,056 条有换行；含 `/image/` token 0 条、@ 提及 0 条、HTML 0 条；含 `||` 的 3 条是用户把发布表单的提示文字粘了进去。网页用 `KunText` 当纯文本渲染 |
| 点赞 | 609 行，`like_count` 与点赞行**零漂移**，自赞 0 |
| 挂在未发布作品上的评分 | 55 条（作品被隐藏/封禁下架）；没有本地行的 0 条 |
| `comment_count` 非零 | 79 条，合计 132（评论墙维护） |
| `view` | 最大 12,079，合计 1,368,978 |

### 1.2 现行语义

- 列表 `GET /galgame-rating/all`：页码（`page`/`limit` ≤ 50）、`sort_field` ∈ `time`/`view`/`overall`（其余回落到 `created`）、`sort_order`、`spoiler_level`/`play_status`/`galgame_type` 三个过滤。先 SQL 数 `total`、取一页，再在渲染时丢掉封禁作者与 SFW 下看不到的作品（`GetBatchPublic`）。
- 详情：作者被封禁 → 404；每读一次 `view + 1`（作者自己读也算）；`liked_users` 是全部点赞者；作品块走 catalog 详情 + 本地评分和/数。
- 发布：同一作者同一作品只能一条；trust 检查（deny → 拒绝，hold → 记日志），后台扫描；按短评长度发萌萌点（`ratingReward`）；懒建本地作品行；写完同步 catalog 的游玩状态（`SyncWorkState`）。
- 编辑：只有作者；萌萌点按新旧短评长度差补发/扣回；同样同步游玩状态。
- 删除：作者本人、**作品页的创建者**、或持 `rating.delete_any` 者；扣回发布时的萌萌点。
- 点赞：`PUT` 当切换用；不能赞自己；作者 ±1 萌萌点；给作者发一条「liked」通知（按发送者+接收者+链接去重，所以反复点不会刷屏）。

### 1.3 疑似 bug

| 编号 | 事实 |
|---|---|
| R1 | 列表的 `total` 在丢行之前数：封禁作者、SFW 下的 NSFW 作品都算进总数，页会变短、页数虚高 |
| R2 | 列表排序没有 `id` 决胜：`overall` 只有 10 个值，翻页必然重/漏 |
| R3 | 详情不看作品可见性：55 条挂在已下架作品上的评分照读 |
| R4 | 发布不验证作品是否存在：任意 `galgame_id` 都会懒建一行本地作品并写入评分 |
| R5 | `recommend`、`spoiler_level`、`galgame_type` 不校验取值：什么字符串都收 |
| R6 | 写面都不看会话用户是否已被封禁（封禁只在刷新令牌时发现） |
| R7 | 点赞用 `PUT` 切换，重放会翻转状态 |
| R8 | `galgame_type` 过滤用 `fmt.Sprintf` 拼 JSON：值里带引号就拼出坏 JSON，查询出错被吞，返回空列表 |
| R9 | 详情的 `liked_users` 不设上限 |
| R10 | 编辑只回一句「评分更新成功」，客户端拿不到新状态 |

## 2. 裁决

| # | 裁决 |
|---|---|
| D1 | 名词 `ratings`，`object: "rating"`（列表 `RatingSummary`，详情 `Rating`）。G 确认：作品详情不嵌论坛评分（作品页读 `GET /ratings?work_id=`），五源外部评分沿用 `external_ratings`，`ratings` 归本轨 |
| D2 | 列表 = 页码集合（旧页有分页器）。可见性在 SQL 里做：读者的 SFW 闸走本地作品的 `content_limit` 缓存（迁移 079），`total` 与 `items` 同一谓词（修 R1 的作品一半）。**封禁作者在渲染时丢**，与话题、评论墙一致——封禁只有 OAuth 知道；`total` 因此可能比页里多出封禁作者的条数，描述里写明 |
| D3 | 排序 `sort` ∈ `created_{desc,asc}` `view_{desc,asc}` `overall_{desc,asc}`，默认 `created_desc`，一律 `id` 同向决胜（修 R2） |
| D4 | 过滤 `work_id`、`author_id`（U3 与 G 的作品页可以直接用）、`spoiler_level`、`play_status`、`game_type`（与 GE 作品子集合同词表，不含 `uncategorized`）、`include_nsfw` |
| D5 | 详情：作品在读者的内容闸外（NSFW 且 `include_nsfw=false`）照旧返回——正文不是成人内容，`work.is_nsfw` 交给客户端决定怎么显示（与话题同）。作品已下架（本地 `published=false`）的评分 → 404（修 R3）。作者被封禁 → 404（照旧） |
| D6 | 发布前向 catalog 确认作品存在，否则 `422 VALIDATION_FAILED` + `/work_id` `UNKNOWN_REFERENCE`（修 R4）；同一作品已评 → `409 ALREADY_EXISTS` |
| D7 | 词表全部封闭枚举，由 schema 校验（修 R5、R8） |
| D8 | 写面先按 OAuth 当前记录判会话用户，封禁 → `403 ACCOUNT_BANNED`；OAuth 不可用 → `503`（K17，修 R6） |
| D9 | 点赞是 K16 槽位 `PUT`/`DELETE /ratings/{rating_id}/like`，幂等，回 `RatingEngagement`（修 R7）；自赞 `403 SELF_LIKE_FORBIDDEN` |
| D10 | 删除的三种权力照旧（作者、作品页创建者、`rating.delete_any`）。Bearer 请求没有 staff 权力（`user.Can`） |
| D11 | 编辑是 `PATCH`，回完整 `Rating`（修 R10）；K18：短评没变不重跑 trust 检查 |
| D12 | 详情带最近的 ≤50 个点赞者（修 R9），总数看 `like_count` |
| D13 | 分项分数 0 在 v1 是 `null`（没打），写入时 `null` 存 0 |
| D14 | 浏览计数照旧每读 +1（R 之外的旧行为，不改） |
| D15 | 无迁移（号段 190–194 不用） |

## 3. 形状

### 3.1 操作（7 个，取代 6 条旧路由）

| operationId | 方法 路径 | 档位 |
|---|---|---|
| `listRatings` | `GET /ratings` | public（带凭证时算 `viewer`） |
| `getRating` | `GET /ratings/{rating_id}` | optional |
| `createRating` | `POST /ratings` | required，`Idempotency-Key` 必带 |
| `updateRating` | `PATCH /ratings/{rating_id}` | required |
| `deleteRating` | `DELETE /ratings/{rating_id}` | required，204 |
| `likeRating` | `PUT /ratings/{rating_id}/like` | required |
| `unlikeRating` | `DELETE /ratings/{rating_id}/like` | required |

### 3.2 `RatingSummary` / `Rating`（`object: "rating"`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | id | |
| `work` | `WorkRef \| null` | X2 的 `work` 规则：恒为可空的 `repr.WorkRef` |
| `author` | `UserRef` | |
| `recommend` | `strong_yes` `yes` `neutral` `no` `strong_no` | |
| `overall` | int 1–10 | |
| `game_types` | `[ba_saku \| plot \| moe \| daily]` | 至少一个 |
| `play_status` | `wish` `doing` `done_one_route` `done_main` `done_all` `on_hold` `dropped` | 词表取自 `internal/galgame/playstate` 的常量，G6 的游玩状态面同名同词表 |
| `spoiler_level` | `none` `portion` `serious` | |
| `short_summary` | string ≤1314 | 纯文本，空串 = 没写。不是正文文档：库里没有任何图片 token 或提及（§1.1），网页一直按纯文本渲染 |
| `aspect_scores` | `{art, story, music, character, route, system, voice, replay_value}`，各 `int 1–10 \| null` | D13 |
| `view_count` `like_count` `comment_count` | int ≥0 | |
| `created_at` `updated_at` | date-time | |
| `viewer` | `{has_liked, can_edit, can_delete} \| null` | 匿名 `null` |
| `work_summary` | `WorkSummary` | 只在 `Rating` |
| `likers` | `[UserRef]` ≤50 | 只在 `Rating`，最近在前 |

`RatingEngagement` = `{object: "rating_engagement", rating_id, like_count, viewer: {has_liked}}`。

### 3.3 写面请求体

`RatingCreate`：`work_id`、`recommend`、`overall`、`game_types`（1–4 个，去重）、`play_status`、`spoiler_level`、`short_summary`（可缺，缺 = 空串）、`aspect_scores`（可缺，缺的项 = 没打）。
`RatingPatch`：以上除 `work_id` 外全部可选；只改提交的字段。

### 3.4 错误码

| 码 | 状态 | 何时 |
|---|---|---|
| `NOT_FOUND` | 404 | 评分不存在、作者被封禁、作品已下架 |
| `VALIDATION_FAILED` | 422 | 字段越界；`work_id` 没有这部作品（`UNKNOWN_REFERENCE`） |
| `ALREADY_EXISTS` | 409 | 已经评过这部作品 |
| `PERMISSION_REQUIRED` | 403 | 编辑别人的评分；删除别人的且不是作品页创建者也没有 `rating.delete_any` |
| `SELF_LIKE_FORBIDDEN` | 403 | 赞自己的评分 |
| `ACCOUNT_BANNED` | 403 | 写面的会话用户已被封禁 |
| `CONTENT_REJECTED` | 422 | trust 拒绝短评 |
| `INVALID_PARAMETER` / `UNKNOWN_ENUM_VALUE` / `UNKNOWN_SORT` | 400 | 参数 |
| `SERVICE_UNAVAILABLE` | 503 | OAuth / catalog 不可用 |

无新码。`SELF_LIKE_FORBIDDEN` 的描述扩到「自己写的内容」。

## 4. 删除

6 条旧路由、`RatingHandler`、`RatingService` 中只为它们存在的方法与 DTO；`legacy_route_baseline` 减 6。网页：列表容器、详情页、发布/编辑表单、删除、点赞、sitemap 切到类型化客户端。作品页（G）仍吃旧 `GalgameRatingCardOnGalgamePage`，发布成功后本轨在边上把 `Rating` 翻译成它。

## 5. 变异题（实现之前提交）

| # | 变异 | 契约语义 |
|---|---|---|
| M1 | 列表 SQL 去掉 `id` 决胜（数据里 `overall` 大量并列） | D3 |
| M2 | 列表的 SFW 闸不生效（`include_nsfw=false` 也列 NSFW 作品的评分） | D2 |
| M3 | 列表的 `total` 不带内容闸（在丢行前数） | D2 |
| M4 | 详情不查作品是否下架 | D5 |
| M5 | 发布不查 catalog 作品是否存在 | D6 |
| M6 | 重复发布不回 409（或写进第二条） | D6 |
| M7 | 写面不判会话用户封禁 | D8 |
| M8 | `PUT like` 在已赞时再切换一次（重放翻转） | D9 |
| M9 | 自赞放行 | D9 |
| M10 | 点赞的萌萌点记给点赞者而不是作者 | 旧语义 |
| M11 | 作品页创建者不再能删别人的评分（或任何人都能删） | D10 |
| M12 | 编辑别人的评分放行 | D11 |
| M13 | 分项 0 发成 `0` 而不是 `null` | D13 |
| M14 | 删除不扣回发布奖励 | 旧语义 |

## 8. 实现中改的（理由在这里，代码在实现提交里）

| # | 改了什么 | 为什么 |
|---|---|---|
| I1 | `listRatings` 的档位是 `optional`，不是 §3.1 写的 `public` | `v1.Public` 根本不解析凭证，`viewer` 永远是 `null`；「带凭证时算 `viewer`」正是 `optional` 的定义 |
| I2 | D5 的「作品已下架」按 catalog 是否还显示判，不按本地 `published` | `published` 是粘性 SEO 标志（首个资源置位、删资源不清，迁移 078），它把「从没人发过资源」和「被隐藏」混在一起——前者的评分会被误判 404。列表同理在渲染时丢，`total` 照数，与封禁作者一致 |
| I3 | 请求里的 `aspect_scores` 与响应共用 `AspectScores`：八个键必须都在，`null` = 没打 | G8：同名属性只能有一个 schema。响应要「必带 + 可空」、请求要「可缺」，一个组件表达不了两种 `required`。§3.3 原写「缺的项 = 没打」，改严了；网页表单本来就总发八个 |
| I4 | 编辑、点赞、取消点赞先按读面的可见性判（作者封禁 / 作品不显示 → 404）；删除只判存在 | 写面不能碰到 `getRating` 说不存在的东西——初稿里作者改一条作品已隐藏的评分，写成功后回读却是 404。删除不设这道闸，作者本人、作品页创建者和 staff 仍能清理封禁作者的评分 |
| I5 | 点赞的萌萌点键 = `kungal:liked:galgame_rating_like_<赞行 id>` / `unliked:…`，确定性的；通知只在真的赞上时写 | 与话题 v1 的赞同一写法（旧面用每次随机的 nonce）。旧面取消点赞也会写一条 `liked` 消息 |
| I6 | 点赞者与作者同一条规则：封禁的丢、账号已删的给已注销用户引用 | 初稿把 OAuth 查不到的点赞者丢掉，「最近 ≤50 个」会悄悄变短 |
| I7 | `RatingRepository` 并进 `RatingStore`（最后一个活查询 `CountReviewsWithMinLength` 搬过去，创作者服务改用它）；旧 handler / service / mapper / DTO / 无人用的模型类型全删，`rawJSON` 挪进 `galgame_mapper.go` | 旧仓库删完只剩一个方法，留两个评分存储没有意义 |
| I8 | 旧 service 的 trust 拒绝 + 后台扫描测试搬进 v1 测试（断言扫描的 kind / subject_id / text） | 删 service 时这条覆盖不能跟着消失 |
| I9 | 网页评分详情的作品块：年龄限制与原始语言换成制作会社与发售日期；JSON-LD 的 `isFamilyFriendly` 取 `!is_nsfw`、`publisher` 取 `maker`、不再发 `inLanguage` | `WorkSummary` 两者都没有；`is_nsfw` 是显示轴，绝不能渲染成年龄轴 |
| I10 | 网页列表保留旧 URL 参数名（`sort_field` / `sort_order` / `spoiler_level` / `play_status` / `galgame_type`），不认识的值丢掉而不是发出去吃 400 | 分享出去的链接要继续能用 |
| I11 | 用户页（U）的评分卡片与作品页（G）仍吃旧形状，`ratingToCard` / `ratingToGalgamePageCard` 在边上翻译；点赞按钮（动态流也在用）只换内部实现，props 不变 | 这些组件归别的轨 |

另记两件没改的事：

- 「总分 1 或 10 时短评不少于 100 字」一直只在网页表单里校验，旧 DTO 从没在服务端拦过；v1 照旧。
- 发布表单写着「短评超过 20 字 5 萌萌点、超过 100 字 10 萌萌点」，服务端按**字节**数、阈值 233 / 666 发（`constants.RatingLenThreshold*`）。旧就不一致，按字还是按字节是产品决定，本轨不动。

## 9. 变异执行结果（14 条 + M11 拆两向，全部杀死）

| # | 变异 | 变红的测试 |
|---|---|---|
| M1 | 列表 SQL 去掉 `id` 决胜 | `TestV1RatingsListWalk`、`TestV1RatingsListFilters` |
| M2 | SFW 闸不生效 | `TestV1RatingsListWalk`、`TestV1RatingsListFilters` |
| M3 | `total` 在内容闸之前数 | `TestV1RatingsListWalk`、`TestV1RatingsListFilters` |
| M4 | 详情不查作品是否还显示 | `TestV1RatingsDetail`（隐藏作品的评分回 200） |
| M5 | 发布不查 catalog 作品是否存在 | `TestV1RatingsCreate` |
| M6 | 重复发布不回 409 | `TestV1RatingsCreate` |
| M7 | 写面不判会话用户封禁 | `TestV1RatingsCreate`、`TestV1RatingsUpdate`、`TestV1RatingsLike` |
| M8 | `PUT like` 已赞时再切换一次 | `TestV1RatingsLike` |
| M9 | 自赞放行 | `TestV1RatingsLike` |
| M10 | 点赞萌萌点记给点赞者 | `TestV1RatingsLike` |
| M11a | 作品页创建者不能删别人的评分 | `TestV1RatingsViewer`、`TestV1RatingsDelete` |
| M11b | 任何人都能删 | `TestV1RatingsViewer`、`TestV1RatingsDelete` |
| M12 | 编辑别人的评分放行 | `TestV1RatingsUpdate` |
| M13 | 分项 0 发成 `0` | `TestV1RatingsDetail`（及 spec 一致性：`minimum: 1`） |
| M14 | 删除不扣回发布奖励 | `TestV1RatingsDelete` |
