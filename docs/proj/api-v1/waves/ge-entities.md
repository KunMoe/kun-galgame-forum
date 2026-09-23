# GE · galgame 实体六件套：标签、会社、引擎、系列、制作人员、角色

> 契约与变异题同一个提交，早于实现。普查基于 `api-v1/ge-entities` @ `9a7ffed4`（= master），生产数字实测于 2026-09-23。
> 共享面的三处约定已与 G 轨（kun-galgame-forum-9b）和 X2 轨（kun-galgame-forum-c8）谈定，见 §2.1。

## 0. 范围

`EntityHandler` 一个人管 18 条旧路由、6 个前缀，全是 `GET`、全是 catalog 的代理读：

| # | 旧路由 | handler | 调用方 |
|---|---|---|---|
| 1 | `GET /galgame-tag` | `GetTagList` | 标签页、sitemap（Nitro） |
| 2 | `GET /galgame-tag/search` | `SearchTags` | 标签页搜索框、编辑表单标签选择器 |
| 3 | `GET /galgame-tag/multi` | `GetMultiTagGalgames` | 标签页多选交集 |
| 4 | `GET /galgame-tag/:id` | `GetTagDetail` | 标签详情页 |
| 5 | `GET /galgame-official` | `GetOfficialList` | 会社列表页、sitemap（Nitro） |
| 6 | `GET /galgame-official/search` | `SearchOfficials` | 会社列表页搜索框、编辑表单 |
| 7 | `GET /galgame-official/legacy/:id` | `ResolveLegacyOfficial` | Nitro 中间件 `legacy-taxonomy.ts`（旧 wiki 会社 id 的 301） |
| 8 | `GET /galgame-official/:id` | `GetOfficialDetail` | 会社详情页两个子页、OG 卡（Nitro） |
| 9 | `GET /galgame-official/:id/relation-graph` | `GetOfficialRelationGraph` | 会社关系图 |
| 10 | `GET /galgame-engine` | `GetEngineList` | 引擎列表页、编辑表单（整表拉回本地过滤）、sitemap（Nitro） |
| 11 | `GET /galgame-engine/:id` | `GetEngineDetail` | 引擎详情页 |
| 12 | `GET /galgame-staff/search` | `SearchStaff` | 编辑表单 |
| 13 | `GET /galgame-staff/:id` | `GetStaffDetail` | 制作人员页（首屏 + 加载更多）、OG 卡（Nitro） |
| 14 | `GET /galgame-character/search` | `SearchCharacters` | 编辑表单 |
| 15 | `GET /galgame-character/:id` | `GetCharacterDetail` | 角色页（首屏 + 加载更多）、角色弹窗、OG 卡（Nitro） |
| 16 | `GET /galgame-series` | `GetSeriesList` | 编辑表单（整表 7250 行拉回本地过滤） |
| 17 | `GET /galgame-series/cards` | `GetSeriesCards` | 系列列表页、galgame 详情页的系列面板（`?ids=`） |
| 18 | `GET /galgame-series/:id` | `GetSeriesDetail` | 系列详情页 |

Flutter App（`../kungal-apps`）零调用。`docs/proj/app-direct-api.md` 没有这些路由。

**不在本轨**：`/search/entity*`（X 轨的站内实体搜索；它经 `EntitySearchService` 用 `TagService`，所以 `TagService` 在本轨之后仍然活着，只有它的三个旧读面方法随路由删除）；galgame 详情里嵌的标签/会社/系列引用（G 轨）。

## 1. 普查

### 1.1 生产数字（`kun_catalog` + 线上旧路由实测，2026-09-23）

