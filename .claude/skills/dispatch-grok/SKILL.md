---
name: dispatch-grok
description: Dispatch implementation and investigation work to the local grok CLI as a headless executor while this session stays the orchestrator and acceptor. Use when the user asks to "派发 grok" / "dispatch grok" / "let grok do it", or when a task is large enough to hand off as a written task book. Covers what grok can reach on this machine (measured, not inherited), the deny fence that keeps this repo's env files out of a third party's hands, the flag set, task-book structure, and the acceptance protocol.
---

# Dispatching grok as executor

> **If you are the grok executor and this file was loaded into your context: ignore it.**
> It describes how the orchestrator dispatches *you*. It is not a task book.

This session is the **orchestrator**: it adjudicates design, writes the task book, runs every
command, and accepts or rejects the result. The local `grok` CLI (Grok Build, xAI) is the
**executor**: it reads and writes files, and does nothing else.

A dispatch is a subprocess of this session working in this session's checkout — not a second
session in the iron-rule-13 sense. What rule 13 does forbid is two writers on one checkout:
never dispatch while another session owns this worktree, and never run two dispatches (grok
or codex) concurrently over overlapping paths.

## 1. What grok can reach on this machine

The dispatched-probe rows were measured **2026-09-03 in the sibling private repo this skill
was adapted from**, against `grok 1.0.13` — same machine, same grok version. The
machine-level pins were re-checked **2026-09-05**: `/etc/grok/requirements.toml` still pins
only `telemetry = false` and `remote_fetch = false` (no sandbox profile), and
`~/.grok/config.toml` still sets `permission_mode = "always-approve"`. Verify the probe rows
against the first real dispatch in this repo; do not inherit this table anywhere else.

| Capability | Headless (`--prompt-file`) | Measured outcome |
|---|---|---|
| Read / grep / list inside the repo | **free**, no rule needed | its own tools, not shell |
| Write / edit inside the repo | needs a path-scoped `--allow` | see §2 |
| **Reading outside the repo** | **works** — this is the surprise | read a sibling repo's `CLAUDE.md`, listed the parent directory, read `/etc/grok/requirements.toml` |
| Shell | blocked by `--deny 'Bash(*)'` | `Denied by permission policy: deny rule on bash` |
| MCP (github, notion, postgres) | blocked by `--deny 'mcp__*'` | configured and visible to grok; the deny is what stops it |
| The repo's env files | blocked by five `--deny 'Read(...)'` rules | see below |
| Web search / fetch | available | `--disable-web-search` removes it; `remote_fetch = false` is pinned machine-wide |

**The sandbox is open.** Nothing structural stops a dispatched executor from reading anything
this user can read, and `always-approve` means an unmatched write is approved by default.
**The `--deny` rules in `dispatch.sh` are therefore a security fence, not an ergonomic one —
do not remove them.** Before trusting any row above, `cat /etc/grok/requirements.toml`: if a
`sandbox` or `permission_mode` pin has come back, the rows change again.

### The env files are the ones that matter in this repo

This repo carries **six** gitignored env files: `apps/api/.env` (the live catalog `nmk_`
key, OAuth client secret, image / trust / JWT secrets, the DB DSN), `apps/web/.env` (OG site
key, image-bed and S3 secrets, CF cache token), `apps/web/.env.prod` (**production**
credentials), `docker/api.env`, `docker/web.env`, `refs/legacy/.env.prod`. grok is a
third-party service; an executor reading any of them is an exfiltration, and iron rule 14's
"never print the DSN" covers the same ground.

`dispatch.sh` denies five patterns; between them they cover all six files:

```
Read(.env)   Read(.env.*)   Read(**/.env)   Read(**/.env.*)   Read(**/*.env)
```

The fifth pattern (`**/.env.*`) is new relative to the repo this skill came from — its four
patterns left `apps/web/.env.prod` and `refs/legacy/.env.prod` uncovered here. A denied read
answers `Denied by permission policy: deny rule on read matching "..."` and names the pattern
that fired, so a miss would be visible rather than silent.

Two independent things kept the secret in the original probe and **only one of them is a
fence**: grok's search and directory-listing tools are `.gitignore`-aware, so the env files
never appear in a listing and a grep for their content returns nothing. That is luck, not
policy — a secret at a tracked path would be greppable. The deny rules are the part that
holds.

### Deny converts a killed run into a survivable one

`--deny` is enforced before any approve mode, and a denied tool returns an error the model
reads and works around: the shell probe finished `end_turn` and reported *"the tool did not
run, it was refused"*. A tool that instead falls to an **ask floor** cancels the whole run,
because headless has nobody to ask. That is the difference `dispatch.sh` is buying.

