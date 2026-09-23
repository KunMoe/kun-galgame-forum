# 普查 · galgame 社区评论（CommunityCommentHandler，7 条）

> 2026-09-22 由 G 轨（`api-v1/g-galgame`）的普查一并做出，随后按用户裁决这 7 条划归 **RC 轨**，拆出本文件交给 RC 当输入。G 轨不再迁移、不删这组 handler 与服务。
> 只读普查，基于 `ef0c9fb3`，全部结论来自代码，**尚未裁决、尚未经督查逐条复核**。行号与编号沿用原 G4 普查。

| # | 方法 | 路径 | handler | 档位 |
|---|---|---|---|---|
| 14 | GET | `/api/galgame/:gid/comments` | `galgame/handler/community_comment_handler.go:37` `List` | optional |
| 15 | GET | `/api/galgame/:gid/comments/locate` | `community_comment_handler.go:154` `Locate` | optional |
| 16 | POST | `/api/galgame/:gid/comments` | `community_comment_handler.go:56` `Create` | required |
| 17 | PUT | `/api/galgame/comments/:postId` | `community_comment_handler.go:79` `Update` | required |
| 18 | DELETE | `/api/galgame/comments/:postId` | `community_comment_handler.go:101` `Delete` | required |
| 19 | PUT | `/api/galgame/comments/:postId/like` | `community_comment_handler.go:116` `ToggleLike` | required |
| 20 | POST | `/api/galgame/comments/:postId/flag` | `community_comment_handler.go:132` `Flag` | required |

## 5. galgame 社区评论（BFF，锚点 `site_game`）

论坛不存评论文本。上游是 infra community：`AnchorKind=1`（`communityclient.AnchorSiteGame`），`AnchorID=strconv.Itoa(galgameID)`（论坛 gid，`community_comment_service.go:74,38`）。本地表只有：

- `galgame_post_like`（展示用点赞计数 / 「我赞过」，迁移 057）
- `galgame_comment_community_map`（旧 `galgame_comment.id` → `(thread_id, post_id, galgame_id)`，给 locate / 深链）
- `galgame.comment_count`（best-effort 计数器）

`GalgameCommentEnforcer`（`comment_enforce.go`）是 trust 处置适配器，不是这些 HTTP 路由。

### 5.1 `GET /api/galgame/:gid/comments`（#14）

**调用方**：`useCommunityCommentList.ts:27-33` 经 `communityComment.ts:41,51` 的 `listUrl = /galgame/${galgameId}/comments`。挂载：`components/galgame/comment/CommunityContainer.vue` ← `Galgame.vue:179`。`limit=30`（`useCommunityCommentList.ts:6,32`）。`lazy: true` 的 SSR `useKunFetch`。Nitro / Flutter 零。

**请求**：路径 `gid` `parsePositive`（`community_comment_handler.go:175`，必须 `>0` 整数），非法 → `400/233 "非法的 galgame ID"`。query `cursor` 无校验、`limit` `omitempty,min=1,max=50`。`limit` 缺席/0 过校验，服务层 `clampReadLimit` 夹到 50（`community_comment_service.go:160`，未知值静默夹，01 §4 要 `400 LIMIT_TOO_LARGE`）。`cursor` 原样当 community 的 `after`（`ops.go:13-17`）。

**响应**（`CommunityCommentPage`，`community_comment_service.go:63`）：`thread_id, posts[], next_cursor, total, locked, anchor_kind, anchor_id`。`locked` 在本路径**恒 false**——`lockedCommentPage`（`:193`）已写、从未从 `GetComments` 调用。题目讨论墙上「作答后才开放」的 lock 是 ResourceCommentHandler 的事。`total` = `page.Thread.PostsCount`（上游线程计数），`posts` 是 `renderPosts` 滤过的。过滤掉封禁作者 / held-对-外人 之后，**`total` 与 `posts.length` 谓词不同**。`next_cursor` 原样透传；空字符串表示没有下一页（前端 `hasMore = nextCursor !== ''`，`useCommunityCommentList.ts:109`）。游标格式由上游决定，本仓看不到。无 `id` tie-breaker 可查（排序在 community）。

