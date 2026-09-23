# D · 文档

> 文档轨，2026-09-23，分支 `api-v1/d-doc`，迁移号段 165–169。
> 契约与变异题同一个提交，早于实现。

## 0. 轨的范围更正

看板原写「D = 文档 `/doc` + `/website-tag` + `/website-category`，共用 `TagHandler` / `CategoryHandler`」。**这是类型同名造成的误判**：`internal/doc/handler.TagHandler` 与 `internal/website/handler.TagHandler` 是两个包里互不相干的类型，`internal/doc/**` 除 `app.go` 外没有任何导入方。

按看板自己的切分规则（按共用的 handler / service 连通分量切），网站标签与分类属于网站那一坨：`WebsiteService` 持有 `categoryRepo` + `tagRepo`，`CategoryService` / `TagService` 持有 `websiteRepo`，标签/分类详情里嵌网站卡片，网站详情里嵌标签/分类摘要。

所以：**D = 文档 16 条**；`/website-tag` 与 `/website-category` 的 10 条并入 **WS**（11 → 21）。已当面通知新开的 WS 会话，看板同一提交里改正。

## 1. 生产取值（2026-09-23，只读查询）

| 事实 | 数字 |
|---|---|
| `doc_article` | **29** 行，`status` 全部为 `1`（`0` 草稿、`2` 隐藏/归档一行都没有，历史上也没有） |
| 置顶 `is_pin` | 9 |
| `doc_category` | 4 行：`galgame` / `notice` / `kun` / `other`，`description` 与 `icon` 全空，建表后从未增删；**分类写路由在网页没有任何调用方** |
| `doc_tag` / `doc_article_tag_relation` | **0 / 0**，从未有过一行 |
| 横幅 | 29 行全是根相对静态路径 `/content/<分类>/<slug>/banner.avif`，**`banner_image_hash` 0 行** |
| `path` | 29 行全等于 `'/doc/' \|\| slug`，是冗余派生列 |
| `author_id` | 全部是 `2` |
| `sort_order` | 0–28，无重复 |
| `edited_time` | 29 行非空 |
| `updated` | 最近一天里全部被刷新过——**是浏览计数刷的**（见 §5 #4），不是编辑时间 |
| 正文 | 最长 8868 字符；原始 HTML 只有 `<br>` 24 处、`<img>` 1 处、`<video>` 1 处；`/image/<hash>` token 8 篇、Markdown 图片 11 处；有 `$` 的 1 篇；每篇都有标题 |
| slug | 全部匹配 `^[a-z0-9-]+$`，最长 24 |

## 2. 调用方

| 旧路由 | 网页 | Nitro `server/` | Flutter App |
|---|---|---|---|
| `GET /doc/article` | `components/doc/Container.vue`（首页 24 条）、`doc/detail/CategoryTree.vue`、`doc/detail/Footer.vue`（上一篇/下一篇）、`home/Carousel.vue`（置顶轮播） | 无 | 无 |
| `GET /doc/article/:slug` | `pages/doc/[...slug].vue`；`pages/admin/doc.vue` 用它取编辑源 | 无 | 无 |
| `GET /doc/category` | `doc/detail/CategoryTree.vue`、`edit/doc/Layout.vue` | 无 | 无 |
| `GET /doc/tag` | `edit/doc/Layout.vue` | 无 | 无 |
| `GET /admin/doc/article` | `pages/admin/doc.vue` | 无 | 无 |
| `POST` / `PUT /doc/article` | `edit/doc/Layout.vue` | 无 | 无 |
| `PUT /doc/article/reorder`、`/pin`、`DELETE /doc/article` | `pages/admin/doc.vue` | 无 | 无 |
| `POST /doc/tag` | `edit/doc/MetadataForm.vue` | 无 | 无 |
| `PUT /doc/tag`、`DELETE /doc/tag`、分类三写 | **无** | 无 | 无 |

`../kungal-apps` 没有一处引用；`docs/proj/app-direct-api.md` 没有列文档。

## 3. 普查抓到的旧面问题

