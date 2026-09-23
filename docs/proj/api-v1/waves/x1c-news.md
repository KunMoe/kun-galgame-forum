# X1c · 情报与话题 RSS

> X1 轨第三段，2026-09-23，分支 `api-v1/x1-news`，迁移号段 195–201（本段不用）。总览见 [x1a](x1a-friend-link-app.md) 开头。
> `/rss/galgame` 不在本段：它是「最新发布的作品」，要建在 G 轨的作品浏览集合上，协调会话已把它划给 G。
> 契约与变异题同一个提交，早于实现。

## 1. 现状与生产取值（2026-09-23）

情报是 infra `/v2/news` 的 BFF（记忆 `kungal-gal-news-tab`）：论坛持有一把只带 `news:read` 的 NextMoe 密钥，上游只有**游标**分页、没有偏移。

| 事实 | 数字 |
|---|---|
| 条目总数 | 4606（2022–2026 年每年 657–1086 条） |
| 来源 | 2 个：`ymgal`（月幕 Galgame）、`galgame_hihyou`（Galgame 批评） |
| 横幅 `banner_url` | 两个来源各抽样 50 条，**全部是空串** |
| 来源的 `publisher` | 线上两个都是 `null` |
| 最忙的一个月 | 不到 200 条（月视图一次取完整月，上限 600） |

## 2. 调用方

| 旧路由 | 网页 | Nitro | App |
|---|---|---|---|
| `GET /news` | `composables/useNewsFeed.ts` → `components/news/Container.vue`（`/news` 总览）、`components/home/news/Feed.vue`（首页「Gal 情报」tab） | 无 | 无 |
| `GET /news/sources` | `components/news/Container.vue`（筛选用的来源目录） | 无 | 无 |
| `GET /news/archive` | `components/news/Container.vue`、`components/news/MonthContainer.vue` | 无 | 无 |
| `GET /news/month` | `components/news/MonthContainer.vue`（`/news/[year]/[month]`，每页 50 条，外加按天的条数） | 无 | 无 |
| `GET /rss/topic` | — | `server/routes/rss/topic.xml.ts`（缓存 5 分钟） | 无 |

## 3. 普查抓到的旧面问题

| # | 问题 | 去向 |
|---|---|---|
| 1 | 上游游标原样透给客户端，**不绑定过滤条件**：换了 `source` / `lane` / 年月再用旧游标，上游照样往下翻，列表悄悄混进别的来源 | v1 用自己的 `cur_` 游标包住上游游标，指纹绑定全部过滤条件 |
| 2 | `month` 不带 `year` 时被静默忽略 | `400 INVALID_PARAMETER` |
| 3 | 页级 `sources` 表只列出当前页出现的来源，网页因此要再请求一次目录兜底（代码里的注释写着这个坑） | 条目只带 `news_source` 键，来源一律来自 `GET /news-sources` |
| 4 | 条目上的 `banner_url` 线上全是空串 | K26：不建模 |
| 5 | 话题 RSS 是第三套话题列表（自己的 SQL、`SUBSTRING` 截原始 Markdown 当摘要） | Nitro 改读 `GET /topics`，摘要取正文文档的纯文本 |

## 4. 逐条去向

| # | 旧路由 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /news` | `GET /api/v1/news-items`（游标） | public |
| 2 | `GET /news/sources` | `GET /api/v1/news-sources`（不分页） | public |
| 3 | `GET /news/archive` | `GET /api/v1/news-archive` | public |
| 4 | `GET /news/month` | `GET /api/v1/news-archive/{year}/{month}`（月概要）＋ `GET /api/v1/news-archive/{year}/{month}/items`（**页码集合**） | public |
| 5 | `GET /rss/topic` | 无新操作：Nitro 读 `GET /topics?sort=created_desc&limit=10`，逐条读 `GET /topics/{topic_id}` 的正文取纯文本摘要 | — |

5 条旧路由全删，`legacy_route_baseline` 下调 **5**。`/rss/galgame` 留在旧路由上。

## 5. 形状

### 5.1 `NewsItem`（`object: "news_item"`）

```
NewsItem  object, id, news_source, lane, title, preview, source_url, published_at
```

- `id`：上游的条目 id（十进制字符串）。
- `news_source`：来源键，开放词表（`^[a-z0-9_]{1,40}$`）。展示名、主页、转载声明一律从 `GET /news-sources` 取：**条目单独渲染时也必须带上来源的署名**（月幕与 Galgame 批评授权转载的条件），客户端用键去目录里找。
- `lane`：封闭枚举 `news` / `column`。
- `preview`：上游给的摘要，≤500；没有正文，`source_url` 是看全文的唯一入口。
- 不下发 `banner_url`（§3 #4）、`work_ids`（论坛不用）。

### 5.2 `GET /api/v1/news-items` → `CountedList[NewsItem]`（游标）

