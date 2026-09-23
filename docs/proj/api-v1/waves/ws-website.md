# WS · 网站目录：网站、分类、标签、标签分组

> WS 轨，2026-09-23。`internal/website` 的 21 条旧路由：`/website` 7 + `/website-tag-group` 4 + `/website-tag` 5 + `/website-category` 5。后两组原本记在 D 轨，D 会话核实后移过来（两个包里 `TagHandler` / `CategoryHandler` 只是类型同名；网站、标签、分类三个 service 互持对方的 repo，详情互相嵌，是一个连通分量）。迁移号段 175–179。
> 契约与变异题同一个提交，早于实现。普查全文（21 条逐条、24 条疑似 bug）存在会话里，本文只收结论与数字。

## 1. 普查（生产实测 2026-09-23）

### 1.1 数字

| 事实 | 数字 |
|---|---|
| 行数 | 网站 87（id 1–95，缺 8 个）、分类 4、标签 71、标签分组 18、标签关系 1455、点赞 80、收藏 43 |
| `age_limit` | `all` 68、`r18` 19 |
| `status` | `normal` 83、`closed` 4、**`unreachable` 0** |
| `language` | `zh-cn` 78、`ja-jp` 5、`en-us` 3、`zh-tw` 1；库默认值 `'JA'` 从未出现 |
| `url`（页面地址的键） | 87/87 小写 LDH，4–29 字符，按小写也唯一；30 个 `www.` 开头，1 个 punycode |
| `name` | 2–30 字符，按小写也唯一 |
| `description` | 11–169 字符，6 条多行，2 条首尾有空白 |
| 图标 | 59 行有 `icon_image_hash`（55 个不同 hash，全是 64 位十六进制）；**28 行只有第三方 favicon 外链**；5 行外链以空格开头（id 6、8、28、31、48，其中 8、31 没有 hash） |
| `domain` jsonb | 全是字符串数组：长度 0–10，共 171 个元素；170 个带 `http(s)://`，**1 个是裸主机名**（`hacg.icu`，id 9）；最长 56；82/87 行把主地址也列在里面——它的意思是「这个站的所有地址」 |
| `create_time` | 4–11 字符的自由文本：`yyyy-mm-dd` 65 行、**5 行带前导空格**、`1 year ago` 4、`约 2014 年` 之类若干 |
| 每站标签数 | 3–20，平均 16.7；单选分组里一站两标签：**0 例** |
| 标签等级和（旧 `price` = `level`） | −39…236；按现行标签表理论上限 275（网页写死 245） |
| 计数 | 点赞/收藏计数与行数**零漂移**；`view` 49…233176 |
| `created` 并列 | 87 行只有 73 个不同值：14 对毫秒级重复，**45 行落在同一秒**（2025-07-20 导入） |
| `updated` | 87 行全在最近 5 天——是「最后一次被浏览」，不是「最后一次编辑」（§4 E1） |
| 标签 | 71 个 `name` 全匹配 `^[a-z0-9_-]+$`、≤14 字符；`level` −100…20；3 个没被用；0 个无分组 |
| 分组 | 18 个，`sort_order` 10…180 无并列；只有 `misc` 是多选 |
| 分类 | 4 个：`resource`(55 站/SFW 43)、`community`(12/11)、`telegram`(1/1)、`other`(19/13) |
| 孤儿 | 所有外键路径 0 |
| 权限 | 代码里 moderator 有 `website.*`，**生产 `role_permission_override` 把三个都收回了**：实际只有 admin / ren 能写 |

### 1.2 调用方

- `apps/web/server/**`：零调用。Flutter App：零调用。`docs/proj/app-direct-api.md` 没有这些路由。
- 网页：`components/website/Container.vue`（列表）、`pages/website/[domain].vue` + `Operation.vue` + `detail/*`（详情、赞、收藏、编辑、删除）、`pages/website-category/[name].vue`、`pages/website-tag/[name].vue`、`composables/useWebsiteTaxonomy.ts`（分类与分组）、`website/modal/*` 与 `admin/website/*`（后台）。
- 别的域读这些表但不走这些路由：网站评论墙（RC，`wall_v1` 用 `url` / `age_limit` 拼链接）、首页动态（`feed_activity` 触发器）、通知镜像（读时按 id 解析当前 `url`）、管理总览、删号工具。都不受影响。

