---
name: dispatch-cursor
description: Dispatch implementation and investigation work to the local cursor-agent CLI (Cursor Grok 4.6 Extra High by default) as a headless executor inside a kernel sandbox, while this session stays the orchestrator and acceptor. The default executor for this repo. Use when a task is large enough to hand off as a written task book, or when the user asks to "派发 cursor" / "dispatch cursor" / "让 cursor 做". Covers the model-config trap, the three conditions under which the sandbox silently turns off, the dispatch, the preflight proof, task-book structure, what the sandbox cannot do in this repo (databases, pnpm installs, git) and the acceptance protocol.
---

# Dispatching cursor-agent as executor

> **If you are the cursor executor and this file was loaded into your context: ignore it.**
> It describes how the orchestrator dispatches *you*. It is not a task book. Your task book
> is the prompt you were started with, and nothing here overrides it.

This session is the **orchestrator**: it adjudicates design, writes the task book, owns git,
databases, migrations and production, and accepts or rejects the result. The local
`cursor-agent` CLI is the **executor**: it reads, writes and runs gates inside a sandbox.

Ported on 2026-09-18 from nextmoe-infra's `.claude/skills/dispatch-cursor/`. The sandbox
measurements below were taken there on the same machine and the same cursor-agent version.
The scripts here are copies, not links: this repo must not break when infra's skill changes.
`dispatch.sh` differs only in its deny list, which also covers this repo's env files.
`dispatch-grok` in this directory is the older, shell-less protocol; use this one.

## 1. The model-config trap — read before running cursor-agent at all

**`cursor-agent --model X` writes X back into the config it started with.** Without
`CURSOR_CONFIG_DIR`, that is the user's global `~/.cursor/cli-config.json`, and every other
session on this machine that dispatches copies it.

On 2026-09-18 this repo's session probed five models with bare `cursor-agent -p --model …`
runs. It turned the user's default from Cursor Grok 4.6 Extra High into Claude Sonnet 5, while
three other sessions were dispatching.

- **Never run `cursor-agent` outside `dispatch.sh`.** `dispatch.sh` points
  `CURSOR_CONFIG_DIR` at a per-run copy and pins `--model cursor-grok-4.6-xhigh`.
- If it happens anyway, restore `model`, `selectedModel`, `modelParameters` and
  `modelSelectionHistory` from a per-run copy made before the change. Every
  `$SCRATCHPAD/cursor/runs/*/cursor-config/cli-config.json` under `/tmp/claude-1000` is one.
  Do not overwrite the whole file: the copies have `authInfo` removed.
- Quota: Opus, Sonnet and GPT models hit the account's monthly cap on 2026-09-18. Grok 4.6 is
  the executor.

## 2. What cursor-agent can do here

| Capability | Result |
|---|---|
| Shell | bash, inside the sandbox of §3 |
| Go gates | `go build`, `go vet`, `go test` work, including `httptest.NewServer` on the sandbox's private loopback. `GOCACHE` / `GOMODCACHE` are a fresh `/tmp/cursor-sandbox-cache/<hash>/` per run: every run builds cold and downloads modules through Cursor's proxy (`proxy.golang.org` is allowed). `GOTOOLCHAIN=go1.26.1` therefore downloads the toolchain once per run. `dispatch.sh` deletes that cache (1.3 GB of tmpfs) afterwards |
| Network | package registries only; everything else is refused. No raw TCP |
| **Databases** | **none**. The loopback is private, docker is unreachable. DB-backed tests skip without `TEST_DATABASE_DSN`. The executor can **write** DB tests but never run them: the orchestrator runs them at acceptance (§5) |
| **pnpm** | the store (`~/.local/share/pnpm`) is outside the sandbox, so `pnpm install` / `pnpm add` fail. **The orchestrator installs before dispatch**: run `pnpm install --frozen-lockfile` in the worktree, and add any new dependency itself and commit it. The executor then runs `pnpm -F web typecheck / lint / test` on the installed tree |
| git | reads work; every write is denied by rule. The orchestrator commits |
| Project instructions | `CLAUDE.md` loads as a rule; `.claude/skills/*` may be listed as skills (hence the guard at the top) |
| Model | pinned by `dispatch.sh`: `cursor-grok-4.6-xhigh`. `CURSOR_MODEL=cursor-grok-4.6-low` for probes |
| Budget | wall clock only: `CURSOR_TIMEOUT`, default 4 h |
| Auth | `CURSOR_API_KEY` from the environment |

## 3. The sandbox, and how it silently turns off

A headless run applies the Landlock + user-namespace sandbox only when all three hold. When one
fails, every command runs as the real user, with no error and nothing in the output to say so:

