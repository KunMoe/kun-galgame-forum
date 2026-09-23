# 02 · 契约治理

## §1 单一来源

- **spec 由运行中的代码产出**（infra A14）。v1 的每个操作用 huma 注册，路由、输入输出类型、描述、状态码在同一处声明；handler 就是被声明的那个函数。
- 注册**不得**依赖活的数据库或 Redis：`go run ./cmd/openapi` 在零依赖下构建同一个 huma API，写出：
  - `apps/api/openapi/kungal-v1.json` —— OpenAPI 3.1，`info.x-stability: preview`；
  - `apps/api/openapi/problems.json` —— 错误码与 reason 注册表的机读形式。
- 两个文件提交进仓，输出确定（键序稳定、两空格缩进、末尾换行），重跑无 diff。
- `make openapi` 是唯一的生成入口。改了 v1 的任何东西，同一个提交里必须带上重新生成的这两个文件（门 G1）。
- 不对外提供 `/api/v1/openapi.json` 与 `/docs`：仓里的文件就是契约。App 仓按论坛的 commit 钉住它。

## §2 客户端

### 2.1 网页（W0b 落地）

- **类型**：`openapi-typescript` 从 `apps/api/openapi/kungal-v1.json` 生成 `apps/web/shared/types/api/v1.d.ts`（提交进仓；eslint / prettier 排除）。版本进 `devDependencies` 与 lockfile，不用 `pnpm dlx`。
- **客户端**：`openapi-fetch`，按「路径 + 方法」推导参数、请求体、响应与错误的类型。写错一个字段名，`pnpm typecheck` 就报错（`pnpm vue-tsc --noEmit` 不会，见 04 §5 第 8 条）。snake_case 迁移时那约 40 处「发出的字段名不对」，在这套机制下全部是编译错误。
- **SSR**：用 openapi-fetch 的中间件做三件事——服务端 / 浏览器两套 base URL、SSR 时转发 `kungal_session` cookie、服务端超时。页面级数据走 `useAsyncData` 包一层（`useApi`），沿用 Nuxt 的 payload 水合，不在客户端重复请求。
- **错误**：客户端把非 2xx 统一收成 `Problem` 对象。展示只走 `problemMessage(problem, locale)`：按 `code` / `reason` / `params` 查 `i18n/locales/zh-CN/problem.json`，未知 code 按 status 兜底。调用点可以按 code 接管展示，例如 404 渲染自己的空态，或表单把 `errors[].pointer` 挂到对应控件下。**不得**把 `title` / `detail` 显示给用户。
- **幂等键**：创建类 `POST` 由客户端在「用户点一次提交」时生成一个 UUIDv7，同一次提交的重试复用它。
- **手写类型**：v1 操作的请求与响应**不得**再手写 TS 类型。需要具名类型时，从生成物里别名：`type TopicSummary = components['schemas']['TopicSummary']`。旧的手写类型随对应旧端点一起删。
- **旧调用的棘轮**：`kunFetch` / `useKunFetch` 的调用点数只减不增（门 F4）。

W0b-1 落地的形状：

