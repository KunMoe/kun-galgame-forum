# T3 · 话题抽奖

> 话题轨第三段，2026-09-23。普查见 [census/polls-lottery-drafts.md](census/polls-lottery-drafts.md) §2、§9。
> 本文是实现的**唯一依据**，契约提交里不含一行实现代码；变异题（§9）与本文同一个提交，早于实现。

## 1. 生产取值（2026-09-23 实测）

| 问什么 | 值 | 决定了什么 |
|---|---|---|
| 抽奖总数 / 状态 | **2 个，都是 `drawn`** | 没有进行中的抽奖会被 v1 的行为改变卡住 |
| `entry_mode` / `draw_mode` | 都是 `signup` / `deadline` | 其余模式零行，但**都已在建抽奖表单里提供**。v1 保留全部玩法——删掉哪一个是产品决定，不是 API 决定 |
| 奖项 | `point` + `fixed` 1 个（5 名额）、`manual` 1 个（10 名额） | `delivery=code` 从未用过；`split` / `random` 零行 |
| 托管兑换码 | **0 条** | 兑换码路径全部是潜伏行为，没有存量数据要迁 |
| 参与记录 | 74 条，单个抽奖最多 62 人 | 参与名单改游标分页不影响任何人 |
| `fulfillment` | pending 7 / shipped 5 / received 2 / forfeited 1 | 四个值都在用；历史行不改写 |
| 凭空铸币 | 350 萌萌点，5 人 | §5 的出资改制不追溯 |
| 带图奖项 | 1 个 | reference-ping 的洞（§7）当前风险面是 1 行 |

`KUN_LOTTERY_CODE_KEY`：用户说已在 Dokploy 面板设置，**但它从未进入容器**——`docker-compose.prod.yml` 的 `x-kungal-api-env` 锚点没有引用它（与 `KUN_NEWS_API_KEY`、`LINK_CHECKER_*`、`KUN_TRUST_*` 同一个坑）。本波在锚点里补一行；由于 Dokploy 的触发器是 `tag` 而仓库没有 tag，宿主机的 compose 检出停在 `2965dc8`，**这一行要真的走一次 Dokploy 部署才生效**（见 §10）。

## 2. 范围

11 条旧路由 → 9 个 v1 操作。旧路由在同一个 PR 里删除，`legacy_route_baseline` 283 → 272。

| 旧 | v1 |
|---|---|
| `GET /topic/:tid/lottery/topic` | `GET /api/v1/topics/{topic_id}/lotteries`（不分页，每话题上限 10） |
| —（新增） | `GET /api/v1/lotteries/{lottery_id}` |
| `POST /topic/:tid/lottery` | `POST /api/v1/topics/{topic_id}/lotteries`，**必填 `Idempotency-Key`**，201 + `Location` + `Lottery` |
| `PUT /topic/:tid/lottery` | `PATCH /api/v1/lotteries/{lottery_id}`（真部分更新） |
| `POST …/draw`、`POST …/cancel` | 同一个 `PATCH`，`{"state": "drawn" \| "cancelled"}`（infra 06 §1.2） |
| `DELETE /topic/:tid/lottery` | `DELETE /api/v1/lotteries/{lottery_id}`，204 |
| `POST …/enter` | `PUT /api/v1/lotteries/{lottery_id}/entries/me`（K16 槽位），200 + `Lottery` |
| `POST …/withdraw` | `DELETE /api/v1/lotteries/{lottery_id}/entries/me`，200 + `Lottery` |
| `GET …/entrants` | `GET /api/v1/lotteries/{lottery_id}/entries`，**游标集合** |
| `PUT …/fulfillment` | `PATCH /api/v1/lotteries/{lottery_id}/winners/{winner_id}`，200 + `LotteryWinner` |
| `POST …/claim` | `POST /api/v1/lotteries/{lottery_id}/code-reveals`，200 + `LotteryCodeReveal` |