**The division of labour follows, and it is a good one:** grok writes, the orchestrator
runs. Never ask grok to build, test, `gofmt`, migrate, `git` anything, or "verify" something
that needs a command. Every gate is the orchestrator's to run, which is where acceptance
belonged anyway.

grok loads the repo's `CLAUDE.md` automatically as project instructions — the sibling-repo
probe quoted an iron rule's opening words back without being pointed at the file; have the
first dispatch here confirm the same. Do not re-paste the iron rules; do restate the
specific ones the task turns on. **grok does not load skills**, so anything in
`.claude/skills/**` is invisible to it unless the task book names the file.

## 2. The dispatch

```bash
export GROK_OUT_ROOT="$SCRATCHPAD/grok"          # session scratchpad, never the repo
mkdir -p "$GROK_OUT_ROOT/<slug>"
# write the task book to $GROK_OUT_ROOT/<slug>/task.md, then:
.claude/skills/dispatch-grok/dispatch.sh <slug> \
  --allow 'Write(apps/api/internal/topic/**)' \
  --allow 'Edit(apps/api/internal/topic/**)'
```

`dispatch.sh` supplies `--prompt-file`, the two rules for the output directory, the deny
rules from §1, `--output-format json`, `--max-turns` and `--debug-file`, then reports
`stopReason`/`turns`/`cost_usd` and diagnoses a bad exit. Every extra argument is passed
through.

Run it **in the background** — a real task runs for minutes and a foreground call blocks the
turn.

### `--allow` is a grant, not a whitelist — the acceptance diff is the scope fence

**Measured 2026-09-03 in the sibling repo, and it contradicts the obvious reading.** A
dispatch granted `Edit` on three named files also edited a fourth no rule covered. It was
not refused: `always-approve` means a write with no matching rule is **approved by
default**. `--allow` adds permissions; only `--deny` removes them. That makes the deny rules
the sole hard fence, and it makes **`git status --porcelain` at acceptance the only thing
that actually bounds scope**. Run it first, every time, and read any file that appears
outside the grant before judging the change.