| # | 问题 | 去向 |
|---|---|---|
| 1 | **草稿匿名可读**：`GET /doc/article/:slug` 不查 `status`；公开列表接受 `?status=0` | 状态整个不建模（K26），问题随之消失 |
| 2 | 文档首页只取 `limit: 24`，而文档有 29 篇：**5 篇从首页消失** | 网页翻完所有页 |
| 3 | 列表、分类、标签的 `Count` / `Find` 错误全部丢弃，失败时回空列表 | v1 一律 500 |
| 4 | `IncrementView` 走 GORM `Update`，**每次浏览都刷新 `updated`**，所以 `updated` 实际上是「最后被看的时间」 | v1 用 `UpdateColumn`；不下发 `created_at` / `updated_at` |
| 5 | 浏览计数在无上下文的 goroutine 里做，错误丢弃 | 同步做，失败只 warn、不影响读 |
| 6 | 创建时 `description` 校验上限 1000，列是 `varchar(777)`：778–1000 字符在库里失败成 500 | schema `maxLength: 777` |
| 7 | slug 撞车 → 500「创建文章失败」 | `409 ALREADY_EXISTS` |
| 8 | 更新、删除不存在的 id 静默成功 | `404 NOT_FOUND` |
| 9 | 重排收任意 id 列表：漏掉的文档保留旧 `sort_order`，与新序号撞车 | 必须恰好是全部文档的一个排列（§6.5） |
| 10 | 新文档 `sort_order` 默认 0，与第一篇并列 | 新文档排在最后（`max + 1`） |
| 11 | 标签列表 `ORDER BY title` 无决胜键；关键词 `ILIKE` 不转义通配符 | 标签不再建模；v1 列表不提供关键词 |
| 12 | 状态标签两处不一致：编辑器 `2 = 隐藏`，管理表 `2 = 已归档` | 状态不建模 |
| 13 | 编辑页取编辑源走公开详情，靠的正是 #1 那个洞 | 编辑源改由 `GET /admin/docs/{doc_id}` 提供 |
| 14 | 横幅永远是旧静态路径：`backfill-cover-hashes` 2026-06-28 跑过一次，29 张 **0 张成功**，因为图床只解 gif / jpeg / png / webp，**不收 AVIF**（`code=80009`） | §8 |

## 4. 逐条去向

| # | 旧路由 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /doc/article` | `GET /api/v1/docs` | public |
| 2 | `GET /admin/doc/article` | 并入 `GET /api/v1/docs`（已无状态可分） | public |
| 3 | `GET /doc/article/:slug` | `GET /api/v1/docs/{doc_slug}` | public |
| 4 | — | `GET /api/v1/admin/docs/{doc_id}`（编辑源；G17 要求它存在） | required，`doc.edit` |
| 5 | `POST /doc/article` | `POST /api/v1/admin/docs` | required，`doc.create` |
| 6 | `PUT /doc/article` | `PATCH /api/v1/admin/docs/{doc_id}` | required，`doc.edit` |
| 7 | `PUT /doc/article/pin` | 并入 `PATCH`（`is_pinned`） | required，`doc.edit` |
| 8 | `PUT /doc/article/reorder` | `PUT /api/v1/admin/doc-order` | required，`doc.edit` |
| 9 | `DELETE /doc/article` | `DELETE /api/v1/admin/docs/{doc_id}`，**204** | required，`doc.delete` |
| 10–12 | `GET` / `POST` / `PUT` / `DELETE /doc/category`（4 条） | **删除，无替代**：分类升格为封闭枚举（K25） | — |
| 13–16 | `GET` / `POST` / `PUT` / `DELETE /doc/tag`（4 条） | **删除，无替代**：文档标签整条移除（§7） | — |

16 条旧路由 → 7 个 v1 操作。16 条在本 PR 全删，`legacy_route_baseline` 下调 **16**。全部挂 `internal/doc/apiv1/`，tag `docs`。

权限判定一律 `user.Can`，Bearer 请求永远不持有 `doc.*`。版主与管理员都持有三个 `doc.*`（`perm.Bundles`）。匿名 → 401，缺权限 → `403 PERMISSION_REQUIRED`。