标识符**只从路径来**（普查 §0.1：旧的 `:tid` 在 11 条里一次都没被读过）。

## 3. 形状

### 3.1 `Lottery`

`object: "lottery"`；`id`、`topic_id` 是 `repr.DecimalID`；`author` 是 `UserRef`。

| 字段 | 说明 |
|---|---|
| `title`、`description` | `description` 空串下发 `""` |
| `entry_mode` | `signup` \| `reply` \| `floor` |
| `floor_rule` | `string \| null`。只有 `floor` 模式有值（`8,18,28` 或 `every:N`） |
| `draw_mode` | `deadline` \| `manual` \| `threshold` |
| `draw_threshold` | `integer \| null`。只有 `threshold` 模式有值 |
| `closes_at` | 旧 `deadline`。`DateTime \| null`，与 `Poll.closes_at` 同型 |
| `min_account_age_days`、`min_moemoepoint` | 整数，0 表示不限 |
| `show_entrants` | bool |
| `state` | `open` \| `drawing` \| `drawn` \| `cancelled`（旧 `status`，01 §命名表） |
| `seed_hash` | `string \| null`。楼层抽奖为 null（没有随机性，不宣称一个不存在的证明） |
| `seed` | `string \| null`。**开奖前恒为 null**，楼层抽奖恒为 null |
| `entry_count`、`slot_count` | 旧 `entry_count` / `total_slots` |
| `drawn_at` | `DateTime \| null` |
| `prizes` | `[LotteryPrize]`，按作者给的顺序 |
| `winners` | `[LotteryWinner]`，开奖前为 `[]`；作者被封禁的中奖者不出现 |
| `created_at`、`updated_at` | |
| `viewer` | `LotteryViewer \| null`，匿名为 null |

**查询参数 `include_nsfw`**（两个读面都有，默认 false）：取代旧面读偏好 cookie 的 `utils.IsSFW(c)`（01 §3：偏好 cookie 不再是输入）。

### 3.2 `LotteryPrize`

`object: "lottery_prize"`、`id`、`name`、`description`、`images`、`delivery`、`point_mode`、`point_amount`、`point_budget`、`slot_count`、`code_count`。

- `delivery`：`code` \| `offline` \| `point`。**旧值 `manual` 改名 `offline`**：`draw_mode` 也有一个 `manual`，同一个词两个含义（普查 §7）。存储不改，映射在边界上做。
- `point_mode`：`fixed` \| `split` \| `random` \| null（非萌萌点奖项为 null）。
- `point_amount`：`integer | null`。`fixed` 时是每人的数，`split` / `random` 时是整个池子。
- `point_budget`：`integer | null`。这个奖项一共要发出去的萌萌点（`fixed` = 数 × 名额，池子 = 池子）。§5 的托管按它算。
- `code_count`：`integer | null`。只有 `code` 奖项有值，是托管了几个码。**码本身永远不在任何读面里**（§6）。
- `images: [LotteryPrizeImage]`，取代旧的四个下标对齐的平行数组（`image_hashes` / `image_urls` / `nsfw_hashes` / `machine_nsfw_hashes`）：

  `LotteryPrizeImage { hash, url: string | null, is_marked_adult: bool, is_graded_explicit: bool }`
  - `is_marked_adult`：作者标的；`is_graded_explicit`：图床分级器判的（只叠加，作者取消不掉）。
  - 两者任一为真且请求没带 `include_nsfw=true` → **`url` 为 null**。`hash` 照发：作者的编辑表单要把整组图原样写回，返回一个变短的列表 = SFW 模式下点一次保存就静默删图（memory「奖品图片的 NSFW 处理」实测过）。

### 3.3 `LotteryWinner`

