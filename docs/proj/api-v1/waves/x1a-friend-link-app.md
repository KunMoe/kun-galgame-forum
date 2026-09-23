# X1a · 友链与 App 版本

> X1 轨第一段，2026-09-23，分支 `api-v1/x1-friend-link`，迁移号段 195–201（本段不用）。
> X1 = 看板 X 行里与 galgame 无关的一半，由协调会话分派；另一半 X2 归 TS/P 会话，两边的 handler 集合已核对、无交集。X1 切四段：
>
> | 段 | 范围 | 旧路由 |
> |---|---|---|
> | **X1a**（本文） | 友链 `/friend-link` + `/admin/friend-link*`；App 版本 `/app/version` | 6 |
> | X1b | 版块 `/section` + `/category` | 2 |
> | X1c | 情报 `/news*` + RSS `/rss/*` | 6 |
> | X1d | 图片上传 `/image/*` | 4 |
>
> 契约与变异题同一个提交，早于实现。

## 1. 生产取值（2026-09-23，只读）

| 事实 | 数字 |
|---|---|
| `friend_link` | 43 行：`official` 4、`galgame` 31、`others` 8 |
| `status` | `normal` 42、`down` 1、**`essential` 0**（历史上也没有） |
| 横幅 | 43/43 有 `banner_image_hash`（2026-06 回填过） |
| `sort_order` | 每个分类内无并列 |
| 字段长度 | `name` ≤19、`link` ≤27、`description` ≤106；`link` 全部以 `http(s)://` 开头 |

## 2. 调用方

| 旧路由 | 网页 | Nitro | Flutter App |
|---|---|---|---|
| `GET /friend-link` | `components/friend-links/Container.vue`（公开页）、`pages/admin/friend-link.vue` | 无 | 无 |
| `POST` / `PUT` / `DELETE /admin/friend-link`、`PUT /admin/friend-link/reorder` | `pages/admin/friend-link.vue` | 无 | 无 |
| `GET /app/version` | `components/client-app/Download.vue`（`/app` 下载页） | 无 | 没接（仓里一个 API 调用都还没写）；`docs/proj/app-direct-api.md` §4 把它定为版本闸 |

## 3. 普查抓到的旧面问题

| # | 问题 | 去向 |
|---|---|---|
| 1 | **友链页的 SEO 描述读的是一份静态 JSON**（`app/config/friend.json`，40 条），而正文读 API（43 条）：新加的 3 个站不在 `<meta description>` 里，删掉的站还在 | 描述改从 API 数据生成，静态 JSON 与 `config/friend.ts` 删除 |
| 2 | 链接校验用 validator 的 `url` 标签，不限 scheme：`javascript:` 能过（记忆 `kungal-resource-link-validation` 同一个坑） | `url` 必须是 `^https?://` |
| 3 | 更新 / 删除不存在的 id 静默成功 | 404 |
| 4 | 重排只写传来的 id，漏掉的保留旧序号，与新序号撞车；传别的分类的 id 被静默忽略 | 必须恰好是该分类全部友链的一个排列（同 D §6.5） |
| 5 | 仓储层读取的错误全部丢弃，失败时回空列表 | 500 |
| 6 | 旧 `status` 把「精选」和「已下线」混在一个枚举里；精选 0 行 | K26：0 行的取值不建模，只剩 `normal` / `down` |

## 4. 逐条去向

| # | 旧路由 | v1 | 档 |
|---|---|---|---|
| 1 | `GET /friend-link` | `GET /api/v1/friend-links`（游标） | public |
| 2 | — | `GET /api/v1/admin/friend-links/{friend_link_id}`（G17） | required，`friend_link.edit` |
| 3 | `POST /admin/friend-link` | `POST /api/v1/admin/friend-links` → **201** | required，`friend_link.create` |
| 4 | `PUT /admin/friend-link` | `PATCH /api/v1/admin/friend-links/{friend_link_id}` | required，`friend_link.edit` |
| 5 | `DELETE /admin/friend-link` | `DELETE /api/v1/admin/friend-links/{friend_link_id}` → **204** | required，`friend_link.delete` |
| 6 | `PUT /admin/friend-link/reorder` | `PUT /api/v1/admin/friend-link-order` → **204** | required，`friend_link.edit` |
| 7 | `GET /app/version` | `GET /api/v1/app/version` | public |