**为什么公开面按 slug、管理面按 id**：页面 URL 是 `/doc/<slug>`，公开读只能按 slug 寻址；slug 可改，写面按不可变的 id 寻址。两者不能共用 `/docs/{x}`（OpenAPI 不允许同层级模板路径只差参数名），而 G17 要求每个写路径有同路径的 GET。于是管理面是 `/admin/docs/{doc_id}`，发 `admin_doc`——与 T4 的 `/admin/topics/{topic_id}` → `admin_topic` 同一个模式：**路径前缀跟着表示走，不跟着权限走**。

## 5. 新 K 决定

**K25 · 管理员维护、但从未变过、客户端早已按 token 本地化的词表，升格为封闭枚举。** 01 §7 要求「属于数据的词表」二选一。文档分类选封闭枚举 `doc_category` ∈ `galgame` / `notice` / `kun` / `other`：

- 4 行自建表起从未增删，`description` / `icon` 全空；分类写路由在网页零调用方；
- 网页早就按 slug 查 `KUN_DOC_CATEGORY_MAP` / `KUN_DOC_CATEGORY_COLOR_MAP` 出标签和颜色——这正是封闭枚举的客户端形态；
- 存储不变：`doc_category` 表留着，读时按 `slug` 映射、写时按 `slug` 反查 `id`。库里出现第五个 slug → **500**，不许静默归入其中一个（U1 §3.10 同一条规矩）。
- 字段叫 `doc_category` 而不是 `category`：`Topic.category` 已经是封闭枚举 `galgame,technique,others`，G8 要求同名同枚举。

**K26 · 没有一行的生命周期不建模。** 文档的 `status`（0/1/2）生产上 29 行全是 `1`、历史上没有别的值 → v1 的文档没有 `state`，全部公开。列留着（新建写默认值 `1`），v1 不读。将来要草稿，加一个 `state` 是加法。同理文档标签（0 行）整条删除（§7）。

K 号：U 轨分支已用到 K24，本轨从 K25 起。合并时若撞号，后合者改号。

## 6. 形状

### 6.1 `DocSummary`（`object: "doc"`）与 `Doc`（`object: "doc"`）

```
DocSummary  object="doc", id, slug, title, description, doc_category,
            banner: Image|null, is_pinned, view_count, published_at, edited_at|null
Doc         DocSummary 的全部字段 + author: UserRef + content: ContentDocument
```

| 字段 | 类型 | 来源与语义 |
|---|---|---|
| `slug` | string，`^[a-z0-9]+(?:-[a-z0-9]+)*$`，≤128 | 页面地址是 `/doc/{slug}`，客户端自己拼；旧的 `path` 列不下发 |
| `title` | string ≤233 | 原文 |
| `description` | string ≤777 | 原文，可为空串 |
| `doc_category` | 封闭枚举（K25） | |
| `banner` | `Image` \| null | **只**来自 `banner_image_hash`；没有 hash → `null`。旧静态路径不下发（`Image.hash` 必填，B10）。宽高 / thumbhash 走 `ImageMeta` |
| `is_pinned` | bool | 旧名 `is_pin`。首页轮播只放置顶 |
| `view_count` | int ≥0 | 旧名 `view` |
| `published_at` | date-time | 旧名 `published_time`，创建时写入，之后不变 |
| `edited_at` | date-time \| null | 旧名 `edited_time`，规则见 §6.4 |
| `author` | `UserRef` | 仅 `Doc`。查不到按 K17 读面规则：`name: null`，不整条失败 |
| `content` | `ContentDocument` | 仅 `Doc`。完整 Markdown 管线（K13，03 文档），目录由客户端从 `heading` 节点算，**不再**下发 `content_html` / `toc` |

不下发：`created_at` / `updated_at`（前者是 2025-12-17 的导入时间，后者被浏览计数污染，见 §3 #4）、`path`、`status`、`sort_order`、`author_id`、`banner` 旧路径、`banner_url`、`tag_ids`、`category_id`。

### 6.2 `GET /api/v1/docs` → `List[DocSummary]`（游标）

- 参数：`cursor`、`limit`（1–100，默认 20）、`sort`、`doc_category`（封闭枚举，缺席 = 全部）、`is_pinned`（`true` / `false`，缺席 = 不过滤）。
- `sort`：
  | token | 顺序 |
  |---|---|
  | `position_asc`（默认） | `sort_order ASC, id ASC`：管理员拖出来的展示顺序 |
  | `published_desc` | `published_time DESC, id DESC` |
  | `views_desc` | `view DESC, id DESC` |