| 事实 | 数字 |
|---|---|
| catalog 行数 | tag 3,470 · label（会社）50,220（其中已删 2,122）· engine 189 · series 7,250 · credit_name 135,671 · character 240,822 |
| 标签索引（`has_works=1`、去掉 hidden 层） | 全部 3,250：`content` 2,135 · `meta` 323 · 成人（`sexual`）792；SFW 读者看到 2,458 |
| 标签 `tier × kind × sexual` | 三个 tier（线上值 `core` 等，hidden 是其中之一）× 两个 kind（`content` 0 / `meta` 1）；**没有第三种 kind** |
| 会社索引（`has_works=1`） | 3,831：`game_brand` 1,379 · `doujin_circle` 1,378 · `publisher` 1,055 · `group` 19；`bunko` / `anime_studio` **0 行**（库里也是 0） |
| 会社列表条目的 `link` / `lang` | 3,831/3,831 都是空串——`officialRow` 从来没填（§1.3 E4） |
| 会社关系 | 8 种关系成对存储：`parent`/`subsidiary` 1,026 对 · `imprint`/`imprint_of` 1,418 对 · `succeeded_by`/`formerly` 606 对 · `spawned`/`origin` 325 对 |
| 引擎 | 189，其中 187 有作品 |
| 系列 | 7,250；catalog 作品数 >0 的 1,854；本站可列（有资源且已发布）≥1 部的：全部 1,304 · SFW 630 |
| 合并记录（`catalog_redirect`） | credit_name 28,921 · character 与另两族共 41,760 · company 2,122 · person 3；tag / engine / series **0** |
| `galgame_resource` 轴 | 50,735 行，jsonb `platforms` / `languages` **0 行为空**；`status` 0（正常）47,306 · 1（失效）3,429 |
| 资源平台轴取值 | `win` 43,974 · `and` 6,158 · `oth` 1,983 · `dvd` 398 · `swi` 178 · `ios` 174 · `psp` 126 · `mac` 44 · `lin` 37 · `psv` 27 · 其余 ≤4 |
| 资源语言轴取值 | `zh-cn` 45,384 · `ja-jp` 5,470 · `zh-tw` 191 · `other` 111 · `en-us` 63 |
| 评分 `galgame_type` | `plot` 2,156 · `daily` 1,403 · `ba_saku` 1,341 · `moe` 1,290（四值全有） |

### 1.2 现行语义

- **全部名字由服务端按偏好选一条下发**：`CatalogEntityName(ctx, …)` 读 `NamePreference`（cookie），标签/特征走不看偏好的 `CatalogVocabularyName`。v1 按 01 §3 发名字原语全三件。
- **简介由服务端选一条**：`preferredIntro` / `seriesIntro` / `pickCharacterIntro` 各有一套语言顺序。v1 按 infra 04 §8.1 发全部。
- **SFW 闸**：`utils.IsSFW` 读 cookie / 账号姿态。成员遍历（`CatalogMemberWorkIDs`）与作品行（`GetBatchPublic`、`CatalogRowsByCatalogIDs`）按它加 `content_limit=sfw`；标签列表与搜索删掉成人标签；角色特征删掉成人特征；系列卡片删掉「含 NSFW 或未知」的系列。v1 改成显式 `include_nsfw`。
- **两个列表引擎各钉一个面**（记忆 `kungal-entity-pages-full-catalogue`）：实体详情页的作品列表 = catalog 成员 id → 本地排序/分页（`hydrateIDPage`）；带资源条件时落到本地 SQL（`ListIDs` + `RestrictIDs`）。发售日排序推进 catalog 遍历（`catalogMemberSort`），其余排序本地排（`OrderRestrictIDs`，`m.ord` 决胜）。多选标签交集走 catalog 搜索（`CatalogWorksSearch`，`released_desc`）——那是另一个人口。
- **成员遍历不缓存**：每次请求整个走完（`taxonomyMemberPageCap` = 200 页）。
- **已合并实体**：会社、制作人员、角色的详情回 `200 {moved_to}`，网页 301。
- **资源筛选用旧标量列**：`ListIDs` 的 `type` / `language` / `platform` 过滤 `gr.type` / `gr.language` / `gr.platform`，卡片上的平台/语言也从标量列聚合（`FindResourceMetaBatch`）。资源轴（jsonb）是 G 轨资源面的词表。

### 1.3 疑似 bug（照实记，裁决见 §2）

