# X2-overview · 管理后台数据总览

> 2026-09-23 立。分支 `api-v1/x2-overview`，迁移号 207（**不用**）。X 轨 galgame 相关的一半（X2）的一段，由协调会话派发。
> 契约与变异题同一个提交，早于实现。

## 1. 普查

### 1.1 路由与调用方

两条旧路由由 `AdminOverviewHandler`（`internal/admin/handler/overview_handler.go`）独占，service / repository 只有它一个用户。

| # | 旧 | 档 | 调用方 |
|---|---|---|---|
| 1 | `GET /api/admin/overview/all` | `admin.dashboard` | `pages/admin/overview.vue`（「加和数据」卡片） |
| 2 | `GET /api/admin/overview/stats?days=` | `admin.dashboard` | 同上（「增量数据」卡片 + `components/admin/OverviewChart.vue` 面积图） |

`apps/web/server/` 零调用；`../kungal-apps` 零调用。`admin.dashboard` 只在管理员基线里（版主没有），生产的角色覆盖与个人覆盖都没有授予它。

### 1.2 生产取值（`kungalgame`，2026-09-23）

九项指标的来源与当前值：

| 旧 `name` | 来源 | 生产 |
|---|---|---|
| `topic` | `topic` 全表 | 3565（其中隐藏 322） |
| `topic_reply` | `topic_reply` 全表 | 14884（隐藏 3） |
| `topic_comment` | `topic_comment` 全表 | 3165 |
| `galgame` | `galgame` 全表 | **16143，其中只有 9773 行 `published`** |
| `galgame_resource` | `galgame_resource` 全表 | 50735 |
| `galgame_comment` | `feed_activity` 里 `GALGAME_COMMENT_CREATION` | 10235 |
| `galgame_website` | `galgame_website` 全表 | 87 |
| `galgame_website_comment` | `feed_activity` 里 `GALGAME_WEBSITE_COMMENT_CREATION` | 119 |
| `chat_message` | `chat_message` 全表 | 9956 |

所有 `created` 列都是 `timestamptz`。应用连接的会话时区是 `Asia/Shanghai`（`database/postgres.go` 的 `withTimeZone`），所以旧 SQL 里的 `date_trunc('day', created)` 碰巧按北京日分桶。

### 1.3 语义与疑似 bug

1. **「Galgame」总数把 6370 个未发布的本地行也算进去了。** 这些是懒建的桩行（草稿、撤回的认领、catalog 名下论坛只碰过一下的作品），不在任何浏览面上。站点其它读者（浏览、RSS、排行、首页 feed，迁移 070）都只认 `published`。
2. **窗口多一天。** `since` 是「今天减 N 天」那天的北京零点，所以 `days=7` 返回 8 个日历日（今天 + 前 7 天），网页却写「最近 7 天新增」并把它们加起来。
3. **序列是稀疏的。** 只有至少一项指标非零的日期才出现，图表的横轴会跳日。
4. **分桶依赖会话时区。** SQL 自己不写时区，换一个不带 `TimeZone` 的连接（测试库、迁移工具），桶就按 UTC 切，早 8 小时。
5. `stats` 的每一行是动态键的 map（`{date, topic, topic_reply, …}`），`all` 下发中文 `label`。
6. `days` 缺省 30，越界（0 或 >365）由旧校验器回 `233`。
7. 两种来源的谓词不同：本地表的计数含隐藏内容；`feed_activity` 的计数不含（隐藏时触发器删掉 feed 行，迁移 056）。这是现状，写进描述，不改。

## 2. 范围

| 旧 | v1 |
|---|---|
| `GET /api/admin/overview/all` | `GET /api/v1/admin/overview` |
| `GET /api/admin/overview/stats?days=` | `GET /api/v1/admin/overview/daily?days=` |

两条都删，`legacy_route_baseline` 下调 2（合并时以 rebase 后重新生成为准）。

## 3. 形状

九项指标在两个响应里同名同型（G8），全部是 `integer ≥ 0`、名字以 `_count` 结尾：