- 纯 TS 模块在 `apps/web/shared/utils/api/`，网页与 Nitro 共用：`client.ts`（`createApiClient({ origin, cookie?, timeoutMs? })`，全站唯一出现 `/api/v1` 的地方；`sessionCookie` 只取 `kungal_session`）、`problem.ts`（`settle` 把 openapi-fetch 的结果收成 `ApiResult`，失败是 `ClientProblem { kind, status, code, errors, requestId }`，不带 `title` / `detail`）、`message.ts`（`problemMessage`、`fieldMessage`）。超时用包一层 `fetch` 的 `AbortSignal.timeout`，并与调用方的 signal 合并。
- `settle` 只在 `Content-Type` 是 `application/problem+json` 且体里有字符串 `code`、数字 `status` 时才认作 problem；代理的 502 页面、空体 429 都是 `kind: 'http'`，按 status 兜底。`TypeError` / `AbortError` 是 `network`，`TimeoutError` 是 `timeout`，其余异常原样抛出（bug 不伪装成断网）。
- Nuxt 层：`useApiClient()`（服务端每次新建，带 cookie 与 10 秒超时；浏览器单例）、`useApi(key, (api, { signal }) => api.GET(…))`（包 `useAsyncData`，handler 返回纯 JSON 的 `ApiResult`，所以 SSR payload 可直接水合；key 可响应式）、`reportProblem(problem)`（toast，服务端无操作）。
- 类型测试：`shared/utils/api/client.typetest.ts` 与 `app/composables/useApi.typetest.ts` 由 `pnpm -F web typecheck` 检查。`@ts-expect-error` 钉住错路径、错查询参数、错排序 token、读错字段名；最后一条就是 `additionalProperties` 那个坑的回归门。
- 会话失效 / 封禁 / 重新授权的全局副作用与 UUIDv7 幂等键**还没做**：等第一个需要它们的 v1 调用（必需鉴权档或创建类 `POST`）一起落地，免得写出没有真实调用验证的代码。

### 2.2 App（kungal-apps）

- tonik 从论坛仓 `apps/api/openapi/kungal-v1.json` 生成 Dart 客户端，按论坛 commit 钉版。
- 只绑定 `/api/v1`；旧 `/api/*` 对 App 视为不存在。
- 错误展示遵守 [01 §2 K8](01-standard.md)：ARB 的键从 `problems.json` 来，App 仓自己做覆盖检查。
- 请求带 `User-Agent: kungal-app/<版本> (<平台>)`。论坛按它统计各版本流量，这是退役旧形状的依据（§4）。

## §3 CI 门

G 编号沿用 infra 07 §2 的同名门，F 编号是论坛补的。**每道门必须有阳性对照**：测试里先构造一个违规的 schema 或样本，证明门会红，再对真实产物断言。对照要走和被断言者**同一条构造路径**（infra 第九轮教训：路径字面量搜索漏掉了把 `/api/v1` 放进环境变量的客户端，而阳性对照照样通过）。