### 1.3 旧面的问题（普查编号 E*）

| 编号 | 事实 |
|---|---|
| E1 | 浏览计数与赞/收藏计数用 GORM `Update`，顺带把 `updated` 刷成现在：详情页的「更新于」与 JSON-LD `dateModified` 其实是最后浏览时间 |
| E2 | 每次详情浏览写两次：`UPDATE galgame_website` + 无条件触发的 `trg_feed_galgame_website` → `feed_upsert` |
| E3 | 赞、收藏、改、删 5 条路由忽略路径参数，从 body/query 读 `website_id`；网页往 `:domain` 里塞数字 id |
| E4 | 页面键 `url` 可改且无别名：3 次改名留下 8 条死掉的评论动态链接、5 条死掉的通知链接 |
| E5 | 5 个站在界面上改不了：表单原样回传带前导空格的 `icon`，后端 `url` 校验拒掉 |
| E6 | 网站详情不走 SFW 闸（列表、分类详情、标签详情都走） |
| E7 | 分类列表的 `website_count` 含 r18，分类详情不含：SFW 读者看到 55 与 43 |
| E8/E9 | 改/删不存在的 id 回成功；外键缺失、名字撞车回 500 或误导性的 400 |
| E10 | 赞/收藏是切换，不回新状态与计数；并发双击撞主键 → 500 |
| E11 | 分类详情、标签详情没有 `ORDER BY`；列表只按 `created` 排且有并列 |
| E12 | 只在网页里执行的规则：每站 ≤20 标签、单选分组互斥、语言词表、主地址小写、分类与分组名字符集；标签名连网页都没有字符集规则 |
| E13 | 吞错误：详情把任何仓储错误（含库挂了）当 404；列表忽略 `Scan` 错误 |
| E14 | `icon_image_hash` 从不被 reference-ping，图床看不到这些图在用（23/55 超过 60 天未刷新） |
| E15 | 删号清理 `DELETE FROM galgame_website WHERE user_id = ?` 会连带删掉别人对这些站的赞与收藏 |
| E16 | 删网站留下死掉的评论动态行和上游孤儿评论串 |
| E20 | 详情的 `comment` 数组只给 JSON-LD 用，每次浏览多一次社区调用 |
| E22 | `GET /website-tag` 直接下发 GORM 模型 |

## 2. 范围与寻址

**寻址沿用 D 轨与 T4 的模式：公开读按页面地址里的键（`host` / `slug`），写面按不可变的 id 挂在 `/admin` 下，管理路径下发管理表示。** 页面地址 `/website/<host>`、`/website-category/<slug>`、`/website-tag/<slug>` 都不变，公开读一次请求就拿到；键会被改名（E4），所以写面与跨资源引用一律用 id。两种键不能共用 `/websites/{x}`（OpenAPI 不允许同层模板只差参数名），G17 又要求每条写路径有同路径的 GET。