`object: "lottery_winner"`、`id`（= 参与记录 id，路径参数 `{winner_id}` 与它对应）、`prize_id`、`winner: UserRef`、`reply_floor: integer | null`（楼层抽奖才有）、`rank_key: string | null`（楼层抽奖为 null）、`fulfillment`、`point_awarded`（非萌萌点奖项为 0）、`won_at`、`claim_expires_at: DateTime | null`（旧 `claim_deadline`，只有兑换码奖项有值）。

`fulfillment`：`pending` \| `shipped` \| `received` \| `forfeited`。

### 3.4 `LotteryViewer`

| 字段 | 说明 |
|---|---|
| `has_entered` | |
| `can_enter` | 与 `PUT entries/me` 的服务端闸**同源**（旧 `entryBlocker` 的设计保留） |
| `enter_blocked_reason` | 封闭枚举或 null：`not_open` `past_closes_at` `no_signup` `own_lottery` `reply_required` `moemoepoint_below_minimum` `account_too_new`。**取代旧的中文句子 `enter_blocked`**（§7 禁止服务端产出给人看的句子）。所需的数值客户端从 `min_moemoepoint` / `min_account_age_days` 取 |
| `can_edit`、`can_delete`、`can_draw`、`can_cancel` | 服务端算。**取代网页自己镜像的权限**：普查 §9.3 抓到网页把抽奖按钮挂在 `poll.create_any` 上，`lottery.*` 三个键全站一次都没被前端查过 |
| `can_view_entries` | `show_entrants` 或作者或 `lottery.view_restricted` |
| `can_manage_fulfillment` | 作者或 `lottery.manage_any`，且已开奖 |
| `winner_id` | `string \| null`。调用者自己中奖时是那条 `LotteryWinner.id`，其余信息从 `winners[]` 取 |
| `can_reveal_code` | 调用者赢了一个兑换码奖项，且未作废 |

旧的 `my_entry_id`、`my_prize_name`、`my_delivery`、`my_point_awarded`、`my_claim_deadline`、`my_code_ready` 全部由 `winner_id` + `winners[]` + `can_reveal_code` 覆盖，不再下发。

### 3.5 `LotteryEntry`（参与名单条目）

`object: "lottery_entry"`、`id`、`entrant: UserRef`、`reply_floor: integer | null`、`created_at`。被封禁的参与者不出现。

### 3.6 `LotteryCodeReveal`

`{object: "lottery_code_reveal", lottery_id, redemption_code}`。旧响应里 `data.code` 与信封顶层的业务码 `code` 同名（普查 §2.8），改名 `redemption_code`。

### 3.7 写面请求体

- 建 `LotteryCreate`：`title`、`description?`、`entry_mode`、`floor_rule?`、`draw_mode`、`draw_threshold?`、`closes_at?`、`min_account_age_days?`、`min_moemoepoint?`、`show_entrants?`（缺省 true）、`prizes: [LotteryPrizeInput]` 1–10。
- `LotteryPrizeInput`：`name`、`description?`、`image_hashes?`（≤9）、`adult_image_hashes?`（必须 ⊆ `image_hashes`，旧 `nsfw_hashes`）、`delivery`、`point_mode?`、`point_amount?`、`slot_count`（1–500）、`codes?`（≤500，每个 1–200）。
- 改 `LotteryPatch`：以上标量全部可选，只改传来的；`closes_at: null` 清除；`prizes` 传了就是**整组替换**；另有 `state?: "drawn" | "cancelled"`。**`state` 不得与任何其它字段同时出现** → 422 `/state` `INCONSISTENT_WITH`。
- 履约 `LotteryWinnerPatch`：`{fulfillment}`。

## 4. 逐条裁决（对普查 §2 与 §9）