| 门 | 检查 | 实现 | 作业 |
|---|---|---|---|
| **G1** | ① 提交的 spec 与 `problems.json` 等于重新生成的结果。② 契约测试打真路由：每条响应都通过 spec 校验（状态码在声明集合里、body 符合 schema、`Content-Type` 正确） | ① `make openapi && git diff --exit-code`。② Go 测试（`santhosh-tekuri/jsonschema/v6`，Draft 2020-12，开格式断言；文档是 OpenAPI 3.1，kin-openapi 只认 3.0），带真库 | api · db |
| **G2** | 操作、参数、property、响应都有非空 `description` | spec 测试 | api |
| **G3** | 每个枚举标 `x-vocabulary-closed`；开放枚举标 `x-vocabulary`。struct tag 写 `enum:"…"` 即声明封闭词表，由 `sealDocument` 补标；开放词表用 `repr.OpenEnum` | spec 测试 | api |
| **G4** | 每个操作声明它真实会发的全部状态码，错误响应一律 `application/problem+json` 且 `$ref` Problem。最低集合由 `apiv1.RequiredStatuses` 从操作推导，`sealDocument` 据此补进文档、门据此检查，同一个函数，任何操作都不手写：全部操作 500；有参数 400；`required` / `optional` 档 401 + 403（封禁）+ 503（会话存储或 OAuth 故障）；路径带 id 404；有请求体 400 + 415 + 422；要求幂等键 409。反向同样查：推导不出的错误状态，其 description 必须点名至少一个注册表里同状态的 code；description 里点名的每个 code 都必须在注册表里且状态一致；不许有 `default` 响应。W0a-5 验收时列表带着一个永远发不出的 422：tier 辅助函数往 `op.Errors` 里塞状态码，huma 见它非空就给每个带参数的操作补 422，见它为空又补 `default`，所以 tier 改为直接声明 500 响应 | spec 测试 | api |
| **G5 / G13** | 注册表七项检查（infra 10 §7）；code ↔ type URI 双向一一对应；`problems.json` 与注册表一致；代码里构造的每个 code / reason 都在注册表里 | Go 测试 + AST 扫描 | api |
| **G6** | 2xx schema 顶层不含 `code` / `message` / `data` / `success` / `status` / `timestamp` / `error` | spec 测试 | api |
| **G7** | 名为 `id`、以 `_id` 结尾的 property 与参数是字符串；以 `_ids` 结尾的是字符串数组 | spec 测试 | api |
| **G8** | 同名 property 全 spec 内 schema 一致：类型、format、可空性、枚举值、`$ref` 目标、数组元素。例外只走具名清单：`object`（各资源各一个单值判别枚举）、`state`（各资源各自的生命周期封闭枚举）、`items`（列表容器的成员，元素类型随列表而变）、`children`（正文节点的子节点，元素类型随父节点而变，[03 §1](03-content-doc.md)）、`viewer`（K9：每种资源各带自己的查看者状态，字段必然不同）。查询 / 路径参数不参与一致性比较（`sort` 等是各集合自己的词表），但禁用名照查 | spec 测试 | api |
| **G9** | 数组 / map 不允许 `null`；没有 `additionalProperties: false`；请求体里没有可空对象（huma 把 `{"type":"null"}` 分支当作匹配任何值，等于关掉该字段的校验，改用 `omitempty`）；v1 包里 `omitempty` / `omitzero` 只在指针字段上 | spec 测试 + Go AST | api |
| **G14** | 字符串必须有 `enum` / `format` / `pattern` 之一或自由文本声明，且全部有 `maxLength`；数值有 `minimum` | spec 测试 | api |
| **G16** | `request_id` 匹配 `^req_[0-9A-HJKMNP-TV-Z]{26}$`；游标匹配 `^cur_` | spec 测试 + 契约测试 | api · db |
| **G17** | 写得进去就读得出来：写操作路径里的每个 `{x_id}`，以它结尾的那段路径必须有 GET，且其 200 响应是带 `id` 的对象。例如 `PUT /topics/{topic_id}/like` 要求 `GET /topics/{topic_id}` | spec 测试 | api |
| **F1** | 命名规则（[01 §3](01-standard.md)），property 与参数都查：布尔以 `is_` / `has_` / `can_` 开头（查询参数另允许 `include_`）、`_at` ↔ date-time、`_date` ↔ date、`_count` 为非负整数、封闭枚举值是 snake_case（具名例外只有 `sections` / `section`：论坛的 URL slug） | spec 测试 | api |
| **F8** | v1 源码（与 G5 同一组目录）里没有中日韩文字的字符串字面量：给人看的文字由客户端按语言出，服务端发 code 或 `null`。W0a-5 验收时发现删号作者经 `userclient.Placeholder` 以「已注销用户」上了线，现在 `UserRef.name` 为 `null` | Go AST | api |
| **F9** | 列表响应声明 `total` 当且仅当操作接受 `include_total`：`repr.List` 不带 `total`，嵌了 `collect.Total` 的集合返回 `repr.CountedList`。W0b-3 之前 `List` 自带 `total`，三个端点都声明了一个永远不会出现的字段，生成的类型里是一个读出来恒为 `undefined` 的 `total?: number` | spec 测试 | api |
| **F10** | 路径模板里的每个 `{变量}` 恰有一个同名 `in: path` 参数，反之亦然。W2 契约初稿把路径参数放进未导出类型的嵌入结构体，huma 静默丢掉，`/topics/{topic_id}` 没有参数也没推导出 400，其余各门全部放行 | spec 测试 | api |
| **F2** | 注册表的每个 code 与 reason 在 `zh-CN/problem.json` 里都有译文，目录里没有多余键。reason 的译文按参数分变体（`default` 必有，其余键是所用参数名升序以 `__` 连接，占位符恰好是这些参数），`status` 至少覆盖注册表里出现的每个状态与 429 / 502 / 504，文本里不许出现 vue-i18n 的特殊字符 `@` `$` `\|` | vitest（`tests/api/problemCatalog.spec.ts`），读 `problems.json` | web |
| **F3** | 旧路由数**等于**基线：`routes.golden` 里 `/api/v1` 以外的路由数 = `legacy_route_baseline`。删路由的提交必须同时下调基线，理由同 F4 | Go 测试 | api |
| **F4** | `kunFetch` / `useKunFetch` 调用点数**等于**基线 `tests/api/legacy-fetch-baseline`：少了也红，删调用点的提交必须同时下调基线，否则基线留下的余量会让新调用悄悄长回来 | vitest 源码扫描（`tests/api/legacyFetchRatchet.spec.ts`） | web |
| **F5** | 提交的 `v1.d.ts` 等于从提交的 spec 重新生成的结果。openapi-typescript 精确钉版，生成物随版本变 | `pnpm -F web gen:api && git diff --exit-code` | web |
| **F6** | 应用代码（`app/` `server/` `shared/`）里不出现以 `/v1` 或 `/api/v1` 开头的字符串字面量或模板片段：v1 只能经类型化客户端调用。唯一例外是 `shared/utils/api/client.ts`。别家服务的 `${base}/v1/…` 也会命中，逐行 `eslint-disable-next-line` 并写明是谁的 API（现有两处：贴纸站、OG 卡片服务） | eslint `no-restricted-syntax`，阳性对照 `tests/api/v1Literals.spec.ts` 用 ESLint Node API 跑仓里的配置 | web |
| **F7** | DB 作业里 `testdb` 不许跳过：设了 `KUN_REQUIRE_TEST_DB=1` 而没有 DSN 就失败，而不是 skip | Go 测试辅助 | db |
| **G12** | oasdiff 对比 master 上的 spec。preview 期只报告，稳定后阻断 | CI | api |