| # | 旧 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /website` | `GET /api/v1/websites`（游标） | public |
| 2 | `GET /website/:domain` | `GET /api/v1/websites/{website_host}` | optional |
| 3 | `PUT /website/:domain/like`（切换） | `PUT` / `DELETE /api/v1/websites/{website_host}/like` | required |
| 4 | `PUT /website/:domain/favorite`（切换） | `PUT` / `DELETE /api/v1/websites/{website_host}/favorite` | required |
| 5 | `POST /website` | `POST /api/v1/admin/websites` → **201** | required，`website.create` |
| — | — | `GET /api/v1/admin/websites/{website_id}`（编辑源；G17） | required，`website.edit` |
| 6 | `PUT /website/:domain` | `PATCH /api/v1/admin/websites/{website_id}` | required，`website.edit` |
| 7 | `DELETE /website/:domain` | `DELETE /api/v1/admin/websites/{website_id}` → **204** | required，`website.delete` |
| 8 | `GET /website-tag-group` | `GET /api/v1/website-tag-groups`（游标） | public |
| 9–11 | `POST` / `PUT` / `DELETE /website-tag-group` | `POST /api/v1/admin/website-tag-groups`；`GET` / `PATCH` / `DELETE /api/v1/admin/website-tag-groups/{website_tag_group_id}` | required，`website.*` |
| 12 | `GET /website-category` | `GET /api/v1/website-categories`（游标） | public |
| 13 | `GET /website-category/:name` | `GET /api/v1/website-categories/{website_category_slug}` | public |
| 14–16 | `POST` / `PUT` / `DELETE /website-category` | `POST /api/v1/admin/website-categories`；`GET` / `PATCH` / `DELETE /api/v1/admin/website-categories/{website_category_id}` | required，`website.*` |
| 17 | `GET /website-tag` | `GET /api/v1/website-tags`（游标） | public |
| 18 | `GET /website-tag/:name` | `GET /api/v1/website-tags/{website_tag_slug}` | public |
| 19–21 | `POST` / `PUT` / `DELETE /website-tag` | `POST /api/v1/admin/website-tags`；`GET` / `PATCH` / `DELETE /api/v1/admin/website-tags/{website_tag_id}` | required，`website.*` |

21 条旧路由全删，`legacy_route_baseline` 下调 21。v1 共 27 个操作（网站 10、分类 6、标签 6、分组 5）。

分类详情与标签详情里的网站列表不再嵌在详情里：页面拿到分类/标签的 `id` 后，另走 `GET /websites?website_category_id=` / `?website_tag_id=`（带 `include_total`）。一个资源不嵌另一个集合（K10），而且这样计数与条目同一谓词（E7）。

## 3. 形状

### 3.1 命名

| 旧 | v1 | 为什么 |
|---|---|---|
| `url`（主机名） | `host` | 它是主机名不是 URL；路径参数 `{website_host}` |
| `name`（网站名） | `title` | `name` 在 `UserRef` 里是可空字符串，G8 要同名同型 |
| `domain`（jsonb 数组） | `urls` | 这个站的全部地址（含主地址与镜像），每个都是带 scheme 的 URL |
| `create_time` | `founded` | 站方自述的建站时间，自由文本（`约 2014 年`），不是时间戳，所以不叫 `_at` / `_date` |
| `age_limit` `all` / `r18` | `is_nsfw` | 与话题同名同义 |
| `status` | `state` | 取值 `normal` / `unreachable` / `closed` 不变（`unreachable` 0 行，但它是后台下拉里的合法选项） |
| `language` | `language` | **开放词表**（BCP 47 语言标签，按库里的小写形式下发：`zh-cn`、`ja-jp`…）。它是标准词表不是论坛自己的，封闭枚举要求 snake_case，写成 `zh_cn` 就不再是语言标签了 |
| `price` 与 `level`（两个同值字段） | `score` | 该站全部标签的 `level` 之和；可以是负数 |
| `icon` + `icon_image_hash` + `icon_url` | `icon: Image \| null` + `external_icon_url` | 见 §3.3 |
| `view` | `view_count` | |
| `category`（卡片上是 slug 字符串，详情里是对象） | `website_category: WebsiteCategoryRef` | 不叫 `category`：话题的 `category` 是另一套封闭枚举（D 轨同理改叫 `doc_category`） |
| `tags` | `website_tags: [WebsiteTag]` | G 轨迟早会有 galgame 的 `tags`，元素类型不同 |
| 分类/标签/分组的 `name`（URL 键） | `slug` | |
| 分类/标签/分组的 `label` | `label` | 显示名；不叫 `name` 的理由同上 |
| 标签的 `group_id` | `website_tag_group_id` | |
| 分组的 `multi_select` | `is_multi_select` | F1 |
| `is_liked` / `is_favorited` | `viewer.has_liked` / `viewer.has_favorited` | K9 |