| 编号 | 事实 |
|---|---|
| E1 | `/galgame-tag/:id` 对 SFW 读者直接返回成人标签的名字与简介（`GetDetail` 不查 `sexual`）；列表、搜索、筛选 chip 三处都藏了，只有详情漏（记忆 `kungal-sfw-hides-adult-tags` 已记为「先前就有的洞」） |
| E2 | **名字偏好污染缓存**：会社索引（30 分钟）与系列索引（10 分钟）用第一个触发重建的请求的 `ctx` 选名，之后所有读者都看这个人选的语言，直到过期。与 `batchCacheKey` 那次是同一个坑（记忆 `kungal-name-preference`） |
| E3 | `/galgame-tag/multi` 超过 10 个 `tag_ids` 时静默截断到前 10 个，不报错 |
| E4 | 会社列表条目的 `link`、`lang` 恒为空串（字段在 DTO 里、从没赋值） |
| E5 | 制作人员详情的 `roles`（页头的职务汇总）只从**当前这一页**的署名里算：署名超过 50 部的人，职务汇总会缺 |
| E6 | 标签搜索的 `total` 是 catalog 的命中数，SFW 下删掉的成人标签不减（代码注释承认「对 SFW 读者偏大」） |
| E7 | 成员遍历每请求走完、不缓存：`/galgame-tag/1333`（7,479 部）超过 25 秒 |
| E8 | `/galgame-series` 每次请求把 7,250 个系列走完（73 次 catalog 调用），编辑表单每打开一次就走一遍 |
| E9 | 服务端产出中文标签：职务名（`staffRoleName` 钉死的「脚本/原画/音乐/导演」+ catalog 的 `role_name`）、外链名（「官方网站」「批评空间」由 `LinkDisplayName` 产出） |
| E10 | 标签详情的 `alias` 恒为 `[]` |
| E11 | 卡片的平台/语言聚合把失效资源（`status=1`）也算进去 |

## 2. 裁决

### 2.1 与 G / X2 谈定的共享面

1. **作品卡片 `WorkSummary`**（`object: "work"`）放在中性包 `internal/galgame/workrepr`（类型 + 水合器），G 的 `/works` 浏览复用它。它**嵌入** X2 在 #199 加的 `repr.WorkRef`（`object`、`id`、`display_name`/`latin`/`localized{}`、`cover` = 竖版原图、`is_nsfw` = 编辑显示轴），再加本轨的字段（§3.2）。G 已审过字段表。
2. **实体作品子集合的筛选参数词表由本轨定、G 的 `/works` 原样复用**：`resource_type` / `resource_platform` / `resource_language` / `game_type` / `sort`（§3.4）。实现是给 `model.GalgameListFilter` + `ListIDs` 加一条**只增不改**的资源轴谓词路径；旧标量路径与它的测试字节不变，`/api/galgame` 继续用旧路径直到 G 删它。
3. **合并实体**：`404 ENTITY_MERGED`，扩展成员 `object` + `current_id`（infra 03 §4）。G 以后给合并作品复用同一个码。`Link: rel="canonical"` 不发——论坛的 problem 写出器没有逐错误的头通道，改它是动 `internal/apiv1` 地基；`current_id` 已经把信息带全。
4. G 同意本轨在 `internal/galgame/client` 加一个导出的 `MakerLabel`（`makerName` 改为调用它），在 `internal/galgame/service` 导出 `EntityUsesLocalList` / `CatalogMemberSort` 两个谓词（行为不变）。

### 2.2 逐条