6 条旧路由全删，`legacy_route_baseline` 下调 **6**。写面挂 `/admin`，与 D、WS 同一个模式。权限一律 `user.Can`，Bearer 永远不持有 `friend_link.*`。

## 5. 形状

### 5.1 `FriendLink`（`object: "friend_link"`）

```
FriendLink  object, id, friend_link_category, name, url, description,
            banner: Image|null, state
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `friend_link_category` | 封闭枚举 `official` / `galgame` / `others` | 不叫 `category`：`Topic.category` 已占用这个名字（G8） |
| `name` | string 1–100 | 原文 |
| `url` | string，`format: uri`，`^https?://`，≤500 | 旧名 `link` |
| `description` | string ≤500 | 可为空串 |
| `banner` | `Image` \| null | 只来自 `banner_image_hash`（同 D §6.1） |
| `state` | 封闭枚举 `normal` / `down` | 旧名 `status`；`essential` 不建模（§3 #6）。库里出现别的值 → 500，不许静默归类 |

不下发：`sort_order`、`banner`（旧路径）、`banner_url`、`created`、`updated`。

公开面与管理面发同一个形状：友链没有只给管理员看的字段。

### 5.2 `GET /api/v1/friend-links` → `List[FriendLink]`（游标）

- 参数：`cursor`、`limit`（1–100，默认 20）、`friend_link_category`（缺席 = 全部）。
- 顺序固定，没有 `sort` 参数：先按分类（`official` → `galgame` → `others`，即枚举顺序），再按 `sort_order ASC`，再按 `id ASC`。游标记三个键。
- 游标指纹绑定 `friend_link_category`。
- 旧响应是按分类分组的对象；v1 是平铺列表，客户端自己分组。

### 5.3 写面

```
FriendLinkCreate  friend_link_category (必填), name (必填), url (必填),
                  description?, banner_image_hash?, state? (默认 normal)
FriendLinkPatch   以上全部可选；至少一个字段
```

- `name` 去首尾空白后不能为空（`422 TOO_SHORT`）；`description` 去首尾空白后存。
- `banner_image_hash`：`^([0-9a-f]{64})?$`，空串 = 没有横幅（同 D）。
- 新友链排在所属分类的最后；`PATCH` 改了分类，就移到新分类的最后。
- 不存在 → 404。

`PUT /api/v1/admin/friend-link-order`：`{ friend_link_category, friend_link_ids }`，必须恰好是该分类全部友链的一个排列。
- 不存在或属于别的分类的 id → `422 UNKNOWN_REFERENCE`（`pointer /friend_link_ids/<i>`）；
- 少了 → `422 TOO_FEW_ITEMS`（`params.min_items` = 该分类的友链数）；
- 重复 → schema `uniqueItems` 拦下。

### 5.4 `GET /api/v1/app/version` → `AppVersion`

```
AppVersion  object="app_version", min_version, latest_version, notes,
            downloads: { android, ios, windows, linux }
```

- 全部来自启动时校验过的配置（`KUN_APP_*`），不碰数据库。
- 版本号 `^[0-9]+\.[0-9]+\.[0-9]+$`；`notes` 自由文本，≤2000；`downloads.*` 是 `format: uri` 的字符串，没配的平台照旧回落到 `https://www.kungal.com/app`（配置层已经这么做）。
- 旧信封里的形状原样保留，只去掉信封。

## 6. 错误码

**无新增。**

## 7. 网页

| 文件 | 改什么 |
|---|---|
| `components/friend-links/Container.vue` | `GET /friend-links`（翻完所有页），客户端按 `friend_link_category` 分组；`url`、`banner?.url`、`state` |
| `pages/friend-links/index.vue` | SEO 描述改用同一份 API 数据；删 `config/friend.json`、`config/friend.ts` |
| `pages/admin/friend-link.vue`、`components/admin/friend-link/{Section,Modal}.vue` | 读用同一个列表；新建 `POST`、编辑 `PATCH`（只发改过的字段）、删除 `DELETE`、拖拽 `PUT /admin/friend-link-order`（422 → 重拉）；状态下拉去掉「精选」 |
| `components/client-app/Download.vue` | `GET /app/version` |
| `shared/types/friend-link.ts`、`shared/types/app-release.ts` 的手写接口 | 删掉或改成生成物别名 |