`CommunityPostItem`（`:41`）：`id, thread_id, content, content_html, galgame_id, user{id,name,avatar}, parent_comment_id, root_comment_id, target_user?, like_count, is_liked, created, edited, edited_by_moderator, status, deleted, held`。

- `content` / `content_html`：`markdown.NormalizeStoredContent` 存的是 Markdown 源；下发 `markdown.Render`（完整 Markdown 管线，`community_comment_service.go:140`）。网页 `Row.vue:163-167` `KunContent` + `renderKatex(content_html)`。这与话题评论 K20（纯文本受限文档、不跑 Markdown）**相反**。composer 是 `KunMilkdownDualEditorProvider`（`Composer.vue:58`）。`maxLength` 5000（`communityComment.ts:44`，handler `max=5000`）。
- `like_count` / `is_liked` 来自**本地** `galgame_post_like`（`CountLikes` / `LikedSet`，`community_post_repo.go:30,50`），不是 community 的 reaction 计数。`Scan` 丢错 → 全 0。
- `status` 裸 int32：0 visible / 1 held / 2 deleted（`communityclient/types.go:26-28`）。
- 删除行：`visibleTo` 对 deleted 返回 true（`community_comment_service.go:123-128`），正文清空，`deleted=true`。held 只给作者看。
- `galgame_id` 是路径上那个 gid，不是上游存的。锚点错了也会被写成路径值。

**错误**：community 未配置 / 网络错 / `ErrForbidden`（client 未绑定 site）→ **200 空页**（`isCommunityDown`，`:205-211,77-78`）。fail-open 成「没人评论」。`APIError` 4xx → `400` + **上游的数字 `Code` 与中文 `Msg`**（`mapCommunityError`，`:222-224`）。429 → `429/233 "发表过于频繁…"`。其它 → `503/233 "评论服务暂不可用"`。

**鉴权**：optional。`optionalUID` 只影响 held 可见性与 `is_liked`。

**可见性**：只做 community 的 held 过滤 + 作者 `IsRenderable`。**不查** `galgame.published`、catalog hidden、NSFW、本地行是否存在。对不存在的 gid 仍打上游 `anchor_id="<gid>"`，没有线程就空页。`target_user` **不跑 `IsRenderable`**（`:113-117`），封禁被回复者照样下发名字。作者过滤与 Hydrate fail-open 同 §1.1。

**偏好**：本 handler 不调 `IsSFW`。`NamePreference` 在链上但本面没有作品名。创建时 `ContentRating: RatingAll` 写死（见 5.3），读面也不按读者立场滤评论。

**测试**：`handler/community_comment_routes_test.go:32,45` 只断言路由挂上、读面匿名不被 401。`service/community_comment_test.go` 覆盖 `visibleTo` / `buildCommunityItem` / `likeEffects` / `prepareMentionIDs` / `clampReadLimit`，不打上游。

---

### 5.2 `GET /api/galgame/:gid/comments/locate`（#15）

**调用方**：`CommunityContainer.vue:37-46`，深链 `?comment=<legacy_id>` 且带 `thread`、或 hash `#galgame-comment-<id>` 在当前列表找不到时。用 raw `$fetch`（不要 kunFetch toast），`credentials: 'include'`。Nitro / Flutter 零。

**路径 `:gid` 完全不读。** 只 parse query `legacy_id` `required,min=1`（`community_comment_handler.go:154-160`）。`Locate`（`community_comment_write.go:291`）`FindMapByLegacyID`（`community_post_repo.go:68`）。命中回 `{post_id, thread_id}`；没有行 404 `"未找到对应评论"`；SQL 错 500 `"查询失败"`。**不校验 map.galgame_id 是否等于路径 gid**——用游戏 A 的 URL 可以解析到游戏 B 的旧评论。map 表不回填新评论（057 原文：import tool 在 cutover 填；之后的 community 帖没有 old_comment_id）。