| # | 裁决 |
|---|---|
| D1 | **名词跟 infra 03 §1**：`tags`、`companies`、`engines`、`series`、`credit-names`、`characters`；`object` 取同名单数。论坛的「制作人员」页实际寻址的是 catalog 的 `credit_name`（署名名义，`/catalog/names/{id}`），不是 `person`——infra 03 §1.1 专门记过这次混淆。网页 URL（`/galgame/tag/:id` 等）不变 |
| D2 | 名字：每个族都发 `display_name` + `latin` + `localized{}`（X2 的 `repr.CatalogName`）。标签、引擎、系列、特征这些词表族也一样（infra 04 §8）。修 E2：缓存里存的是原语，不是选好的一条 |
| D3 | 简介：`intros: [{lang, value, is_machine, source}]`，全部语言，不选（infra 04 §8.1） |
| D4 | `include_nsfw`（默认 `false`）是结果集随 NSFW 变化的每个读面的显式参数。**成人标签的详情在 `include_nsfw=false` 时回 404 `NOT_FOUND`**（修 E1；infra 03 §5 第 1 条：不可见一律 404）。角色特征、系列浏览、作品子集合同理 |
| D5 | 搜索并进族集合的 `q=`（infra 03 §3）。带 `q` 时人口是 **catalog 名字搜索的前 100 个命中**、再过本轨的闸（hidden 层、成人），按相关度排；`page × limit` 不得超过 100。于是 `total` 是这个人口的精确数（修 E6） |
| D6 | `credit-names` 与 `characters` 没有浏览序（13.5 万 / 24 万行），`q` 必填 |
| D7 | 标签、会社、引擎的浏览序 = `catalog_work_count` 降序，`id` 升序决胜（旧面按选出来的名字决胜，名字随偏好变，排序也跟着变） |
| D8 | 系列浏览只列「本站可列 ≥1 部」的系列（与旧卡片页同人口）；带 `q` 时在**全部** 7,250 个系列的名字里做子串匹配（编辑表单要能选到还没有本站作品的系列），命中的卡片数据按页现算。修 E8：全表索引缓存 10 分钟 |
| D9 | 实体作品子集合 = 页码集合，钉在实体车道（catalog 成员 → 本地排序/分页），参数见 §3.4；多选标签交集是独立的 `GET /tagged-works`，钉在 catalog 车道。两个车道从不共用一个集合 |
| D10 | `tagged-works` 的 `tag_ids` 超过 10 个 → `400 INVALID_PARAMETER` + `TOO_MANY_ITEMS`（修 E3，不截断） |
| D11 | 会社作品子集合的条目带 `via_company`（经哪个旗下厂牌算进来的）；「自有 / 经旗下」的两个数用 `via=own|imprint` 过滤 + `total` 求，不再塞进详情 |
| D12 | 制作人员的署名、角色的出演是**游标集合**（「加载更多」的流，上游只给 `next_offset`）。页头不再带职务汇总（修 E5：客户端从已加载的署名里算） |
| D13 | 职务发 `role_key`（catalog 的开放词表，折叠规则照旧：`剧本`→`scenario` 等）+ `display_name`（catalog 的 `role_name`，本站四个钉死的名字照旧）——它属于数据词表，按 01 §7 带 `display_name` 下发。外链只发 `source` token + `url`，标签由客户端按 token 查（修 E9 的外链一半） |
| D14 | 字段删掉：会社列表的 `link` / `lang`（E4）、标签的 `alias`（E10）、会社详情的 `link`（`links[]` 的第一条官网，客户端自己挑）、角色的单条 `intro`、`moved_to`（D-ENTITY_MERGED） |
| D15 | 卡片的平台/语言从资源轴聚合（G 的条件），失效资源照旧算（E11 不改：失效的链接也说明这个平台出过资源；它不是可见性问题） |
| D16 | 旧 wiki 会社 id 的解析成为一个资源：`GET /wiki-company-redirects/{wiki_company_id}`。只有 Nitro 中间件用它 |
| D17 | E7（成员遍历不缓存）本轨不修，行为照旧；记在 §6 |
| D18 | 上游 catalog 失败 → `503 SERVICE_UNAVAILABLE`，不回空列表（旧面回 `233`） |
| D19 | 无迁移。号段 160–164 不用。若资源轴 `EXISTS` 在生产规模的 EXPLAIN 里扫全表，先告诉 G（索引归 G 的号段 140–159） |

## 3. 形状

### 3.1 操作（21 个，取代 18 条旧路由）

全部 `GET`、`v1.Public`（匿名可读，不看凭证）。