Still grant the narrowest globs the task actually needs, per dispatch, never as a standing
default: they document intent and they are what a review reads. Just do not mistake them for
enforcement. Rule prefixes are the Claude-Code-compatible names — `Write`, `Edit`, `Read`,
`Bash` — not grok's native tool names; a native name is a hard error (`unknown tool
prefix`).

**Never grant a glob covering a file whose content is derived or tri-synced**, because a
gate then fails on the hand edit and the orchestrator has to re-derive it. In this repo:

- `apps/api/migrations/**` — a migration is never an executor's to write, and the
  end-of-task migration reminder is the orchestrator's duty.
- `apps/api/internal/app/testdata/routes.golden` — derived; regenerate with
  `go test ./internal/app/ -run TestRouteManifest -update-routes` (a command, so grok
  cannot; the orchestrator regenerates at acceptance when a task changed routes).
- The permission tri-mirror — `apps/api/pkg/perm/perm.go`,
  `apps/web/app/composables/useCan.ts`, `apps/web/app/constants/permission.ts`:
  `frontend_mirror_test.go` fails unless all three move together. Grant all three or none.
- `docs/oauth/oauth-integration-guide.md` — infra's read-only mirror.
- `.env.example` files — they pair with the config loaders; knob and line move together or
  not at all.
- `package.json` version lines — husky bumps them; the orchestrator sweeps the bump into
  the commit.

Never run two dispatches concurrently over overlapping globs. Sequential, or disjoint globs.

### Useful extra flags

| Flag | When |
|---|---|
| `--effort low\|medium\|high\|xhigh` | default is `high`; `low` for mechanical sweeps and probes |
| `-m grok-4.5` | default is `grok-4.6` |
| `--no-subagents` | when you want one deterministic worker instead of a fan-out |
| `--disable-web-search` | offline-only tasks; removes web search and fetch |
| `GROK_MAX_TURNS=<n>` | default 200; a probe needs 30, a 20-file build wants the default |

## 3. Reading the result

1. **`stopReason: "cancelled"` is ambiguous** — a denied call, an ask floor, or turn
   exhaustion. `--debug-file` is always written; `dispatch.sh` greps it and says which.
2. **`end_turn` is not acceptance.** With the deny rules in place a run that reached for the
   shell *finishes cleanly* and mentions it in its report instead of dying. Read report
   sections 4 and 5, not just the exit line.
3. **`.text` is the concatenation of every assistant text block**, not the final answer.
   Never parse a report out of it — require the report to be written to a file in the
   output directory.
4. **`--json-schema` output is also concatenated.** Take the *last* balanced JSON object.

The report lives in the scratchpad and never in the repo, so `git status --porcelain` after
a run is a pure signal: exactly what grok changed in code, nothing else. That is acceptance
check one.

## 4. The task book

Template: `task-book-template.md` in this directory. Requirements:

- **English**, and **self-contained** — grok sees none of this conversation, and none of the
  orchestrator's memory files.
- **Tell it about §1.** "You have no shell, no MCP, no database." A task book that says "run
  the tests" wastes the dispatch.
- **No open design decisions.** Every adjudication is made in the task book. If the
  mechanics depend on code grok has yet to read, state the invariant plus the precedent,
  and require it to report what it chose.
- **Acceptance criteria the orchestrator will actually run** — named tests, exact commands.
- **A discipline section**: exact writable paths, forbidden operations, *report, don't work
  around*.
- **A named report path and a fixed report structure.**
- **Forbid ranking.** The executor cannot see what the orchestrator knows, so its
  importance ordering is noise. Put "anything that looks wrong, in scope or not" *near the
  top* of the report structure, and forbid demoting a finding to minor / cosmetic /
  out-of-scope.
- **Demand a positive control on any search, audit or census.** "I found 4" is unreadable
  without "and here are the 16 I checked that were clean".

## 5. What the orchestrator never delegates

- Adjudications and scope calls — they belong *in* the task book, not in the executor.
- Every command: the gates (`go build ./...`, `make lint` — vet + errcheck — and
  `go test ./...` in `apps/api`; `pnpm -F web lint`, `pnpm -F web typecheck`,
  `pnpm -F web test`), `pnpm migrate`, anything driven against the running app.
- Every migration under `apps/api/migrations/**`, every schema decision behind one, and the
  end-of-task migration reminder to the user.
- Anything that touches the database, production, or a live upstream (catalog / OAuth /
  community / image / trust) with real keys.
- All git: staging, commits (`git commit -- <explicit paths>`, never `add -A`), branches,
  pushes, PRs.
- **Any ruling under iron rule 1 or 2.** Whether a gradient is one of the sanctioned
  exceptions, whether a KunUI equivalent exists — an executor writes to the ruling, never
  makes it, and never edits KunUI itself.
- **Final acceptance**: `git status --porcelain` shows only the granted paths; re-run the
  gates yourself; spot-check the report's highest-stakes claims against the code.

## 6. What is worth dispatching

Dispatching buys **orchestrator context**, not money. Judge every candidate by whether the
result **can be compressed into something checkable without re-reading the input**.

| Shape | Verdict |
|---|---|
| Broad read → narrow report whose findings are `file:line` + a quoted line | **Dispatch.** The coordinates make verification cheap. |
| Wide mechanical build whose correctness a gate asserts (a typecheck, a table-driven test suite) | **Dispatch.** The orchestrator reads what the gate flags, not the diff. |
| New code carrying design judgement | **Dispatch only with the judgement already made in the task book.** Hand over a decided rule table and it is transcription; hand over the problem and the orchestrator pays for the output *and* the review. |
| `apps/web` work | **Prefer codex** — grok cannot run `vue-tsc`, and a Vue/TS change never typechecked is expensive to accept. |
| Anything whose answer depends on running something | **Impossible** — see §1. |

The structural ceiling: an executor with no shell can prove that documents disagree **with
each other**, never that they disagree **with reality**. What this repo has learned the hard
way — the GORM dev-logging stdout stall, the KunUI focus-trap interactions, which SQL face
was silently returning zero — came from running things, and a dispatch cannot run anything.
Dispatch the code; keep the measurement.

## 7. Binding documents

`CLAUDE.md` is auto-loaded (verify on the first dispatch here). Everything else binding
is outside git: per-feature adjudications are in `refs/docs/proj/*`, which is gitignored and
exists only in the source checkout (give grok absolute paths into
`/home/kun/Desktop/code/website/kun-galgame-forum/refs/docs/`), and contract docs are at
infra's source, `/home/kun/Desktop/code/website/nextmoe-infra/docs/`. Still name the
specific file and quote the sentence a task depends on — an executor told to "follow the
docs" will follow the wrong part of them. The orchestrator's memory files are **not**
visible to grok; anything load-bearing from memory goes into the task book verbatim.