- 参数：`cursor`、`limit`（1–50，默认 20；**上限是 50 不是 100**，上游的上限）、`lane`、`news_source`、`year`（1970–9999）、`month`（1–12，必须同时给 `year`）、`include_total`。
- 顺序：上游的发布时间倒序（上游自己的键集，论坛不重排）。
- 游标：`cur_` 包住上游游标，指纹绑定 `lane`、`news_source`、`year`、`month`、`limit`。条件变了还用旧游标 → `400 INVALID_CURSOR`；上游拒绝了游标（400）也回 `400 INVALID_CURSOR`。
- `include_total=true` 才发 `total`，就是上游的 `count`，与 `items` 同一组过滤条件。
- `limit > 50` → `400 LIMIT_TOO_LARGE`。
- 上游没配置（密钥为空）或不可达 → `503 SERVICE_UNAVAILABLE`。

### 5.3 `GET /api/v1/news-sources` → `List[NewsSource]`（不分页）

```
NewsSource  object="news_source", key, display_name, homepage_url, column_url,
            attribution, publisher: UserRef | null
```

- 顺序同上游。
- `publisher`：来源在论坛上的发布账号。账号服务查不到、账号不可渲染、或者**账号服务出错**，都回 `null`：发布者只是一枚装饰性的头像，来源名和转载声明本身就能成立，不值得为它让整个目录失败（旧面的有意取舍，保留）。

### 5.4 `GET /api/v1/news-archive` → `NewsArchive`

```
NewsArchive  object="news_archive", years: [{year, count}], months: [{month, count}]
```

- 参数：`lane`、`news_source`、`year`。
- `years`：有条目的年份，倒序。
- `months`：只有 `year` 给了、且在 `years` 里时才有，1–12 月里有条目的月份；否则是空数组。
- 月份与日期一律按 **Asia/Shanghai** 切分（与旧面相同）。

### 5.5 `GET /api/v1/news-archive/{year}/{month}` → `NewsMonth`

```
NewsMonth  object="news_month", year, month, total, days: [{day, count}]
```

- 参数：`lane`、`news_source`。
- `days`：这个月**每一天**一项，没有条目的天也在（计数 0）；网页只显示非零的。
- `total`：整个月的条目数。

### 5.6 `GET /api/v1/news-archive/{year}/{month}/items` → `PageList[NewsItem]`（页码）

- 参数：`page`、`limit`（1–100，默认 20；网页固定用 50）、`lane`、`news_source`、`day`（1–31，缺席 = 整月）。
- 为什么是页码：月视图是参考页，读者要能直接跳到第三周；一个月最多几百条，整月取来再切页。
- `total` 是按 `day` 过滤后的条数，`total_relation` 恒为 `eq`。
- 顺序同上游（发布时间倒序）。

## 6. 错误码

**无新增。**

## 7. 网页

| 文件 | 改什么 |
|---|---|
| `composables/useNewsFeed.ts` | 读 `GET /news-items`（`include_total=true`）；来源表改由 `useNewsSources()` 从 `GET /news-sources` 取；「过滤变了就作废在途请求」的代际计数保留 |
| `components/news/Container.vue`、`home/news/Feed.vue`、`Partners.vue`、`Filters.vue`、`Card.vue`、`GroupHeader.vue` | 字段改名（`source_key` → `news_source`、`name` → `display_name`、`count` → `total`、`publisher` 是 `UserRef`） |
| `components/news/MonthContainer.vue` | `GET /news-archive/{year}/{month}` + `…/items?page=&day=&limit=50` + `GET /news-archive?year=` |
| `server/routes/rss/topic.xml.ts` | `GET /topics?sort=created_desc&limit=10`；每条 `GET /topics/{id}` 取正文纯文本（前 233 个字符）作摘要；作者取 `author` |
| `shared/types/news.ts` 的手写接口 | 删除或改成生成物别名；`groupNewsItems` 改吃新形状 |

## 8. 变异题（先于实现提交）

| # | 破坏 | 必须变红的断言 |
|---|---|---|
| 1 | 游标指纹去掉 `news_source` | 换来源复用游标 → `400 INVALID_CURSOR` |
| 2 | 年月窗口不传给上游 | 上游收到的 `published_after` / `published_before` 就是该月的北京时间边界 |
| 3 | `month` 不带 `year` 被静默忽略 | → `400 INVALID_PARAMETER` |
| 4 | `include_total=true` 时不发 `total` | `total` 等于上游 `count` |
| 5 | 上游拒绝游标被映射成 503 | 上游对游标回 400 → `400 INVALID_CURSOR` |
| 6 | 上游未配置被映射成 500 | → `503` |
| 7 | 月条目忽略 `day` | 只剩那一天的条目，`total` 同步变小 |
| 8 | `days` 丢掉零条目的天 | 长度等于该月天数 |
| 9 | 月条目的页偏移错一页 | 第 2 页的第一条紧接第 1 页的最后一条 |
| 10 | 来源目录在账号服务出错时回 503 | 仍回 200，`publisher` 为 `null` |
| 11 | `months` 对不在 `years` 里的年份也去问上游 | 该年没有条目 → `months` 为空且不产生上游请求 |

上游用 `httptest.Server` 假冒，按请求参数回数据，并记录收到的查询串。

## 9. 迁移

**无。**
