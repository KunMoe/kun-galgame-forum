# T4 · 话题管理面

> 话题轨第四段（最后一段），2026-09-23。三条旧路由，全部在 `internal/admin/**`，由 `TopicAdminHandler` 独占（普查确认没有别的域共用）。
> 契约与变异题同一个提交，早于实现。

## 1. 生产取值（2026-09-23）

- 话题 3564 条，隐藏 322 条：`author` 307、`moderator` 13、`trust` 2。
- 带托管的开放抽奖：0（T3 刚上线）。

## 2. 范围

| 旧 | v1 |
|---|---|
| `GET /api/admin/topic/hidden` | `GET /api/v1/admin/hidden-topics`，**页码集合**（K11 第一个页码集合） |
| `GET /api/admin/topic/:tid/purge-stats` | `GET /api/v1/admin/topics/{topic_id}/purge-preview` |
| `DELETE /api/admin/topic/:tid` | `DELETE /api/v1/admin/topics/{topic_id}`，**204** |

三条都删；`legacy_route_baseline` 下调 3。话题轨就此清零。

权限不变：列表要 `topic.view_hidden`，另两个要 `topic.delete_any`，一律 `user.Can`——Bearer 请求永远不持有这两个能力。匿名 → 401，缺权限 → `403 PERMISSION_REQUIRED`。

## 3. 形状

### 3.1 页码集合原语（新增，全轨共用）

`internal/apiv1` 里加两个**新文件**，不改任何已有文件：

- `collect.PageNumber`：`page`（≥1，默认 1）、`limit`（1–100，默认 20）；`collect.CheckDepth(page, limit)`：`page × limit > 10000` → `400 INVALID_PARAMETER`，`errors: [{parameter: "page", reason: "OUT_OF_RANGE", params: {maximum: <最大页>}}]`。
- `repr.PageList[T]`：`{object: "list", items, total, total_relation}`，`total_relation` ∈ `eq` \| `gte`。不发 `next_cursor`、不发 `page_count`。

逐字照 01 §4。G 轨的 `/galgame` 浏览会复用它（已口头通知）。

### 3.2 `HiddenTopicSummary`（`object: "topic"`）

`id`、`title`、`state`、`hidden_by`、`author`、`reply_count`、`bumped_at`、`created_at`。**全部是 `Topic` 已有的字段，同名同型**（K10：同一个 object 可以有多个摘要形状，重叠字段必须一致）。

查询参数：`page`、`limit`（本集合默认 30）、`hidden_by`（封闭枚举，缺席 = 不过滤）、`q`（标题关键词，≤100 字符，`ILIKE` 且转义通配符）。排序固定 `bumped_at DESC, id DESC`，没有 `sort` 参数。

### 3.3 `TopicPurgePreview`

`object: "topic_purge_preview"`、`topic_id`、`title`、`state`、`hidden_by`、`author`、`reply_count`、`comment_count`、`poll_count`、`lottery_count`、`drawn_lottery_count`、`favorite_count`、`refunded_point_escrow`（删除时会退还给各抽奖发起人的萌萌点总数）。

计数字段以 `_count` 结尾（01 §命名表）；旧的 `replies` / `polls` 等裸名与 `status` 整数不再下发。

## 4. 逐条裁决

| 旧面 | 裁决 |
|---|---|
| 列表 `ORDER BY status_update_time DESC` 无决胜键 | `bumped_at DESC, id DESC` |
| 列表 offset 无上限 | 深度上限 10000（§3.1） |
| 列表的 `hidden_by` 非法值回 `233` 中文句子 | `400 UNKNOWN_ENUM_VALUE`（schema 枚举） |
| 关键词超长回 `233` | schema `maxLength: 100`，`400 INVALID_PARAMETER` |
| 删除回完整统计 | **204**。网页的提示语用预览里已有的标题 |
| **硬删会级联删掉开放抽奖，发起人托管的萌萌点随行消失**（T3 引入托管之后才成立的洞） | 删除事务里先锁住该话题所有抽奖：有 `drawing` 的 → `409 LOTTERY_DRAWN`（开奖正在写中奖名单）；开放抽奖的 `point_escrow` 在同一事务回写发起人缓存余额，提交后按每个抽奖推 OAuth 退款（幂等键同 T3 的 `lottery_escrow_refund`） |
| 删除不看话题状态 | 保留：持 `topic.delete_any` 的人可以硬删任何话题（与旧面相同） |