### 3.2 对象

**`WebsiteSummary`（`object: "website"`，列表条目）**：`id`、`host`、`title`、`description`、`icon`、`external_icon_url`、`website_category`、`is_nsfw`、`state`、`score`。

**`Website`（`object: "website"`，公开详情）**：`WebsiteSummary` 全部字段，加 `language`、`urls`、`founded`、`website_tags`、`view_count`、`like_count`、`favorite_count`、`comment_count`、`created_at`、`updated_at`、`viewer`。

- `viewer`：`has_liked`、`has_favorited`、`can_edit`（`website.edit`）、`can_delete`（`website.delete`）；匿名为 `null`；Bearer 的两个 `can_*` 恒假。
- `website_tags` 按（分组 `sort_order`、分组 id、标签 `level` 降序、标签 id）排；无分组的排最后。
- 不发 `comment`（E20）：评论走 RC 的 `wall-comments`；网页的 JSON-LD 评论自己取那一页。
- 不发作者：87 行里 83 行是导入数据的占位用户 2。

**`AdminWebsite`（`object: "admin_website"`，编辑源）**：与写面请求体一一对应——`id`、`host`、`title`、`description`、`icon`（预览用）、`external_icon_url`、`website_category_id`、`website_tag_ids`、`is_nsfw`、`state`、`language`、`urls`、`founded`、`created_at`、`updated_at`。编辑表单只回传改过的字段（`PATCH`），E5 随之消失。

**`WebsiteCategoryRef`**（`object: "website_category"`）：`id`、`slug`、`label`。
**`WebsiteCategory`**（同 object）：加 `description`、`sort_order`、`website_count`（**全部**网站，含 NSFW——它是后台「还有 N 个站，不能删」的依据；页面上的计数来自网站列表的 `total`，与条目同一谓词）。
**`WebsiteTag`**（`object: "website_tag"`）：`id`、`slug`、`label`、`description`、`level`、`website_tag_group_id`（可空）。
**`WebsiteTagGroup`**（`object: "website_tag_group"`）：`id`、`slug`、`label`、`description`、`sort_order`、`is_multi_select`。
**`AdminWebsiteCategory` / `AdminWebsiteTag` / `AdminWebsiteTagGroup`**（`admin_website_category` / `admin_website_tag` / `admin_website_tag_group`）：公开表示的全部字段（分类不带 `website_count`），加 `created_at`、`updated_at`。它们只为 G17 与写面的响应存在。

**`WebsiteEngagement`**（`object: "website_engagement"`，赞/收藏的响应）：`website_id`、`like_count`、`favorite_count`、`viewer: {has_liked, has_favorited}`。

### 3.3 图标

- `icon`：有 `icon_image_hash` 就是完整的 `Image`（宽高、thumbhash、`sexual` 由图床元数据填），否则 `null`。
- `external_icon_url`：**只在 `icon` 为 `null` 时**才可能非空——28 个站从没上传过图标，库里只有第三方 favicon 外链。形式是 `uri`；前导空格由迁移 175 清掉。
- `Image` 不能装外链（B10：一个类型处处如此，`hash` 必填），所以外链单独一个字段，而不是塞进 `Image`。这是过渡字段：那 28 个图标补传进图床以后，它恒为 `null`，可以删（preview 期）。
- 写面只收 `icon_image_hash`（64 位十六进制或空串表示清除）；外链不可写（旧表单也没有输入框）。

### 3.4 列表