`apiv1.Setup` 在全部操作注册完之后跑一次 `sealDocument`，文档由它补齐五件事，别的一概不改：G4 的推导状态码；封闭枚举标记；没有 `omitempty` 的指针字段标为可空（huma 把指向结构体的指针渲染成裸 `$ref`，把指向 `DateTime` / `DecimalID` 等自定义 schema 类型的指针渲染成非空，而代码对它们发 `null`）；可空枚举的 `enum` 补上 `null`（huma 渲染成 `type: [string, null]` 而 `enum` 里没有 `null`，它自己的校验器先放过 null 所以从没发现，标准 JSON Schema 校验器会拒；W0a-5 契约测试在 `Image.sexual` 上抓到）；去掉值为 `true` 的 `additionalProperties`（与缺省语义相同，huma 的校验器对 `nil` 与 `true` 一视同仁，所以请求体仍接受未知字段；但 openapi-typescript 把 `true` 渲染成 `[key: string]: unknown`，W0b-1 发现读错字段名只得到 `unknown` 而不报错）。infra 的同类后处理还会强制数组非空、把错误响应改写成 Problem，论坛**不做**这两件：它们会让 G9 与 G4 在真实文档上永远不红，掩盖代码与文档的分歧。

源码门（G5 的 AST 扫描、G9 的 `omitempty` 扫描、F8）与它们的阳性对照走同一个目录扫描函数：对照在临时目录里造违规文件再扫；必需目录扫到零个文件本身就是违规。

作业：

- **api**：现有 `test.yml` 的 unit 作业，加 spec 重生成与 diff。
- **db**：新作业。起 Postgres 与 Redis 服务容器，用仓里的引导脚本从零建库，`TEST_DATABASE_DSN` 显式指向该容器，`go test -count=1 -p 1 ./...`。之前 DB 测试在 CI 里全部「因缺席而绿」，从这一波起真跑。
- **web**：现有 `web.yml`，加 F2 / F4 / F5 / F6。路径过滤加上 `apps/api/openapi/**`，spec 一变网页门就跑。