| 字段 | 旧 `name` | 定义 |
|---|---|---|
| `topic_count` | `topic` | 话题，含隐藏 |
| `reply_count` | `topic_reply` | 回复，含隐藏 |
| `topic_comment_count` | `topic_comment` | 话题评论 |
| `galgame_count` | `galgame` | **已发布**的作品（`galgame.published`），与浏览面同一谓词 |
| `galgame_resource_count` | `galgame_resource` | 作品资源 |
| `galgame_comment_count` | `galgame_comment` | 作品评论墙的帖（社区原语的本地 feed 镜像，不含已隐藏） |
| `website_count` | `galgame_website` | 网站目录条目 |
| `website_comment_count` | `galgame_website_comment` | 网站评论墙的帖（同上） |
| `direct_message_count` | `chat_message` | 私信 |

### 3.1 `AdminOverview`（`GET /admin/overview`）

`{ "object": "admin_overview", <九项> }`：各指标自建站以来的总数。

### 3.2 `OverviewDay`（`GET /admin/overview/daily`）

`{ "object": "list", "items": [ { "object": "overview_day", "bucket_date": "2026-09-23", <九项> }, … ] }`

- `days`：1–365，缺省 30。
- 恰好 `days` 个桶：今天（北京日）及之前的 `days − 1` 天，按日期升序。**稠密**：没有新增的日子也在，九项全为 0。
- 今天的桶只统计到请求那一刻。
- `bucket_date` 用 `_date` 结尾（F1：`date` 格式的字段名必须以 `_date` 结尾）。

**K-X2O1 · 管理后台的日桶是北京日。** 与 U1 签到的 K22 同一条规则：`Asia/Shanghai`（UTC+8，无夏令时）的日历日。SQL 里显式写 `created AT TIME ZONE 'Asia/Shanghai'`，不依赖连接的会话时区。

**K-X2O2 · 窗口是 `days` 个桶，不是 `days + 1` 个**（§1.3 第 2 条）。

## 4. 错误

| 状态 | code | 何时 |
|---|---|---|
| 401 | `MISSING_CREDENTIAL` / `INVALID_CREDENTIAL` | 鉴权档 |
| 403 | `PERMISSION_REQUIRED` | 没有 `admin.dashboard`（版主、Bearer 请求） |
| 403 | `ACCOUNT_BANNED` | 鉴权档 |
| 400 | `INVALID_PARAMETER` | `days` 不是 1–365 的整数 |
| 500 / 503 | `INTERNAL_ERROR` / `SERVICE_UNAVAILABLE` | 通用 |

错误码无新增。能力判断走 `user.Can(perm.AdminDashboard)`，Bearer 永远 403。

## 5. 网页

- `pages/admin/overview.vue` 与 `OverviewChart.vue` 换类型化客户端，指标标签由 `constants/admin.ts` 按字段名给出（服务端不再下发中文 `label`）。
- 图表组件里的 ApexCharts 面积填充渐变（铁律 #1 的第四个文件，带注释）**原样保留**。
- 删 `shared/types/admin.ts` 里的 `AdminOverStats`（`AdminUserContentStats` 属于 U 轨，不动）。
- `legacy-fetch-baseline` 下调 2。

## 6. 变异题（先于实现提交）

测试用注入的时钟（固定在远未来的某天），种子行放在那个窗口里，不受库里其它行影响；总数断言看「种子前后之差」。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 去掉 `admin.dashboard` 检查 | 版主 → `403 PERMISSION_REQUIRED` |
| 2 | `user.Can` → `perm.CanUser` | 被个人授予 `admin.dashboard` 的用户经 Bearer 访问 → 403 |
| 3 | 按 UTC 分桶 | UTC 17:30 的行落在北京次日的桶里 |
| 4 | 稀疏序列（跳过全零日） | `days=5` 恰好 5 个桶，中间的空日九项全 0 |
| 5 | 窗口多一天（旧行为） | `days=5` 的第一个桶是「今天 − 4」，不是「今天 − 5」 |
| 6 | `galgame_count` 不看 `published` | 种一个未发布作品，总数与日桶都不变 |
| 7 | 网站评论取成作品评论的 feed 类型 | 两类 feed 各种不同数量，各自计数正确 |
| 8 | 不校验 `days` 上界 | `days=366` → 400 |
| 9 | 日桶升序改成降序 | 第一个桶最早 |
| 10 | 总数漏掉隐藏话题（加了 `status = 0`） | 种一个隐藏话题，`topic_count` 增 1 |