- 未知 `sort` → `400 UNKNOWN_SORT`；未知 `doc_category` → `400 UNKNOWN_ENUM_VALUE`；`limit > 100` → `400 LIMIT_TOO_LARGE`；游标坏了或换了过滤条件 → `400 INVALID_CURSOR`（指纹绑定 `sort`、`doc_category`、`is_pinned`）。
- 不接受 `include_total`，不发 `total`（F9）。旧的 `keyword`、`tag_id`、`status`、`order_by`、`sort_order` 不再存在（未知参数按 infra 02 §4 忽略）。

### 6.3 `GET /api/v1/docs/{doc_slug}` → `Doc`

- `doc_slug`：同 `slug` 的 pattern 与长度，格式不符 → `400 INVALID_PARAMETER`；没有这个 slug → `404 NOT_FOUND`。
- **每次成功读计一次浏览**：`UPDATE doc_article SET view = view + 1 … RETURNING view`，走 `UpdateColumn`，**不碰 `updated`**。响应里的 `view_count` 含本次。计数失败只打 warn，照常返回（按库里的旧值）。

### 6.4 管理面 `AdminDoc`（`object: "admin_doc"`）

```
AdminDoc  object="admin_doc", id, slug, title, description, doc_category,
          banner: Image|null, is_pinned, view_count, published_at, edited_at|null,
          content_markdown
```

- `GET /api/v1/admin/docs/{doc_id}`：编辑器的数据源。`doc.edit`。
- `POST /api/v1/admin/docs` → **201** + `Location: /api/v1/admin/docs/{id}` + `AdminDoc`。`doc.create`。`Idempotency-Key` 可选（K12「必须支持」）。
- `PATCH /api/v1/admin/docs/{doc_id}` → 200 + `AdminDoc`。`doc.edit`。
- `DELETE /api/v1/admin/docs/{doc_id}` → **204**。`doc.delete`。

请求体：

```
DocCreate  slug (必填), title (必填), doc_category (必填), content_markdown (必填),
           description? (默认 ""), banner_image_hash? (默认 ""), is_pinned? (默认 false)
DocPatch   以上全部可选；至少一个字段 → 否则 422 VALIDATION_FAILED（pointer ""，REQUIRED）
```

| 字段 | 约束 |
|---|---|
| `slug` | `^[a-z0-9]+(?:-[a-z0-9]+)*$`，1–128。撞车 → **`409 ALREADY_EXISTS`**（`doc_article.slug` 与 `path` 两个唯一索引，任一撞上都是它） |
| `title` | 1–233（K19：长度作用于原始值），去首尾空白后存；只有空白 → `422 TOO_SHORT` |
| `description` | 0–777，去首尾空白后存 |
| `doc_category` | 封闭枚举（K25） |
| `content_markdown` | 1–100000，按 `markdown.NormalizeStoredContent` 存；只有空白 → `422 TOO_SHORT` |
| `banner_image_hash` | `^([0-9a-f]{64})?$`。**空串 = 没有横幅**（`PATCH` 里发空串 = 去掉横幅）。不用 `null` 表达清除：huma 的 null 分支会关掉该字段的 pattern 校验（G9 的理由） |
| `is_pinned` | bool |

- `path` 列（`NOT NULL UNIQUE`）照旧写 `'/doc/' + slug`，改 slug 时一起改。`status` 列不写（默认 1）。
- **新文档排在最后**：`sort_order = COALESCE(MAX(sort_order), -1) + 1`，与插入同一事务。
- `published_at` 创建时写入 `now()`，之后不变。
- **`edited_at` 只在「非置顶字段的值确实变了」时刷新**：`PATCH` 里 `slug` / `title` / `description` / `doc_category` / `content_markdown` / `banner_image_hash` 任一与库里不同 → `edited_at = now()`；只改 `is_pinned`、或发来的值与库里相同 → 不动。旧 `PUT` 每次都刷，于是「置顶一下」就变成「最后编辑于今天」。新建时 `edited_at` 为 `null`。
- 不存在 → `404 NOT_FOUND`。