| 普查 | 裁决 |
|---|---|
| §0.2 读面不查可见性 | 旧面已修。v1 一律走 `visibleTopic`（含 K17 的作者封禁判定）；**抽奖作者被封禁时该抽奖也不出现**（普查 §2.10：投票会消失、抽奖不会，两个小程序给相反答案——对齐投票） |
| §2.1 萌萌点凭空铸币 | **用户 2026-09-23 裁决：发起人出资**。见 §5 |
| §2.1 奖品图不在 reference-ping 扫描面内 | 修。见 §7 |
| §2.1 `count, _ :=` 吞错 | 错误上抛 500；上限 10 保留 → `409 LOTTERY_LIMIT_REACHED` |
| §2.1 反诈门槛 | 保留（30 天或 100 萌萌点，`lottery.create_any` 绕过）→ `403 LOTTERY_CREATOR_INELIGIBLE`，扩展 `min_account_age_days` / `min_moemoepoint` |
| §2.2 不传奖项时整个形状校验被跳过 | **改的结果（库里的值合并本次补丁）整体过一遍形状校验**，不论传没传奖项。「deadline 模式却没有 closes_at」「阈值小于名额」「楼层规则解析不了」全部 422 |
| §2.2 / §1.1 deadline 解析失败静默变 null | `closes_at` 是 `repr.DateTime`，格式错由 schema 拒绝（400/422），**绝不静默变 null** |
| §2.2 已有参与后改 `entry_mode` / 奖项 | `422` + `/entry_mode` 或 `/prizes` + `IMMUTABLE`（与投票的 `is_anonymous` 同一条规则） |
| §2.3 可以删一个正在开奖的抽奖 | `drawing` 状态任何人都删不了 → `409 LOTTERY_DRAWN` |
| §2.3 作者删已开奖的 | `409 LOTTERY_DRAWN`（中奖者还要靠它领奖）。版主仍可删，行为与旧面相同 |
| §2.4 事务里任何错误都翻译成「您可能已经参与过了」 | `INSERT … ON CONFLICT DO NOTHING`：已参与 = 200 幂等；其余错误 500 |
| §2.4 参与资格 | 同源函数产出封闭枚举：`409 LOTTERY_CLOSED`（`not_open` / `past_closes_at`）或 `403 LOTTERY_INELIGIBLE`（扩展 `reason`：`no_signup` `own_lottery` `reply_required` `moemoepoint_below_minimum` `account_too_new`） |
| §2.5 退出不查可见性、不查截止、删 0 行也 200 | 查可见性；过了 `closes_at` 或非 open → `409 LOTTERY_CLOSED`；**没参与过 → 200 无副作用**（K16） |
| §2.6 事务错误把 Postgres 原文拼进响应 | 开奖失败 → `500 INTERNAL_ERROR`，detail 不带任何内部信息（K8 第 5 条） |
| §2.6 `afterDraw` 在事务之外 | 不改。通知与萌萌点本来就是提交后的副作用；中奖发放用稳定幂等键 `kungal:lottery_won:<lottery>_<user>`，可人工重放 |
| §2.6 `ref = lottery_5` 不走 `Ref()` | 改 `moemoepoint.Ref("topic_lottery", id)`。幂等键不变 |
| §2.6 楼层 + `random` 池子时 seed 为空，谁分多少任何人都能提前算 | `422` `/prizes/N/point_mode` `INCONSISTENT_WITH`：楼层抽奖不能用拼手气池子 |
| §2.7 `Cancel` 的 UPDATE 没有状态守卫 | `UPDATE … WHERE id = ? AND status = 'open'`，没动到行 → `409 INVALID_STATE_TRANSITION` |
| §2.8 「只能揭示一次」是假的 | **行为不变（可重复揭示）、改文案**。一次性揭示会让关掉页面的中奖者永久丢码；码本来就只属于他一个人 |
| §2.8 `UpdateEntryFields` 返回值被 `_ =` 丢掉 | 推进失败 → 500，不返回码 |
| §2.8 领码不查话题可见性 | **有意保留**：话题被隐藏时中奖者仍能领自己的码，否则 7 天领取期会在隐藏期间静默作废。非中奖者一律 404，所以不构成探测面 |
| §2.8 POST 没有幂等键 | **有意不接受 `Idempotency-Key`**：幂等存储把响应体在 Redis 里存 24 小时，那等于把明文码落盘。这个面天然幂等（同一个人每次拿到同一个码）。记为 K22 |
| §2.9 履约没有状态机 | 见 §4.1 |
| §2.9 `FindWinners` 拉全表再线性找 | 按 `(lottery_id, id, prize_id > 0)` 取一行 |
| §2.10 `enter_blocked` 是中文句子 | `viewer.enter_blocked_reason` 封闭枚举 |
| §2.10 `FindByTopicID` 无 id 决胜键 | `created DESC, id DESC` |
| §2.11 `show_entrants=false` 回 200 + 空数组 | `403 PERMISSION_REQUIRED`（W5b 同一条） |
| §2.11 参与名单无分页 | 游标集合，排序 `created ASC, id ASC`，`limit` 1–100 默认 20 |
| §9.2 参与名单丢了 `reply_floor` | 下发，网页楼层抽奖显示楼层 |
| §9.3 抽奖按钮挂在投票权限键上 | `viewer.can_*`，网页不再自己算 |
| §9.5 打开再保存挪动截止时间 | 网页改用 `closesAtFromPicker`；PATCH 只发改过的字段，不再整包回写 |
| §9.8 揭示后的码从不清空 | 网页改成关闭即清 |