| # | operationId | 路径 | 旧路由 |
|---|---|---|---|
| 1 | `listTags` | `/tags` | 1、2 |
| 2 | `getTag` | `/tags/{tag_id}` | 4（头） |
| 3 | `listTagWorks` | `/tags/{tag_id}/works` | 4（作品） |
| 4 | `listTaggedWorks` | `/tagged-works` | 3 |
| 5 | `listCompanies` | `/companies` | 5、6 |
| 6 | `getCompany` | `/companies/{company_id}` | 8（头） |
| 7 | `listCompanyWorks` | `/companies/{company_id}/works` | 8（作品 + 自有/旗下计数） |
| 8 | `getCompanyGraph` | `/companies/{company_id}/graph` | 9 |
| 9 | `getWikiCompanyRedirect` | `/wiki-company-redirects/{wiki_company_id}` | 7 |
| 10 | `listEngines` | `/engines` | 10 |
| 11 | `getEngine` | `/engines/{engine_id}` | 11（头） |
| 12 | `listEngineWorks` | `/engines/{engine_id}/works` | 11（作品） |
| 13 | `listSeries` | `/series` | 16、17 |
| 14 | `getSeries` | `/series/{series_id}` | 18（头）、17 的 `?ids=`（系列面板改成逐个取） |
| 15 | `listSeriesWorks` | `/series/{series_id}/works` | 18（作品） |
| 16 | `listCreditNames` | `/credit-names` | 12 |
| 17 | `getCreditName` | `/credit-names/{credit_name_id}` | 13（头） |
| 18 | `listCreditNameCredits` | `/credit-names/{credit_name_id}/credits` | 13（署名） |
| 19 | `listCharacters` | `/characters` | 14 |
| 20 | `getCharacter` | `/characters/{character_id}` | 15（头） |
| 21 | `listCharacterAppearances` | `/characters/{character_id}/appearances` | 15（出演） |

路径 id 一律 `pattern ^[1-9][0-9]{0,18}$`，不合格式直接 404（与现有 v1 面一致）。

### 3.2 `WorkSummary`（`object: "work"`，`internal/galgame/workrepr`）

= `repr.WorkRef`（嵌入、展平）+：

| 字段 | 类型 | 说明 |
|---|---|---|
| `banner` | `Image \| null` | 横版展示图，横幅槽位的**原图**（不是 `_mini`） |
| `release_date` | `string(date) \| null` | catalog 的 `2024` / `2024-05` 落到当年/当月 1 日 |
| `release_date_precision` | `"day" \| "month" \| "year" \| null` | 封闭；`release_date` 为 `null` 时也是 `null`（infra 04 §4.1）。不发 `release_status`：作品列表行不带它（A5） |
| `maker` | `CompanyRef \| null` | 署名排名最高的会社（developer > circle > brand > publisher），即旧卡片的 `company` 字符串，改成引用 |
| `view_count` | `integer ≥ 0` | |
| `like_count` | `integer ≥ 0` | |
| `rating_score` | `number \| null` | 贝叶斯分，一位小数；`rating_count = 0` 时 `null` |
| `rating_count` | `integer ≥ 0` | |
| `resource_platforms` | `[enum]` | 资源平台轴 `resourcevocab.PlatformKeys`，去重，按词表顺序 |
| `resource_languages` | `[enum]` | 资源语言轴 `resourcevocab.LanguageKeys`，去重，按词表顺序 |
| `resource_updated_at` | `date-time \| null` | |
| `is_published` | `boolean` | 本站有这部作品的页面且可列（旧 `is_on_forum`） |

catalog 有、本站没有行的成员：本地字段全部 `0` / `null` / `[]` / `false`。没有 `viewer`（赞与收藏状态是 G 的独立面）。

`CompanyRef` = `{object: "company", id, display_name, latin, localized}`。

### 3.3 实体

**`TagSummary` / `Tag`**（`object: "tag"`）

| 字段 | Summary | Tag | 说明 |
|---|---|---|---|
| `id`、名字三件 | ✔ | ✔ | |
| `tag_kind` | ✔ | ✔ | `content` \| `meta`（封闭，infra 04 §5.5 #6） |
| `is_sexual` | ✔ | ✔ | 成人标签 |
| `catalog_work_count` | ✔ | ✔ | catalog 归在它下面的全部作品，**含 NSFW**；读者能翻到多少看作品子集合的 `total` |
| `is_hidden` | | ✔ | hidden 层：不进列表与搜索，详情照发（旧页据此关 SEO） |
| `intros` | | ✔ | `[CatalogIntro]` |

