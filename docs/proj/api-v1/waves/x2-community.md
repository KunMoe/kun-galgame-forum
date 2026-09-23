# X2-community · 评论墙的关注与已读

> 2026-09-23 立。X 轨后半（X2）的一段，分支 `api-v1/x2-community`，迁移号 209（**本段不用**）。
> 契约与变异题同一个提交，早于实现。

## 1. 普查

### 1.1 路由与调用方

三条旧路由由 `CommunityEngagementHandler`（`internal/community/handler/engagement_handler.go` → `internal/community/engagement`）独占。RC 轨把墙的读写搬上 v1 时把它们留在了 X（RC 契约 D20）。

| # | 旧 | 调用方 |
|---|---|---|
| 1 | `POST /api/community/wall/read` | `composables/useCommunityCommentList.ts`（六面墙打开时发一次，裸 `$fetch`，吞掉一切错误） |
| 2 | `POST /api/community/wall/follow` | 同上的关注开关；`pages/message/follow.vue` 的「取消关注」 |
| 3 | `GET /api/community/following` | `pages/message/follow.vue`（「关注的评论区」） |

`apps/web/server/` 零调用；`../kungal-apps` 零调用。

旧面用上游 community 的**锚点**（`anchor_kind` 整数 + `anchor_id`，如 `resource:51434`）寻址墙；网页为此在 `utils/communityComment.ts` 里保留了一份「主体 → 锚点」的映射，注释写明「只为调旧路由而存在」。

### 1.2 生产取值（`kun_community`，2026-09-23）

- `community_anchor_user`（锚点层订阅，「关注的评论区」列表的数据源）：kungal **2 行**，都是 watching（一堵 galgame 墙、一堵资源墙），各 1 人。
- `community_thread_user`（线程层）：kungal watching 165 行（123 人）、normal 3 行。线程层的 watching 多数是发帖/被回复时上游自动建的。
- 墙的通知镜像：本地 `message` 表以 `community_thread_id` / `community_post_number` 记上游线程通知，已读回执要同步它们（`MarkCommunityThreadRead`）。

### 1.3 语义与疑似 bug

1. **关注 = 锚点层与线程层都写 watching（3）；取消 = 都写 normal（1），绝不写 muted（0）**（muted 会连回复与 @ 一起静音；记忆 `kungal-community-notification-mirror`「取关写 1 绝不写 0」）。
2. **已读回执只在「已有线程层状态」或「关注着锚点」时才真的标已读**——对一堵既没发过帖也没关注的墙，不能凭空建状态行。回执同时把本地镜像的通知行标已读。
3. 旧的「墙存在」判定只看锚点格式（网站墙查一次库），**不查主体是否真的存在**：给不存在的资源 id 也能关注成功，并在上游留下一行订阅。题目墙的答题门（RC 的 `QUIZ_ANSWER_REQUIRED`）在这里完全不存在。
4. 上游失败时旧已读回执**回一个「未关注」的假状态**（`idle`），网页的开关因此静默翻回「未关注」。
5. 列表项下发**服务端拼好的网页路由**（`/galgame-rating/…`）与**中文标签**（「游戏评分」），违反「服务端不产出给人看的句子」（01 §7、F8）。标题只对 galgame 墙有（catalog 名字），其余墙只有标签。
6. 列表的游标是上游的原样字符串，不绑定条件；`limit` 1–50。

## 2. 范围

| 旧 | v1 |
|---|---|
| `GET /api/community/following` | `GET /api/v1/me/walls` — 我关注的墙，游标集合 |
| — | `GET /api/v1/me/walls/{subject_type}/{subject_id}` — 我在这堵墙上的状态（G17 要求的读面） |
| `POST /api/community/wall/follow` | `PUT` / `DELETE /api/v1/me/walls/{subject_type}/{subject_id}/follow`（K16 槽位） |
| `POST /api/community/wall/read` | `PUT /api/v1/me/walls/{subject_type}/{subject_id}/read-marker`（M 轨同名先例） |

删 3 条旧路由，连同 `CommunityEngagementHandler`、`internal/community/engagement`；`legacy_route_baseline` 以合并时重新生成为准。

实现放在墙的域 `internal/wall/apiv1`：墙的主体词表（`subject_type`）、存在性与可见性判定（`resolveSubject`，含题目墙的答题门）、上游游标的包装都在那里，不另起一份映射。

## 3. 形状

### 3.1 `WallState`（`object: "wall_state"`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | DecimalID | 墙主体的 id，与 `subject_id` 同值（G17：被 `PUT` 寻址的路径要能 GET 到带 `id` 的对象，同 T4 `admin_topic`） |
| `subject_type` | 与 `WallComment.subject_type` 同一封闭词表 | |
| `subject_id` | DecimalID | |
| `is_following` | boolean | 线程层或锚点层是 watching |

单墙 GET、`follow` 的 `PUT`/`DELETE`、`read-marker` 的 `PUT` 都回它。