### 4.1 履约状态机

- `received`、`forfeited` 是终态。
- **兑换码与萌萌点奖项由系统推进**：揭示 → `received`、领取期满 → `forfeited`（兑换码）；萌萌点在开奖时发放，**v1 开奖时直接记为 `received`**（旧面记 `pending`，于是界面对已到账的萌萌点显示「待发放」）。对这两类奖项的任何 `PATCH` → `409 INVALID_STATE_TRANSITION`。历史行不改写。
- **线下奖项**（`offline`）：

| 从 → 到 | 作者 / `lottery.manage_any` | 中奖者本人 |
|---|---|---|
| `pending` → `shipped` | ✓ | ✗ |
| `shipped` → `pending` | ✓（撤销误点） | ✗ |
| `pending` / `shipped` → `received` | ✓ | ✓ |
| `pending` / `shipped` → `forfeited` | ✓ | ✓ |

- 设成当前值 → 200 无副作用。其余 → `409 INVALID_STATE_TRANSITION`，detail 写当前状态与允许的目标。权限不够 → `403 PERMISSION_REQUIRED`。
- 中奖者改自己的履约**不要求话题可见**（同领码）；作者与版主的路径要求可见。

## 5. 萌萌点奖项由发起人出资（用户 2026-09-23 裁决）

新列 `topic_lottery.point_escrow`：本抽奖当前从发起人那里收着的萌萌点。