- `GET /websites`：`cursor`、`limit`（1–100）、`include_nsfw`（默认 `false`，缺席即排除 `is_nsfw`）、`website_category_id`、`website_tag_id`、`include_total`。排序固定 `created_at DESC, id DESC`，没有 `sort` 参数（网页按 `score` 在本地排，它一次拿全）。游标绑定全部过滤参数。
- `GET /website-categories`、`GET /website-tag-groups`：排序 `sort_order ASC, id ASC`。
- `GET /website-tags`：排序 `id ASC`（与旧面相同）。
- 四个词表集合都是游标，`limit` 上限 100：标签 71 个，一页拿完。

### 3.5 赞与收藏（K16 槽位）

- `PUT` 置位、`DELETE` 撤销，都幂等、都回 200 + `WebsiteEngagement`。重放不改计数。
- 计数随行的插入/删除原子地加减（`INSERT … ON CONFLICT DO NOTHING` 真插进去才 `+1`），不再用 GORM `Update`，**不碰 `updated`**（E1）。
- 调用者先按 OAuth 当前记录判封禁（RC 教训 ④）；不查 `is_nsfw`（能打开详情就能赞）。
- 不发萌萌点、不发通知（旧面也不发）。

### 3.6 浏览计数

`GET /websites/{website_host}` 照旧 `view + 1`（话题详情同样在 GET 里计数），改用不带 `updated` 的列更新（E1）。迁移 175 把动态触发器收窄到 `UPDATE OF name, url, age_limit, user_id, created`，浏览不再顺带写 `feed_activity`（E2）。

### 3.7 写面

- 全部要对应的 `website.*` 权限（`user.Can`，Bearer 恒无）；先按 OAuth 当前记录判封禁。
- **网站**（`WebsiteCreate` / `WebsitePatch`）：
  - `host`：先判原始长度（≤253），再转小写，然后必须是合法主机名（LDH 标签、至少一个点、无尾点）→ 否则 `422 VALIDATION_FAILED` `INVALID_FORMAT`。
  - `title` 1–233；`description` 10–1000（去空白后判下限）；`founded` ≤20，去首尾空白；`urls` ≤10 个，每个 ≤100 且是 `http(s)` URL；`language` ≤10 且匹配语言标签形状（`^[a-z]{2,3}(-[a-z0-9]{2,8})*$`，存小写）。
  - `website_category_id` 不存在 → `422` `/website_category_id` `UNKNOWN_REFERENCE`。
  - `website_tag_ids`：≤20、不许重复（`DUPLICATE_ITEM`）、每个都要存在（`/website_tag_ids/{i}` `UNKNOWN_REFERENCE`）、单选分组里至多一个（第二个 → `/website_tag_ids/{i}` `INCONSISTENT_WITH`）。`PATCH` 里出现即整组替换，缺席不动。
  - `host` 或 `title` 已被别的站占用 → `409 ALREADY_EXISTS`，`errors[]` 指向那个字段（`NOT_ALLOWED_VALUE`）。
- **分类 / 标签 / 分组**：`slug` 1–30 且 `^[a-z0-9_-]+$`；`label` 1–30；`description` ≤300；`sort_order` 0–9999；标签 `level` −100…20；标签的 `website_tag_group_id` 不存在 → `422 UNKNOWN_REFERENCE`，`PATCH` 里 `null` 表示移出分组。`slug` 撞车 → `409 ALREADY_EXISTS`。
- **删除**：网站硬删（赞、收藏、标签关系随外键级联；动态行由触发器删）。标签删掉即从所有站上摘掉。分组删掉，组内标签变成无分组。**分类下还有网站 → `409 WEBSITE_CATEGORY_NOT_EMPTY`**（扩展成员 `website_count`），什么都不删。
- 改/删不存在的 id → `404`（E8）。创建返回 `201` + `Location` + 管理表示。

## 4. 逐条裁决