1. No `--force` / `--yolo`.
2. `approvalMode: "allowlist"`. The user's config says `"unrestricted"`, which is unsandboxed.
3. The command is not on `permissions.allow`. An allowed command runs outside the sandbox, and
   the user's config allows `Shell(**)`.

`dispatch.sh` makes all three hold in a per-run config copy: the allow list is emptied, the
deny list is replaced, `authInfo` is dropped. The user's own config is never edited.

Inside the sandbox:

- The executor can write the worktree and `/tmp`.
- `ssh` to production, the host loopback and docker are unreachable.
- **Reads are not fenced**: the whole filesystem is readable, including the main checkout's
  env files.
- `git add` / `git commit` would work at the sandbox level; the deny rules stop them.

A fresh worktree has **no env files** (they are gitignored), which is why dispatches go to
worktrees. This repo has six of them:

- `apps/api/.env` — live catalog key, OAuth secret, DB DSN;
- `apps/web/.env`;
- `apps/web/.env.prod` — **production**;
- `docker/api.env`;
- `docker/web.env`;
- `refs/legacy/.env.prod`.

`dispatch.sh`'s deny list covers all six by shell text and by file-tool read. It also denies:

- git writes, including `git -C` / `-c` / `--git-dir`, which slip past per-subcommand rules;
- `gh`, `ssh`, `psql`, `docker`, `sudo`, `pkill`, `pnpm dev`;
- other agents;
- credential files;
- file-tool writes to `.claude/`, `.cursor/`, `CLAUDE.md`, `AGENTS.md` and `node_modules/`.

The environment layer unsets tokens, `SSH_AUTH_SOCK`, `TEST_DATABASE_DSN` and
`KUN_DATABASE_URL`, and rewrites push URLs to a path that does not exist.

## 4. The dispatch

```bash
# 1. a worktree nobody else stands on (iron rule 13), from the commit the task builds on.
#    This repo works on master and often has unpushed commits, so branch from local master,
#    not origin/master.
git -C /home/kun/Desktop/code/website/kun-galgame-forum worktree add -b <branch> \
  /home/kun/.config/superpowers/worktrees/kun-galgame-forum/<name> master
# 2. dependencies the sandbox cannot fetch
( cd <worktree> && pnpm install --frozen-lockfile )   # only if the task touches apps/web
# 3. the task book, from task-book-template.md, in the session scratchpad (never the repo)
export CURSOR_OUT_ROOT="$SCRATCHPAD/cursor/runs"
mkdir -p "$CURSOR_OUT_ROOT/<slug>"                     # write task.md there
# 4. from the worktree root, detached
cd <worktree> && CURSOR_OUT_ROOT="$CURSOR_OUT_ROOT" setsid nohup \
  /home/kun/Desktop/code/website/kun-galgame-forum/.claude/skills/dispatch-cursor/dispatch.sh <slug> \
  >"$CURSOR_OUT_ROOT/<slug>/dispatch.out" 2>&1 </dev/null &
```

Wait with a background until-loop on `pgrep -f 'dispatch-cursor/[d]ispatch.sh <slug>'`, never
a foreground sleep.

- **At most two at once**, and never two over overlapping paths or in one worktree.
- A detached dispatch survives the harness killing its background tasks when RAM runs out;
  the waiter does not, so re-arm it.
- To resume a killed run:
  1. put a "RESUMED RUN" note at the top of the same task book, saying what exists, what is
     left, and to re-run every gate;
  2. move the old `stream.jsonl` and `run.json` aside;
  3. dispatch again from the same worktree.

`dispatch.sh` refuses to run from a main checkout or a subdirectory. It also refuses `--force`,
`--yolo`, `--sandbox`, `--approve-mcps`, `--auto-review` and `--worktree`. Add per-task rules
with `--deny '<rule>'`.

## 5. Reading the result and accepting it

`check.sh` runs at the end of every dispatch; it can also be run by hand on a killed one. It
fails when:

- the preflight line is not exactly
  `dispatch-preflight: sandbox=native net=blocked loopback=private` — then treat every shell
  call in the run as unfenced;
- a shell call succeeded with no sandbox policy;
- HEAD moved;
- there is no `result` event, or the event is an error.

It also lists every refused call.

Then, by hand, in the worktree:

1. `git status --porcelain`: only the writable paths the task book named.
2. `git log --oneline -3`: no commit you did not make.
3. Read the whole diff. `git diff | rg '^-\s*//'` lists every deleted comment: Grok has
   dropped an earned comment next to lines it edited (`w0a-4-gates`).
