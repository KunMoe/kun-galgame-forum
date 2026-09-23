# X1b · 版块

> X1 轨第二段，2026-09-23，分支 `api-v1/x1-section`，迁移号段 195–201（本段不用）。总览见 [x1a](x1a-friend-link-app.md) 开头。
> 契约与变异题同一个提交，早于实现。

## 1. 生产取值（2026-09-23，只读）

| 事实 | 数字 |
|---|---|
| `topic_section` | 27 行，id 1–27：`g-` 7 个、`t-` 11 个、`o-` 9 个；与 `SectionSlug` 封闭枚举逐一对应 |
| 每个版块的话题关联 | 2（`t-adobe`）… 904（`g-other`），没有空版块 |
| 话题 `status` | 0 有 3243 条，1（隐藏）有 322 条 |

## 2. 调用方

| 旧路由 | 网页 | Nitro | App |
|---|---|---|---|
| `GET /section?section=&page=&limit=&sort_order=` | `components/section/Container.vue`（`/section/[name]` 页，页码分页，每页 30） | 无 | 无 |
| `GET /category?category=` | `pages/category/[name].vue` → `components/category/Container.vue`（每个版块一张卡：话题数、浏览数、最新话题） | 无 | 无 |

两个 handler 都在 `internal/section`，除 `app.go` 外没有导入方，与 X2 无交集。

## 3. 普查抓到的旧面问题

| # | 问题 | 去向 |
|---|---|---|
| 1 | **版块页不看成人向偏好**：`FindSectionTopics` 没有 `is_nsfw` 条件，SFW 访问者在版块页照样看到 NSFW 话题（首页列表是过滤的） | 版块页改走 `GET /topics`，与首页同一个 `include_nsfw` 闸 |
| 2 | 版块页的 `total` 把被封禁作者的话题也算进去，而条目里把它们丢了：页码与条目数对不上 | 游标集合不发 `total`（F9） |
| 3 | 版块页是第二套话题列表：自己的 SQL、自己的形状（`user`、`is_nsfw_topic`、`created`、裸 `content` 截断）、页码分页；与 W0b 的话题列表是两套口径 | 并入 `GET /topics`（加 `section` 过滤），渲染复用 `TopicCard` |
| 4 | 分类统计里「最新话题」取候选时吞掉错误（`Scan` 结果不看），用户查询失败也被 `Hydrate` 吞掉 | 500 / 503 |
| 5 | 没有话题的版块在分类页上直接消失（`INNER JOIN`） | 27 个版块都在，计数为 0、`latest_topic` 为 `null`（生产上目前没有空版块） |

## 4. 逐条去向

| # | 旧路由 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /section` | `GET /api/v1/topics?section=…`：**已有的话题集合加一个过滤参数**，是加法 | optional（与现有 `listTopics` 同） |
| 2 | `GET /category` | `GET /api/v1/sections?category=…` | public |

2 条旧路由全删，`legacy_route_baseline` 下调 **2**；`internal/section` 整个删除。

## 5. 形状

### 5.1 `GET /api/v1/topics` 加 `section`

- `section`：`SectionSlug` 封闭枚举（与 `Topic.sections[]` 同一个词表），缺席 = 不过滤。
- 语义：话题的 `sections` 里含有它。与 `category`、`include_nsfw`、`sort` 自由组合；`category` 与 `section` 的前缀不一致时结果为空，不报错（与「过滤就是 AND」一致）。
- 游标指纹加入 `section`：换版块复用游标 → `400 INVALID_CURSOR`。
- 其余一切（可见性、排序、封禁作者的处理、`TopicSummary` 形状）不变。

### 5.2 `GET /api/v1/sections` → `List[Section]`（不分页）

```
Section       object="section", section, category, topic_count, view_count,
              latest_topic: SectionLatestTopic | null
SectionLatestTopic  object="topic", id, title, created_at
```

- 参数：`category`（`galgame` / `technique` / `others`，缺席 = 全部 27 个）。不分页：词表是封闭的 27 项，响应仍是 `{object: "list", items}`，没有 `next_cursor`。
- 顺序：`topic_section.id ASC`（词表顺序）。
- `section`：`SectionSlug`；`category`：与 `Topic.category` 同一个枚举，由版块前缀决定。库里出现词表以外的版块名 → 500。
- `topic_count` / `view_count`：该版块里**已发布**（`status != 1`）、**匿名可读**（`access_scope = 'public'`）、**分类一致**的话题数与浏览数之和；NSFW 与封禁作者的话题都计入。这是版块规模的统计，不是某个列表的 `total`，口径沿用旧面并写进 description。
- `latest_topic`：同一口径下 `created` 最新、且作者可渲染（`userclient.IsRenderable`）的话题；取最新 10 条候选，都不可渲染则 `null`。只下发 `id`、`title`、`created_at`，与 `Topic` 同名同型。
- 用户查询失败 → 503（与话题列表一致）；数据库失败 → 500。