`CreateGalgameCommentMention` 的链接是 `/galgame/%d?comment=%d` 且**不加 `thread`**（`interaction.go:79-81` 的注释说「`thread` 让 CommunityContainer 把 comment 当 legacy id」）。所以新 @ 通知走 `commentParam` 且 `hasThreadParam` 为假 → `ensureLoadedAndScroll(commentParam)` 把 community post id 当现 id 滚，**不走 locate**（`CommunityContainer.vue:69-78`）。locate 是给带 `thread` 的旧深链和 hash 回退用的。

**测试**：无 locate 单测。

---

### 5.3 `POST /api/galgame/:gid/comments`（#16）

**调用方**：`Composer.vue:40-44`，`surface.listUrl` + `createBody` = `{content, reply_to_post_id}`（`communityComment.ts:57-60`）。Nitro / Flutter 零。

**请求**：路径 gid；body `content` `required,min=1,max=5000`，`reply_to_post_id` `*int64` 可选。无 `Idempotency-Key`。无 `parent` 与 `topic_id`——父子关系靠 community 的 `ReplyToPostID`。服务端 `NormalizeStoredContent`，从正文抽 `@` token（`markdown.ExtractMentionIDs`）经 `mentionIDsForCreate`：去重、丢掉自己、OAuth 成功则丢掉未知 id，失败则**保留**（`prepareMentionIDs`，`community_comment_write.go:233-250`，测试 `:125`「lookup error keeps ids」）；超过 20 → `400/233 "一条评论最多 @ 20 位用户"`。

打 community `CommentOnAnchor`：`AnchorSiteGame`、`AnchorID="<gid>"`、`ContentRating: RatingAll` 写死（`:37-40`）、`MentionUserIDs`。成功后 `afterCreate`：尝试 `INSERT galgame(id) ON CONFLICT DO NOTHING` 再 `comment_count+1`（`:60-73`）。对不存在的 gid，这会试图插一条只有主键的 `galgame` 行（`published` 有 DEFAULT false，迁移 068）；若其它 NOT NULL 无默认，INSERT 失败，只 `slog.Warn`，评论已在上游。然后 `feedParityUpsert` `GALGAME_COMMENT_CREATION`（post id 超过 int32 则跳过，`feed_parity.go:13-16`）。

**不跑论坛 trust check。** 审核是 community 的 hold（`status=1`，前端 `result.held` 弹「已提交，待审核」，`Composer.vue:49`）。与话题评论 W5a 的 `gate.Decision` 不同。

响应一条 `CommunityPostItem`。`renderPosts` 若因 IsRenderable 滤掉（刚发的自己不太可能），fallback `buildCommunityItem` + Hydrate 自己（`:53-56`）。

不查 galgame 存在 / published / hidden / NSFW。`reply_to_post_id` 是否属于本锚点由上游判，失败走 `mapCommunityError`。

**测试**：无创建路径测试。

---

### 5.4 `PUT /api/galgame/comments/:postId`（#17）——跨族编辑面

**调用方**：所有社区评论族的编辑。`Row.vue:86-92` `surface.editUrl(postId)` + `editQuery`。galgame 族带 `?gid=`（`communityComment.ts:53-54`）；**quiz / rating / website / toolset / resource 的 `editQuery` 是 `{}`**（`:76,99,122,145,168`）。Nitro / Flutter 零。

**请求**：路径 `postId` `parsePositive64`；body `{content}` 同 5000 上限。`optionalGid` 只从 **query `gid`** 读（`community_comment_handler.go:168`），不是路径。

