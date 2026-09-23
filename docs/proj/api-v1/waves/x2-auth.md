# X2-auth · 登录会话与「我是谁」

> 2026-09-23 立。分支 `api-v1/x2-auth`，迁移号 208（**本轨不用**）。
> 契约与变异题同一个提交，早于实现。

## 1. 普查

### 1.1 路由与调用方

三条旧路由由 `OAuthHandler`（`internal/user/handler/oauth_handler.go`）独占，没有别的域共用。

| # | 旧 | 档 | 调用方 |
|---|---|---|---|
| 1 | `POST /api/auth/oauth/callback` | 公开 | `pages/auth/callback.vue`（唯一） |
| 2 | `POST /api/auth/logout` | 公开（读 cookie） | `components/kun/top-bar/Logout.vue`（「退出本设备」「退出全部」两处） |
| 3 | `GET /api/auth/me` | authed | `composables/useRefreshMe.ts`：客户端在页面可见 / 重新联网 / 会话校验时刷新持久化的用户仓库。它直接用 `$fetch` 拼 URL，不经 `kunFetch` |

`apps/web/server/`（Nitro）零调用；`../kungal-apps` 零调用（App 的登录是自己的环回 PKCE 直连账号中心，拿的是 Bearer，从不经过这两条 BFF 路由）。`docs/proj/app-direct-api.md` 第 88 行把 `GET /api/auth/me` 列成 Bearer 可用的面，删路由时同步改。

会话有效性早已由 `plugins/validate-session.client.ts` 走 v1 的 `GET /api/v1/me` 判断；`/auth/me` 只剩「把改过的资料同步进本地仓库」这一件事。

### 1.2 事实

- **账号中心登记的 `redirect_uri` 是网页，不是 API**：论坛客户端「鲲 Galgame 论坛」在生产登记的是 `https://www.kungal.com/auth/callback`、`https://kungal.com/auth/callback`、`http://127.0.0.1:2333/auth/callback`（本地库同）。浏览器被重定向到网页 `/auth/callback`，网页再把 `code` + PKCE `code_verifier` `POST` 给 `/api/auth/oauth/callback`；后端换码时带的 `redirect_uri` 是配置 `OAUTH_REDIRECT_URI`（同一个网页地址）。**`/api/auth/oauth/callback` 这个 API 路径在 infra 里没有任何登记。**
- 回调在 Redis 里建 `SessionData`（`kungal:session:v2:<token>`），下发 HttpOnly、`SameSite=Lax` 的不透明 cookie `kungal_session`，90 天滑动（`docs/proj/session-lifetime.md`）；顺带做一次 community trust boost。
- 登出：用会话里的 refresh token 向账号中心吊销，删 Redis 会话，清 cookie。「退出全部」再把浏览器跳到账号中心 `/oauth/logout`。
- `/auth/me` 的字段：`id` `sub` `name` `avatar` `roles` `moemoepoint` `bio` `adult_confirmed` `nsfw_display`。名字、头像、简介取自 OAuth（userclient，约 10 分钟热缓存），**角色与成人向立场取自会话本身**（Bearer 下角色已剥掉 staff）。网页只用 `name` `avatar` `roles` `adult_confirmed` `nsfw_display`；`sub` `moemoepoint` `bio` 无人读（萌萌点由 `GET /api/v1/me` 提供）。
- 生产 `users` 13 万行里 8943 个有头像哈希，**0 个只有头像 URL 没有哈希**：从哈希构造 `Image` 与旧 `avatar` URL 逐字相同。
- 网页只拿 `roles` 判断 `creator` / `moderator` / `admin` / `ren` 四个有级别的角色，没有任何地方读 `user`。

### 1.3 疑似 bug

1. `/auth/me` 上游失败回 `500`「查询用户信息失败」，查无此人（账号已删）回 `404`；两者在网页端都被 `.catch(() => null)` 吞掉，没有区别。
2. 回调的错误是 `233` + 中文句子（「OAuth 授权码交换失败: …」把上游错误串原样拼给了浏览器）。它留在 v1 之外（§2），这条不在本轨修。

## 2. 裁决

**K-X2U1 · 登录回调与登出不属于 v1 面，原样保留。** 与 TS 的 K-TS1 同理，理由不同：

- 它们是 BFF 的**凭证管道**：一个把 OAuth 授权码换成浏览器的 cookie 载体，一个销毁它。v1（K2）只有一种主体、两种载体，载体本身怎么铸造、怎么销毁是 BFF 的实现，不是 API 资源。
- App 永远不该调它们（App 走环回 PKCE + Bearer）。写进 App 绑定的 spec，生成的客户端里就多两个它不能用的操作。
- 挪路径本身不需要动 infra（§1.2：API 路径没有登记），但登录链路一断就是全站新登录失败，而收益只是路由表整齐。不值。
- 于是 `legacy_route_baseline` 的地板从 2 变成 **4**：`/healthz`、`/api/trust/callback`、`/api/auth/oauth/callback`、`/api/auth/logout`。