- **建**：`escrow = Σ point_budget`（≤ 100000 的旧上限保留）。事务内 `SELECT … FOR UPDATE` 锁住发起人的 `kungal_user_state.moemoepoint`，余额 < escrow → `403 MOEMOEPOINT_INSUFFICIENT`（`required` = escrow）；否则**在同一事务里把缓存余额减掉 escrow**，两个并发的建抽奖串行在这一行锁上，第二个看得到减过的余额。提交后向 OAuth 发 `-escrow`，理由 `content_removed`，ref `topic_lottery_escrow:<id>`，幂等键 `kungal:lottery_escrow:topic_lottery_<id>`。版主建也照扣：出资的是抽奖的作者。
- **改奖项**（仅零参与时）：差额 `Δ = 新 escrow − 旧 escrow`。Δ > 0 按建的规则再扣（余额不足 403）；Δ < 0 退回；结果写回列。重放同一个 PATCH 算出 Δ = 0，不重复扣。
- **取消 / 删除未开奖的**：退回整个 escrow，**同一事务把列清零**，所以取消后再删不会退第二次。
- **开奖**：退回 `escrow − Σ point_awarded`（名额没发满的部分；拼手气与平分池只要有一个中奖者就花光），开奖事务把列清零。
- 退回走 `content_approved` + ref `topic_lottery_escrow:<id>`，幂等键 `kungal:lottery_escrow_refund:topic_lottery_<id>`（开奖、取消、删除三者互斥，一个抽奖一辈子只退一次）；改奖项的差额用带 nonce 的键。
- 萌萌点明细：`content_removed:topic_lottery_escrow` → 「抽奖奖池托管」、`content_approved:topic_lottery_escrow` → 「抽奖奖池退回」、`content_approved:topic_lottery` → 「抽奖中奖」（memory「moemoepoint debit reason」的 ref-kind 改标签法，与推话题同做法）。
- **历史抽奖 `point_escrow = 0`**，开奖时的退款按 `max(0, …)` 算，不追溯扣任何人。
- **残余风险，照实写**：推送 OAuth 是 best-effort（与推话题相同），OAuth 暂时不可用时这一笔会丢，只留一条带幂等键的日志供重放。真正的「不许透支」闸在 infra：`RequireNonNegative` 目前只给 infra 自己的改名扣费用，没有暴露给 s2s。要彻底封死需要 infra 把它开放给下游，不属于本波。

## 6. 兑换码的四道防线（原样保留，普查 §2.0）

1. `model.TopicLotteryCode` 整个 struct 没有 json tag；
2. `code_id` 不进任何响应，只下发 `viewer.can_reveal_code`；
3. 唯一返回明文的是 `POST …/code-reveals`，**永远不能变成 GET**（Nuxt 会把页面加载时取回的数据内联进 `__NUXT__`）；它也**不接受幂等键**（§4 K22）；
4. 迁移 083 的 `COMMENT ON TABLE`。

另加一条测试：对全部 9 个操作的全部响应体做子串断言，任何一个都不得出现种子码的明文。

## 7. 预分配

- **迁移 111**（话题轨号段 110–119）：
  - `topic_lottery.point_escrow integer NOT NULL DEFAULT 0`（纯加列，历史行 0）；
  - `topic_lottery_entry (lottery_id, created, id)` 索引，供参与名单键集分页。
  - 纯加、幂等，**随部署自动跑**，先迁移再部署的顺序由 migrate 服务保证。
- **错误码 7 个**（`kungal` 域，三处一译）：

| code | status | 什么时候 |
|---|---|---|
| `LOTTERY_CLOSED` | 409 | 操作要求抽奖开放，而它已开奖 / 已取消 / 正在开奖 / 过了 `closes_at` |
| `LOTTERY_INELIGIBLE` | 403 | 调用者不满足参与条件；扩展 `reason` |
| `LOTTERY_CREATOR_INELIGIBLE` | 403 | 发起人不满足反诈门槛；扩展 `min_account_age_days`、`min_moemoepoint` |
| `LOTTERY_LIMIT_REACHED` | 409 | 本话题的抽奖已达上限；扩展 `limit` |
| `LOTTERY_DRAWN` | 409 | 删除一个正在开奖或（非版主删）已开奖的抽奖 |
| `REDEMPTION_CODE_FORFEITED` | 409 | 兑换码已作废（领取期满或中奖者放弃） |
| `INVALID_STATE_TRANSITION` | 409 | 抽奖状态或履约状态不允许这次迁移（infra 10 §3.3 同名同义） |

  复用：`NOT_FOUND`、`VALIDATION_FAILED`、`PERMISSION_REQUIRED`、`MOEMOEPOINT_INSUFFICIENT`、`CONTENT_REJECTED`、`SERVICE_UNAVAILABLE`、`IDEMPOTENCY_*`。