### 6.5 `PUT /api/v1/admin/doc-order` → **204**

```
请求  { doc_ids: [id] }   minItems 1，maxItems 1000，uniqueItems
```

- 语义：用这个列表**整体替换**展示顺序，`sort_order` 依次写 0…n-1。
- 必须**恰好**是全部文档的一个排列，在一个事务里对 `doc_article` 全表加行锁后比对：
  - 有不存在的 id → `422 VALIDATION_FAILED`，`pointer /doc_ids/<i>`，`reason UNKNOWN_REFERENCE`；
  - 少了文档 → `422 VALIDATION_FAILED`，`pointer /doc_ids`，`reason TOO_FEW_ITEMS`，`params.min_items = <文档总数>`；
  - 重复 → schema `uniqueItems` 拦下，422。
- 网页拖完一次发全量；列表过期（别人刚建了一篇）就会 422，网页据此重拉。
- 路径不叫 `/admin/docs/order`：避免与 `/admin/docs/{doc_id}` 字面量 / 模板重叠。

### 6.6 错误码

**无新增。** 复用 `NOT_FOUND`、`PERMISSION_REQUIRED`、`ALREADY_EXISTS`（U1 已进注册表）、`VALIDATION_FAILED`、`INVALID_PARAMETER`、`UNKNOWN_ENUM_VALUE`、`UNKNOWN_SORT`、`INVALID_CURSOR`、`LIMIT_TOO_LARGE`，以及档位与幂等键自带的那些。

## 7. 文档标签整条移除

K26：0 行、从未被用过、没有任何公开展示（只有编辑器里能建能选）。删掉：

- 4 条 `/doc/tag` 路由、`tag_id` 过滤、文章的 `tag_ids`；
- 编辑器 `MetadataForm.vue` 的整个标签区（选择 + 就地新建）；
- 模型 `DocTag`、`DocArticleTagRelation`；
- 表 `doc_tag`、`doc_article_tag_relation`：**迁移 165，deploy-then-drop**。旧二进制在文章详情里读 `doc_article_tag_relation`，所以 165 进 `cmd/migrate` 的默认 exclude，部署完成后手动 `--only=165`，确认在所有库上跑过之后再从 exclude 里拿出来（同 049 / 102）。只 DROP 表、不动列，所以**不需要**重启 API。

## 8. 横幅回填（合并前的前置步骤）

v1 的 `banner` 只认 hash，而生产 29 张横幅一张 hash 都没有。不先回填，v1 一上线文档横幅就全变成占位图。

回填本身与 v1 无关、可以先做：旧读面 `resolveBannerURL` 本来就是「有 hash 用 hash」，回填一跑完线上就改从图床出图。做法：

1. 本地把 `apps/web/public/content/**/banner.avif` 29 个文件解码成 PNG（无损；图床收下后自己转 WebP）。
2. 传到生产机，挂进 tools 容器、仍以 `banner.avif` 为名放在 `/app/content/<同路径>`，跑 `backfill-cover-hashes -webroot /app`。回填命令按路径读文件、按内容嗅探格式，库里的 `banner` 列不用改；它对三张表都是幂等的（只碰「有 URL、无 hash」的行），`friend_link` 早已全部完成，`galgame_website` 剩下的 `.ico` 与死链会照旧失败跳过。
3. 核对 29 行 `banner_image_hash` 非空，线上文档页横幅来自图床。

**这是生产写操作，执行前单独征得用户同意。**

## 9. 网页