**`GET /api/auth/me` 迁移**，并且不并进 U1 的 `GET /api/v1/me`：那是状态面（萌萌点、签到、未读），它的契约属 U 轨；「这个凭证代表谁」是另一个资源。

| 旧 | v1 |
|---|---|
| `POST /api/auth/oauth/callback` | **不迁**（K-X2U1） |
| `POST /api/auth/logout` | **不迁**（K-X2U1） |
| `GET /api/auth/me` | `GET /api/v1/me/account` |

删 1 条，`legacy_route_baseline` 159 → **158**（以合并时 rebase 后重新生成为准）。

## 3. 形状

### 3.1 `Account`（`GET /api/v1/me/account`，required 档）

```json
{
  "object": "account", "id": "8",
  "name": "xiaohuo", "avatar": { "url": "https://…", "hash": "…", … },
  "roles": ["admin"],
  "is_adult_confirmed": true,
  "nsfw_display": "blur"
}
```

| 字段 | 类型 | 来源 |
|---|---|---|
| `id` | DecimalID | 凭证 |
| `name` | string ≤ 64 \| null | OAuth 当前记录；账号已不存在时 `null`（同 `UserRef.name`） |
| `avatar` | Image \| null | OAuth 当前记录的头像哈希 |
| `roles` | [`creator` \| `moderator` \| `admin` \| `ren`] | **凭证携带的**有效角色，只保留四个有级别的角色（与 `AccessGrants.roles` 同一词表）。Bearer 请求里 staff 角色本来就被剥掉 |
| `is_adult_confirmed` | boolean | 会话里的成人确认（旧 `adult_confirmed`，F1：布尔以 `is_` 开头） |
| `nsfw_display` | `hide` \| `blur` \| `show` | 会话里的显示立场（与 U1 `NsfwDisplay.nsfw_display` 同型） |

**K-X2U2 · `account` 描述的是「这个凭证代表的账号」，不是账号中心里的档案。** 角色与立场取自凭证（会话 / Bearer），名字与头像取自 OAuth 当前记录。所以同一个人经 Bearer 与经网页会话读到的 `roles` 可以不同，这是 K2 的直接结果，不是不一致。

不下发 `sub`、`moemoepoint`、`bio`：前两者无人读（萌萌点在 `GET /api/v1/me`），`bio` 属资料面（U2 的公开资料）。A5。

### 3.2 错误

- 未带凭证 `401 MISSING_CREDENTIAL`；凭证无效 `401 INVALID_CREDENTIAL`；封禁 `403 ACCOUNT_BANNED`（身份中间件）。
- OAuth `/users/batch` 不可用 → `503 SERVICE_UNAVAILABLE`（不拿会话里登录时的旧名字冒充当前名字）。
- 账号在 OAuth 查无此人 → `200`，`name` 与 `avatar` 为 `null`（与 `UserRef` 对已删账号的表示一致），不是 404。

## 4. 预分配

- 迁移：**无**。错误码：**无新增**。
- K 编号：**K-X2U1**（BFF 登录管道留在 v1 外）、**K-X2U2**（account = 凭证所代表的账号）。

## 5. 网页

- `useRefreshMe.ts` 改用类型化客户端 `GET /me/account`；`name` 为 `null` 时不覆盖本地仓库（沿用旧的 `me?.name` 守卫）；`id` 是字符串，比较前转数字；头像取 `avatar?.url ?? ''`。
- `callback.vue`、`Logout.vue` 不动。
- `docs/proj/app-direct-api.md` 第 88 行改指 `GET /api/v1/me/account`。

## 6. 变异题（先于实现提交）

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | 不过滤角色，原样下发凭证的角色 | 会话角色 `["user","moderator","some-site-name"]` → `roles` 恰为 `["moderator"]`（原样下发还会让 spec 校验失败） |
| 2 | 角色改读 OAuth 档案（userclient）而不是凭证 | OAuth 档案是 moderator 的用户经 Bearer 访问 → `roles` 不含 `moderator` |
| 3 | `is_adult_confirmed` 写死 `false` | 会话里已确认成人 → `true` |
| 4 | `nsfw_display` 写死 `hide` | 会话立场 `blur` → `blur` |
| 5 | 名字取会话里登录时存的名字 | 会话名 `old-name`、OAuth 当前名 `new-name` → `new-name` |
| 6 | OAuth 不可用时退回会话里的名字并回 200 | OAuth 500 → `503 SERVICE_UNAVAILABLE` |
| 7 | 查无此人回 404 / 500 | OAuth 查无此人 → `200`，`name` `avatar` 为 `null` |
| 8 | 头像不从哈希构造（恒 `null`） | 有头像哈希的账号 → `avatar.url` 为图床地址、`avatar.hash` 为该哈希 |
| 9 | 操作挂成 `public` 档 | 匿名 → `401 MISSING_CREDENTIAL` |