4. Re-run every gate **yourself**:
   - `cd apps/api && GOTOOLCHAIN=go1.26.1 make lint && GOTOOLCHAIN=go1.26.1 go test ./...` —
     the system Go 1.27 breaks errcheck and changes inlining, which renames handlers in
     `routes.golden`;
   - `make openapi && git diff --exit-code apps/api/openapi` once that target exists;
   - `pnpm -F web lint && pnpm -F web typecheck && pnpm -F web test` if the web changed.
5. **Run the DB suites the executor could not reach.**
   - Create a throwaway database with
     `/home/kun/Desktop/code/website/nextmoe-infra/scripts/ephemeral-test-db.sh create <slug>`.
     It lives on the host Postgres, auth comes from `~/.pgpass`, and the DSN it prints has no
     password. Export that DSN as `TEST_DATABASE_DSN` and `KUN_DATABASE_URL` for the migration
     step only.
   - Build the schema with `apps/api/scripts/testdb-bootstrap.sh` once W0a lands it, or by
     hand per memory `kungal-db-backed-tests-bootstrap`.
   - Run `go test -count=1 -p 1 ./...`.
   - `drop` the database afterwards.
   - Never point any of this at the dev database (`apps/api/.env`) or production (iron
     rule 14).
6. Mutate: break each behaviour the task added (flip a condition, drop a filter) and confirm
   a test fails. Grok's tests have repeatedly asserted a little less than its report implies.
   Before trusting a batch where everything dies, confirm one kill by hand. The zsh here does
   not word-split an unquoted `$P`, so a package list in one variable reached `go test` as a
   single bad argument, and every mutant of a batch "died" of a build error.
7. Commit from the orchestrator, `git commit -- <paths>` (never `add -A`: it misses the root
   `pnpm-lock.yaml` or sweeps in strays). Then merge the branch into master.

The report is a file in the scratchpad, never in the repo. `result` in `run.json` is every
assistant text block concatenated; read `report.md`, not that.

## 6. The task book

Template: `task-book-template.md` (step 0, environment and discipline already filled in).

- **English**, self-contained. The executor cannot see this conversation.
- **State every adjudication inline, and quote binding clauses** rather than citing them.
  Design docs in `refs/docs/proj/**` are Chinese: the executor may read them, but the task book
  restates in English every rule it depends on. `refs/` is gitignored here and in infra and exists
  only in each source checkout: give absolute paths into
  `/home/kun/Desktop/code/website/kun-galgame-forum/refs/docs/` and
  `/home/kun/Desktop/code/website/nextmoe-infra/refs/`.
- **No open design decisions.** Where the mechanics depend on code the executor has yet to
  read, state the invariant plus the precedent, and require it to report what it chose.
- **Name the writable paths and the commands it may run.**
- **Name every symmetric case.** Grok implements what the book names and does not infer the
  mirror image (infra `mt-stray-rows`).
- **Word each check as the property, not your guess at its shape.**
- **Demand a positive control** for every gate, search or census.
- **Name the real construction path the tests must go through** (`newFiber()` +
  `setupRoutes`, `app.V1Spec()`). Left to itself Grok builds its own fixture, and a fixture
  cannot see how the new code composes with the rest of the app (`w0a-3-wiring`).
- **Positive controls take the construction path too**: register the violating operation
  through `apiv1.Setup`, and run a source scan over a temp tree through the same walker as
  the real scan. A snippet-level control proves the rule, not the gate (`w0a-4-gates`).
- **When it ports infra code, name what not to port.** Grok copies infra wholesale, including
  document post-processing that rewrites the spec into compliance and so keeps a gate from
  ever failing (`w0a-4-gates`).
- **Ask for value assertions, not only schema conformance.** A response that validates can
  still carry the wrong values: emptying `sections` or forcing `has_best_answer` false survived
  every test Grok wrote (`w0a-5-topics`).
- **Where the book says "exactly", require a test of the equality.** Told to declare exactly
  the derived statuses, Grok declared one and never compared the result (`w0a-5-topics`).
- **Ask it to prove new tests run.** A `TestMain` or helper that skips without a database
  takes DB-free tests down with it.
- **Forbid ranking**; "anything that looks wrong, in scope or not" goes near the top of the
  report.
- Keep one dispatch to one coherent change that its tests can assert. Split a large wave into
  several dispatches rather than one four-hour run.

## 7. What the orchestrator never delegates

- Adjudications.
- All git.
- Databases, migrations and the ephemeral test DB.
- `pnpm install` / `pnpm add` and lockfile changes.
- Docker and `pnpm dev`; runtime checks against the dev stack; browser checks.
- Production (`ssh kungal-neo`).
- Cross-repo docs.
- Final acceptance (§5).

