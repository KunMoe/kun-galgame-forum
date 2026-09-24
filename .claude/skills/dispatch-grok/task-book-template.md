# Task: <one line, imperative, what gets built>

You are the **executor**. The orchestrator wrote this task book; it is self-contained.
You cannot see the orchestrator's conversation. Everything you need is below.

## Context

Repository: `<absolute path to this checkout, filled in at dispatch time>` (your working
directory). Branch: `master` at `<short sha>`. Monorepo: `apps/api` is Go 1.26 / Fiber v3 /
GORM / Postgres (module `kun-galgame-api`); `apps/web` is Nuxt 4 with the KunUI component
library (`@kungal/ui-*`).

<Where the relevant code lives — exact paths. What it does today. Why it is changing.
Every prior adjudication this task depends on, stated inline. If the reader would have to ask
"why this way and not the obvious way", answer it here.>

## Your environment (read this, it is not the usual one)

- **You have no shell.** The `run_terminal_command` tool is denied by policy. You cannot
  build, test, format, run `gofmt`, or run anything. **That is expected.** The orchestrator
  runs every gate after you finish. Write code that compiles; do not try to prove that it
  does.
- **You have no MCP servers and no database.**
- **You cannot read any `.env` / `*.env` file** — they are denied, and they hold live
  credentials. Never try.
- Reads inside the repository are free, and so are reads outside it, but this task needs
  nothing outside it.
- The repository `CLAUDE.md` is already in your context. Its iron rules bind you.
- <Delete if not applicable:> Do not use web search.

## Binding constraints for this task

<Name the specific rules, with the file and the quoted line. Not "follow the docs". A task
touching apps/web almost always needs iron rules 1 (no background gradients), 2 (KunUI
first, never modify KunUI) and 11 (single real root element per page) restated here.>

- `CLAUDE.md` iron rule <n>: "<quoted line>"

## Scope

1. <numbered, concrete, each independently checkable>
2. …

## Out of scope

- <what a helpful executor would otherwise wander into>
- Renaming, reformatting, or refactoring anything not named in Scope.

## Precedent to follow

<Point at existing code that already does this correctly: file:line. Say what to copy — the
shape, the error handling, the naming, the comment discipline — and what not to.>

## Acceptance criteria

The orchestrator will run these. You cannot. They are listed so you know what your code
must satisfy:

- `go build ./...` and `make lint` (vet + errcheck) in `apps/api` must be green.
  <Delete if the task has no Go side.>
- `pnpm -F web lint`, `pnpm -F web typecheck` and `pnpm -F web test` must be green.
  <Delete if the task has no web side.>
- Test `<TestName>` in `<file>` must pass.
- `git status --porcelain` must show **only** the writable paths below.

## Report

Write your report to this exact absolute path:

    <GROK_OUT_ROOT>/<slug>/report.md

Structure:

```
# Report: <task>

## 1. What I changed
(file:line per change, one line each, what and why)

## 2. Anything that looks wrong — in scope or not
(report every one, at the same weight, with file:line and the quoted line. Something outside
 this task's scope belongs here, not in section 5, and not with a note that it was out of scope.
 Report it; do not fix it.)

## 3. Mechanics I chose
(any decision the task book left to the code — what you picked and the precedent you followed)

## 4. Deviations from the task book
(if none, write "None.")

## 5. What I could not verify
(everything requiring a command belongs here — be specific about what the orchestrator should
 check, not just "run the tests")
```

Your final stdout message: one short paragraph, the report path plus a one-line status.
Do not paste the report into stdout.

## Discipline

- Writable paths — **exactly** these, nothing else anywhere:
  - `<glob 1>`
  - the report path above
- Forbidden: any shell command; any git operation; any database access; editing files
  outside the writable paths; writing a migration under `apps/api/migrations/**`; touching
  `docs/oauth/oauth-integration-guide.md` (read-only infra mirror);
  hand-editing `apps/api/internal/app/testdata/routes.golden` (it is derived by a command
  you cannot run — if your change alters routes, say so in section 5 and the orchestrator
  regenerates it); touching any `.env.example`.
- <Delete unless the task adds, removes or renames a permission key:> The permission
  vocabulary lives in three synchronized places — `apps/api/pkg/perm/perm.go`,
  `apps/web/app/composables/useCan.ts`, `apps/web/app/constants/permission.ts` — and
  `apps/api/pkg/perm/frontend_mirror_test.go` fails unless all three move together.
  Change all three or none.
- **Comments are earned by a mistake that already happened, not one you predict**
  (`CLAUDE.md`, "Comments"). Default to none. Do not add section banners, restatements of
  the code, or doc comments that echo the identifier. Where this task book tells you *why*
  something is ordered or excluded, that reason is worth a short comment; the mechanism is
  not.
- **Report, don't work around.** If something is missing, contradictory, or blocked, stop
  and write it in section 4 or 5. A blocked task reported accurately is a success; a task
  completed by inventing around the block is not.
- **Do not rank, score, or filter your findings.** Report every one flat, at equal weight.
  Never demote something to "minor", "cosmetic", or "out of scope" — that judgement is the
  orchestrator's and yours will be wrong.
- <Delete unless the task is a search, audit, or census:> **Include a positive control.**
  List what you checked that came back clean, with counts, so a zero can be believed.