| 文件 | 改什么 |
|---|---|
| `pages/doc/[...slug].vue` | `GET /docs/{doc_slug}`；正文改用 `<ContentDocument>` 渲染（不再 `v-html` + `renderKatex`），目录从 `heading` 节点算（复用话题的 TOC 逻辑）；SEO 的作者链接用 `author.id`、og 图用 `banner?.url` |
| `components/doc/Container.vue` | 翻完所有页（修 §3 #2），`position_asc` |
| `components/doc/detail/CategoryTree.vue` | 分类来自枚举 + `KUN_DOC_CATEGORY_MAP`，不再请求分类接口；文章按 `doc_category` 分组 |
| `components/doc/detail/Footer.vue`、`Header.vue` | 字段改名 |
| `components/home/Carousel.vue` | `GET /docs?is_pinned=true&sort=views_desc&limit=10` |
| `pages/admin/doc.vue` | 列表 `GET /docs`；拖拽 → `PUT /admin/doc-order`（422 → 重拉）；置顶 → `PATCH`；删除 → `DELETE`；编辑 → `GET /admin/docs/{doc_id}`；去掉状态标签 |
| `components/edit/doc/*` | `POST /admin/docs` / `PATCH /admin/docs/{doc_id}`；分类下拉用枚举；删状态下拉、删标签区；横幅 `banner_image_hash`，清除发空串；slug 输入 `maxlength` 128 |
| `shared/types/doc.ts`、`components/*/doc/type.d.ts` | 删掉或改成生成物的别名 |
| `constants/doc.ts` | 删 `KUN_DOC_STATUS_OPTIONS`；两张分类表按枚举类型收紧 |

`legacy-fetch-baseline` 按删掉的 `kunFetch` / `useKunFetch` 调用点下调。

## 10. 删旧

16 条路由；`internal/doc/{handler,dto,service}` 全删，`repository` 只留 v1 用到的；`model` 删 `DocTag`、`DocArticleTagRelation`；`app.go` 三个字段；网页手写类型。`rg` 证明零调用方（含 `apps/web/server/`、`../kungal-apps`）；`deadcode -test ./...` 跑到不动点；`routes.golden` 重生成，`legacy_route_baseline` −16。

`cmd/backfill-cover-hashes`、`cmd/backfill-content-images`、`cmd/rewrite-content-image-refs`、`cmd/purge-staging-verify`、`internal/galgame/renumber` 按表名读写 `doc_article`，不经过 `internal/doc`，不受影响。

## 11. 变异题（先于实现提交）

每条一行语义改动，每条都必须让某个测试变红。编译不过的换等价破坏。

| # | 破坏 | 必须变红的断言 |
|---|---|---|
| 1 | `position_asc` 去掉 `id` 决胜键 | 小 `limit` 翻完所有页（**种子里有跨页边界的并列 `sort_order`**）与 SQL 逐条相等、无重无漏 |
| 2 | `views_desc` 去掉 `id` 决胜键 | 同上（种子里有并列 `view`） |
| 3 | 游标指纹去掉 `doc_category` | 换 `doc_category` 复用游标 → `400 INVALID_CURSOR` |
| 4 | `is_pinned` 过滤不生效 | `is_pinned=true` 只返回置顶文档 |
| 5 | 详情不计浏览（或计数写进 `updated`） | 连读两次 `view_count` +1、+1，且 `updated` 列不变 |
| 6 | 创建不查 `doc.create` | 普通用户 → `403 PERMISSION_REQUIRED` |
| 7 | 删除查 `doc.edit` 而不是 `doc.delete` | 版主经用户级覆盖（`perm.SetUserOverrides`）撤掉 `doc.delete`、保留 `doc.edit`：能改、删除 → 403 |
| 8 | slug 唯一冲突映射成 500 | 撞 slug → `409 ALREADY_EXISTS` |
| 9 | 新文档 `sort_order` 写 0 | 新建后在 `position_asc` 里排最后 |
| 10 | `PATCH` 只改 `is_pinned` 也刷新 `edited_at` | 只改置顶 → `edited_at` 不变；改标题 → 变 |
| 11 | 重排不校验「全部文档」 | 少一篇 → `422 TOO_FEW_ITEMS`，`params.min_items` = 文档总数，库里顺序不变 |
| 12 | 重排不校验 id 存在 | 含不存在的 id → `422 UNKNOWN_REFERENCE`，`pointer /doc_ids/<i>` |
| 13 | 删除 / `PATCH` 不存在的 id 静默成功 | → `404 NOT_FOUND` |
| 14 | 分类映射把未知 slug 静默归入 `other` | 库里插第五个分类并挂一篇 → 读面 500 |
| 15 | Bearer 请求拿到管理能力（`perm.CanUser` 代替 `user.Can`） | Bearer 的版主 → 403（行为测试杀不掉时由 `bearer_guard_test` 杀，见 T4 §8 #6） |