`UpdateComment`（`community_comment_write.go:78`）：`resolveModEdit` 先 `ResolvePosts` 看锚点，按锚点选 `comment.*.edit`（`:141-159`，`site_game` → `CommentGalgameEdit`；`site_resource` 再看 `rating:` / `website:` / `quiz:` 等前缀）。`user.Can`（Bearer 恒 false）。**`ResolvePosts` 失败时 fallback 是把六个 `comment.*.edit` 权限 OR 在一起**（`:117-122,138`）——community 一抖，持有 `comment.quiz.edit` 的版主可以 `AsModerator` 编辑一条 galgame 评论。作者编辑走 `AuthorID` + `AsModerator:false`，由上游认主。

`gid` 有值且 resolve 成功才 `refanMentions`（新 @ 的本地 `mentioned` 通知）。创建路径**不**写本地 mention 行，只把 id 交给 community。编辑 galgame 评论时前端带 gid；编辑 quiz 评论时不带，新 @ 没有论坛站内信。

响应 `CommunityPostItem`，`galgame_id` 在没传 gid 时是 0。`buildCommunityItem` 不跑 IsRenderable。不查所属墙的可见性。trust：论坛侧不跑；K18 不适用。

错误：非法 postId 400 `"非法的评论 ID"`；上游 4xx/403/429/503 同 mapCommunityError。

**测试**：无。

---

### 5.5 `DELETE /api/galgame/comments/:postId`（#18）

**调用方**：仅 galgame 族。`communityComment.ts:55-56` `deleteUrl=/galgame/comments/${postId}` + `deleteQuery:{gid}`。其它族走 ResourceCommentHandler 的前缀删除（本段外）。`Row.vue:111-114`。

handler 把 `user.Can(perm.CommentGalgameDelete)` 一个布尔传进服务（`community_comment_handler.go:110`）。**不 resolve 锚点**。持 `comment.galgame.delete` 的人可以对**任意** community post id 打 `DeletePost(..., asModerator=true)`，包括 quiz/rating 的帖。作者自删靠上游 `author_id`。Bearer 的 `Can` 恒 false，只能删自己的。

`gid` 有值才 `comment_count-1`（`GREATEST(...,0)`）。没有 gid 计数不减。然后 `feedParityDelete` + 按 map 删遗留 feed 行。

成功 `OKMessage "评论已删除"`。前端本地把该行标成 tombstone（`useCommunityCommentList.ts:182-187`），与上游「删后留占位」一致。

不查可见性、不查锚点种类、不扣萌萌点（对比 W5a 删除扣分）。无幂等问题（重复删由上游判）。

**测试**：无删除单测。

---

### 5.6 `PUT /api/galgame/comments/:postId/like`（#19）——跨族切换赞

**调用方**：`components/comment/community/Like.vue:29-31`，**所有**社区评论族共用。前端禁止自赞（`:22-26`，消息 10533）；**后端允许**（`likeEffects` 自赞 `localPresent=true, awardDelta=0`，`community_comment_write.go:210-221`，测试 `:91`）。

community `ToggleReaction` like。成功后本地 `EnsureLike` / `RemoveLike`（失败只 Warn）。非自赞则 `moemoepoint.Award(±1, liked, ref=galgame_post:<id>, KeyNonce)`。响应 `{liked, like_count}`，`like_count` 再读本地表——community 已切换、本地插入失败时，`liked=true` 而 `like_count` 仍是旧值。

PUT 切换，非幂等。不查 post 属于哪面墙、作者是否可渲染、帖是否 deleted/held。自赞后端放行。

**测试**：`TestLikeEffects` 单位。无 HTTP。

---

### 5.7 `POST /api/galgame/comments/:postId/flag`（#20）——跨族举报

**调用方**：`FlagModal.vue:38`，所有社区评论族共用。理由 0–4 与 `communityclient.FlagReason*` 对齐（`types.go:39-43`）：垃圾 / 辱骂 / 离题 / 其他 / 分级标注错误。前端 `REASON_OPTIONS`（`FlagModal.vue:4-10`）同词表。