本地门禁：`GOTOOLCHAIN=go1.26.1 make lint && GOTOOLCHAIN=go1.26.1 go test ./...`（见 memory `go-1.27-errcheck-broken`），网页 `pnpm -F web lint && pnpm -F web typecheck && pnpm -F web test`。

## §4 演进、退役与 App 兼容

**两个阶段**（同 infra 07 §3.0）：

| 阶段 | 规则 |
|---|---|
| **preview**（现在） | 任何变更都允许，包括删改。但**必须**写进 `docs/proj/api-v1/CHANGELOG.md` 并告知 App 侧。preview 免的是兼容义务，不是告知义务 |
| **稳定**（用户宣布；前提是 App 首个公开版发布） | 只做加法；G12 转阻断；破坏只能走下面的流程 |

**稳定后怎么做不兼容变更**：

1. **先加**：新字段 / 新端点与旧的并存。旧的在 spec 标 `deprecated: true`，响应带 `Deprecation`、`Sunset` 与 `Link rel="deprecation"`。
2. **再迁**：发一个用新形状的 App 版本。
3. **看数据**：按 `User-Agent: kungal-app/<版本>` 统计，旧版本还有多少流量在打旧形状。
4. **收口**：旧版本流量可以接受时，调高 `/api/v1/app/version` 的 `min_version`，旧 App 弹强制更新。
5. **后删**：到 `Sunset` 后旧的回 `410 GONE`。

整体性的大改（换数据模型）按单个资源开新路径，不整体升 `/api/v2`。

**客户端契约三句**（写进 App 仓文档，有约束力）：

> 客户端必须忽略响应中未知的字段。
> 客户端必须容忍开放词表中未见过的取值；遇到未知的正文节点类型，渲染其子节点或纯文本。
> 客户端必须为未知的错误 `code` 准备一个按 `status` 的兜底分支。

## §5 逐端点迁移清单

每条端点（或一组同资源端点）迁移时，下面全部做完才算完：

1. **普查**：
   - 调用方：网页组件、`server/utils` 下的 Nitro 服务端调用、App 文档里的引用；
   - 取值：每个枚举成员对应的数据库行数，写进任务书；
   - 语义：每个字段到底是什么意思，不凭名字猜。
2. **v1 操作**：
   - 声明全部状态码与错误 code；
   - 类型走 v1 的表示层（id 字符串、Image、UserRef、`viewer`）；
   - **不得复用旧 DTO**，映射从 model / service 结果直接出。
3. **服务层**：
   - 旧 handler 与 v1 共用 service；
   - service 返回领域结构而不是某一代的 DTO；
   - 错误改成具名 problem，不再返回 `233`。
4. **网页**：
   - 调用点切到类型化客户端；
   - 新 code 的 `zh-CN` 译文进目录；
   - 手写类型删掉或改成生成物别名。
5. **删旧**：
   - 删旧路由、handler、DTO、手写 TS 类型；
   - `rg` 证明零调用方，含 `server/` 下的 Nitro 调用；
   - `routes.golden` 重生成；`legacy_route_baseline` 下调。
6. **测试**：
   - 契约测试覆盖每个声明的错误 code，至少各一个用例；
   - 分页全量遍历对照：小 `limit` 翻完全部页，与直接 SQL 的排序结果逐条相等，无重复无遗漏；
   - 网页 typecheck、lint、test 全绿。
7. **Claude 验收**：
   - 读 diff；
   - 在 dev 环境实测：匿名、普通用户、Bearer、有权限者各走一遍；
   - 浏览器里走一遍网页调用点；
   - 核对 spec 与普查结论一致。
8. **文档**：
   - App 相关的变化写进 `CHANGELOG.md`；
   - 本目录波次看板更新状态。