**`CompanySummary` / `Company`**（`object: "company"`）

| 字段 | Summary | Company | 说明 |
|---|---|---|---|
| `id`、名字三件 | ✔ | ✔ | |
| `company_kind` | ✔ | ✔ | `game_brand` \| `bunko` \| `publisher` \| `anime_studio` \| `doujin_circle` \| `group`（封闭，infra 04 §5.5 #5；两个 0 行成员保留） |
| `logo` | ✔ | ✔ | `Image \| null` |
| `aliases` | ✔ | ✔ | `[string]`，别名值，去掉与名字相同的 |
| `catalog_work_count` | ✔ | ✔ | 同上，含 NSFW |
| `lang` | | ✔ | BCP-47 \| `null` |
| `links` | | ✔ | `[CatalogLink]` |
| `intros` | | ✔ | `[CatalogIntro]` |

**`CompanyWork`**（`object: "company_work"`）：`{work: WorkSummary, via_company: CompanyRef | null}`。

**`CompanyGraph`**（`object: "company_graph"`）：`{company_id, nodes: [CompanyGraphNode], edges: [CompanyGraphEdge]}`；节点 = `CompanyRef` 的字段 + `logo` + `catalog_work_count`；边 = `{from_company_id, to_company_id, relation}`，`relation` 封闭 8 值：`parent` `subsidiary` `imprint` `imprint_of` `succeeded_by` `formerly` `spawned` `origin`。上游出现第 9 个值时那条边被丢弃并记一条警告日志（不 500）。

**`WikiCompanyRedirect`**（`object: "wiki_company_redirect"`）：`{wiki_company_id, company_id}`。

**`Engine`**（`object: "engine"`，列表与详情同一形状）：`id`、名字三件、`aliases`、`description`（`string \| null`，catalog 的一段说明，原文）、`catalog_work_count`。

**`SeriesSummary` / `Series`**（`object: "series"`）

| 字段 | 说明 |
|---|---|
| `id`、名字三件 | |
| `has_nsfw_works` | `boolean \| null`：catalog 说这个系列有没有 NSFW 作品；`null` = catalog 不知道。SFW 浏览把 `true` 与 `null` 都排除（旧行为） |
| `catalog_work_count` | 含 NSFW |
| `listed_work_count` | 本站可列（有资源且已发布）的成员数 |
| `sample_works` | `[SeriesSampleWork]`，≤5 部本站可列的成员，按发售日升序；`SeriesSampleWork` = `WorkRef` + `banner`（系列卡片的横幅拼图要横版图） |
| `intros` | 只在 `Series` |

**`CreditNameRef`**（`object: "credit_name"`）：`id`、名字三件、`lang`（`string \| null`）。搜索命中、同一个人的其他名义、角色的声优都用它。

**`CreditName`**（`object: "credit_name"`）：`CreditNameRef` 的字段 + `photo`（`Image \| null`）、`gender`（`male` \| `female` \| `null`）、`birth_year` / `birth_month` / `birth_day`（各自 `integer \| null`：生日常常只知道一部分）、`intros`、`links`、`siblings: [CreditNameRef]`。

**`Credit`**（`object: "credit"`）：`{work: WorkSummary, roles: [CreditRole], characters: [CreditCharacter]}`；`CreditRole` = `{role_key, display_name}`；`CreditCharacter` = `{character_id: string \| null, display_name}`（catalog 只给原文串，有时带 id）。

**`CharacterRef`**（`object: "character"`）：`id`、名字三件、`image`（`Image \| null`）。搜索命中用它。

**`Character`**（`object: "character"`）：`CharacterRef` 的字段 + `lang`、`figure`（`Image \| null`）、`intros`、`traits: [CharacterTrait]`、`links`。

**`CharacterTrait`**（`object: "trait"`）：`id`、名字三件、`group`（`{display_name, localized}`：特征分组的名字）、`spoiler_level`（`none` \| `minor` \| `major`，上游 0/1/2）、`is_lie`、`is_sexual`。