handler：`Reason int validate:"min=0,max=4"`（`community_comment_handler.go:141-144`）。**缺席的 JSON 字段是 Go 零值 0**，`min=0` 放过 → 当成「垃圾信息」。服务层再挡一次 `< FlagReasonSpam || > FlagReasonNsfwMislabel`（`community_comment_write.go:280`），0 仍合法。`note` `max=500`。

打 community `SubmitFlag`。成功 `OKMessage "举报已提交"`。不查 post 存在（上游 4xx）。无幂等键。无测试。

---

## 6. 与话题评论 v1（W5a）的差异

对照 `docs/proj/api-v1/waves/w5a-comments.md` 与 `internal/topic/apiv1/{write_comment,comment_ops,comment_content}.go`。本段是 BFF，话题评论是本地表。

| 点 | 话题评论 v1 | 本段现状 |
|---|---|---|
| 存储 | 本地 `topic_comment` | infra community 原语，锚点 `site_game` + 论坛 gid |
| 正文 | K20：`content` 受限文档，源字段 `text`，不跑 Markdown | Markdown 源 + 服务端 HTML（`markdown.Render`），网页 `KunContent` |
| 上限 | 1007（列宽） | 5000 |
| 树 | 平铺 + `parent_comment_id` | `parent_comment_id` / `root_comment_id` 来自 community；前端 `useCommunityCommentList` 自己拼两层 |
| 可见性 | `visibleReply`：话题隐藏/ACL/作者可渲染 | 不查作品；只滤 held 与作者 Status |
| 写前可见性 | K17：OAuth 不可用 503 | 不查；OAuth 错时 @ 名单 fail-open 保留 |
| trust | 论坛 `gate.Check`，K18 正文未变不跑 | 论坛不跑；community hold |
| 幂等 | POST 必填 `Idempotency-Key` | 无 |
| 赞 | `PUT`/`DELETE` 槽位，`SELF_LIKE_FORBIDDEN` | PUT 切换；后端允许自赞 |
| 删除 | 204；扣分不阻断 | 200 中文消息；不扣分 |
| 点赞计数 | 评论行上聚合 | 本地 `galgame_post_like` 镜像，可与上游脱节 |
| locate | 回复轨，`?comment=` | 本段 `legacy_id` → map 表；路径 gid 不校验 |
| 编辑权限 | `comment.topic.edit` | 按锚点选 `comment.galgame.edit` 等；resolve 失败 OR 六个权限 |
| `viewer` | `has_liked` / `can_edit` / `can_delete` / `can_like` | 没有 viewer；`is_liked` 顶层；能力靠前端 `useCan` |
| id | 字符串 | JSON number（community int64） |
| 创建 HTTP | 201 + Location | 200 + 旧信封 |

ResourceCommentHandler 的题目/资源/评分/网站/工具集评论与本段共用 Update/Like/Flag 三条，列表/创建/删除不共用。v1 资源图要决定这三条是「所有 community 帖的一个资源」还是继续按墙切开。

---

## 8. 疑似 bug（原 G4 §8 的 31–49 条）