## 6. 错误码

**无新增。**

## 7. 网页

| 文件 | 改什么 |
|---|---|
| `components/section/Container.vue` | `useCursorList` + `GET /topics?section=…&sort=created_desc&include_nsfw=<立场>`，条目用 `TopicCard`，「加载更多」取代页码（与首页话题列表一致）；去掉 `usePageQuery` |
| `pages/category/[name].vue`、`components/category/Container.vue` | `GET /sections?category=…`；`section.section`、`latest_topic.created_at` |
| 手写类型 `SectionTopicList`、`CategorySectionStats` | 删除 |

## 8. 变异题（先于实现提交）

| # | 破坏 | 必须变红的断言 |
|---|---|---|
| 1 | `/topics` 的 `section` 过滤不生效 | 结果只含该版块的话题 |
| 2 | 游标指纹不含 `section` | 换版块复用游标 → `INVALID_CURSOR` |
| 3 | `section` 过滤 + 走完全部页时去掉 `id` 决胜键 | 翻完与 SQL 逐条相等（**种子里同版块有并列的 `created`**） |
| 4 | `/sections` 的计数算进隐藏话题 | `topic_count` 等于已发布话题数 |
| 5 | `/sections` 的计数算进非公开（`login`）话题 | 同上 |
| 6 | `latest_topic` 不跳过被封禁作者 | 最新话题的作者被封禁 → 取下一条 |
| 7 | `latest_topic` 取最旧而不是最新 | 是 `created` 最新的那条 |
| 8 | `/sections` 的 `category` 过滤不生效 | 只返回该分类的版块 |
| 9 | 空版块被丢掉（`INNER JOIN`） | 没有话题的版块在列表里，计数 0、`latest_topic` 为 `null` |
| 10 | 用户查询失败被吞掉 | 账号服务不可用 → 503 |

## 9. 迁移

**无。**

## 10. 实现时对本契约的修正（2026-09-23，只增不改）

1. **F1 的 URL slug 具名例外加上单数 `section`**。版块词表带连字符（`g-walkthrough`），F1 原本只对属性名 `sections` 放行；`listTopics` 的 `section` 参数与 `Section.section` 是同一个词表，理由与原例外相同（它是 `/section/{key}` 的 URL 段）。01 §3 的例外表与 02 §3 的 F1 行同步改了。
2. **`listTopics` 的游标指纹只在带 `section` 时才追加这一项**：上线前发出的游标不会失效。
3. **`/sections` 在 public 档上手动声明了 503**：它要问账号服务「最新话题」的作者是否可渲染，而 public 档不会自动推导出 503。
4. `topic_section.name` 超出封闭词表 → 500（`topicapiv1.IsSectionSlug`，与 `SectionSlug` 同一个来源）。
5. 网页：版块页对不认识的 slug 直接显示「没有这个版块」，不发请求；分类页对不认识的分类名显示「没有这个分类」。

### 变异执行结果

10/10 杀：1–3 `TestV1TopicsSectionFilterWalk`；4–8 `TestV1SectionsCountOnlyPublishedPublicTopicsOfTheirCategory`；9 `TestV1SectionsKeepEmptySections`；10 `TestV1SectionsNeedAccounts`。

### 浏览器实测（开发库，无头 Chromium，匿名 + 开了成人向显示的登录用户，18 项）

- 分类页列出 galgame 的 7 个版块，最新话题标题都显示出来。
- 从分类页点进版块页（客户端导航）：列出 30 条话题；请求是 `GET /topics?section=g-walkthrough&sort=created_desc&limit=30`，`include_nsfw` 跟着立场走（匿名 `false`、登录用户 `true`）。
- 「加载更多」会带上游标：匿名 30 → 42 条，登录用户 30 → 47 条。多出来的 5 条是 NSFW 话题，旧面对匿名访问者也会把它们列出来。
- 未知版块、未知分类都显示空态。没有意外的报错。