**`Appearance`**（`object: "appearance"`）：`{work: WorkSummary, voices: [CreditNameRef]}`。

**`CatalogIntro`**：`{lang, value, is_machine, source: string | null}`。**`CatalogLink`**：`{source, url: string(uri) | null}`（DLsite 这类只有来源没有地址的保留）。两者与 `WorkSummary` 同包，G 的作品详情复用。

### 3.4 集合

| 集合 | 风格 | 参数 | 默认 limit | 排序 |
|---|---|---|---|---|
| `listTags` | 页码 | `q`、`include_nsfw` | 100 | 无 `q`：`catalog_work_count` 降、`id` 升；有 `q`：相关度 |
| `listCompanies` | 页码 | `q`、`company_kind` | 50 | 同上 |
| `listEngines` | 页码 | `q`（名字/别名子串） | 100 | `catalog_work_count` 降、`id` 升 |
| `listSeries` | 页码 | `q`（子串）、`include_nsfw` | 12 | `listed_work_count` 降、`id` 升（有 `q`：`catalog_work_count` 降、`id` 升） |
| `listCreditNames` / `listCharacters` | 页码 | `q`（**必填**） | 20 | 相关度 |
| `list{Tag,Company,Engine,Series}Works` | 页码 | 见下 | 24 | `sort`，默认 `resource_updated_desc` |
| `listTaggedWorks` | 页码 | `tag_ids`（1–10，必填）、`include_nsfw` | 24 | 固定：发售日降序 |
| `listCreditNameCredits` / `listCharacterAppearances` | 游标 | `cursor`、`limit`、`include_nsfw` | 20（≤50） | catalog 的顺序 |

会社没有成人与否，所以 `listCompanies` 没有 `include_nsfw`。

带 `q` 的页码集合：`page × limit ≤ 100`，越界 `400 INVALID_PARAMETER` + `OUT_OF_RANGE`（`maximum` = 最大页）。不带 `q`：通用深度上限 10000。

**实体作品子集合的参数**（G 的 `/works` 原样复用）：

| 参数 | 取值 | 语义 |
|---|---|---|
| `resource_type` | `resourcevocab.TypeKeys`（封闭） | 至少一个该类型的资源（标量列 `type`） |
| `resource_platform` | `resourcevocab.PlatformKeys`（封闭） | 至少一个资源的 jsonb `platforms` 含它 |
| `resource_language` | `resourcevocab.LanguageKeys`（封闭） | 至少一个资源的 jsonb `languages` 含它 |
| `game_type` | `ba_saku` `plot` `moe` `daily` `uncategorized`（封闭） | 评分给出的作品类型；`uncategorized` = 没有任何评分标过类型 |
| `sort` | `resource_updated_{desc,asc}` `created_{desc,asc}` `view_{desc,asc}` `view_1d_{desc,asc}` `view_7d_{desc,asc}` `view_30d_{desc,asc}` `release_date_{desc,asc}` `rating_{desc,asc}` | 未知 token → `400 UNKNOWN_SORT` |
| `include_nsfw` | `true`/`false`，默认 `false` | |
| `via` | `own` \| `imprint`（只在会社） | 缺席 = 两者 |

任一资源条件或 `game_type` 出现时，这一页落到本地 SQL（只含有资源的本站行）；否则是 catalog 的全部成员。这与旧面一致（`entityUsesLocalList`），`total` 与 `items` 永远同一谓词。并列排序键的决胜也照旧：catalog 成员车道按成员顺序（`OrderRestrictIDs` 的 `m.ord`），本地 SQL 车道按 `id` 降序。

### 3.5 错误码