| 编号 | 裁决 |
|---|---|
| E1 | 计数改列更新，不再刷 `updated`。**库里 87 行的 `updated` 已经是浏览时间，找不回来**；迁移不动它，下一次真编辑之后才准 |
| E2 | 迁移 175 收窄触发器列 |
| E3 | 路径即目标，body 里不再有 id |
| E4 | **不修**：别名表是新功能。页面地址照旧按 `host`，写面与引用改用 id，至少不会再「写到别的站上」。死链接照实记下 |
| E5 | `PATCH` 只收改过的字段 |
| E6 | **不加闸**，与话题详情同一口径：详情照发，`is_nsfw` 由客户端决定怎么展示（网页照旧对 NSFW 站关 SEO）。列表才是「默认不含 NSFW」的地方 |
| E7 | 分类的 `website_count` 明说含 NSFW；页面计数改用列表 `total` |
| E8/E9/E13 | 具名 problem：404、422 `UNKNOWN_REFERENCE`、409 `ALREADY_EXISTS` / `WEBSITE_CATEGORY_NOT_EMPTY`；仓储错误一律 500，不再冒充 404 |
| E10 | K16 槽位 |
| E11 | 每个集合都有带 id 决胜键的固定排序 |
| E12 | 全部搬进服务端（§3.7）；标签 `slug` 也有了字符集 |
| E14 | **不在本轨修**：reference-ping 是 cron（`infrastructure/cron`），属基础设施；记下，另开 |
| E15 | **不在本轨修**：删号清理属 U 轨/管理面；记下 |
| E16 | **不在本轨修**：记下 |
| E18 | 网页按标签表现算上限（各组最高 `level` 之和），不再写死 245 |
| E20 | 详情不发 `comment` |
| E22 | 标签走 `WebsiteTag` |

## 5. 预分配

- 迁移：**175** `website_v1_cleanup`——`icon` 与 `create_time` 去首尾空白；`domain` 里没有 scheme 的元素补 `https://`；`trg_feed_galgame_website` 重建为 `AFTER INSERT OR DELETE OR UPDATE OF name, url, age_limit, user_id, created`。全部幂等，随部署跑。
- 错误码：新增 **`WEBSITE_CATEGORY_NOT_EMPTY`**（kungal，409，扩展 `website_count: integer`）。三处一译。其余复用：`NOT_FOUND`、`PERMISSION_REQUIRED`、`ACCOUNT_BANNED`、`SERVICE_UNAVAILABLE`、`VALIDATION_FAILED`（`INVALID_FORMAT` / `UNKNOWN_REFERENCE` / `INCONSISTENT_WITH` / `DUPLICATE_ITEM` / `TOO_SHORT` / `TOO_MANY_ITEMS`）、`ALREADY_EXISTS`、`INVALID_CURSOR`、`LIMIT_TOO_LARGE`。
- 权限：不新增。

## 6. 网页

- 列表（`Container.vue`）翻完全部页（`limit` 100），本地按分类分组、按 `score` 排；`include_nsfw` 取读者的立场。
- 详情页一次 `GET /websites/{host}`；JSON-LD 的评论另取 `wall-comments` 第一页（仅 SFW 站，与旧面相同）；赞走 `PUT`/`DELETE`，收藏走 `FavoriteToggle` 的 `action`。
- 编辑：先 `GET /admin/websites/{id}` 取编辑源，`PATCH` 只发改过的字段；删除后回 `/website`。
- 分类页、标签页：先取分类/标签，再取网站列表（`include_total`）。
- 后台 `admin/website/*` 与 `website/modal/*`：全部换类型化客户端；删分类遇到 `WEBSITE_CATEGORY_NOT_EMPTY` 显示译文。
- 手写类型 `shared/types/website.ts` 删掉或改成生成物别名；`legacy-fetch-baseline` 按删掉的调用点下调。

## 7. 变异题（先于实现提交）