### 3.2 `FollowedWall`（`object: "followed_wall"`，`GET /me/walls` 的条目）

| 字段 | 类型 | 说明 |
|---|---|---|
| `subject_type` | 同上 | |
| `subject_id` | DecimalID | |
| `work` | `WorkRef` \| null | galgame 墙才有（共享类型 #199）；其余墙为 `null`——旧面对它们也只有一个标签 |

- 不下发网页路由、不下发标签：客户端按 `subject_type` 自己拼（§1.3 第 5 条）。
- 只列 watching 的锚点订阅；上游返回的 normal 行（取消关注留下的）与本论坛认不出的锚点被跳过，所以一页可能比 `limit` 短，只以 `next_cursor` 是否缺席判断末页。
- 游标：`cur_` 包装上游游标并绑定调用者；`limit` 1–50，默认 20（上游上限 50）。

### 3.3 寻址

`{subject_type}` 是 `WallComment.subject_type` 的封闭词表（`galgame` `galgame_rating` `galgame_resource` `galgame_quiz` `toolset` `website`），`{subject_id}` 是十进制正整数。锚点在服务端内部换算，客户端永不再见到 `anchor_kind` / `anchor_id`。

## 4. 逐条裁决

| 旧面 | 裁决 |
|---|---|
| 不查主体存在 | 复用 RC 的 `resolveSubject`：不存在 → `404 NOT_FOUND`；题目墙未答题 → `403 QUIZ_ANSWER_REQUIRED`；主体作者被封禁 → `404` |
| 上游失败回假状态 | `503 SERVICE_UNAVAILABLE`（上游未配置同样 503）。网页的回执调用仍然吞错——那是网页的决定，不是 API 的 |
| 取消关注 | 锚点层与线程层都写 normal（1），永不写 muted（0） |
| 回执对无关的墙建状态 | 保留旧语义：无线程状态且未关注锚点时不调用上游的标已读，只回状态 |
| 回执同步本地镜像 | 保留 |
| 服务端拼路由与中文标签 | 删除，改 `subject_type` + `work` |
| Bearer | 允许（关注与已读不是 staff 能力） |

错误码：**无新增**。用到 `NOT_FOUND` `QUIZ_ANSWER_REQUIRED` `INVALID_CURSOR` `LIMIT_TOO_LARGE` `UNKNOWN_ENUM_VALUE` `INVALID_PARAMETER` `SERVICE_UNAVAILABLE` 与鉴权档的码。

## 5. 网页

- `useCommunityCommentList.ts`：回执改 `PUT …/read-marker`（仍然静默吞错），关注开关改 `PUT` / `DELETE …/follow`。
- `pages/message/follow.vue`：`GET /me/walls` + `DELETE …/follow`；链接与标签由 `subject_type` 在客户端拼；galgame 墙的标题用 `work` 的名字（渲染规则 `localized[locale] ?? display_name ?? latin`）。
- `utils/communityComment.ts` 的 `wallAnchor` 映射删除；`shared/types/galgame-community-comment.ts` 里这三条旧路由的类型删除。
- `legacy-fetch-baseline` 按删掉的调用点下调。

## 6. 顺带（协调者 37 的通知）

TS 波次文档补一行：生产 64 条停在 `received` 的举报已由 infra 回填成 58 个待处理条目，`kungal` 的站点策略把聚合阈值调到 0.5（单条举报即开条目）。

## 7. 变异题（先于实现提交）

测试用 RC 的内存假 community，加上锚点/线程订阅、标已读与订阅列表四个端点，并记录每次写入的等级。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 取消关注写 muted（0） | 假上游收到的锚点层与线程层等级都是 1 |
| 2 | 关注时不写线程层（墙已有线程） | 线程层等级被写成 3 |
| 3 | 回执对无关的墙也标已读 | 既没线程状态也没关注的墙：`PUT read-marker` 200 且假上游一次标已读都没收到 |
| 4 | 回执不同步本地镜像 | 关注着的墙回执之后，本地 `message` 里该线程的通知行变为已读 |
| 5 | 不查主体存在 | 不存在的网站 id → `404 NOT_FOUND`，假上游没收到任何写 |
| 6 | 跳过题目墙的答题门 | 未答题的剧透题目墙 → `403 QUIZ_ANSWER_REQUIRED` |
| 7 | 列表不过滤 normal 行 | 取消关注留下的 normal 订阅不出现在 `GET /me/walls` |
| 8 | 列表游标不绑定调用者 | A 的游标给 B 用 → `400 INVALID_CURSOR` |
| 9 | 上游失败回假状态 | 假上游 502 → `503 SERVICE_UNAVAILABLE`（单墙 GET 与回执都是） |
| 10 | galgame 墙的 `work` 不填 | 列表里 galgame 墙的 `work.id` 等于主体 id、`work.display_name` 是 catalog 的名字 |