31. 评论读面 community 宕机 / `ErrForbidden` / 非 APIError → 200 空列表（`community_comment_service.go:77-78,205`）。
32. `mapCommunityError` 把上游数字 code 与中文消息当论坛信封回吐（`:222-224`）。
33. `Locate` 忽略路径 gid、不校验 `galgame_id`（`community_comment_handler.go:154`，`community_comment_write.go:291`）。
34. `afterCreate` 对任意正整数 gid `INSERT galgame`（`community_comment_write.go:62-64`）。不存在的作品被评论会试图种本地行；失败只 Warn，上游评论已在。
35. 创建评论不查作品存在/published/hidden/NSFW（`CreateComment`，`:31`）。
36. 删除/点赞/举报不 resolve 锚点种类（`DeleteComment` `:162`，`ToggleLike` `:181`，`Flag` `:279`）。`comment.galgame.delete` 可对 quiz 帖 asModerator 删除。
37. `resolveModEdit` 在 `ResolvePosts` 失败时 OR 六个 edit 权限（`:117-122`）。
38. `PUT …/like` 切换；本地计数与 community 权威切换不是一个事务（`:181-207`）。
39. 后端允许自赞（`likeEffects` `:210`）；前端挡（`Like.vue:22`）。
40. 举报 `reason` 缺省为 0（`community_comment_handler.go:141-144` `min=0`）。
41. 评论创建无幂等键。
42. `target_user` 不跑 `IsRenderable`（`community_comment_service.go:113-117`）。
43. `prepareMentionIDs` 在 Users() 出错时保留未知 id（`community_comment_write.go:243-250`）。
44. 创建路径不写本地 `mentioned` 消息，只交给 community；编辑才 `refanMentions`，且依赖 query `gid`（`:89-94,103`）。
45. `feedParityUpsert` 在 post id > MaxInt32 时静默跳过（`feed_parity.go:13-16`）。
46. 评论 `content_html` 是服务端 Markdown HTML（01 §6 / K13 要结构化节点）。`markdown.Render` 丢错（`markdown.go:163` `html, _ :=`）。
47. `ContentRating: RatingAll` 写死（`community_comment_write.go:39`），与作品 NSFW 无关。
48. `locked` 字段在 galgame 评论页恒 false，前端类型里却有（`community_comment_service.go:87-90` vs `galgame-community-comment.ts:26`）。
49. `clampReadLimit` 把非法 limit 夹成 50（`community_comment_service.go:160`）。

## 9. 生产库 SQL（评论部分）

```sql
-- 评论 BFF
SELECT count(*) FROM galgame_comment_community_map;
SELECT count(*) FROM galgame_comment_community_map m
  WHERE NOT EXISTS (SELECT 1 FROM galgame g WHERE g.id = m.galgame_id);
SELECT count(*) FROM galgame_post_like;
SELECT count(*) FROM galgame_post_like l
  WHERE NOT EXISTS (
    SELECT 1 FROM galgame_comment_community_map m WHERE m.post_id = l.post_id
  );
-- 本地 comment_count 与「至少有过评论」的漂移只能与 community 对账；论坛侧先看
SELECT count(*) FROM galgame WHERE comment_count > 0;
SELECT count(*) FROM galgame WHERE comment_count < 0;
SELECT sum(comment_count) FROM galgame;

```

## 10. 提议的 v1 资源图（评论部分，只是提议）

| 旧 | 提议 v1 | 理由 |
|---|---|---|
| `GET /api/galgame/:gid/comments` | `GET /api/v1/galgames/{galgame_id}/comments` 游标 | 墙是作品的子资源 |
| `GET …/comments/locate` | `GET /api/v1/galgame-comments/legacy/{legacy_id}` | 查的是 map 表，不是墙上的偏移；校验所属 `galgame_id` |
| `POST /api/galgame/:gid/comments` | `POST /api/v1/galgames/{galgame_id}/comments` | 201；必填幂等键 |
| `PUT /api/galgame/comments/:postId` | `PATCH /api/v1/galgame-comments/{comment_id}` | 有自己 id 的资源；跨族共用面要在 RC 轨对齐，或升成 community-posts 共用资源 |
| `DELETE /api/galgame/comments/:postId` | `DELETE /api/v1/galgame-comments/{comment_id}` | 204；先 resolve 锚点再授权 |
| `PUT …/like` | `PUT`/`DELETE /api/v1/galgame-comments/{comment_id}/like` | K16 |
| `POST …/flag` | `POST /api/v1/galgame-comments/{comment_id}/flags` | 举报是新建资源；`reason` 封闭字符串枚举，缺席不得变 spam |