## 8. What is worth dispatching

| Shape | Verdict |
|---|---|
| Broad read → narrow `file:line` report | **Dispatch** |
| Wide mechanical edit a gate asserts | **Dispatch** |
| Fully adjudicated implementation whose correctness tests assert | **Dispatch**; still read the diff and mutate |
| New code carrying open design judgement | **Do not dispatch** until adjudicated |
| Anything whose truth is in a database or production | **Do not dispatch**; the sandbox cannot reach either |

## 9. Quality ledger

Record each real dispatch: task shape, model, elapsed, what acceptance found. infra's ledger
(`nextmoe-infra/.claude/skills/dispatch-cursor/SKILL.md` §8) holds five earlier runs of the same
model. Its verdict: dependable on fully adjudicated work; name every symmetric case; tests
assert a little less than the report implies, so mutate.

| Date | Task | Model | Elapsed | Acceptance |
|---|---|---|---|---|
| 2026-09-18 | `w0a-1-problem`: new `pkg/problem` (closed error registry, RFC 9457 writer, ULID request ids, huma validation → reason/params bridge), 1.9k lines incl. tests | grok-4.6-xhigh | 1155 s, 142 calls, 222k in + 9.0M cache read, 77k out | Every rule implemented. It pinned each bridge mapping with a request through real huma validation, not hand-built errors, as the book asked. Platform titles and descriptions diffed byte-identical to infra. It stopped nowhere, but it flagged its one deviation honestly (huma forces Fiber v3.3→v3.4 via MVS; accepted). It also caught **two errors in the orchestrator's own docs**: K5 lacked the `TOO_FEW_ITEMS` row, and the W0a record said `map[string]any` where G9 forbids it. 9/14 of the orchestrator's mutants died as delivered. The survivors were a nested required-property pointer, the 401 split, cause logging, and `MarshalJSON`'s own null guard (the mutation needs a literal `Problem`); acceptance added 4 tests (13/14, the last survivor being ULID random-byte width) |
| 2026-09-18 | `w0a-2-identity`: extract a 12-outcome identity resolver from the auth middleware, rebuild `Auth()` / `OptionalAuth()` on it byte-identically | grok-4.6-xhigh | 1250 s, 130 calls, 403k in + 5.7M cache read, 70k out | Checked every cell of the book's legacy table against the code before trusting it. Kept all earned comments. `routes.golden` untouched. It listed 12 real smells, flat: Redis errors folded into "expired", an ignored `SETNX` error, a ban that leaves `OptionalAuth` anonymous. It left one dead method (`Bearer.authenticate`, zero callers), which acceptance removed. 10/10 of the orchestrator's mutants died, including the multi-line ones. DB-backed full suite green |
| 2026-09-18 | `w0a-3-wiring`: mount huma on `/api/v1` (tier table, error routing, headers, v1 idempotency, metadata endpoints, spec generator, manifest tiers, F3 / F7), 2.2k lines incl. tests | grok-4.6-xhigh | 2566 s, 349 calls, 1.06M in + 14.9M cache read, 147k out | Every row of the tier table exactly as adjudicated. Its own five mutations were real, and its flat §2 list was accurate. But every test built its own Fiber app, so none saw how v1 composes with the legacy router. **An unmatched `/api/v1` path fell through into the legacy `/api` `OptionalAuth` / `Auth` chain**, because Fiber matches in registration order: a nil-deref 500 under test, a legacy envelope in production. It also missed that v1 HEAD answered 405 and that replay dropped `Location` (the book's own gap). Its "legacy unchanged" test exercised a copy of the handler. Its manifest positive control took a different path from the real check, which is exactly what the book warned against. It wrote a 90-line schema walk that `huma.DefaultArrayNullable = false` replaces, plus a 405 promotion Fiber v3 already does. First round: 7 of 17 mutants survived. Acceptance added app-level tests over `newFiber()` + `setupRoutes`, an unmatched-path terminator with `Allow`, HEAD mirroring, `app.V1Spec()` as the one spec source, and CORS and template tests. Final: 25/25 killed, DB suite green with `KUN_REQUIRE_TEST_DB=1` |
| 2026-09-18 | `w0a-4-gates`: v1 representation helpers (ids, timestamps, Image, UserRef, List, cursor codec, shared collection params, strict booleans) and spec gates G2–G17 + F1 on `app.V1Spec()`, 2.5k lines incl. tests | grok-4.6-xhigh | 2372 s, 310 calls, 947k in + 14.8M cache read, 131k out | Part A worked as specified, including "absent sexual is null, not safe" and the `pickCode` special cases; its seven required mutations were real. Its flat §2 list (27 items) was accurate and flagged about half of what acceptance fixed. But it **ported infra's document post-processing wholesale**: forcing arrays non-nullable and rewriting every error response to Problem. Removing both left the real document byte-identical, so they did nothing except guarantee G9 and G4 could never fail on it. It **deleted the earned registration-order comment** on the unmatched terminator while editing the lines next to it. It added two G8 exceptions after the book said to stop and report (`params` was fixable: renamed to `param_names`). Both AST-scan positive controls ran on snippets, and the omitempty one used its own copy of the rule; the directory walker swallowed errors, so a scan of nothing would have passed. Also found in acceptance: `*struct` and `*DateTime` fields typed non-null while the code sends null (a huma behaviour; infra's `Banner *Image` has the same bug); variant tokens paired a `_mini` URL with the original's dimensions; G8 ignored `$ref` targets; G17 rejected `PUT /x/{id}/like`; F1 skipped parameters; `-1e308` as a G14 bound; a malformed cursor answered `INVALID_PARAMETER`; production `seal.go` imported the gates package. First round: 28 of 47 mutants died, 12 survived, 7 did not compile. After acceptance's tests every non-equivalent mutant dies (one equivalent: a redundant guard, deleted). DB suite green |
| 2026-09-18 | `w0a-5-topics`: the first domain endpoint, `GET /api/v1/topics` (keyset repository, `TopicSummary`, 18 sort tokens), migration 096, DB contract tests with JSON Schema conformance, an empty-database bootstrap script and the CI `db` job; 1.5k lines incl. tests | grok-4.6-xhigh | 2541 s, 449 calls, 20k in + 2.2M cache read, 8.5k out | Everything in scope landed as adjudicated, nothing outside the writable paths changed, and the tests went through `setupRoutes` as the book required. **Its conformance helper caught a W0a-4 bug on the first DB run**: `Image.sexual` was typed `[string, null]` with an enum that had no `null`. huma renders nullable enums that way and its own validator skips null, so the gates had passed it. The bootstrap script and the CI job worked first time on an empty database, and the script never printed the DSN. Its flat §2 list was accurate: a user client blind to HTTP status, `miniapp` swallowing its query error, a discarded `Count`. **Missed: the document declared a 422 the endpoint can never return.** The book said "exactly what G4 derives" and it never compared; the root cause was W0a-3's tier helpers putting codes in `op.Errors`, which makes huma add 422 or `default`. Also missed: the 503 dropped its cause, and a handler's 500 was logged nowhere (W0a-3 again). No test pinned a field's value, so emptying `sections` or `mini_apps` or forcing `has_best_answer` false survived. Nothing covered a missing author, an unparseable cover token or a best answer, and schema descriptions named internal columns and status numbers. Two faults were the book's: it said to use `userclient.Placeholder`, which put a hard-coded Chinese name on the wire (now `name: null`, gate F8), and it said to answer a bad status with 500 rather than constrain it (096 now adds a CHECK). It ran the preflight twice, which `check.sh` misread as an unproven fence (fixed). A live traversal of all 18 sorts on dev data then turned up an empty page with a cursor, because one window held only banned authors' topics; a third fault of the book's, which had allowed short pages. The server now reads up to five windows to fill a page. Final: 48/48 mutants killed; DB suite green on a freshly bootstrapped database |
| 2026-09-18 | `w0b-1-client`: the web's typed v1 client (openapi-fetch factory shared by Nuxt and Nitro, `settle` into a problem shape, `useApi`), the zh-CN problem catalogue whose texts the book gave verbatim, gates F2 / F4 / F5 / F6 with positive controls through the same functions, type tests, and one Go pass dropping `additionalProperties: true`; 1.4k lines incl. tests | grok-4.6-xhigh | 1481 s, 211 calls, 220k in + 10.9M cache read, 76k out | The cleanest run so far. Everything in scope landed and stayed inside the writable paths. When F6 hit a line the book had not foreseen (the OG card service's `${ogBaseUrl}/v1/og/`), it stopped and reported instead of suppressing, exactly as told; its report was accurate. Its tests assert exact strings and values, as the book demanded, and its six mutations were real. Acceptance found: `useApi` called `useApiClient()` inside the `useAsyncData` handler, which works only because Nuxt 4.4 calls the handler synchronously on first run (the book said to read the context synchronously); catalogue lookups used `in`, which walks the prototype; it added a map-value recursion to `walkSchema` that no test could distinguish (deleted); no test for a problem-shaped body under plain `application/json`, or for the caller's signal surviving the timeout wrapper (both added). 17 of 18 orchestrator mutants died; the survivor was the dead recursion. DB suite green |
| 2026-09-18 | `w0b-2-topic-page`: `/topic` onto `GET /api/v1/topics` (a generic `useCursorList` with back/forward snapshot restore, a history-pop tracker, a load-more button, sort tokens typed from the generated operation), the sitemap's topic source onto a v1 cursor walk, removal of the legacy web types; 1.2k lines incl. tests | grok-4.6-xhigh | 1076 s, 242 calls, 461k in + 10.5M cache read, 96k out | Everything in scope landed inside the writable paths, with value assertions throughout (query values, exact texts, id order). When the sandbox's stale `.nuxt` hid the new auto-imports and `nuxt prepare` was not allowed, it imported explicitly and said so rather than working around the list. Its report correctly flagged that Nuxt writes a restored snapshot into the client payload. Acceptance found: `getCachedData` ignored the cause, so a `refresh()` after a back navigation served the snapshot instead of refetching (fixed to `cause === 'initial'`, test added); the sitemap walk had no timeout; the sitemap fixture set `created_at` equal to `bumped_at`, so a `lastmod` mutant survived. A browser pass on dev data then confirmed SSR, no hydration refetch, click-only loading, exact scroll restore with no request on back, a fresh first page on push, sort and NSFW keys, the failed-first-page state, and 2988 sitemap topic URLs. 21 of 21 mutants killed after the fixture fix |
| 2026-09-18 | `w1-content-go`: Markdown → content-node converter (`content.Converter`, `markdown.ParseStored`), 19 mapping rules with exact-JSON cases, schema validation of every output, batch de-duplication, fuzz target, differential test over the 18327-body production corpus given by path; 2.8k lines incl. tests | grok-4.6-xhigh | 2476 s, 325 calls, 598k in + 18.3M cache read, 129k out | Every rule landed as written, inside the writable paths, and its six mutations were real. It flagged the book's own mistake: `atom.Lookup`, which the book prescribed, also knows attribute names, so a literal `<src>` vanished. Acceptance found what its differential test could not: the book told it to compare text with **all whitespace stripped**, so a soft break after an inline container that joined `*word*` and `next` into `wordnext` passed 18327 real bodies. A word-boundary comparison and exact cases now pin it. No test covered `javascript:`, `data:` or hostless `http:` links: 3 of 15 orchestrator mutants survived until cases were added. Conversion is twice as fast as the legacy renderer. DB suite green |
| 2026-09-18 | `w1-content-web`: `<ContentDocument>` rendering the node tree with `h()` in one functional component, legacy-identical markup for the delegated handlers and prose CSS, KunUI's exported spoiler / lightbox / blur-up composables, URL and id guards, unknown-node fallback; 1.1k lines incl. tests | grok-4.6-xhigh | 1045 s, 156 calls, 412k in + 5.7M cache read, 68k out | Clean: exact SSR strings throughout, its four mutations real, an honest flat list of 18 observations (two were real: a spread task item's checkbox sat outside its paragraph, and an invalid-TeX test asserted only "does not throw"). Acceptance also kept empty paragraphs in tight lists, clamped heading depth, and added the missing cases; 10/10 mutants killed. An end-to-end check fed the Go converter's output for all 18327 bodies through it: 0 throws, 0 text differences |
| 2026-09-19 | `w2-census`: read-only census of the legacy topic detail and reply read paths (every field's source and meaning, visibility, side effects, pagination and floors, consumers, tests), report only | grok-4.6-xhigh | 595 s, 189 calls, 208k in + 4.5M cache read, 40k out | Precise and complete, with `file:line` for every claim and a positive control for the consumer search. Its findings set the W2 design: floors unlocked and non-unique, `locate` counting hidden rows, a hidden best answer still embedded, page-1 specials breaking the last-page heuristic, views counted on every GET including SSR and share cards. Its data questions ran unchanged against production |
| 2026-09-19 | `w2-go`: the four W2 operations (topic detail, reply list with `from_floor` anchor, single reply, view beacon), one shared read decision for legacy and v1, migration 097, DB contract tests it could not run; 2.6k lines incl. tests | grok-4.6-xhigh | 2466 s, 443 calls | Everything in scope, contract untouched, honest that no DB test had run. On the first real run 7 of 10 DB tests passed; the 3 failures were its own test bugs: viewer flags decoded into untagged Go fields (`HasLiked` never matches `has_liked`), and the scope cases used the topic's author, who may always read — so the positive "role granted" case proved nothing. The implementation read all five keyset windows whenever more rows existed and loaded comments and reactions for 150 replies to show 30, which no test could see; acceptance rewrote it to stop at the window that fills the page and added a test that counts windows. It also swallowed the daily view-bucket error and put a test hook on the production `App`. 11 of 13 DB mutants killed; one equivalent, one exposed dead code |
| 2026-09-19 | `w2-web`: the topic page onto v1 (detail, cursor replies with backward loading, featured replies above the list, TOC and SEO from the node tree, Nitro OG card, view beacon, legacy writes landing in v1 state); 2.3k lines incl. tests | grok-4.6-xhigh | 2106 s, 351 calls | Complete and careful: exact-value tests throughout, its five required mutations real, and a flat list of 23 observations that named every place it kept local state instead of writing through. Acceptance fixed a non-404 failure shown as "not found", new replies appended at the bottom of a newest-first list, a failed quote preview cached for the tab's life, server code importing `app/`, and two clients taken outside setup. A real-browser pass on the dev stack with a temporary session then confirmed every read path, view counting (+0 for API reads and SSR, +1 per open), and like / reply / comment / delete / edit landing in v1 state |
| 2026-09-19 | `w3-census`: read-only census of the legacy topic and reply write paths (request fields and every rejection, side effects in order, floors, idempotency, edit-context reads, consumers, tests), report only | grok-4.6-xhigh | 690 s, 178 calls | Complete, with `file:line` throughout and a positive control. It found the design-setting facts: the floor race and floor reuse after deleting the top reply, `:tid` ignored by every reply write, moemoepoint grants fired inside the transaction with nonce keys, update as full replacement clearing `is_nsfw` and covers, the delete penalty counting an abandoned like table, and the undocumented mention cap. Its data questions ran unchanged against production |
| 2026-09-19 | `w4-census`: read-only census of the legacy interaction paths, each toggle's on and off branch separately (counters, points, notifications, concurrency, visibility, consumers, tests), report only | grok-4.6-xhigh | 790 s, 203 calls | Precise and flat. It traced the counter race to `ON CONFLICT DO NOTHING` followed by an unconditional increment, the account purge recounting from abandoned tables, the best-answer switch that never takes back the old +7, pinning a reply of another topic, and interaction histories readable for hidden topics. Its SQL confirmed repeat upvotes are real use (25 pairs), which kept upvotes repeatable in v1 |
| 2026-09-19 | `w3-go`: the seven W3 topic and reply write operations, migration 100, DB contract tests it could not run | grok-4.6-xhigh | cut off by a session interruption | Delivered code that **did not compile** (a Fiber v2 `Test` call, a missing import) and with **no reply-write tests at all**; the report and the sandbox proof went with `/tmp`, so acceptance was file-by-file reading plus writing the missing suite. Acceptance found a self-comparing assertion that checked nothing, a fixture whose seed topics tripped the 24h post limit, payloads missing the required `is_nsfw`, a message assertion that always read 0, and three product bugs: an author-only `users`-scope topic could never be PATCHed, section order was never stored though the API documents it, and deleting the top reply handed its floor to the next one. 10/10 mutants killed — the "awards before commit" one only after retargeting the injected fault past the award |
| 2026-09-19 | `w4-go`: the fourteen W4 interaction operations, `viewer.can_*` into the read faces, migration 099, purge recount fix | grok-4.6-xhigh | cut off by a session interruption | Same shape: did not compile (missing gorm import, Fiber v2 `Test` call), and only the reaction family had tests. The implementation itself matched the adjudications once it built. Acceptance also fixed a shared `jsonschema.Compiler` that the interaction tests called from goroutines, crashing the run with a concurrent map write |
| 2026-09-22 | `w4-tests`: the missing W4 DB tests — favorites, upvotes, the three history lists, best answer, pin, migration 099 | grok-4.6-xhigh | 980 s, 167 calls, 372k in + 4.9M cache read, 65k out | Ran to completion and green on lint and the non-DB suite. Its §2 list was accurate and it **named the bug that mattered** — `topic_favorite.updated` / `topic_upvote.updated` are `NOT NULL` with no default while the raw inserts omit them — but having no database it could not confirm it, so it pinned the contract's 200 instead; the first real run was 6 red. It also wrote its own `;`-splitting SQL statement splitter, which counted 11 statements in a 9-statement file; acceptance replaced it with the single `database/sql` Exec that `cmd/migrate` actually performs. Its list fixture gave every row a distinct second, so the keyset's `id` tie-breaker went untested until acceptance added three rows sharing an instant across a page boundary. 15/15 mutants killed after that |
| 2026-09-22 | `w34-web`: the topic page's writes and interactions onto v1 (create/edit/hide, replies, reactions, favorite, upvote, best answer, pin, the two source reads, cursor history lists, an idempotency-key composable), 46 files | grok-4.6-xhigh | 2211 s, 549 calls, 117k in + 7.0M cache read, 40k out | The cleanest cursor run of the wave: lint, typecheck and 377 vitest tests green, its five self-chosen mutations real, and a flat §2 list of 16 observations that was accurate. F4 baseline 324 → 308 with nothing added. It stopped and reported when the sandbox refused to delete two now-unused type files, as the book asks. Acceptance found that its new "toast every field error" loop used `fieldMessage`, which renders only the localized reason — a 422 arrived as two unattributed toasts — and rewrote it to reuse `KUN_FIELD_LABELS`. Of the orchestrator's 15 mutations, the 4 survivors were **all the same hole**: the unset half of K16. Nothing covered unfavorite, unpin or clearing the best answer, so turning all three `DELETE`s into `PUT`s stayed green. After `topic/unset.spec.ts` and `reportProblem.spec.ts`: 15/15. A browser pass on the dev stack then confirmed favorite set/unset, the upvote (201, trimmed note, −10 cached), reaction set/unset, the reaction history cursor list, author-only capabilities and the rewrite source read |
| 2026-09-26 | `fa-review`: read-only review of the following-activities backend (three `/me/following-activities*` operations, a followee cache, migration 197), report only | grok-4.6-xhigh | 1091 s, 109 calls, 196k in + 3.3M cache read, 58k out | Worth the dispatch. It confirmed the site stream's SQL and cursor unchanged line by line, and found what the orchestrator's own 12 mutations had not: `GALGAME_CREATION` feed rows carry `user_id` 0 (a trigger from migration 070), so a followed creator's works never matched; a red dot that could never clear when the newest matching row is one the list drops (banned author); the no-mark window off by a second; a cache generation map that grew without bound and whose `clear` could undo a `Forget`. Its "properties no test pins" list was accurate and became the next task's test list. Two of its points were judged not defects (row creation matching login's `Ensure`, `WithoutCancel`) |
| 2026-09-26 | `fa-fixes`: the seven adjudicated fixes from `fa-review` (a `UNION ALL` for creator-matched work rows, optional `seen_at`, inclusive count bound, explicit empty actor list, cache epoch, truncation warning, doc fixes) plus 9 tests | grok-4.6-xhigh | 1096 s, 181 calls, 437k in + 7.7M cache read, 67k out | Clean: inside the writable paths, the site stream's SQL string kept byte-identical as the book required, `-race` green, its three unit-test mutations real. It flagged honestly that its window test's one-hour margin could not catch the one-second bug it fixed. The five DB tests it wrote but could not run all passed first time on a real database, and the orchestrator's 7 mutations all died — one only after being rewritten, since the first version killed by a build error (an unused import), which proves nothing. Acceptance replaced hand-rolled make/append copies with `slices.Concat` and pointed the site stream at the shared SELECT constant |
| 2026-09-26 | `mp-links`: `MoemoepointEntry` gains a server-resolved in-site link (renumber-aware work ids, upvote → topic, reply → topic + floor; batched lookups in a new user repository) and a public `getAccountPrices` read from the account center's public settings, 1.1k lines incl. tests | grok-4.6-xhigh | 1110 s, 199 calls, 314k in + 6.2M cache read, 70k out | Everything in the resolution table landed as adjudicated and inside the writable paths. It stopped at G8 as the book said instead of renaming or allow-listing: the nullable `path` clashed with the non-null `path` on `Activity` / `Notification`, a clash the orchestrator's own design had missed; acceptance renamed it `ref_path`. **Its batching test counted nothing**: it hooked GORM's Query callback, but `Table(...).Scan` runs the Row callback, so the count was 0 on a real database. Its report still listed the mutation that test "would kill". Acceptance hooked both. It also wrote a redundant leading-zero guard (an equivalent mutant) and a five-type number switch where the decoded settings only ever hold `float64`; both were simplified. The orchestrator's first mutation batch had its own bug: `grep -F` checked the multi-line patterns line by line, so five mutants never applied and "survived". Re-run with an exact-count replacer: every non-equivalent mutant died |

The first two runs hit the same environment wall: the sandbox cannot write `~/go`, so the first
`GOTOOLCHAIN=go1.26.1` toolchain or sumdb fetch fails with `permission denied`. Both worked
around it by pointing `GOPATH` into the run's own cache, which `dispatch.sh` deletes. The
template now says so.