种子：7 个网站同一 `created` 瞬间，跨页边界；含 NSFW 站；一个单选分组下至少两个标签。

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 赞的 `PUT` 当切换用 | 连续两次 `PUT` 后 `has_liked` 仍为真、`like_count` 只加 1 |
| 2 | 计数不管有没有真插入都 `+1` | `PUT` 重放后 `like_count` 不变，且等于 SQL 行数 |
| 3 | 没赞过时 `DELETE` 也 `-1` | `like_count` 不变且与行数一致 |
| 4 | 计数与浏览用会刷 `updated` 的更新 | 赞、收藏、浏览之后 `updated_at` 不变 |
| 5 | 网站列表排序去掉 `id` 决胜键 | 小 `limit` 全量遍历与 SQL 逐条相等、无重无漏 |
| 6 | 忽略 `include_nsfw`（缺席也返回 NSFW 站） | 默认列表里没有 `is_nsfw` 为真的条目，`total` 与之一致 |
| 7 | `total` 不带 `website_category_id` 过滤 | `total` = SQL 里该分类（按 NSFW 谓词）的行数 |
| 8 | 管理写面不查 `website.*` | 普通用户与 Bearer 版主 → 403，库里无变化 |
| 9 | 单选分组互斥不查 | 同组两个标签 → 422 `INCONSISTENT_WITH` |
| 10 | 不存在的分类 id 直接写库 | → 422 `UNKNOWN_REFERENCE`，不是 500 |
| 11 | `host` 不转小写 | 建 `WWW.Example.COM` 后按 `www.example.com` 读得到 |
| 12 | 删分类不先查网站数 | → 409 `WEBSITE_CATEGORY_NOT_EMPTY`，不是 500 |
| 13 | `PATCH` 缺席 `website_tag_ids` 时清空标签 | 只改 `title` 后标签原样 |
| 14 | 改/删不存在的 id 回成功 | → 404 |
| 15 | 写面不按 OAuth 当前记录判封禁 | 会话未封、OAuth 已封的用户点赞 → 403 `ACCOUNT_BANNED` |

## 8. 实现与验收（2026-09-23，只增不改）

实现时对契约的补充：

1. **`TagCreate.website_tag_group_id` 也是可空的**：G8 要求它与 `TagPatch`（`null` 表示移出分组）同型，所以建标签时 `null` 与缺席都表示无分组。
2. 管理面的 `POST` 都支持可选的 `Idempotency-Key`（K12「全部 POST 必须支持」）。
3. `urls` 的元素是带 `format: uri`、`maxLength: 100` 的具名类型；网页表单要求每条都带 `http(s)://`。
4. 删掉了已无引用的模型 `GalgameWebsiteLike` / `GalgameWebsiteFavorite` / `GalgameWebsiteTagRelation`（v1 用 SQL 直写这三张表）。连同旧 handler、service、dto、四个 repo 一起，`internal/website` 只剩模型、`repository/v1_*.go` 与 `apiv1/`。
5. 网页的「价值精算值」上限按标签表现算：单选分组取组内最高 `level`，其余正 `level` 全加。开发库数据算出 **275**，与普查从生产推出的理论上限一致（旧界面写死 245）。
6. `Website.updated_at` 的说明写明：2026-09-23 之前的值可能是浏览时间（E1 的历史值找不回来）。

### 8.1 变异（15 条，4 拆成浏览与计数两处，17 次全杀）

| # | 红的测试 |
|---|---|
| 1 | `TestV1WebsiteSlots`：第二次 `PUT` 后 `has_liked=false`、计数回 0 |
| 2 | `TestV1WebsiteSlots`：重放 `PUT` 计数变 2 |
| 3 | `TestV1WebsiteSlots`：没赞过的 `DELETE` 把计数减到 0 |
| 4a / 4b | `TestV1WebsiteDetail` / `TestV1WebsiteSlots`：浏览、点赞之后 `updated` 被刷成现在 |
| 5 | `TestV1WebsitesWalk`：去掉 `id` 后翻页重复 `…804`、漏掉 5 行 |
| 6 | `TestV1WebsitesFilterAndTotal`：默认列表里出现 NSFW 站 |
| 7 | `TestV1WebsitesFilterAndTotal`：分类 `total 6, want 3` |
| 8 | `TestV1WebsiteAdminNeedsPermissions`：普通用户建站 201 |
| 9 | `TestV1AdminWebsiteCreate`：单选分组两个标签建成 201 |
| 10 | `TestV1AdminWebsiteCreate`：不存在的分类 → 500（外键）而不是 422 |
| 11 | `TestV1AdminWebsiteCreate`：大写 host 让 201 响应不符合 spec 的 `pattern` |
| 12 | `TestV1WebsiteCategories`：删有网站的分类 → 500（外键 RESTRICT） |
| 13 | `TestV1AdminWebsitePatch`：只改标题后标签被清空 |
| 14 | `TestV1AdminWebsiteDelete`：再删一次回 204 |
| 15 | `TestV1WebsiteAdminNeedsPermissions`、`TestV1WebsiteSlots`：OAuth 里已封禁的会话照样能写 |