## 12. 迁移

| 号 | 内容 | 类型 |
|---|---|---|
| 165 | `DROP TABLE doc_article_tag_relation; DROP TABLE doc_tag;` | **deploy-then-drop**：进默认 exclude，部署后手动 `--only=165` |

其余号段 166–169 未用。

## 13. 实现时对本契约的修正（2026-09-23，只增不改）

1. **加了迁移 166**：幂等地补齐四个 `doc_category` 行（`ON CONFLICT (slug) DO NOTHING`）。生产的 4 行来自 2025-12 的数据导入，空库里一行都没有，于是 K25 的枚举在新库上无处落地、建不了文档。生产上是空操作，随部署自动跑。
2. **`is_pinned` 是三态查询参数**。huma 不支持指针型 query 参数（注册时 panic），缺席又必须和 `false` 区分开，于是在 `internal/doc/apiv1` 里写了一个 `ParamWrapper` + `ParamReactor`（`optionalBool`），schema 仍是 `boolean`，所以 `strictBooleans` 照样只收 `true` / `false`。没有改共享的 `internal/apiv1`。
3. **`AdminDoc.content_markdown` 的 `maxLength` 是 100007**（列宽），请求体仍是 100000：响应要能装下库里任何已有的值。
4. **作者被封禁时 `author` 按不可渲染处理**（`name: null`），文档本身照常读：文档是站方内容，不随作者封禁消失。上游 `/users/batch` 失败仍是 503（与话题读面一致）。
5. **变异 13b（`PATCH` 不存在的 id 静默成功）是等价变异**：`Update` 之后还要 `loadAdminDoc` 重读，缺席照样 404。于是删掉了仓储层多余的 `found` 返回值，只留重读这一个判据，这道题也就不再适用。
6. **变异 15 由静态守卫杀掉**，行为测试杀不掉：Bearer 夹具的版主角色在 `SiteRoles` 里，`perm.CanUser(u.ID, u.Roles, …)` 照样判不出管理能力（与 T4 §8 #6 同理）。`internal/middleware/bearer_guard_test.go` 扫到 `perm.CanUser(` 就红。

7. **rebase 之后在最终代码上重跑了全部变异，全杀。** 管理面挪进 `admin.go` 之后，`service.go` 里只剩分类映射这一处用 `fmt`，变异 14 的原写法（删掉 `fmt.Errorf`）变成了未使用导入、编译不过——按规矩这是废题，换成「`if false` 包住原返回、再 `return "other", nil`」的等价破坏，照样被杀。另补了一个用例：账号服务不可用时 `getDoc` 回 `503`、列表不受影响（闸 7 对照时发现漏了）。
### 变异执行结果

| # | 结果 |
|---|---|
| 1 | 杀：`TestV1DocsWalkEverySort` |
| 2 | 杀：`TestV1DocsFilters`、`TestV1DocsWalkEverySort` |
| 3 | 杀：`TestV1DocsFilters` |
| 4 | 杀：`TestV1DocsFilters` |
| 5 | 杀（不计数、计数写 `updated` 两种破坏）：`TestV1DocGetCountsViewsWithoutTouchingUpdated` |
| 6 | 杀：`TestV1AdminDocFacesNeedTheirPermissions` |
| 7 | 杀：`TestV1AdminDocFacesNeedTheirPermissions` |
| 8 | 杀：`TestV1AdminDocCreate`、`TestV1AdminDocPatch` |
| 9 | 杀：`TestV1AdminDocCreate` |
| 10 | 杀：`TestV1AdminDocPatch` |
| 11 | 杀：`TestV1DocOrderReplacesTheWholeSequence` |
| 12 | 杀：`TestV1DocOrderReplacesTheWholeSequence` |
| 13 | 删除：杀（`TestV1AdminDocGetAndDelete`）；`PATCH`：等价变异，见上文 #5 |
| 14 | 杀：`TestV1DocUnknownCategoryIsNotFiledElsewhere` |
| 15 | 杀：`TestCapabilityChecksGoThroughUserInfo`（静态守卫） |