- **reference-ping**：`collectContentImageHashes` 只扫文本列里的 `/image/<hash>` token；奖品图是 jsonb 里的**裸 hash**，两重都不命中。加一个显式来源：`topic_lottery_prize.image_hashes`（`jsonb_array_elements_text`）。
- **compose**：`x-kungal-api-env` 加 `KUN_LOTTERY_CODE_KEY: ${KUN_LOTTERY_CODE_KEY:-}`。
- **新 K 决定**：
  - **K22**：返回秘密的 `POST`（目前只有 `code-reveals`）不接受 `Idempotency-Key`。K12 的「全部 POST 必须支持」对它不适用：幂等存储会把响应体在 Redis 里存 24 小时。

## 8. 网页

- `useLottery.ts` 全部改走类型化客户端；`shared/types/topic-lottery.ts` 删掉，改用生成类型的别名。
- 列表读面用 `useAsyncData` + 类型化客户端，`include_nsfw` 取自用户偏好。
- 表单：`slots` → `slot_count`、`nsfw_hashes` → `adult_image_hashes`、`manual` → `offline`（仅线上值，界面文案不变）、`deadline` → `closes_at`（走 `closesAtFromPicker`）；编辑只发改过的字段，奖项只在零参与时整组发。
- 按钮与参与状态全部读 `viewer`；`enter_blocked_reason` 的中文放在网页的常量表里。
- 领码后显示的码在关闭时清空；「只能揭示一次」文案改掉。
- 萌萌点明细加三个标签（§5）。
- 七个错误码的 `zh-CN` 译文。

## 9. 变异题（先于实现提交）

| # | 改动 | 必须变红的断言 |
|---|---|---|
| 1 | `GET /lotteries/{id}` 不查话题可见性 | 读不到的话题下的抽奖 → 404 |
| 2 | 领码不查「调用者是这个码的中奖者」（跳过 `prize_id > 0`） | 没中奖的参与者领码 → 404，且响应里没有明文 |
| 3 | 建抽奖不查发起人余额 | 余额不足 → 403 `MOEMOEPOINT_INSUFFICIENT` 且没有建出抽奖 |
| 4 | 取消时不把 `point_escrow` 清零 | 取消后再删：只记录到一笔退款 |
| 5 | 开奖退款按整个 escrow 算，不减已发放 | 名额没发满时退款 = escrow − 已发放 |
| 6 | `state: "cancelled"` 不查当前状态 | 已开奖的抽奖取消 → 409 `INVALID_STATE_TRANSITION` |
| 7 | 履约允许离开终态 | `received` → `pending` → 409 `INVALID_STATE_TRANSITION` |
| 8 | 参与名单游标去掉 id 决胜键 | 全量遍历（**种子里有同一时刻的参与记录跨页**）出现重复或遗漏 |
| 9 | `show_entrants=false` 时回 200 空列表 | 非作者 → 403 `PERMISSION_REQUIRED` |
| 10 | PATCH 只在传了奖项时才做形状校验 | `{draw_mode: "deadline", closes_at: null}` → 422 |
| 11 | 参与资格跳过「要求先回帖」 | 未回帖参与 `reply` 抽奖 → 403 `LOTTERY_INELIGIBLE`，`reason = reply_required` |
| 12 | 开奖前下发 `seed` | open 状态 `seed` 必须是 null |
| 13 | SFW 读者拿到成人图的 URL | 不带 `include_nsfw` 时被标记的图 `url` 为 null，`hash` 仍在 |

**编译不过的变异是废题**，按等价破坏法重写（W5a/T2 各踩过一次）。

## 10. 上线

- 迁移 111 随部署自动跑，**先迁移后启动**由 compose 的 `migrate` 服务保证。
- **`KUN_LOTTERY_CODE_KEY` 要走一次 Dokploy 面板部署**才能进容器：手动 `docker compose pull && up` 用的是宿主机上冻结在 `2965dc8` 的旧 compose 文件。
- 旧路由删掉后，`/api/topic/**` 只剩 0 条；话题轨余下 T4（`/admin/topic*` 3 条）。