| 码 | 状态 | 何时 |
|---|---|---|
| `NOT_FOUND` | 404 | id 不存在；成人标签且 `include_nsfw=false`；wiki 会社 id 没有对应 |
| `ENTITY_MERGED` | 404 | **新码**（平台域）。会社、制作人员名义、角色已被合并；带 `object`、`current_id` |
| `INVALID_PARAMETER` | 400 | `tag_ids` 超 10 个（`TOO_MANY_ITEMS`）或格式错；`page × limit` 越界（`OUT_OF_RANGE`）；`limit` 超上限 |
| `UNKNOWN_ENUM_VALUE` | 400 | `company_kind` / `resource_*` / `game_type` / `via` 的未知值（huma 解析期） |
| `UNKNOWN_SORT` | 400 | |
| `INVALID_CURSOR` | 400 | 游标坏了，或换了 `include_nsfw` 再用旧游标 |
| `SERVICE_UNAVAILABLE` | 503 | catalog 或本地库读失败之外的上游失败 |

`ENTITY_MERGED` 三处一译：`registry.go` + `registry_test.go` + `problem.json`。

## 4. 删除

- 18 条旧路由、`EntityHandler`、`app.go` 的 `GalgameEntityHandler` 字段与 `Official/Engine/Series/Staff/Character` 五个旧服务的构造。
- `TagService` 留着（`EntitySearchService` 用它的 `searchHits` / `sexualByID`），删 `GetList` / `GetDetail` / `GetByMultiTag` / `Search`。
- 死代码跑到不动点（`deadcode -test ./...`）；`dto/entity_dto.go` 里没人用的类型一并删。
- `legacy_route_baseline` 159 → 141（rebase 时重算）。
- 网页：`useGalgameOfficialDetail`、六个详情页、四个列表容器、系列面板、编辑表单的六个选择器、Nitro 的 OG 卡三处 + sitemap 三处 + `legacy-taxonomy` 中间件，全部切到生成的类型化客户端；手写类型删掉。

## 5. 网页要点

- 名字按 `localized['zh-Hans'] ?? display_name ?? latin` 渲染，由客户端的「优先原名」开关决定是否先取 `display_name`——与旧服务端选名等价。标签与特征仍不看这个开关（记忆 `kungal-name-preference`）。
- NSFW：页面把读者的姿态翻成 `include_nsfw`（`useContentStance`）。
- 合并实体：`ENTITY_MERGED` → `navigateTo(…current_id…, {redirectCode: 301})`，与旧 `moved_to` 同一个跳转。
- 会社「自有 / 经旗下」：主列表之后并行取 `via=own` 与 `via=imprint` 的 `limit=1` 求 `total`。
- 系列面板：逐个 `getSeries`（一部作品通常 1–2 个系列）。

## 6. 本轨不做

- E7 成员遍历缓存（行为照旧；大标签首屏仍慢）。
- `/search/entity*`（X 轨）。
- 资源轴 jsonb 的索引（若需要，G 的号段）。

## 7. 变异题（实现之前提交；每条都必须让某个测试变红）

| # | 变异 | 契约语义 |
|---|---|---|
| M1 | `getTag` 对成人标签不再看 `include_nsfw`（去掉 404 闸） | D4 / E1 |
| M2 | `listTags` 不再排除 hidden 层 | §3.3 |
| M3 | `listTags` 的成人过滤恒不生效（`include_nsfw=false` 也列成人标签） | D4 |
| M4 | 带 `q` 的 `listTags` 跳过 hidden/成人过滤 | D5 |
| M5 | 资源轴路径的 `resource_platform` 改去比较旧标量列 `gr.platform` | §2.1-2 |
| M6 | `is_nsfw` 改读 `content_rating`（年龄轴），不读认领的 `content_limit` | §3.2 |
| M7 | `rating_count = 0` 时 `rating_score` 发 `0` 而不是 `null` | §3.2 |
| M8 | 合并实体回 `200` + 幸存者（或 `ENTITY_MERGED` 少了 `current_id`） | §2.1-3 |
| M9 | 署名游标的指纹去掉 `include_nsfw` | §3.4 |
| M10 | `tagged-works` 的 `tag_ids` 超 10 个时截断而不是 400 | D10 |
| M11 | catalog 失败时回空列表（200）而不是 503 | D18 |
| M12 | 角色特征在 `include_nsfw=false` 时仍含成人特征 | D4 |
| M13 | `via=imprint` 被忽略（回全部成员） | D11 |
| M14 | 实体作品子集合的资源轴路径去掉 `id` 决胜（数据里有并列排序键） | §3.4 |