### 8.2 浏览器（开发栈，本 worktree 的 API 打本轨临时库，开发库 84 个站的数据导进来并按迁移 175 修过）

- 匿名：`/website` 65 张卡（SFW），四个分类加「已关站」分组，点卡进详情；详情页无编辑/删除；评论墙照常出现。
- 普通用户：点赞 6→7、收藏 2→3，刷新后仍是已赞/已收藏，再点回 6 / 2；库里计数与行数一致，`updated` 没动。
- 分类页 `/website-category/resource` 41 个站（与列表 `total` 一致），标签页 `/website-tag/performance2` 25 个站；详情页价值条显示 `156 / 275`。
- admin：API 建一个站后在详情页点编辑，表单从 `GET /admin/websites/{id}` 预填；改标题并把主域名改成新的 → 保存后跳到新地址，标题已变；删除 → 回 `/website`，旧地址 404。
- 后台 `/admin/website`：4 个非空分类的删除按钮是禁用的；建、改、删一个空分类；用已存在的标识建分类 → 界面显示「已存在」（`ALREADY_EXISTS` 的译文）；建分组、建标签。
- 控制台只有与本轨无关的 `me/preferences` 503（假令牌）与顶栏头像水合不一致。

## 9. 后续修复（2026-09-23，用户拍板「全部按推荐做」，只增不改）

§4 里记下没修的三条，上线当天补上：

| 编号 | 修法 |
|---|---|
| E14 | reference-ping 改为**按列名发现**裸 hash 列：文本列名以 `image_hash` 结尾、或 jsonb 列名以 `image_hashes` 结尾的都会被 ping。现有的是 `doc_article.banner_image_hash`、`friend_link.banner_image_hash`、`galgame_website.icon_image_hash`、`topic_lottery_prize.image_hashes`；前三列此前**从没被 ping 过**（会先进冷存储、365 天后软删）。`TestEveryHashColumnIsPingedOrExempt` 扫全库列名含 `hash` 的列，既没被发现、又不在带理由的豁免表（`topic_lottery.seed_hash`、`topic_lottery_prize.nsfw_hashes`）里就红——以后加 `*_hash` 列要么按约定命名，要么写明为什么不是图片 |
| E15 | 删号清理不再 `DELETE` 被清用户收录的网站，改为 `user_id = DEFAULT`（归还给导入数据的占位用户，87 行里本来就有 83 行是它）。别人的赞、收藏、标签关系都保留；被清用户自己的赞/收藏照旧删并重算计数。后台删号预览里这一项改叫「收录网站 (转交保留)」 |
| E4 | **迁移 176** 把 8 条动态卡片、5 条通知的 `/website/<旧主机名>` 改写成网站的当前主机名。每个旧主机名都用 community 库里评论所在的墙核实过归属（见迁移头注释）。另外，`PATCH /admin/websites/{id}` 改 `host` 时在同一事务里改写 `feed_activity`（两种网站动态）与 `message` 里的 `/website/<旧>`、`/website/<旧>?…`、`/website/<旧>#…`，以后改名不会再留死链 |

E16（删网站留下的评论动态与上游孤儿串）不在这一批。