## 5. 预分配

- 迁移：**无**。
- 错误码：**无新增**。复用 `PERMISSION_REQUIRED`、`NOT_FOUND`、`LOTTERY_DRAWN`、`INVALID_PARAMETER`、`UNKNOWN_ENUM_VALUE`。

## 6. 网页

`pages/admin/topic.vue`：三个调用换成类型化客户端；列表走页码；手写的 `HiddenTopic` / `TopicPurgeStats` 接口删除；删除后的提示用预览里的标题；`status_update_time` → `bumped_at`、`user` → `author`、`replies` 等 → `*_count`。

## 7. 变异题（先于实现提交）

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 列表不查 `topic.view_hidden` | 普通用户 → 403 |
| 2 | 列表把已发布话题也算进来（去掉 `status = 1`） | 结果只含隐藏话题 |
| 3 | 排序去掉 `id` 决胜键 | 翻完所有页（**种子里有同一 `bumped_at` 的隐藏话题跨页**）无重复无遗漏 |
| 4 | 深度上限不查 | `page=101&limit=100` → 400 `OUT_OF_RANGE` |
| 5 | `total` 用了与 `items` 不同的谓词（忽略 `hidden_by`） | 过滤后 `total` 等于过滤后的行数 |
| 6 | 删除不退托管 | 带托管的开放抽奖所在话题删除后，发起人缓存余额回升、记录一笔退款 |
| 7 | 删除不拦 `drawing` | 有 `drawing` 抽奖的话题 → 409 `LOTTERY_DRAWN`，话题仍在 |
| 8 | 删除/预览不查 `topic.delete_any` | 只有 `view_hidden` 的人 → 403 |
| 9 | Bearer 请求能拿到管理能力（`perm.CanUser` 代替 `user.Can`） | Bearer 的版主 → 403 |

## 8. 实现时对本契约的修正（2026-09-23，只增不改）

1. **预览换了路径和名字。** G17 要求被 DELETE 寻址的路径有同路径的 GET，于是 `GET /admin/topics/{topic_id}/purge-preview` 改成 **`GET /admin/topics/{topic_id}`**，对象 **`admin_topic`**（带 `id`）——「管理员看到的这个话题」，DELETE 同一个资源。`refunded_point_escrow` 改名 **`open_lottery_escrow`**。
2. **F9 门认识了页码集合。** F9 原本只认游标集合（`total` 必须配 `include_total`），页码集合按 01 §4 恒发 `total`。门改为：响应声明了 `total_relation` 的是页码集合，必须有 `page` 参数、必须声明 `total`、不得接受 `include_total`；新增三个探针（一个通过、两个违规）。
3. **`limit` 默认 20**（原语的默认值，huma 的 default 标签是静态的），网页固定发 30。
4. `collect.ClampTotal` 应 G 轨要求加入：计数超过深度上限时封顶并回 `gte`（`/galgame` 的总数来自 catalog，会超）。
5. 变异 8 能精确测到：版主有 `topic.view_hidden` 而**没有** `topic.delete_any`（后者属 admin），于是「版主能列表、不能预览/删除」就是现成的判据。
6. 变异 9（`user.Can` 换成 `perm.CanUser`）行为测试杀不掉——测试夹具里 Bearer 的角色本来就不带管理员，换了也照样 403。它由静态守卫 `bearer_guard_test` 杀掉（扫全树的 `perm.CanUser(` 调用），这正是那条守卫存在的理由。

浏览器实测（开发库，admin 会话，独立无头 Chromium）：列表直出、「管理员隐藏」筛选、彻底删除弹窗显示各项计数与「进行中的抽奖托管着 15 萌萌点, 删除时会退回给抽奖发起人」→ `DELETE` 204 → 列表重拉；库里话题与抽奖都没了，作者缓存余额 +15。测试话题、会话已清理，余额已恢复。