`docs/proj/app-direct-api.md` §4 与路由表里的 `/api/app/version` 改成 `/api/v1/app/version`；`CHANGELOG.md` 记一条（App 可见）。

## 8. 变异题（先于实现提交）

| # | 破坏 | 必须变红的断言 |
|---|---|---|
| 1 | 列表去掉 `id` 决胜键 | 小 `limit` 翻完全部，与 SQL 顺序逐条相等（**种子里同一分类有并列的 `sort_order` 跨页**） |
| 2 | 分类按字符串排序而不是枚举顺序 | 第一条是 `official` |
| 3 | 游标指纹去掉 `friend_link_category` | 换分类复用游标 → `400 INVALID_CURSOR` |
| 4 | 新建不查 `friend_link.create` | 普通用户 → 403 |
| 5 | 删除查 `friend_link.edit` 而不是 `.delete` | 用户级覆盖撤掉 `friend_link.delete` 的版主 → 删除 403、编辑 200 |
| 6 | `url` 不限 scheme | `javascript:alert(1)` → 422 |
| 7 | 重排接受不完整的列表 | 少一条 → `422 TOO_FEW_ITEMS`，库里顺序不变 |
| 8 | 重排接受别的分类的 id | → `422 UNKNOWN_REFERENCE` |
| 9 | 新友链 `sort_order` 写 0 | 新建后在该分类最后 |
| 10 | `PATCH` 改分类后不移到新分类末尾 | 改分类后在新分类最后 |
| 11 | 删除 / `PATCH` 不存在的 id 成功 | → 404 |
| 12 | 未知 `status` 值被映射成 `normal` | 库里写入 `essential` → 读面 500 |
| 13 | Bearer 拿到管理能力（`perm.CanUser`） | 静态守卫 `bearer_guard_test` |

## 9. 迁移

**无。**

## 10. 实现时对本契约的修正（2026-09-23，只增不改）

1. **`name` 改名 `title`**。G8 要求全 spec 同名字段同型，而 `UserRef.name` 是 `string|null`，友链的站名是非空字符串。WS 的网站已经把站名叫 `title`，这里跟着用。请求体同步改名。
2. **变异 11 只测删除**：`PATCH` 不存在的 id 由之后的重读回 404，是等价变异（同 D §13 #5），不单列。
3. **网页的编辑只发改过的字段**（`PATCH`），测试时抓到的请求体是 `{"title":"…"}`。

### 变异执行结果

13/13 杀。

| # | 结果 |
|---|---|
| 1 | `TestV1FriendLinksWalk`、`TestV1FriendLinksShelfAndCursor` |
| 2 | `TestV1FriendLinksWalk` |
| 3 | `TestV1FriendLinksShelfAndCursor` |
| 4、5 | `TestV1FriendLinkWritesNeedTheirPermissions`（5 号的做法：用户级覆盖撤掉 `friend_link.delete`，版主仍能改、不能删） |
| 6、9、10、11 | `TestV1FriendLinkCreateAndPatch` |
| 7、8 | `TestV1FriendLinkOrderReplacesTheShelf` |
| 12 | `TestV1FriendLinkUnknownStatusIsNotNormal` |
| 13 | `TestCapabilityChecksGoThroughUserInfo`（静态守卫） |

### 浏览器实测（开发库，worktree 构建的二进制，无头 Chromium）

- **匿名**：友链页列出全部 43 条，三个分类齐全；SEO 描述由 API 数据生成，包含每一条；已下线的站带「已下线」标签；`/app` 下载页显示最新版本号。
- **普通用户**：`/admin/friend-link` 被挡回首页；API 返回 `403 PERMISSION_REQUIRED`。
- **管理员**：`javascript:` 链接在表单里就被拦下；新建的链接排在所属分类的最后；编辑后 `PATCH` 只带改过的 `title`；拖拽后 `PUT /admin/friend-link-order` 返回 204，页面上的顺序与库里一致；删除后接口返回 404。
- 测试完已用 API 把开发库的顺序恢复原样，测试链接已删除。
